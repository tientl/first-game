package client

import "image/color"

// A leg mask, dinos/hd/<file>_legs.png, says which pixels of a painted
// drawing are which leg (painted by cmd/dinopaint; see
// internal/dinopaint/legmask.go): its red is the rig's number of the leg,
// from 1, and 0 the body; its green, where a near leg hides a far one, the
// far leg's number, whose colours there are in dinos/hd/<file>_legsunder.png.
// With one, a leg is cut along its painted outline rather than in columns,
// so it takes none of the belly or the next leg's foot with it as it steps
// and leaves none of its own outline behind; and a near leg stepping away
// shows the far leg whole behind it.

// legOwner is the pixels of one leg in a working canvas: its mask, the
// number it has in it, and the columns it reaches across.
type legOwner struct {
	mask, under *canvas
	n           uint8
	pad         int
	x0, x1      int // in canvas pixels
}

// legOwnerOf is leg i of drawing file at density k, in a canvas padded by
// pad, or nil when the drawing has no leg mask.
func legOwnerOf(file string, i, pad, k int) *legOwner {
	if k != HDDensity {
		return nil
	}
	m, u := dinoImage(file+"_legs", true), dinoImage(file+"_legsunder", true)
	if m == nil || u == nil || u.w != m.w || u.h != m.h {
		return nil
	}
	o := &legOwner{mask: m, under: u, n: uint8(i + 1), pad: pad, x0: m.w, x1: -1}
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if q := m.px[y*m.w+x]; q.A >= 128 && (q.R == o.n || q.G == o.n) {
				o.x0, o.x1 = min(o.x0, x), max(o.x1, x)
			}
		}
	}
	if o.x1 < 0 {
		return nil
	}
	o.x0, o.x1 = o.x0+pad, o.x1+pad
	return o
}

// pick is what the leg has at canvas pixel (x, y), where the canvas has q:
// q itself if the pixel is the leg's, its own colour painted again if a
// near leg hides it there or it lies along one, and mine false if it is not the leg's at all. With no mask,
// every pixel is the leg's.
func (o *legOwner) pick(q color.RGBA, x, y int) (color.RGBA, bool) {
	if o == nil {
		return q, true
	}
	x, y = x-o.pad, y-o.pad
	if x < 0 || y < 0 || x >= o.mask.w || y >= o.mask.h {
		return q, false
	}
	switch m := o.mask.px[y*o.mask.w+x]; {
	case m.A < 128:
		return q, false
	case m.G == o.n:
		return o.under.px[y*o.under.w+x], true // painted again: hidden, or along a near leg
	case m.R == o.n:
		return q, true
	}
	return q, false
}
