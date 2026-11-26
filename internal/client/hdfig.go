package client

import (
	"image/color"
	"math"
)

// The tools the HD figures are drawn with (lookhd.go): shapes given by a
// signed distance (negative inside), so they can be joined, cut, grown and
// lit from their own edge; layers that remember which part each pixel
// belongs to, so one part gets a dark line where another lies over it; and
// shaders for skin, cloth, hair and fur. Everything here is in image
// pixels of an HD canvas: HDDensity of them to an art pixel.

// shape is a region: d is the distance to its edge, negative inside, and
// the box bounds it.
type shape struct {
	x0, y0, x1, y1 float64
	d              func(x, y float64) float64
}

const sFar = 1e6

func sEll(cx, cy, rx, ry float64) shape {
	m := math.Min(rx, ry)
	return shape{cx - rx, cy - ry, cx + rx, cy + ry, func(x, y float64) float64 {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		return (math.Sqrt(dx*dx+dy*dy) - 1) * m
	}}
}

// sSuper is a superellipse: n = 2 is an ellipse, more is boxier.
func sSuper(cx, cy, rx, ry, n float64) shape {
	m := math.Min(rx, ry)
	return shape{cx - rx, cy - ry, cx + rx, cy + ry, func(x, y float64) float64 {
		dx, dy := math.Abs(x-cx)/rx, math.Abs(y-cy)/ry
		return (math.Pow(math.Pow(dx, n)+math.Pow(dy, n), 1/n) - 1) * m
	}}
}

// sBox is a box with corners of radius r.
func sBox(x0, y0, x1, y1, r float64) shape {
	cx, cy, hx, hy := (x0+x1)/2, (y0+y1)/2, (x1-x0)/2, (y1-y0)/2
	r = math.Min(r, math.Min(hx, hy))
	return shape{x0, y0, x1, y1, func(x, y float64) float64 {
		qx, qy := math.Abs(x-cx)-(hx-r), math.Abs(y-cy)-(hy-r)
		return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
	}}
}

// sCap is a line from a to b with a round brush of radius ra at a and rb at b.
func sCap(ax, ay, bx, by, ra, rb float64) shape {
	r := math.Max(ra, rb)
	ex, ey := bx-ax, by-ay
	ee := ex*ex + ey*ey
	return shape{math.Min(ax, bx) - r, math.Min(ay, by) - r, math.Max(ax, bx) + r, math.Max(ay, by) + r, func(x, y float64) float64 {
		t := 0.0
		if ee > 0 {
			t = clamp01(((x-ax)*ex + (y-ay)*ey) / ee)
		}
		return math.Hypot(x-ax-ex*t, y-ay-ey*t) - lerpf(ra, rb, t)
	}}
}

// sPath is a brush drawn through pts (x, y pairs), its radius going from
// w0 at the start to w1 at the end.
func sPath(w0, w1 float64, pts ...float64) shape {
	n := len(pts)/2 - 1
	s := sCap(pts[0], pts[1], pts[2], pts[3], w0, lerpf(w0, w1, 1/float64(n)))
	for i := 1; i < n; i++ {
		a, b := lerpf(w0, w1, float64(i)/float64(n)), lerpf(w0, w1, float64(i+1)/float64(n))
		s = s.or(sCap(pts[2*i], pts[2*i+1], pts[2*i+2], pts[2*i+3], a, b))
	}
	return s
}

// sLock is a lock of hair: a brush along the curve from (x0, y0) bent
// towards (cx, cy) to its tip (x1, y1), w0 thick at the root and w1 at the tip.
func sLock(x0, y0, cx, cy, x1, y1, w0, w1 float64) shape {
	const n = 6
	pts := make([]float64, 0, 2*(n+1))
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		a, b, c := (1-t)*(1-t), 2*t*(1-t), t*t
		pts = append(pts, a*x0+b*cx+c*x1, a*y0+b*cy+c*y1)
	}
	return sPath(w0, w1, pts...)
}

// sPoly is a polygon through pts (x, y pairs).
func sPoly(pts ...float64) shape {
	n := len(pts) / 2
	s := shape{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1), nil}
	for i := 0; i < n; i++ {
		s.x0, s.x1 = math.Min(s.x0, pts[2*i]), math.Max(s.x1, pts[2*i])
		s.y0, s.y1 = math.Min(s.y0, pts[2*i+1]), math.Max(s.y1, pts[2*i+1])
	}
	s.d = func(x, y float64) float64 {
		best, inside := math.Inf(1), false
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
			ex, ey, wx, wy := xj-xi, yj-yi, x-xi, y-yi
			t := 0.0
			if ee := ex*ex + ey*ey; ee > 0 {
				t = clamp01((wx*ex + wy*ey) / ee)
			}
			dx, dy := wx-ex*t, wy-ey*t
			best = math.Min(best, dx*dx+dy*dy)
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				inside = !inside
			}
		}
		if inside {
			return -math.Sqrt(best)
		}
		return math.Sqrt(best)
	}
	return s
}

// sFn is an unbounded region given by its distance alone.
func sFn(d func(x, y float64) float64) shape { return shape{-sFar, -sFar, sFar, sFar, d} }

func sAbove(y0 float64) shape { return sFn(func(_, y float64) float64 { return y - y0 }) }
func sBelow(y0 float64) shape { return sFn(func(_, y float64) float64 { return y0 - y }) }
func sLeftOf(x0 float64) shape {
	return sFn(func(x, _ float64) float64 { return x - x0 })
}
func sRightOf(x0 float64) shape {
	return sFn(func(x, _ float64) float64 { return x0 - x })
}

// sOver is everything above the curve y = f(x).
func sOver(f func(x float64) float64) shape {
	return sFn(func(x, y float64) float64 { return y - f(x) })
}

func (a shape) or(b shape) shape {
	return shape{math.Min(a.x0, b.x0), math.Min(a.y0, b.y0), math.Max(a.x1, b.x1), math.Max(a.y1, b.y1),
		func(x, y float64) float64 { return math.Min(a.d(x, y), b.d(x, y)) }}
}

func (a shape) and(b shape) shape {
	return shape{math.Max(a.x0, b.x0), math.Max(a.y0, b.y0), math.Min(a.x1, b.x1), math.Min(a.y1, b.y1),
		func(x, y float64) float64 { return math.Max(a.d(x, y), b.d(x, y)) }}
}

func (a shape) sub(b shape) shape {
	return shape{a.x0, a.y0, a.x1, a.y1, func(x, y float64) float64 { return math.Max(a.d(x, y), -b.d(x, y)) }}
}

func (a shape) grow(r float64) shape {
	return shape{a.x0 - r, a.y0 - r, a.x1 + r, a.y1 + r, func(x, y float64) float64 { return a.d(x, y) - r }}
}

// moved is a shifted by (dx, dy).
func (a shape) moved(dx, dy float64) shape {
	return shape{a.x0 + dx, a.y0 + dy, a.x1 + dx, a.y1 + dy, func(x, y float64) float64 { return a.d(x-dx, y-dy) }}
}

// flipped is a mirrored about the vertical line x = cx.
func (a shape) flipped(cx float64) shape {
	return shape{2*cx - a.x1, a.y0, 2*cx - a.x0, a.y1, func(x, y float64) float64 { return a.d(2*cx-x, y) }}
}

// jag cuts a's edge into tufts below row y0: teeth period wide and up to
// amp deep.
func (a shape) jag(amp, period, y0 float64, seed uint32) shape {
	return shape{a.x0, a.y0, a.x1, a.y1, func(x, y float64) float64 {
		d := a.d(x, y)
		if y <= y0 || d < -amp-1 {
			return d
		}
		u := x/period + 50
		k := math.Floor(u)
		w := math.Abs(u-k-0.5) * 2 // 0 in the middle of a tuft, 1 between two
		return d + amp*w*(0.55+0.45*hashf(int(k), 3, seed))*clamp01((y-y0)/2)
	}}
}

// lum is the light on the shape at pixel (x, y), as if it bulged: the
// surface turns from facing its edge to facing the viewer over round pixels.
func (a shape) lum(x, y int, round float64) float64 {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	gx := a.d(fx+0.75, fy) - a.d(fx-0.75, fy)
	gy := a.d(fx, fy+0.75) - a.d(fx, fy-0.75)
	n := math.Hypot(gx, gy)
	if n < 1e-9 {
		return lum(0, 0)
	}
	k := (1 - clamp01(-a.d(fx, fy)/round)) / n
	return lum(gx*k, gy*k)
}

// shader colours pixel (x, y) of a shape, d deep inside it (negative).
type shader func(x, y int, d float64) color.RGBA

func plain(c color.RGBA) shader { return func(int, int, float64) color.RGBA { return c } }

// tone picks the colour of r for light v in [0, 1], in clean bands.
func toneOf(r ramp, v float64) color.RGBA { return r.at(int(v * float64(len(r)))) }

// bevel shades a with r as a bulging surface; bias moves it lighter or darker.
func bevel(r ramp, a shape, round, bias float64) shader {
	return func(x, y int, _ float64) color.RGBA { return toneOf(r, a.lum(x, y, round)+bias) }
}

// ext6 makes a six-tone ramp of a four-tone one: a deeper shadow below and
// a highlight above.
func ext6(r ramp) ramp {
	n := len(r) - 1
	return ramp{mix(r[0], colInk, 0.38), r[0], r[1], r[n-1], r[n], mix(r[n], colWhite, 0.42)}
}

// figLayer is one layer of a figure: its pixels, and for each the number of
// the part it belongs to.
type figLayer struct {
	c    *canvas
	part []uint16
}

func newFigLayer(w, h int) *figLayer {
	return &figLayer{newCanvas(w, h), make([]uint16, w*h)}
}

// figHD draws a figure in parts. Each fill starts a new part; add goes on
// with the last one.
type figHD struct {
	n      uint16
	ox, oy float64 // every shape is shifted by this
	// xf bends the figure as it is drawn: each in turn (the innermost
	// first) carries every shape, dot and shader of it, before the shift.
	// A torso leaning from the hips, a head nodding on the neck, a jaw
	// dropped, the whole figure toppling over its feet: see motion.go.
	xf []figMap
}

// figMap carries the figure's own pixels somewhere else: to, and back.
type figMap interface {
	to(x, y float64) (float64, float64)
	from(x, y float64) (float64, float64)
}

// figXf is an affine map: (x, y) goes to (a*x + b*y + c, d*x + e*y + f).
type figXf struct{ a, b, c, d, e, f float64 }

// xfRot turns by ang about (cx, cy): clockwise on the screen.
func xfRot(ang, cx, cy float64) figXf {
	sn, cs := math.Sin(ang), math.Cos(ang)
	return figXf{cs, -sn, cx - cs*cx + sn*cy, sn, cs, cy - sn*cx - cs*cy}
}

// xfScale scales by kx, ky about (cx, cy): a torso bent towards or away
// from the viewer is seen shorter.
func xfScale(kx, ky, cx, cy float64) figXf {
	return figXf{kx, 0, cx - kx*cx, 0, ky, cy - ky*cy}
}

// xfMove shifts by (dx, dy).
func xfMove(dx, dy float64) figXf { return figXf{1, 0, dx, 0, 1, dy} }

func (m figXf) to(x, y float64) (float64, float64) {
	return m.a*x + m.b*y + m.c, m.d*x + m.e*y + m.f
}

func (m figXf) from(x, y float64) (float64, float64) {
	det := m.a*m.e - m.b*m.d
	x, y = x-m.c, y-m.f
	return (m.e*x - m.b*y) / det, (-m.d*x + m.a*y) / det
}

// jawMap drops everything below row y0 between columns x0 and x1 by dy:
// a jaw opened as it chews. The rows it leaves are the lip's, stretched.
type jawMap struct{ x0, x1, y0, dy float64 }

func (m jawMap) to(x, y float64) (float64, float64) {
	if x >= m.x0 && x < m.x1 && y > m.y0 {
		return x, y + m.dy
	}
	return x, y
}

func (m jawMap) from(x, y float64) (float64, float64) {
	if x >= m.x0 && x < m.x1 && y > m.y0 {
		return x, math.Max(m.y0, y-m.dy)
	}
	return x, y
}

// fwd is where the figure's point (x, y) lands, bent by xf (before the
// shift).
func (f *figHD) fwd(x, y float64) (float64, float64) {
	for _, m := range f.xf {
		x, y = m.to(x, y)
	}
	return x, y
}

func (f *figHD) fill(l *figLayer, s shape, col shader) {
	f.n++
	f.add(l, s, col)
}

func (f *figHD) add(l *figLayer, s shape, col shader) {
	if len(f.xf) > 0 {
		f.addBent(l, s, col)
		return
	}
	c := l.c
	x0, y0 := max(0, int(math.Floor(s.x0+f.ox))-1), max(0, int(math.Floor(s.y0+f.oy))-1)
	x1, y1 := min(c.w-1, int(math.Ceil(s.x1+f.ox))+1), min(c.h-1, int(math.Ceil(s.y1+f.oy))+1)
	ix, iy := int(math.Round(f.ox)), int(math.Round(f.oy))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			d := s.d(float64(x)+0.5-f.ox, float64(y)+0.5-f.oy)
			if d > 0 {
				continue
			}
			if p := col(x-ix, y-iy, d); p.A > 0 {
				c.px[y*c.w+x] = p
				l.part[y*c.w+x] = f.n
			}
		}
	}
}

// addBent is add with the figure bent by xf: each pixel of the canvas
// takes the shape and the shading of the point of the figure that lands
// on it.
func (f *figHD) addBent(l *figLayer, s shape, col shader) {
	c := l.c
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	clampF := func(v float64) float64 { return math.Max(-4096, math.Min(4096, v)) }
	for _, p := range [4][2]float64{{s.x0, s.y0}, {s.x1, s.y0}, {s.x0, s.y1}, {s.x1, s.y1}} {
		px, py := f.fwd(clampF(p[0]), clampF(p[1]))
		x0, y0, x1, y1 = math.Min(x0, px), math.Min(y0, py), math.Max(x1, px), math.Max(y1, py)
	}
	// A turn can carry a corner of the box further than the corners: pad
	// by how far the box's middle may swing.
	pad := 0.5 * math.Hypot(clampF(s.x1)-clampF(s.x0), clampF(s.y1)-clampF(s.y0))
	cx, cy := f.fwd((clampF(s.x0)+clampF(s.x1))/2, (clampF(s.y0)+clampF(s.y1))/2)
	x0, y0 = math.Max(x0, cx-pad)-3, math.Max(y0, cy-pad)-3
	x1, y1 = math.Min(x1, cx+pad)+3, math.Min(y1, cy+pad)+3
	ix0, iy0 := max(0, int(math.Floor(x0+f.ox))-1), max(0, int(math.Floor(y0+f.oy))-1)
	ix1, iy1 := min(c.w-1, int(math.Ceil(x1+f.ox))+1), min(c.h-1, int(math.Ceil(y1+f.oy))+1)
	for y := iy0; y <= iy1; y++ {
		for x := ix0; x <= ix1; x++ {
			sx, sy := float64(x)+0.5-f.ox, float64(y)+0.5-f.oy
			for i := len(f.xf) - 1; i >= 0; i-- {
				sx, sy = f.xf[i].from(sx, sy)
			}
			d := s.d(sx, sy)
			if d > 0 {
				continue
			}
			if p := col(int(math.Floor(sx)), int(math.Floor(sy)), d); p.A > 0 {
				c.px[y*c.w+x] = p
				l.part[y*c.w+x] = f.n
			}
		}
	}
}

// canvasAt is the canvas pixel the figure's pixel (x, y) lands on.
func (f *figHD) canvasAt(x, y int) (int, int) {
	if len(f.xf) == 0 {
		return x + int(math.Round(f.ox)), y + int(math.Round(f.oy))
	}
	px, py := f.fwd(float64(x)+0.5, float64(y)+0.5)
	return int(math.Floor(px + f.ox)), int(math.Floor(py + f.oy))
}

// dot colours one pixel: a detail of the part already there, or of the
// last part if the pixel was empty.
func (f *figHD) dot(l *figLayer, x, y int, col color.RGBA) {
	x, y = f.canvasAt(x, y)
	c := l.c
	if !c.in(x, y) {
		return
	}
	i := y*c.w + x
	if c.px[i].A < 128 {
		l.part[i] = f.n
	}
	c.px[i] = col
}

// stamp draws a block of pixels, one letter each, with its top-left at
// (x0, y0); flip mirrors it.
func (f *figHD) stamp(l *figLayer, x0, y0 int, rows []string, flip bool, pal func(byte) (color.RGBA, bool)) {
	for dy, row := range rows {
		for i := 0; i < len(row); i++ {
			col, ok := pal(row[i])
			if !ok {
				continue
			}
			x := x0 + i
			if flip {
				x = x0 + len(row) - 1 - i
			}
			f.dot(l, x, y0+dy, col)
		}
	}
}

// at reads a layer's pixel, in the figure's own coordinates.
func (f *figHD) at(l *figLayer, x, y int) (color.RGBA, uint16) {
	x, y = f.canvasAt(x, y)
	if !l.c.in(x, y) {
		return color.RGBA{}, 0
	}
	return l.c.px[y*l.c.w+x], l.part[y*l.c.w+x]
}

// innerLines darkens, on the layers laid one over the next, each pixel of a
// part that touches a part lying over it: the fine line that parts a hand
// from a sleeve and a lock of hair from the one behind.
func innerLines(strength float64, layers ...*figLayer) {
	c0 := layers[0].c
	type key struct {
		l int
		p uint16
	}
	top := func(x, y int) (key, bool) {
		if !c0.in(x, y) {
			return key{}, false
		}
		i := y*c0.w + x
		for k := len(layers) - 1; k >= 0; k-- {
			if layers[k].c.px[i].A >= 128 {
				return key{k, layers[k].part[i]}, true
			}
		}
		return key{}, false
	}
	var marks []int
	var which []int
	for y := 0; y < c0.h; y++ {
		for x := 0; x < c0.w; x++ {
			a, ok := top(x, y)
			if !ok {
				continue
			}
			for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
				b, ok := top(x+d[0], y+d[1])
				if ok && (b.l > a.l || b.l == a.l && b.p > a.p) {
					marks = append(marks, y*c0.w+x)
					which = append(which, a.l)
					break
				}
			}
		}
	}
	for k, i := range marks {
		px := &layers[which[k]].c.px[i]
		*px = mix(*px, colInk, strength)
	}
}

// --- materials ---------------------------------------------------------------

// furShade is fur: a bulging surface broken into rows of pointed tufts,
// each lit on its crown and dark where it parts from its neighbours.
func furShade(r ramp, a shape, round, bias float64, seed uint32) shader {
	n := len(r)
	return func(x, y int, _ float64) color.RGBA {
		i := int((a.lum(x, y, round) + bias) * float64(n))
		i = max(1, min(n-2, i))
		row := int(math.Floor(float64(y) / 4))
		xo := float64(x) + float64(row&1)*2.5 + 3*hashf(row, 9, seed)
		u := xo / 5
		k := math.Floor(u)
		e := float64(y-row*4)/4 + math.Abs(u-k-0.45)*1.25
		h := hashf(int(k), row, seed)
		switch {
		case e < 0.42+0.2*h:
			i++
		case e > 0.98:
			i--
		}
		return r.at(i)
	}
}

// hairShade is hair combed away from the crown (cx, cy): strands that fan
// out over the skull and hang straight below it, a groove here and there
// between them, and a band of shine across them on the lit side.
func hairShade(r ramp, a shape, round, cx, cy float64, seed uint32) shader {
	return hairShadeK(r, a, round, cx, cy, seed, 1)
}

// hairShadeK is hairShade for a head k times the size: the strands and the
// shine are as much finer.
func hairShadeK(r ramp, a shape, round, cx, cy float64, seed uint32, k float64) shader {
	n := len(r)
	return func(x, y int, _ float64) color.RGBA {
		i := int(a.lum(x, y, round)*float64(n)*0.74) + 1
		i = max(1, min(n-2, i))
		fx, fy := (float64(x)+0.5-cx)/k, (float64(y)+0.5-cy)/k
		const fan = 13.0
		u := fx*fan/math.Max(1.5, math.Min(fy, fan))/2.4 + 200
		k := math.Floor(u)
		h := hashf(int(k), 7, seed)
		if u-k < 0.42 && h < 0.42 && hashf(int(k), int(math.Floor(fy/9+h*7)), seed+1) < 0.62 {
			i-- // a groove, broken along its length
		}
		// The shine: a ragged ring round the crown, and another lower down
		// on hair that hangs.
		rr := math.Hypot(fx*0.9, fy)
		for k, band := range [2]float64{11.5, 33} {
			if e := math.Abs(rr - band - (h-0.5)*4); e < 1.9 && fx < 9 && fx > -19 {
				i++
				if k == 0 && e < 0.9 && h > 0.5 {
					i++
				}
			}
		}
		return r.at(min(i, n-1))
	}
}
