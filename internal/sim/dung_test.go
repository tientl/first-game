package sim

import "testing"

func TestGrazersLeaveDung(t *testing.T) {
	w := flatWorld()
	w.addCreature(CreatureTrike, 10.5, 10.5)
	w.addCreature(CreatureRabbit, 20.5, 20.5)
	stepN(w, 2*DungEvery)
	heaps := 0
	for _, d := range w.Drops() {
		if d.Item != ItemDung {
			continue
		}
		heaps++
		if d.Count != dungOf(CreatureTrike) {
			t.Errorf("a heap of %d: only the three-horn should leave dung, %d at a time", d.Count, dungOf(CreatureTrike))
		}
	}
	if heaps == 0 {
		t.Fatal("a three-horn left no dung in two thirds of a day")
	}
}

func TestDungLastsLongerThanOtherPiles(t *testing.T) {
	w := flatWorld()
	w.dropAt(10, 10, []Ingredient{{ItemDung, 1}, {ItemStick, 1}})
	stepN(w, DropLifeTicks+5)
	var dung, stick bool
	for _, d := range w.Drops() {
		dung = dung || d.Item == ItemDung
		stick = stick || d.Item == ItemStick
	}
	if !dung || stick {
		t.Errorf("after %d ticks: dung lying %v, stick lying %v", DropLifeTicks+5, dung, stick)
	}
}

func TestDungFeedsWhatGrows(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a") // on (16, 16)
	p := w.players[id]
	p.Dir = DirRight

	// Nothing growing in front: the dung stays in the hand.
	holding(w, id, ItemDung)
	w.Apply(id, Command{Kind: CmdPlace})
	if p.Inv.Count(ItemDung) != 1 {
		t.Fatal("dung spent on bare ground")
	}

	// A sapling grows in half the time left.
	sap := w.addEntity(EntitySapling, 17, 16)
	sap.Var, sap.Ready = uint8(SeedBerry), false
	sap.RegrowAt = w.tick + 1000
	w.watch(sap)
	w.Apply(id, Command{Kind: CmdPlace})
	if p.Inv.Count(ItemDung) != 0 {
		t.Fatal("feeding a sapling did not use up the dung")
	}
	if left := sap.RegrowAt - w.tick; left != 1000/DungBoost {
		t.Fatalf("fed sapling: %d ticks left, want %d", left, 1000/DungBoost)
	}
	stepN(w, 1000/DungBoost+1)
	if sap.Kind != EntityBerryBush {
		t.Fatalf("fed sapling is %v, want a berry bush by now", sap.Kind)
	}

	// A bush picked bare fills again sooner.
	sap.Ready, sap.RegrowAt = false, w.tick+BushRegrowTicks
	w.watch(sap)
	holding(w, id, ItemDung)
	w.Apply(id, Command{Kind: CmdPlace})
	if left := sap.RegrowAt - w.tick; left != BushRegrowTicks/DungBoost {
		t.Fatalf("fed bush: %d ticks left, want %d", left, BushRegrowTicks/DungBoost)
	}

	// A fed bed grows its crop twice as fast.
	w.removeEntity(sap)
	bed := w.till(17, 16)
	holding(w, id, ItemDung)
	w.Apply(id, Command{Kind: CmdPlace})
	if !bed.Big {
		t.Fatal("the bed was not fed")
	}
	p.Inv[0] = Slot{Item: ItemPotato, Count: 1}
	if !w.plant(p, bed) || bed.RegrowAt-w.tick != CropGrowTicks/DungBoost {
		t.Fatalf("crop in a fed bed ripens in %d ticks, want %d", bed.RegrowAt-w.tick, CropGrowTicks/DungBoost)
	}
}
