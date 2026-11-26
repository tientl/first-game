package sim

import "math"

// The lairs. A boss is met at a place of its own, not out in the open:
//
//   - Titanoboa lies in a hollow at the foot of an ancient tree, among the
//     stones of a ruin: a nest of leaves, the bones of what it ate, a skin it
//     shed. Round about, trees lie broken, bones and the tracks of its belly
//     lead in, and small animals keep away.
//   - The Megalodon keeps a patch of deep dark water with a wreck in it, and
//     a sandbar near by where a whale's bones lie.
//
// When a fight starts the way out closes: roots rise in a ring round
// Titanoboa's hollow, and the sea turns in a whirlpool round the shark that
// carries boats and swimmers back in. The ring opens when the boss dies or
// goes back to sleep, when its foe has kept well away from it a while, and
// in any case after ArenaMaxTicks, so that nobody is ever shut in for good.
//
// With nobody to fight, Titanoboa goes out now and then after a small
// animal near its hollow and comes back to coil; the shark circles its
// water and snaps at fish.

// LairProp kinds: the Var of an EntityLairProp.
const (
	LairNest  uint8 = iota // the bed of leaves the snake coils on
	LairCover              // the fronds it hides under, a tile in front of the nest
	LairBones              // what is left of its meals
	LairSkin               // a skin it shed
	LairTrail              // the track of its belly
	LairRuin               // a mossy stone of the old ruin
	LairStump              // a tree it broke
	LairWhale              // a whale's bones on the sandbar
	NumLairProps
)

// Lair tuning.
const (
	TitanArena     = 12.0 // tiles from the lair to the ring of roots
	MegaArena      = 20.0 // and to the whirlpool
	ArenaHalfWidth = 0.75 // half the ring's thickness
	ArenaMaxTicks  = 150 * TicksPerSecond
	arenaCalmTicks = 20 * TicksPerSecond // the foe this long away from the boss: the ring opens
	arenaCalmDist  = 14.0
	WhirlBand      = 3.0 // the whirlpool pulls from this far inside its ring
	whirlPull      = 1.3 // times a boat's best speed, at the ring

	LairSignRange  = 30.0 // signs of a boss lie this far out from its lair
	lairDread      = 9.0  // small animals this close to Titanoboa bolt...
	lairWatchers   = 40.0 // ...while a player is this close to see it
	titanHuntEvery = 35 * TicksPerSecond
	titanHuntTicks = 12 * TicksPerSecond
	titanHuntRange = 13.0
	megaCircle     = 4.5 // the shark's round of its water
	megaSnapEvery  = 20 * TicksPerSecond
)

// arenaRadius is how far from boss k's lair its fight is shut in; 0 for a
// boss that fights in the open.
func arenaRadius(k CreatureKind) float64 {
	switch k {
	case CreatureTitanoboa:
		return TitanArena
	case CreatureMegalodon:
		return MegaArena
	}
	return 0
}

// LairStatus is a lair as the players near it are shown it.
type LairStatus struct {
	Kind     CreatureKind
	X, Y     float64
	Radius   float64 // of its arena; 0 if it has none
	Closed   bool    // the arena is shut
	Awake    bool    // its boss is in a fight
	Defeated bool
}

// LairsNear returns the lairs within r tiles of player id.
func (w *World) LairsNear(id PlayerID, r float64) []LairStatus {
	p := w.players[id]
	if p == nil {
		return nil
	}
	var out []LairStatus
	for i := range w.bosses.lairs {
		l := &w.bosses.lairs[i]
		if !l.ok || math.Hypot(l.x-p.X, l.y-p.Y) > r {
			continue
		}
		k := BossKind(i)
		out = append(out, LairStatus{
			Kind: k, X: l.x, Y: l.y, Radius: arenaRadius(k),
			Closed: l.arena, Awake: l.awake && l.c != nil, Defeated: l.defeated,
		})
	}
	return out
}

// ArenaTile reports whether tile (tx, ty) is part of the ring round a lair
// at (lx, ly) whose arena is r tiles wide. The client draws the roots on
// the same tiles.
func ArenaTile(lx, ly, r float64, tx, ty int) bool {
	return math.Abs(math.Hypot(float64(tx)+0.5-lx, float64(ty)+0.5-ly)-r) < ArenaHalfWidth
}

// arenaBlocks reports whether risen roots stand on tile (tx, ty).
func (w *World) arenaBlocks(tx, ty int) bool {
	l := &w.bosses.lairs[0] // only Titanoboa's ring is a wall
	return l.arena && ArenaTile(l.x, l.y, TitanArena, tx, ty)
}

// stepArena shuts and opens boss k's arena, once a tick.
func (w *World) stepArena(l *bossLair, k CreatureKind) {
	r := arenaRadius(k)
	if r == 0 {
		return
	}
	c := l.c
	if c == nil || c.gone || !l.awake {
		l.arena, l.opened, l.arenaT, l.calm = false, false, 0, 0
		return
	}
	if !l.arena && !l.opened {
		l.arena, l.arenaT, l.calm = true, 0, 0
	}
	if !l.arena {
		return
	}
	l.arenaT++
	if foe := w.players[l.foe]; foe == nil || foe.Dead || math.Hypot(foe.X-c.X, foe.Y-c.Y) > arenaCalmDist {
		l.calm++
	} else {
		l.calm = 0
	}
	if l.arenaT > ArenaMaxTicks || l.calm > arenaCalmTicks {
		l.arena, l.opened = false, true // open for the rest of this fight
		return
	}
	if k == CreatureMegalodon {
		w.whirl(l, r)
	}
}

// whirl is a tick of the whirlpool round lair l: whoever is on the water
// near the ring is carried back in, and a little round.
func (w *World) whirl(l *bossLair, r float64) {
	for _, id := range w.playerOrder {
		p := w.players[id]
		dx, dy := p.X-l.x, p.Y-l.y
		d := math.Hypot(dx, dy)
		if p.Dead || d < r-WhirlBand || d > r+2 {
			continue
		}
		k := min(1, (d-(r-WhirlBand))/WhirlBand)
		ix, iy := -dx/d, -dy/d
		ix, iy = ix-iy*0.25, iy+ix*0.25
		if b := w.boatByID(p.boat); p.boat != 0 && b != nil {
			pull := boatDefs[b.Kind].speed * sailBoost(&p.Inv) * whirlPull * k
			if w.boatFits(b.Kind, b.X+ix*pull, b.Y) {
				b.X += ix * pull
			}
			if w.boatFits(b.Kind, b.X, b.Y+iy*pull) {
				b.Y += iy * pull
			}
			p.X, p.Y = b.X, b.Y
		} else if p.boat == 0 && w.inWater(p.X, p.Y) {
			pull := PlayerSpeed * SwimSpeed * swimBoost(&p.Inv) * whirlPull * k
			if w.swimmable(p.X+ix*pull, p.Y) {
				p.X += ix * pull
			}
			if w.swimmable(p.X, p.Y+iy*pull) {
				p.Y += iy * pull
			}
		}
	}
}

// watched reports whether a player is close enough to lair l to see what
// its boss does with nobody to fight.
func (w *World) watched(l *bossLair) bool {
	_, d := w.nearestPlayer(l.x, l.y)
	return d <= lairWatchers
}

// small reports whether c is an animal small enough for Titanoboa to
// swallow, and to fear it.
func (c *Creature) small() bool {
	d := c.Kind.def()
	return !c.gone && !c.Tamed() && !c.Kind.Boss() && c.Kind != CreatureTribesman && c.Kind != CreatureSnake &&
		d.radius <= 0.45 && !d.flies && !d.water && !d.deep && !c.State.OutOfReach()
}

// lairDread sends the small animals near Titanoboa c running from it.
func (w *World) lairDread(c *Creature, l *bossLair) {
	if w.tick%10 != 0 || !w.watched(l) {
		return
	}
	for _, o := range w.fauna.creatures {
		if o.small() && o.hunter == 0 && math.Hypot(o.X-c.X, o.Y-c.Y) <= lairDread {
			o.hunter, o.huntedT = c.ID, huntedTicks
		}
	}
}

// titanProwl is Titanoboa with nobody to fight: now and then it goes after
// a small animal near its hollow, swallows it if it can, and comes home.
// It reports whether that was its step.
func (w *World) titanProwl(c *Creature, l *bossLair) bool {
	w.lairDread(c, l)
	if l.prey == 0 {
		if l.idle++; l.idle < titanHuntEvery || w.tick%TicksPerSecond != 0 || !w.watched(l) {
			return false
		}
		var best *Creature
		bestD := titanHuntRange
		for _, o := range w.fauna.creatures {
			if d := math.Hypot(o.X-l.x, o.Y-l.y); o.small() && d < bestD {
				best, bestD = o, d
			}
		}
		if best == nil {
			l.idle = titanHuntEvery - 5*TicksPerSecond // it looks again soon
			return false
		}
		l.prey, l.huntT = best.ID, titanHuntTicks
	}
	prey := w.creatureByID(l.prey)
	if l.huntT--; prey == nil || !prey.small() || l.huntT <= 0 || math.Hypot(prey.X-l.x, prey.Y-l.y) > titanHuntRange+3 {
		l.prey, l.idle = 0, 0
		return false
	}
	prey.hunter, prey.huntedT = c.ID, huntedTicks
	c.faceToward(prey.X)
	if math.Hypot(prey.X-c.X, prey.Y-c.Y) <= 2.2 {
		prey.gone = true // swallowed whole
		l.prey, l.idle = 0, 0
		c.State, c.timer = CreatureStrike, bossWindup+bossRecover
		l.landed, l.grab = true, false
		return true
	}
	c.State = CreatureWander
	if !w.runAt(c, prey.X, prey.Y, c.Kind.def().run*0.8, 1.5) {
		l.huntT -= 3 // something in the way: it soon gives up
	}
	return true
}

// megaIdle is the Megalodon with nobody to fight: round and round its
// water under the surface, and now and then up to snap at a fish.
func (w *World) megaIdle(c *Creature, l *bossLair) {
	c.State = CreatureLurk
	a := math.Atan2(c.Y-l.y, c.X-l.x) + 0.12
	if !w.runAt(c, l.x+math.Cos(a)*megaCircle, l.y+math.Sin(a)*megaCircle, c.Kind.def().walk*0.7, 0) {
		w.swim(c)
	}
	if l.idle++; l.idle >= megaSnapEvery && w.watched(l) {
		l.idle = 0
		c.State, c.timer, l.landed = CreatureStrike, bossWindup+bossRecover, true
	}
}

// --- dressing the lairs ---------------------------------------------------------

// dressLairs sets every lair's scene: at world creation, and on loading a
// world saved before the lairs were dressed.
func (w *World) dressLairs() {
	for i := range w.bosses.lairs {
		if l := &w.bosses.lairs[i]; l.ok {
			switch BossKind(i) {
			case CreatureTitanoboa:
				w.dressTitanLair(l)
			case CreatureMegalodon:
				if l.bar {
					w.lairPut(LairWhale, l.barX, l.barY)
				}
			}
		}
	}
}

// lairPut sets lair prop v on tile (tx, ty), in place of whatever grows
// there; nothing a player built is touched.
func (w *World) lairPut(v uint8, tx, ty int) bool {
	if !w.inBounds(tx, ty) || !w.tiles[ty*w.width+tx].Walkable() {
		return false
	}
	if e := w.entityAt(tx, ty); e != nil {
		if !wild(e.Kind) {
			return false
		}
		w.removeEntity(e)
	}
	w.addEntity(EntityLairProp, tx, ty).Var = v
	return true
}

// dressTitanLair makes Titanoboa's hollow round lair l, and the signs of it
// in the woods about. It draws no random numbers: the same world always
// gets the same lair.
func (w *World) dressTitanLair(l *bossLair) {
	lx, ly := int(math.Floor(l.x)), int(math.Floor(l.y))
	seed := uint32(w.seed) ^ 0x71a1
	h := func(a, b int) float64 { return unit(hash2(int32(a), int32(b), seed)) }
	// The hollow: nothing grows where it lies.
	for ty := ly - 3; ty <= ly+3; ty++ {
		for tx := lx - 4; tx <= lx+4; tx++ {
			if e := w.entityAt(tx, ty); e != nil && wild(e.Kind) && math.Hypot(float64(tx-lx), float64(ty-ly)*1.25) <= 3.7 {
				w.removeEntity(e)
			}
		}
	}
	if w.giantFits(lx, ly-2) {
		w.raiseGiant(lx, ly-2) // the ancient tree it lies at the foot of
	}
	w.lairPut(LairNest, lx, ly)
	w.lairPut(LairCover, lx, ly+1)
	for _, o := range [][2]int{{-3, -1}, {3, 0}, {-3, 2}, {4, -2}, {2, 3}, {-5, 0}} {
		w.lairPut(LairRuin, lx+o[0], ly+o[1])
	}
	w.lairPut(LairSkin, lx+2, ly+1)
	w.lairPut(LairSkin, lx-6, ly+3)
	at := func(a, r float64) (int, int) {
		return lx + int(math.Round(math.Cos(a)*r)), ly + int(math.Round(math.Sin(a)*r*0.8))
	}
	free := func(tx, ty int) bool {
		e := w.entityAt(tx, ty)
		return e == nil || e.Kind != EntityLairProp
	}
	for i := range 17 { // bones: thick by the nest, a few far out
		r := 2 + h(i, 2)*3.5
		if i >= 7 {
			r = 6 + h(i, 2)*(LairSignRange-8)
		}
		if tx, ty := at(h(i, 1)*2*math.Pi, r); free(tx, ty) && (tx != lx || ty < ly-1 || ty > ly+1) {
			w.lairPut(LairBones, tx, ty)
		}
	}
	for s := range 3 { // the ways it goes out to hunt
		a := (h(s, 5)*0.6 + float64(s)) / 3 * 2 * math.Pi
		for r := 3.5; r <= 25; r += 2.2 {
			if tx, ty := at(a+math.Sin(r*0.35+float64(s))*0.12, r); free(tx, ty) {
				w.lairPut(LairTrail, tx, ty)
			}
		}
	}
	for ty := ly - 22; ty <= ly+22; ty++ { // trees it broke
		for tx := lx - 22; tx <= lx+22; tx++ {
			e := w.entityAt(tx, ty)
			if e == nil || e.Kind != EntityTree {
				continue
			}
			share := 0.05
			if d := math.Hypot(float64(tx-lx), float64(ty-ly)); d > 22 {
				continue
			} else if d < 9 {
				share = 0.22
			}
			if h(tx, ty) < share {
				w.lairPut(LairStump, tx, ty)
			}
		}
	}
}

// raiseSandbar makes the sandbar near the shark's lair l, out of the sea:
// only when the world is made, so every load finds the same land.
func (w *World) raiseSandbar(l *bossLair) {
	seed := uint32(w.seed) ^ 0x5a9d
	a0 := unit(hash2(1, 2, seed)) * 2 * math.Pi
	for i := range 12 {
		a := a0 + float64(i)*2*math.Pi/12
		cx, cy := int(math.Floor(l.x+math.Cos(a)*12)), int(math.Floor(l.y+math.Sin(a)*12))
		open := true
		for ty := cy - 4; ty <= cy+4 && open; ty++ {
			for tx := cx - 5; tx <= cx+5; tx++ {
				if !w.inBounds(tx, ty) || ty >= w.SurfaceHeight() || w.tiles[ty*w.width+tx] != TileWater {
					open = false
					break
				}
			}
		}
		if !open {
			continue
		}
		for ty := cy - 1; ty <= cy+1; ty++ {
			for tx := cx - 2; tx <= cx+2; tx++ {
				if math.Hypot(float64(tx-cx)/2.6, float64(ty-cy)/1.5) <= 1 {
					w.tiles[ty*w.width+tx] = TileSand
				}
			}
		}
		for _, c := range w.fauna.creatures { // nothing swims in sand
			if c != l.c && math.Abs(c.X-float64(cx)-0.5) < 4 && math.Abs(c.Y-float64(cy)-0.5) < 3 {
				c.gone = true
			}
		}
		w.fauna.shallow = nil
		l.bar, l.barX, l.barY = true, cx, cy
		return
	}
}
