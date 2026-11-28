package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The pilot's arms, hands, legs and feet (charpilot.go): each limb a
// surface of two cylinders that narrow and swell as the limb does, its
// muscles rises on them, lit as the trunk is (charpilot_body.go).

// limbHt is the surface of a limb from a to b: a cylinder whose radius is
// out.at(t) on the side the unit (px, py) points to and in.at(t) on the
// other, round at its ends.
func limbHt(ax, ay, bx, by, px, py float64, out, in prof) pHeight {
	ux, uy, n := unitOf(bx-ax, by-ay)
	return func(x, y float64) float64 {
		dx, dy := x-ax, y-ay
		s, c := dx*ux+dy*uy, dx*px+dy*py
		w := out
		if c < 0 {
			w = in
		}
		var r, e float64
		switch {
		case s < 0:
			r, e = w.at(0), math.Hypot(s, c)
		case s > n:
			r, e = w.at(1), math.Hypot(s-n, c)
		default:
			r, e = w.at(s/n), math.Abs(c)
		}
		if e >= r {
			return 0
		}
		return math.Sqrt(r*r - e*e)
	}
}

// limbBump is a muscle on a limb from a (unit u along it, p across), s
// along it and c across (a fraction of the reach r there), ra long, rb
// wide, high high.
type limbBump struct{ s, c, ra, rb, high float64 }

func bumpsH(ax, ay, ux, uy, px, py float64, rOut, rIn prof, n float64, bs []limbBump, k float64) pHeight {
	type at struct{ x, y, ra, rb, high float64 }
	var cs []at
	for _, b := range bs {
		r := rOut.at(b.s / n)
		if b.c < 0 {
			r = rIn.at(b.s / n)
		}
		cs = append(cs, at{ax + ux*b.s + px*b.c*r, ay + uy*b.s + py*b.c*r, b.ra * k, b.rb * k, b.high})
	}
	return func(x, y float64) float64 {
		h := 0.0
		for _, c := range cs {
			if q := ellQ(x, y, c.x, c.y, ux, uy, c.ra, c.rb); q < 1 {
				h += bumpH(q, c.high)
			}
		}
		return h
	}
}

// Arm reaches, a man's: down the upper arm and the forearm, on the outside
// (from the front or the back) and the inside, or in front and behind
// (from the side). The deltoid makes the shoulder; the arm narrows to the
// elbow; the forearm is full just below it and slim at the wrist.
var (
	pUpperOut  = prof{0, 3.7, 0.16, 4.7, 0.36, 4.15, 0.65, 3.45, 0.88, 2.95, 1, 2.6}
	pUpperIn   = prof{0, 3.3, 0.45, 3.2, 0.8, 2.8, 1, 2.5}
	pForeOut   = prof{0, 2.6, 0.22, 3.2, 0.5, 2.8, 0.85, 1.95, 1, 1.7}
	pForeIn    = prof{0, 2.4, 0.3, 2.8, 0.7, 2.1, 1, 1.6}
	pUpperFr   = prof{0, 3.6, 0.2, 4.0, 0.45, 3.45, 0.7, 3.35, 1, 2.6}
	pUpperBk   = prof{0, 3.8, 0.25, 4.3, 0.55, 3.75, 0.8, 3.0, 1, 2.6}
	pForeFr    = prof{0, 2.8, 0.22, 3.2, 0.6, 2.45, 1, 1.75}
	pForeBk    = prof{0, 2.8, 0.3, 2.65, 1, 1.7}
	pUpperOutF = prof{0, 3.4, 0.18, 3.6, 0.4, 3.0, 0.62, 2.6, 0.85, 2.4, 1, 2.0}
	pUpperInF  = prof{0, 2.8, 0.4, 2.55, 0.75, 2.35, 1, 1.9}
	pForeOutF  = prof{0, 2.0, 0.25, 2.4, 0.6, 1.9, 1, 1.35}
	pForeInF   = prof{0, 1.85, 0.35, 2.15, 0.75, 1.6, 1, 1.3}
)

// pArm draws arm i of a figure facing d bare on l, and its hand.
func (g *figGen) pArm(l *figLayer, a armGeom, bias float64, d sim.Dir, i int) {
	f := &g.f
	sp := pspec()
	v := viewOf(d)
	female := g.look.Female
	rp := g.pRamps()
	ku, kf := sp.arm, sp.fore
	mk := sp.muscleK
	if female {
		ku, kf, mk = 0.92, 0.95, 0.35
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
		uo, ui, fo, fi = uo.scaled(0.8), ui.scaled(0.8), fo.scaled(0.82), fi.scaled(0.82)
	}
	uo, ui, fo, fi = uo.scaled(ku), ui.scaled(ku), fo.scaled(kf), fi.scaled(kf)
	// The shoulder's root sits a little further out than the joint: the
	// deltoid caps it.
	sx, sy := a.sx, a.sy+1.8 // the cap of the shoulder sits below the line of the trapezius
	if !v.side {
		sx += px * 0.4 * (ku - 1) * 4
	}
	upper := sLimb(sx, sy, a.ex, a.ey, px, py, uo, ui)
	fore := sLimb(a.ex, a.ey, a.wx, a.wy, qx, qy, fo, fi)

	var ub, fb []limbBump
	switch {
	case v.side:
		ub = []limbBump{{2.2, 0, 5, 4.6, 1.0}, {8.4, 0.35, 4.6, 2.6, 0.6}, {6.2, -0.45, 5, 2.6, 0.5}}
		fb = []limbBump{{3.4, 0.3, 4, 2.4, 0.55}}
	case v.back:
		ub = []limbBump{{2.4, 0.15, 5, 4.6, 1.0}, {7, 0.05, 5.4, 3, 0.6}}
		fb = []limbBump{{3.3, 0.35, 4, 2.4, 0.5}}
	default:
		ub = []limbBump{{2.4, 0.2, 5, 4.6, 1.0}, {8.4, -0.1, 4.4, 2.9, 0.6}}
		fb = []limbBump{{3.0, 0.4, 3.8, 2.4, 0.55}}
	}
	for j := range ub {
		ub[j].high *= mk
	}
	for j := range fb {
		fb[j].high *= mk
	}
	hU := limbHt(sx, sy, a.ex, a.ey, px, py, uo, ui)
	hF := limbHt(a.ex, a.ey, a.wx, a.wy, qx, qy, fo, fi)
	bU := bumpsH(sx, sy, ux, uy, px, py, uo, ui, n, ub, ku)
	bF := bumpsH(a.ex, a.ey, vx, vy, qx, qy, fo, fi, m, fb, kf)
	foreTop := !v.back
	h := func(x, y float64) float64 {
		u, fh := hU(x, y), hF(x, y)
		if u > 0 {
			u += bU(x, y)
		}
		if fh > 0 {
			fh += bF(x, y)
		}
		if foreTop && fh > 0 {
			return math.Max(u*0.85, fh)
		}
		return math.Max(u, fh)
	}
	arm := upper.or(fore)
	pWarm = func(x, y float64) float64 {
		return math.Max(0.5*warmNear(x, y, a.hx, a.hy, 5.5), 0.4*warmNear(x, y, a.ex, a.ey, 3))
	}
	defer func() { pWarm = nil }()
	f.fill(l, arm, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		b := bias
		t := (fx-a.ex)*vx + (fy-a.ey)*vy
		e := (fx-a.ex)*qx + (fy-a.ey)*qy
		switch {
		case v.back && t > -0.6 && t < 1.4 && math.Abs(e) < 1.3: // the point of the elbow
			b += 0.1
		case !v.back && t > -0.4 && t < 1.4 && e < -0.3 && e > -2.4*kf && fore.d(fx, fy) < 0: // the crook of the elbow
			b -= 0.2
		}
		return g.pSkinShade(&rp, arm, h, x, y, b)
	})
	f.pClean(l, f.n, 2)
	g.pHand(l, a, v, &rp, bias, false)
	g.pBlendShoulder(sx, sy, 4.4*ku)
}

// pHand draws the hand at the end of arm a: a fist, its fingers curled
// and the thumb over them, seen from its back sideways and from its thumb's
// side from the front; or open, the fingers out. cuff: out of a sleeve.
func (g *figGen) pHand(l *figLayer, a armGeom, v pView, rp *pRamps, bias float64, cuff bool) {
	f := &g.f
	sp := pspec()
	female := g.look.Female
	k := sp.fist / 6
	if female {
		k *= 0.84
	}
	if a.size > 6 { // thrown at the viewer
		k *= 1.35
	}
	ux, uy, _ := unitOf(a.wx-a.ex, a.wy-a.ey) // towards the fingers
	px, py := uy, -ux                         // across: forwards, sideways
	if !v.side {
		if px < 0 {
			px, py = -px, -py
		}
	}
	cx, cy := a.hx, a.hy
	at := func(t, w float64) (float64, float64) { return cx + ux*t + px*w, cy + uy*t + py*w }
	local := func(x, y float64) (float64, float64) {
		dx, dy := x-cx, y-cy
		return dx*ux + dy*uy, dx*px + dy*py
	}
	if a.open {
		// The palm, the four fingers along the forearm, the thumb aside.
		x0, y0 := at(-2.6*k, 0)
		x1, y1 := at(0.6*k, 0)
		hand := sCap(x0, y0, x1, y1, 1.9*k, 2.2*k)
		for j := -1.5; j <= 1.5; j++ {
			fx0, fy0 := at(0.8*k, j*0.95*k)
			fx1, fy1 := at((3.6-math.Abs(j)*0.5)*k, j*1.1*k)
			hand = hand.or(sCap(fx0, fy0, fx1, fy1, 0.62, 0.52))
		}
		tx0, ty0 := at(-1.8*k, -1.7*k)
		tx1, ty1 := at(0.2*k, -3.3*k)
		hand = hand.or(sCap(tx0, ty0, tx1, ty1, 0.78, 0.58))
		hh := func(x, y float64) float64 { return edgeH(hand.d(x, y), 1.4) }
		f.add(l, hand, func(x, y int, _ float64) color.RGBA { return g.pSkinShade(rp, hand, hh, x, y, bias+0.05) })
		return
	}
	// The fist: L long (along the forearm) and B broad, the knuckles a
	// ridge across its back, the curled fingers below them parted by lines,
	// the thumb along the front.
	L, B := 3.1*k, 2.6*k
	if !v.side && a.size <= 6 {
		B = 2.15 * k // edge on
	}
	fist := sFn(func(x, y float64) float64 {
		t, w := local(x, y)
		qx, qy := math.Abs(t)-(L-1.7*k), math.Abs(w)-(B-1.5*k)
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - 1.5*k
	})
	fist.x0, fist.y0, fist.x1, fist.y1 = cx-L-B, cy-L-B, cx+L+B, cy+L+B
	// the wrist into it
	wt := -L - 1.6
	if cuff { // out of a sleeve: no more of the wrist than shows under the cuff
		wt = -L - 0.5
	}
	wx0, wy0 := at(wt, 0)
	wx1, wy1 := at(-L+0.8, 0)
	wrist := sCap(wx0, wy0, wx1, wy1, 1.5*k, 1.9*k)
	shape := fist.or(wrist)
	hh := func(x, y float64) float64 {
		t, w := local(x, y)
		h := edgeH(fist.d(x, y), 2*k)
		if kn := t - 0.5*k; math.Abs(kn) < 1.2*k { // the knuckles' ridge
			h += 0.5 * (1 - math.Abs(kn)/(1.2*k))
		}
		if v.side && w > B*0.25 && t < 0.6*k { // the thumb's mound
			h += 0.4
		}
		return math.Max(h, edgeH(wrist.d(x, y), 1.6))
	}
	f.add(l, shape, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		t, w := local(fx, fy)
		col := g.pSkinShade(rp, shape, hh, x, y, bias+0.04)
		dark := rp.skin[max(1, len(rp.skin)/2-2)]
		switch {
		case fist.d(fx, fy) > 0:
		case v.side && t > 0.9*k && t < L-0.4 && (math.Abs(w-B*0.05) < 0.45 || math.Abs(w+B*0.55) < 0.42):
			return dark // between the fingers
		case v.side && w > B*0.35 && math.Abs(t+0.35*k) < 0.5:
			return mix(col, dark, 0.6) // the thumb's edge over the fingers
		case !v.side && t > 1.2*k && t < L-0.3 && math.Abs(w-0.15) < 0.45 && a.size <= 6:
			return mix(col, dark, 0.7) // the curled fingers under the thumb
		case a.size > 6 && t > 0.4*k && (math.Abs(w-B*0.5) < 0.45 || math.Abs(w) < 0.45 || math.Abs(w+B*0.5) < 0.45):
			return dark // the knuckles, end on
		}
		return col
	})
}

// Leg reaches: the thigh full under the hip and narrowing to the knee,
// the inner bulge just above it; the calf swelling behind the shin.
var (
	pThighOut = prof{0, 5.2, 0.3, 5.3, 0.7, 4.4, 1, 3.5}
	pThighIn  = prof{0, 4.6, 0.6, 4.1, 0.85, 3.9, 1, 3.3}
	pThighFr  = prof{0, 5.0, 0.35, 5.3, 0.8, 4.2, 1, 3.5}
	pThighBk  = prof{0, 5.6, 0.3, 4.9, 0.75, 3.9, 1, 3.3}
	pShinOut  = prof{0, 3.3, 0.3, 3.9, 0.62, 3.1, 1, 2.2}
	pShinIn   = prof{0, 3.2, 0.3, 3.6, 0.62, 2.9, 1, 2.2}
	pShinFr   = prof{0, 3.3, 0.25, 3.0, 0.65, 2.6, 1, 2.2}
	pShinBk   = prof{0, 3.4, 0.3, 4.4, 0.6, 3.3, 1, 2.0}
)

// pLeg draws a bare leg on L: the thigh, the knee, the calf and the shin
// down to the ankle. dim darkens the far leg.
func (g *figGen) pLeg(lg legGeom, dim float64, view sim.Dir) {
	f := &g.f
	sp := pspec()
	v := viewOf(view)
	female := g.look.Female
	rp := g.pRamps()
	kt, kc := sp.thigh, sp.calf
	if female {
		kt, kc = kt*0.9, kc*0.84
	}
	tx, ty, tn := unitOf(lg.kx-lg.hx, lg.ky-lg.hy)
	sx, sy, sn := unitOf(lg.ax-lg.kx, lg.ay-lg.ky)
	// across: forwards seen from the side, outwards from the front or back
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
	if female {
		to, ti = to.scaled(0.95), ti.scaled(1.02)
	}
	to, ti, so, si = to.scaled(kt), ti.scaled(kt), so.scaled(kc), si.scaled(kc)
	thigh := sLimb(lg.hx, lg.hy, lg.kx, lg.ky, px, py, to, ti)
	shin := sLimb(lg.kx, lg.ky, lg.ax, lg.ay, qx, qy, so, si)
	var tb, sb []limbBump
	mk := sp.muscleK
	if female {
		mk *= 0.4
	}
	switch {
	case v.side:
		tb = []limbBump{{tn * 0.45, 0.35, 8, 3.2, 0.55 * mk}}
		sb = []limbBump{{sn * 0.32, -0.45, 5.6, 2.8, 0.6 * mk}}
	case v.back:
		tb = []limbBump{{tn * 0.5, 0, 8, 3.6, 0.4 * mk}}
		sb = []limbBump{{sn * 0.32, 0.35, 5, 2.6, 0.55 * mk}, {sn * 0.36, -0.35, 4.8, 2.4, 0.5 * mk}}
	default:
		tb = []limbBump{{tn * 0.45, 0.35, 8, 3.2, 0.55 * mk}, {tn * 0.84, -0.4, 3.4, 2.6, 0.4 * mk}, {tn * 1.02, 0.05, 2.4, 2.3, 0.4}}
		sb = []limbBump{{sn * 0.3, 0.45, 4.8, 2.4, 0.5 * mk}, {sn * 0.36, -0.45, 4.4, 2.3, 0.45 * mk}}
	}
	hT := limbHt(lg.hx, lg.hy, lg.kx, lg.ky, px, py, to, ti)
	hS := limbHt(lg.kx, lg.ky, lg.ax, lg.ay, qx, qy, so, si)
	bT := bumpsH(lg.hx, lg.hy, tx, ty, px, py, to, ti, tn, tb, kt)
	bS := bumpsH(lg.kx, lg.ky, sx, sy, qx, qy, so, si, sn, sb, kc)
	h := func(x, y float64) float64 {
		a, b := hT(x, y), hS(x, y)
		if a > 0 {
			a += bT(x, y)
		}
		if b > 0 {
			b += bS(x, y)
		}
		return math.Max(a, b)
	}
	leg := thigh.or(shin)
	pWarm = func(x, y float64) float64 {
		return math.Max(0.45*warmNear(x, y, lg.kx, lg.ky, 3.6), 0.3*clamp01((y-lg.ay+4)/4))
	}
	defer func() { pWarm = nil }()
	hemT := (pspec().loinHem - hipY) / thighLen
	if female {
		hemT = (skirtHemF - hipY) / thighLen
	}
	f.fill(g.L, leg, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		b := -dim
		if t := ((fx-lg.hx)*tx + (fy-lg.hy)*ty) / tn; t > hemT-0.15 && t < hemT+0.12 {
			b -= 0.12 // the fur's shadow on the thigh
		}
		if v.back {
			if t := (fx-lg.kx)*sx + (fy-lg.ky)*sy; t > -1 && t < 1 && math.Abs((fx-lg.kx)*qx+(fy-lg.ky)*qy) < 2.2 {
				b -= 0.12 // the hollow of the knee
			}
		}
		return g.pSkinShade(&rp, leg, h, x, y, b)
	})
	f.pClean(g.L, f.n, 2)
}

// pFoot draws a bare foot on L, of the leg last drawn: from the side its
// heel, its arch and its toes; from the front the toes in a row, the big
// one in; from behind the heel.
func (g *figGen) pFoot(lg legGeom, dim, toe float64, view sim.Dir) {
	f := &g.f
	sp := pspec()
	rp := g.pRamps()
	ax, ay := math.Round(lg.ax*2)/2, lg.ay
	pWarm = func(x, y float64) float64 { return 0.35 + 0.3*clamp01((y-ay-1)/3) }
	defer func() { pWarm = nil }()
	k := sp.foot
	if g.look.Female {
		k *= 0.86
	}
	switch view {
	case sim.DirRight:
		sn, cs := math.Sin(lg.pitch), math.Cos(lg.pitch)
		foot := sPoly(ax-3.3*k, ay+0.4, ax-2*k, ay-2, ax+2.2*k, ay-1.9, ax+4.8*k, ay+0.9, ax+8.4*k, ay+2.6, ax+9.6*k, ay+3.9,
			ax+9.3*k, ay+5.4, ax-3.3*k, ay+5.4).
			or(sEll(ax-1.4*k, ay+2.7, 2.6*k, 2.8)).
			or(sEll(ax+7.9*k, ay+3.9, 2.0*k, 1.45)).
			sub(sEll(ax+2*k, ay+5.6, 2.6*k, 0.95)).
			and(sAbove(ay + 5))
		foot = sRot(foot, ax, ay, lg.pitch).and(sAbove(limbGround))
		hh := func(x, y float64) float64 { return edgeH(foot.d(x, y), 2.6) }
		f.add(g.L, foot, func(px, py int, _ float64) color.RGBA {
			dx, dy := float64(px)+0.5-ax, float64(py)+0.5-ay
			u, w := dx*cs+dy*sn, -dx*sn+dy*cs
			col := g.pSkinShade(&rp, foot, hh, px, py, -dim)
			switch {
			case u > 8.1*k && u < 9.4*k && w > 2.5 && w < 3.6: // the big toe's nail
				return mix(col, rp.skin[len(rp.skin)-1], 0.6)
			case math.Abs(u-6.8*k) < 0.5 && w > 3.1: // where the toes begin
				return rp.skin[max(1, len(rp.skin)/2-2)]
			case w >= 4.2:
				return rp.skin[max(1, len(rp.skin)/2-1)] // the sole
			}
			return col
		})
	case sim.DirUp:
		foot := sBox(ax-2.8*k, ay-0.6, ax+2.8*k, ay+5.4, 2.4).or(sEll(ax+toe*0.7, ay+3.6, 3.6*k, 1.8)).and(sAbove(ay + 5))
		hh := func(x, y float64) float64 { return edgeH(foot.d(x, y), 2.4) }
		f.add(g.L, foot, func(px, py int, _ float64) color.RGBA {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			b := -dim
			if math.Abs(fx-ax) > 1 && math.Abs(fx-ax) < 2 && fy < ay+1.6 {
				b -= 0.15 // either side of the tendon
			}
			if fy >= ay+4.1 {
				return rp.skin[max(1, len(rp.skin)/2-1)]
			}
			return g.pSkinShade(&rp, foot, hh, px, py, b)
		})
	default:
		cx := ax + toe*0.9
		in := 1.0 // towards the other foot: the big toe's side
		if toe > 0 {
			in = -1
		}
		foot := sBox(ax-2.8*k, ay-0.6, ax+2.8*k, ay+3.6, 2.2).or(sEll(cx, ay+3.3, 4.4*k, 2.4)).and(sAbove(ay + 5))
		hh := func(x, y float64) float64 { return edgeH(foot.d(x, y), 2.4) }
		f.add(g.L, foot, func(px, py int, _ float64) color.RGBA {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			col := g.pSkinShade(&rp, foot, hh, px, py, -dim)
			if fy > ay+3 {
				// the toes, the big one widest: a dark line between each two
				for _, o := range [3]float64{1.4, -0.5, -2.1} {
					if math.Abs(fx-(cx+in*o*k)) < 0.5 {
						return rp.skin[max(1, len(rp.skin)/2-2)]
					}
				}
			}
			return col
		})
	}
}
