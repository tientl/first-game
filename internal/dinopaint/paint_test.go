package dinopaint

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const dinos = "../client/dinos"

func loadPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// TestPaintings paints every described species: three times its guide's
// size, its silhouette the guide's, and the same as the painting in
// dinos/hd (so a description changed without painting again is caught).
func TestPaintings(t *testing.T) {
	for _, name := range Names() {
		sp, _ := Lookup(name)
		g := loadPNG(t, filepath.Join(dinos, name+".png"))
		img, p := Paint(sp, g)
		gb := g.Bounds()
		if w, h := Size(gb.Dx(), gb.Dy(), K); img.Bounds().Dx() != w || img.Bounds().Dy() != h {
			t.Fatalf("%s: painted %v, want %dx%d", name, img.Bounds().Size(), w, h)
		}
		// The silhouettes agree: a guide pixel and the middle of its block
		// are mostly both solid or both clear.
		off := 0
		for y := 0; y < gb.Dy(); y++ {
			for x := 0; x < gb.Dx(); x++ {
				_, _, _, a := g.At(gb.Min.X+x, gb.Min.Y+y).RGBA()
				if (a >= 0x8000) != p.In[p.mid(x, y)] {
					off++
				}
			}
		}
		if limit := (gb.Dx() + gb.Dy()) * 2; off > limit {
			t.Errorf("%s: %d pixels of the silhouette differ from the guide's (at most %d)", name, off, limit)
		}
		hd := loadPNG(t, filepath.Join(dinos, "hd", name+".png"))
		diff := 0
		for y := 0; y < p.H; y++ {
			for x := 0; x < p.W; x++ {
				r0, g0, b0, a0 := img.At(x, y).RGBA()
				r1, g1, b1, a1 := hd.At(x, y).RGBA()
				if (a0 >= 0x8000) != (a1 >= 0x8000) || (a0 >= 0x8000 && (r0 != r1 || g0 != g1 || b0 != b1)) {
					diff++
				}
			}
		}
		if diff > 0 {
			t.Errorf("%s: dinos/hd/%s.png differs from its painting in %d pixels: go run ./cmd/dinopaint %s", name, name, diff, name)
		}
	}
}
