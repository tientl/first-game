package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestHeldSlotComesFromTheServerOnce(t *testing.T) {
	g := readyGame()
	s := protocol.Snapshot{Tick: 1}
	s.Self.Selected = 4
	g.onSnapshot(s)
	if g.hotSel != 4 || g.sentSel != 4 {
		t.Fatalf("held %d sent %d, want the saved slot 4 and nothing to resend", g.hotSel, g.sentSel)
	}
	// Later snapshots lag behind the player's own key presses: they must not
	// pull the selection back.
	g.hotSel = 7
	s.Tick, s.Self.Selected = 2, 4
	g.onSnapshot(s)
	if g.hotSel != 7 {
		t.Fatalf("held %d, want the player's 7 kept", g.hotSel)
	}
}

func TestBagGridCoversEverySlotOnce(t *testing.T) {
	b := bagView{gridX: 100, gridY: 50}
	b.hotY = b.gridY + float64(bagRows*(slotPx+slotGap)) + 20
	seen := make(map[int]bool)
	for i := range sim.NumSlots {
		row, col := slotPos(i)
		if got := slotAtPos(row, col); got != i {
			t.Fatalf("slot %d -> (%d,%d) -> %d", i, row, col, got)
		}
		x, y := b.slotXY(i)
		got, ok := b.slotAt(x+slotPx/2, y+slotPx/2)
		if !ok || got != i || seen[got] {
			t.Fatalf("slot %d drawn at (%.0f,%.0f) is hit as %d,%v", i, x, y, got, ok)
		}
		seen[got] = true
	}
	if row, _ := slotPos(0); row != bagRows {
		t.Fatalf("hotbar is row %d, want the last row %d", row, bagRows)
	}
}
