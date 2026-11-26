package sim

import (
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
)

// The designed archipelago of StyleIsles. The home island sits in the
// middle: a gentle meadow with a hill on one side, a river off the hill and
// a sheltered bay. Round it lie the large islands, each with a character of
// its own (a jungle, a river delta, the monkeys' island with its crag, a
// snowy range, a desert of mesas, a volcano), the gentler ones nearer home.
// Medium islands and islets fill the sea between, some of them a narrow
// strait away from a large one, and out towards the edge an atoll rings its
// lagoon.
//
// Each island's coast is a body pushed in and out by a few harmonics, with
// peninsulas reaching out, bays cut in and coves along it; inside, its
// height rises to hills, ridges, mesas or a cone by its character. The rest
// is laid on top: the coasts (coast.go), the height levels and the cliffs
// between them (heights.go), and the rivers and lakes (rivers.go).

// Kinds of island only worlds made since StyleIsles have.
const (
	IslandMonkey IslandKind = numIslandKinds + iota // jungle round a crag of rock: the monkeys' island
	IslandAtoll                                     // a ring of sand and palms round a lagoon
	numIsleKinds
)

// isleRole is the part an island plays in the layout.
type isleRole uint8

const (
	roleHome isleRole = iota
	roleLarge
	roleMedium
	roleIslet
	roleAtoll
)

// Layout tuning. Sizes are shares of the world's size, up to 1024 tiles;
// past that, islands grow with the square root of the size and there are
// more of them, so a bigger world has more islands and wider seas, not just
// bigger ones.
const (
	isleHomeR      = 0.13 // the home island's radius
	isleLargeR     = 0.093
	isleLargeRMax  = 0.126
	isleMediumR    = 0.03
	isleMediumRMax = 0.056
	isleIsletR     = 0.007
	isleIsletRMax  = 0.017
	isleAtollR     = 0.034
	isleReach      = 1.62 // of the radius: how far an island's land can reach, peninsulas and all
	isleRoom       = 1.2  // of the radius: how far apart islands are laid out
	isleGapLarge   = 14   // tiles of sea at least between two islands' reach
	isleGapSmall   = 5
	isleStrait     = 0.3 // share of medium islands set a strait away from a large one
	isleEdge       = 18  // tiles of sea at least between land and the edge of the world
)

// largeKinds are the large islands in the order they are placed: one of
// each, then (in bigger worlds) a second round without the volcano and the
// monkeys' island, which are one of a kind.
var largeKinds = []IslandKind{IslandForest, IslandMarsh, IslandMonkey, IslandSnow, IslandDesert, IslandAsh}
var largeKindsMore = []IslandKind{IslandForest, IslandSnow, IslandDesert, IslandMarsh, IslandMeadow, IslandForest}

// isleBand is how far from the middle a large island of kind k lies, as a
// share of half the world: the gentle ones nearer home.
func isleBand(k IslandKind) (lo, hi float64) {
	switch k {
	case IslandForest, IslandMarsh, IslandMeadow:
		return 0.42, 0.7
	case IslandMonkey:
		return 0.5, 0.78
	}
	return 0.6, 0.92
}

// isle is one island planned for a StyleIsles world.
type isle struct {
	role       isleRole
	kind       IslandKind
	x, y, r    float64
	rot, elong float64       // stretched elong times along angle rot
	harm       [5][2]float64 // the coast's harmonics 2 to 6: amplitude, phase
	arms       []isleArm     // peninsulas
	bays       []isleBay
	hill       [3]float64 // the home island's hill, or the monkeys' crag: x, y, radius
	flat       [3]float64 // ground kept low and level: round the spawn, or the pyramid's plain; x, y, radius
	seed       uint32
}

// isleArm is a peninsula, from its root to its tip.
type isleArm struct{ x0, y0, x1, y1, w0, w1 float64 }

// isleBay is a bay cut into the coast: round, or a fjord from its mouth
// (x0, y0) in to (x1, y1).
type isleBay struct{ x0, y0, x1, y1, r float64 }

// reach is how far from its middle the island's land can lie.
func (s *isle) reach() float64 { return s.r * s.elong * isleReach }

// room is how much sea round its middle an island keeps to itself when the
// islands are laid out: less than its reach, as a peninsula only reaches
// out on one side; one that comes near another island makes a strait.
func (s *isle) room() float64 { return s.r * s.elong * isleRoom }

// isleScale is the factor island radii are multiplied by past 1024 tiles,
// and how many more islands of each role there are.
func isleScale(m float64) (rs, count float64) {
	rs = 1
	if m > 1024 {
		rs = math.Sqrt(1024 / m)
	}
	count = (m / 1024) * (m / 1024) * rs * rs
	return rs, count
}

// planIsles lays out the islands of a world of W by H.
func planIsles(W, H int, seed uint32) []isle {
	m := float64(min(W, H))
	rs, cf := isleScale(m)
	rng := rand.New(rand.NewPCG(uint64(seed), 0x15e5))
	cx, cy := float64(W)/2, float64(H)/2
	rad := func(lo, hi float64) float64 { return (lo + (hi-lo)*rng.Float64()) * m * rs }
	var isles []isle
	fits := func(s *isle, gap float64, partner int) bool {
		if re := s.reach(); s.x-re < isleEdge || s.y-re < isleEdge || s.x+re > float64(W)-isleEdge || s.y+re > float64(H)-isleEdge {
			return false
		}
		re := s.room()
		for i := range isles {
			o := &isles[i]
			g := gap
			if i == partner {
				g = -0.25 * min(re, o.room()) // a strait: as close as their shores allow
			}
			if math.Hypot(o.x-s.x, o.y-s.y) < o.room()+re+g {
				return false
			}
		}
		return true
	}
	place := func(role isleRole, kind IslandKind, rlo, rhi, dlo, dhi, gap float64) bool {
		for try := 0; try < 300; try++ {
			shrink := 1 - 0.4*float64(try)/300
			r := max(rad(rlo, rhi)*shrink, 3)
			a := rng.Float64() * 2 * math.Pi
			d := (dlo + (dhi-dlo)*rng.Float64()) * m / 2
			s := newIsle(rng, role, kind, cx+math.Cos(a)*d, cy+math.Sin(a)*d, r)
			if fits(&s, gap, -1) {
				isles = append(isles, s)
				return true
			}
		}
		return false
	}

	isles = append(isles, newIsle(rng, roleHome, IslandMeadow, cx+(rng.Float64()-0.5)*m*0.03, cy+(rng.Float64()-0.5)*m*0.03, max(14, isleHomeR*m*rs)))
	kinds := append([]IslandKind(nil), largeKinds...)
	for n := len(largeKinds); n < int(math.Round(6*cf)); n++ {
		kinds = append(kinds, largeKindsMore[(n-len(largeKinds))%len(largeKindsMore)])
	}
	for _, k := range kinds {
		lo, hi := isleBand(k)
		if !place(roleLarge, k, isleLargeR, isleLargeRMax, lo, hi, isleGapLarge) {
			place(roleLarge, k, isleLargeR*0.6, isleLargeR, 0.3, 0.96, isleGapSmall) // a crowded world: smaller, anywhere
		}
	}
	for range max(1, int(math.Round(cf*1.4))) {
		if m >= 192 {
			place(roleAtoll, IslandAtoll, isleAtollR, isleAtollR*1.2, 0.62, 0.95, isleGapLarge)
		}
	}
	nLarge := len(isles)
	for range max(6, int(math.Round(13*cf))) {
		far := 0.0
		if rng.Float64() < isleStrait && nLarge > 1 {
			// A strait: close beside a large island.
			o := isles[1+rng.IntN(nLarge-1)]
			for try := 0; try < 60; try++ {
				r := rad(isleMediumR, isleMediumRMax)
				a := rng.Float64() * 2 * math.Pi
				d := o.room() + r*isleRoom*0.9
				s := newIsle(rng, roleMedium, 0, o.x+math.Cos(a)*d, o.y+math.Sin(a)*d, r)
				s.kind = mediumKind(math.Hypot(s.x-cx, s.y-cy)/(m/2), rng)
				if fits(&s, isleGapSmall, slicesIndex(isles, o)) {
					isles = append(isles, s)
					break
				}
			}
			continue
		}
		far = 0.3 + 0.66*math.Sqrt(rng.Float64())
		place(roleMedium, mediumKind(far, rng), isleMediumR, isleMediumRMax, far, far, isleGapSmall+4)
	}
	for range max(4, int(math.Round(46*cf))) {
		far := 0.2 + 0.78*rng.Float64()
		place(roleIslet, mediumKind(far, rng), isleIsletR, isleIsletRMax, far, far, isleGapSmall)
	}
	return isles
}

// slicesIndex is the index of the island at the same place as o.
func slicesIndex(isles []isle, o isle) int {
	for i := range isles {
		if isles[i].x == o.x && isles[i].y == o.y {
			return i
		}
	}
	return -1
}

// mediumKind picks the land of a medium island or an islet by how far out
// it lies (0 at the middle, 1 at the edge).
func mediumKind(far float64, rng *rand.Rand) IslandKind {
	r := rng.Float64()
	switch {
	case far < 0.45:
		return []IslandKind{IslandMeadow, IslandForest, IslandForest}[int(r*3)]
	case far < 0.7:
		return []IslandKind{IslandForest, IslandMarsh, IslandMeadow, IslandDesert}[int(r*4)]
	default:
		return []IslandKind{IslandSnow, IslandDesert, IslandForest, IslandMarsh}[int(r*4)]
	}
}

// newIsle shapes an island of kind k with radius r at (x, y).
func newIsle(rng *rand.Rand, role isleRole, k IslandKind, x, y, r float64) isle {
	s := isle{role: role, kind: k, x: x, y: y, r: r, rot: rng.Float64() * math.Pi, elong: 1, seed: rng.Uint32()}
	wobble, arms, bays, fjords := 0.34, 2+rng.IntN(2), 2+rng.IntN(2), 0
	crescent := role == roleLarge && rng.Float64() < 0.3
	switch {
	case role == roleHome:
		wobble, arms, bays = 0.26, 2, 1 // a sheltered bay for a harbour, and points either side
	case role == roleAtoll:
		wobble, arms, bays = 0.06, 0, 0
	case role == roleIslet:
		wobble, arms, bays = 0.3, 0, 0
	case k == IslandAsh:
		wobble, arms, bays, crescent = 0.1, 1, 0, false // a cone: round, a lava headland
	case k == IslandSnow:
		s.elong = 1.3 + 0.4*rng.Float64() // a range of mountains: long
		bays, fjords = 1, 2+rng.IntN(2)
	case k == IslandDesert:
		wobble = 0.4
	case k == IslandMarsh:
		bays = 3 + rng.IntN(2) // an estuary and creeks
	case role == roleMedium:
		arms, bays = rng.IntN(3), rng.IntN(3)
		s.elong = 1 + 0.5*rng.Float64()
	}
	if s.elong > 1 {
		s.r /= math.Sqrt(s.elong) // the same land, stretched
	}
	for i := range s.harm {
		k := float64(i + 2)
		s.harm[i] = [2]float64{wobble * (0.3 + 0.7*rng.Float64()) / math.Pow(k, 0.8), rng.Float64() * 2 * math.Pi}
	}
	for range arms {
		a := rng.Float64() * 2 * math.Pi
		bend := (rng.Float64() - 0.5) * 0.6
		x0, y0 := s.edge(a, 0.5)
		x1, y1 := s.edge(a+bend, 1.25+0.3*rng.Float64())
		s.arms = append(s.arms, isleArm{x0, y0, x1, y1, s.r * (0.16 + 0.08*rng.Float64()), s.r * (0.04 + 0.05*rng.Float64())})
	}
	for range bays {
		a := rng.Float64() * 2 * math.Pi
		bx, by := s.edge(a, 0.97+0.1*rng.Float64())
		s.bays = append(s.bays, isleBay{bx, by, bx, by, s.r * (0.16 + 0.16*rng.Float64())})
	}
	for range fjords { // long narrow inlets, winding in
		a := rng.Float64() * 2 * math.Pi
		x0, y0 := s.edge(a, 1.1)
		x1, y1 := s.edge(a+(rng.Float64()-0.5)*0.4, 0.35+0.2*rng.Float64())
		s.bays = append(s.bays, isleBay{x0, y0, x1, y1, s.r * (0.045 + 0.03*rng.Float64())})
	}
	if crescent { // one great bay: the island curls round it
		a := rng.Float64() * 2 * math.Pi
		bx, by := s.edge(a, 0.9)
		s.bays = append(s.bays, isleBay{bx, by, bx, by, s.r * (0.45 + 0.12*rng.Float64())})
	}
	switch {
	case role == roleHome:
		a := rng.Float64() * 2 * math.Pi
		s.hill = [3]float64{x + math.Cos(a)*s.r*0.5, y + math.Sin(a)*s.r*0.5, s.r * 0.42}
		s.flat = [3]float64{x, y, s.r * homeCalm}
	case k == IslandDesert && role == roleLarge:
		a := rng.Float64() * 2 * math.Pi
		s.flat = [3]float64{x + math.Cos(a)*s.r*0.25, y + math.Sin(a)*s.r*0.25, max(s.r*0.34, 12)}
	case k == IslandMonkey:
		s.hill = [3]float64{x + (rng.Float64()-0.5)*s.r*0.3, y + (rng.Float64()-0.5)*s.r*0.3, s.r * 0.3}
	}
	return s
}

// edge is the point at f times the coast's distance from the middle, at
// angle a in the island's own frame (before it is stretched and turned).
func (s *isle) edge(a, f float64) (x, y float64) {
	rho := s.r
	for i := range s.harm {
		k := float64(i + 2)
		rho += s.r * s.harm[i][0] * math.Cos(k*a+s.harm[i][1])
	}
	u, v := math.Cos(a)*rho*f*s.elong, math.Sin(a)*rho*f
	c, sn := math.Cos(s.rot), math.Sin(s.rot)
	return s.x + u*c - v*sn, s.y + u*sn + v*c
}

// field is how far into island s the point (x, y) lies, val (above 0 is
// land, 1 its heart), its height h before the island's heights are evened
// out, and for an atoll whether the point is inside the ring.
func (s *isle) field(x, y float64) (val, h float64, inner bool) {
	// The coast wanders: a warp of the whole shape, then harmonics round it.
	ws := s.r * 0.55
	x += (fbm(x/ws+3.1, y/ws, s.seed, 3) - 0.5) * s.r * 0.36
	y += (fbm(x/ws, y/ws+7.7, s.seed+1, 3) - 0.5) * s.r * 0.36
	dx, dy := x-s.x, y-s.y
	c, sn := math.Cos(s.rot), math.Sin(s.rot)
	u, v := (dx*c+dy*sn)/s.elong, -dx*sn+dy*c
	d := math.Hypot(u, v)
	rho := s.r
	if d > 1e-9 {
		ct, st := u/d, v/d
		ck, sk := ct, st // cos and sin of k times the angle, from k = 1
		for i := range s.harm {
			ck, sk = ck*ct-sk*st, sk*ct+ck*st
			a, ph := s.harm[i][0], s.harm[i][1]
			rho += s.r * a * (ck*math.Cos(ph) - sk*math.Sin(ph))
		}
	}
	if s.role == roleAtoll {
		a := math.Atan2(v, u)
		g1, g2 := float64(s.seed%628)/100, float64(s.seed/628%628)/100
		// The ring, wider on its islets (motu) halfway between the gaps.
		motu := math.Max(0, math.Cos(a-g1-math.Pi/2))
		wr := s.r * (0.14 + 0.08*fbm(x/9, y/9, s.seed+2, 2)) * (1 + 1.4*math.Pow(motu, 6))
		wr = math.Max(wr, 2.2)
		val = 1 - math.Abs(d-rho*0.86)/wr
		// Gaps in the ring: the lagoon's mouths.
		if math.Cos(a-g1) > 0.985 || math.Cos(a-g2) > 0.99 {
			val -= 1.2
		}
		val += 0.25 * (fbm(x/4, y/4, s.seed+4, 2) - 0.5)
		inner := d < rho*0.86
		if inner {
			val = math.Max(val, -0.5) // the lagoon is the atoll's own
		}
		return val, 0.2 * clamp(val, 0, 1), inner
	}
	val = 1 - d/rho
	body := val
	for _, a := range s.arms {
		t, dist := segDist(x, y, a.x0, a.y0, a.x1, a.y1)
		w := a.w0 + (a.w1-a.w0)*t
		val = math.Max(val, 0.55*(1-dist/w))
	}
	for _, b := range s.bays {
		_, bd := segDist(x, y, b.x0, b.y0, b.x1, b.y1)
		if q := 1 - bd/b.r; q > 0 {
			val -= 1.3 * smoothstep(q) * math.Max(0.35, 1-body)
		}
	}
	val += 0.1 * (fbm(x/6+1.3, y/6, s.seed+5, 3) - 0.5) // coves and points
	if s.role == roleIslet {
		val += 0.2 * (fbm(x/3, y/3, s.seed+6, 2) - 0.5)
	}

	// The height.
	lv := clamp(val, 0, 1)
	bumps := fbm(x/(s.r*0.38)+11, y/(s.r*0.38), s.seed+7, 3)
	switch {
	case s.role == roleHome:
		hill := math.Max(0, 1-math.Hypot(x-s.hill[0], y-s.hill[1])/s.hill[2])
		h = 0.32*lv + 0.75*math.Pow(hill, 1.1) + 0.1*(bumps-0.5)
	case s.kind == IslandMonkey:
		crag := math.Max(0, 1-math.Hypot(x-s.hill[0], y-s.hill[1])/s.hill[2])
		h = 0.5*lv + 0.3*bumps + 1.1*math.Pow(crag, 0.8)
	case s.kind == IslandSnow:
		ridge := 1 - math.Abs(2*fbm(x/(s.r*0.45)+4, y/(s.r*0.45), s.seed+8, 3)-1)
		h = 0.7*math.Pow(lv, 0.7) + 0.55*ridge*lv
	case s.kind == IslandAsh:
		h = math.Pow(lv, 1.15) + 0.1*(bumps-0.5)
	case s.kind == IslandDesert:
		mesa := smoothstep(clamp((fbm(x/(s.r*0.3)+2, y/(s.r*0.3), s.seed+9, 3)-0.47)/0.08, 0, 1))
		h = 0.35*lv + 0.65*mesa*smoothstep(clamp(lv/0.25, 0, 1))
	case s.kind == IslandMarsh:
		h = 0.6*math.Pow(lv, 1.4) + 0.15*(bumps-0.5)
	default:
		h = math.Pow(lv, 0.9) + 0.6*(bumps-0.5)*smoothstep(clamp(lv/0.3, 0, 1))
	}
	return val, h, false
}

// segDist is the distance from (x, y) to the segment from (x0, y0) to
// (x1, y1), and how far along it (0 to 1) the nearest point lies.
func segDist(x, y, x0, y0, x1, y1 float64) (t, d float64) {
	vx, vy := x1-x0, y1-y0
	l2 := vx*vx + vy*vy
	if l2 > 0 {
		t = clamp(((x-x0)*vx+(y-y0)*vy)/l2, 0, 1)
	}
	return t, math.Hypot(x-x0-t*vx, y-y0-t*vy)
}

// isleMap is the archipelago rasterised: for each tile, the island it
// belongs to (or lies off), how far into it, its height, and whether it is
// inside an atoll's ring.
type isleMap struct {
	isles  []isle
	owner  []int16   // index into isles plus one; 0 for open sea far from any
	val    []float32 // how far into its island: above 0 is land
	h      []float32 // the height; on land evened out to 0 (shore) to 1 (top) per island
	lagoon []bool    // inside an atoll's ring
}

// rasterIsles works out the isleMap of a world of W by H.
func rasterIsles(W, H int, isles []isle) *isleMap {
	n := W * H
	im := &isleMap{isles: isles, owner: make([]int16, n), val: make([]float32, n), h: make([]float32, n), lagoon: make([]bool, n)}
	for i := range im.val {
		im.val[i] = -1
	}
	type patch struct {
		x0, y0, x1, y1 int
		val, h         []float32
		inner          []bool
	}
	patches := make([]patch, len(isles))
	var wg sync.WaitGroup
	jobs := make(chan int)
	for range max(1, runtime.GOMAXPROCS(0)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range jobs {
				s := &isles[k]
				re := s.reach() + 4
				p := patch{x0: max(0, int(s.x-re)), y0: max(0, int(s.y-re)), x1: min(W-1, int(s.x+re)), y1: min(H-1, int(s.y+re))}
				pw, ph := p.x1-p.x0+1, p.y1-p.y0+1
				p.val, p.h, p.inner = make([]float32, pw*ph), make([]float32, pw*ph), make([]bool, pw*ph)
				for ty := p.y0; ty <= p.y1; ty++ {
					for tx := p.x0; tx <= p.x1; tx++ {
						v, h, in := s.field(float64(tx)+0.5, float64(ty)+0.5)
						j := (ty-p.y0)*pw + tx - p.x0
						p.val[j], p.h[j], p.inner[j] = float32(v), float32(h), in
					}
				}
				patches[k] = p
			}
		}()
	}
	for k := range isles {
		jobs <- k
	}
	close(jobs)
	wg.Wait()
	// Laid down in order, so the same seed always gives the same map.
	for k, p := range patches {
		pw := p.x1 - p.x0 + 1
		for ty := p.y0; ty <= p.y1; ty++ {
			for tx := p.x0; tx <= p.x1; tx++ {
				j, i := (ty-p.y0)*pw+tx-p.x0, ty*W+tx
				if v := p.val[j]; v > im.val[i] && v > -0.6 {
					im.val[i], im.h[i], im.owner[i], im.lagoon[i] = v, p.h[j], int16(k+1), p.inner[j]
				}
			}
		}
	}
	// Keep every island's land off the map's edge.
	for i := range im.val {
		x, y := i%W, i/W
		if e := min(x, y, W-1-x, H-1-y); e < isleEdge/2 && im.val[i] > 0 {
			im.val[i] = -0.01
		}
	}
	return im
}

// kindAt is the kind of the island tile i belongs to or lies off.
func (im *isleMap) kindAt(i int) IslandKind {
	if o := im.owner[i]; o > 0 {
		return im.isles[o-1].kind
	}
	return IslandMeadow
}

// isleAt is the island tile i belongs to or lies off, nil for open sea.
func (im *isleMap) isleAt(i int) *isle {
	if o := im.owner[i]; o > 0 {
		return &im.isles[o-1]
	}
	return nil
}

// islesTerrain fills w with the archipelago of StyleIsles and returns the
// spawn tile.
func (w *World) islesTerrain(seed uint32) (sx, sy int) {
	W, H := w.width, w.height
	w.relief = make([]Relief, W*H)
	isles := planIsles(W, H, seed)
	im := rasterIsles(W, H, isles)
	for i, v := range im.val {
		if v > 0 {
			w.tiles[i] = TileGrass
		} else {
			w.tiles[i] = TileWater
		}
	}
	w.cleanCoast(im)       // specks of land and pockets of sea; see coast.go
	w.splitTwins(im, seed) // a channel through one island; see coast.go
	w.laySpits(im, seed)   // sand spits closing off lagoons; see coast.go
	w.evenHeights(im)      // see heights.go
	toSea := w.seaDistance(30)
	levels := w.isleLevels(im, toSea, seed)
	for i, l := range levels {
		w.relief[i] = w.relief[i].withLevel(int(l))
	}
	w.paintIsles(im, toSea, seed)
	w.raiseCliffs(im, seed)      // see heights.go
	w.runRivers(im, toSea, seed) // see rivers.go
	w.fillShallows(im, seed)     // see coast.go
	w.standStacks(im, seed)      // see coast.go
	sx, sy = w.islesSpawn(im)    // see below
	w.isleVolcano(im, seed)      // see openworld.go
	w.joinLand(seed)             // ramps and fords: see heights.go
	w.landings()                 // see heights.go
	w.roomForHamlets()           // see coast.go
	w.mend()                     // see heights.go
	w.wetAndSoft(seed)           // see heights.go
	w.listIsles(im)
	return sx, sy
}

// Island painting tuning.
const (
	isleBeach      = 2.2  // tiles of beach, give or take
	isleForestLine = 0.52 // on the home island's noise: above it, woods
)

// paintIsles gives every land tile its ground by the island's kind, its
// height and its level, and the variants within each biome.
func (w *World) paintIsles(im *isleMap, toSea []uint8, seed uint32) {
	W := w.width
	for i, t := range w.tiles {
		if t != TileGrass {
			continue // the sea, and lakes
		}
		s := im.isleAt(i)
		if s == nil {
			continue
		}
		x, y := float64(i%W)+0.5, float64(i/W)+0.5
		t, v := s.paint(float64(im.h[i]), w.relief[i].Level(), int(toSea[i]), x, y, seed)
		w.tiles[i] = t
		w.relief[i] = w.relief[i].withVariant(v)
	}
}

// paint is the ground at (x, y) on island s: h its height (0 to 1), level
// its height level and sea how many tiles it lies from the sea.
func (s *isle) paint(h float64, level, sea int, x, y float64, seed uint32) (Tile, GroundVariant) {
	// A little dither at every border, so neighbouring biomes reach into
	// each other in fingers and islands instead of meeting in a line.
	jit := 0.05 * (unit(hash2(int32(x), int32(y), seed+0x5e1)) - 0.5)
	woods := fbm(x/22+57, y/22+213, seed+311, 4) + jit
	wet := fbm(x/30+71, y/30+29, seed+42, 3) + jit
	fine := fbm(x/7+19, y/7+3, seed+313, 2)
	beach := float64(sea) <= isleBeach+2.2*(fbm(x/16, y/16, seed+314, 2)-0.5)*2 && level == 0
	switch {
	case s.role == roleAtoll || s.role == roleIslet:
		if beach || sea <= 1 || s.role == roleIslet && h < 0.35 {
			if s.kind == IslandSnow {
				return TileSnow, VarPlain
			}
			return TileSand, VarPlain
		}
		if s.kind == IslandDesert {
			return TileDesert, VarPlain
		}
		return TileGrass, VarPlain
	case beach:
		switch s.kind {
		case IslandSnow:
			return TileSnow, VarPlain
		case IslandMarsh:
			if wet > 0.5 {
				return TileMarsh, VarMudflat
			}
		case IslandAsh:
			if fine > 0.5 {
				return TileAsh, VarPlain // black sand
			}
		}
		if fine > 0.72 {
			return TileSand, VarDunes
		}
		return TileSand, VarPlain
	}
	rock := fbm(x/9+5, y/9+9, seed+315, 2) + jit
	crag := fbm(x/4.5+7, y/4.5+1, seed+316, 2) + jit // small outcrops
	switch s.kind {
	case IslandMeadow:
		switch {
		case h > 0.75 && crag > 0.64:
			return TileRocky, VarOutcrop
		case woods > isleForestLine+0.14:
			return TileForest, VarDenseWood
		case woods > isleForestLine:
			return TileForest, VarPlain
		case woods > isleForestLine-0.05:
			return TileForest, VarOpenWood
		case fine > 0.62:
			return TileGrass, VarMeadow
		case rock > 0.74:
			return TileGrass, VarBoulders
		}
		return TileGrass, VarPlain
	case IslandForest, IslandMonkey:
		peak := s.kind == IslandMonkey && math.Hypot(x-s.hill[0], y-s.hill[1]) < s.hill[2]
		switch {
		case peak && level >= 2:
			if rock > 0.55 {
				return TileRocky, VarBoulders
			}
			return TileRocky, VarPlain
		case level >= 2 && crag > 0.66:
			return TileRocky, VarOutcrop
		case wet > 0.66 && h < 0.4:
			return TileMarsh, VarPlain // a boggy hollow
		case fine > 0.7 && woods < 0.55:
			return TileGrass, VarClearing
		case woods > 0.56:
			return TileForest, VarDenseWood
		case woods > 0.34:
			return TileForest, VarPlain
		case woods > 0.29:
			return TileForest, VarOpenWood
		}
		return TileGrass, VarMeadow
	case IslandMarsh:
		switch {
		case level >= 1 && woods > 0.48:
			return TileForest, VarOpenWood
		case level >= 1:
			return TileGrass, VarMeadow
		case fbm(x/7+13, y/7+57, seed+51, 2) > 0.66 && sea > 4:
			return TileWater, VarLake // a pond
		case woods > 0.62:
			return TileForest, VarPlain // swamp woods
		case wet < 0.4 && fine > 0.55:
			return TileGrass, VarPlain
		case sea <= 6 && fine > 0.45:
			return TileMarsh, VarMudflat
		}
		return TileMarsh, VarPlain
	case IslandSnow:
		switch {
		case level >= 3 && crag > 0.45 || level == 2 && crag > 0.64:
			if fine > 0.6 {
				return TileRocky, VarScree
			}
			return TileRocky, VarPlain
		case rock > 0.7:
			return TileSnow, VarBoulders
		case woods > 0.58 && level <= 1:
			return TileSnow, VarDenseWood // snowy pines
		}
		return TileSnow, VarPlain
	case IslandDesert:
		switch {
		case level >= 1 && crag > 0.66:
			return TileRocky, VarOutcrop // the mesas' caprock
		case level == 0 && fine > 0.48 && rock < 0.55:
			return TileDesert, VarDunes
		case rock > 0.72:
			return TileDesert, VarBoulders
		case wet > 0.68 && level == 0:
			return TileSand, VarPlain
		}
		return TileDesert, VarPlain
	case IslandAsh:
		switch {
		case level >= 2 && crag > 0.6:
			return TileRocky, VarScree
		case rock > 0.68:
			return TileAsh, VarBoulders
		}
		return TileAsh, VarPlain
	}
	return TileGrass, VarPlain
}

// islesSpawn picks the spawn: on the home island, near its middle, on low
// open ground with nothing steep or wet near.
func (w *World) islesSpawn(im *isleMap) (int, int) {
	home := &im.isles[0]
	W := w.width
	hx, hy := int(home.x), int(home.y)
	calm := func(tx, ty int) bool {
		for y := ty - 9; y <= ty+9; y++ {
			for x := tx - 9; x <= tx+9; x++ {
				t := w.TileAt(x, y)
				if t == TileCliff || t == TileRiver || t == TileWaterfall || t.Sea() && max(abs(x-tx), abs(y-ty)) <= 6 {
					return false
				}
				if max(abs(x-tx), abs(y-ty)) <= 5 && (!t.Walkable() || w.LevelAt(x, y) != w.LevelAt(tx, ty)) {
					return false
				}
			}
		}
		return true
	}
	R := int(home.r * 0.9)
	for r := 0; r <= R; r++ {
		for ty := hy - r; ty <= hy+r; ty++ {
			for tx := hx - r; tx <= hx+r; tx++ {
				if max(abs(tx-hx), abs(ty-hy)) != r || !w.inBounds(tx, ty) {
					continue
				}
				if t := w.tiles[ty*W+tx]; (t == TileGrass || t == TileForest) && im.owner[ty*W+tx] == 1 && calm(tx, ty) {
					return tx, ty
				}
			}
		}
	}
	return w.openSpawn()
}

// listIsles notes the islands of the world, by their kinds.
func (w *World) listIsles(im *isleMap) {
	w.islands = w.islands[:0]
	for _, isl := range w.landMasses() {
		if len(isl) < seasMinIsland {
			continue
		}
		var votes [numIsleKinds]int
		home := false
		for _, i := range isl {
			if o := im.owner[i]; o > 0 {
				votes[im.isles[o-1].kind]++
				home = home || o == 1
			}
		}
		best := IslandMeadow
		for k := range votes {
			if votes[k] > votes[best] {
				best = IslandKind(k)
			}
		}
		w.islands = append(w.islands, Island{Kind: best, Tiles: len(isl), Home: home})
	}
}

// isleVolcano raises the volcano at the top of the ash island.
func (w *World) isleVolcano(im *isleMap, seed uint32) {
	for k := range im.isles {
		s := &im.isles[k]
		if s.kind != IslandAsh || s.role != roleLarge {
			continue
		}
		top, best := -1, float32(-1)
		for i, o := range im.owner {
			if o == int16(k+1) && !w.tiles[i].Sea() && im.h[i] > best {
				top, best = i, im.h[i]
			}
		}
		if top >= 0 {
			w.cone(seed, top%w.width, top/w.width, clamp(s.r*0.45, 14, 64))
		}
		return
	}
}
