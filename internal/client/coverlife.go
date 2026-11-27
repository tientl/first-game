package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The life of the ground cover: the wind running through it in gusts, the
// grass bending away from whoever walks through and springing back after,
// tall grass hiding the feet of those standing in it, and the litter and
// moss under the trees. None of it is baked: a plant is drawn sheared and
// squashed about its foot, so nothing is redrawn as it moves.

// --- the wind -----------------------------------------------------------------------

// windSkew is how far, as a shear, a plant at (x, y) tiles leans in the
// wind now, before its own stiffness: gusts travel across the land from the
// west, each plant swaying at its own phase, harder in a wind or a storm.
func (g *Game) windSkew(x, y float64, phase float64) float64 {
	t := float64(g.frame) / 60
	gust := 0.5 + 0.5*math.Sin(t*0.85-x*0.32-y*0.11)
	gust *= gust
	strength, lean := 0.75, 0.0
	if g.curr.Weather.Windy() {
		strength, lean = 2.1, 0.45
	}
	return strength * (lean*gust + (0.35+0.65*gust)*math.Sin(t*2.1+phase+x*0.7))
}

// --- bending under feet ------------------------------------------------------------

// mover is someone walking through the plants: where, how wide, and which
// way they go.
type mover struct{ x, y, r, vx float64 }

// bendState is how far one plant is bent and how fast it is moving back.
type bendState struct {
	v, vel float64
	frame  int
}

// bender keeps the plants bent by those walking through them.
type bender struct {
	movers []mover
	m      map[uint64]*bendState
	frame  int
}

// noteMovers lists the players and the creatures on the ground in view, for
// the plants to bend away from.
func (g *Game) noteMovers(alpha float64) {
	b := &g.bend
	if b.m == nil {
		b.m = map[uint64]*bendState{}
	}
	b.frame = g.frame
	b.movers = b.movers[:0]
	for _, p := range g.curr.Players {
		if p.Dead || p.Swimming || p.Boat != 0 || p.Mount != 0 {
			continue
		}
		x, y, _ := g.lerpPlayer(p, alpha)
		vx := 0.0
		if q, ok := g.prevPlayers[p.ID]; ok {
			vx = float64(p.X - q.X)
		}
		b.movers = append(b.movers, mover{x, y, 0.5, vx})
	}
	for _, c := range g.curr.Creatures {
		if c.Kind == sim.CreatureFish || c.Kind == sim.CreaturePlesio || c.State == sim.CreatureHidden || inTheSky(c) {
			continue
		}
		x, y, _ := g.lerpCreature(c, alpha)
		r := max(0.3, creatureShadow(c.Kind)/2/TileW*1.1)
		vx := 0.3
		if c.FaceLeft {
			vx = -0.3
		}
		b.movers = append(b.movers, mover{x, y, r, vx})
	}
	if len(b.m) > 4096 || g.frame%120 == 0 { // forget plants long at rest
		for k, s := range b.m {
			if g.frame-s.frame > 240 {
				delete(b.m, k)
			}
		}
	}
}

// push is how hard the movers push a plant at (x, y): below 0 to the left,
// above to the right, 1 flat down.
func (b *bender) push(x, y float64) float64 {
	p := 0.0
	for _, m := range b.movers {
		dx, dy := x-m.x, (y-m.y)*1.6
		if math.Abs(dx) > m.r || math.Abs(dy) > m.r {
			continue
		}
		d := math.Hypot(dx, dy)
		if d >= m.r {
			continue
		}
		s := 1 - d/m.r
		side := math.Copysign(1, dx)
		if math.Abs(dx) < 0.08*m.r && m.vx != 0 {
			side = math.Copysign(1, m.vx) // straight ahead: pushed the way they go
		}
		p += side * math.Sqrt(s)
	}
	return max(-1, min(1, p))
}

// at is how far the plant known by key, at (x, y), is bent now: pressed
// away from whoever stands in it, springing back with a wobble once they
// have gone.
func (b *bender) at(key uint64, x, y float64) float64 {
	target := b.push(x, y)
	s, ok := b.m[key]
	if !ok {
		if target == 0 {
			return 0
		}
		s = &bendState{frame: b.frame - 1}
		b.m[key] = s
	}
	for n := min(6, b.frame-s.frame); n > 0; n-- {
		k, damp := 0.09, 0.2
		if target != 0 {
			k = 0.3 // pressed down fast
		}
		s.vel += (target-s.v)*k - s.vel*damp
		s.v += s.vel
	}
	if target != 0 || math.Abs(s.v) > 0.002 || math.Abs(s.vel) > 0.002 {
		s.frame = b.frame
	} else {
		s.v, s.vel, s.frame = 0, 0, b.frame
	}
	return s.v
}

// coverKey names the plant on tile (tx, ty), spot i (-1: the entity there).
func coverKey(tx, ty, i int) uint64 {
	return uint64(uint32(int32(tx)))<<32 | uint64(uint32(int32(ty)))<<4 | uint64(i+1)&15
}

// --- drawing one plant --------------------------------------------------------------

// coverLook is how one plant is drawn: mirrored or not, its size, its tint.
type coverLook struct {
	flip    bool
	scale   float64
	r, g, b float32
}

// coverLookAt is the look of a plant at (x, y) tiles whose hash is h: a
// size within 13% either way, mirrored half the time, a little lighter or
// darker, warmer or cooler, and greener or yellower with the meadow's own
// patches (as grassGround colours the ground under it).
func coverLookAt(x, y float64, h uint32) coverLook {
	f := func(n uint) float64 { return float64(h>>(n*6)&63) / 63 }
	v := 0.93 + 0.12*f(1)
	w := (f(2) - 0.5) * 0.08 // warm or cool
	l := coverLook{flip: h&1 == 0, scale: 0.87 + 0.26*f(3)}
	r, g, b := v*(1+w), v, v*(1-w)
	if hue := fbm2(x*0.45+7, y*0.55, 38, 2); hue > 0.55 {
		k := math.Min(0.22, (hue-0.55)*1.2)
		r, b = r*(1+0.12*k), b*(1-0.25*k)
	} else if hue < 0.42 {
		k := math.Min(0.2, (0.42-hue)*1.2)
		r, b = r*(1-0.15*k), b*(1+0.12*k)
	}
	l.r, l.g, l.b = float32(r), float32(g), float32(b)
	return l
}

// drawCover draws plant sprite s with its foot at (sx, sy) on the screen,
// sheared by skew (wind and feet) and squashed by press (0 to 1, how flat
// it is trodden), in look l.
func drawCover(dst drawTarget, s sprite, sx, sy, skew, press float64, l coverLook) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	fx := l.scale
	if l.flip {
		fx = -fx
	}
	op.GeoM.Scale(fx, l.scale*(1-0.35*press))
	op.GeoM.Skew(skew, 0)
	op.GeoM.Translate(sx, sy)
	op.ColorScale.Scale(l.r, l.g, l.b, 1)
	drawSprite(dst, s, op)
}

// --- tall grass round the feet ------------------------------------------------------

// fringeSets are the blades of tall grass drawn in front of someone's feet
// as they stand in it, by season.
var fringeSets [sim.NumSeasons]lazy[[]sprite]

func init() {
	for s := range sim.NumSeasons {
		fringeSets[s].gen = func() []sprite {
			var out []sprite
			for v := range 3 {
				out = append(out, genFringe(v, s))
			}
			return out
		}
	}
}

// genFringe is a band of tall grass blades for in front of the feet: wide,
// as tall as a shin, fading out at its ends.
func genFringe(v int, s sim.Season) sprite {
	const k = hdK3
	p := coverPals[s%sim.NumSeasons]
	rng := rand.New(rand.NewPCG(uint64(v)+77, 91))
	c := hdCanvas(30, 16)
	bx, by := 15*k, 14*k
	for i := range 34 {
		f := (float64(i) + rng.Float64()) / 34
		x := bx + (f-0.5)*24*k
		h := (5 + rng.Float64()*6) * k * (1 - 0.5*math.Pow(math.Abs(f-0.5)*2, 2))
		lean := (rng.Float64() - 0.5) * 5 * k
		rp := p.tall
		if i%3 == 0 && !p.snow {
			rp = p.grass
		}
		coverBlade(c, x, by, h, lean, 0.15, 1.6+rng.Float64(), rp, 0.1+0.15*rng.Float64(), 0.7+0.25*rng.Float64(), p.snow, 61)
	}
	for y := range c.h { // fade out at either end, and into the ground
		for x := range c.w {
			i := y*c.w + x
			if c.px[i].A == 0 {
				continue
			}
			e := math.Abs(float64(x)+0.5-bx) / (12 * k)
			a := 1 - smooth(clamp01((e-0.55)/0.45))
			if dy := by - float64(y); dy < 2*k {
				a *= 0.5 + 0.5*clamp01(dy/(2*k))
			}
			c.px[i].A = uint8(float64(c.px[i].A) * a)
		}
	}
	return mkUp(c, bx, by)
}

// tallGrassAt is how deep in tall grass someone standing at (x, y) is: 0
// out of it, 1 in the middle of a tuft.
func (g *Game) tallGrassAt(x, y float64) float64 {
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	best := 0.0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			i, ok := g.entAt[[2]int{tx + dx, ty + dy}]
			if !ok {
				continue
			}
			e := g.curr.Entities[i]
			if e.Kind != sim.EntityGrassTuft || !e.Ready {
				continue
			}
			ex, ey := float64(e.TX)+0.5-x, (float64(e.TY)+0.6-y)*1.5
			if d := math.Hypot(ex, ey); d < 0.75 {
				best = max(best, 1-smooth(clamp01((d-0.3)/0.45)))
			}
		}
	}
	return best
}

// drawFeetInGrass draws tall grass over the feet of someone standing in it
// at (wx, wy) tiles, drawn at (sx, sy), w art pixels across.
func (g *Game) drawFeetInGrass(dst drawTarget, wx, wy, sx, sy, w, vx float64) {
	if !envHD {
		return
	}
	deep := g.tallGrassAt(wx, wy)
	if deep <= 0.02 {
		return
	}
	set := fringeSets[g.curr.Season%sim.NumSeasons].get()
	s := set[int(math.Floor(wx*1.7+wy*0.9))&0x7fffffff%len(set)]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	k := w / 22
	op.GeoM.Scale(k, k*(0.55+0.45*deep))
	op.GeoM.Skew(-vx*1.5+g.windSkew(wx, wy, 0)*0.05, 0)
	op.GeoM.Translate(sx, sy+1.5)
	op.ColorScale.ScaleAlpha(float32(min(1, deep*1.3)))
	drawSprite(dst, s, op)
}

// --- shade and litter under the trees -------------------------------------------------

// shadeGrid is, for the tiles in view, how much the trees about shade each.
type shadeGrid struct {
	x0, y0, w, h int
	v            []float32
}

func (s *shadeGrid) at(tx, ty int) float64 {
	x, y := tx-s.x0, ty-s.y0
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	return float64(s.v[y*s.w+x])
}

// treeCrown is how far round its trunk a tree on the ground shades, in
// tiles, and the kind of litter under it; 0 for what is not a tree.
func (g *Game) treeCrown(e protocol.EntityView) (r float64, litter int) {
	switch e.Kind {
	case sim.EntityTree:
		if !e.Ready {
			return 0, 0
		}
		r = 1.6
		if e.Big {
			r = 2.3
		}
		switch sim.TreeLookAt(int(e.TX), int(e.TY), g.tileAt(int(e.TX), int(e.TY))) {
		case sim.LookPine, sim.LookSnowPine, sim.LookRedwood, sim.LookAraucaria:
			return r, litterNeedles
		case sim.LookTreeFern, sim.LookCycad:
			return r * 0.75, litterMoss
		case sim.LookSwamp:
			return 0, 0
		}
		return r, litterLeaves
	case sim.EntityGiantTree:
		return 3.5, litterLeaves
	case sim.EntityFigTree, sim.EntityGinkgoTree:
		return 1.7, litterLeaves
	case sim.EntityPalm:
		return 1.2, litterMoss
	}
	return 0, 0
}

// The kinds of litter under a tree.
const (
	litterLeaves = iota + 1
	litterNeedles
	litterMoss
)

// noteShade works out which tiles in view the trees shade.
func (g *Game) noteShade(tx0, ty0, tx1, ty1 int) {
	s := &g.shade
	s.x0, s.y0, s.w, s.h = tx0, ty0, tx1-tx0+1, ty1-ty0+1
	if cap(s.v) < s.w*s.h {
		s.v = make([]float32, s.w*s.h)
	}
	s.v = s.v[:s.w*s.h]
	clear(s.v)
	for _, e := range g.curr.Entities {
		r, _ := g.treeCrown(e)
		if r == 0 {
			continue
		}
		cx, cy := float64(e.TX)+0.5, float64(e.TY)+0.2 // the crown hangs over the trunk, a little behind
		for y := int(cy - r); y <= int(cy+r); y++ {
			for x := int(cx - r); x <= int(cx+r); x++ {
				i, j := x-s.x0, y-s.y0
				if i < 0 || j < 0 || i >= s.w || j >= s.h {
					continue
				}
				d := math.Hypot(float64(x)+0.5-cx, (float64(y)+0.5-cy)*1.2) / r
				if d < 1 {
					s.v[j*s.w+i] = min(1.5, s.v[j*s.w+i]+float32(1-d*d))
				}
			}
		}
	}
}

// litterSets are the decals of litter under the trees: fallen leaves,
// needles and moss, by season, three of each.
var litterSets [sim.NumSeasons]lazy[[3][3]sprite]

func init() {
	for s := range sim.NumSeasons {
		litterSets[s].gen = func() (out [3][3]sprite) {
			for kind := range 3 {
				for v := range 3 {
					out[kind][v] = genLitter(kind+1, v, s)
				}
			}
			return out
		}
	}
}

// genLitter is the litter under a tree: a soft-edged patch of darker,
// bare-ish earth strewn with what the tree drops, fading out at its rim,
// about a tile and a half round (it is drawn scaled to the crown).
func genLitter(kind, v int, s sim.Season) sprite {
	const k = hdK3
	const rw, rh = 36.0, 20.0 // half size, art pixels
	c := hdCanvas(int(2*rw), int(2*rh))
	cx, cy := rw*k, rh*k
	rng := rand.New(rand.NewPCG(uint64(kind)*31+uint64(v), uint64(s)+3))
	earth := mkRamp(0x2a2414, 0x3a3019, 0x4a3d20, 0x5a4a28)
	var bits []color.RGBA
	switch kind {
	case litterLeaves:
		bits = []color.RGBA{hex(0x6b5430), hex(0x86683a), hex(0x9a7a42), hex(0x5d6b2e), hex(0x4f4424)}
		if s == sim.Autumn {
			bits = []color.RGBA{hex(0xb5662a), hex(0xd08a34), hex(0xe2ae48), hex(0x9a4a22), hex(0x86683a)}
		}
	case litterNeedles:
		bits = []color.RGBA{hex(0x7a4a26), hex(0x95602f), hex(0xa8743a), hex(0x5e3a20)}
	default:
		bits = []color.RGBA{hex(0x4e7a2e), hex(0x628e36), hex(0x76a242), hex(0x3d6328)}
	}
	moss := mkRamp(0x2f4a1f, 0x3d5f26, 0x4f752e, 0x638a38)
	for y := range c.h {
		for x := range c.w {
			px, py := (float64(x)+0.5-cx)/(rw*k), (float64(y)+0.5-cy)/(rh*k)
			wob := 0.12*(vnoise(float64(x)*0.03, float64(y)*0.05, uint32(v)*7+300)-0.5) + 0.06*(vnoise(float64(x)*0.11, float64(y)*0.15, 301)-0.5)
			d := math.Sqrt(px*px+py*py) + wob
			if d >= 1 {
				continue
			}
			a := 0.62 * math.Pow(1-d, 0.9) * smooth(clamp01((1-d)/0.45))
			col := rampAt(earth, 0.4+(vnoise(float64(x)*0.08, float64(y)*0.1, 302)-0.5)*0.8, x, y, 303)
			if kind == litterMoss || vnoise(float64(x)*0.05+9, float64(y)*0.07, 304) > 0.62 {
				col = rampAt(moss, 0.5+(vnoise(float64(x)*0.2, float64(y)*0.25, 305)-0.5)*0.9, x, y, 306)
				a = math.Min(0.85, a*1.25)
			}
			c.px[y*c.w+x] = withAlpha(col, uint8(a*255))
		}
	}
	// What lies on it: leaves, needles, bits of frond, thicker towards the middle.
	n := 160
	if kind == litterNeedles {
		n = 260
	}
	for range n {
		r := math.Sqrt(rng.Float64()) * 0.85
		a := rng.Float64() * 2 * math.Pi
		x, y := cx+math.Cos(a)*r*rw*k, cy+math.Sin(a)*r*rh*k
		col := bits[rng.IntN(len(bits))]
		alpha := uint8(255 * (1 - r*0.7))
		switch kind {
		case litterNeedles: // a short dark-and-light stroke
			ang := rng.Float64() * math.Pi
			for q := 0.0; q < 4; q++ {
				c.blend(int(x+math.Cos(ang)*q), int(y+math.Sin(ang)*q*0.6), withAlpha(col, alpha))
			}
		default: // a leaf: a small lit oval with a dark edge below
			ang := rng.Float64() * math.Pi
			ca, sa := math.Cos(ang), math.Sin(ang)
			for j := -3; j <= 3; j++ {
				for i := -3; i <= 3; i++ {
					u := (float64(i)*ca + float64(j)*sa) / 2.6
					w := (-float64(i)*sa + float64(j)*ca) / 1.4
					if u*u+w*w > 1 {
						continue
					}
					cc := col
					if w > 0.3 {
						cc = mix(col, hex(0x1d1a10), 0.35)
					} else if w < -0.4 {
						cc = mix(col, hex(0xf0e0b0), 0.15)
					}
					c.blend(int(x)+i, int(y)+j, withAlpha(cc, alpha))
				}
			}
		}
	}
	return mkUp(c, cx, cy)
}

// drawLitter lays the litter under the trees in view, before the plants.
func (g *Game) drawLitter(dst drawTarget, cam camera) {
	if !envHD || g.curr.Season == sim.Winter {
		return // under the snow
	}
	set := litterSets[g.curr.Season%sim.NumSeasons].get()
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	for _, e := range g.curr.Entities {
		r, kind := g.treeCrown(e)
		if r == 0 {
			continue
		}
		wx, wy := float64(e.TX)+0.5, float64(e.TY)+0.45
		sx, sy := cam.toScreen(wx, wy)
		if sx < -r*TileW*2 || sy < -r*TileH*2 || sx > w+r*TileW*2 || sy > h+r*TileH*2 {
			continue
		}
		t := g.tileAt(int(e.TX), int(e.TY))
		if t != sim.TileGrass && t != sim.TileForest && t != sim.TilePath && t != sim.TileMarsh && t != sim.TileRocky {
			continue
		}
		hh := sim.HashTile(int(e.TX), int(e.TY))
		s := set[kind-1][hh%3]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.ax, -s.ay)
		k := r * TileW / 36 * (0.9 + 0.2*float64(hh>>8&15)/15)
		fx := k
		if hh>>4&1 == 0 {
			fx = -k
		}
		op.GeoM.Scale(fx, k)
		op.GeoM.Translate(sx, sy)
		a := float32(0.8)
		if t == sim.TileForest {
			a = 0.6 // the woods' floor is littered already
		}
		op.ColorScale.ScaleAlpha(a)
		drawSprite(dst, s, op)
	}
}

// --- the plants that are things in the world ----------------------------------------

// drawCoverEntity draws tall grass, a berry bush or wild flowers in one of
// their drawings, swaying and bending like the undergrowth, and reports
// whether e was one of them.
func (g *Game) drawCoverEntity(dst drawTarget, e protocol.EntityView, d drawable) bool {
	if !envHD {
		return false
	}
	tx, ty := int(e.TX), int(e.TY)
	h := sim.HashTile(tx, ty)
	season := g.curr.Season % sim.NumSeasons
	var s sprite
	var sway, give float64
	switch e.Kind {
	case sim.EntityGrassTuft:
		if e.Ready {
			s, sway, give = coverSprite(cvTall, int(h>>3), season), 0.11, 0.5
		} else {
			s, sway, give = coverSprite(cvStubble, int(h>>3), season), 0.02, 0.1
		}
	case sim.EntityBerryBush:
		k := cvBushBare
		if e.Ready {
			k = cvBush
		}
		s, sway, give = coverSprite(k, int(h>>3), season), 0.025, 0.12
	case sim.EntityFlower:
		if !e.Ready {
			return false
		}
		fl := g.art.tribal.flowers
		s, sway, give = fl[int(e.Var)%len(fl)], 0.08, 0.4
	default:
		return false
	}
	x, y := float64(tx)+0.5, float64(ty)+0.5
	bend := g.bend.at(coverKey(tx, ty, -1), x, y)
	skew := sway*g.windSkew(x, y, float64(h%628)/100) - give*bend
	look := coverLookAt(x, y, h*2654435761)
	if e.Kind == sim.EntityBerryBush {
		look.scale = 0.94 + (look.scale-0.87)*0.45 // a bush keeps about its size: it is a thing to find
	}
	drawCover(dst, s, d.sx, d.sy, skew, math.Abs(bend)*give*1.2, look)
	return true
}

// coverShadow is the width of the round shadow under e, for the plants
// whose drawings carry their own soft shadow: none.
func coverShadow(e protocol.EntityView) (float64, bool) {
	if envHD && (e.Kind == sim.EntityGrassTuft || e.Kind == sim.EntityBerryBush) {
		return 0, true
	}
	return 0, false
}
