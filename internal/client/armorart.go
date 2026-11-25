package client

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Armour is drawn over the body, by tier: a hide tunic, bone ribs, agate
// scales or a tyrant's teeth between the shirt and the hands, and on top an
// agate headband or a skull cap. The shapes follow genPlayer's frame (18x26
// before the headroom, feet at (9, 24), head centred on (9, 7)); torsoSpan
// and sideArm in look.go give the torso and the near arm, broader on a man.

// armorArt is one tier's armour for one figure, by direction and walk frame:
// body is the plate, drawn under the bare arms and the head; helmet goes over
// everything, the shoulder guards and any headgear.
type armorArt struct {
	body, helmet [4][walkFrames]sprite
	whole        [4][walkFrames]sprite // body with no gap for the near arm, sideways only; see punch.go
}

type armorKey struct {
	tier   sim.Tier
	female bool
}

// armor returns the armour sprites of tier for a figure, drawing them on
// first use. It must be called inside the game loop, where images can be
// created.
func (a *art) armor(tier sim.Tier, female bool) *armorArt {
	k := armorKey{min(tier, sim.NumTiers-1), female}
	if s, ok := a.armors[k]; ok {
		return s
	}
	s := &armorArt{}
	for d := sim.DirDown; d <= sim.DirRight; d++ {
		for f := range walkFrames {
			s.body[d][f], s.helmet[d][f] = genArmor(k.tier, female, d, f)
		}
	}
	for f := range walkFrames {
		B, _ := armorLayersPose(k.tier, female, sim.DirRight, f, true)
		s.whole[sim.DirRight][f] = mk(B, 9, bodyFeet)
		s.whole[sim.DirLeft][f] = mirrorSprite(s.whole[sim.DirRight][f])
		for _, d := range []sim.Dir{sim.DirDown, sim.DirUp} {
			s.whole[d][f] = s.body[d][f]
		}
	}
	if a.armors == nil {
		a.armors = make(map[armorKey]*armorArt)
	}
	a.armors[k] = s
	return s
}

// drawArmor draws armour over a body drawn by drawBody with the same
// arguments.
func drawArmor(dst *ebiten.Image, a *armorArt, dir sim.Dir, frame int, x, y, s, bob float64) {
	drawAt(dst, a.body[dir][frame], x, y-bob*s, s, s, 0, nil)
	drawAt(dst, a.helmet[dir][frame], x, y-bob*s, s, s, 0, nil)
}

// genArmor draws one direction and walk frame of tier t's armour: a hide
// tunic, with lashed bone ribs, agate scales or a tyrant's teeth over it,
// and for the best a skull cap fringed with teeth.
func genArmor(t sim.Tier, female bool, d sim.Dir, frame int) (body, helmet sprite) {
	if d == sim.DirLeft {
		b, h := genArmor(t, female, sim.DirRight, frame)
		return mirrorSprite(b), mirrorSprite(h)
	}
	B, H := armorLayers(t, female, d, frame)
	return mk(B, 9, bodyFeet), mk(H, 9, bodyFeet)
}

// armorLayers draws the two layers of genArmor; d is not DirLeft.
func armorLayers(t sim.Tier, female bool, d sim.Dir, frame int) (B, H *canvas) {
	return armorLayersPose(t, female, d, frame, false)
}

// armorLayersPose is armorLayers; whole leaves no gap for the near arm of a
// figure seen from the side, for when that arm is thrown out in a punch.
func armorLayersPose(t sim.Tier, female bool, d sim.Dir, frame int, whole bool) (B, H *canvas) {
	const w, h = 18, 26
	B, H = newCanvas(w, h), newCanvas(w, h)
	step := walkStep(frame)
	hide := hideRamp
	if t == sim.TierDiamond {
		hide = darkHideRamp
	}
	// skin colours one pixel of the tunic, with t's trim over it.
	skin := func(c *canvas, x, y int, l float64, front bool) {
		col := hide.pick(l, x, y)
		if hashf(x, y, 66) < 0.12 {
			col = hide.pick(l-0.25, x, y) // the grain of the hide
		}
		switch {
		case t == sim.TierCopper && (y == 13 || y == 18):
			col = furTrim.pick(l+0.2*float64(x%2), x, y) // fur at the neck and hem
		case t == sim.TierIron && front && (y == 14 || y == 16):
			col = hornRamp.pick(l+0.25, x, y) // bones lashed across
		case t == sim.TierPlatinum && y >= 13 && y <= 17 && (x+y)%2 == 0:
			col = agateRamp.pick(l+0.1, x, y) // agate scales
		case t == sim.TierDiamond && front && y == 14 && x%2 == 0:
			col = toothArt.pick(0.9, x, y) // a row of teeth on a cord
		case t == sim.TierDiamond && front && y == 13:
			col = hide[0]
		}
		c.set(x, y, col)
	}
	shoulder := func(c *canvas, x, y int, l float64) {
		switch t {
		case sim.TierCopper:
			c.set(x, y, furTrim.pick(l, x, y))
		case sim.TierPlatinum:
			c.set(x, y, agateRamp.pick(l, x, y))
		default:
			c.set(x, y, hornRamp.pick(l, x, y)) // bone
		}
	}
	// The plate follows the torso (torsoSpan in look.go); the shoulder guards
	// sit on the round shoulders, so they go on the top sprite, over the bare
	// arms. A man's shoulders are broader.
	switch d {
	case sim.DirDown, sim.DirUp:
		for y := 13; y <= 18; y++ {
			x0, x1, ok := torsoSpan(female, d, y)
			for x := x0; ok && x <= x1; x++ {
				l := 0.85 - float64(x-4)/9*0.45 - float64(y-13)*0.03
				if y == 18 {
					l -= 0.2
				}
				skin(B, x, y, l, d == sim.DirDown)
			}
		}
		left, right := 3, 12 // each guard is three pixels wide
		if female {
			left, right = 4, 11
		}
		for _, sx := range []int{left, right} {
			for dx := range 3 {
				for y := 12; y <= 13; y++ {
					shoulder(H, sx+dx, y, 0.95-float64(sx+dx-2)/13*0.55-float64(y-12)*0.2)
				}
			}
		}
		if t == sim.TierDiamond { // bone spikes on the shoulders
			shoulder(H, left+1, 11, 0.9)
			shoulder(H, right+1, 11, 0.6)
		}
	case sim.DirRight:
		for y := 13; y <= 18; y++ {
			x0, x1, ok := torsoSpan(female, d, y)
			for x := x0; ok && x <= x1; x++ {
				if !whole && sideArm(step, x, y) {
					continue // the arm and the hand
				}
				l := 0.82 - float64(x-5)/7*0.4
				if y == 18 {
					l -= 0.2
				}
				skin(B, x, y, l, x >= 9)
			}
		}
		width := 4
		if female {
			width = 3
		}
		for dx := range width {
			for y := 12; y <= 13; y++ {
				shoulder(H, 7+dx, y, 0.95-float64(dx)*0.25-float64(y-12)*0.2)
			}
		}
		if t == sim.TierDiamond {
			shoulder(H, 8, 11, 0.9)
		}
	}

	B, H = padTop(B, headroom), padTop(H, headroom)
	// Headgear, in the sprite's own rows (look.go): the head is on rows 5 to
	// 15 and the hair a row or two above.
	switch t {
	case sim.TierPlatinum: // a hide band round the brow with an agate set in it
		const row = 8
		x0, x1, _ := headSpan(row)
		for x := x0; x <= x1; x++ {
			H.put(x, row, hide.at(3-(x-x0)/4))
		}
		if d == sim.DirDown {
			H.put(8, row, agateRamp[3])
			H.put(9, row, agateRamp[2])
		}
		if d == sim.DirRight {
			H.put(x1, row, agateRamp[3])
		}
	case sim.TierDiamond: // a skull cap, its brow fringed with teeth
		for y := 3; y <= 14; y++ {
			x0, x1, ok := headSpan(y)
			switch y {
			case 3:
				x0, x1, ok = 6, 11, true
			case 4:
				x0, x1, ok = 4, 13, true
			case 5, 6:
				x0, x1 = 3, 14
			}
			for x := x0; ok && x <= x1; x++ {
				brow := 8
				switch {
				case d == sim.DirUp:
					brow = 13
				case d == sim.DirRight && x < 8:
					brow = 13
				}
				switch {
				case y == brow+1 && x%2 == 0 && d != sim.DirUp:
					H.put(x, y, toothArt[3]) // the teeth
				case y <= brow:
					tone := 3
					switch {
					case x >= x1-1 || y == brow:
						tone = 1
					case x <= x0+4 && y <= 6:
						tone = 4
					case x >= x1-4:
						tone = 2
					}
					if (x == 6 || x == 11) && y == 6 && d == sim.DirDown {
						H.put(x, y, colInk) // eye sockets of the skull
						continue
					}
					H.put(x, y, hornRamp.at(tone))
				}
			}
		}
	}

	B.outline(0.75)
	H.outline(0.75)
	return B, H
}

// drawArmored draws p in armour: the body up to the shirt, the armour,
// then the head, hands and hanging hair over it, and any headgear on top.
// Under the skull cap, hair that stands up (spikes, a bun) is drawn as short
// hair: the cap covers it.
func (g *Game) drawArmored(dst drawTarget, p protocol.PlayerView, dir sim.Dir, frame int, x, y, bob float64, tint color.Color) {
	l := p.Look
	if p.Armor >= sim.TierDiamond && (l.Hair == sim.HairSpiky || l.Hair == sim.HairBun) {
		l.Hair = sim.HairShort
	}
	b, a := g.art.body(l), g.art.armor(p.Armor, l.Female)
	drawAt(dst, b.back[dir][frame], x, y-bob, 1, 1, 0, nil)
	drawAt(dst, b.legs[dir][frame], x, y, 1, 1, 0, nil)
	drawAt(dst, b.shirt[dir][frame], x, y-bob, 1, 1, 0, tint)
	drawAt(dst, a.body[dir][frame], x, y-bob, 1, 1, 0, nil)
	drawAt(dst, b.top[dir][frame], x, y-bob, 1, 1, 0, nil)
	drawAt(dst, a.helmet[dir][frame], x, y-bob, 1, 1, 0, nil)
}

var (
	darkHideRamp = mkRamp(0x2a1c14, 0x45301f, 0x624530, 0x7f5c40, 0x9c7652)
	furTrim      = mkRamp(0x6b5236, 0x947552, 0xbb9d74, 0xdcc49c)
)
