package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The pilot's bare trunk, its loincloth and a woman's fur top
// (charpilot.go). The trunk is a surface: a rounded slab of the body's
// outline with each muscle a rise on it, lit from the top left and cut into
// the skin's hue-shifted tones; creases fall where the muscles meet. Image
// pixels, the middle on column 27, as cavefig.go.

// pView is the facing as the painters ask it.
type pView struct{ side, back, front bool }

func viewOf(d sim.Dir) pView {
	return pView{side: d == sim.DirRight, back: d == sim.DirUp, front: d == sim.DirDown}
}

// pRampsOf is g's colours in the pilot's style.
func (g *figGen) pRamps() pRamps {
	return pRampsFor(int(g.look.Skin), int(g.look.HairColor), g.look.Female)
}

// mirrorPts is a polygon's left half (x, y pairs, top to bottom) closed by
// its mirror about column 27 (bottom to top).
func mirrorPts(left ...float64) []float64 {
	n := len(left) / 2
	pts := append([]float64{}, left...)
	for i := n - 1; i >= 0; i-- {
		pts = append(pts, 54-left[2*i], left[2*i+1])
	}
	return pts
}

// pTrunk is the bare trunk's outline facing v, down to the hips.
func pTrunk(female bool, v pView) shape {
	sp := pspec()
	if female {
		if v.side {
			return sPoly(24.6, 44.6, 29.4, 44.6, 31.2, 47, 32.6, 50.2, 34.4, 53.4, 34.2, 57, 32, 59.6, 31.4, 63.4, 32.4, 68.4, 33.8, 73.6, 33.4, 77,
				20.4, 77, 20.2, 73, 21.6, 67, 22.2, 62.6, 21.4, 58, 21, 52.6, 22, 48)
		}
		return sPoly(mirrorPts(24.4, 44.8, 21.2, 46.6, 18.6, 48.2, 18.2, 51.4, 18.8, 55.4, 20, 58.6, 21.2, 62, 21.2, 65.4, 19.6, 69.6, 18.2, 74.4, 18, 77)...)
	}
	if v.side {
		k := sp.chest / 9.6
		fx := func(x float64) float64 { return 27 + (x-27)*k }
		return sPoly(24.2, 44.2, 29.6, 44.2, fx(31.6), 46.8, fx(34.4), 49.6, fx(35.6), 52.6, fx(35), 56, fx(33.8), 58.2, fx(33.6), 62, fx(33.9), 66.4, fx(34.3), 70.6, fx(34.4), 77,
			fx(20.2), 77, fx(20.4), 72.4, fx(21.6), 67.6, fx(21.2), 62.6, fx(19.9), 57.4, fx(19.4), 52.4, fx(20.2), 48.6, 22.4, 45.8)
	}
	S, C, W, H := sp.shoulder, sp.chest, sp.waist, sp.hips
	return sPoly(mirrorPts(
		24.2, 44.4,
		27-S+5.0, 45.9,
		27-S+2.4, 47.0,
		27-S+0.2, 49.6,
		27-C-0.2, 53.6,
		27-C+0.3, 57.6,
		27-W-1.1, 62,
		27-W, 66.2,
		27-W-0.5, 69.4,
		27-H, 74.6,
		27-H+0.2, 77,
	)...)
}

// pTrunkH is the trunk's surface: the slab of its outline, the muscles of
// a man's chest, belly and back, or a woman's softer forms.
func pTrunkH(female bool, v pView, trunk shape) pHeight {
	sp := pspec()
	mk := sp.muscleK
	type bump struct{ x, y, ra, rb, ux, uy, high float64 }
	var bumps []bump
	var grooves [][4]float64 // x, y0, y1, depth: a groove down the middle
	add := func(x, y, ra, rb, high float64) { bumps = append(bumps, bump{x, y, ra, rb, 1, 0, high}) }
	addT := func(x, y, ra, rb, ux, uy, high float64) { bumps = append(bumps, bump{x, y, ra, rb, ux, uy, high}) }
	both := func(dx, y, ra, rb, high float64) { add(27-dx, y, ra, rb, high); add(27+dx, y, ra, rb, high) }
	bothT := func(dx, y, ra, rb, ux, uy, high float64) {
		addT(27-dx, y, ra, rb, ux, uy, high)
		addT(27+dx, y, ra, rb, ux, -uy, high)
	}
	C := sp.chest
	switch {
	case female && v.side:
		add(31.2, 54.4, 3.4, 3.4, 1.6) // the breast, under the fur
		add(30.4, 65.6, 4, 3.4, 0.5)   // the belly
		add(22.4, 72.8, 3.2, 4, 0.8)   // the seat
	case female && v.back:
		both(4.6, 52.8, 4, 3.4, 0.7) // the shoulder blades
		grooves = append(grooves, [4]float64{27, 48, 70, 0.6})
		both(3, 74.6, 4, 3.4, 0.9)
	case female:
		both(3.9, 54.2, 3.4, 3.3, 1.8) // the breasts, under the fur
		add(27, 66.4, 4.4, 3.4, 0.45)  // the belly
		grooves = append(grooves, [4]float64{27, 60, 66.6, 0.25})
	case v.side:
		k := C / 9.6
		addT(27+4.6*k, 52.4, 4.8*k, 3.9, 0.97, 0.24, 2*mk) // the chest
		add(27+5.2*k, 60, 2.2, 1.6, 0.6*mk)                // the belly's muscles, one over the next
		add(27+5.4*k, 63.3, 2.2, 1.6, 0.6*mk)
		add(27+5.6*k, 66.6, 2.2, 1.6, 0.55*mk)
		addT(27+1.8*k, 60.2, 2.6, 1.1, 0.5, 0.87, 0.5*mk) // the ribs' muscles
		addT(27+0.8*k, 63, 2.6, 1.1, 0.5, 0.87, 0.45*mk)
		add(27-4.6*k, 54, 3.6, 4.6, 1.2*mk) // the back: the lats
		add(27-5, 74, 3, 3, 0.6)
	case v.back:
		both(4.8, 53.8, 3.8, 3.4, 0.9*mk)              // the shoulder blades
		bothT(7.2, 47.8, 4.6, 2.4, 0.9, 0.42, 0.8*mk)  // the trapezius, from the neck to the shoulder
		bothT(C-2.2, 58.4, 5, 2.4, 0.33, 0.94, 0.8*mk) // the lats, sweeping to the waist
		both(2.4, 66, 1.8, 5, 0.8*mk)                  // the long muscles either side of the spine
		grooves = append(grooves, [4]float64{27, 46.5, 72, 1.0 * mk})
		both(3.6, 75.4, 3.6, 3.2, 0.8)
	default:
		both(C*0.47, 52.2, C*0.52, 3.9, 1.7*mk) // the chest
		// the belly's muscles, three pairs, the lowest the longest
		for j, y := range []float64{59.4, 62.6, 65.9} {
			both(2.1, y, 2.05, 1.55+0.15*float64(j), 0.6*mk)
		}
		add(27, 70.8, 5.6, 2.6, 0.35) // under the navel
		// the ribs' muscles, fingers along the flanks under the arms
		for j, y := range []float64{56.6, 59, 61.4} {
			bothT(C-1.6-0.3*float64(j), y, 1.9, 0.9, 0.82, 0.57, 0.4*mk)
		}
		bothT(C-0.6, 64.6, 3.4, 1.4, 0.3, 0.95, 0.45*mk) // the obliques over the hips
		grooves = append(grooves, [4]float64{27, 49.5, 69.5, 0.8 * mk})
	}
	return func(x, y float64) float64 {
		d := trunk.d(x, y)
		h := edgeH(d, 7.5)
		for _, b := range bumps {
			if q := ellQ(x, y, b.x, b.y, b.ux, b.uy, b.ra, b.rb); q < 1 {
				h += bumpH(q, b.high)
			}
		}
		for _, gr := range grooves {
			if y > gr[1] && y < gr[2] {
				e := math.Min(1, math.Min(y-gr[1], gr[2]-y)/2)
				h -= gr[3] * e * math.Exp(-(x-gr[0])*(x-gr[0])/0.9)
			}
		}
		if !female && !v.side && !v.back { // the collarbones
			dx := math.Abs(x - 27)
			if dx > 1.6 && dx < 9 {
				h += 0.45 * math.Exp(-math.Pow((y-(47.1+0.1*dx))/0.75, 2))
			}
		}
		return h
	}
}

// pSkinShade shades skin on surface h of shape s at (x, y): the light in
// the ramp's tones and a crease's shade (the cool rim on the edge away from
// the sun is the outline's: see pilotOutline). bias lightens or darkens.
func (g *figGen) pSkinShade(rp *pRamps, s shape, h pHeight, x, y int, bias float64) color.RGBA {
	sp := pspec()
	fx, fy := float64(x)+0.5, float64(y)+0.5
	v := pLight(h, x, y) + bias
	if c := pCrease(h, x, y, 1.8); c > 0.3 {
		v -= math.Min(0.22, (c-0.3)*0.4)
	}
	col := rp.skin[pToneI(len(rp.skin), v, x, y)]
	if pWarm != nil && sp.warmZones > 0 {
		if w := pWarm(fx, fy) * sp.warmZones; w > 0.02 {
			col = mix(col, shiftLC(col, 0.97, 1.4, -0.14), math.Min(1, w))
		}
	}
	return col
}

// pBody draws the bare trunk and the neck on U, and over it the loincloth,
// or a woman's fur top and skirt.
func (g *figGen) pBody(d sim.Dir, sk skirt) {
	f := &g.f
	v := viewOf(d)
	female := g.look.Female
	rp := g.pRamps()
	trunk := pTrunk(female, v)

	// The neck: its two cords from behind the ears to the hollow of the
	// throat, or the nape; and the trunk.
	neck := sBox(24.2, 39.5, 29.8, 48, 1.4)
	if female {
		neck = sBox(24.8, 39.5, 29.2, 48, 1.2)
	}
	if v.side {
		neck = sPoly(24.4, 39.5, 30.6, 39.5, 31, 47.5, 23.8, 47.5)
		if female {
			neck = sPoly(24.8, 39.5, 30, 39.5, 30.4, 47.5, 24.4, 47.5)
		}
	}
	body := trunk.or(neck)
	th := pTrunkH(female, v, trunk)
	nh := func(x, y float64) float64 {
		h := edgeH(neck.d(x, y), 2.6)
		if !v.side && !v.back && !female { // the cords of the neck, a V to the throat
			for _, s := range [2]float64{-1, 1} {
				cx := 27 + s*lerpf(2.3, 0.9, clamp01((y-41)/6.5))
				h += 0.55 * math.Exp(-math.Pow((x-cx)/0.8, 2))
			}
		}
		return h
	}
	hh := func(x, y float64) float64 {
		if trunk.d(x, y) < 0 {
			return th(x, y)
		}
		return nh(x, y) - 0.5
	}
	f.fill(g.U, body, func(x, y int, _ float64) color.RGBA {
		fy := float64(y) + 0.5
		bias := 0.0
		if fy < 45.5 && trunk.d(float64(x)+0.5, fy) > 0 {
			bias = -0.2 // the neck in the chin's shade
		}
		if !female && v.front && x >= 26 && x <= 27 && y == 47 {
			bias -= 0.25 // the hollow of the throat
		}
		if !female && v.front && x == 26 && y == 68 {
			return rp.skin[1] // the navel
		}
		if female && v.front && x == 26 && y == 67 {
			return rp.skin[2]
		}
		return g.pSkinShade(&rp, body, hh, x, y, bias)
	})
	f.pClean(g.U, f.n, 2)
	if female {
		g.pFurTop(d, trunk, &rp)
	}
	g.pLoin(d, sk, &rp)
}

// pFur shades fur: locks hanging in clumps a few pixels wide, each lit as
// a little cylinder on its left and parted from the next by a dark line,
// lit at its root and darker at its tip; the whole shaded by the light v
// on the garment. seed varies it.
func pFur(r ramp, x, y int, v float64, seed uint32, clump float64) color.RGBA {
	return pFurAt(r, x, y, v, seed, clump, -1e9)
}

// pFurAt is pFur for locks that hang from row top: two tiers of them, the
// upper's pointed tips lying over the roots of the lower, which are in
// their shadow.
func pFurAt(r ramp, x, y int, v float64, seed uint32, clump, top float64) color.RGBA {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	tier, y0 := 0, top
	if top > -1e8 {
		// where the upper tier's locks end, each in a point
		k0 := math.Floor(fx / clump)
		u0 := fx/clump - k0
		end := top + 5.5 + 2.2*hashf(int(k0), 8, seed) + 2.4*(1-math.Abs(2*u0-1))
		if fy > end {
			tier, y0 = 1, end-2.4
			if fy < end+1.6 {
				v -= 0.2 // in the shadow of the tips over it
			}
		}
	}
	off := float64(tier) * clump * 0.5
	wx := fx + off + 0.5*math.Sin(fy*0.35+hashf(int((fx+off)/clump), 1, seed)*6)
	k := math.Floor(wx / clump)
	u := wx/clump - k // across the lock, 0 to 1
	h := hashf(int(k), 3+tier, seed)
	t := clamp01((fy - y0) / 9) // down the lock
	if top < -1e8 {
		ln := 6 + 3*hashf(int(k), 4, seed)
		t = math.Mod(fy+h*ln, ln) / ln
	}
	fk := pspec().furK
	v += 0.26 * fk * (0.45 - math.Abs(u-0.4)*1.5) // round across, lit on its left
	v += 0.12 * (h - 0.5)                         // each lock its own
	v += 0.06 - 0.16*t                            // lit at its root, darker down it
	if u > 1-0.95/clump {                         // a pixel wide, whatever the lock's width
		v -= 0.22 * fk // the parting
	}
	return pTone(r, v, x, y)
}

// pLoinShape is the loincloth or the fur skirt facing v, its ragged hem
// carried by the legs as sk has it; (kx, ky) is where it is knotted.
func pLoinShape(female bool, v pView, sk skirt) (s shape, top, kx, ky float64) {
	sp := pspec()
	top, hem := sp.loinTop, sp.loinHem
	if female {
		top, hem = skirtTop, skirtHemF+0.5
	}
	// The hem in pointed tufts of uneven length.
	rag := func(s shape, y0 float64) shape { return s.jag(1.8, 5.1, y0-6, 91).jag(3.6, 2.4, y0-5, 57) }
	if v.side {
		w0, w1, h0, h1, fx, bx, room := 20.0, 33.8, 19.4, 34.8, 35.2, 19.2, 5.5
		if female {
			w0, w1, h0, h1, fx, bx, room = 21.0, 32.9, 19.4, 34.4, 35.2, 18.8, 5.0
		}
		fy, by := hem, hem
		if sk.set {
			fx, bx = math.Min(math.Max(fx, sk.fx+room), 39.5), math.Max(math.Min(bx, sk.bx-room), 15.5)
			fy, by = hem+sk.fy, hem+sk.by
		}
		s = rag(sPoly(w0, top, w1, top, h1, top+6, fx, fy+1.5, bx, by+1.5, h0, top+6), math.Min(fy, by))
		return s, top, 30.4, top + 1.2
	}
	l, r := hem+sk.dy[1]+1.5, hem+sk.dy[0]+1.5
	mid := math.Max(l, r) + 0.5
	H := sp.hips
	if female {
		s = sPoly(20.4, top, 33.6, top, 36.4, top+8, 37.8, r, 27, mid, 16.2, l, 17.6, top+8)
	} else {
		s = sPoly(27-H+0.4, top, 27+H-0.4, top, 27+H+0.2, top+5, 27+H+0.9, r, 27, mid, 27-H-0.9, l, 27-H-0.2, top+5)
	}
	s = rag(s, math.Min(l, r))
	kx = 27 + H - 2.4
	if v.back {
		kx = 27 - H + 2.4
	}
	if female {
		kx = 27 + (kx-27)*0.8
	}
	return s, top, kx, top + 1.2
}

// pLoin draws the fur about the hips on U: a dark leather thong round the
// waist, knotted at one hip, and under it the fur in hanging locks.
func (g *figGen) pLoin(d sim.Dir, sk skirt, rp *pRamps) {
	f := &g.f
	v := viewOf(d)
	female := g.look.Female
	s, top, kx, ky := pLoinShape(female, v, sk)
	bias := 0.0
	if v.side {
		bias = -0.05
	}
	clump := pspec().furClump
	f.fill(g.U, s, func(x, y int, _ float64) color.RGBA {
		fy := float64(y) + 0.5
		lv := s.lum(x, y, 6)*0.6 + 0.24 + bias
		if fy < top+3 {
			lv -= 0.16 * (1 - (fy-top)/3) // in the thong's shadow
		}
		return pFurAt(rp.fur, x, y, lv, 59, clump, top+1)
	})
	f.dropSpecks(g.U, f.n, 6)
	if pspec().furTones <= 7 {
		f.pClean(g.U, f.n, 1)
	}
	// The thong: a band round the waist, its edge catching the light.
	var band shape
	switch {
	case v.side:
		band = sPoly(19.8, top-0.6, 34.4, top-0.6, 34.8, top+2, 19.6, top+2)
	default:
		w := pspec().hips
		if female {
			w = 7
		}
		band = sPoly(27-w-0.2, top-0.8, 27+w+0.2, top-0.8, 27+w+0.4, top+1.8, 27-w-0.4, top+1.8)
	}
	f.fill(g.U, band, func(x, y int, _ float64) color.RGBA {
		fy := float64(y) + 0.5
		i := 2
		switch {
		case fy < top+0.2:
			i = 3
		case fy > top+1.2:
			i = 1
		}
		if (x+y)%7 == 0 {
			i-- // a seam where it is laced
		}
		return rp.belt.at(i)
	})
	// The knot, and its two ends hanging.
	out := 1.0
	if kx < 27 {
		out = -1
	}
	for _, e := range [2][3]float64{{0.7, 7.2, 1.15}, {-0.7, 5.4, 1.05}} {
		end := sLock(kx, ky, kx+out*e[0]*1.6, ky+e[1]*0.5, kx+out*e[0], ky+e[1], e[2], 0.55)
		f.fill(g.U, end, func(x, y int, _ float64) color.RGBA {
			return pTone(rp.belt, end.lum(x, y, 1.2)*0.8+0.15, x, y)
		})
	}
	knot := sEll(kx, ky, 1.8, 1.5)
	f.fill(g.U, knot, func(x, y int, _ float64) color.RGBA { return pTone(rp.belt, knot.lum(x, y, 1.5)*0.9+0.2, x, y) })
}

// pFurTop draws a woman's top on U: a band of fur wrapped round the chest
// over the breasts, and a strap of it over one shoulder.
func (g *figGen) pFurTop(d sim.Dir, trunk shape, rp *pRamps) {
	f := &g.f
	v := viewOf(d)
	band := trunk.grow(0.6).and(sBelow(50.2)).and(sAbove(58.8))
	var strap shape
	switch {
	case v.side:
		band = trunk.grow(0.7).and(sBelow(50.2)).and(sAbove(58.8))
		strap = sPoly(24.6, 46, 29.6, 46, 31.4, 51, 26, 51)
	case v.back:
		strap = sPoly(26.6, 51, 21.4, 45.8, 18.8, 46.4, 19.2, 51)
	default:
		strap = sPoly(27.4, 51, 32.6, 45.8, 35.2, 46.4, 34.8, 51)
	}
	top := band.or(strap.and(trunk.grow(0.4))).jag(1.4, 2.4, 56.4, 71)
	th := pTrunkH(true, v, trunk)
	f.fill(g.U, top, func(x, y int, _ float64) color.RGBA {
		fy := float64(y) + 0.5
		lv := pLight(func(x, y float64) float64 { return th(x, y) + edgeH(top.d(x, y), 2) }, x, y)
		if fy > 56.5 {
			lv -= 0.08
		}
		return pFur(rp.fur, x, y, lv*0.9+0.08, 73, 2.3)
	})
	f.dropSpecks(g.U, f.n, 5)
}

// pWarm, while a limb or the head is painted, is how much warmer the skin
// is at a point: the hands and the feet, the elbows and the knees, the
// cheeks, the nose and the ears, where the blood shows (0 to 1).
var pWarm func(x, y float64) float64

// warmNear is 1 at (cx, cy), falling to 0 at r from it.
func warmNear(x, y, cx, cy, r float64) float64 {
	return clamp01(1 - math.Hypot(x-cx, y-cy)/r)
}
