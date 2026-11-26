package client

import (
	"image/color"
	"math"
)

// The hunters, side-on and facing right: the raptor, light and striped with
// a feathered crest and a sickle claw, and the tyrant, all head, jaws and
// legs. Tubes and spheres as in dinoart.go.

var (
	raptorRamp  = mkRamp(0x33241a, 0x523a26, 0x765738, 0x9a774d, 0xbd9a68, 0xd8bd8c)
	raptorQuill = mkRamp(0x2a2622, 0x463c32, 0x6a5a48, 0x8c7a62) // dun feathers, barred dark
	rexRamp     = mkRamp(0x1f1a13, 0x352c1f, 0x4e412c, 0x68583b, 0x86734f, 0xa38f68)
	rexBelly    = mkRamp(0x8f7c58, 0xb4a37c, 0xd6c8a0)
)

// Pose indices of the hunters.
const (
	raptorStand = iota
	raptorRunA
	raptorRunB
	raptorPounce // in the air, claws forward, jaws open
	raptorCower  // beaten: crouched low, head down
	raptorEatA   // bent over a kill, tearing: jaws shut...
	raptorEatB   // ...and open
	raptorPoses
)

const (
	rexWalkA = iota
	rexWalkB
	rexRoar // head thrown up, jaws wide
	rexBite // head low and forward, jaws open
	rexEatA // head down at a kill, tearing: jaws nearly shut...
	rexEatB // ...and wide
	rexPoses
)

// jaws draws an open or closed snout from the back of the mouth (x, y),
// pointing along a (radians, 0 is right), len long and h deep; gape opens it.
func jaws(c *canvas, rp ramp, x, y, a, length, h, gape, bias float64) {
	p := func(u, v float64) (float64, float64) { return rot(x+u, y+v, x, y, a) }
	poly := func(q [][2]float64, f func(px, py int) color.RGBA) {
		var pts []float64
		for _, v := range q {
			px, py := p(v[0], v[1])
			pts = append(pts, px, py)
		}
		c.polygon(pts, f)
	}
	upper := rotPts(-gape/2, [][2]float64{{-1, -h}, {length * 0.6, -h * 0.9}, {length, -h * 0.25}, {length, 0.4}, {-1, 0.6}})
	lower := rotPts(gape/2, [][2]float64{{-1, 0}, {length * 0.92, 0.2}, {length * 0.92, h * 0.55}, {-1, h * 0.7}})
	if gape > 0 {
		poly([][2]float64{{-1, -0.5}, {length * 0.9, -gape * length * 0.45}, {length * 0.9, gape * length * 0.45}, {-1, 0.8}}, solid(colMouth))
	}
	poly(lower, func(px, py int) color.RGBA { return rp.pick(0.35+bias, px, py) })
	poly(upper, func(px, py int) color.RGBA { return rp.pick(0.62+bias-float64(py)*0.004, px, py) })
	if gape > 0 { // teeth along both jaws
		for k := 0.25; k < 0.95; k += 0.17 {
			ux, uy := rotPt(-gape/2, k*length, 0.6)
			lx, ly := rotPt(gape/2, k*length*0.95, -0.2)
			px, py := p(ux, uy)
			dot(c, px, py, colTooth)
			dot(c, px+0.4, py+0.5, hornRamp[1]) // a tooth is a fang, not a dot
			px, py = p(lx, ly)
			dot(c, px, py, colTooth)
			dot(c, px+0.4, py-0.5, hornRamp[1])
		}
	} else {
		for _, k := range []float64{0.36, 0.58, 0.74} {
			px, py := p(k*length, 0.8)
			dot(c, px, py, hornRamp[2]) // a tooth tip or two showing past the lip
			dot(c, px, py+0.5, hornRamp[1])
		}
	}
}

func rotPt(a, x, y float64) (float64, float64) {
	s, co := math.Sincos(a)
	return x*co - y*s, x*s + y*co
}

func rotPts(a float64, q [][2]float64) [][2]float64 {
	out := make([][2]float64, len(q))
	for i, v := range q {
		out[i][0], out[i][1] = rotPt(a, v[0], v[1])
	}
	return out
}

// --- raptor -----------------------------------------------------------------

// raptorRun is what a running raptor's stride is built on: stretched out
// low, head and tail level.
var raptorRun = rigPose{tilt: 0.07, head: 0.08, tail: 0.06} // the long tail lifted a little

func genRaptor(pose int) sprite { c, ax, ay := raptorCanvas(pose); return mk(c, ax, ay) }

// raptorLook is a raptor's colours and size: Dakotaraptor, or the smaller,
// darker Acheroraptor.
type raptorLook struct {
	body, quill ramp
	scale       float64
	rig         string // its drawing in dinos/; see dinorigs.go
}

var (
	dakotaLook = raptorLook{raptorRamp, raptorQuill, raptorScale, "dakota"}
	acheroLook = raptorLook{
		mkRamp(0x24201f, 0x3b3532, 0x58504a, 0x786d64, 0x9a8e82, 0xbab0a2),
		mkRamp(0x5a1a14, 0x8c2a1c, 0xbc4428, 0xe27446), acheroScale, "achero",
	}
)

func raptorCanvas(pose int) (*canvas, float64, float64) { return raptorCanvasOf(pose, dakotaLook) }

func raptorCanvasOf(pose int, lk raptorLook) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino(lk.rig, pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	raptorRamp, raptorQuill := lk.body, lk.quill
	c := newScaledCanvas(32, 24, lk.scale)
	gy := 22.0
	tilt, gape, drop := 0.0, 0.0, 0.0
	type limb [3][2]float64 // hip, knee, foot
	near := limb{{12, 12}, {14, 16.5}, {11, 21}}
	far := limb{{11, 12}, {13, 16.5}, {10.5, 21}}
	switch pose {
	case raptorRunA:
		near = limb{{12, 12}, {15.5, 15}, {15, 21}}
		far = limb{{11, 12}, {11.5, 16.5}, {7, 19.5}}
	case raptorRunB:
		near = limb{{12, 12}, {11.5, 16.5}, {7.5, 19.5}}
		far = limb{{11, 12}, {15, 15}, {14.5, 21}}
	case raptorPounce: // airborne: both feet thrust forward, claws out
		tilt, gape = -0.15, 0.55
		near = limb{{12, 11}, {16, 13}, {19, 17}}
		far = limb{{11, 11}, {15, 14}, {18, 18.5}}
	case raptorEatA, raptorEatB: // bent over to eat or drink
		tilt, drop, gape = 0.42, 2.5, 0.08
		if pose == raptorEatB {
			gape = 0.55
		}
	case raptorCower: // crouched, legs folded under, the head hung low
		tilt, drop = 0.22, 6
		near = limb{{12, 17}, {15.5, 19.5}, {11, 21}}
		far = limb{{11, 17}, {14.5, 19.5}, {10, 21}}
	}
	legDraw := func(l limb, bias float64) {
		hip, knee, foot := l[0], l[1], l[2]
		tube(c, spine([3]float64{hip[0], hip[1], 2.6}, [3]float64{knee[0], knee[1], 1.2}, [3]float64{foot[0], foot[1] - 1, 0.8}), raptorRamp, bias)
		c.line(foot[0]-0.5, foot[1]-0.6, foot[0]+3, foot[1]-0.4, func(x, y int, _ float64) color.RGBA { return raptorRamp.at(1 + int(bias*4)) })
		c.set(int(foot[0]+0.5), int(foot[1]-2.4), hornRamp[1]) // the sickle claw, held up
		c.set(int(foot[0]+1.2), int(foot[1]-2), hornRamp[1])
	}
	legDraw(far, -0.25)
	p := func(x, y float64) (float64, float64) { return rot(x, y+drop, 13, 10+drop, tilt) }
	// A stiff tail held straight out, then the body.
	tx0, ty0 := p(9, 10)
	tx1, ty1 := p(1, 8.5)
	tube(c, spine([3]float64{tx0, ty0, 2.6}, [3]float64{(tx0 + tx1) / 2, (ty0 + ty1) / 2, 1.4}, [3]float64{tx1, ty1, 0.6}), raptorRamp, 0)
	bx, by := p(13.5, 10)
	c.ellipse(bx, by, 6, 3.6, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if (x+int(ny*3))%4 == 0 && ny < 0.3 {
			l -= 0.3 // dark stripes across the back
		}
		return raptorRamp.pick(l, x, y)
	})
	// Neck, head and the feathered crest down the neck.
	nx0, ny0 := p(17.5, 9)
	nx1, ny1 := p(20.5, 5.5)
	tube(c, spine([3]float64{nx0, ny0, 2.2}, [3]float64{nx1, ny1, 1.6}), raptorRamp, 0.05)
	for i := range 4 {
		qx, qy := p(16.5+float64(i)*1.1, 7.6-float64(i)*1.1)
		c.set(int(qx), int(qy), raptorQuill.at(1+i%3))
		c.set(int(qx)-1, int(qy), raptorQuill.at(i%3))
	}
	hx, hy := p(21.5, 5.4)
	roundShape(c, ell{hx, hy, 2.2, 1.7}, raptorRamp, 0.1)
	jaws(c, raptorRamp, hx+0.8, hy+0.6, tilt+0.06, 5.5, 1.5, gape, 0.05)
	c.set(int(hx), int(hy-0.6), colInk)
	// The arms, folded, with a fringe of quills.
	ax0, ay0 := p(17, 11)
	tube(c, spine([3]float64{ax0, ay0, 1.1}, [3]float64{ax0 + 2, ay0 + 2.2, 0.6}), raptorRamp, -0.05)
	c.set(int(ax0), int(ay0+1), raptorQuill[2])
	c.set(int(ax0+1), int(ay0+2), raptorQuill[1])
	legDraw(near, 0.05)
	finishBeast(c, 0.8)
	if pose == raptorPounce {
		gy += 2 // in the air: drawn higher than its shadow
	}
	return c, 14, gy
}

// --- tyrant -----------------------------------------------------------------

func genRex(pose int) sprite { c, ax, ay := rexCanvas(pose); return mk(c, ax, ay) }

func rexCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("rex", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(60, 48, rexScale)
	gy := 46.0
	stride := 2.4
	if pose == rexWalkB {
		stride = -2.4
	}
	if pose == rexRoar || pose == rexBite {
		stride = 1.2
	}
	bigLeg := func(hx, hy, swing, bias float64) {
		// A bird's leg built like a tower: the great drumstick of a thigh, the
		// knee forward, the shin raking back to a high ankle, and a long foot
		// forward onto three clawed toes.
		kx, ky := hx+2.6+swing*0.3, hy+7.5
		ax, ay := hx-1.2+swing*0.7, gy-5.5
		fx := hx + 2.2 + swing
		tube(c, spine([3]float64{kx, ky, 3.2}, [3]float64{ax, ay, 2.1}), rexRamp, bias-0.05)
		tube(c, spine([3]float64{ax, ay, 1.9}, [3]float64{fx, gy - 1.2, 1.4}), rexRamp, bias-0.1)
		roundShape(c, ell{hx + 0.6, hy + 1, 5.4, 7}, rexRamp, bias) // the thigh
		for i, toe := range []float64{-2.2, 0.6, 3.4} {
			tube(c, spine([3]float64{fx, gy - 1.2, 1.2}, [3]float64{fx + toe, gy - 0.6, 0.7}), rexRamp, bias-0.15)
			dot(c, fx+toe+0.6, gy-0.4, hornRamp[1+i%2]) // a claw
		}
	}
	bigLeg(20, 28, -stride, -0.25)
	// The tail, held out level as a counterweight.
	tube(c, spine([3]float64{16, 22, 6.5}, [3]float64{10, 20.6, 4.6}, [3]float64{5, 20.2, 2.6}, [3]float64{0.8, 21.2, 0.8}), rexRamp, 0)
	roundShape(c, ell{24, 23, 11.5, 8.4}, rexRamp, 0.03)
	c.ellipse(26, 28, 8.5, 3.2, func(x, y int, nx, ny float64) color.RGBA {
		return rexBelly.pick(0.55+nx*0.25, x, y)
	})
	for _, x := range []float64{14, 18, 22, 26} {
		c.line(x, 15.3, x+2, 19.5, func(px, py int, _ float64) color.RGBA { return mix(c.at(px, py), rexRamp[0], 0.5) })
	}
	// The neck and head; the head turns about the neck to roar or bite.
	a, gape := 0.0, 0.0
	nx, ny := 34.0, 17.0
	switch pose {
	case rexRoar:
		a, gape, ny = -0.55, 0.7, 15.5
	case rexBite:
		a, gape, ny = 0.3, 0.5, 19
	case rexEatA:
		a, gape, ny = 0.95, 0.12, 23
	case rexEatB:
		a, gape, ny = 1.0, 0.55, 23.5
	}
	tube(c, spine([3]float64{31, 20, 6}, [3]float64{nx, ny, 5}), rexRamp, 0.05)
	p := func(x, y float64) (float64, float64) { return rot(nx+x, ny+y, nx, ny, a) }
	sx, sy := p(4, -1.5)
	roundShape(c, ell{sx, sy, 6, 4.8}, rexRamp, 0.1) // the skull
	mx, my := p(4, 0.8)
	jaws(c, rexRamp, mx, my, a, 13, 4, gape, 0.08)
	ex, ey := p(4.4, -3.2)
	c.set(int(ex), int(ey), hex(0xe0b040))
	c.set(int(ex)+1, int(ey), colInk)
	bx, by := p(3, -4.8)
	c.line(bx, by, bx+3.5, by+0.4, func(x, y int, _ float64) color.RGBA { return rexRamp[1] }) // the brow ridge
	// Tiny arms.
	tube(c, spine([3]float64{32.5, 25, 1.3}, [3]float64{34.5, 28, 0.9}, [3]float64{36, 27.5, 0.6}), rexRamp, -0.1)
	bigLeg(23, 29, stride, 0.05)
	finishBeast(c, 0.85)
	return c, 25, gy
}
