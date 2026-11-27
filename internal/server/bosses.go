package server

import (
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// lairViewRange is how far from a lair a player is told of it.
const lairViewRange = 56

// bossViews fills in what player id is told of the bosses: the one they
// fight, the lairs they have seen, and the victory once all have fallen.
func (s *Server) bossViews(snap *protocol.Snapshot, id sim.PlayerID) {
	if b, ok := s.world.BossFight(id); ok {
		snap.Boss = protocol.BossView{
			ID: b.ID, Kind: b.Kind, HP: int32(b.HP), MaxHP: int32(b.MaxHP),
			Enraged: b.Enraged, Holds: b.Holds == id, Stage: uint8(b.Stage),
		}
	}
	for _, b := range s.world.BossesSeenBy(id) {
		snap.BossMarks = append(snap.BossMarks, protocol.BossMark{
			Kind: b.Kind, X: float32(b.LairX), Y: float32(b.LairY), Defeated: b.Defeated,
			Camp: b.Camp, CX: float32(b.CampX), CY: float32(b.CampY),
		})
	}
	for _, l := range s.world.LairsNear(id, lairViewRange) {
		snap.Lairs = append(snap.Lairs, protocol.LairView{
			Kind: l.Kind, X: float32(l.X), Y: float32(l.Y), Radius: float32(l.Radius),
			Closed: l.Closed, Awake: l.Awake, Defeated: l.Defeated,
			Layout: l.Layout, PoolX: float32(l.PoolX), PoolY: float32(l.PoolY),
			PoolRX: float32(l.PoolRX), PoolRY: float32(l.PoolRY), Lagoon: float32(l.Lagoon),
			MarkX: float32(l.MarkX), MarkY: float32(l.MarkY),
			Camp: l.Camp, CampX: float32(l.CampX), CampY: float32(l.CampY),
			Stage: uint8(l.Stage), Flooded: l.Flooded,
		})
	}
	if show, day, beaten := s.world.Victory(id); show {
		snap.Victory = protocol.VictoryView{Show: true, Day: int32(day), Beaten: beaten}
	}
}

// hazardViews adds what the bosses' fights left on the ground within r
// tiles of (x, y), and the spears in the air.
func (s *Server) hazardViews(snap *protocol.Snapshot, x, y, r float64) {
	tick := s.world.Tick()
	for _, h := range s.world.Hazards(x-r, y-r, x+r, y+r) {
		snap.Hazards = append(snap.Hazards, protocol.HazardView{
			Kind: h.Kind, X: float32(h.X), Y: float32(h.Y), R: float32(h.R),
			Age: uint16(min(tick-h.Born, 65535)), Left: uint16(min(h.Ends-tick, 65535)),
		})
	}
	for _, sp := range s.world.Spears(x-r, y-r, x+r, y+r) {
		snap.Spears = append(snap.Spears, protocol.SpearView{
			ID: sp.ID, Item: sp.Item, X: float32(sp.X), Y: float32(sp.Y),
			VX: float32(sp.VX), VY: float32(sp.VY), By: sp.By,
		})
	}
}
