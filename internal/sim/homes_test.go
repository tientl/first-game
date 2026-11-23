package sim

import (
	"math"
	"testing"
)

func TestAnimalsLiveInHomes(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3, 42} {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatal(err)
		}
		homes := map[EntityKind]int{}
		w.EachEntityIn(0, 0, w.width, w.height, func(e Entity) { homes[e.Kind]++ })
		if homes[EntityBurrow] == 0 || homes[EntityDen] == 0 {
			t.Fatalf("seed %d: %d burrows, %d dens, want some of each", seed, homes[EntityBurrow], homes[EntityDen])
		}
		for _, c := range w.fauna.creatures {
			want := c.Kind.def().homeKind
			if want == EntityNone {
				continue
			}
			e := w.home(c)
			if e == nil {
				t.Errorf("seed %d: %s %d has no %s", seed, c.Kind.Name(), c.ID, want.Name())
				continue
			}
			if hx, hy := homeCentre(e); math.Hypot(c.X-hx, c.Y-hy) > homeSpread+0.01 {
				t.Errorf("seed %d: %s %d starts %.1f tiles from its home", seed, c.Kind.Name(), c.ID, math.Hypot(c.X-hx, c.Y-hy))
			}
		}
	}
}

// burrowWorld is flatWorld with a burrow two tiles east of the spawn and a
// rabbit living in it, standing a tile in front of the player's way.
func burrowWorld() (*World, *Entity, *Creature) {
	w := flatWorld()
	b := w.addEntity(EntityBurrow, 20, 16)
	w.fauna.homes = append(w.fauna.homes, homeRec{kind: EntityBurrow, tx: 20, ty: 16, id: b.ID})
	r := w.addCreature(CreatureRabbit, 18.5, 16.5)
	r.home, r.homeX, r.homeY = b.ID, 20.5, 16.5
	return w, b, r
}

func TestRabbitBoltsDownItsBurrow(t *testing.T) {
	w, _, r := burrowWorld()
	id := w.AddPlayer("a") // at 16.5: the burrow is ahead of the rabbit, away from the player
	stepN(w, RabbitFleeTicks)
	if r.State != CreatureHidden || r.X != 20.5 {
		t.Fatalf("state %d at x %.2f, want the rabbit down its burrow", r.State, r.X)
	}
	if cs := w.creatureCandidates(20.5, 16.5); len(cs) != 0 {
		t.Fatalf("a hidden rabbit is a target: %+v", cs)
	}
	w.players[id].X, w.players[id].Y = 19.5, 16.5 // waiting right by the hole
	stepN(w, RabbitHideTicks+2*TicksPerSecond)
	if r.State != CreatureHidden {
		t.Fatal("came out with a player right by the hole")
	}
	w.players[id].X = 4.5 // walks off
	stepN(w, 2*TicksPerSecond)
	if r.State == CreatureHidden {
		t.Fatal("still hiding with nobody about")
	}
}

func TestRabbitWontRunPastThePlayer(t *testing.T) {
	w, _, r := burrowWorld()
	id := w.AddPlayer("a")
	w.players[id].X = 19.5 // between the rabbit and its burrow
	r.X = 17.5
	w.Step()
	x := r.X
	stepN(w, 5)
	if r.State != CreatureFlee || r.X >= x {
		t.Fatalf("state %d, x %.2f → %.2f: want it bolting west, away from the player", r.State, x, r.X)
	}
}

func TestSnakesKeepToTheirDensAtNight(t *testing.T) {
	w := flatWorld()
	den := w.addEntity(EntityDen, 22, 16)
	w.fauna.homes = append(w.fauna.homes, homeRec{kind: EntityDen, tx: 22, ty: 16, id: den.ID})
	s := w.addCreature(CreatureSnake, 18.5, 16.5)
	s.home, s.homeX, s.homeY = den.ID, 22.5, 16.5
	id := w.AddPlayer("a") // two tiles from the snake

	w.tick = uint64(DayTicks*0.85) - 1 // night
	stepN(w, 5*TicksPerSecond)
	if s.State != CreatureHidden {
		t.Fatalf("state %d at (%.1f, %.1f): want the snake down its den", s.State, s.X, s.Y)
	}
	if p := mustPlayer(t, w, id); p.Notice == NoticeBitten {
		t.Fatal("bitten by a snake on its way home")
	}

	w.tick = uint64(DayTicks) + StartTick // the next morning
	w.Step()
	if s.State == CreatureHidden {
		t.Fatal("still in its den by day")
	}
}

func TestNewAnimalsComeOutOfHomes(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 3, Width: 128, Height: 128})
	if err != nil {
		t.Fatal(err)
	}
	w.fauna.creatures = nil
	for range CreatureRespawnTicks { // up to the first refill, and look at once
		if w.Step(); len(w.fauna.creatures) > 0 {
			break
		}
	}
	got := 0
	for _, c := range w.fauna.creatures {
		if c.Kind.def().homeKind == EntityNone {
			continue
		}
		got++
		if e := w.home(c); e == nil || c.X != c.homeX || c.Y != c.homeY {
			t.Errorf("%s %d came out at (%.1f, %.1f), not at a home", c.Kind.Name(), c.ID, c.X, c.Y)
		}
	}
	if got == 0 {
		t.Fatal("no rabbit or snake came back")
	}
}

func TestOldSavesGetTheirHomesBack(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 96, Height: 96})
	if err != nil {
		t.Fatal(err)
	}
	d := w.Save()
	var kept []Entity
	for _, e := range d.Entities { // as saved before homes existed
		if e.Kind != EntityBurrow && e.Kind != EntityDen {
			kept = append(kept, e)
		}
	}
	d.Entities = kept

	l, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	l.Step()
	homed := 0
	for _, c := range l.fauna.creatures {
		if c.Kind.def().homeKind != EntityNone && l.home(c) != nil {
			homed++
		}
	}
	if homed == 0 || len(l.fauna.homes) == 0 {
		t.Fatalf("%d homes, %d animals with one after loading an old save", len(l.fauna.homes), homed)
	}
}
