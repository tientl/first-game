package client

import (
	"math"
)

// The cave spider at HD density, painted (critpaint.go), side-on and facing
// right: a big round abdomen in a velvet of short hairs with a red
// hourglass low on its flank and the spinnerets at its tail, a glossy
// cephalothorax with a cluster of eight red eyes at its brow, fangs on
// heavy chelicerae and two short palps before them, and eight long legs,
// four a side, each bent high at the knee with pale rings at its joints.
// The far legs lie in the body's shade; the near ones come over the
// abdomen and out from under the cephalothorax, so the face stays clear.

var (
	spiderLegS  = newSurf(mkRamp(0x1a1220, 0x2e2238, 0x463452, 0x62506e, 0x84709a), sFur, 0.3, 801).with(2.4, math.Pi/2)
	spiderShell = newSurf(spiderRamp, sShell, 0.7, 802)
	spiderHair  = newSurf(spiderRamp, sFur, 0.15, 803).with(3, math.Pi/2)
	spiderMarkS = newSurf(spiderMark, sShell, 0.9, 804)
	spiderBand  = newSurf(mkRamp(0x3a2c40, 0x66546e, 0x947e98, 0xc4b0c4), sMatte, 0.2, 806).with(2.4, math.Pi/2)
	spiderFang  = newSurf(mkRamp(0x2a1a14, 0x5a4232, 0x9a8468, 0xd8ccb4), sShell, 0.9, 807)
)

func spiderHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(30, 19, 2, 2)
	const gy = 13.6 // the ground under the near feet
	lift, hx, hy, fang := 0.0, 16.0, 9.2, 0.0
	ph, walking := 0.0, false
	switch pose {
	case spiderWalkA:
		if f > 0 {
			ph, walking = cycPh(f), true
		}
	case spiderWalkB:
		ph, walking = 0.5, true
	case spiderSpit: // rears its abdomen up to spit
		lift = -1.5 * (blowAmt(f) + 1) / 2
		if f == 0 {
			lift = -1.5
		}
	case spiderBite: // draws back on its legs, then drives its fangs down
		amt := blowAmt(f)
		hx, hy, fang = lerpf(14.6, 17.4, (amt+1)/2), 9.2-0.8*clamp01(-amt), clamp01(amt)
	}
	bob := 0.0
	if walking {
		bob = 0.18 * math.Sin(4*math.Pi*ph)
	}
	hy += bob

	// A leg from its hip on the cephalothorax up to a high knee, down to the
	// ankle and out to the foot; i is 0 at the front.
	leg := func(i int, far bool) {
		fi := float64(i)
		reach := [4]float64{7.4, 3.8, -2.2, -6.2}[i]
		hipx, hipy := hx-0.2-fi*1, hy+0.9
		fx, fy, b := hx+reach, gy, 0.16
		if far {
			fx, fy, b = hx+reach*0.8+0.8, gy-1.1, -0.1
		}
		if walking {
			off := 0.5 * float64(i%2)
			if far {
				off += 0.5
			}
			dx, up := pace(ph+off, 0.6, 2.4, 1.5)
			fx, fy = fx+dx, fy-up
		}
		kx, ky := lerpf(hipx, fx, 0.36), hy-4.3+0.25*math.Abs(fi-1.2)-lift*0.15
		if far {
			ky += 0.7
		}
		ax, ay := lerpf(kx, fx, 0.6)+math.Copysign(0.5, reach), lerpf(ky, fy, 0.5)
		lk := look{round: 0.5, bias: b, far: far}
		q.body(aCap(hipx, hipy, kx, ky, 0.72, 0.6), spiderLegS, lk)
		q.body(aCap(kx, ky, ax, ay, 0.58, 0.46), spiderLegS, lk)
		q.body(aCap(ax, ay, fx, fy, 0.44, 0.16), spiderLegS, lk)
		// pale rings at the knee and the ankle
		q.coat(aCap(kx, ky, lerpf(kx, ax, 0.26), lerpf(ky, ay, 0.26), 0.7, 0.6), spiderBand)
		q.coat(aCap(ax, ay, lerpf(ax, fx, 0.24), lerpf(ay, fy, 0.24), 0.55, 0.5), spiderBand)
	}
	for i := 3; i >= 0; i-- {
		leg(i, true)
	}

	// The abdomen, hung from a narrow waist behind the cephalothorax.
	ab := sBlend(0.8, aEll(8, 8.2+lift, 6.2, 4.6), aEll(2.6, 9.6+lift*0.8, 1.1, 0.9))
	q.body(ab.furry(0.22, 0.7, 76), spiderHair, look{round: 4.2, bias: 0.04}) // a velvet of short hairs
	q.shade(aEll(9, 11.2+lift, 5.2, 1.6), -0.08)                              // its underside, away from the light
	// The red hourglass, low on the flank.
	mk := aPoly(5.6, 9.6+lift, 7.5, 10.5+lift, 9.4, 9.6+lift, 9.4, 11.8+lift, 7.5, 10.9+lift, 5.6, 11.8+lift).and(ab)
	q.coat(mk, spiderMarkS)
	q.shade(mk, 0.12)
	if pose == spiderSpit && (f == 0 || f >= 3) {
		web := aEll(1.4, 7.4+lift, 1.6, 1.4)
		q.fill(web, shd(ramp{hex(0xa8a8b4), hex(0xc8c8d0), hex(0xf4f4f8)}, web, 1, 0.1))
		q.flat(aCap(1.8, 8.6+lift, 2.2, 9.4+lift, 0.18, 0.14), plain(hex(0xd8d8e0))) // the strand to the spinnerets
	}

	// The near legs, then over their roots the cephalothorax: a glossy
	// dome, the brow at the front where the eyes
	// sit, the groove down its back.
	for i := 3; i >= 0; i-- {
		leg(i, false)
	}
	head := sBlend(1, aEll(hx, hy, 3.3, 2.6), aEll(hx+2.2, hy+0.4, 1.6, 1.5))
	q.body(head, spiderShell, look{round: 2.2, bias: 0.02})
	q.shade(aCap(hx-2, hy-1.5, hx+0.6, hy-2.2, 0.22, 0.2), -0.12)
	q.shade(aEll(hx+1.8, hy-1.4, 1.4, 0.7), 0.06) // the brow catches the light
	// The chelicerae, heavy and hanging, their fangs folding in under them.
	for _, o := range [2]float64{-0.5, 0.4} {
		far := o < 0
		b := 0.02
		if far {
			b = -0.12
		}
		cx, cy := hx+2.6+o, hy+1.2
		che := aCap(cx, cy, cx+0.3, cy+1.5, 0.75, 0.55)
		q.body(che, spiderShell, look{round: 0.6, bias: b, far: far})
		tx, ty := cx+0.35-0.9*(1-fang), cy+2.2+0.9*fang
		q.body(aPath(cx+0.4, cy+1.5, 0.32, lerpf(cx+0.4, tx, 0.5)+0.3, lerpf(cy+1.5, ty, 0.5), 0.24, tx, ty, 0.08), spiderFang, look{round: 0.3, bias: b, far: far, soft: true})
	}
	// The palps, short feelers before the fangs.
	for _, o := range [2]float64{-0.6, 0} {
		far := o < 0
		b := 0.04
		if far {
			b = -0.1
		}
		px, py := hx+2.4+o, hy+0.8
		kx, ky := px+1.6, py-0.6
		ex, ey := kx+0.9, ky+2.4-0.8*fang
		q.body(aCap(px, py, kx, ky, 0.42, 0.36), spiderLegS, look{round: 0.4, bias: b, far: far})
		q.body(aCap(kx, ky, ex, ey, 0.36, 0.3), spiderLegS, look{round: 0.35, bias: b, far: far})
		q.coat(aEll(ex, ey, 0.45, 0.45), spiderBand)
	}
	// Eight eyes in a cluster on the brow: two big ones at the front, the
	// rest smaller behind and above, each a red bead with a glint.
	for _, e := range [][3]float64{{2.9, -0.5, 0.55}, {2.1, -0.95, 0.48}, {1.2, -1.35, 0.34}, {2.5, -1.65, 0.3}, {1.65, -1.95, 0.26}} {
		x, y, r := hx+e[0], hy+e[1], e[2]
		q.flat(aEll(x, y, r+0.12, r+0.12), plain(hex(0x2a0806)))
		q.flat(aEll(x, y, r, r), plain(hex(0xd42418)))
		q.flat(aEll(x+0.06, y+0.08, r*0.6, r*0.6), plain(hex(0xff5a36)))
		q.dot(x-r*0.4, y-r*0.4, 0, 0, hex(0xffd8c0))
	}
	return q.done(0.8, 12, 14)
}
