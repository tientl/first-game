package client

import (
	"image/color"
	"math"
)

// The furred beasts at HD density: the woolly mammoth, the sabre-tooth, the
// cave bear and the monkey. Each is a body of blended lumps under tufted
// fur, on legs of two bones that reach for where the gait puts each foot.

func init() {
	set := func(name string, hd func(pose, f int) (*canvas, float64, float64), anim map[int]int) {
		critKinds[name].hd, critKinds[name].anim = hd, anim
	}
	set("mammoth", mammothHD, map[int]int{mammothWalkA: cycFrames, mammothCharge: cycFrames, mammothGore: blowFrames})
	set("sabre", sabreHD, map[int]int{sabreWalkA: cycFrames, sabreWalkB: cycFrames, sabreStalk: cycFrames, sabreSpring: blowFrames})
	set("bear", bearHD, map[int]int{bearWalkA: cycFrames, bearWalkB: cycFrames, bearClaw: blowFrames, bearBite: blowFrames})
	set("monkey", monkeyHD, map[int]int{monkeyWalkA: cycFrames, monkeyLeap: cycFrames})
}

// gait is a four-footed stride: where each foot is at phase ph. The feet
// are numbered near hind, near fore, far hind, far fore.
type gait struct {
	on           bool
	ph           float64
	stride, lift float64
	duty         float64
	offs         [4]float64
}

var (
	walkOffs = [4]float64{0, 0.25, 0.5, 0.75} // one foot after another
	trotOffs = [4]float64{0, 0.5, 0.5, 0}     // in diagonal pairs
	runOffs  = [4]float64{0, 0.42, 0.1, 0.52} // a gallop: the hind pair, then the fore pair
)

func (g gait) foot(i int) (dx, up float64) {
	if !g.on {
		return 0, 0
	}
	return pace(g.ph+g.offs[i], g.duty, g.stride, g.lift)
}

// bob is how far the body sinks at this point of the stride.
func (g gait) bob(amp float64) float64 {
	if !g.on {
		return 0
	}
	return amp * 0.5 * (1 + math.Sin(4*math.Pi*g.ph))
}

// lerp2 is the point t of the way from a to b.
func lerp2(a, b [2]float64, t float64) [2]float64 {
	return [2]float64{lerpf(a[0], b[0], t), lerpf(a[1], b[1], t)}
}

// --- the woolly mammoth ----------------------------------------------------------

func mammothHD(pose, f int) (*canvas, float64, float64) {
	q := newCrit(124, 84)
	const gy = 79.6
	hair := func(s shape, b float64) shader { return shag(mammothFur, s, 26, b-0.2, 13, 301) }
	g := gait{stride: 9, lift: 3, duty: 0.7, offs: walkOffs}
	headA := 0.0
	// The trunk: its bends, from where it leaves the face.
	hang := [4][2]float64{{0, 0}, {7, 6}, {6, 20}, {3, 31}}
	curl := [4][2]float64{{0, 0}, {6, 9}, {1, 21}, {-7, 16}}
	high := [4][2]float64{{0, 0}, {11, -1}, {16, -17}, {10, -31}}
	trunk := hang
	switch pose {
	case mammothWalkA:
		if f > 0 {
			g.on, g.ph = true, cycPh(f)
			s := math.Sin(2 * math.Pi * g.ph)
			trunk[2][0] += 1.5 * s
			trunk[3][0] += 3 * s
		}
	case mammothWalkB:
		g.on, g.ph = true, 0.5
	case mammothCharge:
		headA, trunk = 0.2, curl
		g.on, g.ph, g.stride, g.lift, g.duty, g.offs = true, 0.1, 13, 4.5, 0.55, runOffs
		if f > 0 {
			g.ph = cycPh(f)
		}
	case mammothGore:
		t := (blowAmt(f) + 1) / 2 // head down and trunk curled, then tossed up
		headA = lerpf(0.26, -0.3, t)
		for i := range trunk {
			trunk[i] = lerp2(curl[i], high[i], smooth(t))
		}
	case mammothEatA:
		headA, trunk = 0.1, [4][2]float64{{0, 0}, {8, 8}, {5, 25}, {-5, 31}}
	case mammothEatB:
		headA, trunk = 0.04, [4][2]float64{{0, 0}, {10, 8}, {9, 21}, {-2, 9}}
	}
	bob := g.bob(1.2)
	const nx, ny = 80.0, 40.0 // the head turns about the neck
	H := func(s shape) shape { return s.turned(nx, ny+bob, headA) }
	HP := func(x, y float64) (float64, float64) { return turnXY(x, y+bob, nx, ny+bob, headA) }

	leg := func(i int, hx, hy float64, near bool) {
		dx, up := g.foot(i)
		fx, fy := hx-1+dx, gy-up
		bend := -1.0
		if i%2 == 1 {
			bend = 1
		}
		kx, ky := ik2(hx, hy+bob, fx, fy-3, 9.9, 9.8, bend)
		lg := aLeg(hx, hy+bob, kx, ky, fx, fy-3.2, 8, 6.6, 6.4).jag(6, 7, (gy-10)*cK, 311)
		b := -0.2
		if near {
			b = -0.04
		}
		q.fill(lg, hair(lg, b))
		foot := aEll(fx+0.4, fy-2, 6.8, 2.4)
		q.fill(foot, shd(mammothSkin, foot, 2, b-0.12))
		for k := 0.0; k < 3; k++ { // toenails
			q.flat(aEll(fx+1.4+2.2*k, fy-1.1, 0.95, 1.05), plain(tuskRamp.at(1+int(k)%2)))
		}
	}
	tusk := func(dx, dy float64, far bool) {
		sx, sy := HP(97.5+dx, 49+dy)
		var c [4][2]float64
		for i, o := range [4][2]float64{{0, 0}, {2, 12}, {14, 19}, {21.5, 3}} {
			c[i][0], c[i][1] = turnXY(sx+o[0], sy+o[1], sx, sy, headA)
		}
		tb := newCTube(bez(16, 2.5, 0.7, c[0], c[1], c[2], c[3]))
		b := 0.08
		if far {
			b = -0.2
		}
		q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
			if u := s * 9; u-math.Floor(u) < 0.12 && s < 0.5 {
				l -= 0.1 // growth rings near the root
			}
			return toneOf(tuskRamp, l+b)
		}))
	}

	leg(2, 45, 55, false)
	leg(3, 83, 54, false)
	tl := aCap(21.5, 40+bob, 19.4, 53+bob, 1.9, 1)
	q.fill(tl, hair(tl, -0.12))
	tuft := aEll(19.2, 55.5+bob, 1.9, 2.8)
	q.fill(tuft, hair(tuft, -0.25))
	tusk(3.5, -1, true)

	body := sBlend(6, aEll(39, 46+bob, 19, 18), aEll(58, 45+bob, 24, 20.5), aEll(74, 36+bob, 17.5, 20)).jag(8, 5, (54+bob)*cK, 305)
	q.fill(body, hair(body, 0))
	leg(0, 36, 57, true)
	leg(1, 75, 57, true)

	head := H(sBlend(4, aEll(92, 29+bob, 12, 15), aEll(96.5, 43+bob, 7.6, 8.5))).jag(7, 8, (38+bob)*cK, 307)
	q.edge = 0.2
	q.fill(head, hair(head, 0.03))
	q.edge = 0.34
	// The trunk: bare ringed skin, thick at the face, a curl at the tip.
	sx, sy := HP(101.5, 43)
	var tc [4][2]float64
	for i, o := range trunk {
		tc[i] = [2]float64{sx + o[0], sy + o[1]}
	}
	tb := newCTube(bez(22, 4.4, 1.9, tc[0], tc[1], tc[2], tc[3]))
	q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
		if u := s * 22; u-math.Floor(u) < 0.22 {
			l -= 0.13
		}
		return toneOf(mammothSkin, l+0.02)
	}))
	brow := H(aEll(96, 36.5+bob, 6, 4.5)) // the fur of the face falls over the root of the trunk
	q.fill(brow.jag(4, 6, (35+bob)*cK, 309), hair(head, 0.05))
	ear := H(aEll(84.5, 36+bob, 5, 7.6))
	q.fill(ear, hair(ear, -0.13))
	tusk(0, 0, false)
	ex, ey := HP(96, 31.5)
	q.flat(aEll(ex, ey, 1.5, 1.3), plain(mammothSkin[1]))
	q.bead(ex+0.2, ey+0.1, 1.7)
	return q.done(0.8, 61, 78)
}

// --- the sabre-tooth ---------------------------------------------------------------

func sabreHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(72, 48, 4, 6)
	const gy = 38.9
	coat := func(s shape, b float64) shader { return shag(sabreFur, s, 7, b-0.04, 5, 401) }
	cream := func(s shape, b float64) shader { return fur(sabreCream, s, 3, b, 402) }
	g := gait{stride: 7.5, lift: 2.6, duty: 0.65, offs: walkOffs}
	var crouch, pitch, hdx, hdy, ha, jaw, reach float64
	earsBack, air := false, false
	switch pose {
	case sabreWalkA:
		g.on, g.ph = true, 0.04
		if f > 0 {
			g.ph = cycPh(f)
		}
	case sabreWalkB:
		g.on, g.ph = true, 0.54
		if f > 0 { // running
			g.ph, g.stride, g.lift, g.duty, g.offs = cycPh(f), 11, 3.6, 0.5, runOffs
			pitch = 0.05 * math.Sin(2*math.Pi*g.ph)
		}
	case sabreStalk:
		crouch, hdx, hdy = 4.5, 2.5, 2.5
		if f > 0 {
			g.on, g.ph, g.stride, g.lift = true, cycPh(f), 5.5, 1.6
		}
	case sabreSpring:
		if amt := blowAmt(f); amt < 0 { // gathered on its haunches
			crouch, pitch, hdx, hdy, jaw = -6*amt, 0.14*amt, 1, -1.5*amt, 0.3
		} else {
			air, reach, pitch, jaw = true, amt, -0.34*amt, 0.4+0.6*amt
			hdx, hdy, ha = 2, -1, -0.1
		}
	case sabreCower:
		crouch, hdx, hdy, ha, earsBack = 8.2, 1, 6, 0.2, true
	case sabreEatA, sabreEatB:
		crouch, pitch, hdx, hdy, ha = 2.5, 0.2, -1, 7.5, 0.75
		if pose == sabreEatB {
			jaw, hdy = 0.7, 6.5
		}
	}
	crouch += g.bob(0.8)
	const cx, cy0 = 27.0, 24.0
	cy := cy0 + crouch
	if air {
		cy -= 3
	}
	T := func(s shape) shape { return s.turned(cx, cy, pitch) }
	P := func(x, y float64) (float64, float64) { return turnXY(x, y+cy-cy0, cx, cy, pitch) }
	paw := func(fx, fy float64, b float64, claws bool) {
		pw := aCap(fx, fy-0.9, fx+2.3, fy-0.7, 1.4, 1.15)
		q.fill(pw, coat(pw, b+0.08))
		if claws {
			for k := 0.0; k < 3; k++ {
				q.fill(aPoly(fx+2.2+0.5*k, fy-1.4+0.5*k, fx+3.9+0.3*k, fy-0.4+0.6*k, fx+2.5+0.4*k, fy-0.5+0.5*k), plain(colBone))
			}
		}
	}
	leg := func(i int, hx0, hy0 float64, near bool) {
		hx, hy := P(hx0, hy0)
		dx, up := g.foot(i)
		fx, fy := hx0-1+dx, gy-up
		hind := i%2 == 0
		if air {
			if hind {
				fx, fy = hx-5-5*reach, hy+9-2*reach
			} else {
				fx, fy = hx+7+4*reach, hy+5-3*reach
			}
		} else if !hind && pitch > 0.1 {
			fx += 3 // eating: the fore paws out in front
		}
		bend := 1.0
		if hind {
			bend = -1
		}
		kx, ky := ik2(hx, hy, fx, fy-1, 6, 6, bend)
		r0 := 3.1
		if hind {
			r0 = 3.9
		}
		lg := aLeg(hx, hy, kx, ky, fx, fy-1.4, r0, 1.9, 1.35)
		b := -0.2
		if near {
			b = 0
		}
		q.fill(lg, coat(lg, b))
		paw(fx, fy, b, air && !hind)
	}
	leg(2, 19, 26, false)
	leg(3, 40, 26, false)
	// A short tail, carried up.
	tx, ty := P(9.5, 21.5)
	tb := newCTube(bez(8, 1.7, 1, [2]float64{tx, ty}, [2]float64{tx - 4, ty - 1}, [2]float64{tx - 6, ty - 6 + 4*clamp01(crouch/8)}, [2]float64{tx - 3.5, ty - 9.5 + 7*clamp01(crouch/8)}))
	q.fill(tb.shape(), coat(tb.shape(), -0.05))

	hx, hy := P(46+hdx, 17+hdy)
	nkx, nky := P(38.5, 21)
	body := sBlend(3.5, T(aEll(17, cy, 8, 7.4)), T(aEll(27, cy+0.2, 10.5, 6.4)), T(aEll(37, cy-0.8, 8, 7.5)), aCap(nkx, nky, hx-1.5, hy+0.8, 5.4, 4.3))
	belly := T(aEll(27.5, cy+6, 12, 2.7))
	q.fill(body, func(x, y int, d float64) color.RGBA {
		ax, ay := (float64(x)+0.5)/cK, (float64(y)+0.5)/cK
		if belly.d(float64(x)+0.5, float64(y)+0.5) < 0 {
			return cream(body, 0.1)(x, y, d)
		}
		col := coat(body, 0)(x, y, d)
		// Faint darker dapples over the back.
		if ay < cy-1 && vnoise(ax*0.55, ay*0.8, 407) > 0.66 {
			col = mix(col, sabreFur[1], 0.22)
		}
		return col
	})
	leg(0, 15.5, 27, true)
	leg(1, 37.5, 27.5, true)

	// The head: a heavy skull, a broad muzzle, and the two long fangs.
	R := func(s shape) shape { return s.turned(hx, hy, ha) }
	RP := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ha) }
	fang := func(dx float64, col color.RGBA) {
		q.fill(R(aPoly(hx+4.9+dx, hy+3, hx+6.5+dx, hy+3.2, hx+6.7+dx, hy+7.6, hx+6.1+dx, hy+9.9, hx+5.4+dx, hy+7)), plain(col))
	}
	fang(-1.5, tuskRamp[1])
	if jaw > 0 {
		lower := R(aCap(hx+0.6, hy+3.4, hx+5.6, hy+4.4+3.4*jaw, 1.5, 1.1))
		q.fill(lower, cream(lower, 0))
		q.flat(R(aPoly(hx+1.2, hy+2.6, hx+6.6, hy+2.6, hx+5.4, hy+3.6+3*jaw)), plain(hex(0x8e2a30)))
	}
	earA := -0.3
	if earsBack {
		earA = -1.5
	}
	ear := R(aEll(hx-1.6, hy-4.9, 1.6, 2.3).turned(hx-1.8, hy-3.4, earA))
	q.fill(ear, coat(ear, -0.12))
	skull := R(sBlend(1.6, aEll(hx, hy, 5.4, 4.9), aEll(hx+4.7, hy+1.5, 3.5, 2.9)))
	chin := R(aEll(hx+4.8, hy+3.5, 4, 2.1))
	q.fill(skull, where(coat(skull, 0.05), cream(skull, 0.08), inside(chin)))
	q.flat(R(aEll(hx-1.4, hy-4.5, 0.7, 1.2).turned(hx-1.8, hy-3.4, earA)), plain(sabreFur[1]))
	fang(0, tuskRamp[4])
	q.dot(hx+6.2, hy+5, 0, 0, tuskRamp[2])
	ex, ey := RP(hx+2.3, hy-1.1)
	q.flat(R(aCap(hx+0.9, hy-2.2, hx+3.6, hy-2.1, 0.35, 0.3)), plain(sabreFur[0])) // the brow
	q.eye(ex, ey, 1.5, hex(0xe8b030))
	nx, ny := RP(hx+7.7, hy+0.4)
	q.flat(aEll(nx, ny, 0.8, 0.7), plain(sabreFur[0]))
	if air { // the fore claws are out
		q.dot(nx, ny, 0, 0, sabreFur[0])
	}
	ay := 38.4
	if air {
		ay = 42.4
	}
	return q.done(0.8, 28.6, ay)
}

// --- the cave bear ---------------------------------------------------------------------

// bearHeadHD draws the bear's head at (hx, hy), turned by ha, the jaws gape open.
func bearHeadHD(q *crit, coat func(shape, float64) shader, hx, hy, ha, gape float64) {
	R := func(s shape) shape { return s.turned(hx, hy, ha) }
	RP := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ha) }
	muz := func(s shape, b float64) shader { return shd(bearMuzzle, s, 1.6, b) }
	if gape > 0 {
		lower := R(aCap(hx+1, hy+2.4, hx+6.4, hy+3+3.2*gape, 1.5, 1.1))
		q.fill(lower, muz(lower, -0.1))
		q.flat(R(aPoly(hx+1.5, hy+1.8, hx+7.6, hy+1.6, hx+6.2, hy+2.6+3*gape)), plain(hex(0x7a1c22)))
		for k := 0.0; k < 2; k++ {
			q.fill(R(aPoly(hx+4.4+1.6*k, hy+2.9+2.6*gape, hx+5+1.6*k, hy+2.9+2.6*gape, hx+4.8+1.6*k, hy+1.6+2.4*gape)), plain(colTooth))
		}
	}
	ear := R(aEll(hx-2.4, hy-4.3, 1.8, 1.8))
	q.fill(ear, coat(ear, -0.1))
	q.flat(R(aEll(hx-2.3, hy-4.2, 0.8, 0.9)), plain(bearRamp[1]))
	skull := R(aEll(hx, hy, 5.1, 4.7))
	q.fill(skull, coat(skull, 0.05))
	snout := R(aCap(hx+2.6, hy+0.9, hx+7.3, hy+1.2, 2.7, 2))
	q.fill(snout, muz(snout, 0.02))
	if gape > 0 {
		for k := 0.0; k < 2; k++ {
			q.fill(R(aPoly(hx+5+1.5*k, hy+2.8, hx+5.7+1.5*k, hy+2.8, hx+5.4+1.5*k, hy+4.3)), plain(colTooth))
		}
	} else {
		q.flat(R(aCap(hx+3.4, hy+2.4, hx+7.6, hy+2.3, 0.25, 0.2)), plain(bearRamp[0]))
	}
	q.flat(R(aEll(hx+9, hy+0.5, 1.15, 0.95)), plain(colInk))
	nx, ny := RP(hx+8.8, hy+0.2)
	q.dot(nx, ny, 0, 0, bearMuzzle[4])
	ex, ey := RP(hx+2.2, hy-1.4)
	q.bead(ex, ey, 1.6)
	q.flat(R(aCap(hx+1, hy-2.6, hx+3.4, hy-2.3, 0.3, 0.25)), plain(bearRamp[0]))
}

func bearHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(60, 53, 4, 7)
	const gy = 41.0
	coat := func(s shape, b float64) shader { return shag(bearRamp, s, 8, b-0.04, 9, 501) }
	paw := func(fx, fy, b float64) {
		pw := aCap(fx-0.4, fy-1.2, fx+2.6, fy-0.9, 2.1, 1.7)
		q.fill(pw, coat(pw, b))
		for k := 0.0; k < 3; k++ {
			q.fill(aPoly(fx+2.6+0.9*k, fy-1.7, fx+4.6+0.7*k, fy-0.1, fx+3+0.9*k, fy-0.2), shd(bearClawRamp, aEll(fx+3.5, fy-1, 2, 1.2), 1, 0.1))
		}
	}
	// An arm or a leg from (hx, hy) to a paw at (fx, fy).
	limb := func(hx, hy, fx, fy, l, r, b, bend float64) {
		kx, ky := ik2(hx, hy, fx, fy-1.6, l, l, bend)
		lg := aLeg(hx, hy, kx, ky, fx, fy-1.8, r, r*0.8, r*0.62)
		q.fill(lg, coat(lg, b))
	}
	if pose == bearClaw {
		// Reared up on its hind legs, roaring: a forepaw held high, then
		// brought down in a swipe.
		amt := blowAmt(f)
		limb(15, 30, 13, gy, 5.9, 3.7, -0.22, -1)
		paw(13, gy, -0.22)
		limb(24, 16, 33.5, 21.5, 5.6, 3.1, -0.24, -1)
		paw(33, 22.5, -0.24)
		body := sBlend(4, aPath(17, 30, 8.2, 18.5, 22, 8.4, 21, 14.5, 7.4), aEll(19.5, 12.5, 6.8, 5.6)).jag(4, 7, 30*cK, 505)
		chest := aEll(24.5, 23, 3.4, 7.5)
		q.fill(body, where(coat(body, 0), coat(body, 0.14), inside(chest)))
		limb(21, 31, 22, gy, 5.9, 3.9, 0.02, -1)
		paw(22, gy, 0.02)
		bearHeadHD(q, coat, 27.5, 10.5, -0.28-0.08*amt, 0.75)
		a := lerpf(-1.25, 0.35, smooth((amt+1)/2)) // the raised arm comes down
		hx, hy := 20.5+15*math.Cos(a), 13+15*math.Sin(a)
		kx, ky := ik2(20.5, 13, hx, hy, 7.8, 7.8, -1)
		arm := aLeg(20.5, 13, kx, ky, hx, hy, 3.7, 3.2, 2.7)
		q.fill(arm, coat(arm, 0.06))
		hand := aEll(hx+1, hy+0.3, 2.8, 2.5)
		q.fill(hand, coat(hand, 0.1))
		for k := -1.0; k <= 1; k++ { // the claws, spread
			ca := a + 0.5 + 0.35*k
			cx, cy := hx+1+2.6*math.Cos(ca), hy+0.3+2.6*math.Sin(ca)
			q.fill(aCap(cx, cy, cx+2.6*math.Cos(ca+0.3), cy+2.6*math.Sin(ca+0.3), 0.6, 0.2), plain(bearClawRamp.at(2-int(math.Abs(k)))))
		}
		return q.done(0.85, 18, gy)
	}
	g := gait{stride: 6, lift: 2.2, duty: 0.68, offs: walkOffs}
	hx, hy, ha, gape := 40.5, 24.5, 0.12, 0.0
	switch pose {
	case bearWalkA:
		g.on, g.ph = true, 0.05
		if f > 0 {
			g.ph = cycPh(f)
		}
	case bearWalkB:
		g.on, g.ph = true, 0.55
		if f > 0 { // a lumbering run
			g.ph, g.stride, g.lift, g.duty, g.offs = cycPh(f), 9, 3, 0.55, runOffs
		}
	case bearBite: // the head drawn back, then lunging, jaws wide
		t := (blowAmt(f) + 1) / 2
		hx, hy, ha, gape = lerpf(37.5, 42.5, t), lerpf(22.5, 27.5, t), lerpf(-0.1, 0.2, t), lerpf(0.15, 0.85, t)
	}
	bob := g.bob(0.9)
	hy += bob
	leg := func(i int, x, y, r, b float64) {
		dx, up := g.foot(i)
		bend := 1.0
		if i%2 == 0 {
			bend = -1
		}
		limb(x, y+bob, x+dx, gy-up, 4.9, r, b, bend)
		paw(x+dx, gy-up, b)
	}
	leg(2, 12, 30, 3.5, -0.24)
	leg(3, 30, 30, 3.4, -0.24)
	tl := aCap(6.5, 24+bob, 4.6, 26+bob, 1.7, 1.2)
	q.fill(tl, coat(tl, -0.1))
	body := sBlend(4, aEll(13, 26.5+bob, 8, 7.4), aEll(21.5, 26.5+bob, 11.5, 7.8), aEll(30, 23.5+bob, 8.2, 8), aCap(33, 23.5+bob, hx-2, hy, 6.4, 4.9)).jag(7, 8, (29+bob)*cK, 503)
	q.fill(body, coat(body, 0))
	leg(0, 16, 31, 3.8, 0.02)
	leg(1, 33, 31, 3.7, 0.04)
	bearHeadHD(q, coat, hx, hy, ha, gape)
	return q.done(0.85, 23, gy)
}

// --- the monkey ----------------------------------------------------------------------------

func monkeyHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(30, 27, 2, 2)
	coat := func(s shape, b float64) shader { return shag(monkeyRamp, s, 2.4, b, 4, 601) }
	face := func(s shape, b float64) shader { return shd(monkeyFace, s, 1.4, b) }
	limb := func(x0, y0, x1, y1, b, bend float64) {
		kx, ky := ik2(x0, y0, x1, y1, 3.3, 3.3, bend)
		lg := aLeg(x0, y0, kx, ky, x1, y1, 1.3, 1, 0.85)
		q.fill(lg, coat(lg, b))
		hand := aEll(x1+0.5, y1+0.1, 1.1, 0.75)
		q.fill(hand, face(hand, -0.1))
	}
	tail := func(pts [4][2]float64) {
		tb := newCTube(bez(12, 0.95, 0.6, pts[0], pts[1], pts[2], pts[3]))
		q.fill(tb.shape(), coat(tb.shape(), -0.12))
	}
	head := func(hx, hy float64) {
		ear := aEll(hx-3.7, hy-0.4, 1.4, 1.6)
		q.fill(ear, face(ear, 0))
		q.flat(aEll(hx-3.6, hy-0.3, 0.6, 0.8), plain(monkeyFace[0]))
		sk := aEll(hx, hy, 4.3, 4)
		q.fill(sk, coat(sk, 0.08))
		fc := sBlend(1, aEll(hx+1.2, hy-0.6, 2.4, 1.9), aEll(hx+1.9, hy+1.5, 2.3, 1.7))
		q.fill(fc, face(fc, 0.1))
		q.bead(hx+1.5, hy-0.7, 1.4)
		q.flat(aCap(hx+2.2, hy+2.1, hx+3.6, hy+1.9, 0.18, 0.18), plain(monkeyFace[0]))
		q.dot(hx+3.4, hy+0.7, 0, 0, monkeyFace[0])
	}
	switch pose {
	case monkeySit, monkeyEat:
		tail([4][2]float64{{10, 20}, {6, 22}, {3, 20}, {3.5, 16.5}})
		body := sBlend(2, aEll(12, 15, 4.6, 5.6), aEll(13, 20.3, 4.1, 2.1))
		chest := aEll(13.9, 16.4, 2.3, 3.3)
		q.fill(body, where(coat(body, 0.05), face(body, 0.05), inside(chest)))
		thigh := aEll(13.6, 19.8, 3, 2.2)
		q.fill(thigh, coat(thigh, 0.1))
		foot := aCap(15.6, 21.2, 18.2, 21.6, 0.95, 0.75)
		q.fill(foot, face(foot, -0.1))
		head(14, 8)
		if pose == monkeyEat {
			limb(14.6, 13.2, 17.4, 10.6, 0.02, 1)
			bn := aCap(18.2, 11, 19.2, 6.8, 0.9, 0.6)
			q.fill(bn, shd(bananaRamp, bn, 0.8, 0.1))
		} else {
			limb(14.4, 13.4, 16.2, 19.2, 0.02, 1)
		}
	case monkeyLeap:
		// A bound: the limbs flung out, gathered under it, flung out again.
		s := 1.0
		if f > 0 {
			s = math.Cos(2 * math.Pi * cycPh(f))
		}
		lag := s
		if f > 0 {
			lag = math.Cos(2*math.Pi*cycPh(f) - 0.9)
		}
		tail([4][2]float64{{6.5, 12}, {3, 11}, {1.5, 7 + lag}, {4, 4.5 + 1.5*lag}})
		limb(8.4, 13.4, 5.5-2.5*s, 17.5+1.5*(1-s), -0.2, -1)
		limb(15.6, 13.4, 17.5+2*s, 17.6+1.2*(1-s), -0.2, 1)
		body := aEll(12, 13, 6.4, 3.2+0.4*(1-s)).turned(12, 13, -0.08*s)
		q.fill(body, coat(body, 0.05))
		limb(7.4, 14, 4.5-2.5*s, 18+1.5*(1-s), 0.02, -1)
		limb(16.4, 14, 18.5+2*s, 18+1.2*(1-s), 0.02, 1)
		head(19, 9+0.5*(1-s))
	default: // on all fours
		g := gait{on: true, ph: 0.05, stride: 3.6, lift: 1.5, duty: 0.6, offs: trotOffs}
		if pose == monkeyWalkB {
			g.ph = 0.55
		}
		if f > 0 {
			g.ph = cycPh(f)
		}
		bob := g.bob(0.5)
		tail([4][2]float64{{7, 13 + bob}, {3.5, 12}, {1.5, 7}, {5, 4.5}})
		leg := func(i int, x, y, b float64) {
			dx, up := g.foot(i)
			bend := 1.0
			if i%2 == 0 {
				bend = -1
			}
			limb(x, y+bob, x+dx, 21.7-up, b, bend)
		}
		leg(2, 9.4, 16, -0.2)
		leg(3, 16.4, 16, -0.2)
		body := sBlend(2, aEll(10.5, 15.2+bob, 4.4, 3.7), aEll(14.5, 14.8+bob, 4.2, 3.4))
		q.fill(body, coat(body, 0.05))
		leg(0, 8.2, 16.6, 0.02)
		leg(1, 15.4, 16.6, 0.02)
		head(18, 11+bob)
	}
	return q.done(0.8, 13, 22)
}
