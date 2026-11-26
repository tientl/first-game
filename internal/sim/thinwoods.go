package sim

import (
	"cmp"
	"math"
	"slices"
)

// Thinning the woods of an older save. A world saved before version 20 has
// its trees as thick as they were made, which, since the trees were drawn
// at the player's scale, is a wall of leaves (see woods.go). Its woods are
// thinned to a plan: going from the biggest crowns down, a tree keeps its
// place unless one already kept stands closer than their spacing allows,
// with room to spare in ordinary woods and less in the thick stretches the
// woods' noise marks (see woodsThickness). The plan is made from the seed
// (planWoods); the woods grow back to it (regrow.go).
//
// On loading an older save, the wild trees that are not in the plan go,
// and now and then some undergrowth takes their place; then the bees leave
// trees until there is a hive to about hivesPerTrees trees. What players
// planted or built by is never touched:
//
//   - trees that were not in the world as it was made (grown from a seed
//     someone planted) stay, and so do trees someone has felled or started
//     to chop, and hives someone has robbed;
//   - so do the trees near anything built, round a boss's lair, and near a
//     cave mouth, the pyramid or an old chest;
//   - giants (and their hives), fruit trees and the props of the lairs stay,
//     though wild trees about them may go.

// Thinning tuning.
const (
	thinOrdinary  = 1.45 // the spacing kept, of each pair's spacing, in ordinary woods...
	thinThick     = 1.05 // ...and in thick ones
	thinNearBuilt = 5    // tiles round anything built where no tree goes
	thinNearLair  = 6    // and round a boss's lair: its hollow, its nest
	thinNearSight = 4    // and round a cave mouth, the pyramid, an old chest
)

// builtKind reports whether an entity of kind k with Var v is something a
// player made or planted.
func builtKind(k EntityKind, v uint8) bool {
	switch k {
	case EntityCampfire, EntityWall, EntityBed, EntityFarmland, EntityCrop, EntityFurnace, EntityStoneWall,
		EntityChest, EntityDoor, EntityFloor, EntitySpikes, EntityTrap, EntityTorchStand, EntitySapling,
		EntityTrophy, EntityGrave, EntityMortar, EntityOven, EntityDryRack:
		return true
	case EntityManEater:
		return v == 1 // a planted guard
	}
	return false
}

// spared is, for every tile, whether the trees on it are left be: near
// what players built, a boss's lair, or a sight of the world.
func (w *World) spared() []bool {
	keep := make([]bool, len(w.tiles))
	around := func(x, y, r int) {
		for ty := max(y-r, 0); ty <= min(y+r, w.height-1); ty++ {
			for tx := max(x-r, 0); tx <= min(x+r, w.width-1); tx++ {
				if (tx-x)*(tx-x)+(ty-y)*(ty-y) <= r*r {
					keep[ty*w.width+tx] = true
				}
			}
		}
	}
	for _, e := range w.entities {
		switch {
		case builtKind(e.Kind, e.Var):
			around(e.TX, e.TY, thinNearBuilt)
		case e.Kind == EntityCaveMouth, e.Kind == EntityPyramid, e.Kind == EntityTreasure:
			around(e.TX, e.TY, thinNearSight)
		}
	}
	for _, l := range w.bosses.lairs {
		if l.ok {
			around(int(l.x), int(l.y), thinNearLair)
		}
	}
	return keep
}

// thinLayout is the trees of w that keep their places when its woods are
// thinned, thick being how thick each tile's wood is. Giants and fruit trees
// keep theirs, and make room for themselves.
func (w *World) thinLayout(thick []float32) []*Entity {
	type stand struct {
		e   *Entity
		c   crown
		pri uint32
	}
	var all []stand
	for _, e := range w.entities {
		if c, ok := w.crownOf(e.Kind, e.TX, e.TY, e.Big); ok {
			all = append(all, stand{e, c, hash2(int32(e.TX), int32(e.TY), 0x7417)})
		}
	}
	// The biggest crowns first; between equals, by a hash of the place.
	slices.SortFunc(all, func(a, b stand) int {
		if c := cmp.Compare(b.c.spacing, a.c.spacing); c != 0 {
			return c
		}
		if c := cmp.Compare(a.pri, b.pri); c != 0 {
			return c
		}
		return cmp.Compare(a.e.ID, b.e.ID)
	})
	W := w.width
	kept := make([]float32, len(w.tiles)) // the spacing of the tree kept on each tile
	reach := int(math.Ceil((giantCrown.spacing + maxSpacing) / 2 * thinOrdinary))
	tooClose := func(x, y int, s, apart float64) bool {
		for ty := max(y-reach, 0); ty <= min(y+reach, w.height-1); ty++ {
			for tx := max(x-reach, 0); tx <= min(x+reach, W-1); tx++ {
				o := kept[ty*W+tx]
				if o == 0 {
					continue
				}
				d := (s + float64(o)) / 2 * apart
				if dx, dy := float64(tx-x), float64(ty-y); dx*dx+dy*dy < d*d {
					return true
				}
			}
		}
		return false
	}
	var out []*Entity
	for _, s := range all {
		e := s.e
		if e.Kind == EntityTree {
			th := 0.0
			if i := e.TY*W + e.TX; i < len(thick) {
				th = float64(thick[i])
			}
			if tooClose(e.TX, e.TY, s.c.spacing, lerp(thinOrdinary, thinThick, th)) {
				continue
			}
			out = append(out, e)
		}
		kept[e.TY*W+e.TX] = float32(s.c.spacing)
	}
	return out
}

// thinWoods thins the trees of a world loaded from an older save to the
// plan of its woods, and the hives to its trees. made holds where each tree
// of the world as it was made stood, by its id: a tree that is not there was
// planted. It returns how many trees went.
func (w *World) thinWoods(made map[EntityID][2]int) int {
	keep := w.spared()
	gone := 0
	for _, e := range w.sortedEntities() {
		// A wild tree, untouched: one someone has felled, started on, or
		// robbed the bees of stays, with the work done on it.
		wild := e.Kind == EntityTree && made[e.ID] == [2]int{e.TX, e.TY} && !keep[e.TY*w.width+e.TX] &&
			e.Ready && e.HP >= maxHP(e.Kind, e.Big) && !(e.Var == hiveVar && e.HoneyAt != 0)
		if !wild || w.planned(e.TX, e.TY) != planNone {
			continue
		}
		x, y := e.TX, e.TY
		w.removeEntity(e)
		w.undergrowthFor(x, y)
		gone++
	}
	w.capHives()
	return gone
}

// undergrowthFor puts down, now and then, something small where a tree was
// taken away: a tuft of tall grass, a fallen branch, a bush, or mushrooms on
// a rotting log.
func (w *World) undergrowthFor(x, y int) {
	r := unit(hash2(int32(x), int32(y), 0x7c1))
	var kind EntityKind
	switch w.TileAt(x, y) {
	case TileForest:
		kind = pick(r, chance{EntityGrassTuft, 0.18}, chance{EntityTwig, 0.1}, chance{EntityBerryBush, 0.04}, chance{EntityMushroom, 0.05})
	case TileGrass:
		kind = pick(r, chance{EntityGrassTuft, 0.25}, chance{EntityTwig, 0.05})
	case TileSnow:
		kind = pick(r, chance{EntityTwig, 0.12})
	case TileMarsh:
		kind = pick(r, chance{EntityGrassTuft, 0.1}, chance{EntityMushroom, 0.05})
	}
	if kind != EntityNone && w.canPlace(x, y) {
		w.addEntity(kind, x, y)
	}
}
