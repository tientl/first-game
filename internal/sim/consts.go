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

// DayTicks is the length of one full day/night cycle (4 minutes).
const DayTicks = 240 * TicksPerSecond

// Player tuning.
const (
	PlayerSpeed  = 4.0 / TicksPerSecond // tiles per tick
	PlayerRadius = 0.25                 // half-size of the collision box, in tiles

	MaxHealth   = 100.0
	MaxHunger   = 100.0
	SpawnHunger = 75.0

	HungerPerTick       = MaxHunger / (2.5 * DayTicks) // full to empty in 2.5 days
	StarveDamagePerTick = 1.0 / TicksPerSecond
	ColdDamagePerTick   = 2.0 / TicksPerSecond
	RegenPerTick        = 0.25 / TicksPerSecond
	WellFedHunger       = 60.0 // regenerate health above this hunger
	ColdAmbient         = 0.1  // ambient light below which it is freezing

	InteractReach       = 1.5 // tiles from player centre to target tile centre
	ActionCooldownTicks = TicksPerSecond / 2
	ActingTicks         = 6 // how long a player shows the "working" pose

	MaxStack = 40 // per item kind
)

// Resource and structure tuning.
const (
	TreeHits    = 3
	BoulderHits = 4

	TreeYield    = 2
	BoulderYield = 3
	BushYield    = 2
	TuftYield    = 1

	TreeRegrowTicks = 2 * DayTicks
	BushRegrowTicks = 3 * DayTicks / 2
	TuftRegrowTicks = DayTicks

	FireStartFuel   = 90 * TicksPerSecond
	FireMaxFuel     = 180 * TicksPerSecond
	FireFuelPerWood = 30 * TicksPerSecond
)
