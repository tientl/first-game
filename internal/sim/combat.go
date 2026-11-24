package sim

// Fighting beasts. Every creature has its own health, and every blow takes
// off as much as the weapon in hand does, give or take a little; now and
// then a blow lands just right and does half as much again. So a rabbit
// falls to one punch, a raptor to a few spear thrusts, and a tyrant takes a
// long fight with a good sword. Tools are weapons too, weaker than a sword
// of the same stone or metal. A bare fist does little through a dinosaur's
// scales and next to nothing to a great beast: see FistShare.

// Combat tuning. Health and damage count in the same points.
const (
	HealthScale  = 10  // a beast's health is its old count of hits times this
	FistDamage   = 10  // a punch
	DamageSpread = 0.2 // a blow does this share more or less, at random
	CritChance   = 0.1 // a blow is a critical one this often...
	CritBonus    = 1.6 // ...and does this much more
)

// The battle axe hits harder than any knife but the obsidian one, and swings
// slower than any other weapon: one big blow, then a breath.
const (
	BattleAxeDamage = 70
	BattleAxeSwing  = 9 * TicksPerSecond / 10
)

// FlinchEvery is how often, at most, a blow makes a small beast flinch: a
// flurry of blows does not keep it from fighting back.
const FlinchEvery = TicksPerSecond

// SwingTicks is how long a blow at a beast with held takes before the next:
// a fist is quick, a sword quicker than a heavy axe or a long ivory spear.
func SwingTicks(held ItemKind) int {
	if _, swing, ok := legendaryBlow(held); ok {
		return swing // see legendary.go
	}
	if _, ok := SwordTier(held); ok {
		return TicksPerSecond / 2
	}
	switch held {
	case ItemSpear:
		return 3 * TicksPerSecond / 5
	case ItemIvorySpear:
		return 7 * TicksPerSecond / 10
	case ItemBattleAxe:
		return BattleAxeSwing
	}
	if class, _ := held.ToolKind(); class != NoTool {
		return 7 * TicksPerSecond / 10
	}
	return 2 * TicksPerSecond / 5 // a punch
}

// toolDamage is a blow with a tool of each tier, by class.
var toolDamage = [NumToolClasses][NumTiers]int{
	ToolAxe:     {14, 17, 21, 26, 32},
	ToolPickaxe: {12, 15, 19, 23, 28},
	ToolHoe:     {9, 11, 13, 16, 19},
}

// MaxHP is the full health of a creature of kind k.
func (k CreatureKind) MaxHP() int { return k.def().hp * HealthScale }

// WeaponDamage is what one blow does, before the spread, with held in hand.
func WeaponDamage(held ItemKind) int {
	if dmg, _, ok := legendaryBlow(held); ok {
		return dmg // see legendary.go
	}
	if t, ok := SwordTier(held); ok {
		return swordDamage[t]
	}
	switch held {
	case ItemSpear:
		return SpearDamage
	case ItemIvorySpear:
		return IvorySpearDamage
	case ItemBattleAxe:
		return BattleAxeDamage
	}
	if class, tier := held.ToolKind(); class != NoTool {
		return toolDamage[class][tier]
	}
	return FistDamage
}

// damageWith is what one of p's blows does, before the spread.
func damageWith(p *Player) int { return WeaponDamage(p.Held()) }

// Punches is whether a blow with held in hand is a punch: no sword, spear
// or tool in it.
func Punches(held ItemKind) bool { return WeaponDamage(held) == FistDamage }

// FistShare is the share of a punch that gets through the hide of a
// creature of kind k: all of it on small soft things and people, next to
// nothing through the scales, plates and thick fur of every dinosaur,
// reptile and great beast.
func FistShare(k CreatureKind) float64 {
	switch k {
	case CreatureRabbit, CreatureSnake, CreatureWorm, CreatureFish, CreatureCrab, CreatureSpider,
		CreatureMonkey, CreatureTribesman:
		return 1
	}
	return 0.1
}

// BlowDamage is what one blow with held does to a creature of kind k,
// before the spread.
func BlowDamage(held ItemKind, k CreatureKind) int {
	d := WeaponDamage(held)
	if Punches(held) {
		d = max(1, int(float64(d)*FistShare(k)+0.5))
	}
	return d
}

// blow rolls the damage of one of p's blows at a creature of kind k, and
// tells p when it is a critical one.
func (w *World) blow(p *Player, k CreatureKind) int {
	r := w.random()
	d := float64(BlowDamage(p.Held(), k)) * (1 + (r.Float64()*2-1)*DamageSpread)
	if r.Float64() < CritChance {
		d *= CritBonus
		p.notify(NoticeCrit, 0)
	}
	return max(1, int(d+0.5))
}
