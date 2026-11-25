package sim

import "testing"

// swingsToBreak puts e next to a fresh player carrying inv, holds Action and
// returns how many swings took e down, or -1 if it never went.
func swingsToBreak(t *testing.T, kind EntityKind, big bool, inv ...ItemKind) (int, *Player) {
	t.Helper()
	w := flatWorld()
	e := w.addEntity(kind, 17, 16)
	e.Big, e.HP = big, maxHP(kind, big)
	id := w.AddPlayer("a")
	for _, k := range inv {
		w.players[id].Inv.Add(k, 1)
	}
	w.SetInput(id, Input{Action: true})
	for n := 1; n <= 40; n++ {
		w.Step()
		if got, ok := w.EntityAt(17, 16); !ok || !got.Ready { // mined away, or a stump
			return n, w.players[id]
		}
		stepN(w, ActionCooldownTicks-1)
	}
	return -1, w.players[id]
}

func TestOresNeedThePickaxeOneTierBelow(t *testing.T) {
	for _, o := range ores {
		need := o.tier - 1
		if n, p := swingsToBreak(t, o.kind, false); n != -1 || p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ToolItem(ToolPickaxe, need) {
			t.Errorf("%s by hand: %d swings, notice %v/%d; want refused, asking for %s",
				o.kind.Name(), n, p.Notice, p.NoticeArg, ToolItem(ToolPickaxe, need).Def().Name)
		}
		if need > TierStone {
			if n, _ := swingsToBreak(t, o.kind, false, ToolItem(ToolPickaxe, need-1)); n != -1 {
				t.Errorf("%s fell to a %s", o.kind.Name(), ToolItem(ToolPickaxe, need-1).Def().Name)
			}
		}
		n, p := swingsToBreak(t, o.kind, false, ToolItem(ToolPickaxe, need))
		if want := (o.hits*HitPower + ToolPower(need) - 1) / ToolPower(need); n != want {
			t.Errorf("%s with a %s: %d swings, want %d", o.kind.Name(), ToolItem(ToolPickaxe, need).Def().Name, n, want)
		}
		if p.Inv.Count(o.item) != o.yield {
			t.Errorf("%s gave %d %s, want %d", o.kind.Name(), p.Inv.Count(o.item), o.item.Def().Name, o.yield)
		}
	}
}

func TestBetterToolsHitHarder(t *testing.T) {
	prev := 1 << 30
	for tier := TierStone; tier < NumTiers; tier++ {
		n, _ := swingsToBreak(t, EntityTree, true, ToolItem(ToolAxe, tier))
		if want := (maxHP(EntityTree, true) + ToolPower(tier) - 1) / ToolPower(tier); n != want || n > prev {
			t.Errorf("big tree with a %s: %d swings, want %d (and no more than %d)", ToolItem(ToolAxe, tier).Def().Name, n, want, prev)
		}
		prev = n
	}
	if n, _ := swingsToBreak(t, EntityTree, true, ItemAxe); n != BigTreeHits {
		t.Errorf("big tree with a stone axe: %d swings, want BigTreeHits = %d", n, BigTreeHits)
	}
	// The best tool carried is the one used.
	if n, _ := swingsToBreak(t, EntityBoulder, true, ItemPickaxe, ItemDiamondPickaxe, ItemCopperPickaxe); n != (maxHP(EntityBoulder, true)+ToolPower(TierDiamond)-1)/ToolPower(TierDiamond) {
		t.Errorf("with stone, copper and diamond pickaxes the big boulder took %d swings", n)
	}
}

func TestEveryTierHasItsTools(t *testing.T) {
	for c := ToolAxe; c < NumToolClasses; c++ {
		for tier := TierStone; tier < NumTiers; tier++ {
			k := ToolItem(c, tier)
			if gc, gt := k.ToolKind(); !k.Valid() || !k.Def().Tool || gc != c || gt != tier {
				t.Fatalf("tool (%d, %d) is %v, which says it is (%d, %d)", c, tier, k, gc, gt)
			}
			if tier == TierStone {
				continue
			}
			r, ok := RecipeByID(TierToolRecipe(c, tier))
			if !ok || r.Gives != k || r.Inputs[1].Item != TierMaterial(tier) {
				t.Errorf("recipe for %s: %+v", k.Def().Name, r)
			}
		}
	}
	// The block of tool recipes ends right where the recipes after it begin.
	if first := RecipeRoastClam; first != TierToolRecipe(NumToolClasses-1, NumTiers-1)+1 {
		t.Errorf("first recipe after the tools = %d, the last tool recipe is %d", first, TierToolRecipe(NumToolClasses-1, NumTiers-1))
	}
}

func TestAFurnaceSmeltsWhileItBurns(t *testing.T) {
	w := flatWorld()
	furnace := w.addEntity(EntityFurnace, 17, 16)
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv.Add(ItemCopperOre, 2)

	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeSmeltCopper})
	if p.Notice != NoticeNeedFurnace || p.Inv.Count(ItemCopperBar) != 0 {
		t.Fatalf("smelting at a cold furnace: notice %v, bars %d", p.Notice, p.Inv.Count(ItemCopperBar))
	}

	p.Inv.Add(ItemWood, 1)
	w.SetInput(id, Input{Action: true})
	w.Step()
	w.SetInput(id, Input{})
	if furnace.Fuel <= 0 || p.Inv.Count(ItemWood) != 0 {
		t.Fatalf("after stoking: fuel %d, wood left %d", furnace.Fuel, p.Inv.Count(ItemWood))
	}
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeSmeltCopper})
	if p.Inv.Count(ItemCopperBar) != 1 || p.Inv.Count(ItemCopperOre) != 1 {
		t.Fatalf("after smelting: %d bars, %d ore", p.Inv.Count(ItemCopperBar), p.Inv.Count(ItemCopperOre))
	}
	if !w.nearFire(p.X, p.Y) {
		t.Error("a lit furnace does not warm or cook")
	}

	w.RemovePlayer(id)
	stepN(w, FireFuelPerWood)
	if furnace.Fuel != 0 {
		t.Fatalf("fuel %d after burning a whole log", furnace.Fuel)
	}
	if _, ok := w.EntityAt(17, 16); !ok {
		t.Fatal("a cold furnace vanished like a campfire")
	}
}

func TestOresLieInTheirOwnRegions(t *testing.T) {
	for _, seed := range []uint64{1, 7, 42} {
		w, err := Generate(GenConfig{Seed: seed, Width: 256, Height: 256, Style: StyleOpen})
		if err != nil {
			t.Fatal(err)
		}
		count := map[EntityKind]int{}
		for _, e := range w.entities {
			if _, ok := oreOf(e.Kind); ok {
				count[e.Kind]++
				if tile := w.TileAt(e.TX, e.TY); !oreLiesOn(e.Kind, tile) {
					t.Fatalf("seed %d: %s on tile %d", seed, e.Kind.Name(), tile)
				}
			}
		}
		for _, o := range ores {
			if count[o.kind] < 5 {
				t.Errorf("seed %d: only %d %s", seed, count[o.kind], o.kind.Name())
			}
		}
	}
}

func TestOlderSavesCountHPInHitPower(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	d.Version = 5
	var tree EntityID
	for i := range d.Entities {
		e := &d.Entities[i]
		e.HP /= HitPower // version 5 counted whole hits
		if e.Kind == EntityTree && e.Big && tree == 0 {
			tree, e.HP = e.ID, 1 // one hit from falling
		}
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	for id, e := range w.entities {
		want := e.HP
		if id == tree {
			want = HitPower
		}
		if got := w2.entities[id].HP; got != want {
			t.Fatalf("%s %d: hp %d after upgrade, want %d", e.Kind.Name(), id, got, want)
		}
	}
}
