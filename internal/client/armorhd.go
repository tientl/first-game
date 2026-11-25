package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// Armour at HD density, over lookhd.go's figure: the same pieces as
// armorart.go draws (a hide tunic with fur, lashed bone ribs, agate scales
// or a tyrant's teeth; shoulder guards; an agate headband or a skull cap),
// in the same places, HDDensity times finer.

var (
	hideHD    = mkRamp(0x3c2414, 0x5c3a22, 0x7f5432, 0xa27348, 0xc39563, 0xdcb682)
	furTrimHD = ext6(furTrim)
	boneHD    = ext6(hornRamp)
	agateHD   = mkRamp(0x3a100e, 0x561c16, 0x8a3222, 0xbb5733, 0xdf8a56, 0xf3c39a)
	toothHD   = ext6(toothArt)
)

// armorLayersHD is armorLayersPose at HD density; d is not DirLeft.
func armorLayersHD(t sim.Tier, female bool, d sim.Dir, frame int, whole bool) (B, H *canvas) {
	g := newFigGen(sim.Look{Female: female}, figW, figH)
	f := &g.f
	P, T := g.S, g.T // the plate and what goes over everything
	sw := walkSwing(frame)
	side := d == sim.DirRight
	hide := hideHD
	if t == sim.TierDiamond {
		hide = bootHD
	}
	bottom := 66.0
	if female {
		bottom = 69
	}
	plate := hdTorso(female, d, 0, 0).and(sBelow(51)).and(sAbove(bottom))
	if side && !whole {
		sl, hd := hdNearArm(sw)
		plate = plate.sub(sl.or(hd).grow(0.6)) // the arm and the hand
	}
	// front reports whether column x is on the figure's front.
	front := func(x int) bool { return d == sim.DirDown || side && x >= 27 }
	bias := 0.0
	if side {
		bias = -0.08
	}
	f.fill(P, plate, func(x, y int, _ float64) color.RGBA {
		i := int((plate.lum(x, y, 8) + bias) * 6)
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
	x0, x1 := 15.0, 39.0
	switch {
	case side && female:
		x0, x1 = 21, 36
	case side:
		x0, x1 = 18, 36
	case female:
		x0, x1 = 18, 36
	}
	switch t {
	case sim.TierCopper: // fur at the neck and the hem
		for _, s := range []shape{plate.and(sAbove(54.4)), plate.and(sBelow(bottom - 3.6))} {
			s = s.jag(1.2, 3.4, s.y0+1, 61)
			f.fill(P, s, furShade(furTrimHD, s, 3, 0.04+bias, 63))
		}
	case sim.TierIron: // bones lashed across
		for _, y := range []float64{54.5, 58.5, 62.5} {
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
			i := int((plate.lum(x, y, 8) + bias + 0.04) * 6)
			row := (y - 51) / 4
			u := (float64(x) + float64(row&1)*2.5) / 5
			e := float64((y-51)%4)/4 + math.Abs(u-math.Floor(u)-0.5)
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
		f.fill(P, plate.and(sAbove(53)), plain(hide[0]))
		for x := x0 + 2; x < x1-2; x += 4 {
			if !front(int(x) + 1) {
				continue
			}
			tooth := sPoly(x-0.2, 52.5, x+3.2, 52.5, x+1.5, 58.4).and(plate)
			f.fill(P, tooth, bevel(toothHD, tooth, 1.4, 0.12))
		}
	}

	// The shoulder guards sit on the round shoulders, over the bare arms.
	guard := func(cx, rx float64, lit float64) {
		s := sEll(cx, 51.2, rx, 3.7)
		switch t {
		case sim.TierCopper:
			s = s.jag(1.1, 3, 51, 67)
			f.fill(T, s, furShade(furTrimHD, s, 3, lit, 69))
		case sim.TierPlatinum:
			f.fill(T, s, bevel(agateHD, s, 3.4, lit+0.06))
			f.dot(T, int(cx-rx*0.4), 49, agateHD[5])
		default:
			f.fill(T, s, bevel(boneHD, s, 3.4, lit))
		}
		if t == sim.TierDiamond { // a bone spike
			sp := sLock(cx, 49.5, cx-0.6, 46.5, cx+0.6, 43.4, 1.9, 0.5)
			f.fill(T, sp, bevel(boneHD, sp, 1.6, lit+0.05))
		}
	}
	switch {
	case side && female:
		guard(25.5, 5, 0.04)
	case side:
		guard(27, 6.4, 0.04)
	case female:
		guard(16.5, 5, 0.06)
		guard(37.5, 5, -0.1)
	default:
		guard(13.5, 5.2, 0.06)
		guard(40.5, 5.2, -0.1)
	}

	// Headgear.
	skull := sSuper(27, 31, 18, 16.5, 2.6)
	switch t {
	case sim.TierPlatinum: // a hide band round the brow with an agate set in it
		band := skull.grow(0.7).and(sBox(0, 23.4, 54, 26.8, 0))
		f.fill(T, band, func(x, y int, _ float64) color.RGBA {
			i := 4 - (x-9)/10
			switch y {
			case 23:
				i++
			case 26:
				i--
			}
			return hide.at(i)
		})
		gem := func(cx float64) {
			s := sEll(cx, 25.1, 2.6, 2.4)
			f.fill(T, s, bevel(agateHD, s, 2.4, 0.1))
			f.dot(T, int(cx-1), 24, agateHD[5])
		}
		if d == sim.DirDown {
			gem(27)
		}
		if side {
			gem(43.6)
		}
	case sim.TierDiamond: // a skull cap, its brow fringed with teeth
		// The brow comes down to the eyes in front and to the nape behind.
		var under shape
		switch d {
		case sim.DirDown:
			under = sAbove(27)
		case sim.DirUp:
			under = sAbove(42)
		default:
			under = sPoly(-9, -9, 60, -9, 60, 27, 26, 27, 23, 31, 21, 42, -9, 42)
		}
		capS := sEll(27, 27.5, 19.6, 19).and(under)
		f.fill(T, capS, func(x, y int, _ float64) color.RGBA {
			i := int((capS.lum(x, y, 11) + 0.1) * 6)
			if !side && math.Abs(float64(x)+0.5-27-math.Sin(float64(y)*1.3)) < 0.6 {
				i -= 2 // the seam of the skull
			}
			if hashf(x/3, y/2, 91) < 0.06 {
				i--
			}
			return boneHD.at(max(i, 1))
		})
		if d == sim.DirDown { // its eye sockets
			for _, ex := range [2]float64{19.5, 34.5} {
				f.add(T, sEll(ex, 20.5, 3, 2.5), plain(mix(boneHD[0], colInk, 0.6)))
			}
			f.add(T, sPoly(27, 22.5, 28.4, 25.4, 25.6, 25.4), plain(mix(boneHD[0], colInk, 0.5)))
		}
		if d != sim.DirUp {
			for x := 10.0; x < 45; x += 3.5 {
				if side && x < 26 {
					continue
				}
				tooth := sPoly(x, 26.6, x+2.6, 26.6, x+1.3, 30.4)
				f.fill(T, tooth, bevel(toothHD, tooth, 1.2, 0.14))
			}
		}
	}

	innerLines(0.4, P)
	innerLines(0.4, T)
	P.c.outline(0.75)
	T.c.outline(0.75)
	return P.c, T.c
}
