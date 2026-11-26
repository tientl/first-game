package sim

import (
	"math"
	"slices"
	"sync"
)

// Free build: a dev-mode setting (on by default there) under which a player
// builds out of nothing. Every recipe is made without its ingredients and
// away from its fire or furnace, and nothing is used up; anything that can
// be set down (a wall, a campfire with a pot on it, tilled soil, a seedling,
// flowers, a head on a post…) is put on a tile straight from the build
// palette (CmdDevBuild), with nothing in hand; Shift and the right button
// take a structure away at once, giving nothing back (CmdDevRemove);
// chopping and mining take one blow and need no tool. The ground keeps its
// rules: nothing goes into the sea or onto rock, onto a tile already built
// on, or where someone stands.
//
// It only works in a world the server runs in dev mode (SetDevMode): a
// normal game can never turn it on, and a normal save never carries it.

// DevReach is how far from a player in free build a tile may be built on or
// cleared, in tiles from their centre to the tile's centre.
const DevReach = 7.0

// DevGroup is a section of the build palette.
type DevGroup uint8

const (
	DevGroupHome  DevGroup = iota // walls, doors, floors, beds, chests
	DevGroupFire                  // fires, furnaces and the cooking gear
	DevGroupFarm                  // tilled soil and seedlings
	DevGroupDecor                 // flowers, heads on posts, old bones and stones
	NumDevGroups
)

// DevGroupNames are the palette's section headings.
var DevGroupNames = [NumDevGroups]string{"Nhà cửa", "Lửa & bếp", "Trồng trọt", "Trang trí"}

// DevPiece is one thing the build palette sets down.
type DevPiece struct {
	Name   string
	Group  DevGroup
	Kind   EntityKind // what goes on the tile
	Var    uint8      // and which one of its kind
	Item   ItemKind   // the item it is in the hand, for its icon; ItemNone for a structure only a recipe builds
	Soft   bool       // only on soft ground: tilled soil, a seedling, flowers
	OnFire bool       // cooking gear: onto a bare campfire there, or with a campfire of its own
}

// devPieces is the palette, made on first use: the names of the cooking gear
// and of the heads are only filled in by the init functions of cooking.go
// and drops.go.
var devPieces = sync.OnceValue(func() []DevPiece {
	ps := []DevPiece{
		{Name: "Tường gỗ", Group: DevGroupHome, Kind: EntityWall},
		{Name: "Tường đá", Group: DevGroupHome, Kind: EntityStoneWall},
		{Name: "Cửa gỗ", Group: DevGroupHome, Kind: EntityDoor},
		{Name: "Sàn gỗ", Group: DevGroupHome, Kind: EntityFloor},
		{Name: "Hàng rào chông", Group: DevGroupHome, Kind: EntitySpikes},
		{Name: "Bẫy thú", Group: DevGroupHome, Kind: EntityTrap},
		{Name: "Giường cỏ", Group: DevGroupHome, Kind: EntityBed},
		{Name: "Rương", Group: DevGroupHome, Kind: EntityChest},
		{Name: "Lều", Group: DevGroupHome, Kind: EntityHut},

		{Name: "Lửa trại", Group: DevGroupFire, Kind: EntityCampfire},
		{Name: "Đuốc cắm", Group: DevGroupFire, Kind: EntityTorchStand, Item: ItemTorch},
		{Name: "Lò nung", Group: DevGroupFire, Kind: EntityFurnace},
		{Name: "Cối giã", Group: DevGroupFire, Kind: EntityMortar},
		{Name: "Lò nướng bánh", Group: DevGroupFire, Kind: EntityOven},
		{Name: "Giàn phơi", Group: DevGroupFire, Kind: EntityDryRack},
	}
	for g := GearPot; g < numGears; g++ {
		ps = append(ps, DevPiece{Name: GearItem(g).Def().Name, Group: DevGroupFire, Kind: EntityCampfire, Var: uint8(g), Item: GearItem(g), OnFire: true})
	}
	ps = append(ps, DevPiece{Name: "Luống đất", Group: DevGroupFarm, Kind: EntityFarmland, Soft: true})
	for k := ItemFlowerSeed; k <= ItemManEaterSeed; k++ {
		ps = append(ps, DevPiece{Name: k.Def().Name, Group: DevGroupFarm, Kind: EntitySapling, Var: uint8(seedOf(k)), Item: k, Soft: true})
	}
	for f := FlowerRed; f < NumFlowers; f++ {
		ps = append(ps, DevPiece{Name: f.Item().Def().Name, Group: DevGroupDecor, Kind: EntityFlower, Var: uint8(f), Item: f.Item(), Soft: true})
	}
	for _, c := range []CreatureKind{CreatureRaptor, CreatureTrike, CreatureRex} {
		ps = append(ps, DevPiece{Name: HeadItem(c).Def().Name, Group: DevGroupDecor, Kind: EntityTrophy, Var: uint8(c), Item: HeadItem(c)})
	}
	ps = append(ps,
		DevPiece{Name: "Cột vật tổ", Group: DevGroupDecor, Kind: EntityTotem},
		DevPiece{Name: "Lửa làng", Group: DevGroupDecor, Kind: EntityBonfire},
		DevPiece{Name: "Đống xương", Group: DevGroupDecor, Kind: EntityLairProp, Var: LairBones},
		DevPiece{Name: "Đá cổ rêu", Group: DevGroupDecor, Kind: EntityLairProp, Var: LairRuin},
		DevPiece{Name: "Gốc cây gãy", Group: DevGroupDecor, Kind: EntityLairProp, Var: LairStump},
		DevPiece{Name: "Da rắn lột", Group: DevGroupDecor, Kind: EntityLairProp, Var: LairSkin},
	)
	if len(ps) > math.MaxUint8 {
		panic("sim: the build palette has more pieces than a command can name")
	}
	return ps
})

// DevPieces is the build palette, in its order; a piece's index in it is
// what CmdDevBuild names.
func DevPieces() []DevPiece { return slices.Clone(devPieces()) }

// DevPieceAt is piece i of the build palette.
func DevPieceAt(i int) (DevPiece, bool) {
	if i < 0 || i >= len(devPieces()) {
		return DevPiece{}, false
	}
	return devPieces()[i], true
}

// DevRemovable reports whether free build may take away an entity of kind
// k: anything on one tile but the fixed sights of the world (the pyramid, a
// cave's mouth, a giant of the old forest) and what holds a player's things
// (a fallen player's body, an old chest of treasure).
func DevRemovable(k EntityKind) bool {
	switch k {
	case EntityNone, EntityPyramid, EntityPyramidStone, EntityCaveMouth, EntityCaveExit, EntityCaveRock,
		EntityGiantTree, EntityGiantRoot, EntityGrave, EntityTreasure:
		return false
	}
	return k < NumEntityKinds
}

// IsDevCommand reports whether k is a command only a dev-mode server runs.
func IsDevCommand(k CommandKind) bool { return k == CmdDevBuild || k == CmdDevRemove }

// DevBuildSpot is BuildSpot for palette piece pc, within DevReach, shared by
// the server and the client's ghost: cooking gear may also go onto a bare
// campfire on the tile. at reports the entity on a tile.
func DevBuildSpot(px, py float64, tx, ty int, pc DevPiece, tileAt func(tx, ty int) Tile, at func(tx, ty int) (EntityKind, uint8, bool), standers [][2]float64) (ok bool, why NoticeKind) {
	if pc.OnFire {
		if k, v, ok := at(tx, ty); ok && k == EntityCampfire && v == 0 {
			if dx, dy := float64(tx)+0.5-px, float64(ty)+0.5-py; dx*dx+dy*dy > DevReach*DevReach {
				return false, NoticeTooFar
			}
			return true, NoticeNone
		}
	}
	taken := func(tx, ty int) bool {
		_, _, ok := at(tx, ty)
		return ok
	}
	return BuildSpotWithin(DevReach, px, py, tx, ty, pc.Kind, pc.Soft, tileAt, taken, standers)
}

// --- the setting ------------------------------------------------------------

// SetDevMode tells the world whether its server runs in dev mode; only then
// can free build be turned on, and only then does a save keep it.
func (w *World) SetDevMode(on bool) { w.dev.on = on }

// DevMode reports whether the world is run in dev mode.
func (w *World) DevMode() bool { return w.dev.on }

// DevSetFreeBuild turns free build on or off for player id. It reports
// false, changing nothing, outside dev mode or for an unknown player.
func (w *World) DevSetFreeBuild(id PlayerID, on bool) bool {
	p := w.devPlayer(id)
	if p == nil || !w.dev.on {
		return false
	}
	p.dev.build, p.dev.buildSet = on, true
	return true
}

// DevFreeBuild reports whether player id builds freely: in dev mode, with
// free build on.
func (w *World) DevFreeBuild(id PlayerID) bool {
	p := w.devPlayer(id)
	return p != nil && w.freeBuild(p)
}

// DevFreeBuildChosen reports whether player id has had free build turned on
// or off, in this session or in the dev save; a player who has not gets it
// on when joining a dev world.
func (w *World) DevFreeBuildChosen(id PlayerID) bool {
	p := w.devPlayer(id)
	return p != nil && p.dev.buildSet
}

// freeBuild reports whether p builds freely.
func (w *World) freeBuild(p *Player) bool { return w.dev.on && p.dev.build }

// placeReach is how far from p a tile may be built on.
func (w *World) placeReach(p *Player) float64 {
	if w.freeBuild(p) {
		return DevReach
	}
	return PlaceReach
}

// DevSave is a player's dev-mode settings, kept only in the save of a dev
// world.
type DevSave struct {
	FreeBuild bool
}

// --- building and clearing ------------------------------------------------------

// devAt is the kind and variant of the entity on tile (tx, ty), for DevBuildSpot.
func (w *World) devAt(tx, ty int) (EntityKind, uint8, bool) {
	if !w.inBounds(tx, ty) {
		return EntityNone, 0, false
	}
	e := w.entityAt(tx, ty)
	if e == nil {
		return EntityNone, 0, false
	}
	return e.Kind, e.Var, true
}

// standers are where the living players stand.
func (w *World) standers() [][2]float64 {
	var out [][2]float64
	for _, id := range w.playerOrder {
		if o := w.players[id]; !o.Dead {
			out = append(out, [2]float64{o.X, o.Y})
		}
	}
	return out
}

// devBuild puts palette piece i on tile (tx, ty) for p, when at is set, else
// on the tile p faces; nothing is needed and nothing is used up. It does
// nothing unless p builds freely.
func (w *World) devBuild(p *Player, i int, at bool, tx, ty int) {
	pc, ok := DevPieceAt(i)
	if !ok || !w.freeBuild(p) {
		return
	}
	if !at {
		dx, dy := p.Dir.Delta()
		tx, ty = int(math.Floor(p.X))+dx, int(math.Floor(p.Y))+dy
	}
	if ok, why := DevBuildSpot(p.X, p.Y, tx, ty, pc, w.TileAt, w.devAt, w.standers()); !ok {
		p.notify(why, 0)
		return
	}
	if !pc.OnFire && !w.canPlace(tx, ty) {
		p.notify(NoticeNoSpace, 0)
		return
	}
	p.Dir = dirToward(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y)
	w.devPut(pc, tx, ty)
}

// devPut sets piece pc down on tile (tx, ty), which DevBuildSpot passed.
func (w *World) devPut(pc DevPiece, tx, ty int) *Entity {
	if pc.OnFire {
		e := w.entityAt(tx, ty)
		if e == nil || e.Kind != EntityCampfire || e.Var != 0 {
			e = w.addEntity(EntityCampfire, tx, ty)
		}
		e.Var = pc.Var
		w.station(e.ID)
		return e
	}
	if pc.Kind == EntityFarmland {
		return w.till(tx, ty)
	}
	e := w.addEntity(pc.Kind, tx, ty)
	e.Var = pc.Var
	switch pc.Kind {
	case EntityTorchStand:
		e.Fuel = TorchStandTicks
	case EntitySapling:
		e.Ready = false
		e.RegrowAt = w.tick + SeedKind(pc.Var).growTicks()
	}
	w.watch(e)
	return e
}

// devRemove takes the entity on tile (tx, ty) away at once for p, when at
// is set, else the one on the tile p faces, giving nothing back: no wood,
// no pot off the fire. What a chest or a cooking station holds is a
// player's own and spills on the ground. It does nothing unless p builds
// freely.
func (w *World) devRemove(p *Player, at bool, tx, ty int) {
	if !w.freeBuild(p) {
		return
	}
	if !at {
		dx, dy := p.Dir.Delta()
		tx, ty = int(math.Floor(p.X))+dx, int(math.Floor(p.Y))+dy
	}
	if !w.inBounds(tx, ty) {
		p.notify(NoticeNothingNear, 0)
		return
	}
	e := w.entityAt(tx, ty)
	if e == nil || !DevRemovable(e.Kind) {
		p.notify(NoticeNothingNear, 0)
		return
	}
	if dx, dy := float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y; dx*dx+dy*dy > DevReach*DevReach {
		p.notify(NoticeTooFar, 0)
		return
	}
	if e.Kind == EntityCampfire {
		e.Var = uint8(GearNone) // the pot goes with the fire
	}
	if inv := w.chests[e.ID]; inv != nil {
		var items []Ingredient
		for _, s := range inv {
			if !s.Empty() {
				items = append(items, Ingredient{Item: s.Item, Count: s.Count})
			}
		}
		if len(items) > 0 {
			w.dropAt(float64(tx)+0.5, float64(ty)+0.5, items)
		}
	}
	p.Dir = dirToward(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y)
	w.removeEntity(e)
}
