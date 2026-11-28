package main

import (
	"flag"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Fire and kills, for checking their art (internal/client/fireart.go,
// corpseart.go):
//
//	shot -fires -held 23 -time 0.85      every fire round the player: a campfire burning well,
//	                                     one burning low, a cold hearth under its pot, a pot on
//	                                     a fire, a village bonfire, a torch on its stand, a lit
//	                                     furnace and an oven; a torch in hand with -held
//	shot -corpses -zoom -1               a body of every beast that leaves one, in rows by size
//	shot -firebloodsheet dir             the fire and body sheets (see firebloodsheet.go)
var (
	firesFlag          = flag.Bool("fires", false, "put every kind of fire round the player: campfires burning well and low, a cold hearth, a pot on a fire, a bonfire, a torch stand, a furnace and an oven")
	corpsesFlag        = flag.Bool("corpses", false, "put the body of every beast that leaves one round the player, in rows by size")
	firebloodSheetFlag = flag.String("firebloodsheet", "", "write the fire and body sheets into this folder instead of a frame: each fire frame by frame and as a GIF, every body, the blood spreading (see internal/client/firebloodsheet.go)")
)

// firebloodScene changes snap as -fires and -corpses ask; (sx, sy) is where
// the player stands.
func firebloodScene(snap *protocol.Snapshot, sx, sy float64) {
	if !*firesFlag && !*corpsesFlag {
		return
	}
	cx, cy := int16(math.Floor(sx)), int16(math.Floor(sy))
	kept := snap.Entities[:0]
	for _, e := range snap.Entities {
		if e.TX < cx-13 || e.TX > cx+13 || e.TY < cy-9 || e.TY > cy+9 {
			kept = append(kept, e)
		}
	}
	snap.Entities = kept
	kc := snap.Creatures[:0]
	for _, c := range snap.Creatures {
		if math.Abs(float64(c.X)-sx) > 14 || math.Abs(float64(c.Y)-sy) > 10 {
			kc = append(kc, c)
		}
	}
	snap.Creatures = kc
	id := sim.EntityID(1<<27 + 900)
	put := func(k sim.EntityKind, dx, dy int16, v uint8, fuel int32, big bool) {
		id++
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: k, TX: cx + dx, TY: cy + dy, Ready: true, Var: v, Fuel: fuel, Big: big})
	}
	if *firesFlag {
		put(sim.EntityCampfire, -5, -2, 0, sim.FireStartFuel, false)
		put(sim.EntityCampfire, -2, -2, 0, sim.FireStartFuel/5, false)
		put(sim.EntityCampfire, 2, -2, uint8(sim.GearPot), 0, false)
		put(sim.EntityCampfire, 5, -2, uint8(sim.GearPot), sim.FireStartFuel, false)
		put(sim.EntityBonfire, -5, 2, 0, 0, false)
		put(sim.EntityTorchStand, -2, 2, 0, sim.TorchStandTicks, false)
		put(sim.EntityFurnace, 2, 2, 0, 100, false)
		put(sim.EntityOven, 5, 2, 0, sim.OvenMaxFuel/2, false)
	}
	if *corpsesFlag {
		// Rows of bodies by size, the biggest at the back.
		rows := [][]sim.CreatureKind{
			{sim.CreatureBrachio, sim.CreatureRex, sim.CreatureMammoth, sim.CreatureToro},
			{sim.CreatureTrike, sim.CreatureStego, sim.CreatureAnkylo, sim.CreatureQuetzal, sim.CreatureThesc},
			{sim.CreatureCroc, sim.CreatureBrachy, sim.CreatureChampso, sim.CreatureBear, sim.CreatureSabre, sim.CreaturePtero},
			{sim.CreatureRaptor, sim.CreatureAchero, sim.CreatureOrnitho, sim.CreatureAnzu, sim.CreatureTroodon, sim.CreatureTribesman, sim.CreatureScorpion},
			{sim.CreatureRabbit, sim.CreatureSnake, sim.CreatureCrab, sim.CreatureSpider, sim.CreatureMonkey},
		}
		gaps := []int16{6, 5, 4, 3, 3}
		y := int16(-8)
		for r, row := range rows {
			x := -int16(len(row)-1) * gaps[r] / 2
			for i, k := range row {
				put(sim.EntityCorpse, x+int16(i)*gaps[r], y, uint8(k), 0, i%2 == 1)
			}
			y += []int16{5, 4, 3, 3, 3}[r]
		}
	}
}
