package client

import (
	"math"
)

// The warrior of the woodland tribes at HD density, painted (critpaint.go),
// side-on and facing right, and as tall as the player (playerArtH): long in
// the leg, lean and hard, his dark skin lit warm with a cool shadow, the
// muscles of the shoulder, the chest and the calf each a form of its own.
// White clay paint in slashes down the chest, rings round the arms and a bar
// across the cheek; a bushy crest of black hair drawn up into a knot with
// two feathers in it, a bone through the ear, a necklace of teeth, armlets
// and anklets of plaited fibre, a hide loincloth with a fur trim on a rope
// belt, and a long spear: an ash shaft, a leaf of knapped flint lashed on
// with sinew, a red tassel under it.

var (
	tribeSkinS  = newSurf(mkRamp(0x24130c, 0x3c2016, 0x5a3220, 0x78482e, 0x96603e), sMatte, 0.3, 901)
	tribeHairS  = newSurf(mkRamp(0x08060a, 0x1a1418, 0x2c2428, 0x40363a), sFur, 0.1, 902).with(2.2, -math.Pi/2+0.4)
	tribeHide   = newSurf(loinRamp, sMatte, 0.05, 903)
	tribeTrim   = newSurf(mkRamp(0x7a6448, 0xa48a68, 0xc8b08c, 0xe6d6b8), sFur, 0, 904).with(2, math.Pi/2)
	tribePaint  = newSurf(mkRamp(0x8e887e, 0xbab2a4, 0xdcd6c8, 0xf6f2e8), sMatte, 0.1, 905)
	tribeShaft  = newSurf(spearShaft, sMatte, 0.2, 906)
	tribeFlint  = newSurf(mkRamp(0x24202a, 0x3c3644, 0x5a5262, 0x7e7686, 0xa8a0ae), sShell, 0.8, 907)
	tribeFibre  = newSurf(thatchRamp, sMatte, 0.1, 908)
	tribeRed    = newSurf(mkRamp(0x5a0c08, 0x9a1c12, 0xd8301c, 0xf86a30, 0xfcb070), sMatte, 0.2, 909)
	tribeBoneS  = newSurf(mkRamp(0x7a7060, 0xaea494, 0xd8d0c0, 0xf6f2e6), sMatte, 0.3, 910)
	tribeQuillS = newSurf(mkRamp(0x2a2622, 0x6a625a, 0xc8c0b4, 0xf4f0ea), sMatte, 0.2, 911)
)

// tribeKey is the warrior's pose: where his pelvis, ribcage and head are,
// his knees and feet, elbows and hands, and the butt and tip of his spear.
func tribeKey(pose int) tribePose {
	const fx = 15.0
	q := tribePose{
		pelvis: [2]float64{fx + 0.5, 25.6}, chest: [2]float64{fx + 1, 17.6}, head: [2]float64{fx + 1.8, 11.2},
		nearLeg: [2][2]float64{{fx + 1.7, 32.7}, {fx + 1.4, 40}}, farLeg: [2][2]float64{{fx - 0.3, 32.7}, {fx - 1, 40}},
		nearArm: [2][2]float64{{fx + 1.8, 21.6}, {fx + 6.3, 23.2}}, farArm: [2][2]float64{{fx - 0.2, 21.6}, {fx + 0.8, 26.4}},
		butt: [2]float64{fx + 6.7, 40.5}, tip: [2]float64{fx + 6.7, 6.5},
	}
	switch pose {
	case tribeWalkA:
		q.nearLeg = [2][2]float64{{fx + 4, 32.4}, {fx + 5, 40}}
		q.farLeg = [2][2]float64{{fx - 1.6, 32.8}, {fx - 4.6, 39.2}}
		q.nearArm = [2][2]float64{{fx - 0.4, 21.6}, {fx + 3.2, 24.2}}
		q.farArm = [2][2]float64{{fx + 2.2, 21.4}, {fx + 5, 24.4}}
		q.butt, q.tip = [2]float64{fx - 5, 31.4}, [2]float64{fx + 16.5, 11}
	case tribeWalkB:
		q.pelvis[1], q.chest, q.head = 25, [2]float64{fx + 1.6, 17.1}, [2]float64{fx + 2.6, 10.8}
		q.nearLeg = [2][2]float64{{fx + 0.2, 32.8}, {fx - 3.4, 39.2}}
		q.farLeg = [2][2]float64{{fx + 3.4, 32.4}, {fx + 4.6, 40}}
		q.nearArm = [2][2]float64{{fx + 2.4, 21.2}, {fx + 5.6, 23.6}}
		q.farArm = [2][2]float64{{fx - 1.2, 21.2}, {fx - 2.4, 25.2}}
		q.butt, q.tip = [2]float64{fx - 2, 31.4}, [2]float64{fx + 19.5, 10.6}
	case tribeThrust:
		q.pelvis, q.chest, q.head = [2]float64{fx, 26.6}, [2]float64{fx + 3.2, 18.8}, [2]float64{fx + 5.4, 13}
		q.nearLeg = [2][2]float64{{fx + 5, 33.2}, {fx + 6.2, 40}}
		q.farLeg = [2][2]float64{{fx - 4, 34}, {fx - 8, 39.6}}
		q.nearArm = [2][2]float64{{fx + 6.8, 21.8}, {fx + 10.4, 20.8}}
		q.farArm = [2][2]float64{{fx + 1.8, 22.6}, {fx + 5.6, 21.6}}
		q.butt, q.tip = [2]float64{fx - 6, 22.4}, [2]float64{fx + 25, 19.4}
	}
	return q
}

func lerpTribe(a, b tribePose, t float64) tribePose {
	l22 := func(x, y [2][2]float64) [2][2]float64 {
		return [2][2]float64{lerp2(x[0], y[0], t), lerp2(x[1], y[1], t)}
	}
	return tribePose{
		pelvis: lerp2(a.pelvis, b.pelvis, t), chest: lerp2(a.chest, b.chest, t), head: lerp2(a.head, b.head, t),
		nearLeg: l22(a.nearLeg, b.nearLeg), farLeg: l22(a.farLeg, b.farLeg),
		nearArm: l22(a.nearArm, b.nearArm), farArm: l22(a.farArm, b.farArm),
		butt: lerp2(a.butt, b.butt, t), tip: lerp2(a.tip, b.tip, t),
	}
}

// tribeHD draws the warrior: standing with his spear upright, walking and
// running with it carried low and forward (frames 1 to cycFrames of
// tribeWalkA and tribeWalkB), and thrusting it (wound back, then driven).
func tribeHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(52, 48, 4, 4)
	const gy = 40.0
	p := tribeKey(pose)
	switch {
	case (pose == tribeWalkA || pose == tribeWalkB) && f > 0:
		ph := cycPh(f)
		stride, lift, lean := 7.6, 2.0, 0.0
		if pose == tribeWalkB { // running, leaning into it
			stride, lift, lean = 10.5, 3.2, 1.3
		}
		p = lerpTribe(tribeKey(tribeWalkA), tribeKey(tribeWalkB), 0.5-0.5*math.Cos(2*math.Pi*ph))
		bob := 0.5 * math.Cos(4*math.Pi*ph)
		p.pelvis[1] = 25.3 + bob
		p.chest[0], p.chest[1] = p.chest[0]+lean, p.chest[1]+bob
		p.head[0], p.head[1] = p.head[0]+lean*1.5, p.head[1]+bob
		for i, l := range []*[2][2]float64{&p.nearLeg, &p.farLeg} {
			dx, up := pace(ph+0.5*float64(i), 0.6, stride, lift)
			hx, hy := p.pelvis[0]+1-2*float64(i), p.pelvis[1]+0.6
			fx, fy := hx+0.6+dx, gy-up
			kx, ky := ik2(hx, hy, fx, fy-1.1, 6.6, 6.6, -1)
			*l = [2][2]float64{{kx, ky}, {fx, fy}}
		}
	case pose == tribeThrust && f > 0:
		// Drawn back first: the weight on the back leg, the spear cocked.
		const fx = 15.0
		back := tribeKey(tribeStand)
		back.pelvis, back.chest, back.head = [2]float64{fx - 1, 26.2}, [2]float64{fx - 1.6, 17.8}, [2]float64{fx - 1, 12.2}
		back.nearLeg = [2][2]float64{{fx + 3.6, 33}, {fx + 5, 40}}
		back.farLeg = [2][2]float64{{fx - 3.6, 33.2}, {fx - 5.6, 40}}
		back.nearArm = [2][2]float64{{fx - 3.6, 22.4}, {fx - 6, 21.1}}
		back.farArm = [2][2]float64{{fx + 1.4, 22.2}, {fx + 3.8, 20.8}}
		back.butt, back.tip = [2]float64{fx - 16, 21.4}, [2]float64{fx + 15, 20.4}
		p = lerpTribe(back, p, smooth((blowAmt(f)+1)/2))
	}
	P, C, H := p.pelvis, p.chest, p.head
	shoulder := [2]float64{C[0] - 0.5, C[1] - 1.9}
	look1 := func(round, bias float64, far bool) look {
		if far {
			bias -= 0.06
		}
		return look{round: round, bias: bias, far: far}
	}
	// ring is a band round a limb at (x, y) running along (dx, dy): an
	// armlet, an anklet, a ring of paint.
	ring := func(x, y, dx, dy, half, r float64, sf *surf) {
		l := math.Hypot(dx, dy)
		nx, ny := -dy/l*half, dx/l*half
		q.coat(aCap(x-nx, y-ny, x+nx, y+ny, r, r), sf)
	}

	leg := func(hipX float64, l [2][2]float64, far bool) {
		hx, hy := hipX, P[1]+0.6
		kx, ky := l[0][0], l[0][1]
		ax, ay := l[1][0], l[1][1]-1.1
		thigh := aCap(hx, hy, kx, ky, 1.6, 1.1)
		q.body(thigh, tribeSkinS, look1(1.1, 0.02, far))
		// the shin, the calf swelling behind it
		cx0, cy0 := lerpf(kx, ax, 0.12)-0.45, lerpf(ky, ay, 0.12)
		cx1, cy1 := lerpf(kx, ax, 0.55)-0.25, lerpf(ky, ay, 0.55)
		shin := sBlend(0.7, aCap(kx, ky, ax, ay, 1.05, 0.6), aCap(cx0, cy0, cx1, cy1, 1.15, 0.75))
		q.body(shin, tribeSkinS, look1(0.9, 0.02, far))
		q.shade(aEll(kx+0.5, ky-0.1, 0.5, 0.6), 0.08) // the kneecap
		foot := sBlend(0.5, aCap(ax-0.4, ay+0.2, ax+2.5, l[1][1]-0.4, 0.75, 0.48), aEll(ax-0.2, l[1][1]-0.6, 0.75, 0.6))
		q.body(foot, tribeSkinS, look1(0.6, 0.04, far))
		ring(lerpf(kx, ax, 0.82), lerpf(ky, ay, 0.82), ax-kx, ay-ky, 1, 0.3, tribeFibre) // an anklet
	}
	arm := func(a [2][2]float64, far bool) {
		sx, sy := shoulder[0], shoulder[1]
		ex, ey := a[0][0], a[0][1]
		hx, hy := a[1][0], a[1][1]
		wx, wy := lerpf(ex, hx, 0.82), lerpf(ey, hy, 0.82)
		upper := sBlend(0.8, aCap(sx, sy, ex, ey, 1.2, 0.8), aEll(sx+0.25, sy+0.7, 1.45, 1.5))
		q.body(upper, tribeSkinS, look1(1, 0.03, far))
		q.body(aCap(ex, ey, wx, wy, 0.9, 0.58), tribeSkinS, look1(0.8, 0.03, far))
		q.body(aEll(hx, hy, 0.82, 0.78), tribeSkinS, look1(0.6, 0.06, far))
		ring(lerpf(sx, ex, 0.62), lerpf(sy, ey, 0.62), ex-sx, ey-sy, 1.2, 0.28, tribeFibre) // an armlet
		if !far {
			ring(lerpf(ex, wx, 0.45), lerpf(ey, wy, 0.45), wx-ex, wy-ey, 1.2, 0.22, tribePaint)
		}
	}

	arm(p.farArm, true)
	leg(P[0]-1, p.farLeg, true)

	// The torso: a deep ribcage over a hard, narrow waist, the buttock
	// behind, the chest swelling in front.
	mx, my := lerpf(P[0], C[0], 0.5), lerpf(P[1], C[1], 0.5)
	torso := sBlend(1.6,
		aCap(P[0], P[1]-0.3, mx, my, 2.15, 2),
		aEll(C[0]+0.2, C[1], 2.6, 3),
		aEll(C[0]+1.2, C[1]-0.3, 1.6, 1.8),
		aEll(P[0]-1, P[1]+0.6, 1.9, 1.8))
	q.body(torso, tribeSkinS, look{round: 1.9, bias: 0.03})
	B := func(u, v float64) (float64, float64) { return C[0] + u, C[1] + v }
	cap := func(u0, v0, u1, v1, r float64) shape {
		x0, y0 := B(u0, v0)
		x1, y1 := B(u1, v1)
		return aCap(x0, y0, x1, y1, r, r)
	}
	q.shade(cap(0.2, 1.7, 2.6, 1.4, 0.24), -0.16)  // under the chest
	q.shade(cap(1.6, 3.2, 1.8, 6.4, 0.18), -0.1)   // the belly's muscles
	q.shade(cap(0.6, 4.4, 2.4, 4.3, 0.16), -0.08)
	q.shade(cap(-2.4, -1, -2.2, 4.6, 0.3), -0.1)   // the groove down the back
	q.shade(cap(0.4, -1.6, 2, -1.3, 0.5), 0.06)    // the top of the chest, lit
	for k := 0.0; k < 3; k++ { // white slashes down the chest
		q.coat(cap(0.5+0.8*k, -0.6+0.3*k, 0.9+0.8*k, 1.1+0.3*k, 0.22), tribePaint)
	}
	for k := 0.0; k < 2; k++ { // and dots on the belly
		q.coat(cap(0.9+0.8*k, 3.6+1.1*k, 0.9+0.8*k, 3.6+1.1*k, 0.32), tribePaint)
	}
	leg(P[0]+1, p.nearLeg, false)

	// The loincloth: a rope belt, a long flap of hide before and a short
	// one behind, both trimmed with pale fur.
	px, py := P[0], P[1]
	behind := aPoly(px-2.7, py-0.4, px-0.6, py-0.4, px-1, py+3.4, px-2.2, py+2.9, px-3.2, py+3.8)
	front := aPoly(px-0.4, py-0.4, px+2.9, py-0.4, px+3.3, py+5.4, px+2, py+4.8, px+0.8, py+6, px-0.2, py+4.6)
	q.body(behind, tribeHide, look{round: 0.8, bias: -0.06, flat: 0.5})
	q.coat(behind.and(sBelow((py+2.6)*cK)), tribeTrim)
	q.body(front, tribeHide, look{round: 0.9, flat: 0.5})
	q.coat(front.and(sBelow((py+4.2)*cK)), tribeTrim)
	q.shade(aCap(px+1.2, py+0.6, px+1.5, py+4.4, 0.2, 0.2), -0.12) // a fold down the flap
	q.body(aCap(px-2.5, py-0.6, px+2.8, py-0.6, 0.42, 0.42), tribeFibre, look{round: 0.45, bias: 0.04})
	q.body(aCap(px+1.8, py-0.1, px+1.9, py+1.5, 0.36, 0.26), tribeBoneS, look{round: 0.3, soft: true}) // a bone toggle

	// The head in profile: the neck, the skull, a heavy brow, a broad nose,
	// full lips, a strong jaw.
	hx, hy := H[0], H[1]
	q.body(aCap(C[0]+0.1, C[1]-2.2, hx-0.5, hy+1.5, 1.2, 1.05), tribeSkinS, look{round: 0.9, bias: -0.04})
	face := sBlend(0.7,
		aEll(hx-0.2, hy-0.3, 2.35, 2.45),
		aEll(hx+0.9, hy+1.2, 1.7, 1.3),
		aCap(hx+1.9, hy-0.4, hx+2.7, hy+0.6, 0.38, 0.45),
		aEll(hx+2.2, hy+1.6, 0.55, 0.42))
	q.body(face, tribeSkinS, look{round: 1.5, bias: 0.06})
	q.shade(aEll(hx+1.4, hy-1.15, 1.1, 0.45), 0.1)                  // the brow, lit
	q.shade(aCap(hx+0.9, hy-0.6, hx+2, hy-0.6, 0.3, 0.25), -0.14)   // its shadow over the eye
	q.shade(aEll(hx+0.9, hy+1.6, 1.2, 0.7), -0.06)                  // under the cheekbone
	q.coat(aCap(hx-0.4, hy+0.5, hx+1.7, hy+0.4, 0.3, 0.28), tribePaint) // a bar of paint across the cheek
	q.coat(aCap(hx+0.5, hy-1.9, hx+1.3, hy-1.8, 0.2, 0.2), tribePaint)  // and one on the brow
	ear := aEll(hx-0.6, hy+0.3, 0.55, 0.85)
	q.body(ear, tribeSkinS, look{round: 0.5, bias: 0.02, soft: true})
	q.body(aCap(hx-1.3, hy+1.2, hx-0.1, hy+1.4, 0.2, 0.16), tribeBoneS, look{round: 0.2, soft: true}) // a bone through it
	// The hair: a bushy crest over the crown and the back of the head,
	// drawn up into a knot with a red cord, two feathers standing from it.
	hair := sBlend(0.8, aEll(hx-0.6, hy-1.5, 2.6, 2), aEll(hx-1.8, hy-0.1, 1.4, 1.9)).sub(aEll(hx+1.9, hy+0.7, 2, 2.1)).sub(ear.grow(0.3 * cK))
	q.body(hair.furry(0.32, 0.7, 97), tribeHairS, look{round: 1.4, bias: 0.04})
	knot := aEll(hx-0.6, hy-3.6, 1.2, 1.1).furry(0.25, 0.6, 98)
	q.body(knot, tribeHairS, look{round: 1, bias: 0.06})
	q.body(aCap(hx-1.4, hy-2.9, hx+0.2, hy-3, 0.24, 0.24), tribeRed, look{round: 0.3, soft: true})
	q.body(newCTube(bez(8, 0.62, 0.12, [2]float64{hx - 1, hy - 4.2}, [2]float64{hx - 2.4, hy - 5.4}, [2]float64{hx - 3.8, hy - 6.6}, [2]float64{hx - 5.2, hy - 7.6})).shape(), tribeRed, look{round: 0.5, bias: 0.04})
	q.body(newCTube(bez(8, 0.5, 0.1, [2]float64{hx - 0.4, hy - 4.4}, [2]float64{hx - 0.6, hy - 6}, [2]float64{hx - 1, hy - 7.4}, [2]float64{hx - 1.6, hy - 8.6})).shape(), tribeQuillS, look{round: 0.4, bias: 0.06})
	// The eye, the nostril and the line of the mouth.
	q.flat(aEll(hx+1.5, hy-0.3, 0.44, 0.3), plain(hex(0xeee2cc)))
	q.flat(aEll(hx+1.72, hy-0.28, 0.24, 0.28), plain(hex(0x1a0c08)))
	q.dot(hx+1.62, hy-0.42, 0, 0, hex(0xfff6e4))
	q.dot(hx+2.35, hy+0.55, 0, 0, hex(0x1a0c08))
	q.flat(aCap(hx+1.4, hy+1.55, hx+2.55, hy+1.5, 0.14, 0.12), plain(hex(0x1e0e0a)))
	// A necklace of teeth hung round the neck.
	for k := 0.0; k < 6; k++ {
		t := k / 5
		x, y := B(-0.6+3*t, -2.4+1.6*math.Sin(math.Pi*t)*0.8)
		q.body(aCap(x, y, x+0.1, y+0.85, 0.28, 0.12), tribeBoneS, look{round: 0.25, soft: true})
	}

	// The spear.
	dx, dy := p.tip[0]-p.butt[0], p.tip[1]-p.butt[1]
	n := math.Hypot(dx, dy)
	dx, dy = dx/n, dy/n
	q.body(aCap(p.butt[0], p.butt[1], p.tip[0], p.tip[1], 0.42, 0.38), tribeShaft, look{round: 0.4, bias: 0.04})
	bx, by := p.tip[0]-dx*0.5, p.tip[1]-dy*0.5
	at := func(u, v float64) (float64, float64) { return bx + dx*u - dy*v, by + dy*u + dx*v }
	var pts []float64
	for _, o := range [][2]float64{{0, 0.9}, {2, 1.75}, {6.6, 0}, {2, -1.75}, {0, -0.9}} {
		x, y := at(o[0], o[1])
		pts = append(pts, x, y)
	}
	blade := aPoly(pts...)
	q.body(blade, tribeFlint, look{round: 0.9, bias: 0.04, flat: 0.4})
	rx0, ry0 := at(0.6, 0)
	rx1, ry1 := at(5.6, 0)
	q.shade(blade.and(aCap(rx0, ry0, rx1, ry1, 0.2, 0.12)), 0.12) // the ridge down the middle
	for k := 0; k < 4; k++ { // the scars of knapping along its edges
		x, y := at(1.2+1.1*float64(k), 0.9-0.2*float64(k))
		q.shade(blade.and(aEll(x, y, 0.45, 0.35)), -0.1)
	}
	for k := 0.3; k <= 1.9; k += 0.8 { // the lashing
		x0, y0 := at(-k, 0.7)
		x1, y1 := at(-k-0.35, -0.7)
		q.body(aCap(x0, y0, x1, y1, 0.24, 0.24), tribeBoneS, look{round: 0.2, soft: true})
	}
	tx, ty := at(-2.6, 0)
	q.body(aCap(tx, ty, tx-0.5, ty+2.6, 0.55, 0.28), tribeRed, look{round: 0.45})
	q.body(aCap(tx-0.3, ty, tx-1.3, ty+2.1, 0.45, 0.22), tribeRed, look{round: 0.4, bias: -0.1})

	// The near arm over it all, its hand round the shaft.
	arm(p.nearArm, false)
	return q.done(0.85, 15, 41)
}
