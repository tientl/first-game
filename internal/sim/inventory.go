package sim

// The inventory is a row of slots. The first HotbarSlots are the hotbar the
// player picks from with the number keys; the rest are the bag. Each slot
// holds one kind of item, up to that kind's stack size (ItemKind.MaxCount).

const (
	HotbarSlots = 10
	BagSlots    = 20
	NumSlots    = HotbarSlots + BagSlots
)

// Slot is one inventory cell: Count of Item. A slot with Count 0 is empty.
type Slot struct {
	Item  ItemKind
	Count int
}

// Empty reports whether the slot holds nothing.
func (s Slot) Empty() bool { return s.Count <= 0 || !s.Item.Valid() }

// Inventory is every slot a player has, hotbar first.
type Inventory [NumSlots]Slot

// InventoryWith returns an inventory holding the given items, placed the way
// picking them up one after another would.
func InventoryWith(items ...Ingredient) Inventory {
	var inv Inventory
	for _, it := range items {
		inv.Add(it.Item, it.Count)
	}
	return inv
}

// ValidSlot reports whether i is a slot index.
func ValidSlot(i int) bool { return i >= 0 && i < NumSlots }

// IsHotbar reports whether slot i is on the hotbar.
func IsHotbar(i int) bool { return i >= 0 && i < HotbarSlots }

// Count returns how many of k the inventory holds across all slots.
func (inv *Inventory) Count(k ItemKind) int {
	n := 0
	for _, s := range inv {
		if !s.Empty() && s.Item == k {
			n += s.Count
		}
	}
	return n
}

// Counts returns how many of each item kind the inventory holds.
func (inv *Inventory) Counts() [NumItems]int {
	var c [NumItems]int
	for _, s := range inv {
		if !s.Empty() {
			c[s.Item] += s.Count
		}
	}
	return c
}

// Room returns how many more of k fit: the space left on its stacks plus
// every empty slot.
func (inv *Inventory) Room(k ItemKind) int {
	if !k.Valid() {
		return 0
	}
	n, max := 0, k.MaxCount()
	for _, s := range inv {
		switch {
		case s.Empty():
			n += max
		case s.Item == k:
			n += max - min(max, s.Count)
		}
	}
	return n
}

// Add puts up to n of k into the inventory and returns how many went in. It
// tops up stacks of k first, then fills empty slots, hotbar first.
func (inv *Inventory) Add(k ItemKind, n int) int {
	if !k.Valid() || n <= 0 {
		return 0
	}
	max, added := k.MaxCount(), 0
	for i := range inv {
		if s := &inv[i]; !s.Empty() && s.Item == k && s.Count < max {
			d := min(n-added, max-s.Count)
			s.Count += d
			added += d
		}
	}
	for i := range inv {
		if added == n {
			break
		}
		if s := &inv[i]; s.Empty() {
			d := min(n-added, max)
			*s = Slot{Item: k, Count: d}
			added += d
		}
	}
	return added
}

// Take removes up to n of k and returns how many were removed. It empties the
// bag's stacks before the hotbar's, so what the player keeps at hand stays.
func (inv *Inventory) Take(k ItemKind, n int) int {
	taken := 0
	for i := len(inv) - 1; i >= 0 && taken < n; i-- {
		if s := &inv[i]; !s.Empty() && s.Item == k {
			d := min(n-taken, s.Count)
			s.Count -= d
			taken += d
			if s.Count == 0 {
				*s = Slot{}
			}
		}
	}
	return taken
}

// TakeSlot removes up to n items from slot i and returns how many it removed.
func (inv *Inventory) TakeSlot(i, n int) int {
	if !ValidSlot(i) || inv[i].Empty() || n <= 0 {
		return 0
	}
	s := &inv[i]
	d := min(n, s.Count)
	s.Count -= d
	if s.Count == 0 {
		*s = Slot{}
	}
	return d
}

// HasAll reports whether the inventory holds every ingredient.
func (inv *Inventory) HasAll(in []Ingredient) bool {
	for _, ing := range in {
		if !ing.Item.Valid() || inv.Count(ing.Item) < ing.Count {
			return false
		}
	}
	return true
}

// TakeAll removes every ingredient. The caller must check HasAll first.
func (inv *Inventory) TakeAll(in []Ingredient) {
	for _, ing := range in {
		inv.Take(ing.Item, ing.Count)
	}
}

// Move moves the stack in slot from onto slot to: into it if it is empty,
// onto it if it holds the same kind (as much as fits), else the two swap.
// It reports whether anything changed.
func (inv *Inventory) Move(from, to int) bool {
	if !ValidSlot(from) || !ValidSlot(to) || from == to || inv[from].Empty() {
		return false
	}
	a, b := &inv[from], &inv[to]
	if !b.Empty() && b.Item == a.Item {
		d := min(a.Count, a.Item.MaxCount()-b.Count)
		if d <= 0 {
			return false
		}
		b.Count += d
		a.Count -= d
		if a.Count == 0 {
			*a = Slot{}
		}
		return true
	}
	*a, *b = *b, *a
	return true
}

// QuickMove sends the stack in slot i across: from the hotbar into the bag,
// or from the bag onto the hotbar, topping up stacks of the same kind first.
// What does not fit stays where it was. It reports whether anything moved.
func (inv *Inventory) QuickMove(i int) bool {
	if !ValidSlot(i) || inv[i].Empty() {
		return false
	}
	lo, hi := HotbarSlots, NumSlots
	if !IsHotbar(i) {
		lo, hi = 0, HotbarSlots
	}
	s := &inv[i]
	max, before := s.Item.MaxCount(), s.Count
	for j := lo; j < hi && s.Count > 0; j++ {
		if t := &inv[j]; !t.Empty() && t.Item == s.Item && t.Count < max {
			d := min(s.Count, max-t.Count)
			t.Count += d
			s.Count -= d
		}
	}
	for j := lo; j < hi && s.Count > 0; j++ {
		if t := &inv[j]; t.Empty() {
			*t, *s = *s, Slot{}
		}
	}
	if s.Count == 0 {
		*s = Slot{}
	}
	return s.Count != before
}

// sanitize empties slots that hold nothing real and caps oversized stacks,
// for inventories that come from outside (a save file).
func (inv *Inventory) sanitize() {
	for i := range inv {
		s := &inv[i]
		if s.Empty() {
			*s = Slot{}
			continue
		}
		s.Count = min(s.Count, s.Item.MaxCount())
	}
}

// HasFuel reports whether there is anything to put on a campfire.
func (inv *Inventory) HasFuel() bool {
	return inv.Count(ItemWood) > 0 || inv.Count(ItemCoal) > 0
}
