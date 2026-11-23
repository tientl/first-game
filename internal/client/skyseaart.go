package client

import (
	"image/color"
	"math"
)

// The reptiles of the air and the sea. The flying reptile is drawn side-on
// like a bird in flight: wings up, down, held level to glide, swept back to
// dive, and folded when it perches. The sea reptile is a hump in the water
// with a long neck reared up, lunging, or nothing but a dark shape below.

var (
	wingRamp   = mkRamp(0x4a2a1e, 0x6c402c, 0x91593d, 0xb57852, 0xd29c72)
	pteroRamp  = mkRamp(0x2e2a2a, 0x4a4442, 0x6a625c, 0x8c837a, 0xaea49a)
	crestRamp  = mkRamp(0x6e1a16, 0xa22f22, 0xd65232, 0xf0824e)
	plesioRamp = mkRamp(0x182a32, 0x253f4b, 0x355766, 0x4c7584, 0x6c95a2, 0x95b8c0)
)

// Pose indices.
const (
	pteroUp = iota
	pteroDown
	pteroGlide
	pteroDive
	pteroPerch
	pteroPoses
)

const (
	plesioUp     = iota // neck reared out of the water
	plesioStrike        // neck lunging, jaws open
	plesioUnder         // a dark shape below the surface
	plesioPoses
)

// pteroAlt is how high above its shadow a flying reptile is drawn, by state.
const (
	pteroSoarAlt  = 34.0
	pteroClimbAlt = 22.0
	pteroDiveAlt  = 9.0
)

func genPtero(pose int) sprite { c, ax, ay := pteroCanvas(pose); return mk(c, ax, ay) }

// pteroLook is a flying reptile's colours and size in flight.
type pteroLook struct {
	wing, body, crest ramp
	scale             float64
	drawn             bool // drawn from dinos/ (the pteranodon); see dinorigs.go
}

var pteranodonLook = pteroLook{wingRamp, pteroRamp, crestRamp, pteroScale, true}

func pteroCanvas(pose int) (*canvas, float64, float64) { return pteroCanvasOf(pose, pteranodonLook) }

func pteroCanvasOf(pose int, lk pteroLook) (*canvas, float64, float64) {
	if lk.drawn {
		name := "pteroFly"
		if pose == pteroPerch {
			name = "pteroPerch"
		}
		if c, ax, ay, ok := drawnDino(name, pose); ok {
			return c, ax, ay // drawn from dinos/; see dinorig.go
		}
	}
	wingRamp, pteroRamp, crestRamp := lk.wing, lk.body, lk.crest
	if pose == pteroPerch {
		return pteroPerchCanvas()
	}
	// In flight it is seen from below, high overhead: the body along the
	// way it flies, the wings spread across it, wide when level and
	// foreshortened at the top and bottom of each beat.
	c := newScaledCanvas(42, 36, lk.scale)
	const bx, by = 18.0, 18.0
	span, sweep := 16.0, 2.0 // how far the wingtips reach out, and back
	switch pose {
	case pteroUp:
		span, sweep = 9, 1
	case pteroDown:
		span, sweep = 12.5, 3.5
	case pteroDive:
		span, sweep = 7, 9
	}
	// wing spreads one wing to side (-1 up the screen, 1 down): leading
	// edge from the shoulder through the wrist to the tip, the membrane
	// trailing back to the hip.
	wing := func(side, bias float64) {
		wx, wy := bx+2.5-sweep*0.4, by+side*span*0.5
		tx, ty := bx-sweep, by+side*span
		rx, ry := bx-5.5-sweep*0.5, by+side*span*0.45 // the trailing edge, bellied out
		c.polygon([]float64{bx + 1.5, by + side*0.8, wx, wy, tx, ty, rx, ry, bx - 4.5, by + side*1.2},
			func(x, y int) color.RGBA {
				l := 0.62 + bias - math.Abs(c.ly(y)-by)/span*0.25
				if hashf(x, y, 77) < 0.07 {
					l -= 0.15
				}
				return wingRamp.pick(l, x, y)
			})
		bone := func(x, y int, _ float64) color.RGBA { return wingRamp.at(1) }
		c.line(bx+1.5, by+side*0.8, wx, wy, bone)
		c.line(wx, wy, tx, ty, bone)
	}
	wing(-1, -0.12)
	wing(1, 0.05)
	// The body and tail along the line of flight, the long beak ahead and
	// the crest swept back over the neck.
	roundShape(c, ell{bx, by, 4.6, 2.3}, pteroRamp, 0.1)
	c.line(bx-4, by, bx-8, by, func(x, y int, _ float64) color.RGBA { return pteroRamp[2] })
	hx := bx + 6.0
	tube(c, spine([3]float64{bx + 3, by, 1.5}, [3]float64{hx, by, 1.3}), pteroRamp, 0.1)
	c.polygon([]float64{hx - 1, by - 0.8, hx - 7.5, by - 1.6, hx - 1, by + 0.4}, func(x, y int) color.RGBA { return crestRamp.pick(0.65, x, y) })
	roundShape(c, ell{hx, by, 2, 1.5}, pteroRamp, 0.18)
	c.polygon([]float64{hx + 1, by - 1, hx + 11, by, hx + 1, by + 1}, func(x, y int) color.RGBA { return hornRamp.pick(0.4, x, y) })
	finishBeast(c, 0.75)
	return c, bx, by + 2
}

// pteroPerchCanvas is the flying reptile on the ground: walking on its
// folded wings, crest up.
func pteroPerchCanvas() (*canvas, float64, float64) {
	c := newScaledCanvas(22, 22, pteroScale)
	const gy = 20.0
	// Folded wings as forelimbs, hind legs short.
	tube(c, spine([3]float64{11, 11, 2.2}, [3]float64{13.5, 15, 1.4}, [3]float64{13, gy - 0.5, 0.8}), wingRamp, -0.2)
	tube(c, spine([3]float64{7, 14, 1.6}, [3]float64{6.5, gy - 0.5, 0.8}), pteroRamp, -0.2)
	c.polygon([]float64{6, 9, 12, 7.5, 13, 13, 7, 15}, func(x, y int) color.RGBA { return wingRamp.pick(0.45, x, y) })
	roundShape(c, ell{9, 11.5, 3.4, 3}, pteroRamp, 0.05)
	tube(c, spine([3]float64{11, 9.5, 1.6}, [3]float64{13, 6, 1.3}), pteroRamp, 0.05)
	c.polygon([]float64{12.5, 5, 7, 1, 12.5, 6.5}, func(x, y int) color.RGBA { return crestRamp.pick(0.6, x, y) })
	roundShape(c, ell{13.5, 5.6, 1.8, 1.4}, pteroRamp, 0.15)
	c.polygon([]float64{14.5, 4.8, 21.5, 6.8, 14.5, 7}, func(x, y int) color.RGBA { return hornRamp.pick(0.35, x, y) })
	c.set(13, 5, colInk)
	tube(c, spine([3]float64{9.5, 13, 1.6}, [3]float64{10.5, gy - 0.5, 0.8}), pteroRamp, 0.05)
	finishBeast(c, 0.75)
	return c, 10, gy
}

// pteroShadow is the dot of shade a flying reptile casts far below.
func pteroShadow() *canvas {
	c := newCanvas(20, 6)
	c.ellipse(10, 3, 9, 2.4, func(x, y int, nx, ny float64) color.RGBA {
		return color.RGBA{0, 0, 0, uint8(70 * (1 - (nx*nx + ny*ny)))}
	})
	return c
}

func genPlesio(pose int) sprite { c, ax, ay := plesioCanvas(pose); return mk(c, ax, ay) }

func plesioCanvas(pose int) (*canvas, float64, float64) {
	if pose != plesioUnder { // under the water it is only a dark shape, drawn below
		if c, ax, ay, ok := drawnDino("plesio", pose); ok {
			return c, ax, ay // drawn from dinos/; see dinorig.go
		}
	}
	c := newScaledCanvas(38, 32, plesioScale)
	const wy = 28.0 // the waterline
	if pose == plesioUnder {
		// A long dark shape: body, flippers and neck seen through the water.
		dark := func(x, y int) color.RGBA { return withAlpha(plesioRamp[0], 200) }
		c.ellipse(13, 26, 8, 3, func(x, y int, _, _ float64) color.RGBA { return dark(x, y) })
		c.polygon([]float64{9, 25, 4, 21.5, 7, 26}, dark)
		c.polygon([]float64{16, 27, 20, 31, 13, 28}, dark)
		tube(c, spine([3]float64{19, 25.5, 2}, [3]float64{25, 23, 1.5}, [3]float64{29, 22, 1.4}), ramp{plesioRamp[0]}, 0)
		return c, 14, wy
	}
	// The back breaking the surface, and its flippers stirring the water.
	c.ellipse(12, wy, 9, 3.6, func(x, y int, nx, ny float64) color.RGBA {
		if c.ly(y) >= wy {
			return color.RGBA{}
		}
		return plesioRamp.pick(lum(nx, ny)*0.9, x, y)
	})
	neck := spine([3]float64{17, 27, 3.2}, [3]float64{19, 18, 2.4}, [3]float64{21.5, 11, 2}, [3]float64{23.5, 7, 1.9})
	hx, hy, a, gape := 25.5, 6.5, 0.1, 0.0
	if pose == plesioStrike {
		neck = spine([3]float64{17, 27, 3.2}, [3]float64{22, 20, 2.4}, [3]float64{27, 15.5, 2}, [3]float64{30, 14.5, 1.9})
		hx, hy, a, gape = 31.5, 14.5, 0.25, 0.6
	}
	drawTubeWith(c, neck, func(x, y int, p *spinePt, t float64) color.RGBA {
		l := lum(p.nx*t*0.85, p.ny*t*0.85)
		if t > 0.55 {
			return mix(plesioRamp.pick(l, x, y), plesioRamp[5], 0.4) // the pale throat
		}
		return plesioRamp.pick(l, x, y)
	})
	roundShape(c, ell{hx, hy, 2.4, 1.8}, plesioRamp, 0.12)
	jaws(c, plesioRamp, hx+1, hy+0.4, a, 4.5, 1.3, gape, 0.08)
	c.set(int(hx), int(hy-0.6), colInk)
	// Ripples where neck and back meet the water.
	for x := 2; x < 26; x++ {
		if (x/2)%2 == 0 {
			c.set(x, int(wy)-1+x%2, withAlpha(hex(0xd8eef2), 200))
		}
	}
	finishBeast(c, 0.75)
	// Nothing below the waterline but the ripples.
	for y := int(math.Floor((wy + 1) * c.k())); y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.put(x, y, color.RGBA{})
		}
	}
	return c, 14, wy
}
