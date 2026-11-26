package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// Armour over the tall figure of tallfig.go: armorhd.go's pieces (a hide
// tunic with fur, lashed bone ribs, agate scales or a tyrant's teeth;
// shoulder guards; an agate headband or a skull cap) fitted to its body and
// its head. It sits on the trunk, so one drawing a facing does for every
// frame of the limbs.

// armorTall draws tier t's armour for a figure facing d: the plate, which
// goes under the arms and the head, and what goes over everything.
func armorTall(t sim.Tier, female bool, d sim.Dir) (body, helmet sprite) {
	if d == sim.DirLeft {
		b, h := armorTall(t, female, sim.DirRight)
		return mirrorSprite(b), mirrorSprite(h)
	}
	B, H := armorLayersTall(t, female, d)
	return limbSprite(B), limbSprite(H)
}

// armorLayersTall draws the two layers of armorTall; d is not DirLeft.
func armorLayersTall(t sim.Tier, female bool, d sim.Dir) (B, H *canvas) {
	g := newLimbGen(sim.Look{Female: female}, 0)
	f := &g.f
	P, T := g.S, g.T // the plate and what goes over everything
	side := d == sim.DirRight
	hide := hideHD
	if t == sim.TierDiamond {
		hide = bootHD
	}
	bottom := 79.0
	if female {
		bottom = 78
	}
	torso, _ := tallTorso(female, d, skirt{})
	plate := torso.and(sBelow(50)).and(sAbove(bottom))
	// front reports whether column x is on the figure's front.
	front := func(x int) bool { return d == sim.DirDown || side && x >= 27 }
	bias := 0.0
	if side {
		bias = -0.08
	}
	f.fill(P, plate, func(x, y int, _ float64) color.RGBA {
		i := int((plate.lum(x, y, 7) + bias) * 6)
		if hashf(x/2, y, 66) < 0.1 {
			i-- // the grain of the hide
		}
		if !side && x == 27 && y%3 != 0 {
			i -= 2 // laced down the middle
		}
		if y >= int(bottom)-1 {
			i--
		}
		return hide.at(max(i, 0))
	})
	x0, x1 := 16.0, 38.0
	switch {
	case side && female:
		x0, x1 = 21, 34
	case side:
		x0, x1 = 20, 34.4
	case female:
		x0, x1 = 19, 35
	}
	switch t {
	case sim.TierCopper: // fur at the neck and the hem
		for _, s := range []shape{plate.and(sAbove(53.4)), plate.and(sBelow(bottom - 3.6))} {
			s = s.jag(1.2, 3.4, s.y0+1, 61)
			f.fill(P, s, furShade(furTrimHD, s, 3, 0.04+bias, 63))
		}
	case sim.TierIron: // bones lashed across
		for _, y := range []float64{55, 60, 65, 70, 75} {
			l, r := x0+1.5, x1-1.5
			if side {
				l = 27
			}
			if d == sim.DirUp {
				break
			}
			rib := sPath(1.5, 1.5, l, y+0.6, (l+r)/2, y-0.4, r, y+0.6).and(plate)
			f.fill(P, rib, bevel(boneHD, rib, 1.6, 0.08+bias))
			for _, lx := range []float64{l + 2, (l + r) / 2, r - 2} {
				f.fill(P, sBox(lx-0.5, y-1.6, lx+0.5, y+2.4, 0).and(plate), plain(hide[0]))
			}
		}
	case sim.TierPlatinum: // agate scales
		sc := plate.and(sAbove(bottom - 1))
		f.fill(P, sc, func(x, y int, _ float64) color.RGBA {
			i := int((plate.lum(x, y, 7) + bias + 0.04) * 6)
			row := (y - 50) / 4
			u := (float64(x) + float64(row&1)*2.5) / 5
			e := float64((y-50)%4)/4 + math.Abs(u-math.Floor(u)-0.5)
			switch {
			case e < 0.3:
				i++
			case e > 0.95:
				return hide[0]
			case e > 0.75:
				i--
			}
			return agateHD.at(max(i, 1))
		})
	case sim.TierDiamond: // a row of teeth on a cord
		f.fill(P, plate.and(sAbove(52.5)), plain(hide[0]))
		for x := x0 + 1.5; x < x1-2; x += 4 {
			if !front(int(x) + 1) {
				continue
			}
			tooth := sPoly(x-0.2, 52, x+3.2, 52, x+1.5, 58.4).and(plate)
			f.fill(P, tooth, bevel(toothHD, tooth, 1.4, 0.12))
		}
	}

	// The shoulder guards sit on the round shoulders, over the arms.
	guard := func(cx, rx float64, lit float64) {
		s := sEll(cx, 50, rx, 3.7)
		switch t {
		case sim.TierCopper:
			s = s.jag(1.1, 3, 50, 67)
			f.fill(T, s, furShade(furTrimHD, s, 3, lit, 69))
		case sim.TierPlatinum:
			f.fill(T, s, bevel(agateHD, s, 3.4, lit+0.06))
			f.dot(T, int(cx-rx*0.4), 48, agateHD[5])
		default:
			f.fill(T, s, bevel(boneHD, s, 3.4, lit))
		}
		if t == sim.TierDiamond { // a bone spike
			sp := sLock(cx, 48.4, cx-0.6, 45.5, cx+0.6, 42.4, 1.9, 0.5)
			f.fill(T, sp, bevel(boneHD, sp, 1.6, lit+0.05))
		}
	}
	switch {
	case side && female:
		guard(27, 4.4, 0.04)
	case side:
		guard(27.4, 5.8, 0.04)
	case female:
		guard(17.5, 4, 0.06)
		guard(36.5, 4, -0.1)
	default:
		guard(14.5, 4.8, 0.06)
		guard(39.5, 4.8, -0.1)
	}

	// Headgear.
	m := headMap{27, 7.0 / 18, 8.5 / 16.5}
	if side {
		m = headMap{27.5, 7.4 / 18, 8.5 / 16.5}
	}
	skull := sSuper(m.cx, headMapY, 18*m.kx, 8.5, 2.5)
	switch t {
	case sim.TierPlatinum: // a hide band round the brow with an agate set in it
		band := skull.grow(0.7).and(sBox(0, 30, 60, 32, 0))
		f.fill(T, band, func(x, y int, _ float64) color.RGBA {
			i := 4 - (x-19)/5
			if y == 31 {
				i--
			}
			return hide.at(i)
		})
		gem := func(cx float64) {
			s := sEll(cx, 31, 1.7, 1.7)
			f.fill(T, s, bevel(agateHD, s, 1.6, 0.1))
			f.dot(T, int(cx-1), 30, agateHD[5])
		}
		if d == sim.DirDown {
			gem(27)
		}
		if side {
			gem(34.4)
		}
	case sim.TierDiamond: // a skull cap, its brow fringed with teeth
		// The brow comes down to the eyes in front and to the nape behind.
		var under shape
		switch d {
		case sim.DirDown:
			under = m.above(27)
		case sim.DirUp:
			under = m.above(42)
		default:
			under = m.poly(-9, -9, 60, -9, 60, 27, 26, 27, 23, 31, 21, 42, -9, 42)
		}
		capS := m.ell(27, 27.5, 19.6, 19).and(under)
		f.fill(T, capS, func(x, y int, _ float64) color.RGBA {
			i := int((capS.lum(x, y, 5) + 0.1) * 6)
			if !side && math.Abs(float64(x)+0.5-27-0.6*math.Sin(float64(y)*1.3)) < 0.6 {
				i -= 2 // the seam of the skull
			}
			if hashf(x/3, y/2, 91) < 0.06 {
				i--
			}
			return boneHD.at(max(i, 1))
		})
		if d == sim.DirDown { // its eye sockets
			for _, ex := range [2]float64{19.5, 34.5} {
				f.add(T, m.ell(ex, 19.5, 3.4, 3), plain(mix(boneHD[0], colInk, 0.6)))
			}
		}
		if d != sim.DirUp {
			y := m.y(27) - 0.4
			for x := m.x(10); x < m.x(45)-1; x += 2.4 {
				if side && x < 27 {
					continue
				}
				tooth := sPoly(x, y, x+2, y, x+1, y+2.6)
				f.fill(T, tooth, bevel(toothHD, tooth, 1, 0.14))
			}
		}
	}

	innerLines(0.4, P)
	innerLines(0.4, T)
	P.c.outline(0.75)
	T.c.outline(0.75)
	return P.c, T.c
}
