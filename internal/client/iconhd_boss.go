package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of the six legendary spoils, one from each boss
// (legendary.go). As their old icons did (bossitems.go), each carries gold
// and a glint, which nothing ordinary in the bag does, in the tones of the
// beast it came from.

func init() {
	regIcon(sim.ItemTitanArmor, icTitanArmor)
	regIcon(sim.ItemSpinoSpear, icSpinoSpear)
	regIcon(sim.ItemKingCoat, icKingCoat)
	regIcon(sim.ItemCarchaSword, icCarchaSword)
	regIcon(sim.ItemGigaAxe, icGigaAxe)
	regIcon(sim.ItemMegaNecklace, icMegaNecklace)
}

var (
	icTitanR = mkRamp(0x0a2010, 0x0e2a16, 0x1a4622, 0x2c6832, 0x468c42, 0x70b45a, 0xa8d880) // iiTitan, longer
	icKingR  = mkRamp(0x0c080a, 0x120d10, 0x24181a, 0x3c2826, 0x5a3e34, 0x7c5a48)
	icSharkR = mkRamp(0x182634, 0x2c4458, 0x4a687c, 0x7090a4, 0xa4c0cc)
	icGoldR  = goldHD
)

// goldBand paints a band of gold from (x0, y0) to (x1, y1), w thick, lit
// round, with ridges across it.
func goldBand(p *icv, x0, y0, x1, y1, w float64) {
	a := mkAxis(x0, y0, x1, y1)
	p.fill(sCap(x0, y0, x1, y1, w, w), func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := a.rod(v, w) + 0.06
		if math.Mod(u*2.2+20, 1) < 0.16 {
			l -= 0.22
		}
		return pick(icGoldR, l)
	})
}

// icTitanArmor is armour of the great snake's scales: a coat of big green
// scales, each overlapping the one below, collared and hemmed in gold.
func icTitanArmor(p *icv) {
	body := garment(p, 13.6, false, func(f *ifrag) color.RGBA {
		return pick(icTitanR, torsoLit(f.x, 8, 7.4)-0.1)
	})
	p.on(body.grow(-0.5), func(f *ifrag) color.RGBA {
		const w, h = 2.0, 1.45
		row := math.Floor(f.y / h)
		off := 0.5 * math.Mod(row, 2)
		cx := (math.Floor(f.x/w+off) + 0.5 - off) * w
		cy := (row + 0.1) * h
		dx, dy := (f.x-cx)/(w*0.56), (f.y-cy)/(h*1.05)
		d := math.Hypot(dx, dy)
		if d > 1 {
			return color.RGBA{}
		}
		l := litOf(dx*0.6, dy*0.6) + 0.04 + 0.2*(torsoLit(f.x, 8, 7.4)-0.6)
		if d > 0.8 && dy > 0 {
			l -= 0.3 // its lower rim
		}
		if math.Abs(dx) < 0.08 && dy < 0.6 {
			l += 0.14 // the keel down each scale
		}
		return pick(icTitanR, l)
	})
	goldBand(p, 3.6, 13.2, 12.4, 13.2, 0.5)
	p.fill(sRing(8.0, 1.8, 2.1, 1.6, 0.45).and(sBelow(1.4)), shBulge(icGoldR, 0.4, 0.08))
	glintAt(p, 5.6, 5.4, 0.5, 0.22)
}

// icSpinoSpear is a spear tipped with the sail-back's tooth: a long
// straight fluted cone of ivory, bound to the haft with gold, a red tassel.
func icSpinoSpear(p *icv) {
	x0, y0, x1, y1 := 1.4, 14.6, 9.6, 6.4
	a := mkAxis(x0, y0, x1, y1)
	p.fill(sCap(x0, y0, x1, y1, 0.62, 0.56), shRod(iiWood, a, 0.6, 0, 1, 921))
	// The tooth: a long cone from the binding to its point.
	fr := frameOf(x1, y1, a.ex, a.ey)
	tooth := fr.poly(-0.6, -1.25, 0.6, -1.4, 3.0, -1.2, 5.6, -0.6, 7.2, 0, 5.6, 0.55, 3.0, 1.1, 0.6, 1.3, -0.6, 1.2)
	p.fill(iBlend(0.4, tooth), func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		hw := lerpf(1.3, 0.1, clamp01(s/7.2))
		nt := max(-1, min(1, t/hw))
		l := fr.ex*0 + litOf(fr.px*nt*0.85, fr.py*nt*0.85) + 0.02
		if math.Abs(math.Sin(nt*4.2)) > 0.9 {
			l -= 0.12 // its flutes
		}
		return pick(icToothR, l)
	})
	gx, gy := fr.at(3.4, 0.7)
	p.on(sTaper(0.05, 0.18, 0.04, gx-1.4, gy+1.4, gx, gy, gx+1.2, gy-1.2), solidC(icGlint))
	bx0, by0 := fr.at(-1.4, 0)
	bx1, by1 := fr.at(0.4, 0)
	goldBand(p, bx0, by0, bx1, by1, 0.95)
	// The tassel of red-dyed hide hanging from the binding.
	tx, ty := fr.at(-1.0, -0.9)
	for i := 0; i < 3; i++ {
		o := float64(i) * 0.5
		p.fill(sLock(tx, ty, tx+0.4+o, ty+1.6, tx+0.2+o*1.2, ty+3.6-o*0.3, 0.24, 0.12), shBulge(bloodRamp, 0.2, 0.15))
	}
}

// icKingCoat is the mammoth king's coat: long black fur, a collar of
// frost-white fur, a clasp of gold at the throat.
func icKingCoat(p *icv) {
	shFur := func(f *ifrag) color.RGBA {
		w := f.x/0.95 + fbm2(f.x*0.4, f.y*0.4, 931, 2)*1.6
		c := w - math.Floor(w)
		l := torsoLit(f.x, 8, 7.4)*0.7 + 0.22 + 0.16*math.Cos((c-0.5)*2*math.Pi)
		if math.Abs(math.Sin((f.x*5+fbm2(f.x, f.y*0.5, 932, 2)*4)*math.Pi)) > 0.9 {
			l -= 0.1
		}
		return pick(icKingR, l)
	}
	garment(p, 14.2, true, shFur)
	p.on(sCap(8.0, 3.2, 8.2, 14.0, 0.16, 0.16), solidC(icKingR[0]))
	furHem(p, icKingR, 3.3, 12.7, 13.9, 1.0, 933)
	// The frost-white collar over the shoulders.
	collar := iBlend(1.0, sEll(8.0, 3.4, 5.6, 2.4)).sub(sEll(8.0, 2.4, 2.0, 1.5)).jag(0.6, 0.7, 3.8, 934).rough(0.2, 0.4, 935)
	p.fill(collar, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.4)*0.7 + 0.32
		if math.Abs(math.Sin((f.x*2.4+f.y*0.6+fbm2(f.x, f.y, 936, 2)*2)*math.Pi)) > 0.86 {
			l -= 0.16
		}
		return pick(iiWhite, l)
	})
	p.fill(sEll(8.0, 5.6, 1.2, 1.2), shSphere(icGoldR, 8.0, 5.6, 1.2, 1.2, 0.02, true))
	glintAt(p, 4.6, 3.0, 0.5, 0.2)
}

// icCarchaSword is a blade set with the shark-toothed lizard's teeth: a
// blade of bone, its edge a row of serrated teeth, hilted in gold, its grip
// bound in rust-red hide.
func icCarchaSword(p *icv) {
	fr := frameOf(2.0, 14.0, 0.71, -0.71) // from the pommel to the point
	// The grip and the pommel.
	g0x, g0y := fr.at(0.6, 0)
	g1x, g1y := fr.at(4.0, 0)
	ga := mkAxis(g0x, g0y, g1x, g1y)
	p.fill(sCap(g0x, g0y, g1x, g1y, 0.8, 0.75), shCord(iiRust, ga, 0.8, 1.8))
	px, py := fr.at(0.2, 0)
	p.fill(sEll(px, py, 1.0, 1.0), shSphere(icGoldR, px, py, 1.0, 1.0, 0.02, false))
	// The teeth along the edge, under the blade.
	for i := 0; i < 6; i++ {
		s := 6.2 + float64(i)*1.6
		tx, ty := fr.at(s, 1.4)
		ox, oy := fr.at(s+0.9, 2.7)
		bx, by := fr.at(s+1.4, 1.4)
		tooth := sPoly(tx, ty, ox, oy, bx, by)
		p.fill(iBlend(0.2, tooth), func(f *ifrag) color.RGBA {
			l := 0.62 + 0.25*toward(f.gx, f.gy)
			if math.Mod((f.x+f.y)*3.4, 1) < 0.2 && -f.d < 0.25 {
				l -= 0.2 // serrations
			}
			return pick(icToothR, l)
		})
	}
	// The blade: a long straight lath of bone, its back thick.
	blade := iBlend(0.5, fr.poly(4.6, -1.0, 14.8, -0.9, 16.6, 0.4, 14.8, 1.5, 4.6, 1.5))
	p.fill(blade, func(f *ifrag) color.RGBA {
		_, t := fr.st(f.x, f.y)
		l := 0.6 - t*0.12
		if math.Abs(t-0.25) < 0.14 {
			l += 0.16 // the ridge down it
		}
		if -f.d < 0.35 && toward(f.gx, f.gy) > 0.2 {
			l += 0.16
		}
		return pick(boneHD, l)
	})
	hx0, hy0 := fr.at(4.2, -2.2)
	hx1, hy1 := fr.at(4.2, 2.6)
	goldBand(p, hx0, hy0, hx1, hy1, 0.6) // the guard
	gx, gy := fr.at(10.0, -0.4)
	p.on(sTaper(0.05, 0.16, 0.04, gx-1.6, gy+1.6, gx, gy, gx+1.6, gy-1.6), solidC(icGlint))
}

// icGigaAxe is an axe whose blade is the giant's fang: a great curved
// tooth lashed across the haft with gold, its root still smouldering.
func icGigaAxe(p *icv) {
	x0, y0, x1, y1 := 2.4, 15.0, 11.6, 3.2
	a := mkAxis(x0, y0, x1, y1)
	p.fill(sCap(x0, y0, x1, y1, 0.78, 0.7), shRod(iiWood, a, 0.76, 0, 1, 951))
	// The fang, curved, its root at the haft's top and its point down to
	// the left, its inner edge the blade.
	const n = 12
	var mid []float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		ang := lerpf(-0.2, 2.2, t)
		mid = append(mid, 10.9-math.Cos(ang)*5.2+1.0, 4.6+math.Sin(ang)*-4.2+3.2*t)
	}
	fang := sTaper(2.2, 1.5, 0.1, mid...)
	p.fill(fang, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.8) + 0.04
		return pick(icToothR, l)
	})
	rx, ry := mid[0], mid[1]
	p.on(sEll(rx, ry, 2.6, 2.6), func(f *ifrag) color.RGBA { // the root, charred and glowing
		d := math.Hypot(f.x-rx, f.y-ry) / 2.6
		if d < 0.55 {
			return pick(iiEmber, 0.9-d)
		}
		return mix(p.px[f.iy*p.n+f.ix], icCharR[1], 0.6*(1-d))
	})
	p.soft(sEll(rx, ry, 4.0, 4.0), func(f *ifrag) color.RGBA {
		d := math.Hypot(f.x-rx, f.y-ry) / 4.0
		return color.RGBA{0xff, 0x8f, 0x1e, uint8(90 * clamp01(1-d) * clamp01(1-d))}
	})
	bx, by := x0+a.ex*10.6, y0+a.ey*10.6
	goldBand(p, bx-a.ex*1.2, by-a.ey*1.2, bx+a.ex*1.2, by+a.ey*1.2, 1.05)
	glintAt(p, 8.4, 6.6, 0.5, 0.2)
}

// icMegaNecklace is a necklace of the Megalodon's teeth: grey-rooted
// triangles strung on a cord, gold beads between, the biggest tooth set in
// gold in the middle.
func icMegaNecklace(p *icv) {
	cx, cy := 8.0, 5.6
	p.fill(sRing(cx, cy, 6.0, 4.6, 0.26), shBulge(hideHD, 0.2, 0.1))
	tooth := func(ang, size float64, gold bool) {
		x, y := cx+math.Cos(ang)*6.0, cy+math.Sin(ang)*4.6
		ox, oy := math.Cos(ang), math.Sin(ang)*1.3
		m := math.Hypot(ox, oy)
		ox, oy = ox/m, oy/m
		px, py := -oy, ox
		tip := sPoly(x+px*size*0.55, y+py*size*0.55, x-px*size*0.55, y-py*size*0.55, x+ox*size*1.5, y+oy*size*1.5)
		p.fill(iBlend(0.25*size, tip), func(f *ifrag) color.RGBA {
			along := (f.x-x)*ox + (f.y-y)*oy
			l := 0.55 + 0.3*toward(f.gx, f.gy)*clamp01(1+f.d/0.4)
			if along < size*0.35 {
				return pick(icSharkR, l-0.05) // its grey root
			}
			if -f.d < 0.18 && math.Mod(along*5, 1) < 0.5 {
				l -= 0.18 // serrations
			}
			return pick(icToothR, l)
		})
		if gold {
			p.fill(sEll(x, y, size*0.62, size*0.4), shSphere(icGoldR, x, y, size*0.62, size*0.4, 0.02, true))
		}
	}
	for i, ang := range []float64{0.75, 1.1, 2.04, 2.39} {
		tooth(ang, 1.5, false)
		_ = i
	}
	for _, ang := range []float64{0.92, 1.24, 1.9, 2.22} { // gold beads
		x, y := cx+math.Cos(ang)*6.0, cy+math.Sin(ang)*4.6
		p.fill(sEll(x, y, 0.55, 0.55), shSphere(icGoldR, x, y, 0.55, 0.55, 0.05, false))
	}
	tooth(math.Pi/2, 2.2, true)
	glintAt(p, 7.4, 11.4, 0.5, 0.2)
}
