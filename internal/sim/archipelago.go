package sim

import (
	"math"
	"slices"
)

// The archipelago: worlds made since StyleSeas are a wide sea dotted with
// islands, one land to each island. The home island in the middle is gentle
// meadow; the further out an island lies, the harsher it is: woods and
// swamps, then snowy mountains and deserts, and at the edge of the world a
// volcano. Getting anywhere means building a boat; see boats.go.

// IslandKind is the land an island is made of.
type IslandKind uint8

const (
	IslandMeadow IslandKind = iota // grass and copses: the home island
	IslandForest                   // deep woods, with a mountain or two
	IslandMarsh                    // swamp and ponds
	IslandSnow                     // snowfields round high mountains
	IslandDesert                   // dunes and rock
	IslandAsh                      // ash and a volcano
	numIslandKinds
)

// Archipelago tuning.
const (
	seasGap          = 7    // tiles of open sea at least between two islands
	seasMinIsland    = 40   // smaller bits of land are islets: sand and grass only
	seasHomeRadius   = 0.11 // of the world's size: the home island's raised middle
	seasBeach        = 0.03 // of the land's height above the sea: the beaches
	seasMountainLine = 0.62 // share of an island's height above which it has mountains
)

// archipelagoTerrain fills w with islands and returns the spawn tile.
func (w *World) archipelagoTerrain(seed uint32) (sx, sy int) {
	W, H := w.width, w.height
	n := float64(max(W, H))
	cx, cy := float64(W)/2, float64(H)/2
	elev := w.islandField(seed)
	sea := 0.0
	for i, e := range elev {
		if e <= sea {
			w.tiles[i] = TileWater
		} else {
			w.tiles[i] = TileGrass
		}
	}
	_ = n

	// Each island, and how far out it lies.
	islands := w.landMasses()
	home := -1
	bestD := math.Inf(1)
	for k, isl := range islands {
		for _, i := range isl {
			if d := math.Hypot(float64(i%W)+0.5-cx, float64(i/W)+0.5-cy); d < bestD {
				bestD, home = d, k
			}
		}
	}
	kinds := make([]IslandKind, len(islands))
	far := make([]float64, len(islands))
	for k, isl := range islands {
		mx, my := 0.0, 0.0
		for _, i := range isl {
			mx += float64(i%W) + 0.5
			my += float64(i/W) + 0.5
		}
		mx, my = mx/float64(len(isl)), my/float64(len(isl))
		far[k] = math.Hypot((mx-cx)/(float64(W)/2), (my-cy)/(float64(H)/2))
		kinds[k] = islandKindAt(far[k], hash2(int32(mx), int32(my), seed+303), k == home)
	}
	w.everyIslandKind(islands, kinds, far, home)

	// Paint each island by its kind and its height.
	for k, isl := range islands {
		top := sea
		for _, i := range isl {
			top = math.Max(top, elev[i])
		}
		small := len(isl) < seasMinIsland
		for _, i := range isl {
			x, y := float64(i%W)+0.5, float64(i/W)+0.5
			h := clamp((elev[i]-sea)/math.Max(1e-9, top-sea), 0, 1)
			w.tiles[i] = w.islandTile(kinds[k], h, x, y, seed, small)
		}
	}

	w.islands = make([]Island, 0, len(islands))
	for k, isl := range islands {
		if len(isl) >= seasMinIsland {
			w.islands = append(w.islands, Island{Kind: kinds[k], Tiles: len(isl), Home: k == home})
		}
	}

	sx, sy = w.openSpawn()
	w.volcano(seed, sx, sy)
	return sx, sy
}

// Island is one island of an archipelago world.
type Island struct {
	Kind  IslandKind
	Tiles int
	Home  bool
}

// Islands returns the islands of an archipelago world, nil for others.
func (w *World) Islands() []Island { return slices.Clone(w.islands) }

// islandKindAt picks an island's land by how far out it lies (0 at the
// middle, 1 at the edge) and a random h.
func islandKindAt(far float64, h uint32, home bool) IslandKind {
	r := unit(h)
	switch {
	case home:
		return IslandMeadow
	case far < 0.3:
		return []IslandKind{IslandMeadow, IslandForest}[int(r*2)]
	case far < 0.52:
		return []IslandKind{IslandForest, IslandMarsh, IslandMeadow}[int(r*3)]
	case far < 0.72:
		return []IslandKind{IslandSnow, IslandDesert, IslandMarsh}[int(r*3)]
	default:
		return []IslandKind{IslandAsh, IslandSnow, IslandDesert}[int(r*3)]
	}
}

// everyIslandKind makes sure the world has at least one sizeable island of
// every kind: missing kinds take the biggest island in their ring.
func (w *World) everyIslandKind(islands [][]int, kinds []IslandKind, far []float64, home int) {
	want := []struct {
		k      IslandKind
		lo, hi float64
	}{{IslandForest, 0.15, 0.6}, {IslandMarsh, 0.3, 0.75}, {IslandSnow, 0.5, 1.1}, {IslandDesert, 0.5, 1.1}, {IslandAsh, 0.6, 1.1}}
	have := func(k IslandKind) bool {
		for i, kk := range kinds {
			if kk == k && len(islands[i]) >= 4*seasMinIsland {
				return true
			}
		}
		return false
	}
	taken := make([]bool, len(islands))
	for _, wnt := range want {
		if have(wnt.k) {
			continue
		}
		best := -1
		for i, isl := range islands {
			if i == home || taken[i] || far[i] < wnt.lo || far[i] > wnt.hi {
				continue
			}
			if best < 0 || len(isl) > len(islands[best]) {
				best = i
			}
		}
		if best >= 0 {
			kinds[best], taken[best] = wnt.k, true
		}
	}
}

// islandTile is the ground of a tile at height h (0 at the shore, 1 at the
// island's top) on an island of kind k.
func (w *World) islandTile(k IslandKind, h, x, y float64, seed uint32, small bool) Tile {
	forest := fbm(x/18+57, y/18+213, seed+311, 4)
	if h < seasBeach || small && h < 0.4 {
		if k == IslandSnow {
			return TileSnow
		}
		return TileSand
	}
	if small {
		return TileGrass
	}
	// No mountains: the high ground is bare rock, and the caves are
	// underground (see underground.go).
	mountain := func(float64) Tile { return TileRocky }
	switch k {
	case IslandMeadow:
		if h > 0.88 && fbm(x/10, y/10, seed+312, 2) > 0.62 {
			return TileRocky
		}
		if forest > 0.55 {
			return TileForest
		}
		return TileGrass
	case IslandForest:
		if h > seasMountainLine+0.08 {
			return mountain(seasMountainLine + 0.16)
		}
		if forest > 0.38 {
			return TileForest
		}
		return TileGrass
	case IslandMarsh:
		if fbm(x/7+13, y/7+57, seed+51, 2) > openPondLevel && h > 0.08 {
			return TileWater // a pond
		}
		if forest > 0.62 {
			return TileForest
		}
		return TileMarsh
	case IslandSnow:
		if h > seasMountainLine-0.1 {
			return mountain(seasMountainLine + 0.02)
		}
		return TileSnow
	case IslandDesert:
		if h > 0.8 {
			return TileRocky
		}
		return TileDesert
	case IslandAsh:
		if h > 0.85 {
			return TileRocky
		}
		return TileAsh
	}
	return TileGrass
}

// landMasses returns the tiles of every separate piece of land.
func (w *World) landMasses() [][]int {
	W := w.width
	seen := make([]bool, len(w.tiles))
	var out [][]int
	for start, t := range w.tiles {
		if t == TileWater || seen[start] {
			continue
		}
		comp := []int{start}
		seen[start] = true
		for k := 0; k < len(comp); k++ {
			x, y := comp[k]%W, comp[k]/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if !w.inBounds(nx, ny) {
					continue
				}
				if j := ny*W + nx; !seen[j] && w.tiles[j] != TileWater {
					seen[j] = true
					comp = append(comp, j)
				}
			}
		}
		out = append(out, comp)
	}
	return out
}

// islandField is the archipelago's height: above 0 is land. Islands are
// placed one by one where they leave open sea between them and the rest;
// each is a rounded mound with a ragged, noisy coast and bays.
func (w *World) islandField(seed uint32) []float64 {
	W, H := w.width, w.height
	n := float64(max(W, H))
	cx, cy := float64(W)/2, float64(H)/2
	rng := newIslandRand(seed)
	type lobe struct{ x, y, r float64 }
	type isl struct {
		x, y, r float64 // the island's centre and the radius that holds it all
		lobes   []lobe
	}
	mk := func(x, y, r float64) isl {
		il := isl{x: x, y: y, r: r, lobes: []lobe{{x, y, r * 0.75}}}
		for range 2 + int(rng.next()*3) {
			a, d := rng.next()*2*math.Pi, r*(0.35+0.35*rng.next())
			il.lobes = append(il.lobes, lobe{x + math.Cos(a)*d, y + math.Sin(a)*d*0.85, r * (0.35 + 0.3*rng.next())})
		}
		return il
	}
	isles := []isl{mk(cx, cy, max(18, n*seasHomeRadius*1.3))} // the home island
	for tries := 0; tries < 6000; tries++ {
		r := max(10, n*(0.04+0.09*rng.next()*rng.next()+0.025*rng.next()))
		a := rng.next() * 2 * math.Pi
		d := (0.2 + 0.8*math.Sqrt(rng.next())) * (n/2 - r - 4)
		x, y := cx+math.Cos(a)*d, cy+math.Sin(a)*d
		ok := true
		for _, o := range isles {
			if math.Hypot(o.x-x, o.y-y) < o.r*1.05+r*1.05+seasGap {
				ok = false
				break
			}
		}
		if ok {
			isles = append(isles, mk(x, y, r))
		}
	}
	elev := make([]float64, W*H)
	for i := range elev {
		elev[i] = -1
	}
	for k, il := range isles {
		reach := il.r * 1.1
		for ty := max(1, int(il.y-reach)); ty <= min(H-2, int(il.y+reach)); ty++ {
			for tx := max(1, int(il.x-reach)); tx <= min(W-2, int(il.x+reach)); tx++ {
				x, y := float64(tx)+0.5, float64(ty)+0.5
				// A coast pushed in and out by noise, with bays where it dips.
				warp := 0.65 + 0.7*fbm(x/(il.r*0.55)+float64(k)*7.1, y/(il.r*0.55), seed+321, 3)
				v := -1.0
				for _, lb := range il.lobes {
					v = math.Max(v, 1-math.Hypot(x-lb.x, y-lb.y)/(lb.r*warp))
				}
				v += 0.3 * (fbm(x/8+float64(k), y/8, seed+322, 2) - 0.5)
				// Never closer to the next island than the gap.
				v = math.Min(v, (il.r*1.05-math.Hypot(x-il.x, y-il.y))/il.r)
				if i := ty*W + tx; v > elev[i] {
					elev[i] = v
				}
			}
		}
	}
	return elev
}

// islandRand is a small seeded random source for laying out islands.
type islandRand struct{ s uint32 }

func newIslandRand(seed uint32) *islandRand { return &islandRand{seed*2654435761 + 1} }

func (r *islandRand) next() float64 {
	r.s = hash2(int32(r.s), int32(r.s>>16), 0x15a7d)
	return unit(r.s)
}
