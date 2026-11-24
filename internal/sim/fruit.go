package sim

import "math/rand/v2"

// Prehistoric fruit trees: figs in the woods and the grass, fruiting ginkgos
// out in the open, coconut palms along the beaches, bananas in the woods and
// the marsh (see jungle.go). Their fruit is picked by hand, like berries, and
// grows back.
//
// A tree with no fruit on it can be felled with an axe, like any other: it
// gives wood and leaves a stump, which grows back into the same tree in a
// few days, bare at first, then fruiting in its own time. Ripe fruit is
// always picked first, and the hold of Space that picked it does not go on
// to chop the tree: felling it takes a fresh press, so a grove is not cut
// down by a hand that only meant to pick.

// fruitDef is one kind of fruit tree.
type fruitDef struct {
	kind   EntityKind
	fruit  ItemKind
	yield  int
	regrow uint64
	hits   int               // blows of a stone axe to fell it
	wood   int               // what it gives felled
	share  [NumTiles]float64 // of the free tiles of each kind that grow one
}

var fruitTrees = [...]fruitDef{
	{EntityFigTree, ItemFig, 3, DayTicks, 4, 4, [NumTiles]float64{TileForest: 0.01, TileGrass: 0.006}},
	{EntityGinkgoTree, ItemGinkgoNut, 4, 3 * DayTicks / 2, 4, 4, [NumTiles]float64{TileGrass: 0.005, TileForest: 0.002}},
	{EntityPalm, ItemCoconut, 2, 2 * DayTicks, 3, 3, [NumTiles]float64{TileSand: 0.025}},
	{EntityBananaTree, ItemBanana, 3, DayTicks, 2, 2, [NumTiles]float64{TileForest: 0.016, TileMarsh: 0.008}}, // see jungle.go
}

// FruitStumpRegrowTicks is how long the stump of a felled fruit tree takes
// to stand as a tree again (its fruit comes later).
const FruitStumpRegrowTicks = 3 * DayTicks

// fruitStumpVar is the Var of a fruit tree that has been felled: a stump.
const fruitStumpVar = 1

// fruitOf is the fruit tree an entity of kind k is, if any.
func fruitOf(k EntityKind) (fruitDef, bool) {
	for _, f := range fruitTrees {
		if f.kind == k {
			return f, true
		}
	}
	return fruitDef{}, false
}

// IsFruitTree reports whether entities of kind k are fruit trees.
func IsFruitTree(k EntityKind) bool {
	_, ok := fruitOf(k)
	return ok
}

// IsFruitStump reports whether an entity of kind k with Var v is the stump
// of a felled fruit tree. The client tells it from the snapshot the same way.
func IsFruitStump(k EntityKind, v uint8) bool { return v == fruitStumpVar && IsFruitTree(k) }

// FruitStump reports whether e is the stump of a felled fruit tree.
func (e *Entity) FruitStump() bool { return IsFruitStump(e.Kind, e.Var) }

// scatterFruit plants fruit trees on free tiles. It has its own rng so the
// rest of the world does not shift.
func (w *World) scatterFruit(rng *rand.Rand) {
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			r := rng.Float64()
			t := w.tiles[ty*w.width+tx]
			for _, f := range fruitTrees {
				if r < f.share[t] {
					if w.canPlace(tx, ty) {
						w.addEntity(f.kind, tx, ty)
					}
					break
				}
				r -= f.share[t]
			}
		}
	}
}

// pickFruit picks the ripe fruit of tree e in one go; it grows back in its
// own time. The tree's HP is the chopping it takes, whole again after.
func (w *World) pickFruit(p *Player, e *Entity, f fruitDef) bool {
	return w.harvest(p, e, f.fruit, f.yield, max(e.HP, 1), func() {
		e.Ready = false
		e.RegrowAt = w.tick + f.regrow
		e.HP = maxHP(e.Kind, false)
		p.picked = e.ID // this hold of Space does not go on to chop it
	})
}

// fellFruit lands one blow of an axe on fruit tree e, which has no fruit.
// The blow that brings it down leaves a stump.
func (w *World) fellFruit(e *Entity, f fruitDef, dmg int) bool {
	return w.knock(e, dmg, func(x, y float64) {
		r := w.random()
		loot := []Ingredient{{Item: ItemWood, Count: f.wood}}
		if r.Float64() < TwigChance {
			loot = append(loot, Ingredient{Item: ItemStick, Count: 1 + r.IntN(2)})
		}
		if e.Ready {
			loot = append(loot, Ingredient{Item: f.fruit, Count: f.yield}) // what was still on it
		}
		e.Ready, e.Var = false, fruitStumpVar
		e.HP = maxHP(e.Kind, false)
		e.RegrowAt = w.tick + FruitStumpRegrowTicks
		w.dropAt(x, y, loot)
	})
}

// standFruitTree grows stump e back into its tree: bare at first, its fruit
// to come in the tree's own time.
func (w *World) standFruitTree(e *Entity) {
	f, _ := fruitOf(e.Kind)
	e.Var = 0
	e.HP = maxHP(e.Kind, false)
	e.RegrowAt = w.tick + f.regrow
}
