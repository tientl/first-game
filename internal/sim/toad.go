package sim

import "math"

// The toads live in the marsh and round the ponds.
//
//   - The green toad is warty and bright: a warning. Its skin is poison.
//     Whoever comes close enough to touch it, or strikes it with a bare
//     fist, is poisoned, as by a spider's bite. Killed, it leaves its skin
//     and a poison gland.
//   - The glow toad is harmless and shy: it hops away from a player. At
//     night it glows with a soft green light. It leaves a glow gland.
//
// Both go about in hops: a short leap, a pause, another leap. Where water
// lilies or lotus float they hop out onto the leaves and sit there a long
// while.

// Toad tuning.
const (
	ToadHopTicks    = 8                  // a hop lasts this long
	ToadHop         = 0.85               // and covers this many tiles
	toadHopStretch  = 0.35               // or this much more, to land right on a leaf
	ToadTouch       = 0.75               // a green toad poisons a player this close
	ToadTouchDamage = 2.0                // the sting of its skin
	ToadVenom       = 5 * TicksPerSecond // how long its poison lasts
	ToadTouchCD     = 3 * TicksPerSecond
	ToadPuffTicks   = 8   // it swells up as it is touched
	ToadShyDist     = 2.6 // a glow toad hops away from a player this close
	ToadGlowRadius  = 1.7 // tiles of a glow toad's light at night
	toadPadSight    = 3.6 // a toad sees a lily pad this far off
	toadPadChance   = 0.6 // and makes for it this often
	toadPadSitTimes = 5   // it sits this many times longer on a pad
	toadSeatChance  = 0.5 // a world starts with this share of its toads on pads, where there are any
)

var toadDef = creatureDef{
	name: "Cóc độc", hp: 2,
	walk: ToadHop / ToadHopTicks, run: ToadHop / ToadHopTicks,
	radius: 0.18, reach: 1.4,
	yield: ItemToadSkin, yieldN: 1, drops: []Ingredient{{ItemPoisonGland, 1}},
	pondside: true, pads: true,
	leash: 3, idle: [2]int{24, 110}, roam: [2]int{ToadHopTicks, ToadHopTicks},
	spawnOn: []Tile{TileMarsh, TileGrass, TileForest}, tilesPer: 90,
}

var glowToadDef = creatureDef{
	name: "Cóc phát sáng", hp: 1,
	walk: ToadHop / ToadHopTicks, run: ToadHop / ToadHopTicks,
	radius: 0.18, reach: 1.4,
	yield: ItemGlowGland, yieldN: 1,
	pondside: true, pads: true,
	leash: 3, idle: [2]int{20, 90}, roam: [2]int{ToadHopTicks, ToadHopTicks},
	spawnOn: []Tile{TileMarsh, TileGrass, TileForest}, tilesPer: 130,
}

func init() {
	creatureDefs[CreatureToad] = toadDef
	creatureDefs[CreatureGlowToad] = glowToadDef
}

// IsToad reports whether k is one of the toads.
func IsToad(k CreatureKind) bool { return k == CreatureToad || k == CreatureGlowToad }

// pondside reports whether a toad may appear on tile (tx, ty): in the
// marsh, or on any ground with a pond beside it.
func (w *World) pondside(tx, ty int) bool {
	return w.TileAt(tx, ty) == TileMarsh || w.byPond(tx, ty)
}

// onPad reports whether c sits on a lily pad.
func (w *World) onPad(c *Creature) bool {
	return w.padAt(int(math.Floor(c.X)), int(math.Floor(c.Y)))
}

// spawnToads seats some of a new world's toads on the lily pads near them.
func (w *World) spawnToads() {
	for _, c := range w.fauna.creatures {
		if !IsToad(c.Kind) || w.random().Float64() >= toadSeatChance {
			continue
		}
		if x, y, ok := w.padNear(c, 2.5, false); ok {
			c.X, c.Y = x, y
		}
	}
}

// padNear finds the nearest lily pad within sight of c that no other toad
// sits on; with clear, only one c can reach in a straight line of hops.
func (w *World) padNear(c *Creature, sight float64, clear bool) (x, y float64, ok bool) {
	cx, cy := int(math.Floor(c.X)), int(math.Floor(c.Y))
	n := int(math.Ceil(sight))
	best := math.Inf(1)
	for ty := cy - n; ty <= cy+n; ty++ {
		for tx := cx - n; tx <= cx+n; tx++ {
			if !w.padAt(tx, ty) || (tx == cx && ty == cy) {
				continue
			}
			px, py := float64(tx)+0.5, float64(ty)+0.5
			d := math.Hypot(px-c.X, py-c.Y)
			if d > sight || d >= best || w.toadAt(px, py, c) || (clear && !w.hopLine(c, px, py)) {
				continue
			}
			best, x, y, ok = d, px, py, true
		}
	}
	return x, y, ok
}

// toadAt reports whether a toad other than c sits within half a tile of (x, y).
func (w *World) toadAt(x, y float64, c *Creature) bool {
	for _, o := range w.fauna.creatures {
		if o != c && !o.gone && IsToad(o.Kind) && math.Abs(o.X-x) < 0.5 && math.Abs(o.Y-y) < 0.5 {
			return true
		}
	}
	return false
}

// hopLen is how far the next hop of a toad d tiles from where it is going
// carries it: a full hop, or a little more to land right on the spot.
func hopLen(d float64) float64 {
	if d-ToadHop < toadHopStretch {
		return d
	}
	return ToadHop
}

// hopLine reports whether c can hop in a straight line to (x, y): every
// landing on the way is ground or a leaf. What it flies over does not
// matter.
func (w *World) hopLine(c *Creature, x, y float64) bool {
	px, py := c.X, c.Y
	for range 16 {
		d := math.Hypot(x-px, y-py)
		if d < 1e-6 {
			return true
		}
		l := hopLen(d)
		px, py = px+(x-px)/d*l, py+(y-py)/d*l
		if !w.creatureFits(c.Kind, px, py) {
			return false
		}
	}
	return false
}

// hopTo starts c on hops leaps towards (x, y), the first one now.
func (w *World) hopTo(c *Creature, x, y float64, hops int) {
	c.goalX, c.goalY, c.hops = x, y, hops
	w.leap(c)
}

// leap begins one hop of c towards its goal, if it can land.
func (w *World) leap(c *Creature) {
	dx, dy := c.goalX-c.X, c.goalY-c.Y
	d := math.Hypot(dx, dy)
	if d < 1e-6 || c.hops <= 0 {
		c.hops = 0
		return
	}
	l := hopLen(d)
	if !w.creatureFits(c.Kind, c.X+dx/d*l, c.Y+dy/d*l) {
		c.hops = 0 // water or a wall where it would come down
		return
	}
	c.hops--
	if l == d {
		c.hops = 0
	}
	c.hx, c.hy = dx/d*l/ToadHopTicks, dy/d*l/ToadHopTicks // a hop's speed, not a heading
	if dx != 0 {
		c.FaceLeft = dx < 0
	}
	c.State, c.timer = CreatureWander, ToadHopTicks
}

func (w *World) stepToad(c *Creature) {
	d := c.Kind.def()
	r := w.random()
	if c.biteCD > 0 {
		c.biteCD--
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	if c.Kind == CreatureToad && p != nil && dist <= ToadTouch && c.biteCD == 0 {
		w.toadPoisons(c, p)
	}
	switch c.State {
	case CreatureStrike: // swollen up
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureIdle, d.idle[0]
		}
		return
	case CreatureWander: // in the air: it comes down where it leapt for
		c.X, c.Y = c.X+c.hx, c.Y+c.hy
		if c.timer--; c.timer <= 0 {
			c.State = CreatureIdle
			c.timer = 3 + r.IntN(5) // gathering itself for the next
			if c.hops == 0 {
				c.timer = d.idle[0] + r.IntN(d.idle[1]-d.idle[0]+1)
				if w.onPad(c) {
					c.timer *= toadPadSitTimes
				}
			}
		}
		return
	}
	if c.State != CreatureIdle {
		c.State, c.timer = CreatureIdle, 1
	}
	if c.Kind == CreatureGlowToad && p != nil && dist < ToadShyDist && c.hops == 0 {
		// Away from the player, three hops.
		ax, ay := c.X-p.X, c.Y-p.Y
		if l := math.Hypot(ax, ay); l > 1e-6 {
			gx, gy := c.X+ax/l*3*ToadHop, c.Y+ay/l*3*ToadHop
			if w.hopTo(c, gx, gy, 3); c.State == CreatureWander {
				return
			}
		}
	}
	if c.timer > 0 {
		c.timer--
	}
	if c.timer > 0 {
		return
	}
	if c.hops > 0 { // on with the run
		if w.leap(c); c.State != CreatureWander {
			c.timer = d.idle[0]
		}
		return
	}
	if r.Float64() >= 0.75 {
		c.timer = d.idle[0] + r.IntN(d.idle[1]-d.idle[0]+1)
		return
	}
	if r.Float64() < toadPadChance {
		if x, y, ok := w.padNear(c, toadPadSight, true); ok {
			w.hopTo(c, x, y, 8)
			return
		}
	}
	a := r.Float64() * 2 * math.Pi
	hops := 1 + r.IntN(3)
	gx, gy := c.X+math.Cos(a)*ToadHop*float64(hops), c.Y+math.Sin(a)*ToadHop*float64(hops)
	if hd := math.Hypot(c.homeX-c.X, c.homeY-c.Y); hd > d.leash {
		gx, gy = c.X+(c.homeX-c.X)/hd*ToadHop*float64(hops), c.Y+(c.homeY-c.Y)/hd*ToadHop*float64(hops)
	}
	if !w.hopLine(c, gx, gy) {
		c.timer = 6 + r.IntN(12) // water or a wall that way: think again
		return
	}
	w.hopTo(c, gx, gy, hops)
}

// toadPoisons is the green toad c's skin on p: it swells, and p is poisoned.
func (w *World) toadPoisons(c *Creature, p *Player) {
	c.biteCD = ToadTouchCD
	if c.State != CreatureWander {
		c.State, c.timer = CreatureStrike, ToadPuffTicks
	}
	if p.dev.god {
		return
	}
	struck(p, ToadTouchDamage)
	p.poison = max(p.poison, ToadVenom)
	p.notify(NoticeBitten, uint8(c.Kind))
}
