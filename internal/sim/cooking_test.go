package sim

import (
	"encoding/json"
	"testing"
)

// cookAt sets a campfire with gear g on it beside the player of a flat
// world and opens it.
func cookAt(t *testing.T, g CookGear) (*World, PlayerID, *Entity) {
	t.Helper()
	w := flatWorld()
	id := w.AddPlayer("a")
	e := w.addEntity(EntityCampfire, 17, 16)
	e.Var = uint8(g)
	w.openCook(w.players[id], e)
	return w, id, e
}

// built sets a built station beside the player and opens it.
func built(t *testing.T, kind EntityKind) (*World, PlayerID, *Entity) {
	t.Helper()
	w := flatWorld()
	id := w.AddPlayer("a")
	e := w.addEntity(kind, 17, 16)
	w.openCook(w.players[id], e)
	return w, id, e
}

// load gives the player items and puts them all into the open station.
func load(t *testing.T, w *World, id PlayerID, items ...Ingredient) {
	t.Helper()
	p := w.players[id]
	p.Inv = InventoryWith(items...)
	for i := range items {
		w.Apply(id, Command{Kind: CmdCookPut, Slot: uint8(i)})
	}
	if p.Inv != (Inventory{}) {
		t.Fatalf("not everything went in: %v (notice %v)", p.Inv[:len(items)], p.Notice)
	}
}

func outCount(w *World, e *Entity, k ItemKind) int {
	n := 0
	for _, s := range w.station(e.ID).Slots[CookIn:] {
		if !s.Empty() && s.Item == k {
			n += s.Count
		}
	}
	return n
}

func TestPotStewsAndBuffs(t *testing.T) {
	w, id, e := cookAt(t, GearPot)
	load(t, w, id, Ingredient{ItemMeat, 1}, Ingredient{ItemPotato, 1}, Ingredient{ItemClayBowl, 1})
	r, _ := CookRecipeByID(CookStew)
	stepN(w, r.Ticks-1)
	if n := outCount(w, e, ItemStew); n != 0 {
		t.Fatalf("stew ready a tick early: %d", n)
	}
	if info, _ := w.CookContents(e.ID); info.State != CookWorking || info.Dish != CookStew || e.Cook&CookFlagWorking == 0 {
		t.Fatalf("while cooking: state %v dish %v flags %b", info.State, info.Dish, e.Cook)
	}
	stepN(w, 1)
	if n := outCount(w, e, ItemStew); n != 1 {
		t.Fatalf("stew: %d, want 1", n)
	}
	if s := w.station(e.ID); s.Slots[0] != (Slot{}) || s.Slots[1] != (Slot{}) || s.Slots[2] != (Slot{}) {
		t.Fatalf("ingredients left: %v", s.Slots[:CookIn])
	}
	w.Apply(id, Command{Kind: CmdCookTake, Slot: CookIn})
	p := w.players[id]
	if p.Inv.Count(ItemStew) != 1 {
		t.Fatalf("took %d stew", p.Inv.Count(ItemStew))
	}
	p.Health = 50
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Buff(BuffRegen) != 60*TicksPerSecond {
		t.Fatalf("regen for %d ticks, want %d", p.Buff(BuffRegen), 60*TicksPerSecond)
	}
	if p.Inv.Count(ItemClayBowl) != 1 {
		t.Fatalf("the bowl did not come back")
	}
	h := p.Health
	stepN(w, 10*TicksPerSecond)
	if got := p.Health - h; got < 9.9 {
		t.Fatalf("healed %.2f in 10 s, want about 10", got)
	}
	stepN(w, 51*TicksPerSecond)
	if p.Buff(BuffRegen) != 0 {
		t.Fatalf("regen still on after 61 s: %d", p.Buff(BuffRegen))
	}
}

func TestSaltMakesOneMore(t *testing.T) {
	w, id, e := cookAt(t, GearPot)
	load(t, w, id, Ingredient{ItemFish, 1}, Ingredient{ItemMushroom, 1}, Ingredient{ItemClayBowl, 1}, Ingredient{ItemSalt, 1})
	stepN(w, 12*TicksPerSecond)
	if n := outCount(w, e, ItemHotSoup); n != 2 {
		t.Fatalf("salted soup: %d, want 2", n)
	}
}

func TestDinoStewBeatsStew(t *testing.T) {
	w, id, e := cookAt(t, GearPot)
	load(t, w, id, Ingredient{ItemMeat, 2}, Ingredient{ItemBone, 1}, Ingredient{ItemSalt, 1}, Ingredient{ItemClayBowl, 1})
	stepN(w, 20*TicksPerSecond)
	if n := outCount(w, e, ItemDinoStew); n != 1 {
		t.Fatalf("dino stew: %d, want 1 (slots %v)", n, w.station(e.ID).Slots)
	}
	p := w.players[id]
	p.Inv = InventoryWith(Ingredient{ItemDinoStew, 1})
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Buff(BuffSated) != 480*TicksPerSecond || p.Buff(BuffMight) != 240*TicksPerSecond {
		t.Fatalf("sated %d might %d", p.Buff(BuffSated), p.Buff(BuffMight))
	}
	// No hunger lost to speak of while sated.
	p.Hunger = 80
	stepN(w, 60*TicksPerSecond)
	lost := 80 - p.Hunger
	want := HungerPerTick * 60 * TicksPerSecond * BuffSatedHunger
	if lost < want*0.99 || lost > want*1.01 {
		t.Fatalf("lost %.3f hunger in a minute sated, want %.3f", lost, want)
	}
	if p.buffDamage() != BuffMightDamage {
		t.Fatalf("might: blows ×%.2f", p.buffDamage())
	}
}

func TestColdHearthKeepsAndWaits(t *testing.T) {
	w, id, e := cookAt(t, GearPot)
	e.Fuel = 5
	load(t, w, id, Ingredient{ItemMeat, 1}, Ingredient{ItemPotato, 1}, Ingredient{ItemClayBowl, 1})
	stepN(w, 60*TicksPerSecond)
	if w.entities[e.ID] == nil || e.Fuel != 0 {
		t.Fatalf("the hearth under a pot went away or burns: %v fuel %d", w.entities[e.ID], e.Fuel)
	}
	if info, _ := w.CookContents(e.ID); info.State != CookCold || outCount(w, e, ItemStew) != 0 {
		t.Fatalf("cold fire: state %v, stew %d", info.State, outCount(w, e, ItemStew))
	}
	w.players[id].Inv.Add(ItemWood, 1)
	w.Apply(id, Command{Kind: CmdCookStoke})
	if e.Fuel != FireFuelPerWood {
		t.Fatalf("stoked to %d", e.Fuel)
	}
	stepN(w, 12*TicksPerSecond+1)
	if outCount(w, e, ItemStew) != 1 {
		t.Fatalf("no stew after stoking")
	}
	// A bare fire still burns out and goes.
	bare := w.addEntity(EntityCampfire, 20, 20)
	bare.Fuel = 3
	stepN(w, 5)
	if w.entities[bare.ID] != nil {
		t.Fatalf("a bare fire stayed when out")
	}
}

func TestGrillRoastsFourAtOnce(t *testing.T) {
	w, id, e := cookAt(t, GearGrill)
	load(t, w, id, Ingredient{ItemMeat, 2}, Ingredient{ItemFish, 1}, Ingredient{ItemPotato, 1}, Ingredient{ItemClam, 2})
	stepN(w, GrillTicks)
	for _, k := range []ItemKind{ItemCookedMeat, ItemCookedFish, ItemRoastPotato, ItemRoastClam} {
		if outCount(w, e, k) != 1 {
			t.Fatalf("after one round: %d %s, want 1 (slots %v)", outCount(w, e, k), k.Def().Name, w.station(e.ID).Slots)
		}
	}
	stepN(w, GrillTicks)
	if outCount(w, e, ItemCookedMeat) != 2 || outCount(w, e, ItemCookedFish) != 1 {
		t.Fatalf("after two rounds: meat %d fish %d", outCount(w, e, ItemCookedMeat), outCount(w, e, ItemCookedFish))
	}
}

func TestHerbFishOnTheGrill(t *testing.T) {
	w, id, e := cookAt(t, GearGrill)
	load(t, w, id, Ingredient{ItemFish, 1}, Ingredient{ItemFlowerPurple, 1})
	stepN(w, GrillTicks)
	if outCount(w, e, ItemHerbFish) != 1 {
		t.Fatalf("no herb fish: %v", w.station(e.ID).Slots)
	}
	p := w.players[id]
	p.Inv = InventoryWith(Ingredient{ItemHerbFish, 1})
	w.Apply(id, Command{Kind: CmdCookClose})
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Buff(BuffSwift) != 120*TicksPerSecond {
		t.Fatalf("swift for %d", p.Buff(BuffSwift))
	}
	x := p.X
	w.SetInput(id, Input{MoveX: -1})
	stepN(w, 20)
	if got, want := x-p.X, 20*PlayerSpeed*BuffSwiftSpeed; got < want*0.999 || got > want*1.001 {
		t.Fatalf("walked %.3f in a second, want %.3f", got, want)
	}
}

func TestHotSoupKeepsTheColdOut(t *testing.T) {
	w := flatWorld()
	w.tick = deepNight + 3*SeasonDays*uint64(DayTicks) // a winter night
	if !w.winterNight() {
		t.Skip("no winter night at this tick")
	}
	id := w.AddPlayer("a")
	p := w.players[id]
	stepN(w, 2)
	if !p.Cold {
		t.Fatalf("not cold on a winter night")
	}
	p.Inv = InventoryWith(Ingredient{ItemHotSoup, 1})
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Buff(BuffWarm) != 240*TicksPerSecond {
		t.Fatalf("warm for %d", p.Buff(BuffWarm))
	}
	stepN(w, 2)
	if p.Cold {
		t.Fatalf("cold with hot soup inside")
	}
}

func TestRackDriesAndSmokes(t *testing.T) {
	w, id, e := built(t, EntityDryRack)
	load(t, w, id, Ingredient{ItemMeat, 1}, Ingredient{ItemFish, 1}, Ingredient{ItemSaltedMeat, 1})
	stepN(w, RackTicks-1)
	if outCount(w, e, ItemJerky) != 0 {
		t.Fatalf("jerky early")
	}
	stepN(w, 1)
	if outCount(w, e, ItemJerky) != 1 || outCount(w, e, ItemDriedFish) != 1 || outCount(w, e, ItemSmokedMeat) != 1 {
		t.Fatalf("rack gave %v", w.station(e.ID).Slots[CookIn:])
	}
	// By a fire it takes half the time.
	w2, id2, e2 := built(t, EntityDryRack)
	w2.addEntity(EntityCampfire, 18, 16).Fuel = FireMaxFuel
	load(t, w2, id2, Ingredient{ItemMeat, 1})
	stepN(w2, RackTicks/RackSmoke)
	if outCount(w2, e2, ItemJerky) != 1 {
		t.Fatalf("no jerky from a smoked rack in half the time")
	}
}

func TestSaltNeedsTheSea(t *testing.T) {
	w, id, e := built(t, EntityDryRack)
	load(t, w, id, Ingredient{ItemSeaWater, 2})
	stepN(w, RackTicks+5)
	if info, _ := w.CookContents(e.ID); info.State != CookNeedSea || outCount(w, e, ItemSalt) != 0 {
		t.Fatalf("inland: state %v, salt %d", info.State, outCount(w, e, ItemSalt))
	}

	// The same rack on a coast: the west edge of the map is sea.
	w = flatWorld()
	for y := 0; y < w.height; y++ {
		for x := 0; x < 14; x++ {
			w.tiles[y*w.width+x] = TileWater
		}
	}
	id = w.AddPlayer("a")
	e = w.addEntity(EntityDryRack, 17, 16)
	w.openCook(w.players[id], e)
	load(t, w, id, Ingredient{ItemSeaWater, 2})
	stepN(w, RackTicks)
	if outCount(w, e, ItemSalt) != SaltYield || outCount(w, e, ItemClayBowl) != 1 {
		t.Fatalf("by the sea: %v", w.station(e.ID).Slots)
	}
	stepN(w, RackTicks)
	if outCount(w, e, ItemSalt) != 2*SaltYield || outCount(w, e, ItemClayBowl) != 2 {
		t.Fatalf("second bowl: %v", w.station(e.ID).Slots)
	}
}

func TestScoopAndBoil(t *testing.T) {
	w := flatWorld()
	for y := 0; y < w.height; y++ { // the sea to the west
		for x := 0; x < 16; x++ {
			w.tiles[y*w.width+x] = TileWater
		}
	}
	w.tiles[16*w.width+18] = TileWater // a pond to the east
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X = 17.5
	p.Inv = InventoryWith(Ingredient{ItemClayBowl, 2}, Ingredient{ItemFlask, 1})
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 15, TY: 16})
	if p.Inv.Count(ItemSeaWater) != 1 || p.Inv.Count(ItemClayBowl) != 1 || p.Notice != NoticeScooped {
		t.Fatalf("bowl in the sea: %v, notice %v", p.Inv[:3], p.Notice)
	}
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 18, TY: 16})
	if p.Notice != NoticeWrongWater {
		t.Fatalf("bowl in a pond: notice %v", p.Notice)
	}
	w.Apply(id, Command{Kind: CmdSelect, Slot: 1})
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 15, TY: 16})
	if p.Notice != NoticeWrongWater {
		t.Fatalf("flask in the sea: notice %v", p.Notice)
	}
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 18, TY: 16})
	if p.Inv.Count(ItemDirtyWater) != 1 || p.Inv.Count(ItemFlask) != 0 {
		t.Fatalf("flask in a pond: %v", p.Inv[:4])
	}

	// Boiled in the pot, then drunk: the flask comes back.
	e := w.addEntity(EntityCampfire, 17, 17)
	e.Var = uint8(GearPot)
	w.openCook(p, e)
	for i := range p.Inv {
		if p.Inv[i].Item == ItemDirtyWater {
			w.Apply(id, Command{Kind: CmdCookPut, Slot: uint8(i)})
		}
	}
	stepN(w, 6*TicksPerSecond)
	if outCount(w, e, ItemCleanWater) != 1 {
		t.Fatalf("not boiled: %v", w.station(e.ID).Slots)
	}
	w.Apply(id, Command{Kind: CmdCookTake, Slot: CookIn})
	for i := range p.Inv {
		if p.Inv[i].Item == ItemCleanWater {
			w.Apply(id, Command{Kind: CmdEat, Slot: uint8(i)})
		}
	}
	if p.Inv.Count(ItemFlask) != 1 || p.Inv.Count(ItemCleanWater) != 0 {
		t.Fatalf("after drinking: %v", p.Inv[:5])
	}
}

func TestGearOnAndOffTheFire(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	fire := w.addEntity(EntityCampfire, 17, 16)
	p.Inv = InventoryWith(Ingredient{ItemClayPot, 1}, Ingredient{ItemGrill, 1})
	w.Apply(id, Command{Kind: CmdPlace})
	if fire.Var != uint8(GearPot) || p.Inv.Count(ItemClayPot) != 0 {
		t.Fatalf("pot not on the fire: var %d", fire.Var)
	}
	w.Apply(id, Command{Kind: CmdSelect, Slot: 1})
	w.Apply(id, Command{Kind: CmdPlace})
	if p.Notice != NoticeNeedHearth || p.Inv.Count(ItemGrill) != 1 {
		t.Fatalf("a grill onto a taken fire: notice %v", p.Notice)
	}
	// Action opens it, wood in the bag or not.
	w.SetInput(id, Input{Action: true})
	w.Step()
	w.SetInput(id, Input{})
	if p.OpenCook() != fire.ID {
		t.Fatalf("Action did not open the pot")
	}
	// Not taken down with something in it.
	p.Inv.Add(ItemMeat, 1)
	w.Apply(id, Command{Kind: CmdCookPut, Slot: 0})
	w.Apply(id, Command{Kind: CmdCookTakeDown})
	if p.Notice != NoticeCookNotEmpty || fire.Var == 0 {
		t.Fatalf("taken down full: notice %v", p.Notice)
	}
	if p.Notice = NoticeNone; true {
		w.Apply(id, Command{Kind: CmdCookPut, Slot: 2}) // nothing there
	}
	w.Apply(id, Command{Kind: CmdCookTake, Slot: 0})
	w.Apply(id, Command{Kind: CmdCookTakeDown})
	if fire.Var != 0 || p.Inv.Count(ItemClayPot) != 1 || p.OpenCook() != 0 {
		t.Fatalf("pot not back: var %d, pots %d", fire.Var, p.Inv.Count(ItemClayPot))
	}
	// Walking away closes it.
	fire.Var = uint8(GearPan)
	w.openCook(p, fire)
	p.X += 5
	w.Step()
	if p.OpenCook() != 0 {
		t.Fatalf("still open from afar")
	}
}

func TestStationRefusesWhatItCannotUse(t *testing.T) {
	w, id, _ := cookAt(t, GearPot)
	p := w.players[id]
	p.Inv = InventoryWith(Ingredient{ItemWood, 3})
	w.Apply(id, Command{Kind: CmdCookPut, Slot: 0})
	if p.Notice != NoticeCantCook || p.Inv.Count(ItemWood) != 3 {
		t.Fatalf("wood in the pot: notice %v", p.Notice)
	}
	p.Inv = InventoryWith(Ingredient{ItemMeat, 5})
	w.Apply(id, Command{Kind: CmdCookPut, Slot: 0, Count: 1})
	if p.Inv.Count(ItemMeat) != 4 {
		t.Fatalf("one piece asked, %d left", p.Inv.Count(ItemMeat))
	}
}

func TestRemovedStationSpills(t *testing.T) {
	w, id, e := built(t, EntityMortar)
	load(t, w, id, Ingredient{ItemFlowerWhite, 1})
	w.removeEntity(e)
	if len(w.drops) != 1 || w.drops[0].Item != ItemFlowerWhite {
		t.Fatalf("drops %v", w.drops)
	}
	w.Step()
	if len(w.stations) != 0 {
		t.Fatalf("station left behind")
	}
}

// Every dish can be made from the first thing of each class it asks for,
// at its own station (bar those whose class has nothing in it yet).
func TestEveryDishCooks(t *testing.T) {
	stationEntity := func(w *World, s StationKind) *Entity {
		switch s {
		case StationMortar:
			return w.addEntity(EntityMortar, 17, 16)
		case StationOven:
			e := w.addEntity(EntityOven, 17, 16)
			e.Fuel = OvenMaxFuel
			return e
		case StationRack:
			return w.addEntity(EntityDryRack, 17, 16)
		}
		e := w.addEntity(EntityCampfire, 17, 16)
		e.Fuel = FireMaxFuel
		e.Var = uint8(map[StationKind]CookGear{StationPot: GearPot, StationGrill: GearGrill, StationPan: GearPan}[s])
		return e
	}
	made := 0
	for id := CookNone + 2; id < NumCooks; id++ {
		r, ok := CookRecipeByID(id)
		if !ok {
			t.Errorf("no recipe for cook id %d", id)
			continue
		}
		var items []Ingredient
		possible := true
		for _, n := range r.Needs {
			k := n.Item
			if !k.Valid() {
				cl := ClassItems(n.Class)
				if len(cl) == 0 {
					possible = false
					break
				}
				k = cl[0]
			}
			items = append(items, Ingredient{k, n.Count})
		}
		if !possible {
			continue
		}
		w := flatWorld()
		pid := w.AddPlayer("a")
		e := stationEntity(w, r.Station)
		w.openCook(w.players[pid], e)
		load(t, w, pid, items...)
		stepN(w, r.Ticks)
		if outCount(w, e, r.Gives) != r.Count {
			t.Errorf("%s: got %d, slots %v", r.Gives.Def().Name, outCount(w, e, r.Gives), w.station(e.ID).Slots)
		}
		if !r.Gives.Edible() && r.Gives != ItemFlour {
			t.Errorf("%s is not edible", r.Gives.Def().Name)
		}
		made++
	}
	if made < 10 {
		t.Errorf("only %d dishes could be made", made)
	}
}

func TestCookItemsAndRecipes(t *testing.T) {
	for k := cookItemBase; k < cookItemsEnd; k++ {
		if !k.Valid() {
			t.Errorf("cook item %d has no name", k)
		}
		if got, ok := FindItem(k.Def().Name); !ok || got != k {
			t.Errorf("give does not find %q", k.Def().Name)
		}
	}
	for id := RecipeClayPot; id <= RecipeSaltMeat; id++ {
		r, _ := RecipeByID(id)
		if r.Name == "" || len(r.Inputs) == 0 || (!r.Gives.Valid() && r.Places == EntityNone) {
			t.Errorf("recipe %d is not filled in: %+v", id, r)
		}
	}
	// The pot is fired at a furnace.
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = InventoryWith(Ingredient{ItemClay, 5})
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeClayPot})
	if p.Notice != NoticeNeedFurnace {
		t.Fatalf("a pot without a furnace: notice %v", p.Notice)
	}
	w.addEntity(EntityFurnace, 17, 16).Fuel = 100
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeClayPot})
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeClayBowl})
	if p.Inv.Count(ItemClayPot) != 1 || p.Inv.Count(ItemClayBowl) != 1 || p.Inv.Count(ItemClay) != 0 {
		t.Fatalf("fired: %v", p.Inv[:3])
	}
}

func TestCookingSurvivesASave(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("a")
	p := w.players[id]
	tx, ty, ok := w.placeSpot(p)
	if !ok {
		t.Fatal("nowhere to build")
	}
	fire := w.addEntity(EntityCampfire, tx, ty)
	fire.Var, fire.Fuel = uint8(GearPot), 0 // a cold hearth with a pot
	w.openCook(p, fire)
	p.Inv = InventoryWith(Ingredient{ItemMeat, 3}, Ingredient{ItemSalt, 2})
	w.Apply(id, Command{Kind: CmdCookPut, Slot: 0})
	w.Apply(id, Command{Kind: CmdCookPut, Slot: 1})
	p.cook.buffs[BuffMight] = 777
	w.Step()

	raw, err := json.Marshal(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	var d SaveData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	e2 := w2.entities[fire.ID]
	if e2 == nil || e2.Var != uint8(GearPot) {
		t.Fatalf("the cold hearth did not come back: %+v", e2)
	}
	s := w2.station(fire.ID)
	if s.Slots[0] != (Slot{ItemMeat, 3}) || s.Slots[1] != (Slot{ItemSalt, 2}) {
		t.Fatalf("contents: %v", s.Slots)
	}
	id2 := w2.AddPlayer("a")
	if got := w2.players[id2].Buff(BuffMight); got != 776 {
		t.Fatalf("buff back as %d, want 776", got)
	}
}

func TestClayLiesByTheWater(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 11, Width: 256, Height: 256, Style: StyleSeas})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range w.entities {
		if e.Kind != EntityClay {
			continue
		}
		n++
		wet := false
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				wet = wet || w.TileAt(e.TX+dx, e.TY+dy) == TileWater
			}
		}
		if !wet {
			t.Fatalf("clay at %d,%d away from water", e.TX, e.TY)
		}
	}
	if n < 5 {
		t.Fatalf("only %d clay beds in a 256×256 world", n)
	}
	t.Logf("%d clay beds", n)

	// Dug by hand, it comes back.
	f := flatWorld()
	id := f.AddPlayer("a")
	e := f.addEntity(EntityClay, 16, 17)
	f.SetInput(id, Input{Action: true})
	f.Step()
	f.SetInput(id, Input{})
	if p := f.players[id]; p.Inv.Count(ItemClay) != ClayYield || e.Ready {
		t.Fatalf("dug %d clay, ready %v", p.Inv.Count(ItemClay), e.Ready)
	}
	f.tick += ClayRegrowTicks
	f.watch(e)
	f.Step()
	if !e.Ready {
		t.Fatalf("the clay did not come back")
	}
}
