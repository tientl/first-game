package client

import (
	"math"
)

// The crab at HD density, painted (critpaint.go): seen from the front, its
// shell a hard, shining dome with grooves and granules and a toothed front
// rim, eyes on stalks, two big claws with knobbled palms and dark-tipped
// fingers, and four jointed legs a side.

var (
	crabShell  = newSurf(crabRamp, sShell, 0.9, 401)
	crabLimb   = newSurf(crabRamp, sShell, 0.6, 402)
	crabLegS   = newSurf(mkRamp(0x5a1a10, 0x8e2c18, 0xc04a26, 0xe07038, 0xf09a5a), sShell, 0.45, 403)
	crabUnder  = newSurf(ramp{hex(0x8a6448), hex(0xc0966a), hex(0xe2c497), hex(0xf6e6c4)}, sPlate, 0.2, 404)
	crabTip    = newSurf(ramp{hex(0x1e1412), hex(0x3a2622), hex(0x5e4036), hex(0x8a6a58)}, sShell, 0.8, 405)
	crabStalk  = newSurf(ramp{hex(0x7e2416), hex(0xb33e22), hex(0xd8622e), hex(0xf09a5a)}, sMatte, 0.2, 406)
	crabJoint  = newSurf(ramp{hex(0xa05a36), hex(0xd08858), hex(0xecb080), hex(0xfadcb0)}, sMatte, 0.3, 408)
	crabPalmTu = newSurf(ramp{hex(0xb33e22), hex(0xd8622e), hex(0xf0904a), hex(0xfac890)}, sShell, 0.8, 409)
)

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
	bob := 0.0
	if walking {
		bob = 0.22 * math.Sin(4*math.Pi*ph)
	}
	cy += bob

	// Legs: four a side, from under the shell's rim: the thigh up and out
	// to a high knee, the shank down, a pointed dark-tipped foot.
	for _, fi := range []float64{3, 2, 1, 0} {
		for side := -1.0; side <= 1; side += 2 {
			dx, up := 0.0, 0.0
			if walking {
				dx, up = pace(ph+0.25*fi+0.25*(side+1), 0.55, 1.6, 1.2)
			}
			// fanned out: the front pair high and forward, each pair behind
			// it lower and further out, so all four read apart
			bx, by := cx+side*(5-0.45*fi), cy+0.3+0.7*fi-bob
			kx, ky := cx+side*(7.6+0.75*fi)+dx*0.5, cy-1.9+0.95*fi-up-bob
			ax, ay := kx+side*(1.1+0.2*fi)+dx*0.35, ky+2-up*0.3
			fx, fy := ax+side*(0.45+0.15*fi)+dx*0.2, 14.9-up-0.45*fi
			b := 0.03 - 0.07*fi // the hind legs further back, in the shell's shade
			q.body(aCap(bx, by, kx, ky, 0.66, 0.56), crabLegS, look{round: 0.5, bias: b + 0.04})
			q.body(aCap(kx, ky, ax, ay, 0.52, 0.42), crabLegS, look{round: 0.4, bias: b})
			q.body(aEll(kx, ky, 0.4, 0.38), crabJoint, look{round: 0.38, bias: b - 0.04, soft: true})
			foot := aCap(ax, ay, fx, fy, 0.36, 0.08)
			q.body(foot, crabLegS, look{round: 0.34, bias: b - 0.02})
			q.coat(aCap(lerpf(ax, fx, 0.55), lerpf(ay, fy, 0.55), fx, fy, 0.4, 0.12), crabTip)
			for h := 0.45; h < 0.9; h += 0.4 { // bristles on the shank's back
				q.dot(lerpf(kx, ax, h)+side*0.5, lerpf(ky, ay, h), int(side), 0, crabRamp[1])
			}
		}
	}

	// Claws: an arm forward and out, a knobbed wrist, then the palm and its
	// two fingers.
	claws := func() {
		for side := -1.0; side <= 1; side += 2 {
			px, py, ang, gap := cx+side*8.4, cy-5.4, side*0.5, 0.5
			if pinch {
				px, py, ang, gap = cx+side*9.1, cy-8.4+0.6*snap, side*0.25, 1.15*(1-snap)+0.12
			}
			sx, sy := cx+side*4.2, cy-0.6
			ex, ey := cx+side*7.2, cy-0.9-1.2*clamp01(-(py-cy)/9)
			q.body(aCap(sx, sy, ex, ey, 1.05, 0.95), crabLimb, look{round: 0.9, bias: -0.06})
			q.body(aCap(ex, ey, px-side*0.5, py+1.7, 0.95, 0.9), crabLimb, look{round: 0.8, bias: -0.02})
			for k := 0.0; k < 2; k++ { // knobs along the wrist's top
				t := 0.3 + 0.3*k
				q.body(aEll(lerpf(ex, px, t)-side*0.25, lerpf(ey, py+1.7, t)-0.6, 0.3, 0.28), crabPalmTu, look{round: 0.3, soft: true})
			}
			// The palm points up and a little in: a broad hand, a fixed
			// finger and a thumb that opens.
			tx, ty := -math.Sin(ang), -math.Cos(ang)
			hand := aEll(px, py, 2.35, 1.95)
			finger := aPoly(px-side*2.15, py-0.6, px-side*1.5+tx*4, py-0.4+ty*4, px-side*0.25, py-1.2)
			thumb := aPoly(px+side*0.1, py-1.3, px+side*(0.5+gap)+tx*3.7, py+ty*3.7, px+side*2.15, py-0.5)
			q.body(sBlend(0.5, finger, aEll(px-side*1.2, py-1, 0.9, 0.8)), crabLimb, look{round: 0.7, bias: -0.04})
			q.body(sBlend(0.5, thumb, aEll(px+side*1.2, py-1, 0.9, 0.8)), crabLimb, look{round: 0.7})
			q.body(hand, crabShell, look{round: 1.8})
			// the dark tips of the fingers, and the teeth on their inner edges
			q.coat(finger.and(aEll(px-side*1.5+tx*4, py-0.4+ty*4, 1.9, 1.9)), crabTip)
			q.coat(thumb.and(aEll(px+side*(0.5+gap)+tx*3.7, py+ty*3.7, 1.8, 1.8)), crabTip)
			for k := 0.0; k < 3; k++ {
				t := 0.35 + 0.2*k
				q.dot(lerpf(px-side*0.4, px-side*1.4+tx*3.8, t)+side*0.35, lerpf(py-1.1, py-0.4+ty*3.8, t), 0, 0, hex(0xf6d8b0))
			}
			// Granules in rows over the palm.
			for k := 0; k < 5; k++ {
				gx := px - side*0.2 + (float64(k%3)-1)*1.1
				gy := py + 0.2 + float64(k/3)*0.9 - 0.4
				q.body(aEll(gx, gy, 0.24, 0.22), crabPalmTu, look{round: 0.24, soft: true, bias: 0.04})
			}
		}
	}
	if !pinch {
		claws() // held low, the shell over their roots
	}

	// The pale underside below the shell, and the mouthparts.
	under := sBlend(1.2, aEll(cx, cy+2.4, 4.6, 1.9), aEll(cx, cy+1.3, 5.4, 1.6))
	q.body(under, crabUnder, look{round: 1.2, bias: 0.1})
	q.body(aPoly(cx-1.3, cy+0.6, cx+1.3, cy+0.6, cx+0.9, cy+3.1, cx-0.9, cy+3.1), crabUnder, look{round: 0.6, bias: 0.04, flat: 0.5})
	q.shade(aCap(cx, cy+1, cx, cy+3, 0.13, 0.13), -0.3)
	// The shell: a dome, widest towards the front, its rim toothed along
	// the upper sides.
	shell := sBlend(1.6, aEll(cx, cy-1, 6.6, 3.3), aEll(cx, cy+0.4, 5.7, 2.5))
	for side := -1.0; side <= 1; side += 2 {
		for k := 0.0; k < 4; k++ {
			a := 0.5 + 0.3*k
			tx, ty := cx+side*6.5*math.Sin(a), cy-1-3.2*math.Cos(a)
			nx, ny := side*math.Sin(a), -math.Cos(a)*0.8
			shell = shell.or(aPoly(tx-ny*0.55-nx*0.4, ty+nx*0.55-ny*0.4, tx+nx*0.85, ty+ny*0.85, tx+ny*0.55-nx*0.4, ty-nx*0.55-ny*0.4))
		}
	}
	q.body(shell, crabShell, look{round: 3.4})
	// The grooves round the middle, the halves either side swelling over
	// the gills, granules over the front.
	for side := -1.0; side <= 1; side += 2 {
		q.shade(aPath(cx+side*1.2, cy-2.8, 0.14, cx+side*2.1, cy-1.2, 0.16, cx+side*1.5, cy+1.1, 0.13), -0.13)
		q.shade(aEll(cx+side*4, cy-0.8, 1.5, 1.2), 0.04)
	}
	for k := 0; k < 12; k++ {
		gx := cx + (hashf(k, 3, 41)-0.5)*9.5
		gy := cy - 3.4 + hashf(k, 4, 41)*2
		if shell.d(gx*cK, gy*cK) < -0.6*cK {
			q.body(aEll(gx, gy, 0.2, 0.18), crabPalmTu, look{round: 0.2, soft: true})
		}
	}
	// Eyes on stalks in their sockets at the front edge.
	for side := -1.0; side <= 1; side += 2 {
		ex := cx + side*2.6
		q.shade(aEll(ex, cy-3.7, 0.95, 0.55), -0.28)
		q.body(aCap(ex, cy-3.7, ex+side*0.25, cy-5.3, 0.42, 0.38), crabStalk, look{round: 0.4})
		gx, gy := ex+side*0.3, cy-5.85
		q.flat(aEll(gx, gy, 0.86, 0.9), plain(hex(0x140e12)))
		q.flat(aEll(gx+0.15, gy+0.25, 0.55, 0.5), plain(hex(0x30262e)))
		q.dot(gx-0.35, gy-0.38, 0, 0, colWhite)
		q.dot(gx-0.35, gy-0.38, 1, 0, hex(0xd8d0d8))
	}
	if pinch {
		claws() // raised high in front
	}
	return q.done(0.85, 14, 11.5+3)
}
