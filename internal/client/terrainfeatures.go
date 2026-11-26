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
// Each look is a plain placeholder for finished art to replace, one
// function apiece:
//
//   - shallowSea: the sea over the shallows (sim.TileShallow): lighter, with
//     coral on reefs, calmer in lagoons, sandy over shoals.
//   - lakePixel: a lake's still water (sim.TileWater with sim.VarLake).
//   - riverPixel: rivers (sim.TileRiver) and their banks, with flowPixel for
//     running water, fordPixel for a ford (sim.TileFord) and fallPixel for
//     a waterfall (sim.TileWaterfall).
//   - cliffPixel: a cliff (sim.TileCliff): a face of rock where the ground
//     drops to the south, its rim elsewhere; a sea stack the same.
//   - reliefShade: the levels and slopes of the land, and the variants of
//     the ground: meadows, clearings, thick and thin woods, outcrops, dunes,
//     scree, mud flats, boulder fields, ramps; and wet ground by water.
//   - edgeWidth: how wide two biomes blend, wider on soft borders.
//   - mapRelief and mapShallow: the same, simply, on the world map and the
//     minimap.

var (
	riverRamp   = mkRamp(0x24607e, 0x2f7495, 0x3c88a8, 0x4f9db9, 0x6ab4ca, 0x93cfdc)
	lakeRamp    = mkRamp(0x1e4866, 0x265a7c, 0x316e92, 0x3f83a6, 0x5598b6)
	fallRamp    = mkRamp(0x7fb8cf, 0xa6d4e4, 0xcbe8f0, 0xeaf7fa)
	bankRamp    = mkRamp(0x4a3c2a, 0x5c4b34, 0x6f5c40, 0x85714f)
	cliffEarth  = mkRamp(0x2f241e, 0x413228, 0x554233, 0x6a5440, 0x80684f, 0x977e61, 0xad957a)
	cliffStone  = mkRamp(0x2a2a2e, 0x3a3a40, 0x4c4b52, 0x605f66, 0x76747a, 0x8c8a8e, 0xa6a3a5)
	cliffSand   = mkRamp(0x4a2418, 0x63321f, 0x7d4429, 0x985834, 0xb06d42, 0xc78553, 0xdc9f68)
	cliffAsh    = mkRamp(0x151316, 0x1f1c20, 0x2a272b, 0x363237, 0x433e44, 0x524c52)
	shallowTint = hex(0x4ec0c6)
	lagoonTint  = hex(0x63d6c2)
	shoalTint   = hex(0xb4d2a4)
	coralCols   = []color.RGBA{hex(0xe58a8a), hex(0xe9a65e), hex(0xb07ad0), hex(0xf0d070)}
	mapRiver    = hex(0x3f8fc0)
	mapCliff    = hex(0x3a2a22)
)

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

// --- the sea and the lakes -------------------------------------------------------

// shallowSea tints sea pixel c at (u, v) over the shallows.
func (t *terrain) shallowSea(c color.RGBA, hx, hy int, u, v float64) color.RGBA {
	if t.relief == nil || !t.has(int(math.Floor(u)), int(math.Floor(v)), featShallow) {
		return c
	}
	// Warped, so the shallows' edge wanders off the lines of the tiles.
	wu := u + (vnoise(u*0.35, v*0.35, 958)-0.5)*1.4
	wv := v + (vnoise(u*0.35+5, v*0.35, 959)-0.5)*1.4
	cs := cornersAt(wu, wv)
	sh := cs.share(func(x, y int) bool { return t.tile(x, y) == sim.TileShallow })
	if sh <= 0 {
		return c
	}
	reef := cs.share(func(x, y int) bool { return t.rel(x, y).Variant() == sim.VarReef })
	lagoon := cs.share(func(x, y int) bool { return t.rel(x, y).Variant() == sim.VarLagoon })
	shoal := cs.share(func(x, y int) bool { return t.rel(x, y).Variant() == sim.VarShoal })
	tint := shallowTint
	tint = mix(tint, lagoonTint, lagoon)
	tint = mix(tint, shoalTint, shoal*0.8)
	sh = smooth(clamp01(sh*1.3 + (vnoise(u*1.7, v*1.7, 960)-0.5)*0.5))
	c = mix(c, withAlpha(tint, max(c.A, 150)), sh*0.8)
	if reef > 0.3 {
		// Coral heads in clumps, each its own colour, dim under the water.
		if n := vnoise(float64(hx)/16, float64(hy)/12, 961); n > 0.7-0.06*reef {
			col := coralCols[int(hashf(fdiv(hx, 40), fdiv(hy, 30), 962)*float64(len(coralCols)))]
			col = mix(col, tint, 0.45)
			if n < 0.74-0.06*reef {
				col = scaleCol(col, 0.75) // its edge
			}
			c = mix(c, withAlpha(col, 215), 0.85)
		}
	}
	return c
}

// lakePixel colours a pixel of a lake, s (< 0) tiles in from its shore, if
// (u, v) is one.
func (t *terrain) lakePixel(hx, hy int, u, v, s float64) (color.RGBA, bool) {
	if t.relief == nil || !t.has(int(math.Floor(u)), int(math.Floor(v)), featShallow) {
		return color.RGBA{}, false
	}
	cs := cornersAt(u, v)
	if cs.share(func(x, y int) bool { return t.rel(x, y).Variant() == sim.VarLake && t.tile(x, y) == sim.TileWater }) <= 0 {
		return color.RGBA{}, false
	}
	if s > -0.12 {
		return bankRamp.pick(0.55+(hashf(hx, hy, 963)-0.5)*0.3, hx, hy), true
	}
	d := clamp01(-s / 2.5)
	c := lakeRamp.pick(0.85-0.75*d+(vnoise(float64(hx)/11, float64(hy)/6, 964)-0.5)*0.25, hx, hy)
	if hashf(fdiv(hx, 3), hy, 965) < 0.006 {
		c = mix(c, colFoam, 0.6) // a glint
	}
	return c, true
}

// --- rivers ------------------------------------------------------------------------

// runningWater reports whether tile kind k is part of a river.
func runningWater(k sim.Tile) bool {
	return k == sim.TileRiver || k == sim.TileFord || k == sim.TileWaterfall
}

// riverPixel colours a pixel of a river or its bank, if (u, v) is one.
func (t *terrain) riverPixel(hx, hy int, u, v float64) (color.RGBA, bool) {
	if t.relief == nil {
		return color.RGBA{}, false
	}
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	if !t.has(tx, ty, featRiver) {
		return color.RGBA{}, false
	}
	// Warped, so the banks wander off the lines of the tiles.
	wu := u + (vnoise(u*0.9, v*0.9, 994)-0.5)*0.7
	wv := v + (vnoise(u*0.9+3, v*0.9, 995)-0.5)*0.7
	f := cornersAt(wu, wv).share(func(x, y int) bool { return runningWater(t.tile(x, y)) })
	f += (vnoise(u*2.3, v*2.3, 966) - 0.5) * 0.22
	if f < 0.3 {
		return color.RGBA{}, false
	}
	if f < 0.44 { // the bank: wet earth and gravel
		c := bankRamp.pick(0.35+(f-0.3)*2+(hashf(hx, hy, 967)-0.5)*0.3, hx, hy)
		if p, pl := pebble(hx, hy, 8, 7, 968, 0.3, 0.8, 1.6); p == pebStone {
			c = mix(c, cliffStone.at(3+int(pl*3)), 0.7)
		}
		return c, true
	}
	// The water: the river tile under the pixel, or the nearest beside it.
	wx, wy := tx, ty
	if !runningWater(t.tile(wx, wy)) {
		best := math.Inf(1)
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := tx+d[0], ty+d[1]
			if !runningWater(t.tile(nx, ny)) {
				continue
			}
			if dd := math.Hypot(float64(nx)+0.5-u, float64(ny)+0.5-v); dd < best {
				best, wx, wy = dd, nx, ny
			}
		}
	}
	depth := clamp01((f - 0.44) / 0.5)
	switch t.tile(wx, wy) {
	case sim.TileWaterfall:
		return fallPixel(hx, hy, u-float64(wx), v-float64(wy)), true
	case sim.TileFord:
		return fordPixel(hx, hy, depth), true
	}
	dx, dy := t.rel(wx, wy).Flow().Delta()
	return flowPixel(hx, hy, u, v, float64(dx), float64(dy), depth), true
}

// flowPixel is running water flowing along (dx, dy), depth 0 at its bank
// to 1 in its middle: streaks along the current, flecks of foam.
func flowPixel(hx, hy int, u, v, dx, dy, depth float64) color.RGBA {
	along := u*dx + v*dy
	across := -u*dy + v*dx
	streak := vnoise(across*5, along*0.6, 969)
	l := 0.75 - 0.45*depth + (streak-0.5)*0.35
	c := riverRamp.pick(l, hx, hy)
	if vnoise(across*9+3, along*2.2, 970) > 0.8 && depth > 0.2 {
		c = mix(c, colFoam, 0.45) // a ripple
	}
	return c
}

// fordPixel is a ford: shallow water over stones, depth 0 at its bank.
func fordPixel(hx, hy int, depth float64) color.RGBA {
	c := riverRamp.pick(0.85-0.25*depth+(hashf(hx, hy, 971)-0.5)*0.1, hx, hy)
	switch p, pl := pebble(hx, hy, 11, 9, 972, 0.75, 1.4, 3.2); p {
	case pebStone:
		c = mix(cliffStone.at(2+int(pl*4)), c, 0.25) // just under the surface
	case pebShadow:
		c = scaleCol(c, 0.85)
	}
	return c
}

// fallPixel is a waterfall, at (fx, fy) within its tile: water pouring down
// in streaks, foaming at the foot.
func fallPixel(hx, hy int, fx, fy float64) color.RGBA {
	s := vnoise(float64(hx)/2.6, float64(hy)/18, 973)
	c := fallRamp.pick(0.35+0.6*s, hx, hy)
	if fy > 0.72 {
		c = mix(c, colFoam, clamp01((fy-0.72)*3.5)) // foam at the foot
	}
	if fy < 0.1 {
		c = riverRamp.pick(0.6, hx, hy) // the lip
	}
	_ = fx
	return c
}

// --- cliffs ------------------------------------------------------------------------

// cliffRamp is the rock of a cliff whose high ground is kind k.
func cliffRampOf(k sim.Tile) ramp {
	switch k {
	case sim.TileSnow, sim.TileRocky:
		return cliffStone
	case sim.TileDesert:
		return cliffSand
	case sim.TileAsh:
		return cliffAsh
	}
	return cliffEarth
}

// Cliffs are drawn along the contours of the levels, not tile by tile: the
// levels are eased between tile centres and warped a little, and where a
// level steps down past a cliff tile the step is a cliff. A step down to
// the south shows its face, hanging from the high ground's lip; a step down
// any other way shows only the lip, a dark line where the ground falls
// away. Steps with no cliff are slopes, shaded by reliefShade.

const (
	cliffFaceH = 0.8  // tiles: how tall a cliff's face stands
	cliffWarp  = 0.28 // tiles: how far a contour wanders from the tiles' edges
)

// contourAt is the eased level at (u, v), warped so contours wander.
func (t *terrain) contourAt(u, v float64) float64 {
	wu := u + (vnoise(u*1.1, v*1.3, 992)-0.5)*2*cliffWarp
	wv := v + (vnoise(u*1.3+7, v*1.1, 993)-0.5)*2*cliffWarp
	return t.levelAt(wu, wv)
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

// cliffPixel colours a pixel of a cliff's face, if (u, v) is on one.
func (t *terrain) cliffPixel(hx, hy int, u, v float64) (color.RGBA, bool) {
	if t.relief == nil {
		return color.RGBA{}, false
	}
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	if !t.has(tx, ty, featCliff) {
		return color.RGBA{}, false
	}
	here := math.Round(t.contourAt(u, v))
	// How far up the step is, if there is one within a face's height.
	const steps = 8
	d := -1.0
	for k := 1; k <= steps; k++ {
		dv := cliffFaceH * float64(k) / steps
		if math.Round(t.contourAt(u, v-dv)) > here {
			d = dv
			break
		}
	}
	if d < 0 {
		return color.RGBA{}, false
	}
	// Only a step at a cliff is a face; a slope or a ramp is not.
	top := int(math.Floor(v - d))
	if t.cliffness(u, v-d-0.2) < 0.45+0.3*(vnoise(u*1.7, v*0.6, 996)-0.5) {
		return color.RGBA{}, false
	}
	f := 1 - d/cliffFaceH // 1 just under the lip, 0 at the foot
	rp := cliffRampOf(t.cliffGround(t.nearestCliff(tx, top)))
	l := 0.16 + 0.66*f
	if dd := vnoise(float64(hx)*0.03, float64(hy)*0.3, 974) - 0.5; math.Abs(dd) < 0.05 {
		l -= 0.16 // a bedding plane
	}
	l += (vnoise(float64(hx)*0.22, float64(hy)*0.05, 975) - 0.5) * 0.3 // weathered streaks
	if hashf(fdiv(hx, 4), fdiv(hy, 11), 976) < 0.06 {
		l -= 0.2 // a crack
	}
	if f > 0.93 {
		l += 0.25 // the lip, catching the light
	}
	return rp.pick(l, hx, hy), true
}

// nearestCliff is the cliff tile at or beside (tx, ty).
func (t *terrain) nearestCliff(tx, ty int) (int, int) {
	for _, d := range [9][2]int{{0, 0}, {0, -1}, {-1, 0}, {1, 0}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		if t.tile(tx+d[0], ty+d[1]) == sim.TileCliff {
			return tx + d[0], ty + d[1]
		}
	}
	return tx, ty
}

// cliffLight darkens land pixel c at (u, v) along a cliff's lip where the
// ground falls away other than to the south, and in the shadow at the
// foot of a face.
func (t *terrain) cliffLight(c color.RGBA, u, v float64) color.RGBA {
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	if !t.has(tx, ty, featCliff) && !t.has(tx, ty-1, featCliff) {
		return c
	}
	here := math.Round(t.contourAt(u, v))
	k := 1.0
	for _, s := range [3][2]float64{{0, -0.13}, {-0.12, 0}, {0.12, 0}} {
		if math.Round(t.contourAt(u+s[0], v+s[1])) < here {
			k = min(k, 1-0.52*t.cliffness(u, v)) // the lip: beyond it, the drop
		}
	}
	// Below a face, its shadow.
	for _, dv := range []float64{cliffFaceH + 0.2, cliffFaceH + 0.5, cliffFaceH + 0.8} {
		if math.Round(t.contourAt(u, v-dv)) > here {
			k = min(k, 1-0.38*(1-(dv-cliffFaceH)/0.9)*t.cliffness(u, v-dv-0.2))
			break
		}
	}
	return scaleCol(c, k)
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
		gx = t.levelAt(u+0.45, v) - t.levelAt(u-0.45, v)
		gy = t.levelAt(u, v+0.45) - t.levelAt(u, v-0.45)
	}
	c = scaleCol(c, 1+0.07*lv+0.16*(gx+gy))
	c = t.cliffLight(c, u, v)
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
	if a := at(sim.VarDunes); a > 0 {
		s := math.Sin((float64(hx)*0.35+float64(hy)*0.9)/3.4 + 5*vnoise(float64(hx)/80, float64(hy)/60, 983))
		c = scaleCol(c, 1+0.08*s*a)
	}
	if a := at(sim.VarScree); a > 0 {
		switch p, pl := pebble(hx, hy, 8, 7, 984, 0.45*a, 0.9, 2.2); p {
		case pebStone:
			c = cliffStone.at(2 + int(pl*4))
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
			c = cliffStone.at(2 + int(pl*4))
		case pebShadow:
			c = scaleCol(c, 0.72)
		}
	}
	if a := at(sim.VarRamp); a > 0 {
		c = mix(c, pathRamp.at(3), 0.6*a)
		if s := a - 0.45; s > 0 {
			// The steps worn into it, wavering.
			if m := math.Mod(v*2+0.15*vnoise(u*3, v, 997), 1); m < 0.12 {
				c = scaleCol(c, 1-0.45*s)
			} else if m < 0.22 {
				c = scaleCol(c, 1+0.2*s)
			}
		}
	}
	if k := t.tile(tx, ty); wet > 0 && (k == sim.TileGrass || k == sim.TileForest || k == sim.TileMarsh) {
		c = mix(c, hex(0x2b5a30), 0.16*wet) // lusher by the water
	}
	return c
}

// --- the map ------------------------------------------------------------------------

// mapRelief lays rivers, cliffs, ramps and the shading of the levels over
// map land pixel c at (u, v).
func (t *terrain) mapRelief(c color.RGBA, u, v float64) color.RGBA {
	if t.relief == nil {
		return c
	}
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	switch k := t.tile(tx, ty); {
	case k == sim.TileCliff:
		return mapCliff
	case k == sim.TileWaterfall:
		return fallRamp.at(3)
	case runningWater(k):
		return mapRiver
	case t.rel(tx, ty).Variant() == sim.VarRamp:
		c = mix(c, pathRamp.at(3), 0.6)
	case k == sim.TileWater && t.rel(tx, ty).Variant() == sim.VarLake:
		return lakeRamp.at(2)
	}
	lv := t.levelAt(u, v)
	gx := t.levelAt(u+0.5, v) - t.levelAt(u-0.5, v)
	gy := t.levelAt(u, v+0.5) - t.levelAt(u, v-0.5)
	return scaleCol(c, 1+0.07*lv+0.25*(gx+gy))
}

// mapShallow tints map sea pixel c at (u, v) over the shallows.
func (t *terrain) mapShallow(c color.RGBA, u, v float64) color.RGBA {
	if t.relief == nil {
		return c
	}
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	switch {
	case t.tile(tx, ty) == sim.TileCliff:
		return mapCliff // a sea stack
	case t.tile(tx, ty) == sim.TileWater && t.rel(tx, ty).Variant() == sim.VarLake:
		return lakeRamp.at(2)
	case t.tile(tx, ty) != sim.TileShallow:
		return c
	}
	tint := shallowTint
	switch t.rel(tx, ty).Variant() {
	case sim.VarReef:
		tint = mix(shallowTint, coralCols[0], 0.3)
	case sim.VarLagoon:
		tint = lagoonTint
	case sim.VarShoal:
		tint = shoalTint
	}
	return mix(c, tint, 0.65)
}

// cliffness is how much a step of the levels at (u, v) is a cliff's: 1 by
// a cliff, 0 on a slope or by a ramp cut through one, eased between tiles
// and warped like the contours, so a cliff's face ends raggedly.
func (t *terrain) cliffness(u, v float64) float64 {
	wu := u + (vnoise(u*1.1, v*1.3, 992)-0.5)*2*cliffWarp
	wv := v + (vnoise(u*1.3+7, v*1.1, 993)-0.5)*2*cliffWarp
	cs := cornersAt(wu, wv)
	cliff := cs.share(func(x, y int) bool { return t.tile(x, y) == sim.TileCliff })
	ramp := cs.share(func(x, y int) bool { return t.tile(x, y).Walkable() && t.rel(x, y).Variant() == sim.VarRamp })
	return clamp01(cliff*2.4 - ramp*3)
}
