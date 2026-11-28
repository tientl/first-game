package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of what is worn and carried: the armours of each tier in
// their world tones (armorhd.go: hide, bone, agate, the tyrant's teeth),
// the dinosaur-hide armour and the fur coat, the clothes (tallfig.go,
// lookhd.go), the saddle, the healing draughts and the bandage, the raft
// and the sailing boat. A garment is seen from the front, lit from the top
// left as if round the body in it.

func init() {
	regIcon(sim.ItemCopperArmor, icHideArmor)
	regIcon(sim.ItemIronArmor, icBoneArmor)
	regIcon(sim.ItemPlatinumArmor, icAgateArmor)
	regIcon(sim.ItemDiamondArmor, icToothArmor)
	regIcon(sim.ItemDinoArmor, icDinoArmor)
	regIcon(sim.ItemFurCoat, icFurCoat)
	regIcon(sim.ItemTunic, icTunic)
	regIcon(sim.ItemMantle, icMantle)
	regIcon(sim.ItemLegWraps, icLegWraps)
	regIcon(sim.ItemBoots, icBoots)
	regIcon(sim.ItemSaddle, icSaddle)
	regIcon(sim.ItemPotion, icPotion)
	regIcon(sim.ItemBigPotion, icBigPotion)
	regIcon(sim.ItemBandage, icBandage)
	regIcon(sim.ItemRaft, icRaft)
	regIcon(sim.ItemSailboat, icSailboat)
}

var (
	icLinenR = mkRamp(0x6e6a5c, 0x969080, 0xbcb6a4, 0xdcd8c8, 0xf4f2e8) // flax linen, undyed
	icSailR  = mkRamp(0x8a7e66, 0xb4a88c, 0xd6ccb0, 0xeee6d0, 0xfffaee)
)

// torsoLit is the light on a garment at (x, y): round like the body under
// it, cx its middle and hw its half width.
func torsoLit(x, cx, hw float64) float64 {
	t := max(-1, min(1, (x-cx)/hw))
	return litOf(t*0.8, -0.05)
}

// garment paints a tunic seen from the front: its sleeves, then its body
// from the shoulders to hem, shaded by sh. It returns the body's shape, to
// lay more on.
func garment(p *icv, hem float64, long bool, sh func(f *ifrag) color.RGBA) shape {
	slv := 7.4
	if long {
		slv = 11.6
	}
	left := sPoly(4.8, 2.6, 0.8, 5.6, 1.4+0.4*b2f(long), slv, 3.6, slv+0.4, 4.6, 6.6)
	right := sPoly(11.2, 2.6, 15.2, 5.6, 14.6-0.4*b2f(long), slv, 12.4, slv+0.4, 11.4, 6.6)
	p.fill(iBlend(0.6, left), sh)
	p.fill(iBlend(0.6, right), sh)
	body := iBlend(0.8, sPoly(4.6, 2.2, 6.4, 1.6, 9.6, 1.6, 11.4, 2.2, 11.6, 6.6, 12.6, hem, 3.4, hem, 4.4, 6.6)).
		sub(sEll(8.0, 1.6, 1.9, 1.5)) // the neck
	p.fill(body, sh)
	// The inside of its back, seen through the neck.
	p.under(sEll(8.0, 2.0, 1.85, 1.3), func(f *ifrag) color.RGBA {
		c := sh(f)
		return mix(c, colInk, 0.55)
	})
	return body
}

// hideShader shades hide of ramp r round the body, its grain.
func hideShader(r ramp, bias float64, seed uint32) func(f *ifrag) color.RGBA {
	return func(f *ifrag) color.RGBA {
		l := torsoLit(f.x, 8, 7.4) + bias + (fbm2(f.x*1.2, f.y*1.2, seed, 2)-0.5)*0.16
		if -f.d < 0.45 && toward(f.gx, f.gy) > 0.2 {
			l += 0.12
		}
		return pick(r, l)
	}
}

// furHem lays a band of fur along y from x0 to x1, shaggy at its lower edge.
func furHem(p *icv, r ramp, x0, x1, y, h float64, seed uint32) {
	band := sBox(x0, y-h/2, x1, y+h/2, 0.4).jag(0.6, 0.7, y, seed).rough(0.15, 0.4, seed+1)
	p.fill(band, func(f *ifrag) color.RGBA {
		l := torsoLit(f.x, 8, 7.4) + 0.05
		if math.Abs(math.Sin((f.x*2.6+fbm2(f.x, f.y*2, seed, 2)*2)*math.Pi)) > 0.86 {
			l -= 0.18
		}
		return pick(r, l)
	})
}

// icHideArmor is a tunic of thick hide, laced up the front, fur at its
// neck and hem, toggles of bone.
func icHideArmor(p *icv) {
	body := garment(p, 13.6, false, hideShader(hideHD, 0, 511))
	_ = body
	for i := 0; i < 4; i++ { // the lacing
		y := 4.2 + float64(i)*1.9
		p.fill(sCap(7.2, y, 8.8, y+1.2, 0.17, 0.17), solidC(hideHD[0]))
		p.fill(sCap(8.8, y, 7.2, y+1.2, 0.17, 0.17), solidC(hideHD[4]))
	}
	for _, y := range []float64{5.0, 8.4} {
		p.fill(sCap(9.6, y, 10.8, y, 0.32, 0.3), shBulge(boneHD, 0.3, 0.1))
	}
	furHem(p, furTrimHD, 3.4, 12.6, 13.6, 1.8, 512)
	p.fill(sRing(8.0, 1.8, 2.1, 1.6, 0.55).and(sBelow(1.6)), shBulge(furTrimHD, 0.5, 0.05))
}

// icBoneArmor is the hide tunic with rows of bones lashed across its chest
// and a guard of bone on each shoulder.
func icBoneArmor(p *icv) {
	garment(p, 13.6, false, hideShader(hideHD, -0.04, 521))
	for i := 0; i < 4; i++ {
		y := 4.4 + float64(i)*2.2
		w := 3.2 - float64(i)*0.15
		a := mkAxis(8-w, y, 8+w, y)
		p.fill(sCap(8-w, y, 8+w, y, 0.55, 0.55), shRod(boneHD, a, 0.55, 0.06, 0, 0))
		p.fill(sEll(8-w, y, 0.75, 0.7), shBulge(boneHD, 0.5, 0.1)) // the knobbed ends
		p.fill(sEll(8+w, y, 0.75, 0.7), shBulge(boneHD, 0.5, 0.0))
		p.fill(sBox(7.4, y-0.7, 8.6, y+0.7, 0.3), shCord(icLashR, mkAxis(7.4, y, 8.6, y-0.4), 0.6, 2.6))
	}
	for _, s := range []float64{-1, 1} { // shoulder guards
		x := 8 + s*5.4
		p.fill(iEll(x, 4.0, 2.2, 1.5, s*0.6), shBulge(boneHD, 1.0, 0.04-0.06*s))
	}
}

// icAgateArmor is the hide tunic sewn over with polished agate scales in
// rows, each overlapping the one below.
func icAgateArmor(p *icv) {
	body := garment(p, 13.6, false, hideShader(hideHD, -0.04, 531))
	m := heldMats[sim.TierPlatinum]
	p.on(body.grow(-0.7), func(f *ifrag) color.RGBA {
		const w, h = 1.6, 1.25
		row := math.Floor(f.y / h)
		off := 0.5 * math.Mod(row, 2)
		cx := (math.Floor(f.x/w+off) + 0.5 - off) * w
		cy := (row + 0.15) * h
		dx, dy := (f.x-cx)/(w*0.55), (f.y-cy)/(h*1.0)
		d := math.Hypot(dx, dy)
		if d > 1 {
			return color.RGBA{} // between the scales: the hide shows
		}
		l := litOf(dx*0.7, dy*0.7) + 0.02
		if d > 0.82 && dy > 0 {
			l -= 0.25 // its lower rim, in its own shadow
		}
		if b := math.Mod(d*2.4, 1); b < 0.25 {
			l += 0.12 // the agate's bands
		}
		if gx, gy := dx+0.4, dy+0.45; gx*gx+gy*gy < 0.04 {
			return m.glint
		}
		return pick(m.r, l)
	})
}

// icToothArmor is armour of dark hide hung with a tyrant's teeth: a collar
// of them round the neck, two rows across the chest, red ochre between.
func icToothArmor(p *icv) {
	body := garment(p, 13.6, false, hideShader(bootHD, 0.02, 541))
	p.on(body, func(f *ifrag) color.RGBA {
		if math.Abs(f.y-7.0) < 0.25 || math.Abs(f.y-10.6) < 0.25 {
			return bloodRamp[3] // stripes of ochre
		}
		return color.RGBA{}
	})
	tooth := func(x, y, l, ang float64) {
		fr := frameOf(x, y, math.Sin(ang), math.Cos(ang))
		s := fr.poly(0, -0.45, l*0.6, -0.35, l, 0.05, l*0.6, 0.4, 0, 0.45)
		p.fill(s, func(f *ifrag) color.RGBA {
			_, t := fr.st(f.x, f.y)
			return pick(toothHD, 0.62-t*0.5+0.1*toward(f.gx, f.gy))
		})
	}
	for i := 0; i < 7; i++ { // the collar, fanned round the neck
		a := -0.9 + float64(i)*0.3
		tooth(8+math.Sin(a)*2.4, 2.4+math.Cos(a)*1.6, 2.0, a*0.6)
	}
	for _, y := range []float64{7.4, 11.0} {
		for i := 0; i < 5; i++ {
			tooth(5.0+float64(i)*1.5, y, 1.7, 0)
		}
	}
}

// icDinoArmor is armour of a dinosaur's hide: pebbly green scales, dark
// bands across them, the edges bound with plain hide.
func icDinoArmor(p *icv) {
	sc := shPebbly(icScaleR, 1.1, 0.02, 681)
	body := garment(p, 13.6, false, func(f *ifrag) color.RGBA {
		c := sc(f)
		if math.Mod(f.y*0.3+fbm2(f.x*0.3, f.y*0.3, 682, 2)*0.9, 1) < 0.3 {
			c = mix(c, icScaleR[0], 0.5)
		}
		return mix(c, pick(icScaleR, torsoLit(f.x, 8, 7.4)), 0.25)
	})
	p.on(body.sub(body.grow(-0.55)), func(f *ifrag) color.RGBA { return pick(hideHD, 0.5+0.3*toward(f.gx, f.gy)) })
	p.fill(sBox(3.6, 9.4, 12.4, 10.6, 0.4).and(body), func(f *ifrag) color.RGBA { // a belt of hide
		return pick(hideHD, torsoLit(f.x, 8, 7.4)-0.05)
	})
}

// icFurCoat is a long coat of fur: shaggy, long sleeves, a deep collar,
// closed with bone toggles.
func icFurCoat(p *icv) {
	shFur := func(f *ifrag) color.RGBA {
		w := f.x/0.9 + fbm2(f.x*0.4, f.y*0.4, 611, 2)*1.4
		c := w - math.Floor(w)
		l := torsoLit(f.x, 8, 7.4)*0.7 + 0.18 + 0.14*math.Cos((c-0.5)*2*math.Pi)
		if math.Abs(math.Sin((f.x*5+fbm2(f.x, f.y*0.5, 612, 2)*4)*math.Pi)) > 0.9 {
			l -= 0.1
		}
		return pick(icFurR, l)
	}
	body := garment(p, 14.4, true, shFur)
	_ = body
	p.on(sCap(8.0, 3.2, 8.2, 14.2, 0.16, 0.16), solidC(icFurR[0])) // where it closes
	for _, y := range []float64{5.4, 8.4, 11.4} {
		p.fill(sCap(7.4, y, 8.8, y, 0.3, 0.28), shBulge(boneHD, 0.3, 0.1))
	}
	// The collar, a ruff of longer fur.
	ruff := sRing(8.0, 2.4, 2.3, 1.4, 0.6).jag(0.4, 0.6, 2.4, 613)
	p.fill(ruff, shFur)
	// Shaggy hem and cuffs.
	furHem(p, icFurR, 3.3, 12.7, 14.1, 1.0, 614)
}

// icTunic is a belted tunic of flax linen: short sleeves, a round neck,
// the hem fringed, a leather belt with a gold buckle.
func icTunic(p *icv) {
	body := garment(p, 13.8, false, func(f *ifrag) color.RGBA {
		l := torsoLit(f.x, 8, 7.4) + 0.04
		if math.Mod(f.x*3.0+20, 1) < 0.12 || math.Mod(f.y*3.0+20, 1) < 0.12 {
			l -= 0.06 // the weave
		}
		if math.Abs(math.Sin((f.x-8)*1.1+f.y*0.15)) < 0.07 && f.y > 9.6 {
			l -= 0.14 // folds below the belt
		}
		return pick(icLinenR, l)
	})
	p.on(body.and(sBelow(13.2)), func(f *ifrag) color.RGBA { // the fringe
		if math.Mod(f.x*2.2, 1) < 0.35 {
			return icLinenR[1]
		}
		return color.RGBA{}
	})
	belt := sBox(4.0, 8.6, 12.0, 9.8, 0.3)
	p.fill(belt.and(body.grow(0.2)), func(f *ifrag) color.RGBA { return pick(leatherHD, torsoLit(f.x, 8, 7.4)) })
	p.fill(sBox(7.2, 8.4, 8.8, 10.0, 0.3), func(f *ifrag) color.RGBA { return pick(goldHD, 0.6+0.3*toward(f.gx, f.gy)) })
	p.on(sBox(7.7, 8.9, 8.3, 9.5, 0.1), solidC(leatherHD[0]))
}

// icMantle is a fur mantle for the shoulders: a cape of grey fur, deep at
// the back, pinned at the throat with a brooch of gold.
func icMantle(p *icv) {
	cape := iBlend(1.2, sEll(8.0, 6.6, 7.2, 5.6).sub(sEll(8.0, 2.4, 2.4, 2.0)), sPoly(2.0, 7.0, 14.0, 7.0, 12.2, 13.4, 3.8, 13.4)).
		jag(0.8, 0.9, 11.6, 621).rough(0.2, 0.5, 622)
	p.fill(cape, func(f *ifrag) color.RGBA {
		// The fur falls from the shoulders down.
		fx, fy := (f.x-8)*0.4, 1.0
		m := math.Hypot(fx, fy)
		u, v := (f.x*fx+f.y*fy)/m, (-f.x*fy+f.y*fx)/m
		w := v/0.9 + fbm2(f.x*0.4, f.y*0.4, 623, 2)*1.4
		c := w - math.Floor(w)
		l := bulge(f, 2.6)*0.6 + 0.22 + 0.16*math.Cos((c-0.5)*2*math.Pi)
		if math.Abs(math.Sin((v*6+fbm2(u*0.5, v*2, 624, 2)*4)*math.Pi)) > 0.9 {
			l -= 0.1
		}
		return pick(furHD, l)
	})
	// The opening down the front, and the brooch at the throat.
	p.on(sPath(0.2, 0.3, 8.0, 4.2, 7.8, 9.0, 8.4, 13.4), solidC(furHD[0]))
	bx, by := 8.0, 4.6
	p.fill(sEll(bx, by, 1.25, 1.25), shSphere(goldHD, bx, by, 1.25, 1.25, 0.02, true))
	p.on(sEll(bx, by, 0.5, 0.5), solidC(bloodRamp[3]))
}

// icLegWraps is a pair of hide trousers bound below the knee with strips
// of pale cloth wound crosswise.
func icLegWraps(p *icv) {
	tr := ext6(trouserRamp)
	wr := ext6(wrapRamp)
	leg := func(x0, x1, foot, bias float64) shape {
		s := iBlend(0.5, sPoly(x0, 2.6, x1, 2.6, x1-0.3, foot, x0+0.5, foot))
		p.fill(s, func(f *ifrag) color.RGBA {
			cx := (x0 + x1) / 2
			l := litOf(max(-1, min(1, (f.x-cx)/((x1-x0)/2)))*0.85, 0) + bias
			if math.Abs(math.Sin((f.x-cx)*1.2+f.y*0.3)) < 0.06 {
				l -= 0.14
			}
			return pick(tr, l)
		})
		return s
	}
	waist := sBox(2.6, 1.8, 13.4, 3.6, 0.5)
	l := leg(2.8, 7.8, 15.0, 0.06)
	r := leg(8.2, 13.2, 15.0, -0.06)
	p.fill(waist, func(f *ifrag) color.RGBA { return pick(tr, torsoLit(f.x, 8, 5.4)+0.04) })
	p.fill(sCap(4.0, 2.8, 12.0, 2.8, 0.22, 0.22), solidC(leatherHD[1])) // the drawstring
	for _, s := range []shape{l, r} {                                   // the wraps, wound crosswise from the knee down
		p.on(s.and(sBelow(9.0)), func(f *ifrag) color.RGBA {
			a := math.Mod(f.y*0.9+f.x*0.55+20, 1.2)
			b := math.Mod(f.y*0.9-f.x*0.55+20, 1.2)
			if a < 0.42 || b < 0.42 {
				lv := 0.6 + 0.2*toward(f.gx, f.gy)
				if a < 0.08 || b < 0.08 {
					lv -= 0.3
				}
				return pick(wr, lv)
			}
			return color.RGBA{}
		})
	}
}

// icBoots is a pair of hide boots, fur turned out at their cuffs, laced
// up the front.
func icBoots(p *icv) {
	boot := func(x, y float64, bias float64, seed uint32) {
		shaft := sBox(x-2.2, y-6.0, x+1.8, y, 0.8)
		foot := sEll(x+1.2, y-0.2, 3.6, 1.9)
		b := iBlend(1.0, shaft, foot)
		p.fill(b, func(f *ifrag) color.RGBA {
			l := bulge(f, 1.6) + bias + (fbm2(f.x*1.2, f.y*1.2, seed, 2)-0.5)*0.14
			if f.y > y+1.0 {
				l -= 0.2 // the sole
			}
			return pick(bootHD, l+0.1)
		})
		for i := 0; i < 3; i++ { // the lacing
			ly := y - 4.6 + float64(i)*1.3
			p.on(sCap(x+0.2, ly, x+1.6, ly+0.7, 0.13, 0.13), solidC(wrapRamp[2]))
		}
		cuff := sBox(x-2.7, y-7.8, x+2.3, y-5.6, 0.7).jag(0.5, 0.6, y-6.4, seed+1).rough(0.15, 0.35, seed+2)
		p.fill(cuff, func(f *ifrag) color.RGBA {
			l := bulge(f, 1.0)*0.7 + 0.25 + bias
			if math.Abs(math.Sin((f.x*2.8+fbm2(f.x, f.y, seed+3, 2)*2)*math.Pi)) > 0.86 {
				l -= 0.16
			}
			return pick(furHD, l)
		})
	}
	boot(10.0, 12.4, -0.06, 1251)
	boot(5.2, 13.6, 0.04, 1252)
}

// icSaddle is a riding saddle seen from the side: a deep seat of leather
// on a wooden tree, high at the cantle behind and the pommel before, a
// skirt of hide under it, a stirrup of bone hanging on its strap.
func icSaddle(p *icv) {
	// The blanket under it, its edge showing.
	p.fill(sBox(2.0, 7.2, 14.0, 10.0, 0.8), func(f *ifrag) color.RGBA {
		l := 0.5 + 0.2*toward(f.gx, f.gy)*clamp01(1+f.d/0.6)
		if math.Mod(f.x*0.8+20, 1) < 0.2 {
			l -= 0.16 // woven stripes
		}
		return pick(bloodRamp, l+0.1)
	})
	// The skirt.
	skirt := iBlend(0.8, sPoly(3.2, 7.4, 12.8, 7.4, 12.0, 12.4, 4.0, 12.4))
	p.fill(skirt, hideShader(leatherHD, 0.02, 581))
	p.on(skirt.grow(-0.55).sub(skirt.grow(-0.8)), func(f *ifrag) color.RGBA { // its stitched edge
		if math.Mod((f.x+f.y)*1.6, 1) < 0.5 {
			return leatherHD[5]
		}
		return color.RGBA{}
	})
	// The seat: low in the middle, the cantle behind rising, the pommel
	// before with its horn.
	var top, bot []float64
	for i := 0; i <= 14; i++ {
		x := lerpf(1.6, 13.4, float64(i)/14)
		u := (x - 7.6) / 5.8
		y := 7.0 - 2.6*u*u*u*u - 0.9*u*u
		if x < 7.6 {
			y = 7.0 - 2.1*u*u*u*u - 0.6*u*u
		}
		top = append(top, x, y)
		bot = append(bot, x, 8.6+0.4*u*u)
	}
	var poly []float64
	poly = append(poly, top...)
	for i := len(bot) - 2; i >= 0; i -= 2 {
		poly = append(poly, bot[i], bot[i+1])
	}
	seat := iBlend(1.0, sPoly(poly...), sEll(2.4, 5.6, 1.1, 1.5))
	p.fill(seat, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.2) + 0.06
		if f.y > 8.0 {
			l -= 0.14
		}
		return pick(leatherHD, l)
	})
	p.fill(sCap(12.8, 4.8, 13.6, 3.0, 0.7, 0.55), shRod(iiWood, mkAxis(12.8, 4.8, 13.6, 3.0), 0.7, 0.05, 1, 582)) // the horn
	p.fill(sEll(13.7, 2.8, 0.9, 0.55), shBulge(iiWood, 0.5, 0.1))
	// The stirrup on its strap.
	p.fill(sCap(7.4, 8.4, 7.6, 12.6, 0.36, 0.34), shRod(icLashR, mkAxis(7.4, 8.4, 7.6, 12.6), 0.36, 0.04, 0, 0))
	p.fill(sRing(7.6, 13.6, 1.4, 1.1, 0.32).and(sBelow(12.8)), shBulge(boneHD, 0.3, 0.1))
	p.fill(sCap(6.2, 14.6, 9.0, 14.6, 0.34, 0.34), shRod(boneHD, mkAxis(6.2, 14.6, 9, 14.6), 0.34, 0.06, 0, 0))
}

// bottle paints a gourd bottle of ramp r at (cx, cy) s big, a stopper of
// wood, a cord round its neck; it returns its body.
func bottle(p *icv, r ramp, cx, cy, s float64) shape {
	b := iBlend(1.2*s, sEll(cx, cy+1.6*s, 4.0*s, 4.2*s), sEll(cx, cy-3.0*s, 1.9*s, 2.0*s), sCap(cx, cy-4.4*s, cx, cy-5.8*s, 0.9*s, 0.8*s))
	p.fill(b, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.8*s)*0.92 + 0.04
		if hashf(int(f.x*3), int(f.y*2.4), 691) < 0.05 {
			l += 0.12
		}
		return pick(r, l)
	})
	p.fill(sCap(cx, cy-6.0*s, cx+0.1, cy-7.6*s, 0.85*s, 0.7*s), shRod(iiWood, mkAxis(cx, cy-6*s, cx, cy-7.6*s), 0.8*s, 0.05, 1, 692))
	p.fill(sBox(cx-1.3*s, cy-1.6*s, cx+1.3*s, cy-0.8*s, 0.3).and(b.grow(0.15)), shCord(icTwineR, mkAxis(cx-1.3*s, cy-1.2*s, cx+1.3*s, cy-1.5*s), 0.6, 2.4))
	glintAt(p, cx-2.2*s, cy+0.6*s, 0.6*s, 0.26*s)
	return b
}

// icPotion is the healing draught: a little gourd bottle, a drop of the
// red draught on its lip, a red heart painted on it.
func icPotion(p *icv) {
	cx, cy := 7.6, 8.8
	bottle(p, iiGourd, cx, cy, 1.0)
	heart := iBlend(0.4, sEll(cx-0.75, cy+1.2, 1.0, 0.95), sEll(cx+0.75, cy+1.2, 1.0, 0.95), sPoly(cx-1.7, cy+1.5, cx+1.7, cy+1.5, cx, cy+3.6))
	p.on(heart, func(f *ifrag) color.RGBA { return pick(iiRed, 0.55+0.25*toward(f.gx, f.gy)) })
	drop := iBlend(0.6, sEll(12.2, 12.4, 1.5, 1.6), sCap(12.2, 11.8, 12.0, 9.8, 1.0, 0.1))
	p.fill(drop, shSphere(iiRed, 12.2, 12.4, 1.5, 1.6, 0.04, true))
}

// icBigPotion is the stronger draught, which also draws out poison: a big
// pale bottle of bone-white gourd, bound in hide, a bone bead on its cord,
// its magenta draught brimming at the mouth.
func icBigPotion(p *icv) {
	cx, cy := 7.8, 9.0
	b := bottle(p, icBoneR, cx, cy, 1.0)
	p.on(b.and(sBelow(cy+1.4)).and(sAbove(cy+3.4)), func(f *ifrag) color.RGBA { // a band of hide
		return pick(hideHD, bulge(f, 2.4)*0.8+0.1)
	})
	p.fill(sEll(cx+2.6, cy-0.2, 0.7, 0.7), shSphere(boneHD, cx+2.6, cy-0.2, 0.7, 0.7, 0.1, false))
	mag := mkRamp(0x5a0a30, 0x9a1a5a, 0xd8408a, 0xffa0d0, 0xffe0f0)
	drop := iBlend(0.6, sEll(12.8, 12.8, 1.5, 1.6), sCap(12.8, 12.2, 12.6, 10.2, 1.0, 0.1))
	p.fill(drop, shSphere(mag, 12.8, 12.8, 1.5, 1.6, 0.04, true))
	p.fill(sCross(cx, cy+5.4, 1.3, 0.45), func(f *ifrag) color.RGBA { return pick(mag, 0.6+0.2*toward(f.gx, f.gy)) })
}

// sCross is a plus sign at (x, y), r to each arm's end, w half thick.
func sCross(x, y, r, w float64) shape {
	return sBox(x-r, y-w, x+r, y+w, w*0.5).or(sBox(x-w, y-r, x+w, y+r, w*0.5))
}

// icBandage is a dressing for a wound: a pad of grass and leaf bound with
// flat strips of spider silk wound crosswise round it, a loose end of the
// strip trailing off.
func icBandage(p *icv) {
	cx, cy := 7.4, 8.4
	pad := tilted(iBlend(1.2, sBox(cx-5.2, cy-3.4, cx+5.2, cy+3.4, 1.8)), cx, cy, -0.35)
	p.fill(pad, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.0)*0.85 + 0.06
		if math.Abs(math.Sin((f.x*1.6+f.y*0.6+fbm2(f.x, f.y, 711, 2)*2)*math.Pi)) > 0.88 {
			l -= 0.16 // blades of grass in it
		}
		return pick(herbLeafRamp, l)
	})
	// The strips of silk, flat, wound round it: lit as the pad under them.
	band := func(x0, y0, x1, y1, w float64) {
		a := mkAxis(x0, y0, x1, y1)
		s := sCap(x0, y0, x1, y1, w, w).and(pad.grow(0.3))
		p.fill(s, func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			nx, ny := (f.x-cx)/5.4, (f.y-cy)/3.8
			l := litOf(nx*0.75, ny*0.75) + 0.12
			if math.Mod(u*2.6+20, 1) < 0.1 {
				l -= 0.08 // the weave
			}
			if math.Abs(v) > w*0.78 {
				l -= 0.14 // its frayed edges
			}
			return pick(icSilkSkeinR, l)
		})
	}
	band(3.6, 12.6, 11.4, 3.4, 1.2)
	band(1.0, 7.4, 14.4, 9.4, 1.1)
	// The loose end, a flat ribbon, its tip frayed.
	end := sTaper(1.05, 1.0, 0.9, 13.4, 9.6, 14.2, 11.8, 12.8, 13.6, 10.2, 14.4)
	p.fill(end.jag(0.25, 0.3, 14.0, 714), func(f *ifrag) color.RGBA {
		return pick(icSilkSkeinR, 0.62+0.22*toward(f.gx, f.gy)*clamp01(1+f.d/0.6)-0.1*clamp01((f.y-11)/3))
	})
}

// icRaft is a raft of logs lashed side by side, seen from above and to the
// side, a paddle across it.
func icRaft(p *icv) {
	ax, ay := 0.92, -0.38 // along a log
	bx, by := 0.42, 0.9   // across the raft
	at := func(s, t float64) (float64, float64) { return 1.6 + ax*s + bx*t, 7.7 + ay*s + by*t }
	for i := 0; i < 5; i++ {
		t := float64(i) * 1.55
		x0, y0 := at(0.2*float64(i%2), t)
		x1, y1 := at(10.6-0.3*float64(i%2), t)
		a := mkAxis(x0, y0, x1, y1)
		p.fill(sCap(x0, y0, x1, y1, 0.82, 0.82), shRod(iiWood, a, 0.82, 0.02*float64(i%2), 1, uint32(851+i)))
		ang := math.Atan2(a.ey, a.ex)
		p.fill(iEll(x0, y0, 0.42, 0.8, ang), shRing(iiPale, x0, y0, 0.42, 0.8, 2, uint32(856+i)))
	}
	for _, s := range []float64{1.6, 8.8} { // the lashings across the logs
		x0, y0 := at(s, -0.7)
		x1, y1 := at(s, 6.9)
		p.fill(sCap(x0, y0, x1, y1, 0.38, 0.38), shCord(icStrawR, mkAxis(x0, y0, x1, y1), 0.38, 2.2))
	}
	// The paddle.
	px0, py0 := at(1.0, 7.4)
	px1, py1 := at(9.6, -1.2)
	pa := mkAxis(px0, py0, px1, py1)
	p.fill(sCap(px0, py0, px1, py1, 0.36, 0.32), shRod(iiPale, pa, 0.36, 0.06, 1, 861))
	bladeX, bladeY := at(10.2, -1.8)
	p.fill(iEll(bladeX, bladeY, 1.8, 0.9, math.Atan2(pa.ey, pa.ex)), shBulge(iiPale, 0.6, 0.04))
}

// icSailboat is a small boat under sail: a planked hull, a mast with a
// square sail of hide bellied by the wind, a ripple of water at her bow.
func icSailboat(p *icv) {
	// The water.
	p.fill(sEll(8.0, 14.2, 7.0, 1.3), func(f *ifrag) color.RGBA {
		l := 0.5 + 0.2*toward(f.gx, f.gy)*clamp01(1+f.d/0.5)
		if math.Mod(f.x*0.7+20, 1.4) < 0.2 {
			l += 0.25
		}
		return pick(ciWater, l)
	})
	p.bareLast()
	// The hull.
	hull := sPoly(1.2, 9.6, 14.8, 9.6, 12.6, 13.6, 3.6, 13.6)
	p.fill(iBlend(0.6, hull), func(f *ifrag) color.RGBA {
		l := 0.55 - (f.y-9.6)*0.06 - (f.x-8)*0.02
		if math.Mod(f.y*1.1+20, 1) < 0.14 {
			l -= 0.18 // the planks
		}
		if f.y < 10.3 {
			l += 0.2 // the gunwale
		}
		return pick(plankRamp, l)
	})
	// The mast and the sail.
	p.fill(sCap(7.8, 10.0, 7.8, 0.8, 0.36, 0.3), shRod(iiWood, mkAxis(7.8, 10, 7.8, 0.8), 0.36, 0.05, 1, 871))
	sail := sPoly(2.6, 1.8, 13.2, 1.8, 13.8, 8.4, 3.0, 8.4)
	p.fill(iBlend(0.4, sail).sub(sEll(8.0, 9.4, 6.4, 1.4)), func(f *ifrag) color.RGBA {
		// Bellied: lit on its left, shaded where it curves away.
		t := (f.x - 2.6) / 11.2
		l := 0.82 - 0.5*t*t + 0.06*math.Sin(f.y*0.9)
		if math.Mod(f.x*0.5+20, 1) < 0.07 {
			l -= 0.14 // its seams
		}
		return pick(icSailR, l)
	})
	p.fill(sCap(2.2, 1.6, 13.6, 1.6, 0.3, 0.3), shRod(iiWood, mkAxis(2.2, 1.6, 13.6, 1.6), 0.3, 0.05, 1, 872)) // the yard
}
