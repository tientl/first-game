package client

import (
	"image/color"
	"math"
)

// The caves under the islands are entered through a cave mouth: an outcrop
// of pale rock with a sheer face turned to the south, towards the viewer,
// turf on its top, and at the foot of the face an arched doorway that goes
// into the dark, trodden earth spilling out of it. It is walked into from
// below. Inside, the way back up is that doorway seen from within: an arch
// of daylight in the rock wall to the north, at the top of a flight of
// worn steps.
//
// Both are drawn at HD density, in art pixels (see HD_ART.md).

// caveArt holds the cave mouth, which stands on the ground above the caves,
// and the way back up inside them.
type caveArt struct {
	mouth sprite // the outcrop with its doorway facing the viewer (south), the path and the loose stones
	exit  sprite // inside the cave: the way out, an arch of daylight to the north above a flight of steps
}

var (
	// Pale grey-beige limestone: the rock of the cave mouth.
	paleRockRamp = mkRamp(0x453d44, 0x625a5c, 0x857b75, 0xa59b8e, 0xc0b7a7, 0xd8d0c0, 0xece6d8, 0xf8f4ea)
	// The rock inside the doorway, in deep blue-grey shadow.
	caveWallRamp = mkRamp(0x0e0d14, 0x191a26, 0x262a3a, 0x363d52, 0x4b556c, 0x646f86)
	// Trodden earth, from the black of the cave to the ochre crumbs in the sun.
	caveDirtRamp = mkRamp(0x1c120c, 0x33211a, 0x4f3420, 0x6f4b29, 0x8d6532, 0xa8813f, 0xc09a52)
	// The rock of the cave wall and floor, seen from inside by daylight.
	ceilRockRamp = mkRamp(0x1c181d, 0x2c262b, 0x40383b, 0x5a5050, 0x7a6f69, 0x9c9184, 0xc0b6a2)
	// Daylight coming in at the mouth.
	dayRamp = mkRamp(0xe8c878, 0xf6dc94, 0xfff0b8, 0xfffbe2)
	// The turf and moss on top of the outcrop, a little deeper than the lawn.
	caveTurfRamp = mkRamp(0x24402a, 0x2f5a30, 0x3f7a37, 0x53943f, 0x6daa49, 0x8cc257)
	// The black at the back of the doorway.
	caveBlack = hex(0x06050a)
)

// Sizes and anchors of the two sprites, in art pixels. The mouth's anchor is
// the middle of its tile: the foot of the face runs along the tile's south
// edge. The exit's is the floor at the foot of the steps.
const (
	caveMouthW, caveMouthH   = 100, 94
	caveMouthAX, caveMouthAY = 50.0, 60.0
	caveExitW, caveExitH     = 84, 92
	caveExitAX, caveExitAY   = 42.0, 72.0
)

func genCaveArt() caveArt {
	return caveArt{
		mouth: spriteHD(caveMouthCanvas().image(), caveMouthAX, caveMouthAY),
		exit:  spriteHD(caveExitCanvas().image(), caveExitAX, caveExitAY),
	}
}

// eachHD calls f for every pixel of the HD canvas c, with the art-pixel
// coordinates of its centre.
func eachHD(c *canvas, f func(x, y int, px, py float64)) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			f(x, y, (float64(x)+0.5)/hdK3, (float64(y)+0.5)/hdK3)
		}
	}
}

// crag breaks rock into blocks sx by sy art pixels, laid like rough
// masonry. For the point (px, py) it gives its block's shade (-0.5 to
// 0.5), how far it is from the crack round the block, and whether it is on
// the block's lit upper-left side.
func crag(px, py, sx, sy float64, seed uint32) (shade, edge float64, lit bool) {
	gy := py / sy
	iy := int(math.Floor(gy))
	d1, d2 := math.Inf(1), math.Inf(1)
	var bx, by float64
	var bi, bj int
	for j := iy - 1; j <= iy+1; j++ {
		gx := px/sx + 0.5*float64(j&1)
		ix := int(math.Floor(gx))
		for i := ix - 1; i <= ix+1; i++ {
			cx := float64(i) + 0.2 + 0.6*hashf(i, j, seed)
			cy := float64(j) + 0.2 + 0.6*hashf(i, j, seed+1)
			d := math.Hypot((gx-cx)*sx, (gy-cy)*sy)
			if d < d1 {
				d1, d2 = d, d1
				bx, by, bi, bj = (gx-cx)*sx, (gy-cy)*sy, i, j
			} else if d < d2 {
				d2 = d
			}
		}
	}
	edge = (d2 - d1) / 2
	if vnoise(px*0.13, py*0.17, seed+3) < 0.42 {
		edge += 0.5 // the crack closes up here: the rock is not masonry
	}
	return hashf(bi, bj, seed+2) - 0.5, edge, bx+by < 0
}

// blockHD draws a squarish block of stone at (x, y), rx by ry art pixels:
// a lit bevel along its top, a dull face, shadow underneath and on the
// right, and a crack where it rests on the rock around it.
func blockHD(c *canvas, x, y, rx, ry, lit float64, rp ramp, seed uint32) {
	x0, x1 := int((x-rx-1)*hdK3), int((x+rx+1)*hdK3)
	y0, y1 := int((y-ry-1)*hdK3), int((y+ry+1)*hdK3)
	for iy := y0; iy <= y1; iy++ {
		for ix := x0; ix <= x1; ix++ {
			px, py := (float64(ix)+0.5)/hdK3, (float64(iy)+0.5)/hdK3
			nx, ny := (px-x)/rx, (py-y)/ry
			// A lumpy edge: the block is not quite square.
			wob := 1 + 0.12*(vnoise(px*0.5, py*0.5, seed)-0.5)
			s := math.Pow(math.Abs(nx), 4) + math.Pow(math.Abs(ny), 4)
			if s > wob {
				ox, oy := (px-x)/(rx+0.5), (py-y)/(ry+0.5)
				if math.Pow(math.Abs(ox), 4)+math.Pow(math.Abs(oy), 4) <= wob && c.opaque(ix, iy) {
					c.put(ix, iy, mix(c.at(ix, iy), rp[0], 0.6))
				}
				continue
			}
			l := 0.64 - 0.2*ny - 0.14*nx + (vnoise(px*0.8, py*0.8, seed+1)-0.5)*0.16
			switch {
			case ny < -0.45+0.2*nx:
				l += 0.17 // the bevel, catching the sun
			case ny > 0.6:
				l -= 0.15
			}
			if nx > 0.62 {
				l -= 0.12
			}
			if hashf(ix/2, iy/2, seed+2) < 0.025 {
				l -= 0.18 // a pit
			}
			c.put(ix, iy, rp.pick(l*lit, ix, iy))
		}
	}
}

// creeperHD hangs a creeper from (x, y), ln art pixels long: a thin dark stem
// with small leaves on alternate sides.
func creeperHD(c *canvas, x, y, ln float64, seed uint32) {
	ph := hashf(int(x), 1, seed) * 6
	at := func(s float64) (float64, float64) {
		return x + math.Sin(s*0.45+ph)*0.9*math.Min(1, s/3), y + s
	}
	for s := 0.0; s < ln; s += 0.2 {
		vx, vy := at(s)
		ix, iy := int(vx*hdK3), int(vy*hdK3)
		c.put(ix, iy, caveTurfRamp[0])
		c.put(ix+1, iy, caveTurfRamp[1])
	}
	side := 1.0
	for s := 0.8; s <= ln; s += 1.9 + hashf(int(s*7), 2, seed) {
		vx, vy := at(s)
		side = -side
		lx, ly := vx+side*1.1+0.3, vy+0.4
		tone := 0.2 * hashf(int(s*9), 3, seed)
		c.ellipse(lx*hdK3, ly*hdK3, 3.4, 2.6, func(px, py int, nx, ny float64) color.RGBA {
			return caveTurfRamp.pick(0.62-0.3*nx-0.35*ny+tone, px, py)
		})
	}
}

// caveMouthCanvas draws the cave mouth: the outcrop, its doorway and the
// path. The anchor (caveMouthAX, caveMouthAY) is the middle of the mouth's
// tile; the outcrop stands on the rock tiles beside and behind it.
func caveMouthCanvas() *canvas {
	const ax, ay = caveMouthAX, caveMouthAY
	const seed = 900
	// The doorway: f0 is its threshold, the arch springs at ys.
	const f0, half, archRY = ay + 8.0, 10.5, 11.0
	const ys = f0 - 13

	unit := func(px float64) float64 { return (px - ax) / 42 }
	// The foot of the face bows back at the ends, as the outcrop is round.
	foot := func(px float64) float64 {
		u := unit(px)
		return f0 - 7*u*u + (vnoise(px*0.3, 0, seed)-0.5)*1.4
	}
	// The brow, where the face meets the top, stepping up and down in
	// ledges; smooth is the same without the ledges.
	brow := func(px float64) (stepped, smooth float64) {
		u := unit(px)
		e := math.Pow(math.Max(0, 1-math.Pow(u, 10)), 0.4)
		h := 28 + 9*(1-u*u) + 5*math.Exp(-math.Pow((u+0.55)/0.2, 2)) - 6*math.Exp(-math.Pow((u-0.66)/0.16, 2))
		st := (math.Floor(vnoise(px*0.1, 7, seed)*4) - 1.5) * 2.6
		if math.Abs(u) < 0.36 {
			st = math.Max(st, -1) // room for the lintel
		}
		return foot(px) - (h+st)*e, foot(px) - h*e
	}
	// The far edge of the top.
	back := func(px float64) float64 {
		u := unit(px)
		_, b := brow(px)
		return b - (5+10*math.Sqrt(math.Max(0, 1-u*u)))*math.Sqrt(math.Max(0, 1-math.Pow(u, 4))) - (math.Floor(vnoise(px*0.13, 3, seed)*3)-1)*2.2*math.Max(0, 1-u*u*1.6)
	}
	// How far the turf hangs over the brow.
	hang := func(x int, px float64) float64 {
		return 0.5 + 2.4*vnoise(px*1.3, 5, seed) + 1.2*hashf(x/2, 0, seed+1)
	}
	inDoor := func(px, py float64) bool {
		dx := (px - ax) / half
		if math.Abs(dx) > 1 || py > f0+0.8*(1-dx*dx) {
			return false
		}
		v := (py - ys) / archRY
		return py >= ys || dx*dx+v*v <= 1
	}

	rocks := hdCanvas(caveMouthW, caveMouthH)
	eachHD(rocks, func(x, y int, px, py float64) {
		u := unit(px)
		if math.Abs(u) >= 1 {
			return
		}
		ft, bk := foot(px), back(px)
		br, _ := brow(px)
		if py < bk || py > ft {
			return
		}
		shade, edge, lit := crag(px, py, 11, 7, seed+2)
		grain := (vnoise(px*0.7, py*0.7, seed+5) - 0.5) * 0.1
		if py < br {
			// The top: bare rock round the rim, turf and moss within.
			rim := math.Min(py-bk, (1-math.Abs(u))*26)
			bare := rim < 2+2.2*vnoise(px*0.4, py*0.4, seed+6) || vnoise(px*0.18, py*0.3, seed+7) > 0.74
			if bare {
				l := 0.8 - 0.14*u + shade*0.16 + grain
				if edge < 0.4 {
					l -= 0.22
				}
				if rim < 0.9 {
					l -= 0.12
				}
				rocks.put(x, y, paleRockRamp.pick(l, x, y))
				return
			}
			n := vnoise(px*0.35, py*0.5, seed+8)*0.55 + hashf(x/2, y/2, seed+9)*0.45
			l := 0.3 + 0.55*n - 0.14*u
			if br-py < 2 {
				l += 0.18 // the sunlit lip
			}
			rocks.put(x, y, caveTurfRamp.pick(l, x, y))
			return
		}
		// The face.
		hg := hang(x, px)
		turf := math.Abs(u) < 0.93
		if turf && py-br < hg {
			rocks.put(x, y, caveTurfRamp.pick(0.5+0.4*hashf(x/2, y/3, seed+10)-0.1*u, x, y))
			return
		}
		t := (py - br) / math.Max(1, ft-br)
		l := 0.6 - 0.16*u - 0.2*t*t + shade*0.2 + grain
		if u > 0.7 {
			l -= (u - 0.7) * 1.1 // the face turns away from the sun
		}
		switch {
		case edge < 0.42:
			l -= 0.3 // a crack
		case edge < 1 && lit:
			l += 0.09
		}
		if turf && py-br < hg+1.6 {
			l -= 0.22 // the shadow under the turf
		}
		if ft-py < 1.4 {
			l -= 0.12
		}
		// Ledges: beds of rock running across the face.
		switch bed := math.Mod(py+2.5*vnoise(px*0.09, 9, seed)+60, 9.5); {
		case bed < 0.7:
			l += 0.13
		case bed < 1.7:
			l -= 0.16
		}
		col := paleRockRamp.pick(l, x, y)
		// Moss creeping down from the top.
		if m := vnoise(px*0.22, py*0.16, seed+11) - 0.45*t; m > 0.5 && edge > 0.42 {
			col = mix(col, caveTurfRamp.pick(l-0.1, x, y), 0.6)
		}
		rocks.put(x, y, col)
	})

	// The doorway: black at the back, the right jamb catching a little
	// light inside, and the day lying on the earth floor at the threshold.
	eachHD(rocks, func(x, y int, px, py float64) {
		if !inDoor(px, py) {
			return
		}
		dx := px - ax
		v := (f0 - py) / (archRY + 13) // 0 at the threshold, 1 at the crown
		col := mix(caveWallRamp[1], caveBlack, clamp01(v*1.8))
		jamb := dx > half-3.4*(1-0.6*v)
		if jamb {
			// The inside of the right jamb.
			l := 0.62 - 0.9*v + (hashf(x/2, y/4, seed+12)-0.5)*0.12
			col = mix(caveWallRamp.pick(l, x, y), caveBlack, clamp01(v*1.3))
		} else if dx < -half+1.5 {
			col = caveWallRamp.pick(0.12, x, y)
		}
		if tt := v / 0.36; tt < 1 {
			// The floor: the sun reaches in past the left jamb.
			l := 0.8 - 0.7*tt - 0.3*clamp01(-dx/half-0.15) + (vnoise(px*0.7, py*1.2, seed+13)-0.5)*0.2
			if jamb {
				l -= 0.25
			}
			col = mix(caveDirtRamp.pick(l, x, y), col, tt*tt)
		}
		rocks.put(x, y, col)
	})
	// Roots hanging in the doorway.
	for _, r := range [][3]float64{{-4.5, 1, 5.5}, {2, 0.5, 3.5}, {5.5, 1.5, 6.5}} {
		for s := 0.0; s < r[2]; s += 0.25 {
			px := ax + r[0] + math.Sin(s*0.7+r[0])*0.5
			rocks.put(int(px*hdK3), int((f0-24+r[1]+s)*hdK3), caveDirtRamp.pick(0.5-0.3*s/r[2], 0, 0))
		}
	}

	// The frame: jambs of stacked blocks, shoulder stones, and the lintel.
	for i, b := range [][5]float64{
		{ax - half - 2.4, f0 - 4.4, 4.2, 4.8, 0.95}, {ax + half + 2.6, f0 - 4.2, 4.4, 4.6, 0.8},
		{ax - half - 2.8, f0 - 13, 3.8, 4.4, 1}, {ax + half + 3, f0 - 12.6, 4, 4.6, 0.84},
		{ax - half - 0.6, f0 - 21, 4.4, 3.9, 1.04}, {ax + half + 0.8, f0 - 20.6, 4.4, 4.1, 0.88},
		{ax + 0.3, f0 - 27.2, 13, 3.7, 1.06},
	} {
		blockHD(rocks, b[0], b[1], b[2], b[3], b[4], paleRockRamp, uint32(seed+20+i))
	}
	// Loose stones at the foot of the face and beside the path.
	for i, s := range [][4]float64{
		{ax - 31, foot(ax-31) + 1.5, 3.4, 2.6}, {ax - 36.5, foot(ax-36.5) + 3, 2, 1.6},
		{ax + 27, foot(ax+27) + 2, 3.8, 2.9}, {ax + 33.5, foot(ax+33.5) + 3.5, 2.2, 1.7},
		{ax - 15.5, f0 + 3, 2.6, 2}, {ax + 15, f0 + 5.5, 3, 2.3}, {ax + 19.5, f0 + 12, 1.8, 1.4},
		{ax - 20, f0 + 14, 2.2, 1.7},
	} {
		stoneHD(rocks, s[0]*hdK3, s[1]*hdK3, s[2]*hdK3, s[3]*hdK3, 0.95, paleRockRamp, uint32(seed+40+i))
	}
	rocks.outline(0.7)

	// The path: scuffed earth spilling out of the doorway and down, lying
	// flat on the ground with no outline, breaking up into crumbs.
	c := hdCanvas(caveMouthW, caveMouthH)
	eachHD(c, func(x, y int, px, py float64) {
		t := (py - f0 + 2) / 27
		if t < 0 || t > 1 {
			return
		}
		s := math.Abs(px-(ax-3.5*t*t)) / (9.5 + 8.5*math.Pow(t, 0.7))
		if s > 1.2 {
			return
		}
		n := vnoise(px*0.22, py*0.3, seed+60)*0.6 + vnoise(px*0.9, py*1.1, seed+61)*0.4
		cover := 1.12 - 0.5*t - 0.62*s*s - 1.6*math.Max(0, t-0.62)
		if n*0.9 > cover {
			return
		}
		l := 0.74 - 0.24*n + 0.06*t + (hashf(x, y, seed+62)-0.5)*0.1
		if vnoise(px*0.5, py*1.3, seed+63) > 0.68 {
			l -= 0.13 // a scuff, a footprint
		}
		if cover-n*0.9 < 0.07 {
			l -= 0.17 // the damp edge
		}
		c.put(x, y, caveDirtRamp.pick(l, x, y))
	})
	// Pebbles on the path.
	for i := 0; i < 14; i++ {
		x := int((ax + (hashf(i, 1, seed+64)*2-1)*13) * hdK3)
		y := int((f0 + 2 + hashf(i, 2, seed+64)*17) * hdK3)
		if !c.opaque(x, y) || !c.opaque(x+2, y+2) {
			continue
		}
		for d := 0; d < 4; d++ {
			dx, dy := d&1, d>>1
			c.put(x+dx, y+dy, paleRockRamp.at(5-dx-dy))
			c.put(x+dx+1, y+2, mix(c.at(x+dx+1, y+2), colInk, 0.4))
		}
	}
	// The outcrop's shadow on the ground along its foot.
	eachHD(c, func(x, y int, px, py float64) {
		if math.Abs(unit(px)) >= 1.02 {
			return
		}
		d := py - foot(px) - 2*clamp01(unit(px)) // deeper on the side away from the sun
		if d < -3 || d > 3.6 {
			return
		}
		a := 0.42 * (1 - clamp01(d/3.6))
		if c.opaque(x, y) {
			c.put(x, y, mix(c.at(x, y), caveDirtRamp[0], a))
		} else {
			c.put(x, y, withAlpha(hex(0x121c14), uint8(a*255)))
		}
	})
	// over keeps only solid pixels: lay the rocks on by hand, to keep the shadow.
	for i, p := range rocks.px {
		if p.A >= 128 {
			c.px[i] = p
		}
	}

	// Creepers hanging down the face, one across a corner of the doorway.
	for i, v := range [][2]float64{{-27, 9}, {-13.5, 15}, {-6.5, 8.5}, {8, 6}, {12.5, 17}, {25, 8}} {
		b, _ := brow(ax + v[0])
		creeperHD(c, ax+v[0], b+1.5, v[1], uint32(seed+80+i))
	}
	// Tufts of grass at the foot, either side of the path.
	for i, g := range [][2]float64{{-24, 0}, {-22, 1}, {-19.5, 0.5}, {21, 1}, {23.5, 0}, {-38, -2}, {37, -3}} {
		bx, by := ax+g[0], foot(ax+g[0])+2.5+g[1]
		for k := -1; k <= 1; k++ {
			tip := bx + float64(k)*1.6 + (hashf(i, k+5, seed+90)-0.5)*1.2
			c.line(bx*hdK3, by*hdK3, tip*hdK3, (by-4-hashf(i, k+1, seed+91)*1.5)*hdK3, func(x, y int, t float64) color.RGBA {
				return grassRamp.pick(0.3+0.7*t, x, y)
			})
		}
	}
	return c
}

// caveExitCanvas draws the way out, seen from inside: an arch of daylight in
// the rock wall to the north, at the top of the screen, and a flight of worn
// steps coming down from it. The anchor (caveExitAX, caveExitAY) is the
// middle of the floor at the foot of the steps.
func caveExitCanvas() *canvas {
	const ax, ay = caveExitAX, caveExitAY
	const seed = 970
	// The opening: an arch that springs at ys, down to the sill.
	const ys, half, archRY, sill = 25.0, 11.5, 13.0, 38.0
	inArch := func(px, py float64) bool {
		dx := (px - ax) / half
		if math.Abs(dx) > 1 || py > sill {
			return false
		}
		v := (py - ys) / archRY
		return py >= ys || dx*dx+v*v <= 1
	}
	// near is how close (px, py) is to the opening: 1 in it, falling off
	// over a few art pixels.
	near := func(px, py float64) float64 {
		if py < ys {
			d := math.Hypot((px-ax)/half, (py-ys)/archRY)
			return clamp01(1 - (d-1)*1.6)
		}
		dx := math.Max(0, math.Abs(px-ax)-half)
		return clamp01(1 - math.Hypot(dx, math.Max(0, py-sill))/8)
	}

	// The day lying on the floor at the foot of the steps: a soft pool.
	c := hdCanvas(caveExitW, caveExitH)
	eachHD(c, func(x, y int, px, py float64) {
		d := math.Hypot((px-ax)/30, (py-(ay+1))/15)
		d += (vnoise(px*0.3, py*0.45, seed+1) - 0.5) * 0.25
		if d >= 1 {
			return
		}
		l := 1 - d
		col := mix(ceilRockRamp.pick(0.35+0.55*l, x, y), dayRamp[0], 0.12+0.3*l)
		c.put(x, y, withAlpha(col, uint8(255*clamp01(l*1.5)*0.85)))
	})

	f := hdCanvas(caveExitW, caveExitH)
	// The rock wall the mouth is cut through: lit round the opening, lost
	// in the gloom further off.
	eachHD(f, func(x, y int, px, py float64) {
		d := math.Hypot((px-ax)/38, (py-26)/30)
		if d > 0.84+0.16*vnoise(px*0.16, py*0.16, seed+2) || py > 46+4*vnoise(px*0.3, 0, seed+3) {
			return
		}
		shade, edge, lit := crag(px, py, 9, 6.5, seed+4)
		l := 0.5 - 0.5*d + shade*0.16 + (vnoise(px*0.6, py*0.6, seed+5)-0.5)*0.1
		switch {
		case edge < 0.42:
			l -= 0.2
		case edge < 1 && !lit:
			l += 0.06 // the light comes from the opening, below and within
		}
		col := ceilRockRamp.pick(l, x, y)
		f.put(x, y, mix(col, dayRamp[0], 0.3*math.Pow(near(px, py), 2)))
	})

	// Through the opening: the white sky, leaves in full sun, and the turf
	// at the lip.
	eachHD(f, func(x, y int, px, py float64) {
		if !inArch(px, py) {
			return
		}
		u := (px - ax) / half
		var col color.RGBA
		switch {
		case py > sill-4.5-1.5*vnoise(px*0.5, 0, seed+6):
			// Sunlit grass at the top of the steps.
			l := 0.55 + 0.45*hashf(x/2, y/2, seed+7) - (sill-py)*0.03
			col = mix(grassRamp.pick(l, x, y), dayRamp[2], 0.3)
		case py > sill-13+4*vnoise(px*0.28, 2, seed+8)+2*vnoise(px*0.9, 3, seed+9):
			// Bushes and trees beyond, bleached by the glare.
			l := 0.35 + 0.5*vnoise(px*0.6, py*0.6, seed+10)
			col = mix(caveTurfRamp.pick(l, x, y), dayRamp[3], 0.42)
		default:
			col = mix(dayRamp.pick(1.08-(py-(ys-archRY))/22-0.2*u*u, x, y), hex(0xd8f0ff), 0.25*clamp01((py-ys+archRY)/20))
		}
		f.put(x, y, col)
	})

	// The steps: slabs of stone and trodden earth, widening as they come
	// down, each tread in the light and each riser in its own shadow.
	const stepH = 5.6
	eachHD(f, func(x, y int, px, py float64) {
		if py < sill || py > ay+3.5 {
			return
		}
		n := math.Floor((py - sill) / stepH)
		fr := (py - sill - n*stepH) / stepH
		t := (n*stepH + stepH/2) / (ay + 3.5 - sill)
		hw := lerpf(half+0.5, 18.5, math.Pow(t, 1.2)) + (hashf(int(n), 1, seed+11)-0.5)*2.4
		off := (hashf(int(n), 2, seed+11) - 0.5) * 1.6
		s := math.Abs(px-ax-off) / hw
		if s > 1 {
			return
		}
		// The front of a slab sags in the middle, where feet have worn it.
		fr -= 0.12 * (1 - s*s)
		l := 1 - 0.5*t + (vnoise(px*0.5, py*0.9, seed+12)-0.5)*0.16 - 0.22*s*s*s - 0.06*(px-ax)/hw
		switch {
		case fr > 0.7:
			l -= 0.36 // the riser
		case fr > 0.56:
			l += 0.1 // the worn edge of the tread
		}
		if vnoise(px*0.35, py*0.5, seed+13) > 0.72 {
			l -= 0.1
		}
		col := mix(caveDirtRamp.pick(l*0.95, x, y), ceilRockRamp.pick(l, x, y), 0.45)
		f.put(x, y, mix(col, dayRamp[2], 0.34*clamp01(1-t*1.3)))
	})

	// The frame of the arch, standing proud of the wall: jambs of stacked
	// blocks, shoulder stones and the lintel.
	for i, b := range [][5]float64{
		{ax - half - 3, sill - 3, 5.2, 5.6, 0.62}, {ax + half + 3.2, sill - 2.6, 5.4, 5.4, 0.56},
		{ax - half - 3.4, sill - 13.5, 4.8, 5.2, 0.6}, {ax + half + 3.6, sill - 13, 4.8, 5.4, 0.54},
		{ax - half + 0.2, ys - 9.5, 5.6, 4.4, 0.58}, {ax + half - 0.2, ys - 9.2, 5.6, 4.6, 0.52},
		{ax, ys - archRY - 3.2, 12.5, 4.6, 0.6},
		{ax - half - 9.5, sill + 3, 4.4, 3.8, 0.5}, {ax + half + 10, sill + 3.5, 4.6, 3.8, 0.46},
	} {
		blockHD(f, b[0], b[1], b[2], b[3], b[4], paleRockRamp, uint32(seed+20+i))
	}
	// Scree along the sides of the steps.
	for i, s := range [][4]float64{
		{ax - 17, ay - 14, 3.2, 2.5}, {ax + 18.5, ay - 12, 2.8, 2.2}, {ax - 21.5, ay - 5, 2.5, 1.9},
		{ax + 21.5, ay - 3, 3.4, 2.6}, {ax - 20, ay + 4, 3.4, 2.6}, {ax + 19, ay + 6, 2.2, 1.7},
		{ax - 9, ay + 9, 2, 1.5},
	} {
		stoneHD(f, s[0]*hdK3, s[1]*hdK3, s[2]*hdK3, s[3]*hdK3, 0.7, paleRockRamp, uint32(seed+40+i))
	}
	f.outline(0.8)

	// Roots hanging from the lintel, dark against the sky.
	for i, r := range [][3]float64{{-6.5, 2.5, 7}, {-3, 0.6, 4}, {3.5, 0.8, 5}, {7.5, 3.4, 8}} {
		for s := 0.0; s < r[2]; s += 0.2 {
			px := ax + r[0] + math.Sin(s*0.6+float64(i))*0.6
			x, y := int(px*hdK3), int((ys-archRY+r[1]+s)*hdK3)
			f.put(x, y, caveDirtRamp.pick(0.3-0.2*s/r[2], x, y))
			f.put(x+1, y, caveDirtRamp.pick(0.15, x, y))
		}
	}
	// Grass that has crept in at the lip, and a few blades further down.
	for i, b := range [][3]float64{
		{-10, sill, -0.3}, {-8.2, sill - 0.5, 0.2}, {-6.5, sill, 0.5}, {6.5, sill - 0.5, -0.4},
		{8.5, sill, 0.1}, {10.2, sill + 0.5, 0.4}, {-14.5, ay - 15, -0.3}, {-13, ay - 15, 0.3}, {16.5, ay - 6, 0.3},
	} {
		bx := ax + b[0]
		f.line(bx*hdK3, b[1]*hdK3, (bx+b[2]*3)*hdK3, (b[1]-4.5-hashf(i, 0, seed+50))*hdK3, func(x, y int, t float64) color.RGBA {
			return grassRamp.pick(0.4+0.6*t, x, y)
		})
	}
	for i, p := range f.px {
		if p.A >= 128 {
			c.px[i] = p
		}
	}
	return c
}
