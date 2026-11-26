package client

import (
	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The light in the natural caves (see cavebake.go). The upper level is
// dark, but not black: the walls can be made out a little way off. The
// lower one is darker. Daylight comes down the shafts over the ways up and
// spills across their halls, pale and cool, dimmer by night.

// naturalDark is how dark each level of the natural caves is, away from any
// light.
var naturalDark = [sim.CaveLevels]float64{0.8, 0.92}

const (
	caveDayRadius   = 7.5  // tiles of daylight round the way up
	caveDayPunch    = 0.62 // how much of the dark it takes away, by day
	caveShaftRadius = 3.6  // tiles of light round the rope up from the deep caves
	caveShaftPunch  = 0.45
)

// naturalCaves reports whether this world has the natural caves.
func (g *Game) naturalCaves() bool { return g.caveCells != nil }

// caveLevel is the level of the natural caves this player is on, if any.
func (g *Game) caveLevel() (int, bool) {
	if !g.naturalCaves() {
		return 0, false
	}
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			k, _, ok := sim.CaveLevelAt(g.height, g.surfaceH, int(p.Y))
			return k, ok
		}
	}
	return 0, false
}

// caveLights punches the caves' light out of the dark (punch), or adds its
// glow over the scene.
func (g *Game) caveLights(dst drawTarget, cam camera, punch bool, in float64) {
	day := 0.35 + 0.65*sim.Ambient(g.curr.Tick)
	for _, e := range g.curr.Entities {
		r, k := caveDayRadius, caveDayPunch*day
		switch e.Kind {
		case sim.EntityCaveExit:
		case sim.EntityCaveUp: // a little of it comes down the shaft to the deep caves
			r, k = caveShaftRadius, caveShaftPunch*day
		default:
			continue
		}
		op := g.lightOp(cam, e.TX, e.TY+1, r)
		if e.Kind == sim.EntityCaveExit {
			op = g.lightOp(cam, e.TX, e.TY+2, r)
		}
		if punch {
			op.Blend = ebiten.BlendDestinationOut
			op.ColorScale.ScaleAlpha(float32(k * in))
		} else {
			k := float32(0.1 * day * in)
			op.ColorScale.Scale(k*0.8, k*0.9, k, k)
			op.Blend = ebiten.BlendLighter
		}
		dst.DrawImage(g.art.light, op)
	}
}
