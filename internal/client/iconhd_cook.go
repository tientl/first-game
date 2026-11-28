package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of the cooking gear and of what it makes (cooking.go): the
// fired-clay pot and bowl, the grill of green sticks, the stone pan, the
// hide flask and the waters, salt and flour, and the dishes. Clay, broth,
// cream, tea and juice are the tones the world's cooking is drawn in
// (cookicons.go, cookart.go); the bowls are the stew's (iconhd_pilot.go).

func init() {
	regIcon(sim.ItemClayPot, icClayPot)
	regIcon(sim.ItemClayBowl, func(p *icv) { clayBowl(p, 8.0, 8.4, 6.9, nil) })
	regIcon(sim.ItemGrill, icGrill)
	regIcon(sim.ItemStonePan, icStonePan)
	regIcon(sim.ItemFlask, func(p *icv) { icFlask(p, nil) })
	regIcon(sim.ItemDirtyWater, func(p *icv) { icFlask(p, ciPond) })
	regIcon(sim.ItemCleanWater, func(p *icv) { icFlask(p, ciWater) })
	regIcon(sim.ItemSeaWater, icSeaWater)
	regIcon(sim.ItemSalt, icSalt)
	regIcon(sim.ItemFlour, icFlour)
	regIcon(sim.ItemSaltedMeat, icSaltedMeat)
	regIcon(sim.ItemSmokedMeat, icSmokedMeat)
	regIcon(sim.ItemJerky, icJerky)
	regIcon(sim.ItemDriedFish, icDriedFish)
	regIcon(sim.ItemHotSoup, icHotSoup)
	regIcon(sim.ItemDinoStew, icDinoStew)
	regIcon(sim.ItemHerbTea, icHerbTea)
	regIcon(sim.ItemHerbFish, icHerbFish)
	regIcon(sim.ItemSearedMeat, icSearedMeat)
	regIcon(sim.ItemFriedEgg, icFriedEgg)
	regIcon(sim.ItemHerbMedicine, icHerbMedicine)
	regIcon(sim.ItemJuice, icJuice)
	regIcon(sim.ItemBread, icBread)
	regIcon(sim.ItemFruitPie, icFruitPie)
}

var (
	icFlaskR  = mkRamp(0x3c2214, 0x5e3820, 0x84542e, 0xa8743e, 0xc89a5c) // a hide flask, oiled dark
	icSmokedR = mkRamp(0x24100c, 0x3e1a10, 0x5e2a16, 0x84401e, 0xae6230)
	icDryFish = mkRamp(0x5a4028, 0x86643c, 0xb08e5a, 0xd4b67e, 0xeedaa8)
	icEggW    = mkRamp(0xb8ae98, 0xdcd4c0, 0xf2ecdc, 0xfffcf2, 0xffffff)
)

// clayBowl paints a clay bowl seen a little from above: its rim an
// ellipse rx wide at rimY, a scored band round it, its foot under it. fill
// paints what is in it, given its surface; nil leaves it empty and dark.
func clayBowl(p *icv, cx, rimY, rx float64, fill func(top shape)) {
	ry := rx * 0.84
	body := sEll(cx, rimY, rx, ry).and(sBelow(rimY))
	p.fill(body, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-rimY)/ry
		l := litOf(nx*0.95, ny*0.6) + 0.04
		if math.Abs(f.y-(rimY+ry*0.33)) < 0.28 {
			l -= 0.3 // a scored band
		}
		return pick(icBowlR, l)
	})
	p.under(sEll(cx, rimY+ry*0.9, rx*0.46, ry*0.2), shBulge(icBowlR, 0.6, -0.3)) // its foot
	p.fill(sEll(cx, rimY, rx, rx*0.36), func(f *ifrag) color.RGBA {
		return pick(icBowlR, 0.78-(f.x-cx)/rx*0.25)
	})
	top := sEll(cx, rimY+0.15, rx*0.855, rx*0.27)
	if fill == nil {
		p.fill(top, func(f *ifrag) color.RGBA {
			ny := (f.y - rimY) / (rx * 0.27)
			return pick(icBowlR, 0.12+max(0, ny)*0.3+max(0, (f.x-cx)/rx)*0.2)
		})
		return
	}
	fill(top)
}

// brothIn fills a bowl's surface with broth of ramp r, shaded by the rim.
func brothIn(p *icv, top shape, r ramp) {
	cx, cy := (top.x0+top.x1)/2, (top.y0+top.y1)/2
	rx, ry := (top.x1-top.x0)/2, (top.y1-top.y0)/2
	p.fill(top, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := 0.55 - nx*0.15 + ny*0.12
		if ny < -0.55 {
			l -= 0.3 // the rim's shadow on it
		}
		return pick(r, l)
	})
}

// icClayPot is a round-bellied cooking pot of fired clay: a flared lip, two
// lugs, a band scratched round its shoulder, soot up its belly.
func icClayPot(p *icv) {
	cx, cy, rx, ry := 8.0, 9.6, 6.4, 5.4
	rimY := cy - ry + 1.4
	for _, s := range []float64{-1, 1} { // the lugs, behind
		lx := cx + s*(rx-0.3)
		p.fill(sRing(lx, cy-2.0, 1.0, 1.15, 0.42), shBulge(icBowlR, 0.4, -0.05))
	}
	p.fill(sEll(cx, cy, rx, ry), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := litOf(nx*0.92, ny*0.85) + 0.04
		v := ny + 0.16*nx*nx
		switch {
		case math.Abs(v+0.36) < 0.05 || math.Abs(v+0.04) < 0.05:
			l -= 0.22
		case v > -0.36 && v < -0.04:
			u := math.Asin(max(-1, min(1, nx))) * 4.2
			if z := math.Abs(math.Mod(u+8, 1)-0.5)*2*0.28 - 0.34; math.Abs(v-z-0.02) < 0.06 {
				l -= 0.18 // the zigzag
			}
		}
		c := pick(icBowlR, l)
		if s := (ny-0.15)*1.5 + (vnoise(f.x*0.55, f.y*0.35, 1511)-0.5)*0.7; s > 0 {
			c = mix(c, icCharR[1+int(clamp01(l)*2)], min(0.85, s*1.2)) // soot
		}
		return c
	})
	neck := sBox(cx-3.9, rimY-0.2, cx+3.9, rimY+1.8, 0.4)
	p.fill(neck, func(f *ifrag) color.RGBA { return pick(icBowlR, 0.75-(f.x-cx)/4*0.3) })
	p.fill(sEll(cx, rimY, 4.6, 1.5), func(f *ifrag) color.RGBA { return pick(icBowlR, 0.92-(f.x-cx)/4.6*0.3) })
	p.fill(sEll(cx, rimY+0.1, 3.7, 1.0), func(f *ifrag) color.RGBA {
		return pick(icBowlR, 0.08+max(0, (f.y-rimY))*0.3) // the dark inside
	})
	glintAt(p, cx-3.6, cy-1.2, 0.6, 0.24)
}

// icGrill is a grill of green sticks seen from above and to the side: two
// rails on forked stakes, the sticks laid across them close together and
// lashed with grass where they cross the rails.
func icGrill(p *icv) {
	// The rack's frame: a along the rails, b across them, in perspective.
	ax, ay := 0.93, -0.37 // along a rail, to the far end
	bx, by := 0.52, 0.85  // across, from the far rail to the near one
	at := func(s, t float64) (float64, float64) { return 1.8 + ax*s + bx*t, 7.0 + ay*s + by*t }
	stake := func(s, t, h, bias float64) {
		x, y := at(s, t)
		a := mkAxis(x, y+h, x, y)
		p.fill(sCap(x, y+h, x, y-0.2, 0.45, 0.4), shRod(trunkRamp, a, 0.45, bias, 1, 1531))
	}
	// The far rail's stakes, then the far rail.
	stake(0, 0, 3.6, -0.14)
	stake(9.0, 0, 3.6, -0.18)
	rail := func(t, r, bias float64, seed uint32) {
		x0, y0 := at(-0.8, t)
		x1, y1 := at(9.8, t)
		p.fill(sCap(x0, y0, x1, y1, r, r), shRod(greenStick, mkAxis(x0, y0, x1, y1), r, bias, 1, seed))
	}
	rail(0, 0.55, -0.06, 1532)
	// The sticks across.
	for i := 0; i < 9; i++ {
		s := 0.4 + float64(i)*1.05
		x0, y0 := at(s, -0.8)
		x1, y1 := at(s+0.2*(hashf(i, 1, 1533)-0.5), 5.8)
		p.fill(sCap(x0, y0, x1, y1, 0.42, 0.46), shRod(greenStick, mkAxis(x0, y0, x1, y1), 0.44, 0.02*float64(i%2), 1, uint32(1534+i)))
	}
	rail(5.0, 0.62, 0.06, 1545)
	stake(0, 5.0, 3.0, 0.04)
	stake(9.0, 5.0, 3.0, -0.04)
	for _, s := range []float64{0, 9.0} { // grass lashed round where they cross
		x, y := at(s, 5.0)
		p.fill(iEll(x, y, 0.95, 0.75, -0.37), shCord(icTwineR, mkAxis(x-0.9, y, x+0.9, y-0.5), 0.9, 2.2))
	}
}

// icStonePan is a flat slab of stone pecked hollow for frying, a stick
// handle lashed to it, grease shining in the hollow.
func icStonePan(p *icv) {
	cx, cy, rx, ry := 7.0, 9.6, 6.2, 3.6
	h := mkAxis(11.2, 8.4, 15.0, 4.6)
	p.fill(sCap(11.2, 8.4, 15.0, 4.6, 0.6, 0.5), shRod(iiWood, h, 0.6, 0, 1, 1541))
	p.fill(sEll(cx, cy+1.4, rx, ry), func(f *ifrag) color.RGBA { // its edge, sooted under
		return mix(pick(stoneRamp, 0.35-(f.x-cx)/rx*0.15), icCharR[1], clamp01((f.y-cy-0.8)/2.2)*0.7)
	})
	p.fill(sEll(cx, cy, rx, ry), func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		d := nx*nx + ny*ny
		l := 0.76 - (nx+1)/2*0.3 - ny*0.12 + (fbm2(f.x*1.4, f.y*1.4, 1542, 2)-0.5)*0.14
		switch {
		case d > 0.78:
			l += 0.1 - ny*0.12 // the rim, pecked round
		case d > 0.62:
			l -= 0.14 + ny*0.12 // the step down into the hollow
		default:
			l -= 0.06
			if vnoise(f.x*0.9, f.y*1.6, 1543) > 0.64 {
				l += 0.16 // grease catching the light
			}
		}
		return pick(stoneRamp, l)
	})
	lx, ly := 11.4, 8.2
	p.fill(sBox(lx-1.0, ly-0.8, lx+1.0, ly+0.8, 0.4), shCord(icTwineR, mkAxis(lx-1, ly, lx+1, ly-0.6), 0.9, 2.0))
}

// icFlask is a flask sewn of hide: a fat bag with a seam down it, its neck
// bound with cord and stoppered with wood; with water, a drop of it beside.
func icFlask(p *icv, water ramp) {
	cx, cy := 7.0, 9.4
	bag := iBlend(1.6, sEll(cx, cy+0.8, 5.0, 5.2), sCap(cx, cy-3.0, cx+0.6, cy-6.4, 1.6, 1.0))
	p.fill(bag, func(f *ifrag) color.RGBA {
		l := bulge(f, 3.4) + (fbm2(f.x*1.1, f.y*1.1, 1551, 2)-0.5)*0.14
		if math.Abs(f.x-cx-0.25*(f.y-cy)+0.6*math.Sin(f.y*0.4)) < 0.18 && f.y > cy-3.0 {
			l -= 0.22 // the seam
		} else if math.Abs(f.x-cx-0.25*(f.y-cy)+0.6*math.Sin(f.y*0.4)-0.45) < 0.12 && f.y > cy-3.0 && math.Mod(f.y*1.6, 1) < 0.5 {
			l += 0.14 // its stitches
		}
		return pick(icFlaskR, l)
	})
	nx0, ny0 := cx+0.5, cy-5.4
	p.fill(sBox(nx0-1.25, ny0-0.7, nx0+1.25, ny0+0.7, 0.3), shCord(icTwineR, mkAxis(nx0-1.2, ny0, nx0+1.2, ny0-0.4), 0.8, 2.4))
	p.fill(sCap(nx0+0.1, ny0-1.0, nx0+0.2, ny0-2.4, 0.75, 0.65), shRod(iiWood, mkAxis(nx0, ny0, nx0, ny0-2.4), 0.75, 0.05, 1, 1552))
	p.fill(sEll(nx0+0.2, ny0-2.5, 0.65, 0.3), solidC(iiWood[3]))
	glintAt(p, cx-3.0, cy-0.6, 0.6, 0.24)
	if water == nil {
		return
	}
	// A drop of what is in it.
	dx, dy := 12.4, 11.4
	drop := iBlend(0.8, sEll(dx, dy+0.6, 2.4, 2.5), sCap(dx, dy-0.6, dx-0.4, dy-3.6, 1.6, 0.1))
	murky := water[0] == ciPond[0]
	p.fill(drop, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-dx)/2.4, (f.y-dy-0.6)/2.5
		l := 0.62 - nx*0.18 - ny*0.12
		if nx*0.45+ny*0.89 > 0.55 {
			l += 0.24 // light through it, gathered at its foot
		}
		if murky && hashf(int(f.x*4), int(f.y*4), 1553) < 0.08 {
			return pick(swampMud, 0.5) // silt in it
		}
		return pick(water, l)
	})
	glintAt(p, dx-0.9, dy-0.4, 0.55, 0.3)
}

// icSeaWater is a clay bowl of sea water, the light rippling on it, a
// crust of salt drying at its rim.
func icSeaWater(p *icv) {
	clayBowl(p, 8.0, 8.4, 6.9, func(top shape) {
		cx, cy := 8.0, 8.55
		p.fill(top, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-cx)/5.9, (f.y-cy)/1.86
			l := 0.5 - nx*0.15 + ny*0.1
			if ny < -0.5 {
				l -= 0.3
			}
			if math.Mod(f.x*0.8+math.Sin(f.y*4)*0.6+20, 1.6) < 0.18 && ny > -0.4 {
				l += 0.3 // ripples
			}
			return pick(ciWater, l)
		})
		for _, s := range [][2]float64{{2.6, 8.0}, {13.2, 8.2}, {4.2, 7.0}} {
			p.fill(sEll(s[0], s[1], 0.7, 0.35), shBulge(saltRamp, 0.3, 0.1))
		}
	})
}

// icSalt is a heap of salt on a leaf: white crystals, each catching the
// light, some fallen beside.
func icSalt(p *icv) {
	leaf(p, herbRamp, 1.6, 12.6, 14.0, 3.0, -0.12, formOval, true, 0)
	heap := iBlend(1.2, sPoly(2.8, 12.4, 8.0, 3.4, 13.2, 12.0), sEll(8.0, 11.6, 5.2, 1.8))
	p.fill(heap, func(f *ifrag) color.RGBA {
		l := 0.66 + 0.25*toward(f.gx, f.gy)*clamp01(1+f.d/1.4) - (f.x-8)*0.03
		// The crystals: little cubes, a lit face and a shaded one.
		cx, cy := math.Floor(f.x/0.7), math.Floor(f.y/0.7)
		u, v := f.x/0.7-cx, f.y/0.7-cy
		if hashf(int(cx), int(cy), 1591) < 0.5 {
			if u+v < 0.8 {
				l += 0.14
			} else {
				l -= 0.12
			}
		}
		return pick(saltRamp, l)
	})
	for _, s := range [][2]float64{{4.0, 7.8}, {7.0, 5.4}, {9.8, 7.6}} {
		glintAt(p, s[0], s[1], 0.3, 0.14)
	}
	for _, s := range [][2]float64{{13.6, 12.6}, {14.6, 11.2}, {2.0, 13.4}} {
		p.fill(sBox(s[0]-0.4, s[1]-0.4, s[0]+0.4, s[1]+0.4, 0.1), shFlat(saltRamp, 2, 0.2))
	}
}

// icFlour is a little sack of flour, tied at the neck, flour spilt in a
// heap before it.
func icFlour(p *icv) {
	sack := mkRamp(0x6a5a40, 0x968260, 0xbcaa84, 0xdcceaa, 0xf2e8cc)
	cx, cy := 6.8, 8.6
	body := iBlend(1.4, sEll(cx, cy+1.2, 4.8, 4.8), sPoly(cx-2.0, cy-3.0, cx+2.0, cy-3.0, cx+1.2, cy-4.6, cx-1.2, cy-4.6))
	p.fill(body, func(f *ifrag) color.RGBA {
		l := bulge(f, 3.2)
		if math.Mod(f.x*2.2+20, 1) < 0.14 || math.Mod(f.y*2.2+20, 1) < 0.14 {
			l -= 0.08 // the weave
		}
		if math.Abs(math.Sin((f.x-cx)*0.9+f.y*0.25)) < 0.08 && f.y > cy-2 {
			l -= 0.14 // creases
		}
		return pick(sack, l)
	})
	p.fill(iBlend(0.6, sEll(cx, cy-5.4, 2.4, 1.3), sEll(cx-1.6, cy-6.0, 1.2, 0.8), sEll(cx+1.6, cy-6.0, 1.2, 0.8)), shBulge(sack, 0.8, 0))
	p.fill(sBox(cx-1.9, cy-4.8, cx+1.9, cy-3.6, 0.4), shCord(icTwineR, mkAxis(cx-1.9, cy-4.2, cx+1.9, cy-4.4), 0.7, 2.2))
	heap := iBlend(1.0, sEll(11.6, 13.0, 3.6, 1.6), sEll(11.2, 11.8, 2.0, 1.6))
	p.fill(heap, func(f *ifrag) color.RGBA {
		return pick(icEggW, 0.6+0.3*toward(f.gx, f.gy)*clamp01(1+f.d/1.0)+(fbm2(f.x*2, f.y*2, 1601, 2)-0.5)*0.12)
	})
}

// meatSlab is a slab of meat lying turned, its cut face up: base is the
// meat's ramp, rind the fat's; face shades the cut face's pixels.
func meatSlab(p *icv, r, rind ramp, face func(f *ifrag, l float64) color.RGBA) shape {
	top := iBlend(2.0, sEll(7.6, 7.0, 6.0, 3.8), sEll(10.6, 8.8, 3.0, 2.3), sEll(4.2, 8.6, 2.6, 2.1))
	p.fill(top.moved(0, 1.8).or(top.moved(0, 0.9)), func(f *ifrag) color.RGBA {
		l := 0.48 - (f.x-8)*0.03
		if top.d(f.x, f.y) < 0.7 {
			return pick(rind, l+0.2)
		}
		return pick(r, l-0.25)
	})
	p.fill(top, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.2) + 0.02
		if -f.d < 0.6 && toward(f.gx, f.gy) > -0.1 {
			return pick(rind, l)
		}
		return face(f, l)
	})
	return top
}

// icSaltedMeat is raw meat rubbed with salt: red, its fat rind, white
// crystals crusted over it.
func icSaltedMeat(p *icv) {
	top := meatSlab(p, icRawR, icFatR, func(f *ifrag, l float64) color.RGBA {
		if math.Abs(fbm2(f.x*0.42, f.y*0.55, 91, 3)-0.5) < 0.03 {
			return pick(icFatR, l)
		}
		return pick(icRawR, l)
	})
	p.on(top.grow(1.0), func(f *ifrag) color.RGBA {
		// Crystals scattered on a jittered grid, each a lit and a shaded half.
		const g = 0.62
		cx, cy := math.Floor(f.x/g), math.Floor(f.y/g)
		if hashf(int(cx), int(cy), 1611) > 0.42 {
			return color.RGBA{}
		}
		px := (cx + 0.3 + 0.4*hashf(int(cx), int(cy), 1612)) * g
		py := (cy + 0.3 + 0.4*hashf(int(cx), int(cy), 1613)) * g
		dx, dy := f.x-px, f.y-py
		if math.Abs(dx)+math.Abs(dy) > 0.2 {
			return color.RGBA{}
		}
		if dx+dy < 0 {
			return saltRamp[3]
		}
		return saltRamp[1]
	})
}

// icSmokedMeat is a haunch smoked dark on the rack: deep red-brown, glossy
// with its fat, tied with a cord to hang by, a wisp of smoke still on it.
func icSmokedMeat(p *icv) {
	bx, by := 12.6, 12.8
	p.fill(sCap(9.0, 9.2, bx, by, 0.9, 0.82), shRod(icBoneR, mkAxis(9.0, 9.2, bx, by), 0.86, 0.0, 0, 0))
	p.fill(sEll(bx+0.5, by-0.4, 1.05, 1.05).or(sEll(bx-0.3, by+0.6, 1.05, 1.05)), shBulge(icBoneR, 0.8, 0.0))
	haunch := iBlend(2.4, iEll(6.6, 6.8, 5.4, 4.2, math.Pi/4), iEll(9.6, 9.6, 2.2, 1.7, math.Pi/4))
	p.fill(haunch, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.6) - 0.02 + (fbm2(f.x*1.2, f.y*1.2, 1621, 2)-0.5)*0.24
		return pick(icSmokedR, l)
	})
	p.on(sTaper(0.08, 0.26, 0.06, 3.2, 6.6, 4.2, 4.2, 6.6, 2.8), solidC(mix(icGlint, icSmokedR[4], 0.35)))
	// The cord it hung by.
	p.fill(sBox(9.2, 9.0, 10.6, 10.4, 0.3), shCord(icTwineR, mkAxis(9.2, 10.4, 10.6, 9.0), 0.7, 2.0))
	p.fill(sLock(9.9, 9.7, 12.4, 6.6, 13.0, 3.2, 0.18, 0.16), solidC(icTwineR[2]))
	steam(p, []float64{4.2}, 4.8, 4.6)
}

// icJerky is strips of meat dried hard: dark, wrinkled, curling at their
// ends.
func icJerky(p *icv) {
	strip := func(x0, y0, x1, y1, w float64, bias float64, seed uint32) {
		a := mkAxis(x0, y0, x1, y1)
		cx, cy := a.across()
		mx, my := (x0+x1)/2+cx*0.8, (y0+y1)/2+cy*0.8
		s := sTaper(w*0.85, w, w*0.7, x0, y0, mx, my, x1, y1).rough(0.18, 0.5, seed)
		p.fill(s, func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := 0.55 + bias + 0.2*toward(f.gx, f.gy)*clamp01(1+f.d/0.5)
			if n := fbm2(u*0.9, v*1.6, seed, 2); n > 0.62 {
				l -= 0.18 // wrinkles
			} else if n < 0.3 {
				l += 0.08
			}
			if -f.d < 0.35 && toward(f.gx, f.gy) < -0.2 {
				l -= 0.12
			}
			if hashf(int(u*3), int(v*3+20), seed) < 0.05 {
				l += 0.2 // a fleck of fat
			}
			return pick(jerkyRamp, l)
		})
	}
	strip(1.8, 6.4, 13.4, 2.6, 1.6, -0.04, 1631)
	strip(2.4, 10.6, 14.2, 7.4, 1.7, 0.02, 1632)
	strip(2.0, 14.4, 13.6, 11.6, 1.5, 0.06, 1633)
}

// icDriedFish is a fish split and dried flat: pale and leathery, its ribs
// showing, hung by its tail on a loop of cord.
func icDriedFish(p *icv) {
	fr := frameOf(2.4, 13.2, 0.72, -0.69) // head to tail
	body := iBlend(1.0, fr.poly(0, 0, 1.6, 2.6, 5.4, 3.4, 9.0, 2.4, 11.0, 0.8, 11.0, -0.8, 9.0, -2.4, 5.4, -3.4, 1.6, -2.6))
	tail := fr.poly(10.4, 0, 13.2, 2.4, 14.0, 1.8, 12.8, 0, 14.0, -1.8, 13.2, -2.4)
	p.fill(tail, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := 0.4
		if math.Mod(math.Atan2(t, s-10.4)*8+50, 1) < 0.3 {
			l -= 0.15
		}
		return pick(icDryFish, l)
	})
	p.fill(body, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := 0.62 + 0.18*toward(f.gx, f.gy)*clamp01(1+f.d/0.6) + (fbm2(f.x*1.3, f.y*1.3, 1641, 2)-0.5)*0.2
		if math.Abs(t) < 0.22 {
			l -= 0.22 // the backbone down the middle
		} else if s > 2.6 && s < 9.6 && math.Mod(s*1.1+math.Abs(t)*0.5, 1) < 0.14 {
			l -= 0.14 // its ribs
		}
		if -f.d < 0.5 {
			l -= 0.12 // the skin round its edge, darker
		}
		return pick(icDryFish, l)
	})
	ex, ey := fr.at(1.4, 1.1)
	p.on(sEll(ex, ey, 0.42, 0.42), solidC(icDryFish[0]))
	tx, ty := fr.at(12.2, 0)
	p.fill(sRing(tx+0.6, ty-1.4, 1.1, 1.3, 0.18), solidC(icTwineR[2]))
}

// icHotSoup is a bowl of thick creamy soup, steaming, herbs and bits of
// root floating in it.
func icHotSoup(p *icv) {
	clayBowl(p, 8.0, 8.4, 6.9, func(top shape) {
		brothIn(p, top, ciCream)
		for i := 0; i < 9; i++ {
			x := 3.6 + hashf(i, 0, 1661)*8.8
			y := 8.0 + (hashf(i, 1, 1661)-0.5)*2.2
			if top.d(x, y) > -0.3 {
				continue
			}
			if i%3 == 0 {
				p.fill(sEll(x, y, 0.55, 0.36), shBulge(icCarrot, 0.3, 0.05))
			} else {
				p.fill(iEll(x, y, 0.6, 0.22, hashf(i, 2, 1661)*3), solidC(icGreenR[1+i%3]))
			}
		}
	})
	steam(p, []float64{5.6, 8.4, 11.0}, 6.4, 6.4)
}

// icDinoStew is a big bowl of dinosaur stewed with roots: a dark rich gravy,
// chunks of meat, and a great bone standing out of it.
func icDinoStew(p *icv) {
	clayBowl(p, 7.6, 9.0, 7.0, func(top shape) {
		brothIn(p, top, ciDark)
		for _, c := range [][3]float64{{4.6, 9.2, 1.2}, {9.6, 9.6, 1.1}, {7.2, 8.4, 1.0}} {
			p.fill(sBox(c[0]-c[2], c[1]-c[2]*0.6, c[0]+c[2], c[1]+c[2]*0.6, 0.3).and(top.grow(0.5)), func(f *ifrag) color.RGBA {
				return pick(icRoastR, 0.6-(f.x-c[0])*0.15-(f.y-c[1])*0.25)
			})
		}
		p.fill(sEll(11.6, 8.6, 0.85, 0.5), shBulge(icCarrot, 0.5, 0.05))
		p.fill(iEll(5.6, 8.0, 1.0, 0.35, 0.3), shBulge(icGreenR, 0.4, 0.05))
	})
	// The bone, out of the stew and over the rim.
	bx0, by0, bx1, by1 := 9.2, 8.6, 13.8, 2.6
	p.fill(sCap(bx0, by0, bx1, by1, 0.9, 0.8), shRod(icBoneR, mkAxis(bx0, by0, bx1, by1), 0.85, 0.06, 0, 0))
	p.fill(sEll(bx1+0.6, by1+0.2, 1.1, 1.1).or(sEll(bx1-0.5, by1-0.6, 1.1, 1.1)), shBulge(icBoneR, 0.8, 0.06))
	steam(p, []float64{4.6, 7.4}, 7.4, 6.8)
}

// icHerbTea is a clay cup of herb tea, pale green, a leaf floating in it,
// steam curling off.
func icHerbTea(p *icv) {
	cx, rimY, rx := 7.6, 7.6, 5.2
	cup := sBox(cx-rx, rimY, cx+rx, 14.0, 2.6).and(sBelow(rimY))
	p.fill(cup, func(f *ifrag) color.RGBA {
		nx := (f.x - cx) / rx
		l := litOf(nx*0.95, 0.1) + 0.04
		if math.Abs(f.y-(rimY+2.4)) < 0.26 {
			l -= 0.28
		}
		return pick(icBowlR, l)
	})
	p.fill(sRing(cx+rx+0.6, rimY+3.2, 1.6, 2.0, 0.55), shBulge(icBowlR, 0.4, -0.06)) // its handle
	p.fill(sEll(cx, rimY, rx, 1.8), func(f *ifrag) color.RGBA { return pick(icBowlR, 0.8-(f.x-cx)/rx*0.25) })
	tea := sEll(cx, rimY+0.12, rx-0.75, 1.25)
	brothIn(p, tea, ciTea)
	leaf(p, herbRamp, cx-2.2, rimY+0.4, 3.4, 0.8, -0.25, formOval, false, 0.08)
	steam(p, []float64{5.6, 9.4}, 5.6, 5.4)
}

// icHerbFish is a fish roasted with herbs, on a broad leaf: browned, sprigs
// of green laid along it, a wedge of something sour by it.
func icHerbFish(p *icv) {
	leaf(p, bananaLeafRamp, 0.8, 12.6, 14.6, 3.2, -0.18, formOval, true, 0)
	fr := frameOf(1.8, 10.2, 0.97, -0.24)
	body := iBlend(1.2, sTaper(0.9, 2.6, 0.7, fr.path(0, 0, 3.2, 0.1, 6.8, 0.05, 10.2, 0)...))
	tail := fr.poly(9.6, 0, 12.4, 2.6, 13.2, 2.3, 12.2, 0, 13.2, -2.3, 12.4, -2.6)
	p.fill(tail, func(f *ifrag) color.RGBA { return pick(icRoastR, 0.35) })
	p.fill(body, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		l := bulge(f, 1.6) + 0.04
		if math.Mod(s*0.62+t*0.5+50, 1) < 0.14 && -f.d > 0.3 {
			l -= 0.32
		}
		return pick(icRoastR, l)
	})
	ex, ey := fr.at(1.3, 0.45)
	p.on(sEll(ex, ey, 0.45, 0.45), solidC(hex(0xf0ece0)))
	for i, s := range [][2]float64{{4.0, 0.9}, {6.4, 1.2}, {8.4, 0.6}} {
		x, y := fr.at(s[0], s[1])
		leaf(p, herbLeafRamp, x, y, 2.6, 0.6, -0.9+0.3*float64(i), formLance, false, 0.06)
	}
	p.fill(sEll(13.0, 12.4, 1.4, 1.1), shSphere(iiGold, 13.0, 12.4, 1.4, 1.1, 0.05, true))
	steam(p, []float64{5.0, 9.0}, 6.4, 5.6)
}

// icSearedMeat is a thick steak seared on hot stone: a dark crust scored
// in a cross-hatch, juices shining, a sprig of herb on it.
func icSearedMeat(p *icv) {
	top := meatSlab(p, icRoastR, icRoastR, func(f *ifrag, l float64) color.RGBA {
		u, v := f.x*0.7+f.y*0.7, f.x*0.7-f.y*0.7
		if math.Mod(u*0.55+20, 1) < 0.14 || math.Mod(v*0.55+20, 1) < 0.14 {
			return pick(icCharR, l*0.6+0.1) // the seared lines
		}
		return pick(icRoastR, l+(fbm2(f.x*1.4, f.y*1.4, 1701, 2)-0.5)*0.2)
	})
	_ = top
	p.on(sTaper(0.08, 0.24, 0.06, 3.2, 6.6, 4.6, 4.4, 7.0, 3.6), solidC(mix(icGlint, icRoastR[4], 0.3)))
	leaf(p, herbLeafRamp, 9.4, 6.4, 3.4, 0.8, -0.6, formSaw, true, 0.04)
	leaf(p, herbLeafRamp, 9.4, 6.4, 3.0, 0.75, 0.5, formSaw, true, 0)
	steam(p, []float64{5.0, 10.0}, 5.4, 5.0)
}

// icFriedEgg is an egg fried on the stone: the white spread and frilled
// brown at its edge, the yolk standing up gold and shining.
func icFriedEgg(p *icv) {
	white := iBlend(1.6, sEll(8.0, 8.6, 6.0, 4.4), sEll(4.0, 10.2, 3.2, 2.6), sEll(11.8, 6.8, 3.0, 2.8), sEll(9.6, 11.6, 3.6, 2.2)).rough(0.35, 1.0, 1711)
	p.fill(white, func(f *ifrag) color.RGBA {
		l := 0.7 + 0.2*toward(f.gx, f.gy)*clamp01(1+f.d/0.8) + (fbm2(f.x*1.2, f.y*1.2, 1712, 2)-0.5)*0.12
		c := pick(icEggW, l)
		if -f.d < 0.7 {
			c = mix(c, breadRamp[2], 0.55*(1+f.d/0.7)) // the frilled brown edge
		}
		return c
	})
	yx, yy := 7.4, 7.8
	p.fill(sEll(yx, yy, 2.8, 2.4), shSphere(ciYolk, yx, yy, 2.8, 2.4, 0.02, true))
}

// icHerbMedicine is a little bowl of crushed herbs pounded to a green
// paste, a leaf of the herb laid by it.
func icHerbMedicine(p *icv) {
	clayBowl(p, 7.6, 8.8, 6.4, func(top shape) {
		paste := mkRamp(0x1e3a16, 0x2e5a20, 0x447c2c, 0x64a03c, 0x8cc458)
		p.fill(top.grow(0.2), func(f *ifrag) color.RGBA {
			l := 0.55 + (fbm2(f.x*1.6, f.y*1.6, 1721, 2)-0.5)*0.5 - (f.x-7.6)*0.04
			if -f.d < 0.3 {
				l -= 0.2
			}
			return pick(paste, l)
		})
		p.fill(sEll(6.4, 8.0, 1.8, 0.9), func(f *ifrag) color.RGBA { return pick(paste, 0.75+0.15*toward(f.gx, f.gy)) })
	})
	leaf(p, herbLeafRamp, 9.4, 4.6, 6.0, 1.6, -0.55, formSaw, true, 0.02)
	glintAt(p, 5.4, 7.6, 0.5, 0.2)
}

// icJuice is a clay cup of berry juice, deep pink, berries heaped on its
// rim.
func icJuice(p *icv) {
	clayBowl(p, 8.0, 9.0, 6.6, func(top shape) {
		brothIn(p, top, ciJuice)
		p.on(sEll(6.2, 9.0, 1.6, 0.35), solidC(ciJuice[4]))
	})
	for _, b := range [][3]float64{{10.4, 6.8, 1.5}, {12.4, 7.6, 1.3}, {11.6, 5.0, 1.2}} {
		p.fill(sEll(b[0], b[1], b[2], b[2]), shSphere(iiBerry, b[0], b[1], b[2], b[2], -0.04, true))
	}
	leaf(p, iiLeaf, 11.4, 4.4, 3.4, 0.9, -0.6, formLance, false, 0)
}

// icBread is a round loaf baked in the earth oven: its crust golden and
// cracked open where it was scored, dusted with flour.
func icBread(p *icv) {
	cx, cy, rx, ry := 8.0, 9.0, 6.8, 5.0
	loaf := iBlend(1.0, sEll(cx, cy, rx, ry))
	p.fill(loaf, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := litOf(nx*0.85, ny*0.9) + (fbm2(f.x*1.2, f.y*1.2, 1741, 2)-0.5)*0.16
		if ny > 0.55 {
			l -= 0.14 // its underside, darker from the hearth
		}
		return pick(breadRamp, l)
	})
	// The scores: three cuts across its top, opened up pale, each with a
	// browned lip on its lit side.
	for i := 0; i < 3; i++ {
		o := float64(i-1) * 2.8
		x0, y0 := cx+o-1.4, cy-3.0+0.35*math.Abs(o)
		x1, y1 := cx+o+1.6, cy+1.4-0.2*math.Abs(o)
		a := mkAxis(x0, y0, x1, y1)
		cut := sTaper(0.15, 0.75, 0.15, x0, y0, (x0+x1)/2+0.2, (y0+y1)/2, x1, y1)
		p.fill(cut, func(f *ifrag) color.RGBA {
			_, v := a.uv(f.x, f.y)
			if v < -0.25 {
				return pick(breadRamp, 0.3)
			}
			return pick(icNutPale, 0.8-0.2*v)
		})
	}
	// Flour dusted on it.
	p.on(loaf.grow(-0.6), func(f *ifrag) color.RGBA {
		if hashf(int(f.x*4), int(f.y*4), 1742) < 0.06 && f.y < cy {
			return icEggW[3]
		}
		return color.RGBA{}
	})
}

// icFruitPie is a pie of berries: a crust crimped round its edge, strips of
// it laid across the top, the red fruit bubbling between them.
func icFruitPie(p *icv) {
	cx, cy, rx, ry := 8.0, 8.6, 7.0, 4.6
	p.fill(sEll(cx, cy+1.2, rx, ry), func(f *ifrag) color.RGBA { return pick(breadRamp, 0.4-(f.x-cx)/rx*0.15) }) // its side
	top := sEll(cx, cy, rx, ry)
	p.fill(top, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		d := math.Hypot(nx, ny)
		if d > 0.8 { // the crimped rim
			ang := math.Atan2(ny, nx)
			l := 0.62 + 0.2*math.Cos(ang*14) - ny*0.1 - nx*0.08
			return pick(breadRamp, l)
		}
		// The lattice over the filling.
		u, v := (f.x-cx)*0.7+(f.y-cy)*0.9, (f.x-cx)*0.7-(f.y-cy)*0.9
		su, sv := math.Mod(u*0.42+20.25, 1), math.Mod(v*0.42+20.25, 1)
		if su < 0.32 || sv < 0.32 {
			return pick(breadRamp, 0.72-ny*0.1-nx*0.1)
		}
		l := 0.5 + (fbm2(f.x*1.4, f.y*1.4, 1751, 2)-0.5)*0.4
		return pick(iiBerry, l)
	})
	glintAt(p, 4.6, 6.8, 0.5, 0.2)
	steam(p, []float64{6.0, 10.0}, 5.4, 5.0)
}
