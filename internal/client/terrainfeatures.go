package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The ground of worlds made since sim.StyleIsles: what each tile's relief
// (sim.Relief: its height level, the variant of its ground, a river's flow)
// adds to the ground bake (groundhd.go) and to the map (worldmap.go). Older
// worlds have no relief, and every function here leaves them as they were.
//
//   - terraincliff.go: the cliffs, walls of chunky stones with height, and
//     the waterfalls pouring over them;
//   - terrainwater.go: rivers in their beds, fords, lakes, the shallows of
//     the sea (reefs, lagoons, shoals);
//   - terrainanim.go: the water moving, drawn over the bake every frame;
//   - terrainmap.go: all of it on the world map and the minimap;
//   - here: which features lie near each tile, the levels and slopes of the
//     land, the variants of the ground (meadows, clearings, thick and thin
//     woods, outcrops, dunes, scree, mud flats, boulder fields, ramps), wet
//     ground by water, and how wide two biomes blend (edgeWidth).

// screeRamp is the grey of loose stones: scree, boulder fields.
var screeRamp = mkRamp(0x2a2a2e, 0x3a3a40, 0x4c4b52, 0x605f66, 0x76747a, 0x8c8a8e, 0xa6a3a5)

// setRelief gives t the world's relief, if it has one, and stills the
// lakes: like swamp ponds they have banks, not surf.
func (t *terrain) setRelief(r []sim.Relief) {
	if len(r) != t.w*t.h {
		return
	}
	t.relief = r
	for i, rr := range r {
		if t.tiles[i] != sim.TileWater || rr.Variant() != sim.VarLake {
			continue
		}
		x, y := i%t.w, i/t.w
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if nx, ny := x+dx, y+dy; nx >= 0 && ny >= 0 && nx < t.w && ny < t.h {
					t.pond[ny*t.w+nx] = 1
				}
			}
		}
	}
	t.feat = reliefFeatures(t.w, t.h, t.tiles, r)
	t.depth = seaDepths(t.w, t.h, t.tiles, r) // see terrainsea.go
}

// What lies near each tile, so the bake skips what is not there (most of
// the world is flat ground and open sea).
const (
	featCliff   uint8 = 1 << iota // a cliff within two tiles
	featRiver                     // running water within a tile
	featStep                      // another level within a tile
	featLook                      // a variant of the ground, wet or soft, within a tile
	featShallow                   // shallows or a lake within two tiles
)

// reliefFeatures works out the feat flags of every tile.
func reliefFeatures(w, h int, tiles []sim.Tile, r []sim.Relief) []uint8 {
	base := make([]uint8, w*h)
	for i, t := range tiles {
		var f uint8
		switch {
		case t == sim.TileCliff:
			f |= featCliff
		case runningWater(t):
			f |= featRiver
		case t == sim.TileShallow || t == sim.TileWater && r[i].Variant() == sim.VarLake:
			f |= featShallow
		}
		if !runningWater(t) && (r[i].Variant() != sim.VarPlain || r[i].Wet() || r[i].Soft()) {
			f |= featLook
		}
		base[i] = f
	}
	// Spread each flag over its reach: rows, then columns.
	spread := func(src []uint8, mask uint8, reach int) []uint8 {
		row := make([]uint8, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if f := src[y*w+x] & mask; f != 0 {
					for dx := max(0, x-reach); dx <= min(w-1, x+reach); dx++ {
						row[y*w+dx] |= f
					}
				}
			}
		}
		out := make([]uint8, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if f := row[y*w+x]; f != 0 {
					for dy := max(0, y-reach); dy <= min(h-1, y+reach); dy++ {
						out[dy*w+x] |= f
					}
				}
			}
		}
		return out
	}
	feat := spread(base, featCliff|featShallow, 2)
	near := spread(base, featRiver|featLook, 1)
	for i := range feat {
		feat[i] |= near[i]
	}
	// Steps: a level other than this tile's within a tile.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			l := r[y*w+x].Level()
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if nx, ny := x+dx, y+dy; nx >= 0 && ny >= 0 && nx < w && ny < h && r[ny*w+nx].Level() != l {
						feat[y*w+x] |= featStep
					}
				}
			}
		}
	}
	return feat
}

// has reports whether flag f is set near tile (tx, ty).
func (t *terrain) has(tx, ty int, f uint8) bool {
	if t.feat == nil || tx < 0 || ty < 0 || tx >= t.w || ty >= t.h {
		return false
	}
	return t.feat[ty*t.w+tx]&f != 0
}

// rel is the relief of tile (tx, ty): zero off the map or without relief.
func (t *terrain) rel(tx, ty int) sim.Relief {
	if t.relief == nil || tx < 0 || ty < 0 || tx >= t.w || ty >= t.h {
		return 0
	}
	return t.relief[ty*t.w+tx]
}

// corners are the four tiles round a point, with how much each counts
// there (eased between their centres).
type corners struct {
	tx, ty [4]int
	wt     [4]float64
}

func cornersAt(u, v float64) corners {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := smooth(x-x0), smooth(y-y0)
	return corners{
		tx: [4]int{ix, ix + 1, ix, ix + 1},
		ty: [4]int{iy, iy, iy + 1, iy + 1},
		wt: [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy},
	}
}

// share is how much of the point the tiles that pass ok make up.
func (c corners) share(ok func(tx, ty int) bool) float64 {
	s := 0.0
	for k := range 4 {
		if ok(c.tx[k], c.ty[k]) {
			s += c.wt[k]
		}
	}
	return s
}

// levelAt is the height level eased between tile centres.
func (t *terrain) levelAt(u, v float64) float64 {
	c := cornersAt(u, v)
	s := 0.0
	for k := range 4 {
		s += c.wt[k] * float64(t.rel(c.tx[k], c.ty[k]).Level())
	}
	return s
}

func scaleCol(c color.RGBA, k float64) color.RGBA {
	f := func(v uint8) uint8 { return uint8(max(0, min(255, math.Round(float64(v)*k)))) }
	return color.RGBA{f(c.R), f(c.G), f(c.B), c.A}
}

// --- rivers ------------------------------------------------------------------------

// runningWater reports whether tile kind k is part of a river.
func runningWater(k sim.Tile) bool {
	return k == sim.TileRiver || k == sim.TileFord || k == sim.TileWaterfall
}

// cliffGround is the high ground cliff tile (tx, ty) is the edge of, which
// its top is drawn as.
func (t *terrain) cliffGround(tx, ty int) sim.Tile {
	lvl := t.rel(tx, ty).Level()
	for _, d := range [4][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} {
		k := t.tile(tx+d[0], ty+d[1])
		if k.Walkable() && k != sim.TileFord && t.rel(tx+d[0], ty+d[1]).Level() >= lvl {
			return k
		}
	}
	return sim.TileRocky
}

// --- the land -----------------------------------------------------------------------

// edgeWidth is how far either side of a border two biomes mix at (u, v):
// wider where the border is soft.
func (t *terrain) edgeWidth(u, v float64) float64 {
	if t.relief == nil || !t.has(int(math.Floor(u)), int(math.Floor(v)), featLook) {
		return edgeW
	}
	soft := cornersAt(u, v).share(func(x, y int) bool { return t.rel(x, y).Soft() })
	return edgeW + 0.34*soft
}

// reliefShade lays the levels, the slopes and the variants of the ground
// over land pixel c at (u, v).
func (t *terrain) reliefShade(c color.RGBA, hx, hy int, u, v float64) color.RGBA {
	if t.relief == nil {
		return c
	}
	// The higher, the lighter; slopes catch the light from the upper left.
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	lv, gx, gy := float64(t.rel(tx, ty).Level()), 0.0, 0.0
	if t.has(tx, ty, featStep) {
		lv = t.levelAt(u, v)
		if !t.has(tx, ty, featCliff) { // a cliff draws its own wall: see terraincliff.go
			gx = t.levelAt(u+0.45, v) - t.levelAt(u-0.45, v)
			gy = t.levelAt(u, v+0.45) - t.levelAt(u, v-0.45)
		}
	}
	c = scaleCol(c, 1+0.04*lv+0.12*(gx+gy))
	if !t.has(tx, ty, featLook) {
		return c
	}

	cs := cornersAt(u, v)
	var w [sim.NumGroundVariants]float64
	wet := 0.0
	for k := range 4 {
		r := t.rel(cs.tx[k], cs.ty[k])
		if runningWater(t.tile(cs.tx[k], cs.ty[k])) {
			continue // a river's variant bits are its flow
		}
		w[r.Variant()] += cs.wt[k]
		if r.Wet() {
			wet += cs.wt[k]
		}
	}
	jit := (vnoise(float64(hx)/10, float64(hy)/8, 978) - 0.5) * 0.4
	at := func(v sim.GroundVariant) float64 { return clamp01(w[v] + jit*w[v]) }
	if a := at(sim.VarMeadow); a > 0 {
		c = mix(c, grassRamp.at(4), 0.12*a)
		if hashf(fdiv(hx, 4), fdiv(hy, 4), 979) < 0.06*a && hashf(hx, hy, 980) < 0.5 {
			c = flowers[int(hashf(fdiv(hx, 4), fdiv(hy, 4), 981)*float64(len(flowers)))]
		}
	}
	if a := at(sim.VarClearing); a > 0 {
		c = mix(c, grassRamp.at(3), 0.3*a)
	}
	if a := at(sim.VarDenseWood); a > 0 {
		c = mix(c, hex(0x14261a), 0.28*a)
	}
	if a := at(sim.VarOpenWood); a > 0 {
		c = mix(c, grassRamp.at(2), 0.18*a)
	}
	if a := at(sim.VarOutcrop); a > 0 && hashf(fdiv(hx, 2), fdiv(hy, 2), 982) < 0.05*a {
		c = mix(c, hex(0xa9b26e), 0.6) // lichen
	}
	if a := at(sim.VarDunes); a > 0 && t.tile(tx, ty) == sim.TileSand {
		c = beachDunes(c, hx, hy, a) // see terrainsea.go
	} else if a > 0 {
		s := math.Sin((float64(hx)*0.35+float64(hy)*0.9)/3.4 + 5*vnoise(float64(hx)/80, float64(hy)/60, 983))
		c = scaleCol(c, 1+0.08*s*a)
	}
	if a := at(sim.VarScree); a > 0 {
		switch p, pl := pebble(hx, hy, 8, 7, 984, 0.45*a, 0.9, 2.2); p {
		case pebStone:
			c = screeRamp.at(2 + int(pl*4))
		case pebShadow:
			c = scaleCol(c, 0.78)
		}
	}
	if a := at(sim.VarMudflat); a > 0 {
		c = mix(c, hex(0x4a3c2b), 0.55*a)
		if vnoise(float64(hx)/14, float64(hy)/6, 985) > 0.66 {
			c = mix(c, hex(0x93b2b8), 0.35*a) // a sheen of water
		}
	}
	if a := at(sim.VarBoulders); a > 0 {
		switch p, pl := pebble(hx, hy, 15, 12, 986, 0.3*a, 1.6, 3.6); p {
		case pebStone:
			c = screeRamp.at(2 + int(pl*4))
		case pebShadow:
			c = scaleCol(c, 0.72)
		}
	}
	if a := at(sim.VarRamp); a > 0 {
		c = t.rampPixel(c, hx, hy, u, v, a)
	}
	if k := t.tile(tx, ty); wet > 0 && (k == sim.TileGrass || k == sim.TileForest || k == sim.TileMarsh) {
		c = mix(c, hex(0x2b5a30), 0.16*wet) // lusher by the water
	}
	return c
}

// rampPixel lays a ramp over ground c at (u, v), a (0 to 1) how much of
// the point is ramp: a path of packed earth cut down through the cliff,
// steps of flat stones across it, each lit along its front edge with a
// dark riser under it, and stones lining its sides.
func (t *terrain) rampPixel(c color.RGBA, hx, hy int, u, v, a float64) color.RGBA {
	// Which way is down it.
	gx := t.levelAt(u+0.5, v) - t.levelAt(u-0.5, v)
	gy := t.levelAt(u, v+0.5) - t.levelAt(u, v-0.5)
	dx, dy := 0.0, 1.0
	if n := math.Hypot(gx, gy); n > 0.05 {
		dx, dy = -gx/n, -gy/n
	}
	k := smooth(clamp01((a - 0.2) / 0.45))
	if k <= 0 {
		return c
	}
	// Packed earth, or packed snow, worn into the ground it crosses.
	earth := mix(c, pathHD.g(0.42+0.25*vnoise(float64(hx)/9, float64(hy)/7, 1401), hx, hy), 0.6)
	switch t.tile(int(math.Floor(u)), int(math.Floor(v))) {
	case sim.TileSnow, sim.TileIce:
		earth = mix(scaleCol(c, 0.9), hex(0x9fb0c2), 0.25)
	case sim.TileDesert, sim.TileSand:
		earth = scaleCol(c, 0.88)
	}
	if k < 0.7 {
		// Its sides: the ground worn, a few stones lining it.
		if p, l := pebble(hx, hy, 13, 11, 1402, 0.3*k, 1.8, 3.6); p == pebStone {
			return stepStone.at(1 + int(l*4.4))
		} else if p == pebShadow {
			return scaleCol(mix(c, earth, k), 0.8)
		}
		return mix(c, earth, k/0.7)
	}
	// The steps: across the slope, a stone tread every so often.
	along := float64(hx)*dx + float64(hy)*dy*1.3 + 4*vnoise(float64(hx)*dy/14+float64(hy)*dx/14, 0, 1403)
	const per = 15.0
	m := along - per*math.Floor(along/per)
	row := int(math.Floor(along / per))
	across := float64(hx)*dy - float64(hy)*dx
	// Each tread is a few stones side by side.
	seg := math.Floor((across + 9*hashf(row, 0, 1404)) / 13)
	tone := hashf(row, int(seg), 1405)
	segEdge := across + 9*hashf(row, 0, 1404) - seg*13
	switch {
	case m < 2.2:
		return mix(stepStone.at(1), earth, 0.35) // the riser's shadow
	case m < 3.6:
		return stepStone.at(4 + int(tone*1.9)) // the front edge of the tread, lit
	case m < 8:
		if segEdge < 1 {
			return stepStone.at(1)
		}
		return stepStone.at(2 + int(tone*2.5+0.3*hashf(hx, hy, 1406)))
	case m < 9.5:
		return scaleCol(earth, 0.8)
	}
	return earth
}
