package dinopaint

import "image"

// The leg mask: which pixels of the painting are which leg, so the rig
// (internal/client/dinolegmask.go) cuts each leg along its painted outline
// instead of in columns. Cut in columns, a leg takes with it whatever else
// is in them (a chip of the belly, the edge of the next leg's foot) and
// leaves the rest of its own outline behind, straight-edged.
//
// dinos/hd/<name>_legs.png is the painting's size; a pixel's red is the
// rig's number of the leg it belongs to, from 1 (0: the body), and its
// alpha is 255 where the painting is.

// LegMask is the parts of each of a species' legs, in the order its rig
// lists them.
type LegMask [][]string

var legMasks = map[string]LegMask{}

// RegisterLegs gives species name a leg mask.
func RegisterLegs(name string, legs LegMask) { legMasks[name] = legs }

// LookupLegs is the leg mask of species name, if it has one.
func LookupLegs(name string) (LegMask, bool) { l, ok := legMasks[name]; return l, ok }

// PaintLegs paints the leg mask of painting m. A detail drawn over the hide
// (a nail, a claw) belongs to the leg whose shape is nearest it, if one is
// near.
func PaintLegs(legs LegMask, m *Painter) *image.NRGBA {
	of := map[int]int{} // part -> leg, from 1
	var shapes []SDF
	for li, ps := range legs {
		var ss []SDF
		for _, n := range ps {
			if j := m.PartID(n); j >= 0 {
				of[j] = li + 1
				ss = append(ss, m.Sp.Parts[j].Shape)
			}
		}
		shapes = append(shapes, Union(ss...))
	}
	img := image.NewNRGBA(image.Rect(0, 0, m.W, m.H))
	for i, in := range m.In {
		if !in {
			continue
		}
		leg := 0
		if j := m.Part[i]; j >= 0 && !m.Pro[i] {
			leg = of[j]
		} else {
			x, y := G(i%m.W, i/m.W)
			best := 4.0 // how near a detail must be to a leg to be its: a claw's tip is far from its toe
			for li, s := range shapes {
				if d := s(x, y); d < best {
					best, leg = d, li+1
				}
			}
		}
		img.Pix[4*i], img.Pix[4*i+3] = uint8(leg), 255
	}
	return img
}

// PaintLegsUnder paints the far legs whole where the near legs hide them,
// so a near leg stepping away shows the far leg behind it, not a hole:
// the colours, and the mask with the far leg's number in its green.
func PaintLegsUnder(sp *Species, legs LegMask, guide image.Image, m *Painter, mask *image.NRGBA) *image.NRGBA {
	var near []string
	type far struct {
		n     uint8
		shape SDF
	}
	var fars []far
	for li, ps := range legs {
		isFar := false
		var ss []SDF
		for _, n := range ps {
			if j := m.PartID(n); j >= 0 {
				isFar = isFar || m.Sp.Parts[j].Far
				ss = append(ss, m.Sp.Parts[j].Shape)
			}
		}
		if isFar {
			fars = append(fars, far{uint8(li + 1), Union(ss...)})
		} else {
			near = append(near, ps...)
		}
	}
	under, p := paintUnder(sp, &Under{Parts: near}, guide, m)
	out := image.NewNRGBA(under.Rect)
	isFar := map[uint8]bool{}
	for _, f := range fars {
		isFar[f.n] = true
	}
	// Near a near leg, a far leg's own pixels are painted again too: there
	// the painting has no outline along its edge, the near leg being in front.
	nearNear := func(i int) bool {
		x, y := i%m.W, i/m.W
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if q := m.Idx(x+dx, y+dy); q >= 0 && mask.Pix[4*q+3] == 255 && mask.Pix[4*q] != 0 && !isFar[mask.Pix[4*q]] {
					return true
				}
			}
		}
		return false
	}
	for i := range m.In {
		switch own := mask.Pix[4*i]; {
		case under.Pix[4*i+3] == 255 && own != 0 && !isFar[own]: // under a near leg
			x, y := G(i%m.W, i/m.W)
			for _, f := range fars {
				if f.shape(x, y) <= 0 {
					copy(out.Pix[4*i:4*i+4], under.Pix[4*i:4*i+4])
					mask.Pix[4*i+1] = f.n
					break
				}
			}
		case isFar[own] && p.In[i] && nearNear(i):
			c := p.Out[i]
			out.Pix[4*i], out.Pix[4*i+1], out.Pix[4*i+2], out.Pix[4*i+3] = c.R, c.G, c.B, 255
			mask.Pix[4*i+1] = own
		}
	}
	return out
}
