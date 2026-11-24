package client

import (
	"image/color"
	"math"
)

// The mammoth king is the mammoth of iceart.go drawn half as big again and
// painted the same way (furPaint), but nearly black, with frost in the
// crown of its head and along its hump, old scars across its flank, and
// tusks that sweep out twice as far. Its whole body can rear about its
// hind legs.

var (
	kingFur   = mkRamp(0x0f0b0e, 0x1b1316, 0x2b1d1e, 0x3e2a27, 0x553a32, 0x6f4e40, 0x8c6752, 0xab866a)
	kingFrost = mkRamp(0x4e5668, 0x78829a, 0xa4b0c4, 0xccd8e4, 0xeaf2f8, 0xffffff)
	kingSkin  = mkRamp(0x171012, 0x271b1c, 0x3a2a28, 0x503c36, 0x685046, 0x82685a)
	kingTusk  = mkRamp(0x6e4e22, 0xa47c3a, 0xd2ae64, 0xeed69a, 0xfdf2cc, 0xfffcf0)
	kingScar  = hex(0xb07a72)
)

func mammothKingCanvas(p bossPose) (*canvas, float64, float64) {
	k := 1.45                 // how much bigger than the mammoth
	const ox, oy = 44.0, 64.0 // room round the design, in its units
	const gy = 80.0
	const pvx, pvy = 25.0, 62.0 // what it rears about
	c := newCanvas(int((106+ox+56)*k), int((gy+4+oy)*k))
	rear, drop := 0.0, 0.0
	sw := 1.2
	a, hdy := 0.0, 0.0
	switch p {
	case bossWalkA:
		sw = 3
	case bossWalkB:
		sw = -3
	case bossAttack:
		sw, a, hdy = 4.5, 0.24, 1
	case bossRoar:
		sw, a, hdy = 2, -0.38, -1
	case bossSpecial:
		rear, a = -0.6, -0.05
	case bossLurk:
		drop, a, hdy, sw = 11, 0.04, -2, 2
	}
	T := func(x, y float64) (float64, float64) {
		X, Y := rot(x, y+drop, pvx, pvy, rear)
		return (X + ox) * k, (Y + oy) * k
	}
	inv := func(px, py int) (float64, float64) {
		x, y := rot((float64(px)+0.5)/k-ox, (float64(py)+0.5)/k-oy, pvx, pvy, -rear)
		return x, y - drop
	}
	B := func(x, y, r float64) knob { X, Y := T(x, y); return knob{X, Y, r * k} }
	G := func(x, y, r float64) knob { return knob{(x + ox) * k, (y + oy) * k, r * k} }
	// The coat: frost lies on whatever is above the line frost (in the
	// design's units), ragged along its edge.
	coat := func(bias, frost float64) pelt {
		return func(x, y int, l, ny, t float64) color.RGBA {
			fx, fy := float64(x), float64(y)
			strand := vnoise(fx*0.75+fy*0.1, fy*0.15, 931)
			lock := vnoise(fx*0.4, fy*0.07, 932)
			l = 0.1 + l*0.76 + (strand-0.5)*0.2 + (lock-0.5)*0.16 - ny*0.08 + bias
			switch q := hashf(x, int(math.Floor((fy+hashf(x, 7, 937)*7)/6)), 938); {
			case q < 0.2:
				l += 0.09
			case q > 0.82:
				l -= 0.08
			}
			v := fy + vnoise(fx*0.11, 0.5, 933)*9 + hashf(x, 0, 934)*2.6
			kk := 0.4 + 0.6*vnoise(fx*0.2, fy*0.2, 935)
			switch ph := math.Mod(v, 11); {
			case ph < 1.8:
				l -= 0.25 * kk
			case ph < 4.4:
				l += 0.1 * kk
			}
			dx, dy := inv(x, y)
			edge := frost + (vnoise(dx*0.5, 0.3, 936)-0.5)*7 + hashf(x, 3, 939)*3
			if dy < edge {
				return kingFrost.soft(l*0.9+0.2-clamp01((dy-edge+6)/6)*0.25, x, y)
			}
			return kingFur.soft(l, x, y)
		}
	}
	nails := func(sh *furShape, foot knob, far bool) {
		for i, nx := range []float64{-1.1, 1.6, 4} {
			if far && i == 0 {
				continue
			}
			px, py := int(foot[0]+nx*k), int(foot[1]+3.4*k)
			for _, q := range [][2]int{{px, py}, {px + 1, py}, {px, py - 1}, {px + 1, py - 1}} {
				if sh.has(q[0], q[1]) {
					d := 0
					if far {
						d = 2
					}
					c.put(q[0], q[1], kingTusk.at(2+min(1, i)-d))
				}
			}
		}
	}
	// A leg planted on the ground, from its shoulder or hip.
	leg := func(x, hy, r, s float64, far bool) {
		bias := 0.0
		if far {
			bias = -0.2
		}
		top := B(x, hy, r)
		foot := G(x+s+0.6, gy-4.6, r*0.8)
		midY := lerpf(top[1]/k-oy, gy-5, 0.45)
		sh := furPaint(c, 0.3, 0, func(px, py int, l, ny, t float64) color.RGBA {
			return coat(bias-0.1*t, -99)(px, py, l, ny, t)
		}, []knob{top, G(x+s*0.35+drop*0.5, midY, r*0.86), G(x+s, gy-6.5, r*0.78), foot})
		nails(sh, foot, far)
	}
	// A foreleg off the ground, folded, when it rears.
	raised := func(x, hy, r, fold float64, far bool) {
		bias := 0.0
		if far {
			bias = -0.2
		}
		foot := B(x+13+fold, hy+18-fold*0.4, r*0.8)
		sh := furPaint(c, 0.3, 0, func(px, py int, l, ny, t float64) color.RGBA {
			return coat(bias-0.1*t, -99)(px, py, l, ny, t)
		}, []knob{B(x, hy, r), B(x+5, hy+11, r*0.86), B(x+11+fold, hy+16-fold*0.4, r*0.78), foot})
		nails(sh, foot, far)
	}

	// The tail, then the legs of the far side.
	furPaint(c, 0, 0, coat(-0.12, -99), []knob{B(8, 40, 1.8), B(4.6, 47, 1.2), B(4, 54, 1.9)})
	far := sw
	if far < 0 {
		far /= 3
	}
	if rear != 0 {
		leg(31, 53, 6.8, 5, true)
		raised(59, 51, 6.6, 6, true)
		leg(21, 55, 7.6, -3, false)
	} else {
		leg(30, 52, 6.8, far*0.8, true)
		leg(59, 51, 6.8, far, true)
		leg(20.5, 54, 7.5, -sw*0.8, false)
		leg(49, 54, 7.5, -sw, false)
	}
	// The body: a sloping back, a great hump over the shoulders.
	body := furPaint(c, 0.25, 0, coat(0, 27),
		[]knob{B(19.5, 48, 11.8), B(33, 45, 13.8), B(47, 41, 17)},
		[]knob{B(49, 29.5, 13), B(43, 35, 12.5)})
	furHang(c, body, 0, c.w-1, kingFur, 0.34, func(x int) float64 {
		dx, _ := inv(x, int((60+oy)*k))
		return (4.5 + 6*math.Sin(clamp01((dx-10)/52)*math.Pi)) * k
	})
	if rear != 0 {
		raised(49, 54, 7.4, 0, false)
	}
	// Old scars across the flank.
	for i, q := range [][4]float64{{24, 39, 33, 50}, {28, 37, 37, 48}, {33, 37, 40, 46}} {
		x0, y0 := T(q[0], q[1])
		x1, y1 := T(q[2], q[3])
		for w := 0.0; w <= 1; w++ {
			c.line(x0+w, y0, x1+w, y1, func(x, y int, t float64) color.RGBA {
				if !body.has(x, y) || hashf(x, y, 941) < 0.12 {
					return color.RGBA{}
				}
				return mix(kingScar, kingFur[3], math.Abs(t-0.5)*1.4+float64(i%2)*0.15)
			})
		}
	}

	// The head swings about the neck, carrying trunk and tusks with it.
	h := func(x, y, r float64) knob {
		X, Y := rot(x, y, 58, 36, a)
		return B(X, Y+hdy, r)
	}
	ivory := func(bias float64) pelt {
		return func(x, y int, l, ny, t float64) color.RGBA {
			l = 0.26 + l*0.74 - (1-t)*0.16 + bias
			if math.Mod(t*9+0.2, 1) < 0.1 && t < 0.6 {
				l -= 0.12 // the rings of its years
			}
			return kingTusk.soft(l, x, y)
		}
	}
	tusk := func(dx, dy, bias float64) {
		furPaint(c, 0.35, 0, ivory(bias), []knob{h(73+dx, 43+dy, 3.4), h(76+dx, 55+dy, 3.4), h(83+dx, 64.5+dy, 3.1),
			h(93+dx, 67+dy, 2.7), h(102+dx, 62.5+dy, 2.1), h(107+dx, 53+dy, 1.5), h(107.5+dx, 44+dy, 0.8)})
	}
	tusk(3.5, -3, -0.26) // the far tusk
	trunk := []knob{h(75.5, 39, 5.9), h(80, 50, 5), h(81, 61, 3.9), h(79.5, 69, 3.1), h(82.5, 73.5, 2.4)}
	switch p {
	case bossAttack: // tucked back out of the way of the tusks
		trunk = []knob{h(75.5, 39, 5.9), h(80, 48, 5), h(84, 55, 3.8), h(89, 53, 3), h(89, 48, 2.4)}
	case bossRoar, bossSpecial: // flung up, trumpeting
		trunk = []knob{h(75.5, 39, 5.9), h(84, 40, 5), h(91, 33, 3.9), h(92.5, 23, 3.1), h(96.5, 18, 2.5)}
	case bossLurk: // curled under
		trunk = []knob{h(75.5, 39, 5.9), h(80.5, 49, 5), h(80, 57, 3.9), h(74.5, 58, 3.1), h(71.5, 53, 2.5)}
	}
	furPaint(c, 0.3, 0, func(x, y int, l, ny, t float64) color.RGBA {
		l = 0.2 + l*0.72 + (vnoise(float64(x)*0.9, float64(y)*0.3, 942)-0.5)*0.14
		if math.Mod(t*17+0.3, 1) < 0.3 {
			l -= 0.1 // the rings of its wrinkles
		}
		return kingSkin.soft(l, x, y)
	}, trunk)
	chin := furPaint(c, 0.25, 0, coat(-0.1, -99), []knob{h(68.5, 43, 5.4), h(72, 45, 3.8)})
	furHang(c, chin, 0, c.w-1, kingFur, 0.3, func(int) float64 { return 6 })
	// The skull: a high dome crowned with frost.
	frost := 22.0 + hdy
	if a != 0 || rear != 0 {
		frost = -99 // the crown is laid on below instead, turned with the head
	}
	furPaint(c, 0.42, 0, coat(-0.02, frost),
		[]knob{h(67, 31, 10.8), h(73, 38, 7.4)},
		[]knob{h(66.5, 21.5, 8.2), h(68.5, 28, 9)})
	if frost < 0 {
		furPaint(c, 0, 0, coat(0, 999), []knob{h(63.5, 16.8, 3.2), h(68, 15.6, 3.6), h(72, 18, 2.8)})
	}
	tusk(0, 0, 0.05)
	furPaint(c, 0.5, 0, coat(0.08, -99), []knob{h(61, 29.5, 3.9), h(62, 36.5, 3.2)}) // the ear
	// The eye, small and pale under its brow.
	e := h(73.5, 31, 0)
	ex, ey := int(e[0]), int(e[1])
	for _, q := range [][2]int{{-1, -1}, {0, -1}, {1, -1}, {2, -1}, {-1, 0}, {2, 0}, {0, 1}, {1, 1}} {
		c.put(ex+q[0], ey+q[1], kingFur[0])
	}
	c.put(ex, ey, hex(0xbfe6ff))
	c.put(ex+1, ey, hex(0x5aa0d8))
	// The shag of the crown and along the back, white with frost.
	shag := func(x, y float64, rp ramp, n int) {
		X, Y := T(x, y)
		for yy := int(Y); yy > 1; yy-- {
			if c.at(int(X), yy).A < 128 {
				for q := 0; q < n; q++ {
					c.put(int(X), yy-q, rp.at(len(rp)-2-q%2))
				}
				break
			}
		}
	}
	for i := -8.0; i <= 7; i += 0.8 {
		if hashf(int(i*10), 3, 943) < 0.6 {
			X, Y := rot(66.5+i, 24, 58, 36, a)
			shag(X, Y+hdy, kingFrost, 1+int(hashf(int(i*10), 4, 943)*3))
		}
	}
	for x := 12.0; x < 58; x += 0.8 {
		if hashf(int(x*10), 5, 944) < 0.4 {
			shag(x, 50, kingFrost, 1+int(hashf(int(x*10), 6, 944)*2.5))
		}
	}
	furEdge(c, int((gy+oy)*k), 0.74)
	if p == bossAttack {
		// The sweep of the tusks.
		t0 := h(107, 46, 0)
		for i := 0; i <= 300; i++ {
			f := float64(i) / 300
			an := -1.25 + f*1.5
			th := 3.6 * math.Sin(f*math.Pi) * (0.3 + 0.7*f)
			for w := 0.0; w <= th; w += 0.5 {
				R := 30*k + w
				qx, qy := t0[0]-18*k+math.Cos(an)*R, t0[1]+6*k+math.Sin(an)*R*0.9
				if !c.opaque(int(qx), int(qy)) && qy < (gy+oy)*k {
					c.put(int(qx), int(qy), withAlpha(mix(kingFrost[3], colWhite, f), 225))
				}
			}
		}
	}
	return bossCrop(c, (43+ox)*k, (gy-2+oy)*k)
}
