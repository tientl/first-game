package main

import (
	"errors"
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// The -combat shots: a real fight on a dev server, the client drawing it as
// the game does (hit-stops, flashes, knockback, shake, effects), the
// player's hands scripted, written as <o>.gif of the middle of the screen.
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -combat feel -o combat_feel.png
//	go run ./cmd/shot -exact -w 1920 -h 1080 -combat shield -o combat_shield.png
//
// feel: a spear and then the battle axe on a raptor and a three-horn.
// shield: a sword and a shield against a raptor: the shield held up (blocks),
// then raised just as it springs (parries), striking back while it reels.

// tapConn passes the server's messages to the client and keeps the last
// snapshot, so the script can see what the beasts are doing.
type tapConn struct {
	inner client.Conn
	ch    chan protocol.ServerMsg
	mu    sync.Mutex
	last  protocol.Snapshot
}

func newTap(c client.Conn) *tapConn {
	t := &tapConn{inner: c, ch: make(chan protocol.ServerMsg, 1024)}
	go func() {
		for m := range c.Messages() {
			if m.Kind == protocol.ServerSnapshot {
				t.mu.Lock()
				t.last = m.Snapshot
				t.mu.Unlock()
			}
			t.ch <- m
		}
		close(t.ch)
	}()
	return t
}

func (t *tapConn) Send(m protocol.ClientMsg) error     { return t.inner.Send(m) }
func (t *tapConn) Messages() <-chan protocol.ServerMsg { return t.ch }
func (t *tapConn) Close() error                        { return t.inner.Close() }

func (t *tapConn) snapshot() protocol.Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}

// fightShot is the game wrapped: the server's clock, the script and the
// frames kept.
type fightShot struct {
	inner      ebiten.Game
	srv        *server.Server
	conn       *tapConn
	n          int
	script     func(f *fightShot, frame int) (in sim.Input, cmds []protocol.ClientMsg)
	from, to   int // frames kept
	step       int
	frames     []*image.RGBA
	strikeAt   map[sim.CreatureID]int // when each beast began its strike, in ticks
	tick       int
	err        error
	cropW, cH  int
	cropOffset int
	w, h       int // the screen, whatever the window
}

func (f *fightShot) Update() error {
	if f.n >= f.to || f.err != nil {
		return ebiten.Termination
	}
	f.n++
	in, cmds := f.script(f, f.n)
	for _, c := range cmds {
		if err := f.conn.Send(c); err != nil {
			f.err = err
		}
	}
	if f.n%(60/sim.TicksPerSecond) == 0 {
		if err := f.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: in}); err != nil {
			f.err = err
		}
		f.srv.Tick()
		f.tick++
		f.noteStrikes()
	}
	return f.inner.Update()
}

// noteStrikes keeps when each beast began its strike.
func (f *fightShot) noteStrikes() {
	s := f.conn.snapshot()
	for _, c := range s.Creatures {
		if c.State == sim.CreatureStrike {
			if _, ok := f.strikeAt[c.ID]; !ok {
				f.strikeAt[c.ID] = f.tick
			}
		} else {
			delete(f.strikeAt, c.ID)
		}
	}
}

// striking is how many ticks the beast nearest the player has been in its
// strike, -1 if none near is striking.
func (f *fightShot) striking() int {
	s := f.conn.snapshot()
	var me protocol.PlayerView
	for _, p := range s.Players {
		me = p
	}
	best := -1
	for _, c := range s.Creatures {
		if at, ok := f.strikeAt[c.ID]; ok && math.Hypot(float64(c.X-me.X), float64(c.Y-me.Y)) < 3.5 {
			best = max(best, f.tick-at)
		}
	}
	return best
}

func (f *fightShot) Draw(screen *ebiten.Image) {
	f.inner.Draw(screen)
	if f.n < f.from || (f.n-f.from)%f.step != 0 {
		return
	}
	b := screen.Bounds()
	full := image.NewRGBA(b)
	screen.ReadPixels(full.Pix)
	cx, cy := b.Dx()/2, b.Dy()/2+f.cropOffset
	r := image.Rect(cx-f.cropW/2, cy-f.cH/2, cx+f.cropW/2, cy+f.cH/2)
	img := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(img, img.Bounds(), full, r.Min, draw.Src)
	f.frames = append(f.frames, img)
}

func (f *fightShot) Layout(w, h int) (int, int) { return f.w, f.h }

func dev(line string) protocol.ClientMsg {
	return protocol.ClientMsg{Kind: protocol.ClientDev, Dev: line}
}

func selectSlot(i int) protocol.ClientMsg {
	return protocol.ClientMsg{Kind: protocol.ClientCommand, Command: sim.Command{Kind: sim.CmdSelect, Slot: uint8(i)}}
}

// runCombat plays fight which and writes it as out's GIF.
func runCombat(which, out string, seed uint64, winW, winH int, dayFrac float64) {
	cfg := server.Config{Seed: seed, Width: 128, Height: 128, Style: sim.StyleLatest, Dev: true}
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	lc, err := srv.ConnectLocal()
	if err != nil {
		log.Fatal(err)
	}
	conn := newTap(lc)
	sword := sim.SwordItem(sim.TierCopper).Def().Name
	setup := []protocol.ClientMsg{dev("god on"), dev("time noon"), dev("weather clear"), dev("kill")}
	var script func(f *fightShot, frame int) (sim.Input, []protocol.ClientMsg)
	from, to := 0, 0
	switch which {
	case "feel":
		from, to = 70, 70+2*230
		script = func(f *fightShot, frame int) (sim.Input, []protocol.ClientMsg) {
			var cmds []protocol.ClientMsg
			switch frame {
			case 5:
				cmds = append(setup, dev("give spear 1"), dev("give battleaxe 1"), selectSlot(0))
			case 30:
				cmds = []protocol.ClientMsg{dev("spawn raptor")}
			case 226:
				cmds = []protocol.ClientMsg{dev("kill")}
			case 230:
				cmds = []protocol.ClientMsg{selectSlot(1), dev("spawn trike")}
			}
			in := sim.Input{Action: frame > 50}
			if frame > 226 {
				in.Action = f.nearest() <= 2.3 // not the piles the raptor left
				if !in.Action {
					in.MoveX = 1 // up to the three-horn
				}
			}
			return in, cmds
		}
	case "shield":
		from, to = 50, 50+2*260
		script = func(f *fightShot, frame int) (sim.Input, []protocol.ClientMsg) {
			var cmds []protocol.ClientMsg
			switch frame {
			case 5:
				cmds = append(setup, dev("give "+sword+" 1"), dev("give shield 1"), selectSlot(0))
			case 30:
				cmds = []protocol.ClientMsg{dev("spawn raptor")}
			}
			in := sim.Input{}
			st := f.striking()
			switch {
			case frame < 18: // a step to face the way it will come from
				in.MoveX = 1
			case frame < 60: // the shield on the back as it comes
			case frame < 300: // held up: blocks
				in.Guard = true
			default: // raised just as it springs: parries; and back at it while it reels
				in.Guard = st >= 4
				s := f.conn.snapshot()
				for _, c := range s.Creatures {
					if c.State == sim.CreatureHurt && !in.Guard {
						in.Action = true
					}
				}
			}
			return in, cmds
		}
	default:
		log.Fatalf("-combat feel or shield, not %q", which)
	}
	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("shot")
	ebiten.SetTPS(ebiten.SyncWithFPS)
	game := client.New(conn, "Tien")
	f := &fightShot{inner: game, srv: srv, conn: conn, script: script, from: from, to: to, step: 2,
		strikeAt: map[sim.CreatureID]int{}, cropW: winW / 3, cH: winH * 2 / 7, cropOffset: -winH / 30, w: winW, h: winH}
	_ = dayFrac
	if err := ebiten.RunGame(&shotGame{f}); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if f.err != nil {
		log.Fatal(f.err)
	}
	if err := writeGIF(gifPath(out), f.frames, 3); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%d frames)", gifPath(out), len(f.frames))
	if err := contactSheet(strings.TrimSuffix(out, ".png")+"_sheet.png", f.frames, 3, 6); err != nil {
		log.Fatal(err)
	}
}

// shotGame is f with a fixed screen size: the window's.
type shotGame struct{ f *fightShot }

func (s *shotGame) Update() error              { return s.f.Update() }
func (s *shotGame) Draw(screen *ebiten.Image)  { s.f.Draw(screen) }
func (s *shotGame) Layout(w, h int) (int, int) { return s.f.Layout(w, h) }

// contactSheet writes every step-th frame of frames tiled cols across, at
// half size, for a look at a GIF in one picture.
func contactSheet(path string, frames []*image.RGBA, step, cols int) error {
	if len(frames) == 0 {
		return nil
	}
	w, h := frames[0].Bounds().Dx()/2, frames[0].Bounds().Dy()/2
	var pick []*image.RGBA
	for i := 0; i < len(frames); i += step {
		pick = append(pick, frames[i])
	}
	rows := (len(pick) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*w, rows*h))
	for i, f := range pick {
		ox, oy := (i%cols)*w, (i/cols)*h
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				sheet.Set(ox+x, oy+y, f.At(x*2, y*2))
			}
		}
	}
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return png.Encode(fh, sheet)
}

// nearest is how far the nearest beast is from the player, in tiles.
func (f *fightShot) nearest() float64 {
	s := f.conn.snapshot()
	var me protocol.PlayerView
	for _, p := range s.Players {
		me = p
	}
	best := math.Inf(1)
	for _, c := range s.Creatures {
		best = math.Min(best, math.Hypot(float64(c.X-me.X), float64(c.Y-me.Y)))
	}
	return best
}
