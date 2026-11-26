package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The head of the tall figure (tallfig.go): the skull covers image columns
// 20 to 33 and rows 27 to 43, the eyes are on rows 35 to 37. It wears the
// hair of lookhd_head.go's big head, cut to its size: headMap carries each
// lock over.

// Eyes, one letter a pixel: K lid, W white, P pupil, I iris, i iris lit
// from below. The outer corner is on the left.
var (
	eyeTall     = []string{"KKK", "WPI", "WPi"}
	eyeTallSide = []string{"KKK", "WIP", "WiP"}
)

// headMap carries a point of the big head, whose skull is 36 by 33 about
// (27, 31), to the tall figure's, 14 by 17 about (cx, 35).
type headMap struct{ cx, kx, ky float64 }

const headMapY = 35.0

func (m headMap) x(x float64) float64 { return m.cx + (x-27)*m.kx }
func (m headMap) y(y float64) float64 { return headMapY + (y-31)*m.ky }

func (m headMap) pts(p ...float64) []float64 {
	q := make([]float64, len(p))
	for i := 0; i < len(p); i += 2 {
		q[i], q[i+1] = m.x(p[i]), m.y(p[i+1])
	}
	return q
}

func (m headMap) ell(cx, cy, rx, ry float64) shape {
	return sEll(m.x(cx), m.y(cy), rx*m.kx, ry*m.ky)
}

func (m headMap) box(x0, y0, x1, y1, r float64) shape {
	return sBox(m.x(x0), m.y(y0), m.x(x1), m.y(y1), r*m.kx)
}

func (m headMap) poly(p ...float64) shape { return sPoly(m.pts(p...)...) }

func (m headMap) lock(x0, y0, mx, my, x1, y1, w0, w1 float64) shape {
	return sLock(m.x(x0), m.y(y0), m.x(mx), m.y(my), m.x(x1), m.y(y1), math.Max(0.9, w0*0.42), math.Max(0.45, w1*0.6))
}

func (m headMap) above(y float64) shape { return sAbove(m.y(y)) }

func (m headMap) over(f func(x float64) float64) shape {
	return sOver(func(x float64) float64 { return m.y(f(27 + (x-m.cx)/m.kx)) })
}

func (m headMap) jag(s shape, amp, period, y0 float64, seed uint32) shape {
	return s.jag(amp*0.6, period*0.6, m.y(y0), seed)
}

// tallLagged is s with its lower end trailing by the figure's lag: hair
// that hangs.
func (g *figGen) tallLagged(s shape) shape {
	if g.lagX == 0 && g.lagY == 0 {
		return s
	}
	lx, ly := g.lagX, g.lagY
	t := func(y float64) float64 { v := clamp01((y - 41) / 15); return v * v }
	return shape{s.x0 - math.Abs(lx), s.y0 - math.Abs(ly), s.x1 + math.Abs(lx), s.y1 + math.Abs(ly), func(x, y float64) float64 {
		k := t(y)
		return s.d(x-lx*k, y-ly*k)
	}}
}

// tallBraid plaits hair in ramp r down from (x, y0) to y1 on layer l:
// knots lying over each other left and right, a tie, and the tuft below it.
func (g *figGen) tallBraid(l *figLayer, r ramp, x, y0, y1 float64, tie color.RGBA) {
	f := &g.f
	k := 0
	for y := y0; y < y1-4; y += 2.3 {
		off := 0.6
		if k%2 == 1 {
			off = -0.6
		}
		s := sEll(x+off, y+1.6, 1.9, 1.9)
		if t := (y - y0) / (y1 - y0); g.lagX != 0 {
			s = s.moved(math.Round(g.lagX*1.5*t*t), 0) // the braid trails; see limbs.go
		}
		f.fill(l, s, bevel(r, s, 1.9, 0.06))
		k++
	}
	if g.lagX != 0 || g.lagY != 0 {
		x += math.Round(g.lagX * 1.5)
		y1 += math.Round(g.lagY)
	}
	t := sBox(x-1.7, y1-4, x+1.7, y1-2, 0.6)
	f.fill(l, t, func(px, py int, _ float64) color.RGBA {
		if py == int(y1-4) && px <= int(x) {
			return mix(tie, colWhite, 0.45)
		}
		if px >= int(x+1) {
			return mix(tie, colInk, 0.3)
		}
		return tie
	})
	tuft := sLock(x, y1-2.2, x+0.4, y1-0.8, x-0.3, y1+0.8, 1.4, 0.5)
	f.fill(l, tuft, bevel(r, tuft, 1.5, -0.05))
}

func (g *figGen) tallHead(d sim.Dir, shut bool) {
	f := &g.f
	l := g.look
	sk, hr := g.sk, g.hr
	side, up := d == sim.DirRight, d == sim.DirUp
	T := g.T
	iris := irisColors[l.Skin]
	pal := func(ch byte) (color.RGBA, bool) {
		switch ch {
		case 'K':
			return eyeDark, true
		case 'W':
			return eyeWhite, true
		case 'P':
			return mix(eyeDark, iris, 0.12), true
		case 'I':
			return iris, true
		case 'i':
			return mix(iris, colWhite, 0.42), true
		}
		return color.RGBA{}, false
	}
	blush := skinTones[l.Skin].blush
	lips := mix(blush, mouthDark, 0.5)
	brow := hr[2]
	if luma(brow) < 45 {
		brow = mix(hr[3], sk[2], 0.3) // black brows would run into the eyes
	}
	if l.Female {
		brow = mix(brow, sk[3], 0.25)
	}

	m := headMap{27, 7.0 / 18, 8.5 / 16.5}
	if side {
		m = headMap{27.5, 7.4 / 18, 8.5 / 16.5}
	}
	skull := sSuper(m.cx, headMapY, 18*m.kx, 8.5, 2.5)
	switch {
	case side: // the nape
		skull = skull.sub(sEll(m.cx-8.4, 44.5, 4.6, 5))
	case l.Female: // a finer jaw
		skull = skull.and(sEll(m.cx, headMapY-3, 18*m.kx+0.6, 12))
	}
	// Ears, behind the skull's edge when seen from the front or the back.
	if !side {
		for _, ex := range [2]float64{19.9, 34.1} {
			e := sEll(ex, 36.6, 1.4, 2.1)
			f.fill(T, e, func(x, y int, dd float64) color.RGBA {
				if dd < -0.9 {
					return sk[2]
				}
				return sk[3]
			})
		}
	}
	f.fill(T, skull, func(x, y int, dd float64) color.RGBA {
		if side { // in profile the whole face is in the light
			if y >= 42 && dd > -1.6 {
				return sk[3]
			}
			return sk[4]
		}
		switch v := skull.lum(x, y, 4.5); {
		case v < 0.3:
			return sk[2]
		case v < 0.5 || y >= 42:
			return sk[3]
		}
		return sk[4]
	})
	skinPart := f.n

	dot := func(x, y int, c color.RGBA) { f.dot(T, x, y, c) }
	switch d {
	case sim.DirDown:
		if shut {
			for x := 0; x < 3; x++ {
				dot(22+x, 36, eyeDark)
				dot(29+x, 36, eyeDark)
			}
		} else {
			f.stamp(T, 22, 35, eyeTall, false, pal)
			f.stamp(T, 29, 35, eyeTall, true, pal)
		}
		for x := 21; x <= 24; x++ { // the brows
			if l.Female && x == 21 {
				dot(x, 34, brow)
				dot(53-x, 34, brow)
				continue
			}
			dot(x, 33, brow)
			dot(53-x, 33, brow)
			if !l.Female && x >= 23 {
				dot(x, 34, mix(brow, sk[3], 0.35))
				dot(53-x, 34, mix(brow, sk[3], 0.35))
			}
		}
		if l.Female {
			if !shut { // lashes
				dot(21, 35, eyeDark)
				dot(32, 35, eyeDark)
			}
			for _, x := range [4]int{22, 23, 30, 31} { // rosy cheeks
				dot(x, 39, mix(sk[4], blush, 0.6))
			}
			dot(26, 41, lips)
			dot(27, 41, lips)
			dot(25, 41, mix(sk[3], lips, 0.45))
			dot(28, 41, mix(sk[3], lips, 0.45))
		} else {
			dot(22, 39, mix(sk[4], blush, 0.7)) // weathered cheeks
			dot(31, 39, mix(sk[4], blush, 0.7))
		}
		// The nose: a glint on its bridge, a shadow under it.
		dot(26, 37, sk[5])
		dot(26, 38, sk[5])
		dot(27, 38, sk[3])
		dot(26, 39, sk[3])
		dot(27, 39, sk[2])
	case sim.DirRight:
		nose := sPoly(34.4, 35.6, 36.4, 38.6, 36, 39.6, 34, 39.8)
		if l.Female {
			nose = sPoly(34.4, 36.2, 35.9, 38.6, 35.6, 39.5, 34, 39.6)
		}
		f.add(T, nose, func(x, y int, _ float64) color.RGBA {
			if y >= 39 {
				return sk[3]
			}
			return sk[4]
		})
		dot(34, 40, sk[2])
		if shut {
			for x := 31; x <= 33; x++ {
				dot(x, 36, eyeDark)
			}
		} else {
			f.stamp(T, 31, 35, eyeTallSide, false, pal)
		}
		for x := 30; x <= 33; x++ {
			dot(x, 33, brow)
			if !l.Female && x >= 32 {
				dot(x, 34, mix(brow, sk[3], 0.35))
			}
		}
		// The ear.
		ear := sEll(26.6, 36.8, 1.6, 2.3)
		f.add(T, ear, func(x, y int, dd float64) color.RGBA {
			switch {
			case dd > -0.8:
				return sk[3]
			case x <= 26:
				return sk[2]
			}
			return sk[4]
		})
		if l.Female {
			dot(34, 35, eyeDark)
			dot(30, 39, mix(sk[4], blush, 0.6))
			dot(31, 39, mix(sk[4], blush, 0.6))
			dot(33, 41, lips)
			dot(34, 41, lips)
		} else {
			dot(31, 39, mix(sk[4], blush, 0.7))
		}
	}

	// The hair.
	const hk = 0.45
	cx, cy := m.x(27), m.y(10)
	if side {
		cx = m.x(23)
	}
	hs := func(s shape) shader { return hairShadeK(hr, s, 3.6, cx, cy, g.seed, hk) }
	hair := func(s shape) { s = g.tallLagged(s); f.fill(T, s, hs(s)) }
	lock := func(x0, y0, mx, my, x1, y1, w0, w1 float64) {
		hair(m.lock(x0, y0, mx, my, x1, y1, w0, w1))
	}
	stubCol := mix(sk[3], hr[2], 0.42)
	stubble := func(s shape) {
		f.fill(T, s, func(x, y int, dd float64) color.RGBA {
			if (x+2*y)%5 == 0 {
				return mix(stubCol, hr[1], 0.25)
			}
			if dd > -1.2 && s.lum(x, y, 3) < 0.5 {
				return mix(stubCol, hr[1], 0.2)
			}
			return stubCol
		})
	}
	bun := func(bx float64) {
		bx = m.x(bx)
		b := sEll(bx, 24.4, 3.6, 3.2)
		f.fill(T, b, func(x, y int, _ float64) color.RGBA {
			i := int(b.lum(x, y, 3)*6*0.74) + 1
			// strands wound round the knot
			a := math.Atan2(float64(y)+0.5-24.4, float64(x)+0.5-bx)
			rr := math.Hypot(float64(x)+0.5-bx, float64(y)+0.5-24.4)
			if int(math.Floor(rr*1.6+a*1.2+20))%3 == 0 {
				i--
			}
			return hr.at(max(1, i))
		})
		t := sBox(bx-2.6, 26.2, bx+2.6, 27.8, 0.6)
		f.fill(T, t, func(x, y int, _ float64) color.RGBA {
			if y == 26 && x < int(bx) {
				return mix(hairTie, colWhite, 0.4)
			}
			if x > int(bx) {
				return mix(hairTie, colInk, 0.3)
			}
			return hairTie
		})
	}
	hairLo := f.n + 1
	stubLo, stubHi := uint16(0), uint16(0)
	switch d {
	case sim.DirDown:
		E := m.ell(27, 28.5, 19.3, 19.5)
		wide := m.ell(27, 29, 21, 20.5)
		switch l.Hair {
		case sim.HairShort: // swept over the top, the sides cropped
			hair(E.and(m.over(func(x float64) float64 { return 20.5 + 0.02*(x-27)*(x-27) })).or(E.and(m.above(31)).sub(m.box(11.8, 0, 42.2, 60, 0))))
			lock(38, 14, 41, 19, 40.5, 25.5, 3.4, 0.7)
			lock(34, 12, 22, 13, 12.5, 25, 4.4, 0.7)
			lock(36, 13, 27, 16, 19.5, 26.5, 4.2, 0.7)
			lock(37, 13.5, 32, 19, 27, 26, 3.8, 0.7)
			lock(38, 14, 36, 20, 33.5, 25, 3.4, 0.7)
		case sim.HairSpiky: // a crest down the middle, the sides shaved
			stubLo = f.n + 1
			stubble(m.ell(27, 30, 18.6, 17.4).and(m.over(func(x float64) float64 { return 21.5 + 0.02*(x-27)*(x-27) })))
			stubHi = f.n
			hair(m.ell(27, 19.5, 7.6, 5.4))
			lock(22, 21, 18, 12, 20.5, 5, 4.4, 0.7)
			lock(32, 21, 36, 12, 33.5, 6, 4.4, 0.7)
			lock(27, 22, 25.5, 10, 27.5, 1.5, 5.4, 0.7)
		case sim.HairBob: // chin length with a straight fringe, a thin braid at one side
			hair(m.jag(wide.and(m.above(46.5)).sub(m.box(12.6, 23, 41.4, 60, 3)), 1.6, 3.4, 43, 41))
			for i := 0; i < 5; i++ {
				x := 14.5 + float64(i)*6.2
				lock(x+0.6, 17, x+0.2, 22, x-0.4+hashf(i, 4, g.seed), 26.5+hashf(i, 5, g.seed), 3.6, 1)
			}
			g.tallBraid(T, hr, 19.6, 42, 51, hairTie)
		case sim.HairLong: // loose, parted in the middle, down past the shoulders
			back := g.tallLagged(sBox(18.6, 34, 35.4, 57, 3).jag(1.6, 3, 54, 43))
			f.fill(g.B, back, hairShadeK(hr, back, 5, cx, cy, g.seed+3, hk))
			opening := sFn(func(x, y float64) float64 {
				hw := 5.6 * math.Pow(clamp01((y-27.6)/6.6), 0.7)
				return math.Max(27.6-y, math.Abs(x-27.2)-hw)
			})
			curt := sBox(17.8, 32, 21.4, 55, 1.6).or(sBox(32.6, 32, 36.2, 55, 1.6)).jag(1.3, 2.4, 52, 45)
			hair(wide.and(m.above(34)).sub(opening).or(curt))
			lock(14, 22, 10.6, 36, 12, 54, 2.3, 0.7)
			lock(40, 22, 43.4, 36, 42, 54, 2.3, 0.7)
		case sim.HairShaggy: // a mane never cut or combed: ragged to the shoulders, tufts standing out of it
			back := g.tallLagged(sBox(16.4, 33, 37.6, 55, 3.4).jag(2.6, 2.7, 42, 47))
			f.fill(g.B, back, hairShadeK(hr, back, 5, cx, cy, g.seed+3, hk))
			opening := sFn(func(x, y float64) float64 {
				u := x/2.3 + 40
				top := 30.3 + 1.5*math.Abs(u-math.Floor(u)-0.5)*hashf(int(math.Floor(u)), 2, g.seed) // a ragged fringe
				hw := 6.3 * math.Pow(clamp01((y-top)/2.4), 0.6)
				return math.Max(top-y, math.Abs(x-27.3)-hw)
			})
			mane := m.ell(27, 28.6, 22.6, 21.4).jag(1.3, 2.6, 0, 45)
			curt := sBox(16.4, 31, 21.3, 52.5, 1.8).or(sBox(32.7, 31, 37.6, 52.5, 1.8)).jag(2.4, 2.3, 42, 49)
			hair(mane.and(m.above(35)).sub(opening).or(curt))
			for _, k := range [][8]float64{
				{22.4, 26.8, 18.6, 25.4, 16.2, 28.4, 1.6, 0.5}, // tufts standing out of it
				{31.8, 26.6, 35.8, 25, 38, 28, 1.6, 0.5},
				{25.4, 25.6, 26.6, 22.6, 29.6, 22.8, 1.4, 0.5},
				{24, 27.6, 22.6, 30.6, 22.2, 33.6, 1.5, 0.6}, // locks over the brow
				{29.6, 27.6, 32.4, 30, 33.2, 33.2, 1.5, 0.6},
				{27, 26.8, 27.8, 29.4, 26.6, 31.6, 1.3, 0.5},
				{19, 36, 16.4, 43, 18.2, 50.5, 1.5, 0.6}, // and down the cheeks
				{35.2, 36, 37.6, 43, 36, 50, 1.5, 0.6},
			} {
				hair(sLock(k[0], k[1], k[2], k[3], k[4], k[5], k[6], k[7]))
			}
		case sim.HairPonytail, sim.HairBun: // pulled back off the brow
			if l.Hair == sim.HairBun {
				bun(27)
			}
			hair(m.ell(27, 28.5, 19, 18.4).and(m.over(func(x float64) float64 { return 22.5 + 0.028*(x-27)*(x-27) - 1.5*math.Exp(-(x-27)*(x-27)/14) })))
			lock(13.5, 22, 10.6, 26, 12.2, 32.5, 1.7, 0.6)
			if l.Hair == sim.HairBun {
				lock(40.5, 22, 43.4, 26, 41.8, 32.5, 1.7, 0.6)
			}
		}
	case sim.DirUp:
		E := m.ell(27, 28.5, 19.3, 19.5)
		wide := m.ell(27, 29, 21, 20.5)
		switch l.Hair {
		case sim.HairShort:
			hair(m.jag(E.and(m.above(43)), 2, 3.6, 38.5, 41))
		case sim.HairSpiky:
			stubLo = f.n + 1
			stubble(m.ell(27, 30, 18.6, 17.4).and(m.above(43.5)))
			stubHi = f.n
			hair(sPath(2.7, 1.5, m.x(27), m.y(10), m.x(27), m.y(37)))
			lock(23, 14, 19, 9, 21, 4, 4, 0.7)
			lock(31, 14, 35, 9, 33.5, 5, 4, 0.7)
			lock(27, 14, 26, 6, 27.5, 1.5, 5, 0.7)
			lock(25, 22, 21, 19, 21.5, 14, 3.2, 0.7)
			lock(29, 24, 33, 21, 32.5, 16, 3.2, 0.7)
			lock(26, 31, 22.5, 28, 22.5, 23.5, 2.8, 0.7)
		case sim.HairBob:
			hair(m.jag(wide.and(m.above(46.5)), 1.8, 3.4, 42.5, 41))
		case sim.HairLong:
			hair(wide.and(m.above(42)).or(sBox(18, 30, 36, 57.5, 4.5)).jag(1.6, 3, 54, 43))
		case sim.HairShaggy:
			mane := m.ell(27, 28.6, 22.6, 21.4)
			hair(mane.and(m.above(42)).or(sBox(16.6, 30, 37.4, 55, 4.5)).jag(2.6, 2.7, 0, 47))
			for _, k := range [][8]float64{
				{22.4, 26.8, 18.6, 25.4, 16.2, 28.4, 1.6, 0.5},
				{31.8, 26.6, 35.8, 25, 38, 28, 1.6, 0.5},
				{25.4, 25.6, 26.6, 22.6, 29.6, 22.8, 1.4, 0.5},
				{20.6, 33, 18.6, 43, 20, 51.5, 1.9, 0.6}, // matted locks down the back
				{24.6, 35, 23.2, 45, 25, 54, 2, 0.6},
				{29.4, 34, 31.4, 44, 29.6, 53, 2, 0.6},
				{33.4, 33, 35.6, 42, 34.2, 50.5, 1.9, 0.6},
			} {
				hair(sLock(k[0], k[1], k[2], k[3], k[4], k[5], k[6], k[7]))
			}
		case sim.HairPonytail, sim.HairBun:
			if l.Hair == sim.HairBun {
				bun(27)
			}
			hair(m.jag(E.and(m.above(44.5)), 1.2, 4, 41, 41))
		}
	case sim.DirRight:
		E := m.ell(25.5, 28.5, 19.8, 19.5)
		wide := m.ell(25, 29, 20.6, 20.5)
		short := m.poly(-9, -9, 60, -9, 60, 20, 47, 24.5, 36, 25.5, 30, 29.5, 25, 30.5, 20.5, 34, 19.5, 40, 15, 44.6, -9, 44.6)
		swept := m.poly(-9, -9, 60, -9, 60, 16, 45, 21, 38, 23, 31, 28, 25, 30.5, 20.5, 34, 19.5, 40, 15, 44.6, -9, 44.6)
		bobCut := m.poly(-9, -9, 60, -9, 60, 21, 47, 25.5, 37, 26, 33, 30, 31.5, 40, 30, 46.5, -9, 46.5)
		fringe := func() {
			lock(27, 15, 31, 22, 31.5, 29.5, 3.4, 0.7)
			lock(31, 14, 38, 19, 40, 27, 3.8, 0.7)
			lock(36, 13, 44, 16, 47.5, 25.5, 3.8, 0.7)
		}
		switch l.Hair {
		case sim.HairShort:
			hair(m.jag(E.and(short), 1.6, 3.6, 39, 41))
			fringe()
		case sim.HairSpiky:
			stubLo = f.n + 1
			stubble(m.ell(26.5, 30, 18.8, 17.4).and(short))
			stubHi = f.n
			hair(m.box(10, 10.5, 45, 18.5, 4))
			lock(16, 16, 13, 10, 13.5, 3.5, 4.2, 0.7)
			lock(24, 15, 22, 8, 23, 1.5, 4.4, 0.7)
			lock(32, 15, 31, 8, 32.5, 1.5, 4.4, 0.7)
			lock(40, 16, 41, 10, 42.5, 4.5, 4, 0.7)
		case sim.HairBob:
			hair(m.jag(wide.and(bobCut), 1.6, 3.4, 43, 41))
			fringe()
			g.tallBraid(T, hr, m.x(20.5), 41, 50, hairTie)
		case sim.HairLong:
			hang := sPoly(m.x(8), m.y(19), m.x(31.6), m.y(19), m.x(31), m.y(40), 26, 47, 23.5, 56.5, 17.5, 56.5, 17, 36).jag(1.5, 2.6, 53, 43)
			hair(wide.and(bobCut).and(m.above(42)).or(hang))
			fringe()
			lock(30, 28, 29, 40, 25, 51, 2.3, 0.7)
		case sim.HairShaggy:
			mane := m.ell(25, 28.6, 22, 21.4)
			hang := sPoly(m.x(5), m.y(18), m.x(31.6), m.y(18), m.x(31.4), m.y(40), 26.8, 46, 25.4, 54.5, 16.4, 54.5, 15.4, 36).jag(2.6, 2.7, 38, 47)
			hair(mane.and(bobCut).and(m.above(42)).jag(1.3, 2.6, 0, 45).or(hang))
			lock(27, 15, 31, 22, 31.5, 30.5, 3.6, 0.8) // a heavy fringe, down to the brow
			lock(31, 14, 39, 19, 41, 28.5, 4, 0.8)
			lock(36, 13, 45, 16, 48.6, 27, 4, 0.8)
			lock(30, 28, 30, 40, 27, 50, 2.8, 0.8) // down the cheek
			for _, k := range [][8]float64{
				{20.6, 28.4, 17, 27, 15, 30, 1.6, 0.5}, // tufts standing out behind
				{19.6, 34, 15.6, 34.6, 14, 38.4, 1.6, 0.5},
				{24, 25.4, 24.6, 22.6, 27.6, 22.4, 1.4, 0.5},
				{20, 40, 17.6, 46, 19.6, 52, 1.8, 0.6},
			} {
				hair(sLock(k[0], k[1], k[2], k[3], k[4], k[5], k[6], k[7]))
			}
		case sim.HairPonytail:
			hair(E.and(swept))
			g.tallBraid(T, hr, m.x(11.5), 39, 58, hairTie)
		case sim.HairBun:
			bun(20)
			hair(E.and(swept))
		}
	}
	hairHi := f.n

	// The hair's shadow on the brow.
	for y := 26; y < 45; y++ {
		for x := 18; x < 38; x++ {
			c, p := f.at(T, x, y)
			if p != skinPart {
				continue
			}
			for _, o := range [2][2]int{{0, -1}, {-1, -1}} {
				if _, q := f.at(T, x+o[0], y+o[1]); q >= hairLo && q <= hairHi && (q < stubLo || q > stubHi) {
					for i := 2; i < len(sk); i++ {
						if sk[i] == c {
							f.dot(T, x, y, sk[i-1])
						}
					}
					break
				}
			}
		}
	}

	// A man's beard: a moustache parted under the nose, a full beard and a
	// short braid with a gold bead.
	if !l.Female {
		br := g.br
		bs := func(s shape) shader { return hairShadeK(br, s, 2.6, m.cx, 35, g.seed+9, hk) }
		switch d {
		case sim.DirDown:
			b := sPoly(20, 36.5, 21.6, 37, 22.3, 39.7, 24.6, 40.5, 27, 40.8, 29.4, 40.5, 31.7, 39.7, 32.4, 37, 34, 36.5,
				34, 41.5, 32.6, 45.5, 30, 48, 27, 49, 24, 48, 21.4, 45.5, 20, 41.5).jag(1.1, 2.2, 45, 51)
			f.fill(T, b, bs(b))
			dot(26, 41, mouthDark)
			dot(27, 41, mouthDark)
			for _, s := range []shape{
				sLock(26.8, 40, 24.6, 39.5, 22.4, 41.8, 1.3, 0.6),
				sLock(27.2, 40, 29.4, 39.5, 31.6, 41.8, 1.3, 0.6),
			} {
				f.fill(T, s, bevel(br, s, 1.4, -0.04))
			}
			g.tallBeardBraid(27, 48.4)
		case sim.DirRight:
			b := sPoly(25.6, 38, 27.6, 38.8, 28.6, 40.6, 31, 41.1, 33.2, 40.7, 35.3, 41, 35.9, 43, 35.4, 45.6, 34, 47.8, 31.6, 48.6,
				29.4, 47.4, 27.2, 44.8, 25.8, 41.5).jag(1.1, 2.2, 45, 51)
			f.fill(T, b, bs(b))
			s := sLock(35.4, 40, 33.2, 39.5, 31.2, 41.8, 1.3, 0.6)
			f.fill(T, s, bevel(br, s, 1.4, -0.04))
			g.tallBeardBraid(32.4, 48)
		}
	}
	switch l.Hair {
	case sim.HairPonytail:
		switch {
		case up && l.Female:
			g.tallBraid(T, hr, 22, 42, 60, hairTie)
			g.tallBraid(T, hr, 32, 42, 60, hairTie)
		case up:
			g.tallBraid(T, hr, 27, 42, 62, hairTie)
		case side:
		case l.Female:
			g.tallBraid(T, hr, 20.2, 40, 58, hairTie)
			g.tallBraid(T, hr, 33.8, 40, 58, hairTie)
		default: // over one shoulder
			g.tallBraid(T, hr, 33.8, 40, 58, hairTie)
		}
	}
}

// tallBeardBraid hangs the beard's short plait from (x, y), with its gold
// bead.
func (g *figGen) tallBeardBraid(x, y float64) {
	f := &g.f
	x, y = x+math.Round(g.lagX*0.6), y+math.Round(g.lagY*0.6) // it swings a little; see limbs.go
	k := sEll(x, y+0.4, 1.7, 1.5)
	f.fill(g.T, k, bevel(g.br, k, 1.5, 0.05))
	bead := sBox(x-1.5, y+1.6, x+1.5, y+3.8, 0.7)
	f.fill(g.T, bead, func(px, py int, _ float64) color.RGBA {
		return toneOf(goldHD, bead.lum(px, py, 1.2)+0.1)
	})
	f.dot(g.T, int(x-1), int(y+1.6), goldHD[5])
	f.fill(g.T, sBox(x-0.8, y+3.8, x+0.8, y+5, 0.4), plain(g.br[2]))
}
