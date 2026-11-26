package main

import (
	"math"

	"survivalgame/internal/sim"
)

// Spots in a world of the designed archipelago (sim.StyleIsles) worth a
// look, for -at:
//
//   - waterfall: below a river dropping over a cliff;
//   - ramp: below a ramp cut through a cliff between two levels;
//   - lagoon: on a spit of sand, a lagoon behind it and reefs out to sea.
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
			default:
				return 0, 0, false
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
