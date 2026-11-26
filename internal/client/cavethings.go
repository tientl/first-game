package client

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The things standing in the natural caves (see cavebake.go): the hole
// down to the deep caves with its rope, the rope back up hanging under its
// shaft, and clusters of stalagmites in the rock of the cave they grow in.
// Drawn at HD the first time one is wanted.

type caveThingArt struct {
	down, up sprite
	stalag   [sim.NumCaveRocks][stalagLooks]sprite
}

const stalagLooks = 4

var caveThingsOnce struct {
	sync.Once
	art caveThingArt
}

func caveThings() *caveThingArt {
	caveThingsOnce.Do(func() {
		a := &caveThingsOnce.art
		a.down = spriteHD(caveDownCanvas().image(), caveDownAX, caveDownAY)
		a.up = spriteHD(caveUpCanvas().image(), caveUpAX, caveUpAY)
		for r := range a.stalag {
			for v := range a.stalag[r] {
				c := stalagCanvas(sim.CaveRock(r), v)
				img := c.image()
				noteCover(img, c)
				a.stalag[r][v] = spriteHD(img, stalagW/2, stalagAY)
			}
		}
	})
	return &caveThingsOnce.art
}

// Sizes and anchors, in art pixels.
const (
	caveDownW, caveDownH   = 62, 42
	caveDownAX, caveDownAY = 31.0, 21.0
	caveUpW, caveUpH       = 44, 118
	caveUpAX, caveUpAY     = 22.0, 104.0
	stalagW, stalagH       = 46, 66
	stalagAY               = 58.0
)

// stalagRamps are the stone of stalagmites in each rock of the caves.
var stalagRamps = [sim.NumCaveRocks]ramp{
	sim.RockLimestone: mkRamp(0x2b222a, 0x3f333b, 0x6d4e4b, 0x867066, 0xb49d7e, 0xc4c6b8, 0xe5efef),
	sim.RockMossy:     mkRamp(0x1a1f2e, 0x282e3b, 0x414453, 0x4d6659, 0x696f80, 0x8d99a4, 0xbac9cd),
	sim.RockSandstone: mkRamp(0x3c252b, 0x5c2b34, 0x8d4830, 0xb77862, 0xc57835, 0xe1a845, 0xeecb90),
	sim.RockBasalt:    mkRamp(0x0b0d14, 0x1a1f2e, 0x282e3b, 0x414453, 0x5c3a6f, 0x7e659b, 0xb8a1c2),
	sim.RockIce:       mkRamp(0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xe5efef, 0xffffff),
	sim.RockMud:       mkRamp(0x0d1018, 0x1a1f2e, 0x2b222a, 0x3f333b, 0x3f4328, 0x5f7132, 0x739a70),
}

// stalagCanvas draws a cluster of stalagmites of rock r, look v: two to
// four cones of stone rising from a mound of flowstone, lit from the left,
// ringed where they grew, glistening down one side.
func stalagCanvas(r sim.CaveRock, v int) *canvas {
	rp := stalagRamps[r]
	c := hdCanvas(stalagW, stalagH)
	const ax, ay = stalagW / 2, stalagAY
	seed := uint32(4100 + int(r)*37 + v*11)
	type spire struct{ x, y, h, r float64 }
	layouts := [stalagLooks][]spire{
		{{-6, -2, 42, 7.5}, {5, -1, 30, 6}, {12, 1.5, 15, 4}, {-13, 1.5, 11, 3.5}},
		{{1, -2, 50, 8}, {-9, 0.5, 24, 5}, {10, 1, 19, 4.5}},
		{{-4, -1.5, 34, 6.5}, {7, -2, 43, 7}, {-12, 1.5, 14, 4}, {1, 2.5, 10, 3.5}},
		{{0, -1, 27, 8.5}, {-10, 1, 18, 4.5}, {9, 0, 36, 6}},
	}
	// The mound of flowstone they stand on.
	eachHD(c, func(x, y int, px, py float64) {
		dx, dy := (px-ax)/19, (py-ay)/5.6
		d := dx*dx + dy*dy + (vnoise(px*0.4, py*0.4, seed)-0.5)*0.3
		if d > 1 {
			return
		}
		l := 0.5 - 0.35*dy - 0.2*dx + (vnoise(px*0.9, py*1.3, seed+1)-0.5)*0.2
		if d > 0.75 {
			l -= 0.15
		}
		c.put(x, y, rp.pick(l*0.85, x, y))
	})
	sp := layouts[v]
	// Back to front: the farthest (highest base) first.
	order := []int{0, 1, 2, 3}[:len(sp)]
	for i := range order {
		for j := i + 1; j < len(order); j++ {
			if sp[order[j]].y < sp[order[i]].y {
				order[i], order[j] = order[j], order[i]
			}
		}
	}
	for _, k := range order {
		s := sp[k]
		bx, by := ax+s.x, ay+s.y
		ph := float64(k) * 1.7
		eachHD(c, func(x, y int, px, py float64) {
			t := (by - py) / s.h // 0 at the foot, 1 at the tip
			if t < -0.15 || t > 1 {
				return
			}
			w := s.r * math.Pow(clamp01(1-t), 0.72) * (1 + 0.07*math.Sin(t*13+ph))
			if t < 0 {
				w = s.r * (1 + 0.6*(-t)/0.15) // a flared foot
			}
			if t > 0.9 {
				w = math.Max(w, s.r*0.12*(1-(t-0.9)/0.1))
			}
			dx := (px - bx) / math.Max(0.3, w)
			if math.Abs(dx) > 1 {
				return
			}
			z := math.Sqrt(1 - dx*dx)
			l := 0.42 - 0.4*dx + 0.22*z + 0.25*t
			// Rings where it grew, and a wet streak down its lit side.
			if ring := math.Mod(py+2*vnoise(px*0.3, 0, seed+2)+ph*3, 3.4); ring < 0.7 {
				l -= 0.12
			}
			if math.Abs(dx+0.42) < 0.09 && vnoise(px*0.2, py*0.15, seed+3) > 0.35 {
				l += 0.28
			}
			if t < 0.15 {
				l -= 0.25 * (0.15 - t) / 0.15
			}
			col := rp.pick(clamp01(l), x, y)
			switch {
			case r == sim.RockMossy && t < 0.35 && dx < 0.2 && vnoise(px*0.7, py*0.8, seed+4) > 0.58:
				col = cmMoss.pick(clamp01(0.2+0.6*l), x, y)
			case r == sim.RockSandstone && math.Mod(py+ph*5, 7) < 2.2:
				col = rp.pick(clamp01(l+0.12), x, y) // a paler band
			case r == sim.RockIce:
				col = withAlpha(col, 225)
			}
			c.put(x, y, col)
		})
	}
	c.outline(0.75)
	return c
}

var (
	caveRopeRamp = mkRamp(0x3c252b, 0x684039, 0x825646, 0xb77862, 0xeecb90)
	stakeRamp    = mkRamp(0x2b222a, 0x533b41, 0x6d4e4b, 0x867066)
)

// caveDownCanvas draws the hole down to the deep caves: a pit in the floor
// ringed with fallen stones, its far wall dropping away into the dark, and
// a stake driven in by its lip with a rope tied to it that goes down over
// the edge.
func caveDownCanvas() *canvas {
	const ax, ay = caveDownAX, caveDownAY
	const seed = 4200
	c := hdCanvas(caveDownW, caveDownH)
	const rx, ry = 20.0, 11.0
	inHole := func(px, py float64) float64 {
		dx, dy := (px-ax)/rx, (py-ay)/ry
		return dx*dx + dy*dy + (vnoise(px*0.35, py*0.5, seed)-0.5)*0.18
	}
	eachHD(c, func(x, y int, px, py float64) {
		d := inHole(px, py)
		switch {
		case d < 1:
			// The far wall of the shaft, lit at its lip and lost below; the
			// near wall cannot be seen: black.
			top := ay - ry*math.Sqrt(math.Max(0, 1-((px-ax)/rx)*((px-ax)/rx)))
			down := (py - top) / (2 * ry)
			l := 0.55 - 1.4*down + (vnoise(px*0.5, py*0.8, seed+1)-0.5)*0.2
			if l < 0.05 {
				c.put(x, y, hex(0x05040a))
				return
			}
			c.put(x, y, caveMats[sim.RockLimestone].face.pick(l*0.8, x, y))
		case d < 1.45:
			// The lip: trodden earth, broken towards the hole.
			l := 0.35 + (d-1)*0.6 + (vnoise(px*0.7, py*0.9, seed+2)-0.5)*0.25
			c.put(x, y, withAlpha(caveDirtRamp.pick(l, x, y), uint8(255*clamp01((1.45-d)/0.12))))
		}
	})
	// Stones fallen round the lip.
	for i := range 15 {
		a := float64(i)/15*2*math.Pi + hashf(i, 0, seed+3)*0.4
		if a > 4.9 && a < 5.9 {
			continue // where the rope goes over
		}
		x, y := ax+math.Cos(a)*(rx+1.8), ay+math.Sin(a)*(ry+1.4)
		s := 1.6 + hashf(i, 1, seed+4)*1.6
		stoneHD(c, x*hdK3, y*hdK3, s*hdK3, s*0.72*hdK3, 0.6, paleRockRamp, uint32(seed+10+i))
	}
	// The stake, and the rope tied to it going over the lip and down.
	const sx, sy = ax + 21.5, ay - 6
	eachHD(c, func(x, y int, px, py float64) {
		if math.Abs(px-sx) < 1.3 && py > sy-9 && py < sy+0.5 {
			l := 0.6 - 0.5*(px-sx)/1.3
			if py < sy-8 {
				l += 0.2 // its cut top
			}
			c.put(x, y, stakeRamp.pick(l, x, y))
		}
	})
	rope := func(x0, y0, x1, y1, sag float64) {
		for s := 0.0; s <= 1; s += 0.004 {
			px := x0 + (x1-x0)*s
			py := y0 + (y1-y0)*s + sag*4*s*(1-s)
			for k := -0.55; k <= 0.55; k += 0.25 {
				x, y := int(px*hdK3), int((py+k)*hdK3)
				tw := math.Mod(s*60+k*2, 2)
				c.put(x, y, caveRopeRamp.pick(0.55-0.6*k+0.15*tw, x, y))
			}
		}
	}
	rope(sx-0.5, sy-6.5, ax+9, ay-ry+1.5, 1.2) // to the lip
	rope(ax+9, ay-ry+1.5, ax+7.5, ay+3.5, 0)   // and down the far wall into the dark
	for i := 0; i < 3; i++ {                   // the turns round the stake
		y := sy - 6.6 + float64(i)*0.9
		for px := sx - 1.4; px <= sx+1.4; px += 0.1 {
			x, yy := int(px*hdK3), int(y*hdK3)
			c.put(x, yy, caveRopeRamp.pick(0.7-0.25*(px-sx), x, yy))
		}
	}
	c.outline(0.7)
	// Fade the rope into the dark of the shaft.
	eachHD(c, func(x, y int, px, py float64) {
		if inHole(px, py) < 1 && py > ay-2 {
			p := c.at(x, y)
			if p.A > 0 {
				c.put(x, y, mix(p, hex(0x05040a), clamp01((py-ay+2)/5)))
			}
		}
	})
	return c
}

// caveUpCanvas draws the rope back up: a knotted rope hanging from the hole
// in the roof far above, a faint shaft of light coming down round it, a
// coil of it on the floor among the stones fallen from up there.
func caveUpCanvas() *canvas {
	const ax, ay = caveUpAX, caveUpAY
	const seed = 4300
	c := hdCanvas(caveUpW, caveUpH)
	// The light coming down: a soft beam, widening to a pool on the floor.
	eachHD(c, func(x, y int, px, py float64) {
		t := py / ay
		hw := 5 + 9*t
		a := 0.0
		if py <= ay {
			a = clamp01(1-math.Abs(px-ax)/hw) * (0.18 + 0.12*t)
			a *= 0.75 + 0.25*vnoise(px*0.15, py*0.05, seed)
		}
		dx, dy := (px-ax)/17, (py-ay)/6.5
		if d := dx*dx + dy*dy; d < 1 {
			a = math.Max(a, 0.32*(1-d))
		}
		if a > 0.01 {
			c.put(x, y, withAlpha(hex(0xd8e8f0), uint8(255*a)))
		}
	})
	f := hdCanvas(caveUpW, caveUpH)
	// Stones fallen from above.
	for i, s := range [][3]float64{{-9, 2, 2.6}, {8.5, 1, 2.2}, {-4.5, 4.5, 1.8}, {11, 4, 1.6}, {-12.5, -1, 1.5}, {5, -2.5, 1.4}} {
		stoneHD(f, (ax+s[0])*hdK3, (ay+s[1])*hdK3, s[2]*hdK3, s[2]*0.7*hdK3, 0.55, paleRockRamp, uint32(seed+10+i))
	}
	// The coil on the floor.
	for k := 0; k < 3; k++ {
		rx, ry := 5.5-float64(k)*1.1, 2.6-float64(k)*0.45
		cy := ay + 1 - float64(k)*1.1
		for a := 0.0; a < 2*math.Pi; a += 0.01 {
			px, py := ax+1.5+math.Cos(a)*rx, cy+math.Sin(a)*ry
			for w := -0.5; w <= 0.5; w += 0.25 {
				x, y := int(px*hdK3), int((py+w)*hdK3)
				l := 0.5 - 0.35*math.Cos(a+0.8) - 0.4*w
				f.put(x, y, caveRopeRamp.pick(l, x, y))
			}
		}
	}
	// The rope itself, swinging a little, knotted for the hands.
	ropeX := func(py float64) float64 { return ax + 1.2*math.Sin(py*0.05+0.6) - 0.6 }
	for py := 0.0; py <= ay-1; py += 0.05 {
		x0 := ropeX(py)
		knot := math.Mod(py, 9) < 1.4
		hw := 0.6
		if knot {
			hw = 1.15
		}
		for w := -hw; w <= hw; w += 0.2 {
			x, y := int((x0+w)*hdK3), int(py*hdK3)
			tw := math.Mod(py*2.2+w*3, 2)
			l := 0.62 - 0.45*w/hw + 0.12*tw
			if knot {
				l += 0.08
			}
			// Lost in the dark higher up.
			col := caveRopeRamp.pick(clamp01(l), x, y)
			if py < 30 {
				col = mix(col, hex(0x05040a), (30-py)/30*0.85)
			}
			f.put(x, y, col)
		}
	}
	f.outline(0.7)
	for i, p := range f.px {
		if p.A >= 128 {
			c.px[i] = p
		}
	}
	return c
}

// caveCellAt is the cave cell of tile (tx, ty), zero off the natural caves.
func (g *Game) caveCellAt(tx, ty int) sim.CaveCell {
	if g.caveCells == nil || tx < 0 || tx >= g.width || ty < g.surfaceH || ty >= g.height {
		return 0
	}
	return g.caveCells[(ty-g.surfaceH)*g.width+tx]
}

// drawCaveThing draws a thing of the natural caves in the sorted pass, and
// reports whether e was one.
func (g *Game) drawCaveThing(dst drawTarget, e protocol.EntityView, d drawable) bool {
	var s sprite
	switch e.Kind {
	case sim.EntityCaveDown:
		s = caveThings().down
	case sim.EntityCaveUp:
		s = caveThings().up
	case sim.EntityStalagmite:
		s = caveThings().stalag[g.caveCellAt(int(e.TX), int(e.TY)).Rock()%sim.NumCaveRocks][int(e.Var)%stalagLooks]
	case sim.EntityCaveProp:
		return g.drawCaveProp(dst, e, d) // see caveprops.go
	default:
		return false
	}
	if solid := g.seeThrough(e, s, d); solid < 1 { // fading while it hides someone; see occlude.go
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.ax, -s.ay)
		op.GeoM.Translate(d.sx, d.sy)
		op.ColorScale.ScaleAlpha(solid)
		drawSprite(dst, s, op)
		return true
	}
	drawAt(dst, s, d.sx, d.sy, 1, 1, 0, nil)
	return true
}

// caveShadow is shadowWidth for the things of the caves.
func caveShadow(e protocol.EntityView) (float64, bool) {
	switch e.Kind {
	case sim.EntityCaveDown, sim.EntityCaveUp:
		return 0, true
	case sim.EntityStalagmite:
		return 26, true
	case sim.EntityCaveProp:
		return 0, true
	}
	return 0, false
}
