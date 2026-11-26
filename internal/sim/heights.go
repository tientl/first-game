package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// Heights. The land of a StyleIsles world rises in levels, 0 at the shore
// to MaxLevel on the highest crags. Where one level drops to the next the
// edge is mostly a cliff, which nobody climbs; elsewhere it is a slope,
// walked up and down. Cut through the cliffs are ramps, enough that every
// part of an island can be walked to from every other, rivers crossed at
// fords; see joinLand.

// levelCuts is where island s's heights step up a level.
func levelCuts(s *isle) []float64 {
	switch {
	case s.role == roleIslet || s.role == roleAtoll || s.r < smallIsle:
		return nil
	case s.role == roleHome:
		return []float64{0.5, 0.83}
	case s.role == roleMedium:
		switch s.kind {
		case IslandForest, IslandSnow, IslandDesert:
			return []float64{0.52}
		}
		return nil
	}
	switch s.kind {
	case IslandMonkey:
		return []float64{0.36, 0.6, 0.8}
	case IslandSnow:
		return []float64{0.3, 0.55, 0.78}
	case IslandDesert:
		return []float64{0.4, 0.7}
	case IslandAsh:
		return []float64{0.33, 0.62}
	case IslandMarsh:
		return []float64{0.8}
	case IslandMeadow:
		return []float64{0.55, 0.85}
	}
	return []float64{0.45, 0.72}
}

// cliffShare is, for island s, how readily a step between levels is a
// cliff (higher: more cliffs), and how much of its coast is cliffs.
func cliffShare(s *isle) (steps, coast float64) {
	switch {
	case s.role == roleIslet || s.role == roleAtoll || s.r < smallIsle:
		return 0, 0
	case s.role == roleHome:
		return 0.5, 0.12
	case s.role == roleMedium:
		return 0.5, 0.2
	}
	switch s.kind {
	case IslandDesert:
		return 0.75, 0.3
	case IslandSnow:
		return 0.65, 0.45
	case IslandMonkey:
		return 0.7, 0.4
	case IslandAsh:
		return 0.55, 0.3
	case IslandMarsh:
		return 0.3, 0
	}
	return 0.5, 0.2
}

// Height tuning.
const (
	smallIsle     = 18  // tiles of radius: a smaller island is low and level all over
	homeCalm      = 0.4 // of the home island's radius: flat round its middle, for the spawn
	minLevelPatch = 24  // a patch of one level smaller than this joins its neighbours
	rampEvery     = 40  // tiles of cliff with no ramp before one more is cut, give or take
	fordEvery     = 36  // tiles of river with no ford before one more is laid
)

// lakeAt reports whether tile i is a lake (still fresh water).
func (w *World) lakeAt(i int) bool {
	return w.tiles[i] == TileWater && w.relief[i].Variant() == VarLake
}

// seaAt reports whether tile i is the sea, deep or shallow (not a lake).
func (w *World) seaAt(i int) bool { return w.tiles[i].Sea() && !w.lakeAt(i) }

// evenHeights scales each island's heights to run from 0 at its lowest
// land to 1 at its top.
func (w *World) evenHeights(im *isleMap) {
	lo := make([]float32, len(im.isles)+1)
	hi := make([]float32, len(im.isles)+1)
	for k := range lo {
		lo[k], hi[k] = math.MaxFloat32, -math.MaxFloat32
	}
	for i, t := range w.tiles {
		if t.Sea() {
			continue
		}
		o := im.owner[i]
		lo[o], hi[o] = min(lo[o], im.h[i]), max(hi[o], im.h[i])
	}
	for i, t := range w.tiles {
		o := im.owner[i]
		if t.Sea() || hi[o] <= lo[o] {
			im.h[i] = 0
			continue
		}
		im.h[i] = (im.h[i] - lo[o]) / (hi[o] - lo[o])
	}
}

// seaDistance is, for every tile, how many steps it lies from the sea (0
// on the sea), up to most.
func (w *World) seaDistance(most int) []uint8 {
	W, H := w.width, w.height
	d := make([]uint8, len(w.tiles))
	var q []int32
	for i := range w.tiles {
		if w.seaAt(i) {
			q = append(q, int32(i))
		} else {
			d[i] = uint8(most)
		}
	}
	for k := 0; k < len(q); k++ {
		i := int(q[k])
		x, y := i%W, i/W
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+n[0], y+n[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			if j := ny*W + nx; d[j] > d[i]+1 {
				d[j] = d[i] + 1
				q = append(q, int32(j))
			}
		}
	}
	return d
}

// isleLevels works out the height level of every land tile; beach (nil
// before StyleBeaches) keeps the ground on and behind a beach level.
func (w *World) isleLevels(im *isleMap, toSea []uint8, beach *beachField, seed uint32) []uint8 {
	W, H := w.width, w.height
	lev := make([]uint8, len(w.tiles))
	for i, t := range w.tiles {
		if t.Sea() {
			continue
		}
		s := im.isleAt(i)
		if s == nil {
			continue
		}
		x, y := float64(i%W)+0.5, float64(i/W)+0.5
		h := float64(im.h[i]) + 0.05*(fbm(x/8, y/8, seed+71, 2)-0.5)
		l := 0
		for _, c := range levelCuts(s) {
			if h > c {
				l++
			}
		}
		_, coast := cliffShare(s)
		sea := int(toSea[i])
		if sea <= 2 || beach.level(i) {
			l = 0
		}
		if coast > 0 && sea <= 4+int(3*fbm(x/11, y/11, seed+72, 2)) && s.cliffCoast(x, y, coast) {
			l = max(l, 1) // a cliff coast: the land stands high over the sea
		}
		if s.flat[2] > 0 && math.Hypot(x-s.flat[0], y-s.flat[1]) < s.flat[2] {
			l = 0 // round the spawn, or the pyramid's plain
		}
		lev[i] = uint8(min(l, MaxLevel))
	}
	// Even out the ragged edges, then fold small patches into their
	// neighbours, so the levels are broad terraces.
	land := func(i int) bool { return !w.tiles[i].Sea() }
	for range 2 {
		next := slices.Clone(lev)
		for y := 1; y < H-1; y++ {
			for x := 1; x < W-1; x++ {
				i := y*W + x
				if !land(i) {
					continue
				}
				var votes [MaxLevel + 1]int
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if j := i + dy*W + dx; land(j) {
							votes[lev[j]]++
						}
					}
				}
				best := int(lev[i])
				for l, v := range votes {
					if v > votes[best] {
						best = l
					}
				}
				if votes[best] >= 5 {
					next[i] = uint8(best)
				}
			}
		}
		lev = next
	}
	seen := make([]bool, len(lev))
	var comp []int
	for start := range lev {
		if seen[start] || !land(start) {
			continue
		}
		comp = append(comp[:0], start)
		seen[start] = true
		var around [MaxLevel + 1]int
		for k := 0; k < len(comp); k++ {
			x, y := comp[k]%W, comp[k]/W
			for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+n[0], y+n[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				j := ny*W + nx
				switch {
				case !land(j):
				case lev[j] == lev[start]:
					if !seen[j] {
						seen[j] = true
						comp = append(comp, j)
					}
				default:
					around[lev[j]]++
				}
			}
		}
		if len(comp) >= minLevelPatch {
			continue
		}
		best := -1
		for l, v := range around {
			if v > 0 && (best < 0 || v > around[best]) {
				best = l
			}
		}
		if best >= 0 {
			for _, i := range comp {
				lev[i] = uint8(best)
			}
		}
	}
	return lev
}

// raiseCliffs turns the edges of the levels into cliffs, where the island
// is steep enough, and every edge of high land over the sea; loose scree
// lies at their feet.
func (w *World) raiseCliffs(im *isleMap, seed uint32) {
	W, H := w.width, w.height
	var cliffs []int
	for i, t := range w.tiles {
		if t.Sea() || w.lakeAt(i) {
			continue
		}
		s := im.isleAt(i)
		if s == nil {
			continue
		}
		l := w.relief[i].Level()
		if l == 0 {
			continue
		}
		x, y := i%W, i/W
		steps, _ := cliffShare(s)
		sea, drop := false, false
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+n[0], y+n[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			j := ny*W + nx
			switch {
			case w.seaAt(j):
				sea = true
			case w.relief[j].Level() < l:
				drop = true
			}
		}
		fx, fy := float64(x)+0.5, float64(y)+0.5
		steep := fbm(fx/22+5, fy/22+3, seed+81, 2)+0.06*(unit(hash2(int32(x), int32(y), seed+82))-0.5) > 1-steps
		if sea || drop && steep {
			cliffs = append(cliffs, i)
		}
	}
	for _, i := range cliffs {
		w.tiles[i] = TileCliff
		w.relief[i] = w.relief[i].withVariant(VarPlain)
	}
	// Scree at their feet.
	for _, i := range cliffs {
		x, y := i%W, i/W
		for _, n := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
			nx, ny := x+n[0], y+n[1]
			if !w.inBounds(nx, ny) {
				continue
			}
			j := ny*W + nx
			if t := w.tiles[j]; t.Walkable() && w.relief[j].Level() < w.relief[i].Level() && w.relief[j].Variant() == VarPlain &&
				unit(hash2(int32(nx), int32(ny), seed+83)) < 0.4 {
				w.relief[j] = w.relief[j].withVariant(VarScree)
			}
		}
	}
}

// bridgeable reports whether a way may be cut through tile t to join two
// parts of an island: a ramp through a cliff, a ford over a river, a crust
// over lava.
func (w *World) bridgeable(i int) bool {
	switch w.tiles[i] {
	case TileCliff:
		return w.relief[i].Variant() != VarStack
	case TileRiver, TileLava:
		return true
	}
	return false
}

// bridge makes tile i walkable: a ramp, a ford or cooled lava. ground is
// the tile on the high side of a ramp, whose ground it takes.
func (w *World) bridge(i int, ground Tile) {
	switch w.tiles[i] {
	case TileCliff:
		if !ground.Walkable() || ground == TileFord || ground == TilePath {
			ground = TileGrass
		}
		w.tiles[i] = ground
		w.relief[i] = w.relief[i].withVariant(VarRamp)
	case TileRiver:
		w.tiles[i] = TileFord
	case TileLava:
		w.tiles[i] = TileAsh
	}
}

// joinLand cuts ramps through the cliffs, lays fords over the rivers and
// crusts over lava until every island can be walked end to end, and then a
// few more, so that nobody walks far along a cliff or a river to cross it.
func (w *World) joinLand(seed uint32) {
	W, H := w.width, w.height
	rng := rand.New(rand.NewPCG(uint64(seed), 0x1a4d))
	comp := make([]int32, len(w.tiles))
	label := func() int { return w.labelWalkable(comp) }
	n := label()
	parent := make([]int32, n)
	for k := range parent {
		parent[k] = int32(k)
	}
	var find func(int32) int32
	find = func(a int32) int32 {
		for parent[a] != a {
			parent[a] = parent[parent[a]]
			a = parent[a]
		}
		return a
	}

	// A crossing: from walkable tile a, over 1 to 3 bridgeable tiles in a
	// straight line, to walkable tile b.
	type crossing struct {
		a, b  int
		tiles [3]int
		n     int
		dx    int // 1: across (east-west), 0: down (north-south)
	}
	var cands []crossing
	for i := range w.tiles {
		if !w.bridgeable(i) {
			continue
		}
		x, y := i%W, i/W
		for _, d := range [2][2]int{{1, 0}, {0, 1}} {
			// Only from the first tile of the run, looking back.
			px, py := x-d[0], y-d[1]
			if px < 0 || py < 0 || !w.tiles[py*W+px].Walkable() {
				continue
			}
			c := crossing{a: py*W + px, dx: d[0]}
			cx, cy := x, y
			for c.n < 3 && cx < W && cy < H && w.bridgeable(cy*W+cx) {
				c.tiles[c.n] = cy*W + cx
				c.n++
				cx, cy = cx+d[0], cy+d[1]
			}
			if cx >= W || cy >= H || !w.tiles[cy*W+cx].Walkable() {
				continue
			}
			c.b = cy*W + cx
			if comp[c.a] != comp[c.b] {
				cands = append(cands, c)
			}
		}
	}
	rng.Shuffle(len(cands), func(a, b int) { cands[a], cands[b] = cands[b], cands[a] })
	// Shorter crossings first: a ramp through one tile of cliff before a
	// ford over three of river.
	slices.SortStableFunc(cands, func(a, b crossing) int { return a.n - b.n })
	cross := func(c crossing) {
		hi := w.tiles[c.a]
		if w.relief[c.b].Level() > w.relief[c.a].Level() {
			hi = w.tiles[c.b]
		}
		for _, i := range c.tiles[:c.n] {
			orig := w.tiles[i]
			w.bridge(i, hi)
			// Two tiles wide at least, three if there is room: beasts use
			// them too.
			x, y := i%W, i/W
			for _, s := range []int{-1, 1} {
				nx, ny := x+s*(1-c.dx), y+s*c.dx
				if w.inBounds(nx, ny) && w.tiles[ny*W+nx] == orig {
					w.bridge(ny*W+nx, hi)
				}
			}
		}
	}
	for _, c := range cands {
		a, b := find(comp[c.a]), find(comp[c.b])
		if a == b {
			continue
		}
		cross(c)
		parent[a] = b
	}

	// A few more, where a cliff or a river runs long without a way across.
	kept := func(i int, r int, ok func(j int) bool) bool {
		x, y := i%W, i/W
		for yy := y - r; yy <= y+r; yy++ {
			for xx := x - r; xx <= x+r; xx++ {
				if w.inBounds(xx, yy) && ok(yy*W+xx) {
					return true
				}
			}
		}
		return false
	}
	for _, c := range cands {
		if c.n > 2 {
			continue
		}
		t := w.tiles[c.tiles[0]]
		switch {
		case t == TileCliff && rng.IntN(rampEvery) == 0 &&
			!kept(c.tiles[0], rampEvery/3, func(j int) bool { return w.relief[j].Variant() == VarRamp && w.tiles[j] != TileCliff }):
			cross(c)
		case t == TileRiver && rng.IntN(fordEvery) == 0 &&
			!kept(c.tiles[0], fordEvery/3, func(j int) bool { return w.tiles[j] == TileFord }):
			cross(c)
		}
	}
	w.mendPockets(comp, label)
}

// mendPockets joins what no straight crossing joined by a winding one: the
// shortest way through cliff, river or lava to other open ground. What is
// still cut off (a pocket walled in by a waterfall, a lake and a cliff) is
// filled in with rock, if it is small. label labels comp afresh.
func (w *World) mendPockets(comp []int32, label func() int) {
	for range 8 {
		if !w.windingCrossings(comp, label()) {
			break
		}
	}
	n := label()
	size := make([]int, n)
	for _, c := range comp {
		if c >= 0 {
			size[c]++
		}
	}
	for _, i := range w.strandedPockets(comp, size) {
		w.tiles[i] = TileCliff
		w.relief[i] = w.relief[i].withVariant(VarPlain)
	}
}

// strandedPockets returns the walkable tiles of every small pocket that is not
// the largest walkable part of its island: those no crossing could reach.
func (w *World) strandedPockets(comp []int32, size []int) []int {
	var out []int
	for _, isl := range w.landMasses() {
		best := int32(-1)
		for _, i := range isl {
			if c := comp[i]; c >= 0 && (best < 0 || size[c] > size[best]) {
				best = c
			}
		}
		for _, i := range isl {
			if c := comp[i]; c >= 0 && c != best && size[c] < 60 {
				out = append(out, i)
			}
		}
	}
	return out
}

// wetAndSoft marks the ground beside fresh water as wet, and the borders
// between biomes that the client blends wide as soft.
func (w *World) wetAndSoft(seed uint32) {
	W, H := w.width, w.height
	fresh := func(t Tile, r Relief) bool {
		return t == TileRiver || t == TileFord || t == TileWaterfall || t == TileWater && r.Variant() == VarLake
	}
	for i, t := range w.tiles {
		if !t.Walkable() {
			continue
		}
		x, y := i%W, i/W
		wet, soft := false, false
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if (dx == 0 && dy == 0) || nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				j := ny*W + nx
				u := w.tiles[j]
				if fresh(u, w.relief[j]) {
					wet = true
				}
				if (dx == 0 || dy == 0) && u.Walkable() && u != t && u != TilePath && u != TileFord && t != TileFord {
					soft = true
				}
			}
		}
		r := w.relief[i]
		if wet {
			r |= reliefWet
		}
		if soft && fbm(float64(x)/14+3, float64(y)/14+8, seed+91, 2) > 0.36 {
			r |= reliefSoft
		}
		w.relief[i] = r
	}
}

// windingCrossings joins each part of an island that is not its largest to
// other open ground by the shortest way through what can be bridged, and
// reports whether it joined any.
func (w *World) windingCrossings(comp []int32, n int) bool {
	W, H := w.width, w.height
	size := make([]int, n)
	for _, c := range comp {
		if c >= 0 {
			size[c]++
		}
	}
	joined := false
	from := make(map[int]int)
	for _, isl := range w.landMasses() {
		best := int32(-1)
		var parts []int32
		for _, i := range isl {
			c := comp[i]
			if c < 0 {
				continue
			}
			if !slices.Contains(parts, c) {
				parts = append(parts, c)
			}
			if best < 0 || size[c] > size[best] {
				best = c
			}
		}
		for _, p := range parts {
			if p == best {
				continue
			}
			// A breadth-first search out of part p through bridgeable tiles.
			clear(from)
			var q []int
			for _, i := range isl {
				if comp[i] == p {
					from[i] = -1
					q = append(q, i)
				}
			}
			end := -1
			for k := 0; k < len(q) && end < 0; k++ {
				i := q[k]
				x, y := i%W, i/W
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if nx < 0 || ny < 0 || nx >= W || ny >= H {
						continue
					}
					j := ny*W + nx
					if _, ok := from[j]; ok {
						continue
					}
					if c := comp[j]; c >= 0 && c != p {
						from[j], end = i, j
						break
					}
					if w.bridgeable(j) {
						from[j] = i
						q = append(q, j)
					}
				}
			}
			if end < 0 {
				continue
			}
			hi := w.tiles[end]
			for i := from[end]; i >= 0 && comp[i] != p; i = from[i] {
				if w.relief[i].Level() > w.relief[end].Level() {
					hi = TileGrass
				}
				w.bridge(i, hi)
			}
			joined = true
		}
	}
	return joined
}

// cliffCoast reports whether the shore of island s at (x, y) is one of its
// cliff coasts: two arcs of its shore, opposite each other, together about
// share of it. The rest is beach, so every island can be landed on.
func (s *isle) cliffCoast(x, y, share float64) bool {
	a := math.Atan2(y-s.y, x-s.x)
	c1 := float64(s.seed%6283) / 1000
	c2 := c1 + math.Pi*(0.7+0.6*float64(s.seed>>16%1000)/1000)
	half := share * math.Pi / 2 * (1 + 0.3*(fbm(math.Cos(a)*2+5, math.Sin(a)*2, s.seed+0xc1f, 2)-0.5))
	for _, c := range []float64{c1, c2} {
		if d := math.Abs(math.Remainder(a-c, 2*math.Pi)); d < half {
			return true
		}
	}
	return false
}

// landings makes sure every island can be landed on from the sea: one
// walled in by cliffs all round gets a ramp down to the water, and a rock
// too small for one is all cliff.
func (w *World) landings() {
	W, H := w.width, w.height
	for _, isl := range w.landMasses() {
		landing, open := false, 0
		ramp := -1
		for _, i := range isl {
			x, y := i%W, i/W
			t := w.tiles[i]
			if t.Walkable() {
				open++
			}
			for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H || !w.seaAt(ny*W+nx) {
					continue
				}
				if t.Walkable() {
					landing = true
				} else if t == TileCliff && ramp < 0 {
					// A cliff over the sea with open ground just inland.
					if bx, by := x-d[0], y-d[1]; w.inBounds(bx, by) && w.tiles[by*W+bx].Walkable() {
						ramp = i
					}
				}
			}
		}
		switch {
		case landing || open == 0:
		case ramp >= 0:
			w.bridge(ramp, TileSand)
		case open < 60 || !w.stairsFromSea(isl):
			for _, i := range isl {
				if w.tiles[i].Walkable() {
					w.tiles[i] = TileCliff
				}
			}
		}
	}
}

// stairsFromSea cuts the shortest way through island isl's cliffs from the
// sea to its open ground, and reports whether there was one.
func (w *World) stairsFromSea(isl []int) bool {
	W, H := w.width, w.height
	from := make(map[int]int)
	var q []int
	for _, i := range isl {
		if w.tiles[i] != TileCliff {
			continue
		}
		x, y := i%W, i/W
		for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
			if nx, ny := x+d[0], y+d[1]; nx >= 0 && ny >= 0 && nx < W && ny < H && w.seaAt(ny*W+nx) {
				from[i] = -1
				q = append(q, i)
				break
			}
		}
	}
	for k := 0; k < len(q); k++ {
		i := q[k]
		x, y := i%W, i/W
		for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			j := ny*W + nx
			if _, ok := from[j]; ok {
				continue
			}
			if w.tiles[j].Walkable() {
				for c := i; c >= 0; c = from[c] {
					w.bridge(c, TileSand)
				}
				return true
			}
			if w.tiles[j] == TileCliff {
				from[j] = i
				q = append(q, j)
			}
		}
	}
	return false
}

// labelWalkable labels every walkable tile by the part of the land it can
// be walked to from (4-connected), -1 elsewhere, and returns how many parts
// there are.
func (w *World) labelWalkable(comp []int32) int {
	W, H := w.width, w.height
	for i := range comp {
		comp[i] = -1
	}
	n := 0
	var q []int
	for start, t := range w.tiles {
		if comp[start] >= 0 || !t.Walkable() {
			continue
		}
		comp[start] = int32(n)
		q = append(q[:0], start)
		for k := 0; k < len(q); k++ {
			x, y := q[k]%W, q[k]/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; comp[j] < 0 && w.tiles[j].Walkable() {
					comp[j] = int32(n)
					q = append(q, j)
				}
			}
		}
		n++
	}
	return n
}

// mend joins or fills whatever part of an island was cut off since
// joinLand: see mendPockets.
func (w *World) mend() {
	comp := make([]int32, len(w.tiles))
	w.mendPockets(comp, func() int { return w.labelWalkable(comp) })
}
