package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD head: a face with real features, hair in locks, a braided beard.
// The skull covers image columns 9 to 44 and rows 15 to 47 (look.go's rows
// 5 to 15), the eyes are on rows 30 to 38.

// Eyes, one letter a pixel: K lid and lashes, W white, w white in the lid's
// shade, H glint, P pupil, D iris in shade, I iris, i iris lit from below,
// l the crease under the eye.
var (
	eyeFrontL = []string{
		".KKKKK.",
		"KKKKKKK",
		"KHHPPDw",
		"WHHPPIW",
		"WIPPPIW",
		"WIPPPIW",
		"WiIPIiW",
		".iiiii.",
		"..lll..",
	}
	eyeFrontR = []string{
		".KKKKK.",
		"KKKKKKK",
		"wHHPPDK",
		"WHHPPIW",
		"WIPPPIW",
		"WIPPPIW",
		"WiIPIiW",
		".iiiii.",
		"..lll..",
	}
	eyeSide = []string{
		".KKKK",
		"KKKKK",
		"wHPPD",
		"WHPPI",
		"WIPPI",
		"WIPPI",
		"WiIPi",
		".iiii",
		"..ll.",
	}
	eyeShut = []string{
		"K.....K",
		".KKKKK.",
	}
	browMan = []string{
		".KKKKKK..",
		"KKKKKKKKK",
		"......KKK",
	}
	browWoman = []string{
		"..KKKK...",
		".K....KK.",
	}
)

func (g *figGen) eyePal() func(byte) (color.RGBA, bool) {
	iris := irisColors[g.look.Skin]
	sk := g.sk
	return func(ch byte) (color.RGBA, bool) {
		switch ch {
		case 'K':
			return eyeDark, true
		case 'W':
			return eyeWhite, true
		case 'w':
			return eyeShade, true
		case 'H':
			return eyeShine, true
		case 'P':
			return mix(eyeDark, iris, 0.12), true
		case 'D':
			return mix(iris, eyeDark, 0.5), true
		case 'I':
			return iris, true
		case 'i':
			return mix(iris, colWhite, 0.42), true
		case 'l':
			return sk[3], true
		}
		return color.RGBA{}, false
	}
}

// braid plaits hair in ramp r down from (x, y0) to y1 on layer l: knots
// lying over each other left and right, a tie, and the tuft below it.
func (g *figGen) braid(l *figLayer, r ramp, x, y0, y1 float64, tie color.RGBA) {
	f := &g.f
	k := 0
	for y := y0; y < y1-6; y += 3 {
		off := 1.0
		if k%2 == 1 {
			off = -1
		}
		s := sEll(x+off, y+2.3, 2.7, 2.7)
		if t := (y - y0) / (y1 - y0); g.lagX != 0 {
			s = s.moved(math.Round(g.lagX*1.5*t*t), 0) // the braid trails; see limbs.go
		}
		f.fill(l, s, bevel(r, s, 2.6, 0.06))
		k++
	}
	if g.lagX != 0 || g.lagY != 0 {
		x += math.Round(g.lagX * 1.5)
		y1 += math.Round(g.lagY)
	}
	t := sBox(x-2.4, y1-6, x+2.4, y1-3.2, 0.9)
	f.fill(l, t, func(px, py int, _ float64) color.RGBA {
		if py == int(y1-6) && px <= int(x) {
			return mix(tie, colWhite, 0.45)
		}
		if px >= int(x+1) {
			return mix(tie, colInk, 0.3)
		}
		return tie
	})
	tuft := sLock(x, y1-3.6, x+0.6, y1-1.5, x-0.4, y1+0.6, 2, 0.6)
	f.fill(l, tuft, bevel(r, tuft, 2, -0.05))
}

func (g *figGen) head(d sim.Dir, shut bool) {
	f := &g.f
	l := g.look
	if l.Hair == sim.HairShaggy {
		l.Hair = sim.HairLong // the stiff figure has no mane of its own
	}
	sk, hr := g.sk, g.hr
	side, up := d == sim.DirRight, d == sim.DirUp
	T := g.T
	pal := g.eyePal()
	blush := skinTones[l.Skin].blush
	lips := mix(blush, mouthDark, 0.5)
	brow := hr[2]
	if luma(brow) < 45 {
		brow = mix(hr[3], sk[2], 0.3) // black brows would run into the eyes
	}

	skull := sSuper(27, 31, 18, 16.5, 2.6)
	// Ears, behind the skull's edge when seen from the front or the back.
	if !side {
		for _, ex := range [2]float64{9.3, 44.7} {
			e := sEll(ex, 34, 2.6, 3.6)
			f.fill(T, e, func(x, y int, dd float64) color.RGBA {
				if dd < -1.2 {
					return sk[2]
				}
				return sk[3]
			})
		}
	}
	f.fill(T, skull, func(x, y int, dd float64) color.RGBA {
		if side { // in profile the whole face is in the light
			if y >= 44 && dd > -3 {
				return sk[3]
			}
			return sk[4]
		}
		switch v := skull.lum(x, y, 8); {
		case v < 0.3:
			return sk[2]
		case v < 0.5 || y >= 45:
			return sk[3]
		}
		return sk[4]
	})
	skinPart := f.n

	dot := func(x, y int, c color.RGBA) { f.dot(T, x, y, c) }
	switch d {
	case sim.DirDown:
		if shut {
			f.stamp(T, 14, 34, eyeShut, false, pal)
			f.stamp(T, 33, 34, eyeShut, false, pal)
		} else {
			f.stamp(T, 14, 30, eyeFrontL, false, pal)
			f.stamp(T, 33, 30, eyeFrontR, false, pal)
		}
		bp := func(c color.RGBA) func(byte) (color.RGBA, bool) {
			return func(ch byte) (color.RGBA, bool) { return c, ch == 'K' }
		}
		if l.Female {
			f.stamp(T, 13, 26, browWoman, false, bp(mix(brow, sk[3], 0.25)))
			f.stamp(T, 32, 26, browWoman, true, bp(mix(brow, sk[3], 0.25)))
			if !shut { // lashes
				dot(13, 30, eyeDark)
				dot(12, 29, eyeDark)
				dot(40, 30, eyeDark)
				dot(41, 29, eyeDark)
			}
			for _, bx := range [2]int{13, 35} { // rosy cheeks
				for y := 39; y <= 41; y++ {
					for x := bx + (y+1)%2; x < bx+6; x++ {
						c := blush
						if (x+y)%2 == 1 {
							c = mix(sk[4], blush, 0.55)
						}
						dot(x, y, c)
					}
				}
			}
			for x := 25; x <= 28; x++ { // the lips
				dot(x, 42, lips)
			}
			dot(24, 41, mix(sk[3], lips, 0.5))
			dot(29, 41, mix(sk[3], lips, 0.5))
			dot(26, 43, mix(lips, blush, 0.6))
			dot(27, 43, mix(lips, colWhite, 0.45))
		} else {
			f.stamp(T, 13, 26, browMan, false, bp(brow))
			f.stamp(T, 32, 26, browMan, true, bp(brow))
			for _, p := range [][2]int{{12, 40}, {13, 39}, {14, 40}, {15, 39}, {38, 40}, {39, 39}, {40, 40}, {41, 39}} {
				dot(p[0], p[1], mix(sk[4], blush, 0.8)) // weathered cheeks
			}
		}
		// The nose: a glint on its tip, a shadow under it.
		dot(26, 37, sk[5])
		dot(27, 37, sk[5])
		dot(28, 37, sk[3])
		dot(28, 38, sk[3])
		dot(26, 39, sk[3])
		dot(27, 39, sk[2])
		dot(28, 39, sk[2])
	case sim.DirRight:
		nose := sEll(45, 37.2, 2, 2.4)
		if l.Female {
			nose = sEll(44.6, 37.4, 1.7, 2)
		}
		f.add(T, nose, func(x, y int, _ float64) color.RGBA {
			if y >= 38 {
				return sk[3]
			}
			return sk[4]
		})
		dot(45, 36, sk[5])
		dot(44, 39, sk[2])
		dot(45, 39, sk[2])
		f.stamp(T, 34, 30, eyeSide, false, pal)
		// The ear.
		ear := sEll(23.5, 35, 3, 3.7)
		f.add(T, ear, func(x, y int, dd float64) color.RGBA {
			switch {
			case dd > -1:
				return sk[3]
			case x <= 23:
				return sk[2]
			}
			return sk[4]
		})
		dot(24, 36, sk[2])
		if l.Female {
			for x := 34; x <= 38; x++ {
				dot(x, 26+(38-x)/3, mix(brow, sk[3], 0.25))
			}
			dot(39, 30, eyeDark)
			dot(40, 29, eyeDark)
			for y := 39; y <= 41; y++ {
				for x := 32 + (y+1)%2; x < 38; x++ {
					c := blush
					if (x+y)%2 == 1 {
						c = mix(sk[4], blush, 0.55)
					}
					dot(x, y, c)
				}
			}
			dot(40, 42, lips)
			dot(41, 42, lips)
			dot(41, 43, mix(lips, blush, 0.6))
		} else {
			for x := 33; x <= 40; x++ {
				dot(x, 27, brow)
				if x >= 35 {
					dot(x, 26, brow)
				}
				if x >= 38 {
					dot(x, 28, brow)
				}
			}
			dot(33, 40, mix(sk[4], blush, 0.8))
			dot(34, 39, mix(sk[4], blush, 0.8))
			dot(35, 40, mix(sk[4], blush, 0.8))
		}
	}

	// The hair.
	cx, cy := 27.0, 10.0
	if side {
		cx = 23
	}
	hs := func(s shape) shader { return hairShade(hr, s, 7, cx, cy, g.seed) }
	hair := func(s shape) { s = g.lagged(s); f.fill(T, s, hs(s)) }
	lock := func(x0, y0, mx, my, x1, y1, w0, w1 float64) {
		hair(sLock(x0, y0, mx, my, x1, y1, w0, w1))
	}
	stubCol := mix(sk[3], hr[2], 0.42)
	stubble := func(s shape) {
		f.fill(T, s, func(x, y int, dd float64) color.RGBA {
			if (x+2*y)%5 == 0 {
				return mix(stubCol, hr[1], 0.25)
			}
			if dd > -2 && s.lum(x, y, 4) < 0.5 {
				return mix(stubCol, hr[1], 0.2)
			}
			return stubCol
		})
	}
	bun := func(bx float64) {
		b := sEll(bx, 6.8, 7.8, 6.2)
		f.fill(T, b, func(x, y int, _ float64) color.RGBA {
			i := int(b.lum(x, y, 6)*6*0.74) + 1
			// strands wound round the knot
			a := math.Atan2(float64(y)+0.5-6.8, float64(x)+0.5-bx)
			rr := math.Hypot(float64(x)+0.5-bx, float64(y)+0.5-6.8)
			if int(math.Floor(rr*0.9+a*1.2+20))%3 == 0 {
				i--
			}
			return hr.at(max(1, i))
		})
		t := sBox(bx-5, 10.6, bx+5, 13.2, 1)
		f.fill(T, t, func(x, y int, _ float64) color.RGBA {
			if y == 10 || y == 11 && x < int(bx) {
				return mix(hairTie, colWhite, 0.4)
			}
			if x > int(bx)+2 {
				return mix(hairTie, colInk, 0.3)
			}
			return hairTie
		})
	}
	hairLo := f.n + 1
	stubLo, stubHi := uint16(0), uint16(0)
	switch d {
	case sim.DirDown:
		E := sEll(27, 28.5, 19.3, 19.5)
		wide := sEll(27, 29, 21, 20.5)
		switch l.Hair {
		case sim.HairShort: // swept over the top, the sides cropped
			hair(E.and(sOver(func(x float64) float64 { return 20.5 + 0.02*(x-27)*(x-27) })).or(E.and(sAbove(31)).sub(sBox(11.8, 0, 42.2, 60, 0))))
			lock(38, 14, 41, 19, 40.5, 25.5, 3.4, 0.7)
			lock(34, 12, 22, 13, 12.5, 25, 4.4, 0.7)
			lock(36, 13, 27, 16, 19.5, 26.5, 4.2, 0.7)
			lock(37, 13.5, 32, 19, 27, 26, 3.8, 0.7)
			lock(38, 14, 36, 20, 33.5, 25, 3.4, 0.7)
		case sim.HairSpiky: // a crest down the middle, the sides shaved
			stubLo = f.n + 1
			stubble(sEll(27, 30, 18.6, 17.4).and(sOver(func(x float64) float64 { return 21.5 + 0.02*(x-27)*(x-27) })))
			stubHi = f.n
			hair(sEll(27, 19.5, 7.6, 5.4))
			lock(22, 21, 18, 12, 20.5, 5, 4.4, 0.7)
			lock(32, 21, 36, 12, 33.5, 6, 4.4, 0.7)
			lock(27, 22, 25.5, 10, 27.5, 1.5, 5.4, 0.7)
		case sim.HairBob: // chin length with a straight fringe, a thin braid at one side
			hair(wide.and(sAbove(46.5)).sub(sBox(12.6, 23, 41.4, 60, 3)).jag(1.6, 3.4, 43, 41))
			for i := 0; i < 7; i++ {
				x := 13.5 + float64(i)*4.5
				lock(x+0.6, 17, x+0.2, 22, x-0.4+hashf(i, 4, g.seed), 27+hashf(i, 5, g.seed), 2.7, 0.8)
			}
			g.braid(T, hr, 8.5, 44, 60, hairTie)
		case sim.HairLong: // loose, parted in the middle, down past the shoulders
			back := g.lagged(sBox(8, 30, 46, 66, 4).jag(2.4, 4.4, 61, 43))
			f.fill(g.B, back, hairShade(hr, back, 9, cx, cy, g.seed+3))
			opening := sFn(func(x, y float64) float64 {
				hw := 14.6 * math.Pow(clamp01((y-16.5)/13), 0.7)
				return math.Max(16.5-y, math.Abs(x-27.6)-hw)
			})
			curt := sBox(5.6, 26, 12.8, 61, 3).or(sBox(41.2, 26, 48.4, 61, 3)).jag(2, 3.6, 56, 45)
			hair(wide.and(sAbove(34)).sub(opening).or(curt))
			lock(14, 22, 10.6, 36, 12, 54, 2.3, 0.7)
			lock(40, 22, 43.4, 36, 42, 54, 2.3, 0.7)
		case sim.HairPonytail, sim.HairBun: // pulled back off the brow
			if l.Hair == sim.HairBun {
				bun(27)
			}
			hair(sEll(27, 28.5, 19, 18.4).and(sOver(func(x float64) float64 { return 22.5 + 0.028*(x-27)*(x-27) - 1.5*math.Exp(-(x-27)*(x-27)/14) })))
			lock(13.5, 22, 10.6, 26, 12.2, 32.5, 1.7, 0.6)
			if l.Hair == sim.HairBun {
				lock(40.5, 22, 43.4, 26, 41.8, 32.5, 1.7, 0.6)
			}
		}
	case sim.DirUp:
		E := sEll(27, 28.5, 19.3, 19.5)
		wide := sEll(27, 29, 21, 20.5)
		switch l.Hair {
		case sim.HairShort:
			hair(E.and(sAbove(43)).jag(2, 3.6, 38.5, 41))
		case sim.HairSpiky:
			stubLo = f.n + 1
			stubble(sEll(27, 30, 18.6, 17.4).and(sAbove(43.5)))
			stubHi = f.n
			hair(sPath(6.4, 3.4, 27, 10, 27, 37))
			lock(23, 14, 19, 9, 21, 4, 4, 0.7)
			lock(31, 14, 35, 9, 33.5, 5, 4, 0.7)
			lock(27, 14, 26, 6, 27.5, 1.5, 5, 0.7)
			lock(25, 22, 21, 19, 21.5, 14, 3.2, 0.7)
			lock(29, 24, 33, 21, 32.5, 16, 3.2, 0.7)
			lock(26, 31, 22.5, 28, 22.5, 23.5, 2.8, 0.7)
		case sim.HairBob:
			hair(wide.and(sAbove(46.5)).jag(1.8, 3.4, 42.5, 41))
		case sim.HairLong:
			hair(wide.and(sAbove(42)).or(sBox(6, 18, 48, 67, 9)).jag(2.6, 4.4, 61, 43))
		case sim.HairPonytail, sim.HairBun:
			if l.Hair == sim.HairBun {
				bun(27)
			}
			hair(E.and(sAbove(44.5)).jag(1.2, 4, 41, 41))
		}
	case sim.DirRight:
		E := sEll(25.5, 28.5, 19.8, 19.5)
		wide := sEll(25, 29, 20.6, 20.5)
		short := sPoly(-9, -9, 60, -9, 60, 20, 47, 24.5, 36, 25.5, 30, 29.5, 25, 30.5, 20.5, 34, 19.5, 40, 15, 44.6, -9, 44.6)
		swept := sPoly(-9, -9, 60, -9, 60, 16, 45, 21, 38, 23, 31, 28, 25, 30.5, 20.5, 34, 19.5, 40, 15, 44.6, -9, 44.6)
		bobCut := sPoly(-9, -9, 60, -9, 60, 21, 47, 25.5, 37, 26, 33, 30, 31.5, 40, 30, 46.5, -9, 46.5)
		fringe := func() {
			lock(27, 15, 31, 22, 31.5, 29.5, 3.4, 0.7)
			lock(31, 14, 38, 19, 40, 27, 3.8, 0.7)
			lock(36, 13, 44, 16, 47.5, 25.5, 3.8, 0.7)
		}
		switch l.Hair {
		case sim.HairShort:
			hair(E.and(short).jag(1.6, 3.6, 39, 41))
			fringe()
		case sim.HairSpiky:
			stubLo = f.n + 1
			stubble(sEll(26.5, 30, 18.8, 17.4).and(short))
			stubHi = f.n
			hair(sBox(10, 10.5, 45, 18.5, 4))
			lock(16, 16, 13, 10, 13.5, 3.5, 4.2, 0.7)
			lock(24, 15, 22, 8, 23, 1.5, 4.4, 0.7)
			lock(32, 15, 31, 8, 32.5, 1.5, 4.4, 0.7)
			lock(40, 16, 41, 10, 42.5, 4.5, 4, 0.7)
		case sim.HairBob:
			hair(wide.and(bobCut).jag(1.6, 3.4, 43, 41))
			fringe()
			g.braid(T, hr, 20.5, 42, 57, hairTie)
		case sim.HairLong:
			hang := sPoly(8, 19, 31.6, 19, 31, 40, 27, 52, 20, 63.5, 6.5, 63.5, 4.6, 30).jag(2.4, 4, 58, 43)
			hair(wide.and(bobCut).and(sAbove(42)).or(hang))
			fringe()
			lock(30, 28, 29, 40, 25, 51, 2.3, 0.7)
		case sim.HairPonytail:
			hair(E.and(swept))
			g.braid(T, hr, 11.5, 38, 66, hairTie)
		case sim.HairBun:
			bun(20)
			hair(E.and(swept))
		}
	}
	hairHi := f.n

	// The hair's shadow on the brow.
	for y := 15; y < 48; y++ {
		for x := 8; x < 47; x++ {
			c, p := f.at(T, x, y)
			if p != skinPart {
				continue
			}
			for _, o := range [5][2]int{{0, -1}, {0, -2}, {-1, -1}, {-1, -2}, {-2, -3}} {
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
		bs := func(s shape) shader { return hairShade(br, s, 5, 27, 30, g.seed+9) }
		switch d {
		case sim.DirDown:
			b := sPoly(9, 34, 12, 35, 13.4, 40.5, 19, 42.6, 27, 43.2, 35, 42.6, 40.6, 40.5, 42, 35, 45, 34,
				45, 41, 41, 47.5, 35, 52, 30, 54, 24, 54, 19, 52, 13, 47.5, 9, 41).jag(1.8, 3.4, 46, 51)
			f.fill(T, b, bs(b))
			dot(26, 43, mouthDark)
			dot(27, 43, mouthDark)
			dot(26, 44, mix(mouthDark, blush, 0.5))
			dot(27, 44, mix(mouthDark, blush, 0.5))
			for _, m := range []shape{
				sLock(26.6, 40, 21, 38.6, 15, 43.4, 2.7, 0.8),
				sLock(27.4, 40, 33, 38.6, 39, 43.4, 2.7, 0.8),
			} {
				f.fill(T, m, bevel(br, m, 2.4, -0.04))
			}
			g.beardBraid(27, 52)
		case sim.DirRight:
			b := sPoly(23.5, 36.5, 27, 37.5, 29, 42, 36, 43, 41, 42.4, 46.5, 43, 47.6, 46, 46.5, 50, 43, 53.5, 38, 54.5,
				33, 52.5, 28, 47.5, 24.5, 42).jag(1.8, 3.4, 47, 51)
			f.fill(T, b, bs(b))
			m := sLock(46.2, 40.4, 41, 39.6, 36.5, 43.6, 2.5, 0.8)
			f.fill(T, m, bevel(br, m, 2.4, -0.04))
			g.beardBraid(38.5, 53)
		}
	}
	switch l.Hair {
	case sim.HairPonytail:
		switch {
		case up && l.Female:
			g.braid(T, hr, 14.5, 42, 66, hairTie)
			g.braid(T, hr, 39.5, 42, 66, hairTie)
		case up:
			g.braid(T, hr, 27, 41, 69, hairTie)
		case side:
		case l.Female:
			g.braid(T, hr, 14.5, 38, 63, hairTie)
			g.braid(T, hr, 39.5, 38, 63, hairTie)
		default: // over one shoulder
			g.braid(T, hr, 41.5, 38, 63, hairTie)
		}
	}
}

// beardBraid hangs the beard's short plait from (x, y), with its gold bead.
func (g *figGen) beardBraid(x, y float64) {
	f := &g.f
	x, y = x+math.Round(g.lagX*0.6), y+math.Round(g.lagY*0.6) // it swings a little; see limbs.go
	k := sEll(x, y+0.6, 2.5, 2.2)
	f.fill(g.T, k, bevel(g.br, k, 2.2, 0.05))
	bead := sBox(x-2.2, y+2, x+2.2, y+5, 1)
	f.fill(g.T, bead, func(px, py int, _ float64) color.RGBA {
		return toneOf(goldHD, bead.lum(px, py, 1.6)+0.1)
	})
	f.dot(g.T, int(x-1.5), int(y+2), goldHD[5])
	f.fill(g.T, sBox(x-1, y+5, x+1, y+6.4, 0.5), plain(g.br[2]))
}
