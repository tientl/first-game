package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// The -bossfight shots: a real game (a dev-mode server and a client, the
// demo's two bosses at their lairs of sim.StyleLairs) taken through a
// scene with the dev console's commands, writing stills and GIFs into -o's
// folder:
//
//	titan: titan_lair.png, titan_camp.png, titan_fight.gif, titan_phase2.gif
//	shark: shark_atoll.png, shark_fight.gif, shark_reefbreak.gif
//	map:   boss_map.png (the revealed map, both lairs and their camps marked)
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -size 1024 -bossfight titan -o out/x.png

// bfStep is something the scene does at a frame.
type bfStep struct {
	at    int
	cmds  []string
	zoom  *int   // the view's zoom from now on
	shot  string // a still to write, by name
	gifOn string // begin taking frames for this GIF...
	gifOff bool  // ...and stop (the GIF is written at the end)
	every int    // frames between a GIF's frames
	overlay string
}

type bfGIF struct {
	frames []*image.RGBA
	delays []int
}

type bossFightShot struct {
	inner   ebiten.Game
	srv     *server.Server
	conn    client.Conn
	game    *client.Game
	steps   []bfStep
	dir     string
	n, end  int
	pending []string
	gifName string
	every   int
	gifs    map[string]*bfGIF
	order   []string
	err     error
}

func (e *bossFightShot) Update() error {
	if e.n >= e.end || e.err != nil {
		return ebiten.Termination
	}
	e.n++
	if e.n%(60/sim.TicksPerSecond) == 0 {
		e.srv.Tick()
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
		if s.zoom != nil {
			client.SetZoom(*s.zoom)
			e.game.QuietZoom()
		}
		if s.overlay != "" {
			e.game.ShowOverlay(s.overlay)
		}
		if s.shot != "" {
			e.pending = append(e.pending, s.shot)
		}
		if s.gifOn != "" {
			e.gifName, e.every = s.gifOn, max(1, s.every)
			if e.gifs[s.gifOn] == nil {
				e.gifs[s.gifOn] = &bfGIF{}
				e.order = append(e.order, s.gifOn)
			}
		}
		if s.gifOff {
			e.gifName = ""
		}
	}
	return e.inner.Update()
}

func (e *bossFightShot) Draw(screen *ebiten.Image) {
	e.inner.Draw(screen)
	takeGIF := e.gifName != "" && e.n%e.every == 0
	if len(e.pending) == 0 && !takeGIF {
		return
	}
	img := image.NewRGBA(screen.Bounds())
	screen.ReadPixels(img.Pix)
	for _, s := range e.pending {
		name := filepath.Join(e.dir, s+".png")
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
	e.pending = nil
	if takeGIF {
		g := e.gifs[e.gifName]
		if len(g.frames)%10 == 0 { // a still of every tenth, full size, to look at
			rev := filepath.Join(e.dir, "review")
			_ = os.MkdirAll(rev, 0o755)
			if f, err := os.Create(filepath.Join(rev, fmt.Sprintf("%s_%03d.png", e.gifName, len(g.frames)))); err == nil {
				_ = png.Encode(f, img)
				f.Close()
			}
		}
		g.frames = append(g.frames, shrinkFrame(img, 800))
		g.delays = append(g.delays, e.every*100/60)
	}
}

func (e *bossFightShot) Layout(w, h int) (int, int) { return e.inner.Layout(w, h) }

// shrinkFrame scales img down to w pixels wide, averaging the pixels.
func shrinkFrame(img *image.RGBA, w int) *image.RGBA {
	b := img.Bounds()
	h := b.Dy() * w / b.Dx()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	fx, fy := float64(b.Dx())/float64(w), float64(b.Dy())/float64(h)
	for y := range h {
		y0, y1 := int(float64(y)*fy), int(float64(y+1)*fy)
		for x := range w {
			x0, x1 := int(float64(x)*fx), int(float64(x+1)*fx)
			var r, g, bl, n int
			for yy := y0; yy < max(y1, y0+1); yy++ {
				for xx := x0; xx < max(x1, x0+1); xx++ {
					c := img.RGBAAt(xx, yy)
					r, g, bl, n = r+int(c.R), g+int(c.G), bl+int(c.B), n+1
				}
			}
			out.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255})
		}
	}
	return out
}

// runBossFight plays scene which and writes its pictures into dir.
func runBossFight(which, dir string, seed uint64, size, winW, winH int) {
	sim.SetDemo(true)
	twin, err := sim.Generate(sim.GenConfig{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest})
	if err != nil {
		log.Fatal(err)
	}
	var titan, shark sim.BossStatus
	for _, b := range twin.Bosses() {
		switch b.Kind {
		case sim.CreatureTitanoboa:
			titan = b
		case sim.CreatureMegalodon:
			shark = b
		}
	}
	tp := func(x, y float64) string { return fmt.Sprintf("tp %.2f %.2f", x, y) }
	z := func(v int) *int { return &v }
	setup := []string{"god on", "time 0.36", "weather clear", "boss calm on", "reveal"}
	var steps []bfStep
	end := 0
	switch which {
	case "titan":
		lx, ly := titan.LairX, titan.LairY
		a := math.Atan2(titan.CampY-ly, titan.CampX-lx)
		in := func(d float64) string { return tp(lx+math.Cos(a)*d, ly+math.Sin(a)*d) }
		steps = []bfStep{
			{at: 20, cmds: append(setup, in(7.5))},
			{at: 40, zoom: z(-1)},
			{at: 200, shot: "titan_lair"},
			{at: 210, zoom: z(0), cmds: []string{in(15.5)}},
			{at: 330, shot: "titan_gate"},
			{at: 340, cmds: []string{"tp trại 1"}},
			{at: 470, shot: "titan_camp"},
			// The fight: into the courtyard, the snake wakes, its moves.
			{at: 480, cmds: []string{in(6.5), "boss calm off"}},
			{at: 800, cmds: []string{"boss skill strike"}, gifOn: "titan_fight", every: 6},
			{at: 940, cmds: []string{"boss skill tail"}},
			{at: 1080, cmds: []string{"boss skill venom"}},
			{at: 1220, cmds: []string{"boss skill coil"}},
			{at: 1440, gifOff: true},
			{at: 1450, cmds: []string{"heal", "boss phase 2", in(6.5)}},
			{at: 1470, cmds: []string{"boss skill dive"}, gifOn: "titan_phase2", every: 6},
			{at: 1640, cmds: []string{"boss skill slam"}},
			{at: 1800, cmds: []string{"boss phase 3"}},
			{at: 1830, cmds: []string{"boss skill slam"}},
			{at: 2000, cmds: []string{"boss skill strike"}},
			{at: 2120, gifOff: true},
			{at: 2125, shot: "titan_flood"},
		}
		end = 2130
	case "shark":
		lx, ly := shark.LairX, shark.LairY
		a := math.Atan2(shark.CampY-ly, shark.CampX-lx) // the side towards home
		at := func(d float64) string { return tp(lx+math.Cos(a)*d, ly+math.Sin(a)*d) }
		steps = []bfStep{
			{at: 20, cmds: append(setup, at(-4))},
			{at: 40, zoom: z(-1)},
			{at: 200, shot: "shark_atoll"},
			{at: 210, zoom: z(0), cmds: []string{"tp trại 2"}},
			{at: 330, shot: "shark_camp"},
			// On the reef by the lagoon: it wakes, lunges at the edge, breaches.
			{at: 340, cmds: []string{at(16.9), "boss calm off"}},
			{at: 660, cmds: []string{"boss skill lunge"}, gifOn: "shark_fight", every: 6},
			{at: 840, cmds: []string{"boss skill breach"}},
			{at: 1040, cmds: []string{"boss skill lunge"}},
			{at: 1200, gifOff: true},
			// The reef broken under them, and settling again.
			{at: 1210, cmds: []string{"heal", at(17.6)}},
			{at: 1240, cmds: []string{"boss skill ram"}, gifOn: "shark_reefbreak", every: 5},
			{at: 1420, gifOff: true},
			{at: 1420 + 9*60, gifOn: "shark_reefbreak", every: 5},
			{at: 1420 + 13*60, gifOff: true},
		}
		end = 1420 + 13*60 + 10
	case "map":
		steps = []bfStep{
			{at: 20, cmds: append(setup, "tp trùm 1")},
			{at: 120, cmds: []string{"tp trùm 2"}},
			{at: 220, cmds: []string{"tp spawn"}},
			{at: 300, overlay: "map"},
			{at: 400, shot: "boss_map"},
		}
		end = 410
	default:
		log.Fatalf("-bossfight: titan, shark or map, not %q", which)
	}

	srv, err := server.New(server.Config{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest, Dev: true})
	if err != nil {
		log.Fatal(err)
	}
	conn, err := srv.ConnectLocal()
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("shot")
	ebiten.SetTPS(ebiten.SyncWithFPS)
	game := client.New(conn, "Tien")
	game.SetAim(0, 3)
	_ = os.MkdirAll(dir, 0o755)
	e := &bossFightShot{inner: game, srv: srv, conn: conn, game: game, steps: steps, dir: dir, end: end, gifs: map[string]*bfGIF{}}
	if err := ebiten.RunGame(e); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if e.err != nil {
		log.Fatal(e.err)
	}
	for _, name := range e.order {
		g := e.gifs[name]
		path := filepath.Join(dir, name+".gif")
		if err := writeSkillGIF(path, g.frames, g.delays); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s (%d frames)", path, len(g.frames))
	}
}
