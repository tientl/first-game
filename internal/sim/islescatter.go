package sim

import "math/rand/v2"

// What grows and lies about in a StyleIsles world: the same things as in
// the older open worlds, but thicker or thinner by the variant of the
// ground (see relief.go): thick woods and thin, clearings and meadows,
// outcrops, boulder fields and scree. Ramps and fords are kept clear.

// scatterIsles places the resources of every biome, by its variant.
func (w *World) scatterIsles(rng, sizeRng *rand.Rand) {
	var shore []uint8 // tiles from the sea, in worlds with wide beaches: see beachThing
	if w.style >= StyleBeaches {
		shore = w.seaDistance(8)
	}
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			r := rng.Float64()
			i := ty*w.width + tx
			tile := w.tiles[i]
			v := w.relief[i].Variant()
			if v == VarRamp || tile == TileFord || !tile.Walkable() {
				continue // the ways up and across stay open
			}
			var kind EntityKind
			switch tile {
			case TileForest:
				trees := tribalForestTrees
				switch v {
				case VarDenseWood:
					trees = 0.5
				case VarOpenWood:
					trees = 0.16
				}
				kind = pick(r, chance{EntityTree, trees}, chance{EntityBerryBush, 0.03}, chance{EntityGrassTuft, 0.03}, chance{EntityTwig, 0.035})
			case TileGrass:
				switch v {
				case VarMeadow:
					kind = pick(r, chance{EntityTree, 0.008}, chance{EntityGrassTuft, 0.11}, chance{EntityBerryBush, 0.035}, chance{EntityTwig, 0.008})
				case VarClearing:
					kind = pick(r, chance{EntityTree, 0.025}, chance{EntityGrassTuft, 0.08}, chance{EntityBerryBush, 0.04}, chance{EntityTwig, 0.03})
				case VarBoulders:
					kind = pick(r, chance{EntityBoulder, 0.1}, chance{EntityGrassTuft, 0.06}, chance{EntityTwig, 0.01})
				default:
					kind = pick(r, chance{EntityTree, grassTrees}, chance{EntityGrassTuft, 0.08}, chance{EntityBerryBush, 0.03}, chance{EntityTwig, 0.015})
				}
			case TileRocky:
				boulders := 0.12
				switch v {
				case VarOutcrop:
					boulders = 0.16
				case VarBoulders:
					boulders = 0.22
				case VarScree:
					boulders = 0.04
				}
				kind = pick(r, chance{EntityBoulder, boulders}, chance{EntityGrassTuft, 0.02})
			case TileSand:
				switch {
				case shore != nil:
					kind = w.beachThing(r, tx, ty, v, int(shore[i]))
				case v == VarDunes:
					kind = pick(r, chance{EntityGrassTuft, 0.03})
				case w.seaside(tx, ty):
					kind = pick(r, chance{EntityClam, clamShare}, chance{EntityGrassTuft, 0.02})
				default:
					kind = pick(r, chance{EntityGrassTuft, 0.02})
				}
			case TileDesert:
				switch v {
				case VarDunes:
					kind = pick(r, chance{EntityCactus, 0.012}, chance{EntityTwig, 0.004})
				case VarBoulders:
					kind = pick(r, chance{EntityBoulder, 0.1}, chance{EntityCactus, 0.02})
				default:
					kind = pick(r, chance{EntityCactus, 0.045}, chance{EntityBoulder, 0.012}, chance{EntityTwig, 0.012})
				}
			case TileMarsh:
				if v == VarMudflat {
					kind = pick(r, chance{EntityReed, 0.05}, chance{EntityMushroom, 0.005})
				} else {
					kind = pick(r, chance{EntityReed, 0.14}, chance{EntityMushroom, 0.03}, chance{EntityTree, marshTrees}, chance{EntityGrassTuft, 0.02})
				}
			case TileCave:
				kind = pick(r, chance{EntityBoulder, 0.07}, chance{EntityNone, 0.05}, chance{EntityMushroom, 0.03})
			case TileAsh:
				boulders := 0.05
				if v == VarBoulders {
					boulders = 0.13
				}
				kind = pick(r, chance{EntityBoulder, boulders})
			case TileSnow:
				trees := 0.1
				switch v {
				case VarDenseWood:
					trees = 0.3
				case VarBoulders:
					trees = 0.02
				}
				boulders := 0.02
				if v == VarBoulders {
					boulders = 0.11
				}
				kind = pick(r, chance{EntityTree, trees}, chance{EntityBoulder, boulders}, chance{EntityTwig, 0.02}, chance{EntityGrassTuft, 0.015}, chance{EntityPebble, 0.008})
			}
			if v == VarScree && tile != TileRocky && kind == EntityNone {
				kind = pick(r, chance{EntityPebble, 0.12}, chance{EntityBoulder, 0.04})
			}
			if kind == EntityNone || kind == EntityTree && w.style >= StyleWoods {
				continue // the trees of the newest worlds are spaced by their crowns: see woods.go
			}
			e := w.addEntity(kind, tx, ty)
			if kind.Sized() && sizeRng.Float64() < bigShare(kind, tile) {
				e.Big = true
				e.HP = maxHP(kind, true)
			}
		}
	}
}

// crossing reports whether tile i is a ramp or a ford: a way up a cliff or
// over a river, which must stay open.
func (w *World) crossing(i int) bool {
	return w.relief != nil && (w.tiles[i] == TileFord || w.tiles[i].Walkable() && w.relief[i].Variant() == VarRamp)
}

// unblockCrossings takes away whatever was built across a ramp or a ford
// while the world was made: a giant tree goes whole, anything else that
// blocks the way on its own.
func (w *World) unblockCrossings() {
	for i := range w.tiles {
		if !w.crossing(i) {
			continue
		}
		e := w.entityAt(i%w.width, i/w.width)
		if e == nil || !blocksPlayers(e.Kind) || e.Kind == EntityPyramid || e.Kind == EntityPyramidStone {
			continue
		}
		if e.Kind == EntityGiantRoot || e.Kind == EntityGiantTree {
			w.felledGiantAt(e.TX, e.TY)
			continue
		}
		w.removeEntity(e)
	}
}

// felledGiantAt takes away the giant tree whose foot covers (tx, ty).
func (w *World) felledGiantAt(tx, ty int) {
	for cy := ty; cy <= ty+GiantD-1; cy++ {
		for cx := tx - GiantW/2; cx <= tx+GiantW/2; cx++ {
			g := w.entityAt(cx, cy)
			if g == nil || g.Kind != EntityGiantTree {
				continue
			}
			for y := cy - GiantD + 1; y <= cy; y++ {
				for x := cx - GiantW/2; x <= cx+GiantW/2; x++ {
					if e := w.entityAt(x, y); e != nil && (e.Kind == EntityGiantRoot || e.Kind == EntityGiantTree) {
						w.removeEntity(e)
					}
				}
			}
			return
		}
	}
}

// unblockIslands takes away what was built across the only way between two
// parts of an island (a giant tree in a narrow pass between cliffs, a hut in
// a gap), so every island can still be walked end to end. Slivers of beach
// behind a cave mouth's rocks are left be.
func (w *World) unblockIslands() {
	W, H := w.width, w.SurfaceHeight()
	part := make([]int32, W*H)
	for range 6 {
		for i := range part {
			part[i] = -1
		}
		var sizes []int
		var q []int
		for start := range part {
			if part[start] >= 0 || !w.tiles[start].Walkable() || w.blocked(start%W, start/W) {
				continue
			}
			n := int32(len(sizes))
			part[start] = n
			q = append(q[:0], start)
			for k := 0; k < len(q); k++ {
				x, y := q[k]%W, q[k]/W
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if nx < 0 || ny < 0 || nx >= W || ny >= H {
						continue
					}
					if j := ny*W + nx; part[j] < 0 && w.tiles[j].Walkable() && !w.blocked(nx, ny) {
						part[j] = n
						q = append(q, j)
					}
				}
			}
			sizes = append(sizes, len(q))
		}
		removed := false
		for _, isl := range w.landMasses() {
			if isl[0]/W >= H {
				continue
			}
			best := int32(-1)
			for _, i := range isl {
				if p := part[i]; p >= 0 && (best < 0 || sizes[p] > sizes[best]) {
					best = p
				}
			}
			for _, i := range isl {
				p := part[i]
				if p < 0 || p == best || sizes[p] <= 8 {
					continue
				}
				// A blocker beside this part with open ground of another part
				// beyond it goes.
				x, y := i%W, i/W
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					e := w.entityAt(x+d[0], y+d[1])
					if e == nil || !blocksPlayers(e.Kind) || e.Kind == EntityPyramid || e.Kind == EntityPyramidStone {
						continue
					}
					if e.Kind == EntityGiantRoot || e.Kind == EntityGiantTree {
						w.felledGiantAt(e.TX, e.TY)
					} else {
						w.removeEntity(e)
					}
					removed = true
				}
			}
		}
		if !removed {
			return
		}
	}
}
