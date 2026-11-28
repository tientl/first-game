package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// A sword is swung in a real blow: raised behind the shoulder, then brought
// down across the front, its tip leaving a bright arc that fades. Each blow
// starts when a snapshot first shows the player acting, so the slash keeps
// time with the server's hits, for every player in view.

const (
	slashFrames = 14   // how long the blade takes to come down, in frames
	slashFade   = 10   // how long the arc lingers after it, in frames
	slashFrom   = -1.3 // blade angle when raised, radians from upright, towards the back
	slashTo     = 2.1  // and when it has come down in front
	slashReach  = 15.0 // grip to tip, in art pixels

	// A backhand (every other blow; see backhand in motionpick.go) sweeps the
	// other way: from low in front up past the shoulder.
	backslashFrom = 2.5
	backslashTo   = -0.6
)

// noteSwings records when each player in s began acting.
func (g *Game) noteSwings(s protocol.Snapshot) {
	if g.swingAt == nil {
		g.swingAt = make(map[sim.PlayerID]int)
	}
	for _, p := range s.Players {
		if p.Acting && !g.prevPlayers[p.ID].Acting {
			g.swingAt[p.ID] = g.frame
		}
	}
}

// slashAngle is the blade's angle k of the way through the blow: fast at
// first, slowing as it comes down.
func slashAngle(k float64) float64 {
	e := 1 - (1-k)*(1-k)
	return slashFrom + (slashTo-slashFrom)*e
}

// backslashAngle is slashAngle for a backhand.
func backslashAngle(k float64) float64 {
	e := 1 - (1-k)*(1-k)
	return backslashFrom + (backslashTo-backslashFrom)*e
}

// drawSlash draws p's held sword in the middle of a blow, and its arc.
func (g *Game) drawSlash(dst drawTarget, p protocol.PlayerView, d drawable, bob float64) {
	tier, ok := sim.SwordTier(p.Held)
	if !ok {
		return
	}
	side := 1.0
	if p.Dir == sim.DirLeft {
		side = -1
	}
	gx, gy := d.sx+6*side, d.sy-7-bob // the grip, at the hand
	if limbGrip.on {                  // the jointed figure's hand; see limbdraw.go
		gx, gy = limbGrip.x, limbGrip.y
	}
	age := float64(g.frame - g.swingAt[p.ID])
	k := min(1, age/slashFrames)
	a, from, way := slashAngle(k), slashFrom, 1.0
	if g.backhand(p) {
		a, from, way = backslashAngle(k), backslashFrom, -1
	}

	// The arc first, so the blade is drawn over it: the stretch the tip has
	// swept lately, brightest just behind the blade.
	rp := tierRamps[tier]
	fade := 1.0
	if age > slashFrames {
		fade = max(0, 1-(age-slashFrames)/slashFade)
	}
	if fade > 0 && heldDefOf(p.Held) != nil { // at the canvas's own resolution; see heldhd.go
		r := heldDefOf(p.Held).reachAt() // the blade's tip: see heldhd_art.go
		swingArc(dst, gx, gy, side, a, from, way, 1.4, r*0.55, r+0.4, mix(rp[len(rp)-1], colWhite, 0.5), fade*0.85)
	} else if fade > 0 {
		tail := 1.4 // radians of arc behind the blade
		for s := 0.0; s <= 1; s += 1.0 / 48 {
			b := a - way*tail*s
			if way*(b-from) < 0 {
				break
			}
			alpha := (1 - s) * fade * 0.85
			for r := slashReach - 3; r <= slashReach+0.5; r++ {
				x := gx + math.Sin(b)*r*side
				y := gy - math.Cos(b)*r
				c := mix(rp[len(rp)-1], colWhite, 0.5)
				fillRect(dst, float32(math.Round(x)), float32(math.Round(y)), 1, 1, premul(c, alpha*(r-slashReach+4)/4.5), false)
			}
		}
	}
	if !drawHeldHD(dst, p.Held, gx, gy, a, side, 1) { // see heldhd.go
		drawAt(dst, g.art.gear.swords[tier], gx, gy, side, 1, a*side, nil)
	}
}

// premul returns c at opacity a, premultiplied as Ebitengine wants.
func premul(c color.RGBA, a float64) color.RGBA {
	a = max(0, min(1, a))
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
}
