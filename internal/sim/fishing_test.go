package sim

import "testing"

// angler is shoreWorld with a player holding a rod, two worms in the bag and
// a fish at the shore beside them, holding still.
func angler(t *testing.T) (*World, PlayerID, *Creature) {
	t.Helper()
	w := shoreWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv.Add(ItemRod, 1) // lands in the held slot 0
	p.Inv.Add(ItemWorm, 2)
	fish := w.addCreature(CreatureFish, 18.5, 16.5)
	fish.State, fish.timer = CreatureIdle, 1000
	return w, id, fish
}

// tap presses and releases Action between two ticks, then steps once.
func tap(w *World, id PlayerID) {
	w.SetInput(id, Input{Action: true})
	w.SetInput(id, Input{})
	w.Step()
}

// waitForBite steps until the float dips, failing if it never does.
func waitForBite(t *testing.T, w *World, id PlayerID) {
	t.Helper()
	for range FishBiteMaxTicks + 1 {
		if st, _, _ := w.players[id].Cast(); st == CastBite {
			return
		}
		w.Step()
	}
	t.Fatal("no bite within the longest wait")
}

func TestQuickTapCastsTheLine(t *testing.T) {
	w, id, fish := angler(t)
	tap(w, id) // shorter than a tick: must not be lost
	p := w.players[id]
	st, bx, by := p.Cast()
	if st != CastWaiting || bx != fish.X || by != fish.Y {
		t.Fatalf("cast %d at (%.1f, %.1f), want the float on the fish at (%.1f, %.1f)", st, bx, by, fish.X, fish.Y)
	}
	if fish.State != CreatureHooked || p.Inv.Count(ItemWorm) != 2 || !p.Acting(w.Tick()) {
		t.Fatalf("fish state %d, worms %d, acting %v: want it hooked, the worm still in the bag, a cast swing",
			fish.State, p.Inv.Count(ItemWorm), p.Acting(w.Tick()))
	}
	// The hooked fish holds still, and nobody else can cast at it.
	x := fish.X
	stepN(w, FishBiteMinTicks-2)
	if fish.X != x {
		t.Error("a hooked fish swam off")
	}
	if cs := w.creatureCandidates(fish.X, fish.Y); len(cs) != 0 {
		t.Errorf("hooked fish offered as a target: %+v", cs)
	}
}

func TestStrikingOnTheBiteLandsTheFish(t *testing.T) {
	w, id, fish := angler(t)
	tap(w, id)
	waitForBite(t, w, id)
	tap(w, id)
	p := mustPlayer(t, w, id)
	if p.Inv.Count(ItemFish) != 1 || p.Inv.Count(ItemWorm) != 1 || !fish.gone && len(w.Creatures()) != 0 {
		t.Fatalf("fish %d, worms %d, creatures %d: want the fish landed for one worm",
			p.Inv.Count(ItemFish), p.Inv.Count(ItemWorm), len(w.Creatures()))
	}
	if p.Notice != NoticeCaught || ItemKind(p.NoticeArg) != ItemFish {
		t.Fatalf("notice %v/%d, want caught", p.Notice, p.NoticeArg)
	}
	if st, _, _ := p.Cast(); st != CastNone {
		t.Fatal("the line is still out")
	}
}

func TestStrikingTooEarlyScaresTheFish(t *testing.T) {
	w, id, fish := angler(t)
	tap(w, id)
	stepN(w, ActionCooldownTicks) // cooldown done, still before any bite
	tap(w, id)
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeTooEarly || p.Inv.Count(ItemWorm) != 2 || p.Inv.Count(ItemFish) != 0 {
		t.Fatalf("notice %v, worms %d, fish %d: want a scared fish and the worm kept", p.Notice, p.Inv.Count(ItemWorm), p.Inv.Count(ItemFish))
	}
	if fish.State == CreatureHooked {
		t.Fatal("the fish is still hooked")
	}
}

func TestHoldingSpaceDoesNotStrike(t *testing.T) {
	w, id, _ := angler(t)
	w.SetInput(id, Input{Action: true}) // pressed to cast, then held on
	stepN(w, FishBiteMaxTicks+FishBiteTicks+2)
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeBaitStolen || p.Inv.Count(ItemFish) != 0 {
		t.Fatalf("notice %v, fish %d: a held key must not strike", p.Notice, p.Inv.Count(ItemFish))
	}
}

func TestMissingTheBiteLosesTheWorm(t *testing.T) {
	w, id, fish := angler(t)
	tap(w, id)
	waitForBite(t, w, id)
	stepN(w, FishBiteTicks+1)
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeBaitStolen || ItemKind(p.NoticeArg) != ItemWorm {
		t.Fatalf("notice %v/%d, want the bait stolen", p.Notice, p.NoticeArg)
	}
	if p.Inv.Count(ItemWorm) != 1 || p.Inv.Count(ItemFish) != 0 || fish.State == CreatureHooked {
		t.Fatalf("worms %d, fish %d, fish state %d: want one worm gone and the fish free", p.Inv.Count(ItemWorm), p.Inv.Count(ItemFish), fish.State)
	}
}

func TestWalkingOrPuttingTheRodAwayReelsIn(t *testing.T) {
	w, id, fish := angler(t)
	tap(w, id)
	w.SetInput(id, Input{MoveX: -1})
	w.Step()
	if st, _, _ := w.players[id].Cast(); st != CastNone || fish.State == CreatureHooked {
		t.Fatal("walking off left the line out")
	}
	if w.players[id].Inv.Count(ItemWorm) != 2 {
		t.Fatal("reeling in cost a worm")
	}

	w, id, fish = angler(t)
	tap(w, id)
	w.players[id].Selected = 5 // an empty hotbar slot
	w.Step()
	if st, _, _ := w.players[id].Cast(); st != CastNone || fish.State == CreatureHooked {
		t.Fatal("putting the rod away left the line out")
	}
}

func TestALeavingAnglerFreesTheFish(t *testing.T) {
	w, id, fish := angler(t)
	tap(w, id)
	w.RemovePlayer(id)
	w.Step()
	if fish.State == CreatureHooked {
		t.Fatal("the fish stays hooked to a player who left")
	}
}
