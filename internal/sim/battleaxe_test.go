package sim

import "testing"

func TestTheBattleAxeHitsHardAndSlow(t *testing.T) {
	if d := WeaponDamage(ItemBattleAxe); d != BattleAxeDamage || d <= swordDamage[TierPlatinum] || d >= swordDamage[TierDiamond] {
		t.Errorf("battle axe damage %d: want more than the agate knife's %d, less than the obsidian one's %d",
			d, swordDamage[TierPlatinum], swordDamage[TierDiamond])
	}
	if Punches(ItemBattleAxe) || BlowDamage(ItemBattleAxe, CreatureRex) != BattleAxeDamage {
		t.Error("the battle axe is not a weapon against a tyrant's scales")
	}
	for _, other := range []ItemKind{ItemNone, ItemSpear, ItemIvorySpear, ItemCopperSword, ItemDiamondSword, ItemAxe} {
		if SwingTicks(ItemBattleAxe) <= SwingTicks(other) {
			t.Errorf("the battle axe swings no slower than item %d", other)
		}
	}
	if !ItemBattleAxe.Def().Tool || ItemBattleAxe.MaxCount() != 1 {
		t.Error("the battle axe is not a kept, single weapon")
	}
	if c, _ := ItemBattleAxe.ToolKind(); c != NoTool {
		t.Error("the battle axe counts as a tool for chopping")
	}
}

func TestCraftingABattleAxe(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	r, ok := RecipeByID(RecipeBattleAxe)
	if !ok || r.Gives != ItemBattleAxe {
		t.Fatalf("recipe %v,%v: want one that gives the battle axe", r, ok)
	}
	w.players[id].Inv = InventoryWith(r.Inputs...)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeBattleAxe})
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeCrafted || p.Inv.Count(ItemBattleAxe) != 1 || p.Inv.Count(ItemIronBar) != 0 {
		t.Fatalf("after crafting: notice %v, inventory %v", p.Notice, p.Inv)
	}
	// One is enough.
	w.players[id].Inv.Add(ItemWood, 2)
	w.players[id].Inv.Add(ItemIronBar, 2)
	w.players[id].Inv.Add(ItemHide, 1)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeBattleAxe})
	if p := mustPlayer(t, w, id); p.Notice != NoticeHaveOne || p.Inv.Count(ItemBattleAxe) != 1 {
		t.Errorf("a second battle axe: notice %v, %d axes", p.Notice, p.Inv.Count(ItemBattleAxe))
	}
}
