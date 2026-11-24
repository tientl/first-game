package client

import (
	"math"
	"slices"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestAnimalSoundsAreUsable(t *testing.T) {
	bank := genSFX()
	for _, name := range []string{"hiss", "splash", "squeak", "pick_worm", "pick_meat", "sizzle", "plop", "pinch", "chomp"} {
		vs := bank[name]
		if len(vs) == 0 {
			t.Errorf("%s: missing", name)
			continue
		}
		for _, x := range vs {
			checkFinite(t, name, x)
			peak, rms := stats(x)
			if peak > 1.0001 || peak < 0.5 || rms < 0.005 {
				t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
			}
			if end := math.Abs(float64(x[len(x)-1])); end > 0.01 {
				t.Errorf("%s: ends at %.3f, would click", name, end)
			}
		}
	}
}

func TestSnakeHissesAndFishSplash(t *testing.T) {
	played := testSound(t)
	snd.setListener(10.5, 10.5)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight}
	snake := protocol.CreatureView{ID: 3, Kind: sim.CreatureSnake, X: 13, Y: 10.5}
	fish := protocol.CreatureView{ID: 4, Kind: sim.CreatureFish, X: 10.5, Y: 12.3}
	s := protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{snake, fish}}
	s.Self.Inv.Add(sim.ItemWorm, 1)
	g.onSnapshot(s)

	// The snake goes for me while I cast at the fish and land it.
	snake.State = sim.CreatureChase
	me.Acting = true
	s = protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{snake}}
	s.Self.Inv.Add(sim.ItemFish, 1)
	g.onSnapshot(s)

	for _, want := range []string{"hiss", "splash", "pick_meat"} {
		if !slices.Contains(*played, want) {
			t.Errorf("played %v, missing %q", *played, want)
		}
	}
}

func TestTargetingAnAnimal(t *testing.T) {
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5}
	rabbit := protocol.CreatureView{ID: 9, Kind: sim.CreatureRabbit, X: 11.2, Y: 10.5, State: sim.CreatureSleep}
	bush := protocol.EntityView{ID: 5, Kind: sim.EntityBerryBush, TX: 11, TY: 11, Ready: true}
	g.onSnapshot(protocol.Snapshot{
		Tick: 1, Players: []protocol.PlayerView{me},
		Entities: []protocol.EntityView{bush}, Creatures: []protocol.CreatureView{rabbit},
	})
	tg, _ := g.target()
	if !tg.ok || !tg.creature || tg.c.ID != rabbit.ID {
		t.Fatalf("target %+v, want the sleeping rabbit right beside me", tg)
	}
	if label, _ := g.creatureLabel(tg.c); label != "Thỏ (đang ngủ) — bắt" {
		t.Errorf("label %q", label)
	}

	fish := protocol.CreatureView{ID: 2, Kind: sim.CreatureFish, X: 10.5, Y: 12.4}
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{fish}})
	if _, col := g.creatureLabel(fish); col != colBad {
		t.Error("a fish with no worm in the bag should be marked as not catchable")
	}
}

func TestEKeyEatsOnlyWhatIsChosen(t *testing.T) {
	g := readyGame()
	if _, ok := g.slotToEat(); ok {
		t.Fatal("found food in an empty bag")
	}
	inv := &g.curr.Self.Inv
	inv[0] = sim.Slot{Item: sim.ItemWood, Count: 3}
	inv[7] = sim.Slot{Item: sim.ItemBerry, Count: 2}
	inv[sim.HotbarSlots+5] = sim.Slot{Item: sim.ItemFish, Count: 1}
	g.hotSel = 0
	if got, ok := g.slotToEat(); ok {
		t.Fatalf("holding wood, E eats slot %d; want nothing", got)
	}
	inv[3] = sim.Slot{Item: sim.ItemMeat, Count: 1}
	g.hotSel = 3
	if got, ok := g.slotToEat(); !ok || got != 3 {
		t.Fatalf("eats slot %d, want the held meat", got)
	}
	g.overlay, g.bag.cursor = overlayBag, sim.HotbarSlots+5
	if got, ok := g.slotToEat(); !ok || got != sim.HotbarSlots+5 {
		t.Fatalf("eats slot %d, want the fish pointed at in the bag", got)
	}
	g.bag.cursor = 0
	if got, ok := g.slotToEat(); ok {
		t.Fatalf("pointing at wood in the bag, E eats slot %d; want nothing", got)
	}
}

func TestHiddenAnimalsAreOutOfReach(t *testing.T) {
	played := testSound(t)
	snd.setListener(10.5, 10.5)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5}
	burrow := protocol.EntityView{ID: 5, Kind: sim.EntityBurrow, TX: 11, TY: 10, Ready: true}
	rabbit := protocol.CreatureView{ID: 9, Kind: sim.CreatureRabbit, X: 11.3, Y: 10.5, State: sim.CreatureFlee}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me},
		Entities: []protocol.EntityView{burrow}, Creatures: []protocol.CreatureView{rabbit}})

	rabbit.State, rabbit.X = sim.CreatureHidden, 11.5
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me},
		Entities: []protocol.EntityView{burrow}, Creatures: []protocol.CreatureView{rabbit}})
	if tg, _ := g.target(); tg.ok {
		t.Fatalf("target %+v: a burrow and a rabbit down it are not targets", tg)
	}
	if !slices.Contains(*played, "rustle") || slices.Contains(*played, "squeak") {
		t.Fatalf("played %v, want a rustle at the hole and no catch", *played)
	}
}

func TestALurkingCrocIsOnlyEyes(t *testing.T) {
	played := testSound(t)
	snd.setListener(10.5, 10.5)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5}
	croc := protocol.CreatureView{ID: 7, Kind: sim.CreatureCroc, X: 11.5, Y: 10.5, State: sim.CreatureLurk}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{croc}})
	if tg, _ := g.target(); tg.ok {
		t.Fatalf("target %+v: a crocodile under water is out of reach", tg)
	}
	croc.State = sim.CreatureChase // out it comes
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{croc}})
	croc.State = sim.CreatureStrike
	g.onSnapshot(protocol.Snapshot{Tick: 3, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{croc}})
	if !slices.Contains(*played, "splash") || !slices.Contains(*played, "chomp") {
		t.Fatalf("played %v, want the splash of the ambush and the chomp", *played)
	}
	if tg, _ := g.target(); !tg.ok || tg.c.ID != croc.ID {
		t.Fatalf("target %+v, want the crocodile once it is out", tg)
	}
	if _, col := g.creatureLabel(croc); col != colBad {
		t.Error("the crocodile's label should warn")
	}
}
