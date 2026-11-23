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

	clearSpawnRadius = 2 // tiles kept free of entities around the spawn
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
	seed := uint32(cfg.Seed) ^ uint32(cfg.Seed>>32)
	rng := rand.New(rand.NewPCG(cfg.Seed, 0x5eed5eed))

	w.generateTerrain(seed)
	w.scatterEntities(rng)
	if err := w.pickSpawn(); err != nil {
		return nil, err
	}
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

func (w *World) scatterEntities(rng *rand.Rand) {
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			r := rng.Float64()
			var kind EntityKind
			switch w.tiles[ty*w.width+tx] {
			case TileForest:
				kind = pick(r, chance{EntityTree, 0.38}, chance{EntityBerryBush, 0.03}, chance{EntityGrassTuft, 0.03})
			case TileGrass:
				kind = pick(r, chance{EntityTree, 0.04}, chance{EntityGrassTuft, 0.08}, chance{EntityBerryBush, 0.03})
			case TileRocky:
				kind = pick(r, chance{EntityBoulder, 0.12}, chance{EntityGrassTuft, 0.02})
			case TileSand:
				kind = pick(r, chance{EntityGrassTuft, 0.02})
			}
			if kind != EntityNone {
				w.addEntity(kind, tx, ty)
			}
		}
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
