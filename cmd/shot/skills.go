package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// The -skills shots: a real game (a dev-mode server and a client) where a
// beast of each kind asked for makes each of its moves at the player (the
// dev console's skill command), and a GIF per kind of every move, wound up,
// struck and recovered, cropped about the player at the screen's own size.
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -skills rex,trike -o /tmp/skills/x.png
//
// writes /tmp/skills/rex.gif and /tmp/skills/trike.gif (and stills of the
// key moments, <kind>_<move>_<n>.png, for checking).

// skillShot is one move to film.
type skillShot struct {
	kind  sim.CreatureKind
	name  string // the dev name of the kind
	skill sim.SkillID
	start int // the frame its command is sent
	end   int
}

type skillFilm struct {
	inner      ebiten.Game
	conn       client.Conn
	srv        *server.Server
	shots      []skillShot
	setup      map[int][]string // dev lines to send at a frame
	n, end     int
	crop       image.Rectangle
	frames     map[string][]*image.RGBA // by kind name
	delays     map[string][]int
	labels     map[string][]string
	dir        string
	err        error
	fixW, fixH int
}

const (
	skillLead = 6  // frames before a move's command, to show it standing
	skillTail = 24 // frames after its recovery
	skillStep = 3  // game frames to a GIF frame: 20 a second, a frame a tick
)

func (e *skillFilm) Update() error {
	if e.n >= e.end || e.err != nil {
		return ebiten.Termination
	}
	e.n++
	if e.n%(60/sim.TicksPerSecond) == 0 {
		e.srv.Tick()
	}
	for _, line := range e.setup[e.n] {
		if err := e.conn.Send(protocol.ClientMsg{Kind: protocol.ClientDev, Dev: line}); err != nil {
			e.err = err
		}
	}
	return e.inner.Update()
}

func (e *skillFilm) Draw(screen *ebiten.Image) {
	e.inner.Draw(screen)
	for _, s := range e.shots {
		if e.n < s.start-skillLead || e.n > s.end || (e.n-s.start)%skillStep != 0 {
			continue
		}
		dk := sim.Skill(s.skill)
		tk := (e.n - s.start) / 3
		busy := tk >= dk.Windup-3 && tk <= dk.Windup+dk.Active+4 // round the blow: every tick
		if !busy && (e.n-s.start)%(2*skillStep) != 0 {
			continue // elsewhere every other tick
		}
		delay := 10
		if busy {
			delay = 5
		}
		e.delays[s.name] = append(e.delays[s.name], delay)
		full := image.NewRGBA(screen.Bounds())
		screen.ReadPixels(full.Pix)
		if (dk.Fx&sim.FxPin != 0 || dk.Stun > 0) && tk == dk.Windup+dk.Active+4 {
			writePNG(filepath.Join(e.dir, fmt.Sprintf("%s_%s_screen.png", s.name, dk.Name)), full) // the whole screen: the HUD's chips
		}
		img := image.NewRGBA(image.Rect(0, 0, e.crop.Dx(), e.crop.Dy()))
		draw.Draw(img, img.Bounds(), full, e.crop.Min, draw.Src)
		d := sim.Skill(s.skill)
		ticks := (e.n - s.start) / 3
		phase := "đứng"
		switch {
		case ticks < 1:
		case ticks <= d.Windup:
			phase = fmt.Sprintf("lấy đà %.2f s", float64(ticks)/sim.TicksPerSecond)
		case ticks <= d.Windup+d.Active+1:
			phase = "RA ĐÒN"
		case ticks <= d.Windup+d.Active+d.Recover+1:
			phase = "hồi (sơ hở)"
		}
		caption(img, fmt.Sprintf("%s — %s (lấy đà %.2f s, hồi %.2f s)", s.kind.Name(), d.Label,
			float64(d.Windup)/sim.TicksPerSecond, float64(d.Recover)/sim.TicksPerSecond), phase)
		e.frames[s.name] = append(e.frames[s.name], img)
		if ticks == d.Windup-2 || ticks == d.Windup+1 || ticks == d.Windup+d.Active+d.Recover/2 {
			writePNG(filepath.Join(e.dir, fmt.Sprintf("%s_%s_%03d.png", s.name, d.Name, ticks)), img)
		}
	}
}

// Layout draws on a screen exactly as big as asked, whatever the window (as -exact).
func (e *skillFilm) Layout(w, h int) (int, int) {
	e.inner.Layout(e.fixW, e.fixH)
	return e.fixW, e.fixH
}

// caption writes title and the phase across the top of img, on a dark band.
func caption(img *image.RGBA, title, phase string) {
	band := image.Rect(0, 0, img.Bounds().Dx(), 40)
	draw.Draw(img, band, image.NewUniform(color.RGBA{12, 14, 18, 200}), image.Point{}, draw.Over)
	big := func(x, y int, s string, col color.Color, k int) {
		small := image.NewRGBA(image.Rect(0, 0, 8*len(s)+16, 16))
		d := font.Drawer{Dst: small, Src: image.NewUniform(col), Face: bitmapfont.Face, Dot: fixed.P(0, 12)}
		d.DrawString(s)
		for py := range small.Bounds().Dy() {
			for px := range small.Bounds().Dx() {
				if c := small.RGBAAt(px, py); c.A > 0 {
					for j := range k {
						for i := range k {
							img.SetRGBA(x+px*k+i, y+py*k+j, c)
						}
					}
				}
			}
		}
	}
	big(12, 6, title, color.RGBA{240, 240, 230, 255}, 2)
	col := color.RGBA{255, 220, 110, 255}
	if strings.HasPrefix(phase, "RA") {
		col = color.RGBA{255, 90, 70, 255}
	}
	big(img.Bounds().Dx()-260, 6, phase, col, 2)
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Print(err)
	}
}

// openSpot is the spot within reach of (x0, y0) with the fewest things
// standing west of it and before it, on open walkable ground, for a beast to
// move on in plain view.
func openSpot(w *sim.World, x0, y0 float64, r int) (float64, float64, bool) {
	bestN, bx, by := 1<<30, 0.0, 0.0
	for d := 0; d < 60; d += 2 {
		for i := 0; i < 8*max(d, 1); i += 2 {
			cx, cy := ring(int(x0), int(y0), d, i)
			ok := true
			for y := cy - 3; y <= cy+5 && ok; y++ {
				for x := cx - r - 2; x <= cx+3 && ok; x++ {
					if w.TileAt(x, y) != sim.TileGrass {
						ok = false
					}
				}
			}
			if !ok {
				continue
			}
			n := 0
			w.EachEntityIn(cx-r-2, cy-4, cx+3, cy+7, func(sim.Entity) { n++ })
			w.EachEntityIn(cx-2, cy-2, cx+2, cy+2, func(sim.Entity) { n += 100 }) // nothing by the player: no label over it
			if n < bestN {
				bestN, bx, by = n, float64(cx)+0.5, float64(cy)+0.5
			}
		}
	}
	return bx, by, bestN < 1<<30
}

// shoreSpot is a tile of land with open deep water east of it.
func shoreSpot(w *sim.World, x0, y0 float64) (float64, float64, bool) {
	ww, hh := w.Size()
	nearLair := func(x, y int) bool {
		for _, b := range w.Bosses() {
			if math.Hypot(b.LairX-float64(x), b.LairY-float64(y)) < 45 {
				return true
			}
		}
		return false
	}
	for d := 0; d < 160; d++ {
		for i := 0; i < 8*max(d, 1); i++ {
			cx, cy := ring(int(x0), int(y0), d, i)
			if cx < 2 || cy < 4 || cx >= ww-12 || cy >= hh-4 || !w.TileAt(cx, cy).Walkable() || nearLair(cx, cy) {
				continue
			}
			ok := true
			for y := cy - 3; y <= cy+3 && ok; y++ {
				for x := cx + 1; x <= cx+10 && ok; x++ {
					if w.TileAt(x, y) != sim.TileWater {
						ok = false
					}
				}
			}
			if ok {
				return float64(cx) + 0.5, float64(cy) + 0.5, true
			}
		}
	}
	return 0, 0, false
}

// runSkills films every move of the kinds named (a comma list, or all) and
// writes a GIF of each kind's moves into dir.
func runSkills(which, dir string, seed uint64, size, winW, winH int) {
	cfg := server.Config{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest, Dev: true}
	twin, err := sim.Generate(sim.GenConfig{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest})
	if err != nil {
		log.Fatal(err)
	}
	sx, sy := twin.Spawn()
	log.Printf("spawn %.1f %.1f tile %v", sx, sy, twin.TileAt(int(sx), int(sy)))
	lx, ly, ok := openSpot(twin, sx, sy, 9)
	if !ok {
		log.Fatal("no open meadow near the spawn")
	}
	cx, cy, _ := shoreSpot(twin, sx, sy)
	names := map[sim.CreatureKind]string{
		sim.CreatureRex: "rex", sim.CreatureRaptor: "dakota", sim.CreatureAchero: "achero", sim.CreatureTroodon: "troodon",
		sim.CreaturePtero: "ptero", sim.CreatureQuetzal: "quetzal", sim.CreaturePlesio: "plesio", sim.CreatureSabre: "sabre",
		sim.CreatureTrike: "trike", sim.CreatureToro: "toro", sim.CreatureStego: "stego", sim.CreatureBrachio: "brachio",
		sim.CreatureAnkylo: "ankylo", sim.CreatureAnzu: "anzu", sim.CreatureThesc: "thesc", sim.CreatureOrnitho: "ornitho",
		sim.CreatureMammoth: "mammoth",
	}
	order := []sim.CreatureKind{sim.CreatureRex, sim.CreatureRaptor, sim.CreatureAchero, sim.CreatureTroodon, sim.CreaturePtero,
		sim.CreatureQuetzal, sim.CreaturePlesio, sim.CreatureSabre, sim.CreatureTrike, sim.CreatureToro, sim.CreatureStego,
		sim.CreatureBrachio, sim.CreatureAnkylo, sim.CreatureAnzu, sim.CreatureThesc, sim.CreatureOrnitho, sim.CreatureMammoth}
	want := map[string]bool{}
	for _, s := range strings.Split(which, ",") {
		want[strings.TrimSpace(s)] = true
	}
	e := &skillFilm{setup: map[int][]string{}, frames: map[string][]*image.RGBA{}, delays: map[string][]int{}, dir: dir}
	at := 30
	e.setup[20] = []string{"weather clear", "time 0.3"}
	for _, k := range order {
		name := names[k]
		if !want["all"] && !want[name] {
			continue
		}
		place := fmt.Sprintf("tp %.1f %.1f", lx, ly)
		clock := "time 0.3"
		switch k {
		case sim.CreatureTroodon:
			clock = "time 0.86"
		case sim.CreatureSabre, sim.CreatureMammoth:
			clock = "time 0.35"
		}
		for _, id := range sim.SkillsOf(k) {
			d := sim.Skill(id)
			p := place
			if k == sim.CreaturePlesio {
				p = fmt.Sprintf("tp %.1f %.1f", cx, cy)
				if d.Need == sim.NeedWet {
					p = fmt.Sprintf("tp %.1f %.1f", cx+2, cy)
				}
			}
			e.setup[at] = []string{"kill 30", clock, p, "heal"}
			start := at + 30
			e.setup[start] = []string{fmt.Sprintf("skill %s %s", name, d.Name)}
			end := start + 3*(d.Windup+d.Active+d.Recover+2) + skillTail
			if d.Fx&sim.FxPin != 0 {
				end += 3 * sim.PinHold
			}
			if d.Torch > 0 {
				end += 3 * d.Torch
			}
			e.shots = append(e.shots, skillShot{kind: k, name: name, skill: id, start: start, end: end})
			at = end + 10
		}
	}
	if len(e.shots) == 0 {
		log.Fatalf("no beast with moves named in %q", which)
	}
	e.end = at + 5
	w, h := winW, winH
	k := float64(h) / 1080 // the crop is laid out for 1080p
	sc := func(v float64) int { return int(math.Round(v * k)) }
	e.crop = image.Rect(w/2-sc(660), h/2-sc(370), w/2+sc(520), h/2+sc(250)).Intersect(image.Rect(0, 0, w, h))

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
	ebiten.SetTPS(ebiten.SyncWithFPS)
	game := client.New(conn, "Tien")
	_ = os.MkdirAll(dir, 0o755)
	e.inner, e.conn, e.srv, e.fixW, e.fixH = game, conn, srv, winW, winH
	if err := ebiten.RunGame(e); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if e.err != nil {
		log.Fatal(e.err)
	}
	for name, frames := range e.frames {
		path := filepath.Join(dir, name+".gif")
		if err := writeSkillGIF(path, frames, e.delays[name]); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s (%d frames)", path, len(frames))
	}
}

// writeSkillGIF encodes frames, a tick apart, with one palette made from them.
func writeSkillGIF(path string, frames []*image.RGBA, delays []int) error {
	pal := skillPalette(frames)
	g := &gif.GIF{}
	cache := map[color.RGBA]uint8{}
	var prev *image.Paletted
	for _, f := range frames {
		b := f.Bounds()
		p := image.NewPaletted(b, pal)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := f.RGBAAt(x, y)
				c.A = 255
				i, ok := cache[c]
				if !ok {
					i = uint8(pal[:len(pal)-1].Index(c))
					cache[c] = i
				}
				p.SetColorIndex(x, y, i)
			}
		}
		// Only what changed since the frame before is kept: the rest is
		// left transparent, over the frame before.
		sub := p
		if prev != nil {
			r := changed(prev, p)
			if r.Empty() {
				r = image.Rect(0, 0, 1, 1)
			}
			sub = image.NewPaletted(r, pal)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					i := p.ColorIndexAt(x, y)
					if i == prev.ColorIndexAt(x, y) {
						i = uint8(len(pal) - 1)
					}
					sub.SetColorIndex(x, y, i)
				}
			}
		}
		g.Image = append(g.Image, sub)
		g.Delay = append(g.Delay, delays[len(g.Image)-1])
		g.Disposal = append(g.Disposal, gif.DisposalNone)
		prev = p
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, g)
}

// changed is the box round the pixels that differ between a and b.
func changed(a, b *image.Paletted) image.Rectangle {
	r := image.Rectangle{}
	bb := b.Bounds()
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			if a.ColorIndexAt(x, y) != b.ColorIndexAt(x, y) {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

// skillPalette is the 256 colours most used in frames, bucketed finely.
func skillPalette(frames []*image.RGBA) color.Palette {
	type acc struct{ r, g, b, n int }
	buckets := map[int]*acc{}
	for i, f := range frames {
		if i%3 != 0 && i != len(frames)-1 {
			continue
		}
		pix := f.Pix
		for j := 0; j < len(pix); j += 4 {
			r, g, b := int(pix[j]), int(pix[j+1]), int(pix[j+2])
			k := (r>>3)<<10 | (g>>3)<<5 | b>>3
			a := buckets[k]
			if a == nil {
				a = &acc{}
				buckets[k] = a
			}
			a.r, a.g, a.b, a.n = a.r+r, a.g+g, a.b+b, a.n+1
		}
	}
	type kv struct {
		c color.RGBA
		n float64
	}
	var all []kv
	for _, a := range buckets {
		c := color.RGBA{uint8(a.r / a.n), uint8(a.g / a.n), uint8(a.b / a.n), 255}
		// Rare but vivid colours (the zone's red, an eye) count for more.
		mx := max(c.R, c.G, c.B)
		mn := min(c.R, c.G, c.B)
		all = append(all, kv{c, math.Sqrt(float64(a.n)) * (1 + float64(mx-mn)/64)})
	}
	// Pick greedily: the heaviest first, skipping ones too like those taken.
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	var pal color.Palette
	near := func(c color.RGBA, d int) bool {
		for _, q := range pal {
			p := q.(color.RGBA)
			if abs(int(p.R)-int(c.R))+abs(int(p.G)-int(c.G))+abs(int(p.B)-int(c.B)) < d {
				return true
			}
		}
		return false
	}
	for _, d := range []int{24, 12, 0} {
		for _, k := range all {
			if len(pal) >= 255 {
				break
			}
			if !near(k.c, d) {
				pal = append(pal, k.c)
			}
		}
	}
	for len(pal) < 2 {
		pal = append(pal, color.RGBA{0, 0, 0, 255})
	}
	return append(pal, color.RGBA{}) // the last: unchanged since the frame before (transparent)
}

// ring is the i-th tile of the square ring d tiles round (x, y), 8d of them.
func ring(x, y, d, i int) (int, int) {
	if d == 0 {
		return x, y
	}
	side, k := i/(2*d), i%(2*d)
	switch side {
	case 0:
		return x - d + k, y - d
	case 1:
		return x + d, y - d + k
	case 2:
		return x + d - k, y + d
	}
	return x - d, y + d - k
}
