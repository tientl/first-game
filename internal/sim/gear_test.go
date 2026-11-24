package sim

import (
	"math"
	"slices"
	"testing"
)

func TestSwordsHitHarderTierByTier(t *testing.T) {
	w := flatWorld()
	p := w.players[w.AddPlayer("a")]
	if got := damageWith(p); got != FistDamage {
		t.Fatalf("bare hands: %d damage, want %d", got, FistDamage)
	}
	holding(w, p.ID, ItemSpear)
	prev := damageWith(p)
	for tier := TierCopper; tier < NumTiers; tier++ {
		holding(w, p.ID, SwordItem(tier))
		got := damageWith(p)
		if got <= prev {
			t.Errorf("%s: %d damage, want more than the tier below's %d", SwordItem(tier).Def().Name, got, prev)
		}
		prev = got
	}
	// A sword in the bag is no use: it must be in hand.
	p.Inv[0] = Slot{}
	p.Inv.Add(ItemDiamondSword, 1)
	p.Selected = 9
	if got := damageWith(p); got != FistDamage {
		t.Errorf("diamond sword in the bag: %d damage, want %d (bare hands)", got, FistDamage)
	}
}

func TestArmourStopsPartOfAnAnimalsAttack(t *testing.T) {
	w := flatWorld()
	p := w.players[w.AddPlayer("a")]
	struck(p, 10)
	if got := MaxHealth - p.Health; got != 10 {
		t.Fatalf("no armour: lost %v health, want 10", got)
	}
	// The best piece carried counts, wherever it is in the bag.
	p.Health = MaxHealth
	p.Inv.Add(ItemCopperArmor, 1)
	p.Inv.Add(ItemPlatinumArmor, 1)
	struck(p, 10)
	if got, want := MaxHealth-p.Health, 10*(1-ArmorBlock(TierPlatinum)); math.Abs(got-want) > 1e-9 {
		t.Errorf("with copper and platinum armour: lost %v, want %v", got, want)
	}
	// Starving is not an attack: armour does nothing for it.
	p.Health = MaxHealth
	hurt(p, 10)
	if got := MaxHealth - p.Health; got != 10 {
		t.Errorf("hurt through armour: lost %v, want 10", got)
	}
	prev := 0.0
	for tier := TierCopper; tier < NumTiers; tier++ {
		if b := ArmorBlock(tier); b <= prev || b >= 1 {
			t.Errorf("%s stops %v of a blow, after %v below it", ArmorItem(tier).Def().Name, b, prev)
		}
		prev = ArmorBlock(tier)
	}
}

func TestEveryTierHasItsSwordAndArmour(t *testing.T) {
	for tier := TierCopper; tier < NumTiers; tier++ {
		for _, g := range []struct {
			item ItemKind
			id   RecipeID
			of   func(ItemKind) (Tier, bool)
		}{{SwordItem(tier), SwordRecipe(tier), SwordTier}, {ArmorItem(tier), ArmorRecipe(tier), ArmorTier}} {
			if got, ok := g.of(g.item); !ok || got != tier || !g.item.Valid() || !g.item.Def().Tool {
				t.Fatalf("%v (tier %d): reads as tier %d, ok %v", g.item, tier, got, ok)
			}
			r, ok := RecipeByID(g.id)
			// Knives are made of their tier's stone, armour of what the hunt gives.
			want := TierMaterial(tier)
			if _, armour := ArmorTier(g.item); armour {
				want = ItemHide
			}
			if !ok || r.Gives != g.item || !slices.ContainsFunc(r.Inputs, func(in Ingredient) bool { return in.Item == want }) {
				t.Errorf("recipe for %s: %+v", g.item.Def().Name, r)
			}
		}
	}
	if _, ok := SwordTier(ItemSpear); ok {
		t.Error("the spear reads as a sword")
	}
	if RecipeChest != ArmorRecipe(NumTiers-1)+1 {
		t.Errorf("the recipe after the armours is %d, the last armour recipe is %d", RecipeChest, ArmorRecipe(NumTiers-1))
	}
}

func TestAStoneWallBlocksAndTakesAPickaxe(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a") // on (16, 16)
	w.players[id].Inv.Add(ItemStone, 4)
	w.players[id].Dir = DirRight
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeStoneWall})
	if e, ok := w.EntityAt(17, 16); !ok || e.Kind != EntityStoneWall || w.players[id].Inv.Count(ItemStone) != 0 {
		t.Fatalf("after building: %+v at (17, 16), %d stone left", e, w.players[id].Inv.Count(ItemStone))
	}

	// Walking right stops at the wall.
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 40)
	if p := mustPlayer(t, w, id); p.X+PlayerRadius > 17 {
		t.Fatalf("walked into the wall: x = %v", p.X)
	}

	// No pickaxe, no way through; an axe does not do.
	w.players[id].Inv.Add(ItemAxe, 1)
	n, p := swingsToBreak(t, EntityStoneWall, false, ItemAxe)
	if n != -1 || p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ItemPickaxe {
		t.Fatalf("with an axe: %d swings, notice %v/%d; want refused, asking for a pickaxe", n, p.Notice, p.NoticeArg)
	}
	n, p = swingsToBreak(t, EntityStoneWall, false, ItemPickaxe)
	if n != StoneWallHits || p.Inv.Count(ItemStone) != StoneWallRefund {
		t.Errorf("with a stone pickaxe: %d swings and %d stone back, want %d and %d", n, p.Inv.Count(ItemStone), StoneWallHits, StoneWallRefund)
	}
}

func TestBlowsVaryAroundTheWeaponsDamage(t *testing.T) {
	w := flatWorld()
	p := w.players[w.AddPlayer("a")]
	holding(w, p.ID, SwordItem(TierIron))
	base := damageWith(p)
	lo, hi, crits := base*10, 0, 0
	for range 400 {
		d := w.blow(p)
		lo, hi = min(lo, d), max(hi, d)
		if float64(d) > float64(base)*(1+DamageSpread)+0.5 {
			crits++
		}
	}
	if lo < int(float64(base)*(1-DamageSpread)) || lo == hi || crits == 0 {
		t.Fatalf("blows of %d to %d with %d critical, base %d", lo, hi, crits, base)
	}
	if WeaponDamage(ItemAxe) <= FistDamage || WeaponDamage(ItemAxe) >= WeaponDamage(SwordItem(TierCopper)) {
		t.Errorf("axe %d: want between a punch and a copper sword", WeaponDamage(ItemAxe))
	}
}
