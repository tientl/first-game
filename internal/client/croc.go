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

func genCroc(pose int) sprite {
	c, ax, ay := crocCanvas(pose)
	return mk(c, ax, ay)
}

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

// crocCanvas draws crocodile pose on a canvas and returns its anchor.
func crocCanvas(pose int) (*canvas, float64, float64) {
	if pose == crocLurk {
		return crocLurkCanvas()
	}
	c := newCanvas(46, 20)
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
	drawTubeWith(c, body, crocSkin)
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
	c.outline(0.8)
	return c, 22, baseY + 2
}

// crocLurkCanvas is a crocodile under water: two eyes on their knobs and the
// nostrils at the end of the snout, just breaking the surface.
func crocLurkCanvas() (*canvas, float64, float64) {
	c := newCanvas(16, 7)
	for _, k := range []ell{{3.5, 3.5, 1.7, 1.3}, {7, 3.2, 1.7, 1.3}} {
		roundShape(c, k, crocRamp, 0.1)
		c.set(int(k.x), int(k.y-0.4), crocEye)
		c.set(int(k.x+0.6), int(k.y-0.4), colInk)
	}
	roundShape(c, ell{13, 3.8, 1.3, 0.9}, crocRamp, 0)
	c.set(13, 3, crocRamp[0])
	c.outline(0.6)
	return c, 8, 5
}
