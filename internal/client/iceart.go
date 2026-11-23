package client

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The ice age: snowy ground, snow falling over it, frosted pines, the
// mammoth and the sabre-tooth, and what they give.

var (
	snowRamp    = mkRamp(0x8fa3b8, 0xaebfd0, 0xc9d6e2, 0xdfe8f0, 0xf2f6fa, 0xffffff)
	mammothRamp = mkRamp(0x2a1a10, 0x43291a, 0x5f3c25, 0x7e5433, 0x9e6e45, 0xbb8a5c)
	ivoryRamp   = mkRamp(0x9c8b68, 0xc9b994, 0xe8dcc0, 0xfbf5e6)
	sabreRamp   = mkRamp(0x40291a, 0x63412a, 0x8a5f3c, 0xae7f52, 0xcda16e, 0xe6c596)
)

// Pose indices.
const (
	mammothWalkA = iota
	mammothWalkB
	mammothCharge // head down, tusks levelled
	mammothGore   // head tossed up
	mammothEatA   // the trunk curled down, pulling grass...
	mammothEatB   // ...and up to the mouth
	mammothPoses
)

const (
	sabreStand = iota
	sabreWalkA
	sabreWalkB
	sabreStalk  // crouched, creeping
	sabreSpring // in the air, jaws open, claws out
	sabreCower  // flat on the ground, ears back
	sabreEatA   // crouched over a kill: jaws shut...
	sabreEatB   // ...and open, tearing
	sabrePoses
)

// snowGround colours a pixel of snow: soft drifts, blue in their hollows,
// and a few glints.
func snowGround(px, py int, u, v float64) color.RGBA {
	n := fbm2(u*0.25, v*0.32, 171, 3)
	i := snowRamp.idx(0.62+(n-0.5)*0.8, px, py)
	r := math.Sin((v*1.1 + u*0.25 + fbm2(u*0.35, v*0.45, 172, 2)*2.5) * 2 * math.Pi)
	if r > 0.94 && hashf(px, py, 176) < 0.7 {
		i-- // the shadowed lee of a drift, faint and broken
	}
	if hashf(px, py, 173) < 0.012 {
		return snowRamp[5] // a glint
	}
	return snowRamp.at(i)
}

// --- mammoth ----------------------------------------------------------------

func genMammoth(pose int) sprite { c, ax, ay := mammothCanvas(pose); return mk(c, ax, ay) }

func mammothCanvas(pose int) (*canvas, float64, float64) {
	c := newScaledCanvas(60, 50, mammothScale)
	const gy = 48.0
	swing := 1.8
	if pose == mammothWalkB {
		swing = -1.8
	}
	if pose == mammothCharge || pose == mammothGore {
		swing = 2.4
	}
	leg(c, mammothRamp, 15, 34, gy, 3.6, swing, true)
	leg(c, mammothRamp, 32, 33, gy, 3.6, -swing, true)
	tube(c, spine([3]float64{11, 26, 2}, [3]float64{8, 32, 1.2}), mammothRamp, -0.1) // the tail
	// A body of shaggy fur, the back rising to a hump over the shoulders,
	// long hair hanging below its belly.
	hair := func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny) * 0.95
		if (x+int(ny*4))%3 == 0 {
			l -= 0.15 // strands
		}
		return mammothRamp.pick(l, x, y)
	}
	c.ellipse(24, 28, 15, 11, hair)
	c.ellipse(31, 21, 9, 8.5, hair)
	for x := 10; x <= 38; x++ { // the fringe
		n := 2 + int(hashf(x, 0, 174)*4)
		for y := 37; y < 37+n; y++ {
			if c.atL(x, y-1).A > 0 || y == 37 {
				c.set(x, y, mammothRamp.at(1+x%2))
			}
		}
	}
	leg(c, mammothRamp, 18, 37, gy, 3.9, -swing, false)
	leg(c, mammothRamp, 34, 36, gy, 3.9, swing, false)

	// The head about its neck, with tusks and trunk.
	hx, hy, a := 39.0, 22.0, 0.0
	switch pose {
	case mammothCharge:
		hy, a = 25, 0.45
	case mammothGore:
		hy, a = 20, -0.4
	case mammothEatA, mammothEatB:
		hy, a = 23, 0.2
	}
	p := func(x, y float64) (float64, float64) { return rot(hx+x, hy+y, hx, hy, a) }
	tusk := func(bias, off float64) {
		x0, y0 := p(4, 5+off)
		x1, y1 := p(11, 9+off)
		x2, y2 := p(15, 3+off)
		tube(c, spine([3]float64{x0, y0, 1.5}, [3]float64{x1, y1, 1.2}, [3]float64{x2, y2, 0.6}), ivoryRamp, bias)
	}
	tusk(-0.25, -1) // the far tusk
	tx0, ty0 := p(5, 3)
	tx1, ty1 := p(8, 11)
	tx2, ty2 := p(7, 19)
	tx3, ty3 := p(9.5, 21)
	switch pose {
	case mammothGore: // the trunk raised
		tx1, ty1 = p(10, 0)
		tx2, ty2 = p(14, -5)
		tx3, ty3 = p(16, -3)
	case mammothEatA: // reaching forward, clear of the legs, curled round a tuft on the ground
		tx1, ty1 = p(10, 11)
		tx2, ty2 = p(12.5, 20)
		tx3, ty3 = p(10, 22)
	case mammothEatB: // curled up to the mouth with it
		tx1, ty1 = p(8.5, 10)
		tx2, ty2 = p(6.5, 8)
		tx3, ty3 = p(4, 6)
	}
	tube(c, spine([3]float64{tx0, ty0, 2.6}, [3]float64{tx1, ty1, 2}, [3]float64{tx2, ty2, 1.4}, [3]float64{tx3, ty3, 1}), mammothRamp, 0.05)
	sx, sy := p(1, -2)
	c.ellipse(sx, sy, 6.5, 7, hair) // the head
	dx, dy := p(1, -7)
	c.ellipse(dx, dy, 4, 3.4, hair) // the domed crown
	ex, ey := p(3, -1.5)
	c.set(int(ex), int(ey), colInk)
	tusk(0.1, 0)
	finishBeast(c, 0.85)
	return c, 26, gy
}

// --- sabre-tooth ------------------------------------------------------------

func genSabre(pose int) sprite { c, ax, ay := sabreCanvas(pose); return mk(c, ax, ay) }

func sabreCanvas(pose int) (*canvas, float64, float64) {
	c := newScaledCanvas(36, 24, sabreScale)
	gy := 22.0
	drop, tilt, gape := 0.0, 0.0, 0.0
	type limb [2][2]float64                                      // shoulder or hip, then the paw
	fl, fr := limb{{24, 13}, {25, 21}}, limb{{10, 13}, {9, 21}}  // near legs
	bl, br := limb{{22, 13}, {21, 21}}, limb{{12, 13}, {13, 21}} // far legs
	switch pose {
	case sabreWalkA:
		fl[1][0], fr[1][0], bl[1][0], br[1][0] = 27, 7, 19, 15
	case sabreWalkB:
		fl[1][0], fr[1][0], bl[1][0], br[1][0] = 22, 12, 26, 6
	case sabreStalk:
		drop = 4
		fl, fr = limb{{24, 15}, {27, 21}}, limb{{10, 15}, {8, 21}}
		bl, br = limb{{22, 15}, {20, 21}}, limb{{12, 15}, {14, 21}}
	case sabreSpring:
		tilt, gape = -0.2, 0.7
		fl, fr = limb{{24, 12}, {31, 15}}, limb{{10, 12}, {3, 16}}
		bl, br = limb{{22, 12}, {29, 16}}, limb{{12, 12}, {5, 15}}
	case sabreEatA, sabreEatB: // crouched over a kill, head down
		drop, tilt, gape = 4, 0.2, 0.1
		if pose == sabreEatB {
			gape = 0.65
		}
		fl, fr = limb{{24, 15}, {27, 21}}, limb{{10, 15}, {8, 21}}
		bl, br = limb{{22, 15}, {20, 21}}, limb{{12, 15}, {14, 21}}
	case sabreCower:
		drop = 7
		fl, fr = limb{{24, 18}, {28, 21}}, limb{{10, 18}, {6, 21}}
		bl, br = limb{{22, 18}, {26, 21}}, limb{{12, 18}, {8, 21}}
	}
	p := func(x, y float64) (float64, float64) { return rot(x, y+drop, 17, 11+drop, tilt) }
	limbAt := func(l limb, bias float64) {
		x0, y0 := p(l[0][0], l[0][1])
		x1, y1 := l[1][0], min(l[1][1], gy-1)
		if pose == sabreSpring {
			x1, y1 = p(l[1][0], l[1][1])
		}
		tube(c, spine([3]float64{x0, y0, 2}, [3]float64{(x0 + x1) / 2, (y0 + y1) / 2, 1.5}, [3]float64{x1, y1, 1.3}), sabreRamp, bias)
	}
	limbAt(bl, -0.25)
	limbAt(br, -0.25)
	tx0, ty0 := p(8, 10)
	tx1, ty1 := p(4, 8)
	tube(c, spine([3]float64{tx0, ty0, 1.4}, [3]float64{tx1, ty1, 1}), sabreRamp, 0) // a short tail
	bx, by := p(17, 11)
	c.ellipse(bx, by, 9, 4.4, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if hashf(x/2, y, 175) < 0.12 && ny < 0.4 {
			l -= 0.25 // faint dapples
		}
		if ny > 0.5 {
			l += 0.25 // a pale belly
		}
		return sabreRamp.pick(l, x, y)
	})
	limbAt(fl, 0.05)
	limbAt(fr, 0.05)
	// The head: ears, a broad muzzle and the two great fangs.
	hx, hy := p(26, 8)
	switch pose {
	case sabreStalk, sabreCower:
		hy += 2
	case sabreEatA, sabreEatB:
		hy += 4
	}
	roundShape(c, ell{hx, hy, 3.8, 3.2}, sabreRamp, 0.12)
	for _, e := range [][2]float64{{-2.2, -3}, {0.2, -3.4}} {
		c.polygon([]float64{hx + e[0] - 1, hy + e[1] + 1.5, hx + e[0] + 0.5, hy + e[1] - 0.6, hx + e[0] + 1.6, hy + e[1] + 1.5},
			func(x, y int) color.RGBA { return sabreRamp.pick(0.45, x, y) })
	}
	jaws(c, sabreRamp, hx+2, hy+0.6, tilt, 4, 1.8, gape, 0.1)
	for _, f := range []float64{3, 4.2} { // the sabres
		fx := hx + f
		top := hy + 1.4
		for y := top; y <= top+3.6; y++ {
			c.set(int(fx), int(y), ivoryRamp[3-int(y-top)/2])
		}
	}
	c.set(int(hx+1), int(hy-0.6), hex(0xe0c040)) // a yellow eye
	finishBeast(c, 0.8)
	if pose == sabreSpring {
		gy += 3 // in the air
	}
	return c, 17, gy
}

// --- what they give ---------------------------------------------------------

// iconFur is a thick, shaggy pelt.
func iconFur() *canvas {
	c := newCanvas(16, 16)
	c.ellipse(8, 8.5, 6.5, 5.5, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if (x+y/2)%3 == 0 {
			l -= 0.25
		}
		return mammothRamp.pick(l, x, y)
	})
	for x := 3; x <= 13; x += 2 { // a ragged edge of long hair
		c.set(x, 14, mammothRamp[2])
	}
	c.outline(0.8)
	return c
}

// iconIvory is a curved tusk.
func iconIvory() *canvas {
	c := newCanvas(16, 16)
	tube(c, spine([3]float64{2, 13, 1.8}, [3]float64{7, 12, 1.6}, [3]float64{12, 8, 1.2}, [3]float64{13.5, 3, 0.6}), ivoryRamp, 0.1)
	c.outline(0.8)
	return c
}

// iconFurCoat is a coat of fur with a hide belt.
func iconFurCoat() *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{2, 4, 6, 2, 10, 2, 14, 4, 13, 9, 12.5, 14.5, 3.5, 14.5, 3, 9}, func(x, y int) color.RGBA {
		l := 0.75 - (float64(x)-8)/8*0.3
		if (x+y/2)%3 == 0 {
			l -= 0.2
		}
		return mammothRamp.pick(l, x, y)
	})
	c.line(4, 9, 12, 9, func(x, y int, _ float64) color.RGBA { return hideRamp.pick(0.6, x, y) })
	c.line(6, 3, 10, 3, func(x, y int, _ float64) color.RGBA { return snowRamp[4] }) // pale fur at the collar
	c.outline(0.8)
	return c
}

// iconIvorySpear is a spear tipped with a long ivory point.
func iconIvorySpear() *canvas {
	c := newCanvas(16, 16)
	c.line(1, 15, 10, 6, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.8, x, y) })
	c.line(2, 15, 11, 6, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	c.polygon([]float64{9, 6, 15.5, 0.5, 11, 7.5}, func(x, y int) color.RGBA { return ivoryRamp.pick(0.95-float64(y)*0.05, x, y) })
	c.set(9, 7, hideRamp[2])
	c.set(10, 7, hideRamp[1])
	c.outline(0.85)
	return c
}

// genHeldIvorySpear is the ivory spear as held, like genHeldSpear.
func genHeldIvorySpear() sprite {
	c := newCanvas(6, 28)
	c.rect(2, 7, 2, 21, func(x, y int) color.RGBA { return woodRamp[1+(3-x)] })
	c.polygon([]float64{3, 0, 5, 6, 3, 8.5, 1, 6}, func(x, y int) color.RGBA { return ivoryRamp.pick(0.95-float64(x)*0.1, x, y) })
	c.set(2, 8, hideRamp[2])
	c.set(3, 9, hideRamp[1])
	c.outline(0.8)
	return mk(c, 3, 22)
}

// --- snow falling -----------------------------------------------------------

// snowCover is how much of the view around the camera is snow, 0 to 1.
func (g *Game) snowCover(cam camera) float64 {
	w, h := float64(g.world.Bounds().Dx()), float64(g.world.Bounds().Dy())
	n, s := 0, 0
	for y := 0.1; y < 1; y += 0.2 {
		for x := 0.1; x < 1; x += 0.2 {
			tx, ty := (w*x+cam.ox)/TileW, (h*y+cam.oy)/TileH
			n++
			if g.tileAt(int(math.Floor(tx)), int(math.Floor(ty))) == sim.TileSnow {
				s++
			}
		}
	}
	return float64(s) / float64(n)
}

// drawSnowfall drifts flakes down over the snowy parts of the world. Like
// the embers, each flake is a function of its index and the clock.
func (g *Game) drawSnowfall(dst *ebiten.Image, cam camera) {
	cover := g.snowCover(cam)
	if cover <= 0 || g.caveIn() > 0.5 {
		return
	}
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	n := int(140 * cover)
	t := float64(g.frame) / 60
	for i := range n {
		r := rand.New(rand.NewPCG(uint64(i), 177))
		speed := 12 + r.Float64()*14
		x := r.Float64()*w + math.Sin(t*0.8+float64(i))*6 + t*4
		y := r.Float64()*h + t*speed
		x, y = math.Mod(x+cam.ox*0.15, w), math.Mod(y+cam.oy*0.15, h)
		if x < 0 {
			x += w
		}
		if y < 0 {
			y += h
		}
		a := uint8(150 + r.IntN(100))
		size := float32(1)
		if r.Float64() < 0.2 {
			size = 2
		}
		vector.FillRect(dst, float32(math.Round(x)), float32(math.Round(y)), size, size, color.RGBA{a, a, a, a}, false)
	}
}

// --- sounds -----------------------------------------------------------------

// genIceSFX adds the ice age's sounds to the effect bank.
func genIceSFX(r *rand.Rand, add func(name string, peak float64, x []float64)) {
	// A mammoth's trumpet: a bright, rising blare.
	add("trumpet", 0.9, growl(r, 1.1, 260, 420, 0.06, 0.5, 0.35, [2]float64{900, 1900}))
	// A sabre-tooth's snarl.
	add("snarl", 0.85, growl(r, 0.6, 140, 95, 0.03, 0.25, 0.9, [2]float64{500, 1100}))
	// A crunch of snow underfoot.
	x := make([]float64, samples(0.18))
	addAt(x, grains(r, 0.14, 500, 0.006, bandpass, 2600, 1.2, func(t float64) float64 { return env(t, 0.01, 0.05) }), 0, 1)
	add("step_snow", 1, x)
}

// iceSound plays what a mammoth or a sabre-tooth did when its state changed
// from o to c, and reports whether c is one of them.
func iceSound(o, c protocol.CreatureView) bool {
	x, y := float64(c.X), float64(c.Y)
	switch c.Kind {
	case sim.CreatureMammoth:
		switch {
		case c.State == sim.CreatureChase && o.State != sim.CreatureStrike:
			snd.playAt("trumpet", x, y, 0.8, 0.1)
		case c.State == sim.CreatureStrike:
			snd.playAt("whack", x, y, 0.9, 0.08)
		}
	case sim.CreatureSabre:
		switch {
		case c.State == sim.CreatureChase && calm(o.State): // it has picked its quarry
			if snd.ready("sabre_roar", 4*time.Second) {
				callAt("sabre_roar", x, y, 0.85, 1, 22)
			}
		case c.State == sim.CreatureStrike:
			callAt("sabre_snarl", x, y, 0.9, 1, 16) // see predsound.go
		case c.State == sim.CreatureCower:
			snd.playAt("hiss", x, y, 0.4, 0.2)
		}
	default:
		return false
	}
	return true
}
