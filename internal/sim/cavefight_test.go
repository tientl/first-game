package sim

import (
	"testing"
)

func fightWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	w, err := Generate(GenConfig{Seed: seed, Width: SizeMedium, Height: SizeMedium, Style: StyleCaveFight})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func countKinds(w *World) map[CreatureKind]int {
	m := map[CreatureKind]int{}
	for _, c := range w.Creatures() {
		if w.Underground(int(c.Y)) {
			m[c.Kind]++
		}
	}
	return m
}

func TestEveryCaveKindHasDefAndSkills(t *testing.T) {
	for k := CreatureArthro; k <= CreatureSkelRex; k++ {
		d := k.def()
		if d.name == "" || d.hp <= 0 && k != CreatureNone || d.radius <= 0 {
			t.Errorf("kind %d has no def", k)
		}
		if !k.IsCaveBeast() {
			t.Errorf("kind %d has no spec", k)
		}
		for _, id := range SkillsOf(k) {
			s := Skill(id)
			if s.Windup < 6 || s.Windup > 32 || s.Recover < 6 {
				t.Errorf("%s/%s: windup %d recover %d", d.name, s.Name, s.Windup, s.Recover)
			}
			if s.Class == SkillSpecial && (!s.Zone || s.Windup < 22) && s.Shape != ZoneNone {
				t.Errorf("%s/%s: a special move needs a red zone and 1.2 s", d.name, s.Name)
			}
		}
	}
}

func TestCaveFightSpawnsOnlyInNewStyle(t *testing.T) {
	newer := map[CreatureKind]int{}
	for _, seed := range []uint64{3, 7, 11, 21} {
		for k, n := range countKinds(fightWorld(t, seed)) {
			newer[k] += n
		}
	}
	boss, queens, skel := 0, 0, 0
	for k, n := range newer {
		switch {
		case k.IsSpiderQueen():
			queens += n
		case k.IsSkeleton():
			skel += n
		case k == CreatureArthroQueen || k == CreatureScorpQueen || k == CreatureMegalania || k == CreatureLionAlpha || k == CreatureBlindCroc || k == CreatureAlphaBear:
			boss += n
		}
	}
	t.Logf("kinds underground: %v", newer)
	if boss == 0 {
		t.Error("no cave boss in a new world")
	}
	if queens == 0 {
		t.Error("no spider queen")
	}
	w, err := Generate(GenConfig{Seed: 7, Width: SizeMedium, Height: SizeMedium, Style: StyleCaveWays})
	if err != nil {
		t.Fatal(err)
	}
	for k, n := range countKinds(w) {
		if k.IsCaveBeast() && n > 0 {
			t.Errorf("older style has %d of kind %d", n, k)
		}
	}
}

func TestBossGuardsHoard(t *testing.T) {
	w := fightWorld(t, 7)
	var guard *Creature
	for _, c := range w.fauna.creatures {
		if c.cv.guard != 0 {
			guard = c
		}
	}
	if guard == nil {
		t.Skip("no guarded hoard in this world")
	}
	e := w.entities[guard.cv.guard]
	id := w.AddPlayer("t")
	p := w.players[id]
	p.X, p.Y = float64(e.TX)+0.5, float64(e.TY)+1.5
	w.openTreasure(p, e)
	if p.Notice != NoticeGuarded {
		t.Fatal("chest opened with its guardian alive")
	}
	w.fell(guard)
	w.caveFelled(guard)
	w.fauna.creatures = w.fauna.creatures[:len(w.fauna.creatures)]
	if w.hoardGuardian(e) != nil || e.Fuel != 1 {
		t.Fatal("the hoard is still guarded after the boss fell")
	}
}

// arena is a world with one player and the first-seen spot of open floor
// round the surface spawn, where beasts may be set for a test.
func arena(t *testing.T, held ItemKind) (*World, *Player) {
	t.Helper()
	w := fightWorld(t, 5)
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	if held != ItemNone {
		p.Inv[0] = Slot{Item: held, Count: 1}
	}
	p.Selected = 0
	return w, p
}

func TestSkeletonWeakToBluntAndFire(t *testing.T) {
	w, p := arena(t, ItemBattleAxe)
	c := &Creature{Kind: CreatureSkelRex, HP: 1000}
	blunt := w.caveBlow(p, c, 70)
	p.Inv[0] = Slot{Item: ItemSpear, Count: 1}
	sharp := w.caveBlow(p, c, 20)
	if blunt <= 70 || sharp >= 20 {
		t.Fatalf("blunt %d (want over 70), spear %d (want under 20)", blunt, sharp)
	}
	p.Inv[0] = Slot{Item: ItemTorch, Count: 1}
	p.torchFuel = 100
	if fire := w.caveBlow(p, c, FistDamage); fire <= 2*FistDamage {
		t.Fatalf("a torch does %d to a skeleton", fire)
	}
}

func TestCoilBlocksAndRecoveryExposes(t *testing.T) {
	w, p := arena(t, ItemSpear)
	c := &Creature{Kind: CreatureArthro, HP: 400}
	coil, ok := FindSkill(CreatureArthro, "coil")
	if !ok {
		t.Fatal("no coil")
	}
	open := armoured(p, c, 40)
	c.sk = skillState{id: coil, phase: PhaseActive}
	coiled := armoured(p, c, 40)
	if coiled*3 > open {
		t.Fatalf("coiled takes %d of %d", coiled, open)
	}
	c.sk.phase = PhaseRecover
	if exposed := w.caveBlow(p, c, 40); exposed <= 40 {
		t.Fatalf("an uncoiling beast takes %d of 40", exposed)
	}
}

func TestCentipedeShedsLegsAndSlows(t *testing.T) {
	w, _ := arena(t, ItemNone)
	c := &Creature{Kind: CreatureCentipede, HP: 80}
	for range 6 {
		w.caveWounded(c, 20, nil)
	}
	if c.cv.legs != LegsMax {
		t.Fatalf("legs shed: %d", c.cv.legs)
	}
	w.caveWounded(c, 5, nil)
	if c.cv.legs != LegsMax {
		t.Fatal("a light blow shed a leg past the limit")
	}
}

func TestEggSacHatchesUnlessBroken(t *testing.T) {
	w, p := arena(t, ItemNone)
	spot := func() (float64, float64) {
		for r := 2.0; r < 30; r += 0.5 {
			if w.creatureFits(CreatureEggSac, p.X+r, p.Y) {
				return p.X + r, p.Y
			}
		}
		t.Fatal("no room")
		return 0, 0
	}
	x, y := spot()
	sac := w.addCreature(CreatureEggSac, x, y)
	before := len(w.fauna.creatures)
	for i := 0; i < eggHatchTicks+5 && !sac.gone; i++ {
		w.stepEggSac(sac)
	}
	if !sac.gone || len(w.fauna.creatures) <= before {
		t.Fatalf("the sac did not hatch: gone %v, creatures %d -> %d", sac.gone, before, len(w.fauna.creatures))
	}
	sac2 := w.addCreature(CreatureEggSac, x, y)
	n := len(w.fauna.creatures)
	w.wound(sac2, sac2.HP, p) // broken
	w.fauna.creatures = w.fauna.creatures[:len(w.fauna.creatures)]
	if len(w.fauna.creatures) != n || !sac2.gone {
		t.Fatal("a broken sac should vanish without spiderlings")
	}
}

func TestCollapseWarnsBeforeItHits(t *testing.T) {
	w, p := arena(t, ItemNone)
	w.cf.spots = append(w.cf.spots, collapseSpot{x: p.X, y: p.Y, r: 10})
	p.Health = 100
	w.caveNoise(p.X, p.Y, collapseNoise+1)
	w.stepCaveFight()
	hz := w.Hazards(p.X-5, p.Y-5, p.X+5, p.Y+5)
	if len(hz) != 1 || hz[0].Kind != HazCollapse || hz[0].Left < CollapseWarn-2 {
		t.Fatalf("no warning: %+v", hz)
	}
	for i := 0; i < CollapseWarn-3; i++ {
		w.stepCaveFight()
		if p.Health < 100 {
			t.Fatal("hurt before the warning ran out")
		}
	}
	for i := 0; i < 6; i++ {
		w.stepCaveFight()
	}
	if p.Health >= 100 {
		t.Fatal("the roof never came down on someone who stood still")
	}
	// and a player who ran is not hurt
	w2, q := arena(t, ItemNone)
	w2.addHazard(HazCollapse, q.X+20, q.Y, CollapseRadius, 2)
	q.Health = 100
	for i := 0; i < 5; i++ {
		w2.stepCaveFight()
	}
	if q.Health < 100 {
		t.Fatal("hurt away from the collapse")
	}
}

func TestFloorWebBurnsWithTorch(t *testing.T) {
	w, p := arena(t, ItemTorch)
	w.addHazard(HazWeb, p.X, p.Y, 1.5, -1)
	w.stepCaveFight()
	if !p.Webbed() {
		t.Fatal("not webbed in a web")
	}
	p.torchFuel = 500
	w.stepCaveFight()
	w.stepCaveFight()
	if len(w.Hazards(p.X-3, p.Y-3, p.X+3, p.Y+3)) != 0 {
		t.Fatal("a lit torch did not burn the web")
	}
}

func TestCaveFightDeterministicAndSaves(t *testing.T) {
	a, b := fightWorld(t, 11), fightWorld(t, 11)
	if worldHash(a) != worldHash(b) {
		t.Fatal("two worlds of one seed differ")
	}
	for range 200 {
		a.Step()
		b.Step()
	}
	if worldHash(a) != worldHash(b) {
		t.Fatal("two stepped worlds diverged")
	}
	d := a.Save()
	if d.Version != SaveVersion || d.Style != StyleCaveFight {
		t.Fatalf("save is version %d style %d", d.Version, d.Style)
	}
	if _, err := Load(d); err != nil {
		t.Fatal(err)
	}
}

func TestFallenBossStaysFallenThroughLoad(t *testing.T) {
	w := fightWorld(t, 7)
	var guard *Creature
	for _, c := range w.fauna.creatures {
		if c.cv.guard != 0 {
			guard = c
		}
	}
	if guard == nil {
		t.Skip("no guarded hoard")
	}
	w.fell(guard)
	w.caveFelled(guard)
	l, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range l.fauna.creatures {
		if c.cv.guard == guard.cv.guard && !c.gone {
			t.Fatal("the boss came back after a load")
		}
	}
}

func TestBatsBurstFromTorch(t *testing.T) {
	w, p := arena(t, ItemTorch)
	var bats []*Creature
	for i := 0; i < 4; i++ {
		for r := 2.0; r < 40; r += 0.4 {
			if w.creatureFits(CreatureBat, p.X+r, p.Y+float64(i)*0.3) {
				c := w.addCreature(CreatureBat, p.X+r, p.Y+float64(i)*0.3)
				c.State = CreatureSleep
				bats = append(bats, c)
				break
			}
		}
	}
	p.torchFuel = 500
	// bring the first one close
	bats[0].X, bats[0].Y = p.X+3, p.Y
	if !w.creatureFits(CreatureBat, bats[0].X, bats[0].Y) {
		t.Skip("no floor beside the player")
	}
	w.stepBat(bats[0], caveSpecOf(CreatureBat))
	if bats[0].cv.swarm == 0 {
		t.Fatal("bat stayed roosting under a torch")
	}
}
