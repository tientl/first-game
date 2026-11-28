package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"survivalgame/internal/sim"
)

// scenes are the places of one world the script goes to, as tile
// coordinates; ok is false for a place the world has none of.
// place is x, y and whether the world has one (1) or not (0).
type place [3]float64

type scenes struct {
	size                    int
	spawn                   [2]float64
	forest, reef, sea, lava place
	crystal                 place
	desert, snow            place
	surfaceH                int
}

func (s place) tp() string { return fmt.Sprintf("tp %.1f %.1f", s[0], s[1]) }

// findScenes scans w for a dense forest, a reef beside a beach, open sea,
// lava and a cave of crystals. It reads the whole world, so it runs before
// the server starts (the world is not shared).
func findScenes(w *sim.World) scenes {
	N, H := w.Size()
	var sc scenes
	sc.size = N
	sx, sy := w.Spawn()
	sc.spawn = [2]float64{sx, sy}
	sc.surfaceH = N
	if w.SurfaceHeight() < H {
		sc.surfaceH = w.SurfaceHeight()
	}
	tiles := w.Tiles()
	at := func(x, y int) sim.Tile {
		if x < 0 || y < 0 || x >= N || y >= H {
			return sim.TileWater
		}
		return tiles[y*N+x]
	}
	// Trees, per 8x8 block of the surface.
	const blk = 8
	bw := N / blk
	trees := make([]int, bw*(sc.surfaceH/blk+1))
	w.EachEntityIn(0, 0, N-1, sc.surfaceH-1, func(e sim.Entity) {
		switch e.Kind {
		case sim.EntityTree, sim.EntityFigTree, sim.EntityGinkgoTree, sim.EntityPalm, sim.EntityBananaTree, sim.EntityGiantTree:
			if i := (int(e.TY)/blk)*bw + int(e.TX)/blk; i >= 0 && i < len(trees) {
				trees[i]++
			}
		}
	})
	best := 0
	for by := 1; by+4 < sc.surfaceH/blk; by++ {
		for bx := 1; bx+4 < bw; bx++ {
			n := 0 // a 4x3 block window, about the view's size
			for dy := range 3 {
				for dx := range 4 {
					n += trees[(by+dy)*bw+bx+dx]
				}
			}
			if n > best {
				best = n
				sc.forest = place{float64((bx+2)*blk) + 0.5, float64((by+1)*blk) + 4.5, 1}
			}
		}
	}
	// A reef: the most shallows within a view of a sand tile that has deep water too.
	bestReef := 0
	for y := 12; y < sc.surfaceH-12; y += 3 {
		for x := 20; x < N-20; x += 3 {
			if at(x, y) != sim.TileSand {
				continue
			}
			shallow, deep := 0, 0
			for dy := -8; dy <= 8; dy += 2 {
				for dx := -15; dx <= 15; dx += 2 {
					switch at(x+dx, y+dy) {
					case sim.TileShallow:
						shallow++
					case sim.TileWater:
						deep++
					}
				}
			}
			if deep > 20 && shallow > bestReef {
				bestReef = shallow
				sc.reef = place{float64(x) + 0.5, float64(y) + 0.5, 1}
			}
		}
	}
	// Open sea: a 19x11 block of deep water, nearest the spawn.
	bd := math.Inf(1)
	for y := 8; y < sc.surfaceH-8; y += 2 {
		for x := 10; x < N-10; x += 2 {
			if d := math.Hypot(float64(x)-sx, float64(y)-sy); d < bd && at(x, y) == sim.TileWater && at(x-9, y-5) == sim.TileWater && at(x+9, y+5) == sim.TileWater && at(x-9, y+5) == sim.TileWater && at(x+9, y-5) == sim.TileWater {
				bd = d
				sc.sea = place{float64(x) + 0.5, float64(y) + 0.5, 1}
			}
		}
	}
	// Lava: a floor tile beside the most lava, nearest the spawn on a tie.
	bestLava := 0
	for y := 3; y < H-3; y++ {
		for x := 3; x < N-3; x++ {
			if t := at(x, y); t == sim.TileLava || t == sim.TileWater || t == sim.TileMountain || t == sim.TileCliff {
				continue
			}
			n := 0
			for dy := -4; dy <= 4; dy++ {
				for dx := -6; dx <= 6; dx++ {
					if at(x+dx, y+dy) == sim.TileLava {
						n++
					}
				}
			}
			if n > bestLava {
				bestLava = n
				sc.lava = place{float64(x) + 0.5, float64(y) + 0.5, 1}
			}
		}
	}
	// A cave with crystals: the crystal prop with most other props near it.
	var props [][2]int
	w.EachEntityIn(0, 0, N-1, H-1, func(e sim.Entity) {
		if e.Kind == sim.EntityCaveProp && sim.CaveProp(e.Var) == sim.PropCrystal {
			props = append(props, [2]int{int(e.TX), int(e.TY)})
		}
	})
	bestC := -1
	for _, p := range props {
		n := 0
		for _, q := range props {
			if abs(p[0]-q[0]) <= 15 && abs(p[1]-q[1]) <= 9 {
				n++
			}
		}
		if n > bestC {
			bestC = n
			sc.crystal = place{float64(p[0]) + 0.5, float64(p[1]) + 1.5, 1}
		}
	}
	return sc
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// sceneStages is the frame-time script: each scene is reached by a teleport,
// then held for secs (the first frames are the art being drawn the first
// time it is shown, so a scene is best run on a fresh process).
func sceneStages(sc scenes, secs float64, only string) []stage {
	herd := []string{"spawn trike 10", "spawn raptor 10", "spawn rex 4", "spawn stego 8", "spawn brachio 4", "spawn ptero 6", "spawn rabbit 10"}
	var out []stage
	add := func(name string, lines ...string) {
		if only != "" && !strings.Contains(name, only) {
			return
		}
		var steps []step
		for i, l := range lines {
			steps = append(steps, step{at: 0.1 + 0.1*float64(i), line: l})
		}
		out = append(out, stage{name: name, secs: secs, steps: steps})
	}
	add("spawn, day", "time day", "weather clear", "tp spawn")
	if sc.forest[2] > 0 {
		add("dense forest", "time day", "weather clear", sc.forest.tp())
	}
	if sc.reef[2] > 0 {
		add("beach and reef", "time day", "weather clear", sc.reef.tp())
	}
	if sc.sea[2] > 0 {
		add("open sea (boat)", "time day", "weather clear", sc.sea.tp(), "boat")
	}
	add("village, day", "time day", "weather clear", "tp spawn", "tp làng")
	add("village, night (fires)", "time 0.95", "weather clear", "tp spawn", "tp làng")
	if sc.forest[2] > 0 {
		add("forest, rain", "time day", "weather rain", sc.forest.tp())
		add("forest, storm, dusk", "time dusk", "weather storm", sc.forest.tp())
	}
	add("many dinosaurs", append([]string{"time day", "weather clear", "tp spawn"}, herd...)...)
	add("cave", "time day", "weather clear", "tp spawn", "tp hang")
	if sc.crystal[2] > 0 {
		add("cave, crystals", "time day", sc.crystal.tp())
	}
	if sc.lava[2] > 0 {
		add("lava", "time day", sc.lava.tp())
	}
	add("boss lair 1", "time day", "weather clear", "tp trùm 1")
	return out
}

// soakStage is stage i of the endless tour: it cycles through walking the
// surface, hopping over the map in strides (the ground chunks stream in and
// out), caves, the sea, villages, boss lairs, crowds of beasts, nights and
// storms, with its places shifting each round; it ends at the deadline.
func soakStage(sc scenes, i, size int, deadline time.Time) (stage, bool) {
	if time.Now().After(deadline) {
		return stage{}, false
	}
	round := i / 9
	hop := func(n int, every float64, spread float64) []step {
		var steps []step
		seed := uint64(round*977 + i*31 + 7)
		for k := range n {
			seed = seed*6364136223846793005 + 1442695040888963407
			x := 20 + float64(seed>>33%uint64(size-40))
			seed = seed*6364136223846793005 + 1442695040888963407
			y := 20 + float64(seed>>33%uint64(max(1, sc.surfaceH-40)))
			steps = append(steps, step{at: 0.1 + float64(k)*every, line: fmt.Sprintf("tp %.1f %.1f", x, y)})
		}
		return steps
	}
	walk := func(t float64) (int8, int8) {
		switch int(t/3) % 4 {
		case 0:
			return 1, 0
		case 1:
			return 1, 1
		case 2:
			return 0, 1
		}
		return -1, -1
	}
	switch i % 9 {
	case 0:
		return stage{name: "walk from spawn (speed 4)", secs: 40, steps: []step{{0.1, "tp spawn"}, {0.2, "speed 4"}, {0.3, "time day"}, {0.4, "weather clear"}}, move: walk}, true
	case 1:
		return stage{name: "hop across the surface", secs: 60, steps: append([]step{{0.05, "speed 1"}}, hop(40, 1.5, 1)...)}, true
	case 2:
		return stage{name: "village then caves", secs: 40, steps: []step{{0.1, "tp spawn"}, {0.2, "tp làng"}, {10, "tp hang"}, {20, "tp spawn"}, {21, "tp hang"}, {30, "tp 700 700"}, {31, "tp hang"}}, move: walk}, true
	case 3:
		sea := sc.sea
		if sea[2] == 0 {
			sea = sc.reef
		}
		return stage{name: "at sea (boat)", secs: 30, steps: []step{{0.1, sea.tp()}, {0.3, "boat"}, {0.5, "speed 3"}}, move: walk}, true
	case 4:
		return stage{name: "crowd of beasts", secs: 40, steps: []step{{0.1, "tp spawn"}, {0.2, "time day"}, {0.3, "spawn trike 12"}, {0.4, "spawn raptor 12"}, {0.5, "spawn rex 4"}, {0.6, "spawn stego 10"}, {0.7, "spawn mammoth 4"}, {0.8, "spawn sabre 4"}, {0.9, "spawn ptero 8"}, {1.0, "spawn plesio 4"}, {10, "god"}, {20, "kill 30"}}, move: walk}, true
	case 5:
		return stage{name: "boss lairs", secs: 50, steps: []step{{0.1, "tp trùm 1"}, {8, "tp trùm 2"}, {16, "tp trùm 3"}, {24, "tp trùm 4"}, {32, "tp trùm 5"}, {40, "tp trùm 6"}}}, true
	case 6:
		return stage{name: "night and storm walk", secs: 40, steps: []step{{0.1, "tp spawn"}, {0.2, "time 0.95"}, {0.3, "weather storm"}, {20, "tp làng"}}, move: walk}, true
	case 7:
		return stage{name: "tour of special places", secs: 40, steps: []step{{0.1, "time day"}, {0.2, "weather snow"}, {0.3, sc.forest.tp()}, {8, sc.reef.tp()}, {16, sc.lava.tp()}, {24, sc.crystal.tp()}, {32, "tp kim tự tháp"}}}, true
	}
	return stage{name: "hop, fast (3 s a hop)", secs: 60, steps: append([]step{{0.05, "weather clear"}, {0.06, "time day"}}, hop(60, 1.0, 1)...)}, true
}
