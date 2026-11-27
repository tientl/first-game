package sim

import "math"

// More beasts of the age of the tyrant, the ones that shared its floodplains:
//
//   - Ornithomimus, an ostrich of a dinosaur, grazes the open plains in
//     flocks and outruns anything; it only stops, panting, after a sprint,
//     and that is the moment to catch it, or to feed it. Tamed, it is the
//     fastest mount there is.
//   - Thescelosaurus, a gentle grazer of middling size, runs from trouble.
//   - Anzu, a giant crested bird of a dinosaur, holds its ground: it raises
//     its crest at anyone coming close, then kicks them if they come closer.
//   - Torosaurus is the three-horn's bigger cousin with a great open frill,
//     and charges like it.
//   - Acheroraptor is a raptor smaller than Dakotaraptor (the raptor of
//     dino.go), hunting in bigger packs.
//   - Quetzalcoatlus, a flying reptile as tall as a giraffe when it stands,
//     stalks the open ground on all fours and stabs with its great beak;
//     badly hurt, it takes to the air and flies off.
//   - Brachychampsa and Champsosaurus lurk in the marsh ponds like the
//     crocodile: one blunt-snouted and hard-biting, one long and narrow.

// Tuning.
const (
	OrnithoFleeDist    = 6.0
	OrnithoSprintTicks = 3 * TicksPerSecond
	OrnithoRestTicks   = 5 * TicksPerSecond / 2 // panting: the moment to catch it
	OrnithoCalmTicks   = 4 * TicksPerSecond     // after a meal it lets you come close

	ToroGoreRange  = 2.7
	ToroGoreDamage = 26.0
	ToroGoreCD     = 3 * TicksPerSecond

	AnzuNotice       = 3.5 // it raises its crest at anyone this close
	AnzuDisplayTicks = 20
	AnzuKickRange    = 1.8
	AnzuKickDamage   = 10.0
	AnzuKickCD       = 9 * TicksPerSecond / 5
	AnzuLeash        = 5.0 // it will not chase far from where it stood

	QuetzalNotice      = 6.0
	QuetzalGiveUp      = 11.0
	QuetzalStabRange   = 2.6
	QuetzalStabDamage  = 18.0
	QuetzalStabCD      = 5 * TicksPerSecond / 2
	QuetzalFlyBelow    = 0.4 // share of its health at which it flies off
	QuetzalFlySpeed    = 5.0 / TicksPerSecond
	QuetzalFlyTicks    = 8 * TicksPerSecond
	QuetzalGroundTicks = 30 * TicksPerSecond // after landing it will not fly off again this soon
)

var (
	ornithoDef = creatureDef{
		name: "Ornithomimus", hp: 4,
		walk: 1.3 / TicksPerSecond, run: 6.5 / TicksPerSecond, // far faster than a player
		radius: 0.32, reach: 1.8,
		yield: ItemMeat, yieldN: 3, drops: []Ingredient{{ItemHide, 1}, {ItemBone, 1}},
		leash: 14, idle: [2]int{30, 120}, roam: [2]int{20, 70},
		spawnOn: []Tile{TileGrass, TileDesert}, tilesPer: 1800,
		openOnly: true, herd: 5, away: 12,
	}
	thescDef = creatureDef{
		name: "Thescelosaurus", hp: 8,
		walk: 0.9 / TicksPerSecond, run: 3.0 / TicksPerSecond,
		radius: 0.45, reach: 2.0,
		yield: ItemMeat, yieldN: 4, drops: []Ingredient{{ItemHide, 1}, {ItemBone, 1}},
		leash: 10, idle: [2]int{40, 160}, roam: [2]int{20, 70},
		spawnOn: []Tile{TileForest, TileGrass}, tilesPer: 2000,
		openOnly: true, herd: 3, away: 10,
	}
	anzuDef = creatureDef{
		name: "Anzu", hp: 12,
		walk: 1.0 / TicksPerSecond, run: 3.0 / TicksPerSecond,
		radius: 0.4, reach: 1.9,
		yield: ItemMeat, yieldN: 3, drops: []Ingredient{{ItemBone, 1}},
		leash: 6, idle: [2]int{40, 160}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileForest, TileMarsh}, tilesPer: 3000,
		fights: true, openOnly: true, herd: 2, away: 18,
	}
	toroDef = creatureDef{
		name: "Torosaurus", hp: 38,
		walk: 0.9 / TicksPerSecond, run: 3.4 / TicksPerSecond,
		radius: 0.85, reach: 2.9,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemHide, 3}, {ItemBone, 2}},
		leash: 14, idle: [2]int{40, 160}, roam: [2]int{30, 90},
		spawnOn: []Tile{TileGrass, TileForest}, tilesPer: 3200,
		fights: true, tough: true, openOnly: true, herd: 3, away: 20,
	}
	acheroDef = creatureDef{
		name: "Acheroraptor", hp: 8,
		walk: 1.5 / TicksPerSecond, run: 3.6 / TicksPerSecond, // still slower than a player
		radius: 0.3, reach: 1.6,
		yield: ItemMeat, yieldN: 1, drops: []Ingredient{{ItemHide, 1}},
		leash: 12, idle: [2]int{30, 120}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileForest}, tilesPer: 2800,
		fights: true, openOnly: true, herd: 5, away: 35,
	}
	quetzalDef = creatureDef{
		name: "Quetzalcoatlus", hp: 25,
		walk: 1.1 / TicksPerSecond, run: 2.2 / TicksPerSecond,
		radius: 0.5, reach: 2.6,
		yield: ItemMeat, yieldN: 6, drops: []Ingredient{{ItemHide, 2}, {ItemBone, 3}},
		leash: 16, idle: [2]int{60, 200}, roam: [2]int{30, 90},
		spawnOn: []Tile{TileGrass, TileSand, TileRocky, TileDesert}, tilesPer: 8000,
		fights: true, tough: true, openOnly: true, away: 30,
	}
	brachyDef = creatureDef{
		name: "Brachychampsa", hp: 12,
		walk: 0.7 / TicksPerSecond, run: 2.4 / TicksPerSecond,
		radius: 0.38, reach: 1.7,
		yield: ItemMeat, yieldN: 3, drops: []Ingredient{{ItemHide, 1}},
		amphibious: true, waterside: true,
		leash: 5, idle: [2]int{60, 200}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileMarsh}, tilesPer: 300,
		fights: true, openOnly: true,
	}
	champsoDef = creatureDef{
		name: "Champsosaurus", hp: 9,
		walk: 0.8 / TicksPerSecond, run: 2.6 / TicksPerSecond,
		radius: 0.3, reach: 1.6,
		yield: ItemMeat, yieldN: 2, drops: []Ingredient{{ItemHide, 1}},
		amphibious: true, waterside: true,
		leash: 5, idle: [2]int{60, 200}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileMarsh}, tilesPer: 320,
		fights: true, openOnly: true,
	}
)

func init() {
	tameDefs[CreatureOrnitho] = tameDef{food: ItemBerry, meals: 4, ride: 7.5 / TicksPerSecond}
	tameDefs[CreatureThesc] = tameDef{food: ItemBerry, meals: 4}
	tameDefs[CreatureAnzu] = tameDef{food: ItemBerry, meals: 5, bite: 3}
	tameDefs[CreatureToro] = tameDef{food: ItemBerry, meals: 6, ride: 5.0 / TicksPerSecond, bite: 5}
	tameDefs[CreatureAchero] = tameDef{food: ItemMeat, meals: 2, bite: 2, cower: true}
}

// raptorTune is how a raptor of kind k hunts: its bite, how close it
// springs from, how far it sees by day and by night, and the hit points at
// which it gives up.
func raptorTune(k CreatureKind) (bite, pounce, sightDay, sightNight float64, cowerAt int) {
	if k == CreatureAchero {
		return 6, 2.4, 8, 11, 1
	}
	return RaptorBiteDamage, RaptorPounceDist, RaptorSightDay, RaptorSightNight, RaptorFleeBelow
}

// crocTune is how a crocodile of kind k ambushes: from how far, how hard
// it bites and from how close.
func crocTune(k CreatureKind) (ambush, bite, reach float64) {
	switch k {
	case CreatureBrachy:
		return 2.8, 16, 1.15
	case CreatureChampso:
		return 2.2, 10, 1.4 // a long, narrow snout: it snaps from further, weaker
	}
	return CrocAmbushDist, CrocBiteDamage, CrocBiteRange
}

// goreTune is how a horned grazer of kind k charges: its reach, the damage
// of a gore and the time between two.
func goreTune(k CreatureKind) (reach, dmg float64, cd int) {
	switch k {
	case CreatureMammoth:
		return MammothTuskRange, MammothTuskDamage, MammothTuskCD
	case CreatureToro:
		return ToroGoreRange, ToroGoreDamage, ToroGoreCD
	}
	return TrikeGoreRange, TrikeGoreDamage, TrikeGoreCD
}

// stepRunner is an Ornithomimus: it bolts from anyone close, sprints until
// it is out of breath, then stands panting.
func (w *World) stepRunner(c *Creature) {
	d := c.Kind.def()
	if c.cool > 0 {
		c.cool--
	}
	if c.rage > 0 && w.runnerKick(c) {
		return // hit close by: a kick before it runs; see skillai.go
	}
	if c.rage > 0 { // hit: off it goes, whoever is near
		c.rage = 0
		if c.State != CreatureFlee {
			c.State, c.timer, c.cool = CreatureFlee, OrnithoSprintTicks, 0
			c.hx, c.hy = 0, 0
		}
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	switch c.State {
	case CreatureFlee:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureRest, OrnithoRestTicks
			return
		}
		if p != nil {
			k := 0.35
			if c.hx == 0 && c.hy == 0 {
				k = 1 // the first stride goes straight away
			}
			c.steerAway(p.X, p.Y, k)
		}
		if !w.moveCreature(c, c.hx*d.run, c.hy*d.run) && p != nil {
			c.dodge(w, p.X, p.Y)
		}
		return
	case CreatureRest:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
		}
		return
	}
	if p != nil && dist < OrnithoFleeDist && c.cool == 0 {
		c.State, c.timer = CreatureFlee, OrnithoSprintTicks
		c.hx, c.hy = 0, 0
		return
	}
	if !w.forage(c) {
		w.wander(c)
	}
}

// stepAnzu holds its ground: a display of its crest first, then kicks.
func (w *World) stepAnzu(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	home := math.Hypot(c.homeX-c.X, c.homeY-c.Y)
	switch c.State {
	case CreatureStrike:
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	case CreatureRoar: // crest up, flapping
		if p != nil {
			c.faceToward(p.X)
		}
		if c.timer--; c.timer <= 0 {
			c.State = CreatureIdle
			if p != nil && dist < AnzuNotice {
				c.State = CreatureChase
			}
		}
		return
	case CreatureChase:
		if p == nil || dist > AnzuNotice+1.5 || home > AnzuLeash {
			c.State, c.timer, c.rage = CreatureIdle, 0, 0
			w.wander(c)
			return
		}
		c.faceToward(p.X)
		if w.trySkill(c, p, dist) || dist <= AnzuKickRange {
			return // a kick, or its crest up and two pecks: see skills.go
		}
		w.runAt(c, p.X, p.Y, d.run, AnzuKickRange*0.8)
		return
	}
	if p != nil && dist < AnzuNotice && !p.Asleep {
		// Someone too close: wings spread, crest up, a hiss, then two pecks.
		c.State, c.timer = CreatureChase, 0
		c.faceToward(p.X)
		w.trySkill(c, p, dist)
		return
	}
	if !w.forage(c) {
		w.wander(c)
	}
}

// stepQuetzal stalks the ground and stabs; badly hurt, it flies off.
func (w *World) stepQuetzal(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	if c.State == CreatureSoar { // flying: over anything, out of reach
		nx := min(max(c.X+c.hx*QuetzalFlySpeed, 1), float64(w.width)-1)
		ny := min(max(c.Y+c.hy*QuetzalFlySpeed, 1), float64(w.height)-1)
		if nx == c.X+c.hx*QuetzalFlySpeed && ny == c.Y+c.hy*QuetzalFlySpeed {
			c.FaceLeft = c.hx < 0
		} else {
			c.hx, c.hy = -c.hx, -c.hy // the edge of the world: turn back
		}
		c.X, c.Y = nx, ny
		if c.timer--; c.timer <= 0 && w.creatureFits(c.Kind, c.X, c.Y) {
			c.State, c.timer, c.cool, c.rage = CreatureIdle, 0, QuetzalGroundTicks, 0
			c.homeX, c.homeY = c.X, c.Y // lands far off, and lives here now
		}
		return
	}
	if w.huntOn(c, QuetzalNotice) {
		return // after prey: see needs.go
	}
	if c.State == CreatureStrike {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	if float64(c.HP) <= float64(c.Kind.MaxHP())*QuetzalFlyBelow && c.cool == 0 {
		c.State, c.timer = CreatureSoar, QuetzalFlyTicks // up and away
		c.hx, c.hy = 1, 0
		if p != nil && dist > 1e-9 {
			c.hx, c.hy = (c.X-p.X)/dist, (c.Y-p.Y)/dist
		}
		return
	}
	if p == nil || dist > QuetzalGiveUp || (c.State != CreatureChase && dist > QuetzalNotice && c.rage == 0) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		if !w.forage(c) {
			w.wander(c)
		}
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if w.trySkill(c, p, dist) || dist <= QuetzalStabRange {
		return // a stab, or a buffet of its wings: see skills.go
	}
	w.runAt(c, p.X, p.Y, d.run, QuetzalStabRange*0.8)
}
