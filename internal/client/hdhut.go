package client

import (
	"image/color"
	"math"
)

// The tribe's hut, drawn at HD and sized by the player: a round wall of
// lashed poles a head taller than a man, a doorway he walks through
// upright (hutDoorH), and a cone of thatch as high again. It stands on the
// middle of its front, like the density-1 hut it replaces (hutCanvas), and
// still blocks only the tile under its door.

// The hut in art pixels: the canvas, the ground point, and its parts.
const (
	hutW, hutH   = 92, 90
	hutCX, hutGY = 46.0, 85.0
	hutWallR     = 33.0 // half the wall's width
	hutWallH     = 45.0 // the wall, from the ground to under the thatch
	hutRoofW     = 41.0 // half the roof's width at the eaves
	hutRoofH     = 31.0 // the apex above the eaves' ends
	hutRoofSag   = 5.5  // how much lower the eaves hang in front
	hutDoorH     = 37.0 // the doorway: 1.2 players high
	hutDoorHalf  = 7.0
	hutTiers     = 7.0
)

// genHutHD draws the hut on an HD canvas and gives its ground point in image
// pixels (see mkUp).
func genHutHD() (*canvas, float64, float64) {
	const k = HDDensity
	c := hdCanvas(hutW, hutH)
	apexY := hutGY - hutWallH - hutRoofH
	px := func(v float64) int { return int(v) }
	// art turns an image pixel into art pixels, at its centre.
	art := func(x, y int) (float64, float64) { return (float64(x) + 0.5) / k, (float64(y) + 0.5) / k }
	eave := func(ax, s float64) float64 { // the front edge of the roof at scale s, in art pixels
		d := (ax - hutCX) / (hutRoofW * s)
		return apexY + s*hutRoofH + s*hutRoofSag*math.Sqrt(math.Max(0, 1-d*d))
	}
	foot := func(ax float64) float64 { // where the wall meets the ground
		u := (ax - hutCX) / hutWallR
		return hutGY - 4.5 + 4.5*math.Sqrt(math.Max(0, 1-u*u))
	}

	// The wall: upright poles round the curve, narrower towards the sides
	// where the wall turns away, each with its own bark, lit from the left.
	const poles = 26
	for y := px((hutGY - hutWallH - 2) * k); y < px((hutGY+1)*k); y++ {
		for x := px((hutCX - hutWallR) * k); x < px((hutCX+hutWallR)*k); x++ {
			ax, ay := art(x, y)
			u := (ax - hutCX) / hutWallR
			if u <= -1 || u >= 1 {
				continue
			}
			th := math.Asin(u)/math.Pi + 0.5 // 0 to 1 round the half of the wall that shows
			fp := th * poles
			pole := px(fp)
			v := fp - float64(pole) // across the pole
			bot := foot(ax) + hashf(pole, 0, 61)*1.6
			if ay >= bot {
				continue
			}
			lit := 0.86 - 0.26*u + 0.16*math.Sqrt(1-u*u) + (hashf(pole, 1, 61)-0.5)*0.2
			if d := ay - eave(ax, 1); d < 7 {
				lit -= 0.42 - d*0.06 // in the shade of the eaves
			}
			if ay > bot-3 {
				lit -= 0.14 // damp at the foot
			}
			col := barkAt(plankRamp, v*2-1, 5, ay*k*1.1+float64(pole)*37, x, y, lit, 70+uint32(pole))
			if v < 0.09 || v > 0.95 {
				col = plankRamp.at(0) // the dark between two poles
			}
			c.put(x, y, col)
		}
	}
	// A knot here and there.
	for i := range 9 {
		ax := hutCX + (hashf(i, 2, 62)*2-1)*hutWallR*0.9
		ay := hutGY - 8 - hashf(i, 3, 62)*26
		if math.Abs(ax-hutCX) < hutDoorHalf+4 {
			continue
		}
		knotHD(c, ax*k, ay*k, 2.2, plankRamp)
	}
	// Three bands of rope lash the poles together, following the wall's curve.
	for _, h := range []float64{9, 21, 33} {
		for x := px((hutCX - hutWallR) * k); x < px((hutCX+hutWallR)*k); x++ {
			ax, _ := art(x, 0)
			y := px((foot(ax) - h) * k)
			if !c.opaque(x, y) {
				continue
			}
			u := (ax - hutCX) / hutWallR
			t := (x + 60) % 4 // the twist of the strands
			l := 0.8 - 0.3*u
			c.put(x, y, thatchRamp.pick(l+0.1-float64(t/3)*0.3, x, y))
			c.put(x, y+1, thatchRamp.pick(l-0.15+float64((t+1)/3%2)*0.2, x, y+1))
			c.put(x, y+2, thatchRamp.pick(l-0.45, x, y+2))
			c.put(x, y+3, plankRamp.at(0)) // its shadow
		}
	}

	// The doorway: dark within, the floor just catching the light.
	doorTop := hutGY - hutDoorH
	for y := px(doorTop * k); y < px((hutGY+1)*k); y++ {
		for x := px((hutCX - hutDoorHalf) * k); x < px((hutCX+hutDoorHalf)*k); x++ {
			ax, ay := art(x, y)
			if ay >= foot(ax) {
				continue
			}
			f := (ay - doorTop) / hutDoorH
			l := f*f*0.55 + (hashf(x/2, y/2, 63)-0.5)*0.06 + (ax-hutCX)/hutDoorHalf*0.05
			c.put(x, y, mix(hex(0x0b0607), hex(0x33211a), l))
		}
	}
	// A hide curtain drawn aside to the left, in folds, with a mark painted on it.
	hide := []float64{
		hutCX - hutDoorHalf, doorTop, hutCX + 4.5, doorTop, hutCX + 1.5, doorTop + 9, hutCX - 2.2, doorTop + 19,
		hutCX - 4.2, doorTop + 29, hutCX - hutDoorHalf, doorTop + 32,
	}
	for i := range hide {
		hide[i] *= k
	}
	c.polygon(hide, func(x, y int) color.RGBA {
		ax, ay := art(x, y)
		d := ax - (hutCX - hutDoorHalf)
		fold := math.Sin(d*1.9 + (ay-doorTop)*0.12)
		l := 0.7 - d*0.035 + fold*0.16 + (vnoise(float64(x)*0.2, float64(y)*0.2, 64)-0.5)*0.2
		if hashf(x, y/3, 65) < 0.08 {
			l -= 0.1 // the hair of the hide
		}
		return loinRamp.pick(l, x, y)
	})
	for _, p := range [][2]float64{{1.6, 6}, {2.6, 6}, {3.6, 6}, {2.6, 7.2}, {2.6, 8.4}, {2.6, 9.6}, {1.4, 12}, {3.8, 12}, {2.6, 13.4}} {
		c.ellipse((hutCX-hutDoorHalf+p[0])*k, (doorTop+p[1])*k, 1.6, 1.6, func(x, y int, nx, ny float64) color.RGBA {
			return paintRamp.at(1 - px(clamp01(nx+ny+0.6)))
		})
	}
	// Stout posts either side and a lintel log across, its cut ends pale.
	lx0, lx1 := px((hutCX-hutDoorHalf-3)*k), px((hutCX+hutDoorHalf+3)*k)
	logHD(c, lx0, lx0+3*k, px((doorTop-1)*k), px(foot(hutCX-hutDoorHalf-1.5)*k), trunkRamp, 0.86, 66)
	logHD(c, lx1-3*k, lx1, px((doorTop-1)*k), px(foot(hutCX+hutDoorHalf+1.5)*k), trunkRamp, 0.74, 67)
	for y := px((doorTop - 3.6) * k); y < px(doorTop*k); y++ {
		for x := lx0 - 2*k; x < lx1+2*k; x++ {
			u := (float64(y)+0.5-(doorTop-1.8)*k)/(1.8*k) + 0
			c.put(x, y, barkAt(trunkRamp, u, 5, float64(x)*1.1, x, y, 0.9, 68))
		}
	}
	for _, ex := range []int{lx0 - 2*k, lx1 + 2*k - 1} {
		c.ellipse(float64(ex), (doorTop-1.8)*k, 2.4, 1.8*k, func(x, y int, nx, ny float64) color.RGBA {
			return woodRamp.pick(0.8-(nx*nx+ny*ny)*0.5, x, y)
		})
	}

	// Stones packed round the foot of the wall.
	for i, sx := range []float64{-30, -25, -19.5, -13, 12.5, 18, 24, 29.5} {
		ax := hutCX + sx
		rx, ry := 2.6+hashf(i, 0, 69)*1.4, 1.9+hashf(i, 1, 69)*0.8
		c.ellipse(ax*k, (foot(ax)-ry*0.5)*k, rx*k, ry*k, func(x, y int, nx, ny float64) color.RGBA {
			l := lum(nx, ny)*0.85 + 0.1 - float64(i%3)*0.05 + (vnoise(float64(x)*0.35, float64(y)*0.35, 69)-0.5)*0.25
			if ny < -0.3 && vnoise(float64(x)*0.2, float64(y)*0.3, 71) > 0.6 {
				return mossRamp.pick(l+0.2, x, y)
			}
			return stoneRamp.pick(l, x, y)
		})
	}
	// A hide shield hung on the wall: a rim, a ring of dots, a bone boss.
	c.ellipse((hutCX-19)*k, (hutGY-22)*k, 6.2*k, 8*k, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		switch {
		case d > 0.8:
			return loinRamp.pick(0.25+lum(nx, ny)*0.2, x, y)
		case d < 0.05:
			return mix(colBoneSh, colBone, clamp01(0.5-nx-ny))
		case d < 0.09:
			return colBoneSh
		case d < 0.34:
			return mix(hex(0x9a2a18), hex(0x5c160e), clamp01(0.4+nx*0.4+ny*0.5))
		case d > 0.44 && d < 0.6 && px((math.Atan2(ny, nx)+math.Pi)*3.2)%2 == 0:
			return paintRamp.at(1)
		}
		return loinRamp.pick(lum(nx, ny)*0.7+0.3+(hashf(x/2, y/2, 72)-0.5)*0.12, x, y)
	})
	// A spear leant by the door, its stone point bound on.
	drawLimbHD(c, limb{(hutCX + 19) * k, (hutGY - 1) * k, (hutCX + 17.5) * k, (hutGY - 20) * k, (hutCX + 16) * k, (hutGY - 40) * k, 1.6, 1.4}, spearShaft, 0.9, 73)
	c.polygon([]float64{(hutCX + 14.9) * k, (hutGY - 40) * k, (hutCX + 15.7) * k, (hutGY - 46.5) * k, (hutCX + 17.3) * k, (hutGY - 40) * k}, func(x, y int) color.RGBA {
		return stoneRamp.pick(0.85-(float64(x)/k-(hutCX+14.9))*0.25, x, y)
	})
	for d := range 4 {
		for x := px((hutCX + 15.2) * k); x < px((hutCX+17.1)*k); x++ {
			c.put(x, px((hutGY-40)*k)+d, ropeRamp.at(2-d%2-d/3))
		}
	}
	// Two clay pots by the door, the big one painted.
	pot := func(ax, r float64, seed uint32) {
		cy := foot(ax) - r*0.9
		c.ellipse(ax*k, cy*k, r*k, r*1.05*k, func(x, y int, nx, ny float64) color.RGBA {
			l := lum(nx, ny)*0.9 + 0.08 + (hashf(x/2, y/2, seed)-0.5)*0.08
			if ny > -0.25 && ny < 0 && px((nx+2)*5)%2 == 0 {
				return paintRamp.at(1)
			}
			return potRamp.pick(l, x, y)
		})
		c.ellipse(ax*k, (cy-r*1.05)*k, r*0.62*k, r*0.26*k, func(x, y int, nx, ny float64) color.RGBA {
			if nx*nx+ny*ny < 0.45 {
				return potRamp.at(0) // its mouth
			}
			return potRamp.pick(0.8-nx*0.3, x, y)
		})
	}
	pot(hutCX+25.5, 3.3, 74)
	pot(hutCX+12, 4.2, 75)

	// The roof: a cone of thatch laid in ragged tiers, each casting a shadow
	// on the one below, every straw a stroke running down the slope.
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			ax, ay := art(x, y)
			a := (ax - hutCX) / hutRoofW
			d := math.Abs(a)
			lo, hi := math.Max(d, 0.02), 1.0
			if ay < eave(ax, lo) {
				continue
			}
			var s float64
			if over := ay - eave(ax, hi); over > 0 { // the ragged eaves
				strand := px(math.Floor(a * 90))
				if d >= 1 || over > 0.5+hashf(strand, 9, 65)*3.2 {
					continue
				}
				s = 1
			} else {
				for range 16 {
					mid := (lo + hi) / 2
					if eave(ax, mid) < ay {
						lo = mid
					} else {
						hi = mid
					}
				}
				s = (lo + hi) / 2
			}
			ang := a / s // -1 at the left edge of the cone, 1 at the right
			tf := s*hutTiers + (hashf(px(math.Floor(ang*(8+s*34))), 7, 65)-0.5)*0.3
			if s >= 1 {
				tf = hutTiers - 0.02
			}
			tier := math.Floor(math.Max(0, tf))
			f := tf - tier
			strand := px(math.Floor(ang * (12 + (tier+1)*13)))
			l := 0.6 - ang*0.3 + (hashf(strand, px(tier), 66)-0.5)*0.26 + (f-0.5)*0.3
			l += (hashf(strand*3+x%2, y/5, 67) - 0.5) * 0.14 // straws
			if hashf(x, y/2, 67) < 0.05 {
				l += 0.16 // stray bright ones
			}
			if tier > 0 {
				switch {
				case f < 0.1:
					l -= 0.5
				case f < 0.24:
					l -= 0.2
				}
			}
			if s > 0.985 {
				l -= 0.1
			}
			c.put(x, y, thatchRamp.pick(l, x, y))
		}
	}
	// The roof poles cross and stand out at the peak, bound with rope.
	for i, p := range [][2]float64{{-6.5, -5}, {7, -4.4}, {0.8, -6}} {
		drawLimbHD(c, limb{(hutCX + p[0]*0.15) * k, (apexY + 5) * k, (hutCX + p[0]*0.5) * k, (apexY + 1) * k, (hutCX + p[0]) * k, (apexY + p[1]) * k, 1.9, 1.3}, trunkRamp, 0.75+float64(i)*0.07, 76+uint32(i))
	}
	for y := px((apexY + 2.4) * k); y < px((apexY+4.4)*k); y++ {
		for x := px((hutCX - 2.6) * k); x < px((hutCX+2.8)*k); x++ {
			c.put(x, y, ropeRamp.pick(0.85-float64(y-px((apexY+2.4)*k))*0.13-float64(x%3)*0.1, x, y))
		}
	}
	// Teeth, bones and red feathers strung along the eaves.
	for i, dx := range []float64{-34, -27.5, -21, -14.5, -8, 8, 14.5, 21, 27.5, 34} {
		ax := hutCX + dx
		x, y := px(ax*k), px(eave(ax, 1)*k)+2
		if i%3 == 1 { // a feather
			for j := range 11 {
				w := 2 - j/5
				for q := -w / 2; q <= w/2; q++ {
					c.put(x+q, y+j, mix(hex(0xd83a26), hex(0x7a160e), float64(j)/11+float64(q+1)*0.1))
				}
			}
			continue
		}
		for j := range 8 { // a tooth: a pale point hanging down
			w := 2 - j/3
			for q := 0; q < w; q++ {
				col := colBone
				if q == w-1 && w > 1 || j > 5 {
					col = colBoneSh
				}
				c.put(x+q, y+j, col)
			}
		}
	}
	// A horned skull over the door: these people do not want visitors.
	sx, sy := hutCX*k, (doorTop-9.5)*k
	for _, side := range []float64{-1, 1} {
		drawLimbHD(c, limb{sx + side*5, sy - 2, sx + side*14, sy - 3, sx + side*19, sy - 14, 2.4, 0.8}, ramp{colBoneSh, colBoneSh, colBone, hex(0xfffaf0)}, 0.95, 77)
	}
	bone := ramp{hex(0xa89c84), colBoneSh, colBone, hex(0xfffaf0)}
	c.ellipse(sx, sy, 8.6, 7.6, func(x, y int, nx, ny float64) color.RGBA { return bone.pick(lum(nx, ny)*0.8+0.25, x, y) })
	c.ellipse(sx, sy+7.5, 5, 4.6, func(x, y int, nx, ny float64) color.RGBA { return bone.pick(lum(nx, ny)*0.7+0.2, x, y) })
	for _, side := range []float64{-1, 1} {
		c.ellipse(sx+side*3.6, sy+0.5, 2.3, 2.6, func(x, y int, nx, ny float64) color.RGBA { return colInk })
	}
	c.ellipse(sx, sy+5.5, 1.2, 1.8, func(x, y int, nx, ny float64) color.RGBA { return colInk })
	for q := -3; q <= 3; q += 2 { // the teeth
		c.put(px(sx)+q, px(sy)+10, colInk)
		c.put(px(sx)+q, px(sy)+11, colInk)
	}
	c.outline(0.85)
	return c, hutCX * k, (hutGY - 1) * k
}
