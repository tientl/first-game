package client

import (
	"image/color"
	"math"
)

// The Ankylosaurus swings its tail club sideways. Seen from the side, a
// tail turning about an upright axis shortens towards its root, and comes
// down the picture as it turns towards the viewer (the ground is seen from
// above: a step towards the viewer is drawn groundDepth of a step down), or
// goes up it, behind the body, turning away. The tail bends as it swings:
// its root turns least, the stiff handle more, and the club at its end,
// rigid, turns with the end of the handle.

// groundDepth is how far down the picture a step towards the viewer is
// drawn, for a step across it.
const groundDepth = float64(TileH) / float64(TileW)

// ankyloClubLen is how long the club at the end of the tail is, in pixels
// of the drawing: it turns as one piece.
const ankyloClubLen = 14.0

// sweepBend is how much of the swing the root of the tail takes; the rest
// comes in along the handle, up to the club.
const sweepBend = 0.35

// sweep swings rig part pr (a tail pointing back, to the left of its pivot)
// by a radians about an upright axis through its pivot on working canvas c
// (kf canvas pixels to a pixel of the drawing): + towards the viewer, over
// the body and the legs; - away, behind them.
func sweep(c *canvas, pr rigPart, a, kf float64) {
	has := pr.tester()
	src := append([]color.RGBA(nil), c.px...)
	isTail := make([]bool, len(c.px))
	x0, x1 := c.w, -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			i := y*c.w + x
			if src[i].A >= 128 && has(float64(x)+0.5, float64(y)+0.5) {
				isTail[i] = true
				c.px[i] = color.RGBA{}
				x0, x1 = min(x0, x), max(x1, x)
			}
		}
	}
	if x1 < 0 {
		return
	}
	// The tail's line through the swing, a quarter pixel at a time: how far
	// back from the pivot and how far towards the viewer each point of it
	// is, by how far along the tail it was.
	reach := pr.pivot.x - float64(x0) + 1
	club := max(reach*0.3, reach-ankyloClubLen*kf)
	const step = 0.25
	n := int(reach/step) + 2
	back, toward := make([]float64, n+1), make([]float64, n+1)
	for i := 1; i <= n; i++ {
		t := min(1, (float64(i)-0.5)*step/club)
		th := a * (sweepBend + (1-sweepBend)*t*t*(3-2*t))
		back[i] = back[i-1] + math.Cos(th)*step
		toward[i] = toward[i-1] + math.Sin(th)*step
	}
	// From the root out: turning towards the viewer, what is further along
	// is nearer them and drawn over; turning away, it is further and only
	// fills what is empty. Each pixel is spread from nine points in it, so
	// the tail drawn shorter or sheared has no holes.
	for x := x1; x >= x0; x-- {
		for y := 0; y < c.h; y++ {
			if !isTail[y*c.w+x] {
				continue
			}
			col := src[y*c.w+x]
			for sy := 0; sy < 3; sy++ {
				for sx := 0; sx < 3; sx++ {
					fx, fy := float64(x)+(float64(sx)+0.5)/3, float64(y)+(float64(sy)+0.5)/3
					i := min(n, max(0, int((pr.pivot.x-fx)/step)))
					dx := int(math.Floor(pr.pivot.x - back[i]))
					dy := int(math.Floor(fy + toward[i]*groundDepth))
					if dx < 0 || dy < 0 || dx >= c.w || dy >= c.h {
						continue
					}
					j := dy*c.w + dx
					if a < 0 && c.px[j].A >= 128 {
						continue
					}
					c.px[j] = col
				}
			}
		}
	}
}
