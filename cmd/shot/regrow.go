package main

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// addRegrowStrip puts the stages a felled tree grows back through (see
// internal/sim/regrow.go) in a row across the view below the player, on
// tiles the client draws a leafy tree on: a big tree's stump, its sapling,
// the young tree and the grown one. Whatever stood round the row, or in
// front of it, is cleared from the view.
func addRegrowStrip(snap *protocol.Snapshot, w *sim.World, sx, sy float64) {
	// The row a little below the player (not hiding them) where four such
	// tiles lie closest together.
	var y int
	var xs []int
	for dy := 6; dy <= 8; dy++ {
		ty := int(math.Floor(sy)) + dy
		var row []int
		for x := int(math.Floor(sx)) - 6; len(row) < 4 && x < int(sx)+40; x++ {
			if sim.TreeLookAt(x, ty, w.TileAt(x, ty)) == sim.LookBroadleaf && (len(row) == 0 || x-row[len(row)-1] >= 3) {
				row = append(row, x)
			}
		}
		if len(row) == 4 && (xs == nil || row[3]-row[0] < xs[3]-xs[0]) {
			y, xs = ty, row
		}
	}
	if xs == nil {
		return
	}
	keep := snap.Entities[:0]
	for _, e := range snap.Entities {
		if int(e.TX) >= xs[0]-2 && int(e.TX) <= xs[3]+3 && int(e.TY) >= y-1 && int(e.TY) <= y+9 {
			continue
		}
		keep = append(keep, e)
	}
	snap.Entities = keep
	stages := []protocol.EntityView{
		{Kind: sim.EntityTree, Big: true},                   // the stump
		{Kind: sim.EntitySapling, Var: uint8(sim.SeedTree)}, // a day on
		{Kind: sim.EntityTree, Ready: true},                 // two and a half days on
		{Kind: sim.EntityTree, Ready: true, Big: true},      // four and a half
	}
	for i, s := range stages {
		s.ID = sim.EntityID(1<<29 + 900 + i)
		s.TX, s.TY = int16(xs[i]), int16(y)
		snap.Entities = append(snap.Entities, s)
	}
}
