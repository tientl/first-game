package client

import (
	"image/color"
	"math"
)

// finishBeast gives a great beast's drawing a living skin and soft edges, so
// it reads less like a cartoon: fine scales and broad blotches over the
// hide, light catching the ridge of its back, shade gathering under its
// belly, jaw and limbs, and an outline drawn from its own darkest colours
// rather than black ink. It replaces the plain outline at the end of each
// beast's drawing.
func finishBeast(c *canvas, strength float64) {
	k := c.k()
	solid := func(x, y int) bool { return c.at(x, y).A >= 128 }
	out := make([]color.RGBA, len(c.px))
	copy(out, c.px)
	cell := max(2, int(math.Round(k*1.6))) // scales grow with the drawing
	shadeDepth := max(2, int(math.Round(2.5*k)))
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			p := c.px[y*c.w+x]
			if p.A < 128 {
				continue
			}
			f := 1.0
			// Scales: staggered cells, each a touch lighter or darker, with
			// faint seams between them.
			row := y / cell
			sx := x + (row%2)*cell/2
			f += (hashf(sx/cell, row, 201) - 0.5) * 0.12
			if sx%cell == 0 || y%cell == 0 {
				f -= 0.035
			}
			// Broad blotches and mottling across the hide.
			f += (fbm2(float64(x)/(7*k), float64(y)/(7*k), 202, 2) - 0.5) * 0.22
			// Light along the top of the back; shade under the belly and limbs.
			if !solid(x, y-1) {
				f += 0.14
			} else if !solid(x, y-2) {
				f += 0.06
			}
			for d := 1; d <= shadeDepth; d++ {
				if !solid(x, y+d) {
					f -= 0.16 * (1 - float64(d-1)/float64(shadeDepth))
					break
				}
			}
			out[y*c.w+x] = shadeRGB(p, f)
		}
	}
	c.px = out
	c.outline(strength * 0.55)
}

// shadeRGB scales a colour's brightness by f, keeping it in range.
func shadeRGB(p color.RGBA, f float64) color.RGBA {
	s := func(v uint8) uint8 { return uint8(min(255, max(0, math.Round(float64(v)*f)))) }
	return color.RGBA{s(p.R), s(p.G), s(p.B), p.A}
}

// dot sets the one pixel at design point (x, y): finer than set, which on a
// scaled canvas fills the whole block a design unit covers. For teeth.
func dot(c *canvas, x, y float64, col color.RGBA) {
	px, py := c.toPx(x, y)
	c.put(int(math.Floor(px)), int(math.Floor(py)), col)
}
