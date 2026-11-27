package sim

import "math"

// The round wooden shield (Khiên gỗ tròn): boards glued edge to edge, a
// rawhide rim and a wooden boss over the grip. Like armour it works from
// anywhere in the bag; it hangs on the back until it is raised (Input.Guard:
// the right button when the hand has nothing to use it for, or Q). Raised:
//
//   - a blow from the front loses GuardShare of its force; the player does
//     not start back (no jolt), but is pushed back a little;
//   - raised no more than ParryTicks before the blow lands it is a parry:
//     the blow of a light beast is turned aside whole and the beast reels
//     (ParryStagger); a big beast or a boss is only blocked;
//   - the player walks at GuardSlow, keeps facing the way they faced (to
//     back off behind it), turns to the pointer, and strikes no blows.
//
// A blow from behind or the side goes past the shield as before.

// Shield tuning.
const (
	GuardShare   = 0.75               // the share of a frontal blow a raised shield stops
	ParryTicks   = TicksPerSecond / 4 // a shield raised this recently parries
	GuardSlow    = 0.55               // walking speed behind a raised shield
	GuardCover   = 0.34               // cosine of how far either side of the facing the shield covers (about 70 degrees)
	ParryStagger = TicksPerSecond     // how long a parried beast reels
	GuardKnock   = 0.4                // share of a throw that still gets through a raised shield
	BlockPush    = 0.25               // tiles a blocked blow pushes the player back
	guardFoeNear = 4.0                // raising it, the player turns to a beast after them this close
)

func init() {
	itemDefs[ItemShield] = ItemDef{Name: "Khiên gỗ tròn", Stack: 1, Tool: true}
	recipes[RecipeShield] = Recipe{
		ID: RecipeShield, Name: itemDefs[ItemShield].Name,
		Inputs: []Ingredient{{Item: ItemWood, Count: 4}, {Item: ItemHide, Count: 1}, {Item: ItemRope, Count: 1}},
		Gives:  ItemShield,
	}
	for _, n := range []string{"shield", "khien", "khiengo", "roundshield"} {
		devItemAliases[n] = ItemShield
	}
}

// guard is a player's shield, while it is up.
type guard struct {
	up    bool
	since uint64 // the tick it went up
	seq   uint8  // goes up with every blow taken on it: the client sparks
	parry bool   // the last of those was a parry
	at    uint64 // the tick of the last
}

// blowSrc is the beast whose turn it is: a blow landing on a player while
// it steps is its blow. Every player points at the world's.
type blowSrc struct {
	c    *Creature
	tick uint64
}

// HasShield reports whether p carries a shield.
func (p *Player) HasShield() bool { return p.Inv.Count(ItemShield) > 0 }

// Guarding reports whether p holds a shield up.
func (p *Player) Guarding() bool { return p.guard.up }

// GuardSeq goes up with every blow p takes on a raised shield; Parried
// tells whether the last was a parry.
func (p *Player) GuardSeq() uint8 { return p.guard.seq }
func (p *Player) Parried() bool   { return p.guard.parry }

// stepGuard raises or lowers p's shield as the input says.
func (w *World) stepGuard(p *Player) {
	want := p.input.Guard && p.HasShield() && p.mount == 0 && p.boat == 0 && p.line.state == CastNone && !w.Swimming(p)
	if want && !p.guard.up {
		p.guard.since = w.tick
		w.faceFoe(p)
	}
	p.guard.up = want
	if !want {
		return
	}
	if ax, ay, ok := p.Aim(); ok {
		if dx, dy := ax-p.X, ay-p.Y; dx*dx+dy*dy > 0.04 {
			p.Dir = dirToward(dx, dy)
		}
	}
}

// faceFoe turns p to the nearest beast within guardFoeNear that is after
// them, if there is one.
func (w *World) faceFoe(p *Player) {
	var best *Creature
	bestD := guardFoeNear
	w.EachCreatureIn(p.X-guardFoeNear, p.Y-guardFoeNear, p.X+guardFoeNear, p.Y+guardFoeNear, func(cv Creature) {
		c := w.creatureByID(cv.ID)
		if c == nil || c.gone || c.Tamed() || !hunting(c, p) {
			return
		}
		if d := math.Hypot(c.X-p.X, c.Y-p.Y); d < bestD {
			best, bestD = c, d
		}
	})
	if best != nil {
		p.Dir = dirToward(best.X-p.X, best.Y-p.Y)
	}
}

// hunting reports whether c is going for p.
func hunting(c *Creature, p *Player) bool {
	if c.foe == p.ID && c.rage > 0 {
		return true
	}
	switch c.State {
	case CreatureChase, CreatureStrike, CreatureRoar, CreatureClaw, CreatureSpit, CreatureSpecial:
		return true
	}
	return c.blow.ticks > 0 && c.blow.foe == p.ID
}

// covers reports whether p's raised shield faces a blow from (x, y).
func (p *Player) covers(x, y float64) bool {
	dx, dy := x-p.X, y-p.Y
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		return true
	}
	fx, fy := p.Dir.Delta()
	return (dx*float64(fx)+dy*float64(fy))/l >= GuardCover
}

// shieldTakes is what is left of a blow of dmg on p once a raised shield
// has taken its share, and whether it did: struck calls it.
func (p *Player) shieldTakes(dmg float64) (float64, bool) {
	g := &p.guard
	if !g.up || p.src == nil || p.src.c == nil {
		return dmg, false
	}
	c, now := p.src.c, p.src.tick
	if !p.covers(c.X, c.Y) {
		return dmg, false
	}
	parry := now-g.since <= ParryTicks
	light := !c.Kind.def().tough && !c.Kind.Boss()
	g.seq++
	g.at = now
	g.parry = parry
	push := BlockPush
	if parry && light {
		dmg = 0
		push *= 0.4
		c.reel = ParryStagger // it reels from its next step on; see knock.go
		c.riposte = 0
	} else {
		dmg *= 1 - GuardShare
	}
	// Pushed back along the blow, a little.
	dx, dy := p.X-c.X, p.Y-c.Y
	if l := math.Hypot(dx, dy); l > 1e-6 {
		p.knockX += dx / l * push
		p.knockY += dy / l * push
	}
	return dmg, true
}

// ParryOutcome describes a blow at a player with a shield up, for tests and
// the balance table: what share gets through and whether a beast of kind k
// is staggered, raised ticks before the blow, from in front or not.
func ParryOutcome(k CreatureKind, ticksUp int, front bool) (share float64, staggered bool) {
	if !front {
		return 1, false
	}
	light := !k.def().tough && !k.Boss()
	if ticksUp <= ParryTicks && light {
		return 0, true
	}
	return 1 - GuardShare, false
}
