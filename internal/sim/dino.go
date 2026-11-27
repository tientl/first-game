package sim

import "math"

// The great beasts of the open world. None of them lives on the old island
// maps, and the dangerous ones keep far from where players start.
//
//   - Three-horns graze the meadows in herds, plate-backs the woods in
//     pairs, long-necks wander both in small groups. They are calm until one
//     of a herd is hurt: then three-horns charge and gore whoever did it,
//     plate-backs swing their spiked tails at anyone close, and long-necks,
//     too big to bother, plod away. Each gives a lot of meat, hides and bones.
//   - Raptors hunt the deep woods in packs. They run a little slower than a
//     player but pounce from close by, then dart back; at night they see
//     further. A badly hurt raptor runs.
//   - The tyrant roams the ash round the volcano and the far deserts. It
//     roars when it sees someone, then charges; it is slower than a player,
//     but its bite is terrible. It gives tyrant teeth.
//   - Flying reptiles circle over the hills and beaches out of reach, dive
//     at a player who comes near, then climb away. They land now and then.
//   - Sea reptiles swim off the coast under water and rear up to snap at
//     anyone on the shore within reach of their necks.

// Dinosaur tuning. Speeds are in tiles per tick.
const (
	CandidateRange = 5.0 // more than any creature's reach, spear included
	toughHurtTicks = 3   // a big beast's flinch

	HerdRageTicks = 12 * TicksPerSecond // how long a herd stays angry once one of them is hurt
	HerdAlarmDist = 8.0                 // herd mates this close join in
	GrazerGiveUp  = 11.0                // a grazer lets a foe this far off go
	AttackTicks   = 8                   // how long a strike pose lasts

	TrikeGoreRange  = 2.3
	TrikeGoreDamage = 22.0
	TrikeGoreCD     = 5 * TicksPerSecond / 2
	StegoTailRange  = 2.8
	StegoTailDamage = 20.0
	StegoTailCD     = 2 * TicksPerSecond

	RaptorSightDay     = 7.0
	RaptorSightNight   = 10.0
	RaptorGiveUp       = 15.0
	RaptorPounceDist   = 2.8
	RaptorPounceSpeed  = 8.0 / TicksPerSecond
	RaptorPounceTicks  = 6
	RaptorBiteRange    = 1.2
	RaptorBiteDamage   = 10.0
	RaptorBiteCD       = 2 * TicksPerSecond
	RaptorBackOffTicks = 3 * TicksPerSecond / 2
	RaptorFleeBelow    = 2 // hit points at which a raptor runs for it
	RaptorFleeTicks    = 4 * TicksPerSecond
	RaptorKeepOff      = 1.5 // how close a raptor runs up before it pounces

	RexSight      = 8.0
	RexGiveUp     = 16.0
	RexRoarTicks  = 24
	RexBiteRange  = 2.6
	RexBiteDamage = 34.0
	RexBiteCD     = 7 * TicksPerSecond / 2
	RexChaseTicks = 20 * TicksPerSecond
	RexCoolTicks  = 8 * TicksPerSecond // after it gives up, it ignores players this long

	PteroSight       = 6.0
	PteroDiveSpeed   = 7.0 / TicksPerSecond
	PteroCircle      = 4.0 // the radius of its circling
	PteroStrikeRange = 1.0
	PteroDamage      = 8.0
	PteroDiveCD      = 6 * TicksPerSecond
	PteroDiveTicks   = 3 * TicksPerSecond // the longest dive
	PteroClimbTicks  = TicksPerSecond
	PteroPerchChance = 0.002 // per tick while soaring: it lands
	PteroPerchMin    = 5 * TicksPerSecond
	PteroPerchMax    = 12 * TicksPerSecond

	PlesioNotice       = 5.0 // it comes up for a player on the shore this close
	PlesioBiteRange    = 2.8 // its neck reaches this far
	PlesioBiteDamage   = 16.0
	PlesioBiteCD       = 2 * TicksPerSecond
	PlesioSurfaceTicks = 6 * TicksPerSecond
	PlesioDiveTicks    = 5 * TicksPerSecond // under water after a while up, before it comes up again
	PlesioBreathe      = 0.0015             // per tick under water: it comes up for air
	plesioNear         = 2                  // it lives this many tiles off the coast...
	plesioFar          = 6                  // ...and no further
)

var (
	trikeDef = creatureDef{
		name: "Khủng long ba sừng", hp: 30,
		walk: 0.9 / TicksPerSecond, run: 3.6 / TicksPerSecond,
		radius: 0.75, reach: 2.6,
		yield: ItemMeat, yieldN: 6, drops: []Ingredient{{ItemHide, 2}, {ItemBone, 1}},
		leash: 14, idle: [2]int{40, 160}, roam: [2]int{30, 90},
		spawnOn: []Tile{TileGrass}, tilesPer: 1500,
		fights: true, tough: true, openOnly: true, herd: 4, away: 14,
	}
	stegoDef = creatureDef{
		name: "Khủng long gai lưng", hp: 34,
		walk: 0.7 / TicksPerSecond, run: 1.6 / TicksPerSecond,
		radius: 0.8, reach: 2.7,
		yield: ItemMeat, yieldN: 6, drops: []Ingredient{{ItemHide, 2}, {ItemBone, 2}},
		leash: 10, idle: [2]int{60, 200}, roam: [2]int{30, 80},
		spawnOn: []Tile{TileForest}, tilesPer: 2500,
		fights: true, tough: true, openOnly: true, herd: 2, away: 14,
	}
	brachioDef = creatureDef{
		name: "Khủng long cổ dài", hp: 70,
		walk: 0.8 / TicksPerSecond, run: 1.4 / TicksPerSecond,
		radius: 1.05, reach: 3.2,
		yield: ItemMeat, yieldN: 12, drops: []Ingredient{{ItemHide, 4}, {ItemBone, 4}},
		leash: 18, idle: [2]int{60, 240}, roam: [2]int{40, 120},
		spawnOn: []Tile{TileGrass, TileForest}, tilesPer: 6000,
		tough: true, openOnly: true, herd: 3, away: 20,
	}
	raptorDef = creatureDef{
		name: "Dakotaraptor", hp: 12,
		walk: 1.4 / TicksPerSecond, run: 3.4 / TicksPerSecond, // slower than a player: running away works
		radius: 0.36, reach: 1.7,
		yield: ItemMeat, yieldN: 2, drops: []Ingredient{{ItemHide, 1}, {ItemBone, 1}},
		leash: 12, idle: [2]int{30, 120}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileForest}, tilesPer: 2500,
		fights: true, openOnly: true, herd: 3, away: 40,
	}
	rexDef = creatureDef{
		name: "Khủng long bạo chúa", hp: 200,
		walk: 0.9 / TicksPerSecond, run: 3.3 / TicksPerSecond,
		radius: 0.95, reach: 3.1,
		yield: ItemMeat, yieldN: 10, drops: []Ingredient{{ItemHide, 4}, {ItemBone, 4}, {ItemRexTooth, 2}},
		leash: 20, idle: [2]int{60, 200}, roam: [2]int{40, 120},
		spawnOn: []Tile{TileAsh, TileDesert}, tilesPer: 6000,
		fights: true, tough: true, openOnly: true, away: 60,
	}
	pteroDef = creatureDef{
		name: "Thằn lằn bay", hp: 4,
		walk: 2.4 / TicksPerSecond, run: 7.0 / TicksPerSecond,
		radius: 0.3, reach: 1.8,
		yield: ItemMeat, yieldN: 2, drops: []Ingredient{{ItemHide, 1}},
		leash: 8, idle: [2]int{20, 60}, roam: [2]int{20, 60},
		spawnOn: []Tile{TileRocky, TileSand}, tilesPer: 1200,
		fights: true, flies: true, openOnly: true, away: 16,
	}
	plesioDef = creatureDef{
		name: "Thằn lằn cổ dài", hp: 24,
		walk: 1.0 / TicksPerSecond, run: 2.2 / TicksPerSecond,
		radius: 0.55, reach: 3.0, // a player on the shore hits its neck from there
		yield: ItemMeat, yieldN: 6, drops: []Ingredient{{ItemBone, 2}, {ItemHide, 1}},
		leash: 7, idle: [2]int{20, 80}, roam: [2]int{30, 90},
		spawnOn: []Tile{TileWater}, tilesPer: 4000,
		fights: true, tough: true, deep: true, openOnly: true, away: 20,
	}
)

// addHerd puts up to n creatures of kind k around (x, y) and returns how
// many fitted.
func (w *World) addHerd(k CreatureKind, x, y float64, n int) int {
	r := w.random()
	got := 0
	for i := range n {
		for range 8 {
			hx, hy := x, y
			if i > 0 {
				a, d := r.Float64()*2*math.Pi, 1.2+r.Float64()*2.4
				hx, hy = x+math.Cos(a)*d, y+math.Sin(a)*d
			}
			if w.creatureFits(k, hx, hy) {
				c := w.addCreature(k, hx, hy)
				c.homeX, c.homeY = x, y // the herd keeps together round one place
				got++
				break
			}
		}
	}
	return got
}

// coastDistance returns, for every tile, how many steps of water lie
// between it and the nearest land (0 for land), counted up to most.
func (w *World) coastDistance(most int) []uint8 {
	n := w.width * w.height
	dist := make([]uint8, n)
	var queue []int32
	for i, t := range w.tiles {
		if t == TileWater {
			dist[i] = uint8(most)
		} else {
			queue = append(queue, int32(i))
		}
	}
	for len(queue) > 0 {
		i := int(queue[0])
		queue = queue[1:]
		x, y := i%w.width, i/w.width
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !w.inBounds(nx, ny) {
				continue
			}
			j := ny*w.width + nx
			if dist[j] > dist[i]+1 {
				dist[j] = dist[i] + 1
				queue = append(queue, int32(j))
			}
		}
	}
	return dist
}

// provoke is what a creature does about being hit by p.
func (w *World) provoke(c *Creature, p *Player) {
	w.skillProvoked(c, p) // a herd call, a kick: see skills.go
	switch c.Kind {
	case CreatureTrike, CreatureStego, CreatureBrachio, CreatureRaptor, CreatureMammoth, CreatureSabre,
		CreatureToro, CreatureThesc, CreatureAchero, CreatureAnzu, CreatureOrnitho, CreatureAnkylo:
		// The whole herd or pack turns on p.
		for _, o := range w.fauna.creatures {
			if o.Kind == c.Kind && !o.gone && math.Hypot(o.X-c.X, o.Y-c.Y) <= HerdAlarmDist {
				o.foe, o.rage = p.ID, HerdRageTicks
				if o.Kind == CreatureBrachio {
					o.foe, o.rage = p.ID, HerdRageTicks/2
				}
			}
		}
		if c.Kind == CreatureTrike || c.Kind == CreatureStego || c.Kind == CreatureMammoth || c.Kind == CreatureToro || c.Kind == CreatureAnzu {
			c.after = CreatureChase
		}
		if c.Kind == CreatureBrachio || c.Kind == CreatureThesc {
			c.after = CreatureFlee
		}
		if c.Kind == CreatureAnkylo && c.after != CreatureStrike {
			c.after = CreatureRoar // its back to p, the club ready: see ankylo.go
		}
	case CreatureQuetzal:
		c.foe, c.rage = p.ID, RexChaseTicks
	case CreatureRex:
		c.foe, c.rage, c.cool = p.ID, RexChaseTicks, 0
		c.after, c.afterT = CreatureChase, RexChaseTicks
	case CreaturePtero:
		c.foe, c.rage = p.ID, PteroDiveTicks
	}
}

// foeOf returns the player c is angry with, if it still is and they are
// still about, and how far off they are.
func (w *World) foeOf(c *Creature) (*Player, float64) {
	if c.rage <= 0 {
		return nil, math.Inf(1)
	}
	c.rage--
	p := w.players[c.foe]
	if p == nil || p.Dead {
		c.rage = 0
		return nil, math.Inf(1)
	}
	return p, math.Hypot(p.X-c.X, p.Y-c.Y)
}

// faceToward turns a side-on creature to look at x.
func (c *Creature) faceToward(x float64) {
	if dx := x - c.X; math.Abs(dx) > 0.05 {
		c.FaceLeft = dx < 0
	}
}

// runAt moves c up to step towards (x, y), stopping keep short of it, and
// reports whether it moved.
func (w *World) runAt(c *Creature, x, y, step, keep float64) bool {
	dx, dy := x-c.X, y-c.Y
	d := math.Hypot(dx, dy)
	if d <= keep || d < 1e-9 {
		return false
	}
	step = min(step, d-keep)
	moved := w.moveCreature(c, dx/d*step, dy/d*step)
	c.faceToward(x)
	return moved
}

func (w *World) stepGrazer(c *Creature) {
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.State == CreatureStrike {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	p, dist := w.foeOf(c)
	if p == nil || dist > GrazerGiveUp {
		c.rage = 0
		if c.State == CreatureChase || c.State == CreatureFlee {
			c.State, c.timer = CreatureIdle, 0
		}
		if w.grazerWarn(c) { // a Torosaurus's frill at someone too close: see skillai.go
			return
		}
		if !w.forage(c) { // see needs.go
			w.wander(c)
		}
		return
	}
	w.grazerFight(c, p, dist) // charges, tosses, tail swings, stomps: see skillai.go
}

func (w *World) stepRaptor(c *Creature) {
	d := c.Kind.def()
	_, _, sightDay, sightNight, cowerAt := raptorTune(c.Kind) // see hellcreek.go
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	if w.huntOn(c, sightDay) {
		return // after prey: see needs.go
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	switch c.State {
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
	}
	if c.State == CreatureCower { // beaten: it lies low a while, then runs
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureFlee, RaptorFleeTicks
			c.hx, c.hy = 0, 0
		}
		return
	}
	if p != nil && c.HP <= cowerAt*HealthScale && dist < RaptorGiveUp {
		c.State, c.timer = CreatureCower, RaptorCowerTicks // badly hurt: it gives up the fight
		c.rage = 0
		return
	}
	sight := sightDay
	if Ambient(w.tick) < 0.3 {
		sight = sightNight
	}
	if p == nil || c.cool > 0 || dist > RaptorGiveUp || (c.State != CreatureChase && dist > sight) {
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
	if w.packFight(c, p, dist, func(dx, dy float64) { w.moveCreature(c, dx, dy) }) {
		return // a pounce, a slash, a screech, or circling while the pack takes turns: see skillai.go
	}
	w.runAt(c, p.X, p.Y, d.run, RaptorKeepOff)
}

func (w *World) stepRex(c *Creature) {
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	if w.huntOn(c, RexSight) {
		return // after prey: see needs.go
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	switch c.State {
	case CreatureRoar:
		if p != nil {
			c.faceToward(p.X)
		}
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureChase, RexChaseTicks
		}
		return
	case CreatureStrike:
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	case CreatureChase:
		c.timer--
		if p == nil || dist > RexGiveUp || c.timer <= 0 {
			c.State, c.timer, c.cool, c.rage = CreatureIdle, 0, RexCoolTicks, 0
			w.wander(c)
			return
		}
		w.rexFight(c, p, dist) // a bite, a roar, a tail sweep, a charge: see skillai.go
		return
	}
	if p != nil && c.cool == 0 && dist <= RexSight {
		// It has seen someone: it roars (a stun, close by), then charges.
		c.State, c.timer = CreatureChase, RexChaseTicks
		w.startSkill(c, SkillRexRoar, p)
		return
	}
	if !w.forage(c) {
		w.wander(c)
	}
}

func (w *World) stepPtero(c *Creature) {
	d := c.Kind.def()
	r := w.random()
	if c.biteCD > 0 {
		c.biteCD--
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	switch c.State {
	case CreatureFlee: // climbing away
		if p != nil {
			c.steerAway(p.X, p.Y, 0.3)
		}
		w.moveCreature(c, c.hx*d.walk*1.5, c.hy*d.walk*1.5)
		if c.timer--; c.timer <= 0 {
			c.State = CreatureSoar
			c.ang = math.Atan2(c.Y-c.homeY, c.X-c.homeX)
		}
		return
	case CreatureIdle: // perched
		if c.timer--; c.timer <= 0 {
			c.State = CreatureSoar
			c.ang = math.Atan2(c.Y-c.homeY, c.X-c.homeX)
		}
		return
	}
	// Soaring: round and round its home, high up; or gliding in high over a
	// foe, to hold still over them and dive (skills.go).
	c.State = CreatureSoar
	if p != nil && !p.Asleep && (dist <= PteroSight || c.rage > 0) && w.pteroWants(c, p) {
		if dist <= 0.8 && w.trySkill(c, p, dist) {
			return
		}
		w.runAt(c, p.X, p.Y, PteroDiveSpeed, 0)
		return
	}
	c.ang += d.walk / PteroCircle
	gx, gy := c.homeX+math.Cos(c.ang)*PteroCircle, c.homeY+math.Sin(c.ang)*PteroCircle*0.7
	w.runAt(c, gx, gy, d.walk*1.4, 0)
	if r.Float64() < PteroPerchChance && w.TileAt(int(c.X), int(c.Y)).Walkable() && w.TileAt(int(c.X), int(c.Y)) != TileCave {
		c.State, c.timer = CreatureIdle, PteroPerchMin+r.IntN(PteroPerchMax-PteroPerchMin+1)
	}
}

func (w *World) stepPlesio(c *Creature) {
	d := c.Kind.def()
	r := w.random()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.cool > 0 {
		c.cool--
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	switch c.State {
	case CreatureStrike:
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, max(c.timer, PlesioSurfaceTicks/2)
		}
		return
	case CreatureIdle, CreatureChase: // up, its neck out of the water
		if c.timer--; c.timer <= 0 {
			c.State, c.cool = CreatureLurk, PlesioDiveTicks
			return
		}
		if p == nil || dist > PlesioNotice+2 {
			c.State = CreatureIdle
			return
		}
		c.State = CreatureChase
		c.faceToward(p.X)
		if w.trySkill(c, p, dist) || dist <= PlesioBiteRange {
			return // its neck reared, then the bite: see skills.go
		}
		w.runAt(c, p.X, p.Y, d.run, PlesioBiteRange*0.7)
		return
	}
	// Under water.
	c.State = CreatureLurk
	if p != nil && dist <= Skill(SkillPlesioAmbush).MaxD && w.trySkill(c, p, dist) {
		return // a swimmer or a boat: a ripple closes in, then it comes up under them
	}
	if c.cool == 0 && ((p != nil && dist <= PlesioNotice) || r.Float64() < PlesioBreathe) {
		c.State, c.timer = CreatureIdle, PlesioSurfaceTicks
		return
	}
	w.swim(c)
}

// swim is wander for a creature that stays under water (CreatureLurk).
func (w *World) swim(c *Creature) {
	d := c.Kind.def()
	r := w.random()
	if c.timer > 0 {
		c.timer--
	}
	if c.timer == 0 {
		c.timer = d.roam[0] + r.IntN(d.roam[1]-d.roam[0]+1)
		a := r.Float64() * 2 * math.Pi
		c.hx, c.hy = math.Cos(a), math.Sin(a)
		if hd := math.Hypot(c.homeX-c.X, c.homeY-c.Y); hd > d.leash {
			c.hx, c.hy = (c.homeX-c.X)/hd, (c.homeY-c.Y)/hd
		}
	}
	if !w.moveCreature(c, c.hx*d.walk, c.hy*d.walk) {
		c.timer = 1 // bumped into the shore: think again
	}
}
