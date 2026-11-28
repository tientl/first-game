package server

import (
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// skillViews adds to snap what the client needs to draw the beasts' moves
// (sim/skills.go): the move each creature in view is making, and the red
// warnings on the ground within r tiles of (x, y).
func (s *Server) skillViews(snap *protocol.Snapshot, x, y, r float64) {
	byID := make(map[sim.CreatureID]int, len(snap.Creatures))
	for i, c := range snap.Creatures {
		byID[c.ID] = i
	}
	s.world.EachCreatureIn(x-r, y-r, x+r, y+r, func(c sim.Creature) {
		id, ph, t, n := c.SkillNow()
		if id == sim.SkillNone {
			return
		}
		if i, ok := byID[c.ID]; ok {
			v := &snap.Creatures[i]
			v.Skill, v.Phase, v.PhaseT, v.PhaseLen = id, ph, uint8(min(t, 255)), uint8(min(n, 255))
		}
	})
	for _, h := range s.world.Hazards(x-r, y-r, x+r, y+r) {
		snap.Hazards = append(snap.Hazards, protocol.HazardView{Kind: h.Kind, X: float32(h.X), Y: float32(h.Y), R: float32(h.R),
			Left: uint16(max(0, min(h.Left, 65535))), Total: uint16(max(0, min(h.Total, 65535)))})
	}
	for _, z := range s.world.Zones(x-r, y-r, x+r, y+r) {
		snap.Zones = append(snap.Zones, protocol.ZoneView{
			Creature: z.Creature, Kind: z.Kind, Skill: z.Skill, Shape: z.Shape,
			X: float32(z.X), Y: float32(z.Y), Angle: float32(z.Angle),
			Reach: float32(z.Reach), Width: float32(z.Width),
			Left: uint8(min(z.Left, 255)), Total: uint8(min(z.Total, 255)),
		})
	}
}
