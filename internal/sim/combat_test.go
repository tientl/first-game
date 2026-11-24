package sim

import (
	"fmt"
	"math"
	"testing"
)

type duelResult struct {
	won     bool
	secs    float64
	hpLeft  float64
	blows   int
	beastHP int
}

func duel(kind CreatureKind, weapon ItemKind, armor Tier, seed int) duelResult {
	w := flatWorld()
	for range seed {
		w.random().Float64()
	}
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv = Inventory{}
	if weapon != ItemNone {
		p.Inv[0] = Slot{Item: weapon, Count: 1}
	}
	if armor > TierStone {
		p.Inv[9] = Slot{Item: ArmorItem(armor), Count: 1}
	}
	p.Selected = 0
	c := w.addCreature(kind, p.X+1.2, p.Y)
	c.rage, c.foe = 60*TicksPerSecond, id // angry at the player from the start
	w.SetInput(id, Input{Action: true})
	start := w.tick
	blows := 0
	lastHP := c.HP
	for w.tick-start < 90*TicksPerSecond {
		// Keep the player right by the beast, facing it, as a player chasing it would.
		dx, dy := c.X-p.X, c.Y-p.Y
		if d := math.Hypot(dx, dy); d > 0.9 {
			p.X += dx / d * (d - 0.9)
			p.Y += dy / d * (d - 0.9)
		}
		w.Step()
		if c.HP < lastHP {
			blows++
		}
		lastHP = c.HP
		if c.gone || c.HP <= 0 || w.creatureByID(c.ID) == nil {
			return duelResult{true, float64(w.tick-start) / TicksPerSecond, p.Health, blows, 0}
		}
		if p.Dead {
			return duelResult{false, float64(w.tick-start) / TicksPerSecond, 0, blows, c.HP}
		}
		p.Health = math.Min(p.Health, MaxHealth) // no regen cheat
	}
	return duelResult{false, 90, p.Health, blows, c.HP}
}

func TestCombatTable(t *testing.T) { // go test -run CombatTable -v prints the table
	if !testing.Verbose() {
		t.Skip()
	}
	kinds := []CreatureKind{CreatureSnake, CreatureRaptor, CreatureAchero, CreatureSabre, CreatureBear, CreatureSpider,
		CreatureScorpion, CreatureTribesman, CreatureCroc, CreatureAnzu, CreatureTrike, CreatureToro, CreatureStego,
		CreatureMammoth, CreatureQuetzal, CreatureRex}
	type kit struct {
		name   string
		weapon ItemKind
		armor  Tier
	}
	kits := []kit{{"tay", ItemNone, 0}, {"riu da", ItemAxe, 0}, {"giao", ItemSpear, 0}, {"kiem dong", SwordItem(TierCopper), 0},
		{"kiem dong+giap", SwordItem(TierCopper), TierCopper}, {"kiem sat+giap", SwordItem(TierIron), TierIron}, {"kiem bk+giap", SwordItem(TierPlatinum), TierPlatinum}, {"kiem kc+giap", SwordItem(TierDiamond), TierDiamond}}
	out := fmt.Sprintf("%-18s", "")
	for _, k := range kits {
		out += fmt.Sprintf("| %-22s", k.name)
	}
	out += "\n"
	for _, kind := range kinds {
		out += fmt.Sprintf("%-18s", kind.Name()+fmt.Sprintf(" %d", kind.MaxHP()))
		for _, k := range kits {
			wins, secs, hp := 0, 0.0, 0.0
			const n = 5
			for s := range n {
				r := duel(kind, k.weapon, k.armor, s)
				if r.won {
					wins++
					secs += r.secs
					hp += r.hpLeft
				}
			}
			cell := fmt.Sprintf("%d/5", wins)
			if wins > 0 {
				cell += fmt.Sprintf(" %4.1fs %3.0fhp", secs/float64(wins), hp/float64(wins))
			}
			out += fmt.Sprintf("| %-22s", cell)
		}
		out += "\n"
	}
	t.Log("\n" + out)
}

// TestCombatBalance holds the fights to their intended difficulty.
func TestCombatBalance(t *testing.T) {
	fight := func(k CreatureKind, weapon ItemKind, armor Tier) (wins int) {
		for s := range 5 {
			if duel(k, weapon, armor, s).won {
				wins++
			}
		}
		return wins
	}
	for _, c := range []struct {
		kind   CreatureKind
		weapon ItemKind
		armor  Tier
		want   bool // the player should win every time, or lose every time
	}{
		{CreatureSnake, ItemNone, 0, true},
		{CreatureRaptor, ItemSpear, 0, true},
		{CreatureMammoth, ItemNone, 0, false},               // a mammoth is no fistfight
		{CreatureTrike, ItemNone, 0, false},                 // nor is any dinosaur:
		{CreatureRaptor, ItemNone, 0, false},                // fists barely get through their hide
		{CreatureBear, ItemNone, 0, false},
		{CreatureTribesman, ItemNone, 0, true}, // a man can be fought with bare hands
		{CreatureRex, SwordItem(TierIron), TierIron, false}, // the tyrant wants the best gear
		{CreatureRex, SwordItem(TierPlatinum), TierPlatinum, true},
		{CreatureBear, SwordItem(TierCopper), 0, true},
	} {
		wins := fight(c.kind, c.weapon, c.armor)
		if c.want && wins != 5 || !c.want && wins != 0 {
			t.Errorf("%s with %s and armour tier %d: player won %d of 5", c.kind.Name(), c.weapon.Def().Name, c.armor, wins)
		}
	}
	if SwingTicks(ItemNone) >= SwingTicks(ItemAxe) || SwingTicks(SwordItem(TierCopper)) >= SwingTicks(ItemIvorySpear) {
		t.Error("a punch should be quicker than an axe, a sword than an ivory spear")
	}
}
