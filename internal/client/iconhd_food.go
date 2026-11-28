package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of what is eaten as found or roasted at a campfire: fruit,
// roots, shellfish, nuts, grain, honey. Each is turned a little and lit
// from the top left; roasting browns it, chars it in places and makes it
// steam. Tones are the world's (fruitart.go, farmart.go, crab.go,
// floracrop.go, jungleart.go).

func init() {
	regIcon(sim.ItemCookedFish, icCookedFish)
	regIcon(sim.ItemRoastBerry, icRoastBerry)
	regIcon(sim.ItemPotato, icPotato)
	regIcon(sim.ItemRoastPotato, icRoastPotato)
	regIcon(sim.ItemCactus, icCactus)
	regIcon(sim.ItemMushroom, icMushroom)
	regIcon(sim.ItemClam, icClam)
	regIcon(sim.ItemRoastClam, icRoastClam)
	regIcon(sim.ItemCrabMeat, icCrabMeat)
	regIcon(sim.ItemCookedCrab, icCookedCrab)
	regIcon(sim.ItemFig, icFig)
	regIcon(sim.ItemGinkgoNut, icGinkgo)
	regIcon(sim.ItemRoastGinkgo, icRoastGinkgo)
	regIcon(sim.ItemCoconut, icCoconut)
	regIcon(sim.ItemHoney, icHoney)
	regIcon(sim.ItemGourd, icGourd)
	regIcon(sim.ItemPitcherWater, icPitcherWater)
	regIcon(sim.ItemRice, icRice)
	regIcon(sim.ItemRoastRice, icRoastRice)
	regIcon(sim.ItemCorn, func(p *icv) { icCorn(p, false) })
	regIcon(sim.ItemRoastCorn, func(p *icv) { icCorn(p, true) })
	regIcon(sim.ItemSquash, icSquash)
	regIcon(sim.ItemRoastSquash, icRoastSquash)
}

var (
	icCharR      = mkRamp(0x1a0e0a, 0x341c12, 0x52301c, 0x744624, 0x9a6230) // the black of the fire
	icFleshR     = mkRamp(0x8a4a14, 0xc07020, 0xe89a30, 0xf8c454, 0xfde494) // a roast root's flesh
	icClamShellR = mkRamp(0x5a4038, 0x8a6a5c, 0xb8988a, 0xdcc4b4, 0xf6eade, 0xffffff)
	icClamMeat   = mkRamp(0x8a3c1c, 0xc0602a, 0xe08a44, 0xf4b468, 0xfcdc9c)
	icCrabRawR   = mkRamp(0x3a1410, 0x5e2216, 0x86301c, 0xa84a2a, 0xc8703c) // a live crab's shell, darker
	icCrabCook   = mkRamp(0x6e1810, 0xb02a18, 0xe0462a, 0xf87a48, 0xffb070) // crab.go's cooked
	icCrabFlesh  = mkRamp(0xc0b0a8, 0xdcd0c8, 0xf0e8e2, 0xfffaf6)
	icFigIn      = mkRamp(0x7a1030, 0xb02448, 0xd84a62, 0xf08a8a, 0xf8c8b0)
	icNutPale    = mkRamp(0x8a7a4a, 0xb4a26a, 0xd8c890, 0xf0e4b8, 0xfffae0)
	icKernelG    = mkRamp(0x3c6a28, 0x5c9234, 0x86b848, 0xb4d86c, 0xdcf0a0) // a roast ginkgo's kernel
	icHusk       = mkRamp(0x2a1a10, 0x45301c, 0x644828, 0x846336, 0xa48250) // coconutRamp and a step lighter
	icCoconutIn  = mkRamp(0xc8c0b0, 0xe4dccc, 0xf6f2ea, 0xffffff)
	icHoneyR     = mkRamp(0x6e3e08, 0xa86410, 0xd89418, 0xf4c030, 0xffe070, 0xfff6c0)
	icCombR      = mkRamp(0x6e4a18, 0x946820, 0xbb8a2c, 0xd9ae4a, 0xf0d070) // hiveRamp's wax
)

// steam rises from (x, y): wisps up to h units, in a soft white.
func steam(p *icv, xs []float64, y, h float64) {
	for i, x := range xs {
		x, sd := x, float64(i)*1.7
		p.soft(sBox(x-1.6, y-h, x+1.6, y, 0), func(f *ifrag) color.RGBA {
			t := (y - f.y) / h
			wob := math.Sin(f.y*1.3+sd) * 0.7 * t
			d := math.Abs(f.x-x-wob) / (0.42 + 0.38*t)
			a := clamp01(1-d) * clamp01(t*3) * clamp01((1-t)*2.2) * 0.55
			return color.RGBA{0xf4, 0xf0, 0xe8, uint8(255 * a)}
		})
	}
}

// char darkens what is painted in blotches: where it lay on the coals.
func char(p *icv, s shape, amt float64, seed uint32) {
	p.on(s, func(f *ifrag) color.RGBA {
		if fbm2(f.x*0.9, f.y*0.9, seed, 2) > 1-amt {
			c := p.px[f.iy*p.n+f.ix]
			return mix(c, icCharR[1], 0.7)
		}
		return color.RGBA{}
	})
}

// icCookedFish is a fish roasted on a stick: its skin browned and crisp,
// the grill's bands across it, its eye gone white.
func icCookedFish(p *icv) {
	fr := frameOf(1.0, 10.4, 0.96, -0.28)
	// The stick through it, out at both ends.
	sx0, sy0 := fr.at(-0.6, -0.3)
	sx1, sy1 := fr.at(14.6, -0.6)
	stick := mkAxis(sx0, sy0, sx1, sy1)
	body := iBlend(1.2,
		sTaper(0.95, 2.9, 0.7, fr.path(0, 0, 3.4, 0.1, 7.2, 0.05, 10.9, 0)...),
		fr.poly(4.4, 2.2, 6.6, 3.7, 8.8, 2.0))
	tail := fr.poly(10.2, 0, 13.4, 3.0, 14.4, 2.7, 13.1, 0, 14.4, -2.7, 13.4, -3.0)
	p.fill(tail, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := 0.4 + t*0.04
		if math.Mod(math.Atan2(t, s-10.2)*9+50, 1) < 0.3 {
			l -= 0.18
		}
		return pick(icCharR, l+0.2)
	})
	p.fill(sCap(sx0, sy0, sx1, sy1, 0.42, 0.38), shRod(iiWood, stick, 0.42, 0, 1, 11))
	p.fill(body, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := bulge(f, 1.8) + 0.02
		if t < -0.85 {
			l += 0.1 // the paler belly
		}
		if m := math.Mod(s*0.62+t*0.5+50, 1); m < 0.14 && -f.d > 0.3 {
			l -= 0.32 // the grill's bands
		}
		l += (fbm2(f.x*1.3, f.y*1.3, 12, 2) - 0.5) * 0.2
		return pick(icRoastR, l)
	})
	char(p, body, 0.18, 13)
	ex, ey := fr.at(1.45, 0.5)
	p.on(sEll(ex, ey, 0.5, 0.5), solidC(hex(0xf0ece0)))
	p.on(sEll(ex+0.05, ey+0.05, 0.22, 0.22), solidC(icCharR[2]))
	p.on(sTaper(0.08, 0.24, 0.06, fr.path(3.6, 1.7, 6.0, 2.3, 8.6, 1.6)...), solidC(mix(icGlint, icRoastR[4], 0.3)))
	steam(p, []float64{6.0, 9.4}, 6.0, 5.4)
}

// icRoastBerry is berries roasted on a stick: their skins split and
// browned, juice shining on them.
func icRoastBerry(p *icv) {
	a := mkAxis(2.0, 14.4, 14.2, 2.0)
	p.fill(sCap(2.0, 14.4, 14.2, 2.0, 0.45, 0.38), shRod(iiWood, a, 0.45, 0, 1, 16))
	roast := mkRamp(0x4a1020, 0x7a1c30, 0xa8303c, 0xd05a4c, 0xf09a7a)
	for i, t := range []float64{0.24, 0.42, 0.6, 0.78} {
		x, y := lerpf(2.0, 14.2, t), lerpf(14.4, 2.0, t)
		r := 2.05 - 0.1*float64(i%2)
		s := sEll(x, y, r, r)
		p.fill(s, shSphere(roast, x, y, r, r, -0.04, true))
		// A split in the skin, pale flesh showing.
		p.on(sCap(x+0.2, y+0.5, x+1.1, y-0.3, 0.16, 0.12), solidC(roast[4]))
		char(p, s, 0.2, uint32(160+i))
	}
	steam(p, []float64{5.2, 9.2}, 5.2, 4.6)
}

// tuber is a sweet potato from (x0, y0) to (x1, y1), fat in the middle,
// ramp r, its skin with eyes and fine roots.
func tuber(p *icv, r ramp, x0, y0, x1, y1, w float64, seed uint32) shape {
	a := mkAxis(x0, y0, x1, y1)
	mx, my := (x0+x1)/2+0.3, (y0+y1)/2+0.4
	ang := math.Atan2(a.ey, a.ex)
	s := iBlend(1.6, iEll(mx, my, a.l*0.34, w, ang), sTaper(0.45, w*0.7, 0.3, x0, y0, mx, my, x1, y1))
	p.fill(s, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		// Lit as a long round body: the normal leans out across it and
		// towards its ends.
		nu, nv := (u-a.l*0.5)/(a.l*0.55), v/(w*1.05)
		nx, ny := a.ex*nu*0.5+(-a.ey)*nv*0.9, a.ey*nu*0.5+a.ex*nv*0.9
		l := litOf(nx, ny) + 0.02
		if math.Abs(math.Sin(u*1.1+fbm2(u*0.6, v*0.6, seed, 2)*2.5)) > 0.95 && math.Abs(v) < w*0.75 {
			l -= 0.14 // the shallow lines round it
		}
		cu, cv := math.Floor(u/1.6), math.Floor(v/1.4)
		ex, ey := (cu+0.5)*1.6, (cv+0.5)*1.4
		if hashf(int(cu), int(cv)+9, seed) < 0.25 && math.Hypot(u-ex, (v-ey)*1.6) < 0.2 {
			l -= 0.2 // an eye
		}
		return pick(r, l)
	})
	return s
}

// icPotato is two sweet potatoes, red-skinned, a root hair at their ends.
func icPotato(p *icv) {
	tuber(p, potatoRamp, 1.8, 7.4, 12.6, 2.4, 2.5, 191)
	tuber(p, potatoRamp, 3.4, 13.8, 14.2, 9.2, 2.8, 192)
	p.fill(sLock(14.2, 9.2, 15.0, 8.6, 15.2, 7.0, 0.14, 0.08), solidC(potatoRamp[1]))
	p.fill(sLock(2.0, 7.8, 1.2, 8.6, 1.0, 10.2, 0.14, 0.08), solidC(potatoRamp[1]))
	glintAt(p, 5.6, 4.6, 0.5, 0.2)
	glintAt(p, 7.4, 9.6, 0.5, 0.2)
}

// icRoastPotato is a sweet potato roasted in the embers: its skin black
// and crackled, split open along its top, the gold flesh steaming.
func icRoastPotato(p *icv) {
	x0, y0, x1, y1 := 1.8, 12.6, 14.2, 5.0
	s := tuber(p, icCharR, x0, y0, x1, y1, 3.6, 201)
	_ = s
	a := mkAxis(x0, y0, x1, y1)
	// The split: a lens of flesh along it.
	fr := frameOf(x0, y0, a.ex, a.ey)
	split := iBlend(0.6, fr.poly(2.6, 0.2, 6.6, 2.2, 10.6, 1.8, 12.4, 0.4, 10.6, -0.8, 6.6, -1.0))
	p.fill(split, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.4)*0.6 + 0.4 + (fbm2(f.x*1.4, f.y*1.4, 202, 2)-0.5)*0.25
		return pick(icFleshR, l)
	})
	p.on(sTaper(0.06, 0.2, 0.05, fr.path(4.0, 1.0, 6.6, 1.6, 9.2, 1.2)...), solidC(icFleshR[4]))
	steam(p, []float64{6.0, 9.6}, 5.6, 5.2)
}

// icCactus is a pad of cactus, cut: green, its spines in tufts, the pale
// flesh at the cut, a pink fruit on its rim.
func icCactus(p *icv) {
	cx, cy := 7.4, 9.0
	pad := iEll(cx, cy, 5.6, 4.4, -0.7)
	p.fill(pad, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.4) + 0.02
		return pick(iiCactus, l)
	})
	// The cut at its foot: pale wet flesh.
	cut := iEll(3.8, 12.6, 1.6, 1.0, -0.7).and(pad.grow(0.2))
	p.fill(cut, func(f *ifrag) color.RGBA {
		return pick(mkRamp(0x8ab070, 0xb4d498, 0xd8ecc0, 0xf0faea), 0.7+0.2*toward(f.gx, f.gy))
	})
	// Tufts of spines on a grid.
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			x := cx - 3.4 + float64(i)*2.2 + float64(j%2)*1.1
			y := cy - 3.4 + float64(j)*2.2 - float64(i)*0.6
			if math.Hypot((x-cx)/5.0, (y-cy)/4.0) > 0.9 {
				continue
			}
			p.on(sEll(x, y, 0.32, 0.32), solidC(iiNut[2]))
			for k := 0; k < 3; k++ {
				ang := -2.4 + float64(k)*0.6
				p.on(sCap(x, y, x+math.Cos(ang)*1.0, y+math.Sin(ang)*1.0, 0.07, 0.03), solidC(iiNut[4]))
			}
		}
	}
	// The fruit on its rim.
	fx, fy := 11.6, 3.6
	fruit := mkRamp(0x7c1a40, 0xb03060, 0xd85480, 0xf08ca8, 0xfcc4d4)
	p.fill(iEll(fx, fy, 2.3, 2.7, 0.6), shSphere(fruit, fx, fy, 2.3, 2.7, 0, true))
	p.on(sEll(fx+0.9, fy-1.9, 0.6, 0.4), solidC(fruit[0]))
}

// icMushroom is two field mushrooms: brown-red caps, cream gills and
// stems, a little earth at their feet.
func icMushroom(p *icv) {
	m := func(x, base, h, r, tilt float64) {
		tx := x + tilt*h
		st := mkAxis(x, base, tx, base-h)
		p.fill(sCap(x, base, tx, base-h, r*0.34, r*0.28), shRod(icBoneR, st, r*0.34, 0.06, 0, 0))
		p.fill(iEll(tx, base-h+0.25, r*0.92, r*0.28, 0), func(f *ifrag) color.RGBA {
			if math.Mod(f.x*3.2, 1) < 0.45 {
				return icBoneR[2]
			}
			return icBoneR[1]
		})
		cap := iEll(tx, base-h, r, r*0.72, 0).and(sAbove(base - h + 0.3))
		p.fill(cap, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-tx)/r, (f.y-base+h)/(r*0.72)
			l := litOf(nx*0.85, ny*0.85) + 0.04
			if hashf(int(f.x*1.6), int(f.y*1.6), 221) < 0.06 {
				l += 0.2 // pale flecks
			}
			return pick(capRamp, l)
		})
		glintAt(p, tx-r*0.42, base-h-r*0.38, r*0.14, r*0.07)
	}
	m(11.2, 13.8, 4.8, 3.0, 0.12)
	m(6.0, 13.8, 7.8, 4.8, -0.06)
}

// icClam is two cockles, shut: each a fan of deep ribs out from the hinge,
// bands of growth across them in browns and creams.
func icClam(p *icv) {
	band := mkRamp(0x4a2e24, 0x7a5040, 0xa8806a, 0xd4b49c, 0xf2e2d0)
	clam := func(cx, cy, r, ang, bias float64) {
		hx, hy := cx+math.Sin(ang)*r*0.95, cy-math.Cos(ang)*r*0.95 // the hinge
		fan := iBlend(1.2*r/4, sEll(cx, cy+0.1*r, r, r*0.86), sPoly(cx-r*0.55, cy-r*0.3, cx+r*0.55, cy-r*0.3, cx, cy-r*1.02))
		s := tilted(fan, cx, cy, ang)
		p.fill(s, func(f *ifrag) color.RGBA {
			d := math.Hypot(f.x-hx, f.y-hy)
			ra := math.Atan2(f.y-hy, f.x-hx)
			// A rib: lit on the side towards the light, dark in the groove.
			rib := math.Mod(ra*7.5+20, 1)
			l := bulge(f, r*0.7)*0.8 + 0.12 + bias + 0.16*math.Cos(rib*2*math.Pi+2)
			if rib < 0.12 {
				l -= 0.16
			}
			c := pick(icClamShellR, l)
			if b := math.Mod(d*0.6+0.12*fbm2(f.x, f.y, 431, 2), 1); b < 0.3 && d > r*0.3 {
				c = mix(c, pick(band, l), 0.7) // a darker band of growth
			}
			return c
		})
	}
	clam(10.6, 10.8, 3.9, 0.45, -0.02)
	clam(5.8, 6.8, 4.3, -0.3, 0.04)
}

// icRoastClam is a clam opened by the fire: one half shell, the cooked
// flesh in it, juice bubbling, steam.
func icRoastClam(p *icv) {
	cx, cy := 8.0, 10.0
	// The upper shell, open behind.
	p.fill(iEll(cx+0.6, cy-4.0, 5.6, 2.6, -0.15), func(f *ifrag) color.RGBA {
		l := bulge(f, 1.6) - 0.1
		if math.Mod(math.Atan2(f.y-cy, f.x-cx)*10+20, 1) < 0.22 {
			l -= 0.14
		}
		return pick(icClamShellR, l)
	})
	shell := iEll(cx, cy, 6.6, 4.2, 0.08)
	p.fill(shell, func(f *ifrag) color.RGBA {
		l := 0.75 - 0.18*clamp01(1+f.d/1.2) + 0.1*toward(f.gx, f.gy)
		return pick(icClamShellR, l) // the pearly inside
	})
	meat := iBlend(1.0, sEll(cx-0.4, cy-0.2, 4.2, 2.6), sEll(cx+2.2, cy+0.6, 2.2, 1.6))
	p.fill(meat, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.6) + (fbm2(f.x*1.2, f.y*1.2, 231, 2)-0.5)*0.3
		return pick(icClamMeat, l)
	})
	char(p, meat, 0.14, 232)
	glintAt(p, cx-2.2, cy-1.2, 0.6, 0.24)
	steam(p, []float64{5.6, 9.6}, 6.6, 6.2)
}

// claw paints a crab's claw from its joint at (x, y) out along ang: the
// palm swollen, the two pincers.
func claw(p *icv, r ramp, x, y, ang, s float64) shape {
	fr := frameOf(x, y, math.Cos(ang), math.Sin(ang))
	palm := iBlend(1.0*s, fr.poly(0, -1.0*s, 0, 1.0*s, 3.6*s, 2.2*s, 6.0*s, 1.8*s, 6.6*s, 0, 5.6*s, -1.6*s, 3.0*s, -1.8*s))
	finger := fr.poly(5.6*s, 1.6*s, 9.6*s, 1.6*s, 10.8*s, 0.6*s, 8.6*s, 0.4*s, 6.2*s, 0.2*s)
	thumb := fr.poly(5.4*s, -0.6*s, 8.0*s, -0.6*s, 9.6*s, -0.2*s, 8.4*s, -1.4*s, 5.8*s, -1.6*s)
	sh := func(f *ifrag) color.RGBA {
		l := bulge(f, 1.2*s) + 0.04
		if hashf(int(f.x*3), int(f.y*3), 241) < 0.05 {
			l += 0.16 // bumps on the shell
		}
		return pick(r, l)
	}
	p.fill(thumb, sh)
	p.fill(palm.or(finger), sh)
	tx, ty := fr.at(10.4, 0.8)
	p.on(sEll(tx, ty, 0.5*s, 0.4*s), solidC(r[0])) // the dark tip
	return palm
}

// icCrabMeat is a crab's claw broken open: the shell dark and bumpy, the
// white meat out of it, glistening.
func icCrabMeat(p *icv) {
	claw(p, icCrabRawR, 1.6, 12.6, -0.75, 1.15)
	meat := iBlend(1.0, sEll(4.4, 12.4, 2.6, 2.0), sEll(6.6, 13.8, 2.0, 1.4))
	p.fill(meat, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.4)
		if math.Abs(math.Sin((f.x*0.8-f.y*1.6)*math.Pi)) > 0.9 {
			l -= 0.12 // its fibres
		}
		c := pick(icCrabFlesh, l)
		if f.y < 11.8 {
			c = mix(c, icCrabCook[3], 0.25)
		}
		return c
	})
	glintAt(p, 3.4, 11.4, 0.5, 0.2)
	glintAt(p, 7.6, 5.4, 0.36, 0.16)
}

// icCookedCrab is a whole crab boiled bright red: its round back, legs
// folded each side, both claws forward.
func icCookedCrab(p *icv) {
	cx, cy := 8.0, 9.4
	for _, side := range []float64{-1, 1} {
		for i := 0; i < 3; i++ {
			y := cy + 0.4 + float64(i)*1.1
			x0 := cx + side*3.8
			pts := []float64{x0, y, x0 + side*2.0, y - 0.8, x0 + side*3.2, y + 1.0 + float64(i)*0.3}
			p.fill(sPath(0.48, 0.28, pts...), shBulge(icCrabCook, 0.4, -0.1))
		}
	}
	claw(p, icCrabCook, cx-2.6, cy-2.0, -2.2, 0.62)
	claw(p, icCrabCook, cx+2.6, cy-2.0, -0.94, 0.62)
	back := iEll(cx, cy, 5.0, 3.6, 0)
	p.fill(back, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/5.0, (f.y-cy)/3.6
		l := litOf(nx*0.8, ny*0.8) + 0.02
		if hashf(int(f.x*2.6), int(f.y*2.6), 251) < 0.05 {
			l += 0.14
		}
		if math.Abs(nx) < 0.05 && ny > -0.4 {
			l -= 0.12
		}
		return pick(icCrabCook, l)
	})
	for _, side := range []float64{-1, 1} { // its eyes on stalks
		ex, ey := cx+side*1.2, cy-3.6
		p.fill(sCap(cx+side*0.9, cy-2.8, ex, ey, 0.22, 0.2), solidC(icCrabCook[1]))
		p.fill(sEll(ex, ey, 0.45, 0.45), solidC(icCharR[0]))
	}
	glintAt(p, cx-2.2, cy-1.6, 0.8, 0.3)
	steam(p, []float64{5.8, 10.6}, 5.8, 5.6)
}

// icFig is two figs and a leaf: one whole, purple and soft, one split to
// show its rose flesh full of seeds.
func icFig(p *icv) {
	leaf(p, figLeafRamp, 9.6, 6.0, 6.0, 2.4, -0.6, formOval, true, 0.04)
	fig := func(cx, cy, r float64, ang float64) shape {
		s := tilted(iBlend(1.0, sEll(cx, cy+0.4*r, r, r), sEll(cx, cy-0.9*r, r*0.42, r*0.5)), cx, cy, ang)
		p.fill(s, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-cx)/r, (f.y-cy-0.3*r)/r
			l := litOf(nx*0.85, ny*0.85) + 0.02
			if math.Mod(math.Atan2(ny, nx)*2.2+20, 1) < 0.1 {
				l -= 0.1 // faint ribs
			}
			return pick(figRamp, l)
		})
		return s
	}
	fig(5.4, 7.6, 3.4, -0.35)
	glintAt(p, 3.8, 6.4, 0.6, 0.24)
	// The cut half, its face to us.
	cx, cy, r := 10.4, 11.0, 3.6
	half := tilted(iBlend(1.0, sEll(cx, cy+0.3*r, r, r*0.95), sEll(cx, cy-0.9*r, r*0.42, r*0.5)), cx, cy, 0.4)
	p.fill(half, func(f *ifrag) color.RGBA { return pick(figRamp, 0.35+0.3*toward(f.gx, f.gy)) })
	in := half.grow(-0.55)
	p.fill(in, func(f *ifrag) color.RGBA {
		d := -f.d
		if d < 0.45 {
			return pick(icNutPale, 0.7) // the pale rind inside the skin
		}
		l := 0.6 - 0.25*clamp01(d/2.2)
		// The seeds: tiny gold specks in streaks out from the middle.
		if hashf(int(f.x*4), int(f.y*4), 631) < 0.14 {
			return pick(iiGold, 0.8)
		}
		return pick(icFigIn, l)
	})
}

// nut paints a smooth nut at (x, y), rx by ry turned by ang, of ramp r, a
// seam along it and a point at its end.
func nut(p *icv, r ramp, x, y, rx, ry, ang, bias float64) shape {
	s := tilted(iBlend(0.5, sEll(x, y, rx, ry), sEll(x+rx*0.85, y, rx*0.3, ry*0.3)), x, y, ang)
	p.fill(s, func(f *ifrag) color.RGBA {
		dx, dy := f.x-x, f.y-y
		u, v := dx*math.Cos(ang)+dy*math.Sin(ang), -dx*math.Sin(ang)+dy*math.Cos(ang)
		l := litOf(dx/rx*0.8, dy/ry*0.8) + bias
		if math.Abs(v-0.1*u) < 0.09 {
			l -= 0.15 // the seam
		}
		return pick(r, l)
	})
	return s
}

// icGinkgo is the ginkgo's fruit as it falls: soft gold plums on their
// stalks, and two of the pale stones from inside them.
func icGinkgo(p *icv) {
	for _, n := range [][2]float64{{10.6, 11.4}, {13.2, 7.2}} {
		nut(p, icNutPale, n[0], n[1], 2.2, 1.6, 0.4, 0.02)
		glintAt(p, n[0]-0.8, n[1]-0.6, 0.34, 0.14)
	}
	for i, n := range [][3]float64{{4.8, 7.4, 3.0}, {8.2, 5.2, 2.6}} {
		x, y, r := n[0], n[1], n[2]
		p.fill(sCap(x+0.4, y-r+0.2, x+1.6-float64(i), y-r-2.0, 0.2, 0.16), solidC(icStemR[1]))
		p.fill(sEll(x, y, r, r*0.92), shSphere(nutRamp, x, y, r, r*0.92, 0, true))
		p.on(sEll(x+0.4, y-r+0.5, 0.45, 0.3), solidC(nutRamp[0]))
	}
}

// icRoastGinkgo is ginkgo nuts roasted: their shells toasted brown and
// cracked, the jade-green kernel showing out of one.
func icRoastGinkgo(p *icv) {
	toast := mkRamp(0x5a3010, 0x8a5018, 0xb87828, 0xdca448, 0xf2cc7c)
	nut(p, toast, 5.0, 5.6, 2.6, 1.9, 0.5, 0)
	nut(p, toast, 11.0, 6.0, 2.6, 1.9, -0.4, -0.02)
	s := nut(p, toast, 8.6, 11.4, 3.0, 2.2, 0.15, 0.02)
	char(p, s, 0.12, 651)
	// The kernel out of the cracked one.
	p.fill(iEll(9.4, 10.8, 1.9, 1.35, 0.15), shSphere(icKernelG, 9.4, 10.8, 1.9, 1.35, 0.05, true))
	p.on(sPath(0.12, 0.1, 6.4, 11.0, 7.2, 10.2, 7.6, 11.2), solidC(icCharR[1]))
	steam(p, []float64{8.0}, 5.0, 4.4)
}

// icCoconut is a coconut in its hairy husk, its three eyes, and a half of
// one broken open, white inside.
func icCoconut(p *icv) {
	cx, cy, r := 6.6, 6.8, 4.8
	p.fill(sEll(cx, cy, r, r*0.95), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/r, (f.y-cy)/(r*0.95)
		l := litOf(nx*0.85, ny*0.85)
		if math.Abs(math.Sin((f.x*1.6+f.y*2.6+fbm2(f.x, f.y, 661, 2)*3)*math.Pi)) > 0.86 {
			l -= 0.14 // the hairs of the husk
		}
		return pick(icHusk, l)
	})
	for _, e := range [][2]float64{{-1.0, -1.4}, {0.6, -1.6}, {-0.1, -0.2}} {
		p.on(sEll(cx+e[0], cy+e[1], 0.42, 0.36), solidC(icHusk[0]))
	}
	hx, hy := 10.6, 11.4
	p.fill(iEll(hx, hy+0.6, 4.6, 3.2, -0.1).and(sBelow(hy-0.6)), shBulge(icHusk, 1.4, -0.04))
	p.fill(iEll(hx, hy, 4.4, 2.0, -0.1), func(f *ifrag) color.RGBA {
		d := math.Hypot((f.x-hx)/4.4, (f.y-hy)/2.0)
		if d > 0.8 {
			return pick(icCoconutIn, 0.6+0.3*toward(f.gx, f.gy)) // the white flesh
		}
		return pick(icCoconutIn, 0.35-0.2*(f.y-hy)) // the hollow
	})
	glintAt(p, 4.4, 4.4, 0.6, 0.22)
}

// icHoney is a piece of honeycomb dripping honey: waxen cells, some
// capped, most full and shining.
func icHoney(p *icv) {
	comb := iBlend(0.8, sPoly(2.0, 4.2, 9.4, 2.0, 14.0, 4.6, 13.6, 10.8, 8.4, 12.6, 2.6, 10.6)).rough(0.2, 1.2, 891)
	p.fill(comb, func(f *ifrag) color.RGBA {
		// Hexagonal cells.
		const s = 1.25
		x, y := f.x/s, f.y/(s*0.866)
		row := math.Floor(y)
		off := 0.5 * math.Mod(row, 2)
		cx := math.Floor(x+off) - off + 0.5
		cy := row + 0.5
		dx, dy := (x-cx)*s, (y-cy)*s*0.866
		d := math.Max(math.Abs(dx)*0.866+math.Abs(dy)*0.5, math.Abs(dy))
		if d > 0.48 {
			return pick(icCombR, 0.55+0.2*toward(f.gx, f.gy)) // the wax walls
		}
		capd := hashf(int(cx), int(row), 892) < 0.3
		if capd {
			return pick(icCombR, 0.8-0.4*d)
		}
		l := 0.45 + 0.4*toward(-dx, -dy)*d/0.48 // honey in a cell, dished
		return pick(icHoneyR, l)
	})
	// Honey dripping off its lower edge.
	drip := iBlend(0.8, sCap(8.6, 11.6, 8.8, 14.4, 0.8, 0.6), sEll(8.8, 14.6, 0.95, 0.85), sCap(11.8, 10.4, 12.0, 12.6, 0.6, 0.45))
	p.fill(drip, shSphere(icHoneyR, 8.6, 12.6, 1.4, 2.4, 0.1, false))
	glintAt(p, 8.4, 13.2, 0.18, 0.5)
	glintAt(p, 5.0, 5.6, 0.6, 0.22)
}

// icGourd is a bottle gourd: pale green, freckled, its stalk curling, a
// heart leaf behind.
func icGourd(p *icv) {
	leaf(p, vineLeafRamp, 8.8, 4.0, 5.4, 2.2, -0.3, formOval, true, 0)
	cx, cy := 7.4, 10.0
	g := iBlend(0.9, sEll(cx, cy+0.6, 4.4, 4.2), sEll(cx+0.4, cy-4.6, 2.5, 2.3), sCap(cx+0.4, cy-6.0, cx+0.5, cy-7.4, 1.1, 0.9))
	p.fill(g, func(f *ifrag) color.RGBA {
		l := bulge(f, 3.2)*0.92 + 0.06
		if hashf(int(f.x*3), int(f.y*2.4), 981) < 0.06 {
			l += 0.14 // pale freckles
		}
		if math.Abs(math.Sin((f.x-cx)*0.9)) < 0.12 {
			l += 0.1 // faint pale stripes down it
		}
		return pick(gourdRamp, l)
	})
	p.fill(sLock(cx+0.4, cy-7.2, cx+0.6, cy-8.6, cx+2.0, cy-8.8, 0.36, 0.2), solidC(vineLeafRamp[1]))
	glintAt(p, cx-2.2, cy-1.4, 0.8, 0.3)
}

// icPitcherWater is a pitcher plant's cup, picked full of rain water: its
// maroon-blotched cup, the lid, the water shining in its mouth.
func icPitcherWater(p *icv) {
	x, y := 7.6, 4.6
	body := iBlend(1.6, sEll(x, y+6.6, 3.6, 4.4), sCap(x, y+4.4, x, y+0.6, 2.6, 3.0))
	p.fill(body, func(f *ifrag) color.RGBA {
		ay := (f.y - y) / 10.6
		l := bulge(f, 3.0)*0.85 + 0.3 - 0.22*(1-ay)
		if vnoise(f.x*1.4, f.y*1.0, 741) > 0.42+0.38*ay {
			l -= 0.32 // maroon blotches, thick at the mouth
		}
		return pick(pitcherRamp, l)
	})
	p.fill(iEll(x+1.8, y-2.4, 3.2, 1.3, -0.45), shBulge(pitcherRamp, 1.0, -0.06)) // the lid
	p.fill(sEll(x, y+0.4, 3.1, 1.4), func(f *ifrag) color.RGBA { return pick(pitcherRamp, 0.5+0.2*toward(f.gx, f.gy)) })
	water := sEll(x, y+0.5, 2.3, 0.9)
	p.fill(water, func(f *ifrag) color.RGBA {
		return pick(icWaterR, 0.45-0.25*(f.x-x)/2.3)
	})
	p.on(sEll(x-0.5, y+0.3, 1.2, 0.3), solidC(icWaterR[4]))
	glintAt(p, x-2.2, y+5.4, 0.6, 0.24)
	// A drop running down its side.
	p.fill(iBlend(0.4, sEll(x+3.6, y+4.0, 0.5, 0.6), sCap(x+3.5, y+3.4, x+3.4, y+2.2, 0.3, 0.1)), shSphere(icWaterR, x+3.6, y+4.0, 0.5, 0.6, 0.1, true))
}

// grain is a grain of rice or a puffed one at (x, y), rx by ry, turned.
func grain(p *icv, r ramp, x, y, rx, ry, ang, bias float64) {
	p.fill(iEll(x, y, rx, ry, ang), func(f *ifrag) color.RGBA {
		return pick(r, litOf((f.x-x)/rx*0.7, (f.y-y)/rx*0.7)+bias)
	})
}

// icRice is a heap of wild rice in the husk and a drooping head of it.
func icRice(p *icv) {
	// The stalk and its head, behind the heap.
	stalk := mkAxis(3.0, 12.0, 9.2, 1.6)
	p.fill(sLock(3.0, 12.0, 6.0, 4.0, 9.2, 1.6, 0.26, 0.18), shRod(huskRamp, stalk, 0.26, 0, 0, 0))
	for i := 0; i < 9; i++ {
		t := float64(i) / 8
		bx, by := lerpf(9.0, 13.4, t), 1.8+t*t*6.2
		grain(p, grainRamp, bx+0.6*float64(i%2), by, 0.95, 0.5, 1.2+0.4*t, 0.02*float64(i%3))
	}
	for i := 0; i < 30; i++ {
		a, d := hashf(i, 0, 881)*2*math.Pi, math.Sqrt(hashf(i, 1, 881))
		x, y := 7.6+math.Cos(a)*d*5.8, 11.6+math.Sin(a)*d*2.8-(1-d)*2.4
		grain(p, grainRamp, x, y, 1.05, 0.55, hashf(i, 2, 881)*3, (hashf(i, 3, 881)-0.5)*0.24)
	}
}

// icRoastRice is green rice toasted into puffs, heaped on a lotus leaf.
func icRoastRice(p *icv) {
	lx, ly := 8.0, 11.4
	p.fill(sEll(lx, ly, 7.2, 3.4), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-lx)/7.2, (f.y-ly)/3.4
		l := 0.5 - nx*0.12 - ny*0.08
		if math.Mod(math.Atan2(ny, nx)*2.8+20, 1) < 0.08 {
			l -= 0.12
		}
		if -f.d < 0.45 && toward(f.gx, f.gy) > 0.2 {
			l += 0.18
		}
		return pick(padRamp, l)
	})
	young := mkRamp(0x5c6c20, 0x86983a, 0xb0c260, 0xd4e08c, 0xf0f4c0) // green rice, toasted pale
	for i := 0; i < 34; i++ {
		a, d := hashf(i, 0, 882)*2*math.Pi, math.Sqrt(hashf(i, 1, 882))
		x, y := 8.0+math.Cos(a)*d*5.0, 9.2+math.Sin(a)*d*2.2-(1-d)*3.2
		r := young
		if hashf(i, 4, 882) < 0.35 {
			r = puffedRamp
		}
		grain(p, r, x, y, 1.0, 0.75, hashf(i, 2, 882)*3, (hashf(i, 3, 882)-0.5)*0.24)
	}
}

// icCorn is an ear of corn, its husk peeled back and its silk hanging;
// roasted, its kernels browned and charred where they lay on the fire.
func icCorn(p *icv, roast bool) {
	x0, y0, x1, y1 := 4.8, 11.4, 12.0, 3.8
	a := mkAxis(x0, y0, x1, y1)
	for _, side := range []float64{-1, 1} { // the husk, peeled back
		ang := math.Atan2(-a.ey, -a.ex) + side*0.55
		leaf(p, huskRamp, x0+a.ex*1.4, y0+a.ey*1.4, 5.2, 1.4, ang, formLance, false, 0.06*side)
	}
	kr := kernelRamp
	if roast {
		kr = mkRamp(0x5a3010, 0x8a5418, 0xc0882c, 0xe4b850, 0xf6dc8a)
	}
	ear := sCap(x0, y0, x1, y1, 2.6, 1.7)
	p.fill(ear, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		r := lerpf(2.6, 1.7, clamp01(u/a.l))
		l := a.rod(v, r)*0.85 + 0.12
		// Kernels in rows along it.
		ku, kv := u/0.95, v/(r*0.36)
		cu := math.Floor(ku+0.5*math.Mod(math.Floor(kv)+20, 2)) + 0.5 - 0.5*math.Mod(math.Floor(kv)+20, 2)
		du, dv := ku-cu, kv-math.Floor(kv)-0.5
		if math.Abs(du) > 0.4 || math.Abs(dv) > 0.38 {
			l -= 0.24 // between the kernels
		} else {
			l += 0.1 * toward(du, dv)
		}
		return pick(kr, l)
	})
	if roast {
		char(p, ear, 0.25, 1151)
	}
	// The silk out of its tip.
	for i := 0; i < 4; i++ {
		o := float64(i)*0.4 - 0.6
		p.fill(sLock(x1, y1, x1+1.2+o, y1-1.0, x1+2.0+o*1.4, y1+0.6+o, 0.12, 0.08), solidC(mkRamp(0x9c6a30, 0xc89a54, 0xe8c888)[i%3]))
	}
	glintAt(p, 6.0, 8.6, 0.6, 0.2)
	if roast {
		steam(p, []float64{6.0, 9.6}, 6.4, 5.8)
	}
}

// icSquash is a squash: round, deeply ribbed, orange, a woody stalk on top.
func icSquash(p *icv) {
	cx, cy, rx, ry := 8.0, 9.4, 6.4, 5.0
	p.fill(sEll(cx, cy, rx, ry), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := litOf(nx*0.85, ny*0.85) + 0.02
		// The ribs: lobes round it, dark in the grooves between.
		rib := math.Abs(math.Sin(math.Asin(max(-1, min(1, nx))) * 3.2))
		l += (rib - 0.6) * 0.3
		if rib < 0.18 {
			l -= 0.14
		}
		return pick(squashRamp, l)
	})
	p.fill(sCap(cx, cy-ry+0.8, cx+0.6, cy-ry-1.4, 0.75, 0.55), shRod(icCupR, mkAxis(cx, cy-ry, cx, cy-ry-2), 0.7, 0.1, 1, 1161))
	glintAt(p, cx-3.0, cy-2.4, 0.8, 0.3)
}

// icRoastSquash is a thick slice of squash roasted: orange flesh browned
// along its cut faces, the dark rind round its back, seeds left in the
// hollow, steaming.
func icRoastSquash(p *icv) {
	cx, cy := 8.0, 6.4
	outer := sEll(cx, cy, 7.0, 8.0).and(sBelow(cy + 0.4))
	hollow := sEll(cx, cy+0.4, 3.4, 2.6)
	slice := tilted(iBlend(0.6, outer.sub(hollow)), cx, cy+4, -0.22)
	hol := tilted(hollow, cx, cy+4, -0.22)
	rim := tilted(sEll(cx, cy, 7.0, 8.0), cx, cy+4, -0.22)
	p.fill(slice, func(f *ifrag) color.RGBA {
		if rim.d(f.x, f.y) > -0.7 {
			return pick(icCharR, 0.45+0.3*toward(f.gx, f.gy)) // the rind
		}
		l := 0.66 + (fbm2(f.x*1.2, f.y*1.2, 1171, 2)-0.5)*0.3
		if hol.d(f.x, f.y) < 0.6 {
			l -= 0.22 // browned along the hollow
		}
		if toward(f.gx, f.gy) > 0.5 && -f.d < 0.6 {
			l -= 0.2 // and along the top
		}
		return pick(squashRamp, l)
	})
	char(p, slice, 0.14, 1172)
	for i, a := range []float64{0.35, 1.45, 2.6} {
		x, y := cx+math.Cos(a)*3.0-0.3, cy+0.4+math.Sin(a)*2.4+0.9
		p.fill(iEll(x, y, 0.9, 0.48, a+math.Pi/2+0.2*float64(i)), shBulge(seedRamp, 0.4, 0.1))
	}
	glintAt(p, 3.0, 9.6, 0.5, 0.2)
	steam(p, []float64{5.4, 10.4}, 6.4, 6.0)
}
