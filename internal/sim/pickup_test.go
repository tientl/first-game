package sim

import "testing"

// pileAt lays a pile of n of item at (x, y), as dropAt would but just there.
func pileAt(w *World, item ItemKind, n int, x, y float64) *Drop {
	w.nextDrop++
	d := &Drop{ID: w.nextDrop, Item: item, Count: n, X: x, Y: y}
	w.drops = append(w.drops, d)
	return d
}

// In the middle of a fight the left button strikes the beast, whatever
// spoils lie about and wherever the pointer is: it never picks up.
func TestActionStrikesTheBeastNotThePile(t *testing.T) {
	for _, aim := range []bool{false, true} {
		w := flatWorld() // the player stands at (16.5, 16.5)
		id := w.AddPlayer("a")
		p := w.players[id]
		raptor := w.addCreature(CreatureRaptor, 17.6, 16.5)
		meat := pileAt(w, ItemMeat, 2, 16.2, 17.2) // in reach, but not walked over
		twig := w.addEntity(EntityTwig, 16, 15)
		hp := raptor.HP
		in := Input{Action: true}
		if aim {
			in = aimAt(meat.X, meat.Y) // the pointer right on the pile
		}
		w.SetInput(id, in)
		w.Step()
		if raptor.HP >= hp && !raptor.gone {
			t.Errorf("aim %v: the raptor took no blow (hp %d of %d)", aim, raptor.HP, hp)
		}
		if p.Inv.Count(ItemMeat) != 0 || meat.gone || len(w.drops) != 1 || !twig.Ready || p.Inv.Count(ItemStick) != 0 {
			t.Errorf("aim %v: meat %d in the bag, pile gone %v, twig there %v: the left button picked something up", aim, p.Inv.Count(ItemMeat), meat.gone, twig.Ready)
		}
		if !p.Acting(w.Tick()) {
			t.Errorf("aim %v: no swing", aim)
		}
	}
}

// With only things to pick up in reach, the left button does nothing at
// all: no pick-up, no swing, no walk to them.
func TestActionPicksNothingUp(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	twig := w.addEntity(EntityTwig, 17, 16)
	pebble := w.addEntity(EntityPebble, 15, 16)
	mushroom := w.addEntity(EntityMushroom, 16, 15)
	stone := pileAt(w, ItemStone, 2, 16.5, 17.3)
	for _, in := range []Input{{Action: true}, aimAt(17.5, 16.6), aimAt(stone.X, stone.Y), aimAt(15.5, 16.6)} {
		w.SetInput(id, in)
		stepN(w, TicksPerSecond)
		w.SetInput(id, Input{})
		w.Step()
	}
	if p.Inv.Count(ItemStick)+p.Inv.Count(ItemPebble)+p.Inv.Count(ItemMushroom)+p.Inv.Count(ItemStone) != 0 {
		t.Fatalf("bag %v: the left button picked something up", p.Inv)
	}
	if !twig.Ready || !pebble.Ready || !mushroom.Ready || stone.gone || stone.Count != 2 {
		t.Fatal("something was taken off the ground by the left button")
	}
	if p.X != 16.5 || p.Y != 16.5 || p.actedAt != 0 {
		t.Errorf("at (%.2f, %.2f), acted at %d: the left button with nothing to work on did something", p.X, p.Y, p.actedAt)
	}
	if tg, _ := w.findTarget(p); tg != nil {
		t.Errorf("the left button's target is a %v", tg.Kind)
	}
}

// E takes what is under the pointer first, a loose thing before a nearer
// pile; then the nearest pile, then the nearest loose thing.
func TestEPicksUpThePointedThenTheNearest(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	twig := w.addEntity(EntityTwig, 15, 16)      // 1 tile west
	pebble := w.addEntity(EntityPebble, 17, 17)  // the far corner of reach
	stone := pileAt(w, ItemStone, 1, 17.1, 16.5) // nearer than either
	w.SetInput(id, Input{Aim: true, AimX: 15.5, AimY: 16.6})
	w.Step() // off tick 0, when nobody is acting
	w.Apply(id, Command{Kind: CmdInteract})
	if p.Inv.Count(ItemStick) != TwigYield || twig.Ready || p.Inv.Count(ItemStone) != 0 {
		t.Fatalf("sticks %d stone %d: want the twig pointed at, not the nearer pile", p.Inv.Count(ItemStick), p.Inv.Count(ItemStone))
	}
	if !p.Acting(w.Tick()) || p.Dir != DirLeft {
		t.Errorf("acting %v dir %v: want a reach to the left", p.Acting(w.Tick()), p.Dir)
	}
	w.Apply(id, Command{Kind: CmdInteract}) // nothing left under the pointer: the nearest pile
	if p.Inv.Count(ItemStone) != 1 || !stone.gone || p.Inv.Count(ItemPebble) != 0 {
		t.Fatalf("stone %d pebbles %d: want the nearest pile", p.Inv.Count(ItemStone), p.Inv.Count(ItemPebble))
	}
	w.Apply(id, Command{Kind: CmdInteract}) // then the nearest loose thing
	if p.Inv.Count(ItemPebble) != PebbleYield || pebble.Ready {
		t.Fatalf("pebbles %d: want the nearest loose thing", p.Inv.Count(ItemPebble))
	}
	w.Apply(id, Command{Kind: CmdInteract})
	if p.Notice != NoticeNothingNear {
		t.Fatalf("notice %v, want nothing near", p.Notice)
	}
}

// E held down (CmdPickUp, sent by the client every PickCooldown) picks up
// one thing after another, piles and loose things, and nothing else: the
// chest beside stays shut and there is no notice when all is taken.
func TestHoldingEPicksUpOneAfterAnother(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.addEntity(EntityChest, 15, 16)
	pileAt(w, ItemWood, 2, 17.2, 17.3)
	pileAt(w, ItemWood, 1, 16.5, 17.4)
	pileAt(w, ItemMeat, 1, 15.9, 17.3)
	twig := w.addEntity(EntityTwig, 17, 16)
	w.Apply(id, Command{Kind: CmdInteract}) // the press: the nearest pile
	if p.Inv.Count(ItemWood) != 1 || p.chest != 0 {
		t.Fatalf("wood %d chest %d after the press: want the nearest pile", p.Inv.Count(ItemWood), p.chest)
	}
	for range 4 {
		stepN(w, PickCooldown)
		w.Apply(id, Command{Kind: CmdPickUp})
	}
	if p.Inv.Count(ItemWood) != 3 || p.Inv.Count(ItemMeat) != 1 || p.Inv.Count(ItemStick) != TwigYield || twig.Ready || len(w.drops) != 0 {
		t.Fatalf("wood %d meat %d sticks %d, %d piles left: want everything picked up", p.Inv.Count(ItemWood), p.Inv.Count(ItemMeat), p.Inv.Count(ItemStick), len(w.drops))
	}
	seq := p.NoticeSeq
	w.Apply(id, Command{Kind: CmdPickUp})
	if p.chest != 0 || p.NoticeSeq != seq {
		t.Errorf("chest %d, notice %v: E held down with nothing left did something else", p.chest, p.Notice)
	}
	w.Apply(id, Command{Kind: CmdInteract}) // a fresh press opens the chest
	if p.chest == 0 {
		t.Error("E with nothing to pick up did not open the chest beside")
	}
}

// A chest pointed at is opened before the nearest pile is picked up, as
// before; without the pointer on it, the pile comes first.
func TestEOpensThePointedChestBeforeTheNearestPile(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	chest := w.addEntity(EntityChest, 15, 16)
	pileAt(w, ItemWood, 1, 17.1, 16.5)
	w.Apply(id, Command{Kind: CmdInteract})
	if p.Inv.Count(ItemWood) != 1 || p.chest != 0 {
		t.Fatalf("wood %d chest %d: E without the pointer should pick up the pile first", p.Inv.Count(ItemWood), p.chest)
	}
	pileAt(w, ItemWood, 1, 17.1, 16.5)
	w.SetInput(id, Input{Aim: true, AimX: 15.5, AimY: 16.4})
	w.Apply(id, Command{Kind: CmdInteract})
	if p.chest != chest.ID || p.Inv.Count(ItemWood) != 1 {
		t.Fatalf("chest %d wood %d: E on the chest should open it", p.chest, p.Inv.Count(ItemWood))
	}
}

// Astride a beast, E picks up what is pointed at; with nothing pointed at
// it gets off, as it always did, whatever lies near.
func TestERidingPicksUpOnlyWhatIsPointedAt(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	mount := tamedNear(w, id, CreatureTrike, 17.6, 16.5)
	p.Inv.Add(ItemSaddle, 1)
	w.Apply(id, Command{Kind: CmdRide})
	if p.Mount() != mount.ID {
		t.Fatal("not riding")
	}
	near := pileAt(w, ItemStone, 1, p.X-0.7, p.Y+0.6)
	far := pileAt(w, ItemWood, 1, p.X+0.6, p.Y+0.7)
	w.SetInput(id, Input{Aim: true, AimX: float32(far.X), AimY: float32(far.Y)})
	w.Apply(id, Command{Kind: CmdInteract})
	if !far.gone || near.gone || p.Mount() != mount.ID {
		t.Fatalf("pointed-at pile gone %v, nearest gone %v, mount %d: want the pointed one picked from the saddle", far.gone, near.gone, p.Mount())
	}
	w.Apply(id, Command{Kind: CmdPickUp}) // held: still only what is pointed at
	if near.gone {
		t.Fatal("E held down in the saddle picked up the nearest pile")
	}
	w.SetInput(id, Input{})
	w.Apply(id, Command{Kind: CmdInteract})
	if p.Mount() != 0 || near.gone {
		t.Fatalf("mount %d, nearest pile gone %v: E with nothing pointed at should get off", p.Mount(), near.gone)
	}
}

// The rule the client marks and names is the one the server keeps.
func TestPickUpTargetIsTheRule(t *testing.T) {
	w := flatWorld()
	twig := w.addEntity(EntityTwig, 15, 16)
	reed := w.addEntity(EntityReed, 17, 17)
	stone := pileAt(w, ItemStone, 1, 17.0, 16.3)
	drops := w.dropCandidates(16.5, 16.5)
	for _, c := range []struct {
		name    string
		aim     bool
		ax, ay  float64
		seated  bool
		want    Pickup
		wantAny bool
	}{
		{"no pointer: the nearest pile", false, 0, 0, false, Pickup{Drop: stone.ID}, true},
		{"on the twig", true, 15.5, 16.6, false, Pickup{Entity: twig.ID, Aimed: true}, true},
		{"on the reed's top", true, 17.5, 16.6, false, Pickup{Entity: reed.ID, Aimed: true}, true},
		{"on the pile", true, 17.05, 16.35, false, Pickup{Drop: stone.ID, Aimed: true}, true},
		{"on empty ground", true, 16.5, 15.2, false, Pickup{Drop: stone.ID}, true},
		{"seated, on empty ground", true, 16.5, 15.2, true, Pickup{}, false},
		{"seated, on the twig", true, 15.5, 16.6, true, Pickup{Entity: twig.ID, Aimed: true}, true},
	} {
		got, ok := PickUpFrom(16.5, 16.5, c.aim, c.ax, c.ay, c.seated, drops, w.targetCandidate)
		if ok != c.wantAny || got != c.want {
			t.Errorf("%s: %+v %v, want %+v %v", c.name, got, ok, c.want, c.wantAny)
		}
	}
	// None of them is ever the left button's.
	for _, aim := range [][2]float64{{15.5, 16.6}, {17.5, 16.6}, {17.05, 16.35}} {
		if tg, ok := ResolveTarget(16.5, 16.5, true, aim[0], aim[1], false, w.targetCandidate, nil, 0); ok {
			t.Errorf("pointing at %v: the left button's target is %+v", aim, tg)
		}
	}
	if !LooseItem(EntityFlower) || LooseItem(EntityBerryBush) || LooseItem(EntityGrassTuft) || LooseItem(EntityFlora) {
		t.Error("loose things are the ones lying on the ground; plants are gathered by the left button")
	}
}
