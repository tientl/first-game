package sim

import "testing"

func TestDifficultyScalesHarmHungerAndCold(t *testing.T) {
	took := func(d Difficulty) (bite, hunger, cold float64) {
		w := flatWorld()
		w.difficulty = d
		w.tick = winterNightTick
		id := w.AddPlayer("a")
		p := w.players[id]
		p.Inv = Inventory{}
		struck(p, 10)
		bite = MaxHealth - p.Health
		p.Health, p.Hunger = MaxHealth, 50
		w.Step()
		hunger = 50 - p.Hunger
		cold = MaxHealth - p.Health
		return
	}
	bb, bh, bc := took(DiffBasic)
	sb, sh, sc := took(DiffSkilled)
	lb, lh, lc := took(DiffLord)
	if !(bb < sb && sb < lb) || !(bh < sh && sh < lh) || !(bc < sc && sc < lc) {
		t.Fatalf("basic/skilled/lord: bite %.1f/%.1f/%.1f hunger %.4f/%.4f/%.4f cold %.3f/%.3f/%.3f", bb, sb, lb, bh, sh, lh, bc, sc, lc)
	}
	if bb != 10 {
		t.Errorf("basic: a blow of 10 took %.1f, want it as it is (×1)", bb)
	}
}

func TestDifficultyIsSavedAndOldSavesAreBasic(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 3, Width: 64, Height: 64, Difficulty: DiffLord})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	back, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if back.Difficulty() != DiffLord {
		t.Fatalf("loaded difficulty %v, want lord", back.Difficulty())
	}
	d.Difficulty = 0 // a save from before difficulty existed
	if old, _ := Load(d); old.Difficulty() != DiffBasic {
		t.Fatalf("an old save loads as %v, want basic", old.Difficulty())
	}
}

func TestDyingKeepsItemsOnlyAtBasic(t *testing.T) {
	die := func(d Difficulty) (*World, PlayerID) {
		w := flatWorld()
		w.difficulty = d
		id := w.AddPlayer("a")
		p := w.players[id]
		p.Inv = Inventory{}
		p.Inv.Add(ItemSpear, 1)
		p.Inv.Add(ItemMeat, 5)
		p.X, p.Y = 10.5, 10.5
		hurt(p, MaxHealth)
		w.Apply(id, Command{Kind: CmdRespawn})
		return w, id
	}
	w, id := die(DiffBasic)
	if p := w.players[id]; p.Inv.Count(ItemSpear) != 1 || p.Inv.Count(ItemMeat) != 5 {
		t.Fatalf("basic: came back with %d spear, %d meat; want everything", p.Inv.Count(ItemSpear), p.Inv.Count(ItemMeat))
	}
	for _, e := range w.entities {
		if e.Kind == EntityGrave {
			t.Fatal("basic: a body was left behind")
		}
	}

	w, id = die(DiffSkilled)
	p := w.players[id]
	if p.Inv.Count(ItemSpear) != 0 || p.Inv.Count(ItemMeat) != 0 {
		t.Fatal("skilled: came back still carrying things")
	}
	var body *Entity
	for _, e := range w.entities {
		if e.Kind == EntityGrave {
			body = e
		}
	}
	if body == nil || body.TX != 10 || body.TY != 10 || w.BodyItems(body.ID) != 2 {
		t.Fatalf("skilled: body %+v holding %d stacks, want one where the player fell holding both", body, w.BodyItems(body.ID))
	}
	// Going back for them.
	p.X, p.Y, p.Dir = 10.5, 11.5, DirUp
	if !w.lootBody(p, body) || p.Inv.Count(ItemSpear) != 1 || p.Inv.Count(ItemMeat) != 5 {
		t.Fatalf("after looting: %d spear, %d meat", p.Inv.Count(ItemSpear), p.Inv.Count(ItemMeat))
	}
	if w.entities[body.ID] != nil {
		t.Fatal("an empty body is still there")
	}
}

func TestABodyIsSaved(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 2, Width: 64, Height: 64, Difficulty: DiffLord})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	p.Inv.Add(ItemStone, 3)
	hurt(p, MaxHealth)
	w.Apply(id, Command{Kind: CmdRespawn})
	var body EntityID
	for _, e := range w.entities {
		if e.Kind == EntityGrave {
			body = e.ID
		}
	}
	if body == 0 {
		t.Fatal("no body left behind")
	}
	back, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	if back.BodyItems(body) != 1 {
		t.Fatalf("after a save the body holds %d stacks, want 1", back.BodyItems(body))
	}
}
