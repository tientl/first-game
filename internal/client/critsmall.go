package client

import (
	"image/color"
	"math"
)

// The small animals at HD density: the rabbit, the worm, the fish, the crab
// and the snake. See critterhd.go for the tools and the frames.

func init() {
	set := func(name string, hd func(pose, f int) (*canvas, float64, float64), anim map[int]int) {
		critKinds[name].hd, critKinds[name].anim = hd, anim
	}
	set("rabbit", rabbitHD, map[int]int{rabbitHop: 3})
	set("worm", wormHD, map[int]int{0: cycFrames})
	set("fish", func(p, f int) (*canvas, float64, float64) { return fishHD(-1, p, f) }, map[int]int{0: cycFrames})
	for v := range fishLooks {
		set(fishKind(v), func(p, f int) (*canvas, float64, float64) { return fishHD(v, p, f) }, map[int]int{0: cycFrames})
	}
	set("crab", crabHD, map[int]int{crabWalkA: cycFrames, crabPinch: 4})
	set("snake", snakeHD, map[int]int{0: cycFrames, snakeStrike: blowFrames})
}

func fishKind(v int) string { return "fish" + string(rune('0'+v)) }

// --- the rabbit --------------------------------------------------------------------

// rabbitHD draws the rabbit: sitting up, asleep in a loaf, and in a leap of
// four frames (0 stretched out level, 1 gathered to spring, 2 rising, 3
// coming down on its fore paws).
func rabbitHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(24, 20, 2, 2)
	dark := func(s shape, b float64) shader { return fur(furRamp, s, 2.4, b, 77) }
	pale := func(s shape, b float64) shader { return fur(paleRamp, s, 2, b, 78) }
	tail := func(s shape) shader { return fur(tailRamp, s, 1.6, 0.12, 79) }
	face := func(ex, ey, nx, ny float64) {
		q.bead(ex, ey, 1.7)
		q.flat(aEll(nx, ny, 0.42, 0.36), plain(colNose))
		q.dot(nx, ny, -3, 2, paleRamp[3]) // whiskers
		q.dot(nx, ny, -5, 3, paleRamp[3])
	}
	switch pose {
	case rabbitHop:
		tilt, hind, fore := 0.0, 1.0, 1.0
		switch f {
		case 1:
			tilt, hind, fore = -0.2, 0.05, 0.1
		case 2:
			tilt, hind, fore = -0.32, 1, 0.55
		case 3:
			tilt, hind, fore = 0.3, 0.45, 1
		}
		T := func(s shape) shape { return s.turned(9.5, 10, tilt) }
		P := func(x, y float64) (float64, float64) { return turnXY(x, y, 9.5, 10, tilt) }
		part := func(s shape, sh func(shape, float64) shader, b float64) { s = T(s); q.fill(s, sh(s, b)) }
		part(aCap(14.6, 6.3, 11.2, 3.4, 0.9, 0.65), dark, -0.2)                                   // far ear
		part(aCap(4.8, 11, lerpf(6.4, 1.8, hind), lerpf(13.2, 13.4, hind), 1.3, 0.8), dark, -0.2) // far hind leg
		part(aCap(12.4, 11, lerpf(12.8, 15.8, fore), lerpf(13.2, 13.6, fore), 0.9, 0.65), dark, -0.2)
		tl := T(aEll(2.6, 8.7, 1.7, 1.6))
		q.fill(tl, tail(tl))
		body := T(sBlend(1.6, aEll(6.6, 10, 4.3, 3.1), aEll(10.8, 9.7, 4, 2.7)))
		q.fill(body, where(dark(body, 0), pale(body, 0.1), inside(T(aEll(9.6, 11.9, 4.2, 1.5)))))
		hx, hy := lerpf(6.6, 2.4, hind), lerpf(13.6, 14.2, hind)
		part(sBlend(1, aEll(5.9, 10.9, 2.6, 2.3), aCap(5.2, 11.8, hx, hy, 1.4, 0.8)), dark, 0.03) // near haunch and leg
		part(aCap(hx, hy, hx-1.7, hy+0.9, 0.8, 0.6), pale, 0.1)                                   // its long foot
		fx, fy := lerpf(13.6, 16.9, fore), lerpf(13.4, 13.3, fore)
		part(aCap(13.2, 11.2, fx, fy, 1, 0.7), dark, 0.04)
		part(aEll(fx+0.4, fy+0.1, 0.8, 0.6), pale, 0.1)
		head := T(sBlend(0.9, aEll(15.2, 7.9, 2.7, 2.3), aEll(17.2, 8.7, 1.35, 1.1)))
		q.fill(head, where(dark(head, 0.06), pale(head, 0.12), inside(T(aEll(16.2, 9.9, 1.9, 0.9)))))
		part(aCap(14, 6.2, 9.8, 4.4, 1.05, 0.75), dark, 0.03) // near ear
		q.mark(T(aCap(13.2, 5.9, 10.6, 4.7, 0.42, 0.25)), plain(earPink))
		ex, ey := P(15.9, 7.5)
		nx, ny := P(18.3, 8.6)
		face(ex, ey, nx, ny)
		return q.done(0.72, 9, 16)
	case rabbitSleep:
		q.fill(aEll(2.5, 12.9, 1.6, 1.5), tail(aEll(2.5, 12.9, 1.6, 1.5)))
		fe := aCap(12.6, 10.9, 8.2, 10.3, 0.9, 0.7)
		q.fill(fe, dark(fe, -0.2))
		loaf := sBlend(1.8, aEll(7.4, 12.6, 5.2, 3.5), aEll(10.8, 13, 3.4, 3))
		q.fill(loaf, dark(loaf, 0))
		foot := aCap(6.4, 15.6, 11, 15.6, 0.75, 0.6)
		q.fill(foot, pale(foot, 0.1))
		head := sBlend(0.9, aEll(14, 12.5, 2.8, 2.4), aEll(16, 13.4, 1.3, 1.1))
		q.fill(head, where(dark(head, 0.06), pale(head, 0.12), inside(aEll(14.6, 14.9, 2.3, 0.9))))
		ne := aCap(13, 10.5, 8, 9.4, 1.05, 0.8)
		q.fill(ne, dark(ne, 0.04))
		q.mark(aCap(12.2, 10.2, 9, 9.5, 0.42, 0.25), plain(earPink))
		q.flat(aCap(13.4, 12.5, 14.7, 12.6, 0.2, 0.2), plain(furRamp[0])) // a closed eye
		q.flat(aEll(17.1, 13.3, 0.42, 0.36), plain(colNose))
		return q.done(0.72, 9, 16)
	}
	// Sitting up, ears high.
	fe := aCap(11.8, 5.9, 10.7, 1.8, 1, 0.7)
	q.fill(fe, dark(fe, -0.2))
	tl := aEll(2.3, 11.9, 1.8, 1.7)
	q.fill(tl, tail(tl))
	body := sBlend(1.6, aEll(6.6, 12.1, 4.5, 3.8), aEll(9.8, 11.3, 3.6, 3.5), aEll(11.9, 12.4, 1.9, 2.7))
	q.fill(body, where(dark(body, 0), pale(body, 0.1), inside(aEll(12.6, 12.9, 1.9, 2.9))))
	foot := aCap(6.2, 15.4, 10.7, 15.5, 0.85, 0.65)
	q.fill(foot, pale(foot, 0.1))
	hip := aEll(6.5, 12.7, 3.7, 3)
	q.fill(hip, dark(hip, 0.03))
	arm := aCap(12.5, 12.4, 12.9, 15.1, 1, 0.75)
	q.fill(arm, dark(arm, 0.02))
	paw := aEll(13.4, 15.5, 1.1, 0.55)
	q.fill(paw, pale(paw, 0.1))
	head := sBlend(0.9, aEll(13.4, 7.9, 3, 2.7), aEll(15.6, 8.8, 1.4, 1.2))
	q.fill(head, where(dark(head, 0.06), pale(head, 0.12), inside(aEll(14.6, 10.2, 1.9, 0.95))))
	ne := aCap(13.3, 5.8, 12.7, 1.7, 1.25, 0.8)
	q.fill(ne, dark(ne, 0.04))
	q.mark(aCap(13.2, 5.2, 12.8, 2.5, 0.5, 0.28), func(x, y int, _ float64) color.RGBA {
		if float64(y)/cK > 4.3 {
			return earDeep
		}
		return earPink
	})
	face(14.4, 7.5, 16.8, 8.5)
	return q.done(0.72, 9, 16)
}

// --- the worm ------------------------------------------------------------------------

// wormHD is an earthworm crawling: a wave runs down its ringed body.
func wormHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(15, 9, 1, 1)
	ph := float64(pose) * math.Pi
	if f > 0 {
		ph = 2 * math.Pi * cycPh(f)
	}
	var pts []tpt
	for i := 0; i <= 24; i++ {
		t := float64(i) / 24
		x := 1.4 + 9.6*t
		pts = append(pts, tpt{x, 4 + 0.85*math.Sin((x-1.4)*0.95-ph), 0.95 * (0.55 + 0.45*smooth(clamp01(math.Min(t, 1-t)/0.14)))})
	}
	tb := newCTube(pts)
	q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
		u := s * 14
		if u-math.Floor(u) < 0.2 {
			l -= 0.22 // the rings
		}
		col := toneOf(wormRamp, l+0.05)
		if s > 0.58 && s < 0.72 {
			col = mix(col, hex(0xf0a070), 0.45) // the saddle
		}
		return col
	}))
	return q.done(0.6, 6, 5)
}

// --- fish ----------------------------------------------------------------------------

// fishHD is a fish: the plain one of the shallows (look < 0) or one of the
// coloured ones of fishLooks. Its tail beats over the frames.
func fishHD(look, pose, f int) (*canvas, float64, float64) {
	body, fin := fishRamp, finRamp
	var stripes, spots color.RGBA
	cy, ay := 4.5, 6.0
	if look >= 0 {
		lk := fishLooks[look%len(fishLooks)]
		body, fin, stripes, spots = lk.body, lk.fin, lk.stripes, lk.spots
		cy, ay = 5, 7
	}
	q := newCritAt(19, 12, 1, 1)
	ph := float64(pose) * math.Pi
	if f > 0 {
		ph = 2 * math.Pi * cycPh(f)
	}
	sw := 1.1 * math.Sin(ph)
	rays := func(b float64) shader {
		return func(x, y int, _ float64) color.RGBA {
			l := 0.62 + b - float64(y)*0.012
			if (2*x+y)%5 == 0 {
				l -= 0.3
			}
			return toneOf(fin, l)
		}
	}
	q.fill(aPoly(4.9, cy-0.7, 0.5, cy-3+sw, 1.7, cy+sw*0.8, 0.5, cy+3+sw, 4.9, cy+0.7), rays(0)) // the tail
	q.fill(aPoly(6.6, cy-1.8, 9.2, cy-3.8, 10.6, cy-3.6, 11.4, cy-1.6), rays(-0.1))              // back fin
	if look >= 0 {
		q.fill(aPoly(8.8, cy+1.9, 10.2, cy+4.2, 11.6, cy+1.9), rays(0.1))
	}
	bd := sBlend(1.5, aEll(9.3, cy, 5.2, 2.65), aEll(5.4, cy+0.45*math.Sin(ph-1.1), 1.6, 1.1))
	sc := scaly(body, bd, 2.6, 0.02, 3, 3, 31)
	q.fill(bd, func(x, y int, d float64) color.RGBA {
		ax, ny := (float64(x)+0.5)/cK, ((float64(y)+0.5)/cK-cy)/2.65
		nx := (ax - 9.3) / 5.2
		col := sc(x, y, d)
		if ny > 0.38 {
			col = toneOf(body, bd.lum(x, y, 2.6*cK)+0.3) // a pale belly
		}
		if stripes.A > 0 && (math.Abs(nx-0.1) < 0.12 || math.Abs(nx+0.45) < 0.11) {
			col = mix(stripes, colInk, 0.25*clamp01(ny))
		}
		if spots.A > 0 && ny < 0.45 && nx < 0.45 && hashf(x/4, y/4, uint32(look)*31+5) < 0.3 &&
			math.Hypot(float64(x%4)-1.5, float64(y%4)-1.5) < 1.9 {
			col = spots
		}
		if ax > 11.6 && ax < 11.95 && math.Abs(ny) < 0.6 {
			col = mix(col, colInk, 0.35) // the gill
		}
		return col
	})
	q.fill(aPoly(10.6, cy+0.3, 8.6, cy+1.7, 10.9, cy+1.4), rays(0.15)) // the side fin
	q.flat(aEll(13, cy-0.6, 0.8, 0.8), plain(mix(body.at(len(body)-1), colWhite, 0.6)))
	q.bead(13.1, cy-0.6, 1.4)
	return q.done(0.6, 8, ay)
}

// --- the crab ------------------------------------------------------------------------

// crabHD is a crab seen from the front. Frames 1 to cycFrames of crabWalkA
// are its sideways scuttle; those of crabPinch its raised claws snapping.
func crabHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(32, 24, 2, 3)
	cx, cy := 14.0, 11.5
	pinch := pose == crabPinch
	ph, walking := 0.0, false
	switch {
	case pose == crabWalkB:
		ph, walking = 0.5, true
	case pose == crabWalkA && f > 0:
		ph, walking = cycPh(f), true
	}
	snap := 0.0 // how far the raised claws are shut
	if pinch && f > 0 {
		snap = [5]float64{0, 0.5, 1, 0.5, 0}[f]
	}
	// Legs: four a side, a knee lifted high, a pointed foot on the sand.
	for side := -1.0; side <= 1; side += 2 {
		for i := range 4 {
			fi := float64(i)
			dx, up := 0.0, 0.0
			if walking {
				dx, up = pace(ph+0.25*fi+0.25*(side+1), 0.55, 1.5, 1.2)
			}
			bx, by := cx+side*(5.2-0.5*fi), cy-0.4+1.1*fi
			kx, ky := bx+side*(3.4+0.2*fi)+dx*0.5, by-1.5-up
			fx, fy := kx+side*(1.7-0.3*fi)+dx*0.5, by+2.5-up
			lg := aLeg(bx, by, kx, ky, fx, fy, 0.6, 0.5, 0.2)
			q.fill(lg, shd(crabLeg, lg, 0.6, 0.05-0.04*fi))
		}
	}
	// Claws: an arm reaching forward and out, then a big pincer.
	for side := -1.0; side <= 1; side += 2 {
		px, py, ang, gap := cx+side*8.2, cy-5.2, side*0.5, 0.5
		if pinch {
			px, py, ang, gap = cx+side*9, cy-8.2+0.6*snap, side*0.25, 1.15*(1-snap)+0.12
		}
		arm := aLeg(cx+side*4.4, cy-1.4, cx+side*7.4, cy-1.2-1.2*clamp01(-(py-cy)/9), px-side*0.4, py+1.6, 1, 0.85, 0.9)
		q.fill(arm, shd(crabRamp, arm, 1, -0.06))
		// The pincer points up and a little inwards: a broad hand, a fixed
		// finger and a thumb that opens.
		tx, ty := -math.Sin(ang), -math.Cos(ang)
		hand := aEll(px, py, 2.3, 1.9)
		finger := aPoly(px-side*2.1, py-0.6, px-side*1.5+tx*3.9, py-0.4+ty*3.9, px-side*0.3, py-1.2)
		thumb := aPoly(px+side*0.1, py-1.3, px+side*(0.5+gap)+tx*3.6, py+ty*3.6, px+side*2.1, py-0.5)
		claw := sBlend(0.6, hand, finger, thumb)
		q.fill(claw, func(x, y int, d float64) color.RGBA {
			col := toneOf(crabRamp, claw.lum(x, y, 1.5*cK)+0.06)
			if finger.d(float64(x)+0.5, float64(y)+0.5) < -0.2 && thumb.d(float64(x)+0.5, float64(y)+0.5) > 0 && float64(y)/cK < py-2.4 {
				col = mix(col, colBone, 0.55) // the pale tips
			}
			return col
		})
	}
	// The shell: wide and flat, widest towards the front, pale teeth along
	// its front rim, grooves either side of the middle, freckles, and the
	// belly's pale edge below.
	shell := sBlend(1.5, aEll(cx, cy-0.5, 6.4, 3.3), aEll(cx, cy+1.1, 4.7, 2.8))
	q.fill(shell, func(x, y int, d float64) color.RGBA {
		ax, ay := (float64(x)+0.5)/cK, (float64(y)+0.5)/cK
		l := shell.lum(x, y, 2.6*cK) + 0.04
		switch {
		case ay > cy+2.5:
			return toneOf(crabBelly, l+0.1)
		case d > -2.2 && ay < cy-1.6 && (x/2)%2 == 0:
			l += 0.3 // teeth
		case math.Abs(math.Abs(ax-cx)-2.1-0.25*(ay-cy)) < 0.2 && ay > cy-2 && ay < cy+1.2:
			l -= 0.2
		case hashf(x/2, y/2, 611) < 0.1:
			l -= 0.14
		}
		return toneOf(crabRamp, l)
	})
	q.flat(aEll(cx-2.6, cy-2, 1.3, 0.5), plain(crabRamp[5])) // where the sun catches it
	// Eyes on stalks at the front edge.
	for side := -1.0; side <= 1; side += 2 {
		ex := cx + side*3.4
		st := aCap(ex, cy-3.2, ex, cy-5, 0.45, 0.45)
		q.fill(st, shd(crabRamp, st, 0.5, 0.1))
		q.flat(aEll(ex, cy-5.6, 0.85, 0.85), plain(hex(0xf6efe0)))
		q.flat(aEll(ex-side*0.25, cy-5.6, 0.45, 0.5), plain(colInk))
		q.dot(ex, cy-5.6, -1, -2, colWhite)
	}
	return q.done(0.85, cx, cy+3)
}

// --- the snake -----------------------------------------------------------------------

// snakeSkinHD colours the snake's body along its length: sandy scales,
// dark diamonds down the back and, with belly, a pale underside.
func snakeSkinHD(s0, s1, length float64, belly bool) func(x, y int, s, v, l float64) color.RGBA {
	return func(x, y int, s, v, l float64) color.RGBA {
		s = lerpf(s0, s1, s)
		if belly && v > 0.5 {
			if u := s * length * 1.5; u-math.Floor(u) < 0.25 {
				l -= 0.15 // belly plates
			}
			return toneOf(bellyRamp, l+0.2)
		}
		u := s*snakeDiamonds + 0.5
		e := math.Abs(u-math.Floor(u)-0.5)*2.4 + math.Abs(v+0.2)*1.15
		switch {
		case s > 0.06 && e < 0.62:
			return toneOf(blotchRamp, l+0.05)
		case s > 0.06 && e < 0.8:
			return toneOf(snakeRamp, l+0.3) // the pale edge of a diamond
		}
		a, b := s*length*1.4+v*1.3, s*length*1.4-v*1.3
		if a-math.Floor(a) < 0.3 || b-math.Floor(b) < 0.3 {
			l -= 0.13 // the seams between scales
		}
		return toneOf(snakeRamp, l+0.06)
	}
}

// snakeHeadHD draws the head at (hx, hy), turned by ang: a blunt wedge with
// a yellow eye; open bares the fangs, tongue flicks it out.
func snakeHeadHD(q *crit, hx, hy, ang, open float64, tongue bool) {
	T := func(s shape) shape { return s.turned(hx, hy, ang) }
	P := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ang) }
	if tongue {
		tg := T(aCap(hx+4.6, hy+0.5, hx+6.8, hy+0.5, 0.22, 0.2).or(aCap(hx+6.8, hy+0.5, hx+7.9, hy-0.3, 0.2, 0.15)).or(aCap(hx+6.8, hy+0.5, hx+7.9, hy+1.3, 0.2, 0.15)))
		q.fill(tg, plain(colTongue))
	}
	if open > 0 {
		jaw := T(aCap(hx+0.4, hy+0.9, hx+3.9, hy+1.2+1.9*open, 0.85, 0.5))
		q.fill(jaw, shd(bellyRamp, jaw, 0.8, 0))
		q.fill(T(aPoly(hx+0.6, hy+0.2, hx+4.4, hy-0.6-0.8*open, hx+3.6, hy+0.9+1.7*open)), plain(hex(0xb2364a)))
		hy -= 0.9 * open
	}
	head := T(sBlend(0.8, aEll(hx+1.1, hy, 2.3, 1.7), aEll(hx+3.4, hy+0.25, 1.7, 1.15)))
	q.fill(head, func(x, y int, d float64) color.RGBA {
		l := head.lum(x, y, 1.6*cK)
		ax, ay := turnXY((float64(x)+0.5)/cK, (float64(y)+0.5)/cK, hx, hy, -ang)
		switch {
		case ay > hy+0.75:
			return toneOf(bellyRamp, l+0.2)
		case ay < hy-0.6 && ax < hx+2.6 && math.Abs(ax-hx-1.1) > 0.35:
			return toneOf(blotchRamp, l+0.1) // the dark crown
		}
		return toneOf(snakeRamp, l+0.08)
	})
	if open > 0 {
		for _, fx := range [2]float64{3.3, 4.3} {
			q.fill(T(aPoly(hx+fx-0.35, hy+1, hx+fx+0.35, hy+1, hx+fx+0.1, hy+1.2+1.3*open)), plain(colFang))
		}
	}
	ex, ey := P(hx+2.3, hy-0.45)
	q.flat(aEll(ex, ey, 0.75, 0.7), plain(mix(snakeEye, colInk, 0.8)))
	q.flat(aEll(ex, ey, 0.55, 0.5), plain(snakeEye))
	q.flat(aEll(ex, ey, 0.17, 0.5), plain(colInk))
	q.dot(ex, ey, -1, -1, colWhite)
	nx, ny := P(hx+4.6, hy-0.1)
	q.dot(nx, ny, 0, 0, blotchRamp[0])
}

// snakeHD draws the snake: slithering (frames 1 to cycFrames of pose 0 are
// the wave running down its body), striking (wound back, then thrown), and
// coiled at rest.
func snakeHD(pose, f int) (*canvas, float64, float64) {
	if pose == snakeCoil || pose == snakeCoilLik {
		return snakeCoilHD(pose == snakeCoilLik)
	}
	q := newCritAt(53, 28, 2, 3)
	const neckX, neckY = 34.0, 13.0
	ph := float64(pose) * 2 * math.Pi / snakeSlither
	if pose == 0 && f > 0 {
		ph = 2 * math.Pi * cycPh(f)
	}
	strike := pose == snakeStrike
	amt := 0.0
	if strike {
		ph, amt = 0, blowAmt(f)
	}
	sp := serpenoid(32, 1.3, 1.15, ph, neckX, neckY)
	hx, hy, ang, open := neckX-0.5, neckY, 0.0, 0.0
	if strike { // the fore-body rears up; it draws back in an S, then the head lunges
		rear := lerpf(7.5, 6, clamp01(amt))
		reach := lerpf(-4.5, 2.2, (amt+1)/2)
		for i := range sp {
			sp[i].y -= rear * smooth(clamp01((sp[i].s-0.55)/0.45))
			sp[i].x += reach * smooth(clamp01((sp[i].s-0.7)/0.3))
		}
		hx, hy = neckX-0.5+reach, neckY-rear
		ang, open = -0.25+0.25*clamp01(amt), clamp01((amt+0.6)/1.2)
	}
	var pts []tpt
	for i := 0; i < len(sp); i += 4 {
		pts = append(pts, tpt{sp[i].x, sp[i].y, sp[i].r * 1.02})
	}
	tb := newCTube(pts)
	q.fill(tb.shape(), tb.skin(snakeSkinHD(0, 1, 32, true)))
	snakeHeadHD(q, hx, hy, ang, open, (strike && amt < 0) || pose == 1 || f == 3 || f == 4)
	return q.done(0.8, 20, 14)
}

func snakeCoilHD(tongue bool) (*canvas, float64, float64) {
	q := newCritAt(34, 26, 2, 1)
	const cx, cy = 14.0, 16.0
	const turns, inner, outer = 1.3, 2.6, 9.2
	const coilEnd = 0.86
	spiral := func(s float64) (float64, float64) {
		ang := 0.3 + 2*math.Pi*turns*s
		rad := outer - (outer-inner)*s/coilEnd
		return cx + rad*math.Cos(ang), cy + rad*math.Sin(ang)*0.72 - 1.6*s
	}
	at := func(s float64) tpt {
		if s < coilEnd {
			x, y := spiral(s)
			return tpt{x, y, snakeWidth(s) * 0.9}
		}
		u := (s - coilEnd) / (1 - coilEnd)
		x0, y0 := spiral(coilEnd)
		return tpt{lerpf(x0, cx+2, u), lerpf(y0, cy-7.5, smooth(u)), snakeWidth(s) * 0.9}
	}
	// In stretches, tail first, so each turn lies over the one before it
	// with a line between them.
	const parts = 7
	for k := range parts {
		s0, s1 := float64(k)/parts, math.Min(1, float64(k+1)/parts+0.02)
		var pts []tpt
		for i := 0; i <= 12; i++ {
			pts = append(pts, at(lerpf(s0, s1, float64(i)/12)))
		}
		tb := newCTube(pts)
		q.fill(tb.shape(), tb.skin(snakeSkinHD(s0, s1, 40, false)))
	}
	snakeHeadHD(q, cx+1.4, cy-8.2, 0, 0, tongue)
	return q.done(0.8, cx, cy+3)
}
