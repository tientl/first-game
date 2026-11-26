package main

import (
	"flag"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

var ankyloFlag = flag.Bool("ankylo", false, "put a small group of ankylosaurs round the player, one with its club turned on them and one grazing, with a plate-back and a three-horn for scale")

// putAnkylo adds -ankylo's beasts round (sx, sy) to snap.
func putAnkylo(snap *protocol.Snapshot, sx, sy float64) {
	if !*ankyloFlag {
		return
	}
	id := sim.CreatureID(1<<28 + 700)
	put := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
		id++
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: id, Kind: k, X: float32(sx + dx), Y: float32(sy + dy), State: st, FaceLeft: left,
		})
	}
	put(sim.CreatureAnkylo, -4.6, -2.6, sim.CreatureIdle, false)
	put(sim.CreatureAnkylo, 3.8, -1.2, sim.CreatureRoar, false) // its back to the player, the club raised
	put(sim.CreatureAnkylo, -1.4, -5.4, sim.CreatureEat, true)
	put(sim.CreatureStego, 3.4, 2.6, sim.CreatureIdle, true)
	put(sim.CreatureTrike, -7.4, 1.8, sim.CreatureIdle, false)
}
