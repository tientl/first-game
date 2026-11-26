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
			Enraged: b.Enraged, Holds: b.Holds == id,
		}
	}
	for _, b := range s.world.BossesSeenBy(id) {
		snap.BossMarks = append(snap.BossMarks, protocol.BossMark{
			Kind: b.Kind, X: float32(b.LairX), Y: float32(b.LairY), Defeated: b.Defeated,
		})
	}
	for _, l := range s.world.LairsNear(id, lairViewRange) {
		snap.Lairs = append(snap.Lairs, protocol.LairView{
			Kind: l.Kind, X: float32(l.X), Y: float32(l.Y), Radius: float32(l.Radius),
			Closed: l.Closed, Awake: l.Awake, Defeated: l.Defeated,
		})
	}
	if show, day, beaten := s.world.Victory(id); show {
		snap.Victory = protocol.VictoryView{Show: true, Day: int32(day), Beaten: beaten}
	}
}
