package sim

import (
	"slices"
	"testing"
)

// migrated is w saved as a game of version 19 and loaded again: its woods
// thinned.
func migrated(t *testing.T, w *World) *World {
	t.Helper()
	d := w.Save()
	d.Version = 19
	l, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// treesOf lists the trees of w by id.
func treesOf(w *World) map[EntityID]Entity {
	out := map[EntityID]Entity{}
	for _, e := range w.entities {
		if e.Kind == EntityTree {
			out[e.ID] = *e
		}
	}
	return out
}

// hiveRatio is how many trees of the woods (giants with them) there are to
// a beehive in w.
func hiveRatio(w *World) (trees, hives int) {
	for _, e := range w.entities {
		if e.Kind == EntityTree && w.TileAt(e.TX, e.TY) == TileForest || e.Kind == EntityGiantTree {
			trees++
			if e.HasHive() {
				hives++
			}
		}
	}
	return trees, hives
}

// An older save of a world with thick woods has them thinned once on
// loading: what players planted, built by, chopped at or robbed the bees of
// is left be, the hives are cut back to the trees, and loading the game
// again thins nothing more.
func TestOldSaveWoodsThinnedOnce(t *testing.T) {
	for _, style := range []WorldStyle{StyleIsles, StyleSeas} {
		w := genStyle(t, style, 7, SizeMedium)
		before := treesOf(w)
		trees0, hives0 := hiveRatio(w)

		// A player's doings in the thick of the woods: trees planted, a camp,
		// a tree half chopped, a hive robbed.
		var planted []EntityID
		var camp *Entity
		var chopped, hive EntityID
		for _, e := range w.sortedEntities() {
			if e.Kind != EntityTree || w.TileAt(e.TX, e.TY) != TileForest {
				continue
			}
			switch {
			case chopped == 0:
				chopped = e.ID
				e.HP = 1
			case hive == 0 && e.Var != hiveVar:
				hive = e.ID
				e.Var, e.HoneyAt = hiveVar, w.tick+HiveRefillTicks
			}
			for _, d := range [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
				x, y := e.TX+d[0], e.TY+d[1]
				if len(planted) < 12 && w.TileAt(x, y) == TileForest && w.canPlace(x, y) {
					planted = append(planted, w.addEntity(EntityTree, x, y).ID) // grown from a seed: a new id
				} else if camp == nil && len(planted) >= 12 && w.TileAt(x, y) == TileForest && w.canPlace(x, y) {
					camp = w.addEntity(EntityCampfire, x, y)
				}
			}
		}
		if len(planted) < 12 || camp == nil {
			t.Fatalf("style %d: planted %d trees, camp %v", style, len(planted), camp)
		}

		l := migrated(t, w)
		after := treesOf(l)
		t.Logf("style %d: %d trees before, %d after", style, len(before)+len(planted), len(after))
		if len(after) > len(before)*6/10 {
			t.Errorf("style %d: %d trees of %d left: hardly thinned", style, len(after), len(before))
		}
		for _, id := range append(planted, chopped, hive) {
			if _, ok := after[id]; !ok {
				t.Errorf("style %d: tree %d (planted, chopped or with bees robbed) was taken away", style, id)
			}
		}
		if !l.entities[hive].HasHive() {
			t.Errorf("style %d: the robbed hive was taken away", style)
		}
		trees, hives := hiveRatio(l)
		t.Logf("style %d: hives %d to %d trees before, %d to %d after", style, hives0, trees0, hives, trees)
		// The bees left the trees thinned away; what is left is cut back to a
		// hive to hivesPerTrees trees, never more.
		if hives == 0 || hives > max(1, trees/hivesPerTrees) || hives*hivesPerTrees*2 < trees {
			t.Errorf("style %d: %d hives to %d trees after thinning, want about one to %d", style, hives, trees, hivesPerTrees)
		}
		for id, e := range treesOf(w) {
			if dx, dy := e.TX-camp.TX, e.TY-camp.TY; dx*dx+dy*dy <= thinNearBuilt*thinNearBuilt {
				if _, ok := after[id]; !ok {
					t.Errorf("style %d: the tree at %d,%d by the camp was taken away", style, e.TX, e.TY)
				}
			}
		}
		sea := seaJoined(w.width, w.height, w.tiles)
		for _, e := range w.entities {
			if e.Kind == EntityFlora && FloraKind(e.Var).Pad() && sea[e.TY*w.width+e.TX] {
				continue // a lily on the sea: pulled up on loading, see pullSeaLilies
			}
			if e.Kind != EntityTree && l.entities[e.ID] == nil {
				t.Errorf("style %d: %s %d was taken away", style, e.Kind.Name(), e.ID)
			}
		}

		// Saved again (version 20) and loaded: nothing more goes, and the
		// thinning is the same every time.
		again, err := Load(l.Save())
		if err != nil {
			t.Fatal(err)
		}
		if len(treesOf(again)) != len(after) {
			t.Errorf("style %d: loading the thinned game again left %d trees of %d", style, len(treesOf(again)), len(after))
		}
		twice := treesOf(migrated(t, w))
		if len(twice) != len(after) {
			t.Fatalf("style %d: two loads of one save kept %d and %d trees", style, len(after), len(twice))
		}
		for id := range after {
			if _, ok := twice[id]; !ok {
				t.Fatalf("style %d: two loads of one save thinned differently", style)
			}
		}
	}
}

// The thinned woods of an old world look like the new ones: ordinary
// stretches and thick ones, no wall.
func TestOldSaveWoodsThinnedToNewLook(t *testing.T) {
	w := migrated(t, genStyle(t, StyleIsles, 3, SizeHuge))
	views := woodsViews(w)
	forest, _ := canopyStats(w, views["forest"])
	dense, _ := canopyStats(w, views["dense"])
	t.Logf("thinned: canopy %.0f%% in ordinary woods, %.0f%% in thick ones", 100*forest, 100*dense)
	if forest < 0.33 || forest > 0.5 {
		t.Errorf("thinned ordinary woods cover %.0f%%, want 35 to 50%%", 100*forest)
	}
	if dense < forest+0.1 || dense > 0.7 {
		t.Errorf("thinned thick woods cover %.0f%%, want well above the ordinary (%.0f%%) and at most 70%%", 100*dense, 100*forest)
	}
}

// A save of version 20 is not thinned, of any style.
func TestNewSaveWoodsNotThinned(t *testing.T) {
	for _, style := range []WorldStyle{StyleSeas, StyleWoods} {
		w := genStyle(t, style, 3, SizeMedium)
		l, err := Load(w.Save())
		if err != nil {
			t.Fatal(err)
		}
		a, b := treesOf(w), treesOf(l)
		if len(a) != len(b) {
			t.Errorf("style %d: %d trees saved, %d loaded", style, len(a), len(b))
		}
		ids := func(m map[EntityID]Entity) []EntityID {
			var s []EntityID
			for id := range m {
				s = append(s, id)
			}
			slices.Sort(s)
			return s
		}
		if !slices.Equal(ids(a), ids(b)) {
			t.Errorf("style %d: the trees loaded are not the trees saved", style)
		}
	}
}
