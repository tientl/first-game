package dinopaint

import (
	"math"
	"math/rand"
)

// Armour of bone, as an ankylosaur wears it: osteoderms (bony plates set
// in the skin) packed in bands across the back and down the flanks, each a
// low dome with a keel along its length and a rim of bare, pale bone, the
// skin between them dark; and the details over it kept with the body.

// ScuteSpec is a field of osteoderms. Every length is in guide pixels.
type ScuteSpec struct {
	Region SDF       // where they lie
	Parts  []string  // on these parts only (nil: any)
	X0, X1 float64   // the bands run across the body from X0 to X1
	Band   float64   // how long a plate is along the body
	Rows   []float64 // the height of each row of plates from the top edge down; the last repeats
	Gap    float64   // the skin between two plates
	Grow   float64   // how much longer a plate gets for each row down (below 0: shorter)
	Keel   float64   // the ridge along a plate, as a share of its height (0: a plain dome)
	Dome   float64   // how strongly a plate is lit as standing out of the hide (0: 0.32)
	Rim    float64   // how wide its pale rim is (0: 0.5)
	Crest  float64   // how far the top row stands out of the silhouette
	Jitter float64   // how irregular the plates are, 0 to 0.5
	Seed   uint32

	PlateMat, RimMat, SkinMat string // the materials of the plates, their rims and the skin between
	// Top is the line rows are counted down from (guide y at x); nil: the
	// top edge of the silhouette.
	Top func(x float64) float64
}

type scute struct {
	x, y, rx, ry, rot float64
	row               int
	pit               uint32
}

// Scutes paints a field of osteoderms over the hide: tiles packed edge to
// edge, as the cells of the plates' centres (each its own size), with the
// skin showing in a thin line between them. It changes the light and the
// materials of the pixels under it, so it comes first among the details,
// followed by Redraw.
func (p *Painter) Scutes(s ScuteSpec) {
	r := rand.New(rand.NewSource(int64(s.Seed) + 101))
	top := s.Top
	if top == nil {
		edge := make([]float64, p.W)
		for px := range edge {
			edge[px] = float64(p.TopEdge(px)) / K
		}
		top = func(x float64) float64 { return edge[min(p.W-1, max(0, int(x*K)))] }
	}
	dome := s.Dome
	if dome == 0 {
		dome = 0.32
	}
	rimW := s.Rim
	if rimW == 0 {
		rimW = 0.4
	}
	jit := s.Jitter
	// Lay the plates out: band after band along the body, row after row
	// down from the top, every other band staggered half a row.
	var plates []scute
	band := 0
	for x := s.X0; x < s.X1; band++ {
		w := s.Band * (1 + jit*(r.Float64()-0.5))
		cx := x + w/2
		slope := math.Atan2(top(cx+1)-top(cx-1), 2)
		d := 0.0
		if band%2 == 1 {
			d = -s.Rows[0] * 0.5
		}
		for row := 0; d < 40; row++ {
			h := s.Rows[min(row, len(s.Rows)-1)] * (1 + jit*(r.Float64()-0.5))
			pw := w * (1 + s.Grow*float64(row))
			ox := (r.Float64() - 0.5) * jit * w * 0.6
			if d+h > 0.4*h {
				plates = append(plates, scute{
					x: cx + ox, y: top(cx+ox) + d + h/2,
					rx: max(0.5, pw/2), ry: max(0.5, h/2),
					rot: slope + (r.Float64()-0.5)*jit*0.4, row: row, pit: r.Uint32(),
				})
			}
			d += h
		}
		x += w
	}
	// Buckets, to find the plates near a point quickly.
	const cell = 3.0
	gw, gh := int(float64(p.GW)/cell)+2, int(float64(p.GH)/cell)+2
	bkt := make([][]int, gw*gh)
	for i, pl := range plates {
		bx, by := int(pl.x/cell), int(pl.y/cell)
		if bx >= 0 && by >= 0 && bx < gw && by < gh {
			bkt[by*gw+bx] = append(bkt[by*gw+bx], i)
		}
	}
	// m is how far (x, y) is from plate pl's middle, as a share of its size
	// that way; u and v where it is along and across the plate.
	at := func(pl *scute, x, y float64) (m, u, v float64) {
		sn, cs := math.Sincos(-pl.rot)
		dx, dy := x-pl.x, y-pl.y
		u, v = (dx*cs-dy*sn)/pl.rx, (dx*sn+dy*cs)/pl.ry
		return math.Sqrt(u*u + v*v), u, v
	}
	// tile is the plate (x, y) lies on, how far in from its edge it is
	// (guide pixels; below 0: in the skin between plates), and its u, v.
	tile := func(x, y float64) (k int, in, u, v float64) {
		k = -1
		m1, m2 := math.Inf(1), math.Inf(1)
		bx, by := int(x/cell), int(y/cell)
		for yy := by - 3; yy <= by+3; yy++ {
			for xx := bx - 3; xx <= bx+3; xx++ {
				if xx < 0 || yy < 0 || xx >= gw || yy >= gh {
					continue
				}
				for _, i := range bkt[yy*gw+xx] {
					m, uu, vv := at(&plates[i], x, y)
					switch {
					case m < m1:
						m2 = m1
						k, m1, u, v = i, m, uu, vv
					case m < m2:
						m2 = m
					}
				}
			}
		}
		if k < 0 {
			return -1, -1, 0, 0
		}
		pl := &plates[k]
		rad := (pl.rx + pl.ry) / 2
		// The edge is half way to the next plate, and never beyond the
		// plate's own oval grown a third.
		in = min((m2-m1)/2*rad, (1.2-m1)*rad) - s.Gap/2
		return k, in, u, v
	}
	// height is the plate's surface at (x, y): flat in the middle, rounding
	// down to its edge, with a keel along it.
	height := func(x, y float64) float64 {
		k, in, u, v := tile(x, y)
		if k < 0 || in <= 0 {
			return 0
		}
		pl := &plates[k]
		R := 0.4 * min(pl.rx, pl.ry)
		t := min(1, in/R)
		h := math.Sqrt(1-(1-t)*(1-t))*0.45 + 0.75*math.Sqrt(max(0, 1-0.8*(u*u+v*v)))*t
		if s.Keel > 0 {
			h += s.Keel * max(0, 1-math.Abs(v+0.1)*1.7) * math.Sqrt(max(0, 1-u*u)) * t
		}
		return h * min(pl.rx, pl.ry) * 0.85
	}
	plateM, rimM, skinM := p.MatID(s.PlateMat), p.MatID(s.RimMat), p.MatID(s.SkinMat)
	var parts []int
	for _, n := range s.Parts {
		parts = append(parts, p.PartID(n))
	}
	onPart := func(j int) bool {
		if len(parts) == 0 {
			return true
		}
		for _, q := range parts {
			if q == j {
				return true
			}
		}
		return false
	}
	// The top row crests the outline: its plates stand out of it a little.
	if s.Crest > 0 {
		add := make([]int, len(p.In)) // the hide each new pixel takes after, +1
		for _, pl := range plates {
			if pl.row != 0 {
				continue
			}
			for py := int((pl.y - pl.ry - s.Crest - 1) * K); py <= int(pl.y*K); py++ {
				for px := int((pl.x - pl.rx) * K); px <= int((pl.x+pl.rx)*K); px++ {
					i := p.Idx(px, py)
					if i < 0 || p.In[i] {
						continue
					}
					x, y := G(px, py)
					if m, u, v := at(&pl, x, y+s.Crest); m >= 0.92 || v > 0 || math.Abs(u) > 0.8 {
						continue
					}
					j := p.Idx(px, py+int(math.Ceil((s.Crest+0.7)*K)))
					if j < 0 || !p.In[j] || !onPart(p.Part[j]) || s.Region(G(j%p.W, j/p.W)) > 0 {
						continue
					}
					add[i] = j + 1
				}
			}
		}
		// No slivers: where a plate barely clears the edge it adds a line a
		// pixel thin, which would be nothing but outline.
		for range 3 {
			for i, j := range add {
				if j == 0 {
					continue
				}
				n := 0
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if q := p.Idx(i%p.W+d[0], i/p.W+d[1]); q >= 0 && (p.In[q] || add[q] > 0) {
						n++
					}
				}
				if n < 3 {
					add[i] = 0
				}
			}
		}
		for i, j := range add {
			if j > 0 {
				j--
				p.In[i], p.Part[i], p.Mat[i], p.Nrm[i], p.Val[i] = true, p.Part[j], p.Mat[j], norm3(-0.15, -0.75, 0.65), p.Val[j]
			}
		}
	}
	val := append([]float64(nil), p.Val...)
	flat := LightOn(norm3(0, 0, 1))
	const e = 1.0 / K
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Pro[i] || !onPart(p.Part[i]) {
				continue
			}
			x, y := G(px, py)
			if s.Region(x, y) > 0 {
				continue
			}
			k, in, _, _ := tile(x, y)
			if k < 0 {
				continue
			}
			pl := &plates[k]
			base := p.Val[i]
			if in <= 0 {
				// The skin between the plates: dark, darkest under a plate's
				// lower edge, away from the light.
				p.Mat[i] = skinM
				_, up, _, _ := tile(x-light2x*0.5, y-light2y*0.5)
				val[i] = base - 0.13 - 0.14*clamp01(up/0.5+0.5)
				continue
			}
			// The plate's surface, lit as standing out of the hide.
			hx := (height(x+e, y) - height(x-e, y)) / (2 * e)
			hy := (height(x, y+e) - height(x, y-e)) / (2 * e)
			n := norm3(-hx, -hy, 1)
			out := base + dome*(LightOn(n)-flat)*2.4
			out += 0.09 * (float64(pl.pit%1000)/1000 - 0.5) // each plate a little lighter or darker
			if hash(px/2, py/2, pl.pit) < 0.05 {
				out -= 0.07 // pitted bone
			}
			p.Mat[i] = plateM
			if in < rimW {
				// The rim of bare bone, pale where it faces the light, dark
				// where it turns away.
				gx := (in - func() float64 { _, o, _, _ := tile(x+e, y); return o }()) / e
				gy := (in - func() float64 { _, o, _, _ := tile(x, y+e); return o }()) / e
				face := gx*light2x + gy*light2y // > 0: this edge faces the light
				if l := math.Hypot(gx, gy); l > 0 {
					face /= l
				}
				switch {
				case face > 0.3:
					p.Mat[i] = rimM
					out = base + 0.05 + 0.1*face
				case face < -0.3:
					out -= 0.12
				}
			}
			val[i] = out
		}
	}
	p.Val = val
}

// Redraw colours and outlines the painting again from its light and
// materials, after Scutes has changed them (the silhouette with it).
func (p *Painter) Redraw() {
	p.Sil = sdfOf(p.In, p.W, p.H)
	for i := range p.Sil.V {
		p.Sil.V[i] /= K
	}
	p.colour()
	p.outline()
}

// Mark is a copy of which pixels the details have painted so far, for Claim.
func (p *Painter) Mark() []bool { return append([]bool(nil), p.Pro...) }

// Claim makes what the details painted since mark the body's own: part of
// part, so that the rig moves it with the body (a spike on the flank stays
// put when the leg below it steps), and no longer a detail of its own.
func (p *Painter) Claim(mark []bool, part string) {
	j := p.PartID(part)
	for i, pro := range p.Pro {
		if pro && !mark[i] {
			p.Part[i], p.Pro[i] = j, false
		}
	}
}
