package client

import (
	"image/color"
	"math"
)

// Animal homes: a rabbit burrow, a mound of dug-out earth with a dark hole
// in its front, and a snake den, a dark crevice under a pile of stones with
// a shed skin by the entrance.

var (
	earthRamp = mkRamp(0x3b2a1c, 0x5a4029, 0x7a5a3a, 0x9a7a52, 0xb89a6c)
	holeRamp  = mkRamp(0x0d0907, 0x1d140e, 0x2e2117)
	skinRamp2 = mkRamp(0xa49c84, 0xcfc7ad, 0xece6d2)
)

// genBurrow draws a rabbit burrow: the hole faces the viewer, the mound of
// dug earth rises behind it, with a few blades of grass and crumbs about.
func genBurrow() sprite { c, ax, ay := burrowCanvas(); return mk(c, ax, ay) }

// burrowCanvas draws the burrow and returns its anchor.
func burrowCanvas() (*canvas, float64, float64) {
	c := newCanvas(26, 16)
	const cx, cy = 13.0, 11.0
	// The mound, lit from the top left, lumpy at the edge.
	c.ellipse(cx, cy-1.5, 11, 5.5, func(x, y int, nx, ny float64) color.RGBA {
		if nx*nx+ny*ny > 0.82+0.18*vnoise(float64(x)*0.6, float64(y)*0.6, 501) {
			return color.RGBA{}
		}
		l := lum(nx*0.9, ny*0.9-0.2) + (vnoise(float64(x)*0.5, float64(y)*0.5, 502)-0.5)*0.25
		return earthRamp.pick(l, x, y)
	})
	// The hole: dark, deepest at the back, with a lit lip of earth above.
	c.ellipse(cx+0.5, cy+0.2, 4.2, 2.9, func(x, y int, nx, ny float64) color.RGBA {
		if ny < -0.55 {
			return earthRamp.pick(0.75, x, y)
		}
		return holeRamp.pick(0.15+0.6*(ny+1)/2, x, y)
	})
	// Loose earth thrown out in front.
	for _, p := range [][2]int{{6, 13}, {8, 14}, {18, 13}, {20, 14}, {11, 14}, {16, 14}} {
		c.set(p[0], p[1], earthRamp[2+(p[0]%2)])
	}
	c.outline(0.7)
	// Grass growing up the sides, after the outline so the blades stay fine.
	for _, b := range [][3]float64{{2, 11, -0.3}, {3, 11, 0.2}, {23, 11, 0.3}, {24, 11, -0.2}} {
		c.line(b[0], b[1], b[0]+b[2]*3, b[1]-3, func(x, y int, t float64) color.RGBA {
			return grassRamp.pick(0.35+t*0.6, x, y)
		})
	}
	return c, cx, cy + 2
}

// genDen draws a snake den: stones piled over a low black crevice, and the
// pale crumple of a shed skin trailing out of it.
func genDen() sprite { c, ax, ay := denCanvas(); return mk(c, ax, ay) }

// denCanvas draws the den and returns its anchor.
func denCanvas() (*canvas, float64, float64) {
	c := newCanvas(30, 20)
	const cx, cy = 15.0, 15.0
	stones := []ell{{7, 12, 5, 4}, {23, 12.5, 5.5, 4}, {15, 7.5, 7, 5}, {10, 6, 3.5, 3}, {20.5, 5.5, 3.5, 3}}
	// The crevice first, so the stones overhang it.
	c.ellipse(cx, cy-1.5, 6.5, 3, func(x, y int, nx, ny float64) color.RGBA {
		return holeRamp.pick(0.2+0.5*(ny+1)/2, x, y)
	})
	for i, s := range stones {
		seed := uint32(510 + i)
		c.ellipse(s.x, s.y, s.rx, s.ry, func(x, y int, nx, ny float64) color.RGBA {
			l := lum(nx, ny) + (vnoise(float64(x)*0.45, float64(y)*0.45, seed)-0.5)*0.25
			col := stoneRamp.pick(l*0.95, x, y)
			if ny < -0.4 && vnoise(float64(x)*0.4, float64(y)*0.5, seed+50) > 0.55 {
				col = mossRamp.pick(l, x, y)
			}
			return col
		})
	}
	// The mouth of the crevice under the stones.
	c.ellipse(cx, cy-1, 4.2, 1.8, func(x, y int, nx, ny float64) color.RGBA {
		return holeRamp.pick(0.1+0.3*(ny+1)/2, x, y)
	})
	// A shed skin trailing out: a pale wavy ribbon with faint scale marks.
	for s := 0.0; s <= 1; s += 0.04 {
		x := cx + 2 + s*9
		y := cy + 1.5 + math.Sin(s*7)*1.2
		col := skinRamp2.pick(0.4+0.5*(1-s), int(x), int(y))
		if int(s*25)%3 == 0 {
			col = skinRamp2[0]
		}
		c.set(int(x), int(y), col)
		if s < 0.6 {
			c.set(int(x), int(y)+1, skinRamp2[0])
		}
	}
	c.outline(0.75)
	return c, cx, cy + 1
}
