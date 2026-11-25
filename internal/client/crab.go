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
	crabLeg   = mkRamp(0x4a140e, 0x7e2416, 0xb33e22, 0xd8622e)
)

// Crab sprite indices into creatureArt.crab.
const (
	crabWalkA = iota
	crabWalkB
	crabPinch
	crabPoses
)

func genCrab(pose int) sprite { return critter("crab", 0, pose, 0) }

// crabClaws are the right claw seen from above, shut and gaping: a round
// palm with a thick fixed finger and a thinner one that works against it.
var crabClaws = [2][]string{
	{
		".HH..h",
		"HHHdhh",
		"HMMdhm",
		"MMMMmm",
		".Mmmm.",
	},
	{
		"HH...h",
		"HHH..h",
		"HMM.hh",
		"MMMMhm",
		"MMMmmm",
		".Mmmm.",
	},
}

// crabCanvas draws crab pose on a canvas and returns its anchor.
func crabCanvas(pose int) (*canvas, float64, float64) {
	c := newCanvas(28, 20)
	cx, cy := 14.0, 11.5 // the shell's centre
	pinch := pose == crabPinch

	// Claws first: an arm reaching forward and out, then a big pincer.
	clawPal := map[byte]color.RGBA{'H': crabRamp[5], 'h': crabRamp[4], 'M': crabRamp[4], 'm': crabRamp[3], 'd': crabRamp[1]}
	for side := -1.0; side <= 1; side += 2 {
		px, py := cx+side*8, cy-4.6
		rows := crabClaws[0]
		if pinch {
			px, py = cx+side*8.6, cy-7.4 // raised high and wide
			rows = crabClaws[1]
		}
		for k := 0.0; k <= 1; k++ {
			c.line(cx+side*4, cy-2+k, px-side*0.5, py+2+k, func(x, y int, t float64) color.RGBA {
				return crabRamp.pick(0.5-k*0.2+t*0.15, x, y)
			})
		}
		cl := newCanvas(len(rows[0]), len(rows))
		stamp(cl, 0, 0, rows, clawPal)
		if side < 0 {
			cl = cl.mirrored()
		}
		c.over(cl, int(math.Round(px))-3, int(math.Round(py))-len(rows)+2)
	}

	// The shell: wide and flat, widest towards the front, with a pale
	// toothed front rim, a bright patch where the sun hits it, darker
	// freckles and the belly's pale edge below.
	c.ellipse(cx, cy, 6.2, 3.9, func(x, y int, nx, ny float64) color.RGBA {
		if math.Abs(nx) > 1-0.18*(ny+1)*(ny+1)/4-0.02 && ny > 0.1 {
			return color.RGBA{} // tapering in towards the back
		}
		l := lum(nx*0.9, ny*0.9-0.1) + 0.08
		switch {
		case ny > 0.62:
			return crabBelly.pick(l+0.25, x, y)
		case ny < -0.72 && (x+int(cx))%2 == 0:
			return crabRamp.pick(l+0.35, x, y) // teeth along the front rim
		case math.Abs(math.Abs(nx)-0.34) < 0.09 && ny > -0.35 && ny < 0.25:
			l -= 0.22 // the grooves either side of the middle
		case hashf(x, y, 611) < 0.12:
			l -= 0.16 // freckles
		}
		return crabRamp.pick(l, x, y)
	})
	// Eyes on stalks at the front edge, looking a little inwards.
	for side := -1.0; side <= 1; side += 2 {
		ex := int(math.Floor(cx + side*3.5))
		c.set(ex, int(cy-3.6), crabRamp[3])
		c.set(ex, int(cy-4.6), crabRamp[4])
		c.set(ex, int(cy-5.6), hex(0xffffff))
		c.set(ex-int(side), int(cy-5.6), colInk)
	}
	c.set(int(cx)-4, int(cy)-2, crabRamp[5]) // where the sun catches the shell
	c.set(int(cx)-3, int(cy)-3, crabRamp[5])
	c.set(int(cx)-2, int(cy)-3, crabRamp[5])
	c.outline(0.85)

	// Legs last, thin, and only on bare canvas so the shell stays in front:
	// four a side, a knee lifted high, the foot on the sand. The walk frames
	// swap which legs are raised.
	for side := -1.0; side <= 1; side += 2 {
		for i := range 4 {
			lift := 0.0
			if !pinch && (i+pose)%2 == int(side+1)/2 {
				lift = 1
			}
			bx, by := cx+side*(5.2-0.5*float64(i)), cy-0.4+1.1*float64(i)
			kx, ky := bx+side*(3.4+0.2*float64(i)), by-1.4-lift
			fx, fy := kx+side*(1.6-0.3*float64(i)), by+2.4-lift
			paint := func(lo, hi float64) func(x, y int, t float64) color.RGBA {
				return func(x, y int, t float64) color.RGBA {
					if c.at(x, y).A > 0 {
						return color.RGBA{}
					}
					return crabLeg.pick(lerpf(lo, hi, t), x, y)
				}
			}
			c.line(bx, by, kx, ky, paint(0.5, 1))
			c.line(kx, ky, fx, fy, paint(0.8, 0))
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
