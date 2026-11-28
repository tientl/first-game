package client

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The props of the natural caves' themed chambers (sim.CaveProp), drawn at
// HD the first time one is wanted: clusters of glowing jade crystals, giant
// glowing mushrooms, webs and egg sacs, bones, a great skeleton in the rock,
// pictures on a wall, the tomb's coffin and jars, roots, icicles, shards of
// obsidian, the bats' roost; and the stalagmites of the themed chambers (a
// carved pillar, a spire of ice, a column of basalt, a crystal taller than a
// man). Those that lie flat on the floor are drawn under everything else.

type cavePropArt struct {
	props   [sim.NumCaveProps][]sprite
	special map[uint8]sprite // EntityStalagmite looks past the ordinary ones
	bat     [4]sprite        // a bat's wingbeat
}

var cavePropsOnce struct {
	sync.Once
	art cavePropArt
}

func caveProps() *cavePropArt {
	cavePropsOnce.Do(func() {
		a := &cavePropsOnce.art
		mk := func(c *canvas, ax, ay float64) sprite {
			img := c.image()
			noteCover(img, c)
			return spriteHD(img, ax, ay)
		}
		for v := range 3 {
			a.props[sim.PropCrystal] = append(a.props[sim.PropCrystal], mk(crystalCanvas(v, false), 16, 30))
			a.props[sim.PropGlowShroom] = append(a.props[sim.PropGlowShroom], mk(glowShroomCanvas(v), 24, 56))
			a.props[sim.PropWeb] = append(a.props[sim.PropWeb], mk(webCanvas(v), 24, 15))
			a.props[sim.PropBones] = append(a.props[sim.PropBones], mk(bonesCanvas(v), 18, 12))
			a.props[sim.PropUrn] = append(a.props[sim.PropUrn], mk(urnCanvas(v), 12, 24))
			a.props[sim.PropRoots] = append(a.props[sim.PropRoots], mk(rootsCanvas(v), 16, 66))
			a.props[sim.PropIcicles] = append(a.props[sim.PropIcicles], mk(iciclesCanvas(v), 16, 34))
			a.props[sim.PropObsidian] = append(a.props[sim.PropObsidian], mk(obsidianCanvas(v), 14, 16))
		}
		a.props[sim.PropEggSac] = []sprite{mk(eggSacCanvas(), 14, 20)}
		a.props[sim.PropFossil] = []sprite{mk(fossilCanvas(), fossilAX, fossilAY)}
		a.props[sim.PropSarcophagus] = []sprite{mk(sarcophagusCanvas(), 26, 30)}
		a.props[sim.PropRoost] = []sprite{mk(roostCanvas(), 18, 14)}
		for p := range sim.PaintingHerds + 1 {
			a.props[sim.PropPainting] = append(a.props[sim.PropPainting], mk(paintingCanvas(p), 36, 60))
		}
		a.special = map[uint8]sprite{
			sim.StalagPillar:  mk(pillarCanvas(), 16, 70),
			sim.StalagIce:     mk(iceSpireCanvas(), 24, 62),
			sim.StalagBasalt:  mk(basaltCanvas(), 20, 58),
			sim.StalagCrystal: mk(crystalCanvas(0, true), 22, 66),
		}
		for f := range a.bat {
			a.bat[f] = spriteHD(batCanvas(f).image(), 7, 5)
		}
	})
	return &cavePropsOnce.art
}

// cavePropFlat reports whether prop p lies flat on the floor, drawn under
// everything that stands.
func cavePropFlat(p sim.CaveProp) bool {
	switch p {
	case sim.PropWeb, sim.PropBones, sim.PropFossil, sim.PropRoost:
		return true
	}
	return false
}

// cavePropSprite is the sprite of prop entity e.
func cavePropSprite(e protocol.EntityView) (sprite, bool) {
	p := sim.CaveProp(e.Var)
	if p >= sim.NumCaveProps {
		return sprite{}, false
	}
	ss := caveProps().props[p]
	if len(ss) == 0 {
		return sprite{}, false
	}
	if p == sim.PropPainting {
		pic, _ := sim.PaintingOf(int(e.Fuel))
		return ss[pic%len(ss)], true
	}
	return ss[int(sim.HashTile(int(e.TX), int(e.TY))>>4)%len(ss)], true
}

// drawCaveProp draws a standing cave prop in the sorted pass (the flat ones
// are drawn by drawCaveGround).
func (g *Game) drawCaveProp(dst drawTarget, e protocol.EntityView, d drawable) bool {
	p := sim.CaveProp(e.Var)
	if cavePropFlat(p) {
		if p == sim.PropRoost {
			g.drawBats(dst, e, d)
		}
		return true
	}
	s, ok := cavePropSprite(e)
	if !ok {
		return true
	}
	if solid := g.seeThrough(e, s, d); solid < 1 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(d.sx-s.ax, d.sy-s.ay)
		op.ColorScale.ScaleAlpha(solid)
		drawSprite(dst, s, op)
		return true
	}
	drawAt(dst, s, d.sx, d.sy, 1, 1, 0, nil)
	return true
}

// drawCaveGround draws the cave props that lie flat on the floor.
func (g *Game) drawCaveGround(dst drawTarget, cam camera) {
	if !g.naturalCaves() {
		return
	}
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityCaveProp || !cavePropFlat(sim.CaveProp(e.Var)) {
			continue
		}
		if s, ok := cavePropSprite(e); ok {
			sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
			drawAt(dst, s, sx, sy, 1, 1, 0, nil)
		}
	}
}

// specialStalag is the sprite of a stalagmite of a themed look, if e is one.
func specialStalag(e protocol.EntityView) (sprite, bool) {
	if e.Var < sim.StalagPillar {
		return sprite{}, false
	}
	s, ok := caveProps().special[e.Var]
	return s, ok
}

// drawBats draws the bats of a roost: a few hanging on the wall above it,
// and one or two flitting round, dark against the rock.
func (g *Game) drawBats(dst drawTarget, e protocol.EntityView, d drawable) {
	a := caveProps()
	h := sim.HashTile(int(e.TX), int(e.TY))
	for k := range 3 {
		x := d.sx - 10 + float64(k)*9 + float64(h>>(k*3)&3)
		y := d.sy - 34 - float64(h>>(k*5+2)&7)
		drawAt(dst, a.bat[0], x, y, 1, 1, 0, nil) // hanging, wings folded
	}
}

// drawCaveBats draws the bats flitting round their roosts, over the dark (a
// bat is seen by its moving, dimly, where the rock round it is not).
func (g *Game) drawCaveBats(dst drawTarget, cam camera) {
	if _, ok := g.caveLevel(); !ok {
		return
	}
	a := caveProps()
	for _, e := range g.curr.Entities {
		if e.Kind != sim.EntityCaveProp || sim.CaveProp(e.Var) != sim.PropRoost {
			continue
		}
		sx, sy := cam.toScreen(float64(e.TX)+0.5, float64(e.TY)+0.5)
		h := sim.HashTile(int(e.TX), int(e.TY))
		for k := range 3 {
			t := float64(g.frame)*0.035 + float64(k)*2.1 + float64(h&255)*0.1
			x := sx + 30*math.Cos(t)*(1+0.3*math.Sin(t*1.7))
			y := sy - 30 + 12*math.Sin(t*1.3)
			op := &ebiten.DrawImageOptions{}
			s := a.bat[1+(g.frame/4+k)%3]
			op.GeoM.Translate(x-s.ax, y-s.ay)
			l := float32(1)
			if g.caveDark.on { // lit as the cave is where it flies, though never quite lost; see cavedark.go
				l = float32(clamp01(0.3 + 1.4*g.caveLightAt(x, y)))
			}
			op.ColorScale.Scale(0.7*l, 0.7*l, 0.7*l, 0.7)
			drawSprite(dst, s, op)
			g.drawBatEyes(dst, x, y, k) // their eyes catch the light in the dark
		}
	}
}

// caveGlow is how a thing of the caves glows, if it does: its light's
// radius in tiles and its colour.
func caveGlow(e protocol.EntityView) (r float64, col [3]float32, ok bool) {
	switch {
	case e.Kind == sim.EntityCaveProp && sim.CaveProp(e.Var) == sim.PropCrystal:
		return 2.4, [3]float32{0.35, 1, 0.8}, true
	case e.Kind == sim.EntityCaveProp && sim.CaveProp(e.Var) == sim.PropGlowShroom:
		return 2.8, [3]float32{0.45, 0.6, 1}, true
	case e.Kind == sim.EntityStalagmite && e.Var == sim.StalagCrystal:
		return 3.6, [3]float32{0.35, 1, 0.8}, true
	}
	return 0, [3]float32{}, false
}

// --- the drawings -------------------------------------------------------------------

var (
	cpJade   = mkRamp(0x1e314b, 0x343f41, 0x4d6659, 0x739a70, 0xa5c67c, 0x9ae5d5, 0xe5efef)
	cpShroom = mkRamp(0x1a1f2e, 0x1e314b, 0x2f4c6c, 0x39275e, 0x5c3a6f, 0x7e659b, 0xb8a1c2)
	cpGlow   = mkRamp(0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xe5efef)
	cpStem   = mkRamp(0x533b41, 0x7d595d, 0xb49d7e, 0xc4c6b8, 0xe3e6ac)
	cpClay   = mkRamp(0x3c252b, 0x684039, 0x8d4830, 0xb77862, 0xc57835, 0xeecb90)
	cpSand   = mkRamp(0x3c252b, 0x5c2b34, 0x8d4830, 0xb77862, 0xe1a845, 0xeecb90, 0xe3e6ac)
	cpOchre  = mkRamp(0x8d3649, 0xb23657, 0xc33c40, 0xc57835)
	cpGlass  = mkRamp(0x0b0d14, 0x1a1f2e, 0x2f193e, 0x39275e, 0x5c3a6f, 0x7e659b, 0xb8a1c2, 0xe5efef)
	cpIce    = mkRamp(0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xe5efef, 0xffffff)
	cpBasalt = mkRamp(0x0b0d14, 0x1a1f2e, 0x282e3b, 0x343f41, 0x414453, 0x696f80, 0x8d99a4)
	cpBat    = mkRamp(0x1a1f2e, 0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b, 0x867066)
)

// prism draws a crystal on c: from its foot (bx, by) out along angle a
// (radians from straight up, clockwise), ln long and half w wide, its
// facets lit from the left, its tip pointed, glowing within.
func prism(c *canvas, bx, by, a, ln, w float64, rp ramp) {
	sa, ca := math.Sin(a), math.Cos(a)
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := px-bx, py-by
		s := dx*sa - dy*ca // along it, up from the foot
		t := dx*ca + dy*sa // across it
		if s < -0.5 || s > ln {
			return
		}
		hw := w
		if tip := ln - 1.7*w; s > tip {
			hw = w * (ln - s) / (1.7 * w)
		}
		if math.Abs(t) > hw {
			return
		}
		f := t / math.Max(hw, 0.3)
		l := 0.55
		switch {
		case f < -0.35:
			l = 0.82 // the lit facet
		case f > 0.35:
			l = 0.32 // the shaded one
		}
		if math.Abs(math.Abs(f)-0.35) < 0.09 {
			l += 0.1 // the edge between facets
		}
		l += 0.15*s/ln - 0.12*(1-s/ln)*(1-s/ln)
		if s > ln-1.7*w && f < 0 {
			l += 0.1 // the tip, catching the light
		}
		c.put(x, y, rp.pick(clamp01(l), x, y))
	})
}

// crystalCanvas draws a cluster of jade crystals growing from a knob of
// rock, look v; big, a crystal taller than a man with its brood.
func crystalCanvas(v int, big bool) *canvas {
	w, h := 32, 34
	ax, ay := 16.0, 30.0
	k := 1.0
	if big {
		w, h, ax, ay, k = 44, 70, 22, 66, 2.1
	}
	c := hdCanvas(w, h)
	// The knob of rock they grow from.
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := (px-ax)/(9*k*0.8), (py-ay+1)/(4.2*k*0.6)
		if dx*dx+dy*dy <= 1 {
			c.put(x, y, cpBasalt.pick(0.45-0.3*dy-0.2*dx, x, y))
		}
	})
	lays := [][][4]float64{
		{{-0.25, 0, 22, 3.2}, {0.35, 2, 16, 2.6}, {-0.7, -3, 12, 2.2}, {0.05, 3, 26, 3.6}, {0.8, -2, 10, 2}},
		{{0.1, 0, 25, 3.6}, {-0.5, -3, 15, 2.6}, {0.6, 3, 14, 2.4}, {-0.15, 4, 9, 2}},
		{{-0.35, -2, 18, 3}, {0.25, 1, 23, 3.4}, {0.75, 3, 11, 2.2}, {-0.8, 2, 9, 1.9}},
	}
	for _, p := range lays[v%len(lays)] {
		prism(c, ax+p[1]*k, ay-1, p[0], p[2]*k, p[3]*k*0.9, cpJade)
	}
	c.outline(0.6)
	return c
}

// glowShroomCanvas draws giant glowing mushrooms, look v: a big one and one
// or two smaller at its foot, each a pale stem with a skirt under a broad
// dark-blue cap, spotted and rimmed with light, its gills glowing beneath.
func glowShroomCanvas(v int) *canvas {
	c := hdCanvas(48, 60)
	const ax, ay = 24.0, 56.0
	type shroom struct{ x, y, ht, capR, lean float64 }
	sets := [][]shroom{
		{{-9, -1, 20, 8, -1.5}, {2, 0, 40, 16, 2}},
		{{0, 0, 44, 18, -1.5}, {10, 1, 16, 7, 1.5}, {-12, 1.5, 12, 5.5, -1}},
		{{-4, 0, 34, 14, 2.5}, {9, 1, 22, 9, -1}},
	}
	for k, s := range sets[v%len(sets)] {
		bx, by := ax+s.x, ay+s.y
		// The stem, leaning a little, flared at its foot.
		eachHD(c, func(x, y int, px, py float64) {
			t := (by - py) / s.ht
			if t < 0 || t > 1 {
				return
			}
			cx := bx + s.lean*t*t
			hw := s.capR*0.2 + s.capR*0.14*math.Pow(1-t, 3)*2
			dx := (px - cx) / hw
			if math.Abs(dx) > 1 {
				return
			}
			l := 0.46 - 0.36*dx + 0.1*t
			if math.Abs(t-0.66) < 0.03 {
				l += 0.18 // the skirt
			}
			c.put(x, y, cpStem.pick(clamp01(l), x, y))
		})
		cx, cy := bx+s.lean, by-s.ht
		eachHD(c, func(x, y int, px, py float64) {
			dx, dy := (px-cx)/s.capR, (py-cy)/(s.capR*0.78)
			if dy > 0.3 || dx*dx+dy*dy > 1 {
				return
			}
			if dy > 0.14 { // the gills, glowing
				l := 0.55 + 0.3*(1-math.Abs(dx))
				if math.Mod(px*1.4, 2) < 0.6 {
					l -= 0.22
				}
				c.put(x, y, cpGlow.pick(clamp01(l), x, y))
				return
			}
			z := math.Sqrt(math.Max(0, 1-dx*dx-dy*dy))
			l := 0.18 + 0.4*z - 0.22*dx - 0.25*dy
			col := cpShroom.pick(clamp01(l), x, y)
			if dx*dx+dy*dy > 0.82 && dy > -0.2 {
				col = cpGlow.pick(0.55, x, y) // the rim, lit from beneath
			}
			if p, _ := pebble(x, y, 10, 8, uint32(4400+v*7+k), 0.45, 1.6, 3.4); p == pebStone {
				col = cpGlow.pick(0.75+0.25*z, x, y) // a glowing spot
			}
			c.put(x, y, col)
		})
	}
	c.outline(0.6)
	return c
}

// webCanvas draws a web spread over the floor, look v: strands from its
// middle and the spiral round them, seen at a slant.
func webCanvas(v int) *canvas {
	c := hdCanvas(48, 30)
	const ax, ay = 24.0, 15.0
	spokes := 9 + v
	rot := float64(v) * 0.4
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := (px-ax)/21, (py-ay)/12
		r := math.Hypot(dx, dy)
		if r > 1 {
			return
		}
		a := math.Atan2(dy, dx) + rot
		on := false
		// A strand from the middle?
		sp := 2 * math.Pi / float64(spokes)
		if d := math.Abs(math.Remainder(a, sp)) * r * 21; d < 0.45 {
			on = true
		}
		// The spiral: rings a little off round.
		ring := r*9 + 0.6*math.Sin(a*float64(spokes)/2)
		if math.Abs(ring-math.Round(ring)) < 0.07*9*0.1+0.04 && r > 0.08 {
			on = true
		}
		if !on {
			return
		}
		al := uint8(200 * (1 - 0.5*r))
		c.put(x, y, withAlpha(mix(hex(0xe5efef), hex(0xbac9cd), r), al))
	})
	return c
}

// eggSacCanvas draws a clutch of the spiders' egg sacs, silk wrapped round
// them, stuck to the floor by a wall.
func eggSacCanvas() *canvas {
	c := hdCanvas(28, 24)
	const ax, ay = 14.0, 20.0
	for i, s := range [][3]float64{{-6, -2, 4.2}, {4, -3, 5}, {-1, 0, 4.6}, {7, 0, 3.4}, {-8, 1, 3}} {
		bx, by, r := ax+s[0], ay+s[1], s[2]
		eachHD(c, func(x, y int, px, py float64) {
			dx, dy := (px-bx)/r, (py-by+r*0.6)/(r*1.1)
			if d := dx*dx + dy*dy; d <= 1 {
				l := 0.62 - 0.3*dx - 0.3*dy
				if math.Mod(px*0.9+py*0.6+float64(i), 2.2) < 0.4 {
					l += 0.12 // the silk's threads
				}
				c.put(x, y, cpStem.pick(clamp01(l), x, y))
			}
		})
	}
	c.outline(0.6)
	return c
}

// segParam is where point (px, py) lies along segment (x0, y0)-(x1, y1),
// 0 to 1, and how far from it.
func segParam(px, py, x0, y0, x1, y1 float64) (t, d float64) {
	vx, vy := x1-x0, y1-y0
	l2 := vx*vx + vy*vy
	if l2 > 0 {
		t = clamp01(((px-x0)*vx + (py-y0)*vy) / l2)
	}
	return t, math.Hypot(px-x0-t*vx, py-y0-t*vy)
}

// bonesCanvas draws bones of a beast's meal strewn on the floor, look v: the
// long bones of its legs, a rib or two, and in one look its skull; drawn in
// the style of the fossil's bones.
func bonesCanvas(v int) *canvas {
	c := hdCanvas(36, 22)
	b := newBoneArt(c)
	// long draws a bone of a leg: a shaft r thick (halved), swelling at
	// each end into a pair of knobs.
	long := func(x0, y0, x1, y1, r float64) {
		b.bone(0, 0.15)
		dx, dy := x1-x0, y1-y0
		l := math.Hypot(dx, dy)
		tx, ty, a := dx/l, dy/l, math.Atan2(dy, dx)
		b.tube(x0, y0, x1, y1, r*0.8, r*0.7)
		for _, e := range [][3]float64{{x0, y0, -1}, {x1, y1, 1}} {
			ex, ey := e[0]+e[2]*tx*r*0.2, e[1]+e[2]*ty*r*0.2
			b.knob(ex-ty*r*0.7, ey+tx*r*0.7, r*0.95, r*0.8, a)
			b.knob(ex+ty*r*0.7, ey-tx*r*0.7, r*0.95, r*0.8, a)
		}
	}
	rib := func(x0, y0, x1, y1, x2, y2 float64) {
		b.bone(-0.04, 0.2)
		b.curve(boneArc(x0, y0, x1, y1, x2, y2, 0, 1), 0.55, 0.32)
	}
	switch v % 3 {
	case 0:
		rib(11, 16.5, 15, 13.5, 20, 17)
		long(6, 12, 20, 9, 1.1)
		long(22, 5.5, 30, 10.5, 0.9)
		// A skull, its jaw gone: the snout, the window before the eye and the
		// eye, a few teeth.
		b.bone(0, 0.2)
		b.plate(boneLoop([]float64{21.6, 16.2, 22.2, 14.6, 24.5, 13.4, 27.6, 12.6, 30, 12.9, 31.4, 14.4, 31.2, 16.6,
			29.4, 17.6, 26.2, 17.4, 23.2, 17}, 4), 0.7)
		for k := range 4 {
			x := 23 + float64(k)*1.2
			b.plate([]float64{x - 0.35, 16.8, x + 0.35, 16.9, x + 0.05, 18}, 0.25)
		}
		b.hole(boneOval(22.9, 15.2, 0.55, 0.42, -0.3), 0.3)
		b.hole(boneOval(25.4, 15.4, 1.1, 0.65, -0.2), 0.35)
		b.hole(boneOval(28.6, 14.6, 1.25, 1.15, 0), 0.4)
	case 1:
		long(5, 8, 17, 14, 1.2)
		rib(18, 15.5, 22, 12.5, 28, 14.5)
		long(19, 9, 31, 6.5, 1)
		long(10, 16.5, 21, 18.5, 0.85)
	default:
		long(12, 7, 16, 16, 0.8)
		long(8, 15, 28, 11, 1.2)
		rib(23, 17.5, 27, 14.5, 31.5, 17.5)
	}
	b.paint(c, 5500+uint32(v))
	return c
}

// --- the fossil: a great skeleton half bared in the rock ----------------------------
//
// Its bones are drawn in a style for every skeleton of the caves to share
// (the ramps and inks below, and boneArt): old bone, ivory where it is
// cleanest and ochre where the earth has stained it, modelled as rounded
// shapes lit from the top left in four tones and a highlight, porous and
// cracked; each bone ringed by a fine dark line, darkest on its side away
// from the light, and casting a short shadow down and to the right onto
// whatever it lies on. A skeleton is built up on a boneArt bone by bone,
// those further off first, and painted onto a canvas over what lies under
// it.

var (
	// fossilBoneRamp is clean old bone, from the shade in its cracks to the
	// highlight on a knob.
	fossilBoneRamp = mkRamp(0x5b3a2a, 0x8a6446, 0xb39470, 0xd6c197, 0xeadfbe, 0xf8f3e0)
	// fossilStainRamp is the same bone stained by the earth it lay in, tone
	// for tone.
	fossilStainRamp = mkRamp(0x4d2b1f, 0x7a482c, 0xa2683a, 0xc28e55, 0xd9b077, 0xead09c)
	// fossilInk rings a bone on its side away from the light, fossilInkLit
	// on its lit side.
	fossilInk    = hex(0x22161a)
	fossilInkLit = hex(0x4e3329)
	// fossilRockRamp is the bed of grey-brown sandstone the skeleton lies in.
	fossilRockRamp = mkRamp(0x221a1b, 0x382b2a, 0x514038, 0x6b5747, 0x857058, 0x9e886c, 0xb6a082, 0xccb898)
)

// The fossil's size and anchor, in art pixels: the anchor, the middle of
// its slab, stands on the middle of the prop's tile.
const (
	fossilW, fossilH   = 100, 60
	fossilAX, fossilAY = 50.0, 31.0
)

// boneArt is a skeleton being drawn at HD. Each pixel knows the bone on it
// (numbered in the order drawn, a later bone lying over an earlier), how
// high that bone's surface stands there and how its tone is shifted; and
// whether rock shows deep in a hole through a bone there.
type boneArt struct {
	w, h     int
	ox, oy   float64   // added to every point drawn, in art pixels
	id       []int32   // 0: no bone
	ht       []float64 // in image pixels
	shift    []float64
	stain    []float64
	deep     []bool
	cur      int32   // the bone being drawn
	curShift float64 // and its shift and stain
	curStain float64
}

func newBoneArt(c *canvas) *boneArt {
	n := c.w * c.h
	return &boneArt{w: c.w, h: c.h, id: make([]int32, n), ht: make([]float64, n),
		shift: make([]float64, n), stain: make([]float64, n), deep: make([]bool, n)}
}

// bone starts the next bone, which lies over those drawn before it. shift
// darkens it (below 0: a bone further off) or lightens it; stain is how much
// of the earth's ochre it has taken up, about 0 to 1.
func (b *boneArt) bone(shift, stain float64) {
	b.cur++
	b.curShift, b.curStain = shift, stain
}

// each calls f for the pixels of the box (x0, y0)-(x1, y1), in art pixels,
// with the point each stands for.
func (b *boneArt) each(x0, y0, x1, y1 float64, f func(i int, px, py float64)) {
	ix0, iy0 := max(0, int((x0+b.ox)*hdK3)-1), max(0, int((y0+b.oy)*hdK3)-1)
	ix1, iy1 := min(b.w-1, int((x1+b.ox)*hdK3)+1), min(b.h-1, int((y1+b.oy)*hdK3)+1)
	for y := iy0; y <= iy1; y++ {
		for x := ix0; x <= ix1; x++ {
			f(y*b.w+x, (float64(x)+0.5)/hdK3-b.ox, (float64(y)+0.5)/hdK3-b.oy)
		}
	}
}

// fill lays the current bone over the box (x0, y0)-(x1, y1) where f says it
// is, f giving its surface's height there in art pixels. Where it meets
// another part of the same bone, the higher surface shows.
func (b *boneArt) fill(x0, y0, x1, y1 float64, f func(px, py float64) (float64, bool)) {
	b.each(x0, y0, x1, y1, func(i int, px, py float64) {
		h, ok := f(px, py)
		if h *= hdK3; !ok || b.id[i] == b.cur && b.ht[i] >= h {
			return
		}
		b.id[i], b.ht[i], b.shift[i], b.stain[i] = b.cur, h, b.curShift, b.curStain
	})
}

// tube adds a round shaft from (x0, y0) to (x1, y1), r0 thick (halved) at
// its start and r1 at its end, its ends rounded.
func (b *boneArt) tube(x0, y0, x1, y1, r0, r1 float64) {
	r := max(r0, r1)
	b.fill(min(x0, x1)-r, min(y0, y1)-r, max(x0, x1)+r, max(y0, y1)+r, func(px, py float64) (float64, bool) {
		t, d := segParam(px, py, x0, y0, x1, y1)
		rr := lerpf(r0, r1, t)
		return math.Sqrt(max(0, rr*rr-d*d)), d < rr
	})
}

// knob adds a rounded lump: an ellipse rx by ry (halved) about (cx, cy),
// turned a radians clockwise.
func (b *boneArt) knob(cx, cy, rx, ry, a float64) {
	r := max(rx, ry)
	sa, ca := math.Sincos(a)
	b.fill(cx-r, cy-r, cx+r, cy+r, func(px, py float64) (float64, bool) {
		dx, dy := px-cx, py-cy
		u, v := (dx*ca+dy*sa)/rx, (dy*ca-dx*sa)/ry
		d := u*u + v*v
		return min(rx, ry) * math.Sqrt(max(0, 1-d)), d < 1
	})
}

// curve adds a shaft along the line through pts (x, y pairs), tapering from
// r0 thick (halved) to r1.
func (b *boneArt) curve(pts []float64, r0, r1 float64) {
	n := len(pts) / 2
	at := make([]float64, n) // how far along the line each point lies
	for i := 1; i < n; i++ {
		at[i] = at[i-1] + math.Hypot(pts[2*i]-pts[2*i-2], pts[2*i+1]-pts[2*i-1])
	}
	x0, y0, x1, y1 := polyBox(pts)
	r := max(r0, r1)
	b.fill(x0-r, y0-r, x1+r, y1+r, func(px, py float64) (float64, bool) {
		d, s := math.Inf(1), 0.0
		for i := 0; i+1 < n; i++ {
			if t, e := segParam(px, py, pts[2*i], pts[2*i+1], pts[2*i+2], pts[2*i+3]); e < d {
				d, s = e, lerpf(at[i], at[i+1], t)
			}
		}
		rr := lerpf(r0, r1, s/max(at[n-1], 1e-9))
		return math.Sqrt(max(0, rr*rr-d*d)), d < rr
	})
}

// plate adds a flat bone, the polygon pts, its edge rounded over bev art
// pixels and its face swelling a little towards its middle.
func (b *boneArt) plate(pts []float64, bev float64) {
	x0, y0, x1, y1 := polyBox(pts)
	b.fill(x0, y0, x1, y1, func(px, py float64) (float64, bool) {
		in, d := polyDist(pts, px, py)
		return bevelH(d, bev) + 0.12*min(d, 4), in
	})
}

// hole cuts an opening, the polygon pts, through the current bone: rock
// shows deep in it, and the bone round it slopes down into it over bev.
func (b *boneArt) hole(pts []float64, bev float64) {
	x0, y0, x1, y1 := polyBox(pts)
	b.each(x0-bev, y0-bev, x1+bev, y1+bev, func(i int, px, py float64) {
		if b.id[i] != b.cur {
			return
		}
		switch in, d := polyDist(pts, px, py); {
		case in:
			b.id[i], b.ht[i], b.deep[i] = 0, 0, true
		case d < bev:
			b.ht[i] = min(b.ht[i], hdK3*bevelH(d, bev))
		}
	})
}

// dent presses a shallow hollow into the current bone: an ellipse rx by ry
// (halved) about (cx, cy), depth art pixels deep in its middle.
func (b *boneArt) dent(cx, cy, rx, ry, depth float64) {
	b.each(cx-rx, cy-ry, cx+rx, cy+ry, func(i int, px, py float64) {
		u, v := (px-cx)/rx, (py-cy)/ry
		if d := u*u + v*v; d < 1 && b.id[i] == b.cur {
			b.ht[i] -= hdK3 * depth * (1 - d) * (1 - d)
		}
	})
}

// crack breaks the current bone across, along (x0, y0)-(x1, y1).
func (b *boneArt) crack(x0, y0, x1, y1 float64) {
	b.each(min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1), func(i int, px, py float64) {
		if _, d := segParam(px, py, x0, y0, x1, y1); d < 0.2 && b.id[i] == b.cur {
			b.id[i], b.ht[i], b.deep[i] = 0, 0, true
		}
	})
}

// bury hides what lies inside polygon pts of every bone drawn so far under
// the rock it lies in.
func (b *boneArt) bury(pts []float64) {
	x0, y0, x1, y1 := polyBox(pts)
	b.each(x0, y0, x1, y1, func(i int, px, py float64) {
		if in, _ := polyDist(pts, px, py); in {
			b.id[i], b.ht[i], b.deep[i] = 0, 0, false
		}
	})
}

// shadeIn darkens the rock inside polygon pts where no bone lies over it: an
// open mouth.
func (b *boneArt) shadeIn(pts []float64) {
	x0, y0, x1, y1 := polyBox(pts)
	b.each(x0, y0, x1, y1, func(i int, px, py float64) {
		if in, _ := polyDist(pts, px, py); in {
			b.deep[i] = true
		}
	})
}

// paint shades the skeleton onto c, over what is drawn there: each bone in
// its tones, lit from the top left by the slope of its surface, porous,
// cracked here and there and stained in patches; the line round it; the
// shadow it casts; and the rock deep in its holes. seed varies the grain.
func (b *boneArt) paint(c *canvas, seed uint32) {
	under := make([]color.RGBA, len(c.px))
	copy(under, c.px)
	idAt := func(x, y int) int32 {
		if x < 0 || y < 0 || x >= b.w || y >= b.h {
			return 0
		}
		return b.id[y*b.w+x]
	}
	htAt := func(x, y int, id int32) float64 {
		if idAt(x, y) != id {
			return 0
		}
		return b.ht[y*b.w+x]
	}
	darken := func(x, y int, k float64) {
		if u := under[y*b.w+x]; u.A >= 128 {
			c.put(x, y, mix(u, fossilInk, k))
		} else {
			c.put(x, y, withAlpha(fossilInk, uint8(200*k)))
		}
	}
	L := lightDir
	for y := range b.h {
		for x := range b.w {
			i := y*b.w + x
			id := b.id[i]
			if id == 0 {
				shadow := idAt(x-1, y-2) != 0 || idAt(x-2, y-3) != 0 || idAt(x-3, y-4) != 0
				switch {
				case idAt(x-1, y) != 0 || idAt(x, y-1) != 0:
					c.put(x, y, fossilInk) // the line on a bone's shaded side
				case idAt(x+1, y) != 0 || idAt(x, y+1) != 0:
					c.put(x, y, fossilInkLit) // and on its lit side
				case b.deep[i] && shadow:
					darken(x, y, 0.75)
				case b.deep[i]:
					darken(x, y, 0.55)
				case shadow:
					darken(x, y, 0.4)
				}
				continue
			}
			// The line where a bone drawn later lies over this one.
			if idAt(x-1, y) > id || idAt(x, y-1) > id {
				c.put(x, y, fossilInk)
				continue
			}
			if idAt(x+1, y) > id || idAt(x, y+1) > id {
				c.put(x, y, fossilBoneRamp[0])
				continue
			}
			hx := (htAt(x+1, y, id) - htAt(x-1, y, id)) / 2
			hy := (htAt(x, y+1, id) - htAt(x, y-1, id)) / 2
			l := (L[2] - hx*L[0] - hy*L[1]) / math.Sqrt(hx*hx+hy*hy+1)
			t := 0.63 + 0.85*(l-0.52) + b.shift[i] + 0.07*(hashf(x, y, seed)-0.5)
			k := max(0, min(5, int(math.Floor(t*5+0.5))))
			px, py := (float64(x)+0.5)/hdK3, (float64(y)+0.5)/hdK3
			if k >= 3 && hashf(x, y, seed+1) < 0.035 {
				k-- // a pore
			}
			if math.Abs(vnoise(px*0.9, py*0.9, seed+2)-0.5) < 0.016 && vnoise(px*0.3, py*0.3, seed+3) > 0.55 {
				k = min(k, 1) // a hairline crack
			}
			rp := fossilBoneRamp
			if b.stain[i]+1.3*(fbm2(px*0.25, py*0.32, seed+4, 3)-0.5)+0.3*(0.55-t)+0.08*hashf(x, y, seed+5) > 0.6 {
				rp = fossilStainRamp
			}
			c.put(x, y, rp[k])
		}
	}
}

// bevelH is the height of a flat bone d in from its edge, rounded over bev.
func bevelH(d, bev float64) float64 {
	s := 1 - min(d, bev)/bev
	return bev * math.Sqrt(1-s*s)
}

// polyDist reports whether (px, py) lies inside polygon pts (x, y pairs),
// and how far it is from its edge.
func polyDist(pts []float64, px, py float64) (in bool, d float64) {
	d = math.Inf(1)
	n := len(pts) / 2
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
		if (yi > py) != (yj > py) && px < (xj-xi)*(py-yi)/(yj-yi)+xi {
			in = !in
		}
		_, e := segParam(px, py, xi, yi, xj, yj)
		d = min(d, e)
	}
	return in, d
}

// polyBox is the box round points pts (x, y pairs).
func polyBox(pts []float64) (x0, y0, x1, y1 float64) {
	x0, y0, x1, y1 = pts[0], pts[1], pts[0], pts[1]
	for i := 2; i+1 < len(pts); i += 2 {
		x0, y0 = min(x0, pts[i]), min(y0, pts[i+1])
		x1, y1 = max(x1, pts[i]), max(y1, pts[i+1])
	}
	return
}

// boneLoop rounds the polygon through pts (x, y pairs) into a smooth closed
// curve, k points to a side; boneLine does the same for an open line.
func boneLoop(pts []float64, k int) []float64 { return catmullPts(pts, k, true) }
func boneLine(pts []float64, k int) []float64 { return catmullPts(pts, k, false) }

func catmullPts(pts []float64, k int, closed bool) []float64 {
	n := len(pts) / 2
	p := func(i int) (float64, float64) {
		if closed {
			i = (i%n + n) % n
		} else {
			i = max(0, min(n-1, i))
		}
		return pts[2*i], pts[2*i+1]
	}
	segs := n
	if !closed {
		segs = n - 1
	}
	out := make([]float64, 0, 2*(segs*k+1))
	for i := range segs {
		x0, y0 := p(i - 1)
		x1, y1 := p(i)
		x2, y2 := p(i + 1)
		x3, y3 := p(i + 2)
		for s := range k {
			t := float64(s) / float64(k)
			out = append(out, catmull(x0, x1, x2, x3, t), catmull(y0, y1, y2, y3, t))
		}
	}
	if !closed {
		out = append(out, pts[2*n-2], pts[2*n-1])
	}
	return out
}

// catmull is the Catmull-Rom spline through b and c, a and d beyond them, t
// of the way from b to c.
func catmull(a, b, c, d, t float64) float64 {
	return 0.5 * (2*b + (c-a)*t + (2*a-5*b+4*c-d)*t*t + (3*b-a-3*c+d)*t*t*t)
}

// boneOval is an ellipse rx by ry (halved) about (cx, cy), turned a radians,
// as a polygon.
func boneOval(cx, cy, rx, ry, a float64) []float64 {
	sa, ca := math.Sincos(a)
	out := make([]float64, 0, 48)
	for k := range 24 {
		s, c := math.Sincos(float64(k) * math.Pi / 12)
		u, v := c*rx, s*ry
		out = append(out, cx+u*ca-v*sa, cy+u*sa+v*ca)
	}
	return out
}

// boneArc is the curve from (x0, y0) to (x2, y2) bowed towards (x1, y1), as
// points for curve; only the stretch from t0 to t1 of it (a broken bone).
func boneArc(x0, y0, x1, y1, x2, y2, t0, t1 float64) []float64 {
	out := make([]float64, 0, 26)
	for k := range 13 {
		t := lerpf(t0, t1, float64(k)/12)
		u := 1 - t
		out = append(out, u*u*x0+2*u*t*x1+t*t*x2, u*u*y0+2*u*t*y1+t*t*y2)
	}
	return out
}

// fossilCanvas draws the great skeleton of the fossil chamber: a tyrant
// lizard longer than four men, lying on its side in a slab of the bone bed,
// half dug out of it. Its head is raised on its curving neck, the jaws a
// little open on their teeth; the ribs hang from its back, the little arms
// fold under its chest, its legs stride, and the tail runs off into the
// rock. A few bones have broken, gone or come loose.
func fossilCanvas() *canvas {
	c := hdCanvas(fossilW, fossilH)
	b := newBoneArt(c)
	b.oy = -2
	fossilSkeleton(b)
	fossilSlab(c, b)
	b.paint(c, 5200)
	c.outline(0.5)
	fossilDust(c)
	return c
}

// fossilSkeleton lays the fossil's bones on b, the furthest first.
func fossilSkeleton(b *boneArt) {
	// The spine, from the back of the skull to the tail's tip, and where
	// along it a point lies.
	sp := boneLine([]float64{23, 16, 26.5, 14.6, 30.5, 15.2, 34, 17.8, 37.5, 20.3, 43, 21.4, 49, 21, 55, 20.6,
		60, 21.2, 65.5, 22.8, 71, 25, 77, 26.8, 83, 27.2, 88.5, 26, 93.5, 23.6}, 10)
	n := len(sp) / 2
	at := make([]float64, n)
	for i := 1; i < n; i++ {
		at[i] = at[i-1] + math.Hypot(sp[2*i]-sp[2*i-2], sp[2*i+1]-sp[2*i-1])
	}
	// along is the point s art pixels along the spine, and the way the spine
	// runs there (tx, ty), towards the tail.
	along := func(s float64) (x, y, tx, ty float64) {
		i := 0
		for i < n-2 && at[i+1] < s {
			i++
		}
		f := (s - at[i]) / max(at[i+1]-at[i], 1e-9)
		dx, dy := sp[2*i+2]-sp[2*i], sp[2*i+3]-sp[2*i+1]
		l := math.Hypot(dx, dy)
		return sp[2*i] + f*dx, sp[2*i+1] + f*dy, dx / l, dy / l
	}
	// sAt is how far along the spine it is to where it crosses column x.
	sAt := func(x float64) float64 {
		for i := 1; i < n; i++ {
			if sp[2*i] >= x {
				return lerpf(at[i-1], at[i], (x-sp[2*i-2])/max(sp[2*i]-sp[2*i-2], 1e-9))
			}
		}
		return at[n-1]
	}
	// vertebra draws one bone of the spine s along it: its body half long
	// and r high (halved), its spine reaching up ns and its chevron down nc.
	vertebra := func(s, half, r, ns, nc float64) {
		x, y, tx, ty := along(s)
		ux, uy := ty, -tx // up, the way the back faces
		b.bone(0, 0.25)
		if nc > 0 { // the chevron under a bone of the tail, raked back
			b.tube(x-ux*r*0.6, y-uy*r*0.6, x-ux*(r+nc)+tx*nc*0.45, y-uy*(r+nc)+ty*nc*0.45, 0.42, 0.25)
		}
		if ns > 0 { // the spine, raked back, a little knob on its tip
			ex, ey := x+ux*(r+ns)+tx*ns*0.25, y+uy*(r+ns)+ty*ns*0.25
			b.tube(x+ux*r*0.5, y+uy*r*0.5, ex, ey, min(0.62, half*0.75), min(0.5, half*0.6))
			b.knob(ex, ey, min(0.6, half*0.7), 0.42, math.Atan2(ty, tx))
		}
		b.knob(x, y, half, r, math.Atan2(ty, tx))
	}

	// The far leg, striding back, in the shadow of the body.
	far := -0.24
	b.bone(far, 0.45)
	b.tube(60, 22.4, 63.6, 33, 1.15, 1.0)
	b.knob(60, 22.4, 1.5, 1.4, 0)
	b.knob(63.6, 33.2, 1.6, 1.2, 0.3)
	b.bone(far, 0.45)
	b.tube(63.9, 34, 70.4, 41.6, 0.95, 0.75)
	b.knob(64.2, 34.3, 1.4, 1, 0.8)
	b.knob(70.4, 41.8, 1.15, 0.85, 0.8)
	for _, m := range [][4]float64{{71.2, 42.4, 68.6, 47.9}, {70.2, 42.6, 67.4, 48}} {
		b.bone(far, 0.45)
		b.tube(m[0], m[1], m[2], m[3], 0.55, 0.48)
	}
	for _, t := range [][]float64{{68.2, 48.3, 66.4, 49, 65, 49.6, 63.6, 50.6}, {67.2, 48.3, 65.6, 48.4, 64.3, 48.6, 63.1, 49.3}} {
		for k := 0; k+3 < len(t); k += 2 {
			b.bone(far, 0.45)
			if k+4 < len(t) {
				b.tube(t[k], t[k+1], t[k+2], t[k+3], 0.5, 0.42)
			} else {
				b.curve(boneArc(t[k], t[k+1], t[k+2]-0.2, t[k+3]-0.5, t[k+2]-0.4, t[k+3]+0.5, 0, 1), 0.42, 0.1)
			}
		}
	}

	// The ribs, hanging from the back and sweeping back, one of them
	// broken and one snapped short.
	sDorsal, sSacrum := sAt(37.5), sAt(55)
	const dorsals = 9
	gap := (sSacrum - sDorsal) / dorsals
	ribLen := [dorsals]float64{7.5, 10, 11.8, 12.8, 13.2, 12.8, 11.6, 9.6, 7.2}
	for k := range dorsals {
		x, y, tx, ty := along(sDorsal + (float64(k)+0.5)*gap)
		dx, dy := -ty, tx // down, the way the belly faces
		L := ribLen[k]
		x0, y0 := x+dx*0.4, y+dy*0.4
		x1, y1 := x0+dx*0.45*L+tx*0.42*L, y0+dy*0.45*L+ty*0.42*L
		x2, y2 := x0+dx*L+tx*0.28*L, y0+dy*L+ty*0.28*L
		b.bone(0, 0.2)
		switch k {
		case 2: // broken, its lower half slipped
			b.curve(boneArc(x0, y0, x1, y1, x2, y2, 0, 0.55), 0.62, 0.45)
			b.bone(-0.04, 0.35)
			b.curve(boneArc(x0+0.5, y0+0.6, x1+0.5, y1+0.6, x2+0.5, y2+0.6, 0.64, 1), 0.44, 0.34)
		case 6: // snapped short
			b.curve(boneArc(x0, y0, x1, y1, x2, y2, 0, 0.6), 0.62, 0.46)
		default:
			b.curve(boneArc(x0, y0, x1, y1, x2, y2, 0, 1), 0.62, 0.36)
		}
	}

	// The spine: the neck, the back, the hips and the tail, two bones of it
	// gone and the last of it still in the rock.
	sNeck := sAt(23.8)
	for k := range 7 {
		g := (sDorsal - sNeck) / 7
		s := sNeck + (float64(k)+0.5)*g
		x, y, tx, ty := along(s)
		b.bone(0, 0.2)
		b.tube(x-ty*0.7, y+tx*0.7, x+tx*g*1.1-ty*1.2, y+ty*g*1.1+tx*1.2, 0.3, 0.14) // a neck rib
		vertebra(s, 0.43*g, 1.05+0.05*float64(k), 0.7+0.12*float64(k), 0)
	}
	for k := range dorsals {
		vertebra(sDorsal+(float64(k)+0.5)*gap, 0.42*gap, 1.2, 2.9+0.35*math.Sin(float64(k)/8*math.Pi), 0)
	}
	sTail := sAt(61.5)
	for k := range 5 {
		g := (sTail - sSacrum) / 5
		vertebra(sSacrum+(float64(k)+0.5)*g, 0.45*g, 1.15, 3.1, 0) // their spines showing over the hip
	}
	s, g := sTail, 1.95
	for k := range 23 {
		kf := float64(k)
		if k != 12 && k != 13 {
			vertebra(s+g/2, 0.42*g, 1.15*(1-kf/42), 2.3*max(0, 1-kf/15), 2.5*max(0, 1-kf/19)*min(1, kf))
		}
		s += g
		g *= 0.975
	}
	ex, ey, _, _ := along(s - 1)
	b.bury(boneOval(ex+0.6, ey+0.2, 2.6, 2.4, 0.5)) // its tip, still in the rock

	// The hips: the long blade of the ilium over the hip bones of the
	// spine, the pubis reaching forward to its boot, the ischium back.
	b.bone(0, 0.2)
	b.plate(boneLoop([]float64{52.4, 21.4, 52.9, 19.6, 54.6, 18.5, 57.5, 18.2, 60.5, 17.8, 63.3, 17.9, 65.4, 18.6,
		66.4, 19.9, 66.2, 21.4, 64.6, 22.2, 62.6, 22.5, 61.2, 23.3, 59.8, 22.7, 58.2, 22.5, 56.8, 23.3, 55.2, 23,
		53.6, 22.6}, 4), 0.7)
	b.dent(56.2, 20.2, 3, 1.3, 0.85)
	b.dent(62.8, 20, 2.6, 1.2, 0.8)
	b.curve(boneLine([]float64{58, 22, 59.6, 21.3, 61.4, 21.9}, 4), 0.95, 0.8) // the rim over the hip joint
	b.bone(0, 0.3)
	b.tube(56, 22.8, 51.6, 33.2, 1.0, 0.75)
	b.knob(51.8, 34, 3, 1.15, -0.12)
	b.bone(0, 0.3)
	b.tube(62.6, 23.2, 66, 30.8, 0.95, 0.5)
	b.knob(66, 30.8, 0.9, 0.7, 0)

	// The shoulder: the strap of the scapula, the coracoid, and the
	// little arm folded under the chest, two claws on its hand.
	b.bone(0.02, 0.25)
	b.tube(41, 18.2, 37.2, 28.2, 1.0, 0.7)
	b.knob(40.8, 18.8, 1.3, 1.0, 1.2)
	b.knob(37.3, 28.2, 1.5, 1.05, 1.0)
	b.bone(0, 0.25)
	b.knob(36, 30.3, 1.7, 1.25, 0.4)
	b.bone(0, 0.2)
	b.tube(36.8, 29.6, 38.6, 34.2, 0.6, 0.55)
	b.knob(36.8, 29.6, 0.85, 0.75, 1.2)
	b.knob(38.6, 34.2, 0.8, 0.7, 1.2)
	b.bone(0, 0.2)
	b.tube(39, 34.6, 37.1, 37.3, 0.42, 0.36)
	b.bone(0, 0.2)
	b.tube(38.4, 34.4, 36.5, 36.9, 0.4, 0.34)
	for _, f := range [][4]float64{{36.6, 37.2, 35.2, 38}, {37.1, 37.6, 36, 38.9}} {
		b.bone(0, 0.2)
		b.tube(f[0], f[1], f[2], f[3], 0.38, 0.34)
		b.curve(boneArc(f[2], f[3], f[2]-1.2, f[3]-0.1, f[2]-1.3, f[3]+1, 0, 1), 0.38, 0.1)
	}

	// The near leg, striding forward: the thighbone, cracked across, the
	// shin with the fibula behind it, the long bones of the foot, and its
	// toes, joint by joint, clawed.
	b.bone(0, 0.2)
	b.tube(59, 22.8, 55, 33.6, 1.25, 1.05)
	b.knob(59, 22.6, 1.7, 1.55, 0.3)
	b.knob(55, 33.8, 1.75, 1.3, 0.35)
	b.crack(56.3, 28.3, 58.4, 29.4)
	b.bone(-0.06, 0.3)
	b.tube(54.4, 35.3, 58, 44.2, 0.45, 0.3)
	b.bone(0, 0.2)
	b.tube(55.3, 34.6, 59.2, 44, 1.0, 0.8)
	b.knob(55.6, 34.9, 1.55, 1.1, 0.4)
	b.knob(59.2, 44.2, 1.3, 0.9, 0.3)
	for _, m := range [][4]float64{{59.9, 45, 57.2, 51.2}, {58.4, 45, 55.4, 51}, {59.2, 45.2, 56.4, 51.5}} {
		b.bone(0, 0.25)
		b.tube(m[0], m[1], m[2], m[3], 0.55, 0.48)
		b.knob(m[2], m[3], 0.62, 0.5, 0.5)
	}
	for _, t := range [][]float64{
		{55.4, 51, 53.4, 51.6, 52, 52.3, 50.6, 52.9},
		{56.4, 51.6, 54.4, 52.6, 52.8, 53.4, 51.6, 53.9, 50.3, 54.5},
		{57.2, 51.4, 55.6, 53, 54.4, 54, 53.3, 54.9},
	} {
		for k := 0; k+3 < len(t); k += 2 {
			b.bone(0, 0.25)
			if k+4 < len(t) {
				b.tube(t[k], t[k+1], t[k+2], t[k+3], 0.5, 0.42)
				b.knob(t[k+2], t[k+3], 0.5, 0.45, 0)
			} else { // the claw, hooked
				dx, dy := t[k+2]-t[k], t[k+3]-t[k+1]
				b.curve(boneArc(t[k], t[k+1], t[k]+dx*1.1, t[k+1]+dy*0.4, t[k]+dx*1.2+0.1, t[k+1]+dy+0.8, 0, 1), 0.45, 0.1)
			}
		}
	}

	// The skull: the lower jaw first, a little open, then the skull over
	// it, deep, its teeth in rows, holed by the nostril, the window before
	// the eye, the eye and the window behind it.
	b.bone(0, 0.3)
	jaw := boneLoop([]float64{22.9, 19.6, 23.3, 20.9, 22.2, 22.4, 19.5, 23.4, 15.5, 23.7, 11.5, 23.3, 8.6, 22.8,
		7.3, 22.2, 7.4, 21.3, 9.5, 21.4, 13, 21.2, 16.5, 20.8, 19, 20.3, 20.5, 19.4, 21.6, 19.9}, 4)
	b.plate(jaw, 0.9)
	for k := range 8 {
		x := 8.6 + float64(k)*1.1
		ln := []float64{0.9, 1.1, 1.15, 1.1, 1.05, 0.95, 0.85, 0.7}[k]
		y := 21.5 - 0.03*float64(k)*float64(k)/2
		b.plate([]float64{x - 0.4, y, x + 0.4, y, x - 0.1, y - ln}, 0.3)
	}
	b.hole(boneOval(19, 21.9, 1.25, 0.48, -0.12), 0.4)
	b.shadeIn([]float64{7.5, 19, 16.5, 19.4, 21.5, 19.7, 19, 20.4, 13, 21.3, 7.4, 21.5})
	b.bone(0, 0.3)
	b.plate(boneLoop([]float64{6.8, 17.6, 7.2, 15.9, 8.6, 14.6, 11.5, 13.6, 14.5, 12.7, 16.6, 11.9, 17.6, 11.3,
		18.8, 11.6, 20.6, 11, 22.6, 11.5, 23.7, 12.8, 23.9, 15, 23.2, 17.2, 22.6, 19.8, 21.2, 19.6, 19.5, 18.9,
		16.5, 19.4, 13, 19.6, 9.8, 19.4, 7.6, 19}, 4), 1.0)
	for k := range 9 {
		x := 8 + float64(k)*1.1
		ln := []float64{1, 1.25, 1.45, 1.5, 1.45, 1.35, 1.2, 1, 0.8}[k]
		y := 19.1 + 0.5*math.Sin(float64(k)/8*math.Pi) - 0.1
		b.plate([]float64{x - 0.45, y - 0.3, x + 0.45, y - 0.3, x + 0.2, y + ln}, 0.32)
	}
	b.hole(boneOval(8.7, 16, 0.95, 0.58, -0.35), 0.4)
	b.hole(boneOval(11.2, 17.2, 0.55, 0.42, 0), 0.35)
	b.hole(boneLoop([]float64{12.2, 18, 13.6, 15.6, 15.4, 14.3, 16.4, 15.4, 16.8, 17.9, 14.5, 18.4}, 4), 0.5)
	b.hole(boneOval(18.5, 13.9, 1.25, 1.25, 0), 0.5)
	b.hole([]float64{17.9, 14.6, 19.1, 14.6, 18.7, 16.9, 18.3, 16.9}, 0.4)
	b.hole(boneLoop([]float64{19.9, 15, 21.5, 13.6, 22.6, 14.2, 22.4, 17.4, 20.9, 18.4, 20, 17.4}, 4), 0.5)

	// Loose in the rock: a bone of the tail that came away, a tooth, a
	// broken rib and a claw of the arm that is gone.
	b.bone(-0.03, 0.4)
	b.knob(80, 36.5, 1.0, 0.75, 1.1)
	b.tube(80, 36.5, 81.7, 38.2, 0.45, 0.35)
	b.bone(-0.03, 0.4)
	b.curve(boneArc(87.5, 38.5, 88.5, 39.2, 89.8, 39.1, 0, 1), 0.4, 0.22)
	b.bone(0, 0.3)
	b.curve(boneArc(11, 27.4, 12.4, 27.1, 13.6, 27.9, 0, 1), 0.42, 0.1)
	b.bone(-0.03, 0.4)
	b.curve(boneArc(43.5, 40.5, 45.4, 40.6, 47, 41.6, 0, 1), 0.45, 0.35)
	b.bone(-0.03, 0.3)
	b.curve(boneArc(30, 32, 31.2, 31.6, 31.6, 32.8, 0, 1), 0.42, 0.1)
	b.bone(-0.05, 0.45)
	b.curve(boneArc(13.5, 41.5, 18.5, 39.8, 23.5, 43.5, 0, 1), 0.6, 0.4)
	b.bone(-0.05, 0.45)
	b.curve(boneArc(24.8, 44.4, 26.2, 45.6, 26.6, 47.4, 0, 1), 0.42, 0.36)
}

// fossilSlab draws the slab of the bone bed the skeleton lies in, a little
// higher than the floor round it: grey-brown sandstone in beds, cracked
// into plates, chipped by the diggers and brushed pale round the bones, its
// edge crumbling into rubble on the floor. b says where the bones lie.
func fossilSlab(c *canvas, b *boneArt) {
	const seed = 5300
	// inside is how far (px, py) lies inside the slab's top, in art pixels
	// (below 0 outside): a block with rounded corners, worn uneven.
	inside := func(px, py float64) float64 {
		const a, b, p = 47.5, 25.0, 3.2
		dx, dy := math.Abs(px-fossilAX)/a, math.Abs(py-fossilAY+0.5)/b
		q := math.Pow(math.Pow(dx, p)+math.Pow(dy, p), 1/p)
		return (1-q)*b + 2.4*(vnoise(px*0.14, py*0.18, seed)-0.5) + 0.9*(vnoise(px*0.6, py*0.7, seed+1)-0.5)
	}
	// Near the bones the rock was brushed clean: how near, in pixels.
	near := make([]float64, len(c.px))
	for i := range near {
		near[i] = 99
	}
	for y := range c.h {
		for x := range c.w {
			if b.id[y*c.w+x] == 0 {
				continue
			}
			for dy := -6; dy <= 6; dy++ {
				for dx := -6; dx <= 6; dx++ {
					if xx, yy := x+dx, y+dy; c.in(xx, yy) {
						j := yy*c.w + xx
						near[j] = min(near[j], math.Hypot(float64(dx), float64(dy)))
					}
				}
			}
		}
	}
	chisel := [][3]float64{{9, 24.5, 0.5}, {45, 36.5, 0.6}, {84, 28, 0.4}, {46.5, 49, 0.7}, {66.5, 36, 0.5}, {27, 26, 0.6}, {49, 12.5, 0.4}}
	const rise = 2.0 // how far the slab stands above the floor
	eachHD(c, func(x, y int, px, py float64) {
		in := inside(px, py)
		if in < 0 {
			if inside(px, py-rise) >= 0 { // its face, turned to the south
				l := 0.36 - 0.1*(py-math.Floor(py/rise)*rise)/rise
				if math.Mod(py*1.7+1.2*math.Sin(px*0.31), 2.2) < 0.45 {
					l -= 0.07 // the beds of the stone
				}
				l += 0.12 * (vnoise(px*0.9, py*1.6, seed+2) - 0.5)
				c.put(x, y, fossilRockRamp.pick(l, x, y))
				return
			}
			// Rubble crumbled off it on the floor round it, most below.
			if in > -4 {
				dens := 0.35 * (1 + in/4) * (0.5 + clamp01((py-fossilAY)/12))
				if part, l := pebble(x, y, 8, 7, seed+5, dens, 1.3, 3); part == pebStone {
					c.put(x, y, fossilRockRamp.pick(0.3+0.6*l, x, y))
				}
			}
			return
		}
		// The top: beds of sandstone, cracked into plates.
		l := 0.54 + 0.08*(fbm2(px*0.12, py*0.15, seed+6, 3)-0.5)
		if math.Mod(py*0.55+1.3*math.Sin(px*0.08)+0.8*vnoise(px*0.1, py*0.1, seed+7), 2) < 0.18 {
			l -= 0.05 // a bed's line
		}
		clean := near[y*c.w+x]
		edge, id, nx, ny := plates(px/9, py/6.5, seed+8)
		edge *= 7.5
		l += 0.05 * (float64(id&255)/255 - 0.5)
		// Cracked into plates, most of all towards its edge.
		if clean > 7 && (in < 6 || vnoise(px*0.08, py*0.1, seed+10) > 0.45) {
			switch {
			case edge < 0.22:
				l = 0.34 // a crack
			case edge < 0.5 && nx*lightDir[0]+ny*lightDir[1] < 0:
				l += 0.08 // the lip beyond it, catching the light
			}
		}
		// The rim: lit along the top and left, shaded along the bottom and
		// right, crumbling.
		if in < 1.1 {
			gx := inside(px+0.3, py) - inside(px-0.3, py)
			gy := inside(px, py+0.3) - inside(px, py-0.3)
			if -gx*lightDir[0]-gy*lightDir[1] < 0 {
				l += 0.12
			} else {
				l -= 0.1
			}
		}
		// Chisel marks, in rows.
		for _, m := range chisel {
			sa, ca := math.Sincos(m[2])
			u, v := (px-m[0])*ca+(py-m[1])*sa, (py-m[1])*ca-(px-m[0])*sa
			if math.Abs(u) < 2.2 && math.Abs(v) < 0.8 {
				switch f := math.Mod(u+2.2, 0.75); {
				case f < 0.22:
					l = 0.3
				case f < 0.42:
					l += 0.12
				}
			}
		}
		// Brushed pale round the bones.
		if clean < 7 {
			l += 0.08 * (1 - clean/7)
		}
		// Its grain.
		switch h := hashf(x, y, seed+9); {
		case h < 0.05:
			l -= 0.1
		case h > 0.965:
			l += 0.1
		}
		c.put(x, y, fossilRockRamp.pick(l, x, y))
	})
}

// fossilDust scatters pale dust from the dig on the floor round the slab,
// and lays the slab's shadow on it to the right and below.
func fossilDust(c *canvas) {
	src := make([]color.RGBA, len(c.px))
	copy(src, c.px)
	solid := func(x, y int) bool { return c.in(x, y) && src[y*c.w+x].A >= 128 }
	for y := range c.h {
		for x := range c.w {
			if solid(x, y) {
				continue
			}
			if solid(x-2, y-3) || solid(x-1, y-4) {
				c.put(x, y, withAlpha(fossilInk, 70))
				continue
			}
			d := 99.0
			for dy := -6; dy <= 6; dy += 2 {
				for dx := -6; dx <= 6; dx += 2 {
					if solid(x+dx, y+dy) {
						d = min(d, math.Hypot(float64(dx), float64(dy)))
					}
				}
			}
			if a := 1 - d/7; a > 0 && fbm2(float64(x)*0.08, float64(y)*0.1, 5400, 2)+0.25*a > 0.6 {
				c.put(x, y, withAlpha(fossilRockRamp[5], uint8(60*a)))
			}
		}
	}
}

// paintingCanvas draws pictures painted long ago on a pale patch of the
// wall, in red ochre and soot, above the floor tile they stand over.
func paintingCanvas(pic int) *canvas {
	c := hdCanvas(72, 64)
	const ax, top, k = 36.0, 16.0, 1.45 // drawn k times as big as designed, about ax, top
	each := func(f func(x, y int, px, py float64)) {
		eachHD(c, func(x, y int, px, py float64) { f(x, y, ax+(px-ax)/k, top+(py-top)/k) })
	}
	// The pale patch of rock, worn smooth.
	each(func(x, y int, px, py float64) {
		dx, dy := (px-ax)/21, (py-top-13)/12
		d := dx*dx + dy*dy + (vnoise(px*0.25, py*0.3, 4600)-0.5)*0.35
		if d < 1 {
			l := 0.55 + (vnoise(px*0.5, py*0.6, 4601)-0.5)*0.15
			c.put(x, y, withAlpha(cpSand.pick(l, x, y), uint8(255*clamp01((1-d)*5)*0.75)))
		}
	})
	ochre := func(x, y int, l float64) { c.put(x, y, cpOchre.pick(l, x, y)) }
	soot := func(x, y int) { c.put(x, y, hex(0x282e3b)) }
	stroke := func(x0, y0, x1, y1, w float64, red bool) {
		each(func(x, y int, px, py float64) {
			if _, d := segParam(px, py, x0, y0, x1, y1); d <= w {
				if red {
					ochre(x, y, 0.4+0.3*hashf(x/2, y/2, 4602))
				} else {
					soot(x, y)
				}
			}
		})
	}
	blob := func(cx, cy, rx, ry float64, red bool) {
		each(func(x, y int, px, py float64) {
			if dx, dy := (px-cx)/rx, (py-cy)/ry; dx*dx+dy*dy <= 1 {
				if red {
					ochre(x, y, 0.45+0.3*hashf(x/2, y/2, 4603))
				} else {
					soot(x, y)
				}
			}
		})
	}
	man := func(x, y float64, red bool) { // a stick hunter with a spear
		blob(x, y-6, 1.1, 1.1, red)
		stroke(x, y-5, x, y-1.5, 0.45, red)
		stroke(x, y-1.5, x-1.5, y+1.5, 0.45, red)
		stroke(x, y-1.5, x+1.5, y+1.5, 0.45, red)
		stroke(x, y-4, x+2.5, y-5.5, 0.4, red)
		stroke(x+2.5, y-8, x+2.5, y+1, 0.3, red)
	}
	switch pic {
	case sim.PaintingHunt:
		blob(ax+3, top+12, 6, 3.2, true) // a horned beast
		stroke(ax+8, top+11, ax+11, top+8, 0.5, true)
		stroke(ax-1, top+14, ax-1, top+17.5, 0.5, true)
		stroke(ax+6, top+14, ax+6, top+17.5, 0.5, true)
		stroke(ax+9.5, top+10, ax+11.5, top+6.5, 0.4, false)
		man(ax-12, top+18, false)
		man(ax-7, top+19, false)
		man(ax+15, top+19, false)
	case sim.PaintingHoard:
		// A chest, an arrow, handprints round it.
		blob(ax-2, top+13, 4.5, 2.6, true)
		stroke(ax-6.5, top+11.5, ax+2.5, top+11.5, 0.35, false)
		stroke(ax+5, top+13, ax+14, top+13, 0.6, true)
		stroke(ax+14, top+13, ax+11, top+10.5, 0.6, true)
		stroke(ax+14, top+13, ax+11, top+15.5, 0.6, true)
		for _, h := range [][2]float64{{-14, 7}, {-12, 18}, {9, 5}, {15, 19}} {
			blob(ax+h[0], top+h[1], 1.6, 1.9, false)
		}
	case sim.PaintingStar:
		// A star of fire falling on an island in the sea.
		blob(ax+9, top+5, 2.2, 2.2, true)
		for k := range 4 {
			stroke(ax+9-float64(k)*2.8, top+5+float64(k)*1.2, ax+7-float64(k)*2.8, top+6.6+float64(k)*1.2, 0.5, true)
		}
		blob(ax-6, top+18, 8, 2.2, false)
		stroke(ax-14, top+20.5, ax+14, top+20.5, 0.3, false)
		man(ax-5, top+16, true)
	case sim.PaintingHerds:
		for k := range 4 {
			x := ax - 13 + float64(k)*8.5
			y := top + 12 + float64(k%2)*3
			blob(x, y, 3.4, 1.8, k%2 == 0)
			stroke(x+2.5, y-1, x+4.5, y-6, 0.55, k%2 == 0) // a long neck
			stroke(x-1.5, y+1, x-1.5, y+4, 0.45, k%2 == 0)
			stroke(x+1.5, y+1, x+1.5, y+4, 0.45, k%2 == 0)
		}
	}
	return c
}

// sarcophagusCanvas drawsgusCanvas draws the tomb's stone coffin: a long chest of
// sandstone, its lid carved with a face and bands.
func sarcophagusCanvas() *canvas {
	c := hdCanvas(52, 40)
	const x0, x1, top, lid, foot = 4.0, 48.0, 8.0, 22.0, 34.0
	eachHD(c, func(x, y int, px, py float64) {
		if px < x0 || px > x1 || py < top || py > foot {
			return
		}
		var l float64
		switch {
		case py < lid: // the lid's top, in the light
			l = 0.7 - 0.2*(py-top)/(lid-top) + (vnoise(px*0.5, py*0.6, 4700)-0.5)*0.1
			if math.Abs(px-x0-7) < 4.5 && py > top+3 && py < lid-3 { // the carved face
				l = 0.85
				if math.Hypot(px-x0-5.5, py-top+0.5-6) < 0.8 || math.Hypot(px-x0-8.5, py-top+0.5-6) < 0.8 {
					l = 0.2
				}
			}
			if math.Mod(px-x0, 9) < 0.9 && px > x0+14 {
				l -= 0.2 // a carved band
			}
		default: // its side, in shade
			l = 0.4 - 0.15*(py-lid)/(foot-lid)
			if math.Abs(py-lid-5) < 0.7 {
				l += 0.15
			}
		}
		if px < x0+1.2 || px > x1-1.2 {
			l -= 0.1
		}
		c.put(x, y, cpSand.pick(clamp01(l), x, y))
	})
	c.outline(0.65)
	return c
}

// urnCanvas draws a clay jar of the tomb, look v: a round body, a neck and
// a lip, painted with bands.
func urnCanvas(v int) *canvas {
	c := hdCanvas(24, 28)
	const ax, ay = 12.0, 24.0
	bodyR := []float64{6.5, 5.5, 7}[v%3]
	ht := []float64{18, 20, 15}[v%3]
	eachHD(c, func(x, y int, px, py float64) {
		t := (ay - py) / ht
		if t < 0 || t > 1 {
			return
		}
		var hw float64
		switch {
		case t < 0.72:
			s := t / 0.72
			hw = bodyR * math.Sin(math.Pi*(0.15+0.85*s)) // the body
		case t < 0.92:
			hw = bodyR * 0.36 // the neck
		default:
			hw = bodyR * 0.55 // the lip
		}
		dx := (px - ax) / math.Max(hw, 0.5)
		if math.Abs(dx) > 1 {
			return
		}
		l := 0.6 - 0.45*dx + 0.1*t
		if math.Abs(t-0.45) < 0.035 || math.Abs(t-0.55) < 0.02 {
			l -= 0.3 // a painted band
		}
		c.put(x, y, cpClay.pick(clamp01(l), x, y))
	})
	c.outline(0.65)
	return c
}

// rootsCanvas draws roots come down through the roof of the cave, look v:
// tangled strands hanging to the floor, rootlets off them, moss on some.
func rootsCanvas(v int) *canvas {
	c := hdCanvas(32, 70)
	const ax, ay = 16.0, 66.0
	n := 4 + v
	for k := range n {
		h := hash2(int32(k), int32(v), 4800)
		x0 := ax - 10 + float64(h&15)*1.3
		ln := 40 + float64(h>>4&31)
		ph := float64(h>>9&63) * 0.1
		for s := 0.0; s <= ln; s += 0.12 {
			y := ay - 66 + s
			x := x0 + 2.2*math.Sin(s/7+ph) + 1.2*math.Sin(s/2.7+ph*2)
			w := 0.9 - 0.5*s/ln
			for d := -w; d <= w; d += 0.33 {
				xx, yy := int((x+d)*hdK3), int(y*hdK3)
				l := 0.55 - 0.4*d/w + 0.1*math.Sin(s)
				col := cmRoot.pick(clamp01(l), xx, yy)
				if hashf(xx/3, yy/4, 4801+uint32(k)) < 0.05 {
					col = cmMoss.at(1)
				}
				c.put(xx, yy, col)
			}
			if int(s*8)%61 == 0 && s > 6 { // a rootlet
				for r := 0.0; r < 4; r += 0.2 {
					xx, yy := int((x+r*0.8)*hdK3), int((y+r*0.6)*hdK3)
					c.put(xx, yy, cmRoot.at(2))
				}
			}
		}
	}
	c.outline(0.55)
	return c
}

// iciclesCanvas draws spikes of ice rising by a wall, look v.
func iciclesCanvas(v int) *canvas {
	c := hdCanvas(32, 36)
	const ax, ay = 16.0, 34.0
	for k := range 5 {
		h := hash2(int32(k), int32(v), 4900)
		bx := ax - 10 + float64(h&15)*1.3
		ht := 9 + float64(h>>4&15)
		w := 1.6 + float64(h>>8&3)*0.4
		eachHD(c, func(x, y int, px, py float64) {
			t := (ay - py) / ht
			if t < 0 || t > 1 {
				return
			}
			hw := w * (1 - t)
			dx := (px - bx) / math.Max(hw, 0.2)
			if math.Abs(dx) > 1 {
				return
			}
			l := 0.55 - 0.35*dx + 0.3*t
			if math.Abs(dx+0.4) < 0.15 {
				l += 0.3
			}
			c.put(x, y, withAlpha(cpIce.pick(clamp01(l), x, y), 230))
		})
	}
	c.outline(0.5)
	return c
}

// obsidianCanvas draws shards of black volcanic glass, look v, with a
// purple sheen on their faces.
func obsidianCanvas(v int) *canvas {
	c := hdCanvas(28, 20)
	const ax, ay = 14.0, 16.0
	for k := range 4 {
		h := hash2(int32(k), int32(v), 5000)
		prism(c, ax-8+float64(h&15), ay, (float64(h>>4&15)-7.5)*0.09, 5+float64(h>>8&7), 1.6+float64(h>>12&3)*0.3, cpGlass)
	}
	c.outline(0.6)
	return c
}

// roostCanvas draws the floor under the bats' roost: a heap of droppings.
func roostCanvas() *canvas {
	c := hdCanvas(36, 22)
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := (px-18)/15, (py-14)/6
		if d := dx*dx + dy*dy + (vnoise(px*0.4, py*0.5, 5100)-0.5)*0.5; d < 1 {
			l := 0.5 - 0.3*dy + (vnoise(px*1.5, py*1.7, 5101)-0.5)*0.3
			c.put(x, y, withAlpha(cpStem.pick(clamp01(l)*0.8, x, y), uint8(255*clamp01((1-d)*4))))
		}
	})
	return c
}

// batCanvas draws a bat, frame f: 0 hanging with its wings folded, 1 to 3
// a wingbeat.
func batCanvas(f int) *canvas {
	c := hdCanvas(14, 10)
	const cx, cy = 7.0, 5.0
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := px-cx, py-cy
		if f == 0 { // hanging upside down: a dark drop
			if math.Hypot(dx/1.8, (dy-0.5)/3) <= 1 {
				c.put(x, y, cpBat.pick(0.55-0.3*dx/1.8, x, y))
			}
			return
		}
		if math.Hypot(dx/1.3, dy/1.6) <= 1 {
			c.put(x, y, cpBat.pick(0.6-0.25*dx, x, y)) // the body
			return
		}
		lift := []float64{0, -2.6, 0.2, 2.2}[f]
		ax := math.Abs(dx)
		if ax < 6.5 {
			wy := lift * ax / 6.5
			if dy > wy-1.2+0.25*math.Sin(ax*1.5) && dy < wy+0.8-0.3*ax/6.5 {
				c.put(x, y, cpBat.pick(0.35+0.25*(1-ax/6.5), x, y)) // a wing, the light through it near the body
			}
		}
	})
	c.outline(0.5)
	return c
}

// pillarCanvas draws a carved pillar of the tomb: a column of sandstone on
// a plinth, under a capital, worn and cracked.
func pillarCanvas() *canvas {
	c := hdCanvas(32, 74)
	const ax, ay = 16.0, 70.0
	eachHD(c, func(x, y int, px, py float64) {
		h := ay - py
		if h < 0 || h > 66 {
			return
		}
		hw := 6.0
		switch {
		case h < 4:
			hw = 9 // the plinth
		case h > 60:
			hw = 9.5 - (h-60)*0.2 // the capital
		}
		dx := (px - ax) / hw
		if math.Abs(dx) > 1 {
			return
		}
		l := 0.62 - 0.4*dx
		if h >= 4 && h <= 60 && math.Mod(px-ax+12, 3) < 0.5 {
			l -= 0.12 // its fluting
		}
		if math.Abs(h-4) < 0.6 || math.Abs(h-60) < 0.6 {
			l -= 0.25
		}
		if h > 20 && h < 40 && math.Abs(px-ax-2-0.4*math.Sin(h/3)) < 0.5 {
			l -= 0.3 // a crack
		}
		c.put(x, y, cpSand.pick(clamp01(l), x, y))
	})
	c.outline(0.65)
	return c
}

// iceSpireCanvas draws a spire of ice taller than a man, with a few
// smaller ones round its foot.
func iceSpireCanvas() *canvas {
	c := hdCanvas(48, 66)
	const ax, ay = 24.0, 62.0
	for _, s := range [][4]float64{{-9, 1, 22, 4}, {0, -1, 56, 7}, {9, 1.5, 30, 4.6}, {-4, 2.5, 14, 3}} {
		bx, by, ht, w := ax+s[0], ay+s[1], s[2], s[3]
		eachHD(c, func(x, y int, px, py float64) {
			t := (by - py) / ht
			if t < 0 || t > 1 {
				return
			}
			hw := w * math.Pow(1-t, 0.85)
			dx := (px - bx) / math.Max(hw, 0.2)
			if math.Abs(dx) > 1 {
				return
			}
			l := 0.5 - 0.3*dx + 0.25*t
			switch {
			case math.Abs(dx+0.45) < 0.12:
				l += 0.35 // a long highlight
			case dx > 0.5:
				l -= 0.12
			}
			if math.Mod(py*0.7+dx*3, 5) < 0.4 {
				l += 0.08 // the grain of the ice
			}
			c.put(x, y, withAlpha(cpIce.pick(clamp01(l), x, y), 225))
		})
	}
	c.outline(0.55)
	return c
}

// basaltCanvas draws a clump of basalt columns, six-sided, standing at
// different heights, their tops flat and pale.
func basaltCanvas() *canvas {
	c := hdCanvas(40, 62)
	const ax, ay = 20.0, 58.0
	for _, s := range [][3]float64{{-8, 32, 4.2}, {8, 38, 4.4}, {0, 50, 5}, {-3, 24, 4}, {5, 20, 3.8}} {
		bx, ht, w := ax+s[0], s[1], s[2]
		top := ay - ht
		eachHD(c, func(x, y int, px, py float64) {
			dx := (px - bx) / w
			if math.Abs(dx) > 1 || py > ay+1-math.Abs(dx)*1.2 {
				return
			}
			capY := top + math.Abs(dx)*1.6 - 1.6
			switch {
			case py < capY-2.6:
				return
			case py < capY: // the flat six-sided top, in the light
				c.put(x, y, cpBasalt.pick(0.85-0.2*math.Abs(dx), x, y))
			default:
				l := 0.45
				switch {
				case dx < -0.33:
					l = 0.62
				case dx > 0.33:
					l = 0.22
				}
				if math.Mod(py-top, 9) < 0.5 {
					l -= 0.15 // a joint
				}
				c.put(x, y, cpBasalt.pick(l, x, y))
			}
		})
	}
	c.outline(0.6)
	return c
}
