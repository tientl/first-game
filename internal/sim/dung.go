package sim

import "math"

// Dung. The grazing beasts leave dung behind them as they go: the big ones
// a heap, the small ones a little; a beast tamed and fed at the player's
// heels leaves it twice as often. It lies on the ground like anything else
// that falls there and is picked up the same way.
//
// Held in the hand, dung feeds what grows: V at a sapling, a sprouting crop,
// or a bush, tree or flower picked bare halves the time it still needs. On
// an empty bed of tilled soil it feeds the bed, so whatever is planted there
// next grows twice as fast.

// Dung tuning.
const (
	DungEvery     = DayTicks / 3 // on average, between one grazer's droppings
	DungCap       = 160          // heaps lying about the world at most
	DungLifeTicks = DayTicks     // a heap nobody takes dries to nothing after this
	DungBoost     = 2            // fed, what is left of the growing is cut by this much
)

// dungOf is how much dung a beast of kind k leaves at a time; 0 for those
// that leave none worth picking up.
func dungOf(k CreatureKind) int {
	switch k {
	case CreatureBrachio:
		return 3
	case CreatureTrike, CreatureToro, CreatureStego, CreatureMammoth, CreatureRex, CreatureAnkylo:
		return 2
	case CreatureThesc, CreatureOrnitho:
		return 1
	}
	return 0
}

// maybeDung lets c leave dung behind it now and then.
func (w *World) maybeDung(c *Creature) {
	n := dungOf(c.Kind)
	if n == 0 || c.gone || c.Ridden() {
		return
	}
	switch c.State {
	case CreatureHidden, CreatureHooked, CreatureTrapped:
		return
	}
	every := float64(DungEvery)
	if c.Tamed() {
		every /= 2
	}
	if w.random().Float64() >= 1/every {
		return
	}
	if !w.walkableAt(c.X, c.Y) || w.dungLying() >= DungCap {
		return
	}
	back := -0.5
	if c.FaceLeft {
		back = 0.5
	}
	w.nextDrop++
	w.drops = append(w.drops, &Drop{
		ID: w.nextDrop, Item: ItemDung, Count: n,
		X: c.X + back, Y: c.Y + 0.1, FromX: c.X + back*0.6, FromY: c.Y - 0.4, Born: w.tick,
	})
}

// dungLying counts the heaps of dung on the ground.
func (w *World) dungLying() int {
	n := 0
	for _, d := range w.drops {
		if d.Item == ItemDung && !d.gone {
			n++
		}
	}
	return n
}

// dropLife is how long a pile of k lies on the ground before it is gone.
func dropLife(k ItemKind) uint64 {
	if k == ItemDung {
		return DungLifeTicks
	}
	if k.Legendary() {
		return math.MaxUint64 // the only one there is: it never rots; see legendary.go
	}
	return DropLifeTicks
}

// Feedable reports whether dung would do e any good at tick: something
// still growing, or an empty bed not yet fed.
func (e *Entity) Feedable(tick uint64) bool {
	switch e.Kind {
	case EntityFarmland:
		return !e.Big
	case EntitySapling:
		return tick < e.RegrowAt
	case EntityCrop, EntityTree, EntityBerryBush, EntityFigTree, EntityGinkgoTree, EntityPalm, EntityBananaTree,
		EntityFlower, EntityGlowPlant, EntityGrassTuft, EntityReed, EntityCactus, EntityMushroom, EntityFlora:
		return !e.Ready && tick < e.RegrowAt
	}
	return false
}

// fertilize spreads one dung from p's hand on the growing thing in front of
// p, or failing that on one beside them.
func (w *World) fertilize(p *Player) { w.fertilizeOn(p, nil) }

// fertilizeAt spreads dung on the growing thing on tile (tx, ty) if p
// reaches it, else as fertilize does.
func (w *World) fertilizeAt(p *Player, tx, ty int) {
	e := w.entityAt(tx, ty)
	if e != nil && (!e.Feedable(w.tick) ||
		math.Hypot(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y) > PlaceReach) {
		e = nil
	}
	w.fertilizeOn(p, e)
}

// fertilizeOn spreads dung on e, or when e is nil on the growing thing in
// front of p, or beside them.
func (w *World) fertilizeOn(p *Player, e *Entity) {
	px, py := int(math.Floor(p.X)), int(math.Floor(p.Y))
	for _, d := range []Dir{p.Dir, DirDown, DirRight, DirUp, DirLeft} {
		if e != nil {
			break
		}
		dx, dy := d.Delta()
		if f := w.entityAt(px+dx, py+dy); f != nil && f.Feedable(w.tick) {
			e = f
			break
		}
	}
	if e == nil {
		p.notify(NoticeNothingToFeed, 0)
		return
	}
	p.Inv.TakeSlot(p.Selected, 1)
	if e.Kind == EntityFarmland {
		e.Big = true // fed: see plant
		e.RegrowAt = w.tick + FarmlandDecayTicks
	} else {
		e.RegrowAt = w.tick + (e.RegrowAt-w.tick)/DungBoost
	}
	w.watch(e)
	p.notify(NoticeFertilized, uint8(e.Kind))
}
