package sim

import (
	"encoding/json"
	"testing"
)

func TestTheBagHasThirtySlots(t *testing.T) {
	if BagSlots != 30 || NumSlots != HotbarSlots+30 {
		t.Fatalf("bag %d slots, %d in all; want 30 and %d", BagSlots, NumSlots, HotbarSlots+30)
	}
	var inv Inventory
	if got := inv.Add(ItemWood, NumSlots*ItemWood.MaxCount()+5); got != NumSlots*ItemWood.MaxCount() {
		t.Errorf("an empty inventory took %d wood, want %d", got, NumSlots*ItemWood.MaxCount())
	}
	if inv[NumSlots-1] != (Slot{ItemWood, ItemWood.MaxCount()}) {
		t.Errorf("the last bag slot holds %+v", inv[NumSlots-1])
	}
}

// A save from before version 14 kept 30 slots per inventory (10 hotbar, 20
// bag) and per chest: it loads with every stack where it was and the new
// slots empty.
func TestAnOldSaveLoadsIntoTheBiggerBag(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	p := w.players[w.AddPlayer("a")]
	p.Inv = Inventory{}
	p.Inv[0] = Slot{ItemAxe, 1}
	p.Inv[29] = Slot{ItemBerry, 7} // the last slot of the old bag
	sx, sy := int(w.spawnX), int(w.spawnY)
	chest := w.addEntity(EntityChest, sx+2, sy)
	w.chests[chest.ID][29] = Slot{ItemStone, 3}

	raw, err := json.Marshal(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	m["Version"] = 13
	for _, ps := range m["Players"].([]any) {
		ps := ps.(map[string]any)
		ps["Slots"] = ps["Slots"].([]any)[:30]
	}
	for _, cs := range m["Chests"].([]any) {
		cs := cs.(map[string]any)
		cs["Slots"] = cs["Slots"].([]any)[:30]
	}
	if raw, err = json.Marshal(m); err != nil {
		t.Fatal(err)
	}
	var d SaveData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	got := w2.players[w2.AddPlayer("a")]
	want := Inventory{}
	want[0], want[29] = Slot{ItemAxe, 1}, Slot{ItemBerry, 7}
	if got.Inv != want {
		t.Errorf("after load the inventory is %v, want %v", got.Inv, want)
	}
	inv, ok := w2.ChestContents(chest.ID)
	if !ok || inv[29] != (Slot{ItemStone, 3}) || inv.Count(ItemStone) != 3 {
		t.Errorf("after load the chest holds %v (ok %v)", inv, ok)
	}
}

// An old chest in a cave opens like a chest with its hoard inside; nothing
// spills on the ground.
func TestAnOldChestOpensWithItsHoardInside(t *testing.T) {
	w := flatWorld()
	p := w.players[w.AddPlayer("a")] // on (16, 16)
	p.Dir = DirRight
	e := w.addEntity(EntityTreasure, 17, 16)
	e.Var = 30
	if !e.Ready {
		t.Fatal("a new old chest is not ready to open")
	}
	loot := treasureLoot(e.ID, int(e.Var))
	open(w, p)
	if p.OpenChest() != e.ID || e.Ready {
		t.Fatalf("after Space: open chest %d (want %d), still shut %v", p.OpenChest(), e.ID, e.Ready)
	}
	if n := len(w.Drops()); n != 0 {
		t.Errorf("%d piles spilled on the ground", n)
	}
	inv, _ := w.ChestContents(e.ID)
	for _, it := range loot {
		if inv.Count(it.Item) != it.Count {
			t.Errorf("the chest holds %d %v, want %d", inv.Count(it.Item), it.Item, it.Count)
		}
	}
	had := p.Inv.Count(loot[0].Item)
	w.Apply(p.ID, Command{Kind: CmdChestTake, Slot: 0})
	if p.Inv.Count(loot[0].Item)-had != loot[0].Count {
		t.Errorf("took %d %v from the chest, want %d", p.Inv.Count(loot[0].Item)-had, loot[0].Item, loot[0].Count)
	}
	w.Apply(p.ID, Command{Kind: CmdChestTakeDown})
	if w.entities[e.ID] == nil {
		t.Fatal("an old chest was taken down")
	}

	// Closed and opened again, the rest is still there; and in a save.
	w.Apply(p.ID, Command{Kind: CmdChestClose})
	for range TicksPerSecond { // the button let go, the swing over
		w.Step()
	}
	open(w, p)
	if p.OpenChest() != e.ID {
		t.Fatal("the old chest does not open again")
	}
	before, _ := w.ChestContents(e.ID)
	saved := w.saveChests()
	delete(w.chests, e.ID)
	w.loadChests(saved)
	if after, ok := w.ChestContents(e.ID); !ok || after != before {
		t.Errorf("after a save the old chest holds %v, want %v", after, before)
	}

	// One opened before version 14 spilled its hoard then: it opens empty.
	old := w.addEntity(EntityTreasure, 16, 17)
	old.Ready = false
	delete(w.chests, old.ID)
	if w.treasureLeft(old) {
		t.Error("an emptied old chest is still worth opening")
	}
	w.Apply(p.ID, Command{Kind: CmdChestClose})
	w.openTreasure(p, old)
	if inv, ok := w.ChestContents(old.ID); p.OpenChest() != old.ID || !ok || inv != (Inventory{}) {
		t.Errorf("an emptied old chest: open %d, holds %v (ok %v)", p.OpenChest(), inv, ok)
	}
}
