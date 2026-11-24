package client

import (
	"testing"

	"survivalgame/internal/sim"
)

// bagWithWindow is bagForMouse with the bag's window laid out round the slots.
func bagWithWindow() (*Game, *sentConn) {
	g, c := bagForMouse()
	b := &g.bag
	b.panelX, b.panelY = b.gridX-24, b.gridY-40
	b.panelW = float64(bagCols*(slotPx+slotGap)) + 48
	b.panelH = b.hotY + slotPx + 200 - b.panelY
	return g, c
}

func TestDragOutOfTheBagThrowsTheStack(t *testing.T) {
	g, c := bagWithWindow()
	b := &g.bag
	x0, y0 := centre(b, 0)
	g.bagMouse(x0, y0, false, true, false, false)
	g.bagMouse(b.panelX-30, y0, true, false, false, false)
	g.bagMouse(b.panelX-30, y0, false, false, true, false) // let go left of the window
	want := sim.Command{Kind: sim.CmdToss, Slot: 0, Count: tossAll}
	if len(c.cmds) != 1 || c.cmds[0] != want || b.lifting {
		t.Fatalf("sent %v, lifting=%v; want %v", c.cmds, b.lifting, want)
	}
}

func TestLettingGoInsideTheBagOffTheSlotsPutsItBack(t *testing.T) {
	g, c := bagWithWindow()
	b := &g.bag
	x0, y0 := centre(b, 0)
	g.bagMouse(x0, y0, false, true, false, false)
	g.bagMouse(b.panelX+8, y0, true, false, true, false) // in the window's margin
	if len(c.cmds) != 0 || b.lifting {
		t.Fatalf("sent %v, lifting=%v; want nothing sent", c.cmds, b.lifting)
	}
}

func TestRightClickThrowsOneOrPutsBack(t *testing.T) {
	g, c := bagWithWindow()
	b := &g.bag
	x0, y0 := centre(b, 0)
	g.bagRightClick(x0, y0)
	want := sim.Command{Kind: sim.CmdToss, Slot: 0, Count: 1}
	if len(c.cmds) != 1 || c.cmds[0] != want {
		t.Fatalf("right click sent %v, want %v", c.cmds, want)
	}
	// With a stack lifted, the right button only puts it back.
	g.bagMouse(x0, y0, false, true, false, false)
	g.bagRightClick(x0, y0)
	if len(c.cmds) != 1 || b.lifting {
		t.Fatalf("sent %v, lifting=%v; want it put back, nothing thrown", c.cmds, b.lifting)
	}
	// An empty slot throws nothing.
	x3, y3 := centre(b, 3)
	g.bagRightClick(x3, y3)
	if len(c.cmds) != 1 {
		t.Fatalf("sent %v after a right click on an empty slot", c.cmds)
	}
}
