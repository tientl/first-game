package client

import (
	"image/color"
	"math"
)

// What the bosses' drawings share: a painter of rounded bodies that knows
// where along and across a limb each pixel lies (so that stripes, bellies and
// sails follow the body through every pose), a turned frame for heads and
// jaws, and the crop that fits a finished pose to its canvas.

// bossPt is what the painter knows about one pixel of a body.
type bossPt struct {
	l, ny float64 // the light on it, and which way it faces (-1 up, 1 down)
	t     float64 // 0..1 along its chain
	s     float64 // the same, in pixels from the chain's first knob
	v     float64 // across the chain: -1 its left edge (the back of a body drawn tail to head), 1 its right
	chain int     // which of the chains it belongs to
}

// bossPelt colours one pixel of a painted part; a clear colour leaves the
// pixel as it was.
type bossPelt func(x, y int, q bossPt) color.RGBA

// bossPaint is furPaint (see iceart.go) for scaly things: the chains of
// knobs swell into one surface, lit from the upper left, and f colours it.
// Where the part lies over what is already drawn it casts a seam.
func bossPaint(c *canvas, seam float64, f bossPelt, chains ...[]knob) *furShape {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, ch := range chains {
		for _, k := range ch {
			x0, y0 = min(x0, k[0]-k[2]), min(y0, k[1]-k[2])
			x1, y1 = max(x1, k[0]+k[2]), max(y1, k[1]+k[2])
		}
	}
	s := &furShape{x0: max(0, int(math.Floor(x0))-1), y0: max(0, int(math.Floor(y0))-1)}
	s.w, s.h = min(c.w, int(math.Ceil(x1))+2)-s.x0, min(c.h, int(math.Ceil(y1))+2)-s.y0
	if s.w <= 0 || s.h <= 0 {
		s.w, s.h = 0, 0
		return s
	}
	type seg struct {
		ax, ay, ar, dx, dy, dr float64
		len, s0, total         float64
		chain                  int
	}
	var segs []seg
	for ci, ch := range chains {
		total := 0.0
		for q := 0; q+1 < len(ch); q++ {
			total += math.Hypot(ch[q+1][0]-ch[q][0], ch[q+1][1]-ch[q][1])
		}
		if len(ch) == 1 {
			segs = append(segs, seg{ax: ch[0][0], ay: ch[0][1], ar: ch[0][2], total: 1, chain: ci})
		}
		acc := 0.0
		for q := 0; q+1 < len(ch); q++ {
			a, b := ch[q], ch[q+1]
			l := math.Hypot(b[0]-a[0], b[1]-a[1])
			segs = append(segs, seg{a[0], a[1], a[2], b[0] - a[0], b[1] - a[1], b[2] - a[2], l, acc, max(total, 1e-9), ci})
			acc += l
		}
	}
	s.in = make([]bool, s.w*s.h)
	hgt := make([]float64, s.w*s.h)
	info := make([]bossPt, s.w*s.h)
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			px, py := float64(s.x0+i)+0.5, float64(s.y0+j)+0.5
			best := 0.0
			var bi bossPt
			for k := range segs {
				g := &segs[k]
				u := 0.0
				if g.len > 0 {
					u = clamp01(((px-g.ax)*g.dx + (py-g.ay)*g.dy) / (g.len * g.len))
				}
				r := g.ar + g.dr*u
				ex, ey := px-(g.ax+g.dx*u), py-(g.ay+g.dy*u)
				d := math.Hypot(ex, ey)
				if d >= r {
					continue
				}
				if hh := math.Sqrt(r*r - d*d); hh > best {
					best = hh
					v := ey / r
					if g.len > 0 {
						v = (g.dx*ey - g.dy*ex) / (g.len * r)
					}
					bi = bossPt{t: (g.s0 + g.len*u) / g.total, s: g.s0 + g.len*u, v: v, chain: g.chain}
				}
			}
			if best > 0 {
				s.in[j*s.w+i] = true
				hgt[j*s.w+i], info[j*s.w+i] = best, bi
			}
		}
	}
	at := func(g []float64, i, j int) float64 {
		if i < 0 || j < 0 || i >= s.w || j >= s.h {
			return 0
		}
		return g[j*s.w+i]
	}
	sm := make([]float64, len(hgt))
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			sm[j*s.w+i] = (4*at(hgt, i, j) + 2*(at(hgt, i-1, j)+at(hgt, i+1, j)+at(hgt, i, j-1)+at(hgt, i, j+1)) +
				at(hgt, i-1, j-1) + at(hgt, i+1, j-1) + at(hgt, i-1, j+1) + at(hgt, i+1, j+1)) / 16
		}
	}
	if seam > 0 {
		done := map[[2]int]bool{}
		for j := 0; j < s.h; j++ {
			for i := 0; i < s.w; i++ {
				if !s.in[j*s.w+i] {
					continue
				}
				for _, d := range [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}} {
					x, y := s.x0+i+d[0], s.y0+j+d[1]
					if s.has(x, y) || done[[2]int{x, y}] || c.at(x, y).A < 128 {
						continue
					}
					k := seam
					if d[0] < 0 || d[1] < 0 {
						k *= 0.6 // the lit side
					}
					done[[2]int{x, y}] = true
					c.put(x, y, shadeRGB(c.at(x, y), 1-k))
				}
			}
		}
	}
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			if !s.in[j*s.w+i] {
				continue
			}
			gx := (at(sm, i+1, j) - at(sm, i-1, j)) / 2
			gy := (at(sm, i, j+1) - at(sm, i, j-1)) / 2
			n := math.Sqrt(gx*gx + gy*gy + 1)
			nx, ny, nz := -gx/n, -gy/n, 1/n
			q := info[j*s.w+i]
			q.l, q.ny = clamp01(0.2+0.8*(nx*lightDir[0]+ny*lightDir[1]+nz*lightDir[2])), ny
			if col := f(s.x0+i, s.y0+j, q); col.A > 0 {
				c.put(s.x0+i, s.y0+j, col)
			} else {
				s.in[j*s.w+i] = false
			}
		}
	}
	return s
}

// bossFrame is a turned frame: u runs along it from its origin, v across it
// (down, for a head looking right).
type bossFrame struct{ ox, oy, a float64 }

func (f bossFrame) at(u, v float64) (float64, float64) {
	x, y := rotPt(f.a, u, v)
	return f.ox + x, f.oy + y
}

// local is the frame's (u, v) of the middle of pixel (x, y).
func (f bossFrame) local(x, y int) (float64, float64) {
	return rotPt(-f.a, float64(x)+0.5-f.ox, float64(y)+0.5-f.oy)
}

func (f bossFrame) k(u, v, r float64) knob {
	x, y := f.at(u, v)
	return knob{x, y, r}
}

func (f bossFrame) knobs(ks []knob) []knob {
	out := make([]knob, len(ks))
	for i, k := range ks {
		out[i] = f.k(k[0], k[1], k[2])
	}
	return out
}

// poly fills the polygon q, given in the frame.
func (f bossFrame) poly(c *canvas, q [][2]float64, col func(x, y int) color.RGBA) {
	pts := make([]float64, 0, 2*len(q))
	for _, v := range q {
		x, y := f.at(v[0], v[1])
		pts = append(pts, x, y)
	}
	c.polygon(pts, col)
}

// bossLine draws a line a pixel wide in one colour.
func bossLine(c *canvas, x0, y0, x1, y1 float64, col color.RGBA) {
	c.line(x0, y0, x1, y1, func(int, int, float64) color.RGBA { return col })
}

// bossScales is the grain of a scaly hide: how much lighter or darker pixel
// (x, y) is than its neighbours.
func bossScales(x, y int, seed uint32) float64 {
	d := (hashf(x, y, seed) - 0.5) * 0.07
	if (x+(y/2)%2)%3 == 0 && y%2 == 0 {
		d -= 0.055 // the seams between staggered scales
	}
	return d
}

// bossCrop cuts a finished drawing down to what is on it, with a pixel to
// spare, and moves the anchor to match.
func bossCrop(c *canvas, ax, ay float64) (*canvas, float64, float64) {
	x0, y0, x1, y1 := c.w, c.h, 0, 0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A > 0 {
				x0, y0 = min(x0, x), min(y0, y)
				x1, y1 = max(x1, x+1), max(y1, y+1)
			}
		}
	}
	if x1 <= x0 {
		return newCanvas(1, 1), 0, 0
	}
	x0, y0 = max(0, x0-1), max(0, y0-1)
	x1, y1 = min(c.w, x1+1), min(c.h, y1+1)
	out := newCanvas(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		copy(out.px[(y-y0)*out.w:(y-y0+1)*out.w], c.px[y*c.w+x0:y*c.w+x1])
	}
	return out, ax - float64(x0), ay - float64(y0)
}

// bossClearBelow clears everything from row y down.
func bossClearBelow(c *canvas, y int) {
	for ; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.put(x, y, color.RGBA{})
		}
	}
}
