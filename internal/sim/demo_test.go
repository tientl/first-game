package sim

import "testing"

func TestTheDemoHasTwoBosses(t *testing.T) {
	SetDemo(true)
	defer SetDemo(false)
	w := genSeas(t, 1, SizeMedium)
	bs := w.Bosses()
	if len(bs) != len(demoBosses) {
		t.Fatalf("%d bosses in the demo, want %d", len(bs), len(demoBosses))
	}
	for _, b := range bs {
		if !bossActive(b.Kind) {
			t.Errorf("%s is not a demo boss", b.Kind.Name())
		}
	}
	for _, c := range w.fauna.creatures {
		if c.Kind.Boss() && !bossActive(c.Kind) {
			t.Errorf("%s raised in the demo", c.Kind.Name())
		}
	}
	// Beating the two wins the game.
	for i, l := range w.bosses.lairs {
		if l.ok {
			l.c.HP = 0
			l.c.gone = true
			w.bosses.lairs[i] = l
		}
	}
	w.stepBosses()
	id := w.AddPlayer("a")
	if show, _, beaten := w.Victory(id); !show || len(beaten) != len(demoBosses) {
		t.Errorf("victory %v with %d beaten, want shown with %d", show, len(beaten), len(demoBosses))
	}
	// The full game still has all six.
	SetDemo(false)
	if n := len(genSeas(t, 1, SizeMedium).Bosses()); n != NumBosses {
		t.Errorf("%d bosses in the full game, want %d", n, NumBosses)
	}
}
