package sim

import (
	"math"
	"slices"
	"testing"
)

// The view of a 1080p screen at the default zoom, in tiles: 640 by 360 art
// pixels, a tile 24 by 18 of them.
const viewW, viewH = 640.0 / 24, 360.0 / 18

// canopyShare is the share of a 1080p view centred on (cx, cy) that the
// crowns of trees and other tall things cover: each drawn as an ellipse of
// its crown's width and area, hanging from the top of its drawing.
func canopyShare(w *World, cx, cy float64) float64 {
	cell := 0.25
	nx, ny := int(math.Ceil(viewW/cell)), int(math.Ceil(viewH/cell))
	cover := make([]bool, nx*ny)
	x0, y0 := cx-viewW/2, cy-viewH/2
	w.EachEntityIn(int(x0)-5, int(y0)-2, int(x0+viewW)+5, int(y0+viewH)+14, func(e Entity) {
		if e.Kind == EntityTree && !e.Ready {
			return // a stump
		}
		c, ok := w.crownOf(e.Kind, e.TX, e.TY, e.Big)
		if !ok {
			return
		}
		ch := c.area / (math.Pi / 4 * c.w)
		ex, ey := float64(e.TX)+0.5, float64(e.TY)+0.8-c.h+ch/2
		for j := 0; j < ny; j++ {
			y := y0 + (float64(j)+0.5)*cell
			dy := (y - ey) / (ch / 2)
			if dy*dy >= 1 {
				continue
			}
			half := c.w / 2 * math.Sqrt(1-dy*dy)
			for i := max(0, int((ex-half-x0)/cell)); i < nx; i++ {
				x := x0 + (float64(i)+0.5)*cell
				if x > ex+half {
					break
				}
				if x >= ex-half {
					cover[j*nx+i] = true
				}
			}
		}
	})
	n := 0
	for _, c := range cover {
		if c {
			n++
		}
	}
	return float64(n) / float64(len(cover))
}

// genSeed is the seed the generator's noise of w was made with.
func genSeed(w *World) uint32 { return uint32(w.seed) ^ uint32(w.seed>>32) }

// thickOf is how thick each surface tile's wood of w is (see woodsThickness).
func thickOf(w *World) []float32 {
	return w.woodsThickness(genSeed(w), int(w.spawnX), int(w.spawnY))
}

// woodsViews are the centres of views over w's woods, by what lies under
// the middle of the view: "forest" (an ordinary wood, well inside), "dense"
// (a thick wood, well inside) and "edge".
func woodsViews(w *World) map[string][][2]float64 {
	W, H := w.width, w.SurfaceHeight()
	depth := w.woodsDepth()
	thick := thickOf(w)
	out := map[string][][2]float64{}
	share := func(x, y int, ok func(i int) bool) float64 {
		n, in := 0, 0
		for ty := y - 9; ty <= y+9; ty += 3 {
			for tx := x - 12; tx <= x+12; tx += 3 {
				n++
				if tx >= 0 && ty >= 0 && tx < W && ty < H && ok(ty*W+tx) {
					in++
				}
			}
		}
		return float64(in) / float64(n)
	}
	wood := func(i int) bool { return w.tiles[i] == TileForest || w.tiles[i] == TilePath }
	dense := func(i int) bool { return wood(i) && thick[i] > 0.9 }
	ordinary := func(i int) bool { return w.tiles[i] == TilePath || w.tiles[i] == TileForest && thick[i] < 0.1 }
	for y := 12; y < H-12; y += 7 {
		for x := 14; x < W-14; x += 9 {
			i := y*W + x
			if !wood(i) {
				continue
			}
			switch {
			case dense(i) && depth[i] >= 4 && share(x, y, dense) > 0.75:
				out["dense"] = append(out["dense"], [2]float64{float64(x) + 0.5, float64(y) + 0.5})
			case depth[i] >= 5 && share(x, y, wood) > 0.92 && share(x, y, ordinary) > 0.85:
				out["forest"] = append(out["forest"], [2]float64{float64(x) + 0.5, float64(y) + 0.5})
			case depth[i] < 2 && share(x, y, wood) > 0.4 && thick[i] < 0.5:
				out["edge"] = append(out["edge"], [2]float64{float64(x) + 0.5, float64(y) + 0.5})
			}
		}
	}
	return out
}

// canopyStats is the mean and the 90th percentile of the canopy share over
// views.
func canopyStats(w *World, views [][2]float64) (mean, p90 float64) {
	if len(views) == 0 {
		return 0, 0
	}
	var s []float64
	for _, v := range views {
		s = append(s, canopyShare(w, v[0], v[1]))
	}
	slices.Sort(s)
	for _, v := range s {
		mean += v
	}
	return mean / float64(len(s)), s[len(s)*9/10]
}

// standingWood is the wood the trees, the giants and the fruit trees of w
// would give felled, and how many trees stand.
func standingWood(w *World) (wood, trees int) {
	for _, e := range w.entities {
		switch e.Kind {
		case EntityTree:
			trees++
			wood += bySize(e.Big, TreeYield, BigTreeYield)
		case EntityGiantTree:
			wood += GiantWood
		default:
			if f, ok := fruitOf(e.Kind); ok {
				wood += f.wood
			}
		}
	}
	return wood, trees
}

// woodWithin is the wood the trees and giants within r tiles of (x, y)
// would give.
func woodWithin(w *World, x, y, r float64) int {
	wood := 0
	w.EachEntityIn(int(x-r), int(y-r), int(x+r), int(y+r), func(e Entity) {
		if math.Hypot(float64(e.TX)+0.5-x, float64(e.TY)+0.5-y) > r {
			return
		}
		switch e.Kind {
		case EntityTree:
			wood += bySize(e.Big, TreeYield, BigTreeYield)
		case EntityGiantTree:
			wood += GiantWood
		}
	})
	return wood
}

// The spacing of each look follows its crown: the wide and the tall need 3
// to 4 tiles, the slim conifers about 2, tree ferns and palms less.
func TestCrownSpacing(t *testing.T) {
	for _, c := range []struct {
		name   string
		s      float64
		lo, hi float64
	}{
		{"big broadleaf", crowns[LookBroadleaf][1].spacing, 3, 4},
		{"big redwood", crowns[LookRedwood][1].spacing, 3, 4},
		{"big monkey-puzzle", crowns[LookAraucaria][1].spacing, 3, 4},
		{"pine", crowns[LookPine][1].spacing, 1.5, 2.5},
		{"cypress", crowns[LookSwamp][1].spacing, 1.5, 2.5},
		{"tree fern", crowns[LookTreeFern][1].spacing, 1, 1.8},
		{"palm", palmCrown.spacing, 1, 2},
	} {
		if c.s < c.lo || c.s > c.hi {
			t.Errorf("%s keeps %.1f tiles, want %.1f to %.1f", c.name, c.s, c.lo, c.hi)
		}
	}
	for l := range NumTreeLooks {
		for b := range 2 {
			if c := crowns[l][b]; c.spacing <= 0 || c.spacing > maxSpacing || c.w <= 0 || c.area <= 0 {
				t.Errorf("look %d size %d: crown %+v", l, b, c)
			}
		}
		if crowns[l][0].spacing > crowns[l][1].spacing {
			t.Errorf("look %d: a young tree keeps more room than a grown one", l)
		}
	}
}

// No two trees of a new world stand closer than their crowns allow (a dense
// grove lets them come a little closer), and nothing stands in a giant's.
func TestWoodsKeepTheirSpacing(t *testing.T) {
	for _, seed := range []uint64{42, 7} {
		w := genStyle(t, StyleWoods, seed, SizeLarge)
		var giants []*Entity
		for _, e := range w.entities {
			if e.Kind == EntityGiantTree {
				giants = append(giants, e)
			}
		}
		for _, e := range w.entities {
			c, ok := w.crownOf(e.Kind, e.TX, e.TY, e.Big)
			if !ok || e.Kind == EntityGiantTree {
				continue
			}
			for _, g := range giants {
				d := (giantCrown.spacing + c.spacing) / 2
				if math.Hypot(float64(g.TX-e.TX), float64(g.TY-e.TY)) < d {
					t.Fatalf("seed %d: %s at %d,%d stands in the crown of the giant at %d,%d", seed, e.Kind.Name(), e.TX, e.TY, g.TX, g.TY)
				}
			}
			if e.Kind != EntityTree {
				continue
			}
			w.EachEntityIn(e.TX-5, e.TY-5, e.TX+5, e.TY+5, func(o Entity) {
				if o.ID == e.ID || o.Kind == EntityGiantTree {
					return
				}
				oc, ok := w.crownOf(o.Kind, o.TX, o.TY, o.Big)
				if !ok {
					return
				}
				d := (c.spacing + oc.spacing) / 2 * thickCloser
				if dist := math.Hypot(float64(o.TX-e.TX), float64(o.TY-e.TY)); dist < d-1e-9 {
					t.Fatalf("seed %d: trees at %d,%d and %d,%d stand %.2f apart, want at least %.2f", seed, e.TX, e.TY, o.TX, o.TY, dist, d)
				}
			})
		}
	}
}

// The woods read as woods but are never a wall: on a 1080p view the crowns
// cover about 35 to 45% of an ordinary wood, less at its edge, 60 to 70% of
// a thick one. The woods of the older worlds cover nearly all.
//
// The crowns are measured here on trees, fruit trees and giants; the game's
// own measure (cmd/shot -canopy) counts every tall drawing, bamboo and the
// like with them, and comes out 5 to 10 points higher. These bounds are on
// the trees alone.
func TestWoodsCanopyShare(t *testing.T) {
	thickSeen := false
	for _, c := range []struct {
		seed uint64
		size int
	}{{42, SizeLarge}, {3, SizeHuge}} {
		w := genStyle(t, StyleWoods, c.seed, c.size)
		views := woodsViews(w)
		if len(views["forest"]) < 3 || len(views["edge"]) < 20 {
			t.Fatalf("seed %d: too few views: %d in the woods, %d at their edge", c.seed, len(views["forest"]), len(views["edge"]))
		}
		forest, fp90 := canopyStats(w, views["forest"])
		edge, _ := canopyStats(w, views["edge"])
		dense, dp90 := canopyStats(w, views["dense"])
		t.Logf("seed %d: canopy %.0f%% at the edge, %.0f%% in ordinary woods (p90 %.0f%%), %.0f%% in %d views of thick ones (p90 %.0f%%)",
			c.seed, 100*edge, 100*forest, 100*fp90, 100*dense, len(views["dense"]), 100*dp90)
		if forest < 0.28 || forest > 0.4 {
			t.Errorf("seed %d: the crowns cover %.0f%% of a view in ordinary woods, want about 30 to 40%% (35 to 45%% in the game)", c.seed, 100*forest)
		}
		if edge >= forest {
			t.Errorf("seed %d: the edge of the woods (%.0f%%) is as thick as inside (%.0f%%)", c.seed, 100*edge, 100*forest)
		}
		if len(views["dense"]) >= 10 {
			thickSeen = true
			if dense < 0.52 || dense > 0.66 {
				t.Errorf("seed %d: the thick woods cover %.0f%%, want 55 to 65%% (60 to 70%% in the game)", c.seed, 100*dense)
			}
		}
		for _, k := range []string{"forest", "dense", "edge"} {
			for _, v := range views[k] {
				if s := canopyShare(w, v[0], v[1]); s > 0.8 {
					t.Errorf("seed %d: a wall of leaves at %.0f,%.0f: %.0f%%", c.seed, v[0], v[1], 100*s)
				}
			}
		}
	}
	if !thickSeen {
		t.Error("no world with views well inside thick woods")
	}
	old := genStyle(t, StyleIsles, 42, SizeLarge)
	if m, _ := canopyStats(old, woodsViews(old)["forest"]); m < 0.85 {
		t.Errorf("the woods of the older islands cover %.0f%%: the measure does not see the wall", 100*m)
	}
}

// Every larger wood has a game trail through it, and no tree stands on a
// trail or right beside one.
func TestTrailsThroughTheWoods(t *testing.T) {
	w := genStyle(t, StyleWoods, 7, SizeLarge)
	W, H := w.width, w.SurfaceHeight()
	seen := make([]bool, W*H)
	woods, trailed := 0, 0
	for start := range W * H {
		if seen[start] || w.tiles[start] != TileForest {
			continue
		}
		seen[start] = true
		q := []int{start}
		n, path := 0, false
		for k := 0; k < len(q); k++ {
			x, y := q[k]%W, q[k]/W
			if w.tiles[q[k]] == TileForest {
				n++
			} else {
				path = true
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; !seen[j] && (w.tiles[j] == TileForest || w.tiles[j] == TilePath) {
					seen[j] = true
					q = append(q, j)
				}
			}
		}
		if n >= trailMinWood {
			woods++
			if path {
				trailed++
			}
		}
	}
	if woods == 0 || trailed < woods {
		t.Errorf("%d of %d larger woods have a trail", trailed, woods)
	}
	for _, e := range w.entities {
		if e.Kind != EntityTree {
			continue
		}
		for y := e.TY - trailClear; y <= e.TY+trailClear; y++ {
			for x := e.TX - trailClear; x <= e.TX+trailClear; x++ {
				if w.TileAt(x, y) == TilePath {
					t.Fatalf("a tree at %d,%d stands on a trail", e.TX, e.TY)
				}
			}
		}
	}
}

// The floor between the trunks has its undergrowth: tall grass, bushes,
// fallen branches, flowers and mushrooms on rotting logs.
func TestWoodsHaveUndergrowth(t *testing.T) {
	w := genStyle(t, StyleWoods, 42, SizeLarge)
	free, under := 0, map[EntityKind]int{}
	for i, tl := range w.tiles {
		if tl != TileForest {
			continue
		}
		e := w.entityAt(i%w.width, i/w.width)
		switch {
		case e == nil:
			free++
		case e.Kind == EntityGrassTuft, e.Kind == EntityBerryBush, e.Kind == EntityTwig, e.Kind == EntityFlower, e.Kind == EntityMushroom:
			under[e.Kind]++
		}
	}
	total := 0
	for _, n := range under {
		total += n
	}
	if share := float64(total) / float64(total+free); share < 0.1 || share > 0.3 {
		t.Errorf("undergrowth on %.0f%% of the woods' open floor, want 10 to 30%%", 100*share)
	}
	for _, k := range []EntityKind{EntityGrassTuft, EntityBerryBush, EntityTwig, EntityFlower, EntityMushroom} {
		if under[k] == 0 {
			t.Errorf("no %s in the woods", k.Name())
		}
	}
}

func TestWoodsAreDeterministic(t *testing.T) {
	a, b := genStyle(t, StyleWoods, 5, SizeMedium), genStyle(t, StyleWoods, 5, SizeMedium)
	if worldHash(a) != worldHash(b) {
		t.Fatal("two worlds of one seed differ")
	}
}

// Fewer trees, each giving more: the wood of a new world stays in reach. A
// new player finds wood for the first tools, a fire and a shelter close by,
// at least as much as when the woods were first spaced (before they had
// thick and ordinary stretches), a screen of woods holds plenty, and the
// woods as a whole hold a good share of what the old walls of trees gave (at
// what a tree gave then: 2, a big one 6).
func TestWoodStaysInReach(t *testing.T) {
	const oldYield, oldBigYield = 2, 6
	nearBefore := map[uint64]int{42: 369, 7: 324} // within 30 tiles of the spawn, when the woods were first spaced
	for _, seed := range []uint64{42, 7} {
		w := genStyle(t, StyleWoods, seed, SizeLarge)
		old := genStyle(t, StyleIsles, seed, SizeLarge)
		wood, trees := standingWood(w)
		oldWood, oldTrees := 0, 0
		for _, e := range old.entities {
			switch e.Kind {
			case EntityTree:
				oldTrees++
				oldWood += bySize(e.Big, oldYield, oldBigYield)
			case EntityGiantTree:
				oldWood += GiantWood
			default:
				if f, ok := fruitOf(e.Kind); ok {
					oldWood += f.wood
				}
			}
		}
		near := woodWithin(w, w.spawnX, w.spawnY, 30)
		least := math.MaxInt
		for _, v := range woodsViews(w)["forest"] {
			least = min(least, woodWithin(w, v[0], v[1], 10))
		}
		t.Logf("seed %d: %d trees with %d wood (the old woods: %d trees, %d wood); %d wood within 30 tiles of the spawn; at least %d within 10 tiles in the woods",
			seed, trees, wood, oldTrees, oldWood, near, least)
		if r := float64(wood) / float64(oldWood); r < 0.2 || r > 1 {
			t.Errorf("seed %d: the woods hold %.0f%% of the wood of the old ones", seed, 100*r)
		}
		if near < nearBefore[seed] {
			t.Errorf("seed %d: only %d wood within 30 tiles of the spawn, %d before", seed, near, nearBefore[seed])
		}
		if least < 40 {
			t.Errorf("seed %d: a spot in the woods has only %d wood within 10 tiles", seed, least)
		}
	}
	if TreeYield < 3 || BigTreeYield < 10 {
		t.Errorf("a tree gives %d wood and a big one %d: the spaced woods need more from each", TreeYield, BigTreeYield)
	}
}
