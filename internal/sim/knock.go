package sim

import "math"

// Knockback. A blow throws what it lands on back along it, eased: the most
// of the way at once, the rest over the next few ticks, so a throw reads as
// a shove and not a jump. A player thrown by a great beast (shove) goes as
// far as before, only not all in one tick; a beast struck by a player is
// pushed a little, less the bigger it is, and not at all a boss.
//
// The beast pushed back comes on again: see riposte.go.

// Knockback tuning.
const (
	KnockEase   = 0.5  // share of what is left of a throw covered each tick
	KnockFirst  = 0.45 // share of a player's throw covered the tick it lands
	knockStop   = 0.01 // tiles: what is left of a throw below this is dropped
	knockMassR  = 0.4  // a beast of this radius or less is thrown the whole way
	knockCritK  = 1.4  // a critical blow throws this much further
	FistKnock   = 0.12 // tiles a punch throws a small beast
	BladeKnock  = 0.3  // a knife or a tool
	SpearKnock  = 0.4  // a spear's thrust
	HeavyKnock  = 0.7  // the battle axe
	LegendKnock = 0.5  // a boss's weapon
)

// WeaponKnock is how far a blow with held throws a small beast, in tiles.
func WeaponKnock(held ItemKind) float64 {
	if _, _, ok := legendaryBlow(held); ok {
		return LegendKnock
	}
	switch held {
	case ItemSpear, ItemIvorySpear:
		return SpearKnock
	case ItemBattleAxe:
		return HeavyKnock
	}
	if Punches(held) {
		return FistKnock
	}
	return BladeKnock
}

// KnockShare is the share of a throw a beast of kind k takes: all of it for
// one of knockMassR or smaller, less the bigger it is, none for a boss.
func KnockShare(k CreatureKind) float64 {
	if k.Boss() {
		return 0
	}
	r := k.def().radius
	if r <= knockMassR {
		return 1
	}
	s := math.Pow(knockMassR/r, 1.5)
	if k.def().tough {
		s *= 0.6 // planted on four great legs
	}
	return s
}

// knockBeast throws c d tiles away from (x, y), eased over its next steps.
func knockBeast(c *Creature, x, y, d float64) {
	d *= KnockShare(c.Kind)
	if d <= knockStop {
		return
	}
	dx, dy := c.X-x, c.Y-y
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		return
	}
	c.kx += dx / l * d
	c.ky += dy / l * d
}

// stepJostle is the start of c's step: a parried beast starts to reel, and
// a thrown one goes on along its throw. It is not turned round by it.
func (w *World) stepJostle(c *Creature) {
	if c.reel > 0 {
		c.State, c.timer = CreatureHurt, c.reel
		c.after, c.afterT = CreatureIdle, 0
		c.blow = pendingBlow{}
		c.flinchCD = c.reel + FlinchEvery
		c.reel = 0
	}
	if c.kx == 0 && c.ky == 0 {
		return
	}
	sx, sy := c.kx*KnockEase, c.ky*KnockEase
	if math.Hypot(c.kx, c.ky) < 3*knockStop {
		sx, sy = c.kx, c.ky
	}
	if w.creatureFits(c.Kind, c.X+sx, c.Y) {
		c.X += sx
		c.kx -= sx
	} else {
		c.kx = 0
	}
	if w.creatureFits(c.Kind, c.X, c.Y+sy) {
		c.Y += sy
		c.ky -= sy
	} else {
		c.ky = 0
	}
	if math.Hypot(c.kx, c.ky) < knockStop {
		c.kx, c.ky = 0, 0
	}
}

// stepKnock carries p on along a throw.
func (w *World) stepKnock(p *Player) {
	if p.knockX == 0 && p.knockY == 0 {
		return
	}
	if p.Dead || p.mount != 0 || p.boat != 0 {
		p.knockX, p.knockY = 0, 0
		return
	}
	w.knockMove(p, KnockEase)
}

// knockMove moves p share of what is left of its throw, as far as the
// ground lets it.
func (w *World) knockMove(p *Player, share float64) {
	can := w.standable
	if w.style >= StyleSeas {
		can = w.swimmable
	}
	sx, sy := p.knockX*share, p.knockY*share
	if math.Hypot(p.knockX, p.knockY) < 3*knockStop {
		sx, sy = p.knockX, p.knockY
	}
	if can(p.X+sx, p.Y) {
		p.X += sx
		p.knockX -= sx
	} else {
		p.knockX = 0
	}
	if can(p.X, p.Y+sy) {
		p.Y += sy
		p.knockY -= sy
	} else {
		p.knockY = 0
	}
	if math.Hypot(p.knockX, p.knockY) < knockStop {
		p.knockX, p.knockY = 0, 0
	}
}
