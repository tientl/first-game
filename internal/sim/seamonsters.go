package sim

// Monsters of the open sea; see boats.go for the boats they hunt.

var (
	mosaDef = creatureDef{
		name: "Thương long", hp: 45,
		walk: 1.6 / TicksPerSecond, run: 5.4 / TicksPerSecond,
		radius: 1.0, reach: 3.2,
		yield: ItemMeat, yieldN: 12, drops: []Ingredient{{ItemBone, 5}, {ItemHide, 2}},
		leash: 30, idle: [2]int{40, 120}, roam: [2]int{40, 120},
		fights: true, tough: true, deep: true, openOnly: true, away: 40,
	}
	krakenDef = creatureDef{
		name: "Mực khổng lồ", hp: 36,
		walk: 1.0 / TicksPerSecond, run: 3.6 / TicksPerSecond,
		radius: 1.1, reach: 3.4,
		yield: ItemMeat, yieldN: 8, drops: []Ingredient{{ItemSilk, 4}},
		leash: 20, idle: [2]int{60, 200}, roam: [2]int{40, 100},
		fights: true, tough: true, deep: true, openOnly: true, away: 50,
	}
)

func init() {
	creatureDefs[CreatureMosa] = mosaDef
	creatureDefs[CreatureKraken] = krakenDef
}
