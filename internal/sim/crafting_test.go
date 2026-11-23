package sim

import "testing"

// deepNight is a tick in the coldest part of the night.
var deepNight = uint64(DayTicks) + uint64(0.85*DayTicks)

func TestEveryNewRecipeCrafts(t *testing.T) {
	for _, id := range []RecipeID{RecipePickaxe, RecipeTorch, RecipeSpear, RecipeRod, RecipeWall, RecipeBed} {
		r, _ := RecipeByID(id)
		w := flatWorld()
		pid := w.AddPlayer("a")
		w.players[pid].Inv = InventoryWith(r.Inputs...)
		w.Apply(pid, Command{Kind: CmdCraft, Recipe: id})
		p := mustPlayer(t, w, pid)
		if p.Notice != NoticeCrafted {
			t.Errorf("%s: notice %v, want crafted", r.Name, p.Notice)
			continue
		}
		switch {
		case r.Gives.Valid() && p.Inv.Count(r.Gives) != 1:
			t.Errorf("%s: carries %d, want 1", r.Name, p.Inv.Count(r.Gives))
		case r.Places != EntityNone:
			if e, ok := w.EntityAt(16, 17); !ok || e.Kind != r.Places {
				t.Errorf("%s: placed %+v, want a %v in front", r.Name, e, r.Places.Name())
			}
		}
		if r.Gives.Valid() && r.Gives.Def().Tool {
			w.players[pid].Inv.Add(ItemStick, 10)
			w.players[pid].Inv.Add(ItemStone, 10)
			w.players[pid].Inv.Add(ItemGrass, 10)
			w.Apply(pid, Command{Kind: CmdCraft, Recipe: id})
			if p := mustPlayer(t, w, pid); p.Notice != NoticeHaveOne || p.Inv.Count(r.Gives) != 1 {
				t.Errorf("%s: a second one: notice %v, count %d", r.Name, p.Notice, p.Inv.Count(r.Gives))
			}
		}
	}
}

func TestRoastBerriesNeedAFire(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Inv = InventoryWith(Ingredient{ItemBerry, 2})
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeRoastBerry})
	if p := mustPlayer(t, w, id); p.Notice != NoticeNeedFire || p.Inv.Count(ItemBerry) != 2 {
		t.Fatalf("away from fire: notice %v, berries %d", p.Notice, p.Inv.Count(ItemBerry))
	}
	w.addEntity(EntityCampfire, 18, 16)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeRoastBerry})
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemRoastBerry) != 1 || p.Inv.Count(ItemBerry) != 1 {
		t.Fatalf("by the fire: roast %d, berries %d", p.Inv.Count(ItemRoastBerry), p.Inv.Count(ItemBerry))
	}
}

func TestBouldersNeedAPickaxe(t *testing.T) {
	hits := func(withPick bool) int {
		w := flatWorld()
		w.addEntity(EntityBoulder, 16, 17)
		id := w.AddPlayer("a")
		if withPick {
			w.players[id].Inv.Add(ItemPickaxe, 1)
		}
		w.SetInput(id, Input{Action: true})
		for n := 1; n < 20; n++ {
			w.Step()
			if _, ok := w.EntityAt(16, 17); !ok {
				return n
			}
			stepN(w, ActionCooldownTicks-1)
		}
		p := w.players[id]
		if p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ItemPickaxe {
			t.Errorf("bare hands: notice %v/%d, want need-tool for the pickaxe", p.Notice, p.NoticeArg)
		}
		return -1
	}
	bare, pick := hits(false), hits(true)
	if bare != -1 || pick != BoulderHits {
		t.Fatalf("hits: bare hands %d (want never), pickaxe %d (want %d)", bare, pick, BoulderHits)
	}
}

func TestTorchBurnsWhileHeldAndKeepsWarm(t *testing.T) {
	w := flatWorld()
	w.tick = deepNight
	a := w.AddPlayer("a")
	b := w.AddPlayer("b")
	w.players[b].X += 2 // within the torch's reach
	w.players[a].Inv.Add(ItemTorch, 2)
	w.players[a].Selected = 5 // an empty slot: the torches are not held
	w.Step()
	if p := mustPlayer(t, w, a); !p.Cold || p.TorchFuel() != 0 {
		t.Fatalf("torch in the bag: cold %v fuel %d, want cold and unlit", p.Cold, p.TorchFuel())
	}
	w.players[a].Selected = 0
	w.Step()
	if p := mustPlayer(t, w, a); p.Cold || p.TorchFuel() != TorchBurnTicks-1 {
		t.Fatalf("torch in hand: cold %v fuel %d", p.Cold, p.TorchFuel())
	}
	if p := mustPlayer(t, w, b); p.Cold {
		t.Error("a friend two tiles away is still freezing")
	}
	stepN(w, TorchBurnTicks-1)
	if p := mustPlayer(t, w, a); p.Inv.Count(ItemTorch) != 1 || p.TorchFuel() != 0 {
		t.Fatalf("after one torch's time: torches %d fuel %d, want one used up", p.Inv.Count(ItemTorch), p.TorchFuel())
	}
}

func TestSpearAndRod(t *testing.T) {
	w := shoreWorld()
	id := w.AddPlayer("a")
	p := w.players[id]

	snake := w.addCreature(CreatureSnake, 17.5, 16.5)
	p.Inv.Add(ItemSpear, 1) // slot 0, held
	w.catch(p, snake)
	w.catch(p, snake)
	if !snake.gone {
		t.Fatalf("snake hp %d after two spear thrusts, want caught", snake.HP)
	}

	// A spear reaches a little further.
	far := []CreatureCandidate{{ID: 1, Kind: CreatureRabbit, X: p.X + CreatureRabbit.Reach() + 0.3, Y: p.Y}}
	none := func(int, int) (TargetCandidate, bool) { return TargetCandidate{}, false }
	if _, ok := PickTarget(p.X, p.Y, false, none, far, 0); ok {
		t.Error("bare hands reach a rabbit out of range")
	}
	if _, ok := PickTarget(p.X, p.Y, false, none, far, ReachBonusFor(ItemSpear)); !ok {
		t.Error("a spear does not reach a rabbit just out of range")
	}

	// No fishing without a rod in hand, even with bait.
	fish := w.addCreature(CreatureFish, 18.5, 16.5)
	p.Inv.Add(ItemWorm, 1)
	if w.catch(p, fish) || p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ItemRod {
		t.Fatalf("fishing with a spear: notice %v/%d", p.Notice, p.NoticeArg)
	}
}

func TestWallsBlockAndComeDownWithAnAxe(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.addEntity(EntityWall, 17, 16)
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 20)
	if p := mustPlayer(t, w, id); p.X+PlayerRadius > 17 {
		t.Fatalf("walked into the wall: x %.2f", p.X)
	}

	w.SetInput(id, Input{Action: true})
	stepN(w, ActionCooldownTicks)
	if p := mustPlayer(t, w, id); p.Notice != NoticeNeedTool {
		t.Fatalf("bare hands on a wall: notice %v, want need-tool", p.Notice)
	}
	w.players[id].Inv.Add(ItemAxe, 1)
	stepN(w, WallHits*ActionCooldownTicks)
	if _, ok := w.EntityAt(17, 16); ok {
		t.Fatal("wall still standing")
	}
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemWood) != WallRefund {
		t.Errorf("wood back %d, want %d", p.Inv.Count(ItemWood), WallRefund)
	}
}

func TestSleepingThroughTheNight(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.addEntity(EntityBed, 17, 16)

	w.SetInput(id, Input{Action: true})
	w.Step()
	if p := mustPlayer(t, w, id); p.Asleep || p.Notice != NoticeNotTired {
		t.Fatalf("by day: asleep %v notice %v, want not tired", p.Asleep, p.Notice)
	}

	w.tick = deepNight
	w.players[id].Health = 50
	stepN(w, ActionCooldownTicks)
	w.SetInput(id, Input{})
	p := mustPlayer(t, w, id)
	if !p.Asleep || p.X != 17.5 || !w.AllAsleep() {
		t.Fatalf("at night: asleep %v at x %.2f, all asleep %v", p.Asleep, p.X, w.AllAsleep())
	}
	stepN(w, 100)
	if p := mustPlayer(t, w, id); p.Cold || p.Health <= 50 {
		t.Fatalf("in bed: cold %v health %.1f, want warm and healing", p.Cold, p.Health)
	}

	// Morning wakes the sleeper.
	for w.players[id].Asleep && w.tick < deepNight+DayTicks {
		w.Step()
	}
	if f := DayFraction(w.tick); w.players[id].Asleep || f > 0.1 {
		t.Fatalf("still asleep %v at day fraction %.2f", w.players[id].Asleep, f)
	}

	// A walk key or a bite wakes a sleeper too.
	w.tick = deepNight
	w.players[id].Asleep, w.players[id].bed = true, w.entityAt(17, 16).ID
	w.SetInput(id, Input{MoveY: 1})
	w.Step()
	if w.players[id].Asleep {
		t.Error("a walk key did not wake the sleeper")
	}
	w.players[id].Asleep = true
	hurt(w.players[id], 1)
	if w.players[id].Asleep {
		t.Error("a bite did not wake the sleeper")
	}
}

func TestTorchFuelIsSaved(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].torchFuel = 123
	ps := savePlayer(w.players[id])
	w2 := flatWorld()
	p := &Player{}
	w2.restorePlayer(p, ps)
	if p.TorchFuel() != 123 {
		t.Fatalf("torch fuel after load = %d, want 123", p.TorchFuel())
	}
}
