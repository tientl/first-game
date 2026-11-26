package main

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// addFlora sets the new plants and the toads round the player, for -flora:
//
//   - pond: lilies and lotus on the water in view, pitcher plants, horsetail
//     and wild rice on its banks, toads on the leaves and the banks (use
//     with -open -at pond);
//   - coast: mangroves along the shore and seaweed off it (use with -coast);
//   - land: every plant of the dry land in rows, full and picked, the crops
//     stage by stage, and three toads.
func addFlora(snap *protocol.Snapshot, w *sim.World, sx, sy float64, mode string) {
	cx, cy := int(math.Floor(sx)), int(math.Floor(sy))
	taken := map[[2]int]bool{}
	for _, e := range snap.Entities {
		taken[[2]int{int(e.TX), int(e.TY)}] = true
	}
	id := sim.EntityID(1 << 26)
	flora := func(f sim.FloraKind, tx, ty int, ready bool) {
		id++
		taken[[2]int{tx, ty}] = true
		snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: sim.EntityFlora, TX: int16(tx), TY: int16(ty), Ready: ready, Var: uint8(f)})
	}
	cid := sim.CreatureID(1 << 28)
	toad := func(k sim.CreatureKind, x, y float64, st sim.CreatureState, left bool) {
		cid++
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{ID: cid, Kind: k, X: float32(x), Y: float32(y), State: st, FaceLeft: left, HP: 1, MaxHP: 1})
	}
	water := func(tx, ty int) bool { return w.TileAt(tx, ty) == sim.TileWater }
	byLand := func(tx, ty int) bool {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if w.TileAt(tx+dx, ty+dy).Walkable() {
					return true
				}
			}
		}
		return false
	}
	bank := func(tx, ty int) bool {
		if !w.TileAt(tx, ty).Walkable() {
			return false
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if water(tx+dx, ty+dy) {
					return true
				}
			}
		}
		return false
	}
	h := func(tx, ty int, salt uint32) float64 {
		return float64(sim.HashTile(tx*7+int(salt), ty*13+int(salt)*3)%1000) / 1000
	}
	// drop takes out what worldgen put where the scene goes.
	drop := func(gone func(e protocol.EntityView) bool) {
		kept := snap.Entities[:0]
		for _, e := range snap.Entities {
			if gone(e) {
				delete(taken, [2]int{int(e.TX), int(e.TY)})
				continue
			}
			kept = append(kept, e)
		}
		snap.Entities = kept
	}
	noToads := func() {
		kept := snap.Creatures[:0]
		for _, c := range snap.Creatures {
			if !sim.IsToad(c.Kind) {
				kept = append(kept, c)
			}
		}
		snap.Creatures = kept
	}
	switch mode {
	case "pond":
		drop(func(e protocol.EntityView) bool { return e.Kind == sim.EntityFlora })
		noToads()
		n := 0
		for ty := cy - 8; ty <= cy+8; ty++ {
			for tx := cx - 14; tx <= cx+14; tx++ {
				if taken[[2]int{tx, ty}] {
					continue
				}
				r := h(tx, ty, 1)
				switch {
				case water(tx, ty) && r < 0.3:
					flora(sim.FloraLily, tx, ty, r > 0.05)
					if n++; n%3 != 2 {
						k := sim.CreatureToad
						if n%2 == 0 {
							k = sim.CreatureGlowToad
						}
						toad(k, float64(tx)+0.5, float64(ty)+0.5, sim.CreatureIdle, n%4 < 2)
					}
				case water(tx, ty) && r < 0.42:
					flora(sim.FloraLotus, tx, ty, r > 0.33)
				case bank(tx, ty) && r < 0.22:
					flora(sim.FloraPitcher, tx, ty, r > 0.04)
				case bank(tx, ty) && r < 0.44:
					flora(sim.FloraHorsetail, tx, ty, true)
				case bank(tx, ty) && r < 0.6:
					flora(sim.FloraWildRice, tx, ty, r > 0.47)
				case bank(tx, ty) && r > 0.86:
					k := sim.CreatureGlowToad
					if r > 0.93 {
						k = sim.CreatureToad
					}
					st := sim.CreatureIdle
					if r > 0.96 {
						st = sim.CreatureWander // caught in the air
					}
					toad(k, float64(tx)+0.3+0.4*h(tx, ty, 2), float64(ty)+0.3+0.4*h(tx, ty, 3), st, h(tx, ty, 4) < 0.5)
				case w.TileAt(tx, ty) == sim.TileMarsh && r > 0.8 && r < 0.83:
					flora(sim.FloraPoisonMushroom, tx, ty, true)
				}
			}
		}
	case "coast":
		drop(func(e protocol.EntityView) bool { return e.Kind == sim.EntityFlora })
		for ty := cy - 6; ty <= cy+9; ty++ {
			for tx := cx - 14; tx <= cx+14; tx++ {
				if taken[[2]int{tx, ty}] || !water(tx, ty) {
					continue
				}
				near := byLand(tx, ty)
				r := h(tx, ty, 5)
				switch {
				case near && tx < cx-3 && r < 0.6, near && tx > cx+7 && r < 0.4:
					flora(sim.FloraMangrove, tx, ty, r > 0.06)
				case near && r < 0.3, !near && ty <= cy+6 && r < 0.2:
					flora(sim.FloraSeaweed, tx, ty, r > 0.02)
				}
			}
		}
	case "land":
		drop(func(e protocol.EntityView) bool {
			return int(e.TX) >= cx-11 && int(e.TX) <= cx+12 && int(e.TY) >= cy-6 && int(e.TY) <= cy+5
		})
		noToads()
		row := []sim.FloraKind{sim.FloraBamboo, sim.FloraGourd, sim.FloraPitcher, sim.FloraFlax, sim.FloraVine, sim.FloraPoisonMushroom,
			sim.FloraNightshade, sim.FloraAloe, sim.FloraHerb, sim.FloraMoss, sim.FloraHorsetail}
		for i, f := range row {
			flora(f, cx-10+2*i, cy-4, true)
			flora(f, cx-10+2*i, cy-2, false)
		}
		for i, f := range []sim.FloraKind{sim.FloraWildRice, sim.FloraWildCorn, sim.FloraWildSquash} {
			flora(f, cx-10+2*i, cy+1, true)
			flora(f, cx-10+2*i, cy+3, false)
		}
		for c := sim.CropKind(0); c < sim.NumCrops; c++ {
			for s := 0; s <= sim.CropStages; s++ {
				id++
				snap.Entities = append(snap.Entities, protocol.EntityView{ID: id, Kind: sim.EntityCrop,
					TX: int16(cx - 2 + s + 6*int(c%2)), TY: int16(cy + 2 + 2*int(c/2)),
					Var: uint8(c), Stage: uint8(min(s, sim.CropStages-1)), Ready: s == sim.CropStages})
			}
		}
		toad(sim.CreatureToad, sx+2.5, sy-0.3, sim.CreatureIdle, true)
		toad(sim.CreatureGlowToad, sx+4, sy-0.2, sim.CreatureIdle, false)
		toad(sim.CreatureToad, sx+5.5, sy-0.3, sim.CreatureStrike, true)
	}
}
