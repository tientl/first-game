package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The pilot's head (charpilot.go): the skull a lit surface with its brow
// ridge, eye sockets, cheekbones, nose and chin; the face's features set
// pixel by pixel over it; the hair of tallhead.go carried onto the bigger
// skull; and a beard that goes with the hair: short and shaped with short
// hair or a bun, full with long hair, plaited with the ponytail, a stubble
// with the bob, shaven with the spikes, wild with the mane.

// pMapT carries a point of tallhead.go's head (the skull about (27, 35), 7
// by 8.5) to the pilot's (about (27, cy), kx and ky times the size).
type pMapT struct{ cy, kx, ky float64 }

func (t pMapT) X(x float64) float64 { return 27 + (x-27)*t.kx }
func (t pMapT) Y(y float64) float64 { return t.cy + (y-35)*t.ky }

// S is shape s, drawn for tallhead.go's head, carried to the pilot's.
func (t pMapT) S(s shape) shape {
	k := math.Min(t.kx, t.ky)
	return shape{t.X(math.Max(s.x0, -4096)), t.Y(math.Max(s.y0, -4096)), t.X(math.Min(s.x1, 4096)), t.Y(math.Min(s.y1, 4096)), func(x, y float64) float64 {
		return s.d(27+(x-27)/t.kx, 35+(y-t.cy)/t.ky) * k
	}}
}

// pFace is a block of the face, one letter a pixel, its top-left at (x, y).
type pFace struct {
	x, y int
	rows []string
}

// The faces, by facing and build: front (columns 20 to 33), side. The eyes
// are bold: dark, with a glint, under a strong brow. Letters: B brow, b
// brow faint, K lash and lid, k lid faint, W white, w white in shade, H
// glint, I iris, i iris lit, P pupil, - a tone darker, = two, + a tone
// lighter, N nostril, M mouth, U lip, u lip lit, . nothing.
var (
	faceMan = pFace{20, 30, []string{
		"..............",
		".BBBB....BBBB.",
		"..-BB+..+BB-..",
		"..-HP....HP-..",
		"..wIP....PIw..",
		"...--....--...",
		"......+-......",
		"......+-......",
		".....-+--.....",
		"......NN......",
		"..............",
		".....-MM-.....",
	}}
	faceWoman = pFace{20, 30, []string{
		"..............",
		"..bBB....BBb..",
		"..............",
		".KKHP....HPKK.",
		"..wIP....PIw..",
		"..............",
		"...u......u...",
		"......+.......",
		"......-=......",
		"..............",
		"..............",
		".....UUUU.....",
		"......uu......",
	}}
	faceManSide = pFace{29, 30, []string{
		"........",
		"..BBBBB.",
		"...-BB..",
		"...HP...",
		"..wIP...",
		"...--...",
		"........",
		"........",
		"......N.",
		"........",
		"........",
		"....-M..",
	}}
	faceWomanSide = pFace{29, 30, []string{
		"........",
		"...bBB..",
		"........",
		"...HPK..",
		"..wIP...",
		"........",
		"...u....",
		"........",
		"......N.",
		"........",
		"........",
		".....U..",
		".....u..",
	}}
)

// pShut closes the eyes of a face: the lids down, a line of lashes.
func pShut(fc pFace) pFace {
	out := pFace{fc.x, fc.y, make([]string, len(fc.rows))}
	for j, r := range fc.rows {
		b := []byte(r)
		for i, ch := range b {
			switch ch {
			case 'K':
				b[i] = '-'
			case 'W', 'w', 'H', 'I', 'P', 'i':
				b[i] = '.'
				if j > 0 && strings0(fc.rows[j-1], i) == 'K' && (j+1 >= len(fc.rows) || !isEye(strings0(fc.rows[j+1], i))) {
					b[i] = 'K'
				}
			}
		}
		out.rows[j] = string(b)
	}
	// the lash line on the eye's lower row
	for j := range out.rows {
		if j+1 < len(fc.rows) {
			b := []byte(out.rows[j])
			for i := range b {
				if isEye(strings0(fc.rows[j], i)) && !isEye(strings0(fc.rows[j+1], i)) {
					b[i] = 'K'
				}
			}
			out.rows[j] = string(b)
		}
	}
	return out
}

func strings0(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return '.'
	}
	return s[i]
}

func isEye(ch byte) bool {
	return ch == 'W' || ch == 'w' || ch == 'H' || ch == 'I' || ch == 'P' || ch == 'i'
}

// pHead draws the head facing d on T (and long hair behind the body on B):
// the skull, the ears, the face, the hair and the beard.
func (g *figGen) pHead(d sim.Dir, shut bool) {
	f := &g.f
	sp := pspec()
	l := g.look
	v := viewOf(d)
	rp := g.pRamps()
	T := g.T
	rx, ry, cy := sp.headRX, sp.headRY, sp.headY
	if l.Female {
		rx -= sp.fHead
		ry -= sp.fHead * 0.6
	}
	mt := pMapT{cy, rx / 7, ry / 8.5}
	n := len(rp.skin)

	// The skull: a man's jaw square, a woman's finer; seen from the side
	// the face's profile in front of the cranium, the nose out from it.
	var skull, nose shape
	ex := 0.0 // the ears' column offset
	switch {
	case v.side:
		cr := sSuper(26.4, cy-0.4, rx+0.5, ry-0.1, 2.3)
		fy := func(y float64) float64 { return cy + (y-35.2)*(ry/9) }
		face := sPoly(30.4, fy(26.8), 33.2, fy(29.6), 34.4, fy(32.4), 33.8, fy(34), 34.4, fy(35.6), 34.6, fy(39.6), 34.2, fy(40.6), 34.6, fy(41.6),
			33.6, fy(43.4), 32.4, fy(44.2), 29.4, fy(44.4), 27.6, fy(42.6), 26.6, fy(38), 27, fy(30))
		if l.Female {
			face = sPoly(30.4, fy(26.8), 33, fy(29.6), 33.9, fy(32.4), 33.5, fy(34), 34, fy(35.6), 34.3, fy(39.6), 34, fy(40.6), 34.3, fy(41.6),
				33.3, fy(43), 31.8, fy(43.8), 29.4, fy(43.8), 27.6, fy(42.2), 26.6, fy(38), 27, fy(30))
		}
		skull = cr.or(face).sub(sEll(26.4-rx+0.6, cy+9.4, 4.4, 4.6))
		nose = sPoly(34.2, fy(35.4), 36.4, fy(38.4), 36.1, fy(39.4), 34.4, fy(39.6))
		if l.Female {
			nose = sPoly(34, fy(35.8), 35.7, fy(38.4), 35.5, fy(39.2), 34.2, fy(39.4))
		}
		nose = nose.grow(0.2)
		skull = skull.or(nose)
	default:
		skull = sSuper(27, cy, rx, ry, 2.5)
		if l.Female {
			skull = skull.and(sEll(27, cy-3, rx+0.7, ry+3))
		} else {
			skull = skull.and(sEll(27, cy-1.5, rx+1.6, ry+2)) // the jaw's corners
		}
		ex = rx + 0.1
	}
	// Ears, behind the skull's edge seen from the front or the back.
	if !v.side {
		for _, s := range [2]float64{-1, 1} {
			e := sEll(27+s*ex, cy+1.4, 1.5, 2.3)
			f.fill(T, e, func(x, y int, dd float64) color.RGBA {
				if dd < -0.9 {
					return rp.skin[max(1, n/2-1)]
				}
				return rp.skin[n/2]
			})
		}
	}
	// The surface: a dome, the brow's ridge, the sockets under it, the
	// cheekbones, the nose and the muzzle, the chin.
	mk := 1.0
	if l.Female {
		mk = 0.45
	}
	hh := func(x, y float64) float64 {
		h := edgeH(skull.d(x, y), 5.5)
		dy := (y - cy) / ry * 9 // in the A head's rows about its middle
		if v.side {
			fx := x - 27
			h += 0.9 * mk * math.Exp(-math.Pow((dy+3)/1.1, 2)) * clamp01((fx-3)/2.5)         // the brow
			h -= 0.7 * math.Exp(-math.Pow((dy+0.6)/1.2, 2)-math.Pow((fx-5.6)/1.4, 2))        // the socket
			h += 0.8 * math.Exp(-math.Pow((dy+5.2-7)/1.4, 2)-math.Pow((fx-3.2)/2.2, 2)) * mk // the cheekbone
			h += 1.6 * edgeH(nose.d(x, y), 1.2)
			return h
		}
		ax := math.Abs(x - 27)
		if v.back {
			return h
		}
		h += 0.9 * mk * math.Exp(-math.Pow((dy+3.2)/1.0, 2)) * clamp01((6.6-ax)/1.5)                // the brow's ridge
		h -= 0.8 * math.Exp(-math.Pow((dy+0.6)/1.25, 2)-math.Pow((ax-3.4)/1.7, 2))                  // the sockets
		h += 0.7 * mk * math.Exp(-math.Pow((dy-1.8)/1.3, 2)-math.Pow((ax-4.8)/1.6, 2))              // the cheekbones
		h += (0.5 + 0.9*clamp01((dy+1)/3.6)) * math.Exp(-math.Pow(ax/1.05, 2)) * b01(dy, -1.6, 3.6) // the nose
		h += 0.6 * math.Exp(-math.Pow((dy-5.4)/1.8, 2)-math.Pow(ax/3.4, 2))                         // the muzzle
		h += 0.5 * mk * math.Exp(-math.Pow((dy-7.6)/1.1, 2)-math.Pow(ax/2.4, 2))                    // the chin
		return h
	}
	cheek := 0.5
	if l.Female {
		cheek = 0.75
	}
	pWarm = func(x, y float64) float64 {
		if v.back {
			return 0
		}
		if v.side {
			return math.Max(cheek*warmNear(x, y, 31.2, cy+2, 2.8), 0.6*warmNear(x, y, 35.4, cy+2.8, 2.2))
		}
		return math.Max(cheek*math.Max(warmNear(x, y, 22.2, cy+2, 2.6), warmNear(x, y, 31.8, cy+2, 2.6)), 0.55*warmNear(x, y, 27, cy+2.8, 1.8))
	}
	f.fill(T, skull, func(x, y int, _ float64) color.RGBA {
		b := 0.04
		if !v.side && float64(y)+0.5 > cy+ry-1.6 {
			b -= 0.1 // under the jaw
		}
		return g.pSkinShade(&rp, skull, hh, x, y, b)
	})
	skinPart := f.n
	f.pClean(T, skinPart, 1)
	pWarm = nil
	if v.side { // the ear, a curl of cartilage before the cranium's back
		e := sEll(26.4, cy+1.4, 1.7, 2.4)
		f.add(T, e, func(x, y int, dd float64) color.RGBA {
			switch {
			case dd > -0.7:
				return rp.skin[max(1, n/2-1)]
			case float64(x)+0.5 < 26.4:
				return rp.skin[max(1, n/2-2)]
			}
			return rp.skin[min(n-2, n/2+1)]
		})
	}

	// The face.
	if !v.back {
		fc := faceMan
		switch {
		case v.side && l.Female:
			fc = faceWomanSide
		case v.side:
			fc = faceManSide
		case l.Female:
			fc = faceWoman
		}
		if shut {
			fc = pShut(fc)
		}
		g.pStampFace(T, fc, &rp)
	}

	// The hair, as tallhead.go draws it, carried onto this skull.
	g.pHair(d, &rp, mt, skinPart)

	if !l.Female {
		g.pBeard(d, &rp)
	}
	if l.Hair == sim.HairPonytail {
		hr := rp.hair
		switch {
		case v.back && l.Female:
			g.pBraid(T, hr, mt, 22, 42, 60)
			g.pBraid(T, hr, mt, 32, 42, 60)
		case v.back:
			g.pBraid(T, hr, mt, 27, 42, 62)
		case v.side:
		case l.Female:
			g.pBraid(T, hr, mt, 20.2, 40, 58)
			g.pBraid(T, hr, mt, 33.8, 40, 58)
		default: // over one shoulder
			g.pBraid(T, hr, mt, 33.8, 40, 58)
		}
	}
}

// b01 is 1 between a and b, easing off a pixel either side.
func b01(t, a, b float64) float64 { return clamp01(t-a+1) * clamp01(b-t+1) }

// pStampFace sets the face's pixels over the skull on l.
func (g *figGen) pStampFace(l *figLayer, fc pFace, rp *pRamps) {
	f := &g.f
	n := len(rp.skin)
	shift := func(x, y, k int) color.RGBA {
		c, _ := f.at(l, x, y)
		best, bi := math.Inf(1), n/2
		for i, s := range rp.skin {
			dr, dg, db := float64(c.R)-float64(s.R), float64(c.G)-float64(s.G), float64(c.B)-float64(s.B)
			if e := dr*dr + dg*dg + db*db; e < best {
				best, bi = e, i
			}
		}
		return rp.skin[max(0, min(n-1, bi+k))]
	}
	lash := mix(eyeDark, rp.brow, 0.2)
	white := mix(eyeWhite, rp.skin[n-1], 0.25)
	whiteS := mix(white, rp.skin[n/2], 0.45)
	iris := mix(rp.iris, eyeDark, 0.25)
	pupil := mix(eyeDark, rp.iris, 0.18)
	for j, row := range fc.rows {
		for i := 0; i < len(row); i++ {
			x, y := fc.x+i, fc.y+j
			var c color.RGBA
			switch row[i] {
			case '.':
				continue
			case 'B':
				c = rp.brow
			case 'b':
				c0, _ := f.at(l, x, y)
				c = mix(rp.brow, c0, 0.5)
			case 'K':
				c = lash
			case 'k':
				c0, _ := f.at(l, x, y)
				c = mix(lash, c0, 0.5)
			case 'W':
				c = white
			case 'w':
				c = whiteS
			case 'H':
				c = eyeShine
			case 'I':
				c = iris
			case 'i':
				c = mix(rp.iris, colWhite, 0.3)
			case 'P':
				c = pupil
			case '-':
				c = shift(x, y, -1)
			case '=':
				c = shift(x, y, -2)
			case '+':
				c = shift(x, y, 1)
			case 'N':
				c = mix(shift(x, y, -3), mouthDark, 0.3)
			case 'M':
				c0 := shift(x, y, -3)
				c = mix(c0, mouthDark, 0.55)
			case 'U':
				c = rp.lip
			case 'u':
				c = mix(rp.lip, rp.skin[n-2], 0.45)
			default:
				continue
			}
			f.dot(l, x, y, c)
		}
	}
}

// pHairShade shades hair combed from the crown (cx, cy): strands fanning
// over the skull and hanging below it, each its own tone, grooves between
// clumps, a band of sheen across them on the lit side; the light on the
// whole as the mass s bulges.
func (g *figGen) pHairShade(r ramp, s shape, round, cx, cy float64, seed uint32) shader {
	sp := pspec()
	cw := sp.hairClump // a clump's width, where the hair hangs
	return func(x, y int, _ float64) color.RGBA {
		v := s.lum(x, y, round)*0.8 + 0.1
		fx, fy := float64(x)+0.5-cx, float64(y)+0.5-cy
		// The clumps fan out from the crown and hang below it.
		w := cw * clamp01(math.Max(fy, 0)/7*0.65+0.35)
		u := fx/w + 200
		k := math.Floor(u)
		fr := u - k // across the clump
		h := hashf(int(k), 7, seed)
		v += 0.34 * (0.45 - math.Abs(fr-0.4)*1.5) // round across, lit on its left
		v += 0.12 * (h - 0.5)
		if fr > 1-math.Min(0.9/w, 0.4) { // about a pixel wide
			v -= 0.26 // the gap to the next clump
		}
		// The sheen: a band round the crown on the lit side, each clump's
		// bit of it a little higher or lower than the next's; and another
		// lower down on hair that hangs.
		rr := math.Hypot(fx*0.85, fy)
		for j, band := range [2]float64{5.2, 15} {
			e := math.Abs(rr - band - (h-0.5)*1.8)
			if e < 0.9 && fx < 3.5 && fx > -9 && (j == 0 || fy > 10) && fr < 0.8 {
				v += 0.2
			}
		}
		v += (vnoise(float64(x)*0.5, float64(y)*0.25, seed+3) - 0.5) * sp.hairGrain // the strands through the clumps
		return pTone(r, v, x, y)
	}
}

// pHair draws the hair facing d: tallhead.go's styles, carried by mt.
func (g *figGen) pHair(d sim.Dir, rp *pRamps, mt pMapT, skinPart uint16) {
	f := &g.f
	l := g.look
	T := g.T
	side := d == sim.DirRight
	hr := rp.hair
	// tallhead.go's own map, from the big head to its tall one
	m := headMap{27, 7.0 / 18, 8.5 / 16.5}
	if side {
		m = headMap{27.5, 7.4 / 18, 8.5 / 16.5}
	}
	cx, cy := mt.X(m.x(27)), mt.Y(m.y(10))
	if side {
		cx = mt.X(m.x(23))
	}
	hs := func(s shape) shader { return g.pHairShade(hr, s, 3.6, cx, cy, g.seed) }
	hair := func(s shape) {
		s = g.tallLagged(mt.S(s))
		f.fill(T, s, hs(s))
		if pspec().hairTones <= 6 {
			f.pClean(T, f.n, 1)
		}
	}
	back := func(s shape) {
		s = g.tallLagged(mt.S(s))
		f.fill(g.B, s, g.pHairShade(hr, s, 5, cx, cy, g.seed+3))
		f.pClean(g.B, f.n, 1)
	}
	lock := func(x0, y0, mx, my, x1, y1, w0, w1 float64) { hair(m.lock(x0, y0, mx, my, x1, y1, w0, w1)) }
	stubCol := mix(rp.skin[len(rp.skin)/2], hr[2], 0.42)
	stubble := func(s shape) {
		s = mt.S(s)
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
		bx = mt.X(m.x(bx))
		by := mt.Y(24.4)
		b := sEll(bx, by, 3.6*mt.kx, 3.2*mt.ky)
		f.fill(T, b, func(x, y int, _ float64) color.RGBA {
			v := b.lum(x, y, 3)*0.85 + 0.05
			a := math.Atan2(float64(y)+0.5-by, float64(x)+0.5-bx)
			rr := math.Hypot(float64(x)+0.5-bx, float64(y)+0.5-by)
			if int(math.Floor(rr*1.5+a*1.3+20))%3 == 0 {
				v -= 0.15 // strands wound round the knot
			}
			return pTone(hr, v, x, y)
		})
		t := sBox(bx-2.6*mt.kx, mt.Y(26.2), bx+2.6*mt.kx, mt.Y(27.8), 0.6)
		f.fill(T, t, func(x, y int, _ float64) color.RGBA {
			if float64(y) < mt.Y(26.2)+0.6 && float64(x) < bx {
				return mix(hairTie, colWhite, 0.4)
			}
			if float64(x) > bx {
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
			g.pBraid(T, hr, mt, 19.6, 42, 51)
		case sim.HairLong: // loose, parted in the middle, down past the shoulders
			back(sBox(18.6, 34, 35.4, 57, 3).jag(1.6, 3, 54, 43))
			opening := sFn(func(x, y float64) float64 {
				hw := 5.6 * math.Pow(clamp01((y-27.6)/6.6), 0.7)
				return math.Max(27.6-y, math.Abs(x-27.2)-hw)
			})
			curt := sBox(17.8, 32, 21.4, 55, 1.6).or(sBox(32.6, 32, 36.2, 55, 1.6)).jag(1.3, 2.4, 52, 45)
			hair(wide.and(m.above(34)).sub(opening).or(curt))
			lock(14, 22, 10.6, 36, 12, 54, 2.3, 0.7)
			lock(40, 22, 43.4, 36, 42, 54, 2.3, 0.7)
		case sim.HairShaggy: // a mane never cut or combed
			back(sBox(16.4, 33, 37.6, 55, 3.4).jag(2.6, 2.7, 42, 47))
			opening := sFn(func(x, y float64) float64 {
				u := x/2.3 + 40
				top := 30.3 + 1.5*math.Abs(u-math.Floor(u)-0.5)*hashf(int(math.Floor(u)), 2, g.seed)
				hw := 6.3 * math.Pow(clamp01((y-top)/2.4), 0.6)
				return math.Max(top-y, math.Abs(x-27.3)-hw)
			})
			mane := m.ell(27, 28.6, 22.6, 21.4).jag(1.3, 2.6, 0, 45)
			curt := sBox(16.4, 31, 21.3, 52.5, 1.8).or(sBox(32.7, 31, 37.6, 52.5, 1.8)).jag(2.4, 2.3, 42, 49)
			hair(mane.and(m.above(35)).sub(opening).or(curt))
			for _, k := range [][8]float64{
				{22.4, 26.8, 18.6, 25.4, 16.2, 28.4, 1.6, 0.5},
				{31.8, 26.6, 35.8, 25, 38, 28, 1.6, 0.5},
				{25.4, 25.6, 26.6, 22.6, 29.6, 22.8, 1.4, 0.5},
				{24, 27.6, 22.6, 30.6, 22.2, 33.6, 1.5, 0.6},
				{29.6, 27.6, 32.4, 30, 33.2, 33.2, 1.5, 0.6},
				{27, 26.8, 27.8, 29.4, 26.6, 31.6, 1.3, 0.5},
				{19, 36, 16.4, 43, 18.2, 50.5, 1.5, 0.6},
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
				{20.6, 33, 18.6, 43, 20, 51.5, 1.9, 0.6},
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
			g.pBraid(T, hr, mt, m.x(20.5), 41, 50)
		case sim.HairLong:
			hang := sPoly(m.x(8), m.y(19), m.x(31.6), m.y(19), m.x(31), m.y(40), 26, 47, 23.5, 56.5, 17.5, 56.5, 17, 36).jag(1.5, 2.6, 53, 43)
			hair(wide.and(bobCut).and(m.above(42)).or(hang))
			fringe()
			lock(30, 28, 29, 40, 25, 51, 2.3, 0.7)
		case sim.HairShaggy:
			mane := m.ell(25, 28.6, 22, 21.4)
			hang := sPoly(m.x(5), m.y(18), m.x(31.6), m.y(18), m.x(31.4), m.y(40), 26.8, 46, 25.4, 54.5, 16.4, 54.5, 15.4, 36).jag(2.6, 2.7, 38, 47)
			hair(mane.and(bobCut).and(m.above(42)).jag(1.3, 2.6, 0, 45).or(hang))
			lock(27, 15, 31, 22, 31.5, 30.5, 3.6, 0.8)
			lock(31, 14, 39, 19, 41, 28.5, 4, 0.8)
			lock(36, 13, 45, 16, 48.6, 27, 4, 0.8)
			lock(30, 28, 30, 40, 27, 50, 2.8, 0.8)
			for _, k := range [][8]float64{
				{20.6, 28.4, 17, 27, 15, 30, 1.6, 0.5},
				{19.6, 34, 15.6, 34.6, 14, 38.4, 1.6, 0.5},
				{24, 25.4, 24.6, 22.6, 27.6, 22.4, 1.4, 0.5},
				{20, 40, 17.6, 46, 19.6, 52, 1.8, 0.6},
			} {
				hair(sLock(k[0], k[1], k[2], k[3], k[4], k[5], k[6], k[7]))
			}
		case sim.HairPonytail:
			hair(E.and(swept))
			g.pBraid(T, hr, mt, m.x(11.5), 39, 58)
		case sim.HairBun:
			bun(20)
			hair(E.and(swept))
		}
	}
	hairHi := f.n

	// The hair's shadow on the brow: the skin a tone down under its edge.
	n := len(rp.skin)
	for y := 22; y < 48; y++ {
		for x := 16; x < 40; x++ {
			c, p := f.at(T, x, y)
			if p != skinPart {
				continue
			}
			for _, o := range [2][2]int{{0, -1}, {-1, -1}} {
				if _, q := f.at(T, x+o[0], y+o[1]); q >= hairLo && q <= hairHi && (q < stubLo || q > stubHi) {
					for i := 1; i < n; i++ {
						if rp.skin[i] == c {
							f.dot(T, x, y, rp.skin[i-1])
						}
					}
					break
				}
			}
		}
	}
}

// pBraid plaits hair in ramp r down from (x, y0) to y1 of tallhead.go's
// head, carried by mt, on layer l: knots lying over each other left and
// right, a tie, and the tuft below it.
func (g *figGen) pBraid(l *figLayer, r ramp, mt pMapT, x, y0, y1 float64) {
	f := &g.f
	x, y0, y1 = mt.X(x), mt.Y(y0), mt.Y(y1)
	// Each knot a strand crossing over, tilted left and right in turn: the
	// plait's chevrons.
	k := 0
	for y := y0; y < y1-4; y += 2.2 {
		dir := 1.0
		if k%2 == 1 {
			dir = -1
		}
		s := sEllAxis(x+0.55*dir, y+1.5, 0.8, 0.6*dir, 2.3, 1.25)
		if t := (y - y0) / (y1 - y0); g.lagX != 0 {
			s = s.moved(math.Round(g.lagX*1.5*t*t), 0)
		}
		f.fill(l, s, func(px, py int, _ float64) color.RGBA { return pTone(r, s.lum(px, py, 1.3)*0.85+0.12, px, py) })
		k++
	}
	if g.lagX != 0 || g.lagY != 0 {
		x += math.Round(g.lagX * 1.5)
		y1 += math.Round(g.lagY)
	}
	t := sBox(x-1.7, y1-4, x+1.7, y1-2, 0.6)
	f.fill(l, t, func(px, py int, _ float64) color.RGBA {
		if py == int(y1-4) && px <= int(x) {
			return mix(hairTie, colWhite, 0.45)
		}
		if px >= int(x+1) {
			return mix(hairTie, colInk, 0.3)
		}
		return hairTie
	})
	tuft := sLock(x, y1-2.2, x+0.4, y1-0.8, x-0.3, y1+0.8, 1.4, 0.5)
	f.fill(l, tuft, func(px, py int, _ float64) color.RGBA { return pTone(r, tuft.lum(px, py, 1.5)*0.8, px, py) })
}

// pBeardKind is the beard a man wears with each hair style.
type pBeardKind uint8

const (
	beardShort   pBeardKind = iota // along the jaw and round the chin, trimmed, with a moustache
	beardFull                      // fuller and longer, shaped to a point on the chest
	beardPlait                     // short, a plait hanging from the chin
	beardStubble                   // a few days' growth
	beardClean                     // shaven
	beardWild                      // never cut: down to the chest in ragged locks
	beardChin                      // a moustache and a beard on the chin alone
)

func beardOf(h sim.HairStyle) pBeardKind {
	switch h {
	case sim.HairSpiky:
		return beardClean
	case sim.HairBob:
		return beardStubble
	case sim.HairLong:
		return beardFull
	case sim.HairPonytail:
		return beardPlait
	case sim.HairBun:
		return beardChin
	case sim.HairShaggy:
		return beardWild
	}
	return beardShort
}

// pBeard draws a man's beard facing d, as his hair has it.
func (g *figGen) pBeard(d sim.Dir, rp *pRamps) {
	if d == sim.DirUp {
		return
	}
	f := &g.f
	sp := pspec()
	T := g.T
	kind := beardOf(g.look.Hair)
	side := d == sim.DirRight
	rx := sp.headRX
	cy := sp.headY
	ky := sp.headRY / 9
	Y := func(y float64) float64 { return cy + (y-35.2)*ky } // rows of the A head
	X := func(x float64) float64 { return 27 + (x-27)*rx/7.4 }
	br := rp.beard
	shadeB := func(s shape, mx, my float64, seed uint32) shader {
		return func(x, y int, dd float64) color.RGBA {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			v := s.lum(x, y, 2.4)*0.75 + 0.12
			u := (fx-mx)/(1+math.Max(0, fy-my)*0.08)/1.3 + 100
			k := math.Floor(u)
			h := hashf(int(k), 11, seed)
			v += (h - 0.5) * 0.16
			if u-k < 0.34 && hashf(int(k), int(math.Floor(fy/3)), seed+1) < 0.5 {
				v -= 0.14 // between two strands
			}
			if fy > my+6 {
				v -= 0.06 * (fy - my - 6) / 4
			}
			if dd > -1.1 && fy > my+3 {
				v -= 0.1 // the ends
			}
			v += (hashf(x, y, seed+5) - 0.5) * sp.hairGrain * 0.7 // fine hairs
			return pTone(br, v, x, y)
		}
	}
	// The mouth's middle, and the chin's bottom.
	mx, my := 27.0, Y(41)
	if side {
		mx = X(34) - 0.4
	}
	switch kind {
	case beardClean:
		return
	case beardStubble:
		var s shape
		if side {
			s = sPoly(X(27.6), Y(37), X(29), Y(37), X(31.4), Y(40.2), X(33.6), Y(40.4), X(34.4), Y(42.2), X(33.4), Y(44), X(29.6), Y(44.6), X(27.6), Y(42.4))
		} else {
			s = sPoly(X(20.2), Y(37), X(21.4), Y(37), X(22.6), Y(39.6), X(24.4), Y(40.6), X(29.6), Y(40.6), X(31.4), Y(39.6), X(32.6), Y(37), X(33.8), Y(37),
				X(33.6), Y(41), X(31.6), Y(43.6), X(27), Y(44.6), X(22.4), Y(43.6), X(20.4), Y(41))
		}
		f.fill(T, s, func(x, y int, _ float64) color.RGBA {
			c, _ := f.at(T, x, y)
			k := 0.3
			if (x*3+y*5)%4 == 0 {
				k = 0.45
			}
			return mix(c, br[1], k)
		})
		return
	}
	var b, moustache shape
	var locks []shape
	low := Y(47.2) // the beard's lowest row in the middle
	switch kind {
	case beardFull:
		low = Y(51.5)
	case beardWild:
		low = Y(54)
	case beardChin:
		low = Y(46.4)
	}
	if side {
		top := Y(35.2)
		if kind == beardChin {
			top = Y(40)
		}
		b = sPoly(X(28.2), top, X(29.4), top, X(30.6), Y(38.6), X(32.4), Y(40), X(34.6), Y(39.8), X(35), Y(41.4), X(34.8), Y(43.6),
			X(34.2), low-1.4, X(32.6), low, X(30.4), low-1.2, X(28.4), Y(44.6), X(27.2), Y(41.4), X(27.4), Y(37.6))
		moustache = sLock(X(34.8), Y(39.4), X(33.4), Y(39.8), X(32.2), Y(41.6), 1.3, 0.55)
	} else {
		top := Y(35)
		if kind == beardChin {
			b = sPoly(X(23.4), Y(41.6), X(30.6), Y(41.6), X(31), Y(43.4), X(29.4), low-0.8, X(27), low, X(24.6), low-0.8, X(23), Y(43.4))
		} else {
			wb := 0.0 // how far the beard swells below the jaw
			if kind == beardFull || kind == beardWild {
				wb = 1
			}
			b = sPoly(X(20.1), top, X(21.3), top, X(22), Y(38), X(23.6), Y(40.6), X(30.4), Y(40.6), X(32), Y(38), X(32.7), top, X(33.9), top,
				X(34.1)+wb*0.4, Y(39.6), X(33.4)+wb, Y(43.2), X(31.6)+wb*0.8, low-2.6, X(29.4), low-0.8, X(27), low, X(24.6), low-0.8, X(22.4)-wb*0.8, low-2.6, X(20.6)-wb, Y(43.2), X(19.9)-wb*0.4, Y(39.6))
		}
		moustache = sLock(X(26.4), Y(39.5), X(24.6), Y(39.6), X(23.2), Y(41.9), 1.4, 0.55).or(sLock(X(27.6), Y(39.5), X(29.4), Y(39.6), X(30.8), Y(41.9), 1.4, 0.55)).or(sCap(X(25), Y(39.5), X(29), Y(39.5), 1.0, 1.0))
	}
	switch kind {
	case beardWild:
		b = b.jag(2.2, 1.9, Y(44), 53)
		if side {
			locks = []shape{sLock(X(30.2), Y(49), X(29.6), Y(52), X(30.4), Y(55), 1.2, 0.5), sLock(X(33.4), Y(49.4), X(34.4), Y(52), X(33.8), Y(55), 1.3, 0.5)}
		} else {
			locks = []shape{
				sLock(X(21.8), Y(48), X(20.8), Y(51), X(21.4), Y(54), 1.2, 0.5),
				sLock(X(24.6), Y(50), X(24.1), Y(53), X(25), Y(56), 1.3, 0.5),
				sLock(X(29.4), Y(50), X(29.9), Y(53), X(29), Y(56), 1.3, 0.5),
				sLock(X(32.2), Y(48), X(33.2), Y(51), X(32.6), Y(54), 1.2, 0.5),
			}
		}
	case beardFull:
		b = b.jag(1.4, 2, Y(45), 53)
	default:
		b = b.jag(1, 1.8, Y(44.5), 53)
	}
	f.fill(T, b, shadeB(b, mx, my-2, g.seed+9))
	// The moustache, of a piece with the beard: no line between them.
	f.add(T, moustache, func(px, py int, _ float64) color.RGBA {
		v := moustache.lum(px, py, 1.4)*0.8 + 0.2
		if hashf(px, 3, g.seed) < 0.35 {
			v -= 0.1
		}
		return pTone(br, v, px, py)
	})
	f.pClean(T, f.n, 1)
	// The mouth, shut: a short dark line under the moustache.
	x, y := int(math.Floor(mx)), int(math.Floor(my))
	shut := mix(mouthDark, colInk, 0.45)
	if side {
		f.dot(T, x, y, shut)
	} else {
		f.dot(T, x-1, y, shut)
		f.dot(T, x, y, shut)
	}
	for _, s := range locks {
		f.fill(T, s, shadeB(s, mx, my-2, g.seed+11))
	}
	if kind == beardPlait {
		bx, by := mx, low-1
		if side {
			bx = X(33)
		}
		bx, by = bx+math.Round(g.lagX*0.6), by+math.Round(g.lagY*0.6)
		for k := 0; k < 2; k++ {
			off := 0.5
			if k == 1 {
				off = -0.5
			}
			s := sEll(bx+off, by+0.4+2*float64(k), 1.7, 1.5)
			f.fill(T, s, func(px, py int, _ float64) color.RGBA { return pTone(br, s.lum(px, py, 1.5)*0.85+0.1, px, py) })
		}
		bead := sBox(bx-1.4, by+3.6, bx+1.4, by+5.8, 0.8)
		f.fill(T, bead, func(px, py int, _ float64) color.RGBA {
			return toneOf(ivoryRamp, bead.lum(px, py, 1.2)+0.05)
		})
		f.fill(T, sBox(bx-0.8, by+5.8, bx+0.8, by+7.2, 0.4), plain(br[2]))
	}
}
