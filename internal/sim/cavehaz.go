package sim

import (
	"math"
	"slices"
)

// What lies on the cave floor and falls from its roof (StyleCaveFight):
//
//   - a poison cloud, sprayed by a wounded Arthropleura: poison for whoever
//     stands in it, for a few seconds;
//   - a cloud of itching hair, flicked by the desert's spider queen: slows;
//   - a floor web, laid by a spider queen: it holds whoever walks into it,
//     and a lit torch burns it away;
//   - a collapse: in some chambers the roof is loose. Noise (blows, big
//     moves) makes dust trickle down, and 1.2 s later a part of the roof
//     comes down where the dust fell, on whoever has not moved.

// HazardKind is what a hazard is.
type HazardKind uint8

const (
	HazNone     HazardKind = iota
	HazPoison              // a green cloud
	HazDust                // a cloud of itching hair or dust
	HazWeb                 // a web on the floor
	HazCollapse            // dust trickling down, then the roof
)

// Hazard tuning.
const (
	CollapseWarn     = 24 // ticks of dust trickling before the roof comes down
	CollapseRadius   = 2.0
	CollapseDamage   = 28.0
	collapseNoise    = 100.0 // noise a chamber takes before it lets go
	collapseCooldown = 25 * TicksPerSecond
	noiseDecay       = 6.0 / TicksPerSecond
	cloudPoisonTicks = PoisonLong / 2
	hazWebRefresh    = 12
	collapseChance   = 0.35 // of the big chambers
	collapseMinR     = 5.0
)

// Hazard is one of them, for the client to draw.
type Hazard struct {
	ID          uint32
	Kind        HazardKind
	X, Y, R     float64
	Left, Total int // ticks left; for a collapse, till the roof falls
	forever     bool
}

// collapseSpot is a chamber with a loose roof.
type collapseSpot struct {
	x, y, r float64
	noise   float64
	cool    int
}

// caveFight is the world's cave hazards.
type caveFight struct {
	haz   []*Hazard
	next  uint32
	spots []collapseSpot
}

// addHazard lays a hazard of kind k centred on (x, y).
func (w *World) addHazard(k HazardKind, x, y, r float64, life int) *Hazard {
	f := &w.cf
	f.next++
	h := &Hazard{ID: f.next, Kind: k, X: x, Y: y, R: r, Left: life, Total: life, forever: life < 0}
	f.haz = append(f.haz, h)
	return h
}

// Hazards lists the hazards in the rectangle [x0, x1] x [y0, y1].
func (w *World) Hazards(x0, y0, x1, y1 float64) []Hazard {
	var out []Hazard
	for _, h := range w.cf.haz {
		if h.X >= x0-h.R && h.X <= x1+h.R && h.Y >= y0-h.R && h.Y <= y1+h.R {
			out = append(out, *h)
		}
	}
	return out
}

// caveNoise is a noise of loudness amount at (x, y): the loose roofs near it
// take it in.
func (w *World) caveNoise(x, y, amount float64) {
	for i := range w.cf.spots {
		s := &w.cf.spots[i]
		if math.Hypot(s.x-x, s.y-y) <= s.r+1 {
			s.noise += amount
		}
	}
}

func (w *World) stepCaveFight() {
	f := &w.cf
	for i := range f.spots {
		s := &f.spots[i]
		if s.cool > 0 {
			s.cool--
		}
		s.noise = max(0, s.noise-noiseDecay)
		if s.noise >= collapseNoise && s.cool == 0 {
			// the dust falls on whoever is in the chamber, where they stand
			for _, id := range w.playerOrder {
				p := w.players[id]
				if !p.Dead && math.Hypot(p.X-s.x, p.Y-s.y) <= s.r {
					w.addHazard(HazCollapse, p.X, p.Y, CollapseRadius, CollapseWarn)
					break
				}
			}
			s.noise, s.cool = 0, collapseCooldown
		}
	}
	if len(f.haz) == 0 {
		return
	}
	for _, h := range f.haz {
		if !h.forever {
			h.Left--
		}
		switch h.Kind {
		case HazPoison, HazDust:
			w.cloudEffects(h)
		case HazWeb:
			w.webEffects(h)
		case HazCollapse:
			if h.Left <= 0 {
				w.roofFalls(h)
			}
		}
	}
	f.haz = slices.DeleteFunc(f.haz, func(h *Hazard) bool { return !h.forever && h.Left <= 0 })
}

func (w *World) cloudEffects(h *Hazard) {
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || math.Hypot(p.X-h.X, p.Y-h.Y) > h.R+PlayerRadius {
			continue
		}
		if h.Kind == HazPoison {
			p.poison = max(p.poison, cloudPoisonTicks)
		} else {
			p.slowFor(TicksPerSecond)
		}
	}
}

func (w *World) webEffects(h *Hazard) {
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || p.boat != 0 {
			continue
		}
		d := math.Hypot(p.X-h.X, p.Y-h.Y)
		if p.torchFuel > 0 && d <= h.R+0.8 {
			h.Left, h.forever = 0, false // a lit torch burns it away
			return
		}
		if d <= h.R {
			p.webbed = max(p.webbed, hazWebRefresh)
		}
	}
}

// roofFalls is a collapse coming down: whoever is under it is hurt, and the
// dust hangs about.
func (w *World) roofFalls(h *Hazard) {
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || math.Hypot(p.X-h.X, p.Y-h.Y) > h.R+PlayerRadius {
			continue
		}
		struck(p, CollapseDamage)
		p.stunFor(8)
		p.notify(NoticeCollapse, 0)
	}
	for _, c := range w.fauna.creatures {
		if !c.gone && !c.Kind.IsCaveBoss() && math.Hypot(c.X-h.X, c.Y-h.Y) <= h.R {
			c.HP -= int(CollapseDamage)
			if c.HP <= 0 {
				w.fell(c)
			}
		}
	}
	w.addHazard(HazDust, h.X, h.Y, h.R, 3*TicksPerSecond)
}
