package client

import (
	"image/color"
	"math"
	"sync"

	"survivalgame/internal/sim"
)

// The art of the bosses' lairs (see sim/lair.go), drawn in code at
// HDDensity like the trees and the buildings: Titanoboa's nest, the fronds
// it hides under, bones, a shed skin, the tracks of its belly, the stones
// of the ruin, broken trees and the roots that shut its arena; the whale's
// bones, the wreck and the shadow of the Megalodon.

const lk = float64(HDDensity) // image pixels to an art pixel

var (
	lairBoneRamp = mkRamp(0x5e5646, 0x8c8470, 0xb9b29c, 0xdcd6c2, 0xf4f0e2)
	shedRamp     = mkRamp(0x77745c, 0x9d9a7e, 0xc0bd9f, 0xdddabf, 0xf0eedb)
	rootRamp     = mkRamp(0x18120e, 0x2a1f16, 0x3f2f1f, 0x57422a, 0x735a38)
	ruinRamp     = mkRamp(0x23262c, 0x363b40, 0x4d5454, 0x687068, 0x868d7e, 0xa7ab97)
	wreckRamp    = mkRamp(0x1c1714, 0x30261f, 0x473829, 0x5f4c36, 0x7a6446)
	ragRamp      = mkRamp(0x4e4a42, 0x7a7566, 0xa39d88, 0xc6c0a8)
	jungleRamp   = mkRamp(0x0c2218, 0x143424, 0x1d4a2c, 0x2b6335, 0x3f7f3d, 0x5c9a48)
)

// lairArt is every sprite of the lairs.
type lairArt struct {
	nest   sprite
	cover  sprite
	bones  [3]sprite
	skin   sprite
	trail  [4]sprite // along 0, 45, 90 and 135 degrees
	ruin   [2]sprite
	stump  [2]sprite
	whale  sprite
	vine   [3]sprite
	mast   sprite
	hull   sprite
	shadow [2]sprite // the shark under the water, facing right and left
	bird   [2]sprite
}

var (
	lairArtOnce sync.Once
	lairArtV    *lairArt
)

// lairArtOf makes the art on first use; it must run inside the game loop.
func lairArtOf() *lairArt {
	lairArtOnce.Do(func() {
		a := &lairArt{nest: genLairNest(), cover: genLairCover(), skin: genLairSkin(), whale: genLairWhale(),
			mast: genLairMast(), hull: genLairHull()}
		for i := range a.bones {
			a.bones[i] = genLairBones(i)
		}
		for i := range a.trail {
			a.trail[i] = genLairTrail(float64(i) * math.Pi / 4)
		}
		for i := range a.ruin {
			a.ruin[i] = genLairRuin(i)
		}
		a.stump[0], a.stump[1] = sidedPair(genLairStump())
		for i := range a.vine {
			a.vine[i] = genLairVine(uint32(i))
		}
		a.shadow[0], a.shadow[1] = sidedPair(genSharkShadow())
		a.bird[0], a.bird[1] = genLairBird(0), genLairBird(1)
		lairArtV = a
	})
	return lairArtV
}

// longBoneHD is a long bone from (x0, y0) to (x1, y1), w pixels half thick,
// knobbed at both ends.
func longBoneHD(c *canvas, x0, y0, x1, y1, w, lit float64, seed uint32) {
	drawLimbHD(c, limb{x0, y0, (x0 + x1) / 2, (y0 + y1) / 2, x1, y1, w, w}, lairBoneRamp, lit, seed)
	for _, e := range [][2]float64{{x0, y0}, {x1, y1}} {
		c.ellipse(e[0], e[1], w*1.7, w*1.5, func(x, y int, nx, ny float64) color.RGBA {
			return lairBoneRamp.pick(lum(nx, ny)*lit+0.1, x, y)
		})
	}
}

// ribHD is a rib arching from (x0, y0) over (cx, cy) to (x1, y1).
func ribHD(c *canvas, x0, y0, cx, cy, x1, y1, w, lit float64, seed uint32) {
	drawLimbHD(c, limb{x0, y0, cx, cy, x1, y1, w, w * 0.45}, lairBoneRamp, lit, seed)
}

// skullHD is a beast's skull at (x, y), r pixels long, looking right.
func skullHD(c *canvas, x, y, r float64) {
	c.ellipse(x, y, r, r*0.72, func(px, py int, nx, ny float64) color.RGBA {
		return lairBoneRamp.pick(lum(nx, ny)+0.08, px, py)
	})
	c.ellipse(x+r*0.9, y+r*0.2, r*0.7, r*0.42, func(px, py int, nx, ny float64) color.RGBA {
		return lairBoneRamp.pick(lum(nx, ny)*0.95, px, py)
	})
	dark := solid(hex(0x2a231c))
	c.ellipse(x+r*0.15, y-r*0.1, r*0.3, r*0.28, func(px, py int, _, _ float64) color.RGBA { return dark(px, py) })
	c.ellipse(x+r*1.25, y+r*0.1, r*0.12, r*0.1, func(px, py int, _, _ float64) color.RGBA { return dark(px, py) })
	for i := range 4 { // teeth
		tx := x + r*(0.55+0.28*float64(i))
		c.rect(int(tx), int(y+r*0.52), int(max(2, r*0.12)), int(r*0.3), solid(lairBoneRamp.at(4)))
	}
}

func genLairNest() sprite {
	c := hdCanvas(56, 30)
	const cx, cy = 28 * lk, 16 * lk
	c.ellipse(cx, cy, 26*lk, 12.5*lk, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		if d > 0.8 && hashf(x/2, y/2, 901) < (d-0.8)*5 {
			return color.RGBA{}
		}
		return soilRamp.pick(0.15+0.45*d+(vnoise(float64(x)*0.12, float64(y)*0.12, 902)-0.5)*0.4, x, y)
	})
	for i := range 150 { // dry leaves, thick round the rim
		a := hashf(i, 0, 903) * 2 * math.Pi
		r := 0.45 + 0.6*math.Sqrt(hashf(i, 1, 903))
		x, y := cx+math.Cos(a)*r*24*lk, cy+math.Sin(a)*r*11*lk
		rp := bushDryRamp
		if i%3 == 0 {
			rp = strawRamp
		}
		leafHD(c, x, y, (2.2+hashf(i, 2, 903)*1.8)*lk, 1.1*lk, hashf(i, 3, 903)*math.Pi, rp, 0.55+hashf(i, 4, 903)*0.4)
	}
	for i := range 9 { // twigs woven in
		a := hashf(i, 5, 903) * 2 * math.Pi
		x, y := cx+math.Cos(a)*20*lk, cy+math.Sin(a)*9*lk
		drawLimbHD(c, limb{x, y, x + 6*lk*math.Cos(a+1.4), y + 2*lk, x + 12*lk*math.Cos(a+1.7), y + 4*lk*math.Sin(a+1.7), 0.5 * lk, 0.4 * lk}, trunkRamp, 0.8, 910+uint32(i))
	}
	c.outline(0.55)
	return mkUp(c, cx, cy)
}

// genLairCover is the tangle Titanoboa sleeps under: roots of the ancient
// tree arching over the hollow, ferns standing in front of it and fronds
// hanging across, so that only a little of its coils shows between them.
func genLairCover() sprite {
	const W, H = 204, 160
	c := hdCanvas(W, H)
	const cx, base = W / 2 * lk, 152 * lk
	for i, r := range []limb{ // the roots, back to front
		{12 * lk, base, 30 * lk, 30 * lk, 100 * lk, 54 * lk, 4.6 * lk, 2 * lk},
		{192 * lk, base, 170 * lk, 24 * lk, 104 * lk, 48 * lk, 5 * lk, 2 * lk},
		{44 * lk, base, 66 * lk, 70 * lk, 124 * lk, 96 * lk, 3.2 * lk, 1.4 * lk},
		{160 * lk, base, 142 * lk, 76 * lk, 82 * lk, 104 * lk, 3 * lk, 1.2 * lk},
	} {
		drawLimbHD(c, r, rootRamp, 0.85, 920+uint32(i))
	}
	fr := func(i int, stiff bool, up0, up1, lo, hi float64, rp ramp) {
		u := hashf(i, 0, 930)
		x := (10 + u*(W-20)) * lk
		in := (cx - x) / (W / 2 * lk)                         // towards the middle
		a := -math.Pi/2 + in*0.5 + (hashf(i, 1, 930)-0.5)*0.7 // leaning in over the hollow
		L := (lo + hashf(i, 2, 930)*(hi-lo)) * lk
		frondHD(c, x, base-(up0+hashf(i, 3, 930)*(up1-up0))*lk, a, L, rp, stiff, 940+uint32(i))
	}
	for i := range 34 { // tall ferns standing behind, over its back
		fr(i, true, 40, 62, 60, 100, jungleRamp[:4])
	}
	for i := range 170 { // big leaves banked up
		x := (8 + hashf(i, 4, 931)*(W-16)) * lk
		y := base - (2+hashf(i, 5, 931)*96)*lk
		leafHD(c, x, y, (5+hashf(i, 6, 931)*6)*lk, (2+hashf(i, 7, 931)*1.8)*lk, -math.Pi/2+(hashf(i, 8, 931)-0.5)*2.2, jungleRamp, 0.45+hashf(i, 9, 931)*0.5)
	}
	for _, s := range []float64{-1, 1} { // the dark it looks out of
		c.ellipse(cx+s*56*lk, base-69*lk, 14*lk, 8*lk, func(x, y int, nx, ny float64) color.RGBA {
			if dither(x, y) < (nx*nx+ny*ny-0.5)*2 {
				return color.RGBA{}
			}
			return hex(0x07140e)
		})
	}
	for i := 40; i < 66; i++ { // fronds hanging across it
		fr(i, false, 56, 100, 44, 80, palmRamp)
	}
	for i := 70; i < 104; i++ { // ferns in front
		fr(i, true, 2, 16, 40, 84, jungleRamp[1:])
	}
	c.outline(0.7)
	return mkUp(c, cx, base)
}

func genLairBones(v int) sprite {
	c := hdCanvas(34, 22)
	switch v {
	case 0: // a rib cage
		longBoneHD(c, 5*lk, 16*lk, 29*lk, 15*lk, 0.8*lk, 0.9, 950)
		for i := range 6 {
			x := (8 + float64(i)*3.6) * lk
			ribHD(c, x, 15.5*lk, x-3*lk, 3*lk, x+4*lk, (4+float64(i%2))*lk, 0.75*lk, 0.75+0.05*float64(i), 951+uint32(i))
		}
	case 1: // a skull, and what it stood on
		longBoneHD(c, 4*lk, 17*lk, 16*lk, 13*lk, 0.8*lk, 0.85, 960)
		longBoneHD(c, 20*lk, 18*lk, 31*lk, 15*lk, 0.7*lk, 0.8, 961)
		skullHD(c, 12*lk, 9*lk, 4.6*lk)
	default: // a heap
		for i := range 6 {
			a := hashf(i, 0, 970) * math.Pi
			x, y := (10+hashf(i, 1, 970)*14)*lk, (9+hashf(i, 2, 970)*8)*lk
			l := (4 + hashf(i, 3, 970)*5) * lk
			longBoneHD(c, x-math.Cos(a)*l, y-math.Sin(a)*l*0.5, x+math.Cos(a)*l, y+math.Sin(a)*l*0.5, 0.7*lk, 0.7+hashf(i, 4, 970)*0.3, 971+uint32(i))
		}
		skullHD(c, 22*lk, 8*lk, 2.8*lk)
	}
	c.outline(0.7)
	return mkUp(c, 17*lk, 14*lk)
}

// genLairSkin is a skin the snake shed: a papery tube lying in a curve,
// torn open here and there.
func genLairSkin() sprite {
	c := hdCanvas(60, 24)
	n := 240
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x := (5 + t*50) * lk
		y := (12 + math.Sin(t*5.2)*6*(1-t*0.4)) * lk
		r := (2.6 - 1.6*t) * lk
		c.ellipse(x, y, r, r*0.8, func(px, py int, nx, ny float64) color.RGBA {
			if vnoise(float64(px)*0.07, float64(py)*0.09, 981) < 0.3 {
				return color.RGBA{} // torn
			}
			l := 0.35 + 0.5*lum(nx, ny)
			if (px/2+py/2)%4 == 0 || (px/2-py/2+400)%4 == 0 {
				l -= 0.18 // the pattern of its scales
			}
			return shedRamp.pick(l, px, py)
		})
	}
	c.outline(0.45)
	return mkUp(c, 30*lk, 13*lk)
}

// genLairTrail is the track Titanoboa's belly leaves, running along angle
// ang: the ground pressed bare in a slow wave, the grass laid flat beside.
func genLairTrail(ang float64) sprite {
	const W, H = 44, 34
	c := hdCanvas(W, H)
	cos, sin := math.Cos(ang), math.Sin(ang)*0.75
	type pt struct{ x, y float64 }
	var path []pt
	for i := 0; i <= 48; i++ {
		t := float64(i)/48 - 0.5
		s := math.Sin(t*2*math.Pi) * 4.2
		path = append(path, pt{(W/2 + cos*t*38 - sin*s) * lk, (H/2 + sin*t*38 + cos*s) * lk})
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			best, at := math.Inf(1), 0
			for i, p := range path {
				if d := math.Hypot(p.x-float64(x), (p.y-float64(y))*1.2); d < best {
					best, at = d, i
				}
			}
			end := math.Min(float64(at), float64(48-at)) / 8 // fading out at both ends
			w := 3.4 * lk * math.Min(1, 0.4+end)
			switch {
			case best < w*0.62:
				if dither(x, y) < 0.78*math.Min(1, end+0.25) {
					c.put(x, y, soilRamp.pick(0.3+0.3*best/w+(hashf(x/3, y/3, 990)-0.5)*0.3, x, y))
				}
			case best < w:
				if dither(x, y) < 0.42*math.Min(1, end) {
					c.put(x, y, strawRamp.pick(0.2+hashf(x, y/4, 991)*0.4, x, y))
				}
			}
		}
	}
	return mkUp(c, W/2*lk, H/2*lk)
}

// genLairRuin is a stone of the old ruin: a standing pillar with its top
// broken off (0), or a fallen block (1), mossy, a glyph cut in its face.
func genLairRuin(v int) sprite {
	c := hdCanvas(36, 46)
	block := func(x0, y0, x1, y1 float64, seed uint32) {
		for y := int(y0 - 3*lk); y < int(y1); y++ {
			for x := int(x0); x < int(x1); x++ {
				fx := (float64(x) - x0) / (x1 - x0)
				top := y0 + (vnoise(float64(x)*0.11, 3, seed)-0.5)*5*lk // broken off unevenly
				if float64(y) < top {
					continue
				}
				l := 0.72 - 0.42*fx + (vnoise(float64(x)*0.2, float64(y)*0.2, seed+1)-0.5)*0.22
				if float64(y) < top+1.6*lk {
					l += 0.2 // the lit break
				}
				if int(float64(y)-y0)%int(8*lk) < 2 {
					l -= 0.2 // the courses
				}
				if float64(y) > y1-2*lk {
					l -= 0.15
				}
				c.put(x, y, ruinRamp.pick(l, x, y))
			}
		}
		// A glyph: a ring and a stroke.
		gx, gy := (x0+x1)/2, y0+(y1-y0)*0.42
		for a := 0.0; a < 2*math.Pi; a += 0.05 {
			c.put(int(gx+math.Cos(a)*2.4*lk), int(gy+math.Sin(a)*2.4*lk), ruinRamp.at(0))
		}
		c.rect(int(gx-0.4*lk), int(gy-4.5*lk), int(lk), int(9*lk), solid(ruinRamp.at(0)))
		// Moss on its head and down its lit side.
		for i := range 46 {
			u := hashf(i, 0, seed+2)
			mx, my := x0+u*(x1-x0), y0+(hashf(i, 1, seed+2)*hashf(i, 2, seed+2))*(y1-y0)*0.7
			c.ellipse(mx, my, (1+hashf(i, 3, seed+2)*1.8)*lk, 1.1*lk, func(px, py int, nx, ny float64) color.RGBA {
				if !c.opaque(px, py) {
					return color.RGBA{}
				}
				return mossRamp.pick(0.3+0.6*lum(nx, ny), px, py)
			})
		}
	}
	if v == 0 {
		block(11*lk, 8*lk, 25*lk, 42*lk, 1000)
	} else {
		block(4*lk, 26*lk, 22*lk, 42*lk, 1010)
		block(19*lk, 31*lk, 33*lk, 43*lk, 1020)
	}
	for i := range 5 { // rubble at its foot
		stoneHD(c, (6+hashf(i, 0, 1030+uint32(v))*24)*lk, (41.5+hashf(i, 1, 1030)*2)*lk, (1.6+hashf(i, 2, 1030)*1.6)*lk, 1.5*lk, 0.85, ruinRamp, 1031+uint32(i))
	}
	for i := range 9 {
		bladeHD(c, (5+hashf(i, 3, 1040+uint32(v))*26)*lk, 45*lk, (hashf(i, 4, 1040)-0.5)*3*lk, (3+hashf(i, 5, 1040)*4)*lk, 1.2*lk, jungleRamp, 0.3, 0.9)
	}
	c.outline(0.75)
	return mkUp(c, 18*lk, 43*lk)
}

// genLairStump is a tree Titanoboa broke: the splintered stump, and the
// trunk lying where it fell.
func genLairStump() sprite {
	c := hdCanvas(60, 34)
	// The fallen trunk and its dead boughs.
	drawLimbHD(c, limb{20 * lk, 26 * lk, 36 * lk, 24 * lk, 54 * lk, 27 * lk, 3.4 * lk, 2.2 * lk}, trunkRamp, 0.85, 1050)
	for i, b := range []limb{
		{40 * lk, 25 * lk, 44 * lk, 17 * lk, 50 * lk, 14 * lk, 1.1 * lk, 0.5 * lk},
		{47 * lk, 26 * lk, 52 * lk, 20 * lk, 58 * lk, 19 * lk, 0.9 * lk, 0.4 * lk},
		{33 * lk, 25 * lk, 34 * lk, 19 * lk, 38 * lk, 16 * lk, 0.8 * lk, 0.4 * lk},
	} {
		drawLimbHD(c, b, trunkRamp, 0.75, 1051+uint32(i))
	}
	for i := range 16 {
		leafHD(c, (36+hashf(i, 0, 1055)*22)*lk, (14+hashf(i, 1, 1055)*16)*lk, 2.6*lk, 1.1*lk, hashf(i, 2, 1055)*math.Pi, bushDryRamp, 0.5+hashf(i, 3, 1055)*0.4)
	}
	// The stump, its break pale and splintered.
	for x := int(5 * lk); x < int(17*lk); x++ {
		fx := (float64(x) - 5*lk) / (12 * lk)
		top := (17 + 4*math.Abs(math.Sin(fx*9.4))*hashf(x/4, 0, 1060) + 2*fx) * lk
		for y := int(top); y < int(30*lk); y++ {
			if float64(y) < top+2.2*lk {
				c.put(x, y, mix(trunkRamp.at(4), woodRamp.at(1), 0.35+0.4*(1-fx)+hashf(x, y, 1061)*0.2))
				continue
			}
			c.put(x, y, grainHD(trunkRamp, x, y, 0.85-0.5*fx, true, 1062))
		}
	}
	for i := range 3 { // its roots
		x := (4 + float64(i)*6.5) * lk
		drawLimbHD(c, limb{x + 2*lk, 27 * lk, x, 30 * lk, x - 2*lk + float64(i)*2.5*lk, 32 * lk, 1.6 * lk, 0.6 * lk}, trunkRamp, 0.7, 1065+uint32(i))
	}
	c.outline(0.75)
	return mkUp(c, 11*lk, 30*lk)
}

// genLairWhale is the bones of a whale on the sandbar: the long spine, the
// ribs standing out of the sand, the great skull.
func genLairWhale() sprite {
	c := hdCanvas(104, 46)
	for i := range 9 { // the far ribs
		x := (16 + float64(i)*6.4) * lk
		h := (30 - 2.2*math.Abs(float64(i)-3.5)) * lk
		ribHD(c, x, 37*lk, x+5*lk, 37*lk-h, x+11*lk, 37*lk-h*0.55, 1.1*lk, 0.55, 1070+uint32(i))
	}
	drawLimbHD(c, limb{6 * lk, 39 * lk, 40 * lk, 35 * lk, 78 * lk, 38 * lk, 1.2 * lk, 2.2 * lk}, lairBoneRamp, 0.9, 1080)
	for i := range 14 { // the knuckles of its back
		t := float64(i) / 13
		x, y := (8+t*68)*lk, (38.6-math.Sin(t*math.Pi)*2.4)*lk
		c.ellipse(x, y, 1.6*lk, 2.2*lk, func(px, py int, nx, ny float64) color.RGBA { return lairBoneRamp.pick(lum(nx, ny)+0.1, px, py) })
	}
	for i := range 9 { // the near ribs
		x := (14 + float64(i)*6.4) * lk
		h := (32 - 2.4*math.Abs(float64(i)-3.5)) * lk
		ribHD(c, x, 39*lk, x-6*lk, 39*lk-h, x+3*lk, 39*lk-h*0.9, 1.3*lk, 0.95, 1090+uint32(i))
	}
	// The skull: a long beak of bone.
	c.ellipse(84*lk, 36*lk, 9*lk, 5.5*lk, func(px, py int, nx, ny float64) color.RGBA { return lairBoneRamp.pick(lum(nx, ny)+0.05, px, py) })
	c.polygon([]float64{86 * lk, 32 * lk, 103 * lk, 38 * lk, 86 * lk, 40 * lk}, func(px, py int) color.RGBA {
		return lairBoneRamp.pick(0.8-float64(py-int(32*lk))/(10*lk)*0.5, px, py)
	})
	c.ellipse(82*lk, 35*lk, 2.2*lk, 1.6*lk, func(px, py int, _, _ float64) color.RGBA { return hex(0x2a231c) })
	for i := range 12 { // sand blown up against it
		stoneHD(c, (6+hashf(i, 0, 1100)*92)*lk, (41+hashf(i, 1, 1100)*3)*lk, (2+hashf(i, 2, 1100)*3)*lk, 1.4*lk, 1, sandRamp, 1101+uint32(i))
	}
	c.outline(0.7)
	return mkUp(c, 52*lk, 40*lk)
}

// genLairVine is a clump of the roots that rise round Titanoboa's arena:
// thick, twisted together, thorned, a few leaves on them.
func genLairVine(v uint32) sprite {
	c := hdCanvas(44, 66)
	const cx, base = 22 * lk, 62 * lk
	seed := 1200 + v*40
	for i := range 8 {
		u := hashf(i, 0, seed)
		x0 := cx + (u-0.5)*32*lk
		top := (4 + hashf(i, 1, seed)*22) * lk
		bend := (hashf(i, 2, seed) - 0.5) * 34 * lk
		l := limb{x0, base, x0 + bend, (base + top) / 2, cx + (hashf(i, 3, seed)-0.5)*24*lk, top, (3.6 - 0.25*float64(i)) * lk, 0.9 * lk}
		drawLimbHD(c, l, rootRamp, 0.8+0.07*float64(i), seed+uint32(i))
		for j := 1; j < 7; j++ { // thorns
			tx, ty := l.at(float64(j) / 7.5)
			side := float64(1 - 2*((i+j)%2))
			c.lw = 2
			c.line(tx, ty, tx+side*3.4*lk, ty-1.7*lk, func(int, int, float64) color.RGBA { return lairBoneRamp.at(1) })
			c.lw = 0
		}
	}
	for i := range 22 {
		leafHD(c, cx+(hashf(i, 4, seed)-0.5)*34*lk, (8+hashf(i, 5, seed)*44)*lk, 3*lk, 1.4*lk, (hashf(i, 6, seed)-0.5)*2.4, jungleRamp[2:], 0.6+hashf(i, 7, seed)*0.4)
	}
	c.outline(0.8)
	return mkUp(c, cx, base)
}

// genLairMast is the wreck's mast standing out of the sea: leaning, its
// yard askew, rags of sail and rope hanging from it.
func genLairMast() sprite {
	c := hdCanvas(54, 88)
	const fx, fy = 30 * lk, 84 * lk
	mast := limb{fx, fy, 26 * lk, 44 * lk, 18 * lk, 5 * lk, 2 * lk, 1.1 * lk}
	sail := func(pts []float64, seed uint32) {
		c.polygon(pts, func(px, py int) color.RGBA {
			if vnoise(float64(px)*0.05, float64(py)*0.06, seed) < 0.3 {
				return color.RGBA{} // torn through
			}
			return ragRamp.pick(0.35+0.5*vnoise(float64(px)*0.03, float64(py)*0.12, seed+1), px, py)
		})
	}
	sail([]float64{6 * lk, 25 * lk, 34 * lk, 19 * lk, 31 * lk, 44 * lk, 22 * lk, 50 * lk, 12 * lk, 40 * lk}, 1300)
	drawLimbHD(c, mast, wreckRamp, 0.9, 1310)
	drawLimbHD(c, limb{3 * lk, 26 * lk, 20 * lk, 21 * lk, 40 * lk, 17 * lk, 1.1 * lk, 0.9 * lk}, wreckRamp, 0.85, 1311) // the yard
	drawLimbHD(c, limb{14 * lk, 58 * lk, 26 * lk, 55 * lk, 42 * lk, 60 * lk, 0.9 * lk, 0.7 * lk}, wreckRamp, 0.75, 1312)
	c.ellipse(23*lk, 33*lk, 5*lk, 2.4*lk, func(px, py int, nx, ny float64) color.RGBA { // the top
		return wreckRamp.pick(0.3+0.6*lum(nx, ny), px, py)
	})
	for i, r := range [][4]float64{{4, 26, 26, 62}, {39, 17, 29, 70}, {20, 8, 5, 30}} { // rigging
		c.line(r[0]*lk, r[1]*lk, r[2]*lk, r[3]*lk, func(px, py int, t float64) color.RGBA {
			if (px+py+i)%5 == 0 {
				return color.RGBA{}
			}
			return ropeRamp.at(0)
		})
	}
	for i := range 14 { // weed on its foot
		leafHD(c, fx+(hashf(i, 0, 1320)-0.5)*7*lk, fy-(1+hashf(i, 1, 1320)*14)*lk, 2*lk, 0.9*lk, hashf(i, 2, 1320)*math.Pi, mossRamp, 0.7)
	}
	c.outline(0.8)
	return mkUp(c, fx, fy)
}

// genLairHull is the wreck's hull lying on the bottom, seen through the
// water: a dark shape, its ribs showing.
func genLairHull() sprite {
	c := hdCanvas(120, 44)
	c.ellipse(60*lk, 22*lk, 56*lk, 16*lk, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		if nx > 0.2 && math.Abs(ny) > 1.25-nx { // the bow comes to a point
			return color.RGBA{}
		}
		if dither(x, y) < (d-0.6)*2.2 {
			return color.RGBA{}
		}
		l := 0.25
		if int(float64(x)/(5*lk))%2 == 0 && math.Abs(ny) < 0.8 {
			l = 0.36 // its ribs
		}
		if math.Abs(ny) < 0.08 {
			l = 0.7 // the keel
		}
		return mix(hex(0x071320), hex(0x25384a), l)
	})
	return mkUp(c, 60*lk, 22*lk)
}

// genSharkShadow is the Megalodon under the water: only a darkness the
// shape of a shark.
func genSharkShadow() sprite {
	c := hdCanvas(150, 56)
	dark := func(x, y int, nx, ny float64) color.RGBA {
		if dither(x, y) < (nx*nx+ny*ny-0.55)*2 {
			return color.RGBA{}
		}
		return hex(0x06101c)
	}
	c.ellipse(80*lk, 28*lk, 46*lk, 11*lk, dark)
	c.ellipse(112*lk, 28*lk, 22*lk, 8*lk, dark) // the head
	flat := func(px, py int) color.RGBA {
		if dither(px, py) < 0.25 {
			return color.RGBA{}
		}
		return hex(0x06101c)
	}
	c.polygon([]float64{40 * lk, 28 * lk, 8 * lk, 8 * lk, 20 * lk, 28 * lk, 6 * lk, 50 * lk}, flat) // the tail
	c.polygon([]float64{84 * lk, 20 * lk, 70 * lk, 2 * lk, 64 * lk, 22 * lk}, flat)                 // fins
	c.polygon([]float64{84 * lk, 36 * lk, 68 * lk, 54 * lk, 64 * lk, 34 * lk}, flat)
	return mkUp(c, 80*lk, 28*lk)
}

// genLairBird is a bird seen from below, wings up (0) or down (1).
func genLairBird(f int) sprite {
	c := hdCanvas(14, 10)
	dy := -3.0
	if f == 1 {
		dy = 2.2
	}
	ink := hex(0x1a1c26)
	c.lw = 2
	for _, s := range []float64{-1, 1} {
		c.line(7*lk, 5*lk, (7+s*3)*lk, (5+dy*0.7)*lk, func(int, int, float64) color.RGBA { return ink })
		c.line((7+s*3)*lk, (5+dy*0.7)*lk, (7+s*6)*lk, (5+dy*0.4)*lk, func(int, int, float64) color.RGBA { return ink })
	}
	c.ellipse(7*lk, 5*lk, 1.5*lk, 1.1*lk, func(int, int, float64, float64) color.RGBA { return ink })
	return mkUp(c, 7*lk, 5*lk)
}

// lairFlat reports whether lair prop v lies flat on the ground, under
// everything that stands.
func lairFlat(v uint8) bool {
	switch v {
	case sim.LairNest, sim.LairBones, sim.LairSkin, sim.LairTrail:
		return true
	}
	return false
}
