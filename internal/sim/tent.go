package sim

// The leaf tent (Lều lá): an A-frame of branches thatched with big leaves,
// the first roof a player can put over their head. It is made by hand from
// what lies about, before there is an axe for the wood a bed takes: sticks
// for the frame, dry grass for the thatch and bedding, flax to lash the
// poles at the top. It is carried rolled up in the bag and set down from
// the hand like a torch (V, or a click on the ground), on earth, sand or
// snow but not on bare rock or in a ford, where no stake goes in.
//
// Once standing it is a bed in every way (IsBedLike): slept in through the
// evening and the night, it is where its sleeper comes back after dying,
// the woods do not grow back over it, and a save keeps it. By day an axe in
// the hand takes it down again, and a bed with it, giving back part of
// what they were made of.

// Tent and bed tuning.
const (
	TentHits = 2 // axe blows to take a tent down
	BedHits  = 3 // and a bed
)

func init() {
	itemDefs[ItemLeafTent] = ItemDef{Name: "Lều lá", Stack: 5}
	recipes[RecipeLeafTent] = Recipe{
		ID: RecipeLeafTent, Name: itemDefs[ItemLeafTent].Name,
		Inputs: []Ingredient{{Item: ItemStick, Count: 6}, {Item: ItemGrass, Count: 4}, {Item: ItemFlax, Count: 2}},
		Gives:  ItemLeafTent,
	}
	for _, n := range []string{"tent", "leu", "leula", "leaftent", "leanto"} {
		devItemAliases[n] = ItemLeafTent
	}
}

// IsBedLike reports whether an entity of kind k is slept in: a bed or a
// leaf tent.
func IsBedLike(k EntityKind) bool { return k == EntityBed || k == EntityLeafTent }

// TentGround reports whether a tent's stakes go into ground t: anything
// walked on but bare rock, a cave floor and the stones of a ford.
func TentGround(t Tile) bool {
	switch t {
	case TileRocky, TileCave, TileFord:
		return false
	}
	return t.Walkable()
}

// bedRefund is what a bed or a tent taken down gives back: about half of
// what it took.
func bedRefund(k EntityKind) []Ingredient {
	if k == EntityLeafTent {
		return []Ingredient{{Item: ItemStick, Count: 3}, {Item: ItemGrass, Count: 2}, {Item: ItemFlax, Count: 1}}
	}
	return []Ingredient{{Item: ItemGrass, Count: 3}, {Item: ItemWood, Count: 1}}
}

// useBed is Action (or E) on a bed or a tent: by day, with an axe in the
// hand, it is taken down; otherwise it is slept in, which it can only be
// in the evening and at night.
func (w *World) useBed(p *Player, e *Entity) bool {
	if c, _ := p.Held().ToolKind(); c == ToolAxe && PhaseOf(w.tick) == PhaseDay {
		return w.dismantle(p, e, ToolAxe, bedRefund(e.Kind), w.freeBuild(p))
	}
	return w.sleep(p, e)
}
