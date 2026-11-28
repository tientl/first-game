package protocol

import "survivalgame/internal/sim"

// What a player is told of the six bosses (see sim/bosses.go).

// BossView is the boss a player is fighting: awake and close enough that a
// big health bar shows at the top of their screen. Kind is CreatureNone when
// there is none.
type BossView struct {
	ID        sim.CreatureID
	Kind      sim.CreatureKind
	HP, MaxHP int32
	Enraged   bool  // below half its health: it attacks faster
	Holds     bool  // it holds this player in its coils
	Stage     uint8 // the stage of its fight (sim.Creature.SkillStage): its moves and its music change
}

// BossMark is a boss's lair on the world map, once the player has seen it.
type BossMark struct {
	Kind     sim.CreatureKind
	X, Y     float32 // the lair, in tiles
	Defeated bool
	Camp     bool    // since sim.StyleLairs: the lair has a camp...
	CX, CY   float32 // ...here
}

// LairView is a boss's lair near the player (see sim/lair.go): what the
// client needs to draw its arena, hide its sleeping boss and set the mood.
type LairView struct {
	Kind     sim.CreatureKind
	X, Y     float32 // the lair, in tiles
	Radius   float32 // of its arena; 0 if it has none
	Closed   bool    // the arena is shut
	Awake    bool    // its boss is in a fight
	Defeated bool

	// Since sim.StyleLairs (sim/lairsite.go): how it is laid out, the
	// temple's black pool, the atoll's lagoon, the landmark (the dead
	// tree's foot, the mast), the camp, and how its fight stands.
	Layout         sim.LairLayout
	PoolX, PoolY   float32
	PoolRX, PoolRY float32
	Lagoon         float32 // the lagoon's radius
	MarkX, MarkY   float32
	Camp           bool
	CampX, CampY   float32
	Stage          uint8 // the stage of its boss's fight; 0 asleep
	Flooded        bool  // the temple's courtyard is under water
	RX, RY         float32 // the bounds the camera is kept in through its fight (sim.ArenaQ's shape): half widths, tiles
}

// HazardView is something a boss's fight left on the ground for a while
// (sim/lairsite.go): a puddle of venom, the flooded courtyard, a broken
// foothold of reef, the shark's splash, its blood in the lagoon.
type HazardView struct {
	Kind    sim.HazardKind
	X, Y, R float32 // tiles
	Age     uint16  // ticks since it was made
	Left    uint16  // ticks till it ends
}

// SpearView is a spear in the air (sim/throw.go).
type SpearView struct {
	ID     sim.SpearID
	Item   sim.ItemKind
	X, Y   float32 // tiles
	VX, VY float32 // tiles a tick
	By     sim.PlayerID
}

// VictoryView is the game-beaten screen: every boss of the world has fallen.
type VictoryView struct {
	Show   bool               // this player has not closed the screen yet (sim.CmdVictorySeen closes it)
	Day    int32              // the day the last boss fell
	Beaten []sim.CreatureKind // the bosses that fell, in order
}
