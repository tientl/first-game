package sim

import "testing"

func TestFruitIsPickedByHandAndGrowsBack(t *testing.T) {
	for _, f := range fruitTrees {
		w := flatWorld()
		tree := w.addEntity(f.kind, 17, 16)
		id := w.AddPlayer("a")
		w.players[id].Inv.Add(ItemAxe, 1) // an axe in the bag changes nothing: fruit trees are not felled
		w.SetInput(id, Input{Action: true})
		w.Step()
		p := mustPlayer(t, w, id)
		if p.Inv.Count(f.fruit) != f.yield || tree.Ready || p.Inv.Count(ItemWood) != 0 {
			t.Fatalf("%s: picked %d %s (want %d), ripe %v, wood %d", f.kind.Name(), p.Inv.Count(f.fruit), f.fruit.Def().Name, f.yield, tree.Ready, p.Inv.Count(ItemWood))
		}
		if Interactable(tree.Kind, tree.Ready, true) {
			t.Errorf("%s: a picked tree is still a target", f.kind.Name())
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
