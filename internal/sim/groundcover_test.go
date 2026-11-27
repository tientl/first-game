package sim

import (
	"math"
	"testing"
)

// coverCounts is how many of each kind of the ground cover a world has.
func coverCounts(w *World) map[EntityKind]int {
	n := map[EntityKind]int{}
	for _, e := range w.entities {
		n[e.Kind]++
	}
	return n
}

func TestGroundCoverIsTheSameEveryTime(t *testing.T) {
	a, b := genStyle(t, StyleGround, 7, SizeMedium), genStyle(t, StyleGround, 7, SizeMedium)
	if ha, hb := worldHash(a), worldHash(b); ha != hb {
		t.Fatalf("seed 7 made %#x, then %#x", ha, hb)
	}
}

// TestGroundCoverClumps checks that tall grass grows in drifts with open
// ground between: counted in squares of 8 by 8 tiles of grass and woods, its
// variance is over twice its mean (about the mean when strewn at random, as
// in older worlds), where the same world of StyleWilds is near 1.
func TestGroundCoverClumps(t *testing.T) {
	for _, seed := range []uint64{7, 9} {
		old, cur := genStyle(t, StyleWilds, seed, SizeMedium), genStyle(t, StyleGround, seed, SizeMedium)
		vo, vn := quadratVMR(old, EntityGrassTuft, 8), quadratVMR(cur, EntityGrassTuft, 8)
		t.Logf("seed %d: tall grass variance/mean %.2f, was %.2f", seed, vn, vo)
		if vn < 2 || vn < 1.6*vo {
			t.Errorf("seed %d: tall grass variance/mean %.2f (was %.2f); want it clumped, over 2", seed, vn, vo)
		}
	}
	// Bushes are fewer: a bigger world to count them in.
	old, cur := genStyle(t, StyleWilds, 11, SizeLarge), genStyle(t, StyleGround, 11, SizeLarge)
	vo, vn := quadratVMR(old, EntityBerryBush, 8), quadratVMR(cur, EntityBerryBush, 8)
	t.Logf("berry bushes variance/mean %.2f, was %.2f", vn, vo)
	if vn < 1.8 || vn < 1.6*vo {
		t.Errorf("berry bushes variance/mean %.2f (was %.2f); want thickets, over 1.8", vn, vo)
	}
}

// TestGroundCoverKeepsTheResources checks that about as much grass, berries,
// wood and flowers grow as in older worlds, a little less at most.
func TestGroundCoverKeepsTheResources(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{7, SizeMedium}, {11, SizeLarge}} {
		old, cur := coverCounts(genStyle(t, StyleWilds, c.seed, c.size)), coverCounts(genStyle(t, StyleGround, c.seed, c.size))
		for _, k := range []struct {
			kind   EntityKind
			lo, hi float64
		}{{EntityGrassTuft, 0.85, 1.02}, {EntityBerryBush, 0.85, 1.05}, {EntityTwig, 0.8, 1.02}, {EntityFlower, 0.75, 1.1}, {EntityReed, 0.8, 1.05}} {
			r := float64(cur[k.kind]) / float64(old[k.kind])
			t.Logf("seed %d: %v %d, was %d (%.2f)", c.seed, k.kind, cur[k.kind], old[k.kind], r)
			if r < k.lo || r > k.hi {
				t.Errorf("seed %d: %d of kind %v, %.2f of the %d before; want %.2f to %.2f", c.seed, cur[k.kind], k.kind, r, old[k.kind], k.lo, k.hi)
			}
		}
	}
}

// TestGroundCoverStaysOnLand checks that no plant or twig stands in the
// water, on a cliff, a ramp or a ford.
func TestGroundCoverStaysOnLand(t *testing.T) {
	w := genStyle(t, StyleGround, 11, SizeLarge)
	for _, e := range w.entities {
		switch e.Kind {
		case EntityGrassTuft, EntityBerryBush, EntityTwig, EntityFlower, EntityReed, EntityMushroom:
		default:
			continue
		}
		i := e.TY*w.width + e.TX
		if t0 := w.tiles[i]; !t0.Walkable() || t0 == TileCliff || w.crossing(i) {
			t.Fatalf("%v at (%d, %d) on tile %d (crossing %v)", e.Kind, e.TX, e.TY, t0, w.crossing(i))
		}
	}
}

// TestWormsKeepToDampSoil checks that a StyleGround world has far fewer
// worms than before, each on damp soil, and that rain brings more up round
// a player, which go back down out of sight some time after it stops.
func TestWormsKeepToDampSoil(t *testing.T) {
	worms := func(w *World) (n int, dry int) {
		for _, c := range w.fauna.creatures {
			if c.Kind == CreatureWorm {
				n++
				if !w.dampSoil(int(c.X), int(c.Y)) {
					dry++
				}
			}
		}
		return n, dry
	}
	old, w := genStyle(t, StyleWilds, 7, SizeMedium), genStyle(t, StyleGround, 7, SizeMedium)
	no, _ := worms(old)
	n, dry := worms(w)
	t.Logf("worms: %d, were %d", n, no)
	if n == 0 || float64(n) > 0.6*float64(no) || dry > 0 {
		t.Errorf("%d worms (%d on dry ground), %d before; want fewer than 60%%, all on damp soil", n, dry, no)
	}

	id := w.AddPlayer("a")
	p := w.players[id]
	w.DevForceWeather(WeatherRain)
	stepN(w, 30*TicksPerSecond)
	up := 0
	for _, wid := range w.fauna.rain.ids {
		if c := w.creatureByID(wid); c != nil && math.Hypot(c.X-p.X, c.Y-p.Y) <= rainWormRange+1 {
			up++
		}
	}
	if up == 0 || up > rainWormsNear {
		t.Fatalf("%d worms came up round the player in the rain; want 1 to %d", up, rainWormsNear)
	}
	w.DevForceWeather(WeatherClear)
	stepN(w, rainWormLinger/2)
	if len(w.fauna.rain.ids) == 0 {
		t.Fatal("the worms went down as soon as the rain stopped; want them to stay up a while")
	}
	p.X, p.Y = w.spawnX+40, w.spawnY // far away: nobody sees them go
	for range 40 {
		if !w.standable(p.X, p.Y) {
			p.X--
		}
	}
	stepN(w, rainWormLinger)
	if len(w.fauna.rain.ids) != 0 {
		t.Errorf("%d rain worms still up long after the rain", len(w.fauna.rain.ids))
	}
}

// quadratVMR is the variance-to-mean ratio of how many entities of kind k
// stand in each square of n by n tiles that is all grass or woods: about 1
// when they are spread at random, more when they clump.
func quadratVMR(w *World, k EntityKind, n int) float64 {
	W, H := w.width, w.SurfaceHeight()
	var counts []float64
	for y0 := 0; y0+n <= H; y0 += n {
		for x0 := 0; x0+n <= W; x0 += n {
			c, ok := 0.0, true
			for y := y0; y < y0+n && ok; y++ {
				for x := x0; x < x0+n; x++ {
					if t := w.tiles[y*W+x]; t != TileGrass && t != TileForest {
						ok = false
						break
					}
					if e := w.entityAt(x, y); e != nil && e.Kind == k {
						c++
					}
				}
			}
			if ok {
				counts = append(counts, c)
			}
		}
	}
	if len(counts) < 2 {
		return 0
	}
	mean := 0.0
	for _, c := range counts {
		mean += c
	}
	mean /= float64(len(counts))
	v := 0.0
	for _, c := range counts {
		v += (c - mean) * (c - mean)
	}
	v /= float64(len(counts) - 1)
	if mean == 0 {
		return 0
	}
	return v / mean
}
