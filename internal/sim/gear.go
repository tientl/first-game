package sim

// Swords and armour of each metal tier, and stone walls. A sword works in
// the hand, like the spear; armour works from anywhere in the bag, the best
// piece carried; a stone wall blocks the way like a wooden one but takes a
// pickaxe, and many more hits, to bring down.

// Gear tuning.
const (
	StoneWallHits   = 8 // hits with a stone pickaxe's power to take a stone wall down
	StoneWallRefund = 2 // stone back from a stone wall taken down
)

// swords and armors list the sword and the armour of each tier. Stone has
// neither: its weapon is the spear.
var (
	swords = [NumTiers]ItemKind{ItemNone, ItemCopperSword, ItemIronSword, ItemPlatinumSword, ItemDiamondSword}
	armors = [NumTiers]ItemKind{ItemNone, ItemCopperArmor, ItemIronArmor, ItemPlatinumArmor, ItemDiamondArmor}

	// swordDamage is the damage of one blow with each tier's sword.
	swordDamage = [NumTiers]int{0, 3, 4, 5, 7}
	// armorBlock is the share of an animal's attack each tier's armour stops.
	armorBlock = [NumTiers]float64{0, 0.25, 0.4, 0.55, 0.7}
)

// SwordItem is the sword made of tier t (ItemNone for stone).
func SwordItem(t Tier) ItemKind { return swords[min(t, NumTiers-1)] }

// ArmorItem is the armour made of tier t (ItemNone for stone).
func ArmorItem(t Tier) ItemKind { return armors[min(t, NumTiers-1)] }

// SwordTier is the tier of sword k; ok is false when k is no sword.
func SwordTier(k ItemKind) (Tier, bool) { return tierIn(&swords, k) }

// ArmorTier is the tier of armour k; ok is false when k is no armour.
func ArmorTier(k ItemKind) (Tier, bool) { return tierIn(&armors, k) }

func tierIn(list *[NumTiers]ItemKind, k ItemKind) (Tier, bool) {
	if k == ItemNone {
		return 0, false
	}
	for t, item := range list {
		if item == k {
			return Tier(t), true
		}
	}
	return 0, false
}

// ArmorBlock is the share of an animal's attack armour of tier t stops.
func ArmorBlock(t Tier) float64 { return armorBlock[min(t, NumTiers-1)] }

// BestArmor is the tier of the best armour in the inventory; ok is false
// when there is none.
func (inv *Inventory) BestArmor() (Tier, bool) {
	for t := NumTiers - 1; t > TierStone; t-- {
		if inv.Count(armors[t]) > 0 {
			return t, true
		}
	}
	return 0, false
}

// struck hurts p with an animal's attack, less what p's armour stops.
// Every creature that harms a player goes through here.
func struck(p *Player, dmg float64) {
	if t, ok := p.Inv.BestArmor(); ok {
		dmg *= 1 - ArmorBlock(t)
	}
	hurt(p, dmg)
}

// SwordRecipe and ArmorRecipe are the recipes for tier t's sword and
// armour (copper or better).
func SwordRecipe(t Tier) RecipeID { return recipeSwords + RecipeID(t-TierCopper) }
func ArmorRecipe(t Tier) RecipeID { return recipeArmors + RecipeID(t-TierCopper) }

// gearCost is how much of its tier's material a sword and an armour take.
const (
	swordCost = 2 // besides a stick for the grip
	armorCost = 5
)

// fillGear writes the sword and armour recipes into the table.
func fillGear() {
	for t := TierCopper; t < NumTiers; t++ {
		m := TierMaterial(t)
		sword, armor := SwordItem(t), ArmorItem(t)
		recipes[SwordRecipe(t)] = Recipe{
			ID: SwordRecipe(t), Name: sword.Def().Name, Gives: sword,
			Inputs: []Ingredient{{Item: ItemStick, Count: 1}, {Item: m, Count: swordCost}},
		}
		recipes[ArmorRecipe(t)] = Recipe{
			ID: ArmorRecipe(t), Name: armor.Def().Name, Gives: armor,
			Inputs: []Ingredient{{Item: m, Count: armorCost}},
		}
	}
}
