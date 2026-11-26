package client

import (
	"image"
	"testing"

	"survivalgame/internal/sim"
)

func TestWorldSizeButton(t *testing.T) {
	var p diffPicker
	if got := p.worldSize(); got != sim.SizeHuge {
		t.Fatalf("a new world is %d tiles across by default, want %d", got, sim.SizeHuge)
	}
	p.size = 1
	if got := p.worldSize(); got != sim.SizeColossal {
		t.Fatalf("the second size is %d, want %d", got, sim.SizeColossal)
	}
	r := diffSizeRect()
	if !r.In(image.Rect(0, 0, diffW, diffH)) || r.Overlaps(diffBackRect()) || r.Dx() < 120 {
		t.Fatalf("size button at %v, back at %v, panel %dx%d", r, diffBackRect(), diffW, diffH)
	}
	for i := range diffBack {
		if r.Overlaps(diffCardRect(i)) {
			t.Fatalf("size button over card %d", i)
		}
	}
}
