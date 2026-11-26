package sim

import (
	"slices"
	"testing"
	"time"
)

// genIsles makes a world of the designed islands, as a new game is made:
// since StyleWoods, with its woods spaced (see woods.go).
func genIsles(t *testing.T, seed uint64, size int) *World {
	t.Helper()
	return genStyle(t, StyleLatest, seed, size)
}

func TestIslesAreDeterministic(t *testing.T) {
	for _, size := range []int{SizeMedium, SizeLarge} {
		a, b := genIsles(t, 9, size), genIsles(t, 9, size)
		if worldHash(a) != worldHash(b) || !slices.Equal(a.relief, b.relief) {
			t.Fatalf("size %d: two worlds from one seed differ", size)
		}
		if c := genIsles(t, 10, size); slices.Equal(a.tiles, c.tiles) {
			t.Fatalf("size %d: two seeds made the same world", size)
		}
	}
}

// Every island has its character, and the world has every new feature of
// the land and the sea.
func TestIslesHaveEveryCharacterAndFeature(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{1, SizeLarge}, {2, SizeLarge}, {42, SizeHuge}} {
		w := genIsles(t, c.seed, c.size)
		var kinds [numIsleKinds]int
		homes := 0
		for _, il := range w.Islands() {
			kinds[il.Kind]++
			if il.Home {
				homes++
			}
		}
		for k, n := range kinds {
			if n == 0 {
				t.Errorf("seed %d size %d: no island of kind %d", c.seed, c.size, k)
			}
		}
		if homes != 1 {
			t.Errorf("seed %d size %d: %d home islands", c.seed, c.size, homes)
		}
		var tiles [NumTiles]int
		var vars [NumGroundVariants]int
		levels := [MaxLevel + 1]int{}
		for i, tl := range w.tiles[:w.width*w.SurfaceHeight()] {
			tiles[tl]++
			if r := w.relief[i]; tl != TileRiver && tl != TileFord && tl != TileWaterfall {
				vars[r.Variant()]++
			}
			levels[w.relief[i].Level()]++
		}
		for _, k := range []Tile{TileShallow, TileRiver, TileFord, TileWaterfall, TileCliff, TileSand, TileForest, TileMarsh, TileSnow, TileDesert, TileAsh, TileLava, TileRocky} {
			if tiles[k] == 0 {
				t.Errorf("seed %d size %d: no tile %d", c.seed, c.size, k)
			}
		}
		for _, v := range []GroundVariant{VarMeadow, VarClearing, VarDenseWood, VarOpenWood, VarOutcrop, VarDunes, VarScree, VarMudflat, VarBoulders, VarRamp, VarReef, VarLagoon, VarShoal, VarLake} {
			if vars[v] == 0 {
				t.Errorf("seed %d size %d: no ground of variant %d", c.seed, c.size, v)
			}
		}
		for l, n := range levels {
			if n == 0 {
				t.Errorf("seed %d size %d: nothing at level %d", c.seed, c.size, l)
			}
		}
		if frac := float64(tiles[TileWater]+tiles[TileShallow]) / float64(w.width*w.SurfaceHeight()); frac < 0.5 {
			t.Errorf("seed %d size %d: only %.0f%% sea", c.seed, c.size, 100*frac)
		}
	}
}

// walkParts labels the walkable tiles of the surface by the part they can
// be walked to from: 4-connected, and with built, with nothing built in
// the way.
func walkParts(w *World, built bool) ([]int32, []int) {
	W, H := w.width, w.SurfaceHeight()
	part := make([]int32, W*H)
	for i := range part {
		part[i] = -1
	}
	free := func(i int) bool { return w.tiles[i].Walkable() && (!built || !w.blocked(i%W, i/W)) }
	var sizes []int
	var q []int
	for start := range part {
		if part[start] >= 0 || !free(start) {
			continue
		}
		n := int32(len(sizes))
		part[start] = n
		q = append(q[:0], start)
		for k := 0; k < len(q); k++ {
			x, y := q[k]%W, q[k]/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; part[j] < 0 && free(j) {
					part[j] = n
					q = append(q, j)
				}
			}
		}
		sizes = append(sizes, len(q))
	}
	return part, sizes
}

// On every island, all the ground can be walked to from all the rest, up
// the ramps and over the fords; every island can be landed on from the sea;
// and from the spawn the whole home island is a walk away. (Other islands
// are a swim or a sail.)
func TestIslesCanBeWalkedEndToEnd(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{1, SizeMedium}, {2, SizeMedium}, {3, SizeLarge}, {42, SizeHuge}} {
		w := genIsles(t, c.seed, c.size)
		W := w.width
		part, sizes := walkParts(w, false)
		built, bsizes := walkParts(w, true)
		home := part[int(w.spawnY)*W+int(w.spawnX)]
		if home < 0 {
			t.Fatalf("seed %d size %d: the spawn is not on open ground", c.seed, c.size)
		}
		for _, isl := range w.landMasses() {
			if isl[0]/W >= w.SurfaceHeight() {
				continue
			}
			parts := map[int32]int{}
			big := map[int32]bool{} // parts left by what is built, but for slivers behind it
			landing := false
			for _, i := range isl {
				if b := built[i]; b >= 0 && bsizes[b] > 8 {
					big[b] = true
				}
				p := part[i]
				if p < 0 {
					continue
				}
				parts[p]++
				x, y := i%W, i/W
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					if w.TileAt(x+d[0], y+d[1]).Sea() {
						landing = true
					}
				}
			}
			if len(parts) == 0 {
				continue // a rock in the sea
			}
			if len(parts) > 1 {
				var got []int
				for p := range parts {
					got = append(got, sizes[p])
				}
				slices.Sort(got)
				t.Errorf("seed %d size %d: an island of %d tiles falls in %d parts that cannot be walked between: %v",
					c.seed, c.size, len(isl), len(parts), got)
			}
			if !landing {
				t.Errorf("seed %d size %d: an island of %d tiles has no shore to land on", c.seed, c.size, len(isl))
			}
			if len(big) > 1 {
				t.Errorf("seed %d size %d: what is built cuts an island of %d tiles in %d", c.seed, c.size, len(isl), len(big))
			}
		}
	}
}

// The spawn is gentle ground: grass or woods, flat, with nothing steep,
// nothing to drown in and nothing that blocks the way near.
func TestIslesSpawnIsSafe(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3, 4, 5} {
		w := genIsles(t, seed, SizeLarge)
		sx, sy := int(w.spawnX), int(w.spawnY)
		if tl := w.TileAt(sx, sy); tl != TileGrass && tl != TileForest {
			t.Errorf("seed %d: spawn on tile %d", seed, tl)
		}
		for y := sy - 6; y <= sy+6; y++ {
			for x := sx - 6; x <= sx+6; x++ {
				switch tl := w.TileAt(x, y); {
				case tl == TileCliff || tl == TileRiver || tl == TileWaterfall:
					t.Errorf("seed %d: tile %d %d tiles from the spawn", seed, tl, max(abs(x-sx), abs(y-sy)))
				case w.LevelAt(x, y) != w.LevelAt(sx, sy):
					t.Errorf("seed %d: a slope by the spawn", seed)
				}
			}
		}
		if il := w.Islands(); !slices.ContainsFunc(il, func(i Island) bool { return i.Home && i.Kind == IslandMeadow }) {
			t.Errorf("seed %d: the home island is not a meadow", seed)
		}
	}
}

// Every river runs to the sea: each stretch of river, its lakes and its
// falls together, reaches the sea somewhere.
func TestRiversReachTheSea(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3} {
		w := genIsles(t, seed, SizeLarge)
		W, H := w.width, w.SurfaceHeight()
		river := func(i int) bool {
			t := w.tiles[i]
			return t == TileRiver || t == TileFord || t == TileWaterfall || w.lakeAt(i)
		}
		seen := make([]bool, W*H)
		systems := 0
		for start := range seen {
			if seen[start] || !river(start) || w.lakeAt(start) {
				continue
			}
			q := []int{start}
			seen[start] = true
			sea := false
			for k := 0; k < len(q); k++ {
				x, y := q[k]%W, q[k]/W
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if nx < 0 || ny < 0 || nx >= W || ny >= H {
						continue
					}
					j := ny*W + nx
					if w.seaAt(j) {
						sea = true
					}
					if !seen[j] && river(j) {
						seen[j] = true
						q = append(q, j)
					}
				}
			}
			systems++
			if !sea {
				t.Errorf("seed %d: a river of %d tiles at (%d,%d) never reaches the sea", seed, len(q), start%W, start/W)
			}
		}
		if systems < 4 {
			t.Errorf("seed %d: only %d rivers", seed, systems)
		}
		// Each river tile flows to another river tile, a lake or the sea.
		for i := range seen {
			if t := w.tiles[i]; t != TileRiver && t != TileFord && t != TileWaterfall {
				continue
			}
			dx, dy := w.relief[i].Flow().Delta()
			if dx == 0 && dy == 0 {
				t.Fatalf("seed %d: river at (%d,%d) does not flow", seed, i%W, i/W)
			}
		}
	}
}

// A bigger world has more of everything, by its land.
func TestIslesCountsGrowWithTheWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	type tally struct {
		land, villages, mouths, creatures, ores, treasure, flora, toads, clay, homes int
	}
	count := func(w *World) tally {
		var c tally
		for i, tl := range w.tiles[:w.width*w.SurfaceHeight()] {
			if tl.Walkable() {
				c.land++
			}
			_ = i
		}
		c.villages = len(w.villages)
		for _, e := range w.entities {
			switch e.Kind {
			case EntityCaveMouth:
				c.mouths++
			case EntityCopperOre, EntityIronOre, EntityPlatinumOre, EntityDiamondOre, EntityCoal:
				c.ores++
			case EntityTreasure:
				c.treasure++
			case EntityFlora:
				c.flora++
			case EntityClay:
				c.clay++
			case EntityBurrow, EntityDen:
				c.homes++
			}
		}
		for _, cr := range w.fauna.creatures {
			c.creatures++
			if IsToad(cr.Kind) {
				c.toads++
			}
		}
		return c
	}
	small, big := count(genIsles(t, 5, SizeLarge)), count(genIsles(t, 5, SizeHuge))
	ratio := func(a, b int) float64 { return float64(b) / float64(max(a, 1)) }
	t.Logf("512: %+v", small)
	t.Logf("1024: %+v", big)
	for name, r := range map[string]float64{
		"land": ratio(small.land, big.land), "villages": ratio(small.villages, big.villages),
		"cave mouths": ratio(small.mouths, big.mouths), "creatures": ratio(small.creatures, big.creatures),
		"ores": ratio(small.ores, big.ores), "treasure": ratio(small.treasure, big.treasure),
		"plants": ratio(small.flora, big.flora), "toads": ratio(small.toads, big.toads),
		"clay": ratio(small.clay, big.clay), "burrows and dens": ratio(small.homes, big.homes),
	} {
		if r < 2 || r > 12 {
			t.Errorf("%s: %.1f times as many in a world of four times the area", name, r)
		}
	}
}

func TestColossalIslesAreQuickEnough(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	t0 := time.Now()
	w := genIsles(t, 1, SizeColossal)
	took := time.Since(t0)
	t.Logf("2048x2048: %v, %d entities, %d creatures, %d islands", took, len(w.entities), len(w.fauna.creatures), len(w.Islands()))
	if took > 25*time.Second {
		t.Errorf("generating took %v", took)
	}
	t0 = time.Now()
	w.AddPlayer("a")
	for range 20 {
		w.Step()
	}
	// A tick has 50 ms at 20 Hz; 40 ms leaves room for a busy test machine
	// (25 ms failed now and then on a desktop shared with other work).
	if per := time.Since(t0) / 20; per > 40*time.Millisecond {
		t.Errorf("a tick takes %v", per)
	}
}

func TestIslesSaveAndLoad(t *testing.T) {
	w := genIsles(t, 7, SizeMedium)
	w.AddPlayer("a")
	stepN(w, 10)
	d := w.Save()
	if d.Style != StyleLatest || d.Version != SaveVersion {
		t.Fatalf("saved style %d version %d", d.Style, d.Version)
	}
	l, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(l.tiles, w.tiles) || !slices.Equal(l.relief, w.relief) || len(l.entities) != len(w.entities) {
		t.Fatalf("reloaded world differs: %d entities, want %d", len(l.entities), len(w.entities))
	}
	if l.Relief() == nil || len(l.Relief()) != len(l.tiles) {
		t.Fatal("no relief after loading")
	}
}

// Ramps and fords are open: nothing stands on them that blocks the way,
// and a player walks over a ford, slowly, and can swim the river.
func TestCrossingsAreOpen(t *testing.T) {
	w := genIsles(t, 3, SizeLarge)
	fords, ramps := 0, 0
	for i := range w.tiles[:w.width*w.SurfaceHeight()] {
		if !w.crossing(i) {
			continue
		}
		if w.tiles[i] == TileFord {
			fords++
		} else {
			ramps++
		}
		if w.blocked(i%w.width, i/w.width) {
			t.Errorf("something blocks the crossing at (%d,%d)", i%w.width, i/w.width)
		}
	}
	if fords == 0 || ramps == 0 {
		t.Fatalf("%d fords, %d ramps", fords, ramps)
	}
	if !TileFord.Walkable() || TileRiver.Walkable() || TileCliff.Walkable() || TileShallow.Walkable() || TileWaterfall.Walkable() {
		t.Fatal("wrong tiles walkable")
	}
	if !TileRiver.Swimmable() || !TileShallow.Swimmable() || TileWaterfall.Swimmable() || TileCliff.Swimmable() {
		t.Fatal("wrong tiles swimmable")
	}
}
