package sim

import (
	"math"
	"slices"
	"sync"
	"testing"
)

// wildWorlds are worlds of 1024 tiles of the newest style, made once for
// the tests of this file: seeds 42 and 7.
var (
	wildOnce   sync.Once
	wildWorlds map[uint64]*World
)

func wildWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	if testing.Short() {
		t.Skip("slow: a world of 1024 tiles")
	}
	wildOnce.Do(func() {
		wildWorlds = map[uint64]*World{}
		for _, s := range []uint64{42, 7} {
			w, err := Generate(GenConfig{Seed: s, Width: SizeHuge, Height: SizeHuge, Style: StyleWilds})
			if err != nil {
				t.Fatal(err)
			}
			wildWorlds[s] = w
		}
	})
	return wildWorlds[seed]
}

// wildGrazer reports whether k is one of the grazers spread by island.
func wildGrazer(k CreatureKind) bool {
	for _, r := range wildRules {
		if r.kind == k {
			return !wildsHunter(k)
		}
	}
	return false
}

// wildIsleOf is the index in w.islands of the island under (x, y), -1 for
// none.
func wildIsleOf(w *World, index []int16, x, y float64) int {
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	if !w.inBounds(tx, ty) || ty >= w.SurfaceHeight() {
		return -1
	}
	return int(index[ty*w.width+tx])
}

// The home island is kind: nothing hunts there by day, and a newcomer sees
// grazers round the spawn.
func TestWildsHomeIsKind(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		index := w.islandIndex()
		home := map[CreatureKind]int{}
		near := 0
		for _, c := range w.fauna.creatures {
			if k := wildIsleOf(w, index, c.X, c.Y); k < 0 || !w.islands[k].Home {
				continue
			}
			home[c.Kind]++
			if wildsHunter(c.Kind) || c.Kind == CreatureSnake || c.Kind.Boss() {
				t.Errorf("seed %d: a %s on the home island", seed, c.Kind.Name())
			}
			if (wildGrazer(c.Kind) || c.Kind == CreatureRabbit) && math.Hypot(c.X-w.spawnX, c.Y-w.spawnY) < wildsNearHi+4 {
				near++
			}
		}
		if near < 10 {
			t.Errorf("seed %d: %d harmless animals within %.0f tiles of the spawn, want 10 or more", seed, near, wildsNearHi+4)
		}
		for _, k := range []CreatureKind{CreatureThesc, CreatureOrnitho, CreatureTrike, CreatureRabbit} {
			if home[k] == 0 {
				t.Errorf("seed %d: no %s on the home island", seed, k.Name())
			}
		}
		// Nor off its shore.
		fromHome := w.homeSeaDistance(index)
		for _, c := range w.fauna.creatures {
			tx, ty := int(c.X), int(c.Y)
			if ty >= w.SurfaceHeight() {
				continue
			}
			if away, ok := wildsHomeAway[c.Kind]; ok && int(fromHome[ty*w.width+tx]) < away {
				t.Errorf("seed %d: a %s %d tiles off the home island", seed, c.Kind.Name(), fromHome[ty*w.width+tx])
			}
		}
		t.Logf("seed %d: on the home island %v; %d harmless animals near the spawn", seed, home, near)
	}
}

// Every island of some size has big animals.
func TestWildsEveryIslandHasFauna(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		index := w.islandIndex()
		groups := make([]int, len(w.islands))
		for _, c := range w.fauna.creatures {
			if k := wildIsleOf(w, index, c.X, c.Y); k >= 0 && wildGrazer(c.Kind) {
				groups[k]++
			}
		}
		for k, is := range w.islands {
			if want := 1 + is.Tiles/wildsMinPer; is.Tiles >= wildsMinTiles && groups[k] < want {
				t.Errorf("seed %d: an island of kind %d and %d tiles has %d grazers, want %d at least", seed, is.Kind, is.Tiles, groups[k], want)
			}
		}
	}
}

// Each kind of island has its own animals.
func TestWildsSignatureSpecies(t *testing.T) {
	signature := map[IslandKind][][]CreatureKind{ // one of each inner list at least
		IslandForest: {{CreatureStego}, {CreatureAchero}, {CreatureAnzu}, {CreatureMonkey}},
		IslandMarsh:  {{CreatureCroc, CreatureBrachy, CreatureChampso}, {CreatureAnzu}, {CreatureToad, CreatureGlowToad}},
		IslandDesert: {{CreatureOrnitho}, {CreatureRex}, {CreatureScorpion}},
		IslandAsh:    {{CreatureRex}},
		IslandSnow:   {{CreatureMammoth}, {CreatureSabre}},
		IslandMonkey: {{CreatureMonkey}},
	}
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		index := w.islandIndex()
		on := make([]map[CreatureKind]int, len(w.islands))
		for _, c := range w.fauna.creatures {
			if k := wildIsleOf(w, index, c.X, c.Y); k >= 0 {
				if on[k] == nil {
					on[k] = map[CreatureKind]int{}
				}
				on[k][c.Kind]++
			}
		}
		for kind, want := range signature {
			best := -1
			for k, is := range w.islands {
				if is.Kind == kind && !is.Home && (best < 0 || is.Tiles > w.islands[best].Tiles) {
					best = k
				}
			}
			if best < 0 {
				t.Errorf("seed %d: no island of kind %d", seed, kind)
				continue
			}
			for _, any := range want {
				n := 0
				for _, k := range any {
					n += on[best][k]
				}
				if n == 0 {
					t.Errorf("seed %d: the biggest island of kind %d has no %s", seed, kind, any[0].Name())
				}
			}
		}
		// The big beasts are a real presence: long-necks in herds, three-horns
		// on more than the home island.
		var brachio, trikeIsles int
		for k := range w.islands {
			brachio += on[k][CreatureBrachio]
			if on[k][CreatureTrike] > 0 {
				trikeIsles++
			}
		}
		if brachio < 50 || trikeIsles < 3 {
			t.Errorf("seed %d: %d long-necks, three-horns on %d islands", seed, brachio, trikeIsles)
		}
	}
}

// Grazers outnumber hunters, four to one at least.
func TestWildsGrazersOutnumberHunters(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		var grazers, hunters int
		for _, c := range w.fauna.creatures {
			switch {
			case wildGrazer(c.Kind):
				grazers++
			case wildsHunter(c.Kind):
				hunters++
			}
		}
		t.Logf("seed %d: %d grazers, %d hunters", seed, grazers, hunters)
		if grazers < 4*hunters {
			t.Errorf("seed %d: %d grazers to %d hunters, want four to one at least", seed, grazers, hunters)
		}
	}
}

// The sea's monsters are far fewer than before, and keep off the home
// island; the plesiosaurs are more.
func TestWildsSeaMonsters(t *testing.T) {
	old := genStyle(t, StyleBeaches, 42, SizeHuge)
	for _, seed := range []uint64{42} {
		w := wildWorld(t, seed)
		count := func(w *World, k CreatureKind) int { return len(creatureByKind(w, k)) }
		for _, k := range []CreatureKind{CreatureMosa, CreatureKraken} {
			if n, was := count(w, k), count(old, k); n > was*3/10 || n == 0 {
				t.Errorf("%d of %s, %d before: want some, and 70%% fewer at least", n, k.Name(), was)
			}
		}
		if n, was := count(w, CreaturePlesio), count(old, CreaturePlesio); n < 2*was {
			t.Errorf("%d plesiosaurs, %d before: want twice as many", n, was)
		}
	}
}

// Respawns follow the same rules: a home island cleared of its animals fills
// again with harmless ones.
func TestWildsRepopulateKeepsTheRings(t *testing.T) {
	w := genStyle(t, StyleWilds, 3, SizeLarge)
	index := w.islandIndex()
	w.fauna.creatures = slices.DeleteFunc(w.fauna.creatures, func(c *Creature) bool { return !c.Kind.Boss() })
	for range 400 {
		w.tick += CreatureRespawnTicks
		w.repopulate()
	}
	got := map[CreatureKind]int{}
	homeN := 0
	for _, c := range w.fauna.creatures {
		got[c.Kind]++
		if k := wildIsleOf(w, index, c.X, c.Y); k >= 0 && w.islands[k].Home {
			homeN++
			if wildsHunter(c.Kind) || c.Kind == CreatureSnake {
				t.Errorf("a %s came back on the home island", c.Kind.Name())
			}
		}
	}
	if homeN < 10 {
		t.Errorf("%d animals came back on the home island", homeN)
	}
	for _, r := range wildRules {
		if w.fauna.target[r.kind] > 0 && got[r.kind] == 0 {
			t.Errorf("no %s came back", r.kind.Name())
		}
	}
}

// The islands are bigger: a large island of 1024 tiles is several screens
// across, the home island has room, and the sea is still most of the world.
func TestWildsIslandsAreBig(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		best := map[IslandKind]int{}
		home := 0
		for _, is := range w.islands {
			if is.Home {
				home = is.Tiles
				continue
			}
			best[is.Kind] = max(best[is.Kind], is.Tiles)
		}
		if home < 65000 {
			t.Errorf("seed %d: the home island has %d tiles", seed, home)
		}
		for _, k := range largeKinds {
			if best[k] < 30000 {
				t.Errorf("seed %d: the biggest island of kind %d has %d tiles", seed, k, best[k])
			}
		}
		if best[IslandForest] < 70000 {
			t.Errorf("seed %d: the deep jungle has %d tiles, want the biggest island", seed, best[IslandForest])
		}
	}
}

// Each large island has a main ground and others besides; the home island
// has grass, woods, water and a beach within reach of the spawn: a walk of
// half a minute or so.
func TestWildsIslandsHaveZones(t *testing.T) {
	const homeReach = 150
	biome := func(t Tile, r Relief) int {
		switch {
		case t == TileSnow && r.Variant() == VarDenseWood:
			return int(TileForest) // pinewoods
		case t == TileLava:
			return int(TileAsh)
		case t == TileCliff:
			return int(TileRocky)
		case t == TileSand || t == TilePath || t == TileFord || !t.Walkable():
			return -1 // the beaches, the trails, the water
		}
		return int(t)
	}
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		index := w.islandIndex()
		shares := make([][NumTiles]int, len(w.islands))
		for i, k := range index {
			if k >= 0 {
				if b := biome(w.tiles[i], w.relief[i]); b >= 0 {
					shares[k][b]++
				}
			}
		}
		for k, is := range w.islands {
			if is.Tiles < 20000 {
				continue
			}
			total, top, others := 0, 0, 0
			for _, n := range shares[k] {
				total += n
				top = max(top, n)
			}
			for _, n := range shares[k] {
				if n != top && n*100 >= total*4 {
					others++
				}
			}
			if top*100 > total*70 || others < 2 {
				t.Errorf("seed %d: island of kind %d: its main ground is %d%% of it, with %d others of 4%% or more", seed, is.Kind, top*100/max(total, 1), others)
			}
		}
		// Round the spawn.
		var grass, woods, water, beach bool
		sx, sy := int(w.spawnX), int(w.spawnY)
		for ty := sy - homeReach; ty <= sy+homeReach; ty++ {
			for tx := sx - homeReach; tx <= sx+homeReach; tx++ {
				switch t := w.TileAt(tx, ty); {
				case t == TileGrass:
					grass = true
				case t == TileForest:
					woods = true
				case t == TileRiver || t == TileWater && w.relief[ty*w.width+tx].Variant() == VarLake:
					water = true
				case t == TileSand:
					beach = true
				}
			}
		}
		if !grass || !woods || !water || !beach {
			t.Errorf("seed %d: round the spawn grass %v, woods %v, fresh water %v, beach %v", seed, grass, woods, water, beach)
		}
	}
}

// The bosses live on the biggest island of their kind, deep in; the
// Megalodon out in open water, its whirlpool clear of every coast.
func TestWildsBossIslands(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := wildWorld(t, seed)
		index := w.islandIndex()
		inland := w.landDistance(60)
		for i := range w.bosses.lairs {
			l := &w.bosses.lairs[i]
			k := BossKind(i)
			if !l.ok {
				if bossActive(k) {
					t.Errorf("seed %d: no lair for %s", seed, k.Name())
				}
				continue
			}
			tx, ty := int(l.x), int(l.y)
			if k == CreatureMegalodon {
				// The nearest island (its sandbar is in the arena on purpose).
				nearest := math.Inf(1)
				for y := ty - 60; y <= ty+60; y++ {
					for x := tx - 60; x <= tx+60; x++ {
						if w.inBounds(x, y) && y < w.SurfaceHeight() && index[y*w.width+x] >= 0 {
							nearest = min(nearest, math.Hypot(float64(x-tx), float64(y-ty)))
						}
					}
				}
				if nearest < MegaArena+10 {
					t.Errorf("seed %d: the Megalodon's lair is %.0f tiles from an island", seed, nearest)
				}
				continue
			}
			want := w.bossIsland(k)
			got := wildIsleOf(w, index, l.x, l.y)
			if k == CreatureSpino && got < 0 {
				got = want // in a pond: checked by its lair test
			}
			if got != want {
				t.Errorf("seed %d: %s lives on island %d, not the biggest of its kind (%d)", seed, k.Name(), got, want)
			}
			if k == CreatureTitanoboa && inland[ty*w.width+tx] < wildsLairCore {
				t.Errorf("seed %d: Titanoboa's lair is %d tiles from the sea", seed, inland[ty*w.width+tx])
			}
		}
	}
}
