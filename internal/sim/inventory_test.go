package sim

import "testing"

func TestInventoryAddStacksThenFillsHotbarFirst(t *testing.T) {
	var inv Inventory
	if got := inv.Add(ItemWood, MaxStack+5); got != MaxStack+5 {
		t.Fatalf("added %d, want %d", got, MaxStack+5)
	}
	if inv[0] != (Slot{ItemWood, MaxStack}) || inv[1] != (Slot{ItemWood, 5}) {
		t.Fatalf("slots 0,1 = %v %v, want a full stack then 5", inv[0], inv[1])
	}
	inv.Add(ItemStone, 3)
	inv.Add(ItemWood, 2) // tops up the stack of 5, no new slot
	if inv[1].Count != 7 || inv[2] != (Slot{ItemStone, 3}) || !inv[3].Empty() {
		t.Fatalf("after more wood: %v", inv[:4])
	}
	if inv.Count(ItemWood) != MaxStack+7 {
		t.Fatalf("count = %d", inv.Count(ItemWood))
	}
}

func TestInventoryRoomAndFull(t *testing.T) {
	var inv Inventory
	for i := range inv {
		inv[i] = Slot{ItemStone, MaxStack}
	}
	inv[NumSlots-1] = Slot{ItemWood, MaxStack - 3}
	if r := inv.Room(ItemWood); r != 3 {
		t.Fatalf("room for wood = %d, want 3", r)
	}
	if r := inv.Room(ItemBerry); r != 0 {
		t.Fatalf("room for berries = %d, want 0", r)
	}
	if got := inv.Add(ItemWood, 10); got != 3 {
		t.Fatalf("added %d, want 3", got)
	}
}

func TestInventoryTakeEmptiesBagBeforeHotbar(t *testing.T) {
	var inv Inventory
	inv[0] = Slot{ItemWood, 4}
	inv[HotbarSlots+2] = Slot{ItemWood, 3}
	if got := inv.Take(ItemWood, 5); got != 5 {
		t.Fatalf("took %d", got)
	}
	if !inv[HotbarSlots+2].Empty() || inv[0].Count != 2 {
		t.Fatalf("after take: hotbar %v, bag %v", inv[0], inv[HotbarSlots+2])
	}
	if inv[HotbarSlots+2] != (Slot{}) {
		t.Fatalf("emptied slot not cleared: %v", inv[HotbarSlots+2])
	}
}

func TestInventoryMoveMergesOrSwaps(t *testing.T) {
	var inv Inventory
	inv[0] = Slot{ItemWood, MaxStack - 2}
	inv[1] = Slot{ItemWood, 5}
	inv[2] = Slot{ItemStone, 1}

	if !inv.Move(1, 0) || inv[0].Count != MaxStack || inv[1].Count != 3 {
		t.Fatalf("merge: %v", inv[:2])
	}
	if !inv.Move(2, 1) || inv[1] != (Slot{ItemStone, 1}) || inv[2] != (Slot{ItemWood, 3}) {
		t.Fatalf("swap: %v", inv[:3])
	}
	if !inv.Move(2, 9) || !inv[2].Empty() || inv[9] != (Slot{ItemWood, 3}) {
		t.Fatalf("move to empty: %v", inv)
	}
	if inv.Move(5, 6) || inv.Move(0, 0) || inv.Move(-1, 0) || inv.Move(0, NumSlots) {
		t.Fatal("moving nothing, onto itself or out of range must do nothing")
	}
	full := inv
	if inv.Move(9, 0) { // the stack at 0 is full
		// Same kind but no room: nothing changes.
		t.Fatalf("merge onto a full stack changed %v to %v", full, inv)
	}
}

func TestInventoryQuickMove(t *testing.T) {
	var inv Inventory
	inv[0] = Slot{ItemBerry, 6}
	inv[HotbarSlots+4] = Slot{ItemBerry, MaxStack - 2}
	if !inv.QuickMove(0) {
		t.Fatal("quick move did nothing")
	}
	// Tops up the bag's stack, then the rest goes to the first empty bag slot.
	if inv[HotbarSlots+4].Count != MaxStack || inv[HotbarSlots] != (Slot{ItemBerry, 4}) || !inv[0].Empty() {
		t.Fatalf("after quick move to bag: %v", inv)
	}
	if !inv.QuickMove(HotbarSlots) || inv[0] != (Slot{ItemBerry, 4}) {
		t.Fatalf("quick move back to the hotbar: %v", inv[:HotbarSlots])
	}
}

func TestInventoryWithAndCounts(t *testing.T) {
	inv := InventoryWith(Ingredient{ItemWood, 3}, Ingredient{ItemBerry, 2}, Ingredient{ItemWood, 1})
	c := inv.Counts()
	if c[ItemWood] != 4 || c[ItemBerry] != 2 || inv[0] != (Slot{ItemWood, 4}) {
		t.Fatalf("counts %v, slots %v", c, inv[:3])
	}
	if !inv.HasAll([]Ingredient{{ItemWood, 4}, {ItemBerry, 1}}) || inv.HasAll([]Ingredient{{ItemWood, 5}}) {
		t.Fatal("HasAll wrong")
	}
	inv.TakeAll([]Ingredient{{ItemWood, 4}, {ItemBerry, 1}})
	if c := inv.Counts(); c[ItemWood] != 0 || c[ItemBerry] != 1 {
		t.Fatalf("after TakeAll: %v", c)
	}
}

func TestInventorySanitize(t *testing.T) {
	var inv Inventory
	inv[0] = Slot{ItemWood, 5000}
	inv[1] = Slot{ItemNone, 3}
	inv[2] = Slot{ItemStone, -4}
	inv[3] = Slot{NumItems + 3, 1}
	inv.sanitize()
	if inv[0].Count != ItemWood.MaxCount() || inv[1] != (Slot{}) || inv[2] != (Slot{}) || inv[3] != (Slot{}) {
		t.Fatalf("sanitized: %v", inv[:4])
	}
}
