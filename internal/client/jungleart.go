package client

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The woods: banana trees, the giants of the old forest, beehives and the
// bees about them, and monkeys. See sim/jungle.go.

type woodsArt struct {
	banana, bananaBare []sprite
	giants             []sprite
	giantStump         sprite
	hive, hiveEmpty    sprite
	monkey             [2][monkeyPoses]sprite
}

// Monkey poses.
const (
	monkeySit = iota
	monkeyWalkA
	monkeyWalkB
	monkeyLeap
	monkeyEat
	monkeyPoses
)

var woods *woodsArt

// woodsArtOf draws the woods' art the first time it is wanted.
func woodsArtOf() *woodsArt {
	if woods != nil {
		return woods
	}
	a := &woodsArt{}
	for v := uint64(1); v <= 3; v++ {
		a.banana = append(a.banana, genBananaTree(fruitScale, v, true))
		a.bananaBare = append(a.bananaBare, genBananaTree(fruitScale, v, false))
	}
	for v, sh := range []treeShape{shapeRound, shapeOak} {
		a.giants = append(a.giants, genBroadleaf(sh, v, true, giantScale, uint64(v)*17+3))
	}
	a.giantStump = genStump(giantScale * 1.15)
	a.hive, a.hiveEmpty = genHive(true), genHive(false)
	for pose := range monkeyPoses {
		s := genMonkey(pose)
		a.monkey[0][pose] = s
		a.monkey[1][pose] = mirrorSprite(s)
	}
	woods = a
	return a
}

// giantScale is how much bigger than a small tree a giant is drawn.
const giantScale = 2.7

var (
	bananaLeafRamp = mkRamp(0x1c4a1e, 0x2a6a24, 0x3d8a2a, 0x5aa833, 0x82c446, 0xaedb62)
	bananaStemRamp = mkRamp(0x2c3a18, 0x44561f, 0x617429, 0x7f9136, 0x9dab4a)
	bananaRamp     = mkRamp(0x6a5a10, 0x9c8418, 0xc9ac22, 0xe8cc3a, 0xf6e472)
	budRamp        = mkRamp(0x3a0e26, 0x5c1838, 0x82264c, 0xa63a62)
	hiveRamp       = mkRamp(0x4a3010, 0x6e4a18, 0x946820, 0xbb8a2c, 0xd9ae4a, 0xf0d070)
	monkeyRamp     = mkRamp(0x2a180c, 0x432814, 0x5e3a1c, 0x7a4f28, 0x956636)
	monkeyFace     = mkRamp(0x6a4630, 0x946448, 0xb8825e, 0xd4a07a)
)

// genBananaTree is a banana plant: a soft green stem under great paddle
// leaves, torn by the wind, with a hanging bunch when ripe.
func genBananaTree(s float64, seed uint64, ripe bool) sprite {
	rng := rand.New(rand.NewPCG(seed, 0xba7a))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	s *= k
	c := markHD(newCanvas(int(math.Ceil(70*s)), int(math.Ceil(70*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k
	lean := r(-3, 3) * s
	tx, ty := bx+lean, by-r(30, 35)*s
	stem := limb{bx, by, bx + lean*0.2, (by + ty) / 2, tx, ty, 3.2 * s, 2.2 * s}
	drawLimbIn(c, stem, bananaStemRamp, 91)
	// Brown dead sheaths low on the stem.
	for t := 0.05; t < 0.45; t += 0.11 {
		x, y := stem.at(t)
		w := lerpf(stem.w0, stem.w1, t)
		c.line(x-w, y, x+w*0.3, y+2*s, func(px, py int, _ float64) color.RGBA { return monkeyRamp.at(2) })
	}
	n := 6 + rng.IntN(2)
	type leafAt struct{ a, L float64 }
	var leaves []leafAt
	for i := range n {
		a := -math.Pi - 0.25 + (math.Pi+0.5)*float64(i)/float64(n-1) + r(-0.15, 0.15)
		leaves = append(leaves, leafAt{a, r(22, 28) * s})
	}
	// The back leaves first, then the bunch, then the front ones over it.
	for i, l := range leaves {
		if i%2 == 0 {
			bananaLeaf(c, tx, ty, l.a, l.L, s, uint32(seed)*7+uint32(i))
		}
	}
	if ripe {
		bunch(c, tx+4*s, ty+3*s, s)
	} else { // the bare stalk, and the bud
		c.line(tx+2*s, ty+2*s, tx+5*s, ty+12*s, func(px, py int, _ float64) color.RGBA { return bananaStemRamp.at(1) })
		c.ellipse(tx+5*s, ty+14*s, 2*s, 3*s, func(x, y int, nx, ny float64) color.RGBA { return budRamp.pick(lum(nx, ny), x, y) })
	}
	for i, l := range leaves {
		if i%2 == 1 {
			bananaLeaf(c, tx, ty, l.a, l.L, s, uint32(seed)*7+uint32(i))
		}
	}
	c.outline(0.6)
	return trimmed(c, bx, by)
}

// bananaLeaf is one great paddle leaf from (x0, y0) at angle a, arching out
// and drooping, its edges torn into strips.
func bananaLeaf(c *canvas, x0, y0, a, L, s float64, seed uint32) {
	ex, ey := x0+math.Cos(a)*L, y0+math.Sin(a)*L*0.6+L*0.55
	cx, cy := x0+math.Cos(a)*L*0.55, y0+math.Sin(a)*L*0.9-L*0.15
	at := func(t float64) (float64, float64) {
		u := 1 - t
		return u*u*x0 + 2*u*t*cx + t*t*ex, u*u*y0 + 2*u*t*cy + t*t*ey
	}
	W := 5.2 * s
	n := int(L*2) + 6
	if c.hd() {
		n = int(L*3) + 6
	}
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := at(t)
		x2, y2 := at(min(1, t+0.01))
		dx, dy := x2-x, y2-y
		d := math.Hypot(dx, dy)
		if d == 0 {
			continue
		}
		nx, ny := -dy/d, dx/d
		if ny > 0 { // so side +1 is always the upper edge
			nx, ny = -nx, -ny
		}
		w := W * math.Pow(math.Sin(math.Pi*min(1, t*1.15+0.05)), 0.6)
		if t < 0.12 {
			w = W * 0.15 // the stalk
		}
		torn := hashf(int(t*40), 0, seed) < 0.3 && t > 0.3
		for k := -w; k <= w; k += 0.5 {
			if torn && math.Abs(k) > w*0.35 && math.Mod(math.Abs(k), 2) < 1 {
				continue // a tear between strips
			}
			px, py := x+nx*k, y+ny*k
			l := 0.55 + k/W*0.3 - t*0.12 + (hashf(int(px), int(py), seed)-0.5)*0.12
			if math.Abs(k) < 0.6 {
				l = 0.85 // the pale midrib
			}
			if c.hd() {
				// Fine parallel veins from the midrib to the edge, the
				// lower half in the leaf's own shade.
				if v := math.Mod(t*L/2.2+math.Abs(k)*0.35, 1); v < 0.3 && math.Abs(k) > 1 {
					l -= 0.1
				}
				if k < -1 {
					l -= 0.08
				}
				if math.Abs(k) < 1.2 {
					l = 0.9
				} else if math.Abs(k) > w-1 {
					l -= 0.12
				}
			}
			c.set(int(px), int(py), bananaLeafRamp.pick(l, int(px), int(py)))
		}
	}
}

// bunch is a hanging bunch of bananas below (x, y), with its purple bud.
func bunch(c *canvas, x, y, s float64) {
	c.line(x-2*s, y-1*s, x, y+3*s, func(px, py int, _ float64) color.RGBA { return bananaStemRamp.at(1) })
	for row := range 4 {
		ry := y + 3*s + float64(row)*2.6*s
		half := 3.4 - float64(row)*0.5
		for k := -half; k <= half; k += 1.5 {
			fx := x + k*s
			c.ellipse(fx, ry, 1.1*s, 2.2*s, func(px, py int, nx, ny float64) color.RGBA {
				return bananaRamp.pick(lum(nx, ny)+0.05, px, py)
			})
			c.set(int(fx), int(ry+2*s), bananaStemRamp.at(0)) // the dark tip
		}
	}
	c.line(x, y+12*s, x+1*s, y+15*s, func(px, py int, _ float64) color.RGBA { return bananaStemRamp.at(1) })
	c.ellipse(x+1*s, y+17*s, 1.8*s, 2.8*s, func(px, py int, nx, ny float64) color.RGBA { return budRamp.pick(lum(nx, ny), px, py) })
}

// genHive is a bees' nest hanging from a branch: a papery, banded, golden
// bulb, dripping honey while it has some.
func genHive(full bool) sprite {
	c := newEnvCanvas(14, 18)
	c.line(7, 0, 7, 3, func(x, y int, _ float64) color.RGBA { return monkeyRamp.at(1) })
	rp := hiveRamp
	if !full {
		rp = mkRamp(0x3a2a18, 0x54402a, 0x6e5838, 0x88704a, 0xa08a62, 0xb8a47c)
	}
	c.ellipse(7, 9.5, 5.6, 6.8, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if envHD { // the bands of its paper: scalloped layers, each lit on top
			band := math.Mod(c.ly(y)+0.35*math.Sin(c.lx(x)*1.7)+30, 2.2)
			switch {
			case band < 0.35:
				l -= 0.3
			case band < 0.8:
				l += 0.1
			}
			l += (hashf(x, y, 193) - 0.5) * 0.08
		} else if y%3 == 0 {
			l -= 0.25 // the bands of its paper
		}
		return rp.pick(l, x, y)
	})
	c.ellipse(7, 13.5, 1.3, 1, func(x, y int, _, _ float64) color.RGBA { return colInk }) // the way in
	if full {
		c.set(4, 15, hiveRamp.at(5))
		c.set(4, 16, hiveRamp.at(4)) // a drip of honey
	}
	c.outline(0.8)
	return mkUp(c, 7, 0)
}

// genMonkey is a monkey facing right in the given pose.
func genMonkey(pose int) sprite {
	c := newCanvas(26, 24)
	limbCol := func(x, y int, _ float64) color.RGBA { return monkeyRamp.pick(0.35, x, y) }
	thick := func(x0, y0, x1, y1 float64) {
		c.line(x0, y0, x1, y1, limbCol)
		c.line(x0+1, y0, x1+1, y1, limbCol)
	}
	fur := func(e ell, bias float64) { roundShape(c, e, monkeyRamp, bias) }
	tail := func(x0, y0 float64, up bool) {
		pts := [][2]float64{{x0, y0}, {x0 - 3, y0 - 1}, {x0 - 5, y0 - 4}, {x0 - 5, y0 - 7}, {x0 - 3, y0 - 8.5}, {x0 - 1.5, y0 - 7}}
		if !up {
			pts = [][2]float64{{x0, y0}, {x0 - 3, y0 + 1}, {x0 - 5.5, y0 + 1}, {x0 - 7, y0 - 1}, {x0 - 7, y0 - 3}}
		}
		for i := 1; i < len(pts); i++ {
			c.line(pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1], limbCol)
		}
	}
	head := func(hx, hy float64) {
		fur(ell{hx, hy, 4.3, 4}, 0.1)
		roundShape(c, ell{hx - 3.6, hy - 0.5, 1.4, 1.6}, monkeyFace, 0) // the ear
		roundShape(c, ell{hx + 1.3, hy + 0.8, 2.6, 2.4}, monkeyFace, 0.15)
		c.set(int(hx+1), int(hy-0.5), colInk) // the eye
		c.set(int(hx+2), int(hy-0.5), color.RGBA{0xfa, 0xf0, 0xd8, 0xff})
		c.set(int(hx+3), int(hy+2), monkeyFace.at(0)) // the mouth
	}
	switch pose {
	case monkeySit, monkeyEat:
		tail(10, 20, false)
		fur(ell{12, 15, 4.6, 5.6}, 0.1)                           // body, upright
		roundShape(c, ell{13.5, 16.5, 2.4, 3.2}, monkeyFace, 0.1) // pale chest
		fur(ell{13, 20.5, 4, 2}, 0.2)                             // haunches
		thick(15.5, 21, 18, 21.5)                                 // a foot
		head(14, 8)
		if pose == monkeyEat {
			thick(15, 13, 17.5, 10.5)
			c.ellipse(18.5, 9, 1, 2.4, func(x, y int, nx, ny float64) color.RGBA { return bananaRamp.pick(lum(nx, ny), x, y) })
		} else {
			thick(14.5, 14, 16, 19) // an arm hanging down
		}
	case monkeyWalkA, monkeyWalkB:
		tail(7, 13, true)
		fur(ell{12, 15, 6, 3.6}, 0.1) // body on all fours
		d := 1.5
		if pose == monkeyWalkB {
			d = -1.5
		}
		thick(8+d, 16, 8-d, 22)
		thick(15-d, 16, 16+d, 22)
		head(18, 11)
	case monkeyLeap:
		tail(6, 12, true)
		fur(ell{12, 13, 6.4, 3.2}, 0.1)
		thick(7, 14, 3, 17)   // hind legs flung back
		thick(16, 14, 20, 17) // arms reaching
		head(19, 9)
	}
	c.outline(0.8)
	return mk(c, 13, 22)
}

// --- drawing ----------------------------------------------------------------------

// woodsEntitySprite is entitySprite for the woods' entities.
func (g *Game) woodsEntitySprite(e protocol.EntityView) (sprite, float64, bool) {
	a := woodsArtOf()
	v := int(sim.HashTile(int(e.TX), int(e.TY)) / 8)
	switch e.Kind {
	case sim.EntityBananaTree:
		if e.Ready {
			return a.banana[v%len(a.banana)], 0.05, true
		}
		return a.bananaBare[v%len(a.bananaBare)], 0.05, true
	case sim.EntityGiantTree:
		if !e.Ready {
			return a.giantStump, 0, true
		}
		return a.giants[v%len(a.giants)], 0.012, true
	case sim.EntityGiantRoot:
		return a.giantStump, 0, true // never drawn; see drawEntity
	}
	return sprite{}, 0, false
}

// woodsShadow is shadowWidth for the woods' entities.
func woodsShadow(e protocol.EntityView) (float64, bool) {
	switch e.Kind {
	case sim.EntityBananaTree:
		return 24, true
	case sim.EntityGiantTree:
		if e.Ready {
			return 96, true
		}
		return 60, true
	case sim.EntityGiantRoot:
		return 0, true
	}
	return 0, false
}

// hasHive reports whether bees nest in e, as the client sees it.
func hasHive(e protocol.EntityView) bool {
	return (e.Kind == sim.EntityTree || e.Kind == sim.EntityGiantTree) && e.Ready && e.Var == 1
}

// drawHive hangs the hive on tree e drawn as s at (sx, sy), with its bees
// about it.
func (g *Game) drawHive(dst drawTarget, e protocol.EntityView, s sprite, sx, sy float64) {
	a := woodsArtOf()
	h := a.hive
	if e.Fuel > 0 {
		h = a.hiveEmpty
	}
	k := 1.0
	if e.Kind == sim.EntityGiantTree {
		k = 1.6
	}
	side := 1.0
	if sim.HashTile(int(e.TX), int(e.TY))%2 == 0 {
		side = -1
	}
	hx, hy := math.Round(sx+side*s.ax*0.18), math.Round(sy-s.ay*0.5)
	drawAt(dst, h, hx, hy, k, k, 0, nil)
	if g.darknessNow() < 0.5 { // bees are about by day
		g.drawBees(dst, hx, hy+9*k, 4, 7*k, uint32(e.ID))
	}
}

// bee is one pixel of a bee.
var bee [2]*ebiten.Image

// drawBees draws n bees buzzing round (x, y) within r pixels.
func (g *Game) drawBees(dst drawTarget, x, y float64, n int, r float64, seed uint32) {
	if bee[0] == nil {
		bee[0], bee[1] = ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)
		bee[0].Fill(color.RGBA{0xf2, 0xc8, 0x2a, 0xff})
		bee[1].Fill(color.RGBA{0x1a, 0x16, 0x10, 0xff})
	}
	t := float64(g.frame)
	for i := range n {
		k := float64(i)*1.9 + float64(seed%113)
		a := t*(0.13+0.03*float64(i%3)) + k
		bx := x + math.Cos(a)*r*(0.6+0.4*math.Sin(k)) + math.Sin(a*3.1)*1.5
		by := y + math.Sin(a*1.3)*r*0.6
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(bx), math.Round(by))
		dst.DrawImage(bee[0], op)
		op.GeoM.Translate(1, 0)
		dst.DrawImage(bee[1], op)
	}
}

// drawSwarm draws the bees round a player they are after.
func (g *Game) drawSwarm(dst drawTarget, p protocol.PlayerView, sx, sy float64) {
	if p.Bees {
		g.drawBees(dst, sx, sy-14, 12, 11, uint32(p.ID))
	}
}

// woodsSprite is creatureSprite for the woods' creatures.
func (g *Game) woodsSprite(c protocol.CreatureView, moving bool) (sprite, float64, bool) {
	if c.Kind != sim.CreatureMonkey {
		return sprite{}, 0, false
	}
	a := woodsArtOf()
	side := 0
	if c.FaceLeft {
		side = 1
	}
	t := g.frame + int(c.ID)*7
	switch {
	case feeding(c):
		return a.monkey[side][monkeyEat], 0, true
	case moving && c.State == sim.CreatureFlee:
		if (t/5)%2 == 0 {
			return a.monkey[side][monkeyLeap], 3, true
		}
		return a.monkey[side][monkeyWalkA], 0, true
	case moving:
		return a.monkey[side][monkeyWalkA+(t/7)%2], 0, true
	}
	return a.monkey[side][monkeySit], 0, true
}

// genWoodsIcons draws the bag icons of bananas and honey.
func genWoodsIcons(icons *[sim.NumItems]*ebiten.Image) {
	b := newCanvas(16, 16)
	for i, d := range []float64{-3, 0, 3} {
		x0 := 8 + d
		b.polygon([]float64{x0 - 1, 3, x0 + 1, 3, x0 + 2 + d*0.3, 9, x0 + 0.5 + d*0.6, 14, x0 - 1 + d*0.6, 13.5, x0 - 0.5 + d*0.3, 9}, func(x, y int) color.RGBA {
			return bananaRamp.pick(0.7-float64(i)*0.1-float64(y)*0.015, x, y)
		})
	}
	b.line(6, 2, 10, 2, func(x, y int, _ float64) color.RGBA { return bananaStemRamp.at(1) })
	b.outline(0.8)
	icons[sim.ItemBanana] = b.image()

	h := newCanvas(16, 16)
	h.polygon([]float64{4, 6, 12, 6, 12.5, 13, 11, 14.5, 5, 14.5, 3.5, 13}, func(x, y int) color.RGBA {
		return mkRamp(0x5a4a3a, 0x7a6650, 0x9a8468, 0xb8a284).pick(0.6-float64(x)*0.02, x, y) // a clay pot
	})
	h.polygon([]float64{4, 6, 12, 6, 11, 8, 5, 8}, func(x, y int) color.RGBA { return hiveRamp.pick(0.8, x, y) })
	h.line(5, 7, 5, 11, func(x, y int, _ float64) color.RGBA { return hiveRamp.at(4) }) // a drip down the side
	h.ellipse(8, 4.5, 4.5, 1.6, func(x, y int, nx, ny float64) color.RGBA { return hiveRamp.pick(lum(nx, ny)+0.2, x, y) })
	h.outline(0.8)
	icons[sim.ItemHoney] = h.image()
}

// --- sounds -----------------------------------------------------------------------

// genWoodsSFX adds an angry swarm's buzz to the effect bank, with a random
// source of its own.
func genWoodsSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0xb33, 0x5))
	for _, k := range []float64{1, 1.07} {
		x := make([]float64, samples(1.2))
		for i := range 6 { // many bees, each a buzzing saw a little out of tune
			f := (210 + float64(i)*17) * k
			ph, wob := 0.0, r.Float64()*6
			addAt(x, render(1.1, func(t float64, _ int) float64 {
				ph += f * (1 + 0.04*fsin(t*(5+wob))) / sampleRate
				saw := 2*(ph-math.Floor(ph)) - 1
				return saw * env(t, 0.15, 0.6) * (0.7 + 0.3*fsin(t*(9+wob)))
			}), 0.02*float64(i), 0.2)
		}
		add("buzz", 0.6, x)
	}
}

// swarmSounds buzzes while bees are after me.
func (g *Game) swarmSounds() {
	for _, p := range g.curr.Players {
		if p.ID == g.myID && p.Bees && snd.ready("buzz", 900*time.Millisecond) {
			snd.playVar("buzz", 0.55, 0, 0.05)
		}
	}
}
