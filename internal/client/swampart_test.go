package client

import (
	"slices"
	"testing"

	"survivalgame/internal/sim"
)

func TestSwampPondsAreTheSmallMarshWaters(t *testing.T) {
	// A 30x12 map: sea along the west edge, grass, a lake in the grass, and a
	// marsh with a pond in it.
	const w, h = 30, 12
	tiles := make([]sim.Tile, w*h)
	for y := range h {
		for x := range w {
			t := sim.TileGrass
			switch {
			case x < 3:
				t = sim.TileWater // the sea, running off the map
			case x >= 18:
				t = sim.TileMarsh
			}
			tiles[y*w+x] = t
		}
	}
	set := func(x0, y0, x1, y1 int, t sim.Tile) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				tiles[y*w+x] = t
			}
		}
	}
	set(7, 4, 10, 7, sim.TileWater)  // the lake
	set(22, 4, 25, 7, sim.TileWater) // the pond
	pond := swampPonds(w, h, tiles)
	at := func(x, y int) float32 { return pond[y*w+x] }

	if at(23, 5) != 1 || at(21, 5) != 1 || at(26, 8) != 1 {
		t.Error("the marsh pond, or its bank, is not marked")
	}
	if at(28, 1) != 0 {
		t.Error("marsh away from the pond is marked")
	}
	if at(1, 5) != 0 || at(8, 5) != 0 || at(6, 5) != 0 {
		t.Error("the sea or the lake in the grass is marked as a swamp pond")
	}
	terr := &terrain{w: w, h: h, tiles: tiles, pond: pond}
	if !terr.pondTile(23, 5) || terr.pondTile(21, 5) || terr.pondTile(8, 5) || terr.pondTile(-1, 5) {
		t.Error("pondTile is wrong")
	}
	if p := terr.pondAt(23.5, 5.5); p != 1 {
		t.Errorf("pondAt in the middle of the pond = %v, want 1", p)
	}
}

func TestFloatersStayOnTheirTile(t *testing.T) {
	var kinds [numFloaterKinds]int
	for ty := -30; ty < 30; ty++ {
		for tx := -30; tx < 30; tx++ {
			a := floatersOn(tx, ty, nil)
			if b := floatersOn(tx, ty, nil); !slices.Equal(a, b) {
				t.Fatalf("tile (%d, %d) floats %v, then %v", tx, ty, a, b)
			}
			for _, f := range a {
				if f.x < float64(tx) || f.x >= float64(tx+1) || f.y < float64(ty) || f.y >= float64(ty+1) || f.variant >= floaterVariants {
					t.Fatalf("a floater of tile (%d, %d): %+v", tx, ty, f)
				}
				kinds[f.kind]++
			}
		}
	}
	if kinds[floatLily] <= kinds[floatAlgae] || kinds[floatAlgae] <= kinds[floatStick] || kinds[floatStick] == 0 {
		t.Errorf("floaters: %v; want mostly lily pads, then algae, a few sticks", kinds)
	}
}
