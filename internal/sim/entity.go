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
	TilePath     // tribal worlds: a beaten trail, quicker going; see paths.go
	TileIce      // never in a world: what the client draws frozen ponds as in winter
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
	EntityFigTree   // fruit trees, picked by hand, felled bare: see fruit.go
	EntityGinkgoTree
	EntityPalm

	// The tribal age: see drops.go, build.go, plants.go, tribe.go, cavelife.go.
	EntityCorpse     // a kill lying in its blood; Var is its CreatureKind; butchered for its loot
	EntityTrophy     // a head on a post; Var is the CreatureKind
	EntityDoor       // built: players pass, beasts do not
	EntityFloor      // built: wooden planks on the ground
	EntitySpikes     // built: a fence of sharpened stakes that wounds beasts
	EntityTrap       // built: a snare that holds a beast; Ready while set
	EntityTorchStand // a torch stuck in the ground; Fuel burns down
	EntityFlower     // wild flowers; Var is the FlowerKind
	EntitySapling    // something planted from a seed; Var is the SeedKind
	EntityGlowPlant  // glows at night; Var is its look (0 mushrooms, 1 fern, 2 tree)
	EntityManEater   // a man-eating plant; Var 1 when planted by a player (it spares players)
	EntityTreasure   // an old chest deep in a cave; Ready until it is opened
	EntityHut        // a tribal hut of wood and thatch
	EntityTotem      // the carved post at a village's heart
	EntityBonfire    // a village fire that never goes out

	// The desert island's pyramid: see pyramid.go.
	EntityPyramid      // the pyramid itself, standing on the middle of its south face
	EntityPyramidStone // the rest of its foot: blocks the way, drawn as part of the pyramid

	// The caves under the archipelago: see underground.go.
	EntityCaveMouth // a mouth of rock on the ground: walk in to go down
	EntityCaveExit  // the way up, under a shaft of daylight: walk in to go up
	EntityCaveRock  // the rest of a cave mouth's rocks: blocks the way, drawn with the mouth

	// The woods: see jungle.go.
	EntityBananaTree // a fruit tree: bunches of bananas, picked by hand
	EntityGiantTree  // a giant of the old forest, standing on the middle of its front edge
	EntityGiantRoot  // the rest of a giant's foot: blocks the way, drawn with the giant

	// Death at the harder difficulties: see grave.go.
	EntityGrave // a fallen player's body, holding what they carried

	// A boss's lair: see lair.go.
	EntityLairProp // part of a lair's scene; Var is which (LairNest...); nothing stops at it

	// The new plants: see flora.go.
	EntityFlora // a wild plant picked for what it gives; Var is its FloraKind
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

	EntityCorpse:     "Xác",
	EntityTrophy:     "Đầu thú",
	EntityDoor:       "Cửa gỗ",
	EntityFloor:      "Sàn gỗ",
	EntitySpikes:     "Hàng rào chông",
	EntityTrap:       "Bẫy thú",
	EntityTorchStand: "Đuốc cắm",
	EntityFlower:     "Hoa",
	EntitySapling:    "Cây non",
	EntityGlowPlant:  "Cây phát sáng",
	EntityManEater:   "Cây ăn thịt người",
	EntityTreasure:   "Rương cổ",
	EntityHut:        "Lều thổ dân",
	EntityTotem:      "Cột vật tổ",
	EntityBonfire:    "Lửa làng",

	EntityPyramid:      "Kim tự tháp",
	EntityPyramidStone: "Kim tự tháp",
	EntityCaveMouth:    "Cửa hang",
	EntityCaveExit:     "Lối ra",
	EntityCaveRock:     "Cửa hang",

	EntityBananaTree: "Cây chuối",
	EntityGiantTree:  "Cây cổ thụ khổng lồ",
	EntityGiantRoot:  "Cây cổ thụ khổng lồ",

	EntityGrave:    "Xác người chơi",
	EntityLairProp: "Dấu vết của trùm",
	EntityFlora:    "Cây dại",
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
	Fuel     int    // ticks of burning left (campfires, furnaces, standing torches); a man-eater's bite cooldown
	Var      uint8  `json:",omitempty"` // which one of a kind: a corpse's or trophy's creature, a flower's colour, a seedling's seed; on a tree, 1 if bees nest in it; on a fruit tree, 1 while it is a stump
	HoneyAt  uint64 `json:",omitempty"` // a beehive: the tick its honey is back
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
		EntityFigTree, EntityGinkgoTree, EntityPalm, EntityBananaTree, EntityGiantTree:
		return ready
	case EntityBoulder, EntityWall, EntityStoneWall, EntityBed, EntityFarmland, EntityChest:
		return true
	case EntityCampfire, EntityFurnace:
		return hasWood
	case EntityCorpse, EntityTrophy, EntityDoor, EntityFloor, EntitySpikes, EntityTorchStand,
		EntityManEater:
		return true
	case EntityGrave:
		return true // see grave.go
	case EntityTreasure:
		return true // opened, it opens again like a chest; see cavelife.go
	case EntityTrap, EntityFlower, EntityGlowPlant, EntityFlora:
		return ready || kind == EntityTrap // a sprung trap is set again
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
	if f, ok := fruitOf(kind); ok {
		return f.hits // to fell it; its fruit is picked in one go
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
	case EntityDoor, EntityFloor:
		return WallHits
	case EntitySpikes:
		return SpikesHits
	case EntityManEater:
		return ManEaterHits
	case EntityGiantTree:
		return GiantTreeHits
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
