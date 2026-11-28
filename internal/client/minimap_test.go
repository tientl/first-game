package client

import (
	"math"
	"testing"
)

func TestMinimapCentresOnThePlayer(t *testing.T) {
	tw, th := miniScale()
	if math.Abs(th/tw-float64(TileH)/TileW) > 1e-9 {
		t.Fatalf("minimap tiles are %.2fx%.2f, want the world's %d:%d shape", tw, th, TileW, TileH)
	}
	for _, p := range [][2]float64{{0, 0}, {10.5, 7.25}, {63.9, 1.1}, {500, 900}} {
		ox, oy := miniOrigin(p[0], p[1])
		if ox != math.Round(ox) || oy != math.Round(oy) {
			t.Errorf("origin for %v is not on whole pixels: %v,%v", p, ox, oy)
		}
		x, y := ox+p[0]*tw, oy+p[1]*th
		if math.Abs(x-miniW/2) > 0.5 || math.Abs(y-miniH/2) > 0.5 {
			t.Errorf("player at %v drawn at %.1f,%.1f, want the middle %d,%d", p, x, y, miniW/2, miniH/2)
		}
	}
	if miniBottom() <= miniTop+miniD {
		t.Error("miniBottom is not below the minimap")
	}
	if math.Abs(miniW*miniZoom-miniD) > 0.5 || miniW != miniH {
		t.Errorf("the minimap's canvas %dx%d at %.2f is not the round map %d across", miniW, miniH, miniZoom, miniD)
	}
}
