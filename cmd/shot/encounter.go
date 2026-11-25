package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// The -encounter shots: a real game (a dev-mode server and a client, the
// demo's two bosses) taken through meeting a boss, with the dev console's
// commands as the player's legs. Stills are written at the moments that
// matter: coming near (the signs), by the lair (the boss hidden), the
// entrance (the scene of bossintro.go) and the arena shut. With -encgif
// every fourth frame of the entrance is written too, to make a GIF of.
//
//	xvfb-run -a -s '-screen 0 1920x1080x24' go run ./cmd/shot -encounter titan -w 1920 -h 1080 -o /tmp/enc
//	ffmpeg -y -framerate 15 -i /tmp/enc_gif_%03d.png -vf "scale=960:-1:flags=lanczos,split[a][b];[a]palettegen[p];[b][p]paletteuse" enc.gif

// encStep is something the encounter does at a frame.
type encStep struct {
	at   int
	cmds []string // dev console lines
	shot string   // a still to write, named after -o
}

type encounter struct {
	inner    ebiten.Game
	srv      *server.Server
	conn     client.Conn
	steps    []encStep
	out      string
	gif      [2]int // frames between which the GIF's frames are written; none if equal
	n, gifN  int
	end      int
	pending  []string
	gifShoot bool
	err      error
}

func (e *encounter) Update() error {
	if e.n >= e.end || e.err != nil {
		return ebiten.Termination
	}
	e.n++
	if e.n%(60/sim.TicksPerSecond) == 0 {
		e.srv.Tick() // the server keeps the frames' time, however slowly they are drawn
	}
	for _, s := range e.steps {
		if s.at != e.n {
			continue
		}
		for _, c := range s.cmds {
			if err := e.conn.Send(protocol.ClientMsg{Kind: protocol.ClientDev, Dev: c}); err != nil {
				e.err = err
			}
		}
		if s.shot != "" {
			e.pending = append(e.pending, s.shot)
		}
	}
	e.gifShoot = e.n >= e.gif[0] && e.n < e.gif[1] && (e.n-e.gif[0])%4 == 0
	return e.inner.Update()
}

func (e *encounter) Draw(screen *ebiten.Image) {
	e.inner.Draw(screen)
	if len(e.pending) == 0 && !e.gifShoot {
		return
	}
	img := image.NewRGBA(screen.Bounds())
	screen.ReadPixels(img.Pix)
	write := func(name string) {
		f, err := os.Create(name)
		if err == nil {
			err = png.Encode(f, img)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			e.err = err
		}
		log.Printf("wrote %s", name)
	}
	for _, s := range e.pending {
		write(e.out + "_" + s + ".png")
	}
	e.pending = nil
	if e.gifShoot {
		write(fmt.Sprintf("%s_gif_%03d.png", e.out, e.gifN))
		e.gifN++
		e.gifShoot = false
	}
}

func (e *encounter) Layout(w, h int) (int, int) { return e.inner.Layout(w, h) }

// runEncounter plays the meeting with boss which ("titan" or "shark") and
// writes its stills as out_<moment>.png.
func runEncounter(which, out string, seed uint64, size, winW, winH int, dayFrac float64, gif bool) {
	sim.SetDemo(true)
	cfg := server.Config{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest, Dev: true}
	twin, err := sim.Generate(sim.GenConfig{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest})
	if err != nil {
		log.Fatal(err)
	}
	kind := sim.CreatureTitanoboa
	if which == "shark" {
		kind = sim.CreatureMegalodon
	}
	var lx, ly float64
	found := false
	for _, b := range twin.Bosses() {
		if b.Kind == kind {
			lx, ly, found = b.LairX, b.LairY, true
		}
	}
	if !found {
		log.Fatalf("this world has no lair for %s: try another -seed", kind.Name())
	}
	tp := func(dx, dy float64) string { return fmt.Sprintf("tp %.2f %.2f", lx+dx, ly+dy) }
	clock := fmt.Sprintf("time %.2f", dayFrac)
	var steps []encStep
	var gifAt [2]int
	end := 0
	if kind == sim.CreatureTitanoboa {
		land := func(sign float64) (n int) { // how much ground there is to walk in on from that side
			for d := 3.0; d < 22; d++ {
				if twin.TileAt(int(lx+sign*d), int(ly+2)).Walkable() {
					n++
				}
			}
			return n
		}
		if land(1) > land(-1) {
			at := tp
			tp = func(dx, dy float64) string { return at(-dx, dy) }
		}
		steps = []encStep{
			{at: 20, cmds: []string{"god on", clock, "weather clear", tp(-16, 3)}},
			{at: 110, shot: "approach"},
			{at: 120, cmds: []string{tp(-7.7, 2.4)}},
			{at: 230, shot: "lair"},
			{at: 240, cmds: []string{tp(-5.5, 1.5)}},
			{at: 240 + 84, shot: "intro"},
			{at: 470, cmds: []string{tp(-9.5, 0.5)}},
			{at: 540, shot: "arena"},
		}
		gifAt, end = [2]int{232, 232 + 200}, 560
	} else {
		// In from the side with the most open sea.
		sea := func(sign float64) (n int) {
			for d := 3.0; d < 30; d++ {
				if twin.TileAt(int(lx+sign*d), int(ly+2)) == sim.TileWater && twin.TileAt(int(lx+sign*d), int(ly-2)) == sim.TileWater {
					n++
				}
			}
			return n
		}
		sx := -1.0 // the wreck lies on that side of the lair
		if sea(-1) < 24 && sea(1) > sea(-1) {
			sx = 1
		}
		at := func(d, dy float64) string { return tp(sx*d, dy) }
		steps = []encStep{
			{at: 20, cmds: []string{"god on", clock, "weather clear", at(22, 2), "boat"}},
			{at: 150, shot: "approach"},
			{at: 160, cmds: []string{at(12.7, 2)}},
			{at: 330, shot: "lair"},
			{at: 340, cmds: []string{at(3, 1.5)}},
			{at: 340 + 84, shot: "intro"},
			{at: 560, cmds: []string{at(17.5, 1)}},
			{at: 640, shot: "arena"},
		}
		gifAt, end = [2]int{332, 332 + 200}, 660
	}
	if !gif {
		gifAt = [2]int{}
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := srv.ConnectLocal()
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("shot")
	ebiten.SetTPS(ebiten.SyncWithFPS) // a frame drawn for every frame played, however long it takes
	game := client.New(conn, "Tien")
	if dir := filepath.Dir(out); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	e := &encounter{inner: game, srv: srv, conn: conn, steps: steps, out: out, gif: gifAt, end: end}
	if err := ebiten.RunGame(e); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if e.err != nil {
		log.Fatal(e.err)
	}
}
