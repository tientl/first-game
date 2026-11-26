package dinopaint

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

// TestUnderLayers paints the under-layer of every species that has one and
// checks dinos/hd/<name>_under.png is the same, pixel for pixel, alpha too.
func TestUnderLayers(t *testing.T) {
	for _, name := range Names() {
		u, ok := LookupUnder(name)
		if !ok {
			continue
		}
		sp, _ := Lookup(name)
		g := loadPNG(t, filepath.Join(dinos, name+".png"))
		_, p := Paint(sp, g)
		img := PaintUnder(sp, u, g, p)
		have := loadPNG(t, filepath.Join(dinos, "hd", name+"_under.png"))
		if have.Bounds() != img.Bounds() {
			t.Errorf("%s: dinos/hd/%s_under.png is %v, want %v", name, name, have.Bounds().Size(), img.Bounds().Size())
			continue
		}
		hidden, diff := 0, 0
		for y := 0; y < p.H; y++ {
			for x := 0; x < p.W; x++ {
				want := img.NRGBAAt(x, y)
				if want.A == 255 {
					hidden++
				}
				if got := nrgbaAt(have, x, y); got != want && (got.A != 0 || want.A != 0) {
					diff++
				}
			}
		}
		if diff > 0 {
			t.Errorf("%s: dinos/hd/%s_under.png differs from its painting in %d pixels: go run ./cmd/dinopaint %s", name, name, diff, name)
		}
		if hidden == 0 {
			t.Errorf("%s: its under-layer hides no body", name)
		}
	}
}

func nrgbaAt(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}
