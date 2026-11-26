package sim

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
)

// Laying out and digging the natural caves of each island: see cavesys.go.
//
// An island's system is planned on a grid over the island and a margin: the
// halls under its mouths first, then chambers spread evenly over the land
// (each picked as the best of a few tries, as far from the rest as it can
// be), on each level. The chambers of a level are joined into a tree of the
// shortest tunnels, with a few extra short tunnels making loops, and a way
// round from the end of any side branch that runs too far. The tunnels wind
// from chamber to chamber; each is dug as a run of three-by-three squares
// (so no part of it is narrower than three tiles) under a wider, wavering
// round brush. The chambers are blobs warped by noise, with pillars of rock
// standing in the big ones away from the tunnels. Then the walls are
// smoothed, every floor tile not in some open three-by-three square is
// filled in, and whatever cannot be reached from the mouths is too.

// Cave system tuning.
const (
	caveHallR       = 5.5   // tiles of radius: the hall under a mouth
	caveChamberMin  = 7.0   // the smallest chamber: 15 tiles across
	caveChamberMax  = 20.0  // the biggest: 40 across
	caveUpperPer    = 4200  // tiles of island to each chamber of the upper level
	caveLowerPer    = 6500  // and of the lower
	caveLowerFrom   = 6000  // an island this big has a lower level
	caveMaxChambers = 26    // on a level, besides the halls
	caveGap         = 7.0   // tiles of rock at least between two chambers
	caveTunnelMin   = 3.4   // tiles: how wide a tunnel is dug, least
	caveTunnelMax   = 6.0   // and most
	caveRouteMin    = 4.2   // a tunnel of the main route, least
	caveDeadEnd     = 34.0  // tiles of tunnel: a side branch longer than this gets a way round
	caveLoops       = 0.35  // chance that a short tunnel closing a loop is dug
	caveLoopMax     = 80.0  // tiles: no loop is closed by a longer tunnel
	caveMouthsMax   = 4     // mouths to an island, most
	caveMouthPer    = 20000 // tiles of island to each mouth past the first
	caveSecretReach = 150.0 // tiles: lower levels this close may be joined by a secret passage
	caveSecretPer   = 3     // systems with a lower level to each secret passage
	caveGridPad     = 12    // tiles round an island its grid reaches
	cavePillarFrom  = 9.0   // a chamber this big has pillars of rock
)

// caveChamber is one chamber of a system.
type caveChamber struct {
	x, y    float64 // its middle, in tiles of the surface (the same column and row on both levels)
	r       float64 // its radius, in tiles
	aspect  float64 // how much wider than tall, along rot
	rot     float64 // radians
	level   int
	hall    bool // the hall under a mouth
	zone    CaveZone
	rock    CaveRock
	route   bool         // on the main route
	adj     []int        // its tunnels
	pillars [][3]float64 // the rock pillars standing in it: x, y, radius
	leaf    bool         // the end of a side branch: something worth the detour lies here
}

// caveTunnel is a tunnel joining chambers a and b of one level.
type caveTunnel struct {
	a, b  int
	level int
	path  [][2]float64 // tiles of the surface
	width float64      // the least it is dug, in tiles
	route bool
}

// caveSystem is the cave system under one island.
type caveSystem struct {
	island         int // index in w.islands
	ww             int // the width of the world, by which tiles are indexed
	area           int // tiles of island
	x0, y0, gw, gh int // its grid: surface tiles x0 to x0+gw-1, y0 to y0+gh-1
	mouths         [][2]int
	exits          [][2]int // the ways up under them, map tiles
	chambers       []caveChamber
	tunnels        []caveTunnel
	levels         int
	descent        int    // the chamber with the hole down, -1 with one level
	arrival        int    // the chamber the rope comes down into, -1 with one level
	down, up       [2]int // the hole and the rope, map tiles
	dest           int    // the destination chamber, at the end of the main route
	owner          [CaveLevels][]int16
	open           [CaveLevels][]bool // per grid cell and level: dug out
	floor          [CaveLevels]int    // floor tiles of each level
}

// caveSecret is a long tunnel under the sea joining the lower levels of two
// systems.
type caveSecret struct {
	a, b int          // the systems
	ca   int          // the chamber of a it leaves from
	cb   int          // and of b it reaches
	path [][2]float64 // tiles of the surface
}

// gridAt is the grid cell of surface tile (tx, ty), ok false off the grid.
func (s *caveSystem) gridAt(tx, ty int) (int, bool) {
	x, y := tx-s.x0, ty-s.y0
	if x < 0 || y < 0 || x >= s.gw || y >= s.gh {
		return 0, false
	}
	return y*s.gw + x, true
}

// mapTile is the map tile of grid cell i on level k.
func (w *World) caveMapTile(s *caveSystem, k, i int) (int, int) {
	return s.x0 + i%s.gw, CaveBandTop(w.surfaceH, k) + s.y0 + i/s.gw
}

// digCaveSystems sets the mouths on every island but the smallest islets and
// digs the cave system under each, then the secret passages between them.
func (w *World) digCaveSystems(rng *rand.Rand) {
	index := w.islandIndex()
	tilesOf := make([][]int, len(w.islands))
	for i, k := range index {
		if k >= 0 {
			tilesOf[k] = append(tilesOf[k], i)
		}
	}
	var placed [][2]int
	for k, isl := range w.islands {
		tiles := tilesOf[k]
		if len(tiles) < mouthIslandMin {
			continue
		}
		want := min(caveMouthsMax, 1+len(tiles)/caveMouthPer)
		mouths := w.pickCaveMouths(rng, tiles, isl.Home, want, &placed)
		if len(mouths) == 0 {
			continue
		}
		srng := rand.New(rand.NewPCG(w.seed^0xca7e5, uint64(k)))
		w.caves.systems = append(w.caves.systems, w.buildCaveSystem(srng, k, tiles, mouths))
	}
	w.digSecretPassages(rand.New(rand.NewPCG(w.seed, 0x5ec7e7)))
}

// pickCaveMouths chooses up to want mouths on an island, spread over it, by
// the rules of the older worlds' mouths (digCaveMouths): on open ground,
// clear of villages and the pyramid, away from the spawn on the home island.
func (w *World) pickCaveMouths(rng *rand.Rand, tiles []int, home bool, want int, placed *[][2]int) [][2]int {
	W, H := w.width, w.surfaceH
	sx, sy := int(w.spawnX), int(w.spawnY)
	spread := max(mouthSpacing, int(math.Sqrt(float64(len(tiles)))*0.45))
	good := func(tx, ty, margin int) bool {
		if w.inVillage(tx, ty, margin) {
			return false // not in a village's clearing; see islandtribes.go
		}
		for y := ty - 1; y <= ty+2; y++ {
			for x := tx - 2; x <= tx+2; x++ {
				if y >= H || !w.TileAt(x, y).Walkable() || w.TileAt(x, y) == TilePath || w.TileAt(x, y) == TileLava || w.crossing(y*W+x) {
					return false
				}
				if y <= ty+1 {
					if e := w.entityAt(x, y); e != nil && (blocksPlayers(e.Kind) || e.Kind == EntityTreasure) {
						return false // a hut, the pyramid and its chest
					}
				}
			}
		}
		for _, q := range *placed {
			if abs(q[0]-tx)+abs(q[1]-ty) < mouthSpacing {
				return false
			}
		}
		return true
	}
	spots := slices.Clone(tiles)
	rng.Shuffle(len(spots), func(a, b int) { spots[a], spots[b] = spots[b], spots[a] })
	var out [][2]int
	for k, margin := range []int{3, 0, -hamletRadius} {
		if len(out) >= want || k > 0 && len(out) > 0 {
			break
		}
		for _, i := range spots {
			if len(out) >= want {
				break
			}
			tx, ty := i%W, i/W
			if home && math.Hypot(float64(tx-sx), float64(ty-sy)) < homeMouthAway {
				continue
			}
			if !good(tx, ty, margin) {
				continue
			}
			apart := true
			for _, m := range out {
				if abs(m[0]-tx)+abs(m[1]-ty) < spread {
					apart = false
					break
				}
			}
			if !apart {
				continue
			}
			out = append(out, [2]int{tx, ty})
			*placed = append(*placed, [2]int{tx, ty})
		}
	}
	return out
}

// buildCaveSystem plans and digs the system under island k (its tiles, on
// the surface), entered by mouths.
func (w *World) buildCaveSystem(rng *rand.Rand, k int, tiles []int, mouths [][2]int) *caveSystem {
	W, H := w.width, w.surfaceH
	s := &caveSystem{island: k, ww: W, area: len(tiles), mouths: mouths, descent: -1, arrival: -1, levels: 1}
	minX, minY, maxX, maxY := W, H, 0, 0
	for _, i := range tiles {
		minX, minY = min(minX, i%W), min(minY, i/W)
		maxX, maxY = max(maxX, i%W), max(maxY, i/W)
	}
	s.x0, s.y0 = max(1, minX-caveGridPad), max(1, minY-caveGridPad)
	s.gw, s.gh = min(W-2, maxX+caveGridPad)-s.x0+1, min(H-2, maxY+caveGridPad)-s.y0+1
	// How far each tile lies inside the island.
	inside := make([]uint8, s.gw*s.gh)
	var q []int
	land := make([]bool, s.gw*s.gh)
	for _, i := range tiles {
		if c, ok := s.gridAt(i%W, i/W); ok {
			land[c] = true
		}
	}
	for c := range land {
		if !land[c] {
			q = append(q, c)
		} else {
			inside[c] = 255
		}
	}
	for n := 0; n < len(q); n++ {
		c := q[n]
		x, y := c%s.gw, c/s.gw
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= s.gw || ny >= s.gh {
				continue
			}
			if j := ny*s.gw + nx; inside[j] > inside[c]+1 {
				inside[j] = inside[c] + 1
				q = append(q, j)
			}
		}
	}

	// The halls under the mouths, then the chambers of the upper level.
	for _, m := range mouths {
		s.chambers = append(s.chambers, caveChamber{
			x: float64(m[0]) + 0.5, y: float64(m[1]) + 4.5, r: caveHallR + rng.Float64(),
			aspect: 1.3, level: 0, hall: true, zone: ZoneEntrance,
		})
	}
	rmax := clamp(math.Sqrt(float64(len(tiles)))/6.5, caveChamberMin-1.5, caveChamberMax)
	n0 := min(caveMaxChambers, len(tiles)/caveUpperPer)
	if n0 == 0 && len(tiles) >= 1200 {
		n0 = 1
	}
	s.placeChambers(rng, 0, n0, rmax, tiles, inside)
	s.join(rng, 0)

	// The way down, at the far end of the upper level, and the lower level
	// round where it comes out.
	if len(tiles) >= caveLowerFrom {
		far, best := -1, -1.0
		dist := s.distances(0)
		for i, c := range s.chambers {
			if c.level == 0 && !c.hall && dist[i] > best {
				far, best = i, dist[i]
			}
		}
		if far >= 0 {
			s.levels, s.descent = 2, far
			d := s.chambers[far]
			s.arrival = len(s.chambers)
			s.chambers = append(s.chambers, caveChamber{
				x: d.x, y: d.y, r: max(caveChamberMin, d.r*0.9), aspect: 0.85 + 0.4*rng.Float64(),
				rot: rng.Float64() * math.Pi, level: 1, zone: ZoneHall,
			})
			n1 := clamp(float64(len(tiles)/caveLowerPer), 1, caveMaxChambers)
			s.placeChambers(rng, 1, int(n1), rmax, tiles, inside)
			s.join(rng, 1)
		}
	}

	// The destination: the far end of the deepest level.
	s.dest = 0
	if s.levels == 2 {
		dist := s.distances(s.arrival)
		best := -1.0
		for i, c := range s.chambers {
			if c.level == 1 && dist[i] > best {
				s.dest, best = i, dist[i]
			}
		}
		s.markRoute(0, s.descent)
		s.markRoute(s.arrival, s.dest)
	} else {
		dist := s.distances(0)
		best := -1.0
		for i, c := range s.chambers {
			if !c.hall && dist[i] > best {
				s.dest, best = i, dist[i]
			}
		}
		s.markRoute(0, s.dest)
	}
	for i := range s.chambers {
		c := &s.chambers[i]
		c.leaf = len(c.adj) == 1 && !c.hall && !c.route
	}
	s.assignZones(rng) // see cavedress.go

	for i := range s.tunnels {
		s.windTunnel(rng, i)
	}
	for i := range s.chambers {
		s.placePillars(rng, i)
	}
	for i := range s.chambers {
		s.chambers[i].rock = w.rockAbove(&s.chambers[i], w.islands[k].Kind)
	}
	w.digSystem(s)
	return s
}

// placeChambers spreads n more chambers over the island on level lv, each
// the best of a few tries: inside the island, as far from the others as
// it can be.
func (s *caveSystem) placeChambers(rng *rand.Rand, lv, n int, rmax float64, tiles []int, inside []uint8) {
	for range n {
		r := caveChamberMin + (rmax-caveChamberMin)*math.Pow(rng.Float64(), 1.25)
		if rmax < caveChamberMin {
			r = rmax
		}
		bx, by, best := 0.0, 0.0, -1.0
		for range 24 {
			i := tiles[rng.IntN(len(tiles))]
			c, ok := s.gridAt(i%s.worldW(), i/s.worldW())
			if !ok || float64(inside[c]) < max(3, 0.5*r) {
				continue
			}
			x, y := float64(i%s.worldW())+0.5, float64(i/s.worldW())+0.5
			gap := math.Inf(1)
			for _, o := range s.chambers {
				if o.level == lv {
					gap = min(gap, math.Hypot(o.x-x, o.y-y)-o.r-r)
				}
			}
			if gap < caveGap {
				continue
			}
			if gap > best {
				bx, by, best = x, y, gap
			}
		}
		if best < 0 {
			continue // no room left for one this big
		}
		s.chambers = append(s.chambers, caveChamber{
			x: bx, y: by, r: r, aspect: 0.75 + 0.55*rng.Float64(), rot: rng.Float64() * math.Pi,
			level: lv, zone: ZoneHall,
		})
	}
}

// worldW is the width of the world, by which the island's tiles are indexed.
func (s *caveSystem) worldW() int { return s.ww }

// tunnelLen is how long a tunnel between chambers a and b is: the rock
// between their edges.
func tunnelLen(a, b caveChamber) float64 { return math.Hypot(a.x-b.x, a.y-b.y) - a.r - b.r }

// join digs the tunnels of level lv: the shortest that join every chamber,
// a few short ones more that close loops, and a way round from every side
// branch that runs on too far.
func (s *caveSystem) join(rng *rand.Rand, lv int) {
	var ids []int
	for i, c := range s.chambers {
		if c.level == lv {
			ids = append(ids, i)
		}
	}
	type pair struct {
		a, b int
		d    float64
	}
	var pairs []pair
	for x, a := range ids {
		for _, b := range ids[x+1:] {
			pairs = append(pairs, pair{a, b, tunnelLen(s.chambers[a], s.chambers[b])})
		}
	}
	slices.SortStableFunc(pairs, func(p, q pair) int { return cmp.Compare(p.d, q.d) })
	parent := make(map[int]int, len(ids))
	var find func(int) int
	find = func(i int) int {
		if p, ok := parent[i]; ok && p != i {
			r := find(p)
			parent[i] = r
			return r
		}
		return i
	}
	longest := make(map[int]float64, len(ids))
	adjacent := func(a, b int) bool {
		for _, t := range s.chambers[a].adj {
			if tt := s.tunnels[t]; tt.a == b || tt.b == b {
				return true
			}
		}
		return false
	}
	add := func(a, b int) {
		s.tunnels = append(s.tunnels, caveTunnel{a: a, b: b, level: lv, width: caveTunnelMin})
		t := len(s.tunnels) - 1
		s.chambers[a].adj = append(s.chambers[a].adj, t)
		s.chambers[b].adj = append(s.chambers[b].adj, t)
	}
	crosses := func(a, b int) bool { return s.crossesAny(lv, ids, a, b, true) }
	for _, p := range pairs {
		if ra, rb := find(p.a), find(p.b); ra != rb {
			parent[ra] = rb
			add(p.a, p.b)
			longest[p.a] = max(longest[p.a], p.d)
			longest[p.b] = max(longest[p.b], p.d)
		}
	}
	for _, p := range pairs {
		if adjacent(p.a, p.b) || p.d > caveLoopMax || p.d > 1.6*max(longest[p.a], longest[p.b]) {
			continue
		}
		if rng.Float64() < caveLoops && !crosses(p.a, p.b) {
			add(p.a, p.b)
		}
	}
	// A side branch that runs on too far gets a way round from its end:
	// the shortest tunnel that crosses none, reaching further each round.
	for pass := range 6 {
		reach := caveLoopMax * (1 + 0.4*float64(pass))
		for _, a := range ids {
			if len(s.chambers[a].adj) != 1 || s.chambers[a].hall || s.branchLen(a) <= caveDeadEnd {
				continue
			}
			for _, p := range pairs {
				if p.a != a && p.b != a {
					continue
				}
				b := p.a + p.b - a
				if p.d <= reach && !adjacent(a, b) && !s.crossesAny(lv, ids, a, b, pass < 3) {
					add(a, b)
					break
				}
			}
		}
	}
}

// crossesAny reports whether a tunnel of level lv from chamber a to b would
// cross another, or with through, run through a chamber of ids on its way.
func (s *caveSystem) crossesAny(lv int, ids []int, a, b int, through bool) bool {
	ca, cb := s.chambers[a], s.chambers[b]
	for _, t := range s.tunnels {
		if t.level != lv || t.a == a || t.a == b || t.b == a || t.b == b {
			continue
		}
		p, q := s.chambers[t.a], s.chambers[t.b]
		if segmentsCross(ca.x, ca.y, cb.x, cb.y, p.x, p.y, q.x, q.y) {
			return true
		}
	}
	if !through {
		return false
	}
	for _, i := range ids {
		if i == a || i == b {
			continue
		}
		c := s.chambers[i]
		if _, d := segDist(c.x, c.y, ca.x, ca.y, cb.x, cb.y); d < c.r+2 {
			return true
		}
	}
	return false
}

// branchLen is how far from chamber a, the end of a side branch, the
// branch runs before it meets a chamber where three or more tunnels meet:
// the tunnels' lengths added up.
func (s *caveSystem) branchLen(a int) float64 {
	total, prev, cur := 0.0, -1, a
	for steps := 0; steps < len(s.chambers); steps++ {
		c := s.chambers[cur]
		if cur != a && len(c.adj) != 2 {
			return total
		}
		next := -1
		for _, t := range c.adj {
			tt := s.tunnels[t]
			o := tt.a + tt.b - cur
			if o != prev {
				next = o
				total += tunnelLen(s.chambers[cur], s.chambers[o])
				break
			}
		}
		if next < 0 {
			return total
		}
		prev, cur = cur, next
	}
	return total
}

// distances is how far each chamber lies from chamber from, through the
// tunnels (their lengths and the chambers crossed); +Inf for none of its
// level.
func (s *caveSystem) distances(from int) []float64 {
	d := make([]float64, len(s.chambers))
	for i := range d {
		d[i] = math.Inf(1)
	}
	d[from] = 0
	done := make([]bool, len(s.chambers))
	for {
		cur, best := -1, math.Inf(1)
		for i, v := range d {
			if !done[i] && v < best {
				cur, best = i, v
			}
		}
		if cur < 0 {
			return d
		}
		done[cur] = true
		for _, t := range s.chambers[cur].adj {
			tt := s.tunnels[t]
			o := tt.a + tt.b - cur
			if v := best + math.Hypot(s.chambers[o].x-s.chambers[cur].x, s.chambers[o].y-s.chambers[cur].y); v < d[o] {
				d[o] = v
			}
		}
	}
}

// markRoute marks the shortest way from chamber a to chamber b as the main
// route: its chambers and its tunnels.
func (s *caveSystem) markRoute(a, b int) {
	d := s.distances(a)
	if math.IsInf(d[b], 1) {
		return
	}
	cur := b
	s.chambers[cur].route = true
	for cur != a {
		for _, t := range s.chambers[cur].adj {
			tt := &s.tunnels[t]
			o := tt.a + tt.b - cur
			step := math.Hypot(s.chambers[o].x-s.chambers[cur].x, s.chambers[o].y-s.chambers[cur].y)
			if math.Abs(d[o]+step-d[cur]) < 1e-6 {
				tt.route, tt.width = true, caveRouteMin
				cur = o
				s.chambers[cur].route = true
				break
			}
		}
	}
}

// windTunnel lays tunnel i's path from the middle of one chamber to the
// other: a straight line bent again and again, each half's middle pushed
// aside a little.
func (s *caveSystem) windTunnel(rng *rand.Rand, i int) {
	t := &s.tunnels[i]
	a, b := s.chambers[t.a], s.chambers[t.b]
	// From a hall, a tunnel heading north first goes round the side of the
	// hall, so it never passes the way up in its north wall.
	round := func(h, o caveChamber) ([2]float64, bool) {
		if !h.hall || o.y > h.y-1 {
			return [2]float64{}, false
		}
		side := 1.0
		if o.x < h.x || o.x == h.x && rng.IntN(2) == 0 {
			side = -1
		}
		return [2]float64{h.x + side*(h.r*h.aspect+1.5), h.y + 0.5}, true
	}
	pts := [][2]float64{{a.x, a.y}}
	if wp, ok := round(a, b); ok {
		pts = append(pts, wp)
	}
	if wp, ok := round(b, a); ok {
		pts = append(pts, wp)
	}
	pts = append(pts, [2]float64{b.x, b.y})
	for range 5 {
		var out [][2]float64
		for j := 0; j+1 < len(pts); j++ {
			p, q := pts[j], pts[j+1]
			out = append(out, p)
			dx, dy := q[0]-p[0], q[1]-p[1]
			l := math.Hypot(dx, dy)
			if l < 6 {
				continue
			}
			off := (rng.Float64()*2 - 1) * l * 0.2
			out = append(out, [2]float64{(p[0]+q[0])/2 - dy/l*off, (p[1]+q[1])/2 + dx/l*off})
		}
		pts = append(out, pts[len(pts)-1])
	}
	t.path = pts
}

// placePillars stands pillars of rock in chamber i if it is big, away from
// its middle and every tunnel through it.
func (s *caveSystem) placePillars(rng *rand.Rand, i int) {
	c := &s.chambers[i]
	if c.hall || c.r < cavePillarFrom {
		return
	}
	n := int(c.r/6) + rng.IntN(2)
	for tries := 0; len(c.pillars) < n && tries < 30; tries++ {
		a, d := rng.Float64()*2*math.Pi, c.r*(0.3+0.4*rng.Float64())
		px, py := c.x+math.Cos(a)*d*c.aspect, c.y+math.Sin(a)*d/c.aspect
		pr := 1.1 + 1.3*rng.Float64()
		if s.nearTunnel(c.level, px, py, pr+2.6) {
			continue
		}
		ok := true
		for _, p := range c.pillars {
			if math.Hypot(p[0]-px, p[1]-py) < p[2]+pr+3.5 {
				ok = false
				break
			}
		}
		if ok {
			c.pillars = append(c.pillars, [3]float64{px, py, pr})
		}
	}
}

// nearTunnel reports whether (x, y) lies within r of the path of a tunnel
// of level lv.
func (s *caveSystem) nearTunnel(lv int, x, y, r float64) bool {
	for _, t := range s.tunnels {
		if t.level != lv {
			continue
		}
		for j := 0; j+1 < len(t.path); j++ {
			p, q := t.path[j], t.path[j+1]
			if _, d := segDist(x, y, p[0], p[1], q[0], q[1]); d < r {
				return true
			}
		}
	}
	return false
}

// segmentsCross reports whether segments (a, b) and (c, d) cross.
func segmentsCross(ax, ay, bx, by, cx, cy, dx, dy float64) bool {
	o := func(px, py, qx, qy, rx, ry float64) float64 { return (qx-px)*(ry-py) - (qy-py)*(rx-px) }
	d1, d2 := o(cx, cy, dx, dy, ax, ay), o(cx, cy, dx, dy, bx, by)
	d3, d4 := o(ax, ay, bx, by, cx, cy), o(ax, ay, bx, by, dx, dy)
	return (d1 > 0) != (d2 > 0) && (d3 > 0) != (d4 > 0)
}

// rockAbove is the rock of chamber c, by the land above it: the ground most
// of it lies under, or for open ground the island's own.
func (w *World) rockAbove(c *caveChamber, kind IslandKind) CaveRock {
	var votes [NumCaveRocks]int
	R := int(c.r)
	for dy := -R; dy <= R; dy += 2 {
		for dx := -R; dx <= R; dx += 2 {
			if dx*dx+dy*dy > R*R {
				continue
			}
			votes[rockOfTile(w.TileAt(int(c.x)+dx, int(c.y)+dy), kind)]++
		}
	}
	best := rockOfIsland(kind)
	for r := range votes {
		if votes[r] > votes[best] {
			best = CaveRock(r)
		}
	}
	return best
}

// rockOfTile is the rock under ground t of an island of kind k.
func rockOfTile(t Tile, k IslandKind) CaveRock {
	switch t {
	case TileForest:
		return RockMossy
	case TileMarsh:
		return RockMud
	case TileDesert:
		return RockSandstone
	case TileAsh, TileLava:
		return RockBasalt
	case TileSnow:
		return RockIce
	}
	return rockOfIsland(k)
}

// rockOfIsland is the rock under the open ground (grass, sand, bare rock)
// of an island of kind k.
func rockOfIsland(k IslandKind) CaveRock {
	switch k {
	case IslandForest, IslandMonkey:
		return RockMossy
	case IslandMarsh:
		return RockMud
	case IslandDesert:
		return RockSandstone
	case IslandAsh:
		return RockBasalt
	case IslandSnow:
		return RockIce
	}
	return RockLimestone
}
