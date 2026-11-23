package sim

import (
	"math"
	"testing"
)

func genSeas(t *testing.T, seed uint64, size int) *World {
	t.Helper()
	w, err := Generate(GenConfig{Seed: seed, Width: size, Height: size, Style: StyleSeas})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestArchipelagoHasIslandsOfEveryKind(t *testing.T) {
	for _, c := range []struct {
		seed uint64
		size int
	}{{1, SizeMedium}, {2, SizeMedium}, {7, SizeLarge}} {
		w := genSeas(t, c.seed, c.size)
		if n := len(w.Islands()); n < 6 {
			t.Errorf("seed %d: %d islands", c.seed, n)
		}
		var have [numIslandKinds]bool
		for _, il := range w.Islands() {
			have[il.Kind] = true
		}
		for k, ok := range have {
			if !ok {
				t.Errorf("seed %d size %d: no island of kind %d", c.seed, c.size, k)
			}
		}
		// The spawn is on the home island, which no other land touches.
		if tl := w.TileAt(int(w.spawnX), int(w.spawnY)); tl != TileGrass && tl != TileForest {
			t.Errorf("seed %d: spawn on tile %d", c.seed, tl)
		}
		var count [NumTiles]int
		for _, tl := range w.tiles {
			count[tl]++
		}
		for _, k := range []Tile{TileSnow, TileDesert, TileMarsh, TileAsh, TileLava, TileCave, TileMountain} {
			if count[k] == 0 {
				t.Errorf("seed %d size %d: no tile %d", c.seed, c.size, k)
			}
		}
		if frac := float64(count[TileWater]) / float64(len(w.tiles)); frac < 0.5 {
			t.Errorf("seed %d: only %.0f%% sea", c.seed, 100*frac)
		}
	}
}

// seaWorld is open water with a strip of beach on the left, the spawn on it.
func seaWorld() *World {
	w := newWorld(64, 32)
	w.style = StyleSeas
	for i := range w.tiles {
		w.tiles[i] = TileWater
		if i%w.width < 4 {
			w.tiles[i] = TileSand
		}
	}
	w.spawnX, w.spawnY = 2.5, 16.5
	return w
}

func TestLaunchBoardSailAndLand(t *testing.T) {
	w := seaWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X, p.Dir = 3.5, DirRight
	p.Inv[0] = Slot{Item: ItemRaft, Count: 1}
	p.Selected = 0
	w.Apply(id, Command{Kind: CmdPlace})
	if len(w.boats) != 1 || p.Inv.Count(ItemRaft) != 0 {
		t.Fatalf("%d boats: want the raft on the water", len(w.boats))
	}
	w.Apply(id, Command{Kind: CmdRide})
	if p.boat == 0 {
		t.Fatal("did not get aboard")
	}
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 5*TicksPerSecond)
	if p.X < 12 {
		t.Fatalf("x %.1f after five seconds of paddling", p.X)
	}
	if p.Health < MaxHealth-0.01 && p.Breath() < 1 {
		t.Fatal("a player in a boat swims")
	}
	// Back to the beach, and ashore.
	w.SetInput(id, Input{MoveX: -1})
	stepN(w, 12*TicksPerSecond)
	w.SetInput(id, Input{})
	w.Apply(id, Command{Kind: CmdRide})
	if p.boat != 0 || !w.standable(p.X, p.Y) {
		t.Fatalf("boat %d at (%.1f,%.1f): want ashore", p.boat, p.X, p.Y)
	}
	if d := w.Save(); len(d.Boats) != 1 {
		t.Fatal("the boat was not saved")
	}
}

func TestSailRunsBeforeTheWind(t *testing.T) {
	if SailFactor(0, 0, 1) <= 2*SailFactor(math.Pi, 0, 1) {
		t.Fatal("a sail is not much faster before the wind than into it")
	}
}

func TestSwimmersTireAndDrown(t *testing.T) {
	w := seaWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 4*TicksPerSecond)
	if !w.Swimming(p) || p.X < 5 {
		t.Fatalf("x %.1f: want the player swimming out", p.X)
	}
	w.SetInput(id, Input{})
	stepN(w, SwimTireTicks+5*TicksPerSecond)
	if p.Health >= MaxHealth {
		t.Fatal("a tired swimmer does not drown")
	}
}

func TestMosasaurBitesBoats(t *testing.T) {
	w := seaWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.nextBoat++
	b := &Boat{ID: w.nextBoat, Kind: BoatRaft, X: 30.5, Y: 16.5, HP: BoatRaft.MaxHP()}
	w.boats = append(w.boats, b)
	p.X, p.Y, p.boat, b.Rider = b.X, b.Y, b.ID, id
	m := w.addCreature(CreatureMosa, 40.5, 16.5)
	stepN(w, 6*TicksPerSecond)
	if b.HP >= BoatRaft.MaxHP() {
		t.Fatalf("mosasaur state %d at %.1f: the raft was never bitten", m.State, m.X)
	}
	stepN(w, 20*TicksPerSecond)
	if !b.gone && b.HP > 0 {
		t.Fatalf("raft hp %d: want it sunk", b.HP)
	}
	if p.boat != 0 {
		t.Fatal("still aboard a sunk raft")
	}
}

func TestKrakenHoldsABoat(t *testing.T) {
	w := seaWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.nextBoat++
	b := &Boat{ID: w.nextBoat, Kind: BoatSail, X: 30.5, Y: 16.5, HP: BoatSail.MaxHP()}
	w.boats = append(w.boats, b)
	p.X, p.Y, p.boat, b.Rider = b.X, b.Y, b.ID, id
	k := w.addCreature(CreatureKraken, 32.5, 16.5)
	w.Step()
	if k.State != CreatureGrab {
		t.Fatalf("state %d: want the squid to seize the boat", k.State)
	}
	w.SetInput(id, Input{MoveX: 1})
	x := b.X
	stepN(w, 2*TicksPerSecond)
	if b.X-x > 0.3 || b.HP >= BoatSail.MaxHP() {
		t.Fatalf("moved %.2f, hp %d: want held fast and crushed", b.X-x, b.HP)
	}
}

func TestSeaWorldRuns(t *testing.T) {
	w := genSeas(t, 3, SizeMedium)
	id := w.AddPlayer("a")
	for i := range 3000 {
		w.SetInput(id, Input{MoveX: int8(i/400%3 - 1), MoveY: int8(i/900%3 - 1)})
		if w.players[id].Dead {
			w.Apply(id, Command{Kind: CmdRespawn})
		}
		w.Step()
	}
	if _, err := Load(w.Save()); err != nil {
		t.Fatal(err)
	}
	monsters := 0
	for _, c := range w.fauna.creatures {
		if c.Kind == CreatureMosa || c.Kind == CreatureKraken {
			monsters++
		}
	}
	if monsters == 0 {
		t.Fatal("no sea monsters")
	}
}

func TestPyramidStandsOnAFarDesertIsland(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3} {
		w := genSeas(t, seed, SizeMedium)
		tx, ty, ok := w.Pyramid()
		if !ok {
			t.Fatalf("seed %d: no pyramid", seed)
		}
		if w.TileAt(tx, ty) != TileDesert && w.TileAt(tx, ty) != TileSand {
			t.Errorf("seed %d: pyramid on tile %d", seed, w.TileAt(tx, ty))
		}
		if d := math.Hypot(float64(tx)-w.spawnX, float64(ty)-w.spawnY); d < SizeMedium/5 {
			t.Errorf("seed %d: pyramid %.0f tiles from the spawn", seed, d)
		}
		if e, ok := w.EntityAt(tx, ty+1); !ok || e.Kind != EntityTreasure {
			t.Errorf("seed %d: no chest in the doorway", seed)
		}
		if !w.blocked(tx-2, ty-2) {
			t.Errorf("seed %d: the pyramid does not block the way", seed)
		}
		scorpions := 0
		for _, c := range w.fauna.creatures {
			if c.Kind == CreatureScorpion {
				scorpions++
			}
		}
		if scorpions < pyramidGuards {
			t.Errorf("seed %d: %d scorpions", seed, scorpions)
		}
	}
}

func TestScorpionPinchesThenStings(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.addCreature(CreatureScorpion, 17.6, 16.5)
	stepN(w, 4*TicksPerSecond)
	p := w.players[id]
	if p.Health > MaxHealth-ScorpionPinch-ScorpionSting+1 || !p.Poisoned() {
		t.Fatalf("health %.0f poisoned %v: want pinched and stung", p.Health, p.Poisoned())
	}
}
