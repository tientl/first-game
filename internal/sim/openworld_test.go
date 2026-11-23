package sim

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func genOpen(t *testing.T, seed uint64, size int) *World {
	t.Helper()
	w, err := Generate(GenConfig{Seed: seed, Width: size, Height: size, Style: StyleTribal})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// reachable floods walkable tiles from the spawn.
func reachable(w *World) []bool {
	seen := make([]bool, w.width*w.height)
	sx, sy := int(w.spawnX), int(w.spawnY)
	stack := []int{sy*w.width + sx}
	seen[stack[0]] = true
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := i%w.width, i/w.width
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.inBounds(nx, ny) || !w.TileAt(nx, ny).Walkable() {
				continue
			}
			if j := ny*w.width + nx; !seen[j] {
				seen[j] = true
				stack = append(stack, j)
			}
		}
	}
	return seen
}

func TestOpenWorldHasEveryBiome(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{1, SizeMedium}, {2, SizeMedium}, {3, SizeMedium}, {42, SizeMedium}, {1, SizeLarge}, {7, SizeLarge}} {
		w := genOpen(t, c.seed, c.size)
		var count [NumTiles]int
		for _, tl := range w.tiles {
			count[tl]++
		}
		for k := TileWater; k < NumTiles; k++ {
			if count[k] == 0 && k != TileIce { // ice is only the client's winter look
				t.Errorf("seed %d size %d: no tile %d", c.seed, c.size, k)
			}
		}

		if tl := w.TileAt(int(w.spawnX), int(w.spawnY)); tl != TileGrass && tl != TileForest {
			t.Errorf("seed %d size %d: spawn on tile %d", c.seed, c.size, tl)
		}

		// The volcano is far from the start.
		nearest := math.Inf(1)
		for i, tl := range w.tiles {
			if tl == TileLava {
				nearest = math.Min(nearest, math.Hypot(float64(i%w.width)-w.spawnX, float64(i/w.width)-w.spawnY))
			}
		}
		if nearest < float64(c.size)/5 {
			t.Errorf("seed %d size %d: lava %.0f tiles from the spawn", c.seed, c.size, nearest)
		}

		// From the spawn you can walk to the other biomes (some may be on
		// islands, but the main land must offer most of them, caves included).
		seen := reachable(w)
		var walkTo [NumTiles]int
		for i, ok := range seen {
			if ok {
				walkTo[w.tiles[i]]++
			}
		}
		for _, k := range []Tile{TileCave, TileDesert, TileMarsh, TileRocky} {
			if walkTo[k] == 0 {
				t.Errorf("seed %d size %d: cannot walk from the spawn to tile %d", c.seed, c.size, k)
			}
		}

		// The new resources are out there.
		var ents [NumEntityKinds]int
		w.EachEntityIn(0, 0, w.width, w.height, func(e Entity) { ents[e.Kind]++ })
		for _, k := range []EntityKind{EntityCactus, EntityReed, EntityMushroom, EntityCoal} {
			if ents[k] == 0 {
				t.Errorf("seed %d size %d: no %s", c.seed, c.size, k.Name())
			}
		}
	}
}

func TestOpenWorldIsDeterministicAndSaves(t *testing.T) {
	a, b := genOpen(t, 9, SizeMedium), genOpen(t, 9, SizeMedium)
	if !slices.Equal(a.tiles, b.tiles) || a.spawnX != b.spawnX || len(a.entities) != len(b.entities) {
		t.Fatal("two open worlds from one seed differ")
	}
	d := a.Save()
	if d.Style != StyleTribal {
		t.Fatalf("saved style %d, want %d", d.Style, StyleTribal)
	}
	c, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.tiles, c.tiles) || len(c.entities) != len(a.entities) {
		t.Fatalf("reloaded world differs: %d entities, want %d", len(c.entities), len(a.entities))
	}
}

func TestHugeOpenWorldIsQuickEnough(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	t0 := time.Now()
	w := genOpen(t, 1, SizeHuge)
	took := time.Since(t0)
	t.Logf("1024x1024: %v, %d entities, %d creatures", took, len(w.entities), len(w.Creatures()))
	if took > 8*time.Second {
		t.Errorf("generating took %v", took)
	}
	t0 = time.Now()
	for range 20 {
		w.Step()
	}
	t.Logf("20 ticks with no players: %v", time.Since(t0))
}

func TestMudSlowsAndLavaWarms(t *testing.T) {
	w := flatWorld()
	for x := 17; x < 32; x++ {
		w.tiles[16*w.width+x] = TileMarsh
	}
	id := w.AddPlayer("a")
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 20)
	if got := mustPlayer(t, w, id).X; got > 16.5+20*PlayerSpeed*0.8 {
		t.Errorf("walked to x %.2f through mud, too fast", got)
	}

	w.tiles[10*w.width+10] = TileLava
	if !w.warm(11.5, 10.5) || w.warm(15.5, 10.5) {
		t.Error("lava warmth wrong")
	}
	if TileLava.Walkable() || TileMountain.Walkable() || !TileCave.Walkable() {
		t.Error("walkability of the new tiles is wrong")
	}
}

func TestCoalFeedsTheFire(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	fire := w.addEntity(EntityCampfire, 17, 16)
	w.players[id].Inv = InventoryWith(Ingredient{ItemCoal, 1}, Ingredient{ItemWood, 1})
	before := fire.Fuel
	w.SetInput(id, Input{Action: true})
	w.Step()
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemCoal) != 0 || p.Inv.Count(ItemWood) != 1 || fire.Fuel != before+CoalFuel-1 {
		t.Fatalf("coal %d wood %d fuel +%d, want the coal burnt first for %d", p.Inv.Count(ItemCoal), p.Inv.Count(ItemWood), fire.Fuel-before, CoalFuel)
	}
}

// TestDumpOpenWorld writes the open worlds as PNG maps when OPEN_DUMP names
// a directory: go test ./internal/sim -run DumpOpen with OPEN_DUMP=dir.
func TestDumpOpenWorld(t *testing.T) {
	dir := os.Getenv("OPEN_DUMP")
	if dir == "" {
		t.Skip("set OPEN_DUMP to a directory")
	}
	cols := [NumTiles]color.RGBA{
		TileWater: {40, 90, 170, 255}, TileSand: {226, 206, 150, 255}, TileGrass: {98, 160, 72, 255},
		TileForest: {46, 100, 52, 255}, TileRocky: {140, 132, 118, 255}, TileDesert: {232, 176, 92, 255},
		TileMarsh: {84, 96, 58, 255}, TileMountain: {82, 78, 74, 255}, TileCave: {30, 26, 30, 255},
		TileAsh: {60, 56, 58, 255}, TileLava: {255, 90, 20, 255}, TileSnow: {240, 244, 250, 255}, TilePath: {150, 110, 70, 255},
	}
	for _, c := range []struct {
		seed uint64
		size int
	}{{1, SizeMedium}, {1, SizeLarge}, {7, SizeLarge}} {
		w, err := Generate(GenConfig{Seed: c.seed, Width: c.size, Height: c.size, Style: StyleLatest})
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rect(0, 0, w.width, w.height))
		for i, tl := range w.tiles {
			img.Set(i%w.width, i/w.width, cols[tl])
		}
		sx, sy := int(w.spawnX), int(w.spawnY)
		for d := -2; d <= 2; d++ {
			img.Set(sx+d, sy, color.RGBA{255, 0, 255, 255})
			img.Set(sx, sy+d, color.RGBA{255, 0, 255, 255})
		}
		f, err := os.Create(filepath.Join(dir, "open_"+itoa(int(c.seed))+"_"+itoa(c.size)+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, img)
		f.Close()
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	s := ""
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	return s
}

func TestClamsLieByTheSea(t *testing.T) {
	w := genOpen(t, 1, SizeMedium)
	n := 0
	w.EachEntityIn(0, 0, w.width, w.height, func(e Entity) {
		if e.Kind != EntityClam {
			return
		}
		n++
		if !w.seaside(e.TX, e.TY) {
			t.Errorf("clams at %d,%d are not on sand by the water", e.TX, e.TY)
		}
	})
	if n < 20 {
		t.Fatalf("%d clam beds on a 256 world, want plenty", n)
	}

	fw := flatWorld()
	id := fw.AddPlayer("a")
	clam := fw.addEntity(EntityClam, 17, 16)
	fw.SetInput(id, Input{Action: true})
	fw.Step()
	if p := mustPlayer(t, fw, id); p.Inv.Count(ItemClam) != ClamYield || clam.Ready {
		t.Fatalf("clams %d, bed ready %v: want them picked by hand", p.Inv.Count(ItemClam), clam.Ready)
	}
	fw.SetInput(id, Input{})
	fw.addEntity(EntityCampfire, 15, 16)
	fw.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeRoastClam})
	if p := mustPlayer(t, fw, id); p.Inv.Count(ItemRoastClam) != 1 || p.Inv.Count(ItemClam) != 0 {
		t.Fatalf("roasting: %d roast, %d raw", p.Inv.Count(ItemRoastClam), p.Inv.Count(ItemClam))
	}
}

func TestCavesOpenOnlyToTheSouth(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3} {
		w := genOpen(t, seed, SizeLarge)
		W := w.width
		seen := make([]bool, len(w.tiles))
		systems, mouths := 0, 0
		for i, tl := range w.tiles {
			if tl != TileCave {
				continue
			}
			x, y := i%W, i/W
			if w.caveOutside(x-1, y) || w.caveOutside(x+1, y) || w.caveOutside(x, y-1) {
				t.Fatalf("seed %d: cave at %d,%d opens sideways or north", seed, x, y)
			}
			if seen[i] {
				continue
			}
			// Walk this cave system and look for a mouth.
			systems++
			stack, found := []int{i}, false
			seen[i] = true
			for len(stack) > 0 {
				j := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				jx, jy := j%W, j/W
				if w.caveOutside(jx, jy+1) {
					found = true
				}
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := jx+d[0], jy+d[1]
					if w.inBounds(nx, ny) && !seen[ny*W+nx] && w.TileAt(nx, ny) == TileCave {
						seen[ny*W+nx] = true
						stack = append(stack, ny*W+nx)
					}
				}
			}
			if found {
				mouths++
			}
		}
		if systems == 0 || mouths != systems {
			t.Errorf("seed %d: %d cave systems, %d with a mouth", seed, systems, mouths)
		}
	}
}

func TestNewWorldsHaveBiggerDesertsSwampsAndBeaches(t *testing.T) {
	count := func(style WorldStyle) (desert, marsh, sand int) {
		w, err := Generate(GenConfig{Seed: 4, Width: SizeLarge, Height: SizeLarge, Style: style})
		if err != nil {
			t.Fatal(err)
		}
		for _, tl := range w.tiles {
			switch tl {
			case TileDesert:
				desert++
			case TileMarsh:
				marsh++
			case TileSand:
				sand++
			}
		}
		return
	}
	od, om, os := count(StyleOpenV2)
	nd, nm, ns := count(StyleOpen)
	if nd < od*5/4 || nm < om*3/2 || ns < os*3/2 {
		t.Errorf("desert %d→%d, marsh %d→%d, sand %d→%d: want each clearly larger", od, nd, om, nm, os, ns)
	}
}
