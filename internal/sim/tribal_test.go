package sim

import (
	"math"
	"testing"
)

func TestFelledTreeDropsWoodAndBranchesToPickUp(t *testing.T) {
	w := flatWorld()
	w.addEntity(EntityTree, 17, 16)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemAxe, 1)
	w.SetInput(id, Input{Action: true})
	stepN(w, 1+(TreeHits-1)*ActionCooldownTicks)
	wood := 0
	for _, d := range w.drops {
		if d.Item == ItemWood {
			wood += d.Count
		}
	}
	if wood != TreeYield {
		t.Fatalf("%d wood on the ground, want %d", wood, TreeYield)
	}
	// Holding Space and walking past, the piles in reach are picked up one by one.
	w.SetInput(id, Input{Action: true, MoveX: 1})
	stepN(w, TicksPerSecond/2)
	w.SetInput(id, Input{Action: true})
	stepN(w, 2*TicksPerSecond)
	if got := w.players[id].Inv.Count(ItemWood); got != TreeYield {
		t.Fatalf("picked up %d wood, want %d", got, TreeYield)
	}
}

func TestWalkingOverAPilePicksItUp(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	w.dropAt(20.5, 16.5, []Ingredient{{Item: ItemStone, Count: 3}})
	d := w.drops[0]
	d.X, d.Y = 20.5, 16.5
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 2*TicksPerSecond)
	if got := w.players[id].Inv.Count(ItemStone); got != 3 || len(w.drops) != 0 {
		t.Fatalf("stone %d, %d piles left: want the pile walked up", got, len(w.drops))
	}
}

func TestPilesRotAway(t *testing.T) {
	w := flatWorld()
	w.dropAt(10.5, 10.5, []Ingredient{{Item: ItemStone, Count: 1}})
	stepN(w, DropLifeTicks+2)
	if len(w.drops) != 0 {
		t.Fatal("a pile lay forever")
	}
}

func TestKillLeavesABodyToButcher(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureRaptor, 17.6, 16.5)
	c.HP = 1
	id := w.AddPlayer("a")
	hitUntil(w, id, TicksPerSecond, func() bool { return c.gone })
	bodies := entitiesOf(w, EntityCorpse)
	if len(bodies) != 1 || CreatureKind(bodies[0].Var) != CreatureRaptor {
		t.Fatalf("bodies %v: want one raptor lying there", bodies)
	}
	if n := w.players[id].Inv.Count(ItemMeat); n != 0 {
		t.Fatalf("meat %d straight in the bag, want it in the body", n)
	}
	hitUntil(w, id, 4*TicksPerSecond, func() bool { return len(entitiesOf(w, EntityCorpse)) == 0 })
	if len(entitiesOf(w, EntityCorpse)) != 0 {
		t.Fatal("the body was not butchered")
	}
	reap(w, id)
	inv := &w.players[id].Inv
	if inv.Count(ItemMeat) != raptorDef.yieldN || inv.Count(HeadItem(CreatureRaptor)) != 1 || inv.Count(ItemDinoHide) != 1 {
		t.Fatalf("meat %d head %d dino hide %d: want the raptor's meat, its head and its hide",
			inv.Count(ItemMeat), inv.Count(HeadItem(CreatureRaptor)), inv.Count(ItemDinoHide))
	}
}

func TestBodiesRotAway(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureRabbit, 10.5, 10.5)
	w.fell(c)
	if len(entitiesOf(w, EntityCorpse)) != 1 {
		t.Fatal("no body")
	}
	stepN(w, CorpseRotTicks+1)
	if len(entitiesOf(w, EntityCorpse)) != 0 {
		t.Fatal("the body never rotted")
	}
}

func TestHeadsHaveNamesAndGoOnPosts(t *testing.T) {
	if HeadItem(CreatureRex).Def().Name != "Đầu khủng long bạo chúa" {
		t.Errorf("rex head is called %q", HeadItem(CreatureRex).Def().Name)
	}
	if HeadItem(CreatureTribesman).Valid() || HeadItem(CreatureFish).Valid() {
		t.Error("people and fish have no trophy heads")
	}
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv[0] = Slot{Item: HeadItem(CreatureBear), Count: 1}
	p.Selected, p.Dir = 0, DirRight
	w.Apply(id, Command{Kind: CmdPlace})
	e, ok := w.EntityAt(17, 16)
	if !ok || e.Kind != EntityTrophy || CreatureKind(e.Var) != CreatureBear || p.Inv.Count(HeadItem(CreatureBear)) != 0 {
		t.Fatalf("entity %+v %v: want the bear's head on a post", e, ok)
	}
}

func TestDoorsLetPlayersThroughButNotBeasts(t *testing.T) {
	w := flatWorld()
	for ty := 0; ty < w.height; ty++ {
		w.addEntity(EntityWall, 18, ty)
	}
	w.removeEntity(w.entityAt(18, 16))
	w.addEntity(EntityDoor, 18, 16)
	if w.creatureFits(CreatureRaptor, 18.5, 16.5) {
		t.Error("a raptor fits in a doorway")
	}
	id := w.AddPlayer("a")
	w.SetInput(id, Input{MoveX: 1})
	stepN(w, 2*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.X < 19 {
		t.Fatalf("x %.2f: the player did not go through the door", p.X)
	}
}

func TestSnareHoldsABeast(t *testing.T) {
	w := flatWorld()
	trap := w.addEntity(EntityTrap, 10, 10)
	c := w.addCreature(CreatureThesc, 10.5, 10.5)
	w.Step()
	if c.State != CreatureTrapped || trap.Ready {
		t.Fatalf("state %d, trap set %v: want the beast caught", c.State, trap.Ready)
	}
	x := c.X
	stepN(w, 2*TicksPerSecond)
	if c.X != x {
		t.Fatal("a snared beast walked off")
	}
}

func TestSpikesWoundBeasts(t *testing.T) {
	w := flatWorld()
	w.addEntity(EntitySpikes, 10, 10)
	c := w.addCreature(CreatureThesc, 10.5, 11.3)
	hp := c.HP
	stepN(w, 3*TicksPerSecond)
	if c.HP >= hp {
		t.Fatal("the stakes did nothing")
	}
}

func TestStandingTorchWarmsAndBurnsOut(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv[0] = Slot{Item: ItemTorch, Count: 1}
	p.Selected, p.Dir = 0, DirUp
	w.Apply(id, Command{Kind: CmdPlace})
	e, ok := w.EntityAt(16, 15)
	if !ok || e.Kind != EntityTorchStand {
		t.Fatalf("got %+v: want a torch in the ground", e)
	}
	if !w.warm(16.5, 16.5) {
		t.Fatal("a standing torch gives no warmth")
	}
	w.RemovePlayer(id)
	stepN(w, TorchStandTicks+1)
	if _, ok := w.EntityAt(16, 15); ok {
		t.Fatal("the torch never burnt out")
	}
}

func TestSeedsGrow(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv[0] = Slot{Item: ItemTreeSeed, Count: 1}
	p.Selected, p.Dir = 0, DirLeft
	w.Apply(id, Command{Kind: CmdPlace})
	e := w.entityAt(15, 16)
	if e == nil || e.Kind != EntitySapling {
		t.Fatalf("got %+v: want a seedling", e)
	}
	w.RemovePlayer(id)
	stepN(w, int(SeedTree.growTicks())+1)
	if e.Kind != EntityTree || !e.Ready {
		t.Fatalf("got kind %d: want a tree", e.Kind)
	}
}

func TestManEaterBitesPlayersButAGuardSparesThem(t *testing.T) {
	w := flatWorld()
	e := w.addEntity(EntityManEater, 17, 16)
	id := w.AddPlayer("a")
	w.Step()
	if p := mustPlayer(t, w, id); p.Health >= MaxHealth || p.Notice != NoticePlantBite {
		t.Fatalf("health %.0f notice %d: want bitten", p.Health, p.Notice)
	}
	w.players[id].Health = MaxHealth
	e.Var = 1
	e.RegrowAt = 0
	stepN(w, 3*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health < MaxHealth {
		t.Fatal("a planted guard bit its own people")
	}
}

func TestBedIsWhereYouComeBack(t *testing.T) {
	w := flatWorld()
	bed := w.addEntity(EntityBed, 5, 5)
	id := w.AddPlayer("a")
	w.tick = uint64(DayTicks*NightStart) + DayTicks // night
	p := w.players[id]
	w.sleep(p, bed)
	if p.HomeBed() != bed.ID {
		t.Fatal("sleeping did not make the bed home")
	}
	hurt(p, 1000)
	w.Apply(id, Command{Kind: CmdRespawn})
	if math.Hypot(p.X-5.5, p.Y-5.5) > 1.5 {
		t.Fatalf("came back at (%.1f, %.1f), want by the bed", p.X, p.Y)
	}
	// Saved and loaded, it is still home.
	ps := savePlayer(p)
	if ps.Home != bed.ID {
		t.Fatal("home bed not saved")
	}
}

func TestPotionHeals(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Health = 20
	p.poison = 100
	p.Inv[0] = Slot{Item: ItemBigPotion, Count: 1}
	w.Apply(id, Command{Kind: CmdEat, Slot: 0})
	if p.Health < 99 || p.Poisoned() {
		t.Fatalf("health %.0f poisoned %v: want healed and cured", p.Health, p.Poisoned())
	}
}

func TestWeatherIsSharedAndFollowsTheSeasons(t *testing.T) {
	counts := [NumSeasons][NumWeathers]int{}
	for tick := uint64(0); tick < 40*SeasonDays*DayTicks; tick += WeatherTicks {
		if WeatherAt(7, tick) != WeatherAt(7, tick) {
			t.Fatal("the weather is not a function of seed and tick")
		}
		counts[SeasonOf(tick)][WeatherAt(7, tick)]++
	}
	if counts[Winter][WeatherSnow] == 0 || counts[Summer][WeatherSnow] != 0 || counts[Summer][WeatherStorm] == 0 {
		t.Fatalf("weather by season: %v", counts)
	}
	if SeasonOf(StartTick) != Spring || SeasonOf(StartTick+uint64(3*SeasonDays*DayTicks)) != Winter {
		t.Fatal("seasons out of order")
	}
}

func TestPondsFreezeInWinter(t *testing.T) {
	w := flatWorld()
	for ty := 10; ty < 14; ty++ {
		for tx := 10; tx < 14; tx++ {
			w.tiles[ty*w.width+tx] = TileWater
		}
	}
	if w.landAt(11, 11) {
		t.Fatal("a pond is solid in spring")
	}
	w.tick = uint64(3*SeasonDays*DayTicks) + DayTicks/2
	if !w.landAt(11, 11) {
		t.Fatal("a pond does not freeze in winter")
	}
}

func TestLightningHurtsWhoItHits(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	for range 200 {
		w.lightning()
	}
	if len(w.wx.strikes) == 0 {
		t.Fatal("no strikes")
	}
	if p.Health >= MaxHealth && !p.Dead {
		t.Fatal("two hundred bolts and never a near miss")
	}
}

func TestSpiderWebsAndPoison(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	s := w.addCreature(CreatureSpider, 20.5, 16.5)
	stepN(w, 2*TicksPerSecond)
	p := w.players[id]
	if !p.Webbed() && !p.Poisoned() && p.Health >= MaxHealth {
		t.Fatalf("spider state %d: it did nothing to a player 4 tiles off", s.State)
	}
	stepN(w, 5*TicksPerSecond)
	if !p.Poisoned() && p.Health >= MaxHealth {
		t.Fatal("never bitten")
	}
}

func TestBearClawsThenBites(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	b := w.addCreature(CreatureBear, 18.0, 16.5)
	sawClaw := false
	for range 6 * TicksPerSecond {
		w.Step()
		if b.State == CreatureClaw {
			sawClaw = true
		}
	}
	if !sawClaw || w.players[id].Health >= MaxHealth-BearClaw {
		t.Fatalf("claw %v health %.0f: want the bear to maul", sawClaw, w.players[id].Health)
	}
}

func TestTribalWorlds(t *testing.T) {
	w := genOpen(t, 1, SizeMedium)
	if len(w.villages) < 2 {
		t.Fatalf("%d villages", len(w.villages))
	}
	sx, sy := w.Spawn()
	paths, huts, folk, treasure, flowers, glow := 0, 0, 0, 0, 0, 0
	for _, tl := range w.tiles {
		if tl == TilePath {
			paths++
		}
	}
	for _, e := range w.entities {
		switch e.Kind {
		case EntityHut:
			huts++
		case EntityTreasure:
			treasure++
		case EntityFlower:
			flowers++
		case EntityGlowPlant:
			glow++
		}
	}
	for _, c := range w.fauna.creatures {
		if c.Kind == CreatureTribesman {
			folk++
		}
	}
	for _, v := range w.villages {
		if math.Hypot(v.X-sx, v.Y-sy) < villageMinAway {
			t.Errorf("village at (%.0f,%.0f) is close to the spawn", v.X, v.Y)
		}
	}
	if paths < 100 || huts < 6 || folk < 6 || treasure < 2 || flowers < 50 || glow < 20 {
		t.Fatalf("paths %d huts %d folk %d treasure %d flowers %d glowing %d", paths, huts, folk, treasure, flowers, glow)
	}
	// The snow and the volcano are a long way off.
	snowAway := tribalSnowAway * float64(SizeMedium) * 0.85
	for i, tl := range w.tiles {
		d := math.Hypot(float64(i%w.width)+0.5-sx, float64(i/w.width)+0.5-sy)
		if tl == TileSnow && d < snowAway-1 {
			t.Fatalf("snow %.0f tiles from the spawn", d)
		}
		if tl == TileLava && d < float64(SizeMedium)/4 {
			t.Fatalf("lava %.0f tiles from the spawn", d)
		}
	}
}

func TestDropsAndHomeAreSaved(t *testing.T) {
	w := flatWorld()
	w.dropAt(10.5, 10.5, []Ingredient{{Item: ItemStone, Count: 2}})
	d := w.Save()
	if len(d.Drops) != 1 || d.Drops[0].Count != 2 {
		t.Fatalf("saved drops %+v", d.Drops)
	}
}

// TestTribalWorldRuns steps a new world for a while, through every kind of
// weather, with a player wandering about, to shake out panics.
func TestTribalWorldRuns(t *testing.T) {
	w := genOpen(t, 5, SizeMedium)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemSpear, 1)
	for i := range 6000 {
		if i%400 == 0 {
			w.tick += WeatherTicks * 3 // skip ahead to other weather and seasons
		}
		w.SetInput(id, Input{MoveX: int8(i/300%3 - 1), MoveY: int8(i/700%3 - 1), Action: i%7 == 0})
		if p := w.players[id]; p.Dead {
			w.Apply(id, Command{Kind: CmdRespawn})
		}
		w.Step()
	}
	if _, err := Load(w.Save()); err != nil {
		t.Fatal(err)
	}
}

func TestManEaterSwallowsSmallDinosaurs(t *testing.T) {
	w := flatWorld()
	e := w.addEntity(EntityManEater, 10, 10)
	small := w.addCreature(CreatureThesc, 11.6, 10.5)
	small.State, small.timer = CreatureIdle, 1000
	stepN(w, 2*TicksPerSecond)
	if !small.gone || len(entitiesOf(w, EntityCorpse)) != 0 {
		t.Fatalf("thesc gone %v: want it swallowed whole, no body", small.gone)
	}
	if !ManEaterChewing(e.RegrowAt, w.tick) {
		t.Fatal("not chewing after a meal")
	}
	big := w.addCreature(CreatureTrike, 11.9, 10.5)
	hp := big.HP
	stepN(w, ManEaterDigest+2*TicksPerSecond)
	if big.gone || big.HP >= hp {
		t.Fatalf("trike gone %v hp %d: want bitten, not swallowed", big.gone, big.HP)
	}
}
