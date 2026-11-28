package client

import (
	"math"
)

// The giant scorpion of the desert at HD density, painted (critpaint.go),
// side-on and facing right: black chitin with a deep amber gleam. A low
// shield of a carapace with its eyes on top, the back in seven plates, each
// swelling and lit along its rear edge; the tail of five keeled segments
// arching over it to the swollen, paler venom bulb and its curved barb; two
// great claws, knobbed and ridged, their fingers dark at the tips; and four
// legs a side, each in three bends.

var (
	scorpShell = newSurf(scorpRamp[1:7], sShell, 1, 805)
	scorpLeg   = newSurf(scorpRamp[1:7], sShell, 0.7, 806)
	scorpBulb  = newSurf(mkRamp(0x2a1411, 0x5a2a14, 0x96461a, 0xcb7b2c, 0xf4c062), sShell, 1, 807)
	scorpTip   = newSurf(mkRamp(0x080506, 0x160c0c, 0x2a1814, 0x4a2c1e), sShell, 1, 808)
)

func lerpClaw(a, b scorpClaw, t float64) scorpClaw {
	return scorpClaw{lerpf(a.ex, b.ex, t), lerpf(a.ey, b.ey, t), lerpf(a.x, b.x, t), lerpf(a.y, b.y, t), lerpf(a.ang, b.ang, t), lerpf(a.open, b.open, t)}
}

// lerpScorp is the pose t of the way from a to b.
func lerpScorp(a, b scorpPose, t float64) scorpPose {
	p := scorpPose{dx: lerpf(a.dx, b.dx, t), dy: lerpf(a.dy, b.dy, t), near: lerpClaw(a.near, b.near, t), far: lerpClaw(a.far, b.far, t)}
	for i := range p.tail {
		p.tail[i] = lerp2(a.tail[i], b.tail[i], t)
	}
	for i := range p.sting {
		p.sting[i] = lerp2(a.sting[i], b.sting[i], t)
	}
	return p
}

func scorpionHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(scorpW+6, scorpH+4, 2, 3)
	p := scorpPoses[pose]
	ph, walking := 0.0, false
	switch pose {
	case scorpionWalkA:
		if f > 0 {
			ph, walking = cycPh(f), true
			p = lerpScorp(scorpPoses[scorpionWalkA], scorpPoses[scorpionWalkB], 0.5-0.5*math.Cos(2*math.Pi*ph))
		}
	case scorpionWalkB:
		ph, walking = 0.5, true
	case scorpionSting, scorpionPinch:
		if f > 0 { // drawn back first: the tail coiled high, the claws cocked open
			back := scorpPoses[scorpionWalkA]
			if pose == scorpionSting {
				for i := 1; i < len(back.tail); i++ {
					back.tail[i][0] -= 0.7 * float64(i)
					back.tail[i][1] -= 0.3 * float64(i)
				}
				back.sting = [2][2]float64{{back.tail[5][0] + 3.6, back.tail[5][1] - 1}, {back.tail[5][0] + 6.5, back.tail[5][1] + 1.5}}
			} else {
				back.dx = -1.5
				back.near = scorpClaw{40.5, 18, 44, 20, -0.25, 1.2}
				back.far = scorpClaw{39.5, 14, 43, 11.5, -0.5, 1.1}
			}
			p = lerpScorp(back, p, smooth((blowAmt(f)+1)/2))
		}
	}
	bx, by := p.dx, p.dy

	// A leg: from its hip under the plates up to a knee above them, down
	// to the ankle and out to a pointed foot.
	leg := func(i int, far bool) {
		fi := float64(i)
		b, gy, k := 0.06, scorpGy, 1.0
		if far {
			b, gy, k = -0.08, scorpGy-1.6, 0.85
		}
		dx, up := 0.0, 0.0
		if walking {
			off := 0.5 * float64(i%2)
			if far {
				off += 0.5
			}
			dx, up = pace(ph+off, 0.62, 4.2, 2.2)
		}
		hx, hy := 19.5+fi*4.6+bx, 23.6+by
		fx := hx + (fi-1.5)*5.6*k + dx
		kx, ky := lerpf(hx, fx, 0.5), 20.4-up*0.5+by+0.3*math.Abs(fi-1.5)
		ax, ay := lerpf(kx, fx, 0.55)+(fi-1.5)*0.6, lerpf(ky, gy-up, 0.5)
		lk := look{round: 0.6, bias: b, far: far}
		q.body(aCap(hx, hy, kx, ky, 1.1, 0.85), scorpLeg, lk)
		q.body(aCap(kx, ky, ax, ay, 0.8, 0.6), scorpLeg, lk)
		q.body(aCap(ax, ay, fx, gy-up, 0.55, 0.2), scorpLeg, lk)
		q.coat(aCap(lerpf(ax, fx, 0.6), lerpf(ay, gy-up, 0.6), fx, gy-up, 0.5, 0.2), scorpTip)
	}
	// A claw: the arm out to the elbow and on to the wrist, the hand
	// swelling round its knobs, the fixed finger and the one that opens.
	claw := func(c scorpClaw, far bool) {
		b := 0.04
		if far {
			b = -0.1
		}
		lk := func(round, bias float64) look { return look{round: round, bias: b + bias, far: far} }
		q.body(aCap(39+bx, 21+by, c.ex+bx, c.ey+by, 1.7, 1.5), scorpShell, lk(1.2, -0.04))
		q.body(aCap(c.ex+bx, c.ey+by, c.x+bx, c.y+by, 1.6, 1.9), scorpShell, lk(1.2, 0))
		x, y := c.x+bx, c.y+by
		T := func(s shape, a float64) shape { return s.turned(x, y, a) }
		up := T(aPath(x+3.2, y-1.9, 1.5, x+7, y-2.3, 1.05, x+10.4, y-1.1, 0.3), c.ang-c.open*0.5)
		lo := T(aPath(x+3.2, y+1.7, 1.3, x+6.8, y+1.9, 0.9, x+9.8, y+0.9, 0.28), c.ang+c.open*0.5)
		q.body(lo, scorpShell, lk(0.9, -0.04))
		q.coat(lo.and(T(aEll(x+9.8, y+0.9, 1.5, 1.4), c.ang+c.open*0.5)), scorpTip)
		hand := T(sBlend(1.2, aEll(x+2.4, y, 4.1, 3.2), aEll(x+4.6, y-0.4, 2.4, 2.4)), c.ang)
		q.body(hand, scorpShell, lk(2.4, 0.04))
		q.shade(T(aCap(x+0.4, y-1.2, x+5.6, y-1.6, 0.25, 0.22), c.ang), -0.14) // a ridge down the hand
		q.shade(T(aCap(x+0.6, y+0.6, x+5.4, y+0.5, 0.22, 0.2), c.ang), -0.1)
		q.body(up, scorpShell, lk(0.9, 0.06))
		q.coat(up.and(T(aEll(x+10.4, y-1.1, 1.6, 1.4), c.ang-c.open*0.5)), scorpTip)
	}
	for i := 3; i >= 0; i-- {
		leg(i, true)
	}
	claw(p.far, true)

	// The tail: five keeled segments, each a little narrower than the last,
	// then the bulb of venom and its barb.
	for i := 0; i < 5; i++ {
		a, b := p.tail[i], p.tail[i+1]
		fi := float64(i)
		ax, ay := a[0]+bx*0.5, a[1]+by
		ex, ey := lerpf(a[0], b[0], 0.86)+bx*0.5, lerpf(a[1], b[1], 0.86)+by
		seg := aCap(ax, ay, ex, ey, 2.9-0.2*fi, 2.5-0.2*fi)
		q.body(seg, scorpShell, look{round: 1.8, bias: 0.02})
		// the keel along its top, lit, and the joint at its root, dark
		nx, ny := ey-ay, -(ex - ax)
		if ny > 0 {
			nx, ny = -nx, -ny
		}
		l := math.Hypot(nx, ny)
		nx, ny = nx/l*1.4, ny/l*1.4
		q.shade(seg.and(aCap(lerpf(ax, ex, 0.15)+nx, lerpf(ay, ey, 0.15)+ny, lerpf(ax, ex, 0.85)+nx, lerpf(ay, ey, 0.85)+ny, 0.35, 0.3)), 0.1)
		q.shade(seg.and(aEll(ax, ay, 1.1, 1.1).sub(aEll(lerpf(ax, ex, 0.3), lerpf(ay, ey, 0.3), 1.3, 1.3))), -0.16)
	}
	tx, ty := p.tail[5][0]+bx*0.5, p.tail[5][1]+by
	bulb := sBlend(1, aEll(tx+1.4, ty+0.4, 3.2, 2.6), aEll(tx+3.2, ty+0.6, 1.8, 1.6))
	q.body(bulb, scorpBulb, look{round: 2, bias: 0.04})
	barb := newCTube(bez(10, 1.05, 0.12, [2]float64{tx + 3.6, ty + 0.4}, p.sting[0], p.sting[0], p.sting[1]))
	q.body(barb.shape(), scorpTip, look{round: 0.6, bias: 0.1})
	if pose == scorpionSting {
		q.dot(p.sting[1][0], p.sting[1][1], 0, 1, venomGreen)
		q.dot(p.sting[1][0], p.sting[1][1], 0, 3, venomGreen)
	}

	// The back: seven plates in one low arch, each swelling a little, the
	// seams between them dark and each plate's rear edge catching the light.
	var plates []shape
	for i := 0; i < 7; i++ {
		fi := float64(i)
		plates = append(plates, aEll(16+fi*3.6+bx, 20.6-0.25*fi+0.04*fi*fi+by, 3, 4.4+0.6*math.Sin(fi*0.5)))
	}
	back := sBlend(1.4, plates...)
	q.body(back, scorpShell, look{round: 2.6, bias: 0.02})
	for i := 0; i < 6; i++ {
		fi := float64(i)
		sx := 17.8 + fi*3.6 + bx
		top := 16.4 - 0.3*fi + by
		q.shade(back.and(aCap(sx, top, sx+0.5, top+8, 0.28, 0.24)), -0.18)
		q.shade(back.and(aCap(sx-0.9, top+0.2, sx-0.5, top+5, 0.4, 0.3)), 0.06)
	}
	q.shade(back.and(sBelow((23.2+by)*cK)), -0.08) // the belly, turned from the light

	// The carapace, a low shield over the head, and its eyes: a pair on top,
	// a few small ones round the front edge.
	head := sBlend(1, aEll(40+bx, 20.2+by, 4.4, 3.5), aEll(42.8+bx, 21.4+by, 2.1, 2))
	q.body(head, scorpShell, look{round: 2.2, bias: 0.04, flat: 0.3})
	q.shade(aCap(37.6+bx, 18.2+by, 41.6+bx, 17.6+by, 0.3, 0.26), 0.08)
	for _, e := range [][3]float64{{40.4, 17.4, 0.55}, {41.4, 17.6, 0.5}} {
		x, y := e[0]+bx, e[1]+by
		q.flat(aEll(x, y, e[2]+0.1, e[2]+0.1), plain(hex(0x0a0606)))
		q.flat(aEll(x+0.08, y+0.1, e[2]*0.55, e[2]*0.5), plain(hex(0x3a2418)))
		q.dot(x-e[2]*0.4, y-e[2]*0.45, 0, 0, hex(0xffd890))
	}
	for k := 0.0; k < 3; k++ {
		q.dot(43.6+bx+0.5*k, 19.4+by+0.4*k, 0, 0, hex(0xd88020))
	}

	for i := 3; i >= 0; i-- {
		leg(i, false)
	}
	claw(p.near, false)
	return q.done(0.85, scorpAx, scorpGy)
}
