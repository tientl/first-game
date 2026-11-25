package client

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPickFilter(t *testing.T) {
	for _, c := range []struct {
		name      string
		scale     float64
		tx        float64
		want      ebiten.Filter
		wantTx    float64
		askLinear bool
		rotate    bool
	}{
		{name: "1080p: one image pixel to a screen pixel", scale: 1, tx: 10.0000001, want: ebiten.FilterNearest, wantTx: 10},
		{name: "720p: two thirds", scale: 2.0 / 3, tx: 10, want: ebiten.FilterLinear, wantTx: 10},
		{name: "zoomed in twice", scale: 2, tx: 7, want: ebiten.FilterNearest, wantTx: 7},
		{name: "asked smooth", scale: 1, tx: 3, askLinear: true, want: ebiten.FilterLinear, wantTx: 3},
		{name: "turned", scale: 1, tx: 3, rotate: true, want: ebiten.FilterLinear},
	} {
		o := &ebiten.DrawImageOptions{}
		o.GeoM.Scale(c.scale, c.scale)
		if c.rotate {
			o.GeoM.Rotate(0.3)
		}
		o.GeoM.Translate(c.tx, 0)
		asked := ebiten.FilterNearest
		if c.askLinear {
			asked = ebiten.FilterLinear
		}
		pickFilter(o, asked)
		if o.Filter != c.want {
			t.Errorf("%s: filter %v, want %v", c.name, o.Filter, c.want)
		}
		if !c.rotate {
			if got := o.GeoM.Element(0, 2); got != c.wantTx {
				t.Errorf("%s: x %v, want %v", c.name, got, c.wantTx)
			}
		}
	}
}

func TestWorldGrid(t *testing.T) {
	wd := &worldDst{r: 3}
	for _, c := range []struct{ v, want float64 }{
		{5, 0}, {5.3333333, 1.0 / 3}, {-0.3333333, 2.0 / 3}, {7.9999999, 0}, {2.6666667, 2.0 / 3},
	} {
		if got := wd.frac(c.v); got != c.want {
			t.Errorf("frac(%v) = %v, want %v", c.v, got, c.want)
		}
	}
	if layerShift(0) != 0 || math.Abs(layerShift(1.0/3)-2.0/3) > 1e-12 {
		t.Error("layerShift: the whole-pixel grid must not move")
	}
	// A plain shift lands on whole layer pixels, halves up as the old
	// canvas's sampling had it.
	var m ebiten.GeoM
	m.Translate(4.5, -2.5)
	snapGeoM(&m, ebiten.FilterNearest)
	if x, y := m.Element(0, 2), m.Element(1, 2); x != 5 || y != -2 {
		t.Errorf("snapGeoM: (%v, %v), want (5, -2)", x, y)
	}
}

func TestCameraFine(t *testing.T) {
	c := newCamera(10.51, 3.3, 640, 360, 3)
	x, _ := c.toScreen(10.51, 3.3)
	fx, _ := c.toScreenFine(10.51, 3.3)
	if fx != 320 {
		t.Errorf("the followed point is off the middle: %v", fx)
	}
	if x == fx {
		t.Errorf("toScreen should keep whole art pixels from the world origin: %v", x)
	}
	if old := newCamera(10.51, 3.3, 640, 360, 1); old.ox != float64(int(old.ox)) {
		t.Errorf("without sub-pixels the camera is on whole art pixels: %v", old.ox)
	}
}
