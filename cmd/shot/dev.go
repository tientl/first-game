package main

import (
	"log"
	"math"

	"survivalgame/internal/client"
	"survivalgame/internal/sim"
)

// devShotLines are the commands the -devconsole shot shows, run for real
// on the world so the answers are the game's own.
var devShotLines = []string{"help", "give gỗ 20", "give \"rìu đá lửa\"", "spawn trike 3", "tp làng", "weather storm", "list beasts khung long", "god"}

// showDevConsole opens the dev console on game with devShotLines and what
// the world answered.
func showDevConsole(game *client.Game, w *sim.World) {
	id := w.AddPlayer("Dev")
	var answers [][]string
	for _, l := range devShotLines {
		answers = append(answers, w.DevCommand(id, l).Lines)
	}
	game.ShowDevConsole(devShotLines, answers)
}

// devBasePlan is the -devbase shot's base, piece by piece, in tiles from
// the player: a wooden house with a door to the south, planks inside, a bed,
// a chest and a mortar, torches at the door; a kitchen of fires to the west
// behind a row of stakes; tilled soil and seedlings to the east; flowers and
// heads on posts in front.
var devBasePlan = func() []devBasePart {
	var out []devBasePart
	add := func(k sim.EntityKind, v uint8, dx, dy int) { out = append(out, devBasePart{k, v, dx, dy}) }
	for dx := -3; dx <= 3; dx++ {
		add(sim.EntityWall, 0, dx, -6)
		if dx != 0 {
			add(sim.EntityWall, 0, dx, -2)
		}
	}
	for dy := -5; dy <= -3; dy++ {
		add(sim.EntityWall, 0, -3, dy)
		add(sim.EntityWall, 0, 3, dy)
	}
	add(sim.EntityDoor, 0, 0, -2)
	add(sim.EntityBed, 0, -2, -5)
	add(sim.EntityChest, 0, 0, -5)
	add(sim.EntityTorchStand, 0, 2, -5)
	add(sim.EntityMortar, 0, 2, -3)
	for dy := -5; dy <= -3; dy++ {
		for dx := -2; dx <= 2; dx++ {
			add(sim.EntityFloor, 0, dx, dy) // where nothing else stands
		}
	}
	add(sim.EntityTorchStand, 0, -1, -1)
	add(sim.EntityTorchStand, 0, 1, -1)
	// The kitchen, west.
	add(sim.EntityCampfire, 0, -6, -4)
	add(sim.EntityCampfire, uint8(sim.GearPot), -6, -4) // onto that fire
	add(sim.EntityCampfire, uint8(sim.GearGrill), -6, -1)
	add(sim.EntityDryRack, 0, -8, -4)
	add(sim.EntityFurnace, 0, -8, -1)
	for dy := -5; dy <= 1; dy++ {
		add(sim.EntitySpikes, 0, -10, dy)
	}
	// The farm, east.
	for dx := 5; dx <= 7; dx++ {
		for dy := -1; dy <= 1; dy++ {
			add(sim.EntityFarmland, 0, dx, dy)
		}
	}
	add(sim.EntitySapling, 1, 5, 2)
	add(sim.EntitySapling, 2, 6, 2)
	add(sim.EntitySapling, 0, 7, 2)
	// The front.
	add(sim.EntityFlower, 0, -3, -1)
	add(sim.EntityFlower, 1, -2, -1)
	add(sim.EntityFlower, 2, 2, -1)
	add(sim.EntityFlower, 5, 3, -1)
	add(sim.EntityTrophy, uint8(sim.CreatureRex), -5, 2)
	add(sim.EntityTrophy, uint8(sim.CreatureTrike), 3, 2)
	return out
}()

type devBasePart struct {
	kind   sim.EntityKind
	v      uint8
	dx, dy int
}

// buildDevBase builds devBasePlan round the player at (sx, sy) as free build
// does, through the world's own commands: a builder in dev mode clears the
// ground (Shift and the right button), then puts down each piece from the
// build palette, standing a few tiles south of it.
func buildDevBase(w *sim.World, sx, sy float64) {
	w.SetDevMode(true)
	id := w.AddPlayer("Builder")
	if !w.DevSetFreeBuild(id, true) {
		log.Fatal("free build would not turn on")
	}
	cx, cy := int(math.Floor(sx)), int(math.Floor(sy))
	at := func(tx, ty int) { w.DevTeleport(id, float64(tx)+0.5, float64(ty)+3.5) }
	for ty := cy - 7; ty <= cy+6; ty++ {
		for tx := cx - 11; tx <= cx+10; tx++ {
			if e, ok := w.EntityAt(tx, ty); ok && sim.DevRemovable(e.Kind) {
				at(tx, ty)
				w.Apply(id, sim.Command{Kind: sim.CmdDevRemove, At: true, TX: int16(tx), TY: int16(ty)})
			}
		}
	}
	pieces := sim.DevPieces()
	type kv struct {
		k sim.EntityKind
		v uint8
	}
	byKind := map[kv]int{}
	for i, pc := range pieces {
		byKind[kv{pc.Kind, pc.Var}] = i
	}
	placed, skipped := 0, 0
	for _, part := range devBasePlan {
		i, ok := byKind[kv{part.kind, part.v}]
		if !ok {
			log.Fatalf("no piece %v %d in the build palette", part.kind, part.v)
		}
		tx, ty := cx+part.dx, cy+part.dy
		if e, ok := w.EntityAt(tx, ty); ok && !pieces[i].OnFire && e.Kind != sim.EntityNone {
			skipped++
			continue // already built on: the planks go round the furniture
		}
		at(tx, ty)
		w.Apply(id, sim.Command{Kind: sim.CmdDevBuild, Slot: uint8(i), At: true, TX: int16(tx), TY: int16(ty)})
		if e, ok := w.EntityAt(tx, ty); ok && e.Kind == pieces[i].Kind {
			placed++
		}
	}
	w.RemovePlayer(id)
	log.Printf("free build: %d of %d pieces of the base put down", placed, len(devBasePlan)-skipped)
}
