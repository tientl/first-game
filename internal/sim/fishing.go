package sim

// Fishing takes patience. Action on a fish, with a rod in hand and a worm in
// the bag, casts the line: the float lands on the fish, which stays put
// under it. After a while the fish bites and the float dips; the player must
// strike (press Action again) before it lets go.
//
//   - Striking in time lands the fish and uses up the worm.
//   - Striking before the bite scares the fish off; the worm is kept.
//   - Missing the bite lets the fish steal the worm.
//   - Walking, putting the rod away, sleeping or dying reels the line in.
//
// Only a fresh press counts as a strike, not a held key, so holding Space
// from the cast on catches nothing.

// CastState is where a player's fishing line is.
type CastState uint8

const (
	CastNone    CastState = iota
	CastWaiting           // the float sits still: wait
	CastBite              // the float dips: strike now
)

// Fishing tuning.
const (
	FishBiteMinTicks = 3 * TicksPerSecond / 2 // shortest wait for a bite
	FishBiteMaxTicks = 9 * TicksPerSecond / 2 // longest wait
	FishBiteTicks    = 14                     // how long the fish holds on: the time to strike
	fishFleeTicks    = 2 * TicksPerSecond     // how long a fish swims off after getting away
)

// line is a player's fishing line while it is out.
type line struct {
	state      CastState
	fish       CreatureID
	timer      int // ticks left in state
	bobX, bobY float64
}

// Cast reports where p's line is and where its float sits.
func (p *Player) Cast() (state CastState, bobX, bobY float64) {
	if p.Dead || p.Asleep {
		return CastNone, 0, 0
	}
	return p.line.state, p.line.bobX, p.line.bobY
}

// cast throws p's line onto fish c, which the caller has checked p may
// catch.
func (w *World) cast(p *Player, c *Creature) {
	wait := FishBiteMinTicks + w.random().IntN(FishBiteMaxTicks-FishBiteMinTicks+1)
	p.line = line{state: CastWaiting, fish: c.ID, timer: wait, bobX: c.X, bobY: c.Y}
	c.State, c.timer, c.hookedBy = CreatureHooked, 0, p.ID
}

// stepLine runs one tick of p's line. pressed is a fresh press of Action
// since the last tick, moving whether p is trying to walk.
func (w *World) stepLine(p *Player, pressed, moving bool) {
	c := w.creatureByID(p.line.fish)
	if c == nil || c.State != CreatureHooked || c.hookedBy != p.ID {
		p.line = line{} // the fish is gone
		return
	}
	d := c.Kind.def()
	if moving || (d.tool.Valid() && p.Held() != d.tool) {
		w.unhook(c, p, false) // reeled in quietly
		return
	}
	if pressed {
		p.actedAt, p.actionCD = w.tick, ActionCooldownTicks // the strike
		if p.line.state == CastWaiting {
			w.unhook(c, p, true)
			p.notify(NoticeTooEarly, uint8(c.Kind))
			return
		}
		if p.Inv.Room(d.yield) < d.yieldN {
			w.unhook(c, p, true)
			p.notify(NoticeBagFull, uint8(d.yield))
			return
		}
		p.line = line{}
		if d.bait.Valid() {
			p.Inv.Take(d.bait, 1)
		}
		p.Inv.Add(d.yield, d.yieldN)
		c.gone = true
		p.notify(NoticeCaught, uint8(d.yield))
		return
	}
	if p.line.timer--; p.line.timer > 0 {
		return
	}
	switch p.line.state {
	case CastWaiting:
		p.line.state, p.line.timer = CastBite, FishBiteTicks
	default: // too slow: the fish takes the worm and goes
		if d.bait.Valid() {
			p.Inv.Take(d.bait, 1)
		}
		w.unhook(c, p, true)
		p.notify(NoticeBaitStolen, uint8(d.bait))
	}
}

// unhook lets fish c go from p's line; a scared fish swims off away from p.
func (w *World) unhook(c *Creature, p *Player, scared bool) {
	p.line = line{}
	c.hookedBy = 0
	c.State, c.timer = CreatureIdle, 0
	if scared {
		c.State, c.timer = CreatureWander, fishFleeTicks
		c.hx, c.hy = 0, 0
		c.steerAway(p.X, p.Y, 1)
	}
}

// stillHooked reports whether the player holding c's line is still fishing
// for it; a line is dropped when its player leaves, dies or falls asleep.
func (w *World) stillHooked(c *Creature) bool {
	p, ok := w.players[c.hookedBy]
	return ok && !p.Dead && !p.Asleep && p.line.state != CastNone && p.line.fish == c.ID
}
