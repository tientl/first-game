package sim

import (
	"math"
	"math/rand/v2"
)

// The woods.
//
//   - Banana trees grow in the woods and the marsh: their bunches are picked
//     by hand, like any fruit tree's, and grow back. A bare one can be felled
//     with an axe for a little wood; its stump grows back (see fruit.go).
//   - Now and then a giant of the old forest towers over the rest: a trunk
//     three tiles wide that nothing walks through. It takes a great many
//     blows of an axe and gives a great heap of wood; its stump stays.
//   - Bees nest in some trees, giants most of all. Space at a hive takes its
//     honey; it fills again in a day. The bees swarm whoever robs or fells
//     their tree and sting until they are left behind, or the thief jumps in
//     the water. A burning torch in the hand smokes them calm.
//   - Troops of monkeys live in the woods. They flee a player, rest when
//     blown, and can be tamed with bananas.

// Woods tuning.
const (
	GiantW, GiantD  = 3, 2 // a giant's foot, in tiles: wide, deep
	GiantTreeHits   = 24   // blows with a stone axe
	GiantWood       = 30   // what a felled giant gives
	GiantSticks     = 8
	GiantSeeds      = 3
	giantPerForest  = 1400 // a giant per this many woodland tiles
	giantSpacing    = 22   // tiles between two giants at least
	giantHiveChance = 0.6  // a giant has a hive this often
	treeHiveChance  = 1.0 / 40
	HoneyYield      = 3
	HiveRefillTicks = DayTicks
	BeeSwarmTicks   = 8 * TicksPerSecond // how long a swarm keeps after a thief
	BeeStingCD      = TicksPerSecond
	BeeSting        = 3.0
	BeeRange        = 10.0 // a thief this far from the hive is let go
	hiveVar         = 1    // Var of a tree with bees in it
)

var monkeyDef = creatureDef{
	name: "Khỉ", hp: 5,
	walk: 1.5 / TicksPerSecond, run: 5.2 / TicksPerSecond, // quicker than a player
	radius: 0.3, reach: 1.6,
	yield: ItemMeat, yieldN: 2, drops: []Ingredient{{ItemFur, 1}},
	leash: 12, idle: [2]int{30, 110}, roam: [2]int{20, 70},
	spawnOn: []Tile{TileForest}, tilesPer: 300,
	openOnly: true, herd: 4, away: 10,
}

func init() {
	creatureDefs[CreatureMonkey] = monkeyDef
	diets[CreatureMonkey] = dietDef{
		grazes: true, hunger: [2]int{40 * TicksPerSecond, 90 * TicksPerSecond},
		thirst: [2]int{100 * TicksPerSecond, 200 * TicksPerSecond}, eatTicks: 4 * TicksPerSecond,
		skittish: true,
	}
	tameDefs[CreatureMonkey] = tameDef{food: ItemBanana, meals: 3, bite: 1}
}

// HasHive reports whether bees nest in e.
func (e *Entity) HasHive() bool {
	return (e.Kind == EntityTree || e.Kind == EntityGiantTree) && e.Var == hiveVar && e.Ready
}

// HoneyReady reports whether e's hive has honey to take at tick.
func (e *Entity) HoneyReady(tick uint64) bool { return e.HasHive() && tick >= e.HoneyAt }

// --- making the woods ----------------------------------------------------------

// growGiants stands the giants of the old forest in the woods, and lets bees
// nest in some trees. It has its own rng so the rest of the world does not
// shift.
func (w *World) growGiants(rng *rand.Rand) {
	var woods []int
	for i, t := range w.tiles {
		if t == TileForest {
			woods = append(woods, i)
		}
	}
	want := len(woods) / giantPerForest
	rng.Shuffle(len(woods), func(a, b int) { woods[a], woods[b] = woods[b], woods[a] })
	var placed [][2]int
	for _, i := range woods {
		if len(placed) >= want {
			break
		}
		cx, cy := i%w.width, i/w.width
		if !w.giantFits(cx, cy) {
			continue
		}
		near := false
		for _, q := range placed {
			if abs(q[0]-cx)+abs(q[1]-cy) < giantSpacing {
				near = true
				break
			}
		}
		if near || math.Hypot(float64(cx)-w.spawnX, float64(cy)-w.spawnY) < 6 {
			continue
		}
		g := w.raiseGiant(cx, cy)
		if rng.Float64() < giantHiveChance {
			g.Var = hiveVar
		}
		placed = append(placed, [2]int{cx, cy})
	}
	for _, e := range w.entities {
		if e.Kind == EntityTree && w.TileAt(e.TX, e.TY) == TileForest && rng.Float64() < treeHiveChance {
			e.Var = hiveVar
		}
	}
}

// giantFits reports whether a giant's foot fits with (cx, cy) the middle of
// its front edge: woodland under it all, a free tile in front.
func (w *World) giantFits(cx, cy int) bool {
	for ty := cy - GiantD + 1; ty <= cy+1; ty++ {
		for tx := cx - GiantW/2; tx <= cx+GiantW/2; tx++ {
			if t := w.TileAt(tx, ty); t != TileForest && t != TileGrass {
				return false
			}
			if ty == cy+1 && tx != cx {
				continue
			}
			if e := w.entityAt(tx, ty); e != nil && !wild(e.Kind) {
				return false
			}
		}
	}
	return true
}

// wild reports whether entities of kind k are only what grows or lies about
// in the woods, which a giant may stand in place of.
func wild(k EntityKind) bool {
	switch k {
	case EntityTree, EntityGrassTuft, EntityTwig, EntityPebble, EntityBerryBush, EntityFlower, EntityMushroom,
		EntityBoulder, EntityFigTree, EntityGinkgoTree, EntityBananaTree:
		return true
	}
	return false
}

// raiseGiant stands a giant with the middle of its front edge on (cx, cy).
func (w *World) raiseGiant(cx, cy int) *Entity {
	for ty := cy - GiantD + 1; ty <= cy+1; ty++ {
		for tx := cx - GiantW/2; tx <= cx+GiantW/2; tx++ {
			if e := w.entityAt(tx, ty); e != nil {
				w.removeEntity(e)
			}
		}
	}
	for ty := cy - GiantD + 1; ty <= cy; ty++ {
		for tx := cx - GiantW/2; tx <= cx+GiantW/2; tx++ {
			if tx != cx || ty != cy {
				w.addEntity(EntityGiantRoot, tx, ty)
			}
		}
	}
	return w.addEntity(EntityGiantTree, cx, cy)
}

// --- bees and honey ------------------------------------------------------------

// takeHoney robs e's hive, if it has honey. The bees go for p unless p holds a
// burning torch. It reports whether there was a hive to rob: when there was
// not, the blow falls on the tree.
func (w *World) takeHoney(p *Player, e *Entity) bool {
	if !e.HoneyReady(w.tick) {
		return false
	}
	e.HoneyAt = w.tick + HiveRefillTicks
	w.dropAt(float64(e.TX)+0.5, float64(e.TY)+0.8, []Ingredient{{Item: ItemHoney, Count: HoneyYield}})
	w.angerBees(p, e)
	return true
}

// angerBees sets the bees of e's hive on p, unless p smokes them with a torch.
func (w *World) angerBees(p *Player, e *Entity) {
	if p.Held() == ItemTorch {
		p.notify(NoticeBeesCalm, 0)
		return
	}
	p.bees, p.beeHive = BeeSwarmTicks, [2]float64{float64(e.TX) + 0.5, float64(e.TY) + 0.5}
	p.notify(NoticeBees, 0)
}

// stepBees stings p while a swarm is after them.
func (w *World) stepBees(p *Player) {
	if p.bees <= 0 {
		return
	}
	if p.Dead || w.Swimming(p) || math.Hypot(p.X-p.beeHive[0], p.Y-p.beeHive[1]) > BeeRange {
		p.bees = 0 // under the water, or left behind
		return
	}
	p.bees--
	if p.bees%BeeStingCD == 0 {
		hurt(p, BeeSting)
		p.flinch()
	}
}

// Swarmed reports whether bees are after p.
func (p *Player) Swarmed() bool { return p.bees > 0 }

// --- felling a giant -------------------------------------------------------------

// fellGiant lands one blow of an axe on giant e.
func (w *World) fellGiant(p *Player, e *Entity, dmg int) bool {
	if e.HasHive() && w.random().Float64() < 0.25 {
		w.angerBees(p, e) // the hive shakes
	}
	return w.knock(e, dmg, func(x, y float64) {
		loot := []Ingredient{{ItemWood, GiantWood}, {ItemStick, GiantSticks}, {ItemTreeSeed, GiantSeeds}}
		if e.HoneyReady(w.tick) {
			loot = append(loot, Ingredient{ItemHoney, HoneyYield})
		}
		if e.HasHive() {
			w.angerBees(p, e)
		}
		e.Ready, e.Var = false, 0   // a great stump, for good
		e.RegrowAt = math.MaxUint64 // giants do not grow back in a lifetime
		w.dropAt(x, y+0.6, loot)
	})
}
