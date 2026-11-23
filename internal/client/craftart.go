package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Art and sounds for the crafted tools, foods and buildings: pickaxe, torch,
// spear, fishing rod, cooked food, wooden wall and grass bed. Icons are
// 16x16 art pixels like every other bag icon.

var (
	roastRamp = mkRamp(0x3a1a0e, 0x6b3418, 0x9a5426, 0xc47c3c, 0xe2a865)
	charCol   = hex(0x2a1710)
	ropeRamp  = mkRamp(0x6e6231, 0x9a8a46, 0xc4b264)
)

// --- icons ------------------------------------------------------------------

// genCraftIcons draws the bag icons of the crafted tools and roast berries.
// (Cooked meat and fish are assigned with the animals' icons.)
func genCraftIcons(icons *[sim.NumItems]*ebiten.Image) {
	icons[sim.ItemTorch] = iconTorch()
	icons[sim.ItemSpear] = iconSpear()
	icons[sim.ItemRod] = iconRod()
	icons[sim.ItemRoastBerry] = iconRoastBerries()
}

func iconTorch() *ebiten.Image {
	c := newCanvas(16, 16)
	c.line(5, 15, 9, 6, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.75, x, y) })
	c.line(6, 15, 10, 6, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	c.ellipse(10, 5.5, 2.4, 2, func(x, y int, nx, ny float64) color.RGBA { return strawRamp.pick(lum(nx, ny)*0.8, x, y) })
	c.ellipse(11, 2.6, 2.2, 2.6, func(x, y int, nx, ny float64) color.RGBA {
		return fireRamp.pick(1-math.Hypot(nx, ny)*0.7-(ny+1)*0.15, x, y)
	})
	c.outline(0.7)
	return c.image()
}

func iconSpear() *ebiten.Image {
	c := newCanvas(16, 16)
	c.line(1, 15, 11, 5, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.8, x, y) })
	c.line(2, 15, 12, 5, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	c.polygon([]float64{10, 4.5, 15.5, 0.5, 13, 7}, func(x, y int) color.RGBA {
		return stoneRamp.pick(0.95-float64(y)*0.06+float64(x-12)*0.04, x, y)
	})
	c.set(10, 6, ropeRamp[2])
	c.set(11, 6, ropeRamp[1])
	c.outline(0.85)
	return c.image()
}

func iconRod() *ebiten.Image {
	c := newCanvas(16, 16)
	c.line(1, 15, 12, 1, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.85-0.4*0, x, y) })
	c.line(2, 15, 12, 2, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	line := hex(0xe8eef2)
	for y := 2; y <= 12; y++ {
		c.set(13+int(0.6*math.Sin(float64(y)*0.5)), y, line)
	}
	c.set(13, 13, hex(0xd83a3a)) // the float
	c.set(13, 14, colWhite)
	c.ellipse(4, 12, 2, 2, func(x, y int, nx, ny float64) color.RGBA { return ropeRamp.pick(lum(nx, ny), x, y) })
	c.outline(0.7)
	return c.image()
}

// grillMarks draws two dark diagonal bars across what is already on c.
func grillMarks(c *canvas, off float64) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			v := float64(x+y) - off
			if c.opaque(x, y) && (math.Abs(v-12) < 0.6 || math.Abs(v-16) < 0.6) {
				c.set(x, y, charCol)
			}
		}
	}
}

func iconCookedMeat() *ebiten.Image {
	c := newCanvas(16, 16)
	bone, boneSh := hex(0xece4d2), hex(0xbfb39b)
	c.line(8, 8, 13, 13, func(int, int, float64) color.RGBA { return bone })
	c.line(8, 9, 13, 14, func(int, int, float64) color.RGBA { return boneSh })
	c.set(14, 13, bone)
	c.set(13, 14, boneSh)
	c.ellipse(6.5, 6.5, 5.5, 5, func(x, y int, nx, ny float64) color.RGBA {
		return roastRamp.pick(lum(nx, ny)*0.9+0.08, x, y)
	})
	grillMarks(c, 0)
	c.outline(0.85)
	return c.image()
}

func iconCookedFish() *ebiten.Image {
	c := newCanvas(16, 16)
	c.polygon([]float64{5.5, 8, 0.5, 3.6, 0.5, 12.4}, func(x, y int) color.RGBA { return roastRamp.pick(0.35, x, y) })
	c.ellipse(9.6, 8, 5.8, 3.4, func(x, y int, nx, ny float64) color.RGBA {
		return roastRamp.pick(lum(nx, ny)*0.85+0.15, x, y)
	})
	grillMarks(c, 2)
	c.set(13, 7, colInk)
	c.outline(0.85)
	return c.image()
}

func iconRoastBerries() *ebiten.Image {
	c := newCanvas(16, 16)
	c.line(1, 15, 14, 2, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.7, x, y) })
	dark := mkRamp(0x3a0a18, 0x6e1a2c, 0xa8344a, 0xe07a7a)
	for i, b := range [][2]float64{{5, 11}, {8, 8}, {11, 5}} {
		c.ellipse(b[0], b[1], 2.6, 2.6, func(x, y int, nx, ny float64) color.RGBA {
			if hashf(x, y, 900+uint32(i)) < 0.15 {
				return charCol
			}
			return dark.pick(lum(nx, ny), x, y)
		})
	}
	c.outline(0.85)
	return c.image()
}

// --- world sprites ------------------------------------------------------------

// genWall is one tile of palisade: sharpened stakes lashed together.
func genWall() sprite {
	c := newCanvas(24, 30)
	const by = 26
	for i := 0; i < 5; i++ {
		x0 := 1 + i*5
		top := 4 + int(hashf(i, 0, 910)*4)
		for y := top; y <= by; y++ {
			for dx := 0; dx < 4; dx++ {
				// Sharpened point: narrow at the very top.
				if y-top < 2 && (dx == 0 || dx == 3) {
					continue
				}
				l := 0.85 - float64(dx)*0.2
				if hashf(x0+dx, y/3, 911) < 0.18 {
					l -= 0.2
				}
				c.set(x0+dx, y, trunkRamp.pick(l, x0+dx, y))
			}
		}
		c.set(x0+1, top, woodRamp[2]) // fresh-cut tip
	}
	for _, ry := range []int{11, 21} {
		for x := 1; x < 24; x++ {
			if c.opaque(x, ry) {
				c.set(x, ry, ropeRamp.at(1+(x/2)%2))
			}
		}
	}
	c.outline(0.8)
	return mk(c, 12, by)
}

// genBed is a mattress of dry grass in a log frame, with a rolled pillow.
func genBed() sprite {
	c := newCanvas(30, 20)
	const cx, by = 15, 16
	c.rect(1, 6, 28, 11, func(x, y int) color.RGBA { return trunkRamp.pick(0.75-float64(y-6)*0.05, x, y) })
	c.rect(3, 6, 24, 8, func(x, y int) color.RGBA {
		l := 0.55 + (vnoise(float64(x)*0.7, float64(y)*0.9, 920)-0.5)*0.6
		if (x+y)%5 == 0 {
			l += 0.15
		}
		return strawRamp.pick(l, x, y)
	})
	c.ellipse(7, 8, 4, 2.4, func(x, y int, nx, ny float64) color.RGBA {
		return mix(hex(0xd9cfb4), hex(0xf4eedb), lum(nx, ny))
	})
	c.rect(1, 15, 28, 2, func(x, y int) color.RGBA { return trunkRamp.pick(0.3, x, y) })
	c.outline(0.8)
	return mk(c, cx, by)
}

// genBlanket is drawn over a sleeping player.
func genBlanket() sprite {
	c := newCanvas(20, 10)
	quilt := mkRamp(0x5a3a6a, 0x7a5290, 0x9c72b4, 0xc29ad6)
	c.rect(1, 1, 18, 8, func(x, y int) color.RGBA {
		l := 0.6 - float64(y-1)*0.06
		if (x/3+y/3)%2 == 0 {
			l += 0.2
		}
		return quilt.pick(l, x, y)
	})
	c.outline(0.8)
	return mk(c, 10, 8)
}

// Held tools are drawn rotated about their grip, like the axe.

func genHeldSpear() sprite {
	c := newCanvas(6, 26)
	c.rect(2, 6, 2, 20, func(x, y int) color.RGBA { return woodRamp[1+(3-x)] })
	c.polygon([]float64{3, 0, 5.5, 5, 3, 7.5, 0.5, 5}, func(x, y int) color.RGBA { return stoneRamp.pick(0.95-float64(x)*0.1, x, y) })
	c.set(2, 7, ropeRamp[2])
	c.set(3, 8, ropeRamp[1])
	c.outline(0.8)
	return mk(c, 3, 20)
}

func genHeldTorch() sprite {
	c := newCanvas(6, 14)
	c.rect(2, 4, 2, 10, func(x, y int) color.RGBA { return woodRamp[1+(3-x)] })
	c.ellipse(3, 3.5, 2.6, 2.2, func(x, y int, nx, ny float64) color.RGBA { return strawRamp.pick(lum(nx, ny)*0.7, x, y) })
	c.outline(0.75)
	return mk(c, 3, 12)
}

func genHeldRod() sprite {
	c := newCanvas(4, 28)
	for y := 0; y < 28; y++ {
		c.set(1, y, woodRamp.pick(0.85, 1, y))
		if y > 8 {
			c.set(2, y, trunkRamp.pick(0.5, 2, y))
		}
	}
	c.outline(0.7)
	return mk(c, 2, 24)
}

// --- sounds -----------------------------------------------------------------

// genCraftSFX adds the sounds of the crafted things to the effect bank.
func genCraftSFX(r *rand.Rand, add func(name string, peak float64, x []float64)) {
	{ // Torch catching: a short whoosh and a few crackles.
		flt := newFilter(lowpass, 400, 0.8)
		x := render(0.8, func(t float64, _ int) float64 {
			flt.set(lowpass, 400+2600*math.Sin(math.Pi*min(1, t/0.5)), 0.8)
			return flt.do(r.Float64()*2-1) * env(t, 0.06, 0.18)
		})
		for g := 0; g < 8; g++ {
			addAt(x, noiseBurst(r, 0.02, bandpass, 1800+r.Float64()*3500, 2, 0.0002, 0.003), 0.1+r.Float64()*0.6, 0.4+r.Float64()*0.5)
		}
		add("torch_on", 1, x)
	}
	{ // Food on the fire: a sizzle that swells and fades, with spits.
		hp := newFilter(highpass, 3000, 0.7)
		x := render(1.6, func(t float64, _ int) float64 {
			return hp.do(r.Float64()*2-1) * env(t, 0.15, 0.6) * (0.75 + 0.25*fsin(9*t))
		})
		for g := 0; g < 18; g++ {
			addAt(x, noiseBurst(r, 0.02, bandpass, 3000+r.Float64()*4000, 2.5, 0.0002, 0.002), r.Float64()*1.3, 0.6+r.Float64()*0.6)
		}
		add("sizzle", 1, x)
	}
	{ // A spear thrust: a fast airy swish.
		flt := newFilter(bandpass, 800, 1.2)
		x := render(0.25, func(t float64, _ int) float64 {
			flt.set(bandpass, 700+5000*t/0.25, 1.4)
			return flt.do(r.Float64()*2-1) * math.Sin(math.Pi*t/0.25)
		})
		add("swish", 1, x)
	}
	{ // Casting a line, then the bobber landing.
		flt := newFilter(bandpass, 1500, 1)
		x := render(0.9, func(t float64, _ int) float64 {
			if t > 0.35 {
				return 0
			}
			flt.set(bandpass, 3500-2500*t/0.35, 1.5)
			return flt.do(r.Float64()*2-1) * math.Sin(math.Pi*t/0.35) * 0.6
		})
		addAt(x, tone(0.2, 520, 260, 0.08, 0.002, 0.05), 0.55, 0.8)
		addAt(x, noiseBurst(r, 0.2, lowpass, 1200, 0.7, 0.003, 0.05), 0.55, 0.5)
		add("cast", 1, x)
	}
	{ // Pickaxe on stone: brighter and shorter than the bare-hand hit.
		x := partials(0.3, 1900, []float64{1, 2.41, 3.9, 5.8}, []float64{1, 0.7, 0.45, 0.3}, []float64{0.07, 0.05, 0.03, 0.02})
		addAt(x, noiseBurst(r, 0.03, bandpass, 4200, 1.2, 0.0002, 0.004), 0, 1)
		add("pick_hit", 1, x)
	}
	{ // Lying down: a soft descending lullaby of three bells.
		x := make([]float64, samples(2.2))
		for i, m := range []float64{76, 72, 67} {
			addAt(x, bell(midiFreq(m), 1.6, 2, 1, 0.8), float64(i)*0.32, 0.7)
		}
		add("sleep", 1, x)
	}
	{ // Waking up at dawn: a rising major arpeggio.
		x := make([]float64, samples(1.8))
		for i, m := range []float64{67, 71, 74, 79} {
			addAt(x, bell(midiFreq(m), 1.2, 3.5, 1.2, 0.5), float64(i)*0.09, 0.8)
		}
		add("wake", 1, x)
	}
	{ // Planting a stake: two wooden thunks.
		x := make([]float64, samples(0.45))
		for i, t := range []float64{0, 0.12} {
			addAt(x, tone(0.2, 180, 95, 0.05, 0.001, 0.045), t, 1-float64(i)*0.2)
			addAt(x, partials(0.15, 330, []float64{1, 1.6, 2.7}, []float64{0.6, 0.3, 0.1}, []float64{0.05, 0.03, 0.02}), t, 0.8)
		}
		add("build", 1, x)
	}
}
