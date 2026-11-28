package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of the other trophy heads: the mammals (rabbit, mammoth and
// the mammoth king, sabre-tooth, cave bear, monkey), the Megalodon, and the
// heads of the beasts with no neck to cut (crab, spider, scorpion, giant
// squid), which show their front. As in iconhd_heads.go: in profile facing
// right, the neck cut at the lower left, in the beast's own tones.

func init() {
	head := func(k sim.CreatureKind, f iconPainter) { regIcon(sim.HeadItem(k), f) }
	head(sim.CreatureRabbit, icHeadRabbit)
	head(sim.CreatureCrab, icHeadCrab)
	head(sim.CreatureMammoth, func(p *icv) { icHeadMammoth(p, false) })
	head(sim.CreatureMammothKing, func(p *icv) { icHeadMammoth(p, true) })
	head(sim.CreatureSabre, icHeadSabre)
	head(sim.CreatureSpider, icHeadSpider)
	head(sim.CreatureBear, icHeadBear)
	head(sim.CreatureKraken, icHeadKraken)
	head(sim.CreatureScorpion, icHeadScorpion)
	head(sim.CreatureMonkey, icHeadMonkey)
	head(sim.CreatureMegalodon, icHeadMegalodon)
}

// shFur shades fur of ramp r: bulged round over round units, strands
// running along (dx, dy), a paler ramp b below the line belly(x) (b nil for
// none).
func shFur(r, b ramp, belly func(x float64) float64, round, bias float64, dx, dy float64, seed uint32) ishader {
	return func(f *ifrag) color.RGBA {
		l := bulge(f, round) + bias
		u, v := f.x*dx+f.y*dy, -f.x*dy+f.y*dx
		n := fbm2(u*0.6, v*2.6, seed, 2)
		if n > 0.64 {
			l += 0.1
		} else if n < 0.34 {
			l -= 0.12
		}
		if b != nil && belly != nil && f.y > belly(f.x) {
			return pick(b, l+0.08)
		}
		return pick(r, l)
	}
}

// fringe paints locks of fur of ramp r hanging from the points (x, y
// pairs), each len long down and back (to the lower left).
func fringe(p *icv, r ramp, length, w float64, pts ...float64) {
	for i := 0; i+1 < len(pts); i += 2 {
		x, y := pts[i], pts[i+1]
		k := float64(i / 2)
		lock(p, r, x, y, x-length*0.15, y+length*0.6, x-length*0.45-0.2*math.Mod(k, 2), y+length, w, 0.04*math.Mod(k, 3)-0.04, icFill)
	}
}

// --- the mammals ----------------------------------------------------------------

// icHeadRabbit is a rabbit's head: soft brown, its long ears laid back, a
// pale chin, a big dark eye, a pink nose and whiskers.
func icHeadRabbit(p *icv) {
	inner := mkRamp(0xa86a6a, 0xd09090, 0xeab4ac)
	ear := func(x0, y0, x1, y1, w, bias float64) {
		a := mkAxis(x0, y0, x1, y1)
		s := sLock(x0, y0, (x0+x1)/2-0.4, (y0+y1)/2, x1, y1, w, w*0.55).grow(0.1)
		p.fill(s, func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := bulge(f, w*0.8) + bias
			if math.Abs(v) < w*0.45 && u > 0.8 && u < a.l-0.5 && -f.d > 0.3 {
				return pick(inner, l) // the pink inside
			}
			return pick(furRamp, l)
		})
	}
	ear(8.0, 6.2, 4.6, 0.9, 1.2, -0.08) // the far ear
	neck := sCap(5.0, 13.2, 7.8, 9.8, 2.3, 2.4)
	head := iBlend(1.4, neck, sEll(9.6, 8.2, 3.4, 2.8), sEll(12.6, 9.0, 2.0, 1.7)).rough(0.12, 0.4, 1101)
	p.fill(head, shFur(furRamp, paleRamp, func(x float64) float64 { return 9.8 + 1.1*math.Max(0, 10.4-x) }, 2.4, 0.04, 0.8, 0.6, 1103))
	ear(9.6, 6.0, 7.0, 0.8, 1.3, 0.02) // the near ear
	// The face: the eye, the nose and the split lip, whiskers.
	eyeAt(p, 10.6, 7.4, 0.72, hex(0x3a1e10), false)
	p.on(sEll(14.4, 8.6, 0.42, 0.34), solidC(hex(0xd88888)))
	mouthLine(p, 0.1, 14.2, 9.0, 14.0, 9.8, 13.4, 10.2)
	for _, w := range [][4]float64{{13.6, 9.2, 15.6, 8.4}, {13.6, 9.5, 15.6, 9.8}, {13.4, 9.8, 15.2, 11.0}} {
		p.on(sPath(0.06, 0.04, w[0], w[1], w[2], w[3]), solidC(paleRamp[2]))
		p.fill(sPath(0.06, 0.04, (w[0]+w[2])/2+0.6, (w[1]+w[3])/2, w[2], w[3]).sub(head), solidC(paleRamp[2]))
		p.bareLast()
	}
	neckCut(p, furRamp, 5.0, 13.2, 7.8, 9.8, 2.3)
}

// icHeadMammoth is a mammoth's head (king: the mammoth king's, black with
// moss on its crown and greater tusks): the high domed skull, the small
// ear, shaggy hair hanging from its cheeks, the trunk curled under and a
// tusk sweeping forward and up.
func icHeadMammoth(p *icv, king bool) {
	fur, tusk := mammothRamp, tuskRamp
	tw := 1.0
	if king {
		fur, tusk, tw = kingFur[1:], kingIvory, 1.25
	}
	tuskS := func(dx, dy, w float64) shape {
		return sTaper(w, w*0.85, 0.2, bezPts(8, 10.4+dx, 10.0+dy, 12.6+dx, 16.4+dy, 15.4+dx, 10.2+dy)...)
	}
	shTusk := func(bias float64) ishader {
		return func(f *ifrag) color.RGBA {
			l := bulge(f, 0.7*tw) + bias
			if math.Mod(f.x*1.6+f.y*0.6+20, 1) < 0.12 {
				l -= 0.1 // growth lines
			}
			return pick(tusk, l)
		}
	}
	p.fill(tuskS(-1.3, -0.5, 0.8*tw), shTusk(-0.2)) // the far tusk
	neck := sCap(4.4, 12.4, 6.8, 8.8, 3.0, 3.0)
	skull := iBlend(1.6, neck, sEll(8.4, 6.6, 3.9, 3.6), sEll(7.4, 3.8, 2.8, 2.4), sEll(10.8, 7.6, 1.9, 2.2)).rough(0.18, 0.45, 1201)
	p.fill(skull, shFur(fur, nil, nil, 2.6, 0.02, 0.35, 0.94, 1203))
	// The shaggy hair hanging from its cheek and throat: of the skull, so no
	// line parts them from it.
	for i, h := range [][3]float64{{5.4, 9.6, 2.4}, {6.4, 10.0, 3.0}, {7.4, 10.2, 2.6}, {8.4, 10.0, 3.0}, {9.4, 9.8, 2.2}, {5.9, 10.6, 2.2}, {7.9, 10.8, 2.4}} {
		x, y, length := h[0], h[1], h[2]
		lock(p, fur, x, y, x-0.3, y+length*0.6, x-0.9+0.2*float64(i%2), y+length, 0.62, -0.06+0.05*float64(i%3), icAdd)
	}
	if king { // moss and lichen on its crown
		p.on(sEll(7.0, 3.4, 2.6, 1.6).rough(0.5, 0.6, 1205).or(sEll(9.6, 5.0, 1.0, 0.7).rough(0.3, 0.4, 1206)), func(f *ifrag) color.RGBA {
			return pick(kingMoss, 0.45+0.3*toward(f.gx, f.gy)*clamp01(1+f.d/0.4)+(fbm2(f.x*2.5, f.y*2.5, 1207, 2)-0.5)*0.4)
		})
	}
	// The ear.
	p.fill(sEll(5.8, 7.4, 1.3, 1.7).rough(0.15, 0.4, 1209), shFur(fur, nil, nil, 0.9, -0.1, 0, 1, 1211))
	// The trunk, hanging from the face and curled under.
	trunk := sTaper(2.0, 1.35, 0.7, 11.4, 7.6, 13.0, 9.8, 13.4, 12.4, 12.6, 14.4, 11.4, 14.2)
	p.fill(trunk, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.2) + 0.06
		if math.Mod(f.y*2.4+f.x*0.3+20, 1) < 0.16 && f.y > 9.4 {
			l -= 0.14 // its rings
		}
		return pick(fur, l)
	})
	eyeAt(p, 10.0, 6.4, 0.42, hex(0x2a1a10), false)
	brow(p, 10.0, 6.4, 0.6)
	p.fill(tuskS(0, 0, 1.0*tw), shTusk(0.04)) // the near tusk
	glintAt(p, 12.4, 12.9, 0.5, 0.16)
	neckCut(p, fur, 4.4, 12.4, 6.8, 8.8, 3.0)
}

// icHeadSabre is the sabre-tooth's head: a tawny cat's, a cream muzzle and
// chin, faint dark stripes, a gold eye, and the two long sabres.
func icHeadSabre(p *icv) {
	ear := sPoly(6.0, 5.4, 6.4, 2.2, 8.6, 4.4)
	p.fill(iBlend(0.6, ear), func(f *ifrag) color.RGBA {
		if -f.d > 0.45 && f.y > 3.6 {
			return pick(sabreDark, 0.5) // the dark inside
		}
		return pick(sabreFur, bulge(f, 0.6)+0.04)
	})
	neck := sCap(4.8, 12.8, 7.2, 9.4, 2.6, 2.6)
	head := iBlend(1.4, neck, sEll(8.8, 7.0, 3.6, 3.0), sEll(12.2, 8.6, 2.4, 1.9), sEll(10.6, 10.4, 2.2, 1.2)).rough(0.12, 0.4, 1301)
	p.fill(head, func(f *ifrag) color.RGBA {
		sh := shFur(sabreFur, sabreCream, func(x float64) float64 { return 9.6 + 1.2*math.Max(0, 10.2-x) }, 2.4, 0.02, 0.9, 0.4, 1303)
		c := sh(f)
		if f.x > 10.8 && f.y > 8.7+0.3*(f.x-12) { // the lower muzzle and chin, cream
			return pick(sabreCream, bulge(f, 1.4)+0.04)
		}
		if math.Mod(f.x*0.8-f.y*0.5+20, 1) < 0.14 && f.x < 10.0 && f.y < 9.2 {
			c = mix(c, sabreDark[2], 0.5) // stripes
		}
		return c
	})
	p.on(sEll(14.0, 7.8, 0.55, 0.42), shBulge(mkRamp(0x2a1a14, 0x4a302a, 0x6a4a40), 0.3, 0)) // the nose
	mouthLine(p, 0.11, 10.4, 9.8, 12.4, 9.9, 13.8, 9.4)
	// The sabres: long, curved, flattened blades.
	for _, sb := range [][2]float64{{-0.9, -0.15}, {0, 0.04}} {
		dx, bias := sb[0], sb[1]
		s := sTaper(0.62, 0.5, 0.06, 12.4+dx, 9.4, 12.6+dx, 11.6, 12.2+dx, 13.4, 11.4+dx, 14.8)
		p.fill(s, func(f *ifrag) color.RGBA { return pick(sabreIvory, bulge(f, 0.4)+bias) })
	}
	eyeAt(p, 10.2, 6.8, 0.52, icEyeGold, true)
	brow(p, 10.2, 6.8, 0.65)
	neckCut(p, sabreFur, 4.8, 12.8, 7.2, 9.4, 2.6)
}

// icHeadBear is the cave bear's head: big and round, dark brown, a paler
// muzzle, a small round ear, its lip curled back from a fang.
func icHeadBear(p *icv) {
	p.fill(sEll(6.6, 3.8, 1.5, 1.4).rough(0.12, 0.35, 1401), shFur(bearRamp, nil, nil, 1.0, 0.0, 0, 1, 1403)) // the ear
	p.on(sEll(6.7, 4.0, 0.7, 0.65), solidC(bearRamp[1]))
	neck := sCap(4.4, 12.8, 6.8, 9.2, 3.0, 3.0)
	head := iBlend(1.6, neck, sEll(8.6, 7.2, 4.0, 3.6), sEll(12.4, 9.0, 2.6, 1.9)).rough(0.18, 0.45, 1405)
	p.fill(head, shFur(bearRamp, nil, nil, 2.8, 0.04, 0.85, 0.5, 1407))
	p.on(iEll(12.6, 9.0, 2.6, 1.8, 0.08).rough(0.1, 0.4, 1409), func(f *ifrag) color.RGBA {
		return pick(bearMuzzle, bulge(f, 1.2)+0.0) // the muzzle
	})
	p.fill(iEll(14.6, 8.4, 0.75, 0.6, -0.2), shBulge(mkRamp(0x0e0a08, 0x1e1612, 0x3a2e26, 0x5a4a40), 0.4, 0.04)) // the nose
	p.on(sEll(14.9, 8.6, 0.16, 0.12), solidC(colInk))
	mouthLine(p, 0.12, 10.6, 10.4, 12.4, 10.6, 14.2, 9.8)
	p.fill(sPoly(12.2, 10.4, 12.9, 10.4, 12.6, 11.6), func(f *ifrag) color.RGBA { return pick(bearClawRamp, 0.8+0.2*toward(f.gx, f.gy)) }) // a fang
	eyeAt(p, 10.4, 6.8, 0.42, hex(0x4a2410), false)
	brow(p, 10.4, 6.8, 0.6)
	neckCut(p, bearRamp, 4.4, 12.8, 6.8, 9.2, 3.0)
}

// icHeadMonkey is a monkey's head, turned to face us: round and brown,
// a pale heart-shaped face, bright eyes under a brow, a jutting muzzle,
// round ears out at the sides, a tuft on the crown.
func icHeadMonkey(p *icv) {
	for _, s := range []float64{-1, 1} { // the ears
		x := 8.0 + s*4.6
		p.fill(sEll(x, 7.6, 1.5, 1.7), func(f *ifrag) color.RGBA {
			if -f.d > 0.4 {
				return pick(monkeyFace, 0.38+0.25*toward(f.x-x, f.y-7.6))
			}
			return pick(monkeyFace, bulge(f, 0.5)+0.06)
		})
	}
	neck := sCap(8.0, 14.0, 8.0, 10.6, 2.6, 2.8)
	head := iBlend(1.4, neck, sEll(8.0, 7.4, 4.2, 4.0)).rough(0.16, 0.42, 1501)
	p.fill(head, shFur(monkeyRamp, nil, nil, 2.8, 0.08, 0.2, 1, 1503))
	for i, t := range [][4]float64{{6.6, 4.0, 5.8, 2.4}, {8.0, 3.6, 8.1, 2.0}, {9.4, 4.0, 10.4, 2.6}} { // the crown's tuft
		lock(p, monkeyRamp, t[0], t[1], (t[0]+t[2])/2, t[3]+0.6, t[2], t[3], 0.6, 0.04*float64(i%2), icAdd)
	}
	// The bare face: round about the eyes, the muzzle below.
	face := iBlend(1.2, sEll(6.6, 7.4, 2.0, 1.9), sEll(9.4, 7.4, 2.0, 1.9), sEll(8.0, 9.8, 2.4, 1.9))
	p.fill(face, func(f *ifrag) color.RGBA { return pick(monkeyFace, bulge(f, 1.4)+0.08) })
	p.fill(sEll(8.0, 10.4, 1.8, 1.3), func(f *ifrag) color.RGBA { return pick(monkeyFace, bulge(f, 1.0)+0.12) }) // the muzzle
	p.on(sPath(0.16, 0.1, 5.2, 5.8, 6.6, 5.2, 8.0, 5.8, 9.4, 5.2, 10.8, 5.8), func(f *ifrag) color.RGBA { return pick(monkeyRamp, 0.25) }) // the brow
	eyeAt(p, 6.7, 7.0, 0.62, hex(0x8a4a18), false)
	eyeAt(p, 9.3, 7.0, 0.62, hex(0x8a4a18), false)
	nostril(p, 7.5, 9.8, 0.24)
	nostril(p, 8.5, 9.8, 0.24)
	mouthLine(p, 0.11, 6.8, 11.0, 8.0, 11.4, 9.2, 11.0)
	neckCut(p, monkeyRamp, 8.0, 14.0, 8.0, 10.6, 2.6)
}

// --- the sea ------------------------------------------------------------------

// icHeadMegalodon is the Megalodon's head: slate blue over a white belly,
// its jaws open on rows of great triangular teeth, a black eye, gill slits
// by the cut.
func icHeadMegalodon(p *icv) {
	// The lower jaw, dropped open.
	jaw := iBlend(0.8, sPoly(5.6, 10.6, 12.6, 10.6, 13.4, 11.4, 11.6, 13.2, 6.4, 13.4))
	p.fill(jaw, shFur(megaBelly, nil, nil, 1.4, -0.06, 1, 0, 1601))
	head := iBlend(1.8, sCap(3.0, 8.4, 7.6, 7.8, 3.8, 3.4), sEll(10.2, 7.2, 3.8, 2.6), sPoly(9.0, 5.0, 15.4, 6.8, 14.6, 8.6, 9.0, 10.4))
	p.fill(head, skin{r: megaSkin[1:], b: megaBelly, round: 2.6, belly: func(x float64) float64 { return 8.6 + 0.2*(x-6) - 0.05*math.Max(0, x-12)*4 }}.sh())
	// The mouth: dark inside, the gums, the teeth.
	mouth := sPoly(6.6, 9.6, 14.0, 8.4, 12.8, 11.0, 7.0, 11.6)
	p.fill(mouth, func(f *ifrag) color.RGBA {
		if f.y > 10.6 {
			return pick(icNeckR, 0.5)
		}
		return pick(icNeckR, 0.12)
	})
	teeth(p, 7.6, 9.4, 13.6, 8.5, 6, 1.1, false)
	teeth(p, 7.6, 11.4, 12.6, 10.9, 5, 1.0, true)
	// The eye, the snout's pores, the gills.
	p.on(sEll(11.2, 6.2, 0.5, 0.5), solidC(colInk))
	p.on(sEll(11.05, 6.05, 0.14, 0.14), solidC(icGlint))
	for i := 0; i < 4; i++ {
		x := 13.0 + 0.5*float64(i)
		p.on(sEll(x, 6.6+0.15*float64(i), 0.08, 0.08), solidC(megaSkin[1]))
	}
	for i := 0; i < 3; i++ {
		x := 5.2 + 0.9*float64(i)
		p.on(sLock(x, 5.6, x+0.5, 7.8, x, 10.0, 0.12, 0.1), solidC(megaSkin[0]))
	}
	glintAt(p, 8.6, 5.0, 0.8, 0.2)
	neckCut(p, megaSkin[1:], 3.0, 8.4, 7.6, 7.8, 3.8)
}

// icHeadKraken is the giant squid's head: its mantle rising behind, a
// great gold eye, the stumps of its arms coiling below, pale suckers.
func icHeadKraken(p *icv) {
	// The arms, behind the head: thick, tapering, curled.
	arm := func(x0, y0, cx, cy, x1, y1, w float64, bias float64) {
		s := sLock(x0, y0, cx, cy, x1, y1, w, w*0.35)
		p.fill(s, func(f *ifrag) color.RGBA {
			l := bulge(f, w*0.8) + bias
			// Suckers along the underside.
			if f.gy > 0.3 && math.Mod(f.x*1.6+f.y*1.6+20, 1) < 0.45 && -f.d < w*0.55 {
				return pick(krakenPale, l)
			}
			return pick(krakenRamp, l)
		})
	}
	arm(9.0, 10.4, 9.0, 14.6, 6.0, 14.4, 1.3, -0.08)
	arm(10.6, 10.6, 12.4, 15.0, 14.8, 13.0, 1.3, -0.04)
	arm(9.8, 10.8, 10.4, 14.4, 11.6, 15.0, 1.4, 0.02)
	arm(11.6, 9.8, 15.4, 11.0, 15.0, 8.4, 1.15, 0.0)
	// The mantle, a long cone up and back, its fins at the tip.
	fins := iBlend(0.6, iEll(3.0, 2.6, 2.4, 1.1, -0.4), iEll(2.4, 4.2, 2.2, 0.9, 0.9))
	p.fill(fins, shFur(krakenRamp, nil, nil, 0.8, -0.06, 0.7, 0.7, 1701))
	mantle := iBlend(1.4, sLock(9.4, 8.8, 6.4, 6.4, 3.2, 3.2, 3.6, 1.2), sEll(10.2, 8.8, 3.0, 2.6))
	p.fill(mantle, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.6) + 0.04
		if fbm2(f.x*1.4, f.y*1.4, 1703, 2) > 0.6 {
			l -= 0.12 // its blotches
		}
		return pick(krakenRamp, l)
	})
	eyeAt(p, 11.2, 8.6, 1.1, hex(0xf0c040), false)
	p.on(iEll(11.3, 8.6, 0.75, 0.22, 0), solidC(colInk)) // a squid's slot of a pupil
	p.on(sEll(10.8, 8.1, 0.25, 0.2), solidC(icGlint))
	glintAt(p, 6.0, 5.2, 0.8, 0.22)
}

// --- the crawlers -----------------------------------------------------------------

// icHeadCrab is a crab, seen from the front: the red shell, its eyes up on
// their stalks, the big claw raised, the small one, its legs.
func icHeadCrab(p *icv) {
	// The legs, behind the shell.
	for i := 0; i < 3; i++ {
		fi := float64(i)
		for _, s := range []float64{-1, 1} {
			x0 := 8.0 + s*(3.0+fi*0.9)
			x1, y1 := 8.0+s*(5.4+fi*0.7), 13.4+fi*0.5
			p.fill(sLock(x0, 11.0, x1+s*0.3, 10.6, x1, y1, 0.55, 0.22), shBulge(crabRamp, 0.4, -0.12))
		}
	}
	// The eye stalks.
	for _, s := range []float64{-1, 1} {
		x := 8.0 + s*1.2
		p.fill(sCap(x, 7.6, x+s*0.5, 5.4, 0.32, 0.28), shBulge(crabRamp, 0.3, 0))
		p.fill(sEll(x+s*0.5, 5.0, 0.62, 0.62), shSphere(mkRamp(0x0c0a0c, 0x1c1418, 0x3a2a30), x+s*0.5, 5.0, 0.62, 0.62, 0, true))
	}
	// The shell.
	shell := iBlend(1.0, sEll(8.0, 9.6, 5.6, 3.0), sEll(8.0, 8.4, 4.4, 2.2))
	p.fill(shell, func(f *ifrag) color.RGBA {
		l := litOf((f.x-8.0)/5.6*0.8, (f.y-9.0)/3.0*0.8) + 0.04
		if fbm2(f.x*1.6, f.y*1.6, 1801, 2) > 0.62 {
			l += 0.1 // bumps
		}
		if f.y > 10.6 {
			l -= 0.1
		}
		return pick(crabRamp, l)
	})
	p.on(sPath(0.12, 0.08, 4.6, 10.6, 6.4, 11.4, 8.0, 11.6, 9.6, 11.4, 11.4, 10.6), solidC(crabRamp[1])) // the shell's lip
	// The small claw, left; the big one, raised, right.
	claw := func(ax, ay, bx, by, size float64, open float64) {
		p.fill(sCap(ax, ay, bx, by, size*0.45, size*0.5), shBulge(crabRamp, size*0.4, 0))
		palm := iEll(bx+0.2, by-size*0.7, size*0.8, size*1.0, -0.2)
		p.fill(palm, shSphere(crabRamp, bx, by-size*0.7, size*0.8, size, 0.04, false))
		fx, fy := bx+0.2, by-size*1.5
		p.fill(sLock(fx-size*0.4, fy+0.2, fx-size*0.6-open, fy-size*0.9, fx-size*0.1, fy-size*1.5, size*0.4, 0.06), shBulge(crabRamp, 0.4, 0.06))
		p.fill(sLock(fx+size*0.3, fy+0.2, fx+size*0.7+open, fy-size*0.7, fx+size*0.1, fy-size*1.4, size*0.36, 0.06), shBulge(crabRamp, 0.4, -0.04))
		p.on(sEll(fx-size*0.2, fy-size*1.25, 0.2, 0.2), solidC(crabRamp[5])) // a pale tip
	}
	claw(3.4, 10.4, 2.0, 8.2, 1.2, 0.1)
	claw(12.4, 10.4, 13.4, 7.6, 1.7, 0.3)
	glintAt(p, 6.0, 7.6, 0.8, 0.22)
}

// icHeadSpider is a cave spider's head, from the front: a hairy dome of
// purple-black, a red mark on it, a cluster of glossy eyes, the swollen
// jaws with their black fangs hooked under.
func icHeadSpider(p *icv) {
	// The palps, either side, behind the jaws.
	for _, s := range []float64{-1, 1} {
		p.fill(sLock(8.0+s*3.4, 10.8, 8.0+s*5.2, 11.6, 8.0+s*4.8, 14.2, 0.85, 0.55), shFur(spiderRamp, nil, nil, 0.6, 0.02, 0, 1, 1901))
	}
	dome := sEll(8.0, 7.6, 5.8, 4.8).rough(0.2, 0.4, 1903)
	p.fill(dome, func(f *ifrag) color.RGBA {
		l := litOf((f.x-8)/5.8*0.85, (f.y-7.6)/4.8*0.85) + 0.06
		if n := fbm2(f.x*3.2, f.y*1.4, 1905, 2); n > 0.64 {
			l += 0.12 // bristles
		} else if n < 0.32 {
			l -= 0.1
		}
		return pick(spiderRamp, l)
	})
	// The red mark: a chevron on the crown.
	p.on(sPath(0.42, 0.3, 5.6, 3.6, 8.0, 5.0, 10.4, 3.6), func(f *ifrag) color.RGBA { return pick(spiderMark, 0.6-0.2*(f.y-3.6)) })
	// The eyes: two great ones, a pair above, two little ones out at the sides.
	shiny := mkRamp(0x0a0608, 0x1e0c12, 0x481420, 0x8a2830)
	for _, e := range [][3]float64{{6.6, 7.2, 1.0}, {9.4, 7.2, 1.0}, {7.0, 5.4, 0.55}, {9.0, 5.4, 0.55}, {4.6, 6.4, 0.42}, {11.4, 6.4, 0.42}} {
		x, y, r := e[0], e[1], e[2]
		p.fill(sEll(x, y, r, r), shSphere(shiny, x, y, r, r, 0.06, true))
	}
	// The jaws, swollen and hairy, and their fangs.
	for _, s := range []float64{-1, 1} {
		x := 8.0 + s*1.3
		p.fill(iEll(x, 11.0, 1.35, 1.9, s*0.15), shFur(spiderRamp, nil, nil, 1.0, 0.08, 0, 1, 1907))
		fang := sTaper(0.55, 0.38, 0.05, x+s*0.2, 12.6, x+s*0.3, 13.8, x-s*0.4, 14.8, x-s*1.0, 15.0)
		p.fill(fang, func(f *ifrag) color.RGBA { return pick(icFangR, bulge(f, 0.35)+0.08) })
	}
}

// icHeadScorpion is a giant scorpion's front: the dark carapace with its
// eyes, a great pincer reaching forward, the tail arched over behind with
// its sting.
func icHeadScorpion(p *icv) {
	sh := func(round, bias float64) ishader {
		return func(f *ifrag) color.RGBA { return pick(scorpRamp[2:], bulge(f, round)+bias) }
	}
	// The tail, arched over from behind: segments, then the sting.
	segs := [][3]float64{{3.4, 9.4, 1.25}, {2.6, 7.2, 1.15}, {2.8, 5.0, 1.05}, {4.0, 3.2, 1.0}, {5.8, 2.2, 0.95}}
	for i, s := range segs {
		// Each segment a barrel along the arc, its end lapped by the next.
		ax, ay := s[0], s[1]
		var bx, by float64
		if i+1 < len(segs) {
			bx, by = segs[i+1][0], segs[i+1][1]
		} else {
			bx, by = 7.6, 2.4
		}
		ang := math.Atan2(by-ay, bx-ax)
		p.fill(iEll((ax*0.6+bx*0.4), (ay*0.6+by*0.4), s[2]*1.15, s[2]*0.85, ang), sh(s[2]*0.7, 0))
	}
	p.fill(iBlend(0.4, sEll(7.6, 2.4, 1.1, 0.9), sTaper(0.55, 0.3, 0.04, 8.2, 2.6, 9.2, 3.0, 9.6, 3.8, 9.4, 4.8)), sh(0.6, 0.04)) // the bulb and its sting
	p.on(sEll(9.3, 4.4, 0.18, 0.3), solidC(scorpRamp[1]))
	// The far pincer, then the body, the legs, the near pincer.
	pincer := func(dx, dy, bias float64) {
		p.fill(sCap(8.6+dx, 10.4+dy, 10.8+dx, 8.6+dy, 0.6, 0.65), sh(0.5, bias))
		p.fill(iEll(12.2+dx, 7.4+dy, 1.9, 1.3, -0.6), sh(1.0, bias+0.04))
		p.fill(sLock(12.8+dx, 6.2+dy, 13.8+dx, 4.6+dy, 15.0+dx, 4.4+dy, 0.6, 0.08), sh(0.4, bias+0.02))
		p.fill(sLock(13.6+dx, 7.0+dy, 14.8+dx, 6.6+dy, 15.2+dx, 5.6+dy, 0.5, 0.08), sh(0.4, bias-0.04))
	}
	pincer(-1.4, -0.8, -0.14)
	for i := 0; i < 3; i++ {
		x := 5.2 + 1.5*float64(i)
		p.fill(sLock(x, 11.4, x-0.4, 13.0, x-1.0+0.3*float64(i), 14.8, 0.42, 0.16), sh(0.3, -0.06))
	}
	body := iBlend(1.2, sEll(6.8, 10.4, 3.6, 2.2), sEll(9.2, 10.2, 1.8, 1.6))
	p.fill(body, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.8) + 0.02
		if math.Mod(f.x*0.9+20, 1) < 0.1 && f.x < 7.0 {
			l -= 0.14 // the joins of its plates
		}
		return pick(scorpRamp[2:], l)
	})
	p.fill(sEll(9.0, 9.2, 0.38, 0.34), shSphere(mkRamp(0x0a0608, 0x1a1012, 0x3a2a2a), 9.0, 9.2, 0.38, 0.34, 0, true)) // its eyes
	p.fill(sEll(8.2, 9.0, 0.3, 0.28), shSphere(mkRamp(0x0a0608, 0x1a1012, 0x3a2a2a), 8.2, 9.0, 0.3, 0.28, 0, true))
	pincer(0, 0, 0.02)
	glintAt(p, 11.4, 6.8, 0.6, 0.2)
}

// bezPts is n+1 points along the quadratic curve from p0 by c to p1, as
// x, y pairs: a brush's path.
func bezPts(n int, p0x, p0y, cx, cy, p1x, p1y float64) []float64 {
	pts := make([]float64, 0, 2*(n+1))
	for i := 0; i <= n; i++ {
		x, y := bezAt(float64(i)/float64(n), p0x, p0y, cx, cy, p1x, p1y)
		pts = append(pts, x, y)
	}
	return pts
}
