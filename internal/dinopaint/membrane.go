package dinopaint

import (
	"image/color"
	"math"
)

// The wing of a pterosaur: a thin skin stretched from the long fourth
// finger back to the body and the legs, stiffened by fine fibres
// (actinofibrils) that run out along it towards the tip. It is lit as a
// taut, nearly flat sheet; its fibres show as faint lines; where it is
// thinnest, along its free edge and at the tip, light comes through it,
// warm; and the bones of the arm and the finger read through as ridges
// with a lit side and a shadow beside them.
//
// Membrane is called in a species' Details after Plumage (which lights the
// furred body), and paints over what is there.

// MembraneSpec is one wing's membrane.
type MembraneSpec struct {
	Region SDF      // where the membrane is
	Mat    []string // only over these materials (nil: any)

	Fibres Coords  // the fibres run along s; they are Gap apart across it
	Gap    float64 // guide pixels (0: 1.2)
	Fibre  float64 // how dark a fibre is (0: 0.08)

	Flat   float64 // how far the body's rounded light is flattened, 0-1 (0: 0.6)
	Camber SDF     // the membrane bellies out, lighter, near this (the leading edge); nil: none

	Glow     float64  // how much light shines through near the free edge (0: 0.55)
	GlowW    float64  // how far in from the edge it reaches, guide pixels (0: 1.8)
	GlowRamp []uint32 // the colour of the light through it, dark to light
	Free     SDF      // where the edge is free (inside: the trailing edge, the tip); nil: all of it

	// Distal darkens it towards the tip: by this much from DistalFrom to
	// DistalTo along the fibres (s).
	Distal, DistalFrom, DistalTo float64

	Tone  float64 // how much of the drawing's broad light and dark is kept (0: none)
	Bones []Bone  // what reads through it, painted over it
	Seed  uint32
}

// Bone is a bone along the polyline Pts, R thick at its start and R1 at its
// end (0: R), lit as a tube in Ramp (dark to light).
type Bone struct {
	Pts   []float64
	R, R1 float64
	Ramp  []uint32
	Shade float64 // added to its light: below 0 for a bone under the membrane or far off
}

// Membrane paints a wing's membrane over the hide in its region.
func (p *Painter) Membrane(m MembraneSpec) {
	gap := m.Gap
	if gap == 0 {
		gap = 1.2
	}
	fib := m.Fibre
	if fib == 0 {
		fib = 0.08
	}
	flat := m.Flat
	if flat == 0 {
		flat = 0.6
	}
	glow := m.Glow
	if glow == 0 {
		glow = 0.55
	}
	gw := m.GlowW
	if gw == 0 {
		gw = 1.8
	}
	mats := map[int]bool{}
	for _, n := range m.Mat {
		mats[p.MatID(n)] = true
	}
	pick := func(r []uint32, v float64) color.NRGBA {
		return rgb(r[max(0, min(len(r)-1, int(math.Floor(clamp01(v)*float64(len(r))-1e-9))))])
	}
	flatV := LightOn(norm3(0, 0, 1))
	detail := p.guideDetail()
	var tone []float64
	if m.Tone != 0 {
		tone = p.toneField()
	}
	out := append([]color.NRGBA(nil), p.Out...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] || (len(mats) > 0 && !mats[p.Mat[i]]) {
				continue
			}
			x, y := G(px, py)
			if m.Region(x, y) > 0 {
				continue
			}
			edge := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if !p.Inside(px+d[0], py+d[1]) {
					edge = true
				}
			}
			v := lerp(p.Val[i], flatV, flat)
			if detail != nil {
				v += flat * p.Sp.GuideDetail * detail.Sample(x, y) // the drawing's mottling, kept
			}
			if tone != nil {
				v += m.Tone * tone[i]
			}
			if m.Camber != nil {
				// Bellied out behind the leading edge, then falling to the trailing one.
				d := max(0, m.Camber(x, y))
				v += 0.13*math.Exp(-d/2) - 0.07*smoothstep(2, 6, d)
			}
			if m.Fibres != nil && m.Distal != 0 {
				s, _ := m.Fibres(x, y)
				v -= m.Distal * smoothstep(m.DistalFrom, m.DistalTo, s)
			}
			// The fibres: faint dark lines with a lit lip beside them.
			if m.Fibres != nil {
				s, t := m.Fibres(x, y)
				f := t/gap - math.Floor(t/gap)
				var_ := 0.55 + 0.45*fbm(s*0.45, t*0.2, m.Seed+11, 2)
				switch {
				case f < 0.2:
					v -= fib * var_
				case f < 0.4:
					v += fib * 0.35 * var_
				}
			}
			// Thin where it is free: light comes through.
			g := 0.0
			if m.Free == nil || m.Free(x, y) <= 0 {
				e := -p.Sil.V[i] // in from the outline
				g = glow * math.Exp(-max(0, e-0.2)/gw)
			}
			c := p.Tone(p.Mat[i], v)
			if len(m.GlowRamp) > 0 && g > 0 {
				c = Mix(c, pick(m.GlowRamp, v+0.15), g)
			}
			if edge {
				// A thin edge, the light through it dimmed: no ink line.
				k := p.ToneIdx(p.Mat[i], v)
				e := p.Step(p.Mat[i], k-2)
				if len(m.GlowRamp) > 0 && g > 0 {
					e = Mix(e, pick(m.GlowRamp, v-0.15), g*0.8)
				}
				c = Mix(e, Ink, 0.4-0.25*g)
			}
			p.Val[i] = v
			out[i] = c
		}
	}
	p.Out = out
	for _, b := range m.Bones {
		p.bone(b)
	}
}

// bone paints a bone that reads through the membrane: a tube lit on the
// side towards the light, a shadow along its other side on the skin.
func (p *Painter) bone(b Bone) {
	r1 := b.R1
	if r1 == 0 {
		r1 = b.R
	}
	c := newCurve(b.Pts)
	total := c.run[len(c.run)-1]
	pick := func(v float64) color.NRGBA {
		r := b.Ramp
		return rgb(r[max(0, min(len(r)-1, int(math.Floor(clamp01(v)*float64(len(r))-1e-9))))])
	}
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for k := range c.x {
		x0, y0, x1, y1 = min(x0, c.x[k]), min(y0, c.y[k]), max(x1, c.x[k]), max(y1, c.y[k])
	}
	rr := max(b.R, r1) + 1
	for py := int((y0 - rr) * K); py <= int((y1+rr)*K); py++ {
		for px := int((x0 - rr) * K); px <= int((x1+rr)*K); px++ {
			i := p.Idx(px, py)
			if i < 0 || !p.In[i] {
				continue
			}
			x, y := G(px, py)
			s, t := c.at(x, y)
			if s < 0 || s > total {
				continue
			}
			u := s / total
			r := lerp(b.R, r1, u)
			// Which side of the bone faces the light.
			k := min(len(c.x)-2, max(0, int(u*float64(len(c.x)-1))))
			dx, dy := c.x[k+1]-c.x[k], c.y[k+1]-c.y[k]
			l := math.Hypot(dx, dy)
			if l == 0 {
				continue
			}
			nx, ny := -dy/l, dx/l // + t is this side
			lit := -(nx*light2x + ny*light2y)
			at := t / r // -1 to 1 across
			edge := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if !p.Inside(px+d[0], py+d[1]) {
					edge = true
				}
			}
			switch {
			case math.Abs(at) <= 1:
				// The tube: lit across it towards the light.
				v := 0.55 + 0.35*(-at*lit) - 0.25*at*at + b.Shade
				if edge {
					p.Out[i] = Mix(pick(v-0.35), Ink, 0.45)
				} else {
					p.Out[i] = pick(v)
				}
				p.Pro[i] = true
			case math.Abs(at) <= 1+0.55/r && at*lit > 0 && !edge && !p.Pro[i]:
				// Its shadow on the skin beside it.
				p.Out[i] = Mix(p.Out[i], Ink, 0.22)
			}
		}
	}
}
