package dinopaint

import (
	"image"
	"path/filepath"
	"testing"
)

// TestThighs checks dinos/hd/<name>_legs.png, _legsunder.png and
// _legsbody.png are the thigh pictures of each species' painting as it
// stands, and that every leg has pixels and a thigh.
func TestThighs(t *testing.T) {
	for _, name := range Names() {
		th, ok := LookupThighs(name)
		if !ok {
			continue
		}
		sp, _ := Lookup(name)
		g := loadPNG(t, filepath.Join(dinos, name+".png"))
		_, p := Paint(sp, g)
		mask, under, body := PaintThighs(sp, th, g, p)
		for _, f := range []struct {
			suffix string
			img    *image.NRGBA
		}{{"_legs", mask}, {"_legsunder", under}, {"_legsbody", body}} {
			have := loadPNG(t, filepath.Join(dinos, "hd", name+f.suffix+".png"))
			diff := 0
			for y := 0; y < p.H; y++ {
				for x := 0; x < p.W; x++ {
					if want, got := f.img.NRGBAAt(x, y), nrgbaAt(have, x, y); (want.A == 0) != (got.A == 0) || (want.A != 0 && want != got) {
						diff++
					}
				}
			}
			if diff > 0 {
				t.Errorf("%s: dinos/hd/%s%s.png differs from its painting in %d pixels: go run ./cmd/dinopaint %s", name, name, f.suffix, diff, name)
			}
		}
		own, thigh := make([]int, len(th.Legs)+1), make([]int, len(th.Legs)+1)
		for i := 0; i < len(mask.Pix); i += 4 {
			if l := int(mask.Pix[i]); mask.Pix[i+3] != 0 && l < len(own) {
				own[l]++
				if l > 0 && mask.Pix[i+2] == ThighOwn {
					thigh[l]++
				}
			}
		}
		for l := 1; l < len(own); l++ {
			if own[l] == 0 || thigh[l] == 0 {
				t.Errorf("%s: leg %d of its mask has %d pixels, %d of them its thigh's", name, l, own[l], thigh[l])
			}
		}
	}
}
