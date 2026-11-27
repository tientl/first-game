package client

import (
	"math"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestUIScaleFollowsTheScreen(t *testing.T) {
	defer SetUIScale(0)
	SetUIScale(0)
	for _, c := range []struct {
		sh   int
		want float64
	}{{240, 1}, {720, 1.25}, {768, 1.25}, {900, 1.5}, {1050, 1.75}, {1080, 2}, {1200, 2}, {1440, 2.5}, {1600, 2.75}, {1620, 3}, {2160, 4}} {
		if got := uiScaleFor(c.sh); got != c.want {
			t.Errorf("auto scale at %dp is %g, want %g", c.sh, got, c.want)
		}
		// The interface is always laid out at least 540 UI pixels tall
		// (but on screens shorter than that), and never much taller.
		if _, uh := uiSize(16*c.sh/9, c.sh, uiScaleFor(c.sh)); c.sh >= 540 && (uh < 540 || uh > 620) {
			t.Errorf("at %dp the interface is %d UI pixels tall", c.sh, uh)
		}
	}
	SetUIScale(2)
	if uiScaleFor(1080) != 2 || uiScaleFor(720) != 1.5 || uiScaleFor(2160) != 2 {
		t.Errorf("chosen scale 2: %g at 1080p, %g at 720p, %g at 4K", uiScaleFor(1080), uiScaleFor(720), uiScaleFor(2160))
	}
	SetUIScale(2.5) // 250%: too big for 1080p, where it stops at 2.25
	if uiScaleFor(1440) != 2.5 || uiScaleFor(1080) != 2.25 {
		t.Errorf("chosen scale 2.5: %g at 1440p, %g at 1080p", uiScaleFor(1440), uiScaleFor(1080))
	}
}

func TestUIPoint(t *testing.T) {
	for _, c := range []struct {
		mx, my int
		s      float64
		x, y   int
	}{
		{0, 0, 2, 0, 0}, {1, 1, 2, 0, 0}, {2, 3, 2, 1, 1}, {1919, 1079, 2, 959, 539}, {-1, 5, 2, -1, 2}, {7, 9, 1, 7, 9},
		// At 2.5 a UI pixel is two and a half screen pixels: each screen
		// pixel goes to the UI pixel its centre is in.
		{0, 0, 2.5, 0, 0}, {1, 2, 2.5, 0, 1}, {4, 5, 2.5, 1, 2}, {2559, 1439, 2.5, 1023, 575},
		{3, 3, 1.25, 2, 2}, {1279, 719, 1.25, 1023, 575},
	} {
		if x, y := uiPoint(c.mx, c.my, c.s); x != c.x || y != c.y {
			t.Errorf("uiPoint(%d, %d, %g) = %d, %d; want %d, %d", c.mx, c.my, c.s, x, y, c.x, c.y)
		}
	}
}

// At 1440p the interface is drawn at UI scale 2.5 on a 1024×576 canvas:
// every screen pixel of a hotbar slot, edges and all, picks that slot, and
// no pixel of the gap between two slots picks either. The slots and the
// gaps land on fractions of screen pixels, so this is where rounding goes
// wrong if it does.
func TestClicksLandOnTheSlotsAtAFractionalScale(t *testing.T) {
	defer SetUIScale(0)
	SetUIScale(0)
	const sw, sh = 2560, 1440
	k := uiScaleFor(sh)
	if k != 2.5 {
		t.Fatalf("1440p is at UI scale %g, want 2.5", k)
	}
	uw, uh := uiSize(sw, sh, k)
	if uw != 1024 || uh != 576 {
		t.Fatalf("the UI canvas at 1440p is %dx%d, want 1024x576", uw, uh)
	}
	x0, y0 := hotbarX0(uw), hotbarY(uh)
	for i := range sim.HotbarSlots {
		left := float64(x0+i*(hotPx+hotGap)) * k // the slot's left edge on the screen
		right := left + float64(hotPx)*k
		top := float64(y0) * k
		for mx := int(math.Floor(left)) - 1; mx <= int(math.Ceil(right)); mx++ {
			cx := float64(mx) + 0.5 // the pixel's centre
			ux, uy := uiPoint(mx, int(top)+3, k)
			got, ok := hotbarSlotAt(ux, uy, uw, uh)
			switch {
			case cx > left+0.01 && cx < right-0.01 && (!ok || got != i):
				t.Fatalf("screen x %d (slot %d spans %.1f to %.1f) hits slot %d %v", mx, i, left, right, got, ok)
			case (cx < left-0.01 || cx > right+0.01) && ok && got == i:
				t.Fatalf("screen x %d, outside slot %d (%.1f to %.1f), hits it", mx, i, left, right)
			}
		}
	}
}

// At 1080p the interface is drawn at UI scale 2 on a 960×540 canvas: a
// click anywhere on a slot's 2×2 screen pixels picks that slot, on the
// hotbar and in the chest window alike.
func TestClicksLandOnTheSlotsAtUIScale2(t *testing.T) {
	const sw, sh, s = 1920, 1080, 2
	uw, uh := sw/s, sh/s
	x0, y0 := hotbarX0(uw), hotbarY(uh)
	for i := range sim.HotbarSlots {
		for _, d := range []int{0, 1, 2*hotPx - 1} { // both edges of the slot on screen
			mx, my := s*(x0+i*(hotPx+hotGap))+d, s*y0+d
			if got, ok := hotbarSlotAt(mx/s, my/s, uw, uh); !ok || got != i {
				t.Fatalf("screen (%d, %d) hits hotbar slot %d %v, want %d", mx, my, got, ok, i)
			}
		}
	}

	g, c := chestGame(sim.EntityChest)
	g.chest.layout(uw, uh)
	g.chest.drawn = true
	cx, cy := g.chest.chestSlotXY(1)
	ux, uy := uiPoint(int(cx*s)+2*chestPx-1, int(cy*s)+2*chestPx-1, s) // the slot's last screen pixel
	g.chestMouse(float64(ux), float64(uy), true)
	if want := (sim.Command{Kind: sim.CmdChestTake, Slot: 1}); len(c.cmds) != 1 || c.cmds[0] != want {
		t.Fatalf("a click on chest slot 1 sent %v, want %+v", c.cmds, want)
	}
	ix, iy := g.chest.invSlotXY(0) // the hotbar's first slot, holding wood
	ux, uy = uiPoint(int(ix*s)+1, int(iy*s)+1, s)
	g.chestMouse(float64(ux), float64(uy), true)
	if want := (sim.Command{Kind: sim.CmdChestPut, Slot: 0}); len(c.cmds) != 2 || c.cmds[1] != want {
		t.Fatalf("a click on my first slot sent %v, want %+v", c.cmds, want)
	}
	if _, _, _, ph := g.chest.layout(uw, uh); ph > float64(uh) {
		t.Errorf("the chest window is %.0f tall, more than the %d of a Full HD canvas", ph, uh)
	}
}

// chestGame is a game whose snapshot says chest 9, of kind k, is open with
// berries in its slot 1; wood is in my hotbar slot 0.
func chestGame(k sim.EntityKind) (*Game, *sentConn) {
	g := readyGame()
	c := &sentConn{}
	g.conn = c
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight}
	self := protocol.SelfState{Health: 50, Hunger: 50, Chest: 9}
	self.Inv[0] = sim.Slot{Item: sim.ItemWood, Count: 5}
	self.ChestSlots[1] = sim.Slot{Item: sim.ItemBerry, Count: 4}
	chest := protocol.EntityView{ID: 9, Kind: k, TX: 11, TY: 10}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Self: self, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{chest}})
	return g, c
}

// An old chest in a cave opens the chest window with its hoard, which the
// player takes from like any chest's; it cannot be taken down.
func TestAnOldChestOpensTheChestWindow(t *testing.T) {
	g, c := chestGame(sim.EntityTreasure)
	g.syncChest()
	if g.overlay != overlayChest || g.openChestKind() != sim.EntityTreasure {
		t.Fatalf("overlay %v, chest kind %v; want the chest window on an old chest", g.overlay, g.openChestKind())
	}
	g.chest.layout(1280, 720)
	g.chest.drawn = true
	x, y := g.chest.chestSlotXY(1)
	g.chestMouse(x+3, y+3, true)
	if want := (sim.Command{Kind: sim.CmdChestTake, Slot: 1}); len(c.cmds) != 1 || c.cmds[0] != want {
		t.Fatalf("a click on the hoard sent %v, want %+v", c.cmds, want)
	}
}
