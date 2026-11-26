package client

import "testing"

// A tree sapling stands between the stump and the young tree: about as tall
// as a person (31 art pixels) or a little more, slim, anchored at its foot,
// and the same every time it is drawn.
func TestTreeSaplings(t *testing.T) {
	for v := range saplingLooks {
		c, ax, ay := treeSaplingCanvas(v)
		x0, y0, x1, y1 := c.w, c.h, -1, -1
		for y := range c.h {
			for x := range c.w {
				if c.px[y*c.w+x].A > 0 {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
				}
			}
		}
		if x0 <= 0 || y0 <= 0 || x1 >= c.w-1 || y1 >= c.h-1 {
			t.Errorf("sapling %d touches the edge of its canvas: %d,%d to %d,%d of %dx%d", v, x0, y0, x1, y1, c.w, c.h)
		}
		tall := (ay - float64(y0)) / HDDensity
		t.Logf("sapling %d: %.1f art px tall, %.1f wide", v, tall, float64(x1-x0+1)/HDDensity)
		if tall < 31 || tall > 41 {
			t.Errorf("sapling %d is %.1f art pixels tall, want 31 to 41", v, tall)
		}
		if !c.opaque(int(ax), int(ay)-HDDensity) {
			t.Errorf("sapling %d has no trunk above its ground point", v)
		}
		again, _, _ := treeSaplingCanvas(v)
		for i := range c.px {
			if c.px[i] != again.px[i] {
				t.Fatalf("sapling %d is not drawn the same twice", v)
			}
		}
	}
}
