package client

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// fillPath fills p, laid out in dst's own units, on any drawTarget the
// interface draws on.
func fillPath(dst drawTarget, p *vector.Path, op *vector.DrawPathOptions) {
	switch d := dst.(type) {
	case *ebiten.Image:
		vector.FillPath(d, p, nil, op)
	case *uiDst:
		var q vector.Path
		q.AddPath(p, &vector.AddPathOptions{GeoM: d.geoM()})
		vector.FillPath(d.img, &q, nil, op)
	default:
		panic("fillPath: not an image")
	}
}

// uiProbe, while the interface's tests set it, is told of every string
// drawn and every box drawn to hold text, in image pixels, so a test can see
// that text stays inside its box (uiscreens_test.go).
var uiProbe *probe

type probeRect struct {
	x, y, w, h float64
	s          string // the text, for a string
}

type probe struct {
	texts []probeRect
	boxes []probeRect
}

// text notes a string drawn on d over the UI rectangle (x, y, w, h).
func (p *probe) text(d *uiDst, s string, x, y, w, h float64) {
	p.texts = append(p.texts, probeRect{d.X(x), d.Y(y), w * d.k, h * d.k, s})
}

// box notes a box drawn to hold text on d.
func (p *probe) box(d *uiDst, x, y, w, h float64) {
	p.boxes = append(p.boxes, probeRect{x: d.X(x), y: d.Y(y), w: w * d.k, h: h * d.k})
}

// noteBox tells the probe, if there is one, of a box that holds text.
func noteBox(d *uiDst, x, y, w, h float64) {
	if uiProbe != nil {
		uiProbe.box(d, x, y, w, h)
	}
}
