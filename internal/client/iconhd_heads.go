package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of the trophy heads (sim.HeadItem), the dinosaurs and the
// other reptiles: each head in profile, facing right as the beasts do in
// the world, cut at the neck (a ring of red flesh round the pale bone shows
// at the lower left), in the tones and markings of the beast's own art
// (dinos/hd, hellart.go, seaart.go, croc.go, the bosses' files). The
// mammals, the sea's beasts and the small things are in iconhd_beasts.go.

func init() {
	head := func(k sim.CreatureKind, f iconPainter) { regIcon(sim.HeadItem(k), f) }
	head(sim.CreatureRex, icHeadRex)
	head(sim.CreatureRaptor, icHeadDakota)
	head(sim.CreatureAchero, icHeadAchero)
	head(sim.CreatureTroodon, icHeadTroodon)
	head(sim.CreatureCarcharo, icHeadCarcharo)
	head(sim.CreatureGiganoto, icHeadGiganoto)
	head(sim.CreatureSpino, icHeadSpino)
	head(sim.CreatureTrike, func(p *icv) { icHeadCerato(p, false) })
	head(sim.CreatureToro, func(p *icv) { icHeadCerato(p, true) })
	head(sim.CreatureStego, icHeadStego)
	head(sim.CreatureAnkylo, icHeadAnkylo)
	head(sim.CreatureBrachio, icHeadBrachio)
	head(sim.CreatureThesc, icHeadThesc)
	head(sim.CreatureOrnitho, icHeadOrnitho)
	head(sim.CreatureAnzu, icHeadAnzu)
	head(sim.CreaturePtero, func(p *icv) { icHeadPtero(p, false) })
	head(sim.CreatureQuetzal, func(p *icv) { icHeadPtero(p, true) })
	head(sim.CreaturePlesio, icHeadPlesio)
	head(sim.CreatureMosa, icHeadMosa)
	head(sim.CreatureCroc, func(p *icv) { icHeadCroc(p, crocRamp, crocBelly, 1.0, 1.0) })
	head(sim.CreatureBrachy, func(p *icv) {
		icHeadCroc(p, mkRamp(0x17150f, 0x272418, 0x3a3622, 0x504a2e, 0x6a623c, 0x857c4e), mkRamp(0x7c7450, 0xa09868, 0xc4bc8a), 0.78, 1.3)
	})
	head(sim.CreatureChampso, func(p *icv) {
		icHeadCroc(p, mkRamp(0x1d2418, 0x2f3a25, 0x445234, 0x5c6c46, 0x788a5c, 0x97a878), mkRamp(0x8e9070, 0xb2b490, 0xd2d4b0), 1.25, 0.62)
	})
	head(sim.CreatureSnake, func(p *icv) { icHeadSnake(p, snakeRamp, blotchRamp, bellyRamp, false) })
	head(sim.CreatureTitanoboa, func(p *icv) { icHeadSnake(p, tbMoss[3:], tbBronze[3:], tbBelly[4:], true) })
}

// --- the kit ----------------------------------------------------------------------

var (
	icEyeGold = hex(0xf0b830)
	icEyeRed  = hex(0xe04a28)
	icNeckR  = mkRamp(0x4a0e14, 0x7a1c22, 0xa83234, 0xd0584c) // the cut neck
)


// skin shades a head's hide of ramp r: bulged round over round units, a
// paler belly (b, nil for none) below the line belly(x), and marks from
// mark (0 none, 1 full) in ramp dark.
type skin struct {
	r, b, dark ramp
	round      float64
	bias       float64
	belly      func(x float64) float64
	mark       func(x, y float64) float64
}

func (s skin) sh() ishader {
	return func(f *ifrag) color.RGBA {
		l := bulge(f, s.round) + s.bias
		if s.belly != nil && s.b != nil && f.y > s.belly(f.x) {
			return pick(s.b, l+0.1)
		}
		c := pick(s.r, l)
		if s.mark != nil && s.dark != nil {
			if m := s.mark(f.x, f.y); m > 0 {
				c = mix(c, pick(s.dark, l), clamp01(m))
			}
		}
		return c
	}
}

// stripes is a mark of bands across a hide, slanting, every period units.
func stripes(period, slant, width float64, seed uint32) func(x, y float64) float64 {
	return func(x, y float64) float64 {
		u := (x + y*slant + fbm2(x*0.3, y*0.3, seed, 2)*1.4) / period
		if u-math.Floor(u) < width {
			return 0.75
		}
		return 0
	}
}

// spots is a mark of round spots on a jittered grid, cell units apart.
func spots(cell, r, share float64, seed uint32) func(x, y float64) float64 {
	return func(x, y float64) float64 {
		cx, cy := math.Floor(x/cell), math.Floor(y/cell)
		if hashf(int(cx), int(cy), seed) > share {
			return 0
		}
		px := (cx + 0.25 + 0.5*hashf(int(cx), int(cy), seed+1)) * cell
		py := (cy + 0.25 + 0.5*hashf(int(cx), int(cy), seed+2)) * cell
		if math.Hypot(x-px, y-py) < r {
			return 0.9
		}
		return 0
	}
}

// neckCut cuts a neck off at its end (x, y), the neck running from there
// towards (tx, ty), w its half width there: what lies past the cut is
// wiped, and the cut face put in its place, seen a little from below: a
// rim of hide r, the red flesh, the pale bone of the spine.
func neckCut(p *icv, r ramp, x, y, tx, ty, w float64) {
	ux, uy := x-tx, y-ty
	l := math.Hypot(ux, uy)
	ux, uy = ux/l, uy/l
	for iy := 0; iy < p.n; iy++ {
		for ix := 0; ix < p.n; ix++ {
			dx, dy := (float64(ix)+0.5-p.o)/p.k-x, (float64(iy)+0.5-p.o)/p.k-y
			if dx*ux+dy*uy > 0 && math.Hypot(dx, dy) < w*1.9 {
				i := iy*p.n + ix
				p.px[i], p.part[i] = color.RGBA{}, 0
			}
		}
	}
	ra, rb := w*0.45, w
	p.fill(iEll(x, y, ra, rb, math.Atan2(uy, ux)), func(f *ifrag) color.RGBA {
		dx, dy := f.x-x, f.y-y
		a, b := (dx*ux+dy*uy)/ra, (-dx*uy+dy*ux)/rb
		d := math.Hypot(a, b)
		if -f.d < 0.28 {
			return pick(r, 0.42+0.25*toward(f.gx, f.gy)) // the hide's edge
		}
		if math.Hypot(a, b+0.3) < 0.3 {
			return pick(icBoneR, 0.8-d*0.5) // the spine
		}
		return pick(icNeckR, 0.82-d*0.45+0.12*toward(-dx, -dy))
	})
}

// eyeAt paints an eye at (x, y), r big: a dark lid round it, the iris,
// the pupil (a slit if slit), a glint.
func eyeAt(p *icv, x, y, r float64, iris color.RGBA, slit bool) {
	p.on(sEll(x, y, r*1.35, r*1.15), func(f *ifrag) color.RGBA {
		return mix(p.px[f.iy*p.n+f.ix], colInk, 0.55)
	})
	p.fill(sEll(x, y, r, r*0.9), func(f *ifrag) color.RGBA {
		l := 0.75 - (f.y-y)/r*0.25
		return mix(iris, colInk, clamp01(0.35-l*0.4))
	})
	if slit {
		p.on(sEll(x+r*0.1, y, r*0.22, r*0.8), solidC(colInk))
	} else {
		p.on(sEll(x+r*0.12, y, r*0.45, r*0.45), solidC(colInk))
	}
	p.on(sEll(x-r*0.3, y-r*0.35, r*0.26, r*0.2), solidC(icGlint))
}

// brow darkens a crescent of shadow over an eye at (x, y) and lights the
// ridge above it.
func brow(p *icv, x, y, w float64) {
	p.on(iEll(x+0.1, y-w*0.7, w*1.3, w*0.45, -0.1), func(f *ifrag) color.RGBA {
		c := p.px[f.iy*p.n+f.ix]
		if f.y < y-w*0.75 {
			return mix(c, icGlint, 0.18)
		}
		return mix(c, colInk, 0.35)
	})
}

// teeth paints a row of n teeth along the line from (x0, y0) to (x1, y1),
// each size long, pointing down (or up, if up), smaller towards the ends.
func teeth(p *icv, x0, y0, x1, y1 float64, n int, size float64, up bool) {
	dir := 1.0
	if up {
		dir = -1
	}
	for i := 0; i < n; i++ {
		t := (float64(i) + 0.5) / float64(n)
		x, y := lerpf(x0, x1, t), lerpf(y0, y1, t)
		s := size * (0.7 + 0.3*math.Sin(t*math.Pi))
		w := s * 0.42
		p.fill(sPoly(x-w, y, x+w, y, x+w*0.2, y+s*dir), func(f *ifrag) color.RGBA {
			return pick(icToothR, 0.75+0.2*toward(f.gx, f.gy))
		})
	}
}

// mouthLine draws the line of a closed mouth through pts.
func mouthLine(p *icv, w float64, pts ...float64) {
	p.on(sPath(w, w*0.7, pts...), func(f *ifrag) color.RGBA {
		return mix(p.px[f.iy*p.n+f.ix], colInk, 0.7)
	})
}

// nostril is a small dark slot at (x, y).
func nostril(p *icv, x, y, r float64) {
	p.on(iEll(x, y, r, r*0.55, -0.3), func(f *ifrag) color.RGBA {
		return mix(p.px[f.iy*p.n+f.ix], colInk, 0.6)
	})
}

// horn paints a tapering horn from its root (x0, y0) through (cx, cy) to
// its tip (x1, y1), w thick at the root, ringed near the root.
func horn(p *icv, r ramp, x0, y0, cx, cy, x1, y1, w float64) {
	s := sLock(x0, y0, cx, cy, x1, y1, w, 0.06)
	a := mkAxis(x0, y0, x1, y1)
	p.fill(s, func(f *ifrag) color.RGBA {
		u, _ := a.uv(f.x, f.y)
		l := bulge(f, w*0.8) + 0.04
		if u < a.l*0.4 && math.Mod(u*2.2, 1) < 0.15 {
			l -= 0.12
		}
		return pick(r, l)
	})
}

// --- the theropods -------------------------------------------------------------------

// thero is a meat-eater's head: its hide, the depth of its skull, how far
// its snout reaches, its eye and its teeth.
type thero struct {
	sk       skin
	deep     float64 // 1 for a rex
	tip      float64 // the snout's tip, x
	narrow   float64 // 1, less for a long narrow snout
	eye      color.RGBA
	teeth    float64
	open     float64 // how far the jaw hangs open, units
	ridge    bool    // a bony ridge over the eye
	before   func(p *icv)
	after    func(p *icv)
}

func theropod(p *icv, t thero) {
	d := t.deep
	if t.before != nil {
		t.before(p)
	}
	neck := sCap(4.0, 12.6, 6.8, 8.4, 2.5*d, 2.6*d)
	cran := sEll(7.8, 6.4, 3.3*d, 2.9*d)
	snout := sPoly(7.0, 3.8*d+0.8, t.tip-1.2, 5.6+0.4*(1-d), t.tip, 6.4, t.tip-0.2, 7.6+0.4*t.narrow, 7.2, 8.6)
	upper := iBlend(1.4, neck, cran, iBlend(0.8, snout))
	jaw := iBlend(0.8, sPoly(6.6, 8.2, t.tip-1.6, 7.9+t.open, t.tip-1.2, 8.6+t.open+0.6*t.narrow, 9.0, 10.4+t.open*0.5, 6.4, 10.6))
	head := iBlend(1.0, upper, jaw)
	sk := t.sk
	p.fill(head, sk.sh())
	if t.open > 0.3 { // the mouth open: dark inside, a tongue
		mouth := sPoly(7.2, 8.4, t.tip-0.4, 7.2+0.4*t.narrow, t.tip-1.4, 7.9+t.open+0.2)
		p.on(mouth, func(f *ifrag) color.RGBA {
			if f.y > 7.9+t.open*0.6 && f.x < t.tip-2.4 {
				return pick(icNeckR, 0.55)
			}
			return pick(icNeckR, 0.15)
		})
		teeth(p, 7.8, 8.0+0.1, t.tip-0.8, 7.0+0.4*t.narrow, 7, t.teeth, false)
		teeth(p, 8.4, 9.0+t.open*0.75, t.tip-1.8, 7.9+t.open+0.1, 5, t.teeth*0.85, true)
	} else {
		mouthLine(p, 0.14, 7.0, 8.9, 9.6, 8.5, t.tip-0.6, 7.6+0.3*t.narrow)
		teeth(p, 8.0, 8.6, t.tip-1.0, 7.8+0.3*t.narrow, 6, t.teeth, false)
	}
	nostril(p, t.tip-0.9, 6.1, 0.32)
	eyeAt(p, 8.9, 5.3, 0.62*math.Min(1, d+0.15), t.eye, true)
	if t.ridge {
		brow(p, 8.9, 5.3, 0.8)
	}
	neckCut(p, sk.r, 4.0, 12.6, 6.8, 8.4, 2.5*d)
	if t.after != nil {
		t.after(p)
	}
}

// icHeadRex is the tyrant's head: deep and heavy, olive with dark bands,
// a bony brow over a gold eye, a row of great teeth.
func icHeadRex(p *icv) {
	rex := mkRamp(0x2c2a16, 0x4a4626, 0x6a6436, 0x8c8448, 0xaea45e, 0xcac07a)
	theropod(p, thero{
		sk: skin{r: rex, b: mkRamp(0x8a7a52, 0xb0a070, 0xd2c494), dark: mkRamp(0x1e1c10, 0x32301a, 0x4a4626),
			round: 2.6, belly: func(x float64) float64 { return 9.6 - 0.05*x }, mark: stripes(1.9, 0.35, 0.3, 601)},
		deep: 1.12, tip: 15.0, narrow: 1, eye: icEyeGold, teeth: 1.0, open: 1.2, ridge: true,
	})
}

// icHeadDakota is the raptor's head: long and low, grey-blue, a crest of
// orange-red feathers down the back of the head and neck.
func icHeadDakota(p *icv) {
	grey := mkRamp(0x1e2430, 0x2e3644, 0x44505e, 0x5e6c7a, 0x7c8a96, 0xa0acb4)
	feath := mkRamp(0x5a1a0e, 0x8a2c14, 0xb8461e, 0xe06a2a, 0xf49a48)
	theropod(p, thero{
		sk:   skin{r: grey, b: mkRamp(0x8e9aa4, 0xb4bec6, 0xdce2e6), round: 2.2, belly: func(x float64) float64 { return 9.0 }},
		deep: 0.86, tip: 14.8, narrow: 1.2, eye: icEyeGold, teeth: 0.7,
		before: func(p *icv) {
			for i := 0; i < 7; i++ { // the crest: feathers fanned back from the crown
				x, y := 6.4+float64(i)*0.9, 3.6+0.1*float64(i)
				lock(p, feath, x, y, x-1.6, y-1.6, x-3.4, y+0.2-0.4*float64(i%2), 0.75, 0.04*float64(i%3), icFill)
			}
		},
	})
}

// icHeadAchero is the smaller raptor's head: blue-grey, a dark band
// through the eye, a red eye.
func icHeadAchero(p *icv) {
	grey := mkRamp(0x1a2030, 0x283044, 0x3a465c, 0x52627a, 0x6e809a, 0x92a2b8)
	theropod(p, thero{
		sk: skin{r: grey, b: mkRamp(0x9aa6b4, 0xc0cad2, 0xe4e8ec), dark: mkRamp(0x0c1018, 0x141a24, 0x1e2634),
			round: 2.0, belly: func(x float64) float64 { return 9.0 },
			mark: func(x, y float64) float64 {
				if math.Abs(y-5.4-(x-9)*0.1) < 0.7 && x > 6 {
					return 0.85 // the mask through the eye
				}
				return 0
			}},
		deep: 0.8, tip: 14.6, narrow: 1.1, eye: icEyeRed, teeth: 0.7, open: 0.8,
	})
}

// icHeadTroodon is the night hunter's head: small, green, a huge gold eye.
func icHeadTroodon(p *icv) {
	green := mkRamp(0x0e1a0e, 0x1a2c14, 0x28421c, 0x3a5c26, 0x547a32, 0x78a040)
	theropod(p, thero{
		sk: skin{r: green, b: mkRamp(0x7ea83a, 0xa6cc52, 0xcce880), dark: mkRamp(0x0a120a, 0x121e0e),
			round: 2.0, belly: func(x float64) float64 { return 8.8 }, mark: spots(1.4, 0.3, 0.35, 611)},
		deep: 0.78, tip: 14.6, narrow: 1.0, eye: icEyeGold, teeth: 0.6,
		after: func(p *icv) { eyeAt(p, 8.9, 5.3, 1.1, icEyeGold, true) },
	})
}

// icHeadCarcharo is Carcharo's head: long, sand and rust, ridged over the
// eye, its jaws open on rows of blade-like teeth.
func icHeadCarcharo(p *icv) {
	theropod(p, thero{
		sk: skin{r: carchSkin, b: carchBelly, dark: carchRust, round: 2.6,
			belly: func(x float64) float64 { return 9.4 }, mark: stripes(1.6, 0.25, 0.32, 621)},
		deep: 1.05, tip: 15.0, narrow: 1.1, eye: icEyeGold, teeth: 1.1, open: 1.6, ridge: true,
		after: func(p *icv) {
			for i := 0; i < 3; i++ { // the rough bumps along its snout
				p.fill(sEll(10.0+float64(i)*1.5, 4.9+0.25*float64(i), 0.45, 0.32), shBulge(carchRust, 0.3, 0.1))
			}
		},
	})
}

// icHeadGiganoto is the giant's head: grey-black and scarred, a horn of
// bone over its eye, embers in its scars.
func icHeadGiganoto(p *icv) {
	theropod(p, thero{
		sk: skin{r: gigaSkin, b: gigaBelly, dark: gigaEmber, round: 2.6,
			belly: func(x float64) float64 { return 9.6 },
			mark: func(x, y float64) float64 {
				if math.Abs(fbm2(x*0.5, y*0.5, 631, 2)-0.5) < 0.025 {
					return 0.9 // the glowing scars
				}
				return 0
			}},
		deep: 1.12, tip: 15.0, narrow: 1, eye: gigaEmber[4], teeth: 1.1, open: 1.4, ridge: true,
		after: func(p *icv) { horn(p, gigaBelly, 8.2, 4.2, 8.4, 3.0, 9.6, 2.2, 0.7) },
	})
}

// icHeadSpino is the sail-back's head: a long narrow snout like a
// crocodile's, teal with a pale throat, the root of its red sail behind.
func icHeadSpino(p *icv) {
	theropod(p, thero{
		sk:   skin{r: spinoSkin, b: spinoBelly, dark: spinoDark, round: 2.2, belly: func(x float64) float64 { return 9.0 }, mark: spots(1.2, 0.28, 0.4, 641)},
		deep: 0.84, tip: 15.2, narrow: 0.3, eye: icEyeGold, teeth: 0.8,
		before: func(p *icv) { // the front of the sail, over the neck
			sail := sPoly(1.6, 10.6, 2.6, 3.6, 4.0, 2.6, 5.4, 6.4, 5.8, 10.0)
			p.fill(iBlend(0.4, sail), func(f *ifrag) color.RGBA {
				l := 0.5 + 0.3*clamp01((10-f.y)/7)
				if math.Mod(f.x*1.4+20, 1) < 0.16 {
					l -= 0.2 // its spines
				}
				return pick(spinoSail, l)
			})
		},
	})
}

// --- the horned and the armoured ------------------------------------------------------

// icHeadCerato is a three-horn's head (toro: the bigger cousin): the great
// frill behind it, painted with eyes, two long brow horns and a short one
// on the nose, a hooked beak.
func icHeadCerato(p *icv, toro bool) {
	body := mkRamp(0x4a2210, 0x6e3418, 0x9a4e24, 0xc06e34, 0xd88e4c, 0xeab070)
	frill := mkRamp(0x6a1810, 0x9a2a18, 0xc84a1c, 0xe8a030, 0xf8d050)
	if toro {
		body = mkRamp(0x1e2230, 0x2e3446, 0x46506a, 0x60708c, 0x8090aa, 0xa4b2c6)
		frill = mkRamp(0x4a0c10, 0x7a1418, 0xa82424, 0xd04234, 0xe86a4a)
	}
	// The frill, behind everything.
	fx, fy := 5.2, 5.6
	fr := 4.9
	if toro {
		fr = 5.1
	}
	frillS := sEll(fx, fy, fr*0.85, fr)
	p.fill(frillS.jag(0.35, 0.9, -10, 661), func(f *ifrag) color.RGBA {
		d := math.Hypot((f.x-fx)/(fr*0.85), (f.y-fy)/fr)
		l := 0.62 - (f.x-fx)*0.04 - (f.y-fy)*0.05
		if d > 0.86 {
			return pick(frill, l-0.3) // a dark rim
		}
		return pick(frill, l)
	})
	// The eye spots on it.
	for _, s := range [][3]float64{{3.4, 3.6, 1.3}, {5.6, 2.4, 1.0}, {2.8, 7.0, 1.0}} {
		r := s[2]
		p.on(sEll(s[0], s[1], r, r), func(f *ifrag) color.RGBA {
			d := math.Hypot(f.x-s[0], f.y-s[1]) / r
			if toro {
				if d > 0.7 {
					return hex(0xf4efe0)
				}
				return hex(0x1a2a4a)
			}
			if d > 0.62 {
				return hex(0xf8f0d8)
			}
			return frill[0]
		})
	}
	for k := 0; k < 9; k++ { // the knobs round its rim
		a := -2.6 + float64(k)*0.42
		x, y := fx+math.Cos(a)*fr*0.86, fy+math.Sin(a)*fr
		p.fill(sEll(x, y, 0.42, 0.42), shBulge(icToothR, 0.3, 0.05))
	}
	head := iBlend(1.4, sEll(9.0, 9.0, 4.2, 3.8), sPoly(10.0, 6.4, 14.6, 8.6, 15.0, 10.6, 13.2, 12.4, 9.0, 12.6), sCap(6.0, 12.6, 8.0, 10.4, 2.4, 2.4))
	sk := skin{r: body, round: 2.4, mark: stripes(2.4, -0.5, 0.18, 662), dark: body[:3]}
	p.fill(head, sk.sh())
	// The beak.
	beak := sPoly(13.0, 9.2, 15.4, 10.2, 15.2, 11.6, 13.8, 12.6, 12.6, 11.4)
	p.fill(iBlend(0.4, beak), func(f *ifrag) color.RGBA { return pick(beakRamp, bulge(f, 0.8)+0.06) })
	mouthLine(p, 0.12, 9.6, 11.4, 12.0, 11.6, 13.8, 11.4)
	nostril(p, 13.0, 9.6, 0.36)
	// The horns: the two over the eye, then the one on the nose.
	horn(p, icToothR, 9.8, 6.8, 11.0, 3.6, 14.4, 0.8, 0.9)
	horn(p, icToothR, 8.8, 6.6, 9.4, 3.8, 11.8, 1.2, 0.75)
	horn(p, icToothR, 13.2, 8.8, 13.6, 7.6, 13.4, 6.8, 0.55)
	eyeAt(p, 9.6, 8.0, 0.55, icEyeGold, false)
	neckCut(p, body, 6.0, 12.6, 8.0, 10.4, 2.4)
}

// icHeadStego is the plate-back's head: small and low on its neck, green
// with dark bands, the first of its plates rising behind.
func icHeadStego(p *icv) {
	green := mkRamp(0x1a2a10, 0x2a4218, 0x3e5c22, 0x587a2c, 0x76983a, 0x98b44c)
	// The first plates of its back, rising from the top of the neck behind
	// the head: kites of bone under red-brown skin, veined.
	for _, pl := range [][5]float64{{3.8, 11.2, 7.4, 2.0, -1.78}, {6.0, 9.6, 5.8, 1.6, -1.62}, {8.0, 8.4, 3.0, 1.0, -1.35}} {
		x, y, length, width, ang := pl[0], pl[1], pl[2], pl[3], pl[4]
		a := mkAxis(x, y, x+math.Cos(ang)*length, y+math.Sin(ang)*length)
		p.fill(leafShape(x, y, length, width, ang, formLance), func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := 0.52 - v/width*0.22 + 0.18*toward(f.gx, f.gy)*clamp01(1+f.d/0.45)
			if math.Abs(math.Mod(math.Atan2(v, u+1.2)*7+20, 1)-0.5) < 0.1 {
				l -= 0.12 // veins fanning up it
			}
			return pick(plateRamp, l)
		})
	}
	neck := sCap(3.4, 13.0, 7.6, 9.4, 2.6, 2.0)
	head := iBlend(1.2, neck, sEll(9.4, 9.2, 2.6, 2.0), sPoly(9.4, 7.4, 14.8, 9.4, 14.6, 10.8, 9.6, 11.2))
	sk := skin{r: green, round: 2.0, b: mkRamp(0x6e7a38, 0x94a050, 0xb8c272), belly: func(x float64) float64 { return 10.4 + 0.1*(x-8) },
		dark: mkRamp(0x0e1a08, 0x18280e, 0x243a14), mark: stripes(1.7, -0.6, 0.35, 671)}
	p.fill(head, sk.sh())
	mouthLine(p, 0.12, 10.6, 10.6, 12.6, 10.4, 14.6, 10.2)
	nostril(p, 14.0, 9.4, 0.28)
	eyeAt(p, 10.6, 8.8, 0.45, icEyeGold, false)
	neckCut(p, green, 3.4, 13.0, 7.6, 9.4, 2.6)
}

// icHeadAnkylo is the armoured one's head: broad and low, its top a shell
// of bony plates, horns out from the back of its cheeks, a beak.
func icHeadAnkylo(p *icv) {
	hide := mkRamp(0x1e1a16, 0x2e2822, 0x423a30, 0x584e40, 0x726652, 0x8e8068)
	head := iBlend(1.6, sCap(3.8, 12.8, 6.8, 9.6, 2.6, 2.8), sEll(9.4, 8.6, 5.4, 3.6), sPoly(10.0, 6.0, 15.0, 8.4, 15.0, 10.4, 10.4, 12.0))
	sk := skin{r: hide, round: 2.6, b: mkRamp(0x6a5a44, 0x8c7a5e, 0xae9c7c), belly: func(x float64) float64 { return 10.8 }}
	p.fill(head, sk.sh())
	// The plates on its top: knobbed scutes in rows.
	for i, pl := range [][3]float64{{6.4, 6.2, 1.2}, {8.6, 5.4, 1.3}, {10.9, 5.6, 1.2}, {12.9, 6.6, 1.0}, {7.6, 7.8, 1.0}, {10.0, 7.4, 1.0}, {12.2, 8.2, 0.85}} {
		x, y, r := pl[0], pl[1], pl[2]
		p.fill(iEll(x, y, r, r*0.75, 0.2*float64(i%3)), func(f *ifrag) color.RGBA {
			l := litOf((f.x-x)/r*0.8, (f.y-y)/r*0.8) + 0.02
			if fbm2(f.x*2, f.y*2, uint32(681+i), 2) < 0.3 {
				l -= 0.12 // pits
			}
			return pick(osteoPlate, l+0.08)
		})
	}
	// The horns out of the back of the cheek, and over the eye.
	horn(p, osteoBone, 6.6, 9.6, 5.6, 10.6, 4.0, 11.0, 0.8)
	horn(p, osteoBone, 7.8, 5.6, 7.2, 4.4, 6.0, 3.6, 0.7)
	beak := sPoly(13.6, 8.4, 15.4, 9.2, 15.2, 10.6, 13.6, 10.8)
	p.fill(iBlend(0.3, beak), func(f *ifrag) color.RGBA { return pick(osteoBone, bulge(f, 0.6)) })
	mouthLine(p, 0.12, 9.4, 11.0, 12.0, 10.8, 13.8, 10.4)
	eyeAt(p, 10.2, 8.8, 0.45, icEyeGold, false)
	brow(p, 10.2, 8.8, 0.6)
	neckCut(p, hide, 3.8, 12.8, 6.8, 9.6, 2.6)
}

// icHeadBrachio is the long-neck's head: small, a dome of bone over the
// nose, on the top of the long green neck.
func icHeadBrachio(p *icv) {
	green := mkRamp(0x1a2c18, 0x284224, 0x3a5a32, 0x527642, 0x6e9256, 0x8eae6e)
	neck := sLock(3.6, 14.0, 5.0, 8.0, 8.6, 5.6, 2.4, 1.6)
	head := iBlend(1.2, neck, sEll(10.6, 5.4, 3.0, 2.2), sEll(10.0, 3.4, 2.0, 1.8), sPoly(10.6, 4.4, 15.0, 6.0, 14.8, 7.4, 10.6, 7.6))
	sk := skin{r: green, round: 2.0, dark: mkRamp(0x10200e, 0x1a3018, 0x243e20), mark: spots(1.5, 0.45, 0.45, 691),
		b: mkRamp(0x8a9a6a, 0xacbc88, 0xccd8a8), belly: func(x float64) float64 { return 7.0 + (12-x)*0.6 }}
	p.fill(head, sk.sh())
	mouthLine(p, 0.11, 11.0, 6.9, 13.0, 6.8, 14.8, 6.6)
	nostril(p, 11.2, 2.6, 0.32)
	eyeAt(p, 11.6, 4.6, 0.45, icEyeGold, false)
	neckCut(p, green, 3.6, 14.0, 5.0, 8.0, 2.4)
}

// icHeadThesc is the gentle grazer's head: blunt, a beak at the front,
// the face blue, the neck olive with pale spots.
func icHeadThesc(p *icv) {
	head := iBlend(1.4, sCap(3.8, 13.0, 7.0, 9.4, 2.6, 2.4), sEll(9.6, 7.6, 3.6, 2.8), sPoly(10.0, 5.4, 14.6, 7.4, 14.8, 9.6, 10.2, 10.8))
	sk := skin{r: thescRamp, round: 2.4, dark: mkRamp(0xc8d0a8, 0xe0e6c4, 0xf4f6e4), mark: spots(1.2, 0.3, 0.4, 701),
		b: mkRamp(0x7a8458, 0x9aa472, 0xbcc496), belly: func(x float64) float64 { return 10.2 }}
	p.fill(head, sk.sh())
	face := mkRamp(0x1c2e6a, 0x2a4498, 0x3c62c4, 0x5a86e0, 0x88aef0)
	p.on(iEll(11.6, 7.8, 3.8, 2.6, 0.2), func(f *ifrag) color.RGBA {
		return pick(face, bulge(f, 2.0)+0.04) // the blue face
	})
	beak := sPoly(13.4, 7.2, 15.4, 8.4, 15.0, 10.0, 13.2, 9.8)
	p.fill(iBlend(0.3, beak), func(f *ifrag) color.RGBA { return pick(beakRamp, bulge(f, 0.6)) })
	mouthLine(p, 0.12, 10.4, 9.8, 12.4, 9.8, 13.6, 9.6)
	eyeAt(p, 11.0, 6.8, 0.5, icEyeGold, false)
	neckCut(p, thescRamp, 3.8, 13.0, 7.0, 9.4, 2.6)
}

// icHeadOrnitho is the ostrich-dinosaur's head: small on a long bare neck,
// a toothless yellow beak, a big eye.
func icHeadOrnitho(p *icv) {
	pink := mkRamp(0x5a3a3a, 0x845656, 0xaa7a74, 0xc89c94, 0xe2c0b6)
	head := iBlend(1.0, sLock(3.6, 14.0, 4.6, 9.0, 7.6, 6.0, 1.9, 1.3), sEll(9.0, 5.4, 2.4, 2.0))
	// A mane of brown feathers down the back of its neck.
	for i := 0; i < 7; i++ {
		t := 0.08 + 0.1*float64(i)
		x, y := bezAt(t, 3.6, 14.0, 4.6, 9.0, 7.6, 6.0)
		x2, y2 := bezAt(t+0.02, 3.6, 14.0, 4.6, 9.0, 7.6, 6.0)
		tx, ty := x2-x, y2-y
		tl := math.Hypot(tx, ty)
		tx, ty = tx/tl, ty/tl
		nx, ny := ty, -tx // the back of the neck
		w := 1.9 - 0.6*t
		rx, ry := x+nx*w*0.5, y+ny*w*0.5
		lx, ly := 1.5-0.5*t, 1.1
		lock(p, ornithoRamp, rx, ry, rx+nx*lx*0.8-tx*ly*0.3, ry+ny*lx*0.8-ty*ly*0.3, rx+nx*lx-tx*ly, ry+ny*lx-ty*ly, 0.75, 0.05*float64(i%2), icFill)
	}
	p.fill(head, skin{r: pink, round: 1.8}.sh())
	beak := sPoly(10.2, 4.4, 15.0, 6.0, 14.6, 6.8, 10.4, 7.0)
	p.fill(iBlend(0.4, beak), func(f *ifrag) color.RGBA {
		l := bulge(f, 0.8) + 0.06
		if f.y > 6.0+(f.x-10)*0.12 {
			l -= 0.16 // the lower half
		}
		return pick(beakRamp, l)
	})
	eyeAt(p, 9.0, 4.8, 0.55, icEyeGold, false)
	neckCut(p, pink, 3.6, 14.0, 4.6, 9.0, 1.9)
}

// icHeadAnzu is the crested bird-dinosaur's head: dark blue-black, a tall
// red crest, a beak, a red wattle under it.
func icHeadAnzu(p *icv) {
	crest := iBlend(0.8, sEll(9.0, 3.4, 2.2, 3.0), sPoly(7.4, 4.6, 9.0, 0.6, 11.0, 4.6))
	p.fill(crest, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.2) + 0.08
		if math.Abs(math.Sin(f.x*2.2)) < 0.14 {
			l -= 0.12
		}
		return pick(anzuCrest, l)
	})
	head := iBlend(1.2, sLock(3.8, 14.0, 4.6, 9.4, 7.6, 7.0, 2.2, 1.6), sEll(9.4, 6.4, 2.8, 2.4))
	p.fill(head, skin{r: anzuRamp, round: 2.0, bias: 0.1}.sh())
	face := mkRamp(0x6a7890, 0x8a98ae, 0xaab8ca, 0xcad4e0)
	p.on(sEll(10.4, 6.6, 1.8, 1.5), func(f *ifrag) color.RGBA { return pick(face, bulge(f, 1.0)) }) // the bare face
	beak := sPoly(11.4, 5.6, 14.8, 7.0, 13.6, 8.6, 11.4, 8.4)
	p.fill(iBlend(0.4, beak), func(f *ifrag) color.RGBA { return pick(beakRamp, bulge(f, 0.8)+0.04) })
	p.fill(iEll(11.0, 9.4, 1.0, 1.4, 0.2), shBulge(anzuCrest, 0.6, 0.0)) // the wattle
	eyeAt(p, 10.2, 6.2, 0.5, icEyeGold, false)
	neckCut(p, anzuRamp, 3.8, 14.0, 4.6, 9.4, 2.2)
}

// icHeadPtero is a flying reptile's head (quetzal: the giant): a long
// pointed beak, a crest swept back, on a slender neck.
func icHeadPtero(p *icv, quetzal bool) {
	body, crest, beakR := pteroRamp, mkRamp(0x7a2010, 0xb83418, 0xe85a20, 0xf8903a), beakRamp
	bx := 15.2
	if quetzal {
		body = quetzalBody
		crest = mkRamp(0x5a1010, 0x9a1a1a, 0xd03024, 0xf0604a)
	} else {
		body = mkRamp(0x5a4a3a, 0x7e6a52, 0xa48c6c, 0xc4ae8c, 0xe2d0b0)
	}
	// The crest, swept back.
	cr := sPoly(6.2, 5.0, 1.8, 1.4, 3.6, 1.2, 8.4, 3.8)
	if quetzal {
		cr = sPoly(6.6, 4.6, 4.4, 0.8, 6.2, 0.8, 8.8, 3.8)
	}
	p.fill(iBlend(0.5, cr), func(f *ifrag) color.RGBA { return pick(crest, 0.4+0.4*clamp01((6-f.y)/5)+0.1*toward(f.gx, f.gy)) })
	if quetzal {
		p.on(sPoly(5.2, 2.0, 6.0, 1.0, 6.8, 2.4), solidC(hex(0x3c62c4)))
	}
	head := iBlend(1.0, sLock(3.6, 14.2, 4.0, 10.0, 6.2, 6.6, 1.6, 1.3), sEll(7.6, 5.4, 2.2, 1.8))
	p.fill(head, skin{r: body, round: 1.6, bias: 0.04}.sh())
	beak := sPoly(8.2, 4.0, bx, 6.8, bx-0.4, 7.4, 8.4, 6.8)
	p.fill(iBlend(0.3, beak), func(f *ifrag) color.RGBA {
		l := bulge(f, 0.8) + 0.06
		if f.y > 5.5+(f.x-8)*0.22 {
			l -= 0.16
		}
		c := pick(beakR, l)
		if quetzal && f.x > bx-2.0 {
			c = mix(c, crest[2], 0.6) // the red tip
		}
		return c
	})
	mouthLine(p, 0.1, 8.6, 5.9, 11.4, 6.4, bx-0.6, 7.0)
	eyeAt(p, 7.6, 4.8, 0.5, icEyeGold, false)
	neckCut(p, body, 3.6, 14.2, 4.0, 10.0, 1.6)
}

// icHeadPlesio is the sea reptile's head: small and flat, teal above and
// pale below, its teeth showing, on the top of the long neck.
func icHeadPlesio(p *icv) {
	neck := sLock(3.4, 14.2, 4.0, 8.6, 7.6, 6.2, 2.2, 1.6)
	head := iBlend(1.2, neck, sEll(9.4, 5.8, 2.6, 2.0), sPoly(9.4, 4.0, 15.0, 6.0, 14.8, 7.4, 9.4, 8.0))
	sk := skin{r: plesioRamp, round: 2.0, b: mkRamp(0x8aa4a8, 0xb4c8c8, 0xdce8e4), belly: func(x float64) float64 { return 6.8 + (12-x)*0.55 }}
	p.fill(head, sk.sh())
	mouthLine(p, 0.12, 9.4, 7.2, 12.4, 7.0, 14.8, 6.8)
	teeth(p, 10.0, 7.1, 14.4, 6.8, 6, 0.6, false)
	teeth(p, 10.6, 7.2, 14.0, 6.9, 4, 0.5, true)
	nostril(p, 13.6, 5.6, 0.26)
	eyeAt(p, 10.0, 5.0, 0.5, icEyeGold, false)
	glintAt(p, 6.0, 8.0, 0.5, 0.2) // wet
	neckCut(p, plesioRamp, 3.4, 14.2, 4.0, 8.6, 2.2)
}

// icHeadMosa is the Mosasaurus's head: long jaws full of teeth, dark sea
// green above, pale below, wet.
func icHeadMosa(p *icv) {
	head := iBlend(1.4, sCap(3.6, 13.0, 6.8, 9.4, 2.8, 2.6), sEll(8.0, 7.6, 3.2, 2.8), sPoly(8.0, 5.0, 15.2, 7.6, 15.0, 8.8, 8.4, 10.2), sPoly(7.6, 9.6, 14.2, 9.6, 13.8, 10.8, 8.0, 12.0))
	sk := skin{r: mosaRamp, round: 2.4, b: mosaBelly, belly: func(x float64) float64 { return 9.0 }}
	p.fill(head, sk.sh())
	mouth := sPoly(8.0, 9.2, 15.0, 8.4, 14.2, 9.8, 8.4, 9.9)
	p.on(mouth, func(f *ifrag) color.RGBA { return pick(icNeckR, 0.2) })
	teeth(p, 8.6, 9.0, 14.6, 8.4, 9, 0.8, false)
	teeth(p, 8.8, 10.0, 14.0, 9.8, 7, 0.7, true)
	nostril(p, 14.0, 7.6, 0.3)
	eyeAt(p, 9.0, 6.6, 0.55, icEyeGold, true)
	brow(p, 9.0, 6.6, 0.7)
	glintAt(p, 7.0, 5.6, 0.6, 0.22)
	neckCut(p, mosaRamp, 3.6, 13.0, 6.8, 9.4, 2.8)
}

// icHeadCroc is a crocodile's head (long a snout's length, wide its
// breadth): flat and long, scaled, its eye up on top, teeth out over its
// lips.
func icHeadCroc(p *icv, r, belly ramp, long, wide float64) {
	tip := min(15.2, 10.0+5.0*long)
	sw := 1.2 * wide
	head := iBlend(1.2, sCap(3.4, 12.2, 6.4, 9.6, 2.6, 2.4), sEll(7.6, 8.6, 2.8, 2.2), sPoly(7.6, 7.4-sw*0.3, tip, 8.6, tip, 9.6+sw*0.4, 7.6, 10.8))
	sk := skin{r: r, round: 1.8, b: belly, belly: func(x float64) float64 { return 9.6 + 0.04*(x-8) },
		dark: r[:2], mark: func(x, y float64) float64 {
			if math.Mod(x*0.9+20, 1) < 0.12 {
				return 0.6 // the bands of scutes across it
			}
			return 0
		}}
	p.fill(head, sk.sh())
	mouthLine(p, 0.12, 7.4, 9.8, 10.6, 9.6, tip-0.4, 9.4)
	teeth(p, 8.4, 9.6, tip-0.8, 9.4, int(4+4*long), 0.65, false)
	teeth(p, 8.8, 9.8, tip-1.2, 9.6, int(3+3*long), 0.6, true)
	// The eye up on its brow, the nose at the tip.
	p.fill(sEll(8.4, 6.8, 1.0, 0.8), skin{r: r, round: 0.6, bias: 0.06}.sh())
	eyeAt(p, 8.5, 6.8, 0.42, icEyeGold, true)
	p.fill(sEll(tip-0.9, 8.4, 0.6, 0.45), skin{r: r, round: 0.4}.sh())
	nostril(p, tip-0.9, 8.3, 0.22)
	glintAt(p, 6.4, 8.0, 0.5, 0.2)
	neckCut(p, r, 3.4, 12.2, 6.4, 9.6, 2.6)
}

// icHeadSnake is a snake's head (or the Titanoboa's, bigger): the scaled
// head on its coil, the blotches of its back, a forked tongue.
func icHeadSnake(p *icv, r, blotch, belly ramp, titan bool) {
	// The neck rising from the cut, curving up to the head.
	neck := sLock(3.2, 14.2, 3.4, 8.8, 7.0, 7.0, 2.0, 1.8)
	head := iBlend(1.2, neck, sEll(9.4, 6.6, 3.2, 2.2), sPoly(9.4, 4.8, 14.6, 6.2, 14.4, 7.6, 9.4, 8.6))
	sk := skin{r: r, round: 1.8, b: belly, belly: func(x float64) float64 { return 7.8 + (10-x)*0.5 },
		dark: blotch, mark: func(x, y float64) float64 {
			cx, cy := math.Floor(x/2.2), math.Floor(y/2.2)
			px, py := (cx+0.5)*2.2, (cy+0.5)*2.2
			if math.Abs(x-px)+math.Abs(y-py) < 0.8 {
				return 0.85 // a diamond
			}
			return 0
		}}
	p.fill(head, sk.sh())
	// Scales: a fine net over it.
	p.on(head, func(f *ifrag) color.RGBA {
		u, v := f.x*1.6+f.y*0.9, f.x*1.6-f.y*0.9
		if math.Mod(u+20, 1) < 0.08 || math.Mod(v+20, 1) < 0.08 {
			return mix(p.px[f.iy*p.n+f.ix], colInk, 0.22)
		}
		return color.RGBA{}
	})
	mouthLine(p, 0.11, 9.8, 7.7, 12.6, 7.4, 14.4, 7.0)
	// The forked tongue.
	tongue := mkRamp(0x5a0a1a, 0x9a1a30, 0xd03a4a)
	p.fill(sPath(0.14, 0.12, 14.0, 7.2, 15.0, 7.6, 15.4, 7.2), solidC(tongue[1]))
	p.fill(sPath(0.12, 0.1, 15.0, 7.6, 15.4, 8.2), solidC(tongue[1]))
	nostril(p, 13.6, 6.2, 0.22)
	iris := icEyeGold
	if titan {
		iris = hex(0xd0a030)
	}
	eyeAt(p, 10.8, 5.9, 0.5, iris, true)
	brow(p, 10.8, 5.9, 0.6)
	glintAt(p, 8.0, 5.0, 0.5, 0.2)
	neckCut(p, r, 3.2, 14.2, 3.4, 8.8, 2.0)
}
