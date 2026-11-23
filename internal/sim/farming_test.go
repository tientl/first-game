package sim

import "testing"

// holding puts item in hotbar slot 0 and selects it.
func holding(w *World, id PlayerID, item ItemKind) {
	p := w.players[id]
	p.Inv[0] = Slot{Item: item, Count: 1}
	p.Selected = 0
}

func TestPebblesArePickedByHand(t *testing.T) {
	w := flatWorld()
	pebble := w.addEntity(EntityPebble, 17, 16)
	id := w.AddPlayer("a")
	w.SetInput(id, Input{Action: true})
	w.Step()
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemPebble) != PebbleYield || pebble.Ready {
		t.Fatalf("pebbles %d, still lying there %v: want it picked in one go", p.Inv.Count(ItemPebble), pebble.Ready)
	}
	w.RemovePlayer(id)
	stepN(w, PebbleRegrowTicks)
	if !pebble.Ready {
		t.Error("no pebble turned up again")
	}
}

func TestTheFirstToolsComeFromTheGround(t *testing.T) {
	for _, id := range []RecipeID{RecipeAxe, RecipePickaxe, RecipeHoe} {
		r, _ := RecipeByID(id)
		for _, in := range r.Inputs {
			if in.Item != ItemStick && in.Item != ItemPebble {
				t.Errorf("%s needs %s, which is not picked up from the ground", r.Name, in.Item.Def().Name)
			}
		}
	}
}

func TestHoeTillsTheSoilInFront(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a") // on (16, 16)
	w.players[id].Dir = DirRight
	w.players[id].Inv.Add(ItemHoe, 1)
	w.players[id].Selected = 9 // the hoe is in the bag, not in hand
	w.SetInput(id, Input{Action: true})
	w.Step()
	if _, ok := w.EntityAt(17, 16); ok {
		t.Fatal("tilled without holding the hoe")
	}

	holding(w, id, ItemHoe)
	w.Step() // no target before, so no cooldown either
	e, ok := w.EntityAt(17, 16)
	if !ok || e.Kind != EntityFarmland {
		t.Fatalf("in front of the player: %+v, want tilled soil", e)
	}
	if !w.players[id].Acting(w.Tick()) {
		t.Error("tilling shows no swing")
	}

	// Rocky ground and taken tiles are not tilled.
	w.tiles[16*w.width+15] = TileRocky
	w.addEntity(EntityGrassTuft, 16, 17)
	for _, d := range []Dir{DirLeft, DirDown} {
		w.players[id].Dir = d
		dx, dy := d.Delta()
		before, _ := w.EntityAt(16+dx, 16+dy)
		if w.dig(w.players[id]) {
			t.Errorf("tilled (%d, %d)", 16+dx, 16+dy)
		}
		if after, _ := w.EntityAt(16+dx, 16+dy); after != before {
			t.Errorf("(%d, %d) changed from %+v to %+v", 16+dx, 16+dy, before, after)
		}
	}
}

func TestTillingTurnsUpWormsAndSweetPotatoes(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	holding(w, id, ItemHoe)
	p := w.players[id]
	p.Dir = DirRight
	const n = 400
	for range n {
		if !w.dig(p) {
			t.Fatal("could not till the free grass in front")
		}
		e := w.entityAt(17, 16)
		w.removeEntity(e) // the same tile again
	}
	worms, potatoes := p.Inv.Count(ItemWorm), p.Inv.Count(ItemPotato)
	if worms < n*DigWormChance/2 || worms > n*DigWormChance*2 || potatoes < n*DigPotatoChance/2 || potatoes > n*DigPotatoChance*2 {
		t.Errorf("%d tillings found %d worms and %d sweet potatoes, want about %v and %v",
			n, worms, potatoes, n*DigWormChance, n*DigPotatoChance)
	}
}

func TestSweetPotatoesGrowFromPlantedOnes(t *testing.T) {
	w := flatWorld()
	bed := w.till(17, 16)
	id := w.AddPlayer("a")
	press := func() {
		w.SetInput(id, Input{Action: true})
		w.Step()
		w.SetInput(id, Input{})
		stepN(w, ActionCooldownTicks)
	}

	press()
	if p := mustPlayer(t, w, id); bed.Kind != EntityFarmland || p.Notice != NoticeNeedSeed || ItemKind(p.NoticeArg) != ItemPotato {
		t.Fatalf("planting with nothing to plant: bed %v, notice %v/%d", bed.Kind, p.Notice, p.NoticeArg)
	}

	w.players[id].Inv.Add(ItemPotato, 1)
	press()
	if p := mustPlayer(t, w, id); bed.Kind != EntityCrop || bed.Ready || p.Inv.Count(ItemPotato) != 0 {
		t.Fatalf("after planting: bed %v ready %v, sweet potatoes left %d", bed.Kind, bed.Ready, p.Inv.Count(ItemPotato))
	}
	if Interactable(bed.Kind, bed.Ready, false) {
		t.Error("a growing crop can be dug up")
	}

	w.RemovePlayer(id) // keep the wait free of night-time deaths
	stage := bed.GrowthStage(w.Tick())
	stepN(w, CropGrowTicks/2)
	if s := bed.GrowthStage(w.Tick()); s <= stage || bed.Ready {
		t.Fatalf("half grown: stage %d (was %d), ripe %v", s, stage, bed.Ready)
	}
	stepN(w, CropGrowTicks/2)
	if !bed.Ready {
		t.Fatal("not ripe after CropGrowTicks")
	}

	id = w.AddPlayer("a")
	press()
	if p := mustPlayer(t, w, id); p.Inv.Count(ItemPotato) != CropYield || bed.Kind != EntityFarmland {
		t.Fatalf("after digging up: %d sweet potatoes, bed %v; want %d and empty soil", p.Inv.Count(ItemPotato), bed.Kind, CropYield)
	}
}

func TestEmptySoilTurnsBackToGrass(t *testing.T) {
	w := flatWorld()
	w.till(17, 16)
	stepN(w, FarmlandDecayTicks-1)
	if _, ok := w.EntityAt(17, 16); !ok {
		t.Fatal("tilled soil gone too early")
	}
	stepN(w, 1)
	if _, ok := w.EntityAt(17, 16); ok {
		t.Fatal("empty soil still there after FarmlandDecayTicks")
	}
}

func TestGeneratedWorldsHavePebbles(t *testing.T) {
	for _, seed := range testSeeds {
		w, err := Generate(GenConfig{Seed: seed, Width: 128, Height: 128})
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, e := range w.entities {
			if e.Kind == EntityPebble {
				n++
			}
		}
		if n < 30 {
			t.Errorf("seed %d: only %d pebbles", seed, n)
		}
	}
}

func TestOlderSavesGetPebblesAndKeepMining(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 5, Width: 48, Height: 48})
	if err != nil {
		t.Fatal(err)
	}
	pebbles := 0
	d := w.Save()
	d.Version = 4
	kept := d.Entities[:0]
	for _, e := range d.Entities {
		if e.Kind == EntityPebble { // a version 4 world had none
			pebbles++
			continue
		}
		kept = append(kept, e)
	}
	d.Entities = kept
	d.Players = []PlayerSave{{Name: "Cũ", X: w.spawnX, Y: w.spawnY, Health: 80, Hunger: 50,
		Slots: InventoryWith(Ingredient{ItemStone, 3})}}

	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for _, e := range w2.entities {
		if e.Kind == EntityPebble {
			got++
		}
	}
	if got != pebbles {
		t.Errorf("loaded %d pebbles, want the %d the world generates", got, pebbles)
	}
	if p := mustPlayer(t, w2, w2.AddPlayer("Cũ")); p.Inv.Count(ItemPickaxe) != 1 || p.Inv.Count(ItemStone) != 3 {
		t.Errorf("old player: %d pickaxes, %d stone; want a pickaxe added to the 3 stone", p.Inv.Count(ItemPickaxe), p.Inv.Count(ItemStone))
	}
}

func TestCropsSurviveASave(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	sx, sy := int(w.spawnX), int(w.spawnY)
	bed := w.till(sx+1, sy)
	bed.Kind, bed.Ready, bed.RegrowAt = EntityCrop, false, w.tick+100
	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	if got := *w2.entities[bed.ID]; got != *bed {
		t.Errorf("crop after load = %+v, want %+v", got, *bed)
	}
}
