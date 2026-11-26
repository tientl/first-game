// Package dinopaint paints the HD dinosaurs: each species' density-1
// drawing (internal/client/dinos/<name>.png) is the guide for the silhouette,
// the colour regions and the layout, and the drawing three times as fine
// (dinos/hd/<name>.png) is painted over it from a short description of the
// animal: its parts in depth order, the muscles on them, its colours, its
// skin, and the details of its head and feet. See dinos/hd/README.md.
package dinopaint

import "math"

// Field is a grid of numbers, one per pixel.
type Field struct {
	W, H int
	V    []float64
}

// NewField is a w x h field of zeros.
func NewField(w, h int) *Field { return &Field{w, h, make([]float64, w*h)} }

// At is the value at pixel (x, y), clamped to the edges.
func (f *Field) At(x, y int) float64 {
	x = min(f.W-1, max(0, x))
	y = min(f.H-1, max(0, y))
	return f.V[y*f.W+x]
}

// Sample is the field at (x, y) in pixels, interpolated between pixel
// centres (which are at +0.5).
func (f *Field) Sample(x, y float64) float64 {
	x -= 0.5
	y -= 0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	tx, ty := x-x0, y-y0
	ix, iy := int(x0), int(y0)
	a := f.At(ix, iy)*(1-tx) + f.At(ix+1, iy)*tx
	b := f.At(ix, iy+1)*(1-tx) + f.At(ix+1, iy+1)*tx
	return a*(1-ty) + b*ty
}

// Blur is the field blurred by a gaussian of sigma pixels.
func (f *Field) Blur(sigma float64) *Field {
	if sigma <= 0 {
		return &Field{f.W, f.H, append([]float64(nil), f.V...)}
	}
	r := int(math.Ceil(sigma * 2.5))
	k := make([]float64, 2*r+1)
	sum := 0.0
	for i := range k {
		d := float64(i - r)
		k[i] = math.Exp(-d * d / (2 * sigma * sigma))
		sum += k[i]
	}
	for i := range k {
		k[i] /= sum
	}
	tmp := NewField(f.W, f.H)
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			s := 0.0
			for i, w := range k {
				s += w * f.At(x+i-r, y)
			}
			tmp.V[y*f.W+x] = s
		}
	}
	out := NewField(f.W, f.H)
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			s := 0.0
			for i, w := range k {
				s += w * tmp.At(x, y+i-r)
			}
			out.V[y*f.W+x] = s
		}
	}
	return out
}

// edt is the distance from each pixel's centre to the centre of the nearest
// pixel where mask is set (exact Euclidean; Felzenszwalb and Huttenlocher).
func edt(mask []bool, w, h int) []float64 {
	const inf = 1e20
	g := make([]float64, w*h)
	for i, m := range mask {
		if m {
			g[i] = 0
		} else {
			g[i] = inf
		}
	}
	n := max(w, h)
	f, d := make([]float64, n), make([]float64, n)
	v, z := make([]int, n), make([]float64, n+1)
	pass := func(m int) {
		k := 0
		v[0], z[0], z[1] = 0, -inf, inf
		for q := 1; q < m; q++ {
			s := ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) / float64(2*q-2*v[k])
			for s <= z[k] {
				k--
				s = ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) / float64(2*q-2*v[k])
			}
			k++
			v[k], z[k], z[k+1] = q, s, inf
		}
		k = 0
		for q := 0; q < m; q++ {
			for z[k+1] < float64(q) {
				k++
			}
			d[q] = float64((q-v[k])*(q-v[k])) + f[v[k]]
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			f[y] = g[y*w+x]
		}
		pass(h)
		for y := 0; y < h; y++ {
			g[y*w+x] = d[y]
		}
	}
	for y := 0; y < h; y++ {
		copy(f[:w], g[y*w:(y+1)*w])
		pass(w)
		for x := 0; x < w; x++ {
			g[y*w+x] = math.Sqrt(d[x])
		}
	}
	return g
}

// sdfOf is the signed distance of a mask, in pixels: negative inside,
// about -0.5 on the pixels along its edge and +0.5 just outside.
func sdfOf(mask []bool, w, h int) *Field {
	out := make([]bool, len(mask))
	for i, m := range mask {
		out[i] = !m
	}
	din, dout := edt(mask, w, h), edt(out, w, h)
	f := NewField(w, h)
	for i, m := range mask {
		if m {
			f.V[i] = -(dout[i] - 0.5)
		} else {
			f.V[i] = din[i] - 0.5
		}
	}
	return f
}

func clamp01(v float64) float64 { return min(1, max(0, v)) }

func smoothstep(a, b, x float64) float64 {
	t := clamp01((x - a) / (b - a))
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// hash is a number in [0, 1) for the integer point (x, y) and seed s.
func hash(x, y int, s uint32) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + s*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(1<<24)
}

// noise is smooth value noise in [0, 1) at (x, y), a cell a unit across.
func noise(x, y float64, s uint32) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	tx, ty := x-x0, y-y0
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	ix, iy := int(x0), int(y0)
	a := lerp(hash(ix, iy, s), hash(ix+1, iy, s), tx)
	b := lerp(hash(ix, iy+1, s), hash(ix+1, iy+1, s), tx)
	return lerp(a, b, ty)
}

// fbm is a few octaves of noise.
func fbm(x, y float64, s uint32, oct int) float64 {
	sum, amp, tot := 0.0, 1.0, 0.0
	for i := 0; i < oct; i++ {
		sum += amp * noise(x, y, s+uint32(i)*101)
		tot += amp
		amp *= 0.5
		x, y = x*2.03+17.1, y*2.03+5.3
	}
	return sum / tot
}
