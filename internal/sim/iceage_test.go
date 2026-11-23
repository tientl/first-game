package sim

import "testing"

func TestTheNorthFreezesOver(t *testing.T) {
	w := genOpen(t, 1, SizeLarge)
	n, sumY := 0, 0.0
	for i, tl := range w.tiles {
		if tl == TileSnow {
			n++
			sumY += float64(i / w.width)
		}
	}
	if n == 0 {
		t.Fatal("no snow in a new world")
	}
	if avg := sumY / float64(n); avg > float64(w.height)*0.45 {
		t.Errorf("snow centred on row %.0f of %d: it should lie to the north", avg, w.height)
	}
	if w.TileAt(int(w.spawnX), int(w.spawnY)) == TileSnow {
		t.Error("the spawn is on the snow")
	}
	for _, k := range []CreatureKind{CreatureMammoth, CreatureSabre} {
		if got := len(creatureByKind(w, k)); got < creatureMinPerKind {
			t.Errorf("%d %s", got, k.Name())
		}
	}
	old, err := Generate(GenConfig{Seed: 1, Width: SizeLarge, Height: SizeLarge, Style: StyleOpen})
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range old.tiles {
		if tl == TileSnow {
			t.Fatal("snow in a world made before the ice age: old saves would change")
		}
	}
}

// snowWorld is flatWorld under snow, from column 12 eastwards.
func snowWorld() *World {
	w := flatWorld()
	for i := range w.tiles {
		if i%w.width >= 12 {
			w.tiles[i] = TileSnow
		}
	}
	return w
}

func TestTheSnowIsFreezingWithoutFur(t *testing.T) {
	w := snowWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.Step()
	if !p.Cold || Ambient(w.tick) < ColdAmbient {
		t.Fatalf("cold %v by day on the snow: it should freeze", p.Cold)
	}
	p.Inv.Add(ItemFurCoat, 1)
	w.Step()
	if p.Cold {
		t.Fatal("cold in a fur coat")
	}
	p.Inv.Take(ItemFurCoat, 1)
	p.X = 6.5 // off the snow
	w.Step()
	if p.Cold {
		t.Fatal("cold on the grass by day")
	}
	p.X = 16.5
	mount := tamedNear(w, id, CreatureMammoth, 17.5, 16.5)
	p.Inv.Add(ItemSaddle, 1)
	w.Apply(id, Command{Kind: CmdRide})
	w.Step()
	if p.Mount() != mount.ID || p.Cold {
		t.Fatalf("mount %d cold %v: a mammoth's wool keeps its rider warm", p.Mount(), p.Cold)
	}
	if w.speedAt(16.5, 16.5) >= 1 {
		t.Error("deep snow should slow walking")
	}
}

func TestMammothsGoreWhoeverHurtsThem(t *testing.T) {
	w := snowWorld()
	a := w.addCreature(CreatureMammoth, 18.4, 16.5)
	id := w.AddPlayer("a")
	hitUntil(w, id, 1, func() bool { return false })
	if a.HP == mammothDef.hp*HealthScale {
		t.Fatal("the hit missed")
	}
	stepN(w, 6*TicksPerSecond)
	if p := mustPlayer(t, w, id); p.Health > MaxHealth-MammothTuskDamage+1 {
		t.Fatalf("health %.0f: an angry mammoth should gore", p.Health)
	}
}

func TestASabreToothStalksThenSprings(t *testing.T) {
	w := snowWorld()
	s := w.addCreature(CreatureSabre, 24.5, 16.5)
	id := w.AddPlayer("a")
	w.players[id].X = 17.5
	bitten := false
	for range 8 * TicksPerSecond {
		w.Step()
		if p := w.players[id]; p.Health < MaxHealth-SabreBiteDamage*0.5 && CreatureKind(p.NoticeArg) == CreatureSabre {
			bitten = true
			break
		}
	}
	if !bitten {
		t.Fatalf("never bitten; sabre at (%.1f, %.1f) state %d", s.X, s.Y, s.State)
	}
	s.HP = SabreCowerBelow
	stepN(w, SabreBackOff+2)
	if s.State != CreatureCower {
		t.Fatalf("state %d: a beaten sabre-tooth should cower", s.State)
	}
	p := w.players[id]
	inHand(p, ItemMeat, 6)
	s.X, s.Y = p.X+1.2, p.Y
	hitUntil(w, id, 8*TicksPerSecond, func() bool { return s.Tamed() })
	if !s.Tamed() {
		t.Fatalf("tame %d: four meals should tame a cowering sabre-tooth", s.Tame)
	}
}

func TestTheIvorySpearReachesFurthestAndHitsHard(t *testing.T) {
	w := flatWorld()
	p := w.players[w.AddPlayer("a")]
	inHand(p, ItemIvorySpear, 1)
	if damageWith(p) <= SpearDamage || p.reachBonus() <= SpearReachBonus {
		t.Fatalf("damage %d reach %.1f: ivory should beat the stone spear", damageWith(p), p.reachBonus())
	}
	if r, ok := RecipeByID(RecipeIvorySpear); !ok || r.Gives != ItemIvorySpear {
		t.Fatalf("recipe %+v", r)
	}
}
