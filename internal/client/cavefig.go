package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The figure with nothing on but what everyone starts in: a man bare to
// the waist in a shaggy fur loincloth tied at the hip, a woman in a fur
// wrap and a fur skirt, both barefoot. It is the body under the clothes of
// tallfig.go: each of those (sim.Wear) covers its part when it is carried.
//
// Skin is shaded in the face's tones (tallhead.go), so it works for every
// skin colour: the lit tone, a mid tone where a surface turns away, a
// shadow in the folds between muscles, and a glint on what bulges most.
// Each muscle is lit as a bulge of its own.

// loinHD is the pale grey-brown fur of the loincloth; loinHideHD the hide's
// bare side, where it is rolled at the waist.
var (
	loinHD     = mkRamp(0x2c241f, 0x483d34, 0x695b4e, 0x8a7b6a, 0xa99a87, 0xc7baa6, 0xe0d6c4)
	loinHideHD = mkRamp(0x3d2f24, 0x5f4c3a, 0x846d55, 0xa68d70, 0xc4ad8e, 0xdccaae)
)

// The rows the loincloth and a woman's fur skirt hang from and to.
const (
	loinTop   = 69.0
	loinHem   = 83.5
	skirtTop  = 67.0
	skirtHemF = 86.5
)

// The density-1 art (the map's head marker, a fallen body) has no mane of
// its own: it wears long hair.
func init() { hairArts[sim.HairShaggy] = hairArts[sim.HairLong] }

// skinBand is the tone of bare skin under light v: 2 in shadow, 3 where it
// turns away, 4 lit, and 5 on what catches the sun.
func skinBand(v float64) int {
	switch {
	case v < 0.33:
		return 2
	case v < 0.56:
		return 3
	case v < 0.77:
		return 4
	}
	return 5
}

// skin is the skin's tone i, never darker than a fold nor lighter than a
// glint.
func (g *figGen) skin(i int) color.RGBA { return g.sk.at(max(1, min(5, i))) }

// shagTone is shaggy fur: strands that hang, each lock to its own length
// and lit at its root, dark where it ends over the next. The returned
// function takes the light to add at (x, y).
func shagTone(r ramp, a shape, round float64, seed uint32) func(x, y int, bias float64) color.RGBA {
	n := len(r)
	return func(x, y int, bias float64) color.RGBA {
		i := int((a.lum(x, y, round)*0.8 + 0.12 + bias) * float64(n))
		i = max(1, min(n-2, i))
		col := int(math.Floor(float64(x) / 2))
		yo := float64(y) + 7*hashf(col, 3, seed)
		row := int(math.Floor(yo / 7))
		switch h := hashf(col, row, seed+2); { // a lock lighter or darker than the next
		case h < 0.3:
			i--
		case h > 0.72:
			i++
		}
		switch t := yo/7 - float64(row); {
		case t < 0.16:
			i++
		case t > 0.86:
			i -= 2 // its end, over the lock below
		case x&1 == 0 && hashf(col, row, seed+1) < 0.5:
			i-- // between two strands
		}
		return r.at(max(i, 0))
	}
}

// bareTrunk is the shape of the bare trunk facing d (never DirLeft), down
// to the hips: a man's tapers from the shoulders to the waist.
func bareTrunk(female bool, d sim.Dir) shape {
	if female {
		t, _ := tallTorso(true, d, skirt{})
		return t.and(sAbove(77))
	}
	if d == sim.DirRight {
		return sPoly(21.4, 46.5, 32, 46.5, 35.4, 53.4, 34.8, 57, 33, 61, 32.7, 68, 33.8, 76, 20.4, 76, 21.3, 67, 19.6, 56, 20, 50)
	}
	return sPoly(17, 47, 22.5, 46.3, 31.5, 46.3, 37, 47, 38, 52, 37.2, 57, 34.9, 66, 35.5, 70, 36.5, 76,
		17.5, 76, 18.5, 70, 19.1, 66, 16.8, 57, 16, 52)
}

// bareBody draws the bare trunk and the neck on U, and over it the
// loincloth, or a woman's fur wrap and skirt.
func (g *figGen) bareBody(d sim.Dir, sk skirt) {
	if pilotOn() { // a style pilot's figure: see charpilot.go
		g.pBody(d, sk)
		return
	}
	f := &g.f
	female, up, side := g.look.Female, d == sim.DirUp, d == sim.DirRight
	U := g.U
	trunk := bareTrunk(female, d)

	// The neck and the slope of the shoulders, in the chin's shade.
	neck := sBox(24.4, 40, 29.6, 48.5, 1)
	switch {
	case side:
		neck = sPoly(24.6, 40, 30.4, 40, 30.6, 48.5, 24, 48.5)
	case female:
		neck = neck.and(sBox(25, 0, 30, 60, 0))
	default:
		neck = neck.or(sPoly(24.4, 43.6, 29.6, 43.6, 33.6, 47.4, 20.4, 47.4))
	}
	f.fill(U, trunk, g.trunkShader(trunk, d))
	f.add(U, neck.sub(trunk), func(x, y int, _ float64) color.RGBA {
		if y < 45 || x >= 29 {
			return g.sk[2]
		}
		return g.sk[3]
	})
	if !up && !side { // the hollow of the throat
		f.dot(U, 27, 47, g.sk[2])
	}

	if female {
		g.furWrap(d, trunk)
	}
	g.loincloth(d, sk)
}

// trunkShader shades the bare trunk: the chest and the belly's muscles
// from the front, the back's from behind, and both in profile.
func (g *figGen) trunkShader(trunk shape, d sim.Dir) shader {
	female, up, side := g.look.Female, d == sim.DirUp, d == sim.DirRight
	blush := mix(g.sk[3], skinTones[g.look.Skin].blush, 0.45)
	// A man's chest, or his shoulder blades: one on each side.
	var slab []shape
	switch {
	case female:
	case side:
		slab = []shape{sEll(31.2, 52.6, 4.4, 4.7)}
	case up:
		slab = []shape{sEll(21.4, 53.2, 5, 4.6), sEll(32.6, 53.2, 5, 4.6)}
	default:
		slab = []shape{sEll(21.6, 52.5, 5.6, 4.9), sEll(32.4, 52.5, 5.6, 4.9)}
	}
	return func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		dx, ax := fx-27, math.Abs(fx-27)
		v := trunk.lum(x, y, 8) + 0.03
		on := false // on the chest or a shoulder blade
		for _, s := range slab {
			if s.d(fx, fy) < 0 {
				v, on = s.lum(x, y, 4.4)+0.05, true
			}
		}
		i := skinBand(v)
		switch {
		case female && side:
			if fx < 22.6 && fy > 59 { // the small of the back
				i--
			}
		case female && up:
			if x == 27 && fy > 58 && fy < 68 { // the spine
				i--
			}
		case female:
			switch {
			case y == 48 && ax > 2.6 && ax < 6.5: // the collarbones
				i++
			case x == 27 && fy > 59.5 && fy < 66: // the belly's middle line
				i--
			case x == 27 && y == 66: // the navel
				i -= 2
			case (x == 23 || x == 30) && fy > 60 && fy < 65: // toned sides
				i--
			}
		case side:
			switch {
			case on:
				i = max(i, 3)
			case fx > 29.6 && y == 57: // under the chest
				i = 2
			case fx > 31.2 && (y == 61 || y == 64 || y == 67): // the belly's muscles, one over the next
				i--
			case fx > 29.4 && fx < 30.6 && fy > 58 && fy < 68: // and their edge
				i--
			case fx < 22.4 && fy > 58: // the small of the back
				i--
			}
		case up:
			switch {
			case x == 27 && fy > 49.5: // the spine
				i = min(i, 3) - 1
			case on:
				i = max(i, 3)
			case fy > 57 && fy < 67 && math.Abs(ax-(10.2-(fy-57)*0.36)) < 0.55: // the broad muscles running to the waist
				i--
			case y == 68 && ax > 2 && ax < 4: // dimples over the hips
				i--
			case ax > 1 && ax < 3 && fy > 58 && fy < 68: // the long muscles by the spine
				i = max(i, 4)
			case fy > 58: // the flanks turn away
				i = min(i, 3)
			}
		default:
			switch {
			case x == 27 && fy > 49.6 && fy < 58: // the breastbone
				i = 2
			case y == 48 && ax > 2.6 && ax < 9.6: // the collarbones
				i++
			case on && math.Abs(ax-6.2) < 0.6 && y == 55: // nipples
				return blush
			case on:
				i = max(i, 3)
			case fy < 58.6: // under the chest
				i = min(i, 3)
				if ax < 9 && fy > 56 {
					i = 2
				}
			case fy < 69 && ax < 5.4: // the belly's muscles: three pairs
				switch {
				case x == 27 && y == 68: // the navel
					i = 1
				case x == 27:
					i = 2
				case ax > 4.4 && fy < 67:
					i = min(i, 3)
				case y == 61 || y == 64 || y == 67:
					i = min(i, 3)
				case (y == 59 || y == 62 || y == 65) && ax > 1 && ax < 3.6 && dx < 0:
					i = 5
				default:
					i = 4
				}
			case fy < 67 && ax > 6 && ax < 8.6 && int(math.Floor(fy+ax*0.8))%3 == 0: // the ribs' muscles
				i = 2
			default: // the flanks turn away
				i = min(i, 3)
			}
		}
		return g.skin(i)
	}
}

// furWrap draws a woman's top on U: a band of fur wrapped round the chest
// and held by a strap over one shoulder.
func (g *figGen) furWrap(d sim.Dir, trunk shape) {
	f := &g.f
	up, side := d == sim.DirUp, d == sim.DirRight
	band := trunk.and(sBelow(49.8)).and(sAbove(58.6))
	var strap shape
	switch {
	case side:
		band = trunk.grow(0.5).and(sBelow(49.8)).and(sAbove(58.6))
		strap = sBox(25.4, 46.3, 30.2, 50.6, 0.6)
	case up:
		strap = sPoly(26.4, 50.6, 21.6, 46.2, 19.4, 46.5, 19, 50.6)
	default:
		strap = sPoly(27.6, 50.6, 32.4, 46.2, 34.6, 46.5, 35, 50.6)
	}
	top := band.or(strap.and(trunk.grow(0.3))).jag(1.6, 2.5, 55.4, 71)
	fur := shagTone(loinHD, top, 3.5, 73)
	f.fill(g.U, top, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		bias := 0.02
		switch {
		case up || side && fx < 30:
		case side:
			if fy > 51 && fy < 55.5 {
				bias += 0.08
			}
		case math.Hypot((math.Abs(fx-27)-3.6)/2.8, (fy-53.4)/2.6) < 1 && fy < 54.4:
			bias += 0.08 // the breast under it
		case math.Abs(fx-27) < 0.6 && fy > 51:
			bias -= 0.1
		}
		if fy > 57 {
			bias -= 0.08
		}
		return fur(x, y, bias)
	})
}

// loinShape is the loincloth or the fur skirt facing d (never DirLeft),
// its ragged hem carried by the legs as sk has it; (kx, ky) is where it is
// knotted.
func loinShape(female bool, d sim.Dir, sk skirt) (s shape, kx, ky float64) {
	top, hem := loinTop, loinHem
	if female {
		top, hem = skirtTop, skirtHemF
	}
	// The hem hangs in rags: long locks and short, in uneven bunches.
	rag := func(s shape, y0 float64) shape { return s.jag(2.2, 7.3, y0-6, 91).jag(4.6, 2.7, y0-6, 57) }
	if d == sim.DirRight {
		w0, w1, h0, h1, fx, bx, room := 20.5, 33.3, 19.8, 34.4, 34.8, 19.6, 5.5
		if female {
			w0, w1, h0, h1, fx, bx, room = 21.3, 32.5, 19.6, 34.2, 35, 19, 5
		}
		fy, by := hem, hem
		if sk.set {
			// Swung, but not so far that its ragged corner comes away in a sliver.
			fx, bx = math.Min(math.Max(fx, sk.fx+room), 39.5), math.Max(math.Min(bx, sk.bx-room), 15.5)
			fy, by = hem+sk.fy, hem+sk.by
		}
		s = rag(sPoly(w0, top, w1, top, h1, top+6, fx, fy+1.5, bx, by+1.5, h0, top+6), math.Min(fy, by))
		return s, 30.6, top + 1.4
	}
	l, r := hem+sk.dy[1]+1.5, hem+sk.dy[0]+1.5
	mid := math.Max(l, r) + 0.5
	if female {
		s = sPoly(20.6, top, 33.4, top, 36.4, top+8, 37.8, r, 27, mid, 16.2, l, 17.6, top+8)
	} else {
		s = sPoly(18, top, 36, top, 36.9, top+5, 37.6, r, 27, mid, 16.4, l, 17.1, top+5)
	}
	s = rag(s, math.Min(l, r))
	kx = 33.4
	if d == sim.DirUp {
		kx = 20.6
	}
	if female {
		kx = 27 + (kx-27)*0.8
	}
	return s, kx, top + 1.4
}

// loincloth draws the fur about the hips on U: shaggy, its upper edge
// rolled and knotted at one hip, its hem hanging in rags.
func (g *figGen) loincloth(d sim.Dir, sk skirt) {
	f := &g.f
	female, side := g.look.Female, d == sim.DirRight
	s, kx, ky := loinShape(female, d, sk)
	top := loinTop
	if female {
		top = skirtTop
	}
	bias := 0.0
	if side {
		bias = -0.06
	}
	fur := shagTone(loinHD, s, 5, 59)
	// The edge it is wrapped over itself along, from the knot down.
	out := 1.0
	if kx < 27 {
		out = -1
	}
	lap := func(fy float64) float64 { return kx - out*(fy-top)*0.22 }
	f.fill(g.U, s, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		switch {
		case fy < top+2.6: // rolled over at the waist: the hide's bare side
			i := 4 - int(fy-top)
			if (x+2*y)%5 == 0 {
				i--
			}
			if side && fx < 23.5 || !side && fx > 31 {
				i--
			}
			return loinHideHD.at(i)
		case fy < top+3.6:
			return fur(x, y, bias-0.16) // in its shadow
		case !side && math.Abs(fx-lap(fy)) < 0.55:
			return fur(x, y, bias-0.2)
		}
		return fur(x, y, bias)
	})
	f.dropSpecks(g.U, f.n, 6)
	// The knot, and its two ends hanging.
	for _, e := range [2][3]float64{{0.6, 6.4, 1.2}, {-0.9, 4.6, 1.1}} {
		end := sLock(kx, ky, kx+out*e[0]*1.6, ky+e[1]*0.5, kx+out*e[0], ky+e[1], e[2], 0.6)
		end = end.and(s.grow(0.6)) // never below the rags
		f.fill(g.U, end, bevel(loinHideHD, end, 1.4, -0.04))
	}
	knot := sEll(kx, ky, 1.9, 1.7)
	f.fill(g.U, knot, bevel(loinHideHD, knot, 1.7, 0.08))
}

// dropSpecks clears the pixels of part p on l that lie in islands of fewer
// than n: the tip of a rag that its jagged edge has cut off from the rest.
func (f *figHD) dropSpecks(l *figLayer, p uint16, n int) {
	c := l.c
	seen := make([]bool, len(c.px))
	var comp, stack []int
	for i := range c.px {
		if seen[i] || l.part[i] != p || c.px[i].A == 0 {
			continue
		}
		comp, stack = comp[:0], append(stack[:0], i)
		seen[i] = true
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, j)
			x, y := j%c.w, j/c.w
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if nx, ny := x+d[0], y+d[1]; c.in(nx, ny) {
					if k := ny*c.w + nx; !seen[k] && l.part[k] == p && c.px[k].A > 0 {
						seen[k] = true
						stack = append(stack, k)
					}
				}
			}
		}
		if len(comp) < n {
			for _, j := range comp {
				c.px[j], l.part[j] = color.RGBA{}, 0
			}
		}
	}
}

// sEllAxis is an ellipse about (cx, cy) with radius ra along the unit
// direction (ux, uy) and rb across it: a muscle along a limb.
func sEllAxis(cx, cy, ux, uy, ra, rb float64) shape {
	r, m := math.Max(ra, rb), math.Min(ra, rb)
	return shape{cx - r, cy - r, cx + r, cy + r, func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		a, b := (dx*ux+dy*uy)/ra, (dx*uy-dy*ux)/rb
		return (math.Sqrt(a*a+b*b) - 1) * m
	}}
}

// unitOf is (x, y) made a unit long, and how long it was.
func unitOf(x, y float64) (ux, uy, n float64) {
	n = math.Max(math.Hypot(x, y), 0.01)
	return x / n, y / n, n
}

// prof is how far a limb reaches from its axis along its length: pairs of
// t (0 at one joint, 1 at the next) and the reach there, eased between.
type prof []float64

func (p prof) at(t float64) float64 {
	t = clamp01(t)
	for i := 2; i < len(p); i += 2 {
		if t <= p[i] {
			s := (t - p[i-2]) / math.Max(p[i]-p[i-2], 1e-6)
			return lerpf(p[i-1], p[i+1], smooth(s))
		}
	}
	return p[len(p)-1]
}

func (p prof) scaled(k float64) prof {
	q := append(prof{}, p...)
	for i := 1; i < len(q); i += 2 {
		q[i] *= k
	}
	return q
}

func (p prof) most() float64 {
	r := 0.0
	for i := 1; i < len(p); i += 2 {
		r = math.Max(r, p[i])
	}
	return r
}

// sLimb is a limb from (ax, ay) to (bx, by): t of the way along it, it
// reaches out.at(t) to the side the unit (px, py) points to, square to the
// limb, and in.at(t) to the other. Its ends are round.
func sLimb(ax, ay, bx, by, px, py float64, out, in prof) shape {
	ux, uy, n := unitOf(bx-ax, by-ay)
	r := math.Max(out.most(), in.most())
	return shape{math.Min(ax, bx) - r, math.Min(ay, by) - r, math.Max(ax, bx) + r, math.Max(ay, by) + r, func(x, y float64) float64 {
		dx, dy := x-ax, y-ay
		s, c := dx*ux+dy*uy, dx*px+dy*py
		w := out
		if c < 0 {
			w = in
		}
		switch {
		case s < 0:
			return math.Hypot(s, c) - w.at(0)
		case s > n:
			return math.Hypot(s-n, c) - w.at(1)
		}
		return math.Abs(c) - w.at(s/n)
	}}
}

// The reach of a bare arm from its axis, a man's, down the upper arm and
// the forearm: on the outside (seen from the front or the back) and on the
// inside, or in front and behind (seen from the side). The shoulder's cap
// is widest a little below the joint and ends half way down the outside of
// the arm; the arm narrows to the elbow; the forearm is fullest just below
// the elbow, on the thumb's side, and slim at the wrist.
var (
	upperOut = prof{0, 4.3, 0.14, 4.85, 0.32, 4.3, 0.48, 3.05, 0.56, 3.0, 0.72, 3.15, 0.88, 2.65, 1, 2.2}
	upperIn  = prof{0, 3.3, 0.3, 3.0, 0.6, 3.2, 0.84, 2.5, 1, 2.1}
	foreOut  = prof{0, 2.3, 0.2, 2.95, 0.36, 2.85, 0.62, 2.2, 0.86, 1.65, 1, 1.6}
	foreIn   = prof{0, 2.05, 0.32, 2.6, 0.52, 2.45, 0.78, 1.8, 1, 1.5}

	upperFront = prof{0, 3.7, 0.25, 3.55, 0.45, 3.0, 0.62, 3.35, 0.8, 3.05, 1, 2.45}
	upperBack  = prof{0, 3.8, 0.25, 4.0, 0.45, 3.75, 0.7, 3.0, 1, 2.6}
	foreFront  = prof{0, 2.6, 0.2, 3.15, 0.4, 2.85, 0.7, 2.15, 1, 1.7}
	foreBack   = prof{0, 2.7, 0.15, 2.65, 0.5, 2.3, 1, 1.7}

	// A woman's, from the front or the back: slimmer, the muscles hardly marked.
	upperOutF = prof{0, 3.3, 0.15, 3.55, 0.36, 3.1, 0.56, 2.55, 0.8, 2.45, 1, 1.95}
	upperInF  = prof{0, 2.7, 0.4, 2.5, 0.75, 2.3, 1, 1.85}
	foreOutF  = prof{0, 1.95, 0.25, 2.35, 0.6, 1.85, 1, 1.35}
	foreInF   = prof{0, 1.8, 0.35, 2.1, 0.75, 1.55, 1, 1.3}
)

// armMuscle is a muscle that bulges on a bare arm: on the upper arm or the
// forearm, s pixels down it and c across (towards p or q, as a fraction of
// the reach there), ra long and rb wide.
type armMuscle struct {
	fore          bool
	s, c, ra, rb  float64
	strong, light float64 // how much it bulges, and how much lighter it is
}

// bareArm draws arm i of a figure facing d bare, on l: the cap of the
// shoulder coming to a point half way down the outside of the arm, the
// biceps in front and the triceps behind, the arm narrowing to the elbow,
// the forearm full below the elbow and slim at the wrist. It is lit as a
// limb, and each muscle as a bulge on it. A woman's is slighter and less
// marked.
func (g *figGen) bareArm(l *figLayer, a armGeom, bias float64, d sim.Dir, i int) {
	if pilotOn() {
		g.pArm(l, a, bias, d, i)
		return
	}
	f := &g.f
	side, back := d == sim.DirRight, d == sim.DirUp
	k, mk := 1.0, 1.0 // how thick the arm is, and how marked its muscles
	if g.look.Female {
		k, mk = 0.8, 0.4
	}
	ux, uy, n := unitOf(a.ex-a.sx, a.ey-a.sy)
	vx, vy, m := unitOf(a.wx-a.ex, a.wy-a.ey)
	// p and q point across the upper arm and the forearm: to the front of
	// the arm seen from the side, where the biceps is, and away from the
	// body seen from the front or the back.
	px, py, qx, qy := uy, -ux, vy, -vx
	if !side {
		lat := 1.0 - 2*float64(i)
		if px*lat < 0 || px == 0 && py > 0 {
			px, py = -px, -py
		}
		if qx*lat < 0 || qx == 0 && qy > 0 {
			qx, qy = -qx, -qy
		}
	}
	uo, ui, fo, fi := upperOut, upperIn, foreOut, foreIn
	switch {
	case side:
		uo, ui, fo, fi = upperFront.scaled(k), upperBack.scaled(k), foreFront.scaled(k), foreBack.scaled(k)
	case g.look.Female:
		uo, ui, fo, fi = upperOutF, upperInF, foreOutF, foreInF
	}
	upper := sLimb(a.sx, a.sy, a.ex, a.ey, px, py, uo, ui)
	fore := sLimb(a.ex, a.ey, a.wx, a.wy, qx, qy, fo, fi)

	var muscles []armMuscle
	switch {
	case side:
		muscles = []armMuscle{
			{false, 1.9, 0, 4.6, 4.2, 0.9, 0.04},    // the shoulder's cap
			{false, 8.3, 0.35, 4.2, 2.5, 0.7, 0.03}, // the biceps
			{false, 5.6, -0.45, 4.6, 2.5, 0.5, 0},   // the triceps
			{true, 3.4, 0.3, 3.8, 2.4, 0.6, 0.02},   // the forearm's mass
		}
	case back:
		muscles = []armMuscle{
			{false, 2, 0.15, 4.6, 4.2, 0.9, 0.04},
			{false, 6.6, 0.1, 5, 3, 0.8, 0.02},     // the triceps
			{true, 3.3, 0.35, 3.8, 2.4, 0.6, 0.02}, // down the outside of the forearm
		}
	default:
		muscles = []armMuscle{
			{false, 2.2, 0.2, 4.4, 4.3, 1.3, 0.02},
			{false, 8.6, -0.15, 4.0, 2.6, 1.3, 0.03}, // the biceps
			{true, 2.8, 0.45, 3.4, 2.2, 0.9, 0.04},   // the thumb's side of the forearm
			{true, 4.4, -0.35, 3.8, 2.1, 0.6, 0},     // and the other
		}
	}
	type bump struct {
		s             shape
		cx, cy, round float64
		fore          bool
		strong, light float64
	}
	bumps := make([]bump, len(muscles))
	for j, mu := range muscles {
		ax, ay, bx, by, cx, cy, r := a.sx, a.sy, ux, uy, px, py, ui.at(mu.s/n)
		if mu.fore {
			ax, ay, bx, by, cx, cy, r = a.ex, a.ey, vx, vy, qx, qy, fi.at(mu.s/m)
		}
		if mu.c > 0 {
			if mu.fore {
				r = fo.at(mu.s / m)
			} else {
				r = uo.at(mu.s / n)
			}
		}
		x, y := ax+bx*mu.s+cx*mu.c*r, ay+by*mu.s+cy*mu.c*r
		bumps[j] = bump{sEllAxis(x, y, bx, by, mu.ra*k, mu.rb*k), x, y, mu.rb * k, mu.fore, mu.strong * mk, mu.light * mk}
	}
	flat := lum(0, 0)
	// away is the way the light goes, on the screen: a bulge throws its
	// shadow that way.
	const awayX, awayY = 0.62, 0.78

	// The forearm lies over the upper arm unless it goes away from the viewer.
	foreTop := !back
	f.fill(l, upper.or(fore), func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		du, df := upper.d(fx, fy), fore.d(fx, fy)
		onFore := df < 0 && (foreTop || du >= 0)
		// The limb is lit softly, so that its muscles show on it, each lit
		// on its own and throwing a little shadow below it.
		var v float64
		if onFore {
			v = fore.lum(x, y, fo.at(0.3))
		} else {
			v = upper.lum(x, y, uo.at(0.4))
		}
		v = flat + (v-flat)*0.4
		for _, b := range bumps {
			if b.fore != onFore {
				continue
			}
			switch e := b.s.d(fx, fy); {
			case e < 0:
				v += (b.s.lum(x, y, b.round)-flat)*b.strong + b.light
			case e < 0.85:
				if dx, dy, _ := unitOf(fx-b.cx, fy-b.cy); dx*awayX+dy*awayY > 0.35 {
					v -= 0.13 * b.strong
				}
			}
		}
		s := (fx-a.sx)*ux + (fy-a.sy)*uy // down the upper arm
		c := (fx-a.sx)*px + (fy-a.sy)*py // across it
		if !onFore {
			// Where the shoulder's cap ends on the arm: in a point half way
			// down its outside, seen from the front or the back, and in a
			// point in the middle seen from the side.
			r := uo.at(s / n)
			if c < 0 {
				r = ui.at(s / n)
			}
			cn := c / math.Max(r, 0.1)
			if side {
				if math.Abs(s-n*(0.52-0.2*math.Abs(cn))) < 0.55 {
					v -= 0.16 * mk
				}
			} else if cn > 0.1 && math.Abs(s-n*(0.28+0.22*cn)) < 0.6 {
				v -= 0.16 * mk
			}
		} else {
			t := (fx-a.ex)*vx + (fy-a.ey)*vy // down the forearm
			e := (fx-a.ex)*qx + (fy-a.ey)*qy
			switch {
			case back && t < 1.2 && math.Abs(e) < 1.3: // the point of the elbow
				v += 0.12
			case !back && t < 1.3 && e < -0.2 && e > -2.2*k: // the crook of the elbow
				v -= 0.24
			case t > m-1.6: // the wrist
				v -= 0.06
			}
		}
		return g.skin(skinBand(v + bias))
	})
}

// bareLeg draws a bare leg on L: the thigh, the knee, the calf and the
// shin down to the ankle. view is the facing; dim darkens the far leg.
func (g *figGen) bareLeg(lg legGeom, dim float64, view sim.Dir) {
	if pilotOn() {
		g.pLeg(lg, dim, view)
		return
	}
	f := &g.f
	side, back := view == sim.DirRight, view == sim.DirUp
	rt0, rt1, rk, rc, ra := 5.2, 4.2, 3.3, 4.2, 2.4
	if g.look.Female {
		rt0, rt1, rk, rc, ra = 4.6, 3.7, 2.9, 3.5, 2.1
	}
	tx, ty := lg.kx-lg.hx, lg.ky-lg.hy
	tn := math.Max(math.Hypot(tx, ty), 0.01)
	sx, sy := lg.ax-lg.kx, lg.ay-lg.ky
	n := math.Max(math.Hypot(sx, sy), 0.01)
	sx, sy = sx/n, sy/n
	// The calf swells behind the shin: backwards when seen from the side.
	bx, by := 0.0, 0.0
	if side {
		bx, by = -sy*1.5, sx*1.5
	}
	thigh := sCap(lg.hx, lg.hy, lg.kx, lg.ky, rt0, rt1)
	shin := sCap(lg.kx, lg.ky, lg.ax, lg.ay, rk, ra)
	calf := sCap(lg.kx+sx*n*0.16+bx, lg.ky+sy*n*0.16+by, lg.kx+sx*n*0.5+bx*0.7, lg.ky+sy*n*0.5+by*0.7, rc, rc-1.1)
	// The kneecap, on the front of the knee.
	cx, cy := lg.kx, lg.ky
	if side {
		cx, cy = lg.kx+sy*1.2, lg.ky-sx*1.2
	}
	knee := sEll(cx, cy, 2.4, 2.5)
	// Round the knee the thigh and the shin are lit as one: no seam where
	// they meet, however far the knee bends.
	limb := thigh.or(shin)
	nearKnee := func(fx, fy float64) float64 { return clamp01(1 - (math.Hypot(fx-lg.kx, fy-lg.ky)-2)/5) }
	hemT := skirtT(g.look.Female, g.wear) // the fur's shadow falls on the thigh
	f.fill(g.L, thigh, func(px, py int, _ float64) color.RGBA {
		fx, fy := float64(px)+0.5, float64(py)+0.5
		i := skinBand(lerpf(thigh.lum(px, py, rt0), limb.lum(px, py, rt0), nearKnee(fx, fy)) - dim + 0.03)
		if t := ((fx-lg.hx)*tx + (fy-lg.hy)*ty) / (tn * tn); t > hemT-0.1 && t < hemT+0.13 {
			i = min(i, 3)
		}
		return g.skin(i)
	})
	f.add(g.L, shin.or(calf), func(px, py int, _ float64) color.RGBA {
		fx, fy := float64(px)+0.5, float64(py)+0.5
		v := shin.lum(px, py, rk)
		if dc := calf.d(fx, fy); dc < 0 && dc < shin.d(fx, fy) {
			v = calf.lum(px, py, rc)
		}
		v = lerpf(v, limb.lum(px, py, rt0), nearKnee(fx, fy))
		i := skinBand(v - dim + 0.03)
		t := (fx-lg.kx)*sx + (fy-lg.ky)*sy // down the shin
		c := (fx-lg.kx)*sy - (fy-lg.ky)*sx // across it, forwards
		switch {
		case back && math.Abs(c) < 0.55 && t > n*0.16 && t < n*0.46: // the calf's two halves
			i = min(i, 3) - 1
		case back && t > -0.6 && t < 0.5 && math.Abs(c) < 2.2: // the hollow of the knee
			i = min(i, 3)
		}
		return g.skin(i)
	})
	if !back {
		f.add(g.L, knee.and(thigh.or(shin)), func(px, py int, _ float64) color.RGBA {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			i := skinBand(knee.lum(px, py, 2.4) - dim + 0.08)
			if fy > cy+1.5 && math.Abs(fx-cx) < 1.6 { // the fold under it
				i = min(i, 3)
			}
			return g.skin(i)
		})
	}
}

// bareFoot draws a bare foot on L, of the leg last drawn: seen from the
// side its heel, its arch and its toes, the big one's nail; from the front
// the toes in a row; from behind the heel. toe is how far the toes point
// sideways.
func (g *figGen) bareFoot(lg legGeom, dim, toe float64, view sim.Dir) {
	if pilotOn() {
		g.pFoot(lg, dim, toe, view)
		return
	}
	f := &g.f
	ax, ay := math.Round(lg.ax*2)/2, lg.ay
	k := 1.0
	if g.look.Female {
		k = 0.9
	}
	switch view {
	case sim.DirRight:
		sn, cs := math.Sin(lg.pitch), math.Cos(lg.pitch)
		foot := sPoly(ax-3.2*k, ay+0.2, ax-1.8*k, ay-1.8, ax+2.2*k, ay-1.8, ax+4.6*k, ay+1, ax+8.4*k, ay+2.8, ax+9.4*k, ay+4.2,
			ax+9*k, ay+5.4, ax-3.2*k, ay+5.4).
			or(sEll(ax-1.3*k, ay+2.6, 2.5*k, 2.8)).
			or(sEll(ax+7.7*k, ay+3.9, 1.9, 1.3)).
			sub(sEll(ax+1.7*k, ay+5.5, 2.5*k, 1)).
			and(sAbove(ay + 5))
		foot = sRot(foot, ax, ay, lg.pitch).and(sAbove(limbGround)) // the toes bend flat on the ground
		f.add(g.L, foot, func(px, py int, _ float64) color.RGBA {
			dx, dy := float64(px)+0.5-ax, float64(py)+0.5-ay
			u, v := dx*cs+dy*sn, -dx*sn+dy*cs // along the foot, and down it
			i := skinBand(foot.lum(px, py, 3) - dim)
			switch {
			case u > 7.9*k && u < 9.2*k && v > 2.4 && v < 3.6: // the big toe's nail
				i = 5
			case math.Abs(u-6.6*k) < 0.55 && v > 3: // the toes begin
				i--
			case v >= 4: // the sole
				i = min(i, 3)
			case math.Hypot(u+0.3, v+0.2) < 1.3: // the ankle bone
				i++
			}
			return g.skin(i)
		})
	case sim.DirUp:
		foot := sBox(ax-2.6*k, ay-0.6, ax+2.6*k, ay+5.4, 2.3).or(sEll(ax+toe*0.7, ay+3.5, 3.5*k, 1.8)).and(sAbove(ay + 5))
		f.add(g.L, foot, func(px, py int, _ float64) color.RGBA {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			i := skinBand(foot.lum(px, py, 3) - dim)
			switch {
			case math.Abs(fx-ax) > 1.1 && math.Abs(fx-ax) < 2.1 && fy < ay+1.6: // either side of the tendon
				i--
			case fy >= ay+4:
				i = min(i, 3)
			}
			return g.skin(i)
		})
	default:
		cx := ax + toe*0.9
		in := 1.0 // towards the other foot: the big toe's side
		if toe > 0 {
			in = -1
		}
		foot := sBox(ax-2.7*k, ay-0.6, ax+2.7*k, ay+3.6, 2.2).or(sEll(cx, ay+3.2, 4.3*k, 2.4)).and(sAbove(ay + 5))
		f.add(g.L, foot, func(px, py int, _ float64) color.RGBA {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			i := skinBand(foot.lum(px, py, 3) - dim)
			if fy > ay+3 {
				// Five toes, the big one widest.
				for j, o := range [4]float64{1.5, -0.3, -1.9, -3.2} {
					switch e := fx - (cx + in*o*k); {
					case math.Abs(e) < 0.5:
						return g.skin(min(i, 3) - 1)
					case j == 0 && fy < ay+4 && math.Abs(e-in*1.3) < 0.55:
						i = 5 // its nail
					}
				}
			}
			return g.skin(i)
		})
	}
}
