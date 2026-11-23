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
	homeApart      = 4    // tiles kept between two homes
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

func (w *World) nearHome(tx, ty, r int) bool {
	for _, h := range w.fauna.homes {
		if max(abs(h.tx-tx), abs(h.ty-ty)) < r {
			return true
		}
	}
	return false
}

// placeHomes digs up to n homes for kind k on the given spawn tiles, where
// ok allows, and puts a family around each. It returns how many it dug.
func (w *World) placeHomes(k CreatureKind, spots []int32, n int, ok func(tx, ty int) bool) int {
	d := k.def()
	dug := 0
	for range n {
		for range homeSpawnTries {
			i := spots[w.random().IntN(len(spots))]
			tx, ty := int(i)%w.width, int(i)/w.width
			if !ok(tx, ty) || !w.canPlace(tx, ty) || w.nearHome(tx, ty, homeApart) {
				continue
			}
			e := w.addEntity(d.homeKind, tx, ty)
			w.fauna.homes = append(w.fauna.homes, homeRec{kind: d.homeKind, tx: tx, ty: ty, id: e.ID})
			for range d.perHome {
				w.addAtHome(k, e, false)
			}
			dug++
			break
		}
	}
	return dug
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
// homes that no player is near, and reports whether it could.
func (w *World) repopulateAtHome(k CreatureKind) bool {
	var homes []*Entity
	for _, h := range w.fauna.homes {
		if e := w.entities[h.id]; e != nil && e.Kind == k.def().homeKind {
			if _, d := w.nearestPlayer(homeCentre(e)); d >= creatureSpawnAway {
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
