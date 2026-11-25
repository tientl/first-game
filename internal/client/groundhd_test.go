package client

import (
	"testing"

	"survivalgame/internal/sim"
)

// groundTestTerrain is a small map with a strip of every kind of ground, sea
// round it, and a cave under its mountain.
func groundTestTerrain() *terrain {
	const w, h = 48, 32
	tiles := make([]sim.Tile, w*h)
	for y := 2; y < h-2; y++ {
		for x := 2; x < w-2; x++ {
			k := sim.TileSand + sim.Tile((x-2)*int(sim.NumTiles-1)/(w-4))
			if k == sim.TileCave && y < 12 {
				k = sim.TileMountain
			}
			tiles[y*w+x] = k
		}
	}
	return newTerrain(w, h, tiles)
}

// Every chunk bakes to images of the right size in both looks, the sea
// stays see-through and the two looks differ on grass.
func TestBakeChunk(t *testing.T) {
	terr := groundTestTerrain()
	defer SetGroundStyle(GroundGreen)
	var grass [2][]byte
	for i, style := range []GroundStyle{GroundGreen, GroundWarm} {
		SetGroundStyle(style)
		shores, roofs := 0, 0
		for cy := -1; cy <= 4; cy++ {
			for cx := -1; cx <= 6; cx++ {
				g, s, r := terr.bakeChunk(chunkKey{cx, cy})
				if len(g) != chunkPW*chunkPH*4 {
					t.Fatalf("chunk %d,%d: %d bytes of ground", cx, cy, len(g))
				}
				if s != nil {
					shores++
					if len(s) != chunkW*chunkH*4 {
						t.Fatalf("chunk %d,%d: %d bytes of shore", cx, cy, len(s))
					}
				}
				if r != nil {
					roofs++
					if len(r) != chunkPW*chunkPH*4 {
						t.Fatalf("chunk %d,%d: %d bytes of roof", cx, cy, len(r))
					}
				}
				if cx == -1 && cy == -1 && g[3] == 255 {
					t.Errorf("the open sea is opaque")
				}
				if cx == 1 && cy == 1 {
					grass[i] = g
				}
			}
		}
		if shores == 0 || roofs == 0 {
			t.Errorf("style %d: %d shore images, %d roofs", style, shores, roofs)
		}
	}
	same := true
	for i := range grass[0] {
		if grass[0][i] != grass[1][i] {
			same = false
			break
		}
	}
	if same {
		t.Errorf("the green and the warm ground bake the same grass")
	}
}

func BenchmarkBakeChunk(b *testing.B) {
	terr := groundTestTerrain()
	for _, c := range []struct {
		name string
		key  chunkKey
		warm bool
	}{
		{"sea", chunkKey{-1, -1}, false},
		{"coast", chunkKey{0, 0}, false},
		{"grass", chunkKey{1, 1}, false},
		{"warm", chunkKey{1, 1}, true},
		{"rock", chunkKey{2, 1}, false},
		{"mountain", chunkKey{3, 1}, false},
		{"lava", chunkKey{4, 1}, false},
		{"snow", chunkKey{5, 1}, false},
	} {
		b.Run(c.name, func(b *testing.B) {
			SetWarmGround(c.warm)
			defer SetWarmGround(false)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				terr.bakeChunk(c.key)
			}
		})
	}
}
