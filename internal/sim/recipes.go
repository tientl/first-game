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
	recipeSwords                                           // the swords, copper to diamond: see SwordRecipe
	recipeArmors     = recipeSwords + RecipeID(NumTiers-1) // the armours, copper to diamond: see ArmorRecipe
	RecipeChest      = recipeArmors + RecipeID(NumTiers-1)
	RecipeSaddle     = RecipeChest + 1
	RecipeFurCoat    = RecipeSaddle + 1 // see iceage.go
	RecipeIvorySpear = RecipeFurCoat + 1

	RecipeRoastGinkgo = RecipeIvorySpear + 1 // see fruit.go

	// The tribal age: see build.go.
	RecipeDoor      = RecipeRoastGinkgo + 1
	RecipeFloor     = RecipeDoor + 1
	RecipeSpikes    = RecipeFloor + 1
	RecipeTrap      = RecipeSpikes + 1
	RecipePotion    = RecipeTrap + 1
	RecipeBigPotion = RecipePotion + 1
	RecipeBandage   = RecipeBigPotion + 1
	RecipeDinoArmor = RecipeBandage + 1

	// The open sea: see boats.go.
	RecipeRaft     = RecipeDinoArmor + 1
	RecipeSailboat = RecipeRaft + 1

	NumRecipes = RecipeSailboat + 1
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
		Name:        "Lưỡi đá lửa",
		Inputs:      []Ingredient{{Item: ItemCopperOre, Count: 1}},
		Gives:       ItemCopperBar,
		NearFurnace: true,
	},
	RecipeSmeltIron: {
		ID:          RecipeSmeltIron,
		Name:        "Ngọc bích mài",
		Inputs:      []Ingredient{{Item: ItemIronOre, Count: 1}},
		Gives:       ItemIronBar,
		NearFurnace: true,
	},
	RecipeSmeltPlatinum: {
		ID:          RecipeSmeltPlatinum,
		Name:        "Mã não mài",
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
	RecipeSaddle: {
		ID:     RecipeSaddle,
		Name:   "Yên da",
		Inputs: []Ingredient{{Item: ItemHide, Count: 4}, {Item: ItemGrass, Count: 2}, {Item: ItemBone, Count: 1}},
		Gives:  ItemSaddle,
	},
	RecipeFurCoat: {
		ID:     RecipeFurCoat,
		Name:   "Áo lông",
		Inputs: []Ingredient{{Item: ItemFur, Count: 4}, {Item: ItemHide, Count: 2}},
		Gives:  ItemFurCoat,
	},
	RecipeIvorySpear: {
		ID:     RecipeIvorySpear,
		Name:   "Giáo ngà",
		Inputs: []Ingredient{{Item: ItemStick, Count: 2}, {Item: ItemIvory, Count: 1}, {Item: ItemGrass, Count: 1}},
		Gives:  ItemIvorySpear,
	},
	RecipeRoastGinkgo: {
		ID:       RecipeRoastGinkgo,
		Name:     "Hạt bạch quả rang",
		Inputs:   []Ingredient{{Item: ItemGinkgoNut, Count: 1}},
		Gives:    ItemRoastGinkgo,
		NearFire: true,
	},
	RecipeDoor: {
		ID: RecipeDoor, Name: "Cửa gỗ",
		Inputs: []Ingredient{{Item: ItemWood, Count: 3}},
		Places: EntityDoor,
	},
	RecipeFloor: {
		ID: RecipeFloor, Name: "Sàn gỗ",
		Inputs: []Ingredient{{Item: ItemWood, Count: 1}},
		Places: EntityFloor,
	},
	RecipeSpikes: {
		ID: RecipeSpikes, Name: "Hàng rào chông",
		Inputs: []Ingredient{{Item: ItemWood, Count: 2}, {Item: ItemStick, Count: 3}},
		Places: EntitySpikes,
	},
	RecipeTrap: {
		ID: RecipeTrap, Name: "Bẫy thú",
		Inputs: []Ingredient{{Item: ItemStick, Count: 3}, {Item: ItemGrass, Count: 2}, {Item: ItemStone, Count: 1}},
		Places: EntityTrap,
	},
	RecipePotion: {
		ID: RecipePotion, Name: "Bình máu",
		Inputs:   []Ingredient{{Item: ItemFlowerRed, Count: 2}, {Item: ItemBerry, Count: 1}, {Item: ItemMushroom, Count: 1}},
		Gives:    ItemPotion,
		NearFire: true,
	},
	RecipeBigPotion: {
		ID: RecipeBigPotion, Name: "Bình máu lớn",
		Inputs:   []Ingredient{{Item: ItemPotion, Count: 1}, {Item: ItemGlowSpore, Count: 2}, {Item: ItemFlowerBlue, Count: 1}},
		Gives:    ItemBigPotion,
		NearFire: true,
	},
	RecipeBandage: {
		ID: RecipeBandage, Name: "Băng tơ nhện",
		Inputs: []Ingredient{{Item: ItemSilk, Count: 2}, {Item: ItemGrass, Count: 1}},
		Gives:  ItemBandage,
	},
	RecipeDinoArmor: {
		ID: RecipeDinoArmor, Name: "Giáp da khủng long",
		Inputs: []Ingredient{{Item: ItemDinoHide, Count: 4}, {Item: ItemBone, Count: 2}, {Item: ItemSilk, Count: 1}},
		Gives:  ItemDinoArmor,
	},
	RecipeRaft: {
		ID: RecipeRaft, Name: "Bè gỗ",
		Inputs: []Ingredient{{Item: ItemWood, Count: 8}, {Item: ItemGrass, Count: 4}},
		Gives:  ItemRaft,
	},
	RecipeSailboat: {
		ID: RecipeSailboat, Name: "Thuyền buồm",
		Inputs: []Ingredient{{Item: ItemWood, Count: 20}, {Item: ItemHide, Count: 4}, {Item: ItemGrass, Count: 8}},
		Gives:  ItemSailboat,
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

	// Taming and riding: see tame.go. Arg: CreatureKind unless noted.
	NoticeFed      // one more meal towards taming it
	NoticeTamed    // it is yours now
	NoticeTooWild  // a hunter must be beaten down before it takes food
	NoticeAngry    // an angry beast will not eat from your hand
	NoticeNotYours // someone else's beast
	NoticePetStay  // your beast waits here
	NoticePetCome  // your beast follows you
	NoticeNoMount  // no beast of yours to ride close by; Arg unused
	NoticeCantRide // this beast cannot be ridden
	NoticeHealed   // your beast ate and is mended
	NoticeSkittish // it bolts: wait till it stops for breath, then feed it

	// The tribal age.
	NoticePickedUp  // Arg: ItemKind picked up off the ground
	NoticeCantPlace // Arg: ItemKind held that V cannot set down
	NoticePlantBite // a man-eating plant bit you
	NoticeLightning // struck by lightning
	NoticeTornado   // caught in a tornado
	NoticeWebbed    // tangled in a spider's web
	NoticeTreasure  // an old chest opened
	NoticeHomeSet   // this bed is where you will come back

	// The open sea: see boats.go. Arg: BoatKind unless noted.
	NoticeLaunched  // a boat set on the water
	NoticeNeedWater // no open water in front to set the boat on
	NoticeNoShore   // no ground near enough to step ashore
	NoticeSunk      // your boat sank under you
	NoticeDrowning  // out of strength in the water; Arg unused

	NoticeIntoCave  // down into the caves
	NoticeOutOfCave // back up into daylight

	NoticeFertilized    // dung spread; Arg: the EntityKind fed
	NoticeNothingToFeed // nothing growing in front to feed dung to
	NoticeBees          // a swarm of bees is after you
	NoticeBeesCalm      // the torch smoke calms the bees
	NoticeCrit          // your blow was a critical one
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
	CmdRide                      // get on my nearest beast that can be ridden, or off the one I ride
	CmdPlace                     // set down what I hold: a head on a post, a torch in the ground, a seed
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
