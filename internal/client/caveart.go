package client

import (
	"image/color"
	"math"
)

// The caves under the islands are entered through a cave mouth: a low, lumpy
// arch of pale boulders lying on the ground around a hole that goes down
// into the dark, with dug-out earth spilling from it. Inside, the way back
// up is a rope ladder hanging from a hole in the rock ceiling, in a shaft of
// daylight.

// caveArt holds the cave mouth, which stands on the ground above the caves,
// and the way back up inside them.
type caveArt struct {
	mouth sprite // the whole rocky cave mouth with its dirt spill and loose stones, opening facing the viewer (south)
	exit  sprite // inside the cave: the way up, a rope ladder under a hole in the ceiling, in a shaft of daylight
}

var (
	// Pale grey-beige limestone: the boulders of the cave mouth.
	paleRockRamp = mkRamp(0x453d44, 0x625a5c, 0x857b75, 0xa59b8e, 0xc0b7a7, 0xd8d0c0, 0xece6d8, 0xf8f4ea)
	// The rock wall inside the mouth, in deep blue-grey shadow.
	caveWallRamp = mkRamp(0x0e0d14, 0x191a26, 0x262a3a, 0x363d52, 0x4b556c, 0x646f86)
	// Dug earth, from the black of the pit to the ochre crumbs in the sun.
	caveDirtRamp = mkRamp(0x1c120c, 0x33211a, 0x4f3420, 0x6f4b29, 0x8d6532, 0xa8813f, 0xc09a52)
	// The rock of the cave ceiling, seen from inside by daylight.
	ceilRockRamp = mkRamp(0x1c181d, 0x2c262b, 0x40383b, 0x5a5050, 0x7a6f69, 0x9c9184, 0xc0b6a2)
	// Daylight coming down the shaft.
	dayRamp = mkRamp(0xe8c878, 0xf6dc94, 0xfff0b8, 0xfffbe2)
)

// Anchors of the two sprites, in canvas pixels.
const (
	caveMouthAX, caveMouthAY = 52.0, 51.0
	caveExitAX, caveExitAY   = 28.0, 66.0
)

func genCaveArt() caveArt {
	return caveArt{
		mouth: mk(caveMouthCanvas(), caveMouthAX, caveMouthAY),
		exit:  mk(caveExitCanvas(), caveExitAX, caveExitAY),
	}
}

// caveRock is one boulder of a pile: an ellipse with a lumpy edge. A flat
// rock is cut level underneath, to sit on the ground.
type caveRock struct {
	x, y, rx, ry float64
	flat         bool
}

// drawCaveRock draws a chunky faceted boulder lit from the top left, with a
// shadowed lower-right rim, and darkens the stones it overlaps along its
// edge, so that a pile reads as separate rocks with crevices between them.
func drawCaveRock(c *canvas, r caveRock, rp ramp, bias float64, seed uint32) {
	// The stone is an ellipse with its corners chopped off by a few planes;
	// each cut is a facet, and what is left in the middle is the flat top.
	type facet struct{ x, y, off float64 }
	var facets [7]facet
	for i := range facets {
		a := (float64(i) + hashf(i, 1, seed)*0.8) / float64(len(facets)) * 2 * math.Pi
		facets[i] = facet{math.Cos(a), math.Sin(a), 0.6 + 0.3*hashf(i, 2, seed)}
	}
	x0, x1 := int(math.Floor(r.x-r.rx))-1, int(math.Ceil(r.x+r.rx))+1
	y0, y1 := int(math.Floor(r.y-r.ry))-1, int(math.Ceil(r.y+r.ry))+1
	w, h := x1-x0+1, y1-y0+1
	in := make([]bool, w*h)
	cols := make([]color.RGBA, w*h)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			nx, ny := (float64(x)+0.5-r.x)/r.rx, (float64(y)+0.5-r.y)/r.ry
			if nx*nx+ny*ny > 1 || (r.flat && ny > 0.62) {
				continue
			}
			// edge is 1 on the nearest cut and 0 at the middle of the stone.
			best, edge := facets[0], -1.0
			for _, f := range facets {
				if e := (nx*f.x + ny*f.y) / f.off; e > edge {
					best, edge = f, e
				}
			}
			if edge > 1 {
				continue
			}
			var l float64
			if edge < 0.5 {
				l = lum(0.3*nx-0.12, 0.3*ny-0.3) // the top
			} else {
				l = lum(0.25*nx+0.7*best.x, 0.25*ny+0.7*best.y)
			}
			l += (vnoise(float64(x)*0.45, float64(y)*0.45, seed+9)-0.5)*0.14 + bias
			// The rim: shadow under and to the right.
			if edge > 0.8 && best.x*0.6+best.y*0.8 > 0.1 {
				l -= 0.18
			}
			if r.flat && ny > 0.35 {
				l -= 0.15
			}
			in[(y-y0)*w+x-x0] = true
			cols[(y-y0)*w+x-x0] = rp.pick(l, x, y)
		}
	}
	has := func(x, y int) bool {
		x, y = x-x0, y-y0
		return x >= 0 && y >= 0 && x < w && y < h && in[y*w+x]
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if has(x, y) {
				c.put(x, y, cols[(y-y0)*w+x-x0])
				continue
			}
			// A crevice where this rock rests on the ones behind it.
			if c.opaque(x, y) && (has(x+1, y) || has(x-1, y) || has(x, y+1) || has(x, y-1)) {
				c.put(x, y, mix(c.at(x, y), rp[0], 0.55))
			}
		}
	}
}

// caveMouthCanvas draws the cave mouth. The anchor (caveMouthAX,
// caveMouthAY) is the middle of the opening at ground level.
func caveMouthCanvas() *canvas {
	const ax, ay = caveMouthAX, caveMouthAY
	rocks := newCanvas(100, 78)

	// The rocks behind and beside the hole: a long heap trailing off to the
	// left, the back of the arch, and a shorter shoulder on the right.
	back := []caveRock{
		{26, 13, 6, 4.5, true}, // a loose boulder lying behind
		{10.5, 48, 5, 3.5, true},
		{17, 45, 7, 5.5, true},
		{25, 41, 8, 7.5, false},
		{32, 35, 8, 8, false},
		{41, 27, 9, 7.5, false},
		{53, 23.5, 11, 7.5, false},
		{65, 28, 9, 7.5, false},
		{73, 37, 8, 8, false},
		{79, 46.5, 6.5, 5.5, true},
		{21, 49, 6, 4, true},
		{93, 37, 5, 4.5, true}, // a loose boulder off to the right
	}
	for i, r := range back {
		drawCaveRock(rocks, r, paleRockRamp, -0.15, uint32(900+i))
	}

	// The hole: the back wall in blue shadow, lit a little on the left, and
	// in the floor a pit of earth falling away into black.
	const hx, hy, hrx, hry = ax, 39.0, 15.5, 13.0
	for y := 24; y <= 56; y++ {
		for x := 34; x <= 70; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			u := (px - hx) / hrx
			if math.Abs(u) > 1 {
				continue
			}
			if py < hy {
				if v := (py - hy) / hry; u*u+v*v > 1 {
					continue
				}
			} else if py > ay+2.5*(1-u*u) {
				continue
			}
			// q is the distance from the middle of the pit.
			q := math.Hypot((px-(ax+2))/11.5, (py-43.5)/7)
			var col color.RGBA
			switch {
			case q < 0.5:
				col = holeRamp.pick(0.1+0.5*q, x, y)
			case py > 43.5+2.5*u:
				// The earth floor, sloping down from the lip into the pit.
				t := clamp01((q - 0.5) / 0.75)
				l := 0.08 + 0.62*t + (vnoise(px*0.5, py*0.9, 921)-0.5)*0.2
				if hashf(x/2, y, 922) < 0.18 {
					l -= 0.12 // ledges stepping down
				}
				col = caveDirtRamp.pick(l, x, y)
			default:
				// The rock wall at the back, streaked top to bottom.
				t := clamp01((q - 0.5) / 0.9)
				l := 0.08 + t*(0.62-0.5*u) + (hashf(x, y/5, 923)-0.5)*0.16
				if py < 32 {
					l -= (32 - py) * 0.04 // under the lintel
				}
				col = caveWallRamp.pick(l, x, y)
			}
			rocks.put(x, y, col)
		}
	}

	// The rim of the arch, overhanging the hole: two pillars of stacked
	// boulders, a lintel, and the feet.
	rim := []caveRock{
		{41.5, 27.5, 7, 5.5, false},
		{62.5, 27.5, 7, 5.5, false},
		{52, 24.5, 8.5, 5, false},
		{34, 36, 6, 6.5, false},
		{70, 37, 5.5, 6.5, false},
		{35, 45.5, 6, 7, true},
		{70.5, 46.5, 6, 7, true},
		{30.5, 51, 5, 3.6, true},
		{75.5, 52, 5, 3.6, true},
	}
	for i, r := range rim {
		drawCaveRock(rocks, r, paleRockRamp, -0.08, uint32(940+i))
	}
	// Small stones strung out from the right foot, hemming in the spill.
	for i, r := range []caveRock{
		{74, 58, 3.8, 3, true}, {68.5, 62, 3.6, 2.8, true}, {62, 65.5, 3.6, 2.6, true},
		{55, 68.5, 3.2, 2.4, true}, {48.5, 71, 2.8, 2.1, true}, {42.5, 73, 2.2, 1.7, true},
		{6, 51.5, 2.6, 2, true},
	} {
		drawCaveRock(rocks, r, paleRockRamp, -0.08, uint32(960+i))
	}
	rocks.outline(0.75)

	// The dirt spill, streaking out of the mouth to the south-west. It lies
	// flat on the ground and has no outline; it breaks up into crumbs.
	c := newCanvas(rocks.w, rocks.h)
	const dx, dy = -0.93, 0.37
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			px, py := float64(x)+0.5-(ax+4), float64(y)+0.5-(ay+1)
			u := (px*dx + py*dy) / 52
			v := (-px*dy + py*dx) / (6 + 6*u)
			if u < -0.08 || u > 1 || math.Abs(v) > 1 {
				continue
			}
			// Long streaks along the flow, thinning with distance.
			n := vnoise(u*2.6, v*6, 931)*0.65 + vnoise(u*8, v*11, 932)*0.35
			cover := 1.12 - 0.85*u - 0.5*v*v
			if u > 0.3 {
				cover -= 0.1 * (u - 0.3) / 0.7
			}
			if n > cover {
				continue
			}
			l := 0.72 - 0.3*n + 0.25*u + (hashf(x, y, 933)-0.5)*0.2
			if cover-n < 0.08 {
				l -= 0.2 // the damp edge of a streak
			}
			c.put(x, y, caveDirtRamp.pick(l, x, y))
		}
	}
	c.over(rocks, 0, 0)
	// The earth runs out over the lip of the hole without a break.
	c.ellipse(ax-1.5, ay+2.6, 11.5, 2.4, func(x, y int, nx, ny float64) color.RGBA {
		l := 0.62 + 0.2*ny + (vnoise(float64(x)*0.6, float64(y)*0.9, 934)-0.5)*0.3
		return caveDirtRamp.pick(l, x, y)
	})
	return c
}

// caveExitCanvas draws the way up. The anchor (caveExitAX, caveExitAY) is
// the middle of the rubble under the ladder.
func caveExitCanvas() *canvas {
	const ax, ay = caveExitAX, caveExitAY
	const holeY = 10.0
	c := newCanvas(56, 80)

	// The pool of daylight on the floor.
	c.ellipse(ax, ay+1.5, 21, 8.5, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		if e := 0.55 + 0.45*vnoise(float64(x)*0.35, float64(y)*0.5, 971); d > e || (e-d)*5 < dither(x, y) {
			return color.RGBA{}
		}
		l := 0.95 - 0.75*d + (vnoise(float64(x)*0.5, float64(y)*0.7, 972)-0.5)*0.25
		return mix(ceilRockRamp.pick(0.3+0.6*l, x, y), dayRamp[0], 0.18+0.2*l)
	})
	// The shaft of light, widening on its way down.
	for y := int(holeY) + 2; y < int(ay)+6; y++ {
		t := (float64(y) - holeY) / (ay + 6 - holeY)
		half := lerpf(9.5, 17.5, t)
		for x := 0; x < c.w; x++ {
			s := math.Abs(float64(x)+0.5-ax) / half
			if s > 1 {
				continue
			}
			a := (0.78 - 0.3*t) * (1 - 0.8*s*s)
			// Dusty rays, fanning out from the hole.
			a += (vnoise((float64(x)+0.5-ax)/half*4, t*1.5, 973) - 0.5) * 0.22
			a = clamp01(a)
			// Quantise the glow to a few dithered levels.
			lv := math.Floor(a*5) / 5
			if a*5-math.Floor(a*5) > dither(x, y) {
				lv += 1.0 / 5
			}
			light := dayRamp.pick(0.9-0.5*s, x, y)
			if c.at(x, y).A > 0 {
				c.put(x, y, mix(c.at(x, y), light, lv*0.75))
			} else if lv > 0 {
				c.put(x, y, withAlpha(light, uint8(min(215, lv*255))))
			}
		}
	}

	f := newCanvas(c.w, c.h)
	// The hole in the ceiling: a ragged collar of rock, lit from within.
	f.ellipse(ax, holeY, 19, 8.5, func(x, y int, nx, ny float64) color.RGBA {
		d := math.Hypot(nx, ny)
		if d > 0.78+0.22*vnoise(nx/d*2.2+5, ny/d*2.2+9, 974) {
			return color.RGBA{}
		}
		// Brightest next to the hole, falling to black at the fringe.
		l := 1.5 - 1.5*d + (vnoise(float64(x)*0.5, float64(y)*0.6, 975)-0.5)*0.4
		if ny > 0 {
			l += 0.12
		}
		return ceilRockRamp.pick(l, x, y)
	})
	// Up the shaft: the sky, and the far wall of earth under the turf.
	f.ellipse(ax, holeY, 10.5, 4.6, func(x, y int, nx, ny float64) color.RGBA {
		switch {
		case ny < -0.5 && hashf(x, 0, 976) < 0.55:
			return grassRamp.pick(0.3+0.5*hashf(x, 1, 976), x, y)
		case ny < -0.1-0.25*vnoise(float64(x)*0.5, 0, 977):
			return caveDirtRamp.pick(0.55+0.3*hashf(x, y, 978), x, y)
		}
		return dayRamp.pick(1-0.5*(nx*nx+ny*ny), x, y)
	})
	// Roots hanging from the rim.
	for _, r := range [][3]float64{{17.5, 12, 7}, {21, 14, 4}, {37, 13.5, 6}, {40.5, 11, 4}} {
		f.line(r[0], r[1], r[0]+math.Copysign(0.8, ax-r[0]), r[1]+r[2], func(x, y int, t float64) color.RGBA {
			return caveDirtRamp.pick(0.75-0.35*t, x, y)
		})
	}

	// Rubble fallen from the ceiling, lit by the shaft, round the foot of
	// the ladder.
	for i, r := range []caveRock{
		{ax + 9.5, ay - 3, 3.2, 2.6, true}, {ax - 10, ay - 2, 4, 3.2, true},
		{ax + 8, ay + 2.5, 4.6, 3.4, true}, {ax - 13.5, ay + 3, 2.6, 2, true},
		{ax - 4, ay + 5.5, 4, 2.8, true}, {ax + 14, ay + 4.5, 2.4, 1.9, true},
		{ax + 2.5, ay + 7.5, 2.4, 1.8, true},
	} {
		drawCaveRock(f, r, paleRockRamp, 0.08, uint32(980+i))
	}
	// A sod of turf that came down with it.
	f.ellipse(ax-3.5, ay+1.2, 4.2, 1.6, func(x, y int, nx, ny float64) color.RGBA {
		if ny > 0.2 {
			return caveDirtRamp.pick(0.45+0.2*hashf(x, y, 990), x, y)
		}
		return grassRamp.pick(0.55+0.4*hashf(x, y, 991), x, y)
	})

	f.outline(0.8)

	// The rope ladder, narrowing towards the hole it hangs from. It is drawn
	// without an outline, each rope and rung shaded on its own, to stay fine.
	top, bot := holeY+1, ay-2
	railX := func(y float64, side float64) float64 {
		t := (y - top) / (bot - top)
		return ax + side*lerpf(3.4, 5.2, t) + math.Sin(t*5)*0.5*t
	}
	ink := func(col color.RGBA) color.RGBA { return mix(col, colInk, 0.65) }
	for y := int(top); y <= int(bot); y++ {
		fy := float64(y) + 0.5
		l, r := int(math.Floor(railX(fy, -1))), int(math.Floor(railX(fy, 1)))
		if (y-int(top))%7 == 5 {
			for x := l + 1; x < r; x++ {
				f.put(x, y, woodRamp.at(3-(x-l)/4))
				f.put(x, y+1, woodRamp.at(1-(x-l)/6))
				f.put(x, y+2, ink(woodRamp[0]))
			}
		}
		// Twisted rope: the strands alternate down its length.
		for i, x := range [2]int{l, r} {
			f.put(x, y, ropeRamp.at(1+((y+i)/2)%2))
			f.put(x+1, y, ink(ropeRamp[0]))
		}
	}
	// Grass among the rubble, after the outline so the blades stay fine.
	for _, b := range [][3]float64{
		{ax - 6, ay + 0.5, -0.3}, {ax - 4, ay + 0.5, 0.15}, {ax - 1.5, ay + 0.5, 0.4},
		{ax + 12, ay + 1, 0.3}, {ax - 15, ay + 1, -0.25},
	} {
		f.line(b[0], b[1], b[0]+b[2]*3, b[1]-3.5, func(x, y int, t float64) color.RGBA {
			return grassRamp.pick(0.45+t*0.55, x, y)
		})
	}
	c.over(f, 0, 0)
	return c
}
