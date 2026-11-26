package sim

import "math"

// Ankylosaurus, the armoured one ("Khủng long bọc giáp"): a low, wide grazer
// of the woods and the meadows, in ones, twos and threes, now and then by a
// river or out in the marsh. It minds nobody who keeps their distance.
//
//   - Come too close and it turns its back on you, its tail club raised, and
//     grunts: that is its guard. Come closer still, or hit it, and it crouches
//     with the club drawn back, then swings it sideways (telegraph.go): a heavy
//     blow that throws a player back. A hunter after it is clubbed the same
//     way, and reels however big it is; even a tyrant.
//   - Its back, flanks and tail are bony plates: a blow there does half. A blow
//     from before its head lands on the head, or the soft throat and belly
//     under it, and does it all. It is slow to turn round, above all just after
//     a swing: that is the moment to get round to its head.
//   - It is tamed with berries like the other grazers (it lets anyone holding
//     them come close) and ridden as a slow mount that fights hard for its owner.
//   - It gives meat, hides, bones and its bony plates (osteoderms), which go
//     into bone armour (gear.go).

// Tuning. Distances in tiles, times in ticks.
const (
	AnkyloNotice       = 3.4  // a player this close has it turn its back and raise its club
	AnkyloClubRange    = 2.6  // and this close, it swings
	AnkyloClubDamage   = 30.0 // the heaviest blow of any grazer
	AnkyloClubKnock    = 1.6  // tiles the club throws a player
	AnkyloClubCD       = 3 * TicksPerSecond
	AnkyloWindup       = 12                     // crouched, the club drawn back: a clear warning
	AnkyloTurnTicks    = 3 * TicksPerSecond / 4 // turning round takes it this long
	AnkyloGuardTicks   = 2 * TicksPerSecond     // its guard stays up this long after the threat has gone
	AnkyloBackUp       = 7.0                    // hit, it backs towards whoever did it while they are this close
	AnkyloArmour       = 0.5                    // the share of a blow on its plates that does not get through
	AnkyloDefendDamage = 14                     // its club on a hunter, in hits
	AnkyloStaggerTicks = 6 * TicksPerSecond / 5 // a clubbed hunter reels this long, however big
	AnkyloBeastKnock   = 1.0                    // and is thrown back this far
)

var ankyloDef = creatureDef{
	name: "Khủng long bọc giáp", hp: 36,
	walk: 0.55 / TicksPerSecond, run: 1.2 / TicksPerSecond, // slower than the plate-back
	radius: 0.8, reach: 2.8,
	yield: ItemMeat, yieldN: 7, drops: []Ingredient{{ItemHide, 2}, {ItemBone, 2}, {ItemOsteoderm, 3}},
	leash: 10, idle: [2]int{60, 220}, roam: [2]int{30, 80},
	spawnOn: []Tile{TileForest, TileGrass, TileMarsh}, tilesPer: 3200,
	fights: true, tough: true, openOnly: true, herd: 3, herdMin: 1, away: 16,
	since: StyleBeaches,
}

func init() {
	creatureDefs[CreatureAnkylo] = ankyloDef
	tameDefs[CreatureAnkylo] = tameDef{food: ItemBerry, meals: 6, ride: 4.2 / TicksPerSecond, bite: 6}
	diets[CreatureAnkylo] = dietDef{
		grazes: true, hunger: [2]int{40 * TicksPerSecond, 90 * TicksPerSecond},
		thirst: [2]int{100 * TicksPerSecond, 200 * TicksPerSecond}, eatTicks: 7 * TicksPerSecond,
		defends: true,
	}
	windupTicks[CreatureAnkylo] = AnkyloWindup
	itemDefs[ItemOsteoderm] = ItemDef{Name: "Tấm giáp xương"}
	for _, n := range []string{"ankylo", "ankylosaurus", "ankylosaur", "giaplong"} {
		devBeastAliases[n] = CreatureAnkylo
	}
	devItemAliases["osteoderm"] = ItemOsteoderm
}

// herdSize is how many of kind k appear together this time: herd, or for a
// kind that comes in groups of different sizes, herdMin to herd at random.
func (w *World) herdSize(k CreatureKind) int {
	d := k.def()
	if d.herdMin <= 0 || d.herdMin >= d.herd {
		return d.herd
	}
	return d.herdMin + w.random().IntN(d.herd-d.herdMin+1)
}

// headOn reports whether (x, y) lies before c's head: where a blow lands on
// the head or the soft throat and belly under it, and where its club does
// not reach.
func headOn(c *Creature, x, y float64) bool {
	dx, dy := x-c.X, y-c.Y
	if c.FaceLeft {
		dx = -dx
	}
	return dx > 0.3 && dx >= 0.8*math.Abs(dy)
}

// armoured is what is left of a blow of dmg by p at c once its armour has
// taken its share: an Ankylosaurus's plates stop half of a blow on its back,
// flanks or tail, and none of one at its head.
func armoured(p *Player, c *Creature, dmg int) int {
	if c.Kind != CreatureAnkylo || headOn(c, p.X, p.Y) {
		return dmg
	}
	p.notify(NoticeArmoured, uint8(c.Kind))
	return max(1, int(float64(dmg)*(1-AnkyloArmour)+0.5))
}

// turnBack turns c's back on x, its club towards it, if it is not so
// already and it can turn yet. It reports whether its back is to x.
func (c *Creature) turnBack(x float64) bool {
	if math.Abs(x-c.X) < 0.25 {
		return true // beside it: the club sweeps that way, whichever way it faces
	}
	away := x > c.X // facing left, its tail points right
	if c.FaceLeft != away && c.cool == 0 {
		c.FaceLeft, c.cool = away, AnkyloTurnTicks
	}
	return c.FaceLeft == away
}

// ankyloThreat is the player c guards against, and how far off: whoever hit
// it, while it is still angry and they are about; else the nearest player
// awake who comes too close, unless they hold out its food.
func (w *World) ankyloThreat(c *Creature) (*Player, float64) {
	if p, d := w.foeOf(c); p != nil {
		if d <= GrazerGiveUp {
			return p, d
		}
		c.rage = 0
	}
	p, d := w.nearestPlayer(c.X, c.Y)
	if p == nil || d > AnkyloNotice || p.Asleep || p.Held() == c.Kind.TameFood() {
		return nil, math.Inf(1)
	}
	return p, d
}

// stepAnkylo is a wild Ankylosaurus: it grazes and wanders until someone
// comes too close or hits it, then keeps its back to them and swings its
// club at anyone in reach.
func (w *World) stepAnkylo(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	if c.State == CreatureStrike { // the blow itself lands in telegraph.go
		if c.timer--; c.timer <= 0 {
			// Swung round, it is slow to turn again.
			c.State, c.timer, c.cool = CreatureRoar, AnkyloGuardTicks, AnkyloTurnTicks
		}
		return
	}
	p, dist := w.ankyloThreat(c)
	if p == nil {
		if c.State == CreatureRoar && c.timer > 0 {
			c.timer-- // its guard stays up a moment
			return
		}
		if c.State == CreatureRoar || c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		if !w.forage(c) { // see needs.go
			w.wander(c)
		}
		return
	}
	back := c.turnBack(p.X)
	switch {
	case dist <= AnkyloClubRange && back && !headOn(c, p.X, p.Y) && c.biteCD == 0:
		strike(c, p, AnkyloClubDamage, AnkyloClubCD) // crouched, the club drawn back: see telegraph.go
		c.blow.knock = AnkyloClubKnock
	case c.rage > 0 && back && dist > AnkyloClubRange*0.9 && dist <= AnkyloBackUp:
		// Hit from further off: it backs towards them, club first.
		c.State = CreatureChase
		face := c.FaceLeft
		w.moveCreature(c, (p.X-c.X)/dist*d.walk, (p.Y-c.Y)/dist*d.walk)
		c.FaceLeft = face
	default:
		c.State, c.timer = CreatureRoar, AnkyloGuardTicks
	}
}

// ankyloDefend is an Ankylosaurus hunted by h (see needs.go): it turns its
// club on it and swings when it comes close, after the same crouch as at a
// player. It reports whether that was its step.
func (w *World) ankyloDefend(c, h *Creature) bool {
	dist := math.Hypot(h.X-c.X, h.Y-c.Y)
	if dist > PreyFleeDist && c.State != CreatureStrike {
		return false
	}
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	reach := c.Kind.def().radius + h.Kind.def().radius + 1.4 // further than any hunter's bite
	if c.State == CreatureStrike {
		if c.timer--; c.timer == AttackTicks && !h.gone && dist <= reach+BlowSlack && !headOn(c, h.X, h.Y) {
			w.clubBeast(c, h)
		}
		if c.timer <= 0 {
			c.State, c.timer, c.cool = CreatureRoar, AnkyloGuardTicks, AnkyloTurnTicks
		}
		return true
	}
	back := c.turnBack(h.X)
	if dist <= reach && back && !headOn(c, h.X, h.Y) && c.biteCD == 0 {
		c.State, c.timer, c.biteCD = CreatureStrike, AttackTicks+WindupTicks(c.Kind), AnkyloClubCD
		return true
	}
	c.State, c.timer = CreatureRoar, AnkyloGuardTicks
	return true
}

// clubBeast lands c's club on the beast o: it reels back, whatever its
// size, and loses any blow it was drawing back.
func (w *World) clubBeast(c, o *Creature) {
	if w.maul(o, AnkyloDefendDamage) {
		return // the club killed it
	}
	if o.State != CreatureHurt {
		o.after, o.afterT = o.State, o.timer
	}
	o.State, o.timer = CreatureHurt, AnkyloStaggerTicks
	o.blow = pendingBlow{}
	dx, dy := o.X-c.X, o.Y-c.Y
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		dx, dy, l = 1, 0, 1
	}
	face := o.FaceLeft // thrown back, not turned round
	for s := 0.0; s < AnkyloBeastKnock; s += 0.1 {
		if !w.moveCreature(o, dx/l*0.1, dy/l*0.1) {
			break
		}
	}
	o.FaceLeft = face
}
