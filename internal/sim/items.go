package sim

// ItemKind identifies a stackable inventory item.
type ItemKind uint8

const (
	ItemNone ItemKind = iota
	ItemWood
	ItemStone
	ItemGrass
	ItemBerry
	ItemStick // picked up from the ground
	ItemAxe   // a tool: needed to chop trees
	ItemMeat  // from rabbits and snakes
	ItemWorm  // bait for fishing
	ItemFish
	ItemCookedMeat // cooked at a campfire
	ItemCookedFish
	ItemPickaxe     // tool: needed to mine boulders
	ItemTorch       // held: light and warmth, burns down
	ItemSpear       // held tool: hits creatures harder, from further
	ItemRod         // held tool: needed to fish
	ItemRoastBerry  // berries cooked at a campfire
	ItemPebble      // a small stone picked up from the ground: for the first tools
	ItemHoe         // held tool: tills soil, turning up worms and sweet potatoes
	ItemPotato      // sweet potato: eaten, roasted, or planted in tilled soil
	ItemRoastPotato // sweet potato roasted at a campfire
	ItemCactus      // desert: cactus flesh, eaten
	ItemMushroom    // marsh and caves: eaten
	ItemCoal        // caves and ash: burns in a campfire twice as long as wood

	// Ores, bars and the tools of each metal tier: see ores.go.
	ItemCopperOre
	ItemIronOre
	ItemPlatinumOre
	ItemDiamond
	ItemCopperBar
	ItemIronBar
	ItemPlatinumBar
	ItemCopperAxe
	ItemIronAxe
	ItemPlatinumAxe
	ItemDiamondAxe
	ItemCopperPickaxe
	ItemIronPickaxe
	ItemPlatinumPickaxe
	ItemDiamondPickaxe
	ItemCopperHoe
	ItemIronHoe
	ItemPlatinumHoe
	ItemDiamondHoe
	ItemClam      // clams dug from the sand by the sea
	ItemRoastClam // clams roasted at a campfire
	ItemCrabMeat  // from crabs on the beach
	ItemCookedCrab

	// Swords and armour of each metal tier: see gear.go.
	ItemCopperSword
	ItemIronSword
	ItemPlatinumSword
	ItemDiamondSword
	ItemCopperArmor
	ItemIronArmor
	ItemPlatinumArmor
	ItemDiamondArmor
	NumItems
)

// ItemDef is the static description of an item kind.
type ItemDef struct {
	Name   string
	Hunger float64 // restored when eaten; 0 means not edible
	Health float64 // restored when eaten; negative hurts (raw food), but never below 1
	Stack  int     // most one inventory slot holds; 0 means MaxStack
	Tool   bool    // kept, not used up; a player carries at most one
}

var itemDefs = [NumItems]ItemDef{
	ItemNone:  {Name: "?"},
	ItemWood:  {Name: "Gỗ"},
	ItemStone: {Name: "Đá"},
	ItemGrass: {Name: "Cỏ khô"},
	ItemBerry: {Name: "Quả mọng", Hunger: 12, Health: 1},
	ItemStick: {Name: "Cành cây"},
	ItemAxe:   {Name: "Rìu đá", Stack: 1, Tool: true},
	ItemMeat:  {Name: "Thịt sống", Hunger: 15, Health: -2},
	ItemWorm:  {Name: "Giun"},
	ItemFish:  {Name: "Cá sống", Hunger: 10, Health: -1},

	ItemCookedMeat: {Name: "Thịt nướng", Hunger: 35, Health: 5},
	ItemCookedFish: {Name: "Cá nướng", Hunger: 28, Health: 6},
	ItemPickaxe:    {Name: "Cúp đá", Stack: 1, Tool: true},
	ItemTorch:      {Name: "Đuốc", Stack: 5},
	ItemSpear:      {Name: "Giáo", Stack: 1, Tool: true},
	ItemRod:        {Name: "Cần câu", Stack: 1, Tool: true},
	ItemRoastBerry: {Name: "Quả mọng nướng", Hunger: 16, Health: 3},

	ItemPebble:      {Name: "Đá cuội"},
	ItemHoe:         {Name: "Cuốc đá", Stack: 1, Tool: true},
	ItemPotato:      {Name: "Khoai lang", Hunger: 8},
	ItemRoastPotato: {Name: "Khoai nướng", Hunger: 26, Health: 3},
	ItemCactus:      {Name: "Thịt xương rồng", Hunger: 8, Health: 2},
	ItemMushroom:    {Name: "Nấm", Hunger: 6, Health: 1},
	ItemCoal:        {Name: "Than"},

	ItemCopperOre:   {Name: "Quặng đồng"},
	ItemIronOre:     {Name: "Quặng sắt"},
	ItemPlatinumOre: {Name: "Quặng bạch kim"},
	ItemDiamond:     {Name: "Kim cương"},
	ItemCopperBar:   {Name: "Thỏi đồng"},
	ItemIronBar:     {Name: "Thỏi sắt"},
	ItemPlatinumBar: {Name: "Thỏi bạch kim"},

	ItemCopperAxe:       {Name: "Rìu đồng", Stack: 1, Tool: true},
	ItemIronAxe:         {Name: "Rìu sắt", Stack: 1, Tool: true},
	ItemPlatinumAxe:     {Name: "Rìu bạch kim", Stack: 1, Tool: true},
	ItemDiamondAxe:      {Name: "Rìu kim cương", Stack: 1, Tool: true},
	ItemCopperPickaxe:   {Name: "Cúp đồng", Stack: 1, Tool: true},
	ItemIronPickaxe:     {Name: "Cúp sắt", Stack: 1, Tool: true},
	ItemPlatinumPickaxe: {Name: "Cúp bạch kim", Stack: 1, Tool: true},
	ItemDiamondPickaxe:  {Name: "Cúp kim cương", Stack: 1, Tool: true},
	ItemCopperHoe:       {Name: "Cuốc đồng", Stack: 1, Tool: true},
	ItemIronHoe:         {Name: "Cuốc sắt", Stack: 1, Tool: true},
	ItemPlatinumHoe:     {Name: "Cuốc bạch kim", Stack: 1, Tool: true},
	ItemDiamondHoe:      {Name: "Cuốc kim cương", Stack: 1, Tool: true},
	ItemClam:            {Name: "Sò", Hunger: 6},
	ItemRoastClam:       {Name: "Sò nướng", Hunger: 18, Health: 4},
	ItemCrabMeat:        {Name: "Thịt cua", Hunger: 10, Health: -1},
	ItemCookedCrab:      {Name: "Cua nướng", Hunger: 26, Health: 6},

	ItemCopperSword:   {Name: "Kiếm đồng", Stack: 1, Tool: true},
	ItemIronSword:     {Name: "Kiếm sắt", Stack: 1, Tool: true},
	ItemPlatinumSword: {Name: "Kiếm bạch kim", Stack: 1, Tool: true},
	ItemDiamondSword:  {Name: "Kiếm kim cương", Stack: 1, Tool: true},
	ItemCopperArmor:   {Name: "Giáp đồng", Stack: 1, Tool: true},
	ItemIronArmor:     {Name: "Giáp sắt", Stack: 1, Tool: true},
	ItemPlatinumArmor: {Name: "Giáp bạch kim", Stack: 1, Tool: true},
	ItemDiamondArmor:  {Name: "Giáp kim cương", Stack: 1, Tool: true},
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

// MaxCount is the most of k one inventory slot holds.
func (k ItemKind) MaxCount() int {
	if n := k.Def().Stack; n > 0 {
		return n
	}
	return MaxStack
}

// Edible reports whether eating k does anything.
func (k ItemKind) Edible() bool { return k.Def().Hunger > 0 }

// Ingredient is an item kind and a count, used by recipes.
type Ingredient struct {
	Item  ItemKind
	Count int
}
