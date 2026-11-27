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
	if pilotOn() { // a style pilot's head: see charpilot_head.go
		g.pHead(d, shut)
		return
	}
	f := &g.f
	l := g.look
	sk, hr := g.sk, g.hr
	side, up := d == sim.DirRight, d == sim.DirUp
	T := g.T
	// The eye's colours: the iris dark, its colour only a tinge; the white
	// dulled by the shade it is in; the lashes and the line of the lid.
	irisCol := mix(irisColors[l.Skin], eyeDark, 0.62)
	whiteCol := mix(eyeWhite, sk[4], 0.3)
	lash := mix(eyeDark, sk[2], 0.35)
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
		// A grown face: each eye small and set in the shade of the brow, a
		// dark iris with a little of the white beside it towards the nose;
		// the nose lit down its bridge. A man's brows lie heavy on a ridge,
		// the eyes deep under it, and his nose is broad; a woman's brows
		// are fine, her upper lids marked, her lips full.
		for _, e := range [2][2]int{{22, 1}, {31, -1}} {
			x0, x1, x2 := e[0], e[0]+e[1], e[0]+2*e[1] // the outer corner, the iris, the inner corner
			if l.Female {
				dot(x0, 33, mix(brow, sk[3], 0.5))
				dot(x1, 33, brow)
				dot(x2, 33, brow)
				dot(x0, 34, lash)
				dot(x1, 34, mix(lash, sk[3], 0.35))
				dot(x2, 34, sk[3])
			} else {
				dot(x0-e[1], 33, mix(brow, sk[3], 0.45))
				dot(x0, 33, brow)
				dot(x1, 33, brow)
				dot(x2, 33, brow)
				dot(x0-e[1], 34, sk[3])
				dot(x0, 34, sk[2])
				dot(x1, 34, sk[2])
				dot(x2, 34, sk[3])
			}
			switch {
			case shut:
				dot(x0, 35, sk[3])
				dot(x1, 35, mix(lash, sk[2], 0.3))
				dot(x2, 35, mix(lash, sk[3], 0.45))
			case l.Female:
				dot(x0, 35, sk[3])
				dot(x1, 35, irisCol)
				dot(x2, 35, whiteCol)
			default:
				dot(x0, 35, sk[2])
				dot(x1, 35, irisCol)
				dot(x2, 35, whiteCol)
			}
			dot(x1, 36, sk[3]) // the lower lid
		}
		if l.Female {
			for _, x := range [2]int{22, 31} { // a little colour in the cheeks
				dot(x, 38, mix(sk[4], blush, 0.4))
			}
			// The nose: a glint on its bridge, a soft shadow under it.
			dot(26, 37, sk[5])
			dot(26, 38, sk[5])
			dot(27, 38, sk[3])
			dot(26, 39, sk[3])
			dot(27, 39, sk[3])
			// The lips, the lower one catching the light.
			dot(25, 41, mix(sk[3], lips, 0.45))
			dot(26, 41, lips)
			dot(27, 41, lips)
			dot(28, 41, mix(sk[3], lips, 0.45))
			dot(26, 42, mix(lips, sk[4], 0.55))
		} else {
			dot(22, 37, sk[5]) // the cheekbones, the lit one bright
			dot(31, 37, sk[4])
			// The nose: lit down its bridge, in shade down its far side, broad
			// and in shadow at the nostrils.
			dot(26, 35, sk[4])
			dot(26, 36, sk[5])
			dot(26, 37, sk[5])
			dot(26, 38, sk[5])
			dot(27, 36, sk[3])
			dot(27, 37, sk[3])
			dot(27, 38, sk[3])
			dot(25, 39, sk[3])
			dot(26, 39, sk[2])
			dot(27, 39, sk[2])
			dot(28, 39, sk[3])
		}
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
		// The eye, seen from the side: the iris at its front, the white
		// behind it, under the lid; a man's under the ridge of his brow.
		if l.Female {
			dot(31, 33, mix(brow, sk[3], 0.5))
			dot(32, 33, brow)
			dot(33, 33, brow)
			dot(31, 34, lash)
			dot(32, 34, mix(lash, sk[3], 0.35))
			dot(33, 34, mix(lash, sk[3], 0.35))
		} else {
			for x := 30; x <= 34; x++ {
				dot(x, 33, brow)
			}
			dot(30, 34, sk[3])
			dot(31, 34, sk[2])
			dot(32, 34, sk[2])
			dot(33, 34, sk[2])
			dot(34, 34, sk[3])
			dot(32, 37, sk[3]) // the hollow under the eye
		}
		if shut {
			dot(32, 35, mix(lash, sk[2], 0.3))
			dot(33, 35, mix(lash, sk[3], 0.45))
		} else {
			dot(32, 35, whiteCol)
			dot(33, 35, irisCol)
		}
		dot(33, 36, sk[3])
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
			dot(31, 39, mix(sk[4], blush, 0.45))
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

	// A man's beard, never trimmed: from the sideburns over the cheeks and
	// the jaw, a moustache hanging past the corners of the mouth, ragged
	// locks of it down to the chest. With the long braid of hair it is
	// shorter and plaited at the chin instead.
	if !l.Female {
		br := g.br
		braided := l.Hair == sim.HairPonytail
		bs := func(s shape, cx float64) shader { return beardShade(br, s, cx, 39, g.seed+9) }
		lip := mix(lips, br[2], 0.45)
		var b, moustache shape
		var locks []shape
		var mx, my, bx, by float64 // the mouth, and where a plait hangs from
		switch d {
		case sim.DirDown:
			b = sPoly(19.7, 34.8, 20.9, 36.6, 21.5, 38.3, 23, 39.1, 24.6, 39.5, 27, 39.7, 29.4, 39.5, 31, 39.1, 32.5, 38.3, 33.1, 36.6, 34.3, 34.8,
				35.1, 38.4, 35.4, 42.4, 34.8, 46.2, 33.2, 49.2, 30.6, 51.2, 27, 52, 23.4, 51.2, 20.8, 49.2, 19.2, 46.2, 18.6, 42.4, 18.9, 38.4)
			moustache = sLock(26.6, 39.2, 24.4, 39.3, 22.8, 42.6, 1.5, 0.55).or(sLock(27.4, 39.2, 29.6, 39.3, 31.2, 42.6, 1.5, 0.55))
			locks = []shape{
				sLock(21.6, 47.5, 20.6, 50.5, 21.3, 53.4, 1.2, 0.5),
				sLock(24.6, 49.5, 24.1, 52.4, 25, 55, 1.3, 0.5),
				sLock(28.2, 50.5, 28.8, 53.2, 27.9, 55.8, 1.3, 0.5),
				sLock(31.4, 49, 32.4, 51.8, 31.8, 54.2, 1.2, 0.5),
				sLock(33.8, 46, 35.1, 48.4, 34.7, 50.8, 1.1, 0.5),
			}
			mx, my, bx, by = 26.5, 41, 27, 48.6
		case sim.DirRight:
			b = sPoly(28.4, 35, 29.6, 36.8, 31, 38, 32.6, 38.8, 34, 39.4, 35.6, 39.8, 36.8, 41, 37.4, 43.2, 37, 46.6, 35.4, 49.8, 32.4, 51.4,
				29.2, 50.2, 27, 47.2, 26, 43.4, 26.4, 39)
			moustache = sLock(35.7, 39.6, 34, 39.8, 32.6, 42.6, 1.5, 0.55)
			locks = []shape{
				sLock(30.2, 49, 29.6, 52, 30.4, 54.4, 1.2, 0.5),
				sLock(33.4, 49.4, 34.4, 52, 33.8, 54.6, 1.3, 0.5),
				sLock(36, 46.4, 37.6, 48.8, 37.2, 51.4, 1.1, 0.5),
			}
			mx, my, bx, by = 34.5, 41, 33.4, 48.4
		}
		if d == sim.DirDown || d == sim.DirRight {
			if braided {
				b = b.and(sAbove(49.5)).jag(1.6, 1.9, 44, 53)
				locks = nil
			} else {
				b = b.jag(2.2, 1.9, 44, 53)
			}
			f.fill(T, b, bs(b, mx))
			for _, s := range locks {
				f.fill(T, s, bs(s, mx))
			}
			// The mouth, a dark gap in it, and the lower lip under it.
			x, y := int(mx), int(my)
			if d == sim.DirDown {
				dot(x-1, y, br[1])
				dot(x+2, y, br[1])
				dot(x+1, y, mouthDark)
				dot(x+1, y+1, lip)
			} else {
				dot(x-1, y, br[1])
				dot(x+1, y+1, lip)
			}
			dot(x, y, mouthDark)
			dot(x, y+1, lip)
			f.fill(T, moustache, func(px, py int, _ float64) color.RGBA {
				i := int(moustache.lum(px, py, 1.5)*float64(len(br))*0.8) + 1
				if hashf(px, 3, g.seed) < 0.35 {
					i--
				}
				return br.at(max(1, min(len(br)-2, i)))
			})
			if braided {
				g.tallBeardBraid(bx, by)
			}
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

// beardShade shades a beard grown long and never combed: strands hanging
// from round the mouth (cx, cy) and fanning out as they fall, each a little
// lighter or darker than the next, a fine dark line here and there between
// two, darker lower down where the light does not reach and at the ragged
// ends.
func beardShade(r ramp, a shape, cx, cy float64, seed uint32) shader {
	n := len(r)
	return func(x, y int, d float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		i := int(a.lum(x, y, 2.6)*float64(n)*0.8) + 1
		u := (fx-cx)/(1+math.Max(0, fy-cy)*0.06)/1.25 + 100
		k := math.Floor(u)
		switch h := hashf(int(k), 11, seed); {
		case h < 0.3:
			i--
		case h > 0.78:
			i++
		}
		if u-k < 0.32 && hashf(int(k), int(math.Floor(fy/3)), seed+1) < 0.5 {
			i-- // between two strands
		}
		if d > -1.1 && fy > cy+5 {
			i-- // the ends
		}
		if fy > cy+11 {
			i--
		}
		return r.at(max(1, min(n-2, i)))
	}
}

// tallBeardBraid hangs a short plait of the beard from (x, y), tied with a
// thong and a bead of bone.
func (g *figGen) tallBeardBraid(x, y float64) {
	f := &g.f
	x, y = x+math.Round(g.lagX*0.6), y+math.Round(g.lagY*0.6) // it swings a little; see limbs.go
	for k := 0; k < 2; k++ {
		off := 0.5
		if k == 1 {
			off = -0.5
		}
		s := sEll(x+off, y+0.4+2*float64(k), 1.7, 1.5)
		f.fill(g.T, s, bevel(g.br, s, 1.5, 0.05))
	}
	bead := sBox(x-1.4, y+3.6, x+1.4, y+5.8, 0.8)
	f.fill(g.T, bead, func(px, py int, _ float64) color.RGBA {
		return toneOf(ivoryRamp, bead.lum(px, py, 1.2)+0.05)
	})
	f.fill(g.T, sBox(x-0.8, y+5.8, x+0.8, y+7.2, 0.4), plain(g.br[2]))
}
