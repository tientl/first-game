package main

import (
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// demoBosses leaves only the two bosses of the demo in addBosses.
var demoBosses bool

// addBosses puts the six bosses round the player standing on a beach, the
// sea below: the land ones on the sand and grass, the Megalodon and a
// lurking Spinosaurus in the water. pose shifts which state each shows, so
// a few shots show every pose. The Giganotosaurus fights the player: its
// bar is up, and the map marks every lair.
func addBosses(snap *protocol.Snapshot, sx, sy float64, pose int) {
	states := []sim.CreatureState{sim.CreatureIdle, sim.CreatureChase, sim.CreatureStrike, sim.CreatureSpecial, sim.CreatureRoar}
	put := func(i int, k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
		if demoBosses && k != sim.CreatureTitanoboa && k != sim.CreatureMegalodon {
			return
		}
		hp := int16(min(k.MaxHP(), 32767))
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: sim.CreatureID(1<<23 + i), Kind: k, X: float32(sx + dx), Y: float32(sy + dy),
			State: st, FaceLeft: left, HP: hp * 2 / 3, MaxHP: hp,
		})
	}
	st := func(i int) sim.CreatureState { return states[(i+pose)%len(states)] }
	put(0, sim.CreatureTitanoboa, -7, -3.5, st(0), false)
	put(2, sim.CreatureMammothKing, 0.5, -5.5, st(2), true)
	put(3, sim.CreatureCarcharo, 7.5, -3, st(3), true)
	put(4, sim.CreatureGiganoto, -6.5, 1.2, st(4), false)
	// In the sea, which starts 2.5 tiles below the player.
	put(1, sim.CreatureSpino, -3, 4.2, sim.CreatureLurk, false)
	megaSt := []sim.CreatureState{sim.CreatureChase, sim.CreatureSpecial, sim.CreatureLurk, sim.CreatureStrike}[pose%4]
	put(5, sim.CreatureMegalodon, 4.5, 5, megaSt, true)
	if pose%2 == 1 { // and a Spinosaurus bursting out
		put(6, sim.CreatureSpino, 1.5, 3.2, sim.CreatureSpecial, true)
	}
	snap.Boss = protocol.BossView{ID: 1<<23 + 4, Kind: sim.CreatureGiganoto, HP: 6200, MaxHP: int32(sim.CreatureGiganoto.MaxHP()), Enraged: true}
	if demoBosses {
		hp := int32(sim.CreatureTitanoboa.MaxHP())
		snap.Boss = protocol.BossView{ID: 1 << 23, Kind: sim.CreatureTitanoboa, HP: hp * 2 / 5, MaxHP: hp, Enraged: pose%2 == 1}
	}
	for i := range sim.NumBosses {
		snap.BossMarks = append(snap.BossMarks, protocol.BossMark{Kind: sim.BossKind(i),
			X: float32(sx) + float32(i%3-1)*30, Y: float32(sy) + float32(i/3*2-1)*25, Defeated: i%2 == 0})
	}
}

// addVictory shows the game-beaten screen.
func addVictory(snap *protocol.Snapshot) {
	snap.Victory = protocol.VictoryView{Show: true, Day: 37}
	for i := range sim.NumBosses {
		snap.Victory.Beaten = append(snap.Victory.Beaten, sim.BossKind(i))
	}
}
