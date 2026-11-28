package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of what is picked from the plants: the flowers, the seeds,
// the herbs and greens, the marsh's and the pond's plants, and what the
// toads give. Their tones are the plants' own (floraart.go, florawet.go,
// toadart.go), so what is in the bag looks like what was picked.

func init() {
	regIcon(sim.ItemFlowerRed, icPoppy)
	regIcon(sim.ItemFlowerYellow, func(p *icv) { icDaisy(p, iiGold, iiCooked) })
	regIcon(sim.ItemFlowerWhite, func(p *icv) { icDaisy(p, iiWhite, iiGold) })
	regIcon(sim.ItemFlowerBlue, icBlueFlower)
	regIcon(sim.ItemFlowerPurple, icLavender)
	regIcon(sim.ItemFlowerPink, icWildRose)
	regIcon(sim.ItemFlowerSeed, icFlowerSeed)
	regIcon(sim.ItemBerrySeed, icBerrySeed)
	regIcon(sim.ItemTreeSeed, icTreeSeed)
	regIcon(sim.ItemGlowSeed, icGlowSeed)
	regIcon(sim.ItemManEaterSeed, icManEaterSeed)
	regIcon(sim.ItemSquashSeed, icSquashSeed)
	regIcon(sim.ItemAloe, icAloe)
	regIcon(sim.ItemHerb, icHerb)
	regIcon(sim.ItemMoss, icMoss)
	regIcon(sim.ItemSeaweed, icSeaweed)
	regIcon(sim.ItemLily, icLily)
	regIcon(sim.ItemLotusSeed, icLotusSeed)
	regIcon(sim.ItemHorsetail, icHorsetail)
	regIcon(sim.ItemPoisonMushroom, icPoisonMushroom)
	regIcon(sim.ItemNightshade, icNightshade)
	regIcon(sim.ItemToadSkin, icToadSkin)
	regIcon(sim.ItemPoisonGland, icPoisonGland)
	regIcon(sim.ItemGlowGland, icGlowGland)
}

var (
	icStemR  = mkRamp(0x1e3e1c, 0x305c24, 0x487e2e, 0x66a03c, 0x8cc054)
	icSeedR  = mkRamp(0x5a4428, 0x84683c, 0xae9258, 0xd2b87c, 0xeedcaa)
	icPipR   = mkRamp(0x4a1c14, 0x7a3020, 0xa8502c, 0xcc7a44, 0xe8a868)
	icAcornR = mkRamp(0x3a2410, 0x5e3c1a, 0x8a5a26, 0xb47e38, 0xd8a65a)
	icCupR   = mkRamp(0x2e2418, 0x4a3c26, 0x6a5834, 0x8c7848, 0xae9a62)
	icWaterR = mkRamp(0x1c4a78, 0x2c74a8, 0x4ca0d0, 0x84c8ec, 0xc8ecfc)
)

// stem paints a flower's stalk from (x0, y0) up to (x1, y1), with a leaf
// each side low on it.
func stem(p *icv, x0, y0, x1, y1 float64) {
	a := mkAxis(x0, y0, x1, y1)
	p.fill(sLock(x0, y0, (x0+x1)/2+0.5, (y0+y1)/2, x1, y1, 0.42, 0.34), shRod(icStemR, a, 0.4, 0.05, 0, 0))
	leaf(p, iiLeaf, x0+0.2, y0-2.2, 5.2, 1.25, -2.55, formLance, false, 0)
	leaf(p, iiLeaf, x0+0.1, y0-1.4, 5.0, 1.2, -0.45, formLance, false, -0.04)
}

// petal paints a petal from the flower's middle (cx, cy) out along ang:
// length long, width wide, cupped (lit on its far half), with lines down it.
func petal(p *icv, r ramp, cx, cy, length, width, ang float64, form func(t float64) float64, bias float64) {
	s := leafShape(cx, cy, length, width, ang, form)
	a := mkAxis(cx, cy, cx+math.Cos(ang)*length, cy+math.Sin(ang)*length)
	p.fill(s, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		t := u / length
		// Cupped: its inner end shaded, its outer turned up to the light.
		l := 0.45 + 0.35*t + bias + 0.12*toward(math.Cos(ang), math.Sin(ang))
		if math.Abs(math.Sin(v/width*4.5)) > 0.92 && t > 0.2 {
			l -= 0.1 // fine lines down it
		}
		if -f.d < 0.3 && toward(f.gx, f.gy) > 0.2 {
			l += 0.12
		}
		return pick(r, l)
	})
}

// --- flowers ------------------------------------------------------------------------

// icPoppy is a red poppy: four broad crinkled petals cupped round a black
// heart, the seed head in it.
func icPoppy(p *icv) {
	stem(p, 8.0, 15.2, 8.2, 8.0)
	cx, cy := 8.2, 6.2
	form := func(t float64) float64 { return math.Pow(math.Sin(math.Pi*math.Min(1, t*0.95)), 0.5) * (0.8 + 0.2*t) }
	for i, a := range []float64{-2.4, -0.75, 0.75, 2.4} {
		petal(p, iiRed, cx, cy, 5.0, 3.1, a-math.Pi/2+0.1*float64(i), form, 0.02*float64(i))
	}
	for i, a := range []float64{-1.55, 1.6} { // the two in front
		petal(p, iiRed, cx, cy+0.4, 4.4, 2.7, a+math.Pi/2*0.2-0.15*float64(i), form, 0.1)
	}
	p.fill(sEll(cx, cy, 1.7, 1.4), func(f *ifrag) color.RGBA { return pick(iiCoal, 0.15+0.1*toward(f.gx, f.gy)) })
	p.fill(sEll(cx, cy-0.2, 0.9, 0.75), shSphere(icStemR, cx, cy-0.2, 0.9, 0.75, 0.05, false))
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		p.on(sEll(cx+math.Cos(a)*1.3, cy+math.Sin(a)*1.05, 0.2, 0.2), solidC(iiCoal[1]))
	}
}

// icDaisy is a daisy: a ring of narrow petals round a domed heart of tiny
// florets, petals r, heart h.
func icDaisy(p *icv, r, h ramp) {
	stem(p, 8.0, 15.2, 8.0, 8.6)
	cx, cy := 8.0, 6.2
	for k := 0; k < 14; k++ {
		a := float64(k)*2*math.Pi/14 + 0.1
		ln := 5.2 - 0.6*math.Abs(math.Sin(a)) // foreshortened, seen a little from below
		petal(p, r, cx, cy, ln, 0.78, a, formOval, 0.08)
	}
	p.fill(sEll(cx, cy, 2.0, 1.8), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/2.0, (f.y-cy)/1.8
		l := litOf(nx*0.8, ny*0.8)
		// The florets: a dot in each cell of a spiral.
		if math.Mod(math.Hypot(nx, ny)*3+math.Atan2(ny, nx)*0.9+20, 1) < 0.35 {
			l -= 0.18
		}
		return pick(h, l)
	})
}

// icBlueFlower is a blue flower of five round petals, notched at their
// ends, round a gold eye.
func icBlueFlower(p *icv) {
	stem(p, 8.0, 15.2, 8.2, 8.0)
	cx, cy := 8.2, 6.2
	form := func(t float64) float64 {
		w := math.Sqrt(math.Max(0, math.Sin(math.Pi*math.Min(1, t*0.92))))
		if t > 0.88 {
			w *= 1 - 3*(t-0.88) // the notch
		}
		return w
	}
	for k := 0; k < 5; k++ {
		a := float64(k)*2*math.Pi/5 - math.Pi/2
		petal(p, iiBlue, cx, cy, 5.0, 2.4, a, form, 0.05)
	}
	p.fill(sEll(cx, cy, 1.3, 1.2), shSphere(iiGold, cx, cy, 1.3, 1.2, 0.05, false))
	for k := 0; k < 5; k++ {
		a := float64(k)*2*math.Pi/5 - math.Pi/2 + math.Pi/5
		p.on(sEll(cx+math.Cos(a)*0.75, cy+math.Sin(a)*0.7, 0.22, 0.22), solidC(iiCooked[0]))
	}
}

// icWildRose is a wild rose: five broad pink petals, pale at their hearts,
// a ring of gold stamens.
func icWildRose(p *icv) {
	stem(p, 8.0, 15.2, 8.2, 8.0)
	cx, cy := 8.2, 6.2
	form := func(t float64) float64 {
		return math.Pow(math.Sin(math.Pi*math.Min(1, t*0.9)), 0.45) * (0.75 + 0.25*t)
	}
	for k := 0; k < 5; k++ {
		a := float64(k)*2*math.Pi/5 - math.Pi/2 + 0.3
		petal(p, iiPink, cx, cy, 5.1, 2.9, a, form, 0.02)
	}
	p.fill(sEll(cx, cy, 1.9, 1.75), func(f *ifrag) color.RGBA { return pick(iiPink, 0.95) })
	for k := 0; k < 12; k++ {
		a := float64(k) * 2 * math.Pi / 12
		x, y := cx+math.Cos(a)*1.25, cy+math.Sin(a)*1.15
		p.fill(sCap(cx+math.Cos(a)*0.5, cy+math.Sin(a)*0.45, x, y, 0.1, 0.1), solidC(iiGold[2]))
		p.fill(sEll(x, y, 0.3, 0.3), shSphere(iiGold, x, y, 0.3, 0.3, 0.1, false))
	}
	p.fill(sEll(cx, cy, 0.7, 0.65), shSphere(iiCactus, cx, cy, 0.7, 0.65, 0.05, false))
}

// icLavender is a sprig of lavender: a spike of tiny violet florets in
// whorls up a stalk, two narrow grey-green leaves at its foot.
func icLavender(p *icv) {
	x0, y0, x1, y1 := 6.4, 15.2, 10.6, 1.4
	a := mkAxis(x0, y0, x1, y1)
	p.fill(sLock(x0, y0, 7.6, 8.0, x1, y1, 0.36, 0.22), shRod(icStemR, a, 0.36, 0.05, 0, 0))
	sage := mkRamp(0x2a4a34, 0x46684a, 0x6a8c64, 0x94b086, 0xbcd0a8)
	leaf(p, sage, 6.8, 13.6, 5.6, 0.75, -2.3, formLance, false, 0)
	leaf(p, sage, 6.9, 13.0, 5.8, 0.75, -0.75, formLance, false, -0.04)
	// The florets: whorls up the top half, smaller towards the tip.
	for i := 0; i < 8; i++ {
		t := 0.42 + 0.58*float64(i)/7
		x, y := lerpf(x0, x1, t)+0.4*math.Sin(t*3), lerpf(y0, y1, t)
		r := lerpf(1.55, 0.7, float64(i)/7)
		for j, dx := range []float64{-0.6, 0.6, 0} {
			fx, fy := x+dx*r, y+0.15*float64(j)
			s := sEll(fx, fy, r*0.62, r*0.55)
			p.fill(s, shSphere(iiPurple, fx, fy, r*0.62, r*0.55, 0.12-0.05*float64(j), false))
		}
	}
}

// --- seeds ------------------------------------------------------------------------

// seedAt paints one seed: an almond of ramp r at (x, y), rx by ry, turned
// by ang, a ridge down it and a glint.
func seedAt(p *icv, r ramp, x, y, rx, ry, ang, bias float64) {
	s := iEll(x, y, rx, ry, ang)
	a := mkAxis(x-math.Cos(ang)*rx, y-math.Sin(ang)*rx, x+math.Cos(ang)*rx, y+math.Sin(ang)*rx)
	p.fill(s, func(f *ifrag) color.RGBA {
		_, v := a.uv(f.x, f.y)
		l := litOf((f.x-x)/rx*0.6, (f.y-y)/ry*0.6)*0.9 + bias + 0.06
		if math.Abs(v+0.15*ry) < 0.08 {
			l -= 0.16
		}
		return pick(r, l)
	})
	glintAt(p, x-rx*0.35, y-ry*0.45, rx*0.2, ry*0.12)
}

// icFlowerSeed is a scatter of small brown seeds and the dried head of
// the flower they were shaken from.
func icFlowerSeed(p *icv) {
	// The dried head: a cup of bracts on a stalk.
	// The dried head: a cup of papery bracts on a stalk, open at the top.
	p.fill(sLock(4.0, 10.6, 3.4, 7.4, 4.4, 4.6, 0.26, 0.22), solidC(icCupR[3]))
	dry := mkRamp(0x5a4a30, 0x84704a, 0xa8946a, 0xcab890, 0xe6dab8)
	for k := 0; k < 5; k++ {
		a := -math.Pi/2 + (float64(k)-2)*0.42
		leaf(p, dry, 4.4, 5.0, 3.0, 0.75, a, formLance, false, -0.04*float64(k%2))
	}
	p.fill(sEll(4.4, 4.6, 1.0, 0.6), solidC(dry[0]))
	for i, s := range [][3]float64{{6.4, 11.4, 0.4}, {9.8, 8.6, -0.6}, {12.0, 12.4, 0.9}, {9.2, 13.6, 0.1}, {11.6, 5.6, 1.4}, {13.4, 9.0, -0.2}} {
		seedAt(p, icSeedR, s[0], s[1], 1.35, 0.85, s[2], -0.04*float64(i%3))
	}
}

// icBerrySeed is the pips of a berry bush and a berry they came from.
func icBerrySeed(p *icv) {
	p.fill(sEll(5.6, 5.4, 3.2, 3.2), shSphere(iiBerry, 5.6, 5.4, 3.2, 3.2, -0.04, true))
	p.fill(sCap(5.8, 2.4, 6.8, 0.9, 0.24, 0.2), solidC(icStemR[2]))
	leaf(p, iiLeaf, 6.6, 1.6, 3.6, 0.9, -0.2, formLance, false, 0)
	for i, s := range [][3]float64{{9.4, 9.0, 0.5}, {12.6, 11.4, -0.4}, {7.0, 12.0, 1.2}, {10.4, 13.6, 0.1}, {13.2, 7.4, 0.9}} {
		seedAt(p, icPipR, s[0], s[1], 1.1, 0.8, s[2], -0.03*float64(i%2))
	}
}

// icTreeSeed is a nut in its scaly cup, a twig and a leaf on it.
func icTreeSeed(p *icv) {
	cx, cy := 7.8, 9.6
	p.fill(iEll(cx, cy+0.6, 4.1, 5.0, 0.2), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/4.1, (f.y-cy)/5.0
		l := litOf(nx*0.85, ny*0.85) + 0.02
		if math.Abs(nx+0.2*ny) < 0.06 {
			l -= 0.1
		}
		return pick(icAcornR, l)
	})
	glintAt(p, cx-1.8, cy+0.4, 0.5, 0.22)
	p.on(sEll(cx+0.6, cy+5.3, 0.6, 0.45), solidC(icAcornR[0])) // its tip
	cup := iEll(cx-0.2, cy-3.0, 4.4, 2.4, 0.2)
	p.fill(cup, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.4) - 0.04
		// Scales in rows round the cup.
		if math.Mod(f.x*1.6+math.Floor(f.y*1.7)*0.5, 1) < 0.22 || math.Mod(f.y*1.7, 1) < 0.18 {
			l -= 0.16
		}
		return pick(icCupR, l)
	})
	p.fill(sLock(8.2, 1.0, 8.6, 2.2, 7.8, 5.0, 0.32, 0.36), shRod(icBarkR, mkAxis(8.2, 1.0, 7.8, 5.0), 0.34, 0.1, 0, 0))
	leaf(p, iiLeaf, 8.6, 1.8, 5.6, 1.5, 0.15, formOval, true, 0)
}

// icGlowSeed is seeds of the plants that glow at night: pale seeds lit
// from within, cyan, a glow about them.
func icGlowSeed(p *icv) {
	ss := [][4]float64{{5.4, 6.0, 0.5, 0}, {10.6, 4.8, -0.6, 0}, {7.6, 11.0, 0.2, 0}, {12.2, 11.4, 1.0, 1}}
	for _, s := range ss {
		p.soft(sEll(s[0], s[1], 3.6, 3.6), func(f *ifrag) color.RGBA {
			d := math.Hypot(f.x-s[0], f.y-s[1]) / 3.6
			return color.RGBA{0x56, 0xe2, 0xe2, uint8(80 * clamp01(1-d) * clamp01(1-d))}
		})
	}
	for _, s := range ss {
		r := glowCyan
		if s[3] > 0 {
			r = iiPurple
		}
		x, y, ang := s[0], s[1], s[2]
		p.fill(iEll(x, y, 1.9, 1.25, ang), func(f *ifrag) color.RGBA {
			d := math.Hypot((f.x-x)/1.9, (f.y-y)/1.25)
			return pick(r, 1.0-d*0.55+0.1*toward(f.gx, f.gy))
		})
		glintAt(p, x-0.6, y-0.4, 0.34, 0.14)
	}
}

// icManEaterSeed is the seed of the man-eater: a fat purple pod, a row of
// pale teeth round its seam, a red mouth showing between them.
func icManEaterSeed(p *icv) {
	cx, cy := 8.0, 8.8
	body := sEll(cx, cy, 5.8, 5.2)
	p.fill(body, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/5.8, (f.y-cy)/5.2
		l := litOf(nx*0.85, ny*0.85)
		if math.Mod(math.Atan2(ny, nx)*2.4+20, 1) < 0.12 {
			l -= 0.14 // ribs
		}
		return pick(iiPurple, l)
	})
	// The seam: a red mouth with a row of teeth along it.
	seam := func(x float64) float64 { return cy + 0.6 + 0.9*math.Sin((x-cx)*0.45) }
	p.on(sFn(func(x, y float64) float64 { return math.Abs(y-seam(x)) - 0.75 }).and(body.grow(-0.4)), func(f *ifrag) color.RGBA {
		return pick(maneatMouth, 0.4+0.3*(seam(f.x)-f.y))
	})
	for i := 0; i < 7; i++ {
		x := cx - 4.2 + float64(i)*1.4
		y := seam(x)
		up := 1.0
		if i%2 == 1 {
			up = -1
		}
		p.fill(sPoly(x-0.45, y-0.7*up, x+0.45, y-0.7*up, x, y+0.5*up), func(f *ifrag) color.RGBA { return pick(icBoneR, 0.7+0.15*toward(f.gx, f.gy)) })
	}
	p.fill(sLock(cx+0.4, cy-5.0, cx+0.8, cy-6.4, cx+2.2, cy-7.0, 0.4, 0.2), solidC(icStemR[2]))
	glintAt(p, cx-3.0, cy-2.6, 0.7, 0.3)
}

// icSquashSeed is three flat pale squash seeds, each with its rim.
func icSquashSeed(p *icv) {
	for i, s := range [][3]float64{{5.0, 5.8, 0.9}, {11.0, 6.4, -0.7}, {7.8, 11.6, 0.25}} {
		x, y, ang := s[0], s[1], s[2]
		sd := tilted(iBlend(0.8, sEll(x, y+0.8, 2.3, 3.0), sEll(x, y-2.0, 1.1, 1.3)), x, y, ang)
		p.fill(sd, func(f *ifrag) color.RGBA {
			// A flat face, a little hollow, inside a raised rim.
			l := 0.5 - 0.12*toward(f.x-x, f.y-y)/3 + 0.04*float64(i)
			if -f.d < 0.5 {
				l = 0.62 + 0.3*toward(f.gx, f.gy) // the raised rim
			} else if -f.d < 0.7 {
				l -= 0.12
			}
			return pick(seedRamp, l)
		})
		gx, gy := x-1.2, y-0.6
		glintAt(p, gx, gy, 0.32, 0.12)
	}
}

// --- herbs and greens ---------------------------------------------------------------------

// icAloe is two thick aloe leaves, cut: the clear gel at the cut end, pale
// teeth along their edges.
func icAloe(p *icv) {
	al := func(x0, y0, x1, y1, w float64, bias float64) {
		s := sTaper(w, w*0.9, 0.08, x0, y0, (x0+x1)/2+0.6, (y0+y1)/2+0.4, x1, y1)
		a := mkAxis(x0, y0, x1, y1)
		p.fill(s, func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := a.rod(v, w*0.9)*0.9 + 0.08 + bias
			if hashf(int(u*2.2), int(v*4+20), 783) < 0.12 {
				l += 0.16 // pale flecks
			}
			return pick(aloeRamp, l)
		})
		// Teeth along the lower edge.
		for t := 0.15; t < 0.9; t += 0.12 {
			x, y := lerpf(x0, x1, t), lerpf(y0, y1, t)
			cx, cy := a.across()
			hw := w * (1 - t*0.8)
			p.under(sEll(x+cx*(hw+0.15), y+cy*(hw+0.15), 0.22, 0.22), solidC(aloeRamp[4]))
		}
	}
	al(3.2, 13.0, 13.6, 2.4, 2.1, 0)
	al(7.4, 14.2, 14.6, 8.6, 1.5, -0.06)
	p.fill(iEll(3.2, 13.0, 1.55, 2.1, -0.8), func(f *ifrag) color.RGBA {
		d := math.Hypot(f.x-3.2, f.y-13.0) / 2.0
		if d > 0.75 {
			return aloeRamp[2]
		}
		return mix(hex(0xd8eec4), hex(0xf4fce8), 1-d) // the gel
	})
	glintAt(p, 2.7, 12.4, 0.4, 0.2)
}

// icHerb is a sprig of a healing herb: saw-edged leaves up a stalk, a head
// of small cream flowers.
func icHerb(p *icv) {
	x0, y0 := 6.6, 15.2
	a := mkAxis(x0, y0, 8.4, 4.4)
	p.fill(sLock(x0, y0, 7.0, 10.0, 8.4, 4.4, 0.4, 0.3), shRod(herbLeafRamp, a, 0.4, -0.1, 0, 0))
	leaf(p, herbLeafRamp, 6.9, 12.8, 6.2, 1.9, -2.6, formSaw, true, 0)
	leaf(p, herbLeafRamp, 7.1, 12.0, 6.6, 1.9, -0.45, formSaw, true, -0.02)
	leaf(p, herbLeafRamp, 7.6, 9.4, 4.6, 1.4, -2.1, formSaw, true, 0.04)
	leaf(p, herbLeafRamp, 7.8, 8.8, 4.6, 1.4, -0.9, formSaw, true, 0.02)
	for k := 0; k < 11; k++ {
		ang := math.Pi * (0.05 + 0.9*float64(k)/10)
		x, y := 8.6-math.Cos(ang)*3.4, 4.6-math.Sin(ang)*2.0-0.4*float64(k%2)
		p.fill(sEll(x, y, 0.9, 0.8), shSphere(creamRamp, x, y, 0.9, 0.8, 0.05-0.08*float64(k%2), false))
	}
}

// icMoss is three soft cushions of moss, its shoots catching the light,
// thin stalks of its capsules standing out of them.
func icMoss(p *icv) {
	pad := func(cx, cy, rx, ry float64, seed uint32) {
		s := iEll(cx, cy, rx, ry, 0).rough(0.35, 0.5, seed)
		p.fill(s, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
			l := litOf(nx*0.85, ny*0.85)*0.9 + 0.05
			n := vnoise(f.x*3.2, f.y*3.2, seed+1)
			if n > 0.62 {
				l += 0.18 // the tips of the shoots
			} else if n < 0.3 {
				l -= 0.16
			}
			return pick(mossPadRamp, l)
		})
	}
	pad(10.4, 7.2, 4.2, 3.2, 811)
	pad(4.8, 8.8, 3.4, 2.8, 812)
	pad(7.8, 11.4, 6.4, 3.6, 813)
	for i, s := range [][2]float64{{5.4, 7.0}, {9.8, 5.0}, {11.6, 9.4}, {7.2, 9.6}} {
		x, y := s[0], s[1]
		h := 2.4 + 0.6*float64(i%2)
		p.fill(sCap(x, y, x+0.3, y-h, 0.12, 0.1), solidC(hex(0xa86a2c)))
		p.fill(sEll(x+0.3, y-h, 0.32, 0.45), shSphere(icAcornR, x+0.3, y-h, 0.32, 0.45, 0.1, false))
	}
}

// icSeaweed is a hank of kelp: long ribbons, frilled at their edges and
// swaying, the bladders that float them at their feet, wet and glossy.
func icSeaweed(p *icv) {
	frond := func(x0, y0, cx, cy, x1, y1, w float64, bias float64, seed uint32) {
		const n = 16
		var pts []float64
		for i := 0; i <= n; i++ {
			t := float64(i) / n
			a, b, c := (1-t)*(1-t), 2*t*(1-t), t*t
			pts = append(pts, a*x0+b*cx+c*x1, a*y0+b*cy+c*y1)
		}
		body := sTaper(w*0.4, w, 0.15, pts...)
		ax := mkAxis(x0, y0, x1, y1)
		s := shape{body.x0 - 0.4, body.y0 - 0.4, body.x1 + 0.4, body.y1 + 0.4, func(x, y float64) float64 {
			u, _ := ax.uv(x, y)
			return body.d(x, y) + 0.22*math.Sin(u*4.4) // the frilled edge
		}}
		p.fill(s, func(f *ifrag) color.RGBA {
			u, v := ax.uv(f.x, f.y)
			l := 0.55 + bias + 0.18*toward(f.gx, f.gy)*clamp01(1+f.d/0.6)
			if math.Abs(v-0.4*math.Sin(u*0.6)) < 0.12 {
				l -= 0.14 // the midrib
			}
			if math.Mod(u*1.4+math.Abs(v)*0.8+fbm2(u, v, seed, 2), 1) < 0.12 {
				l -= 0.1 // ripples across it
			}
			return pick(kelpRamp, l)
		})
	}
	frond(5.6, 14.6, 1.4, 9.0, 4.4, 1.2, 1.9, 0.02, 851)
	frond(7.0, 14.8, 12.8, 10.4, 12.6, 1.6, 2.0, 0.06, 852)
	frond(6.4, 14.6, 7.6, 9.0, 8.2, 3.6, 1.5, -0.08, 853)
	for _, b := range [][2]float64{{4.6, 12.8}, {8.2, 13.6}, {6.0, 11.2}} {
		p.fill(sEll(b[0], b[1], 1.15, 1.0), shSphere(kelpRamp, b[0], b[1], 1.15, 1.0, 0.12, true))
	}
}

// icLily is a water lily's flower on its pad: pink pointed petals, a gold
// heart, the round pad notched, wet.
func icLily(p *icv) {
	cx, cy := 8.0, 11.0
	p.fill(sEll(cx, cy, 7.2, 3.4).sub(sPoly(cx, cy, cx+7.6, cy+1.2, cx+7.6, cy+3.6)), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/7.2, (f.y-cy)/3.4
		l := 0.55 - nx*0.12 - ny*0.08
		if math.Mod(math.Atan2(ny, nx)*2.8+20, 1) < 0.08 {
			l -= 0.12 // veins out from the middle
		}
		if -f.d < 0.45 && toward(f.gx, f.gy) > 0.2 {
			l += 0.18
		}
		return pick(padRamp, l)
	})
	bx, by := 8.0, 8.0
	form := func(t float64) float64 { return math.Pow(math.Sin(math.Pi*t), 0.8) }
	for i, a := range []float64{-2.7, -0.45, -1.95, -1.2, -2.35, -0.8} { // back petals, then front ones
		petal(p, lilyPetalRamp, bx, by, 5.2-0.3*float64(i%2), 1.45, a, form, 0.04*float64(i))
	}
	for _, a := range []float64{-3.0, -0.15, 0.4, 2.7} {
		petal(p, lilyPetalRamp, bx, by+0.4, 4.0, 1.3, a, form, 0.12)
	}
	p.fill(sEll(bx, by-0.4, 1.5, 1.0), shSphere(iiGold, bx, by-0.4, 1.5, 1.0, 0.08, false))
	glintAt(p, 3.4, 10.0, 0.5, 0.16)
}

// icLotusSeed is a lotus's seed head, a cone with its seeds set in holes
// on its flat top, and two seeds shaken out.
func icLotusSeed(p *icv) {
	cx, cy := 7.0, 6.4
	p.fill(sLock(cx+0.4, cy+3.6, cx+1.2, cy+6.0, cx+0.4, 15.2, 0.4, 0.34), shRod(podRamp, mkAxis(cx, cy, cx, 15), 0.4, 0, 0, 0))
	cone := sPoly(cx-5.0, cy-0.9, cx+5.0, cy-0.9, cx+1.6, cy+5.4, cx-1.6, cy+5.4)
	p.fill(iBlend(1.0, cone), shBulge(podRamp, 1.8, -0.04))
	top := sEll(cx, cy-0.9, 5.2, 2.0)
	p.fill(top, func(f *ifrag) color.RGBA {
		return pick(podRamp, 0.72-0.12*(f.x-cx)/5.2+0.12*toward(f.gx, f.gy)*clamp01(1+f.d/0.4))
	})
	for i, h := range [][2]float64{{-3.0, 0}, {-1.0, -0.8}, {1.1, -0.7}, {3.0, 0.1}, {-1.6, 0.8}, {0.9, 0.8}, {0, 0}} {
		x, y := cx+h[0], cy-0.9+h[1]
		p.on(sEll(x, y, 0.62, 0.42), solidC(podRamp[0]))
		if i%2 == 0 {
			p.on(sEll(x+0.05, y-0.05, 0.38, 0.26), solidC(hex(0xe8e2b0)))
		}
	}
	for _, s := range [][2]float64{{11.6, 12.4}, {13.4, 9.6}} {
		seedAt(p, seedRamp, s[0], s[1], 1.3, 1.6, 0.3, 0)
	}
}

// icHorsetail is two jointed stems of horsetail, whorls of thin branches
// at each joint, dark bands round the joints.
func icHorsetail(p *icv) {
	for i, s := range [][5]float64{{4.6, 15.0, 8.8, 1.8, 0.62}, {9.6, 15.2, 12.8, 4.4, 0.55}} {
		x0, y0, x1, y1, r := s[0], s[1], s[2], s[3], s[4]
		a := mkAxis(x0, y0, x1, y1)
		bias := -0.06 * float64(i)
		for j := 1; j <= 5; j++ { // whorls of thin branches at the joints
			t := float64(j) / 5.8
			jx, jy := lerpf(x0, x1, t), lerpf(y0, y1, t)
			ln := 4.4 * (1 - t*0.65)
			for _, d := range []float64{-1, 1} {
				ex, ey := jx+d*ln*0.85, jy-ln*0.55
				p.fill(sLock(jx, jy, jx+d*ln*0.5, jy-ln*0.45, ex, ey, 0.17, 0.08), shBulge(horsetailRamp, 0.15, 0.1+bias))
			}
		}
		p.fill(sCap(x0, y0, x1, y1, r, r*0.6), func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := a.rod(v, r) + bias
			if m := math.Mod(u/a.l*5.8+20, 1); m < 0.1 {
				return pick(icCupR, 0.3) // the dark sheath at a joint
			} else if m < 0.2 {
				l += 0.12
			}
			if math.Abs(math.Sin(v/r*3*math.Pi)) > 0.85 {
				l -= 0.1 // its ridges
			}
			return pick(horsetailRamp, l)
		})
		p.fill(iEll(x1, y1, r*0.6, r*1.1, math.Atan2(a.ey, a.ex)), shBulge(icCupR, 0.3, 0.1)) // the cone at its tip
	}
}

// icPoisonMushroom is two toadstools, violet caps freckled with sickly
// yellow warts, pale gills and stems with torn rings.
func icPoisonMushroom(p *icv) {
	ts := func(x, base, h, r, tilt float64, seed uint32) {
		tx := x + tilt*h
		st := mkAxis(x, base, tx, base-h)
		p.fill(sCap(x, base, tx, base-h, r*0.3, r*0.24), shRod(poisonStem, st, r*0.3, 0.06, 0, 0))
		ry := lerpf(x, tx, 0.6)
		p.fill(iEll(ry, base-h*0.6, r*0.42, r*0.16, 0), shBulge(poisonStem, 0.2, -0.1))
		p.fill(iEll(tx, base-h+0.2, r*0.92, r*0.3, 0), func(f *ifrag) color.RGBA {
			if math.Mod(f.x*3.4, 1) < 0.5 {
				return poisonCap[4]
			}
			return poisonCap[2]
		})
		cap := iEll(tx, base-h, r, r*0.8, 0).and(sAbove(base - h + 0.25))
		p.fill(cap, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-tx)/r, (f.y-base+h)/(r*0.8)
			l := litOf(nx*0.85, ny*0.85)
			cx, cy := math.Floor(f.x/1.1), math.Floor(f.y/0.9)
			wx, wy := (cx+0.3+0.4*hashf(int(cx), int(cy), seed))*1.1, (cy+0.3+0.4*hashf(int(cx), int(cy), seed+1))*0.9
			if hashf(int(cx), int(cy), seed+2) < 0.6 && math.Hypot(f.x-wx, (f.y-wy)*1.25) < 0.3 {
				return mix(hex(0xe2f48a), hex(0x8aa83c), clamp01(1.1-l*1.3))
			}
			return pick(poisonCap, l)
		})
	}
	ts(11.4, 14.3, 5.6, 3.0, 0.1, 765)
	ts(5.6, 14.0, 9.2, 5.0, -0.06, 763)
	glintAt(p, 3.0, 3.4, 0.6, 0.22)
}

// icNightshade is a sprig of nightshade: dark leaves and three glossy
// black berries hanging from it.
func icNightshade(p *icv) {
	leaf(p, shadeLeafRamp, 8.2, 6.0, 6.6, 2.3, -2.6, formOval, true, 0.02)
	leaf(p, shadeLeafRamp, 8.2, 6.0, 6.4, 2.2, -0.4, formOval, true, 0)
	for _, b := range [][4]float64{{5.4, 11.0, 2.0, 0}, {11.0, 11.6, 1.9, 0}, {8.3, 9.2, 1.7, 0}} {
		p.fill(sCap(8.2, 6.0, b[0], b[1]-b[2], 0.16, 0.14), solidC(shadeLeafRamp[3]))
		p.fill(sEll(b[0], b[1]-b[2]+0.2, 0.7, 0.45), solidC(shadeLeafRamp[2])) // its calyx
	}
	for _, b := range [][3]float64{{5.4, 11.0, 2.0}, {11.0, 11.6, 1.9}, {8.3, 9.2, 1.7}} {
		p.fill(sEll(b[0], b[1], b[2], b[2]), shSphere(blackBerry, b[0], b[1], b[2], b[2], 0.08, true))
	}
}

// --- what the toads give ------------------------------------------------------------

// icToadSkin is a toad's skin laid out: green, warty, dark blotches on it.
func icToadSkin(p *icv) {
	s := iBlend(1.5, sEll(8, 8, 4.6, 5.2), sCap(4.6, 4.6, 2.6, 2.4, 1.3, 0.9), sCap(11.4, 4.6, 13.4, 2.4, 1.3, 0.9),
		sCap(4.6, 11.6, 2.4, 13.8, 1.4, 0.9), sCap(11.4, 11.6, 13.6, 13.8, 1.4, 0.9), sEll(8, 3, 2, 1.6))
	p.fill(s, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.0)*0.75 - 0.02
		if vnoise(f.x*1.1+3, f.y*1.2+7, 961) > 0.68 {
			return pick(toadBlotch[2:], l)
		}
		// Warts: small domes on a jittered grid.
		cx, cy := math.Floor(f.x/1.3), math.Floor(f.y/1.3)
		wx, wy := (cx+0.25+0.5*hashf(int(cx), int(cy), 962))*1.3, (cy+0.25+0.5*hashf(int(cx), int(cy), 963))*1.3
		if d := math.Hypot(f.x-wx, f.y-wy); d < 0.38 {
			l += 0.24 * toward(f.x-wx, f.y-wy) / 0.38
			l += 0.1
		}
		return pick(toadGreen, l)
	})
}

// icPoisonGland is a toad's poison gland: an orange sac, veined, a green
// drop of its venom beneath.
func icPoisonGland(p *icv) {
	sac := iBlend(1.2, sEll(7.4, 7.0, 4.8, 4.0), sEll(11.2, 4.6, 2.0, 1.7))
	p.fill(sac, func(f *ifrag) color.RGBA {
		l := bulge(f, 3.0)*0.9 + 0.06
		if math.Abs(fbm2(f.x*0.5, f.y*0.5, 971, 2)-0.5) < 0.03 {
			l -= 0.24 // veins
		}
		return pick(toadGland, l)
	})
	glintAt(p, 5.2, 5.0, 0.7, 0.28)
	drop := iBlend(0.6, sEll(8.0, 13.4, 1.5, 1.6), sCap(8.0, 12.8, 8.0, 10.8, 0.7, 0.15))
	dr := mkRamp(0x3c6c14, 0x64a01c, 0x9cd42c, 0xd4f66c)
	p.fill(drop, shSphere(dr, 8.0, 13.4, 1.5, 1.6, 0.05, true))
}

// icGlowGland is the glow toad's gland: a pale sac lit from within, its
// light spilling round it.
func icGlowGland(p *icv) {
	p.soft(sEll(7.8, 8.4, 7.6, 7.2), func(f *ifrag) color.RGBA {
		d := math.Hypot((f.x-7.8)/7.6, (f.y-8.4)/7.2)
		return color.RGBA{0x74, 0xf8, 0xc8, uint8(110 * clamp01(1-d) * clamp01(1-d))}
	})
	sac := iBlend(1.2, sEll(7.8, 8.6, 4.4, 3.8), sEll(10.6, 5.6, 1.9, 1.6))
	p.fill(sac, func(f *ifrag) color.RGBA {
		d := math.Hypot(f.x-7.4, f.y-8.4)
		if d < 2.4 {
			return mix(glowSpotHot, glowSpot, d/2.4)
		}
		return pick(glowBelly, bulge(f, 2.6)*0.9+0.1)
	})
	glintAt(p, 5.6, 6.0, 0.5, 0.2)
}
