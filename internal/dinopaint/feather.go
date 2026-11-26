package dinopaint

import (
	"image/color"
	"math"
)

// Feathers. A coat is laid in tracts, as a bird's is: small contour
// feathers over the body, long ones along the arms and down the tail,
// a crest on the head. In a tract the feathers lie in rows along its flow
// (Coords), each row half a feather further on and shifted half a feather
// across, every feather over the root of the ones behind it: what shows of
// each is its tip, a scallop. Each feather is lit as a vane curved over its
// shaft, its tip catching the light on one side and dark on the other, and
// casts a little shadow on the feathers it lies over. Where a tract reaches
// the edge of the body its feathers make the outline, soft and broken by
// their tips.
//
// Plumage is called first thing in a species' Details: it reshapes the
// silhouette, lights the feathers, and colours and outlines the painting
// again. What no feather covers (scaly feet, a bare snout) keeps its skin.

// Tract is a patch of feathers of one kind.
type Tract struct {
	Name     string
	Region   SDF      // where the feathers grow: those whose middle is inside it
	Parts    []string // only on these parts (nil: any)
	Flow     Coords
	Len, Wid float64 // a feather's length and breadth, guide pixels
	Rows     float64 // how far apart the rows are, in feather lengths (0: 0.42)
	Cols     float64 // and the feathers in a row, in breadths (0: 0.82)
	Grow     float64 // how much longer the feathers get with each pixel along the flow, in Len
	Tip      float64 // the tips' shape: 0 round, 1 pointed
	Shaft    float64 // how much lighter (+) or darker (-) the shaft is than the vane (0: none)
	Relief   float64 // how strongly each feather is modelled (0: 1)
	Fringe   float64 // the edge of each tip lighter (+) or darker (-)
	Fluff    float64 // how far the feathers make the outline, in from the guide's edge and out past it (0: they keep to it)
	Inset    float64 // how far inside the body a feather's middle must be (0: a quarter of Wid)
	Snap     bool    // each feather lit all over with the drawing's marks at its middle
	SnapMat  bool    // and all one material, the one at its middle
	Hair     bool    // fine strands, not vaned feathers: pycnofibres, fur, down
	Jitter   float64 // how irregularly they lie (0: 0.3)
	Tint     float64 // how much one feather differs from the next (0: 0.04; hair 0.08)
	Lift     float64 // how much the whole tract stands out of the body (light, 0: none)
	Camber   float64 // how much a vane curves over its shaft (0: 0.05, or 0.13 with a shaft)
	TipMat   string  // the tips of the feathers are this material (a pale fringe, a dark edge)
	TipGuide bool    // only where the drawing has that material near the tip
	TipElse  string  // and the drawing's marks of it that are not at a tip become this: flecks made feather tips
	TipLen   float64 // how far down from the tip, in feather lengths (0: 0.2)
	Mat      string  // all its feathers this material ("": the guide's)
}

// Plumage is a coat of feathers: its tracts, back to front.
type Plumage struct {
	Tracts []Tract
	Seed   uint32
	Shadow float64 // how dark a tract shades what lies under it (0: 0.13)
	Edge   float64 // how dark the soft outline of the feathers is (0: 0.3)
	Tone   float64 // how much of the drawing's own light and dark is kept, broadly: a dark belly, a pale back (0: none)
}

// feather is one feather of a tract, made the first time a pixel asks.
type feather struct {
	sTip, tMid float64 // where its tip and middle are, in the tract's coordinates
	l, w       float64
	tint       float64
	ok         bool // it grows: its middle is inside the tract and the body
	mat, part  int
	gx, gy     float64 // its middle, in guide pixels
	tx, ty     float64 // and near its tip
	tip        int     // whether the drawing marks its tip: 0 not asked, 1 yes, 2 no
}

type fkey struct{ tr, i, j int }

// fowner is what covers a pixel.
type fowner struct {
	tr      int // the tract, -1: none
	f       *feather
	u, v, e float64 // along it (0 root, 1 tip), across (-1 to 1), how far in from its edge (guide pixels)
	shadow  float64 // under the edge of the feather above
	shaft   bool
	ax, ay  float64 // the flow and the across directions on the screen
	bx, by  float64
}

type plumage struct {
	p   *Painter
	pl  *Plumage
	sil *Field // the silhouette's distance, the picture's own edge taken as outside
	fs  map[fkey]*feather
	mat []int
	prt []int
}

// tract is tract ti of the coat.
func (q *plumage) tract(ti int) *Tract { return &q.pl.Tracts[ti] }

func (t *Tract) rows() float64 {
	if t.Rows == 0 {
		return 0.42
	}
	return t.Rows
}

func (t *Tract) cols() float64 {
	if t.Cols == 0 {
		return 0.82
	}
	return t.Cols
}

func (t *Tract) jitter() float64 {
	if t.Jitter == 0 {
		return 0.3
	}
	return t.Jitter
}

// prof is the half-breadth of a feather u along it, in halves of its
// breadth: full from the root, then rounding or tapering to the tip.
func prof(u, tip float64) float64 {
	if u < 0 || u > 1 {
		return 0
	}
	if u < 0.5 {
		return 0.82 + 0.18*u/0.5
	}
	k := (u - 0.5) / 0.5
	round := math.Sqrt(max(0, 1-k*k))
	point := math.Pow(max(0, 1-k), 0.85)
	return lerp(round, point, tip)
}

// get is feather (i, j) of tract ti; (x, y) is a guide point near it and
// jac the tract's coordinates' derivatives there, to find its middle on the
// body.
func (q *plumage) get(ti, i, j int, x, y, s, t float64, jac [4]float64) *feather {
	k := fkey{ti, i, j}
	if f, ok := q.fs[k]; ok {
		return f
	}
	tr := q.tract(ti)
	seed := q.pl.Seed + uint32(ti)*7919
	h := func(n int) float64 { return hash(i*31+n, j*17-n*5, seed) }
	ds, dt := tr.rows()*tr.Len, tr.cols()*tr.Wid
	jit := tr.jitter()
	f := &feather{}
	f.sTip = float64(i)*ds + (h(1)-0.5)*ds*jit
	off := 0.0
	if i&1 == 1 {
		off = 0.5
	}
	f.tMid = (float64(j)+off)*dt + (h(2)-0.5)*dt*jit*0.7
	f.l = tr.Len * max(0.3, 1+tr.Grow*f.sTip) * (1 + (h(3)-0.5)*jit*0.4)
	f.w = tr.Wid * (1 + (h(4)-0.5)*jit*0.3)
	tint := tr.Tint
	if tint == 0 {
		tint = 0.04
		if tr.Hair {
			tint = 0.08
		}
	}
	f.tint = (h(5) - 0.5) * 2 * tint
	// Where its middle and its tip are on the body.
	f.mat, f.part = -1, -1
	mx, my, ok1 := locate(tr, f.sTip-0.45*f.l, f.tMid, x, y, s, t, jac)
	tx, ty, ok2 := locate(tr, f.sTip-0.15*f.l, f.tMid, mx, my, f.sTip-0.45*f.l, f.tMid, jac)
	f.gx, f.gy, f.tx, f.ty = mx, my, tx, ty
	// Its tip must be in the picture: a feather that would reach past the
	// picture's edge is shorter, so a tail cut off by it still ends in
	// tips, not in a straight line.
	inPic := func(x, y float64) bool {
		return x >= 0.25 && y >= 0.25 && x*K < float64(q.p.W)-0.75 && y*K < float64(q.p.H)-0.75
	}
	if ex, ey := mx+(tx-mx)*1.5, my+(ty-my)*1.5; ok1 && ok2 && !inPic(ex, ey) {
		sm := f.sTip - 0.45*f.l
		lo, hi := sm, f.sTip
		for range 10 {
			mid := (lo + hi) / 2
			x1, y1, ok := locate(tr, mid, f.tMid, mx, my, sm, f.tMid, jac)
			if ok && inPic(x1, y1) {
				lo = mid
			} else {
				hi = mid
			}
		}
		f.l -= f.sTip - lo
		f.sTip = lo
		ok1 = f.l > 0.45*tr.Len
	}
	if ok1 && ok2 {
		inset := tr.Inset
		if inset == 0 {
			inset = tr.Wid / 4
		}
		p := q.p
		px, py := int(math.Floor(mx*K)), int(math.Floor(my*K))
		if idx := p.Idx(px, py); idx >= 0 && q.sil.V[idx] <= -inset && (tr.Region == nil || tr.Region(mx, my) <= 0) {
			f.mat, f.part = q.mat[idx], q.prt[idx]
			f.ok = len(tr.Parts) == 0
			for _, n := range tr.Parts {
				f.ok = f.ok || f.part == p.PartID(n)
			}
		}
	}
	q.fs[k] = f
	return f
}

// locate is the guide point where tract tr's coordinates are (s0, t0),
// found by stepping from (x, y), where they are (s, t) and change as jac
// says.
func locate(tr *Tract, s0, t0, x, y, s, t float64, jac [4]float64) (float64, float64, bool) {
	for range 8 {
		det := jac[0]*jac[3] - jac[1]*jac[2]
		if det == 0 {
			return x, y, false
		}
		dS, dT := s0-s, t0-t
		if math.Abs(dS)+math.Abs(dT) < 0.02 {
			return x, y, true
		}
		// No step longer than a few pixels: the coordinates bend.
		sx := (jac[3]*dS - jac[1]*dT) / det
		sy := (-jac[2]*dS + jac[0]*dT) / det
		if l := math.Hypot(sx, sy); l > 3 {
			sx, sy = sx*3/l, sy*3/l
		}
		x, y = x+sx, y+sy
		s, t, jac = coords(tr, x, y)
	}
	return x, y, math.Abs(s0-s)+math.Abs(t0-t) < 0.1
}

// coords are tract tr's coordinates at guide point (x, y), and their
// derivatives: ds/dx, ds/dy, dt/dx, dt/dy.
func coords(tr *Tract, x, y float64) (s, t float64, jac [4]float64) {
	const h = 0.2
	s, t = tr.Flow(x, y)
	s1, t1 := tr.Flow(x+h, y)
	s2, t2 := tr.Flow(x, y+h)
	dt1, dt2 := t1-t, t2-t
	// Across a fan's seam t jumps: take the other side's.
	if math.Abs(dt1) > 5*h {
		dt1 = 0
	}
	if math.Abs(dt2) > 5*h {
		dt2 = 0
	}
	return s, t, [4]float64{(s1 - s) / h, (s2 - s) / h, dt1 / h, dt2 / h}
}

// find is the feather of tract ti over guide point (x, y), with (s, t) its
// coordinates there, or nil.
func (q *plumage) find(ti int, x, y, s, t float64, jac [4]float64) (f *feather, u, v, e float64) {
	tr := q.tract(ti)
	ds, dt := tr.rows()*tr.Len, tr.cols()*tr.Wid
	lmax := tr.Len * max(0.3, 1+tr.Grow*(s+tr.Len*2)) * 1.25
	i0 := int(math.Floor(s/ds)) - 1
	i1 := int(math.Ceil((s+lmax)/ds)) + 1
	reach := int(math.Ceil(tr.Wid/dt*0.6)) + 1
	for i := i0; i <= i1; i++ { // the rows from the one on top down
		off := 0.0
		if i&1 == 1 {
			off = 0.5
		}
		jc := int(math.Round(t/dt - off))
		var best *feather
		bu, bv, be := 0.0, 0.0, 0.0
		for j := jc - reach; j <= jc+reach; j++ {
			g := q.get(ti, i, j, x, y, s, t, jac)
			if !g.ok {
				continue
			}
			uu := 1 - (g.sTip-s)/g.l
			hw := g.w / 2 * prof(uu, tr.Tip)
			d := t - g.tMid
			if hw <= 0 || math.Abs(d) > hw {
				continue
			}
			if ee := hw - math.Abs(d); best == nil || ee > be {
				best, bu, bv, be = g, uu, d/(g.w/2), ee
			}
		}
		if best != nil {
			// e is how far in from its edge: the side, or the tip.
			be = min(be, (best.sTip-s)*0.8)
			return best, bu, bv, be
		}
	}
	return nil, 0, 0, 0
}

// Plumage lays the coat pl over the painting.
func (p *Painter) Plumage(pl *Plumage) {
	q := &plumage{p: p, pl: pl, fs: map[fkey]*feather{}}
	// The picture's own edge is outside too, so a tail cut off by it
	// still ends in feathers.
	inB := append([]bool(nil), p.In...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			if px == 0 || py == 0 || px == p.W-1 || py == p.H-1 {
				inB[py*p.W+px] = false
			}
		}
	}
	q.sil = sdfOf(inB, p.W, p.H)
	for i := range q.sil.V {
		q.sil.V[i] /= K
	}
	q.mat = append([]int(nil), p.Mat...)
	q.prt = append([]int(nil), p.Part...)
	n := p.W * p.H
	skinVal := append([]float64(nil), p.Val...)
	oldIn := append([]bool(nil), p.In...)
	maxFluff := 0.0
	for _, tr := range pl.Tracts {
		maxFluff = max(maxFluff, tr.Fluff)
	}
	own := make([]fowner, n)
	for i := range own {
		own[i].tr = -1
	}
	// What covers each pixel: the frontmost tract with a feather there.
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !oldIn[i] && q.sil.V[i] > maxFluff {
				continue
			}
			x, y := G(px, py)
			for ti := len(pl.Tracts) - 1; ti >= 0; ti-- {
				tr := &pl.Tracts[ti]
				if !oldIn[i] && q.sil.V[i] > tr.Fluff {
					continue
				}
				if tr.Region != nil && tr.Region(x, y) > tr.Len*1.2 {
					continue
				}
				s, t, jac := coords(tr, x, y)
				f, u, v, e := q.find(ti, x, y, s, t, jac)
				if f == nil {
					continue
				}
				o := &own[i]
				o.tr, o.f, o.u, o.v, o.e = ti, f, u, v, e
				// Just below the edge of the feather over it?
				const step = 0.42
				if g, _, _, _ := q.find(ti, x, y, s-step, t, jac); g != nil && g != f && g.sTip < f.sTip {
					o.shadow = 1
				} else if g, _, _, _ := q.find(ti, x, y, s-2*step, t, jac); g != nil && g != f && g.sTip < f.sTip {
					o.shadow = 0.45
				}
				// The flow and across directions on the screen.
				ax, ay := jac[0], jac[1]
				if l := math.Hypot(ax, ay); l > 0 {
					ax, ay = ax/l, ay/l
				}
				bx, by := jac[2], jac[3]
				if l := math.Hypot(bx, by); l > 0 {
					bx, by = bx/l, by/l
				}
				o.ax, o.ay, o.bx, o.by = ax, ay, bx, by
				o.shaft = !tr.Hair && tr.Shaft != 0 && f.w*K >= 3.5 && math.Abs(t-f.tMid) < 0.55/K && u > 0.06 && u < 0.93
				break
			}
		}
	}
	// The outline where the feathers make it: their tips out past the
	// guide's edge, the gaps between them cut in.
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			sil := q.sil.V[i]
			if sil > maxFluff || sil < -maxFluff {
				continue
			}
			x, y := G(px, py)
			fluffy := false
			for ti := range pl.Tracts {
				tr := &pl.Tracts[ti]
				if tr.Fluff > 0 && sil >= -tr.Fluff && sil <= tr.Fluff && (tr.Region == nil || tr.Region(x, y) <= 0) {
					fluffy = true
				}
			}
			if !fluffy {
				continue
			}
			o := own[i]
			p.In[i] = o.tr >= 0 && (oldIn[i] || pl.Tracts[o.tr].Fluff > 0)
		}
	}
	// No lone pixels left by that.
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			c := 0
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if p.Inside(px+d[0], py+d[1]) {
					c++
				}
			}
			if p.In[i] && c == 0 {
				p.In[i] = false
			}
		}
	}
	// A pixel the feathers added takes the hide of the nearest one in the body.
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || oldIn[i] {
				continue
			}
			best, bd := -1, math.Inf(1)
			for dy := -6; dy <= 6; dy++ {
				for dx := -6; dx <= 6; dx++ {
					if j := p.Idx(px+dx, py+dy); j >= 0 && oldIn[j] {
						if d := float64(dx*dx + dy*dy); d < bd {
							best, bd = j, d
						}
					}
				}
			}
			if best < 0 {
				p.In[i] = false
				continue
			}
			p.Mat[i], p.Part[i], p.Nrm[i] = p.Mat[best], p.Part[best], p.Nrm[best]
		}
	}
	// A feather is one material when its tract says so, and of the part its
	// middle grows from.
	for i, o := range own {
		if !p.In[i] || o.tr < 0 {
			continue
		}
		tr := &pl.Tracts[o.tr]
		if tr.SnapMat && o.f.mat >= 0 {
			p.Mat[i] = o.f.mat
		}
		if tr.Mat != "" {
			p.Mat[i] = p.MatID(tr.Mat)
		}
		if tr.TipMat != "" {
			tl := tr.TipLen
			if tl == 0 {
				tl = 0.2
			}
			switch tip := p.MatID(tr.TipMat); {
			case o.u > 1-tl && (!tr.TipGuide || q.tipMarked(o.f, tr)):
				p.Mat[i] = tip
			case tr.TipElse != "" && p.Mat[i] == tip:
				p.Mat[i] = p.MatID(tr.TipElse)
			}
		}
		if len(tr.Parts) > 0 && o.f.part >= 0 {
			p.Part[i] = o.f.part
		}
	}
	p.Sil = sdfOf(p.In, p.W, p.H)
	for i := range p.Sil.V {
		p.Sil.V[i] /= K
	}
	p.shade()
	// The feathers lit, over the light on the body. A snapped feather takes
	// the drawing's marks at its middle, all over: the mottling of the
	// guide laid feather by feather.
	detail := p.guideDetail()
	lay := make([]int, n)
	for i := range p.In {
		lay[i] = -1
		if !p.In[i] {
			continue
		}
		o := own[i]
		if o.tr < 0 {
			if oldIn[i] {
				p.Val[i] = skinVal[i]
			}
			continue
		}
		lay[i] = o.tr
		if detail != nil && pl.Tracts[o.tr].Snap {
			x, y := G(i%p.W, i/p.W)
			p.Val[i] += p.Sp.GuideDetail * (detail.Sample(o.f.gx, o.f.gy) - detail.Sample(x, y))
		}
		p.Val[i] += q.light(&o)
	}
	if pl.Tone != 0 {
		q.guideTone(pl.Tone)
	}
	// Each tract shades what lies under it, down and away from the light.
	shadow := pl.Shadow
	if shadow == 0 {
		shadow = 0.13
	}
	val := append([]float64(nil), p.Val...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] {
				continue
			}
			for k, w := range [2]float64{1, 0.5} {
				d := float64(k + 1)
				j := p.Idx(px+int(math.Round(light2x*d*1.2)), py+int(math.Round(light2y*d*1.2)))
				if j >= 0 && p.In[j] && lay[j] > lay[i] {
					val[i] -= shadow * w
					break
				}
			}
		}
	}
	p.Val = val
	p.colour()
	// The feathers keep their fine lines: no tone is cleaned away on them.
	for i, o := range own {
		if p.In[i] && o.tr >= 0 {
			p.Out[i] = p.Tone(p.Mat[i], p.Val[i])
		}
	}
	p.outline()
	q.finish(own, oldIn)
}

// light is how much the feather over a pixel changes the light on it.
func (q *plumage) light(o *fowner) float64 {
	tr := q.tract(o.tr)
	rel := tr.Relief
	if rel == 0 {
		rel = 1
	}
	f := o.f
	la := o.ax*light2x + o.ay*light2y // the flow towards the light
	lb := o.bx*light2x + o.by*light2y // across it
	dv := f.tint + tr.Lift
	if tr.Hair {
		// A strand: lit down its middle, dark between it and the next.
		dv += rel * (0.05 - 0.13*smoothstep(0.35, 1, math.Abs(o.v)))
		dv += rel * 0.05 * smoothstep(0.55, 1, o.u)
		dv -= rel * 0.1 * o.shadow
		return dv
	}
	// The vane curves down from its shaft to its edges.
	camber := tr.Camber
	if camber == 0 {
		camber = 0.05
		if tr.Shaft != 0 {
			camber = 0.13
		}
	}
	dv += rel * camber * o.v * lb
	// The tip curls down onto the feathers below.
	dv += rel * 0.07 * smoothstep(0.55, 1, o.u) * la
	// In the shadow of the feather over it.
	dv -= rel * 0.17 * o.shadow
	// The edge of the tip: lit where it faces the light, dark where not.
	if o.e < 0.5/K*1.6 && o.u > 0.45 {
		nx := o.ax*(o.u-0.45) + o.bx*o.v
		ny := o.ay*(o.u-0.45) + o.by*o.v
		if l := math.Hypot(nx, ny); l > 0 {
			face := (nx*light2x + ny*light2y) / l
			dv += rel * (0.09*face - 0.03)
		}
		dv += tr.Fringe
	}
	return dv
}

// finish softens the outline the feathers make and draws their shafts.
func (q *plumage) finish(own []fowner, oldIn []bool) {
	p := q.p
	edge := q.pl.Edge
	if edge == 0 {
		edge = 0.3
	}
	out := append([]color.NRGBA(nil), p.Out...)
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || own[i].tr < 0 {
				continue
			}
			o := own[i]
			tr := q.tract(o.tr)
			m := p.Mat[i]
			k := p.ToneIdx(m, p.Val[i])
			isEdge := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if !p.Inside(px+d[0], py+d[1]) {
					isEdge = true
				}
			}
			switch {
			case isEdge:
				gx := p.Sil.At(px+1, py) - p.Sil.At(px-1, py)
				gy := p.Sil.At(px, py+1) - p.Sil.At(px, py-1)
				face := 0.0
				if l := math.Hypot(gx, gy); l > 0 {
					face = -(gx*light2x + gy*light2y) / l
				}
				lit := smoothstep(-0.2, 0.8, face)
				drop := 3
				if tr.Hair {
					drop = 2
				}
				c := p.Step(m, max(0, k-drop+int(math.Round(2*lit))))
				out[i] = Mix(c, Ink, edge*(1-0.7*lit))
			case o.shaft:
				out[i] = p.Tone(m, p.Val[i]+tr.Shaft)
			}
		}
	}
	p.Out = out
}

// tipMarked reports whether the drawing has tract tr's tip material near
// the tip of feather f.
func (q *plumage) tipMarked(f *feather, tr *Tract) bool {
	if f.tip == 0 {
		f.tip = 2
		m := q.p.MatID(tr.TipMat)
		r := 0.55
		for py := int((f.ty - r) * K); py <= int((f.ty+r)*K); py++ {
			for px := int((f.tx - r) * K); px <= int((f.tx+r)*K); px++ {
				x, y := G(px, py)
				if i := q.p.Idx(px, py); i >= 0 && q.mat[i] == m && math.Hypot(x-f.tx, y-f.ty) <= r {
					f.tip = 1
				}
			}
		}
	}
	return f.tip == 1
}

// guideTone lightens and darkens the painting as the drawing is, broadly:
// each pixel by how much lighter or darker the drawing is there than its
// material's colours are on average, so the drawing's dark undersides and
// pale backs, and the strength of its darks, are kept.
func (q *plumage) guideTone(k float64) {
	p := q.p
	for i, d := range p.toneField() {
		p.Val[i] += k * d
	}
}

// toneField is, for each pixel, how much lighter (+) or darker (-) the
// drawing is there, broadly, than its material's colours on average, in
// steps of light; the lighter half counts half.
func (p *Painter) toneField() []float64 {
	out := make([]float64, p.W*p.H)
	l, w := NewField(p.GW, p.GH), NewField(p.GW, p.GH)
	for i, c := range p.Guide {
		if p.GSolid[i] && lum(c) >= p.Sp.Ink {
			l.V[i], w.V[i] = lum(c), 1
		}
	}
	lb, wb := l.Blur(0.8), w.Blur(0.8)
	at := func(i int) (float64, bool) {
		x, y := G(i%p.W, i/p.W)
		if s := wb.Sample(x, y); s > 0.05 {
			return lb.Sample(x, y) / s, true
		}
		return 0, false
	}
	nm := len(p.Sp.Materials)
	sum, cnt := make([]float64, nm), make([]float64, nm)
	for i, in := range p.In {
		if v, ok := at(i); in && ok {
			sum[p.Mat[i]] += v
			cnt[p.Mat[i]]++
		}
	}
	for i, in := range p.In {
		v, ok := at(i)
		if !in || !ok || cnt[p.Mat[i]] == 0 {
			continue
		}
		m := p.Mat[i]
		d := clamp((v-sum[m]/cnt[m])/90, -0.4, 0.4)
		if d > 0 {
			d *= 0.5 // the drawing's darks matter most; the form already lights the rest
		}
		out[i] = d
	}
	return out
}
