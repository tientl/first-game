package sim

import (
	"math"
	"testing"
	"time"
)

// About a third of the woods is thick, the rest ordinary, turning from one
// into the other gradually; the jungle islands lean thick and the woods by
// the spawn ordinary.
func TestWoodsThickAndOrdinary(t *testing.T) {
	w := genStyle(t, StyleWoods, 3, SizeHuge)
	thick := thickOf(w)
	kinds := w.islandKinds()
	W := w.width
	var all, jungle, home, near, between, jumps int
	var thickAll, thickJungle, thickHome, thickNear int
	for i, th := range thick {
		if !w.woodland(i) {
			continue
		}
		all++
		t1 := th > 0.5
		if t1 {
			thickAll++
		}
		if th > 0.05 && th < 0.95 {
			between++
		}
		switch kinds[i] {
		case islandJungle:
			jungle++
			if t1 {
				thickJungle++
			}
		case islandOpen:
			home++
			if t1 {
				thickHome++
			}
		}
		x, y := i%W, i/W
		if math.Hypot(float64(x)-w.spawnX, float64(y)-w.spawnY) < 40 {
			near++
			if t1 {
				thickNear++
			}
		}
		if x+1 < W && w.woodland(i+1) && math.Abs(float64(thick[i+1]-th)) > 0.5 {
			jumps++
		}
	}
	share := func(a, b int) float64 { return float64(a) / float64(max(b, 1)) }
	t.Logf("thick: %.0f%% of the woods, %.0f%% on the jungle islands, %.0f%% on the meadows, %.0f%% near the spawn; %.0f%% of the woods between",
		100*share(thickAll, all), 100*share(thickJungle, jungle), 100*share(thickHome, home), 100*share(thickNear, near), 100*share(between, all))
	if s := share(thickAll, all); s < 0.3 || s > 0.4 {
		t.Errorf("%.0f%% of the woods is thick, want 30 to 40%%", 100*s)
	}
	if share(thickJungle, jungle) <= share(thickAll, all) || share(thickHome, home) >= share(thickAll, all) {
		t.Errorf("the jungle islands (%.0f%%) should lean thick and the meadows (%.0f%%) ordinary", 100*share(thickJungle, jungle), 100*share(thickHome, home))
	}
	if near > 0 && share(thickNear, near) > 0.1 {
		t.Errorf("%.0f%% of the woods by the spawn is thick", 100*share(thickNear, near))
	}
	if share(between, all) < 0.04 || share(jumps, all) > 0.002 {
		t.Errorf("the woods turn thick too suddenly: %d tiles between, %d jumps", between, jumps)
	}
}

// A new world has a beehive to about hivesPerTrees trees of its woods.
func TestNewWorldHivesInProportion(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := genStyle(t, StyleWoods, seed, SizeLarge)
		trees, hives := hiveRatio(w)
		t.Logf("seed %d: %d hives to %d trees", seed, hives, trees)
		if hives == 0 || trees/hives < 40 || trees/hives > 60 {
			t.Errorf("seed %d: %d hives to %d trees, want one to 40 to 60", seed, hives, trees)
		}
	}
}

// A felled big tree grows back: stump, sapling, young tree, grown tree, on
// time, and the grown tree is the same size it was.
func TestFelledTreeGrowsBackInStages(t *testing.T) {
	w := genStyle(t, StyleWoods, 7, SizeMedium)
	var tree *Entity
	for _, e := range w.sortedEntities() {
		if e.Kind == EntityTree && e.Big && w.planned(e.TX, e.TY) == planGrown {
			tree = e
			break
		}
	}
	if tree == nil {
		t.Fatal("no grown tree")
	}
	if w.DevFell(float64(tree.TX)+0.5, float64(tree.TY)+0.5, 0.1) != 1 {
		t.Fatal("did not fell it")
	}
	stages := []struct {
		after uint64
		kind  EntityKind
		ready bool
		big   bool
	}{
		{StumpTicks - 1, EntityTree, false, true},
		{1, EntitySapling, false, false},
		{SeedTree.growTicks() - 1, EntitySapling, false, false},
		{1, EntityTree, true, false},
		{YoungTreeTicks - 1, EntityTree, true, false},
		{1, EntityTree, true, true},
	}
	for k, s := range stages {
		w.DevPassTime(s.after)
		if tree.Kind != s.kind || tree.Ready != s.ready || tree.Big != s.big {
			t.Fatalf("stage %d: %s ready=%v big=%v, want %s ready=%v big=%v", k, tree.Kind.Name(), tree.Ready, tree.Big, s.kind.Name(), s.ready, s.big)
		}
	}
	if tree.HP != maxHP(EntityTree, true) {
		t.Errorf("grown again with %d hp, want %d", tree.HP, maxHP(EntityTree, true))
	}

	// Its stages survive a save.
	w.DevFell(float64(tree.TX)+0.5, float64(tree.TY)+0.5, 0.1)
	w.DevPassTime(StumpTicks + 10)
	l, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	s := l.entities[tree.ID]
	if s == nil || s.Kind != EntitySapling || s.Fuel != saplingGrows || s.RegrowAt != tree.RegrowAt {
		t.Fatalf("a sapling saved and loaded: %+v", s)
	}
	l.DevPassTime(SeedTree.growTicks() + YoungTreeTicks)
	if !s.Big || !s.Ready {
		t.Errorf("the loaded sapling did not grow up: %+v", s)
	}
}

// Where a tree of the plan is gone, the woods grow back into the gap, but
// not by a camp; and the same world grows back the same way.
func TestWoodsFillTheirGaps(t *testing.T) {
	grow := func() (w *World, gaps [][2]int, camp [2]int, before EntityID) {
		w = genStyle(t, StyleWoods, 7, SizeMedium)
		camped := false
		for _, e := range w.sortedEntities() {
			if e.Kind != EntityTree || w.TileAt(e.TX, e.TY) != TileForest || w.nearWorks(e.TX, e.TY) {
				continue
			}
			switch {
			case len(gaps) < 40:
				gaps = append(gaps, [2]int{e.TX, e.TY})
				w.removeEntity(e)
			case !camped && w.canPlace(e.TX+1, e.TY):
				camped, camp = true, [2]int{e.TX, e.TY}
				w.removeEntity(e)
				w.addEntity(EntityCampfire, e.TX+1, e.TY).Fuel = 1 << 30
			}
		}
		before = w.nextEnt
		w.DevPassTime(3 * fillCycleTicks)
		return w, gaps, camp, before
	}
	w, gaps, camp, before := grow()
	back := 0
	for _, g := range gaps {
		if e := w.entityAt(g[0], g[1]); e != nil && (e.Kind == EntitySapling || e.Kind == EntityTree) {
			back++
		}
	}
	t.Logf("%d of %d gaps grown back into after three scans of the map", back, len(gaps))
	if back < len(gaps)/2 {
		t.Errorf("only %d of %d gaps grew back", back, len(gaps))
	}
	for _, e := range w.entities {
		if e.ID <= before || e.Kind != EntitySapling {
			continue
		}
		if w.nearWorks(e.TX, e.TY) {
			t.Errorf("a sapling came up by people's works at %d,%d", e.TX, e.TY)
		}
		if w.planned(e.TX, e.TY) == planNone {
			t.Errorf("a sapling came up at %d,%d, where the woods had no tree", e.TX, e.TY)
		}
	}
	if e := w.entityAt(camp[0], camp[1]); e != nil && e.ID > before {
		t.Errorf("a %s came up by the camp", e.Kind.Name())
	}
	again, _, _, _ := grow()
	if worldHash(again) != worldHash(w) {
		t.Error("the same world grew back differently")
	}
}

// The scan for gaps and the regrowth cost next to nothing a tick, even on
// the biggest map.
func TestWoodsRegrowthTickTime(t *testing.T) {
	if testing.Short() {
		t.Skip("a 2048 world")
	}
	w := genStyle(t, StyleWoods, 5, SizeColossal)
	w.DevFell(w.spawnX, w.spawnY, 30) // something regrowing
	const n = 2000
	t0 := time.Now()
	for range n {
		w.tick++
		w.stepWoodsFill()
	}
	fill := time.Since(t0) / n
	t0 = time.Now()
	w.DevPassTime(n)
	both := time.Since(t0) / n
	t.Logf("2048: the gap scan takes %v a tick, with all regrowth %v", fill, both)
	if fill > 200*time.Microsecond || both > 2*time.Millisecond {
		t.Errorf("the woods' regrowth takes %v a tick (the scan %v)", both, fill)
	}
}
