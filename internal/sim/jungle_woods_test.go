package sim

import "testing"

func TestTheWoodsHaveGiantsHivesBananasAndMonkeys(t *testing.T) {
	w := genSeas(t, 1, SizeMedium)
	var giants, hives, bananas, monkeys int
	for _, e := range w.entities {
		switch e.Kind {
		case EntityGiantTree:
			giants++
			if !w.blocked(e.TX, e.TY) || !w.blocked(e.TX-1, e.TY-1) || !w.blocked(e.TX+1, e.TY) {
				t.Errorf("giant at %d,%d: its foot does not block the way", e.TX, e.TY)
			}
		case EntityBananaTree:
			bananas++
		}
		if e.HasHive() {
			hives++
		}
	}
	for _, c := range w.fauna.creatures {
		if c.Kind == CreatureMonkey {
			monkeys++
		}
	}
	t.Logf("giants %d, hives %d, banana trees %d, monkeys %d", giants, hives, bananas, monkeys)
	if giants == 0 || hives == 0 || bananas == 0 || monkeys == 0 {
		t.Fatal("the woods are missing something")
	}
}

func TestRobbingAHive(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a") // on (16, 16)
	p := w.players[id]
	p.Dir = DirRight
	tree := w.addEntity(EntityTree, 17, 16)
	tree.Var = hiveVar
	health := p.Health

	w.SetInput(id, Input{Action: true})
	w.Step()
	w.SetInput(id, Input{})
	honey := 0
	for _, d := range w.Drops() {
		if d.Item == ItemHoney {
			honey += d.Count
		}
	}
	if honey != HoneyYield || !tree.Ready {
		t.Fatalf("honey dropped %d, tree standing %v: want %d honey and the tree left alone", honey, tree.Ready, HoneyYield)
	}
	if !p.Swarmed() {
		t.Fatal("no bees after the thief")
	}
	stepN(w, 3*BeeStingCD)
	if p.Health >= health {
		t.Fatal("the bees did not sting")
	}
	if tree.HoneyReady(w.tick) {
		t.Fatal("the hive filled again at once")
	}

	// Running far enough leaves the swarm behind.
	p.X += BeeRange + 2
	w.Step()
	if p.Swarmed() {
		t.Fatal("the bees followed a thief far away")
	}

	// With a burning torch in the hand, the smoke keeps them calm.
	p.X -= BeeRange + 2
	tree.HoneyAt = w.tick
	holding(w, id, ItemTorch)
	stepN(w, TicksPerSecond)
	w.SetInput(id, Input{Action: true})
	w.Step()
	if p.Swarmed() || tree.HoneyReady(w.tick) {
		t.Fatalf("with a torch: swarmed %v, honey still there %v", p.Swarmed(), tree.HoneyReady(w.tick))
	}
}

func TestFellingAGiant(t *testing.T) {
	w := flatWorld()
	for i := range w.tiles {
		w.tiles[i] = TileForest
	}
	g := w.raiseGiant(17, 14)
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X, p.Y, p.Dir = 17.5, 15.5, DirUp
	holding(w, id, ItemAxe)
	w.SetInput(id, Input{Action: true})
	stepN(w, GiantTreeHits*2*TicksPerSecond)
	if g.Ready {
		t.Fatalf("the giant still stands after %d seconds of chopping (HP %d)", GiantTreeHits*2, g.HP)
	}
	wood := 0
	for _, d := range w.Drops() {
		if d.Item == ItemWood {
			wood += d.Count
		}
	}
	wood += p.Inv.Count(ItemWood)
	if wood < GiantWood {
		t.Errorf("a felled giant gave %d wood, want %d", wood, GiantWood)
	}
}

func TestMonkeysAreTamedWithBananas(t *testing.T) {
	if CreatureMonkey.TameFood() != ItemBanana {
		t.Fatalf("monkeys are tamed with %v", CreatureMonkey.TameFood())
	}
}
