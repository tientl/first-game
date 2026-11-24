package sim

import (
	"math"
	"testing"
)

// TestBossFightsAreLongDangerousButWinnable measures a fight with every
// boss (see fightBoss). One prepared player (an obsidian knife, tyrant-tooth
// armour, eight healing draughts and four strong ones) who steps back from
// what they see coming wins in a minute and a half to three, and needs
// medicine to; one who just stands and trades blows pays far more; one with
// no armour and no medicine dies.
func TestBossFightsAreLongDangerousButWinnable(t *testing.T) {
	const limit = 5 * 60 * TicksPerSecond
	for i := range NumBosses {
		k := BossKind(i)
		w, p, l := bossArena(t, k, 8, 4)
		careful := fightBoss(w, p, l, true, limit)
		w, p, l = bossArena(t, k, 8, 4)
		careless := fightBoss(w, p, l, false, limit)
		w, p, l = bossArena(t, k, 0, 0)
		p.Inv.Take(ItemDiamondArmor, 1)
		naked := fightBoss(w, p, l, true, limit)
		t.Logf("%-20s %5d hp | careful: %v", k.Name(), k.MaxHP(), careful)
		t.Logf("%-20s       | careless: %v", "", careless)
		t.Logf("%-20s       | no armour, no medicine: %v", "", naked)

		if !careful.won || careful.secs < 90 || careful.secs > 180 {
			t.Errorf("%s: the careful player's fight went %v, want won in 90 to 180 seconds", k.Name(), careful)
		}
		if careful.taken < 1.2*MaxHealth || careful.potions == 0 {
			t.Errorf("%s: the careful player lost only %.0f health and drank %d medicines: not dangerous", k.Name(), careful.taken, careful.potions)
		}
		if careful.enragedAt == 0 {
			t.Errorf("%s was never enraged", k.Name())
		}
		if !careless.won || careless.taken < careful.taken {
			t.Errorf("%s: standing and trading blows went %v, want won but at a higher cost than %.0f", k.Name(), careless, careful.taken)
		}
		if !naked.died {
			t.Errorf("%s: a player with no armour and no medicine survived: %v", k.Name(), naked)
		}
		if k == CreatureTitanoboa && careful.held == 0 {
			t.Errorf("Titanoboa never seized the player")
		}
		if k == CreatureMegalodon && careful.boatLeft <= 0 {
			t.Errorf("the boat sank under a prepared player")
		}
	}
}

func TestABossWakesRoaringAndRagesAtHalfHealth(t *testing.T) {
	w, p, l := bossArena(t, CreatureGiganoto, 0, 0)
	c := l.c
	stepN(w, 5)
	if l.awake || c.State == CreatureRoar {
		t.Fatalf("it woke for a player %.0f tiles off", math.Hypot(c.X-p.X, c.Y-p.Y))
	}
	p.X = c.X - 8
	w.Step()
	if !l.awake || c.State != CreatureRoar {
		t.Fatalf("a player 8 tiles off: awake %v, state %d; want a roar", l.awake, c.State)
	}
	calm := w.bossCD(l, bossTuneOf(c.Kind).biteCD)
	c.HP = c.Kind.MaxHP()/2 - 1
	for range 3 * TicksPerSecond {
		if c.State == CreatureRoar && l.enraged {
			break
		}
		w.Step()
	}
	if !l.enraged || c.State != CreatureRoar {
		t.Fatalf("below half health: enraged %v, state %d; want a roar of rage", l.enraged, c.State)
	}
	if fast := w.bossCD(l, bossTuneOf(c.Kind).biteCD); fast >= calm {
		t.Errorf("enraged it bites every %d ticks, calm every %d", fast, calm)
	}
}

func TestABossKeepsToItsLairAndHealsThere(t *testing.T) {
	w, p, l := bossArena(t, CreatureCarcharo, 0, 0)
	c := l.c
	p.Health, p.X = 1e9, c.X-6 // nothing kills them: this is about the boss
	w.Step()
	c.HP -= 3000
	far := 0.0
	for range 30 * TicksPerSecond { // the player runs off west; it follows, but only so far
		w.SetInput(p.ID, Input{MoveX: -1})
		w.Step()
		far = max(far, math.Hypot(c.X-l.x, c.Y-l.y))
	}
	if far > bossLeash+0.5 {
		t.Errorf("it chased %.1f tiles from its lair, leash %.0f", far, bossLeash)
	}
	w.SetInput(p.ID, Input{})
	p.X = 1.5 // and gone
	stepN(w, 60*TicksPerSecond)
	if d := math.Hypot(c.X-l.x, c.Y-l.y); l.awake || c.HP != c.Kind.MaxHP() || d > c.Kind.def().leash+1 {
		t.Errorf("a minute after everyone left: awake %v, health %d of %d, %.1f tiles from its lair", l.awake, c.HP, c.Kind.MaxHP(), d)
	}
}

func TestTitanoboaHoldsItsPreyFast(t *testing.T) {
	w, p, l := bossArena(t, CreatureTitanoboa, 0, 0)
	c := l.c
	p.Health = 1e9
	p.X = c.X - 2
	for range 20 * TicksPerSecond {
		if l.held == p.ID {
			break
		}
		w.Step()
	}
	if l.held != p.ID || c.State != CreatureGrab {
		t.Fatalf("it never seized the player")
	}
	x, y, hp := p.X, p.Y, p.Health
	w.SetInput(p.ID, Input{MoveX: -1})
	w.Step()
	if p.X != x || p.Y != y {
		t.Errorf("held fast, the player moved from (%.2f, %.2f) to (%.2f, %.2f)", x, y, p.X, p.Y)
	}
	held := 1
	for l.held == p.ID && held < TitanoboaGrabTicks {
		w.Step()
		held++
	}
	if held >= TitanoboaGrabTicks*3/4 {
		t.Errorf("struggling, the player was held %d ticks; at most %d without", held, TitanoboaGrabTicks)
	}
	if p.Health >= hp {
		t.Errorf("the coils did not hurt")
	}
}

// defeat kills boss k of w at once, as by a last blow of p's.
func defeat(t *testing.T, w *World, p *Player, k CreatureKind) {
	t.Helper()
	i, _ := BossIndex(k)
	c := w.bosses.lairs[i].c
	if c == nil {
		t.Fatalf("no %s to defeat", k.Name())
	}
	c.HP = 1
	w.wound(c, 10, p)
	w.Step()
}

func TestFallenBossesStayFallenAndTheGameIsWon(t *testing.T) {
	w := genSeas(t, 1, SizeMedium)
	id := w.AddPlayer("Tiên")
	p := w.players[id]
	for i := range NumBosses - 1 {
		defeat(t, w, p, BossKind(i))
		if show, _, _ := w.Victory(id); show {
			t.Fatalf("the game is won with %d bosses still standing", NumBosses-1-i)
		}
	}
	// Each dropped its legendary item where it fell, once, and it never rots.
	count := map[ItemKind]int{}
	for _, d := range w.Drops() {
		if d.Item.Legendary() {
			count[d.Item] += d.Count
		}
	}
	for i := range NumBosses - 1 {
		if n := count[LegendaryItem(BossKind(i))]; n != 1 {
			t.Errorf("%d %s on the ground, want 1", n, LegendaryItem(BossKind(i)).Def().Name)
		}
	}
	stepN(w, 2*DropLifeTicks)
	left := 0
	for _, d := range w.Drops() {
		if d.Item.Legendary() {
			left++
		}
	}
	if left != NumBosses-1 {
		t.Errorf("%d legendary items left on the ground after a while, want %d", left, NumBosses-1)
	}

	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	stepN(w2, TicksPerSecond)
	for _, b := range w2.Bosses() {
		standing := b.Kind == CreatureMegalodon
		if b.Defeated == standing || (b.ID != 0) != standing {
			t.Errorf("after loading, %s: defeated %v, id %d", b.Kind.Name(), b.Defeated, b.ID)
		}
	}
	for _, c := range w2.Creatures() {
		if c.Kind.Boss() && c.Kind != CreatureMegalodon {
			t.Errorf("a fallen %s is back", c.Kind.Name())
		}
	}

	id2 := w2.AddPlayer("Tiên")
	p2 := w2.players[id2]
	defeat(t, w2, p2, CreatureMegalodon)
	show, day, beaten := w2.Victory(id2)
	if !show || day != DayOf(w2.tick) || len(beaten) != NumBosses {
		t.Fatalf("all fallen: show %v, day %d, %d beaten", show, day, len(beaten))
	}
	other := w2.AddPlayer("Bạn")
	w2.Apply(id2, Command{Kind: CmdVictorySeen})
	if show, _, _ := w2.Victory(id2); show {
		t.Errorf("the screen still shows after it was closed")
	}
	if show, _, _ := w2.Victory(other); !show {
		t.Errorf("another player was not shown the victory")
	}
	w3, err := Load(w2.Save())
	if err != nil {
		t.Fatal(err)
	}
	a, b := w3.AddPlayer("Tiên"), w3.AddPlayer("Bạn")
	if show, d, _ := w3.Victory(a); show || d != day {
		t.Errorf("after loading, the victory shows again for whoever closed it, or moved to day %d", d)
	}
	if show, _, _ := w3.Victory(b); !show {
		t.Errorf("after loading, the victory no longer shows for whoever had not seen it")
	}
	if n := len(w3.Bosses()); n != NumBosses {
		t.Errorf("%d bosses after loading", n)
	}
}

func TestAnOldSaveGetsItsBosses(t *testing.T) {
	w := genSeas(t, 2, SizeMedium)
	d := w.Save()
	d.Version, d.Bosses = 11, nil
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, b := range w2.Bosses() {
		if !b.Defeated && w2.creatureByID(b.ID) != nil {
			n++
		}
	}
	if n != NumBosses {
		t.Errorf("an old save loaded with %d bosses, want %d", n, NumBosses)
	}
}

func TestAPlayerSeesWhereABossLives(t *testing.T) {
	w := genSeas(t, 1, SizeMedium)
	id := w.AddPlayer("Tiên")
	p := w.players[id]
	if n := len(w.BossesSeenBy(id)); n != 0 {
		t.Fatalf("a new player knows %d lairs", n)
	}
	b := w.Bosses()[3]
	p.X, p.Y = b.LairX-9, b.LairY
	stepN(w, 10)
	seen := w.BossesSeenBy(id)
	if len(seen) != 1 || seen[0].Kind != b.Kind {
		t.Fatalf("near %s's lair, the map knows %+v", b.Kind.Name(), seen)
	}
	if _, ok := w.BossFight(id); !ok {
		t.Errorf("9 tiles from %s, no boss bar", b.Kind.Name())
	}
}
