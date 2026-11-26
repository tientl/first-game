// Package sim is the authoritative game simulation: world generation, players,
// resources, crafting and the day/night cycle.
//
// It is deliberately free of rendering, input and networking so the same code
// runs inside a dedicated server, inside the single-player client, and in
// tests. All durations are counted in ticks; the simulation never reads the
// wall clock, so stepping it is deterministic for a given seed and input
// sequence.
package sim

import "time"

// Simulation rate.
const (
	TicksPerSecond = 20
	TickDuration   = time.Second / TicksPerSecond
)

// DayTicks is the length of one full day/night cycle (16 minutes): about 9
// minutes of day, 2½ of dusk and 5 of night at the equinox; summer days are
// longer and winter nights longer (see DuskAt).
const DayTicks = 960 * TicksPerSecond

// Player tuning.
const (
	PlayerSpeed  = 4.0 / TicksPerSecond // tiles per tick
	PlayerRadius = 0.25                 // half-size of the collision box, in tiles

	MaxHealth   = 100.0
	MaxHunger   = 100.0
	SpawnHunger = 75.0

	HungerPerTick       = MaxHunger / (600 * TicksPerSecond) // full to empty in 10 minutes, a day and a quarter
	StarveDamagePerTick = 1.0 / TicksPerSecond
	ColdDamagePerTick   = 2.0 / TicksPerSecond
	RegenPerTick        = 0.25 / TicksPerSecond
	WellFedHunger       = 60.0 // regenerate health above this hunger
	ColdAmbient         = 0.1  // ambient light below which it is freezing

	InteractReach       = 1.5 // tiles from player centre to target tile centre
	ActionCooldownTicks = TicksPerSecond / 2
	ActingTicks         = 6 // how long a player shows the "working" pose

	MaxStack = 40 // per inventory slot
)

// Resource and structure tuning. Trees and boulders come small or big: a big
// one takes more hits and gives more, a little more per hit.
const (
	TreeHits       = 2
	BigTreeHits    = 5
	BoulderHits    = 3
	BigBoulderHits = 6

	TreeYield       = 3  // since save version 20 the woods are spaced by the trees' crowns (woods.go):
	BigTreeYield    = 10 // a fifth as many trees, each giving more; and a big one grows back slowly (regrow.go)
	BoulderYield    = 2
	BigBoulderYield = 6
	BushYield       = 2
	TuftYield       = 1
	TwigYield       = 1
	PebbleYield     = 1
	CropYield       = 3 // sweet potatoes dug from a ripe crop; one was planted

	TreeRegrowTicks   = 2 * DayTicks
	BushRegrowTicks   = 3 * DayTicks / 2
	TuftRegrowTicks   = DayTicks
	TwigRegrowTicks   = DayTicks
	PebbleRegrowTicks = 2 * DayTicks

	// Farming.
	CropGrowTicks      = DayTicks     // planted to ripe
	FarmlandDecayTicks = 3 * DayTicks // tilled soil left empty turns back to grass
	DigWormChance      = 0.25         // tilling turns up a worm...
	DigPotatoChance    = 0.1          // ...or else, sometimes, a sweet potato

	FireStartFuel   = 90 * TicksPerSecond
	FireMaxFuel     = 180 * TicksPerSecond
	FireFuelPerWood = 30 * TicksPerSecond
)
