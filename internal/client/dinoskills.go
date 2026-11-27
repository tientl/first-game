package client

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The beasts' moves drawn (sim/skills.go). Every move has three beats, each
// of them a pose of the beast's rig (dinorig.go) or of its code-drawn body
// (critbeast.go):
//
//   - the wind-up: an anticipation that reads at a glance across the screen,
//     eased into over its first third and then held, often with a fidget (the
//     hips wiggling, a forefoot scraping, a head shaking, a tail twitching)
//     so a held pose still looks alive and about to go;
//   - the strike: snapped into in a frame or two, held through the active
//     ticks; a charge runs in its strike pose, a leap arcs off the ground;
//   - the recovery: held a moment, then eased back to standing: the opening.
//
// Frames are poses of the rig, drawn the first time each is shown and kept
// (rigFrames); the in-betweens are quantised so a move costs a dozen frames.

// skillLook is how one move is drawn on a rig.
type skillLook struct {
	wind, wind2 rigPose // drawn back, and the other end of a fidget held there (zero: none)
	fidget      int     // frames each end of the fidget is held
	strike      rigPose // the blow
	strike2     rigPose // a second blow, alternating through the active ticks (two slashes, pecks)
	recover     rigPose // the pose the recovery starts from; zero: the strike's
	run         bool    // the active phase is a charge: a stride in the strike pose
	leap        float64 // the active phase is a leap: this many art pixels up at its height
	shake       float64 // art pixels the body trembles through the active phase (a roar)
	flush       bool    // reddens through the wind-up (a frill, plates, a crest)
	dust        bool    // kicks up dust while it winds up (a scraping charge)
}

// skillLooks is every rig-drawn move.
var skillLooks = map[sim.SkillID]skillLook{
	// The tyrant.
	sim.SkillRexBite: {
		wind:   rigPose{head: -0.5, open: 0.42, tilt: -0.1, squash: 0.94, tail: -0.12, step: -0.4},
		strike: rigPose{head: 0.32, open: 0.12, tilt: 0.12, step: 0.7, tail: 0.08},
	},
	sim.SkillRexRoar: {
		wind:   rigPose{head: -0.3, tilt: -0.28, open: 0.1, tail: -0.2, step: -0.3},
		strike: rigPose{head: -0.62, open: 0.6, tilt: -0.16, tail: 0.12},
		shake:  1,
	},
	sim.SkillRexTail: {
		wind:   rigPose{tail: 0.42, sweep: -0.35, tilt: 0.08, head: 0.2, step: -0.6},
		strike: rigPose{tail: -0.2, sweep: 0.7, tilt: -0.04, head: -0.1, step: 0.4},
	},
	sim.SkillRexCharge: {
		wind:    rigPose{head: 0.5, tilt: 0.14, squash: 0.92, tail: 0.14, step: -0.7, open: 0.15},
		wind2:   rigPose{head: 0.5, tilt: 0.14, squash: 0.92, tail: 0.14, step: -0.2, open: 0.15},
		fidget:  8,
		strike:  rigPose{head: 0.42, tilt: 0.12, open: 0.25, tail: 0.1},
		recover: rigPose{head: -0.12, open: 0.32, squash: 0.95},
		run:     true,
		dust:    true,
	},
	// The raptors.
	sim.SkillDakotaPounce: {
		wind:    rigPose{squash: 0.7, head: 0.25, tail: 0.34, step: 0.5, tilt: 0.06},
		wind2:   rigPose{squash: 0.72, head: 0.22, tail: 0.24, step: 0.35, tilt: -0.02},
		fidget:  4,
		strike:  rigPose{step: 1, tilt: -0.3, open: 0.45, lift: 3},
		recover: rigPose{squash: 0.82, head: 0.5, open: 0.35, tilt: 0.18},
		leap:    6,
	},
	sim.SkillDakotaSickle: {
		wind:    rigPose{cycle: 0.3, step: 1.8, tilt: -0.14, tail: 0.18, squash: 0.96},
		strike:  rigPose{cycle: 0.55, step: 2, tilt: 0.1, head: 0.15, open: 0.3},
		strike2: rigPose{cycle: 0.8, step: 2, tilt: 0.04, head: 0.1, open: 0.2},
	},
	sim.SkillDakotaScreech: {
		wind:   rigPose{head: -0.4, open: 0.3, tilt: -0.12, tail: 0.12},
		strike: rigPose{head: -0.5, open: 0.62, tilt: -0.14, tail: 0.18},
		shake:  0.5,
	},
	sim.SkillAcheroSnap: {
		wind:    rigPose{head: 0.75, squash: 0.76, tail: 0.2, step: 0.3},
		strike:  rigPose{head: 0.05, open: 0.5, tilt: 0.1, step: 0.9},
		recover: rigPose{step: -0.7, tilt: -0.12, head: -0.1},
	},
	// The night hunter (its frames carry its eye: troodon.go).
	sim.SkillTroodonAmbush: {
		wind:    rigPose{squash: 0.62, head: 0.42, tail: 0.14, step: 0.6},
		wind2:   rigPose{squash: 0.64, head: 0.4, tail: 0.04, step: 0.6},
		fidget:  10,
		strike:  rigPose{step: 1.2, tilt: -0.22, head: 0.15, open: 0.45, tail: -0.1},
		recover: rigPose{squash: 0.85, head: 0.2},
		leap:    4,
	},
	sim.SkillTroodonVenom: {
		wind:   rigPose{squash: 0.74, head: 0.3, open: 0.38, tail: 0.16, step: 0.4},
		wind2:  rigPose{squash: 0.76, head: 0.26, open: 0.3, tail: 0.1, step: 0.4},
		fidget: 3,
		strike: rigPose{step: 1.2, tilt: 0.16, head: 0.3, open: 0.5, tail: 0.12},
	},
	sim.SkillTroodonCall: {
		wind:   rigPose{head: -0.45, open: 0.2, tilt: -0.12, tail: 0.1},
		strike: rigPose{head: -0.55, open: 0.6, tilt: -0.14, tail: 0.14},
		shake:  0.5,
	},
	// The walking pterosaur.
	sim.SkillQuetzalStab: {
		wind:   rigPose{neck: -0.38, head: -0.4, tilt: -0.08, step: -0.4},
		strike: rigPose{neck: 0.3, head: -0.15, open: 0.18, tilt: 0.06, step: 0.6},
	},
	sim.SkillQuetzalBuffet: {
		wind:   rigPose{tilt: -0.3, neck: -0.32, head: -0.25, lift: 3, open: 0.2},
		strike: rigPose{tilt: 0.1, neck: 0.1, head: 0.1, open: 0.3, step: 0.6},
	},
	// The sea reptile.
	sim.SkillPlesioNeck: {
		wind:   rigPose{head: -0.45, open: 0.18},
		strike: rigPose{head: 0.75, open: 0.45},
	},
	sim.SkillPlesioAmbush: {
		strike: rigPose{head: -0.25, open: 0.55, lift: 4},
	},
	// The horned ones.
	sim.SkillTrikeCharge: {
		wind:    rigPose{head: 0.58, squash: 0.92, step: -0.55, tail: 0.12},
		wind2:   rigPose{head: 0.62, squash: 0.92, step: -0.15, tail: 0.12},
		fidget:  7,
		strike:  dinoPoses["trike"][trikeCharge],
		recover: rigPose{head: -0.15, squash: 0.97, open: 0.2},
		run:     true,
		dust:    true,
	},
	sim.SkillTrikeToss: {
		wind:   rigPose{head: 0.62, squash: 0.95, step: -0.3},
		strike: rigPose{head: -0.48, open: 0.3, tilt: -0.06, step: 0.5},
	},
	sim.SkillTrikeCall: {
		wind:   rigPose{head: -0.3, open: 0.15},
		strike: rigPose{head: -0.45, open: 0.45},
	},
	sim.SkillToroDisplay: {
		wind:   rigPose{head: 0.1, tilt: -0.03},
		wind2:  rigPose{head: 0.3, tilt: 0.03},
		fidget: 5,
		strike: rigPose{head: 0.2},
		flush:  true,
	},
	sim.SkillToroCharge: {
		wind:    rigPose{head: 0.6, squash: 0.91, step: -0.6, tail: 0.14},
		wind2:   rigPose{head: 0.64, squash: 0.91, step: -0.15, tail: 0.14},
		fidget:  7,
		strike:  dinoPoses["trike"][trikeCharge],
		recover: rigPose{head: -0.15, squash: 0.97, open: 0.2},
		run:     true,
		dust:    true,
	},
	// The plate-back.
	sim.SkillStegoFlush: {
		wind:   rigPose{head: 0.2, tail: 0.12, squash: 0.97},
		strike: rigPose{head: 0.1, tail: 0.2},
		flush:  true,
	},
	sim.SkillStegoTail: {
		wind:   rigPose{tail: 0.85, squash: 0.94, head: 0.15, step: -0.3},
		strike: rigPose{tail: -0.35, sweep: 0.9, squash: 0.97},
	},
	// The long-neck.
	sim.SkillBrachioStomp: {
		wind:    rigPose{tilt: -0.3, lift: 9, head: -0.1},
		strike:  rigPose{tilt: 0.04, squash: 0.92, neck: 0.2, head: 0.1},
		recover: rigPose{squash: 0.95, neck: 0.1},
	},
	sim.SkillBrachioWhip: {
		wind:   rigPose{tail: 0.38, sweep: -0.4, neck: -0.1},
		strike: rigPose{tail: -0.1, sweep: 0.75, neck: 0.05},
	},
	// The armoured one: its club is drawn by ankylo.go.
	sim.SkillAnkyloSlam: {
		wind:   rigPose{tail: 0.62, squash: 0.94, head: 0.15},
		strike: rigPose{tail: -0.28, squash: 0.88, head: 0.05},
	},
	sim.SkillAnkyloTurtle: {
		wind:   rigPose{head: 0.45, squash: 0.9},
		strike: rigPose{head: 0.6, squash: 0.86, tail: -0.05},
	},
	// The crested one.
	sim.SkillAnzuKick: {
		wind:   rigPose{tilt: -0.16, head: -0.25, step: -1.2, squash: 0.97},
		strike: dinoPoses["anzu"][anzuKick],
	},
	sim.SkillAnzuDisplay: {
		wind:    rigPose{head: -0.32, open: 0.42, tilt: -0.06},
		wind2:   rigPose{head: -0.26, open: 0.36, tilt: -0.08},
		fidget:  4,
		strike:  rigPose{head: 0.5, open: 0.25, tilt: 0.06},
		strike2: rigPose{head: 0.15, open: 0.1},
		flush:   true,
	},
	// The runners: a flinch, a kick, then away.
	sim.SkillThescKick: {
		wind:   rigPose{tilt: 0.14, step: -0.8, head: 0.1},
		strike: rigPose{tilt: -0.08, step: 1.8},
	},
	sim.SkillOrnithoKick: {
		wind:   rigPose{tilt: 0.12, step: -0.8, neck: 0.15},
		strike: rigPose{tilt: -0.08, step: 1.8, neck: -0.1},
	},
}

// skillRigs is which rig draws each kind's moves.
var skillRigs = map[sim.CreatureKind]string{
	sim.CreatureRex: "rex", sim.CreatureRaptor: "dakota", sim.CreatureAchero: "achero", sim.CreatureTroodon: "troodon",
	sim.CreatureQuetzal: "quetzal", sim.CreaturePlesio: "plesio", sim.CreatureTrike: "trike", sim.CreatureToro: "toro",
	sim.CreatureStego: "stego", sim.CreatureBrachio: "brachio", sim.CreatureAnkylo: "ankylo", sim.CreatureAnzu: "anzu",
	sim.CreatureThesc: "thesc", sim.CreatureOrnitho: "ornitho",
}

// skillClock is when a creature was last seen at a tick of its move.
type skillClock struct {
	id    sim.SkillID
	ph    sim.SkillPhase
	t     uint8
	frame int
}

var skillClocks = map[sim.CreatureID]skillClock{}

// skillTime is how far c is into the phase of its move, in ticks, with the
// frames since the snapshot added: it moves on smoothly between ticks.
func (g *Game) skillTime(c protocol.CreatureView) float64 {
	k, ok := skillClocks[c.ID]
	if !ok || k.id != c.Skill || k.ph != c.Phase || k.t != c.PhaseT || g.frame < k.frame {
		if len(skillClocks) > 256 {
			clear(skillClocks)
		}
		k = skillClock{c.Skill, c.Phase, c.PhaseT, g.frame}
		skillClocks[c.ID] = k
	}
	return float64(c.PhaseT) + min(0.99, float64(g.frame-k.frame)*sim.TicksPerSecond/60)
}

// quant rounds v (0 to 1) down to steps steps, so a move has few frames.
func quant(v float64, steps int) float64 {
	return math.Floor(clamp01(v)*float64(steps)+1e-9) / float64(steps)
}

// easeInOut is smoothstep.
func easeInOut(v float64) float64 { v = clamp01(v); return v * v * (3 - 2*v) }

// skillPose is the rig pose of move look at time ts ticks into phase ph of
// length n; f is the frame count, for a fidget. lift is how far a leap has
// carried it off the ground, in art pixels.
func skillPose(look skillLook, ph sim.SkillPhase, ts, n float64, f, every int) (p rigPose, lift float64) {
	u := ts / max(1, n)
	switch ph {
	case sim.PhaseWindup:
		e := easeInOut(quant(u/0.34, 4)) // into the pose over the first third, then held
		p = lerpPose(rigPose{}, look.wind, e)
		if e >= 1 && look.wind2 != (rigPose{}) && (f/max(1, look.fidget))%2 == 1 {
			p = look.wind2
		}
		return p, 0
	case sim.PhaseActive:
		switch {
		case look.leap > 0:
			lift = look.leap * math.Sin(math.Pi*quant(u, 6))
			return lerpPose(look.wind, look.strike, quant(min(1, (ts+1)/2), 2)), lift
		case look.run:
			return stridePose(look.strike, f/3, 1.6), 0
		}
		if ts < 1 { // the snap: a frame between drawn back and struck
			return lerpPose(look.wind, look.strike, 0.5), 0
		}
		if look.strike2 != (rigPose{}) && every > 0 && (int(ts)/every)%2 == 1 {
			return look.strike2, 0
		}
		return look.strike, 0
	case sim.PhaseRecover:
		from := look.recover
		if from == (rigPose{}) {
			from = look.strike
		}
		if u < 0.3 {
			return from, 0 // held: the opening
		}
		return lerpPose(from, rigPose{}, easeInOut(quant((u-0.3)/0.7, 4))), 0
	}
	return rigPose{}, 0
}

// skillSprite is creatureSprite for a beast in the middle of a move; ok is
// false for one that is not, or whose move this does not draw.
func (g *Game) skillSprite(c protocol.CreatureView, side int) (sprite, float64, bool) {
	if c.Skill == sim.SkillNone || c.Phase == sim.PhaseNone || c.Tamed {
		return sprite{}, 0, false
	}
	ts, n := g.skillTime(c), float64(c.PhaseLen)
	f := g.frame + int(c.ID)*7
	switch c.Kind {
	case sim.CreatureMammoth, sim.CreatureSabre:
		return g.critterSkill(c, side, ts, n, f) // see critbeast.go
	case sim.CreaturePtero:
		return g.pteroSkill(c, side, ts, n, f)
	case sim.CreatureAnkylo:
		if c.Skill == sim.SkillAnkyloClub {
			return ankyloClubSkill(c, side, ts), 0, true // see ankylo.go
		}
	case sim.CreaturePlesio:
		if c.Skill == sim.SkillPlesioAmbush && c.Phase == sim.PhaseWindup {
			return sprite{}, 0, false // under the water: drawn as a ripple coming closer
		}
	}
	rig, ok := skillRigs[c.Kind]
	look, known := skillLooks[c.Skill]
	if !ok || !known {
		return sprite{}, 0, false
	}
	p, lift := skillPose(look, c.Phase, ts, n, f, sim.Skill(c.Skill).Every)
	return rigSprite(rig, side, p), lift, true
}

// skillFrame is skillSprite for the night hunter, with where its eye is.
func (g *Game) skillFrame(c protocol.CreatureView, side int) (rigFrame, float64, bool) {
	look, ok := skillLooks[c.Skill]
	if !ok || c.Skill == sim.SkillNone || c.Phase == sim.PhaseNone {
		return rigFrame{}, 0, false
	}
	p, lift := skillPose(look, c.Phase, g.skillTime(c), float64(c.PhaseLen), g.frame+int(c.ID)*7, sim.Skill(c.Skill).Every)
	return rigFrameOf("troodon", side, p), lift, true
}

// skillTint is how a beast making a move is drawn over: reddening as a
// frill or plates flush, trembling through a roar. It reports false when
// there is nothing to change.
func (g *Game) skillTint(c protocol.CreatureView) (dx float64, r, gb float32, ok bool) {
	look, known := skillLooks[c.Skill]
	if !known || c.Skill == sim.SkillNone {
		return 0, 1, 1, false
	}
	r, gb = 1, 1
	if look.flush && c.Phase == sim.PhaseWindup {
		u := clamp01(g.skillTime(c) / max(1, float64(c.PhaseLen)))
		pulse := 0.5 + 0.5*math.Sin(float64(g.frame)*0.35)
		k := float32(u * (0.55 + 0.45*pulse))
		r, gb = 1+0.25*k, 1-0.42*k
		ok = true
	}
	if look.shake > 0 && c.Phase == sim.PhaseActive {
		dx = math.Round(look.shake * math.Sin(float64(g.frame)*2.3))
		ok = true
	}
	return dx, r, gb, ok
}

// ankyloClubSkill is the armoured one's club, ts ticks into the phase of its
// move: the blow of ankylo.go, timed from the start of the move.
func ankyloClubSkill(c protocol.CreatureView, side int, ts float64) sprite {
	d := sim.Skill(sim.SkillAnkyloClub)
	switch c.Phase {
	case sim.PhaseActive:
		ts += float64(d.Windup)
	case sim.PhaseRecover:
		ts += float64(d.Windup + d.Active)
	}
	return ankyloBlow(side, int(ts*60/sim.TicksPerSecond), false)
}
