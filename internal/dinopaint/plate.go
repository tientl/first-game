package dinopaint

import (
	"image/color"
	"math"
)

// PlateSpec is a bony plate standing up out of the back, seen flat on, as
// a stegosaur's: its base sunk in the back from (BX0, BY0) to (BX1, BY1),
// its point at (TX, TY), its sides swelling Bulge past the straight lines
// from the base to the point.
type PlateSpec struct {
	BX0, BY0, BX1, BY1 float64
	TX, TY             float64
	Bulge              float64
	Face               []uint32 // its colours from dark to light
	Rim                []uint32 // the thin edge the light shines through, dark to light
	Shade              float64  // added to its light: below 0 for a plate on the far side
	Grooves            int      // the blood vessels' grooves fanning up from the base
	Seed               uint32
}

// outline is the plate's edge as a polygon: the base, then a curve up the
// back side to the point and down the front side.
func (s PlateSpec) outline() []float64 {
	mx, my := (s.BX0+s.BX1)/2, (s.BY0+s.BY1)/2
	ax, ay := s.TX-mx, s.TY-my
	l := math.Hypot(ax, ay)
	px, py := -ay/l, ax/l // across the plate, to the front (right of the axis when it points up)
	if px < 0 {
		px, py = -px, -py
	}
	side := func(bx, by, sgn float64, pts []float64, rev bool) []float64 {
		// A cubic from the base to the point: broadest a third of the way
		// up, rounding over into the point.
		c1x := lerp(bx, s.TX, 0.3) + sgn*px*2.4*s.Bulge
		c1y := lerp(by, s.TY, 0.3) + sgn*py*2.4*s.Bulge
		c2x := lerp(bx, s.TX, 0.85) + sgn*px*1.6*s.Bulge
		c2y := lerp(by, s.TY, 0.85) + sgn*py*1.6*s.Bulge
		const n = 14
		var seg []float64
		for i := 0; i <= n; i++ {
			t := float64(i) / n
			a, b, c, d := (1-t)*(1-t)*(1-t), 3*(1-t)*(1-t)*t, 3*(1-t)*t*t, t*t*t
			seg = append(seg, a*bx+b*c1x+c*c2x+d*s.TX, a*by+b*c1y+c*c2y+d*s.TY)
		}
		if rev {
			for i, j := 0, len(seg)-2; i < j; i, j = i+2, j-2 {
				seg[i], seg[j] = seg[j], seg[i]
				seg[i+1], seg[j+1] = seg[j+1], seg[i+1]
			}
		}
		return append(pts, seg...)
	}
	var pts []float64
	pts = side(s.BX0, s.BY0, -1, pts, false) // up the back side
	pts = side(s.BX1, s.BY1, 1, pts, true)   // and down the front
	// The point once, not twice: an edge of no length has no distance.
	h := len(pts) / 2
	return append(pts[:h], pts[h+2:]...)
}

// Shape is the plate's outline as a shape.
func (s PlateSpec) Shape() SDF { return Poly(s.outline()...) }

// Plate paints a plate over the hide, added to the silhouette: two faces
// meeting along a ridge from the base to the point, the one towards the
// light the brighter; darker where it grows out of the back; grooves for
// its blood vessels fanning up from the base; a thin edge the light shines
// through, warm and bright, inside a dark outline.
func (p *Painter) Plate(s PlateSpec) {
	shape := s.Shape()
	mx, my := (s.BX0+s.BX1)/2, (s.BY0+s.BY1)/2
	ax, ay := s.TX-mx, s.TY-my
	al := math.Hypot(ax, ay)
	ux, uy := ax/al, ay/al
	nx, ny := -uy, ux // across the plate, towards its front
	if nx < 0 {
		nx, ny = -nx, -ny
	}
	pick := func(r []uint32, v float64) color.NRGBA {
		return rgb(r[max(0, min(len(r)-1, int(math.Floor(clamp01(v)*float64(len(r))))))])
	}
	// The grooves: from points along the base, fanning out towards the
	// edge, wandering a little, each with a short branch.
	type line struct{ pts []float64 }
	var grooves []line
	for k := 0; k < s.Grooves; k++ {
		t := (float64(k) + 0.5 + 0.4*(hash(k, 1, s.Seed)-0.5)) / float64(s.Grooves)
		bx, by := lerp(s.BX0, s.BX1, t), lerp(s.BY0, s.BY1, t)
		bx, by = bx+ux*0.6, by+uy*0.6
		fan := (t - 0.5) * 1.5 // the outer ones lean out
		dx, dy := ux+fan*nx, uy+fan*ny
		dl := math.Hypot(dx, dy)
		dx, dy = dx/dl, dy/dl
		reach := al * (0.55 + 0.3*hash(k, 2, s.Seed)) * (1 - 0.45*math.Abs(t-0.5))
		var pts []float64
		x, y := bx, by
		const n = 6
		for i := 0; i <= n; i++ {
			pts = append(pts, x, y)
			w := (noise(float64(i)*0.9, float64(k)*3.1, s.Seed+7) - 0.5) * 0.7
			x += dx*reach/n + -dy*w*reach/n
			y += dy*reach/n + dx*w*reach/n
		}
		grooves = append(grooves, line{pts})
		// A branch off its middle, towards the edge it fans to.
		mi := 2 * (n / 2)
		bxm, bym := pts[mi], pts[mi+1]
		sgn := 1.0
		if t < 0.5 {
			sgn = -1
		}
		bdx, bdy := dx*0.8+sgn*-dy*0.6, dy*0.8+sgn*dx*0.6
		bl := reach * 0.35
		grooves = append(grooves, line{[]float64{bxm, bym, bxm + bdx*bl*0.5, bym + bdy*bl*0.5, bxm + bdx*bl, bym + bdy*bl}})
	}
	groove := func(x, y float64) (float64, float64) { // distance to the nearest groove, and how far up it
		best, at := math.Inf(1), 0.0
		for _, g := range grooves {
			for i := 0; i+3 < len(g.pts); i += 2 {
				d, t := segDist(x, y, g.pts[i], g.pts[i+1], g.pts[i+2], g.pts[i+3])
				if d < best {
					best, at = d, (float64(i/2)+t)/float64(len(g.pts)/2-1)
				}
			}
		}
		return best, at
	}
	const e = 0.25
	p.Fill(shape, true, func(px, py int, d float64) color.NRGBA {
		x, y := G(px, py)
		// along: 0 at the base, 1 at the point; across: + towards the front.
		rx, ry := x-mx, y-my
		along := (rx*ux + ry*uy) / al
		across := rx*nx + ry*ny
		// The two faces: the back one (across < 0) turned towards the light.
		face := smoothstep(-0.35, 0.35, across) // 0 back face, 1 front face
		v := lerp(0.66, 0.46, face) + s.Shade
		v += 0.12 * smoothstep(0.25, 1, along)         // lighter towards the point
		v -= 0.22 * (1 - smoothstep(0.02, 0.3, along)) // in the back's shadow at its root
		if math.Abs(across) < 0.3 && along > 0.1 && along < 0.92 {
			v += 0.07 // the ridge catches the light
		}
		if gd, gt := groove(x, y); gd < 0.2 && d > 0.6 {
			v -= 0.15 * (1 - 0.5*gt)
		} else if gd < 0.45 && d > 0.6 {
			v += 0.04
		}
		// The edge: which way it faces.
		gx := shape(x+e, y) - shape(x-e, y)
		gy := shape(x, y+e) - shape(x, y-e)
		gl := math.Hypot(gx, gy)
		lit := 0.0
		if gl > 0 {
			lit = smoothstep(-0.2, 0.8, -(gx*light2x+gy*light2y)/gl)
		}
		// Where it grows out of the hide there is no edge: it is sunk in the
		// back, or behind the plates in front of it. Over the air, there is.
		i := py*p.W + px
		base := 0.0
		if p.In[i] {
			base = 1 - smoothstep(0.15, 0.4, along)
		}
		switch {
		case d < 0.4/K*1.5 && base < 0.5:
			// The outline: dark, lighter where the edge faces the light.
			return Mix(pick(s.Rim, 0.15+0.3*lit+s.Shade), Ink, 0.55-0.35*lit)
		case d < 0.85 && base < 0.5:
			// The thin edge: the light shines through it, brightest where it
			// faces away from the light and is lit from behind.
			return pick(s.Rim, 0.55+0.25*lit+0.15*(1-lit)*smoothstep(0.3, 1, along)+s.Shade+0.15*(v-0.55))
		}
		return pick(s.Face, v)
	})
}
