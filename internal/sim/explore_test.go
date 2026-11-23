package sim

import "testing"

func TestRevealMarksACircle(t *testing.T) {
	e := NewExplored(64, 48)
	if !e.Reveal(20.5, 20.5) {
		t.Fatal("first reveal changed nothing")
	}
	if !e.Has(20, 20) || !e.Has(20+RevealRadius-1, 20) || !e.Has(20, 20-RevealRadius+1) {
		t.Error("tiles inside the radius are not seen")
	}
	if e.Has(20+RevealRadius+1, 20) || e.Has(20+RevealRadius-1, 20+RevealRadius-1) {
		t.Error("tiles outside the radius (or the circle's corner) are seen")
	}
	if e.Reveal(20.9, 20.1) {
		t.Error("moving within the same tile revealed something")
	}
	before := e.Count()
	if !e.Reveal(21.5, 20.5) || e.Count() <= before {
		t.Error("stepping onto the next tile revealed nothing new")
	}
}

func TestRevealAtTheEdgeStaysOnTheMap(t *testing.T) {
	e := NewExplored(10, 7) // 70 bits: the last byte is part padding
	e.Reveal(0.5, 0.5)
	e.Reveal(9.5, 6.5)
	if e.Has(-1, 0) || e.Has(10, 6) || e.Has(0, 7) {
		t.Error("off-map tiles reported as seen")
	}
	if got := e.Count(); got > 70 {
		t.Errorf("count %d is more than the map has", got)
	}
}

func TestExploredRoundTrip(t *testing.T) {
	e := NewExplored(33, 17)
	e.Reveal(5, 5)
	e.Reveal(30, 15)
	back := ExploredFrom(33, 17, e.Bytes())
	for ty := range 17 {
		for tx := range 33 {
			if back.Has(tx, ty) != e.Has(tx, ty) {
				t.Fatalf("tile %d,%d differs after a round trip", tx, ty)
			}
		}
	}
	if ExploredFrom(33, 17, []byte{1, 2, 3}).Count() != 0 {
		t.Error("data for another map size was accepted")
	}
	junk := make([]byte, len(e.Bytes()))
	for i := range junk {
		junk[i] = 0xff
	}
	if got := ExploredFrom(33, 17, junk).Count(); got != 33*17 {
		t.Errorf("all-ones data counts %d tiles, want the map's %d", got, 33*17)
	}
	var none *Explored
	if none.Has(0, 0) || none.Reveal(1, 1) || none.Bytes() != nil {
		t.Error("a nil set must be empty and inert")
	}
}

func TestPlayersMapWhatTheyWalkPast(t *testing.T) {
	w := flatWorld() // 32x32, spawn in the middle
	id := w.AddPlayer("a")
	p := w.players[id]
	if !p.explored.Has(16, 16) || p.explored.Has(31, 16) {
		t.Fatal("on joining, want the spawn seen and the far edge not")
	}
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 30) // six tiles east
	if !p.explored.Has(31, 16) {
		t.Fatalf("at x=%.1f the east edge is still unseen", p.X)
	}
	if got := ExploredFrom(32, 32, w.Explored(id)); !got.Has(31, 16) || got.Count() != p.explored.Count() {
		t.Error("World.Explored does not hand out what the player has seen")
	}
	if w.Explored(id+99) != nil {
		t.Error("an unknown player has an explored set")
	}
}

func TestExploredIsSavedWithThePlayer(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 48, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("Tiến")
	far := w.players[id]
	far.explored.Reveal(3.5, 3.5) // as if they had been to the corner
	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	back := w2.players[w2.AddPlayer("Tiến")]
	if !back.explored.Has(3, 3) || back.explored.Count() != far.explored.Count() {
		t.Errorf("rejoined with %d tiles seen, want the %d saved, corner included", back.explored.Count(), far.explored.Count())
	}
	if fresh := w2.players[w2.AddPlayer("Khác")]; fresh.explored.Has(3, 3) {
		t.Error("a new player starts out knowing someone else's map")
	}
}
