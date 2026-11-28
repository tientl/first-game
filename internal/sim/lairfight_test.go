package sim

import (
	"math"
	"testing"
)

// The lairs of StyleLairs and their fights: temple.go, atoll.go,
// titanfight.go, sharkfight.go, throw.go.

// lairsWorld is a world of StyleLairs at its usual size, with both lairs.
func lairsWorld(t *testing.T) *World {
	t.Helper()
	w, err := Generate(GenConfig{Seed: 42, Width: SizeHuge, Height: SizeHuge, Style: StyleLairs})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestTheTempleAndTheAtollAreLaidOut(t *testing.T) {
	w := lairsWorld(t)
	// The temple.
	l := &w.bosses.lairs[0]
	s := &l.site
	if !l.ok || s.layout != LayoutTemple {
		t.Fatalf("Titanoboa's lair: ok %v layout %d, want the temple", l.ok, s.layout)
	}
	// The whole of a screen (27 by 20 tiles at 1080p, a little more drawn
	// back in a fight) fits in the courtyard with room over.
	if 2*s.rx < 34 || 2*s.ry < 26 {
		t.Errorf("the courtyard is %.0f by %.0f tiles, want 34 by 26 at least", 2*s.rx, 2*s.ry)
	}
	if q := s.q(15, 11); q > 1 {
		t.Errorf("a screen drawn back (30 by 22 tiles) does not fit in the courtyard: its corner at q %.2f", q)
	}
	trees, columns, pillars, tree := 0, 0, 0, false
	for _, e := range w.entities {
		d := math.Hypot(float64(e.TX)+0.5-l.x, float64(e.TY)+0.5-l.y)
		switch {
		case s.q(float64(e.TX)+0.5-l.x, float64(e.TY)+0.5-l.y) < 0.97 && (e.Kind == EntityTree || e.Kind == EntityGiantTree || e.Kind == EntityPalm):
			trees++
		case e.Kind == EntityLairProp && e.Var == LairColumnM && d < s.r:
			columns++
		case e.Kind == EntityLairProp && e.Var == LairPillar && d < s.r:
			pillars++
		case e.Kind == EntityLairProp && e.Var == LairDeadTree && d < s.r:
			tree = true
		}
	}
	if trees > 0 {
		t.Errorf("%d trees stand in the courtyard: it should be open to the sky", trees)
	}
	if columns < 4 || columns > 6 {
		t.Errorf("%d fallen columns in the courtyard, want 4 to 6 to hide behind", columns)
	}
	if pillars < 4 || !tree {
		t.Errorf("%d pillars, the dead tree %v: the temple wants its colonnade and its landmark", pillars, tree)
	}
	if !s.inPool(l.c.X, l.c.Y) || w.TileAt(int(s.poolX), int(s.poolY)) != TileWater {
		t.Errorf("the snake does not sleep in its pool")
	}
	if !s.camp {
		t.Fatalf("the temple has no camp")
	}
	if d := math.Hypot(s.campX-l.x, s.campY-l.y); d < 30 || d > 40 {
		t.Errorf("the camp is %.1f tiles from the temple, want 30 to 40", d)
	}
	if !w.standable(s.campX, s.campY) || !w.standable(s.campX, s.campY+1) || !w.standable(s.campX+1, s.campY) {
		t.Errorf("no room at the camp for a fire and a tent")
	}

	// The atoll.
	m := &w.bosses.lairs[5]
	a := &m.site
	if !m.ok || a.layout != LayoutAtoll {
		t.Fatalf("the Megalodon's lair: ok %v layout %d, want the atoll", m.ok, a.layout)
	}
	if across := 2 * a.lagoon; across < 34 {
		t.Errorf("the lagoon is %.0f tiles across, want 34 at least", across)
	}
	if d := math.Hypot(15, 11); d > m.viewRX() {
		t.Errorf("a screen drawn back does not fit in the atoll's bounds (%.1f)", m.viewRX())
	}
	if len(a.reef) < 100 {
		t.Errorf("only %d footholds of reef", len(a.reef))
	}
	channels, wreck, mast := 0, 0, false
	for ang := 0.0; ang < 2*math.Pi; ang += 0.02 {
		x, y := m.x+math.Cos(ang)*atollRing(), m.y+math.Sin(ang)*atollRing()
		if w.TileAt(int(x), int(y)) == TileShallow {
			channels++
		}
	}
	for _, e := range w.entities {
		if e.Kind == EntityLairProp && (e.Var == LairWreck || e.Var == LairWreckPart) {
			wreck++
		}
		if e.Kind == EntityLairProp && e.Var == LairMast {
			mast = true
		}
	}
	if channels == 0 {
		t.Errorf("the ring of reef has no channel a boat can come in by")
	}
	if wreck < 12 || !mast {
		t.Errorf("the wreck: %d tiles of deck, a mast %v", wreck, mast)
	}
	if !a.camp {
		t.Fatalf("the atoll has no camp")
	}
	if d := math.Hypot(a.campX-m.x, a.campY-m.y); d < 30 || d > 40 || !w.standable(a.campX, a.campY) {
		t.Errorf("the camp's cay is %.1f tiles off (standable %v), want 30 to 40", d, w.standable(a.campX, a.campY))
	}
	if !w.creatureFits(CreatureMegalodon, m.x, m.y) || w.creatureFits(CreatureMegalodon, m.x+atollRing(), m.y) && w.TileAt(int(m.x+atollRing()), int(m.y)) != TileShallow {
		t.Errorf("the shark does not keep to its lagoon")
	}
	if l.c == nil || m.c == nil {
		t.Errorf("a boss was not raised at its lair")
	}
}

// templeScene is a flat jungle with Titanoboa's temple in its middle, the
// snake out on the paving south of its pool, awake, and a player.
func templeScene(t *testing.T) (*World, *Player, *bossLair) {
	t.Helper()
	const n = 96
	w := newWorld(n, n)
	w.seed, w.style = calmSeed(), StyleLairs
	for i := range w.tiles {
		w.tiles[i] = TileForest
	}
	w.spawnX, w.spawnY = 48.5, 8.5
	w.bosses.placed = true
	l := &w.bosses.lairs[0]
	l.x, l.y, l.ok = 48.5, 48.5, true
	w.buildTemple(l, 0.75)
	w.raiseBoss(0)
	p := w.players[w.AddPlayer("Thử")]
	p.Hunger = WellFedHunger - 5 // no hunger, no healing: what is lost is the boss's doing
	l.c.X, l.c.Y = 48.5, 55
	return w, p, l
}

// atollScene is the open sea with the Megalodon's atoll in its middle, and
// a player.
func atollScene(t *testing.T) (*World, *Player, *bossLair) {
	t.Helper()
	const n = 110
	w := newWorld(n, n)
	w.seed, w.style = calmSeed(), StyleLairs
	w.spawnX, w.spawnY = 55.5, 4.5
	w.bosses.placed = true
	l := &w.bosses.lairs[5]
	l.x, l.y, l.ok = 55.5, 55.5, true
	w.buildAtoll(l)
	w.raiseBoss(5)
	p := w.players[w.AddPlayer("Thử")]
	p.Hunger = WellFedHunger - 5
	return w, p, l
}

// spotNear is the nearest point to (x, y) a player can stand on.
func spotNear(t *testing.T, w *World, x, y float64) (float64, float64) {
	t.Helper()
	tx, ty, ok := w.nearestGround(x, y, 6)
	if !ok {
		t.Fatalf("no ground near (%.1f, %.1f)", x, y)
	}
	return float64(tx) + 0.5, float64(ty) + 0.5
}

// reefEdge is a foothold of lair l's reef at the lagoon's edge, with reef
// behind it too: where the shark lunges and rams.
func reefEdge(t *testing.T, w *World, l *bossLair) (float64, float64) {
	t.Helper()
	for ang := 0.1; ang < 2*math.Pi; ang += 0.13 {
		for r := l.site.lagoon - 1; r < l.site.lagoon+AtollReef; r += 0.25 {
			x, y := l.x+math.Cos(ang)*r, l.y+math.Sin(ang)*r
			tx, ty := int(x), int(y)
			if w.TileAt(tx, ty) == TileWater {
				continue
			}
			bx, by := int(x+math.Cos(ang)), int(y+math.Sin(ang))
			if l.reefTile(ty*w.width+tx) && l.reefTile(by*w.width+bx) && w.standable(float64(tx)+0.5, float64(ty)+0.5) {
				return float64(tx) + 0.5, float64(ty) + 0.5
			}
			break
		}
	}
	t.Fatal("no edge of reef by the lagoon")
	return 0, 0
}

// putFor stands p where boss move id of l's boss is made at them.
func putFor(t *testing.T, w *World, p *Player, l *bossLair, id SkillID) {
	t.Helper()
	c := l.c
	switch id {
	case titanVenom:
		p.X, p.Y = spotNear(t, w, c.X+5, c.Y)
	case titanDive:
		c.X, c.Y = l.x, l.y+l.site.poolRY+1.2
		p.X, p.Y = spotNear(t, w, l.x, l.y+l.site.poolRY+3.2)
	case titanStrike, titanTail, titanCoil, titanSlam:
		p.X, p.Y = spotNear(t, w, c.X+2.2, c.Y)
	case sharkLunge, sharkRam, sharkRamMore, sharkBreach:
		p.X, p.Y = reefEdge(t, w, l)
		c.X, c.Y = w.lagoonPoint(l, p.X, p.Y, 4)
	case sharkCapsize, sharkBite:
		p.X, p.Y = l.x+3, l.y
		c.X, c.Y = l.x-1, l.y
		if id == sharkCapsize {
			w.nextBoat++
			b := &Boat{ID: w.nextBoat, Kind: BoatRaft, X: p.X, Y: p.Y, HP: BoatRaft.MaxHP(), Rider: p.ID}
			w.boats = append(w.boats, b)
			p.boat = b.ID
		}
	}
}

func TestEveryBossMoveLandsOnlyAfterItsWindup(t *testing.T) {
	for _, id := range append(TitanSkills(), SharkSkills()...) {
		d := Skill(id)
		var w *World
		var p *Player
		var l *bossLair
		if d.Kind == CreatureTitanoboa {
			w, p, l = templeScene(t)
		} else {
			w, p, l = atollScene(t)
		}
		putFor(t, w, p, l, id)
		c := l.c
		l.awake, l.roared, l.foe = true, true, p.ID
		if d.Stages != 0 && d.Stages&1 == 0 { // a move of a later stage
			c.HP = int(0.5 * float64(c.Kind.MaxHP()))
			if d.Stages&(1<<1) == 0 {
				c.HP = int(0.2 * float64(c.Kind.MaxHP()))
			}
		}
		hp := p.Health
		boatHP := 0
		if b := w.boatByID(p.boat); b != nil {
			boatHP = b.HP
		}
		if !w.ForceSkill(c.ID, id, p.ID) {
			t.Fatalf("%s: could not begin it", d.Label)
		}
		wind := c.sk.wind
		if wind < rageWindup {
			t.Errorf("%s: wound up for %d ticks, under %d", d.Label, wind, rageWindup)
		}
		for i := 0; i < wind-1; i++ {
			w.Step()
			if p.Health < hp {
				t.Errorf("%s: struck %d ticks into a wind-up of %d", d.Label, i+1, wind)
				break
			}
		}
		landed := false
		for range d.Active + 3 {
			w.Step()
			if p.Health < hp || boatHP > 0 && p.boat == 0 {
				landed = true
				break
			}
		}
		if !landed {
			t.Errorf("%s: never landed on the player standing in it (health %.0f of %.0f)", d.Label, p.Health, hp)
		}
	}
}

func TestTitanoboasFightChangesWithItsStages(t *testing.T) {
	w, p, l := templeScene(t)
	c := l.c
	p.X, p.Y = spotNear(t, w, c.X+2.2, c.Y)
	l.awake, l.roared, l.foe = true, true, p.ID
	ready := func(id SkillID) bool {
		i := skillSlot(c.Kind, id)
		c.sk.cd[i] = 0
		return w.ready(c, i, Skill(id), p, math.Hypot(p.X-c.X, p.Y-c.Y))
	}
	if c.SkillStage() != 0 || ready(titanSlam) {
		t.Errorf("at full health: stage %d, the slam ready %v; want stage 0 without it", c.SkillStage(), ready(titanSlam))
	}
	c.HP = int(TitanStage2*float64(c.Kind.MaxHP())) - 1
	if c.SkillStage() != 1 || !ready(titanSlam) {
		t.Errorf("under 60%%: stage %d, the slam ready %v", c.SkillStage(), ready(titanSlam))
	}
	calm := Skill(titanStrike).Windup
	c.HP = int(TitanStage3*float64(c.Kind.MaxHP())) - 1
	w.startSkill(c, titanStrike, p)
	if c.sk.wind >= calm || c.sk.wind < rageWindup {
		t.Errorf("enraged it winds up the strike in %d ticks, calm %d; want quicker, never under %d", c.sk.wind, calm, rageWindup)
	}
	w.cancelSkill(c)
	stepN(w, BossRoarTicks+3) // it roars in its rage first
	if !w.flooded(l) {
		t.Fatalf("under 25%% the courtyard does not flood")
	}
	stepN(w, FloodRise+2)
	if !p.Wading() || p.statusSpeed() >= 1 {
		t.Errorf("in the flooded courtyard the player is not slowed: wading %v, speed %.2f", p.Wading(), p.statusSpeed())
	}
	// The fight over, the flood drains.
	defeat(t, w, p, CreatureTitanoboa)
	if w.flooded(l) {
		t.Errorf("the flood stays after the snake has fallen")
	}
}

func TestFallenColumnsShieldFromTheSweepAndTheCoils(t *testing.T) {
	w, p, l := templeScene(t)
	c := l.c
	var col *Entity
	for _, e := range w.entities {
		if e.Kind == EntityLairProp && e.Var == LairColumnM {
			col = e
			break
		}
	}
	if col == nil {
		t.Fatal("no fallen column")
	}
	// The snake to the north of the column, the player tight behind it.
	cx, cy := float64(col.TX)+0.5, float64(col.TY)+0.5
	for _, id := range []SkillID{titanTail, titanCoil} {
		c.X, c.Y = cx, cy-1.6
		if !w.creatureFits(c.Kind, c.X, c.Y) {
			c.X, c.Y = cx, cy-2.2
		}
		p.X, p.Y = cx, cy+0.9
		w.cancelSkill(c)
		l.awake, l.roared, l.foe = true, true, p.ID
		hp := p.Health
		w.ForceSkill(c.ID, id, p.ID)
		stepN(w, c.sk.wind+2)
		if p.Health < hp || p.PinnedBy() != 0 {
			t.Errorf("%s: behind a column the player lost %.0f health (pinned %v)", Skill(id).Label, hp-p.Health, p.PinnedBy() != 0)
		}
	}
}

func TestTheReefBreaksUnderThePlayerAndSettles(t *testing.T) {
	w, p, l := atollScene(t)
	p.X, p.Y = reefEdge(t, w, l)
	tx, ty := int(p.X), int(p.Y)
	l.c.X, l.c.Y = w.lagoonPoint(l, p.X, p.Y, 3)
	l.awake, l.roared, l.foe = true, true, p.ID
	w.ForceSkill(l.c.ID, sharkRam, p.ID)
	stepN(w, Skill(sharkRam).Windup+2)
	if w.TileAt(tx, ty) != TileShallow || !w.ReefBroken(tx, ty) {
		t.Fatalf("the ram did not break the reef under the player: tile %d", w.TileAt(tx, ty))
	}
	if !w.Swimming(p) {
		t.Errorf("the player still stands on the broken reef")
	}
	if w.Tiles()[ty*w.width+tx] != TileSand {
		t.Errorf("a player joining now is told the broken reef is water for good")
	}
	l.awake = false // let it be: this is about the reef
	stepN(w, ReefDownTicks)
	if w.TileAt(tx, ty) != TileSand || w.ReefBroken(tx, ty) {
		t.Errorf("after %d ticks the reef has not settled back", ReefDownTicks)
	}
}

func TestNoSandbarIsSafeFromTheShark(t *testing.T) {
	w, p, l := atollScene(t)
	// On the sandbar, as far as can be from the water, out of its reach.
	var bx, by float64
	best := -1.0
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			if w.TileAt(tx, ty) != TileSand || l.reefTile(ty*w.width+tx) || math.Hypot(float64(tx)+0.5-l.x, float64(ty)+0.5-l.y) > l.site.r {
				continue
			}
			if e := w.entityAt(tx, ty); e != nil {
				continue
			}
			wx, wy := w.lagoonPoint(l, float64(tx)+0.5, float64(ty)+0.5, 0)
			if d := math.Hypot(float64(tx)+0.5-wx, float64(ty)+0.5-wy); d > best {
				best, bx, by = d, float64(tx)+0.5, float64(ty)+0.5
			}
		}
	}
	if best < 0 {
		t.Fatal("the atoll has no sandbar")
	}
	p.X, p.Y = bx, by
	p.Health = 1e9
	stepN(w, 40*TicksPerSecond)
	if !l.awake || !l.arena {
		t.Errorf("40 seconds on a sandbar: the shark awake %v, the arena shut %v; want it still hunting", l.awake, l.arena)
	}
}

func TestThrownSpearsStrikeTheShark(t *testing.T) {
	w, p, l := atollScene(t)
	p.X, p.Y = reefEdge(t, w, l)
	c := l.c
	c.X, c.Y = w.lagoonPoint(l, p.X, p.Y, 2.5)
	c.State = CreatureLurk
	w.dev.calm = true // it circles where it is put: this is about the spear
	inHand(p, ItemSpear, 1)
	hp := c.HP
	w.SetInput(p.ID, Input{Aim: true, AimX: float32(c.X), AimY: float32(c.Y), Draw: true})
	w.Step()
	w.Apply(p.ID, Command{Kind: CmdThrow})
	if p.Held() == ItemSpear {
		t.Fatalf("the spear was not thrown")
	}
	got := 0
	for range 2 * TicksPerSecond {
		w.Step()
		if c.HP < hp && got == 0 {
			got = hp - c.HP // before it heals, asleep at home
		}
	}
	want := float64(SpearDamage) * ThrowShare
	if float64(got) < want*0.75 {
		t.Errorf("a thrown spear took %d from the shark, want about %.0f", got, want)
	}
	if float64(got) < float64(SpearDamage)*2 {
		t.Errorf("a throw (%d) is not much better than a thrust (%d)", got, SpearDamage)
	}
	found := false
	for _, d := range w.drops {
		if !d.gone && d.Item == ItemSpear && w.walkableAt(d.X, d.Y) && math.Hypot(d.X-p.X, d.Y-p.Y) < 12 {
			found = true
		}
	}
	if !found {
		t.Errorf("the spear did not wash up on the reef to be picked up again")
	}
	if len(w.spears) != 0 {
		t.Errorf("%d spears still in the air", len(w.spears))
	}
}

func TestTheBeachedSharkCanBeStruck(t *testing.T) {
	w, p, l := atollScene(t)
	p.X, p.Y = reefEdge(t, w, l)
	c := l.c
	c.X, c.Y = w.lagoonPoint(l, p.X, p.Y, 6)
	l.awake, l.roared, l.foe = true, true, p.ID
	p.Health = 1e9
	inHand(p, ItemDiamondSword, 1)
	w.ForceSkill(c.ID, sharkLunge, p.ID)
	hp := c.HP
	beached := false
	for range Skill(sharkLunge).Windup + Skill(sharkLunge).Active + Skill(sharkLunge).Recover {
		w.SetInput(p.ID, Input{Action: true})
		w.Step()
		if _, ph, _, _ := c.SkillNow(); ph == PhaseRecover && !w.creatureFits(c.Kind, c.X, c.Y) {
			beached = true
		}
	}
	if !beached {
		t.Errorf("the shark never lay beached after its lunge")
	}
	if c.HP >= hp {
		t.Errorf("the beached shark could not be struck")
	}
	if !w.creatureFits(c.Kind, c.X, c.Y) {
		stepN(w, 2)
		if !w.creatureFits(c.Kind, c.X, c.Y) {
			t.Errorf("the shark was left on the reef")
		}
	}
}

func TestLairFightsAreDeterministic(t *testing.T) {
	run := func() (int, int, float64, float64) {
		w, p, l := templeScene(t)
		p.X, p.Y = spotNear(t, w, l.c.X+3, l.c.Y)
		p.Health = 1e9
		for range 40 * TicksPerSecond {
			w.SetInput(p.ID, Input{Action: true})
			w.Step()
		}
		w2, p2, m := atollScene(t)
		p2.X, p2.Y = reefEdge(t, w2, m)
		p2.Health = 1e9
		stepN(w2, 40*TicksPerSecond)
		return l.c.HP, m.c.HP, m.c.X, l.c.Y
	}
	a1, b1, c1, d1 := run()
	a2, b2, c2, d2 := run()
	if a1 != a2 || b1 != b2 || c1 != c2 || d1 != d2 {
		t.Errorf("two runs of the same fights differ: %d %d %.3f %.3f and %d %d %.3f %.3f", a1, b1, c1, d1, a2, b2, c2, d2)
	}
}

func TestLeavingASaveMidFightResetsIt(t *testing.T) {
	w := lairsWorld(t)
	m := &w.bosses.lairs[5]
	p := w.players[w.AddPlayer("Thử")]
	p.Hunger = MaxHunger
	p.X, p.Y = reefEdge(t, w, m)
	p.Health = 1e9
	m.c.X, m.c.Y = w.lagoonPoint(m, p.X, p.Y, 3)
	stepN(w, 3*TicksPerSecond) // it wakes
	if !m.awake {
		t.Fatalf("the shark did not wake for a player on its reef")
	}
	w.ForceSkill(m.c.ID, sharkRam, p.ID)
	stepN(w, Skill(sharkRam).Windup+2)
	m.c.HP -= 2000
	broken := 0
	for _, h := range w.hz.list {
		if h.Kind == HazardReef {
			broken++
		}
	}
	if broken == 0 {
		t.Fatalf("the ram broke no reef")
	}
	w2, err := Load(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	m2 := &w2.bosses.lairs[5]
	if m2.c == nil || m2.awake || m2.arena || m2.c.HP != m2.c.Kind.MaxHP() {
		t.Errorf("loaded mid-fight: the shark awake %v, the arena shut %v; want it asleep and whole at its lair", m2.awake, m2.arena)
	}
	if len(w2.hz.list) != 0 {
		t.Errorf("loaded: %d hazards left over from the fight", len(w2.hz.list))
	}
	for _, i := range m2.site.reef {
		if w2.tiles[i] != TileSand {
			t.Fatalf("loaded: a foothold of reef is still broken")
		}
	}
}

func TestBossMovesAreReadableAndOpenAfter(t *testing.T) {
	for _, id := range append(TitanSkills(), SharkSkills()...) {
		d := Skill(id)
		lo, hi := 16, 20 // heavy: 0.8-1 s
		if d.Class == SkillSpecial {
			lo, hi = 24, 30 // special: 1.2-1.5 s
		}
		if d.Windup < lo || d.Windup > hi {
			t.Errorf("%s: wind-up %d ticks, want %d to %d", d.Label, d.Windup, lo, hi)
		}
		if d.Windup*3/4 < rageWindup {
			t.Errorf("%s: enraged its wind-up would be under %d ticks", d.Label, rageWindup)
		}
		if d.Recover < 3*TicksPerSecond/4 {
			t.Errorf("%s: open only %d ticks after it, want at least three quarters of a second", d.Label, d.Recover)
		}
		if d.Damage > 0 && d.Shape == ZoneNone {
			t.Errorf("%s: it wounds, but has no zone", d.Label)
		}
	}
}
