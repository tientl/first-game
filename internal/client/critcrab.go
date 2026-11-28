package client

import (
	"math"
)

// The crab at HD density, painted (critpaint.go): seen from the front, its
// shell a hard, shining dome with grooves and granules and a toothed front
// rim, black eyes on stalks, two big claws with dark-tipped fingers, and
// three jointed legs a side splayed out past the shell, so it reads at its
// small size (crabSize).

var (
	crabShell  = newSurf(crabRamp, sShell, 0.9, 401)
	crabLimb   = newSurf(crabRamp, sShell, 0.6, 402)
	crabLegS   = newSurf(mkRamp(0x5a1a10, 0x8e2c18, 0xc04a26, 0xe07038, 0xf09a5a), sShell, 0.45, 403)
	crabUnder  = newSurf(ramp{hex(0x8a6448), hex(0xc0966a), hex(0xe2c497), hex(0xf6e6c4)}, sPlate, 0.2, 404)
	crabTip    = newSurf(ramp{hex(0x1e1412), hex(0x3a2622), hex(0x5e4036), hex(0x8a6a58)}, sShell, 0.8, 405)
	crabGrain  = newSurf(ramp{hex(0xb33e22), hex(0xd8622e), hex(0xf0904a), hex(0xfac890)}, sShell, 0.8, 409)
	crabStalk  = newSurf(ramp{hex(0x7e2416), hex(0xb33e22), hex(0xd8622e), hex(0xf09a5a)}, sMatte, 0.2, 406)
)

// crabHD is a crab seen from the front. Frames 1 to cycFrames of crabWalkA
// are its sideways scuttle; those of crabPinch its raised claws snapping.
func crabHD(pose, f int) (*canvas, float64, float64) {
	q := newCritZoom(32, 24, 2, 3, crabSize)
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

	// Legs: three a side, from under the shell's rim: the thigh up and out
	// to a high knee, the shank down, a pointed dark-tipped foot.
	for _, fi := range []float64{2, 1, 0} {
		for side := -1.0; side <= 1; side += 2 {
			dx, up := 0.0, 0.0
			if walking {
				dx, up = pace(ph+0.33*fi+0.25*(side+1), 0.55, 1.7, 1.3)
			}
			// fanned out: the front pair high and in, each pair behind it
			// lower and further out, so all three read apart
			bx, by := cx+side*(4.8-0.5*fi), cy+0.4+0.8*fi-bob
			kx, ky := cx+side*(8+1.1*fi)+dx*0.5, cy-2.3+1.2*fi-up-bob
			ax, ay := kx+side*(1.2+0.3*fi)+dx*0.35, ky+2.2-up*0.3
			fx, fy := ax+side*(0.5+0.2*fi)+dx*0.2, 14.9-up-0.5*fi
			b := 0.06 - 0.08*fi // the hind legs further back, in the shell's shade
			q.body(aCap(bx, by, kx, ky, 0.62, 0.5), crabLegS, look{round: 0.5, bias: b + 0.04})
			q.body(aCap(kx, ky, ax, ay, 0.5, 0.38), crabLegS, look{round: 0.4, bias: b})
			foot := aCap(ax, ay, fx, fy, 0.36, 0.1)
			q.body(foot, crabLegS, look{round: 0.34, bias: b - 0.02})
			q.coat(aCap(lerpf(ax, fx, 0.5), lerpf(ay, fy, 0.5), fx, fy, 0.4, 0.14), crabTip)
		}
	}

	// Claws: an arm forward and out, then the palm and its two fingers.
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
			// The palm points up and a little in: a broad hand and two
			// fingers, the outer fixed, the inner opening, each a curved
			// point bending in to meet the other, dark at the tip.
			tx, ty := -math.Sin(ang), -math.Cos(ang)
			ox, oy := -ty*side, tx*side // across the palm, outwards
			finger := func(across, spread, bias float64) shape {
				a := spread * across * side
				dx, dy := turnXY(tx, ty, 0, 0, a)
				bx, by := px+tx*1.2+ox*across*1.05, py+ty*1.2+oy*across*1.05
				mx, my := bx+dx*2.1, by+dy*2.1
				ex, ey := bx+dx*4.1-ox*across*0.6, by+dy*4.1-oy*across*0.6
				f := aPath(bx, by, 1.15, mx, my, 0.85, ex, ey, 0.18)
				q.body(f, crabLimb, look{round: 0.7, bias: bias})
				q.coat(f.and(aEll(ex, ey, 1.6, 1.6)), crabTip)
				return f
			}
			finger(1, gap*0.5, -0.04)
			finger(-1, gap*0.5, 0.04)
			hand := aEll(px, py+0.2, 2.6, 2.2)
			q.body(hand, crabShell, look{round: 1.8})
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
			q.body(aEll(gx, gy, 0.2, 0.18), crabGrain, look{round: 0.2, soft: true})
		}
	}
	// Eyes on stalks in their sockets at the front edge.
	for side := -1.0; side <= 1; side += 2 {
		ex := cx + side*2.6
		q.shade(aEll(ex, cy-3.7, 0.95, 0.55), -0.28)
		q.body(aCap(ex, cy-3.7, ex+side*0.3, cy-5.4, 0.5, 0.45), crabStalk, look{round: 0.45})
		gx, gy := ex+side*0.35, cy-6
		q.flat(aEll(gx, gy, 1.05, 1.05), plain(hex(0x140e12)))
		q.flat(aEll(gx+0.2, gy+0.3, 0.6, 0.55), plain(hex(0x3a2e38)))
		q.dot(gx-0.35, gy-0.38, 0, 0, colWhite)
		q.dot(gx-0.35, gy-0.38, 1, 0, hex(0xd8d0d8))
	}
	if pinch {
		claws() // raised high in front
	}
	return q.done(0.85, 14, 11.5+3)
}
