package sim

import "math/rand/v2"

// The natural caves of worlds made since StyleCaves. Under every island of
// some size lies a cave system of its own, sized by the island: big
// chambers with round, ragged walls, rock pillars and stalagmites standing
// in them, joined by winding tunnels three to six tiles wide, with a loop
// here and there and few dead ends. Every mouth on the island leads into its
// system, under a shaft of daylight.
//
// A system has two levels. The upper one lies right under the island, in the
// first band of rows below the surface map (see underground.go); its halls
// near the mouths get some daylight, and it holds the common ores. Big
// islands have a lower level too, in a second band under the first: a hole
// with a rope in the far end of the upper level goes down to it, and a rope
// back up. It is darker, it holds jade and the deep ores, and more danger.
// A main route runs from the first mouth to the hole down and on to the
// destination at the far end of the lower level (or of the only level): a
// hoard, by a lake or in a beast's den. Side branches are short and end in
// something worth the detour. Here and there a long tunnel joins the lower
// levels of two neighbouring islands: a secret way under the sea.
//
// Tiles stay tiles: rock is TileMountain, the floor TileCave, water
// TileRiver (fresh and swum), lava TileLava, a frozen pool TileIce. What
// else a tile of the caves is (which chamber, its rock, the main route) is
// kept in a CaveCell per tile for the client to draw it by, worked out with
// the world and never saved.

// naturalCaves reports whether w's caves are the natural systems of
// StyleCaves rather than the one maze of the worlds before. Every check for
// the new caves goes through here, so the style is renumbered in one place.
func (w *World) naturalCaves() bool { return naturalCavesStyle(w.style) }

func naturalCavesStyle(s WorldStyle) bool { return s >= StyleCaves }

// CaveLevels is how many levels deep the natural caves go.
const CaveLevels = 2

// CaveBandTop is the first map row of cave level k (0 the upper, 1 the
// lower) under a surface of surfaceH rows. Each level is as tall as the
// surface, so a tile of the caves lies right under the tile of the surface
// at the same column and row within its band.
func CaveBandTop(surfaceH, k int) int {
	return surfaceH + UndergroundGap + k*(surfaceH+UndergroundGap)
}

// CaveLevelsOf is how many levels of caves a map height rows tall, surfaceH
// of them above ground, holds: 0 with no caves, 1 for the maze of the older
// worlds, 2 for the natural caves.
func CaveLevelsOf(height, surfaceH int) int {
	switch {
	case surfaceH <= 0 || height <= surfaceH:
		return 0
	case height >= CaveBandTop(surfaceH, 1)+surfaceH:
		return 2
	}
	return 1
}

// CaveLevelAt is the level of caves row ty of such a map lies in, and the
// row within its band; ok is false above ground and in the rock between.
func CaveLevelAt(height, surfaceH, ty int) (level, row int, ok bool) {
	for k := CaveLevelsOf(height, surfaceH) - 1; k >= 0; k-- {
		if top := CaveBandTop(surfaceH, k); ty >= top {
			if ty-top < surfaceH {
				return k, ty - top, true
			}
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// caveLevel is the level of the natural caves row ty lies in, -1 for none.
func (w *World) caveLevel(ty int) int {
	if w.caves == nil {
		return -1
	}
	if k, _, ok := CaveLevelAt(w.height, w.surfaceH, ty); ok {
		return k
	}
	return -1
}

// CaveCell is what one tile of the natural caves is: bits 0-3 its zone
// (CaveZone), bits 4-6 its rock (CaveRock), bit 7 set on the main route.
type CaveCell uint8

// CaveZone is the part of a cave system a tile lies in: a tunnel, or a
// chamber of some kind.
type CaveZone uint8

const (
	ZoneTunnel    CaveZone = iota // a passage between chambers
	ZoneHall                      // a plain chamber
	ZoneEntrance                  // the hall under a mouth, where daylight comes down the shaft
	ZoneNest                      // spiders' nest: webs, egg sacs, and the spiders
	ZoneDen                       // a cave bear's den: bones and trodden earth
	ZoneCrystal                   // a cavern of glowing jade crystals, ore in its walls
	ZoneLake                      // an underground lake, with blind fish in it
	ZoneMushroom                  // a grove of glowing mushrooms, and some to eat
	ZoneRoost                     // where the bats hang, and their droppings
	ZoneFossil                    // a great skeleton in the rock
	ZonePaintings                 // pictures painted on the wall long ago
	ZoneTomb                      // the desert's ancient tomb, with its hoard
	ZoneLava                      // a lava tube: glowing channels of lava
	ZoneFrozen                    // an ice cave round a frozen pool
	ZoneFlooded                   // the marsh's flooded cave, crocodiles in its pools
	ZoneStream                    // the jungle's underground stream, roots hanging over it
	NumCaveZones
)

// CaveRock is the rock of a part of a cave, by the land above it.
type CaveRock uint8

const (
	RockLimestone CaveRock = iota // pale and plain, under meadows: gentle
	RockMossy                     // dark and wet, mossy, roots through the roof: under woods
	RockSandstone                 // red and layered: under the desert
	RockBasalt                    // black, glassy obsidian here and there: under the volcano
	RockIce                       // blue ice over the stone: under the snow
	RockMud                       // dark, dripping, muddy: under the marsh
	NumCaveRocks
)

// Zone, Rock and Route read a CaveCell.
func (c CaveCell) Zone() CaveZone { return CaveZone(c & 0x0f) }
func (c CaveCell) Rock() CaveRock { return CaveRock(c>>4) & 0x07 }
func (c CaveCell) Route() bool    { return c&0x80 != 0 }

func makeCaveCell(z CaveZone, r CaveRock, route bool) CaveCell {
	c := CaveCell(z&0x0f) | CaveCell(r&0x07)<<4
	if route {
		c |= 0x80
	}
	return c
}

// caveWorld is what a world's natural caves are, worked out with it and
// never saved (the terrain is regenerated from the seed).
type caveWorld struct {
	cells   []CaveCell // per tile from row surfaceH down
	systems []*caveSystem
	secrets []caveSecret // the passages joining two systems
}

// caveCell is the cave cell of tile (tx, ty); zero outside the caves.
func (w *World) caveCell(tx, ty int) CaveCell {
	if w.caves == nil || !w.inBounds(tx, ty) || ty < w.surfaceH {
		return 0
	}
	return w.caves.cells[(ty-w.surfaceH)*w.width+tx]
}

func (w *World) setCaveCell(tx, ty int, c CaveCell) {
	if w.caves == nil || !w.inBounds(tx, ty) || ty < w.surfaceH {
		return
	}
	w.caves.cells[(ty-w.surfaceH)*w.width+tx] = c
}

// CaveCells returns a copy of the cave cells of every tile from row
// SurfaceHeight down, row-major, or nil when the world has no natural caves.
func (w *World) CaveCells() []CaveCell {
	if w.caves == nil {
		return nil
	}
	return append([]CaveCell(nil), w.caves.cells...)
}

// addCaveBands grows the map downwards by the two levels of the natural
// caves, all rock for now: the systems are dug once the mouths are chosen
// (digCaveSystems), after the villages and the pyramid.
func (w *World) addCaveBands() {
	W, H := w.width, w.height
	newH := CaveBandTop(H, CaveLevels-1) + H + undergroundPad
	tiles := make([]Tile, W*newH)
	copy(tiles, w.tiles)
	for i := W * H; i < len(tiles); i++ {
		tiles[i] = TileMountain
	}
	occ := make([]EntityID, W*newH)
	copy(occ, w.occ)
	w.tiles, w.occ, w.height, w.surfaceH = tiles, occ, newH, H
	if w.relief != nil {
		w.relief = append(w.relief, make([]Relief, W*newH-len(w.relief))...)
	}
	w.fauna.shallow = nil // sized for the old map; rebuilt on demand
	w.caves = &caveWorld{cells: make([]CaveCell, W*(newH-H))}
	w.links = make(map[[2]int]caveLink)
}

// caveUnderground is addUnderground, or for the natural caves the bands
// they will be dug in.
func (w *World) caveUnderground(seed uint32) {
	if w.naturalCaves() {
		w.addCaveBands()
		return
	}
	w.addUnderground(seed)
}

// caveMouths is digCaveMouths, or for the natural caves the mouths of every
// island and the systems under them.
func (w *World) caveMouths(rng *rand.Rand) {
	if w.naturalCaves() {
		w.digCaveSystems(rng)
		return
	}
	w.digCaveMouths(rng)
}
