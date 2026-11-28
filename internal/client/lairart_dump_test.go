package client

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"
)

// TestDumpLairArt writes the art of the temple and the atoll on a sheet,
// twice as big as at 1080p, each beside a bar as tall as the player:
//
//	LAIR_DUMP=dir go test ./internal/client -run DumpLairArt
func TestDumpLairArt(t *testing.T) {
	dir := os.Getenv("LAIR_DUMP")
	if dir == "" {
		t.Skip("set LAIR_DUMP to a folder")
	}
	type item struct {
		c      *canvas
		ax, ay float64
	}
	var items []item
	add := func(c *canvas, ax, ay float64) { items = append(items, item{c, ax, ay}) }
	for v := range 3 {
		add(templePillarCanvas(v))
	}
	add(fallenColumnCanvas(0))
	add(serpentCanvas())
	add(campRingCanvas())
	add(campLogCanvas())
	add(deadGiantCanvas())
	add(tallMastCanvas())
	add(wreckDeckCanvas())
	const z = 2
	W, H := 40, 0
	for _, it := range items {
		W += it.c.w*z + 60
		H = max(H, it.c.h*z+80)
	}
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{58, 74, 52, 255}}, image.Point{}, draw.Src)
	x := 40
	for _, it := range items {
		top := H - 40 - it.c.h*z
		for y := 0; y < it.c.h; y++ {
			for xx := 0; xx < it.c.w; xx++ {
				c := it.c.px[y*it.c.w+xx]
				if c.A == 0 {
					continue
				}
				for dy := range z {
					for dx := range z {
						img.Set(x+xx*z+dx, top+y*z+dy, c)
					}
				}
			}
		}
		// The player's height beside it, from its ground point.
		gy := top + int(it.ay*z)
		for y := gy - 93*z; y < gy; y++ {
			img.Set(x-10, y, color.RGBA{255, 220, 120, 255})
			img.Set(x-9, y, color.RGBA{255, 220, 120, 255})
		}
		x += it.c.w*z + 60
	}
	writePNG(t, filepath.Join(dir, "lair_art.png"), img)
}
