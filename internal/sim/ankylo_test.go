package sim

import (
	"math"
	"slices"
	"testing"
)

// ankyloAt puts an Ankylosaurus on the flat world facing left or right,
// calm and well fed, and a player beside it at (px, 16.5).
func ankyloAt(t *testing.T, x float64, left bool, px float64) (*World, *Creature, *Player) {
	t.Helper()
	w := flatWorld()
	c := w.addCreature(CreatureAnkylo, x, 16.5)
	c.FaceLeft = left
	c.fed = true // not off to graze or drink in the middle of a test
	id := w.AddPlayer("a")
	p := w.players[id]
	p.X, p.Y = px, 16.5
	p.Inv = Inventory{}
	return w, c, p
}

func TestAnkylosaursComeInSmallGroupsInTheNewestWorlds(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: SizeLarge, Height: SizeLarge, Style: StyleLatest})
	if err != nil {
		t.Fatal(err)
	}
	cs := creatureByKind(w, CreatureAnkylo)
	t.Logf("%d ankylosaurs, the world tops up to %d", len(cs), w.fauna.target[CreatureAnkylo])
	if len(cs) < creatureMinPerKind {
		t.Fatalf("%d ankylosaurs, want at least %d", len(cs), creatureMinPerKind)
	}
	groups := map[[2]float64]int{}
	for _, c := range cs {
		if tl := w.TileAt(int(c.homeX), int(c.homeY)); tl != TileForest && tl != TileGrass && tl != TileMarsh {
			t.Errorf("a group gathers on tile %d", tl)
		}
		if !w.creatureFits(CreatureAnkylo, c.X, c.Y) {
			t.Errorf("one at (%.1f, %.1f) is outside its habitat", c.X, c.Y)
		}
		if math.Hypot(c.X-w.spawnX, c.Y-w.spawnY) < ankyloDef.away-4 {
			t.Errorf("one at (%.1f, %.1f) is right by where players start", c.X, c.Y)
		}
		groups[[2]float64{c.homeX, c.homeY}]++
	}
	sizes := map[int]bool{}
	for _, n := range groups {
		if n < 1 || n > ankyloDef.herd {
			t.Errorf("a group of %d, want 1 to %d", n, ankyloDef.herd)
		}
		sizes[n] = true
	}
	if len(groups) > 2 && len(sizes) < 2 {
		t.Errorf("every one of %d groups has the same size: they should vary", len(groups))
	}
	// The count follows the islands, as the other grazers' do (see wilds.go):
	// the world tops up to as many as it was made with.
	if w.fauna.target[CreatureAnkylo] != len(cs) {
		t.Errorf("target %d, want the %d it was made with", w.fauna.target[CreatureAnkylo], len(cs))
	}
}

func TestAnOlderWorldGetsItsAnkylosaursAsItIsToppedUp(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: SizeMedium, Height: SizeMedium, Style: StyleWoods})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(creatureByKind(w, CreatureAnkylo)); n != 0 {
		t.Fatalf("%d ankylosaurs in a world of an older style as it is made: it must be made as before", n)
	}
	if w.fauna.target[CreatureAnkylo] < creatureMinPerKind {
		t.Fatalf("target %d: an older world should still top up to some", w.fauna.target[CreatureAnkylo])
	}
	for range 40 {
		w.repopulate()
	}
	if n := len(creatureByKind(w, CreatureAnkylo)); n == 0 {
		t.Fatal("no ankylosaur came as the world topped up its animals")
	}
}

func TestAnAnkylosaurMindsNobodyAtADistance(t *testing.T) {
	w, c, p := ankyloAt(t, 22.5, false, 16.5)
	stepN(w, 3*TicksPerSecond)
	if c.State == CreatureRoar || c.State == CreatureStrike || c.State == CreatureChase || p.Health < MaxHealth {
		t.Fatalf("state %d, health %.0f: six tiles off, it should go on grazing", c.State, p.Health)
	}
}

func TestAnAnkylosaurTurnsItsClubOnSomeoneTooClose(t *testing.T) {
	// It faces left, its head towards the player three tiles off.
	w, c, p := ankyloAt(t, 19.6, true, 16.5)
	w.Step()
	if c.State != CreatureRoar || c.FaceLeft {
		t.Fatalf("state %d, facing left %v: it should turn its back, the club towards the player", c.State, c.FaceLeft)
	}
	stepN(w, 2*TicksPerSecond)
	if p.Health < MaxHealth || c.State == CreatureStrike {
		t.Fatalf("health %.0f, state %d: out of the club's reach it only warns", p.Health, c.State)
	}
	// Holding berries out, a player may come close: it lets them.
	inHand(p, ItemBerry, 5)
	p.X = 18
	stepN(w, AnkyloGuardTicks+TicksPerSecond)
	if c.State == CreatureStrike || c.State == CreatureRoar || p.Health < MaxHealth {
		t.Fatalf("state %d, health %.0f: someone holding out its food is no threat", c.State, p.Health)
	}
}

func TestTheClubIsWoundUpThenSwungHard(t *testing.T) {
	w, c, p := ankyloAt(t, 19.6, false, 17.6) // its tail to the player, two tiles off
	w.Step()
	if c.State != CreatureStrike || !c.windingUp() {
		t.Fatalf("state %d: in reach of its club it should crouch and draw it back", c.State)
	}
	stepN(w, AnkyloWindup-2)
	if p.Health < MaxHealth {
		t.Fatalf("health %.0f: the club landed before the wind-up was done", p.Health)
	}
	x0 := p.X
	stepN(w, 3)
	if got := MaxHealth - p.Health; got < AnkyloClubDamage-1 {
		t.Fatalf("lost %.0f health, want the club's %v", got, AnkyloClubDamage)
	}
	if CreatureKind(p.NoticeArg) != CreatureAnkylo || p.Notice != NoticeBitten {
		t.Errorf("notice %d (%d): the player should be told what hit them", p.Notice, p.NoticeArg)
	}
	stepN(w, 4) // the throw is eased over a few ticks: see knock.go
	if p.X > x0-AnkyloClubKnock*0.8 {
		t.Errorf("thrown from %.2f to %.2f: the club should throw a player %.1f tiles", x0, p.X, AnkyloClubKnock)
	}
	if AnkyloClubDamage <= StegoTailDamage || AnkyloClubDamage <= TrikeGoreDamage {
		t.Error("the club should hit harder than the other grazers")
	}
}

func TestStepOutOfReachOrRoundToItsHeadAndTheClubMisses(t *testing.T) {
	w, c, p := ankyloAt(t, 19.6, false, 17.6)
	w.Step()
	if !c.windingUp() {
		t.Fatal("it should be winding up")
	}
	p.X = 13 // back out of reach
	stepN(w, AnkyloWindup+2)
	if p.Health < MaxHealth {
		t.Fatalf("health %.0f: stepping out of reach during the wind-up should be safe", p.Health)
	}
	w, c, p = ankyloAt(t, 19.6, false, 17.6)
	w.Step()
	p.X = 21.6 // round to its head
	stepN(w, AnkyloWindup+2)
	if p.Health < MaxHealth {
		t.Fatalf("health %.0f: the club swings behind it, not before its head", p.Health)
	}
	// Its swing done, it is slow to turn round: a moment at its head.
	stepN(w, AttackTicks)
	if c.FaceLeft {
		t.Fatal("it turned round at once after its swing")
	}
	stepN(w, AnkyloTurnTicks+2)
	if !c.FaceLeft {
		t.Fatal("it should turn its back on the player at its head in the end")
	}
}

func TestItsArmourStopsHalfABlowButNotAtItsHead(t *testing.T) {
	w, c, p := ankyloAt(t, 19.6, false, 17.6) // facing right: the player at its tail
	if got := armoured(p, c, 40); got != 20 || p.Notice != NoticeArmoured {
		t.Fatalf("a blow of 40 at its tail does %d (notice %d), want 20 and the player told", got, p.Notice)
	}
	p.Y = 14.5 // above its back
	p.X = 19.6
	if got := armoured(p, c, 40); got != 20 {
		t.Fatalf("a blow of 40 on its back does %d, want 20", got)
	}
	p.X, p.Y = 21.8, 16.8 // before its head
	if got := armoured(p, c, 40); got != 40 {
		t.Fatalf("a blow of 40 at its head does %d, want all of it", got)
	}
	// Through the game's own blows, with a knife: about half as much at the tail.
	inHand(p, SwordItem(TierIron), 1)
	took := func(x, y float64) int {
		total := 0
		for range 200 {
			c.HP, c.FaceLeft, c.cool = c.Kind.MaxHP(), false, 0
			p.X, p.Y = x, y
			w.catch(p, c)
			total += c.Kind.MaxHP() - c.HP
		}
		return total
	}
	head, tail := took(21.8, 16.5), took(17.6, 16.5)
	if r := float64(tail) / float64(head); r < 0.42 || r > 0.58 {
		t.Fatalf("its tail took %d, its head %d: the plates should stop half", tail, head)
	}
	_ = w
}

func TestTheClubStaggersEvenATyrant(t *testing.T) {
	w := flatWorld()
	a := w.addCreature(CreatureAnkylo, 14.5, 16.5)
	a.FaceLeft = true // its tail towards the tyrant coming from the right
	a.fed = true
	rex := w.addCreature(CreatureRex, 21.5, 16.5)
	rex.FaceLeft = true
	rex.fed = true
	rex.prey = a.ID
	hp := rex.HP
	reeled := false
	for range 8 * TicksPerSecond {
		w.Step()
		if rex.State == CreatureHurt {
			reeled = true
			break
		}
	}
	if !reeled || rex.HP > hp-AnkyloDefendDamage*HealthScale {
		t.Fatalf("rex state %d, hp %d of %d: the club should land on a hunter and make it reel", rex.State, rex.HP, hp)
	}
	if !rex.FaceLeft {
		t.Error("a clubbed hunter is thrown back, not turned round")
	}
	if a.HP == a.Kind.MaxHP() && a.State != CreatureStrike && a.State != CreatureRoar {
		t.Errorf("ankylosaur state %d: it should stand its ground", a.State)
	}
}

func TestAnAnkylosaurGivesItsPlates(t *testing.T) {
	loot := (&World{}).bodyLoot(CreatureAnkylo)
	has := func(k ItemKind) int {
		n := 0
		for _, in := range loot {
			if in.Item == k {
				n += in.Count
			}
		}
		return n
	}
	if has(ItemOsteoderm) < 2 || has(ItemMeat) < 6 || has(ItemHide) == 0 || has(ItemBone) == 0 || has(ItemDinoHide) == 0 || has(HeadItem(CreatureAnkylo)) != 1 {
		t.Fatalf("a butchered ankylosaur gives %+v", loot)
	}
	if ItemOsteoderm.Def().Name == "" || !ItemOsteoderm.Valid() {
		t.Fatal("its plates have no name")
	}
	// Killed and butchered in the game, the plates fall on the ground.
	w, c, p := ankyloAt(t, 19.6, false, 21.8)
	inHand(p, SwordItem(TierIron), 1)
	c.HP = 1
	w.catch(p, c)
	var body *Entity
	for _, e := range w.entities {
		if e.Kind == EntityCorpse && CreatureKind(e.Var) == CreatureAnkylo {
			body = e
		}
	}
	if body == nil {
		t.Fatal("no body left to butcher")
	}
	for range 20 {
		if _, ok := w.entities[body.ID]; !ok {
			break
		}
		w.butcher(p, body)
	}
	plates := 0
	for _, d := range w.drops {
		if d.Item == ItemOsteoderm {
			plates += d.Count
		}
	}
	if plates != has(ItemOsteoderm) {
		t.Fatalf("%d plates on the ground, want %d", plates, has(ItemOsteoderm))
	}
	// Bone armour does not need them: a rare beast must not gate a tier.
	r, _ := RecipeByID(ArmorRecipe(TierIron))
	if slices.ContainsFunc(r.Inputs, func(in Ingredient) bool { return in.Item == ItemOsteoderm }) {
		t.Fatalf("bone armour is made of %+v: it should not need the plates", r.Inputs)
	}
}

func TestAnAnkylosaurIsTamedWithBerriesAndRidden(t *testing.T) {
	w, c, p := ankyloAt(t, 18.2, false, 16.5)
	inHand(p, ItemBerry, 10)
	hitUntil(w, p.ID, 20*TicksPerSecond, func() bool { return c.Tamed() })
	if !c.Tamed() || p.Health < MaxHealth {
		t.Fatalf("tamed %v (%d meals), health %.0f: berries held out should tame it without a blow", c.Tamed(), c.Tame, p.Health)
	}
	if !CreatureAnkylo.Rideable() || CreatureAnkylo.tameDef().ride >= CreatureToro.tameDef().ride {
		t.Error("it should be ridden, slower than the bull-frill")
	}
	if _, ok := devBeastAliases["ankylo"]; !ok {
		t.Error("the dev console should know it as ankylo")
	}
}
