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
	// s scales the drawing calls (ellipse, rect, polygon, line, set and
	// drawTubeWith): a shape drawn in design units comes out s times bigger,
	// with the same one-pixel detail. 0 means 1. Callbacks, at and the
	// pixel loops of outline and over work in pixels.
	s float64
	// tx and ty shift the drawing calls by that many design units, to make room
	// above or beside a design without redrawing it.
	tx, ty float64
	// ol marks the pixels outline drew, so that upscale3 (hdup.go) can thin
	// them; nil until outline is called. dens is HDDensity for a canvas whose
	// pixels are already HD image pixels although its drawing calls are not
	// scaled (see mkUp).
	ol   []bool
	dens int
	// lw is how many pixels wide line draws, 0 or 1 for a hairline: a stem
	// drawn at HD keeps some body.
	lw int
}

func newCanvas(w, h int) *canvas { return &canvas{w: w, h: h, px: make([]color.RGBA, w*h)} }

// newScaledCanvas is a canvas of w by h design units drawn s times bigger.
func newScaledCanvas(w, h int, s float64) *canvas {
	c := newCanvas(int(math.Ceil(float64(w)*s)), int(math.Ceil(float64(h)*s)))
	c.s = s
	return c
}

// k is the canvas's scale.
func (c *canvas) k() float64 {
	if c.s == 0 {
		return 1
	}
	return c.s
}

// lx and ly are the design coordinates of the centre of pixel column x and
// row y.
func (c *canvas) lx(x int) float64 { return (float64(x)+0.5)/c.k() - c.tx }
func (c *canvas) ly(y int) float64 { return (float64(y)+0.5)/c.k() - c.ty }

// toPx maps design point (x, y) to pixel space.
func (c *canvas) toPx(x, y float64) (float64, float64) {
	k := c.k()
	return (x + c.tx) * k, (y + c.ty) * k
}

func (c *canvas) in(x, y int) bool { return x >= 0 && y >= 0 && x < c.w && y < c.h }

// at reads the pixel at (x, y).
func (c *canvas) at(x, y int) color.RGBA {
	if !c.in(x, y) {
		return color.RGBA{}
	}
	return c.px[y*c.w+x]
}

// atL reads the pixel at design point (x, y).
func (c *canvas) atL(x, y int) color.RGBA {
	px, py := c.toPx(float64(x)+0.5, float64(y)+0.5)
	return c.at(int(math.Floor(px)), int(math.Floor(py)))
}

// put writes the pixel at (x, y).
func (c *canvas) put(x, y int, col color.RGBA) {
	if c.in(x, y) {
		c.px[y*c.w+x] = col
		if c.ol != nil {
			c.ol[y*c.w+x] = false
		}
	}
}

// set fills design unit (x, y): one pixel, or on a scaled canvas the block
// of pixels it covers.
func (c *canvas) set(x, y int, col color.RGBA) {
	if (c.s == 0 || c.s == 1) && c.tx == 0 && c.ty == 0 {
		c.put(x, y, col)
		return
	}
	x0, y0 := c.toPx(float64(x), float64(y))
	x1, y1 := c.toPx(float64(x+1), float64(y+1))
	for py := int(math.Floor(y0)); py < int(math.Floor(y1)); py++ {
		for px := int(math.Floor(x0)); px < int(math.Floor(x1)); px++ {
			c.put(px, py, col)
		}
	}
}

func (c *canvas) opaque(x, y int) bool { return c.at(x, y).A >= 128 }

// ellipse fills the pixels whose centres lie inside the ellipse, colouring
// each from its normalised offset (nx, ny) in [-1, 1].
func (c *canvas) ellipse(cx, cy, rx, ry float64, f func(x, y int, nx, ny float64) color.RGBA) {
	k := c.k()
	cx, cy = c.toPx(cx, cy)
	rx, ry = rx*k, ry*k
	for y := int(math.Floor(cy - ry)); y <= int(math.Ceil(cy+ry)); y++ {
		for x := int(math.Floor(cx - rx)); x <= int(math.Ceil(cx+rx)); x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if nx*nx+ny*ny <= 1 {
				if col := f(x, y, nx, ny); col.A > 0 {
					c.put(x, y, col)
				}
			}
		}
	}
}

func (c *canvas) rect(x0, y0, w, h int, f func(x, y int) color.RGBA) {
	fx0, fy0 := c.toPx(float64(x0), float64(y0))
	fx1, fy1 := c.toPx(float64(x0+w), float64(y0+h))
	px0, py0 := int(math.Floor(fx0)), int(math.Floor(fy0))
	px1, py1 := int(math.Floor(fx1)), int(math.Floor(fy1))
	for y := py0; y < py1; y++ {
		for x := px0; x < px1; x++ {
			if col := f(x, y); col.A > 0 {
				c.put(x, y, col)
			}
		}
	}
}

func solid(col color.RGBA) func(x, y int) color.RGBA {
	return func(int, int) color.RGBA { return col }
}

// polygon fills the pixels whose centres are inside the polygon (even-odd).
func (c *canvas) polygon(pts []float64, f func(x, y int) color.RGBA) {
	if c.k() != 1 || c.tx != 0 || c.ty != 0 {
		scaled := make([]float64, len(pts))
		for i := 0; i+1 < len(pts); i += 2 {
			scaled[i], scaled[i+1] = c.toPx(pts[i], pts[i+1])
		}
		pts = scaled
	}
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
					c.put(x, y, col)
				}
			}
		}
	}
}

// line plots a 1-pixel line; f gets the position t in [0, 1] along it.
func (c *canvas) line(x0, y0, x1, y1 float64, f func(x, y int, t float64) color.RGBA) {
	x0, y0 = c.toPx(x0, y0)
	x1, y1 = c.toPx(x1, y1)
	n := int(math.Ceil(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2)) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := int(math.Floor(lerpf(x0, x1, t))), int(math.Floor(lerpf(y0, y1, t)))
		if c.lw > 1 {
			for dy := 0; dy < c.lw; dy++ {
				for dx := 0; dx < c.lw; dx++ {
					if col := f(x+dx-c.lw/2, y+dy-c.lw/2, t); col.A > 0 {
						c.put(x+dx-c.lw/2, y+dy-c.lw/2, col)
					}
				}
			}
			continue
		}
		if col := f(x, y, t); col.A > 0 {
			c.put(x, y, col)
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
					if c.ol == nil {
						c.ol = make([]bool, len(c.px))
					}
					c.ol[y*c.w+x] = true
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
				c.put(x+dx, y+dy, p)
				if c.in(x+dx, y+dy) && (o.ol != nil || c.ol != nil) {
					if c.ol == nil {
						c.ol = make([]bool, len(c.px))
					}
					c.ol[(y+dy)*c.w+x+dx] = o.ol != nil && o.ol[y*o.w+x]
				}
			}
		}
	}
}

func (c *canvas) mirrored() *canvas {
	m := newCanvas(c.w, c.h)
	m.dens = c.dens
	if c.ol != nil {
		m.ol = make([]bool, len(c.ol))
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			m.px[y*c.w+(c.w-1-x)] = c.px[y*c.w+x]
			if c.ol != nil {
				m.ol[y*c.w+(c.w-1-x)] = c.ol[y*c.w+x]
			}
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
	artBytes.Add(int64(4 * c.w * c.h)) // see ArtCost in scalesheet.go
	img := ebiten.NewImage(c.w, c.h)
	img.WritePixels(c.bytes())
	return img
}
