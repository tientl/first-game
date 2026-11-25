package client

import (
	"image/color"
	"math"
)

// HD art for the things that stand in the world. Two roads lead to an HD
// sprite of the environment:
//
//   - art drawn at HD: on a canvas HDDensity times finer (newScaledCanvas,
//     or a plain canvas marked with dens, drawn at three times the size by
//     code that works in pixels), with leaves, bark, grain and so on a third
//     of an art pixel fine (hdenv.go, treeart.go and others);
//   - art drawn at density 1 and refined by upscale3: the same drawing three
//     times as fine, its outline thinned to one image pixel, its staircase
//     edges cut to diagonals and its dithered shading redone at the finer
//     grain. That is for the things not redrawn by hand yet.
//
// mkUp picks the road from the canvas it is given, so a generator can be
// redrawn at HD without touching its callers.

// envHD turns the HD environment on. Off, every sprite made with mkUp is the
// original density-1 art (cmd/shot -envhd=false, to compare).
var envHD = true

// SetEnvHD turns the HD art of trees, plants, rocks and buildings on or
// off. Call it before the art is made.
func SetEnvHD(on bool) { envHD = on }

// hdK is the scale environment art is drawn at: HDDensity image pixels to
// an art pixel, or 1 when the HD environment is off.
func hdK() float64 {
	if envHD {
		return HDDensity
	}
	return 1
}

// newEnvCanvas is a canvas of w by h art pixels for a thing in the world,
// drawn hdK times finer: the drawing calls take art pixels (see
// newScaledCanvas).
func newEnvCanvas(w, h int) *canvas {
	c := newScaledCanvas(w, h, hdK())
	if envHD {
		c.lw = 2
	}
	return c
}

// mkUp makes the sprite of a thing in the world out of c, anchored at
// design point (ax, ay): as it is when c is drawn at HD, and refined by
// upscale3 when c is density-1 art.
func mkUp(c *canvas, ax, ay float64) sprite {
	switch {
	case c.dens == HDDensity: // pixels are image pixels, anchor too
		return spriteHD(c.image(), ax/HDDensity, ay/HDDensity)
	case c.k() == HDDensity:
		return mkHD(c, ax, ay)
	case !envHD || c.k() != 1 || c.tx != 0 || c.ty != 0:
		return mk(c, ax, ay)
	}
	return spriteHD(upscale3(c).image(), ax, ay)
}

// similarCol reports whether a and b are close enough to be two shades of
// one surface (neighbours on a ramp, dithered together).
func similarCol(a, b color.RGBA) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R)+d(a.G, b.G)+d(a.B, b.B) <= 110
}

// upscale3 redraws density-1 art three times as fine. Outline pixels (those
// canvas.outline drew) become lines one image pixel wide through the middle
// of the old ones, joined along rows, columns and diagonals, so a staircase
// turns into a slanted line; what they gave up goes to whatever lay on that
// side. Filled pixels are shaded anew: the dither between two shades of a
// ramp is smoothed out and laid again at the finer grain.
func upscale3(c *canvas) *canvas {
	const K = HDDensity
	o := newCanvas(c.w*K, c.h*K)
	o.dens = K
	const (
		clear = iota
		fill
		line
	)
	class := func(x, y int) int {
		if !c.in(x, y) || c.px[y*c.w+x].A == 0 {
			return clear
		}
		if c.ol != nil && c.ol[y*c.w+x] {
			return line
		}
		return fill
	}
	var gauss [3][3]float64 // weight of a neighbour at (d-1) pixels from sub-pixel i
	for i := range 3 {
		for d := range 3 {
			dist := float64(d-1) - (float64(i)-1)/K
			gauss[i][d] = math.Exp(-dist * dist / (2 * 0.65 * 0.65))
		}
	}
	shade := func(x, y, i, j int) color.RGBA {
		p := c.px[y*c.w+x]
		if p.A < 128 {
			return p
		}
		var sr, sg, sb, sw float64
		var cand [9]color.RGBA
		n := 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if class(x+dx, y+dy) != fill {
					continue
				}
				q := c.px[(y+dy)*c.w+x+dx]
				if q.A < 128 || !similarCol(p, q) {
					continue
				}
				w := gauss[i][dx+1] * gauss[j][dy+1]
				sr, sg, sb, sw = sr+w*float64(q.R), sg+w*float64(q.G), sb+w*float64(q.B), sw+w
				seen := false
				for _, k := range cand[:n] {
					if k == q {
						seen = true
						break
					}
				}
				if !seen {
					cand[n] = q
					n++
				}
			}
		}
		if n < 2 {
			return p
		}
		sr, sg, sb = sr/sw, sg/sw, sb/sw
		dist := func(q color.RGBA) float64 {
			return math.Abs(float64(q.R)-sr) + math.Abs(float64(q.G)-sg) + math.Abs(float64(q.B)-sb)
		}
		c1, c2 := cand[0], cand[1]
		d1, d2 := dist(c1), dist(c2)
		if d2 < d1 {
			c1, c2, d1, d2 = c2, c1, d2, d1
		}
		for _, q := range cand[2:n] {
			switch d := dist(q); {
			case d < d1:
				c2, d2, c1, d1 = c1, d1, q, d
			case d < d2:
				c2, d2 = q, d
			}
		}
		if d1+d2 > 0 && 0.65*dither(x*K+i, y*K+j)+0.35*hashf(x*K+i, y*K+j, 77) < d1/(d1+d2) {
			return c2
		}
		return c1
	}
	for y := range c.h {
		for x := range c.w {
			cl := class(x, y)
			if cl == clear {
				continue
			}
			p := c.px[y*c.w+x]
			// content is what lies at neighbour (nx, ny), to hand a part of
			// this outline pixel over to.
			content := func(nx, ny int) color.RGBA {
				switch class(nx, ny) {
				case clear:
					return color.RGBA{}
				case fill:
					return c.px[ny*c.w+nx]
				}
				return p
			}
			for j := range K {
				for i := range K {
					col := p
					if cl == fill {
						col = shade(x, y, i, j)
					} else if dx, dy := i-1, j-1; dx != 0 || dy != 0 {
						switch {
						case dy == 0:
							if class(x+dx, y) != line {
								col = content(x+dx, y)
							}
						case dx == 0:
							if class(x, y+dy) != line {
								col = content(x, y+dy)
							}
						default:
							l, v, d := class(x+dx, y) == line, class(x, y+dy) == line, class(x+dx, y+dy) == line
							switch {
							case l && v && d, !l && !v && d:
							case l && v:
								col = content(x+dx, y+dy)
							case l:
								col = content(x, y+dy)
							case v:
								col = content(x+dx, y)
							case class(x+dx, y) == clear || class(x, y+dy) == clear:
								col = color.RGBA{}
							default:
								col = content(x+dx, y)
							}
						}
					}
					if col.A > 0 {
						o.px[(y*K+j)*o.w+x*K+i] = col
					}
				}
			}
		}
	}
	return o
}
