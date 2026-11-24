package client

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/sim"
)

// Throwing things out of the bag onto the ground (sim/toss.go). In play, G
// throws one of what is held and Shift+G or Ctrl+G the whole stack. In the
// bag, G does the same for the slot pointed at, a stack dragged out of the
// window falls on the ground, and a right click on a slot throws one (while
// a stack is lifted, a right click puts it back instead). Q was taken: it
// makes an axe.

// tossKey is the key that throws things out of the bag.
const tossKey = ebiten.KeyG

// tossAll is the count sent for a whole stack: the server clamps it to
// what the slot holds.
const tossAll = 255

// tossSlot asks the server to throw one, or the whole stack, of slot i.
func (g *Game) tossSlot(i int, all bool) {
	if !sim.ValidSlot(i) || g.curr.Self.Inv[i].Empty() || g.curr.Self.Dead {
		return
	}
	n := uint8(1)
	if all {
		n = tossAll
	}
	g.command(sim.Command{Kind: sim.CmdToss, Slot: uint8(i), Count: n})
}

// updateToss handles the throw key: what is held in play, what the cursor
// points at in the bag.
func (g *Game) updateToss() {
	if !inpututil.IsKeyJustPressed(tossKey) {
		return
	}
	all := ebiten.IsKeyPressed(ebiten.KeyShift) || ctrlHeld()
	switch g.overlay {
	case overlayNone:
		g.tossSlot(g.hotSel, all)
	case overlayBag:
		g.tossSlot(g.bag.cursor, all)
		g.bag.lifting = false
	}
}

// outsideBag reports whether screen point (x, y) is off the bag's window,
// as last drawn. Before the first draw nothing is outside.
func (b *bagView) outsideBag(x, y float64) bool {
	if b.panelW <= 0 {
		return false
	}
	return x < b.panelX || x >= b.panelX+b.panelW || y < b.panelY || y >= b.panelY+b.panelH
}

// bagRightClick is the right button at (x, y) in the bag: it puts back a
// stack that is lifted, or else throws one of the stack under the pointer.
func (g *Game) bagRightClick(x, y float64) {
	b := &g.bag
	if b.lifting {
		b.lifting = false // put it back
		return
	}
	if i, ok := b.slotAt(x, y); ok {
		g.tossSlot(i, false)
	}
}

// drawTossHint tells, under a stack carried out of the bag's window, that
// letting go throws it on the ground.
func (g *Game) drawTossHint(dst *ebiten.Image, x, y float64) {
	if g.bag.outsideBag(x, y) {
		g.text(dst, "thả ra: vứt xuống đất", x, y+28, colHunger, text.AlignCenter, 1)
	}
}

// tossSound is a soft thump of something set down on the ground.
func tossSound() { snd.play("pick_wood", 0.35, 0, 0.7) }
