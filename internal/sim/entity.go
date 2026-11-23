package sim

// Tile is the ground type of one map cell.
type Tile uint8

const (
	TileWater Tile = iota
	TileSand
	TileGrass
	TileForest
	TileRocky
	TileDesert   // open world: hot sand dunes
	TileMarsh    // open world: swamp mud, slow going
	TileMountain // open world: rock face, impassable
	TileCave     // open world: a tunnel floor inside a mountain, always dark
	TileAsh      // open world: volcanic ash around the volcano
	TileLava     // open world: impassable, glows, warms whoever stands near
	TileSnow     // open world: the frozen north, freezing to stand on; see iceage.go
	NumTiles
)

// Walkable reports whether players can stand on t.
func (t Tile) Walkable() bool {
	switch t {
	case TileWater, TileMountain, TileLava:
		return false
	}
	return t < NumTiles
}

// EntityID identifies a static world entity (resource node or structure).
type EntityID uint32

// EntityKind is the type of a static world entity.
type EntityKind uint8

const (
	EntityNone EntityKind = iota
	EntityTree
	EntityBoulder
	EntityBerryBush
	EntityGrassTuft
	EntityCampfire
	EntityTwig     // a fallen branch, picked up by hand
	EntityWall     // built: blocks the way; an axe takes it down
	EntityBed      // built: sleep the night through
	EntityPebble   // a small stone on the ground, picked up by hand
	EntityFarmland // soil tilled with a hoe, ready for planting
	EntityCrop     // a sweet potato growing in tilled soil; Ready when ripe
	EntityBurrow   // a rabbit burrow; see homes.go
	EntityDen      // a snake den
	EntityCactus   // desert: needs an axe, gives cactus flesh
	EntityReed     // marsh: picked by hand, gives dry grass
	EntityMushroom // marsh and caves: picked by hand, eaten
	EntityCoal     // caves and ash: a coal seam, needs a pickaxe

	// Ores, each in its own region, and the furnace that smelts them: see ores.go.
	EntityCopperOre
	EntityIronOre
	EntityPlatinumOre
	EntityDiamondOre
	EntityFurnace   // built: smelts ore while it burns
	EntityClam      // open world: clams on the sand by the sea, picked by hand
	EntityStoneWall // built: blocks the way; a pickaxe takes it down
	EntityChest     // built: stores things; see chest.go
	EntityFigTree   // fruit trees, picked by hand: see fruit.go
	EntityGinkgoTree
	EntityPalm
	NumEntityKinds
)

var entityNames = [NumEntityKinds]string{
	EntityNone:      "?",
	EntityTree:      "Cây",
	EntityBoulder:   "Tảng đá",
	EntityBerryBush: "Bụi quả mọng",
	EntityGrassTuft: "Bụi cỏ",
	EntityCampfire:  "Lửa trại",
	EntityTwig:      "Cành cây",
	EntityWall:      "Tường gỗ",
	EntityBed:       "Giường cỏ",
	EntityPebble:    "Đá cuội",
	EntityFarmland:  "Luống đất",
	EntityCrop:      "Khoai lang",
	EntityBurrow:    "Hang thỏ",
	EntityDen:       "Hang rắn",
	EntityCactus:    "Xương rồng",
	EntityReed:      "Lau sậy",
	EntityMushroom:  "Nấm",
	EntityCoal:      "Mỏ than",

	EntityCopperOre:   "Mạch đá lửa",
	EntityIronOre:     "Mạch ngọc bích",
	EntityPlatinumOre: "Mạch mã não",
	EntityDiamondOre:  "Mạch hắc diện thạch",
	EntityFurnace:     "Lò nung",
	EntityClam:        "Bãi sò",
	EntityStoneWall:   "Tường đá",
	EntityChest:       "Rương",
	EntityFigTree:     "Cây sung",
	EntityGinkgoTree:  "Cây bạch quả",
	EntityPalm:        "Cây dừa",
}

// Name is the display name of k.
func (k EntityKind) Name() string {
	if k >= NumEntityKinds {
		return entityNames[EntityNone]
	}
	return entityNames[k]
}

// Entity is a static thing that occupies exactly one tile: a tree, a boulder,
// a bush, a campfire. At most one entity sits on a tile.
type Entity struct {
	ID     EntityID
	Kind   EntityKind
	TX, TY int

	Big      bool   // the big size of a tree or boulder: more hits, more yield
	HP       int    // damage left before it is harvested, in HitPower units per hit
	Ready    bool   // can be harvested now; false while a tree is a stump or a crop grows
	RegrowAt uint64 // tick at which a harvested node becomes Ready again; for tilled soil, when it turns back to grass
	Fuel     int    // ticks of burning left (campfires, furnaces)
}

// Interactable reports whether a player can act on an entity of this kind and
// state. hasWood matters for campfires, where the action is adding fuel. It is
// shared by the server (to pick the target) and the client (to highlight it).
func Interactable(kind EntityKind, ready, hasWood bool) bool {
	switch kind {
	case EntityCactus, EntityReed, EntityMushroom, EntityClam:
		return ready
	case EntityCoal, EntityCopperOre, EntityIronOre, EntityPlatinumOre, EntityDiamondOre:
		return true
	case EntityTree, EntityBerryBush, EntityGrassTuft, EntityTwig, EntityPebble, EntityCrop,
		EntityFigTree, EntityGinkgoTree, EntityPalm:
		return ready
	case EntityBoulder, EntityWall, EntityStoneWall, EntityBed, EntityFarmland, EntityChest:
		return true
	case EntityCampfire, EntityFurnace:
		return hasWood
	default:
		return false
	}
}

// FireRadius is how far, in tiles, a campfire with the given fuel lights and
// warms. The client draws its light with the same radius.
func FireRadius(fuel int) float64 {
	frac := min(1, max(0, float64(fuel)/FireStartFuel))
	return 1.5 + 3.5*frac
}

// MaxFireRadius is FireRadius at full fuel.
const MaxFireRadius = 5.0

// Sized reports whether entities of kind come in a small and a big size.
func (k EntityKind) Sized() bool { return k == EntityTree || k == EntityBoulder }

// maxHP is the HP of a fresh entity: its hits with a stone tool (or by hand)
// times HitPower.
func maxHP(kind EntityKind, big bool) int { return hits(kind, big) * HitPower }

// hits is how many hits by hand or with a stone tool harvest an entity.
func hits(kind EntityKind, big bool) int {
	if o, ok := oreOf(kind); ok {
		return o.hits
	}
	switch kind {
	case EntityTree:
		return bySize(big, TreeHits, BigTreeHits)
	case EntityBoulder:
		return bySize(big, BoulderHits, BigBoulderHits)
	case EntityWall:
		return WallHits
	case EntityStoneWall:
		return StoneWallHits
	case EntityCactus:
		return CactusHits
	case EntityCoal:
		return CoalHits
	default:
		return 1
	}
}

func bySize(big bool, small, large int) int {
	if big {
		return large
	}
	return small
}

// Dir is the direction a player faces.
type Dir uint8

const (
	DirDown Dir = iota
	DirUp
	DirLeft
	DirRight
)

// Delta returns the unit tile offset of d.
func (d Dir) Delta() (dx, dy int) {
	switch d {
	case DirUp:
		return 0, -1
	case DirLeft:
		return -1, 0
	case DirRight:
		return 1, 0
	default:
		return 0, 1
	}
}

// dirToward returns the facing that best points along (dx, dy). Horizontal
// wins on a tie so diagonal movement shows the side view.
func dirToward(dx, dy float64) Dir {
	ax, ay := dx, dy
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	switch {
	case ax >= ay && dx < 0:
		return DirLeft
	case ax >= ay && dx > 0:
		return DirRight
	case dy < 0:
		return DirUp
	default:
		return DirDown
	}
}
