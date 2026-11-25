package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The bosses' lairs on screen (see sim/lair.go; the art is in lairart.go):
//
//   - the scene the world holds as entities: nest, bones, ruin, broken
//     trees, the whale's bones;
//   - what only the eye needs, worked out here from where the lair is: the
//     fronds over the sleeping snake with its eyes glinting between them,
//     the roots that rise round its arena, the dark water of the shark's
//     deep with its wreck, the whirlpool, the shark as a shadow;
//   - and the signs on the way in: birds bursting out of the trees, fish
//     scattering and leaping, rings on the water, a fin now and then.
//
// The moment the fight starts is in bossintro.go.

// lairFX is the client's own state of the lairs' scenes.
type lairFX struct {
	roots  float64 // how far the roots round Titanoboa's arena have risen, 0 to 1
	whirl  float64 // how strong the whirlpool round the shark turns, 0 to 1
	hide   float64 // how much the fronds still hide the snake, 1 to 0
	armed  [2]bool // the bird bursts not yet set off on this way in
	birds  []lairBird
	leaves []lairLeaf
	fish   []lairFish
	fishAt [2]float32 // the lair the fish were set round
	extras []lairExtra
	intro  bossIntro // see bossintro.go
	awake  map[sim.CreatureKind]bool
}

type lairBird struct {
	x, y, vx, vy float64 // tiles, tiles a frame
	alt, climb   float64 // art pixels up, and a frame
	life, flap   int
}

type lairLeaf struct {
	x, y, vx, vy float64 // art pixels from the nest
	spin         float64
	life         int
}

type lairFish struct {
	x, y, vx, vy float64
	hx, hy       float64 // where it keeps to
	jump         int     // frames left of a leap
	seed         float64
}

// lairExtra is something of a lair's scene that stands among the sprites:
// drawn in the sorted pass at its place.
type lairExtra struct {
	s      sprite
	sx, sy float64
	scaleY float64
	alpha  float64
}

// lairOf is the lair of boss k, if it is near.
func (g *Game) lairOf(k sim.CreatureKind) (protocol.LairView, bool) {
	for _, l := range g.curr.Lairs {
		if l.Kind == k {
			return l, true
		}
	}
	return protocol.LairView{}, false
}

// bossOf is boss k, if it is in view.
func (g *Game) bossOf(k sim.CreatureKind) (protocol.CreatureView, bool) {
	for _, c := range g.curr.Creatures {
		if c.Kind == k {
			return c, true
		}
	}
	return protocol.CreatureView{}, false
}

func lairEase(v, to, step float64) float64 {
	if v < to {
		return min(to, v+step)
	}
	return max(to, v-step)
}

// lairUpdate moves the lairs' scenes on a frame.
func (g *Game) lairUpdate() {
	f := &g.lair
	me, _ := g.me()
	px, py := float64(me.X), float64(me.Y)

	t, tok := g.lairOf(sim.CreatureTitanoboa)
	if !tok {
		f.armed = [2]bool{true, true} // far off: the next way in startles the birds again
	}
	f.roots = lairEase(f.roots, b2f(tok && t.Closed), 1.0/40)
	f.hide = lairEase(f.hide, b2f(!tok || !t.Awake), 1.0/30)
	if tok && !t.Defeated && !t.Awake {
		d := math.Hypot(float64(t.X)-px, float64(t.Y)-py)
		for i, at := range [2]float64{24, 14} {
			switch {
			case d > at+6:
				f.armed[i] = true
			case d < at && f.armed[i]:
				f.armed[i] = false
				g.birdBurst(px+(float64(t.X)-px)*0.45, py+(float64(t.Y)-py)*0.45, float64(t.X), float64(t.Y), 7+i*4)
			}
		}
	}
	for i := 0; i < len(f.birds); i++ {
		b := &f.birds[i]
		b.x, b.y, b.alt = b.x+b.vx, b.y+b.vy, b.alt+b.climb
		if b.life--; b.life <= 0 {
			f.birds[i] = f.birds[len(f.birds)-1]
			f.birds = f.birds[:len(f.birds)-1]
			i--
		}
	}
	for i := 0; i < len(f.leaves); i++ {
		l := &f.leaves[i]
		l.x, l.y, l.vy, l.vx = l.x+l.vx, l.y+l.vy, l.vy+0.07, l.vx*0.98
		if l.life--; l.life <= 0 {
			f.leaves[i] = f.leaves[len(f.leaves)-1]
			f.leaves = f.leaves[:len(f.leaves)-1]
			i--
		}
	}

	m, mok := g.lairOf(sim.CreatureMegalodon)
	f.whirl = lairEase(f.whirl, b2f(mok && m.Closed), 1.0/60)
	if mok && !m.Defeated {
		g.lairFishStep(m)
	} else {
		f.fish = f.fish[:0]
	}
	g.introUpdate() // see bossintro.go
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// birdBurst sends n birds up out of the trees round (x, y), away from the
// lair at (lx, ly).
func (g *Game) birdBurst(x, y, lx, ly float64, n int) {
	f := &g.lair
	var trees [][2]float64
	for _, e := range g.curr.Entities {
		if e.Kind == sim.EntityTree && math.Hypot(float64(e.TX)-x, float64(e.TY)-y) < 9 {
			trees = append(trees, [2]float64{float64(e.TX) + 0.5, float64(e.TY) - 1})
		}
	}
	away := math.Atan2(y-ly, x-lx)
	for i := range n {
		bx, by := x+(rand.Float64()-0.5)*8, y+(rand.Float64()-0.5)*6
		if len(trees) > 0 {
			t := trees[rand.IntN(len(trees))]
			bx, by = t[0]+(rand.Float64()-0.5), t[1]
		}
		a := away + (rand.Float64()-0.5)*1.8
		sp := 0.09 + rand.Float64()*0.06
		f.birds = append(f.birds, lairBird{x: bx, y: by, vx: math.Cos(a) * sp, vy: math.Sin(a) * sp * 0.7,
			alt: 14, climb: 0.35 + rand.Float64()*0.3, life: 150 + rand.IntN(80) - i*3, flap: rand.IntN(20)})
	}
	if snd != nil {
		snd.playAt("rustle", x, y, 1, 0.15)
		snd.playAt("bird", x, y, 1, 0.3)
		snd.playAt("screech", x, y, 0.35, 0.3)
	}
}

// lairFishStep moves the fish that keep to the shark's water: slowly
// about their own places, and off like arrows when the shark comes by.
func (g *Game) lairFishStep(m protocol.LairView) {
	f := &g.lair
	if len(f.fish) == 0 || f.fishAt != [2]float32{m.X, m.Y} {
		f.fish, f.fishAt = f.fish[:0], [2]float32{m.X, m.Y}
		for i := range 26 {
			a, r := hashf(i, 0, 77)*2*math.Pi, 3+hashf(i, 1, 77)*9
			x, y := float64(m.X)+math.Cos(a)*r, float64(m.Y)+math.Sin(a)*r
			f.fish = append(f.fish, lairFish{x: x, y: y, hx: x, hy: y, seed: hashf(i, 2, 77) * 100})
		}
	}
	shark, sok := g.bossOf(sim.CreatureMegalodon)
	me, _ := g.me()
	t := float64(g.frame)
	for i := range f.fish {
		p := &f.fish[i]
		ax := (p.hx-p.x)*0.0012 + math.Cos(t*0.021+p.seed)*0.0016
		ay := (p.hy-p.y)*0.0012 + math.Sin(t*0.017+p.seed*1.7)*0.0016
		flee := func(x, y, within, push float64) {
			if d := math.Hypot(p.x-x, p.y-y); d < within && d > 1e-3 {
				k := push * (1 - d/within)
				ax, ay = ax+(p.x-x)/d*k, ay+(p.y-y)/d*k
			}
		}
		if sok {
			flee(float64(shark.X), float64(shark.Y), 4.5, 0.02)
		}
		flee(float64(me.X), float64(me.Y), 2.2, 0.01)
		p.vx, p.vy = (p.vx+ax)*0.96, (p.vy+ay)*0.96
		p.x, p.y = p.x+p.vx, p.y+p.vy
		if p.jump > 0 {
			p.jump--
		} else if rand.IntN(60*40) == 0 {
			p.jump = 34
		}
	}
}

// --- in the water: under everything ------------------------------------------

// drawLairWater draws what lies in the sea round the shark's lair: the
// dark of its deep, the wreck's hull on the bottom, the fish, the rings on
// the water and the whirlpool.
func (g *Game) drawLairWater(dst drawTarget, cam camera, alpha float64) {
	m, ok := g.lairOf(sim.CreatureMegalodon)
	if !ok || g.caveIn() > 0.5 {
		return
	}
	f := &g.lair
	lx, ly := float64(m.X), float64(m.Y)
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	tx0, ty0 := int(math.Floor(cam.ox/TileW)), int(math.Floor(cam.oy/TileH))
	tx1, ty1 := int(math.Floor((cam.ox+w)/TileW)), int(math.Floor((cam.oy+h)/TileH))

	// The deep: the water darkens towards the lair.
	const deep = 23.0
	shade := func(x, y float64) float32 {
		return float32(0.66 * smooth(clamp01(1.25*(1-math.Hypot(x-lx, (y-ly)*1.1)/deep))))
	}
	var vs []ebiten.Vertex
	var is []uint16
	for ty := max(ty0, int(ly-deep)-1); ty <= min(ty1, int(ly+deep)+1); ty++ {
		for tx := max(tx0, int(lx-deep)-1); tx <= min(tx1, int(lx+deep)+1); tx++ {
			if g.tileAt(tx, ty) != sim.TileWater {
				continue
			}
			if len(vs) > 60000 {
				break
			}
			n := uint16(len(vs))
			for _, c := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				x, y := float64(tx+c[0]), float64(ty+c[1])
				a := shade(x, y)
				for _, o := range [4][2]int{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}} { // clear by the shore, so no tile's edge shows
					if g.tileAt(tx+c[0]+o[0], ty+c[1]+o[1]) != sim.TileWater {
						a = 0
					}
				}
				sx, sy := cam.toScreen(x, y)
				vs = append(vs, ebiten.Vertex{DstX: float32(sx), DstY: float32(sy), SrcX: 1, SrcY: 1,
					ColorR: 0.02 * a, ColorG: 0.05 * a, ColorB: 0.13 * a, ColorA: a})
			}
			is = append(is, n, n+1, n+2, n+1, n+3, n+2)
		}
	}
	if len(vs) > 0 {
		dst.DrawTriangles(vs, is, whitePixel(), nil)
	}

	a := lairArtOf()
	if wx, wy, ok := g.wreckAt(m); ok { // the hull, seen through the water
		sx, sy := cam.toScreen(wx+1.2, wy+0.6)
		drawAt(dst, a.hull, sx, sy, 1, 1, 0, color.RGBA{150, 150, 150, 150})
	}

	// Fish: dark darts under the surface; one leaps now and then.
	for _, p := range f.fish {
		sx, sy := cam.toScreenFine(p.x, p.y)
		if sx < -8 || sy < -8 || sx > w+8 || sy > h+8 || g.tileAt(int(math.Floor(p.x)), int(math.Floor(p.y))) != sim.TileWater {
			continue
		}
		dx, dy := p.vx, p.vy
		if d := math.Hypot(dx, dy); d > 1e-5 {
			dx, dy = dx/d, dy/d
		} else {
			dx, dy = 1, 0
		}
		if p.jump > 0 {
			u := 1 - float64(p.jump)/34
			up := math.Sin(u*math.Pi) * 9
			strokeLine(dst, float32(sx-dx*2.5), float32(sy-up), float32(sx+dx*2.5), float32(sy-up+(u-0.5)*5), 1.6, color.RGBA{190, 215, 225, 255}, true)
			strokeCircle(dst, float32(sx), float32(sy), float32(2+u*6), 0.7, color.RGBA{120, 130, 135, uint8(130 * (1 - u))}, true)
			continue
		}
		strokeLine(dst, float32(sx-dx*3), float32(sy-dy*2.2), float32(sx+dx*3), float32(sy+dy*2.2), 1.7, color.RGBA{3, 12, 22, 120}, true)
		strokeLine(dst, float32(sx-dx*3), float32(sy-dy*2.2), float32(sx-dx*5-dy*1.4), float32(sy-dy*3.6+dx*1.4), 1, color.RGBA{3, 12, 22, 110}, true)
	}

	// The whirlpool: streaks of foam running round the ring.
	if f.whirl > 0 {
		const n = 150
		for ring := range 5 {
			r := float64(m.Radius) - 2.2 + float64(ring)*1.05
			turn := float64(g.frame) * (0.006 + 0.0015*float64(ring))
			for i := range n {
				th := turn + float64(i)*2*math.Pi/n + float64(ring)*0.7
				if hashf(i, ring, 55) < 0.3 {
					continue
				}
				x0, y0 := lx+math.Cos(th)*r, ly+math.Sin(th)*r
				if g.tileAt(int(math.Floor(x0)), int(math.Floor(y0))) != sim.TileWater {
					continue
				}
				x1, y1 := lx+math.Cos(th+0.06)*(r-0.12), ly+math.Sin(th+0.06)*(r-0.12)
				sx0, sy0 := cam.toScreenFine(x0, y0)
				sx1, sy1 := cam.toScreenFine(x1, y1)
				if sx0 < -30 || sy0 < -30 || sx0 > w+30 || sy0 > h+30 {
					continue
				}
				v := uint8(f.whirl * (90 + 130*hashf(i, ring, 56)))
				strokeLine(dst, float32(sx0), float32(sy0), float32(sx1), float32(sy1), 1.2+float32(hashf(i, ring, 57))*1.4, color.RGBA{v, v, v, v}, true)
			}
		}
	}
}

// wreckAt is where the wreck's mast stands by lair m: the nearest of a few
// places that is open water.
func (g *Game) wreckAt(m protocol.LairView) (x, y float64, ok bool) {
	for _, o := range [][2]float64{{-1.5, -2.5}, {1.5, -2.5}, {-2.5, 3}, {2.5, 3.5}, {0, -6}} {
		x, y = math.Floor(float64(m.X)+o[0])+0.5, math.Floor(float64(m.Y)+o[1])+0.5
		if g.tileAt(int(x), int(y)) == sim.TileWater && g.tileAt(int(x)+2, int(y)) == sim.TileWater {
			return x, y, true
		}
	}
	return 0, 0, false
}

// sharkAsleep reports whether the Megalodon c is to be drawn as a sleeping
// lair's shark: a shadow under the water.
func (g *Game) sharkAsleep(c protocol.CreatureView) bool {
	if c.Kind != sim.CreatureMegalodon || c.State != sim.CreatureLurk {
		return false
	}
	m, ok := g.lairOf(sim.CreatureMegalodon)
	return ok && !m.Awake
}

// drawSleepingShark draws the Megalodon with nobody to fight: a great
// shadow going round under the water, rings where it passed, and now and
// then its fin cutting the surface.
func (g *Game) drawSleepingShark(dst drawTarget, cam camera, c protocol.CreatureView, alpha float64) {
	x, y, _ := g.lerpCreature(c, alpha)
	sx, sy := cam.toScreenFine(x, y)
	a := lairArtOf()
	side := 0
	if c.FaceLeft {
		side = 1
	}
	sway := math.Sin(float64(g.frame)*0.05) * 0.03
	drawAt(dst, a.shadow[side], sx, sy, 1, 1, sway, color.RGBA{150, 150, 150, 150})
	for k := range 3 { // rings on the water
		ph := math.Mod(float64(g.frame)/150+float64(k)/3, 1)
		v := uint8(70 * (1 - ph))
		strokeCircleOval(dst, sx, sy, 10+ph*44, 0.42, color.RGBA{v, v, v, v})
	}
	// The fin: up for a few seconds, then gone again.
	if up := math.Sin(float64(g.frame)*2*math.Pi/600 + float64(c.ID)); up > 0.45 {
		s, _ := g.creatureSprite(c, true)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-s.ax, sy-s.ay)
		op.ColorScale.ScaleAlpha(float32(clamp01((up - 0.45) * 6)))
		drawSprite(dst, s, op)
	}
}

// strokeCircleOval draws a ring lying flat on the ground: r wide, squashed
// to r*flat high.
func strokeCircleOval(dst drawTarget, cx, cy, r, flat float64, clr color.Color) {
	const n = 28
	for i := range n {
		a0, a1 := 2*math.Pi*float64(i)/n, 2*math.Pi*float64(i+1)/n
		strokeLine(dst, float32(cx+math.Cos(a0)*r), float32(cy+math.Sin(a0)*r*flat), float32(cx+math.Cos(a1)*r), float32(cy+math.Sin(a1)*r*flat), 0.8, clr, true)
	}
}

// --- on the ground, and among the sprites --------------------------------------

// lairSprite is the sprite of lair prop e.
func (g *Game) lairSprite(e protocol.EntityView) sprite {
	a := lairArtOf()
	h := int(sim.HashTile(int(e.TX), int(e.TY)) / 16)
	switch e.Var {
	case sim.LairNest:
		return a.nest
	case sim.LairCover:
		return a.cover
	case sim.LairBones:
		return a.bones[h%len(a.bones)]
	case sim.LairSkin:
		return a.skin
	case sim.LairTrail:
		dir := float64(h%4) * math.Pi / 4
		if l, ok := g.lairOf(sim.CreatureTitanoboa); ok {
			dir = math.Atan2((float64(l.Y)-float64(e.TY)-0.5)*0.75, float64(l.X)-float64(e.TX)-0.5)
		}
		return a.trail[((int(math.Round(dir/(math.Pi/4)))%4)+4)%4]
	case sim.LairRuin:
		return a.ruin[h%len(a.ruin)]
	case sim.LairStump:
		return a.stump[h%len(a.stump)]
	case sim.LairWhale:
		return a.whale
	}
	return a.bones[0]
}

// lairShadow is shadowWidth for a lair's props.
func lairShadow(e protocol.EntityView) (float64, bool) {
	if e.Kind != sim.EntityLairProp {
		return 0, false
	}
	switch e.Var {
	case sim.LairRuin:
		return 26, true
	case sim.LairStump:
		return 20, true
	}
	return 0, true
}

// drawLairGround draws the lairs' props that lie flat, under everything
// that stands.
func (g *Game) drawLairGround(dst drawTarget, cam camera) {
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityLairProp || !lairFlat(e.Var) || g.hiddenInCave(float64(e.TX)+0.5, float64(e.TY)+0.5) {
			continue
		}
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		drawAt(dst, g.lairSprite(e), sx, sy, 1, 1, 0, nil)
	}
}

// drawLairProp draws a standing lair prop in the sorted pass.
func (g *Game) drawLairProp(dst drawTarget, e protocol.EntityView, d drawable) {
	if lairFlat(e.Var) {
		return
	}
	s := g.lairSprite(e)
	if e.Var != sim.LairCover {
		drawAt(dst, s, d.sx, d.sy, 1, 1, 0, nil)
		return
	}
	// The fronds over the sleeping snake: thrown aside when it wakes.
	f := &g.lair
	hide := smooth(f.hide)
	sway := 0.012 * math.Sin(float64(g.frame)*0.03)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Scale(1, 0.3+0.7*hide)
	op.GeoM.Skew(sway, 0)
	op.GeoM.Translate(d.sx, d.sy)
	op.ColorScale.ScaleAlpha(float32(0.55 + 0.45*hide))
	drawSprite(dst, s, op)
	for _, l := range f.leaves {
		v := uint8(255 * clamp01(float64(l.life)/20))
		c := jungleRamp[2+int(l.spin*10)%3]
		fillRect(dst, float32(d.sx+l.x), float32(d.sy-30+l.y), 2, 1.4, color.RGBA{uint8(int(c.R) * int(v) / 255), uint8(int(c.G) * int(v) / 255), uint8(int(c.B) * int(v) / 255), v}, false)
	}
	// Its eyes, glinting out of the dark between the leaves.
	c, ok := g.bossOf(sim.CreatureTitanoboa)
	if !ok || hide < 0.95 || c.State != sim.CreatureIdle || math.Hypot(float64(c.X)-float64(e.TX)-0.5, float64(c.Y)-float64(e.TY)+0.5) > 1.2 {
		return
	}
	blink := (g.frame+int(e.TX)*13)%260 < 9
	if blink {
		return
	}
	side := 1.0
	if c.FaceLeft {
		side = -1
	}
	ex, ey := d.sx+side*53, d.sy-TileH-50
	glow := 0.6 + 0.4*math.Sin(float64(g.frame)*0.07)
	for _, o := range []float64{0, 7} {
		x := float32(ex + side*o)
		fillCircle(dst, x, float32(ey), 5, color.RGBA{uint8(46 * glow), uint8(36 * glow), 0, uint8(52 * glow)}, true)
		fillCircle(dst, x, float32(ey), 3, color.RGBA{uint8(80 * glow), uint8(62 * glow), 0, uint8(90 * glow)}, true)
		fillRect(dst, x-1.34, float32(ey)-1, 2.67, 2, color.RGBA{255, 218, 70, 255}, false)
		fillRect(dst, x-0.34, float32(ey)-1, 0.67, 2, color.RGBA{30, 16, 0, 255}, false)
	}
}

// leafBurst throws leaves up from the nest as the snake comes out.
func (g *Game) leafBurst() {
	f := &g.lair
	for range 70 {
		a := -math.Pi/2 + (rand.Float64()-0.5)*2.6
		sp := 1 + rand.Float64()*2.6
		f.leaves = append(f.leaves, lairLeaf{x: (rand.Float64() - 0.5) * 120, y: (rand.Float64() - 0.5) * 30,
			vx: math.Cos(a) * sp, vy: math.Sin(a) * sp, spin: rand.Float64(), life: 50 + rand.IntN(50)})
	}
}

// lairDrawables adds what stands of the lairs' scenes and is not an entity
// to the sorted pass: the roots of the arena, the wreck's mast.
func (g *Game) lairDrawables(list []drawable, cam camera, w, h float64) []drawable {
	f := &g.lair
	f.extras = f.extras[:0]
	a := lairArtOf()
	add := func(s sprite, x, y, scaleY float64) {
		sx, sy := cam.toScreen(x, y)
		if sx < -60 || sy < -10 || sx > w+60 || sy > h+110 {
			return
		}
		f.extras = append(f.extras, lairExtra{s: s, sx: sx, sy: sy, scaleY: scaleY, alpha: 1})
		list = append(list, drawable{y: y, id: uint32(len(f.extras)), extra: true, idx: len(f.extras) - 1, sx: sx, sy: sy})
	}
	if t, ok := g.lairOf(sim.CreatureTitanoboa); ok && f.roots > 0 {
		lx, ly, r := float64(t.X), float64(t.Y), float64(t.Radius)
		for ty := int(ly-r) - 1; ty <= int(ly+r)+1; ty++ {
			for tx := int(lx-r) - 1; tx <= int(lx+r)+1; tx++ {
				if t := g.tileAt(tx, ty); !sim.ArenaTile(lx, ly, r, tx, ty) || t == sim.TileMountain || t == sim.TileLava {
					continue
				}
				hsh := sim.HashTile(tx, ty)
				grow := smooth(clamp01(f.roots*1.7 - float64(hsh%97)/97*0.7)) // each in its own time
				if grow > 0 {
					add(a.vine[hsh/128%3], float64(tx)+0.5, float64(ty)+0.7, grow*(0.85+float64(hsh/7%30)/100))
				}
			}
		}
	}
	if m, ok := g.lairOf(sim.CreatureMegalodon); ok {
		if x, y, ok := g.wreckAt(m); ok {
			add(a.mast, x, y, 1)
		}
	}
	return list
}

// drawLairExtra draws one of lairDrawables' things.
func (g *Game) drawLairExtra(dst drawTarget, d drawable) {
	e := g.lair.extras[d.idx]
	if e.s.img == lairArtOf().mast.img { // the sea laps round its foot
		ph := math.Mod(float64(g.frame)/110, 1)
		v := uint8(90 * (1 - ph))
		strokeCircleOval(dst, d.sx, d.sy, 5+ph*10, 0.4, color.RGBA{v, v, v, v})
	}
	sway := 0.0
	if e.scaleY < 1 {
		sway = 0.25 * (1 - e.scaleY) * math.Sin(float64(g.frame)*0.5+d.sx)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-e.s.ax, -e.s.ay)
	op.GeoM.Scale(1, e.scaleY)
	op.GeoM.Skew(sway, 0)
	op.GeoM.Translate(d.sx, d.sy)
	drawSprite(dst, e.s, op)
}

// drawLairSky draws the birds that burst out of the trees near a lair.
func (g *Game) drawLairSky(dst drawTarget, cam camera) {
	a := lairArtOf()
	for _, b := range g.lair.birds {
		sx, sy := cam.toScreenFine(b.x, b.y)
		v := uint8(255 * clamp01(float64(b.life)/40))
		drawAt(dst, a.bird[(g.frame+b.flap)/5%2], sx, sy-b.alt, 1, 1, 0, color.RGBA{v, v, v, v})
	}
}
