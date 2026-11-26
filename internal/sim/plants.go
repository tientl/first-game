package sim

import (
	"math"
	"math/rand/v2"
)

// The plants of the tribal age:
//
//   - Wild flowers of six colours dot the meadows and the woods. Picked by
//     hand they give the flower, and often a seed.
//   - Seeds are planted with V on soft ground in front: flower seed grows
//     flowers, berry seed a berry bush, tree seed a tree, glow seed a plant
//     that glows at night, man-eater seed a man-eating plant that guards a
//     home from beasts (it spares players). Berry bushes and felled trees
//     give their seeds too.
//   - Some plants of the deep woods, the swamps and the caves glow at night,
//     blue and violet, like a forest in a dream. Picking one gives glowing
//     spores (and a seed); it glows again once grown back.
//   - Man-eating plants grow wild in the swamps and the far jungle. They snap
//     at anyone and anything that comes within reach. A few blows cut one
//     down, and it leaves its seeds.

// FlowerKind is a colour of wild flower.
type FlowerKind uint8

const (
	FlowerRed FlowerKind = iota
	FlowerYellow
	FlowerBlue
	FlowerPurple
	FlowerWhite
	FlowerPink
	NumFlowers
)

// Item is the flower item a picked flower of kind f gives.
func (f FlowerKind) Item() ItemKind { return ItemFlowerRed + ItemKind(f%NumFlowers) }

// SeedKind is what a seedling grows into.
type SeedKind uint8

const (
	SeedFlower SeedKind = iota
	SeedBerry
	SeedTree
	SeedGlow
	SeedManEater
	NumSeeds
)

func seedOf(k ItemKind) SeedKind {
	switch k {
	case ItemBerrySeed:
		return SeedBerry
	case ItemTreeSeed:
		return SeedTree
	case ItemGlowSeed:
		return SeedGlow
	case ItemManEaterSeed:
		return SeedManEater
	}
	return SeedFlower
}

// IsSeed reports whether k is planted with V.
func IsSeed(k ItemKind) bool { return k >= ItemFlowerSeed && k <= ItemManEaterSeed }

func (s SeedKind) growTicks() uint64 {
	switch s {
	case SeedFlower:
		return DayTicks / 3
	case SeedTree:
		return 3 * DayTicks / 2
	}
	return DayTicks
}

// Plant tuning.
const (
	FlowerRegrowTicks  = DayTicks
	FlowerSeedChance   = 0.4
	BerrySeedChance    = 0.25
	GlowRegrowTicks    = DayTicks
	GlowSeedChance     = 0.5
	GlowSporeYield     = 2
	GlowLightRadius    = 2.6 // tiles of a glowing plant's light at night (the tree's is bigger)
	ManEaterReach      = 2.2
	ManEaterDamage     = 16.0
	ManEaterBeastBite  = 3
	ManEaterCooldown   = 3 * TicksPerSecond / 2
	ManEaterDigest     = 8 * TicksPerSecond // after swallowing a small beast whole, it chews this long
	ManEaterSnapTicks  = 6                  // the snap pose
	manEaterSpawnAway  = 30
	manEaterSeedsYield = 2
)

// pickFlower picks the flower e: the flower, and maybe a seed.
func (w *World) pickFlower(p *Player, e *Entity) bool {
	item := FlowerKind(e.Var).Item()
	if p.Inv.Room(item) < 1 {
		p.notify(NoticeBagFull, uint8(item))
		return false
	}
	p.Inv.Add(item, 1)
	if w.random().Float64() < FlowerSeedChance {
		p.Inv.Add(ItemFlowerSeed, 1)
	}
	e.Ready = false
	e.RegrowAt = w.tick + FlowerRegrowTicks
	w.watch(e)
	return true
}

// pickGlow picks spores off a glowing plant, and maybe a seed.
func (w *World) pickGlow(p *Player, e *Entity) bool {
	if p.Inv.Room(ItemGlowSpore) < GlowSporeYield {
		p.notify(NoticeBagFull, uint8(ItemGlowSpore))
		return false
	}
	p.Inv.Add(ItemGlowSpore, GlowSporeYield)
	if w.random().Float64() < GlowSeedChance {
		p.Inv.Add(ItemGlowSeed, 1)
	}
	e.Ready = false
	e.RegrowAt = w.tick + GlowRegrowTicks
	w.watch(e)
	return true
}

// growSapling turns a seedling that has come up into what it grows into.
func (w *World) growSapling(e *Entity) {
	s := SeedKind(e.Var)
	fuel := e.Fuel
	e.Var, e.Ready, e.Big, e.Fuel = 0, true, false, 0
	switch s {
	case SeedFlower:
		e.Kind = EntityFlower
		e.Var = uint8(hash2(int32(e.TX), int32(e.TY), 0xf10) % uint32(NumFlowers))
	case SeedBerry:
		e.Kind = EntityBerryBush
	case SeedTree:
		e.Kind = EntityTree
		w.youngGrows(e, fuel) // see regrow.go
	case SeedGlow:
		e.Kind = EntityGlowPlant
		e.Var = uint8(hash2(int32(e.TX), int32(e.TY), 0x6109) % 2)
	case SeedManEater:
		e.Kind = EntityManEater
		e.Var = 1 // a guard: it spares players
	}
	e.HP = maxHP(e.Kind, false)
}

// Man-eaters sit still, so they are not stepped every tick: players and
// beasts look for one in reach instead. RegrowAt is the tick it may snap
// again.

// manEaterNear returns a man-eater ready to snap at something at (x, y)
// with the given radius, sparing players when it is a planted guard.
func (w *World) manEaterNear(x, y, radius float64, player bool) *Entity {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - 2; ty <= cy+2; ty++ {
		for tx := cx - 2; tx <= cx+2; tx++ {
			e := w.entityAt(tx, ty)
			if e == nil || e.Kind != EntityManEater || w.tick < e.RegrowAt || (player && e.Var != 0) {
				continue
			}
			if math.Hypot(float64(tx)+0.5-x, float64(ty)+0.5-y) <= ManEaterReach+radius {
				return e
			}
		}
	}
	return nil
}

// manEaterBites lets a wild man-eater in reach snap at p.
func (w *World) manEaterBites(p *Player) {
	if e := w.manEaterNear(p.X, p.Y, 0, true); e != nil {
		e.RegrowAt = w.tick + ManEaterCooldown
		struck(p, ManEaterDamage)
		p.notifyPlant()
	}
}

// manEaterBitesBeast lets a man-eater in reach snap at c. A small beast (a
// rabbit, a little dinosaur) is swallowed whole; a big one is only bitten.
func (w *World) manEaterBitesBeast(c *Creature) {
	if e := w.manEaterNear(c.X, c.Y, c.Kind.def().radius, false); e != nil {
		if SwallowedWhole(c.Kind) {
			c.gone = true // eaten: nothing left to butcher
			e.RegrowAt = w.tick + ManEaterDigest
			return
		}
		e.RegrowAt = w.tick + ManEaterCooldown
		if c.HP -= ManEaterBeastBite * HealthScale; c.HP <= 0 {
			w.fell(c)
		} else if !c.Kind.def().tough {
			c.State, c.timer = CreatureHurt, CreatureHurtTicks
		}
	}
}

// notifyPlant marks the latest bite as a plant's, for the client's text.
func (p *Player) notifyPlant() { p.notify(NoticePlantBite, 0) }

// ManEaterSnapping reports whether a man-eater that may snap again at
// readyAt is in its snap pose at tick.
func ManEaterSnapping(readyAt, tick uint64) bool {
	return readyAt > tick && readyAt-tick > ManEaterCooldown-ManEaterSnapTicks
}

// cutManEater lands a blow on a man-eating plant; it needs a weapon or an
// axe, and the last blow leaves its seeds on the ground.
func (w *World) cutManEater(p *Player, e *Entity) bool {
	dmg := max(HitPower, damageWith(p)*HitPower/HealthScale)
	if t, ok := p.Inv.BestTool(ToolAxe); ok {
		dmg = max(dmg, ToolPower(t))
	}
	e.HP -= dmg
	if e.HP > 0 {
		return true
	}
	x, y := float64(e.TX)+0.5, float64(e.TY)+0.5
	w.removeEntity(e)
	w.dropAt(x, y, []Ingredient{{Item: ItemManEaterSeed, Count: manEaterSeedsYield}})
	return true
}

// GlowRadius is the light, in tiles, of a glowing plant of look v.
func GlowRadius(v uint8) float64 {
	if v == 2 {
		return GlowLightRadius * 1.6
	}
	return GlowLightRadius
}

// scatterPlants places flowers, glowing plants and man-eaters in a tribal
// world.
func (w *World) scatterPlants(rng *rand.Rand, seed uint32) {
	sx, sy := w.spawnX, w.spawnY
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			if !w.canPlace(tx, ty) {
				continue
			}
			x, y := float64(tx)+0.5, float64(ty)+0.5
			r := rng.Float64()
			far := math.Hypot(x-sx, y-sy) > manEaterSpawnAway
			var kind EntityKind
			var v uint8
			switch w.tiles[ty*w.width+tx] {
			case TileGrass:
				// Flowers grow in drifts of one colour.
				patch := fbm(x/9+5, y/9+11, seed+71, 2)
				if r < 0.012+0.05*smoothstep(clamp((patch-0.55)/0.15, 0, 1)) {
					kind = EntityFlower
					v = uint8(int(fbm(x/14+3, y/14+7, seed+72, 2)*float64(NumFlowers)*1.6) % int(NumFlowers))
				}
			case TileForest:
				switch {
				case r < 0.008:
					kind, v = EntityFlower, uint8(rng.IntN(int(NumFlowers)))
				case r < 0.008+0.006:
					kind, v = EntityGlowPlant, uint8(rng.IntN(3))
				case r < 0.008+0.006+0.0012 && far:
					kind = EntityManEater
				}
			case TileMarsh:
				switch {
				case r < 0.014:
					kind, v = EntityGlowPlant, uint8(rng.IntN(2))
				case r < 0.014+0.004 && far:
					kind = EntityManEater
				}
			case TileCave:
				if r < 0.035 {
					kind, v = EntityGlowPlant, 0
				}
			}
			if kind != EntityNone {
				e := w.addEntity(kind, tx, ty)
				e.Var = v
				e.HP = maxHP(kind, false)
			}
		}
	}
}

// SwallowedWhole reports whether a man-eating plant swallows a creature of
// kind k whole: the small beasts and the little dinosaurs.
func SwallowedWhole(k CreatureKind) bool {
	switch k {
	case CreatureRabbit, CreatureSnake, CreatureCrab, CreatureWorm, CreatureThesc, CreatureOrnitho,
		CreatureAchero, CreatureRaptor, CreatureAnzu, CreatureSpider, CreatureScorpion, CreatureToad, CreatureGlowToad:
		return true
	}
	return false
}

// ManEaterChewing reports whether a man-eater that may snap again at
// readyAt is chewing at tick: it swallowed something.
func ManEaterChewing(readyAt, tick uint64) bool {
	return readyAt > tick && readyAt-tick > ManEaterCooldown
}
