package sim

import "math"

// Throwing things out of the bag. G drops one of what is held, Shift+G (or
// Ctrl+G) the whole stack; in the bag, a stack dragged out of the window
// falls on the ground, and a right click on a slot drops one. What falls
// lies in front of the player, on the spot they face, or at their feet if
// that spot is a wall or water; it is a pile like any other, that
// anyone picks up with Space or by walking over it, saved with the world
// and rotting after a while.
//
// The sea keeps nothing: a swimmer, or someone in a boat, can only throw
// things onto dry land in front of them. Over open water the throw is
// refused with a notice and the items stay in the bag.

// Toss tuning.
const (
	TossReach = 0.9 // a thrown pile lands this far in front of the player
	TossMerge = 0.6 // a pile of the same thing this close is topped up instead of a new one
)

// tossSpot is where what p throws lands: just in front, else at p's feet;
// ok is false when both are water (or otherwise no ground to lie on).
func (w *World) tossSpot(p *Player) (x, y float64, ok bool) {
	dx, dy := p.Dir.Delta()
	if fx, fy := p.X+float64(dx)*TossReach, p.Y+float64(dy)*TossReach; w.walkableAt(fx, fy) {
		return fx, fy, true
	}
	if w.walkableAt(p.X, p.Y) && !w.inWater(p.X, p.Y) {
		return p.X, p.Y, true
	}
	return 0, 0, false
}

// toss throws up to n items from p's slot onto the ground in front of p. n
// is clamped to what the slot holds; an empty or unknown slot does nothing.
func (w *World) toss(p *Player, slot, n int) {
	if p.Dead || p.Asleep || p.chest != 0 || !ValidSlot(slot) || n <= 0 {
		return
	}
	s := p.Inv[slot]
	if s.Empty() {
		return
	}
	n = min(n, s.Count)
	x, y, ok := w.tossSpot(p)
	if !ok {
		p.notify(NoticeCantToss, uint8(s.Item))
		return
	}
	wasHeld := slot == p.Selected && s.Item == ItemTorch
	n = p.Inv.TakeSlot(slot, n)
	if wasHeld && p.Held() != ItemTorch {
		p.torchFuel = 0 // the one burning in the hand went on the ground
	}
	w.pileAt(p, x, y, s.Item, n)
	p.notify(NoticeTossed, uint8(s.Item))
}

// pileAt lays n of k at (x, y), thrown by p: onto a pile of the same thing
// lying there if it has room, the rest as a new pile. p does not walk the
// pile up again until p has stepped off it (Space still picks it up).
func (w *World) pileAt(p *Player, x, y float64, k ItemKind, n int) {
	for _, d := range w.drops {
		if n == 0 {
			return
		}
		if d.gone || d.Item != k || d.Count >= k.MaxCount() || math.Hypot(d.X-x, d.Y-y) > TossMerge {
			continue
		}
		add := min(n, k.MaxCount()-d.Count)
		d.Count += add
		n -= add
		d.FromX, d.FromY, d.Born = p.X, p.Y, w.tick // it falls again, freshly
		d.tosser = p.ID
	}
	for n > 0 {
		c := min(n, k.MaxCount())
		n -= c
		w.nextDrop++
		w.drops = append(w.drops, &Drop{ID: w.nextDrop, Item: k, Count: c, X: x, Y: y, FromX: p.X, FromY: p.Y, Born: w.tick, tosser: p.ID})
	}
}

// settleToss forgets who threw d once they are clear of it: from then on
// it is a pile like any other, and walking back over it picks it up. Until
// then the thrower does not walk it up (stepDrops skips d.tosser).
func (w *World) settleToss(d *Drop) {
	if d.tosser == 0 {
		return
	}
	t := w.players[d.tosser]
	if t == nil || t.Dead || math.Abs(t.X-d.X) > DropTouch || math.Abs(t.Y-d.Y) > DropTouch {
		d.tosser = 0
	}
}
