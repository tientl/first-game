package sim

// RecipeID identifies a crafting recipe.
type RecipeID uint8

const (
	RecipeCampfire RecipeID = iota
	RecipeAxe
	RecipeCookMeat
	RecipeCookFish
	RecipePickaxe
	RecipeTorch
	RecipeSpear
	RecipeRod
	RecipeWall
	RecipeBed
	RecipeRoastBerry
	RecipeHoe
	RecipeRoastPotato
	RecipeFurnace
	RecipeSmeltCopper
	RecipeSmeltIron
	RecipeSmeltPlatinum
	recipeTierTools // the metal tools start here: see TierToolRecipe
)

// Recipes after the block of metal tools. Append new ones before NumRecipes.
const (
	RecipeRoastClam = recipeTierTools + RecipeID(NumToolClasses-1)*RecipeID(NumTiers-1) + iota
	RecipeCookCrab
	RecipeStoneWall
	recipeSwords                                       // the swords, copper to diamond: see SwordRecipe
	recipeArmors = recipeSwords + RecipeID(NumTiers-1) // the armours, copper to diamond: see ArmorRecipe
	RecipeChest  = recipeArmors + RecipeID(NumTiers-1)

	NumRecipes = RecipeChest + 1
)

// Recipe turns ingredients into either a structure placed in front of the
// player (Places) or an item put in the bag (Gives).
type Recipe struct {
	ID          RecipeID
	Name        string
	Inputs      []Ingredient
	Places      EntityKind
	Gives       ItemKind
	NearFire    bool // only in the light of a burning campfire: cooking
	NearFurnace bool // only beside a lit furnace: smelting
}

var recipes = [NumRecipes]Recipe{
	RecipeCampfire: {
		ID:   RecipeCampfire,
		Name: "Lửa trại",
		Inputs: []Ingredient{
			{Item: ItemWood, Count: 2},
			{Item: ItemGrass, Count: 2},
			{Item: ItemStone, Count: 3},
		},
		Places: EntityCampfire,
	},
	// The first tools are made from what lies on the ground: stone only comes
	// from boulders, and those need the pickaxe.
	RecipeAxe: {
		ID:   RecipeAxe,
		Name: "Rìu đá",
		Inputs: []Ingredient{
			{Item: ItemStick, Count: 2},
			{Item: ItemPebble, Count: 1},
		},
		Gives: ItemAxe,
	},
	RecipeCookMeat: {
		ID:       RecipeCookMeat,
		Name:     "Thịt nướng",
		Inputs:   []Ingredient{{Item: ItemMeat, Count: 1}},
		Gives:    ItemCookedMeat,
		NearFire: true,
	},
	RecipeCookFish: {
		ID:       RecipeCookFish,
		Name:     "Cá nướng",
		Inputs:   []Ingredient{{Item: ItemFish, Count: 1}},
		Gives:    ItemCookedFish,
		NearFire: true,
	},
	RecipePickaxe: {
		ID:     RecipePickaxe,
		Name:   "Cúp đá",
		Inputs: []Ingredient{{Item: ItemStick, Count: 2}, {Item: ItemPebble, Count: 2}},
		Gives:  ItemPickaxe,
	},
	RecipeTorch: {
		ID:     RecipeTorch,
		Name:   "Đuốc",
		Inputs: []Ingredient{{Item: ItemStick, Count: 1}, {Item: ItemGrass, Count: 1}},
		Gives:  ItemTorch,
	},
	RecipeSpear: {
		ID:     RecipeSpear,
		Name:   "Giáo",
		Inputs: []Ingredient{{Item: ItemStick, Count: 2}, {Item: ItemStone, Count: 1}},
		Gives:  ItemSpear,
	},
	RecipeRod: {
		ID:     RecipeRod,
		Name:   "Cần câu",
		Inputs: []Ingredient{{Item: ItemStick, Count: 3}, {Item: ItemGrass, Count: 2}},
		Gives:  ItemRod,
	},
	RecipeWall: {
		ID:     RecipeWall,
		Name:   "Tường gỗ",
		Inputs: []Ingredient{{Item: ItemWood, Count: 2}},
		Places: EntityWall,
	},
	RecipeBed: {
		ID:     RecipeBed,
		Name:   "Giường cỏ",
		Inputs: []Ingredient{{Item: ItemGrass, Count: 6}, {Item: ItemWood, Count: 2}},
		Places: EntityBed,
	},
	RecipeRoastBerry: {
		ID:       RecipeRoastBerry,
		Name:     "Quả mọng nướng",
		Inputs:   []Ingredient{{Item: ItemBerry, Count: 1}},
		Gives:    ItemRoastBerry,
		NearFire: true,
	},
	RecipeHoe: {
		ID:     RecipeHoe,
		Name:   "Cuốc đá",
		Inputs: []Ingredient{{Item: ItemStick, Count: 3}, {Item: ItemPebble, Count: 1}},
		Gives:  ItemHoe,
	},
	RecipeRoastClam: {
		ID:       RecipeRoastClam,
		Name:     "Sò nướng",
		Inputs:   []Ingredient{{Item: ItemClam, Count: 2}},
		Gives:    ItemRoastClam,
		NearFire: true,
	},
	RecipeCookCrab: {
		ID:       RecipeCookCrab,
		Name:     "Cua nướng",
		Inputs:   []Ingredient{{Item: ItemCrabMeat, Count: 1}},
		Gives:    ItemCookedCrab,
		NearFire: true,
	},
	RecipeRoastPotato: {
		ID:       RecipeRoastPotato,
		Name:     "Khoai nướng",
		Inputs:   []Ingredient{{Item: ItemPotato, Count: 1}},
		Gives:    ItemRoastPotato,
		NearFire: true,
	},
	RecipeFurnace: {
		ID:     RecipeFurnace,
		Name:   "Lò nung",
		Inputs: []Ingredient{{Item: ItemStone, Count: 8}, {Item: ItemWood, Count: 2}},
		Places: EntityFurnace,
	},
	RecipeSmeltCopper: {
		ID:          RecipeSmeltCopper,
		Name:        "Thỏi đồng",
		Inputs:      []Ingredient{{Item: ItemCopperOre, Count: 1}},
		Gives:       ItemCopperBar,
		NearFurnace: true,
	},
	RecipeSmeltIron: {
		ID:          RecipeSmeltIron,
		Name:        "Thỏi sắt",
		Inputs:      []Ingredient{{Item: ItemIronOre, Count: 1}},
		Gives:       ItemIronBar,
		NearFurnace: true,
	},
	RecipeSmeltPlatinum: {
		ID:          RecipeSmeltPlatinum,
		Name:        "Thỏi bạch kim",
		Inputs:      []Ingredient{{Item: ItemPlatinumOre, Count: 1}},
		Gives:       ItemPlatinumBar,
		NearFurnace: true,
	},
	RecipeStoneWall: {
		ID:     RecipeStoneWall,
		Name:   "Tường đá",
		Inputs: []Ingredient{{Item: ItemStone, Count: 4}},
		Places: EntityStoneWall,
	},
	RecipeChest: {
		ID:     RecipeChest,
		Name:   "Rương",
		Inputs: []Ingredient{{Item: ItemWood, Count: 6}},
		Places: EntityChest,
	},
	// The metal tools are filled in by fillTierTools, swords and armour by fillGear.
}

// RecipeByID returns the recipe with the given id. The Inputs slice is a copy,
// so callers cannot change the recipe table.
func RecipeByID(id RecipeID) (Recipe, bool) {
	if id >= NumRecipes {
		return Recipe{}, false
	}
	r := recipes[id]
	r.Inputs = append([]Ingredient(nil), r.Inputs...)
	return r, true
}

// NoticeKind is a short message the server shows to one player, such as why
// an action failed. The client turns it into text.
type NoticeKind uint8

const (
	NoticeNone          NoticeKind = iota
	NoticeNotEnough                // Arg: RecipeID
	NoticeNoSpace                  // nowhere free to place the structure
	NoticeNotEdible                // Arg: ItemKind
	NoticeNoneLeft                 // Arg: ItemKind the player tried to use
	NoticeBagFull                  // Arg: ItemKind that did not fit
	NoticeCrafted                  // Arg: RecipeID
	NoticeNeedTool                 // Arg: ItemKind of the tool the action needs
	NoticeHaveOne                  // Arg: ItemKind the player already carries as many of as allowed
	NoticeNeedBait                 // Arg: ItemKind used up as bait
	NoticeBitten                   // Arg: CreatureKind that bit the player
	NoticeNeedFire                 // Arg: RecipeID that must be made by a burning campfire
	NoticeNotTired                 // a bed by day: sleep comes only in the evening and at night
	NoticeTooEarly                 // Arg: CreatureKind scared off by striking before the bite
	NoticeBaitStolen               // Arg: ItemKind of the bait a fish took
	NoticeCaught                   // Arg: ItemKind landed on the line
	NoticeNeedSeed                 // Arg: ItemKind to plant in tilled soil
	NoticeNeedFurnace              // Arg: RecipeID that must be made beside a lit furnace
	NoticeChestFull                // Arg: ItemKind that did not fit in the chest
	NoticeChestNotEmpty            // a chest is only taken down empty
)

// CommandKind is a discrete player request, as opposed to held input.
type CommandKind uint8

const (
	CmdNone          CommandKind = iota
	CmdEat                       // eat one of what is in Slot
	CmdCraft                     // craft Recipe
	CmdRespawn                   // come back after dying
	CmdSelect                    // hold hotbar slot Slot
	CmdMoveSlot                  // move the stack in Slot onto To: into, onto or swapped
	CmdQuickMove                 // send the stack in Slot between the hotbar and the bag
	CmdChestPut                  // move the stack in my Slot into my open chest
	CmdChestTake                 // move the stack in the open chest's Slot into my inventory
	CmdChestClose                // close my open chest
	CmdChestTakeDown             // take my open chest down; it must be empty
)

// Command is a discrete player request. Fields not used by Kind are ignored.
type Command struct {
	Kind     CommandKind
	Recipe   RecipeID
	Slot, To uint8 // inventory slot indices
}

// Input is the continuously held player input, sampled every tick.
type Input struct {
	MoveX, MoveY int8 // each -1, 0 or 1
	Action       bool // work on the nearest target (chop, mine, pick, refuel)
}

func (in Input) sanitized() Input {
	clamp := func(v int8) int8 { return max(-1, min(1, v)) }
	return Input{MoveX: clamp(in.MoveX), MoveY: clamp(in.MoveY), Action: in.Action}
}
