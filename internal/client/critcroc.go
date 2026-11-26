package client

import (
	"image/color"
	"math"
)

// The crocodile and its pond cousins at HD density, side-on and facing
// right: a long armoured body on four sprawled legs that step in diagonal
// pairs, a tail that sways with the walk, jaws that hinge open on rows of
// teeth. A bite is wound up first: the head drawn back, the jaws parting.

// crocHD is the look of one kind of crocodile.
type crocHD struct {
	skin, belly   ramp
	snout, r0, r1 float64 // the jaws: their length from the hinge and half their height at the hinge and at the tip
	girth         float64 // how stout the body is
	zoom          float64
}

func init() {
	looks := map[string]crocHD{
		"croc":    {muggerRamp, ext4(muggerBelly), 10, 1.45, 0.95, 1, 1},
		"brachy":  {brachyLook.skin, ext4(brachyLook.belly), 7.4, 1.8, 1.45, 1.08, brachyScale},
		"champso": {champsoLook.skin, ext4(champsoLook.belly), 11, 1, 0.5, 0.86, champsoScale},
	}
	for name, lk := range looks {
		critKinds[name].hd = func(p, f int) (*canvas, float64, float64) { return crocDrawHD(lk, p, f) }
		critKinds[name].anim = map[int]int{crocWalkA: cycFrames, crocWalkB: cycFrames, crocBite: blowFrames}
	}
}

// ext4 makes sure a ramp has at least four tones.
func ext4(r ramp) ramp {
	for len(r) < 4 {
		r = append(ramp{mix(r[0], colInk, 0.3)}, r...)
	}
	return r
}

// topOf is the art row where shape s begins at art column x, looking down
// from row y0; ok is false if it is not found within 12 rows.
func topOf(s shape, x, y0 float64) (float64, bool) {
	for y := y0; y < y0+12; y += 1.0 / cK {
		if s.d(x*cK, y*cK) < 0 {
			return y, true
		}
	}
	return 0, false
}

func crocDrawHD(lk crocHD, pose, f int) (*canvas, float64, float64) {
	if pose == crocLurk {
		return crocLurkHD(lk)
	}
	q := newCritZoom(53, 25, 3, 4, lk.zoom)
	const gy = 15.1 // the ground
	g := lk.girth
	ph, walking, stride := 0.0, false, 4.2
	switch {
	case pose == crocWalkA && f > 0:
		ph, walking = cycPh(f), true
	case pose == crocWalkB:
		ph, walking = 0.5, true
		if f > 0 { // the scrambling rush
			ph, stride = cycPh(f), 5.6
		}
	}
	bite := pose == crocBite
	amt, open, hx0 := 0.0, 0.0, 0.0
	if bite {
		amt = blowAmt(f)
		open = lerpf(0.22, 1, clamp01((amt+1)/1.6))
		hx0 = lerpf(-1.8, 1.2, (amt+1)/2)
	}
	sway := 0.45
	if walking {
		sway = math.Sin(2 * math.Pi * ph)
	}
	if bite {
		sway = -0.8 * amt
	}

	var tp []tpt
	for i := 0; i <= 12; i++ {
		t := float64(i) / 12
		tp = append(tp, tpt{1.2 + 15.5*t, 13.3 - 2*t + sway*1.3*math.Sin(2.6*(1-t))*(1-t), 0.45 + 2.5*g*smooth(t)})
	}
	tail := newCTube(tp).shape()
	body := sBlend(2.2, tail, aEll(24+hx0*0.3, 10.7, 8.4, 3.9*g), aEll(33+hx0*0.7, 10.3-0.5*open, 3.5, 2.9*g))
	hide := func(s shape, bias float64) shader {
		sc := scaly(lk.skin, s, 2.6, bias, 4, 3, 91)
		return func(x, y int, d float64) color.RGBA {
			ax, ay := (float64(x)+0.5)/cK, (float64(y)+0.5)/cK
			if ay > 13.05 && ax > 9 { // the pale belly, in plates
				l := s.lum(x, y, 2.4*cK) + 0.22
				if x%4 == 0 {
					l -= 0.14
				}
				return toneOf(lk.belly, l)
			}
			col := sc(x, y, d)
			if ax < 21 && math.Mod(ax+0.25*ay, 3.6) < 1.3 && ay < 12.4 {
				col = mix(col, lk.skin[0], 0.3) // dark bands down the tail
			}
			return col
		}
	}
	leg := func(hx, hy, off, size float64, near bool, bend float64) {
		dx, up := 0.0, 0.0
		if walking {
			dx, up = pace(ph+off, 0.62, stride, 1.4)
		}
		fx, fy := hx+0.9+dx, gy-up
		kx, ky := ik2(hx, hy, fx, fy, 2.7*size, 2.7*size, bend)
		lg := aLeg(hx, hy, kx, ky, fx, fy, 1.55*size, 1.15*size, 0.8*size).or(aCap(fx, fy, fx+1.7*size, fy+0.25, 0.75*size, 0.5*size))
		bias := -0.2
		if near {
			bias = -0.02
		}
		q.fill(lg, scaly(lk.skin, lg, 1.2, bias, 3, 3, 93))
		for k := 0.0; k < 3; k++ { // claws
			q.dot(fx+1.2*size+0.5*k, fy+0.5, 0, 0, colBone)
		}
	}
	leg(19.6, 12.2, 0.55, 1, false, -1)
	leg(30.4, 12.1, 0.05, 0.88, false, 1)
	q.fill(body, hide(body, 0))

	// The jaws, hinged behind the eye.
	hx, hy := 34.5+hx0, 10.9-0.5*open
	L := lk.snout
	up, down := -0.46*open, 0.2*open
	lower := aCap(hx-0.6, hy+1.25, hx+L-0.7, hy+1.15, lk.r0*0.75, lk.r1*0.7).turned(hx, hy, down)
	q.fill(lower, func(x, y int, _ float64) color.RGBA { return toneOf(lk.belly, lower.lum(x, y, 1.2*cK)+0.12) })
	if open > 0.1 {
		ux, uy := turnXY(hx+L-0.8, hy, hx, hy, up)
		lx, ly := turnXY(hx+L-1, hy+0.6, hx, hy, down)
		q.flat(aPoly(hx-0.8, hy+0.2, ux, uy, lx, ly), func(x, y int, _ float64) color.RGBA {
			return mix(hex(0x6e1c26), colMouth, clamp01(((float64(x)+0.5)/cK-hx)/(L*0.7)))
		})
	}
	if open <= 0.1 { // the line of the shut mouth, for the teeth to show against
		q.flat(aCap(hx+0.4, hy+0.55, hx+L-0.4, hy+0.6, 0.45, 0.4), plain(mix(lk.skin[0], colInk, 0.5)))
	}
	upper := sBlend(0.7, aCap(hx-0.6, hy-0.95, hx+L, hy-0.3, lk.r0*1.1, lk.r1), aEll(hx+L-0.5, hy-1.05, 1.05, 0.75), aEll(hx+0.9, hy-2.4, 1.45, 1.25)).turned(hx, hy, up)
	q.fill(upper, scaly(lk.skin, upper, 1.6, 0.03, 4, 3, 95))
	teeth := 0.8 + 0.3*open
	for k := 1.4; k < L-0.6; k += 1.25 {
		ty := hy - 0.3 + lerpf(lk.r0*1.1, lk.r1, k/L) - 0.3
		q.fill(aPoly(hx+k-0.3, ty, hx+k+0.3, ty, hx+k, ty+teeth).turned(hx, hy, up), plain(colTooth))
		if open > 0.1 {
			by := hy + 1.2 - lerpf(lk.r0, lk.r1, k/L)*0.7 + 0.2
			q.fill(aPoly(hx+k+0.3, by, hx+k+0.9, by, hx+k+0.6, by-teeth*0.8).turned(hx, hy, down), plain(colTooth))
		}
	}
	ex, ey := turnXY(hx+1, hy-2.5, hx, hy, up)
	q.eye(ex, ey, 1.6, crocEye)
	nx, ny := turnXY(hx+L-0.3, hy-1.2, hx, hy, up)
	q.dot(nx, ny, 0, 0, lk.skin[0])

	// The scutes stand up along the back and the tail.
	for x := 4.0; x <= 31.5; x += 1.55 {
		if y, ok := topOf(body, x, 2); ok {
			h := 0.75 + 0.5*clamp01((x-4)/10)
			col := lk.skin.at(3 + int(x/1.55)%2)
			q.fill(aPoly(x-0.75, y+0.5, x-0.1, y-h, x+0.85, y+0.5), plain(col))
		}
	}
	leg(18.2, 12.7, 0, 1, true, -1)
	leg(29.6, 12.8, 0.5, 0.88, true, 1)
	return q.done(0.82, 22, 14)
}

// crocLurkHD is the crocodile waiting under water: the eye on its knob, the
// top of the skull and snout, the nostrils, and a few scutes of the back
// breaking the surface behind.
func crocLurkHD(lk crocHD) (*canvas, float64, float64) {
	q := newCritZoom(23, 8, 1, 1, lk.zoom)
	L := lk.snout
	water := sAbove(5.6 * cK)
	head := sBlend(0.8, aEll(8.5, 5.2, 2.5, 2.2), aEll(8.3, 3, 1.25, 1.15), aCap(9.5, 5.3, 8.4+L, 5.5, 1.15, 0.85), aEll(8+L, 4.7, 0.95, 0.75)).and(water)
	q.fill(head, func(x, y int, d float64) color.RGBA {
		l := head.lum(x, y, 1.5*cK)
		if float64(y)/cK > 4.9 {
			l -= 0.2 // wet at the waterline
		}
		return toneOf(lk.skin, l+0.05)
	})
	for i, x := range [3]float64{1.4, 3.3, 5.3} {
		q.fill(aPoly(x-0.9, 5.6, x, 4.4-0.25*float64(i), x+0.9, 5.6), plain(lk.skin.at(2+i%2)))
	}
	q.eye(8.5, 2.9, 1.5, crocEye)
	q.dot(8+L, 4.3, 0, 0, lk.skin[0])
	return q.done(0.7, 10, 5)
}
