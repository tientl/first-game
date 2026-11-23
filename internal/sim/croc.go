package sim

import "math"

// Crocodiles live in the marsh, where the ponds meet the mud. Most of the
// time one lurks in the shallows with only its eyes above the water, out of
// reach. Anyone who comes close to that water gets an ambush: it bursts out
// faster than a player can run and bites hard, then slides back in and waits
// a while before it can strike again. On land it is slow, so walking away
// works once it is out; and now and then it climbs out to bask on the mud,
// which is when it can be fought. It takes several hits.

// Crocodile tuning.
const (
	CrocAmbushDist     = 3.0 // a lurking crocodile strikes at a player this close
	CrocLungeSpeed     = 6.0 / TicksPerSecond
	CrocLungeTicks     = 18 // how long the burst out of the water lasts
	CrocBiteRange      = 1.2
	CrocBiteDamage     = 18.0
	CrocBiteCooldown   = 2 * TicksPerSecond
	CrocStrikeTicks    = 8
	CrocAmbushCooldown = 4 * TicksPerSecond // back in the water, it waits this long before striking again
	CrocSnapDist       = 2.5                // a basking crocodile goes for a player this close
	CrocGiveUpDist     = 5.0                // and gives up the chase past this
	CrocChaseTicks     = 3 * TicksPerSecond // or after this long: it will not stray far from the water
	CrocBaskChance     = 0.003              // per tick, a lurking crocodile climbs out to bask
)

// waterside reports whether tile (tx, ty) can be walked on and touches water.
func (w *World) waterside(tx, ty int) bool {
	if !w.TileAt(tx, ty).Walkable() {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && w.inBounds(tx+dx, ty+dy) && w.TileAt(tx+dx, ty+dy) == TileWater {
				return true
			}
		}
	}
	return false
}

func (w *World) stepCroc(c *Creature) {
	d := c.Kind.def()
	ambush, bite, reach := crocTune(c.Kind) // see hellcreek.go: its cousins of the ponds
	if c.biteCD > 0 {
		c.biteCD--
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	switch c.State {
	case CreatureStrike:
		if c.timer--; c.timer <= 0 {
			w.crocRetreat(c) // bitten: back to the water
		}
	case CreatureLurk:
		if c.timer > 0 {
			c.timer--
		}
		switch {
		case p != nil && dist < ambush && c.timer == 0:
			c.State, c.timer = CreatureChase, CrocChaseTicks+CrocLungeTicks // out of the water in a burst
		case c.timer == 0 && (p == nil || dist >= ambush) && w.drinkerNear(c, ambush) != nil:
			// Something come down to drink: a snatch from the water. See needs.go.
			q := w.drinkerNear(c, ambush) // the same one: the drinkers do not change mid-tick
			c.faceToward(q.X)
			c.State, c.timer, c.biteCD = CreatureStrike, CrocStrikeTicks, CrocBiteCooldown
			w.maul(q, int(bite)/crocPreyBiteBits)
			q.hunter, q.huntedT = c.ID, huntedTicks // it bolts from the water
		case w.random().Float64() < CrocBaskChance && (p == nil || dist > CrocGiveUpDist):
			c.State, c.timer = CreatureIdle, 0 // climbs out to bask
		}
	case CreatureChase:
		if c.timer--; p == nil || dist > CrocGiveUpDist || c.timer <= 0 {
			w.crocRetreat(c)
			return
		}
		speed := d.run
		if c.timer > CrocChaseTicks {
			speed = CrocLungeSpeed // still bursting out of the water
		}
		if dx := p.X - c.X; dx != 0 {
			c.FaceLeft = dx < 0
		}
		if dist <= reach {
			if c.biteCD == 0 {
				c.State, c.timer, c.biteCD = CreatureStrike, CrocStrikeTicks, CrocBiteCooldown
				struck(p, bite)
				p.notify(NoticeBitten, uint8(c.Kind))
			}
			return
		}
		step := min(speed, dist-reach*0.8)
		if !w.moveCreature(c, (p.X-c.X)/dist*step, (p.Y-c.Y)/dist*step) {
			w.crocRetreat(c) // something in the way: give up
		}
	case CreatureFlee: // making for the water
		dx, dy := c.goalX-c.X, c.goalY-c.Y
		l := math.Hypot(dx, dy)
		switch {
		case w.inShallows(c):
			c.State, c.timer = CreatureLurk, CrocAmbushCooldown
		case l < 1e-9 || !w.moveCreature(c, dx/l*min(d.run, l), dy/l*min(d.run, l)):
			c.State, c.timer = CreatureIdle, 0 // cut off from the water: it stays on land, in sight
		}
	default: // basking or wandering about
		_, _, water := w.nearestShallows(c.X, c.Y)
		switch {
		case c.HP < d.hp && !w.inShallows(c) && water:
			w.crocRetreat(c) // wounded: back to the water
		case p != nil && dist < CrocSnapDist && c.biteCD == 0:
			c.State, c.timer = CreatureChase, CrocChaseTicks // on land: a slow lunge
		case w.inShallows(c) && c.State == CreatureIdle:
			c.State, c.timer = CreatureLurk, 0 // in the water again: sink to the eyes
		default:
			w.wander(c)
		}
	}
}

// crocRetreat sends c back to the nearest water, or straight under if it is
// in the water already.
func (w *World) crocRetreat(c *Creature) {
	c.biteCD = max(c.biteCD, CrocBiteCooldown) // no snapping again straight away
	if w.inShallows(c) {
		c.State, c.timer = CreatureLurk, CrocAmbushCooldown
		return
	}
	if gx, gy, ok := w.nearestShallows(c.X, c.Y); ok {
		c.State, c.goalX, c.goalY = CreatureFlee, gx, gy
		return
	}
	c.State, c.timer = CreatureIdle, 0 // no water near: it stays where it is
}
