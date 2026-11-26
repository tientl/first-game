package client

import (
	"image/color"
	"math"
)

// Legs that swing whole. A drawing painted with dinos/hd/<file>_legsbody.png
// (by cmd/dinopaint; see internal/dinopaint/thigh.go) has its legs cut
// whole from the hip down, thigh and all, and turned about the hip as a
// layer of their own: the near legs over the body but under what its mask
// marks in front of them (the folded wing, the flank's feathers over the
// hips), the far legs behind the body, painted whole where it hides them
// (their thighs up into the hip) so they never come away from it. Where
// a leg was, the body painted under it shows: the flank behind a thigh
// swung forward is the flank, not a hole.
//
// The mask's blue (dinolegmask.go has its red and green): on the body,
// legFront where it is in front of the near legs; on a leg, legThigh where
// the pixel turns with the thigh, whatever bone it is nearest.
const (
	legFront = 1
	legThigh = 2
)

// whole reports whether the leg swings whole.
func (o *legOwner) whole() bool { return o != nil && o.body != nil }

// at is the mask at canvas pixel (x, y), the body sunk dy, or nothing.
func (o *legOwner) at(x, y, dy int) color.RGBA {
	x, y = x-o.pad, y-o.pad-dy
	if x < 0 || y < 0 || x >= o.mask.w || y >= o.mask.h {
		return color.RGBA{}
	}
	return o.mask.px[y*o.mask.w+x]
}

// owns reports whether canvas pixel (x, y) is the leg's own, shown in the
// drawing, not hidden behind something.
func (o *legOwner) owns(x, y int) bool {
	q := o.at(x, y, 0)
	return q.A >= 128 && q.R == o.n
}

// thigh reports whether canvas pixel (x, y) turns with the leg's thigh.
func (o *legOwner) thigh(x, y int) bool {
	q := o.at(x, y, 0)
	return q.A >= 128 && q.R == o.n && q.B == legThigh
}

// front reports whether canvas pixel (x, y), the body sunk dy, is the
// body's in front of the near legs.
func (o *legOwner) front(x, y, dy int) bool {
	q := o.at(x, y, dy)
	return q.A >= 128 && q.R == 0 && q.B == legFront
}

// bodyAt is what is left at canvas pixel (x, y) with the legs lifted off:
// the body painted under them, or nothing.
func (o *legOwner) bodyAt(x, y int) color.RGBA {
	x, y = x-o.pad, y-o.pad
	if x < 0 || y < 0 || x >= o.body.w || y >= o.body.h {
		return color.RGBA{}
	}
	if q := o.body.px[y*o.body.w+x]; q.A >= 128 {
		return q
	}
	return color.RGBA{}
}

// legHipBend is how much of its thigh's turn the flesh at the hip takes:
// the thigh bends from the body, its top staying on the hip, the drumstick
// below swinging the whole way by the knee, rather than turning stiff with
// its corners swung out past the belly.
const legHipBend = 0.12

// bendAt is how much of the thigh's turn a point t of the way from the hip
// to the knee (along the thigh) takes.
func bendAt(t float64) float64 { return legHipBend + (1-legHipBend)*ease(t/0.75) }

// bendBack is where in the drawing the thigh's pixel (dx, dy) from the
// turned hip comes from, the thigh turned th: (fx, fy) where it would, the
// thigh stiff. Each point turns about the hip by its own share of th.
func (c *legCut) bendBack(dx, dy, th, fx, fy float64) (float64, float64) {
	hip, bone := c.P[0], c.P[1].sub(c.P[0])
	l2 := bone.x*bone.x + bone.y*bone.y
	for range 3 {
		t := ((fx-hip.x)*bone.x + (fy-hip.y)*bone.y) / l2
		sn, cs := math.Sincos(-th * bendAt(t))
		fx, fy = hip.x+dx*cs-dy*sn, hip.y+dx*sn+dy*cs
	}
	return fx, fy
}

// trimEdge clears, about whole leg c with the body sunk dy, the hairlines
// the leg leaves behind: pixels of the body along the leg's edge in the
// drawing (its outline, a tuft over it), left with one solid neighbour or
// none once the leg has swung away. leg marks what the legs drew, which
// stays.
func (c *legCut) trimEdge(w *canvas, leg []bool, dy, k int) {
	o := c.own
	x0, y0 := max(1, c.bx-2*k), max(1, c.by+dy)
	x1, y1 := min(w.w-1, c.bx+c.bw+2*k), min(w.h-1, c.by+c.bh+dy)
	solid := func(x, y int) bool { return w.px[y*w.w+x].A >= 128 }
	byLeg := func(x, y int) bool {
		for v := -k; v <= k; v++ {
			for u := -k; u <= k; u++ {
				if o.owns(x+u, y-dy+v) {
					return true
				}
			}
		}
		return false
	}
	for range 3 {
		var gone []int
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				if i := y*w.w + x; !solid(x, y) || leg[i] {
					continue
				}
				n := 0
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if solid(x+d[0], y+d[1]) {
						n++
					}
				}
				if n <= 1 && byLeg(x, y) {
					gone = append(gone, y*w.w+x)
				}
			}
		}
		if len(gone) == 0 {
			break
		}
		for _, i := range gone {
			w.px[i] = color.RGBA{}
		}
	}
}
