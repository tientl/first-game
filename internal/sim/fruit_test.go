package sim

import "testing"

func TestFruitIsPickedByHandAndGrowsBack(t *testing.T) {
	for _, f := range fruitTrees {
		w := flatWorld()
		tree := w.addEntity(f.kind, 17, 16)
		id := w.AddPlayer("a")
		w.players[id].Inv.Add(ItemAxe, 1) // ripe fruit is picked by hand, axe or not
		w.SetInput(id, Input{Action: true})
		w.Step()
		p := mustPlayer(t, w, id)
		if p.Inv.Count(f.fruit) != f.yield || tree.Ready || tree.HP != maxHP(f.kind, false) || len(w.drops) != 0 {
			t.Fatalf("%s: picked %d %s (want %d), ripe %v, hp %d, %d piles: want the fruit picked and the trunk whole",
				f.kind.Name(), p.Inv.Count(f.fruit), f.fruit.Def().Name, f.yield, tree.Ready, tree.HP, len(w.drops))
		}
		// Space still held: the hold that picked the fruit does not go on to fell the tree.
		stepN(w, 3*f.hits*ActionCooldownTicks)
		if tree.HP != maxHP(f.kind, false) || tree.FruitStump() || len(w.drops) != 0 {
			t.Fatalf("%s: hp %d, stump %v after holding Space on: want it untouched", f.kind.Name(), tree.HP, tree.FruitStump())
		}
		w.RemovePlayer(id)
		stepN(w, int(f.regrow))
		if !tree.Ready {
			t.Errorf("%s: no fruit after %d ticks", f.kind.Name(), f.regrow)
		}
		if !f.fruit.Edible() {
			t.Errorf("%s cannot be eaten", f.fruit.Def().Name)
		}
	}
}

func TestFruitTreeIsFelledOncePicked(t *testing.T) {
	for _, f := range fruitTrees {
		w := flatWorld()
		tree := w.addEntity(f.kind, 17, 16)
		id := w.AddPlayer("a")
		w.players[id].Inv.Add(ItemAxe, 1)
		w.SetInput(id, Input{Action: true})
		w.Step() // picks the fruit
		w.SetInput(id, Input{})
		stepN(w, ActionCooldownTicks)
		w.SetInput(id, Input{Action: true}) // a fresh press: the axe
		if f.hits > 1 {
			stepN(w, 1+(f.hits-2)*ActionCooldownTicks)
			if tree.FruitStump() || tree.HP >= maxHP(f.kind, false) {
				t.Fatalf("%s: stump %v, hp %d after %d blows: want it standing, hurt", f.kind.Name(), tree.FruitStump(), tree.HP, f.hits-1)
			}
			stepN(w, ActionCooldownTicks)
		} else {
			w.Step()
		}
		if !tree.FruitStump() || tree.Ready || w.entities[tree.ID] != tree || tree.Kind != f.kind {
			t.Fatalf("%s: stump %v, ripe %v after %d blows of a stone axe: want a stump", f.kind.Name(), tree.FruitStump(), tree.Ready, f.hits)
		}
		if w.players[id].Inv.Count(ItemWood) != 0 || len(w.drops) == 0 {
			t.Fatalf("%s: the wood went straight into the bag; want it on the ground", f.kind.Name())
		}
		// A stump is not a target: holding Space does nothing more.
		if e, _ := w.findTarget(w.players[id]); e != nil {
			t.Fatalf("%s: the stump is still a target", f.kind.Name())
		}
		reap(w, id)
		p := mustPlayer(t, w, id)
		if p.Inv.Count(ItemWood) != f.wood || p.Inv.Count(f.fruit) != f.yield {
			t.Fatalf("%s: wood %d (want %d), %s %d (want the %d picked)", f.kind.Name(), p.Inv.Count(ItemWood), f.wood, f.fruit.Def().Name, p.Inv.Count(f.fruit), f.yield)
		}

		w.RemovePlayer(id)
		stepN(w, FruitStumpRegrowTicks)
		if tree.FruitStump() || tree.Ready || tree.Kind != f.kind || tree.HP != maxHP(f.kind, false) {
			t.Fatalf("%s after %d ticks: stump %v, ripe %v, kind %s: want the same tree back, bare", f.kind.Name(), FruitStumpRegrowTicks, tree.FruitStump(), tree.Ready, tree.Kind.Name())
		}
		stepN(w, int(f.regrow))
		if !tree.Ready {
			t.Errorf("%s: no fruit %d ticks after it grew back", f.kind.Name(), f.regrow)
		}
	}
}

func TestBareFruitTreeNeedsAnAxe(t *testing.T) {
	w := flatWorld()
	tree := w.addEntity(EntityFigTree, 17, 16)
	tree.Ready, tree.RegrowAt = false, w.tick+DayTicks // picked bare
	id := w.AddPlayer("a")
	w.SetInput(id, Input{Action: true})
	stepN(w, 1+4*ActionCooldownTicks)

	p := mustPlayer(t, w, id)
	if tree.HP != maxHP(EntityFigTree, false) || tree.FruitStump() || len(w.drops) != 0 {
		t.Fatalf("hp %d, stump %v: chopped without an axe", tree.HP, tree.FruitStump())
	}
	if p.Notice != NoticeNeedTool || ItemKind(p.NoticeArg) != ItemAxe {
		t.Fatalf("notice = %v/%d, want need-tool for the axe", p.Notice, p.NoticeArg)
	}
	if p.Acting(w.Tick()) {
		t.Error("a refused hit shows the working pose")
	}
	if got, _ := w.findTarget(w.players[id]); got != tree {
		t.Errorf("target = %v, want the bare tree, so the client can say why nothing happens", got)
	}
}

func TestFruitStumpIsSaved(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	var stump, bare *Entity
	for _, e := range w.entities {
		switch {
		case !IsFruitTree(e.Kind):
		case stump == nil:
			stump = e
		case bare == nil:
			bare = e
		}
	}
	if bare == nil {
		t.Fatal("not two fruit trees in the test world")
	}
	f, _ := fruitOf(stump.Kind)
	w.fellFruit(stump, f, maxHP(stump.Kind, false))
	bare.Ready, bare.RegrowAt, bare.HP = false, w.tick+DayTicks, HitPower // as a save from before felling held it

	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	s2, b2 := w2.entities[stump.ID], w2.entities[bare.ID]
	if s2 == nil || !s2.FruitStump() || s2.Ready || s2.Kind != stump.Kind || s2.RegrowAt != stump.RegrowAt {
		t.Fatalf("loaded %+v, want the stump %+v", s2, stump)
	}
	if b2 == nil || b2.FruitStump() || b2.HP != maxHP(bare.Kind, false) {
		t.Fatalf("loaded %+v: want a standing tree with a whole trunk", b2)
	}
	stepN(w2, FruitStumpRegrowTicks)
	if s2.FruitStump() || s2.Kind != stump.Kind {
		t.Errorf("loaded stump after %d ticks: %+v, want the tree back", FruitStumpRegrowTicks, s2)
	}
}

func TestGinkgoNutsAreBetterRoasted(t *testing.T) {
	raw, roast := ItemGinkgoNut.Def(), ItemRoastGinkgo.Def()
	if raw.Health >= 0 || roast.Hunger <= raw.Hunger || roast.Health <= 0 {
		t.Fatalf("raw %+v, roasted %+v: want raw nuts to hurt and roasted ones to feed and heal", raw, roast)
	}
	w := flatWorld()
	w.addEntity(EntityCampfire, 17, 16)
	p := w.players[w.AddPlayer("a")]
	p.Inv.Add(ItemGinkgoNut, 2)
	w.Apply(p.ID, Command{Kind: CmdCraft, Recipe: RecipeRoastGinkgo})
	if p.Inv.Count(ItemRoastGinkgo) != 1 || p.Inv.Count(ItemGinkgoNut) != 1 {
		t.Fatalf("by the fire: %d roasted, %d raw", p.Inv.Count(ItemRoastGinkgo), p.Inv.Count(ItemGinkgoNut))
	}
}

func TestFruitTreesGrowWhereTheyBelong(t *testing.T) {
	for _, style := range []WorldStyle{StyleIsland, StyleOpen} {
		w, err := Generate(GenConfig{Seed: 42, Width: 256, Height: 256, Style: style})
		if err != nil {
			t.Fatal(err)
		}
		count := map[EntityKind]int{}
		for _, e := range w.entities {
			f, ok := fruitOf(e.Kind)
			if !ok {
				continue
			}
			count[e.Kind]++
			if f.share[w.TileAt(e.TX, e.TY)] == 0 {
				t.Fatalf("style %d: %s on tile %d", style, e.Kind.Name(), w.TileAt(e.TX, e.TY))
			}
		}
		for _, f := range fruitTrees {
			if count[f.kind] < 5 {
				t.Errorf("style %d: only %d %s", style, count[f.kind], f.kind.Name())
			}
		}
	}
}

func TestOlderSavesGetFruitTrees(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	trees := 0
	d := w.Save()
	d.Version = 8
	kept := d.Entities[:0]
	for _, e := range d.Entities {
		if IsFruitTree(e.Kind) { // a version 8 world had none
			trees++
			continue
		}
		kept = append(kept, e)
	}
	d.Entities = kept
	if trees == 0 {
		t.Fatal("no fruit trees in the test world")
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for _, e := range w2.entities {
		if IsFruitTree(e.Kind) {
			got++
		}
	}
	if got != trees {
		t.Errorf("loaded %d fruit trees, want the %d the world grows", got, trees)
	}
}
