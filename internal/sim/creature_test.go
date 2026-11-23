package sim

import (
	"math"
	"slices"
	"testing"
)

func creatureByKind(w *World, k CreatureKind) []Creature {
	var out []Creature
	for _, c := range w.Creatures() {
		if c.Kind == k {
			out = append(out, c)
		}
	}
	return out
}

func TestCreaturesPopulateTheIsland(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3, 42} {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatal(err)
		}
		sx, sy := int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY))
		for k := CreatureKind(1); k < NumCreatureKinds; k++ {
			cs := creatureByKind(w, k)
			if len(w.fauna.spawnAt[k]) == 0 {
				continue // no home on an island: crocodiles keep to the open world's marshes
			}
			if len(cs) < creatureMinPerKind {
				t.Errorf("seed %d: %d %s, want at least %d", seed, len(cs), k.Name(), creatureMinPerKind)
			}
			for _, c := range cs {
				if !w.creatureFits(k, c.X, c.Y) {
					t.Errorf("seed %d: %s at (%.1f, %.1f) is outside its habitat", seed, k.Name(), c.X, c.Y)
				}
				if k == CreatureSnake && max(abs(int(c.X)-sx), abs(int(c.Y)-sy)) < snakeSpawnClear {
					t.Errorf("seed %d: snake at (%.1f, %.1f) right by the spawn", seed, c.X, c.Y)
				}
			}
		}
		again, _ := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if !slices.Equal(w.Creatures(), again.Creatures()) {
			t.Errorf("seed %d: creatures differ between two generations", seed)
		}
	}
}

func TestCreaturesStayInTheirHabitat(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 96, Height: 96})
	if err != nil {
		t.Fatal(err)
	}
	stepN(w, 60*TicksPerSecond)
	for _, c := range w.Creatures() {
		if !w.creatureFits(c.Kind, c.X, c.Y) {
			t.Errorf("%s %d wandered to (%.2f, %.2f), outside its habitat", c.Kind.Name(), c.ID, c.X, c.Y)
		}
	}
}

func TestRabbitBoltsThenTires(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	r := w.addCreature(CreatureRabbit, 18.5, 16.5) // two tiles east of the player
	w.Step()
	if r.State != CreatureFlee || r.X <= 18.5 {
		t.Fatalf("state %d at x %.2f: want the rabbit bolting east", r.State, r.X)
	}
	stepN(w, RabbitFleeTicks)
	if r.State != CreatureRest {
		t.Fatalf("state %d after the sprint, want resting", r.State)
	}
	// Faster than a player while it runs.
	if d := r.X - 18.5; d < float64(RabbitFleeTicks-1)*PlayerSpeed {
		t.Fatalf("ran %.2f tiles, want more than a player could", d)
	}
	_ = id
}

func TestCatchingARestingRabbit(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	r := w.addCreature(CreatureRabbit, 17.5, 16.5)
	r.State, r.timer = CreatureRest, RabbitRestTicks
	w.SetInput(id, Input{Action: true})
	w.Step()
	if n := len(entitiesOf(w, EntityCorpse)); n != 1 {
		t.Fatalf("%d bodies, want the rabbit lying there", n)
	}
	reap(w, id)
	p := mustPlayer(t, w, id)
	if p.Inv.Count(ItemMeat) != creatureDefs[CreatureRabbit].yieldN {
		t.Fatalf("meat = %d, want the rabbit caught", p.Inv.Count(ItemMeat))
	}
	if len(w.Creatures()) != 0 {
		t.Fatalf("%d creatures left, want the rabbit gone", len(w.Creatures()))
	}
	if !p.Acting(w.Tick()) || p.Dir != DirRight {
		t.Errorf("acting %v, dir %v: want a swing to the right", p.Acting(w.Tick()), p.Dir)
	}
}

func TestRabbitsSleepAtNight(t *testing.T) {
	w := flatWorld()
	w.tick = uint64(DayTicks * 0.85)
	w.AddPlayer("a")
	r := w.addCreature(CreatureRabbit, 17.5, 16.5)
	stepN(w, 10)
	if r.State != CreatureSleep || r.X != 17.5 {
		t.Fatalf("state %d at x %.2f: want it asleep where it was", r.State, r.X)
	}
}

func TestSnakeChasesAndBites(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	s := w.addCreature(CreatureSnake, 19.5, 16.5) // three tiles east
	stepN(w, 2*TicksPerSecond)
	p := mustPlayer(t, w, id)
	// A well-fed player has healed a little since.
	if bitten := MaxHealth - SnakeBiteDamage; p.Health < bitten || p.Health > bitten+1 {
		t.Fatalf("health %.1f, want one bite (%.1f)", p.Health, bitten)
	}
	if p.Notice != NoticeBitten || CreatureKind(p.NoticeArg) != CreatureSnake {
		t.Fatalf("notice %v/%d, want bitten by a snake", p.Notice, p.NoticeArg)
	}
	if d := math.Hypot(s.X-p.X, s.Y-p.Y); d > SnakeBiteRange {
		t.Fatalf("snake %.2f tiles away, want it within biting range", d)
	}
}

func TestWalkingAwayFromASnake(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	s := w.addCreature(CreatureSnake, 19, 16.5) // 2.5 tiles east
	w.SetInput(id, Input{MoveX: -1})
	stepN(w, 2*TicksPerSecond)
	p := mustPlayer(t, w, id)
	if p.Health != MaxHealth {
		t.Fatalf("health %.1f: a player walking away was bitten", p.Health)
	}
	if s.State != CreatureChase {
		t.Fatalf("state %d, want the snake giving chase", s.State)
	}
	if d := math.Hypot(s.X-p.X, s.Y-p.Y); d < 2.5 {
		t.Fatalf("snake %.2f tiles behind, want the player pulling away", d)
	}

	w.players[id].X = s.X - SnakeGiveUpDist - 1
	w.Step()
	if s.State == CreatureChase {
		t.Fatal("the snake is still chasing a player far away")
	}
}

func TestKillingASnake(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.addCreature(CreatureSnake, 17.5, 16.5)
	w.SetInput(id, Input{Action: true})
	stepN(w, 1+2*creatureDefs[CreatureSnake].hp*ActionCooldownTicks)
	reap(w, id)
	p := mustPlayer(t, w, id)
	if len(creatureByKind(w, CreatureSnake)) != 0 || p.Inv.Count(ItemMeat) != creatureDefs[CreatureSnake].yieldN {
		t.Fatalf("snakes %d, meat %d: want the snake killed for its meat", len(creatureByKind(w, CreatureSnake)), p.Inv.Count(ItemMeat))
	}
	if p.Dead {
		t.Fatal("died to one snake")
	}
}

// shoreWorld is flatWorld with the sea from column 18 eastwards.
func shoreWorld() *World {
	w := flatWorld()
	for ty := 0; ty < w.height; ty++ {
		for tx := 18; tx < w.width; tx++ {
			w.tiles[ty*w.width+tx] = TileWater
		}
	}
	return w
}

func TestFishingNeedsAWorm(t *testing.T) {
	w := shoreWorld()
	id := w.AddPlayer("a")
	if !w.creatureFits(CreatureFish, 18.5, 16.5) || w.creatureFits(CreatureFish, 20.5, 16.5) {
		t.Fatal("fish should swim right by the shore only")
	}
	fish := w.addCreature(CreatureFish, 18.5, 16.5)
	fish.State, fish.timer = CreatureIdle, 1000 // hold still for the test
	w.players[id].Inv.Add(ItemRod, 1)           // lands in the held slot 0; see crafting_test for no rod
	w.SetInput(id, Input{Action: true})
	w.Step()
	p := mustPlayer(t, w, id)
	if p.Notice != NoticeNeedBait || ItemKind(p.NoticeArg) != ItemWorm || p.Inv.Count(ItemFish) != 0 {
		t.Fatalf("notice %v/%d, fish %d: want a request for bait", p.Notice, p.NoticeArg, p.Inv.Count(ItemFish))
	}

	// With a worm the line goes out; fishing_test.go plays the rest.
	w.players[id].Inv.Add(ItemWorm, 2)
	stepN(w, ActionCooldownTicks)
	if st, _, _ := w.players[id].Cast(); st != CastWaiting || fish.State != CreatureHooked {
		t.Fatalf("cast %d, fish state %d: want the line out on the fish", st, fish.State)
	}
}

func TestPickingUpAWorm(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.addCreature(CreatureWorm, 16.5, 17.3)
	w.SetInput(id, Input{Action: true})
	w.Step()
	reap(w, id)
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemWorm) != 1 || p.Dir != DirDown {
		t.Fatalf("worms %d, dir %v: want the worm picked up below", p.Inv.Count(ItemWorm), p.Dir)
	}
}

func TestPickTargetPrefersTheNearest(t *testing.T) {
	tree := TargetCandidate{ID: 5, Kind: EntityTree, Ready: true}
	at := func(tx, ty int) (TargetCandidate, bool) { return tree, tx == 17 && ty == 16 } // 1 tile east
	near := []CreatureCandidate{{ID: 9, Kind: CreatureRabbit, X: 16.5, Y: 17.0}}         // half a tile south
	if tg, ok := PickTarget(16.5, 16.5, false, at, near, 0); !ok || tg.Creature != 9 {
		t.Fatalf("target %+v, want the nearer rabbit", tg)
	}
	far := []CreatureCandidate{{ID: 9, Kind: CreatureRabbit, X: 18.5, Y: 16.5}} // beyond reach
	if tg, ok := PickTarget(16.5, 16.5, false, at, far, 0); !ok || tg.Entity != 5 {
		t.Fatalf("target %+v, want the tree", tg)
	}
	fish := []CreatureCandidate{{ID: 3, Kind: CreatureFish, X: 18.5, Y: 16.5}} // two tiles, within a cast
	if tg, ok := PickTarget(16.5, 16.5, false, func(int, int) (TargetCandidate, bool) { return TargetCandidate{}, false }, fish, 0); !ok || tg.Creature != 3 {
		t.Fatalf("target %+v, want the fish", tg)
	}
}

func TestAnimalsComeBackAwayFromPlayers(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 3, Width: 128, Height: 128})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("a")
	before := len(w.Creatures())
	w.fauna.creatures = nil
	stepN(w, 3*CreatureRespawnTicks)
	got := w.Creatures()
	if len(got) == 0 || len(got) >= before {
		t.Fatalf("%d creatures after three refills, want a few back but not all at once (had %d)", len(got), before)
	}
	p := mustPlayer(t, w, id)
	for _, c := range got {
		// They may have walked a little since appearing.
		if d := math.Hypot(c.X-p.X, c.Y-p.Y); d < creatureSpawnAway-3 {
			t.Errorf("%s appeared %.1f tiles from the player", c.Kind.Name(), d)
		}
	}
}
