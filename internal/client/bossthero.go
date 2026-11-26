package client

import (
	"image/color"
	"math"
)

// The three great meat-eaters among the bosses (Spinosaurus,
// Carcharodontosaurus, Giganotosaurus) are one skeleton in three looks:
// a chain of knobs from the tip of the tail to the back of the skull, two
// legs bent by their joints, small arms, and a head whose jaws turn about
// their hinge. A pose is a handful of angles; the look gives the sizes,
// the hide and whatever stands on the back (a sail, a row of spikes).

// The parts of a theropod, for its hide.
const (
	theroBody = iota
	theroLeg
	theroArm
	theroHead
	theroJaw
)

type theroLook struct {
	skin, belly, mark ramp
	eye               color.RGBA
	leg, torso        float64 // the height of the hips, and from hips to shoulders
	rHip, rBelly      float64
	rChest, rThigh    float64
	neck, rNeck       float64
	neckA, headA      float64 // how the neck and the head are carried, radians (- is up)
	skull, jaw        []knob  // in the head's frame: u along the snout from the hinge, v down
	eyeU, eyeV        float64
	tooth, toothStep  float64
	tail              float64
	arm, rArm         float64
	colour            func(lk *theroLook, x, y int, l float64, q bossPt, part int) color.RGBA
	back              func(c *canvas, lk *theroLook, body []knob, hip int) // what stands on the back, drawn before the body
	crest             func(c *canvas, lk *theroLook, fr bossFrame)         // on the skull, drawn after it
}

type theroPose struct {
	body, neck, head  float64   // the body's pitch (+ nose down), and the neck and head turned from how they are carried
	gape              float64   // 0 shut .. 1 wide
	tailUp, tailCurl  float64   // the tail's root and its tip, radians up
	tailShape         []float64 // or each of its seven lengths, radians up from straight back; drawn over the legs
	nearDX, farDX     float64   // the feet, ahead of the hips
	nearLift, farLift float64
	crouch            float64 // the share of their height the hips sink
	arm               float64 // the arms swung forward, radians
	water             bool    // out of the water: sunk to the hips, no legs
	dust              bool    // kicking up dust behind
	sweep             bool    // the arc its tail cuts through the air
	tailLen           float64 // the tail drawn this many times its length (swung towards us); 0 is 1
}

func (lk *theroLook) pelt(part int, bias float64) bossPelt {
	return func(x, y int, q bossPt) color.RGBA {
		l := 0.1 + q.l*0.82 + bias + bossScales(x, y, 911)
		if part == theroBody {
			l -= max(0, q.v-0.2) * 0.16 // the underside in its own shade
		}
		return lk.colour(lk, x, y, l, q, part)
	}
}

// theroCanvas draws pose p of look lk, with its anchor under its hips.
func theroCanvas(lk *theroLook, p theroPose) (*canvas, float64, float64) {
	const W, H, gy, hx = 340, 260, 210.0, 170.0
	c := newCanvas(W, H)
	hy := gy - lk.leg*(1-p.crouch)
	a := p.body
	dx, dy := math.Cos(a), math.Sin(a)

	// The tail, from the hips out to its tip.
	const tn = 7
	var tail []knob
	tx, ty := hx-dx*lk.rHip*0.3, hy-dy*lk.rHip*0.3
	r0 := lk.rHip * 0.82
	tl := lk.tail
	if p.tailLen > 0 {
		tl *= p.tailLen
	}
	for i := range tn {
		f := float64(i+1) / tn
		up := p.tailUp + p.tailCurl*f*f
		if p.tailShape != nil {
			up = p.tailShape[i]
		}
		ang := math.Pi + a*0.6 + up
		tx += math.Cos(ang) * tl / tn
		ty += math.Sin(ang) * tl / tn
		r := lerpf(r0, 1.3, math.Pow(f, 0.85))
		tail = append(tail, knob{tx, ty - (r0-r)*0.4, r})
	}
	// The body: tail, hips, belly, shoulders, neck.
	var body []knob
	if p.tailShape == nil {
		for i := tn - 1; i >= 0; i-- {
			body = append(body, tail[i])
		}
	}
	hip := len(body)
	sx, sy := hx+dx*lk.torso, hy+dy*lk.torso
	na := lk.neckA + p.neck + a*0.5
	n1x, n1y := sx+math.Cos(na-0.14)*lk.neck*0.5, sy+math.Sin(na-0.14)*lk.neck*0.5
	bx, by := n1x+math.Cos(na+0.14)*lk.neck*0.5, n1y+math.Sin(na+0.14)*lk.neck*0.5
	body = append(body,
		knob{hx, hy, lk.rHip},
		knob{hx + dx*lk.torso*0.5 - dy*1.5, hy + dy*lk.torso*0.5 + dx*1.5, lk.rBelly},
		knob{sx, sy, lk.rChest},
		knob{n1x, n1y, lerpf(lk.rChest, lk.rNeck, 0.72)},
		knob{bx, by, lk.rNeck})

	leg := func(fdx, lift, bias, seam float64) {
		L := lk.leg
		px, py := hx, hy+lk.rHip*0.2
		fx, fy := hx+fdx, gy-3-lift
		ax, ay := fx-0.11*L, fy-0.23*L
		bone := 0.42 * L
		ex, ey := ax-px, ay-py
		d := math.Hypot(ex, ey)
		h := math.Sqrt(max(0, bone*bone-d*d/4))
		kx, ky := px+ex/2+ey/d*h, py+ey/2-ex/d*h
		rt := lk.rThigh
		droop := lift * 0.5
		bossPaint(c, seam, lk.pelt(theroLeg, bias),
			[]knob{{px, py, rt}, {lerpf(px, kx, 0.55), lerpf(py, ky, 0.55), rt * 0.9}, {kx, ky, rt * 0.66}, {lerpf(kx, ax, 0.45) - 1, lerpf(ky, ay, 0.45), rt * 0.56}, {ax, ay, rt * 0.4}, {fx, fy, rt * 0.36}},
			[]knob{{fx, fy, rt * 0.33}, {fx + 0.13*L, fy + 0.6 + droop*0.5, rt * 0.25}, {fx + 0.21*L, fy + 1 + droop, rt * 0.16}},
			[]knob{{fx, fy, rt * 0.3}, {fx - 0.07*L, fy + 0.8, rt * 0.14}})
		claw := mix(hornRamp[0], colInk, 0.45+max(0, -bias))
		for _, q := range [][2]float64{{0.21*L + 1, 1.4 + droop}, {0.22*L + 2, 2 + droop}, {0.12 * L, 2.2 + droop*0.5}, {-0.07*L - 1, 1.8}} {
			c.put(int(fx+q[0]), int(fy+q[1]), claw)
		}
	}
	arm := func(bias, seam, off float64) {
		ox, oy := sx-dx*lk.rChest*0.15+off, sy+lk.rChest*0.5
		e1, e2 := rotPt(-p.arm, -0.22*lk.arm, 0.6*lk.arm)
		h1, h2 := rotPt(-p.arm, 0.5*lk.arm, 0.38*lk.arm)
		ex, ey := ox+e1, oy+e2
		hx2, hy2 := ex+h1, ey+h2
		bossPaint(c, seam, lk.pelt(theroArm, bias),
			[]knob{{ox, oy, lk.rArm}, {ex, ey, lk.rArm * 0.8}, {hx2, hy2, lk.rArm * 0.6}})
		claw := mix(hornRamp[1], colInk, 0.25+max(0, -bias))
		c.put(int(hx2+lk.rArm*0.6+0.5), int(hy2+0.5), claw)
		c.put(int(hx2+lk.rArm*0.6), int(hy2+1.5), claw)
		c.put(int(hx2), int(hy2+lk.rArm*0.6+0.5), claw)
	}

	if !p.water {
		leg(p.farDX, p.farLift, -0.16, 0)
	}
	arm(-0.2, 0, 3)
	if lk.back != nil {
		lk.back(c, lk, body, hip)
	}
	bossPaint(c, 0.22, lk.pelt(theroBody, 0), body)
	if !p.water {
		leg(p.nearDX, p.nearLift, 0.07, 0.3)
	}

	// The head: the skull and the jaw turn apart about the hinge.
	ha := lk.headA + p.head + a*0.3
	hgx, hgy := bx+math.Cos(ha)*lk.rNeck*0.2, by+math.Sin(ha)*lk.rNeck*0.2+lk.rNeck*0.25
	fr := bossFrame{ox: hgx, oy: hgy, a: ha - p.gape*0.32}
	fj := bossFrame{ox: hgx, oy: hgy, a: ha + p.gape*0.62}
	tipU, tipJ := lk.skull[len(lk.skull)-1][0], lk.jaw[len(lk.jaw)-1][0]
	if p.gape > 0 {
		ux, uy := fr.at(tipU, 0.4)
		jx, jy := fj.at(tipJ, 0)
		bkx, bky := fr.at(-3, 0)
		b2x, b2y := fj.at(-3, 1)
		c.polygon([]float64{bkx, bky, ux, uy, jx, jy, b2x, b2y}, func(x, y int) color.RGBA {
			d := math.Hypot(float64(x)+0.5-hgx, float64(y)+0.5-hgy)
			return mix(colGullet, colMouth, clamp01((d-tipU*0.2)/(tipU*0.5))-dither(x, y)*0.2)
		})
	}
	bossPaint(c, 0.25, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if u > -1 && v < -0.2 {
			return color.RGBA{}
		}
		if u > 0 && v < 1 {
			if p.gape > 0 {
				return mix(colMouth, hex(0xe08a88), 0.35) // the gum
			}
			return lk.skin[0] // the line of the mouth
		}
		return lk.pelt(theroJaw, 0)(x, y, q)
	}, fj.knobs(lk.jaw))
	if p.gape > 0.3 { // the tongue
		fj.poly(c, [][2]float64{{1, -0.2}, {tipJ * 0.45, -2.2}, {tipJ * 0.72, -1.6}, {tipJ * 0.6, -0.2}}, func(x, y int) color.RGBA {
			_, v := fj.local(x, y)
			return mix(hex(0xe58a94), hex(0xa83a52), clamp01((v+2)/2))
		})
	}
	bossPaint(c, 0.3, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > -1 && v > 0.6 {
			return color.RGBA{}
		}
		return lk.pelt(theroHead, 0)(x, y, q)
	}, fr.knobs(lk.skull))
	// Teeth: the upper row always shows, the lower when the jaws part.
	tooth := func(f bossFrame, u, v, n float64) {
		x0, y0 := f.at(u, v)
		x1, y1 := f.at(u+0.3*n/math.Abs(n), v+n)
		c.line(x0, y0, x1, y1, func(x, y int, t float64) color.RGBA {
			if t > 0.7 {
				return hornRamp[2]
			}
			return colTooth
		})
	}
	for i, u := 0, lk.toothStep*1.4; u < tipU; i, u = i+1, u+lk.toothStep {
		n := lk.tooth * (0.55 + 0.45*math.Sin(clamp01(u/tipU)*math.Pi)) * (0.8 + 0.2*float64(i%2))
		if p.gape == 0 {
			n *= 0.72
		}
		tooth(fr, u, 0.6, n)
	}
	if p.gape > 0.15 {
		for i, u := 0, lk.toothStep*1.9; u < tipJ-1; i, u = i+1, u+lk.toothStep {
			n := lk.tooth * 0.8 * (0.55 + 0.45*math.Sin(clamp01(u/tipJ)*math.Pi)) * (0.8 + 0.2*float64(i%2))
			tooth(fj, u, -0.2, -n)
		}
	}
	// The eye under its brow, and the nostril.
	ex, ey := fr.at(lk.eyeU, lk.eyeV)
	ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
	for q := -1; q <= 1; q++ {
		for w := -1; w <= 2; w++ {
			c.put(ix+w, iy+q, lk.skin[0])
		}
	}
	c.put(ix, iy, lk.eye)
	c.put(ix+1, iy, colInk)
	c.put(ix, iy-1, mix(lk.eye, colWhite, 0.5))
	b0x, b0y := fr.at(lk.eyeU-3.5, lk.eyeV-3)
	b1x, b1y := fr.at(lk.eyeU+4, lk.eyeV-1.6)
	bossLine(c, b0x, b0y, b1x, b1y, lk.skin[1])
	tk := lk.skull[len(lk.skull)-1]
	nx, ny := fr.at(tk[0]-1, tk[1]-tk[2]*0.35)
	c.put(int(nx), int(ny), lk.skin[0])
	c.put(int(nx)-1, int(ny), lk.skin[0])
	if lk.crest != nil {
		lk.crest(c, lk, fr)
	}
	arm(0, 0.3, 0)
	if p.tailShape != nil {
		bossPaint(c, 0.4, lk.pelt(theroBody, 0.02), append([]knob{{hx - dx*lk.rHip*0.3, hy, r0}}, tail...))
	}

	c.outline(0.78)
	if p.sweep {
		// The arc the tail has just cut, round the near side.
		R := lk.tail * 0.78
		for i := 0; i <= 500; i++ {
			f := float64(i) / 500
			an := math.Pi - 0.75 + f*1.75
			th := 4.5 * math.Sin(f*math.Pi) * (0.3 + 0.7*f)
			for k := 0.0; k <= th; k += 0.5 {
				qx := hx - 4 + math.Cos(an)*(R+k)
				qy := hy - 4 + math.Sin(an)*(R+k)
				if !c.opaque(int(qx), int(qy)) {
					c.put(int(qx), int(qy), withAlpha(mix(gigaEmber[3], hex(0xfff4d0), clamp01(f*1.2-k/8)), 235))
				}
			}
		}
	}
	ax, ay := hx+2, gy-1
	if p.water {
		wy := int(hy + lk.rHip*0.55)
		sinkBelow(c, wy)
		seaRing(c, hx+lk.torso*0.3, wy, lk.torso*0.95, 4, 951)
		// Water thrown up round the chest.
		for i := 0; i < 26; i++ {
			sxp := hx - lk.torso*0.6 + hashf(i, 0, 952)*lk.torso*2.1
			h := 3 + hashf(i, 1, 952)*hashf(i, 2, 952)*26
			col := colFoam
			if i%3 == 0 {
				col = colFoamDim
			}
			c.put(int(sxp), wy-int(h), col)
			if i%4 == 0 {
				c.put(int(sxp), wy-int(h)+1, colFoamDim)
			}
		}
		ax, ay = hx+lk.torso*0.3, float64(wy)
	} else {
		bossClearBelow(c, int(gy))
	}
	if p.dust {
		for i := 0; i < 40; i++ {
			qx := hx - lk.leg*0.2 - hashf(i, 0, 953)*lk.tail*0.8
			qy := gy - 1 - hashf(i, 1, 953)*hashf(i, 2, 953)*14
			if !c.opaque(int(qx), int(qy)) {
				c.put(int(qx), int(qy), withAlpha(mix(hex(0xd9c08a), hex(0xf2e6c0), hashf(i, 3, 953)), 220))
			}
		}
	}
	return bossCrop(c, ax, ay)
}

// theroRidge stands a row of spikes along the back of body between knobs
// from and to: each h tall on a base w wide, every step pixels.
func theroRidge(c *canvas, body []knob, from, to int, step, w, h float64, col func(x, y int, up float64) color.RGBA) {
	carry := 0.0
	for i := from; i < to && i+1 < len(body); i++ {
		a, b := body[i], body[i+1]
		l := math.Hypot(b[0]-a[0], b[1]-a[1])
		if l == 0 {
			continue
		}
		ux, uy := (b[0]-a[0])/l, (b[1]-a[1])/l
		nx, ny := uy, -ux // up, off the back
		for d := carry; d < l; d += step {
			t := d / l
			r := lerpf(a[2], b[2], t)
			k := 0.55 + 0.45*clamp01(r/8) // smaller where the body thins
			bx, by := a[0]+ux*d+nx*(r-1.5), a[1]+uy*d+ny*(r-1.5)
			tx, ty := bx+nx*(h*k+1.5)-ux*w*0.5, by+ny*(h*k+1.5)-uy*w*0.5
			c.polygon([]float64{bx - ux*w*0.6, by - uy*w*0.6, tx, ty, bx + ux*w*0.6, by + uy*w*0.6}, func(x, y int) color.RGBA {
				up := ((float64(x)+0.5-bx)*nx + (float64(y)+0.5-by)*ny) / (h*k + 1.5)
				return col(x, y, up)
			})
		}
		carry = math.Mod(carry-l, step)
		if carry < 0 {
			carry += step
		}
	}
}

// --- Giganotosaurus: ash and embers -------------------------------------------

var (
	gigaSkin  = mkRamp(0x151318, 0x221f26, 0x322e36, 0x464149, 0x5d575d, 0x787174, 0x968e8c)
	gigaBelly = mkRamp(0x332a2a, 0x4e4040, 0x6e5c56, 0x90796c, 0xb09a86)
	gigaEmber = mkRamp(0x7a1a0c, 0xb8300f, 0xe85a14, 0xff8f1e, 0xffc440, 0xfff0a0)
)

func gigaColour(lk *theroLook, x, y int, l float64, q bossPt, part int) color.RGBA {
	switch part {
	case theroJaw:
		return lk.belly.soft(l*0.9+0.08, x, y)
	case theroBody:
		// Cracks of fire down the back, widest on the spine.
		band := math.Mod(q.s+q.v*4+200, 13) / 13
		w := 0.17 * clamp01((0.3-q.v)/0.8)
		if q.t > 0.64 {
			w *= clamp01((-0.1 - q.v) * 3) // only along the nape
		}
		if d := math.Abs(band - 0.5); d < w && q.s > 12 {
			return gigaEmber.pick(0.95-d/w*0.75-max(0, q.v)*0.6, x, y)
		}
		if q.v > 0.5+dither(x, y)*0.25 {
			return lk.belly.soft(l+0.12, x, y)
		}
	case theroLeg:
		if q.chain == 0 && q.s > 5 && q.s < 30 {
			band := math.Mod(q.s+q.v*3, 9) / 9
			if d := math.Abs(band - 0.5); d < 0.13 && q.v < 0.45 {
				return gigaEmber.pick(0.62-d*2.5, x, y)
			}
		}
	case theroHead:
		// A smouldering streak from the brow down the snout.
		if q.v < -0.25 && q.v > -0.62 && q.s > 12 && int(q.s)%7 < 4 {
			return gigaEmber.pick(0.55+hashf(x, y, 915)*0.3, x, y)
		}
	}
	return lk.skin.soft(l, x, y)
}

var gigaLook = theroLook{
	skin: gigaSkin, belly: gigaBelly, mark: gigaEmber, eye: hex(0xffb020),
	leg: 62, torso: 32, rHip: 19, rBelly: 20, rChest: 16, rThigh: 14,
	neck: 25, rNeck: 9.5, neckA: -0.95, headA: 0.1,
	skull: []knob{{-4, -9.5, 9}, {6, -10, 9}, {15, -8, 7.6}, {24, -6.2, 6}, {31, -5, 4.8}},
	jaw:   []knob{{-3, 3.6, 5.6}, {8, 3.8, 5}, {19, 3.2, 4}, {28.5, 2.6, 3}},
	eyeU:  5, eyeV: -12, tooth: 4.4, toothStep: 2.6,
	tail: 68, arm: 10, rArm: 2.8,
	colour: gigaColour,
	back: func(c *canvas, lk *theroLook, body []knob, hip int) {
		theroRidge(c, body, 2, len(body)-1, 6, 4.4, 5.5, func(x, y int, up float64) color.RGBA {
			if up < 0.45 {
				return gigaEmber.pick(0.75-up, x, y) // glowing at the root
			}
			return gigaSkin.pick(0.3, x, y)
		})
	},
	crest: func(c *canvas, lk *theroLook, fr bossFrame) {
		// A horned brow over the eye.
		fr.poly(c, [][2]float64{{1, -17.5}, {4.5, -22}, {8, -17.2}}, func(x, y int) color.RGBA { return gigaSkin.pick(0.35, x, y) })
		fr.poly(c, [][2]float64{{8.5, -17}, {11, -20}, {13.5, -15.6}}, func(x, y int) color.RGBA { return gigaSkin.pick(0.3, x, y) })
	},
}

// --- Carcharodontosaurus: sand and rust ---------------------------------------

var (
	carchSkin  = mkRamp(0x432a18, 0x664426, 0x8c6234, 0xb28546, 0xd3aa62, 0xecca88, 0xf8e2b0)
	carchRust  = mkRamp(0x3c0f0a, 0x5e1a0e, 0x842a12, 0xa83f1a, 0xc65a26, 0xdc7a3c)
	carchBelly = mkRamp(0x8e7850, 0xb8a474, 0xdccc9c, 0xf2e8c4, 0xfdf8e4)
)

func carchColour(lk *theroLook, x, y int, l float64, q bossPt, part int) color.RGBA {
	switch part {
	case theroJaw:
		return lk.belly.soft(l*0.9+0.1, x, y)
	case theroBody:
		if q.v > 0.42+dither(x, y)*0.25 {
			return lk.belly.soft(l+0.1, x, y)
		}
		// A rust saddle along the spine, running down into tiger stripes.
		band := math.Mod(q.s-q.v*5+200, 10) / 10
		edge := -0.5 + 0.9*clamp01(1-math.Abs(band-0.5)*3.4)
		if q.v < edge && q.s > 8 {
			return lk.mark.soft(l*0.95-0.02, x, y)
		}
	case theroLeg:
		if q.chain == 0 && q.s < 34 {
			band := math.Mod(q.s+q.v*2, 8) / 8
			if band < 0.3 && q.v < 0.5 {
				return lk.mark.soft(l, x, y)
			}
		}
	case theroHead:
		if q.v < -0.42 {
			return lk.mark.soft(l+0.05, x, y) // the rust runs on over the skull
		}
	}
	return lk.skin.soft(l+0.04, x, y)
}

var carchLook = theroLook{
	skin: carchSkin, belly: carchBelly, mark: carchRust, eye: hex(0xf6e23a),
	leg: 56, torso: 31, rHip: 14.5, rBelly: 14.6, rChest: 12, rThigh: 11,
	neck: 23, rNeck: 7.2, neckA: -0.72, headA: 0.12,
	skull: []knob{{-3.5, -7, 6.6}, {6, -7.4, 6.5}, {16, -5.8, 5.2}, {25, -4.6, 4.2}, {31.5, -3.8, 3.3}},
	jaw:   []knob{{-3, 2.8, 4.2}, {9, 2.9, 3.6}, {20, 2.4, 2.9}, {29, 2, 2.2}},
	eyeU:  5.5, eyeV: -9, tooth: 3.6, toothStep: 1.9,
	tail: 62, arm: 9, rArm: 2.4,
	colour: carchColour,
	back: func(c *canvas, lk *theroLook, body []knob, hip int) {
		theroRidge(c, body, 3, len(body)-1, 3.2, 2.4, 2.2, func(x, y int, up float64) color.RGBA {
			return carchRust.pick(0.25+up*0.3, x, y)
		})
	},
	crest: func(c *canvas, lk *theroLook, fr bossFrame) {
		// The ridge over the eye, rust red.
		fr.poly(c, [][2]float64{{0.5, -13.2}, {5, -16.2}, {11, -13}}, func(x, y int) color.RGBA { return carchRust.pick(0.7, x, y) })
	},
}

// --- Spinosaurus: the sail ----------------------------------------------------

var (
	spinoSkin  = mkRamp(0x0c2428, 0x143a3a, 0x1e524b, 0x2c6d5c, 0x42896c, 0x62a67e, 0x8cc096)
	spinoDark  = mkRamp(0x08161c, 0x0e2428, 0x153434, 0x1e4640)
	spinoBelly = mkRamp(0x84804e, 0xb0ab6a, 0xd6d08c, 0xeee8b4, 0xfaf6d8)
	spinoSail  = mkRamp(0x4c1020, 0x7c1c26, 0xb03028, 0xdc5a28, 0xf48c30, 0xfcc050, 0xfff0a0)
)

func spinoColour(lk *theroLook, x, y int, l float64, q bossPt, part int) color.RGBA {
	switch part {
	case theroJaw:
		return lk.belly.soft(l*0.9+0.1, x, y)
	case theroBody:
		if q.v > 0.38+dither(x, y)*0.25 {
			l2 := l + 0.1
			if int(q.s)%4 == 0 {
				l2 -= 0.1 // belly plates
			}
			return lk.belly.soft(l2, x, y)
		}
		band := math.Mod(q.s+q.v*3+200, 9) / 9
		if w := 0.26 * clamp01((0.25-q.v)/0.9); math.Abs(band-0.5) < w {
			return spinoDark.soft(l+0.1, x, y)
		}
	case theroLeg:
		if q.chain == 0 && q.s < 28 && math.Mod(q.s+q.v*2, 7) < 2.4 && q.v < 0.4 {
			return spinoDark.soft(l+0.1, x, y)
		}
	case theroHead:
		if q.v < -0.5 && int(q.s)%6 < 3 {
			return spinoDark.soft(l+0.15, x, y)
		}
	}
	return lk.skin.soft(l, x, y)
}

// spinoSailOn stands the sail on the line from (x0, y0) to (x1, y1): h tall
// at its highest, starting base above the line.
func spinoSailOn(c *canvas, x0, y0, x1, y1, base, h float64) {
	l := math.Hypot(x1-x0, y1-y0)
	ux, uy := (x1-x0)/l, (y1-y0)/l
	nx, ny := uy, -ux
	top := func(u float64) float64 {
		if u <= 0 || u >= 1 {
			return 0
		}
		w := math.Pow(math.Sin(math.Pi*math.Pow(u, 0.9)), 0.6)
		return base + h*w + 1.3*math.Abs(math.Sin(u*l*math.Pi/5.5)) // scalloped between the spines
	}
	r := int(l + h + base + 4)
	cx, cy := int((x0+x1)/2), int((y0+y1)/2)
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			px, py := float64(x)+0.5-x0, float64(y)+0.5-y0
			u, up := (px*ux+py*uy)/l, px*nx+py*ny
			tp := top(u)
			if up < 0 || up > tp {
				continue
			}
			f := up / (base + h)
			li := 0.3 + 0.5*f + 0.12*lum(-0.4+u*0.5, -f)
			if math.Mod(u*l+0.6, 5.5) < 1.3 {
				li -= 0.22 // the spines that hold it up
			}
			if g := math.Mod(f*4.2+u*0.6, 1); f > 0.3 && g < 0.2 {
				li -= 0.14 // bands of colour across it
			}
			col := spinoSail.pick(li, x, y)
			if tp-up < 1.6 {
				col = spinoSail.at(5) // the bright rim
			}
			c.put(x, y, col)
		}
	}
}

var spinoLook = theroLook{
	skin: spinoSkin, belly: spinoBelly, mark: spinoDark, eye: hex(0xffd838),
	leg: 46, torso: 40, rHip: 15, rBelly: 16, rChest: 13, rThigh: 10.5,
	neck: 27, rNeck: 7.6, neckA: -0.62, headA: 0.2,
	skull: []knob{{-3.5, -6, 6}, {6, -6.2, 5.6}, {16, -4.6, 4.3}, {27, -3.6, 3.4}, {36, -3.3, 3.2}, {41, -3.2, 3.3}},
	jaw:   []knob{{-3, 2.4, 3.6}, {12, 2.2, 2.8}, {28, 1.8, 2.1}, {39, 2, 2.3}},
	eyeU:  5, eyeV: -8, tooth: 3, toothStep: 2.5,
	tail: 60, arm: 15, rArm: 3.6,
	colour: spinoColour,
	back: func(c *canvas, lk *theroLook, body []knob, hip int) {
		h, s := body[hip], body[hip+2]
		l := math.Hypot(s[0]-h[0], s[1]-h[1])
		ux, uy := (s[0]-h[0])/l, (s[1]-h[1])/l
		spinoSailOn(c, h[0]-ux*17, h[1]-uy*17, s[0]+ux*9, s[1]+uy*9, 6, 37)
	},
	crest: func(c *canvas, lk *theroLook, fr bossFrame) {
		// The little crest before the eyes.
		fr.poly(c, [][2]float64{{7, -11}, {10, -14.5}, {14, -9}}, func(x, y int) color.RGBA { return spinoSail.pick(0.55, x, y) })
	},
}

// theroPoses is how the three stand in each pose; bossSpecial is each one's own.
func theroPoses(b bossID, p bossPose) theroPose {
	switch p {
	case bossWalkA:
		return theroPose{body: 0.03, tailUp: -0.08, tailCurl: 0.4, nearDX: 13, farDX: -13, farLift: 3, arm: 0.2}
	case bossWalkB:
		return theroPose{body: 0.03, tailUp: -0.14, tailCurl: 0.25, nearDX: -12, farDX: 12, nearLift: 3, arm: -0.2}
	case bossAttack:
		return theroPose{body: 0.2, neck: 0.5, head: 0, gape: 0.85, tailUp: 0.05, tailCurl: 0.3, nearDX: 14, farDX: -9, arm: 0.5}
	case bossRoar:
		return theroPose{body: -0.16, neck: -0.2, head: -0.62, gape: 1, tailUp: -0.2, tailCurl: 0.2, nearDX: 5, farDX: -8, arm: 0.6}
	case bossLurk:
		return theroPose{body: 0.06, neck: 0.62, head: -0.1, crouch: 0.45, tailUp: -0.12, tailCurl: 0.1, nearDX: 7, farDX: -5}
	case bossSpecial:
		switch b {
		case bossSpino:
			return theroPose{body: -0.6, neck: 0.45, head: -0.2, gape: 1.1, tailUp: -0.3, arm: 1.1, water: true}
		case bossCarcharo:
			return theroPose{body: 0.24, neck: 0.72, head: -0.08, gape: 0.4, tailUp: 0.1, tailCurl: 0.15, nearDX: 24, farDX: -24, farLift: 9, nearLift: 0, arm: -0.5, dust: true}
		case bossGiganoto:
			return theroPose{body: 0.26, neck: 0.5, head: -0.1, gape: 0.55, crouch: 0.12, nearDX: 15, farDX: -13, arm: 0.3,
				tailShape: []float64{0.15, 0.4, 0.7, 1.0, 1.3, 1.55, 1.75}, tailLen: 1.2, sweep: true}
		}
	}
	return theroPose{tailUp: -0.1, tailCurl: 0.32, nearDX: 4, farDX: -8}
}
