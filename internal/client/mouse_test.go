package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestRightClickDispatchByItemType(t *testing.T) {
	spot := ghostView{ok: true, valid: true, tx: 12, ty: 9}
	head := sim.HeadItem(sim.CreatureRaptor)
	for _, c := range []struct {
		name string
		held sim.ItemKind
		ctx  useContext
		want sim.Command
	}{
		{"berries are eaten", sim.ItemBerry, useContext{}, sim.Command{Kind: sim.CmdEat, Slot: 3}},
		{"medicine is taken", sim.ItemBandage, useContext{}, sim.Command{Kind: sim.CmdEat, Slot: 3}},
		{"raw meat away from a fire is eaten", sim.ItemMeat, useContext{}, sim.Command{Kind: sim.CmdEat, Slot: 3}},
		{"raw meat by a fire is cooked", sim.ItemMeat, useContext{byFire: true}, sim.Command{Kind: sim.CmdCraft, Recipe: sim.RecipeCookMeat}},
		{"raw fish by a fire is cooked", sim.ItemFish, useContext{byFire: true}, sim.Command{Kind: sim.CmdCraft, Recipe: sim.RecipeCookFish}},
		{"berries on a beast that eats them feed it", sim.ItemBerry, useContext{feed: true, byFire: true}, sim.Command{Kind: sim.CmdFeed}},
		{"a torch goes in front, not where the pointer is", sim.ItemTorch, useContext{spot: spot}, sim.Command{Kind: sim.CmdPlace}},
		{"a seed is planted", sim.ItemBerrySeed, useContext{spot: spot}, sim.Command{Kind: sim.CmdPlace, At: true, TX: 12, TY: 9}},
		{"a head goes on a post", head, useContext{spot: spot}, sim.Command{Kind: sim.CmdPlace, At: true, TX: 12, TY: 9}},
		{"dung goes on what is pointed at", sim.ItemDung, useContext{aim: true, aimTX: 4, aimTY: 5}, sim.Command{Kind: sim.CmdPlace, At: true, TX: 4, TY: 5}},
		{"a boat is launched", sim.ItemRaft, useContext{spot: spot}, sim.Command{Kind: sim.CmdPlace}},
		{"an axe does nothing", sim.ItemAxe, useContext{}, sim.Command{}},
		{"wood does nothing", sim.ItemWood, useContext{byFire: true}, sim.Command{}},
		{"an empty hand does nothing", sim.ItemNone, useContext{}, sim.Command{}},
	} {
		got, hint := rightClickUse(c.held, 3, c.ctx)
		if got != c.want {
			t.Errorf("%s: sent %+v, want %+v", c.name, got, c.want)
		}
		if (got.Kind == sim.CmdNone) != (hint != "") {
			t.Errorf("%s: hint %q with %+v; want a hint exactly when nothing is done", c.name, hint, got)
		}
	}
}

// craftForMouse is a game with the crafting window open and laid out.
func craftForMouse() (*Game, *sentConn) {
	g := readyGame()
	c := &sentConn{}
	g.conn = c
	g.overlay = overlayCraft
	cv := &g.craft
	cv.drawn, cv.cardX, cv.cardY, cv.cardW, cv.rowsShown = true, 40, 120, 240, 40
	cv.tabY, cv.tabH = 80, 28
	for t := range numCraftTabs {
		cv.tabX[t], cv.tabW[t] = 40+float64(t)*130, 120
	}
	return g, c
}

func TestCraftPanelListsEveryRecipeAndAClickCrafts(t *testing.T) {
	g, c := craftForMouse()
	cv := &g.craft
	if n := len(cv.shown()); n != int(sim.NumRecipes) {
		t.Fatalf("the all tab lists %d recipes, want all %d", n, sim.NumRecipes)
	}
	seen := 0
	for tab := tabTools; tab < numCraftTabs; tab++ {
		cv.tab = tab
		seen += len(cv.shown())
	}
	if seen != int(sim.NumRecipes) {
		t.Fatalf("the tabs list %d recipes between them, want each of %d once", seen, sim.NumRecipes)
	}

	// A click on a tab opens it.
	cv.tab = tabAll
	g.craftMouse(cv.tabX[tabCook]+5, cv.tabY+5, true)
	if cv.tab != tabCook {
		t.Fatalf("tab %d after clicking the cooking tab", cv.tab)
	}

	// A click on an item's card crafts it.
	cv.tab = tabAll
	i := -1
	for j, r := range cv.shown() {
		if r.ID == sim.RecipeAxe {
			i = j
		}
	}
	x, y := cv.cardXY(i)
	g.craftMouse(x+5, y+5, false)
	if cv.hover != i || len(c.cmds) != 0 {
		t.Fatalf("hovering: hover %d sent %v", cv.hover, c.cmds)
	}
	g.craftMouse(x+5, y+5, true)
	if len(c.cmds) != 1 || c.cmds[0] != (sim.Command{Kind: sim.CmdCraft, Recipe: sim.RecipeAxe}) {
		t.Fatalf("clicking the axe sent %v", c.cmds)
	}
	if g.canCraft(cv.shown()[i]) {
		t.Error("an axe lit with an empty bag")
	}
	g.curr.Self.Inv.Add(sim.ItemStick, 2)
	g.curr.Self.Inv.Add(sim.ItemPebble, 1)
	if !g.canCraft(cv.shown()[i]) {
		t.Error("an axe not lit with its sticks and pebble")
	}

	// A structure without its ingredients: nothing sent, nothing chosen.
	x, y = cv.cardXY(int(sim.RecipeCampfire))
	g.craftMouse(x+5, y+5, true)
	if len(c.cmds) != 1 || g.build.on {
		t.Fatalf("a campfire without wood: sent %v, build %+v", c.cmds[1:], g.build)
	}
	// With them: the window closes and the campfire waits for a tile.
	g.curr.Self.Inv.Add(sim.ItemWood, 2)
	g.curr.Self.Inv.Add(sim.ItemGrass, 2)
	g.curr.Self.Inv.Add(sim.ItemStone, 3)
	g.craftMouse(x+5, y+5, true)
	if len(c.cmds) != 1 || !g.build.on || g.build.recipe != sim.RecipeCampfire || g.overlay != overlayNone {
		t.Fatalf("a campfire: sent %v, build %+v, overlay %d; want build mode", c.cmds[1:], g.build, g.overlay)
	}
}

// fieldGame is a ready game with me at (10.5, 10.5) on grass.
func fieldGame(ents ...protocol.EntityView) (*Game, *sentConn) {
	g := readyGame()
	c := &sentConn{}
	g.conn = c
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Entities: ents})
	return g, c
}

func TestBuildGhostAndClick(t *testing.T) {
	bush := protocol.EntityView{ID: 5, Kind: sim.EntityBerryBush, TX: 9, TY: 10, Ready: true}
	g, c := fieldGame(bush)
	g.curr.Self.Inv.Add(sim.ItemWood, 4)
	g.StartBuild(sim.RecipeWall)

	g.SetAim(2, 0.2) // two tiles east
	gv := g.ghost()
	if !gv.ok || !gv.valid || gv.tx != 12 || gv.ty != 10 || gv.kind != sim.EntityWall {
		t.Fatalf("ghost two tiles east: %+v, want a green wall on (12,10)", gv)
	}
	if !g.leftClick(-1, -1) {
		t.Fatal("a left click in build mode was not used up")
	}
	want := sim.Command{Kind: sim.CmdCraft, Recipe: sim.RecipeWall, At: true, TX: 12, TY: 10}
	if len(c.cmds) != 1 || c.cmds[0] != want {
		t.Fatalf("sent %v, want %+v", c.cmds, want)
	}

	for _, at := range []struct {
		dx, dy float64
		why    sim.NoticeKind
	}{{5, 0, sim.NoticeTooFar}, {-1.2, 0, sim.NoticeNoSpace}, {0, 0, sim.NoticeNoSpace}} {
		g.SetAim(at.dx, at.dy)
		if gv := g.ghost(); !gv.ok || gv.valid || gv.why != at.why {
			t.Errorf("ghost at %+.1f,%+.1f: %+v, want red (%v)", at.dx, at.dy, gv, at.why)
		}
	}
	g.leftClick(-1, -1)
	if len(c.cmds) != 1 {
		t.Fatalf("a click on a red ghost sent %v", c.cmds[1:])
	}
	if tg, _ := g.target(); tg.ok {
		t.Errorf("building: the left button targets %+v", tg)
	}

	// Out of wood: build mode ends.
	g.curr.Self.Inv = sim.Inventory{}
	g.keepBuild()
	if g.build.on {
		t.Error("build mode outlived the ingredients")
	}
}

// A torch in the hand already lights the way: no ghost shows where it would
// go, and the right button plants it in front.
func TestATorchInHandShowsNoGhost(t *testing.T) {
	bush := protocol.EntityView{ID: 5, Kind: sim.EntityBerryBush, TX: 11, TY: 10, Ready: true}
	g, c := fieldGame(bush)
	g.curr.Players[0].Held = sim.ItemTorch
	g.curr.Self.Inv[0] = sim.Slot{Item: sim.ItemTorch, Count: 2}
	g.SetAim(0, 1.2) // the free tile below
	if gv := g.ghost(); gv.ok {
		t.Fatalf("torch over free ground: ghost %+v, want none", gv)
	}
	if g.leftClick(-1, -1) {
		t.Error("a left click with a torch set something down")
	}
	g.rightClick()
	if want := (sim.Command{Kind: sim.CmdPlace}); len(c.cmds) != 1 || c.cmds[0] != want {
		t.Fatalf("right click with a torch sent %v, want %+v (in front)", c.cmds, want)
	}
	if key, s, _ := g.contextHint(); key != "Chuột phải:" || s != "cắm đuốc xuống trước mặt" {
		t.Errorf("the hint over the hotbar is %q %q", key, s)
	}
	g.SetAim(1, 0) // on the bush
	if tg, _ := g.target(); !tg.ok || !tg.aimed || tg.e.ID != bush.ID {
		t.Fatalf("target %+v, want the bush under the pointer", tg)
	}
}

func TestHoverTargetsWhatIsUnderThePointer(t *testing.T) {
	bush := protocol.EntityView{ID: 5, Kind: sim.EntityBerryBush, TX: 11, TY: 10, Ready: true}
	tree := protocol.EntityView{ID: 6, Kind: sim.EntityTree, TX: 11, TY: 11, Ready: true}
	far := protocol.EntityView{ID: 7, Kind: sim.EntityTree, TX: 15, TY: 10, Ready: true}
	g, _ := fieldGame(bush, tree, far)
	g.curr.Self.Inv.Add(sim.ItemAxe, 1)

	g.SetAim(1, 1) // the tree's tile
	tg, fv := g.target()
	if !tg.ok || !tg.aimed || tg.e.ID != tree.ID || fv.ok {
		t.Fatalf("pointing at the tree: target %+v far %+v", tg, fv)
	}
	g.SetAim(5, 0) // the far tree: red, and the left button falls back to the nearest
	tg, fv = g.target()
	if !fv.ok || fv.e.ID != far.ID || !tg.ok || tg.aimed || tg.e.ID != bush.ID {
		t.Fatalf("pointing at the far tree: target %+v far %+v; want the bush, and the far tree red", tg, fv)
	}
}

func TestHotbarSlotAt(t *testing.T) {
	const sw, sh = 1280, 720
	x0, y := hotbarX0(sw), hotbarY(sh)
	for i := range sim.HotbarSlots {
		if got, ok := hotbarSlotAt(x0+i*(hotPx+hotGap)+hotPx/2, y+5, sw, sh); !ok || got != i {
			t.Fatalf("slot %d is hit as %d %v", i, got, ok)
		}
	}
	if _, ok := hotbarSlotAt(x0+hotPx+1, y+5, sw, sh); ok {
		t.Error("the gap between slots is a slot")
	}
	if _, ok := hotbarSlotAt(sw/2, y-10, sw, sh); ok {
		t.Error("above the hotbar is a slot")
	}
}
