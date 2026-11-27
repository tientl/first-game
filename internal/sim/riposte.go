package sim

import "math"

// The riposte: no free blows. A player could strike a beast once and step
// back out of its reach before it answered, again and again, and fight a
// tyrant without a scratch. Now a beast that fights, struck by a player and
// not badly hurt, comes straight back at them once it has shrugged the blow
// off: for RiposteTicks it closes in faster than a player walks
// (RiposteSpeed), until it is in reach of its own blow; from there its own
// way of fighting takes over (its wind-up, its bite). Standing to trade
// blows changes nothing; stepping back only buys the time of its wind-up,
// which a raised shield can use (shield.go).

// Riposte tuning.
const (
	RiposteTicks  = 3 * TicksPerSecond / 2 // how long a struck beast keeps coming
	RiposteSpeed  = 1.6                    // times a player's walking pace, at least
	riposteRunK   = 1.8                    // times its own run, if that is faster
	riposteHurtAt = 4                      // below a quarter of its health it does not come back
)

// ripostes reports whether a beast of kind k comes back at whoever strikes
// it: the fighters and the hunters on land, but not the armoured one, which
// turns its club on them instead (ankylo.go), nor the sea's or the sky's.
func ripostes(k CreatureKind) bool {
	if k.Boss() {
		return false
	}
	switch k {
	case CreatureAnkylo, CreaturePlesio, CreatureMosa, CreatureKraken, CreaturePtero, CreatureFish:
		return false
	case CreatureRaptor, CreatureAchero, CreatureRex, CreatureSabre, CreatureBear, CreatureTroodon,
		CreatureScorpion, CreatureSpider, CreatureTribesman, CreatureAnzu, CreatureQuetzal, CreatureSnake:
		return true
	}
	return k.def().fights
}

// riposteReach is how close c comes before its own blow takes over.
func riposteReach(c *Creature) float64 { return max(0.9, c.Kind.def().reach*0.85) }

// struckBy is what follows p's blow on c, which did dmg: c is thrown back
// (knock.go) and, if it fights, readies its riposte.
func (w *World) struckBy(p *Player, c *Creature, crit bool) {
	if c.gone || c.HP <= 0 {
		return
	}
	d := WeaponKnock(p.Held())
	if crit {
		d *= knockCritK
	}
	knockBeast(c, p.X, p.Y, d)
	if !ripostes(c.Kind) || c.Tamed() || c.HP*riposteHurtAt < c.Kind.MaxHP() {
		return
	}
	c.riposte, c.ripFoe, c.ripChased = RiposteTicks, p.ID, false
}

// stepRiposte runs c's riposte, once it is over its flinch; it reports
// whether it moved c this tick (and so c's own step is skipped).
func (w *World) stepRiposte(c *Creature) bool {
	if c.riposte <= 0 {
		return false
	}
	c.riposte--
	p := w.players[c.ripFoe]
	switch {
	case p == nil || p.Dead || p.Asleep || p.mount != 0 || p.boat != 0 || c.Tamed(),
		c.State == CreatureCower, c.State == CreatureTrapped, c.State == CreatureHidden,
		c.HP*riposteHurtAt < c.Kind.MaxHP():
		c.riposte, c.ripChased = 0, false
		return false
	}
	dist := math.Hypot(p.X-c.X, p.Y-c.Y)
	reach := riposteReach(c)
	speed := max(c.Kind.def().run*riposteRunK, PlayerSpeed*RiposteSpeed)
	if c.State == CreatureStrike || c.windingUp() {
		// Its blow comes with it: it lunges as it strikes, so stepping back
		// while it winds up does not take them out of reach.
		if dist > reach*0.6 {
			step := min(speed, dist-reach*0.6)
			w.lunge(c, p, dist, step)
		}
		if b := &c.blow; b.ticks > 0 && b.foe == p.ID {
			b.reach = max(b.reach, math.Hypot(p.X-c.X, p.Y-c.Y)+BlowSlack)
		}
		return false
	}
	if dist <= reach {
		if c.ripChased {
			c.biteCD = 0 // it came after them to strike: it does, at once
		}
		return false // its own way of fighting takes over
	}
	c.ripChased = true
	c.foe, c.rage = p.ID, max(c.rage, 6*TicksPerSecond)
	c.State = CreatureChase
	w.lunge(c, p, dist, min(speed, dist-reach*0.8))
	return true
}

// lunge moves c step tiles towards p, dist away, facing them.
func (w *World) lunge(c *Creature, p *Player, dist, step float64) {
	left := c.FaceLeft
	if !w.moveCreature(c, (p.X-c.X)/dist*step, (p.Y-c.Y)/dist*step) && c.State == CreatureChase {
		c.dodge(w, p.X, p.Y)
	}
	c.FaceLeft = left
	c.faceToward(p.X)
}
