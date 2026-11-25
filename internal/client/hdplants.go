package client

import (
	"image/color"
	"math"
	"math/rand/v2"
)

// The small things of the ground at HD: bushes, grass, ferns, flowers,
// stumps, twigs and stones. Each is drawn in image pixels, K to an art pixel,
// on a canvas marked HD, and anchored where its density-1 drawing was.

const hdK3 = float64(HDDensity)

// hdCanvas is a canvas of w by h art pixels drawn in HD image pixels.
func hdCanvas(w, h int) *canvas {
	c := newCanvas(w*HDDensity, h*HDDensity)
	c.dens = HDDensity
	return c
}

// scaled is blobs K times bigger: art pixels to image pixels.
func scaledBlobs(bs []blob) []blob {
	out := make([]blob, len(bs))
	for i, b := range bs {
		out[i] = blob{b.x * hdK3, b.y * hdK3, b.r * hdK3}
	}
	return out
}

// bladeHD is one blade of grass from (x0, y0) up h pixels, leaning over by
// lean at its tip, w pixels wide at the foot and tapering to a point; its
// tone runs from lo at the foot to hi at the tip, the left edge lit.
func bladeHD(c *canvas, x0, y0, lean, h, w float64, rp ramp, lo, hi float64) {
	n := int(h*2) + 2
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := x0+lean*t*t, y0-h*t
		half := max(0.5, w/2*math.Pow(1-t, 0.6))
		for px := int(math.Floor(x - half)); px <= int(math.Floor(x+half-0.01)); px++ {
			u := (float64(px) + 0.5 - x) / half
			l := lo + (hi-lo)*t - u*0.14
			py := int(math.Floor(y))
			c.put(px, py, rp.pick(l, px, py))
		}
	}
}

// berryHD is a round berry r pixels in radius at (x, y), with a highlight
// and a shadow on the leaves under it.
func berryHD(c *canvas, x, y, r float64, rp ramp, shade color.RGBA) {
	c.ellipse(x+0.6, y+1.4, r, r, func(int, int, float64, float64) color.RGBA { return shade })
	c.ellipse(x, y, r, r, func(px, py int, nx, ny float64) color.RGBA {
		return rp.pick(lum(nx, ny)*0.85+0.05, px, py)
	})
	c.put(int(x-r*0.45), int(y-r*0.5), rp.at(len(rp)-1))
}

func genBushHD(berries bool) sprite {
	c := hdCanvas(28, 22)
	rp := bushRamp
	if !berries {
		rp = bushDryRamp
	}
	blobs := scaledBlobs([]blob{{8, 14, 6}, {20, 14, 6}, {14, 11, 8}, {10, 8, 5}, {18, 8, 5}, {14, 15, 6}})
	leafClusterHD(c, blobs, 14*hdK3, 11*hdK3, 12*hdK3, 9*hdK3, rp, 160, leafSmall)
	if berries {
		for i, b := range [][2]float64{{7, 11}, {12, 7}, {18, 10}, {22, 14}, {10, 15}, {15, 13}, {19, 5}, {5, 15}} {
			x, y := (b[0]+1)*hdK3, (b[1]+1)*hdK3
			berryHD(c, x, y, 3.1, berryRamp, rp[0])
			if i%2 == 0 { // some hang in pairs
				berryHD(c, x+4, y+2.5, 2.4, berryRamp, rp[0])
			}
		}
	} else { // the bare stalks the berries hung on
		for _, b := range [][2]float64{{7, 11}, {12, 7}, {18, 10}, {22, 14}, {15, 13}} {
			x, y := int((b[0]+1)*hdK3), int((b[1]+1)*hdK3)
			c.put(x, y, trunkRamp[2])
			c.put(x, y+1, trunkRamp[1])
			c.put(x+1, y+1, trunkRamp[2])
		}
	}
	c.outline(0.75)
	return mkUp(c, 14*hdK3, 19*hdK3)
}

func genTuftHD(full bool) sprite {
	rng := rand.New(rand.NewPCG(11, 12))
	c := hdCanvas(22, 20)
	const bx, by = 11 * hdK3, 18 * hdK3
	// Three rows of blades, back to front, each row lighter.
	for row := 0; row < 3; row++ {
		n := 8 + row*2
		for i := 0; i < n; i++ {
			x0 := bx + (-5+(float64(i)+rng.Float64()*0.6)*10/float64(n))*hdK3
			h := (13 - float64(row)*2.5 + rng.Float64()*3) * hdK3
			lean := (x0-bx)*0.55 + (rng.Float64()-0.5)*2.5*hdK3
			lo, hi := 0.05+float64(row)*0.12, 0.7+float64(row)*0.12
			if !full {
				h = (2 + rng.Float64()*2) * hdK3
				lean *= 0.3
				lo, hi = 0.1, 0.35
			}
			bladeHD(c, x0, by, lean, h, 2.4+rng.Float64()*1.2, strawRamp, lo, hi)
			if full && rng.Float64() < 0.3 { // a seed head, bent over
				tx, ty := x0+lean, by-h
				for j := range 5 {
					fj := float64(j)
					px, py := int(tx+math.Copysign(fj*0.8, lean)), int(ty+fj*0.9-1)
					c.put(px, py, strawRamp.at(4))
					c.put(px, py+1, strawRamp.at(3))
					if j%2 == 0 {
						c.put(px+1, py, strawRamp.at(3))
					}
				}
			}
		}
	}
	for x := int(bx - 5*hdK3); x < int(bx+5*hdK3); x++ {
		if c.opaque(x, int(by)-1) {
			c.put(x, int(by), strawRamp[0])
			c.put(x, int(by)+1, strawRamp[0])
		}
	}
	c.outline(0.4)
	return mkUp(c, bx, by)
}

// woodEndHD shades the sawn end of a log or stump at (nx, ny) in its
// ellipse: growth rings round a dark heart, with a crack or two.
func woodEndHD(x, y int, nx, ny float64, rings float64, seed uint32) color.RGBA {
	d := math.Sqrt(nx*nx + ny*ny)
	a := math.Atan2(ny, nx)
	wob := d + 0.035*math.Sin(a*3+float64(seed)) + 0.02*math.Sin(a*7)
	l := 0.62 + 0.3*math.Sin(wob*rings*math.Pi) - d*0.15
	switch {
	case d < 0.12:
		l = 0.1
	case d > 0.88:
		l -= 0.25 // the bark's inner edge
	}
	for i := range 3 { // cracks from the heart
		ca := float64(i)*2.3 + float64(seed%7)
		if da := math.Abs(math.Mod(a-ca+5*math.Pi, 2*math.Pi) - math.Pi); da < 0.07/max(d, 0.1) && d > 0.1 && d < 0.55+0.15*float64(i) {
			l = 0.05
		}
	}
	l -= (nx + ny) * 0.08 // lit from the upper left
	return woodRamp.pick(l+(hashf(x, y, seed)-0.5)*0.08, x, y)
}

func genStumpHD(s float64) sprite {
	w, h := int(math.Ceil(16*s)), int(math.Ceil(12*s))
	s *= hdK3
	c := hdCanvas(w, h)
	cx, rx := math.Round(float64(w)/2)*hdK3, 6*s
	base := math.Round(9*s/hdK3) * hdK3 // ground point
	top := base - 4.5*s
	// The side: bark, with roots spreading at the foot.
	for y := int(top); y <= int(base+2*s); y++ {
		for x := int(cx - rx - 2*s); x <= int(cx+rx+2*s); x++ {
			px, py := float64(x)+0.5-cx, float64(y)+0.5
			f := clamp01((py - top) / (base - top)) // 0 at the cut, 1 at the ground
			flare := 1 + 0.16*f*f*f + 0.05*f*f*math.Sin(px*0.55)
			u := px / (rx * flare)
			foot := base + 2*s*math.Sqrt(max(0, 1-u*u)) // the round front of the foot
			if math.Abs(u) > 1 || py > foot {
				continue
			}
			col := barkAt(trunkRamp, u, rx, py*1.2, x, y, 0.8, 9)
			if f < 0.12 {
				col = trunkRamp[0] // the shade under the rim
			}
			c.put(x, y, col)
		}
	}
	c.ellipse(cx, top, rx, 2.5*s, func(x, y int, nx, ny float64) color.RGBA {
		return woodEndHD(x, y, nx, ny, 5+s/hdK3*2, 9)
	})
	c.outline(0.75)
	return mkUp(c, cx, base)
}

// leafHD is one small leaf at (x, y), a pixels half long and b half wide,
// pointing along angle ang, with a midrib.
func leafHD(c *canvas, x, y, a, b, ang float64, rp ramp, lit float64) {
	cos, sin := math.Cos(ang), math.Sin(ang)
	r := max(a, b) + 1
	for py := int(y - r); py <= int(y+r); py++ {
		for px := int(x - r); px <= int(x+r); px++ {
			dx, dy := float64(px)+0.5-x, float64(py)+0.5-y
			un := (dx*cos + dy*sin) / a
			if un < -1 || un > 1 {
				continue
			}
			v := -dx*sin + dy*cos
			w := b * math.Sqrt(1-un*un) * (1 - 0.35*un)
			if math.Abs(v) > w {
				continue
			}
			l := lit - (dx*0.62+dy*0.78)/a*0.22
			if math.Abs(v) < 0.5 {
				l += 0.18
			}
			c.put(px, py, rp.pick(l, px, py))
		}
	}
}

func genTwigHD() sprite {
	c := hdCanvas(24, 14)
	stick := func(x0, y0, x1, y1, w float64) {
		const k = hdK3
		drawLimbHD(c, limb{x0 * k, y0*k + 1, (x0 + x1) / 2 * k, (y0+y1)/2*k + 2, x1 * k, y1*k + 1, w, w * 0.6}, trunkRamp, 0.9, 7)
	}
	stick(2, 9, 20, 6, 2.4)
	stick(11, 7.5, 16, 2, 1.7)
	stick(6, 8.5, 4, 4, 1.5)
	knotHD(c, 26, 26.5, 1.6, trunkRamp)
	for _, l := range [][3]float64{{17, 1.5, -0.5}, {21, 5.5, 0.3}, {3.5, 3.2, -2.4}, {14.5, 3.2, -2.8}} {
		leafHD(c, l[0]*hdK3, l[1]*hdK3, 4.2, 2.2, l[2], leafRamps[1], 0.62)
	}
	c.ellipse(2*hdK3+1, 9*hdK3+2, 2, 2.6, func(x, y int, nx, ny float64) color.RGBA { // the broken end
		return woodRamp.pick(0.7-nx*0.3, x, y)
	})
	c.outline(0.75)
	return mkUp(c, 12*hdK3, 9*hdK3)
}

// stoneHD is a small rounded stone at (x, y), rx by ry pixels: lit from the
// upper left, with a flat lit face, a grain and a darker foot.
func stoneHD(c *canvas, x, y, rx, ry, lit float64, rp ramp, seed uint32) {
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		// Flatten the normal into two or three faces.
		fx, fy := math.Round(nx*1.6)/1.6, math.Round(ny*1.6)/1.6
		l := lum(0.5*nx+0.5*fx, 0.5*ny+0.5*fy)*lit + (hashf(px, py, seed)-0.5)*0.09
		if ny > 0.55 {
			l -= 0.15
		}
		if hashf(px/2, py/2, seed+1) < 0.03 {
			l -= 0.2 // a pit
		}
		return rp.pick(l, px, py)
	})
}

func genPebblesHD() sprite {
	c := hdCanvas(16, 10)
	for i, s := range []struct{ x, y, rx, ry, l float64 }{
		{8, 4.5, 2.2, 1.8, 1}, {5, 6.5, 3, 2.3, 0.9}, {10.5, 7, 2.6, 2, 0.75}, {13.2, 5.6, 1.1, 0.9, 0.9},
	} {
		stoneHD(c, s.x*hdK3, s.y*hdK3, s.rx*hdK3, s.ry*hdK3, s.l, stoneRamp, 520+uint32(i))
	}
	c.outline(0.7)
	return mkUp(c, 8*hdK3, 8*hdK3)
}

// --- the undergrowth ----------------------------------------------------------

// frondAlongHD lines the curve at (t from 0 to 1) with leaflets: blades two
// pixels wide every gap pixels on both sides, ln(t) long, swept forward and
// sagging; dark dims the whole frond.
func frondAlongHD(c *canvas, at func(t float64) (float64, float64), steps int, ln func(t float64) float64, gap, sag float64, rp ramp, dark float64, seed uint32) {
	next, walked := gap, 0.0
	lx, ly := at(0)
	blade := 0
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x, y := at(t)
		walked += math.Hypot(x-lx, y-ly)
		lx, ly = x, y
		x2, y2 := at(t + 0.02)
		tx, ty := x2-x, y2-y
		d := math.Hypot(tx, ty)
		if d == 0 {
			continue
		}
		tx, ty = tx/d, ty/d
		nx, ny := -ty, tx
		if walked >= next && t > 0.1 {
			next += gap
			blade++
			n := ln(t)
			for _, side := range []float64{-1, 1} {
				dx, dy := nx*side+tx*0.5, ny*side+ty*0.5+sag
				dl := math.Hypot(dx, dy)
				dx, dy = dx/dl, dy/dl
				tone := 0.4 + 0.4*t - dark + (hashf(blade, int(side)+2, seed)-0.5)*0.16
				if side*ny < 0 {
					tone += 0.15 // the leaflets facing up catch the light
				}
				for q := 0.0; q <= n; q += 0.5 {
					px, py := int(math.Floor(x+dx*q)), int(math.Floor(y+dy*q))
					if q < n*0.7 {
						if math.Abs(dx) > math.Abs(dy) {
							c.put(px, py+1, rp.pick(tone-0.28, px, py+1))
						} else {
							c.put(px+1, py, rp.pick(tone-0.28, px+1, py))
						}
					}
					c.put(px, py, rp.pick(tone-q/n*0.1, px, py))
				}
			}
		}
		px, py := int(math.Floor(x)), int(math.Floor(y))
		c.put(px, py, rp.pick(0.3+0.4*t-dark, px, py))
	}
}

func genFernHD(rng *rand.Rand) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(30, 17)
	const bx, by = 15.0 * k, 16.0 * k
	n := 6 + rng.IntN(2)
	type frond struct{ a, l float64 }
	fronds := make([]frond, n)
	for i := range fronds {
		f := (float64(i) + 0.5 + (rng.Float64()-0.5)*0.5) / float64(n)
		fronds[i] = frond{math.Pi * (1.08 - 1.16*f), (9 + rng.Float64()*4) * k}
	}
	for pass := 0; pass < 2; pass++ { // back to front: the most upright first
		for i, fr := range fronds {
			upright := math.Abs(math.Cos(fr.a)) < 0.45
			if upright != (pass == 0) {
				continue
			}
			dark := 0.0
			if upright {
				dark = 0.18
			}
			at := func(t float64) (float64, float64) {
				return bx + math.Cos(fr.a)*fr.l*t, by - math.Sin(fr.a)*fr.l*t + fr.l*0.55*t*t
			}
			frondAlongHD(c, at, int(fr.l*2.5), func(t float64) float64 { return (0.8 + 2.4*(1-t)) * k * math.Min(1, t*5) }, 2.5, 0.1, fernRamp, dark, 700+uint32(i))
		}
	}
	c.outline(0.5)
	return c, bx, by
}

func genBigLeavesHD(rng *rand.Rand) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(26, 20)
	const bx, by = 13.0 * k, 19.0 * k
	n := 3 + rng.IntN(2)
	for i := 0; i < n; i++ {
		f := (float64(i) + 0.5) / float64(n)
		a := math.Pi * (0.85 - 0.7*f + (rng.Float64()-0.5)*0.15)
		stalk := (2 + rng.Float64()*3) * k
		L, W := (8+rng.Float64()*3)*k, (3.2+rng.Float64()*1.2)*k
		ux, uy := math.Cos(a), -math.Sin(a)*0.8 // foreshortened: leaves lie back a little
		vx, vy := -uy, ux
		x0, y0 := bx+ux*stalk, by+uy*stalk
		for _, o := range []float64{0, 1} {
			c.line(bx+o, by, x0+o, y0, func(x, y int, _ float64) color.RGBA { return broadRamp[1+int(o)] })
		}
		var pts []float64
		const steps = 18
		wAt := func(t float64) float64 {
			return W * math.Pow(math.Sin(math.Pi*math.Min(1, t*1.15)), 0.7) * (1 - 0.15*t)
		}
		for s := 0; s <= steps; s++ { // one edge out to the tip...
			t := float64(s) / steps
			pts = append(pts, x0+ux*L*t+vx*wAt(t), y0+uy*L*t+vy*wAt(t))
		}
		for s := steps; s >= 0; s-- { // ...and the other back
			t := float64(s) / steps
			pts = append(pts, x0+ux*L*t-vx*wAt(t), y0+uy*L*t-vy*wAt(t))
		}
		c.polygon(pts, func(x, y int) color.RGBA {
			px, py := float64(x)+0.5-x0, float64(y)+0.5-y0
			along, across := (px*ux+py*uy)/L, (px*vx+py*vy)/W
			l := 0.55 + 0.25*across*vy*-1 + 0.2*along - 0.25*math.Abs(across)
			// Side veins sweep from the midrib towards the tip.
			if v := math.Mod(along*7-math.Abs(across)*1.6+8, 1); v < 0.14 {
				l -= 0.13
			} else if v < 0.3 {
				l += 0.06
			}
			if math.Abs(px*vx+py*vy) > wAt(clamp01(along))-1.2 {
				l -= 0.12 // the rim
			}
			return broadRamp.pick(l, x, y)
		})
		c.line(x0, y0, x0+ux*L*0.9, y0+uy*L*0.9, func(x, y int, t float64) color.RGBA {
			return broadRamp.at(5 - int(t*2))
		})
	}
	c.outline(0.6)
	return c, bx, by
}

func genClumpHD(rng *rand.Rand, seed uint32) (*canvas, float64, float64) {
	c := hdCanvas(18, 11)
	var blobs []blob
	for i := 0; i < 4; i++ {
		blobs = append(blobs, blob{4 + rng.Float64()*10, 5.5 + rng.Float64()*2.5, 2.8 + rng.Float64()*1.4})
	}
	leafClusterHD(c, scaledBlobs(blobs), 9*hdK3, 6*hdK3, 7*hdK3, 4.5*hdK3, fernRamp, 900+seed, leafSmall)
	c.outline(0.6)
	return c, 9 * hdK3, 9.5 * hdK3
}

func genBladesHD(rng *rand.Rand) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(18, 14)
	const bx, by = 9.0 * k, 13.0 * k
	for row := 0; row < 2; row++ {
		n := 8 + row*2
		for i := 0; i < n; i++ {
			x0 := bx + (-5+(float64(i)+rng.Float64()*0.7)*10/float64(n))*k
			h := (9 - float64(row)*2 + rng.Float64()*3) * k
			lean := (x0-bx)*0.5 + (rng.Float64()-0.5)*3*k
			bladeHD(c, x0, by, lean, h, 2.6+rng.Float64(), bladeRamp, 0.1+float64(row)*0.15, 0.7+float64(row)*0.15)
		}
	}
	c.outline(0.4)
	return c, bx, by
}

// bloomHD is a small flower at (x, y): five petals round a yellow eye.
func bloomHD(c *canvas, x, y, r float64, petals ramp, eye color.RGBA, turn float64) {
	for i := range 5 {
		a := float64(i)*2*math.Pi/5 - math.Pi/2 + turn
		px, py := x+math.Cos(a)*r*0.62, y+math.Sin(a)*r*0.5
		c.ellipse(px, py, r*0.52, r*0.44, func(ix, iy int, nx, ny float64) color.RGBA {
			return petals.pick(0.55+lum(nx, ny)*0.5-float64(i%2)*0.12, ix, iy)
		})
	}
	c.ellipse(x, y, max(1, r*0.3), max(1, r*0.26), func(int, int, float64, float64) color.RGBA { return eye })
}

func genFlowersHD(rng *rand.Rand, petals ramp) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(14, 11)
	const bx, by = 7.0 * k, 10.0 * k
	n := 3 + rng.IntN(3)
	for i := 0; i < n; i++ {
		x := bx + (-4+rng.Float64()*8)*k
		h := (3 + rng.Float64()*4) * k
		lean := (rng.Float64() - 0.5) * 2 * k
		c.line(x, by, x+lean, by-h, func(px, py int, t float64) color.RGBA { return bladeRamp.pick(0.2+0.4*t, px, py) })
		c.line(x+1, by, x+lean+1, by-h, func(px, py int, t float64) color.RGBA { return bladeRamp.pick(0.05+0.3*t, px, py) })
		leafHD(c, x+lean*0.2+2.5, by-h*0.4, 2.6, 1.3, -0.6, bladeRamp, 0.6) // a leaf on the stem
		bloomHD(c, x+lean, by-h, 4.2, petals, hex(0xf6d860), rng.Float64())
	}
	c.outline(0.35)
	return c, bx, by
}

// --- cacti, reeds, crops ------------------------------------------------------

// cactusLimbHD is one ribbed limb of a cactus between columns x0 and x1
// from top down to bottom, round on top when capped, with a row of spines
// down every rib.
func cactusLimbHD(c *canvas, x0, x1, top, bottom float64, capped bool) {
	half := (x1 - x0) / 2
	cx := x0 + half
	ribs := max(3, math.Round(half/2.2))
	for y := int(top); y <= int(bottom); y++ {
		for x := int(x0); x < int(math.Ceil(x1)); x++ {
			px, py := float64(x)+0.5-cx, float64(y)+0.5
			u := px / half
			if capped && py < top+half { // the domed top
				d := (top + half - py) / half
				if u*u+d*d > 1 {
					continue
				}
			}
			// Ribs: round ridges across the limb, crowded towards its edges.
			a := math.Asin(clamp(u, -1, 1)) / (math.Pi / 2) // -1 to 1 round the limb
			r := math.Mod(a*ribs/2+ribs, 1)
			l := 0.82 - (u+1)/2*0.6 + 0.16*math.Sin(r*math.Pi) - 0.08
			if r < 0.12 {
				l -= 0.18 // the groove between ribs
			}
			l += (hashf(x, y, 170) - 0.5) * 0.07
			col := cactusRamp.pick(l, x, y)
			if r > 0.42 && r < 0.6 && y%5 == 2 { // an areole and its spines
				col = hex(0xe8e2c0)
			}
			c.put(x, y, col)
		}
	}
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }

func genCactusHD(full bool) sprite {
	const k = hdK3
	c := hdCanvas(22, 34)
	const bx, by = 11 * k, 31 * k
	if !full {
		cactusLimbHD(c, bx-3*k, bx+3*k, by-7*k, by, false)
		c.ellipse(bx, by-7*k, 3*k, 1.3*k, func(x, y int, nx, ny float64) color.RGBA { // the cut: pale, wet flesh
			d := nx*nx + ny*ny
			if d > 0.7 {
				return cactusRamp.at(3)
			}
			return mix(hex(0xc4dca4), hex(0x90b47a), d+(hashf(x, y, 171)-0.5)*0.3)
		})
		c.outline(0.75)
		return mkUp(c, bx, by)
	}
	// The arms first: an elbow out from the trunk, then up.
	elbow := func(x0, x1, y0, y1 float64) {
		for y := int(y0); y <= int(y1); y++ {
			for x := int(x0); x < int(x1); x++ {
				u := (float64(y)+0.5-y0)/(y1-y0)*2 - 1
				l := 0.7 - (u+1)/2*0.5 + 0.1*math.Sin(float64(x)*0.9)
				c.put(x, y, cactusRamp.pick(l, x, y))
			}
		}
	}
	elbow(4*k, bx-2*k, 15*k, 19*k-1)
	cactusLimbHD(c, 3*k, 7*k, 10*k, 18*k, true) // left arm, rising
	elbow(bx+2*k, 18*k, 19*k, 23*k-1)
	cactusLimbHD(c, 15*k, 19*k, 13*k, 22*k, true) // right arm
	cactusLimbHD(c, bx-3*k, bx+3*k, 4*k, by, true)
	// A flower on the crown.
	bloomHD(c, bx+1, 4*k+1, 3.6, mkRamp(0xc8507a, 0xf08aa8, 0xffd0dc), hex(0xf6d860), 0.3)
	c.outline(0.75)
	return mkUp(c, bx, by)
}

func genReedsHD(full bool) sprite {
	const k = hdK3
	c := hdCanvas(20, 28)
	const bx, by = 10 * k, 26 * k
	for i := 0; i < 8; i++ {
		x0 := bx - 5*k + float64(i)*1.4*k
		h := (15 + hashf(i, 0, 180)*8) * k
		lean := (x0 - bx) * 0.25
		if !full {
			h = 3 * k
		}
		lo := 0.2 - float64(i%2)*0.15
		if full && i%2 == 1 { // a long leaf beside the stem, bending over
			bladeHD(c, x0+2, by, lean*2.4+4, h*0.8, 3.2, reedRamp, lo, lo+0.55)
		}
		bladeHD(c, x0, by, lean, h, 2.6, reedRamp, lo, lo+0.7)
		if full && i%2 == 0 {
			tx, ty := x0+lean*0.75, by-h+4*k
			c.ellipse(tx+0.5, ty, 1.5*k*0.8, 2.8*k, func(x, y int, nx, ny float64) color.RGBA {
				return cattail.pick(lum(nx, ny)*0.9+(hashf(x, y, 181)-0.5)*0.35, x, y) // the velvet head
			})
			for d := range 5 { // and the bare spike above it
				c.put(int(tx+0.5), int(ty-2.8*k)-d, reedRamp.at(3))
			}
		}
	}
	c.outline(0.5)
	return mkUp(c, bx, by)
}

func genCropHD(stage int, w, h int, r float64, ripe bool) sprite {
	const k = hdK3
	c := hdCanvas(w, h)
	cx, base := math.Round(float64(w)/2)*k, float64(h-3)*k
	rng := rand.New(rand.NewPCG(uint64(stage)+31, 7))
	r *= k
	n := 2 + 2*stage
	var blobs []blob
	for i := range n {
		a := math.Pi * (0.08 + 0.84*float64(i)/float64(n-1))
		d := r * (0.7 + rng.Float64()*0.5)
		blobs = append(blobs, blob{cx - math.Cos(a)*d*1.35, base - r*0.5 - math.Sin(a)*d, r * (0.5 + rng.Float64()*0.2)})
	}
	for _, b := range blobs { // the vines
		c.line(cx, base, b.x, b.y, func(int, int, float64) color.RGBA { return stemCol })
		c.line(cx+1, base, b.x+1, b.y, func(x, y int, _ float64) color.RGBA { return mix(stemCol, colBlack, 0.3) })
	}
	leafClusterHD(c, blobs, cx, base-r, r*1.6, r*1.3, leafRamps[1], 300+uint32(stage), leafSmall)
	if ripe {
		for i, t := range []struct{ x, y, rx, ry float64 }{{cx - 4*k, base + 0.5*k, 2.6 * k, 1.6 * k}, {cx + 3.5*k, base + 1*k, 2.4 * k, 1.5 * k}} {
			c.ellipse(t.x, t.y, t.rx, t.ry, func(x, y int, nx, ny float64) color.RGBA {
				l := lum(nx, ny) + (hashf(x, y, 310)-0.5)*0.08
				if hashf(x/2, y/2, 311+uint32(i)) < 0.05 {
					l -= 0.25 // an eye
				}
				return potatoRamp.pick(l, x, y)
			})
		}
	}
	c.outline(0.7)
	return mkUp(c, cx, base)
}
