package client

import (
	"image/color"
	"math"
)

// glowTreeHD is the glowing tree of the woods drawn at HD, glowTreeS times
// its first size: two stems twisted round each other, boughs fanning out
// under caps like a toadstool's, ringed and spotted, their gills faintly
// alight, and strands of luminous beads hanging under them. The second
// sprite is just what glows, drawn bright over the dark at night (see
// drawGlowMasks). lit is false for a tree picked bare: nothing on it glows.
func glowTreeHD(lit bool) (sprite, sprite) {
	const k = HDDensity
	const gx, gy = 28.0, 84.0 // the ground point, art pixels
	const hubX, hubY = 29.0, 27.0
	c, m := hdCanvas(56, 87), hdCanvas(56, 87)
	glowing := func(rp ramp) ramp {
		if lit {
			return rp
		}
		return glowStem
	}
	// A bead of light: round, lit from the top left, its halo and itself on
	// the mask.
	bead := func(x, y, r float64, rp ramp) {
		if lit {
			halo := rp.at(1)
			m.ellipse(x*k, y*k, r*2.1*k, r*2.1*k, func(px, py int, nx, ny float64) color.RGBA {
				return mix(color.RGBA{0, 0, 0, 255}, halo, 0.55*(1-(nx*nx+ny*ny)))
			})
			m.ellipse(x*k, y*k, r*k, r*k, func(px, py int, nx, ny float64) color.RGBA { return rp.pick(0.95-ny*0.2, px, py) })
		}
		c.ellipse(x*k, y*k, r*k, r*k, func(px, py int, nx, ny float64) color.RGBA {
			return glowing(rp).pick(lum(nx, ny)*0.9+0.15, px, py)
		})
	}

	// The stems, twisted round each other, and the neck they rise into.
	drawLimbHD(c, limb{(gx + 1.6) * k, gy * k, (gx - 5) * k, (gy - 27) * k, (hubX + 1) * k, (hubY + 11) * k, 2.4 * k, 1.3 * k}, glowStem, 0.62, 961)
	drawLimbHD(c, limb{(gx - 1.6) * k, gy * k, (gx + 5) * k, (gy - 21) * k, (hubX - 1) * k, (hubY + 13) * k, 2.7 * k, 1.5 * k}, glowStem, 0.9, 962)
	drawLimbHD(c, limb{hubX * k, (hubY + 15) * k, (hubX - 2.5) * k, (hubY + 7) * k, hubX * k, hubY * k, 2.1 * k, 1.5 * k}, glowStem, 0.82, 963)
	for _, side := range []float64{-1, 1} { // the foot spreads into roots
		drawLimbHD(c, limb{(gx + side*1.5) * k, (gy - 4) * k, (gx + side*4) * k, (gy - 1) * k, (gx + side*7) * k, (gy + 0.5) * k, 1.4 * k, 0.5 * k}, glowStem, 0.7, 964)
	}
	// Glowing specks up the stems, like foxfire.
	for i := range 9 {
		t := 0.15 + float64(i)*0.08
		x := gx + math.Sin(t*9+float64(i))*3.5
		y := gy - t*52
		if c.opaque(int(x*k), int(y*k)) {
			bead(x, y, 0.45+hashf(i, 0, 965)*0.3, []ramp{glowCyan, glowViolet}[i%2])
		}
	}

	// The boughs, arching up and out, and the caps at their ends: drawn from
	// the back (the middle, highest) to the front.
	type bough struct{ ex, ey float64 }
	var bs []bough
	for i := range 7 {
		a := -math.Pi/2 + (float64(i)-3)*0.5
		bs = append(bs, bough{hubX + math.Cos(a)*19, hubY + 3 + math.Sin(a)*14})
	}
	order := []int{3, 2, 4, 1, 5, 0, 6}
	for _, i := range order {
		b := bs[i]
		drawLimbHD(c, limb{hubX * k, hubY * k, (hubX + b.ex) / 2 * k, (math.Min(hubY, b.ey) - 3) * k, b.ex * k, b.ey * k, 1.3 * k, 0.7 * k}, glowStem, 0.8, 966+uint32(i))
	}
	for _, i := range order {
		b := bs[i]
		rp := []ramp{glowCyan, glowViolet}[i%2]
		cx, cy := b.ex, b.ey-1
		const rx, ry = 5.4, 3.7
		c.ellipse(cx*k, cy*k, rx*k, ry*k, func(px, py int, nx, ny float64) color.RGBA {
			if ny > 0.28 { // the gills underneath
				l := 0.25 + 0.25*float64(int((nx+1)*9)%2) - (ny-0.28)*0.3
				return mix(glowStem.at(0), glowing(rp).at(2), clamp01(l))
			}
			l := lum(nx, ny)*0.75 + 0.15 + (vnoise(float64(px)*0.25, float64(py)*0.25, 967)-0.5)*0.2
			if d := nx*nx + ny*ny; d > 0.62 && d < 0.78 {
				l += 0.18 // a pale ring round the cap
			}
			return glowStem.pick(l, px, py)
		})
		if lit { // the gills glow faintly
			m.ellipse(cx*k, cy*k, rx*k, ry*k, func(px, py int, nx, ny float64) color.RGBA {
				if ny <= 0.28 {
					return color.RGBA{}
				}
				return mix(color.RGBA{0, 0, 0, 255}, rp.at(2), 0.35+0.25*float64(int((nx+1)*9)%2))
			})
		}
		for j := range 4 { // spots on the cap, alight
			sx := cx + (hashf(i, j, 968)-0.5)*rx*1.3
			sy := cy - 0.5 - hashf(i, j, 969)*ry*0.9
			bead(sx, sy, 0.45+hashf(i, j, 970)*0.25, rp)
		}
		// Two strands of beads hang under each cap, the beads smaller and
		// fewer towards the ends.
		for s, dx := range []float64{-2.4, 2.3} {
			n := 3 + (i+s)%3
			x := cx + dx
			top, bot := cy+ry*0.7, cy+ry*0.7+float64(n)*3.9
			for y := top * k; y < bot*k; y++ {
				c.put(int(x*k), int(y), glowStem.at(1)) // the thread
			}
			for q := range n {
				bead(x+math.Sin(float64(q)*1.3+float64(i))*0.3, top+2.4+float64(q)*3.9, 1.3-float64(q)*0.13, []ramp{rp, glowCyan, glowViolet}[(q+s)%3])
			}
		}
	}
	c.outline(0.6)
	return mkUp(c, gx*k, gy*k), mkUp(m, gx*k, gy*k)
}
