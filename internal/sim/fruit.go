package sim

import "math/rand/v2"

// Prehistoric fruit trees: figs in the woods and the grass, fruiting ginkgos
// out in the open, coconut palms along the beaches. Their fruit is picked by
// hand, like berries, and grows back; the trees themselves are not felled,
// so a grove keeps feeding whoever finds it.

// fruitDef is one kind of fruit tree.
type fruitDef struct {
	kind   EntityKind
	fruit  ItemKind
	yield  int
	regrow uint64
	share  [NumTiles]float64 // of the free tiles of each kind that grow one
}

var fruitTrees = [...]fruitDef{
	{EntityFigTree, ItemFig, 3, DayTicks, [NumTiles]float64{TileForest: 0.01, TileGrass: 0.006}},
	{EntityGinkgoTree, ItemGinkgoNut, 4, 3 * DayTicks / 2, [NumTiles]float64{TileGrass: 0.005, TileForest: 0.002}},
	{EntityPalm, ItemCoconut, 2, 2 * DayTicks, [NumTiles]float64{TileSand: 0.025}},
}

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

// pickFruit picks the ripe fruit of tree e; it grows back in its own time.
func (w *World) pickFruit(p *Player, e *Entity, f fruitDef) bool {
	return w.harvest(p, e, f.fruit, f.yield, HitPower, func() {
		e.Ready = false
		e.RegrowAt = w.tick + f.regrow
	})
}
