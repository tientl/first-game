package sim

import (
	"cmp"
	"math"
	"slices"
)

// The woods grow back. A felled tree leaves its stump; after a day a
// sapling comes up by it, after a day and a half more it is a young tree,
// and a young tree where a grown one stood is grown after two more days:
//
//	stump (1 day) → sapling (1½ days) → young tree (2 days) → grown tree
//
// so a felled small tree is back in two and a half days, a big one in four
// and a half, with a young tree to cut in between. (A day is 16 minutes.)
//
// The woods also keep to their plan: where each tree of the world stood as
// it was made (in an older world, as the woods were thinned: see
// thinwoods.go). A slow scan goes over the whole map every two days, a few
// tiles a tick, and where a planned tree is missing, now and then a sapling
// comes up, so long as a grown tree would have room there and nothing
// people made or a lair or a cave is near. The plan is made again from the
// seed when a world is loaded, and the scan follows from the tick, so none
// of it is saved but the trees themselves.

// Regrowth tuning.
const (
	StumpTicks     = DayTicks     // a felled tree's stump, before a sapling comes up
	YoungTreeTicks = 2 * DayTicks // a young tree where a grown one stood, until it is grown
	fillCycleTicks = 2 * DayTicks // the slow scan goes over the map once in this long
	fillChance     = 0.35         // a gap the scan finds gets a sapling this often
	fillRoom       = 0.75         // a new sapling keeps this much of a grown tree's spacing
	hivesPerTrees  = 45           // trees of the woods to a beehive: now and then one
)

// What the plan has on a tile.
const (
	planNone  uint8 = iota
	planYoung       // a young tree
	planGrown       // a grown tree
)

// saplingGrows is the Fuel of a sapling that grows into a grown tree, not
// just a young one: one coming up by a big tree's stump, or where the plan
// has a grown tree.
const saplingGrows = 1

// woodsPlan is the plan of a world's woods, and, while the world is made,
// the depth and thickness of its woods (see woods.go).
type woodsPlan struct {
	at           []uint8 // per surface tile: planNone, planYoung or planGrown
	stride       int     // the step the scan walks the tiles by, prime to their number
	depth, thick []float32
	cover        *coverField // since StyleGround, while the world is made: see groundcover.go
}

// planWoods notes where the trees stand when a world has been made: in a
// world of StyleWoods all of them; in an older one, the ones that keep their
// places when its woods are thinned.
func (w *World) planWoods(seed uint32) {
	W, H := w.width, w.SurfaceHeight()
	at := make([]uint8, W*H)
	mark := func(e *Entity) {
		if e.Kind == EntityTree && e.TY < H {
			at[e.TY*W+e.TX] = planYoung
			if e.Big {
				at[e.TY*W+e.TX] = planGrown
			}
		}
	}
	if w.style >= StyleWoods {
		for _, e := range w.entities {
			mark(e)
		}
	} else {
		thick := w.woodsThickness(seed, int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY)))
		for _, e := range w.thinLayout(thick) {
			mark(e)
		}
	}
	n := len(at)
	stride := max(1, int(float64(n)*0.6180339887)|1)
	for gcd(stride, n) != 1 {
		stride += 2
	}
	w.woods = woodsPlan{at: at, stride: stride}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// planned is what the plan has on tile (tx, ty).
func (w *World) planned(tx, ty int) uint8 {
	if tx < 0 || ty < 0 || tx >= w.width || ty*w.width+tx >= len(w.woods.at) {
		return planNone
	}
	return w.woods.at[ty*w.width+tx]
}

// sproutStump turns the stump of felled tree e into a sapling. It grows into
// a grown tree again if the felled one was grown or the plan has one here.
func (w *World) sproutStump(e *Entity) {
	grows := e.Big || w.planned(e.TX, e.TY) == planGrown
	e.Kind, e.Var, e.Big, e.Ready, e.HoneyAt = EntitySapling, uint8(SeedTree), false, false, 0
	e.HP = maxHP(EntitySapling, false)
	e.Fuel = 0
	if grows {
		e.Fuel = saplingGrows
	}
	e.RegrowAt = w.tick + SeedTree.growTicks()
}

// youngGrows starts young tree e, just grown from a sapling whose Fuel was
// fuel, on its way to a grown tree if it is to be one.
func (w *World) youngGrows(e *Entity, fuel int) {
	e.RegrowAt = 0
	if fuel == saplingGrows {
		e.RegrowAt = w.tick + YoungTreeTicks
	}
}

// growsUp reports whether e is a young tree on its way to a grown one.
func growsUp(e *Entity) bool {
	return e.Kind == EntityTree && e.Ready && !e.Big && e.RegrowAt != 0
}

// growUp makes young tree e a grown one, when its time has come. Now and
// then bees settle in a tree of the woods as it grows.
func (w *World) growUp(e *Entity) {
	if w.tick < e.RegrowAt {
		return
	}
	e.RegrowAt = 0
	e.Big = true
	e.HP = maxHP(EntityTree, true)
	if w.TileAt(e.TX, e.TY) == TileForest && unit(hash2(int32(e.TX), int32(e.TY), uint32(w.tick)^0x41fe)) < 1.0/hivesPerTrees {
		e.Var = hiveVar
	}
}

// stepWoodsFill goes on with the slow scan of the map for gaps in the woods.
func (w *World) stepWoodsFill() {
	n := len(w.woods.at)
	if n == 0 {
		return
	}
	per := (n + fillCycleTicks - 1) / fillCycleTicks
	base := w.tick * uint64(per)
	for k := range per {
		v := base + uint64(k)
		i := int(v%uint64(n)) * w.woods.stride % n
		if w.woods.at[i] == planNone || w.occ[i] != 0 {
			continue
		}
		x, y := i%w.width, i/w.width
		if unit(hash2(int32(x), int32(y), uint32(v/uint64(n))*0x9e3779b1^0x5eed)) >= fillChance {
			continue
		}
		w.sproutAt(x, y)
	}
}

// sproutAt lets a sapling come up on (tx, ty), a gap in the woods' plan, if
// a grown tree has room there and nothing people made, no lair and no cave
// is near.
func (w *World) sproutAt(tx, ty int) {
	if !w.canPlace(tx, ty) || w.nearWorks(tx, ty) {
		return
	}
	c, _ := w.crownOf(EntityTree, tx, ty, true)
	if w.crowdedBy(tx, ty, c, fillRoom) {
		return
	}
	e := w.addEntity(EntitySapling, tx, ty)
	e.Var, e.Ready = uint8(SeedTree), false
	if w.planned(tx, ty) == planGrown {
		e.Fuel = saplingGrows
	}
	e.RegrowAt = w.tick + SeedTree.growTicks()
	w.watch(e)
}

// nearWorks reports whether (tx, ty) is near something people made (a
// sapling aside), a village, a cave mouth, the pyramid, an old chest or a
// boss's lair, where the woods do not grow back on their own.
func (w *World) nearWorks(tx, ty int) bool {
	r := max(thinNearBuilt, thinNearSight)
	for y := ty - r; y <= ty+r; y++ {
		for x := tx - r; x <= tx+r; x++ {
			e := w.entityAt(x, y)
			if e == nil {
				continue
			}
			d2 := (x-tx)*(x-tx) + (y-ty)*(y-ty)
			switch {
			case builtKind(e.Kind, e.Var) && e.Kind != EntitySapling, e.Kind == EntityHut, e.Kind == EntityTotem, e.Kind == EntityBonfire:
				if d2 <= thinNearBuilt*thinNearBuilt {
					return true
				}
			case e.Kind == EntityCaveMouth, e.Kind == EntityCaveRock, e.Kind == EntityPyramid, e.Kind == EntityPyramidStone,
				e.Kind == EntityTreasure, e.Kind == EntityLairProp:
				if d2 <= thinNearSight*thinNearSight {
					return true
				}
			}
		}
	}
	for _, l := range w.bosses.lairs {
		if l.ok && math.Hypot(float64(tx)+0.5-l.x, float64(ty)+0.5-l.y) <= thinNearLair+thinNearSight {
			return true
		}
	}
	return false
}

// crowdedBy reports whether a tall thing of crown c on (tx, ty) would stand
// closer to a tree than the given share of their spacing.
func (w *World) crowdedBy(tx, ty int, c crown, share float64) bool {
	r := int(math.Ceil((c.spacing + giantCrown.spacing) / 2 * share))
	for ny := ty - r; ny <= ty+r; ny++ {
		for nx := tx - r; nx <= tx+r; nx++ {
			e := w.entityAt(nx, ny)
			if e == nil || nx == tx && ny == ty {
				continue
			}
			o, ok := w.crownOf(e.Kind, nx, ny, e.Big)
			if e.Kind == EntitySapling && SeedKind(e.Var) == SeedTree {
				o, ok = w.crownOf(EntityTree, nx, ny, e.Fuel == saplingGrows) // what it will be
			}
			if !ok {
				continue
			}
			d := (c.spacing + o.spacing) / 2 * share
			if dx, dy := float64(nx-tx), float64(ny-ty); dx*dx+dy*dy < d*d {
				return true
			}
		}
	}
	return false
}

// capHives takes the bees out of trees until there is a hive to about
// hivesPerTrees trees of the woods, giants' hives counted, choosing by a
// hash of the place. A hive someone has robbed, and a giant's, stays.
func (w *World) capHives() int {
	trees, hives := 0, 0
	var free []*Entity
	for _, e := range w.entities {
		switch {
		case e.Kind == EntityTree && w.TileAt(e.TX, e.TY) == TileForest:
			trees++
			if e.HasHive() {
				hives++
				if e.HoneyAt == 0 {
					free = append(free, e)
				}
			}
		case e.Kind == EntityGiantTree:
			trees++
			if e.HasHive() {
				hives++
			}
		}
	}
	excess := hives - trees/hivesPerTrees
	if excess <= 0 {
		return 0
	}
	key := func(e *Entity) uint32 { return hash2(int32(e.TX), int32(e.TY), 0xb33) }
	slices.SortFunc(free, func(a, b *Entity) int {
		if c := cmp.Compare(key(a), key(b)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	gone := min(excess, len(free))
	for _, e := range free[:gone] {
		e.Var = 0
	}
	return gone
}

// DevFell fells every standing tree within r tiles of (x, y), as an axe
// would but leaving nothing on the ground, and returns how many fell. For
// cmd/shot.
func (w *World) DevFell(x, y, r float64) int {
	n := 0
	for _, e := range w.sortedEntities() {
		if e.Kind != EntityTree || !e.Ready || math.Hypot(float64(e.TX)+0.5-x, float64(e.TY)+0.5-y) > r {
			continue
		}
		e.Ready, e.Var, e.RegrowAt = false, 0, w.tick+StumpTicks
		w.watch(e)
		n++
	}
	return n
}

// DevWoodsThickness is how thick each surface tile's wood is, 0 (ordinary)
// to 1 (thick); see woodsThickness. For cmd/worldstat's map of the woods.
func (w *World) DevWoodsThickness() []float32 {
	return w.woodsThickness(uint32(w.seed)^uint32(w.seed>>32), int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY)))
}

// DevPassTime lets ticks go by for what grows: the regrowth of everything
// and the scan for gaps in the woods, but nobody moves. For cmd/shot.
func (w *World) DevPassTime(ticks uint64) {
	for range ticks {
		w.tick++
		w.stepEntities()
		w.stepWoodsFill()
	}
}
