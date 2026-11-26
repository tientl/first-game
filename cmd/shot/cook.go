package main

import (
	"flag"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The cooking gear, for checking its art:
//
//	shot -cook -time 0.6          a camp: a pot simmering on its fire, a laden grill, a pan,
//	                              a mortar, a lit oven with a loaf out, drying racks, clay beds
//	shot -cook -cookui pot        the same with the pot's window open (grill, pan, mortar, oven, rack)
//	shot -buffs                   every buff of a dish on the HUD
//	shot -overlay bag -bagfrom 150   the icons of the cooking items
var (
	cookFlag   = flag.Bool("cook", false, "put a cooking camp round the player: a pot, a grill and a pan over fires, a mortar, a lit oven, drying racks and clay beds")
	cookUIFlag = flag.String("cookui", "", "with -cook: open the cooking window of pot, grill, pan, mortar, oven or rack, something cooking in it")
	buffsFlag  = flag.Bool("buffs", false, "the buffs of dishes on the player: every chip, each with time left")
)

// cookScene changes snap as the cooking flags ask; (sx, sy) is where the
// player stands.
func cookScene(snap *protocol.Snapshot, sx, sy float64) {
	if *buffsFlag {
		snap.Self.Cook.Buffs = [sim.NumBuffs]int32{52 * 20, 215 * 20, 96 * 20, 171 * 20, 438 * 20}
	}
	if !*cookFlag && *cookUIFlag == "" {
		return
	}
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	// Clear the ground the camp stands on.
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < cx-8 || e.TX > cx+8 || e.TY < cy-4 || e.TY > cy+5 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	id := sim.EntityID(1<<30 + 500)
	ids := map[string]sim.EntityID{}
	add := func(name string, kind sim.EntityKind, dx, dy int16, gear sim.CookGear, fuel int32, flags uint8) {
		id++
		ids[name] = id
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: kind, TX: cx + dx, TY: cy + dy, Ready: true, Var: uint8(gear), Fuel: fuel, Cook: flags})
	}
	const busy = sim.CookFlagLoaded | sim.CookFlagWorking
	add("pot", sim.EntityCampfire, -2, -1, sim.GearPot, sim.FireStartFuel, busy)
	add("grill", sim.EntityCampfire, 2, -1, sim.GearGrill, sim.FireStartFuel, busy)
	add("pan", sim.EntityCampfire, 0, 2, sim.GearPan, sim.FireStartFuel*2/3, busy)
	add("", sim.EntityCampfire, -5, 3, sim.GearPot, 0, 0) // a cold hearth, its pot empty
	add("mortar", sim.EntityMortar, 2, 2, 0, 0, sim.CookFlagLoaded)
	add("oven", sim.EntityOven, 5, 0, 0, sim.OvenMaxFuel/2, busy|sim.CookFlagDone)
	add("rack", sim.EntityDryRack, -5, 0, 0, 0, busy)
	add("", sim.EntityDryRack, -4, -3, 0, 0, busy|sim.CookFlagFish)
	add("", sim.EntityDryRack, 4, -3, 0, 0, busy|sim.CookFlagSalt)
	add("", sim.EntityDryRack, 7, 2, 0, 0, sim.CookFlagDone)
	add("", sim.EntityDryRack, -7, -2, 0, 0, 0)
	add("", sim.EntityClay, 4, 4, 0, 0, 0)
	add("", sim.EntityClay, 5, 3, 0, 0, 0)
	snap.Entities[len(snap.Entities)-1].Ready = false // dug out
	add("", sim.EntityClay, -2, 4, 0, 0, 0)

	inv := &snap.Self.Inv
	for _, it := range []sim.Ingredient{
		{Item: sim.ItemClayBowl, Count: 4}, {Item: sim.ItemSalt, Count: 6}, {Item: sim.ItemMeat, Count: 7},
		{Item: sim.ItemFish, Count: 3}, {Item: sim.ItemPotato, Count: 9}, {Item: sim.ItemMushroom, Count: 5},
		{Item: sim.ItemFlowerPurple, Count: 4}, {Item: sim.ItemDirtyWater, Count: 2}, {Item: sim.ItemBone, Count: 3},
		{Item: sim.ItemFlour, Count: 6}, {Item: sim.ItemBerry, Count: 12}, {Item: sim.ItemSeaWater, Count: 2},
		{Item: sim.ItemStew, Count: 2}, {Item: sim.ItemHerbFish, Count: 1}, {Item: sim.ItemBread, Count: 3},
		{Item: sim.ItemGinkgoNut, Count: 8}, {Item: sim.ItemSaltedMeat, Count: 2}, {Item: sim.ItemCoal, Count: 4},
	} {
		inv.Add(it.Item, it.Count)
	}

	if *cookUIFlag == "" {
		return
	}
	c := &snap.Self.Cook
	c.ID = ids[*cookUIFlag]
	c.State = sim.CookWorking
	c.Fuel = sim.FireStartFuel
	put := func(i int, k sim.ItemKind, n int) { c.Slots[i] = sim.Slot{Item: k, Count: n} }
	dish := func(id sim.CookID, frac float64) {
		r, _ := sim.CookRecipeByID(id)
		c.Dish, c.Total, c.Progress = id, int32(r.Ticks), int32(float64(r.Ticks)*frac)
	}
	switch *cookUIFlag {
	case "grill":
		c.Kind = sim.StationGrill
		put(0, sim.ItemMeat, 3)
		put(1, sim.ItemFish, 2)
		put(2, sim.ItemPotato, 4)
		put(3, sim.ItemClam, 2)
		put(4, sim.ItemCookedMeat, 2)
		put(5, sim.ItemRoastPotato, 1)
		c.Dish, c.Total, c.Progress = sim.CookBatch, sim.GrillTicks, sim.GrillTicks*2/5
	case "pan":
		c.Kind = sim.StationPan
		put(0, sim.ItemMeat, 2)
		put(1, sim.ItemSalt, 3)
		put(4, sim.ItemSearedMeat, 1)
		dish(sim.CookSear, 0.7)
	case "mortar":
		c.Kind = sim.StationMortar
		put(0, sim.ItemFlowerWhite, 4)
		put(4, sim.ItemHerbMedicine, 1)
		dish(sim.CookMedicine, 0.5)
	case "oven":
		c.Kind = sim.StationOven
		c.ID = ids["oven"]
		c.Fuel = sim.OvenMaxFuel / 2
		put(0, sim.ItemFlour, 4)
		put(1, sim.ItemFig, 2)
		put(4, sim.ItemBread, 2)
		dish(sim.CookPie, 0.35)
	case "rack":
		c.Kind = sim.StationRack
		put(0, sim.ItemMeat, 3)
		put(1, sim.ItemFish, 2)
		put(2, sim.ItemSeaWater, 2)
		put(4, sim.ItemJerky, 2)
		put(5, sim.ItemSalt, 4)
		put(6, sim.ItemClayBowl, 2)
		c.Dish, c.Total, c.Progress = sim.CookBatch, sim.RackTicks, sim.RackTicks*3/5
	default:
		c.ID = ids["pot"]
		c.Kind = sim.StationPot
		put(0, sim.ItemMeat, 2)
		put(1, sim.ItemPotato, 3)
		put(2, sim.ItemClayBowl, 2)
		put(3, sim.ItemSalt, 1)
		put(4, sim.ItemStew, 2)
		dish(sim.CookStew, 0.62)
	}
}
