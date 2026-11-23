package sim

import (
	"math"
	"testing"
)

var hellKinds = []CreatureKind{CreatureOrnitho, CreatureThesc, CreatureAnzu, CreatureToro, CreatureAchero, CreatureQuetzal, CreatureBrachy, CreatureChampso}

func TestTheTyrantsNeighboursLiveInTheOpenWorld(t *testing.T) {
	w := genOpen(t, 1, SizeLarge)
	for _, k := range hellKinds {
		cs := creatureByKind(w, k)
		t.Logf("%s: %d", k.Name(), len(cs))
		if len(cs) < creatureMinPerKind {
			t.Errorf("%d %s, want at least %d", len(cs), k.Name(), creatureMinPerKind)
		}
		for _, c := range cs {
			if !w.creatureFits(k, c.X, c.Y) {
				t.Errorf("%s at (%.1f, %.1f) is outside its habitat", k.Name(), c.X, c.Y)
			}
		}
	}
	if CreatureRaptor.Name() != "Dakotaraptor" {
		t.Errorf("the big raptor is called %q", CreatureRaptor.Name())
	}
}

func TestAnOrnithomimusOutrunsYouButStopsForBreath(t *testing.T) {
	w := newWorld(64, 32)
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	w.spawnX, w.spawnY = 20.5, 16.5
	o := w.addCreature(CreatureOrnitho, 24.5, 16.5)
	id := w.AddPlayer("a")
	stepN(w, OrnithoSprintTicks/2)
	if o.State != CreatureFlee || math.Hypot(o.X-20.5, o.Y-16.5) < 8 {
		t.Fatalf("state %d at %.1f tiles: it should be sprinting away, faster than a player", o.State, math.Hypot(o.X-20.5, o.Y-16.5))
	}
	stepN(w, OrnithoSprintTicks)
	if o.State != CreatureRest {
		t.Fatalf("state %d: after its sprint it should stop for breath", o.State)
	}
	// Walk up while it pants, with berries: it can be fed, and stays.
	p := w.players[id]
	p.X, p.Y = o.X-1.2, o.Y
	inHand(p, ItemBerry, 10)
	hitUntil(w, id, 8*TicksPerSecond, func() bool { return o.Tamed() })
	if !o.Tamed() {
		t.Fatalf("tame %d state %d: berries fed while it rests should tame it", o.Tame, o.State)
	}
	if !CreatureOrnitho.Rideable() || CreatureOrnitho.tameDef().ride <= CreatureRaptor.tameDef().ride {
		t.Error("a tamed ornithomimus should be the fastest mount")
	}
}

func TestAnzuRaisesItsCrestThenKicks(t *testing.T) {
	w := flatWorld()
	a := w.addCreature(CreatureAnzu, 19.5, 16.5)
	id := w.AddPlayer("a")
	w.Step()
	if a.State != CreatureRoar {
		t.Fatalf("state %d: it should raise its crest at someone three tiles off", a.State)
	}
	stepN(w, 4*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health > MaxHealth-AnzuKickDamage+1 {
		t.Fatalf("health %.0f: standing close, a player should be kicked", p.Health)
	}
	// Walk away: it does not follow far from where it stood.
	w.players[id].X = 2.5
	stepN(w, 6*TicksPerSecond)
	if math.Hypot(a.X-a.homeX, a.Y-a.homeY) > AnzuLeash+1 {
		t.Fatalf("it chased %.1f tiles from home", math.Hypot(a.X-a.homeX, a.Y-a.homeY))
	}
}

func TestTorosaurusGoresHarderThanTheThreeHorn(t *testing.T) {
	if _, dmg, _ := goreTune(CreatureToro); dmg <= TrikeGoreDamage {
		t.Fatal("a torosaurus should gore harder")
	}
	w := flatWorld()
	c := w.addCreature(CreatureToro, 19.3, 16.5)
	id := w.AddPlayer("a")
	hitUntil(w, id, 1, func() bool { return false })
	stepN(w, 5*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health > MaxHealth-ToroGoreDamage+1 || c.HP == toroDef.hp {
		t.Fatalf("health %.0f hp %d: an angry torosaurus should gore", p.Health, c.HP)
	}
}

func TestAcheroraptorsBiteLess(t *testing.T) {
	bite, _, _, _, _ := raptorTune(CreatureAchero)
	big, _, _, _, _ := raptorTune(CreatureRaptor)
	if bite >= big || acheroDef.herd <= raptorDef.herd || acheroDef.hp >= raptorDef.hp {
		t.Fatal("acheroraptors should be smaller, weaker and in bigger packs")
	}
	w := flatWorld()
	w.addCreature(CreatureAchero, 20.5, 16.5)
	id := w.AddPlayer("a")
	stepN(w, 4*TicksPerSecond)
	p := mustPlayer(t, w, id)
	if p.Health == MaxHealth || CreatureKind(p.NoticeArg) != CreatureAchero {
		t.Fatalf("health %.0f: an acheroraptor should hunt a player four tiles off", p.Health)
	}
}

func TestQuetzalcoatlusStabsThenFliesOffWhenHurt(t *testing.T) {
	w := flatWorld()
	q := w.addCreature(CreatureQuetzal, 20.5, 16.5)
	id := w.AddPlayer("a")
	stepN(w, 4*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health > MaxHealth-QuetzalStabDamage+1 {
		t.Fatalf("health %.0f: it should stalk up and stab", p.Health)
	}
	q.HP = int(float64(quetzalDef.hp) * QuetzalFlyBelow)
	w.Step()
	if q.State != CreatureSoar || !q.State.OutOfReach() {
		t.Fatalf("state %d: badly hurt, it should take to the air out of reach", q.State)
	}
	stepN(w, QuetzalFlyTicks+TicksPerSecond)
	if q.State == CreatureSoar || !w.creatureFits(CreatureQuetzal, q.X, q.Y) {
		t.Fatalf("state %d at (%.1f, %.1f): it should land again on open ground", q.State, q.X, q.Y)
	}
}

func TestThePondCrocodilesAmbushLikeTheCrocodile(t *testing.T) {
	for _, k := range []CreatureKind{CreatureBrachy, CreatureChampso} {
		w := marshWorld(true)
		c := w.addCreature(k, 18.5, 16.5)
		c.State = CreatureLurk
		id := w.AddPlayer("a")
		w.players[id].X = 16.8
		if len(w.creatureCandidates(16.8, 16.5)) != 0 {
			t.Fatalf("%s: a lurker can be targeted", k.Name())
		}
		stepN(w, 3*TicksPerSecond)
		if p := mustPlayer(t, w, id); p.Health == MaxHealth || CreatureKind(p.NoticeArg) != k {
			t.Errorf("%s: health %.0f, it should burst out and bite", k.Name(), p.Health)
		}
	}
}
