package sim

import "math"

// Crabs live on the sand by the water and wade into the shallows. One that
// sees a player close by scuttles for the nearest water and goes under,
// coming up again once nobody is about. Cut off from the water, it stands
// its ground with its claws up and pinches anyone who comes too close. So a
// crab is caught by getting between it and the sea, or by finding one that
// strayed up the beach.

// Crab tuning.
const (
	CrabFleeDist      = 2.6 // a crab makes for the water from a player this close
	CrabWaterRun      = 4   // tiles it looks for water within
	CrabHideTicks     = 6 * TicksPerSecond
	CrabPeekDist      = 3.5 // it stays under while a player is this close
	CrabPinchRange    = 0.9
	CrabPinchDamage   = 4.0
	CrabPinchCooldown = 25
	CrabPinchTicks    = 6
)

func (w *World) stepCrab(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.State == CreatureStrike {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	p, dist := w.nearestPlayer(c.X, c.Y)
	if p == nil || dist > CrabFleeDist+1 {
		if c.State == CreatureFlee || c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0 // calm again
		}
		w.wander(c)
		return
	}
	if w.inShallows(c) {
		c.State, c.timer = CreatureHidden, CrabHideTicks // already wading: just go under
		return
	}
	if c.State != CreatureFlee && dist < CrabFleeDist {
		if gx, gy, ok := w.nearestShallows(c.X, c.Y); ok {
			c.State, c.goalX, c.goalY = CreatureFlee, gx, gy
		} else {
			c.State = CreatureChase // cut off: claws up
		}
	}
	switch c.State {
	case CreatureFlee:
		dx, dy := c.goalX-c.X, c.goalY-c.Y
		l := math.Hypot(dx, dy)
		step := min(d.run, l)
		if l < 1e-9 || !w.moveCreature(c, dx/l*step, dy/l*step) {
			c.State = CreatureChase // blocked on the way: stand and fight
		}
		if w.inShallows(c) {
			c.State, c.timer = CreatureHidden, CrabHideTicks
		}
	case CreatureChase:
		if dx := p.X - c.X; dx != 0 {
			c.FaceLeft = dx < 0
		}
		if dist <= CrabPinchRange && c.biteCD == 0 {
			c.State, c.timer, c.biteCD = CreatureStrike, CrabPinchTicks, CrabPinchCooldown
			struck(p, CrabPinchDamage)
			p.notify(NoticeBitten, uint8(c.Kind))
		}
	default:
		w.wander(c)
	}
}

// inShallows reports whether c stands in water.
func (w *World) inShallows(c *Creature) bool {
	return w.isShallow(int(math.Floor(c.X)), int(math.Floor(c.Y)))
}

// nearestShallows is the centre of the nearest shallow water within
// CrabWaterRun tiles of (x, y), if there is any.
func (w *World) nearestShallows(x, y float64) (gx, gy float64, ok bool) {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	best := math.Inf(1)
	for ty := cy - CrabWaterRun; ty <= cy+CrabWaterRun; ty++ {
		for tx := cx - CrabWaterRun; tx <= cx+CrabWaterRun; tx++ {
			if !w.isShallow(tx, ty) {
				continue
			}
			px, py := float64(tx)+0.5, float64(ty)+0.5
			if d := math.Hypot(px-x, py-y); d < best {
				best, gx, gy, ok = d, px, py, true
			}
		}
	}
	return gx, gy, ok
}
