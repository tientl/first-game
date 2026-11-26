package sim

import "math"

// Hunger, thirst and the hunt.
//
// The great beasts get hungry and thirsty as time goes by. A grazer crops
// whatever grows where it stands, head down, chewing. A hunter goes looking
// for prey of its size, chases it down and, once it has killed, stands over
// the kill and eats. A thirsty beast walks to the nearest water's edge and
// drinks, which is when the crocodiles of the ponds strike. Hunted animals
// run until they are blown; the horned grazers instead turn and gore the
// hunter, which gives up once it is badly hurt. Players come first: a beast
// near a player stops eating or hunting to deal with them as before.
// Pets are fed by their owners and take no part in any of this.

// Needs tuning.
const (
	HuntSight        = 11.0 // a hungry hunter spots prey this far off
	HuntGiveUp       = 16.0 // and lets it go this far away
	huntSearchTicks  = TicksPerSecond
	PreyFleeDist     = 8.0                 // prey runs from a hunter after it this close
	PreyRunTicks     = 4 * TicksPerSecond  // the longest a hunted animal can run...
	PreyBlownTicks   = 2 * TicksPerSecond  // ...before it must stand, blown
	huntedTicks      = 2 * TicksPerSecond  // how long a hunter's attention lasts without renewal
	WaterSearch      = 22                  // tiles a thirsty beast looks for water
	DrinkTicks       = 4 * TicksPerSecond  // how long it drinks
	DrinkTripTicks   = 30 * TicksPerSecond // a walk to water taking longer is given up
	DefendDamage     = 3                   // a horned grazer's gore on a hunter
	HunterQuitBelow  = 0.5                 // a hunter gives up a fight below this share of its health
	huntBiteCD       = 3 * TicksPerSecond / 2
	disturbNear      = 2.5 // a beast stops eating or drinking for a player this close
	disturbSkittish  = 6.0 // or this close, for the skittish
	crocPreyBiteBits = 2   // a crocodile's bite on a drinker is its player bite over this
)

// dietDef is how a kind of creature feeds.
type dietDef struct {
	prey     []CreatureKind // what it hunts; none for a grazer
	grazes   bool           // crops what grows where it stands
	hunger   [2]int         // ticks between meals, least and most
	thirst   [2]int         // ticks between drinks, least and most
	eatTicks int            // how long a meal takes
	bite     int            // a hunter's bite on its prey
	defends  bool           // turns and gores a hunter instead of running
	skittish bool           // stops eating for a player much further off
}

var diets [NumCreatureKinds]dietDef

func init() {
	grazer := func(defends, skittish bool) dietDef {
		return dietDef{
			grazes: true, hunger: [2]int{40 * TicksPerSecond, 90 * TicksPerSecond},
			thirst: [2]int{100 * TicksPerSecond, 200 * TicksPerSecond}, eatTicks: 6 * TicksPerSecond,
			defends: defends, skittish: skittish,
		}
	}
	hunter := func(bite, eat int, prey ...CreatureKind) dietDef {
		return dietDef{
			prey: prey, hunger: [2]int{100 * TicksPerSecond, 220 * TicksPerSecond},
			thirst: [2]int{120 * TicksPerSecond, 240 * TicksPerSecond}, eatTicks: eat, bite: bite,
		}
	}
	diets[CreatureTrike] = grazer(true, false)
	diets[CreatureToro] = grazer(true, false)
	diets[CreatureStego] = grazer(true, false)
	diets[CreatureMammoth] = grazer(true, false)
	diets[CreatureBrachio] = grazer(false, false)
	diets[CreatureThesc] = grazer(false, true)
	diets[CreatureOrnitho] = grazer(false, true)
	diets[CreatureAnzu] = grazer(false, false)
	diets[CreatureRex] = hunter(12, 12*TicksPerSecond, CreatureThesc, CreatureOrnitho, CreatureTrike, CreatureToro, CreatureBrachio, CreatureAnkylo)
	diets[CreatureRaptor] = hunter(3, 8*TicksPerSecond, CreatureRabbit, CreatureThesc, CreatureOrnitho)
	diets[CreatureAchero] = hunter(2, 8*TicksPerSecond, CreatureRabbit, CreatureOrnitho, CreatureThesc)
	diets[CreatureSabre] = hunter(4, 8*TicksPerSecond, CreatureRabbit, CreatureOrnitho, CreatureThesc)
	diets[CreatureQuetzal] = hunter(3, 6*TicksPerSecond, CreatureRabbit, CreatureCrab)
}

func (k CreatureKind) diet() *dietDef {
	if k >= NumCreatureKinds {
		return &diets[CreatureNone]
	}
	return &diets[k]
}

// Feeds reports whether creatures of kind k get hungry and thirsty.
func (k CreatureKind) Feeds() bool { d := k.diet(); return d.grazes || len(d.prey) > 0 }

// Hunts reports whether creatures of kind k hunt others.
func (k CreatureKind) Hunts() bool { return len(k.diet().prey) > 0 }

// due is when, in ticks since the last meal or drink, c next wants one:
// somewhere in r, the same for c every time.
func (c *Creature) due(r [2]int, salt uint32) int {
	if r[1] <= r[0] {
		return r[0]
	}
	h := uint32(c.ID)*2654435761 ^ salt*40503
	h ^= h >> 15
	return r[0] + int(h%uint32(r[1]-r[0]))
}

// stepNeeds runs before a wild creature's own step: it keeps its hunger and
// thirst, has it run from (or fight) a hunter after it, and lets it go on
// eating or drinking. It reports whether that was the creature's whole step.
func (w *World) stepNeeds(c *Creature) bool {
	diet := c.Kind.diet()
	if c.hunter != 0 {
		if c.huntedT--; c.huntedT <= 0 {
			c.hunter = 0
		} else if h := w.creatureByID(c.hunter); h == nil || h.gone {
			c.hunter = 0
		} else if w.evade(c, h, diet) {
			return true
		}
	}
	if !c.Kind.Feeds() {
		return false
	}
	if !c.fed {
		// A world starts with its beasts at every stage of hunger and thirst.
		c.hunger = c.due(diet.hunger, 7) * int(uint32(c.ID)*31%100) / 100
		c.thirst = c.due(diet.thirst, 11) * int(uint32(c.ID)*57%100) / 100
		c.fed = true
	}
	c.hunger++
	c.thirst++
	if c.State == CreatureEat || c.State == CreatureDrink {
		if w.disturbed(c, diet) {
			c.State, c.timer = CreatureIdle, 0
			return false
		}
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, 0
		}
		return true
	}
	return false
}

// disturbed reports whether something should stop c eating or drinking.
func (w *World) disturbed(c *Creature, diet *dietDef) bool {
	if c.rage > 0 || c.hunter != 0 {
		return true
	}
	near := disturbNear
	if diet.skittish {
		near = disturbSkittish
	}
	_, d := w.nearestPlayer(c.X, c.Y)
	return d < near
}

// forage has a creature with nothing better to do drink, graze or hunt, if
// it needs to. It reports whether that was its step.
func (w *World) forage(c *Creature) bool {
	diet := c.Kind.diet()
	if c.Tamed() || !c.Kind.Feeds() {
		return false
	}
	if c.thirst >= c.due(diet.thirst, 31) && w.goDrink(c) {
		return true
	}
	if c.hunger < c.due(diet.hunger, 17) {
		return false
	}
	if diet.grazes {
		switch w.TileAt(int(math.Floor(c.X)), int(math.Floor(c.Y))) {
		case TileGrass, TileForest, TileMarsh, TileSnow, TileDesert:
			c.State, c.timer, c.hunger = CreatureEat, diet.eatTicks, 0
			return true
		}
		return false
	}
	return w.hunt(c)
}

// goDrink walks c to the nearest water's edge and has it drink there.
func (w *World) goDrink(c *Creature) bool {
	d := c.Kind.def()
	if c.trip == 0 {
		gx, gy, ok := w.waterNear(c.X, c.Y, WaterSearch)
		if !ok {
			c.thirst = 0 // no water in reach: it does without
			return false
		}
		c.goalX, c.goalY, c.trip = gx, gy, DrinkTripTicks
	}
	if c.trip--; c.trip <= 0 {
		c.trip, c.thirst = 0, 0 // a walk too far: it gives up for now
		return false
	}
	if math.Hypot(c.goalX-c.X, c.goalY-c.Y) <= d.radius+1.1 {
		c.State, c.timer = CreatureDrink, DrinkTicks
		c.thirst, c.trip = 0, 0
		c.faceToward(c.goalX)
		return true
	}
	c.State = CreatureWander
	if !w.runAt(c, c.goalX, c.goalY, d.walk*1.8, 0) {
		// The shore in the way: if the water is right there, drink.
		if math.Hypot(c.goalX-c.X, c.goalY-c.Y) <= d.radius+2 {
			c.State, c.timer = CreatureDrink, DrinkTicks
			c.thirst, c.trip = 0, 0
			c.faceToward(c.goalX)
		} else {
			c.dodge(w, c.goalX, c.goalY)
			w.moveCreature(c, c.hx*d.walk, c.hy*d.walk)
		}
	}
	return true
}

// waterNear is the centre of the nearest drinkable water (water by a shore)
// within r tiles of (x, y).
func (w *World) waterNear(x, y float64, r int) (gx, gy float64, ok bool) {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	best := math.Inf(1)
	for ty := cy - r; ty <= cy+r; ty++ {
		for tx := cx - r; tx <= cx+r; tx++ {
			if !w.isShallow(tx, ty) {
				continue
			}
			px, py := float64(tx)+0.5, float64(ty)+0.5
			if d := math.Hypot(px-x, py-y); d < best {
				best, gx, gy, ok = d, px, py, true
			}
		}
	}
	return gx, gy, ok
}

// huntOn keeps a hunter after the prey it has, unless a player is close
// enough to deal with first. It reports whether that was its step.
func (w *World) huntOn(c *Creature, sight float64) bool {
	if c.prey == 0 {
		return false
	}
	if _, d := w.nearestPlayer(c.X, c.Y); d <= sight {
		c.prey = 0 // a player first
		return false
	}
	return w.hunt(c)
}

// hunt has a hungry hunter look for prey, chase it and bite it; when the
// prey dies it eats. It reports whether that was its step.
func (w *World) hunt(c *Creature) bool {
	diet := c.Kind.diet()
	prey := w.creatureByID(c.prey)
	if prey != nil && (prey.Tamed() || prey.State.OutOfReach() || math.Hypot(prey.X-c.X, prey.Y-c.Y) > HuntGiveUp) {
		prey = nil
	}
	if float64(c.HP) < float64(c.Kind.MaxHP())*HunterQuitBelow {
		prey = nil // too badly hurt to go on
		c.hunger /= 2
	}
	if prey == nil {
		c.prey = 0
		if (w.tick+uint64(c.ID))%huntSearchTicks != 0 {
			return false
		}
		if prey = w.findPrey(c, diet.prey); prey == nil {
			return false
		}
		c.prey = prey.ID
	}
	prey.hunter, prey.huntedT = c.ID, huntedTicks
	reach := c.Kind.def().radius + prey.Kind.def().radius + 0.7
	dist := math.Hypot(prey.X-c.X, prey.Y-c.Y)
	c.faceToward(prey.X)
	if dist <= reach {
		if c.biteCD == 0 {
			c.State, c.timer, c.biteCD = CreatureStrike, AttackTicks, huntBiteCD
			if w.maul(prey, diet.bite) {
				c.State, c.timer = CreatureEat, diet.eatTicks // stands over the kill and eats
				c.prey, c.hunger = 0, 0
			}
		}
		return true
	}
	c.State = CreatureChase
	w.runAt(c, prey.X, prey.Y, c.Kind.def().run, reach*0.8)
	return true
}

// findPrey is the nearest wild creature c hunts, within its sight.
func (w *World) findPrey(c *Creature, kinds []CreatureKind) *Creature {
	var best *Creature
	bestD := HuntSight
	for _, o := range w.fauna.creatures {
		if o.gone || o == c || o.Tamed() || o.State.OutOfReach() {
			continue
		}
		hunted := false
		for _, k := range kinds {
			if o.Kind == k {
				hunted = true
				break
			}
		}
		if !hunted {
			continue
		}
		if d := math.Hypot(o.X-c.X, o.Y-c.Y); d < bestD {
			best, bestD = o, d
		}
	}
	return best
}

// maul takes dmg from a creature bitten by a beast, and reports whether
// that killed it (it is gone; nobody gets its meat).
func (w *World) maul(c *Creature, dmg int) bool {
	c.HP -= dmg * HealthScale
	c.Tame = 0
	if c.HP <= 0 {
		c.gone = true
		return true
	}
	if !c.Kind.def().tough && c.State != CreatureFlee {
		c.State, c.timer = CreatureHurt, CreatureHurtTicks
	}
	return false
}

// evade is a hunted creature's step: run from hunter h until blown, or for
// the horned grazers, turn and gore it. It reports whether it acted.
func (w *World) evade(c, h *Creature, diet *dietDef) bool {
	if c.Kind == CreatureAnkylo {
		return w.ankyloDefend(c, h) // its club, not its horns: see ankylo.go
	}
	d := c.Kind.def()
	dist := math.Hypot(h.X-c.X, h.Y-c.Y)
	if dist > PreyFleeDist {
		return false
	}
	if diet.defends {
		if c.State == CreatureStrike {
			if c.timer--; c.timer > 0 {
				return true
			}
		}
		c.faceToward(h.X)
		c.State = CreatureChase
		reach := d.radius + h.Kind.def().radius + 0.8
		if dist <= reach {
			if c.biteCD > 0 {
				c.biteCD--
			} else {
				c.State, c.timer, c.biteCD = CreatureStrike, AttackTicks, 2*TicksPerSecond
				w.maul(h, DefendDamage)
			}
			return true
		}
		w.runAt(c, h.X, h.Y, d.run*0.7, reach*0.8)
		return true
	}
	if c.State == CreatureRest { // blown: it can only stand and pant
		if c.timer--; c.timer <= 0 {
			c.State, c.timer, c.fleeT = CreatureIdle, 0, 0
		}
		return true
	}
	if c.fleeT++; c.fleeT > PreyRunTicks {
		c.State, c.timer, c.fleeT = CreatureRest, PreyBlownTicks, 0
		return true
	}
	if c.State != CreatureFlee {
		c.hx, c.hy = 0, 0
	}
	c.State = CreatureFlee
	k := 0.4
	if c.hx == 0 && c.hy == 0 {
		k = 1
	}
	c.steerAway(h.X, h.Y, k)
	if !w.moveCreature(c, c.hx*max(d.run, d.walk), c.hy*max(d.run, d.walk)) {
		c.dodge(w, h.X, h.Y)
	}
	return true
}

// drinkerNear is a creature drinking within r of the lurking crocodile c,
// for it to take.
func (w *World) drinkerNear(c *Creature, r float64) *Creature {
	for _, o := range w.fauna.drinkers { // the few drinking, not every creature: crocs ask every tick
		if o.gone || o.State != CreatureDrink || o.Tamed() || crocKind(o.Kind) {
			continue
		}
		if math.Hypot(o.X-c.X, o.Y-c.Y) <= r {
			return o
		}
	}
	return nil
}

func crocKind(k CreatureKind) bool {
	return k == CreatureCroc || k == CreatureBrachy || k == CreatureChampso
}
