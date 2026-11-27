package client

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Away from any light the caves are near black and a little blue, and the
// brightness setting starts dark and stays among its choices.
func TestTheCavesAreDarkAwayFromLight(t *testing.T) {
	for k := range sim.CaveLevels {
		a := caveAmbient[k]
		l := 0.2126*a[0] + 0.7152*a[1] + 0.0722*a[2]
		if l < 0.04 || l > 0.09 || a[2] <= a[0] {
			t.Errorf("level %d: the dark is %v (light %.3f): want 4-9%%, bluish", k, a, l)
		}
	}
	if DefaultSettings().CaveLight != 0 {
		t.Errorf("the cave brightness starts at %d, not dark", DefaultSettings().CaveLight)
	}
	for _, c := range []struct{ in, want int }{{-3, 0}, {1, 1}, {9, len(caveBrightness) - 1}} {
		s := DefaultSettings()
		s.CaveLight = c.in
		if got := s.valid().CaveLight; got != c.want {
			t.Errorf("cave brightness %d kept as %d, want %d", c.in, got, c.want)
		}
	}
}

// A lamp's light falls from full at its centre to nothing at its edge,
// smoothly, and eyes shine only in the dark.
func TestLampsFallOffSoftly(t *testing.T) {
	if lampFall(0) != 1 || lampFall(1) != 0 || lampFall(2) != 0 {
		t.Errorf("lampFall: %v at 0, %v at 1, %v past it", lampFall(0), lampFall(1), lampFall(2))
	}
	for d := 0.0; d < 1; d += 0.01 {
		if lampFall(d+0.01) > lampFall(d) {
			t.Fatalf("lampFall rises at %.2f", d)
		}
	}
	if caveEyeShine(caveEyesFull) != 1 || caveEyeShine(caveEyesNone) != 0 || caveEyeShine(0.5) != 0 {
		t.Errorf("eyes: %v in the dark, %v at the edge, %v in the light", caveEyeShine(caveEyesFull), caveEyeShine(caveEyesNone), caveEyeShine(0.5))
	}
}

// The vignette reaches the screen's very edges: they fall on the centres
// of its edge pixels, not half a pixel past them.
func TestStretchOverReachesTheEdges(t *testing.T) {
	var m ebiten.GeoM
	stretchOver(&m, image.Rect(0, 0, 128, 128), 1920, 1080)
	for _, c := range [][4]float64{{0.5, 0.5, 0, 0}, {127.5, 127.5, 1920, 1080}} {
		x, y := m.Apply(c[0], c[1])
		if math.Abs(x-c[2]) > 1e-6 || math.Abs(y-c[3]) > 1e-6 {
			t.Errorf("pixel centre (%v, %v) lands at (%.2f, %.2f), want (%v, %v)", c[0], c[1], x, y, c[2], c[3])
		}
	}
}
