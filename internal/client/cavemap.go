package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The natural caves on the world map (M) and the minimap. The map of such a
// world is baked layer by layer, each as big as the islands' map: the
// surface, the upper caves, the lower caves; the map shows the one the
// player is on, under the same fog of what they have seen. The caves are
// drawn plainly: their floor in the colour of their rock, the main route a
// paler trail down it, the walls a line round it, water, lava and ice in
// their colours, and the ways up and down this player has seen marked.

var (
	caveMapRock  = hex(0x231e28)
	caveMapWall  = hex(0x4a4152)
	caveMapRoute = hex(0xf0dfae)
	caveMapIce   = hex(0xcfe9f2)
	// caveMapFloor is the floor of each rock.
	caveMapFloor = [sim.NumCaveRocks]color.RGBA{
		sim.RockLimestone: hex(0x9c8b77),
		sim.RockMossy:     hex(0x627a62),
		sim.RockSandstone: hex(0xb87a4e),
		sim.RockBasalt:    hex(0x5c5a6a),
		sim.RockIce:       hex(0x8fbdd0),
		sim.RockMud:       hex(0x6b6b45),
	}
)

// bakeCaveMaps bakes the map of a world with natural caves, a layer at a
// time: the surface, then each level of the caves.
func (t *terrain) bakeCaveMaps(gen int, out chan<- mapResult) {
	H := t.cave.from
	sx := mapScale(t.w, H)
	sy := sx * TileH / TileW
	w, h := max(1, int(float64(t.w)*sx)), max(1, int(float64(H)*sy))
	bake := func(r0 int) []byte {
		c := newCanvas(w, h)
		inRows(h, func(y int) {
			for x := 0; x < w; x++ {
				c.px[y*w+x] = t.mapPixel((float64(x)+0.5)/sx, float64(r0)+(float64(y)+0.5)/sy, x, y)
			}
		})
		return c.bytes()
	}
	out <- mapResult{gen: gen, w: w, h: h, data: bake(0)}
	for k := 0; k < t.cave.levels; k++ {
		out <- mapResult{gen: gen, w: w, h: h, data: bake(sim.CaveBandTop(H, k)), layer: k + 1}
	}
}

// caveMapPixel colours the map at (u, v), in tiles, in the caves.
func (t *terrain) caveMapPixel(u, v float64, px, py int) color.RGBA {
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	switch t.tile(tx, ty) {
	case sim.TileMountain:
		for _, d := range [4][2]float64{{0.45, 0}, {-0.45, 0}, {0, 0.45}, {0, -0.45}} {
			if caveOpen(t.tile(int(math.Floor(u+d[0])), int(math.Floor(v+d[1])))) {
				return caveMapWall
			}
		}
		return mix(caveMapRock, colInk, 0.3*vnoise(u*0.2, v*0.25, 3170))
	case sim.TileRiver, sim.TileFord:
		return mapLake
	case sim.TileLava:
		return lavaRamp.at(3)
	case sim.TileIce:
		return caveMapIce
	}
	c := t.caveCell(tx, ty)
	col := caveMapFloor[c.Rock()%sim.NumCaveRocks]
	col = scaleCol(col, 0.92+0.16*vnoise(u*0.7, v*0.8, 3171))
	switch c.Zone() {
	case sim.ZoneEntrance:
		col = mix(col, hex(0xf2e6b0), 0.35)
	case sim.ZoneNest:
		col = mix(col, hex(0xa070b0), 0.4)
	case sim.ZoneDen:
		col = mix(col, hex(0xa05a48), 0.4)
	case sim.ZoneCrystal:
		col = mix(col, hex(0x63c4cc), 0.45)
	case sim.ZoneMushroom:
		col = mix(col, hex(0x7e659b), 0.45)
	case sim.ZoneTomb:
		col = mix(col, hex(0xe1a845), 0.45)
	case sim.ZoneFrozen:
		col = mix(col, caveMapIce, 0.4)
	case sim.ZoneFossil, sim.ZonePaintings:
		col = mix(col, hex(0xe3e6ac), 0.3)
	}
	if c.Route() {
		col = mix(col, caveMapRoute, 0.45)
	}
	return col
}

// mapImage is the map image of the rows r0 to r0+rows the map shows.
func (g *Game) mapImage(r0, rows float64) *ebiten.Image {
	if !g.naturalCaves() {
		return g.mapLayer(g.wmap.img, r0, rows)
	}
	if k, ok := g.caveLevel(); ok && int(r0) >= g.surfaceH {
		return g.wmap.layers[k]
	}
	return g.wmap.img
}

// caveMark is a way up or down this client has seen.
type caveMark struct {
	kind   sim.EntityKind
	tx, ty int16
}

// rememberCaveMarks notes the ways up and down in a snapshot, so the map
// shows them once seen.
func (g *Game) rememberCaveMarks(s protocol.Snapshot) {
	if !g.naturalCaves() {
		return
	}
	if g.caveMarks == nil {
		g.caveMarks = make(map[sim.EntityID]caveMark)
	}
	for _, e := range s.Entities {
		switch e.Kind {
		case sim.EntityCaveExit, sim.EntityCaveDown, sim.EntityCaveUp:
			g.caveMarks[e.ID] = caveMark{e.Kind, e.TX, e.TY}
		}
	}
}

// drawCaveMarks draws the ways up and down seen on rows r0 to r0+rows of
// the map, at(x, y) placing a point of the world.
func (g *Game) drawCaveMarks(dst *ebiten.Image, at func(x, y float64) (float32, float32), r0, rows float64) {
	for _, m := range g.caveMarks {
		if float64(m.ty) < r0 || float64(m.ty) >= r0+rows {
			continue
		}
		x, y := at(float64(m.tx)+0.5, float64(m.ty)+0.5)
		drawCaveMark(dst, m.kind, x, y)
	}
}

// drawCaveMark draws a way up (a sun: daylight) or a rope (a dark hole with
// an arrow, up or down) centred on (x, y).
func drawCaveMark(dst *ebiten.Image, k sim.EntityKind, x, y float32) {
	switch k {
	case sim.EntityCaveExit:
		vector.FillCircle(dst, x, y, 6, colInk, true)
		vector.FillCircle(dst, x, y, 5, hex(0xffe28a), true)
		vector.FillCircle(dst, x-1, y-1, 2, hex(0xfffbe2), true)
	default:
		vector.FillCircle(dst, x, y, 6, hex(0xe0c890), true)
		vector.FillCircle(dst, x, y, 4.5, colInk, true)
		s := float32(1)
		if k == sim.EntityCaveUp {
			s = -1
		}
		vector.StrokeLine(dst, x, y-3*s, x, y+3*s, 1.5, hex(0xe0c890), true)
		vector.StrokeLine(dst, x-2.5, y+0.5*s, x, y+3*s, 1.5, hex(0xe0c890), true)
		vector.StrokeLine(dst, x+2.5, y+0.5*s, x, y+3*s, 1.5, hex(0xe0c890), true)
	}
}

// SetMapZoom sets the world map's zoom, 1 (the whole map) to the most: for
// the screenshot tool.
func (g *Game) SetMapZoom(z int) { g.mapZoom = min(maxMapZoom, max(1, z)) }
