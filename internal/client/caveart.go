package client

import (
	"image/color"
	"math"
)

// The caves under the islands are entered through a cave mouth: a low, lumpy
// arch of pale boulders lying on the ground around a hole that goes down
// into the dark, with dug-out earth spilling from it. Inside, the way back
// up is the same mouth seen from within: an arch of boulders in the rock
// wall, the day showing through it, and a ramp of earth climbing to it.

// caveArt holds the cave mouth, which stands on the ground above the caves,
// and the way back up inside them.
type caveArt struct {
	mouth sprite // the whole rocky cave mouth with its dirt spill and loose stones, opening facing the viewer (south)
	exit  sprite // inside the cave: the way out, the mouth seen from within, daylight coming in down a ramp of earth
}

var (
	// Pale grey-beige limestone: the boulders of the cave mouth.
	paleRockRamp = mkRamp(0x453d44, 0x625a5c, 0x857b75, 0xa59b8e, 0xc0b7a7, 0xd8d0c0, 0xece6d8, 0xf8f4ea)
	// The rock wall inside the mouth, in deep blue-grey shadow.
	caveWallRamp = mkRamp(0x0e0d14, 0x191a26, 0x262a3a, 0x363d52, 0x4b556c, 0x646f86)
	// Dug earth, from the black of the pit to the ochre crumbs in the sun.
	caveDirtRamp = mkRamp(0x1c120c, 0x33211a, 0x4f3420, 0x6f4b29, 0x8d6532, 0xa8813f, 0xc09a52)
	// The rock of the cave wall and floor, seen from inside by daylight.
	ceilRockRamp = mkRamp(0x1c181d, 0x2c262b, 0x40383b, 0x5a5050, 0x7a6f69, 0x9c9184, 0xc0b6a2)
	// Daylight coming in at the mouth.
	dayRamp = mkRamp(0xe8c878, 0xf6dc94, 0xfff0b8, 0xfffbe2)
)

// Anchors of the two sprites, in canvas pixels.
const (
	caveMouthAX, caveMouthAY = 52.0, 51.0
	caveExitAX, caveExitAY   = 36.0, 64.0
)

func genCaveArt() caveArt {
	return caveArt{
		mouth: mkUp(caveMouthCanvas(), caveMouthAX, caveMouthAY),
		exit:  mkUp(caveExitCanvas(), caveExitAX, caveExitAY),
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

// caveExitCanvas draws the way out, seen from inside: the cave mouth in the
// rock wall to the north, with the daylight coming in down a ramp of earth.
// The anchor (caveExitAX, caveExitAY) is the middle of the floor at the foot
// of the ramp.
func caveExitCanvas() *canvas {
	const ax, ay = caveExitAX, caveExitAY
	// The opening: an arch whose curve springs from archY, down to the sill.
	const archY, archRX, archRY, sillY = 25.0, 12.5, 14.0, 40.0
	// The ramp runs from the grass outside (rampTop) down to the anchor.
	const rampTop = 27.0
	c := newCanvas(72, 78)

	// The pool of daylight on the floor at the foot of the ramp.
	c.ellipse(ax, ay-1, 27, 13, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		if e := 0.6 + 0.4*vnoise(float64(x)*0.35, float64(y)*0.5, 971); d > e || (e-d)*4 < dither(x, y) {
			return color.RGBA{}
		}
		l := 0.95 - 0.8*d + (vnoise(float64(x)*0.5, float64(y)*0.7, 972)-0.5)*0.25
		return mix(ceilRockRamp.pick(0.3+0.6*l, x, y), dayRamp[0], 0.15+0.25*l)
	})

	f := newCanvas(c.w, c.h)
	// The rock wall the mouth is cut through: dark, catching a little light
	// next to the opening, and lost in the gloom further off.
	for y := 0; y < 50; y++ {
		for x := 0; x < f.w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Hypot((px-ax)/34, (py-30)/27)
			if d > 0.8+0.2*vnoise(px*0.16, py*0.16, 973) || py > 45+4*vnoise(px*0.3, 0, 974) {
				continue
			}
			l := 0.62 - 0.6*d + (hashf(x/2, y/6, 975)-0.5)*0.14 + (vnoise(px*0.4, py*0.3, 976)-0.5)*0.2
			f.put(x, y, ceilRockRamp.pick(l, x, y))
		}
	}
	// The boulders behind the arch and trailing away along the wall.
	back := []caveRock{
		{7.5, 41, 5, 4, true}, {64.5, 42, 5, 4, true},
		{13, 34, 6.5, 6, false}, {59, 35, 6.5, 6, false},
		{17, 24, 7, 7, false}, {55, 24.5, 7, 7, false},
		{21, 12, 8, 6.5, false}, {51, 12, 8, 6.5, false},
		{36, 6, 11, 5, false},
	}
	for i, r := range back {
		drawCaveRock(f, r, paleRockRamp, -0.3, uint32(980+i))
	}

	// rampHalf is the half width of the ramp at height y, and rampLum how
	// bright its earth is there: washed out in the sun at the top, falling
	// into the dusk of the cave at the foot, in ledges stepping down.
	rampHalf := func(py float64) float64 {
		t := clamp01((py - rampTop) / (ay - rampTop))
		return lerpf(11, 17.5, t*t) + (vnoise(py*0.35, 3, 977)-0.5)*3*t
	}
	rampLum := func(x, y int) float64 {
		px, py := float64(x)+0.5, float64(y)+0.5
		t := (py - rampTop) / (ay + 3 - rampTop)
		l := 1.02 - 0.48*t + (vnoise(px*0.45, py*0.8, 978)-0.5)*0.22
		// The ledges sag in the middle, where feet have worn them.
		step := math.Mod(py+2.2*(1-math.Pow((px-ax)/16, 2))+vnoise(px*0.2, 0, 979)*4, 5.5)
		switch {
		case step < 1:
			l -= 0.28 // the shaded riser under a ledge
		case step > 4.5:
			l += 0.14 // its lit edge
		}
		// The light falls from the left jamb; the right side is duller.
		return l - 0.1*(px-ax)/16
	}
	// Through the mouth: the sky, the sunlit grass outside, and the top of
	// the ramp.
	for y := int(archY - archRY); y < int(sillY); y++ {
		for x := 23; x <= 48; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			u := (px - ax) / archRX
			if math.Abs(u) > 1 {
				continue
			}
			if v := (py - archY) / archRY; py < archY && u*u+v*v > 1 {
				continue
			}
			var col color.RGBA
			switch {
			case py >= rampTop-1.5*vnoise(px*0.4, 0, 981):
				col = mix(caveDirtRamp.pick(rampLum(x, y), x, y), dayRamp[2], 0.3)
			case py >= 21.5-2.5*vnoise(px*0.3, 1, 982)-2*hashf(x, 0, 983):
				// Grass in full sun, bleached by the glare.
				l := 0.5 + 0.5*hashf(x, y, 984) - (py-21)*0.05
				col = mix(grassRamp.pick(l, x, y), dayRamp[2], 0.25)
			default:
				// The sky, whitest at the top of the arch.
				col = dayRamp.pick(1.05-(py-(archY-archRY))/16-0.25*u*u, x, y)
			}
			f.put(x, y, col)
		}
	}
	// The ramp inside the cave, widening and breaking up into crumbs at the
	// foot.
	for y := int(sillY); y < int(ay)+5; y++ {
		for x := 0; x < f.w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			s := math.Abs(px-ax) / rampHalf(py)
			if s > 1 {
				continue
			}
			// Ragged where it runs out over the floor.
			t := clamp01((py - (ay - 6)) / 11)
			if cover := vnoise(px*0.4, py*0.9, 985) + 0.1 - t*(1.1+0.5*s); cover < 0 || cover*4 < t*dither(x, y) {
				continue
			}
			l := rampLum(x, y) - 0.22*s*s*s
			f.put(x, y, mix(caveDirtRamp.pick(l, x, y), dayRamp[2], 0.3*clamp01(1-(py-rampTop)/22)))
		}
	}

	// The frame of the arch, standing proud of the wall: two jambs of stacked
	// boulders with their feet either side of the ramp, and the lintel.
	rim := []caveRock{
		{23, 15.5, 6, 5, false}, {49, 15.5, 6, 5, false},
		{28.5, 9, 6, 4, false}, {43.5, 9, 6, 4, false},
		{36, 6.5, 7.5, 4, false},
		{21.5, 26, 5, 6, false}, {50.5, 26.5, 5, 6, false},
		{20.5, 36.5, 5.5, 6.5, false}, {51.5, 37, 5.5, 6.5, false},
		{18, 45.5, 6, 5.5, true}, {54.5, 46, 6, 5.5, true},
		{12, 48.5, 4, 3, true}, {61, 49, 4, 3, true},
	}
	for i, r := range rim {
		drawCaveRock(f, r, paleRockRamp, 0.02, uint32(1000+i))
	}
	// Scree: stones that have rolled down the ramp and lie along its sides.
	for i, r := range []caveRock{
		{ax - 15, ay - 10, 3.4, 2.7, true}, {ax + 16.5, ay - 9, 3, 2.4, true},
		{ax - 19.5, ay - 3.5, 2.6, 2, true}, {ax + 19, ay - 2, 3.6, 2.8, true},
		{ax - 16, ay + 3.5, 3.8, 2.9, true}, {ax + 13.5, ay + 5.5, 2.4, 1.9, true},
		{ax - 7, ay + 8.5, 2.2, 1.7, true}, {ax + 5, ay - 13, 2, 1.6, true},
		{ax - 6.5, ay - 19, 1.8, 1.4, true},
	} {
		drawCaveRock(f, r, paleRockRamp, 0.06, uint32(1020+i))
	}
	f.outline(0.8)

	// Pebbles on the ramp and the floor, too small for an outline.
	for i := 0; i < 16; i++ {
		py := lerpf(sillY-6, ay+9, hashf(i, 1, 1030))
		px := ax + (hashf(i, 2, 1030)*2-1)*(rampHalf(py)+2)
		x, y := int(px), int(py)
		if !f.opaque(x, y) || !f.opaque(x+1, y) {
			continue
		}
		f.put(x, y, paleRockRamp.at(5+i%2))
		f.put(x+1, y, paleRockRamp.at(3))
		f.put(x, y+1, mix(f.at(x, y+1), colInk, 0.45))
		f.put(x+1, y+1, mix(f.at(x+1, y+1), colInk, 0.45))
	}
	// Roots hanging from the lintel, dark against the sky.
	for _, r := range [][3]float64{{28.5, 13, 7}, {31.5, 11.5, 4}, {40, 11.5, 5}, {43.5, 13, 8}} {
		f.line(r[0], r[1], r[0]+math.Copysign(0.9, ax-r[0]), r[1]+r[2], func(x, y int, t float64) color.RGBA {
			return caveDirtRamp.pick(0.3-0.2*t, x, y)
		})
	}
	// Grass that has crept in at the lip, and a few blades further down.
	for _, b := range [][3]float64{
		{ax - 10.5, sillY - 1, -0.2}, {ax - 8.5, sillY - 2, 0.25}, {ax - 6.5, sillY - 1.5, 0.5},
		{ax + 7.5, sillY - 2.5, -0.4}, {ax + 9.5, sillY - 2, 0.1}, {ax + 11, sillY - 0.5, 0.35},
		{ax - 12, ay - 12, -0.3}, {ax - 10.5, ay - 12, 0.3}, {ax + 14, ay - 4, 0.3},
	} {
		f.line(b[0], b[1], b[0]+b[2]*3, b[1]-4, func(x, y int, t float64) color.RGBA {
			return grassRamp.pick(0.35+t*0.65, x, y)
		})
	}
	c.over(f, 0, 0)
	return c
}
