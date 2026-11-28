package sim

import "math"

// Throwing spears. A spear in hand is aimed (AimedItem): a click is a
// thrust as ever, but holding the button down draws the spear back
// (Input.Draw), the player turning to the pointer, and letting go throws it
// (CmdThrow) at where the pointer is, up to ThrowRange tiles. It flies
// straight and fast and strikes the first beast in its way, harder than a
// thrust (ThrowShare times the spear's blow): from the reef at the shark in
// its lagoon, at a beast across a river. Then it falls: where it struck or
// ran out, if that is ground, else it washes up on the nearest ground (the
// reef by the lagoon), to be picked up again. A spear thrown where no ground
// is near is lost to the sea.

// Throw tuning.
const (
	ThrowRange = 9.0                    // tiles a spear flies at most
	ThrowSpeed = 0.9                    // tiles a tick
	ThrowShare = 3.0                    // a throw's blow, in thrusts of the same spear
	ThrowCD    = 4 * TicksPerSecond / 5 // ticks before the next throw or blow
	throwWash  = 18                     // a spear in the water washes up on ground this near, at most
)

// SpearID identifies a spear in the air.
type SpearID uint32

// Spear is a spear in the air.
type Spear struct {
	ID         SpearID
	Item       ItemKind
	X, Y       float64 // where it is
	VX, VY     float64 // tiles a tick
	Left       float64 // tiles it has still to fly
	By         PlayerID
	gone       bool
	FromX, FrY float64 // where it was thrown from
}

// Throwable reports whether k is thrown: the spears.
func Throwable(k ItemKind) bool {
	return k == ItemSpear || k == ItemIvorySpear || k == ItemSpinoSpear
}

func init() {
	for _, k := range []ItemKind{ItemSpear, ItemIvorySpear, ItemSpinoSpear} {
		aimedItems[k] = true
	}
}

// throwSpear throws the spear in p's hand at where they point (or the way
// they face).
func (w *World) throwSpear(p *Player) {
	k := p.Held()
	if p.Dead || p.Asleep || !Throwable(k) || p.actionCD > 0 || p.mount != 0 || w.Swimming(p) ||
		p.Stunned() || p.PinnedBy() != 0 || p.guard.up {
		return
	}
	dx, dy := p.Dir.Delta()
	ux, uy := float64(dx), float64(dy)
	if ax, ay, ok := p.Aim(); ok {
		if l := math.Hypot(ax-p.X, ay-p.Y); l > 0.3 {
			ux, uy = (ax-p.X)/l, (ay-p.Y)/l
		}
	}
	if l := math.Hypot(ux, uy); l > 1e-9 {
		ux, uy = ux/l, uy/l
	} else {
		ux, uy = 1, 0
	}
	if p.Inv.TakeSlot(p.Selected, 1) == 0 {
		return
	}
	p.Dir = dirToward(ux, uy)
	p.actionCD = ThrowCD
	p.actedAt = w.tick
	w.nextSpear++
	w.spears = append(w.spears, &Spear{ID: w.nextSpear, Item: k, X: p.X + ux*0.4, Y: p.Y + uy*0.4,
		VX: ux * ThrowSpeed, VY: uy * ThrowSpeed, Left: ThrowRange, By: p.ID, FromX: p.X, FrY: p.Y})
	p.did(DeedThrow, k)
}

// stepSpears flies every spear in the air a tick.
func (w *World) stepSpears() {
	if len(w.spears) == 0 {
		return
	}
	for _, s := range w.spears {
		w.flySpear(s)
	}
	keep := w.spears[:0]
	for _, s := range w.spears {
		if !s.gone {
			keep = append(keep, s)
		}
	}
	clear(w.spears[len(keep):])
	w.spears = keep
}

// flySpear moves spear s on, in short steps, striking the first beast in
// its way.
func (w *World) flySpear(s *Spear) {
	const sub = 0.3
	step := min(ThrowSpeed, s.Left)
	for moved := 0.0; moved < step && !s.gone; moved += sub {
		d := min(sub, step-moved)
		nx, ny := s.X+s.VX/ThrowSpeed*d, s.Y+s.VY/ThrowSpeed*d
		if w.blocked(int(math.Floor(nx)), int(math.Floor(ny))) {
			w.spearFalls(s) // into a wall, a column, a tree's foot
			return
		}
		s.X, s.Y = nx, ny
		if c := w.spearHits(s); c != nil {
			w.spearStrikes(s, c)
			w.spearFalls(s)
			return
		}
	}
	if s.Left -= step; s.Left <= 0 {
		w.spearFalls(s)
	}
}

// spearHits is the beast spear s is in, if any: the nearest.
func (w *World) spearHits(s *Spear) *Creature {
	var best *Creature
	bestD := math.Inf(1)
	for _, c := range w.fauna.creatures {
		if c.gone || c.Tamed() || c.rider != 0 {
			continue
		}
		if c.State.OutOfReach() && !(c.Kind == CreatureMegalodon && c.State == CreatureLurk) {
			continue // under the water, up in the air: but the shark's fin is a mark
		}
		r := c.Kind.def().radius + 0.25
		if dx, dy := c.X-s.X, c.Y-s.Y; math.Abs(dx) <= r && math.Abs(dy) <= r {
			if d := math.Hypot(dx, dy); d <= r && d < bestD {
				best, bestD = c, d
			}
		}
	}
	return best
}

// spearStrikes is spear s striking c.
func (w *World) spearStrikes(s *Spear, c *Creature) {
	p := w.players[s.By]
	r := w.random()
	dmg := float64(WeaponDamage(s.Item)) * ThrowShare * (1 + (r.Float64()*2-1)*DamageSpread)
	if p != nil {
		dmg *= p.buffDamage()
	}
	if r.Float64() < CritChance {
		dmg *= CritBonus
		if p != nil {
			p.notify(NoticeCrit, 0)
		}
	}
	w.wound(c, max(1, int(dmg+0.5)), p)
	if p != nil {
		p.notify(NoticeSpearHit, uint8(c.Kind))
	}
}

// spearFalls brings spear s down: on the ground where it is, else washed up
// on the nearest ground; lost if there is none.
func (w *World) spearFalls(s *Spear) {
	s.gone = true
	x, y := s.X, s.Y
	if !w.walkableAt(x, y) || w.inWater(x, y) {
		tx, ty, ok := w.nearestGround(x, y, throwWash)
		if !ok {
			return // the sea keeps it
		}
		x, y = float64(tx)+0.5, float64(ty)+0.5
	}
	w.nextDrop++
	w.drops = append(w.drops, &Drop{ID: w.nextDrop, Item: s.Item, Count: 1, X: x, Y: y, FromX: s.X, FromY: s.Y, Born: w.tick})
}

// Spears lists the spears in the air in the rectangle [x0, x1] x [y0, y1].
func (w *World) Spears(x0, y0, x1, y1 float64) []Spear {
	var out []Spear
	for _, s := range w.spears {
		if !s.gone && s.X >= x0 && s.X <= x1 && s.Y >= y0 && s.Y <= y1 {
			out = append(out, *s)
		}
	}
	return out
}
