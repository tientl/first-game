package dinopaint

import (
	"path/filepath"
	"testing"
)

// TestLegMasks checks dinos/hd/<name>_legs.png is the leg mask of each
// species' painting as it stands.
func TestLegMasks(t *testing.T) {
	for _, name := range Names() {
		legs, ok := LookupLegs(name)
		if !ok {
			continue
		}
		sp, _ := Lookup(name)
		g := loadPNG(t, filepath.Join(dinos, name+".png"))
		_, p := Paint(sp, g)
		img := PaintLegs(legs, p)
		under := PaintLegsUnder(sp, legs, g, p, img)
		have := loadPNG(t, filepath.Join(dinos, "hd", name+"_legs.png"))
		haveUnder := loadPNG(t, filepath.Join(dinos, "hd", name+"_legsunder.png"))
		diff, counts := 0, make([]int, len(legs)+1)
		for y := 0; y < p.H; y++ {
			for x := 0; x < p.W; x++ {
				want := img.NRGBAAt(x, y)
				if want.A == 255 && int(want.R) < len(counts) {
					counts[want.R]++
				}
				if got := nrgbaAt(have, x, y); (got.A == 0) != (want.A == 0) || (want.A != 0 && (got.R != want.R || got.G != want.G)) {
					diff++
				}
				if u, got := under.NRGBAAt(x, y), nrgbaAt(haveUnder, x, y); (u.A == 255) != (got.A == 255) || (u.A == 255 && u != got) {
					diff++
				}
			}
		}
		if diff > 0 {
			t.Errorf("%s: dinos/hd/%s_legs.png differs from its painting in %d pixels: go run ./cmd/dinopaint %s", name, name, diff, name)
		}
		for i, n := range counts[1:] {
			if n == 0 {
				t.Errorf("%s: leg %d of its mask has no pixels", name, i+1)
			}
		}
	}
}
