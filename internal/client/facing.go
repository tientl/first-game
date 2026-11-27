package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"survivalgame/internal/sim"
)

// Turning what is being placed (the rules are in sim/facing.go): while a
// structure waits for a tile (build mode, a palette piece in free build) or
// something that turns is in the hand (a leaf tent, a head on a post, a
// grill for the fire), R turns it a quarter clockwise and the wheel turns
// it either way, while its ghost shows; the ghost is drawn facing that way
// with a little arrow at its front. Each kind remembers its facing, so the
// next one put down faces the same way. The hint over the hotbar says so.

// turnState is which way the things being placed face.
type turnState struct {
	face  [sim.NumEntityKinds]sim.Dir // the facing last chosen for each kind
	wheel float64                     // wheel movement not yet a whole turn
	at    int                         // the frame it last turned, for the ghost's arrow
}

// placing is the kind and variant being placed, when it turns: a structure
// in build mode, a palette piece, or what is held. Nothing is placed while
// riding, sailing, swimming or dead.
func (g *Game) placing() (sim.EntityKind, uint8, bool) {
	me, ok := g.me()
	if !ok || me.Dead || g.curr.Self.Dead || me.Mount != 0 || me.Boat != 0 || me.Swimming {
		return sim.EntityNone, 0, false
	}
	k, v := sim.EntityNone, uint8(0)
	switch {
	case g.build.on && g.build.dev:
		pc, ok := sim.DevPieceAt(g.build.piece)
		if !ok {
			return sim.EntityNone, 0, false
		}
		k, v = pc.Kind, pc.Var
	case g.build.on:
		r, ok := sim.RecipeByID(g.build.recipe)
		if !ok {
			return sim.EntityNone, 0, false
		}
		k = r.Places
	default:
		held := g.myHeld()
		if gear := sim.GearOf(held); gear != sim.GearNone {
			k, v = sim.EntityCampfire, uint8(gear) // onto a fire; see cookrender.go
		} else if pk, _, ok := sim.PlacedBy(held); ok {
			k = pk
		}
	}
	return k, v, sim.Turns(k, v)
}

// facingFor is the way the next thing of kind k, variant v, put down faces.
func (g *Game) facingFor(k sim.EntityKind, v uint8) sim.Dir {
	if k >= sim.NumEntityKinds {
		return sim.DirDown
	}
	return sim.FacingOf(k, v, g.turn.face[k])
}

// turnPlacing turns what is being placed by steps (clockwise seen from
// above for steps > 0) and reports whether anything turned.
func (g *Game) turnPlacing(steps int) bool {
	k, v, ok := g.placing()
	if !ok || steps == 0 {
		return false
	}
	g.turn.face[k] = sim.NextFacing(k, v, g.facingFor(k, v), steps)
	g.turn.at = g.frame
	snd.ui("ui_move")
	return true
}

// TurnPlacing turns what is being placed by steps, as R does: for the
// screenshot tool.
func (g *Game) TurnPlacing(steps int) bool { return g.turnPlacing(steps) }

// PlacingFacing is the way what is being placed faces now, and its kind:
// for the screenshot tool.
func (g *Game) PlacingFacing() (sim.EntityKind, sim.Dir, bool) {
	k, v, ok := g.placing()
	return k, g.facingFor(k, v), ok
}

// turnKey is R in play: it turns what is being placed, and reports
// whether it did (else R gets on or off a beast, as before).
func (g *Game) turnKey() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyR) && g.turnPlacing(1)
}

// wheelTurns turns what is being placed with the wheel while its ghost
// shows, and reports whether the wheel was taken (so it does not also pick
// a hotbar slot). Wheel away from you: back; towards you: clockwise.
func (g *Game) wheelTurns() bool {
	if _, _, ok := g.placing(); !ok || !g.ghost().ok {
		g.turn.wheel = 0
		return false
	}
	_, dy := ebiten.Wheel()
	g.turn.wheel += dy
	for ; g.turn.wheel >= 1; g.turn.wheel-- {
		g.turnPlacing(-1)
	}
	for ; g.turn.wheel <= -1; g.turn.wheel++ {
		g.turnPlacing(1)
	}
	return true
}

// turnHint is the hint over the hotbar while something that turns is
// being placed.
func (g *Game) turnHint() (key, s string, ok bool) {
	if _, _, ok := g.placing(); !ok {
		return "", "", false
	}
	return "R / lăn chuột:", "xoay", true
}

// facingName names facing d of kind k, for the ghost's label.
func facingName(k sim.EntityKind, v uint8, d sim.Dir) string {
	if !sim.Turns(k, v) {
		return ""
	}
	switch sim.FacingOf(k, v, d) {
	case sim.DirRight:
		if k == sim.EntityWall || k == sim.EntityStoneWall || k == sim.EntitySpikes || k == sim.EntityCampfire {
			return "dọc"
		}
		return "hướng đông"
	case sim.DirLeft:
		if k == sim.EntityTrophy {
			return "nhìn trái"
		}
		return "hướng tây"
	case sim.DirUp:
		return "hướng bắc"
	}
	switch k {
	case sim.EntityWall, sim.EntityStoneWall, sim.EntitySpikes, sim.EntityCampfire:
		return "ngang"
	case sim.EntityTrophy:
		return "nhìn phải"
	}
	return "hướng nam"
}

var colTurnArrow = color.RGBA{255, 236, 170, 255}

// drawFacingArrow marks the front of a ghost of kind k facing d on the tile
// whose top left is (x, y) on the world canvas: a chevron on the edge it
// faces, which swings in for a moment after a turn. A run gets a bar along
// it instead.
func (g *Game) drawFacingArrow(dst drawTarget, k sim.EntityKind, v uint8, d sim.Dir, x, y float64, col color.RGBA) {
	if !sim.Turns(k, v) {
		return
	}
	cx, cy := x+TileW/2, y+TileH/2
	pulse := uint8(170 + 60*math.Sin(float64(g.frame)*0.15))
	c := color.RGBA{uint8(int(col.R) * int(pulse) / 255), uint8(int(col.G) * int(pulse) / 255), uint8(int(col.B) * int(pulse) / 255), pulse}
	if k == sim.EntityWall || k == sim.EntityStoneWall || k == sim.EntitySpikes || k == sim.EntityCampfire {
		if sim.RunsNS(d) {
			strokeLine(dst, float32(cx), float32(y)+3, float32(cx), float32(y)+TileH-3, 1, c, true)
		} else {
			strokeLine(dst, float32(x)+3, float32(cy), float32(x)+TileW-3, float32(cy), 1, c, true)
		}
		return
	}
	if k == sim.EntityTrophy && d != sim.DirLeft {
		d = sim.DirRight // a profile: the way the head looks
	}
	fx, fy := d.Delta()
	// Just turned: the chevron slides out from the middle.
	t := min(1, float64(g.frame-g.turn.at)/8)
	ex, ey := float64(fx)*(TileW/2+1.5), float64(fy)*(TileH/2+1.5)
	px, py := cx+ex*(0.4+0.6*t), cy+ey*(0.4+0.6*t)
	// The chevron's arms, back from its point.
	bx, by := -float64(fx)*3, -float64(fy)*2.4
	sx, sy := float64(fy)*3, float64(fx)*2.4
	strokeLine(dst, float32(px), float32(py), float32(px+bx+sx), float32(py+by+sy), 1.2, c, true)
	strokeLine(dst, float32(px), float32(py), float32(px+bx-sx), float32(py+by-sy), 1.2, c, true)
}
