package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// This file holds the small software rasteriser the placeholder art is built
// with: hard-edged pixel shapes, palette ramps with ordered dithering, and an
// outline pass. Everything is drawn in art pixels; the world is later scaled
// up with nearest-neighbour filtering, so the result reads as pixel art.

// --- noise ------------------------------------------------------------------

func hash2(x, y int32, seed uint32) uint32 {
	h := uint32(x)*0x27d4eb2d ^ uint32(y)*0x165667b1 ^ seed*0x9e3779b1
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

// hashf is a stable pseudo-random value in [0, 1) for an integer point.
func hashf(x, y int, seed uint32) float64 {
	return float64(hash2(int32(x), int32(y), seed)) / (1 << 32)
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func lerpf(a, b, t float64) float64 { return a + (b-a)*t }

func clamp01(v float64) float64 { return max(0, min(1, v)) }

// vnoise is smooth value noise in [0, 1).
func vnoise(x, y float64, seed uint32) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	sx, sy := smooth(x-x0), smooth(y-y0)
	return lerpf(
		lerpf(hashf(ix, iy, seed), hashf(ix+1, iy, seed), sx),
		lerpf(hashf(ix, iy+1, seed), hashf(ix+1, iy+1, seed), sx), sy)
}

// pnoise is value noise that tiles with period (px, py) lattice cells.
func pnoise(x, y float64, px, py int, seed uint32) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	sx, sy := smooth(x-x0), smooth(y-y0)
	m := func(v, p int) int { return ((v % p) + p) % p }
	h := func(a, b int) float64 { return hashf(m(a, px), m(b, py), seed) }
	return lerpf(lerpf(h(ix, iy), h(ix+1, iy), sx), lerpf(h(ix, iy+1), h(ix+1, iy+1), sx), sy)
}

func fbm2(x, y float64, seed uint32, octaves int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * vnoise(x, y, seed+uint32(i)*1013)
		norm += amp
		amp *= 0.5
		x, y = x*2, y*2
	}
	return sum / norm
}

var bayer4 = [4][4]float64{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// dither returns the ordered-dither threshold in (0, 1) for a pixel.
func dither(x, y int) float64 { return (bayer4[y&3][x&3] + 0.5) / 16 }

// --- colours ----------------------------------------------------------------

func hex(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = clamp01(t)
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), f(a.A, b.A)}
}

func withAlpha(c color.RGBA, a uint8) color.RGBA { c.A = a; return c }

// ramp is a palette from dark to light.
type ramp []color.RGBA

func mkRamp(vs ...uint32) ramp {
	r := make(ramp, len(vs))
	for i, v := range vs {
		r[i] = hex(v)
	}
	return r
}

// idx maps t in [0, 1] to a palette index, dithering between neighbours.
func (r ramp) idx(t float64, x, y int) int {
	p := clamp01(t) * float64(len(r)-1)
	i := int(p)
	if p-float64(i) > dither(x, y) {
		i++
	}
	return min(i, len(r)-1)
}

func (r ramp) at(i int) color.RGBA { return r[max(0, min(len(r)-1, i))] }

func (r ramp) pick(t float64, x, y int) color.RGBA { return r[r.idx(t, x, y)] }

// lightDir points towards the sun: up, left and out of the screen.
var lightDir = func() [3]float64 {
	x, y, z := -0.55, -0.7, 0.55
	n := math.Sqrt(x*x + y*y + z*z)
	return [3]float64{x / n, y / n, z / n}
}()

// lum returns the brightness of a sphere-like surface with the given screen
// normal (nx, ny); nz is derived.
func lum(nx, ny float64) float64 {
	d := nx*nx + ny*ny
	if d > 1 {
		s := math.Sqrt(d)
		nx, ny, d = nx/s, ny/s, 1
	}
	nz := math.Sqrt(1 - d)
	l := nx*lightDir[0] + ny*lightDir[1] + nz*lightDir[2]
	return clamp01(0.2 + 0.8*l)
}

// --- canvas -----------------------------------------------------------------

type canvas struct {
	w, h int
	px   []color.RGBA // straight (not premultiplied) alpha
}

func newCanvas(w, h int) *canvas { return &canvas{w: w, h: h, px: make([]color.RGBA, w*h)} }

func (c *canvas) in(x, y int) bool { return x >= 0 && y >= 0 && x < c.w && y < c.h }

func (c *canvas) at(x, y int) color.RGBA {
	if !c.in(x, y) {
		return color.RGBA{}
	}
	return c.px[y*c.w+x]
}

func (c *canvas) set(x, y int, col color.RGBA) {
	if c.in(x, y) {
		c.px[y*c.w+x] = col
	}
}

func (c *canvas) opaque(x, y int) bool { return c.at(x, y).A >= 128 }

// ellipse fills the pixels whose centres lie inside the ellipse, colouring
// each from its normalised offset (nx, ny) in [-1, 1].
func (c *canvas) ellipse(cx, cy, rx, ry float64, f func(x, y int, nx, ny float64) color.RGBA) {
	for y := int(math.Floor(cy - ry)); y <= int(math.Ceil(cy+ry)); y++ {
		for x := int(math.Floor(cx - rx)); x <= int(math.Ceil(cx+rx)); x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if nx*nx+ny*ny <= 1 {
				if col := f(x, y, nx, ny); col.A > 0 {
					c.set(x, y, col)
				}
			}
		}
	}
}

func (c *canvas) rect(x0, y0, w, h int, f func(x, y int) color.RGBA) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			if col := f(x, y); col.A > 0 {
				c.set(x, y, col)
			}
		}
	}
}

func solid(col color.RGBA) func(x, y int) color.RGBA {
	return func(int, int) color.RGBA { return col }
}

// polygon fills the pixels whose centres are inside the polygon (even-odd).
func (c *canvas) polygon(pts []float64, f func(x, y int) color.RGBA) {
	minY, maxY := math.Inf(1), math.Inf(-1)
	minX, maxX := math.Inf(1), math.Inf(-1)
	for i := 0; i < len(pts); i += 2 {
		minX, maxX = math.Min(minX, pts[i]), math.Max(maxX, pts[i])
		minY, maxY = math.Min(minY, pts[i+1]), math.Max(maxY, pts[i+1])
	}
	n := len(pts) / 2
	for y := int(minY); y <= int(maxY); y++ {
		for x := int(minX); x <= int(maxX); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			inside := false
			for i, j := 0, n-1; i < n; j, i = i, i+1 {
				xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
				if (yi > py) != (yj > py) && px < (xj-xi)*(py-yi)/(yj-yi)+xi {
					inside = !inside
				}
			}
			if inside {
				if col := f(x, y); col.A > 0 {
					c.set(x, y, col)
				}
			}
		}
	}
}

// line plots a 1-pixel line; f gets the position t in [0, 1] along it.
func (c *canvas) line(x0, y0, x1, y1 float64, f func(x, y int, t float64) color.RGBA) {
	n := int(math.Ceil(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2)) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := int(math.Floor(lerpf(x0, x1, t))), int(math.Floor(lerpf(y0, y1, t)))
		if col := f(x, y, t); col.A > 0 {
			c.set(x, y, col)
		}
	}
}

var colInk = hex(0x1b1424)

// outline adds a dark rim around every opaque region, tinted by the colour
// it borders, the usual "selective outline" of pixel art.
func (c *canvas) outline(strength float64) {
	src := make([]color.RGBA, len(c.px))
	copy(src, c.px)
	get := func(x, y int) color.RGBA {
		if !c.in(x, y) {
			return color.RGBA{}
		}
		return src[y*c.w+x]
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if get(x, y).A >= 128 {
				continue
			}
			for _, d := range [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
				if n := get(x+d[0], y+d[1]); n.A >= 128 {
					c.px[y*c.w+x] = mix(withAlpha(n, 255), colInk, strength)
					break
				}
			}
		}
	}
}

// over draws o onto c with its top-left at (dx, dy).
func (c *canvas) over(o *canvas, dx, dy int) {
	for y := 0; y < o.h; y++ {
		for x := 0; x < o.w; x++ {
			if p := o.px[y*o.w+x]; p.A >= 128 {
				c.set(x+dx, y+dy, p)
			}
		}
	}
}

func (c *canvas) mirrored() *canvas {
	m := newCanvas(c.w, c.h)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			m.px[y*c.w+(c.w-1-x)] = c.px[y*c.w+x]
		}
	}
	return m
}

func (c *canvas) bytes() []byte {
	b := make([]byte, len(c.px)*4)
	for i, p := range c.px {
		a := uint32(p.A)
		b[4*i] = uint8(uint32(p.R) * a / 255)
		b[4*i+1] = uint8(uint32(p.G) * a / 255)
		b[4*i+2] = uint8(uint32(p.B) * a / 255)
		b[4*i+3] = p.A
	}
	return b
}

func (c *canvas) image() *ebiten.Image {
	img := ebiten.NewImage(c.w, c.h)
	img.WritePixels(c.bytes())
	return img
}
