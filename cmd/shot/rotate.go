package main

import (
	"flag"
	"fmt"
	"log"

	"survivalgame/internal/client"
	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// What turns (internal/sim/facing.go, internal/client/facing.go), for
// checking its art:
//
//	shot -exact -w 1920 -h 1080 -rotsheet -o sheet.png
//	    every piece that turns, each way, sleepers each way, and walls and
//	    fences joined (see internal/client/rotsheet.go)
//	shot -exact -w 1920 -h 1080 -rotbase -o base.png
//	    a small base built through free build's commands with things facing
//	    every way: walls joined round corners, doors both ways, a bed, a
//	    tent, a furnace and an oven turned
//	shot -exact -w 1920 -h 1080 -freebuild -devpiece 6 -aim 1,3 -rotgif -gif 90 -o ghost.png
//	    a palette piece in hand, turned with R three times, then put down
//	shot -exact -w 1920 -h 1080 -rotbase -rotbaseat 0,5 -build 9 -aim -1,-2 -turn 3 -frames 160 -o bed.png
//	    standing inside the base, a bed's ghost turned to face east, with
//	    the hint to turn it over the hotbar
var (
	rotSheetFlag = flag.Bool("rotsheet", false, "write the sheet of what turns instead of a frame: every piece each way it faces, sleepers, and walls and fences joined (see internal/client/rotsheet.go)")
	rotBaseFlag  = flag.Bool("rotbase", false, "build a small base round the player through free build's own commands with things facing every way: joined walls, doors both ways, a bed, a tent, a furnace, an oven (see rotate.go)")
	rotGIFFlag   = flag.Bool("rotgif", false, "with -devpiece and -aim: turn the piece in hand with R every half second, three times, then put it down where its ghost is (use with -gif)")
	turnFlag     = flag.Int("turn", 0, "turn what is being placed (-build, -devpiece, or a tent, a head or a grill held) this many times, as R does, before the capture")
)

// rotBasePlan is the -rotbase shot's base, in tiles from the player: a
// hut of palisade with a stone wall on its west side, a door south and a
// door east, a bed lying north–south, a chest, a furnace facing west; out
// in front a tent facing east, an oven turned east, a grill turned
// north–south over a fire, a fence of stakes round a corner and a head on
// a post looking left.
var rotBasePlan = func() []devBasePart {
	var out []devBasePart
	add := func(k sim.EntityKind, v uint8, dx, dy int, d sim.Dir) {
		out = append(out, devBasePart{kind: k, v: v, dx: dx, dy: dy, face: d})
	}
	for dx := -4; dx <= 4; dx++ {
		add(sim.EntityWall, 0, dx, -8, sim.DirDown)
		if dx != 0 {
			add(sim.EntityWall, 0, dx, -3, sim.DirDown)
		}
	}
	add(sim.EntityDoor, 0, 0, -3, sim.DirDown)
	for dy := -7; dy <= -4; dy++ {
		add(sim.EntityStoneWall, 0, -4, dy, sim.DirRight)
		if dy != -5 {
			add(sim.EntityWall, 0, 4, dy, sim.DirRight)
		}
	}
	add(sim.EntityDoor, 0, 4, -5, sim.DirRight)
	add(sim.EntityBed, 0, -3, -6, sim.DirLeft)
	add(sim.EntityChest, 0, 1, -7, sim.DirDown)
	add(sim.EntityFurnace, 0, 2, -5, sim.DirLeft)
	add(sim.EntityTorchStand, 0, -1, -2, sim.DirDown)
	add(sim.EntityTorchStand, 0, 1, -2, sim.DirDown)
	// Out in front.
	add(sim.EntityLeafTent, 0, -6, -1, sim.DirRight)
	add(sim.EntityOven, 0, 6, -1, sim.DirRight)
	add(sim.EntityCampfire, uint8(sim.GearGrill), 3, 1, sim.DirRight)
	add(sim.EntityTrophy, uint8(sim.CreatureRex), -3, 1, sim.DirLeft)
	add(sim.EntityMortar, 0, 7, 2, sim.DirLeft)
	for dx := -9; dx <= -7; dx++ {
		add(sim.EntitySpikes, 0, dx, 3, sim.DirDown)
	}
	for dy := 0; dy <= 2; dy++ {
		add(sim.EntitySpikes, 0, -9, dy, sim.DirRight)
	}
	return out
}()

// rotScript turns the piece in hand with R every half second, three times,
// then puts it down where its ghost stands: the -rotgif shot. The fake
// server does not build, so it sends the snapshot with the piece in it.
type rotScript struct {
	conn   *fakeConn
	snap   protocol.Snapshot
	game   *client.Game
	tx, ty int16
	start  int // the frame the GIF starts at
	done   bool
}

const rotEvery = 30 // frames between turns

func (r *rotScript) step(frame int) {
	t := frame - r.start
	if t <= 0 || r.done || t%rotEvery != 0 {
		return
	}
	if t/rotEvery <= 3 { // south to west, north and east
		r.game.TurnPlacing(1)
		return
	}
	k, face, ok := r.game.PlacingFacing()
	if !ok {
		log.Print("rotgif: nothing that turns is in hand")
		r.done = true
		return
	}
	s := r.snap
	s.Entities = append(append([]protocol.EntityView(nil), s.Entities...), protocol.EntityView{
		ID: sim.EntityID(1 << 28), Kind: k, TX: r.tx, TY: r.ty, Ready: true, Face: face,
	})
	s.Tick++
	r.snap, r.done = s, true
	r.game.StopBuilding()
	select {
	case r.conn.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: s}:
	default:
	}
}

var rotBaseAt = flag.String("rotbaseat", "", "dx,dy: build the -rotbase base this many tiles from the player (5,5 puts them inside it, by its south door)")

// rotBaseShifted is rotBasePlan moved as -rotbaseat says.
func rotBaseShifted() []devBasePart {
	var dx, dy int
	fmt.Sscanf(*rotBaseAt, "%d,%d", &dx, &dy)
	out := make([]devBasePart, len(rotBasePlan))
	for i, p := range rotBasePlan {
		p.dx, p.dy = p.dx+dx, p.dy+dy
		out[i] = p
	}
	return out
}
