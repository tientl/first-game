package client

import (
	"image/color"
	"math"
)

// The tribal warrior at HD density (the cave spider is in critspider.go,
// the giant scorpion in critscorp.go).

func init() {
	set := func(name string, hd func(pose, f int) (*canvas, float64, float64), anim map[int]int) {
		critKinds[name].hd, critKinds[name].anim = hd, anim
	}
	set("spider", spiderHD, map[int]int{spiderWalkA: cycFrames, spiderBite: blowFrames, spiderSpit: blowFrames})
	set("scorpion", scorpionHD, map[int]int{scorpionWalkA: cycFrames, scorpionPinch: blowFrames, scorpionSting: blowFrames})
	set("tribe", tribeHD, map[int]int{tribeWalkA: cycFrames, tribeWalkB: cycFrames, tribeThrust: blowFrames})
}

// gloss shades a as hard, shining shell: clean bands and a bright glint
// where the light strikes it.
func gloss(r ramp, a shape, round, bias float64) shader {
	n := len(r)
	return func(x, y int, _ float64) color.RGBA {
		l := a.lum(x, y, round*cK)
		i := int((l*0.8 + bias) * float64(n))
		if l > 0.9 {
			i += 2
		}
		return r.at(max(0, min(n-1, i)))
	}
}

// --- the tribal warrior -----------------------------------------------------------------------

// tribeKey is the warrior's pose as the density-1 art had it.
func tribeKey(pose int) tribePose {
	const fx = 15.0
	q := tribePose{
		pelvis: [2]float64{fx + 0.5, 29}, chest: [2]float64{fx + 1, 21.5}, head: [2]float64{fx + 2, 14.5},
		nearLeg: [2][2]float64{{fx + 1.8, 35}, {fx + 1.5, 40}}, farLeg: [2][2]float64{{fx - 0.6, 35}, {fx - 1.2, 40}},
		nearArm: [2][2]float64{{fx + 2.5, 26.5}, {fx + 8.5, 26.5}}, farArm: [2][2]float64{{fx + 2.5, 25.5}, {fx + 4, 28}},
		butt: [2]float64{fx + 9.5, 40.5}, tip: [2]float64{fx + 9.5, 5},
	}
	switch pose {
	case tribeWalkA:
		q.nearLeg = [2][2]float64{{fx + 4, 34.5}, {fx + 5, 40}}
		q.farLeg = [2][2]float64{{fx - 1.6, 35}, {fx - 5, 39}}
		q.nearArm = [2][2]float64{{fx + 0.5, 25.5}, {fx + 4.5, 27}}
		q.farArm = [2][2]float64{{fx + 4, 25}, {fx + 6.5, 26.5}}
		q.butt, q.tip = [2]float64{fx - 0.5, 36}, [2]float64{fx + 13.5, 9}
	case tribeWalkB:
		q.pelvis[1], q.chest[1], q.head[1] = 28, 20.5, 13.5
		q.nearLeg = [2][2]float64{{fx + 0.2, 35}, {fx - 3.6, 39.2}}
		q.farLeg = [2][2]float64{{fx + 3.4, 34.5}, {fx + 4.6, 40}}
		q.nearArm = [2][2]float64{{fx + 1.5, 24.5}, {fx + 5.5, 25.5}}
		q.farArm = [2][2]float64{{fx - 0.5, 24.5}, {fx - 1.5, 27.5}}
		q.butt, q.tip = [2]float64{fx + 0.5, 35}, [2]float64{fx + 14.5, 8}
	case tribeThrust:
		q.pelvis, q.chest, q.head = [2]float64{fx, 30}, [2]float64{fx + 4, 23.5}, [2]float64{fx + 7.5, 17.5}
		q.nearLeg = [2][2]float64{{fx + 5.5, 34}, {fx + 6.5, 40}}
		q.farLeg = [2][2]float64{{fx - 4, 35.5}, {fx - 8.5, 39.6}}
		q.nearArm = [2][2]float64{{fx + 7.5, 26.6}, {fx + 12, 24.6}}
		q.farArm = [2][2]float64{{fx + 0.5, 26.5}, {fx + 1.5, 24.8}}
		q.butt, q.tip = [2]float64{fx - 7, 25}, [2]float64{fx + 24, 24}
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

// tribeHD draws a warrior side-on, facing right: black-skinned and
// bare-chested under white war paint, in a hide loincloth and a necklace of
// bones, a crest of hair with a red feather, a stone-tipped spear in his fist.
func tribeHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(52, 48, 4, 4)
	const gy = 40.0
	p := tribeKey(pose)
	switch {
	case (pose == tribeWalkA || pose == tribeWalkB) && f > 0:
		ph := cycPh(f)
		stride, lift := 7.5, 2.0
		if pose == tribeWalkB { // running
			stride, lift = 10.5, 3.2
		}
		p = lerpTribe(tribeKey(tribeWalkA), tribeKey(tribeWalkB), 0.5-0.5*math.Cos(2*math.Pi*ph))
		p.pelvis[1] = 28.4 + 0.6*math.Cos(4*math.Pi*ph)
		for i, l := range []*[2][2]float64{&p.nearLeg, &p.farLeg} {
			dx, up := pace(ph+0.5*float64(i), 0.6, stride, lift)
			hx, hy := p.pelvis[0]+1-2*float64(i), p.pelvis[1]+1
			fx, fy := hx+0.6+dx, gy-up
			kx, ky := ik2(hx, hy, fx, fy-1.2, 5.6, 5.6, -1)
			*l = [2][2]float64{{kx, ky}, {fx, fy}}
		}
	case pose == tribeThrust && f > 0:
		// Drawn back first: the weight on the back leg, the spear cocked.
		back := tribeKey(tribeStand)
		back.pelvis[0], back.chest[0], back.head[0] = back.pelvis[0]-1.5, back.chest[0]-2.5, back.head[0]-3
		back.nearLeg = [2][2]float64{{18.5, 34.5}, {20, 40}}
		back.farLeg = [2][2]float64{{11.5, 35}, {9.5, 40}}
		back.nearArm = [2][2]float64{{12.5, 25.5}, {8.5, 23}}
		back.farArm = [2][2]float64{{17.5, 25.5}, {21, 23.5}}
		back.butt, back.tip = [2]float64{-1, 22.6}, [2]float64{30, 23.8}
		p = lerpTribe(back, p, smooth((blowAmt(f)+1)/2))
	}
	skin := func(s shape, b float64) shader { return shd(tribeSkin, s, 1.5, b) }
	shoulder := [2]float64{p.chest[0] - 0.6, p.chest[1] - 1.4}
	leg := func(hipX float64, l [2][2]float64, b float64) {
		lg := aLeg(hipX, p.pelvis[1]+1, l[0][0], l[0][1], l[1][0], l[1][1]-1.5, 2.3, 1.85, 1.3)
		q.fill(lg, skin(lg, b))
		ft := aCap(l[1][0]-0.6, l[1][1]-0.5, l[1][0]+2.7, l[1][1]-0.3, 0.95, 0.7)
		q.fill(ft, skin(ft, b+0.05))
		if b >= 0 { // an anklet of white paint
			q.mark(aCap(l[1][0]-1.4, l[1][1]-2.9, l[1][0]+1.6, l[1][1]-2.9, 0.35, 0.35), plain(paintRamp.at(0)))
		}
	}
	arm := func(a [2][2]float64, b float64) {
		am := aLeg(shoulder[0], shoulder[1], a[0][0], a[0][1], a[1][0], a[1][1], 2.1, 1.55, 1.25)
		q.fill(am, skin(am, b))
		fist := aEll(a[1][0], a[1][1], 1.5, 1.5)
		q.fill(fist, skin(fist, b+0.08))
	}
	leg(p.pelvis[0]-1, p.farLeg, -0.25)
	arm(p.farArm, -0.25)
	leg(p.pelvis[0]+1, p.nearLeg, 0.02)

	// The spear: an ash shaft, a leaf of knapped stone lashed on with sinew,
	// and a red tassel under the head.
	dx, dy := p.tip[0]-p.butt[0], p.tip[1]-p.butt[1]
	n := math.Hypot(dx, dy)
	dx, dy = dx/n, dy/n
	shaft := aCap(p.butt[0], p.butt[1], p.tip[0], p.tip[1], 0.55, 0.5)
	q.fill(shaft, func(x, y int, d float64) color.RGBA {
		if d > -1.2 && (float64(x)*dy-float64(y)*dx) > (p.butt[0]*dy-p.butt[1]*dx)*cK {
			return spearShaft[0]
		}
		return spearShaft.at(1 + (x/3+y/3)%2)
	})
	bx, by := p.tip[0]-dx*0.5, p.tip[1]-dy*0.5
	at := func(u, v float64) (float64, float64) { return bx + dx*u - dy*v, by + dy*u + dx*v }
	var pts []float64
	for _, o := range [][2]float64{{0, 1.3}, {2.3, 2.1}, {6.8, 0}, {2.3, -2.1}, {0, -1.3}} {
		x, y := at(o[0], o[1])
		pts = append(pts, x, y)
	}
	blade := aPoly(pts...)
	q.fill(blade, func(x, y int, d float64) color.RGBA {
		u := (float64(x)/cK-bx)*-dy + (float64(y)/cK-by)*dx
		l := 0.5 + 0.22*math.Copysign(1, u) + (hashf(x/2, y/2, 95)-0.5)*0.3
		if math.Abs(u) < 0.2 {
			l = 0.95 // the ridge down the middle
		}
		return toneOf(stoneRamp, l)
	})
	for k := 0.3; k <= 2; k += 0.8 { // the lashing
		x0, y0 := at(-k, 0.75)
		x1, y1 := at(-k-0.3, -0.75)
		q.flat(aCap(x0, y0, x1, y1, 0.22, 0.22), plain(colBone))
	}
	tx, ty := at(-3, 0)
	q.fill(aCap(tx, ty, tx-0.6, ty+2.6, 0.6, 0.3), plain(hex(0xd8301c)))
	q.fill(aCap(tx-0.4, ty, tx-1.5, ty+2, 0.45, 0.25), plain(hex(0x9a1c12)))

	// The torso: a deep bare chest over a hard, narrow waist.
	lean := (p.chest[0] - p.pelvis[0]) / (p.pelvis[1] - p.chest[1])
	torso := sBlend(1.5, aPath(p.pelvis[0], p.pelvis[1], 3.1, lerpf(p.pelvis[0], p.chest[0], 0.5), lerpf(p.pelvis[1], p.chest[1], 0.5), 2.9,
		p.chest[0], p.chest[1], 4.1, p.chest[0]+lean*2, p.chest[1]-2.4, 2.8))
	q.fill(torso, skin(torso, 0.03))
	B := func(u, v float64) (float64, float64) { return p.chest[0] + u - lean*v, p.chest[1] + v }
	line := func(u0, v0, u1, v1, r float64, col color.RGBA) {
		x0, y0 := B(u0, v0)
		x1, y1 := B(u1, v1)
		q.mark(aCap(x0, y0, x1, y1, r, r), plain(col))
	}
	line(0.4, 1.7, 3.4, 1.3, 0.22, tribeSkin[0]) // the shadow under the chest
	line(2.2, 3.4, 2.4, 6.2, 0.18, tribeSkin[1]) // the belly's muscles
	line(1.2, 4.6, 3.2, 4.5, 0.18, tribeSkin[1])
	// War paint: slashes down the chest and dots on the belly.
	for k := 0.0; k < 3; k++ {
		line(0.6+1.3*k, -1.2+0.3*k, 1.2+1.3*k, 1+0.3*k, 0.3, paintRamp.at(1))
	}
	for k := 0.0; k < 3; k++ {
		line(1+0.9*k, 3.3+1.1*k, 1+0.9*k, 3.3+1.1*k, 0.36, paintRamp.at(1))
	}
	// A necklace of bones and teeth.
	for k := -1.5; k <= 4; k += 0.9 {
		sag := math.Max(0, 2-math.Abs(k-2)) * 0.6
		x, y := B(k, -3.3+sag)
		q.mark(aCap(x, y, x+0.1, y+0.9, 0.3, 0.14), plain(colBone))
	}
	// A loincloth of hide on a rope belt: a long flap before, a short behind.
	px, py := p.pelvis[0], p.pelvis[1]
	hide := func(x, y int, _ float64) color.RGBA {
		l := 0.78 - (float64(y)/cK-py)*0.08 + (hashf(x/2, y/2, 88)-0.5)*0.22
		if float64(y)/cK > py+3.8+1.5*hashf(x/2, 4, 88) {
			return loinRamp.at(3) // a trim of pale fur
		}
		return toneOf(loinRamp, l)
	}
	q.fill(aPoly(px-3.6, py-0.6, px+0.5, py-0.6, px-0.4, py+4.2, px-2.2, py+3.4, px-4.2, py+5), hide)
	q.fill(aPoly(px-0.5, py-0.6, px+4.2, py-0.6, px+4.8, py+5.6, px+3.2, py+4.8, px+1.8, py+6.4, px+0.2, py+5), hide)
	q.fill(aCap(px-3.8, py-0.7, px+4.2, py-0.7, 0.55, 0.55), func(x, y int, _ float64) color.RGBA { return thatchRamp.at(1 + (x/2)%2*2) })
	q.flat(aCap(px+2.2, py-0.2, px+2.2, py+1.2, 0.4, 0.3), plain(colBone))

	// The head: a crest of black hair, a heavy brow drawn down over a
	// glaring eye, paint across the cheek, teeth bared.
	hx, hy := p.head[0], p.head[1]
	neck := aCap(p.chest[0]+lean*2, p.chest[1]-2, hx-0.4, hy+3, 1.8, 1.7)
	q.fill(neck, skin(neck, -0.05))
	// A red feather, tied in at the back of the crest.
	q.fill(aCap(hx-2.6, hy-4, hx-7, hy-10.4, 0.7, 0.3), func(x, y int, _ float64) color.RGBA {
		return mix(hex(0xb81c14), hex(0xf86a30), clamp01((hy-4-float64(y)/cK)/6.4))
	})
	q.fill(aCap(hx-1.8, hy-4.6, hx-5.6, hy-10.2, 0.6, 0.25), func(x, y int, _ float64) color.RGBA {
		t := clamp01((hy - 4.6 - float64(y)/cK) / 5.6)
		return mix(hex(0xd8301c), hex(0xfce0a0), t*t)
	})
	hair := sBlend(0.8, aEll(hx-1, hy-1.6, 3.9, 3.9), aEll(hx-2.6, hy+1.4, 2, 2.6)).jag(-3.5, 4.5, -sFar, 97)
	q.fill(hair, func(x, y int, _ float64) color.RGBA { return tribeHair.at((x/2 + y/3) % 5 / 4) })
	for k := 0.0; k < 3; k++ { // the crest stands up in spikes
		q.fill(aPoly(hx-3+2*k, hy-4.4, hx-2.4+2*k, hy-6.6, hx-1.2+2*k, hy-4.6), plain(tribeHair.at(int(k)%2)))
	}
	faceS := sBlend(1, aEll(hx+0.9, hy+0.4, 3.3, 4.3), aEll(hx+3.4, hy+0.8, 1.1, 1), aEll(hx+1.6, hy+3.7, 2.1, 1.7))
	q.fill(faceS, skin(faceS, 0.1))
	q.flat(aCap(hx+0.4, hy-0.9, hx+4, hy-0.5, 0.42, 0.36), plain(tribeSkin[0])) // the brow
	q.flat(aEll(hx+2.6, hy+0.1, 0.6, 0.42), plain(hex(0xfff6e4)))
	q.dot(hx+2.9, hy+0.1, 0, 0, colInk)
	q.mark(aCap(hx-0.8, hy+1.5, hx+2.8, hy+1.3, 0.42, 0.36), plain(paintRamp.at(1))) // paint across the cheek
	q.mark(aCap(hx+1.2, hy+2.3, hx+1.4, hy+3.3, 0.3, 0.2), plain(paintRamp.at(1)))
	q.dot(hx+3.6, hy+0.9, 0, 0, hex(0xc8281a))
	q.flat(aCap(hx+1.6, hy+3.5, hx+3.6, hy+3.4, 0.45, 0.4), plain(hex(0x1a080a))) // the mouth
	q.flat(aCap(hx+2.3, hy+3.4, hx+3.6, hy+3.3, 0.2, 0.2), plain(colTooth))
	q.flat(aCap(hx-1.4, hy+0.6, hx-1.2, hy+1.6, 0.5, 0.4), plain(tribeSkin[2])) // the ear
	q.flat(aCap(hx-1.9, hy+2, hx-0.9, hy+2.4, 0.25, 0.2), plain(colBone))       // a bone through it

	// The near arm over it all, its fist round the shaft.
	arm(p.nearArm, 0.08)
	ex, ey := lerpf(shoulder[0], p.nearArm[0][0], 0.6), lerpf(shoulder[1], p.nearArm[0][1], 0.6)
	q.mark(aCap(ex-1.3, ey-0.6, ex+1.3, ey+0.6, 0.38, 0.38), plain(paintRamp.at(1)))
	return q.done(0.85, 15, 41)
}
