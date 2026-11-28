package sim

import (
	"cmp"
	"math"
	"math/rand/v2"
	"slices"
)

// The cave ways of worlds made since StyleCaveWays: the natural caves of
// cavesys.go laid out so that there is more than one way to go, without
// becoming a maze again.
//
// The systems of StyleCaves under the small and middling islands were one
// line: the hall, a chamber, the rope down, a chamber, the hoard; and even
// under the big ones the main route was all but the only sensible way, its
// side branches single dead ends. Now:
//
//   - A level has about twice as many chambers, of mixed sizes: a few great
//     halls, more middling chambers and small grottos (at least three
//     besides the hall on an island of some size, and three below).
//   - Its tunnels are still the shortest that join every chamber, but grown
//     from the way in, and a chamber may be joined to the middle of a
//     passage already there rather than to a chamber: a fork in the passage
//     (a junction, a small widening where the ways meet).
//   - The main route of every level, from the way in (the first hall, or
//     where the rope comes down) to the way on (the hole down, or the
//     hoard), lies on a loop: tunnels as wide as the route's are dug until
//     no single passage has to be taken, so there are always two ways
//     through, and where they part three passages meet.
//   - Side pockets: a short spur (8 to 20 tiles) off a tunnel or a chamber
//     to a small grotto with something in it: a seam of coal (a crystal
//     grotto with jade below), glowing mushrooms, a pool, a skeleton to dig
//     bones out of, now and then a small cache of supplies. Every third of
//     the main route has some way leading off it.
//   - Under the big islands a second hole goes down from another part of
//     the upper level, so the lower one is entered from two places.
//
// The old chests are fewer, and leaner (see caveWaysLoot): supplies in
// most of them, ore by how far the island lies from home, and the best
// things only in the hoard at the end of the deepest level.

// caveWays reports whether w's caves are laid out as those of
// StyleCaveWays. Every check for them goes through here, so the style is
// renumbered in one place.
func (w *World) caveWays() bool { return caveWaysStyle(w.style) }

func caveWaysStyle(s WorldStyle) bool { return s >= StyleCaveWays }

// Cave ways tuning.
const (
	caveWaysUpperPer = 2100  // tiles of island to each chamber of the upper level, of the first caveWaysDense
	caveWaysLowerPer = 3200  // and of the lower
	caveWaysDense    = 15000 // tiles of island: past this, the chambers are sparser
	caveWaysUpperFar = 3600  // tiles of island past caveWaysDense to each chamber of the upper level
	caveWaysLowerFar = 5500  // and of the lower
	caveWaysRadius   = 7.2   // the biggest chamber's radius is the square root of the island's tiles over this
	caveWaysFew      = 3     // chambers at least on a level of an island this big (besides the halls, or the arrival)
	caveWaysFewFrom  = 2500  // tiles of island
	caveWaysMaxUpper = 30    // chambers most on the upper level, besides the halls
	caveWaysMaxLower = 24    // and on the lower
	caveWaysGap      = 6.0   // tiles of rock at least between two chambers
	caveGrottoMin    = 5.0   // the smallest chamber: 11 tiles across
	caveWaysBig      = 0.2   // share of the chambers that are great halls
	caveWaysSmall    = 0.4   // and that are small grottos
	caveWaysLoops    = 0.25  // chance that a short tunnel closing a loop is dug
	caveJunctionR    = 3.0   // tiles of radius: the widening where a passage forks
	caveForkEnd      = 7.0   // tiles of tunnel at least between a fork and the chamber at either end
	caveForkBias     = 0.9   // a chamber is joined to a passage rather than a chamber when that is this much shorter
	caveBypassMax    = 110.0 // tiles: the longest tunnel dug to give the main route a second way
	caveBypassSlack  = 15.0  // tiles added to a bypass's length when weighing it against how much it bypasses
	caveForkShort    = 0.9   // a bypass leaves the main route at least this long: no short cut
	caveForkRounds   = 16    // bypasses dug at most on a level
	cavePocketMin    = 3.0   // a side pocket's radius, least
	cavePocketMax    = 5.0   // and most
	caveSpurMin      = 8.0   // tiles of spur to a pocket, least
	caveSpurMax      = 20.0  // and most
	cavePocketUpper  = 3500  // tiles of island to each pocket of the upper level
	cavePocketLower  = 5000  // and of the lower
	cavePocketsMost  = 12    // pockets on a level, besides those off the main route's thirds
	caveThirdTries   = 48    // tries at a pocket off a third of the main route with no way off it
	caveThirdsFrom   = 50.0  // tiles, middle to middle: a main route shorter than this is one stretch, not three
	caveTwoWaysDown  = 40000 // tiles: an island this big has a second way down
	caveShaftApart   = 70.0  // tiles at least between its two ways down
	caveDestFar      = 0.75  // the destination lies at least this share of the way to the furthest chamber
	caveWaysWall     = 3.5   // tiles of rock at least round a pocket and its spur
)

// What lies in a side pocket (caveChamber.pocket).
const (
	pocketNone     uint8 = iota
	pocketOre            // a seam of coal; below, a grotto of glowing crystals with jade in its walls
	pocketMushroom       // glowing mushrooms, and some to eat
	pocketPool           // a pool of fresh water
	pocketBones          // a skeleton in the rock, to dig bones out of
)

// caveShaft is a way down past the first, under the big islands.
type caveShaft struct {
	descent, arrival int    // the chamber with the hole, and the one the rope comes down into
	down, up         [2]int // the hole and the rope, map tiles
}

// shaft reports whether chamber i is where a way down past the first
// leaves or arrives.
func (s *caveSystem) shaft(i int) bool {
	for _, sh := range s.shafts {
		if sh.descent == i || sh.arrival == i {
			return true
		}
	}
	return false
}

// planCaveWays lays out the chambers and tunnels of s, its ways down and
// its main route, by the rules of StyleCaveWays; the halls under the
// mouths are there already.
func (w *World) planCaveWays(rng *rand.Rand, s *caveSystem, tiles []int, inside []uint8) {
	area := len(tiles)
	rmax := clamp(math.Sqrt(float64(area))/caveWaysRadius, caveChamberMin-1.5, caveChamberMax)
	count := func(per, far int) int { // chambers to the island's tiles, sparser past caveWaysDense
		return min(area, caveWaysDense)/per + max(0, area-caveWaysDense)/far
	}
	n0 := min(caveWaysMaxUpper, count(caveWaysUpperPer, caveWaysUpperFar))
	switch {
	case area >= caveWaysFewFrom:
		n0 = max(n0, caveWaysFew)
	case area >= 1200:
		n0 = max(n0, 1)
	}
	s.placeWays(rng, 0, n0, rmax, tiles, inside)
	s.joinWays(rng, 0, 0)

	// The way down, at the far end of the upper level, a second one under
	// a big island, and the lower level round where they come out.
	if area >= caveLowerFrom {
		dist := s.distances(0)
		far := s.farthest(0, dist, func(i int) bool { return true })
		if far >= 0 {
			s.levels, s.descent = 2, far
			second := -1
			if area >= caveTwoWaysDown {
				d := s.chambers[far]
				second = s.farthest(0, dist, func(i int) bool {
					c := s.chambers[i]
					return i != far && dist[i] >= 0.45*dist[far] && math.Hypot(c.x-d.x, c.y-d.y) >= caveShaftApart
				})
			}
			s.arrival = s.addArrival(rng, far)
			if second >= 0 {
				s.shafts = append(s.shafts, caveShaft{descent: second, arrival: s.addArrival(rng, second)})
			}
			n1 := clamp1(count(caveWaysLowerPer, caveWaysLowerFar), caveWaysFew, caveWaysMaxLower)
			s.placeWays(rng, 1, n1, rmax, tiles, inside)
			s.joinWays(rng, 1, s.arrival)
		}
	}

	// The destination: the far end of the deepest level, as far as can be
	// from every way into it, and of the far chambers the biggest.
	s.dest = 0
	if s.levels == 2 {
		ways := [][]float64{s.distances(s.arrival)}
		for _, sh := range s.shafts {
			ways = append(ways, s.distances(sh.arrival))
		}
		far := make([]float64, len(s.chambers))
		for i := range far {
			far[i] = math.Inf(1)
			for _, dist := range ways {
				far[i] = min(far[i], dist[i])
			}
		}
		s.dest = s.bigFar(1, far, func(i int) bool { return i != s.arrival && !s.shaft(i) })
		if s.dest < 0 {
			s.dest = s.arrival
		}
		s.ensureFork(0, 0, s.descent)
		s.ensureFork(1, s.arrival, s.dest)
		s.markRoute(0, s.descent)
		s.markRoute(s.arrival, s.dest)
	} else {
		s.dest = max(0, s.bigFar(0, s.distances(0), func(int) bool { return true }))
		s.ensureFork(0, 0, s.dest)
		s.markRoute(0, s.dest)
	}
	for i := range s.chambers {
		c := &s.chambers[i]
		c.leaf = len(c.adj) == 1 && !c.hall && !c.route && !c.junction
	}
}

// farthest is the chamber of level lv, not a hall nor a fork, that ok
// holds for and lies furthest by dist; a big enough one for a hole if there
// is one. -1 for none.
func (s *caveSystem) farthest(lv int, dist []float64, ok func(i int) bool) int {
	far, best, roomy := -1, -1.0, false
	for i, c := range s.chambers {
		if c.level != lv || c.hall || c.junction || math.IsInf(dist[i], 1) || !ok(i) {
			continue
		}
		big := c.r >= caveChamberMin-0.5
		if big && !roomy || big == roomy && dist[i] > best {
			far, best, roomy = i, dist[i], big
		}
	}
	return far
}

// bigFar is the biggest chamber of level lv, not a hall nor a fork, that ok
// holds for, of those lying nearly as far by dist as the furthest: the
// destination, a hall for a hoard and the beast that guards it. -1 for none.
func (s *caveSystem) bigFar(lv int, dist []float64, ok func(i int) bool) int {
	most := 0.0
	for i, c := range s.chambers {
		if c.level == lv && !c.hall && !c.junction && !math.IsInf(dist[i], 1) && ok(i) {
			most = max(most, dist[i])
		}
	}
	best, bestR := -1, -1.0
	for i, c := range s.chambers {
		if c.level != lv || c.hall || c.junction || math.IsInf(dist[i], 1) || !ok(i) || dist[i] < caveDestFar*most {
			continue
		}
		if c.r > bestR || c.r == bestR && dist[i] > dist[best] {
			best, bestR = i, c.r
		}
	}
	return best
}

// addArrival adds the chamber of the lower level a rope comes down into,
// under chamber d of the upper, and returns it.
func (s *caveSystem) addArrival(rng *rand.Rand, d int) int {
	c := s.chambers[d]
	s.chambers = append(s.chambers, caveChamber{
		x: c.x, y: c.y, r: max(caveChamberMin, c.r*0.9), aspect: 0.85 + 0.4*rng.Float64(),
		rot: rng.Float64() * math.Pi, level: 1, zone: ZoneHall,
	})
	return len(s.chambers) - 1
}

// placeWays spreads n more chambers over the island on level lv: a few
// great halls, middling chambers and small grottos, the biggest placed
// first, each the best of a few tries as far from the others as it can be.
func (s *caveSystem) placeWays(rng *rand.Rand, lv, n int, rmax float64, tiles []int, inside []uint8) {
	rs := make([]float64, n)
	for i := range rs {
		switch u := rng.Float64(); {
		case rmax < caveChamberMin:
			rs[i] = rmax
		case u < caveWaysBig:
			rs[i] = max(caveChamberMin, rmax*(0.78+0.22*rng.Float64()))
		case u < caveWaysBig+caveWaysSmall:
			rs[i] = caveGrottoMin + (caveChamberMin-caveGrottoMin)*rng.Float64()
		default:
			rs[i] = caveChamberMin + (max(caveChamberMin, 0.72*rmax)-caveChamberMin)*rng.Float64()
		}
	}
	if n >= 4 && rmax >= caveChamberMin {
		rs[0] = max(rs[0], rmax*(0.8+0.2*rng.Float64())) // one great hall at least
	}
	slices.SortStableFunc(rs, func(a, b float64) int { return cmp.Compare(b, a) })
	W := s.worldW()
	for _, r := range rs {
		for shrink := range 2 {
			if shrink > 0 {
				r = max(caveGrottoMin, r*0.75)
			}
			bx, by, best := 0.0, 0.0, -1.0
			for range 32 {
				i := tiles[rng.IntN(len(tiles))]
				c, ok := s.gridAt(i%W, i/W)
				if !ok || float64(inside[c]) < max(3, 0.5*r) {
					continue
				}
				x, y := float64(i%W)+0.5, float64(i/W)+0.5
				gap := math.Inf(1)
				for _, o := range s.chambers {
					if o.level == lv {
						gap = min(gap, math.Hypot(o.x-x, o.y-y)-o.r-r)
					}
				}
				if gap < caveWaysGap {
					continue
				}
				if gap > best {
					bx, by, best = x, y, gap
				}
			}
			if best >= 0 {
				s.chambers = append(s.chambers, caveChamber{
					x: bx, y: by, r: r, aspect: 0.75 + 0.55*rng.Float64(), rot: rng.Float64() * math.Pi,
					level: lv, zone: ZoneHall,
				})
				break
			}
		}
	}
}

// reach is how far chamber c reaches from its middle at most, before the
// noise on its edge.
func (c caveChamber) reach() float64 { return c.r * max(c.aspect, 1/c.aspect) }

// addTunnel adds a tunnel of level lv from chamber a to b, width wide at
// least, and returns it.
func (s *caveSystem) addTunnel(lv, a, b int, width float64) int {
	s.tunnels = append(s.tunnels, caveTunnel{a: a, b: b, level: lv, width: width})
	t := len(s.tunnels) - 1
	s.chambers[a].adj = append(s.chambers[a].adj, t)
	s.chambers[b].adj = append(s.chambers[b].adj, t)
	return t
}

// adjacent reports whether a tunnel joins chambers a and b.
func (s *caveSystem) adjacent(a, b int) bool {
	for _, t := range s.chambers[a].adj {
		if tt := s.tunnels[t]; tt.a == b || tt.b == b {
			return true
		}
	}
	return false
}

// forkPoint is where on tunnel t, not yet wound, a passage from (x, y)
// would meet it: the nearest point of the line between its chambers, if
// that lies far enough from both.
func (s *caveSystem) forkPoint(t int, x, y float64) (float64, float64, bool) {
	tt := s.tunnels[t]
	a, b := s.chambers[tt.a], s.chambers[tt.b]
	f, _ := segDist(x, y, a.x, a.y, b.x, b.y)
	px, py := a.x+(b.x-a.x)*f, a.y+(b.y-a.y)*f
	if math.Hypot(px-a.x, py-a.y)-a.reach() < caveForkEnd || math.Hypot(px-b.x, py-b.y)-b.reach() < caveForkEnd {
		return 0, 0, false
	}
	return px, py, true
}

// forkAt splits tunnel t at (x, y) into two meeting at a new junction there,
// and returns the junction. A wound tunnel is split at point cut of its
// path (which lies at (x, y)); one not yet wound has cut -1.
func (s *caveSystem) forkAt(t int, x, y float64, cut int) int {
	tt := s.tunnels[t]
	j := len(s.chambers)
	s.chambers = append(s.chambers, caveChamber{
		x: x, y: y, r: caveJunctionR, aspect: 1, level: tt.level, zone: ZoneTunnel, junction: true, route: tt.route,
	})
	nt := caveTunnel{a: j, b: tt.b, level: tt.level, width: tt.width, route: tt.route}
	if cut >= 0 {
		nt.path = slices.Clone(tt.path[cut:])
		s.tunnels[t].path = slices.Clone(tt.path[:cut+1])
	}
	s.tunnels[t].b = j
	s.tunnels = append(s.tunnels, nt)
	n := len(s.tunnels) - 1
	for k, a := range s.chambers[tt.b].adj {
		if a == t {
			s.chambers[tt.b].adj[k] = n
		}
	}
	s.chambers[j].adj = []int{t, n}
	return j
}

// wayClear reports whether a tunnel of level lv along the line from (x0,
// y0) to (x1, y1) would cross none (but tunnel skip and those of the
// chambers in ends) nor, with through, run through a chamber (but those in
// ends). Without through it may graze a chamber, but never a fork.
func (s *caveSystem) wayClear(lv int, x0, y0, x1, y1 float64, skip int, through bool, ends ...int) bool {
	for ti, t := range s.tunnels {
		if t.level != lv || ti == skip || slices.Contains(ends, t.a) || slices.Contains(ends, t.b) {
			continue
		}
		p, q := s.chambers[t.a], s.chambers[t.b]
		if segmentsCross(x0, y0, x1, y1, p.x, p.y, q.x, q.y) {
			return false
		}
	}
	for i, c := range s.chambers {
		if c.level != lv || slices.Contains(ends, i) || !through && !c.junction {
			continue
		}
		if _, d := segDist(c.x, c.y, x0, y0, x1, y1); d < c.r+2 {
			return false
		}
	}
	return true
}

// joinWays digs the tunnels of level lv: grown from chamber from, each
// chamber in turn joined by the shortest tunnel to one joined already, or
// to the middle of a tunnel when that is shorter still; then a few short
// ones more that close loops, and a way round from every side branch that
// runs on too far.
func (s *caveSystem) joinWays(rng *rand.Rand, lv, from int) {
	var ids []int
	for i, c := range s.chambers {
		if c.level == lv {
			ids = append(ids, i)
		}
	}
	in := map[int]bool{from: true}
	type link struct {
		c, o, t int // chamber c to chamber o, or (o < 0) to tunnel t at (x, y)
		x, y, l float64
	}
	for left := len(ids) - 1; left > 0; left-- {
		var links []link
		for _, c := range ids {
			if in[c] {
				continue
			}
			cc := s.chambers[c]
			for _, o := range ids {
				if in[o] && !s.chambers[o].junction {
					links = append(links, link{c: c, o: o, t: -1, l: tunnelLen(cc, s.chambers[o])})
				}
			}
			for t, tt := range s.tunnels {
				if tt.level != lv {
					continue
				}
				if x, y, ok := s.forkPoint(t, cc.x, cc.y); ok {
					l := math.Hypot(x-cc.x, y-cc.y) - cc.r - caveJunctionR
					links = append(links, link{c: c, o: -1, t: t, x: x, y: y, l: l / caveForkBias})
				}
			}
		}
		if len(links) == 0 {
			break
		}
		slices.SortStableFunc(links, func(a, b link) int { return cmp.Compare(a.l, b.l) })
		pick := links[0]
		for _, k := range links {
			cc := s.chambers[k.c]
			if k.o >= 0 && s.wayClear(lv, cc.x, cc.y, s.chambers[k.o].x, s.chambers[k.o].y, -1, true, k.c, k.o) ||
				k.o < 0 && s.wayClear(lv, cc.x, cc.y, k.x, k.y, k.t, true, k.c) {
				pick = k
				break
			}
		}
		in[pick.c] = true
		to := pick.o
		if to < 0 {
			to = s.forkAt(pick.t, pick.x, pick.y, -1)
			ids = append(ids, to)
			in[to] = true
		}
		s.addTunnel(lv, pick.c, to, caveTunnelMin)
	}

	// Loops: a few short tunnels more between chambers near each other.
	type pair struct {
		a, b int
		d    float64
	}
	var pairs []pair
	longest := make(map[int]float64, len(ids))
	for _, t := range s.tunnels {
		if t.level == lv {
			l := tunnelLen(s.chambers[t.a], s.chambers[t.b])
			longest[t.a], longest[t.b] = max(longest[t.a], l), max(longest[t.b], l)
		}
	}
	for x, a := range ids {
		for _, b := range ids[x+1:] {
			if !s.chambers[a].junction && !s.chambers[b].junction {
				pairs = append(pairs, pair{a, b, tunnelLen(s.chambers[a], s.chambers[b])})
			}
		}
	}
	slices.SortStableFunc(pairs, func(p, q pair) int { return cmp.Compare(p.d, q.d) })
	for _, p := range pairs {
		if s.adjacent(p.a, p.b) || p.d > caveLoopMax || p.d > 1.6*max(longest[p.a], longest[p.b]) {
			continue
		}
		if rng.Float64() < caveWaysLoops && !s.crossesAny(lv, ids, p.a, p.b, true) {
			s.addTunnel(lv, p.a, p.b, caveTunnelMin)
		}
	}
	// A side branch that runs on too far gets a way round from its end.
	for pass := range 6 {
		reach := caveLoopMax * (1 + 0.4*float64(pass))
		for _, a := range ids {
			c := s.chambers[a]
			if len(c.adj) != 1 || c.hall || c.junction || s.branchLen(a) <= caveDeadEnd {
				continue
			}
			for _, p := range pairs {
				if p.a != a && p.b != a {
					continue
				}
				b := p.a + p.b - a
				if p.d <= reach && !s.adjacent(a, b) && !s.crossesAny(lv, ids, a, b, pass < 3) {
					s.addTunnel(lv, a, b, caveTunnelMin)
					break
				}
			}
		}
	}
}

// caveGraph is the chambers and tunnels of a system as a graph, with one
// node more than it has chambers, for a fork being weighed.
type caveGraph struct {
	n   int
	e   [][2]int
	pos [][2]float64 // where each node lies
}

func (s *caveSystem) graph() caveGraph {
	g := caveGraph{n: len(s.chambers) + 1, e: make([][2]int, len(s.tunnels)), pos: make([][2]float64, len(s.chambers)+1)}
	for i, t := range s.tunnels {
		g.e[i] = [2]int{t.a, t.b}
	}
	for i, c := range s.chambers {
		g.pos[i] = [2]float64{c.x, c.y}
	}
	return g
}

// dist is how far b lies from a through g, from node to node.
func (g caveGraph) dist(a, b int) float64 {
	adj := make([][]int, g.n)
	for i, e := range g.e {
		adj[e[0]] = append(adj[e[0]], i)
		adj[e[1]] = append(adj[e[1]], i)
	}
	d := make([]float64, g.n)
	for i := range d {
		d[i] = math.Inf(1)
	}
	d[a] = 0
	done := make([]bool, g.n)
	for {
		u, best := -1, math.Inf(1)
		for i, v := range d {
			if !done[i] && v < best {
				u, best = i, v
			}
		}
		if u < 0 || u == b {
			return d[b]
		}
		done[u] = true
		for _, e := range adj[u] {
			v := g.e[e][0] + g.e[e][1] - u
			if l := best + math.Hypot(g.pos[v][0]-g.pos[u][0], g.pos[v][1]-g.pos[u][1]); l < d[v] {
				d[v] = l
			}
		}
	}
}

// way is a way from a to b in g not through edge skip, as its edges; nil
// for none, empty when a is b.
func (g caveGraph) way(adj [][]int, a, b, skip int) []int {
	if a == b {
		return []int{}
	}
	via := make([]int, g.n)
	for i := range via {
		via[i] = -2
	}
	via[a] = -1
	q := []int{a}
	for n := 0; n < len(q); n++ {
		u := q[n]
		for _, e := range adj[u] {
			if e == skip {
				continue
			}
			v := g.e[e][0] + g.e[e][1] - u
			if via[v] != -2 {
				continue
			}
			via[v] = e
			if v == b {
				var out []int
				for x := b; x != a; {
					out = append(out, via[x])
					x = g.e[via[x]][0] + g.e[via[x]][1] - x
				}
				return out
			}
			q = append(q, v)
		}
	}
	return nil
}

// cut is how many of g's edges every way from a to b must take: 0 when
// there are two ways with none in common, -1 when there is none.
func (g caveGraph) cut(a, b int) int {
	adj := make([][]int, g.n)
	for i, e := range g.e {
		adj[e[0]] = append(adj[e[0]], i)
		adj[e[1]] = append(adj[e[1]], i)
	}
	way := g.way(adj, a, b, -1)
	if way == nil {
		return -1
	}
	n := 0
	for _, e := range way {
		if g.way(adj, a, b, e) == nil {
			n++
		}
	}
	return n
}

// ensureFork digs tunnels on level lv, as wide as the main route's, until
// there are two ways from chamber a to chamber b with no passage in
// common: each time the one that leaves fewest passages that must be taken
// for its length, between two chambers or from a chamber to the middle of
// a passage. None is a short cut: the way from a to b stays about as long.
// Where nothing short and clear is found, a longer one will do, then one
// grazing a chamber on its way.
func (s *caveSystem) ensureFork(lv, a, b int) {
	type cand struct {
		x, y, t int // chamber x to chamber y, or (y < 0) to tunnel t at (px, py)
		px, py  float64
		l       float64
	}
	for range caveForkRounds {
		g := s.graph()
		must := g.cut(a, b)
		if must <= 0 {
			return
		}
		route := g.dist(a, b)
		var pick *cand
		for pass, p := range caveForkPasses {
			var cs []cand
			for x, cx := range s.chambers {
				if cx.level != lv || cx.junction {
					continue
				}
				for y := x + 1; y < len(s.chambers); y++ {
					cy := s.chambers[y]
					if cy.level != lv || cy.junction || s.adjacent(x, y) {
						continue
					}
					if l := tunnelLen(cx, cy); l <= p.most {
						cs = append(cs, cand{x: x, y: y, t: -1, l: l})
					}
				}
				for t, tt := range s.tunnels {
					if tt.level != lv || tt.a == x || tt.b == x {
						continue
					}
					if px, py, ok := s.forkPoint(t, cx.x, cx.y); ok {
						if l := math.Hypot(px-cx.x, py-cx.y) - cx.r - caveJunctionR; l <= p.most {
							cs = append(cs, cand{x: x, y: -1, t: t, px: px, py: py, l: l})
						}
					}
				}
			}
			slices.SortStableFunc(cs, func(p, q cand) int { return cmp.Compare(p.l, q.l) })
			best, bestV := -1, math.Inf(1)
			for i, c := range cs {
				if (c.l+caveBypassSlack)/float64(must) >= bestV {
					break
				}
				cx := s.chambers[c.x]
				h := caveGraph{n: g.n, e: slices.Clone(g.e), pos: g.pos}
				if c.y >= 0 {
					if !s.wayClear(lv, cx.x, cx.y, s.chambers[c.y].x, s.chambers[c.y].y, -1, pass < 2, c.x, c.y) {
						continue
					}
					h.e = append(h.e, [2]int{c.x, c.y})
				} else {
					if !s.wayClear(lv, cx.x, cx.y, c.px, c.py, c.t, pass < 2, c.x) {
						continue
					}
					j, tt := g.n-1, s.tunnels[c.t]
					h.pos = slices.Clone(g.pos)
					h.pos[j] = [2]float64{c.px, c.py}
					h.e[c.t] = [2]int{tt.a, j}
					h.e = append(h.e, [2]int{j, tt.b}, [2]int{c.x, j})
				}
				left := h.cut(a, b)
				if left < 0 || left >= must || h.dist(a, b) < route*p.short {
					continue
				}
				if v := (c.l + caveBypassSlack) / float64(must-left); v < bestV {
					best, bestV = i, v
				}
			}
			if best >= 0 {
				pick = &cs[best]
				break
			}
		}
		if pick == nil {
			return
		}
		to := pick.y
		if to < 0 {
			to = s.forkAt(pick.t, pick.px, pick.py, -1)
		}
		s.addTunnel(lv, pick.x, to, caveRouteMin)
	}
}

// caveForkPasses are how long a bypass may be, and how much shorter it may
// make the way through, each time nothing was found before; from the third
// pass on it may graze a chamber, and on the last it may be a short cut
// (where the far end lies near the way in, but the way to it winds).
var caveForkPasses = []struct{ most, short float64 }{
	{caveBypassMax, caveForkShort},
	{caveBypassMax * 1.6, 0.8},
	{caveBypassMax * 2, 0.7},
	{caveBypassMax * 2, 0},
}

// levelEnds are the way into level lv of s and the way on: the first hall
// and the hole down (or the destination), the rope's foot and the
// destination.
func (s *caveSystem) levelEnds(lv int) (int, int) {
	switch {
	case lv == 1:
		return s.arrival, s.dest
	case s.levels == 2:
		return 0, s.descent
	}
	return 0, s.dest
}

// routeAlong is the main route from chamber a to chamber b (its tunnels
// marked by markRoute), the chambers and forks on it in order, and how far
// along it each lies.
func (s *caveSystem) routeAlong(a, b int) ([]int, []float64) {
	via := make([]int, len(s.chambers)) // the tunnel each was reached by, along the route
	for i := range via {
		via[i] = -1
	}
	q := []int{a}
	via[a] = -2
	for n := 0; n < len(q) && via[b] == -1; n++ {
		for _, t := range s.chambers[q[n]].adj {
			if o := s.tunnels[t].a + s.tunnels[t].b - q[n]; s.tunnels[t].route && via[o] == -1 {
				via[o] = t
				q = append(q, o)
			}
		}
	}
	if via[b] == -1 {
		return []int{a}, []float64{0}
	}
	out := []int{b}
	for cur := b; cur != a; {
		t := s.tunnels[via[cur]]
		cur = t.a + t.b - cur
		out = append(out, cur)
	}
	slices.Reverse(out)
	at := make([]float64, len(out))
	for i := 1; i < len(out); i++ {
		p, c := s.chambers[out[i-1]], s.chambers[out[i]]
		at[i] = at[i-1] + math.Hypot(c.x-p.x, c.y-p.y)
	}
	return out, at
}

// thirdOf is which third of a route total long a point pos along it lies
// in; a short route is one stretch (see routeThirds).
func thirdOf(pos, total float64) int {
	if total < caveThirdsFrom {
		return 0
	}
	return clamp1(int(3*pos/total), 0, 2)
}

// routeThirds is how many stretches a route total long is told in, each
// with some way leading off it: its thirds, or one for a short route (a
// tunnel from the hall to the far end of a small cave, with no room for a
// fork in its middle).
func routeThirds(total float64) int {
	if total < caveThirdsFrom {
		return 1
	}
	return 3
}

func clamp1(v, lo, hi int) int { return max(lo, min(hi, v)) }

// sideIn reports whether a passage leaves the route (its chambers in order,
// and how far along each lies) in the given third of it.
func (s *caveSystem) sideIn(route []int, at []float64, third int) bool {
	total := at[len(at)-1]
	for k, c := range route {
		if thirdOf(at[k], total) != third {
			continue
		}
		for _, t := range s.chambers[c].adj {
			tt := s.tunnels[t]
			o := tt.a + tt.b - c
			if k > 0 && o == route[k-1] || k+1 < len(route) && o == route[k+1] {
				continue
			}
			return true
		}
	}
	return false
}

// pocketAnchor is where a spur to a side pocket leaves from: chamber c, or
// (c < 0) point j of the path of tunnel t.
type pocketAnchor struct{ c, t, j int }

// spurOf reports whether tunnel t leads to a side pocket.
func (s *caveSystem) spurOf(t int) bool {
	tt := s.tunnels[t]
	return s.chambers[tt.a].pocket != 0 || s.chambers[tt.b].pocket != 0
}

// forkSpots are the points of tunnel t's path a spur may leave from: far
// enough from the chambers at both ends. Each comes with how far along the
// path it lies, as a share of the path's length.
func (s *caveSystem) forkSpots(t int) (js []int, along []float64) {
	tt := s.tunnels[t]
	p := tt.path
	if len(p) < 3 {
		return nil, nil
	}
	arc := make([]float64, len(p))
	for j := 1; j < len(p); j++ {
		arc[j] = arc[j-1] + math.Hypot(p[j][0]-p[j-1][0], p[j][1]-p[j-1][1])
	}
	total := arc[len(p)-1]
	ra, rb := s.chambers[tt.a].reach()+caveForkEnd, s.chambers[tt.b].reach()+caveForkEnd
	for j := 1; j+1 < len(p); j++ {
		if arc[j] >= ra && total-arc[j] >= rb {
			js = append(js, j)
			along = append(along, arc[j]/total)
		}
	}
	return js, along
}

// pocketHost reports whether a spur to a side pocket may leave chamber i:
// a chamber with two ways or more, not a hall, a fork, a pocket, the end of
// a side branch or one with a way down or up.
func (s *caveSystem) pocketHost(i int) bool {
	c := s.chambers[i]
	return !c.hall && !c.junction && c.pocket == 0 && !c.leaf && i != s.descent && i != s.arrival && !s.shaft(i)
}

// addPockets digs the side pockets of s, its tunnels wound: first a way
// leading off each third of every level's main route that has none, then
// more here and there, as many as the island is big, and more still on a
// level of four chambers or more until two places at least have three ways
// or more to choose from.
func (w *World) addPockets(rng *rand.Rand, s *caveSystem) {
	seed := s.digSeed(w)
	kind := w.islands[s.island].Kind
	try := func(lv int, anchors []pocketAnchor, tries int) bool {
		if len(anchors) == 0 {
			return false
		}
		rng.Shuffle(len(anchors), func(x, y int) { anchors[x], anchors[y] = anchors[y], anchors[x] })
		for k := range tries {
			if w.pocketAt(rng, s, lv, anchors[k%len(anchors)], seed, kind, k >= tries/2) {
				return true
			}
		}
		return false
	}
	for lv := range s.levels {
		a, b := s.levelEnds(lv)
		_, at := s.routeAlong(a, b)
		for third := range routeThirds(at[len(at)-1]) {
			route, at := s.routeAlong(a, b) // anew: a pocket off a tunnel of it splits the tunnel
			if at[len(at)-1] > 0 && !s.sideIn(route, at, third) {
				try(lv, s.routeAnchors(route, at, third), caveThirdTries)
			}
		}
		per := cavePocketUpper
		if lv == 1 {
			per = cavePocketLower
		}
		want := min(cavePocketsMost, s.area/per)
		if lv == 0 && s.area >= caveWaysFewFrom {
			want = max(want, 1)
		}
		for placed, tries := 0, 0; placed < want && tries < want*8; tries++ {
			var an []pocketAnchor
			if rng.IntN(2) == 0 {
				an = s.chamberAnchors(lv, false)
			} else {
				an = s.tunnelAnchors(lv)
			}
			if len(an) > 0 && w.pocketAt(rng, s, lv, an[rng.IntN(len(an))], seed, kind, false) {
				placed++
			}
		}
		if s.chambersOn(lv) >= 4 {
			for range 3 {
				if s.forksOn(lv) >= 2 {
					break
				}
				if !try(lv, s.chamberAnchors(lv, true), caveThirdTries/2) {
					try(lv, s.tunnelAnchors(lv), caveThirdTries/2)
				}
			}
		}
	}
}

// routeAnchors are the places a spur may leave the main route (its chambers
// in order, and how far along each lies) from in the given third of it:
// its chambers, and points along its tunnels.
func (s *caveSystem) routeAnchors(route []int, at []float64, third int) []pocketAnchor {
	total := at[len(at)-1]
	var out []pocketAnchor
	for k, c := range route {
		if thirdOf(at[k], total) == third && s.pocketHost(c) {
			out = append(out, pocketAnchor{c: c, t: -1})
		}
		if k+1 == len(route) {
			break
		}
		for _, t := range s.chambers[c].adj {
			tt := s.tunnels[t]
			if tt.a+tt.b-c != route[k+1] {
				continue
			}
			js, along := s.forkSpots(t)
			for n, j := range js {
				f := along[n]
				if tt.a != c {
					f = 1 - f
				}
				// Well inside the third: the path winds, and a fork there
				// moves the chambers' middles along the route a little.
				pos := at[k] + f*(at[k+1]-at[k])
				if u := float64(routeThirds(total))*pos/total - float64(third); thirdOf(pos, total) == third && u >= 0.12 && u <= 0.88 {
					out = append(out, pocketAnchor{c: -1, t: t, j: j})
				}
			}
		}
	}
	return out
}

// chamberAnchors are the chambers of level lv a spur may leave from; with
// two, only those with two ways, that one more makes a fork.
func (s *caveSystem) chamberAnchors(lv int, two bool) []pocketAnchor {
	var out []pocketAnchor
	for i, c := range s.chambers {
		if c.level == lv && s.pocketHost(i) && (!two || len(c.adj) == 2) {
			out = append(out, pocketAnchor{c: i, t: -1})
		}
	}
	return out
}

// tunnelAnchors are the points of the tunnels of level lv a spur may leave
// from, but those to the pockets.
func (s *caveSystem) tunnelAnchors(lv int) []pocketAnchor {
	var out []pocketAnchor
	for t, tt := range s.tunnels {
		if tt.level != lv || s.spurOf(t) {
			continue
		}
		js, _ := s.forkSpots(t)
		for _, j := range js {
			out = append(out, pocketAnchor{c: -1, t: t, j: j})
		}
	}
	return out
}

// chambersOn is how many chambers level lv has, with the halls and the
// arrival, but not the forks nor the pockets.
func (s *caveSystem) chambersOn(lv int) int {
	n := 0
	for _, c := range s.chambers {
		if c.level == lv && !c.junction && c.pocket == 0 {
			n++
		}
	}
	return n
}

// forksOn is how many chambers and forks of level lv have three ways or
// more.
func (s *caveSystem) forksOn(lv int) int {
	n := 0
	for _, c := range s.chambers {
		if c.level == lv && len(c.adj) >= 3 {
			n++
		}
	}
	return n
}

// pocketAt tries to dig a side pocket of level lv off anchor an: a spur
// winding out from it to a small grotto, clear of everything else. It
// reports whether it did.
func (w *World) pocketAt(rng *rand.Rand, s *caveSystem, lv int, an pocketAnchor, seed uint32, isle IslandKind, tight bool) bool {
	what := pocketOre
	switch u := rng.Float64(); {
	case u < 0.3:
	case u < 0.58:
		what = pocketMushroom
	case u < 0.8:
		what = pocketPool
	default:
		what = pocketBones
	}
	rp := cavePocketMin + (cavePocketMax-cavePocketMin)*rng.Float64()
	if what == pocketPool {
		rp = max(rp, 4.5)
	}
	spur := caveSpurMin + (caveSpurMax-caveSpurMin)*rng.Float64()
	wall := caveWaysWall
	if tight { // where there is little room: the smallest pocket, close by
		if what == pocketPool {
			what = pocketMushroom
		}
		rp, spur, wall = cavePocketMin, caveSpurMin+3*rng.Float64(), caveWaysWall-1
	}
	var fx, fy, sx, sy, dx, dy float64 // the spur's start, where it leaves the anchor, and its way
	skip := 0.0                        // the length of spur that lies in the anchor
	if an.c >= 0 {
		c := s.chambers[an.c]
		a := rng.Float64() * 2 * math.Pi
		dx, dy = math.Cos(a), math.Sin(a)
		e := s.edgeAlong(an.c, dx, dy, seed)
		fx, fy = c.x, c.y
		sx, sy, skip = c.x+dx*e, c.y+dy*e, e+1
	} else {
		p := s.tunnels[an.t].path
		tx, ty := p[an.j+1][0]-p[an.j-1][0], p[an.j+1][1]-p[an.j-1][1]
		l := math.Hypot(tx, ty)
		if l == 0 {
			return false
		}
		side := 1.0
		if rng.IntN(2) == 0 {
			side = -1
		}
		a := math.Atan2(tx*side, -ty*side) + (rng.Float64()-0.5)*0.9
		dx, dy = math.Cos(a), math.Sin(a)
		fx, fy = p[an.j][0], p[an.j][1]
		sx, sy, skip = fx, fy, 3.5
	}
	cx, cy := sx+dx*(spur+rp), sy+dy*(spur+rp)
	m := rp*1.6 + 3
	if cx-m < float64(s.x0+2) || cy-m < float64(s.y0+2) || cx+m > float64(s.x0+s.gw-3) || cy+m > float64(s.y0+s.gh-3) {
		return false
	}
	// The spur ends on the pocket's near side, so its middle stays free.
	path := windPath(rng, [][2]float64{{fx, fy}, {cx - dx*rp*0.5, cy - dy*rp*0.5}}, 0.15)
	if !s.pocketClear(lv, an, cx, cy, rp, path, skip, wall, seed) {
		return false
	}
	z := ZoneHall
	switch what {
	case pocketOre:
		if lv == 1 {
			z = ZoneCrystal
		}
	case pocketMushroom:
		z = ZoneMushroom
	case pocketPool:
		z = ZoneLake
	case pocketBones:
		z = ZoneFossil
	}
	p := len(s.chambers)
	s.chambers = append(s.chambers, caveChamber{
		x: cx, y: cy, r: rp, aspect: 0.85 + 0.3*rng.Float64(), rot: rng.Float64() * math.Pi,
		level: lv, zone: z, pocket: what,
	})
	s.chambers[p].rock = w.rockAbove(&s.chambers[p], isle)
	from := an.c
	if from < 0 {
		from = s.forkAt(an.t, fx, fy, an.j)
	}
	t := s.addTunnel(lv, from, p, caveTunnelMin)
	s.tunnels[t].path = path
	return true
}

// pocketClear reports whether a pocket of radius rp at (cx, cy) on level lv,
// reached by a spur along path (whose first skip tiles lie in its anchor),
// keeps a wall of rock round it and its spur from every other chamber and
// tunnel.
func (s *caveSystem) pocketClear(lv int, an pocketAnchor, cx, cy, rp float64, path [][2]float64, skip, wall float64, seed uint32) bool {
	var spur, far [][2]float64 // the spur's points past its anchor, and those well clear of it
	along := 0.0
	for j, q := range path {
		if j > 0 {
			along += math.Hypot(q[0]-path[j-1][0], q[1]-path[j-1][1])
		}
		if along >= skip {
			spur = append(spur, q)
		}
		if along >= skip+caveTunnelMax {
			far = append(far, q)
		}
	}
	x0, y0, x1, y1 := cx-rp, cy-rp, cx+rp, cy+rp
	for _, q := range spur {
		x0, y0, x1, y1 = min(x0, q[0]), min(y0, q[1]), max(x1, q[0]), max(y1, q[1])
	}
	pr := rp * 1.3 // the pocket's reach, with the noise on its edge
	for i, o := range s.chambers {
		if o.level != lv {
			continue
		}
		most := o.reach()*1.6 + 1 // o's wall lies no further than this from its middle
		switch {
		case o.junction:
			most = caveJunctionR
		case o.pocket != 0:
			most = o.r * 1.3
		}
		// clear reports whether (x, y) lies more than gap outside o's wall.
		clear := func(x, y, gap float64) bool {
			d := math.Hypot(x-o.x, y-o.y)
			if d-most >= gap {
				return true
			}
			if o.junction || o.pocket != 0 || d == 0 {
				return false
			}
			return d-s.edgeAlong(i, (x-o.x)/d, (y-o.y)/d, seed) >= gap
		}
		if R := most + pr + wall; o.x+R < x0 || o.x-R > x1 || o.y+R < y0 || o.y-R > y1 {
			continue
		}
		if !clear(cx, cy, pr+wall) {
			return false
		}
		if i == an.c {
			continue
		}
		for _, q := range spur {
			if !clear(q[0], q[1], wall+1.7) {
				return false
			}
		}
	}
	for ti, t := range s.tunnels {
		if t.level != lv || len(t.path) == 0 {
			continue
		}
		pts := spur
		if an.c < 0 && ti == an.t || an.c >= 0 && (t.a == an.c || t.b == an.c) {
			pts = far // the spur leaves where this one does: only its points well away
		}
		bx0, by0, bx1, by1 := t.path[0][0], t.path[0][1], t.path[0][0], t.path[0][1]
		for _, q := range t.path {
			bx0, by0, bx1, by1 = min(bx0, q[0]), min(by0, q[1]), max(bx1, q[0]), max(by1, q[1])
		}
		m := pr + caveTunnelMax/2 + wall
		if bx1+m < x0 || bx0-m > x1 || by1+m < y0 || by0-m > y1 {
			continue
		}
		for j := 0; j+1 < len(t.path); j++ {
			p, q := t.path[j], t.path[j+1]
			if _, d := segDist(cx, cy, p[0], p[1], q[0], q[1]); d < m {
				return false
			}
			for _, u := range pts {
				if _, d := segDist(u[0], u[1], p[0], p[1], q[0], q[1]); d < caveTunnelMax+1.5 {
					return false
				}
			}
		}
	}
	return true
}

// windPath bends a line again and again, each half's middle pushed aside
// by up to bend of its length, as the tunnels wind.
func windPath(rng *rand.Rand, pts [][2]float64, bend float64) [][2]float64 {
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
			off := (rng.Float64()*2 - 1) * l * bend
			out = append(out, [2]float64{(p[0]+q[0])/2 - dy/l*off, (p[1]+q[1])/2 + dx/l*off})
		}
		pts = append(out, pts[len(pts)-1])
	}
	return pts
}

// digSeed is the seed of the noise system s is dug by.
func (s *caveSystem) digSeed(w *World) uint32 { return uint32(w.seed) ^ uint32(s.island+1)*0x9e3779b1 }

// edgeAlong is how far from the middle of chamber i its floor reaches the
// way (dx, dy), a unit vector, as it will be dug.
func (s *caveSystem) edgeAlong(i int, dx, dy float64, seed uint32) float64 {
	c := &s.chambers[i]
	if c.junction {
		return caveJunctionR
	}
	R := c.r*1.5*max(c.aspect, 1/c.aspect) + 1
	out := 0.0
	for d := 0.0; d <= R; d += 0.5 {
		if s.inBlob(c, c.x+dx*d, c.y+dy*d, seed) {
			out = d
		}
	}
	return out
}

// The old chests of the caves of StyleCaveWays, by where they lie. Their
// Var is the kind times four plus the ring of the island above (0 home, 1
// near, 2 far): what is in them is decided when they are first opened, by
// caveWaysLoot.
const (
	chestCache = iota // a small cache of supplies: in a side pocket, or an islet's hall
	chestSide         // at the end of a side branch, or off the main route
	chestEnd          // at the end of the main route of a cave of one level
	chestDeep         // the hoard at the end of the lower level's main route, where a beast will stand guard
	numChestKinds
)

// Cave ways chest tuning.
const (
	caveLeafChest    = 0.27  // chance of a chest at the end of a side branch
	cavePocketChest  = 0.1   // and in a side pocket
	caveIsletChest   = 0.35  // and in an islet's hall
	caveWaysHoardPer = 12000 // floor tiles of the upper level to each more chest off the main route (half as many below)
)

// waysChests hides the old chests of one level of a system of
// StyleCaveWays, and dresses its pockets: the hoard at the end of the main
// route, a chest at the end of some side branches, a cache in a few
// pockets and a few more off the main route.
func (w *World) waysChests(rng *rand.Rand, fl *caveFloor) {
	s := fl.s
	ring := 1
	if k := s.island; k < len(w.caves.rings) {
		ring = int(w.caves.rings[k])
	}
	chest := func(ci, kind int) { w.caveHoard(rng, fl, ci, kind<<2|ring) }
	for ci, c := range s.chambers {
		if c.level != fl.lv || c.junction {
			continue
		}
		switch {
		case ci == s.dest && c.hall: // a cave of one hall, on an islet
			if rng.Float64() < caveIsletChest {
				chest(ci, chestCache)
			}
		case ci == s.dest && s.levels == 2:
			chest(ci, chestDeep)
		case ci == s.dest:
			chest(ci, chestEnd)
		case c.pocket != 0:
			if c.pocket == pocketOre && fl.lv == 0 {
				w.caveSeams(rng, fl, fl.cellsOf(ci), EntityCoal, 1, [2]int{3, 5})
			}
			if rng.Float64() < cavePocketChest {
				chest(ci, chestCache)
			}
		case c.leaf:
			if rng.Float64() < caveLeafChest {
				chest(ci, chestSide)
			}
		}
	}
	var off []int
	for ci, c := range s.chambers {
		if c.level == fl.lv && !c.hall && !c.route && !c.leaf && !c.junction && c.pocket == 0 && ci != s.dest {
			off = append(off, ci)
		}
	}
	rng.Shuffle(len(off), func(a, b int) { off[a], off[b] = off[b], off[a] })
	for _, ci := range off[:min(len(off), s.floor[fl.lv]/(caveWaysHoardPer>>fl.lv))] {
		chest(ci, chestSide)
	}
}

// caveLoot is a thing an old chest of StyleCaveWays may hold: how many,
// the chance of it by the kind of chest, and the nearest ring of islands it
// is found under.
type caveLoot struct {
	item   ItemKind
	lo, hi int
	p      [numChestKinds]float64
	ring   int
}

// caveSupplies are what the old chests of StyleCaveWays mostly hold: what
// a caver runs short of, with ore by how far the island lies from home.
var caveSupplies = []caveLoot{
	{ItemTorch, 2, 4, [numChestKinds]float64{0.5, 0.5, 0.6, 0.7}, 0},
	{ItemCoal, 2, 5, [numChestKinds]float64{0.4, 0.45, 0.5, 0.5}, 0},
	{ItemBone, 2, 4, [numChestKinds]float64{0.35, 0.4, 0.4, 0.4}, 0},
	{ItemCopperOre, 2, 4, [numChestKinds]float64{0.3, 0.35, 0.35, 0.4}, 0}, // flint
	{ItemPotion, 1, 1, [numChestKinds]float64{0.3, 0.45, 0.6, 0.8}, 0},
	{ItemBandage, 1, 2, [numChestKinds]float64{0.1, 0.2, 0.3, 0.4}, 0},
	{ItemRope, 1, 2, [numChestKinds]float64{0.25, 0.3, 0.3, 0.35}, 0},
	{ItemFlax, 2, 4, [numChestKinds]float64{0.25, 0.25, 0.2, 0.2}, 0},
	{ItemTreeSeed, 1, 3, [numChestKinds]float64{0.15, 0.2, 0.2, 0.2}, 0},
	{ItemGlowSeed, 1, 3, [numChestKinds]float64{0.15, 0.2, 0.25, 0.25}, 0},
	{ItemBerrySeed, 1, 3, [numChestKinds]float64{0.15, 0.15, 0.15, 0.15}, 0},
	{ItemIronOre, 1, 2, [numChestKinds]float64{0.05, 0.25, 0.35, 0.5}, 1},     // jade
	{ItemPlatinumOre, 1, 1, [numChestKinds]float64{0, 0.1, 0.15, 0.3}, 1},     // agate
	{ItemDinoHide, 1, 2, [numChestKinds]float64{0, 0.1, 0.15, 0.25}, 1},       //
	{ItemDiamond, 1, 1, [numChestKinds]float64{0, 0.04, 0.08, 0}, 2},          // obsidian, now and then
	{ItemManEaterSeed, 1, 1, [numChestKinds]float64{0, 0.05, 0.08, 0.1}, 1},   //
	{ItemCookedMeat, 1, 2, [numChestKinds]float64{0.15, 0.15, 0.15, 0.15}, 0}, // dried
}

// caveBest are the best things an old chest of StyleCaveWays may hold, only
// in the hoard at the end of the lower level, and one of them at most: the
// chance of one by ring, and their weights among themselves by ring.
var (
	caveBestChance = [3]float64{0, 0.4, 0.65}
	caveBest       = []struct {
		item   ItemKind
		lo, hi int
		weight [3]float64
	}{
		{ItemBigPotion, 1, 1, [3]float64{0, 0.65, 0.4}},
		{ItemDiamond, 2, 3, [3]float64{0, 0, 0.3}}, // obsidian in quantity
		{ItemRexTooth, 1, 1, [3]float64{0, 0.22, 0.18}},
		{ItemIvorySpear, 1, 1, [3]float64{0, 0.13, 0.12}},
	}
)

// caveWaysLoot is what an old chest of kind under an island of ring holds,
// in the caves of StyleCaveWays, decided by its id.
func caveWaysLoot(id EntityID, kind, ring int) []Ingredient {
	kind, ring = clamp1(kind, 0, numChestKinds-1), clamp1(ring, 0, 2)
	r := rand.New(rand.NewPCG(uint64(id), uint64(kind*3+ring)*7919+0xca7e))
	var out []Ingredient
	for _, t := range caveSupplies {
		if ring >= t.ring && r.Float64() < t.p[kind] {
			out = append(out, Ingredient{Item: t.item, Count: t.lo + r.IntN(t.hi-t.lo+1)})
		}
	}
	if kind == chestDeep && r.Float64() < caveBestChance[ring] {
		sum := 0.0
		for _, b := range caveBest {
			sum += b.weight[ring]
		}
		v, pick := r.Float64()*sum, -1
		for i, b := range caveBest {
			if b.weight[ring] <= 0 {
				continue
			}
			pick = i
			if v -= b.weight[ring]; v < 0 {
				break
			}
		}
		if pick >= 0 {
			b := caveBest[pick]
			out = append(out, Ingredient{Item: b.item, Count: b.lo + r.IntN(b.hi-b.lo+1)})
		}
	}
	if len(out) == 0 {
		out = append(out, Ingredient{Item: ItemTorch, Count: 2})
	}
	return out
}

// chestLoot is what old chest e holds when it is first opened: by the
// rules of StyleCaveWays in its caves, else by its depth.
func (w *World) chestLoot(e *Entity) []Ingredient {
	if w.caveWays() && w.caveLevel(e.TY) >= 0 {
		return caveWaysLoot(e.ID, int(e.Var>>2), int(e.Var&3))
	}
	return treasureLoot(e.ID, int(e.Var))
}
