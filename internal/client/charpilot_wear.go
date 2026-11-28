package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The figure's clothes (charpilot.go), painted as its body is: each a
// surface over the body under it, lit from the top left and cut into an
// OKLab ramp, with its folds as grooves in it. The tunic and the sleeves
// are in greys, for the player's colour (upperTinted); the trousers, the
// wraps, the boots and the mantle in their own colours. The shapes are the
// old ones where other art follows them (tallTorso: the armour; the
// shoulders and the hems), so everything worn over them still fits.

var (
	// pCloth is the greys the tunic and the sleeves are painted in, to be
	// multiplied by the player's colour: a touch of violet in the folds.
	pCloth = func() ramp {
		r := make(ramp, 7)
		for k := range r {
			t := float64(k) / float64(len(r)-1)
			r[k] = okColor(lerpf(0.36, 0.97, t), 0.014*(1-t)*(1-t), 4.9)
		}
		return r
	}()
	pTrouser = hueRamp(mkRamp(0x221812, 0x2c2019, 0x443227, 0x5c4636, 0x765c47, 0x8e735a), 7, 0.25, 0.62, -0.16, 0.1, 1.1, 0.9)
	pWrap    = hueRamp(mkRamp(0x5e4e36, 0x7d6a4c, 0xa8936c, 0xcdb98f, 0xe6d8b4), 7, 0.45, 0.9, -0.2, 0.08, 1.15, 0.85)
	pBootR   = hueRamp(bootHD, 7, 0.2, 0.62, -0.18, 0.12, 1.15, 0.9)
	pMantleR = hueRamp(furHD, 9, 0.3, 0.84, -0.12, 0.08, 1.1, 0.8)
	pCuffR   = hueRamp(mkRamp(0x3a2a20, 0x5c4532, 0x86694c, 0xae9070, 0xd2b893), 7, 0.4, 0.82, -0.2, 0.12, 1.1, 0.8)
)

// pClothTone is the tone of cloth ramp r for light v.
func pClothTone(r ramp, v float64) color.RGBA {
	return r.at(int(math.Floor(v * float64(len(r)))))
}

// pTallBody is tallBody for the figure: bare (pBody), or in a tunic or a
// dress; and the mantle over it.
func (g *figGen) pTallBody(d sim.Dir, sk skirt) {
	if g.wear&sim.WearTunic == 0 {
		g.pBody(d, sk)
		if g.hook != nil {
			g.hook()
		}
	} else {
		g.pTunic(d, sk)
	}
	if g.wear&sim.WearMantle != 0 {
		g.pMantle(d)
	}
}

// pArmFig is arm for the figure: arm i bare with its hand on hd, or its
// sleeve on sl and the hand out of it on hd.
func (g *figGen) pArmFig(sl, hd *figLayer, d sim.Dir, i int, a armGeom) {
	bias := 0.12
	switch {
	case d == sim.DirRight && i == 1:
		bias = -0.2
	case d == sim.DirRight:
		bias = 0.14
	case i == 0: // on the right of the screen: in the body's shade
		bias = -0.16
	}
	if g.wear&sim.WearTunic == 0 {
		g.pArm(hd, a, bias*0.5, d, i)
		return
	}
	g.pSleeve(sl, a, bias*0.6, d, i)
	rp := g.pRamps()
	pWarm = func(x, y float64) float64 { return 0.5 * warmNear(x, y, a.hx, a.hy, 5.5) }
	g.pHand(hd, a, viewOf(d), &rp, bias*0.5, true)
	pWarm = nil
}

// pLoose is a limb's reach p scaled by k with cloth over it: its swells
// eased towards its mean, add further out.
func pLoose(p prof, k, ease, add float64) prof {
	q := p.scaled(k)
	m, n := 0.0, 0
	for i := 1; i < len(q); i += 2 {
		m += q[i]
		n++
	}
	m /= float64(n)
	for i := 1; i < len(q); i += 2 {
		q[i] = lerpf(q[i], m, ease) + add
	}
	return q
}

// pSleeve draws the sleeve of arm i on l: the arm's own shape with cloth
// over it, folds in the crook of a bent elbow, and a turned cuff at the
// wrist, cut square.
func (g *figGen) pSleeve(l *figLayer, a armGeom, bias float64, d sim.Dir, i int) {
	f := &g.f
	sp := pspec()
	v := viewOf(d)
	female := g.look.Female
	ku, kf := sp.arm, sp.fore
	if female {
		ku, kf = 0.92, 0.95
	}
	ux, uy, n := unitOf(a.ex-a.sx, a.ey-a.sy)
	vx, vy, m := unitOf(a.wx-a.ex, a.wy-a.ey)
	px, py, qx, qy := uy, -ux, vy, -vx
	if !v.side {
		lat := 1.0 - 2*float64(i)
		if px*lat < 0 || px == 0 && py > 0 {
			px, py = -px, -py
		}
		if qx*lat < 0 || qx == 0 && qy > 0 {
			qx, qy = -qx, -qy
		}
	}
	uo, ui, fo, fi := pUpperOut, pUpperIn, pForeOut, pForeIn
	switch {
	case v.side:
		uo, ui, fo, fi = pUpperFr, pUpperBk, pForeFr, pForeBk
	case female:
		uo, ui, fo, fi = pUpperOutF, pUpperInF, pForeOutF, pForeInF
	}
	if female && v.side {
		ku, kf = ku*0.8, kf*0.82
	}
	add := 0.55
	if female {
		add = 0.45
	}
	uo, ui = pLoose(uo, ku, 0.45, add), pLoose(ui, ku, 0.45, add)
	fo, fi = pLoose(fo, kf, 0.5, add+0.15), pLoose(fi, kf, 0.5, add+0.15)
	sx, sy := a.sx, a.sy+1.8
	if !v.side {
		sx += px * 0.4 * (ku - 1) * 4
	}
	cut := sFn(func(x, y float64) float64 { return (x-a.ex)*vx + (y-a.ey)*vy - (m + 0.7) })
	sleeve := sLimb(sx, sy, a.ex, a.ey, px, py, uo, ui).or(sLimb(a.ex, a.ey, a.wx, a.wy, qx, qy, fo, fi).and(cut))
	hU := clothHt(sx, sy, a.ex, a.ey, px, py, uo, ui)
	hF := clothHt(a.ex, a.ey, a.wx, a.wy, qx, qy, fo, fi)
	foreTop := !v.back

	// The crook of the elbow: the side the forearm bends to. The more it
	// bends, the deeper the folds that gather there.
	cx, cy, _ := unitOf(vx-ux, vy-uy)
	bend := clamp01((math.Acos(math.Max(-1, math.Min(1, ux*vx+uy*vy))) - 0.25) / 1.2)
	r0 := uo.at(1)
	kx, ky := a.ex+cx*r0*0.7, a.ey+cy*r0*0.7
	cuff := m - 2.2 // where the cuff begins, down the forearm
	h := func(x, y float64) float64 {
		u, fh := hU(x, y), hF(x, y)
		if (x-a.ex)*vx+(y-a.ey)*vy > m+0.7 {
			fh = 0
		}
		var hh float64
		if foreTop && fh > 0 {
			hh = math.Max(u*0.85, fh)
		} else {
			hh = math.Max(u, fh)
		}
		if hh <= 0 {
			return 0
		}
		// the folds round the crook, in rings
		if dk := math.Hypot(x-kx, y-ky); bend > 0 && dk < 6 && (x-a.ex)*cx+(y-a.ey)*cy > -1.5 {
			for _, ri := range [3]float64{1.4, 3.0, 4.6} {
				hh -= 0.75 * bend * math.Exp(-math.Pow((dk-ri)/0.6, 2))
			}
		}
		// a long soft fold down the upper arm where it hangs
		if t := (x-sx)*ux + (y-sy)*uy; t > 3 && t < n-2 {
			c := (x-sx)*px + (y-sy)*py
			hh -= 0.35 * (1 - bend) * math.Exp(-math.Pow((c-uo.at(t/n)*0.35)/0.7, 2))
		}
		// the cuff, turned back: a raised band
		if t := (x-a.ex)*vx + (y-a.ey)*vy; t > cuff-0.5 {
			hh += 0.7 * clamp01((t-cuff+0.5)/0.8)
		}
		return hh
	}
	f.fill(l, sleeve, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		lv := pLight(h, x, y)*0.95 + bias + 0.02
		if c := pCrease(h, x, y, 1.6); c > 0.25 {
			lv -= math.Min(0.24, (c-0.25)*0.45)
		}
		if t := (fx-a.ex)*vx + (fy-a.ey)*vy; t > cuff && t < cuff+1 && !female {
			lv += 0.08 // the cuff's fold catches the light
		}
		return pClothTone(pCloth, lv)
	})
	f.pClean(l, f.n, 2)
}

// pTunic draws a man's tunic or a woman's dress on S, as tunic does: its
// cloth lit over the body under it, the skirt in folds from the belt, a
// stitched hem; the neck, the belt and a dress's brooches on T, and the
// hook after the belt.
func (g *figGen) pTunic(d sim.Dir, sk skirt) {
	f := &g.f
	female := g.look.Female
	v := viewOf(d)
	torso, hemAt := tallTorso(female, d, sk)
	under := pTrunkH(female, v, pTrunk(female, v)) // the body's forms show through
	bias := 0.03
	switch {
	case female && v.side:
		bias = -0.12
	case female:
		bias = -0.04
	case v.side:
		bias = -0.1 // a step down from the near arm
	}
	folds := []float64{-0.3, -0.1, 0.12, 0.33} // the skirt's folds, fanning from the belt
	fold0 := 50.0
	if female {
		folds, fold0 = []float64{-0.3, -0.12, 0.1, 0.3}, 46
		if v.side {
			folds = []float64{-0.24, 0.02, 0.26}
		}
	} else if v.side {
		folds = []float64{-0.22, 0.04, 0.28}
	}
	skirtFrom := beltY + 3
	if female {
		skirtFrom = 67
	}
	h := func(x, y float64) float64 {
		hh := edgeH(torso.d(x, y), 8) + 0.1*under(x, y)
		dx := x - 27
		if y > skirtFrom {
			e := clamp01((y - skirtFrom) / 5)
			for _, k := range folds {
				hh -= 1.2 * e * math.Exp(-math.Pow((dx-k*(y-fold0))/1.05, 2))
			}
		}
		if !female && y > beltY-6 && y < beltY { // gathered into the belt
			e := clamp01((y - beltY + 6) / 6)
			u := dx/3.4 + 50
			hh -= 0.45 * e * math.Pow(math.Abs(u-math.Floor(u)-0.5)*2, 3)
		}
		if hm := hemAt(x); y > hm-4.5 && !female { // the hem, turned up: a band
			hh += 0.5 * clamp01((y-hm+4.5)/0.8)
		}
		return hh
	}
	f.fill(g.S, torso, func(x, y int, _ float64) color.RGBA {
		lv := pLight(h, x, y) + bias
		if c := pCrease(h, x, y, 1.6); c > 0.25 {
			lv -= math.Min(0.24, (c-0.25)*0.45)
		}
		dx := float64(x) + 0.5 - 27
		hm := int(math.Round(hemAt(float64(x) + 0.5)))
		if !female {
			switch {
			case y >= hm-4: // the band along the hem, stitched
				lv += 0.1
				if y == hm-3 && x%3 == 0 {
					lv -= 0.2
				}
				if y >= hm-1 {
					lv -= 0.12
				}
			case !v.back && !v.side && x == 27 && y < 59:
				lv -= 0.32 // the slit at the neck
			case !v.back && !v.side && (x == 25 || x == 29) && y < 59:
				lv += 0.12 // and its trim
			}
			return pClothTone(pCloth, lv)
		}
		// A light shift, and over it the apron dress, darker.
		lv -= 0.06
		switch {
		case y >= hm-3: // the shift's hem under the apron
			lv = math.Max(lv+0.3, 0.62)
			if y == hm-3 {
				lv = 0.22
			}
		case y >= hm-7 && y <= hm-5: // a woven band
			lv = 0.24
			if y == hm-6 && x%4 == 1 {
				lv = 0.86
			} else if y != hm-6 && x%4 == 3 {
				lv = 0.52
			}
		case y == 65 || y == 66: // tied at the waist
			lv -= 0.12 + 0.12*float64(66-y)
		case v.side:
		case !v.back && y < 57 && math.Abs(dx) < (57-float64(y))*0.5: // the shift at the neck
			lv = math.Max(lv+0.3, 0.72)
			if y < 51 {
				lv -= 0.12
			}
		case !v.back && y < 64 && math.Abs(math.Abs(dx)-5) < 1.2: // the straps
			lv -= 0.14
		}
		if v.back && y < 64 { // the straps cross on the back
			if c := 5.5 - 11*(float64(y)-48)/16; math.Abs(dx-c) < 1.2 || math.Abs(dx+c) < 1.2 {
				lv -= 0.14
			}
		}
		return pClothTone(pCloth, lv)
	})
	f.pClean(g.S, f.n, 2)

	switch {
	case female:
	case v.side:
		g.tallBelt(21, 33.6, false)
		for y := int(beltY); y <= int(beltY)+2; y++ { // the buckle, round the front
			f.dot(g.T, 31, y, goldHD[int(beltY)+3-y])
			f.dot(g.T, 32, y, goldHD[int(beltY)+4-y])
		}
	default:
		g.tallBelt(18.2, 35.8, !v.back)
	}
	if g.hook != nil {
		g.hook()
	}

	// The neck, as the bare body's, down into the collar.
	rp := g.pRamps()
	neck := sBox(24.2, 39.5, 29.8, 47.4, 1.4)
	if female {
		neck = sBox(24.8, 39.5, 29.2, 47.4, 1.2)
	}
	if v.side {
		neck = sPoly(24.4, 39.5, 30.6, 39.5, 30.9, 47, 24, 47)
		if female {
			neck = sPoly(24.8, 39.5, 30, 39.5, 30.3, 47, 24.5, 47)
		}
	}
	nh := func(x, y float64) float64 {
		hh := edgeH(neck.d(x, y), 2.6)
		if !v.side && !v.back && !female {
			for _, s := range [2]float64{-1, 1} {
				cx := 27 + s*lerpf(2.3, 0.9, clamp01((y-41)/6.5))
				hh += 0.55 * math.Exp(-math.Pow((x-cx)/0.8, 2))
			}
		}
		return hh
	}
	f.fill(g.T, neck, func(x, y int, _ float64) color.RGBA {
		b := -0.06
		if float64(y)+0.5 < 45.5 {
			b = -0.2 // in the chin's shade
		}
		return g.pSkinShade(&rp, neck, nh, x, y, b)
	})

	switch {
	case female && v.side:
		g.brooch(32, 53, 2, false)
	case female && !v.back:
		// Two oval brooches with a string of beads between.
		for i := 0; i < 5; i++ {
			t := float64(i+1) / 6
			bx, by := lerpf(22.5, 31.5, t), 54+4*4*t*(1-t)
			col := beadColors[i%len(beadColors)]
			f.n++
			f.dot(g.T, int(bx), int(by), col)
			f.dot(g.T, int(bx), int(by)+1, pLineOf(col, 0.35))
		}
		g.brooch(22, 53, 2, false)
		g.brooch(32, 53, 2, false)
	}
}

// pMantle draws the fur mantle on T, over the shoulders: its locks hanging
// from the neck, pinned with a brooch.
func (g *figGen) pMantle(d sim.Dir) {
	f := &g.f
	female, v := g.look.Female, viewOf(d)
	var m shape
	top := 45.5
	switch {
	case female && v.side:
		m = sBox(21.5, 45.5, 33, 50.5, 2.4).jag(1.1, 3, 48.5, 23)
	case female:
		m = sBox(17, 45.5, 37, 50.5, 2.4).jag(1.1, 3, 48.5, 23)
	case v.side:
		m = sBox(20, 45.5, 34.6, 50.5, 3).or(sEll(27, 50.5, 7.4, 4.8)).jag(1.4, 3.4, 51.5, 21)
	case v.back:
		m = sBox(12.5, 45.5, 41.5, 57, 3.5).jag(1.7, 4, 54, 21)
	default:
		m = sBox(12.5, 45.5, 41.5, 50, 2.5).or(sEll(16.2, 50.5, 5.2, 4.6)).or(sEll(37.8, 50.5, 5.2, 4.6)).jag(1.4, 3.4, 51.5, 21)
	}
	f.fill(g.T, m, func(x, y int, _ float64) color.RGBA {
		lv := m.lum(x, y, 4.5)*0.7 + 0.18
		return pFurAt(pMantleR, x, y, lv, 31, 3, top)
	})
	f.dropSpecks(g.T, f.n, 5)
	switch {
	case female:
	case v.side:
		g.brooch(33.2, 52, 2.2, true)
	case !v.back:
		g.brooch(19.5, 52, 2.4, true)
	}
}

// pLegFig is leg for the figure: bare (pLeg), or in a trouser leg with its
// shin wrapped; and the foot bare (pFoot) or booted.
func (g *figGen) pLegFig(lg legGeom, dim, toe float64, view sim.Dir) {
	wraps := g.wear&sim.WearLegWraps != 0
	if wraps {
		g.pWrappedLeg(lg, dim, view)
	} else {
		g.pLeg(lg, dim, view)
	}
	switch {
	case lg.tuck:
	case g.wear&sim.WearBoots != 0:
		g.pBoot(lg, dim, toe, view == sim.DirRight)
	default:
		if wraps {
			g.f.n++ // a part of its own under the wraps
		}
		g.pFoot(lg, dim, toe, view)
	}
}

// pWrappedLeg draws a leg in its trouser leg, down over the knee, and the
// shin bound in strips of linen with a leather thong crossed over them.
func (g *figGen) pWrappedLeg(lg legGeom, dim float64, view sim.Dir) {
	f := &g.f
	sp := pspec()
	v := viewOf(view)
	female := g.look.Female
	kt, kc := sp.thigh, sp.calf
	if female {
		kt, kc = kt*0.9, kc*0.84
	}
	tx, ty, tn := unitOf(lg.kx-lg.hx, lg.ky-lg.hy)
	sx, sy, sn := unitOf(lg.ax-lg.kx, lg.ay-lg.ky)
	px, py, qx, qy := ty, -tx, sy, -sx
	if !v.side {
		out := 1.0
		if lg.hx < 27 {
			out = -1
		}
		if px*out < 0 {
			px, py = -px, -py
		}
		if qx*out < 0 {
			qx, qy = -qx, -qy
		}
	}
	to, ti, so, si := pThighOut, pThighIn, pShinOut, pShinIn
	if v.side {
		to, ti, so, si = pThighFr, pThighBk, pShinFr, pShinBk
	}
	to, ti = pLoose(to, kt, 0.4, 0.5), pLoose(ti, kt, 0.4, 0.5)
	so, si = pLoose(so, kc, 0.55, 0.75), pLoose(si, kc, 0.55, 0.75)
	k := math.Min(4.5, sn/2) // the trouser leg comes this far down the shin
	wrapAt := k - 0.4
	thigh := sLimb(lg.hx, lg.hy, lg.kx, lg.ky, px, py, to, ti)
	shin := sLimb(lg.kx, lg.ky, lg.ax, lg.ay, qx, qy, so, si)
	along := func(x, y float64) float64 { return (x-lg.kx)*sx + (y-lg.ky)*sy }
	trouser := thigh.or(shin.and(sFn(func(x, y float64) float64 { return along(x, y) - k })))
	hT := clothHt(lg.hx, lg.hy, lg.kx, lg.ky, px, py, to, ti)
	hS := clothHt(lg.kx, lg.ky, lg.ax, lg.ay, qx, qy, so, si)
	// The trouser leg: soft folds behind the knee and where it is tucked
	// into the wraps, a crease down the thigh.
	ht := func(x, y float64) float64 {
		hh := math.Max(hT(x, y), hS(x, y))
		if hh <= 0 {
			return 0
		}
		t := ((x-lg.hx)*tx + (y-lg.hy)*ty) / tn
		c := (x-lg.hx)*px + (y-lg.hy)*py
		hh -= 0.2 * math.Exp(-math.Pow((c-to.at(t)*0.2)/0.8, 2)) * b01(t*tn, 4, tn-3) // the crease
		if s := along(x, y); s > -4 && s < k {
			for _, r := range [2]float64{-2.2, 0.6} { // at the knee
				hh -= 0.5 * math.Exp(-math.Pow((s-r-0.3*c)/0.55, 2))
			}
		}
		if s := along(x, y); s > k-1.6 { // bloused over the wraps
			hh += 0.5 * clamp01((s-k+1.6)/1.2)
		}
		return hh
	}
	f.fill(g.L, trouser, func(x, y int, _ float64) color.RGBA {
		lv := pLight(ht, x, y) - dim + 0.04
		if c := pCrease(ht, x, y, 1.6); c > 0.25 {
			lv -= math.Min(0.24, (c-0.25)*0.45)
		}
		return pClothTone(pTrouser, lv)
	})
	f.pClean(g.L, f.n, 2)

	// The wraps: strips wound slanting round the shin, each lit on its top
	// edge and shaded under, a dark line between two; and the thong crossed
	// over them in front.
	wr := shin.and(sFn(func(x, y float64) float64 { return wrapAt - along(x, y) }))
	const pitch = 3.1
	f.fill(g.L, wr, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		s, c := along(fx, fy), (fx-lg.kx)*qx+(fy-lg.ky)*qy
		w := so.at(s / sn)
		if c < 0 {
			w = si.at(s / sn)
		}
		lv := pLight(hS, x, y)*0.9 - dim + 0.06
		u := (s - wrapAt + 0.42*c) / pitch
		switch fr := u - math.Floor(u); {
		case fr < 0.3:
			lv += 0.1 // the strip's top, rounded to the light
		case fr > 0.74:
			lv -= 0.26 // tucked under the next
		case fr > 0.55:
			lv -= 0.08
		}
		// the thong, crossing over the front of the shin
		if !v.back && s > wrapAt+2 && s < sn-1.5 && math.Abs(c) < w*0.8 {
			a, b := s+0.9*c, s-0.9*c
			if math.Abs(a-9*math.Round(a/9)) < 0.55 || math.Abs(b-9*math.Round(b/9)) < 0.55 {
				return pBootR[2+int(clamp01(lv)*2.9)]
			}
		}
		return pClothTone(pWrap, lv)
	})
}

// pBoot draws a leg's boot on L: soft leather shaped round the foot, a
// darker sole, a seam over the instep, and a fur cuff rolled over the top.
func (g *figGen) pBoot(lg legGeom, dim, toe float64, side bool) {
	f := &g.f
	sp := pspec()
	ax, ay := math.Round(lg.ax*2)/2, lg.ay
	sn, cs := math.Sin(lg.pitch), math.Cos(lg.pitch)
	k := sp.foot * 1.04
	if g.look.Female {
		k *= 0.9
	}
	var boot shape
	if side {
		boot = sPoly(ax-3.9*k, ay-0.8, ax-3.4*k, ay-2.6, ax+2.9*k, ay-2.6, ax+4.4*k, ay+0.2, ax+7.6*k, ay+1.6, ax+9.6*k, ay+3,
			ax+10*k, ay+5, ax-3.9*k, ay+5).
			or(sEll(ax-1.6*k, ay+2.6, 2.7*k, 2.7)).
			or(sEll(ax+8*k, ay+3.4, 2.2*k, 1.8)).
			and(sAbove(ay + 5))
		boot = sRot(boot, ax, ay, lg.pitch).and(sAbove(limbGround))
	} else {
		hw, tw := 4.1, 4.5
		boot = sRot(sBox(ax-hw, ay-0.6, ax+hw, ay+5, 2.4).or(sEll(ax+toe, ay+2.8, tw, 2.5)).and(sAbove(ay+5)), ax, ay, lg.pitch)
	}
	hb := func(x, y float64) float64 { return edgeH(boot.d(x, y), 2.6) }
	f.fill(g.L, boot, func(px, py int, _ float64) color.RGBA {
		dx, dy := float64(px)+0.5-ax, float64(py)+0.5-ay
		u, w := dx*cs+dy*sn, -dx*sn+dy*cs
		if w >= 4.1 { // the sole
			if w < 4.9 && side {
				return pBootR[1]
			}
			return pBootR[0]
		}
		lv := pLight(hb, px, py) - dim - 0.04
		switch {
		case side && math.Abs(w-(0.6+0.45*u)) < 0.5 && u > 0.5 && u < 6.5*k: // the seam over the instep
			lv -= 0.2
		case !side && math.Abs(dx-toe*0.6) < 0.5 && w > 0.6 && w < 3.2:
			lv -= 0.18
		case side && u > 6.8*k && w < 3: // the toe, scuffed lighter
			lv += 0.06
		}
		c := pClothTone(pBootR, lv)
		if lv > 0.84 {
			c = mix(c, pBootR[6], 0.5) // the leather's sheen
		}
		return c
	})
	// The cuff: fur rolled over the top of the boot, square to the shin.
	sx, sy := lg.ax-lg.kx, lg.ay-lg.ky
	cw := 4.7
	if side {
		cw = 5.1
	}
	cuff := sRot(sBox(ax-cw, ay-4.6, ax+cw, ay+0.6, 2.4).jag(1.2, 2.8, ay-1.6, 5), ax, ay, -math.Atan2(sx, sy)*0.7)
	f.fill(g.L, cuff, func(px, py int, _ float64) color.RGBA {
		lv := cuff.lum(px, py, 3)*0.75 + 0.15 - dim
		return pFurAt(pCuffR, px, py, lv, 11, 2.8, ay-4.8)
	})
	f.dropSpecks(g.L, f.n, 4)
}

// clothHt is limbHt for cloth over a limb: as high across its middle on
// either side, however much further it reaches on one (limbHt's surface
// steps where out and in differ, which cloth, smooth, shows).
func clothHt(ax, ay, bx, by, px, py float64, out, in prof) pHeight {
	ux, uy, n := unitOf(bx-ax, by-ay)
	return func(x, y float64) float64 {
		dx, dy := x-ax, y-ay
		s, c := dx*ux+dy*uy, dx*px+dy*py
		t := clamp01(s / n)
		ro, ri := out.at(t), in.at(t)
		r := ro
		if c < 0 {
			r = ri
		}
		e := math.Abs(c)
		switch {
		case s < 0:
			e = math.Hypot(s, c)
		case s > n:
			e = math.Hypot(s-n, c)
		}
		if e >= r {
			return 0
		}
		q := e / r
		return (ro + ri) / 2 * math.Sqrt(1-q*q)
	}
}
