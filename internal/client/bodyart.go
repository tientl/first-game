package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The body a player leaves where they fell (sim/grave.go), and the bones it
// has gone to once they have died again without fetching their things. Both
// are the player themselves: the body is their own figure (the jointed one
// of limbs.go, in the furs or the clothes they had on) laid on its back,
// head to the left, one arm thrown up past the head, in a small dark stain;
// the bones lie the same way, with rags of what they wore (the tunic in the
// player's colour, or the fur) and a few strands of their hair. Beside the
// shins is the bag of what they carried, an axe and a coin or two spilling
// out of it: that is what says there are things to take here.
//
// The figure is drawn standing and laid down by swapping x and y: a row
// becomes a column, a column a row. The side the light fell on standing is
// the upper side lying. The whole drawing is at HDDensity: the body and
// the bones are drawn so, the stain and the bag are refined by upscale3.

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
	wear  sim.Wear
	col   int
	bones bool
}

// fallenCache holds the bodies drawn so far.
var fallenCache = map[fallenKey]sprite{}

// graveSprite is the sprite of the body e. It must be called inside the game
// loop, where images can be created.
func graveSprite(e protocol.EntityView) sprite {
	look, seed := sim.GraveLook(e.Fuel)
	k := fallenKey{look, sim.GraveWear(e.Fuel), seed % len(playerColors), e.Var == sim.GraveBones}
	s, ok := fallenCache[k]
	if !ok {
		s = spriteHD(genFallen(k.look, k.wear, playerColors[k.col], k.bones).image(), fallenAX, fallenAY)
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

// genFallen draws the body of a player of look l who had wear on, in the
// colour tint, or their bones, with their bag: fallenW by fallenH art
// pixels at HDDensity.
func genFallen(l sim.Look, wear sim.Wear, tint color.RGBA, bones bool) *canvas {
	l = l.Sanitized()
	c := newCanvas(fallenW, fallenH)
	if !bones {
		fallenStain(c)
	}
	hd := upscale3(c)
	if bones {
		fallenBones(hd, l, wear, tint)
	} else {
		fallenBody(hd, l, wear, tint)
	}
	bag := newCanvas(fallenW, fallenH)
	bag.over(genLootBag(), fallenBagX, fallenBagY)
	hd.over(upscale3(bag), 0, 0)
	return hd
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

// fallenTop is the row of the standing figure that comes to fallenOX lying:
// a little above the crown.
const fallenTop = 18

// layDown lays src, a layer of the jointed figure (limbs.go) standing, down
// on dst, an HD canvas the size of a body's frame: a row becomes a column, a
// column a row, the crown to the left and the figure's middle on the row the
// stain is drawn round. In the player's colour if tint is not nil.
func layDown(dst, src *canvas, tint *color.RGBA) {
	for y := 0; y < src.h; y++ {
		for x := 0; x < src.w; x++ {
			p := src.px[y*src.w+x]
			if p.A < 128 {
				continue
			}
			if tint != nil {
				p = tinted(p, *tint)
			}
			dst.put(fallenOX*HDDensity+y-fallenTop, fallenOY*HDDensity+x-limbPad*HDDensity, p)
		}
	}
}

// fallenBody lays the player on their back on dst, an HD canvas: the
// jointed figure as it stood, in what it wore, eyes shut, the arm on the far
// side thrown up past the head, its hand fallen open.
func fallenBody(dst *canvas, l sim.Look, wear sim.Wear, tint color.RGBA) {
	up := bakeUpper(l, wear, upKey{dir: sim.DirDown, anim: upFallen}, headCache{})
	lo, _ := bakeLower(l, wear, loKey{dir: sim.DirDown})
	layDown(dst, up.layer[upFar], &tint)
	layDown(dst, up.layer[upBack], nil)
	layDown(dst, lo, nil)
	layDown(dst, up.layer[upBody], nil)
	layDown(dst, up.layer[upShirt], &tint)
	layDown(dst, up.layer[upArms], &tint)
	layDown(dst, up.layer[upTop], nil)
	layDown(dst, up.layer[upArmsOver], &tint)
	layDown(dst, up.layer[upTopOver], nil)
}

var (
	skelHD     = ext6(ivoryRamp)
	boneHollow = hex(0x2b2230)
	hairRot    = hex(0x5c544c)
	dirt       = hex(0x4a4a4a)
)

// fallenBones lays on dst, an HD canvas, what is left of a player of look l
// who had wear on, in the colour tint: their skeleton lying as the body lay,
// drawn to the same measure, with rags of what they wore over it (the fur
// about the hips, or the tunic in the player's colour, and boots if they had
// them) and strands of their hair about the skull.
func fallenBones(dst *canvas, l sim.Look, wear sim.Wear, tint color.RGBA) {
	seed := uint32(l.Hair)*7 + uint32(l.HairColor)*31 + uint32(l.Skin)*131 + 977
	female := l.Female
	if female {
		seed += 4001
	}
	g := newLimbGen(l, wear)
	f := &g.f
	B, T := g.B, g.T // the bones, and the rags over them
	bone := func(s shape, round, bias float64) { f.fill(B, s, bevel(skelHD, s, round, bias)) }
	hollow := func(s shape) { f.fill(B, s, plain(boneHollow)) }
	// limb is a long bone from (x0, y0) to (x1, y1), r thick, with a knob
	// at each end.
	limb := func(x0, y0, x1, y1, r, k0, k1 float64) {
		bone(sCap(x0, y0, x1, y1, r, r*0.9).or(sEll(x0, y0, k0, k0)).or(sEll(x1, y1, k1, k1)), r+0.6, 0.04)
	}
	k := 1.0
	if female {
		k = 0.9
	}

	// Hair, still about the crown: locks of it lying spread behind the
	// skull, matted and dulled, long or short as it was worn.
	hr := hairColors[l.HairColor].ramp
	long, n := 3.5, 5
	switch l.Hair {
	case sim.HairLong, sim.HairShaggy, sim.HairPonytail, sim.HairBob:
		long, n = 8.5, 8
	case sim.HairSpiky:
		long, n = 2.5, 4
	}
	for j := 0; j < n; j++ {
		u := (float64(j)+0.5)/float64(n)*2 - 1 // across the crown, -1 to 1
		a := u*0.75 + (hashf(j, 3, seed)-0.5)*0.25
		x0, y0 := 27+math.Sin(a)*3, 34.4-math.Cos(a)*3.4
		ln := 3.6 + long*(0.55+0.5*hashf(j, 1, seed))
		bend := (hashf(j, 2, seed) - 0.5) * 3.2
		x1, y1 := x0+math.Sin(a)*ln*1.2+bend, y0-math.Cos(a)*ln
		s := sLock(x0, y0, (x0+x1)/2+bend, (y0+y1)/2, x1, y1, 2.3, 0.7)
		f.fill(B, s, func(x, y int, _ float64) color.RGBA {
			i := int(s.lum(x, y, 1.2)*4) + (j % 2)
			return mix(hr.at(min(3, max(0, i))), hairRot, 0.35)
		})
	}

	// The legs, the hips, the spine and the ribs, the arms, the skull.
	for i := 0; i < 2; i++ {
		lg := legGeo(sim.DirDown, i, lowerPose{})
		out := 1.0
		if lg.hx < 27 {
			out = -1
		}
		limb(lg.hx-out*1.2, lg.hy+1, lg.kx, lg.ky-1.2, 1.7*k, 2.1*k, 2.2*k)         // the thigh bone
		bone(sEll(lg.kx, lg.ky-0.4, 1.5, 1.7), 1.5, 0.1)                            // the kneecap
		limb(lg.kx+out*1.7, lg.ky+1.6, lg.ax+out*1.4, lg.ay-1, 0.65, 0.8, 0.9)      // the thin bone of the shin
		limb(lg.kx-out*0.3, lg.ky+1.2, lg.ax-out*0.2, lg.ay-0.8, 1.4*k, 1.9*k, 1.5) // and the thick one
		for j := -1.0; j <= 1; j++ {                                                // the foot
			x := lg.ax + j*1.2 + out*0.3
			bone(sCap(lg.ax, lg.ay+0.6, x+j*0.5, lg.ay+4.4, 0.6, 0.45), 0.8, 0)
		}
	}
	hw := 4.4 // half the breadth of the hips
	if female {
		hw = 5.4
	}
	for _, s := range []float64{-1, 1} {
		bone(sEll(27+s*hw*0.95, 71.2, 3.6*k+0.6, 3.1), 3, 0.02) // a wing of the hip
		hollow(sEll(27+s*2.5, 75.6, 1.2, 1))
	}
	bone(sCap(27-hw*0.75, 75.6, 27+hw*0.75, 75.6, 1.2, 1.2).sub(sEll(27-2.5, 75.6, 1.2, 1)).sub(sEll(27+2.5, 75.6, 1.2, 1)), 1.2, 0)
	bone(sBox(25.5, 68.5, 28.5, 75, 1.3), 1.5, 0.06) // the root of the spine
	for y := 44.5; y < 68.5; y += 1.6 {              // the spine
		bone(sBox(25.9, y, 28.1, y+1.2, 0.5), 0.9, -0.02)
	}
	shw := 9.6 // the collar bones reach the shoulders
	if female {
		shw = 8
	}
	for j := 0; j < 6; j++ { // the ribs
		y := 49.3 + float64(j)*2.35
		w := (8.4 - math.Abs(float64(j)-2)*0.55) * k
		for _, s := range []float64{-1, 1} {
			bone(sLock(27+s*1.3, y, 27+s*w*0.85, y-0.2, 27+s*w, y+2.6+float64(j)*0.25, 0.9, 0.7), 0.9, 0.02)
		}
	}
	bone(sCap(27, 48.4, 27, 57.6, 1.25, 0.8), 1.2, 0.08) // the breastbone
	for _, s := range []float64{-1, 1} {
		bone(sLock(27+s*1.2, 47.2, 27+s*5, 46.6, 27+s*shw, 47.6, 0.95, 0.8), 0.9, 0.06)
	}
	p := upperPoseOf(upKey{dir: sim.DirDown, anim: upFallen})
	for i := 0; i < 2; i++ {
		a := armGeo(female, sim.DirDown, i, p.arm[i])
		ux, uy, _ := unitOf(a.wx-a.ex, a.wy-a.ey)
		px, py := uy, -ux
		limb(a.sx, a.sy, a.ex, a.ey, 1.35*k, 2*k, 1.7*k) // the upper arm
		limb(a.ex+px*0.8, a.ey+py*0.8, a.wx+px*0.8, a.wy+py*0.8, 0.7, 0.9, 0.9)
		limb(a.ex-px*0.7, a.ey-py*0.7, a.wx-px*0.7, a.wy-py*0.7, 0.85, 1, 0.8)
		bone(sEll(a.wx+ux*0.8, a.wy+uy*0.8, 1.4, 1.4), 1.2, 0)
		for j := -1.5; j <= 1.5; j++ { // the fingers, fanned
			x0, y0 := a.wx+ux*1.8+px*j*0.7, a.wy+uy*1.8+py*j*0.7
			bone(sCap(x0, y0, x0+ux*3.6+px*j*0.5, y0+uy*3.6+py*j*0.5, 0.5, 0.4), 0.7, 0)
		}
	}
	// The skull, looking up: the brow over two dark sockets, the hole of
	// the nose, the teeth, the jaw a little open.
	m := 0.0
	if female {
		m = 0.4
	}
	skull := sEll(27, 34.4, 6.8-m, 7.3-m).or(sBox(23.2, 38, 30.8, 44.6, 2.6))
	bone(skull, 4.5, 0.08)
	for _, s := range []float64{-1, 1} {
		hollow(sEll(27+s*2.6, 36, 1.7, 1.55))
	}
	hollow(sPoly(27, 37.6, 25.9, 40, 28.1, 40))
	for x := 24; x <= 29; x++ { // the teeth
		f.dot(B, x, 41, skelHD[4-x%2])
		f.dot(B, x, 42, boneHollow)
	}
	for x := 24; x <= 29; x++ {
		f.dot(B, x, 43, skelHD[3])
	}

	// Rags of what they wore, faded and dirty, torn into holes.
	rag := func(at func(x, y int) bool, col func(x, y int) color.RGBA, l *figLayer) func(x, y int, _ float64) color.RGBA {
		return func(x, y int, _ float64) color.RGBA {
			if !at(x, y) {
				return color.RGBA{}
			}
			return col(x, y)
		}
	}
	torn := func(cut float64, salt uint32) func(x, y int) bool {
		return func(x, y int) bool {
			return vnoise(float64(x)/3.4, float64(y)/3.4, seed+salt) >= cut && hashf(x, y, seed+salt+1) >= 0.1
		}
	}
	if wear&sim.WearTunic != 0 {
		torso, _ := tallTorso(female, sim.DirDown, skirt{})
		S := g.S
		f.fill(S, torso, rag(torn(0.5, 3), func(x, y int) color.RGBA {
			return mix(shirtHD.at(int((torso.lum(x, y, 7)+0.02)*6)), dirt, 0.3)
		}, S))
		if !female { // the belt
			b := sBox(18.6, beltY, 35.4, beltY+3, 0.4)
			f.fill(T, b, rag(torn(0.35, 5), func(x, y int) color.RGBA { return mix(leatherHD.at(3-(x+y)%2), dirt, 0.25) }, T))
		} else {
			g.brooch(22, 53, 2, false)
			g.brooch(32, 53, 2, false)
		}
	} else {
		s, _, _ := loinShape(female, sim.DirDown, skirt{})
		fur := shagTone(loinHD, s, 5, 59)
		f.fill(T, s, rag(torn(0.45, 7), func(x, y int) color.RGBA { return mix(fur(x, y, -0.05), dirt, 0.3) }, T))
		if female {
			band := sBox(17.5, 49.8, 36.5, 58.6, 2)
			wrap := shagTone(loinHD, band, 3.5, 73)
			f.fill(T, band, rag(torn(0.55, 9), func(x, y int) color.RGBA { return mix(wrap(x, y, -0.05), dirt, 0.3) }, T))
		}
	}
	if wear&sim.WearMantle != 0 {
		ma := sBox(12.5, 45.5, 41.5, 50, 2.5)
		f.fill(T, ma, rag(torn(0.55, 11), func(x, y int) color.RGBA { return mix(furHD.at(2+(x+2*y)%3), dirt, 0.3) }, T))
	}
	for i := 0; i < 2; i++ {
		lg := legGeo(sim.DirDown, i, lowerPose{})
		if wear&sim.WearLegWraps != 0 {
			w := sCap(lg.kx, lg.ky+3, lg.ax, lg.ay-1, 3.6, 2.9)
			f.fill(T, w, rag(torn(0.5, 13+uint32(i)), func(x, y int) color.RGBA { return mix(wrapHD.at(1+(x+y)%3), dirt, 0.3) }, T))
		}
		if wear&sim.WearBoots != 0 { // leather lasts
			g.boot(lg, 0.15, [2]float64{1.4, -1.4}[i], false)
		}
	}

	layDown(dst, B.c, nil)
	layDown(dst, g.L.c, nil)
	layDown(dst, g.S.c, &tint)
	layDown(dst, T.c, nil)
	dst.outline(0.78)
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
