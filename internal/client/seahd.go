package client

import (
	"image/color"
	"math"
)

// The monsters of the open sea, drawn at HDDensity with the kit of
// bosshd.go: the Mosasaurus (its head and neck reared out of the water, its
// bite, its back rolling through the surface, its shape under it) and the
// giant squid (its mantle, its arms, its shape under the water). Each is a
// run of frames, drawn the first time it is shown; searender.go picks them.

// artKnobs turns knobs given in art pixels into the canvas's own.
func artKnobs(c *canvas, ks ...knob) []knob {
	out := make([]knob, len(ks))
	for i, k := range ks {
		x, y := c.toPx(k[0], k[1])
		out[i] = knob{x, y, k[2] * c.k()}
	}
	return out
}

// --- the Mosasaurus -----------------------------------------------------------

const (
	mosaW, mosaH   = 240, 190
	mosaCX, mosaWY = 100.0, 150.0
	mosaK          = 1.12 // art pixels to a unit of the head's drawing
)

// mosaHide colours the beast's hide: l the light on it, v across it (-1 its
// back, 1 its belly), s along it in art pixels.
func mosaHide(x, y int, l, v, s float64) color.RGBA {
	// Scales: staggered cells with dark seams.
	row := y / 5
	if (x+row%2*3)%6 == 0 || y%5 == 0 {
		l -= 0.07
	}
	edge := 0.3 + 0.07*math.Sin(s*0.3) + (hashf(x/3, y/3, 331)-0.5)*0.1
	if v > edge {
		if f := s / 3.4; f-math.Floor(f) < 0.16 {
			l -= 0.1 // the plates of its throat
		}
		return rq(mosaBelly, l+0.08-max(0, v-0.6)*0.3, x, y)
	}
	if v > edge-0.08 {
		return rq(mosaRamp, l+0.22, x, y)
	}
	// Dark bars across the back, broken into blotches.
	if f := s/13 + v*0.25; f-math.Floor(f) < 0.42 && v < edge-0.2 && hashf(x/7, y/7, 332) > 0.18 {
		l -= 0.17
	}
	if v < -0.4 && v > -0.82 && l > 0.5 { // wet
		if h := hashf(x/5, y/2, 333); h < 0.14 {
			return mix(mosaRamp[5], megaWet, 0.55)
		}
	}
	return rq(mosaRamp, l+0.03, x, y)
}

// mosaSide paints the head and neck reared out of the water: rise is how far
// up (0..1), lean throws the head back (-1) or forward (1), ha is where the
// head looks and gape opens the jaws.
func mosaSide(c *canvas, rise, lean, ha, gape float64) {
	const bx, wy = mosaCX, mosaWY
	hx, hy := bx+30+lean*20, wy-58*rise+math.Abs(lean)*6
	// A low crest of spines down the nape.
	neck := []knob{{bx - 2, wy + 20 + (1-rise)*30, 16}, {bx + 4 + lean*3, wy - 14*rise, 14.5}, {bx + 16 + lean*10, wy - 38*rise, 12.5}, {hx, hy, 10.5}}
	for i := 0; i < 3; i++ {
		a, b := neck[i], neck[i+1]
		for k := 0.0; k < 1; k += 0.25 {
			x, y, r := lerpf(a[0], b[0], k), lerpf(a[1], b[1], k), lerpf(a[2], b[2], k)
			dx, dy := b[0]-a[0], b[1]-a[1]
			l := math.Hypot(dx, dy)
			nx, ny := dy/l, -dx/l // to the left of the way up: the nape
			c.polygon([]float64{x + nx*(r-1) - dx/l*2.4, y + ny*(r-1) - dy/l*2.4, x + nx*(r+3.2), y + ny*(r+3.2), x + nx*(r-1) + dx/l*2.4, y + ny*(r-1) + dy/l*2.4},
				func(px, py int) color.RGBA { return rq(mosaRamp, 0.3+hashf(px/3, py/3, 334)*0.2, px, py) })
		}
	}
	bossPaint(c, 0, func(x, y int, q bossPt) color.RGBA {
		return mosaHide(x, y, 0.1+q.l*0.84, q.v, q.s/c.k())
	}, artKnobs(c, neck...))
	fr := hdFrameAt(c, hx, hy, ha-gape*0.3, mosaK)
	fj := hdFrameAt(c, hx, hy, ha+gape*0.62, mosaK)
	if gape > 0.03 {
		fr.poly2(c, fj, [][3]float64{{-3, 1, 0}, {50, 2.2, 0}, {46, 1.6, 1}, {-3, 2.6, 1}}, func(x, y int) color.RGBA {
			u, v := fr.local(x, y)
			d := math.Hypot(u, v)
			return mix(colGullet, hex(0xd8687a), clamp01((d-4)/26)-hashf(x/2, y/2, 335)*0.12)
		})
	}
	tooth := func(f bossFrame, u, v, h float64) {
		f.tri(c, u-0.95, v, u+0.95, v, u-0.3, v+h, func(x, y int) color.RGBA {
			if lu, _ := f.local(x, y); lu > u+0.1 {
				return hex(0xd2ccba)
			}
			return colTooth
		})
	}
	bossPaint(c, 0.25, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if u > -1 && v < 1.4 {
			return color.RGBA{}
		}
		if u > 0 && v < 2.4 {
			if gape > 0.05 {
				return mix(hex(0xe89aa0), hex(0xa85868), clamp01((v-1.4)/1))
			}
			return mosaRamp[0]
		}
		l := 0.16 + q.l*0.8
		if f := u / 3; f-math.Floor(f) < 0.12 && v < 5 {
			l -= 0.14
		}
		return rq(mosaBelly, l, x, y)
	}, fj.knobs([]knob{{-2, 3.6, 6.2}, {14, 4, 4.8}, {30, 3.6, 3.3}, {46, 3, 2.2}}))
	bossPaint(c, 0.3, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > -1 && v > 2 {
			return color.RGBA{}
		}
		l := 0.12 + q.l*0.84
		switch {
		case v > 0.4: // the pale lip
			if f := u / 3; f-math.Floor(f) < 0.12 {
				l -= 0.14
			}
			return rq(mosaBelly, l+0.06, x, y)
		case v > -0.3:
			return rq(mosaRamp, l-0.12, x, y)
		case v > -3.2 && v < -1 && u < 9:
			return rq(mosaRamp, l-0.2, x, y) // a dark line back from the eye
		}
		row := math.Floor(v / 2.8)
		f := u/3.8 + row*0.5
		if fu, fv := f-math.Floor(f), v/2.8-row; fu < 0.1 || fv < 0.12 {
			l -= 0.12 // the plates of its skull
		}
		if q.l > 0.74 && hashf(x/5, y/2, 336) < 0.2 {
			return mix(mosaRamp[5], megaWet, 0.5)
		}
		return rq(mosaRamp, l+0.04, x, y)
	}, fr.knobs([]knob{{-2, -1, 9.2}, {12, -2, 8.2}, {27, -1.5, 6.2}, {40, -0.3, 4.4}, {50, 0.8, 2.9}}))
	if gape > 0.2 {
		for u := 5.0; u < 49; u += 3.3 {
			tooth(fr, u, 1.9, 2.4+2.2*math.Abs(math.Sin(u*1.7)))
			tooth(fj, u+1.6, 1.5, -(2.2 + 1.8*math.Abs(math.Sin(u*2.3))))
		}
	} else {
		for u := 8.0; u < 49; u += 3.3 { // the teeth over its lip
			tooth(fr, u, 1.5, 2.3)
		}
	}
	hdEye(c, fr, 12, -4.6, 1.75, mosaRamp[0], hex(0xf2c230), hex(0xfff0a0), true)
	frameQuad(c, fr, 8.6, -7.2, 15.4, -6.4, 0.9, mosaRamp[1])
	nx, ny := fr.at(45, -1.8)
	c.dotPx(nx, ny, 0.6*fr.k1(), mosaRamp[0])
}

// mosaBiteAges is the age, in frames at 60 a second, of each frame of its
// bite, which lasts 24.
var mosaBiteAges = [...]int{0, 2, 4, 7, 10, 12, 14, 17, 20}

var (
	mosaRise = newBHAnim(8, func(i int) (*canvas, float64, float64) {
		w := 2 * math.Pi * float64(i) / 8
		c := hdBossCanvas(mosaW, mosaH)
		mosaSide(c, 1+0.03*math.Cos(w), 0.12*math.Sin(w), -0.1+0.05*math.Sin(w), 0.1+0.07*math.Sin(2*w))
		hdOut(c, mosaWY, mosaCX, 12, float64(i)/8, -1, 0, 341)
		return hdCrop(c, mosaCX, mosaWY)
	})
	mosaBite = newBHAnim(len(mosaBiteAges), func(i int) (*canvas, float64, float64) {
		// Thrown back, jaws wide; then down and forward, and they shut.
		age := float64(mosaBiteAges[i])
		back := smooth(clamp01(age/6)) * (1 - smooth(clamp01((age-8)/4)))
		lunge := smooth(clamp01((age-8)/5)) * (1 - 0.6*smooth(clamp01((age-17)/7)))
		gape := 0.2 + 0.6*smooth(clamp01(age/6))
		if age >= 14 {
			gape = 0.14
		}
		c := hdBossCanvas(mosaW, mosaH)
		mosaSide(c, 1+0.06*back-0.14*lunge, -0.55*back+1.25*lunge, -0.35*back+0.42*lunge, gape)
		hdOut(c, mosaWY, mosaCX, 12, age/24, (age+3)/26, 26, 342)
		return hdCrop(c, mosaCX, mosaWY)
	})
	mosaBack = newBHAnim(10, func(i int) (*canvas, float64, float64) {
		// Its back rolls up through the surface and under again, the tail after it.
		u := (float64(i) + 0.5) / 10
		up := math.Sin(u * math.Pi)
		c := hdBossCanvas(170, 60)
		const cx, wy = 85.0, 42.0
		b := hdBossCanvas(170, 60)
		hump := func(x, w, h float64, seed uint32) {
			if h < 0.6 {
				return
			}
			for k := -w; k < w; k += 4.2 { // the scutes of its spine
				y := wy + 3 - (h+3)*math.Sqrt(max(0, 1-k*k/(w*w)))
				b.polygon([]float64{x + k - 2.2, y + 1.5, x + k + 0.4, y - 2.2, x + k + 2.2, y + 1.5}, func(px, py int) color.RGBA { return rq(mosaRamp, 0.3, px, py) })
			}
			b.ellipse(x, wy+3, w, h+3, func(px, py int, nx, ny float64) color.RGBA {
				return mosaHide(px, py, 0.62-ny*0.42-nx*0.1, -0.5+ny*0.2, float64(px)/HDDensity)
			})
		}
		hump(cx+14-30*u+30, 44, 9*up, 343)
		hump(cx-40-10*u, 22, 5*math.Sin(clamp01(u*1.3-0.2)*math.Pi), 344)
		if t := math.Sin(clamp01(u*1.4-0.35) * math.Pi); t > 0.1 { // the fluke of its tail
			x := cx - 70
			b.polygon([]float64{x + 8, wy + 2, x - 2, wy - 13*t, x - 10, wy - 16*t, x - 6, wy + 2}, func(px, py int) color.RGBA {
				return rq(mosaRamp, 0.42+float64(py%6)*0.02, px, py)
			})
		}
		b.outline(0.8)
		c.blendOver(b, 0, 0)
		hdOut(c, wy, cx, 5, u, -1, 0, 345)
		return hdCrop(c, cx, wy)
	})
	mosaShadow = newBHAnim(8, func(i int) (*canvas, float64, float64) {
		// From above, under the water: a long body with four paddles, swimming.
		ph := 2 * math.Pi * float64(i) / 8
		c := hdBossCanvas(190, 90)
		const cx, cy, L = 95.0, 45.0, 150.0
		at := func(s float64) (float64, float64) {
			back := 1 - s
			return cx + (s-0.5)*L, cy + 7*math.Sin(1+4.2*back-ph)*math.Pow(back, 1.2)
		}
		const n = 500
		pts := make([]spinePt, 0, n+1)
		for j := 0; j <= n; j++ {
			s := float64(j) / n
			x, y := at(s)
			r := 1.4 + 9.6*math.Pow(math.Sin(math.Pi*math.Pow(s, 0.8)), 0.9)
			if s > 0.8 { // the long head
				r = max(r, lerpf(6.4, 2.2, (s-0.8)/0.2))
			}
			pts = append(pts, spinePt{x: x, y: y, r: r, s: s})
		}
		dark := solid(colMosaShadow)
		hdTube(c, pts, 1e9, func(int, int, tubePt) color.RGBA { return colMosaShadow })
		for k, s := range []float64{0.66, 0.36} {
			x, y := at(s)
			sw := 5 * math.Sin(ph+float64(k)*1.6)
			l := 22 - 5*float64(k)
			for _, side := range []float64{-1, 1} {
				c.polygon([]float64{x + 5, y + side*6, x - l*0.55 + sw, y + side*(l+2), x - l + sw, y + side*l, x - 6, y + side*6}, dark)
			}
		}
		tx, ty := at(0)
		t2x, t2y := at(0.14)
		sw := 6 * math.Cos(1+4.2-ph)
		c.polygon([]float64{t2x, t2y - 2.4, tx - 4, ty + sw - 7, tx - 9, ty + sw, tx - 3, ty + sw + 5, t2x, t2y + 2.4}, dark)
		hdSilhouette(c, colMosaShadow, 1)
		return hdCrop(c, cx, cy)
	})
)

// --- the giant squid ----------------------------------------------------------

// krakenHide colours the squid's hide: wet red, freckled; l is the light on
// it.
func krakenHide(x, y int, l float64) color.RGBA {
	switch h := hashf(x/2, y/2, 351); {
	case h < 0.035:
		l += 0.15
	case h < 0.08:
		l -= 0.1
	}
	if l > 0.8 && hashf(x/5, y/2, 352) < 0.3 {
		return mix(krakenRamp[5], colWhite, 0.4) // wet
	}
	return rq(krakenRamp, l, x, y)
}

// krakenLimb is the colour of an arm: red above, pale with a row of suckers
// on the side it curls towards (side 1 its right, -1 its left).
func krakenLimb(side float64) func(x, y int, q tubePt) color.RGBA {
	return func(x, y int, q tubePt) color.RGBA {
		t := max(-1, min(1, q.t))
		l := lum(q.nx*t*0.9, q.ny*t*0.9)
		if v := t * side; v > 0.2 {
			f := q.arc / 3.6
			if d := math.Hypot((f-math.Floor(f)-0.5)*3.6, (v-0.62)*q.r); d < min(1.25, q.r*0.34) {
				if d < 0.45 {
					return krakenPale[0]
				}
				return krakenPale[3] // a sucker
			}
			return rq(krakenPale, l*0.8+0.12, x, y)
		}
		if f := q.arc / 7; f-math.Floor(f) < 0.3 && t*side < -0.2 {
			l -= 0.08 // faint bands
		}
		return krakenHide(x, y, l+0.04)
	}
}

// krakenLimbPts lays an arm of length L from (x0, y0) heading a0, waving and
// curling by wave and curl along it.
func krakenLimbPts(x0, y0, a0, L, r0 float64, n int, bend func(s float64) float64) []spinePt {
	pts := make([]spinePt, 0, n+1)
	x, y := x0, y0
	for i := 0; i <= n; i++ {
		s := float64(i) / float64(n)
		a := a0 + bend(s)
		pts = append(pts, spinePt{x: x, y: y, r: 1.1 + r0*math.Pow(1-s, 0.85), s: s})
		x, y = x+math.Cos(a)*L/float64(n), y+math.Sin(a)*L/float64(n)
	}
	return pts
}

const krakArmN = 12

var (
	krakenArm = newBHAnim(krakArmN, func(i int) (*canvas, float64, float64) {
		// One arm out of the water, swaying, its tip curling and uncurling.
		w := 2 * math.Pi * float64(i) / krakArmN
		c := hdBossCanvas(130, 130)
		const bx, wy = 50.0, 110.0
		pts := krakenLimbPts(bx, wy+12, -math.Pi/2, 104, 6.4, 420, func(s float64) float64 {
			return 0.3*math.Sin(w-2.4*s)*s + (2.6+1.1*math.Sin(w))*s*s*s + 0.12
		})
		hdTube(c, pts, 16, krakenLimb(1))
		hdOut(c, wy, bx, 8, float64(i)/krakArmN, -1, 0, 361)
		return hdCrop(c, bx, wy)
	})
	krakenHead = newBHAnim(8, func(i int) (*canvas, float64, float64) {
		// Its mantle out of the water, breathing, an eye as big as a shield.
		w := 2 * math.Pi * float64(i) / 8
		b := 1 + 0.035*math.Sin(w)
		c := hdBossCanvas(130, 130)
		const cx, wy = 65.0, 104.0
		tipX, tipY := cx-10+1.5*math.Sin(w), wy-74-1.5*math.Sin(w)
		fin := func(side float64) {
			f := 3 * math.Sin(w+side)
			c.polygon([]float64{tipX, tipY + 1, tipX + side*14, tipY + 12 + f*0.5, tipX + side*27, tipY + 30 + f, tipX + side*16, tipY + 40 + f*0.4, tipX + 2, tipY + 44}, func(x, y int) color.RGBA {
				return krakenHide(x, y, 0.4-side*0.14+float64(y%9)*0.012)
			})
		}
		fin(-1)
		fin(1)
		bossPaint(c, 0, func(x, y int, q bossPt) color.RGBA {
			l := 0.1 + q.l*0.8 - 0.2*q.v - 0.1*q.v*q.v
			if f := q.s / (9 * c.k()); f-math.Floor(f) < 0.5 && math.Abs(q.v) < 0.5 && q.t < 0.7 {
				l += 0.05 // the bands that run over it as it flushes
			}
			if q.v > 0.55 {
				return rq(krakenPale, l*0.8+0.05, x, y)
			}
			return krakenHide(x, y, l)
		}, artKnobs(c, knob{cx + 2, wy + 14, 21 * b}, knob{cx, wy - 16, 20 * b}, knob{cx - 4, wy - 42, 15 * b}, knob{cx - 8, wy - 62, 8.5}, knob{tipX, tipY, 3}))
		fr := hdFrameAt(c, cx+9, wy-13, 0, 1)
		hdEye(c, fr, 0, 0, 5.2, krakenRamp[0], hex(0xf0c040), hex(0xfff2b0), false)
		frameQuad(c, fr, -5.5, -6.6, 5.5, -7.4, 1.1, krakenRamp[1])
		hdOut(c, wy, cx, 10, float64(i)/8, -1, 0, 362)
		return hdCrop(c, cx, wy)
	})
	krakenShadow = newBHAnim(8, func(i int) (*canvas, float64, float64) {
		// From above, under the water: the mantle and its fins, the arms trailing.
		w := 2 * math.Pi * float64(i) / 8
		c := hdBossCanvas(200, 120)
		const cx, cy = 90.0, 60.0
		dark := func(int, int, tubePt) color.RGBA { return colKrakShadow }
		pulse := 1 + 0.06*math.Sin(w)
		mantle := make([]spinePt, 0, 201)
		for j := 0; j <= 200; j++ {
			s := float64(j) / 200
			mantle = append(mantle, spinePt{x: cx - 62 + 70*s, y: cy, r: (2 + 13*math.Pow(math.Sin(math.Pi*math.Pow(s, 0.7)*0.93), 0.8)) * pulse, s: s})
		}
		hdTube(c, mantle, 1e9, dark)
		for _, side := range []float64{-1, 1} {
			c.polygon([]float64{cx - 62, cy, cx - 52, cy + side*(17+2*math.Sin(w)), cx - 36, cy + side*9}, solid(colKrakShadow))
		}
		for k := 0; k < 10; k++ {
			a := (float64(k) - 4.5) * 0.15
			L, r := 62.0, 3.0
			if k == 2 || k == 7 { // the two long ones
				L, r = 96, 2.2
			}
			ph := w + float64(k)*0.9
			pts := krakenLimbPts(cx+6, cy+float64(k-5)*1.6+0.8, a, L, r, 260, func(s float64) float64 {
				return 0.5*math.Sin(ph-3.2*s)*s - a*s*0.6
			})
			hdTube(c, pts, 1e9, dark)
		}
		hdSilhouette(c, colKrakShadow, 1)
		return hdCrop(c, cx, cy)
	})
)
