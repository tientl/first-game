package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of what the cave beasts leave (sim/cavefight.go): the spider
// queens' venom gland, a plate of chitin, bat guano, ancient amber, the
// centipede's venom claw, the blind cave salamander, a fossil bone.

func init() {
	regIcon(sim.ItemVenomGland, icVenomGland)
	regIcon(sim.ItemChitin, icChitin)
	regIcon(sim.ItemGuano, icGuano)
	regIcon(sim.ItemAmber, icAmber)
	regIcon(sim.ItemCentipedeClaw, icCentipedeClaw)
	regIcon(sim.ItemOlm, icOlm)
	regIcon(sim.ItemFossilBone, icFossilBone)
}

var (
	icVenomSacR = mkRamp(0x3a1640, 0x5e2462, 0x8a3c86, 0xb664a8, 0xd894c4, 0xf2c4e0) // a spider's gland, violet
	icVenomR    = mkRamp(0x2c5a10, 0x4c8c18, 0x84c824, 0xc8f05c, 0xf0ffb0)           // its venom, acid green
	icFangR     = mkRamp(0x0e0a0c, 0x1c1216, 0x2c1c22, 0x44282e, 0x7a4c50)           // a spider's fang
	icChitinR   = mkRamp(0x0e0a0a, 0x1a1412, 0x281e18, 0x3a2b1e, 0x503c28, 0x6a5232, 0x8a6e42)
	icChitinSh  = mkRamp(0x2a6a6a, 0x4aa49a, 0x8adcc8, 0xd8fff0) // its sheen, oil on water
	icGuanoR    = mkRamp(0x161210, 0x241c16, 0x36291f, 0x4a392a, 0x604c37, 0x7a6448)
	icGuanoWR   = mkRamp(0x8a867c, 0xb4b0a4, 0xd8d4c8, 0xf2efe6)
	icAmberR    = mkRamp(0x4a1806, 0x7a3008, 0xa84e0e, 0xd07a18, 0xeca634, 0xfcd270, 0xfff0b8)
	icCentiR    = mkRamp(0x3a200c, 0x6a3e16, 0x9a6224, 0xc48a38, 0xe4b45a, 0xf8dc94) // a house centipede's yellow
	icCentiDkR  = mkRamp(0x0e0a12, 0x1e1626, 0x30243c, 0x463656)                     // its dark bands
	icOlmR      = mkRamp(0x7a4a50, 0xa8706e, 0xd09a90, 0xecc4b6, 0xfce4da, 0xfff6f0)
	icGillR     = mkRamp(0x6a1020, 0xa82232, 0xd8444a, 0xf47c70)
	icFossilR   = mkRamp(0x2a1c16, 0x46301f, 0x664628, 0x8a6236, 0xae844c, 0xcaa66c, 0xe2c896)
	icMatrixR   = mkRamp(0x2e2a26, 0x46403a, 0x645c52, 0x847a6c, 0xa49a88, 0xc2b8a4)
)

// icVenomGland is a spider queen's venom gland: a violet sac, veined, the
// green venom showing through its lower half, on its duct a black fang
// with a drop of venom at its tip.
func icVenomGland(p *icv) {
	// The sac and its duct.
	cx, cy, rx, ry := 5.8, 9.6, 4.7, 4.3
	sac := iBlend(1.6, sEll(cx, cy, rx, ry), sCap(8.0, 7.4, 10.0, 5.4, 2.0, 1.1))
	p.fill(sac, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
		l := litOf(nx*0.85, ny*0.85) + 0.04
		if math.Abs(fbm2(f.x*0.55, f.y*0.55, 981, 2)-0.5) < 0.028 {
			l -= 0.26 // veins
		}
		c := pick(icVenomSacR, l)
		// The venom inside, seen through the skin of the lower half: brightest
		// low on the right, where the light comes through; a pale line where
		// it meets the air.
		lev := cy + 0.3 + 0.35*math.Sin(f.x*1.3)
		if w := f.y - lev; w > 0 && -f.d > 0.45 {
			v := 0.4 + 0.32*clamp01((nx+ny)*0.8) - 0.18*clamp01(1+f.d/1.2)
			if w < 0.3 {
				v += 0.3
			}
			c = mix(c, pick(icVenomR, v), 0.7)
		}
		return c
	})
	glintAt(p, 3.6, 7.0, 0.85, 0.32)
	p.on(sEll(5.2, 6.0, 0.32, 0.22), solidC(icGlint))
	// The fang on the duct's end: black, glossy, hooked down.
	fang := sTaper(1.5, 0.9, 0.05, 9.6, 5.0, 11.6, 3.6, 13.6, 3.7, 14.7, 5.4, 14.4, 7.8, 13.5, 10.0)
	p.fill(fang, func(f *ifrag) color.RGBA { return pick(icFangR, bulge(f, 0.7)+0.02) })
	p.on(sPath(0.17, 0.07, 11.4, 3.3, 13.2, 3.2, 14.3, 4.4, 14.5, 6.2), solidC(icFangR[4])) // the gloss along its back
	p.fill(sEll(9.9, 5.0, 1.0, 1.25), shBulge(icVenomSacR, 0.6, -0.08))                    // the socket it sits in
	// The drop at the fang's tip.
	drop := iBlend(0.5, sEll(13.3, 13.4, 1.3, 1.4), sCap(13.4, 12.4, 13.4, 10.6, 0.56, 0.1))
	p.fill(drop, shSphere(icVenomR, 13.3, 13.4, 1.3, 1.4, 0.08, true))
}

// icChitin is two plates of a giant millipede's armour, one lapped over
// the other: broad arches of glossy dark chitin, a raised rim along the
// front of each, a keel down the middle, rows of tubercles, the flanks
// drawn out into points.
func icChitin(p *icv) {
	chitinPlate(p, 7.6, 1.0, 6.6, 5.6, 6.0, 3.4, -0.12, 991) // the one behind: its front shows
	chitinPlate(p, 8.2, 4.2, 7.2, 6.0, 6.4, 3.6, 0.0, 993)
}

// chitinPlate paints one arched plate: its front edge an arch h high and
// w wide each side of cx, its top at top; thick deep in the middle, its
// back edge an arch ih high.
func chitinPlate(p *icv, cx, top, w, h, thick, ih, bias float64, seed uint32) {
	ocy := top + h
	icy, irx := top+thick+ih, w*1.06
	plate := sEll(cx, ocy, w, h).sub(sEll(cx, icy, irx, ih))
	edges := func(x float64) (yo, yi float64) { // the front and back edges above x
		dx := (x - cx) / w
		yo = ocy - h*math.Sqrt(math.Max(0, 1-dx*dx))
		di := (x - cx) / irx
		yi = icy - ih*math.Sqrt(math.Max(0, 1-di*di))
		return
	}
	p.fill(plate, func(f *ifrag) color.RGBA {
		yo, yi := edges(f.x)
		t := clamp01((yi - f.y) / math.Max(0.3, yi-yo)) // 0 at the back, 1 at the front
		ax := (f.x - cx) / w
		// Rounded across (a curved shell) and along the arch, lit on the
		// upper left.
		l := litOf(ax*0.9, (0.45-t)*1.5) - 0.08 + bias
		switch {
		case t > 0.88:
			l += 0.14 // the rim, raised
		case t > 0.8:
			l -= 0.26 // the groove behind it
		default:
			if math.Abs(f.x-cx) < 0.34 {
				l += 0.08 * -math.Copysign(1, f.x-cx) // the keel, lit on its left
			}
			for _, row := range [...]float64{0.28, 0.56} { // tubercles
				for k := -4; k <= 4; k++ {
					if k == 0 {
						continue
					}
					tx := cx + float64(k)*1.45
					to, ti := edges(tx)
					ty := ti + (to-ti)*row
					if dx, dy := (f.x-tx)/0.48, (f.y-ty)/0.42; dx*dx+dy*dy < 1 {
						l += 0.3 * toward(dx, dy)
					}
				}
			}
		}
		l += (fbm2(f.x*1.3, f.y*1.3, seed, 2) - 0.5) * 0.08
		c := pick(icChitinR, l)
		if t > 0.6 && t < 0.74 && ax > -0.82 && ax < -0.12 {
			// The gloss along the arch, shot with colour like oil on water.
			c = mix(c, pick(icChitinSh, 0.15+(ax+0.82)*1.2), 0.75)
		}
		return c
	})
}

// icGuano is a heap of bat droppings: soft dark lumps, crumbly, splashed
// with chalky white, a few loose pellets at its foot.
func icGuano(p *icv) {
	pellet := func(x, y, a float64, i int) {
		p.fill(iEll(x, y, 0.75, 0.4, a), func(f *ifrag) color.RGBA {
			nx, ny := (f.x-x)/0.75, (f.y-y)/0.75
			return pick(icGuanoR, litOf(nx*0.8, ny*1.6)+0.04*float64(i%2))
		})
	}
	pellet(1.8, 13.2, 0.5, 0)
	pellet(14.0, 12.6, -0.6, 1)
	pellet(12.8, 14.4, 0.2, 0)
	lump := func(s shape, bias float64, seed uint32) {
		p.fill(s.rough(0.22, 0.7, seed), func(f *ifrag) color.RGBA {
			l := bulge(f, 1.4)*0.95 - 0.06 + bias + (fbm2(f.x*2.6, f.y*2.6, seed, 2)-0.5)*0.14
			if hashf(int(f.x*7), int(f.y*7), seed) < 0.06 {
				l -= 0.2 // crumbs
			}
			return pick(icGuanoR, l)
		})
	}
	lump(iBlend(1.2, sEll(8.0, 12.4, 6.0, 2.3), sEll(4.4, 12.6, 2.6, 1.9)), -0.04, 1001)
	lump(sEll(6.0, 9.8, 2.9, 2.1), 0, 1002)
	lump(sEll(10.2, 10.0, 2.7, 2.0), 0, 1003)
	lump(sEll(8.2, 7.4, 2.3, 1.9), 0.04, 1004)
	pellet(4.0, 14.4, -0.3, 1)
	pellet(10.4, 14.4, 0.4, 0)
	// The white splashed over the top, and running down.
	splash := iBlend(0.5, sEll(8.0, 6.6, 1.5, 0.75), sEll(9.6, 7.6, 0.8, 0.5), sEll(5.4, 8.6, 1.0, 0.5), sEll(10.8, 8.8, 0.9, 0.45)).rough(0.25, 0.55, 1007)
	drips := sPath(0.34, 0.16, 7.0, 7.0, 6.7, 8.4).or(sPath(0.26, 0.12, 11.0, 9.0, 11.3, 10.2))
	p.on(splash.or(drips), func(f *ifrag) color.RGBA {
		return pick(icGuanoWR, 0.82-0.12*(f.y-6.6)+(fbm2(f.x*3, f.y*3, 1009, 1)-0.5)*0.3)
	})
}

// icAmber is a nugget of ancient amber: deep gold, lit through from
// within, a gnat caught in it long ago.
func icAmber(p *icv) {
	cx, cy := 8.0, 8.4
	lump := iBlend(1.8, sEll(7.4, 9.0, 5.6, 5.0), sEll(10.6, 6.0, 3.4, 3.0), sEll(5.2, 11.6, 3.0, 2.6)).rough(0.3, 1.8, 1023)
	p.fill(lump, func(f *ifrag) color.RGBA {
		nx, ny := (f.x-cx)/6.0, (f.y-cy)/6.0
		// Light goes through amber: dark at its rim, brightest inside, low
		// on the side away from the light.
		in := clamp01(-f.d / 2.2)
		l := 0.18 + 0.42*in + 0.22*clamp01(nx*0.6+ny*0.8+0.2)*in
		l += (fbm2(f.x*0.6, f.y*0.6, 1021, 3) - 0.5) * 0.14
		if -f.d < 0.5 && toward(f.gx, f.gy) > 0.3 {
			l += 0.3 // the lit lip
		}
		return pick(icAmberR, l)
	})
	// Flecks and a crack in it.
	for i := 0; i < 6; i++ {
		x, y := 4.6+7*hashf(i, 0, 1031), 4.6+8*hashf(i, 1, 1031)
		p.on(sEll(x, y, 0.16, 0.16).and(lump.grow(-1.0)), solidC(icAmberR[1]))
	}
	p.on(sPath(0.09, 0.05, 10.6, 3.4, 11.4, 5.0, 12.6, 5.8).and(lump.grow(-0.4)), solidC(icAmberR[5]))
	// The gnat, dark through the gold.
	gnat := func(c color.RGBA) ishader { return solidC(mix(c, icAmberR[2], 0.25)) }
	body := icAmberR[0]
	wing := mix(icAmberR[6], icAmberR[4], 0.4)
	p.on(iEll(9.2, 7.2, 1.9, 0.7, -0.5), gnat(wing)) // its wings, spread back
	p.on(iEll(9.6, 8.8, 1.7, 0.6, 0.3), gnat(wing))
	for _, l := range [][4]float64{ // its legs
		{7.4, 9.0, 6.0, 10.6}, {7.8, 9.2, 7.4, 11.4}, {8.2, 9.2, 9.0, 11.2},
		{7.4, 8.4, 5.6, 8.0}, {6.6, 7.8, 5.4, 6.4},
	} {
		p.on(sPath(0.11, 0.08, l[0], l[1], l[2], l[3]), gnat(body))
	}
	p.on(iEll(9.2, 8.6, 1.5, 0.55, 0.15), gnat(body))   // abdomen
	p.on(sEll(7.4, 8.4, 0.62, 0.55), gnat(body))         // thorax
	p.on(sEll(6.4, 8.0, 0.42, 0.42), gnat(body))         // head
	p.on(sPath(0.07, 0.05, 6.0, 7.8, 4.8, 6.8), gnat(body)) // its feelers
	// The gloss: a long glint on the upper left, a small one by it.
	glintAt(p, 5.4, 4.6, 1.1, 0.32)
	p.on(sEll(4.2, 6.6, 0.3, 0.3), solidC(icGlint))
}

// icCentipedeClaw is a venom claw of the giant house centipede: a jointed
// sickle of glossy yellow chitin banded dark, blackening to its point, a
// drop of venom at the tip.
func icCentipedeClaw(p *icv) {
	// The joint it grows from: a stout yellow segment, banded dark.
	a := mkAxis(2.5, 13.8, 4.6, 11.0)
	p.fill(sCap(2.5, 13.8, 4.6, 11.0, 1.8, 1.75), func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, 1.9) + 0.06
		if u > a.l-0.8 || math.Abs(u-a.l*0.3) < 0.36 {
			return pick(icCentiDkR, l+0.12) // the bands
		}
		return pick(icCentiR, l)
	})
	// The claw: hooked from the joint up and over to its point.
	p0x, p0y, c0x, c0y, p1x, p1y := 4.6, 11.0, 3.4, 1.8, 13.8, 4.6
	claw := sLock(p0x, p0y, c0x, c0y, p1x, p1y, 2.0, 0.06)
	p.fill(claw, func(f *ifrag) color.RGBA {
		t := bezT(f.x, f.y, p0x, p0y, c0x, c0y, p1x, p1y)
		l := bulge(f, 1.3) + 0.06
		c := pick(icCentiR, l)
		if t > 0.42 { // blackening to the point
			c = mix(c, pick(icCentiDkR, l+0.1), clamp01((t-0.42)*3))
		}
		if t < 0.06 {
			c = mix(c, pick(icCentiDkR, l+0.1), 0.8) // the joint's seam
		}
		return c
	})
	// Notches along its inner edge, and the gloss along its back.
	for _, t := range [...]float64{0.2, 0.3, 0.4} {
		x, y := bezAt(t, p0x, p0y, c0x, c0y, p1x, p1y)
		p.on(sEll(x+0.9, y+0.9, 0.38, 0.38), func(f *ifrag) color.RGBA { return mix(p.px[f.iy*p.n+f.ix], colInk, 0.45) })
	}
	p.on(sPath(0.2, 0.1, 3.0, 9.4, 3.2, 5.8, 5.0, 3.6, 8.2, 2.8, 10.6, 3.2), solidC(icCentiR[5]))
	// The venom drop at its point.
	drop := iBlend(0.5, sEll(13.6, 8.8, 1.2, 1.3), sCap(13.7, 7.8, 13.8, 5.0, 0.52, 0.08))
	p.fill(drop, shSphere(icVenomR, 13.6, 8.8, 1.2, 1.3, 0.08, true))
}

// bezAt is the point at t on the quadratic curve from p0 by c to p1.
func bezAt(t, p0x, p0y, cx, cy, p1x, p1y float64) (float64, float64) {
	a, b, c := (1-t)*(1-t), 2*t*(1-t), t*t
	return a*p0x + b*cx + c*p1x, a*p0y + b*cy + c*p1y
}

// bezT is how far along the quadratic curve from p0 by c to p1 the point
// nearest (x, y) is, 0 to 1.
func bezT(x, y, p0x, p0y, cx, cy, p1x, p1y float64) float64 {
	best, bt := math.Inf(1), 0.0
	for i := 0; i <= 32; i++ {
		t := float64(i) / 32
		bx, by := bezAt(t, p0x, p0y, cx, cy, p1x, p1y)
		if d := (bx-x)*(bx-x) + (by-y)*(by-y); d < best {
			best, bt = d, t
		}
	}
	return bt
}

// icOlm is a blind cave salamander, seen from above: long, pale and pink,
// its tiny legs splayed, red feathery gills at its neck, no eyes.
func icOlm(p *icv) {
	// The spine, tail (low left) to snout (top right).
	spine := []float64{1.0, 14.2, 3.2, 13.4, 5.4, 11.6, 7.6, 10.2, 9.8, 8.4, 11.6, 6.4, 13.4, 4.0}
	// Legs first: under the body.
	leg := func(x, y, dx, dy float64) {
		ex, ey := x+dx, y+dy
		p.fill(sCap(x, y, ex, ey, 0.5, 0.34), shBulge(icOlmR, 0.35, -0.02))
		for k := -1; k <= 1; k++ { // three little toes
			a := math.Atan2(dy, dx) + float64(k)*0.6
			p.add(sCap(ex, ey, ex+math.Cos(a)*0.6, ey+math.Sin(a)*0.6, 0.16, 0.1), shBulge(icOlmR, 0.2, -0.02))
		}
	}
	leg(10.6, 7.6, 1.6, 1.4)  // front right
	leg(9.6, 7.4, -1.6, -1.2) // front left
	leg(5.6, 11.6, 1.4, 1.5)
	leg(4.6, 11.2, -1.6, -1.0)
	// The gills: three plumes each side of the neck.
	gill := func(x, y, ang float64) {
		for k := 0; k < 3; k++ {
			a := ang + (float64(k)-1)*0.42
			ex, ey := x+math.Cos(a)*2.0, y+math.Sin(a)*2.0
			s := sTaper(0.36, 0.5, 0.16, x, y, (x+ex)/2, (y+ey)/2, ex, ey)
			p.fill(s, func(f *ifrag) color.RGBA {
				l := bulge(f, 0.35) + 0.04
				if math.Mod((f.x+f.y)*3.4+20, 1) < 0.3 {
					l -= 0.18 // feathery
				}
				return pick(icGillR, l)
			})
		}
	}
	gill(12.6, 6.6, 0.55)
	gill(11.4, 5.2, -2.3)
	// The body.
	body := sTaper(0.25, 1.5, 1.15, spine...)
	head := iEll(13.4, 3.8, 1.9, 1.15, -0.85)
	olm := iBlend(1.0, body, head)
	p.fill(olm, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.0) + 0.06
		c := pick(icOlmR, l)
		if fbm2(f.x*1.2, f.y*1.2, 1041, 2) > 0.62 {
			c = mix(c, icOlmR[1], 0.25) // blood showing through
		}
		return c
	})
	// A faint spot under the skin where its eyes would be, the nostrils, a
	// glint of wet.
	p.on(sEll(13.1, 3.5, 0.24, 0.2), solidC(mix(icOlmR[3], icOlmR[1], 0.4)))
	p.on(sEll(14.4, 2.6, 0.12, 0.12), solidC(icOlmR[1]))
	p.on(sPath(0.13, 0.08, 6.0, 10.4, 8.0, 9.0, 9.6, 7.6), solidC(icOlmR[5]))
}

// icFossilBone is a heavy old bone turned to stone: a thigh bone, brown
// and mottled, cracked across, its lower end still in a slab of layered
// rock.
func icFossilBone(p *icv) {
	// The slab, under the lower end: layered.
	slab := sPoly(0.6, 12.2, 2.0, 9.8, 5.2, 10.2, 8.0, 12.0, 7.6, 14.6, 4.0, 15.2, 1.0, 14.8).rough(0.22, 0.9, 1061)
	p.fill(slab, func(f *ifrag) color.RGBA {
		l := 0.5 + (fbm2(f.x*1.4, f.y*1.4, 1062, 2)-0.5)*0.2
		if math.Mod(f.y*1.25-f.x*0.18+20, 1) < 0.2 {
			l -= 0.16 // its layers
		}
		if -f.d < 0.45 {
			if w := toward(f.gx, f.gy); w > 0.3 {
				l += 0.24
			} else if w < -0.3 {
				l -= 0.16
			}
		}
		return pick(icMatrixR, l)
	})
	ax, ay, bx, by := 4.4, 11.6, 11.4, 4.6
	a := mkAxis(ax, ay, bx, by)
	cx, cy := a.across()
	// The upper end: the ball of the hip joint to one side, a knob to the
	// other; the lower end two knuckles.
	top := iBlend(1.0, sEll(bx+a.ex*0.4-cx*1.3, by+a.ey*0.4-cy*1.3, 1.9, 1.9), sEll(bx+cx*1.2, by+cy*1.2, 1.4, 1.4))
	bot := iBlend(0.9, sEll(ax-a.ex*0.5+cx*1.2, ay-a.ey*0.5+cy*1.2, 1.75, 1.75), sEll(ax-a.ex*0.5-cx*1.2, ay-a.ey*0.5-cy*1.2, 1.75, 1.75))
	bone := iBlend(1.3, sCap(ax, ay, bx, by, 1.25, 1.15), top, bot)
	p.fill(bone, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.5)
		l += (fbm2(f.x*0.9, f.y*0.9, 1051, 3) - 0.5) * 0.34 // mottled, as stone
		if hashf(int(f.x*6), int(f.y*6), 1053) < 0.05 {
			l -= 0.18 // pits
		}
		return pick(icFossilR, l)
	})
	// Cracks across the shaft, as old bone breaks.
	for _, c := range [][6]float64{{6.2, 8.8, 6.8, 9.6, 7.6, 9.8}, {8.8, 5.8, 9.2, 6.8, 10.0, 7.2}} {
		p.on(sPath(0.11, 0.07, c[:]...), solidC(icFossilR[0]))
		p.on(sPath(0.07, 0.04, c[0]-0.25, c[1]-0.2, c[2]-0.25, c[3]-0.2), solidC(icFossilR[5]))
	}
	// Rock still crusted over the lower knuckles.
	crust := sEll(2.6, 13.2, 2.4, 1.6).rough(0.3, 0.7, 1065).and(bone.grow(0.25))
	p.fill(crust, func(f *ifrag) color.RGBA {
		return pick(icMatrixR, 0.55+0.25*toward(f.gx, f.gy)*clamp01(1+f.d/0.4)+(fbm2(f.x*2, f.y*2, 1066, 2)-0.5)*0.2)
	})
	// Grit, and a pale chip knocked off the ball.
	for i := 0; i < 8; i++ {
		x, y := 1.4+6*hashf(i, 0, 1071), 11.4+3.4*hashf(i, 1, 1071)
		p.on(sEll(x, y, 0.15, 0.15).and(slab.grow(-0.3)), solidC(icMatrixR[1+i%3]))
	}
	p.on(sEll(bx-cx*1.6-0.4, by-cy*1.6-0.6, 0.5, 0.35), solidC(icFossilR[6]))
}
