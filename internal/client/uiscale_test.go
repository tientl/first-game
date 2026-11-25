package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestUIScaleFollowsTheScreen(t *testing.T) {
	defer SetUIScale(0)
	for _, c := range []struct{ sh, want int }{{240, 1}, {720, 1}, {900, 1}, {1080, 2}, {1440, 2}, {1620, 3}, {2160, 4}} {
		if got := uiScaleFor(c.sh); got != c.want {
			t.Errorf("auto scale at %dp is %d, want %d", c.sh, got, c.want)
		}
	}
	SetUIScale(2)
	if uiScaleFor(1080) != 2 || uiScaleFor(720) != 1 || uiScaleFor(2160) != 2 {
		t.Errorf("chosen scale 2: %d at 1080p, %d at 720p, %d at 4K", uiScaleFor(1080), uiScaleFor(720), uiScaleFor(2160))
	}
}

func TestUIPoint(t *testing.T) {
	for _, c := range []struct{ mx, my, s, x, y int }{
		{0, 0, 2, 0, 0}, {1, 1, 2, 0, 0}, {2, 3, 2, 1, 1}, {1919, 1079, 2, 959, 539}, {-1, 5, 2, -1, 2}, {7, 9, 1, 7, 9},
	} {
		if x, y := uiPoint(c.mx, c.my, c.s); x != c.x || y != c.y {
			t.Errorf("uiPoint(%d, %d, %d) = %d, %d; want %d, %d", c.mx, c.my, c.s, x, y, c.x, c.y)
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
