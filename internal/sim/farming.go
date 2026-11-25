package sim

import "math"

// Farming. A hoe tills the tile in front of the player into a bed of soil,
// sometimes turning up a worm or a sweet potato. A sweet potato planted in
// the bed grows into a crop that is dug up for more. A bed left empty goes
// back to grass after a while.

// CropStages is how many looks a growing crop has before it is ripe.
const CropStages = 3

// Soft reports whether a hoe can till tiles of this kind.
func (t Tile) Soft() bool {
	switch t {
	case TileGrass, TileForest, TileSand, TileMarsh, TileAsh, TilePath:
		return true
	}
	return false
}

// DigSpot is the tile a player at (x, y) facing dir would till with a hoe,
// and whether it can be tilled: soft ground with nothing on it. The server
// uses it to dig and the client to mark the spot.
func DigSpot(x, y float64, dir Dir, tileAt func(tx, ty int) Tile, occupied func(tx, ty int) bool) (tx, ty int, ok bool) {
	dx, dy := dir.Delta()
	tx, ty = int(math.Floor(x))+dx, int(math.Floor(y))+dy
	return tx, ty, tileAt(tx, ty).Soft() && !occupied(tx, ty)
}

// GrowthStage is how far along a growing crop is, from 0 (just planted) to
// CropStages-1. It is 0 for anything else, ripe crops included.
func (e *Entity) GrowthStage(tick uint64) int {
	if e.Kind != EntityCrop || e.Ready {
		return 0
	}
	grow := CropKind(e.Var).growTicks()
	left := min(grow, int(e.RegrowAt-min(e.RegrowAt, tick)))
	return min(CropStages-1, (grow-left)*CropStages/grow)
}

// dig tills the tile p faces when p holds a hoe, and reports whether it did.
// A better hoe turns up sweet potatoes more often.
func (w *World) dig(p *Player) bool {
	class, tier := p.Held().ToolKind()
	if class != ToolHoe {
		return false
	}
	tx, ty, ok := DigSpot(p.X, p.Y, p.Dir, w.TileAt, func(tx, ty int) bool { return w.entityAt(tx, ty) != nil })
	if !ok {
		return false
	}
	w.till(tx, ty)
	var found ItemKind
	switch r := w.random().Float64(); {
	case r < DigWormChance:
		found = ItemWorm
	case r < DigWormChance+DigPotatoChance*(1+0.5*float64(tier)):
		found = ItemPotato
	}
	if found != ItemNone && p.Inv.Add(found, 1) == 0 {
		p.notify(NoticeBagFull, uint8(found))
	}
	return true
}

// till turns (tx, ty) into an empty bed of soil.
func (w *World) till(tx, ty int) *Entity {
	e := w.addEntity(EntityFarmland, tx, ty)
	e.RegrowAt = w.tick + FarmlandDecayTicks
	return e
}
