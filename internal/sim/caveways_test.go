package sim

import (
	"fmt"
	"math"
	"testing"
)

// The cave ways of StyleCaveWays: see caveways.go.

// caveWaysWorlds are the worlds the cave ways are looked at in: those the
// other cave tests make too (genCaves keeps them).
var caveWaysWorlds = []struct {
	seed uint64
	size int
}{{42, SizeHuge}, {3, SizeHuge}, {42, SizeLarge}, {3, SizeLarge}}

func TestCaveWaysForkTheMainRoute(t *testing.T) {
	medium, levels, sides, short := 0, 0, 0, 0
	for _, g := range caveWaysWorlds {
		w := genCaves(t, g.seed, g.size)
		for _, s := range w.caves.systems {
			at := func(lv int) string {
				return fmt.Sprintf("seed %d size %d island %d (%d tiles) level %d", g.seed, g.size, s.island, s.area, lv)
			}
			for lv := range s.levels {
				st := caveStatsOf(s, lv)
				// Small and middling islands have more than a hall and a
				// chamber: three chambers at least on each level.
				if s.area >= caveWaysFewFrom && st.chambers < 1+caveWaysFew {
					t.Errorf("%s: %d chambers with the hall or the arrival", at(lv), st.chambers)
				}
				if s.area >= 6000 && s.area <= 15000 {
					medium++
					if st.routes < 2 {
						t.Errorf("%s: a middling island's level with %d ways from its way in to its way on", at(lv), st.routes)
					}
				}
				if st.chambers < 4 {
					continue
				}
				levels++
				// Two ways through, with no passage in common, parting where
				// three passages or more meet.
				if st.routes < 2 {
					t.Errorf("%s: %d chambers, but %d way through", at(lv), st.chambers, st.routes)
				}
				if st.forks < 2 {
					t.Errorf("%s: %d chambers, but %d forks", at(lv), st.chambers, st.forks)
				}
				// Some way leads off every third of the main route (off a
				// short one, a tunnel from the hall to the far end of a small
				// cave, somewhere along it).
				a, b := s.levelEnds(lv)
				route, along := s.routeAlong(a, b)
				if along[len(along)-1] < caveThirdsFrom {
					short++
				}
				for third := range routeThirds(along[len(along)-1]) {
					if s.sideIn(route, along, third) {
						sides++
					} else {
						t.Errorf("%s: nothing leads off third %d of the main route (%d chambers on it, %.0f tiles)", at(lv), third, len(route), along[len(along)-1])
					}
				}
			}
		}
	}
	t.Logf("%d levels of middling islands, %d levels of four chambers or more (%d with a short main route), %d thirds of a main route with a way off", medium, levels, short, sides)
	if medium == 0 || levels == 0 {
		t.Fatal("no middling islands, or no levels of four chambers")
	}
}

func TestCaveWaysForksPocketsAndShafts(t *testing.T) {
	junctions, pockets, shafts := 0, 0, 0
	for _, g := range caveWaysWorlds[:2] {
		w := genCaves(t, g.seed, g.size)
		for _, s := range w.caves.systems {
			for i, c := range s.chambers {
				switch {
				case c.junction:
					junctions++
					if len(c.adj) < 3 {
						t.Errorf("seed %d island %d: a fork in a passage with %d ways", g.seed, s.island, len(c.adj))
					}
				case c.pocket != 0:
					pockets++
					if c.r < cavePocketMin || c.r > cavePocketMax || len(c.adj) != 1 {
						t.Errorf("seed %d island %d: a side pocket of radius %.1f with %d ways in", g.seed, s.island, c.r, len(c.adj))
					}
					if _, ok := chamberTile(w, s, i); !ok {
						t.Errorf("seed %d island %d: side pocket %d was not dug, or cannot be reached", g.seed, s.island, i)
					}
				}
			}
			// The main route's tunnels are wide, and so are those dug to give
			// it a second way.
			for _, tt := range s.tunnels {
				if tt.route && tt.width < caveRouteMin {
					t.Errorf("seed %d island %d: a tunnel of the main route %.1f wide", g.seed, s.island, tt.width)
				}
			}
			// The big islands have a second way down, far from the first,
			// with its rope back up.
			if s.area >= caveTwoWaysDown && s.levels == 2 {
				if len(s.shafts) == 0 {
					t.Errorf("seed %d island %d (%d tiles): one way down", g.seed, s.island, s.area)
				}
				for _, sh := range s.shafts {
					shafts++
					l, ok := w.links[sh.down]
					if !ok || w.caveLevel(int(l.y)) != 1 {
						t.Errorf("seed %d island %d: the second hole down leads nowhere", g.seed, s.island)
					}
					if back, ok := w.links[sh.up]; !ok || w.caveLevel(int(back.y)) != 0 {
						t.Errorf("seed %d island %d: no rope up from the second hole", g.seed, s.island)
					}
					if d := hypot(sh.down[0]-s.down[0], sh.down[1]-s.down[1]); d < caveShaftApart {
						t.Errorf("seed %d island %d: the two ways down %.0f tiles apart", g.seed, s.island, d)
					}
				}
			}
		}
	}
	t.Logf("%d forks in a passage, %d side pockets, %d second ways down", junctions, pockets, shafts)
	if junctions == 0 || pockets == 0 || shafts == 0 {
		t.Fatal("no forks in a passage, side pockets or second ways down")
	}
}

func hypot(dx, dy int) float64 { return math.Hypot(float64(dx), float64(dy)) }

func TestCaveWaysChests(t *testing.T) {
	w := genCaves(t, 42, SizeHuge)
	var leaves, leafChests, pocketN, pocketChests int
	kinds := [numChestKinds]int{}
	for _, s := range w.caves.systems {
		ring := w.caves.rings[s.island]
		deep := 0
		for _, e := range w.entities {
			if e.Kind != EntityTreasure {
				continue
			}
			lv := w.caveLevel(e.TY)
			if lv < 0 || lv >= s.levels {
				continue
			}
			cell, ok := s.gridAt(e.TX, e.TY-CaveBandTop(w.surfaceH, lv))
			if !ok || s.owner[lv][cell] == 0 {
				continue
			}
			ci := int(s.owner[lv][cell]) - 1
			kind := int(e.Var >> 2)
			if kind >= numChestKinds || int(e.Var&3) != ring {
				t.Errorf("island %d (ring %d): a chest of Var %d", s.island, ring, e.Var)
				continue
			}
			kinds[kind]++
			switch c := s.chambers[ci]; {
			case kind == chestDeep:
				deep++
				if ci != s.dest || s.levels != 2 {
					t.Errorf("island %d: the deep hoard in chamber %d, the destination is %d", s.island, ci, s.dest)
				}
			case c.pocket != 0:
				pocketChests++
				if kind != chestCache {
					t.Errorf("island %d: a chest of kind %d in a side pocket", s.island, kind)
				}
			case c.leaf:
				leafChests++
			}
		}
		if s.levels == 2 && deep != 1 {
			t.Errorf("island %d: %d deep hoards", s.island, deep)
		}
		for _, c := range s.chambers {
			switch {
			case c.pocket != 0:
				pocketN++
			case c.leaf:
				leaves++
			}
		}
	}
	t.Logf("chests by kind (cache, side, end, deep) %v; %d of %d side branches, %d of %d pockets", kinds, leafChests, leaves, pocketChests, pocketN)
	// Far fewer than the four in five of the side branches of StyleCaves,
	// and a small cache in one pocket in ten or so.
	if float64(leafChests) > 0.45*float64(leaves)+2 {
		t.Errorf("a chest at the end of %d of %d side branches", leafChests, leaves)
	}
	if share := float64(pocketChests) / float64(pocketN); share < 0.03 || share > 0.2 {
		t.Errorf("a chest in %.2f of the side pockets", share)
	}
	// The hoards that will be guarded are a few.
	if kinds[chestDeep] == 0 || kinds[chestDeep] > 20 {
		t.Errorf("%d deep hoards", kinds[chestDeep])
	}
}

func TestCaveWaysLootByRing(t *testing.T) {
	const n = 6000
	best := map[ItemKind]bool{ItemBigPotion: true, ItemRexTooth: true, ItemIvorySpear: true}
	gear := map[ItemKind]bool{ItemIronOre: true, ItemPlatinumOre: true, ItemDiamond: true, ItemDinoHide: true, ItemManEaterSeed: true,
		ItemBigPotion: true, ItemRexTooth: true, ItemIvorySpear: true}
	// The old chests at the end of a main route, deep down, before.
	oldTop := 0.0
	for id := 1; id <= n; id++ {
		for _, it := range treasureLoot(EntityID(id), caveTreasureTop+20) {
			if best[it.Item] || it.Item == ItemDiamond {
				oldTop++
			}
		}
	}
	oldTop /= n
	for kind := range numChestKinds {
		for ring := range 3 {
			chance := map[ItemKind]float64{}
			var supplies, gears, tops float64
			for id := 1; id <= n; id++ {
				nb := 0
				for _, it := range caveWaysLoot(EntityID(id), kind, ring) {
					chance[it.Item] += 1.0 / n
					switch {
					case gear[it.Item]:
						gears++
					default:
						supplies++
					}
					if best[it.Item] || it.Item == ItemDiamond && it.Count >= 2 {
						nb++
					}
					if best[it.Item] || it.Item == ItemDiamond {
						tops++
					}
				}
				if nb > 1 {
					t.Fatalf("kind %d ring %d: chest %d holds %d of the best things at once", kind, ring, id, nb)
				}
			}
			supplies, gears, tops = supplies/n, gears/n, tops/n
			t.Logf("kind %d ring %d: %.2f supplies, %.2f gear; big potion %.3f rex tooth %.3f ivory spear %.3f obsidian %.3f jade %.3f agate %.3f",
				kind, ring, supplies, gears, chance[ItemBigPotion], chance[ItemRexTooth], chance[ItemIvorySpear], chance[ItemDiamond], chance[ItemIronOre], chance[ItemPlatinumOre])
			// Supplies first, everywhere.
			if supplies < 2*gears || supplies < 1.5 {
				t.Errorf("kind %d ring %d: %.2f supplies to %.2f gear", kind, ring, supplies, gears)
			}
			// Home: supplies only. Near: no obsidian.
			if ring == 0 && gears > 0 {
				t.Errorf("kind %d: a chest under the home island holds gear (%.3f a chest)", kind, gears)
			}
			if ring == 1 && chance[ItemDiamond] > 0 {
				t.Errorf("kind %d: obsidian under a near island (%.3f)", kind, chance[ItemDiamond])
			}
			// The best only in the deep hoard, and not too often there.
			bestP := chance[ItemBigPotion] + chance[ItemRexTooth] + chance[ItemIvorySpear]
			if kind != chestDeep && bestP > 0 {
				t.Errorf("kind %d ring %d: the best things in a chest off the deep hoard (%.3f)", kind, ring, bestP)
			}
			if kind != chestDeep && chance[ItemDiamond] > 0.1 {
				t.Errorf("kind %d ring %d: obsidian in %.3f of the chests", kind, ring, chance[ItemDiamond])
			}
			if kind == chestDeep {
				if bestP+chance[ItemDiamond] > caveBestChance[ring]+0.2 || ring > 0 && bestP < 0.15 {
					t.Errorf("ring %d: the best things in %.3f of the deep hoards", ring, bestP)
				}
				if tops > 0.7*oldTop {
					t.Errorf("ring %d: %.2f of the best things a deep hoard, %.2f before", ring, tops, oldTop)
				}
			}
			if ring == 2 && kind == chestDeep {
				for _, c := range []struct {
					item   ItemKind
					lo, hi float64
				}{{ItemBigPotion, 0.2, 0.33}, {ItemRexTooth, 0.08, 0.16}, {ItemIvorySpear, 0.05, 0.11}} {
					if p := chance[c.item]; p < c.lo || p > c.hi {
						t.Errorf("a far island's deep hoard holds %v in %.3f of them, want %.2f to %.2f", c.item, p, c.lo, c.hi)
					}
				}
			}
		}
	}
	t.Logf("before: %.2f of the best things to a chest at the end of the main route", oldTop)
}

// The chests of the older styles' caves are filled as they always were.
func TestCaveWaysOlderChestsAsBefore(t *testing.T) {
	old := genStyle(t, StyleFlint, 7, SizeMedium)
	n := 0
	for _, e := range old.entities {
		if e.Kind == EntityTreasure {
			n++
			if got, want := old.chestLoot(e), treasureLoot(e.ID, int(e.Var)); !sameLoot(got, want) {
				t.Fatalf("an old world's chest holds %v, it held %v", got, want)
			}
		}
	}
	if n == 0 {
		t.Fatal("no chests in an old world")
	}
	// A new world's cave chests by their kind and ring, the pyramid's by its
	// depth as before.
	w := genCaves(t, 42, SizeLarge)
	cave, above := 0, 0
	for _, e := range w.entities {
		if e.Kind != EntityTreasure {
			continue
		}
		want := treasureLoot(e.ID, int(e.Var))
		if w.caveLevel(e.TY) >= 0 {
			cave++
			want = caveWaysLoot(e.ID, int(e.Var>>2), int(e.Var&3))
		} else {
			above++
		}
		if got := w.chestLoot(e); !sameLoot(got, want) {
			t.Fatalf("chest at (%d,%d) holds %v, want %v", e.TX, e.TY, got, want)
		}
	}
	if cave == 0 {
		t.Fatal("no chests in the caves")
	}
	t.Logf("%d chests in the caves, %d above", cave, above)
}

func sameLoot(a, b []Ingredient) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
