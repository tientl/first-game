package client

import (
	"image/color"
	"math"
)

// The plesiosaur lurking under the water: no longer a few blocky shapes
// drawn at the art's own coarse pixels, but its own outline, the body, the
// long neck held out ahead, the head and the four paddles, traced three
// times as fine with soft edges, fainter and bluer the deeper each part
// lies, and wavering with the ripples above it (a few frames, cycled).

// plesioUnderFrames is how many ripple frames the shape is drawn in.
const plesioUnderFrames = 6

var plesioUnderSet lazy[[2][plesioUnderFrames]sprite]

func init() {
	plesioUnderSet.gen = func() (s [2][plesioUnderFrames]sprite) {
		for f := range plesioUnderFrames {
			c, ax, ay := plesioUnderCanvas(2 * math.Pi * float64(f) / plesioUnderFrames)
			s[0][f], s[1][f] = sidedPair(mk(c, ax, ay))
		}
		return s
	}
}

// plesioUnderWater is the lurking plesiosaur's shape for frame t, facing left or
// right: its ripples move on every ten frames.
func plesioUnderWater(left bool, t int) sprite {
	side := 0
	if left {
		side = 1
	}
	return plesioUnderSet.get()[side][(t/10)%plesioUnderFrames]
}

// plesioUnderCanvas draws the shape at ripple phase ph, in the pixels of the
// plesiosaur's drawing (dinos/plesio.png, anchored at (23, 45) as it is),
// HDDensity times as fine. It returns the canvas and its anchor in its own
// pixels.
func plesioUnderCanvas(ph float64) (*canvas, float64, float64) {
	const k = HDDensity
	const gx0, gy0, gw, gh = -4.0, 31.0, 72.0, 24.0 // the part of the drawing's frame it fills
	c := newCanvas(int(gw*k), int(gh*k))
	c.dens = k
	type cap struct{ ax, ay, ra, bx, by, rb, far float64 }
	// The body and tail, the neck and head, then the paddles, the far ones
	// fainter.
	parts := []cap{
		{9, 43.4, 4.4, 30, 43, 4.6, 0},  // the body
		{-1, 44, 0.5, 10, 43.6, 3.6, 0}, // the tail
		{30, 42.6, 3.2, 45, 39.6, 2.1, 0},
		{45, 39.6, 2.1, 53, 38.4, 1.9, 0}, // the neck, held out ahead
		{52.5, 38.4, 2.4, 57.5, 38.8, 1.4, 0},
		{28.5, 45.6, 1.7, 21, 50.4, 1.0, 0},  // the near fore paddle, swept back
		{13.5, 45.4, 1.4, 7.5, 49.6, 0.8, 0}, // the near hind one
		{29, 41.2, 1.5, 23, 37.4, 0.8, 1},    // the far ones, above, fainter
		{14, 41.4, 1.2, 9, 38.4, 0.7, 1},
	}
	var cones []func(x, y float64) float64
	for _, p := range parts {
		cones = append(cones, coneSD(p.ax, p.ay, p.ra, p.bx, p.by, p.rb))
	}
	deep := [3]float64{0x0c, 0x2a, 0x3e} // its colour, dark teal going to deep blue
	deeper := [3]float64{0x10, 0x26, 0x4c}
	for py := 0; py < c.h; py++ {
		for px := 0; px < c.w; px++ {
			y := gy0 + (float64(py)+0.5)/k
			// The ripples bend the shape a little, row by row.
			x := gx0 + (float64(px)+0.5)/k + 0.5*math.Sin(y*1.25+ph) + 0.25*math.Sin(y*0.55-ph*0.7)
			d, far := math.Inf(1), 0.0
			for i, f := range cones {
				if v := f(x, y); v < d {
					d, far = v, parts[i].far
				}
			}
			a := clamp01((0.7 - d) / 1.4) // a soft edge, a pixel and a half wide
			if a <= 0 {
				continue
			}
			depth := clamp01((y - 37) / 14)       // 0 near the surface, 1 deep
			a *= (1 - 0.5*depth) * (1 - 0.45*far) // the deeper, the fainter
			a = math.Round(a*7) / 7               // in a few clean steps
			if a <= 0 {
				continue
			}
			col := color.RGBA{A: uint8(220 * a)}
			for ch, v := range [3]*uint8{&col.R, &col.G, &col.B} {
				*v = uint8(deep[ch] + (deeper[ch]-deep[ch])*depth)
			}
			c.px[py*c.w+px] = col
		}
	}
	return c, (23 - gx0) * k, (45 - gy0) * k
}

// coneSD is the signed distance to a limb round at both ends, ra at (ax,
// ay) and rb at (bx, by), in the units its points are given in.
func coneSD(ax, ay, ra, bx, by, rb float64) func(x, y float64) float64 {
	return func(x, y float64) float64 {
		dx, dy := bx-ax, by-ay
		l2 := dx*dx + dy*dy
		t := clamp01(((x-ax)*dx + (y-ay)*dy) / l2)
		return math.Hypot(x-ax-t*dx, y-ay-t*dy) - (ra + (rb-ra)*t)
	}
}
