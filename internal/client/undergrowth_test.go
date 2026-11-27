package client

import (
	"slices"
	"testing"

	"survivalgame/internal/sim"
)

// plantsPerTile is the average number of plants on a 64x64 field of t, in
// summer; the woods under their trees.
func plantsPerTile(t sim.Tile) float64 {
	var ps []plant
	n := 0
	shade := 0.0
	if t == sim.TileForest {
		shade = 0.8
	}
	for ty := 0; ty < 64; ty++ {
		for tx := 0; tx < 64; tx++ {
			ps = plantsAt(tx, ty, t, shade, sim.Summer, ps[:0])
			n += len(ps)
		}
	}
	return float64(n) / (64 * 64)
}

func TestUndergrowthIsThickestInTheWoods(t *testing.T) {
	forest, marsh, grass := plantsPerTile(sim.TileForest), plantsPerTile(sim.TileMarsh), plantsPerTile(sim.TileGrass)
	if forest < 1.5 || forest <= grass || grass <= marsh || marsh < 0.15 {
		t.Errorf("plants per tile: forest %.2f, marsh %.2f, grass %.2f; want the woods thick, meadows dotted, the swamp mud open", forest, marsh, grass)
	}
	for _, tile := range []sim.Tile{sim.TileWater, sim.TileSand, sim.TileDesert, sim.TileMountain, sim.TileCave, sim.TileAsh, sim.TileLava, sim.TileSnow} {
		if n := plantsPerTile(tile); n != 0 {
			t.Errorf("%.2f plants per tile on tile %d, want none", n, tile)
		}
	}
}

func TestUndergrowthStaysPut(t *testing.T) {
	var kinds [numPlantKinds]int
	for ty := -20; ty < 20; ty++ {
		for tx := -20; tx < 20; tx++ {
			a := plantsOn(tx, ty, sim.TileForest, nil)
			if b := plantsOn(tx, ty, sim.TileForest, nil); !slices.Equal(a, b) {
				t.Fatalf("tile (%d, %d) grows %v, then %v", tx, ty, a, b)
			}
			for _, p := range a {
				if p.x < float64(tx) || p.x >= float64(tx+1) || p.y < float64(ty) || p.y >= float64(ty+1) {
					t.Fatalf("a plant of tile (%d, %d) grows at (%.2f, %.2f)", tx, ty, p.x, p.y)
				}
				if p.variant < 0 || p.variant >= 2*plantVariants {
					t.Fatalf("variant %d out of range", p.variant)
				}
				kinds[p.kind]++
			}
		}
	}
	for k, n := range kinds {
		if n == 0 {
			t.Errorf("no plant of kind %d in the woods", k)
		}
	}
}

// TestUndergrowthClumps checks that the meadow's grass and flowers grow in
// drifts and patches with bare glades between, not evenly: counted in
// squares of 4 by 4 tiles, their variance is well over their mean (it is
// about the mean when plants are strewn at random).
func TestUndergrowthClumps(t *testing.T) {
	for _, k := range []plantKind{plantBlades, plantFlowers} {
		var counts []float64
		var ps []plant
		empty := 0
		for by := 0; by < 160; by += 4 {
			for bx := 0; bx < 160; bx += 4 {
				n := 0.0
				for ty := by; ty < by+4; ty++ {
					for tx := bx; tx < bx+4; tx++ {
						ps = plantsAt(tx, ty, sim.TileGrass, 0, sim.Summer, ps[:0])
						for _, p := range ps {
							if p.kind == k {
								n++
							}
						}
					}
				}
				if n == 0 {
					empty++
				}
				counts = append(counts, n)
			}
		}
		mean, v := 0.0, 0.0
		for _, c := range counts {
			mean += c
		}
		mean /= float64(len(counts))
		for _, c := range counts {
			v += (c - mean) * (c - mean)
		}
		v /= float64(len(counts) - 1)
		if mean == 0 || v/mean < 2 {
			t.Errorf("plant kind %d: %.2f to a square, variance/mean %.2f; want them clumped (over 2)", k, mean, v/mean)
		}
		if empty == 0 {
			t.Errorf("plant kind %d: no square of the meadow without it; want glades", k)
		}
	}
}
