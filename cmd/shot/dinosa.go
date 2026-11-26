package main

import (
	"flag"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

var dinosA = flag.Bool("dinosa", false, "put the bull-frill, the stegosaur, the brachiosaur, the little runner, the rex, an ankylosaur and a three-horn round the player (with -coast: a plesiosaur striking and one lurking under the water)")

// putDinosA adds -dinosa's beasts round (sx, sy) to snap.
func putDinosA(snap *protocol.Snapshot, sx, sy float64, coast bool) {
	if !*dinosA {
		return
	}
	id := sim.CreatureID(1<<28 + 500)
	put := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
		id++
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: id, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
		})
	}
	put(sim.CreatureBrachio, -7.5, -4.5, sim.CreatureIdle, false)
	put(sim.CreatureToro, -3.2, -3, sim.CreatureIdle, false)
	put(sim.CreatureStego, -1.2, -6.4, sim.CreatureIdle, true)
	put(sim.CreatureThesc, -1.4, -1.4, sim.CreatureIdle, true)
	put(sim.CreatureTrike, -5.8, -0.8, sim.CreatureIdle, false)
	put(sim.CreatureRex, 3.2, -4.6, sim.CreatureIdle, true)
	put(sim.CreatureAnkylo, 6.5, -1.2, sim.CreatureIdle, true)
	if coast {
		put(sim.CreaturePlesio, -2.4, 3.6, sim.CreatureIdle, true)
		put(sim.CreaturePlesio, -6, 5.4, sim.CreatureLurk, false)
		put(sim.CreaturePlesio, 0.4, 6.4, sim.CreatureLurk, true)
	}
}
