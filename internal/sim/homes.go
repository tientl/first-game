package sim

import "math"

// Rabbits live in burrows and snakes in dens: EntityBurrow and EntityDen,
// dug when the world is made, a few animals to each. A home is where its
// animals wander back to, where new ones come out, and a place to hide:
//
//   - A bolting rabbit makes for its burrow, unless the player is nearer to
//     it, and dives in. It peeks out again once nobody is close.
//   - Snakes go down into their den at nightfall and come out at dawn.
//
// An animal down its home is out of reach. Homes are entities, so they are
// saved with the world; a world loaded from a save made before homes existed
// gets them back on its first tick.

// Home tuning.
const (
	RabbitBurrowRun = 6.0                // a bolting rabbit makes for its burrow from this close
	RabbitHideTicks = 8 * TicksPerSecond // the least time a rabbit stays down its burrow
	RabbitPeekDist  = 4.0                // and it stays down while a player is this close
	SnakeDenBelow   = 0.25               // ambient light below which snakes keep to their dens

	homeReach      = 0.45 // close enough to a home's mouth to go in
	homeSpread     = 1.6  // animals start this close to their home
	homeApart      = 4    // tiles kept between any two homes; see creatureDef.homeSpacing
	homeSpawnTries = 12
)

// homeRec remembers where a home was dug, to put it back after loading.
type homeRec struct {
	kind   EntityKind
	tx, ty int
	id     EntityID
}

func homeCentre(e *Entity) (float64, float64) { return float64(e.TX) + 0.5, float64(e.TY) + 0.5 }

// home returns c's home, or nil if it has none (any more).
func (w *World) home(c *Creature) *Entity {
	if c.home == 0 {
		return nil
	}
	e := w.entities[c.home]
	if e == nil || e.Kind != c.Kind.def().homeKind {
		return nil
	}
	return e
}

// nearHome reports whether a home of kind homeKind dug at (tx, ty) would be
// too close to another: homes of the same kind keep spacing apart (their
// animals' ranges), any two keep homeApart.
func (w *World) nearHome(tx, ty int, homeKind EntityKind, spacing int) bool {
	for _, h := range w.fauna.homes {
		r := homeApart
		if h.kind == homeKind {
			r = max(r, spacing)
		}
		if max(abs(h.tx-tx), abs(h.ty-ty)) < r {
			return true
		}
	}
	return false
}

// homeTile is how well tile t suits a home of d's kind, 0 (never) to 1.
func (d *creatureDef) homeTile(t Tile) float64 {
	if d.homeOn == nil {
		return 1
	}
	return d.homeOn[t]
}

// meanFamily is how many animals a home of d's kind starts with, on average.
func (d *creatureDef) meanFamily() float64 {
	if d.family[1] == 0 {
		return float64(max(d.perHome, 1))
	}
	return float64(d.family[0]+d.family[1]) / 2
}

// homeWeight is how much room for homes of kind k the spawn tiles hold: each
// tile counts as much as it suits a home.
func (w *World) homeWeight(k CreatureKind, spots []int32) float64 {
	d, sum := k.def(), 0.0
	for _, i := range spots {
		sum += d.homeTile(w.tiles[i])
	}
	return sum
}

// nearTile reports whether a tile of kind t lies within r tiles of (tx, ty).
func (w *World) nearTile(tx, ty int, t Tile, r int) bool {
	for y := ty - r; y <= ty+r; y++ {
		for x := tx - r; x <= tx+r; x++ {
			if w.inBounds(x, y) && w.tiles[y*w.width+x] == t {
				return true
			}
		}
	}
	return false
}

// placeHomes digs up to n homes for kind k on the given spawn tiles, where
// ok allows and the kind's habits agree: the ground suits it (homeOn), cover
// is near (cover, coverDist) and no other home of its kind is too close
// (homeSpacing). Each gets a family of family[0] to family[1] animals. It
// returns how many animals it placed.
func (w *World) placeHomes(k CreatureKind, spots []int32, n int, ok func(tx, ty int) bool) int {
	d := k.def()
	r := w.random()
	placed := 0
	for range n {
		for range homeSpawnTries * 10 { // habits make most tiles unfit: look hard
			i := spots[r.IntN(len(spots))]
			tx, ty := int(i)%w.width, int(i)/w.width
			if r.Float64() >= d.homeTile(w.tiles[i]) || !ok(tx, ty) || !w.canPlace(tx, ty) ||
				w.nearHome(tx, ty, d.homeKind, d.homeSpacing) ||
				(d.coverDist > 0 && !w.nearTile(tx, ty, d.cover, d.coverDist)) {
				continue
			}
			e := w.addEntity(d.homeKind, tx, ty)
			w.fauna.homes = append(w.fauna.homes, homeRec{kind: d.homeKind, tx: tx, ty: ty, id: e.ID})
			family := d.perHome
			if d.family[1] > 0 {
				family = d.family[0] + r.IntN(d.family[1]-d.family[0]+1)
			}
			for range family {
				w.addAtHome(k, e, false)
			}
			placed += family
			break
		}
	}
	return placed
}

// addAtHome puts a new animal of kind k by home e, or right at its mouth
// when it is just coming out.
func (w *World) addAtHome(k CreatureKind, e *Entity, atMouth bool) *Creature {
	hx, hy := homeCentre(e)
	x, y := hx, hy
	for range homeSpawnTries {
		if atMouth {
			break
		}
		a, r := w.random().Float64()*2*math.Pi, 0.6+w.random().Float64()*(homeSpread-0.6)
		if nx, ny := hx+math.Cos(a)*r, hy+math.Sin(a)*r; w.creatureFits(k, nx, ny) {
			x, y = nx, ny
			break
		}
	}
	c := w.addCreature(k, x, y)
	c.home, c.homeX, c.homeY = e.ID, hx, hy
	return c
}

// repopulateAtHome brings out one new animal of kind k from one of its
// homes that no player is near and that has room (fewer than perHome living
// there), and reports whether it could.
func (w *World) repopulateAtHome(k CreatureKind) bool {
	d := k.def()
	living := make(map[EntityID]int)
	for _, c := range w.fauna.creatures {
		if c.Kind == k && c.home != 0 && !c.gone {
			living[c.home]++
		}
	}
	var homes []*Entity
	for _, h := range w.fauna.homes {
		if e := w.entities[h.id]; e != nil && e.Kind == d.homeKind && living[e.ID] < max(d.perHome, 1) {
			if _, dist := w.nearestPlayer(homeCentre(e)); dist >= creatureSpawnAway {
				homes = append(homes, e)
			}
		}
	}
	if len(homes) == 0 {
		return false
	}
	w.addAtHome(k, homes[w.random().IntN(len(homes))], true)
	return true
}

// reseatHomes makes sure every recorded home stands. A world loaded from a
// save made before homes existed has lost them: they are dug again where the
// tile is still free, and dropped where something was built over them.
func (w *World) reseatHomes() {
	moved := make(map[EntityID]EntityID)
	kept := w.fauna.homes[:0]
	for _, h := range w.fauna.homes {
		e := w.entityAt(h.tx, h.ty)
		switch {
		case e != nil && e.Kind == h.kind:
		case e == nil && w.canPlace(h.tx, h.ty):
			e = w.addEntity(h.kind, h.tx, h.ty)
		default:
			continue
		}
		if e.ID != h.id {
			moved[h.id] = e.ID
			h.id = e.ID
		}
		kept = append(kept, h)
	}
	w.fauna.homes = kept
	for _, c := range w.fauna.creatures {
		if id, ok := moved[c.home]; ok {
			c.home = id
		}
	}
}

// goIn sends c down home e.
func (c *Creature) goIn(e *Entity) {
	c.X, c.Y = homeCentre(e)
	c.State, c.timer = CreatureHidden, RabbitHideTicks
}

// stepHidden runs an animal that is down its home: it comes out when its
// kind is ready to.
func (w *World) stepHidden(c *Creature) {
	if c.Kind.def().homeKind != EntityNone && w.home(c) == nil {
		c.State, c.timer = CreatureIdle, 0 // the home is gone: out it comes
		return
	}
	if c.Kind == CreatureSnake {
		if Ambient(w.tick) >= SnakeDenBelow {
			c.State, c.timer = CreatureIdle, 0 // morning
		}
		return
	}
	if c.timer > 0 {
		c.timer--
		return
	}
	peek := RabbitPeekDist
	if c.Kind == CreatureCrab { // under the water rather than down a burrow; see crab.go
		peek = CrabPeekDist
	}
	if _, d := w.nearestPlayer(c.X, c.Y); d < peek {
		c.timer = TicksPerSecond // someone is still about: look again later
		return
	}
	c.State, c.timer = CreatureIdle, 0
}

// burrowToRun is the burrow a rabbit bolting from p should make for, or nil
// when it is too far or p is nearer to it.
func (w *World) burrowToRun(c *Creature, p *Player) *Entity {
	e := w.home(c)
	if e == nil {
		return nil
	}
	hx, hy := homeCentre(e)
	mine := math.Hypot(hx-c.X, hy-c.Y)
	if mine > RabbitBurrowRun || math.Hypot(hx-p.X, hy-p.Y) < mine {
		return nil
	}
	return e
}

// headHome moves a snake towards its den at nightfall, and in once there. It
// reports whether the snake is busy going home.
func (w *World) headHome(c *Creature) bool {
	e := w.home(c)
	if e == nil || Ambient(w.tick) >= SnakeDenBelow {
		return false
	}
	hx, hy := homeCentre(e)
	dist := math.Hypot(hx-c.X, hy-c.Y)
	step := c.Kind.def().run * 0.6
	if dist <= homeReach+step {
		c.goIn(e)
		return true
	}
	c.State = CreatureWander
	if !w.moveCreature(c, (hx-c.X)/dist*step, (hy-c.Y)/dist*step) {
		c.goIn(e) // stuck on the way: it slips in through a gap anyway
	}
	return true
}
