package sim

import (
	"container/heap"
	"math"
	"math/rand/v2"
	"slices"
)

// Trails. In a tribal world beaten paths wind from where players start to
// the villages and the cave mouths, and from village to village, keeping
// to easy ground. Nothing grows on a path, and walking it is quicker.

// Path tuning.
const (
	PathSpeed    = 1.15 // walking on a trail
	pathMouths   = 3    // cave mouths joined to the spawn
	pathWideness = 0.45 // share of a trail's length that is two tiles wide
)

// pathCost is the cost of walking onto tile t when laying a trail; 0 means
// a trail cannot cross it.
func pathCost(t Tile) float64 {
	switch t {
	case TileGrass, TilePath:
		return 1
	case TileForest:
		return 1.7
	case TileSand:
		return 1.6
	case TileDesert, TileRocky:
		return 2.2
	case TileSnow:
		return 2.6
	case TileAsh:
		return 3
	case TileMarsh:
		return 5
	}
	return 0
}

// carvePaths lays the trails of a tribal world.
func (w *World) carvePaths(rng *rand.Rand, sx, sy int) {
	type pt struct{ x, y int }
	spawn := pt{sx, sy}
	var mouths []pt
	for i, t := range w.tiles {
		x, y := i%w.width, i/w.width
		if t == TileCave && w.caveOutside(x, y+1) {
			mouths = append(mouths, pt{x, y + 1})
		}
	}
	dist := func(a, b pt) float64 { return math.Hypot(float64(a.x-b.x), float64(a.y-b.y)) }
	nearest := func(from pt, list []pt, n int) []pt {
		s := slices.Clone(list)
		slices.SortFunc(s, func(a, b pt) int {
			da, db := dist(from, a), dist(from, b)
			switch {
			case da < db:
				return -1
			case da > db:
				return 1
			}
			return 0
		})
		return s[:min(n, len(s))]
	}
	var legs [][2]pt
	var villages []pt
	for _, v := range w.villages {
		villages = append(villages, pt{int(v.X), int(v.Y)})
	}
	for _, v := range villages {
		legs = append(legs, [2]pt{spawn, v})
		for _, m := range nearest(v, mouths, 1) {
			legs = append(legs, [2]pt{v, m})
		}
		if o := nearest(v, villages, 2); len(o) == 2 {
			legs = append(legs, [2]pt{v, o[1]})
		}
	}
	for _, m := range nearest(spawn, mouths, pathMouths) {
		legs = append(legs, [2]pt{spawn, m})
	}
	for _, l := range legs {
		route := w.findPath(l[0].x, l[0].y, l[1].x, l[1].y)
		for k, i := range route {
			if pathCost(w.tiles[i]) == 0 || max(abs(i%w.width-sx), abs(i/w.width-sy)) <= clearSpawnRadius {
				continue // the spawn keeps its own ground
			}
			w.tiles[i] = TilePath
			// Now and then the trail is two tiles wide.
			if k > 0 && rng.Float64() < pathWideness {
				x, y := i%w.width, i/w.width
				px, py := route[k-1]%w.width, route[k-1]/w.width
				nx, ny := x+(py-y), y+(x-px) // to one side of the way it runs
				if w.inBounds(nx, ny) && pathCost(w.tiles[ny*w.width+nx]) > 0 {
					w.tiles[ny*w.width+nx] = TilePath
				}
			}
		}
	}
}

// findPath returns the tiles of the easiest way from (x0, y0) to (x1, y1),
// or nil if there is none.
func (w *World) findPath(x0, y0, x1, y1 int) []int {
	if !w.inBounds(x0, y0) || !w.inBounds(x1, y1) {
		return nil
	}
	n := w.width * w.height
	cost := make([]float32, n)
	from := make([]int32, n)
	for i := range cost {
		cost[i] = float32(math.Inf(1))
		from[i] = -1
	}
	start, goal := y0*w.width+x0, y1*w.width+x1
	h := func(i int) float64 {
		return math.Hypot(float64(i%w.width-x1), float64(i/w.width-y1))
	}
	q := &pathQueue{}
	cost[start] = 0
	heap.Push(q, pathNode{i: start, f: h(start)})
	for q.Len() > 0 {
		cur := heap.Pop(q).(pathNode)
		if cur.i == goal {
			break
		}
		x, y := cur.i%w.width, cur.i/w.width
		for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.inBounds(nx, ny) {
				continue
			}
			j := ny*w.width + nx
			c := pathCost(w.tiles[j])
			if c == 0 && j != goal {
				continue
			}
			if c == 0 {
				c = 1
			}
			// A little noise makes the way wander instead of running straight.
			c *= 0.8 + 0.4*unit(hash2(int32(nx), int32(ny), 0x9a7))
			if d[0] != 0 && d[1] != 0 {
				c *= math.Sqrt2
			}
			if nc := cost[cur.i] + float32(c); nc < cost[j] {
				cost[j] = nc
				from[j] = int32(cur.i)
				heap.Push(q, pathNode{i: j, f: float64(nc) + h(j)})
			}
		}
	}
	if from[goal] < 0 && goal != start {
		return nil
	}
	var route []int
	for i := goal; i >= 0; i = int(from[i]) {
		route = append(route, i)
		if i == start {
			break
		}
	}
	slices.Reverse(route)
	return route
}

type pathNode struct {
	i int
	f float64
}

type pathQueue []pathNode

func (q pathQueue) Len() int           { return len(q) }
func (q pathQueue) Less(a, b int) bool { return q[a].f < q[b].f }
func (q pathQueue) Swap(a, b int)      { q[a], q[b] = q[b], q[a] }
func (q *pathQueue) Push(x any)        { *q = append(*q, x.(pathNode)) }
func (q *pathQueue) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}
