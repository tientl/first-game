package client

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// What a player holds (the selected hotbar slot, PlayerView.Held) is drawn
// in their hand. Tools have their own sprite; anything else is its bag icon
// shrunk to half size, which reads at the player's scale.

// handAt is where the hand that holds things is, in art pixels from the
// player sprite's anchor (between the feet), for the facing and the walk
// step (-1, 0 or 1, as genPlayer swings the arms). It matches genPlayer:
// facing down or up it is the hand on the right of the screen, sideways the
// near hand.
func handAt(dir sim.Dir, step int) (dx, dy float64) {
	switch dir {
	case sim.DirRight:
		return float64(-2 * step), -5.5
	case sim.DirLeft:
		return float64(2 * step), -5.5
	default:
		return 4.5, -5.5 + float64(step)
	}
}

// walkStep is the arm swing genPlayer draws for walk frame f.
func walkStep(f int) int {
	switch f {
	case 1:
		return -1
	case 3:
		return 1
	}
	return 0
}

// facingSide is 1 when the player's hand side is on the right of the screen.
func facingSide(dir sim.Dir) float64 {
	if dir == sim.DirLeft {
		return -1
	}
	return 1
}

// heldBehind reports whether the held item is hidden behind the body.
func heldBehind(dir sim.Dir) bool { return dir == sim.DirUp }

// drawHeld draws item k in the hand of a player standing at (sx, sy) in art
// pixels. bob is how far the body is lifted this frame.
func (g *Game) drawHeld(dst drawTarget, k sim.ItemKind, dir sim.Dir, sx, sy, bob float64, step int) {
	hx, hy := handAt(dir, step)
	g.drawHeldAt(dst, k, dir, sx+hx, sy+hy-bob)
}

// drawHeldAt draws item k in a hand at (hx, hy) in art pixels, of a player
// facing dir.
func (g *Game) drawHeldAt(dst drawTarget, k sim.ItemKind, dir sim.Dir, hx, hy float64) {
	if !k.Valid() {
		return
	}
	side := facingSide(dir)

	// Tools of every tier: the head is drawn in the tier's material.
	switch class, tier := k.ToolKind(); class {
	case sim.ToolAxe, sim.ToolPickaxe:
		// Gripped low on the handle: beside the body it stands nearly
		// upright; sideways the hand is in front of the face, so the head
		// tips forward, clear of it.
		tilt := 0.35
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 1.1
		}
		s := g.art.ore.axes[tier]
		if class == sim.ToolPickaxe {
			s = g.art.ore.picks[tier]
		}
		drawAt(dst, s, math.Round(hx), math.Round(hy)+1, side, 1, tilt*side, nil)
		return
	case sim.ToolHoe:
		// Carried nearly upright, so the blade hangs from the top like a "7"
		// and does not read as an axe.
		drawAt(dst, g.art.ore.hoes[tier], math.Round(hx), math.Round(hy)+1, side, 1, 0.15*side, nil)
		return
	}

	if t, ok := sim.SwordTier(k); ok {
		// Upright beside the body; sideways it points ahead, like the spear.
		tilt := 0.2
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 1.0
		}
		drawAt(dst, g.art.gear.swords[t], math.Round(hx), math.Round(hy)+1, side, 1, tilt*side, nil)
		return
	}

	switch k {
	case sim.ItemBattleAxe:
		drawBattleAxeHeld(dst, dir, hx, hy, side)
	case sim.ItemSpear, sim.ItemIvorySpear:
		// Upright beside the body; sideways it points ahead.
		tilt := 0.15
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 1.25
		}
		s := g.art.heldSpear
		if k == sim.ItemIvorySpear {
			s = g.art.heldIvorySpear
		}
		drawAt(dst, s, math.Round(hx), math.Round(hy)+1, side, 1, tilt*side, nil)
	case sim.ItemRod:
		tilt := 0.5
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 0.9
		}
		drawAt(dst, g.art.heldRod, math.Round(hx), math.Round(hy)+1, side, 1, tilt*side, nil)
	case sim.ItemTorch:
		// Held out in front, so the flame burns beside the face, not on it.
		tilt := 0.3 * side
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 0.85 * side
		}
		x, y := math.Round(hx), math.Round(hy)+1
		drawAt(dst, g.art.heldTorch, x, y, side, 1, tilt, nil)
		// The flame sits on the wrapped head, 10 pixels up the stick.
		const up = 10
		f := g.art.flames[(g.frame/4)%flameFrames]
		drawAt(dst, f, math.Round(x+math.Sin(tilt)*up), math.Round(y-math.Cos(tilt)*up)+1, 0.4, 0.45, 0, nil)
	default:
		icon := g.art.icons[k]
		if icon == nil {
			return
		}
		// Held a little out from the hand. A half-size linear draw averages
		// each 2x2 block of the icon, a clean pixel-art shrink, as long as it
		// lands on whole pixels.
		b := icon.Bounds()
		cx, cy := math.Round(hx+side*2), math.Round(hy+2)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
		op.GeoM.Scale(0.5*side, 0.5)
		op.GeoM.Translate(cx, cy)
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(icon, op)
	}
}
