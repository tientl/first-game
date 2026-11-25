package sim

import (
	"math"
	"math/rand/v2"
)

// Materials come in tiers: stone, then flint, jade, agate and obsidian. (In
// the code the tiers keep the names of the metals they replaced: copper,
// iron, platinum and diamond, so saves stay readable.) Each lies in veins in
// its own region, farther from the spawn and harder to reach the better it
// is, and only a pickaxe of the tier below digs it. Raw flint, jade and
// agate are worked at a lit kiln into blades and polished stone; those and
// obsidian make axes, pickaxes and hoes of their tier, which hit harder.

// Tier is a material grade.
type Tier uint8

const (
	TierStone Tier = iota
	TierCopper
	TierIron
	TierPlatinum
	TierDiamond
	NumTiers
)

// ToolClass is what a tool does, whatever it is made of.
type ToolClass uint8

const (
	NoTool      ToolClass = iota
	ToolAxe               // chops trees, cacti and walls
	ToolPickaxe           // mines boulders, coal and ores
	ToolHoe               // tills soil, when held
	NumToolClasses
)

// HitPower is the damage of one hit by hand or with a stone tool. Entity HP
// counts in these units: something that takes n hits has n*HitPower.
const HitPower = 4

// toolPower is the damage of one hit with a tool of each tier.
var toolPower = [NumTiers]int{HitPower, 5, 6, 8, 10}

// ToolPower is the damage one hit with a tool of tier t does.
func ToolPower(t Tier) int { return toolPower[min(t, NumTiers-1)] }

// tools lists every tool item by class and tier.
var tools = [NumToolClasses][NumTiers]ItemKind{
	ToolAxe:     {ItemAxe, ItemCopperAxe, ItemIronAxe, ItemPlatinumAxe, ItemDiamondAxe},
	ToolPickaxe: {ItemPickaxe, ItemCopperPickaxe, ItemIronPickaxe, ItemPlatinumPickaxe, ItemDiamondPickaxe},
	ToolHoe:     {ItemHoe, ItemCopperHoe, ItemIronHoe, ItemPlatinumHoe, ItemDiamondHoe},
}

// ToolItem is the tool of class c made of tier t.
func ToolItem(c ToolClass, t Tier) ItemKind {
	if c >= NumToolClasses || t >= NumTiers {
		return ItemNone
	}
	return tools[c][t]
}

// ToolKind is the class and tier of k, or NoTool.
func (k ItemKind) ToolKind() (ToolClass, Tier) {
	for c := ToolAxe; c < NumToolClasses; c++ {
		for t := TierStone; t < NumTiers; t++ {
			if tools[c][t] == k {
				return c, t
			}
		}
	}
	return NoTool, 0
}

// BestTool is the highest tier of class c in the inventory.
func (inv *Inventory) BestTool(c ToolClass) (Tier, bool) {
	for t := NumTiers; t > 0; t-- {
		if inv.Count(ToolItem(c, t-1)) > 0 {
			return t - 1, true
		}
	}
	return 0, false
}

// TierMaterial is what tier t's tools are made of: a bar, or diamonds.
func TierMaterial(t Tier) ItemKind {
	return [NumTiers]ItemKind{ItemPebble, ItemCopperBar, ItemIronBar, ItemPlatinumBar, ItemDiamond}[min(t, NumTiers-1)]
}

// toolCost is how much of its tier's material a tool takes, besides 2 sticks.
var toolCost = [NumToolClasses]int{ToolAxe: 3, ToolPickaxe: 3, ToolHoe: 2}

// TierToolRecipe is the recipe for the tool of class c and tier t (copper
// or better; stone tools have their own recipes).
func TierToolRecipe(c ToolClass, t Tier) RecipeID {
	return recipeTierTools + RecipeID(c-ToolAxe)*RecipeID(NumTiers-1) + RecipeID(t-TierCopper)
}

func init() {
	fillTierTools()
	fillGear()
}

// fillTierTools writes the metal and diamond tool recipes into the table.
func fillTierTools() {
	for c := ToolAxe; c < NumToolClasses; c++ {
		for t := TierCopper; t < NumTiers; t++ {
			id, item := TierToolRecipe(c, t), ToolItem(c, t)
			recipes[id] = Recipe{
				ID:     id,
				Name:   item.Def().Name,
				Inputs: []Ingredient{{Item: ItemStick, Count: 2}, {Item: TierMaterial(t), Count: toolCost[c]}},
				Gives:  item,
			}
		}
	}
}

// oreDef is one kind of ore in the world.
type oreDef struct {
	kind  EntityKind
	item  ItemKind // what mining it gives
	tier  Tier     // a pickaxe of the tier below mines it
	hits  int      // with a stone pickaxe's power
	yield int
	home  Tile    // the region it lies in
	share float64 // of the free home tiles that carry one
}

var ores = [...]oreDef{
	{EntityCopperOre, ItemCopperOre, TierCopper, 6, 3, TileRocky, 0.025},
	{EntityIronOre, ItemIronOre, TierIron, 9, 3, TileCave, 0.06},                 // caves are small and dark
	{EntityPlatinumOre, ItemPlatinumOre, TierPlatinum, 12, 2, TileDesert, 0.006}, // deserts are wide
	{EntityDiamondOre, ItemDiamond, TierDiamond, 15, 2, TileAsh, 0.02},
}

// oreOf is the ore an entity of kind k is, if any.
func oreOf(k EntityKind) (oreDef, bool) {
	for _, o := range ores {
		if o.kind == k {
			return o, true
		}
	}
	return oreDef{}, false
}

// OreTier is the tier of the ore an entity of kind k is; ok is false for
// anything that is not an ore. The pickaxe it needs is one tier lower.
func OreTier(k EntityKind) (t Tier, ok bool) {
	o, ok := oreOf(k)
	return o.tier, ok
}

// scatterOres lays each ore on free tiles of its home region. It has its own
// rng so the rest of the world does not shift.
func (w *World) scatterOres(rng *rand.Rand) {
	w.scatterVeins(rng, veins, nil) // see orevein.go; jade comes with scatterDeepOres
}

// mine lands one pickaxe hit on the ore e.
func (w *World) mine(p *Player, e *Entity, o oreDef) bool {
	need := o.tier - 1
	tier, ok := p.Inv.BestTool(ToolPickaxe)
	if !ok || tier < need {
		p.notify(NoticeNeedTool, uint8(ToolItem(ToolPickaxe, need)))
		return false
	}
	return w.harvest(p, e, o.item, o.yield, ToolPower(tier), func() { w.removeEntity(e) })
}

// Furnace tuning.
const (
	FurnaceRadius  = 2.0                  // tiles within which a lit furnace smelts, cooks and warms
	FurnaceMaxFuel = 240 * TicksPerSecond // the most fuel a furnace holds
)

// stoke adds one coal, or else one wood, to the fire in e, up to most ticks
// of fuel.
func (w *World) stoke(p *Player, e *Entity, most int) bool {
	switch {
	case p.Inv.Take(ItemCoal, 1) == 1:
		e.Fuel = min(most, e.Fuel+CoalFuel)
	case p.Inv.Take(ItemWood, 1) == 1:
		e.Fuel = min(most, e.Fuel+FireFuelPerWood)
	default:
		p.notify(NoticeNoneLeft, uint8(ItemWood))
		return false
	}
	w.watch(e) // a cold furnace starts burning
	return true
}

// nearFurnace reports whether (x, y) is beside a lit furnace.
func (w *World) nearFurnace(x, y float64) bool {
	reach := int(math.Ceil(FurnaceRadius))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - reach; ty <= cy+reach; ty++ {
		for tx := cx - reach; tx <= cx+reach; tx++ {
			e := w.entityAt(tx, ty)
			if e == nil || e.Kind != EntityFurnace || e.Fuel <= 0 {
				continue
			}
			if dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y; dx*dx+dy*dy <= FurnaceRadius*FurnaceRadius {
				return true
			}
		}
	}
	return false
}

// FireReach is how far, in tiles, an entity of kind burning with fuel warms,
// cooks and lights; ok is false for anything that gives no fire. Server and
// client both use it.
func FireReach(kind EntityKind, fuel int) (r float64, ok bool) {
	if fuel <= 0 {
		return 0, false
	}
	switch kind {
	case EntityCampfire:
		return FireRadius(fuel), true
	case EntityFurnace:
		return FurnaceRadius, true
	case EntityOven:
		return OvenRadius, true // see cooking.go
	case EntityTorchStand:
		return TorchStandReach, true
	case EntityBonfire:
		return MaxFireRadius, true
	case EntityCaveExit:
		return caveExitLight, true // daylight down the shaft
	}
	return 0, false
}
