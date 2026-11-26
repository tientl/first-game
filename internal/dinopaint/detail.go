package dinopaint

import (
	"image/color"
	"math"
)

// The details drawn last, over the lit hide: eyes, nostrils, teeth, lips,
// claws and horns. Positions and sizes are in guide pixels.

// Fill paints every pixel inside shape s with f(px, py, d), d its depth
// inside in guide pixels; grow adds the pixels to the silhouette.
func (p *Painter) Fill(s SDF, grow bool, f func(px, py int, d float64) color.NRGBA) {
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			x, y := G(px, py)
			d := -s(x, y)
			if d < 0 {
				continue
			}
			i := py*p.W + px
			if !p.In[i] && !grow {
				continue
			}
			if c := f(px, py, d); c.A != 0 {
				p.In[i], p.Out[i], p.Pro[i] = true, c, true
			}
		}
	}
}

// Shift moves the light of the pixels inside s by dv, and recolours them.
func (p *Painter) Shift(s SDF, dv float64) {
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] {
				continue
			}
			x, y := G(px, py)
			if d := -s(x, y); d > 0 {
				p.Val[i] += dv * smoothstep(0, 0.6, d)
				p.Out[i] = p.Tone(p.Mat[i], p.Val[i])
			}
		}
	}
}

// Local is the hide's material and light at guide point (x, y).
func (p *Painter) Local(x, y float64) (int, float64) {
	i := p.Idx(int(x*K), int(y*K))
	if i < 0 {
		return 0, 0.5
	}
	return p.Mat[i], p.Val[i]
}

// EyeSpec is an eye: its middle and radius, the iris's colours from dark
// to light, a slit pupil or a round one, and how much of it the upper lid
// covers (0 to 0.5).
type EyeSpec struct {
	X, Y, R float64
	Iris    []uint32
	Slit    bool
	Lid     float64
	Squint  float64 // how much wider than tall (0: 1.25)
}

// Eye paints an eye in a dark socket, lit from below through the lens,
// with the lid's shadow over it, a pupil and a glint.
func (p *Painter) Eye(e EyeSpec) {
	sq := e.Squint
	if sq == 0 {
		sq = 1.25
	}
	// Worked in image pixels: an eye is only a few of them across.
	cx, cy := e.X*K, e.Y*K
	rx, ry := e.R*sq*K, e.R*K
	m, v := p.Local(e.X, e.Y-e.R*1.8)
	socket := Mix(p.Step(m, 0), Ink, 0.7)
	shadow := Mix(p.Tone(m, v-0.45), Ink, 0.35)
	iris := func(t float64) color.NRGBA {
		r := e.Iris
		return rgb(r[max(0, min(len(r)-1, int(clamp01(t)*float64(len(r)))))])
	}
	lid := func(x float64) float64 { // the edge of the upper lid, drooping at the front
		u := (x - cx) / rx
		return cy - ry + 2*ry*e.Lid + 0.6*u*u - 0.5*u
	}
	in := func(x, y float64) bool {
		u, w := (x-cx)/rx, (y-cy)/ry
		return u*u+w*w <= 1 && y >= lid(x)
	}
	for py := int(cy - ry - 3); py <= int(cy+ry+2); py++ {
		for px := int(cx - rx - 3); px <= int(cx+rx+3); px++ {
			i := p.Idx(px, py)
			if i < 0 || !p.In[i] {
				continue
			}
			x, y := float64(px)+0.5, float64(py)+0.5
			u, w := (x-cx)/rx, (y-cy)/ry
			switch {
			case in(x, y):
				t := 0.3 + 0.75*clamp01((y-lid(x))/(cy+ry-lid(x))) + 0.15*u
				if (u*u+w*w > 0.62 && w > -0.2) || y < lid(x)+1 {
					t -= 0.3 // a dark rim, and the lid's shadow along the top
				}
				switch {
				case e.Slit && math.Abs(x-cx-0.3) < 0.8 && math.Abs(w) < 0.85:
					p.Out[i] = rgb(0x140b08)
				case !e.Slit && math.Hypot(u, w) < 0.45:
					p.Out[i] = rgb(0x140b08)
				default:
					p.Out[i] = iris(t)
				}
			case in(x-1, y) || in(x+1, y) || in(x, y-1) || in(x, y+1) || in(x-1, y+1) || in(x+1, y+1):
				p.Out[i] = socket // the socket's dark edge
			case y < cy && math.Abs(u) < 1.15 && y > lid(x)-1.6:
				p.Out[i] = shadow // the lid, in the brow's shadow
			default:
				continue
			}
			p.Pro[i] = true
		}
	}
	// The glint: where the light from the top left meets the lens.
	gx := int(math.Floor(cx - rx*0.45))
	gy := int(math.Floor(lid(cx-rx*0.45) + 1.2))
	for _, d := range [][3]int{{0, 0, 255}, {1, 0, 150}, {0, 1, 110}} {
		if i := p.Idx(gx+d[0], gy+d[1]); i >= 0 && p.Pro[i] {
			p.Out[i] = Mix(p.Out[i], rgb(0xfffbe8), float64(d[2])/255)
		}
	}
}

// Nostril paints a nostril: a dark hollow under a lit rim.
func (p *Painter) Nostril(x, y, rx, ry, rot float64) {
	m, v := p.Local(x, y)
	hollow := Mix(p.Step(m, 0), Ink, 0.5)
	rim := p.Tone(m, v+0.22)
	in := Ellipse(x, y, rx, ry, rot)
	p.Fill(Ellipse(x, y, rx+0.4, ry+0.35, rot), false, func(px, py int, d float64) color.NRGBA {
		gx, gy := G(px, py)
		switch {
		case in(gx, gy) <= 0 && gy > y+ry*0.2 && gx > x:
			return Mix(hollow, rim, 0.25) // light down the far wall
		case in(gx, gy) <= 0:
			return hollow
		case gy > y:
			return rim // the lit lip below
		}
		return p.Tone(m, v-0.15)
	})
}

// TeethSpec is a row of teeth along a gum line from (X0, Y0) at the back
// of the mouth to (X1, Y1) at the front; they hang down to no lower than
// Bottom (or stand up to no higher, with Up), the longest Len long.
type TeethSpec struct {
	X0, Y0, X1, Y1 float64
	Bottom         float64
	N              int
	Len            float64
	Up             bool
	Ramp           []uint32 // the tooth: shadow to light
	Gap            uint32   // between the teeth: the dark of the mouth
	Seed           uint32
}

// Teeth paints the gap of the mouth along the gum line and a row of
// recurved, pointed teeth in it, lit down their front edge.
func (p *Painter) Teeth(t TeethSpec) {
	gum := func(x float64) float64 { return lerp(t.Y0, t.Y1, clamp01((x-t.X0)/(t.X1-t.X0))) }
	gap := rgb(t.Gap)
	sp := (t.X1 - t.X0) / float64(t.N)
	// The dark of the mouth between the lips.
	for px := int(t.X0 * K); px < int(t.X1*K); px++ {
		x, _ := G(px, 0)
		g := gum(x)
		for py := int(math.Floor(g * K)); float64(py) < t.Bottom*K; py++ {
			if i := p.Idx(px, py); i >= 0 && p.In[i] {
				_, y := G(px, py)
				p.Out[i] = Mix(gap, Ink, 0.4*clamp01((y-g)/(t.Bottom-g)))
				p.Pro[i] = true
			}
		}
	}
	tooth := func(r float64) color.NRGBA {
		return rgb(t.Ramp[max(0, min(len(t.Ramp)-1, int(r*float64(len(t.Ramp)))))])
	}
	for k := 0; k < t.N; k++ {
		h := hash(k, 7, t.Seed)
		cx := t.X0 + (float64(k)+0.5)*sp + (h-0.5)*sp*0.25
		u := (cx - t.X0) / (t.X1 - t.X0)
		// The biggest teeth a third of the way back from the snout; the front ones small.
		prof := 0.7 + 0.3*math.Sin(math.Pi*clamp01(1.25*(1-u)))
		if u > 0.9 {
			prof *= 0.75
		}
		l := t.Len * prof * (0.7 + 0.45*hash(k, 9, t.Seed))
		w := sp * (0.8 + 0.12*h)
		g := gum(cx)
		if t.Up {
			l = min(l, g-t.Bottom-0.05)
		} else {
			l = min(l, t.Bottom-g-0.05)
		}
		for py := int(math.Floor((g - 0.2) * K)); py <= int(math.Ceil((g+l)*K)); py++ {
			for px := int((cx - w) * K); px <= int((cx+w)*K); px++ {
				i := p.Idx(px, py)
				if i < 0 || !p.In[i] {
					continue
				}
				x, y := G(px, py)
				s := (y - g) / l // 0 at the gum, 1 at the tip
				if t.Up {
					s = (g - y) / l
				}
				if s < 0 || s > 1 || (!t.Up && y > t.Bottom) || (t.Up && y < t.Bottom) {
					continue
				}
				// Recurved: the tip leans back, towards the throat.
				mid := cx - 0.22*sp*math.Pow(s, 1.5)
				half := w / 2 * math.Pow(1-s, 0.9)
				dx := (x - mid) / max(half, 0.01)
				if math.Abs(dx) > 1 {
					continue
				}
				r := 0.8 - 0.4*dx // lit down the front edge
				if s < 0.2 {
					r -= 0.3 * (1 - s/0.2) // in the gum's shadow
				}
				if dx > 0.5 {
					r -= 0.17
				}
				p.Out[i] = tooth(clamp01(r))
				p.Pro[i] = true
			}
		}
	}
}

// Lip draws the edge of the lip along a polyline: a dark line with the
// lip's lit edge above it and a row of small lip scales.
func (p *Painter) Lip(pts []float64, seed uint32) {
	for k := 0; k+3 < len(pts); k += 2 {
		ax, ay, bx, by := pts[k], pts[k+1], pts[k+2], pts[k+3]
		for px := int(ax * K); px < int(bx*K); px++ {
			x, _ := G(px, 0)
			y := lerp(ay, by, (x-ax)/(bx-ax))
			row := int(math.Floor(y * K))
			if i := p.Idx(px, row-1); i >= 0 && p.In[i] {
				m := p.Mat[i]
				v := p.Val[i]
				p.Out[i] = Mix(p.Tone(m, v-0.35), Ink, 0.35)
				p.Pro[i] = true
				// The lip scales: little plates in a row, each lit at its top.
				for dy := 2; dy <= 4; dy++ {
					j := p.Idx(px, row-dy)
					if j < 0 || !p.In[j] || p.Pro[j] {
						continue
					}
					cellx := int(math.Floor((x + 0.3*hash(int(y), 1, seed)) / 1.25))
					edge := math.Mod(x+0.3*hash(int(y), 1, seed), 1.25) < 1.0/K
					dv := 0.06
					switch {
					case edge:
						dv = -0.16
					case dy == 4:
						dv = 0.12
					}
					if hash(cellx, 2, seed) < 0.3 {
						dv -= 0.05
					}
					p.Out[j] = p.Tone(p.Mat[j], p.Val[j]+dv)
					p.Pro[j] = true
				}
			}
		}
	}
}

// Claw paints a claw from its root (x0, y0), w thick, to its tip (x1,
// y1), its middle pushed bend aside (+: down and back, a hooked talon), in
// horn colours from dark to light, added to the silhouette.
func (p *Painter) Claw(x0, y0, x1, y1, w, bend float64, ramp []uint32) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	nx, ny := -dy/l, dx/l
	if ny < 0 {
		nx, ny = -nx, -ny // the side the claw curls towards: down
	}
	mx, my := (x0+x1)/2+nx*bend, (y0+y1)/2+ny*bend
	const n = 24
	var cx, cy [n + 1]float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		a, b, c := (1-t)*(1-t), 2*(1-t)*t, t*t
		cx[i], cy[i] = a*x0+b*mx+c*x1, a*y0+b*my+c*y1
	}
	pick := func(r float64) color.NRGBA {
		return rgb(ramp[max(0, min(len(ramp)-1, int(r*float64(len(ramp)))))])
	}
	bx0, by0 := min(x0, x1, mx)-w, min(y0, y1, my)-w
	bx1, by1 := max(x0, x1, mx)+w, max(y0, y1, my)+w
	for py := int(by0 * K); py <= int(by1*K); py++ {
		for px := int(bx0 * K); px <= int(bx1*K); px++ {
			i := p.Idx(px, py)
			if i < 0 {
				continue
			}
			x, y := G(px, py)
			best, bt, side := math.Inf(1), 0.0, 0.0
			for k := 0; k < n; k++ {
				d, t := segDist(x, y, cx[k], cy[k], cx[k+1], cy[k+1])
				if d < best {
					best, bt = d, (float64(k)+t)/n
					tx, ty := cx[k+1]-cx[k], cy[k+1]-cy[k]
					side = (x-cx[k])*-ty + (y-cy[k])*tx // which side of the curve
					if ny*tx-nx*ty < 0 {
						side = -side
					}
				}
			}
			half := w / 2 * math.Pow(1-bt, 0.85)
			if best > half+0.5/K {
				continue
			}
			l := 0.6 - 0.35*bt
			if side < 0 {
				l += 0.3 // the upper, lit side
			} else {
				l -= 0.15
			}
			c := pick(clamp01(l))
			if best > half-0.6/K {
				c = Mix(c, Ink, 0.55) // its outline
				if side < 0 {
					c = Mix(pick(0.3), Ink, 0.25)
				}
			}
			p.In[i], p.Out[i], p.Pro[i] = true, c, true
		}
	}
}

// HornSpec is a horn from its root (X0, Y0), W thick there, to its tip
// (X1, Y1), its middle pushed Bend aside (+: towards the side the light
// does not reach, which is how most horns curve forward), in Ramp from dark
// to light. Grooves runs fine lines along it.
type HornSpec struct {
	X0, Y0, X1, Y1 float64
	W, Bend        float64
	Ramp           []uint32
	Grooves        bool
	Shade          float64 // added to its light: below 0 for a horn on the far side
}

// Horn paints a horn over the hide, added to the silhouette: lit along its
// side towards the light, pale towards the tip, dark where it leaves the
// skin, outlined darker on its shaded side.
func (p *Painter) Horn(h HornSpec) {
	dx, dy := h.X1-h.X0, h.Y1-h.Y0
	l := math.Hypot(dx, dy)
	nx, ny := -dy/l, dx/l
	if nx*light2x+ny*light2y > 0 {
		nx, ny = -nx, -ny // n points away from the light
	}
	mx, my := (h.X0+h.X1)/2+nx*h.Bend, (h.Y0+h.Y1)/2+ny*h.Bend
	const n = 32
	var cx, cy [n + 1]float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		a, b, c := (1-t)*(1-t), 2*(1-t)*t, t*t
		cx[i], cy[i] = a*h.X0+b*mx+c*h.X1, a*h.Y0+b*my+c*h.Y1
	}
	pick := func(r float64) color.NRGBA {
		return rgb(h.Ramp[max(0, min(len(h.Ramp)-1, int(clamp01(r)*float64(len(h.Ramp)))))])
	}
	bx0, by0 := min(h.X0, h.X1, mx)-h.W, min(h.Y0, h.Y1, my)-h.W
	bx1, by1 := max(h.X0, h.X1, mx)+h.W, max(h.Y0, h.Y1, my)+h.W
	for py := int(by0 * K); py <= int(by1*K); py++ {
		for px := int(bx0 * K); px <= int(bx1*K); px++ {
			i := p.Idx(px, py)
			if i < 0 {
				continue
			}
			x, y := G(px, py)
			best, bt, side := math.Inf(1), 0.0, 0.0
			for k := 0; k < n; k++ {
				d, t := segDist(x, y, cx[k], cy[k], cx[k+1], cy[k+1])
				if d < best {
					best, bt = d, (float64(k)+t)/n
					tx, ty := cx[k+1]-cx[k], cy[k+1]-cy[k]
					tl := math.Hypot(tx, ty)
					// + on the side away from the light
					side = ((x-cx[k])*-ty + (y-cy[k])*tx) / tl
					if -ty*nx+tx*ny < 0 {
						side = -side
					}
				}
			}
			half := h.W / 2 * math.Pow(1-bt, 0.8)
			if best > half+0.4/K || (bt <= 0 && best > 0.3) {
				continue
			}
			u := side / max(half, 0.01) // -1 lit edge, +1 shaded edge
			r := 0.35 + 0.4*bt - 0.4*u + h.Shade
			if bt < 0.12 {
				r -= 0.3 * (1 - bt/0.12) // the sheath where it leaves the skin
			}
			if h.Grooves && half*K > 2.5 && u > -0.2 && math.Mod(math.Abs(u)*half*K+bt*3, 2.2) < 0.6 {
				r -= 0.15
			}
			c := pick(r)
			switch {
			case best > half-0.5/K && u > 0:
				c = Mix(pick(0), Ink, 0.5) // the shaded edge
			case best > half-0.5/K:
				c = pick(r - 0.25) // the lit edge, lightly
			}
			p.In[i], p.Out[i], p.Pro[i] = true, c, true
		}
	}
}

// Nail paints a blunt nail or hoof at the front of a foot: a rounded plate
// w wide and h high, its bottom on row y, centred on x.
func (p *Painter) Nail(x, y, w, h float64, ramp []uint32, shade float64) {
	pick := func(r float64) color.NRGBA {
		return rgb(ramp[max(0, min(len(ramp)-1, int(clamp01(r)*float64(len(ramp)))))])
	}
	s := Ellipse(x, y, w/2, h, 0)
	above := func(_, gy float64) float64 { return gy - y }
	p.Fill(Inter(s, above), true, func(px, py int, d float64) color.NRGBA {
		gx, gy := G(px, py)
		u, v := (gx-x)/(w/2), (gy-(y-h))/h
		r := 0.75 - 0.35*u - 0.25*v + shade
		if d < 0.45/K*1.5 && (u > 0 || v > 0.6) {
			return Mix(pick(0), Ink, 0.4)
		}
		return pick(r)
	})
}

// Speck lightens or darkens single pixels inside s at random, density
// dens: pores, flecks of colour, the grain of horn.
func (p *Painter) Speck(s SDF, dens, dv float64, seed uint32) {
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] || hash(px, py, seed) > dens {
				continue
			}
			x, y := G(px, py)
			if s(x, y) <= 0 {
				p.Out[i] = p.Tone(p.Mat[i], p.Val[i]+dv)
			}
		}
	}
}
