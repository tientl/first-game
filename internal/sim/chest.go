package sim

import (
	"math"
	"slices"
)

// Storage chests. A chest is built like a wall and holds as many slots as a
// player's whole inventory. Action beside it opens it; while it is open the
// player's snapshot carries what is inside, and the player moves stacks in
// and out with commands. Walking away closes it. An empty chest can be taken
// down for some of its wood back.

// Chest tuning.
const (
	ChestReach  = InteractReach + 0.75 // how far a player may step from an open chest
	ChestRefund = 4                    // wood back from a chest taken down
)

// ChestSave is a chest's contents in a save.
type ChestSave struct {
	ID    EntityID
	Slots Inventory
}

// OpenChest is the chest p has open, 0 when none.
func (p *Player) OpenChest() EntityID { return p.chest }

// ChestContents is what chest id holds; ok is false when there is no such
// chest.
func (w *World) ChestContents(id EntityID) (Inventory, bool) {
	inv := w.chests[id]
	if inv == nil {
		return Inventory{}, false
	}
	return *inv, true
}

// openChest opens chest e for p. It is no swing, so it reports false.
func (w *World) openChest(p *Player, e *Entity) bool {
	if w.chests[e.ID] == nil {
		w.chests[e.ID] = &Inventory{}
	}
	p.chest = e.ID
	return false
}

// keepChest closes p's chest when it is gone or p has walked off from it.
func (w *World) keepChest(p *Player) {
	if p.chest == 0 {
		return
	}
	e := w.entities[p.chest]
	if p.Dead || p.Asleep || e == nil || e.Kind != EntityChest ||
		math.Hypot(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y) > ChestReach {
		p.chest = 0
	}
}

// chestCommand moves a stack between p's inventory and p's open chest, or
// closes or takes down the chest.
func (w *World) chestCommand(p *Player, cmd Command) {
	w.keepChest(p)
	chest := w.chests[p.chest]
	if p.chest == 0 || chest == nil {
		return // closed meanwhile: the click is stale
	}
	i := int(cmd.Slot)
	switch cmd.Kind {
	case CmdChestPut:
		if ValidSlot(i) && !p.Inv[i].Empty() {
			s := p.Inv[i]
			if moved := chest.Add(s.Item, s.Count); moved > 0 {
				p.Inv.TakeSlot(i, moved)
			} else {
				p.notify(NoticeChestFull, uint8(s.Item))
			}
		}
	case CmdChestTake:
		if ValidSlot(i) && !chest[i].Empty() {
			s := chest[i]
			if moved := p.Inv.Add(s.Item, s.Count); moved > 0 {
				chest.TakeSlot(i, moved)
			} else {
				p.notify(NoticeBagFull, uint8(s.Item))
			}
		}
	case CmdChestClose:
		p.chest = 0
	case CmdChestTakeDown:
		if *chest != (Inventory{}) {
			p.notify(NoticeChestNotEmpty, 0)
			return
		}
		if p.Inv.Room(ItemWood) < ChestRefund {
			p.notify(NoticeBagFull, uint8(ItemWood))
			return
		}
		p.Inv.Add(ItemWood, ChestRefund)
		w.removeEntity(w.entities[p.chest]) // everyone's view of it closes next tick
		p.chest = 0
	}
}

// saveChests lists every chest's contents, by id.
func (w *World) saveChests() []ChestSave {
	var out []ChestSave
	for id, inv := range w.chests {
		out = append(out, ChestSave{ID: id, Slots: *inv})
	}
	slices.SortFunc(out, func(a, b ChestSave) int { return int(a.ID) - int(b.ID) })
	return out
}

// loadChests gives every chest in the world its saved contents, or an empty
// inventory. Contents of chests that are not in the world are dropped.
func (w *World) loadChests(saved []ChestSave) {
	for _, cs := range saved {
		if e := w.entities[cs.ID]; e != nil && (e.Kind == EntityChest || e.Kind == EntityGrave) {
			inv := cs.Slots
			inv.sanitize()
			w.chests[cs.ID] = &inv
		}
	}
	for id, e := range w.entities {
		if e.Kind == EntityChest && w.chests[id] == nil {
			w.chests[id] = &Inventory{}
		}
	}
}
