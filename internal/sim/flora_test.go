package sim

import (
	"math"
	"slices"
	"testing"
)

// floraHome reports whether a plant of kind f belongs on tile t, with wet
// saying the tile touches water.
func floraHome(f FloraKind, t Tile) bool {
	homes := map[FloraKind][]Tile{
		FloraBamboo:         {TileForest, TileMarsh},
		FloraGourd:          {TileGrass, TileForest},
		FloraPitcher:        {TileMarsh, TileForest},
		FloraFlax:           {TileGrass},
		FloraVine:           {TileForest, TileMarsh},
		FloraPoisonMushroom: {TileForest, TileMarsh, TileCave},
		FloraNightshade:     {TileForest},
		FloraAloe:           {TileDesert, TileSand},
		FloraHerb:           {TileGrass, TileForest, TileRocky},
		FloraMoss:           {TileRocky, TileForest, TileMarsh, TileCave},
		FloraMangrove:       {TileWater, TileSand},
		FloraSeaweed:        {TileWater},
		FloraLily:           {TileWater},
		FloraLotus:          {TileWater},
		FloraHorsetail:      {TileGrass, TileForest, TileMarsh},
		FloraWildRice:       {TileGrass, TileMarsh},
		FloraWildCorn:       {TileGrass},
		FloraWildSquash:     {TileGrass},
	}
	return slices.Contains(homes[f], t)
}

func TestEveryPlantGrowsInItsOwnCountry(t *testing.T) {
	var total [NumFlora]int
	for _, seed := range []uint64{3, 9} {
		w := genOpen(t, seed, SizeMedium)
		for _, e := range w.entities {
			if e.Kind != EntityFlora {
				continue
			}
			f := FloraKind(e.Var)
			if f >= NumFlora {
				t.Fatalf("seed %d: a plant of kind %d", seed, f)
			}
			total[f]++
			tile := w.TileAt(e.TX, e.TY)
			if !floraHome(f, tile) {
				t.Errorf("seed %d: %s on tile %d at (%d, %d)", seed, f.Name(), tile, e.TX, e.TY)
			}
			if f.InWater() != (tile == TileWater) && f != FloraMangrove {
				t.Errorf("seed %d: %s in water: %v", seed, f.Name(), tile == TileWater)
			}
			switch f {
			case FloraLily, FloraLotus:
				if !w.pondWater(e.TX, e.TY) {
					t.Errorf("seed %d: %s not on a pond at (%d, %d)", seed, f.Name(), e.TX, e.TY)
				}
			case FloraSeaweed:
				if w.pondWater(e.TX, e.TY) || !w.isShallow(e.TX, e.TY) {
					t.Errorf("seed %d: seaweed at (%d, %d) is not in the shallows of the sea", seed, e.TX, e.TY)
				}
			case FloraWildRice:
				if !w.waterside(e.TX, e.TY) {
					t.Errorf("seed %d: wild rice away from the water at (%d, %d)", seed, e.TX, e.TY)
				}
			}
			if e.HP != e.fullHP() || !e.Ready {
				t.Errorf("seed %d: a new %s has %d hp, ready %v", seed, f.Name(), e.HP, e.Ready)
			}
		}
	}
	for f := FloraKind(0); f < NumFlora; f++ {
		if total[f] == 0 {
			t.Errorf("no %s in two worlds", f.Name())
		}
	}
}

func TestToadsLiveByThePondsAndInTheMarsh(t *testing.T) {
	w := genOpen(t, 9, SizeMedium)
	onPads := 0
	for _, k := range []CreatureKind{CreatureToad, CreatureGlowToad} {
		cs := creatureByKind(w, k)
		if len(cs) < creatureMinPerKind {
			t.Fatalf("%d %s in the world", len(cs), k.Name())
		}
		for _, c := range cs {
			tx, ty := int(math.Floor(c.X)), int(math.Floor(c.Y))
			if w.padAt(tx, ty) {
				onPads++
				continue
			}
			if !w.pondside(tx, ty) {
				t.Errorf("%s at (%.1f, %.1f), tile %d: not in the marsh nor by a pond", k.Name(), c.X, c.Y, w.TileAt(tx, ty))
			}
		}
	}
	if onPads == 0 {
		t.Error("no toad sits on a lily pad")
	}
}

// pondWorld is flatWorld with a pond of 3 x 3 tiles east of the spawn, a
// lily on its nearest tile.
func pondWorld() (*World, *Entity) {
	w := flatWorld()
	for ty := 15; ty <= 17; ty++ {
		for tx := 19; tx <= 21; tx++ {
			w.tiles[ty*w.width+tx] = TileWater
		}
	}
	return w, w.addFlora(FloraLily, 19, 16)
}

func TestToadsHopOntoLilyPadsButNotIntoTheWater(t *testing.T) {
	w, _ := pondWorld()
	if w.creatureFits(CreatureToad, 20.5, 16.5) {
		t.Fatal("a toad fits on open water")
	}
	if !w.creatureFits(CreatureToad, 19.5, 16.5) {
		t.Fatal("a toad does not fit on a lily pad")
	}
	if w.creatureFits(CreatureRabbit, 19.5, 16.5) {
		t.Fatal("a rabbit fits on a lily pad")
	}
	c := w.addCreature(CreatureToad, 18.5, 16.5)
	sat, hopped := false, false
	for range 6000 {
		w.Step()
		if c.State != CreatureWander && w.TileAt(int(math.Floor(c.X)), int(math.Floor(c.Y))) == TileWater && !w.onPad(c) {
			t.Fatalf("the toad is in the water at (%.2f, %.2f)", c.X, c.Y)
		}
		hopped = hopped || c.State == CreatureWander
		sat = sat || w.onPad(c) && c.State == CreatureIdle
	}
	if !hopped || !sat {
		t.Fatalf("hopped %v, sat on the pad %v", hopped, sat)
	}
}

func TestAHopIsOneLeapThenAPause(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureGlowToad, 10.5, 10.5)
	c.timer = 1
	var run, longest int
	x0, y0 := c.X, c.Y
	far := 0.0
	for range 2000 {
		w.Step()
		if c.State == CreatureWander {
			if run == 0 {
				x0, y0 = c.X, c.Y
			}
			run++
			longest = max(longest, run)
			far = math.Max(far, math.Hypot(c.X-x0, c.Y-y0))
		} else {
			run = 0
		}
	}
	if longest == 0 || longest > ToadHopTicks {
		t.Fatalf("the longest hop lasted %d ticks, want 1 to %d", longest, ToadHopTicks)
	}
	if far > ToadHop+toadHopStretch+1e-6 {
		t.Fatalf("a hop covered %.2f tiles, want at most %.2f", far, ToadHop+toadHopStretch)
	}
}

func TestTheGreenToadPoisonsWhoTouchesIt(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	glow := w.addCreature(CreatureGlowToad, p.X+0.4, p.Y)
	glow.timer = 1000
	w.Step()
	if p.Poisoned() {
		t.Fatal("the glow toad poisons")
	}
	glow.gone = true
	c := w.addCreature(CreatureToad, p.X+0.5, p.Y)
	c.timer = 1000
	w.Step()
	if !p.Poisoned() || p.Notice != NoticeBitten || CreatureKind(p.NoticeArg) != CreatureToad {
		t.Fatalf("poisoned %v, notice %d/%d after touching a green toad", p.Poisoned(), p.Notice, p.NoticeArg)
	}
	if c.State != CreatureStrike {
		t.Errorf("the toad is in state %d, want it swollen up", c.State)
	}
	before := p.Health
	stepN(w, 10)
	if p.Health >= before {
		t.Error("the poison does not eat at the player")
	}

	// A herb draws it out.
	c.gone = true
	p.Inv[3] = Slot{ItemHerb, 1}
	w.Apply(id, Command{Kind: CmdEat, Slot: 3})
	if p.Poisoned() || p.Notice != NoticeCured {
		t.Fatalf("poisoned %v, notice %d after a herb", p.Poisoned(), p.Notice)
	}
}

func TestKillingToadsGivesTheirGlands(t *testing.T) {
	for k, want := range map[CreatureKind][]ItemKind{
		CreatureToad:     {ItemToadSkin, ItemPoisonGland},
		CreatureGlowToad: {ItemGlowGland},
	} {
		w := flatWorld()
		c := w.addCreature(k, 10.5, 10.5)
		w.fell(c)
		var got []ItemKind
		for _, d := range w.drops {
			got = append(got, d.Item)
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s leaves %v, want %v", k.Name(), got, want)
		}
		if len(w.entities) != 0 {
			t.Errorf("%s leaves a body", k.Name())
		}
	}
}

func TestPoisonFoodPoisonsAndMedicineHeals(t *testing.T) {
	for _, item := range []ItemKind{ItemPoisonMushroom, ItemNightshade} {
		w := flatWorld()
		id := w.AddPlayer("a")
		p := w.players[id]
		p.Inv[3] = Slot{item, 1}
		w.Apply(id, Command{Kind: CmdEat, Slot: 3})
		if !p.Poisoned() || p.Notice != NoticeFoodPoison || ItemKind(p.NoticeArg) != item {
			t.Fatalf("%s: poisoned %v, notice %d", item.Def().Name, p.Poisoned(), p.Notice)
		}
		if p.Health < 1 {
			t.Fatalf("%s killed outright", item.Def().Name)
		}
		// Moss mends but does not cure; a herb cures.
		p.Health = 50
		p.Inv[4] = Slot{ItemMoss, 1}
		w.Apply(id, Command{Kind: CmdEat, Slot: 4})
		if !p.Poisoned() || p.Health != 50+ItemMoss.Def().Health {
			t.Fatalf("after moss: poisoned %v, health %.1f", p.Poisoned(), p.Health)
		}
		p.Inv[4] = Slot{ItemHerb, 1}
		w.Apply(id, Command{Kind: CmdEat, Slot: 4})
		if p.Poisoned() {
			t.Fatal("a herb did not draw the poison out")
		}
	}
	for _, item := range []ItemKind{ItemAloe, ItemHerb, ItemMoss, ItemPitcherWater} {
		if !item.Healing() {
			t.Errorf("%s is not a medicine", item.Def().Name)
		}
	}
}

func TestPickingPlants(t *testing.T) {
	w := flatWorld()
	w.dev.weather, w.dev.forced = WeatherClear, true // no lightning to shout over the notices
	id := w.AddPlayer("a")                           // on (16, 16)
	p := w.players[id]
	p.dev.god = true // the days of waiting would starve them
	herb := w.addFlora(FloraHerb, 17, 16)
	w.SetInput(id, Input{Action: true})
	w.Step()
	if p.Inv.Count(ItemHerb) != 2 || herb.Ready {
		t.Fatalf("%d herbs, the plant still full %v", p.Inv.Count(ItemHerb), herb.Ready)
	}
	w.SetInput(id, Input{})
	stepN(w, DayTicks+1)
	if !herb.Ready {
		t.Fatal("the herb did not grow back in a day")
	}
	w.removeEntity(herb)

	// Bamboo wants an axe, and two blows of a stone one.
	bamboo := w.addFlora(FloraBamboo, 17, 16)
	w.SetInput(id, Input{Action: true})
	stepN(w, 3*ActionCooldownTicks)
	if p.Inv.Count(ItemBamboo) != 0 || p.Notice != NoticeNeedTool {
		t.Fatalf("%d bamboo by hand, notice %d", p.Inv.Count(ItemBamboo), p.Notice)
	}
	p.Inv.Add(ItemAxe, 1)
	w.Step()
	if p.Inv.Count(ItemBamboo) != 0 {
		t.Fatal("bamboo fell to one blow")
	}
	stepN(w, 2*ActionCooldownTicks)
	if p.Inv.Count(ItemBamboo) != 3 || bamboo.Ready {
		t.Fatalf("%d bamboo after two blows", p.Inv.Count(ItemBamboo))
	}
	w.SetInput(id, Input{})
	stepN(w, 2*DayTicks)
	if !bamboo.Ready || bamboo.HP != 2*HitPower {
		t.Fatalf("regrown bamboo: ready %v, hp %d", bamboo.Ready, bamboo.HP)
	}

	// A squash plant gives its seeds too.
	w.removeEntity(bamboo)
	w.addFlora(FloraWildSquash, 17, 16)
	w.SetInput(id, Input{Action: true})
	stepN(w, ActionCooldownTicks+1)
	if p.Inv.Count(ItemSquash) != 1 || p.Inv.Count(ItemSquashSeed) != SquashSeedYield {
		t.Fatalf("%d squash, %d seeds", p.Inv.Count(ItemSquash), p.Inv.Count(ItemSquashSeed))
	}
}

func TestLiliesArePickedFromTheBank(t *testing.T) {
	w, lily := pondWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X = 18.5
	w.SetInput(id, Input{Action: true})
	w.Step()
	if p.Inv.Count(ItemLily) != 1 || lily.Ready {
		t.Fatalf("%d lilies picked from the bank", p.Inv.Count(ItemLily))
	}
	if !w.padAt(19, 16) {
		t.Error("the leaves went with the flower")
	}
}

func TestPitcherPlantsFillInTheRain(t *testing.T) {
	w := flatWorld()
	e := w.addFlora(FloraPitcher, 5, 5)
	e.Ready, e.RegrowAt = false, w.tick+2*DayTicks
	w.watch(e)
	w.dev.weather, w.dev.forced = WeatherRain, true
	stepN(w, pitcherRainTicks+2)
	if !e.Ready {
		t.Fatal("a pitcher plant stayed empty in the rain")
	}
}

func TestCropsOfEveryKindGrowAndAreReaped(t *testing.T) {
	for c := CropKind(0); c < NumCrops; c++ {
		d := c.def()
		w := flatWorld()
		id := w.AddPlayer("a")
		p := w.players[id]
		p.dev.god = true // growing takes a day and a night
		p.Dir = DirRight
		bed := w.till(17, 16)
		p.Inv.Add(ItemPotato, 1) // carried, but not what is held
		holding(w, id, d.seed)
		p.Inv[0].Count = 2
		w.SetInput(id, Input{Action: true})
		w.Step()
		w.SetInput(id, Input{})
		if bed.Kind != EntityCrop || CropKind(bed.Var) != c || bed.Ready {
			t.Fatalf("%s: sown bed is %+v", d.name, *bed)
		}
		if p.Inv[0].Count != 1 {
			t.Fatalf("%s: %d seeds left of 2", d.name, p.Inv[0].Count)
		}
		if EntityLabel(bed.Kind, bed.Var) != d.name {
			t.Errorf("%s is labelled %q", d.name, EntityLabel(bed.Kind, bed.Var))
		}
		// It goes through every stage, in order, and is ripe on time.
		last := 0
		for i := 0; i < d.grow-2; i++ {
			w.Step()
			s := bed.GrowthStage(w.tick)
			if s < last || s >= CropStages {
				t.Fatalf("%s: stage %d after stage %d", d.name, s, last)
			}
			last = s
		}
		if last != CropStages-1 || bed.Ready {
			t.Fatalf("%s: stage %d, ripe %v just before its time", d.name, last, bed.Ready)
		}
		stepN(w, 3)
		if !bed.Ready {
			t.Fatalf("%s: not ripe after %d ticks", d.name, d.grow)
		}
		have := p.Inv.Count(d.item)
		w.SetInput(id, Input{Action: true})
		w.Step()
		if got := p.Inv.Count(d.item) - have; got != d.yield {
			t.Fatalf("%s: reaped %d, want %d", d.name, got, d.yield)
		}
		if d.bonus.Valid() && p.Inv.Count(d.bonus) < d.bonusN {
			t.Fatalf("%s: %d %s", d.name, p.Inv.Count(d.bonus), d.bonus.Def().Name)
		}
		if bed.Kind != EntityFarmland || bed.Var != 0 {
			t.Fatalf("%s: the bed after reaping is %+v", d.name, *bed)
		}
	}
}

func TestWaterPlantsAndCropsSurviveASave(t *testing.T) {
	w := genOpen(t, 9, SizeMedium)
	var before [NumFlora]int
	for _, e := range w.entities {
		if e.Kind == EntityFlora {
			before[e.Var]++
		}
	}
	d := w.Save()
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	var after [NumFlora]int
	for _, e := range w2.entities {
		if e.Kind == EntityFlora {
			after[e.Var]++
		}
	}
	if before != after {
		t.Fatalf("plants before the save %v, after %v", before, after)
	}

	// A world saved before the plants existed gets them on loading.
	d.Version = 14
	d.Entities = slices.DeleteFunc(d.Entities, func(e Entity) bool { return e.Kind == EntityFlora })
	w3, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range w3.entities {
		if e.Kind == EntityFlora {
			n++
		}
	}
	if n == 0 {
		t.Fatal("an older save loads with none of the new plants")
	}
}

func TestNewRecipes(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv.Add(ItemFlax, 4)
	p.Inv.Add(ItemVine, 2)
	p.Inv.Add(ItemMoss, 2)
	p.Inv.Add(ItemBamboo, 11)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeRope})
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeVineRope})
	if p.Inv.Count(ItemRope) != 2 {
		t.Fatalf("%d ropes from flax and vines", p.Inv.Count(ItemRope))
	}
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeMossBandage})
	if p.Inv.Count(ItemBandage) != 1 {
		t.Fatal("no bandage of moss and flax")
	}
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeBambooRaft})
	if p.Inv.Count(ItemRaft) != 1 || p.Inv.Count(ItemRope) != 0 {
		t.Fatal("no raft of bamboo and rope")
	}
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeBambooWall})
	found := false
	for _, e := range w.entities {
		found = found || e.Kind == EntityWall
	}
	if !found || p.Inv.Count(ItemBamboo) != 0 {
		t.Fatalf("a bamboo wall stands: %v; %d bamboo left", found, p.Inv.Count(ItemBamboo))
	}
	for _, name := range []string{"bamboo", "tre", "thảo dược", "herb", "hạt sen", "rope"} {
		if _, ok := FindItem(name); !ok {
			t.Errorf("the dev console does not know the item %q", name)
		}
	}
	for name, want := range map[string]CreatureKind{"toad": CreatureToad, "glowtoad": CreatureGlowToad, "cóc độc": CreatureToad, "cóc phát sáng": CreatureGlowToad} {
		if k, ok := FindCreature(name); !ok || k != want {
			t.Errorf("the dev console takes %q for %d", name, k)
		}
	}
}
