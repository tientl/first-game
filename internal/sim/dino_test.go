package sim

import (
	"math"
	"testing"
)

var dinoKinds = []CreatureKind{CreatureTrike, CreatureStego, CreatureBrachio, CreatureRaptor, CreatureRex, CreaturePtero, CreaturePlesio}

func TestDinosaursRoamTheOpenWorld(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 1, Width: 512, Height: 512, Style: StyleOpen})
	if err != nil {
		t.Fatal(err)
	}
	sx, sy := w.Spawn()
	for _, k := range dinoKinds {
		cs := creatureByKind(w, k)
		t.Logf("%s: %d", k.Name(), len(cs))
		if len(cs) < creatureMinPerKind {
			t.Errorf("%d %s, want at least %d", len(cs), k.Name(), creatureMinPerKind)
		}
		for _, c := range cs {
			if !w.creatureFits(k, c.X, c.Y) {
				t.Errorf("%s at (%.1f, %.1f) is outside its habitat", k.Name(), c.X, c.Y)
			}
			if d := math.Hypot(c.X-sx, c.Y-sy); d < k.def().away-4 { // a herd spreads a little
				t.Errorf("%s at (%.1f, %.1f), only %.0f tiles from the spawn", k.Name(), c.X, c.Y, d)
			}
		}
	}
	island, err := Generate(GenConfig{Seed: 1, Width: 128, Height: 128})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range dinoKinds {
		if n := len(creatureByKind(island, k)); n != 0 {
			t.Errorf("%d %s on the old island", n, k.Name())
		}
	}
}

func TestDinosaursStayInTheirHabitat(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 3, Width: 256, Height: 256, Style: StyleOpen})
	if err != nil {
		t.Fatal(err)
	}
	stepN(w, 40*TicksPerSecond)
	for _, c := range w.Creatures() {
		if !w.creatureFits(c.Kind, c.X, c.Y) {
			t.Errorf("%s %d wandered to (%.2f, %.2f), outside its habitat", c.Kind.Name(), c.ID, c.X, c.Y)
		}
	}
}

// hitUntil holds Action down for p until done reports true or n ticks pass.
func hitUntil(w *World, id PlayerID, n int, done func() bool) {
	w.SetInput(id, Input{Action: true})
	for range n {
		if done() {
			break
		}
		w.Step()
	}
	w.SetInput(id, Input{})
}

// huntDown has p walk after c, hitting it whenever it is in reach, until it
// is dead or n ticks pass.
func huntDown(w *World, id PlayerID, c *Creature, n int) {
	p := w.players[id]
	for range n {
		if c.gone {
			break
		}
		in := Input{Action: true}
		if d := math.Hypot(c.X-p.X, c.Y-p.Y); d > c.Kind.Reach()*0.8 {
			in.MoveX, in.MoveY = sign(c.X-p.X), sign(c.Y-p.Y)
		}
		w.SetInput(id, in)
		w.Step()
	}
	w.SetInput(id, Input{})
}

func sign(v float64) int8 {
	switch {
	case v > 0.2:
		return 1
	case v < -0.2:
		return -1
	}
	return 0
}

func TestATrikeHerdTurnsOnWhoeverHurtsOne(t *testing.T) {
	w := flatWorld()
	a := w.addCreature(CreatureTrike, 18.2, 16.5)
	b := w.addCreature(CreatureTrike, 22.5, 18.5)
	id := w.AddPlayer("a")
	stepN(w, 2*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health != MaxHealth {
		t.Fatalf("health %.0f: a calm herd hurt someone", p.Health)
	}
	a.X, a.Y = 18.2, 16.5
	hitUntil(w, id, 1, func() bool { return false })
	if a.HP == trikeDef.hp || b.rage == 0 || b.foe != id {
		t.Fatalf("hp %d, herd mate rage %d foe %d: the hit should anger the herd", a.HP, b.rage, b.foe)
	}
	stepN(w, 5*TicksPerSecond)
	p := mustPlayer(t, w, id)
	if p.Health > MaxHealth-TrikeGoreDamage || p.Notice != NoticeBitten || CreatureKind(p.NoticeArg) != CreatureTrike {
		t.Fatalf("health %.0f notice %d: want gored by an angry three-horn", p.Health, p.Notice)
	}
}

func TestKillingABigBeastGivesHidesAndBones(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureBrachio, 18.6, 16.5)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemSpear, 1)
	huntDown(w, id, c, 60*TicksPerSecond)
	if !c.gone {
		t.Fatalf("hp %d left after a minute of spearing", c.HP)
	}
	reap(w, id)
	p := mustPlayer(t, w, id)
	for _, in := range CreatureBrachio.Loot() {
		if got := p.Inv.Count(in.Item); got != in.Count {
			t.Errorf("%s: %d, want %d", in.Item.Def().Name, got, in.Count)
		}
	}
	if p.Health != MaxHealth {
		t.Errorf("health %.0f: a long-neck never fights back", p.Health)
	}
}

func TestAFullBagStillFightsBack(t *testing.T) {
	w := flatWorld()
	c := w.addCreature(CreatureRaptor, 17.6, 16.5)
	c.HP = 1
	id := w.AddPlayer("a")
	p := w.players[id]
	for i := range p.Inv {
		p.Inv[i] = Slot{Item: ItemCoal, Count: MaxStack} // nothing else fits
	}
	hitUntil(w, id, 2*TicksPerSecond, func() bool { return c.gone })
	if !c.gone {
		t.Fatal("a full bag kept a player from killing a raptor that attacks")
	}
	if n := len(entitiesOf(w, EntityCorpse)); n != 1 {
		t.Errorf("%d bodies: want the raptor lying on the ground, its meat not lost", n)
	}
}

func TestOutrunningARaptorPack(t *testing.T) {
	w := newWorld(64, 32)
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	w.spawnX, w.spawnY = 50.5, 16.5
	for i := range 3 {
		w.addCreature(CreatureRaptor, 56.5+float64(i)*0.6, 16.5+float64(i)*0.4)
	}
	id := w.AddPlayer("a")
	stepN(w, TicksPerSecond/2)
	if s := creatureByKind(w, CreatureRaptor)[0].State; s != CreatureChase {
		t.Fatalf("state %d: raptors did not go for a player six tiles off", s)
	}
	// Run west; the pack falls behind.
	w.SetInput(id, Input{MoveX: -1})
	stepN(w, 4*TicksPerSecond)
	w.SetInput(id, Input{})
	p := mustPlayer(t, w, id)
	near := math.Inf(1)
	for _, c := range creatureByKind(w, CreatureRaptor) {
		near = min(near, math.Hypot(c.X-p.X, c.Y-p.Y))
	}
	if p.Dead || near < 4 {
		t.Fatalf("dead %v, nearest raptor %.1f tiles: a running player should get away", p.Dead, near)
	}
}

func TestTheTyrantRoarsThenCharges(t *testing.T) {
	w := flatWorld()
	rex := w.addCreature(CreatureRex, 22.5, 16.5)
	id := w.AddPlayer("a")
	w.Step()
	if rex.State != CreatureRoar {
		t.Fatalf("state %d: the tyrant should roar on seeing a player six tiles off", rex.State)
	}
	stepN(w, RexRoarTicks+5*TicksPerSecond)
	p := mustPlayer(t, w, id)
	if p.Health > MaxHealth-RexBiteDamage+1 {
		t.Fatalf("health %.0f: a player standing still should be bitten", p.Health)
	}
}

func TestHittingTheTyrantDoesNotStopIt(t *testing.T) {
	w := flatWorld()
	rex := w.addCreature(CreatureRex, 18.4, 16.5)
	id := w.AddPlayer("a")
	w.players[id].Inv.Add(ItemSpear, 1)
	hitUntil(w, id, 8*TicksPerSecond, func() bool { return rex.gone })
	p := mustPlayer(t, w, id)
	if p.Health > MaxHealth-2*RexBiteDamage+1 {
		t.Fatalf("health %.0f after 8 s toe to toe: hits should not stun-lock the tyrant", p.Health)
	}
}

func TestFlyingReptilesDiveThenClimbOutOfReach(t *testing.T) {
	w := flatWorld()
	pt := w.addCreature(CreaturePtero, 16.5, 10.5)
	pt.State = CreatureSoar
	pt.homeX, pt.homeY = 16.5, 10.5
	id := w.AddPlayer("a")
	if len(w.creatureCandidates(16.5, 11.5)) != 0 {
		t.Fatal("a soaring reptile can be targeted")
	}
	hurtBy := false
	for range 6 * TicksPerSecond {
		w.Step()
		if p := mustPlayer(t, w, id); p.Health < MaxHealth {
			hurtBy = CreatureKind(p.NoticeArg) == CreaturePtero
			break
		}
	}
	if !hurtBy {
		t.Fatal("the flying reptile never dove at the player below")
	}
	stepN(w, AttackTicks+PteroClimbTicks+1)
	if pt.State != CreatureSoar {
		t.Fatalf("state %d: it should be back up in the air", pt.State)
	}
}

func TestSeaReptilesSnapAtTheShore(t *testing.T) {
	w := shoreWorld() // the sea from column 18
	pl := w.addCreature(CreaturePlesio, 19.6, 16.5)
	pl.State = CreatureLurk
	id := w.AddPlayer("a")
	w.players[id].X = 17.5
	if len(w.creatureCandidates(17.5, 16.5)) != 0 {
		t.Fatal("a sea reptile under water can be targeted")
	}
	stepN(w, 4*TicksPerSecond)
	p := mustPlayer(t, w, id)
	if p.Health > MaxHealth-PlesioBiteDamage+1 || pl.X < 18 {
		t.Fatalf("health %.0f, reptile at x %.1f: it should rear up and bite from the water", p.Health, pl.X)
	}
	if !w.creatureFits(CreaturePlesio, pl.X, pl.Y) {
		t.Fatal("the sea reptile left the water")
	}
}
