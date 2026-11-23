package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// Ground art for the open world's biomes: desert dunes, swamp mud, volcanic
// ash and lava, cave floors, and mountains, which stand up out of the ground
// with a rock face on their south side like the sea cliffs.

const mtnFaceH = 30 // art pixels of rock face along a mountain's south side

var (
	desertRamp  = mkRamp(0xb9783d, 0xcf8f4b, 0xe0a65c, 0xecbb72, 0xf5d090)
	marshRamp   = mkRamp(0x2f3a1c, 0x3e4b24, 0x4c5b2b, 0x5b6b32, 0x6b7b3a, 0x7d8c45)
	ashRamp2    = mkRamp(0x1f1d20, 0x2a272a, 0x363236, 0x433e41, 0x524c4e)
	lavaRamp    = mkRamp(0x5a120a, 0xa3240e, 0xe2481a, 0xff8a26, 0xffc85a, 0xfff2b0)
	caveRamp    = mkRamp(0x1a171b, 0x241f24, 0x2e282c, 0x3a3236, 0x463c3f)
	mtnRamp     = mkRamp(0x2f2c30, 0x423e42, 0x57525a, 0x6e6870, 0x8a8489, 0xaba5a6, 0xcdc8c4)
	mtnFace     = mkRamp(0x221e22, 0x2f2a2f, 0x3d373c, 0x4c454a, 0x5c545a, 0x6e666b)
	caveWall    = mkRamp(0x0e1014, 0x161a20, 0x1f252c, 0x2a323a, 0x36404a)
	crystalRamp = mkRamp(0x2b5a8a, 0x3f86c4, 0x62b8ea, 0x9ee2ff, 0xe8fbff)
	colEmber    = hex(0xff7a2a)
)

// openBiome colours a pixel of the open world's new ground kinds; ok is false
// for the kinds the island already had.
func openBiome(k sim.Tile, px, py int, u, v float64) (c color.RGBA, ok bool) {
	switch k {
	case sim.TileDesert:
		n := fbm2(u*0.2, v*0.28, 101, 3)
		i := desertRamp.idx(0.5+(n-0.5)*0.9, px, py)
		// Long dune ripples, bent by the wind.
		r := math.Sin((v*1.5 + u*0.35 + fbm2(u*0.3, v*0.4, 102, 2)*3) * 2 * math.Pi)
		switch {
		case r > 0.9:
			i -= 2
		case r > 0.75:
			i--
		case r < -0.93:
			i++
		}
		if hashf(px, py, 103) < 0.02 {
			i++
		}
		return desertRamp.at(i), true
	case sim.TileMarsh:
		return marshGround(px, py, u, v), true // see swampart.go
	case sim.TileAsh:
		n := fbm2(u*0.4, v*0.5, 121, 3)
		i := ashRamp2.idx(0.45+(n-0.5)*1.1, px, py)
		if hashf(px, py, 122) < 0.006 {
			return colEmber, true // a glowing ember
		}
		return ashRamp2.at(i), true
	case sim.TileLava:
		// A crust of dark plates with glowing cracks between them.
		n := vnoise(u*2.2, v*2.6, 131)
		crack := math.Abs(n - 0.5)
		t := 1 - crack*4.5 + (vnoise(u*0.8, v*0.8, 132)-0.5)*0.5
		return lavaRamp.pick(t, px, py), true
	case sim.TileCave:
		n := fbm2(u*0.5, v*0.6, 141, 3)
		i := caveRamp.idx(0.45+(n-0.5)*1.1, px, py)
		if hashf(px/2, py, 142) < 0.04 {
			i += 2 // a pale pebble
		}
		return caveRamp.at(i), true
	case sim.TileMountain:
		return mountainTop(px, py, u, v), true
	case sim.TileSnow:
		return snowGround(px, py, u, v), true // see iceart.go
	}
	return color.RGBA{}, false
}

func mountainTop(px, py int, u, v float64) color.RGBA {
	// A height field of ridges and crags, lit from the north-west.
	h := func(u, v float64) float64 {
		r := 1 - math.Abs(2*fbm2(u*1.6, v*2.0, 154, 3)-1)
		return fbm2(u*0.7, v*0.9, 151, 3) + 0.55*r*r
	}
	const e = 0.07
	c := h(u, v)
	l := 0.5 + (h(u-e, v-e)-c)*7 + (c-0.6)*0.6
	if hashf(px, py, 153) < 0.02 {
		l += 0.2
	}
	return mtnRamp.pick(l, px, py)
}

// Mountains are drawn standing up: seen from above and to the south, each
// shows its flat top and, along its south edge, a tall face of rock that
// covers the southern part of its own tiles. Caves run under the mountains:
// the ground bake shows their floors with tunnel walls, and a separate
// "roof" bake draws them covered over, with a dark mouth where a tunnel
// comes out of a south face. The roof hides the caves until a player walks
// in (see caves.go).

// solidAt reports whether tile (tx, ty) is rock: a mountain, or with roof
// also a cave under its roof.
func (t *terrain) solidAt(tx, ty int, roof bool) bool {
	k := t.tile(tx, ty)
	return k == sim.TileMountain || (roof && k == sim.TileCave)
}

// isMountain is the smooth rock field at (u, v), thresholded.
func (t *terrain) isMountain(u, v float64, roof bool) bool {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := smooth(x-x0), smooth(y-y0)
	m := func(tx, ty int) float64 {
		if t.solidAt(tx, ty, roof) {
			return 1
		}
		return 0
	}
	f := lerpf(lerpf(m(ix, iy), m(ix+1, iy), fx), lerpf(m(ix, iy+1), m(ix+1, iy+1), fx), fy)
	if f == 0 {
		return false
	}
	if f == 1 {
		return true
	}
	return f+(fbm2(u*1.1, v*1.3, 160, 2)-0.5)*0.35 > 0.5
}

// mountainNear reports whether rock could show at (u, v): rock on a nearby
// tile, including the tiles below, whose faces reach up into this one.
func (t *terrain) mountainNear(u, v float64, roof bool) bool {
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	for y := ty - 1; y <= ty+1; y++ {
		for x := tx - 1; x <= tx+1; x++ {
			if t.solidAt(x, y, roof) {
				return true
			}
		}
	}
	return false
}

// caveNear reports whether a cave tile is near enough to (u, v) for the
// roof to look different from the open ground there.
func (t *terrain) caveNear(u, v float64) bool {
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	for y := ty - 1; y <= ty+3; y++ {
		for x := tx - 1; x <= tx+1; x++ {
			if t.tile(x, y) == sim.TileCave {
				return true
			}
		}
	}
	return false
}

// mountainPixel colours a pixel near rock: its top or its south face (ok),
// or for open ground (!ok) how much it lies in the shadow at the foot of a
// face. With roof, caves count as rock and their mouths are cut into faces.
func (t *terrain) mountainPixel(px, py int, u, v float64, roof bool) (c color.RGBA, shadow float64, ok bool) {
	if !t.mountainNear(u, v, roof) {
		return color.RGBA{}, 0, false
	}
	if !t.isMountain(u, v, roof) {
		for k := 1; k <= 3; k++ {
			if t.isMountain(u, v-float64(k)/TileH, roof) {
				return color.RGBA{}, 0.32 - float64(k-1)*0.1, false
			}
		}
		return color.RGBA{}, 0, false
	}
	// How high above the bottom edge of the rock this pixel is.
	for k := 1; k <= mtnFaceH; k++ {
		bu, bv := u, v+float64(k)/TileH
		if t.isMountain(bu, bv, roof) {
			continue
		}
		// The face. What lies at its foot decides what it looks like.
		if roof {
			// The face's foot wanders a little either side of the tile
			// edge, so the mouth tile is the one at the foot or above it.
			ftx, fty := int(math.Floor(u)), int(math.Floor(v+(float64(k)-0.5)/TileH))
			for _, ty := range []int{fty, fty - 1} {
				if c, ok := t.mouthPixel(px, ftx, ty, k); ok {
					return c, 0, true
				}
			}
		}
		wall := t.tile(int(math.Floor(bu)), int(math.Floor(bv))) == sim.TileCave
		return faceColor(px, py, k, wall), 0, true
	}
	return mountainTop(px, py, u, v), 0, true
}

// faceColor is the rock face k pixels above its foot. Tunnel walls inside a
// cave are darker and wet, with crystals growing at their foot.
func faceColor(px, py, k int, wall bool) color.RGBA {
	if k >= mtnFaceH-1 {
		return mtnRamp.at(len(mtnRamp) - 2) // the lit lip
	}
	f := float64(k) / mtnFaceH // 0 at the foot, 1 at the top
	streak := vnoise(float64(px)*0.45, float64(py)*0.08, 161)
	l := 0.3 + f*0.6 + (streak-0.5)*0.55
	if hashf(px/2, py/4, 162) < 0.08 {
		l -= 0.3 // a crack
	}
	if !wall {
		return mtnFace.pick(l, px, py)
	}
	if c, ok := crystalPixel(px, py, k); ok {
		return c
	}
	if hashf(px, py/2, 163) < 0.05 {
		return hex(0x3c5a66) // a wet glint
	}
	return caveWall.pick(l, px, py)
}

// mouthPixel cuts the opening of a cave into the face above the mouth tile
// (tx, ty), k pixels above the face's foot. Neighbouring mouth tiles share
// one wide arch.
func (t *terrain) mouthPixel(px, tx, ty, k int) (color.RGBA, bool) {
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
	dx := (float64(px) + 0.5 - cx) / hw
	dy := (float64(k) - 0.5) / ah
	d := dx*dx + dy*dy
	switch {
	case d <= 1:
		// Inside: black, with a little light on the floor at the very front.
		if dy < 0.18 {
			return hex(0x1c171c), true
		}
		return hex(0x050406), true
	case d <= 1.22:
		return mtnRamp.at(3), true // a frame of paler stones around the opening
	case d <= 1.4:
		return mtnFace.at(0), true // and its shadow
	}
	return color.RGBA{}, false
}

// crystalPixel draws clusters of glowing crystals at the foot of tunnel
// walls. crystalAt gives the same spots, for their light.
func crystalPixel(px, py, k int) (color.RGBA, bool) {
	if k > 11 {
		return color.RGBA{}, false
	}
	tx := int(math.Floor(float64(px) / TileW))
	cx, ok := crystalX(tx, py+k)
	if !ok {
		return color.RGBA{}, false
	}
	for i, sh := range [3][3]float64{{-3, 9, 1.6}, {0, 11, 2}, {3, 7, 1.4}} { // x offset, height, half-width
		x := float64(px) + 0.5 - (cx + sh[0])
		h := float64(k) - 0.5
		if h < sh[1] && math.Abs(x) < sh[2]*(1-h/sh[1])+0.4 {
			l := 0.55 + 0.45*(1-h/sh[1]) - x*0.15
			return crystalRamp.pick(l+float64(i)*0.05, px, py), true
		}
	}
	return color.RGBA{}, false
}

// crystalX is where along tile column tx a crystal cluster grows at a wall
// whose foot is at world pixel row footY, if one does.
func crystalX(tx, footY int) (float64, bool) {
	ty := footY / TileH
	if hashf(tx, ty, 170) > 0.28 {
		return 0, false
	}
	return float64(tx*TileW) + 6 + hashf(tx, ty, 171)*12, true
}

// roofPixel is how (px, py) looks with the caves roofed over, if that
// differs from ground, the pixel without the roof.
func (t *terrain) roofPixel(px, py int, ground color.RGBA) (color.RGBA, bool) {
	u, v := (float64(px)+0.5)/TileW, (float64(py)+0.5)/TileH
	if !t.caveNear(u, v) {
		return color.RGBA{}, false
	}
	c, shadow, ok := t.mountainPixel(px, py, u, v, true)
	switch {
	case ok:
	case shadow > 0:
		c = mix(ground, colInk, shadow)
	default:
		return color.RGBA{}, false
	}
	return c, c != ground
}
