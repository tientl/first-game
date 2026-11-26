package sim

import "testing"

// A new character has no clothes; each one carried is worn.
func TestClothesAreWornFromTheBag(t *testing.T) {
	w := flatWorld()
	p := mustPlayer(t, w, w.AddPlayer("An"))
	if got := p.Inv.Worn(); got != 0 {
		t.Fatalf("a new player wears %b, want nothing", got)
	}
	p.Inv.Add(ItemBoots, 1)
	p.Inv.Add(ItemTunic, 1)
	if got := p.Inv.Worn(); got != WearTunic|WearBoots {
		t.Errorf("wears %b, want the tunic and the boots", got)
	}
	for i, k := range Clothes() {
		r, ok := RecipeByID(RecipeTunic + RecipeID(i))
		if !ok || r.Gives != k || len(r.Inputs) == 0 || !k.Valid() {
			t.Errorf("%s: recipe %+v", k.Def().Name, r)
		}
	}
}

// A character from a save made before clothes were things keeps the
// outfit it wore: the clothes are put in its bag, once.
func TestOldSavesKeepTheirClothes(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("An")
	w.players[id].Inv.Add(ItemWood, 5)
	w.players[id].Selected = 3 // an empty hotbar slot is in the hand
	data := w.Save()
	data.Version = 17
	w2, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	p := mustPlayer(t, w2, w2.AddPlayer("An"))
	if p.Inv.Worn() != WearAll || p.Inv.Count(ItemWood) != 5 {
		t.Errorf("after loading an old save: wears %b, inventory %v", p.Inv.Worn(), p.Inv)
	}
	for i := 0; i < HotbarSlots; i++ {
		for _, k := range Clothes() {
			if p.Inv[i].Item == k {
				t.Errorf("%s landed on the hotbar", k.Def().Name)
			}
		}
	}
	if fresh := mustPlayer(t, w2, w2.AddPlayer("Khác")); fresh.Inv.Worn() != 0 {
		t.Errorf("a new player in an old world wears %b", fresh.Inv.Worn())
	}

	// Saved again and loaded, nothing more is given: a tunic thrown away stays gone.
	w2.players[p.ID].Inv.Take(ItemTunic, 1)
	w3, err := Load(w2.Save())
	if err != nil {
		t.Fatal(err)
	}
	back := mustPlayer(t, w3, w3.AddPlayer("An"))
	if got := back.Inv.Worn(); got != WearAll&^WearTunic {
		t.Errorf("after a second load: wears %b", got)
	}
}
