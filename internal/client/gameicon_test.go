package client

import "testing"

// The icon comes in every listed size, square, with the tile's corners cut
// away and its middle filled.
func TestGameIcons(t *testing.T) {
	icons := GameIcons()
	if len(icons) != len(GameIconSizes) {
		t.Fatalf("got %d icons, want %d", len(icons), len(GameIconSizes))
	}
	for i, img := range icons {
		n := GameIconSizes[i]
		if b := img.Bounds(); b.Dx() != n || b.Dy() != n {
			t.Errorf("icon %d is %dx%d, want %dx%d", i, b.Dx(), b.Dy(), n, n)
			continue
		}
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
			t.Errorf("size %d: the corner is not transparent", n)
		}
		for _, p := range [][2]int{{n / 2, n / 2}, {n / 2, 0}, {0, n / 2}, {n - 1, n / 2}, {n / 2, n - 1}} {
			if _, _, _, a := img.At(p[0], p[1]).RGBA(); a != 0xffff {
				t.Errorf("size %d: pixel %v is not opaque", n, p)
			}
		}
	}
}
