package sim

import "math"

// Who uses which move when (the moves are in skills.go). Each kind's own
// step (dino.go, hellcreek.go, iceage.go, ankylo.go, nightstalker.go) calls
// in here once it is angry with someone: a move if one is ready, else it
// closes in, circles or waits.

// grazerFight is a grazer angry with p, dist away. Herbivores fight only
// when provoked or defending the herd: see provoke in dino.go.
func (w *World) grazerFight(c *Creature, p *Player, dist float64) {
	d := c.Kind.def()
	switch c.Kind {
	case CreatureThesc: // mostly it runs; hit close by, now and then it kicks out first
		if c.sk.call && dist <= 2 && w.random().IntN(2) == 0 && w.trySkill(c, p, dist) {
			return
		}
		c.sk.call = false
		w.fleeFrom(c, p)
	case CreatureBrachio: // too big to bother with a foe far off; close by, it stamps and lashes
		if dist > 6 {
			w.fleeFrom(c, p)
			return
		}
		c.State = CreatureChase
		w.trySkill(c, p, dist)
	case CreatureStego: // slow; it keeps its spiked tail to the foe
		c.State = CreatureChase
		if w.trySkill(c, p, dist) || dist <= StegoTailRange {
			return
		}
		if behind(c, p.X) {
			w.backTo(c, p, d.run, StegoTailRange*0.8)
			return
		}
		w.runAt(c, p.X, p.Y, d.run, StegoTailRange*0.8)
	default: // the horned ones and the mammoth: head down, a charge or a toss
		c.State = CreatureChase
		if w.trySkill(c, p, dist) {
			return
		}
		const close = 2.2
		if dist <= close {
			c.faceToward(p.X) // waiting for its horns to be ready again
			return
		}
		w.runAt(c, p.X, p.Y, d.run, close*0.8)
	}
}

// grazerWarn has a calm Torosaurus warn off a player who comes too close: its
// frill flushed, its head shaking. It reports whether it did.
func (w *World) grazerWarn(c *Creature) bool {
	if c.Kind != CreatureToro {
		return false
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	if p == nil || p.Asleep || dist > Skill(SkillToroDisplay).MaxD || p.Held() == c.Kind.TameFood() {
		return false
	}
	return w.trySkill(c, p, dist)
}

// fleeFrom has c make off from p.
func (w *World) fleeFrom(c *Creature, p *Player) {
	d := c.Kind.def()
	c.State = CreatureFlee
	c.steerAway(p.X, p.Y, 0.3)
	if !w.moveCreature(c, c.hx*d.run, c.hy*d.run) {
		c.dodge(w, p.X, p.Y)
	}
}

// backTo has c back towards p, tail first, stopping keep short of it.
func (w *World) backTo(c *Creature, p *Player, speed, keep float64) {
	dx, dy := p.X-c.X, p.Y-c.Y
	l := math.Hypot(dx, dy)
	if l <= keep || l < 1e-9 {
		return
	}
	step := min(speed, l-keep)
	face := c.FaceLeft
	w.moveCreature(c, dx/l*step, dy/l*step)
	c.FaceLeft = face
}

// rexFight is the tyrant after p, dist away.
func (w *World) rexFight(c *Creature, p *Player, dist float64) {
	if w.trySkill(c, p, dist) {
		return
	}
	const close = 2.6
	if dist <= close {
		c.faceToward(p.X)
		return
	}
	w.runAt(c, p.X, p.Y, c.Kind.def().run, close*0.8)
}

// packFight is a pack hunter after p, dist away: spread round its foe after
// its leader's screech, a move if one is ready and it is its turn, else
// circling at a little distance while the others go in. move moves it (a
// Troodon keeps out of the light). It reports whether it did any of that;
// if not, it should close in.
func (w *World) packFight(c *Creature, p *Player, dist float64, move func(dx, dy float64)) bool {
	d := c.Kind.def()
	if c.sk.flank > 0 {
		w.circleFoe(c, p, 3.4, d.run, 0.7, move)
		return true
	}
	if w.trySkill(c, p, dist) {
		return true
	}
	if dist < 4.5 && w.packBusy(c, p) {
		w.circleFoe(c, p, 3, d.run*0.8, 0.35, move)
		return true
	}
	return false
}

// circleFoe moves c round p at about r tiles, spin radians ahead on the
// ring each step (half the pack one way, half the other).
func (w *World) circleFoe(c *Creature, p *Player, r, speed, spin float64, move func(dx, dy float64)) {
	if c.ID%2 == 1 {
		spin = -spin
	}
	a := math.Atan2(c.Y-p.Y, c.X-p.X) + spin
	dx, dy := p.X+math.Cos(a)*r-c.X, p.Y+math.Sin(a)*r-c.Y
	if l := math.Hypot(dx, dy); l > 1e-6 {
		st := min(speed, l)
		move(dx/l*st, dy/l*st)
	}
	c.State = CreatureChase
	c.faceToward(p.X)
}

// pteroWants reports whether the flier c has a move ready for p: a dive, or
// a swoop on food.
func (w *World) pteroWants(c *Creature, p *Player) bool {
	cd := c.sk.cd
	return cd[0] == 0 || cd[1] == 0 && w.foodNear(p)
}

// runnerKick is an Ornithomimus just hit: close by, now and then it kicks out
// before it runs. It reports whether it did.
func (w *World) runnerKick(c *Creature) bool {
	if !c.sk.call {
		return false
	}
	c.sk.call = false
	p, dist := w.nearestPlayer(c.X, c.Y)
	if p == nil || dist > 2 || w.random().IntN(2) != 0 {
		return false
	}
	c.sk.call = true
	if w.trySkill(c, p, dist) {
		c.rage = 0
		return true
	}
	c.sk.call = false
	return false
}

// troodonsCalled answers a Troodon's cry: another pack comes out of the dark
// towards p, if not too many are about already.
func (w *World) troodonsCalled(p *Player) {
	near := 0
	for _, c := range w.fauna.creatures {
		if c.Kind == CreatureTroodon && !c.gone && math.Hypot(c.X-p.X, c.Y-p.Y) < TroodonGone {
			near++
		}
	}
	if near >= 2*TroodonPack*w.difficulty.tune().packs {
		return
	}
	r := w.random()
	for range 12 {
		a := r.Float64() * 2 * math.Pi
		d := troodonSpawnNear + r.Float64()*(troodonSpawnFar-troodonSpawnNear)
		x, y := p.X+math.Cos(a)*d, p.Y+math.Sin(a)*d
		if w.Underground(int(math.Floor(y))) != w.Underground(int(math.Floor(p.Y))) || !w.outdoors(x, y) ||
			w.inLight(x, y) || !w.creatureFits(CreatureTroodon, x, y) {
			continue
		}
		first := w.fauna.nextID + 1
		w.addHerd(CreatureTroodon, x, y, TroodonPack)
		for _, c := range w.fauna.creatures {
			if c.ID >= first {
				c.State, c.foe, c.rage = CreatureChase, p.ID, HerdRageTicks
				c.faceToward(p.X)
			}
		}
		return
	}
}
