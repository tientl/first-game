package sim

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"
)

// CAVES_DUMP=dir go test ./internal/sim -run DumpCaves -v prints what the natural
// caves of a world of 1024 hold, island by island, and writes a picture of
// the biggest system, both its levels side by side: the zones in their
// colours, the main route pale, ways up and down white, chests yellow,
// jade green, spiders magenta and bears red.
func TestDumpCaves(t *testing.T) {
	dir := os.Getenv("CAVES_DUMP")
	if dir == "" {
		t.Skip()
	}
	size := SizeHuge
	t0 := time.Now()
	w, err := Generate(GenConfig{Seed: 42, Width: size, Height: size, Style: StyleCaves})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("generated in %v, map %dx%d", time.Since(t0), w.width, w.height)
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
		n0, n1 := 0, 0
		for _, c := range s.chambers {
			if c.level == 0 {
				n0++
			} else {
				n1++
			}
		}
		floor += s.floor[0] + s.floor[1]
		t.Logf("island %d %v area %d: mouths %d, chambers %d+%d, tunnels %d, floor %d+%d, dest %d", s.island, w.islands[s.island].Kind, s.area, len(s.mouths), n0, n1, len(s.tunnels), s.floor[0], s.floor[1], s.dest)
	}
	t.Logf("secret passages %d; floor %d", len(w.caves.secrets), floor)
	// The biggest system, both levels side by side.
	big := w.caves.systems[0]
	for _, s := range w.caves.systems {
		if s.area > big.area {
			big = s
		}
	}
	for k, s := range w.caves.systems {
		if s != big && k != 1 {
			continue
		}
		px := 3
		img := image.NewRGBA(image.Rect(0, 0, (2*s.gw+4)*px, s.gh*px))
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
					ox := lv * (s.gw + 4)
					for yy := 0; yy < px; yy++ {
						for xx := 0; xx < px; xx++ {
							img.Set((ox+x)*px+xx, y*px+yy, c)
						}
					}
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
			ox := lv * (s.gw + 4)
			for yy := -1; yy <= px; yy++ {
				for xx := -1; xx <= px; xx++ {
					img.Set((ox+x)*px+xx, y*px+yy, col)
				}
			}
		}
		f, _ := os.Create(dir + "/" + string(rune('a'+k%26)) + "_system.png")
		png.Encode(f, img)
		f.Close()
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
