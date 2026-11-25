package client

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestDumpCaveArt writes the cave mouth and the way out, twice their size on
// grass and on cave floor, to the directory named by CAVE_ART:
//
//	CAVE_ART=/tmp/cave go test ./internal/client -run DumpCaveArt
func TestDumpCaveArt(t *testing.T) {
	dir := os.Getenv("CAVE_ART")
	if dir == "" {
		t.Skip("set CAVE_ART to a directory to write the drawings")
	}
	for name, d := range map[string]struct {
		c  *canvas
		bg color.RGBA
	}{
		"mouth.png": {caveMouthCanvas(), hex(0x4b8a3d)},
		"exit.png":  {caveExitCanvas(), hex(0x2a2224)},
	} {
		const k = 2
		img := image.NewRGBA(image.Rect(0, 0, d.c.w*k, d.c.h*k))
		for y := 0; y < d.c.h*k; y++ {
			for x := 0; x < d.c.w*k; x++ {
				p := d.c.px[y/k*d.c.w+x/k]
				img.SetRGBA(x, y, withAlpha(mix(d.bg, withAlpha(p, 255), float64(p.A)/255), 255))
			}
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}
