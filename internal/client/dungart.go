package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Dung: its bag icon, the heaps on the ground with flies about them, and a
// bed of soil that has been fed with it. See sim/dung.go.

var dungRamp = mkRamp(0x24160a, 0x3e2812, 0x5a3c1c, 0x7a5428, 0x96703a)

// genDungIcon draws the bag icon of sim.ItemDung: a coiled heap with bits of
// chewed grass in it.
func genDungIcon(icons *[sim.NumItems]*ebiten.Image) {
	c := newCanvas(16, 16)
	for _, e := range []ell{{8, 12, 6, 2.6}, {7.6, 9, 4.6, 2.2}, {8.2, 6.4, 3, 1.9}, {8.8, 4.4, 1.4, 1.1}} {
		roundShape(c, e, dungRamp, 0.1)
	}
	straw := color.RGBA{0x9a, 0x9a, 0x3a, 0xff}
	for _, p := range [][2]int{{4, 12}, {11, 11}, {6, 9}, {9, 6}} {
		c.put(p[0], p[1], straw)
	}
	c.put(6, 5, dungRamp.at(4)) // a wet shine
	c.put(10, 8, dungRamp.at(4))
	c.outline(0.8)
	icons[sim.ItemDung] = c.image()
}

// fedSoil is dark flecks of dung over a bed of soil, drawn over the soil.
var fedSoil sprite

func fedSoilSprite() sprite {
	if fedSoil.img != nil {
		return fedSoil
	}
	c := newCanvas(22, 15)
	for y := 1; y < c.h-1; y++ {
		for x := 1; x < c.w-1; x++ {
			nx, ny := (float64(x)+0.5-11)/9.5, (float64(y)+0.5-7.5)/6
			if nx*nx+ny*ny > 1 {
				continue
			}
			switch h := sim.HashTile(x*7+3, y*13+5) % 9; h {
			case 0, 1:
				c.put(x, y, dungRamp.at(int(h)))
			case 2:
				c.put(x, y, color.RGBA{0x7a, 0x7a, 0x30, 0xff}) // straw
			}
		}
	}
	fedSoil = mk(c, 11, 8)
	return fedSoil
}

// fly is one pixel of a fly.
var fly *ebiten.Image

// drawFlies draws two or three flies buzzing about a heap of dung at
// (sx, sy) in the art canvas.
func (g *Game) drawFlies(dst *ebiten.Image, sx, sy float64, seed uint32) {
	if fly == nil {
		fly = ebiten.NewImage(1, 1)
		fly.Fill(color.RGBA{0x14, 0x14, 0x18, 0xff})
	}
	t := float64(g.frame)
	for i := range 2 + int(seed%2) {
		k := float64(i)*2.1 + float64(seed%97)
		a := t*(0.09+0.02*float64(i)) + k
		x := sx + math.Cos(a)*(3+float64(i)) + math.Sin(a*2.7)*1.2
		y := sy - 9 - float64(i)*2 + math.Sin(a*1.3)*2.5
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(x), math.Round(y))
		dst.DrawImage(fly, op)
	}
}
