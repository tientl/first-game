package client

import (
	"image/color"
	"math"
)

// The HD trial: a mossy stone with a flower on it, drawn twice from the same
// code, once as the original pixel art (density 1) and once at HDDensity,
// set down side by side next to the player. It proves the HD path: at 1080p
// the HD one lands pixel for pixel on the screen while everything round it
// is drawn as before. Off unless SetHDTest turns it on (cmd/shot -hdtest).

var hdTest bool

// SetHDTest shows the HD trial stones.
func SetHDTest(on bool) { hdTest = on }

var hdTestArt [2]sprite // density 1, HDDensity

// genHDTestStone draws the stone k times finer than art pixels. The shapes
// are in design units (art pixels); the canvas puts k pixels in each, so
// the same code makes both densities and only the fine one shows the grain,
// the veins of the moss and the petals.
func genHDTestStone(k float64) *canvas {
	const w, h = 26, 20
	c := newScaledCanvas(w, h, k)
	stone := mkRamp(0x2b2a33, 0x43414c, 0x5d5a66, 0x7b7784, 0x9b97a2, 0xbdb9c2)
	moss := mkRamp(0x24401f, 0x34592a, 0x4b7634, 0x679640, 0x8bb653)
	// The stone: a lumpy dome lit from the upper left, with fine grain.
	c.ellipse(13, 13, 11.5, 7, func(x, y int, nx, ny float64) color.RGBA {
		lx, ly := c.lx(x), c.ly(y)
		bump := 0.12*math.Sin(lx*1.7+ly*0.9) + 0.08*math.Sin(lx*3.1-ly*2.3)
		grain := 0.06 * (hashf(x, y, 11) - 0.5)
		t := lum(nx*0.9+bump, ny*1.1-0.2) + grain
		return stone.pick(t, x, y)
	})
	// A crack across its face.
	c.line(7, 12, 11, 14.5, func(x, y int, t float64) color.RGBA { return stone[0] })
	c.line(11, 14.5, 14, 13.5, func(x, y int, t float64) color.RGBA { return stone[1] })
	// Moss over the top, ragged at its lower edge.
	c.ellipse(12, 9.5, 9, 3.6, func(x, y int, nx, ny float64) color.RGBA {
		lx := c.lx(x)
		edge := 0.55 + 0.25*math.Sin(lx*2.3) + 0.15*math.Sin(lx*5.1)
		if ny > edge {
			return color.RGBA{}
		}
		t := lum(nx, ny-0.3) + 0.1*(hashf(x, y, 12)-0.5)
		return moss.pick(t, x, y)
	})
	// A flower: five petals round a yellow eye, on a short stem.
	petal := mkRamp(0xb8417a, 0xe06aa0, 0xf6a6c8, 0xfde0ee)
	c.line(17, 8.5, 17.5, 5, func(x, y int, t float64) color.RGBA { return moss[1] })
	for i := range 5 {
		a := float64(i)*2*math.Pi/5 - math.Pi/2
		px, py := 17.5+math.Cos(a)*1.4, 4.2+math.Sin(a)*1.1
		c.ellipse(px, py, 1.1, 0.9, func(x, y int, nx, ny float64) color.RGBA {
			return petal.pick(lum(nx, ny)*0.9+0.15, x, y)
		})
	}
	c.ellipse(17.5, 4.2, 0.65, 0.55, func(x, y int, nx, ny float64) color.RGBA {
		return mix(hex(0xf2c94c), hex(0xfff3b0), clamp01(-nx-ny))
	})
	c.outline(0.8)
	return c
}

func hdTestSprites() [2]sprite {
	if hdTestArt[0].img == nil {
		hdTestArt[0] = mk(genHDTestStone(1), 13, 19)
		hdTestArt[1] = mkHD(genHDTestStone(HDDensity), 13, 19)
	}
	return hdTestArt
}

// drawHDTest sets the two stones down by the player: the original art to
// the left, the HD one to the right.
func (g *Game) drawHDTest(dst drawTarget, cam camera) {
	if !hdTest {
		return
	}
	p, ok := g.me()
	if !ok {
		return
	}
	art := hdTestSprites()
	for i, dx := range [2]float64{-4, 2} {
		x, y := cam.toScreen(math.Floor(float64(p.X))+dx+0.5, math.Floor(float64(p.Y))+3.5)
		g.drawShadow(dst, x, y, 22)
		drawAt(dst, art[i], x, y, 1, 1, 0, nil)
	}
}
