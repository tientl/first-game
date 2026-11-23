package sim

// Fighting beasts. Every creature has its own health, and every blow takes
// off as much as the weapon in hand does, give or take a little; now and
// then a blow lands just right and does half as much again. So a rabbit
// falls to one punch, a raptor to a few spear thrusts, and a tyrant takes a
// long fight with a good sword. Tools are weapons too, weaker than a sword
// of the same stone or metal.

// Combat tuning. Health and damage count in the same points.
const (
	HealthScale  = 10  // a beast's health is its old count of hits times this
	FistDamage   = 10  // a punch
	DamageSpread = 0.2 // a blow does this share more or less, at random
	CritChance   = 0.1 // a blow is a critical one this often...
	CritBonus    = 1.6 // ...and does this much more
)

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
	if t, ok := SwordTier(held); ok {
		return swordDamage[t]
	}
	switch held {
	case ItemSpear:
		return SpearDamage
	case ItemIvorySpear:
		return IvorySpearDamage
	}
	if class, tier := held.ToolKind(); class != NoTool {
		return toolDamage[class][tier]
	}
	return FistDamage
}

// damageWith is what one of p's blows does, before the spread.
func damageWith(p *Player) int { return WeaponDamage(p.Held()) }

// blow rolls the damage of one of p's blows, and tells p when it is a
// critical one.
func (w *World) blow(p *Player) int {
	r := w.random()
	d := float64(damageWith(p)) * (1 + (r.Float64()*2-1)*DamageSpread)
	if r.Float64() < CritChance {
		d *= CritBonus
		p.notify(NoticeCrit, 0)
	}
	return max(1, int(d+0.5))
}
