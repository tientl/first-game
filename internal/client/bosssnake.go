package client

import (
	"image/color"
	"math"
)

// Titanoboa: the snake of snake.go grown monstrous. Its body is the same
// tube along a curve (drawTubeWith), thick as a tree, deep jungle green with
// black saddles rimmed in gold and a golden belly; its head is painted like
// the other bosses' heads, with jaws that gape and long fangs.

var (
	titanSkin  = mkRamp(0x0c2012, 0x15341a, 0x214c24, 0x326930, 0x48873e, 0x68a650, 0x8fc068)
	titanDark  = mkRamp(0x070b08, 0x0e150e, 0x172214, 0x22321c)
	titanBelly = mkRamp(0x7a5a16, 0xb08a26, 0xdcb83e, 0xf2da64, 0xfcf0a0)
	titanEye   = hex(0xff4a28)
)

const titanSaddles = 24 // down the back

// titanWidth is the body's half-width at s: 0 the tip of the tail, 1 the neck.
func titanWidth(s float64) float64 {
	return 1.6 + 8.2*smooth(clamp01(s/0.3)) - 2.8*smooth(clamp01((s-0.8)/0.2))
}

func titanScale(x, y int, p *spinePt, t float64) color.RGBA {
	l := lum(p.nx*t*0.9, p.ny*t*0.9) + bossScales(x, y, 921)
	back := t + 0.2
	if math.Abs(back) < 0.22 {
		l += 0.1 // the light along the spine
	}
	at := math.Abs(math.Mod(p.s*titanSaddles+0.5, 1)-0.5) * 2
	d := at + math.Abs(back)*0.95
	marked := p.s > 0.04
	switch {
	case t > 0.56:
		if int(p.s*260)%3 == 0 {
			l -= 0.14 // belly plates
		}
		return titanBelly.pick(l+0.12, x, y)
	case marked && d < 0.6:
		return flat(titanDark, l+0.12)
	case marked && d < 0.8:
		return flat(titanBelly, l*0.6+0.25) // the gold rim of each saddle
	}
	return titanSkin.pick(l+0.02, x, y)
}

var (
	titanSkull = []knob{{-4, -1.5, 7.2}, {6, -2.2, 7}, {15, -1.2, 5.4}, {22.5, -0.2, 3.7}}
	titanJaw   = []knob{{-4, 3, 4.8}, {8, 3.3, 4}, {20, 2.6, 2.7}}
)

// titanHead paints the head with the back of its skull at (hx, hy), looking
// along a; gape opens the jaws (0..1) and tongue flicks it.
func titanHead(c *canvas, hx, hy, a, gape float64, tongue bool) {
	fr := bossFrame{hx, hy, a - gape*0.4}
	fj := bossFrame{hx, hy, a + gape*0.55}
	if gape > 0 {
		ux, uy := fr.at(23, 1.6)
		jx, jy := fj.at(20.5, 1.2)
		b0x, b0y := fr.at(-3, 1)
		b1x, b1y := fj.at(-3, 2)
		c.polygon([]float64{b0x, b0y, ux, uy, jx, jy, b1x, b1y}, func(x, y int) color.RGBA {
			d := math.Hypot(float64(x)+0.5-hx, float64(y)+0.5-hy)
			return mix(colGullet, hex(0xd8606a), clamp01((d-4)/12)-dither(x, y)*0.2)
		})
	}
	if tongue {
		f := fj
		tu, tv := 20.0, 0.6
		if gape == 0 {
			f, tv = fr, 1.6
			tu = 22
		}
		x0, y0 := f.at(tu-6, tv)
		x1, y1 := f.at(tu+6, tv-0.5)
		bossLine(c, x0, y0, x1, y1, colTongue)
		for _, dv := range []float64{-2.4, 1.6} {
			x2, y2 := f.at(tu+10, tv-0.5+dv)
			bossLine(c, x1, y1, x2, y2, colTongue)
		}
	}
	bossPaint(c, 0.25, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if u > -1 && v < 1.3 {
			return color.RGBA{}
		}
		if u > 0 && v < 2.3 {
			if gape > 0 {
				return hex(0xe89aa0)
			}
			return titanDark[0]
		}
		return titanBelly.soft(0.15+q.l*0.8, x, y)
	}, fj.knobs(titanJaw))
	bossPaint(c, 0.3, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > -1 && v > 1.7 {
			return color.RGBA{}
		}
		l := 0.12 + q.l*0.82 + bossScales(x, y, 922)
		switch {
		case v > 0.2:
			return titanBelly.soft(l+0.1, x, y) // the pale lip
		case v > -2.8 && u < 9:
			return titanDark.soft(l+0.1, x, y) // the stripe behind the eye
		case v < -5.5 && u > 1 && int(u)%6 < 3:
			return titanDark.soft(l+0.15, x, y) // the plates of the crown
		}
		return titanSkin.soft(l+0.05, x, y)
	}, fr.knobs(titanSkull))
	if gape > 0.25 {
		for _, u := range []float64{16.5, 20.5} { // the fangs
			for _, w := range []float64{0, 0.9} {
				x0, y0 := fr.at(u+w, 1.6)
				x1, y1 := fr.at(u-1.2, 1.6+7.5*min(1, gape))
				bossLine(c, x0, y0, x1, y1, colFang)
			}
		}
		for u := 4.0; u < 15; u += 2.6 {
			x0, y0 := fr.at(u, 1.7)
			x1, y1 := fr.at(u, 3.4)
			bossLine(c, x0, y0, x1, y1, colFang)
			x0, y0 = fj.at(u+1.3, 1.3)
			x1, y1 = fj.at(u+1.3, -0.4)
			bossLine(c, x0, y0, x1, y1, colFang)
		}
	}
	ex, ey := fr.at(10, -3.6)
	ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
	for q := -1; q <= 1; q++ {
		for w := -1; w <= 2; w++ {
			c.put(ix+w, iy+q, titanDark[0])
		}
	}
	c.put(ix, iy, titanEye)
	c.put(ix+1, iy, colInk)
	c.put(ix, iy-1, hex(0xffb060))
	c.put(ix+1, iy-1, titanEye)
	nx, ny := fr.at(21.5, -1.6)
	c.put(int(nx), int(ny), titanDark[0])
}

// titanCoil lays the body in a spiral pile about (cx, cy), winding in from
// radius outer to inner over turns and climbing rise as it goes; the last of
// it leaves the pile as a neck, by way of (mx, my) to (nx, ny).
func titanCoil(cx, cy, outer, inner, turns, rise, squash, mx, my, nx, ny float64) []spinePt {
	const n = 460
	const coilEnd = 0.8
	spiral := func(s float64) (float64, float64) {
		ang := 0.5 + 2*math.Pi*turns*s
		rad := outer - (outer-inner)*s/coilEnd
		return cx + rad*math.Cos(ang), cy + rad*math.Sin(ang)*squash - rise*s/coilEnd
	}
	pts := make([]spinePt, 0, n+1)
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		var x, y float64
		if s < coilEnd {
			x, y = spiral(s)
		} else {
			u := (s - coilEnd) / (1 - coilEnd)
			x0, y0 := spiral(coilEnd)
			// A bend through the middle point.
			ax, ay := lerpf(x0, mx, u), lerpf(y0, my, u)
			bx, by := lerpf(mx, nx, u), lerpf(my, ny, u)
			x, y = lerpf(ax, bx, u), lerpf(ay, by, u)
		}
		pts = append(pts, spinePt{x: x, y: y, r: titanWidth(s), s: s})
	}
	return withNormals(pts)
}

func titanoboaCanvas(p bossPose) (*canvas, float64, float64) {
	const W, H = 320, 240
	const cx, cy = 160.0, 170.0
	c := newCanvas(W, H)
	var pts []spinePt
	hx, hy, ha, gape, tongue := 0.0, 0.0, 0.0, 0.0, false
	ax, ay := cx, cy
	switch p {
	case bossWalkA, bossWalkB, bossAttack:
		ph := 0.0
		if p == bossWalkB {
			ph = math.Pi
		}
		const neckX, neckY = cx + 62, cy - 4
		pts = serpenoid(196, 1.0, 1.45, ph, neckX, neckY)
		for i := range pts {
			pts[i].r = titanWidth(pts[i].s)
		}
		hx, hy = neckX-2, neckY-1
		tongue = p == bossWalkA
		// It lies on the ground along the middle of its waves.
		lo, hi, sum := math.Inf(1), math.Inf(-1), 0.0
		for _, q := range pts {
			lo, hi, sum = min(lo, q.x), max(hi, q.x), sum+q.y
		}
		ax, ay = (lo+hi)/2+8, sum/float64(len(pts))+7
		if p == bossAttack { // the fore-body rears and the head lunges
			for i := range pts {
				u := clamp01((pts[i].s - 0.52) / 0.48)
				pts[i].y -= 56*smooth(u) - 12*u*u*u
				pts[i].x += 16*u*u - 20*math.Sin(u*math.Pi)
			}
			pts = withNormals(pts)
			last := pts[len(pts)-1]
			hx, hy, ha, gape, tongue = last.x-2, last.y, 0.2, 1, true
		}
	case bossSpecial:
		// Wound tight, ring on ring, round whatever it has caught.
		pts = titanCoil(cx, cy-4, 30, 23, 3.4, 52, 0.36, cx+6, cy-84, cx+28, cy-70)
		hx, hy, ha, gape = cx+26, cy-70, 0.5, 0.45
		ay = cy + 7
	case bossRoar:
		pts = titanCoil(cx-4, cy, 48, 20, 1.7, 12, 0.5, cx-16, cy-86, cx+6, cy-90)
		for i := range pts { // the neck spread into a hood
			pts[i].r *= 1 + 0.5*math.Sin(clamp01((pts[i].s-0.86)/0.13)*math.Pi)
		}
		hx, hy, ha, gape, tongue = cx+4, cy-90, -0.1, 1, true
		ay = cy + 25
	case bossLurk:
		pts = titanCoil(cx-6, cy, 54, 24, 1.75, 3, 0.42, cx+36, cy-14, cx+52, cy+2)
		hx, hy, ha = cx+50, cy+1, 0.12
		ay = cy + 29
	default:
		pts = titanCoil(cx-4, cy, 50, 19, 2.2, 15, 0.5, cx-12, cy-62, cx+8, cy-60)
		hx, hy, ha = cx+6, cy-60, 0.08
		ay = cy + 27
	}
	drawTubeWith(c, pts, titanScale)
	titanHead(c, hx, hy, ha, gape, tongue)
	c.outline(0.8)
	if p == bossLurk {
		// Ferns in front of it.
		for i := 0; i < 9; i++ {
			fx := cx - 62 + float64(i)*15 + hashf(i, 0, 923)*6
			fy := cy + 30
			for k := -3; k <= 3; k++ {
				ex := fx + float64(k)*4.5 + hashf(i, k+5, 924)*2
				ey := fy - 15 - hashf(i, k+9, 925)*9 + math.Abs(float64(k))*3.4
				rp := leafRamps[i%len(leafRamps)]
				c.line(fx, fy, ex, ey, func(x, y int, t float64) color.RGBA { return rp.at(2 + int(t*3.2)) })
				c.line(fx+1, fy, ex+1, ey+1, func(x, y int, t float64) color.RGBA { return rp.at(1 + int(t*2.2)) })
			}
		}
	}
	return bossCrop(c, ax, ay)
}
