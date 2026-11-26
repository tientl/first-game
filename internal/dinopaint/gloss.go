package dinopaint

import (
	"image/color"
	"math"
)

// GlossSpec is the sheen of wet skin: highlights where the surface turns
// the light from the top left straight back to the eye, sharp as on
// something wet.
type GlossSpec struct {
	Region   SDF     // where the skin is wet (nil: everywhere)
	Strength float64 // how white the brightest of it goes (0 to 1)
	Sharp    float64 // the highlight's exponent: higher, smaller and sharper (0: 24)
	Tint     uint32  // the light's colour
	Steps    int     // how many tones it is drawn in, pixel art style (0: 3)
}

// Gloss lays a wet sheen over the hide painted so far (not over details):
// a few hard-edged tones of the light's colour on the forms facing the
// reflection of the light.
func (p *Painter) Gloss(g GlossSpec) {
	sharp := g.Sharp
	if sharp == 0 {
		sharp = 24
	}
	steps := g.Steps
	if steps == 0 {
		steps = 3
	}
	// The half-way vector between the light and the eye (straight out of
	// the screen).
	h := norm3(Light.x, Light.y, Light.z+1)
	tint := rgb(g.Tint)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] {
				continue
			}
			if g.Region != nil {
				if x, y := G(px, py); g.Region(x, y) > 0 {
					continue
				}
			}
			s := math.Pow(clamp01(p.Nrm[i].dot(h)), sharp)
			k := math.Floor(s * float64(steps+1))
			if k < 1 {
				continue
			}
			t := g.Strength * k / float64(steps)
			p.Out[i] = Mix(p.Out[i], tint, t)
		}
	}
}

// Trace paints, inside s, the guide's pixels for which is holds, traced
// smoothly three times as fine as the guide (not in 3x3 blocks), with
// paint; they are added to the silhouette and left without an outline.
func (p *Painter) Trace(s SDF, is func(g color.NRGBA) bool, paint func(px, py int) color.NRGBA) {
	m := NewField(p.GW, p.GH)
	for i, g := range p.Guide {
		if p.GSolid[i] && is(g) {
			m.V[i] = 1
		}
	}
	m = m.Blur(0.5)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			x, y := G(px, py)
			if s(x, y) > 0 || m.Sample(x, y) < 0.5 {
				continue
			}
			i := py*p.W + px
			p.In[i], p.Out[i], p.Pro[i] = true, paint(px, py), true
		}
	}
}

// Clear takes everything inside s out of the silhouette, but what a
// detail has painted.
func (p *Painter) Clear(s SDF) {
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			if x, y := G(px, py); s(x, y) <= 0 && !p.Pro[py*p.W+px] {
				p.In[py*p.W+px] = false
			}
		}
	}
}
