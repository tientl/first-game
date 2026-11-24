package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// A warmer ground: ochre earth with grass growing over it in patches, the
// edges of each patch ragged with blades, a scatter of pebbles and cracks in
// the bare earth, all in the warm light of a late afternoon. A trial look,
// off unless SetWarmGround turns it on.

var warmGround bool

// SetWarmGround turns the warm ground on or off. Call it before the world
// is shown: the ground is baked when the world arrives.
func SetWarmGround(on bool) { warmGround = on }

var (
	warmDirtRamp  = mkRamp(0x5a3a22, 0x74492a, 0x8e5c33, 0xa8733f, 0xbf8a4c, 0xd4a35f)
	warmGrassRamp = mkRamp(0x3b4e22, 0x4e6229, 0x627631, 0x77893a, 0x8d9b46, 0xa4ad57)
	warmMossRamp  = mkRamp(0x2c4220, 0x3a5426, 0x4b672c, 0x5e7b34, 0x74903f)
	warmStone     = mkRamp(0x5e5650, 0x7a706a, 0x968a80, 0xb2a698)
)

// warmBiome colours a pixel of grass or wood floor in the warm look.
func warmBiome(k sim.Tile, px, py int, u, v float64) (color.RGBA, bool) {
	if !warmGround || (k != sim.TileGrass && k != sim.TileForest) {
		return color.RGBA{}, false
	}
	forest := k == sim.TileForest
	// How grassy this spot is: big soft patches, broken up finer.
	g := fbm2(u*0.16, v*0.21, 701, 3)*0.75 + vnoise(u*0.9, v*1.15, 702)*0.25
	cut := 0.5
	if forest {
		cut = 0.36 // the woods are mossier
	}
	// Blades: along a patch's edge, single pixels of grass poke out over the
	// earth and earth shows through the grass, in short vertical strokes.
	edge := g - cut
	blade := (hashf(px, py, 703) - 0.5) * 0.1
	if hashf(px, py/2, 704) < 0.5 {
		blade += (hashf(px, py+1, 703) - 0.5) * 0.06
	}
	grassy := edge+blade > 0
	if grassy {
		rp := warmGrassRamp
		if forest {
			rp = warmMossRamp
		}
		n := fbm2(u*0.35, v*0.45, 705, 2)
		l := 0.5 + (n-0.5)*0.9 + math.Min(edge, 0.15)*1.5
		i := rp.idx(l, px, py)
		// Dark blades with a lit tip, as on the old grass.
		const p = 0.06
		switch {
		case hashf(px, py, 706) < p || hashf(px, py-1, 706) < p:
			i--
		case hashf(px, py+1, 706) < p:
			i++
		}
		if edge < 0.03 {
			i-- // the thin, sparse grass at a patch's rim is darker
		}
		return rp.at(i), true
	}
	// Bare earth.
	n := fbm2(u*0.3, v*0.38, 707, 3)
	i := warmDirtRamp.idx(0.55+(n-0.5)*0.8+edge*0.6, px, py)
	// Pebbles: one per small cell, lit on top with a shadow under it.
	cx, cy := px/6, py/5
	ox, oy := cx*6+1+int(hashf(cx, cy, 708)*3), cy*5+1+int(hashf(cx, cy, 709)*2)
	if hashf(cx, cy, 710) < 0.18 {
		switch {
		case (px == ox || px == ox+1) && py == oy:
			return warmStone.at(3), true
		case (px == ox || px == ox+1) && py == oy+1:
			return warmStone.at(1), true
		case px == ox+2 && py == oy+1:
			return warmDirtRamp.at(0), true
		}
	}
	// Dry cracks.
	if math.Abs(vnoise(u*1.9, v*2.4, 711)-0.5) < 0.016 && fbm2(u*0.5, v*0.5, 712, 2) > 0.5 {
		return warmDirtRamp.at(0), true
	}
	// Speckle.
	if h := hashf(px, py, 713); h < 0.04 {
		i += []int{-1, 1}[int(h*100)%2]
	}
	return warmDirtRamp.at(i), true
}
