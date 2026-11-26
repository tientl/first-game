package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// Cooking gear and what is cooked with it.
//
//   - Clay is dug by the ponds (and here and there on the beach) and fired
//     beside a lit furnace into a pot or a bowl.
//   - A clay pot, a grill of green sticks and a stone pan are set on a
//     campfire with V: the fire then keeps, cold, when it burns out, and
//     Action on it opens the cooking window instead of stoking (the window
//     has a button for that). A mortar, an earth oven and a drying rack are
//     built like a chest and opened the same way.
//   - A station holds CookIn stacks of ingredients and CookOut of what came
//     out. While it has heat (the fire under a pot, the oven's own; a mortar
//     and a rack need none) it makes the first dish its ingredients allow,
//     again and again. The grill and the rack work on every stack at once,
//     a piece of each at a time.
//   - Dishes give a buff for a while when eaten (see BuffKind), and the bowl
//     or flask they came in goes back in the bag.
//   - A bowl scoops sea water (V at the shore), which dries to salt on a
//     rack near the sea; a flask takes pond water, boiled clean in the pot.
//     TODO(thirst): clean water only mends a little until thirst exists.
//
// Food does not spoil yet (PLAN.md: "để sau"); dried, salted and smoked
// food is simply better food for now.

// --- items -------------------------------------------------------------------

// The cooking items sit at a place of their own, below the heads, so the
// plants added elsewhere before itemsEnd never move them.
const cookItemBase ItemKind = 150

const (
	ItemClay       ItemKind = cookItemBase + iota // dug by the water
	ItemClayPot                                   // set on a campfire: soups and stews
	ItemClayBowl                                  // holds a soup; scoops sea water
	ItemGrill                                     // set on a campfire: roasts four things at once
	ItemStonePan                                  // set on a campfire: fries and sears
	ItemFlask                                     // a hide flask, empty
	ItemDirtyWater                                // a flask of pond water: boil it
	ItemCleanWater                                // a flask of boiled water
	ItemSeaWater                                  // a bowl of sea water: dries to salt
	ItemSalt
	ItemFlour
	ItemSaltedMeat // meat rubbed with salt: smoked on the rack
	ItemSmokedMeat
	ItemJerky
	ItemDriedFish
	ItemStew // meat and vegetables: heals fast
	ItemHotSoup
	ItemDinoStew
	ItemHerbTea
	ItemHerbFish
	ItemSearedMeat
	ItemFriedEgg
	ItemHerbMedicine
	ItemJuice
	ItemBread
	ItemFruitPie
	cookItemsEnd
)

var cookItemDefs = map[ItemKind]ItemDef{
	ItemClay:         {Name: "Đất sét"},
	ItemClayPot:      {Name: "Nồi đất", Stack: 5},
	ItemClayBowl:     {Name: "Bát đất", Stack: 20},
	ItemGrill:        {Name: "Giàn nướng", Stack: 5},
	ItemStonePan:     {Name: "Chảo đá", Stack: 5},
	ItemFlask:        {Name: "Bình da", Stack: 10},
	ItemDirtyWater:   {Name: "Bình nước ao", Stack: 10},
	ItemCleanWater:   {Name: "Bình nước sạch", Health: 5, Stack: 10},
	ItemSeaWater:     {Name: "Bát nước biển", Stack: 20},
	ItemSalt:         {Name: "Muối"},
	ItemFlour:        {Name: "Bột"},
	ItemSaltedMeat:   {Name: "Thịt muối", Hunger: 20, Health: 2},
	ItemSmokedMeat:   {Name: "Thịt hun khói", Hunger: 42, Health: 8},
	ItemJerky:        {Name: "Thịt khô", Hunger: 30, Health: 4},
	ItemDriedFish:    {Name: "Cá khô", Hunger: 24, Health: 4},
	ItemStew:         {Name: "Canh thịt rau", Hunger: 40, Health: 10, Stack: 10},
	ItemHotSoup:      {Name: "Súp nóng", Hunger: 32, Health: 8, Stack: 10},
	ItemDinoStew:     {Name: "Thịt hầm khủng long", Hunger: 70, Health: 15, Stack: 10},
	ItemHerbTea:      {Name: "Trà thảo dược", Health: 12, Stack: 10},
	ItemHerbFish:     {Name: "Cá nướng thảo mộc", Hunger: 34, Health: 10},
	ItemSearedMeat:   {Name: "Thịt áp chảo", Hunger: 45, Health: 10},
	ItemFriedEgg:     {Name: "Trứng rán", Hunger: 25, Health: 8},
	ItemHerbMedicine: {Name: "Thuốc thảo dược", Health: 30, Stack: 10},
	ItemJuice:        {Name: "Nước ép quả", Hunger: 10, Health: 15, Stack: 10},
	ItemBread:        {Name: "Bánh mì", Hunger: 45, Health: 6},
	ItemFruitPie:     {Name: "Bánh quả", Hunger: 55, Health: 12},
}

// --- what goes in a dish -----------------------------------------------------

// FoodClass is a kind of ingredient a dish asks for: any meat, any herb.
type FoodClass uint8

const (
	ClassNone FoodClass = iota
	ClassMeat
	ClassFish
	ClassVeg
	ClassHerb
	ClassFruit
	ClassGrain
	ClassEgg
	NumFoodClasses
)

var foodClassNames = [NumFoodClasses]string{
	ClassMeat: "Thịt", ClassFish: "Cá, sò", ClassVeg: "Rau củ", ClassHerb: "Thảo mộc",
	ClassFruit: "Quả", ClassGrain: "Hạt", ClassEgg: "Trứng",
}

// Name is the class's name in a recipe.
func (c FoodClass) Name() string {
	if c >= NumFoodClasses {
		return ""
	}
	return foodClassNames[c]
}

// ============================================================================
// The ingredients of each class. Herbs, grain and more vegetables come from
// the plants of flora.go; lavender and daisies still do for herbs, and
// ginkgo nuts for grain. MERGE: the eggs of PLAN.md's "Trứng khủng long" go
// here; until they exist nothing is an egg (so nothing can be fried yet).
// ============================================================================
var foodClasses = [NumFoodClasses][]ItemKind{
	ClassMeat:  {ItemMeat, ItemCrabMeat, ItemSaltedMeat},
	ClassFish:  {ItemFish, ItemClam},
	ClassVeg:   {ItemPotato, ItemMushroom, ItemCactus, ItemSquash, ItemGourd, ItemSeaweed, ItemLotusSeed},
	ClassHerb:  {ItemHerb, ItemFlowerPurple, ItemFlowerWhite, ItemFlowerYellow},
	ClassFruit: {ItemBerry, ItemFig, ItemBanana, ItemCoconut},
	ClassGrain: {ItemRice, ItemCorn, ItemGinkgoNut},
	ClassEgg:   {},
}

// ClassItems lists the items of class c.
func ClassItems(c FoodClass) []ItemKind {
	if c >= NumFoodClasses {
		return nil
	}
	return slices.Clone(foodClasses[c])
}

// ClassOf is the class item k belongs to, ClassNone if none.
func ClassOf(k ItemKind) FoodClass {
	for c, items := range foodClasses {
		if slices.Contains(items, k) {
			return FoodClass(c)
		}
	}
	return ClassNone
}

// --- stations ------------------------------------------------------------------

// CookGear is what sits on a campfire: its Entity.Var.
type CookGear uint8

const (
	GearNone CookGear = iota
	GearPot
	GearGrill
	GearPan
	numGears
)

// GearItem is the item that is set on a fire as gear g.
func GearItem(g CookGear) ItemKind {
	switch g {
	case GearPot:
		return ItemClayPot
	case GearGrill:
		return ItemGrill
	case GearPan:
		return ItemStonePan
	}
	return ItemNone
}

// GearOf is the gear item k is, GearNone if it is none.
func GearOf(k ItemKind) CookGear {
	for g := GearPot; g < numGears; g++ {
		if GearItem(g) == k {
			return g
		}
	}
	return GearNone
}

// StationKind is a place where things are cooked.
type StationKind uint8

const (
	StationNone StationKind = iota
	StationPot
	StationGrill
	StationPan
	StationMortar
	StationOven
	StationRack
	NumStations
)

var stationNames = [NumStations]string{
	StationPot: "Nồi đất", StationGrill: "Giàn nướng", StationPan: "Chảo đá",
	StationMortar: "Cối giã", StationOven: "Lò nướng bánh", StationRack: "Giàn phơi",
}

// Name is the station's name.
func (s StationKind) Name() string {
	if s >= NumStations {
		return ""
	}
	return stationNames[s]
}

// StationOf is the station an entity of kind with Var v is, StationNone for
// anything else (a bare campfire included).
func StationOf(kind EntityKind, v uint8) StationKind {
	switch kind {
	case EntityCampfire:
		switch CookGear(v) {
		case GearPot:
			return StationPot
		case GearGrill:
			return StationGrill
		case GearPan:
			return StationPan
		}
	case EntityMortar:
		return StationMortar
	case EntityOven:
		return StationOven
	case EntityDryRack:
		return StationRack
	}
	return StationNone
}

// NeedsHeat reports whether station s only works over a burning fire.
func (s StationKind) NeedsHeat() bool {
	return s == StationPot || s == StationGrill || s == StationPan || s == StationOven
}

// Cooking tuning.
const (
	CookIn    = 4 // stacks of ingredients a station holds
	CookOut   = 4 // stacks of what came out
	CookSlots = CookIn + CookOut

	CookReach = InteractReach + 0.75 // how far a player may step from an open station

	GrillTicks  = 8 * TicksPerSecond   // one round on the grill: a piece of every stack
	RackTicks   = 120 * TicksPerSecond // one round on the drying rack
	RackSmoke   = 2                    // a rack in the light of a fire dries this much faster
	SaltReach   = 4                    // sea water dries to salt on a rack this near the sea, in tiles
	SaltYield   = 2
	OvenMaxFuel = 240 * TicksPerSecond
	OvenRadius  = 1.5 // tiles a lit oven lights and warms

	ClayYield       = 2
	ClayRegrowTicks = 2 * DayTicks
	clayByPond      = 0.07  // of the free ground beside a pond
	clayBySea       = 0.006 // of the beach beside the sea
	ScoopReach      = 2.0   // tiles to the water a bowl or flask is filled from

	MortarRefund = 3 // stone back
	OvenRefund   = 4 // stone back
	RackRefund   = 2 // wood back
)

// CookID identifies what a station is making.
type CookID uint8

const (
	CookNone  CookID = iota
	CookBatch        // the grill and the rack: a piece of every stack, each by itself
	CookDinoStew
	CookStew
	CookHotSoup
	CookHerbTea
	CookBoil
	CookHerbFish
	CookSear
	CookFryEgg
	CookMedicine
	CookFlour
	CookJuice
	CookPie
	CookBread
	NumCooks
)

// CookNeed is one ingredient of a dish: Count of Item, or of anything of
// Class when Item is ItemNone.
type CookNeed struct {
	Item  ItemKind
	Class FoodClass
	Count int
}

// Name is how the need reads in a recipe.
func (n CookNeed) Name() string {
	if n.Item.Valid() {
		return n.Item.Def().Name
	}
	return n.Class.Name()
}

// Takes reports whether item k fills the need.
func (n CookNeed) Takes(k ItemKind) bool {
	if n.Item.Valid() {
		return k == n.Item
	}
	return n.Class != ClassNone && ClassOf(k) == n.Class
}

// CookRecipe is a dish: what a station makes of several ingredients.
type CookRecipe struct {
	ID      CookID
	Station StationKind
	Needs   []CookNeed
	Gives   ItemKind
	Count   int
	Back    ItemKind // what comes back empty: the flask the water was in
	Ticks   int
	Salty   bool // a pinch of salt beside the ingredients makes one more
}

// cookRecipes are the dishes, the fullest first: a station makes the first
// one its ingredients allow.
var cookRecipes = []CookRecipe{
	{ID: CookDinoStew, Station: StationPot, Gives: ItemDinoStew, Count: 1, Ticks: 20 * TicksPerSecond,
		Needs: []CookNeed{{Class: ClassMeat, Count: 2}, {Item: ItemBone, Count: 1}, {Item: ItemSalt, Count: 1}, {Item: ItemClayBowl, Count: 1}}},
	{ID: CookStew, Station: StationPot, Gives: ItemStew, Count: 1, Ticks: 12 * TicksPerSecond, Salty: true,
		Needs: []CookNeed{{Class: ClassMeat, Count: 1}, {Class: ClassVeg, Count: 1}, {Item: ItemClayBowl, Count: 1}}},
	{ID: CookHotSoup, Station: StationPot, Gives: ItemHotSoup, Count: 1, Ticks: 12 * TicksPerSecond, Salty: true,
		Needs: []CookNeed{{Class: ClassFish, Count: 1}, {Class: ClassVeg, Count: 1}, {Item: ItemClayBowl, Count: 1}}},
	{ID: CookHerbTea, Station: StationPot, Gives: ItemHerbTea, Count: 1, Ticks: 8 * TicksPerSecond,
		Needs: []CookNeed{{Class: ClassHerb, Count: 2}, {Item: ItemDirtyWater, Count: 1}}},
	{ID: CookBoil, Station: StationPot, Gives: ItemCleanWater, Count: 1, Ticks: 6 * TicksPerSecond,
		Needs: []CookNeed{{Item: ItemDirtyWater, Count: 1}}},
	{ID: CookHerbFish, Station: StationGrill, Gives: ItemHerbFish, Count: 1, Ticks: GrillTicks,
		Needs: []CookNeed{{Item: ItemFish, Count: 1}, {Class: ClassHerb, Count: 1}}},
	{ID: CookSear, Station: StationPan, Gives: ItemSearedMeat, Count: 1, Ticks: 6 * TicksPerSecond,
		Needs: []CookNeed{{Class: ClassMeat, Count: 1}, {Item: ItemSalt, Count: 1}}},
	{ID: CookFryEgg, Station: StationPan, Gives: ItemFriedEgg, Count: 1, Ticks: 5 * TicksPerSecond, Salty: true,
		Needs: []CookNeed{{Class: ClassEgg, Count: 1}}},
	{ID: CookJuice, Station: StationMortar, Gives: ItemJuice, Count: 1, Ticks: 3 * TicksPerSecond,
		Needs: []CookNeed{{Class: ClassFruit, Count: 2}, {Item: ItemClayBowl, Count: 1}}},
	{ID: CookMedicine, Station: StationMortar, Gives: ItemHerbMedicine, Count: 1, Ticks: 3 * TicksPerSecond,
		Needs: []CookNeed{{Class: ClassHerb, Count: 2}}},
	{ID: CookFlour, Station: StationMortar, Gives: ItemFlour, Count: 1, Ticks: 3 * TicksPerSecond,
		Needs: []CookNeed{{Class: ClassGrain, Count: 2}}},
	{ID: CookPie, Station: StationOven, Gives: ItemFruitPie, Count: 1, Ticks: 18 * TicksPerSecond,
		Needs: []CookNeed{{Item: ItemFlour, Count: 2}, {Class: ClassFruit, Count: 1}}},
	{ID: CookBread, Station: StationOven, Gives: ItemBread, Count: 1, Ticks: 15 * TicksPerSecond, Salty: true,
		Needs: []CookNeed{{Item: ItemFlour, Count: 2}}},
}

// CookRecipes lists the dishes made at station s, in the order they are
// tried.
func CookRecipes(s StationKind) []CookRecipe {
	var out []CookRecipe
	for _, r := range cookRecipes {
		if r.Station == s {
			r.Needs = slices.Clone(r.Needs)
			out = append(out, r)
		}
	}
	return out
}

// CookRecipeByID is the dish with the given id.
func CookRecipeByID(id CookID) (CookRecipe, bool) {
	for _, r := range cookRecipes {
		if r.ID == id {
			r.Needs = slices.Clone(r.Needs)
			return r, true
		}
	}
	return CookRecipe{}, false
}

// CookConv is what a grill or a rack makes of one stack by itself.
type CookConv struct {
	N     int      // pieces of the stack that go in
	Gives ItemKind // what comes out
	Count int
	Back  ItemKind // the bowl the sea water was in
	Sea   bool     // only near the sea: salt
}

// rackConvs is what dries on the rack.
var rackConvs = map[ItemKind]CookConv{
	ItemMeat:       {N: 1, Gives: ItemJerky, Count: 1},
	ItemCrabMeat:   {N: 1, Gives: ItemJerky, Count: 1},
	ItemFish:       {N: 1, Gives: ItemDriedFish, Count: 1},
	ItemSaltedMeat: {N: 1, Gives: ItemSmokedMeat, Count: 1},
	ItemSeaWater:   {N: 1, Gives: ItemSalt, Count: SaltYield, Back: ItemClayBowl, Sea: true},
}

// ConvFor is what station s makes of a stack of k by itself: on the grill
// whatever is roasted by a fire from one kind of thing, on the rack what
// dries.
func ConvFor(s StationKind, k ItemKind) (CookConv, bool) {
	switch s {
	case StationGrill:
		for i := range recipes {
			r := &recipes[i]
			if r.NearFire && len(r.Inputs) == 1 && r.Inputs[0].Item == k && r.Gives.Valid() {
				return CookConv{N: r.Inputs[0].Count, Gives: r.Gives, Count: 1}, true
			}
		}
	case StationRack:
		c, ok := rackConvs[k]
		return c, ok
	}
	return CookConv{}, false
}

// StationAccepts reports whether item k is of any use at station s: an
// ingredient of one of its dishes, salt for them, or something it roasts or
// dries.
func StationAccepts(s StationKind, k ItemKind) bool {
	if !k.Valid() {
		return false
	}
	if _, ok := ConvFor(s, k); ok {
		return true
	}
	for _, r := range cookRecipes {
		if r.Station != s {
			continue
		}
		if r.Salty && k == ItemSalt {
			return true
		}
		for _, n := range r.Needs {
			if n.Takes(k) {
				return true
			}
		}
	}
	return false
}

// CookState says what a station is doing, for its window.
type CookState uint8

const (
	CookIdle    CookState = iota // nothing in it
	CookNoDish                   // what is in it makes nothing
	CookWorking                  // cooking
	CookCold                     // it would cook, but the fire is out
	CookFull                     // no room for what would come out
	CookNeedSea                  // sea water only dries to salt near the sea
)

// Entity.Cook bits: what everyone sees of a station.
const (
	CookFlagLoaded  = 1 << iota // ingredients in it
	CookFlagWorking             // cooking now
	CookFlagDone                // something waits to be taken out
	CookFlagFish                // a rack: fish hang on it
	CookFlagSalt                // a rack: bowls of brine stand on it
)

// Station is what a cooking station holds: the ingredients in the first
// CookIn slots, what came out in the rest.
type Station struct {
	Slots    [CookSlots]Slot
	Dish     CookID // what it is making, CookNone when nothing
	Progress int    // ticks of work done on it
	Total    int    // ticks it takes
	State    CookState
}

// CookSave is a station in a save.
type CookSave struct {
	ID       EntityID
	Slots    [CookSlots]Slot
	Progress int `json:",omitempty"`
}

// station is the contents of station entity id, made empty when first asked.
func (w *World) station(id EntityID) *Station {
	if w.stations == nil {
		w.stations = make(map[EntityID]*Station)
	}
	s := w.stations[id]
	if s == nil {
		s = &Station{}
		w.stations[id] = s
	}
	return s
}

// cookPlayer is a player's part in cooking.
type cookPlayer struct {
	open  EntityID      // the station open, 0 when none
	buffs [NumBuffs]int // ticks left of what a dish gave
}

// OpenCook is the station p has open, 0 when none.
func (p *Player) OpenCook() EntityID { return p.cook.open }

// CookInfo is everything a station's window shows.
type CookInfo struct {
	Station
	Kind StationKind
	Fuel int // ticks the fire under it (or in it) has left
}

// CookContents is what station id holds and does; ok is false when there is
// no such station.
func (w *World) CookContents(id EntityID) (CookInfo, bool) {
	e := w.entities[id]
	if e == nil {
		return CookInfo{}, false
	}
	k := StationOf(e.Kind, e.Var)
	if k == StationNone {
		return CookInfo{}, false
	}
	return CookInfo{Station: *w.station(id), Kind: k, Fuel: e.Fuel}, true
}

// --- the slots -------------------------------------------------------------------

// addTo puts up to n of k into slots, onto stacks of it first, and returns
// how many went in.
func addTo(slots []Slot, k ItemKind, n int) int {
	if !k.Valid() || n <= 0 {
		return 0
	}
	most, added := k.MaxCount(), 0
	for i := range slots {
		if s := &slots[i]; !s.Empty() && s.Item == k && s.Count < most {
			d := min(n-added, most-s.Count)
			s.Count += d
			added += d
		}
	}
	for i := range slots {
		if added == n {
			break
		}
		if s := &slots[i]; s.Empty() {
			d := min(n-added, most)
			*s = Slot{Item: k, Count: d}
			added += d
		}
	}
	return added
}

// takeFrom removes n of what need takes from slots; it reports whether
// there was enough (nothing is removed otherwise).
func takeFrom(slots []Slot, need CookNeed) bool {
	have := 0
	for _, s := range slots {
		if !s.Empty() && need.Takes(s.Item) {
			have += s.Count
		}
	}
	if have < need.Count {
		return false
	}
	left := need.Count
	for i := range slots {
		if s := &slots[i]; left > 0 && !s.Empty() && need.Takes(s.Item) {
			d := min(left, s.Count)
			s.Count -= d
			left -= d
			if s.Count == 0 {
				*s = Slot{}
			}
		}
	}
	return true
}

// cookOutput is what one go of recipe r gives from the ingredients in: its
// dish (one more with a spare pinch of salt), what comes back empty, and
// the ingredients left. ok is false when the ingredients are not all there.
func cookOutput(r *CookRecipe, in [CookIn]Slot) (left [CookIn]Slot, out []Ingredient, ok bool) {
	for _, n := range r.Needs {
		if !takeFrom(in[:], n) {
			return in, nil, false
		}
	}
	n := r.Count
	if r.Salty && takeFrom(in[:], CookNeed{Item: ItemSalt, Count: 1}) {
		n++
	}
	out = []Ingredient{{Item: r.Gives, Count: n}}
	if r.Back.Valid() {
		out = append(out, Ingredient{Item: r.Back, Count: 1})
	}
	return in, out, true
}

// fits reports whether everything in items goes into slots, and puts it
// there if so.
func fits(slots *[CookOut]Slot, items []Ingredient) bool {
	try := *slots
	for _, it := range items {
		if addTo(try[:], it.Item, it.Count) != it.Count {
			return false
		}
	}
	*slots = try
	return true
}

// --- a station's tick ------------------------------------------------------------

// stepStations runs every station that has something in it.
func (w *World) stepStations() {
	for id, s := range w.stations {
		e := w.entities[id]
		if e == nil || StationOf(e.Kind, e.Var) == StationNone {
			delete(w.stations, id) // deleting during range is safe in Go
			continue
		}
		w.stepStation(e, s)
	}
}

// nearSea reports whether tile (tx, ty) is within SaltReach of the sea.
func (w *World) nearSea(tx, ty int) bool {
	for y := ty - SaltReach; y <= ty+SaltReach; y++ {
		for x := tx - SaltReach; x <= tx+SaltReach; x++ {
			if w.inBounds(x, y) && w.tiles[y*w.width+x].Sea() && !w.freshWater(x, y) {
				return true
			}
		}
	}
	return false
}

// freshWater reports whether tile (tx, ty) is the water of a pond or lake,
// not the sea.
func (w *World) freshWater(tx, ty int) bool {
	if !w.inBounds(tx, ty) {
		return false
	}
	switch i := ty*w.width + tx; {
	case w.tiles[i] == TileRiver || w.tiles[i] == TileFord || w.tiles[i] == TileWaterfall:
		return true // a river: see rivers.go
	case w.tiles[i] != TileWater:
		return false
	case w.relief != nil && w.relief[i].Variant() == VarLake:
		return true
	}
	if !w.wx.pondDone {
		w.wx.pond = PondMask(w.width, w.height, w.tiles)
		w.wx.pondDone = true
	}
	return w.wx.pond[ty*w.width+tx]
}

func (w *World) stepStation(e *Entity, s *Station) {
	kind := StationOf(e.Kind, e.Var)
	var in [CookIn]Slot
	copy(in[:], s.Slots[:CookIn])
	var out [CookOut]Slot
	copy(out[:], s.Slots[CookIn:])

	loaded := false
	for _, sl := range in {
		loaded = loaded || !sl.Empty()
	}
	e.Cook = 0
	if loaded {
		e.Cook |= CookFlagLoaded
	}
	for _, sl := range out {
		if !sl.Empty() {
			e.Cook |= CookFlagDone
		}
	}
	if kind == StationRack {
		for _, sl := range in {
			if sl.Empty() {
				continue
			}
			switch sl.Item {
			case ItemFish:
				e.Cook |= CookFlagFish
			case ItemSeaWater:
				e.Cook |= CookFlagSalt
			}
			break // the first stack says what hangs there
		}
	}

	// What it would make now.
	dish, total, state := CookNone, 0, CookIdle
	var recipe *CookRecipe
	var left [CookIn]Slot
	if loaded {
		state = CookNoDish
		for i := range cookRecipes {
			r := &cookRecipes[i]
			if r.Station != kind {
				continue
			}
			if l, g, ok := cookOutput(r, in); ok {
				recipe, left = r, l
				dish, total, state = r.ID, r.Ticks, CookWorking
				if !fits(&out, g) {
					state = CookFull
				}
				break
			}
		}
		if recipe == nil {
			dish, total, state = w.planBatch(e, kind, &in, &out)
		}
	}
	if state == CookWorking && kind.NeedsHeat() && e.Fuel <= 0 {
		state = CookCold
	}
	if dish != s.Dish {
		s.Dish, s.Progress = dish, 0
	}
	s.Total, s.State = total, state
	if state != CookWorking {
		return
	}
	e.Cook |= CookFlagWorking
	s.Progress++
	if kind == StationRack && w.nearFire(float64(e.TX)+0.5, float64(e.TY)+0.5) {
		s.Progress += RackSmoke - 1 // smoked over a fire
	}
	if s.Progress < total {
		return
	}
	s.Progress = 0
	if recipe != nil {
		copy(s.Slots[:CookIn], left[:])
	} else {
		copy(s.Slots[:CookIn], in[:]) // planBatch took the pieces
	}
	copy(s.Slots[CookIn:], out[:])
}

// planBatch is a grill's or a rack's round: a piece of every stack that
// roasts or dries, each by itself. It takes the pieces out of in and puts
// what they become in out (the caller keeps both only when the round is
// done) and says what the station does meanwhile.
func (w *World) planBatch(e *Entity, kind StationKind, in *[CookIn]Slot, out *[CookOut]Slot) (CookID, int, CookState) {
	total := GrillTicks
	if kind == StationRack {
		total = RackTicks
	}
	any, full, dry := false, false, false
	sea := -1 // not looked yet
	for i := range in {
		sl := &in[i]
		if sl.Empty() {
			continue
		}
		c, ok := ConvFor(kind, sl.Item)
		if !ok || sl.Count < c.N {
			continue
		}
		if c.Sea {
			if sea < 0 {
				sea = 0
				if w.nearSea(e.TX, e.TY) {
					sea = 1
				}
			}
			if sea == 0 {
				dry = true
				continue
			}
		}
		gives := []Ingredient{{Item: c.Gives, Count: c.Count}}
		if c.Back.Valid() {
			gives = append(gives, Ingredient{Item: c.Back, Count: 1})
		}
		if !fits(out, gives) {
			full = true
			continue
		}
		if sl.Count -= c.N; sl.Count == 0 {
			*sl = Slot{}
		}
		any = true
	}
	switch {
	case any:
		return CookBatch, total, CookWorking
	case full:
		return CookBatch, total, CookFull
	case dry:
		return CookBatch, total, CookNeedSea
	}
	return CookNone, 0, CookNoDish
}

// --- the player at a station ---------------------------------------------------

// isStation reports whether e is somewhere to cook.
func isStation(e *Entity) bool { return e != nil && StationOf(e.Kind, e.Var) != StationNone }

// openCook opens station e for p. It is no swing, so it reports false.
func (w *World) openCook(p *Player, e *Entity) bool {
	w.station(e.ID)
	p.cook.open, p.chest = e.ID, 0
	return false
}

// keepCook closes p's station when it is gone or p has walked off from it.
func (w *World) keepCook(p *Player) {
	if p.cook.open == 0 {
		return
	}
	e := w.entities[p.cook.open]
	if p.Dead || p.Asleep || !isStation(e) || p.chest != 0 ||
		math.Hypot(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y) > CookReach {
		p.cook.open = 0
	}
}

// cookCommand moves a stack between p's inventory and p's open station,
// stokes its fire, closes it or takes it down.
func (w *World) cookCommand(p *Player, cmd Command) {
	w.keepCook(p)
	e := w.entities[p.cook.open]
	if p.cook.open == 0 || e == nil {
		return // closed meanwhile: the click is stale
	}
	s := w.station(e.ID)
	kind := StationOf(e.Kind, e.Var)
	i := int(cmd.Slot)
	switch cmd.Kind {
	case CmdCookPut:
		if !ValidSlot(i) || p.Inv[i].Empty() {
			return
		}
		sl := p.Inv[i]
		if !StationAccepts(kind, sl.Item) {
			p.notify(NoticeCantCook, uint8(sl.Item))
			return
		}
		n := sl.Count
		if cmd.Count > 0 {
			n = min(n, int(cmd.Count))
		}
		if moved := addTo(s.Slots[:CookIn], sl.Item, n); moved > 0 {
			p.Inv.TakeSlot(i, moved)
		} else {
			p.notify(NoticeCookFull, uint8(sl.Item))
		}
	case CmdCookTake:
		if i < 0 || i >= CookSlots || s.Slots[i].Empty() {
			return
		}
		sl := &s.Slots[i]
		if moved := p.Inv.Add(sl.Item, sl.Count); moved > 0 {
			if sl.Count -= moved; sl.Count == 0 {
				*sl = Slot{}
			}
		} else {
			p.notify(NoticeBagFull, uint8(sl.Item))
		}
	case CmdCookClose:
		p.cook.open = 0
	case CmdCookStoke:
		switch e.Kind {
		case EntityCampfire:
			w.stoke(p, e, FireMaxFuel)
		case EntityOven:
			w.stoke(p, e, OvenMaxFuel)
		}
	case CmdCookTakeDown:
		if s.Slots != ([CookSlots]Slot{}) {
			p.notify(NoticeCookNotEmpty, 0)
			return
		}
		back := stationRefund(e)
		if p.Inv.Room(back.Item) < back.Count {
			p.notify(NoticeBagFull, uint8(back.Item))
			return
		}
		p.Inv.Add(back.Item, back.Count)
		p.cook.open = 0
		delete(w.stations, e.ID)
		if e.Kind == EntityCampfire {
			e.Var, e.Cook = 0, 0
			if e.Fuel <= 0 {
				w.removeEntity(e) // a dead fire with nothing on it is gone
			}
			return
		}
		w.removeEntity(e)
	}
}

// stationRefund is what taking station e down gives back.
func stationRefund(e *Entity) Ingredient {
	switch e.Kind {
	case EntityCampfire:
		return Ingredient{Item: GearItem(CookGear(e.Var)), Count: 1}
	case EntityMortar:
		return Ingredient{Item: ItemStone, Count: MortarRefund}
	case EntityOven:
		return Ingredient{Item: ItemStone, Count: OvenRefund}
	}
	return Ingredient{Item: ItemWood, Count: RackRefund}
}

// spillStation lets what station e holds, and the gear on a fire, fall on
// the ground: e is going away.
func (w *World) spillStation(e *Entity) {
	var items []Ingredient
	if e.Kind == EntityCampfire && GearItem(CookGear(e.Var)).Valid() {
		items = append(items, Ingredient{Item: GearItem(CookGear(e.Var)), Count: 1})
	}
	if s := w.stations[e.ID]; s != nil {
		for _, sl := range s.Slots {
			if !sl.Empty() {
				items = append(items, Ingredient{Item: sl.Item, Count: sl.Count})
			}
		}
		delete(w.stations, e.ID)
	}
	if len(items) > 0 {
		w.dropAt(float64(e.TX)+0.5, float64(e.TY)+0.5, items)
	}
}

// useCookItem is V with a piece of cooking gear in hand: a pot, grill or
// pan goes on a campfire, a bowl or flask is filled at the water. It
// reports whether held was such a thing.
func (w *World) useCookItem(p *Player, held ItemKind, at bool, atX, atY int) bool {
	if g := GearOf(held); g != GearNone {
		w.mountGear(p, g, at, atX, atY)
		return true
	}
	if held == ItemClayBowl || held == ItemFlask {
		w.scoop(p, held, at, atX, atY)
		return true
	}
	return false
}

// mountGear sets gear g on the bare campfire on tile (atX, atY) when at is
// set, else on the nearest one in reach.
func (w *World) mountGear(p *Player, g CookGear, at bool, atX, atY int) {
	var fire *Entity
	bare := func(e *Entity) bool { return e != nil && e.Kind == EntityCampfire && e.Var == 0 }
	if at {
		if e := w.entityAt(atX, atY); bare(e) && math.Hypot(float64(atX)+0.5-p.X, float64(atY)+0.5-p.Y) <= PlaceReach {
			fire = e
		}
	}
	if fire == nil {
		best := PlaceReach
		cx, cy := int(math.Floor(p.X)), int(math.Floor(p.Y))
		for ty := cy - 3; ty <= cy+3; ty++ {
			for tx := cx - 3; tx <= cx+3; tx++ {
				if e := w.entityAt(tx, ty); bare(e) {
					if d := math.Hypot(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y); d <= best {
						fire, best = e, d
					}
				}
			}
		}
	}
	if fire == nil {
		p.notify(NoticeNeedHearth, uint8(GearItem(g)))
		return
	}
	p.Inv.TakeSlot(p.Selected, 1)
	fire.Var = uint8(g)
	w.station(fire.ID)
	p.Dir = dirToward(float64(fire.TX)+0.5-p.X, float64(fire.TY)+0.5-p.Y)
	p.actedAt = w.tick
}

// scoop fills the bowl or flask p holds at the nearest water in reach (the
// tile chosen, when at is set): a bowl takes sea water, a flask fresh.
func (w *World) scoop(p *Player, held ItemKind, at bool, atX, atY int) {
	wet := func(tx, ty int) bool {
		return w.inBounds(tx, ty) && w.tiles[ty*w.width+tx].Swimmable() && !w.frozenAt(tx, ty)
	}
	tx, ty, found := 0, 0, false
	if at && wet(atX, atY) && math.Hypot(float64(atX)+0.5-p.X, float64(atY)+0.5-p.Y) <= PlaceReach {
		tx, ty, found = atX, atY, true
	}
	if !found {
		best := ScoopReach
		cx, cy := int(math.Floor(p.X)), int(math.Floor(p.Y))
		for y := cy - 2; y <= cy+2; y++ {
			for x := cx - 2; x <= cx+2; x++ {
				if !wet(x, y) {
					continue
				}
				if d := math.Hypot(float64(x)+0.5-p.X, float64(y)+0.5-p.Y); d < best {
					tx, ty, best, found = x, y, d, true
				}
			}
		}
	}
	if !found {
		p.notify(NoticeNoWater, uint8(held))
		return
	}
	fresh := w.freshWater(tx, ty)
	var full ItemKind
	switch {
	case held == ItemClayBowl && !fresh:
		full = ItemSeaWater
	case held == ItemFlask && fresh:
		full = ItemDirtyWater
	default:
		p.notify(NoticeWrongWater, uint8(held)) // a bowl is for brine, a flask for fresh water
		return
	}
	after := p.Inv
	after.TakeSlot(p.Selected, 1)
	if after.Add(full, 1) == 0 {
		p.notify(NoticeBagFull, uint8(full))
		return
	}
	p.Inv = after
	p.Dir = dirToward(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y)
	p.actedAt = w.tick
	p.notify(NoticeScooped, uint8(full))
}

// --- buffs -------------------------------------------------------------------------

// BuffKind is a passing good a dish does.
type BuffKind uint8

const (
	BuffRegen BuffKind = iota // health comes back fast
	BuffWarm                  // the cold does not bite
	BuffSwift                 // walks faster
	BuffMight                 // blows land harder
	BuffSated                 // hunger grows slowly
	NumBuffs
)

// Buff tuning.
const (
	BuffRegenPerTick = 1.0 / TicksPerSecond // health a second while BuffRegen lasts
	BuffSwiftSpeed   = 1.25
	BuffMightDamage  = 1.3
	BuffSatedHunger  = 0.4
)

var buffNames = [NumBuffs]string{"Hồi máu", "Ấm bụng", "Nhanh chân", "Sung sức", "No lâu"}

// Name is the buff's name on its chip.
func (b BuffKind) Name() string {
	if b >= NumBuffs {
		return ""
	}
	return buffNames[b]
}

// Buff is a buff a dish gives, and for how long.
type Buff struct {
	Kind  BuffKind
	Ticks int
}

// dishFx is what eating a dish does beyond filling and mending.
type dishFx struct {
	buffs []Buff
	cure  bool     // draws poison out
	back  ItemKind // the bowl or flask it was in
}

var dishes = map[ItemKind]dishFx{
	ItemStew:         {buffs: []Buff{{BuffRegen, 60 * TicksPerSecond}}, back: ItemClayBowl},
	ItemHotSoup:      {buffs: []Buff{{BuffWarm, 240 * TicksPerSecond}}, back: ItemClayBowl},
	ItemDinoStew:     {buffs: []Buff{{BuffSated, 480 * TicksPerSecond}, {BuffMight, 240 * TicksPerSecond}}, back: ItemClayBowl},
	ItemHerbTea:      {buffs: []Buff{{BuffRegen, 30 * TicksPerSecond}}, cure: true, back: ItemFlask},
	ItemHerbFish:     {buffs: []Buff{{BuffSwift, 120 * TicksPerSecond}}},
	ItemSearedMeat:   {buffs: []Buff{{BuffMight, 90 * TicksPerSecond}}},
	ItemFruitPie:     {buffs: []Buff{{BuffRegen, 45 * TicksPerSecond}}},
	ItemHerbMedicine: {cure: true},
	ItemJuice:        {back: ItemClayBowl},
	ItemCleanWater:   {back: ItemFlask},
}

// DishBuffs lists the buffs eating item k gives.
func DishBuffs(k ItemKind) []Buff { return slices.Clone(dishes[k].buffs) }

// DishCures reports whether eating k draws poison out.
func DishCures(k ItemKind) bool { return dishes[k].cure }

// Buff is how many ticks of buff b p has left.
func (p *Player) Buff(b BuffKind) int {
	if b >= NumBuffs {
		return 0
	}
	return p.cook.buffs[b]
}

// Buffs is the ticks left of every buff.
func (p *Player) Buffs() [NumBuffs]int { return p.cook.buffs }

// ate is the rest of eating k: its buffs start (or start over, never
// shortened), poison goes, and the bowl comes back.
func (w *World) ate(p *Player, k ItemKind) {
	fx, ok := dishes[k]
	if !ok {
		return
	}
	for _, b := range fx.buffs {
		p.cook.buffs[b.Kind] = max(p.cook.buffs[b.Kind], b.Ticks)
	}
	if fx.cure {
		p.cure()
	}
	if fx.back.Valid() && p.Inv.Add(fx.back, 1) == 0 {
		w.dropAt(p.X, p.Y, []Ingredient{{Item: fx.back, Count: 1}})
	}
}

// stepBuffs runs p's buffs down a tick: BuffRegen heals, BuffWarm keeps
// the cold out.
func (p *Player) stepBuffs() {
	if p.cook.buffs[BuffWarm] > 0 {
		p.Cold = false
	}
	if p.cook.buffs[BuffRegen] > 0 && !p.Dead {
		p.Health = min(MaxHealth, p.Health+BuffRegenPerTick)
	}
	for i := range p.cook.buffs {
		if p.cook.buffs[i] > 0 {
			p.cook.buffs[i]--
		}
	}
}

// buffSpeed is how much faster p walks for what they ate.
func (p *Player) buffSpeed() float64 {
	if p.cook.buffs[BuffSwift] > 0 {
		return BuffSwiftSpeed
	}
	return 1
}

// buffHunger is how much slower p's hunger grows for what they ate.
func (p *Player) buffHunger() float64 {
	if p.cook.buffs[BuffSated] > 0 {
		return BuffSatedHunger
	}
	return 1
}

// buffDamage is how much harder p's blows land for what they ate.
func (p *Player) buffDamage() float64 {
	if p.cook.buffs[BuffMight] > 0 {
		return BuffMightDamage
	}
	return 1
}

// --- clay ----------------------------------------------------------------------------

// scatterClay sets clay beds on the ground beside the ponds, and a few on
// the beaches. It has its own stream, so it moves nothing else.
func (w *World) scatterClay(r *rand.Rand) {
	top := w.height
	if w.surfaceH > 0 {
		top = w.surfaceH // not down in the caves
	}
	for ty := 0; ty < top; ty++ {
		for tx := 0; tx < w.width; tx++ {
			roll := r.Float64()
			if roll >= clayByPond || !w.canPlace(tx, ty) {
				continue
			}
			t := w.tiles[ty*w.width+tx]
			if t == TilePath || t == TileCave || t == TileSnow {
				continue
			}
			pond, sea := false, false
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if !w.TileAt(tx+dx, ty+dy).Swimmable() || !w.inBounds(tx+dx, ty+dy) {
						continue
					}
					if w.freshWater(tx+dx, ty+dy) {
						pond = true
					} else {
						sea = true
					}
				}
			}
			if pond || (sea && t == TileSand && roll < clayBySea) {
				w.addEntity(EntityClay, tx, ty)
			}
		}
	}
}

// --- saving ------------------------------------------------------------------------

func saveBuffs(p *Player) []int {
	if p.cook.buffs == ([NumBuffs]int{}) {
		return nil
	}
	return slices.Clone(p.cook.buffs[:])
}

func loadBuffs(p *Player, saved []int) {
	for i := range p.cook.buffs {
		if i < len(saved) {
			p.cook.buffs[i] = min(max(0, saved[i]), DayTicks)
		}
	}
}

func (w *World) saveStations() []CookSave {
	var out []CookSave
	for id, s := range w.stations {
		if s.Slots != ([CookSlots]Slot{}) {
			out = append(out, CookSave{ID: id, Slots: s.Slots, Progress: s.Progress})
		}
	}
	slices.SortFunc(out, func(a, b CookSave) int { return int(a.ID) - int(b.ID) })
	return out
}

func (w *World) loadStations(saved []CookSave) {
	for _, ss := range saved {
		if !isStation(w.entities[ss.ID]) {
			continue
		}
		s := w.station(ss.ID)
		for i, sl := range ss.Slots {
			if !sl.Empty() {
				s.Slots[i] = Slot{Item: sl.Item, Count: min(sl.Count, sl.Item.MaxCount())}
			}
		}
		s.Progress = max(0, ss.Progress)
	}
}

func init() {
	if itemsEnd > cookItemBase || cookItemsEnd > ItemHeadBase {
		panic("sim: the cooking items collide with their neighbours")
	}
	for k, d := range cookItemDefs {
		itemDefs[k] = d
	}
	recipes[RecipeClayPot] = Recipe{ID: RecipeClayPot, Name: "Nồi đất",
		Inputs: []Ingredient{{Item: ItemClay, Count: 4}}, Gives: ItemClayPot, NearFurnace: true}
	recipes[RecipeClayBowl] = Recipe{ID: RecipeClayBowl, Name: "Bát đất",
		Inputs: []Ingredient{{Item: ItemClay, Count: 1}}, Gives: ItemClayBowl, NearFurnace: true}
	recipes[RecipeGrill] = Recipe{ID: RecipeGrill, Name: "Giàn nướng",
		Inputs: []Ingredient{{Item: ItemStick, Count: 6}, {Item: ItemGrass, Count: 2}}, Gives: ItemGrill}
	recipes[RecipeStonePan] = Recipe{ID: RecipeStonePan, Name: "Chảo đá",
		Inputs: []Ingredient{{Item: ItemStone, Count: 3}, {Item: ItemStick, Count: 1}}, Gives: ItemStonePan}
	recipes[RecipeMortar] = Recipe{ID: RecipeMortar, Name: "Cối giã",
		Inputs: []Ingredient{{Item: ItemStone, Count: 4}, {Item: ItemStick, Count: 1}}, Places: EntityMortar}
	recipes[RecipeOven] = Recipe{ID: RecipeOven, Name: "Lò nướng bánh",
		Inputs: []Ingredient{{Item: ItemStone, Count: 6}, {Item: ItemClay, Count: 4}}, Places: EntityOven}
	recipes[RecipeDryRack] = Recipe{ID: RecipeDryRack, Name: "Giàn phơi",
		Inputs: []Ingredient{{Item: ItemWood, Count: 2}, {Item: ItemStick, Count: 6}, {Item: ItemGrass, Count: 3}}, Places: EntityDryRack}
	recipes[RecipeFlask] = Recipe{ID: RecipeFlask, Name: "Bình da",
		Inputs: []Ingredient{{Item: ItemHide, Count: 2}, {Item: ItemGrass, Count: 1}}, Gives: ItemFlask}
	recipes[RecipeSaltMeat] = Recipe{ID: RecipeSaltMeat, Name: "Thịt muối",
		Inputs: []Ingredient{{Item: ItemMeat, Count: 1}, {Item: ItemSalt, Count: 1}}, Gives: ItemSaltedMeat}
}
