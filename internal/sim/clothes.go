package sim

// Clothes. A new character has nothing on but a fur loincloth (a woman a
// fur wrap and skirt) and goes barefoot. What used to be the settler's
// outfit is made by hand from hides, fur and fibre, and, like armour, is
// worn simply by being carried: a tunic (on a woman an apron dress), a fur
// mantle, trousers with leg wraps, and boots. They are only for show.
// Armour goes over them.

// Wear is the clothes a player has on, a bit for each.
type Wear uint8

const (
	WearTunic Wear = 1 << iota
	WearMantle
	WearLegWraps
	WearBoots

	// WearAll is the settler's whole outfit.
	WearAll = WearTunic | WearMantle | WearLegWraps | WearBoots
)

// clothes lists the clothes, in the order of Wear's bits.
var clothes = [...]ItemKind{ItemTunic, ItemMantle, ItemLegWraps, ItemBoots}

// Clothes returns the items that are worn when carried, in the order of
// Wear's bits.
func Clothes() []ItemKind { return clothes[:] }

// Worn is the clothes carried in inv.
func (inv *Inventory) Worn() Wear {
	var w Wear
	for i, k := range clothes {
		if inv.Count(k) > 0 {
			w |= 1 << i
		}
	}
	return w
}

// dress puts each of the clothes inv lacks into it: the bag's last free
// slots first, so nothing lands in the hand.
func (inv *Inventory) dress() {
	for _, k := range clothes {
		if inv.Count(k) > 0 {
			continue
		}
		for i := NumSlots - 1; i >= 0; i-- {
			if inv[i].Empty() {
				inv[i] = Slot{Item: k, Count: 1}
				break
			}
		}
	}
}

func init() {
	itemDefs[ItemTunic] = ItemDef{Name: "Áo vải lanh", Stack: 1, Tool: true}
	itemDefs[ItemMantle] = ItemDef{Name: "Khăn choàng lông", Stack: 1, Tool: true}
	itemDefs[ItemLegWraps] = ItemDef{Name: "Quần quấn ống", Stack: 1, Tool: true}
	itemDefs[ItemBoots] = ItemDef{Name: "Giày da", Stack: 1, Tool: true}

	recipes[RecipeTunic] = Recipe{
		ID: RecipeTunic, Name: itemDefs[ItemTunic].Name,
		Inputs: []Ingredient{{Item: ItemFlax, Count: 6}, {Item: ItemGrass, Count: 2}},
		Gives:  ItemTunic,
	}
	recipes[RecipeMantle] = Recipe{
		ID: RecipeMantle, Name: itemDefs[ItemMantle].Name,
		Inputs: []Ingredient{{Item: ItemFur, Count: 2}, {Item: ItemHide, Count: 1}},
		Gives:  ItemMantle,
	}
	recipes[RecipeLegWraps] = Recipe{
		ID: RecipeLegWraps, Name: itemDefs[ItemLegWraps].Name,
		Inputs: []Ingredient{{Item: ItemHide, Count: 2}, {Item: ItemFlax, Count: 2}},
		Gives:  ItemLegWraps,
	}
	recipes[RecipeBoots] = Recipe{
		ID: RecipeBoots, Name: itemDefs[ItemBoots].Name,
		Inputs: []Ingredient{{Item: ItemHide, Count: 2}, {Item: ItemGrass, Count: 2}},
		Gives:  ItemBoots,
	}
}
