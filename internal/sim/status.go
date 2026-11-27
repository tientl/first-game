package sim

import "math"

// What the beasts' moves (skills.go) leave on a player besides their wounds:
//
//   - stunned: a tyrant's roar, a long-neck's stomp, a club slammed down.
//     They can neither move nor act for a moment; then for a moment more a
//     new stun does not take, so stuns never chain.
//   - bleeding: a sabre-tooth's bite, a plate-back's spikes. Health drips
//     away for a while; armour does not stop it.
//   - slowed: a Troodon's venom. They walk at a little over half speed.
//   - pinned: a Dakotaraptor's pounce. Held down and bitten, they cannot
//     move; every press of Action is a struggle, and enough of them throw it
//     off, reeling. It lets go by itself after a while.

// Status tuning.
const (
	StunGrace    = TicksPerSecond      // after a stun, a new one does not take this long
	BleedPerTick = 2.0 / TicksPerSecond // health a bleeding player loses
	SlowShare    = 0.55                 // a slowed player's speed
	PinMash      = 5                    // presses of Action that break a pin
	PinThrow     = 1.5                  // tiles the beast is thrown off
)

// playerStatus is what the beasts' moves leave on a player.
type playerStatus struct {
	stun, grace int
	bleed       int
	slow        int
	pin         CreatureID // the beast holding them down, 0 if none
	mash        int        // presses of Action against it so far
}

func (p *Player) stunFor(n int) {
	if n > 0 && p.status.grace == 0 && !p.dev.god {
		p.status.stun = max(p.status.stun, n)
		p.wake()
	}
}

func (p *Player) bleedFor(n int) {
	if n > 0 && !p.dev.god {
		p.status.bleed = max(p.status.bleed, n)
	}
}

func (p *Player) slowFor(n int) {
	if n > 0 && !p.dev.god {
		p.status.slow = max(p.status.slow, n)
	}
}

// Stunned reports whether p is dazed: no moving, no acting.
func (p *Player) Stunned() bool { return p.status.stun > 0 }

// Bleeding reports whether p is losing blood.
func (p *Player) Bleeding() bool { return p.status.bleed > 0 }

// Slowed reports whether p's legs are heavy with venom.
func (p *Player) Slowed() bool { return p.status.slow > 0 }

// PinnedBy is the beast holding p down, 0 if none.
func (p *Player) PinnedBy() CreatureID { return p.status.pin }

// statusSpeed is the share of their speed p walks at.
func (p *Player) statusSpeed() float64 {
	if p.status.slow > 0 {
		return SlowShare
	}
	return 1
}

// stepStatus runs p's statuses for a tick; pressed is whether Action went
// down since the last. It reports whether p is held: stunned or pinned, so
// they neither move nor act this tick.
func (w *World) stepStatus(p *Player, pressed bool) (held bool) {
	s := &p.status
	switch {
	case s.stun > 0:
		if s.stun--; s.stun == 0 {
			s.grace = StunGrace
		}
	case s.grace > 0:
		s.grace--
	}
	if s.bleed > 0 {
		s.bleed--
		dmg := BleedPerTick
		if p.harm > 0 {
			dmg *= p.harm // the difficulty; see difficulty.go
		}
		hurt(p, dmg)
	}
	if s.slow > 0 {
		s.slow--
	}
	if s.pin != 0 {
		c := w.creatureByID(s.pin)
		if c == nil || c.sk.pin != p.ID || p.Dead {
			s.pin = 0
		} else if pressed {
			if s.mash++; s.mash >= PinMash {
				w.breakFree(p, c)
			}
		}
	}
	return s.stun > 0 || s.pin != 0
}

// breakFree throws off the beast c that holds p down: it reels.
func (w *World) breakFree(p *Player, c *Creature) {
	p.status.pin, p.status.mash = 0, 0
	w.cancelSkill(c)
	c.State, c.timer = CreatureHurt, 2*StaggerTicks
	c.flinchCD = FlinchEvery
	dx, dy := c.X-p.X, c.Y-p.Y
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		dx, dy, l = 1, 0, 1
	}
	face := c.FaceLeft
	for s := 0.0; s < PinThrow; s += 0.1 {
		if !w.moveCreature(c, dx/l*0.1, dy/l*0.1) {
			break
		}
	}
	c.FaceLeft = face
}

// clearStatus takes every status off p: they died or came back.
func (p *Player) clearStatus() { p.status = playerStatus{} }
