package client

import (
	"image/color"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The body a player leaves where they fell (sim/grave.go), and the bones it
// has gone to once they have died again without fetching their things. Both
// are the player themselves: the body is their own figure laid on its back,
// head to the left, one arm thrown up past the head, in a small dark stain;
// the bones lie the same way, with rags of the tunic in the player's colour
// and a few strands of their hair. Beside the shins is the bag of what they
// carried, an axe and a coin or two spilling out of it: that is what says
// there are things to take here.
//
// The figure is drawn standing, in the frame of look.go, and laid down by
// swapping x and y: sprite row y becomes column fallenOX+y, column x becomes
// row fallenOY+x. The side the light fell on standing is the upper side lying.

const (
	fallenW, fallenH   = 40, 32
	fallenOX, fallenOY = 2, 6
	fallenAX, fallenAY = 19, 20 // the anchor: the middle of the tile
	fallenBagX         = fallenOX + 22
	fallenBagY         = fallenOY + 11 // the bag's corner: in front of the shins
)

// fallenKey names one drawing: whose body, in which colour, how far gone.
type fallenKey struct {
	look  sim.Look
	col   int
	bones bool
}

// fallenCache holds the bodies drawn so far.
var fallenCache = map[fallenKey]sprite{}

// graveSprite is the sprite of the body e. It must be called inside the game
// loop, where images can be created.
func graveSprite(e protocol.EntityView) sprite {
	look, seed := sim.GraveLook(e.Fuel)
	k := fallenKey{look, seed % len(playerColors), e.Var == sim.GraveBones}
	s, ok := fallenCache[k]
	if !ok {
		s = mk(genFallen(k.look, playerColors[k.col], k.bones), fallenAX, fallenAY)
		fallenCache[k] = s
	}
	return s
}

// graveLabel names the body e for the hint over it.
func graveLabel(e protocol.EntityView) string {
	if e.Var == sim.GraveBones {
		return "Bộ xương — lấy lại đồ"
	}
	return e.Kind.Name() + " — lấy lại đồ"
}

// graveShadow is shadowWidth for a body: it lies flat, so only the bag
// throws a little shade, and the art carries that itself.
func graveShadow(e protocol.EntityView) (float64, bool) {
	return 0, e.Kind == sim.EntityGrave
}

// tinted is c in the player's colour, as drawing it with a colour scale would
// make it.
func tinted(c, tint color.RGBA) color.RGBA {
	m := func(a, b uint8) uint8 { return uint8(uint32(a) * uint32(b) / 255) }
	return color.RGBA{m(c.R, tint.R), m(c.G, tint.G), m(c.B, tint.B), c.A}
}

// genFallen draws the body of a player of look l who wore tint, or their
// bones, with their bag.
func genFallen(l sim.Look, tint color.RGBA, bones bool) *canvas {
	l = l.Sanitized()
	c := newCanvas(fallenW, fallenH)
	if bones {
		c.over(fallenBones(l, tint), 0, 0)
	} else {
		fallenStain(c)
		c.over(fallenBody(l, tint), 0, 0)
	}
	c.over(genLootBag(), fallenBagX, fallenBagY)
	return c
}

// fallenStain is the blood under a fresh body: a dark pool spreading from
// under the chest, more of it on the near side.
func fallenStain(c *canvas) {
	cx, cy := float64(fallenOX)+13, float64(fallenOY)+13.5
	c.ellipse(cx, cy, 13, 6.5, func(x, y int, nx, ny float64) color.RGBA {
		r := nx*nx + ny*ny
		if r > 0.55+0.45*vnoise(float64(x)/2.6, float64(y)/2.2, 71) {
			return color.RGBA{}
		}
		return bloodRamp.pick(0.5-r*0.35+(hashf(x, y, 9)-0.5)*0.15, x, y)
	})
	for _, p := range [][2]int{{-1, 17}, {20, 21}, {4, 21}} { // spatters
		c.put(fallenOX+p[0], fallenOY+p[1], bloodRamp.at(0))
	}
}

// lay lays the standing layer src down on fig, in the player's colour if
// tint is not nil; keep, if not nil, picks the pixels and may change them.
func lay(fig, src *canvas, tint *color.RGBA, keep func(x, y int, p color.RGBA) (color.RGBA, bool)) {
	for y := 0; y < src.h; y++ {
		for x := 0; x < src.w; x++ {
			p := src.at(x, y)
			if p.A < 128 {
				continue
			}
			if keep != nil {
				var ok bool
				if p, ok = keep(x, y, p); !ok {
					continue
				}
			}
			if tint != nil {
				p = tinted(p, *tint)
			}
			fig.put(fallenOX+y, fallenOY+x, p)
		}
	}
}

// fallenBody is the player lying on their back, eyes shut, the arm on the
// far side thrown up past the head.
func fallenBody(l sim.Look, tint color.RGBA) *canvas {
	B, L, S, T := playerLayersPose(l, sim.DirDown, 1, true, armOff)
	fig := newCanvas(fallenW, fallenH)
	lay(fig, B, nil, nil)
	lay(fig, L, nil, nil)
	lay(fig, S, &tint, nil)
	lay(fig, T, nil, nil)

	// The arm: from the shoulder up the screen, bending towards the head, the
	// open hand beyond it. Its rim is drawn here, as the layers' was.
	sk := skinTones[l.Skin].ramp
	y0 := 3 // the shoulder's row before the offset: the arm's column standing
	if l.Female {
		y0 = 4
	}
	type px struct {
		x, y int
		col  color.RGBA
	}
	var arm []px
	sleeve := func(x, y, a, b int) {
		arm = append(arm,
			px{x, y, tinted(shirtRamp.at(a), tint)},
			px{x + 1, y, tinted(shirtRamp.at(b), tint)})
	}
	sleeve(16, y0, 3, 2)
	sleeve(16, y0-1, 3, 2)
	sleeve(15, y0-2, 3, 1)
	sleeve(15, y0-3, 3, 1)
	sleeve(14, y0-4, 2, 1)
	if l.Female {
		sleeve(14, y0-5, 3, 2)
	} else {
		sleeve(14, y0-5, 1, 1) // the cuff
	}
	arm = append(arm,
		px{13, y0 - 6, sk.at(3)}, px{14, y0 - 6, sk.at(2)},
		px{12, y0 - 7, sk.at(3)}, px{13, y0 - 7, sk.at(2)}, px{14, y0 - 7, sk.at(1)})
	on := make(map[[2]int]bool, len(arm))
	for _, p := range arm {
		on[[2]int{fallenOX + p.x, fallenOY + p.y}] = true
	}
	for _, p := range arm {
		x, y := fallenOX+p.x, fallenOY+p.y
		for _, d := range [4][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} {
			if nx, ny := x+d[0], y+d[1]; !on[[2]int{nx, ny}] && !fig.opaque(nx, ny) {
				fig.put(nx, ny, mix(p.col, colInk, 0.72))
			}
		}
	}
	for _, p := range arm {
		fig.put(fallenOX+p.x, fallenOY+p.y, p.col)
	}
	return fig
}

// The skeleton, standing, in the frame of the player sprite:
//
//	W w b B  bone, light to dark      k  a hollow
var (
	boneSkull = pixMap{5, []string{
		"......wWWWwb......",
		".....wWWWWwwb.....",
		"....wWWWWwwwwb....",
		"....WWWWwwwwwb....",
		"....Wkkkwwkkkb....",
		"....WkkkwwkkkB....",
		"....wwkkwwkkbB....",
		".....wwwkkwwb.....",
		".....bwwwwwbB.....",
		"......wkwkwk......",
		"......bwwwbB......",
	}}
	// The collar bones and ribs on the spine, the hips, two short legs and
	// the arm that lies along the body.
	boneFrame = pixMap{16, []string{
		".....wWWwwwwbb....",
		"........wb...b....",
		".....WWwwwwwb.b...",
		"........wb....b...",
		"......Wwwwwb..b...",
		"........wb....w...",
		"......WwwwwB..wb..",
		"......wb..wb......",
		"......Wb..wb......",
		".....wWb..wwb.....",
		"......wb..wb......",
		"....wWWb..wwwb....",
	}}
)

var (
	boneHollow = hex(0x2b2230)
	hairRot    = hex(0x5c544c)
)

func bonePalette(ch byte) (color.RGBA, bool) {
	switch ch {
	case 'W':
		return ivoryRamp[3], true
	case 'w':
		return ivoryRamp[2], true
	case 'b':
		return ivoryRamp[1], true
	case 'B':
		return ivoryRamp[0], true
	case 'k':
		return boneHollow, true
	}
	return color.RGBA{}, false
}

// fallenBones is what is left of a player of look l who wore tint: their
// skeleton lying as the body lay, in rags of their clothes, with what is
// left of their hair.
func fallenBones(l sim.Look, tint color.RGBA) *canvas {
	seed := uint32(l.Hair)*7 + uint32(l.HairColor)*31 + uint32(l.Skin)*131 + 977
	if l.Female {
		seed += 4001
	}
	bone := newCanvas(bodyW, bodyH)
	boneSkull.stamp(bone, bonePalette)
	boneFrame.stamp(bone, bonePalette)

	fig := newCanvas(fallenW, fallenH)
	lay(fig, bone, nil, nil)
	// The arm thrown up past the head, as on the body.
	for i, p := range [][2]int{{16, 4}, {16, 3}, {15, 2}, {15, 1}, {14, 0}, {14, -1}, {13, -2}, {14, -2}, {12, -3}, {14, -3}} {
		tone := 2 + (i+1)%2
		if i >= 8 {
			tone = 2 // the fingers
		}
		fig.put(fallenOX+p[0], fallenOY+p[1], ivoryRamp.at(tone))
	}

	// Rags: patches of the clothes, faded and dirty, lying over the bones.
	B, _, S, T := playerLayersPose(l, sim.DirDown, 1, true, armOff)
	isCloth := func(p color.RGBA) bool {
		for _, c := range shirtRamp {
			if p == c {
				return true
			}
		}
		return false
	}
	lay(fig, S, &tint, func(x, y int, p color.RGBA) (color.RGBA, bool) {
		if !isCloth(p) || y < 17 {
			return p, false
		}
		n := vnoise(float64(x)/2.3, float64(y)/2.3, seed)
		if n < 0.52 || hashf(x, y, seed+1) < 0.12 {
			return p, false
		}
		return mix(p, hex(0x4a4a4a), 0.35), true
	})
	fig.put(fallenOX+14, fallenOY+0, tinted(mix(shirtRamp[2], hex(0x4a4a4a), 0.35), tint)) // a shred of the sleeve
	fig.put(fallenOX+15, fallenOY+1, tinted(mix(shirtRamp[1], hex(0x4a4a4a), 0.35), tint))

	// What the clothes were held with: a man's belt, a woman's brooches.
	if l.Female {
		for _, x := range []int{6, 11} {
			fig.put(fallenOX+17, fallenOY+x, broochRamp[3])
			fig.put(fallenOX+18, fallenOY+x, broochRamp[1])
		}
	} else {
		for x := 5; x <= 11; x++ {
			if x != 7 {
				fig.put(fallenOX+21, fallenOY+x, leatherRamp.at(2-(x-5)/3))
			}
		}
		fig.put(fallenOX+21, fallenOY+9, broochRamp[2])
	}

	// Hair: strands of it still on the crown and trailing where it hung.
	ha := hairArts[l.Hair]
	strand := func(x, y int, p color.RGBA) (color.RGBA, bool) {
		if !ha.front.has(x, y) && !ha.behind.has(x, y) {
			return p, false
		}
		crown := y <= 5 && hashf(x, y, seed+2) < 0.7
		lock := (x <= 3 || x >= 14) && hashf(x, 0, seed+3) < 0.5 && hashf(x, y/3, seed+4) < 0.6
		if !crown && !lock {
			return p, false
		}
		return mix(p, hairRot, 0.3), true
	}
	isHair := func(p color.RGBA) bool {
		for _, c := range hairColors[l.HairColor].ramp {
			if p == c {
				return true
			}
		}
		return false
	}
	for _, src := range []*canvas{B, T} {
		lay(fig, src, nil, func(x, y int, p color.RGBA) (color.RGBA, bool) {
			if !isHair(p) {
				return p, false
			}
			return strand(x, y, p)
		})
	}
	fig.outline(0.78)
	return fig
}

// genLootBag draws the bag of a fallen player's things: a leather sack tied
// at the neck, an axe stuck in it, coins spilt on the ground in front.
func genLootBag() *canvas {
	c := newCanvas(15, 14)
	// The axe: its haft leaning out of the sack, the head up and to the right.
	c.line(6, 7, 11, 2, func(x, y int, t float64) color.RGBA { return leatherRamp.at(3 - (x+y)%2) })
	for _, p := range [][3]int{{10, 1, 1}, {10, 2, 2}, {11, 0, 2}, {11, 1, 3}, {11, 2, 2}, {11, 3, 1}, {12, 0, 4}, {12, 1, 4}, {12, 2, 3}, {12, 3, 3}, {12, 4, 2}, {13, 1, 4}, {13, 2, 4}, {13, 3, 3}} {
		c.put(p[0], p[1], stoneRamp.at(p[2])) // the blade, its edge the brightest
	}
	// The sack.
	c.ellipse(5.5, 8.5, 4.6, 3.6, func(x, y int, nx, ny float64) color.RGBA {
		return leatherRamp.pick(lum(nx*0.9, ny*0.9)*1.05-0.08, x, y)
	})
	c.rect(4, 4, 3, 2, func(x, y int) color.RGBA { return leatherRamp.at(3 - (x - 4)) }) // the gathered neck
	c.put(3, 3, leatherRamp.at(3))
	c.put(5, 3, leatherRamp.at(2))
	c.put(7, 3, leatherRamp.at(1))
	c.rect(4, 5, 3, 1, solid(wrapRamp.at(3))) // the cord
	c.put(7, 6, wrapRamp.at(2))
	c.put(4, 8, leatherRamp.at(3)) // a shine on the bulge
	c.put(3, 9, leatherRamp.at(3))
	// Coins.
	c.put(10, 11, broochRamp[3])
	c.put(11, 11, broochRamp[2])
	c.put(12, 10, broochRamp[3])
	c.put(13, 10, broochRamp[1])
	c.put(8, 12, broochRamp[2])
	c.outline(0.78)
	return c
}
