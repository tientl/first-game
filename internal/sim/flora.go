package sim

import (
	"math"
	"math/rand/v2"
)

// The wild plants that are picked for what they give, each in its own
// country. They are all one kind of entity, EntityFlora, told apart by Var:
//
//   - For water one day: bamboo in groves in the woods (stalks to build
//     with), gourds on the meadows, pitcher plants in the marsh, whose cups
//     fill with rain.
//   - For rope: flax in drifts on the meadows, lianas in the woods.
//   - Poison: the red-capped mushroom and the nightshade's black berries.
//     Eating either poisons.
//   - Medicine: aloe in the desert (it heals), herbs on meadows and in the
//     woods (they draw poison out), moss on rocks (laid on a wound).
//   - By the water: mangroves and seaweed in the shallows of the sea, water
//     lilies and lotus on the ponds (toads sit on their leaves), horsetail on
//     the banks.
//   - To farm: wild rice by the ponds, wild corn and squash on the meadows.
//     Their seed is sown in tilled soil like the sweet potato.
//
// Lilies, lotus, seaweed and mangroves stand in the water: the only
// entities on tiles nobody walks on.

// FloraKind is which wild plant an EntityFlora is.
type FloraKind uint8

const (
	FloraBamboo FloraKind = iota
	FloraGourd
	FloraPitcher
	FloraFlax
	FloraVine
	FloraPoisonMushroom
	FloraNightshade
	FloraAloe
	FloraHerb
	FloraMoss
	FloraMangrove
	FloraSeaweed
	FloraLily
	FloraLotus
	FloraHorsetail
	FloraWildRice
	FloraWildCorn
	FloraWildSquash
	NumFlora
)

// Flora and food-poison tuning.
const (
	FoodPoisonTicks  = 8 * TicksPerSecond // how long a poison mushroom or berry eats at you
	floraSpawnClear  = 3                  // tiles round the spawn kept free of the new plants
	SquashSeedYield  = 2                  // seeds out of a squash plant
	pitcherRainTicks = 20 * TicksPerSecond
)

type floraDef struct {
	name   string
	item   ItemKind
	yield  int
	regrow uint64
	axe    bool     // cut with an axe
	hits   int      // blows with a stone axe; 0 means one
	water  bool     // stands in the water
	bonus  ItemKind // given besides, bonusN of it
	bonusN int
}

func (f *floraDef) hp() int { return max(1, f.hits) * HitPower }

var floraDefs = [NumFlora]floraDef{
	FloraBamboo:         {name: "Bụi tre", item: ItemBamboo, yield: 3, regrow: 3 * DayTicks / 2, axe: true, hits: 2},
	FloraGourd:          {name: "Giàn bầu", item: ItemGourd, yield: 2, regrow: 3 * DayTicks / 2},
	FloraPitcher:        {name: "Cây nắp ấm", item: ItemPitcherWater, yield: 2, regrow: 2 * DayTicks},
	FloraFlax:           {name: "Cây lanh", item: ItemFlax, yield: 2, regrow: DayTicks},
	FloraVine:           {name: "Dây leo rừng", item: ItemVine, yield: 2, regrow: DayTicks},
	FloraPoisonMushroom: {name: "Nấm độc", item: ItemPoisonMushroom, yield: 2, regrow: DayTicks},
	FloraNightshade:     {name: "Bụi cà độc", item: ItemNightshade, yield: 3, regrow: DayTicks},
	FloraAloe:           {name: "Lô hội", item: ItemAloe, yield: 2, regrow: 3 * DayTicks / 2},
	FloraHerb:           {name: "Thảo dược", item: ItemHerb, yield: 2, regrow: DayTicks},
	FloraMoss:           {name: "Đám rêu", item: ItemMoss, yield: 2, regrow: DayTicks},
	FloraMangrove:       {name: "Cây đước", item: ItemWood, yield: 2, regrow: TreeRegrowTicks, axe: true, hits: 3, water: true, bonus: ItemStick, bonusN: 1},
	FloraSeaweed:        {name: "Rong biển", item: ItemSeaweed, yield: 2, regrow: DayTicks, water: true},
	FloraLily:           {name: "Hoa súng", item: ItemLily, yield: 1, regrow: DayTicks, water: true},
	FloraLotus:          {name: "Sen", item: ItemLotusSeed, yield: 3, regrow: 3 * DayTicks / 2, water: true},
	FloraHorsetail:      {name: "Cỏ đuôi ngựa", item: ItemHorsetail, yield: 2, regrow: DayTicks},
	FloraWildRice:       {name: "Lúa hoang", item: ItemRice, yield: 3, regrow: 3 * DayTicks / 2},
	FloraWildCorn:       {name: "Ngô hoang", item: ItemCorn, yield: 2, regrow: 3 * DayTicks / 2},
	FloraWildSquash:     {name: "Bí hoang", item: ItemSquash, yield: 1, regrow: 3 * DayTicks / 2, bonus: ItemSquashSeed, bonusN: SquashSeedYield},
}

func (f FloraKind) def() *floraDef { return &floraDefs[f%NumFlora] }

// Name is the display name of f.
func (f FloraKind) Name() string { return f.def().name }

// Item is what picking f gives, and how many.
func (f FloraKind) Item() (ItemKind, int) { d := f.def(); return d.item, d.yield }

// NeedsAxe reports whether f is cut with an axe rather than picked by hand.
func (f FloraKind) NeedsAxe() bool { return f.def().axe }

// InWater reports whether f stands in the water.
func (f FloraKind) InWater() bool { return f.def().water }

// Pad reports whether f floats leaves a toad can sit on.
func (f FloraKind) Pad() bool { return f%NumFlora == FloraLily || f%NumFlora == FloraLotus }

// EntityLabel is the name of an entity of kind k whose Var is v: its own
// plant's or crop's name where one kind of entity is many things.
func EntityLabel(k EntityKind, v uint8) string {
	switch k {
	case EntityFlora:
		return FloraKind(v).Name()
	case EntityCrop:
		return CropKind(v).Name()
	}
	return k.Name()
}

// pickFlora lands one blow of dmg on the plant e, or picks it.
func (w *World) pickFlora(p *Player, e *Entity, dmg int) bool {
	f := FloraKind(e.Var).def()
	if !e.Ready || f.water && w.frozenAt(e.TX, e.TY) {
		return false
	}
	if f.bonus.Valid() && e.HP <= dmg && p.Inv.Room(f.bonus) < f.bonusN {
		p.notify(NoticeBagFull, uint8(f.bonus))
		return false
	}
	return w.harvest(p, e, f.item, f.yield, dmg, func() {
		if f.bonus.Valid() {
			p.Inv.Add(f.bonus, f.bonusN)
		}
		e.Ready = false
		e.RegrowAt = w.tick + f.regrow
	})
}

// catchRain brings an empty pitcher plant's cups back much sooner while it
// rains: they fill in pitcherRainTicks.
func (w *World) catchRain(e *Entity) {
	if e.Kind == EntityFlora && FloraKind(e.Var) == FloraPitcher && !e.Ready &&
		e.RegrowAt > w.tick+pitcherRainTicks && w.Weather().Wet() {
		e.RegrowAt = w.tick + pitcherRainTicks
	}
}

// padAt reports whether a lily or lotus floats its leaves on tile (tx, ty).
func (w *World) padAt(tx, ty int) bool {
	e := w.entityAt(tx, ty)
	return e != nil && e.Kind == EntityFlora && FloraKind(e.Var).Pad() && !w.frozenAt(tx, ty)
}

// pondWater reports whether tile (tx, ty) is the water of a lake or pond.
func (w *World) pondWater(tx, ty int) bool {
	if !w.inBounds(tx, ty) {
		return false
	}
	if !w.wx.pondDone {
		w.wx.pond = PondMask(w.width, w.height, w.tiles)
		w.wx.pondDone = true
	}
	return w.wx.pond[ty*w.width+tx]
}

// byPond reports whether walkable tile (tx, ty) has pond water on one of
// its eight sides.
func (w *World) byPond(tx, ty int) bool {
	if !w.TileAt(tx, ty).Walkable() {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && w.pondWater(tx+dx, ty+dy) {
				return true
			}
		}
	}
	return false
}

// canHold reports whether an entity of kind k, with Var v, may go on tile
// (tx, ty): free ground, or for a water plant free water.
func (w *World) canHold(k EntityKind, v uint8, tx, ty int) bool {
	if k == EntityFlora && v < uint8(NumFlora) && FloraKind(v).InWater() {
		return w.inBounds(tx, ty) && w.tiles[ty*w.width+tx].Sea() && w.occ[ty*w.width+tx] == 0
	}
	return w.canPlace(tx, ty)
}

// fullHP is the HP of e untouched.
func (e *Entity) fullHP() int {
	if e.Kind == EntityFlora {
		return FloraKind(e.Var).def().hp()
	}
	return maxHP(e.Kind, e.Big)
}

// addFlora plants f on (tx, ty). The caller checks canHold first.
func (w *World) addFlora(f FloraKind, tx, ty int) *Entity {
	e := w.addEntity(EntityFlora, tx, ty)
	e.Var = uint8(f)
	e.HP = f.def().hp()
	return e
}

type floraChance struct {
	f FloraKind
	p float64
}

func either(yes bool, a, b float64) float64 {
	if yes {
		return a
	}
	return b
}

// scatterFlora grows the wild plants over a new world, each in its own
// country, on tiles nothing else took. It runs after everything else is
// placed, so it moves nothing that was there before it.
func (w *World) scatterFlora(rng *rand.Rand, seed uint32) {
	sx, sy := int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY))
	patch := func(x, y, scale float64, salt uint32, level float64) bool {
		return fbm(x/scale+float64(salt%17), y/scale+float64(salt%29), seed+salt, 2) > level
	}
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			i := ty*w.width + tx
			r := rng.Float64()
			if w.occ[i] != 0 || max(abs(tx-sx), abs(ty-sy)) <= floraSpawnClear {
				continue
			}
			x, y := float64(tx)+0.5, float64(ty)+0.5
			kind := NumFlora
			pickOf := func(cs ...floraChance) {
				sum := 0.0
				for _, c := range cs {
					if sum += c.p; r < sum {
						kind = c.f
						return
					}
				}
			}
			tile := w.tiles[i]
			wet := tile.Walkable() && w.waterside(tx, ty)
			switch tile {
			case TileWater, TileShallow:
				switch {
				case w.underground(ty):
				case w.pondWater(tx, ty):
					near := w.isShallow(tx, ty)
					pickOf(floraChance{FloraLily, either(near, 0.16, 0.07)}, floraChance{FloraLotus, either(near, 0.07, 0.03)})
				case w.isShallow(tx, ty):
					pickOf(floraChance{FloraMangrove, either(patch(x, y, 11, 501, 0.6), 0.3, 0)}, floraChance{FloraSeaweed, 0.035})
				}
			case TileGrass:
				switch {
				case wet && w.byPond(tx, ty):
					pickOf(floraChance{FloraHorsetail, 0.1}, floraChance{FloraWildRice, 0.09}, floraChance{FloraHerb, 0.01})
				case wet:
					pickOf(floraChance{FloraHorsetail, 0.05})
				default:
					pickOf(floraChance{FloraFlax, either(patch(x, y, 8, 511, 0.64), 0.07, 0)},
						floraChance{FloraWildCorn, either(patch(x, y, 7, 523, 0.68), 0.06, 0)},
						floraChance{FloraHerb, 0.004}, floraChance{FloraGourd, 0.0025}, floraChance{FloraWildSquash, 0.0025})
				}
			case TileForest:
				pickOf(floraChance{FloraBamboo, either(patch(x, y, 9, 537, 0.66), 0.22, 0)},
					floraChance{FloraHorsetail, either(wet, 0.07, 0)}, floraChance{FloraVine, 0.012},
					floraChance{FloraHerb, 0.005}, floraChance{FloraMoss, 0.006}, floraChance{FloraNightshade, 0.004},
					floraChance{FloraPoisonMushroom, 0.004}, floraChance{FloraPitcher, 0.002}, floraChance{FloraGourd, 0.0015})
			case TileMarsh:
				pickOf(floraChance{FloraWildRice, either(wet, 0.1, 0)}, floraChance{FloraHorsetail, either(wet, 0.1, 0.004)},
					floraChance{FloraPitcher, 0.014}, floraChance{FloraPoisonMushroom, 0.007}, floraChance{FloraMoss, 0.006},
					floraChance{FloraVine, 0.004}, floraChance{FloraBamboo, 0.004})
			case TileSand:
				if w.seaside(tx, ty) {
					pickOf(floraChance{FloraMangrove, either(patch(x, y, 11, 501, 0.6), 0.1, 0)})
				} else {
					pickOf(floraChance{FloraAloe, 0.004})
				}
			case TileDesert:
				pickOf(floraChance{FloraAloe, 0.014})
			case TileRocky:
				pickOf(floraChance{FloraMoss, 0.02}, floraChance{FloraHerb, 0.003})
			case TileCave:
				pickOf(floraChance{FloraMoss, 0.012}, floraChance{FloraPoisonMushroom, 0.008})
			}
			if kind < NumFlora && w.canHold(EntityFlora, uint8(kind), tx, ty) {
				w.addFlora(kind, tx, ty)
			}
		}
	}
}

// underground reports whether row ty lies in the caves below the world.
func (w *World) underground(ty int) bool { return w.surfaceH > 0 && ty >= w.surfaceH }

// --- crops ---------------------------------------------------------------------

// CropKind is what grows in a bed of tilled soil; it is an EntityCrop's Var.
type CropKind uint8

const (
	CropPotato CropKind = iota
	CropRice
	CropCorn
	CropSquash
	NumCrops
)

type cropDef struct {
	name   string
	seed   ItemKind // what is sown
	item   ItemKind // what is reaped
	yield  int
	bonus  ItemKind // and besides, bonusN of it
	bonusN int
	grow   int // ticks from sown to ripe
}

var cropDefs = [NumCrops]cropDef{
	CropPotato: {name: "Khoai lang", seed: ItemPotato, item: ItemPotato, yield: CropYield, grow: CropGrowTicks},
	CropRice:   {name: "Lúa", seed: ItemRice, item: ItemRice, yield: 5, grow: CropGrowTicks},
	CropCorn:   {name: "Ngô", seed: ItemCorn, item: ItemCorn, yield: 3, grow: 5 * CropGrowTicks / 4},
	CropSquash: {name: "Bí", seed: ItemSquashSeed, item: ItemSquash, yield: 2, bonus: ItemSquashSeed, bonusN: SquashSeedYield, grow: 3 * CropGrowTicks / 2},
}

func (c CropKind) def() *cropDef { return &cropDefs[c%NumCrops] }

// Name is the display name of c.
func (c CropKind) Name() string { return c.def().name }

// Seed is what is sown to grow c.
func (c CropKind) Seed() ItemKind { return c.def().seed }

// GrowTicks is how long c takes from sown to ripe.
func (c CropKind) GrowTicks() int { return c.def().grow }

func (c CropKind) growTicks() int { return c.def().grow }

// CropOf is the crop item k is the seed of.
func CropOf(k ItemKind) (CropKind, bool) {
	for c := CropKind(0); c < NumCrops; c++ {
		if cropDefs[c].seed == k {
			return c, true
		}
	}
	return 0, false
}

// SeedToSow is what a player holding held, with inv, would sow in an empty
// bed: what is in the hand if that is a seed, else the first seed carried.
// The server and the client's label both use it.
func SeedToSow(held ItemKind, inv *Inventory) (CropKind, bool) {
	if c, ok := CropOf(held); ok && inv.Count(held) > 0 {
		return c, true
	}
	for c := CropKind(0); c < NumCrops; c++ {
		if inv.Count(cropDefs[c].seed) > 0 {
			return c, true
		}
	}
	return 0, false
}

// plant sows the empty bed e with the seed p holds, or failing that the
// first kind of seed p carries.
func (w *World) plant(p *Player, e *Entity) bool {
	c, ok := SeedToSow(p.Held(), &p.Inv)
	if !ok {
		p.notify(NoticeNeedSeed, uint8(ItemPotato))
		return false
	}
	p.Inv.Take(c.def().seed, 1)
	fed := e.Big // dung in the bed: see dung.go
	e.Kind, e.Ready, e.Big, e.HP, e.Var = EntityCrop, false, false, maxHP(EntityCrop, false), uint8(c)
	grow := uint64(c.growTicks())
	e.RegrowAt = w.tick + grow
	if fed {
		e.RegrowAt = w.tick + grow/DungBoost
	}
	return true
}

// reap digs up or cuts the ripe crop e, leaving its bed free again.
func (w *World) reap(p *Player, e *Entity) bool {
	c := CropKind(e.Var).def()
	if c.bonus.Valid() && p.Inv.Room(c.bonus) < c.bonusN {
		p.notify(NoticeBagFull, uint8(c.bonus))
		return false
	}
	return w.harvest(p, e, c.item, c.yield, HitPower, func() {
		if c.bonus.Valid() {
			p.Inv.Add(c.bonus, c.bonusN)
		}
		e.Kind, e.Ready, e.Var = EntityFarmland, true, 0 // the bed is free again
		e.RegrowAt = w.tick + FarmlandDecayTicks
	})
}
