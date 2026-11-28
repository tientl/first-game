package client

import (
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
		a.props[sim.PropFossil] = []sprite{mk(fossilCanvas(), 40, 26)}
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
	cpBone   = mkRamp(0x533b41, 0x867066, 0xb49d7e, 0xc4c6b8, 0xe3e6ac)
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

// caveBone draws a bone from (x0, y0) to (x1, y1), w thick, with knobs at its
// ends.
func caveBone(c *canvas, x0, y0, x1, y1, w float64) {
	eachHD(c, func(x, y int, px, py float64) {
		t, d := segParam(px, py, x0, y0, x1, y1)
		r := w
		if t < 0.12 || t > 0.88 {
			r = w * 1.6
		}
		if d > r {
			return
		}
		l := 0.62 - 0.4*(py-math.Min(y0, y1))/math.Max(1, math.Abs(y1-y0)+2*w) - 0.15*d/r
		c.put(x, y, cpBone.pick(clamp01(l), x, y))
	})
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

// bonesCanvas draws bones of a beast's meal strewn on the floor, look v.
func bonesCanvas(v int) *canvas {
	c := hdCanvas(36, 22)
	sets := [][][5]float64{
		{{6, 12, 20, 9, 1.1}, {14, 15, 27, 17, 1}, {22, 6, 30, 12, 0.9}},
		{{5, 8, 17, 14, 1.2}, {19, 9, 31, 7, 1}, {10, 16, 22, 18, 0.9}},
		{{8, 15, 28, 11, 1.2}, {12, 7, 16, 16, 0.8}, {24, 15, 31, 18, 0.8}},
	}
	for _, b := range sets[v%len(sets)] {
		caveBone(c, b[0], b[1], b[2], b[3], b[4])
	}
	if v == 0 { // and a skull
		eachHD(c, func(x, y int, px, py float64) {
			dx, dy := (px-27)/4.5, (py-15)/3.4
			if d := dx*dx + dy*dy; d <= 1 {
				l := 0.65 - 0.3*dx - 0.3*dy
				if math.Hypot(px-26, py-14.5) < 1.1 {
					l = 0.05 // its eye
				}
				c.put(x, y, cpBone.pick(clamp01(l), x, y))
			}
		})
	}
	c.outline(0.55)
	return c
}

// fossilCanvas draws a great skeleton half bared in a slab of the floor:
// skull with its teeth, a curving spine, ribs, legs and a long tail.
func fossilCanvas() *canvas {
	c := hdCanvas(80, 44)
	// The slab of rock it lies in.
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := (px-40)/37, (py-25)/14
		if d := dx*dx + dy*dy + (vnoise(px*0.2, py*0.3, 4500)-0.5)*0.4; d < 1 {
			l := 0.35 - 0.25*dy + (vnoise(px*0.6, py*0.8, 4501)-0.5)*0.15
			if d > 0.85 {
				l -= 0.12
			}
			c.put(x, y, cpSand.pick(clamp01(l)*0.6, x, y))
		}
	})
	// The spine, from the neck to the tail's tip.
	spine := func(t float64) (float64, float64) {
		x := 18 + t*58
		return x, 22 + 5*math.Sin(t*3.6-0.6) - 3*t
	}
	for t := 0.0; t <= 1; t += 0.035 {
		x, y := spine(t)
		r := 2.2 * (1 - 0.75*t)
		eachHD(c, func(xx, yy int, px, py float64) {
			if d := math.Hypot((px-x)/r, (py-y)/(r*0.8)); d <= 1 {
				c.put(xx, yy, cpBone.pick(0.7-0.35*d-0.2*(py-y)/r, xx, yy))
			}
		})
	}
	// Ribs, arching from the back.
	for k := range 8 {
		t := 0.08 + float64(k)*0.04
		x, y := spine(t)
		for s := 0.0; s <= 1; s += 0.03 {
			rx := x + 2*s - 1
			ry := y + 2 + s*9*(1-0.25*float64(k%2))
			rx += 3 * math.Sin(s*2.4)
			xx, yy := int(rx*hdK3), int(ry*hdK3)
			for d := 0; d < 3; d++ {
				c.put(xx+d, yy, cpBone.pick(0.65-0.3*s, xx+d, yy))
			}
		}
	}
	// Legs.
	for _, l := range [][4]float64{{26, 25, 22, 37}, {30, 26, 33, 38}, {50, 25, 46, 36}, {54, 24, 58, 35}} {
		caveBone(c, l[0], l[1], l[2], l[3], 1.1)
	}
	// The skull, with its teeth and the hole of its eye.
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := (px-11)/8, (py-21)/4.5
		if d := dx*dx + dy*dy; d <= 1 && !(py > 22 && px < 9) {
			l := 0.66 - 0.3*dx - 0.3*dy
			if math.Hypot(px-13, py-19.5) < 1.6 {
				l = 0.05
			}
			c.put(x, y, cpBone.pick(clamp01(l), x, y))
		}
		if py > 23.4 && py < 25 && px > 4 && px < 16 && math.Mod(px, 1.6) < 0.8 {
			c.put(x, y, cpBone.pick(0.85, x, y)) // the teeth
		}
	})
	c.outline(0.5)
	return c
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
