package client

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"survivalgame/internal/sim"
)

// TERRAIN_SHEET=dir go test ./internal/client -run TerrainSheet -v bakes
// the ground round a few spots of a world of the newest style (a gorge
// with a waterfall, cliffs in each biome with a ramp, a ford, a lake, a
// reef with sea stacks) and writes each as a PNG at full size, the sea's
// moving layer under it a plain blue. TERRAIN_SIZE and TERRAIN_SEED pick
// another world; TERRAIN_ONLY keeps the spots whose name has it.
func TestTerrainSheet(t *testing.T) {
	dir := os.Getenv("TERRAIN_SHEET")
	if dir == "" {
		t.Skip("set TERRAIN_SHEET to a directory to write the sheets")
	}
	size, seed := 1024, uint64(42)
	if s, err := strconv.Atoi(os.Getenv("TERRAIN_SIZE")); err == nil {
		size = s
	}
	if s, err := strconv.ParseUint(os.Getenv("TERRAIN_SEED"), 10, 64); err == nil {
		seed = s
	}
	w, err := sim.Generate(sim.GenConfig{Seed: seed, Width: size, Height: size, Style: sim.StyleLatest})
	if err != nil {
		t.Fatal(err)
	}
	W, H := w.Size()
	terr := newTerrain(W, H, w.Tiles())
	terr.setRelief(w.Relief())
	for _, s := range sheetSpots(terr, W, w.SurfaceHeight()) {
		if o := os.Getenv("TERRAIN_ONLY"); o != "" && !strings.Contains(s.name, o) {
			continue
		}
		img := terr.bakeRegion(s.tx-s.rx, s.ty-s.ry, 2*s.rx+1, 2*s.ry+1)
		p := filepath.Join(dir, "terrain_"+s.name+".png")
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
		t.Logf("%s at %d,%d: %s", s.name, s.tx, s.ty, p)
	}
}

type sheetSpot struct {
	name           string
	tx, ty, rx, ry int
}

// sheetSpots finds the spots the sheet shows, nearest the middle of the
// home island first.
func sheetSpots(t *terrain, W, H int) []sheetSpot {
	count := func(tx, ty, rx, ry int, f func(x, y int) bool) int {
		n := 0
		for y := ty - ry; y <= ty+ry; y++ {
			for x := tx - rx; x <= tx+rx; x++ {
				if f(x, y) {
					n++
				}
			}
		}
		return n
	}
	is := func(k sim.Tile) func(x, y int) bool { return func(x, y int) bool { return t.tile(x, y) == k } }
	find := func(name string, rx, ry int, ok func(x, y int) bool, score func(x, y int) float64) (sheetSpot, bool) {
		best, bx, by := math.Inf(-1), 0, 0
		for y := ry + 2; y < H-ry-2; y++ {
			for x := rx + 2; x < W-rx-2; x++ {
				if !ok(x, y) {
					continue
				}
				if s := score(x, y); s > best {
					best, bx, by = s, x, y
				}
			}
		}
		return sheetSpot{name, bx, by, rx, ry}, !math.IsInf(best, -1)
	}
	var out []sheetSpot
	add := func(s sheetSpot, ok bool) {
		if ok {
			out = append(out, s)
		}
	}
	forest := func(x, y int) float64 { return 0.3 * float64(count(x, y, 9, 6, is(sim.TileForest))) }
	add(find("gorge", 10, 6, func(x, y int) bool {
		return t.tile(x, y) == sim.TileWaterfall && t.rel(x, y).Flow() == sim.FlowSouth && t.tile(x, y-1) == sim.TileRiver
	}, func(x, y int) float64 {
		return 3*float64(count(x, y, 3, 1, is(sim.TileWaterfall))) + float64(count(x, y, 9, 5, is(sim.TileCliff))) - forest(x, y)
	}))
	add(find("fallside", 10, 6, func(x, y int) bool {
		f := t.rel(x, y).Flow()
		return t.tile(x, y) == sim.TileWaterfall && (f == sim.FlowEast || f == sim.FlowWest)
	}, func(x, y int) float64 { return float64(count(x, y, 9, 5, is(sim.TileCliff))) - forest(x, y) }))
	for _, b := range []struct {
		name string
		k    sim.Tile
	}{{"cliffs_grass", sim.TileGrass}, {"cliffs_desert", sim.TileDesert}, {"cliffs_snow", sim.TileSnow}, {"cliffs_ash", sim.TileAsh}, {"cliffs_forest", sim.TileForest}} {
		add(find(b.name, 10, 6, func(x, y int) bool {
			return t.tile(x, y) == b.k && t.rel(x, y).Variant() == sim.VarRamp && t.tile(x-2, y) == sim.TileCliff
		}, func(x, y int) float64 { return float64(count(x, y, 10, 5, is(sim.TileCliff))) - forest(x, y) }))
	}
	add(find("ford", 9, 5, is(sim.TileFord), func(x, y int) float64 {
		return float64(count(x, y, 8, 5, is(sim.TileRiver))) - forest(x, y)
	}))
	add(find("lake", 10, 6, func(x, y int) bool { return t.tile(x, y).Walkable() }, func(x, y int) float64 {
		return float64(count(x, y, 9, 5, func(x, y int) bool { return t.tile(x, y) == sim.TileWater && t.rel(x, y).Variant() == sim.VarLake })) - forest(x, y)
	}))
	add(find("reef", 11, 6, func(x, y int) bool { return t.tile(x, y).Walkable() }, func(x, y int) float64 {
		return float64(count(x, y, 10, 6, func(x, y int) bool { return t.tile(x, y) == sim.TileShallow && t.rel(x, y).Variant() == sim.VarReef })) +
			8*float64(min(4, count(x, y, 10, 6, func(x, y int) bool { return t.tile(x, y) == sim.TileCliff && t.rel(x, y).Variant() == sim.VarStack })))
	}))
	add(find("lagoon", 11, 6, func(x, y int) bool { return t.tile(x, y) == sim.TileSand }, func(x, y int) float64 {
		return float64(count(x, y, 10, 6, func(x, y int) bool { return t.tile(x, y) == sim.TileShallow && t.rel(x, y).Variant() == sim.VarLagoon }))
	}))
	add(find("seacliff", 10, 6, func(x, y int) bool { return t.tile(x, y) == sim.TileCliff && t.tile(x, y+1).Sea() }, func(x, y int) float64 {
		return float64(count(x, y, 8, 1, func(x, y int) bool { return t.tile(x, y) == sim.TileCliff && t.tile(x, y+1).Sea() }))
	}))
	add(find("rapids", 10, 6, func(x, y int) bool {
		dx, dy := t.rel(x, y).Flow().Delta()
		return t.tile(x, y) == sim.TileRiver && t.tile(x+dx, y+dy) == sim.TileRiver && t.rel(x+dx, y+dy).Level() < t.rel(x, y).Level()
	}, func(x, y int) float64 { return float64(count(x, y, 6, 4, is(sim.TileRiver))) - forest(x, y) }))
	return out
}

// bakeRegion bakes the ground of the w by h tiles from (tx, ty) into one
// image, its sea over plain water.
func (t *terrain) bakeRegion(tx, ty, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w*hdTW, h*hdTH))
	x0, y0 := tx*hdTW, ty*hdTH
	chunks := map[chunkKey][]byte{}
	for y := 0; y < h*hdTH; y++ {
		for x := 0; x < w*hdTW; x++ {
			wx, wy := x0+x, y0+y
			k := chunkKey{fdiv(wx, chunkPW), fdiv(wy, chunkPH)}
			g, ok := chunks[k]
			if !ok {
				g, _, _ = t.bakeChunk(k)
				chunks[k] = g
			}
			i := ((wy-k.cy*chunkPH)*chunkPW + wx - k.cx*chunkPW) * 4
			c := color.RGBA{g[i], g[i+1], g[i+2], g[i+3]}
			// The pixels are premultiplied: lay them over the water.
			a := float64(c.A) / 255
			bg := colWaterBG
			img.SetRGBA(x, y, color.RGBA{
				uint8(float64(c.R) + float64(bg.R)*(1-a)), uint8(float64(c.G) + float64(bg.G)*(1-a)),
				uint8(float64(c.B) + float64(bg.B)*(1-a)), 255})
		}
	}
	return img
}

var _ = fmt.Sprint

// BenchmarkBakeIsles bakes the chunks round the sheet's spots of a world of
// the newest style (cliffs, falls, rivers, fords, lakes, reefs): what a
// chunk of such ground costs to bake.
func BenchmarkBakeIsles(b *testing.B) {
	w, err := sim.Generate(sim.GenConfig{Seed: 42, Width: 1024, Height: 1024, Style: sim.StyleLatest})
	if err != nil {
		b.Fatal(err)
	}
	W, H := w.Size()
	terr := newTerrain(W, H, w.Tiles())
	terr.setRelief(w.Relief())
	var keys []chunkKey
	for _, s := range sheetSpots(terr, W, w.SurfaceHeight()) {
		keys = append(keys, chunkKey{fdiv(s.tx*hdTW, chunkPW), fdiv(s.ty*hdTH, chunkPH)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		terr.bakeChunk(keys[i%len(keys)])
	}
}
