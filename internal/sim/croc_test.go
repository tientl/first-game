package sim

import (
	"math"
	"testing"
)

// marshWorld is flatWorld turned to marsh, with a pond from column 18 east.
func marshWorld(pond bool) *World {
	w := flatWorld()
	for i := range w.tiles {
		w.tiles[i] = TileMarsh
		if pond && i%w.width >= 18 {
			w.tiles[i] = TileWater
		}
	}
	return w
}

func TestCrocLurksOutOfReachThenAmbushes(t *testing.T) {
	w := marshWorld(true)
	croc := w.addCreature(CreatureCroc, 18.5, 16.5)
	croc.State = CreatureLurk
	id := w.AddPlayer("a")
	w.players[id].X = 14.5 // four tiles off: safe
	stepN(w, TicksPerSecond)
	if croc.State != CreatureLurk {
		t.Fatalf("state %d, want it still lurking", croc.State)
	}
	if cs := w.creatureCandidates(croc.X, croc.Y); len(cs) != 0 {
		t.Fatal("a lurking crocodile is a target")
	}

	w.players[id].X = 16.2 // too near the water
	for range CrocLungeTicks + 2 {
		if w.Step(); mustPlayer(t, w, id).Notice == NoticeBitten {
			break
		}
	}
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeBitten || CreatureKind(p.NoticeArg) != CreatureCroc {
		t.Fatalf("notice %v/%d, want a crocodile bite", p.Notice, p.NoticeArg)
	}
	if p.Health > MaxHealth-CrocBiteDamage+1 {
		t.Fatalf("health %.1f, want a bite's worth gone", p.Health)
	}
	stepN(w, 3*TicksPerSecond)
	if croc.State != CreatureLurk {
		t.Fatalf("state %d at x %.1f, want it back in the water", croc.State, croc.X)
	}
}

func TestOutrunningACrocOnLand(t *testing.T) {
	w := marshWorld(false)
	croc := w.addCreature(CreatureCroc, 16.5, 16.5)
	id := w.AddPlayer("a")
	w.players[id].X = 14.5
	w.Step()
	if croc.State != CreatureChase {
		t.Fatalf("state %d, want it snapping at the player", croc.State)
	}
	w.SetInput(id, Input{MoveX: -1}) // through the mud: barely faster than the crocodile
	stepN(w, CrocChaseTicks+TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health != MaxHealth {
		t.Fatalf("health %.1f: bitten while running away on land", p.Health)
	}
	if croc.State == CreatureChase {
		t.Fatal("still chasing a player far off")
	}
}

func TestKillingACroc(t *testing.T) {
	w := marshWorld(false) // no water to slip into
	croc := w.addCreature(CreatureCroc, 17.6, 16.5)
	id := w.AddPlayer("a")
	w.SetInput(id, Input{Action: true})
	for range 10 * TicksPerSecond {
		if w.Step(); croc.gone || len(creatureByKind(w, CreatureCroc)) == 0 {
			break
		}
	}
	reap(w, id)
	p := mustPlayer(t, w, id)
	if len(creatureByKind(w, CreatureCroc)) != 0 || p.Inv.Count(ItemMeat) != 3 {
		t.Fatalf("crocs %d, meat %d: want it killed for 3 meat", len(creatureByKind(w, CreatureCroc)), p.Inv.Count(ItemMeat))
	}
	if p.Dead {
		t.Fatal("died fighting one crocodile on land")
	}
}

func TestCrocsKeepToTheMarsh(t *testing.T) {
	for _, seed := range []uint64{1, 7} {
		w, err := Generate(GenConfig{Seed: seed, Width: 256, Height: 256, Style: StyleOpen})
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range w.fauna.creatures {
			if c.Kind != CreatureCroc {
				continue
			}
			n++
			tx, ty := int(math.Floor(c.X)), int(math.Floor(c.Y))
			if w.TileAt(tx, ty) != TileMarsh || !w.waterside(tx, ty) {
				t.Errorf("seed %d: crocodile %d starts on %v at (%d, %d), not marsh by the water", seed, c.ID, w.TileAt(tx, ty), tx, ty)
			}
		}
		if n < creatureMinPerKind {
			t.Errorf("seed %d: %d crocodiles in an open world", seed, n)
		}
	}
}
