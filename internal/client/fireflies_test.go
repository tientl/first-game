package client

import (
	"math"
	"testing"

	"survivalgame/internal/sim"
)

// countFireflies counts the fireflies out over a 64x64 field of t at day
// fraction f.
func countFireflies(t sim.Tile, f float64) int {
	n := 0
	for ty := 0; ty < 64; ty++ {
		for tx := 0; tx < 64; tx++ {
			if _, ok := fireflyAt(tx, ty, t, f, 100); ok {
				n++
			}
		}
	}
	return n
}

func TestFirefliesComeOutInTheEvening(t *testing.T) {
	for _, f := range []float64{0, 0.02, 0.3, sim.DuskStart, 0.995} {
		if n := countFireflies(sim.TileGrass, f); n != 0 {
			t.Errorf("%d fireflies at day fraction %.3f, want none", n, f)
		}
	}
	evening := countFireflies(sim.TileGrass, sim.NightStart+0.06)
	late := countFireflies(sim.TileGrass, 0.95)
	if evening < 64*64/50 {
		t.Errorf("%d fireflies over a 64x64 meadow in the evening, want plenty", evening)
	}
	if late >= evening/2 {
		t.Errorf("%d fireflies late at night against %d in the evening, want them thinning out", late, evening)
	}
}

func TestFirefliesLikeWetAndWoodedGround(t *testing.T) {
	f := sim.NightStart + 0.06
	grass, forest, marsh := countFireflies(sim.TileGrass, f), countFireflies(sim.TileForest, f), countFireflies(sim.TileMarsh, f)
	if !(grass < forest && forest < marsh) {
		t.Errorf("fireflies: grass %d, forest %d, marsh %d; want more in woods and the most in marsh", grass, forest, marsh)
	}
	for _, tile := range []sim.Tile{sim.TileWater, sim.TileSand, sim.TileDesert, sim.TileMountain, sim.TileCave, sim.TileAsh, sim.TileLava} {
		if n := countFireflies(tile, f); n != 0 {
			t.Errorf("%d fireflies over tile %d, want none", n, tile)
		}
	}
}

func TestFirefliesDriftNearHomeAndBlink(t *testing.T) {
	f := sim.NightStart + 0.06
	checked := 0
	for tx := 0; tx < 64 && checked < 10; tx++ {
		if _, ok := fireflyAt(tx, 7, sim.TileMarsh, f, 0); !ok {
			continue
		}
		checked++
		lo, hi := math.Inf(1), 0.0
		var x0, y0 float64
		for i := 0; i < 600; i++ { // ten seconds at 60 frames a second
			secs := float64(i) / 60
			ff, _ := fireflyAt(tx, 7, sim.TileMarsh, f, secs)
			if math.Abs(ff.x-float64(tx)-0.5) > 1.5 || math.Abs(ff.y-7.5) > 1.5 {
				t.Fatalf("firefly of tile %d strayed to (%.2f, %.2f)", tx, ff.x, ff.y)
			}
			if i == 0 {
				x0, y0 = ff.x, ff.y
			}
			if i == 599 && math.Hypot(ff.x-x0, ff.y-y0) < 0.01 {
				t.Errorf("firefly of tile %d did not move in ten seconds", tx)
			}
			lo, hi = min(lo, ff.glow), max(hi, ff.glow)
		}
		if hi < 0.9 || lo > 0.15 {
			t.Errorf("firefly of tile %d glows between %.2f and %.2f in ten seconds, want it to flash", tx, lo, hi)
		}
	}
	if checked == 0 {
		t.Fatal("no firefly over a marsh row to check")
	}
}
