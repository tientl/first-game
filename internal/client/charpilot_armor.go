package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The armour of the figure (charpilot.go), each set a clear step up from
// the last and each made of what a survivor of the stone age could get:
//
//	hide (Áo da thú):            a laced jerkin of tanned hide, fur at the
//	                             neck and on the shoulders, a dagged hem.
//	bone (Giáp xương):           curved bones lashed over dark hide like
//	                             ribs, a spine of vertebrae, shoulder blades
//	                             for pauldrons; a beast's skull for a cap.
//	agate (Giáp mã não):         polished agate scales in rows bound in
//	                             leather, a gorget, pauldrons of three lames;
//	                             a scaled helm with cheek guards and a crest
//	                             of red quills.
//	tyrant (Giáp răng bạo chúa): rex hide hung with rows of its great teeth,
//	                             its jaws on the shoulders, its skull for a
//	                             helm: the head in its mouth, the teeth round
//	                             the face, embers in its eye sockets.
//	dino hide (Giáp da khủng     green-brown hide set with bony scutes, the
//	long):                       belly's banded scales down the front.
//	fur coat (Áo lông):          a long shaggy coat of fur, open in front,
//	                             a great collar.
//
// Two layers, as tallarmor.go has them: the plate (under the arms and the
// head) and what goes over everything (the shoulders, the collar, the
// helm). Each part is lit as the figure is, from the top left, in its own
// OKLab ramp, its fine lines and outline the figure's (pLines).

// The armour looks beyond the tiers: dinosaur hide and the fur coat, worn
// when no better armour is carried (shownArmor).
const (
	armorDino = sim.Tier(sim.NumTiers) + iota
	armorFur
	numArmorLooks
)

// shownArmor is the armour drawn on p: the best tier carried; dinosaur hide
// (it stops half a blow: sim.DinoArmorBlock) over hide or bone; the fur coat
// when there is no armour at all.
func shownArmor(p protocol.PlayerView) sim.Tier {
	if !pilotOn() {
		return p.Armor
	}
	switch {
	case p.DinoArmor && p.Armor < sim.TierPlatinum:
		return armorDino
	case p.Armor > sim.TierStone:
		return p.Armor
	case p.FurCoat:
		return armorFur
	}
	return sim.TierStone
}

// armorHelmed reports whether armour t has a helm that hides hair standing
// up off the crown (spikes, a bun).
func armorHelmed(t sim.Tier) bool {
	if !pilotOn() {
		return t >= sim.TierDiamond && t < sim.NumTiers
	}
	return t >= sim.TierIron && t <= sim.TierDiamond
}

var (
	aHide    = hueRamp(hideHD, 7, 0.34, 0.82, -0.18, 0.1, 1.1, 0.9)
	aHideDk  = hueRamp(darkHideRamp, 7, 0.2, 0.58, -0.15, 0.08, 1.1, 0.9)
	aBone    = hueRamp(mkRamp(0x6e6248, 0x9c8b68, 0xc9b994, 0xe8dcc0, 0xfbf5e6), 7, 0.46, 0.97, -0.3, 0.1, 1.25, 0.7)
	aTooth   = hueRamp(mkRamp(0x7a6a4e, 0xb8a47c, 0xe2d6b6, 0xfffaee), 6, 0.5, 0.99, -0.25, 0.08, 1.15, 0.6)
	aAgate   = hueRamp(mkRamp(0x260808, 0x4a1410, 0x7a281a, 0xa8462a, 0xd27c4a, 0xf2c8a0), 8, 0.24, 0.9, -0.1, 0.25, 1.15, 0.9)
	aSinew   = hueRamp(mkRamp(0x2e2016, 0x4a3624, 0x6e5438, 0x8e7250), 5, 0.24, 0.6, -0.15, 0.1, 1.1, 1)
	aScale   = hueRamp(mkRamp(0x1a2414, 0x2e3e24, 0x4a5a34, 0x6e7846, 0x9a9a62), 7, 0.24, 0.74, -0.25, 0.25, 1.15, 0.9)
	aBelly   = hueRamp(mkRamp(0x5a5030, 0x8a7c4c, 0xb8a872, 0xdcd09a), 6, 0.42, 0.88, -0.2, 0.1, 1.1, 0.9)
	aFur     = hueRamp(mkRamp(0x2a1e18, 0x4a382a, 0x6e5640, 0x94785a, 0xbca07e, 0xd8c09c), 9, 0.24, 0.86, -0.15, 0.1, 1.1, 0.85)
	aQuill   = hueRamp(mkRamp(0x3a0a08, 0x7a1810, 0xc0301a, 0xec6430, 0xffb070), 6, 0.28, 0.86, -0.1, 0.3, 1.15, 0.9)
	aRexHide = hueRamp(mkRamp(0x18160f, 0x2a261c, 0x403a2a, 0x5a5038, 0x7a6c4a), 7, 0.18, 0.6, -0.2, 0.12, 1.1, 0.9)
	aEmber   = mkRamp(0x5a0c06, 0xb02a0e, 0xff7a2a, 0xffd27a)
)

// aLit is a shader of ramp r over surface h, bias lighter or darker.
func aLit(r ramp, h pHeight, bias float64) shader {
	return func(x, y int, _ float64) color.RGBA { return pClothTone(r, pLight(h, x, y)+bias) }
}

// aSlab is a slab of shape s, rounded over round from its edge.
func aSlab(s shape, round float64) pHeight {
	return func(x, y float64) float64 { return edgeH(s.d(x, y), round) }
}

// armorLayersA draws armour t (a tier, armorDino or armorFur) for a figure
// facing d (never DirLeft), bent by xf: the plate, and what goes over
// everything.
func armorLayersA(t sim.Tier, female bool, d sim.Dir, xf []figMap) (B, H *canvas) {
	g := newLimbGen(sim.Look{Female: female}, 0)
	g.f.xf = xf
	a := &aFit{g: g, f: &g.f, P: g.S, T: g.T, d: d, v: viewOf(d), female: female}
	a.fit()
	switch t {
	case sim.TierCopper:
		a.hideSet()
	case sim.TierIron:
		a.boneSet()
	case sim.TierPlatinum:
		a.agateSet()
	case sim.TierDiamond:
		a.tyrantSet()
	case armorDino:
		a.dinoSet()
	case armorFur:
		a.furSet()
	}
	g.pLines(a.P)
	g.pLines(a.T)
	return a.P.c, a.T.c
}

// aFit is the figure the armour is fitted to: its trunk, its shoulders and
// its head, as the painters drew them.
type aFit struct {
	g      *figGen
	f      *figHD
	P, T   *figLayer
	d      sim.Dir
	v      pView
	female bool

	torso       shape   // the cuirass, neck to hips
	under       pHeight // the body's forms under it
	top, bottom float64 // its first and last rows
	neck        shape   // the hole for the neck
	shoulders   [][3]float64
	// The head: the skull's middle, half width and half height.
	hx, hy, hrx, hry float64
}

func (a *aFit) fit() {
	sp := pspec()
	a.top, a.bottom = 46.5, 79
	if a.female {
		a.bottom = 78
	}
	torso, _ := tallTorso(a.female, a.d, skirt{})
	a.neck = sEll(27, 45.6, 4.3, 3.1)
	if a.v.side {
		a.neck = sEll(27.6, 45.4, 3.6, 2.6)
	}
	if a.female {
		a.neck = sEll(27, 45.6, 3.8, 2.8)
	}
	a.torso = torso.and(sAbove(a.bottom)).sub(a.neck)
	a.under = pTrunkH(a.female, a.v, pTrunk(a.female, a.v))
	switch {
	case a.v.side && a.female:
		a.shoulders = [][3]float64{{27, 49.6, 4.6}}
	case a.v.side:
		a.shoulders = [][3]float64{{27.4, 49.8, 5.6}}
	case a.female:
		a.shoulders = [][3]float64{{17.2, 49.2, 4.4}, {36.8, 49.2, 4.4}}
	default:
		a.shoulders = [][3]float64{{15.4, 49.4, 5.2}, {38.6, 49.4, 5.2}}
	}
	a.hx, a.hy, a.hrx, a.hry = 27, sp.headY, sp.headRX, sp.headRY
	if a.female {
		a.hrx -= sp.fHead
		a.hry -= sp.fHead * 0.6
	}
	if a.v.side {
		a.hx = 26.4
	}
}

// plateH is the cuirass's surface: a slab of its outline over the body's
// forms.
func (a *aFit) plateH(s shape, round, forms float64) pHeight {
	return func(x, y float64) float64 { return edgeH(s.d(x, y), round) + forms*a.under(x, y) }
}

// lat is +1 on the side of the figure x is on (the screen's right), -1 on
// the other; for each shoulder, the way out from the body.
func (a *aFit) out(cx float64) float64 {
	if a.v.side {
		return 1
	}
	if cx < 27 {
		return -1
	}
	return 1
}

// front reports whether column x is on the figure's front.
func (a *aFit) front(x float64) bool { return a.v.front || a.v.side && x >= 27 }

// --- hide -------------------------------------------------------------------

func (a *aFit) hideSet() {
	f, P, T := a.f, a.P, a.T
	plate := a.torso.jag(1.6, 3.8, a.bottom-3, 61)
	h := a.plateH(plate, 5.5, 0.35)
	f.fill(P, plate, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		lv := pLight(h, x, y) + 0.02
		if hashf(x/2, y, 66) < 0.12 {
			lv -= 0.07 // the grain of the hide
		}
		switch {
		case a.v.front && math.Abs(fx-27) < 0.6 && fy < 64: // where the two halves meet
			lv -= 0.3
		case a.v.back && math.Abs(fx-27) < 0.6 && int(fy)%3 != 0: // a seam down the back
			lv -= 0.18
		case fy > a.bottom-6 && fy < a.bottom-5: // a stitched band above the hem
			if x%3 == 0 {
				lv -= 0.22
			} else {
				lv += 0.06
			}
		}
		return pClothTone(aHide, lv)
	})
	f.pClean(P, f.n, 1)
	// The lacing: holes either side of the opening and a thong crossed
	// between them.
	if a.v.front {
		for y := 49.5; y < 63; y += 2.6 {
			for _, s := range [2]float64{-1, 1} {
				f.dot(P, int(27+s*1.9), int(y), aHide[0])
			}
			l := sPath(0.6, 0.6, 25.4, y, 28.6, y+2.6).or(sPath(0.6, 0.6, 28.6, y, 25.4, y+2.6))
			f.fill(P, l, func(x, y int, _ float64) color.RGBA {
				if float64(x)+0.5 > 27 {
					return aSinew[2]
				}
				return aSinew[3]
			})
		}
	}
	if a.v.side { // the opening's edge round the front
		f.fill(P, sBox(32.2, 48, 33.2, 64, 0).and(plate), plain(aHide[1]))
	}
	// A fur collar round the neck.
	collar := a.neck.grow(2.8).sub(a.neck).and(sBelow(45.2)).jag(1, 2.4, 48, 63)
	f.fill(T, collar, func(x, y int, _ float64) color.RGBA {
		return pFurAt(aFur, x, y, collar.lum(x, y, 2)*0.7+0.2, 63, 2.4, 43)
	})
	// Fur on the shoulders.
	for _, s := range a.shoulders {
		pad := sEll(s[0], s[1], s[2]+0.4, 3.8).jag(1.2, 2.6, s[1], 67)
		f.fill(T, pad, func(x, y int, _ float64) color.RGBA {
			return pFurAt(aFur, x, y, pad.lum(x, y, 3)*0.75+0.18, 69, 2.6, s[1]-4)
		})
		f.dropSpecks(T, f.n, 4)
	}
}

// --- bone -------------------------------------------------------------------

func (a *aFit) boneSet() {
	f, P, T := a.f, a.P, a.T
	plate := a.torso.jag(1.4, 3.2, a.bottom-3, 71)
	h := a.plateH(plate, 5, 0.3)
	f.fill(P, plate, aLit(aHideDk, h, 0.04))
	f.pClean(P, f.n, 1)
	bone := func(s shape, round, bias float64) {
		hb := aSlab(s, round)
		f.fill(P, s, aLit(aBone, hb, bias))
	}
	lash := func(x, y float64) { // sinew wound round a joint
		s := sBox(x-1.2, y-1.4, x+1.2, y+1.4, 0.6)
		f.fill(P, s, func(px, py int, _ float64) color.RGBA {
			i := 2
			if (px+py)%2 == 0 {
				i = 1
			}
			if float64(px)+0.5 < x {
				i++
			}
			return aSinew[i]
		})
	}
	switch {
	case a.v.front || a.v.back:
		// Ribs: curved bones out from the middle, four a side; a spine of
		// vertebrae down the middle.
		w := 9.4
		if a.female {
			w = 7.8
		}
		ys := []float64{51.6, 55.8, 60, 64.2}
		if a.v.back {
			ys = []float64{52.4, 57, 61.6}
		}
		for i, y := range ys {
			for _, s := range [2]float64{-1, 1} {
				ww := w - float64(i)*0.5
				rib := sPath(1.45, 1.05, 27+s*1.6, y-0.6, 27+s*ww*0.55, y-0.9, 27+s*ww, y+1.8).and(plate)
				bone(rib, 1.4, 0.06)
			}
		}
		for y := 48.6; y < 68; y += 2.3 {
			v := sEll(27, y, 1.7, 1.05)
			if a.v.back {
				v = sEll(27, y, 1.4, 1.1).or(sEll(27, y-0.3, 2.3, 0.6))
			}
			bone(v, 1.1, 0.1)
		}
		for _, y := range ys {
			lash(27, y)
		}
	case a.v.side:
		for _, y := range []float64{51.6, 55.8, 60, 64.2} {
			rib := sPath(1.45, 1.05, 34.4, y-0.4, 30, y-0.6, 24, y+1.2).and(plate)
			bone(rib, 1.4, 0.06)
			lash(31.2, y)
		}
	}
	// A belt of hide at the waist, bone toggles on it.
	belt := plate.and(sBox(0, 67.4, 60, 70.2, 0))
	f.fill(P, belt, aLit(aSinew, aSlab(belt, 1.4), 0.08))
	if !a.v.back {
		bx := 27.0
		if a.v.side {
			bx = 32
		}
		bone(sCap(bx-2, 68.8, bx+2, 68.8, 0.9, 0.9), 0.9, 0.12)
	}
	// Shoulder blades for pauldrons, lashed on.
	for _, s := range a.shoulders {
		o := a.out(s[0])
		pl := sEll(s[0], s[1]+0.2, s[2]+0.8, 4.2).or(sEll(s[0]+o*1.2, s[1]+2.6, s[2]*0.75, 3)).sub(sEll(s[0]-o*(s[2]+1.2), s[1]+5.4, 3.4, 3))
		hb := func(x, y float64) float64 {
			hh := edgeH(pl.d(x, y), 3)
			// the blade's spine, a ridge across it
			hh += 0.8 * math.Exp(-math.Pow((y-s[1]+0.8-0.25*o*(x-s[0]))/0.8, 2))
			return hh
		}
		f.fill(T, pl, func(x, y int, _ float64) color.RGBA {
			lv := pLight(hb, x, y) + 0.06
			if math.Abs(float64(x)+0.5-(s[0]-o*1.6)) < 0.6 { // the lashing over it
				return aSinew[2+(y%2)]
			}
			return pClothTone(aBone, lv)
		})
	}
	a.skullCap()
}

// skullCap is a beast's skull worn as a cap: its brow ridge on the
// wearer's, its empty sockets above, two short horns, a strap of sinew.
func (a *aFit) skullCap() {
	f, T := a.f, a.T
	hx, hy, rx := a.hx, a.hy, a.hrx
	brow := hy - 4.2
	var cs shape
	switch {
	case a.v.side:
		cs = sEll(hx+0.6, brow-1.4, rx+2, 7.6).and(sAbove(brow + 0.6)).
			or(sPoly(hx+5, brow-6, hx+9.6, brow-3.6, hx+10, brow-0.4, hx+6, brow+0.6)) // the snout's stub over the brow
	case a.v.back:
		cs = sEll(hx, brow-0.6, rx+2, 8.4).and(sAbove(brow + 3.4))
	default:
		cs = sEll(hx, brow-1.4, rx+2, 7.6).and(sAbove(brow + 0.6))
	}
	hc := func(x, y float64) float64 {
		hh := edgeH(cs.d(x, y), 4)
		if !a.v.back && y > brow-2 { // the brow ridge
			hh += 0.8 * clamp01((y-brow+2)/1)
		}
		return hh
	}
	f.fill(T, cs, func(x, y int, _ float64) color.RGBA {
		lv := pLight(hc, x, y) + 0.08
		if !a.v.side && math.Abs(float64(x)+0.5-hx-0.4*math.Sin(float64(y)*1.3)) < 0.55 && float64(y) < brow-2 {
			lv -= 0.3 // the skull's seam
		}
		return pClothTone(aBone, lv)
	})
	socket := func(cx, cy, r float64) {
		s := sEll(cx, cy, r, r*0.8)
		f.fill(T, s, func(x, y int, dd float64) color.RGBA {
			if dd > -0.6 && float64(y)+0.5 < cy {
				return aBone[1]
			}
			return mix(aSinew[0], colInk, 0.4)
		})
	}
	horn := func(x0, y0, x1, y1 float64) {
		s := sLock(x0, y0, (x0+x1)/2+(x1-x0)*0.3, (y0+y1)/2+0.6, x1, y1, 1.5, 0.45)
		f.fill(T, s, aLit(aTooth, aSlab(s, 1.2), 0.1))
	}
	switch {
	case a.v.side:
		socket(hx+5.2, brow-3, 1.5)
		horn(hx+1.6, brow-6.2, hx-0.8, brow-10)
	case a.v.front:
		socket(hx-3.2, brow-3.2, 1.6)
		socket(hx+3.2, brow-3.2, 1.6)
		horn(hx-6.4, brow-4.4, hx-9, brow-8.6)
		horn(hx+6.4, brow-4.4, hx+9, brow-8.6)
	default:
		horn(hx-6.4, brow-4.4, hx-9, brow-8.6)
		horn(hx+6.4, brow-4.4, hx+9, brow-8.6)
	}
	if !a.v.back { // the strap under the jaw
		xs := []float64{hx - rx - 0.6, hx + rx + 0.6}
		if a.v.side {
			xs = []float64{hx + 1.4}
		}
		for _, x := range xs {
			st := sBox(x-0.6, brow, x+0.6, hy+4, 0.3)
			f.fill(T, st, func(px, py int, _ float64) color.RGBA { return aSinew[2-(py%3)/2] })
		}
	}
}

// --- agate ------------------------------------------------------------------

// agateScale shades a polished agate scale: its local coordinates (u
// across, -0.5 to 0.5; v down, 0 to 1), the light lv on it, a seed for its
// bands.
func agateScale(u, v, lv float64, seed int) color.RGBA {
	// a dome, lit from the top left
	lv += 0.42*(0.25-u) + 0.3*(0.4-v)
	// the banding of the stone, rings about a point off its middle
	r := math.Hypot(u-0.12, (v-0.62)*0.9)
	band := math.Sin(r*15 + hashf(seed, 3, 41)*6)
	lv += 0.1 * band
	c := pClothTone(aAgate, lv)
	if u < -0.12 && u > -0.34 && v > 0.18 && v < 0.42 { // the shine of the polish
		c = mix(c, aAgate[7], 0.75)
	}
	return c
}

func (a *aFit) agateSet() {
	f, P, T := a.f, a.P, a.T
	plate := a.torso
	h := a.plateH(plate, 5, 0.3)
	f.fill(P, plate, aLit(aSinew, h, 0.1))
	// Scales in rows, each hanging over the top of the one below, the rows
	// offset by half a scale.
	const sw, sh = 5.4, 4.3
	sc := plate.grow(-0.6).and(sBelow(50.4)).and(sAbove(a.bottom - 3))
	f.fill(P, sc, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		lv := pLight(h, x, y)*0.6 + 0.2
		row := int(math.Floor((fy - 50.4) / sh))
		off := float64(row&1) * sw / 2
		u := (fx+off-27)/sw + 0.5
		col := int(math.Floor(u))
		u -= float64(col) + 0.5
		v := (fy-50.4)/sh - float64(row)
		// the scale's rounded foot: below it is the next row's top
		edge := 0.74 + 0.32*math.Sqrt(math.Max(0, 0.25-u*u))*2
		switch {
		case v > edge:
			return aSinew[0] // the leather between, in the scale's shadow
		case math.Abs(u) > 0.44:
			return aSinew[2] // the lacing between two scales
		}
		return agateScale(u, v, lv, row*31+col)
	})
	// The gorget: a crescent of agate under the neck, a stone set in it.
	if !a.v.back {
		gx := 27.0
		if a.v.side {
			gx = 29.6
		}
		gor := sEll(gx, 46.2, 7, 4.6).sub(sEll(gx, 44.2, 5, 3.4)).and(sBelow(44)).and(plate.grow(0.6))
		f.fill(P, gor, func(x, y int, _ float64) color.RGBA {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			return agateScale((fx-gx)/14, (fy-44)/5.6, 0.62, 7)
		})
		if a.v.front {
			gem := sEll(27, 48.8, 1.5, 1.4)
			f.fill(P, gem, func(x, y int, dd float64) color.RGBA {
				if dd > -0.5 {
					return goldHD[1]
				}
				if float64(x)+0.5 < 27 && float64(y)+0.5 < 48.8 {
					return hex(0xe8f6ff)
				}
				return hex(0x3a8a9a)
			})
		}
	}
	// A leather belt, an agate on its buckle.
	belt := plate.and(sBox(0, 68, 60, 70.6, 0))
	f.fill(P, belt, aLit(aSinew, aSlab(belt, 1.2), 0.14))
	// Pauldrons: three lames of agate, each over the next.
	for _, s := range a.shoulders {
		o := a.out(s[0])
		for j := 2; j >= 0; j-- {
			jj := float64(j)
			cx, cy := s[0]+o*jj*0.7, s[1]-0.8+jj*2.4
			l := sEll(cx, cy, s[2]+0.6-jj*0.2, 3.1-jj*0.3)
			if j > 0 {
				l = l.and(sBelow(cy - 0.6))
			}
			f.fill(T, l, func(x, y int, dd float64) color.RGBA {
				fx, fy := float64(x)+0.5, float64(y)+0.5
				if dd > -0.7 && fy > cy {
					return aSinew[1] // the leather edging
				}
				return agateScale((fx-cx)/(2*s[2]+1.2), (fy-cy+3)/6.2, 0.56-jj*0.06, j*5+int(cx))
			})
		}
	}
	a.agateHelm()
}

// agateHelm is a cap of agate scales on leather, a polished band at the
// brow, guards over the cheeks and a crest of red quills from brow to nape.
func (a *aFit) agateHelm() {
	f, T := a.f, a.T
	hx, hy, rx := a.hx, a.hy, a.hrx
	brow := hy - 3.6
	var dome shape
	switch {
	case a.v.side:
		dome = sEll(hx+0.4, brow-1.2, rx+1.8, 8).and(sAbove(brow + 0.8)).
			or(sPoly(hx-rx-1.6, brow, hx-1, brow, hx-0.4, brow+6.4, hx-rx-1.8, brow+5)) // the guard over the nape and the cheek
	case a.v.back:
		dome = sEll(hx, brow-0.4, rx+1.8, 8.4).and(sAbove(brow + 4.6))
	default:
		dome = sEll(hx, brow-1.2, rx+1.8, 8).and(sAbove(brow + 0.8))
		for _, s := range [2]float64{-1, 1} { // the cheek guards
			dome = dome.or(sPoly(hx+s*(rx+1.6), brow-0.6, hx+s*(rx-0.6), brow-0.6, hx+s*(rx-0.4), brow+6.2, hx+s*(rx+1.2), brow+7.4))
		}
	}
	hd := aSlab(dome, 4)
	f.fill(T, dome, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		lv := pLight(hd, x, y)
		if fy > brow-1.6 && fy < brow+0.9 && !a.v.back { // the band at the brow, polished
			return agateScale((fx-hx)/(2*rx+3.6), 0.4+(fy-brow)*0.12, lv*0.5+0.32, 3)
		}
		if fy > brow+0.9 || a.v.side && fx < hx-1 && fy > brow-1 {
			return pClothTone(aSinew, lv*0.8+0.1) // the leather guards
		}
		// scales over the dome, in rows round it
		row := int(math.Floor((brow - fy) / 2.6))
		off := float64(row&1) * 1.6
		u := (fx+off-hx)/3.2 + 0.5
		col := int(math.Floor(u))
		u -= float64(col) + 0.5
		v := 1 - ((brow-fy)/2.6 - float64(row))
		if v > 0.86 || math.Abs(u) > 0.45 {
			return pClothTone(aSinew, lv*0.6)
		}
		return agateScale(u, v, lv*0.55+0.2, row*17+col)
	})
	// The crest: stiff quills set in a ridge, fanning back.
	var crest shape
	var cx, cy float64 // where the quills spring from
	switch {
	case a.v.side:
		crest = sPoly(hx+5.6, brow-6.2, hx+3, brow-12.6, hx-3.4, brow-14, hx-8.6, brow-10.6, hx-9.4, brow-6.4, hx-5, brow-8, hx+1.6, brow-8.6).
			jag(1, 1.8, 0, 17)
		cx, cy = hx-1, brow-4
	case a.v.back:
		crest = sLock(hx, brow-7.2, hx-0.4, brow-10.4, hx+0.2, brow-14.6, 3.4, 1.6)
		cx, cy = hx, brow-5
	default:
		crest = sLock(hx, brow-7.4, hx+0.4, brow-10.6, hx-0.2, brow-15, 3.2, 1.6)
		cx, cy = hx, brow-5
	}
	f.fill(T, crest, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		ang := math.Atan2(fx-cx, cy-fy) // round the root
		k := math.Floor(ang * 9)
		fr := ang*9 - k
		lv := crest.lum(x, y, 2.4)*0.7 + 0.18 + 0.12*(hashf(int(k), 2, 9)-0.5)
		if fr > 0.72 {
			lv -= 0.2 // between two quills
		}
		return pClothTone(aQuill, lv)
	})
	if a.v.front { // a stone at the brow
		gem := sEll(hx, brow-0.4, 1.4, 1.3)
		f.fill(T, gem, func(x, y int, dd float64) color.RGBA {
			if dd > -0.5 {
				return goldHD[1]
			}
			if float64(x)+0.5 < hx {
				return hex(0xe8f6ff)
			}
			return hex(0x3a8a9a)
		})
	}
}

// --- tyrant -----------------------------------------------------------------

// tooth draws a great tooth on l from its root (x, y), len long, w wide at
// the root, pointing along (dx, dy), curving a little towards (cx, cy).
func (a *aFit) tooth(l *figLayer, x, y, dx, dy, ln, w float64) {
	ux, uy, _ := unitOf(dx, dy)
	px, py := -uy, ux
	tx, ty := x+ux*ln+px*ln*0.18, y+uy*ln+py*ln*0.18
	s := sLock(x, y, x+ux*ln*0.55+px*ln*0.06, y+uy*ln*0.55+py*ln*0.06, tx, ty, w, 0.35)
	a.f.fill(l, s, func(px2, py2 int, _ float64) color.RGBA {
		fx, fy := float64(px2)+0.5, float64(py2)+0.5
		t := clamp01(((fx-x)*ux + (fy-y)*uy) / ln)
		lv := s.lum(px2, py2, w*0.8)*0.8 + 0.2
		if t < 0.18 {
			lv -= 0.3 // the root, stained
		}
		return pClothTone(aTooth, lv)
	})
}

func (a *aFit) tyrantSet() {
	f, P, T := a.f, a.P, a.T
	plate := a.torso.jag(1.2, 3, a.bottom-2.5, 81)
	h := a.plateH(plate, 5, 0.3)
	// Rex hide: pebbled, darker at the edges.
	f.fill(P, plate, func(x, y int, _ float64) color.RGBA {
		lv := pLight(h, x, y)
		u := vnoise(float64(x)*0.7, float64(y)*0.7, 13)
		if u > 0.62 {
			lv += 0.1 // a pebble of the hide catching the light
		} else if u < 0.3 {
			lv -= 0.08
		}
		return pClothTone(aRexHide, lv)
	})
	// A cord across the chest, and another below it, each hung with teeth.
	rows := []struct{ y, sag, ln, step float64 }{{50.2, 2.2, 6.4, 3.3}, {58.6, 2, 5.2, 3}}
	if a.female {
		rows = []struct{ y, sag, ln, step float64 }{{50.4, 1.8, 5.6, 3}, {58, 1.6, 4.6, 2.8}}
	}
	if !a.v.back {
		x0, x1 := 16.6, 37.4
		if a.female {
			x0, x1 = 19, 35
		}
		if a.v.side {
			x0, x1 = 27, 35.6
		}
		for _, r := range rows {
			cord := func(x float64) float64 {
				t := (x - x0) / (x1 - x0)
				if a.v.side {
					t = 0.5 + t*0.5
				}
				return r.y + r.sag*4*t*(1-t)
			}
			cs := sFn(func(x, y float64) float64 { return math.Abs(y-cord(x)) - 0.6 }).and(plate).and(sRightOf(x0)).and(sLeftOf(x1))
			f.fill(P, cs, func(px, py int, _ float64) color.RGBA { return aSinew[2-(px%3)/2] })
			for x := x0 + r.step*0.6; x < x1-r.step*0.4; x += r.step {
				mid := (x - (x0+x1)/2) / (x1 - x0)
				ln := r.ln * (1 - 0.6*mid*mid)
				lean := mid * 2.4
				if a.v.side {
					lean = 0.6
				}
				a.tooth(P, x, cord(x)+0.4, lean, 6, ln, 1.25)
			}
		}
		// the great fang in the middle
		if a.v.front {
			a.tooth(P, 27, rows[1].y+rows[1].sag+0.4, 0, 1, 8.6, 1.9)
		}
	} else { // down the back, a ridge of scutes
		for y := 49.0; y < a.bottom-4; y += 3.4 {
			sp := sPoly(25.4, y+2.6, 27, y-0.6, 28.6, y+2.6).and(plate)
			f.fill(P, sp, aLit(aRexHide, aSlab(sp, 1.2), 0.22))
		}
	}
	// The belt: a rex's lower jaw, its teeth up.
	belt := plate.and(sBox(0, 67.6, 60, 70.4, 0))
	f.fill(P, belt, aLit(aBone, aSlab(belt, 1.3), -0.02))
	if !a.v.back {
		x0, x1 := 19.0, 35.0
		if a.v.side {
			x0, x1 = 28, 34.6
		}
		for x := x0 + 1; x < x1; x += 2.2 {
			a.tooth(P, x, 68, 0, -1, 2.2, 0.8)
		}
	}
	// On the shoulders: the jaws, bristling.
	for _, s := range a.shoulders {
		o := a.out(s[0])
		pad := sEll(s[0], s[1]+0.4, s[2]+1.2, 4.4)
		f.fill(T, pad, func(x, y int, _ float64) color.RGBA {
			lv := pLight(aSlab(pad, 3.4), x, y)
			if vnoise(float64(x)*0.8, float64(y)*0.8, 17) > 0.62 {
				lv += 0.1
			}
			return pClothTone(aRexHide, lv)
		})
		jaw := sPath(1.5, 1.1, s[0]-o*(s[2]+0.6), s[1]-1.4, s[0], s[1]-3.4, s[0]+o*(s[2]+1.6), s[1]-1)
		f.fill(T, jaw, aLit(aBone, aSlab(jaw, 1.2), 0.06))
		n := 3
		if a.v.side {
			n = 4
		}
		for j := 0; j < n; j++ {
			t := (float64(j) + 0.5) / float64(n)
			x := lerpf(s[0]-o*(s[2]+0.2), s[0]+o*(s[2]+1.2), t)
			y := s[1] - 3.4*4*t*(1-t) - 1.2
			a.tooth(T, x, y, o*(t-0.45)*1.6, -1, 3.6+1.2*(1-math.Abs(t-0.5)*2), 1.05)
		}
	}
	a.rexHelm()
}

// rexHelm is a tyrant's skull worn as a helm: the head in its mouth, its
// teeth round the face, its snout over the brow, embers in its sockets.
func (a *aFit) rexHelm() {
	f, T := a.f, a.T
	hx, hy, rx := a.hx, a.hy, a.hrx
	brow := hy - 4.4
	stain := func(x, y int, lv float64) float64 { // old bone: stained and weathered in patches
		if vnoise(float64(x)*0.45, float64(y)*0.45, 71) > 0.64 {
			lv -= 0.1
		}
		return lv
	}
	ember := func(cx, cy float64) {
		f.dot(T, int(math.Floor(cx)), int(math.Floor(cy)), aEmber[2])
		f.dot(T, int(math.Floor(cx))+1, int(math.Floor(cy)), aEmber[1])
	}
	socket := func(cx, cy, rw, rh float64) {
		s := sEll(cx, cy, rw, rh)
		f.fill(T, s, func(x, y int, dd float64) color.RGBA {
			if dd > -0.7 && float64(y)+0.5 < cy {
				return aBone[1]
			}
			return mix(aRexHide[0], colInk, 0.5)
		})
		ember(cx-0.4, cy+0.2)
	}
	horn := func(x0, y0, x1, y1 float64) {
		s := sLock(x0, y0, (x0+x1)/2, (y0+y1)/2, x1, y1, 1.8, 0.5)
		f.fill(T, s, aLit(aBone, aSlab(s, 1.4), 0.04))
	}
	switch {
	case a.v.side:
		// The skull in profile: the cranium over the head, the snout long
		// over the face, the upper jaw's edge above the eyes and its teeth.
		jawY := brow + 0.6
		sk := sPoly(hx-rx-1.4, brow+4.6, hx-rx-2, brow-3, hx-4, brow-9.6, hx+3, brow-10.4, hx+8, brow-8, hx+13.2, brow-5, hx+14.4, brow-2.4,
			hx+13.8, jawY, hx+2, jawY, hx-1.4, brow+5.8)
		hs := func(x, y float64) float64 {
			hh := edgeH(sk.d(x, y), 5)
			hh += 0.9 * math.Exp(-math.Pow((y-(brow-6.8+0.18*(x-hx)))/1, 2)) // the ridge over the eyes and along the snout
			return hh
		}
		f.fill(T, sk, func(x, y int, _ float64) color.RGBA {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			lv := stain(x, y, pLight(hs, x, y)-0.02)
			if fy > jawY-1.2 && fx > hx+1 {
				lv -= 0.14 // the jaw's lip
			}
			if math.Abs(fy-(brow-2.6)-0.12*(fx-hx-6)) < 0.5 && fx > hx+6 && fx < hx+12 {
				lv -= 0.18 // the line of the jaw's bones
			}
			return pClothTone(aBone, lv)
		})
		for x := hx + 2.6; x < hx+13.6; x += 1.7 {
			ln := 2.6
			if int(x)%2 == 0 {
				ln = 3.2
			}
			a.tooth(T, x, jawY-0.4, -0.15, 1, ln, 0.85)
		}
		socket(hx+3.4, brow-4.8, 1.9, 2.2)
		f.fill(T, sEll(hx+12.4, brow-5.2, 0.9, 0.7), plain(mix(aRexHide[0], colInk, 0.4))) // the nostril
		horn(hx+3, brow-7.8, hx+1.4, brow-11.6)
	case a.v.back:
		sk := sSuper(hx, brow+0.4, rx+2.6, 9.6, 2.4).and(sAbove(hy + 4.4))
		holes := sEllAxis(hx-3.4, brow-6.2, 0.5, -0.86, 2.2, 1.1).or(sEllAxis(hx+3.4, brow-6.2, 0.5, 0.86, 2.2, 1.1)) // the openings in the skull's roof
		hs := func(x, y float64) float64 {
			hh := edgeH(sk.d(x, y), 5.5)
			hh += 0.9 * math.Exp(-math.Pow((x-hx)/1.1, 2)) * clamp01((brow+3-y)/3) // the crest between them
			hh -= 0.8 * clamp01(1-holes.d(x, y)/1.4)
			return hh
		}
		f.fill(T, sk, func(x, y int, _ float64) color.RGBA {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if holes.d(fx, fy) < 0 {
				return mix(aRexHide[1], colInk, 0.3)
			}
			return pClothTone(aBone, stain(x, y, pLight(hs, x, y)-0.04))
		})
		horn(hx-rx-0.4, brow-5.4, hx-rx-3.2, brow-9.6)
		horn(hx+rx+0.4, brow-5.4, hx+rx+3.2, brow-9.6)
	default:
		// From the front: the skull over the head and down either side of
		// the face like a hood, the face in its open mouth, the teeth along
		// the jaw round it; the snout's tip over the brow, the sockets wide
		// on the temples.
		mouth := sEll(hx, hy+2.8, rx+1.2, 9.2)
		sk := sSuper(hx, brow-1.4, rx+3.2, 9.8, 2.3).and(sAbove(hy + 5.6)).sub(mouth)
		hs := func(x, y float64) float64 {
			hh := edgeH(sk.d(x, y), 5.5)
			hh += 1.2 * math.Exp(-math.Pow((x-hx)/2.6, 2)) * clamp01((brow+1-y)/2) // the snout's ridge
			return hh
		}
		f.fill(T, sk, func(x, y int, _ float64) color.RGBA {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			lv := stain(x, y, pLight(hs, x, y)-0.02)
			if mouth.d(fx, fy) < 1.2 {
				lv -= 0.2 // the jaw's lip round the face
			}
			return pClothTone(aBone, lv)
		})
		// the teeth round the face: down from the snout over the brow, in
		// from the jaws along the cheeks
		edge := func(y float64) float64 { return (rx + 1.2) * math.Sqrt(math.Max(0, 1-math.Pow((y-hy-2.8)/9.2, 2))) }
		for j, x := 0, hx-4.6; x <= hx+4.7; j, x = j+1, x+1.85 {
			my := hy + 2.8 - 9.2*math.Sqrt(math.Max(0, 1-math.Pow((x-hx)/(rx+1.2), 2)))
			ln := 2.0
			if j%2 == 0 {
				ln = 2.5
			}
			a.tooth(T, x, my-0.5, (hx-x)*0.04, 1, ln, 0.8)
		}
		for _, s := range [2]float64{-1, 1} {
			for y := hy + 0.4; y < hy+5.6; y += 2.1 {
				a.tooth(T, hx+s*(edge(y)+0.5), y, -s, 0.3, 2.1, 0.8)
			}
		}
		for _, s := range [2]float64{-1, 1} {
			socket(hx+s*(rx+0.4), brow-3.2, 1.7, 2.1)
			f.fill(T, sEll(hx+s*1.6, brow-7.4, 0.7, 0.6), plain(mix(aRexHide[0], colInk, 0.4))) // the nostrils
			horn(hx+s*(rx-0.6), brow-6.8, hx+s*(rx+2.2), brow-11)
		}
	}
}

// --- dinosaur hide ----------------------------------------------------------

func (a *aFit) dinoSet() {
	f, P, T := a.f, a.P, a.T
	plate := a.torso.jag(1.2, 3.4, a.bottom-2.5, 91)
	h := a.plateH(plate, 5, 0.3)
	// Scutes in rows: bony ovals in the hide, bigger high on the chest.
	scH := func(x, y float64) (hh float64, row, col int, u, v float64) {
		rh := 3.6
		row = int(math.Floor((y - 48) / rh))
		w := 3.8 + 0.6*math.Max(0, 3-float64(row))*0.4
		off := float64(row&1) * w / 2
		uu := (x+off-27)/w + 0.5
		col = int(math.Floor(uu))
		u = uu - float64(col) - 0.5
		v = (y-48)/rh - float64(row) - 0.5
		q := math.Hypot(u*2.3, v*2.4)
		if q < 1 {
			hh = 1.1 * (1 - q*q)
		}
		return
	}
	hs := func(x, y float64) float64 {
		s, _, _, _, _ := scH(x, y)
		return h(x, y) + s
	}
	belly := 4.2 // the belly's band down the front
	if a.female {
		belly = 3.4
	}
	f.fill(P, plate, func(x, y int, _ float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		if a.front(fx) && (a.v.front && math.Abs(fx-27) < belly || a.v.side && fx > 32.4) && fy > 48 {
			// banded scales across the belly
			lv := pLight(h, x, y)*0.8 + 0.12
			b := math.Mod(fy-48, 2.6)
			if b < 0.8 {
				lv -= 0.24
			} else if b < 1.6 {
				lv += 0.08
			}
			return pClothTone(aBelly, lv)
		}
		s, row, col, _, _ := scH(fx, fy)
		lv := pLight(hs, x, y)
		if s > 0 {
			lv += 0.08
			if hashf(row, col, 5) < 0.3 {
				lv += 0.06
			}
		} else {
			lv -= 0.1
		}
		return pClothTone(aScale, lv)
	})
	f.pClean(P, f.n, 1)
	// Shoulders: a plate of hide with three big scutes and a spike.
	for _, s := range a.shoulders {
		o := a.out(s[0])
		pad := sEll(s[0], s[1]+0.2, s[2]+0.8, 4.2)
		hp := func(x, y float64) float64 {
			hh := edgeH(pad.d(x, y), 3)
			for j := -1.0; j <= 1; j++ {
				if q := math.Hypot(x-(s[0]+j*o*2.6), (y-(s[1]-0.4+math.Abs(j)*0.8))*1.1) / 1.6; q < 1 {
					hh += 1.2 * (1 - q*q)
				}
			}
			return hh
		}
		f.fill(T, pad, aLit(aScale, hp, 0.04))
		spike := sPoly(s[0]+o*0.6-1.3, s[1]-2.6, s[0]+o*1.8, s[1]-7.4, s[0]+o*0.6+1.3, s[1]-2.4)
		f.fill(T, spike, aLit(aBelly, aSlab(spike, 1), 0.1))
	}
}

// --- fur coat ---------------------------------------------------------------

func (a *aFit) furSet() {
	f, P, T := a.f, a.P, a.T
	hem := 84.5
	if a.female {
		hem = 87
	}
	torso, _ := tallTorso(a.female, a.d, skirt{})
	var coat shape
	switch {
	case a.v.side:
		coat = torso.or(sPoly(19.2, 48, 33, 48, 34.6, hem, 18.4, hem)).and(sAbove(hem + 2)).and(sLeftOf(31.6).or(sAbove(52)))
	case a.v.back:
		coat = torso.or(sPoly(15.6, 48, 38.4, 48, 39.2, hem, 14.8, hem)).and(sAbove(hem + 2))
	default:
		open := sPoly(25.6, 46, 28.4, 46, 30.2, hem+3, 23.8, hem+3) // open down the front
		coat = torso.or(sPoly(15.6, 48, 38.4, 48, 39.2, hem, 14.8, hem)).and(sAbove(hem + 2)).sub(open)
	}
	coat = coat.sub(a.neck).jag(2, 2.8, hem-2, 97)
	f.fill(P, coat, func(x, y int, _ float64) color.RGBA {
		lv := coat.lum(x, y, 5)*0.7 + 0.26
		return pFurAt(aFur, x, y, lv, 97, 3.4, 47)
	})
	f.dropSpecks(P, f.n, 5)
	// Belted at the waist with a thong of hide.
	belt := coat.and(sBox(0, 67.6, 60, 69.8, 0))
	f.fill(P, belt, aLit(aSinew, aSlab(belt, 1), 0.12))
	// The collar, a thick roll of fur round the neck and over the shoulders.
	collar := a.neck.grow(3.2).sub(a.neck).and(sBelow(42.8)).or(sBox(18, 45.2, 36, 50.4, 2.6))
	switch {
	case a.v.side:
		collar = a.neck.grow(3.2).sub(a.neck).and(sBelow(42.8)).or(sBox(21, 45.2, 34, 50.6, 2.6))
	case a.female:
		collar = collar.or(sBox(19.6, 45.2, 34.4, 50, 2.4))
	}
	for _, s := range a.shoulders {
		collar = collar.or(sEll(s[0], s[1]+0.4, s[2]+1.4, 4.4))
	}
	collar = collar.jag(1.4, 2.4, 50, 99)
	f.fill(T, collar, func(x, y int, _ float64) color.RGBA {
		lv := collar.lum(x, y, 4)*0.75 + 0.2
		return pFurAt(aFur, x, y, lv, 99, 2.6, 43)
	})
	f.dropSpecks(T, f.n, 5)
	if !a.v.back { // a horn toggle at the throat
		tx := 27.0
		if a.v.side {
			tx = 31.6
		}
		tg := sCap(tx-1.6, 49.6, tx+1.6, 49.2, 0.8, 0.7)
		f.fill(T, tg, aLit(aTooth, aSlab(tg, 0.8), 0.1))
	}
}
