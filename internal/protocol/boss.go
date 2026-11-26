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
	Enraged   bool // below half its health: it attacks faster
	Holds     bool // it holds this player in its coils
}

// BossMark is a boss's lair on the world map, once the player has seen it.
type BossMark struct {
	Kind     sim.CreatureKind
	X, Y     float32 // the lair, in tiles
	Defeated bool
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
}

// VictoryView is the game-beaten screen: every boss of the world has fallen.
type VictoryView struct {
	Show   bool               // this player has not closed the screen yet (sim.CmdVictorySeen closes it)
	Day    int32              // the day the last boss fell
	Beaten []sim.CreatureKind // the bosses that fell, in order
}
