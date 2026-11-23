package sim

import (
	"fmt"
	"math"
	"math/rand/v2"
)

// GenConfig controls world generation.
type GenConfig struct {
	Seed          uint64
	Width, Height int
	Style         WorldStyle // StyleIsland (the zero value) or one of the open-world styles
}

// World size limits. The upper bound keeps tile coordinates inside int16 for
// the wire format.
const (
	MinWorldSize = 32
	MaxWorldSize = 4096
)

// Generation thresholds, tuned by eye on the ASCII dump in worldgen_test.go.
const (
	featureScale = 22.0 // tiles per noise unit

	waterLevel = 0.36
	sandLevel  = 0.41
	forestLine = 0.54 // moisture above this is forest
	rockLine   = 0.66 // rock noise above this is rocky ground

	// jungleForestLine is forestLine in open worlds made since StyleJungle.
	jungleForestLine = 0.42

	// Share of tiles with a tree, by ground: the woods are thick, the
	// meadows dotted, the swamps have a few.
	forestTrees = 0.55
	grassTrees  = 0.07
	marshTrees  = 0.1

	clearSpawnRadius = 2 // tiles kept free of entities around the spawn

	// Share of trees and boulders that are big. Old forest grows the big trees;
	// out in the open most trees are young.
	bigForestTree = 0.45
	bigGrassTree  = 0.15
	bigBoulder    = 0.3
)

// Generate builds a new island world from cfg. The same config always yields
// the same world.
func Generate(cfg GenConfig) (*World, error) {
	if cfg.Width < MinWorldSize || cfg.Height < MinWorldSize ||
		cfg.Width > MaxWorldSize || cfg.Height > MaxWorldSize {
		return nil, fmt.Errorf("sim: world size %dx%d outside [%d, %d]",
			cfg.Width, cfg.Height, MinWorldSize, MaxWorldSize)
	}
	w := newWorld(cfg.Width, cfg.Height)
	w.seed = cfg.Seed
	w.genH = cfg.Height
	w.style = cfg.Style
	seed := uint32(cfg.Seed) ^ uint32(cfg.Seed>>32)
	rng := rand.New(rand.NewPCG(cfg.Seed, 0x5eed5eed))

	if cfg.Style != StyleIsland {
		var sx, sy int
		if cfg.Style >= StyleSeas {
			sx, sy = w.archipelagoTerrain(seed)
		} else {
			sx, sy = w.openTerrain(seed)
		}
		tribal := cfg.Style >= StyleTribal
		if tribal {
			trng := rand.New(rand.NewPCG(cfg.Seed, 0x7e1b))
			w.pickVillages(trng, sx, sy)
			w.carvePaths(trng, sx, sy)
		}
		if cfg.Style >= StyleSeas {
			w.addUnderground(seed) // see underground.go
		}
		w.scatterOpen(rng, rand.New(rand.NewPCG(cfg.Seed, 0xb16b16)))
		w.scatterPebbles(rand.New(rand.NewPCG(cfg.Seed, 0x9ebb1e)))
		w.scatterOres(rand.New(rand.NewPCG(cfg.Seed, 0x0de5)))
		w.scatterFruit(rand.New(rand.NewPCG(cfg.Seed, 0xf1f)))
		w.spawnX, w.spawnY = float64(sx)+0.5, float64(sy)+0.5
		if tribal {
			prng := rand.New(rand.NewPCG(cfg.Seed, 0x91a47))
			w.scatterPlants(prng, seed)
			w.buildVillages(prng)
			if cfg.Style >= StyleSeas {
				w.raisePyramid(prng)
				w.digCaveMouths(prng) // after the pyramid, which clears its ground
			}
			w.placeTreasure(prng)
		}
		w.clearAround(sx, sy, clearSpawnRadius)
		w.starterKit(rng)
		w.spawnCreatures()
		if tribal {
			w.spawnTribes()
			w.guardPyramid()
		}
		return w, nil
	}

	w.generateTerrain(seed)
	// Sizes come from their own stream so they do not shift what rng places.
	w.scatterEntities(rng, rand.New(rand.NewPCG(cfg.Seed, 0xb16b16)))
	w.scatterPebbles(rand.New(rand.NewPCG(cfg.Seed, 0x9ebb1e)))
	w.scatterOres(rand.New(rand.NewPCG(cfg.Seed, 0x0de5)))
	w.scatterFruit(rand.New(rand.NewPCG(cfg.Seed, 0xf1f)))
	if err := w.pickSpawn(); err != nil {
		return nil, err
	}
	w.starterKit(rng)
	w.spawnCreatures()
	return w, nil
}

func (w *World) generateTerrain(seed uint32) {
	cx, cy := float64(w.width)/2, float64(w.height)/2
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			nx, ny := float64(tx)/featureScale, float64(ty)/featureScale

			// Elevation: noise pushed down towards the edges to make an island.
			dx, dy := (float64(tx)+0.5-cx)/cx, (float64(ty)+0.5-cy)/cy
			d2 := dx*dx + dy*dy
			elev := fbm(nx, ny, seed, 5) + 0.34 - 0.9*d2

			var t Tile
			switch {
			case elev < waterLevel:
				t = TileWater
			case elev < sandLevel:
				t = TileSand
			case fbm(nx*0.8+311, ny*0.8+97, seed+101, 3) > rockLine:
				t = TileRocky
			case fbm(nx*1.3+57, ny*1.3+213, seed+202, 4) > forestLine:
				t = TileForest
			default:
				t = TileGrass
			}
			w.tiles[ty*w.width+tx] = t
		}
	}
}

func (w *World) scatterEntities(rng, sizeRng *rand.Rand) {
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			r := rng.Float64()
			var kind EntityKind
			tile := w.tiles[ty*w.width+tx]
			switch tile {
			case TileForest:
				kind = pick(r, chance{EntityTree, forestTrees}, chance{EntityBerryBush, 0.03}, chance{EntityGrassTuft, 0.03}, chance{EntityTwig, 0.035})
			case TileGrass:
				kind = pick(r, chance{EntityTree, grassTrees}, chance{EntityGrassTuft, 0.08}, chance{EntityBerryBush, 0.03}, chance{EntityTwig, 0.015})
			case TileRocky:
				kind = pick(r, chance{EntityBoulder, 0.12}, chance{EntityGrassTuft, 0.02})
			case TileSand:
				kind = pick(r, chance{EntityGrassTuft, 0.02})
			}
			if kind == EntityNone {
				continue
			}
			e := w.addEntity(kind, tx, ty)
			if kind.Sized() && sizeRng.Float64() < bigShare(kind, tile) {
				e.Big = true
				e.HP = maxHP(kind, true)
			}
		}
	}
}

// scatterPebbles drops small stones on free tiles: many on rocky ground and
// beaches, a few in the grass and the woods. It has its own rng so the rest
// of the world does not shift.
func (w *World) scatterPebbles(rng *rand.Rand) {
	share := [NumTiles]float64{
		TileRocky: 0.07, TileSand: 0.035, TileGrass: 0.012, TileForest: 0.008,
		TileDesert: 0.03, TileAsh: 0.04, TileCave: 0.05, TileMarsh: 0.005,
	}
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			if rng.Float64() < share[w.tiles[ty*w.width+tx]] && w.canPlace(tx, ty) {
				w.addEntity(EntityPebble, tx, ty)
			}
		}
	}
}

func bigShare(kind EntityKind, tile Tile) float64 {
	switch {
	case kind == EntityBoulder:
		return bigBoulder
	case tile == TileForest:
		return bigForestTree
	default:
		return bigGrassTree
	}
}

type chance struct {
	kind EntityKind
	p    float64
}

// pick maps a uniform r in [0, 1) onto consecutive probability bands.
func pick(r float64, options ...chance) EntityKind {
	for _, o := range options {
		if r < o.p {
			return o.kind
		}
		r -= o.p
	}
	return EntityNone
}

// pickSpawn chooses a grass or forest tile close to the map centre, searching
// outwards ring by ring, then clears the entities around it so a new player
// has room to move.
func (w *World) pickSpawn() error {
	cx, cy := w.width/2, w.height/2
	maxR := max(w.width, w.height)
	for r := 0; r < maxR; r++ {
		best, bestD := -1, math.MaxInt
		// Walk the ring at Chebyshev distance r and keep the closest match.
		for ty := cy - r; ty <= cy+r; ty++ {
			for tx := cx - r; tx <= cx+r; tx++ {
				if max(abs(tx-cx), abs(ty-cy)) != r || !w.inBounds(tx, ty) {
					continue
				}
				if t := w.tiles[ty*w.width+tx]; t != TileGrass && t != TileForest {
					continue
				}
				if d := (tx-cx)*(tx-cx) + (ty-cy)*(ty-cy); d < bestD {
					best, bestD = ty*w.width+tx, d
				}
			}
		}
		if best >= 0 {
			sx, sy := best%w.width, best/w.width
			w.spawnX, w.spawnY = float64(sx)+0.5, float64(sy)+0.5
			w.clearAround(sx, sy, clearSpawnRadius)
			return nil
		}
	}
	return fmt.Errorf("sim: no grass or forest tile to spawn on")
}

// Starter materials near the spawn: a new player starts with empty hands and
// must find enough here for the first tools (twigs and pebbles), the stone
// for a campfire (boulders) and its tinder (grass tufts).
const (
	starterTwigs    = 5
	starterTufts    = 3
	starterBoulders = 2 // rocky ground can be far away; two small boulders are 4 stone
	starterPebbles  = 4 // an axe, a pickaxe and a hoe
	starterMinR     = clearSpawnRadius + 1
	starterMaxR     = 6
)

// starterKit tops up the twigs, grass tufts, boulders and pebbles within
// starterMaxR of the spawn, so the first tools never depend on luck.
func (w *World) starterKit(rng *rand.Rand) {
	cx, cy := int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY))
	for _, want := range []struct {
		kind EntityKind
		n    int
	}{{EntityTwig, starterTwigs}, {EntityGrassTuft, starterTufts}, {EntityBoulder, starterBoulders}, {EntityPebble, starterPebbles}} {
		have := 0
		var free [][2]int
		for ty := cy - starterMaxR; ty <= cy+starterMaxR; ty++ {
			for tx := cx - starterMaxR; tx <= cx+starterMaxR; tx++ {
				if max(abs(tx-cx), abs(ty-cy)) < starterMinR {
					continue
				}
				if e := w.entityAt(tx, ty); e != nil {
					if e.Kind == want.kind {
						have++
					}
					continue
				}
				if w.canPlace(tx, ty) {
					free = append(free, [2]int{tx, ty})
				}
			}
		}
		rng.Shuffle(len(free), func(i, j int) { free[i], free[j] = free[j], free[i] })
		for i := 0; have < want.n && i < len(free); i++ {
			w.addEntity(want.kind, free[i][0], free[i][1])
			have++
		}
	}
}

func (w *World) clearAround(cx, cy, r int) {
	for ty := cy - r; ty <= cy+r; ty++ {
		for tx := cx - r; tx <= cx+r; tx++ {
			if e := w.entityAt(tx, ty); e != nil {
				w.removeEntity(e)
			}
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
