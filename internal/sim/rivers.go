package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// Rivers. On the larger islands rivers rise in the high ground and run
// down to the sea, the way water would: every tile of land knows its way
// down to the shore (a priority flood from the coast up), and a river
// follows it from its spring. Rivers grow wider as they go and as others
// join them; where one drops over a cliff it is a waterfall; here and there
// it pools into a lake; the delta island's river splits into channels
// before the sea. Fords are laid over them later, with the ramps (see
// joinLand), and the ground beside them is wet (see wetAndSoft).

// River tuning.
const (
	FordSpeed     = 0.7 // wading across a ford
	riverMinLen   = 14  // a shorter run from spring to sea is no river
	riverWide2    = 45  // tiles from its spring a river is two tiles wide
	riverWide3    = 120 // and three
	riverSpacing  = 0.3 // of an island's radius: between two springs at least
	riverSpringH  = 0.6 // the least height (0 to 1) a spring rises at
	lakeChance    = 0.5
	deltaChannels = 3
)

// springs is how many rivers island s has: since StyleWilds, more on a
// bigger island, one more for every wildsRiverR tiles of radius past the
// first.
func springs(s *isle) int {
	n := baseSprings(s)
	if s.lay != nil && s.lay.wilds && n > 0 && s.role != roleMedium {
		n += int(max(0, s.r-wildsRiverR) / wildsRiverR)
		if s.role == roleHome {
			n = max(n, 2)
		}
	}
	return n
}

// wildsRiverR is how many tiles of an island's radius bring it one more
// river, since StyleWilds.
const wildsRiverR = 60

// baseSprings is how many rivers island s has, whatever its size.
func baseSprings(s *isle) int {
	switch {
	case s.role == roleIslet || s.role == roleAtoll:
		return 0
	case s.role == roleHome:
		return 1
	case s.role == roleMedium:
		if s.r < 18 || s.kind == IslandDesert {
			return 0
		}
		return 1
	}
	switch s.kind {
	case IslandAsh:
		return 0
	case IslandDesert:
		return 1
	case IslandMarsh:
		return 1
	case IslandMonkey, IslandMeadow:
		return 2
	}
	return 3
}

// flowHeap is a min-heap of tiles by height, ties by index, for the flood.
type flowHeap struct {
	h []float32
	i []int32
}

func (q *flowHeap) less(a, b int) bool {
	return q.h[a] < q.h[b] || q.h[a] == q.h[b] && q.i[a] < q.i[b]
}

func (q *flowHeap) push(h float32, i int32) {
	q.h, q.i = append(q.h, h), append(q.i, i)
	for k := len(q.h) - 1; k > 0; {
		p := (k - 1) / 2
		if !q.less(k, p) {
			break
		}
		q.h[k], q.h[p], q.i[k], q.i[p] = q.h[p], q.h[k], q.i[p], q.i[k]
		k = p
	}
}

func (q *flowHeap) pop() (float32, int32) {
	h, i := q.h[0], q.i[0]
	n := len(q.h) - 1
	q.h[0], q.i[0] = q.h[n], q.i[n]
	q.h, q.i = q.h[:n], q.i[:n]
	for k := 0; ; {
		l, r, m := 2*k+1, 2*k+2, k
		if l < n && q.less(l, m) {
			m = l
		}
		if r < n && q.less(r, m) {
			m = r
		}
		if m == k {
			break
		}
		q.h[k], q.h[m], q.i[k], q.i[m] = q.h[m], q.h[k], q.i[m], q.i[k]
		k = m
	}
	return h, i
}

// downhill works out, for every tile of land, the next tile on its way down
// to the sea (-1 for the shore itself, and the sea).
func (w *World) downhill(im *isleMap, seed uint32) []int32 {
	W, H := w.width, w.height
	next := make([]int32, len(w.tiles))
	done := make([]bool, len(w.tiles))
	q := &flowHeap{}
	height := func(i int) float32 {
		x, y := float64(i%W)+0.5, float64(i/W)+0.5
		return im.h[i] + float32(0.06*(fbm(x/5, y/5, seed+0x41, 2)-0.5))
	}
	for i := range w.tiles {
		next[i] = -1
		if w.seaAt(i) {
			done[i] = true
			continue
		}
		x, y := i%W, i/W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if nx, ny := x+d[0], y+d[1]; nx >= 0 && ny >= 0 && nx < W && ny < H && w.seaAt(ny*W+nx) {
				done[i] = true
				q.push(height(i), int32(i))
				break
			}
		}
	}
	for len(q.h) > 0 {
		lvl, ii := q.pop()
		i := int(ii)
		x, y := i%W, i/W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			j := ny*W + nx
			if done[j] {
				continue
			}
			done[j] = true
			next[j] = int32(i)
			q.push(max(height(j), lvl), int32(j))
		}
	}
	return next
}

// runRivers lays the rivers, their lakes and waterfalls, and the delta.
func (w *World) runRivers(im *isleMap, toSea []uint8, seed uint32) {
	W := w.width
	next := w.downhill(im, seed)
	rng := rand.New(rand.NewPCG(uint64(seed), 0x41e5))
	// Springs: the high ground of each island, well apart.
	cands := make([][]int32, len(im.isles))
	for i, t := range w.tiles {
		if o := im.owner[i]; o > 0 && t.Walkable() && im.h[i] >= riverSpringH && toSea[i] >= 10 {
			cands[o-1] = append(cands[o-1], int32(i))
		}
	}
	flow := make([]uint8, len(w.tiles))  // rivers through each tile
	run := make([]uint16, len(w.tiles))  // tiles from the spring, the most of any river through it
	dir := make([]FlowDir, len(w.tiles)) // the way each river tile flows
	var rivers [][]int32
	for k := range im.isles {
		s := &im.isles[k]
		want := springs(s)
		if want == 0 || len(cands[k]) == 0 {
			continue
		}
		c := cands[k]
		rng.Shuffle(len(c), func(a, b int) { c[a], c[b] = c[b], c[a] })
		// The highest first, among a shuffled few.
		slices.SortStableFunc(c, func(a, b int32) int {
			switch ha, hb := im.h[a] > 0.85, im.h[b] > 0.85; {
			case ha && !hb:
				return -1
			case hb && !ha:
				return 1
			}
			return 0
		})
		var got []int32
		for _, src := range c {
			if len(got) == want {
				break
			}
			sx, sy := float64(src%int32(W)), float64(src/int32(W))
			far := true
			for _, g := range got {
				if math.Hypot(sx-float64(g%int32(W)), sy-float64(g/int32(W))) < s.r*riverSpacing {
					far = false
					break
				}
			}
			if !far {
				continue
			}
			var path []int32
			for i := src; i >= 0; i = next[i] {
				path = append(path, i)
			}
			if len(path) < riverMinLen {
				continue
			}
			got = append(got, src)
			rivers = append(rivers, path)
		}
	}
	for _, path := range rivers {
		for k, i := range path {
			flow[i] = min(flow[i]+1, 250)
			run[i] = max(run[i], uint16(min(k, 60000)))
			if k+1 < len(path) {
				j := path[k+1]
				dir[i] = flowTo(int(j%int32(W))-int(i%int32(W)), int(j/int32(W))-int(i/int32(W)))
			} else {
				dir[i] = w.seaward(int(i))
			}
		}
	}
	// The delta: before the sea the delta island's river splits.
	for _, path := range rivers {
		if s := im.isleAt(int(path[0])); s != nil && s.kind == IslandMarsh && s.role == roleLarge {
			w.splitDelta(im, path, toSea, flow, run, dir, rng)
		}
	}
	// Widen, by how far each has come and how many have joined.
	type wide struct {
		i int
		d FlowDir
	}
	var more []wide
	for i := range w.tiles {
		if flow[i] == 0 {
			continue
		}
		width := 1
		if run[i] >= riverWide2 || flow[i] >= 2 {
			width = 2
		}
		if run[i] >= riverWide3 || flow[i] >= 3 {
			width = 3
		}
		dx, dy := dir[i].Delta()
		x, y := i%W, i/W
		for k, side := range []int{1, -1} {
			if k+1 >= width {
				break
			}
			nx, ny := x-dy*side, y+dx*side
			if w.inBounds(nx, ny) && flow[ny*W+nx] == 0 {
				more = append(more, wide{ny*W + nx, dir[i]})
			}
		}
	}
	for _, m := range more {
		if j := m.i; !w.seaAt(j) && !w.lakeAt(j) && flow[j] == 0 {
			flow[j], dir[j] = 1, m.d
		}
	}
	// Lakes, where a river crosses flat ground.
	for _, path := range rivers {
		if len(path) < 50 || rng.Float64() >= lakeChance {
			continue
		}
		c := int(path[len(path)*2/5+rng.IntN(len(path)/5)])
		if toSea[c] < 12 {
			continue
		}
		w.poolLake(c, 3+rng.Float64()*4*min(2, float64(w.width)/1024), seed)
	}
	// Lay the water: over a cliff, a waterfall where the river drops.
	for i := range w.tiles {
		if flow[i] == 0 || w.lakeAt(i) || w.seaAt(i) {
			continue
		}
		t := TileRiver
		if w.tiles[i] == TileCliff {
			dx, dy := dir[i].Delta()
			if j := (i/W+dy)*W + i%W + dx; w.inBounds(i%W+dx, i/W+dy) && w.relief[j].Level() < w.relief[i].Level() {
				t = TileWaterfall
			}
		}
		w.tiles[i] = t
		w.relief[i] = w.relief[i].withVariant(GroundVariant(dir[i]))
	}
}

// seaward is the way from tile i to the sea beside it, if any.
func (w *World) seaward(i int) FlowDir {
	W := w.width
	x, y := i%W, i/W
	for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
		if nx, ny := x+d[0], y+d[1]; w.inBounds(nx, ny) && w.seaAt(ny*W+nx) {
			return flowTo(d[0], d[1])
		}
	}
	return FlowSouth
}

// poolLake floods the land round tile c on its level into a lake.
func (w *World) poolLake(c int, r float64, seed uint32) {
	W := w.width
	cx, cy := c%W, c/W
	lvl := w.relief[c].Level()
	R := int(math.Ceil(r * 1.4))
	for y := cy - R; y <= cy+R; y++ {
		for x := cx - R; x <= cx+R; x++ {
			if !w.inBounds(x, y) {
				continue
			}
			i := y*W + x
			d := math.Hypot(float64(x-cx), float64(y-cy)) / r
			if d+0.5*(fbm(float64(x)/4, float64(y)/4, seed+0x1a3, 2)-0.5) > 1 {
				continue
			}
			if t := w.tiles[i]; t.Sea() || t == TileCliff || w.relief[i].Level() != lvl {
				continue
			}
			w.tiles[i] = TileWater
			w.relief[i] = w.relief[i].withVariant(VarLake)
		}
	}
}

// splitDelta branches the last stretch of a river into channels running
// out to the sea apart from each other, the mud between them.
func (w *World) splitDelta(im *isleMap, path []int32, toSea []uint8, flow []uint8, run []uint16, dir []FlowDir, rng *rand.Rand) {
	W := w.width
	// Branch where the river comes within reach of the sea.
	b := -1
	for k := len(path) - 1; k >= 0; k-- {
		if toSea[path[k]] >= 16 {
			b = int(path[k])
			break
		}
	}
	if b < 0 {
		return
	}
	bx, by := b%W, b/W
	mouth := int(path[len(path)-1])
	mx, my := mouth%W, mouth/W
	owner := im.owner[b]
	// Coast tiles of the same island, apart from the mouth and each other.
	var coast []int
	for y := by - 60; y <= by+60; y++ {
		for x := bx - 60; x <= bx+60; x++ {
			if !w.inBounds(x, y) {
				continue
			}
			i := y*W + x
			if im.owner[i] == owner && toSea[i] == 1 && w.tiles[i].Walkable() {
				d := math.Hypot(float64(x-bx), float64(y-by))
				if d > 16 && d < 55 && math.Hypot(float64(x-mx), float64(y-my)) > 14 {
					coast = append(coast, i)
				}
			}
		}
	}
	rng.Shuffle(len(coast), func(a, c int) { coast[a], coast[c] = coast[c], coast[a] })
	var ends []int
	for _, c := range coast {
		if len(ends) == deltaChannels-1 {
			break
		}
		ok := true
		for _, e := range ends {
			if math.Hypot(float64(c%W-e%W), float64(c/W-e/W)) < 16 {
				ok = false
			}
		}
		if ok {
			ends = append(ends, c)
		}
	}
channels:
	for _, e := range ends {
		ex, ey := e%W, e/W
		l := math.Hypot(float64(ex-bx), float64(ey-by))
		bend := (rng.Float64() - 0.5) * l * 0.35
		px, py := bx, by
		for t := 0.0; t <= 1; t += 0.5 / l {
			ox := -float64(ey-by) / l * bend * math.Sin(t*math.Pi)
			oy := float64(ex-bx) / l * bend * math.Sin(t*math.Pi)
			x := int(math.Floor(float64(bx) + 0.5 + float64(ex-bx)*t + ox))
			y := int(math.Floor(float64(by) + 0.5 + float64(ey-by)*t + oy))
			for px != x || py != y { // a 4-connected channel
				nx, ny := px, py
				if px != x {
					nx += stepToward(x - px)
				} else {
					ny += stepToward(y - py)
				}
				if !w.inBounds(nx, ny) {
					return
				}
				i, j := py*W+px, ny*W+nx
				if w.seaAt(j) {
					dir[i] = flowTo(nx-px, ny-py)
					continue channels
				}
				dir[i] = flowTo(nx-px, ny-py)
				if flow[j] == 0 {
					flow[j], run[j], dir[j] = 1, run[i]+1, flowTo(nx-px, ny-py)
				}
				px, py = nx, ny
			}
		}
		if i := py*W + px; flow[i] > 0 {
			dir[i] = w.seaward(i)
		}
	}
	// Mud flats along the channels' lower reaches.
	for y := by - 60; y <= by+60; y++ {
		for x := bx - 60; x <= bx+60; x++ {
			if !w.inBounds(x, y) {
				continue
			}
			i := y*W + x
			if im.owner[i] == owner && w.tiles[i] == TileMarsh && toSea[i] <= 10 && hash2(int32(x), int32(y), 0xde17)%3 != 0 {
				w.relief[i] = w.relief[i].withVariant(VarMudflat)
			}
		}
	}
}
