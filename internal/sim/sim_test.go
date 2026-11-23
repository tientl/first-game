package sim

import (
	"math"
	"testing"
)

// flatWorld is a 32x32 grass field with the spawn in the middle and no
// entities, for hand-built scenarios.
func flatWorld() *World {
	w := newWorld(32, 32)
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	w.spawnX, w.spawnY = 16.5, 16.5
	return w
}

func stepN(w *World, n int) {
	for i := 0; i < n; i++ {
		w.Step()
	}
}

func mustPlayer(t *testing.T, w *World, id PlayerID) Player {
	t.Helper()
	p, ok := w.Player(id)
	if !ok {
		t.Fatalf("player %d missing", id)
	}
	return p
}

func TestMovementIsBlockedByWater(t *testing.T) {
	w := flatWorld()
	for ty := 0; ty < w.height; ty++ {
		w.tiles[ty*w.width+18] = TileWater // a river two tiles east of spawn
	}
	id := w.AddPlayer("a")

	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 2*TicksPerSecond) // far enough to cross the river if nothing stopped it

	p := mustPlayer(t, w, id)
	if want := 18 - PlayerRadius; math.Abs(p.X-want) > PlayerSpeed {
		t.Fatalf("x = %.3f, want the player stopped at the bank (%.3f)", p.X, want)
	}
	if p.Dir != DirRight {
		t.Fatalf("dir = %v, want right", p.Dir)
	}

	// Sliding: moving diagonally into the river still moves along it.
	w.SetInput(id, Input{MoveX: 1, MoveY: 1})
	y0 := p.Y
	stepN(w, TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Y <= y0 {
		t.Fatalf("y = %.3f, want the player to slide south from %.3f", p.Y, y0)
	}
}

func TestDiagonalMovementIsNormalised(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.SetInput(id, Input{MoveX: 1, MoveY: 1})
	w.Step()
	p := mustPlayer(t, w, id)
	if d := math.Hypot(p.X-16.5, p.Y-16.5); math.Abs(d-PlayerSpeed) > 1e-9 {
		t.Fatalf("moved %.4f in one tick, want %.4f", d, PlayerSpeed)
	}
}

func TestInputIsClamped(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.SetInput(id, Input{MoveX: 100, MoveY: -128})
	w.Step()
	p := mustPlayer(t, w, id)
	if d := math.Hypot(p.X-16.5, p.Y-16.5); math.Abs(d-PlayerSpeed) > 1e-9 {
		t.Fatalf("moved %.4f in one tick with out-of-range input, want %.4f", d, PlayerSpeed)
	}
}

func TestChoppingATree(t *testing.T) {
	w := flatWorld()
	tree := w.addEntity(EntityTree, 17, 16) // right next to spawn
	id := w.AddPlayer("a")
	w.SetInput(id, Input{Action: true})

	// Hits land on the first tick and then every ActionCooldownTicks.
	stepN(w, 1+(TreeHits-1)*ActionCooldownTicks)
	p := mustPlayer(t, w, id)
	if p.Inv[ItemWood] != TreeYield {
		t.Fatalf("wood = %d, want %d", p.Inv[ItemWood], TreeYield)
	}
	if tree.Ready {
		t.Fatal("tree should be a stump after the last hit")
	}
	if p.Dir != DirRight {
		t.Fatalf("dir = %v, want the player to face the tree", p.Dir)
	}

	// A stump is not a target: holding Action does nothing more.
	stepN(w, 5*ActionCooldownTicks)
	if p := mustPlayer(t, w, id); p.Inv[ItemWood] != TreeYield {
		t.Fatalf("wood = %d after chopping a stump, want %d", p.Inv[ItemWood], TreeYield)
	}

	w.RemovePlayer(id) // keep the regrow wait free of night-time deaths
	stepN(w, TreeRegrowTicks)
	if !tree.Ready || tree.HP != TreeHits {
		t.Fatalf("tree after regrow: ready=%v hp=%d, want ready with %d hp", tree.Ready, tree.HP, TreeHits)
	}
}

func TestBoulderIsRemovedWhenMined(t *testing.T) {
	w := flatWorld()
	w.addEntity(EntityBoulder, 16, 17)
	id := w.AddPlayer("a")
	w.SetInput(id, Input{Action: true})
	stepN(w, 1+(BoulderHits-1)*ActionCooldownTicks)

	if p := mustPlayer(t, w, id); p.Inv[ItemStone] != BoulderYield {
		t.Fatalf("stone = %d, want %d", p.Inv[ItemStone], BoulderYield)
	}
	if _, ok := w.EntityAt(16, 17); ok {
		t.Fatal("boulder still on its tile")
	}
}

func TestFullBagRefusesTheHarvest(t *testing.T) {
	w := flatWorld()
	bush := w.addEntity(EntityBerryBush, 17, 16)
	id := w.AddPlayer("a")
	w.players[id].Inv[ItemBerry] = MaxStack - 1 // room for 1, the bush yields 2
	w.SetInput(id, Input{Action: true})
	w.Step()

	p := mustPlayer(t, w, id)
	if p.Inv[ItemBerry] != MaxStack-1 || !bush.Ready {
		t.Fatalf("berries=%d ready=%v, want the harvest refused", p.Inv[ItemBerry], bush.Ready)
	}
	if p.Notice != NoticeBagFull || ItemKind(p.NoticeArg) != ItemBerry || p.NoticeSeq == 0 {
		t.Fatalf("notice = %v/%d seq %d, want bag full for berries", p.Notice, p.NoticeArg, p.NoticeSeq)
	}
}

func TestNearestTargetPrefersCloserThenLowerID(t *testing.T) {
	w := flatWorld()
	far := w.addEntity(EntityBoulder, 15, 15) // diagonal, ~1.41 away
	near := w.addEntity(EntityBoulder, 16, 17)
	w.addEntity(EntityBoulder, 19, 16) // out of reach
	at := func(tx, ty int) (TargetCandidate, bool) {
		e := w.entityAt(tx, ty)
		if e == nil {
			return TargetCandidate{}, false
		}
		return TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready}, true
	}

	if id, ok := NearestTarget(16.5, 16.5, false, at); !ok || id != near.ID {
		t.Fatalf("target = %d,%v, want the adjacent boulder %d", id, ok, near.ID)
	}
	w.removeEntity(near)
	if id, ok := NearestTarget(16.5, 16.5, false, at); !ok || id != far.ID {
		t.Fatalf("target = %d,%v, want the diagonal boulder %d", id, ok, far.ID)
	}

	// Equidistant: the lower id wins.
	w2 := flatWorld()
	a := w2.addEntity(EntityBoulder, 15, 16)
	w2.addEntity(EntityBoulder, 17, 16)
	if id, _ := NearestTarget(16.5, 16.5, false, func(tx, ty int) (TargetCandidate, bool) {
		e := w2.entityAt(tx, ty)
		if e == nil {
			return TargetCandidate{}, false
		}
		return TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready}, true
	}); id != a.ID {
		t.Fatalf("tie went to %d, want the lower id %d", id, a.ID)
	}
}

func TestCraftingACampfire(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")

	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCampfire})
	if p := mustPlayer(t, w, id); p.Notice != NoticeNotEnough {
		t.Fatalf("notice = %v, want not enough", p.Notice)
	}

	w.players[id].Inv = Inventory{ItemWood: 5, ItemGrass: 2, ItemStone: 3}
	w.players[id].Dir = DirUp
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCampfire})

	p := mustPlayer(t, w, id)
	if p.Notice != NoticeCrafted {
		t.Fatalf("notice = %v, want crafted", p.Notice)
	}
	if want := (Inventory{ItemWood: 3}); p.Inv != want {
		t.Fatalf("inventory = %v, want %v", p.Inv, want)
	}
	fire, ok := w.EntityAt(16, 15) // the tile the player faces
	if !ok || fire.Kind != EntityCampfire || fire.Fuel != FireStartFuel {
		t.Fatalf("entity at facing tile = %+v,%v, want a fresh campfire", fire, ok)
	}

	// Refuelling: Action next to the fire spends one wood.
	w.SetInput(id, Input{Action: true})
	w.Step()
	fire, _ = w.EntityAt(16, 15)
	if p := mustPlayer(t, w, id); p.Inv[ItemWood] != 2 {
		t.Fatalf("wood = %d after refuelling, want 2", p.Inv[ItemWood])
	}
	if want := FireStartFuel + FireFuelPerWood - 1; fire.Fuel != want {
		t.Fatalf("fuel = %d, want %d", fire.Fuel, want)
	}
}

func TestCraftingWithNoSpaceKeepsTheIngredients(t *testing.T) {
	w := flatWorld()
	for _, d := range []Dir{DirDown, DirUp, DirLeft, DirRight} {
		dx, dy := d.Delta()
		w.tiles[(16+dy)*w.width+16+dx] = TileWater
	}
	id := w.AddPlayer("a")
	inv := Inventory{ItemWood: 2, ItemGrass: 2, ItemStone: 3}
	w.players[id].Inv = inv
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeCampfire})

	p := mustPlayer(t, w, id)
	if p.Notice != NoticeNoSpace || p.Inv != inv {
		t.Fatalf("notice=%v inv=%v, want no space and the ingredients kept", p.Notice, p.Inv)
	}
}

func TestCampfireBurnsOut(t *testing.T) {
	w := flatWorld()
	w.addEntity(EntityCampfire, 10, 10)
	stepN(w, FireStartFuel-1)
	if _, ok := w.EntityAt(10, 10); !ok {
		t.Fatal("campfire went out early")
	}
	w.Step()
	if _, ok := w.EntityAt(10, 10); ok {
		t.Fatal("campfire still burning after its fuel ran out")
	}
}

func TestNightColdKillsWithoutFire(t *testing.T) {
	night := uint64(DayTicks * (NightStart + 0.05))

	w := flatWorld()
	w.tick = night
	cold := w.AddPlayer("cold")
	w.Step()
	if p := mustPlayer(t, w, cold); !p.Cold {
		t.Fatal("player in the dark at night should be cold")
	}
	stepN(w, int(MaxHealth/ColdDamagePerTick))
	if p := mustPlayer(t, w, cold); !p.Dead {
		t.Fatalf("health = %.1f, want the player frozen to death", p.Health)
	}

	w2 := flatWorld()
	w2.tick = night
	w2.addEntity(EntityCampfire, 17, 16)
	warm := w2.AddPlayer("warm")
	stepN(w2, 10*TicksPerSecond)
	if p := mustPlayer(t, w2, warm); p.Cold || p.Health != MaxHealth {
		t.Fatalf("player by the fire: cold=%v health=%.1f, want warm and healthy", p.Cold, p.Health)
	}
}

func TestEating(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Inv[ItemBerry] = 1
	w.players[id].Hunger = 50

	w.Apply(id, Command{Kind: CmdEat, Item: ItemBerry})
	p := mustPlayer(t, w, id)
	if p.Inv[ItemBerry] != 0 || p.Hunger != 50+ItemBerry.Def().Hunger {
		t.Fatalf("berries=%d hunger=%.1f after eating", p.Inv[ItemBerry], p.Hunger)
	}

	w.Apply(id, Command{Kind: CmdEat, Item: ItemBerry})
	if p := mustPlayer(t, w, id); p.Notice != NoticeNoneLeft {
		t.Fatalf("notice = %v, want none left", p.Notice)
	}
	w.Apply(id, Command{Kind: CmdEat, Item: ItemStone})
	if p := mustPlayer(t, w, id); p.Notice != NoticeNotEdible {
		t.Fatalf("notice = %v, want not edible", p.Notice)
	}
	w.Apply(id, Command{Kind: CmdEat, Item: 200}) // garbage from the network
	if p := mustPlayer(t, w, id); p.Notice != NoticeNotEdible {
		t.Fatalf("notice = %v, want not edible for an invalid item", p.Notice)
	}
}

func TestStarvation(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Hunger = 0
	w.Step()
	if p := mustPlayer(t, w, id); math.Abs(p.Health-(MaxHealth-StarveDamagePerTick)) > 1e-9 {
		t.Fatalf("health = %.4f, want one tick of starvation damage", p.Health)
	}
}

func TestRespawnKeepsHeldInput(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.players[id].Health = StarveDamagePerTick / 2
	w.players[id].Hunger = 0
	w.players[id].Inv[ItemWood] = 7
	w.Step()
	if p := mustPlayer(t, w, id); !p.Dead {
		t.Fatal("player should have died")
	}

	// The key goes down while dead and stays down through the respawn.
	w.SetInput(id, Input{MoveX: 1})
	w.Apply(id, Command{Kind: CmdRespawn})
	p := mustPlayer(t, w, id)
	if p.Dead || p.Health != MaxHealth || p.Inv != (Inventory{}) || p.X != 16.5 {
		t.Fatalf("after respawn: %+v", p)
	}
	w.Step()
	if p := mustPlayer(t, w, id); p.X <= 16.5 {
		t.Fatal("held movement key ignored after respawn")
	}
}

func TestRemovePlayer(t *testing.T) {
	w := flatWorld()
	a, b, c := w.AddPlayer("a"), w.AddPlayer("b"), w.AddPlayer("c")
	w.RemovePlayer(b)
	w.RemovePlayer(b) // unknown ids are ignored
	got := w.Players()
	if len(got) != 2 || got[0].ID != a || got[1].ID != c {
		t.Fatalf("players = %+v, want a and c in order", got)
	}
	if d := w.AddPlayer("d"); d == b {
		t.Fatal("player id reused")
	}
}

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{
		"  Tiến  ":                   "Tiến",
		"a\x00b\nc":                  "abc",
		"\xff\xfeok":                 "ok",
		"Một cái tên rất là dài quá": "Một cái tên rất",
	} {
		if got := cleanName(in); got != want {
			t.Errorf("cleanName(%q) = %q, want %q", in, got, want)
		}
	}
	w := flatWorld()
	if p := mustPlayer(t, w, w.AddPlayer("   ")); p.Name != "Người chơi 1" {
		t.Errorf("blank name became %q", p.Name)
	}
}

func TestAmbientAndPhases(t *testing.T) {
	at := func(f float64) uint64 { return uint64(f * DayTicks) }
	if got := Ambient(StartTick); got != 1 {
		t.Errorf("ambient at start = %v, want full daylight", got)
	}
	if PhaseOf(at(0.3)) != PhaseDay || PhaseOf(at(0.6)) != PhaseDusk || PhaseOf(at(0.9)) != PhaseNight {
		t.Error("phase boundaries wrong")
	}
	if got := Ambient(at(0.9)); got != 0 {
		t.Errorf("ambient in deep night = %v, want 0", got)
	}
	// No jumps: the light changes by a small amount between consecutive ticks.
	prev := Ambient(0)
	for tick := uint64(1); tick <= 2*DayTicks; tick++ {
		cur := Ambient(tick)
		if math.Abs(cur-prev) > 0.02 {
			t.Fatalf("ambient jumps from %.3f to %.3f at tick %d", prev, cur, tick)
		}
		prev = cur
	}
	if DayOf(StartTick) != 1 || DayOf(DayTicks) != 2 {
		t.Error("day numbering wrong")
	}
}
