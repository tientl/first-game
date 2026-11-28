package client

import (
	"math"
)

// The monkey at HD density, painted (critpaint.go): a long-tailed monkey of
// the woods in a coat of warm brown locks, paler on the chest and the inside
// of the limbs, with a bare, pinkish-tan face under a heavy brow, bright
// beads of eyes, a short muzzle, round bare ears and leathery hands and
// feet. Sitting (and eating a banana), on all fours, and in a bound.
// Drawn monkeySize times its design.

var (
	monkFur  = newSurf(monkeyRamp, sFur, 0, 611).with(4, furFlow)
	monkPale = newSurf(mkRamp(0x4a2c16, 0x6e4626, 0x8e6238, 0xae8050, 0xc8a070), sFur, 0, 612).with(3.4, furFlow)
	monkTail = newSurf(monkeyRamp, sFur, 0, 613).with(3, 0)
	monkSkin = newSurf(monkeyFace, sMatte, 0.25, 614)
	monkHand = newSurf(mkRamp(0x3a2418, 0x5a3a28, 0x7a5440, 0x9a7058), sMatte, 0.15, 615)
	monkNana = newSurf(bananaRamp, sMatte, 0.35, 616)
)

func monkeyHD(pose, f int) (*canvas, float64, float64) {
	q := newCritZoom(30, 27, 2, 2, monkeySize)
	// limb is an arm or a leg from (x0, y0) to a hand or a foot at (x1, y1),
	// the elbow or the knee bent to bend; far ones lie in the body's shade.
	limb := func(x0, y0, x1, y1, bend float64, far bool) {
		kx, ky := ik2(x0, y0, x1, y1, 3.3, 3.3, bend)
		b := 0.03
		if far {
			b = -0.04
		}
		lg := aLeg(x0, y0, kx, ky, x1, y1, 1.3, 1, 0.8).furry(0.18, 0.8, 621)
		q.body(lg, monkFur, look{round: 0.9, bias: b, far: far})
		hand := sBlend(0.4, aEll(x1+0.5, y1+0.1, 1.1, 0.72), aCap(x1+0.9, y1+0.3, x1+1.7, y1+0.5, 0.35, 0.28))
		q.body(hand, monkHand, look{round: 0.6, bias: b, far: far})
	}
	tail := func(pts [4][2]float64) {
		tb := newCTube(bez(14, 0.9, 0.55, pts[0], pts[1], pts[2], pts[3]))
		q.body(tb.shape().furry(0.14, 0.7, 622), monkTail, look{round: 0.8, bias: -0.06})
	}
	head := func(hx, hy float64) {
		// The ear, bare and round, behind the jaw.
		ear := aEll(hx-3.4, hy+0.2, 1.35, 1.55)
		q.body(ear, monkSkin, look{round: 0.9, bias: -0.04})
		q.shade(aEll(hx-3.3, hy+0.3, 0.6, 0.8), -0.18)
		// The skull in its cap of fur, the ruff round the face.
		sk := sBlend(1, aEll(hx-0.3, hy-0.2, 4, 3.8), aEll(hx+0.4, hy+1.8, 3.6, 2.6)).furry(0.3, 0.8, 623)
		q.body(sk, monkFur, look{round: 2.4, bias: 0.06})
		// The bare face: a heart shape round the eyes, a short muzzle.
		fc := sBlend(1, aEll(hx+1.3, hy-0.3, 2.3, 1.9), aEll(hx+2.1, hy+1.6, 2.2, 1.6))
		q.body(fc, monkSkin, look{round: 1.3, bias: 0.06})
		q.shade(aCap(hx+0.2, hy-1.6, hx+3.2, hy-1.5, 0.5, 0.45), 0.12)     // the brow ridge, lit
		q.shade(aCap(hx+0.4, hy-0.8, hx+3, hy-0.7, 0.35, 0.3), -0.16)     // and its shadow over the eyes
		q.shade(aEll(hx+2.9, hy+1.3, 1.1, 0.8), 0.07)                      // the muzzle's top
		q.shade(aCap(hx+1.2, hy+2.6, hx+3.4, hy+2.5, 0.3, 0.25), -0.12)   // under the lip
		q.glassEye(hx+1.65, hy-0.45, 0.62, hex(0x5a3416), false)
		q.dot(hx+3.75, hy+0.9, 0, 0, monkeyFace[0]) // the nostril
		q.flat(aCap(hx+2.3, hy+2.15, hx+3.7, hy+2, 0.16, 0.14), plain(mix(monkeyFace[0], colInk, 0.3)))
	}
	switch pose {
	case monkeySit, monkeyEat:
		tail([4][2]float64{{10, 20}, {5.5, 22.5}, {2.4, 20}, {3.4, 16}})
		limb(11.4, 13.6, 13.4, 19.6, 1, true) // the far arm, its hand on the knee
		body := sBlend(2, aEll(12, 15, 4.6, 5.6), aEll(13, 20.3, 4.1, 2.1)).furry(0.3, 1, 624)
		q.body(body, monkFur, look{round: 3})
		q.coat(aEll(14.1, 16.4, 2.2, 3.4), monkPale) // the paler chest and belly
		thigh := aEll(13.6, 19.8, 3, 2.2).furry(0.2, 0.9, 625)
		q.body(thigh, monkFur, look{round: 1.8, bias: 0.06})
		q.coat(thigh.and(sBelow(20.4*cK)), monkPale)
		foot := sBlend(0.4, aCap(15.4, 21.2, 18, 21.6, 0.95, 0.72), aCap(17.4, 21.5, 18.7, 21.2, 0.3, 0.25))
		q.body(foot, monkHand, look{round: 0.6, bias: 0.02})
		head(14, 8)
		if pose == monkeyEat {
			limb(14.6, 13.2, 17.4, 10.6, 1, false)
			bn := newCTube(bez(6, 0.8, 0.45, [2]float64{18.1, 11.4}, [2]float64{18.9, 9.8}, [2]float64{19.2, 8.2}, [2]float64{18.8, 6.6}))
			q.body(bn.shape(), monkNana, look{round: 0.7, bias: 0.06})
			q.flat(aEll(18.8, 6.5, 0.32, 0.3), plain(bananaRamp[0])) // the stalk end
			q.body(aEll(17.9, 10.9, 1.15, 0.8), monkHand, look{round: 0.6}) // the fingers round it
		} else {
			limb(14.4, 13.4, 16.2, 19.2, 1, false)
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
		limb(8.4, 13.4, 5.5-2.5*s, 17.5+1.5*(1-s), -1, true)
		limb(15.6, 13.4, 17.5+2*s, 17.6+1.2*(1-s), 1, true)
		body := aEll(12, 13, 6.4, 3.2+0.4*(1-s)).turned(12, 13, -0.08*s).furry(0.3, 1, 624)
		q.body(body, monkFur, look{round: 2.6})
		q.coat(aEll(12.6, 15.2, 4.8, 1.3).turned(12, 13, -0.08*s), monkPale)
		limb(7.4, 14, 4.5-2.5*s, 18+1.5*(1-s), -1, false)
		limb(16.4, 14, 18.5+2*s, 18+1.2*(1-s), 1, false)
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
		leg := func(i int, x, y float64, far bool) {
			dx, up := g.foot(i)
			bend := 1.0
			if i%2 == 0 {
				bend = -1
			}
			limb(x, y+bob, x+dx, 21.7-up, bend, far)
		}
		leg(2, 9.4, 16, true)
		leg(3, 16.4, 16, true)
		body := sBlend(2, aEll(10.5, 15.2+bob, 4.4, 3.7), aEll(14.5, 14.8+bob, 4.2, 3.4)).furry(0.3, 1, 624)
		q.body(body, monkFur, look{round: 2.8})
		q.coat(aEll(12.8, 17.6+bob, 4.2, 1.2), monkPale)
		leg(0, 8.2, 16.6, false)
		leg(1, 15.4, 16.6, false)
		head(18, 11+bob)
	}
	return q.done(0.8, 13, 22)
}
