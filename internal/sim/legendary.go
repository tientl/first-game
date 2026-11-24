package sim

// The legendary spoils. Each boss drops one where it falls (see bosses.go),
// the only one its world will ever have. They fit the gear there is:
//
//   - Giáp vảy Titanoboa, armour of the great snake's scales, works from the
//     bag like any armour: it stops more of a beast's attack than any other,
//     and no venom or poison gets through it.
//   - Giáo răng Spinosaurus, a spear tipped with the sail-back's tooth, is
//     held: the hardest thrust of any spear, and the longest reach.
//   - Áo lông Voi Chúa, the mammoth king's fur, works from the bag like the
//     fur coat: whoever carries it is never cold, on the snow, on a winter
//     night or in winter's snow and wind.
//   - Kiếm răng cưa Carcharo, a blade set with serrated teeth, is held: the
//     quickest blows of any weapon, and hard ones.
//   - Rìu nanh Giganoto, an axe with a fang for its blade, is held for the
//     hardest single blow of any weapon; carried, it chops trees faster than
//     any stone axe.
//   - Vòng răng Megalodon, a necklace of the shark's teeth, is worn: it works
//     from the bag, and whoever wears it swims and keeps going in the water
//     longer, and sails and paddles faster.

// Legendary tuning.
const (
	TitanArmorBlock   = 0.8 // the share of a beast's attack it stops
	SpinoSpearDamage  = 60
	SpinoSpearReach   = 1.2 // extra reach on creatures, held
	SpinoSpearSwing   = 3 * TicksPerSecond / 5
	CarchaSwordDamage = 80
	CarchaSwordSwing  = 2 * TicksPerSecond / 5
	GigaAxeDamage     = 150
	GigaAxeSwing      = 4 * TicksPerSecond / 5
	GigaAxeChop       = 16  // a stroke at a tree, in HitPower units: an obsidian axe does 10
	MegaSwimBoost     = 1.6 // swimming speed, times
	MegaBreathBoost   = 1.5 // how long a swimmer keeps going, times
	MegaSailBoost     = 1.25
)

// legendaries lists the legendary item of each boss, in boss order.
var legendaries = [NumBosses]ItemKind{ItemTitanArmor, ItemSpinoSpear, ItemKingCoat, ItemCarchaSword, ItemGigaAxe, ItemMegaNecklace}

func init() {
	for k, name := range map[ItemKind]string{
		ItemTitanArmor:   "Giáp vảy Titanoboa",
		ItemSpinoSpear:   "Giáo răng Spinosaurus",
		ItemKingCoat:     "Áo lông Voi Chúa",
		ItemCarchaSword:  "Kiếm răng cưa Carcharo",
		ItemGigaAxe:      "Rìu nanh Giganoto",
		ItemMegaNecklace: "Vòng răng Megalodon",
	} {
		itemDefs[k] = ItemDef{Name: name, Stack: 1, Tool: true}
	}
	// The heads were named in drops.go, whose init runs before this file's.
	if h := HeadItem(CreatureMammothKing); h.Valid() {
		itemDefs[h].Name = "Đầu Voi Ma Mút Chúa"
	}
}

// LegendaryItem is the legendary item boss k drops, or ItemNone.
func LegendaryItem(k CreatureKind) ItemKind {
	if i, ok := BossIndex(k); ok {
		return legendaries[i]
	}
	return ItemNone
}

// Legendary reports whether k is one of the bosses' legendary items.
func (k ItemKind) Legendary() bool { return k >= ItemTitanArmor && k <= ItemMegaNecklace }

// legendaryBlow is the damage and swing of a legendary weapon; ok is false
// for anything else.
func legendaryBlow(held ItemKind) (dmg, swing int, ok bool) {
	switch held {
	case ItemSpinoSpear:
		return SpinoSpearDamage, SpinoSpearSwing, true
	case ItemCarchaSword:
		return CarchaSwordDamage, CarchaSwordSwing, true
	case ItemGigaAxe:
		return GigaAxeDamage, GigaAxeSwing, true
	}
	return 0, 0, false
}

// legendaryReach is the extra reach a held legendary weapon gives.
func legendaryReach(held ItemKind) float64 {
	if held == ItemSpinoSpear {
		return SpinoSpearReach
	}
	return 0
}

// legendaryArmor is the share of an attack the legendary armour in inv
// stops, 0 if there is none.
func legendaryArmor(inv *Inventory) float64 {
	if inv.Count(ItemTitanArmor) > 0 {
		return TitanArmorBlock
	}
	return 0
}

// PoisonProof reports whether inv holds what keeps venom out.
func PoisonProof(inv *Inventory) bool { return inv.Count(ItemTitanArmor) > 0 }

// NeverCold reports whether inv holds what keeps every cold out.
func NeverCold(inv *Inventory) bool { return inv.Count(ItemKingCoat) > 0 }

// swimBoost is how much faster than others a swimmer with inv swims.
func swimBoost(inv *Inventory) float64 {
	if inv.Count(ItemMegaNecklace) > 0 {
		return MegaSwimBoost
	}
	return 1
}

// sailBoost is how much faster than others a boat goes with inv aboard.
func sailBoost(inv *Inventory) float64 {
	if inv.Count(ItemMegaNecklace) > 0 {
		return MegaSailBoost
	}
	return 1
}

// legendaryChop is the power of a stroke at a tree with the legendary axe
// in inv, 0 if there is none.
func legendaryChop(inv *Inventory) int {
	if inv.Count(ItemGigaAxe) > 0 {
		return GigaAxeChop
	}
	return 0
}
