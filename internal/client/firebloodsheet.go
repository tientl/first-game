package client

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The fire and body sheets, for review, drawn through the game's own code:
//
//	go run ./cmd/shot -exact -w 1920 -h 1080 -firebloodsheet dir
//
// Every picture is first written as a panel into dir/now, at 1080p (one
// image pixel to a screen pixel). If dir/before holds the panels of an
// earlier build (rename its now to before), each sheet sets them side by
// side, before on the left. It must run inside the game loop (Draw).

// fbFire is one fire on the sheets.
type fbFire struct {
	name string
	e    protocol.EntityView
	k    fireKind
	dy   float64 // the flames' root from the entity's point
	hand bool    // a torch in the hand: its flame alone
}

var fbFires = []fbFire{
	{name: "campfire", e: protocol.EntityView{ID: 11, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel}, k: fireCamp, dy: 1},
	{name: "campfire low", e: protocol.EntityView{ID: 12, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel / 5}, k: fireCampLow, dy: 1},
	{name: "hearth out", e: protocol.EntityView{ID: 18, Kind: sim.EntityCampfire, Ready: true, Fuel: 0, Var: uint8(sim.GearPot)}, k: fireCampLow, dy: 1},
	{name: "pot on the fire", e: protocol.EntityView{ID: 17, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel, Var: uint8(sim.GearPot)}, k: fireCook, dy: 1},
	{name: "grill on the fire", e: protocol.EntityView{ID: 19, Kind: sim.EntityCampfire, Ready: true, Fuel: sim.FireStartFuel, Var: uint8(sim.GearGrill)}, k: fireCook, dy: 1},
	{name: "bonfire", e: protocol.EntityView{ID: 14, Kind: sim.EntityBonfire, Ready: true}, k: fireBonfire, dy: 1},
	{name: "torch stand", e: protocol.EntityView{ID: 13, Kind: sim.EntityTorchStand, Ready: true, Fuel: 100}, k: fireTorch},
	{name: "furnace", e: protocol.EntityView{ID: 15, Kind: sim.EntityFurnace, Ready: true, Fuel: 100}, k: fireMouth, dy: -1},
	{name: "oven", e: protocol.EntityView{ID: 16, Kind: sim.EntityOven, Ready: true, Fuel: 100}, k: fireMouth},
	{name: "torch in hand", e: protocol.EntityView{ID: 20}, k: fireHand, hand: true},
}

// fbCorpses is every kind of beast that leaves a body.
func fbCorpses() []sim.CreatureKind {
	var out []sim.CreatureKind
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		switch k {
		case sim.CreatureWorm, sim.CreatureFish, sim.CreatureToad, sim.CreatureGlowToad:
			continue // what they give falls at once: see sim/drops.go leavesBody
		}
		out = append(out, k)
	}
	return out
}

const (
	fbFireW, fbFireH = 64, 64 // a fire's cell, art pixels
	fbGrass          = 0x4f6a40
)

// FireBloodSheets writes the panels into dir/now and, with dir/before, the
// sheets into dir.
func (g *Game) FireBloodSheets(dir string) error {
	keepFrame, keepEnts := g.frame, g.curr.Entities
	defer func() { g.frame, g.curr.Entities = keepFrame, keepEnts }()
	now, before := filepath.Join(dir, "now"), filepath.Join(dir, "before")
	if err := os.MkdirAll(now, 0o755); err != nil {
		return err
	}
	// The fires: every frame of the loop.
	buf := newFineBuf(fbFireW, fbFireH)
	for _, it := range fbFires {
		for f := range fireFrames {
			img := g.fbFirePanel(buf, it, 1200+f*fireStep, false)
			if err := sheetPNG(filepath.Join(now, fbName("fire", it.name, f)), img); err != nil {
				return err
			}
		}
		img := g.fbFirePanel(buf, it, 1200, true)
		if err := sheetPNG(filepath.Join(now, fbName("firenight", it.name, 0)), img); err != nil {
			return err
		}
	}
	// The campfire's loop, a GIF frame every second game frame.
	for _, it := range fbFires[:3] {
		var frames []*image.RGBA
		for t := 0; t < fireFrames*fireStep*2; t += 2 {
			img := g.fbFirePanel(buf, it, 1200+t, false)
			frames = append(frames, enlarge(img, 3))
		}
		if err := sheetGIF(filepath.Join(now, fbName("loop", it.name, -1)+".gif"), frames, 3); err != nil {
			return err
		}
	}
	// The bodies.
	for _, k := range fbCorpses() {
		img := g.fbCorpsePanel(k, false, 1<<20)
		if err := sheetPNG(filepath.Join(now, fbName("corpse", fbKindName(k), 0)), img); err != nil {
			return err
		}
	}
	if err := g.fbExtra(now); err != nil {
		return err
	}
	if _, err := os.Stat(before); err != nil {
		return nil // nothing to set them beside
	}
	return fbCompose(dir, now, before)
}

func fbName(what, name string, f int) string {
	s := what + "_" + fbSlug(name)
	if f >= 0 {
		s += fmt.Sprintf("_%02d", f)
	}
	return s + ".png"
}

func fbSlug(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c == ' ' {
			b[i] = '_'
		}
	}
	return string(b)
}

func fbKindName(k sim.CreatureKind) string {
	if n, ok := map[sim.CreatureKind]string{
		sim.CreatureRabbit: "rabbit", sim.CreatureSnake: "snake", sim.CreatureCrab: "crab", sim.CreatureCroc: "croc",
		sim.CreatureTrike: "trike", sim.CreatureStego: "stego", sim.CreatureBrachio: "brachio", sim.CreatureRaptor: "raptor",
		sim.CreatureRex: "rex", sim.CreaturePtero: "ptero", sim.CreaturePlesio: "plesio", sim.CreatureMammoth: "mammoth",
		sim.CreatureSabre: "sabre", sim.CreatureOrnitho: "ornitho", sim.CreatureThesc: "thesc", sim.CreatureAnzu: "anzu",
		sim.CreatureToro: "toro", sim.CreatureAchero: "achero", sim.CreatureQuetzal: "quetzal", sim.CreatureBrachy: "brachy",
		sim.CreatureChampso: "champso", sim.CreatureSpider: "spider", sim.CreatureBear: "bear", sim.CreatureTribesman: "tribesman",
		sim.CreatureMosa: "mosa", sim.CreatureKraken: "kraken", sim.CreatureScorpion: "scorpion", sim.CreatureMonkey: "monkey",
		sim.CreatureTroodon: "troodon", sim.CreatureTitanoboa: "titanoboa", sim.CreatureSpino: "spino",
		sim.CreatureMammothKing: "mammoth king", sim.CreatureCarcharo: "carcharo", sim.CreatureGiganoto: "giganoto",
		sim.CreatureMegalodon: "megalodon", sim.CreatureAnkylo: "ankylo",
	}[k]; ok {
		return n
	}
	return fmt.Sprintf("kind %d", k)
}

// fbFirePanel draws fire it at game frame t on buf, with its smoke, warmth
// and embers; night: in the dark, as the lighting leaves it round the fire.
func (g *Game) fbFirePanel(buf *fineBuf, it fbFire, t int, night bool) *image.RGBA {
	g.frame = t
	bg := hex(fbGrass)
	dark := 0.15
	if night {
		bg, dark = color.RGBA{26, 30, 40, 255}, 0.85
	}
	buf.img.Fill(bg)
	d := drawable{sx: fbFireW / 2, sy: fbFireH - 16, idx: 0}
	align(buf, d.sx, d.sy)
	e := it.e
	e.TX, e.TY = 40, 40
	g.curr.Entities = []protocol.EntityView{e}
	src := fireSource{d.sx, d.sy + it.dy, it.k, uint32(e.ID)}
	if it.k == fireTorch {
		src.y = d.sy - g.art.tribal.torchStick.ay + 4
	}
	if it.hand {
		src.y = d.sy - 18
	}
	if fireHD && (e.Fuel > 0 || e.Kind == sim.EntityBonfire || it.hand) {
		g.drawFireSmokeOf(buf, []fireSource{src})
	}
	if it.hand {
		drawFire(buf, fireHand, g.fireFrame(uint32(e.ID)), src.x, src.y)
	} else {
		g.drawEntity(buf, e, d)
	}
	if fireHD && (e.Fuel > 0 || e.Kind == sim.EntityBonfire || it.hand) {
		g.drawFireFXOf(buf, []fireSource{src}, dark)
	}
	img := image.NewRGBA(buf.img.Bounds())
	buf.img.ReadPixels(img.Pix)
	return img
}

// fbCorpseCell is the size of the cell a body of kind k is drawn in, art
// pixels: room for the beast as it stands, and more.
func (g *Game) fbCorpseCell(k sim.CreatureKind) (w, h int) {
	s, _ := g.creatureSprite(protocol.CreatureView{Kind: k, State: sim.CreatureIdle}, false)
	if s.img == nil {
		return 48, 40
	}
	return max(48, int(s.w()*1.25)+16), max(40, int(s.h()*0.9)+20)
}

// fbCorpsePanel draws the body of a beast of kind k, age game frames after
// it fell (huge: long dead), on the grass.
func (g *Game) fbCorpsePanel(k sim.CreatureKind, left bool, age int) *image.RGBA {
	w, h := g.fbCorpseCell(k)
	buf := newFineBuf(w, h)
	buf.img.Fill(hex(fbGrass))
	e := protocol.EntityView{ID: sim.EntityID(5000 + int(k)), Kind: sim.EntityCorpse, TX: 40, TY: 40, Ready: true, Var: uint8(k), Big: left}
	g.curr.Entities = []protocol.EntityView{e}
	d := drawable{sx: float64(w / 2), sy: float64(h) - 14, idx: 0}
	align(buf, d.sx, d.sy)
	g.fbAgeCorpse(e, age)
	g.drawEntity(buf, e, d)
	img := image.NewRGBA(buf.img.Bounds())
	buf.img.ReadPixels(img.Pix)
	return img
}

// enlarge is img z times bigger.
func enlarge(img *image.RGBA, z int) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*z, b.Dy()*z))
	blowUp(out, img, 0, 0, z)
	return out
}

func fbLoad(path string) *image.RGBA {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return nil
	}
	out := image.NewRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out
}

// fbSheet lays out rows of panels, z times bigger, under a title; a nil
// panel is left empty.
func fbSheet(path, title string, rows [][]*image.RGBA, names []string, heads []string, z int) error {
	cw, ch := 0, 0
	for _, r := range rows {
		for _, p := range r {
			if p != nil {
				cw, ch = max(cw, p.Bounds().Dx()*z), max(ch, p.Bounds().Dy()*z)
			}
		}
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	const top, gap = 34, 6
	sheet := image.NewRGBA(image.Rect(0, 0, cols*(cw+gap)+gap, top+len(rows)*(ch+gap+16)))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{28, 30, 28, 255}}, image.Point{}, draw.Src)
	fbLabel(sheet, 6, 6, title)
	for j, hd := range heads {
		fbLabel(sheet, gap+j*(cw+gap)+4, 22, hd)
	}
	for i, r := range rows {
		y := top + i*(ch+gap+16)
		if i < len(names) {
			fbLabel(sheet, 6, y+2, names[i])
		}
		for j, p := range r {
			if p != nil {
				blowUp(sheet, p, gap+j*(cw+gap), y+14, z)
			}
		}
	}
	return sheetPNG(path, sheet)
}

// fbLabel is label with digits and a few marks too.
func fbLabel(img *image.RGBA, x, y int, s string) {
	digits := map[rune][5]uint8{
		'0': {7, 5, 5, 5, 7}, '1': {2, 6, 2, 2, 7}, '2': {6, 1, 2, 4, 7}, '3': {6, 1, 2, 1, 6}, '4': {5, 5, 7, 1, 1},
		'5': {7, 4, 6, 1, 6}, '6': {3, 4, 7, 5, 7}, '7': {7, 1, 2, 2, 2}, '8': {7, 5, 7, 5, 7}, '9': {7, 5, 7, 1, 6},
		'.': {0, 0, 0, 0, 2}, ',': {0, 0, 0, 2, 4}, ':': {0, 2, 0, 2, 0}, '-': {0, 0, 7, 0, 0}, '/': {1, 1, 2, 4, 4},
		'+': {0, 2, 7, 2, 0}, 'x': {0, 5, 2, 5, 0},
	}
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if gl, ok := digits[r]; ok && r != 'x' {
			for row := 0; row < 5; row++ {
				for col := 0; col < 3; col++ {
					if gl[row]>>(2-col)&1 == 1 {
						for dy := 0; dy < 2; dy++ {
							for dx := 0; dx < 2; dx++ {
								img.SetRGBA(x+i*8+col*2+dx, y+row*2+dy, color.RGBA{255, 255, 255, 255})
							}
						}
					}
				}
			}
			continue
		}
		glyph(img, x+i*8, y, r)
	}
}

// fbCompose sets the panels of before and now side by side.
func fbCompose(dir, now, before string) error {
	both := func(name string) (*image.RGBA, *image.RGBA) {
		return fbLoad(filepath.Join(before, name)), fbLoad(filepath.Join(now, name))
	}
	// The campfire, burning well, low and out: three frames of each, before
	// and after, four times 1080p.
	var rows [][]*image.RGBA
	var names []string
	for _, it := range fbFires[:3] {
		var r []*image.RGBA
		for _, set := range []string{before, now} {
			for _, f := range []int{0, 4, 8} {
				r = append(r, fbLoad(filepath.Join(set, fbName("fire", it.name, f))))
			}
		}
		rows, names = append(rows, r), append(names, it.name)
	}
	if err := fbSheet(filepath.Join(dir, "campfire_compare.png"), "campfire: before (left 3) and after (right 3), frames 0 4 8, 4x 1080p",
		rows, names, []string{"before", "", "", "after"}, 4); err != nil {
		return err
	}
	// Every fire: before above, after below, two frames each, twice 1080p.
	rows, names = nil, nil
	for _, set := range []string{before, now} {
		for _, f := range []int{0, 6} {
			var r []*image.RGBA
			for _, it := range fbFires {
				r = append(r, fbLoad(filepath.Join(set, fbName("fire", it.name, f))))
			}
			rows = append(rows, r)
			names = append(names, map[string]string{before: "before", now: "after"}[set]+fmt.Sprintf(" frame %d", f))
		}
	}
	var heads []string
	for _, it := range fbFires {
		heads = append(heads, it.name)
	}
	if err := fbSheet(filepath.Join(dir, "fires_all.png"), "every fire, 2x 1080p", rows, names, heads, 2); err != nil {
		return err
	}
	// The same at night.
	rows = [][]*image.RGBA{nil, nil}
	for i, set := range []string{before, now} {
		for _, it := range fbFires {
			rows[i] = append(rows[i], fbLoad(filepath.Join(set, fbName("firenight", it.name, 0))))
		}
	}
	if err := fbSheet(filepath.Join(dir, "fires_night.png"), "every fire in the dark, 2x 1080p", rows, []string{"before", "after"}, heads, 2); err != nil {
		return err
	}
	// Every body: before and after side by side, twice 1080p.
	ks := fbCorpses()
	rows, names = nil, nil
	for i := 0; i < len(ks); i += 3 {
		var r []*image.RGBA
		var n string
		for _, k := range ks[i:min(len(ks), i+3)] {
			b, a := both(fbName("corpse", fbKindName(k), 0))
			r = append(r, b, a)
			n += fmt.Sprintf("%-40s", fbKindName(k))
		}
		rows, names = append(rows, r), append(names, n)
	}
	return fbSheet(filepath.Join(dir, "corpses_compare.png"), "every body: before | after, three kinds to a row, 2x 1080p", rows, names, nil, 2)
}

// fbExtra writes what has no before: a body falling and its blood
// spreading, frame by frame (fall_<kind>.png), and every body lying fresh.
func (g *Game) fbExtra(dir string) error {
	for _, k := range []sim.CreatureKind{sim.CreatureRaptor, sim.CreatureTrike, sim.CreatureBear, sim.CreatureRabbit} {
		ages := []int{0, 4, 8, 12, 16, 20, 30, 45, 60, 90, 60 * 20, 60 * 120}
		var row []*image.RGBA
		for _, a := range ages {
			row = append(row, g.fbCorpsePanel(k, false, a))
		}
		var names []string
		for _, a := range ages {
			names = append(names, fmt.Sprintf("%.2fs", float64(a)/60))
		}
		if err := fbSheet(filepath.Join(dir, "..", "fall_"+fbKindName(k)+".png"),
			fbKindName(k)+": the fall and the blood spreading, then dried at 20s and 2min, 3x 1080p",
			[][]*image.RGBA{row}, nil, names, 3); err != nil {
			return err
		}
	}
	return nil
}
