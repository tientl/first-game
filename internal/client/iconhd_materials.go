package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The HD icons of the raw materials: stones and ores in the COMFY52 tones of
// the boulders (rockart.go), worked stone in its tier's (heldMats), what the
// beasts give, the fibres and canes of the plants.

func init() {
	regIcon(sim.ItemPebble, icPebble)
	regIcon(sim.ItemCoal, icCoal)
	regIcon(sim.ItemIronOre, icJadeOre)
	regIcon(sim.ItemPlatinumOre, icAgateOre)
	regIcon(sim.ItemDiamond, icObsidian)
	regIcon(sim.ItemCopperBar, icFlintBlades)
	regIcon(sim.ItemIronBar, icJadeGround)
	regIcon(sim.ItemPlatinumBar, icAgateGround)
	regIcon(sim.ItemRexTooth, icRexTooth)
	regIcon(sim.ItemFur, icFur)
	regIcon(sim.ItemIvory, icIvory)
	regIcon(sim.ItemDinoHide, icDinoHide)
	regIcon(sim.ItemSilk, icSilk)
	regIcon(sim.ItemGlowSpore, icGlowSpore)
	regIcon(sim.ItemDung, icDung)
	regIcon(sim.ItemBamboo, icBamboo)
	regIcon(sim.ItemFlax, icFlax)
	regIcon(sim.ItemVine, icVine)
	regIcon(sim.ItemWorm, icWorm)
}

var (
	icSandR      = rkSandMat.tone
	icCoalR      = mkRamp(0x0f1219, 0x1a1f2e, 0x282e3b, 0x343f41, 0x414453, 0x696f80) // rkCoalMat's, and its sheen
	icJadeOreR   = mkRamp(0x2d4a40, 0x4d6659, 0x739a70, 0xa5c67c, 0x9ae5d5, 0xe5efef) // rkJadeMat's
	icAgateOreR  = mkRamp(0x3c252b, 0x5c2b34, 0x8d4830, 0xc57835, 0xe1a845, 0xeecb90) // rkAgateMat's
	icGlassR     = rkGlassMat.tone
	icIvoryR     = mkRamp(0x7d6a52, 0xb09a78, 0xd8c8a4, 0xf0e6cc, 0xfffaee)
	icToothR     = mkRamp(0x6e5a48, 0xa8946e, 0xd8caa0, 0xf2ead0, 0xfffcf0)
	icRootR      = mkRamp(0x3c252b, 0x5c3a30, 0x825646, 0xa8784e)
	icFurR       = mkRamp(0x2a1e1c, 0x46342c, 0x6a5040, 0x8e7058, 0xb49474, 0xd8bc98) // iiFur, a step longer
	icScaleR     = mkRamp(0x1a2c22, 0x2a4a30, 0x46683c, 0x6a8a48, 0x98ae60, 0xc4cc84)
	icSilkSkeinR = mkRamp(0x6c7488, 0x9aa2b4, 0xc6ccd8, 0xe6eaf0, 0xffffff)
	icDungR      = mkRamp(0x24180c, 0x3c2a14, 0x5a3e1c, 0x7a5628, 0x9c7438, 0xc09a58)
)

// icPebble is three smooth river pebbles, one of granite, one of sandstone
// and a dark one, worn round by the water.
func icPebble(p *icv) {
	peb := func(cx, cy, rx, ry, ang float64, r ramp, bias float64, seed uint32) {
		s := iEll(cx, cy, rx, ry, ang)
		p.fill(s, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
			l := litOf(nx*0.85, ny*0.85) + bias + (fbm2(f.x*1.4, f.y*1.4, seed, 2)-0.5)*0.16
			if hashf(int(f.x*5), int(f.y*5), seed) < 0.04 {
				l -= 0.14 // a grain in the stone
			}
			return pick(r, l)
		})
	}
	peb(6.2, 7.0, 4.6, 3.5, -0.35, icStoneR, 0.02, 31)
	peb(11.4, 9.6, 3.2, 2.5, 0.4, icSandR, 0.06, 32)
	peb(6.6, 12.0, 3.6, 2.4, 0.15, rkBasaltMat.tone, 0.1, 33)
	glintAt(p, 4.4, 5.4, 0.5, 0.22)
	glintAt(p, 5.2, 11.0, 0.36, 0.16)
}

// icCoal is lumps of coal: black faces with a blue sheen on the lit ones,
// a glassy fleck here and there.
func icCoal(p *icv) {
	facetFan(p, icCoalR, 7.2, 7.4, 0.8, -0.04, 0.1, 41,
		2.0, 8.2, 3.6, 4.4, 7.0, 2.4, 10.8, 3.4, 12.0, 6.6, 10.4, 10.6, 6.2, 11.6, 2.8, 10.6)
	facetFan(p, icCoalR, 12.0, 11.6, 0.8, 0.02, 0.1, 42,
		9.2, 11.6, 10.6, 9.2, 13.4, 8.8, 14.8, 11.2, 13.6, 13.8, 10.4, 14.0)
	facetFan(p, icCoalR, 4.4, 13.0, 0.75, 0.04, 0.1, 43,
		1.8, 13.0, 3.0, 11.2, 5.6, 11.0, 6.6, 13.2, 5.0, 14.8, 2.6, 14.6)
	for _, g := range [][2]float64{{5.0, 4.6}, {8.6, 3.6}, {11.6, 9.6}, {3.2, 11.8}} {
		glintAt(p, g[0], g[1], 0.34, 0.14)
	}
}

// icJadeOre is a lump of granite with green jade grown out of it: the lumps
// faceted, a vein of it running into the stone.
func icJadeOre(p *icv) {
	facetFan(p, icStoneR, 6.4, 10.4, 0.75, -0.02, 0.12, 51,
		1.4, 11.0, 2.8, 7.6, 6.2, 6.4, 10.0, 7.8, 11.4, 11.6, 9.6, 14.6, 4.0, 14.6)
	// The vein: a crooked line of green in the stone.
	p.on(sPath(0.35, 0.22, 2.6, 12.6, 4.6, 11.8, 6.2, 12.6, 8.4, 11.4), shBulge(icJadeOreR, 0.3, 0.05))
	// The lumps, back to front.
	facetFan(p, icJadeOreR, 11.6, 5.6, 0.8, 0.06, 0.06, 52,
		8.8, 6.8, 9.4, 3.4, 11.8, 1.4, 14.4, 2.8, 15.0, 6.4, 12.8, 8.8, 10.2, 9.0)
	facetFan(p, icJadeOreR, 7.4, 6.2, 0.8, 0.08, 0.06, 53,
		5.2, 7.0, 5.8, 4.6, 7.6, 3.6, 9.4, 4.8, 9.6, 7.2, 7.8, 8.6, 5.8, 8.4)
	facetFan(p, icJadeOreR, 12.6, 10.6, 0.8, 0.02, 0.06, 54,
		10.8, 10.8, 11.6, 8.8, 13.6, 8.6, 14.8, 10.4, 13.8, 12.6, 11.6, 12.8)
	glintAt(p, 10.6, 3.2, 0.5, 0.2)
	glintAt(p, 6.6, 4.8, 0.34, 0.15)
}

// icAgateOre is an agate nodule broken open: a rough rind round bands of
// orange and cream, a hollow of crystals in the heart.
func icAgateOre(p *icv) {
	body := iBlend(1.4, sEll(8.2, 8.6, 6.4, 5.4), sEll(4.6, 10.6, 3.2, 2.8), sEll(11.6, 6.6, 3.0, 2.8)).rough(0.35, 1.0, 61)
	p.fill(body, func(f *ifrag) color.RGBA {
		l := bulge(f, 2.2) - 0.08 + (fbm2(f.x*2, f.y*2, 62, 2)-0.5)*0.3
		return pick(icSandR, l)
	})
	cx, cy := 8.6, 8.2
	face := iEll(cx, cy, 5.0, 3.9, -0.25).rough(0.18, 0.6, 63)
	p.fill(face, func(f *ifrag) color.RGBA {
		dx, dy := (f.x-cx)/5.0, (f.y-cy)/3.9
		d := math.Hypot(dx, dy) + 0.12*fbm2(f.x*0.8, f.y*0.8, 64, 2)
		// The bands, close at the rim, wider inside.
		b := math.Mod(math.Sqrt(max(0, 1-d))*5.2, 1)
		l := 0.62 - (f.x-cx)*0.02 - (f.y-cy)*0.03
		switch {
		case d > 0.9:
			l = 0.25 // a dark rind line inside the crust
		case b < 0.28:
			l += 0.28
		case b < 0.45:
			l -= 0.16
		}
		if d < 0.3 { // the hollow: crystals catching the light
			l = 0.85 - 0.6*hashf(int(f.x*3), int(f.y*3), 65)
			return pick(icJadeOreR[3:], l)
		}
		return pick(icAgateOreR, l)
	})
	glintAt(p, 6.2, 6.6, 0.5, 0.18)
	glintAt(p, 8.4, 8.2, 0.26, 0.2)
}

// icObsidian is a lump of black volcanic glass: broken in big shell-like
// scars, the ripples of the blow on its face, a violet sheen where a face
// turns to the light, sharp glints on its edges; a flake struck off it.
func icObsidian(p *icv) {
	lump := []float64{1.6, 10.6, 3.4, 5.0, 7.6, 2.0, 12.4, 2.6, 14.6, 6.8, 13.4, 11.4, 9.2, 14.4, 4.0, 14.0}
	facetFan(p, icGlassR, 8.4, 7.8, 0.95, -0.18, 0, 71, lump...)
	// The scar of the last blow: a shallow dish with ripples about where it
	// landed, at the top left.
	ox, oy := 3.6, 5.4
	scar := iEll(6.6, 7.6, 4.6, 3.4, 0.5)
	p.fill(scar, func(f *ifrag) color.RGBA {
		d := math.Hypot(f.x-ox, f.y-oy)
		l := 0.42 - d*0.035 + 0.14*toward(f.gx, f.gy)*clamp01(1+f.d/0.8)
		if math.Mod(math.Sqrt(d)*2.4, 1) < 0.12 {
			l += 0.16
		}
		if -f.d < 0.35 && toward(f.gx, f.gy) < 0 {
			l += 0.3 // the far rim of the dish catches the light
		}
		return pick(icGlassR, l)
	})
	// A flake struck off it.
	facetFan(p, icGlassR, 12.6, 12.8, 0.9, -0.1, 0, 72, 10.6, 13.6, 11.6, 11.0, 14.2, 10.6, 15.0, 12.8, 13.2, 14.8)
	p.on(sCap(7.4, 2.6, 11.8, 2.9, 0.15, 0.08), solidC(icGlint))
	p.on(sCap(2.0, 9.8, 3.2, 6.0, 0.13, 0.08), solidC(icGlint))
	glintAt(p, 5.0, 6.0, 0.4, 0.16)
	glintAt(p, 12.0, 11.2, 0.26, 0.12)
}

// icFlintBlades is two long flint blades struck from a core, one on the
// other: each knapped in fine scars, a ridge down it, its edges sharp.
func icFlintBlades(p *icv) {
	m := heldMats[sim.TierCopper]
	blade := func(x0, y0, x1, y1, w float64, seed uint32) {
		fr := frameOf(x0, y0, x1-x0, y1-y0)
		l := math.Hypot(x1-x0, y1-y0)
		s := fr.poly(0, -w*0.55, l*0.25, -w, l*0.65, -w*0.9, l, 0, l*0.7, w*0.85, l*0.3, w, 0, w*0.6, -0.3, 0)
		p.fill(s, func(f *ifrag) color.RGBA {
			ss, t := fr.st(f.x, f.y)
			ridge := 0.15 * w * math.Sin(ss/l*3)
			base := 0.62
			if t < ridge {
				base = 0.36
			}
			v := knap(f.x, f.y, base, 1.0, seed)
			if math.Abs(t-ridge) < 0.16 {
				v += 0.18
			}
			if -f.d < 0.4 && toward(f.gx, f.gy) > 0 {
				v += 0.28
			}
			return pick(m.r, v)
		})
	}
	blade(4.6, 14.8, 14.6, 5.4, 1.9, 82)
	blade(1.4, 12.2, 12.4, 1.6, 2.1, 81)
	p.on(sCap(5.0, 7.0, 8.2, 4.0, 0.14, 0.08), solidC(icGlint))
}

// icJadeGround is jade ground and polished: a thick tablet, its face flat
// and glossy with clouds in the green, its edge rounded over, a glint; a
// polished chip beside it.
func icJadeGround(p *icv) {
	m := heldMats[sim.TierIron]
	ang := -0.5
	cx, cy, rx, ry := 7.6, 7.6, 6.4, 4.6
	p.fill(iEll(cx+0.35, cy+1.1, rx, ry, ang), func(f *ifrag) color.RGBA {
		return pick(m.r, 0.3-0.12*clamp01(toward(f.gx, f.gy)*-1)) // its side
	})
	p.fill(iEll(cx, cy, rx, ry, ang), func(f *ifrag) color.RGBA {
		// Flat on top, turning over at the rim: lit on the rim towards the
		// light, dark on the far one.
		t := clamp01(1 + f.d/1.1)
		l := 0.6 + (litOf(f.gx*t, f.gy*t)-0.62)*1.1
		c := fbm2(f.x*0.45, f.y*0.45, 91, 3)
		l += (c - 0.5) * 0.4 // clouds in the stone
		if c > 0.68 {
			l += 0.12
		}
		return pick(m.r, l)
	})
	p.on(sTaper(0.06, 0.24, 0.05, 3.0, 8.2, 4.4, 5.4, 7.0, 3.8), solidC(m.glint))
	glintAt(p, 8.4, 4.2, 0.26, 0.12)
	p.fill(iEll(12.8, 13.0, 1.9, 1.4, 0.4), shPolish(m.r, 12.8, 13.0, 1.9, 1.4, 0.02, m.glint, 92))
}

// icAgateGround is an agate cut in half and polished: the bands round its
// middle, a lip of the crust round it.
func icAgateGround(p *icv) {
	m := heldMats[sim.TierPlatinum]
	cx, cy := 8.0, 8.2
	rim := iEll(cx, cy, 6.9, 5.8, -0.3)
	p.fill(rim, func(f *ifrag) color.RGBA { return pick(icSandR, bulge(f, 1.0)-0.05) })
	face := iEll(cx, cy, 6.1, 5.0, -0.3)
	p.fill(face, func(f *ifrag) color.RGBA {
		dx, dy := (f.x-cx-0.4)/6.1, (f.y-cy-0.3)/5.0
		d := math.Hypot(dx, dy) + 0.14*fbm2(f.x*0.6, f.y*0.6, 93, 2)
		l := 0.55 - (f.x-cx)*0.025 - (f.y-cy)*0.03
		switch b := math.Mod(d*6.2, 1); {
		case b < 0.22:
			l += 0.3
		case b < 0.36:
			l -= 0.2
		}
		if d < 0.16 {
			l = 0.9
		}
		if -f.d < 0.4 && toward(f.gx, f.gy) > 0.3 {
			l += 0.2
		}
		if gx, gy := dx+0.45, dy+0.5; gx*gx*1.8+gy*gy < 0.025 {
			return m.glint
		}
		return pick(m.r, l)
	})
}

// icRexTooth is a tyrant's tooth: a thick curved blade of ivory, both
// edges finely serrated, the root brown and broken where it sat in the jaw.
func icRexTooth(p *icv) {
	// The middle line, from the root (bottom left) to the tip, bowed down
	// and to the right as the tooth curves back.
	const n = 14
	var mid []float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		mid = append(mid, lerpf(2.6, 13.8, t)+1.6*math.Sin(t*math.Pi), lerpf(14.0, 1.4, t)+1.6*math.Sin(t*math.Pi))
	}
	tooth := sTaper(1.8, 2.4, 0.1, mid...)
	a := mkAxis(2.6, 14.0, 13.8, 1.4)
	saw := shape{tooth.x0 - 0.5, tooth.y0 - 0.5, tooth.x1 + 0.5, tooth.y1 + 0.5, func(x, y float64) float64 {
		d := tooth.d(x, y)
		if u, _ := a.uv(x, y); u > 4.6 && d > -0.5 {
			d += 0.16 * math.Abs(math.Sin(u*5.2)) // the serrations
		}
		return d
	}}
	p.fill(saw, func(f *ifrag) color.RGBA {
		u, v := a.uv(f.x, f.y)
		l := bulge(f, 2.0) + 0.04
		if u < 2.0 { // the root, ragged where it broke
			l += (fbm2(f.x*2, f.y*2, 57, 2) - 0.5) * 0.3
			return pick(icRootR, l-0.12)
		}
		if math.Mod(u*0.55+v*0.25+20, 1) < 0.07 {
			l -= 0.1 // the faint lines of its growth
		}
		if -f.d < 0.3 && u > 4.6 {
			l -= 0.12 // the serrated edge, a darker line
		}
		return pick(icToothR, l)
	})
	p.on(sTaper(0.06, 0.24, 0.05, 5.6, 9.8, 7.8, 6.6, 10.6, 3.8), solidC(icGlint))
}

// icFur is a pelt laid fur side up: long hair flowing from the spine out
// to the flanks in locks, each lit along its middle and parted dark from
// the next, shaggy at the edge, the legs and tail out at its corners.
func icFur(p *icv) {
	pelt := iBlend(1.0,
		sPoly(4.4, 2.4, 8.0, 1.6, 11.6, 2.4, 12.4, 6.0, 13.0, 10.4, 11.4, 13.8, 8.0, 14.6, 4.6, 13.8, 3.0, 10.4, 3.6, 6.0),
		sCap(4.6, 3.6, 1.8, 2.2, 1.1, 0.7), sCap(11.4, 3.6, 14.2, 2.2, 1.1, 0.7),
		sCap(4.4, 12.0, 1.6, 14.4, 1.2, 0.7), sCap(11.6, 12.0, 14.4, 14.4, 1.2, 0.7)).
		rough(0.45, 0.35, 101)
	p.fill(pelt, func(f *ifrag) color.RGBA {
		// The hair runs from the spine (x = 8) down and out.
		fx, fy := (f.x-8)*0.55, 1.0
		m := math.Hypot(fx, fy)
		fx, fy = fx/m, fy/m
		u, v := f.x*fx+f.y*fy, -f.x*fy+f.y*fx
		w := v/0.95 + fbm2(f.x*0.35, f.y*0.35, 102, 2)*1.4
		c := w - math.Floor(w)
		k := hashf(int(math.Floor(w)), int(math.Floor(u/3)), 103)
		l := bulge(f, 1.4)*0.6 + 0.2 + (k-0.5)*0.24
		l += 0.13 * math.Cos((c-0.5)*2*math.Pi)
		if math.Abs(math.Sin((v*6+fbm2(u*0.6, v*2, 104, 2)*4)*math.Pi)) > 0.9 {
			l -= 0.1 // strands
		}
		if math.Abs(f.x-8) < 0.35 {
			l -= 0.18 // the parting down the spine
		}
		return pick(icFurR, l)
	})
}

// icIvory is a mammoth's tusk: a long sweep of ivory thick at its root,
// cut clean there, curling up to its point, the rings of its growth faint
// along it.
func icIvory(p *icv) {
	const n = 16
	x0, y0, cx, cy, x1, y1 := 2.6, 11.0, 12.4, 16.6, 13.6, 2.0
	var mid []float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		a, b, c := (1-t)*(1-t), 2*t*(1-t), t*t
		mid = append(mid, a*x0+b*cx+c*x1, a*y0+b*cy+c*y1)
	}
	tusk := sTaper(1.9, 1.5, 0.2, mid...)
	ax := mkAxis(x0, y0, x1, y1)
	p.fill(tusk, func(f *ifrag) color.RGBA {
		l := bulge(f, 1.7) + 0.02
		u, v := ax.uv(f.x, f.y)
		if math.Mod(u*0.9-v*0.5+fbm2(f.x, f.y, 111, 2)*0.4+20, 1) < 0.07 {
			l -= 0.12
		}
		return pick(icIvoryR, l)
	})
	// The cut root: rings round the dark of the hollow.
	ang := math.Atan2(mid[3]-y0, mid[2]-x0)
	p.fill(iEll(x0, y0, 0.95, 1.85, ang), func(f *ifrag) color.RGBA {
		d := math.Hypot(f.x-x0, f.y-y0) / 1.85
		l := 0.66 - d*0.2
		if math.Mod(d*3, 1) < 0.25 {
			l -= 0.15
		}
		if d < 0.28 {
			return pick(icRootR, 0.3)
		}
		return pick(icIvoryR, l)
	})
	p.on(sTaper(0.07, 0.22, 0.06, 6.2, 12.2, 9.6, 12.4, 11.8, 9.2), solidC(icGlint))
}

// icDinoHide is a dinosaur's hide laid out: rows of pebbly scales, darker
// bands across the back, the pale flesh at its cut edge.
func icDinoHide(p *icv) {
	pelt := iBlend(1.2,
		sPoly(3.6, 2.8, 8.0, 1.8, 12.4, 2.8, 13.6, 6.6, 14.6, 11.4, 12.2, 14.4, 8.0, 13.2, 3.8, 14.4, 1.4, 11.0, 2.4, 6.6),
		sCap(3.0, 3.0, 1.4, 1.8, 0.9, 0.6), sCap(13.0, 3.0, 14.8, 2.0, 0.9, 0.6)).rough(0.2, 0.6, 121)
	p.fill(pelt, func(f *ifrag) color.RGBA { return pick(icHideIn, 0.55+0.2*toward(f.gx, f.gy)) })
	sc := shPebbly(icScaleR, 1.25, 0.02, 122)
	p.fill(pelt.grow(-0.45), func(f *ifrag) color.RGBA {
		c := sc(f)
		// Bands across the back, and lighter towards the light.
		if math.Mod(f.y*0.3+fbm2(f.x*0.3, f.y*0.3, 123, 2)*0.9, 1) < 0.34 {
			c = mix(c, icScaleR[0], 0.6)
		}
		if -f.d < 0.5 && toward(f.gx, f.gy) > 0.2 {
			c = mix(c, icScaleR[5], 0.3)
		}
		return c
	})

}

// icSilk is spider silk wound on a stick: a spindle of white thread, the
// turns of it crossing in a helix, a sheen along it, a loose end drifting.
func icSilk(p *icv) {
	x0, y0, x1, y1 := 2.4, 13.8, 13.4, 2.6
	a := mkAxis(x0, y0, x1, y1)
	p.fill(sCap(x0, y0, x1, y1, 0.5, 0.45), shRod(icBarkR, a, 0.5, 0.1, 1, 131))
	L := a.l
	fr := frameOf(x0, y0, a.ex, a.ey)
	hw := func(s float64) float64 { // the spindle's half width along the stick
		t := clamp01((s - L*0.18) / (L * 0.62))
		return 3.2 * math.Pow(math.Sin(math.Pi*t), 0.8)
	}
	var top, bot []float64
	for i := 0; i <= 16; i++ {
		s := L*0.18 + L*0.62*float64(i)/16
		top = append(top, s, hw(s))
		bot = append(bot, s, -hw(s))
	}
	var poly []float64
	poly = append(poly, top...)
	for i := len(bot) - 2; i >= 0; i -= 2 {
		poly = append(poly, bot[i], bot[i+1])
	}
	spindle := fr.poly(poly...)
	p.fill(spindle, func(f *ifrag) color.RGBA {
		s, t := fr.st(f.x, f.y)
		w := max(0.3, hw(s))
		nt := max(-1, min(1, t/w))
		nx, ny := fr.px*nt*0.9, fr.py*nt*0.9
		l := litOf(nx, ny) + 0.06
		// The turns: a helix of thread, seen as lines slanting across.
		if math.Mod(s*0.95+nt*1.2+50, 1) < 0.24 {
			l -= 0.3
		}
		if math.Mod(s*0.95-nt*1.8+50.5, 1) < 0.12 {
			l -= 0.16
		}
		return pick(icSilkSkeinR, l)
	})
	p.fill(sLock(10.8, 9.6, 13.6, 11.2, 14.2, 14.8, 0.13, 0.08), solidC(icSilkSkeinR[3]))
	p.bareLast()
	gx, gy := fr.at(L*0.45, 1.6)
	p.on(sTaper(0.06, 0.2, 0.05, gx-1.4, gy+1.3, gx, gy, gx+1.4, gy-1.3), solidC(icGlint))
}

// icGlowSpore is a pinch of the spores that glow at night: bright beads of
// cyan and violet, a halo round each.
func icGlowSpore(p *icv) {
	type sp struct {
		x, y, r float64
		ramp    ramp
	}
	sps := []sp{{6.2, 9.2, 3.1, iiCyan}, {11.0, 5.6, 2.3, iiPurple}, {10.8, 11.6, 2.0, iiCyan}, {4.0, 4.2, 1.2, iiCyan}, {13.6, 13.6, 0.9, iiPurple}}
	for _, s := range sps {
		p.soft(sEll(s.x, s.y, s.r*2.1, s.r*2.1), func(f *ifrag) color.RGBA {
			d := math.Hypot(f.x-s.x, f.y-s.y) / (s.r * 2.1)
			c := s.ramp[3]
			return color.RGBA{c.R, c.G, c.B, uint8(110 * clamp01(1-d) * clamp01(1-d))}
		})
	}
	for _, s := range sps {
		p.fill(sEll(s.x, s.y, s.r, s.r), func(f *ifrag) color.RGBA {
			nx, ny := (f.x-s.x)/s.r, (f.y-s.y)/s.r
			// Lit from within: brightest in the middle, a rim of the dark.
			l := 1.0 - math.Hypot(nx+0.15, ny+0.15)*0.6
			return pick(s.ramp, l)
		})
	}
	for _, s := range sps[:3] {
		glintAt(p, s.x-s.r*0.38, s.y-s.r*0.4, s.r*0.24, s.r*0.12)
	}
}

// icDung is a pile of dung: three coils, wet and dark, bits of chewed
// grass in it, a fly over it.
func icDung(p *icv) {
	coil := func(cx, cy, rx, ry float64, seed uint32) {
		s := iEll(cx, cy, rx, ry, 0).rough(0.15, 0.8, seed)
		p.fill(s, func(f *ifrag) color.RGBA {
			nx, ny := (f.x-cx)/rx, (f.y-cy)/ry
			l := litOf(nx*0.8, ny*0.8) - 0.04 + (fbm2(f.x*1.6, f.y*1.6, seed, 2)-0.5)*0.25
			if hashf(int(f.x*3), int(f.y*3), seed+1) < 0.05 {
				return pick(icStrawR, 0.5) // a bit of grass
			}
			return pick(icDungR, l)
		})
	}
	coil(8.0, 12.2, 6.6, 2.8, 141)
	coil(7.6, 8.9, 4.9, 2.3, 142)
	coil(7.8, 6.2, 3.2, 1.8, 143)
	p.fill(sCap(8.4, 4.8, 9.6, 3.2, 0.9, 0.3), shBulge(icDungR, 0.6, 0.0))
	// The wet shine.
	p.on(sTaper(0.06, 0.18, 0.05, 3.2, 11.4, 4.6, 10.2, 6.4, 9.8), solidC(mix(icGlint, icDungR[5], 0.4)))
	p.on(sEll(5.8, 7.6, 0.4, 0.2), solidC(mix(icGlint, icDungR[5], 0.3)))
	// The fly.
	fx, fy := 13.0, 3.4
	p.fill(sEll(fx-0.6, fy-0.5, 0.8, 0.45), solidC(hex(0xbcd0dc)))
	p.fill(sEll(fx+0.5, fy-0.6, 0.7, 0.4), solidC(hex(0xd8e6ee)))
	p.fill(sEll(fx, fy, 0.65, 0.5), shSphere(icCoalR, fx, fy, 0.65, 0.5, 0.1, false))
}

// icBamboo is two lengths of bamboo cane, cut at the nodes, a leaf on a
// twig at one of them.
func icBamboo(p *icv) {
	cane := func(x0, y0, x1, y1, r float64, seed uint32) {
		a := mkAxis(x0, y0, x1, y1)
		p.fill(sCap(x0, y0, x1, y1, r, r), func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := a.rod(v, r) + 0.02
			// The nodes: a swollen ring, lit above and dark below.
			if m := math.Mod(u+50-1.0, 3.6); m < 0.5 {
				l += 0.22
			} else if m < 0.8 {
				l -= 0.25
			}
			if math.Abs(math.Sin((v/r*2.4+fbm2(u*0.3, v, seed, 2)*2)*math.Pi)) > 0.94 {
				l -= 0.1
			}
			return pick(bambooRamp, l)
		})
		// The cut end: a ring of cane round the hollow.
		ang := math.Atan2(a.ey, a.ex)
		p.fill(iEll(x0, y0, r*0.45, r*0.98, ang), func(f *ifrag) color.RGBA {
			d := math.Hypot((f.x-x0)/(r*0.45), (f.y-y0)/(r*0.98))
			_ = d
			return pick(bambooRamp, 0.82)
		})
		p.on(iEll(x0+a.ex*0.08, y0+a.ey*0.08, r*0.27, r*0.66, ang), solidC(bambooRamp[0]))
	}
	cane(2.4, 13.0, 13.2, 3.6, 1.3, 151)
	cane(5.6, 14.4, 14.6, 7.4, 1.15, 152)
	// A twig with leaves out of a node.
	p.fill(sCap(9.0, 7.0, 10.6, 4.6, 0.22, 0.16), solidC(bambooRamp[2]))
	leaf := func(x, y, l, w, ang float64) {
		cx, cy := x+math.Cos(ang)*l/2, y+math.Sin(ang)*l/2
		rib := mkAxis(x, y, x+math.Cos(ang)*l, y+math.Sin(ang)*l)
		p.fill(iEll(cx, cy, l/2, w, ang), func(f *ifrag) color.RGBA {
			_, v := rib.uv(f.x, f.y)
			lv := bulge(f, w*0.9)
			if math.Abs(v) < 0.1 {
				lv -= 0.2
			}
			if v > 0 {
				lv -= 0.12
			}
			return pick(bambooLeafRamp, lv)
		})
	}
	leaf(10.6, 4.6, 5.0, 0.85, -2.6)
	leaf(10.6, 4.6, 4.4, 0.8, -1.1)
}

// icFlax is a hank of combed flax fibre, tied in the middle, a blue flax
// flower tucked in.
func icFlax(p *icv) {
	// Fibres from end to end, flaring out at both ends from the tie.
	for i := 0; i < 11; i++ {
		o := float64(i) - 5
		px, py := 0.707*o, 0.707*o // across the hank
		x0, y0 := 2.5+px*0.4, 13.5+py*0.4
		x1, y1 := 13.5+px*0.36, 2.5+py*0.36
		cx, cy := 7.9+px*0.1, 7.9+py*0.1
		a := mkAxis(x0, y0, x1, y1)
		seed := uint32(160 + i)
		p.fill(sLock(x0, y0, cx+(hashf(i, 1, 161)-0.5)*0.6, cy, x1, y1, 0.42, 0.34), func(f *ifrag) color.RGBA {
			u, v := a.uv(f.x, f.y)
			l := 0.62 - o*0.04 + (fbm2(u*0.25, v*5, seed, 2)-0.5)*0.45
			return pick(fibreRamp, l)
		})
	}

	// The tie.
	tie := mkAxis(6.2, 6.2, 9.6, 9.6)
	p.fill(sCap(6.2, 6.2, 9.6, 9.6, 0.95, 0.95), shCord(cordRamp, tie, 0.95, 1.6))
	// The flower: five blue petals round a yellow eye.
	fx, fy := 11.8, 11.8
	for k := 0; k < 5; k++ {
		a := float64(k)*2*math.Pi/5 - 0.3
		px, py := fx+math.Cos(a)*1.25, fy+math.Sin(a)*1.25
		p.fill(iEll(px, py, 1.3, 0.95, a), func(f *ifrag) color.RGBA {
			l := 0.7 - math.Hypot(f.x-px, f.y-py)*0.15 + 0.2*toward(f.x-fx, f.y-fy)
			return pick(flaxBlue, l)
		})
	}
	p.fill(sEll(fx, fy, 0.65, 0.65), shSphere(iiGold, fx, fy, 0.65, 0.65, 0.1, false))
}

// icVine is a liana coiled in a ring, its end loose, two heart leaves on it.
func icVine(p *icv) {
	cx, cy, rx, ry := 7.6, 9.0, 5.2, 4.4
	shV := func(f *ifrag) color.RGBA {
		ang := math.Atan2((f.y-cy)/ry, (f.x-cx)/rx)
		l := bulge(f, 0.8)
		if math.Mod(ang*5+f.d*1.6+50, 1) < 0.22 {
			l -= 0.22
		}
		return pick(lianaRamp, l)
	}
	p.fill(sRing(cx, cy, rx, ry, 0.9), shV)
	p.fill(sRing(cx+0.4, cy-0.8, rx-0.6, ry-0.4, 0.8), shV)
	// The loose end, up and over.
	end := []float64{12.4, 7.4, 13.8, 4.8, 12.8, 2.0, 10.2, 1.6}
	a := mkAxis(end[0], end[1], end[6], end[7])
	p.fill(sPath(0.8, 0.5, end...), func(f *ifrag) color.RGBA {
		_, v := a.uv(f.x, f.y)
		return pick(lianaRamp, 0.55-v*0.2)
	})
	heart := func(x, y, s, ang float64) {
		lobe := iBlend(0.6, sEll(-0.9*s, 0, 1.2*s, 1.15*s), sEll(0.9*s, 0, 1.2*s, 1.15*s), sPoly(-2.0*s, 0.2*s, 2.0*s, 0.2*s, 0, 2.6*s))
		sh := tilted(lobe.moved(x, y), x, y, ang)
		p.fill(sh, func(f *ifrag) color.RGBA {
			l := bulge(f, 0.9*s)
			dx, dy := f.x-x, f.y-y
			ux, uy := -math.Sin(ang), math.Cos(ang)
			if math.Abs(dx*uy-dy*ux) < 0.12 && dx*ux+dy*uy > -0.2 {
				l -= 0.2 // the midrib
			}
			return pick(vineLeafRamp, l)
		})
	}
	heart(3.4, 5.0, 1.15, 0.6)
	heart(10.6, 12.6, 1.05, -0.4)
}

// icWorm is an earthworm in a loose S, pink and ringed, its saddle a
// darker band, glistening.
func icWorm(p *icv) {
	const n = 24
	var pts []float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		pts = append(pts, 2.2+11.6*t, 9.2+3.4*math.Sin(t*2*math.Pi+0.5))
	}
	cum := []float64{0}
	for i := 2; i < len(pts); i += 2 {
		cum = append(cum, cum[len(cum)-1]+math.Hypot(pts[i]-pts[i-2], pts[i+1]-pts[i-1]))
	}
	along := func(x, y float64) float64 {
		best, bs := 1e9, 0.0
		for i := 0; i+3 < len(pts); i += 2 {
			ax, ay, bx, by := pts[i], pts[i+1], pts[i+2], pts[i+3]
			ex, ey := bx-ax, by-ay
			k := clamp01(((x-ax)*ex + (y-ay)*ey) / (ex*ex + ey*ey))
			if d := math.Hypot(x-ax-ex*k, y-ay-ey*k); d < best {
				best, bs = d, cum[i/2]+k*math.Hypot(ex, ey)
			}
		}
		return bs
	}
	p.fill(sTaper(0.75, 1.35, 0.6, pts...), func(f *ifrag) color.RGBA {
		s := along(f.x, f.y)
		l := bulge(f, 1.1) + 0.02
		if math.Mod(s*1.5, 1) < 0.18 {
			l -= 0.14 // its rings
		}
		c := pick(iiWorm, l)
		if s > 4.6 && s < 6.8 {
			c = mix(c, iiTuber[2], 0.5) // the saddle
		}
		return c
	})
	p.on(sTaper(0.06, 0.18, 0.05, 2.8, 9.6, 3.8, 11.0, 5.0, 11.8), solidC(icGlint))
	p.on(sTaper(0.06, 0.16, 0.05, 8.6, 7.6, 9.8, 6.2, 11.2, 5.8), solidC(icGlint))
}
