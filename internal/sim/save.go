package sim

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// SaveVersion is bumped on every change to what a save means. Load still
// reads older versions and upgrades them.
//
//   - 1: first version.
//   - 2: trees need an axe; fallen branches (twigs) exist.
//   - 3: trees and boulders are small or big.
//   - 4: the inventory is slots (hotbar and bag); the held slot is kept.
//   - 5: pebbles lie on the ground; boulders need a pickaxe; tilled soil and crops.
//   - 6: entity HP counts in HitPower units (tools of each tier hit harder);
//     ores and furnaces exist.
//   - 7: a day lasts twice as long (8 minutes); older ticks are doubled.
//   - 8: tamed beasts are saved with their owners' names.
//   - 9: fruit trees grow wild.
//   - 10: things lie on the ground; players come back by their last bed.
//   - 11: a beast's health counts in HealthScale units (weapons do damage, not hits).
//   - 12: the bosses: which have fallen, who has seen their lairs, the victory.
//   - 13: a day lasts 16 minutes and a season 12 days; older ticks are doubled.
//   - 14: the bag has 30 slots (it had 20), and so do chests; older inventories
//     load as they were, with the new slots empty. An old chest in a cave
//     opened like a chest; one opened before spilled its hoard already.
//   - 15: the bosses' lairs are dressed (see lair.go); older worlds get theirs on loading.
//   - 16: cooking: clay beds by the water, pots on fires (a campfire's Var),
//     what stations hold, and the buffs of dishes.
//   - 17: the wild plants of flora.go (some stand in the water) and crops of
//     four kinds; an older world keeps the plants a fresh one would grow.
//   - 18: everyone starts in a fur loincloth and clothes are things carried
//     (clothes.go); a player of an older save is given the clothes they wore.
//   - 19: worlds of StyleIsles: designed islands with shallows, cliffs, ramps,
//     rivers and fords, and the 2048-tile size. Nothing older changes, but an
//     older game would make the wrong world of such a save, so it must refuse it.
//   - 20: worlds of StyleWoods, whose trees are spaced by their crowns, thick
//     and ordinary stretches, trails and undergrowth (woods.go); an older save
//     has its woods thinned once on loading, and its hives to its trees
//     (thinwoods.go), sparing what players planted, built by or robbed. A
//     felled tree grows back through a sapling and a young tree (regrow.go):
//     a tree sapling's Fuel is 1 when it will be a grown tree, and a young
//     tree's RegrowAt is when it is grown (older saves' Ready trees drop
//     theirs).
//   - 21: worlds of StyleBeaches, with beaches five to twelve tiles deep, dunes
//     and driftwood (beaches.go). Nothing older changes, but an older game
//     would make the wrong world of such a save, so it must refuse it.
//   - 22: the Ankylosaurus (ankylo.go): a save may hold a tamed one, and its
//     bony plates in a bag or a chest, which an older game would lose.
//     Nothing older changes.
//   - 23: worlds of StyleWilds: fewer, bigger islands, and the animals laid
//     out by island and by distance from home (wilds.go). Nothing older
//     changes, but an older game would make the wrong world of such a save,
//     so it must refuse it.
const SaveVersion = 23

// v6DayTicks is how long a day was in saves before version 7, and
// v12DayTicks before version 13.
const (
	v6DayTicks  = 240 * TicksPerSecond
	v12DayTicks = 480 * TicksPerSecond
)

// SaveData is everything needed to rebuild a world later. The terrain is not
// stored: it is regenerated from the seed, which always yields the same map.
type SaveData struct {
	Version       int
	Seed          uint64
	Width, Height int
	Tick          uint64
	NextEntity    EntityID
	Entities      []Entity
	Players       []PlayerSave // online and offline, matched to joiners by name
	Style         WorldStyle   `json:",omitempty"` // which generator made the terrain
	Chests        []ChestSave  `json:",omitempty"` // what each chest holds
	Pets          []PetSave    `json:",omitempty"` // tamed beasts; wild ones are not saved
	Drops         []Drop       `json:",omitempty"` // piles on the ground
	Boats         []BoatSave   `json:",omitempty"` // boats on the water
	Difficulty    Difficulty   `json:",omitempty"` // absent: normal; see difficulty.go
	Bosses        *BossSave    `json:",omitempty"` // the bosses fallen, and the victory; see bosses.go
	Stations      []CookSave   `json:",omitempty"` // what cooking stations hold; see cooking.go
}

// PlayerSave is the persistent part of a player.
type PlayerSave struct {
	Name           string
	Look           Look `json:",omitzero"` // zero before looks existed: the original survivor
	X, Y           float64
	Dir            Dir
	Health, Hunger float64
	Slots          Inventory
	Selected       int    // held hotbar slot
	Explored       []byte `json:",omitempty"` // tiles seen, as Explored.Bytes
	Dead           bool
	TorchFuel      int      `json:",omitempty"` // ticks left on the torch being burnt
	Home           EntityID `json:",omitempty"` // the last bed slept in
	Buffs          []int    `json:",omitempty"` // ticks left of each buff, by BuffKind; see cooking.go

	// Inv is how saves before version 4 kept the inventory: a count per item
	// kind, indexed by ItemKind. Load turns it into Slots.
	Inv []int `json:",omitempty"`
}

func savePlayer(p *Player) PlayerSave {
	return PlayerSave{
		Name: p.Name, Look: p.Look, X: p.X, Y: p.Y, Dir: p.Dir,
		Health: p.Health, Hunger: p.Hunger, Slots: p.Inv, Selected: p.Selected, Explored: p.explored.Bytes(), Dead: p.Dead,
		TorchFuel: p.torchFuel, Home: p.home, Buffs: saveBuffs(p),
	}
}

// legacyInventory lays out a count-per-kind inventory from an old save in
// slots, each kind capped at what one slot holds as the old game capped it.
func legacyInventory(counts []int) Inventory {
	var inv Inventory
	for k := ItemKind(1); k < NumItems && int(k) < len(counts); k++ {
		inv.Add(k, min(k.MaxCount(), max(0, counts[k])))
	}
	return inv
}

// restorePlayer applies a saved state to p, keeping p at the spawn if the
// saved position is not somewhere a player can stand.
func (w *World) restorePlayer(p *Player, ps PlayerSave) {
	p.Look = ps.Look.Sanitized()
	if finite(ps.X, ps.Y) && w.standable(ps.X, ps.Y) {
		p.X, p.Y = ps.X, ps.Y
	} else if finite(ps.X, ps.Y) {
		// Saved at sea: back aboard the boat that is still there.
		for _, b := range w.boats {
			if b.Rider == 0 && math.Hypot(b.X-ps.X, b.Y-ps.Y) < 0.5 {
				p.X, p.Y, p.boat, b.Rider = b.X, b.Y, b.ID, p.ID
				break
			}
		}
	}
	if ps.Dir <= DirRight {
		p.Dir = ps.Dir
	}
	if finite(ps.Health, ps.Hunger) {
		p.Health = min(MaxHealth, max(0, ps.Health))
		p.Hunger = min(MaxHunger, max(0, ps.Hunger))
	}
	p.Inv = ps.Slots
	p.Inv.sanitize()
	p.explored = ExploredFrom(w.width, w.height, ps.Explored)
	p.torchFuel = min(TorchBurnTicks, max(0, ps.TorchFuel))
	loadBuffs(p, ps.Buffs)
	if b := w.entities[ps.Home]; b != nil && b.Kind == EntityBed {
		p.home = ps.Home
	}
	if IsHotbar(ps.Selected) {
		p.Selected = ps.Selected
	}
	p.Dead = ps.Dead || p.Health <= 0
}

func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// Save captures the world's state. Players currently connected are saved as
// they are now; players who left earlier keep the state they left with.
func (w *World) Save() SaveData {
	d := SaveData{
		Version:    SaveVersion,
		Seed:       w.seed,
		Width:      w.width,
		Height:     w.genH,
		Tick:       w.tick,
		Style:      w.style,
		Difficulty: w.difficulty,
	}
	d.NextEntity = w.nextEnt
	d.Chests = w.saveChests()
	d.Pets = w.savePets() // see tame.go
	d.Boats = w.saveBoats()
	d.Bosses = w.saveBosses()
	d.Stations = w.saveStations()
	for _, dr := range w.drops {
		if !dr.gone {
			d.Drops = append(d.Drops, *dr)
		}
	}
	for _, e := range w.entities {
		d.Entities = append(d.Entities, *e)
	}
	slices.SortFunc(d.Entities, func(a, b Entity) int { return int(a.ID) - int(b.ID) })

	online := make(map[string]bool)
	for _, id := range w.playerOrder {
		p := w.players[id]
		online[p.Name] = true
		d.Players = append(d.Players, savePlayer(p))
	}
	for name, ps := range w.offline {
		if !online[name] {
			d.Players = append(d.Players, ps)
		}
	}
	slices.SortFunc(d.Players, func(a, b PlayerSave) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		}
		return 0
	})
	return d
}

// ErrBadSave is wrapped by every error Load returns for unusable data.
var ErrBadSave = errors.New("sim: bad save data")

// Load rebuilds a world from saved data. The terrain is regenerated from the
// seed; entities that do not fit it (off the map, on water, on top of another
// entity) are dropped rather than failing the whole load.
func Load(d SaveData) (*World, error) {
	if d.Version < 1 || d.Version > SaveVersion {
		return nil, fmt.Errorf("%w: version %d, this game reads 1 to %d", ErrBadSave, d.Version, SaveVersion)
	}
	w, err := Generate(GenConfig{Seed: d.Seed, Width: d.Width, Height: d.Height, Style: d.Style, Difficulty: d.Difficulty})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadSave, err)
	}
	// Before version 7 a day was shorter. Every tick in the save is scaled up,
	// so the world keeps its day number and time of day, and each regrowth
	// keeps its share of the time it had left.
	ticks := uint64(1)
	if d.Version < 13 {
		ticks = DayTicks / v12DayTicks
	}
	if d.Version < 7 {
		ticks = DayTicks / v6DayTicks
	}
	w.tick = max(d.Tick*ticks, StartTick)

	// Worlds saved before a kind of ground litter existed keep the freshly
	// generated ones: version 1 had no twigs, versions before 5 no pebbles.
	var fresh []Entity
	for _, e := range w.entities {
		if e.Kind == EntityTwig && d.Version < 2 || e.Kind == EntityPebble && d.Version < 5 ||
			IsFruitTree(e.Kind) && d.Version < 9 || e.Kind == EntityClay && d.Version < 16 || e.Kind == EntityFlora && d.Version < 17 {
			fresh = append(fresh, *e)
		}
	}
	// Before version 3 trees and boulders had one size. Each takes the size
	// the regenerated world gives the same kind on its tile, and keeps the
	// share of its hits it had left.
	var bigAt map[int]EntityKind
	if d.Version < 3 {
		bigAt = make(map[int]EntityKind)
		for _, e := range w.entities {
			if e.Big {
				bigAt[e.TY*w.width+e.TX] = e.Kind
			}
		}
	}
	// Before version 20 the woods were as thick as they were made, and are
	// thinned once (see thinwoods.go). Where each tree of the world as it was
	// made stood tells the wild trees from the planted ones.
	var made map[EntityID][2]int
	if d.Version < 20 {
		made = make(map[EntityID][2]int)
		for _, e := range w.entities {
			if e.Kind == EntityTree {
				made[e.ID] = [2]int{e.TX, e.TY}
			}
		}
	}
	clear(w.entities)
	clear(w.active)
	clear(w.occ)
	w.nextEnt = d.NextEntity
	for _, e := range d.Entities {
		if e.ID == 0 || e.Kind == EntityNone || e.Kind >= NumEntityKinds ||
			!w.canHold(e.Kind, e.Var, e.TX, e.TY) || w.entities[e.ID] != nil {
			continue
		}
		if d.Version < 3 {
			e.Big = bigAt[e.TY*w.width+e.TX] == e.Kind
		}
		e.Big = e.Big && e.Kind.Sized()
		e.RegrowAt *= ticks
		if d.Version < 20 && e.Kind == EntityTree && e.Ready {
			e.RegrowAt = 0 // left over from regrowing; now it would mean a young tree growing up (regrow.go)
		}
		switch old := v2MaxHP(e.Kind); {
		case d.Version < 3 && old > 0:
			e.HP = (e.HP*maxHP(e.Kind, e.Big) + old - 1) / old // keep the share of hits left
		case d.Version < 6:
			e.HP *= HitPower // HP counted whole hits before version 6
		}
		e.HP = min(max(e.HP, 1), maxHP(e.Kind, e.Big))
		if IsFruitTree(e.Kind) {
			e.HP = maxHP(e.Kind, false) // a few blows at it are not kept: older saves hold the one hit of picking
		}
		switch e.Kind {
		case EntityGrave:
			if d.Version < 18 {
				e.Fuel |= int(WearAll) << graveWearShift // they fell dressed
			}
		case EntityFlora:
			if e.Var >= uint8(NumFlora) {
				continue
			}
			e.HP = FloraKind(e.Var).def().hp() // a blow or two at a bamboo is not kept
		case EntityCrop:
			if e.Var >= uint8(NumCrops) {
				e.Var = 0
			}
		case EntityCampfire:
			if e.Var >= uint8(numGears) {
				e.Var = 0
			}
			if e.Fuel <= 0 && e.Var == 0 {
				continue // only a hearth with a pot on it stays, cold
			}
			e.Fuel = min(max(e.Fuel, 0), FireMaxFuel)
		case EntityFurnace:
			e.Fuel = min(max(e.Fuel, 0), FurnaceMaxFuel)
		case EntityOven:
			e.Fuel = min(max(e.Fuel, 0), OvenMaxFuel)
		}
		ent := e
		w.entities[e.ID] = &ent
		w.occ[e.TY*w.width+e.TX] = e.ID
		w.nextEnt = max(w.nextEnt, e.ID)
	}
	w.loadChests(d.Chests)
	w.loadStations(d.Stations)
	if d.Version < 15 {
		w.dressLairs() // the bones, the nest and the ruin of each lair: see lair.go
	}
	if d.Version < 20 {
		w.thinWoods(made)
	}
	for _, e := range fresh {
		if w.canHold(e.Kind, e.Var, e.TX, e.TY) {
			w.nextEnt++
			e.ID = w.nextEnt
			ent := e
			w.entities[e.ID] = &ent
			w.occ[e.TY*w.width+e.TX] = e.ID
		}
	}
	for _, ps := range d.Players {
		if d.Version < 4 {
			ps.Slots, ps.Selected = legacyInventory(ps.Inv), 0
		}
		ps.Inv = nil
		if d.Version < 2 && ps.Slots.Count(ItemAxe) == 0 {
			ps.Slots.Add(ItemAxe, 1) // they could already chop: keep that
		}
		if d.Version < 5 && ps.Slots.Count(ItemPickaxe) == 0 {
			ps.Slots.Add(ItemPickaxe, 1) // they could already mine: keep that
		}
		if d.Version < 18 {
			ps.Slots.dress() // they were dressed already: keep that
		}
		if name := cleanName(ps.Name); name != "" {
			ps.Name = name
			w.offline[name] = ps
		}
	}
	if d.Version < 11 {
		for i := range d.Pets {
			d.Pets[i].HP *= HealthScale
		}
	}
	w.loadPets(d.Pets)
	w.loadBoats(d.Boats)
	w.loadBosses(d.Bosses) // the fallen stay fallen
	for _, dr := range d.Drops {
		if !dr.Item.Valid() || dr.Count <= 0 || !finite(dr.X, dr.Y, dr.FromX, dr.FromY) {
			continue
		}
		dr.Count = min(dr.Count, dr.Item.MaxCount())
		dr.Born = min(dr.Born, w.tick)
		w.nextDrop++
		dr.ID = w.nextDrop
		x := dr
		w.drops = append(w.drops, &x)
	}
	w.watchAll() // regrowing plants, fires and soil from the save keep going
	return w, nil
}

// v2MaxHP is the hits a fresh tree or boulder had in saves before version 3,
// zero for kinds whose hits did not change.
func v2MaxHP(kind EntityKind) int {
	switch kind {
	case EntityTree:
		return 3
	case EntityBoulder:
		return 4
	default:
		return 0
	}
}
