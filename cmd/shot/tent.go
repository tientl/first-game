package main

import (
	"flag"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The leaf tent (sim/tent.go, internal/client/tentart.go), for checking
// its art:
//
//	shot -tent -time 0.85              a camp: a leaf tent, a fire burning before it
//	shot -tent -tentsleep -time 0.85   the same, the other player asleep in the tent
//	shot -tent -held 127 -aim -2,1     a tent in hand, its ghost where it would stand
//	shot -tentsheet dir                every thatch close up with the icon, and a GIF
//	                                   of someone going to bed in one (see tentsheet.go)
var (
	tentFlag      = flag.Bool("tent", false, "put a leaf tent a little up and left of the player, with a burning campfire by it")
	tentSleepFlag = flag.Bool("tentsleep", false, "with -tent: the other player asleep in the tent")
	tentSheetFlag = flag.String("tentsheet", "", "write the leaf tent's sheet and sleeping GIF into this folder instead of a frame (tent_sheet.png, tent_sleep.gif)")
)

// tentScene changes snap as the tent flags ask; (sx, sy) is where the
// player stands.
func tentScene(snap *protocol.Snapshot, sx, sy float64) {
	if !*tentFlag {
		return
	}
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	tx, ty := cx-1, cy-1
	// Clear the ground the camp stands on.
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < tx-3 || e.TX > cx+4 || e.TY < ty-4 || e.TY > cy+2 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	id := sim.EntityID(1 << 26)
	snap.Entities = append(snap.Entities,
		protocol.EntityView{ID: id, Kind: sim.EntityLeafTent, TX: tx, TY: ty, Ready: true},
		protocol.EntityView{ID: id + 1, Kind: sim.EntityCampfire, TX: cx + 2, TY: cy, Ready: true, Fuel: sim.FireStartFuel})
	if *tentSleepFlag {
		p := &snap.Players[1]
		p.X, p.Y, p.Asleep, p.Held = float32(tx)+0.5, float32(ty)+0.5, true, sim.ItemNone
	}
}
