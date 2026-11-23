package sim

// RecipeID identifies a crafting recipe.
type RecipeID uint8

const (
	RecipeCampfire RecipeID = iota
	NumRecipes
)

// Recipe turns ingredients into a placed structure.
type Recipe struct {
	ID     RecipeID
	Name   string
	Inputs []Ingredient
	Places EntityKind
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
	NoticeNone      NoticeKind = iota
	NoticeNotEnough            // Arg: RecipeID
	NoticeNoSpace              // nowhere free to place the structure
	NoticeNotEdible            // Arg: ItemKind
	NoticeNoneLeft             // Arg: ItemKind the player tried to use
	NoticeBagFull              // Arg: ItemKind that did not fit
	NoticeCrafted              // Arg: RecipeID
)

// CommandKind is a discrete player request, as opposed to held input.
type CommandKind uint8

const (
	CmdNone CommandKind = iota
	CmdEat
	CmdCraft
	CmdRespawn
)

// Command is a discrete player request. Fields not used by Kind are ignored.
type Command struct {
	Kind   CommandKind
	Item   ItemKind
	Recipe RecipeID
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
