package sim

import (
	"math"
	"testing"
)

// woods measures a 256x256 open world of style st: the share of its land that
// is forest, and the share of forest tiles with a tree.
func woods(t *testing.T, st WorldStyle) (forest, trees float64) {
	t.Helper()
	w, err := Generate(GenConfig{Seed: 3, Width: 256, Height: 256, Style: st})
	if err != nil {
		t.Fatal(err)
	}
	land, nForest, nTrees := 0, 0, 0
	for i, tl := range w.tiles {
		if tl == TileWater {
			continue
		}
		land++
		if tl == TileForest {
			nForest++
			if e := w.entityAt(i%w.width, i/w.width); e != nil && e.Kind == EntityTree {
				nTrees++
			}
		}
	}
	return float64(nForest) / float64(land), float64(nTrees) / float64(nForest)
}

func TestJungleWorldsHaveWiderThickerWoods(t *testing.T) {
	iceForest, _ := woods(t, StyleIce)
	forest, trees := woods(t, StyleJungle)
	if forest < 1.6*iceForest {
		t.Errorf("forest is %.1f%% of the land, %.1f%% before jungle worlds: want much more", 100*forest, 100*iceForest)
	}
	if math.Abs(trees-forestTrees) > 0.06 {
		t.Errorf("%.0f%% of forest tiles have a tree, want about %.0f%%", 100*trees, 100*forestTrees)
	}
	// Worlds made before keep their terrain: it is regenerated from the seed.
	if (&World{style: StyleIce}).forestLine() != forestLine {
		t.Error("an ice-age world would load with different woods")
	}
	if StyleLatest != StyleJungle {
		t.Error("new games are not jungle worlds")
	}
}
