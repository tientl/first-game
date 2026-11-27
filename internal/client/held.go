package client

import (
	"math"

	"survivalgame/internal/sim"
)

// What a player holds (the selected hotbar slot, PlayerView.Held) is drawn
// in their hand. Tools and weapons are painted at HD and turned with the
// hand (heldhd.go; the old density-1 sprites below are kept for
// SetHeldHD(false)); anything else is its bag icon shrunk to half size,
// which reads at the player's scale.

// heldCarry is how far item k is tipped from upright towards the facing
// as it is carried: beside the body a tool stands nearly upright; sideways
// the hand is in front of the face, so the head tips forward, clear of it.
func heldCarry(k sim.ItemKind, dir sim.Dir) float64 {
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	pick := func(beside, ahead float64) float64 {
		if sideways {
			return ahead
		}
		return beside
	}
	switch class, _ := k.ToolKind(); class {
	case sim.ToolAxe, sim.ToolPickaxe:
		return pick(0.35, 1.1)
	case sim.ToolHoe:
		return 0.15 // nearly upright, so the blade hangs from the top like a 7
	}
	if isSword(k) {
		return pick(0.2, 1.0)
	}
	switch k {
	case sim.ItemBattleAxe:
		return cleaveCarry
	case sim.ItemSpear, sim.ItemIvorySpear:
		return pick(0.15, 1.25)
	case sim.ItemRod:
		return pick(0.5, 0.9)
	case sim.ItemTorch: // held out in front, so the flame burns beside the face
		return pick(0.3, 0.85)
	}
	return 0
}

// handAt is where the hand that holds things is, in art pixels from the
// player sprite's anchor (between the feet), for the facing and the walk
// step (-1 to 1, as genPlayer swings the arms: walkSwing). It matches genPlayer:
// facing down or up it is the hand on the right of the screen, sideways the
// near hand.
func handAt(dir sim.Dir, step float64) (dx, dy float64) {
	switch dir {
	case sim.DirRight:
		return -2 * step, -5.5
	case sim.DirLeft:
		return 2 * step, -5.5
	default:
		return 4.5, -5.5 + step
	}
}

// walkStep is the arm swing genPlayer draws for its walk frame f (of four;
// the HD figure's is walkSwing).
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
func (g *Game) drawHeld(dst drawTarget, k sim.ItemKind, dir sim.Dir, sx, sy, bob, step float64) {
	hx, hy := handAt(dir, step)
	g.drawHeldAt(dst, k, dir, sx+hx, sy+hy-bob)
}

// heldEnds draws what goes on the end of k, drawn gripped at (hx, hy),
// turned th towards side, foreshortened to q: a torch's flame; and notes
// where the flame and a rod's tip are, for the light and the line.
func (g *Game) heldEnds(dst drawTarget, k sim.ItemKind, d *heldDef, hx, hy, th, side, q float64) {
	switch k {
	case sim.ItemTorch:
		fx, fy := heldPoint(d, hx, hy, th, side, 0, torchFlameV*q)
		drawTorchFlame(dst, g.frame/fireStep+int(heldBy.id)*3, fx, fy)
		g.markHeld(func(m *heldMark) { m.flame, m.fx, m.fy = true, fx, fy })
	case sim.ItemRod:
		tx, ty := heldPoint(d, hx, hy, th, side, rodTipU, rodTipV*q)
		g.markHeld(func(m *heldMark) { m.tip, m.tipX, m.tipY = true, tx, ty })
	}
}

// carryKind is how an item is carried by a figure walking, running or
// standing (drawCarried).
type carryKind uint8

const (
	carryHang  carryKind = iota // hanging from the hand, the head down and forward: axes, picks, hoes, knives
	carryPole                   // along the body, the point forward and up: spears, the rod
	carryTorch                  // held up and forward, the flame over the shoulder (upTorch)
)

// Carried, a hanging tool leans its head forward from straight down by
// hangLean, and turns with the forearm by hangFollow of its swing; a pole
// points poleUp from upright (30 degrees above level) and turns less.
const (
	hangLean   = 0.5
	hangFollow = 0.45
	poleUp     = 1.05
)

// carriedBehind reports whether item k, carried by a figure facing dir, is
// drawn behind the body: seen from the front or the back a tool hangs or a
// pole stands along the body's side, foreshortened, its head beside the hip.
func carriedBehind(k sim.ItemKind, dir sim.Dir) bool {
	d := heldDefOf(k)
	return d != nil && d.carry != carryTorch && (dir == sim.DirDown || dir == sim.DirUp)
}

// drawCarried draws item k carried in a hand at (hx, hy) by a figure facing
// dir that walks, runs or stands, its forearm turned swing towards the
// facing (see carryKind). It reports false for an item that has no held
// painter.
func (g *Game) drawCarried(dst drawTarget, k sim.ItemKind, dir sim.Dir, hx, hy, swing float64) bool {
	d := heldDefOf(k)
	if d == nil {
		return false
	}
	side := facingSide(dir) * heldFlip
	sideways := dir == sim.DirLeft || dir == sim.DirRight
	switch d.carry {
	case carryHang:
		// Mirrored, so that hanging head down its edge faces forward; seen
		// from the front or the back it hangs outwards, foreshortened.
		if sideways {
			th := math.Pi + hangLean + hangFollow*swing
			drawHeldDef(dst, d, hx, hy, th, -side, 1)
		} else {
			drawHeldLow(dst, d, hx, hy, math.Pi+0.35+0.25*swing, -side)
		}
	case carryPole:
		th, q := poleUp-0.3*swing, 1.0
		if sideways {
			drawHeldDef(dst, d, hx, hy, th, side, 1)
		} else { // upright at the side, beyond the body
			th = 0.12
			drawHeldDef(dst, d, hx, hy, th, side, 1)
		}
		g.heldEnds(dst, k, d, hx, hy, th, side, q)
	case carryTorch:
		th := 0.12
		if sideways {
			th = 0.3 - 0.2*swing
		}
		drawHeldDef(dst, d, hx, hy, th, side, 1)
		g.heldEnds(dst, k, d, hx, hy, th, side, 1)
	}
	return true
}

// heldFlip mirrors what drawHeldAt draws: -1 for the hand on the left of
// the screen of a figure seen from the front or the back.
var heldFlip = 1.0

// drawHeldAt draws item k in a hand at (hx, hy) in art pixels, of a player
// facing dir.
func (g *Game) drawHeldAt(dst drawTarget, k sim.ItemKind, dir sim.Dir, hx, hy float64) {
	if !k.Valid() {
		return
	}
	side := facingSide(dir)
	side *= heldFlip
	if d := heldDefOf(k); d != nil { // painted at HD and turned cleanly; see heldhd.go
		th := heldCarry(k, dir) + heldTurn
		drawHeldDef(dst, d, hx, hy, th, side, 1)
		g.heldEnds(dst, k, d, hx, hy, th, side, 1)
		return
	}

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
		drawAt(dst, s, math.Round(hx), math.Round(hy)+1, side, 1, (tilt+heldTurn)*side, nil)
		return
	case sim.ToolHoe:
		// Carried nearly upright, so the blade hangs from the top like a "7"
		// and does not read as an axe.
		drawAt(dst, g.art.ore.hoes[tier], math.Round(hx), math.Round(hy)+1, side, 1, (0.15+heldTurn)*side, nil)
		return
	}

	if t, ok := sim.SwordTier(k); ok {
		// Upright beside the body; sideways it points ahead, like the spear.
		tilt := 0.2
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 1.0
		}
		drawAt(dst, g.art.gear.swords[t], math.Round(hx), math.Round(hy)+1, side, 1, (tilt+heldTurn)*side, nil)
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
		drawAt(dst, s, math.Round(hx), math.Round(hy)+1, side, 1, (tilt+heldTurn)*side, nil)
	case sim.ItemRod:
		tilt := 0.5
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = 0.9
		}
		drawAt(dst, g.art.heldRod, math.Round(hx), math.Round(hy)+1, side, 1, (tilt+heldTurn)*side, nil)
	case sim.ItemTorch:
		// Held out in front, so the flame burns beside the face, not on it.
		tilt := (0.3 + heldTurn) * side
		if dir == sim.DirLeft || dir == sim.DirRight {
			tilt = (0.85 + heldTurn) * side
		}
		x, y := math.Round(hx), math.Round(hy)+1
		drawAt(dst, g.art.heldTorch, x, y, side, 1, tilt, nil)
		// The flame sits on the wrapped head, 10 pixels up the stick.
		const up = 10
		f := g.art.flames[(g.frame/4)%flameFrames]
		drawAt(dst, f, math.Round(x+math.Sin(tilt)*up), math.Round(y-math.Cos(tilt)*up)+1, 0.4, 0.45, 0, nil)
	default:
		// Held a little out from the hand, at half its bag size: an HD icon
		// baked at that size (iconhd.go), else the old one redrawn three
		// times as fine (iconFine), or as it was (drawIconAny).
		g.drawIconAny(dst, k, math.Round(hx+side*2), math.Round(hy+2), 0.5, side, 0)
	}
}
