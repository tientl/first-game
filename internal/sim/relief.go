package sim

// The relief of worlds made since StyleIsles. Beside its kind, every tile
// keeps one byte saying how high the ground lies and how it varies within
// its biome, for the sim (more trees in thick woods, none in a clearing) and
// for the client, which draws the levels, the cliffs between them and the
// looks of the ground from it. Older worlds have no relief: every tile is
// level 0, plain.

// Relief is one tile's relief:
//
//   - bits 0-1: the height level, 0 (the shore) to MaxLevel. A cliff tile has
//     the level of the high ground it is the edge of; a waterfall the level
//     of the river above it.
//   - bits 2-5: the variant of the ground (GroundVariant); on running water
//     (a river, a ford or a waterfall) the way it flows instead (FlowDir).
//   - bit 6: soft, a wide border: the client blends this tile far into the
//     ground next to it.
//   - bit 7: wet, beside fresh water: lusher grass, wetter ground.
type Relief uint8

// MaxLevel is the highest height level.
const MaxLevel = 3

const (
	reliefLevelMask = 0x03
	reliefVarShift  = 2
	reliefVarMask   = 0x0f << reliefVarShift
	reliefSoft      = 0x40
	reliefWet       = 0x80
)

// Level is the height level, 0 to MaxLevel.
func (r Relief) Level() int { return int(r & reliefLevelMask) }

// Variant is how the ground varies within its biome. It means nothing on
// running water: see Flow.
func (r Relief) Variant() GroundVariant { return GroundVariant(r&reliefVarMask) >> reliefVarShift }

// Flow is the way running water flows, FlowNone on anything else.
func (r Relief) Flow() FlowDir { return FlowDir(r&reliefVarMask) >> reliefVarShift }

// Soft reports whether the client blends this tile widely into its
// neighbours: a border between two kinds of ground that fades.
func (r Relief) Soft() bool { return r&reliefSoft != 0 }

// Wet reports whether the tile lies beside fresh water.
func (r Relief) Wet() bool { return r&reliefWet != 0 }

func (r Relief) withLevel(l int) Relief {
	return r&^reliefLevelMask | Relief(min(max(l, 0), MaxLevel))
}

func (r Relief) withVariant(v GroundVariant) Relief {
	return r&^reliefVarMask | Relief(v&0x0f)<<reliefVarShift
}

// GroundVariant is how a tile's ground varies within its biome.
type GroundVariant uint8

const (
	VarPlain     GroundVariant = iota
	VarMeadow                  // grass: an open meadow, flowery, no trees
	VarClearing                // woods: a clearing, grass with a tree or two
	VarDenseWood               // woods: the thickest of them, dark under the canopy
	VarOpenWood                // woods: thin, the trees well apart
	VarOutcrop                 // rocky ground standing out of another biome: a rock outcrop
	VarDunes                   // desert or sand: dunes, rippled by the wind
	VarScree                   // loose stones fallen below a cliff
	VarMudflat                 // marsh or sand: wet mud flats by the water
	VarBoulders                // a field of boulders
	VarRamp                    // a slope cut through a cliff between two levels: walked up and down
	VarReef                    // shallows: a coral reef
	VarLagoon                  // shallows: a calm lagoon behind a spit or inside an atoll
	VarShoal                   // shallows: a sandbank in a strait between islands
	VarStack                   // a cliff standing alone in the sea: a sea stack
	VarLake                    // still fresh water: a lake (TileWater)
	NumGroundVariants
)

// FlowDir is the way a river flows across a tile.
type FlowDir uint8

const (
	FlowNone FlowDir = iota
	FlowEast
	FlowSouth
	FlowWest
	FlowNorth
)

// Delta is the step of one tile downstream.
func (f FlowDir) Delta() (dx, dy int) {
	switch f {
	case FlowEast:
		return 1, 0
	case FlowSouth:
		return 0, 1
	case FlowWest:
		return -1, 0
	case FlowNorth:
		return 0, -1
	}
	return 0, 0
}

// flowTo is the FlowDir of a step by (dx, dy).
func flowTo(dx, dy int) FlowDir {
	switch {
	case dx > 0:
		return FlowEast
	case dx < 0:
		return FlowWest
	case dy > 0:
		return FlowSouth
	case dy < 0:
		return FlowNorth
	}
	return FlowNone
}

// Sea reports whether t is the sea: deep or shallow.
func (t Tile) Sea() bool { return t == TileWater || t == TileShallow }

// Swimmable reports whether t is open water a swimmer can be in: the sea,
// its shallows, a lake or a river (not a waterfall).
func (t Tile) Swimmable() bool { return t == TileWater || t == TileShallow || t == TileRiver }

// Relief returns a copy of the relief of every tile, row-major, or nil for
// a world made before StyleIsles.
func (w *World) Relief() []Relief {
	if w.relief == nil {
		return nil
	}
	return append([]Relief(nil), w.relief...)
}

// ReliefAt is the relief of tile (tx, ty): zero off the map and in worlds
// without relief.
func (w *World) ReliefAt(tx, ty int) Relief {
	if w.relief == nil || !w.inBounds(tx, ty) {
		return 0
	}
	return w.relief[ty*w.width+tx]
}

// LevelAt is the height level of tile (tx, ty).
func (w *World) LevelAt(tx, ty int) int { return w.ReliefAt(tx, ty).Level() }
