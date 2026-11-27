package sim

import (
	"math"
	"runtime"
	"slices"
	"sync"
)

// allSpawnSpots works out, for every kind of creature, the tiles where it
// may appear when a world is made (see spawnCreatures), all kinds at once
// on every core: a big world has millions of tiles to look at for each.
// Where one kind may appear does not depend on where another was put, so
// the lists are the same as if they were made one after another.
func (w *World) allSpawnSpots(sx, sy int) [NumCreatureKinds][]int32 {
	// What the checks fill in lazily is filled in first, so no two cores
	// fill it at once.
	w.isShallow(0, 0)
	w.pondWater(0, 0)
	var offshore []uint8 // tiles from land, for sea creatures; see coastDistance
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		if d := k.def(); d.deep && !(d.openOnly && w.style == StyleIsland) {
			offshore = w.coastDistance(250)
			break
		}
	}
	var fromHome []uint8 // since StyleWilds: tiles from the home island's land; see wilds.go
	if w.style >= StyleWilds {
		fromHome = w.homeSeaDistance(w.islandIndex())
	}
	var out [NumCreatureKinds][]int32
	kinds := make(chan CreatureKind)
	var wg sync.WaitGroup
	for range max(1, runtime.GOMAXPROCS(0)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range kinds {
				out[k] = w.spawnSpots(k, sx, sy, offshore, fromHome)
			}
		}()
	}
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		if d := k.def(); !(d.openOnly && w.style == StyleIsland) {
			kinds <- k
		}
	}
	close(kinds)
	wg.Wait()
	return out
}

// spawnSpots is the tiles where a creature of kind k may appear in a new
// world whose spawn is (sx, sy); fromHome is nil, or since StyleWilds how
// far each tile of the surface lies from the home island.
func (w *World) spawnSpots(k CreatureKind, sx, sy int, offshore, fromHome []uint8) []int32 {
	away := 0
	if fromHome != nil {
		away = wildsHomeAway[k]
	}
	d := k.def()
	var spots []int32
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			t := w.tiles[ty*w.width+tx]
			if !slices.Contains(d.spawnOn, t) || (d.water && !w.isShallow(tx, ty)) || (d.seaside && !w.seaside(tx, ty)) || (d.waterside && !w.waterside(tx, ty)) || (d.pondside && !w.pondside(tx, ty)) {
				continue
			}
			if d.deep {
				lo, hi := plesioNear, plesioFar
				if d.offshore[1] > 0 {
					lo, hi = d.offshore[0], d.offshore[1]
				}
				if o, ok := wildsOffshore[k]; ok && fromHome != nil {
					lo, hi = o[0], o[1]
				}
				if o := int(offshore[ty*w.width+tx]); o < lo || o > hi {
					continue
				}
			}
			if d.away > 0 && math.Hypot(float64(tx-sx), float64(ty-sy)) < d.away {
				continue
			}
			if away > 0 && ty < len(fromHome)/w.width && int(fromHome[ty*w.width+tx]) < away {
				continue // the home island's waters
			}
			if k == CreatureWorm && w.style >= StyleGround && !w.dampSoil(tx, ty) {
				continue // worms keep to damp soil: see groundcover.go
			}
			if !w.creatureFits(k, float64(tx)+0.5, float64(ty)+0.5) {
				continue
			}
			spots = append(spots, int32(ty*w.width+tx))
		}
	}
	return spots
}
