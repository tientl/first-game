package dinopaint

import (
	"image"
	"image/color"
)

// The under-layer: what the body looks like under the parts that turn with
// the head. A three-horn's frill stands over its shoulders; when the rig
// lowers the head to eat, the frill swings off them, and the shoulders and
// the back under it must be there to show. The painter paints them whole
// into dinos/hd/<name>_under.png, the same size as the painting, and the
// rig (internal/client/dinounder.go) takes from it, pixel by pixel, which
// pixels turn with the head and what shows where they were.
//
// Its alpha says what the painting is at each pixel:
//
//	255     a part that turns with the head, with the body under it: the
//	        colour is that body, painted as if the head were not there;
//	UnderKeep  the body's own, which stays put when the head turns, even
//	        on the head's side of the rig's cut (the colour is the
//	        painting's, for looking at);
//	0       a part that turns with the head with nothing under it, or
//	        nothing at all.

// UnderKeep is the alpha of a pixel of the under-layer that is the body's
// own: below 128, so the game never draws it.
const UnderKeep = 64

// Under is how a species' under-layer is painted.
type Under struct {
	// Parts are the parts that turn with the head; every other part is the
	// body's. The body's parts should be drawn whole where the head hides
	// them: an extra part (a neck under a frill), hidden in the painting by
	// those in front of it, is painted only here.
	Parts []string
	Mat   string // the material of the hide they hide
	// Region limits where the hidden body is painted (nil: wherever a
	// part of the body holds it).
	Region SDF
}

var unders = map[string]*Under{}

// RegisterUnder gives species name an under-layer.
func RegisterUnder(name string, u *Under) { unders[name] = u }

// LookupUnder is the under-layer of species name, if it has one.
func LookupUnder(name string) (*Under, bool) { u, ok := unders[name]; return u, ok }

// PaintUnder paints the under-layer of species sp, whose painting m is,
// over its guide.
func PaintUnder(sp *Species, u *Under, guide image.Image, m *Painter) *image.NRGBA {
	omit := map[int]bool{}
	for _, n := range u.Parts {
		if j := m.PartID(n); j >= 0 {
			omit[j] = true
		}
	}
	sub := *sp
	sub.Parts = nil
	for j, pt := range sp.Parts {
		if !omit[j] {
			sub.Parts = append(sub.Parts, pt)
		}
	}
	sub.GuideDetail = 0 // the guide shows the head here, not the body
	sub.Details = nil
	p := New(&sub, guide)

	// The silhouette: the painting's, less what is only the head's.
	hidden := make([]bool, len(m.In))
	keep := make([]bool, len(m.In))
	for i, in := range m.In {
		if !in {
			continue
		}
		j := m.Part[i]
		switch {
		case j >= 0 && !omit[j]:
			p.In[i], keep[i] = true, true
		case j >= 0:
			x, y := G(i%m.W, i/m.W)
			if u.Region != nil && u.Region(x, y) > 0 {
				continue
			}
			for _, pt := range sub.Parts {
				if pt.Shape(x, y) <= 0 {
					p.In[i], hidden[i] = true, true
					break
				}
			}
		}
	}
	p.placeKnobs()
	p.Sil = sdfOf(p.In, p.W, p.H)
	for i := range p.Sil.V {
		p.Sil.V[i] /= K
	}
	p.materials()
	p.recolor()
	if mat := p.MatID(u.Mat); mat >= 0 {
		for i, h := range hidden {
			if h {
				p.Mat[i] = mat
			}
		}
	}
	p.assignParts()
	p.form()
	p.shade()
	p.skin()
	p.colour()
	p.outline()

	img := image.NewNRGBA(image.Rect(0, 0, p.W, p.H))
	set := func(i int, c color.NRGBA, a uint8) {
		img.Pix[4*i], img.Pix[4*i+1], img.Pix[4*i+2], img.Pix[4*i+3] = c.R, c.G, c.B, a
	}
	for i := range m.In {
		switch {
		case keep[i]:
			set(i, m.Out[i], UnderKeep)
		case hidden[i]:
			set(i, p.Out[i], 255)
		}
	}
	return img
}
