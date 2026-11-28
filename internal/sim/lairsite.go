package sim

import "math"

// The lairs of worlds made since StyleLairs (see temple.go and atoll.go),
// and what both fights leave on the ground for a while (hazards).
//
// Each of the demo's two bosses lives at a place built for its fight: an
// open arena 25 to 35 tiles across with some cover in it, no canopy over
// it, a hard way in, a landmark seen from far off and a safe camp 30 to 40
// tiles out, with room for a campfire and a leaf tent. An older world keeps
// the lair it was made with (lair.go), and its fight.

// LairLayout is how a lair is laid out.
type LairLayout uint8

const (
	LayoutHollow LairLayout = iota // the worlds before StyleLairs: Titanoboa's hollow, the shark's open sea
	LayoutTemple                   // Titanoboa's ruined temple round a black pool: see temple.go
	LayoutAtoll                    // the Megalodon's broken atoll round a deep lagoon: see atoll.go
)

// More LairProp kinds (the Var of an EntityLairProp), since StyleLairs.
const (
	LairPillar    = NumLairProps + iota // a column of the temple still standing, broken off: blocks the way
	LairColumnL                         // a fallen column lying across three tiles, left to right: blocks the way...
	LairColumnM                         // ...its middle, where it is drawn...
	LairColumnR                         // ...and its right end
	LairDeadTree                        // the giant dead tree over the temple, on the middle of its foot: blocks the way
	LairTreeRoot                        // the rest of the dead tree's foot: blocks the way, drawn with the tree
	LairSerpent                         // a stone serpent's head at the pool's edge, where the water pours
	LairCampRing                        // the stones of an old fire ring at a camp
	LairCampLog                         // a log to sit on at a camp
	LairWreck                           // the wreck's hull lying on the reef, on the middle of it (drawn there; walked on)
	LairWreckPart                       // the rest of the hull's deck: walked on, drawn with the wreck
	LairMast                            // the wreck's mast, fallen from it: the landmark (drawn standing)
	LairPalm                            // a lone palm on a sandbar or the camp's cay
	numLairPropsV2
)

// lairSolid reports whether lair prop v stands in the way of walkers.
func lairSolid(v uint8) bool {
	switch v {
	case LairPillar, LairColumnL, LairColumnM, LairColumnR, LairDeadTree, LairTreeRoot, LairMast, LairPalm:
		return true
	}
	return false
}

// LairSolid reports whether lair prop v blocks the way: for the client,
// which treats it as a wall.
func LairSolid(v uint8) bool { return lairSolid(v) }

// lairCover reports whether lair prop v hides who is behind it from a sweep
// of the tail or the coils: the temple's stones.
func lairCover(v uint8) bool {
	switch v {
	case LairPillar, LairColumnL, LairColumnM, LairColumnR, LairDeadTree, LairTreeRoot:
		return true
	}
	return false
}

// solidProp reports whether e is a lair prop that blocks the way.
func solidProp(e *Entity) bool { return e != nil && e.Kind == EntityLairProp && lairSolid(e.Var) }

// lairSite is a lair's layout since StyleLairs: kept through the boss's
// deaths and returns (raiseBoss keeps it).
type lairSite struct {
	layout       LairLayout
	r            float64 // the arena's outer radius: the courtyard's longer half width, the atoll's to past its reef
	rx, ry       float64 // the arena's half widths: the courtyard is a rounded oblong (see ArenaQ), the atoll round
	poolX, poolY float64 // the temple's black pool, and its half widths
	poolRX       float64
	poolRY       float64
	lagoon       float64 // the atoll's lagoon's radius, open deep water inside it
	camp         bool    // a camp was found for it...
	campX, campY float64 // ...here
	markX, markY float64 // the landmark: the dead tree's foot, the mast
	reef         []int32 // the atoll's footholds of reef, which the shark can break
	reefAt       map[int32]bool
}

// coverBetween reports whether a stone of the temple stands between (x0,
// y0) and (x1, y1): a sweep of the tail or the coils from one does not
// reach the other.
func (w *World) coverBetween(x0, y0, x1, y1 float64) bool {
	d := math.Hypot(x1-x0, y1-y0)
	n := int(d / 0.2)
	lastX, lastY := math.MinInt, math.MinInt
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		tx, ty := int(math.Floor(x0+(x1-x0)*t)), int(math.Floor(y0+(y1-y0)*t))
		if tx == lastX && ty == lastY {
			continue
		}
		lastX, lastY = tx, ty
		if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityLairProp && lairCover(e.Var) {
			return true
		}
	}
	return false
}

// lootSpot is where what a boss that fell at (x, y) leaves comes to lie:
// there if a player can stand there, else the nearest ground within reach
// (the reef by the lagoon it died in, the edge of its pool).
func (w *World) lootSpot(x, y float64) (float64, float64) {
	if w.standable(x, y) && !w.inWater(x, y) {
		return x, y
	}
	if tx, ty, ok := w.nearestGround(x, y, 20); ok {
		return float64(tx) + 0.5, float64(ty) + 0.5
	}
	return x, y
}

// nearestGround is the walkable tile nearest (x, y) within r tiles, in
// rings out from it; ties go to the first found, so it draws no random
// numbers.
func (w *World) nearestGround(x, y float64, r int) (int, int, bool) {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	best, bx, by := math.Inf(1), 0, 0
	for ring := 0; ring <= r; ring++ {
		for ty := cy - ring; ty <= cy+ring; ty++ {
			for tx := cx - ring; tx <= cx+ring; tx++ {
				if max(abs(tx-cx), abs(ty-cy)) != ring {
					continue
				}
				fx, fy := float64(tx)+0.5, float64(ty)+0.5
				if !w.standable(fx, fy) || w.inWater(fx, fy) {
					continue
				}
				if d := math.Hypot(fx-x, fy-y); d < best {
					best, bx, by = d, tx, ty
				}
			}
		}
		if !math.IsInf(best, 1) && float64(ring) > best+1 {
			break // nothing further out can be nearer
		}
	}
	return bx, by, !math.IsInf(best, 1)
}

// --- hazards ------------------------------------------------------------------

// HazardKind is what a fight leaves on the ground for a while.
type HazardKind uint8

const (
	HazardNone   HazardKind = iota
	HazardVenom             // a puddle of Titanoboa's venom: whoever stands in it is slowed
	HazardFlood             // the temple's courtyard under shallow water (its last stage): wading
	HazardReef              // a foothold of the atoll's reef broken by the shark: water till it settles
	HazardSplash            // where the shark came down from its breach: the water still heaving
	HazardBlood             // the shark's blood in the lagoon (its frenzy): only to see
)

// Hazard is one of them.
type Hazard struct {
	Kind       HazardKind
	X, Y, R    float64 // its middle and radius, in tiles; a reef's is its tile's middle
	Born, Ends uint64  // ticks
}

// Hazard tuning.
const (
	VenomTicks     = 9 * TicksPerSecond // a puddle of venom lies this long
	VenomSlowTicks = TicksPerSecond / 2 // and slows whoever steps in it this long after
	WadeShare      = 0.72               // a player wading the flooded courtyard walks this share of their speed
	FloodRise      = 4 * TicksPerSecond // the courtyard floods over this long
	ReefDownTicks  = 12 * TicksPerSecond
	ReefSettle     = 3 * TicksPerSecond // the last of ReefDownTicks: the reef rises back, already walkable at the end
	SplashTicks    = 2 * TicksPerSecond
)

// hazards is what the fights have left on the ground.
type hazards struct {
	list []Hazard
}

func (w *World) addHazard(h Hazard) {
	h.Born = w.tick
	w.hz.list = append(w.hz.list, h)
}

// Hazards lists what lies on the ground in the rectangle [x0, x1] x [y0, y1].
func (w *World) Hazards(x0, y0, x1, y1 float64) []Hazard {
	var out []Hazard
	for _, h := range w.hz.list {
		if h.X+h.R >= x0 && h.X-h.R <= x1 && h.Y+h.R >= y0 && h.Y-h.R <= y1 {
			out = append(out, h)
		}
	}
	return out
}

// stepHazards ends what has run its time and works what lies on players.
func (w *World) stepHazards() {
	if len(w.hz.list) == 0 {
		return
	}
	keep := w.hz.list[:0]
	for _, h := range w.hz.list {
		if w.tick >= h.Ends {
			if h.Kind == HazardReef {
				w.settleReef(h) // see atoll.go
			}
			continue
		}
		keep = append(keep, h)
	}
	clear(w.hz.list[len(keep):])
	w.hz.list = keep
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || p.boat != 0 {
			continue
		}
		for _, h := range w.hz.list {
			switch h.Kind {
			case HazardVenom:
				if math.Hypot(p.X-h.X, (p.Y-h.Y)*1.15) <= h.R && !w.inWater(p.X, p.Y) {
					p.slowFor(VenomSlowTicks)
				}
			case HazardFlood:
				risen := min(1, float64(w.tick-h.Born)/FloodRise)
				s := &w.bosses.lairs[0].site // the temple's courtyard, rising from the pool
				if s.layout == LayoutTemple && s.q(p.X-h.X, p.Y-h.Y) <= risen {
					p.status.wade = 2
				}
			}
		}
	}
}

// clearHazards takes away everything a lair's fight left within r of (x,
// y): the fight is over.
func (w *World) clearHazards(x, y, r float64) {
	keep := w.hz.list[:0]
	for _, h := range w.hz.list {
		if math.Hypot(h.X-x, h.Y-y) <= r {
			if h.Kind == HazardReef {
				w.settleReef(h)
			}
			continue
		}
		keep = append(keep, h)
	}
	clear(w.hz.list[len(keep):])
	w.hz.list = keep
}

// flooded reports whether a flood lies over the temple of lair l.
func (w *World) flooded(l *bossLair) bool {
	for _, h := range w.hz.list {
		if h.Kind == HazardFlood && math.Hypot(h.X-l.x, h.Y-l.y) < 1 {
			return true
		}
	}
	return false
}

// ArenaQ is how far out (dx, dy) lies in the arena of a lair of layout lo
// and half widths rx, ry, from its middle: 1 at its edge. The temple's
// courtyard is a rounded oblong (a superellipse of power four), wider than
// it is deep, so that the whole of a screen fits in it; the atoll and the
// older lairs are round.
func ArenaQ(lo LairLayout, rx, ry, dx, dy float64) float64 {
	if lo == LayoutTemple {
		a, b := dx/rx, dy/ry
		return math.Sqrt(math.Sqrt(a*a*a*a + b*b*b*b))
	}
	return math.Hypot(dx/rx, dy/ry)
}

// q is ArenaQ for site s.
func (s *lairSite) q(dx, dy float64) float64 { return ArenaQ(s.layout, s.rx, s.ry, dx, dy) }

// edge is how far the arena's edge lies from its middle along angle a.
func (s *lairSite) edge(a float64) float64 {
	return 1 / s.q(math.Cos(a), math.Sin(a))
}

// ArenaTileShape reports whether tile (tx, ty) is part of the ring round
// the arena of a lair at (lx, ly) of layout lo and half widths rx, ry: where
// its roots rise. The client draws them on the same tiles.
func ArenaTileShape(lo LairLayout, lx, ly, rx, ry float64, tx, ty int) bool {
	q := ArenaQ(lo, rx, ry, float64(tx)+0.5-lx, float64(ty)+0.5-ly)
	return math.Abs(q-1)*min(rx, ry) < ArenaHalfWidth
}
