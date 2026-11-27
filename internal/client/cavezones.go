package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The themes of the natural caves in the cave bake (see cavebake.go): the
// waters, the lava and the ice, and how each kind of chamber dresses its
// floor and its walls.
//
//   - Water lies still and dark, the floor seen through it at the shore,
//     ripples and glints of light on it, a wet rim round it. A ford is the
//     same, shallow over stones.
//   - Lava is a crust of dark plates cracked with fire, molten bright where
//     it is deep; the floor beside it is scorched and glows.
//   - A frozen pool is pale blue ice, cracked, with bubbles caught in it
//     and frost round its edge.
//   - The floors: glinting crystal grit in a crystal cavern, threads of
//     mycelium in a mushroom grove, sheets of silk in a nest, trodden earth
//     and bone chips in a den, droppings under a roost, laid flagstones in
//     the tomb, frost in an ice cave, mud and weed in a flooded one.
//   - The walls: roots hanging over the lip under the woods, icicles and
//     glassy ice under the snow, crystals growing out of a crystal cavern's
//     walls, veins of fire in a lava tube's.

// The liquids of the caves, as the bake marks them.
const (
	caveDry uint8 = iota
	caveWater
	caveLava
	caveIce
	caveFord
)

// caveLiquid is what liquid tile k is in the caves.
func caveLiquid(k sim.Tile) uint8 {
	switch k {
	case sim.TileRiver, sim.TileWater:
		return caveWater
	case sim.TileLava:
		return caveLava
	case sim.TileIce:
		return caveIce
	case sim.TileFord:
		return caveFord
	}
	return caveDry
}

var (
	cmWater = mkRamp(0x0b0f1a, 0x101a2c, 0x1a1f2e, 0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5)
	cmLava  = mkRamp(0x2b1416, 0x5a120a, 0x8d3010, 0xc33c20, 0xe2481a, 0xff8a26, 0xffc85a, 0xfff2b0)
	cmIce   = mkRamp(0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xbac9cd, 0xe5efef, 0xffffff)
	cmSilk  = hex(0xe5efef)
	cmBone  = mkRamp(0x867066, 0xb49d7e, 0xc4c6b8, 0xe3e6ac)
	cmShard = mkRamp(0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xe5efef)
	cmRoot  = mkRamp(0x1a1310, 0x2b222a, 0x3c252b, 0x684039, 0x825646)
	cmMyc   = hex(0xb8a1c2)
	cmEmber = hex(0xff8a26)
)

// liquids works out where the waters, lava and ice lie in the bake, and
// how far each pixel is from their shores.
func (b *caveBake) liquids() {
	b.liq = make([]uint8, b.gw*b.gh)
	for gy := 0; gy < b.gh; gy++ {
		v := (float64(b.y0-caveMYT+gy) + 0.5) / hdTH
		for gx := 0; gx < b.gw; gx++ {
			i := gy*b.gw + gx
			if b.rock[i] {
				continue
			}
			u := (float64(b.x0-caveMX+gx) + 0.5) / hdTW
			b.liq[i] = b.liquidAt(u, v)
		}
	}
	b.toDry = chamfer(b.gw, b.gh, func(i int) bool { return b.liq[i] == caveDry })
	b.toWet = chamfer(b.gw, b.gh, func(i int) bool { return b.liq[i] != caveDry && b.liq[i] != caveFord })
}

// liquidAt is the liquid at (u, v), in tiles: the tiles' eased between
// their middles, the edge wavering.
func (b *caveBake) liquidAt(u, v float64) uint8 {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := smooth(x-x0), smooth(y-y0)
	var f [5]float64
	for _, c := range [4]struct {
		dx, dy int
		w      float64
	}{{0, 0, (1 - fx) * (1 - fy)}, {1, 0, fx * (1 - fy)}, {0, 1, (1 - fx) * fy}, {1, 1, fx * fy}} {
		f[caveLiquid(b.tkind[b.tileAt(ix+c.dx, iy+c.dy)])] += c.w
	}
	best := caveDry
	for k := caveWater; k <= caveFord; k++ {
		if f[k] > f[best] || best == caveDry && f[k] > 0 {
			best = k
		}
	}
	if best == caveDry {
		return caveDry
	}
	if f[best]+(vnoise(u*2.3, v*2.6, 3200)-0.5)*0.4 > 0.5 {
		return best
	}
	return caveDry
}

// liquidPixel colours a pixel of water, lava, ice or a ford.
func (b *caveBake) liquidPixel(k uint8, gi, hx, hy int, u, v float64) color.RGBA {
	d := float64(b.toDry[gi]) / 3 // pixels from the shore
	deep := smooth(clamp01(d / 46))
	switch k {
	case caveLava:
		// A crust of dark plates, cracked with fire; molten where it runs
		// deep and the crust has broken up.
		wu := u*1.4 + 0.5*vnoise(u*0.6, v*0.7, 3201)
		wv := v*1.8 + 0.5*vnoise(u*0.6+9, v*0.7, 3201)
		e, id, nx, ny := plates(wu, wv, 3202)
		w := e * hdTW / 1.4
		t := 0.1 + 0.16*deep + (idf(id)-0.5)*0.12 + (fbm2(u*1.4, v*1.7, 3203, 2)-0.5)*0.16
		if m := fbm2(u*0.7, v*0.8, 3217, 2); deep > 0.35 && m > 0.64 {
			t = 0.55 + (m-0.64)*1.6 // open molten rock
		}
		switch {
		case w < 1.2:
			t = 0.86 // the fire in the crack
		case w < 2.6:
			t = math.Max(t, 0.45)
		case w < 4.5 && nx+ny < 0:
			t += 0.06 // the plate's lit edge
		}
		if d < 2.5 {
			t = 0.04 // cooled black at its edge
		}
		return cmLava.g(t, hx, hy)
	case caveIce:
		t := 0.38 + 0.14*deep + (vnoise(u*1.3, v*1.6, 3204)-0.5)*0.14
		e, _, nx, ny := plates(u*1.2, v*1.5, 3205)
		switch w := e * hdTW / 1.2; {
		case w < 0.8:
			t -= 0.3 // a crack
		case w < 2 && nx+ny < 0:
			t += 0.08
		}
		// Light glancing off it in long bands.
		if s := math.Mod((u*hdTW+v*hdTH*1.6)/22+vnoise(u, v, 3206), 1); s < 0.06 {
			t += 0.14
		}
		if p, _ := pebble(hx, hy, 20, 16, 3207, 0.12, 0.8, 1.8); p == pebStone {
			t = 0.95 // a bubble caught in it
		}
		if d < 5 {
			t = math.Max(t, 0.8-d*0.04) // frost at its edge
		}
		return cmIce.g(t, hx, hy)
	case caveFord:
		// Shallow running water over stones.
		if p, l := pebble(hx, hy, 18, 14, 3208, 0.7, 3, 6); p == pebStone {
			return mix(caveMats[sim.RockLimestone].grit.at(1+int(l*3)), cmWater.at(5), 0.35)
		}
		t := 0.6 + (vnoise(u*3, v*1.5, 3209)-0.5)*0.2
		if math.Abs(vnoise(u*2.4, v*4.2, 3210)-0.5) < 0.03 {
			t += 0.2
		}
		return cmWater.g(t, hx, hy)
	}
	// Still water: the floor seen through it at the shore, dark further in,
	// a few ripples and glints, a bright wet rim.
	if d < 1.5 {
		return cmWater.at(6)
	}
	t := 0.64 - 0.36*deep + (fbm2(u*0.8, v*0.9, 3211, 2)-0.5)*0.12
	r := vnoise(u*1.1+0.4*vnoise(u*0.3, v*0.3, 3212), v*2.3, 3213)
	if math.Abs(r-0.5) < 0.012 || math.Abs(r-0.3) < 0.008 || math.Abs(r-0.7) < 0.008 {
		t += 0.2 // ripples, catching the light
	}
	if hashf(hx/2, hy, 3214) < 0.004 {
		t = 0.95 // a glint
	}
	c := cmWater.g(t, hx, hy)
	if d < 12 { // the bottom seen at the shore
		fl := caveMats[sim.RockLimestone].floor.g(0.35, hx, hy)
		c = mix(c, fl, 0.45*(1-d/12))
	}
	return c
}

// shore darkens and wets the floor round water, scorches it round lava and
// frosts it round ice.
func (b *caveBake) shore(c color.RGBA, gi, hx, hy int) color.RGBA {
	if b.toWet == nil {
		return c
	}
	d := float64(b.toWet[gi]) / 3
	if d > 30 {
		return c
	}
	u, v := (float64(hx)+0.5)/hdTW, (float64(hy)+0.5)/hdTH
	k := caveDry
	for dy := -1; dy <= 1 && k == caveDry; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if l := caveLiquid(b.tkind[b.tileAt(int(math.Floor(u))+dx, int(math.Floor(v))+dy)]); l != caveDry && l != caveFord {
				k = l
				break
			}
		}
	}
	switch k {
	case caveWater:
		return mix(c, colInk, 0.35*math.Exp(-d/9))
	case caveLava:
		c = mix(c, colInk, 0.4*math.Exp(-d/16))
		if math.Abs(vnoise(u*3.2, v*3.8, 3215)-0.5) < 0.03 && d < 22 {
			return mix(c, cmEmber, 0.8-d/30) // a glowing crack in the scorched rock
		}
		return mix(c, cmEmber, 0.22*math.Exp(-d/7))
	case caveIce:
		if d < 10+5*vnoise(u*2, v*2, 3216) {
			return mix(c, cmIce.at(6), 0.6)
		}
	}
	return c
}

// zoneFloor dresses floor pixel c in the colours of its chamber z.
func (b *caveBake) zoneFloor(c color.RGBA, z sim.CaveZone, gi, hx, hy int, u, v float64) color.RGBA {
	near := float64(b.toRock[gi]) / 3
	switch z {
	case sim.ZoneCrystal:
		c = mix(c, cmShard.at(1), 0.12)
		if h := hashf(hx, hy, 3220); h < 0.006 {
			return cmShard.at(3 + int(h*300)%2) // a glint of crystal grit
		}
	case sim.ZoneMushroom:
		if vnoise(u*0.9, v*1.1, 3219) > 0.5 && math.Abs(vnoise(u*2.8, v*3.3, 3221)-0.5) < 0.009 {
			return mix(c, cmMyc, 0.32) // the threads of the mycelium
		}
		if vnoise(u*1.8, v*2, 3223) > 0.62 {
			c = mix(c, hex(0x39275e), 0.35)
		}
	case sim.ZoneNest:
		if s := vnoise(u*1.2, v*1.4, 3224); s > 0.62 {
			k := (s - 0.62) * 2.6
			e, _, _, _ := plates(u*3.1, v*3.6, 3225)
			if e*hdTW/3.1 < 0.9 {
				k += 0.35 // a strand
			}
			return mix(c, cmSilk, clamp01(k)*0.55)
		}
	case sim.ZoneDen:
		c = mix(c, colInk, 0.12)
		if p, l := pebble(hx, hy, 26, 20, 3226, 0.08, 1.2, 3.2); p == pebStone {
			return cmBone.at(int(l * 3.5)) // a chip of bone
		}
	case sim.ZoneRoost:
		if p, _ := pebble(hx, hy, 14, 11, 3227, 0.18, 1, 3.4); p == pebStone {
			return mix(c, hex(0xc4c6b8), 0.7) // droppings
		}
	case sim.ZoneFrozen:
		if near < 16+8*vnoise(u*1.5, v*1.5, 3228) || vnoise(u*1.1, v*1.3, 3229) > 0.64 {
			return mix(c, cmIce.at(6), 0.55) // frost and drifted snow
		}
	case sim.ZoneFlooded:
		c = mix(c, colInk, 0.1)
		if vnoise(u*2.6, v*3, 3230) > 0.66 {
			return mix(c, hex(0x3f4328), 0.5) // weed
		}
	case sim.ZoneLava:
		c = mix(c, colInk, 0.15)
	}
	return c
}

// tombFloor is the floor of the desert's tomb: flagstones laid in rows, sand
// drifted into their joints.
func (b *caveBake) tombFloor(m *caveMat, gi, hx, hy int, u, v float64) color.RGBA {
	const sw, sh = 62, 42 // a flagstone, in pixels
	row := fdiv(hy, sh)
	off := 0
	if row%2 != 0 {
		off = sw / 2
	}
	col := fdiv(hx+off, sw)
	x, y := hx+off-col*sw, hy-row*sh
	near := float64(b.toRock[gi]) / 3
	t := 0.55 + (hashf(col, row, 3231)-0.5)*0.22 + (vnoise(u*2, v*2.5, 3232)-0.5)*0.12
	switch {
	case x < 2 || y < 2:
		if vnoise(u*1.6, v*1.8, 3233) > 0.4 {
			return mix(m.floor.g(0.75, hx, hy), colInk, 0.1) // sand in the joint
		}
		t = 0.18
	case x < 4 || y < 4:
		t += 0.1 // the worn edge of the slab
	}
	if hashf(col, row, 3234) < 0.12 && math.Abs(float64(x-y)-float64(sw/2)) < 1 {
		t -= 0.2 // a cracked slab
	}
	t -= b.shade(gi, near)
	return m.floor.g(t, hx, hy)
}

// faceAccent dresses a pixel of a wall's face by its chamber z and its rock
// r: d pixels down from the lip, foot above the floor. It may change tone,
// or give the pixel a colour of its own.
func (b *caveBake) faceAccent(z sim.CaveZone, r sim.CaveRock, hx, hy int, d float64, foot int, tone *float64) (color.RGBA, bool) {
	fx, fy := float64(hx), float64(hy)
	switch {
	case r == sim.RockMossy || z == sim.ZoneStream:
		// Roots hanging over the lip, thin and dark, each its own length.
		cx := fdiv(hx, 9)
		h := hash2(int32(cx), 0, 3240)
		if h&0xff < 110 {
			x := cx*9 + int(h>>8&7)
			x += int(1.5 * math.Sin(fy/9+float64(h>>16&15)))
			ln := 14 + float64(h>>20&63)*0.9
			if d < ln && (hx == x || hx == x+1) {
				l := 0.55 - 0.3*float64(hx-x) - 0.25*d/ln
				return cmRoot.pick(clamp01(l), hx, hy), true
			}
		}
	case r == sim.RockIce || z == sim.ZoneFrozen:
		// Glassy ice: lighter, smooth, with long highlights, icicles under
		// the lip.
		*tone += 0.12
		if s := math.Mod(fx/13+fy/40+vnoise(fx/30, fy/50, 3241), 1); s < 0.05 {
			*tone += 0.25
		}
		cx := fdiv(hx, 6)
		if h := hash2(int32(cx), 1, 3242); h&0xff < 120 {
			x := cx*6 + int(h>>8&3)
			ln := 6 + float64(h>>12&15)
			if w := (1 - d/ln) * 2.2; d < ln && math.Abs(fx-float64(x)) < w {
				return cmIce.pick(0.9-0.3*d/ln, hx, hy), true
			}
		}
	}
	switch z {
	case sim.ZoneCrystal:
		// Crystals growing out of the wall near its foot, catching the light.
		const cw, ch = 30, 26
		cx, cy := fdiv(hx, cw), fdiv(hy, ch)
		if h := hash2(int32(cx), int32(cy), 3243); h&0xff < 90 && foot < 60 {
			px := float64(cx*cw) + 8 + float64(h>>8&15)
			py := float64(cy*ch) + 22
			ht := 10 + float64(h>>16&15)
			dx, dy := fx-px, py-fy
			if dy > 0 && dy < ht && math.Abs(dx) < 3.2*(1-dy/ht)+0.6 {
				l := 0.6 - 0.35*dx/3.2 + 0.3*dy/ht
				return cmShard.pick(clamp01(l), hx, hy), true
			}
		}
	case sim.ZoneLava:
		// Veins of fire in the rock low down.
		if foot < 50 && math.Abs(vnoise(fx/14, fy/9, 3244)-0.5) < 0.02 {
			return mix(cmEmber, hex(0xffc85a), clamp01(1-float64(foot)/50)), true
		}
	case sim.ZoneFlooded, sim.ZoneLake:
		if s := vnoise(fx/3, fy/40, 3245); s > 0.8 {
			*tone += 0.2 // wet, glistening
		}
	}
	return color.RGBA{}, false
}
