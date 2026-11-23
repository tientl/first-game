package client

import "image/color"

// The tyrant's neighbours (sim/hellcreek.go). The Torosaurus, Acheroraptor,
// the pond crocodiles and the Quetzalcoatlus in flight are the three-horn,
// the raptor, the crocodile and the flying reptile in other colours and
// sizes; the Ornithomimus, Thescelosaurus, Anzu and the Quetzalcoatlus on
// the ground are drawn here.

const (
	toroScale       = 1.75
	acheroScale     = 1.0
	ornithoScale    = 1.45
	thescScale      = 1.3
	anzuScale       = 1.4
	quetzalScale    = 1.7
	quetzalFlyScale = 2.6
	brachyScale     = 0.95
	champsoScale    = 0.9
)

// Pose indices.
const (
	ornithoStand = iota
	ornithoRunA
	ornithoRunB
	ornithoRest // crouched, panting
	ornithoEatA // head down to the ground: beak shut...
	ornithoEatB // ...and open
	ornithoPoses
)

const (
	thescStand = iota
	thescWalkA
	thescWalkB
	thescEatA // head down, chewing: shut...
	thescEatB // ...and open
	thescPoses
)

const (
	anzuStand = iota
	anzuWalkA
	anzuWalkB
	anzuDisplay // crest up, wings spread
	anzuKick
	anzuEatA // head down to the ground: beak shut...
	anzuEatB // ...and open
	anzuPoses
)

const (
	quetzalWalkA = iota
	quetzalWalkB
	quetzalStab // neck down, beak driving forward
	quetzalEatA // beak to the ground: shut...
	quetzalEatB // ...and open, gulping
	quetzalPoses
)

var (
	ornithoRamp = mkRamp(0x3b3024, 0x5a4a38, 0x7c6a52, 0x9e8c70, 0xc0ae90, 0xdcd0b4)
	ornithoWing = mkRamp(0x2a221c, 0x40342a, 0x5a4a3c)
	thescRamp   = mkRamp(0x25301c, 0x384a2a, 0x4f663a, 0x6a8350, 0x8aa36c, 0xa9bd8a)
	anzuRamp    = mkRamp(0x15141a, 0x232129, 0x34313b, 0x48444f, 0x5e5966)
	anzuCrest   = mkRamp(0x5e1410, 0x92241a, 0xc23a24, 0xe8603a)
	anzuLeg     = mkRamp(0x5a4a30, 0x86703f, 0xb29654)
	quetzalBody = mkRamp(0x6a6660, 0x8e8a82, 0xb2aea4, 0xd2cec4, 0xeeeae2)
	quetzalWing = mkRamp(0x2e2a28, 0x4a4440, 0x6a625a, 0x8a8076, 0xa89c90)
	beakRamp    = mkRamp(0x7a6640, 0xa88c58, 0xd2b678, 0xeedca4)

	brachyLook = crocLook{
		mkRamp(0x17150f, 0x272418, 0x3a3622, 0x504a2e, 0x6a623c, 0x857c4e),
		mkRamp(0x7c7450, 0xa09868, 0xc4bc8a), brachyWidth, brachyScale,
	}
	champsoLook = crocLook{
		mkRamp(0x1d2418, 0x2f3a25, 0x445234, 0x5c6c46, 0x788a5c, 0x97a878),
		mkRamp(0x8e9070, 0xb2b490, 0xd2d4b0), champsoWidth, champsoScale,
	}
	quetzalFlyLook = pteroLook{quetzalWing, quetzalBody, anzuCrest, quetzalFlyScale, "quetzalFly"}
)

// brachyWidth is a blunt, short-snouted crocodile, broad in the jaw.
func brachyWidth(s float64) float64 {
	switch {
	case s < 0.5:
		return 0.5 + 2.6*smooth(s/0.5)
	case s < 0.7:
		return 3.1
	case s < 0.8:
		return lerpf(3.1, 2.4, (s-0.7)/0.1)
	default:
		return lerpf(2.6, 1.7, (s-0.8)/0.2) // the blunt crushing snout
	}
}

// champsoWidth is a slender crocodile-like reptile with a long, needle-thin
// snout, like a gharial's.
func champsoWidth(s float64) float64 {
	switch {
	case s < 0.45:
		return 0.4 + 2.0*smooth(s/0.45)
	case s < 0.62:
		return 2.4
	case s < 0.72:
		return lerpf(2.4, 1.7, (s-0.62)/0.1)
	default:
		return lerpf(1.3, 0.55, (s-0.72)/0.28) // the long, thin snout
	}
}

func genToro(p int) sprite    { c, ax, ay := trikeCanvasOf(p, toroLook); return mk(c, ax, ay) }
func genAchero(p int) sprite  { c, ax, ay := raptorCanvasOf(p, acheroLook); return mk(c, ax, ay) }
func genBrachy(p int) sprite  { c, ax, ay := crocCanvasOf(p, brachyLook); return mk(c, ax, ay) }
func genChampso(p int) sprite { c, ax, ay := crocCanvasOf(p, champsoLook); return mk(c, ax, ay) }
func genQuetzalFly(p int) sprite {
	c, ax, ay := pteroCanvasOf(p, quetzalFlyLook)
	return mk(c, ax, ay)
}

// stilt draws a long bird-like leg from hip through knee to foot, with toes.
func stilt(c *canvas, rp ramp, hip, knee, foot [2]float64, r, bias float64, toes color.RGBA) {
	tube(c, spine([3]float64{hip[0], hip[1], r}, [3]float64{knee[0], knee[1], r * 0.55}, [3]float64{foot[0], foot[1] - 0.6, r * 0.4}), rp, bias)
	c.line(foot[0]-1, foot[1]-0.4, foot[0]+2.2, foot[1]-0.4, func(int, int, float64) color.RGBA { return toes })
}

// --- Ornithomimus -----------------------------------------------------------

func genOrnitho(pose int) sprite { c, ax, ay := ornithoCanvas(pose); return mk(c, ax, ay) }

func ornithoCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("ornitho", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(34, 32, ornithoScale)
	const gy = 30.0
	type leg [3][2]float64 // hip, knee, foot
	near, far := leg{{14, 16}, {16.5, 22}, {14.5, gy}}, leg{{13, 16}, {15, 22}, {13, gy}}
	drop, lean := 0.0, 0.0
	switch pose {
	case ornithoRunA:
		near, far = leg{{14, 16}, {18.5, 20}, {21.5, gy - 2}}, leg{{13, 16}, {11.5, 22.5}, {7, gy - 1}}
		lean = 0.25
	case ornithoRunB:
		near, far = leg{{14, 16}, {11.5, 22.5}, {7.5, gy - 1}}, leg{{13, 16}, {18, 20}, {21, gy - 2}}
		lean = 0.25
	case ornithoRest:
		drop = 7
		near, far = leg{{14, 23}, {18, 27}, {12.5, gy}}, leg{{13, 23}, {17, 27}, {11.5, gy}}
	}
	toes := anzuLeg[1]
	stilt(c, anzuLeg, far[0], far[1], far[2], 1.7, -0.25, toes)
	// The tail held out straight, the feathered body, the long S of a neck.
	tube(c, spine([3]float64{10, 14 + drop, 2.4}, [3]float64{4, 13 + drop, 1.2}, [3]float64{0.5, 13.5 + drop, 0.5}), ornithoRamp, 0)
	c.ellipse(14, 14+drop, 6.4, 4.2, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if ny > 0.45 {
			l += 0.3 // a pale belly
		}
		return ornithoRamp.pick(l, x, y)
	})
	c.polygon([]float64{11, 12 + drop, 18, 12.5 + drop, 15.5, 16 + drop, 10, 15.5 + drop}, func(x, y int) color.RGBA {
		return ornithoWing.pick(0.6-float64(x)*0.01, x, y) // the folded wing feathers
	})
	n1, n2, hx, hy := [2]float64{21, 8 + drop}, [2]float64{22, 4 + drop}, 23.4, 3.0+drop
	if lean > 0 {
		n1, n2, hx, hy = [2]float64{22.5, 10}, [2]float64{25, 7}, 26.6, 6.4
	}
	if pose == ornithoRest {
		n1, n2, hx, hy = [2]float64{21, 13 + drop - 4}, [2]float64{22.5, 9 + drop - 4}, 24, 8.4+drop-4
	}
	if pose == ornithoEatA || pose == ornithoEatB {
		n1, n2, hx, hy = [2]float64{21.5, 15}, [2]float64{24, 22}, 25.5, 24.5
	}
	tube(c, spine([3]float64{18.5, 12.5 + drop, 1.8}, [3]float64{n1[0], n1[1], 1.3}, [3]float64{n2[0], n2[1], 1.1}), ornithoRamp, 0.12)
	roundShape(c, ell{hx, hy, 1.9, 1.4}, ornithoRamp, 0.2)
	openBeak(c, hx+1, hy, 3.4, 1.5, pose == ornithoEatB)
	c.set(int(hx), int(hy-0.4), colInk)
	// A clawed arm folded under the chest.
	tube(c, spine([3]float64{18, 15 + drop, 0.9}, [3]float64{20.5, 17.5 + drop, 0.6}), ornithoWing, 0)
	stilt(c, anzuLeg, near[0], near[1], near[2], 1.9, 0.05, toes)
	finishBeast(c, 0.8)
	return c, 14, gy
}

// --- Thescelosaurus ---------------------------------------------------------

func genThesc(pose int) sprite { c, ax, ay := thescCanvas(pose); return mk(c, ax, ay) }

func thescCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("thesc", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(36, 26, thescScale)
	const gy = 24.0
	swing := 1.5
	if pose == thescWalkB {
		swing = -1.5
	}
	if pose == thescStand {
		swing = 0
	}
	legAt := func(hx float64, sw, bias float64) {
		tube(c, spine([3]float64{hx, 14, 2.8}, [3]float64{hx + 1.5 + sw*0.5, 18.5, 1.5}, [3]float64{hx - 0.5 + sw, gy - 0.8, 1.1}), thescRamp, bias)
		c.line(hx-1.5+sw, gy-0.5, hx+1.8+sw, gy-0.5, func(int, int, float64) color.RGBA { return thescRamp[0] })
	}
	legAt(15, -swing, -0.25)
	tube(c, spine([3]float64{11, 12, 3}, [3]float64{4, 10, 1.6}, [3]float64{0.5, 9.5, 0.6}), thescRamp, 0)
	c.ellipse(16, 13, 7.5, 4.6, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if ny > 0.5 {
			l += 0.25
		}
		if ny < -0.2 && (x/2)%3 == 0 {
			l -= 0.25 // pale stripes broken by dark ones down the back
		}
		return thescRamp.pick(l, x, y)
	})
	hdy := 0.0 // head down to eat
	if pose == thescEatA || pose == thescEatB {
		hdy = 9
	}
	tube(c, spine([3]float64{21.5, 11, 2.4}, [3]float64{25, 8.8 + hdy*0.8, 2}), thescRamp, 0.08)
	roundShape(c, ell{27, 8.4 + hdy, 2.7, 2.1}, thescRamp, 0.15)
	openBeak(c, 28.8, 8.9+hdy, 2.8, 2.4, pose == thescEatB)
	c.set(27, int(7+hdy), colInk)
	tube(c, spine([3]float64{21, 13.5, 0.9}, [3]float64{22.5, 15.5, 0.6}), thescRamp, -0.05) // a small arm
	legAt(17, swing, 0.05)
	finishBeast(c, 0.8)
	return c, 17, gy
}

// --- Anzu -------------------------------------------------------------------

func genAnzu(pose int) sprite { c, ax, ay := anzuCanvas(pose); return mk(c, ax, ay) }

func anzuCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("anzu", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(30, 34, anzuScale)
	const gy = 32.0
	type leg [3][2]float64
	near, far := leg{{13, 19}, {15.5, 25}, {13.5, gy}}, leg{{12, 19}, {14, 25}, {12, gy}}
	switch pose {
	case anzuWalkA:
		near[2][0], far[2][0] = 16.5, 9.5
	case anzuWalkB:
		near[2][0], far[2][0] = 10, 16
	case anzuKick: // the near foot lashing out
		near = leg{{13, 19}, {18, 21}, {22, 23}}
	}
	stilt(c, anzuLeg, far[0], far[1], far[2], 1.8, -0.25, anzuLeg[0])
	// A fan of tail feathers, the dark feathered body.
	c.polygon([]float64{8, 14, 1.5, 9.5, 0.5, 13.5, 2, 17, 8, 18}, func(x, y int) color.RGBA {
		if x < 3 {
			return snowRamp.pick(0.7, x, y) // pale tips
		}
		return anzuRamp.pick(0.45, x, y)
	})
	roundShape(c, ell{13, 16, 6.2, 5.2}, anzuRamp, 0.05)
	// The wing: folded down the side, or raised and spread in display.
	wing := []float64{10, 13, 17, 12.5, 16, 19, 9.5, 18}
	if pose == anzuDisplay {
		wing = []float64{11, 13, 18, 12, 21, 4, 15, 6}
	}
	c.polygon(wing, func(x, y int) color.RGBA {
		l := 0.55 - float64(y)*0.01
		if (x+y)%3 == 0 {
			l -= 0.2
		}
		return anzuRamp.pick(l, x, y)
	})
	// Neck, head, the tall red crest and a toothless beak.
	hx, hy := 20.5, 6.0
	if pose == anzuDisplay {
		hx, hy = 19.5, 4.5
	}
	if pose == anzuEatA || pose == anzuEatB {
		hx, hy = 22.5, 15
	}
	tube(c, spine([3]float64{16.5, 13.5, 1.9}, [3]float64{19, 9.5, 1.5}, [3]float64{hx - 0.3, hy + 1, 1.4}), anzuRamp, 0.1)
	crestTop := hy - 6.5
	if pose == anzuDisplay {
		crestTop -= 1.5
	}
	c.polygon([]float64{hx - 1.4, hy - 0.6, hx - 0.4, crestTop, hx + 1.8, crestTop + 1, hx + 1.6, hy - 0.8}, func(x, y int) color.RGBA {
		return anzuCrest.pick(0.8-c.ly(y)*0.0+float64(x)*0.01, x, y)
	})
	roundShape(c, ell{hx, hy, 2.1, 1.8}, mkRamp(0x6a7890, 0x8a98ae, 0xaab8ca), 0.1) // the bare blue-grey face
	openBeak(c, hx+1.2, hy+0.7, 2.8, 2.2, pose == anzuEatB)
	c.set(int(hx), int(hy-0.3), colInk)
	stilt(c, anzuLeg, near[0], near[1], near[2], 2, 0.05, anzuLeg[0])
	finishBeast(c, 0.8)
	return c, 14, gy
}

// --- Quetzalcoatlus on the ground -------------------------------------------

func genQuetzal(pose int) sprite { c, ax, ay := quetzalCanvas(pose); return mk(c, ax, ay) }

func quetzalCanvas(pose int) (*canvas, float64, float64) {
	if c, ax, ay, ok := drawnDino("quetzal", pose); ok {
		return c, ax, ay // drawn from dinos/; see dinorig.go
	}
	c := newScaledCanvas(48, 58, quetzalScale)
	const gy = 56.0
	sw := 1.5
	if pose == quetzalWalkB {
		sw = -1.5
	}
	if pose == quetzalStab {
		sw = 0.5
	}
	// Far limbs first: a hind leg, and a folded wing walked on like a crutch.
	tube(c, spine([3]float64{17, 36, 1.8}, [3]float64{16 - sw*0.3, 45, 1.2}, [3]float64{15 - sw, gy - 0.6, 0.9}), quetzalWing, -0.3)
	tube(c, spine([3]float64{23, 30, 2.1}, [3]float64{27 + sw*0.3, 41, 1.5}, [3]float64{25 + sw, gy - 0.6, 1}), quetzalWing, -0.3)
	// The wing membrane folded along the forelimb, then the small body.
	c.polygon([]float64{22, 29, 28, 41, 26, 49, 20, 38}, func(x, y int) color.RGBA { return quetzalWing.pick(0.45, x, y) })
	roundShape(c, ell{20.5, 33, 5.2, 4.2}, quetzalBody, 0.05)
	// The great neck and head: up and forward, or driving down to stab.
	n := [][3]float64{{24.5, 29, 2.3}, {28.5, 17, 1.8}, {31, 8, 1.6}}
	hx, hy, a := 32.0, 6.5, 0.15
	if pose == quetzalStab {
		n = [][3]float64{{24.5, 29, 2.3}, {31, 29, 1.8}, {35, 33, 1.6}}
		hx, hy, a = 36.5, 34.5, 0.6
	}
	if pose == quetzalEatA || pose == quetzalEatB {
		n = [][3]float64{{24.5, 29, 2.3}, {30.5, 31, 1.8}, {33.5, 38, 1.6}}
		hx, hy, a = 34.5, 40.5, 0.95
	}
	tube(c, spine(n...), quetzalBody, 0.1)
	p := func(x, y float64) (float64, float64) { return rot(hx+x, hy+y, hx, hy, a) }
	var beak []float64
	beakPts := [][2]float64{{0.5, -1.4}, {11, 0.4}, {0.5, 1.2}}
	if pose == quetzalEatB { // the bill parted: a gulp
		beakPts = [][2]float64{{0.5, -1.4}, {11, -1.2}, {0.5, 0.2}}
		var lower []float64
		for _, q := range [][2]float64{{0.5, 0.6}, {10, 2.6}, {0.5, 1.6}} {
			x, y := p(q[0], q[1])
			lower = append(lower, x, y)
		}
		c.polygon(lower, func(x, y int) color.RGBA { return beakRamp.pick(0.45, x, y) })
	}
	for _, q := range beakPts {
		x, y := p(q[0], q[1])
		beak = append(beak, x, y)
	}
	c.polygon(beak, func(x, y int) color.RGBA { return beakRamp.pick(0.6, x, y) })
	var crest []float64
	for _, q := range [][2]float64{{-0.5, -1.2}, {-4, -3.5}, {-1, 0.2}} {
		x, y := p(q[0], q[1])
		crest = append(crest, x, y)
	}
	c.polygon(crest, func(x, y int) color.RGBA { return anzuCrest.pick(0.7, x, y) })
	roundShape(c, ell{hx, hy, 2.4, 1.9}, quetzalBody, 0.15)
	ex, ey := p(0.3, -0.5)
	c.set(int(ex), int(ey), colInk)
	// Near limbs.
	tube(c, spine([3]float64{19, 36, 2}, [3]float64{18 + sw*0.3, 45, 1.3}, [3]float64{17 + sw, gy - 0.6, 1}), quetzalWing, 0)
	tube(c, spine([3]float64{24, 31, 2.3}, [3]float64{28 - sw*0.3, 42, 1.6}, [3]float64{27 - sw, gy - 0.6, 1.1}), quetzalWing, 0.05)
	finishBeast(c, 0.8)
	return c, 21, gy
}

// openBeak draws a beak from its base (x, y), len long and h deep, pointing
// right: shut, or parted in a bite with the mouth showing between.
func openBeak(c *canvas, x, y, length, h float64, open bool) {
	if !open {
		c.polygon([]float64{x, y - h/2, x + length, y + h*0.1, x, y + h/2}, func(px, py int) color.RGBA { return beakRamp.pick(0.5, px, py) })
		return
	}
	c.polygon([]float64{x, y - h*0.2, x + length*0.8, y + h*0.4, x, y + h*0.9}, solid(hex(0x3a1a14)))
	c.polygon([]float64{x, y - h/2, x + length, y - h*0.15, x, y + h*0.05}, func(px, py int) color.RGBA { return beakRamp.pick(0.55, px, py) })
	c.polygon([]float64{x, y + h*0.55, x + length*0.85, y + h*0.95, x, y + h*1.05}, func(px, py int) color.RGBA { return beakRamp.pick(0.4, px, py) })
}
