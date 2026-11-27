package sim

import (
	"math"
	"testing"
)

// guardedAt is a world with a player at (16.5, 16.5) facing right with a
// shield in the bag, and a beast of kind k at (x, 16.5).
func guardedAt(t *testing.T, k CreatureKind, x float64) (*World, *Creature, *Player) {
	t.Helper()
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	p.Inv[9] = Slot{Item: ItemShield, Count: 1}
	p.X, p.Y, p.Dir = 16.5, 16.5, DirRight
	p.harm = 1
	c := w.addCreature(k, x, 16.5)
	c.fed = true
	return w, c, p
}

// blowFrom lands a blow of dmg on p from c, as c's own step would.
func blowFrom(w *World, c *Creature, p *Player, dmg float64) {
	w.striker = blowSrc{c, w.tick}
	struck(p, dmg)
	w.striker = blowSrc{}
}

func TestTheShieldIsMadeOfWoodHideAndRope(t *testing.T) {
	r, ok := RecipeByID(RecipeShield)
	if !ok || r.Gives != ItemShield || ItemShield.Def().Name != "Khiên gỗ tròn" {
		t.Fatalf("recipe %+v: the round wooden shield should be made by hand", r)
	}
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	for _, in := range r.Inputs {
		p.Inv.Add(in.Item, in.Count)
	}
	w.Apply(id, Command{Kind: CmdCraft, Recipe: RecipeShield})
	if !p.HasShield() {
		t.Fatal("crafting gave no shield")
	}
}

func TestARaisedShieldTakesMostOfABlowFromTheFront(t *testing.T) {
	w, c, p := guardedAt(t, CreatureTrike, 18)
	w.SetInput(p.ID, Input{Guard: true})
	stepN(w, ParryTicks+2) // up long enough: no parry, a block
	c.X, c.Y = 18, 16.5
	if !p.Guarding() {
		t.Fatal("the shield should be up with Guard held and a shield in the bag")
	}
	h0, jolt := p.Health, p.HurtSeq()
	blowFrom(w, c, p, 40)
	if got, want := h0-p.Health, 40*(1-GuardShare); math.Abs(got-want) > 1e-6 {
		t.Errorf("lost %.2f, want %.2f: the shield stops %.0f%% of a frontal blow", got, want, GuardShare*100)
	}
	if p.HurtSeq() != jolt || p.GuardSeq() != 1 || p.Parried() {
		t.Errorf("jolt %d->%d, guard seq %d, parried %v: a block jars the shield, not the player", jolt, p.HurtSeq(), p.GuardSeq(), p.Parried())
	}
	// From behind it does nothing.
	c.X = 15
	h0 = p.Health
	blowFrom(w, c, p, 40)
	if got := h0 - p.Health; math.Abs(got-40) > 1e-6 || p.HurtSeq() == jolt {
		t.Errorf("lost %.2f from behind, want all 40 and a jolt", got)
	}
	// Lowered, nothing either.
	w.SetInput(p.ID, Input{})
	w.Step()
	c.X = 18
	h0 = p.Health
	blowFrom(w, c, p, 40)
	if got := h0 - p.Health; math.Abs(got-40) > 1e-6 {
		t.Errorf("lost %.2f with the shield down, want 40", got)
	}
}

func TestAShieldRaisedJustInTimeParriesAndStaggersALightBeast(t *testing.T) {
	w, c, p := guardedAt(t, CreatureRaptor, 17.6)
	w.SetInput(p.ID, Input{Guard: true})
	stepN(w, ParryTicks-1)
	c.X, c.Y = 17.6, 16.5
	h0 := p.Health
	blowFrom(w, c, p, RaptorBiteDamage)
	if p.Health != h0 || !p.Parried() {
		t.Fatalf("lost %.2f, parried %v: a shield raised %d ticks before the bite should turn it aside", h0-p.Health, p.Parried(), ParryTicks-1)
	}
	w.Step()
	if c.State != CreatureHurt || c.timer < ParryStagger-2 {
		t.Fatalf("state %d timer %d: a parried raptor should reel for %d ticks", c.State, c.timer, ParryStagger)
	}
	// A big beast is only blocked, however well timed.
	w, c, p = guardedAt(t, CreatureRex, 18.5)
	w.SetInput(p.ID, Input{Guard: true})
	w.Step()
	c.X, c.Y = 18.5, 16.5
	h0 = p.Health
	blowFrom(w, c, p, RexBiteDamage)
	if got, want := h0-p.Health, RexBiteDamage*(1-GuardShare); math.Abs(got-want) > 1e-6 {
		t.Errorf("lost %.2f to a parried tyrant, want the block's %.2f", got, want)
	}
	w.Step()
	if c.State == CreatureHurt {
		t.Error("a tyrant should not reel from a parry")
	}
}

func TestParryMaths(t *testing.T) {
	for _, c := range []struct {
		k      CreatureKind
		up     int
		front  bool
		share  float64
		stagge bool
	}{
		{CreatureRaptor, 0, true, 0, true},
		{CreatureRaptor, ParryTicks, true, 0, true},
		{CreatureRaptor, ParryTicks + 1, true, 1 - GuardShare, false},
		{CreatureRaptor, 0, false, 1, false},
		{CreatureTrike, 0, true, 1 - GuardShare, false}, // tough: a block
		{CreatureTitanoboa, 0, true, 1 - GuardShare, false},
		{CreatureSabre, 2, true, 0, true},
	} {
		share, st := ParryOutcome(c.k, c.up, c.front)
		if math.Abs(share-c.share) > 1e-9 || st != c.stagge {
			t.Errorf("%s up %d front %v: share %.2f staggered %v, want %.2f %v", c.k.Name(), c.up, c.front, share, st, c.share, c.stagge)
		}
	}
	if GuardShare < 0.7 || GuardShare > 0.8 {
		t.Errorf("a block stops %.0f%%, want 70-80%%", GuardShare*100)
	}
	if ParryTicks*1000/TicksPerSecond != 250 {
		t.Errorf("the parry window is %d ms, want 250", ParryTicks*1000/TicksPerSecond)
	}
}

func TestBehindAShieldOneWalksSlowAndStrikesNoBlows(t *testing.T) {
	w, c, p := guardedAt(t, CreatureRabbit, 30)
	_ = c
	w.SetInput(p.ID, Input{MoveX: -1, Guard: true})
	x0 := p.X
	stepN(w, 10)
	if got, want := x0-p.X, 10*PlayerSpeed*GuardSlow; math.Abs(got-want) > 0.05 {
		t.Errorf("walked %.2f behind the shield, want %.2f", got, want)
	}
	if p.Dir != DirRight {
		t.Errorf("faces %d: backing off behind a shield, a player keeps facing the foe", p.Dir)
	}
	// No blows while it is up.
	tree := w.addEntity(EntityTree, int(p.X)+1, int(p.Y))
	w.SetInput(p.ID, Input{Guard: true, Action: true})
	stepN(w, 2*TicksPerSecond)
	if p.actedAt != 0 {
		t.Error("a player behind a raised shield should not strike")
	}
	_ = tree
	// Without a shield, Guard does nothing.
	p.Inv = Inventory{}
	w.SetInput(p.ID, Input{Guard: true})
	w.Step()
	if p.Guarding() {
		t.Error("no shield, no guard")
	}
}

func TestAShieldTakesMostOfAThrow(t *testing.T) {
	w, c, p := guardedAt(t, CreatureTrike, 18)
	w.SetInput(p.ID, Input{Guard: true})
	stepN(w, ParryTicks+2)
	c.X = 18
	x0 := p.X
	blowFrom(w, c, p, 10)
	p.guard.at = w.tick
	w.shove(p, c.X, c.Y, 2)
	stepN(w, 10)
	if got := x0 - p.X; got > 2*GuardKnock+BlockPush+0.05 || got < BlockPush {
		t.Errorf("thrown %.2f tiles with the shield up, want about %.2f", got, 2*GuardKnock+BlockPush)
	}
}

func TestAThrowIsEasedButGoesAllTheWay(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X, p.Y = 16.5, 16.5
	w.shove(p, 15.5, 16.5, 2)
	first := p.X - 16.5
	if first < 2*KnockFirst-0.01 || first > 2*KnockFirst+0.01 {
		t.Errorf("first tick of a throw went %.2f, want %.2f", first, 2*KnockFirst)
	}
	prev := first
	for range 8 {
		w.Step()
		d := p.X - 16.5
		if d < prev {
			t.Fatal("a throw went backwards")
		}
		prev = d
	}
	if math.Abs(prev-2) > 0.02 {
		t.Errorf("thrown %.2f in all, want 2", prev)
	}
}

func TestABlowThrowsABeastBackTheLessTheBiggerItIs(t *testing.T) {
	throw := func(k CreatureKind, held ItemKind) float64 {
		w := flatWorld()
		id := w.AddPlayer("a")
		p := w.players[id]
		p.Inv = Inventory{}
		inHand(p, held, 1)
		p.X, p.Y = 16.5, 16.5
		c := w.addCreature(k, 17.5, 16.5)
		c.fed = true
		c.HP = 10000 // not killed
		x0 := c.X
		w.struckBy(p, c, false)
		for range 8 {
			w.stepJostle(c)
		}
		return c.X - x0
	}
	small, big := throw(CreatureRaptor, ItemBattleAxe), throw(CreatureRex, ItemBattleAxe)
	if math.Abs(small-HeavyKnock) > 0.02 {
		t.Errorf("a raptor is thrown %.2f by the battle axe, want %.2f", small, HeavyKnock)
	}
	if big >= small/3 || big <= 0 {
		t.Errorf("a tyrant thrown %.2f, a raptor %.2f: the bigger the less", big, small)
	}
	if fist := throw(CreatureRaptor, ItemNone); fist >= small/2 {
		t.Errorf("a punch throws %.2f, the battle axe %.2f", fist, small)
	}
	if b := throw(CreatureTitanoboa, ItemBattleAxe); b != 0 {
		t.Errorf("a boss thrown %.2f: bosses stand their ground", b)
	}
}

// hitAndRun has a player with a spear strike a beast of kind k once and
// walk straight away; it returns the health they lost in the next few
// seconds. With noRiposte the beast's riposte is taken away each tick, as
// the game was before it.
func hitAndRun(t *testing.T, k CreatureKind, noRiposte bool) float64 {
	t.Helper()
	w := flatWorld()
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	w = bigFlat(96)
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	inHand(p, ItemSpear, 1)
	p.X, p.Y, p.Dir = 48.5, 48.5, DirRight
	p.harm = 1
	c := w.addCreature(k, p.X+1.6, p.Y)
	c.fed = true
	c.HP = c.Kind.MaxHP() * 3 // not felled by the one blow
	h0 := p.Health
	w.SetInput(id, Input{Action: true})
	for range 2 * TicksPerSecond {
		w.Step()
		if c.HP < c.Kind.MaxHP()*3 {
			break
		}
	}
	if c.HP == c.Kind.MaxHP()*3 {
		t.Fatalf("%s: the spear never landed", k.Name())
	}
	w.SetInput(id, Input{MoveX: -1}) // and away
	for range 3 * TicksPerSecond {
		if noRiposte {
			c.riposte = 0
		}
		w.Step()
	}
	return h0 - p.Health
}

// bigFlat is a grass world n tiles across.
func bigFlat(n int) *World {
	w := newWorld(n, n)
	for i := range w.tiles {
		w.tiles[i] = TileGrass
	}
	w.spawnX, w.spawnY = float64(n)/2+0.5, float64(n)/2+0.5
	return w
}

func TestStrikeAndStepBackIsAnswered(t *testing.T) {
	for _, k := range []CreatureKind{CreatureRaptor, CreatureSabre, CreatureTrike, CreatureBear} {
		if lost := hitAndRun(t, k, false); lost <= 0 {
			t.Errorf("%s: struck once and walked away unharmed: it should come back at them", k.Name())
		}
	}
	// It was free before: the beasts that caught nobody then show the riposte is what does it.
	free := 0
	for _, k := range []CreatureKind{CreatureRaptor, CreatureSabre, CreatureTrike, CreatureBear} {
		if hitAndRun(t, k, true) == 0 {
			free++
		}
	}
	if free == 0 {
		t.Log("every beast caught the runner even without the riposte")
	}
}

func TestABadlyHurtBeastDoesNotRiposte(t *testing.T) {
	w := bigFlat(64)
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	inHand(p, ItemSpear, 1)
	c := w.addCreature(CreatureRaptor, p.X+1.5, p.Y)
	c.HP = c.Kind.MaxHP() / 5
	w.struckBy(p, c, false)
	if c.riposte != 0 {
		t.Error("a raptor at a fifth of its health should not come back")
	}
	c = w.addCreature(CreatureRabbit, p.X+1, p.Y)
	w.struckBy(p, c, false)
	if c.riposte != 0 {
		t.Error("a rabbit does not fight back")
	}
}
