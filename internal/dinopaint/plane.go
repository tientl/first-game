package dinopaint

import "math"

// Planes and grain: a crest or a casque is a thin plate seen side on, not a
// rounded mass. Plane lights the pixels of a material inside a region as one
// flat face tilted towards (or away from) the light, with a bright rim along
// the edge that faces the light and a dark one along the other, and fine
// growth lines across it.

// PlaneSpec is a flat plate: where it is (Region, and only over material
// Mat), how it faces (Tilt: -1 away from the light, +1 towards it), how
// its growth lines run (Angle, radians, Gap apart in guide pixels; Lines 0:
// none) and how much the light falls off along it, from (FromX, FromY).
type PlaneSpec struct {
	Region       SDF
	Mat          []string
	Tilt         float64
	Angle, Gap   float64
	Lines        float64
	FromX, FromY float64
	Fall         float64 // the light lost per guide pixel from (FromX, FromY)
	Rim          float64 // how bright the lit rim is (0: 0.2)
	Seed         uint32
}

// Plane relights a plate: a face of one light, its rims, its growth lines.
func (p *Painter) Plane(s PlaneSpec) {
	rim := s.Rim
	if rim == 0 {
		rim = 0.2
	}
	mats := map[int]bool{}
	for _, m := range s.Mat {
		mats[p.MatID(m)] = true
	}
	ca, sa := math.Cos(s.Angle), math.Sin(s.Angle)
	base := LightOn(norm3(0, 0, 1)) + 0.12*s.Tilt
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] || (len(mats) > 0 && !mats[p.Mat[i]]) {
				continue
			}
			x, y := G(px, py)
			if d := -s.Region(x, y); d < 0 {
				continue
			}
			edge := false
			for _, o := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if !p.Inside(px+o[0], py+o[1]) {
					edge = true
				}
			}
			if edge {
				continue // the outline stays
			}
			// The edge: which way it faces, from the silhouette's slope.
			gx := p.Sil.At(px+1, py) - p.Sil.At(px-1, py)
			gy := p.Sil.At(px, py+1) - p.Sil.At(px, py-1)
			face := 0.0
			if l := math.Hypot(gx, gy); l > 0 {
				face = -(gx*light2x + gy*light2y) / l
			}
			sil := -p.Sil.V[i] // how far in from the outline
			v := base - s.Fall*math.Hypot(x-s.FromX, y-s.FromY)
			v = lerp(p.Val[i], v, smoothstep(0, 0.8, sil)) // the plate's own edge keeps the body's light
			if sil < 0.9 && sil > 0.3 {
				v += rim * face // the bevel of its edge
			}
			if s.Lines > 0 && s.Gap > 0 {
				t := (x*-sa + y*ca) / s.Gap
				t += 0.35 * (fbm(x*0.6, y*0.6, s.Seed+5, 2) - 0.5)
				if f := t - math.Floor(t); f < 0.22 {
					v -= s.Lines
				} else if f < 0.4 {
					v += s.Lines * 0.4
				}
			}
			p.Val[i] = v
			p.Out[i] = p.Tone(p.Mat[i], v)
		}
	}
}
