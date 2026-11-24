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
	nb, nh, nc := took(DiffNormal)
	eb, eh, ec := took(DiffEasy)
	hb, hh, hc := took(DiffHard)
	if !(eb < nb && nb < hb) || !(eh < nh && nh < hh) || !(ec < nc && nc < hc) {
		t.Fatalf("easy/normal/hard: bite %.1f/%.1f/%.1f hunger %.4f/%.4f/%.4f cold %.3f/%.3f/%.3f", eb, nb, hb, eh, nh, hh, ec, nc, hc)
	}
}

func TestDifficultyIsSavedAndOldSavesAreNormal(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 3, Width: 64, Height: 64, Difficulty: DiffHard})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	back, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if back.Difficulty() != DiffHard {
		t.Fatalf("loaded difficulty %v, want hard", back.Difficulty())
	}
	d.Difficulty = 0 // a save from before difficulty existed
	if old, _ := Load(d); old.Difficulty() != DiffNormal {
		t.Fatalf("an old save loads as %v, want normal", old.Difficulty())
	}
}
