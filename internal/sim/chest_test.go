package sim

import "testing"

// chestBeside builds a chest to the right of a new player on a flat field.
func chestBeside(t *testing.T) (*World, *Player, *Entity) {
	t.Helper()
	w := flatWorld()
	p := w.players[w.AddPlayer("a")] // on (16, 16)
	p.Dir = DirRight
	p.Inv.Add(ItemWood, 6)
	w.Apply(p.ID, Command{Kind: CmdCraft, Recipe: RecipeChest})
	e := w.entityAt(17, 16)
	if e == nil || e.Kind != EntityChest || p.Inv.Count(ItemWood) != 0 {
		t.Fatalf("after building: %+v at (17, 16), %d wood left", e, p.Inv.Count(ItemWood))
	}
	return w, p, e
}

func open(w *World, p *Player) {
	w.SetInput(p.ID, Input{Action: true})
	w.Step()
	w.SetInput(p.ID, Input{})
}

func TestAChestStoresWhatIsPutInIt(t *testing.T) {
	w, p, chest := chestBeside(t)
	if inv, ok := w.ChestContents(chest.ID); !ok || inv != (Inventory{}) {
		t.Fatalf("a new chest holds %v (ok %v), want nothing", inv, ok)
	}

	// Commands before it is open do nothing.
	p.Inv.Add(ItemBerry, 5)
	w.Apply(p.ID, Command{Kind: CmdChestPut, Slot: 0})
	if p.Inv.Count(ItemBerry) != 5 {
		t.Fatal("put into a chest that was not open")
	}

	open(w, p)
	if p.OpenChest() != chest.ID || p.Acting(w.Tick()) {
		t.Fatalf("after Space: open chest %d (want %d), swinging %v", p.OpenChest(), chest.ID, p.Acting(w.Tick()))
	}
	w.Apply(p.ID, Command{Kind: CmdChestPut, Slot: 0})
	inv, _ := w.ChestContents(chest.ID)
	if p.Inv.Count(ItemBerry) != 0 || inv.Count(ItemBerry) != 5 {
		t.Fatalf("after putting: %d berries on me, %d in the chest", p.Inv.Count(ItemBerry), inv.Count(ItemBerry))
	}
	w.Apply(p.ID, Command{Kind: CmdChestTake, Slot: 0})
	inv, _ = w.ChestContents(chest.ID)
	if p.Inv.Count(ItemBerry) != 5 || inv.Count(ItemBerry) != 0 {
		t.Fatalf("after taking: %d berries on me, %d in the chest", p.Inv.Count(ItemBerry), inv.Count(ItemBerry))
	}

	// Several tools of a kind fit in a chest.
	p.Inv.Add(ItemPickaxe, 1)
	w.Apply(p.ID, Command{Kind: CmdChestPut, Slot: uint8(slotOf(p, ItemPickaxe))})
	p.Inv.Add(ItemPickaxe, 1)
	w.Apply(p.ID, Command{Kind: CmdChestPut, Slot: uint8(slotOf(p, ItemPickaxe))})
	if inv, _ = w.ChestContents(chest.ID); inv.Count(ItemPickaxe) != 2 {
		t.Errorf("chest holds %d pickaxes, want 2", inv.Count(ItemPickaxe))
	}
}

func slotOf(p *Player, k ItemKind) int {
	for i, s := range p.Inv {
		if s.Item == k && !s.Empty() {
			return i
		}
	}
	return -1
}

func TestWalkingAwayClosesTheChest(t *testing.T) {
	w, p, chest := chestBeside(t)
	open(w, p)
	if p.OpenChest() != chest.ID {
		t.Fatal("did not open")
	}
	p.X = float64(chest.TX) + 0.5 + ChestReach + 0.5 // just out of reach
	w.Step()
	if p.OpenChest() != 0 {
		t.Fatal("still open after walking off")
	}
	open(w, p) // too far to reach it now
	if p.OpenChest() != 0 {
		t.Fatal("opened from afar")
	}
}

func TestOnlyAnEmptyChestComesDown(t *testing.T) {
	w, p, chest := chestBeside(t)
	open(w, p)
	p.Inv.Add(ItemStone, 1)
	w.Apply(p.ID, Command{Kind: CmdChestPut, Slot: uint8(slotOf(p, ItemStone))})
	w.Apply(p.ID, Command{Kind: CmdChestTakeDown})
	if _, ok := w.EntityAt(17, 16); !ok || p.Notice != NoticeChestNotEmpty {
		t.Fatalf("taking down a full chest: still there %v, notice %v", ok, p.Notice)
	}
	w.Apply(p.ID, Command{Kind: CmdChestTake, Slot: 0})
	w.Apply(p.ID, Command{Kind: CmdChestTakeDown})
	if _, ok := w.EntityAt(17, 16); ok || p.Inv.Count(ItemWood) != ChestRefund || p.OpenChest() != 0 {
		t.Fatalf("taking down the empty chest: still there %v, wood %d, open %d", ok, p.Inv.Count(ItemWood), p.OpenChest())
	}
	if _, ok := w.ChestContents(chest.ID); ok {
		t.Error("the chest's contents outlived it")
	}
}

func TestChestsKeepTheirContentsInASave(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	sx, sy := int(w.spawnX), int(w.spawnY)
	full, empty := w.addEntity(EntityChest, sx+1, sy), w.addEntity(EntityChest, sx-1, sy)
	w.chests[full.ID].Add(ItemIronBar, 7)
	w.chests[full.ID].Add(ItemDiamondSword, 1)
	want, _ := w.ChestContents(full.ID)

	d := w.Save()
	d.Chests = d.Chests[:1] // as if the empty one had never been saved
	if d.Chests[0].ID != full.ID {
		t.Fatalf("saved chests %+v, want the full one first", d.Chests)
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := w2.ChestContents(full.ID); !ok || got != want {
		t.Errorf("after load the chest holds %v, want %v", got, want)
	}
	if got, ok := w2.ChestContents(empty.ID); !ok || got != (Inventory{}) {
		t.Errorf("the unsaved chest holds %v (ok %v), want an empty one", got, ok)
	}
}
