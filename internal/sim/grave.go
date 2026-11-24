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

// leaveBody lays p's things down by where p fell and empties p's hands. It
// reports whether it found ground to leave them on; if not, p keeps them.
func (w *World) leaveBody(p *Player) bool {
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
	inv := p.Inv
	w.chests[e.ID] = &inv
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
