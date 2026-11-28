package sim

import (
	"math"
	"slices"
	"testing"
)

// lairWorld is a generated world with both of the demo's lairs in it, as they
// were before StyleLairs (the hollow and the open sea).
func lairWorld(t *testing.T) *World {
	t.Helper()
	w, err := Generate(GenConfig{Seed: 13, Width: 320, Height: 320, Style: StyleLairs - 1}) // the last style of the old lairs: see lairsite.go
	if err != nil {
		t.Fatal(err)
	}
	if !w.bosses.lairs[0].ok || !w.bosses.lairs[NumBosses-1].ok {
		t.Fatal("this world lacks a lair: pick another seed")
	}
	return w
}

// lairProps counts the lair props of each kind within r tiles of (x, y).
func lairProps(w *World, x, y, r float64) (n [NumLairProps]int) {
	for _, e := range w.entities {
		if e.Kind == EntityLairProp && e.Var < NumLairProps && math.Hypot(float64(e.TX)+0.5-x, float64(e.TY)+0.5-y) <= r {
			n[e.Var]++
		}
	}
	return n
}

func TestLairsAreDressed(t *testing.T) {
	w := lairWorld(t)
	l := &w.bosses.lairs[0]
	n := lairProps(w, l.x, l.y, LairSignRange)
	if n[LairNest] != 1 || n[LairCover] != 1 {
		t.Errorf("Titanoboa's lair has %d nests and %d covers, want one of each", n[LairNest], n[LairCover])
	}
	if e := w.entityAt(int(l.x), int(l.y)); e == nil || e.Kind != EntityLairProp || e.Var != LairNest {
		t.Errorf("the nest is not under the snake")
	}
	if n[LairBones] < 8 || n[LairSkin] < 1 || n[LairTrail] < 8 || n[LairRuin] < 3 || n[LairStump] < 3 {
		t.Errorf("round Titanoboa's lair: %d bones, %d skins, %d tracks, %d ruin stones, %d broken trees", n[LairBones], n[LairSkin], n[LairTrail], n[LairRuin], n[LairStump])
	}
	if near := lairProps(w, l.x, l.y, 6); near[LairBones] < 3 {
		t.Errorf("%d bones by the nest", near[LairBones])
	}
	if l.c == nil || !w.creatureFits(CreatureTitanoboa, l.x, l.y) {
		t.Errorf("the snake no longer fits in its dressed lair")
	}

	m := &w.bosses.lairs[NumBosses-1]
	if !m.bar {
		t.Fatalf("no sandbar by the shark's lair")
	}
	if d := math.Hypot(float64(m.barX)+0.5-m.x, float64(m.barY)+0.5-m.y); d < 9 || d > 14 {
		t.Errorf("the sandbar is %.1f tiles from the lair", d)
	}
	if w.TileAt(m.barX, m.barY) != TileSand {
		t.Errorf("the sandbar is not sand")
	}
	if e := w.entityAt(m.barX, m.barY); e == nil || e.Kind != EntityLairProp || e.Var != LairWhale {
		t.Errorf("no whale bones on the sandbar")
	}
	if !w.creatureFits(CreatureMegalodon, m.x, m.y) {
		t.Errorf("the shark no longer fits in its lair")
	}

	// The same world again is the same lair.
	props := func(w *World) (out []Entity) {
		for _, e := range w.Save().Entities {
			if e.Kind == EntityLairProp {
				out = append(out, e)
			}
		}
		return out
	}
	if a, b := props(w), props(lairWorld(t)); len(a) == 0 || !slices.Equal(a, b) {
		t.Errorf("two worlds of one seed have different lairs: %d and %d props", len(a), len(b))
	}
}

func TestOldSavesGetTheirLairsDressed(t *testing.T) {
	w := lairWorld(t)
	l := w.bosses.lairs[0]
	d := w.Save()
	d.Entities = slices.DeleteFunc(d.Entities, func(e Entity) bool { return e.Kind == EntityLairProp })
	kept, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if n := lairProps(kept, l.x, l.y, LairSignRange); n[LairNest] != 0 {
		t.Errorf("a save of this version had its lair dressed again")
	}
	d.Version = 13
	d.Tick /= 2
	old, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if n := lairProps(old, l.x, l.y, LairSignRange); n[LairNest] != 1 || n[LairBones] < 8 {
		t.Errorf("an old save's lair: %d nests, %d bones", n[LairNest], n[LairBones])
	}
	m := old.bosses.lairs[NumBosses-1]
	if e := old.entityAt(m.barX, m.barY); e == nil || e.Var != LairWhale {
		t.Errorf("an old save has no whale bones on its sandbar")
	}
}

// wake brings p to Titanoboa and steps until the fight has begun.
func wake(t *testing.T, w *World, p *Player, l *bossLair) {
	t.Helper()
	p.dev.god = true
	p.X, p.Y = l.x-5, l.y
	stepN(w, 3)
	if !l.awake || !l.arena {
		t.Fatalf("a player 5 tiles off: awake %v, arena shut %v", l.awake, l.arena)
	}
}

func TestTheArenaShutsWhenAFightStartsAndOpensAgain(t *testing.T) {
	ring := func(l *bossLair) (int, int) { return int(l.x - TitanArena), int(l.y) }

	w, p, l := bossArena(t, CreatureTitanoboa, 0, 0)
	tx, ty := ring(l)
	stepN(w, 3)
	if l.arena || w.blocked(tx, ty) {
		t.Fatalf("the arena is shut before anyone came")
	}
	wake(t, w, p, l)
	if !w.blocked(tx, ty) || w.standable(float64(tx)+0.5, float64(ty)+0.5) {
		t.Fatalf("no roots on the ring")
	}
	// Nobody walks out through it.
	p.X, p.Y = l.x-TitanArena+2, l.y
	p.input.MoveX = -1
	for range 6 * TicksPerSecond {
		w.Step()
		if l.arena && math.Hypot(p.X-l.x, p.Y-l.y) > TitanArena {
			t.Fatalf("the player walked out of the shut arena, to %.1f tiles", math.Hypot(p.X-l.x, p.Y-l.y))
		}
	}
	// The boss falls: the way is open.
	l.c.HP, l.c.gone = 0, true
	w.Step()
	if l.arena || w.blocked(tx, ty) {
		t.Errorf("the arena stayed shut after the boss fell")
	}

	// The player gets away (carried off, or dead): it sleeps, the way opens.
	w, p, l = bossArena(t, CreatureTitanoboa, 0, 0)
	wake(t, w, p, l)
	p.X, p.Y = l.x-30, l.y
	stepN(w, bossLoseTicks+3)
	if l.awake || l.arena {
		t.Errorf("with its foe gone: awake %v, arena shut %v", l.awake, l.arena)
	}

	// A fight that drags on: the roots give way, and stay down.
	w, p, l = bossArena(t, CreatureTitanoboa, 0, 0)
	wake(t, w, p, l)
	for i := 0; l.arena; i++ {
		if i > ArenaMaxTicks+5 {
			t.Fatalf("the arena never opened")
		}
		p.Health = MaxHealth
		w.Step()
	}
	stepN(w, 5*TicksPerSecond)
	if l.arena {
		t.Errorf("the arena shut again in the same fight")
	}

	// A foe who keeps well away from it is let go sooner.
	w, p, l = bossArena(t, CreatureTitanoboa, 0, 0)
	wake(t, w, p, l)
	l.c.X, l.c.Y = l.x+3, l.y
	n := 0
	for ; l.arena && n < ArenaMaxTicks; n++ {
		p.X, p.Y = l.x-11, l.y
		l.c.X, l.c.Y = l.x+4, l.y // held back
		w.Step()
	}
	if n > arenaCalmTicks+2*TicksPerSecond {
		t.Errorf("kept away from the boss, the arena opened after %d ticks", n)
	}
}

func TestTheWhirlpoolCarriesBoatsBackIn(t *testing.T) {
	w, p, l := bossArena(t, CreatureMegalodon, 0, 0)
	p.dev.god = true
	b := w.boatByID(p.boat)
	b.X, b.Y = l.x-8, l.y
	p.X, p.Y = b.X, b.Y
	p.input.MoveX = -1
	far := 0.0
	for range 40 * TicksPerSecond {
		b.HP = b.Kind.MaxHP()
		w.Step()
		if !l.arena {
			continue
		}
		far = max(far, math.Hypot(p.X-l.x, p.Y-l.y))
	}
	if !l.awake {
		t.Fatalf("the shark never woke")
	}
	if far < MegaArena-WhirlBand || far > MegaArena+0.5 {
		t.Errorf("sailing out of the shut arena the boat got %.1f tiles from the lair, want held near %.0f", far, MegaArena)
	}
	// Once it opens, the boat sails free.
	l.arenaT = ArenaMaxTicks
	for range 20 * TicksPerSecond {
		b.HP = b.Kind.MaxHP()
		w.Step()
	}
	if d := math.Hypot(p.X-l.x, p.Y-l.y); l.arena || d < MegaArena+2 {
		t.Errorf("with the arena open (shut: %v) the boat is %.1f tiles out", l.arena, d)
	}
}

func TestSmallAnimalsFleeTitanoboa(t *testing.T) {
	w, p, l := bossArena(t, CreatureTitanoboa, 0, 0)
	p.X, p.Y = l.x-16, l.y // near enough to watch, too far to wake it
	r := w.addCreature(CreatureRabbit, l.x+4, l.y+1)
	stepN(w, 4*TicksPerSecond)
	if l.awake {
		t.Fatalf("the snake woke")
	}
	if d := math.Hypot(r.X-l.c.X, r.Y-l.c.Y); d < 7 {
		t.Errorf("a rabbit 4 tiles from the sleeping snake is %.1f tiles from it 4 seconds on", d)
	}
}

func TestTitanoboaHuntsNearItsLairAndComesHome(t *testing.T) {
	w, p, l := bossArena(t, CreatureTitanoboa, 0, 0)
	p.X, p.Y = l.x-20, l.y-14
	c := l.c
	w.addCreature(CreatureRabbit, l.x+10, l.y+2)
	l.idle = titanHuntEvery
	out := 0.0
	for range 40 * TicksPerSecond {
		w.Step()
		out = max(out, math.Hypot(c.X-l.x, c.Y-l.y))
		if l.awake {
			t.Fatalf("the snake woke")
		}
	}
	if out < 4 {
		t.Errorf("the snake went %.1f tiles from its nest after a rabbit", out)
	}
	if d := math.Hypot(c.X-l.x, c.Y-l.y); d > 3.5 || c.State != CreatureIdle {
		t.Errorf("after the hunt it is %.1f tiles from its nest, state %d; want coiled at home", d, c.State)
	}
}

func TestTheSharkCirclesItsWater(t *testing.T) {
	w, p, l := bossArena(t, CreatureMegalodon, 0, 0)
	b := w.boatByID(p.boat)
	b.X, b.Y = l.x-30, l.y
	p.X, p.Y = b.X, b.Y
	c := l.c
	turned, last, snapped := 0.0, math.Atan2(c.Y-l.y, c.X-l.x), false
	for range 30 * TicksPerSecond {
		w.Step()
		if d := math.Hypot(c.X-l.x, c.Y-l.y); d > megaCircle+2 {
			t.Fatalf("idle, the shark is %.1f tiles from its lair", d)
		}
		a := math.Atan2(c.Y-l.y, c.X-l.x)
		turned += math.Remainder(a-last, 2*math.Pi)
		last = a
		snapped = snapped || c.State == CreatureStrike
	}
	if l.awake {
		t.Fatalf("the shark woke for a boat 30 tiles off")
	}
	if turned < math.Pi {
		t.Errorf("in 30 seconds the shark went %.1f radians round its lair", turned)
	}
	if !snapped {
		t.Errorf("it never came up to snap at a fish")
	}
}
