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

	// What the great beasts give: see dino.go.
	ItemHide     // a hide, for clothes and saddles
	ItemBone     // bones, for tools and armour
	ItemRexTooth // a tyrant's tooth: the finest blade edge there is
	ItemSaddle   // carried: lets a player ride a tamed beast; see tame.go

	// The ice age: see iceage.go.
	ItemFur        // thick fur, from mammoths and sabre-tooths
	ItemIvory      // a mammoth's tusk
	ItemFurCoat    // carried: keeps the cold of the snow out
	ItemIvorySpear // held: a spear tipped with ivory, the best reach and a hard thrust

	// The fruit of the fruit trees: see fruit.go.
	ItemFig
	ItemGinkgoNut
	ItemRoastGinkgo
	ItemCoconut
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

	ItemCopperOre:   {Name: "Đá lửa thô"},
	ItemIronOre:     {Name: "Ngọc bích thô"},
	ItemPlatinumOre: {Name: "Mã não thô"},
	ItemDiamond:     {Name: "Hắc diện thạch"},
	ItemCopperBar:   {Name: "Lưỡi đá lửa"},
	ItemIronBar:     {Name: "Ngọc bích mài"},
	ItemPlatinumBar: {Name: "Mã não mài"},

	ItemCopperAxe:       {Name: "Rìu đá lửa", Stack: 1, Tool: true},
	ItemIronAxe:         {Name: "Rìu ngọc bích", Stack: 1, Tool: true},
	ItemPlatinumAxe:     {Name: "Rìu mã não", Stack: 1, Tool: true},
	ItemDiamondAxe:      {Name: "Rìu hắc diện thạch", Stack: 1, Tool: true},
	ItemCopperPickaxe:   {Name: "Cúp đá lửa", Stack: 1, Tool: true},
	ItemIronPickaxe:     {Name: "Cúp ngọc bích", Stack: 1, Tool: true},
	ItemPlatinumPickaxe: {Name: "Cúp mã não", Stack: 1, Tool: true},
	ItemDiamondPickaxe:  {Name: "Cúp hắc diện thạch", Stack: 1, Tool: true},
	ItemCopperHoe:       {Name: "Cuốc đá lửa", Stack: 1, Tool: true},
	ItemIronHoe:         {Name: "Cuốc ngọc bích", Stack: 1, Tool: true},
	ItemPlatinumHoe:     {Name: "Cuốc mã não", Stack: 1, Tool: true},
	ItemDiamondHoe:      {Name: "Cuốc hắc diện thạch", Stack: 1, Tool: true},
	ItemClam:            {Name: "Sò", Hunger: 6},
	ItemRoastClam:       {Name: "Sò nướng", Hunger: 18, Health: 4},
	ItemCrabMeat:        {Name: "Thịt cua", Hunger: 10, Health: -1},
	ItemCookedCrab:      {Name: "Cua nướng", Hunger: 26, Health: 6},

	ItemCopperSword:   {Name: "Dao đá lửa", Stack: 1, Tool: true},
	ItemIronSword:     {Name: "Dao ngọc bích", Stack: 1, Tool: true},
	ItemPlatinumSword: {Name: "Dao mã não", Stack: 1, Tool: true},
	ItemDiamondSword:  {Name: "Dao hắc diện thạch", Stack: 1, Tool: true},
	ItemCopperArmor:   {Name: "Áo da thú", Stack: 1, Tool: true},
	ItemIronArmor:     {Name: "Giáp xương", Stack: 1, Tool: true},
	ItemPlatinumArmor: {Name: "Giáp mã não", Stack: 1, Tool: true},
	ItemDiamondArmor:  {Name: "Giáp răng bạo chúa", Stack: 1, Tool: true},

	ItemHide:     {Name: "Da thú"},
	ItemBone:     {Name: "Xương"},
	ItemRexTooth: {Name: "Răng bạo chúa"},
	ItemSaddle:   {Name: "Yên da", Stack: 1, Tool: true},

	ItemFur:        {Name: "Lông thú"},
	ItemIvory:      {Name: "Ngà voi"},
	ItemFurCoat:    {Name: "Áo lông", Stack: 1, Tool: true},
	ItemIvorySpear: {Name: "Giáo ngà", Stack: 1, Tool: true},

	ItemFig:         {Name: "Quả sung", Hunger: 14, Health: 2},
	ItemGinkgoNut:   {Name: "Hạt bạch quả", Hunger: 6, Health: -2},
	ItemRoastGinkgo: {Name: "Hạt bạch quả rang", Hunger: 18, Health: 4},
	ItemCoconut:     {Name: "Dừa", Hunger: 12, Health: 8},
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
