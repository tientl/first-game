package sim

// ItemKind identifies a stackable inventory item.
type ItemKind uint8

const (
	ItemNone ItemKind = iota
	ItemWood
	ItemStone
	ItemGrass
	ItemBerry
	NumItems
)

// ItemDef is the static description of an item kind.
type ItemDef struct {
	Name   string
	Hunger float64 // restored when eaten; 0 means not edible
	Health float64 // restored when eaten
}

var itemDefs = [NumItems]ItemDef{
	ItemNone:  {Name: "?"},
	ItemWood:  {Name: "Gỗ"},
	ItemStone: {Name: "Đá"},
	ItemGrass: {Name: "Cỏ khô"},
	ItemBerry: {Name: "Quả mọng", Hunger: 12, Health: 1},
}

// Valid reports whether k is a real item kind (not ItemNone, in range).
func (k ItemKind) Valid() bool { return k > ItemNone && k < NumItems }

// Def returns the static description of k. Invalid kinds get the ItemNone def.
func (k ItemKind) Def() ItemDef {
	if !k.Valid() {
		return itemDefs[ItemNone]
	}
	return itemDefs[k]
}

// Edible reports whether eating k does anything.
func (k ItemKind) Edible() bool { return k.Def().Hunger > 0 }

// Ingredient is an item kind and a count, used by recipes.
type Ingredient struct {
	Item  ItemKind
	Count int
}

// Inventory holds a count per item kind, each capped at MaxStack.
type Inventory [NumItems]int

// Room returns how many more of k fit in the inventory.
func (inv *Inventory) Room(k ItemKind) int {
	if !k.Valid() {
		return 0
	}
	return MaxStack - inv[k]
}

// HasAll reports whether the inventory holds every ingredient.
func (inv *Inventory) HasAll(in []Ingredient) bool {
	for _, ing := range in {
		if !ing.Item.Valid() || inv[ing.Item] < ing.Count {
			return false
		}
	}
	return true
}

// TakeAll removes every ingredient. The caller must check HasAll first.
func (inv *Inventory) TakeAll(in []Ingredient) {
	for _, ing := range in {
		inv[ing.Item] -= ing.Count
	}
}

// Add adds up to n of k and returns how many were actually added.
func (inv *Inventory) Add(k ItemKind, n int) int {
	n = min(n, inv.Room(k))
	if n <= 0 {
		return 0
	}
	inv[k] += n
	return n
}
