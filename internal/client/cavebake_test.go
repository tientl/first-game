package client

import (
	"math"
	"sync"
	"testing"

	"survivalgame/internal/sim"
)

var caveTestWorld struct {
	sync.Once
	terr *terrain
	w    *sim.World
}

// caveTestTerrain is the terrain of a small world with natural caves.
func caveTestTerrain(t testing.TB) (*terrain, *sim.World) {
	caveTestWorld.Do(func() {
		w, err := sim.Generate(sim.GenConfig{Seed: 42, Width: sim.SizeLarge, Height: sim.SizeLarge, Style: sim.StyleCaves})
		if err != nil {
			t.Fatal(err)
		}
		W, H := w.Size()
		terr := newTerrain(W, H, w.Tiles())
		terr.setRelief(w.Relief())
		terr.under = w.SurfaceHeight()
		terr.setCaves(w.CaveCells(), w.SurfaceHeight())
		caveTestWorld.terr, caveTestWorld.w = terr, w
	})
	return caveTestWorld.terr, caveTestWorld.w
}

// caveChunkWith finds a chunk of the caves with about the share of open
// floor wanted.
func caveChunkWith(terr *terrain, open float64) chunkKey {
	best, bestD := chunkKey{}, 2.0
	for cy := terr.cave.from / chunkTiles; cy < terr.h/chunkTiles; cy++ {
		for cx := 0; cx < terr.w/chunkTiles; cx++ {
			n := 0
			for ty := cy * chunkTiles; ty < (cy+1)*chunkTiles; ty++ {
				for tx := cx * chunkTiles; tx < (cx+1)*chunkTiles; tx++ {
					if terr.tile(tx, ty) != sim.TileMountain {
						n++
					}
				}
			}
			if d := math.Abs(float64(n)/float64(chunkTiles*chunkTiles) - open); d < bestD {
				best, bestD = chunkKey{cx, cy}, d
			}
		}
	}
	return best
}

func BenchmarkBakeCaveChunk(b *testing.B) {
	terr, _ := caveTestTerrain(b)
	for _, c := range []struct {
		name string
		open float64
	}{{"rock", 0}, {"tunnel", 0.3}, {"chamber", 0.8}} {
		k := caveChunkWith(terr, c.open)
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				terr.bakeChunk(k)
			}
		})
	}
}
