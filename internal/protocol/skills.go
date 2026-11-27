package protocol

import "survivalgame/internal/sim"

// ZoneView is a red warning on the ground: where a beast's special move will
// land, and how soon (sim/skills.go). The client draws it under everything
// standing (telegraphzone.go).
type ZoneView struct {
	Creature     sim.CreatureID
	Kind         sim.CreatureKind
	Skill        sim.SkillID
	Shape        sim.ZoneShape
	X, Y         float32 // a cone's or a line's start, a circle's centre, in tiles
	Angle        float32 // the aim, radians (0 east, y down)
	Reach, Width float32 // a cone's length and half-angle (radians), a circle's radius, a line's length and half-width (tiles)
	Left, Total  uint8   // ticks till it lands, and the whole wind-up
}
