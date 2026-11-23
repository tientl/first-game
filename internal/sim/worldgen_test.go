package sim

import (
	"os"
	"slices"
	"strings"
	"testing"
)

var testSeeds = []uint64{1, 5, 7, 42, 99, 31337, 123456789, 20261123}

func TestGenerateIsDeterministic(t *testing.T) {
	a, err := Generate(GenConfig{Seed: 42, Width: 96, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(GenConfig{Seed: 42, Width: 96, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.tiles, b.tiles) || !slices.Equal(a.occ, b.occ) {
		t.Fatal("same seed produced different worlds")
	}
	for id, e := range a.entities {
		if *b.entities[id] != *e {
			t.Fatalf("entity %d differs: %+v vs %+v", id, *e, *b.entities[id])
		}
	}
	ax, ay := a.Spawn()
	bx, by := b.Spawn()
	if ax != bx || ay != by {
		t.Fatalf("spawn differs: (%v,%v) vs (%v,%v)", ax, ay, bx, by)
	}

	c, err := Generate(GenConfig{Seed: 43, Width: 96, Height: 80})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(a.tiles, c.tiles) {
		t.Fatal("different seeds produced the same terrain")
	}
}

func TestGenerateRejectsBadSizes(t *testing.T) {
	for _, cfg := range []GenConfig{
		{Width: MinWorldSize - 1, Height: 64},
		{Width: 64, Height: 0},
		{Width: MaxWorldSize + 1, Height: 64},
	} {
		if _, err := Generate(cfg); err == nil {
			t.Errorf("Generate(%dx%d) = nil error, want error", cfg.Width, cfg.Height)
		}
	}
}

func TestGeneratedIslandsArePlayable(t *testing.T) {
	for _, seed := range testSeeds {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		land := 0
		for _, tile := range w.tiles {
			if tile.Walkable() {
				land++
			}
		}
		if frac := float64(land) / float64(len(w.tiles)); frac < 0.2 || frac > 0.7 {
			t.Errorf("seed %d: land fraction %.2f outside [0.2, 0.7]", seed, frac)
		}

		// The map edge must be sea so players cannot leave the island.
		for tx := 0; tx < w.width; tx++ {
			if w.TileAt(tx, 0).Walkable() || w.TileAt(tx, w.height-1).Walkable() {
				t.Fatalf("seed %d: land on the top/bottom edge", seed)
			}
		}
		for ty := 0; ty < w.height; ty++ {
			if w.TileAt(0, ty).Walkable() || w.TileAt(w.width-1, ty).Walkable() {
				t.Fatalf("seed %d: land on the left/right edge", seed)
			}
		}

		var kinds [NumEntityKinds]int
		for _, e := range w.entities {
			kinds[e.Kind]++
			if !w.TileAt(e.TX, e.TY).Walkable() {
				t.Fatalf("seed %d: %s on water at (%d,%d)", seed, e.Kind.Name(), e.TX, e.TY)
			}
			if w.occ[e.TY*w.width+e.TX] != e.ID {
				t.Fatalf("seed %d: occupancy grid out of sync for entity %d", seed, e.ID)
			}
		}
		for _, k := range []EntityKind{EntityTree, EntityBoulder, EntityBerryBush, EntityGrassTuft} {
			if kinds[k] < 30 {
				t.Errorf("seed %d: only %d %s", seed, kinds[k], k.Name())
			}
		}

		sx, sy := w.Spawn()
		if !w.standable(sx, sy) {
			t.Fatalf("seed %d: spawn (%v,%v) not standable", seed, sx, sy)
		}
		stx, sty := int(sx), int(sy)
		for ty := sty - clearSpawnRadius; ty <= sty+clearSpawnRadius; ty++ {
			for tx := stx - clearSpawnRadius; tx <= stx+clearSpawnRadius; tx++ {
				if _, ok := w.EntityAt(tx, ty); ok {
					t.Fatalf("seed %d: entity left at (%d,%d) next to spawn", seed, tx, ty)
				}
			}
		}
	}
}

// TestDumpWorld prints a generated island as ASCII (every other row) for
// tuning the generation thresholds by eye. Run with SIM_DUMP=1 go test -run Dump -v.
func TestDumpWorld(t *testing.T) {
	if os.Getenv("SIM_DUMP") == "" {
		t.Skip("set SIM_DUMP=1 to print a map")
	}
	tileCh := map[Tile]byte{TileWater: ' ', TileSand: '.', TileGrass: ',', TileForest: 'f', TileRocky: 'o'}
	entCh := map[EntityKind]byte{EntityTree: 'T', EntityBoulder: 'B', EntityBerryBush: '*', EntityGrassTuft: '"'}
	for _, seed := range testSeeds[:3] {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatal(err)
		}
		sx, sy := w.Spawn()
		var b strings.Builder
		for ty := 0; ty < w.height; ty += 2 {
			for tx := 0; tx < w.width; tx++ {
				ch := tileCh[w.TileAt(tx, ty)]
				if e, ok := w.EntityAt(tx, ty); ok {
					ch = entCh[e.Kind]
				}
				if tx == int(sx) && ty/2 == int(sy)/2 {
					ch = '@'
				}
				b.WriteByte(ch)
			}
			b.WriteByte('\n')
		}
		t.Logf("seed %d (%d entities)\n%s", seed, w.EntityCount(), b.String())
	}
}
