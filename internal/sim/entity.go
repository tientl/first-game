package sim

// Tile is the ground type of one map cell.
type Tile uint8

const (
	TileWater Tile = iota
	TileSand
	TileGrass
	TileForest
	TileRocky
	NumTiles
)

// Walkable reports whether players can stand on t.
func (t Tile) Walkable() bool { return t != TileWater && t < NumTiles }

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
	NumEntityKinds
)

var entityNames = [NumEntityKinds]string{
	EntityNone:      "?",
	EntityTree:      "Cây",
	EntityBoulder:   "Tảng đá",
	EntityBerryBush: "Bụi quả mọng",
	EntityGrassTuft: "Bụi cỏ",
	EntityCampfire:  "Lửa trại",
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

	HP       int    // hits left before it is harvested (trees, boulders)
	Ready    bool   // can be harvested now; false while a tree is a stump or a bush regrows
	RegrowAt uint64 // tick at which a harvested node becomes Ready again
	Fuel     int    // ticks of burning left (campfires)
}

// Interactable reports whether a player can act on an entity of this kind and
// state. hasWood matters for campfires, where the action is adding fuel. It is
// shared by the server (to pick the target) and the client (to highlight it).
func Interactable(kind EntityKind, ready, hasWood bool) bool {
	switch kind {
	case EntityTree, EntityBerryBush, EntityGrassTuft:
		return ready
	case EntityBoulder:
		return true
	case EntityCampfire:
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

func maxHP(kind EntityKind) int {
	switch kind {
	case EntityTree:
		return TreeHits
	case EntityBoulder:
		return BoulderHits
	default:
		return 1
	}
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
