package client

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Armour is drawn over the body, in its tier's metal: a breastplate and
// shoulder guards between the shirt and the hands, and a helmet over the
// hair. The shapes follow genPlayer's frame (18x26 before the headroom, feet
// at (9, 24), head centred on (9, 7)); only a woman's waist changes them.

// armorArt is one tier's armour for one figure, by direction and walk frame.
type armorArt struct {
	body, helmet [4][walkFrames]sprite
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
			s.body[d][f], s.helmet[d][f] = genArmor(tierRamps[k.tier], female, d, f)
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

// genArmor draws one direction and walk frame of armour in rp.
func genArmor(rp ramp, female bool, d sim.Dir, frame int) (body, helmet sprite) {
	if d == sim.DirLeft {
		b, h := genArmor(rp, female, sim.DirRight, frame)
		return mirrorSprite(b), mirrorSprite(h)
	}
	const w, h = 18, 26
	const ax, ay = 9, 24
	B, H := newCanvas(w, h), newCanvas(w, h)
	step := 0
	switch frame {
	case 1:
		step = -1
	case 3:
		step = 1
	}
	metal := func(c *canvas, x, y int, l float64) { c.set(x, y, rp.pick(l, x, y)) }
	waist := func(x, y, left, right int) bool { return female && (y == 16 || y == 17) && (x == left || x == right) }

	switch d {
	case sim.DirDown, sim.DirUp:
		// The breastplate (or backplate): lit from the left, a ridge down
		// the middle in front, a darker rim at the bottom.
		for y := 13; y <= 18; y++ {
			for x := 5; x <= 12; x++ {
				if waist(x, y, 5, 12) {
					continue
				}
				l := 0.92 - float64(x-5)/7*0.55 - float64(y-13)*0.03
				switch {
				case y == 18:
					l -= 0.3
				case d == sim.DirDown && (x == 8 || x == 9):
					l += 0.15
				}
				metal(B, x, y, l)
			}
		}
		if d == sim.DirDown { // rivets
			B.set(6, 14, rp[len(rp)-1])
			B.set(11, 14, rp[1])
		}
		// Shoulder guards over the tops of the arms.
		for _, sx := range []int{4, 12} {
			for dx := range 2 {
				for y := 12; y <= 13; y++ {
					metal(B, sx+dx, y, 0.95-float64(sx+dx-4)/9*0.55-float64(y-12)*0.2)
				}
			}
		}
	case sim.DirRight:
		// The near arm swings in front of the torso: the plate shows round it.
		arm := func(x, y int) bool {
			off := 0
			if y >= 16 {
				off = -step
			}
			return y >= 13 && y <= 17 && (x == 8+off || x == 9+off)
		}
		for y := 13; y <= 18; y++ {
			for x := 6; x <= 11; x++ {
				if arm(x, y) || waist(x, y, 6, 11) || y == 18 && (x == 8-step || x == 9-step) {
					continue // the arm, the waist and the hand
				}
				l := 0.88 - float64(x-6)/5*0.5
				if y == 18 {
					l -= 0.3
				}
				metal(B, x, y, l)
			}
		}
		for dx := range 3 { // one guard, on the near shoulder
			for y := 12; y <= 13; y++ {
				metal(B, 7+dx, y, 0.95-float64(dx)*0.25-float64(y-12)*0.2)
			}
		}
	}

	// The helmet: a dome a little bigger than the head, down to the brow in
	// front and lower behind, with a brim.
	const cx, cy, rx, ry = 9.0, 7.0, 6.6, 6.4
	for y := 0; y < 14; y++ {
		for x := 0; x < w; x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if nx*nx+ny*ny > 1 {
				continue
			}
			brow := 6 // the last row covered
			switch {
			case d == sim.DirUp:
				brow = 9 // the back of the head
			case d == sim.DirRight && x < 8:
				brow = 9
			}
			if y > brow {
				continue
			}
			l := lum(nx, ny) + 0.1
			if y == brow {
				l -= 0.35 // the brim
			}
			metal(H, x, y, l)
		}
	}
	// A crest along the top.
	for y := 1; y <= 3; y++ {
		metal(H, 9, y, 0.95)
	}
	if d == sim.DirRight {
		metal(H, 13, 7, 0.4) // the brim reaches forward a little
	}

	B, H = padTop(B, headroom), padTop(H, headroom)
	B.outline(0.75)
	H.outline(0.75)
	return mk(B, ax, ay+headroom), mk(H, ax, ay+headroom)
}

// drawArmored draws p in armour: the body up to the shirt, the breastplate,
// then the head, hands and hanging hair over it, and the helmet on top.
// Under a helmet, hair that stands up (spikes, a bun) is drawn as short hair:
// the helmet covers it.
func (g *Game) drawArmored(dst *ebiten.Image, p protocol.PlayerView, dir sim.Dir, frame int, x, y, bob float64, tint color.Color) {
	l := p.Look
	if l.Hair == sim.HairSpiky || l.Hair == sim.HairBun {
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
