package dinopaint

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// K is how much finer the painting is than its guide (HDDensity).
const K = 3

type vec3 struct{ x, y, z float64 }

func norm3(x, y, z float64) vec3 {
	l := math.Sqrt(x*x + y*y + z*z)
	return vec3{x / l, y / l, z / l}
}

func (a vec3) dot(b vec3) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }

// Light comes from the top left, a little towards the viewer.
var Light = norm3(-0.45, -0.8, 0.45)

// light2 is the light's direction on the screen.
var light2x, light2y = func() (float64, float64) {
	l := math.Hypot(Light.x, Light.y)
	return Light.x / l, Light.y / l
}()

// Ink is the darkest colour an outline goes to: a deep warm brown.
var Ink = rgb(0x1a1418)

// Painter holds a painting in progress. Positions passed to its methods
// are in guide pixels unless they say otherwise; P converts.
type Painter struct {
	Sp     *Species
	GW, GH int
	Guide  []color.NRGBA
	GSolid []bool

	W, H int
	In   []bool    // the silhouette
	Sil  *Field    // signed distance to its edge, in guide pixels
	Mat  []int     // the material of each pixel
	Part []int     // the part of each pixel, -1 outside
	Nrm  []vec3    // the surface's normal
	Hgt  []*Field  // each part's height, in guide pixels
	Val  []float64 // the light on each pixel, 0 (deep shadow) to 1 (highlight)
	Out  []color.NRGBA
	Pro  []bool // pixels a detail has painted, which the outline leaves alone

	mats  map[string]int
	parts map[string]int
	cells *scaleSet
	knobs []knob
}

// New starts painting species sp over its guide drawing.
func New(sp *Species, guide image.Image) *Painter {
	b := guide.Bounds()
	p := &Painter{Sp: sp, GW: b.Dx(), GH: b.Dy()}
	p.Guide = make([]color.NRGBA, p.GW*p.GH)
	p.GSolid = make([]bool, p.GW*p.GH)
	for y := 0; y < p.GH; y++ {
		for x := 0; x < p.GW; x++ {
			c := color.NRGBAModel.Convert(guide.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			p.Guide[y*p.GW+x] = c
			p.GSolid[y*p.GW+x] = c.A >= 128
		}
	}
	p.W, p.H = p.GW*K, p.GH*K
	n := p.W * p.H
	p.In, p.Mat, p.Part = make([]bool, n), make([]int, n), make([]int, n)
	p.Nrm, p.Val, p.Out, p.Pro = make([]vec3, n), make([]float64, n), make([]color.NRGBA, n), make([]bool, n)
	p.mats, p.parts = map[string]int{}, map[string]int{}
	for i, m := range sp.Materials {
		p.mats[m.Name] = i
	}
	for i, pt := range sp.Parts {
		p.parts[pt.Name] = i
	}
	return p
}

// Paint paints species sp over its guide and returns the painting.
func Paint(sp *Species, guide image.Image) (*image.NRGBA, *Painter) {
	p := New(sp, guide)
	p.silhouette()
	p.materials()
	p.recolor()
	p.assignParts()
	p.form()
	p.shade()
	p.skin()
	p.colour()
	p.outline()
	if sp.Details != nil {
		sp.Details(p)
	}
	return p.Image(), p
}

// G is the guide position of the centre of image pixel (px, py).
func G(px, py int) (float64, float64) {
	return (float64(px) + 0.5) / K, (float64(py) + 0.5) / K
}

// Idx is the index of image pixel (x, y), or -1 off the picture.
func (p *Painter) Idx(x, y int) int {
	if x < 0 || y < 0 || x >= p.W || y >= p.H {
		return -1
	}
	return y*p.W + x
}

// Inside reports whether image pixel (x, y) is part of the animal.
func (p *Painter) Inside(x, y int) bool { i := p.Idx(x, y); return i >= 0 && p.In[i] }

// MatID is the index of a material by name, or -1.
func (p *Painter) MatID(name string) int {
	if i, ok := p.mats[name]; ok {
		return i
	}
	return -1
}

// PartID is the index of a part by name, or -1.
func (p *Painter) PartID(name string) int {
	if i, ok := p.parts[name]; ok {
		return i
	}
	return -1
}

// Image is the painting as it stands.
func (p *Painter) Image() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, p.W, p.H))
	for i, in := range p.In {
		if in {
			c := p.Out[i]
			c.A = 255
			img.Pix[4*i], img.Pix[4*i+1], img.Pix[4*i+2], img.Pix[4*i+3] = c.R, c.G, c.B, 255
		}
	}
	return img
}

func rgb(h uint32) color.NRGBA { return color.NRGBA{uint8(h >> 16), uint8(h >> 8), uint8(h), 255} }

func lum(c color.NRGBA) float64 { return 0.3*float64(c.R) + 0.59*float64(c.G) + 0.11*float64(c.B) }

// Mix is a t of the way from a to b.
func Mix(a, b color.NRGBA, t float64) color.NRGBA {
	t = clamp01(t)
	m := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.NRGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// silhouette traces the guide's outline three times as fine: smooth
// curves instead of steps, thin parts kept.
func (p *Painter) silhouette() {
	m := NewField(p.GW, p.GH)
	for i, s := range p.GSolid {
		if s {
			m.V[i] = 1
		}
	}
	m = m.Blur(0.95)
	// A line one pixel thin in the guide would blur away: it keeps its
	// middle.
	solid := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < p.GW && y < p.GH && p.GSolid[y*p.GW+x]
	}
	thin := func(px, py int) bool {
		gx, gy := px/K, py/K
		if !solid(gx, gy) {
			return false
		}
		narrowX := !solid(gx-1, gy) && !solid(gx+1, gy)
		narrowY := !solid(gx, gy-1) && !solid(gx, gy+1)
		return (narrowX && px%K == K/2) || (narrowY && py%K == K/2)
	}
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			x, y := G(px, py)
			in := m.Sample(x, y) >= 0.5 || thin(px, py)
			for _, a := range p.Sp.Add {
				in = in || a(x, y) <= 0
			}
			for _, r := range p.Sp.Remove {
				in = in && r(x, y) > 0
			}
			p.In[py*p.W+px] = in
		}
	}
	// No lone pixels or one-pixel notches.
	for range 3 {
		for py := 0; py < p.H; py++ {
			for px := 0; px < p.W; px++ {
				n := 0
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if p.Inside(px+d[0], py+d[1]) {
						n++
					}
				}
				i := py*p.W + px
				if p.In[i] && n <= 1 {
					p.In[i] = false
				} else if !p.In[i] && n >= 3 {
					p.In[i] = true
				}
			}
		}
	}
	p.placeKnobs()
	p.Sil = sdfOf(p.In, p.W, p.H)
	for i := range p.Sil.V {
		p.Sil.V[i] /= K
	}
}

// labish is a colour as lightness and two hue axes, for matching colours
// by eye rather than by numbers.
func labish(c color.NRGBA) (float64, float64, float64) {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	return 0.3*r + 0.59*g + 0.11*b, r - g, (r+g)/2 - b
}

// materials gives every pixel its material: the guide's pixels matched to
// the seed colours, the outline and the edges given the material beside
// them, and the regions traced smoothly three times as fine.
func (p *Painter) materials() {
	sp := p.Sp
	nm := len(sp.Materials)
	cls := make([]int, p.GW*p.GH)
	type seed struct {
		l, a, b float64
		m       int
	}
	var seeds []seed
	for i, m := range sp.Materials {
		for _, s := range m.Seeds {
			l, a, b := labish(rgb(s))
			seeds = append(seeds, seed{l, a, b, i})
		}
	}
	var queue []int
	for i, c := range p.Guide {
		cls[i] = -1
		if !p.GSolid[i] || (sp.Ink > 0 && lum(c) < sp.Ink) || len(seeds) == 0 {
			continue
		}
		l, a, b := labish(c)
		best, bd := 0, math.Inf(1)
		for _, s := range seeds {
			d := (l-s.l)*(l-s.l) + 2.5*(a-s.a)*(a-s.a) + 2.5*(b-s.b)*(b-s.b)
			if d < bd {
				best, bd = s.m, d
			}
		}
		cls[i] = best
		queue = append(queue, i)
	}
	// The outline and the empty pixels take the material next to them.
	for len(queue) > 0 {
		var next []int
		for _, i := range queue {
			x, y := i%p.GW, i/p.GW
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= p.GW || ny >= p.GH {
					continue
				}
				if j := ny*p.GW + nx; cls[j] < 0 {
					cls[j] = cls[i]
					next = append(next, j)
				}
			}
		}
		queue = next
	}
	masks := make([]*Field, nm)
	for m := range masks {
		f := NewField(p.GW, p.GH)
		for i, c := range cls {
			if c == m {
				f.V[i] = 1
			}
		}
		sm := sp.Materials[m].Smooth
		if sm == 0 {
			sm = 0.45
		}
		masks[m] = f.Blur(sm)
	}
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			x, y := G(px, py)
			best, bv := 0, math.Inf(-1)
			for m := range masks {
				v := masks[m].Sample(x, y) + 0.12*(fbm(x*0.8, y*0.8, uint32(31+m), 2)-0.5) + sp.Materials[m].Bias
				if v > bv {
					best, bv = m, v
				}
			}
			p.Mat[py*p.W+px] = best
		}
	}
	// Regions painted on by the description.
	for m, mat := range sp.Materials {
		if mat.Region == nil {
			continue
		}
		var only []int
		for _, o := range mat.Only {
			only = append(only, p.MatID(o))
		}
		for py := 0; py < p.H; py++ {
			for px := 0; px < p.W; px++ {
				x, y := G(px, py)
				if mat.Region(x, y) > 0 {
					continue
				}
				i := py*p.W + px
				ok := len(only) == 0
				for _, o := range only {
					ok = ok || p.Mat[i] == o
				}
				if ok {
					p.Mat[i] = m
				}
			}
		}
	}
}

// recolor applies the description's changes of material.
func (p *Painter) recolor() {
	for _, rc := range p.Sp.Recolor {
		to := p.MatID(rc.To)
		var from []int
		for _, f := range rc.From {
			from = append(from, p.MatID(f))
		}
		for py := 0; py < p.H; py++ {
			for px := 0; px < p.W; px++ {
				i := py*p.W + px
				if x, y := G(px, py); rc.Region(x, y) <= 0 {
					for _, f := range from {
						if p.Mat[i] == f {
							p.Mat[i] = to
						}
					}
				}
			}
		}
	}
}

// assignParts gives every pixel the frontmost part that holds it.
func (p *Painter) assignParts() {
	parts := p.Sp.Parts
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			p.Part[i] = -1
			if !p.In[i] {
				continue
			}
			x, y := G(px, py)
			best, bd := 0, math.Inf(1)
			for j := len(parts) - 1; j >= 0; j-- {
				d := parts[j].Shape(x, y)
				if d <= 0 {
					best = j
					break
				}
				if d < bd {
					best, bd = j, d
				}
			}
			p.Part[i] = best
			if m := parts[best].Mat; m != "" {
				p.Mat[i] = p.MatID(m)
			}
		}
	}
}

// partDepth is how far inside part j the point (x, y) is, in guide pixels:
// inside its shape and the silhouette both.
func (p *Painter) partDepth(j int, px, py int) float64 {
	x, y := G(px, py)
	return -max(p.Sp.Parts[j].Shape(x, y), p.Sil.V[py*p.W+px])
}

// form models each part as a mass rounded over from its edge with its
// muscles bulging on it, and gives every pixel the normal of its surface.
func (p *Painter) form() {
	parts := p.Sp.Parts
	p.Hgt = make([]*Field, len(parts))
	raw := make([]*Field, len(parts))
	for j, pt := range parts {
		R := pt.Round
		if R == 0 {
			R = 5
		}
		h := NewField(p.W, p.H)
		for py := 0; py < p.H; py++ {
			for px := 0; px < p.W; px++ {
				d := p.partDepth(j, px, py)
				if d <= 0 {
					continue
				}
				x, y := G(px, py)
				R := R
				if pt.RoundAt != nil {
					R = pt.RoundAt(x, y)
				}
				t := 1 - min(d/R, 1)
				v := R * math.Sqrt(1-t*t)
				for _, m := range pt.Muscles {
					s, c := math.Sincos(-m.Rot)
					dx, dy := x-m.X, y-m.Y
					u, w := (dx*c-dy*s)/m.RX, (dx*s+dy*c)/m.RY
					if q := u*u + w*w; q < 1 {
						v += m.Amp * (1 - q) * (1 - q) * smoothstep(0, 1.2, d)
					}
				}
				h.V[py*p.W+px] = v
			}
		}
		if o := p.PartID(pt.Onto); o >= 0 && o < j {
			for py := 0; py < p.H; py++ {
				for px := 0; px < p.W; px++ {
					x, y := G(px, py)
					w := 1.0
					if pt.Seam != nil {
						w = smoothstep(-1.5, 1.5, pt.Seam(x, y))
					}
					if i := py*p.W + px; w > 0 {
						h.V[i] = lerp(h.V[i], smax(h.V[i], raw[o].V[i], 1.5), w)
					}
				}
			}
		}
		raw[j] = h
		soft := pt.Soft
		if soft == 0 {
			soft = 0.8
		}
		p.Hgt[j] = h.Blur(soft * K)
	}
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] {
				continue
			}
			h := p.Hgt[p.Part[i]]
			hx := (h.At(px+1, py) - h.At(px-1, py)) / 2 * K
			hy := (h.At(px, py+1) - h.At(px, py-1)) / 2 * K
			p.Nrm[i] = norm3(-hx, -hy, 1)
		}
	}
}

// smax is the larger of a and b, the corner between them rounded k wide.
func smax(a, b, k float64) float64 {
	h := max(k-math.Abs(a-b), 0) / k
	return max(a, b) + h*h*k/4
}

// LightOn is the light on a surface facing n: 0 in deep shadow, 1 where it
// faces the light.
func LightOn(n vec3) float64 { return 0.18 + 0.82*clamp01(0.38+0.62*n.dot(Light)) }

// shade lights every pixel: the light on its surface, the shadow cast by
// what is in front of it, the far side of the body darker, and a rim of
// light along the edges turned away.
func (p *Painter) shade() {
	parts := p.Sp.Parts
	detail := p.guideDetail()
	// The shadow each part gets from the parts in front of it: their shape
	// moved a little down and right, away from the light.
	occl := make([][]float64, len(parts))
	sh := int(math.Round(0.9 * K))
	for j := range parts {
		m := make([]bool, p.W*p.H)
		any := false
		for py := 0; py < p.H; py++ {
			for px := 0; px < p.W; px++ {
				sx, sy := px-sh, py-sh
				if q := p.Idx(sx, sy); q >= 0 && p.Part[q] > j {
					if x, y := G(sx, sy); parts[p.Part[q]].creased(x, y) {
						m[py*p.W+px] = true
						any = true
					}
				}
			}
		}
		if any {
			occl[j] = edt(m, p.W, p.H)
		}
	}
	for i, in := range p.In {
		if !in {
			continue
		}
		n := p.Nrm[i]
		v := LightOn(n)
		j := p.Part[i]
		if occl[j] != nil {
			d := occl[j][i] / K
			v -= 0.32 * math.Exp(-d/1.4)
		}
		rim := smoothstep(0.45, 0.95, 1-n.z) * clamp01(n.x*0.55+n.y*0.83)
		v += 0.2 * rim
		if detail != nil {
			x, y := G(i%p.W, i/p.W)
			v += p.Sp.GuideDetail * detail.Sample(x, y)
		}
		if parts[j].Far {
			v = v*0.82 - 0.05
		}
		p.Val[i] = v
	}
}

// colour turns the light into the colours of each pixel's material.
func (p *Painter) colour() {
	step := make([]int, len(p.In))
	for i, in := range p.In {
		if in {
			step[i] = p.ToneIdx(p.Mat[i], p.Val[i])
		}
	}
	// A lone pixel of a tone, with none of its neighbours sharing it, takes
	// its neighbours' tone instead: clusters, not noise.
	clean := append([]int(nil), step...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] {
				continue
			}
			var count [16]int
			alone, n := true, 0
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				q := p.Idx(px+d[0], py+d[1])
				if q < 0 || !p.In[q] || p.Mat[q] != p.Mat[i] {
					continue
				}
				n++
				if step[q] == step[i] {
					alone = false
				}
				count[step[q]&15]++
			}
			if !alone || n < 3 {
				continue
			}
			best := step[i]
			for s, c := range count {
				if c > count[best&15] {
					best = s
				}
			}
			clean[i] = best
		}
	}
	for i, in := range p.In {
		if in {
			p.Out[i] = p.Step(p.Mat[i], clean[i])
		}
	}
}

// Tone is material m's colour at light v.
func (p *Painter) Tone(m int, v float64) color.NRGBA {
	r := p.Sp.Materials[m].Ramp
	k := int(math.Floor(clamp01(v)*float64(len(r)) - 1e-9))
	return rgb(r[max(0, min(len(r)-1, k))])
}

// ToneIdx is the step of material m's ramp at light v.
func (p *Painter) ToneIdx(m int, v float64) int {
	n := len(p.Sp.Materials[m].Ramp)
	return max(0, min(n-1, int(math.Floor(clamp01(v)*float64(n)-1e-9))))
}

// Step is material m's ramp at step k, clamped.
func (p *Painter) Step(m, k int) color.NRGBA {
	r := p.Sp.Materials[m].Ramp
	return rgb(r[max(0, min(len(r)-1, k))])
}

// outline draws the edges: along the silhouette a line of the hide's own
// darkest colours, lighter where the edge faces the light and darkest below
// and behind; where a part lies over another, a softer line along its edge.
func (p *Painter) outline() {
	out := append([]color.NRGBA(nil), p.Out...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] {
				continue
			}
			edge, behind := false, false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				q := p.Idx(px+d[0], py+d[1])
				if q < 0 || !p.In[q] {
					edge = true
				} else if p.Part[q] < p.Part[i] {
					x, y := G(px, py)
					behind = behind || p.Sp.Parts[p.Part[i]].creased(x, y)
				}
			}
			m := p.Mat[i]
			k := p.ToneIdx(m, p.Val[i])
			switch {
			case edge:
				gx := p.Sil.At(px+1, py) - p.Sil.At(px-1, py)
				gy := p.Sil.At(px, py+1) - p.Sil.At(px, py-1)
				l := math.Hypot(gx, gy)
				face := 0.0
				if l > 0 {
					face = -(gx*light2x + gy*light2y) / l // 1: the edge faces the light
				}
				lit := smoothstep(-0.2, 0.8, face)
				step := k - 5 + int(math.Round(2*lit))
				c := p.Step(m, max(0, step))
				out[i] = Mix(c, Ink, 0.55-0.4*lit)
			case behind:
				n := p.Nrm[i]
				l := math.Hypot(n.x, n.y)
				face := 0.0
				if l > 0 {
					face = -(n.x*light2x + n.y*light2y) / l
				}
				lit := smoothstep(-0.3, 0.7, face)
				c := p.Step(m, k-3+int(math.Round(2*lit)))
				out[i] = Mix(c, Ink, 0.3-0.3*lit)
			}
		}
	}
	p.Out = out
}

// Report is a line on what was painted.
func (p *Painter) Report() string {
	n := 0
	for _, in := range p.In {
		if in {
			n++
		}
	}
	return fmt.Sprintf("%s: %dx%d, %d pixels", p.Sp.Name, p.W, p.H, n)
}

// guideDetail is how much lighter or darker than its surroundings each
// pixel of the guide is, in steps of light, or nil when the description
// takes none of it.
func (p *Painter) guideDetail() *Field {
	if p.Sp.GuideDetail == 0 {
		return nil
	}
	l := NewField(p.GW, p.GH)
	for i, c := range p.Guide {
		l.V[i] = lum(c)
	}
	// Outside the drawing, the nearest of it, so its edge is not a mark.
	for range 3 {
		for i := range l.V {
			if p.GSolid[i] {
				continue
			}
			x, y := i%p.GW, i/p.GW
			s, n := 0.0, 0
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if nx, ny := x+d[0], y+d[1]; nx >= 0 && ny >= 0 && nx < p.GW && ny < p.GH && l.V[ny*p.GW+nx] > 0 {
					s += l.V[ny*p.GW+nx]
					n++
				}
			}
			if n > 0 {
				l.V[i] = s / float64(n)
			}
		}
	}
	mean := l.Blur(1.3)
	d := NewField(p.GW, p.GH)
	for i := range d.V {
		d.V[i] = clamp((l.V[i]-mean.V[i])/90, -0.3, 0.3)
	}
	return d
}
