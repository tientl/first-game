package sim

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// isleColors are the dump's colours of the tiles.
var isleColors = [NumTiles]color.RGBA{
	TileWater: {40, 90, 170, 255}, TileSand: {226, 206, 150, 255}, TileGrass: {98, 160, 72, 255},
	TileForest: {46, 100, 52, 255}, TileRocky: {140, 132, 118, 255}, TileDesert: {232, 176, 92, 255},
	TileMarsh: {84, 96, 58, 255}, TileMountain: {82, 78, 74, 255}, TileCave: {30, 26, 30, 255},
	TileAsh: {60, 56, 58, 255}, TileLava: {255, 90, 20, 255}, TileSnow: {240, 244, 250, 255}, TilePath: {150, 110, 70, 255},
	TileShallow: {80, 170, 200, 255}, TileRiver: {60, 130, 230, 255}, TileFord: {120, 180, 220, 255},
	TileWaterfall: {230, 245, 255, 255}, TileCliff: {70, 50, 40, 255},
}

// TestDumpIsles writes the surface of a StyleIsles world as a PNG, one
// pixel a tile, the levels shaded: ISLES_DUMP=dir [ISLES_SIZE=n ISLES_SEED=s].
func TestDumpIsles(t *testing.T) {
	dir := os.Getenv("ISLES_DUMP")
	if dir == "" {
		t.Skip("set ISLES_DUMP to a directory")
	}
	size, seed := SizeHuge, uint64(42)
	if v, err := strconv.Atoi(os.Getenv("ISLES_SIZE")); err == nil {
		size = v
	}
	if v, err := strconv.ParseUint(os.Getenv("ISLES_SEED"), 10, 64); err == nil {
		seed = v
	}
	w, err := Generate(GenConfig{Seed: seed, Width: size, Height: size, Style: StyleIsles})
	if err != nil {
		t.Fatal(err)
	}
	H := w.SurfaceHeight()
	img := image.NewRGBA(image.Rect(0, 0, w.width, H))
	for i, tl := range w.tiles[:w.width*H] {
		c := isleColors[tl]
		r := w.relief[i]
		k := 1 + 0.1*float64(r.Level())
		if r.Variant() == VarLake && tl == TileWater {
			c = color.RGBA{70, 120, 200, 255}
		}
		if tl == TileShallow && r.Variant() == VarReef {
			c = color.RGBA{110, 200, 190, 255}
		}
		if tl.Walkable() && r.Variant() == VarRamp {
			c = color.RGBA{255, 220, 0, 255}
		}
		img.Set(i%w.width, i/w.width, color.RGBA{uint8(min(255, float64(c.R)*k)), uint8(min(255, float64(c.G)*k)), uint8(min(255, float64(c.B)*k)), 255})
	}
	sx, sy := int(w.spawnX), int(w.spawnY)
	for d := -3; d <= 3; d++ {
		img.Set(sx+d, sy, color.RGBA{255, 0, 255, 255})
		img.Set(sx, sy+d, color.RGBA{255, 0, 255, 255})
	}
	var out image.Image = img
	name := "isles_" + strconv.FormatUint(seed, 10) + "_" + strconv.Itoa(size)
	var cx, cy, cw, ch int
	if n, _ := fmt.Sscanf(os.Getenv("ISLES_CROP"), "%d,%d,%d,%d", &cx, &cy, &cw, &ch); n == 4 {
		big := image.NewRGBA(image.Rect(0, 0, cw*4, ch*4))
		for y := 0; y < ch*4; y++ {
			for x := 0; x < cw*4; x++ {
				big.Set(x, y, img.At(cx+x/4, cy+y/4))
			}
		}
		out, name = big, name+"_crop"
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		t.Fatal(err)
	}
}
