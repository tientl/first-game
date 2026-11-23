package sim

import (
	"math"
	"testing"
)

func TestCrabsLiveOnTheBeach(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3, 42} {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range w.fauna.creatures {
			if c.Kind != CreatureCrab {
				continue
			}
			n++
			if tx, ty := int(math.Floor(c.X)), int(math.Floor(c.Y)); !w.seaside(tx, ty) {
				t.Errorf("seed %d: crab %d starts on %v at (%d, %d), not on sand by the water", seed, c.ID, w.TileAt(tx, ty), tx, ty)
			}
		}
		if n < creatureMinPerKind {
			t.Errorf("seed %d: %d crabs", seed, n)
		}
	}
}

func TestCrabGoesUnderTheWater(t *testing.T) {
	w := shoreWorld() // the sea from column 18
	crab := w.addCreature(CreatureCrab, 17.4, 16.5)
	id := w.AddPlayer("a") // at 16.5: a tile away, cutting it off from nothing
	w.players[id].X = 15.5
	stepN(w, TicksPerSecond)
	if crab.State != CreatureHidden || !w.inShallows(crab) {
		t.Fatalf("state %d at x %.2f, want the crab under the water", crab.State, crab.X)
	}
	if cs := w.creatureCandidates(crab.X, crab.Y); len(cs) != 0 {
		t.Fatal("a crab under the water is a target")
	}
	stepN(w, CrabHideTicks+TicksPerSecond)
	if crab.State != CreatureHidden {
		t.Fatal("came up with a player on the beach right there")
	}
	w.players[id].X = 4.5
	stepN(w, 2*TicksPerSecond)
	if crab.State == CreatureHidden {
		t.Fatal("still under with nobody about")
	}
}

func TestCrabCutOffFromTheWaterPinches(t *testing.T) {
	w := flatWorld() // no water anywhere
	crab := w.addCreature(CreatureCrab, 18, 16.5)
	id := w.AddPlayer("a") // at 16.5: 1.5 tiles off
	w.Step()
	if crab.State != CreatureChase {
		t.Fatalf("state %d, want the crab standing its ground", crab.State)
	}
	w.players[id].X = 17.3 // too close
	w.Step()
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeBitten || CreatureKind(p.NoticeArg) != CreatureCrab {
		t.Fatalf("notice %v/%d, want a pinch", p.Notice, p.NoticeArg)
	}
	if p.Health > MaxHealth-CrabPinchDamage+0.1 {
		t.Fatalf("health %.1f, want a pinch's worth gone", p.Health)
	}

	w.SetInput(id, Input{Action: true})
	w.Step()
	reap(w, id)
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemCrabMeat) != 1 {
		t.Fatalf("crab meat %d, want the crab caught", p.Inv.Count(ItemCrabMeat))
	}
}

func TestCrabIsCookedAtTheFire(t *testing.T) {
	r, ok := RecipeByID(RecipeCookCrab)
	if !ok || !r.NearFire || r.Gives != ItemCookedCrab || r.Inputs[0].Item != ItemCrabMeat {
		t.Fatalf("recipe %+v", r)
	}
	raw, cooked := ItemCrabMeat.Def(), ItemCookedCrab.Def()
	if cooked.Hunger <= raw.Hunger || cooked.Health <= raw.Health {
		t.Fatal("cooked crab should beat raw")
	}
}
