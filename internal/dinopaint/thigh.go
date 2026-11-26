package dinopaint

import (
	"image"
	"image/color"
)

// Thighs: a biped's legs painted to swing whole from the hip, the thigh a
// feathered drumstick of its own under the flank, not a piece of the flank
// that stays put while the shank swings below it. The painter writes three
// pictures the painting's size, which the rig (internal/client/
// dinolegmask.go) cuts and lays the legs with:
//
//	<name>_legs.png       red: the leg each pixel is, the rig's number from 1
//	                      (0: the body). Green: where a far leg lies hidden,
//	                      behind a near leg or the body, its number. Blue: on
//	                      the body, ThighFront where it is in front of the
//	                      near legs (the folded wing, the flank over the
//	                      hips); on a leg, ThighOwn where it is the thigh's.
//	<name>_legsunder.png  the far legs where they are hidden, painted whole,
//	                      so the thigh behind turns about its hip with no
//	                      gap under the belly.
//	<name>_legsbody.png   the body where the legs hide it: the flank behind
//	                      the near thigh, painted as if the leg were not
//	                      there, to show as the thigh swings off it.

// What the blue of a thigh mask says.
const (
	ThighFront = 1 // a pixel of the body over the near legs
	ThighOwn   = 2 // a pixel of a leg that turns with its thigh
)

// Thighs is how a species' legs are painted to swing whole.
type Thighs struct {
	Legs LegMask // each leg's parts, in the rig's order
	// Front is where the body is in front of the near legs: the flank's
	// feathers over the hips, past which a thigh swung up goes under them.
	// FrontParts are in front wherever they are: the folded wing, the arm.
	Front      SDF
	FrontParts []string
	// Thigh is each leg's thigh (nil: none): its pixels turn with the
	// thigh, the bone nearest them or not, so the drumstick's back over the
	// knee goes with it.
	Thigh []SDF
	// Coat paints the species' coat (its plumage, its fuzz) over the body
	// where the legs hide it and the legs where something hides them, as it
	// is over the painting (nil: none).
	Coat func(p *Painter)
}

var thighs = map[string]*Thighs{}

// RegisterThighs gives species name legs that swing whole.
func RegisterThighs(name string, t *Thighs) { thighs[name] = t }

// LookupThighs is how species name's legs swing whole, if they do.
func LookupThighs(name string) (*Thighs, bool) { t, ok := thighs[name]; return t, ok }

// PaintThighs paints the thigh mask of painting m and the far legs and the
// body where they are hidden.
func PaintThighs(sp *Species, t *Thighs, guide image.Image, m *Painter) (mask, under, body *image.NRGBA) {
	legOf := map[int]int{} // part -> leg, from 1
	farLeg := map[int]bool{}
	legShape := map[int]SDF{}
	for li, ps := range t.Legs {
		var ss []SDF
		for _, n := range ps {
			if j := m.PartID(n); j >= 0 {
				legOf[j] = li + 1
				farLeg[li+1] = farLeg[li+1] || sp.Parts[j].Far
				ss = append(ss, sp.Parts[j].Shape)
			}
		}
		legShape[li+1] = Union(ss...)
	}
	front := map[int]bool{}
	for _, n := range t.FrontParts {
		if j := m.PartID(n); j >= 0 {
			front[j] = true
		}
	}
	var bodyShapes []SDF
	for j, pt := range sp.Parts {
		if legOf[j] == 0 {
			bodyShapes = append(bodyShapes, pt.Shape)
		}
	}
	inBody := Union(bodyShapes...)

	// Whose each pixel is. A detail (a claw) or a feather's tip out past
	// the parts is the part nearest it, a leg's only if it is that near.
	mask = image.NewNRGBA(image.Rect(0, 0, m.W, m.H))
	own := make([]int, len(m.In))
	for i, in := range m.In {
		if !in {
			continue
		}
		x, y := G(i%m.W, i/m.W)
		j := m.Part[i]
		if j < 0 || m.Pro[i] {
			j = nearestPart(sp, x, y, 4)
		}
		leg := 0
		if j >= 0 {
			leg = legOf[j]
		}
		own[i] = leg
		mask.Pix[4*i], mask.Pix[4*i+3] = uint8(leg), 255
		switch {
		case leg > 0 && leg <= len(t.Thigh) && t.Thigh[leg-1] != nil && t.Thigh[leg-1](x, y) <= 0:
			mask.Pix[4*i+2] = ThighOwn
		case leg == 0 && j >= 0 && (front[j] || (t.Front != nil && t.Front(x, y) <= 0)):
			mask.Pix[4*i+2] = ThighFront
		}
	}

	// The legs whole where something hides them: a near thigh's top under
	// what is in front of it, a far leg under the body and the near legs.
	// A far leg's own edge along a near leg is painted again too: the near
	// leg had the outline there.
	under = image.NewNRGBA(mask.Rect)
	for pass := range 2 { // the near legs first: theirs is what shows first
		for li := range t.Legs {
			leg := li + 1
			if farLeg[leg] != (pass == 1) {
				continue
			}
			var thigh SDF
			if li < len(t.Thigh) {
				thigh = t.Thigh[li]
			}
			if !farLeg[leg] && thigh == nil {
				continue
			}
			shown, hidden := make([]bool, len(m.In)), make([]bool, len(m.In))
			for i := range m.In {
				shown[i] = m.In[i] && own[i] == leg
			}
			// Only so far from what shows of it: a thigh bends at the hip, its
			// top little, and what lies deeper under the body would only swing
			// out past the belly as a flap.
			near := edt(shown, m.W, m.H)
			for i, in := range m.In {
				if !in {
					continue
				}
				x, y := G(i%m.W, i/m.W)
				switch {
				case shown[i]:
				case mask.Pix[4*i+1] != 0, near[i] > hiddenReach*K:
				case farLeg[leg] && legShape[leg](x, y) <= 0:
					hidden[i] = true
				case !farLeg[leg] && own[i] == 0 && mask.Pix[4*i+2] == ThighFront && thigh(x, y) <= 0:
					hidden[i] = true
				}
			}
			p := paintPartly(sp, func(j int) bool { return legOf[j] == leg }, guide, m, shown, hidden, t.Coat)
			for i := range m.In {
				again := farLeg[leg] && shown[i] && nearOther(own, m.W, m.H, i, leg, 2)
				if (hidden[i] || again) && p.In[i] {
					c := p.Out[i]
					under.Pix[4*i], under.Pix[4*i+1], under.Pix[4*i+2], under.Pix[4*i+3] = c.R, c.G, c.B, 255
					mask.Pix[4*i+1] = uint8(leg)
				}
			}
		}
	}

	// The body where the legs hide it.
	shown, hidden := make([]bool, len(m.In)), make([]bool, len(m.In))
	for i, in := range m.In {
		if !in {
			continue
		}
		x, y := G(i%m.W, i/m.W)
		switch {
		case own[i] == 0:
			shown[i] = true
		case inBody(x, y) <= 0:
			hidden[i] = true
		}
	}
	p := paintPartly(sp, func(j int) bool { return legOf[j] == 0 }, guide, m, shown, hidden, t.Coat)
	body = image.NewNRGBA(mask.Rect)
	for i, h := range hidden {
		if h && p.In[i] {
			c := p.Out[i]
			body.Pix[4*i], body.Pix[4*i+1], body.Pix[4*i+2], body.Pix[4*i+3] = c.R, c.G, c.B, 255
		}
	}
	return mask, under, body
}

// hiddenReach is how far, in guide pixels, a leg is painted on under what
// hides it, from what shows of it.
const hiddenReach = 2.2

// nearestPart is the part of sp whose shape holds guide point (x, y), the
// frontmost, or else the nearest within reach, or -1.
func nearestPart(sp *Species, x, y, reach float64) int {
	best, bj := reach, -1
	for j := len(sp.Parts) - 1; j >= 0; j-- {
		d := sp.Parts[j].Shape(x, y)
		if d <= 0 {
			return j
		}
		if d < best {
			best, bj = d, j
		}
	}
	return bj
}

// nearOther reports whether pixel i has, within r pixels, one that is
// another leg's (not leg's, not the body's).
func nearOther(own []int, w, h, i, leg, r int) bool {
	x, y := i%w, i/w
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			X, Y := x+dx, y+dy
			if X < 0 || Y < 0 || X >= w || Y >= h {
				continue
			}
			if o := own[Y*w+X]; o != 0 && o != leg {
				return true
			}
		}
	}
	return false
}

// paintPartly paints only the parts of sp that keep says, where painting
// m shows them (shown) and where it has something else in front of them
// (hidden), its coat painted over them (nil: none). The drawing's colours at
// the hidden pixels are taken from the nearest shown ones, so the hide's
// regions and markings run on under what hid them.
func paintPartly(sp *Species, keep func(j int) bool, guide image.Image, m *Painter, shown, hidden []bool, coat func(p *Painter)) *Painter {
	sub := *sp
	sub.Parts = nil
	for j, pt := range sp.Parts {
		if keep(j) {
			sub.Parts = append(sub.Parts, pt)
		}
	}
	sub.Details = nil
	p := New(&sub, smear(guide, m, shown, hidden, sp.Ink))
	for i := range p.In {
		p.In[i] = shown[i] || hidden[i]
	}
	p.placeKnobs()
	p.Sil = sdfOf(p.In, p.W, p.H)
	for i := range p.Sil.V {
		p.Sil.V[i] /= K
	}
	p.materials()
	p.recolor()
	p.assignParts()
	p.form()
	p.shade()
	p.skin()
	p.colour()
	p.outline()
	if coat != nil {
		coat(p)
	}
	return p
}

// smear is guide with each of its pixels whose middle is hidden in
// painting m given the colour of the nearest shown pixel that is not
// outline (darker than ink).
func smear(guide image.Image, m *Painter, shown, hidden []bool, ink float64) image.Image {
	b := guide.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	src := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(guide.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			out.SetNRGBA(x, y, c)
			src[y*w+x] = shown[(y*K+K/2)*m.W+x*K+K/2] && c.A >= 128 && lum(c) >= ink
		}
	}
	const reach = 12
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !hidden[(y*K+K/2)*m.W+x*K+K/2] {
				continue
			}
			best, bx, by := reach*reach+1, -1, -1
			for dy := -reach; dy <= reach; dy++ {
				for dx := -reach; dx <= reach; dx++ {
					X, Y := x+dx, y+dy
					if d := dx*dx + dy*dy; d < best && X >= 0 && Y >= 0 && X < w && Y < h && src[Y*w+X] {
						best, bx, by = d, X, Y
					}
				}
			}
			if bx >= 0 {
				c := out.NRGBAAt(bx, by)
				c.A = 255
				out.SetNRGBA(x, y, c)
			}
		}
	}
	return out
}

// Plumed is a coat of plumage pl, for Thighs.
func Plumed(pl func() *Plumage) func(p *Painter) { return func(p *Painter) { p.Plumage(pl()) } }
