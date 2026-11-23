package sim

import "math"

// Each player remembers which tiles they have seen, for the map: one bit
// per tile, row-major. Being within RevealRadius of a tile's centre counts
// as seeing it. The server keeps the set (it is saved with the player) and
// sends it in Welcome; after that the client reveals with the same rule
// itself, so the set never has to travel again.

// RevealRadius is how far around the player, in tiles, counts as seen.
const RevealRadius = 10

// Explored is the set of tiles one player has seen.
type Explored struct {
	w, h int
	bits []byte

	lastTX, lastTY int // the tile last revealed around; -1 before the first
}

// NewExplored returns an empty set for a w by h map.
func NewExplored(w, h int) *Explored {
	return &Explored{w: max(0, w), h: max(0, h), bits: make([]byte, (max(0, w)*max(0, h)+7)/8), lastTX: -1, lastTY: -1}
}

// ExploredFrom rebuilds a set from Bytes. Data of the wrong size (another
// map, or garbage) gives an empty set rather than a wrong one.
func ExploredFrom(w, h int, b []byte) *Explored {
	e := NewExplored(w, h)
	if len(b) == len(e.bits) {
		copy(e.bits, b)
		if pad := len(e.bits)*8 - e.w*e.h; pad > 0 {
			e.bits[len(e.bits)-1] &= byte(0xff >> pad) // no bits past the map
		}
	}
	return e
}

// Size returns the map size the set covers, in tiles.
func (e *Explored) Size() (w, h int) {
	if e == nil {
		return 0, 0
	}
	return e.w, e.h
}

// Bytes returns a copy of the set, for saving and sending.
func (e *Explored) Bytes() []byte {
	if e == nil {
		return nil
	}
	return append([]byte(nil), e.bits...)
}

// Has reports whether tile (tx, ty) has been seen. Off the map is never seen.
func (e *Explored) Has(tx, ty int) bool {
	if e == nil || tx < 0 || ty < 0 || tx >= e.w || ty >= e.h {
		return false
	}
	i := ty*e.w + tx
	return e.bits[i/8]&(1<<(i%8)) != 0
}

// Count returns how many tiles have been seen.
func (e *Explored) Count() int {
	n := 0
	for _, b := range e.bits {
		for ; b != 0; b &= b - 1 {
			n++
		}
	}
	return n
}

// Explored returns a copy of the tiles player id has seen, as
// Explored.Bytes, or nil for an unknown player.
func (w *World) Explored(id PlayerID) []byte {
	if p, ok := w.players[id]; ok {
		return p.explored.Bytes()
	}
	return nil
}

// Reveal marks the tiles within RevealRadius of (x, y), in tiles, as seen
// and reports whether any was new. Standing on the same tile as last time
// does nothing, so calling it every tick is cheap.
func (e *Explored) Reveal(x, y float64) bool {
	if e == nil || !finite(x, y) {
		return false
	}
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	if cx == e.lastTX && cy == e.lastTY {
		return false
	}
	e.lastTX, e.lastTY = cx, cy
	changed := false
	const r = RevealRadius
	for ty := max(0, cy-r); ty <= min(e.h-1, cy+r); ty++ {
		for tx := max(0, cx-r); tx <= min(e.w-1, cx+r); tx++ {
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			if dx*dx+dy*dy > r*r {
				continue
			}
			i := ty*e.w + tx
			if m := byte(1 << (i % 8)); e.bits[i/8]&m == 0 {
				e.bits[i/8] |= m
				changed = true
			}
		}
	}
	return changed
}
