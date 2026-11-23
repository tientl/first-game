package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The crab: a shore crab seen from above at the game's 3/4 angle, a wide
// red-orange shell on eight spindly legs, two big claws held out in front
// and eyes on stalks. It walks sideways, so one sprite serves both ways; the
// legs alternate between two frames, and a third pose raises the claws wide
// open to pinch.

var (
	crabRamp  = mkRamp(0x4a140e, 0x7e2416, 0xb33e22, 0xd8622e, 0xf0904a, 0xf8c07a)
	crabBelly = mkRamp(0xc8a070, 0xe4c79a)
	crabLeg   = mkRamp(0x5c1a10, 0x8e3020)
)

// Crab sprite indices into creatureArt.crab.
const (
	crabWalkA = iota
	crabWalkB
	crabPinch
	crabPoses
)

func genCrab(pose int) sprite {
	c, ax, ay := crabCanvas(pose)
	return mk(c, ax, ay)
}

// crabCanvas draws crab pose on a canvas and returns its anchor.
func crabCanvas(pose int) (*canvas, float64, float64) {
	c := newCanvas(28, 20)
	cx, cy := 14.0, 11.5 // the shell's centre

	// Claws first: an arm reaching forward and out, then a big pincer.
	for side := -1.0; side <= 1; side += 2 {
		px, py := cx+side*7.4, cy-4.4
		if pose == crabPinch {
			px, py = cx+side*8.2, cy-7.6 // raised high and wide
		}
		c.line(cx+side*3, cy-2, px-side*1.4, py+1, func(x, y int, t float64) color.RGBA {
			return crabRamp.pick(0.45+t*0.15, x, y)
		})
		c.line(cx+side*3, cy-1, px-side*1.4, py+2, func(x, y int, t float64) color.RGBA {
			return crabRamp.pick(0.3, x, y)
		})
		c.ellipse(px, py, 2.6, 1.9, func(x, y int, nx, ny float64) color.RGBA {
			return crabRamp.pick(lum(nx*side, ny)*0.9+0.15, x, y)
		})
		// The gap between the fingers at the tip, wide open when pinching.
		gap := 1.0
		if pose == crabPinch {
			gap = 2.0
		}
		for k := 0.0; k < gap; k++ {
			c.set(int(math.Floor(px+side*(1.8-k*0.5))), int(math.Floor(py-0.4+k*0.6)), color.RGBA{})
		}
	}

	// The shell: wide and flat, a pale front rim, a groove down the middle.
	c.ellipse(cx, cy, 5.6, 3.4, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx*0.9, ny*0.9-0.15) + (vnoise(float64(x)*0.9, float64(y)*0.9, 610)-0.5)*0.18
		switch {
		case ny > 0.6:
			return crabBelly.pick(l, x, y)
		case ny < -0.7:
			l += 0.15
		case math.Abs(nx) < 0.12 && ny < 0.3:
			l -= 0.2
		}
		return crabRamp.pick(l+0.1, x, y)
	})
	// Eyes on stalks at the front edge.
	for side := -1.0; side <= 1; side += 2 {
		ex := int(math.Floor(cx + side*1.7))
		c.set(ex, int(cy-3.4), crabRamp[2])
		c.set(ex, int(cy-4.4), colInk)
	}
	c.outline(0.85)

	// Legs last, thin and dark, and only on bare canvas so the shell stays
	// in front: four a side, a knee lifted high, the foot on the sand. The
	// walk frames swap which legs are raised.
	for side := -1.0; side <= 1; side += 2 {
		for i := range 4 {
			lift := 0.0
			if pose != crabPinch && (i+pose)%2 == int(side+1)/2 {
				lift = 1
			}
			bx, by := cx+side*(4.8-0.35*float64(i)), cy-0.6+1.2*float64(i)
			kx, ky := bx+side*(3.2+0.3*float64(i)), by-1.3-lift
			fx, fy := kx+side*(1.5-0.2*float64(i)), by+2.2-lift
			paint := func(x, y int, t float64) color.RGBA {
				if c.at(x, y).A > 0 {
					return color.RGBA{}
				}
				return crabLeg.pick(0.8-t*0.6, x, y)
			}
			c.line(bx, by, kx, ky, paint)
			c.line(kx, ky, fx, fy, paint)
		}
	}
	return c, cx, cy + 3
}

// iconCrabMeat is a big red claw broken off at the joint, white meat showing.
func iconCrabMeat() *ebiten.Image {
	c := newCanvas(16, 16)
	c.ellipse(7, 8.5, 5.2, 4.2, func(x, y int, nx, ny float64) color.RGBA {
		return crabRamp.pick(lum(nx, ny)*0.95+0.1, x, y)
	})
	// The fingers: a thick upper one and a thinner lower one, a gap between.
	c.polygon([]float64{10, 4.5, 15, 2.5, 15.5, 4.5, 11.5, 7}, func(x, y int) color.RGBA { return crabRamp.pick(0.75, x, y) })
	c.polygon([]float64{11, 9, 15, 8, 15, 9.5, 11.5, 10.5}, func(x, y int) color.RGBA { return crabRamp.pick(0.5, x, y) })
	// The broken end, meat inside.
	c.ellipse(2.6, 10.5, 2, 2.2, func(x, y int, nx, ny float64) color.RGBA {
		if nx*nx+ny*ny > 0.6 {
			return crabRamp[1]
		}
		return crabBelly.pick(0.9-ny*0.3, x, y)
	})
	c.outline(0.8)
	return c.image()
}

// iconCookedCrab is a whole crab roasted bright red, steam rising off it.
func iconCookedCrab() *ebiten.Image {
	c := newCanvas(16, 16)
	cooked := mkRamp(0x6e1810, 0xb02a18, 0xe0462a, 0xf87a48, 0xffb070)
	for side := -1.0; side <= 1; side += 2 {
		c.ellipse(8+side*5.2, 7.5, 2, 1.6, func(x, y int, nx, ny float64) color.RGBA { return cooked.pick(lum(nx, ny)+0.1, x, y) })
		for i := range 3 {
			bx, by := 8+side*3.6, 10.0+float64(i)
			c.line(bx, by, bx+side*3.2, by+1.5, func(x, y int, t float64) color.RGBA { return cooked[1] })
		}
	}
	c.ellipse(8, 10, 5, 3.4, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny) + 0.05
		if (x+y)%5 == 0 && ny < 0.3 {
			l -= 0.25 // char from the fire
		}
		return cooked.pick(l, x, y)
	})
	for _, p := range [][2]int{{5, 3}, {6, 2}, {10, 3}, {11, 2}} {
		c.set(p[0], p[1], color.RGBA{232, 232, 232, 200})
	}
	c.outline(0.8)
	return c.image()
}
