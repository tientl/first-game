package sim

import "testing"

func addBig(w *World, kind EntityKind, tx, ty int) *Entity {
	e := w.addEntity(kind, tx, ty)
	e.Big, e.HP = true, maxHP(kind, true)
	return e
}

func TestBigTreeTakesMoreHitsAndGivesMore(t *testing.T) {
	w := flatWorld()
	tree := addBig(w, EntityTree, 17, 16)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemAxe, 1)
	w.SetInput(id, Input{Action: true})

	// As many hits as fell a small tree leave a big one standing.
	stepN(w, 1+(TreeHits-1)*ActionCooldownTicks)
	if !tree.Ready || tree.HP != (BigTreeHits-TreeHits)*HitPower || w.players[id].Inv.Count(ItemWood) != 0 {
		t.Fatalf("after %d hits: ready=%v hp=%d wood=%d, want standing with %d hp and no wood",
			TreeHits, tree.Ready, tree.HP, w.players[id].Inv.Count(ItemWood), (BigTreeHits-TreeHits)*HitPower)
	}
	stepN(w, (BigTreeHits-TreeHits)*ActionCooldownTicks)
	reap(w, id)
	if p := mustPlayer(t, w, id); tree.Ready || p.Inv.Count(ItemWood) != BigTreeYield {
		t.Fatalf("after %d hits: ready=%v wood=%d, want a stump and %d wood", BigTreeHits, tree.Ready, p.Inv.Count(ItemWood), BigTreeYield)
	}

	w.RemovePlayer(id)
	stepN(w, TreeRegrowTicks)
	if !tree.Ready || !tree.Big || tree.HP != maxHP(EntityTree, true) {
		t.Fatalf("regrown tree: ready=%v big=%v hp=%d, want a big tree with %d hp", tree.Ready, tree.Big, tree.HP, maxHP(EntityTree, true))
	}
}

func TestBigBoulderTakesMoreHitsAndGivesMore(t *testing.T) {
	w := flatWorld()
	addBig(w, EntityBoulder, 16, 17)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemPickaxe, 1)
	w.SetInput(id, Input{Action: true})

	stepN(w, 1+(BigBoulderHits-2)*ActionCooldownTicks)
	if _, ok := w.EntityAt(16, 17); !ok || w.players[id].Inv.Count(ItemStone) != 0 {
		t.Fatalf("one hit short: boulder there=%v stone=%d, want it still there and no stone", ok, w.players[id].Inv.Count(ItemStone))
	}
	stepN(w, ActionCooldownTicks)
	reap(w, id)
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemStone) != BigBoulderYield {
		t.Fatalf("stone = %d, want %d", p.Inv.Count(ItemStone), BigBoulderYield)
	}
	if _, ok := w.EntityAt(16, 17); ok {
		t.Fatal("big boulder still on its tile")
	}
}

func TestBigGivesMoreForEveryHit(t *testing.T) {
	if BigTreeHits <= TreeHits || BigBoulderHits <= BoulderHits {
		t.Fatal("big trees and boulders must take more hits than small ones")
	}
	if BigTreeYield*TreeHits < TreeYield*BigTreeHits || BigBoulderYield*BoulderHits < BoulderYield*BigBoulderHits {
		t.Fatal("a big tree or boulder gives less per hit than a small one")
	}
}

func TestGeneratedWorldsHaveBothSizes(t *testing.T) {
	for _, seed := range testSeeds {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatal(err)
		}
		var count [NumEntityKinds][2]int
		for _, e := range w.entities {
			if e.Big && !e.Kind.Sized() {
				t.Fatalf("seed %d: a big %s", seed, e.Kind.Name())
			}
			if e.HP != maxHP(e.Kind, e.Big) {
				t.Fatalf("seed %d: %s (big=%v) has %d hp, want %d", seed, e.Kind.Name(), e.Big, e.HP, maxHP(e.Kind, e.Big))
			}
			count[e.Kind][sizeOf(e.Big)]++
		}
		for _, k := range []EntityKind{EntityTree, EntityBoulder} {
			small, big := count[k][0], count[k][1]
			if small == 0 || big == 0 || big > small {
				t.Errorf("seed %d: %d small and %d big %s, want both, mostly small", seed, small, big, k.Name())
			}
		}
	}
}

func sizeOf(big bool) int {
	if big {
		return 1
	}
	return 0
}

func TestLoadingAnOlderSaveGivesTreesAndBouldersASize(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	// Write the save the way version 2 did: one size, trees with 3 hits and
	// boulders with 4. One big tree was hit once already.
	d := w.Save()
	d.Version = 2
	var hitTree EntityID
	for i := range d.Entities {
		e := &d.Entities[i]
		switch e.Kind {
		case EntityTree:
			e.HP = 3
			if e.Big && hitTree == 0 {
				hitTree, e.HP = e.ID, 2
			}
		case EntityBoulder:
			e.HP = 4
		}
		e.Big = false
	}
	if hitTree == 0 {
		t.Fatal("no big tree in the test world")
	}

	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	for id, e := range w.entities {
		got := w2.entities[id]
		if got == nil {
			t.Fatalf("%s %d lost on load", e.Kind.Name(), id)
		}
		want := maxHP(e.Kind, e.Big)
		if id == hitTree {
			want = (2*maxHP(EntityTree, true) + 2) / 3 // 2 of 3 hits left, rounded up
		}
		if got.Big != e.Big || got.HP != want {
			t.Fatalf("%s %d: big=%v hp=%d, want big=%v hp=%d", e.Kind.Name(), id, got.Big, got.HP, e.Big, want)
		}
	}
}

func TestLoadKeepsBigOnlyOnSizedKinds(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 3, Width: 48, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	for i := range d.Entities {
		d.Entities[i].Big = true
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range w2.entities {
		if e.Big != e.Kind.Sized() {
			t.Fatalf("%s loaded with big=%v", e.Kind.Name(), e.Big)
		}
	}
}
