package client

import (
	"image/color"
	"math"
)

// The crocodile, side-on and facing right like the other animals: a long
// armoured body drawn as a tube (see drawTubeWith in snake.go), a row of
// ridged scutes down the back and tail, four short splayed legs, a long flat
// snout with teeth along the jaw, and an eye on a raised knob. Two walk
// frames sway the body and swap the legs; the bite opens the jaws wide. A
// lurking crocodile is only its eyes and nostrils above the water.

var (
	crocRamp  = mkRamp(0x1a2112, 0x2a351a, 0x3f4d25, 0x566634, 0x748446, 0x95a35c)
	crocBelly = mkRamp(0x8a8352, 0xb0a96e, 0xcfc78c)
	crocEye   = hex(0xd8c84a)
	colMouth  = hex(0xb04a4a)
	colTooth  = hex(0xf4f0e2)
)

// Crocodile sprite indices into creatureArt.croc.
const (
	crocWalkA = iota
	crocWalkB
	crocBite
	crocLurk
	crocPoses
)

func genCroc(pose int) sprite { return critter("croc", 0, pose, 0) }

// The marsh crocodile's outline, side-on: its back and its belly from the
// tip of the tail (x = 1) to the end of the snout, as (x, y) points joined
// by straight lines. The eye sits on a knob at the back of the flat skull
// and the nostrils on a bump at the end of the long snout.
var (
	muggerTop = [][2]float64{{1, 12.5}, {6, 11.5}, {12, 9.9}, {16, 8.7}, {20, 7.7}, {24, 7.2}, {28, 7.6}, {31, 8.5}, {33, 8.3},
		{34.2, 7.1}, {36.6, 7.2}, {38, 9.1}, {41.5, 9.4}, {42.6, 8.9}, {44, 9.2}, {45, 10.2}}
	muggerBot = [][2]float64{{1, 13.4}, {6, 13.4}, {12, 13.6}, {16, 13.8}, {20, 14.4}, {25, 14.6}, {29, 14.2}, {32, 13.4},
		{35, 13.4}, {38, 12.95}, {45, 12.95}}
	muggerRamp  = mkRamp(0x151d0d, 0x283716, 0x3f5222, 0x5a6e2e, 0x7c8e3e, 0xa4b058)
	muggerBelly = mkRamp(0x7d7444, 0xb3a765, 0xdcd08f, 0xf3ecbc)
)

const (
	muggerJawY   = 11   // the row the jaws meet on
	muggerHingeX = 34.5 // where they hinge
)

// profAt reads the outline p at x.
func profAt(p [][2]float64, x float64) float64 {
	for i := 1; i < len(p); i++ {
		if x <= p[i][0] {
			return lerpf(p[i-1][1], p[i][1], clamp01((x-p[i-1][0])/(p[i][0]-p[i-1][0])))
		}
	}
	return p[len(p)-1][1]
}

// muggerHide colours the point (fx, fy) of the crocodile standing side-on, for
// the canvas pixel (x, y); clear outside it. sway lifts or drops the tail.
// The back is dark olive crossed by darker bands, with a row of pale scutes
// along its ridge; the flank is scaly; the belly, throat and lower jaw are
// pale yellow; teeth show all along the line of the jaws.
func muggerHide(fx, fy, sway float64, x, y int) color.RGBA {
	if fx < 1 || fx >= 45 {
		return color.RGBA{}
	}
	lift := sway * smooth(clamp01((17-fx)/16))
	top, bot := profAt(muggerTop, fx)+lift, profAt(muggerBot, fx)+lift
	if fy < top || fy > bot {
		return color.RGBA{}
	}
	ix, iy := int(math.Floor(fx)), int(math.Floor(fy-lift))
	v := (fy - top) / (bot - top)
	l := 0.92 - v*0.6 - (fx-22)/44*0.1
	head := fx >= muggerHingeX
	if head {
		l += 0.12 - v*0.1 // the flat skull and snout catch the light
	}
	switch {
	case head && iy == muggerJawY: // the line of the mouth
		if ix >= 36 && ix%2 == 0 {
			return colTooth
		}
		return muggerRamp[0]
	case head && iy > muggerJawY:
		return muggerBelly.pick(0.62-(fx-34)*0.012, x, y) // the lower jaw
	case !head && v > 0.66 && fx > 8: // belly plates
		lb := 0.85 - (v-0.66)*1.5
		if ix%3 == 0 {
			lb -= 0.14
		}
		return muggerBelly.pick(lb, x, y)
	case fy-top < 1 && fx < 33: // the armoured ridge
		if ix%2 == 0 {
			return muggerRamp.pick(l+0.1, x, y)
		}
		return muggerRamp.pick(l-0.14, x, y)
	}
	if !head && v < 0.62 && (ix+iy/2)%6 < 2 {
		l -= 0.2 // dark bands across the back and tail
	}
	if (ix+2*iy)%4 == 0 {
		l -= 0.08 // scales
	}
	if !head && v > 0.5 {
		l += 0.06 // the flank turning towards the pale belly
	}
	return muggerRamp.pick(l, x, y)
}

// muggerLeg draws one short splayed leg from its hip at (hx, hy), the foot
// swing pixels ahead; the far pair is darker.
func muggerLeg(c *canvas, hx, hy, swing, size float64, near bool) {
	bias := 0.1
	if !near {
		bias = -0.22
	}
	const ground = 15
	fx := hx + swing*1.6
	for k := -1.0; k <= 0.5; k += 0.5 { // the shin
		c.line(hx+k, hy+0.5, fx+k, ground, func(x, y int, t float64) color.RGBA {
			return muggerRamp.pick(0.4+bias-k*0.12, x, y)
		})
	}
	c.ellipse(hx, hy, 2.1*size, 1.7*size, func(x, y int, nx, ny float64) color.RGBA { // the thigh
		l := lum(nx, ny)*0.5 + 0.3 + bias
		if (x+2*y)%4 == 0 {
			l -= 0.08
		}
		return muggerRamp.pick(l, x, y)
	})
	for k := -1; k <= 2; k++ { // the foot, toes forward
		col := muggerRamp.at(2)
		if !near {
			col = muggerRamp.at(1)
		}
		if k == 2 {
			col = muggerBelly.at(1) // claws
		}
		c.set(int(math.Floor(fx))+k, ground, col)
	}
}

// muggerCanvas draws the marsh crocodile in pose and returns its anchor.
func muggerCanvas(pose int) (*canvas, float64, float64) {
	if pose == crocLurk {
		return muggerLurkCanvas()
	}
	c := newCanvas(46, 20)
	sway, swing := 0.9, 1.0
	switch pose {
	case crocWalkB:
		sway, swing = -0.9, -1
	case crocBite:
		sway, swing = -1.4, 0
	}
	muggerLeg(c, 19.5, 12.2, -swing, 1, false)
	muggerLeg(c, 30.5, 12.2, swing, 0.85, false)
	bite := pose == crocBite
	const hx, hy = muggerHingeX, float64(muggerJawY)
	up, down := -0.44, 0.2 // how far the jaws swing open
	turn := func(x, y, a float64) (float64, float64) {
		dx, dy := x-hx, y-hy
		return hx + dx*math.Cos(a) - dy*math.Sin(a), hy + dx*math.Sin(a) + dy*math.Cos(a)
	}
	if bite { // the inside of the mouth, between the jaws
		ux, uy := turn(43.5, hy, up)
		lx, ly := turn(43.5, hy+1, down)
		c.polygon([]float64{hx - 1, hy, ux, uy, lx, ly}, func(x, y int) color.RGBA {
			return mix(hex(0x6e1c26), colMouth, clamp01((float64(x)-hx)/6))
		})
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if !bite {
				if col := muggerHide(fx, fy, sway, x, y); col.A > 0 {
					c.put(x, y, col)
				}
				continue
			}
			if fx < hx+1 { // the body and the back of the skull stay put
				if col := muggerHide(fx, fy, sway, x, y); col.A > 0 && !(fx >= hx && fy >= hy) {
					c.put(x, y, col)
				}
			}
			if sx, sy := turn(fx, fy, -up); sx >= hx-0.5 && sy < hy {
				if col := muggerHide(sx, hy-(hy-sy)*0.72, 0, x, y); col.A > 0 { // seen a little from below: thicker
					c.put(x, y, col)
				}
			}
			if sx, sy := turn(fx, fy, -down); sx >= hx-1 && sy >= hy+0.6 {
				if col := muggerHide(sx, sy-0.6, 0, x, y); col.A > 0 { // a deeper lower jaw
					c.put(x, y, muggerBelly.pick(0.6-(sy-hy)*0.12, x, y))
				}
			}
		}
	}
	// The eye on its knob and the nostril, and in an open mouth the teeth.
	spot := func(x, y, a float64, col color.RGBA) {
		if bite {
			x, y = turn(x, y, a)
		}
		c.set(int(math.Floor(x)), int(math.Floor(y)), col)
	}
	eyeY, noseY := 8.5, 9.5
	if bite {
		eyeY, noseY = 7.5, 8.9
	}
	spot(35.5, eyeY, up, crocEye)
	spot(36.5, eyeY, up, colInk)
	spot(35.5, eyeY-1, up, muggerRamp[5])
	spot(43.5, noseY, up, muggerRamp[0])
	if bite {
		for k := 2.5; k < 9.5; k += 2 {
			spot(hx+k, hy+0.5, up, colTooth)
			spot(hx+k+0.5, hy+0.4, down, colTooth)
		}
	}
	muggerLeg(c, 18, 12.6, swing, 1, true)
	muggerLeg(c, 29.5, 12.8, -swing, 0.85, true)
	// The scutes stand up along the back and the tail.
	for x := 5; x <= 31; x += 2 {
		lift := sway * smooth(clamp01((17-float64(x)-0.5)/16))
		y := int(math.Floor(profAt(muggerTop, float64(x)+0.5)+lift)) - 1
		if c.at(x, y).A == 0 {
			c.set(x, y, muggerRamp.at(3+x/2%2))
		}
	}
	c.outline(0.82)
	return c, 22, 14
}

// muggerLurks is the crocodile under water, side-on: the eye on its knob, the
// flat top of the skull and snout, the nostrils, and a few scutes of the
// back breaking the surface behind.
var muggerLurks = []string{
	".......hh..........",
	"......hyih.......hh",
	".h.h.mmmmmmhmmmmmmn",
	"dmdmddddddddddddddd",
}

func muggerLurkCanvas() (*canvas, float64, float64) {
	c := newCanvas(21, 7)
	stamp(c, 1, 1, muggerLurks, map[byte]color.RGBA{
		'h': muggerRamp[4], 'm': muggerRamp[3], 'd': muggerRamp[1], 'n': muggerRamp[0], 'y': crocEye, 'i': colInk,
	})
	c.outline(0.7)
	return c, 10, 5
}

// crocLook is a crocodile's colours, the width of its body along its length
// (snout and all) and its size: the crocodile, or its pond cousins.
type crocLook struct {
	skin, belly ramp
	width       func(s float64) float64
	scale       float64
}

var crocStd = crocLook{crocRamp, crocBelly, crocWidth, 1}

func crocCanvas(pose int) (*canvas, float64, float64) { return muggerCanvas(pose) }

// crocWidth is the body's half-width at s, 0 the tail tip and 1 the snout.
func crocWidth(s float64) float64 {
	switch {
	case s < 0.5:
		return 0.5 + 2.4*smooth(s/0.5) // the tail thickening into the body
	case s < 0.66:
		return 2.9
	case s < 0.78:
		return lerpf(2.9, 2.1, (s-0.66)/0.12) // the neck
	case s < 0.84:
		return lerpf(2.1, 2.3, (s-0.78)/0.06) // the jaw's hinge
	default:
		return lerpf(2.3, 0.9, (s-0.84)/0.16) // the long snout
	}
}

// crocSkin colours one pixel of the body: a ridge of scutes along the back,
// pale on the near flank, dull green elsewhere.
func crocSkin(x, y int, p *spinePt, t float64) color.RGBA {
	return crocSkinOf(crocRamp, crocBelly)(x, y, p, t)
}

// crocSkinOf is crocSkin in the colours rp and belly.
func crocSkinOf(crocRamp, crocBelly ramp) func(x, y int, p *spinePt, t float64) color.RGBA {
	return func(x, y int, p *spinePt, t float64) color.RGBA {
		l := lum(p.nx*t*0.8, p.ny*t*0.8) + 0.05
		scute := math.Mod(p.s*34, 1) < 0.5
		switch {
		case t > 0.68 && p.s > 0.12:
			return crocBelly.pick(l, x, y)
		case t > -0.55 && t < -0.15 && p.s < 0.84: // the armoured ridge down the back
			if scute {
				return crocRamp.pick(l+0.25, x, y)
			}
			return crocRamp.pick(l-0.15, x, y)
		}
		i := crocRamp.idx(l, x, y)
		if (x+2*y)%5 == 0 {
			i-- // scaly
		}
		return crocRamp.at(i)
	}
}

// crocCanvasOf draws crocodile pose, in the look lk, on a canvas and returns
// its anchor.
func crocCanvasOf(pose int, lk crocLook) (*canvas, float64, float64) {
	if pose == crocLurk {
		return crocLurkCanvasOf(lk)
	}
	crocRamp, crocWidth := lk.skin, lk.width
	c := newScaledCanvas(46, 20, lk.scale)
	const snoutX, baseY = 43.0, 12.0
	ph := 0.0
	if pose == crocWalkB {
		ph = math.Pi
	}
	// A gentle sway: crocodiles walk with the body low and nearly straight.
	pts := serpenoid(41, 0.22, 1, ph, snoutX, baseY)
	for i := range pts {
		pts[i].r = crocWidth(pts[i].s)
	}
	at := func(s float64) spinePt { return pts[int(s*float64(len(pts)-1))] }

	// Legs: stubby, splayed, a claw at the foot; the far pair under the body.
	leg := func(s, side, swing float64) {
		b := at(s)
		bx, by := b.x+b.nx*b.r*0.55*side, b.y+b.ny*b.r*0.55*side
		fx, fy := bx+swing*2, by+3.2
		col := crocRamp[2]
		if side < 0 {
			col = crocRamp[1]
		}
		for k := 0.0; k <= 1; k += 0.25 { // a thick leg: three lines side by side
			c.line(bx-1+k*2, by, fx-1+k*2, fy, func(int, int, float64) color.RGBA { return col })
		}
		c.set(int(fx+swing+1.5), int(fy), crocRamp[0]) // claws
		c.set(int(fx+swing-0.5), int(fy), crocRamp[0])
	}
	swing := 1.0
	if pose == crocWalkB {
		swing = -1
	}
	if pose == crocBite {
		swing = 0
	}
	leg(0.36, -1, -swing)
	leg(0.66, -1, swing)

	body := pts
	if pose == crocBite {
		body = pts[:int(0.84*float64(len(pts)-1))] // the jaws are drawn open below
	}
	drawTubeWith(c, body, crocSkinOf(lk.skin, lk.belly))
	leg(0.36, 1, swing)
	leg(0.66, 1, -swing)

	hinge := at(0.84)
	if pose == crocBite {
		// Jaws open: the lower one along the ground, the upper one raised, the
		// pink mouth and a row of teeth between them.
		hx, hy := hinge.x, hinge.y
		lower := []float64{hx - 1.5, hy - 0.4, hx + 8.5, hy + 0.8, hx + 8.5, hy + 2.6, hx - 1.5, hy + 2.6}
		upper := []float64{hx - 1.5, hy - 2.8, hx + 6.2, hy - 7.4, hx + 8, hy - 5.6, hx + 0.6, hy + 0.2}
		c.polygon([]float64{hx, hy + 0.2, hx + 6.8, hy - 5.2, hx + 8, hy + 1}, solid(colMouth))
		c.polygon(lower, func(x, y int) color.RGBA { return crocRamp.pick(0.4+float64(x-int(hx))*0.02, x, y) })
		c.polygon(upper, func(x, y int) color.RGBA { return crocRamp.pick(0.55+float64(x-int(hx))*0.03, x, y) })
		for k := 2.0; k < 8; k += 2 {
			c.set(int(hx+k), int(hy+0.4), colTooth)                 // up from the lower jaw
			c.set(int(hx+0.6+k*0.85), int(hy-0.8-k*0.62), colTooth) // down from the upper
		}
		roundShape(c, ell{hx - 0.4, hy - 3.2, 1.3, 1.1}, crocRamp, 0.1)
		c.set(int(hx-0.2), int(hy-3.5), crocEye)
	} else {
		// Teeth showing along the closed jaw, a nostril at the tip, the eye
		// on its knob at the back of the head.
		for _, s := range []float64{0.88, 0.94} { // two teeth showing past the lip
			p := at(s)
			c.set(int(p.x), int(p.y+p.r+0.3), colTooth)
		}
		tip := at(0.995)
		c.set(int(tip.x), int(tip.y-tip.r-0.2), crocRamp[0])
		roundShape(c, ell{hinge.x + 0.5, hinge.y - 2.1, 1.3, 1.1}, crocRamp, 0.1)
		c.set(int(hinge.x+0.8), int(hinge.y-2.4), crocEye)
		c.set(int(hinge.x+1.2), int(hinge.y-2.4), colInk)
	}
	finishBeast(c, 0.8)
	return c, 22, baseY + 2
}

// crocLurkCanvas is a crocodile under water: two eyes on their knobs and the
// nostrils at the end of the snout, just breaking the surface.
func crocLurkCanvas() (*canvas, float64, float64) { return crocLurkCanvasOf(crocStd) }

func crocLurkCanvasOf(lk crocLook) (*canvas, float64, float64) {
	crocRamp := lk.skin
	c := newScaledCanvas(16, 7, lk.scale)
	for _, k := range []ell{{3.5, 3.5, 1.7, 1.3}, {7, 3.2, 1.7, 1.3}} {
		roundShape(c, k, crocRamp, 0.1)
		c.set(int(k.x), int(k.y-0.4), crocEye)
		c.set(int(k.x+0.6), int(k.y-0.4), colInk)
	}
	roundShape(c, ell{13, 3.8, 1.3, 0.9}, crocRamp, 0)
	c.set(13, 3, crocRamp[0])
	finishBeast(c, 0.6)
	return c, 8, 5
}
