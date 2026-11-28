// Command soak plays the real client (internal/client.Game) in a window
// against a real in-process server (dev mode, a LocalConn, as cmd/game
// does), along a script, and measures it: frame times per scene (the median,
// the 99th percentile, the first frames after arriving, where art is drawn
// the first time it is shown), and the memory (Go heap, the process's working
// set and private bytes, the client's caches; see client.CacheStats) over a
// long tour of the world.
//
//	go build -o soak.exe ./cmd/soak
//	soak.exe -mode scenes -size 1024 -quality 100 -zoom 0     # frame times of heavy scenes
//	soak.exe -mode soak -size 2048 -minutes 10                # memory over a long tour
//
// It needs a display (a window of -w by -h is opened; the frame is drawn at
// exactly that size, as cmd/shot -exact does) and prints markdown tables.
// Vsync is off, and every frame is read back from the graphics card, so the
// times include the GPU's work.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

type step struct {
	at   float64 // seconds into the stage
	line string  // a dev console line
}

type stage struct {
	name  string
	secs  float64
	steps []step
	move  func(t float64) (int8, int8) // nil: stand still
}

type stageStats struct {
	name  string
	times []time.Duration // every frame of the stage, in order
	first time.Duration   // the longest of the first 20 frames
	hitch []string        // frames of 50 ms or more, with their index
	mem   memRow
}

type memRow struct {
	t                  time.Duration
	heapInuse, heapSys uint64
	live               uint64 // after a collection (with -gc), else 0
	ws, priv           uint64
	artBytes           int64
	chunks             int
	rig, critter       int
}

type runner struct {
	inner  *client.Game
	conn   *server.LocalConn
	next   func(i int) (stage, bool)
	gc     bool
	maxRun time.Duration

	begin     time.Time
	frames    int
	idx       int
	cur       stage
	curStats  *stageStats
	started   time.Time
	nextStep  int
	lastDraw  time.Time
	sentMove  [2]int8
	done      bool
	stats     []*stageStats
	lastMem   time.Time
	memEvery  time.Duration
	periodic  []memRow
	firstDraw time.Duration
	w, h      int
}

const settleFrames = 90

func (r *runner) Layout(int, int) (int, int) { return r.w, r.h }

func (r *runner) Update() error {
	if r.done || (r.maxRun > 0 && time.Since(r.begin) > r.maxRun) {
		return ebiten.Termination
	}
	r.frames++
	if err := r.inner.Update(); err != nil {
		return err
	}
	if r.frames < settleFrames {
		return nil
	}
	now := time.Now()
	if r.curStats == nil || now.Sub(r.started).Seconds() >= r.cur.secs {
		if r.curStats != nil {
			r.finishStage()
		}
		s, ok := r.next(r.idx)
		if !ok {
			r.done = true
			return nil
		}
		r.idx++
		r.cur, r.started, r.nextStep = s, now, 0
		r.curStats = &stageStats{name: s.name}
		r.stats = append(r.stats, r.curStats)
		log.Printf("stage %d: %s", r.idx, s.name)
	}
	t := now.Sub(r.started).Seconds()
	for r.nextStep < len(r.cur.steps) && r.cur.steps[r.nextStep].at <= t {
		_ = r.conn.Send(protocol.ClientMsg{Kind: protocol.ClientDev, Dev: r.cur.steps[r.nextStep].line})
		r.nextStep++
	}
	var mv [2]int8
	if r.cur.move != nil {
		mx, my := r.cur.move(t)
		mv = [2]int8{mx, my}
	}
	if mv != r.sentMove {
		r.sentMove = mv
		_ = r.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: sim.Input{MoveX: mv[0], MoveY: mv[1]}})
	}
	if now.Sub(r.lastMem) >= r.memEvery {
		r.lastMem = now
		r.periodic = append(r.periodic, r.sample(false))
		r.lastDraw = time.Time{}
	}
	return nil
}

func (r *runner) sample(forceGC bool) memRow {
	var m runtime.MemStats
	row := memRow{t: time.Since(r.begin)}
	if forceGC {
		runtime.GC()
	}
	runtime.ReadMemStats(&m)
	row.heapInuse, row.heapSys = m.HeapInuse, m.HeapSys
	if forceGC {
		row.live = m.HeapAlloc
	}
	row.ws, row.priv = processMemory()
	row.artBytes, _, _ = client.ArtCost()
	row.chunks, _, _ = client.GroundCost()
	for _, c := range client.CacheStats() {
		switch c.Name {
		case "dino rig frames":
			row.rig = c.N
		case "critter frames":
			row.critter = c.N
		}
	}
	return row
}

func (r *runner) finishStage() {
	r.curStats.mem = r.sample(r.gc)
	r.lastDraw = time.Time{} // the sampling is not a frame
	r.curStats = nil
}

func (r *runner) Draw(screen *ebiten.Image) {
	r.inner.Draw(screen)
	px := make([]byte, 4)
	screen.SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image).ReadPixels(px) // waits for the graphics card
	now := time.Now()
	if r.firstDraw == 0 {
		r.firstDraw = now.Sub(r.begin)
	}
	if !r.lastDraw.IsZero() && r.curStats != nil {
		d := now.Sub(r.lastDraw)
		st := r.curStats
		st.times = append(st.times, d)
		if len(st.times) <= 20 && d > st.first {
			st.first = d
		}
		if d >= 50*time.Millisecond && len(st.hitch) < 12 {
			st.hitch = append(st.hitch, fmt.Sprintf("#%d %.0f ms", len(st.times), ms(d)))
		}
	}
	r.lastDraw = now
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func pct(ts []time.Duration, p float64) time.Duration {
	if len(ts) == 0 {
		return 0
	}
	return ts[min(len(ts)-1, int(float64(len(ts))*p))]
}

func main() {
	mode := flag.String("mode", "scenes", "scenes (frame times of heavy scenes) or soak (a long tour, for memory)")
	size := flag.Int("size", 1024, "world size in tiles")
	seed := flag.Uint64("seed", 20240601, "world seed")
	w := flag.Int("w", 1920, "frame width")
	h := flag.Int("h", 1080, "frame height")
	quality := flag.Int("quality", 100, "render quality: 100, 75 or 50")
	zoom := flag.Int("zoom", 0, "view zoom, -3 to 3")
	minutes := flag.Float64("minutes", 5, "with -mode soak: how long the tour lasts")
	sceneSecs := flag.Float64("scenesecs", 12, "with -mode scenes: seconds in each scene")
	only := flag.String("only", "", "with -mode scenes: only the scenes whose name contains this")
	gc := flag.Bool("gc", false, "collect at the end of each stage to see the live heap (a hitch the frame times leave out)")
	every := flag.Duration("sample", 15*time.Second, "how often memory is sampled")
	maxRun := flag.Duration("max", 40*time.Minute, "give up after this long")
	flag.Parse()

	begin := time.Now()
	log.Printf("scanning a %d world for scenes", *size)
	scan, err := sim.Generate(sim.GenConfig{Seed: *seed, Width: *size, Height: *size, Style: sim.StyleLatest})
	if err != nil {
		log.Fatal(err)
	}
	genTime := time.Since(begin)
	sc := findScenes(scan)
	scan = nil
	runtime.GC()

	client.SetZoom(*zoom)
	client.OverrideSettings(func(s *client.Settings) { s.Quality = *quality })
	t0 := time.Now()
	srv, err := server.New(server.Config{Seed: *seed, Width: *size, Height: *size, Style: sim.StyleLatest, Dev: true})
	if err != nil {
		log.Fatal(err)
	}
	serverTime := time.Since(t0)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	conn, err := srv.ConnectLocal()
	if err != nil {
		log.Fatal(err)
	}
	game := client.New(conn, "Dev")
	game.SetDev(true)
	ebiten.SetWindowSize(*w, *h)
	ebiten.SetWindowTitle("soak")
	ebiten.SetVsyncEnabled(false)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	var next func(i int) (stage, bool)
	switch *mode {
	case "scenes":
		list := sceneStages(sc, *sceneSecs, *only)
		next = func(i int) (stage, bool) {
			if i >= len(list) {
				return stage{}, false
			}
			return list[i], true
		}
	case "soak":
		deadline := time.Now().Add(time.Duration(*minutes * float64(time.Minute)))
		next = func(i int) (stage, bool) { return soakStage(sc, i, *size, deadline) }
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
	r := &runner{inner: game, conn: conn, next: next, gc: *gc, maxRun: *maxRun, begin: time.Now(), memEvery: *every, w: *w, h: *h}
	base := r.sample(true)
	if err := ebiten.RunGame(r); err != nil && err != ebiten.Termination {
		log.Print(err)
	}
	report(r, base, *mode, *size, *quality, *zoom, *w, *h, genTime, serverTime)
	cancel()
	<-done
	_ = conn.Close()
	os.Exit(0)
}

func report(r *runner, base memRow, mode string, size, quality, zoom, w, h int, genTime, serverTime time.Duration) {
	mb := func(b uint64) float64 { return float64(b) / (1 << 20) }
	fmt.Printf("\n## soak: %s, world %d, %dx%d, quality %d, zoom %+d, %s/%s, %d CPUs\n\n", mode, size, w, h, quality, zoom, runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	fmt.Printf("generate (scan world) %.1f s; server.New (generate) %.1f s; first frame drawn %.1f s after the client was made\n\n", genTime.Seconds(), serverTime.Seconds(), r.firstDraw.Seconds())
	fmt.Printf("baseline before the first frame: heap in use %.0f MB, heap from OS %.0f MB, live %.0f MB, working set %.0f MB, private %.0f MB\n\n", mb(base.heapInuse), mb(base.heapSys), mb(base.live), mb(base.ws), mb(base.priv))
	fmt.Println("| stage | frames | median ms | p95 ms | p99 ms | max ms | first-20 max ms | >50 ms frames |")
	fmt.Println("|---|---|---|---|---|---|---|---|")
	for _, st := range r.stats {
		ts := slices.Clone(st.times)
		if len(ts) == 0 {
			continue
		}
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		fmt.Printf("| %s | %d | %.1f | %.1f | %.1f | %.1f | %.1f | %s |\n", st.name, len(ts), ms(pct(ts, .5)), ms(pct(ts, .95)), ms(pct(ts, .99)), ms(ts[len(ts)-1]), ms(st.first), strings.Join(st.hitch, ", "))
	}
	fmt.Println("\n| t (s) | stage end | heap in use MB | heap sys MB | live MB | working set MB | private MB | art MB (cumulative) | ground chunks | rig frames | critter frames |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|")
	for _, st := range r.stats {
		m := st.mem
		if m.t == 0 {
			continue
		}
		fmt.Printf("| %.0f | %s | %.0f | %.0f | %.0f | %.0f | %.0f | %.1f | %d | %d | %d |\n", m.t.Seconds(), st.name, mb(m.heapInuse), mb(m.heapSys), mb(m.live), mb(m.ws), mb(m.priv), float64(m.artBytes)/(1<<20), m.chunks, m.rig, m.critter)
	}
	if len(r.periodic) > 0 {
		fmt.Println("\nperiodic samples:\n\n| t (s) | heap in use MB | working set MB | private MB | art MB | chunks | rig | critter |\n|---|---|---|---|---|---|---|---|")
		for _, m := range r.periodic {
			fmt.Printf("| %.0f | %.0f | %.0f | %.0f | %.1f | %d | %d | %d |\n", m.t.Seconds(), mb(m.heapInuse), mb(m.ws), mb(m.priv), float64(m.artBytes)/(1<<20), m.chunks, m.rig, m.critter)
		}
	}
	fmt.Println("\nclient caches at the end:\n\n| cache | entries | MB |\n|---|---|---|")
	for _, c := range client.CacheStats() {
		fmt.Printf("| %s | %d | %.1f |\n", c.Name, c.N, float64(c.Bytes)/(1<<20))
	}
}
