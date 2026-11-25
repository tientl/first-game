package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The bag icons of the new plants' and the toads' items. Each is drawn with
// the brushes of the plants themselves (floraart.go) on a canvas three
// times as fine, then brought down to the 16 pixels of an icon and given
// the dark rim the other icons have, so what is in the bag looks like what
// was picked.

// iconFrom makes a 16x16 icon of what f holds: a 48x48 drawing.
func iconFrom(f *flo) *canvas {
	const k = HDDensity
	c := newCanvas(16, 16)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			var r, g, b, n float64
			for j := 0; j < k; j++ {
				for i := 0; i < k; i++ {
					p := f.c.px[(y*k+j)*f.c.w+x*k+i]
					if p.A >= 128 {
						r, g, b, n = r+float64(p.R), g+float64(p.G), b+float64(p.B), n+1
					}
				}
			}
			if n >= 4 { // mostly covered
				c.px[y*16+x] = color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255}
			}
		}
	}
	c.outline(0.8)
	return c
}

var (
	fibreRamp  = mkRamp(0x7c6c40, 0xa89868, 0xccbe8c, 0xe6dcb0, 0xf8f2d8)
	cordRamp   = mkRamp(0x5c4220, 0x886430, 0xb08a48, 0xd0ae6c, 0xe8d098)
	seedRamp   = mkRamp(0x9c8c5c, 0xc8b880, 0xe4d8a8, 0xf6eecc, 0xfffcec)
	charRamp   = mkRamp(0x4a2410, 0x7c4218, 0xb06a24, 0xd89638, 0xf2c466, 0xfce6a0)
	puffedRamp = mkRamp(0x9c7430, 0xc8a04c, 0xe6c878, 0xf6e4a8, 0xfffad8)
)

// ring is a rope or vine coiled in a ring round art point (x, y).
func (f *flo) ring(x, y, rx, ry, r float64, rp ramp, twist float64) {
	var pts []tpt
	for i := 0; i <= 40; i++ {
		a := 2 * math.Pi * float64(i) / 40
		pts = append(pts, tpt{x + math.Cos(a)*rx, y + math.Sin(a)*ry, r})
	}
	tb := newCTube(pts)
	total := tb.len[len(tb.len)-1]
	f.q.fill(tb.shape(), tb.skin(func(px, py int, s, v, l float64) color.RGBA {
		u := s*total/twist + v*0.5
		if u-math.Floor(u) < 0.36 {
			return toneOf(rp, l*0.8-0.06)
		}
		return toneOf(rp, l*0.8+0.2)
	}))
}

// lotusPod is the seed head of a lotus at art point (x, y), s big.
func (f *flo) lotusPod(x, y, s float64) {
	cone := aPoly(x-2.9*s, y-0.6*s, x+2.9*s, y-0.6*s, x+1*s, y+3.4*s, x-1*s, y+3.4*s)
	f.lump(cone.grow(0.6), podRamp, 1.6*s, -0.06)
	top := aEll(x, y-0.6*s, 3.1*s, 1.3*s)
	f.q.fill(top, shd(podRamp, top, 1, 0.2))
	for i, p := range [][2]float64{{-1.8, 0}, {-0.6, -0.5}, {0.7, -0.45}, {1.9, 0.05}, {-0.9, 0.45}, {0.5, 0.5}} {
		f.q.flat(aEll(x+p[0]*s, y+(p[1]-0.6)*s, 0.46*s, 0.34*s), plain(podRamp[0]))
		if i%2 == 0 {
			f.q.flat(aEll(x+p[0]*s, y+(p[1]-0.64)*s, 0.26*s, 0.2*s), plain(hex(0xe8e2b0)))
		}
	}
}

// cornEar is an ear of corn from art point (x0, y0), its foot in the husk,
// to its tip (x1, y1).
func (f *flo) cornEar(x0, y0, x1, y1, r float64, kernels ramp) {
	ear := aCap(x0, y0, x1, y1, r, r*0.62)
	ux, uy := x1-x0, y1-y0
	ln := math.Hypot(ux, uy)
	ux, uy = ux/ln, uy/ln
	f.q.fill(ear, func(px, py int, _ float64) color.RGBA {
		dx, dy := (float64(px)+0.5)/fk-x0, (float64(py)+0.5)/fk-y0
		along, across := dx*ux+dy*uy, -dx*uy+dy*ux
		l := ear.lum(px, py, 1.8*fk)*0.85 + 0.12
		if math.Mod(along*1.1+8, 1) < 0.22 || math.Mod(across*1.2+8, 1) < 0.2 {
			l -= 0.2 // between the kernels
		}
		return toneOf(kernels, l)
	})
	for _, side := range []float64{-1, 1} { // the husk, peeled back on both sides
		f.leafAt(x0-ux*1.2, y0-uy*1.2, math.Atan2(-uy, ux)+side*0.5, ln*0.62, r*0.8, 0, leafLance, huskRamp, 0, 0.05*side)
	}
}

func genFloraIcons(icons *[sim.NumItems]*ebiten.Image) {
	for k, c := range floraIconCanvases() {
		icons[k] = c.image()
	}
}

// floraIconCanvases draws the icons, by item.
func floraIconCanvases() map[sim.ItemKind]*canvas {
	out := map[sim.ItemKind]*canvas{}
	mk := func(k sim.ItemKind, draw func(f *flo)) {
		f := newFlo(16, 16)
		draw(f)
		out[k] = iconFrom(f)
	}
	mk(sim.ItemBamboo, func(f *flo) {
		f.bambooCulm(3.5, 14.5, 12.5, 4.5, 1.25, -0.04, bambooRamp)
		f.bambooCulm(8, 14.5, 13, 4.5, 1.45, 0.04, bambooRamp)
		f.leafAt(12.4, 2.4, -0.7, 4, 0.9, 0, leafLance, bambooLeafRamp, 0, 0.05)
		f.leafAt(8.2, 4, 2.6, 4, 0.9, 0, leafLance, bambooLeafRamp, 0, 0)
	})
	mk(sim.ItemGourd, func(f *flo) {
		f.gourd(8, 10.3, 1.5, gourdRamp, 721)
		f.leafAt(8.3, 2.6, 0.3, 4.4, 1.9, 0.1, leafHeart, vineLeafRamp, 3, 0)
	})
	mk(sim.ItemPitcherWater, func(f *flo) { f.pitcher(7.4, 4.6, 1.62, true, 741) })
	mk(sim.ItemFlax, func(f *flo) {
		for i := 0; i < 5; i++ { // a hank of combed fibre, tied in the middle
			o := float64(i) - 2
			f.stalk(fibreRamp, 0.7, 0.5, 0.1+0.04*o, [2]float64{2.5 + o*0.5, 13.5 + o*0.4}, [2]float64{5 + o, 9 + o*0.6}, [2]float64{10 - o*0.4, 8 - o}, [2]float64{13.2 + o*0.3, 2.8 + o*0.7}, nil)
		}
		tie := aCap(6.2, 6.6, 9.6, 10.4, 0.8, 0.8)
		f.q.fill(tie, shd(cordRamp, tie, 0.9, -0.1))
		bloomHD(f.c, 12.4*fk, 12.2*fk, 5.4, flaxBlue, hex(0xf6e27a), 0.4)
	})
	mk(sim.ItemVine, func(f *flo) {
		f.ring(8, 8.8, 5.2, 4.6, 0.95, lianaRamp, 1.6)
		f.liana(0.8, [2]float64{12.6, 7}, [2]float64{14.4, 3}, [2]float64{10, 1.6}, [2]float64{7.6, 2.6})
		f.leafAt(4.4, 5, 2.4, 3.6, 1.8, 0, leafHeart, vineLeafRamp, 3, 0.04)
		f.leafAt(10.6, 12, -0.5, 3.4, 1.7, 0, leafHeart, vineLeafRamp, 3, -0.04)
	})
	mk(sim.ItemRope, func(f *flo) {
		f.ring(7.6, 9, 5.4, 4.4, 1.15, cordRamp, 1.3)
		f.ring(7.6, 7.6, 4.6, 3.6, 1.05, cordRamp, 1.3)
		end := aCap(11.6, 5.4, 14, 2.4, 0.95, 0.8)
		f.q.fill(end, shd(cordRamp, end, 1, 0.1))
	})
	mk(sim.ItemPoisonMushroom, func(f *flo) {
		f.toadstool(11.6, 14.6, 5.4, 2.6, 0.1, false, 765)
		f.toadstool(6.4, 14.8, 9.6, 5, -0.03, false, 763)
	})
	mk(sim.ItemNightshade, func(f *flo) {
		f.leafAt(8, 6, 2.5, 6.4, 2.3, 0.05, leafLance, shadeLeafRamp, 4, 0.06)
		f.leafAt(8, 6, 0.4, 6, 2.2, 0.05, leafLance, shadeLeafRamp, 4, -0.04)
		f.c.line(8*fk, 6*fk, 6*fk, 10*fk, func(int, int, float64) color.RGBA { return shadeLeafRamp[3] })
		f.c.line(8*fk, 6*fk, 10.6*fk, 10.6*fk, func(int, int, float64) color.RGBA { return shadeLeafRamp[3] })
		berryHD(f.c, 5.6*fk, 11.2*fk, 8.2, blackBerry, shadeLeafRamp[0])
		berryHD(f.c, 10.8*fk, 11.6*fk, 7.6, blackBerry, shadeLeafRamp[0])
		berryHD(f.c, 8.2*fk, 8.6*fk, 6.4, blackBerry, shadeLeafRamp[0])
	})
	mk(sim.ItemAloe, func(f *flo) {
		f.aloeLeaf(3.4, 13.2, 13.4, 2.6, 2.3, 0, false, 781)
		f.aloeLeaf(7.6, 14, 14, 8.8, 1.6, -0.06, false, 782)
		f.q.flat(aEll(3.2, 13.4, 1.9, 1.5), plain(hex(0xcfe6b4))) // the cut: clear gel
		f.q.flat(aEll(3.1, 13.3, 1.1, 0.8), plain(hex(0xeef8dc)))
	})
	mk(sim.ItemHerb, func(f *flo) {
		f.stalk(herbRamp, 0.5, 0.4, -0.1, [2]float64{7.6, 14.6}, [2]float64{7.8, 11}, [2]float64{8.4, 8}, [2]float64{8.8, 5}, nil)
		f.leafAt(7.8, 12.6, 2.5, 6.4, 2.1, 0.12, leafSaw, herbRamp, 5, 0)
		f.leafAt(7.8, 12, 0.6, 6.6, 2.1, 0.12, leafSaw, herbRamp, 5, 0.06)
		f.leafAt(7.9, 10.6, 1.9, 4.4, 1.5, 0.05, leafSaw, herbRamp, 4, 0.1)
		for k := 0; k < 9; k++ {
			a := math.Pi * (0.06 + 0.88*float64(k)/8)
			f.c.ellipse((8.8-math.Cos(a)*3.6)*fk, (4.4-math.Sin(a)*2)*fk, 3.4, 3, func(px, py int, nx, ny float64) color.RGBA {
				return creamRamp.pick(0.74+lum(nx, ny)*0.4-0.12*float64(k%2), px, py)
			})
		}
	})
	mk(sim.ItemMoss, func(f *flo) {
		mossPad(f.c, 7.6*fk, 9.6*fk, 6.2*fk, 4.6*fk, true, 811)
		mossPad(f.c, 10.2*fk, 7*fk, 4*fk, 3*fk, true, 812)
		mossPad(f.c, 4.8*fk, 7.6*fk, 3*fk, 2.4*fk, true, 813)
	})
	mk(sim.ItemSeaweed, func(f *flo) {
		frill := leafForm(func(t float64) float64 {
			return math.Min(1, t*5) * math.Pow(1-t, 0.3) * (0.76 + 0.24*math.Sin(t*26))
		})
		f.leafAt(3, 14, 0.95, 14, 2.1, -0.02, frill, kelpRamp, 0, 0)
		f.leafAt(6, 14.4, 0.6, 11, 1.8, -0.05, frill, kelpRamp, 0, -0.08)
		for _, b := range [][2]float64{{6.6, 8.2}, {9.4, 4.6}, {10.4, 10.6}} {
			bl := aEll(b[0], b[1], 1.15, 1)
			f.q.flat(bl, shd(kelpRamp, bl, 1, 0.3))
		}
	})
	mk(sim.ItemLily, func(f *flo) {
		f.pad(8, 11, 6.4, 0.9, padRamp, 0, 836)
		f.bloom(8, 7.4, 6.4, 0.8, lilyPetalRamp, 3)
	})
	mk(sim.ItemLotusSeed, func(f *flo) {
		f.lotusPod(7, 6.4, 1.75)
		for _, s := range [][2]float64{{11.4, 12.6}, {13.2, 10}} { // seeds shaken out
			sd := aEll(s[0], s[1], 1.25, 1.5)
			f.q.fill(sd, shd(seedRamp, sd, 1.1, 0.05))
		}
	})
	mk(sim.ItemHorsetail, func(f *flo) {
		for i, s := range [][5]float64{{5, 14.6, 9, 2.4, 0.75}, {9.4, 14.6, 12.6, 5, 0.65}} {
			x0, y0, x1, y1 := s[0], s[1], s[2], s[3]
			for j := 1; j <= 5; j++ { // whorls of thin branches at the joints
				t := float64(j) / 5.6
				jx, jy := lerpf(x0, x1, t), lerpf(y0, y1, t)
				ln := 4.2 * (1 - t*0.7)
				for _, d := range []float64{-1, 1} {
					f.c.line(jx*fk, jy*fk, (jx+d*ln*0.9)*fk, (jy-ln*0.7)*fk, func(int, int, float64) color.RGBA { return horsetailRamp[3-i] })
					f.c.line(jx*fk, jy*fk, (jx+d*ln*0.45)*fk, (jy-ln*0.95)*fk, func(int, int, float64) color.RGBA { return horsetailRamp[4-i] })
				}
			}
			f.stalk(horsetailRamp, s[4], s[4]*0.6, 0, [2]float64{x0, y0}, [2]float64{lerpf(x0, x1, 0.3), lerpf(y0, y1, 0.3)}, [2]float64{lerpf(x0, x1, 0.7), lerpf(y0, y1, 0.7)}, [2]float64{x1, y1},
				func(sv, v, l float64, col color.RGBA) color.RGBA {
					if u := sv * 5.6; u-math.Floor(u) < 0.2 {
						return horsetailRamp[0]
					}
					return col
				})
		}
	})
	grains := func(f *flo, rp ramp, seed uint32) { // a heap of grain
		for i := 0; i < 34; i++ {
			a, d := hashf(i, 0, seed)*2*math.Pi, math.Sqrt(hashf(i, 1, seed))
			x, y := 8+math.Cos(a)*d*5.6, 10.4+math.Sin(a)*d*3.2-(1-d)*2.6
			g := aEll(x, y, 1.15, 0.7).turned(x, y, hashf(i, 2, seed)*3)
			f.q.fill(g, shd(rp, g, 0.8, (hashf(i, 3, seed)-0.5)*0.3))
		}
	}
	mk(sim.ItemRice, func(f *flo) { grains(f, grainRamp, 881) })
	mk(sim.ItemRoastRice, func(f *flo) {
		bowl := aEll(8, 11.6, 6.4, 3).and(sBelow(11.2 * fk))
		f.lump(bowl, trunkRamp, 2, 0.1)
		for i := 0; i < 26; i++ {
			a, d := hashf(i, 0, 882)*2*math.Pi, math.Sqrt(hashf(i, 1, 882))
			x, y := 8+math.Cos(a)*d*5, 9.2+math.Sin(a)*d*1.9-(1-d)*2.4
			g := aEll(x, y, 1.2, 1)
			f.q.fill(g, shd(puffedRamp, g, 0.9, (hashf(i, 3, 882)-0.5)*0.3))
		}
	})
	mk(sim.ItemCorn, func(f *flo) { f.cornEar(4.6, 12.4, 12.6, 3.4, 3, kernelRamp) })
	mk(sim.ItemRoastCorn, func(f *flo) {
		f.cornEar(4.6, 12.4, 12.6, 3.4, 3, charRamp)
		for _, m := range [][2]float64{{6.6, 9.6}, {8.8, 7.2}, {11, 4.8}} { // charred where it lay on the fire
			f.q.mark(aCap(m[0]-1.2, m[1]-1.4, m[0]+1.2, m[1]+1.2, 0.4, 0.4), plain(charRamp[0]))
		}
	})
	mk(sim.ItemSquash, func(f *flo) { f.squash(8, 9.6, 6.2, 4.8, squashRamp, false, 903) })
	mk(sim.ItemRoastSquash, func(f *flo) {
		// A wedge, roasted: the browned flesh, the dark rind under it.
		rind := aEll(8, 8.6, 6.4, 5.4).and(sBelow(7.4 * fk))
		f.lump(rind, charRamp, 2.4, -0.2)
		flesh := aEll(8, 7.6, 6.4, 2.2)
		f.q.fill(flesh, func(px, py int, _ float64) color.RGBA {
			l := flesh.lum(px, py, 1.2*fk)*0.5 + 0.5
			if hashf(px/3, py/2, 904) < 0.16 {
				l -= 0.3 // the brown of the fire
			}
			return toneOf(charRamp[1:], l)
		})
		for _, s := range [][2]float64{{6, 7.4}, {8.2, 7.9}, {10.2, 7.3}} {
			f.q.flat(aEll(s[0], s[1], 0.7, 0.4), plain(seedRamp[3]))
		}
	})
	mk(sim.ItemSquashSeed, func(f *flo) {
		for i, s := range [][3]float64{{5.2, 6, 0.5}, {10.6, 6.6, -0.6}, {7.8, 11.2, 0.1}} {
			sd := sBlend(0.8, aEll(s[0], s[1]+0.8, 2.3, 2.9), aEll(s[0], s[1]-2, 1.1, 1.2)).turned(s[0], s[1], s[2])
			f.q.fill(sd, shd(seedRamp, sd, 1.6, 0.04*float64(i)))
			f.q.mark(sd.grow(-1.8), plain(seedRamp[4]))
		}
	})
	hide := func(f *flo) shape {
		return sBlend(1.5, aEll(8, 8, 4.6, 5.2), aCap(4.6, 4.6, 2.6, 2.4, 1.3, 0.9), aCap(11.4, 4.6, 13.4, 2.4, 1.3, 0.9),
			aCap(4.6, 11.6, 2.4, 13.8, 1.4, 0.9), aCap(11.4, 11.6, 13.6, 13.8, 1.4, 0.9), aEll(8, 3, 2, 1.6))
	}
	mk(sim.ItemToadSkin, func(f *flo) {
		s := hide(f)
		f.q.fill(s, func(px, py int, _ float64) color.RGBA {
			l := s.lum(px, py, 2.6*fk)*0.85 + 0.1
			u, v := float64(px)/fk, float64(py)/fk
			if vnoise(u*0.62+3, v*0.8+7, 961) > 0.6 {
				return toneOf(toadBlotch, l)
			}
			if hashf(px/2, py/2, 962) < 0.1 {
				l += 0.2
			}
			return toneOf(toadGreen, l)
		})
	})
	mk(sim.ItemPoisonGland, func(f *flo) {
		sac := sBlend(1.2, aEll(7.4, 7.4, 4.6, 3.8), aEll(11, 5, 2, 1.7))
		f.q.fill(sac, func(px, py int, _ float64) color.RGBA {
			l := sac.lum(px, py, 3*fk)*0.9 + 0.08
			if hashf(px/3, py/3, 971) < 0.12 {
				return toneOf(toadGreen, l)
			}
			return toneOf(toadGland, l)
		})
		f.glint(5.8, 5.6, colWhite)
		drop := sBlend(0.6, aEll(8, 13.6, 1.3, 1.4), aCap(8, 13, 8, 11.2, 0.6, 0.2))
		f.q.fill(drop, shd(mkRamp(0x3c6c14, 0x64a01c, 0x9cd42c, 0xd4f66c), drop, 1, 0.1))
	})
	mk(sim.ItemGlowGland, func(f *flo) {
		for i := 0; i < 12; i++ { // the light it gives off
			a := 2 * math.Pi * float64(i) / 12
			f.q.dot(8+math.Cos(a)*6.6, 8.4+math.Sin(a)*6, 0, 0, glowSpot)
			f.q.dot(8+math.Cos(a)*6.6, 8.4+math.Sin(a)*6, 1, 1, glowSpot)
		}
		sac := sBlend(1.2, aEll(7.8, 8.6, 4.4, 3.8), aEll(10.6, 5.6, 1.9, 1.6))
		f.q.fill(sac, func(px, py int, _ float64) color.RGBA {
			l := sac.lum(px, py, 3*fk)*0.9 + 0.1
			if d := math.Hypot(float64(px)/fk-7.4, float64(py)/fk-8.4); d < 2.2 {
				return mix(glowSpotHot, glowSpot, d/2.2)
			}
			return toneOf(glowBelly, l)
		})
	})
	return out
}
