package sim

import (
	"fmt"
	"math"
	"testing"
)

// skillScene sets a beast of move id's kind by a player standing where the
// move strikes, and has it begin the move at them.
func skillScene(t *testing.T, id SkillID) (*World, *Creature, *Player) {
	t.Helper()
	d := Skill(id)
	w := flatWorld()
	px, py := 16.5, 16.5
	switch d.Kind {
	case CreaturePlesio:
		w = shoreWorld() // the sea from column 18
		px = 17.4
		if d.Need == NeedWet {
			px = 20.5 // out swimming
		}
	}
	w.spawnX, w.spawnY = px, py
	pid := w.AddPlayer("a")
	p := w.players[pid]
	p.X, p.Y = px, py
	dist := min(max(d.MinD+0.3, 1.1), d.MaxD)
	switch {
	case d.Kind == CreaturePlesio && d.Need == NeedWet:
		dist = 3
	case d.Kind == CreaturePlesio:
		dist = 2.1
	case d.Motion == MoveLeap:
		dist = (d.MinD + d.MaxD) / 2
	case d.Anchor == AtAhead || d.Anchor == AtBehind:
		dist = d.Offset
	}
	c := w.addCreature(d.Kind, px+dist, py)
	if c.Kind == CreaturePlesio {
		c.State, c.timer = CreatureChase, PlesioSurfaceTicks
	}
	c.FaceLeft = true // looking at the player, its tail away
	if d.Fx&FxSteal != 0 {
		inHand(p, ItemBerry, 3)
	}
	if !w.ForceSkill(c.ID, id, pid) {
		t.Fatalf("%s: could not begin it", d.Label)
	}
	return w, c, p
}

// attacks lists every move that wounds.
func attacks() []SkillID {
	var out []SkillID
	for id := SkillID(1); id < NumSkills; id++ {
		if d := Skill(id); d.Damage > 0 && d.Shape != ZoneNone {
			out = append(out, id)
		}
	}
	return out
}

func skillName(id SkillID) string {
	d := Skill(id)
	return fmt.Sprintf("%s %s", d.Kind.Name(), d.Name)
}

func TestEveryMoveLandsOnlyAfterItsWindup(t *testing.T) {
	for _, id := range attacks() {
		t.Run(skillName(id), func(t *testing.T) {
			w, c, p := skillScene(t, id)
			d := Skill(id)
			_, _, _, wind := c.SkillNow()
			for i := 0; i < wind-1; i++ {
				w.Step()
				if p.Health < MaxHealth {
					t.Fatalf("hit %d ticks into a wind-up of %d", i+1, wind)
				}
				if c.sk.phase != PhaseWindup {
					t.Fatalf("phase %d after %d ticks of a wind-up of %d", c.sk.phase, i+1, wind)
				}
			}
			for range d.Active + 2 {
				w.Step()
			}
			if p.Health >= MaxHealth {
				t.Fatalf("a player standing still in its way was never hit (health %.0f)", p.Health)
			}
		})
	}
}

func TestSteppingOutOfTheZoneDodgesTheMove(t *testing.T) {
	for _, id := range attacks() {
		t.Run(skillName(id), func(t *testing.T) {
			w, c, p := skillScene(t, id)
			d := Skill(id)
			_, _, _, wind := c.SkillNow()
			// It follows its foe till it locks its aim; then a step aside.
			lock := max(d.Lock, 1)
			if d.Lock < 0 {
				lock = wind - 1
			}
			for range wind - lock {
				w.Step()
			}
			p.Y -= 5 // well aside of anything it aims
			for n := 0; c.sk.id == id && n < 10*TicksPerSecond; n++ {
				w.Step()
			}
			if p.Health < MaxHealth {
				t.Fatalf("hit for %.0f though out of its zone when it landed", MaxHealth-p.Health)
			}
		})
	}
}

func TestTimingsFollowTheirClass(t *testing.T) {
	for id := SkillID(1); id < NumSkills; id++ {
		d := Skill(id)
		var lo, hi int
		switch d.Class {
		case SkillLight:
			lo, hi = 7, 12 // 0.35-0.6 s
		case SkillHeavy:
			lo, hi = 12, 20 // 0.6-1 s
		case SkillSpecial:
			lo, hi = 24, 30 // 1.2-1.5 s
		default:
			continue
		}
		if d.Windup < lo || d.Windup > hi {
			t.Errorf("%s: wind-up %d ticks, want %d to %d", skillName(id), d.Windup, lo, hi)
		}
		if d.Recover < TicksPerSecond/2-4 || d.Recover > TicksPerSecond+4 {
			t.Errorf("%s: recovery %d ticks, want about half a second to a second", skillName(id), d.Recover)
		}
		if d.Zone != (d.Class == SkillSpecial) && id != SkillRexRoar {
			t.Errorf("%s: zone %v: a special move, and only a special one, shows its zone", skillName(id), d.Zone)
		}
	}
	for k := CreatureKind(1); k < NumCreatureKinds; k++ {
		if n := len(SkillsOf(k)); n > 0 && k != CreatureAchero && k != CreatureThesc && k != CreatureOrnitho && n < 2 {
			t.Errorf("%s has only %d move", k.Name(), n)
		}
	}
}

func TestAfterAMoveItIsOpenThenCoolsDown(t *testing.T) {
	w, c, p := skillScene(t, SkillRexBite)
	d := Skill(SkillRexBite)
	for c.sk.phase != PhaseRecover {
		w.Step()
	}
	for i := 0; i < d.Recover-1; i++ {
		w.Step()
		if c.sk.id != SkillRexBite || c.sk.phase != PhaseRecover {
			t.Fatalf("%d ticks into its recovery it is at move %d phase %d", i+1, c.sk.id, c.sk.phase)
		}
	}
	// Hit while it recovers: it takes the blow, and it goes on recovering.
	hp := c.HP
	w.wound(c, 30, p)
	if c.HP >= hp {
		t.Fatal("a blow in its recovery did nothing")
	}
	i := skillSlot(CreatureRex, SkillRexBite)
	if c.sk.cd[i] == 0 || w.ready(c, i, d, p, 1.5) {
		t.Fatal("the bite is ready again at once")
	}
	for c.sk.cd[i] > 0 {
		w.stepSkills(c)
	}
	c.sk.id = SkillNone
	if !w.ready(c, i, d, p, 1.5) {
		t.Fatal("the bite never cooled down")
	}
}

func TestAPackTakesTurns(t *testing.T) {
	for _, k := range []CreatureKind{CreatureRaptor, CreatureAchero} {
		w := flatWorld()
		pid := w.AddPlayer("a")
		p := w.players[pid]
		w.players[pid].dev.god = true // a long fight, to watch the pack
		for i := range 5 {
			a := float64(i) * 2 * math.Pi / 5
			c := w.addCreature(k, p.X+2.5*math.Cos(a), p.Y+2.5*math.Sin(a))
			c.foe, c.rage = pid, HerdRageTicks
		}
		most := 0
		for range 15 * TicksPerSecond {
			w.Step()
			n := 0
			for _, c := range w.fauna.creatures {
				if c.Kind == k && c.sk.id != SkillNone && Skill(c.sk.id).Damage > 0 && c.sk.phase != PhaseRecover {
					n++
				}
			}
			most = max(most, n)
		}
		if most == 0 || most > packLimit(k) {
			t.Errorf("%s: at most %d striking at once, want 1 to %d", k.Name(), most, packLimit(k))
		}
	}
}

func TestAHeavyBlowStaggersARaptorButNotTheTyrant(t *testing.T) {
	w, c, p := skillScene(t, SkillDakotaSickle)
	w.wound(c, StaggerDamage-5, p)
	if c.sk.id != SkillDakotaSickle {
		t.Fatal("a light blow knocked the raptor out of its wind-up")
	}
	w.wound(c, StaggerDamage, p)
	if c.sk.id != SkillNone || c.State != CreatureHurt {
		t.Fatalf("move %d state %d: a heavy blow should stagger it", c.sk.id, c.State)
	}
	w, c, p = skillScene(t, SkillRexBite)
	w.wound(c, 80, p)
	if c.sk.id != SkillRexBite {
		t.Fatal("the tyrant was knocked out of its bite")
	}
	// Any blow cuts short a Troodon's cry for the pack.
	w, c, p = skillScene(t, SkillTroodonCall)
	w.wound(c, 1, p)
	if c.sk.id != SkillNone {
		t.Fatal("a blow did not cut short the cry")
	}
}

func TestStunHoldsThePlayerThenCannotChain(t *testing.T) {
	w := flatWorld()
	pid := w.AddPlayer("a")
	p := w.players[pid]
	x := p.X
	p.stunFor(TicksPerSecond)
	w.SetInput(pid, Input{MoveX: 1})
	for range TicksPerSecond - 1 {
		w.Step()
		if p.X != x {
			t.Fatal("moved while stunned")
		}
	}
	if !p.Stunned() {
		t.Fatal("the stun wore off early")
	}
	w.Step()
	p.stunFor(TicksPerSecond)
	if p.Stunned() {
		t.Fatal("stunned again straight after a stun")
	}
	stepN(w, 5)
	if p.X <= x {
		t.Fatal("did not walk on once the stun wore off")
	}
}

func TestBleedingAndSlowing(t *testing.T) {
	w := flatWorld()
	pid := w.AddPlayer("a")
	p := w.players[pid]
	p.Inv[9] = Slot{Item: ArmorItem(TierDiamond), Count: 1} // no armour stops bleeding
	p.bleedFor(2 * TicksPerSecond)
	stepN(w, 3*TicksPerSecond)
	if lost := MaxHealth - p.Health; math.Abs(lost-2*BleedPerTick*TicksPerSecond) > 1 {
		t.Fatalf("lost %.1f bleeding for two seconds, want %.1f", lost, 2*BleedPerTick*TicksPerSecond)
	}
	x := p.X
	w.SetInput(pid, Input{MoveX: 1})
	stepN(w, TicksPerSecond)
	full := p.X - x
	p.slowFor(2 * TicksPerSecond)
	x = p.X
	stepN(w, TicksPerSecond)
	if slow := p.X - x; math.Abs(slow-full*SlowShare) > 0.05 {
		t.Fatalf("slowed, walked %.2f tiles a second, want %.2f", slow, full*SlowShare)
	}
}

func TestAPounceHoldsThePlayerDownTillTheyStruggleFree(t *testing.T) {
	w, c, p := skillScene(t, SkillDakotaPounce)
	for range c.sk.wind + Skill(SkillDakotaPounce).Active {
		w.Step()
	}
	if p.PinnedBy() != c.ID || c.sk.pin != p.ID {
		t.Fatalf("pinned by %d: the pounce should hold the player down", p.PinnedBy())
	}
	x, hp := p.X, p.Health
	w.SetInput(p.ID, Input{MoveX: -1})
	stepN(w, PinBiteEvery*2)
	if p.X != x || p.Health >= hp {
		t.Fatalf("moved %.2f, health %.0f → %.0f: held down, the player is bitten and cannot move", p.X-x, hp, p.Health)
	}
	for range PinMash {
		w.SetInput(p.ID, Input{Action: true})
		w.Step()
		w.SetInput(p.ID, Input{})
		w.Step()
	}
	if p.PinnedBy() != 0 || c.State != CreatureHurt || c.sk.id != SkillNone {
		t.Fatalf("pinned by %d, raptor state %d: struggling should throw it off, reeling", p.PinnedBy(), c.State)
	}
	// Left alone, it lets go by itself.
	w, c, p = skillScene(t, SkillDakotaPounce)
	stepN(w, c.sk.wind+Skill(SkillDakotaPounce).Active+PinHold+2)
	if p.PinnedBy() != 0 || p.Dead {
		t.Fatalf("still pinned (%d) or dead (%v) after %d ticks", p.PinnedBy(), p.Dead, PinHold)
	}
}

func TestSpecialMovesShowTheirZone(t *testing.T) {
	w, c, _ := skillScene(t, SkillBrachioStomp)
	zs := w.Zones(0, 0, 32, 32)
	if len(zs) != 1 || zs[0].Creature != c.ID || zs[0].Shape != ZoneCircle || zs[0].Left != c.sk.wind {
		t.Fatalf("zones %+v: want the stomp's circle, all its wind-up to go", zs)
	}
	w.Step()
	if zs = w.Zones(0, 0, 32, 32); len(zs) != 1 || zs[0].Left != c.sk.wind-1 {
		t.Fatalf("zones %+v: the time left should count down", zs)
	}
	w, _, _ = skillScene(t, SkillRexBite)
	if zs = w.Zones(0, 0, 32, 32); len(zs) != 0 {
		t.Fatalf("zones %+v: a heavy bite has no zone, only its posture", zs)
	}
}

func TestATorosaurusWarnsFirst(t *testing.T) {
	for _, stay := range []bool{false, true} {
		w := flatWorld()
		pid := w.AddPlayer("a")
		p := w.players[pid]
		c := w.addCreature(CreatureToro, p.X+3, p.Y)
		c.FaceLeft = true
		w.Step()
		if c.sk.id != SkillToroDisplay {
			t.Fatalf("move %d: a player three tiles off should be warned off", c.sk.id)
		}
		if !stay {
			p.X -= 4 // backing off
		}
		stepN(w, 4*TicksPerSecond)
		hurt := p.Health < MaxHealth
		if hurt != stay {
			t.Errorf("stayed %v, hurt %v: it charges only a player who did not back off", stay, hurt)
		}
	}
}

func TestAHerdBeastCallsItsHerdWhenHit(t *testing.T) {
	w := flatWorld()
	pid := w.AddPlayer("a")
	p := w.players[pid]
	a := w.addCreature(CreatureTrike, p.X+2, p.Y)
	b := w.addCreature(CreatureTrike, p.X+2+HerdAlarmDist+1, p.Y) // beyond the herd's alarm, within its call
	w.wound(a, 10, p)
	if b.rage > 0 {
		t.Fatal("a herd mate this far off heard the blow itself")
	}
	stepN(w, CreatureHurtTicks+2)
	if a.sk.id != SkillTrikeCall {
		t.Fatalf("move %d: the one hit should bellow for its herd", a.sk.id)
	}
	stepN(w, Skill(SkillTrikeCall).Windup+1)
	if b.rage == 0 || b.foe != pid {
		t.Fatal("the bellow did not bring the herd")
	}
}

func TestATyrantsRoarStunsAndScatters(t *testing.T) {
	w := flatWorld()
	pid := w.AddPlayer("a")
	p := w.players[pid]
	rex := w.addCreature(CreatureRex, p.X+5, p.Y)
	rabbit := w.addCreature(CreatureRabbit, p.X+3, p.Y+3)
	w.Step()
	if rex.sk.id != SkillRexRoar {
		t.Fatalf("move %d: the tyrant should roar on seeing a player", rex.sk.id)
	}
	stepN(w, rex.sk.wind)
	if !p.Stunned() {
		t.Fatal("a player five tiles off was not stunned by the roar")
	}
	if rabbit.hunter != rex.ID {
		t.Fatal("a rabbit near by did not run from the roar")
	}
}

func TestSkillsAreDeterministic(t *testing.T) {
	run := func() string {
		w := flatWorld()
		pid := w.AddPlayer("a")
		p := w.players[pid]
		for i, k := range []CreatureKind{CreatureRaptor, CreatureRaptor, CreatureRaptor, CreatureTrike, CreatureSabre} {
			c := w.addCreature(k, p.X+3+float64(i), p.Y+float64(i%3)-1)
			c.foe, c.rage = pid, HerdRageTicks
		}
		w.SetInput(pid, Input{Action: true})
		out := ""
		for i := range 20 * TicksPerSecond {
			w.Step()
			if i%20 == 0 {
				out += fmt.Sprintf("%.3f %d|", p.Health, p.status.stun)
				for _, c := range w.fauna.creatures {
					out += fmt.Sprintf("%d %.3f %.3f %d %d;", c.ID, c.X, c.Y, c.sk.id, c.sk.phase)
				}
			}
		}
		return out
	}
	if a, b := run(), run(); a != b {
		t.Fatal("two runs of the same fight went differently")
	}
}
