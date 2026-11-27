package main

import (
	"math"
	"strings"

	"survivalgame/internal/sim"
)

// Spots in the natural caves (sim.StyleCaves) worth a look, for -at:
//
//   - cavezone:NAME: in a chamber of that kind (crystal, lake, mushroom,
//     nest, den, roost, fossil, paintings, tomb, lava, frozen, flooded,
//     stream, hall, entrance, tunnel);
//   - caverock:NAME: in a plain chamber under that rock (limestone, mossy,
//     sandstone, basalt, ice, mud).
//
// Each is the one that fills the screen best, nearest the spawn of those.

var caveZoneNames = map[string]sim.CaveZone{
	"tunnel": sim.ZoneTunnel, "hall": sim.ZoneHall, "entrance": sim.ZoneEntrance, "nest": sim.ZoneNest,
	"den": sim.ZoneDen, "crystal": sim.ZoneCrystal, "lake": sim.ZoneLake, "mushroom": sim.ZoneMushroom,
	"roost": sim.ZoneRoost, "fossil": sim.ZoneFossil, "paintings": sim.ZonePaintings, "tomb": sim.ZoneTomb,
	"lava": sim.ZoneLava, "frozen": sim.ZoneFrozen, "flooded": sim.ZoneFlooded, "stream": sim.ZoneStream,
}

var caveRockNames = map[string]sim.CaveRock{
	"limestone": sim.RockLimestone, "mossy": sim.RockMossy, "sandstone": sim.RockSandstone,
	"basalt": sim.RockBasalt, "ice": sim.RockIce, "mud": sim.RockMud,
}

// caveSpot finds where to stand for -at cavezone:NAME or caverock:NAME.
func caveSpot(w *sim.World, what string, sx, sy float64) (x, y float64, ok bool) {
	cells := w.CaveCells()
	if cells == nil {
		return 0, 0, false
	}
	W, H := w.Size()
	from := w.SurfaceHeight()
	var match func(c sim.CaveCell) bool
	switch {
	case strings.HasPrefix(what, "cavezone:"):
		z, ok := caveZoneNames[strings.TrimPrefix(what, "cavezone:")]
		if !ok {
			return 0, 0, false
		}
		match = func(c sim.CaveCell) bool { return c.Zone() == z }
	case strings.HasPrefix(what, "caverock:"):
		r, ok := caveRockNames[strings.TrimPrefix(what, "caverock:")]
		if !ok {
			return 0, 0, false
		}
		match = func(c sim.CaveCell) bool { return c.Rock() == r && c.Zone() == sim.ZoneHall }
	default:
		return 0, 0, false
	}
	tiles := w.Tiles()
	in := func(tx, ty int) bool {
		if tx < 0 || tx >= W || ty < from || ty >= H {
			return false
		}
		i := ty*W + tx
		return tiles[i] != sim.TileMountain && match(cells[(ty-from)*W+tx])
	}
	best, bestD := -1.0, math.Inf(1)
	for ty := from; ty < H; ty += 2 {
		for tx := 0; tx < W; tx += 2 {
			if !in(tx, ty) || !tiles[ty*W+tx].Walkable() {
				continue
			}
			n := 0
			for y := ty - 5; y <= ty+5; y++ {
				for x := tx - 8; x <= tx+8; x++ {
					if in(x, y) {
						n++
					}
				}
			}
			share := float64(n) / (11 * 17)
			d := math.Hypot(float64(tx)-sx, float64(ty)-sy)
			if share > best+0.08 || share > best-0.08 && d < bestD {
				best, bestD, x, y = math.Max(best, share), d, float64(tx)+0.5, float64(ty)+0.5
			}
		}
	}
	return x, y, best > 0
}

var cavePropNames = map[string]sim.CaveProp{
	"crystal": sim.PropCrystal, "glowshroom": sim.PropGlowShroom, "web": sim.PropWeb, "eggsac": sim.PropEggSac,
	"bones": sim.PropBones, "fossil": sim.PropFossil, "painting": sim.PropPainting, "sarcophagus": sim.PropSarcophagus,
	"urn": sim.PropUrn, "roots": sim.PropRoots, "icicles": sim.PropIcicles, "obsidian": sim.PropObsidian, "roost": sim.PropRoost,
}

// cavePropSpot finds where to stand for -at caveprop:NAME: a little south of
// the nearest prop of that kind, dx, dy tiles off it.
func cavePropSpot(w *sim.World, what string, sx, sy, dx, dy float64) (x, y float64, ok bool) {
	p, known := cavePropNames[strings.TrimPrefix(what, "caveprop:")]
	if !strings.HasPrefix(what, "caveprop:") || !known {
		return 0, 0, false
	}
	best := math.Inf(1)
	w.EachEntityIn(0, w.SurfaceHeight(), 1<<14, 1<<14, func(e sim.Entity) {
		if e.Kind != sim.EntityCaveProp || sim.CaveProp(e.Var) != p {
			return
		}
		if d := math.Hypot(float64(e.TX)-sx, float64(e.TY)-sy); d < best {
			best, x, y, ok = d, float64(e.TX)+0.5+dx, float64(e.TY)+0.5+dy, true
		}
	})
	return x, y, ok
}
