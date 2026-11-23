package sim

import (
	"encoding/json"
	"testing"
)

func TestSelectingAHotbarSlot(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Inv[3] = Slot{ItemAxe, 1}

	w.Apply(id, Command{Kind: CmdSelect, Slot: 3})
	if p := mustPlayer(t, w, id); p.Selected != 3 || p.Held() != ItemAxe {
		t.Fatalf("selected %d holding %v, want slot 3 and the axe", p.Selected, p.Held())
	}
	w.Apply(id, Command{Kind: CmdSelect, Slot: HotbarSlots}) // a bag slot cannot be held
	if p := mustPlayer(t, w, id); p.Selected != 3 {
		t.Fatalf("selected %d after asking for a bag slot, want 3 kept", p.Selected)
	}
	w.Apply(id, Command{Kind: CmdSelect, Slot: 0})
	if p := mustPlayer(t, w, id); p.Held() != ItemNone {
		t.Fatalf("holding %v from an empty slot", p.Held())
	}
}

func TestMovingStacks(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Inv[0] = Slot{ItemWood, 5}

	w.Apply(id, Command{Kind: CmdMoveSlot, Slot: 0, To: HotbarSlots + 2})
	if p := mustPlayer(t, w, id); !p.Inv[0].Empty() || p.Inv[HotbarSlots+2] != (Slot{ItemWood, 5}) {
		t.Fatalf("after moving into the bag: %v", p.Inv)
	}
	w.Apply(id, Command{Kind: CmdQuickMove, Slot: HotbarSlots + 2})
	if p := mustPlayer(t, w, id); p.Inv[0] != (Slot{ItemWood, 5}) {
		t.Fatalf("after a quick move to the hotbar: %v", p.Inv)
	}

	before := w.players[id].Inv
	w.Apply(id, Command{Kind: CmdMoveSlot, Slot: 0, To: 250}) // garbage from the network
	w.players[id].Dead = true
	w.Apply(id, Command{Kind: CmdMoveSlot, Slot: 0, To: 1})
	w.Apply(id, Command{Kind: CmdQuickMove, Slot: 0})
	if p := mustPlayer(t, w, id); p.Inv != before {
		t.Fatalf("a bad slot or a dead player changed the inventory to %v", p.Inv)
	}
}

// fullBagFor fills every slot with wood except the first ones, which hold
// r's ingredients, each with extra more than the recipe needs.
func fullBagFor(r Recipe, extra int) Inventory {
	var inv Inventory
	for i := range inv {
		inv[i] = Slot{ItemWood, MaxStack}
	}
	for i, in := range r.Inputs {
		inv[i] = Slot{in.Item, in.Count + extra}
	}
	return inv
}

func TestCraftingIntoAFullBag(t *testing.T) {
	r, _ := RecipeByID(RecipeAxe)

	// The ingredients use up whole slots: the axe goes into one of them.
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Inv = fullBagFor(r, 0)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeAxe})
	if p := mustPlayer(t, w, id); p.Notice != NoticeCrafted || p.Inv.Count(ItemAxe) != 1 {
		t.Fatalf("notice %v, axes %d: want the axe in a freed slot", p.Notice, p.Inv.Count(ItemAxe))
	}

	// Nothing frees up: refused, and nothing is used.
	w = flatWorld()
	id = w.AddPlayer("a")
	full := fullBagFor(r, 1)
	w.players[id].Inv = full
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeAxe})
	if p := mustPlayer(t, w, id); p.Notice != NoticeBagFull || p.Inv != full {
		t.Fatalf("notice %v: want bag full and the ingredients kept", p.Notice)
	}
}

func TestVersion3InventoryBecomesSlots(t *testing.T) {
	// A version 3 file kept a count per item kind.
	var ps PlayerSave
	if err := json.Unmarshal([]byte(`{"Name":"Cũ","Health":80,"Hunger":50,"Inv":[0,4,0,0,2,0,1]}`), &ps); err != nil {
		t.Fatalf("an old player record no longer decodes: %v", err)
	}

	w, err := Generate(GenConfig{Seed: 5, Width: 48, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	d.Version = 3
	ps.X, ps.Y = w.spawnX, w.spawnY
	d.Players = []PlayerSave{ps}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	p := mustPlayer(t, w2, w2.AddPlayer("Cũ"))
	// The pickaxe comes from the version 5 upgrade: boulders were mined by hand.
	want := InventoryWith(Ingredient{ItemWood, 4}, Ingredient{ItemBerry, 2}, Ingredient{ItemAxe, 1}, Ingredient{ItemPickaxe, 1})
	if p.Inv != want {
		t.Fatalf("upgraded inventory = %v, want %v", p.Inv, want)
	}
	if got := w2.Save().Players[0]; got.Inv != nil || got.Slots != want {
		t.Fatalf("saved again: legacy %v, slots %v; want only slots", got.Inv, got.Slots)
	}
}
