package client

import (
	"math"

	"survivalgame/internal/sim"
)

// The open world's biomes: their palettes, and the rock of the mountains,
// which stand up out of the ground with a face on their south side like the
// sea cliffs. How each kind of ground is drawn is in groundart.go.

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
	caveDeep    = mkRamp(0x020203, 0x07070a, 0x0d0d12, 0x15151b)
	pathRamp    = mkRamp(0x5a4128, 0x6e5233, 0x846440, 0x9a774e, 0xb08c60)
	iceRamp     = mkRamp(0x5a86a8, 0x76a2c2, 0x94bcd6, 0xb4d4e6, 0xd2e8f2, 0xf0faff)
)

// Mountains are drawn standing up: seen from above and to the south, each
// shows its flat top and, along its south edge, a tall face of rock that
// covers the southern part of its own tiles. Caves run under the mountains:
// the ground bake shows their floors with tunnel walls, and a separate
// "roof" bake draws them covered over, with a dark mouth where a tunnel
// comes out of a south face (see groundhd.go). The roof hides the caves until a player walks
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

// crystalX is where along tile column tx a crystal cluster grows at a wall
// whose foot is at world pixel row footY, if one does.
func crystalX(tx, footY int) (float64, bool) {
	ty := footY / TileH
	if hashf(tx, ty, 170) > 0.28 {
		return 0, false
	}
	return float64(tx*TileW) + 6 + hashf(tx, ty, 171)*12, true
}
