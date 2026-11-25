package client

import (
	"image/color"
	"math"
)

// The things people build, at HD: palisade and stone walls, the door, the
// plank floor, the bed, the chests, the fire rings, posts and totems. Drawn
// in image pixels (hdK3 to an art pixel) on canvases marked HD, each with
// the footprint and anchor of its density-1 drawing.

// logHD is an upright round log between columns x0 and x1 from row top to
// bottom: bark lit from the left, furrowed along its length.
func logHD(c *canvas, x0, x1, top, bottom int, rp ramp, lit float64, seed uint32) {
	half := float64(x1-x0) / 2
	for y := top; y <= bottom; y++ {
		for x := x0; x < x1; x++ {
			u := (float64(x) + 0.5 - float64(x0) - half) / half
			c.put(x, y, barkAt(rp, u, half, float64(y)*1.1+float64(seed)*13, x, y, lit, seed))
		}
	}
}

// grainHD shades a plank: l lit, with a grain of long streaks running along
// (dx, dy) = (1, 0) for a lying plank or (0, 1) for an upright one, and a
// knot here and there.
func grainHD(rp ramp, x, y int, l float64, upright bool, seed uint32) color.RGBA {
	a, b := float64(x), float64(y)
	if upright {
		a, b = b, a
	}
	g := vnoise(a*0.07+float64(seed)*3.1, b*0.8, seed)
	switch {
	case g < 0.3:
		l -= 0.14
	case g > 0.72:
		l += 0.08
	}
	if k := vnoise(a*0.16, b*0.2, seed+1); k > 0.86 {
		l -= 0.3 // a knot
	} else if k > 0.8 {
		l += 0.05
	}
	return rp.pick(l+(hashf(x, y, seed+2)-0.5)*0.06, x, y)
}

// nailHD is a nail head at (x, y): a dark dot with a glint.
func nailHD(c *canvas, x, y int) {
	c.put(x, y, bandRamp.at(3))
	c.put(x+1, y, bandRamp.at(1))
	c.put(x, y+1, bandRamp.at(1))
	c.put(x+1, y+1, bandRamp.at(0))
}

// ropeHD winds a rope three pixels thick along row y over whatever is
// opaque between x0 and x1, sagging by sag pixels in the middle.
func ropeHD(c *canvas, x0, x1, y int, sag float64) {
	for x := x0; x < x1; x++ {
		f := float64(x-x0) / float64(x1-x0)
		yy := y + int(math.Round(sag*math.Sin(f*math.Pi)))
		if !c.opaque(x, yy) && !c.opaque(x, yy+2) {
			continue
		}
		t := (x + 60) % 4 // the twist of the strands
		c.put(x, yy, ropeRamp.at(2-t/3))
		c.put(x, yy+1, ropeRamp.at(1+(t+1)/3%2))
		c.put(x, yy+2, ropeRamp.at(0))
		c.put(x, yy+3, trunkRamp.at(0)) // its shadow
	}
}

func genWallHD() sprite {
	const k = HDDensity
	c := hdCanvas(24, 30)
	const by = 26 * k
	for i := 0; i < 5; i++ {
		x0 := (1 + i*5) * k
		top := (4 + int(hashf(i, 0, 910)*4)) * k
		// The dark between this stake and the next.
		for y := top + 3*k; y <= by && i < 4; y++ {
			for x := x0 + 4*k; x < x0+5*k; x++ {
				c.put(x, y, trunkRamp.at(0))
			}
		}
		logHD(c, x0, x0+4*k, top+2*k, by, trunkRamp, 0.82, 911+uint32(i))
		// The sharpened point: three cut faces of pale wood.
		for y := top; y < top+2*k+2; y++ {
			f := float64(y-top) / float64(2*k+2) // 0 at the tip
			half := 0.8 + f*5.4
			cx := float64(x0) + 6
			for x := int(math.Floor(cx - half)); x < int(math.Ceil(cx+half)); x++ {
				u := (float64(x) + 0.5 - cx) / half
				l := 0.75 - u*0.45 + (hashf(x, y, 912)-0.5)*0.1
				if math.Abs(u) < 0.18 {
					l -= 0.2 // the ridge between two faces
				}
				c.put(x, y, woodRamp.pick(l, x, y))
			}
		}
	}
	ropeHD(c, 3, 72, 11*k, 0)
	ropeHD(c, 3, 72, 21*k, 0)
	c.outline(0.8)
	return mkUp(c, 12*k, by)
}

func genStoneWallHD() (*canvas, float64, float64) {
	const k = HDDensity
	c := hdCanvas(24, 30)
	const by = 26 * k
	tops := [6]int{6, 9, 6, 9, 6, 9} // battlements: a few stones higher than the rest
	const B = 4 * k                  // a stone, with its mortar
	for y := 4 * k; y < by+k; y++ {
		for x := 0; x < 24*k; x++ {
			if y < tops[x/B]*k {
				continue
			}
			row := (y - 4*k) / B
			sx := x + B/2*(row%2)
			bx, by4 := sx%B, (y-4*k)%B
			stone := hashf(sx/B, row, 920)
			l := 0.74 - float64(x)/(24*k)*0.32 + (stone-0.5)*0.26
			l += (hashf(x, y, 923) - 0.5) * 0.08
			l += (vnoise(float64(x)*0.4, float64(y)*0.4, 924) - 0.5) * 0.14
			round := (bx == 0 || bx == B-2) && (by4 == 0 || by4 == B-2) // worn corners
			switch {
			case bx == B-1 || by4 == B-1 || round:
				l = 0.12 + (hashf(x, y, 925)-0.5)*0.1 // mortar, in shadow
			case bx == 0 || by4 == 0:
				l += 0.16 // the lit edges
			case bx == B-2 || by4 == B-2:
				l -= 0.16
			case hashf(x/2, y/2, 926) < 0.03:
				l -= 0.2 // a pit
			}
			if stone > 0.8 && by4 > 2 && by4 < B-3 { // a crack across the stone
				if cy := 3 + int(hashf(sx/B, row, 927)*4) + int(vnoise(float64(x)*0.5, 0, 928)*3); by4 == cy {
					l -= 0.28
				}
			}
			if y > by-3*k {
				l -= 0.1
			}
			col := stoneRamp.pick(l, x, y)
			// Moss creeping up from the ground and in the joints.
			m := vnoise(float64(x)*0.2, float64(y)*0.25, 921)
			if up := float64(by-y) / k; m > 0.42+up*0.05 || (bx >= B-2 || by4 >= B-2) && m > 0.62 {
				col = mossRamp.pick(l+0.35+(hashf(x, y, 922)-0.5)*0.4, x, y)
			}
			c.put(x, y, col)
		}
	}
	c.outline(0.8)
	return c, 12 * k, by
}

func genDoorHD() sprite {
	const k = HDDensity
	c := hdCanvas(24, 30)
	// The frame: two posts and a lintel, like the wall's.
	logHD(c, 1*k, 4*k, 3*k, 27*k-1, trunkRamp, 0.8, 930)
	logHD(c, 20*k, 23*k, 3*k, 27*k-1, trunkRamp, 0.75, 931)
	for y := 2 * k; y < 5*k; y++ {
		for x := 1 * k; x < 23*k; x++ {
			u := (float64(y-2*k)+0.5)/(1.5*k) - 1
			c.put(x, y, barkAt(trunkRamp, u, 4.5, float64(x)*1.1, x, y, 0.85, 932))
		}
	}
	// The door: upright planks with their grain, each with a lit edge.
	const pw = 4 * k
	for x := 4 * k; x < 20*k; x++ {
		for y := 5 * k; y < 27*k; y++ {
			px := (x - 4*k) % pw
			plank := (x - 4*k) / pw
			l := 0.52 + float64(plank%2)*0.07 - float64(y-5*k)*0.003 + (hashf(plank, 0, 933)-0.5)*0.1
			switch px {
			case 0:
				l -= 0.32
			case 1:
				l += 0.12
			case pw - 1:
				l -= 0.1
			}
			c.put(x, y, grainHD(plankRamp, x, y, l, true, 934+uint32(plank)))
		}
	}
	// Two battens across, nailed to every plank, and a brace between them.
	batten := func(y0 int) {
		for y := y0; y < y0+2*k; y++ {
			for x := 4*k + 1; x < 20*k-1; x++ {
				l := 0.62 - float64(y-y0)*0.05
				if y == y0+2*k-1 {
					l = 0.2
				}
				c.put(x, y, grainHD(plankRamp, x, y, l, false, 940))
			}
		}
		for p := range 4 {
			nailHD(c, 4*k+p*pw+5, y0+2)
		}
	}
	for i := 0; i <= 44; i++ { // the brace, from the low batten to the high one
		f := float64(i) / 44
		x, y := int(lerpf(5*k+2, 18*k, f)), int(lerpf(22*k-1, 9*k+3, f))
		for d := range 4 {
			c.put(x, y+d, plankRamp.at(3-d))
		}
	}
	batten(7 * k)
	batten(22 * k)
	// The iron ring to pull it by, on its plate.
	c.ellipse(16.5*k, 15*k, 3, 3.6, func(x, y int, nx, ny float64) color.RGBA { return bandRamp.pick(0.3+lum(nx, ny)*0.5, x, y) })
	for a := 0.0; a < 2*math.Pi; a += 0.1 {
		x, y := int(16.5*k+math.Cos(a)*3.6), int(17.2*k+math.Sin(a)*4.4)
		c.put(x, y, bandRamp.pick(0.75-math.Sin(a)*0.3-math.Cos(a)*0.2, x, y))
	}
	c.outline(0.7)
	return mkUp(c, 12*k, 26*k)
}

func genFloorHD() sprite {
	const k = HDDensity
	c := hdCanvas(TileW, TileH)
	for y := 0; y < TileH*k; y++ {
		row := y / (3 * k)
		ry := y % (3 * k)
		for x := 0; x < TileW*k; x++ {
			sx := x + row*7*k
			plank := sx / (9 * k)
			px := sx % (9 * k)
			l := 0.5 + (hashf(row, plank, 91)-0.5)*0.3
			switch {
			case ry == 3*k-1 || px == 0:
				l = 0.08 // the gaps between planks
			case ry == 0 || px == 1:
				l += 0.12 // the worn, lit edge
			case ry == 3*k-2:
				l -= 0.12
			}
			col := grainHD(plankRamp, sx, y+row*40, l, false, 92+uint32(row))
			if (px == 3 || px == 9*k-4) && (ry == 2 || ry == 5) { // nails at the plank ends
				col = bandRamp.at(1 + ry/5*2)
			}
			c.put(x, y, col)
		}
	}
	return mkUp(c, 0, 0)
}

func genBedHD() sprite {
	const k = HDDensity
	c := hdCanvas(30, 20)
	const cx, by = 15 * k, 16 * k
	// The frame of logs, lying: side rails, and the round ends of the cross logs.
	for y := 6 * k; y < 17*k; y++ {
		for x := 1 * k; x < 29*k; x++ {
			u := (float64(y-6*k)+0.5)/(5.5*k) - 1
			c.put(x, y, barkAt(trunkRamp, u*0.6, 6, float64(x)*1.1, x, y, 0.78, 919))
		}
	}
	// The mattress: dry grass, in short strokes lying every way.
	for y := 6 * k; y < 14*k; y++ {
		for x := 3 * k; x < 27*k; x++ {
			l := 0.5 + (vnoise(float64(x)*0.23, float64(y)*0.3, 920)-0.5)*0.5
			// Stalks: thin streaks along one of two slants, by patch.
			slant := 1
			if vnoise(float64(x)*0.09, float64(y)*0.11, 921) > 0.5 {
				slant = -1
			}
			switch s := hashf((x+slant*y)/1, (x-slant*y)/9, 922); {
			case s < 0.2:
				l += 0.25
			case s > 0.84:
				l -= 0.25
			}
			if y >= 13*k {
				l -= 0.2 // the edge, in the rail's shade
			}
			c.put(x, y, strawRamp.pick(l, x, y))
		}
	}
	// The pillow: a rolled hide, with a seam and a crease.
	c.ellipse(7*k, 8*k, 4*k, 2.4*k, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if math.Abs(ny-0.25*math.Sin(nx*3)) < 0.09 {
			l -= 0.3
		}
		if d := nx*nx + ny*ny; d > 0.72 && d < 0.8 && (x+y)%3 != 0 {
			l -= 0.25 // stitches round the edge
		}
		return mix(hex(0xb0a284), hex(0xf4eedb), l)
	})
	// The front rail, with the cut ends of the cross logs at each side.
	for y := 15 * k; y < 17*k; y++ {
		for x := 1 * k; x < 29*k; x++ {
			c.put(x, y, barkAt(trunkRamp, float64(y-15*k)/3-1, 3, float64(x), x, y, 0.5, 918))
		}
	}
	for _, x := range []float64{2.2 * k, 27.8 * k} {
		c.ellipse(x, 15.2*k, 3.4, 4.2, func(px, py int, nx, ny float64) color.RGBA { return woodEndHD(px, py, nx, ny, 3, 7) })
	}
	c.outline(0.8)
	return mkUp(c, cx, by)
}

// chestHD draws a chest of planks w art pixels wide in a 24 by 22 canvas,
// bound in iron (band), closed with a lock or open on its dark inside.
func chestHD(open bool, wood, lid, band ramp, lock ramp, seed uint32) *canvas {
	const k = HDDensity
	c := hdCanvas(24, 22)
	const x0, x1, top, base = 2 * k, 22 * k, 10 * k, 20*k + 2
	bandAt := func(x int) bool { return x >= x0+3*k && x < x0+5*k || x >= x1-5*k && x < x1-3*k }
	bandCol := func(x, y int, l float64) color.RGBA {
		col := band.pick(l+0.1, x, y)
		if (y+2)%7 == 0 && (x-x0)%(2*k) == 2 { // rivets
			col = band.at(len(band) - 1)
		}
		return col
	}
	// The box: three planks a side, lit from the left.
	for y := top; y <= base; y++ {
		for x := x0; x < x1; x++ {
			l := 0.72 - float64(x-x0)/float64(x1-x0)*0.42
			switch (y - top) % (3*k + 2) {
			case 3*k + 1:
				l = 0.1 // a seam between planks
			case 0:
				l += 0.1
			}
			col := grainHD(wood, x, y+(y-top)/(3*k+2)*50, l, false, seed)
			if bandAt(x) {
				col = bandCol(x, y, l)
			}
			c.put(x, y, col)
		}
	}
	if open {
		// The lid tipped back, seen from inside, over the dark of the box.
		for y := 2 * k; y < top; y++ {
			f := float64(y-2*k) / float64(top-2*k)
			inset := int((1 - f) * 2 * k)
			for x := x0 + inset; x < x1-inset; x++ {
				l := 0.42 - float64(x-x0)/float64(x1-x0)*0.25 + f*0.1
				col := grainHD(lid, x, y, l, false, seed+3)
				if bandAt(x) {
					col = bandCol(x, y, l-0.15)
				}
				if y >= top-2*k && x > x0+2 && x < x1-3 {
					col = mix(hex(0x1a0e08), hex(0x2c1a10), f) // the inside
				}
				c.put(x, y, col)
			}
		}
	} else {
		// The lid: a rounded top over the box, and the lock.
		for y := top - 5*k; y < top; y++ {
			f := float64(y-(top-5*k)) / (5 * k) // 0 at the top
			inset := int(2.2 * k * (1 - math.Sqrt(1-(1-f)*(1-f))))
			for x := x0 + inset; x < x1-inset; x++ {
				l := 0.92 - float64(x-x0)/float64(x1-x0)*0.45 - f*0.25
				if y == top-1 {
					l = 0.12 // the lid's lip
				}
				col := grainHD(lid, x, y, l, false, seed+3)
				if bandAt(x) {
					col = bandCol(x, y, l)
				}
				c.put(x, y, col)
			}
		}
		c.ellipse(12*k, float64(top)+3, 3.6, 4.4, func(x, y int, nx, ny float64) color.RGBA { return lock.pick(0.25+lum(nx, ny)*0.7, x, y) })
		c.ellipse(12*k, float64(top)+3, 1.1, 1.8, func(int, int, float64, float64) color.RGBA { return hex(0x1a0e08) })
	}
	c.outline(0.8)
	return c
}

// stoneRingHD sets n stones in a ring rx by ry round (cx, cy): those of the
// back half or the front half.
func stoneRingHD(c *canvas, cx, cy, rx, ry float64, n int, r float64, front bool, seed uint32) {
	for i := 0; i < n; i++ {
		a := 2*math.Pi*float64(i)/float64(n) + 0.2
		if (math.Sin(a) >= 0) != front {
			continue
		}
		v := 0.85 + hashf(i, 0, seed)*0.3
		stoneHD(c, cx+math.Cos(a)*rx, cy+math.Sin(a)*ry, r*v, r*0.78*v, 0.9+hashf(i, 1, seed)*0.15, stoneRamp, seed+uint32(i))
	}
}

// fireLogHD lays a charred log from (x0, y0) to (x1, y1), w pixels half
// thick, its cut ends pale.
func fireLogHD(c *canvas, x0, y0, x1, y1, w float64, lit float64, seed uint32) {
	drawLimbHD(c, limb{x0, y0, (x0 + x1) / 2, (y0 + y1) / 2, x1, y1, w, w}, trunkRamp, lit, seed)
	for _, e := range [][2]float64{{x0, y0}, {x1, y1}} {
		c.ellipse(e[0], e[1], w*0.75, w, func(x, y int, nx, ny float64) color.RGBA {
			return woodRamp.pick(0.75-(nx*nx+ny*ny)*0.5, x, y)
		})
	}
}

func genCampfireHD() (back, front sprite) {
	const k = HDDensity
	const cx, cy = 15.0 * k, 10.0 * k
	b, f := hdCanvas(30, 18), hdCanvas(30, 18)
	stoneRingHD(b, cx, cy, 10*k, 4.5*k, 11, 2.6*k, false, 200)
	b.ellipse(cx, cy, 7.5*k, 3.2*k, func(x, y int, nx, ny float64) color.RGBA {
		if hashf(x/2, y/2, 200) < 0.1 {
			return fireRamp[1+int(hashf(x, y, 201)*2)] // embers
		}
		if hashf(x, y, 202) < 0.06 {
			return ashRamp[2]
		}
		return ashRamp.pick(0.3+nx*0.2+(vnoise(float64(x)*0.3, float64(y)*0.3, 203)-0.5)*0.5, x, y)
	})
	fireLogHD(b, cx-7*k, cy+2*k, cx+7*k, cy-2*k, 1.5*k, 0.7, 7)
	fireLogHD(b, cx-7*k, cy-2*k, cx+7*k, cy+2*k, 1.5*k, 0.82, 8)
	stoneRingHD(f, cx, cy, 10*k, 4.5*k, 11, 2.6*k, true, 200)
	b.outline(0.7)
	f.outline(0.7)
	return mkUp(b, cx, cy), mkUp(f, cx, cy)
}

func genBonfireHD() sprite {
	const k = HDDensity
	c := hdCanvas(40, 24)
	stoneRingHD(c, 20*k, 14*k, 15*k, 6.5*k, 12, 3*k, false, 210)
	for i, l := range [][4]float64{{8, 17, 32, 9}, {8, 9, 32, 17}, {12, 19, 28, 7}} {
		fireLogHD(c, l[0]*k, l[1]*k, l[2]*k, l[3]*k, 1.6*k, 0.7+float64(i)*0.06, 11+uint32(i))
	}
	stoneRingHD(c, 20*k, 14*k, 15*k, 6.5*k, 12, 3*k, true, 210)
	c.outline(0.7)
	return mkUp(c, 20*k, 14*k)
}

func genPostHD() sprite {
	const k = HDDensity
	c := hdCanvas(6, 22)
	logHD(c, 2*k, 5*k, 2*k, 21*k-1, trunkRamp, 0.82, 950)
	c.polygon([]float64{2 * k, 2*k + 1, 3.5 * k, 0, 5 * k, 2*k + 1}, func(x, y int) color.RGBA {
		return woodRamp.pick(0.8-(float64(x)-2*k)/9*0.6, x, y)
	})
	knotHD(c, 3.4*k, 12*k, 1.6, trunkRamp)
	c.outline(0.6)
	return mkUp(c, 3*k, 21*k)
}

func genTotemHD() sprite {
	const k = HDDensity
	c := hdCanvas(16, 46)
	logHD(c, 4*k, 12*k, 6*k, 45*k-1, trunkRamp, 0.85, 960)
	// Three carved, painted faces stacked up the post: a brow, sunken
	// painted eyes, a nose and a mouth of teeth.
	for i, fy := range []int{10, 22, 34} {
		paint := []color.RGBA{hex(0xc83020), hex(0xf0e8d0), hex(0x2a6ab0)}[i]
		y0 := fy * k
		for x := 4*k + 2; x < 12*k-2; x++ { // the brow, and the groove above the face
			c.put(x, y0-2*k, trunkRamp.at(0))
			c.put(x, y0-2*k+1, trunkRamp.at(4))
			c.put(x, y0-k-1, trunkRamp.at(1))
		}
		for _, ex := range []float64{6, 10} {
			c.ellipse(ex*k, float64(y0), 3.4, 3.4, func(x, y int, nx, ny float64) color.RGBA { return trunkRamp.at(0) })
			c.ellipse(ex*k, float64(y0), 2.6, 2.6, func(x, y int, nx, ny float64) color.RGBA {
				return mix(paint, colBlack, clamp01((nx+ny)*0.25+0.1))
			})
			c.ellipse(ex*k, float64(y0), 0.9, 0.9, func(int, int, float64, float64) color.RGBA { return hex(0x1a0c08) })
		}
		for y := y0; y < y0+2*k+2; y++ { // the nose: a lit ridge
			c.put(8*k-2, y, trunkRamp.at(4))
			c.put(8*k-1, y, trunkRamp.at(3))
			c.put(8*k, y, trunkRamp.at(1))
			c.put(8*k+1, y, trunkRamp.at(0))
		}
		for x := 5 * k; x < 11*k; x++ { // the mouth, with teeth
			for d := range 4 {
				col := hex(0x1a0c08)
				if (d == 1 || d == 2) && x%3 != 0 && x > 5*k+1 && x < 11*k-2 {
					col = mix(paint, hex(0xf0e8d0), 0.6)
				}
				c.put(x, y0+3*k+d, col)
			}
		}
		for x := 4 * k; x < 12*k; x++ { // a painted band under each face
			if u := x - 4*k; (u/3)%2 == 0 {
				c.put(x, y0+5*k+1, paint)
				c.put(x, y0+5*k+2, mix(paint, colBlack, 0.3))
			}
		}
	}
	// Wings at the top, feathered.
	for _, side := range []float64{-1, 1} {
		x0 := 8*k + side*4*k
		c.polygon([]float64{x0, 6 * k, x0 + side*4*k, 2 * k, x0 + side*4*k, 8 * k}, func(x, y int) color.RGBA {
			l := 0.25
			if (y+int(math.Abs(float64(x)-x0))/2)%4 == 0 {
				l = 0.6 // the gaps between feathers
			}
			return mix(hex(0xe04030), hex(0x70140c), l+float64(y-2*k)/18*0.3)
		})
	}
	c.outline(0.8)
	return mkUp(c, 8*k, 44*k)
}
