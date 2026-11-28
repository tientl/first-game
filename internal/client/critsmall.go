package client

import (
	"image/color"
	"math"
)

// The small animals at HD density, painted (critpaint.go): the rabbit, the
// worm, the fish and the snake (the crab is in critcrab.go). See
// critterhd.go for the tools and the frames.

func init() {
	set := func(name string, hd func(pose, f int) (*canvas, float64, float64), anim map[int]int) {
		critKinds[name].hd, critKinds[name].anim = hd, anim
	}
	set("rabbit", rabbitHD, map[int]int{rabbitHop: cycFrames})
	set("worm", wormHD, map[int]int{0: cycFrames})
	set("fish", func(p, f int) (*canvas, float64, float64) { return fishHD(-1, p, f) }, map[int]int{0: cycFrames})
	for v := range fishLooks {
		set(fishKind(v), func(p, f int) (*canvas, float64, float64) { return fishHD(v, p, f) }, map[int]int{0: cycFrames})
	}
	set("crab", crabHD, map[int]int{crabWalkA: cycFrames, crabPinch: 4})
	set("snake", snakeHD, map[int]int{0: cycFrames, snakeStrike: blowFrames})
}

func fishKind(v int) string { return "fish" + string(rune('0'+v)) }

// How big the small animals are drawn against their designs, so they stand
// beside the player (playerArtH, 93 px at 1080p) at about their size: the
// rabbit sitting up with its ears high comes to the player's knee (0.3 of
// the player), the crab with its claws up to the shin (0.25), the coiled
// snake's raised head to the knee (0.35), a monkey sitting to the hip
// (0.4). A little bigger than life, so they still read at a glance.
// `shot -scalesheet -scaleonly small` shows them.
const (
	rabbitSize = 0.65
	crabSize   = 0.65
	snakeSize  = 0.82
	monkeySize = 0.72
)

// --- the rabbit --------------------------------------------------------------------

var (
	rabFur   = newSurf(furRamp, sFur, 0, 77).with(3.6, furFlow)
	rabPale  = newSurf(paleRamp, sFur, 0, 78).with(3.2, furFlow)
	rabTail  = newSurf(tailRamp, sFur, 0, 79).with(3, -math.Pi/2)
	rabEarIn = newSurf(mkRamp(0x8a4a4c, 0xb96a6c, 0xe59a94, 0xf4c0b6), sMatte, 0.15, 80)
	rabFoot  = newSurf(paleRamp, sFur, 0, 81).with(3, 0)
)

// hopKey is the leap at one moment: the body's tilt (+ nose down), how far
// the hind legs reach back and the fore legs forward (0 tucked under, 1
// out), how stretched the body is, and the ears swept back.
type hopKey struct{ ph, tilt, hind, fore, long, ears float64 }

var hopKeys = []hopKey{
	{0, -0.12, 0.05, 0.15, 0, 0.2},     // gathered, the hind feet under the chest
	{0.22, -0.34, 1, 0.5, 0.6, 0.6},    // the push: nose up, hind legs driving back
	{0.48, -0.02, 1, 1, 1, 1},          // stretched out in the air
	{0.7, 0.3, 0.45, 0.95, 0.55, 0.85}, // landing on the fore paws, nose down
	{0.86, 0.12, 0.08, 0.35, 0.15, 0.4}, // the hind legs swinging through
	{1, -0.12, 0.05, 0.15, 0, 0.2},
}

// hopAt is the leap at phase ph.
func hopAt(ph float64) hopKey {
	ph -= math.Floor(ph)
	for i := 1; i < len(hopKeys); i++ {
		a, b := hopKeys[i-1], hopKeys[i]
		if ph <= b.ph {
			t := smooth((ph - a.ph) / (b.ph - a.ph))
			return hopKey{ph, lerpf(a.tilt, b.tilt, t), lerpf(a.hind, b.hind, t), lerpf(a.fore, b.fore, t), lerpf(a.long, b.long, t), lerpf(a.ears, b.ears, t)}
		}
	}
	return hopKeys[0]
}

// rabbitHD draws the rabbit: sitting up, asleep in a loaf, and leaping:
// frame 0 of rabbitHop stretched out level, frames 1 to cycFrames a whole
// bound, gathered, pushing off, stretched out, landing on the fore paws and
// the hind feet swinging through.
func rabbitHD(pose, f int) (*canvas, float64, float64) {
	q := newCritZoom(24, 20, 2, 2, rabbitSize)
	face := func(ex, ey, nx, ny, ang float64) {
		// a pale ring round the eye, the eye big and dark with a glint, the
		// whisker pad, the pink nose and the whiskers
		q.coat(aEll(ex, ey, 1.05, 0.95), rabPale)
		q.flat(aEll(ex, ey, 0.66, 0.66), plain(hex(0x1c120e)))
		q.flat(aEll(ex+0.12, ey+0.18, 0.4, 0.36), plain(hex(0x3e2618)))
		q.dot(ex-0.25, ey-0.25, 0, 0, colWhite)
		q.dot(ex-0.25, ey-0.25, 1, 0, hex(0xd8d0c8))
		q.flat(aEll(nx, ny, 0.4, 0.32).turned(nx, ny, ang), plain(colNose))
		q.dot(nx, ny, -1, -1, hex(0xf2a8a8))
		for k := 0.0; k < 2; k++ { // whiskers
			for t := 1.5; t < 4; t++ {
				wx, wy := turnXY(nx-0.6+t*0.42, ny+0.4+k*0.3+t*0.06*(k-1), nx, ny, ang)
				q.dot(wx, wy, 0, 0, mix(paleRamp[2], colWhite, 0.3))
			}
		}
	}
	ear := func(s shape, in shape, far bool) {
		b := 0.04
		if far {
			b = -0.1
		}
		q.body(s, rabFur, look{round: 0.8, bias: b, far: far})
		if !far {
			q.coat(in, rabEarIn)
			q.shade(in.grow(-0.35*cK), -0.08) // deeper down the middle
		}
		// the dark rim at the tip
		q.shade(s.and(aEll(s.x0/cK+0.6, s.y0/cK+0.6, 1.2, 1.2)), -0.14)
	}
	switch pose {
	case rabbitHop:
		k := hopKeys[2] // frame 0: stretched out level
		if f > 0 {
			k = hopAt(cycPh(f))
		}
		tilt := k.tilt
		T := func(s shape) shape { return s.turned(9.5, 10, tilt) }
		P := func(x, y float64) (float64, float64) { return turnXY(x, y, 9.5, 10, tilt) }
		long := k.long
		bx := lerpf(7.4, 6.4, long) // the rump: further back when stretched
		hx, hy := lerpf(8.2, 2.4, k.hind), lerpf(15, 14.2, k.hind)
		fx, fy := lerpf(12.4, 16.9, k.fore), lerpf(14.6, 13.3, k.fore)
		ea := k.ears
		// far ear, far hind leg and far fore leg, in the body's shade
		ear(T(aCap(14.4, 6.4, lerpf(12.4, 10.8, ea), lerpf(2.6, 4.2, ea), 0.85, 0.6)), shape{}, true)
		q.body(T(aCap(bx-1.4, 11, hx+0.6, hy-0.6, 1.2, 0.75)), rabFur, look{round: 0.9, far: true})
		q.body(T(aCap(12.4, 11, fx-0.4, fy, 0.85, 0.6)), rabFur, look{round: 0.7, far: true})
		// the puff of a tail
		tl := T(aEll(bx-4.4, 8.9, 1.7, 1.6)).furry(0.4, 0.9, 79)
		q.body(tl, rabTail, look{round: 1.2, bias: 0.06})
		// the body, a pale belly under it
		body := T(sBlend(1.8, aEll(bx, 10, 4.4, 3.2), aEll(10.9, 9.8, 4, 2.8))).furry(0.35, 1.2, 77)
		q.body(body, rabFur, look{round: 2.6})
		q.coat(T(aEll(9.8, 12.2, 4.3, 1.5)), rabPale)
		// the near haunch and its long foot
		q.body(T(sBlend(1, aEll(bx-0.6, 10.9, 2.8, 2.5), aCap(bx-1.2, 11.8, hx, hy, 1.4, 0.8))), rabFur, look{round: 1.6, bias: 0.04})
		q.body(T(aCap(hx, hy, hx-1.9, hy+0.9, 0.8, 0.55)), rabFoot, look{round: 0.6, bias: 0.06})
		// the near fore leg and paw
		q.body(T(aCap(13.2, 11.2, fx, fy, 0.95, 0.68)), rabFur, look{round: 0.8, bias: 0.03})
		q.body(T(aEll(fx+0.4, fy+0.1, 0.8, 0.6)), rabFoot, look{round: 0.6, bias: 0.06})
		// the head and muzzle, a pale chin
		head := T(sBlend(0.9, aEll(15.2, 7.9, 2.75, 2.35), aEll(17.2, 8.7, 1.4, 1.15))).furry(0.25, 0.9, 82)
		q.body(head, rabFur, look{round: 1.8, bias: 0.04})
		q.coat(T(aEll(16.4, 9.9, 1.9, 0.9)), rabPale)
		q.shade(T(aEll(14.2, 9, 1, 0.8)), 0.05) // the cheek
		ear(T(aCap(14, 6.2, lerpf(11.6, 9.8, ea), lerpf(1.9, 4.4, ea), 1.08, 0.78)), T(aCap(13.4, 6, lerpf(12, 10.4, ea), lerpf(2.6, 4.7, ea), 0.48, 0.3)), false)
		ex, ey := P(15.9, 7.5)
		nx, ny := P(18.3, 8.6)
		face(ex, ey, nx, ny, tilt)
		return q.done(0.72, 9, 16)
	case rabbitSleep:
		tl := aEll(2.5, 12.9, 1.6, 1.5).furry(0.4, 0.9, 79)
		q.body(tl, rabTail, look{round: 1.1})
		ear(aCap(12.6, 10.9, 8.2, 10.3, 0.9, 0.7), shape{}, true)
		loaf := sBlend(1.8, aEll(7.4, 12.6, 5.2, 3.5), aEll(10.8, 13, 3.4, 3)).furry(0.35, 1.2, 77)
		q.body(loaf, rabFur, look{round: 2.8})
		q.body(aCap(6.4, 15.6, 11, 15.6, 0.75, 0.6), rabFoot, look{round: 0.6})
		head := sBlend(0.9, aEll(14, 12.5, 2.8, 2.4), aEll(16, 13.4, 1.3, 1.1)).furry(0.25, 0.9, 82)
		q.body(head, rabFur, look{round: 1.8, bias: 0.04})
		q.coat(aEll(14.6, 14.9, 2.3, 0.9), rabPale)
		ear(aCap(13, 10.5, 8, 9.4, 1.05, 0.8), aCap(12.2, 10.2, 9, 9.5, 0.42, 0.25), false)
		q.flat(aPath(13.3, 12.4, 0.16, 14, 12.75, 0.18, 14.8, 12.5, 0.16), plain(hex(0x3a2618))) // a closed eye
		q.flat(aEll(17.1, 13.3, 0.42, 0.36), plain(colNose))
		return q.done(0.72, 9, 16)
	}
	// Sitting up, ears high.
	ear(aCap(11.8, 5.9, 10.7, 1.8, 1, 0.7), shape{}, true)
	tl := aEll(2.3, 11.9, 1.8, 1.7).furry(0.4, 0.9, 79)
	q.body(tl, rabTail, look{round: 1.2, bias: 0.06})
	body := sBlend(1.6, aEll(6.6, 12.1, 4.5, 3.8), aEll(9.8, 11.3, 3.6, 3.5), aEll(11.9, 12.4, 1.9, 2.7)).furry(0.35, 1.2, 77)
	q.body(body, rabFur, look{round: 3})
	q.coat(aEll(12.6, 12.9, 1.9, 2.9), rabPale)
	q.body(aCap(6.2, 15.4, 10.7, 15.5, 0.85, 0.65), rabFoot, look{round: 0.7, bias: 0.04})
	q.body(aEll(6.5, 12.7, 3.7, 3).furry(0.25, 1, 83), rabFur, look{round: 2.4, bias: 0.04}) // the haunch
	q.body(aCap(12.5, 12.4, 12.9, 15.1, 1, 0.75), rabFur, look{round: 0.8, bias: 0.02})
	q.body(aEll(13.4, 15.5, 1.1, 0.55), rabFoot, look{round: 0.5, bias: 0.06})
	head := sBlend(0.9, aEll(13.4, 7.9, 3, 2.7), aEll(15.6, 8.8, 1.4, 1.2)).furry(0.25, 0.9, 82)
	q.body(head, rabFur, look{round: 2, bias: 0.04})
	q.coat(aEll(14.6, 10.2, 1.9, 0.95), rabPale)
	q.shade(aEll(12.6, 9, 1.1, 0.9), 0.05) // the cheek
	ear(aCap(13.3, 5.8, 12.7, 1.7, 1.25, 0.8), aCap(13.2, 5.2, 12.8, 2.5, 0.52, 0.3), false)
	face(14.4, 7.5, 16.8, 8.5, 0)
	return q.done(0.72, 9, 16)
}

// --- the worm ------------------------------------------------------------------------

var (
	wormSkin   = newSurf(mkRamp(0x6a463a, 0x8c5e52, 0xac7a6a, 0xc69686, 0xdcb2a2), sWet, 0.7, 501)
	wormSaddle = newSurf(mkRamp(0x6a4038, 0x925e52, 0xb07c6c, 0xc8988a, 0xdcb4a4), sWet, 0.6, 502)
)

// wormHD is an earthworm crawling, small, its tail still in the soil: a
// wave runs down its ringed, glistening body, out of a crumb of dark earth.
func wormHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(12, 7, 1, 1)
	ph := float64(pose) * math.Pi
	if f > 0 {
		ph = 2 * math.Pi * cycPh(f)
	}
	// The soil it comes out of, under its tail.
	q.flat(aEll(2.1, 3.9, 1.6, 0.8), plain(hex(0x2a2116)))
	q.flat(aEll(1.7, 3.6, 0.9, 0.4), plain(hex(0x3d3020)))
	q.dot(2.8, 3.7, 0, 0, hex(0x4a3a26))
	var pts []tpt
	for i := 0; i <= 20; i++ {
		t := float64(i) / 20
		x := 1.6 + 6.4*t
		pts = append(pts, tpt{x, 3.4 + 0.6*math.Sin((x-1.6)*1.2-ph), 0.62 * (0.55 + 0.45*smooth(clamp01(math.Min(t+0.3, 1-t)/0.16)))})
	}
	tb := newCTube(pts)
	q.body(tb.shape(), wormSkin, look{round: 0.6, bias: 0.12})
	q.coatBy(tb.shape(), func(x, y float64) *surf {
		if _, s, _ := tb.at(x, y); s > 0.6 && s < 0.74 {
			return wormSaddle
		}
		return nil
	})
	// the rings, and the soil still on it where it leaves the ground
	ring := shape{tb.shape().x0, tb.shape().y0, tb.shape().x1, tb.shape().y1, func(x, y float64) float64 {
		_, s, _ := tb.at(x/cK, y/cK)
		u := s * 12
		if u-math.Floor(u) < 0.2 {
			return -1
		}
		return 1
	}}
	q.shade(ring, -0.12)
	q.shade(tb.shape().and(aEll(1.8, 3.4, 1.4, 1.4)), -0.2)
	return q.done(0.35, 5, 4.2)
}

// --- fish ----------------------------------------------------------------------------

// fishHD is a fish: the plain one of the shallows (look < 0) or one of the
// coloured ones of fishLooks. Its body bends and its tail beats over the
// frames; its scales shine.
func fishHD(kind, pose, f int) (*canvas, float64, float64) {
	body, fin := fishRamp, finRamp
	var stripes, spots color.RGBA
	cy, ay := 4.5, 6.0
	seed := uint32(600)
	if kind >= 0 {
		lk := fishLooks[kind%len(fishLooks)]
		body, fin, stripes, spots = lk.body, lk.fin, lk.stripes, lk.spots
		cy, ay = 5, 7
		seed += uint32(kind) * 10
	}
	skin := newSurf(body, sScale, 0.9, seed).with(2.6, 0)
	belly := newSurf(ramp{mix(body[len(body)/2], colWhite, 0.3), mix(body[len(body)-1], colWhite, 0.5), mix(body[len(body)-1], colWhite, 0.8), colWhite}, sScale, 1, seed+1).with(2.4, 0)
	finS := newSurf(fin, sMatte, 0.3, seed+2)
	q := newCritAt(19, 12, 1, 1)
	ph := float64(pose) * math.Pi
	if f > 0 {
		ph = 2 * math.Pi * cycPh(f)
	}
	sw := 1.1 * math.Sin(ph)
	bend := 0.45 * math.Sin(ph-1.1) // the rear of the body swings with the tail
	rays := func(s shape, b float64, far bool) {
		q.body(s, finS, look{round: 0.5, bias: b, far: far, flat: 0.6, soft: true})
		// the rays: darker lines fanning from the root
		q.shade(s.and(shape{s.x0, s.y0, s.x1, s.y1, func(x, y float64) float64 {
			if (int(x)*2+int(y))%5 == 0 {
				return -1
			}
			return 1
		}}), -0.12)
	}
	// the far pectoral fin behind, then the tail, the back fin and the belly fins
	rays(aPoly(10.4, cy+0.6, 8.8, cy+2, 10.6, cy+1.7), -0.05, true)
	tail := aPoly(4.9, cy-0.7+bend*0.4, 0.5, cy-3.1+sw, 1.9, cy+sw*0.8, 0.5, cy+3.1+sw, 4.9, cy+0.7+bend*0.4)
	rays(tail, 0, false)
	q.shade(tail.and(aEll(0.9, cy+sw*0.8, 1.2, 3.4)), -0.1) // darker at its fork
	rays(aPoly(6.4, cy-1.8, 8.8, cy-3.9, 10.2, cy-3.8, 11.4, cy-1.6), -0.05, false)
	rays(aPoly(6, cy+1.6, 7.4, cy+3, 8.6, cy+1.8), 0, false) // the anal fin
	if kind >= 0 {
		rays(aPoly(9, cy+1.9, 10.3, cy+4.1, 11.6, cy+1.9), 0.05, false)
	}
	bd := sBlend(1.5, aEll(9.3, cy, 5.2, 2.65), aEll(5.4, cy+bend, 1.6, 1.1))
	q.body(bd, skin, look{round: 2.2})
	q.coat(bd.and(sBelow((cy+0.95)*cK)), belly)
	q.shade(bd.and(sAbove((cy-1.2)*cK)), -0.08) // the darker back
	if stripes.A > 0 {
		st := newSurf(ramp{mix(stripes, colInk, 0.5), mix(stripes, colInk, 0.25), stripes, mix(stripes, colWhite, 0.5)}, sScale, 0.9, seed+3).with(2.6, 0)
		q.coat(bd.and(aCap(10.2, cy-3, 10.7, cy+3, 0.55, 0.55).or(aCap(6.9, cy-3, 7.3, cy+3, 0.5, 0.5))), st)
	}
	if spots.A > 0 {
		sp := newSurf(ramp{mix(spots, colInk, 0.4), spots, mix(spots, colWhite, 0.25), mix(spots, colWhite, 0.5)}, sScale, 0.6, seed+4).with(2.6, 0)
		for k := 0; k < 7; k++ {
			sx, sy := 5.5+hashf(k, 1, seed)*7, cy-1.6+hashf(k, 2, seed)*2.2
			q.coat(bd.and(aEll(sx, sy, 0.6, 0.55)), sp)
		}
	}
	// the lateral line, the gill cover and the mouth
	q.shade(bd.and(aPath(5, cy+0.1+bend*0.5, 0.12, 9, cy-0.3, 0.12, 12, cy-0.2, 0.12)), 0.1)
	q.shade(bd.and(aEll(13.1, cy+0.1, 1.9, 2.4)).sub(aEll(13.5, cy+0.1, 1.7, 2.4)), -0.2)
	q.shade(aCap(13.8, cy+0.9, 14.5, cy+0.7, 0.13, 0.12), -0.35)
	rays(aPoly(11.2, cy+0.4, 9.4, cy+1.9, 11.5, cy+1.5), 0.1, false) // the near side fin
	// the eye: a gold ring round a big dark pupil
	q.flat(aEll(13, cy-0.6, 0.9, 0.9), plain(mix(body[0], colInk, 0.3)))
	q.flat(aEll(13, cy-0.6, 0.72, 0.72), plain(hex(0xd8b450)))
	q.flat(aEll(13.06, cy-0.56, 0.46, 0.48), plain(colInk))
	q.dot(12.8, cy-0.85, 0, 0, colWhite)
	return q.done(0.6, 8, ay)
}

// --- the snake -----------------------------------------------------------------------

var (
	snakeScaleS = newSurf(snakeRamp, sScale, 0.7, 701).with(3.2, 0)
	snakeBlotch = newSurf(blotchRamp, sScale, 0.6, 702).with(3.2, 0)
	snakeEdge   = newSurf(mkRamp(0x8a6a2a, 0xc4a050, 0xe8cc78, 0xf8e6a8), sScale, 0.7, 703).with(3.2, 0)
	snakeBelly  = newSurf(bellyRamp, sPlate, 0.5, 704)
)

// snakeSkin paints the tube tb, which runs from s0 to s1 of the snake's
// length: sandy scales, dark diamonds edged pale down the back and, with
// belly, a pale underside in plates.
func snakeSkin(q *crit, tb *ctube, s0, s1, length float64, belly bool) {
	sh := tb.shape()
	// The scales lie in rows along the body, round its bends.
	uv := func(x, y float64) (float64, float64) {
		_, s, v := tb.at(x/cK, y/cK)
		return lerpf(s0, s1, s) * length * cK, (v + 1) * 1.7 * cK
	}
	along := func(s *surf) *surf { c := *s; c.uv = uv; return &c }
	scales, blotch, edge := along(snakeScaleS), along(snakeBlotch), along(snakeEdge)
	q.body(sh, scales, look{round: 1.1})
	q.coatBy(sh, func(x, y float64) *surf {
		_, s, v := tb.at(x, y)
		s = lerpf(s0, s1, s)
		if belly && v > 0.5 {
			return snakeBelly
		}
		u := s*snakeDiamonds + 0.5
		e := math.Abs(u-math.Floor(u)-0.5)*2.4 + math.Abs(v+0.2)*1.15
		switch {
		case s > 0.06 && e < 0.62:
			return blotch
		case s > 0.06 && e < 0.82:
			return edge
		}
		return nil
	})
	if belly {
		q.shade(shape{sh.x0, sh.y0, sh.x1, sh.y1, func(x, y float64) float64 {
			_, s, v := tb.at(x/cK, y/cK)
			if u := lerpf(s0, s1, s) * length * 1.5; v > 0.5 && u-math.Floor(u) < 0.22 {
				return -1
			}
			return 1
		}}, -0.14)
	}
}

// snakeHeadHD draws the head at (hx, hy), turned by ang: a blunt wedge with
// a dark crown, a golden eye slit down the middle under a heavy brow; open
// bares the fangs, tongue flicks it out.
func snakeHeadHD(q *crit, hx, hy, ang, open float64, tongue bool) {
	T := func(s shape) shape { return s.turned(hx, hy, ang) }
	P := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ang) }
	if tongue {
		tg := T(aCap(hx+4.6, hy+0.5, hx+6.8, hy+0.5, 0.22, 0.2).or(aCap(hx+6.8, hy+0.5, hx+7.9, hy-0.3, 0.2, 0.15)).or(aCap(hx+6.8, hy+0.5, hx+7.9, hy+1.3, 0.2, 0.15)))
		q.fill(tg, plain(colTongue))
	}
	if open > 0 {
		jaw := T(aCap(hx+0.4, hy+0.9, hx+3.9, hy+1.2+1.9*open, 0.85, 0.5))
		q.body(jaw, snakeBelly, look{round: 0.7})
		q.fill(T(aPoly(hx+0.6, hy+0.2, hx+4.4, hy-0.6-0.8*open, hx+3.6, hy+0.9+1.7*open)), func(x, y int, _ float64) color.RGBA {
			return mix(hex(0x7a1c30), hex(0xc24a5a), clamp01(((float64(x)+0.5)/cK-hx)/4))
		})
		hy -= 0.9 * open
	}
	head := T(sBlend(0.8, aEll(hx+1.1, hy, 2.3, 1.7), aEll(hx+3.4, hy+0.25, 1.7, 1.15)))
	q.body(head, snakeScaleS.with(2.4, 0), look{round: 1.3, bias: 0.03})
	q.coatBy(head, func(x, y float64) *surf {
		ax, ay := turnXY(x, y, hx, hy, -ang)
		switch {
		case ay > hy+0.75:
			return snakeBelly
		case ay < hy-0.6 && ax < hx+2.6 && math.Abs(ax-hx-1.1) > 0.35:
			return snakeBlotch // the dark crown
		}
		return nil
	})
	q.shade(T(aEll(hx+2.3, hy-1.05, 1.1, 0.4)), 0.12) // the brow ridge over the eye
	if open > 0 {
		for _, fx := range [2]float64{3.3, 4.3} {
			q.fill(T(aPoly(hx+fx-0.35, hy+1, hx+fx+0.35, hy+1, hx+fx+0.1, hy+1.2+1.3*open)), plain(colFang))
		}
	}
	ex, ey := P(hx+2.3, hy-0.45)
	q.flat(aEll(ex, ey, 0.78, 0.72), plain(mix(snakeEye, colInk, 0.82)))
	q.flat(aEll(ex, ey, 0.6, 0.54), func(x, y int, _ float64) color.RGBA {
		return mix(mix(snakeEye, colInk, 0.3), mix(snakeEye, colWhite, 0.2), clamp01(((float64(y)+0.5)/cK-ey+0.5)))
	})
	q.flat(aEll(ex+0.05, ey, 0.15, 0.5), plain(colInk))
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
	q := newCritZoom(53, 28, 2, 3, snakeSize)
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
	snakeSkin(q, newCTube(pts), 0, 1, 32, true)
	snakeHeadHD(q, hx, hy, ang, open, (strike && amt < 0) || pose == 1 || f == 3 || f == 4)
	return q.done(0.8, 20, 14)
}

func snakeCoilHD(tongue bool) (*canvas, float64, float64) {
	q := newCritZoom(34, 26, 2, 1, snakeSize)
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
	// with a crease and a shadow between them.
	const parts = 7
	for k := range parts {
		s0, s1 := float64(k)/parts, math.Min(1, float64(k+1)/parts+0.02)
		var pts []tpt
		for i := 0; i <= 12; i++ {
			pts = append(pts, at(lerpf(s0, s1, float64(i)/12)))
		}
		snakeSkin(q, newCTube(pts), s0, s1, 40, false)
	}
	snakeHeadHD(q, cx+1.4, cy-8.2, 0, 0, tongue)
	return q.done(0.8, cx, cy+3)
}
