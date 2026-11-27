package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The look of each kind of ground at HD density. Every function colours one
// image pixel (hx, hy) of the world (groundDens of them to an art pixel)
// lying at (u, v) in tiles. The broad tones come from the same noise as the
// old art-pixel ground, so the world keeps its light and dark places; the
// detail is drawn at image-pixel size: blades and tufts of grass, grains,
// pebbles lit from the upper left with a shadow under them, cracks.

// sub is r with n shades to each of its steps, for art three times finer.
func (r ramp) sub(n int) ramp {
	out := make(ramp, 0, (len(r)-1)*n+1)
	for i := 0; i < len(r)-1; i++ {
		for j := 0; j < n; j++ {
			out = append(out, mix(r[i], r[i+1], float64(j)/float64(n)))
		}
	}
	return append(out, r[len(r)-1])
}

// gi maps t in [0, 1] to an index of a fine ramp, with a grain: neighbouring
// shades mingle instead of meeting along a line.
func (r ramp) gi(t float64, hx, hy int) int {
	p := clamp01(t)*float64(len(r)-1) + (hashf(hx, hy, 900)-0.5)*1.2
	return max(0, min(len(r)-1, int(math.Round(p))))
}

func (r ramp) g(t float64, hx, hy int) color.RGBA { return r[r.gi(t, hx, hy)] }

var (
	grassHD     = grassRamp.sub(3)
	forestHD    = forestRamp.sub(3)
	sandHD      = sandRamp.sub(3)
	rockHD      = rockRamp.sub(3)
	cliffHD     = cliffRamp.sub(3)
	litterHD    = mkRamp(0x4a3a24, 0x5a4a2f, 0x75603b, 0x8f7444, 0xa8853f).sub(2)
	desertHD    = desertRamp.sub(3)
	marshHD     = marshRamp.sub(3)
	ashHD       = ashRamp2.sub(3)
	lavaHD      = lavaRamp.sub(4)
	caveHD      = caveRamp.sub(3)
	mtnHD       = mtnRamp.sub(3)
	mtnFaceHD   = mtnFace.sub(3)
	caveWallHD  = caveWall.sub(3)
	crystalHD   = crystalRamp.sub(3)
	caveDeepHD  = caveDeep.sub(3)
	pathHD      = pathRamp.sub(3)
	iceHD       = iceRamp.sub(3)
	snowHD      = snowRamp.sub(3)
	swampMudHD  = swampMud.sub(3)
	pondHD      = pondRamp.sub(3)
	algaeHD     = algaeRamp.sub(3)
	puddleHD    = puddleRamp.sub(3)
	warmDirtHD  = warmDirtRamp.sub(3)
	warmGrassHD = warmGrassRamp.sub(3)
	warmMossHD  = warmMossRamp.sub(3)
	warmStoneHD = warmStone.sub(3)
	warmPathHD  = mkRamp(0x7d5a36, 0x967046, 0xae8757, 0xc49f6c, 0xd8b784).sub(3)
	sandStoneHD = mkRamp(0x7d6a50, 0x9a8667, 0xb8a482, 0xd6c6a6).sub(3)
)

// --- shapes -------------------------------------------------------------------

// Parts of a blade of grass.
const (
	bladeNone = iota
	bladeFoot
	bladeStem
	bladeTip
)

// blade reports whether (hx, hy) lies on a blade of grass of a layer with
// one blade, at most, to each cell of cw by ch pixels, and on which part of
// it. dens is the share of cells that grow one.
func blade(hx, hy, cw, ch int, seed uint32, dens float64) int {
	cx := fdiv(hx, cw)
	y := hy + int(hashf(cx, 7, seed)*float64(ch)) // each column of cells is shifted, so the feet do not line up
	cy := fdiv(y, ch)
	h := hash2(int32(cx), int32(cy), seed+1)
	if float64(h&0xffff)/65536 >= dens {
		return bladeNone
	}
	x0 := cx*cw + 1 + int(h>>16&0xff)*(cw-2)/256
	n := 3 + int(h>>24)*(ch-3)/256
	r := y - (cy*ch + ch - n) // 0 at the tip
	if r < 0 || r >= n {
		return bladeNone
	}
	if n >= 5 && r < n/2 {
		if lean := int(hash2(int32(cx), int32(cy), seed+2) & 3); lean < 3 {
			x0 += lean - 1 // the upper half leans
		}
	}
	if hx != x0 {
		return bladeNone
	}
	switch r {
	case 0:
		return bladeTip
	case n - 1:
		return bladeFoot
	}
	return bladeStem
}

// tuft is blade for a layer of tufts: a fan of a few blades from one root.
func tuft(hx, hy, cw, ch int, seed uint32, dens float64) int {
	cx := fdiv(hx, cw)
	y := hy + int(hashf(cx, 7, seed)*float64(ch))
	cy := fdiv(y, ch)
	h := hash2(int32(cx), int32(cy), seed+1)
	if float64(h&0xffff)/65536 >= dens {
		return bladeNone
	}
	rootX := float64(cx*cw) + float64(cw)/2 + float64(int(h>>16&3)) - 1.5
	rootY := cy*ch + ch - 1
	up := rootY - y // 0 at the root
	if up < 0 {
		return bladeNone
	}
	nb := 3 + int(h>>18&3)
	part := bladeNone
	for j := 0; j < nb; j++ {
		hj := hash2(int32(cx*8+j), int32(cy), seed+2)
		tall := float64(ch-2) * (0.55 + 0.45*float64(hj&0xff)/255)
		if float64(up) > tall {
			continue
		}
		tt := float64(up) / tall
		spread := (float64(j)/float64(nb-1) - 0.5) * float64(cw-3) * (0.75 + 0.25*float64(hj>>8&0xff)/255)
		bx := rootX + spread*tt*(0.4+0.6*tt)
		if math.Abs(float64(hx)+0.5-bx) > 0.5 {
			continue
		}
		switch {
		case tt > 0.78:
			return bladeTip
		case tt < 0.25:
			part = max(part, bladeFoot)
		default:
			part = max(part, bladeStem)
		}
	}
	return part
}

// Parts of a pebble.
const (
	pebNone = iota
	pebStone
	pebShadow
)

// pebble reports whether (hx, hy) lies on a pebble or in its shadow, in a
// layer with at most one pebble to each cell of cw by ch pixels, between
// rmin and rmax pixels across (halved); l is how brightly a stone pixel is
// lit, from the upper left.
func pebble(hx, hy, cw, ch int, seed uint32, dens, rmin, rmax float64) (part int, l float64) {
	cx, cy := fdiv(hx, cw), fdiv(hy, ch)
	h := hash2(int32(cx), int32(cy), seed)
	if float64(h&0xffff)/65536 >= dens {
		return pebNone, 0
	}
	rx := rmin + (rmax-rmin)*float64(h>>16&0xff)/255
	ry := rx * (0.62 + 0.25*float64(h>>24)/255)
	h2 := hash2(int32(cx), int32(cy), seed+1)
	px := float64(cx*cw) + rx + 0.5 + float64(h2&0xff)/255*math.Max(0, float64(cw)-2*rx-2.5)
	py := float64(cy*ch) + ry + 0.5 + float64(h2>>8&0xff)/255*math.Max(0, float64(ch)-2*ry-2.5)
	dx, dy := (float64(hx)+0.5-px)/rx, (float64(hy)+0.5-py)/ry
	if d := dx*dx + dy*dy; d <= 1 {
		l = 0.56 - 0.34*dx - 0.46*dy
		if d > 0.6 && dx+dy > 0.2 {
			l -= 0.12 // the rim in its own shade
		}
		return pebStone, clamp01(l)
	}
	sx, sy := (float64(hx)+0.5-px-1)/rx, (float64(hy)+0.5-py-1.3)/ry
	if sx*sx+sy*sy <= 1 {
		return pebShadow, 0
	}
	return pebNone, 0
}

// plates is a cracked surface: (x, y), in cells, lies on one of the plates
// the surface is broken into. edge is how far it is from the plate's border
// (in cells), id tells the plates apart and (nx, ny) points across the
// nearest border to the plate beyond.
func plates(x, y float64, seed uint32) (edge float64, id uint32, nx, ny float64) {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	d1, d2 := math.Inf(1), math.Inf(1)
	var p1x, p1y, p2x, p2y float64
	for cy := iy - 1; cy <= iy+1; cy++ {
		for cx := ix - 1; cx <= ix+1; cx++ {
			h := hash2(int32(cx), int32(cy), seed)
			qx := float64(cx) + 0.1 + 0.8*float64(h&0xffff)/65535
			qy := float64(cy) + 0.1 + 0.8*float64(h>>16)/65535
			d := (qx-x)*(qx-x) + (qy-y)*(qy-y)
			switch {
			case d < d1:
				d2, p2x, p2y = d1, p1x, p1y
				d1, p1x, p1y, id = d, qx, qy, h
			case d < d2:
				d2, p2x, p2y = d, qx, qy
			}
		}
	}
	nx, ny = p2x-p1x, p2y-p1y
	n := math.Hypot(nx, ny)
	if n == 0 {
		return 1, id, 0, 0
	}
	return (d2 - d1) / (2 * n), id, nx / n, ny / n
}

// idf is a plate's id as a number in [0, 1).
func idf(id uint32) float64 { return float64(id*0x9e3779b1>>8) / (1 << 24) }

// --- grass and woods ------------------------------------------------------------

// turf draws grass of ramp rp with tone t: a carpet of short blades, taller
// ones among them and here and there a tuft, each dark at the foot and lit
// at the tip. lush (0 to 1) is how thick the tall growth is.
func turf(rp ramp, t float64, hx, hy int, seed uint32, lush float64) color.RGBA {
	i := rp.gi(t, hx, hy)
	switch blade(hx, hy, 3, 6, seed, 0.55) {
	case bladeFoot:
		i -= 2
	case bladeStem:
		i++
	case bladeTip:
		i += 2
	}
	switch blade(hx+1, hy+2, 5, 10, seed+10, 0.3+0.3*lush) {
	case bladeFoot:
		i -= 3
	case bladeStem:
		i += 2
	case bladeTip:
		i += 4
	default:
		if blade(hx, hy+2, 5, 10, seed+10, 0.3+0.3*lush) != bladeNone {
			i -= 2 // the dark between two blades
		}
	}
	switch tuft(hx, hy, 19, 15, seed+20, 0.06+0.34*lush) {
	case bladeFoot:
		i -= 4
	case bladeStem:
		i += 3
	case bladeTip:
		i += 5
	default:
		if tuft(hx-1, hy-1, 19, 15, seed+20, 0.06+0.34*lush) != bladeNone {
			i -= 3 // a tuft's shadow
		}
	}
	return rp.at(i)
}

// flower draws the small flowers of a meadow: four petals round a heart,
// on a short stalk. dens is the share of 9 by 9 pixel cells with one.
func flower(hx, hy int, seed uint32, dens float64) (color.RGBA, bool) {
	cx, cy := fdiv(hx, 9), fdiv(hy, 9)
	h := hash2(int32(cx), int32(cy), seed)
	if float64(h&0xffff)/65536 >= dens {
		return color.RGBA{}, false
	}
	dx := hx - (cx*9 + 2 + int(h>>16&0xff)*5/256)
	dy := hy - (cy*9 + 2 + int(h>>24)*4/256)
	col := flowers[int(hashf(cx, cy, seed+1)*float64(len(flowers)))]
	switch {
	case dx == 0 && dy == 0:
		if col == flowers[1] {
			return hex(0xd9892b), true
		}
		return hex(0xf6d55c), true
	case dx*dx+dy*dy == 1:
		return col, true
	case dx == 0 && (dy == 2 || dy == 3):
		return grassHD.at(1), true
	}
	return color.RGBA{}, false
}

func grassGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.22, v*0.3, 31, 3)
	t := 0.42 + (n-0.5)*1.3 + (vnoise(u*1.7, v*2.2, 32)-0.5)*0.35
	clump := vnoise(u*6.3, v*8.1, 37)
	t += (clump - 0.5) * 0.14
	sward := coverSward(u, v) // thick and dark here, thin and light there: see coverground.go
	t -= sward * 0.06
	meadow := fbm2(u*0.16+40, v*0.2, 34, 2)
	dens := 0.003
	if meadow > 0.62 {
		dens = 0.03
	}
	if c, ok := flower(hx, hy, 35, dens); ok {
		return c
	}
	patch, pw := coverPatch(hx, hy, u, v) // clover, moss, worn earth: see coverground.go
	if pw >= 1 {
		return patch
	}
	c := turf(grassHD, t, hx, hy, 33, clamp01((clump-0.35)*2.2+sward*0.4))
	if pw > 0 {
		c = mix(c, patch, pw)
	}
	// The meadow is not all one green: sunnier here, deeper there.
	if hue := fbm2(u*0.45+7, v*0.55, 38, 2); hue > 0.55 {
		c = mix(c, hex(0x9bb548), math.Min(0.22, (hue-0.55)*1.2))
	} else if hue < 0.42 {
		c = mix(c, hex(0x2f6e4a), math.Min(0.2, (0.42-hue)*1.2))
	}
	return c
}

func forestGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.3, v*0.4, 41, 3)
	moss := vnoise(u*4.7, v*5.9, 44)
	t := 0.45 + (n-0.5)*1.2 + (moss-0.5)*0.2
	if c, ok := forestPatch(hx, hy, u, v); ok { // moss and bare duff: see coverground.go
		return c
	}
	// Fallen leaves, each a pixel or three with a lit end, in drifts.
	cx, cy := fdiv(hx, 6), fdiv(hy+int(hashf(fdiv(hx, 6), 3, 42)*5), 5)
	if h := hash2(int32(cx), int32(cy), 42); float64(h&0xff) < 46*litterThick(u, v) {
		dx := hx - (cx*6 + 1 + int(h>>8&3))
		dy := hy + int(hashf(cx, 3, 42)*5) - (cy*5 + 1 + int(h>>10&3)%3)
		long := 2 + int(h>>12&1)
		slope := int(h>>13&3) - 1 // -1, 0, 1, 2
		if slope == 2 {
			slope = 0
		}
		if dx >= 0 && dx < long && dy == dx*slope/2 {
			i := int(h >> 16 & 0xff * uint32(len(litterHD)-2) / 256)
			if dx == 0 {
				i += 2
			}
			return litterHD.at(i)
		}
	}
	// Twigs.
	tx, ty := fdiv(hx, 23), fdiv(hy, 17)
	if h := hash2(int32(tx), int32(ty), 45); h&0xff < 34 {
		x0, y0 := tx*23+2+int(h>>8&7), ty*17+4+int(h>>11&7)
		long := 6 + int(h>>14&7)
		slope := (float64(h>>17&0xff)/255 - 0.5) * 0.9
		if dx := hx - x0; dx >= 0 && dx < long {
			switch hy - (y0 + int(math.Round(float64(dx)*slope))) {
			case 0:
				return litterHD.at(4 + dx%2)
			case 1:
				return forestHD.at(0) // its shadow
			}
		}
	}
	return turf(forestHD, t, hx, hy, 43, clamp01((moss-0.45)*1.6))
}

// warmRag is how far the ragged rim of a patch of grass reaches at a pixel:
// blades poke out over the earth, and earth shows between them.
func warmRag(hx, hy int) float64 {
	row := fdiv(hy+int(hashf(hx, 0, 703)*7), 7)
	return (hashf(hx, row, 704)-0.5)*0.075 +
		(hashf(fdiv(hx, 2), fdiv(hy, 3), 705)-0.5)*0.04 +
		(vnoise(float64(hx)/10, float64(hy)/9, 706)-0.5)*0.1
}

// warmEarth is the warm look's grass or wood floor: ochre earth with grass
// (moss, in the woods) growing over it in ragged patches, pebbles and dry
// cracks in the bare earth.
func warmEarth(hx, hy int, u, v float64, forest bool) color.RGBA {
	// How grassy this spot is: big soft patches, broken up finer.
	g := fbm2(u*0.16, v*0.21, 701, 3)*0.75 + vnoise(u*0.9, v*1.15, 702)*0.25
	cut := 0.5
	if forest {
		cut = 0.36 // the woods are mossier
	}
	edge := g - cut
	if e := edge + warmRag(hx, hy); e > 0 {
		rp := warmGrassHD
		if forest {
			rp = warmMossHD
		}
		clump := vnoise(u*6.3, v*8.1, 37)
		n := fbm2(u*0.35, v*0.45, 705, 2)
		t := 0.5 + (n-0.5)*0.9 + math.Min(edge, 0.15)*1.5 + (clump-0.5)*0.14
		if e < 0.03 {
			t -= 0.12 // the thin grass at a patch's rim is darker
		}
		if edge+warmRag(hx, hy+2) <= 0 {
			t -= 0.2 // and darkest at its foot, over the earth
		}
		if !forest {
			if c, ok := flower(hx, hy, 35, 0.004); ok {
				return c
			}
		}
		return turf(rp, t, hx, hy, 33, clamp01((clump-0.35)*2.2)*clamp01(edge*12))
	}
	// Bare earth.
	n := fbm2(u*0.3, v*0.38, 707, 3)
	t := 0.55 + (n-0.5)*0.8 + edge*0.6 + (vnoise(u*5.1, v*6.7, 718)-0.5)*0.1
	if edge+warmRag(hx, hy-2) > 0 || edge+warmRag(hx-1, hy-1) > 0 {
		t -= 0.22 // the shade of the grass standing over it
	}
	i := warmDirtHD.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 17, 14, 708, 0.2, 1.3, 3.3); p {
	case pebStone:
		return warmStoneHD.g(0.15+l*0.85, hx, hy)
	case pebShadow:
		return warmDirtHD.at(i - 5)
	}
	switch p, l := pebble(hx+6, hy+4, 8, 7, 709, 0.12, 0.8, 1.3); p {
	case pebStone:
		return warmStoneHD.g(0.25+l*0.7, hx, hy)
	case pebShadow:
		return warmDirtHD.at(i - 4)
	}
	// Dry cracks, where the earth has baked.
	if dry := fbm2(u*0.5, v*0.5, 712, 2); dry > 0.5 {
		e, _, nx, ny := plates(u*2.3, v*2.9, 711)
		w := e * hdTW / 2.3 // in pixels, near enough
		open := 0.45 + clamp01((dry-0.5)*8)*0.5
		if vnoise(u*4.3, v*4.9, 716) > 0.38 {
			switch {
			case w < open:
				return warmDirtHD.at(i - 6)
			case w < open+1 && nx+ny < 0:
				i += 2 // the lit lip of the crack
			}
		}
	}
	// A blade or two that has taken root.
	if blade(hx, hy, 9, 10, 717, 0.045) != bladeNone {
		if forest {
			return warmMossHD.at(5)
		}
		return warmGrassHD.at(6)
	}
	// Grains.
	switch h := hashf(hx, hy, 713); {
	case h < 0.035:
		i += 2
	case h < 0.07:
		i -= 2
	case h < 0.078:
		i -= 4
	}
	return warmDirtHD.at(i)
}

// --- sand, stone, paths --------------------------------------------------------

func sandGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.3, v*0.4, 61, 2)
	t := 0.55 + (n-0.5)*0.8
	// Ripples: each lit on the side facing the sun, with a thin shadow behind
	// its crest.
	sn, cs := math.Sincos((v*2.4 + u*0.5 + fbm2(u*0.5, v*0.6, 62, 2)*2.5) * 2 * math.Pi)
	amp := 0.35 + 0.65*vnoise(u*0.8, v*0.9, 64)
	if sn > 0.55 {
		k := (sn - 0.55) / 0.45
		if cs > 0 {
			t += 0.12 * k * amp
		} else {
			t -= 0.24 * k * k * amp
		}
	}
	t += (vnoise(u*7.1, v*8.3, 66) - 0.5) * 0.08
	i := sandHD.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 24, 19, 65, 0.07, 0.9, 2.2); p {
	case pebStone:
		return sandStoneHD.g(0.2+l*0.8, hx, hy)
	case pebShadow:
		return sandHD.at(i - 4)
	}
	switch h := hashf(hx, hy, 63); {
	case h < 0.05:
		i += 3
	case h < 0.1:
		i -= 3
	case h < 0.105:
		return mix(sandHD.at(0), hex(0x6b5a3e), 0.5) // a dark grain
	case h < 0.107:
		return hex(0xfbf3de) // a chip of shell
	}
	return sandHD.at(i)
}

func desertGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.2, v*0.28, 101, 3)
	t := 0.5 + (n-0.5)*0.9
	// Dunes, bent by the wind: the slope up to each crest brightens, and
	// behind the crest lies its shadow, fading out.
	ph := v*1.5 + u*0.35 + fbm2(u*0.3, v*0.4, 102, 2)*3
	f := ph - math.Floor(ph)
	amp := 0.55 + 0.45*vnoise(u*0.45, v*0.5, 104)
	switch {
	case f < 0.72:
		t += 0.2 * amp * (f / 0.72) * (f / 0.72)
	case f < 0.75:
		t += 0.2 * amp * (1 - (f-0.72)/0.03*2.6) // over the crest
	default:
		k := 1 - (f-0.75)/0.25
		t -= 0.32 * amp * k * k
	}
	// Fine ripples the wind draws across the dunes.
	if r := math.Sin((v*13 + u*3.5 + fbm2(u*0.9, v*1.1, 105, 2)*5) * 2 * math.Pi); r > 0.6 {
		t -= 0.06
	} else if r < -0.7 {
		t += 0.04
	}
	i := desertHD.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 29, 23, 106, 0.05, 0.9, 2.4); p {
	case pebStone:
		return mix(desertHD.at(1), sandStoneHD.g(0.2+l*0.8, hx, hy), 0.7)
	case pebShadow:
		return desertHD.at(i - 4)
	}
	switch h := hashf(hx, hy, 103); {
	case h < 0.05:
		i += 3
	case h < 0.09:
		i -= 3
	}
	return desertHD.at(i)
}

func rockyGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.35, v*0.45, 51, 3)
	t := 0.45 + (n-0.5)*1.1
	// Slabs of stone, each its own shade, with cracks between them: dark,
	// and lit along the edge that faces the sun.
	e, id, nx, ny := plates(u*1.9, v*2.4, 56)
	t += (idf(id) - 0.5) * 0.2
	w := e * hdTW / 1.9
	if split := vnoise(u*1.3, v*1.5, 59); split > 0.4 { // the cracks come and go
		open := 0.3 + clamp01((split-0.4)*5)*0.5
		switch {
		case w < open:
			t -= 0.4
		case w < open+1.1:
			if nx+ny < 0 {
				t += 0.14
			} else {
				t -= 0.12
			}
		}
	}
	t += (vnoise(u*6.1, v*7.7, 60) - 0.5) * 0.14
	i := rockHD.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 17, 14, 52, 0.15, 1, 3.4); p {
	case pebStone:
		return rockHD.g(0.3+l*0.7, hx, hy)
	case pebShadow:
		return rockHD.at(i - 5)
	}
	// Lichen.
	if lich := vnoise(u*2.7, v*3.1, 57); lich > 0.7 && hashf(fdiv(hx, 2), fdiv(hy, 2), 58) < (lich-0.7)*3 {
		return mix(rockHD.at(i), hex(0x8a9a5a), 0.35)
	}
	switch h := hashf(hx, hy, 55); {
	case h < 0.04:
		i += 2
	case h < 0.08:
		i -= 2
	}
	return rockHD.at(i)
}

// pathGround is a beaten trail: packed earth with ruts worn by feet, stones
// trodden into it and grit.
func pathGround(hx, hy int, u, v float64, warm bool) color.RGBA {
	rp := pathHD
	if warm {
		rp = warmPathHD
	}
	n := fbm2(u*0.4, v*0.5, 181, 3)
	t := 0.5 + (n-0.5)*0.7 + (vnoise(u*5.3, v*6.1, 184)-0.5)*0.1
	if r := math.Abs(vnoise(u*1.3, v*0.6, 182) - 0.5); r < 0.05 {
		t -= 0.16 * (1 - r/0.05) // a rut
	} else if r < 0.065 {
		t += 0.05 // its rim
	}
	i := rp.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 14, 11, 183, 0.2, 0.9, 2.6); p {
	case pebStone:
		return warmStoneHD.g(0.1+l*0.8, hx, hy)
	case pebShadow:
		return rp.at(i - 5)
	}
	switch h := hashf(hx, hy, 185); {
	case h < 0.04:
		i += 2
	case h < 0.08:
		i -= 2
	case h < 0.086:
		i -= 4
	}
	// Scuffs: short strokes along the way people walk.
	if hashf(fdiv(hx, 4), hy, 186) < 0.03 {
		i--
	}
	return rp.at(i)
}

// --- swamp ---------------------------------------------------------------------

// swampBank darkens land at the edge of a pond (s tiles from the water,
// 0 <= s < 0.22) into wet mud.
func swampBank(c color.RGBA, hx, hy int, s float64) color.RGBA {
	return mix(c, swampMudHD.g(0.6, hx, hy), 0.75*(1-s/0.22))
}

// swampWater colours a pixel of a pond, s (< 0) tiles out from the bank and
// up pixels below the nearest land: a lip of mud, then a band of algae
// along the edge, then still, murky water speckled with duckweed. It is
// nearly opaque: a pond barely ripples.
func swampWater(hx, hy int, u, v, s float64, up int) color.RGBA {
	// Below a bank on the north shore the mud drops a few pixels to the water.
	if up <= 3*groundDens {
		return swampMudHD.at(len(swampMudHD) - 1 - up + int(hashf(hx, hy, 157)*2))
	}
	if s > -0.08-(vnoise(u*4, v*4, 150)-0.5)*0.1 {
		return swampMudHD.g(0.35, hx, hy)
	}
	d := clamp01(-s / 3)
	i := pondHD.gi(0.85-d*0.75+(vnoise(u*0.8, v*0.9, 155)-0.5)*0.3, hx, hy)
	if up <= 3*groundDens+4 {
		i -= 3 // the bank's shadow on the water
	}
	edge := fbm2(u*1.6+3, v*1.9+5, 151, 2)
	if s > -0.55 && edge > 0.5 {
		clump := 0.45*hashf(fdiv(hx, 2), fdiv(hy, 2), 152) + 0.55*vnoise(float64(hx)/5, float64(hy)/4, 158)
		if clump < 0.3+(edge-0.5)*2.2 {
			return algaeHD.g(0.35+(edge-0.5)*1.2+(clump-0.3)*0.9, hx, hy)
		}
	}
	// Duckweed: small round leaves afloat.
	cx, cy := fdiv(hx, 11), fdiv(hy, 9)
	if h := hash2(int32(cx), int32(cy), 154); h&0xff < 26 {
		dx, dy := hx-(cx*11+2+int(h>>8&7)), hy-(cy*9+2+int(h>>11&3))
		switch {
		case (dx == 0 || dx == 1) && dy == 0:
			return algaeHD.at(len(algaeHD) - 1 - dx*2)
		case (dx == 0 || dx == 1) && dy == 1:
			return algaeHD.at(4)
		}
	}
	// The sky on still water: a few faint streaks.
	if hashf(fdiv(hx, 6), hy, 156) < 0.012 {
		i += 2
	}
	return withAlpha(pondHD.at(i), 232)
}

// marshGround colours a pixel of swamp mud: olive, blotched darker where it
// is wetter, with sedge growing on it and puddles ringed by brown mud.
func marshGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.35, v*0.45, 111, 3)
	blot := fbm2(u*0.55+3, v*0.7+7, 115, 3)
	l := 0.6 + (n-0.5)*0.6 + (vnoise(u*5.7, v*6.9, 116)-0.5)*0.1
	if blot < 0.45 {
		l -= (0.45 - blot) * 2.2 // wet, dark patches
	}
	// Puddles of standing water with a muddy rim.
	m := fbm2(u*1.3+5, v*1.7+9, 112, 2)
	switch {
	case m > 0.71:
		if hashf(fdiv(hx, 3), hy, 113) < 0.025 {
			return puddleHD.at(len(puddleHD) - 1 - int(hashf(hx, hy, 117)*3)) // the sky glinting on it
		}
		if m < 0.716 {
			return puddleHD.at(1) // dark under the rim
		}
		return puddleHD.g(0.15+(0.78-m)*6, hx, hy)
	case m > 0.68:
		c := swampMudHD.g(0.5+(m-0.68)*8, hx, hy)
		if hashf(hx, hy, 118) < 0.05 {
			c = swampMudHD.at(0)
		}
		return c
	case m > 0.662:
		l -= 0.28 * (m - 0.662) / 0.018 // the mud darkens towards a puddle
	}
	i := marshHD.gi(l, hx, hy)
	wet := clamp01((0.5 - blot) * 5)
	switch blade(hx, hy, 4, 8, 114, 0.4*(1-wet)) {
	case bladeFoot:
		i -= 3
	case bladeStem:
		i++
	case bladeTip:
		i += 3
	}
	switch tuft(hx, hy, 19, 15, 119, 0.22*(1-wet)) {
	case bladeFoot:
		i -= 4
	case bladeStem:
		i += 2
	case bladeTip:
		i += 5
	}
	if wet > 0.3 && hashf(hx, hy, 120) < 0.03 {
		i -= 3 // clods in the wet mud
	}
	return marshHD.at(i)
}

// --- the volcano ---------------------------------------------------------------

func ashGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.4, v*0.5, 121, 3)
	t := 0.45 + (n-0.5)*1.1 + (vnoise(u*5.9, v*7.3, 123)-0.5)*0.14
	// An ember now and then, glowing in the ash round it.
	cx, cy := fdiv(hx, 15), fdiv(hy, 13)
	if h := hash2(int32(cx), int32(cy), 122); h&0xff < 26 {
		dx, dy := hx-(cx*15+3+int(h>>8&0xff)*9/256), hy-(cy*13+3+int(h>>16&0xff)*7/256)
		switch d := dx*dx + dy*dy; {
		case d == 0:
			return hex(0xffd27a)
		case d == 1:
			return colEmber
		case d <= 5:
			return mix(ashHD.g(t, hx, hy), hex(0x9a2f10), 0.55-float64(d)*0.07)
		}
	}
	// Cracks in the crust.
	if e, _, nx, ny := plates(u*1.6, v*2.0, 125); vnoise(u*3.1, v*3.7, 126) > 0.42 {
		w := e * hdTW / 1.6
		switch {
		case w < 0.6:
			t -= 0.35
		case w < 1.6 && nx+ny < 0:
			t += 0.1
		}
	}
	i := ashHD.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 16, 13, 124, 0.14, 1, 2.6); p {
	case pebStone:
		return ashHD.g(0.05+l*0.75, hx, hy) // a cinder
	case pebShadow:
		return ashHD.at(i - 4)
	}
	switch h := hashf(hx, hy, 127); {
	case h < 0.03:
		i += 3 // a flake of pale ash
	case h < 0.06:
		i -= 2
	case h < 0.064:
		i += 6
	}
	return ashHD.at(i)
}

func lavaGround(hx, hy int, u, v float64) color.RGBA {
	// A crust of dark plates with glowing cracks between them.
	n := vnoise(u*2.2, v*2.6, 131)
	crack := math.Abs(n - 0.5)
	t := 1 - crack*4.5 + (vnoise(u*0.8, v*0.8, 132)-0.5)*0.5
	if t < 0.42 {
		// The crust itself: rough, and split by hair cracks that glow faintly.
		e, id, _, _ := plates(u*5.1, v*6.3, 133)
		w := e * hdTW / 5.1
		cool := clamp01((0.42 - t) * 4)
		switch {
		case w < 0.6:
			t = math.Max(t, 0.3+0.14*hashf(hx, hy, 134)-cool*0.12)
		default:
			t += (idf(id)-0.5)*0.1 - cool*0.05
			if hashf(hx, hy, 135) < 0.05 {
				t -= 0.08
			}
		}
	} else if hashf(hx, hy, 136) < 0.015 {
		t += 0.12 // a spark in the melt
	}
	return lavaHD.g(t, hx, hy)
}

// --- snow and ice ---------------------------------------------------------------

// snowGround colours a pixel of snow: soft drifts, blue in their hollows,
// with glints where the light catches.
func snowGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.25, v*0.32, 171, 3)
	t := 0.62 + (n-0.5)*0.8
	// Drifts: a bright crest and the blue shadow in its lee.
	sn, cs := math.Sincos((v*1.1 + u*0.25 + fbm2(u*0.35, v*0.45, 172, 2)*2.5) * 2 * math.Pi)
	if sn > 0.45 {
		k := (sn - 0.45) / 0.55
		if cs > 0 {
			t += 0.06 * k
		} else {
			t -= 0.2 * k * k * (0.6 + 0.4*vnoise(u*2.1, v*2.3, 176))
		}
	}
	t += (vnoise(u*4.1, v*5.3, 177) - 0.5) * 0.06
	p := clamp01(t)*float64(len(snowHD)-1) + (hashf(hx, hy, 900)-0.5)*0.9
	i := int(math.Round(p))
	// Sparkles: a point of light, some with rays.
	cx, cy := fdiv(hx, 19), fdiv(hy, 17)
	if h := hash2(int32(cx), int32(cy), 174); h&0xff < 40 {
		dx, dy := hx-(cx*19+2+int(h>>8&0xff)*15/256), hy-(cy*17+2+int(h>>16&0xff)*13/256)
		switch {
		case dx == 0 && dy == 0:
			return colWhite
		case h>>24&1 == 1 && dx*dx+dy*dy == 1:
			return snowHD.at(i + 3)
		}
	}
	switch h := hashf(hx, hy, 173); {
	case h < 0.006:
		return colWhite // a glint
	case h < 0.012:
		i -= 2 // a pit
	}
	return snowHD.at(i)
}

// iceGround is pond ice: pale and clear, split by white cracks, with a
// dusting of snow blown over it.
func iceGround(hx, hy int, u, v float64) color.RGBA {
	// Drifted snow, ragged at its edge.
	drift := fbm2(u*0.6+9, v*0.7, 193, 2) + (vnoise(float64(hx)/7, float64(hy)/6, 197)-0.5)*0.05 + (hashf(hx, hy, 198)-0.5)*0.02
	if drift > 0.62 {
		return snowGround(hx, hy, u, v)
	}
	n := fbm2(u*0.3, v*0.4, 191, 3)
	t := 0.45 + (n-0.5)*0.6 + (vnoise(u*3.1, v*3.7, 196)-0.5)*0.12
	if drift > 0.59 {
		t += 0.25 * (drift - 0.59) / 0.03 // frost at the snow's edge
	}
	e, id, nx, ny := plates(u*0.8, v*1.0, 192)
	t += (idf(id) - 0.5) * 0.08
	w := e * hdTW / 0.8
	if split := fbm2(u*0.9+3, v*1.1, 200, 2); split > 0.46 { // the cracks come and go
		switch {
		case w < 0.4+clamp01((split-0.46)*6)*0.5:
			return iceHD.at(len(iceHD) - 1) // a crack
		case w < 2.4:
			if nx+ny < 0 {
				t += 0.16 // the crack seen through the ice
			} else {
				t -= 0.06
			}
		}
	}
	// Hair cracks branching off.
	if math.Abs(vnoise(u*4.9, v*5.7, 195)-0.5) < 0.006 {
		t += 0.22
	}
	i := iceHD.gi(t, hx, hy)
	// Bubbles frozen in, and glints.
	switch h := hashf(hx, hy, 194); {
	case h < 0.004:
		return colWhite
	case h < 0.012:
		i += 3
	}
	// Scratches of light across the surface.
	if hashf(fdiv(hx+hy, 5), hx-hy, 199) < 0.012 {
		i += 2
	}
	return iceHD.at(i)
}

// --- rock ----------------------------------------------------------------------

func caveGround(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.5, v*0.6, 141, 3)
	t := 0.45 + (n-0.5)*1.1
	e, id, nx, ny := plates(u*1.7, v*2.2, 143)
	t += (idf(id) - 0.5) * 0.18
	w := e * hdTW / 1.7
	switch {
	case w < 0.7:
		t -= 0.4
	case w < 1.7 && nx+ny < 0:
		t += 0.12
	}
	i := caveHD.gi(t, hx, hy)
	switch p, l := pebble(hx, hy, 14, 12, 142, 0.16, 1, 2.6); p {
	case pebStone:
		return caveHD.at(i + 2 + int(l*7)) // a pale pebble
	case pebShadow:
		return caveHD.at(i - 4)
	}
	switch h := hashf(hx, hy, 144); {
	case h < 0.03:
		i += 2
	case h < 0.06:
		i -= 2
	}
	return caveHD.at(i)
}

func mountainTop(hx, hy int, u, v float64) color.RGBA {
	// A height field of ridges and crags, lit from the north-west.
	h := func(u, v float64) float64 {
		r := 1 - math.Abs(2*fbm2(u*1.6, v*2.0, 154, 3)-1)
		return fbm2(u*0.7, v*0.9, 151, 3) + 0.55*r*r
	}
	const e = 0.07
	c := h(u, v)
	l := 0.5 + (h(u-e, v-e)-c)*7 + (c-0.6)*0.6
	// Strata: fine breaks running across the rock.
	if s := vnoise(u*1.3, v*7.9, 155); math.Abs(s-0.5) < 0.012 {
		l -= 0.14
	}
	l += (vnoise(u*9.1, v*11.3, 156) - 0.5) * 0.16 // the rough of the rock
	if e, _, nx, ny := plates(u*2.6, v*3.3, 157); vnoise(u*1.7, v*1.9, 158) > 0.5 {
		switch w := e * hdTW / 2.6; { // clefts
		case w < 0.7:
			l -= 0.3
		case w < 1.7 && nx+ny < 0:
			l += 0.1
		}
	}
	switch k := hashf(hx, hy, 153); {
	case k < 0.04:
		l += 0.12
	case k < 0.08:
		l -= 0.1
	}
	return mtnHD.g(l, hx, hy)
}

// caveRockTop is the rock between the passages of the caves underground:
// black, with only a faint grain.
func caveRockTop(hx, hy int, u, v float64) color.RGBA {
	n := fbm2(u*0.8, v*1.0, 171, 2)
	l := 0.05 + (n-0.5)*0.12
	if hashf(hx, hy, 172) < 0.012 {
		l += 0.08
	}
	return caveDeepHD.g(l, hx, hy)
}

// faceColor is the rock face foot image pixels above its foot. Tunnel walls
// inside a cave are darker and wet, with crystals growing at their foot.
func faceColor(hx, hy, foot int, wall bool) color.RGBA {
	if top := rockDown - foot; top < groundDens {
		return mtnHD.at(len(mtnHD) - 3 - top*2) // the lit lip
	}
	f := float64(foot) / rockDown // 0 at the foot, 1 at the top
	fx, fy := float64(hx)/groundDens, float64(hy)/groundDens
	streak := vnoise(fx*0.45, fy*0.08, 161)*0.7 + vnoise(fx*1.4, fy*0.11, 164)*0.3
	l := 0.3 + f*0.6 + (streak-0.5)*0.6
	// Ledges: breaks across the face, lit on top and dark beneath.
	if d := vnoise(fx*0.09, fy*0.5, 165) - 0.5; math.Abs(d) < 0.03 {
		if d < 0 {
			l += 0.13
		} else {
			l -= 0.2
		}
	}
	// Cracks running down it.
	cx := fdiv(hx, 8)
	y := hy + int(hashf(cx, 5, 162)*17)
	cy := fdiv(y, 17)
	if h := hash2(int32(cx), int32(cy), 162); h&0xff < 46 {
		x := cx*8 + 2 + int(h>>8&3) + int(hashf(cx, fdiv(y, 4), 166)*2)
		long := 6 + int(h>>10&7)
		if y-cy*17 < long {
			switch hx - x {
			case 0:
				l -= 0.3
			case 1:
				l += 0.07
			}
		}
	}
	if !wall {
		return mtnFaceHD.g(l, hx, hy)
	}
	if c, ok := crystalPixel(hx, hy, foot); ok {
		return c
	}
	if hashf(hx, fdiv(hy, 4), 163) < 0.02 {
		return hex(0x3c5a66) // a wet glint
	}
	return caveWallHD.g(l, hx, hy)
}

// mouthPixel cuts the opening of a cave into the face above the mouth tile
// (tx, ty), foot image pixels above the face's foot. Neighbouring mouth
// tiles share one wide arch, framed in stones.
func (t *terrain) mouthPixel(hx, hy, tx, ty, foot int) (color.RGBA, bool) {
	mouth := func(x int) bool {
		return t.tile(x, ty) == sim.TileCave && t.tile(x, ty+1) != sim.TileCave && t.tile(x, ty+1).Walkable()
	}
	if !mouth(tx) {
		return color.RGBA{}, false
	}
	x0, x1 := tx, tx
	for mouth(x0 - 1) {
		x0--
	}
	for mouth(x1 + 1) {
		x1++
	}
	n := float64(x1 - x0 + 1)
	cx := (float64(x0) + n/2) * TileW
	hw := n*TileW/2 - 2
	ah := math.Min(mtnFaceH-5, 14+n*5)
	dx := ((float64(hx)+0.5)/groundDens - cx) / hw
	dy := (float64(foot) - 0.5) / groundDens / ah
	d := dx*dx + dy*dy
	switch {
	case d <= 1:
		// Inside: black, with a little light on the floor at the very front.
		if dy < 0.2 {
			return mix(hex(0x241d22), hex(0x050406), clamp01(dy/0.2+(hashf(hx, hy, 167)-0.5)*0.3)), true
		}
		return hex(0x050406), true
	case d <= 1.22:
		// A frame of paler stones around the opening.
		a := math.Atan2(dy, dx) / math.Pi * (5 + 3*n)
		st := a - math.Floor(a)
		if st < 0.1 || d < 1.02 {
			return mtnFaceHD.at(1), true // the joints, and the inner edge in shade
		}
		l := 0.5 + (hashf(int(math.Floor(a)), tx, 168)-0.5)*0.25 + (d-1.1)*1.2 - dx*0.1
		return mtnHD.g(l, hx, hy), true
	case d <= 1.4:
		return mtnFaceHD.g(0.08+(d-1.22)*1.2, hx, hy), true // and its shadow
	}
	return color.RGBA{}, false
}

// crystalPixel draws clusters of glowing crystals at the foot of tunnel
// walls. crystalX gives the same spots, for their light.
func crystalPixel(hx, hy, foot int) (color.RGBA, bool) {
	if foot > 11*groundDens {
		return color.RGBA{}, false
	}
	px := fdiv(hx, groundDens)
	cx, ok := crystalX(fdiv(px, TileW), fdiv(hy+foot, groundDens))
	if !ok {
		return color.RGBA{}, false
	}
	h := (float64(foot) - 0.5) / groundDens
	for i, sh := range [3][3]float64{{-3, 9, 1.6}, {0, 11, 2}, {3, 7, 1.4}} { // x offset, height, half-width
		x := (float64(hx)+0.5)/groundDens - (cx + sh[0])
		if h >= sh[1] {
			continue
		}
		w := sh[2]*(1-h/sh[1]) + 0.3
		if math.Abs(x) < w {
			l := 0.5 + 0.4*(1-h/sh[1]) + float64(i)*0.05
			switch {
			case math.Abs(x) < 0.2:
				l += 0.3 // the edge between its two faces
			case x < 0:
				l += 0.1
			default:
				l -= 0.22
			}
			if w-math.Abs(x) < 0.34 {
				l -= 0.15
			}
			return crystalHD.g(l, hx, hy), true
		}
	}
	return color.RGBA{}, false
}

// --- the sea -------------------------------------------------------------------

// seaCliff is the earth wall under a south-facing shore, up image pixels
// below the land's edge.
func seaCliff(hx, hy, up int) color.RGBA {
	const tall = cliffH * groundDens
	if up <= 2 {
		return cliffHD.at(len(cliffHD) - up) // the lit edge of the land
	}
	f := float64(up-1) / (tall - 1)
	streak := vnoise(float64(hx)*0.2, float64(hy)*0.085, 70)*0.7 + vnoise(float64(hx)*0.55, float64(hy)*0.1, 75)*0.3
	l := 0.92 - f*0.85 + (streak-0.5)*0.45
	// Layers of earth.
	if d := vnoise(float64(hx)*0.05, float64(hy)*0.6, 76) - 0.5; math.Abs(d) < 0.04 {
		l -= 0.12
	}
	// Stones in the earth.
	switch p, pl := pebble(hx, hy, 9, 7, 71, 0.16, 0.9, 1.7); p {
	case pebStone:
		return mix(cliffHD.g(l, hx, hy), warmStoneHD.at(2+int(pl*8)), 0.6)
	case pebShadow:
		l -= 0.2
	}
	if hashf(hx, fdiv(hy, 3), 77) < 0.05 {
		l -= 0.14 // a root, a crack
	}
	return cliffHD.g(l, hx, hy)
}

// shoreFoam is the foam along the water's edge at a sea pixel s (< 0)
// tiles out from the shore, if there is foam there.
func shoreFoam(hx, hy int, u, v, s float64) (color.RGBA, bool) {
	foamEdge := -0.1 - (vnoise(u*4, v*4, 72)-0.5)*0.12
	if s > foamEdge {
		return withAlpha(colFoam, 225), true
	}
	if lace := s - foamEdge; lace > -0.09 { // the foam thins out in lace
		if 0.5*hashf(fdiv(hx, 2), fdiv(hy, 2), 78)+0.5*vnoise(float64(hx)/4, float64(hy)/3, 79) < 0.75+lace*5.5 {
			return withAlpha(colFoam, uint8(200+lace*1100)), true
		}
	}
	return color.RGBA{}, false
}

// seaWater is the tint laid over the animated water at a pixel s (< 0)
// tiles out from the shore: foam along the edge, then turquoise shallows
// fading to deep navy.
func seaWater(hx, hy int, u, v, s float64) color.RGBA {
	if c, ok := shoreFoam(hx, hy, u, v, s); ok {
		return c
	}
	d := clamp01(-s / 4.5)
	p := math.Sqrt(d)*5 + (hashf(hx, hy, 82)-0.5)*0.12
	var a float64
	col := colShallow
	switch {
	case p < 1:
		a = lerpf(150, 70, p)
	case p < 2:
		a = lerpf(70, 0, p-1)
	case p < 3:
		col, a = colDeep, lerpf(0, 70, p-2)
	case p < 4:
		col, a = colDeep, lerpf(70, 130, p-3)
	default:
		col, a = colDeep, lerpf(130, 170, math.Min(p-4, 1))
	}
	return withAlpha(col, uint8(math.Round(a)))
}
