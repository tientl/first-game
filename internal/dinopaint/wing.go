package dinopaint

import (
	"image/color"
	"math"
)

// A folded wing, as a bird's lies against its flank: the long flight
// feathers (remiges) laid back from the forearm and the hand one over
// another, the shortest at the top, the longest at the bottom reaching
// furthest back, each lower one's tip stepping out past the one above so
// the wing ends in a row of separate points; a row of short coverts over
// their roots. Each remex is long and narrow: its trailing vane, under the
// feather above, dark; its shaft a fine pale line; its leading vane lit,
// its very edge catching the light; its tip dark. Each casts a thin shadow
// on the one below. Painted last, over the lit hide, as a detail.

// FoldedWing is how a folded wing is painted. Positions are in guide
// pixels.
type FoldedWing struct {
	// N remiges, from the top one to the bottom one: their roots along the
	// forearm and the hand from Root0 to Root1, their tips from Tip0 to
	// Tip1, each Width across at its broadest.
	N            int
	Root0, Root1 [2]float64
	Tip0, Tip1   [2]float64
	Width        float64
	// CN coverts over the remiges' roots, their roots from Covert0 to
	// Covert1, each CLen long and CWid across, lying back as the remiges do.
	CN               int
	Covert0, Covert1 [2]float64
	CLen, CWid       float64
	// The colours, each from deep shadow to the highlight: Root at the
	// feathers' roots and on the coverts, Vane down them (the lower remiges
	// more of it), Tip on their last TipLen of their length.
	Root, Vane, Tip []uint32
	TipLen          float64
	Seed            uint32
}

// wfeather is one feather of a folded wing.
type wfeather struct {
	rx, ry, tx, ty float64 // root and tip
	ux, uy, nx, ny float64 // along it, and across it, downwards
	l, hw          float64 // its length, its half-breadth
	covert         bool
	m              float64 // how far down the wing (0 top, 1 bottom)
	tint           float64
}

func newWFeather(rx, ry, tx, ty, w float64, covert bool, m, tint float64) wfeather {
	dx, dy := tx-rx, ty-ry
	l := math.Hypot(dx, dy)
	f := wfeather{rx: rx, ry: ry, tx: tx, ty: ty, ux: dx / l, uy: dy / l, l: l, hw: w / 2, covert: covert, m: m, tint: tint}
	f.nx, f.ny = -f.uy, f.ux
	if f.ny < 0 {
		f.nx, f.ny = -f.nx, -f.ny
	}
	return f
}

// at is where guide point (x, y) is on the feather: u along it (0 root, 1
// tip), d across it (+ down), and its half-breadth there (< 0: off its end).
func (f *wfeather) at(x, y float64) (u, d, h float64) {
	px, py := x-f.rx, y-f.ry
	u = (px*f.ux + py*f.uy) / f.l
	d = px*f.nx + py*f.ny
	return u, d, f.hw * f.half(u)
}

// half is the feather's half-breadth u along it, as a share of its widest:
// narrow at the root, a remex's tip pointed, a covert's round.
func (f *wfeather) half(u float64) float64 {
	switch {
	case u < 0 || u > 1:
		return -1
	case u < 0.18:
		return 0.55 + 0.45*u/0.18
	case f.covert && u > 0.55:
		k := (u - 0.55) / 0.45
		return math.Sqrt(max(0, 1-k*k))
	case !f.covert && u > 0.72:
		k := (u - 0.72) / 0.28
		return math.Pow(max(0, 1-k), 1.15)
	}
	return 1
}

// FoldedWing paints a folded wing over the painting.
func (p *Painter) FoldedWing(w FoldedWing) {
	lerp2 := func(a, b [2]float64, t float64) (float64, float64) {
		return a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t
	}
	// In order from the top of the stack: the coverts, then the remiges
	// from the top one down.
	var fs []wfeather
	for i := range w.CN {
		t := float64(i) / float64(max(1, w.CN-1))
		rx, ry := lerp2(w.Covert0, w.Covert1, t)
		// a covert lies back as the remex under it does
		rt := t * 0.75
		r0x, r0y := lerp2(w.Root0, w.Root1, rt)
		t0x, t0y := lerp2(w.Tip0, w.Tip1, rt)
		dx, dy := t0x-r0x, t0y-r0y
		l := math.Hypot(dx, dy)
		fs = append(fs, newWFeather(rx, ry, rx+dx/l*w.CLen, ry+dy/l*w.CLen, w.CWid, true, t, (hash(i, 7, w.Seed)-0.5)*0.06))
	}
	for i := range w.N {
		t := float64(i) / float64(max(1, w.N-1))
		rx, ry := lerp2(w.Root0, w.Root1, t)
		tx, ty := lerp2(w.Tip0, w.Tip1, t)
		fs = append(fs, newWFeather(rx, ry, tx, ty, w.Width, false, t, (hash(i, 3, w.Seed)-0.5)*0.07))
	}
	top := func(x, y float64) int {
		for k := range fs {
			if u, d, h := fs[k].at(x, y); h > 0 && math.Abs(d) <= h && u >= 0 {
				return k
			}
		}
		return -1
	}
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, f := range fs {
		x0, x1 = min(x0, f.rx, f.tx)-f.hw, max(x1, f.rx, f.tx)+f.hw
		y0, y1 = min(y0, f.ry, f.ty)-f.hw, max(y1, f.ry, f.ty)+f.hw
	}
	pick := func(ramp []uint32, v float64) color.NRGBA {
		return rgb(ramp[max(0, min(len(ramp)-1, int(v*float64(len(ramp)))))])
	}
	own := map[int]int{}
	for py := int(y0 * K); py <= int(y1*K)+1; py++ {
		for px := int(x0 * K); px <= int(x1*K)+1; px++ {
			i := p.Idx(px, py)
			if i < 0 || !p.In[i] {
				continue
			}
			x, y := G(px, py)
			if k := top(x, y); k >= 0 {
				own[i] = k
			}
		}
	}
	for i, k := range own {
		px, py := i%p.W, i/p.W
		x, y := G(px, py)
		f := &fs[k]
		u, d, h := f.at(x, y)
		s := d / h // -1 its top edge, +1 its bottom
		var v float64
		ramp := func(v float64) color.NRGBA {
			t := clamp01(-0.2 + 0.95*u + 0.4*f.m)
			if f.covert {
				t = clamp01(0.15*u + 0.2*f.m)
			}
			return Mix(pick(w.Root, v), pick(w.Vane, v), t)
		}
		if f.covert {
			v = 0.62 - 0.2*s // rounded, lit from above
		} else {
			// The trailing vane, under the feather above, darker; the leading
			// vane lit, more so to its edge.
			v = 0.4 + 0.26*(s+1)/2
			// the shaft, a fine pale line
			if math.Abs(d-0.3*h)*K < 0.55 && u > 0.04 && u < 0.86 {
				v += 0.14
			}
			// the leading edge catching the light
			if (h-d)*K < 1 {
				v = 0.74
			}
			// the fine barbs
			v += 0.03 * (hash(int(u*f.l*K*0.8-d*K*1.1), k, w.Seed+5) - 0.5)
		}
		// the trailing edge, dark where it shows
		if (h+d)*K < 1 {
			v -= 0.15
		}
		// in the shadow of the feather above
		if j := top(x-0.2, y-0.42); j >= 0 && j < k {
			v -= 0.32
		}
		v += f.tint + 0.06*(1-2*clamp01((y-y0)/(y1-y0)))
		c := ramp(clamp01(v))
		tipFrom := 1 - w.TipLen + 0.05*(hash(px, py, w.Seed+9)-0.5)
		if !f.covert && u > tipFrom && (h-d)*K >= 1 {
			c = pick(w.Tip, clamp01(v))
		} else if f.covert && u > 0.86 {
			c = Mix(c, Ink, 0.35) // the covert's edge over the remiges
		}
		// the wing's own outline against the flank
		edge := false
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if j := p.Idx(px+n[0], py+n[1]); j >= 0 && p.In[j] {
				if _, ok := own[j]; !ok {
					edge = true
				}
			}
		}
		if edge {
			c = Mix(c, Ink, 0.55)
		}
		p.Out[i], p.Pro[i] = c, true
	}
	// Its shadow on the flank below it.
	for py := int(y0 * K); py <= int(y1*K)+3; py++ {
		for px := int(x0 * K); px <= int(x1*K)+1; px++ {
			i := p.Idx(px, py)
			if i < 0 || !p.In[i] || p.Pro[i] {
				continue
			}
			if _, ok := own[i]; ok {
				continue
			}
			for dy := 1; dy <= 2; dy++ {
				if j := p.Idx(px+dy/2, py-dy); j >= 0 {
					if _, ok := own[j]; ok {
						p.Out[i] = Mix(p.Out[i], Ink, 0.4/float64(dy))
						break
					}
				}
			}
		}
	}
}
