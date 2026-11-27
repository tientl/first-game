package sim

import (
	"math"
	"testing"
)

// holdTent gives p one stack of k in the first hotbar slot, held.
func holdTent(p *Player, k ItemKind, n int) {
	p.Inv[0] = Slot{Item: k, Count: n}
	p.Selected = 0
}

func TestLeafTentIsMadeByHand(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	r, ok := RecipeByID(RecipeLeafTent)
	if !ok || r.Gives != ItemLeafTent || r.NearFire || r.NearFurnace {
		t.Fatalf("recipe %+v: want a leaf tent made by hand", r)
	}
	p.Inv = InventoryWith(r.Inputs...)
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeLeafTent})
	if p.Inv.Count(ItemLeafTent) != 1 || p.Inv.Count(ItemStick) != 0 || p.Inv.Count(ItemGrass) != 0 || p.Inv.Count(ItemFlax) != 0 {
		t.Fatalf("after crafting: %v, want one tent and nothing else", p.Inv)
	}
	// Cheap: no wood, so no axe is needed for it, unlike the bed.
	for _, in := range r.Inputs {
		if in.Item == ItemWood || in.Item == ItemStone {
			t.Errorf("a tent takes %s", in.Item.Def().Name)
		}
	}
	if k, ok := FindItem("tent"); !ok || k != ItemLeafTent {
		t.Error("the dev console does not know a tent")
	}
	if k, ok := FindItem("lều lá"); !ok || k != ItemLeafTent {
		t.Error("the dev console does not know Lều lá")
	}
}

func TestLeafTentIsSetDownOnSoftGroundOnly(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	holdTent(p, ItemLeafTent, 2)

	// Bare rock and a ford: no stake goes in.
	w.tiles[16*w.width+17] = TileRocky
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 17, TY: 16})
	if _, ok := w.EntityAt(17, 16); ok || p.Notice != NoticeHardGround || p.Inv.Count(ItemLeafTent) != 2 {
		t.Fatalf("on rock: placed %v, notice %v, %d tents left", ok, p.Notice, p.Inv.Count(ItemLeafTent))
	}
	// Nor in the water.
	w.tiles[16*w.width+15] = TileWater
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 15, TY: 16})
	if _, ok := w.EntityAt(15, 16); ok || p.Notice != NoticeNoSpace {
		t.Fatalf("in the water: placed %v, notice %v", ok, p.Notice)
	}
	// On grass it stands.
	w.Apply(id, Command{Kind: CmdPlace, At: true, TX: 16, TY: 17})
	if e, ok := w.EntityAt(16, 17); !ok || e.Kind != EntityLeafTent || p.Inv.Count(ItemLeafTent) != 1 {
		t.Fatalf("on grass: %+v, %d tents left", e, p.Inv.Count(ItemLeafTent))
	}
	// V with no tile chosen sets it in front, and on snow and sand too.
	w.tiles[16*w.width+17] = TileSnow
	p.Dir = DirRight
	w.Apply(id, Command{Kind: CmdPlace})
	if e, ok := w.EntityAt(17, 16); !ok || e.Kind != EntityLeafTent || p.Inv.Count(ItemLeafTent) != 0 {
		t.Fatalf("in front on the snow: %+v", e)
	}
	if !TentGround(TileSand) || !TentGround(TileDesert) || TentGround(TileWater) || TentGround(TileCave) {
		t.Error("TentGround: want sand and desert, not water or a cave floor")
	}
}

func TestLeafTentSleepsTheNightThroughAndIsHome(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	tent := w.addEntity(EntityLeafTent, 17, 16)

	w.SetInput(id, Input{Action: true})
	w.Step()
	if p := mustPlayer(t, w, id); p.Asleep || p.Notice != NoticeNotTired || p.NoticeArg != uint8(EntityLeafTent) {
		t.Fatalf("by day: asleep %v notice %v (%d), want not tired in a tent", p.Asleep, p.Notice, p.NoticeArg)
	}

	w.tick = deepNight
	w.players[id].Health = 50
	stepN(w, ActionCooldownTicks)
	w.SetInput(id, Input{})
	p := mustPlayer(t, w, id)
	if !p.Asleep || p.X != 17.5 || p.Y != 16.5 || !w.AllAsleep() {
		t.Fatalf("at night: asleep %v at %.2f,%.2f, all asleep %v", p.Asleep, p.X, p.Y, w.AllAsleep())
	}
	if w.players[id].HomeBed() != tent.ID {
		t.Fatal("sleeping in the tent did not make it home")
	}
	stepN(w, 100)
	if p := mustPlayer(t, w, id); p.Cold || p.Health <= 50 {
		t.Fatalf("in the tent: cold %v health %.1f, want warm and healing", p.Cold, p.Health)
	}
	for w.players[id].Asleep && w.tick < deepNight+DayTicks {
		w.Step()
	}
	if f := DayFraction(w.tick); w.players[id].Asleep || f > 0.1 {
		t.Fatalf("still asleep %v at day fraction %.2f", w.players[id].Asleep, f)
	}

	// Killed far away, they come back by the tent.
	pp := w.players[id]
	pp.X, pp.Y = 4.5, 4.5
	hurt(pp, 1000)
	w.Apply(id, Command{Kind: CmdRespawn})
	if math.Hypot(pp.X-17.5, pp.Y-16.5) > 1.5 {
		t.Fatalf("came back at (%.1f, %.1f), want by the tent", pp.X, pp.Y)
	}
}

func TestLeafTentIsTakenDownByDayWithAnAxe(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.addEntity(EntityLeafTent, 17, 16)

	// At night an axe in the hand does not stop a sleep.
	holdTent(p, ItemAxe, 1)
	w.tick = deepNight
	w.SetInput(id, Input{Action: true})
	stepN(w, ActionCooldownTicks)
	w.SetInput(id, Input{})
	if !p.Asleep {
		t.Fatal("with an axe at night: not asleep")
	}
	p.wake()
	if _, ok := w.EntityAt(17, 16); !ok {
		t.Fatal("the tent came down at night")
	}

	// By day with nothing in the hand: not tired, and it stands.
	w.tick = uint64(2*DayTicks) + uint64(0.2*DayTicks)
	p.Selected = 1
	w.SetInput(id, Input{Action: true})
	stepN(w, 3*ActionCooldownTicks)
	if _, ok := w.EntityAt(17, 16); !ok || p.Notice != NoticeNotTired {
		t.Fatalf("by day bare-handed: notice %v", p.Notice)
	}

	// By day with the axe held it comes down, its sticks and thatch on the ground.
	p.Selected = 0
	stepN(w, TentHits*ActionCooldownTicks+1)
	w.SetInput(id, Input{})
	if _, ok := w.EntityAt(17, 16); ok {
		t.Fatal("tent still standing")
	}
	got := map[ItemKind]int{}
	for _, d := range w.Drops() {
		got[d.Item] += d.Count
	}
	for _, in := range bedRefund(EntityLeafTent) {
		if got[in.Item]+p.Inv.Count(in.Item) != in.Count {
			t.Errorf("%s back: %d on the ground, %d in the bag, want %d", in.Item.Def().Name, got[in.Item], p.Inv.Count(in.Item), in.Count)
		}
	}

	// A bed comes down the same way now.
	w.drops = nil // else Action picks the piles up first
	w.addEntity(EntityBed, 17, 16)
	w.SetInput(id, Input{Action: true})
	stepN(w, BedHits*ActionCooldownTicks+1)
	if _, ok := w.EntityAt(17, 16); ok {
		t.Fatal("bed still standing")
	}
}

func TestLeafTentIsSaved(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("Tiến")
	p := w.players[id]
	// A free patch of soft ground by the player.
	tx, ty := -1, -1
	px, py := int(p.X), int(p.Y)
	for r := 1; r < 12 && tx < 0; r++ {
		for y := py - r; y <= py+r && tx < 0; y++ {
			for x := px - r; x <= px+r; x++ {
				if w.canPlace(x, y) && TentGround(w.TileAt(x, y)) && w.standable(float64(x)+0.5, float64(y)+0.5) {
					tx, ty = x, y
					break
				}
			}
		}
	}
	if tx < 0 {
		t.Fatal("nowhere to pitch a tent")
	}
	tent := w.addEntity(EntityLeafTent, tx, ty)
	w.tick = uint64(DayTicks*NightStart) + DayTicks
	if !w.sleep(p, tent) {
		t.Fatal("could not sleep in the tent")
	}
	p.Inv.Add(ItemLeafTent, 2) // and another rolled up in the bag

	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	if e := w2.entities[tent.ID]; e == nil || e.Kind != EntityLeafTent || e.TX != tx || e.TY != ty {
		t.Fatalf("tent after load: %+v", e)
	}
	p2 := w2.players[w2.AddPlayer("Tiến")]
	if p2.HomeBed() != tent.ID {
		t.Fatalf("home after load = %d, want the tent %d", p2.HomeBed(), tent.ID)
	}
	if p2.Inv.Count(ItemLeafTent) != 2 {
		t.Fatalf("tents in the bag after load: %d, want 2", p2.Inv.Count(ItemLeafTent))
	}
}

func TestLeafTentKeepsTheWoodsBack(t *testing.T) {
	w := flatWorld()
	if !builtKind(EntityLeafTent, 0) {
		t.Fatal("a tent is not something built")
	}
	w.addEntity(EntityLeafTent, 10, 10)
	keep := w.spared()
	if !keep[10*w.width+10+thinNearBuilt-1] || !keep[(10+thinNearBuilt-1)*w.width+10] {
		t.Error("thinning old woods would take trees by the tent")
	}
	if !w.nearWorks(10+thinNearBuilt-1, 10) {
		t.Error("the woods would grow back over the tent")
	}
	if w.nearWorks(10+thinNearBuilt+3, 10+thinNearBuilt+3) {
		t.Error("the woods are kept back too far from the tent")
	}
}
