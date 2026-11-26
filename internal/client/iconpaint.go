package client

import (
	"image/color"
	"math"
)

// The painter of the HD item icons (iconhd.go). An icon is painted in a
// 16 x 16 design box, in parts drawn back to front, each a shape (hdfig.go:
// a signed distance, here in design units) coloured pixel by pixel by a
// shader. Whatever the size it is baked at, the painter runs once per pixel
// of the result, so the same code gives a crisp icon at 48, 96 or 192
// pixels. Then, as a pixel artist would:
//
//   - a part lying over another gets a dark line on the one beneath;
//   - the outline is selective: near ink on the shadow side (bottom right),
//     the colour it borders darkened on the lit side (top left);
//   - a soft shadow falls down and to the right of the whole thing;
//   - what glows (a flame's halo, steam) goes over all that, soft.
//
// The light comes from the top left, like the world's (lightDir). Shaders
// pick tones from a ramp in clean bands, never blended: 3 or 4 steps of a
// material and a highlight.

// iconBox is the design box: an HD icon covers what the old 16x16 one did.
const iconBox = 16.0

// icv is an icon being painted, n x n pixels.
type icv struct {
	n     int
	k     float64      // pixels to a design unit
	o     float64      // the margin, pixels: design point 0 is pixel o
	px    []color.RGBA // straight alpha
	part  []uint8      // the part each pixel belongs to, 0 for none
	parts uint8
	bare  [256]bool    // parts that get no outline nor contact line (a flame)
	glow  []color.RGBA // soft paint over everything, premultiplied
}

// newIcv is an icon n pixels across, the design box inside a margin m
// pixels wide all round, where the outline goes.
func newIcv(n, m int) *icv {
	return &icv{n: n, k: float64(n-2*m) / iconBox, o: float64(m), px: make([]color.RGBA, n*n), part: make([]uint8, n*n)}
}

// ifrag is what a shader knows of the pixel it colours.
type ifrag struct {
	x, y   float64 // the pixel's centre, design units
	ix, iy int     // the pixel
	d      float64 // distance to the part's edge, design units, <= 0 inside
	gx, gy float64 // the outward normal of that edge, unit length
	px     float64 // one pixel, in design units
}

// ishader colours a pixel of a part; a colour with A = 0 leaves it alone.
type ishader func(f *ifrag) color.RGBA

const (
	icFill  = iota // a new part
	icAdd          // more of the last part
	icOn           // only over pixels already painted, keeping their part
	icUnder        // only where nothing is painted yet, as a new part
)

func (p *icv) paint(s shape, sh ishader, mode int) {
	if mode == icFill || mode == icUnder {
		p.parts++
	}
	k, o := p.k, p.o
	x0, y0 := max(0, int(math.Floor(s.x0*k+o))-1), max(0, int(math.Floor(s.y0*k+o))-1)
	x1, y1 := min(p.n-1, int(math.Ceil(s.x1*k+o))+1), min(p.n-1, int(math.Ceil(s.y1*k+o))+1)
	h := 0.35 / k
	f := ifrag{px: 1 / k}
	for iy := y0; iy <= y1; iy++ {
		for ix := x0; ix <= x1; ix++ {
			x, y := (float64(ix)+0.5-o)/k, (float64(iy)+0.5-o)/k
			d := s.d(x, y)
			if d > 0 {
				continue
			}
			i := iy*p.n + ix
			switch mode {
			case icOn:
				if p.px[i].A == 0 {
					continue
				}
			case icUnder:
				if p.px[i].A != 0 {
					continue
				}
			}
			gx, gy := s.d(x+h, y)-s.d(x-h, y), s.d(x, y+h)-s.d(x, y-h)
			if m := math.Hypot(gx, gy); m > 1e-12 {
				gx, gy = gx/m, gy/m
			}
			f.x, f.y, f.ix, f.iy, f.d, f.gx, f.gy = x, y, ix, iy, d, gx, gy
			c := sh(&f)
			if c.A == 0 {
				continue
			}
			p.px[i] = c
			if mode != icOn {
				p.part[i] = p.parts
			}
		}
	}
}

// fill paints s as a new part.
func (p *icv) fill(s shape, sh ishader) { p.paint(s, sh, icFill) }

// add paints s as more of the last part: no line between them.
func (p *icv) add(s shape, sh ishader) { p.paint(s, sh, icAdd) }

// on paints s only over what is painted already: a mark on it.
func (p *icv) on(s shape, sh ishader) { p.paint(s, sh, icOn) }

// under paints s as a new part only where nothing is painted yet: a part
// behind the others, drawn after them.
func (p *icv) under(s shape, sh ishader) { p.paint(s, sh, icUnder) }

// bare marks the last part as one that gets no outline and puts no line on
// what it covers: a flame, a wisp.
func (p *icv) bareLast() { p.bare[p.parts] = true }

// soft paints s over everything at the end, blended, without an outline:
// a glow or steam. sh's alpha is how much it shows.
func (p *icv) soft(s shape, sh ishader) {
	if p.glow == nil {
		p.glow = make([]color.RGBA, p.n*p.n)
	}
	k, o := p.k, p.o
	x0, y0 := max(0, int(math.Floor(s.x0*k+o))-1), max(0, int(math.Floor(s.y0*k+o))-1)
	x1, y1 := min(p.n-1, int(math.Ceil(s.x1*k+o))+1), min(p.n-1, int(math.Ceil(s.y1*k+o))+1)
	f := ifrag{px: 1 / k}
	for iy := y0; iy <= y1; iy++ {
		for ix := x0; ix <= x1; ix++ {
			x, y := (float64(ix)+0.5-o)/k, (float64(iy)+0.5-o)/k
			d := s.d(x, y)
			if d > 0 {
				continue
			}
			f.x, f.y, f.ix, f.iy, f.d = x, y, ix, iy, d
			c := sh(&f)
			if c.A == 0 {
				continue
			}
			i := iy*p.n + ix
			p.glow[i] = overPM(icPM(c), p.glow[i])
		}
	}
}

// --- light and tone -------------------------------------------------------------

// litOf is the light on a surface whose normal leans (nx, ny) out of the
// screen's plane, as lum (pix.go): 0.2 facing away, 1 facing the light,
// 0.62 facing the viewer.
func litOf(nx, ny float64) float64 { return lum(nx, ny) }

// bulge is the light on a part as if it swelled from its edge: flat on top
// once round units in from the edge, turned to the edge at it.
func bulge(f *ifrag, round float64) float64 {
	t := 1 - clamp01(-f.d/round)
	t = math.Sqrt(t) // the turn is sharp at the very edge, as on a cylinder
	return litOf(f.gx*t, f.gy*t)
}

// pick is the colour of r for light v in [0, 1], in clean bands.
func pick(r ramp, v float64) color.RGBA {
	return r.at(int(math.Floor(clamp01(v) * float64(len(r)) * 0.9999)))
}

// toward is how much the direction (dx, dy) faces the light, -1 to 1.
func toward(dx, dy float64) float64 {
	const lx, ly = -0.6185, -0.7858 // lightDir across the screen, unit
	return dx*lx + dy*ly
}

// iaxis measures points along the line from a to b: u units along it from
// a, v units across it, positive on its right (seen going from a to b).
type iaxis struct{ ax, ay, ex, ey, l float64 }

func mkAxis(ax, ay, bx, by float64) iaxis {
	ex, ey := bx-ax, by-ay
	l := math.Hypot(ex, ey)
	return iaxis{ax, ay, ex / l, ey / l, l}
}

func (a iaxis) uv(x, y float64) (u, v float64) {
	dx, dy := x-a.ax, y-a.ay
	return dx*a.ex + dy*a.ey, dx*-a.ey + dy*a.ex
}

// across is the unit vector of v.
func (a iaxis) across() (float64, float64) { return -a.ey, a.ex }

// rod is the light on a round rod about the axis a, r units thick there,
// at v across it.
func (a iaxis) rod(v, r float64) float64 {
	t := max(-1, min(1, v/r))
	cx, cy := a.across()
	return litOf(cx*t, cy*t)
}

// --- materials ------------------------------------------------------------------

// shRod shades a round rod along a: a stick, a haft, a bone's shaft.
// grain (0 to 1) streaks it along its length; seed varies the streaks.
func shRod(r ramp, a iaxis, rad, bias, grain float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, rad) + bias
		if grain > 0 {
			// Streaks along the rod: noise stretched along it, cut into thin lines.
			n := fbm2(u*0.45, v*3.2/rad+7, seed, 2)
			if s := math.Abs(math.Sin((v/rad*2.6 + n*3.5) * math.Pi)); s > 1-0.16*grain {
				l -= 0.2
			}
		}
		return pick(r, l)
	}
}

// shBulge shades a part as a swelling of r, bias lighter or darker.
func shBulge(r ramp, round, bias float64) ishader {
	return func(f *ifrag) color.RGBA { return pick(r, bulge(f, round)+bias) }
}

// shFlat is one tone of r for the whole part, with a lit lip on the edges
// that face the light and a dark one on those that face away, lip units
// wide (at least a pixel).
func shFlat(r ramp, tone int, lip float64) ishader {
	return func(f *ifrag) color.RGBA {
		t := tone
		if -f.d < max(lip, f.px) {
			if w := toward(f.gx, f.gy); w > 0.35 {
				t++
			} else if w < -0.35 {
				t--
			}
		}
		return r.at(t)
	}
}

// shFacet shades a flat face of stone: tone v of the light, speckled, with
// a lit lip along the edges facing the light and a crease along the rest.
func shFacet(r ramp, v float64, speck float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		l := v
		if speck > 0 {
			l += (fbm2(f.x*1.6, f.y*1.6, seed, 2) - 0.5) * speck
		}
		if -f.d < max(0.45, 1.2*f.px) {
			if w := toward(f.gx, f.gy); w > 0.3 {
				l += 0.22
			} else if w < -0.3 {
				l -= 0.16
			}
		}
		return pick(r, l)
	}
}

// solidC is one colour.
func solidC(c color.RGBA) ishader { return func(*ifrag) color.RGBA { return c } }

// shRing shades a cut end of wood: rings about (cx, cy), squashed to an
// ellipse rx by ry, darker towards the bark.
func shRing(r ramp, cx, cy, rx, ry float64, rings int, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		dx, dy := (f.x-cx)/rx, (f.y-cy)/ry
		d := math.Hypot(dx, dy)
		w := vnoise(math.Atan2(dy, dx)*2+5, 3, seed) * 0.18
		ring := math.Mod((d+w)*float64(rings), 1)
		l := 0.78 - 0.18*d
		if ring < 0.22 {
			l -= 0.2
		}
		if d < 0.12 {
			l -= 0.25 // the pith
		}
		return pick(r, l)
	}
}

// --- the bake ------------------------------------------------------------------

// bakeIcon paints an icon n pixels across. Under 40 pixels it is painted
// m times bigger and shrunk, which smooths its edges cleanly; the outline
// and shadow are then put on at the small size, a pixel thick.
func bakeIcon(f iconPainter, n int) *canvas {
	m := 1
	if n < 40 {
		m = (48 + n - 1) / n
	}
	w := lineW(n)
	// The design box sits inside a margin as wide as the outline and a
	// pixel more, so the outline is never cut off at the image's edge.
	p := newIcv(n*m, (w+1)*m)
	f(p)
	p.contact(w * m)
	px := p.px
	if m > 1 {
		px = shrink(px, n*m, m)
	}
	px = outlineIcon(px, n, w, p.edgeBare(m))
	px = dropShadow(px, n, float64(n)/iconBox)
	if p.glow != nil {
		g := p.glow
		if m > 1 {
			g = shrinkPM(g, n*m, m)
		}
		for i := range px {
			px[i] = icUnPM(overPM(g[i], icPM(px[i])))
		}
	}
	c := newCanvas(n, n)
	copy(c.px, px)
	return c
}

// lineW is how many pixels thick outlines are on an icon n across: one up
// to about 100, then two, so they keep their weight at 4K.
func lineW(n int) int { return max(1, int(math.Round(float64(n)/80))) }

// contact darkens, on each part, a line w pixels wide where a part painted
// after it lies over it.
func (p *icv) contact(w int) {
	n := p.n
	dark := make([]bool, n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := y*n + x
			a := p.part[i]
			if a == 0 {
				continue
			}
		near:
			for dy := -w; dy <= w; dy++ {
				for dx := -w; dx <= w; dx++ {
					if iabs(dx)+iabs(dy) > w || (dx == 0 && dy == 0) {
						continue
					}
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= n || yy >= n {
						continue
					}
					if b := p.part[yy*n+xx]; b > a && !p.bare[b] {
						dark[i] = true
						break near
					}
				}
			}
		}
	}
	for i, d := range dark {
		if d {
			c := p.px[i]
			p.px[i] = withAlpha(mix(c, colInk, 0.62), c.A)
		}
	}
}

// edgeBare marks the pixels, at the baked size, that belong to bare parts,
// which the outline leaves alone.
func (p *icv) edgeBare(m int) []bool {
	n := p.n / m
	any := false
	for _, b := range p.bare {
		any = any || b
	}
	if !any {
		return nil
	}
	out := make([]bool, n*n)
	for y := 0; y < p.n; y++ {
		for x := 0; x < p.n; x++ {
			if p.bare[p.part[y*p.n+x]] && p.part[y*p.n+x] != 0 {
				out[(y/m)*n+x/m] = true
			}
		}
	}
	return out
}

// outlineIcon rims the opaque pixels of px (n across) with a line w
// pixels thick: near ink where the edge faces away from the light, the
// colour beside it darkened where it faces the light. bare pixels are not
// rimmed.
func outlineIcon(px []color.RGBA, n, w int, bare []bool) []color.RGBA {
	solid := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < n && y < n && px[y*n+x].A >= 128
	}
	out := make([]color.RGBA, len(px))
	copy(out, px)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if solid(x, y) {
				continue
			}
			// The nearest solid pixel within w, and the way out from the
			// solid ones round about.
			best, bd := -1, 1<<30
			var ox, oy float64
			for dy := -w; dy <= w; dy++ {
				for dx := -w; dx <= w; dx++ {
					if !solid(x+dx, y+dy) {
						continue
					}
					if dd := dx*dx + dy*dy; dd <= w*w && dd < bd {
						best, bd = (y+dy)*n+x+dx, dd
					}
					ox -= float64(dx)
					oy -= float64(dy)
				}
			}
			if best < 0 || (bare != nil && bare[best]) {
				continue
			}
			c := withAlpha(px[best], 255)
			lit := 0.0
			if m := math.Hypot(ox, oy); m > 0 {
				lit = toward(ox/m, oy/m)
			}
			k := 0.86 - 0.34*clamp01(lit*0.8+0.4)
			o := mix(c, colInk, k)
			// Over what is there (a soft edge of a shrunk icon).
			out[y*n+x] = icUnPM(overPM(icPM(px[y*n+x]), icPM(o)))
		}
	}
	return out
}

// dropShadow lays a soft shadow under px, down and to the right.
func dropShadow(px []color.RGBA, n int, k float64) []color.RGBA {
	ox, oy := int(math.Round(0.35*k)), int(math.Round(0.6*k))
	a := make([]float64, n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			sx, sy := x-ox, y-oy
			if sx >= 0 && sy >= 0 && sx < n && sy < n {
				a[y*n+x] = float64(px[sy*n+sx].A) / 255
			}
		}
	}
	r := max(1, int(math.Round(0.45*k)))
	a = boxBlur(a, n, r)
	a = boxBlur(a, n, r)
	out := make([]color.RGBA, len(px))
	for i, c := range px {
		sh := color.RGBA{0, 0, 0, uint8(math.Round(a[i] * 0.42 * 255))}
		out[i] = icUnPM(overPM(icPM(c), sh))
	}
	return out
}

func boxBlur(a []float64, n, r int) []float64 {
	tmp := make([]float64, len(a))
	out := make([]float64, len(a))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			s := 0.0
			for d := -r; d <= r; d++ {
				if xx := x + d; xx >= 0 && xx < n {
					s += a[y*n+xx]
				}
			}
			tmp[y*n+x] = s / float64(2*r+1)
		}
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			s := 0.0
			for d := -r; d <= r; d++ {
				if yy := y + d; yy >= 0 && yy < n {
					s += tmp[yy*n+x]
				}
			}
			out[y*n+x] = s / float64(2*r+1)
		}
	}
	return out
}

// shrink averages each m x m block of px (straight alpha, nm across).
func shrink(px []color.RGBA, nm, m int) []color.RGBA {
	pm := make([]color.RGBA, len(px))
	for i, c := range px {
		pm[i] = icPM(c)
	}
	s := shrinkPM(pm, nm, m)
	for i, c := range s {
		s[i] = icUnPM(c)
	}
	return s
}

// shrinkPM averages each m x m block of premultiplied pixels.
func shrinkPM(px []color.RGBA, nm, m int) []color.RGBA {
	n := nm / m
	out := make([]color.RGBA, n*n)
	mm := uint32(m * m)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var r, g, b, a uint32
			for dy := 0; dy < m; dy++ {
				for dx := 0; dx < m; dx++ {
					c := px[(y*m+dy)*nm+x*m+dx]
					r, g, b, a = r+uint32(c.R), g+uint32(c.G), b+uint32(c.B), a+uint32(c.A)
				}
			}
			out[y*n+x] = color.RGBA{uint8((r + mm/2) / mm), uint8((g + mm/2) / mm), uint8((b + mm/2) / mm), uint8((a + mm/2) / mm)}
		}
	}
	return out
}

func icPM(c color.RGBA) color.RGBA {
	a := uint32(c.A)
	return color.RGBA{uint8(uint32(c.R) * a / 255), uint8(uint32(c.G) * a / 255), uint8(uint32(c.B) * a / 255), c.A}
}

func icUnPM(c color.RGBA) color.RGBA {
	if c.A == 0 {
		return color.RGBA{}
	}
	a := uint32(c.A)
	f := func(v uint8) uint8 { return uint8(min(255, (uint32(v)*255+a/2)/a)) }
	return color.RGBA{f(c.R), f(c.G), f(c.B), c.A}
}

// overPM is premultiplied a over premultiplied b.
func overPM(a, b color.RGBA) color.RGBA {
	k := 255 - uint32(a.A)
	f := func(x, y uint8) uint8 { return uint8(min(255, uint32(x)+(uint32(y)*k+127)/255)) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), f(a.A, b.A)}
}
