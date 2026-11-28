package sim

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"testing"
	"time"
)

// CAVES_DUMP=dir go test ./internal/sim -run DumpCaves -v prints what the natural
// caves of a world of 1024 hold, island by island, and writes pictures of
// the biggest system and of a middling one, both its levels side by side:
// the zones in their colours, the main route pale, ways up and down white,
// forks cyan, chests yellow, jade green, spiders magenta and bears red.
//
// CAVES_STYLE picks the world style by number (StyleLatest by default),
// CAVES_SEED the seed (42), CAVES_ISLAND the island of the second picture
// (by default the biggest of 6,000 to 15,000 tiles).
func TestDumpCaves(t *testing.T) {
	dir := os.Getenv("CAVES_DUMP")
	if dir == "" {
		t.Skip()
	}
	style, seed, pick := StyleLatest, uint64(42), -1
	if v, err := strconv.Atoi(os.Getenv("CAVES_STYLE")); err == nil {
		style = WorldStyle(v)
	}
	if v, err := strconv.ParseUint(os.Getenv("CAVES_SEED"), 10, 64); err == nil {
		seed = v
	}
	if v, err := strconv.Atoi(os.Getenv("CAVES_ISLAND")); err == nil {
		pick = v
	}
	size := SizeHuge
	t0 := time.Now()
	w, err := Generate(GenConfig{Seed: seed, Width: size, Height: size, Style: style})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("style %d seed %d: generated in %v, map %dx%d", style, seed, time.Since(t0), w.width, w.height)
	kinds := map[EntityKind]int{}
	for _, e := range w.entities {
		if w.Underground(e.TY) {
			kinds[e.Kind]++
		}
	}
	t.Logf("underground entities: coal %d jade %d agate %d obsidian %d treasure %d stalag %d boulders %d pebbles %d mush %d glow %d",
		kinds[EntityCoal], kinds[EntityIronOre], kinds[EntityPlatinumOre], kinds[EntityDiamondOre], kinds[EntityTreasure], kinds[EntityStalagmite], kinds[EntityBoulder], kinds[EntityPebble], kinds[EntityMushroom], kinds[EntityGlowPlant])
	cr := map[CreatureKind]int{}
	for _, c := range w.fauna.creatures {
		if w.Underground(int(c.Y)) {
			cr[c.Kind]++
		}
	}
	t.Logf("underground creatures: %v", cr)
	liquid := map[Tile]int{}
	for i, tl := range w.tiles {
		if i >= w.width*w.surfaceH && (tl == TileRiver || tl == TileLava || tl == TileIce || tl == TileFord) {
			liquid[tl]++
		}
	}
	zones := map[CaveZone]int{}
	for _, s := range w.caves.systems {
		for _, c := range s.chambers {
			zones[c.zone]++
		}
	}
	t.Logf("water %d, lava %d, ice %d, fords %d; chambers by zone %v", liquid[TileRiver], liquid[TileLava], liquid[TileIce], liquid[TileFord], zones)
	floor := 0
	for _, s := range w.caves.systems {
		floor += s.floor[0] + s.floor[1]
		line := fmt.Sprintf("island %d %v area %d: mouths %d, shafts %d, floor %d+%d", s.island, w.islands[s.island].Kind, s.area, len(s.mouths), len(s.shafts), s.floor[0], s.floor[1])
		for lv := range s.levels {
			st := caveStatsOf(s, lv)
			line += fmt.Sprintf(" | L%d chambers %d pockets %d forks %d (junctions %d) tunnels %d loops %d routes %d",
				lv, st.chambers, st.pockets, st.forks, st.junctions, st.tunnels, st.loops, st.routes)
		}
		t.Log(line)
	}
	t.Logf("secret passages %d; floor %d", len(w.caves.secrets), floor)
	if w.caveWays() {
		var byKind [numChestKinds][3]int
		for _, e := range w.entities {
			if e.Kind == EntityTreasure && w.caveLevel(e.TY) >= 0 {
				byKind[min(int(e.Var>>2), numChestKinds-1)][e.Var&3]++
			}
		}
		t.Logf("chests by kind (cache, side, end, deep) and ring (home, near, far): %v", byKind)
	}
	// The biggest system, and a middling one.
	big, mid := w.caves.systems[0], (*caveSystem)(nil)
	for _, s := range w.caves.systems {
		if s.area > big.area {
			big = s
		}
		if s.island == pick || pick < 0 && s.area >= 6000 && s.area <= 15000 && (mid == nil || s.area > mid.area) {
			mid = s
		}
	}
	dumpCaveSystem(t, w, big, fmt.Sprintf("%s/style%d_big_island%d.png", dir, style, big.island))
	if mid != nil {
		dumpCaveSystem(t, w, mid, fmt.Sprintf("%s/style%d_medium_island%d.png", dir, style, mid.island))
	}
}

// dumpCaveSystem writes a picture of system s to path, its levels side by
// side.
func dumpCaveSystem(t *testing.T, w *World, s *caveSystem, path string) {
	px := 3
	if s.gw < 200 {
		px = 6
	}
	img := image.NewRGBA(image.Rect(0, 0, (2*s.gw+4)*px, s.gh*px))
	dot := func(lv, x, y, r int, col color.RGBA) {
		ox := lv * (s.gw + 4)
		for yy := -r; yy < px+r; yy++ {
			for xx := -r; xx < px+r; xx++ {
				img.Set((ox+x)*px+xx, y*px+yy, col)
			}
		}
	}
	for lv := 0; lv < 2; lv++ {
		top := CaveBandTop(w.surfaceH, lv)
		for y := 0; y < s.gh; y++ {
			for x := 0; x < s.gw; x++ {
				tx, ty := s.x0+x, top+s.y0+y
				c := color.RGBA{20, 18, 24, 255}
				if w.TileAt(tx, y+s.y0) != TileWater && w.TileAt(tx, y+s.y0) != TileShallow {
					c = color.RGBA{36, 40, 34, 255}
				}
				switch w.TileAt(tx, ty) {
				case TileCave:
					cell := w.caveCell(tx, ty)
					c = zoneColor[cell.Zone()]
					if cell.Route() {
						c = color.RGBA{220, 190, 120, 255}
					}
				case TileRiver:
					c = color.RGBA{40, 90, 200, 255}
				case TileLava:
					c = color.RGBA{255, 100, 20, 255}
				case TileIce:
					c = color.RGBA{200, 230, 255, 255}
				}
				if e := w.entityAt(tx, ty); e != nil {
					switch e.Kind {
					case EntityTreasure:
						c = color.RGBA{255, 230, 0, 255}
					case EntityCaveExit, EntityCaveDown, EntityCaveUp:
						c = color.RGBA{255, 255, 255, 255}
					case EntityStalagmite:
						c = color.RGBA{90, 80, 70, 255}
					case EntityIronOre:
						c = color.RGBA{0, 255, 120, 255}
					case EntityCoal:
						c = color.RGBA{0, 0, 0, 255}
					}
				}
				dot(lv, x, y, 0, c)
			}
		}
	}
	for _, c := range s.chambers {
		if c.junction {
			dot(c.level, int(c.x)-s.x0, int(c.y)-s.y0, 1, color.RGBA{0, 230, 255, 255})
		}
	}
	for _, e := range w.entities {
		if e.Kind != EntityTreasure {
			continue
		}
		if lv := w.caveLevel(e.TY); lv >= 0 {
			x, y := e.TX-s.x0, e.TY-CaveBandTop(w.surfaceH, lv)-s.y0
			if x >= 0 && y >= 0 && x < s.gw && y < s.gh {
				dot(lv, x, y, 1, color.RGBA{255, 230, 0, 255})
			}
		}
	}
	for _, c := range w.fauna.creatures {
		lv := w.caveLevel(int(c.Y))
		if lv < 0 {
			continue
		}
		x, y := int(c.X)-s.x0, int(c.Y)-CaveBandTop(w.surfaceH, lv)-s.y0
		if x < 0 || y < 0 || x >= s.gw || y >= s.gh {
			continue
		}
		col := color.RGBA{255, 0, 255, 255}
		if c.Kind == CreatureBear {
			col = color.RGBA{255, 0, 0, 255}
		}
		dot(lv, x, y, 1, col)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
	t.Logf("wrote %s", path)
}

// caveLevelStat is how one level of a system is laid out.
type caveLevelStat struct {
	chambers  int // besides the forks and the pockets, with the halls
	pockets   int
	junctions int // forks in a passage
	forks     int // chambers and junctions where three passages or more meet
	tunnels   int
	loops     int // tunnels past those a tree joining every chamber needs
	routes    int // ways through the level with no passage in common
}

func caveStatsOf(s *caveSystem, lv int) caveLevelStat {
	var st caveLevelStat
	nodes := 0
	for _, c := range s.chambers {
		if c.level != lv {
			continue
		}
		nodes++
		switch {
		case c.junction:
			st.junctions++
		case c.pocket != 0:
			st.pockets++
		default:
			st.chambers++
		}
		if len(c.adj) >= 3 {
			st.forks++
		}
	}
	for _, t := range s.tunnels {
		if t.level == lv {
			st.tunnels++
		}
	}
	st.loops = st.tunnels - nodes + 1
	a, b := s.levelEnds(lv)
	st.routes = disjointRoutes(s, a, b)
	return st
}

// disjointRoutes is how many ways lead from chamber a to chamber b with no
// tunnel in common: the most flow through the tunnels, one each.
func disjointRoutes(s *caveSystem, a, b int) int {
	if a == b {
		return 0
	}
	type arc struct{ to, rev, cap int }
	g := make([][]arc, len(s.chambers))
	for _, t := range s.tunnels {
		g[t.a] = append(g[t.a], arc{t.b, len(g[t.b]), 1})
		g[t.b] = append(g[t.b], arc{t.a, len(g[t.a]) - 1, 1})
	}
	for flow := 0; ; flow++ {
		prev := make([][2]int, len(g))
		for i := range prev {
			prev[i] = [2]int{-1, -1}
		}
		prev[a] = [2]int{a, -1}
		for q := []int{a}; len(q) > 0 && prev[b][0] < 0; q = q[1:] {
			u := q[0]
			for k, e := range g[u] {
				if e.cap > 0 && prev[e.to][0] < 0 {
					prev[e.to] = [2]int{u, k}
					q = append(q, e.to)
				}
			}
		}
		if prev[b][0] < 0 {
			return flow
		}
		for v := b; v != a; {
			u, k := prev[v][0], prev[v][1]
			g[u][k].cap--
			g[v][g[u][k].rev].cap++
			v = u
		}
	}
}

var zoneColor = [NumCaveZones]color.RGBA{
	ZoneTunnel: {110, 100, 92, 255}, ZoneHall: {140, 130, 118, 255}, ZoneEntrance: {200, 200, 160, 255},
	ZoneNest: {170, 90, 170, 255}, ZoneDen: {150, 70, 50, 255}, ZoneCrystal: {80, 220, 200, 255},
	ZoneLake: {60, 120, 200, 255}, ZoneMushroom: {120, 200, 90, 255}, ZoneRoost: {90, 80, 60, 255},
	ZoneFossil: {230, 220, 190, 255}, ZonePaintings: {210, 120, 80, 255}, ZoneTomb: {220, 180, 90, 255},
	ZoneLava: {200, 70, 30, 255}, ZoneFrozen: {170, 210, 240, 255}, ZoneFlooded: {70, 100, 70, 255},
	ZoneStream: {60, 150, 120, 255},
}
