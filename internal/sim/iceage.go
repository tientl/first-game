package sim

import "math"

// The ice age: the frozen north of worlds made since StyleIce.
//
// Snow covers the coldest lowland. It is freezing to stand on, day and
// night, unless a fire, a torch or lava is close, the player carries a fur
// coat, or rides a mammoth. Deep snow slows walking.
//
//   - Mammoths graze the snow in herds. Like three-horns they are calm until
//     one of them is hurt; then they charge and gore with their tusks. They
//     give fur and ivory, can be tamed with dry grass, and ridden.
//   - Sabre-tooths stalk the snow alone: one creeps up on a player it has
//     seen, then springs from close by, bites hard and draws back to spring
//     again. It runs slower than a player. Beaten down, it cowers and can be
//     tamed with meat, like a raptor; it fights for its owner but carries
//     nobody.

// Ice-age tuning.
const (
	snowShare = 0.13 // of the lowland: the coldest part freezes
	snowSpeed = 0.85 // walking in deep snow

	MammothTuskRange  = 2.6
	MammothTuskDamage = 26.0
	MammothTuskCD     = 3 * TicksPerSecond

	SabreSight       = 9.0
	SabreGiveUp      = 14.0
	SabreSpringDist  = 3.4 // it springs from this close
	SabreSpringSpeed = 9.0 / TicksPerSecond
	SabreSpringTicks = 6
	SabreBiteRange   = 1.3
	SabreBiteDamage  = 20.0
	SabreBiteCD      = 5 * TicksPerSecond / 2
	SabreBackOff     = 3 * TicksPerSecond / 2
	SabreCowerBelow  = 3 // hit points at which it gives up
	SabreCowerTicks  = 8 * TicksPerSecond

	IvorySpearDamage = 40
	IvorySpearReach  = 0.8 // extra reach on creatures, held
)

var (
	mammothDef = creatureDef{
		name: "Voi ma mút", hp: 60,
		walk: 0.8 / TicksPerSecond, run: 3.4 / TicksPerSecond,
		radius: 0.95, reach: 3.0,
		yield: ItemMeat, yieldN: 10, drops: []Ingredient{{ItemFur, 4}, {ItemIvory, 2}, {ItemHide, 1}},
		leash: 16, idle: [2]int{50, 200}, roam: [2]int{30, 100},
		spawnOn: []Tile{TileSnow}, tilesPer: 1300,
		fights: true, tough: true, openOnly: true, herd: 3, away: 20,
	}
	sabreDef = creatureDef{
		name: "Hổ răng kiếm", hp: 22,
		walk: 1.2 / TicksPerSecond, run: 3.6 / TicksPerSecond, // slower than a player
		radius: 0.42, reach: 1.8,
		yield: ItemMeat, yieldN: 4, drops: []Ingredient{{ItemFur, 2}, {ItemBone, 1}},
		leash: 14, idle: [2]int{40, 160}, roam: [2]int{30, 80},
		spawnOn: []Tile{TileSnow}, tilesPer: 2600,
		fights: true, openOnly: true, away: 30,
	}
)

func init() {
	tameDefs[CreatureMammoth] = tameDef{food: ItemGrass, meals: 6, ride: 5.0 / TicksPerSecond, bite: 5}
	tameDefs[CreatureSabre] = tameDef{food: ItemMeat, meals: 4, bite: 4, cower: true}
}

// freezing reports whether p stands on snow with nothing to keep the cold
// out: no fur coat, not on a mammoth. A fire nearby is checked apart.
func (w *World) freezing(p *Player) bool {
	if w.TileAt(int(math.Floor(p.X)), int(math.Floor(p.Y))) != TileSnow || p.Inv.Count(ItemFurCoat) > 0 {
		return false
	}
	if c := w.creatureByID(p.mount); c != nil && c.Kind == CreatureMammoth {
		return false // the warmth of a mammoth's wool
	}
	return true
}

func (w *World) stepSabre(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	if w.huntOn(c, SabreSight) {
		return // after prey: see needs.go
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	switch c.State {
	case CreatureStrike: // the spring
		c.timer--
		w.moveCreature(c, c.hx*SabreSpringSpeed, c.hy*SabreSpringSpeed)
		if p != nil && dist <= SabreBiteRange && c.biteCD == 0 {
			struck(p, SabreBiteDamage)
			p.notify(NoticeBitten, uint8(c.Kind))
			c.biteCD = SabreBiteCD
			c.State, c.timer = CreatureFlee, SabreBackOff // draws back to spring again
			c.hx, c.hy = 0, 0
		} else if c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	case CreatureFlee:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
			return
		}
		if p != nil {
			c.steerAway(p.X, p.Y, 0.5)
		}
		if !w.moveCreature(c, c.hx*d.run, c.hy*d.run) && p != nil {
			c.dodge(w, p.X, p.Y)
		}
		return
	case CreatureCower:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureFlee, 4*TicksPerSecond
			c.hx, c.hy = 0, 0
		}
		return
	}
	if p != nil && c.HP <= SabreCowerBelow*HealthScale && dist < SabreGiveUp {
		c.State, c.timer, c.rage = CreatureCower, SabreCowerTicks, 0 // beaten
		return
	}
	if p == nil || c.cool > 0 || dist > SabreGiveUp || (c.State != CreatureChase && dist > SabreSight) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		if !w.forage(c) {
			w.wander(c)
		}
		return
	}
	// Stalking: it creeps closer, then springs.
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= SabreSpringDist && c.biteCD == 0 {
		c.State, c.timer = CreatureStrike, SabreSpringTicks
		c.hx, c.hy = (p.X-c.X)/dist, (p.Y-c.Y)/dist
		return
	}
	speed := d.walk * 1.6 // creeping
	if dist < 6 || c.rage > 0 {
		speed = d.run
	}
	w.runAt(c, p.X, p.Y, speed, SabreSpringDist*0.8)
}
