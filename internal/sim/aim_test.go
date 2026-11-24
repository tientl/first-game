package sim

import (
	"math"
	"testing"
)

// aimAt is held Action with the mouse at (x, y).
func aimAt(x, y float64) Input {
	return Input{Action: true, Aim: true, AimX: float32(x), AimY: float32(y)}
}

func TestAimPicksTheFarTreeOverTheNearBush(t *testing.T) {
	w := flatWorld() // the player stands at (16.5, 16.5)
	bush := w.addEntity(EntityBerryBush, 17, 16)
	tree := w.addEntity(EntityTree, 17, 17) // diagonal: further, but still in reach
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemAxe, 1)
	bushHP, treeHP := bush.HP, tree.HP

	w.SetInput(id, aimAt(17.5, 17.5)) // on the tree's own tile
	w.Step()
	if tree.HP >= treeHP || bush.HP != bushHP {
		t.Fatalf("aiming at the tree: tree hp %d (was %d), bush hp %d (was %d); want the tree hit", tree.HP, treeHP, bush.HP, bushHP)
	}
	if p := mustPlayer(t, w, id); p.Dir != DirRight { // a diagonal turns sideways
		t.Errorf("dir = %v, want the player turned to the tree", p.Dir)
	}

	// The canopy above the trunk is the tree too.
	if tg, in, ok := AimedTarget(16.5, 16.5, 17.5, 15.9, false, w.targetCandidate, nil, 0); !ok || !in || tg.Entity != tree.ID {
		t.Errorf("pointing at the canopy picks %+v (in reach %v, found %v), want the tree", tg, in, ok)
	}

	// Without an aim the nearest still wins: the bush.
	w2 := flatWorld()
	bush2 := w2.addEntity(EntityBerryBush, 17, 16)
	tree2 := w2.addEntity(EntityTree, 17, 17)
	id2 := w2.AddPlayer("a")
	w2.players[id2].Inv.Add(ItemAxe, 1)
	b2, t2 := bush2.HP, tree2.HP
	w2.SetInput(id2, Input{Action: true})
	w2.Step()
	if bush2.HP >= b2 || tree2.HP != t2 {
		t.Fatalf("no aim: bush hp %d (was %d), tree hp %d (was %d); want the nearest, the bush", bush2.HP, b2, tree2.HP, t2)
	}
}

func TestAimOutOfReachFallsBackToTheNearest(t *testing.T) {
	w := flatWorld()
	bush := w.addEntity(EntityBerryBush, 17, 16)
	far := w.addEntity(EntityTree, 20, 16) // four tiles off
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemAxe, 1)
	bushHP, farHP := bush.HP, far.HP

	if _, in, ok := AimedTarget(16.5, 16.5, 20.5, 16.5, false, w.targetCandidate, nil, 0); !ok || in {
		t.Fatalf("the far tree: found %v, in reach %v; want found and out of reach", ok, in)
	}
	w.SetInput(id, aimAt(20.5, 16.5))
	w.Step()
	if far.HP != farHP || bush.HP >= bushHP {
		t.Fatalf("far tree hp %d (was %d), bush hp %d (was %d): want the bush in reach worked on instead", far.HP, farHP, bush.HP, bushHP)
	}
}

func TestTheServerRejectsAimBeyondReach(t *testing.T) {
	w := flatWorld()
	far := w.addEntity(EntityTree, 20, 16)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemAxe, 1)
	hp := far.HP
	w.SetInput(id, aimAt(20.5, 16.5))
	stepN(w, 3*ActionCooldownTicks)
	if far.HP != hp || w.players[id].actedAt != 0 {
		t.Fatalf("tree hp %d (was %d), acted at %d: a tree four tiles off was chopped", far.HP, hp, w.players[id].actedAt)
	}

	// Rubbish aim from the network is no aim at all.
	w.SetInput(id, Input{Action: true, Aim: true, AimX: float32(math.NaN()), AimY: 3})
	if _, _, ok := w.players[id].Aim(); ok {
		t.Error("a NaN aim was kept")
	}
}

func TestAimPrefersTargetsTowardsThePointer(t *testing.T) {
	w := flatWorld()
	left := w.addEntity(EntityBerryBush, 15, 16)  // 1 tile west
	right := w.addEntity(EntityGrassTuft, 17, 16) // 1 tile east: as near, but a later id
	at := w.targetCandidate
	if tg, ok := ResolveTarget(16.5, 16.5, false, 0, 0, false, at, nil, 0); !ok || tg.Entity != left.ID {
		t.Fatalf("no aim: %+v, want the bush, first on a tie", tg)
	}
	// Pointing east at empty ground well past the tuft: the tuft, ahead, beats the bush behind.
	if tg, ok := ResolveTarget(16.5, 16.5, true, 25, 16.5, false, at, nil, 0); !ok || tg.Entity != right.ID {
		t.Fatalf("aiming east: %+v, want the tuft ahead", tg)
	}
}

func TestThePlayerFacesThePointer(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.SetInput(id, Input{MoveX: 1, Aim: true, AimX: 10, AimY: 16.5})
	w.Step()
	if p := mustPlayer(t, w, id); p.Dir != DirLeft || p.X <= 16.5 {
		t.Fatalf("walking east aiming west: dir %v x %.2f, want facing left while moving right", p.Dir, p.X)
	}
	w.SetInput(id, Input{MoveX: 1})
	w.Step()
	if p := mustPlayer(t, w, id); p.Dir != DirRight {
		t.Fatalf("no aim: dir %v, want the way walked", p.Dir)
	}
}

func TestBuildingOnTheTileChosen(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	give := func() {
		r, _ := RecipeByID(RecipeCampfire)
		for _, in := range r.Inputs {
			p.Inv.Add(in.Item, in.Count)
		}
	}
	give()
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCampfire, At: true, TX: 18, TY: 16})
	if e := w.entityAt(18, 16); e == nil || e.Kind != EntityCampfire {
		t.Fatalf("campfire not built on the tile chosen two tiles east (notice %v)", p.Notice)
	}
	if p.Dir != DirRight {
		t.Errorf("dir %v, want the builder turned to it", p.Dir)
	}

	// Too far: refused, nothing used up.
	give()
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCampfire, At: true, TX: 22, TY: 16})
	if w.entityAt(22, 16) != nil || p.Notice != NoticeTooFar || p.Inv.Count(ItemWood) != 2 {
		t.Fatalf("building six tiles off: entity %v, notice %v, wood %d; want refused as too far", w.entityAt(22, 16), p.Notice, p.Inv.Count(ItemWood))
	}
	// Taken, or the builder's own tile: refused.
	for _, tile := range [][2]int16{{18, 16}, {16, 16}} {
		w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCampfire, At: true, TX: tile[0], TY: tile[1]})
		if p.Notice != NoticeNoSpace || p.Inv.Count(ItemWood) != 2 {
			t.Fatalf("building on %v: notice %v, wood %d; want no space", tile, p.Notice, p.Inv.Count(ItemWood))
		}
	}
	// A wall is not built on another player.
	other := w.AddPlayer("b")
	w.players[other].X, w.players[other].Y = 15.5, 17.5
	p.Inv.Add(ItemWood, 2)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeWall, At: true, TX: 15, TY: 17})
	if w.entityAt(15, 17) != nil {
		t.Fatal("a wall was built on top of another player")
	}
}

func TestATorchGoesOnThePointedTile(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv.Add(ItemTorch, 2) // slot 0, held
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 15, TY: 15})
	if e := w.entityAt(15, 15); e == nil || e.Kind != EntityTorchStand || p.Inv.Count(ItemTorch) != 1 {
		t.Fatalf("torch at (15,15): %v, torches left %d", e, p.Inv.Count(ItemTorch))
	}
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 12, TY: 16})
	if w.entityAt(12, 16) != nil || p.Notice != NoticeTooFar || p.Inv.Count(ItemTorch) != 1 {
		t.Fatalf("a torch four tiles off: notice %v, torches %d; want refused", p.Notice, p.Inv.Count(ItemTorch))
	}
	// Without a tile it still goes in front, as V always did.
	p.Dir = DirDown
	w.Apply(id, Command{Kind: CmdPlace})
	if e := w.entityAt(16, 17); e == nil || e.Kind != EntityTorchStand {
		t.Fatal("V without a tile did not set the torch in front")
	}
}

func TestEPicksUpThePilePointedAt(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	near := &Drop{ID: 1, Item: ItemStick, Count: 1, X: 16.9, Y: 16.5}
	pointed := &Drop{ID: 2, Item: ItemPebble, Count: 1, X: 16.1, Y: 17.4}
	w.drops = append(w.drops, near, pointed)
	w.SetInput(id, Input{Aim: true, AimX: 16.1, AimY: 17.3})
	w.Apply(id, Command{Kind: CmdInteract})
	p := w.players[id]
	if p.Inv.Count(ItemPebble) != 1 || p.Inv.Count(ItemStick) != 0 {
		t.Fatalf("pebbles %d sticks %d: want the pile pointed at, not the nearest", p.Inv.Count(ItemPebble), p.Inv.Count(ItemStick))
	}
	// No aim: the nearest.
	w.SetInput(id, Input{})
	w.Apply(id, Command{Kind: CmdInteract})
	if p.Inv.Count(ItemStick) != 1 {
		t.Fatal("E without an aim did not pick up the nearest pile")
	}
	// Nothing left: E says so.
	w.Apply(id, Command{Kind: CmdInteract})
	if p.Notice != NoticeNothingNear {
		t.Fatalf("notice %v, want nothing near", p.Notice)
	}
}

func TestEOpensAChest(t *testing.T) {
	w := flatWorld()
	chest := w.addEntity(EntityChest, 17, 16)
	id := w.AddPlayer("a")
	w.Apply(id, Command{Kind: CmdInteract})
	if w.players[id].chest != chest.ID {
		t.Fatal("E beside a chest did not open it")
	}
}

func TestRightClickFeedsTheBeastPointedAt(t *testing.T) {
	w := flatWorld()
	trike := w.addCreature(CreatureTrike, 17.6, 16.5)
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv.Add(ItemBerry, 5) // held
	w.SetInput(id, Input{Aim: true, AimX: float32(trike.X), AimY: float32(trike.Y - 0.3)})
	w.Apply(id, Command{Kind: CmdFeed})
	if trike.Tame != 1 || p.Inv.Count(ItemBerry) != 4 {
		t.Fatalf("after a right click with berries: tame %d berries %d; want one meal given", trike.Tame, p.Inv.Count(ItemBerry))
	}
	if trike.HP != trike.Kind.MaxHP() {
		t.Error("feeding hurt the beast")
	}
	// Feeding by the left button on the beast still works too.
	stepN(w, FeedCD+1)
	w.SetInput(id, aimAt(float64(trike.X), float64(trike.Y)-0.3))
	w.Step()
	if trike.Tame != 2 {
		t.Fatalf("tame %d after a left click with berries, want 2", trike.Tame)
	}
}
