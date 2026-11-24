package sim

import "math"

// The body left behind. At the harder difficulties a player who dies comes
// back with nothing: everything they carried stays with their body where
// they fell, and they must go back for it. Action at the body takes back all
// that fits; the body is gone once it is empty. It does not rot, and anyone
// can loot it. At the basic difficulty a player keeps everything instead.

// graveSearch is how far from where a player fell their body may come to
// rest: on the nearest free ground, should they die in the water or on a
// spot something already stands on.
const graveSearch = 10

// A body is drawn as the player who fell, in their look and colour (packed
// into the entity's Fuel; see GraveLook). Should they die again before
// fetching their things, their older bodies are bones by then: Var is
// GraveBones. Bones are looted the same way.

// GraveBones is the Var of a body that has gone to bones.
const GraveBones = 1

// packGraveLook packs l and the colour seed col into a body's Fuel.
func packGraveLook(l Look, col int) int {
	f := 0
	if l.Female {
		f = 1
	}
	return f | int(l.Hair&7)<<1 | int(l.HairColor&7)<<4 | int(l.Skin&7)<<7 | (col&15)<<10
}

// GraveLook unpacks the look of the player whose body has fuel, and the seed
// of their colour (the client picks the player colour from it as it does
// from a player's id).
func GraveLook(fuel int32) (Look, int) {
	f := int(fuel)
	l := Look{Female: f&1 == 1, Hair: HairStyle(f >> 1 & 7), HairColor: uint8(f >> 4 & 7), Skin: uint8(f >> 7 & 7)}
	return l.Sanitized(), f >> 10 & 15
}

// leaveBody lays p's things down by where p fell and empties p's hands. It
// reports whether it found ground to leave them on; if not, p keeps them.
// Any body p left before and never came back for has gone to bones.
func (w *World) leaveBody(p *Player) bool {
	for id, owner := range w.graveOwner {
		if e := w.entities[id]; owner == p.Name && e != nil && e.Kind == EntityGrave {
			e.Var = GraveBones
		}
	}
	if p.Inv == (Inventory{}) {
		return true
	}
	cx, cy := int(math.Floor(p.X)), int(math.Floor(p.Y))
	bx, by, best := 0, 0, math.Inf(1)
	for ty := cy - graveSearch; ty <= cy+graveSearch; ty++ {
		for tx := cx - graveSearch; tx <= cx+graveSearch; tx++ {
			if !w.canPlace(tx, ty) || w.Underground(ty) != w.Underground(cy) {
				continue
			}
			if d := math.Hypot(float64(tx-cx), float64(ty-cy)); d < best {
				bx, by, best = tx, ty, d
			}
		}
	}
	if math.IsInf(best, 1) {
		return false
	}
	e := w.addEntity(EntityGrave, bx, by)
	e.Fuel = packGraveLook(p.Look, int(p.ID))
	inv := p.Inv
	w.chests[e.ID] = &inv
	if w.graveOwner == nil {
		w.graveOwner = make(map[EntityID]string)
	}
	w.graveOwner[e.ID] = p.Name
	p.Inv = Inventory{}
	p.notify(NoticeBodyLeft, 0)
	return true
}

// lootBody takes back from body e all that fits in p's bag; the body is
// gone once it is empty.
func (w *World) lootBody(p *Player, e *Entity) bool {
	inv := w.chests[e.ID]
	if inv == nil {
		w.removeEntity(e)
		return true
	}
	took := false
	for i := range inv {
		s := inv[i]
		if s.Empty() {
			continue
		}
		if moved := p.Inv.Add(s.Item, s.Count); moved > 0 {
			inv.TakeSlot(i, moved)
			took = true
		}
	}
	if *inv == (Inventory{}) {
		w.removeEntity(e)
		p.notify(NoticeBodyLooted, 0)
		return true
	}
	if !took {
		p.notify(NoticeBagFull, 0)
		return false
	}
	return true
}

// BodyItems is how many stacks body id still holds.
func (w *World) BodyItems(id EntityID) int {
	inv := w.chests[id]
	if inv == nil {
		return 0
	}
	n := 0
	for _, s := range inv {
		if !s.Empty() {
			n++
		}
	}
	return n
}
