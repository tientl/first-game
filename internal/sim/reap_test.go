package sim

// reap does for player id what walking round with Space would: butchers
// every body and picks up every pile on the ground, so tests of a kill or a
// harvest can count what it gave.
func reap(w *World, id PlayerID) {
	p := w.players[id]
	for _, e := range w.entities {
		if e.Kind == EntityCorpse {
			for e.Kind == EntityCorpse && w.entities[e.ID] != nil {
				w.butcher(p, e)
			}
		}
	}
	for _, d := range w.drops {
		w.pickUp(p, d, false)
	}
	w.stepDrops()
}

// entitiesOf returns every entity of kind k.
func entitiesOf(w *World, k EntityKind) []*Entity {
	var out []*Entity
	for _, e := range w.entities {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}
