package dinopaint

import (
	"math"
	"math/rand"
)

// The skin: scales of many sizes packed over the hide, big pebbly ones
// along the back and small ones on the flanks and belly, each lit as a
// little dome; rows of bony osteoderms along the spine; creases at the
// neck and the joints.

type cell struct {
	x, y, r float64 // image pixels
	w       float64 // its weight in the power diagram: bigger cells reach further
	osteo   bool
	part    int
	v       float64 // the light at its middle
	tint    float64 // a little lighter or darker than its neighbours
}

type scaleSet struct {
	cells []cell
	g     float64 // the size of a bucket
	gw    int
	gh    int
	bkt   [][]int32
}

func (s *scaleSet) add(c cell) {
	s.cells = append(s.cells, c)
	bx, by := int(c.x/s.g), int(c.y/s.g)
	bx, by = min(s.gw-1, max(0, bx)), min(s.gh-1, max(0, by))
	s.bkt[by*s.gw+bx] = append(s.bkt[by*s.gw+bx], int32(len(s.cells)-1))
}

// near calls f for every cell within a bucket or two of (x, y).
func (s *scaleSet) near(x, y float64, reach int, f func(i int)) {
	bx, by := int(x/s.g), int(y/s.g)
	for yy := by - reach; yy <= by+reach; yy++ {
		for xx := bx - reach; xx <= bx+reach; xx++ {
			if xx < 0 || yy < 0 || xx >= s.gw || yy >= s.gh {
				continue
			}
			for _, i := range s.bkt[yy*s.gw+xx] {
				f(int(i))
			}
		}
	}
}

// at is the cell image pixel (x, y) lies in, how far it is from the cell's
// edge, in image pixels, and the cell's index (-1: none near).
func (s *scaleSet) at(x, y float64) (int, float64) {
	best, bd := -1, math.Inf(1)
	var cand [64]int
	nc := 0
	s.near(x, y, 1, func(i int) {
		c := &s.cells[i]
		d := (x-c.x)*(x-c.x) + (y-c.y)*(y-c.y) - c.w
		if nc < len(cand) {
			cand[nc] = i
			nc++
		}
		if d < bd {
			best, bd = i, d
		}
	})
	if best < 0 {
		return -1, 0
	}
	b := &s.cells[best]
	e := math.Inf(1)
	for _, i := range cand[:nc] {
		if i == best {
			continue
		}
		c := &s.cells[i]
		l := math.Hypot(c.x-b.x, c.y-b.y)
		if l == 0 {
			continue
		}
		pc := (x-c.x)*(x-c.x) + (y-c.y)*(y-c.y) - c.w
		pb := (x-b.x)*(x-b.x) + (y-b.y)*(y-b.y) - b.w
		e = min(e, (pc-pb)/(2*l))
	}
	return best, e
}

// columns are, for each pixel, how far it is below the top edge and above
// the bottom edge of the silhouette in its column, in guide pixels.
func (p *Painter) columns() (up, down []float64) {
	up, down = make([]float64, p.W*p.H), make([]float64, p.W*p.H)
	for x := 0; x < p.W; x++ {
		run := 0
		for y := 0; y < p.H; y++ {
			if p.In[y*p.W+x] {
				run++
			} else {
				run = 0
			}
			up[y*p.W+x] = float64(run) / K
		}
		run = 0
		for y := p.H - 1; y >= 0; y-- {
			if p.In[y*p.W+x] {
				run++
			} else {
				run = 0
			}
			down[y*p.W+x] = float64(run) / K
		}
	}
	return up, down
}

// knob is an osteoderm, in image pixels.
type knob struct {
	x, y, r float64
	parts   []string
}

// placeKnobs sets the osteoderms of every row along the top edge of the
// silhouette as it stands, and raises the edge over those that crest it.
func (p *Painter) placeKnobs() {
	seed := p.Sp.Skin.Seed
	var crest []knob
	for ri, row := range p.Sp.Osteoderms {
		n := 0
		for x := row.X0; x <= row.X1; x += row.Gap {
			n++
			px := int(x * K)
			if px < 0 || px >= p.W {
				continue
			}
			top := p.TopEdge(px)
			if top < 0 {
				continue
			}
			h := func(s uint32) float64 { return hash(n, ri, seed+s) }
			cy := float64(top) + (row.Off+(h(3)-0.5)*0.3)*K
			cx := x*K + (h(4)-0.5)*row.Gap*0.55*K
			k := knob{cx, cy, row.R * K * (0.7 + 0.6*h(5)), row.Parts}
			p.knobs = append(p.knobs, k)
			if row.Crest > 0 {
				k.y = float64(top) + (0.5-row.Crest*(0.55+0.8*h(6)))*K
				k.r *= 0.85
				crest = append(crest, k)
			}
		}
	}
	for _, k := range crest {
		for py := int(k.y - k.r - 1); py <= int(k.y+k.r+1); py++ {
			for px := int(k.x - k.r - 1); px <= int(k.x+k.r+1); px++ {
				dx, dy := float64(px)+0.5-k.x, (float64(py)+0.5-k.y)*1.25
				if i := p.Idx(px, py); i >= 0 && dx*dx+dy*dy <= k.r*k.r {
					p.In[i] = true
				}
			}
		}
	}
}

// TopEdge is the first row of the silhouette in image column px, or -1.
func (p *Painter) TopEdge(px int) int {
	for y := 0; y < p.H; y++ {
		if p.In[y*p.W+px] {
			return y
		}
	}
	return -1
}

// skin covers the hide with scales and osteoderms and creases it.
func (p *Painter) skin() {
	sk := p.Sp.Skin
	if sk.Flank == 0 {
		sk.Flank = 1.8
	}
	if sk.Back == 0 {
		sk.Back = sk.Flank * 1.6
	}
	if sk.Belly == 0 {
		sk.Belly = sk.Flank * 0.8
	}
	if sk.BackDepth == 0 {
		sk.BackDepth = 5
	}
	if sk.BellyDepth == 0 {
		sk.BellyDepth = 6
	}
	if sk.Relief == 0 {
		sk.Relief = 1
	}
	up, down := p.columns()
	size := func(i int) float64 {
		back := smoothstep(sk.BackDepth, 0.5, up[i])
		belly := smoothstep(sk.BellyDepth, 1, down[i]) * (1 - back)
		r := sk.Flank + (sk.Back-sk.Flank)*back*back + (sk.Belly-sk.Flank)*belly
		if s := p.Sp.Parts[p.Part[i]].Scale; s > 0 {
			r *= s
		}
		return r
	}
	maxR := max(sk.Flank, sk.Back, sk.Belly) * 2.2
	for _, row := range p.Sp.Osteoderms {
		maxR = max(maxR, row.R*K)
	}
	s := &scaleSet{g: 2 * maxR}
	s.gw, s.gh = int(float64(p.W)/s.g)+1, int(float64(p.H)/s.g)+1
	s.bkt = make([][]int32, s.gw*s.gh)
	rng := rand.New(rand.NewSource(int64(sk.Seed) + 7))

	fits := func(x, y, r, gap float64) bool {
		ok := true
		s.near(x, y, 1, func(i int) {
			c := &s.cells[i]
			if math.Hypot(c.x-x, c.y-y) < gap*(r+c.r) {
				ok = false
			}
		})
		return ok
	}
	// The osteoderms first: knobs in a row along the back.
	for _, k := range p.knobs {
		i := p.Idx(int(k.x), int(k.y))
		if i < 0 || !p.In[i] {
			continue
		}
		ok := len(k.parts) == 0
		for _, n := range k.parts {
			ok = ok || p.Part[i] == p.PartID(n)
		}
		if !ok || !fits(k.x, k.y, k.r, 0.9) {
			continue
		}
		s.add(cell{x: k.x, y: k.y, r: k.r, w: (0.9 * k.r) * (0.9 * k.r), osteo: true, part: p.Part[i]})
	}
	// Then scales, packed by darts thrown over the hide in a random order.
	type dart struct{ x, y float64 }
	var darts []dart
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			if p.In[py*p.W+px] {
				darts = append(darts, dart{float64(px) + rng.Float64(), float64(py) + rng.Float64()})
			}
		}
	}
	rng.Shuffle(len(darts), func(i, j int) { darts[i], darts[j] = darts[j], darts[i] })
	for _, d := range darts {
		i := p.Idx(int(d.x), int(d.y))
		r := size(i) * (0.8 + 0.4*rng.Float64())
		if !fits(d.x, d.y, r, 0.92) {
			continue
		}
		s.add(cell{x: d.x, y: d.y, r: r, w: (0.6 * r) * (0.6 * r), part: p.Part[i], tint: rng.Float64() - 0.5})
	}
	// The light at each cell's middle, so a scale reads as one piece.
	for k := range s.cells {
		c := &s.cells[k]
		i := p.Idx(int(c.x), int(c.y))
		c.v = -1
		if i >= 0 && p.In[i] && p.Part[i] == c.part {
			c.v = p.Val[i]
		}
	}
	p.cells = s

	val := append([]float64(nil), p.Val...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] {
				continue
			}
			k, e := s.at(float64(px)+0.5, float64(py)+0.5)
			if k < 0 {
				continue
			}
			c := &s.cells[k]
			rel := sk.Relief
			if r := p.Sp.Materials[p.Mat[i]].Relief; r > 0 {
				rel *= r
			}
			v := p.Val[i]
			cv := v
			if c.v >= 0 && c.part == p.Part[i] {
				cv = c.v
				v = lerp(v, c.v, 0.55) // one tone over a scale, as a painter puts it down
			}
			// The texture shows most in the half-light, least in deep shadow
			// and full light, as a painter only hints at it there.
			show := rel * (0.6 + 0.4*clamp01(1-math.Abs(cv-0.55)*1.6))
			if c.osteo {
				show = max(show, 0.7*rel)
			}
			// The scale as a little dome: lit on the side towards the light.
			ox, oy := (float64(px)+0.5-c.x)/c.r, (float64(py)+0.5-c.y)/c.r
			toward := -(ox*light2x + oy*light2y)
			amp := 0.08
			if c.osteo {
				amp = 0.17
			} else if c.r > 2.6 {
				amp = 0.09
			}
			v += show * amp * clamp(toward, -1.2, 1.2)
			v += show * 0.03 * c.tint
			// The cracks between scales, dark on the side away from the light.
			if e < 0.6 {
				switch {
				case c.osteo:
					v -= 0.12 + 0.08*clamp01(-toward)
				case toward < 0.15:
					v -= show * 0.13 * min(1, c.r/1.8)
				default:
					v -= show * 0.035
				}
			} else if c.osteo && e < 1.7 && toward > 0.35 {
				v += 0.09 // the lit lip of a knob
			}
			val[i] = v
		}
	}
	p.Val = val
	for _, f := range p.Sp.Folds {
		p.fold(f)
	}
}

func clamp(v, a, b float64) float64 { return min(b, max(a, v)) }

// fold creases the skin along N lines like f.Pts: a dark crease with a lit
// lip on the side towards the light, fading out at both ends.
func (p *Painter) fold(f Fold) {
	depth := f.Depth
	if depth == 0 {
		depth = 1
	}
	for n := 0; n < max(1, f.N); n++ {
		ox, oy := float64(n)*f.DX, float64(n)*f.DY
		pts := make([]float64, len(f.Pts))
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for i := 0; i+1 < len(f.Pts); i += 2 {
			pts[i], pts[i+1] = f.Pts[i]+ox, f.Pts[i+1]+oy
			x0, y0, x1, y1 = min(x0, pts[i]), min(y0, pts[i+1]), max(x1, pts[i]), max(y1, pts[i+1])
		}
		total := 0.0
		for i := 0; i+3 < len(pts); i += 2 {
			total += math.Hypot(pts[i+2]-pts[i], pts[i+3]-pts[i+1])
		}
		for py := int((y0 - 1) * K); py <= int((y1+1)*K); py++ {
			for px := int((x0 - 1) * K); px <= int((x1+1)*K); px++ {
				i := p.Idx(px, py)
				if i < 0 || !p.In[i] {
					continue
				}
				x, y := G(px, py)
				best, at, side := math.Inf(1), 0.0, 0.0
				run := 0.0
				for s := 0; s+3 < len(pts); s += 2 {
					d, t := segDist(x, y, pts[s], pts[s+1], pts[s+2], pts[s+3])
					l := math.Hypot(pts[s+2]-pts[s], pts[s+3]-pts[s+1])
					if d < best {
						best, at = d, (run+t*l)/total
						// which side of the line: + is towards the light
						nx, ny := -(pts[s+3] - pts[s+1]), pts[s+2]-pts[s]
						side = ((x-pts[s])*nx + (y-pts[s+1])*ny) * -(nx*light2x + ny*light2y)
					}
					run += l
				}
				taper := math.Pow(math.Sin(math.Pi*clamp01(at)), 0.6)
				w := 0.2 + 0.2*taper
				switch {
				case best < w:
					p.Val[i] -= 0.2 * depth * taper
				case best < w+0.4 && side > 0:
					p.Val[i] += 0.1 * depth * taper
				}
			}
		}
	}
}
