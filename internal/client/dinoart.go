package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The grazing dinosaurs, side-on and facing right like the other animals:
// the three-horn, the plate-back and the long-neck. Bodies are lit spheres,
// tails, necks and legs are tubes (drawTubeWith in snake.go), so each pose
// is a few control points.

var (
	trikeRamp  = mkRamp(0x2c2a1e, 0x45412e, 0x5f5a40, 0x7a7454, 0x958e6a, 0xb0a986)
	frillRamp  = mkRamp(0x3e2a1c, 0x5c3a24, 0x7c4e2e, 0x9a6440, 0xb48058) // the frill: the hide flushed a warm brown
	hornRamp   = mkRamp(0x8d8068, 0xbfb396, 0xe6ddc4, 0xf8f2e2)
	stegoRamp  = mkRamp(0x262c1e, 0x38402c, 0x4e573d, 0x666f50, 0x808966)
	plateRamp  = mkRamp(0x4a221a, 0x6c3424, 0x8c4a32, 0xa86446, 0xc08560)
	brachRamp  = mkRamp(0x2a2a24, 0x3e3d34, 0x56544a, 0x6f6c60, 0x8a877a, 0xa6a394) // elephant grey with a brown cast
	brachBelly = mkRamp(0x8a8676, 0xa8a492, 0xc4c0ae)
)

// Pose indices of the grazers.
const (
	trikeWalkA = iota
	trikeWalkB
	trikeCharge // head down, horns levelled
	trikeGore   // head tossed up
	trikeEatA   // head down to the ground, beak shut...
	trikeEatB   // ...and open: chewing
	trikePoses
)

const (
	stegoWalkA = iota
	stegoWalkB
	stegoSwipe // the spiked tail swung up
	stegoEatA  // head down, chewing: shut...
	stegoEatB  // ...and open
	stegoPoses
)

const (
	brachWalkA = iota
	brachWalkB
	brachEatA // neck down to the ground, chewing: shut...
	brachEatB // ...and open
	brachPoses
)

// spine samples a path through control points {x, y, radius} finely enough
// for drawTubeWith, the radius changing smoothly along it.
func spine(ctrl ...[3]float64) []spinePt {
	var pts []spinePt
	for i := 0; i+1 < len(ctrl); i++ {
		a, b := ctrl[i], ctrl[i+1]
		n := max(1, int(math.Ceil(math.Hypot(b[0]-a[0], b[1]-a[1])*3)))
		for j := range n {
			t := float64(j) / float64(n)
			pts = append(pts, spinePt{x: lerpf(a[0], b[0], t), y: lerpf(a[1], b[1], t), r: lerpf(a[2], b[2], t)})
		}
	}
	last := ctrl[len(ctrl)-1]
	pts = append(pts, spinePt{x: last[0], y: last[1], r: last[2]})
	for i := range pts {
		pts[i].s = float64(i) / float64(len(pts)-1)
	}
	return withNormals(pts)
}

// tube draws a spine lit like a round limb in rp; bias brightens or darkens.
func tube(c *canvas, pts []spinePt, rp ramp, bias float64) {
	drawTubeWith(c, pts, func(x, y int, p *spinePt, t float64) color.RGBA {
		return rp.pick(lum(p.nx*t*0.85, p.ny*t*0.85)+bias, x, y)
	})
}

// leg draws a sturdy leg from the hip (x, y) down to the ground at gy, the
// foot swung forward by swing; far legs are darker.
func leg(c *canvas, rp ramp, x, y, gy, r, swing float64, far bool) {
	bias := 0.0
	if far {
		bias = -0.22
	}
	knee := [3]float64{x + swing*0.4, lerpf(y, gy, 0.55), r * 0.9}
	foot := [3]float64{x + swing, gy - r*0.6, r * 0.85}
	tube(c, spine([3]float64{x, y, r}, knee, foot), rp, bias)
	// Toenails.
	for k := -1.0; k <= 1; k++ {
		c.set(int(foot[0]+k*r*0.55+0.5), int(gy-0.5), hornRamp.at(1+int(bias*5)))
	}
}

// rot turns (x, y) by a about (ox, oy).
func rot(x, y, ox, oy, a float64) (float64, float64) {
	s, co := math.Sincos(a)
	dx, dy := x-ox, y-oy
	return ox + dx*co - dy*s, oy + dx*s + dy*co
}

// --- three-horn -------------------------------------------------------------

func genTrike(pose int) sprite { c, ax, ay := trikeCanvas(pose); return mk(c, ax, ay) }

// hornLook is a horned grazer's colours, frill and size: the three-horn,
// or its cousin the Torosaurus with its great open frill.
type hornLook struct {
	body, frill    ramp
	frillW, frillH float64 // the frill's half-width and half-height
	holes          bool    // two openings through the frill
	scale          float64
	rig            string // its drawing in dinos/; see dinorigs.go
}

var (
	trikeLook = hornLook{trikeRamp, frillRamp, 6.6, 7.6, false, trikeScale, "trike"}
	toroLook  = hornLook{toroRamp, toroFrill, 8.6, 9.8, false, toroScale, "toro"}
	toroRamp  = mkRamp(0x2b2420, 0x463a31, 0x645344, 0x84705b, 0xa48d74, 0xc0ab90)
	toroFrill = mkRamp(0x3a2420, 0x56342a, 0x744838, 0x905e4a, 0xa87a62)
)

func trikeCanvas(pose int) (*canvas, float64, float64) { return trikeCanvasOf(pose, trikeLook) }

func trikeCanvasOf(pose int, lk hornLook) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino(lk.rig, pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	trikeRamp := lk.body
	c := newScaledCanvas(48, 34, lk.scale)
	c.ty = 4 // room above for a big frill
	const gy = 28.0
	swing := 1.6
	if pose == trikeWalkB {
		swing = -1.6
	}
	if pose == trikeCharge || pose == trikeGore {
		swing = 2.2
	}
	leg(c, trikeRamp, 14.5, 19, gy, 2.6, swing, true)
	leg(c, trikeRamp, 29, 19, gy, 2.4, -swing, true)
	// The tail, low and thick.
	tube(c, spine([3]float64{12, 16, 5}, [3]float64{7.5, 17.5, 3.5}, [3]float64{3.5, 20, 2}, [3]float64{1, 22.5, 0.8}), trikeRamp, 0)
	roundShape(c, ell{15.5, 13.6, 8.6, 6.4}, trikeRamp, 0.06) // the hips stand higher than the shoulders
	roundShape(c, ell{21, 16, 12.5, 7.4}, trikeRamp, 0.04)
	// A pale belly and a few darker bands across the back.
	c.ellipse(21, 20.5, 10, 2.6, func(x, y int, nx, ny float64) color.RGBA {
		return mix(c.at(x, y), trikeRamp[5], 0.35)
	})
	for _, bx := range []float64{14, 19, 24} {
		c.line(bx, 9.6, bx+1.5, 13.5, func(x, y int, _ float64) color.RGBA { return mix(c.at(x, y), trikeRamp[0], 0.45) })
	}
	leg(c, trikeRamp, 16.5, 20, gy, 2.9, -swing, false)
	leg(c, trikeRamp, 30.5, 20, gy, 2.7, swing, false)

	// The head turns about the neck: down to charge, up to toss.
	hx, hy, a := 32.0, 14.0, 0.0
	switch pose {
	case trikeCharge:
		hy, a = 16.5, 0.38
	case trikeGore:
		hy, a = 12.5, -0.45
	case trikeEatA, trikeEatB:
		hy, a = 19.5, 0.8
	}
	trikeHead(c, hx, hy, a, lk, pose == trikeEatB)
	finishBeast(c, 0.8)
	return c, 22, gy
}

// trikeHead draws the frill, face, beak and horns about the neck (hx, hy),
// tilted by a (positive: nose down).
func trikeHead(c *canvas, hx, hy, a float64, lk hornLook, open bool) {
	trikeRamp, frillRamp := lk.body, lk.frill
	p := func(x, y float64) (float64, float64) { return rot(hx+x, hy+y, hx, hy, a) }
	// The frill: a big shield behind the face with a scalloped pale rim.
	fx, fy := p(0, -4.5*lk.frillH/7.6)
	c.ellipse(fx, fy, lk.frillW, lk.frillH, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		switch {
		case lk.holes && d < 0.34 && d > 0.1 && math.Abs(nx) > 0.28 && ny < -0.12:
			return color.RGBA{} // the openings through the frill
		case d > 0.78 && (int(math.Atan2(ny, nx)*4)+8)%2 == 0:
			return hornRamp.pick(lum(nx, ny)*0.6, x, y) // knobs round the edge
		case d > 0.5 && d < 0.62:
			return frillRamp[0] // a dark band
		}
		return frillRamp.pick(lum(nx*0.6, ny*0.6), x, y)
	})
	// The face.
	cx, cy := p(4.5, 0.5)
	roundShape(c, ell{cx, cy, 5.6, 4.2}, trikeRamp, 0.1)
	// The parrot beak, shut or open in a bite.
	beak := func(q [][2]float64, l float64) {
		var b []float64
		for _, v := range q {
			x, y := p(v[0], v[1])
			b = append(b, x, y)
		}
		c.polygon(b, func(x, y int) color.RGBA { return hornRamp.pick(l, x, y) })
	}
	if open {
		beak([][2]float64{{8, 1.6}, {11.5, 3.6}, {8, 4.6}}, 0.05) // the mouth
		beak([][2]float64{{8, -1.5}, {13, 1.6}, {9.5, 2.2}, {7.5, 1.6}}, 0.3)
		beak([][2]float64{{8, 4}, {11.5, 5.2}, {7.5, 5.8}}, 0.2)
	} else {
		beak([][2]float64{{8, -1.5}, {13, 1.8}, {9.5, 4.4}, {7.5, 3}}, 0.3)
	}
	// Horns: two long brow horns and a short one on the nose.
	horn := func(x0, y0, x1, y1, r, bias float64) {
		ax, ay := p(x0, y0)
		bx, by := p(x1, y1)
		tube(c, spine([3]float64{ax, ay, r}, [3]float64{bx, by, 0.35}), hornRamp, bias)
	}
	horn(3, -2.8, 10.5, -8.6, 1.1, -0.25) // the far brow horn
	horn(4, -2.2, 12, -7.6, 1.25, 0.1)
	horn(9, -0.6, 10.6, -3.2, 0.8, 0.05)
	ex, ey := p(4.6, -1.2)
	c.set(int(ex), int(ey), colInk)
}

// --- plate-back -------------------------------------------------------------

func genStego(pose int) sprite { c, ax, ay := stegoCanvas(pose); return mk(c, ax, ay) }

func stegoCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("stego", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(52, 34, stegoScale)
	const gy = 32.0
	swing := 1.4
	if pose == stegoWalkB {
		swing = -1.4
	}
	if pose == stegoSwipe {
		swing = 0.6
	}
	leg(c, stegoRamp, 15, 23, gy, 3, swing, true)
	leg(c, stegoRamp, 32, 24, gy, 2.4, -swing, true)

	// Plates down the back, tallest over the hips, far row first.
	top := func(x float64) float64 {
		u := (x - 22) / 14
		return 22 - math.Sqrt(max(0, 1-u*u))*9.5
	}
	plate := func(x, h, bias float64) {
		y := top(x) + 1.5
		c.polygon([]float64{x - 2.4, y, x - 0.4, y - h, x + 2.4, y - h*0.35, x + 1.6, y + 1},
			func(px, py int) color.RGBA {
				l := 0.75 - (c.ly(py)-(y-h))/h*0.5 + bias
				return plateRamp.pick(l, px, py)
			})
	}
	for _, x := range []float64{12, 17, 22, 27, 32} {
		plate(x+2.5, 5.5+3*math.Cos((x-21)/9), -0.25)
	}

	// The tail: up and back, or swung high with its spikes in the air.
	tailPts := [][3]float64{{11, 22, 5}, {5, 20, 2.6}, {0.8, 17, 1}}
	if pose == stegoSwipe {
		tailPts = [][3]float64{{11, 21, 5}, {6, 13, 2.6}, {4, 5, 1}}
	}
	tail := spine(tailPts...)
	tube(c, tail, stegoRamp, 0)
	// The four spikes near its tip, splayed.
	for i, s := range []float64{0.78, 0.9} {
		p := tail[int(s*float64(len(tail)-1))]
		for side := -1.0; side <= 1; side += 2 {
			dx, dy := -p.ny*side, p.nx*side // across the tail
			dx, dy = dx*0.8-0.5, dy*0.8-0.5
			l := 4.5 - float64(i)
			tube(c, spine([3]float64{p.x, p.y, 0.8}, [3]float64{p.x + dx*l, p.y + dy*l, 0.25}), hornRamp, -0.1)
		}
	}
	// The body: a high arched back over the hips.
	roundShape(c, ell{22, 22, 14, 8}, stegoRamp, 0.02)
	roundShape(c, ell{21, 18.5, 10, 6}, stegoRamp, 0.08)
	c.ellipse(23, 26.5, 11, 2.5, func(x, y int, nx, ny float64) color.RGBA {
		return mix(c.at(x, y), stegoRamp[4], 0.3) // the paler belly
	})
	for _, x := range []float64{14, 19.5, 25, 30.5} {
		plate(x, 7+3*math.Cos((x-20)/9), 0.05) // the near row
	}
	leg(c, stegoRamp, 17, 25, gy, 3.3, -swing, false)
	leg(c, stegoRamp, 33.5, 26, gy, 2.6, swing, false)
	// A small head on a low neck; down to the ground to eat.
	dy := 0.0
	if pose == stegoEatA || pose == stegoEatB {
		dy = 4.5
	}
	tube(c, spine([3]float64{34, 23, 3.6}, [3]float64{40, 25.5 + dy, 2.4}), stegoRamp, 0.05)
	roundShape(c, ell{43, 26 + dy, 3.6, 2.4}, stegoRamp, 0.12)
	c.polygon([]float64{45, 25 + dy, 48.5, 26.6 + dy, 45, 28 + dy}, func(x, y int) color.RGBA { return hornRamp.pick(0.25, x, y) })
	if pose == stegoEatB {
		c.polygon([]float64{45, 26.6 + dy, 48, 27.2 + dy, 45, 28.6 + dy}, solid(hex(0x3a1a14))) // a mouthful
	}
	c.set(43, int(25+dy), colInk)
	finishBeast(c, 0.8)
	return c, 24, gy
}

// --- long-neck --------------------------------------------------------------

func genBrachio(pose int) sprite { c, ax, ay := brachioCanvas(pose); return mk(c, ax, ay) }

func brachioCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("brachio", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(60, 62, brachioScale)
	const gy = 60.0
	swing, sway := 1.6, 0.0
	if pose == brachWalkB {
		swing, sway = -1.6, 0.8
	}
	leg(c, brachRamp, 15, 44, gy, 3.2, swing, true)
	leg(c, brachRamp, 32, 41, gy, 3.2, -swing, true)
	tube(c, spine([3]float64{13, 42, 5.5}, [3]float64{8, 44.5, 3.8}, [3]float64{4, 48.5, 2.2}, [3]float64{1.5, 53, 0.9}), brachRamp, 0)
	// The neck, rising from high shoulders to a small head.
	eating := pose == brachEatA || pose == brachEatB
	if eating { // the neck swung down to the ground
		tube(c, spine([3]float64{33, 37, 5}, [3]float64{39, 35, 3.6}, [3]float64{45, 41, 2.9}, [3]float64{48.5, 49, 2.5}), brachRamp, 0.06)
	} else {
		tube(c, spine([3]float64{33, 37, 5}, [3]float64{38 + sway*0.3, 25, 3.5}, [3]float64{43 + sway, 14, 2.8}, [3]float64{46 + sway, 8.5, 2.5}), brachRamp, 0.06)
	}
	roundShape(c, ell{24, 42, 13.5, 8.6}, brachRamp, 0.02)
	roundShape(c, ell{30, 38, 9, 7}, brachRamp, 0.08)
	c.ellipse(24, 47, 11, 3, func(x, y int, nx, ny float64) color.RGBA {
		if c.at(x, y).A < 128 {
			return color.RGBA{}
		}
		return mix(c.at(x, y), brachBelly.pick(0.5+nx*0.2, x, y), 0.35*(1-ny*ny))
	})
	// Faint dapples down the flank.
	for i := range 9 {
		x, y := 14+float64(i)*2.6, 38+math.Sin(float64(i)*1.7)*3
		c.set(int(x), int(y), brachRamp[1])
	}
	leg(c, brachRamp, 17, 46, gy, 3.6, -swing, false)
	leg(c, brachRamp, 34, 43, gy, 3.6, swing, false)
	// The head: a domed brow over a blunt snout.
	hx, hy := 49+sway, 7.5
	if eating {
		hx, hy = 51, 51
	}
	roundShape(c, ell{hx, hy, 4.2, 2.8}, brachRamp, 0.12)
	roundShape(c, ell{hx - 1, hy - 2.3, 2.2, 1.8}, brachRamp, 0.2)
	c.polygon([]float64{hx + 2, hy - 1.1, hx + 6, hy + 0.5, hx + 2, hy + 2.3}, func(x, y int) color.RGBA { return brachRamp.pick(0.45, x, y) })
	if pose == brachEatB {
		c.polygon([]float64{hx + 2, hy + 0.6, hx + 5, hy + 1.2, hx + 2, hy + 2.6}, solid(hex(0x3a1a14))) // a mouthful
	}
	c.set(int(hx), int(hy-1.5), colInk)
	finishBeast(c, 0.8)
	return c, 25, gy
}

// --- what they give ---------------------------------------------------------

var (
	hideRamp = mkRamp(0x5c3a22, 0x7f5432, 0xa27348, 0xc39563, 0xdcb682)
	toothArt = mkRamp(0x7a6a4a, 0xb9a77e, 0xe2d6b4, 0xf8f1dc)
)

// iconHide is a stretched hide: a pelt with four leg flaps and a few spots.
func iconHide() *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{
		1.5, 1.5, 5, 4.5, 11, 4.5, 14.5, 1.5, 12, 7, 12, 9.5, 14.5, 14.5,
		11, 11.5, 5, 11.5, 1.5, 14.5, 4, 9.5, 4, 7,
	}, func(x, y int) color.RGBA {
		l := 0.7 - math.Hypot(float64(x)-8, float64(y)-8)/9
		if hashf(x, y, 51) < 0.12 {
			l -= 0.35
		}
		return hideRamp.pick(l, x, y)
	})
	c.outline(0.8)
	return c
}

// iconBone is a big bone with knobbly ends.
func iconBone() *canvas {
	c := newCanvas(16, 16)
	tube(c, spine([3]float64{4, 11.5, 1.4}, [3]float64{11.5, 4, 1.4}), hornRamp, 0.15)
	for _, e := range []ell{{2.6, 11.2, 1.7, 1.7}, {4.6, 13.2, 1.7, 1.7}, {11.2, 2.6, 1.7, 1.7}, {13.2, 4.6, 1.7, 1.7}} {
		roundShape(c, e, hornRamp, 0.15)
	}
	c.outline(0.8)
	return c
}

// iconRexTooth is a tyrant's tooth: a long curved, serrated cone.
func iconRexTooth() *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{4, 13, 7, 14.5, 11, 13.5, 13.5, 2, 10.5, 6, 7, 9}, func(x, y int) color.RGBA {
		l := 0.85 - float64(y)/22
		if y >= 12 {
			return hideRamp.pick(0.35, x, y) // the root
		}
		if (x+y)%3 == 0 && x >= 10 {
			l -= 0.25 // serrations along the edge
		}
		return toothArt.pick(l, x, y)
	})
	c.outline(0.8)
	return c
}

// How much bigger than their design each beast is drawn: the great ones
// tower over a player, the hunters stand a little above them.
const (
	trikeScale   = 1.6
	stegoScale   = 1.6
	brachioScale = 1.8
	raptorScale  = 1.3
	rexScale     = 1.8
	pteroScale   = 1.4
	plesioScale  = 1.5
	mammothScale = 1.6
	sabreScale   = 1.35
)

// beastScale is how much bigger than its design a creature of kind k is
// drawn; 1 for the small animals.
func beastScale(k sim.CreatureKind) float64 {
	switch k {
	case sim.CreatureTrike:
		return trikeScale
	case sim.CreatureStego:
		return stegoScale
	case sim.CreatureBrachio:
		return brachioScale
	case sim.CreatureRaptor:
		return raptorScale
	case sim.CreatureRex:
		return rexScale
	case sim.CreaturePtero:
		return pteroScale
	case sim.CreaturePlesio:
		return plesioScale
	case sim.CreatureMammoth:
		return mammothScale
	case sim.CreatureSabre:
		return sabreScale
	case sim.CreatureToro:
		return toroScale
	case sim.CreatureAchero:
		return acheroScale
	case sim.CreatureOrnitho:
		return ornithoScale
	case sim.CreatureThesc:
		return thescScale
	case sim.CreatureAnzu:
		return anzuScale
	case sim.CreatureQuetzal:
		return quetzalScale
	case sim.CreatureBrachy:
		return brachyScale
	case sim.CreatureChampso:
		return champsoScale
	case sim.CreatureAnkylo:
		return ankyloScale
	}
	return 1
}
