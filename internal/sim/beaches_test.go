package sim

import (
	"math"
	"slices"
	"testing"
)

// islandOf is, for every land tile of the surface, the island it is on as
// Islands lists them; nil for the bits of land too small to be listed.
func islandOf(w *World) []*Island {
	out := make([]*Island, len(w.tiles))
	k := 0
	for _, isl := range w.landMasses() {
		if len(isl) < seasMinIsland || isl[0]/w.width >= w.SurfaceHeight() || k >= len(w.islands) {
			continue
		}
		for _, i := range isl {
			out[i] = &w.islands[k]
		}
		k++
	}
	return out
}

// typicalBeaches is the depth of the beach at every tile of sand on the
// shore of the islands whose coasts are sandy, meadow, woods, the monkeys'
// crag and desert (marsh, snow and ash shores are narrow by design), big
// enough to be more than a beach and a bit of green (see beachWidths).
func typicalBeaches(w *World) []int {
	isl := islandOf(w)
	bd, _ := w.BeachDepths()
	var d []int
	for _, b := range bd {
		s := isl[b.Y*w.width+b.X]
		if s == nil || s.Tiles < 1000 {
			continue
		}
		switch s.Kind {
		case IslandMeadow, IslandForest, IslandMonkey, IslandDesert:
			d = append(d, b.N)
		}
	}
	slices.Sort(d)
	return d
}

// quantileOf is the value below which share of the sorted ds lie.
func quantileOf(ds []int, share float64) int {
	return ds[min(len(ds)-1, int(share*float64(len(ds))))]
}

func TestBeachesAreWide(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	for _, c := range []struct {
		seed uint64
		size int
	}{{42, SizeHuge}, {7, SizeLarge}, {3, SizeMedium}} {
		w := genStyle(t, StyleBeaches, c.seed, c.size)
		old := genStyle(t, StyleWoods, c.seed, c.size)
		d, od := typicalBeaches(w), typicalBeaches(old)
		if len(d) < 100 {
			t.Fatalf("seed %d size %d: only %d tiles of sandy shore", c.seed, c.size, len(d))
		}
		wide, mean := 0, 0.0
		for _, v := range d {
			if v >= 5 {
				wide++
			}
			mean += float64(v)
		}
		mean /= float64(len(d))
		sd := 0.0
		for _, v := range d {
			sd += (float64(v) - mean) * (float64(v) - mean)
		}
		sd = math.Sqrt(sd / float64(len(d)))
		share := float64(wide) / float64(len(d))
		t.Logf("seed %d size %d: %d shore tiles, depth p25 %d median %d p75 %d p90 %d, mean %.1f sd %.1f, %.0f%% at least 5 (before: median %d, p90 %d)",
			c.seed, c.size, len(d), quantileOf(d, 0.25), quantileOf(d, 0.5), quantileOf(d, 0.75), quantileOf(d, 0.9), mean, sd, 100*share, quantileOf(od, 0.5), quantileOf(od, 0.9))
		if m := quantileOf(d, 0.5); m < 5 || m > 9 {
			t.Errorf("seed %d size %d: the median beach is %d tiles deep, want 5 to 9", c.seed, c.size, m)
		}
		if share < 0.6 {
			t.Errorf("seed %d size %d: only %.0f%% of sandy shore has a beach 5 tiles deep or more", c.seed, c.size, 100*share)
		}
		if sd < 1.5 {
			t.Errorf("seed %d size %d: beaches %.1f tiles either way of their mean: one even band", c.seed, c.size, sd)
		}
		if quantileOf(od, 0.5) >= quantileOf(d, 0.5) {
			t.Errorf("seed %d size %d: beaches no wider than before", c.seed, c.size)
		}
	}
}

// The sand takes the land behind the waterline: the islands keep their
// shapes, their number, and what can be walked on.
func TestBeachesKeepTheLand(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{7, SizeLarge}, {3, SizeMedium}} {
		w := genStyle(t, StyleBeaches, c.seed, c.size)
		old := genStyle(t, StyleWoods, c.seed, c.size)
		count := func(w *World) (land, walk, sand int) {
			for _, t := range w.tiles[:w.width*w.SurfaceHeight()] {
				if !t.Sea() {
					land++
				}
				if t.Walkable() {
					walk++
				}
				if t == TileSand {
					sand++
				}
			}
			return
		}
		l, wk, s := count(w)
		ol, owk, os := count(old)
		t.Logf("seed %d size %d: land %d (before %d), walkable %d (%d), sand %d (%d), islands %d (%d)",
			c.seed, c.size, l, ol, wk, owk, s, os, len(w.Islands()), len(old.Islands()))
		if math.Abs(float64(l-ol)) > 0.005*float64(ol) {
			t.Errorf("seed %d size %d: %d tiles of land, %d before", c.seed, c.size, l, ol)
		}
		if float64(wk) < 0.99*float64(owk) {
			t.Errorf("seed %d size %d: %d walkable tiles, %d before", c.seed, c.size, wk, owk)
		}
		if len(w.Islands()) != len(old.Islands()) {
			t.Errorf("seed %d size %d: %d islands, %d before", c.seed, c.size, len(w.Islands()), len(old.Islands()))
		}
		if s < 3*os/2 {
			t.Errorf("seed %d size %d: %d tiles of sand, %d before", c.seed, c.size, s, os)
		}
	}
}

// seaCliffSand is how many cliffs stand over the sea, and how many of them
// have sand within two tiles (ramps cut down through them aside).
func seaCliffSand(w *World) (cliffs, sandy int) {
	W, H := w.width, w.SurfaceHeight()
	for y := 1; y < H-1; y++ {
		for x := 1; x < W-1; x++ {
			i := y*W + x
			if w.tiles[i] != TileCliff || w.relief[i].Variant() == VarStack {
				continue
			}
			sea := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				sea = sea || w.seaAt((y+d[1])*W+x+d[0])
			}
			if !sea {
				continue
			}
			cliffs++
		near:
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if j := (y+dy)*W + x + dx; w.TileAt(x+dx, y+dy) == TileSand && w.relief[j].Variant() != VarRamp {
						sandy++
						break near
					}
				}
			}
		}
	}
	return cliffs, sandy
}

// A cliff coast keeps no beach under it: the sand narrows to nothing as
// the shore comes up to one.
func TestNoSandUnderSeaCliffs(t *testing.T) {
	w := genStyle(t, StyleBeaches, 7, SizeLarge)
	c, s := seaCliffSand(w)
	oc, os := seaCliffSand(genStyle(t, StyleWoods, 7, SizeLarge))
	t.Logf("%d sea cliffs, %d with sand within two tiles (before: %d of %d)", c, s, os, oc)
	if c < 50 {
		t.Fatalf("only %d tiles of sea cliff", c)
	}
	if float64(s) > 0.03*float64(c) || s > os {
		t.Errorf("%d of %d sea cliffs have sand under them (before: %d of %d)", s, c, os, oc)
	}
}

func TestBeachesAreDeterministic(t *testing.T) {
	a, b := genStyle(t, StyleBeaches, 9, SizeMedium), genStyle(t, StyleBeaches, 9, SizeMedium)
	if worldHash(a) != worldHash(b) || !slices.Equal(a.relief, b.relief) {
		t.Fatal("the same seed made two different worlds")
	}
}

// The widest beaches have dunes at their back, grass on them and palms;
// driftwood lies along the tide line and clam beds at the water's edge.
func TestBeachesHaveDunesAndDriftwood(t *testing.T) {
	w := genStyle(t, StyleBeaches, 7, SizeLarge)
	toSea := w.seaDistance(16)
	dunes, duneDepth := 0, 0
	for i, tl := range w.tiles[:w.width*w.SurfaceHeight()] {
		if tl == TileSand && w.relief[i].Variant() == VarDunes {
			dunes++
			duneDepth += int(toSea[i])
		}
	}
	var drift, driftDepth, palms, palmFront, clams int
	for _, e := range w.entities {
		if w.TileAt(e.TX, e.TY) != TileSand {
			continue
		}
		d := int(toSea[e.TY*w.width+e.TX])
		switch e.Kind {
		case EntityTwig:
			drift++
			driftDepth += d
		case EntityPalm:
			palms++
			if d <= 2 {
				palmFront++
			}
		case EntityClam:
			clams++
		}
	}
	t.Logf("%d tiles of dunes (%.1f tiles from the sea on average), %d driftwood (%.1f), %d palms (%d by the water), %d clam beds",
		dunes, float64(duneDepth)/float64(max(dunes, 1)), drift, float64(driftDepth)/float64(max(drift, 1)), palms, palmFront, clams)
	if dunes < 100 || float64(duneDepth) < 4*float64(dunes) {
		t.Errorf("%d tiles of dunes, %.1f tiles from the sea on average: want them at the back of wide beaches", dunes, float64(duneDepth)/float64(max(dunes, 1)))
	}
	if drift < 30 || clams < 30 || palms < 30 {
		t.Errorf("%d driftwood, %d clam beds, %d palms on the beaches", drift, clams, palms)
	}
	if palmFront*4 > palms {
		t.Errorf("%d of %d palms stand by the water: they grow at the back of a beach", palmFront, palms)
	}
}
