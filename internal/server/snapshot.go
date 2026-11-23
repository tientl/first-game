package server

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// snapshotFor builds the view of the world for player id: its own state plus
// the players and entities within the interest radius. players is the
// current player list (shared across all snapshots of one tick). Every
// returned slice is freshly allocated, because the snapshot is handed to
// another goroutine.
func (s *Server) snapshotFor(id sim.PlayerID, players []sim.Player) (protocol.Snapshot, bool) {
	var self *sim.Player
	for i := range players {
		if players[i].ID == id {
			self = &players[i]
			break
		}
	}
	if self == nil {
		return protocol.Snapshot{}, false
	}

	tick := s.world.Tick()
	snap := protocol.Snapshot{
		Tick: tick,
		Self: protocol.SelfState{
			Health:    float32(self.Health),
			Hunger:    float32(self.Hunger),
			Inv:       self.Inv,
			Dead:      self.Dead,
			Cold:      self.Cold,
			Notice:    self.Notice,
			NoticeArg: self.NoticeArg,
			NoticeSeq: self.NoticeSeq,
		},
	}

	r := float64(s.cfg.InterestRadius)
	for i := range players {
		p := &players[i]
		if math.Abs(p.X-self.X) > r || math.Abs(p.Y-self.Y) > r {
			continue
		}
		snap.Players = append(snap.Players, protocol.PlayerView{
			ID:     p.ID,
			Name:   p.Name,
			X:      float32(p.X),
			Y:      float32(p.Y),
			Dir:    p.Dir,
			Dead:   p.Dead,
			Acting: p.Acting(tick),
		})
	}

	cx, cy := int(math.Floor(self.X)), int(math.Floor(self.Y))
	ri := s.cfg.InterestRadius
	s.world.EachEntityIn(cx-ri, cy-ri, cx+ri, cy+ri, func(e sim.Entity) {
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID:    e.ID,
			Kind:  e.Kind,
			TX:    int16(e.TX), // world size is capped at sim.MaxWorldSize
			TY:    int16(e.TY),
			Ready: e.Ready,
			Fuel:  int32(e.Fuel),
		})
	})
	return snap, true
}
