package main

import (
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// walker moves the first player across the world at a steady pace, sending
// the client a snapshot every server tick, as -walk asks (to see the grass
// bend as someone goes through it).
type walker struct {
	conn   *fakeConn
	snap   protocol.Snapshot
	vx, vy float32 // tiles a tick
}

// step sends the next snapshot, every third frame (a server tick).
func (w *walker) step(frame int) {
	if frame%(60/sim.TicksPerSecond) != 0 {
		return
	}
	s := w.snap
	s.Players = append([]protocol.PlayerView(nil), s.Players...)
	s.Players[0].X += w.vx
	s.Players[0].Y += w.vy
	s.Tick++
	w.snap = s
	select {
	case w.conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: s}:
	default:
	}
}
