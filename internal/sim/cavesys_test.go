package sim

import (
	"hash/fnv"
	"math"
	"sync"
	"testing"
	"time"
)

// The natural caves of StyleCaves: see cavesys.go.

// caveStyle is the style the natural caves are tested in: as a new game
// makes them (see caveways.go). The caves of the older styles are held as
// they were by the golden digests (golden_test.go).
const caveStyle = StyleLatest

var caveWorlds struct {
	sync.Mutex
	m map[[2]uint64]*World
}

// genCaves is a world with natural caves, made once per seed and size for
// the tests that only look at it.
func genCaves(t *testing.T, seed uint64, size int) *World {
	t.Helper()
	caveWorlds.Lock()
	defer caveWorlds.Unlock()
	if caveWorlds.m == nil {
		caveWorlds.m = map[[2]uint64]*World{}
	}
	k := [2]uint64{seed, uint64(size)}
	if w := caveWorlds.m[k]; w != nil {
		return w
	}
	w := genStyle(t, caveStyle, seed, size)
	caveWorlds.m[k] = w
	return w
}

// caveWalk is how far every tile is from start, walking the caves: over
// ground that is not blocked, through the ways up and down; -1 for tiles
// not reached. Swimming is not walking.
func caveWalk(w *World, starts ...[2]int) []int32 {
	d := make([]int32, len(w.tiles))
	for i := range d {
		d[i] = -1
	}
	var q []int32
	for _, s := range starts {
		i := s[1]*w.width + s[0]
		d[i] = 0
		q = append(q, int32(i))
	}
	for n := 0; n < len(q); n++ {
		i := int(q[n])
		x, y := i%w.width, i/w.width
		next := [][2]int{{x + 1, y}, {x - 1, y}, {x, y + 1}, {x, y - 1}}
		if l, ok := w.links[[2]int{x, y}]; ok && w.Underground(y) && w.Underground(int(l.y)) {
			next = append(next, [2]int{int(l.x), int(l.y)})
		}
		for _, p := range next {
			if !w.inBounds(p[0], p[1]) || !w.TileAt(p[0], p[1]).Walkable() || w.blocked(p[0], p[1]) {
				continue
			}
			if j := p[1]*w.width + p[0]; d[j] < 0 {
				d[j] = d[i] + 1
				q = append(q, int32(j))
			}
		}
	}
	return d
}

// chamberTile is a floor tile of chamber c of system s: the open tile of
// its floor nearest its middle.
func chamberTile(w *World, s *caveSystem, c int) ([2]int, bool) {
	ch := s.chambers[c]
	best, bd := [2]int{}, math.Inf(1)
	for i, o := range s.owner[ch.level] {
		if int(o)-1 != c {
			continue
		}
		tx, ty := w.caveMapTile(s, ch.level, i)
		if d := math.Hypot(float64(tx)+0.5-ch.x, float64(ty-CaveBandTop(w.surfaceH, ch.level))+0.5-ch.y); d < bd && w.TileAt(tx, ty).Walkable() && !w.blocked(tx, ty) {
			best, bd = [2]int{tx, ty}, d
		}
	}
	return best, !math.IsInf(bd, 1)
}

func TestCavesEveryMouthReachesTheDestination(t *testing.T) {
	for _, seed := range []uint64{3, 42} {
		w := genCaves(t, seed, SizeLarge)
		if len(w.caves.systems) == 0 {
			t.Fatal("no cave systems")
		}
		for _, s := range w.caves.systems {
			dest, ok := chamberTile(w, s, s.dest)
			if !ok {
				t.Fatalf("seed %d island %d: the destination chamber has no floor", seed, s.island)
			}
			for _, e := range s.exits {
				d := caveWalk(w, [2]int{e[0], e[1] + 1})
				if d[dest[1]*w.width+dest[0]] < 0 {
					t.Fatalf("seed %d island %d: from the way up at %v the destination at %v cannot be walked to", seed, s.island, e, dest)
				}
				for c := range s.chambers {
					if p, ok := chamberTile(w, s, c); ok && d[p[1]*w.width+p[0]] < 0 {
						t.Fatalf("seed %d island %d: chamber %d at %v cannot be walked to", seed, s.island, c, p)
					}
				}
			}
			if s.levels == 2 && w.caveLevel(dest[1]) != 1 {
				t.Errorf("seed %d island %d: the destination of a system with two levels is on the upper one", seed, s.island)
			}
		}
	}
}

func TestCavesAreThreeTilesWideEverywhere(t *testing.T) {
	w := genCaves(t, 42, SizeLarge)
	open := func(x, y int) bool { return w.caveLevel(y) >= 0 && w.TileAt(x, y) != TileMountain }
	bad := 0
	for ty := CaveBandTop(w.surfaceH, 0); ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			if !open(tx, ty) {
				continue
			}
			ok := false
			for cy := ty - 1; cy <= ty+1 && !ok; cy++ {
				for cx := tx - 1; cx <= tx+1 && !ok; cx++ {
					full := true
					for y := cy - 1; y <= cy+1 && full; y++ {
						for x := cx - 1; x <= cx+1; x++ {
							if !open(x, y) {
								full = false
								break
							}
						}
					}
					ok = full
				}
			}
			if !ok {
				bad++
				if bad < 4 {
					t.Errorf("cave tile (%d,%d) lies in no open three-by-three square", tx, ty)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d cave tiles narrower than three", bad)
	}
}

func TestCavesGrowWithTheIsland(t *testing.T) {
	w := genCaves(t, 42, SizeHuge)
	type sz struct {
		area, chambers, floor int
		r                     float64
	}
	var small, big []sz
	for _, s := range w.caves.systems {
		n, r := 0, 0.0
		for _, c := range s.chambers {
			if !c.hall && !c.junction && c.pocket == 0 {
				n++
				r += c.r
			}
		}
		if n > 0 {
			r /= float64(n)
		}
		v := sz{s.area, n, s.floor[0] + s.floor[1], r}
		switch {
		case s.area >= 30000:
			big = append(big, v)
		case s.area < 8000:
			small = append(small, v)
		}
		// The caves are a share of the island, whatever its size.
		if share := float64(v.floor) / float64(s.area); s.area >= 1500 && (share < 0.08 || share > 0.7) {
			t.Errorf("island of %d tiles: %d tiles of cave, %.2f of it", s.area, v.floor, share)
		}
	}
	if len(big) == 0 || len(small) == 0 {
		t.Fatalf("%d big and %d small systems", len(big), len(small))
	}
	for _, b := range big {
		for _, s := range small {
			if b.chambers <= s.chambers || b.floor <= 2*s.floor || b.r < s.r {
				t.Errorf("island of %d tiles: %+v; one of %d: %+v", b.area, b, s.area, s)
			}
		}
		// Of mixed sizes since StyleCaveWays: great halls, and more small
		// grottos than before.
		if b.r < 8 || b.r > 20 {
			t.Errorf("island of %d tiles: chambers of %.1f tiles' radius on average", b.area, b.r)
		}
	}
}

func TestCavesHaveNoLongDeadEnds(t *testing.T) {
	for _, seed := range []uint64{3, 42} {
		w := genCaves(t, seed, SizeHuge)
		for _, s := range w.caves.systems {
			// No tunnel leads nowhere: every stretch of tunnel joins two
			// chambers at least.
			for lv := 0; lv < s.levels; lv++ {
				own := s.owner[lv]
				seen := make([]bool, len(own))
				for start, o := range own {
					if o != 0 || seen[start] || !s.open[lv][start] {
						continue
					}
					touches := map[int16]bool{}
					q := []int{start}
					seen[start] = true
					for n := 0; n < len(q); n++ {
						x, y := q[n]%s.gw, q[n]/s.gw
						for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
							nx, ny := x+d[0], y+d[1]
							if nx < 0 || ny < 0 || nx >= s.gw || ny >= s.gh {
								continue
							}
							j := ny*s.gw + nx
							if !s.open[lv][j] {
								continue
							}
							if own[j] != 0 {
								touches[own[j]] = true
							} else if !seen[j] {
								seen[j] = true
								q = append(q, j)
							}
						}
					}
					if len(touches) < 2 && s.reachFromChambers(lv, q) > 10 {
						x, y := w.caveMapTile(s, lv, start)
						t.Errorf("seed %d island %d: a tunnel of %d tiles at (%d,%d) leads to %d chambers", seed, s.island, len(q), x, y, len(touches))
					}
				}
			}
			// A side branch ends soon, in something worth the detour; a
			// side pocket, a dead end on purpose, sooner still.
			for i, c := range s.chambers {
				switch {
				case c.pocket != 0:
					if l := s.branchLen(i); len(c.adj) != 1 || l > caveSpurMost {
						t.Errorf("seed %d island %d: a side pocket at the end of %.0f tiles of spur, %d ways in", seed, s.island, l, len(c.adj))
					}
				case c.leaf:
					if l := s.branchLen(i); l > caveDeadEndMost {
						t.Errorf("seed %d island %d: a side branch of %.0f tiles of tunnel", seed, s.island, l)
					}
				}
			}
		}
	}
}

// caveDeadEndMost is the longest a side branch's tunnels may run, and
// caveSpurMost a side pocket's spur (measured from the edges of the round
// chambers it joins, so a little longer than it is dug).
const (
	caveDeadEndMost = 60
	caveSpurMost    = caveSpurMax + 8
)

func TestCavesDescentGoesDownAndBack(t *testing.T) {
	w := genCaves(t, 42, SizeLarge)
	var s *caveSystem
	for _, c := range w.caves.systems {
		if c.levels == 2 {
			s = c
			break
		}
	}
	if s == nil {
		t.Fatal("no system with a lower level")
	}
	id := w.AddPlayer("a")
	p := w.players[id]
	defer w.RemovePlayer(id)
	p.X, p.Y = float64(s.down[0])+0.5, float64(s.down[1])+1.6 // just south of the hole
	walk := func(to int) {                                    // north until p is on level to, then stop
		w.SetInput(id, Input{MoveY: -1})
		for range TicksPerSecond {
			if w.Step(); w.caveLevel(int(p.Y)) == to {
				break
			}
		}
		w.SetInput(id, Input{})
	}
	walk(1)
	if w.caveLevel(int(p.Y)) != 1 || !w.standable(p.X, p.Y) {
		t.Fatalf("at (%.1f,%.1f): walking into the hole did not go down to the lower level", p.X, p.Y)
	}
	if p.Notice != NoticeDeepCaves {
		t.Errorf("notice %d, want the deep caves'", p.Notice)
	}
	stepN(w, TravelCooldown+1)
	walk(0)
	if w.caveLevel(int(p.Y)) != 0 || math.Abs(p.X-float64(s.down[0])-0.5) > 1.5 {
		t.Fatalf("at (%.1f,%.1f): climbing the rope did not come back up by the hole", p.X, p.Y)
	}
}

func TestCavesOresLieByDepth(t *testing.T) {
	w := genCaves(t, 42, SizeHuge)
	var jade, deepJade, coal, upperCoal int
	for _, e := range w.entities {
		lv := w.caveLevel(e.TY)
		switch e.Kind {
		case EntityIronOre:
			jade++
			if lv == 1 {
				deepJade++
			}
		case EntityCoal:
			if lv >= 0 {
				coal++
				if lv == 0 {
					upperCoal++
				}
			}
		}
	}
	t.Logf("jade %d (%d deep), coal in the caves %d (%d above)", jade, deepJade, coal, upperCoal)
	if jade < 400 || jade > 3000 || deepJade != jade {
		t.Errorf("%d jade, %d of it on the lower level: want 400 to 3000, all deep", jade, deepJade)
	}
	if coal == 0 || upperCoal != coal {
		t.Errorf("%d coal in the caves, %d of it above", coal, upperCoal)
	}
}

func TestCavesSpidersKeepToTheirNests(t *testing.T) {
	w := genCaves(t, 42, SizeHuge)
	floor := 0
	for _, s := range w.caves.systems {
		floor += s.floor[0] + s.floor[1]
	}
	spiders, bears := 0, 0
	for _, c := range w.fauna.creatures {
		if c.Kind != CreatureSpider && c.Kind != CreatureBear {
			continue
		}
		want := ZoneNest
		if c.Kind == CreatureBear {
			want = ZoneDen
			bears++
		} else {
			spiders++
		}
		if z := w.caveCell(int(c.X), int(c.Y)).Zone(); z != want {
			t.Errorf("%s at (%.0f,%.0f) out of its home, in zone %d", c.Kind.def().name, c.X, c.Y, z)
		}
	}
	t.Logf("%d spiders and %d bears in %d tiles of cave", spiders, bears, floor)
	if spiders == 0 || bears == 0 {
		t.Fatalf("%d spiders, %d bears", spiders, bears)
	}
	if spiders*caveSpiderCap > floor {
		t.Errorf("%d spiders in %d tiles of cave: more than one to %d", spiders, floor, caveSpiderCap)
	}
	for _, i := range w.fauna.spawnAt[CreatureSpider] {
		if z := w.caveCell(int(i)%w.width, int(i)/w.width).Zone(); z != ZoneNest {
			t.Fatalf("spiders come back outside their nests, in zone %d", z)
		}
	}
}

// caveSpiderCap is the fewest tiles of cave to a spider.
const caveSpiderCap = 600

func TestCavesAreDeterministic(t *testing.T) {
	a := genStyle(t, caveStyle, 9, SizeLarge)
	b := genStyle(t, caveStyle, 9, SizeLarge)
	if worldHash(a) != worldHash(b) {
		t.Fatal("the same seed made two different worlds")
	}
	h := func(w *World) uint64 {
		f := fnv.New64a()
		for _, c := range w.caves.cells {
			f.Write([]byte{byte(c)})
		}
		return f.Sum64()
	}
	if h(a) != h(b) || len(a.links) != len(b.links) {
		t.Fatal("the same seed made two different cave systems")
	}
}

func TestCavesSaveAndLoad(t *testing.T) {
	w := genStyle(t, caveStyle, 5, SizeLarge)
	// A torch set in the caves burns on, through a save.
	var at [2]int
	found := false
	for _, s := range w.caves.systems {
		if p, ok := chamberTile(w, s, s.dest); ok {
			for dx := -3; dx <= 3 && !found; dx++ {
				if w.canPlace(p[0]+dx, p[1]+1) {
					at, found = [2]int{p[0] + dx, p[1] + 1}, true
				}
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("nowhere to set a torch")
	}
	e := w.addEntity(EntityTorchStand, at[0], at[1])
	e.Fuel = 40 // a torch nearly burnt down
	w.watch(e)
	stepN(w, 100)
	if got, ok := w.EntityAt(at[0], at[1]); !ok || got.Kind != EntityTorchStand || got.Fuel <= 0 {
		t.Fatal("a torch in the caves burnt out")
	}
	d := w.Save()
	if d.Version != SaveVersion || d.Style != caveStyle {
		t.Fatalf("saved version %d style %d", d.Version, d.Style)
	}
	l, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if l.height != w.height || len(l.links) != len(w.links) || len(l.caves.systems) != len(w.caves.systems) {
		t.Fatalf("loaded %d rows, %d links, %d systems; saved %d, %d, %d", l.height, len(l.links), len(l.caves.systems), w.height, len(w.links), len(w.caves.systems))
	}
	for i := range w.caves.cells {
		if l.caves.cells[i] != w.caves.cells[i] {
			t.Fatal("the caves came back different")
		}
	}
	if got, ok := l.EntityAt(at[0], at[1]); !ok || got.Kind != EntityTorchStand {
		t.Fatal("the torch was lost on loading")
	}
	if len(l.entities) != len(w.entities) {
		t.Errorf("%d entities loaded, %d saved", len(l.entities), len(w.entities))
	}
}

func TestCavesTickAtColossalSize(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	t0 := time.Now()
	w := genStyle(t, caveStyle, 1, SizeColossal)
	t.Logf("2048x2048 with its caves: %v, %d rows, %d systems, %d creatures", time.Since(t0), w.height, len(w.caves.systems), len(w.fauna.creatures))
	// Two players: one on the surface, one deep in the biggest system.
	var big *caveSystem
	for _, s := range w.caves.systems {
		if s.levels == 2 && (big == nil || s.area > big.area) {
			big = s
		}
	}
	w.AddPlayer("a")
	id := w.AddPlayer("b")
	p := w.players[id]
	p.X, p.Y = float64(big.up[0])+0.5, float64(big.up[1])+1.5
	best := time.Duration(1<<63 - 1)
	for range 3 {
		t0 = time.Now()
		for range 20 {
			w.Step()
		}
		best = min(best, time.Since(t0)/20)
	}
	t.Logf("a tick: %v", best)
	if best > 40*time.Millisecond {
		t.Errorf("a tick takes %v", best)
	}
}

func TestCavesOlderStylesHaveNone(t *testing.T) {
	w := genStyle(t, StyleWilds, 7, SizeMedium)
	if w.caves != nil || w.CaveCells() != nil || CaveLevelsOf(w.height, w.surfaceH) != 1 {
		t.Fatal("a world of an older style has natural caves")
	}
}

// The mouths of the natural caves keep the rules of the older ones.
func TestCavesMouthsLeadDownAndOpenSouth(t *testing.T) {
	testCaveMouths(t, genStyle(t, caveStyle, 2, SizeLarge))
	testMouthsOpenSouth(t, genStyle(t, caveStyle, 2, SizeLarge))
}

// reachFromChambers is how far into the tunnel cells cells (of level lv's
// grid) a walker gets from the chambers beside them: a bulge of a tunnel
// beside a chamber is not a way leading off.
func (s *caveSystem) reachFromChambers(lv int, cells []int) int {
	in := make(map[int]int, len(cells))
	for _, c := range cells {
		in[c] = -1
	}
	var q []int
	for _, c := range cells {
		x, y := c%s.gw, c/s.gw
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if j := (y+d[1])*s.gw + x + d[0]; s.owner[lv][j] != 0 {
				in[c] = 0
				q = append(q, c)
				break
			}
		}
	}
	most := 0
	for n := 0; n < len(q); n++ {
		x, y := q[n]%s.gw, q[n]/s.gw
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			j := (y+d[1])*s.gw + x + d[0]
			if v, ok := in[j]; ok && v < 0 {
				in[j] = in[q[n]] + 1
				most = max(most, in[j])
				q = append(q, j)
			}
		}
	}
	return most
}
