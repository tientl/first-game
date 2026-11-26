package client

import (
	"image/color"
	"math"
	"sort"
)

// Titanoboa, drawn at HDDensity: a monster of the old jungle, a body thicker
// than a tree trunk and a score of tiles long, a tube along a curve. Moss
// green scales with saddles of dark bronze, a pale belly of ridged plates;
// a curved spine of horn on every saddle down the whole of its back, and a
// row of spiked fins along each flank (titanBody stamps them in with the
// body, so a coil lying over another hides its spines too). A great
// armoured head, painted like the other bosses' heads: a crown of long
// horns swept back, a brow like a ledge over eyes that glow, jaws that gape
// on rows of fangs. Its frames: under way with its neck reared and its
// coils trailing, asleep in a pile of coils, rearing to hiss, drawing back
// and striking, and squeezing its prey.

var (
	titanSkin  = mkRamp(0x0b1108, 0x17210d, 0x263614, 0x394e1d, 0x516929, 0x6e8639, 0x90a352)
	titanDark  = mkRamp(0x0a0705, 0x191109, 0x2b1d0e, 0x432e16)
	titanRim   = mkRamp(0x3a2810, 0x5e431a, 0x866428, 0xad8a3c, 0xcdae5c)
	titanBelly = mkRamp(0x4a4430, 0x777052, 0xa39a74, 0xc8c098, 0xe4debc)
	titanHorn  = mkRamp(0x1a120a, 0x302112, 0x4c3720, 0x705536, 0x988058, 0xc2b088, 0xe2d8b8)
	titanEye   = hex(0xff9a1e)
	titanSheen = [...]color.RGBA{hex(0x9fd66a), hex(0xd8c860), hex(0xe0a050), hex(0xc8d878), hex(0xffe08a)}
)

const (
	titanLen    = 480.0 // art pixels of body under way
	titanSaddle = 16.0  // art pixels from one saddle to the next
	titanHeadK  = 1.8   // art pixels to a unit of the head's drawing
)

// titanWidth is the body's half-width at s: 0 the tip of the tail, 1 the neck.
func titanWidth(s float64) float64 {
	return 2.5 + 12.5*smooth(clamp01(s/0.24)) - 3.6*smooth(clamp01((s-0.82)/0.18))
}

// titanSkinAt colours the body; slide moves its markings along it.
func titanSkinAt(slide float64) func(x, y int, q tubePt) color.RGBA {
	lx, ly := lightDir[0], lightDir[1]
	ll := math.Hypot(lx, ly)
	return func(x, y int, q tubePt) color.RGBA {
		t := max(-1, min(1, q.t))
		a := q.arc + slide
		l := lum(q.nx*t*0.92, q.ny*t*0.92)
		b := math.Asin(t) * q.r
		// The scales: a lattice of diamonds, each catching the light.
		pu, pv := a/3.5+b/2.9, a/3.5-b/2.9
		fu, fv := pu-math.Floor(pu), pv-math.Floor(pv)
		seam, shine := fu < 0.2 || fv < 0.2, fu > 0.55 && fv > 0.55
		down := t * q.ny      // how far towards the ground side of the body
		back := t + 0.22*q.ny // from the line of the spine
		toLight := t * (q.nx*lx + q.ny*ly) / ll
		sheen := math.Exp(-math.Pow((toLight-0.5)/0.3, 2))
		tone := func(r ramp, l float64, wet float64) color.RGBA {
			if seam {
				l -= 0.14
			} else if shine {
				l += 0.09
				wet *= 2.4
			}
			col := rq(r, l, x, y)
			if k := wet * sheen; k > 0.02 { // wet scales, where the light falls
				h := a*0.045 + toLight*2.2 + hashf(int(math.Floor(pu)), int(math.Floor(pv)), 931)*0.9
				h -= math.Floor(h/5) * 5
				i := int(h)
				col = mix(col, mix(titanSheen[i%5], titanSheen[(i+1)%5], h-float64(i)), min(0.3, k))
			}
			return col
		}
		if down > 0.58+(hashf(x/2, y/2, 932)-0.5)*0.08 { // the belly's plates, each with a ridge
			f := a / (titanSaddle / 4)
			switch ff := f - math.Floor(f); {
			case ff < 0.16:
				l -= 0.22
			case ff < 0.42:
				l += 0.1
			}
			return rq(titanBelly, l+0.14-(down-0.58)*0.5, x, y)
		}
		if q.s > 0.03 {
			f := a / titanSaddle
			at := math.Abs(f-math.Floor(f)-0.5) * 2 // 0 between two saddles, 1 in the middle of one
			d := (1 - at) + math.Abs(back)*1.05
			switch {
			case d < 0.62:
				return tone(titanDark, l+0.14, 0.2) // a saddle of dark bronze
			case d < 0.8:
				return rq(titanRim, l*0.7+0.16, x, y) // and its rim
			}
			// A dark blotch on the flank between two saddles.
			if d2 := math.Hypot(at*1.15, (down-0.34)*2.7); d2 < 0.3 {
				return tone(titanDark, l+0.14, 0.2)
			} else if d2 < 0.42 {
				return rq(titanRim, l*0.7+0.1, x, y)
			}
		}
		// Moss green, mottled darker.
		if vnoise(a/9, b/5, 936) > 0.62 {
			l -= 0.09
		}
		return tone(titanSkin, l+0.04, 0.12)
	}
}

// --- the spines of its back and the fins of its flanks -------------------

// titanSpike is a curved horn growing out of the body: from its root by way
// of a bend to its tip, in art pixels.
type titanSpike struct {
	at             int     // it is set on once the body is laid up to this point
	arc            float64 // where along the body it grows
	x0, y0, bx, by float64
	x1, y1, w      float64 // its tip, and half its width at the root
	shade          float64
}

// titanSpikes grows the spines and fins of a body laid along pts: a spine
// on the middle of every saddle, a fin on each flank between two.
func titanSpikes(pts []spinePt, arc []float64, slide float64) []titanSpike {
	n := len(pts)
	find := func(a float64) int { return min(n-1, sort.Search(n, func(i int) bool { return arc[i] >= a })) }
	total := arc[n-1]
	var out []titanSpike
	for k := int(math.Floor(slide/titanSaddle)) - 1; ; k++ {
		a := (float64(k)+0.5)*titanSaddle - slide
		if a > total {
			break
		}
		if a < 0 {
			continue
		}
		i := find(a)
		p := pts[i]
		tx, ty := p.ny, -p.nx // the way the body runs here
		if p.s > 0.045 && p.s < 0.975 {
			h := 1.1*p.r + 3 + 4.5*smooth(clamp01((p.s-0.72)/0.2))
			if k&1 != 0 {
				h = 0.6*p.r + 1.5 // a lesser one between two great ones
			}
			x0, y0 := p.x-p.nx*0.22*p.ny*p.r, p.y-p.ny*0.22*p.ny*p.r
			at := n
			if e := a + p.r + 0.8*h; e < total {
				at = find(e)
			}
			out = append(out, titanSpike{at: at, arc: a, x0: x0, y0: y0,
				bx: x0 + tx*0.12*h, by: y0 - 0.78*h + ty*0.12*h,
				x1: x0 - tx*0.62*h, y1: y0 - h - ty*0.62*h, w: max(1.3, 0.21*p.r+0.07*h), shade: 0.04})
		}
		a -= titanSaddle / 2
		if a < 0 || k&1 != 0 {
			continue
		}
		i = find(a)
		p = pts[i]
		tx, ty = p.ny, -p.nx
		if p.s < 0.1 || p.s > 0.95 {
			continue
		}
		for _, sg := range [2]float64{-1, 1} {
			nx, ny := sg*p.nx, sg*p.ny
			x0, y0 := p.x+nx*0.86*p.r, p.y+ny*0.86*p.r
			l := 0.55*p.r + 2
			sh := -0.3
			if ny > 0 {
				sh = -0.18 // the near flank, in the light
			}
			out = append(out, titanSpike{at: find(a - 1), arc: a, x0: x0, y0: y0,
				bx: x0 + nx*0.75*l + tx*0.1*l, by: y0 + ny*0.75*l + ty*0.1*l,
				x1: x0 + nx*l - tx*0.95*l, y1: y0 + ny*l - ty*0.95*l, w: max(1.2, 0.2*p.r), shade: sh})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// titanBody paints the body along pts like hdTube, and with it the spines
// and fins that grow from it: each is laid on in its turn along the body,
// so a later coil covers the spines of the one under it.
func titanBody(c *canvas, pts []spinePt, gap, slide float64, shade func(x, y int, q tubePt) color.RGBA) {
	arc := hdSpine(pts)
	spikes := titanSpikes(pts, arc, slide)
	k := c.k()
	sp := make([]spinePt, len(pts))
	for i, p := range pts {
		p.x, p.y = c.toPx(p.x, p.y)
		p.r *= k
		sp[i] = p
	}
	// idx is what each pixel shows: point i of the body, or spike -2-j.
	idx := make([]int32, c.w*c.h)
	for i := range idx {
		idx[i] = -1
	}
	horn := make([]color.RGBA, c.w*c.h)
	ll := math.Hypot(lightDir[0], lightDir[1])
	stamp := func(j int) {
		s := &spikes[j]
		m := int((math.Hypot(s.bx-s.x0, s.by-s.y0)+math.Hypot(s.x1-s.bx, s.y1-s.by))*k) + 1
		for q := 0; q <= m; q++ {
			u := float64(q) / float64(m)
			v := 1 - u
			px, py := c.toPx(v*v*s.x0+2*u*v*s.bx+u*u*s.x1, v*v*s.y0+2*u*v*s.by+u*u*s.y1)
			rad := (s.w*math.Pow(v, 0.8) + 0.3) * k
			for y := max(0, int(py-rad)); y <= min(c.h-1, int(py+rad)); y++ {
				dy := float64(y) + 0.5 - py
				for x := max(0, int(px-rad)); x <= min(c.w-1, int(px+rad)); x++ {
					dx := float64(x) + 0.5 - px
					if dx*dx+dy*dy > rad*rad {
						continue
					}
					l := 0.38 + 0.26*u + 0.4*(dx*lightDir[0]+dy*lightDir[1])/(rad*ll) + s.shade
					if f := u * 7; f-math.Floor(f) < 0.16 && u < 0.7 {
						l -= 0.1 // the rings of its growth
					}
					idx[y*c.w+x], horn[y*c.w+x] = int32(-2-j), rq(titanHorn, l, x, y)
				}
			}
		}
	}
	next := 0
	for i := range sp {
		for ; next < len(spikes) && spikes[next].at <= i; next++ {
			stamp(next)
		}
		p := &sp[i]
		r2 := p.r * p.r
		for y := max(0, int(p.y-p.r)); y <= min(c.h-1, int(p.y+p.r)); y++ {
			dy := float64(y) + 0.5 - p.y
			for x := max(0, int(p.x-p.r)); x <= min(c.w-1, int(p.x+p.r)); x++ {
				if dx := float64(x) + 0.5 - p.x; dx*dx+dy*dy <= r2 {
					idx[y*c.w+x] = int32(i)
				}
			}
		}
	}
	for ; next < len(spikes); next++ {
		stamp(next)
	}
	arcOf := func(i int32) float64 {
		if i <= -2 {
			return spikes[-2-i].arc
		}
		return arc[i]
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			i := idx[y*c.w+x]
			switch {
			case i == -1:
			case i <= -2:
				col := horn[y*c.w+x]
				for _, o := range [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} { // a line where it meets the body or another
					if nx, ny := x+o[0], y+o[1]; c.in(nx, ny) && idx[ny*c.w+nx] != i && idx[ny*c.w+nx] != -1 {
						col = mix(col, colInk, 0.5)
						break
					}
				}
				c.put(x, y, col)
			default:
				p := &sp[i]
				dx, dy := float64(x)+0.5-p.x, float64(y)+0.5-p.y
				along := (dx*p.ny - dy*p.nx) / k // ahead of the point, in art pixels
				q := tubePt{s: p.s, arc: arc[i] + along, t: (dx*p.nx + dy*p.ny) / p.r, nx: p.nx, ny: p.ny, r: p.r / k}
				if col := shade(x, y, q); col.A > 0 {
					c.put(x, y, col)
				} else {
					idx[y*c.w+x] = -1
				}
			}
		}
	}
	// What lies under a later coil is in its shadow.
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			i := idx[y*c.w+x]
			if i == -1 {
				continue
			}
			ai := arcOf(i)
			for d := 1; d <= 6; d++ {
				dark := 0.0
				for _, o := range [4][2]int{{0, -d}, {-d, 0}, {d, 0}, {0, d}} {
					nx, ny := x+o[0], y+o[1]
					if !c.in(nx, ny) {
						continue
					}
					if j := idx[ny*c.w+nx]; j != -1 && arcOf(j) > ai+gap {
						dark = 0.62 - 0.09*float64(d)
						if o[0] > 0 || o[1] > 0 {
							dark *= 0.5 // the lit side of what is above
						}
						break
					}
				}
				if dark > 0 {
					c.put(x, y, mix(c.at(x, y), colInk, dark))
					break
				}
			}
		}
	}
}

// --- the head -----------------------------------------------------------

var (
	titanSkull = []knob{{-4, -2, 8.6}, {6, -3.2, 8.4}, {15, -2.2, 6.6}, {23.5, -0.6, 4.4}}
	titanJaw   = []knob{{-4, 3.4, 5.4}, {8, 3.8, 4.6}, {21, 3, 3.1}}
)

// frameQuad fills a line from (u0, v0) to (u1, v1) of the frame, w thick.
func frameQuad(c *canvas, f bossFrame, u0, v0, u1, v1, w float64, col color.RGBA) {
	dx, dy := u1-u0, v1-v0
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	f.poly(c, [][2]float64{{u0 + nx, v0 + ny}, {u1 + nx, v1 + ny}, {u1 - nx, v1 - ny}, {u0 - nx, v0 - ny}}, solid(col))
}

// frameHorn grows a horn in the frame from (u0, v0) to its tip (u1, v1),
// bowed bend to the side and w thick at the root.
func frameHorn(c *canvas, f bossFrame, u0, v0, u1, v1, bend, w, shade float64) {
	dx, dy := u1-u0, v1-v0
	l := math.Hypot(dx, dy)
	bu, bv := (u0+u1)/2-dy/l*bend, (v0+v1)/2+dx/l*bend
	ll := math.Hypot(lightDir[0], lightDir[1])
	m := int(1.3*l*f.k1()) + 1
	for q := 0; q <= m; q++ {
		u := float64(q) / float64(m)
		v := 1 - u
		px, py := f.at(v*v*u0+2*u*v*bu+u*u*u1, v*v*v0+2*u*v*bv+u*u*v1)
		rad := (w/2*math.Pow(v, 0.8) + 0.16) * f.k1()
		for y := int(py - rad); y <= int(py+rad); y++ {
			ey := float64(y) + 0.5 - py
			for x := int(px - rad); x <= int(px+rad); x++ {
				ex := float64(x) + 0.5 - px
				d := (ex*ex + ey*ey) / (rad * rad)
				if d > 1 {
					continue
				}
				lt := 0.34 + 0.36*u + 0.42*(ex*lightDir[0]+ey*lightDir[1])/(rad*ll) + shade
				if g := u * 8; g-math.Floor(g) < 0.15 && u < 0.75 {
					lt -= 0.1
				}
				c.put(x, y, rq(titanHorn, lt, x, y))
			}
		}
	}
}

// titanHead paints the head with the back of its skull at (hx, hy) art
// pixels, looking along a; gape opens the jaws (0..1) and tongue (0..1) puts
// the tongue out.
func titanHead(c *canvas, hx, hy, a, gape, tongue float64) {
	fr := hdFrameAt(c, hx, hy, a-gape*0.42, titanHeadK)
	fj := hdFrameAt(c, hx, hy, a+gape*0.58, titanHeadK)
	// The crown: long horns swept back from the skull, and the spikes of
	// the frill behind the jaw.
	frameHorn(c, fj, -4, 6.5, -17, 13, -2, 2.6, -0.14)
	frameHorn(c, fj, -7, 4.5, -23, 7.5, -1.5, 3, -0.12)
	frameHorn(c, fr, -9, 0, -28, 0.5, 1.5, 3.2, -0.1)
	frameHorn(c, fr, -8, -5, -31, -9, 3, 3.8, -0.06)
	frameHorn(c, fr, -4, -8.5, -27, -19, 4, 4.2, 0)
	frameHorn(c, fr, 2, -10, -15, -26, 4.5, 3.8, 0.02)
	if gape > 0 { // the inside of the mouth
		fr.poly2(c, fj, [][3]float64{{-3, 1, 0}, {24, 1.8, 0}, {21.5, 1.4, 1}, {-3, 2, 1}}, func(x, y int) color.RGBA {
			u, v := fr.local(x, y)
			d := math.Hypot(u, v)
			col := mix(colGullet, hex(0xd0606c), clamp01((d-3)/15)-hashf(x/2, y/2, 933)*0.12)
			if f := u / 2.6; f-math.Floor(f) < 0.16 && d > 6 {
				col = mix(col, colGullet, 0.25) // the ridges of its palate
			}
			return col
		})
	}
	if tongue > 0 {
		f, tu, tv := fj, 20.0, 0.4
		if gape < 0.1 {
			f, tu, tv = fr, 23, 1.6
		}
		e := 8 * tongue
		frameQuad(c, f, tu-6, tv, tu+e, tv-0.4, 0.95, colTongue)
		frameQuad(c, f, tu-6, tv+0.3, tu+e-2, tv, 0.5, hex(0x8c1c30))
		for _, dv := range []float64{-2.6, 1.9} {
			frameQuad(c, f, tu+e-0.3, tv-0.4, tu+e+4.2*tongue, tv-0.4+dv*tongue, 0.6, colTongue)
		}
	}
	bossPaint(c, 0.25, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if u > -1 && v < 1.3 {
			return color.RGBA{}
		}
		if u > 0 && v < 2.2 {
			if gape > 0.05 {
				return mix(hex(0xd88890), hex(0x9c4858), clamp01((v-1.3)/0.9))
			}
			return titanDark[0]
		}
		l := 0.15 + q.l*0.8
		if f := u / 2.6; f-math.Floor(f) < 0.14 && v < 5 {
			l -= 0.18 // the plates of the lip
		}
		if v > 6 {
			return rq(titanRim, l-0.1, x, y) // the armour under the jaw
		}
		return rq(titanBelly, l, x, y)
	}, fj.knobs(titanJaw))
	for i, u := range []float64{16.5, 11, 5.5, 0.5} { // spikes under the chin
		frameHorn(c, fj, u, 6.6+0.35*float64(i), u-3.5, 10+0.7*float64(i), -0.5, 1.5+0.2*float64(i), -0.1)
	}
	bossPaint(c, 0.3, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > -1 && v > 1.7 {
			return color.RGBA{}
		}
		l := 0.12 + q.l*0.84
		switch {
		case v > 0.25: // the pale lip, with its heat pits
			f := u / 2.6
			ff := f - math.Floor(f)
			if ff < 0.14 {
				l -= 0.18
			}
			if u > 5 && u < 22 && math.Hypot((ff-0.56)*2.6, v-1) < 0.42 {
				return titanDark[1]
			}
			return rq(titanBelly, l+0.08, x, y)
		case v > -0.5 && u > 9.5:
			return rq(titanDark, l+0.2, x, y) // the line of the mouth
		}
		// A dark wedge back from the eye, rimmed in bronze.
		if w := math.Abs(v+3.6) - 0.9 - 0.2*(9-u); u < 9 && w < 0 {
			return rq(titanDark, l+0.14, x, y)
		} else if u < 9.6 && w < 0.7 {
			return rq(titanRim, l*0.7+0.2, x, y)
		}
		// The armour of the crown: plates bigger than the body's scales,
		// bronze along the middle of the skull, mossy at its edges.
		pu, pv := u/4+v/3.3, u/4-v/3.3
		fu, fv := pu-math.Floor(pu), pv-math.Floor(pv)
		l += -0.02*v - 0.04
		shine := fu > 0.5 && fv > 0.5
		if fu < 0.15 || fv < 0.15 {
			l -= 0.16
		} else if shine {
			l += 0.09
		}
		if d := math.Abs(v+9.4) + max(0, u-4)*0.16 + 0.8*vnoise(u/2.5, v/2, 939); u < 19 && d < 2.3 {
			return rq(titanRim, l-0.12, x, y)
		}
		return rq(titanSkin, l+0.05, x, y)
	}, fr.knobs(titanSkull))
	fang := func(x, y int) color.RGBA {
		if hashf(x/2, y, 934) < 0.25 {
			return hex(0xcfc8b0)
		}
		return colFang
	}
	if gape > 0.25 {
		g := min(1, gape)
		for _, u := range []float64{17.2, 21.8} { // the fangs, curved back
			fr.poly(c, [][2]float64{{u - 0.3, 1.5}, {u + 1.9, 1.5}, {u + 0.8, 1.6 + 5.2*g}, {u - 1.9, 1.6 + 9.6*g}, {u - 0.7, 1.6 + 4.4*g}}, fang)
		}
		for i, u := 0, 2.4; u < 16; i, u = i+1, u+1.75 { // and the rows of teeth behind them
			h := 2.9 + 1.5*hashf(i, 5, 938)
			fr.tri(c, u-0.6, 1.6, u+0.6, 1.6, u-0.5, 1.6+h, fang)
			fj.tri(c, u+0.3, 1.4, u+1.5, 1.4, u+1.3, 1.4-0.8*h, fang)
		}
		fj.tri(c, 18, 1.3, 19.8, 1.3, 19.3, -3, fang)
	}
	// The eye, burning under a brow like a ledge.
	ex, ey := fr.at(11, -4.6)
	for gr := 3.6 * fr.k1(); gr > fr.k1(); gr -= 1 {
		c.dotGlow(ex, ey, gr, withAlpha(titanEye, 16))
	}
	hdEye(c, fr, 11, -4.6, 1.75, titanDark[0], titanEye, hex(0xfff2a0), true)
	frameQuad(c, fr, 6.4, -8, 14.6, -6, 2, titanDark[1])
	frameQuad(c, fr, 6.6, -8.9, 14.2, -7, 0.7, titanRim[3])
	frameHorn(c, fr, 8, -8.6, 2.5, -14.5, 1, 2.2, 0) // the horn of the brow
	for i, u := range []float64{13.5, 18, 22} {      // the ridge of spikes down the snout
		frameHorn(c, fr, u, -8.4+1.5*float64(i), u-2.6+0.4*float64(i), -12.6+2.2*float64(i), 0.5, 1.7-0.25*float64(i), 0.02)
	}
	nx, ny := fr.at(23.4, -2)
	c.dotPx(nx, ny, 0.55*fr.k1(), titanDark[0])
}

// dotGlow lays a disc of light of radius r pixels about pixel (px, py),
// only over what is drawn already.
func (c *canvas) dotGlow(px, py, r float64, col color.RGBA) {
	for y := int(py - r); y <= int(py+r); y++ {
		for x := int(px - r); x <= int(px+r); x++ {
			dx, dy := float64(x)+0.5-px, float64(y)+0.5-py
			if dx*dx+dy*dy <= r*r && c.in(x, y) && c.px[y*c.w+x].A > 0 {
				c.blend(x, y, col)
			}
		}
	}
}

// titanSerp lays a body of length L along a serpenoid curve ending with its
// neck at (x1, y1), heading right; the head end barely sways.
func titanSerp(L, bend, waves, phase, squash, x1, y1 float64) []spinePt {
	const n = 1400
	us, vs := make([]float64, n+1), make([]float64, n+1)
	for i := 1; i <= n; i++ {
		s := float64(i) / n
		steady := 1 - 0.8*smooth(clamp01((s-0.7)/0.3))
		th := bend * steady * math.Cos(2*math.Pi*waves*s-phase)
		us[i] = us[i-1] + math.Cos(th)*L/n
		vs[i] = vs[i-1] + math.Sin(th)*L/n
	}
	// Keep it on the line of its travel: take out the drift of the whole wave.
	drift := (vs[n*7/10] - vs[0]) / (us[n*7/10] - us[0])
	pts := make([]spinePt, n+1)
	for i := range pts {
		s := float64(i) / n
		v := vs[i] - vs[n] - drift*(us[i]-us[n])*(1-smooth(clamp01((s-0.7)/0.3)))
		pts[i] = spinePt{x: x1 + us[i] - us[n], y: y1 + v*squash, r: titanWidth(s), s: s}
	}
	return pts
}

// titanCoil lays the body in a spiral pile about (cx, cy), winding in from
// radius outer to inner over turns and climbing rise as it goes; the last of
// it leaves the pile as a neck, by way of (mx, my) to (nx, ny).
func titanCoil(cx, cy, outer, inner, turns, rise, squash, mx, my, nx, ny float64) []spinePt {
	const n = 1800
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
			ax, ay := lerpf(x0, mx, u), lerpf(y0, my, u)
			bx, by := lerpf(mx, nx, u), lerpf(my, ny, u)
			x, y = lerpf(ax, bx, u), lerpf(ay, by, u)
		}
		pts = append(pts, spinePt{x: x, y: y, r: titanWidth(s), s: s})
	}
	return pts
}

// titanPose is one frame of the snake.
type titanPose struct {
	pts          []spinePt
	hx, hy, ha   float64 // the back of the skull, and where it looks
	gape, tongue float64
	slide        float64 // its markings moved along it
	ax, ay       float64 // the anchor
	shade        float64 // the body up to here (0..1 along it) lies on the ground
}

const titanW, titanH, titanCX, titanCY = 640, 400, 440.0, 300.0

func (p titanPose) canvas() (*canvas, float64, float64) {
	c := hdBossCanvas(titanW, titanH)
	hdGroundShadow(c, p.pts, 0, p.shade)
	body := hdBossCanvas(titanW, titanH)
	titanBody(body, p.pts, 44, p.slide, titanSkinAt(p.slide))
	titanHead(body, p.hx, p.hy, p.ha, p.gape, p.tongue)
	body.outline(0.8)
	for i, q := range body.px {
		if q.A > 0 {
			c.px[i] = q
		}
	}
	return hdCrop(c, p.ax, p.ay)
}

// titanSlitherPose is the snake under way: ph is the phase of the wave. Its
// neck is reared off the ground, the head carried high over the coils that
// trail behind it.
func titanSlitherPose(ph float64) titanPose {
	const nx, ny = titanCX + 30, titanCY - 10
	pts := titanSerp(titanLen, 1.3, 2.3, ph, 0.56, nx, ny)
	for j := range pts {
		u := clamp01((pts[j].s - 0.76) / 0.24)
		pts[j].y -= 40*smooth(u) - 6*u*u*u
		pts[j].x -= 12 * math.Sin(u*math.Pi)
	}
	last := pts[len(pts)-1]
	return titanPose{pts: pts, hx: last.x - 2, hy: last.y - 1, ha: 0.08, ax: titanCX, ay: titanCY, shade: 0.8}
}

const (
	titanSlitherN = 12
	titanIdleN    = 8
	titanRoarN    = 8
	titanGrabN    = 8
)

// titanStrikeAges is the age, in frames at 60 a second, each frame of the
// strike shows: drawing back, held there, the lunge (the blow lands 36
// frames in), the bite, and back.
var titanStrikeAges = [...]int{0, 3, 6, 9, 14, 20, 26, 31, 33, 35, 37, 41, 45, 49, 53}

var (
	titanSlither = newBHAnim(titanSlitherN, func(i int) (*canvas, float64, float64) {
		p := titanSlitherPose(-2 * math.Pi * float64(i) / titanSlitherN)
		if i == 5 || i == 6 {
			p.tongue = 0.6 + 0.4*float64(i-5)
		}
		return p.canvas()
	})
	titanStrike = newBHAnim(len(titanStrikeAges), func(i int) (*canvas, float64, float64) {
		age := float64(titanStrikeAges[i])
		p := titanSlitherPose(0)
		rear := smooth(clamp01(age / 9))
		lunge := smooth(clamp01((age - 31) / 5))
		if age > 42 {
			lunge = 1 - smooth(clamp01((age-42)/12))
			rear = 0
		} else {
			rear *= 1 - 0.85*lunge
		}
		shake := 0.0
		if age > 9 && age < 31 {
			shake = math.Sin(age * 1.9)
		}
		for j := range p.pts {
			u := clamp01((p.pts[j].s - 0.6) / 0.4)
			q := &p.pts[j]
			q.y -= rear*(40*smooth(u)-6*u*u*u) - lunge*(26*smooth(u)-3*u*u) + shake*0.7*u
			q.x += -rear*52*u*u + rear*20*math.Sin(u*math.Pi) + lunge*50*u*u
		}
		last := p.pts[len(p.pts)-1]
		p.hx, p.hy = last.x-2, last.y-1
		p.ha = -0.42*rear + 0.34*lunge
		p.gape = 0.12 + 0.88*smooth(clamp01(age/14))
		switch {
		case age >= 42:
			p.gape = 0.3 * (1 - clamp01((age-42)/8))
		case age >= 36:
			p.gape = 0.3 // shut on its prey
		}
		if age >= 9 && age < 31 {
			p.tongue = 1
		}
		p.shade = 0.55 + 0.25*(1-max(rear, lunge))
		return p.canvas()
	})
	titanIdle = newBHAnim(titanIdleN, func(i int) (*canvas, float64, float64) {
		// Asleep: a pile of coils that swells as it breathes, the head laid on it.
		b := math.Sin(2 * math.Pi * float64(i) / titanIdleN)
		const cx, cy = titanCX - 20, titanCY - 30
		pts := titanCoil(cx, cy, 64+b*0.7, 24, 2.3, 20+b, 0.5, cx+30, cy-40, cx+48, cy-13+b*0.5)
		for j := range pts {
			pts[j].r *= 1 + 0.03*b*math.Sin(pts[j].s*math.Pi)
		}
		p := titanPose{pts: pts, hx: cx + 46, hy: cy - 14 + b*0.5, ha: 0.14, ax: cx, ay: cy + 41, shade: 0.42}
		if i == 2 || i == 3 {
			p.tongue = 0.5 + 0.5*float64(i-2)
		}
		return p.canvas()
	})
	titanRoar = newBHAnim(titanRoarN, func(i int) (*canvas, float64, float64) {
		// Reared out of its coils, its neck spread, hissing from side to side.
		w := 2 * math.Pi * float64(i) / titanRoarN
		sx := math.Sin(w) * 8
		const cx, cy = titanCX - 20, titanCY - 30
		pts := titanCoil(cx, cy, 66, 26, 2, 16, 0.5, cx-28+sx*0.4, cy-106, cx+6+sx, cy-116+math.Cos(w)*2)
		for j := range pts {
			pts[j].r *= 1 + 0.32*math.Sin(clamp01((pts[j].s-0.86)/0.13)*math.Pi)
		}
		return titanPose{pts: pts, hx: cx + 4 + sx, hy: cy - 116 + math.Cos(w)*2, ha: -0.12 + math.Sin(w)*0.1,
			gape: 0.92 + 0.08*math.Sin(2*w), tongue: 0.6 + 0.4*math.Sin(2*w), ax: cx, ay: cy + 40, shade: 0.5}.canvas()
	})
	titanGrab = newBHAnim(titanGrabN, func(i int) (*canvas, float64, float64) {
		// Wound tight, ring on ring, round whatever it has caught: the rings
		// draw in and ease, and slide round as it tightens.
		w := 2 * math.Pi * float64(i) / titanGrabN
		sq := 0.5 - 0.5*math.Cos(w)
		const cx, cy = titanCX - 20, titanCY - 26
		pts := titanCoil(cx, cy, 44-3.5*sq, 33-3*sq, 3.3, 74+3*sq, 0.36, cx+8, cy-120-2*sq, cx+36, cy-102+2*sq)
		for j := range pts {
			pts[j].r *= 1 + 0.06*sq*math.Sin(pts[j].s*math.Pi)
		}
		return titanPose{pts: pts, hx: cx + 34, hy: cy - 102 + 2*sq, ha: 0.55 + 0.08*math.Sin(w), gape: 0.4 + 0.25*sq, tongue: sq,
			slide: -titanSaddle * float64(i) / titanGrabN, ax: cx, ay: cy + 24, shade: 0.3}.canvas()
	})
)

// titanStrikeFrame is the frame of the strike age frames into it.
func titanStrikeFrame(age int) int {
	f := 0
	for i, a := range titanStrikeAges {
		if age >= a {
			f = i
		}
	}
	return f
}
