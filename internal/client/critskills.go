package client

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The moves (sim/skills.go) of the beasts not drawn on a rig: the mammoth and
// the sabre-tooth, drawn in code (critbeast.go), whose skill poses are more
// poses of their drawings, and the flying reptile, whose dive is told by its
// shadow.

// pteroDiveHold is how high, in art pixels, a diver holds over its foe: high
// enough that its shadow lies clear on the ground under it.
const pteroDiveHold = 58.0

// More poses of the mammoth and the sabre-tooth, past the ones the art
// holds; critter draws them the first time they are shown.
const (
	mammothSkHeadDown = mammothPoses + iota // head down, ears flared, a forefoot scraping
	mammothSkCurl                           // the trunk curled up high
	mammothSkSwipe                          // the trunk lashed out forwards
	mammothSkTrumpet                        // the trunk raised
)

const (
	sabreSkLow  = sabrePoses + iota // flat to the ground, the haunches twitching
	sabreSkGape                     // the jaws open very wide
	sabreSkBite                     // the bite
)

// The mammoth's trunk curled up over its brow, and lashed out straight.
var (
	trunkCoil = [6]float64{1.4, 2.4, 3.3, 4.2, 5.0, 5.6}
	trunkLash = [6]float64{1.05, 1.3, 1.45, 1.55, 1.6, 1.8}
)

// critterSkill is skillSprite for the mammoth and the sabre-tooth, ts
// ticks into a phase n long.
func (g *Game) critterSkill(c protocol.CreatureView, side int, ts, n float64, f int) (sprite, float64, bool) {
	u := ts / max(1, n)
	fr := func(kind string, pose, fr int) (sprite, float64, bool) { return critter(kind, side, pose, fr), 0, true }
	switch c.Skill {
	case sim.SkillMammothCharge:
		switch c.Phase {
		case sim.PhaseWindup:
			if u < 0.25 {
				return fr("mammoth", mammothCharge, 0)
			}
			return fr("mammoth", mammothSkHeadDown, (f/6)%3)
		case sim.PhaseActive:
			return fr("mammoth", mammothCharge, cyc(f, 4))
		}
		if u < 0.4 {
			return fr("mammoth", mammothSkHeadDown, 0)
		}
		return fr("mammoth", mammothWalkA, 0)
	case sim.SkillMammothSwipe:
		switch {
		case c.Phase == sim.PhaseWindup && u < 0.3:
			return fr("mammoth", mammothGore, 1)
		case c.Phase == sim.PhaseWindup:
			return fr("mammoth", mammothSkCurl, 0)
		case c.Phase == sim.PhaseActive || u < 0.35:
			return fr("mammoth", mammothSkSwipe, 0)
		}
		return fr("mammoth", mammothWalkA, 0)
	case sim.SkillMammothTrumpet:
		if c.Phase == sim.PhaseRecover && u > 0.4 {
			return fr("mammoth", mammothWalkA, 0)
		}
		if c.Phase == sim.PhaseWindup && u < 0.3 {
			return fr("mammoth", mammothGore, 3)
		}
		return fr("mammoth", mammothSkTrumpet, 0)
	case sim.SkillSabrePounce:
		switch c.Phase {
		case sim.PhaseWindup:
			if u < 0.3 {
				return fr("sabre", sabreStalk, 0)
			}
			return fr("sabre", sabreSkLow, (f/5)%2)
		case sim.PhaseActive: // in the air, claws out
			return critter("sabre", side, sabreSpring, 4), 7 * math.Sin(math.Pi*quant(u, 6)), true
		}
		if u < 0.4 {
			return fr("sabre", sabreSkBite, 0)
		}
		return fr("sabre", sabreStalk, 0)
	case sim.SkillSabreBite:
		switch {
		case c.Phase == sim.PhaseWindup && u < 0.3:
			return fr("sabre", sabreSpring, 1)
		case c.Phase == sim.PhaseWindup:
			return fr("sabre", sabreSkGape, 0)
		case c.Phase == sim.PhaseActive || u < 0.35:
			return fr("sabre", sabreSkBite, 0)
		}
		return fr("sabre", sabreStalk, 0)
	}
	return sprite{}, 0, false
}

// pteroSkill is skillSprite for the flying reptile: its dive and its swoop.
// Wound up, it holds still high overhead (drawSky draws it, its shadow
// shrinking as it gathers itself: skyShadow); then it drops, pecks low and
// flaps up again.
func (g *Game) pteroSkill(c protocol.CreatureView, side int, ts, n float64, f int) (sprite, float64, bool) {
	u := clamp01(ts / max(1, n))
	a := &g.art.beasts
	switch c.Phase {
	case sim.PhaseWindup: // high over its foe, wings beating to hold still, its shadow on them
		alt := pteroDiveHold - 6*easeInOut(u)
		if (f/6)%2 == 0 {
			return rigSprite("pteroFly", side, rigPose{squash: 0.8, wings: 0.12}), alt, true
		}
		return rigSprite("pteroFly", side, rigPose{squash: 0.9, wings: -0.15}), alt, true
	case sim.PhaseActive: // the drop: on its foe in a blink
		return a.ptero[side][pteroDive], (pteroDiveHold - 6) * (1 - u) * (1 - u), true
	}
	return rigSprite("pteroFly", side, flapPose(f/2)), 2 + pteroDiveAlt*u, true
}

// skyShadow is how big and how dark a flier's shadow is drawn: a diver's
// shrinks and darkens as it gathers itself overhead, then holds still.
func skyShadow(c protocol.CreatureView, ts float64) (scale, alpha float64) {
	if c.Kind != sim.CreaturePtero || c.Phase != sim.PhaseWindup {
		return 1, 0.6
	}
	u := clamp01(ts / max(1, float64(c.PhaseLen)))
	return 2.8 - 1.3*easeInOut(min(1, u*1.4)), 1 + 2*u // wide and faint as it gathers, then small, dark and still
}
