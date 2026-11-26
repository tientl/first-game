package main

import (
	"math"
	"strings"

	"survivalgame/internal/sim"
)

// Spots in a world of the designed archipelago (sim.StyleIsles) worth a
// look, for -at:
//
//   - waterfall: below a river dropping over a cliff;
//   - ramp: below a ramp cut through a cliff between two levels;
//   - lagoon: on a spit of sand, a lagoon behind it and reefs out to sea;
//   - beach: on a wide beach, the sea in view (since sim.StyleBeaches).
//
// Each is the one nearest the spawn that shows the most of what it is.

// islesSpot finds the spot of kind what near (sx, sy), where the player
// stands to see it.
func islesSpot(w *sim.World, what string, sx, sy float64) (x, y float64, ok bool) {
	W, _ := w.Size()
	H := w.SurfaceHeight()
	tiles := w.Tiles()
	rel := w.Relief()
	if rel == nil {
		return 0, 0, false
	}
	at := func(tx, ty int) (sim.Tile, sim.Relief) {
		if tx < 0 || ty < 0 || tx >= W || ty >= H {
			return sim.TileWater, 0
		}
		return tiles[ty*W+tx], rel[ty*W+tx]
	}
	count := func(tx, ty, rx, ry int, f func(sim.Tile, sim.Relief) bool) int {
		n := 0
		for y := ty - ry; y <= ty+ry; y++ {
			for x := tx - rx; x <= tx+rx; x++ {
				if t, r := at(x, y); f(t, r) {
					n++
				}
			}
		}
		return n
	}
	best := math.Inf(-1)
	for ty := 12; ty < H-12; ty++ {
		for tx := 16; tx < W-16; tx++ {
			t, r := at(tx, ty)
			score, px, py := 0.0, tx, ty
			switch what {
			case "waterfall":
				// A fall with river above and below, open ground below it.
				if t != sim.TileWaterfall {
					continue
				}
				up, _ := at(tx, ty-1)
				if up != sim.TileRiver && up != sim.TileWaterfall {
					continue
				}
				py = ty + 4
				if g, _ := at(px, py); !g.Walkable() {
					px++
					if g, _ := at(px, py); !g.Walkable() {
						continue
					}
				}
				score = float64(count(tx, ty, 10, 6, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileRiver })) +
					3*float64(count(tx, ty, 10, 6, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileWaterfall })) +
					float64(count(tx, ty, 12, 7, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileCliff })) -
					0.6*float64(count(tx, ty, 12, 8, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileForest })) // in the open, not hidden by trees
			case "ramp":
				// A ramp with cliff either side of it and below it lower ground.
				if !t.Walkable() || r.Variant() != sim.VarRamp {
					continue
				}
				if l, _ := at(tx-2, ty); l != sim.TileCliff {
					continue
				}
				if rr, _ := at(tx+2, ty); rr != sim.TileCliff {
					continue
				}
				py = ty + 4
				if g, _ := at(px, py); !g.Walkable() {
					continue
				}
				score = float64(count(tx, ty, 12, 6, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileCliff }))
			case "lagoon":
				// Sand on a spit: a lagoon on one side, reefs or open sea beyond.
				if t != sim.TileSand {
					continue
				}
				lagoon := count(tx, ty, 12, 7, func(t sim.Tile, r sim.Relief) bool { return t == sim.TileShallow && r.Variant() == sim.VarLagoon })
				reef := count(tx, ty, 14, 8, func(t sim.Tile, r sim.Relief) bool { return t == sim.TileShallow && r.Variant() == sim.VarReef })
				shallow := count(tx, ty, 12, 7, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileShallow })
				if lagoon < 12 {
					continue
				}
				score = float64(lagoon) + 1.5*float64(reef) + 0.3*float64(shallow)
			case "beach":
				// Sand underfoot, a broad beach round, the sea a few tiles off.
				if t != sim.TileSand {
					continue
				}
				sea := count(tx, ty-6, 16, 3, func(t sim.Tile, _ sim.Relief) bool { return t.Sea() })
				if sea < 20 {
					continue
				}
				// Land behind it, not an islet all of sand; near the spawn.
				green := count(tx, ty+6, 16, 4, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileGrass || t == sim.TileForest })
				if green < 30 {
					continue
				}
				sand := count(tx, ty, 16, 8, func(t sim.Tile, _ sim.Relief) bool { return t == sim.TileSand })
				score = float64(min(sand, 160)) + 0.2*float64(sea) - math.Hypot(float64(tx)-sx, float64(ty)-sy)/4
			default:
				var ok bool
				if score, px, py, ok = moreSpots(what, tx, ty, t, r, at, count); !ok {
					if score == math.Inf(-1) {
						return 0, 0, false
					}
					continue
				}
			}
			score -= math.Hypot(float64(tx)-sx, float64(ty)-sy) / 40 // the nearer the better, other things equal
			if score > best {
				best, x, y, ok = score, float64(px)+0.5, float64(py)+0.5, true
			}
		}
	}
	return x, y, ok
}

// revealAll is an explored set with every tile of a w by h map seen.
func revealAll(w, h int) []byte {
	b := sim.NewExplored(w, h).Bytes()
	for i := range b {
		b[i] = 0xff
	}
	return sim.ExploredFrom(w, h, b).Bytes()
}

// moreSpots scores tile (tx, ty) as a spot of kind what, for the terrain's
// art (see terrain.go):
//
//   - gorge: below a waterfall pouring south between cliffs;
//   - ford: by a ford over a river;
//   - reef: by reefs and sea stacks off a coast;
//   - lake: on the shore of a lake;
//   - cliffs, or cliffs-desert, cliffs-snow, cliffs-ash, cliffs-grass:
//     below a long cliff with a ramp, its high ground of that kind.
//
// ok=false skips the tile; a score of -Inf says what is no kind of spot.
func moreSpots(what string, tx, ty int, t sim.Tile, r sim.Relief, at func(x, y int) (sim.Tile, sim.Relief),
	count func(tx, ty, rx, ry int, f func(sim.Tile, sim.Relief) bool) int) (score float64, px, py int, ok bool) {
	is := func(k sim.Tile) func(sim.Tile, sim.Relief) bool {
		return func(t sim.Tile, _ sim.Relief) bool { return t == k }
	}
	switch {
	case what == "gorge":
		if t != sim.TileWaterfall || r.Flow() != sim.FlowSouth {
			return 0, 0, 0, false
		}
		if up, _ := at(tx, ty-1); up != sim.TileRiver {
			return 0, 0, 0, false
		}
		score = 3*float64(count(tx, ty, 3, 1, is(sim.TileWaterfall))) + float64(count(tx, ty, 9, 5, is(sim.TileCliff))) +
			0.5*float64(count(tx, ty, 9, 5, is(sim.TileRiver))) - 0.5*float64(count(tx, ty, 10, 6, is(sim.TileForest)))
		return score, tx + 3, ty + 3, true
	case what == "ford":
		if t != sim.TileFord {
			return 0, 0, 0, false
		}
		score = float64(count(tx, ty, 8, 5, is(sim.TileRiver))) + 2*float64(count(tx, ty, 2, 2, is(sim.TileFord))) -
			0.5*float64(count(tx, ty, 9, 6, is(sim.TileForest)))
		return score, tx + 2, ty + 2, true
	case what == "reef":
		if !t.Walkable() {
			return 0, 0, 0, false
		}
		reef := count(tx, ty, 12, 7, func(t sim.Tile, r sim.Relief) bool { return t == sim.TileShallow && r.Variant() == sim.VarReef })
		stacks := count(tx, ty, 12, 7, func(t sim.Tile, r sim.Relief) bool { return t == sim.TileCliff && r.Variant() == sim.VarStack })
		if reef < 10 {
			return 0, 0, 0, false
		}
		return float64(reef) + 6*float64(min(stacks, 6)) + 0.5*float64(count(tx, ty, 12, 7, is(sim.TileCliff))), tx, ty, true
	case what == "lake":
		if !t.Walkable() {
			return 0, 0, 0, false
		}
		lake := count(tx, ty, 10, 6, func(t sim.Tile, r sim.Relief) bool { return t == sim.TileWater && r.Variant() == sim.VarLake })
		if lake < 20 {
			return 0, 0, 0, false
		}
		return float64(lake) - 0.7*float64(count(tx, ty, 10, 6, is(sim.TileForest))), tx, ty, true
	case what == "cliffs" || strings.HasPrefix(what, "cliffs-"):
		if !t.Walkable() || r.Variant() != sim.VarRamp {
			return 0, 0, 0, false
		}
		want := map[string]sim.Tile{"cliffs-desert": sim.TileDesert, "cliffs-snow": sim.TileSnow, "cliffs-ash": sim.TileAsh,
			"cliffs-grass": sim.TileGrass}[what]
		if want != 0 && t != want {
			return 0, 0, 0, false
		}
		score = float64(count(tx, ty, 11, 6, is(sim.TileCliff))) - 0.6*float64(count(tx, ty, 11, 7, is(sim.TileForest)))
		return score, tx, ty + 3, true
	}
	return math.Inf(-1), 0, 0, false
}
