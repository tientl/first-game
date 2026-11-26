package sim

import "math"

// Dozing. In a big world most beasts are far from every player, where
// nobody sees them: they step only one tick in dozeEvery, so what the world
// costs to run follows what its players are near, not its size. A beast
// within dozeRange of a player (far past what a screen shows) steps every
// tick, as do the bosses and tamed beasts. With no players about, nobody
// dozes.
const (
	dozeRange = 96 // tiles
	dozeEvery = 8
)

// awake is where the players are this tick, for dozes.
func (w *World) awake() [][2]float64 {
	var at [][2]float64
	for _, id := range w.playerOrder {
		p := w.players[id]
		at = append(at, [2]float64{p.X, p.Y})
	}
	return at
}

// dozes reports whether c skips this tick: far from every player in at.
func (w *World) dozes(c *Creature, at [][2]float64) bool {
	if len(at) == 0 || c.Kind.Boss() || c.Tamed() || (w.tick+uint64(c.ID))%dozeEvery == 0 {
		return false
	}
	for _, p := range at {
		if math.Abs(c.X-p[0]) <= dozeRange && math.Abs(c.Y-p[1]) <= dozeRange {
			return false
		}
	}
	return true
}
