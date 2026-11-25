package sim

import "testing"

func TestABlowLandsOnlyAfterTheWindUp(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	rex := w.addCreature(CreatureRex, p.X+2, p.Y)
	strike(rex, p, RexBiteDamage, RexBiteCD)
	before := p.Health
	for range WindupTicks(CreatureRex) - 1 {
		w.stepBlow(rex)
	}
	if p.Health != before {
		t.Fatalf("hit while still drawing back: %.1f → %.1f", before, p.Health)
	}
	w.stepBlow(rex)
	if p.Health >= before {
		t.Fatal("the blow never landed on a player who stood still")
	}

	// One who steps back during the wind-up is missed.
	p.Health = MaxHealth
	strike(rex, p, RexBiteDamage, RexBiteCD)
	p.X -= 1.5
	for range WindupTicks(CreatureRex) {
		w.stepBlow(rex)
	}
	if p.Health != MaxHealth {
		t.Fatalf("hit after stepping back out of reach: %.1f", p.Health)
	}

	// A beast drawn back does not flinch out of its blow.
	p.X += 1.5
	strike(rex, p, RexBiteDamage, RexBiteCD)
	w.wound(rex, 10, p)
	if rex.State != CreatureStrike || !rex.windingUp() {
		t.Fatalf("state %v: a blow knocked it out of its wind-up", rex.State)
	}
}
