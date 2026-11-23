package sim

import "math"

// Monsters of the open sea. They live far out, where the water is deep, and
// go for anyone on the water: in a boat or swimming. Close to the coast a
// boat is safe from them (the sea reptiles of dino.go still lurk there).
//
//   - The Mosasaurus, a sea lizard as long as a ship, swims just under the
//     surface (only a ridge of its back shows) faster than a raft, slower
//     than a sail before the wind. Close enough, it rears up and bites: a
//     boat takes a great bite out of its side, a swimmer rarely survives
//     two. It stays up a few moments, when it can be fought, then dives.
//   - The giant squid waits deeper still. A boat that comes over it is
//     seized in its arms and held fast, the hull cracking, until the squid
//     tires or is hurt enough to let go.

// Sea monster tuning.
const (
	MosaSight       = 14.0
	MosaGiveUp      = 24.0
	MosaBiteRange   = 2.4
	MosaBoatBite    = 30 // boat hit points
	MosaRiderBite   = 8.0
	MosaSwimBite    = 34.0
	MosaBiteCD      = 3 * TicksPerSecond
	MosaSurfaceTick = 4 * TicksPerSecond // up and fightable after a bite
	MosaDiveTicks   = 5 * TicksPerSecond // then under, before it can bite again

	KrakenNotice     = 3.4
	KrakenGrabTicks  = 7 * TicksPerSecond
	KrakenBoatSqueez = 8 // boat hit points a second
	KrakenSqueeze    = 4.0
	KrakenLetGo      = 12 // hit points lost that make it let go
	KrakenRestTicks  = 20 * TicksPerSecond
	krakenStrikeTick = TicksPerSecond
)

var (
	mosaDef = creatureDef{
		name: "Thương long", hp: 45,
		walk: 1.6 / TicksPerSecond, run: 5.4 / TicksPerSecond,
		radius: 1.0, reach: 3.2,
		yield: ItemMeat, yieldN: 12, drops: []Ingredient{{ItemBone, 5}, {ItemHide, 2}},
		leash: 40, idle: [2]int{40, 120}, roam: [2]int{40, 120},
		spawnOn: []Tile{TileWater}, tilesPer: 2200,
		fights: true, tough: true, deep: true, openOnly: true, away: 30,
		offshore: [2]int{8, 250},
	}
	krakenDef = creatureDef{
		name: "Mực khổng lồ", hp: 36,
		walk: 0.8 / TicksPerSecond, run: 3.6 / TicksPerSecond,
		radius: 1.1, reach: 3.4,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemSilk, 4}},
		leash: 14, idle: [2]int{60, 200}, roam: [2]int{40, 100},
		spawnOn: []Tile{TileWater}, tilesPer: 4500,
		fights: true, tough: true, deep: true, openOnly: true, away: 40,
		offshore: [2]int{12, 250},
	}
)

func init() {
	creatureDefs[CreatureMosa] = mosaDef
	creatureDefs[CreatureKraken] = krakenDef
}

// seaPrey returns the nearest player on the water within sight of c: in a
// boat or swimming.
func (w *World) seaPrey(c *Creature, sight float64) (*Player, float64) {
	var best *Player
	bestD := sight
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || (p.boat == 0 && !w.inWater(p.X, p.Y)) {
			continue
		}
		if d := math.Hypot(p.X-c.X, p.Y-c.Y); d <= bestD {
			best, bestD = p, d
		}
	}
	return best, bestD
}

// seaBite lands a sea monster's attack on p: on the boat p is in, with a
// knock for p, or on p if swimming.
func (w *World) seaBite(c *Creature, p *Player, boatDmg int, riderDmg, swimDmg float64) {
	if b := w.boatByID(p.boat); b != nil {
		w.damageBoat(b, boatDmg)
		struck(p, riderDmg)
	} else {
		struck(p, swimDmg)
	}
	p.notify(NoticeBitten, uint8(c.Kind))
}

func (w *World) stepMosa(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	p, dist := w.seaPrey(c, MosaGiveUp)
	switch c.State {
	case CreatureStrike: // the bite, reared out of the water
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureChase, MosaSurfaceTick
		}
		return
	case CreatureChase: // up at the surface: it can be fought
		if c.timer--; c.timer <= 0 || p == nil {
			c.State, c.timer, c.cool = CreatureLurk, 0, MosaDiveTicks
			return
		}
		c.faceToward(p.X)
		if dist <= MosaBiteRange && c.biteCD == 0 {
			c.State, c.timer, c.biteCD = CreatureStrike, AttackTicks, MosaBiteCD
			w.seaBite(c, p, MosaBoatBite, MosaRiderBite, MosaSwimBite)
			return
		}
		w.runAt(c, p.X, p.Y, d.walk*1.5, MosaBiteRange*0.7)
		return
	}
	c.State = CreatureLurk // under the surface: out of reach
	if p == nil || c.cool > 0 || (dist > MosaSight && c.foe == 0) {
		c.foe = 0
		w.swim(c)
		return
	}
	c.foe = p.ID // remembered while the chase lasts
	if dist <= MosaBiteRange && c.biteCD == 0 {
		c.State, c.timer, c.biteCD = CreatureStrike, AttackTicks, MosaBiteCD
		c.faceToward(p.X)
		w.seaBite(c, p, MosaBoatBite, MosaRiderBite, MosaSwimBite)
		return
	}
	if !w.runAt(c, p.X, p.Y, d.run, MosaBiteRange*0.8) {
		c.dodge(w, p.X, p.Y) // the coast is in the way
		w.moveCreature(c, c.hx*d.run, c.hy*d.run)
	}
}

func (w *World) stepKraken(c *Creature) {
	if c.cool > 0 {
		c.cool--
	}
	if c.State == CreatureGrab {
		p := w.players[c.foe]
		lost := c.HP <= c.grabHP-KrakenLetGo
		if c.timer--; c.timer <= 0 || p == nil || p.Dead || lost {
			c.State, c.timer, c.cool = CreatureLurk, 0, KrakenRestTicks
			if b := w.boatByID(p.boatOrZero()); b != nil {
				b.Held = 0
			}
			return
		}
		if b := w.boatByID(p.boat); b != nil {
			b.Held = c.timer // held fast
			if c.timer%TicksPerSecond == 0 {
				w.damageBoat(b, KrakenBoatSqueez)
				struck(p, KrakenSqueeze)
				p.notify(NoticeBitten, uint8(c.Kind))
			}
		} else if c.timer%TicksPerSecond == 0 {
			struck(p, KrakenSqueeze*2) // a swimmer, dragged under
			p.notify(NoticeBitten, uint8(c.Kind))
		}
		c.faceToward(p.X)
		return
	}
	c.State = CreatureLurk
	if c.cool > 0 {
		w.swim(c)
		return
	}
	p, dist := w.seaPrey(c, KrakenNotice)
	if p == nil {
		w.swim(c)
		return
	}
	// Seized.
	c.State, c.timer, c.foe = CreatureGrab, KrakenGrabTicks, p.ID
	c.grabHP = c.HP
	_ = dist
}

// boatOrZero is p's boat, or 0 when p is nil.
func (p *Player) boatOrZero() BoatID {
	if p == nil {
		return 0
	}
	return p.boat
}
