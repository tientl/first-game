package dinopaint

import (
	"image/color"
	"math"
)

// FuzzSpec is a short fuzzy coat of fine filaments: Region is where the
// hide has it; the hairs lie along Flow (radians, screen angle: 0 to the
// right, π/2 down); tufts of them stand up to Len guide pixels off the
// outline on Dens of its pixels.
type FuzzSpec struct {
	Region SDF
	Flow   float64
	Len    float64 // (0: 0.8)
	Dens   float64 // (0: 0.45)
	Seed   uint32
}

// Fuzz lays a coat of fine filaments over the hide: short streaks of a
// tone lighter or darker running along the hair, and small tufts standing
// off the outline, so its edge is soft.
func (p *Painter) Fuzz(f FuzzSpec) {
	l, dens := f.Len, f.Dens
	if l == 0 {
		l = 0.8
	}
	if dens == 0 {
		dens = 0.45
	}
	fx, fy := math.Cos(f.Flow), math.Sin(f.Flow)
	inR := func(px, py int) bool {
		x, y := G(px, py)
		return f.Region == nil || f.Region(x, y) <= 0
	}
	// The streaks: tone along short runs of the hair's direction.
	val := append([]float64(nil), p.Val...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] || !inR(px, py) {
				continue
			}
			x, y := float64(px), float64(py)
			along, across := x*fx+y*fy, -x*fy+y*fx
			s := noise(along*0.35, across*1.3, f.Seed) - 0.5
			val[i] += 0.16 * s
		}
	}
	p.Val = val
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if p.In[i] && !p.Pro[i] && inR(px, py) {
				p.Out[i] = p.Tone(p.Mat[i], p.Val[i])
			}
		}
	}
	// The tufts: from the outline outwards, bending with the hair.
	type strand struct {
		x, y, dx, dy float64
		n            int
		c            color.NRGBA
	}
	var strands []strand
	for py := 1; py < p.H-1; py++ {
		for px := 1; px < p.W-1; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] || !inR(px, py) || hash(px, py, f.Seed+3) > dens {
				continue
			}
			edge := !p.In[i-1] || !p.In[i+1] || !p.In[i-p.W] || !p.In[i+p.W]
			if !edge {
				continue
			}
			gx := p.Sil.At(px+1, py) - p.Sil.At(px-1, py)
			gy := p.Sil.At(px, py+1) - p.Sil.At(px, py-1)
			gl := math.Hypot(gx, gy)
			if gl == 0 {
				continue
			}
			nx, ny := gx/gl, gy/gl // outwards
			dx, dy := nx+0.8*fx, ny+0.8*fy
			dl := math.Hypot(dx, dy)
			n := 1 + int(hash(px, py, f.Seed+5)*l*K)
			strands = append(strands, strand{float64(px) + 0.5, float64(py) + 0.5, dx / dl, dy / dl, n, Mix(p.Out[i], Ink, 0.3)})
		}
	}
	for _, s := range strands {
		for k := 1; k <= s.n; k++ {
			q := p.Idx(int(math.Floor(s.x+s.dx*float64(k))), int(math.Floor(s.y+s.dy*float64(k))))
			if q < 0 || p.In[q] {
				continue
			}
			p.In[q], p.Out[q], p.Pro[q] = true, s.c, true
		}
	}
}
