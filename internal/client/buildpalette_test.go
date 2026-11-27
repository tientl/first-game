package client

import (
	"testing"

	"survivalgame/internal/sim"
)

// The palette's rows hold every piece once, under its section, however
// many columns fit.
func TestPaletteRowsHoldEveryPiece(t *testing.T) {
	pieces := sim.DevPieces()
	for _, cols := range []int{4, 7, palMaxCols} {
		seen := map[int]int{}
		heads := 0
		for _, r := range paletteRows(cols) {
			if r.head {
				heads++
				continue
			}
			if len(r.pieces) == 0 || len(r.pieces) > cols {
				t.Errorf("%d columns: a row of %d", cols, len(r.pieces))
			}
			for _, i := range r.pieces {
				seen[i]++
				if pieces[i].Group != r.group {
					t.Errorf("%s under %s", pieces[i].Name, sim.DevGroupNames[r.group])
				}
			}
		}
		if len(seen) != len(pieces) || heads != int(sim.NumDevGroups) {
			t.Errorf("%d columns: %d of %d pieces, %d sections", cols, len(seen), len(pieces), heads)
		}
		for i, n := range seen {
			if n != 1 {
				t.Errorf("%s shown %d times", pieces[i].Name, n)
			}
		}
	}
}

// In free build every recipe is lit, but a second of a tool.
func TestFreeBuildLightsEveryRecipe(t *testing.T) {
	g := &Game{}
	r, _ := sim.RecipeByID(sim.RecipeSailboat)
	if g.canCraft(r) {
		t.Fatal("a sailboat from an empty bag outside free build")
	}
	g.curr.Self.DevBuild = true
	for _, r := range recipes() {
		if !g.canCraft(r) {
			t.Errorf("%s is not lit in free build", r.Name)
		}
	}
	g.curr.Self.Inv.Add(sim.ItemAxe, 1)
	if r, _ := sim.RecipeByID(sim.RecipeAxe); g.canCraft(r) {
		t.Error("a second stone axe is lit")
	}
}
