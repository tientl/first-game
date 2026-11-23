package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// sentConn records what the client sends.
type sentConn struct{ cmds []sim.Command }

func (c *sentConn) Send(m protocol.ClientMsg) error {
	if m.Kind == protocol.ClientCommand {
		c.cmds = append(c.cmds, m.Command)
	}
	return nil
}
func (c *sentConn) Messages() <-chan protocol.ServerMsg { return nil }
func (c *sentConn) Close() error                        { return nil }

// bagForMouse is a game with the bag open and laid out, wood in hotbar slot 0.
func bagForMouse() (*Game, *sentConn) {
	g := readyGame()
	c := &sentConn{}
	g.conn = c
	g.overlay = overlayBag
	g.bag.open(0, sim.ItemNone)
	g.bag.drawn = true
	g.bag.gridX, g.bag.gridY = 100, 50
	g.bag.hotY = g.bag.gridY + float64(bagRows*(slotPx+slotGap)) + 20
	g.curr.Self.Inv[0] = sim.Slot{Item: sim.ItemWood, Count: 5}
	return g, c
}

func centre(b *bagView, i int) (float64, float64) {
	x, y := b.slotXY(i)
	return x + slotPx/2, y + slotPx/2
}

func TestDragAStackOntoAnotherSlot(t *testing.T) {
	g, c := bagForMouse()
	b := &g.bag
	x0, y0 := centre(b, 0)
	x1, y1 := centre(b, sim.HotbarSlots+2)

	g.bagMouse(x0, y0, true, false, false, false) // hover
	g.bagMouse(x0, y0, false, true, false, false) // press: picked up
	if !b.lifting || b.lifted != 0 || !b.byMouse {
		t.Fatalf("after pressing on the wood: lifting=%v lifted=%d", b.lifting, b.lifted)
	}
	g.bagMouse((x0+x1)/2, (y0+y1)/2, true, false, false, false) // dragging over the grid
	g.bagMouse(x1, y1, true, false, true, false)                // let go on the target
	want := sim.Command{Kind: sim.CmdMoveSlot, Slot: 0, To: sim.HotbarSlots + 2}
	if len(c.cmds) != 1 || c.cmds[0] != want || b.lifting {
		t.Fatalf("sent %v, lifting=%v; want one move %v", c.cmds, b.lifting, want)
	}
}

func TestDropOutsideTheSlotsPutsItBack(t *testing.T) {
	g, c := bagForMouse()
	b := &g.bag
	x0, y0 := centre(b, 0)
	g.bagMouse(x0, y0, false, true, false, false)
	g.bagMouse(x0, y0-300, true, false, true, false) // far above the panel
	if len(c.cmds) != 0 || b.lifting {
		t.Fatalf("sent %v, lifting=%v; want nothing moved", c.cmds, b.lifting)
	}
}

func TestClickThenClickStillMoves(t *testing.T) {
	g, c := bagForMouse()
	b := &g.bag
	x0, y0 := centre(b, 0)
	x1, y1 := centre(b, 5)
	g.bagMouse(x0, y0, false, true, false, false)
	g.bagMouse(x0+1, y0, true, false, true, false) // a click, not a drag: still held
	if !b.lifting || len(c.cmds) != 0 {
		t.Fatalf("after a click: lifting=%v sent %v; want it held", b.lifting, c.cmds)
	}
	g.bagMouse(x1, y1, true, false, false, false)
	g.bagMouse(x1, y1, false, true, false, false) // second click puts it down
	g.bagMouse(x1, y1, false, false, true, false)
	want := sim.Command{Kind: sim.CmdMoveSlot, Slot: 0, To: 5}
	if len(c.cmds) != 1 || c.cmds[0] != want || b.lifting {
		t.Fatalf("sent %v, lifting=%v; want %v", c.cmds, b.lifting, want)
	}
}

func TestPressingAnEmptySlotLiftsNothing(t *testing.T) {
	g, c := bagForMouse()
	b := &g.bag
	x, y := centre(b, 3)
	g.bagMouse(x, y, false, true, false, false)
	g.bagMouse(x+40, y, true, false, true, false)
	if b.lifting || len(c.cmds) != 0 {
		t.Fatalf("lifting=%v sent %v from an empty slot", b.lifting, c.cmds)
	}
}
