package client

import (
	"image/color"
	"math"
)

// An under-layer, dinos/hd/<file>_under.png, is a painted drawing's body
// where the parts that turn with the head hide it (painted by
// cmd/dinopaint; see internal/dinopaint/under.go): a three-horn's shoulders
// and back under its frill. With one, the head turns cleanly: only the
// head's own pixels turn, the body's stay put even on the head's side of
// the cut, and where the head was, the body painted under it shows,
// instead of the head swept round to fill the joint.
//
// Its alpha says what the drawing is at each pixel: 255, the head over
// body (the colour is that body); 1 to 127, the body's own; 0, the head
// with nothing under it, or nothing.

// dinoUnder is the under-layer of drawing file at density k, or nil.
func dinoUnder(file string, k int) *canvas {
	if k != HDDensity {
		return nil
	}
	return dinoImage(file+"_under", true)
}

// turnOver turns the head, rigPart pr, by a radians about its pivot over
// under-layer u, whose top left is at (ox, oy) on c.
func turnOver(c *canvas, pr rigPart, a float64, u *canvas, ox, oy int) {
	src := append([]color.RGBA(nil), c.px...)
	has := pr.tester()
	ul := func(x, y int) color.RGBA {
		x, y = x-ox, y-oy
		if x < 0 || y < 0 || x >= u.w || y >= u.h {
			return color.RGBA{}
		}
		return u.px[y*u.w+x]
	}
	in := func(x, y int) bool {
		if src[y*c.w+x].A < 128 || !has(float64(x)+0.5, float64(y)+0.5) {
			return false
		}
		q := ul(x, y)
		return q.A == 0 || q.A >= 128 // not the body's own
	}
	reach := 0.0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if !in(x, y) {
				continue
			}
			reach = max(reach, math.Hypot(float64(x)+0.5-pr.pivot.x, float64(y)+0.5-pr.pivot.y))
			c.px[y*c.w+x] = color.RGBA{}
			if q := ul(x, y); q.A >= 128 {
				c.px[y*c.w+x] = q // the body under the head
			}
		}
	}
	x0, x1 := max(0, int(pr.pivot.x-reach)-1), min(c.w, int(pr.pivot.x+reach)+2)
	y0, y1 := max(0, int(pr.pivot.y-reach)-1), min(c.h, int(pr.pivot.y+reach)+2)
	sn, cs := math.Sincos(-a)
	r2 := (reach + 1) * (reach + 1)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dx, dy := float64(x)+0.5-pr.pivot.x, float64(y)+0.5-pr.pivot.y
			if dx*dx+dy*dy > r2 {
				continue
			}
			sx := int(math.Floor(pr.pivot.x + dx*cs - dy*sn))
			sy := int(math.Floor(pr.pivot.y + dx*sn + dy*cs))
			if sx >= 0 && sy >= 0 && sx < c.w && sy < c.h && in(sx, sy) {
				c.px[y*c.w+x] = src[sy*c.w+sx]
			}
		}
	}
}
