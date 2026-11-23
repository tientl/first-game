package sim

import "testing"

func TestCookingNeedsABurningFire(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Inv = InventoryWith(Ingredient{Item: ItemMeat, Count: 2}, Ingredient{Item: ItemFish, Count: 1})

	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCookMeat})
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeNeedFire || RecipeID(p.NoticeArg) != RecipeCookMeat {
		t.Fatalf("notice %v/%d, want a request to stand by a fire", p.Notice, p.NoticeArg)
	}
	if p.Inv.Count(ItemMeat) != 2 || p.Inv.Count(ItemCookedMeat) != 0 {
		t.Fatal("cooked without a fire")
	}

	fire := w.addEntity(EntityCampfire, 18, 16) // in its light, 1.5 tiles away
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCookMeat})
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCookFish})
	p = mustPlayer(t, w, id)
	if p.Inv.Count(ItemMeat) != 1 || p.Inv.Count(ItemCookedMeat) != 1 ||
		p.Inv.Count(ItemFish) != 0 || p.Inv.Count(ItemCookedFish) != 1 {
		t.Fatalf("meat %d→%d, fish %d→%d: want one of each cooked",
			p.Inv.Count(ItemMeat), p.Inv.Count(ItemCookedMeat), p.Inv.Count(ItemFish), p.Inv.Count(ItemCookedFish))
	}
	if p.Notice != NoticeCrafted || RecipeID(p.NoticeArg) != RecipeCookFish {
		t.Fatalf("notice %v/%d, want the fish cooked", p.Notice, p.NoticeArg)
	}

	w.removeEntity(fire) // burnt out
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCookMeat})
	if p := mustPlayer(t, w, id); p.Notice != NoticeNeedFire || p.Inv.Count(ItemMeat) != 1 {
		t.Fatal("cooked on a fire that went out")
	}

	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCookFish})
	if p := mustPlayer(t, w, id); p.Notice != NoticeNotEnough {
		t.Fatalf("notice %v: with no raw fish the missing fish comes first", p.Notice)
	}
}

func TestCookedFoodBeatsRaw(t *testing.T) {
	for _, pair := range [][2]ItemKind{{ItemMeat, ItemCookedMeat}, {ItemFish, ItemCookedFish}} {
		raw, cooked := pair[0].Def(), pair[1].Def()
		if !pair[0].Edible() || cooked.Hunger <= raw.Hunger || cooked.Health <= raw.Health || cooked.Health <= 0 {
			t.Errorf("%s (%+v) should be edible and worse than %s (%+v)", raw.Name, raw, cooked.Name, cooked)
		}
	}
}

func TestRawFoodHurtsButNeverKills(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = InventoryWith(Ingredient{Item: ItemMeat, Count: 3})
	p.Health, p.Hunger = 50, 20

	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Health != 50+ItemMeat.Def().Health || p.Hunger != 20+ItemMeat.Def().Hunger {
		t.Fatalf("health %.1f, hunger %.1f after raw meat", p.Health, p.Hunger)
	}

	p.Health = 1.5
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Health != 1 || p.Dead {
		t.Fatalf("health %.1f, dead %v: raw meat must not take a player below 1", p.Health, p.Dead)
	}
	p.Health = 0.5 // already lower: a meal does not heal it either
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Health != 0.5 {
		t.Fatalf("health %.2f, want raw meat to leave it alone", p.Health)
	}
}
