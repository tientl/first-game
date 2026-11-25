package sim

import "math"

// Telegraphed blows. A beast that strikes first draws back for a moment, as
// the client draws it (dinoanim.go): it rears its head, drops its horns,
// coils its tail or crouches. Only when that is done does the blow land, and
// only on a player still in reach: one who sees it coming and steps back is
// missed. A beast knocked out of its strike while drawing back loses the blow.

// BlowSlack is how much further than at the start of the wind-up a player
// may be and still be hit: the blow follows a little.
const BlowSlack = 0.45

// windupTicks is how long each kind draws back before its blow lands.
var windupTicks = [NumCreatureKinds]int{
	CreatureRex:     10, // rears its head
	CreatureTrike:   9,  // drops its horns
	CreatureToro:    9,
	CreatureMammoth: 9,
	CreatureStego:   9, // coils its tail
	CreaturePlesio:  9, // rears its neck
	CreatureQuetzal: 8, // draws its beak back
	CreatureAnzu:    7, // leans back to kick
	CreatureRaptor:  6, // crouches to spring
	CreatureAchero:  6,
}

// WindupTicks is how long a beast of kind k draws back before its blow lands:
// 0 for one that strikes at once.
func WindupTicks(k CreatureKind) int {
	if k >= NumCreatureKinds {
		return 0
	}
	return windupTicks[k]
}

// pendingBlow is a blow drawn back, waiting to land.
type pendingBlow struct {
	ticks int // left to wait; 0: none pending
	dmg   float64
	foe   PlayerID
	reach float64
}

// strike starts c's blow at p: at once for a beast with no wind-up, else
// after it has drawn back.
func strike(c *Creature, p *Player, dmg float64, cd int) {
	wind := WindupTicks(c.Kind)
	c.State, c.timer, c.biteCD = CreatureStrike, AttackTicks+wind, cd
	if wind == 0 {
		land(c, p, dmg)
		return
	}
	c.blow = pendingBlow{ticks: wind, dmg: dmg, foe: p.ID, reach: math.Hypot(p.X-c.X, p.Y-c.Y) + BlowSlack}
}

func land(c *Creature, p *Player, dmg float64) {
	struck(p, dmg)
	p.notify(NoticeBitten, uint8(c.Kind))
}

// stepBlow counts down c's drawn-back blow and lands it.
func (w *World) stepBlow(c *Creature) {
	b := &c.blow
	if b.ticks <= 0 {
		return
	}
	striking := c.State == CreatureStrike || c.State == CreatureHurt && c.after == CreatureStrike
	if !striking {
		*b = pendingBlow{} // knocked out of it
		return
	}
	if b.ticks--; b.ticks > 0 {
		return
	}
	p := w.players[b.foe]
	if p == nil || p.Dead || p.Asleep || math.Hypot(p.X-c.X, p.Y-c.Y) > b.reach {
		return // it saw it coming and stepped out of reach
	}
	land(c, p, b.dmg)
}

// windingUp reports whether c is still drawing back its blow.
func (c *Creature) windingUp() bool { return c.blow.ticks > 0 }
