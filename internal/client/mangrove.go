package client

import (
	"cmp"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"sync"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The mangrove at HD: a red mangrove (Rhizophora) standing in the shallows
// or on the beach on its stilt roots.
//
// The wood is laid out in the tree's own space (x across, z towards the
// viewer, h up from the ground, in art pixels) and seen from the game's
// angle, so the roots spread round the trunk in depth: the far ones go down
// behind it, darker, the near ones come down in front of it. A few thick
// legs leave the foot of the trunk and more roots leave it higher up, each
// thick where it leaves and thinner as it goes; it arches out and comes
// down steeply, wandering a little this way and that and swelling here and
// there. Some fork before they reach the ground, some split into fingers at
// their foot, and roots hang from the crown on long thin cords, one of them
// not down yet. The bark is grey-brown with reddish patches, fine rings and
// pale lenticels, lit from the top left with a highlight along the lit
// side; the tide leaves the lower part of every root dark and wet, crusted
// with algae and barnacles. Each root casts its shadow on the ground under
// its arch. Where a root meets the water a ring of water stands round it
// (the ripples and its reflection are drawn under the tree, where the
// ground under that root is the sea: drawMangroveWater); on the sand, it
// goes into a little mound. Pencil roots poke up from the mud round it.
//
// There are mangroveVariants looks, which differ in their roots, the lean
// of the trunk and the crown. Each is drawn in the water and on the sand,
// full grown and felled, when a mangrove is first near (prefetchArt).

const mangroveVariants = 4

// The mangrove's canvas, in art pixels: as wide as it was first drawn (52 *
// mangroveS), its ground point where it was, and room under that for the
// roots that come towards the viewer.
const (
	mgW    = 86
	mgH    = 101
	mgBX   = 43.0
	mgBase = 92.0
	mgGY   = 0.5 // a step towards the viewer is half a step down the screen
)

var (
	mgBarkGrey = mkRamp(0x1f1b19, 0x322b27, 0x48403a, 0x60564e, 0x7a7065, 0x958b7e, 0xb2a99a)
	mgBarkRed  = mkRamp(0x261b17, 0x3a2920, 0x50382c, 0x684939, 0x815d4a, 0x9a745f, 0xb18c76)
	mgYoung    = mkRamp(0x29281d, 0x413f2c, 0x5a583e, 0x757352, 0x928f69, 0xafad86)
	mgAlgae    = mkRamp(0x1b2114, 0x28311a, 0x364222, 0x45532b, 0x566434, 0x69763f, 0x7d894c)
	mgBarnacle = mkRamp(0x4e4a42, 0x7c776b, 0xa9a496, 0xd2cebf, 0xeceadf)
	mgMound    = mkRamp(0x4e3f2c, 0x6a573e, 0x857053, 0x9f8866, 0xb69f79, 0xcab48d)
	mgWetCol   = hex(0x1a201d)
	mgLenticel = hex(0xd2c6a6)
)

// mgP is a point in the tree's own space, in art pixels.
type mgP struct{ x, z, h float64 }

func (p mgP) plus(q mgP) mgP      { return mgP{p.x + q.x, p.z + q.z, p.h + q.h} }
func (p mgP) times(k float64) mgP { return mgP{p.x * k, p.z * k, p.h * k} }

// sx and sy are where p is on the canvas, in art pixels.
func (p mgP) sx() float64 { return mgBX + p.x }
func (p mgP) sy() float64 { return mgBase + p.z*mgGY - p.h }

// mgOut is a step of length d over the ground at angle a: 0 to the right, a
// quarter turn towards the viewer.
func mgOut(a, d float64) mgP { return mgP{math.Cos(a) * d, math.Sin(a) * d, 0} }

// mgNode is a point of the spine of a piece of wood and its radius there.
type mgNode struct {
	p mgP
	r float64
}

// The kinds of wood a mangrove is drawn from.
const (
	mgRoot   = iota // a stilt root, or a fork or a finger of one
	mgAerial        // a root hanging from the crown
	mgPneu          // a pencil root poking up from the mud
	mgTrunk
	mgBranch
)

// mgPart is one piece of wood.
type mgPart struct {
	kind   int
	nodes  []mgNode
	z      float64 // how near: what is nearer is drawn later
	tone   float64 // lighter (more) or darker: the far roots are in shade
	merge  float64 // how far from its start it grows out of the wood it leaves, with no seam (art pixels)
	h0     float64 // how high it leaves the trunk (a fork or a finger: its root's)
	foot   bool    // it reaches the ground at its end (a pencil root: at its start)
	young  bool    // an aerial root not down yet: a pale, soft tip
	parent int     // the part a fork or a finger leaves, or -1
	seed   uint32
}

// mgFoot is where a root meets the ground, in art pixels from the tree's
// ground point, and how thick it is there.
type mgFoot struct {
	x, y, r float64
	pneu    bool // a pencil root
}

// mgLayout is the wood and the crown of a variant.
type mgLayout struct {
	parts  []mgPart
	hb, ht float64     // the trunk ends at hb in its roots and goes into the crown at ht
	cut    float64     // where a felled one was cut
	crown  []blob      // in art pixels
	cx, cy float64     // the middle of the crown
	wide   float64     // how wide the crown is, to the first drawing's
	feet   [2][]mgFoot // felled (0) and full grown (1)
}

// mgBez is the point at t on the cubic through p0..p3.
func mgBez(p0, p1, p2, p3 mgP, t float64) mgP {
	u := 1 - t
	a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return mgP{a*p0.x + b*p1.x + c*p2.x + d*p3.x, a*p0.z + b*p1.z + c*p2.z + d*p3.z, a*p0.h + b*p1.h + c*p2.h + d*p3.h}
}

// mgCurve is the spine of a piece of wood along the cubic through p0..p3,
// rad its radius at t (0 to 1). It wanders across its way by up to about
// bend art pixels (never at its ends) and swells a little here and there.
func mgCurve(p0, p1, p2, p3 mgP, rad func(t float64) float64, bend float64, seed uint32) []mgNode {
	ln, prev := 0.0, p0
	for i := 1; i <= 16; i++ {
		p := mgBez(p0, p1, p2, p3, float64(i)/16)
		ln += math.Hypot(p.sx()-prev.sx(), p.sy()-prev.sy())
		prev = p
	}
	n := max(8, int(ln*2.4))
	nodes := make([]mgNode, n+1)
	for i := range nodes {
		t := float64(i) / float64(n)
		nodes[i] = mgNode{mgBez(p0, p1, p2, p3, t), rad(t)}
	}
	off := make([]mgP, n+1)
	for i := range nodes {
		t := float64(i) / float64(n)
		a, b := nodes[max(0, i-1)].p, nodes[min(n, i+1)].p
		dx, dy := b.sx()-a.sx(), b.sy()-a.sy()
		l := math.Hypot(dx, dy)
		if l < 1e-9 {
			continue
		}
		nx, ny := -dy/l, dx/l
		w := bend * math.Pow(math.Sin(math.Pi*t), 0.7) *
			(2*(vnoise(t*ln/8, 0.5, seed)-0.5) + 0.8*(vnoise(t*ln/2.8, 1.5, seed+1)-0.5))
		off[i] = mgP{nx * w, 0, -ny * w}
	}
	for i := range nodes {
		L := float64(i) / float64(n) * ln
		nodes[i].p = nodes[i].p.plus(off[i])
		knob := math.Max(0, vnoise(L/2.2, 3.5, seed+3)-0.68) * 0.7
		nodes[i].r *= 1 + 0.24*(vnoise(L/1.7, 2.5, seed+2)-0.5) + knob
	}
	return nodes
}

// mgRootRad is the radius of a root r0 thick where it leaves the wood and
// r1 at its foot: a collar where it leaves, a flare where it goes in.
func mgRootRad(r0, r1 float64) func(t float64) float64 {
	return func(t float64) float64 {
		r := r1 + (r0-r1)*math.Pow(1-t, 1.3)
		r += r0 * 0.2 * math.Max(0, 1-t/0.08)
		r += r1 * 0.4 * smooth(clamp01((t-0.86)/0.14))
		return r
	}
}

var (
	mgLayouts    [mangroveVariants]*mgLayout
	mgLayoutOnce [mangroveVariants]sync.Once
	// mgSeeds picks the roots of each variant: chosen by eye from many.
	mgSeeds = [mangroveVariants]uint64{20, 1037, 2088, 3122}
)

// mangroveLayoutOf is the layout of variant v, made the first time it is
// wanted.
func mangroveLayoutOf(v int) *mgLayout {
	v = ((v % mangroveVariants) + mangroveVariants) % mangroveVariants
	mgLayoutOnce[v].Do(func() { mgLayouts[v] = mangroveLayout(v, mgSeeds[v]) })
	return mgLayouts[v]
}

// mangroveLayout lays out variant v, its roots placed at random from seed.
func mangroveLayout(v int, seed uint64) *mgLayout {
	rng := rand.New(rand.NewPCG(seed, 29))
	sd := uint32(v)*131 + 9
	L := &mgLayout{}
	L.hb = 11 + rng.Float64()*3
	L.ht = 49 + rng.Float64()*4
	L.cut = L.hb + 13 + rng.Float64()*3
	lean := [mangroveVariants]float64{0.8, -3.8, 4.4, -1.8}[v] + (rng.Float64()-0.5)*0.8
	crook := (rng.Float64() - 0.5) * 3.2
	// trunkAt is the middle of the trunk at height h and its radius there:
	// thicker just above where the roots leave it, narrowing below that.
	trunkAt := func(h float64) (float64, float64) {
		f := clamp01((h - L.hb) / (L.ht - L.hb))
		x := lean*math.Pow(f, 1.4) + crook*math.Sin(math.Pi*f)
		r := 2.9 - 0.65*f + 0.35*math.Max(0, 1-(h-L.hb)/4)
		if h < L.hb {
			r *= 1 - 0.5*clamp01((L.hb-h)/1.2)
		}
		return x, r
	}
	add := func(p mgPart) int {
		L.parts = append(L.parts, p)
		return len(L.parts) - 1
	}
	var tn []mgNode
	for h := L.hb - 1.2; h <= L.ht+6; h += 0.4 {
		x, r := trunkAt(h)
		tn = append(tn, mgNode{mgP{x, 0, h}, r * (1 + 0.1*(vnoise(h/3.2, 7.5, sd)-0.5))})
	}
	add(mgPart{kind: mgTrunk, nodes: tn, z: 0, tone: 0.02, parent: -1, seed: sd})

	// The roots: a few legs from the foot of the trunk, the rest higher up,
	// spread round it unevenly.
	type spec struct {
		a, h0, R, r0, r1 float64
		leg              bool
	}
	var specs []spec
	for _, a := range [mangroveVariants][]float64{{0.3, 0.84, -0.17}, {0.16, 0.7, -0.84, -0.12}, {0.4, 0.95, -0.8}, {0.1, 0.62, -0.88, -0.2}}[v] {
		s := spec{a: a*math.Pi + (rng.Float64()-0.5)*0.3, h0: L.hb + 0.2 + rng.Float64()*1.6,
			R: 11 + rng.Float64()*6, r0: 1.8 + rng.Float64()*0.4, r1: 0.78 + rng.Float64()*0.16, leg: true}
		if math.Sin(s.a) < -0.5 { // going away: further, or it would show only as a stub
			s.R += 6
			s.h0 += 1.5
		}
		specs = append(specs, s)
	}
	// near is how close angle a comes to a root already laid out.
	near := func(a float64) float64 {
		d := math.Pi
		for _, s := range specs {
			d = math.Min(d, math.Abs(math.Remainder(a-s.a, 2*math.Pi)))
		}
		return d
	}
	nProps := [mangroveVariants]int{7, 8, 6, 8}[v]
	high := [mangroveVariants]float64{15, 17, 12, 16}[v]
	a0 := rng.Float64() * 2 * math.Pi
	front := 0
	for _, s := range specs {
		if math.Sin(s.a) > 0.7 {
			front++
		}
	}
	for i := range nProps {
		a := a0 + 2*math.Pi*(float64(i)+(rng.Float64()-0.5)*0.85)/float64(nProps)
		if math.Sin(a) > 0.7 { // not many straight at the viewer, hiding the trunk
			if front++; front > 2 {
				a += math.Copysign(0.85, math.Cos(a)+0.01)
			}
		}
		if near(a) < 0.26 { // not two side by side, like the bars of a cage
			a += math.Copysign(0.34, rng.Float64()-0.5)
		}
		h0 := L.hb + 2 + rng.Float64()*high
		specs = append(specs, spec{a: a, h0: h0, R: 12 + (h0-L.hb)*0.85 + rng.Float64()*8,
			r0: 1.15 + rng.Float64()*0.45 + (h0-L.hb)*0.012, r1: 0.55 + rng.Float64()*0.2})
	}
	// fit keeps a foot at angle a and reach R on the canvas, and not too
	// far towards the viewer.
	fit := func(a, R float64) float64 {
		if c := math.Abs(math.Cos(a)); c*R > 36 {
			R = 36 / c
		}
		if s := math.Sin(a); s*R > 13 {
			R = 13 / s
		} else if -s*R > 24 {
			R = 24 / -s
		}
		return R
	}
	depth := func(z float64) float64 { return clamp(z*0.012, -0.2, 0.05) }
	for i, s := range specs {
		R := fit(s.a, s.R)
		tx, tr := trunkAt(s.h0)
		o := mgP{tx, 0, s.h0}.plus(mgOut(s.a, tr*0.5))
		af := s.a + (rng.Float64()-0.5)*0.25
		foot := mgOut(af, fit(af, R))
		k1 := 0.22 + rng.Float64()*0.25
		lift := (rng.Float64()*0.34 - 0.08) * s.h0
		k2 := -0.06 + rng.Float64()*0.2
		k3 := 0.45 + rng.Float64()*0.4
		p1 := o.plus(mgOut(s.a, R*k1)).plus(mgP{0, 0, lift})
		p2 := foot.plus(mgOut(s.a, -R*k2)).plus(mgP{0, 0, s.h0 * k3})
		seed := sd + uint32(i)*17 + 1
		nodes := mgCurve(o, p1, p2, foot, mgRootRad(s.r0, s.r1), 0.9+rng.Float64()*0.8, seed)
		z := (o.z + 2*foot.z) / 3
		ri := add(mgPart{kind: mgRoot, nodes: nodes, z: z, tone: depth(z), merge: 2, h0: s.h0, foot: true, parent: -1, seed: seed})
		last := len(nodes) - 1
		if !s.leg && R > 14 && rng.Float64() < 0.45 { // it forks on its way down
			k := int(float64(last) * (0.45 + rng.Float64()*0.22))
			q := nodes[k]
			a2 := s.a + math.Copysign(0.3+rng.Float64()*0.32, rng.Float64()-0.5)
			f2 := mgOut(a2, fit(a2, math.Hypot(q.p.x, q.p.z)+3+rng.Float64()*6))
			tg := nodes[k+1].p.plus(nodes[k-1].p.times(-1))
			tg = tg.times(2.5 / math.Max(1e-6, math.Hypot(math.Hypot(tg.x, tg.z), tg.h)))
			fs := seed + 7
			fn := mgCurve(q.p, q.p.plus(tg), f2.plus(mgP{0, 0, q.p.h * 0.6}), f2, mgRootRad(q.r*0.72, s.r1*0.85), 0.7, fs)
			fz := (q.p.z + 2*f2.z) / 3
			add(mgPart{kind: mgRoot, nodes: fn, z: fz, tone: depth(fz), merge: 1.2, h0: s.h0, foot: true, parent: ri, seed: fs})
		}
		if rng.Float64() < 0.38 { // it splits into two fingers at its foot
			k := last
			for k > 0 && nodes[k].p.h < 2.6 {
				k--
			}
			q := nodes[k]
			side := math.Copysign(1, rng.Float64()-0.5)
			f3 := nodes[last].p.plus(mgP{side * (1.5 + rng.Float64()*1.4), (rng.Float64() - 0.5) * 2, 0})
			f3.h = 0
			fs := seed + 11
			fn := mgCurve(q.p, q.p.plus(mgP{side * 0.5, 0, -0.7}), f3.plus(mgP{0, 0, 1.1}), f3, mgRootRad(q.r*0.62, 0.42), 0.15, fs)
			add(mgPart{kind: mgRoot, nodes: fn, z: f3.z + 0.01, tone: depth(f3.z), merge: 0.8, h0: s.h0, foot: true, parent: ri, seed: fs})
		}
	}

	// Branches up into the crown, and roots hanging from it.
	for i, b := range [][2]float64{{-11, 9}, {11, 10}, {1, 14}, {-17, 3}, {16, 4}} {
		h0 := L.ht - 6 + float64(i)*1.3
		tx, _ := trunkAt(h0)
		o := mgP{tx, 0, h0}
		end := mgP{lean + b[0]*(0.9+rng.Float64()*0.2), (rng.Float64() - 0.5) * 6, L.ht + b[1]}
		seed := sd + 300 + uint32(i)*7
		nodes := mgCurve(o, o.plus(mgP{b[0] * 0.15, 0, 3}), end.plus(mgP{-b[0] * 0.3, 0, -2}), end,
			func(t float64) float64 { return 1.35 - 0.65*t + 0.5*math.Max(0, 1-t/0.12) }, 0.6, seed)
		add(mgPart{kind: mgBranch, nodes: nodes, z: 0.01, merge: 2, parent: -1, seed: seed})
	}
	// The aerial roots drop from the branches under the crown: one down in
	// the water and arching out at its foot, and a young one, not down yet.
	for i, young := range [mangroveVariants][]bool{{false, true}, {false}, {true, false}, {false, true}}[v] {
		side := float64(1 - 2*((i+v)%2))
		x := side*(9+rng.Float64()*6) + lean*0.7
		z := (rng.Float64() - 0.5) * 8
		top := mgP{x, z, 47 + rng.Float64()*3}
		end := mgP{x + side*(5+rng.Float64()*4), z + (rng.Float64()-0.5)*3, 0}
		if young {
			end = mgP{x + side*2.5, z, 17 + rng.Float64()*9}
		}
		dh := top.h - end.h
		rad := func(t float64) float64 {
			r := 0.5 - 0.08*t
			if young {
				return r - 0.1*smooth(clamp01((t-0.75)/0.25))
			}
			return r + 0.3*smooth(clamp01((t-0.88)/0.12))
		}
		seed := sd + 400 + uint32(i)*13
		p1, p2 := top.plus(mgP{side * 2.5, 0, -dh * 0.35}), end.plus(mgP{-side * 1.2, 0, dh * 0.4})
		if young {
			p2 = end.plus(mgP{-side * 0.4, 0, dh * 0.4})
		}
		nodes := mgCurve(top, p1, p2, end, rad, 0.75, seed)
		add(mgPart{kind: mgAerial, nodes: nodes, z: end.z, tone: depth(end.z), merge: 3, h0: top.h, foot: !young, young: young, parent: -1, seed: seed})
	}

	// Pencil roots poking up from the mud round the roots' feet.
	var taken, feet [][2]float64
	for _, p := range L.parts {
		if p.foot {
			e := p.nodes[len(p.nodes)-1].p
			taken = append(taken, [2]float64{e.x, e.z})
			if p.kind == mgRoot {
				feet = append(feet, [2]float64{e.x, e.z})
			}
		}
	}
	free := func(x, z, d float64) bool {
		for _, t := range taken {
			if math.Hypot(t[0]-x, (t[1]-z)*0.8) < d {
				return false
			}
		}
		return true
	}
	for i, np := 0, 0; np < 11 && i < 400; i++ {
		ft := feet[rng.IntN(len(feet))]
		p := mgP{ft[0], ft[1], 0}.plus(mgOut(rng.Float64()*2*math.Pi, 2.5+rng.Float64()*5))
		if math.Abs(p.x) > 37 || p.z > 12 || p.z < -22 || !free(p.x, p.z, 2.2) {
			continue
		}
		h := 1.2 + rng.Float64()*2
		r := 0.24 + rng.Float64()*0.1
		lean := (rng.Float64() - 0.5) * 0.8
		top := p.plus(mgP{lean, 0, h})
		seed := sd + 500 + uint32(np)*5
		nodes := mgCurve(p, p.plus(mgP{0, 0, h * 0.4}), top.plus(mgP{-lean * 0.3, 0, -h * 0.3}), top,
			func(t float64) float64 { return r * (1.2 - 0.55*t) }, 0.06, seed)
		add(mgPart{kind: mgPneu, nodes: nodes, z: p.z, tone: depth(p.z), foot: true, parent: -1, seed: seed})
		taken = append(taken, [2]float64{p.x, p.z})
		np++
	}

	// The crown: the blobs of the first drawing, moved about a little, over
	// the top of the trunk.
	L.wide = [mangroveVariants]float64{1, 1.07, 0.94, 1.03}[v]
	lift := [mangroveVariants]float64{0, -1.5, 2.5, 0.5}[v]
	shift := lean * 0.85
	blobs := []blob{{13, 25, 8.5}, {39, 26, 8.5}, {26, 18, 10.5}, {18, 18, 8}, {34, 17.5, 8}, {26, 26, 9}, {8, 29, 5}, {44, 30, 5}}
	switch v {
	case 2: // taller, a crown on top
		blobs = append(blobs, blob{28, 12, 6.5})
	case 3: // lopsided: low on the left
		blobs[7] = blob{4, 32, 4.5}
	}
	for _, b := range blobs {
		r := b.r * mangroveS * (0.9 + rng.Float64()*0.2)
		x := mgBX + shift + (b.x-26)*mangroveS*L.wide + (rng.Float64()-0.5)*4
		y := mgBase - (61-b.y)*mangroveS - lift + (rng.Float64()-0.5)*3
		L.crown = append(L.crown, blob{clamp(x, r+3, mgW-r-3), math.Max(y, r+3), r})
	}
	L.cx, L.cy = mgBX+shift, mgBase-(61-22)*mangroveS-lift

	for f := range 2 {
		for _, p := range L.partsFor(f == 1) {
			if !p.foot {
				continue
			}
			k := len(p.nodes) - 1
			if p.kind == mgPneu {
				k = 0
			}
			e := p.nodes[k].p
			L.feet[f] = append(L.feet[f], mgFoot{x: math.Round((e.sx()-mgBX)*fk) / fk, y: math.Round((e.sy()-mgBase)*fk) / fk,
				r: p.nodes[k].r, pneu: p.kind == mgPneu})
		}
	}
	return L
}

// partsFor is the wood of a full grown mangrove, or of one felled: its
// trunk cut, the branches and what hung from them gone, and the roots that
// left the trunk above the cut.
func (L *mgLayout) partsFor(full bool) []*mgPart {
	var out []*mgPart
	gone := make([]bool, len(L.parts))
	for i := range L.parts {
		p := &L.parts[i]
		if !full {
			switch {
			case p.kind == mgBranch, p.kind == mgAerial, p.parent >= 0 && gone[p.parent], p.kind == mgRoot && (p.h0 > L.cut-3 || p.top() > L.cut+1):
				gone[i] = true
				continue
			case p.kind == mgTrunk:
				q := *p
				q.nodes = nil
				for _, n := range p.nodes {
					if n.p.h <= L.cut {
						q.nodes = append(q.nodes, n)
					}
				}
				p = &q
			}
		}
		out = append(out, p)
	}
	return out
}

// mgDraw is a mangrove being drawn.
type mgDraw struct {
	f    *flo
	c    *canvas
	wet  bool
	soft []bool // what lies on the ground at the roots' feet: no outline round it
}

// genMangrove draws variant v of the mangrove, full grown or felled, in
// the water (wet) or on the sand. The anchor is its ground point.
func genMangrove(v int, full, wet bool) (*canvas, float64, float64) {
	return genMangroveOf(mangroveLayoutOf(v), v, full, wet)
}

// genMangroveOf draws variant v laid out as L.
func genMangroveOf(L *mgLayout, v int, full, wet bool) (*canvas, float64, float64) {
	f := newFlo(mgW, mgH)
	d := &mgDraw{f: f, c: f.c, wet: wet, soft: make([]bool, mgW*mgH*HDDensity*HDDensity)}
	parts := L.partsFor(full)
	if wet { // fewer pencil roots show above the water
		kept := parts[:0:0]
		n := 0
		for _, p := range parts {
			if p.kind == mgPneu {
				if n++; n%3 != 0 {
					continue
				}
			}
			kept = append(kept, p)
		}
		parts = kept
	}
	d.shadows(parts)
	slices.SortStableFunc(parts, func(a, b *mgPart) int { return cmp.Compare(a.z, b.z) })
	// Far to near, each foot's water or mound straight after its root, so
	// what is nearer lies over it.
	for _, p := range parts {
		d.part(p)
		if p.kind == mgTrunk && !full { // felled: the pale cut on the stump
			top := p.nodes[len(p.nodes)-1]
			d.c.ellipse(top.p.sx()*fk, top.p.sy()*fk, top.r*fk, top.r*0.45*fk, func(x, y int, nx, ny float64) color.RGBA {
				return woodEndHD(x, y, nx, ny, 4, 86+uint32(v))
			})
		}
		if !p.foot {
			continue
		}
		n, b := p.nodes[len(p.nodes)-1], p.nodes[max(0, len(p.nodes)-3)]
		if p.kind == mgPneu {
			n, b = p.nodes[0], p.nodes[2]
		}
		if wet {
			d.waterline(n, b, p.kind == mgPneu)
		} else {
			d.mound(n, p.kind == mgPneu, p.seed)
		}
	}
	if full {
		d.crown(L, v)
	}
	d.outline(0.66)
	return d.c, mgBX * fk, mgBase * fk
}

// put colours pixel (x, y) with what lies on the ground (soft) or not.
func (d *mgDraw) put(x, y int, col color.RGBA, soft bool) {
	if d.c.in(x, y) {
		d.c.put(x, y, col)
		d.soft[y*d.c.w+x] = soft
	}
}

// outline draws the dark line round the tree, but not round what lies on
// the ground at its feet, which stays soft against the ground.
func (d *mgDraw) outline(strength float64) {
	c := d.c
	before := slices.Clone(c.px)
	c.outline(strength)
	if c.ol == nil {
		return
	}
	for y := range c.h {
		for x := range c.w {
			i := y*c.w + x
			if !c.ol[i] {
				continue
			}
			hard := false
			for _, o := range [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
				if nx, ny := x+o[0], y+o[1]; c.in(nx, ny) && before[ny*c.w+nx].A >= 128 && !d.soft[ny*c.w+nx] {
					hard = true
					break
				}
			}
			if !hard {
				c.px[i], c.ol[i] = before[i], false
			}
		}
	}
}

// genMangroveHD is the first look of the mangrove, on the sand, for the
// plants' sheet and the scale sheet; the world shows every look
// (mangroveSprite).
func genMangroveHD(full bool) (*canvas, float64, float64) { return genMangrove(0, full, false) }

// shadows lays the shadows of the wood on the ground before it is drawn:
// each root's cast away from the sun under its arch, darker the lower it
// is, and a dark spot where each one goes in.
func (d *mgDraw) shadows(parts []*mgPart) {
	c := d.c
	a := make([]uint8, c.w*c.h)
	splat := func(cx, cy, rx, ry, al float64) {
		for y := max(1, int(cy-ry)); y <= min(c.h-2, int(cy+ry)+1); y++ {
			for x := max(1, int(cx-rx)); x <= min(c.w-2, int(cx+rx)+1); x++ {
				nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
				if nx*nx+ny*ny <= 1 {
					a[y*c.w+x] = max(a[y*c.w+x], uint8(al))
				}
			}
		}
	}
	for _, p := range parts {
		if p.kind == mgBranch {
			continue
		}
		for i, n := range p.nodes {
			h := n.p.h
			if i%2 == 1 || h > 32 || p.kind == mgAerial && h > 14 {
				continue
			}
			sx, sy := n.p.sx()+h*0.5, mgBase+n.p.z*mgGY+h*0.16
			splat(sx*fk, sy*fk, n.r*fk*1.1+0.5, n.r*fk*0.55+1, 62*(1-h/36))
		}
		if p.foot {
			n := p.nodes[len(p.nodes)-1]
			if p.kind == mgPneu {
				n = p.nodes[0]
			}
			splat((n.p.sx()+n.r*0.9)*fk, (n.p.sy()+0.3)*fk, (n.r*1.8+0.9)*fk, 0.75*fk, 84)
		}
	}
	col := hex(0x2c2216)
	if d.wet {
		col = hex(0x082230)
	}
	for i, al := range a {
		if al > 0 && c.px[i].A == 0 {
			c.px[i] = withAlpha(col, al)
		}
	}
}

// part draws one piece of wood: each pixel takes the nearest point of the
// spine, so it knows how far along and across the wood it lies, and is lit
// as a round stem there. Where it lies over wood drawn before, its edge is
// darkened, but not where it grows out of that wood.
func (d *mgDraw) part(p *mgPart) {
	c := d.c
	n := len(p.nodes)
	if n < 2 {
		return
	}
	type sn struct{ x, y, r, h, s float64 }
	sp := make([]sn, n)
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for i, nd := range p.nodes {
		sp[i] = sn{nd.p.sx() * fk, nd.p.sy() * fk, nd.r * fk, nd.p.h, 0}
		if i > 0 {
			sp[i].s = sp[i-1].s + math.Hypot(sp[i].x-sp[i-1].x, sp[i].y-sp[i-1].y)/fk
		}
		x0, y0 = math.Min(x0, sp[i].x-sp[i].r), math.Min(y0, sp[i].y-sp[i].r)
		x1, y1 = math.Max(x1, sp[i].x+sp[i].r), math.Max(y1, sp[i].y+sp[i].r)
	}
	bx0, by0 := max(0, int(math.Floor(x0))-1), max(0, int(math.Floor(y0))-1)
	bx1, by1 := min(c.w-1, int(math.Ceil(x1))+1), min(c.h-1, int(math.Ceil(y1))+1)
	if bx1 < bx0 || by1 < by0 {
		return
	}
	w := bx1 - bx0 + 1
	type hit struct {
		d, u, v float64
		seg     int
	}
	hits := make([]hit, w*(by1-by0+1))
	for i := range hits {
		hits[i].d = math.Inf(1)
	}
	for i := 0; i+1 < n; i++ {
		a, b := sp[i], sp[i+1]
		ex, ey := b.x-a.x, b.y-a.y
		ee := ex*ex + ey*ey
		if ee < 1e-9 {
			continue
		}
		ln := math.Sqrt(ee)
		px, py := -ey/ln, ex/ln
		r := math.Max(a.r, b.r) + 1
		for y := max(by0, int(math.Min(a.y, b.y)-r)); y <= min(by1, int(math.Max(a.y, b.y)+r)); y++ {
			for x := max(bx0, int(math.Min(a.x, b.x)-r)); x <= min(bx1, int(math.Max(a.x, b.x)+r)); x++ {
				fx, fy := float64(x)+0.5, float64(y)+0.5
				u := clamp01(((fx-a.x)*ex + (fy-a.y)*ey) / ee)
				dx, dy := fx-a.x-ex*u, fy-a.y-ey*u
				rr := lerpf(a.r, b.r, u)
				dd := math.Hypot(dx, dy) - rr
				if k := (y-by0)*w + x - bx0; dd < hits[k].d {
					hits[k] = hit{dd, u, clamp((dx*px+dy*py)/rr, -1, 1), i}
				}
			}
		}
	}
	total := sp[n-1].s
	for y := by0; y <= by1; y++ {
		for x := bx0; x <= bx1; x++ {
			ht := hits[(y-by0)*w+x-bx0]
			if ht.d > 0 {
				continue
			}
			a, b := sp[ht.seg], sp[ht.seg+1]
			ex, ey := b.x-a.x, b.y-a.y
			ln := math.Hypot(ex, ey)
			along := lerpf(a.s, b.s, ht.u)
			col := d.shade(p, x, y, along, total, ht.v, -ey/ln, ex/ln, lerpf(a.h, b.h, ht.u), lerpf(a.r, b.r, ht.u)/fk)
			if ht.d > -1.05 && c.opaque(x, y) && along > p.merge {
				col = mix(col, colInk, 0.3)
			}
			d.put(x, y, col, false)
		}
	}
}

// shade colours a pixel of wood p: along art pixels from its start (of
// total), v across it (-1 to 1) on the side (nx, ny) points to, h above
// the ground, where the wood is r thick.
func (d *mgDraw) shade(p *mgPart, x, y int, along, total, v, nx, ny, h, r float64) color.RGBA {
	sd := p.seed
	cz := math.Sqrt(math.Max(0, 1-v*v))
	l := nx*v*lightDir[0] + ny*v*lightDir[1] + cz*lightDir[2]
	t := 0.13 + 0.82*math.Max(0, l) + p.tone
	if p.kind == mgTrunk || p.kind == mgBranch {
		t -= 0.1 * clamp01((h-34)/12) // in the shade of the crown
	}
	side := nx*lightDir[0] + ny*lightDir[1]
	lit := v
	if side < 0 {
		lit = -v
	}
	if math.Abs(side) > 0.18 {
		switch {
		case lit > 0.5 && lit < 0.84:
			t += 0.12 // a highlight along the lit side
		case lit < -0.8:
			t += 0.06 // light thrown back from the ground
		}
	}
	acr := v * r // across, in art pixels
	// Fine rings round it, broken, each with a lit lip.
	ph := along/1.55 + 0.4*vnoise(along*0.5, 0.5, sd) + 0.15*cz
	switch fr := ph - math.Floor(ph); {
	case fr < 0.19 && vnoise(along*0.8, acr*0.9, sd+1) > 0.3:
		t -= 0.13
	case fr < 0.34 && lit > 0:
		t += 0.04
	}
	if p.kind == mgTrunk { // long fissures up the trunk
		if f := math.Abs(vnoise(acr*1.5, along*0.06, sd+20) - 0.5); f < 0.035 {
			t -= 0.22
		} else if f < 0.065 {
			t -= 0.07
		}
	}
	if vnoise(acr*2.2, along*0.12, sd+2) < 0.2 { // fine cracks along it
		t -= 0.07
	}
	t += (vnoise(along*0.6, acr*1.5, sd+3) - 0.5) * 0.12
	young := p.kind == mgPneu || p.young && along > total-6
	var col color.RGBA
	if young {
		k := clamp01((along - (total - 1.2)) / 1.2) // the tip, paler
		col = barkPick(mgYoung, t+0.12*k, x, y, sd+4)
	} else {
		col = barkPick(mgBarkGrey, t, x, y, sd+4)
		if red := smooth(clamp01((vnoise(along*0.16, acr*0.4+3, sd+5) - 0.42) / 0.26)); red > 0 {
			col = mix(col, barkPick(mgBarkRed, t, x, y, sd+4), red*0.55)
		}
	}
	// Lenticels: small pale dots.
	if ca, cv := along/1.15, acr/0.8+50; math.Abs(v) < 0.8 && !young || p.kind == mgPneu {
		ia, iv := math.Floor(ca), math.Floor(cv)
		if hashf(int(ia), int(iv), sd+6) < 0.13 && math.Abs(ca-ia-0.5) < 0.16 && math.Abs(cv-iv-0.5) < 0.22 {
			col = mix(mgLenticel, col, 0.2+0.45*(1-clamp01(t)))
		}
	}
	// What the tide leaves: dark and wet just over the water, algae and
	// barnacles up to a ragged line above that.
	if h < 6.5 {
		top := 3.4 + 1.6*vnoise(along*0.35, 9.5, sd+7) + 0.8*(vnoise(acr*1.3, along*0.2, sd+8)-0.5)
		wet := 0.6 + 0.6*vnoise(along*0.5, 11.5, sd+9)
		if p.kind == mgPneu {
			top, wet = top*0.45, wet*0.5
		}
		switch {
		case h < wet:
			col = mix(col, mgWetCol, 0.3)
		case h < top:
			k := clamp01((top-h)/1.4) * (0.3 + 0.4*vnoise(along*0.9, acr*1.1, sd+10))
			col = mix(col, barkPick(mgAlgae, t+0.05, x, y, sd+11), k)
		}
		if ca, cv := along/0.8, acr/0.7+50; h > wet*0.6 && h < top-0.2 && math.Abs(v) < 0.86 && p.kind != mgPneu {
			ia, iv := math.Floor(ca), math.Floor(cv)
			if hashf(int(ia), int(iv), sd+12) < 0.3 {
				dx, dy := (ca-ia-0.5)/0.32, (cv-iv-0.5)/0.36
				if q := dx*dx + dy*dy; q < 1 {
					b := t + 0.3 - q*0.35
					col = mgBarnacle.at(int(clamp01(b) * float64(len(mgBarnacle))))
				}
			}
		}
	}
	return col
}

// crown draws the leaves over the top of the trunk, as the first drawing
// had them: thick, glossy leaves in a broad low dome, a few old ones
// turning yellow, and the seedlings hanging under it like green pencils.
func (d *mgDraw) crown(L *mgLayout, v int) {
	f := d.f
	rng := rand.New(rand.NewPCG(uint64(v)+86, 31))
	leafClusterHD(f.c, scaledBlobs(L.crown), L.cx*fk, L.cy*fk, 21*mangroveS*L.wide*fk, 13*mangroveS*fk, mangroveLeaf, 866+uint32(v), leafLook{4.8, 4.6, 2})
	// at is a point of the first drawing: dx from the trunk, yOld down from
	// its top.
	at := func(dx, yOld float64) (float64, float64) {
		return L.cx + dx*mangroveS*L.wide, L.cy + (yOld-22)*mangroveS
	}
	for range 24 {
		x, y := at(rng.Float64()*38-19, 12+rng.Float64()*19)
		if f.c.opaque(int(x*fk), int(y*fk)) {
			leafHD(f.c, x*fk, y*fk, 3.6, 1.7, 0.5+rng.Float64()*2, mkRamp(0x7c7c24, 0xa4a030, 0xc8c044, 0xe4dc6c), 0.55)
		}
	}
	for i := range 16 {
		x, y := at(float64(i)*2.4+rng.Float64()*1.2-17, 30+3.5*math.Sin(float64(i)*1.3)+rng.Float64()*2)
		if !f.c.opaque(int(x*fk), int((y-1.5)*fk)) {
			continue
		}
		pr := aCap(x, y, x+0.2, y+4+rng.Float64()*1.5, 0.3, 0.42)
		f.q.fill(pr, shd(mangroveLeaf, pr, 0.8, 0.22))
	}
}

// mound heaps a little mound of wet sand round the foot of a root at n (b
// a little way up it), over the root, its shaded side away from the sun.
func (d *mgDraw) mound(n mgNode, pneu bool, seed uint32) {
	c := d.c
	rx, ry := (n.r*1.45+0.8)*fk, (n.r*0.4+0.5)*fk
	if pneu {
		rx, ry = (n.r*1.5+0.5)*fk, 0.45*fk
	}
	cx, cy := n.p.sx()*fk, n.p.sy()*fk+ry*0.25
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			q := nx*nx + ny*ny
			if q > 1-0.25*(hashf(x, y, seed)-0.5) || !c.in(x, y) {
				continue
			}
			l := lum(nx*0.8, ny*0.9-0.25)*0.75 + 0.12
			if q > 0.6 && ny > 0 {
				l -= 0.12 // where it meets the sand, in its own shade
			}
			d.put(x, y, mgMound.pick(l+(hashf(x, y, seed+1)-0.5)*0.12, x, y), true)
		}
	}
}

// waterline stands the foot of a root at n (b a little way up it) in the
// water: under the surface it goes on dimly, and a ring of water stands
// round it, bright where it is in front of the root.
func (d *mgDraw) waterline(n, b mgNode, pneu bool) {
	c := d.c
	cx, cy := n.p.sx()*fk, n.p.sy()*fk
	ux, uy := n.p.sx()-b.p.sx(), n.p.sy()-b.p.sy()
	if l := math.Hypot(ux, uy); l > 1e-6 {
		ux, uy = ux/l, uy/l
	} else {
		ux, uy = 0, 1
	}
	if pneu {
		ux, uy = 0, 1
	}
	r := n.r * fk * 0.9
	under := mix(mgWetCol, hex(0x2c6a80), 0.55)
	ex, ey := cx+ux*1.4*fk, cy+uy*1.4*fk
	for y := int(cy - r - 1); y <= int(ey+r+1); y++ {
		for x := int(math.Min(cx, ex) - r - 1); x <= int(math.Max(cx, ex)+r+1); x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if fy < cy || c.opaque(x, y) {
				continue
			}
			u := clamp01(((fx-cx)*(ex-cx) + (fy-cy)*(ey-cy)) / ((ex-cx)*(ex-cx) + (ey-cy)*(ey-cy)))
			px, py := cx+(ex-cx)*u, cy+(ey-cy)*u
			if math.Hypot(fx-px, fy-py) <= r*(1-0.3*u) {
				d.put(x, y, withAlpha(under, uint8(150*(1-0.6*u))), true)
			}
		}
	}
	rx := (n.r + 0.75) * fk
	if pneu {
		rx = (n.r + 0.5) * fk
	}
	ry := rx * 0.38
	ring := color.RGBA{0xd2, 0xea, 0xee, 190}
	steps := int(rx*8) + 8
	for k := range steps {
		a := 2 * math.Pi * float64(k) / float64(steps)
		x, y := int(math.Floor(cx+math.Cos(a)*rx)), int(math.Floor(cy+math.Sin(a)*ry))
		switch {
		case math.Sin(a) >= 0:
			d.put(x, y, ring, true) // in front of the root
		case !c.opaque(x, y):
			d.put(x, y, withAlpha(ring, 110), true) // behind it, beside it
		}
	}
}

// --- in the game -----------------------------------------------------------------------

// mangroveArt is the looks of the mangrove: by variant, felled or full.
type mangroveArt [mangroveVariants][2]sprite

// mangroveSets is the mangroves on the sand and in the water, drawn when
// first wanted.
var mangroveSets [2]lazy[mangroveArt]

func init() {
	for w := range mangroveSets {
		mangroveSets[w].gen = func() mangroveArt {
			var a mangroveArt
			inParallel(2*mangroveVariants, func(i int) {
				a[i/2][i%2] = trimmed(genMangrove(i/2, i%2 == 1, w == 1))
			})
			return a
		}
	}
}

func boolIdx(b bool) int {
	if b {
		return 1
	}
	return 0
}

// mangroveVariant is the look of the mangrove on tile (tx, ty): never the
// same as a neighbour's in the same block of four tiles by four.
func mangroveVariant(tx, ty int) int {
	return int((uint32(3*tx+2*ty) + sim.HashTile(tx>>2, ty>>2)) % mangroveVariants)
}

// mangroveSprite is the mangrove on tile (tx, ty), full grown or felled,
// standing in the water or not.
func mangroveSprite(tx, ty int, full, wet bool) sprite {
	return mangroveSets[boolIdx(wet)].get()[mangroveVariant(tx, ty)][boolIdx(full)]
}

// mgRipples is the water about a root's foot: by thickness, frames of its
// rings spreading, and its reflection under it.
var mgRipples = lazy[[2][3]sprite]{gen: func() (out [2][3]sprite) {
	for big := range 2 {
		for fr := range 3 {
			out[big][fr] = mkUp(genMgRipple(big == 1, fr))
		}
	}
	return out
}}

// genMgRipple is frame fr of the water about the foot of a root, thick or
// not: rings spreading over the water, and the root's reflection in it,
// broken by the ripples.
func genMgRipple(big bool, fr int) (*canvas, float64, float64) {
	c := hdCanvas(16, 10)
	cx, cy := 8*fk, 3*fk
	r := 0.6
	if big {
		r = 0.95
	}
	refl := color.RGBA{12, 30, 34, 0}
	n := 13
	for row := 1; row <= n; row++ {
		if (row+fr)%3 == 0 {
			continue // broken by a ripple
		}
		k := 1 - float64(row)/float64(n+1)
		y := cy + float64(row)
		hw := r * fk * (0.85 + 0.25*math.Sin(float64(row)*1.3+float64(fr)*2.1))
		off := math.Sin(float64(row)*0.9+float64(fr)*2) * 1.2
		for x := int(math.Floor(cx - hw + off)); x <= int(math.Ceil(cx+hw+off)); x++ {
			c.put(x, int(y), withAlpha(refl, uint8(120*k)))
		}
	}
	ph := float64(fr) / 3
	for ring := range 2 {
		rr := (r + 0.8 + ph*1.7) * (1 + 0.8*float64(ring))
		al := 150 * (1 - 0.55*ph) * (1 - 0.5*float64(ring))
		rx, ry := rr*fk, rr*fk*0.4
		for k := range 72 {
			if (k/6+ring+fr)%4 == 0 {
				continue // broken rings
			}
			a := 2 * math.Pi * float64(k) / 72
			x, y := int(math.Floor(cx+math.Cos(a)*rx)), int(math.Floor(cy+math.Sin(a)*ry))
			if c.in(x, y) {
				c.put(x, y, withAlpha(rippleCol, uint8(al)))
			}
		}
	}
	return c, cx, cy
}

// mangroveFeet is where the roots of the mangrove on tile (tx, ty) meet
// the ground, full grown or felled.
func mangroveFeet(tx, ty int, full bool) []mgFoot {
	return mangroveLayoutOf(mangroveVariant(tx, ty)).feet[boolIdx(full)]
}

// drawMangroveWater draws, under mangrove e, the rings of water spreading
// from each root that stands in the sea and its reflection.
func (g *Game) drawMangroveWater(dst drawTarget, e protocol.EntityView, d drawable) {
	rs := mgRipples.get()
	for i, ft := range mangroveFeet(int(e.TX), int(e.TY), e.Ready) {
		if ft.pneu {
			continue // a pencil root has its ring of water in the drawing
		}
		wx := float64(e.TX) + 0.5 + ft.x/TileW
		wy := float64(e.TY) + 0.5 + ft.y/TileH
		if !g.tileAt(int(math.Floor(wx)), int(math.Floor(wy))).Sea() {
			continue
		}
		big := boolIdx(ft.r >= 0.75)
		fr := (g.frame/20 + int(e.ID) + i*5) % 3
		drawAt(dst, rs[big][fr], d.sx+ft.x, d.sy+ft.y, 1, 1, 0, nil)
	}
}

// top is the height of the highest point of p.
func (p *mgPart) top() float64 {
	h := 0.0
	for _, n := range p.nodes {
		h = math.Max(h, n.p.h)
	}
	return h
}
