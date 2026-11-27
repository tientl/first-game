package sim

import "math"

// The ground cover of a StyleGround world. In older worlds tall grass,
// bushes and fallen branches took a tile each with the same chance all over
// a kind of ground, so they lay as evenly as salt shaken on a table. Here
// the same things grow the way plants do:
//
//   - tall grass in drifts, thickest along the edges of the woods, by fresh
//     water and beside the trails, thin in between;
//   - berry bushes in thickets, at the edges of the woods more than out in
//     the open;
//   - fallen branches under and beside the trees;
//   - flowers in patches on the meadows, of one colour each;
//   - open glades and bare spots, where little grows at all.
//
// Each is a chance multiplied by a field of the seed's noise and of how far
// a tile lies from a wood's edge, water and a trail, so a world comes out
// the same every time. The fields are scaled so that about as much grows as
// before, a little less: grass, fibre, berries and wood stay as easy to
// find, only gathered where they grow. Worms keep to damp soil and come up
// when it rains (see rainWorms).

// Ground cover tuning.
const (
	coverReach    = 4    // tiles: how far a wood's edge, water or a trail draws the grass
	coverTotal    = 0.92 // of what grew before, about: a little less
	coverBareFrom = 0.63 // of the bare noise: a glade begins...
	coverBareFull = 0.69 // ...and is bare from here
)

// coverField is what the ground cover is laid out by, worked out once while
// a world is made.
type coverField struct {
	seed  uint32
	W, H  int
	tiles []Tile
	edge  []uint8 // tiles to the edge of a wood (either side of it), up to coverReach
	wet   []uint8 // tiles to fresh water or a marsh, up to coverReach
	trail []uint8 // tiles to a trail, up to coverReach
	norm  [3]float64
}

// The kinds the cover spreads, as indices of coverField.norm.
const (
	coverTuft = iota
	coverBush
	coverTwig
)

// newCover works out the cover's fields for the surface of w.
func (w *World) newCover(seed uint32) *coverField {
	W, H := w.width, w.SurfaceHeight()
	c := &coverField{seed: seed, W: W, H: H, tiles: w.tiles}
	forest := func(i int) bool { return w.tiles[i] == TileForest }
	c.edge = w.tileDistance(func(i int) bool {
		// A tile of the woods with open ground beside it, or open ground
		// beside the woods.
		x, y := i%W, i/W
		f := forest(i)
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			j := ny*W + nx
			if t := w.tiles[j]; forest(j) != f && t.Walkable() {
				return true
			}
		}
		return false
	}, coverReach)
	c.wet = w.tileDistance(func(i int) bool {
		t := w.tiles[i]
		return t == TileRiver || t == TileFord || t == TileMarsh || w.relief != nil && w.lakeAt(i)
	}, coverReach)
	c.trail = w.tileDistance(func(i int) bool { return w.tiles[i] == TilePath }, coverReach)
	// Scale each field so that, over the ground it grows on, it is one on
	// the whole: a sample of every third tile each way is plenty.
	var sum, n [3]float64
	for y := 1; y < H; y += 3 {
		for x := 1; x < W; x += 3 {
			i := y*W + x
			t := w.tiles[i]
			switch t {
			case TileGrass, TileForest:
				for k := range 3 {
					sum[k] += c.raw(k, i, x, y)
					n[k]++
				}
			case TileSand, TileSnow, TileRocky, TileMarsh:
				sum[coverTuft] += c.raw(coverTuft, i, x, y)
				n[coverTuft]++
			}
		}
	}
	for k := range 3 {
		c.norm[k] = coverTotal
		if sum[k] > 0 {
			c.norm[k] = coverTotal * n[k] / sum[k]
		}
	}
	return c
}

// near is how close a tile d tiles from something is to it: 1 on it or
// beside it, 0 from r tiles out.
func near(d uint8, r float64) float64 {
	if d == 255 {
		return 0
	}
	return math.Max(0, 1-math.Max(0, float64(d)-1)/r)
}

// bare is how much grows on tile (x, y) for the glades: 1 in most places,
// falling to 0 in an open glade or a bare spot.
func (c *coverField) bare(x, y float64) float64 {
	b := fbm(x/10+71, y/10+13, c.seed+0x6c2, 2)
	return 1 - smoothstep(clamp((b-coverBareFrom)/(coverBareFull-coverBareFrom), 0, 1))
}

// raw is the cover's field for kind k on tile i at (tx, ty), before it is
// scaled.
func (c *coverField) raw(k, i, tx, ty int) float64 {
	x, y := float64(tx)+0.5, float64(ty)+0.5
	edge, wet, trail := near(c.edge[i], coverReach), near(c.wet[i], coverReach), near(c.trail[i], 2.5)
	switch k {
	case coverTuft:
		// Drifts: a noise stretched a little across, so they lie in bands
		// more than in blobs.
		n := fbm(x/7+3.1, y/5+8.7, c.seed+0x6c1, 3)
		d := smoothstep(clamp((n-0.5)/0.13, 0, 1))
		// Along an edge the grass gathers too, in drifts of its own.
		e := smoothstep(clamp((fbm(x/3.5+17, y/3.5+5, c.seed+0x6c3, 2)-0.45)/0.2, 0, 1))
		m := 0.03 + 3.4*d*d*d + (2.2*edge+1.8*wet+1.5*trail)*e*e
		return m * c.bare(x, y)
	case coverBush:
		// Thickets: tighter patches than the grass.
		n := fbm(x/5+11, y/5+2, c.seed+0x6c4, 2)
		d := smoothstep(clamp((n-0.54)/0.13, 0, 1))
		return (0.03 + 3.2*d*d*d + 1.2*edge*d) * c.bare(x, y)
	case coverTwig:
		// Under and beside the trees: in the woods and at their edges.
		n := fbm(x/5+29, y/5+31, c.seed+0x6c5, 2)
		d := smoothstep(clamp((n-0.4)/0.25, 0, 1))
		m := 0.2 + 1.2*d + 1.8*edge
		if c.edge[i] == 255 && c.tiles[i] != TileForest { // far out in the open: little falls there
			m *= 0.5
		}
		return m
	}
	return 1
}

// mul is the chance of kind k on tile i at (tx, ty), as a share of the
// chance it had in older worlds.
func (c *coverField) mul(k EntityKind, i, tx, ty int) float64 {
	switch k {
	case EntityGrassTuft:
		return c.norm[coverTuft] * c.raw(coverTuft, i, tx, ty)
	case EntityBerryBush:
		return c.norm[coverBush] * c.raw(coverBush, i, tx, ty)
	case EntityTwig:
		return c.norm[coverTwig] * c.raw(coverTwig, i, tx, ty)
	case EntityReed:
		// Stands of reeds, with open mud between.
		x, y := float64(tx)+0.5, float64(ty)+0.5
		d := smoothstep(clamp((fbm(x/4+5, y/4+9, c.seed+0x6c6, 2)-0.42)/0.2, 0, 1))
		return 0.15 + 1.75*d
	}
	return 1
}

// picker is pick with each chance scaled by the cover on tile i.
func (c *coverField) picker(i, tx, ty int) func(r float64, options ...chance) EntityKind {
	return func(r float64, options ...chance) EntityKind {
		for _, o := range options {
			p := o.p * c.mul(o.kind, i, tx, ty)
			if r < p {
				return o.kind
			}
			r -= p
		}
		return EntityNone
	}
}

// tileDistance is, for every tile of the surface, how many steps (eight
// ways) it lies from a tile src says yes to, up to most; 255 further.
func (w *World) tileDistance(src func(i int) bool, most int) []uint8 {
	W, H := w.width, w.SurfaceHeight()
	d := make([]uint8, W*H)
	var q []int32
	for i := range d {
		if src(i) {
			q = append(q, int32(i))
		} else {
			d[i] = 255
		}
	}
	for k := 0; k < len(q); k++ {
		i := int(q[k])
		if int(d[i]) >= most {
			continue
		}
		x, y := i%W, i/W
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; d[j] == 255 {
					d[j] = d[i] + 1
					q = append(q, int32(j))
				}
			}
		}
	}
	return d
}

// undergrowthMul is how the woods' floor (fillUndergrowth) is spread in a
// StyleGround world: for each kind, a share of its old chance on tile i,
// trees the number of trees standing within two tiles.
func (c *coverField) undergrowthMul(k EntityKind, i, tx, ty, trees int) float64 {
	x, y := float64(tx)+0.5, float64(ty)+0.5
	switch k {
	case EntityTwig:
		// Branches fall under the crowns.
		return []float64{0.35, 1.1, 1.6, 1.9}[min(trees, 3)]
	case EntityMushroom:
		return []float64{0.5, 1.1, 1.3, 1.4}[min(trees, 3)]
	case EntityFlower:
		// Patches where the sun gets in.
		d := smoothstep(clamp((fbm(x/6+51, y/6+3, c.seed+0x6c7, 2)-0.48)/0.16, 0, 1))
		return (0.1 + 2.6*d*d) * []float64{1.3, 1, 0.6, 0.4}[min(trees, 3)]
	case EntityGrassTuft:
		// Grass where the sun gets in, not under the crowns.
		return c.mul(k, i, tx, ty) * []float64{1.5, 1, 0.55, 0.35}[min(trees, 3)]
	}
	return c.mul(k, i, tx, ty)
}

// treesNear is how many trees stand within two tiles of (tx, ty).
func (w *World) treesNear(tx, ty int) int {
	n := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if dx*dx+dy*dy > 5 {
				continue
			}
			if e := w.entityAt(tx+dx, ty+dy); e != nil && (e.Kind == EntityTree || e.Kind == EntityGiantTree) {
				n++
			}
		}
	}
	return n
}

// flowerPatch is the chance of a wild flower on a meadow tile at (x, y) of
// a StyleGround world: few on their own, most in patches.
func flowerPatch(x, y float64, seed uint32) float64 {
	patch := fbm(x/8+5, y/8+11, seed+71, 2)
	return 0.003 + 0.072*smoothstep(clamp((patch-0.56)/0.12, 0, 1))
}

// pickChance is pick, for where a scaled pick stands in for it.
func pickChance(r float64, options ...chance) EntityKind { return pick(r, options...) }

// Worms since StyleGround: they keep to damp soil, by water and in the
// marsh, fewer than the one to every 150 tiles of grass of older worlds;
// when it rains more come up round the players, and go back down out of
// sight a while after it stops.
const (
	wormDampTilesPer = 110                 // a worm to this many tiles of damp soil
	wormDampReach    = 2                   // tiles from water or marsh that are damp
	rainWormsNear    = 4                   // worms up in the rain round a player, at most
	rainWormLinger   = 90 * TicksPerSecond // they stay up this long after the rain
	rainWormEvery    = 2 * TicksPerSecond  // how often the rain brings one up
	rainWormRange    = 12.0                // tiles round a player they come up within...
	rainWormNear     = 5.0                 // ...and no closer
)

// rainWorms are the worms the rain brought up.
type rainWorms struct {
	ids   []CreatureID
	wetAt uint64 // the last tick it rained; 0: not since the world was made or loaded
}

// dampSoil reports whether tile (tx, ty) is damp: marsh, or ground within
// wormDampReach tiles of a river, a ford, a lake or a marsh.
func (w *World) dampSoil(tx, ty int) bool {
	for dy := -wormDampReach; dy <= wormDampReach; dy++ {
		for dx := -wormDampReach; dx <= wormDampReach; dx++ {
			x, y := tx+dx, ty+dy
			if !w.inBounds(x, y) {
				continue
			}
			i := y*w.width + x
			switch w.tiles[i] {
			case TileMarsh, TileRiver, TileFord:
				return true
			case TileWater:
				if w.relief != nil && w.lakeAt(i) {
					return true
				}
			}
		}
	}
	return false
}

// stepRainWorms brings worms up round the players while it rains, and lets
// them go back down, unseen, a while after it stops.
func (w *World) stepRainWorms() {
	f := &w.fauna
	if w.Weather().Wet() {
		f.rain.wetAt = w.tick
	}
	if w.tick%rainWormEvery != 0 {
		return
	}
	wet := f.rain.wetAt > 0 && w.tick-f.rain.wetAt < rainWormLinger
	ids := f.rain.ids[:0]
	for _, id := range f.rain.ids {
		c := w.creatureByID(id)
		if c == nil {
			continue
		}
		if !wet && c.State != CreatureHooked {
			if _, d := w.nearestPlayer(c.X, c.Y); d > creatureSpawnAway {
				c.gone = true // back down into the soil
				continue
			}
		}
		ids = append(ids, id)
	}
	f.rain.ids = ids
	if !w.Weather().Wet() {
		return
	}
	H := w.SurfaceHeight()
	for _, pid := range w.playerOrder {
		p := w.players[pid]
		if p.Dead || int(p.Y) >= H {
			continue
		}
		n := 0
		for _, id := range f.rain.ids {
			if c := w.creatureByID(id); c != nil && math.Hypot(c.X-p.X, c.Y-p.Y) < rainWormRange {
				n++
			}
		}
		if n >= rainWormsNear {
			continue
		}
		rng := w.random()
		for range 6 {
			a, r := rng.Float64()*2*math.Pi, rainWormNear+rng.Float64()*(rainWormRange-rainWormNear)
			tx, ty := int(math.Floor(p.X+math.Cos(a)*r)), int(math.Floor(p.Y+math.Sin(a)*r))
			if !w.inBounds(tx, ty) || ty >= H {
				continue
			}
			if t := w.tiles[ty*w.width+tx]; t != TileGrass && t != TileForest && t != TileMarsh {
				continue
			}
			x, y := float64(tx)+0.5, float64(ty)+0.5
			if !w.creatureFits(CreatureWorm, x, y) {
				continue
			}
			f.rain.ids = append(f.rain.ids, w.addCreature(CreatureWorm, x, y).ID)
			break
		}
	}
}
