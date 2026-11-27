package client

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The cliffs of worlds made since sim.StyleIsles, as the ground bake draws
// them.
//
// A cliff tile is the edge of high ground, and its wall is drawn inside it:
// the lip along the side of the high ground, the foot along the side of the
// low ground. So the wall covers no ground anyone walks on, and whatever
// stands in front of it (a player at its foot, a tree) is drawn over it, as
// it should be; nothing ever stands behind it. A wall one level high is a
// tile tall; two cliff tiles stacked make a wall twice as tall.
//
// Before a chunk is baked its walls are found, for the chunk and a margin
// round it (wallField): which pixels are wall, which open ground, and the
// level of each. Two distance fields over that grid tell every wall pixel
// how far it is from the high ground and from the low ground beside it: the
// first is how far down the face it is, the two together how tall the face
// is there, and which way the first grows is which way the face looks.
// A face looking south (towards the viewer) shows courses of chunky stones,
// each with a lit top and a darker side, crisp cracks between them, moss in
// the cracks, under an overhang of the ground above and over a dark shadow
// at its foot. Looking east or west it shows the same stones from the side,
// and looking north only their tops along the rim. The rock is the high
// ground's: grey granite on grass, sandstone and red mesa in the desert,
// black basalt on the ash, icy blue-grey under snow.

const (
	wallMX   = 100  // pixels of margin, either side of a chunk, its walls are found in
	wallMY   = 116  // and above and below it
	wallWarp = 0.11 // tiles: how far a wall's edges wander off the tiles' edges
	wallFoot = 0.5  // tiles: how far a front face reaches down over the low ground below its tile
	wallLift = 0.2  // and up over the edge of the high ground above it
	wallBand = 14   // pixels down a tile of low ground a figure's feet may come: the face below is drawn over them (fronts)
	wallFar  = 0xfff0
)

// What a pixel of the wall grid is.
const (
	wallOpen uint8 = iota
	wallRock
	wallFall
)

// wallField is where the walls lie round a chunk being baked.
type wallField struct {
	w, h int     // the grid: the chunk and its margins
	kind []uint8 // wallOpen, wallRock or wallFall
	lvl  []uint8 // the level of the wall, or of the open ground (a ramp's counts one lower)
	mat  []uint8 // a wall pixel's rock (rockMats)
	// For each level of wall in the grid: the chamfer distance (3 to a
	// pixel) from every pixel to the nearest open ground at or above that
	// level (the high ground the wall is the edge of), and below it.
	top, low [sim.MaxLevel + 1][]uint16
	lowLvl   [sim.MaxLevel + 1][]uint8 // the level of the low ground nearest each pixel
	ext      []bool                    // the face reaching down over low ground below its tile (raise)
	// For each pixel, how far up and down its column the kind lasts
	// (wall: to the open ground; open: to a wall), capped.
	up, down []uint8
}

// isWallTile reports whether k is drawn as a wall.
func isWallTile(k sim.Tile) bool { return k == sim.TileCliff || k == sim.TileWaterfall }

// wallLevel is the level tile (tx, ty) counts as next to a wall: a ramp
// one lower than it is, so the walls either side of it show their ends.
func (t *terrain) wallLevel(tx, ty int) int {
	r := t.rel(tx, ty)
	l := r.Level()
	if r.Variant() == sim.VarRamp && t.tile(tx, ty).Walkable() && l > 0 {
		l--
	}
	return l
}

// wallTiles is how much each tile round a chunk counts as wall: a cliff or
// a waterfall wholly, and half a tile in the crook of two that meet at a
// corner, so a cliff running diagonally in steps of single tiles is drawn
// as one band rather than a chain of beads.
type wallTiles struct {
	t        *terrain
	tx0, ty0 int
	w, h     int
	ind      []float32
}

func (t *terrain) wallTilesFor(tx0, ty0, tx1, ty1 int) *wallTiles {
	g := &wallTiles{t: t, tx0: tx0, ty0: ty0, w: tx1 - tx0 + 1, h: ty1 - ty0 + 1}
	g.ind = make([]float32, g.w*g.h)
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			tx, ty := tx0+x, ty0+y
			switch {
			case isWallTile(t.tile(tx, ty)):
				g.ind[y*g.w+x] = 1
			case (isWallTile(t.tile(tx, ty-1)) || isWallTile(t.tile(tx, ty+1))) &&
				(isWallTile(t.tile(tx-1, ty)) || isWallTile(t.tile(tx+1, ty))):
				g.ind[y*g.w+x] = 0.5
			}
		}
	}
	return g
}

func (g *wallTiles) at(tx, ty int) float64 {
	x, y := tx-g.tx0, ty-g.ty0
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		if isWallTile(g.t.tile(tx, ty)) {
			return 1
		}
		return 0
	}
	return float64(g.ind[y*g.w+x])
}

// wallAt classifies point (u, v) for the wall grid: wall or open, its
// level, and for wall the tile it belongs to.
func (g *wallTiles) wallAt(u, v float64) (kind, lvl uint8, wtx, wty int) {
	t := g.t
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	near := false
	for dy := -2; dy <= 2 && !near; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if g.at(tx+dx, ty+dy) > 0 {
				near = true
				break
			}
		}
	}
	if !near {
		return wallOpen, uint8(t.wallLevel(tx, ty)), 0, 0
	}
	wu := u + (vnoise(u*1.3, v*1.7, 1201)-0.5)*2*wallWarp + (vnoise(u*4.1, v*3.3, 1203)-0.5)*0.05
	wv := v + (vnoise(u*1.7+9, v*1.3, 1202)-0.5)*2*wallWarp + (vnoise(u*3.7, v*4.3, 1204)-0.5)*0.04
	// How much of the point is wall: the tiles' shares smoothed by a cubic
	// B-spline, so walls are bands with rounded ends and corners, with a
	// little of the plain blend between tile centres, so a stack of one
	// tile still stands.
	bx, ix := bspline4(wu - 0.5)
	by, iy := bspline4(wv - 0.5)
	cs := cornersAt(wu, wv)
	share, bw, bo := 0.0, -1.0, -1.0
	var wx, wy, ox, oy int
	for j := range 4 {
		for i := range 4 {
			x, y := ix-1+i, iy-1+j
			wt := bx[i] * by[j]
			ind := g.at(x, y)
			share += 0.72 * wt * ind
			if ind == 1 {
				if wt > bw {
					bw, wx, wy = wt, x, y
				}
			} else if wt > bo {
				bo, ox, oy = wt, x, y
			}
		}
	}
	for k := range 4 {
		share += 0.28 * cs.wt[k] * g.at(cs.tx[k], cs.ty[k])
	}
	if share >= 0.47 && bw >= 0 {
		kind = wallRock
		if t.tile(wx, wy) == sim.TileWaterfall && t.waterShare(u, v) >= riverEdge-0.03 {
			kind = wallFall // as wide as the river pouring over it
		}
		return kind, uint8(t.rel(wx, wy).Level()), wx, wy
	}
	return wallOpen, uint8(t.wallLevel(ox, oy)), 0, 0
}

// bspline4 is the weights of the four samples round s (in sample spacings)
// for a cubic B-spline, and the index of the second of them.
func bspline4(s float64) ([4]float64, int) {
	i := math.Floor(s)
	f := s - i
	f2, f3 := f*f, f*f*f
	return [4]float64{(1 - 3*f + 3*f2 - f3) / 6, (3*f3 - 6*f2 + 4) / 6, (-3*f3 + 3*f2 + 3*f + 1) / 6, f3 / 6}, int(i)
}

// cliffRock is the rock of cliff tile (tx, ty): that of the ground most
// of the high land round it is, so a cliff is one rock along its length
// rather than changing with every outcrop on its top.
func (t *terrain) cliffRock(tx, ty int) uint8 {
	var n [sim.NumTiles]int
	lvl := t.rel(tx, ty).Level()
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			k := t.tile(tx+dx, ty+dy)
			if !k.Walkable() || t.rel(tx+dx, ty+dy).Level() < lvl-1 {
				continue
			}
			switch k {
			case sim.TileRocky, sim.TilePath, sim.TileFord:
				n[k]++
			default:
				n[k] += 3
			}
		}
	}
	best := sim.TileRocky
	for k := range n {
		if n[k] > n[best] {
			best = sim.Tile(k)
		}
	}
	return rockMatOf(best)
}

// walls finds the walls round the chunk, or returns nil if there are none.
func (b *bake) walls() *wallField {
	t := b.t
	tx0, ty0 := fdiv(b.x0-wallMX, hdTW), fdiv(b.y0-wallMY, hdTH)
	tx1, ty1 := fdiv(b.x0+chunkPW+wallMX, hdTW), fdiv(b.y0+chunkPH+wallMY, hdTH)
	any := false
	for ty := ty0; ty <= ty1 && !any; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			if isWallTile(t.tile(tx, ty)) {
				any = true
				break
			}
		}
	}
	if !any {
		return nil
	}
	f := &wallField{w: chunkPW + 2*wallMX, h: chunkPH + 2*wallMY}
	wt := t.wallTilesFor(tx0-3, ty0-3, tx1+3, ty1+3)
	n := f.w * f.h
	f.kind, f.lvl, f.mat = make([]uint8, n), make([]uint8, n), make([]uint8, n)
	rows := make([]int32, n) // the row of the tile each wall pixel belongs to
	matOf := map[[2]int]uint8{}
	var levels [sim.MaxLevel + 1]bool
	for y := 0; y < f.h; y++ {
		v := (float64(b.y0+y-wallMY) + 0.5) / hdTH
		for x := 0; x < f.w; x++ {
			u := (float64(b.x0+x-wallMX) + 0.5) / hdTW
			k, l, wx, wy := wt.wallAt(u, v)
			i := y*f.w + x
			f.kind[i], f.lvl[i] = k, l
			if k != wallOpen {
				levels[l] = true
				m, ok := matOf[[2]int{wx, wy}]
				if !ok {
					m = t.cliffRock(wx, wy)
					matOf[[2]int{wx, wy}] = m
				}
				f.mat[i] = m
				rows[i] = int32(wy)
			}
		}
	}
	f.raise(b, rows)
	for l, ok := range levels {
		if !ok {
			continue
		}
		L := uint8(l)
		f.top[l] = chamfer(f.w, f.h, func(i int) bool { return f.kind[i] == wallOpen && f.lvl[i] >= L })
		f.low[l], f.lowLvl[l] = chamferL(f.w, f.h, func(i int) bool { return f.kind[i] == wallOpen && f.lvl[i] < L }, f.lvl)
	}
	// Runs up and down the columns.
	f.up, f.down = make([]uint8, n), make([]uint8, n)
	for x := 0; x < f.w; x++ {
		run := 255
		for y := 0; y < f.h; y++ {
			i := y*f.w + x
			if y > 0 && (f.kind[i] == wallOpen) != (f.kind[i-f.w] == wallOpen) {
				run = 0
			}
			run = min(run+1, 255)
			f.up[i] = uint8(run)
		}
		run = 255
		for y := f.h - 1; y >= 0; y-- {
			i := y*f.w + x
			if y < f.h-1 && (f.kind[i] == wallOpen) != (f.kind[i+f.w] == wallOpen) {
				run = 0
			}
			run = min(run+1, 255)
			f.down[i] = uint8(run)
		}
	}
	return f
}

// chamferL is chamfer, and for every pixel the label of the seed nearest it.
func chamferL(w, h int, seed func(i int) bool, label []uint8) ([]uint16, []uint8) {
	d := make([]uint16, w*h)
	lb := make([]uint8, w*h)
	for i := range d {
		if seed(i) {
			d[i], lb[i] = 0, label[i]
		} else {
			d[i] = wallFar
		}
	}
	try := func(i, j int, add uint16) {
		if d[j] < wallFar && d[j]+add < d[i] {
			d[i], lb[i] = d[j]+add, lb[j]
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if x > 0 {
				try(i, i-1, 3)
			}
			if y > 0 {
				try(i, i-w, 3)
				if x > 0 {
					try(i, i-w-1, 4)
				}
				if x < w-1 {
					try(i, i-w+1, 4)
				}
			}
		}
	}
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			i := y*w + x
			if x < w-1 {
				try(i, i+1, 3)
			}
			if y < h-1 {
				try(i, i+w, 3)
				if x < w-1 {
					try(i, i+w+1, 4)
				}
				if x > 0 {
					try(i, i+w-1, 4)
				}
			}
		}
	}
	return d, lb
}

// chamfer is the 3-4 chamfer distance from every pixel of a w by h grid to
// the nearest pixel seed holds for (3 to a pixel across, 4 diagonally).
func chamfer(w, h int, seed func(i int) bool) []uint16 {
	d := make([]uint16, w*h)
	for i := range d {
		if seed(i) {
			d[i] = 0
		} else {
			d[i] = wallFar
		}
	}
	at := func(i int, add uint16) uint16 {
		if d[i] >= wallFar {
			return wallFar
		}
		return d[i] + add
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			m := d[i]
			if x > 0 {
				m = min(m, at(i-1, 3))
			}
			if y > 0 {
				m = min(m, at(i-w, 3))
				if x > 0 {
					m = min(m, at(i-w-1, 4))
				}
				if x < w-1 {
					m = min(m, at(i-w+1, 4))
				}
			}
			d[i] = m
		}
	}
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			i := y*w + x
			m := d[i]
			if x < w-1 {
				m = min(m, at(i+1, 3))
			}
			if y < h-1 {
				m = min(m, at(i+w, 3))
				if x < w-1 {
					m = min(m, at(i+w+1, 4))
				}
				if x > 0 {
					m = min(m, at(i+w-1, 4))
				}
			}
			d[i] = m
		}
	}
	return d
}

// gi is the grid index of chunk pixel (x, y).
func (f *wallField) gi(x, y int) int { return (y+wallMY)*f.w + x + wallMX }

// dist is a chamfer distance in pixels.
func dist(d uint16) float64 {
	if d >= wallFar {
		return math.Inf(1)
	}
	return float64(d) / 3
}

// wallPx is how a wall pixel lies on its wall.
type wallPx struct {
	kind   uint8
	mat    *rockMat
	d, h   float64 // pixels down from the lip, and the face's height there
	f      float64 // d / h: 0 at the lip, 1 at the foot
	gx, gy float64 // which way the face looks: away from the high ground
	stack  bool    // a sea stack or a wall with no high ground beside it
	top    bool    // a stack's top, not its face
	rubble bool    // the rough top of a band of cliff wider than its face is tall
}

// at reads the wall at chunk pixel (x, y): ok=false off the walls.
func (f *wallField) at(x, y int) (p wallPx, ok bool) {
	i := f.gi(x, y)
	if f.kind[i] == wallOpen {
		return p, false
	}
	p.kind = f.kind[i]
	p.mat = &rockMats[f.mat[i]]
	l := f.lvl[i]
	top, low := f.top[l], f.low[l]
	dt, dl := dist(top[i]), dist(low[i])
	if dt > 2.2*hdTH || top == nil {
		// No high ground beside it: a stack standing on its own. Its top is
		// the upper part of it, its face the rest.
		p.stack = true
		up, down := float64(f.up[i]), float64(f.down[i])
		hh := up + down
		th := min(0.42*hh, 0.5*hdTH)
		if up < th {
			p.top, p.d, p.h, p.f = true, up, th, up/th
			return p, true
		}
		p.d, p.h = up-th, hh-th
		p.f = clamp01(p.d / p.h)
		p.gy = 1
		return p, true
	}
	// Which way the face looks: away from the high ground and towards
	// the low.
	const s = 4
	diff := func(a, b int) float64 {
		a, b = max(0, min(len(top)-1, a)), max(0, min(len(top)-1, b))
		return float64(int(top[a])-int(top[b])) - float64(int(low[a])-int(low[b]))
	}
	gx, gy := diff(i+s, i-s), diff(i+s*f.w, i-s*f.w)
	if n := math.Hypot(gx, gy); n > 0 {
		p.gx, p.gy = gx/n, gy/n
	} else {
		p.gy = 1
	}
	// The face is as tall as the levels it drops; where the band of cliff
	// is wider than that, the rest of it is rough ground on top.
	drop := max(1, int(l)-int(f.lowLvl[l][i]))
	face := (float64(drop) + wallFoot + wallLift) * hdTH * (1.02 + 0.3*math.Abs(p.gx))
	p.d, p.h = dt, dt+dl
	if p.h > face*1.12 {
		if dt < p.h-face {
			p.rubble, p.d = true, dt
			return p, true
		}
		p.d, p.h = dt-(p.h-face), face
	}
	p.f = clamp01(p.d / max(1, p.h))
	return p, true
}

// --- the rock ----------------------------------------------------------------------

// rockMat is the rock a wall is built of, and how its stones lie.
type rockMat struct {
	ramp    ramp       // the crack's black to the lit top's highlight
	strata  []ramp     // the courses in these rocks by turns instead, if set: layered rock
	course  [2]float64 // pixels: how tall a course of stones is
	width   float64    // pixels: how far apart the stones of a course are
	squash  float64    // how much wider than tall a stone is drawn (its cells' metric)
	pillow  float64    // pixels in from its edge a stone rounds over
	topFrac float64    // how much of a stone, from its top, is its lit top
	moss    ramp       // moss on the tops of the stones, if any
	mossy   float64    // how much
	snow    bool       // snow lies on the tops of the stones
	lip     float64    // pixels the ground above hangs over the face, at most
	hang    bool       // grass hangs down over the lip in blades
	shadow  color.RGBA // the shadow at the wall's foot
	grain   float64    // how many specks of a lighter grain
	lift    float64    // tones lighter than the rest
}

const (
	matGranite uint8 = iota
	matMossy
	matSandstone
	matBasalt
	matIce
	matPale
)

var rockMats = []rockMat{
	matGranite: {
		ramp:   mkRamp(0x15161c, 0x2b2e37, 0x40454f, 0x575d68, 0x6f7681, 0x89909a, 0xa6abb1, 0xcbcecd),
		course: [2]float64{27, 37}, width: 60, squash: 1.35, pillow: 11, topFrac: 0.3,
		moss: mkRamp(0x3b5a22, 0x55772b, 0x759737), mossy: 0.35, lip: 8, hang: true,
		shadow: hex(0x12141a), grain: 0.012,
	},
	matMossy: {
		ramp:   mkRamp(0x121511, 0x262c24, 0x384035, 0x4b5545, 0x606b58, 0x77826b, 0x909a81, 0xadb59a),
		course: [2]float64{25, 35}, width: 56, squash: 1.35, pillow: 11, topFrac: 0.3,
		moss: mkRamp(0x2f4a1b, 0x456b22, 0x62892e), mossy: 0.8, lip: 7, hang: true,
		shadow: hex(0x0f130e), grain: 0.015,
	},
	matSandstone: {
		ramp: mkRamp(0x3a1a12, 0x6a2f1d, 0x8c4227, 0xa95732, 0xc06f3f, 0xd38a50, 0xe3a766, 0xf2c98a),
		strata: []ramp{
			mkRamp(0x3a1a12, 0x6a2f1d, 0x8c4227, 0xa95732, 0xc06f3f, 0xd38a50, 0xe3a766, 0xf2c98a),
			mkRamp(0x331410, 0x5a2219, 0x772d1f, 0x8f3a25, 0xa5492d, 0xb95e38, 0xcd7b4b, 0xe09d64),
			mkRamp(0x3d2416, 0x6e4426, 0x8f5c31, 0xab743e, 0xc18d50, 0xd3a566, 0xe2bd80, 0xf1d8a4),
		},
		course: [2]float64{19, 27}, width: 72, squash: 2.3, pillow: 7, topFrac: 0.26,
		lip: 5, shadow: hex(0x3a1c12), grain: 0.025,
	},
	matBasalt: {
		ramp:   mkRamp(0x09080c, 0x16141b, 0x221f28, 0x2f2c36, 0x3d3945, 0x4d4856, 0x625c6b, 0x7e7887),
		course: [2]float64{60, 90}, width: 20, squash: 0.45, pillow: 4, topFrac: 0.1,
		lip: 3, shadow: hex(0x08070a), grain: 0.03,
	},
	matIce: {
		ramp:   mkRamp(0x18202c, 0x2d3a49, 0x435565, 0x5b7283, 0x758ea0, 0x95abba, 0xb8c9d4, 0xdfe9ef),
		course: [2]float64{27, 37}, width: 58, squash: 1.35, pillow: 11, topFrac: 0.28,
		snow: true, lip: 7, shadow: hex(0x26303e), grain: 0.02, lift: 0.7,
	},
	matPale: {
		ramp:   mkRamp(0x3b3027, 0x615244, 0x7e6e5a, 0x988770, 0xb19f83, 0xc8b798, 0xdccfb1, 0xefe5cd),
		course: [2]float64{20, 28}, width: 64, squash: 2.0, pillow: 7, topFrac: 0.28,
		lip: 5, shadow: hex(0x3d3228), grain: 0.03,
	},
}

// rockMatOf is the rock of a cliff whose high ground is kind k.
func rockMatOf(k sim.Tile) uint8 {
	switch k {
	case sim.TileDesert:
		return matSandstone
	case sim.TileAsh, sim.TileLava:
		return matBasalt
	case sim.TileSnow, sim.TileIce:
		return matIce
	case sim.TileMarsh, sim.TileForest:
		return matMossy
	case sim.TileSand:
		return matPale
	}
	return matGranite
}

// stone is where a point of a wall lies among its stones.
type stone struct {
	id     uint32
	course int
	x, y   float64 // pixels from the stone's middle
	hh     float64 // its half height, roughly
	edge   float64 // pixels to the crack round it
	nx, ny float64 // which way its surface leans here, towards its edge
}

// stoneAt finds the stone at (a, d) of a wall of rock m: a pixels along it,
// d down from its lip. The stones are the cells round points laid in
// courses down the wall, wider than tall, so they lie in rough rows like a
// dry-stone wall or a heap of boulders, each its own shape.
func stoneAt(m *rockMat, a, d float64, seed uint32) stone {
	k := courseOf(m, d, seed)
	type pt struct {
		x, y float64
		id   uint32
		c    int
		hh   float64
	}
	var p1, p2 pt
	f1, f2 := math.Inf(1), math.Inf(1)
	sq := m.squash
	for c := k - 1; c <= k+1; c++ {
		ctop, cbot := courseTop(m, c, seed), courseTop(m, c+1, seed)
		hc := cbot - ctop
		off := hashf(c, 0, seed+11) * m.width
		jc := int(math.Floor((a + off) / m.width))
		for j := jc - 2; j <= jc+2; j++ {
			h := hash2(int32(c), int32(j), seed+12)
			if h>>24 < 52 && c >= 0 {
				continue // no stone here: its neighbours grow into a boulder
			}
			px := (float64(j)+0.12+0.76*float64(h&0xffff)/65535)*m.width - off
			py := ctop + hc*(0.36+0.3*float64(h>>16&0xff)/255)
			dx, dy := a-px, (d-py)*sq
			dd := dx*dx + dy*dy
			p := pt{px, py, h, c, hc / 2}
			switch {
			case dd < f1:
				f2, p2 = f1, p1
				f1, p1 = dd, p
			case dd < f2:
				f2, p2 = dd, p
			}
		}
	}
	st := stone{id: p1.id, course: p1.c, x: a - p1.x, y: d - p1.y, hh: p1.hh}
	// How far it is to the border with the next stone, in pixels.
	bx, by := p2.x-p1.x, (p2.y-p1.y)*sq
	if n := math.Hypot(bx, by); n > 0 {
		st.edge = (f2 - f1) / (2 * n) / math.Hypot(bx/n, by/n*sq)
	}
	if r := math.Hypot(st.x, st.y*sq); r > 0 {
		st.nx, st.ny = st.x/r, st.y*sq/r
	}
	return st
}

// courseTop is where course k of a wall of rock m begins, pixels down from
// the lip: the first right at it, the rest a course's height apart, give or
// take.
func courseTop(m *rockMat, k int, seed uint32) float64 {
	if k == 0 {
		return 0
	}
	avg := (m.course[0] + m.course[1]) / 2
	return float64(k)*avg + (hashf(k, 7, seed+10)-0.5)*(m.course[1]-m.course[0])
}

// courseOf is the course d pixels down a wall of rock m lies in.
func courseOf(m *rockMat, d float64, seed uint32) int {
	k := int(math.Floor(d / ((m.course[0] + m.course[1]) / 2)))
	for d < courseTop(m, k, seed) {
		k--
	}
	for d >= courseTop(m, k+1, seed) {
		k++
	}
	return k
}

// wallPixel colours a pixel of a wall: rock, or the overhang of the ground
// above it (ground is that ground's colour here). A front face is a stack
// of big rounded stones in rough courses: each lit on its top and along its
// upper left, its front in half shade, darker round its lower right where
// it tucks into the crack under it; the cracks crisp and dark, with moss
// and small plants in them here and there; the ground above hanging over
// the top in a lip, grass dangling from it; the whole face darker towards
// its foot.
func (b *bake) wallPixel(p wallPx, hx, hy int, u, v float64, ground func() color.RGBA) color.RGBA {
	m := p.mat
	if p.top {
		return stackTop(m, hx, hy, p)
	}
	if p.kind == wallFall {
		return fallPixel(hx, hy, p)
	}
	if p.rubble {
		return rubbleTop(m, hx, hy, ground())
	}
	// A wall seen from the side (looking east or west) or from behind
	// (north) is narrower than the tile it stands in: the high ground runs
	// on over the first part of it, as it would seen from the south and
	// above.
	d, f := p.d, p.f
	if !p.stack {
		side := p.gx*p.gx*0.22 + max(0, -p.gy)*max(0, -p.gy)*0.36
		if f < side {
			g := ground()
			if f > side-0.045 {
				return lipEdge(m, g, hx, hy) // the edge of the high ground, in the light
			}
			return g
		}
		d = p.d - side*p.h
		f = (f - side) / (1 - side)
	}
	// A face looking south or north lays its courses along its lip; one
	// looking east or west is a heap of the same stones seen from the
	// south, in rows across the screen.
	front := p.gy >= 0.55 || p.stack
	a, sd := float64(hx), d
	if !front {
		sd = float64(hy) + 7919
	}
	// The lip: the ground above hangs over the face, ragged, and grass
	// dangles from it.
	lw := 0.0
	if !p.stack {
		lw = m.lip * (0.45 + 0.75*vnoise(a/11, float64(hy)/40, 1211))
		if m.snow {
			lw += 2 + 3*vnoise(a/6, 3, 1214) // a cornice of snow
		}
		if d < lw {
			g := ground()
			switch {
			case d > lw-1.4:
				return scaleCol(g, 0.42) // its underside
			case d > lw-3.4:
				return lipEdge(m, g, hx, hy)
			}
			return scaleCol(g, 1.08)
		}
		if m.hang && front {
			if bl := hashf(hx, 1, 1212); bl < 0.3 {
				if l := 2 + 9*hashf(hx, 2, 1213); d < lw+l {
					k := (d - lw) / l
					g := ground()
					if hashf(hx, 3, 1215) < 0.5 {
						return scaleCol(g, 0.62+0.3*k) // a blade in the shade of the lip
					}
					return scaleCol(g, 0.85+0.2*(1-k))
				}
			}
		}
	}
	dl := d - lw // pixels under the lip
	if !p.stack && math.Abs(p.gy-0.55) < 0.03 && dl > 2 {
		return m.ramp.at(0) // where the stones laid along the lip meet those laid in rows, a joint
	}
	st := blockAt(m, a, sd, 1220)
	rp := m.ramp
	if m.strata != nil {
		rp = m.strata[(st.course%len(m.strata)+len(m.strata))%len(m.strata)]
	}
	// Plants and moss growing out of the cracks.
	if m.hang || m.mossy > 0 {
		if c, ok := crackPlant(m, a, sd, hx, hy); ok {
			return c
		}
	}
	// The crack between stones, dark; at the foot the ground shows between
	// the rounded bottoms of the stones.
	foot := !p.stack && f > 0.86
	if st.edge < 0.35 {
		if foot {
			return scaleCol(ground(), 0.4)
		}
		if m.mossy > 0 && st.ny > 0.4 && vnoise(float64(hx)/6, float64(hy)/5, 1234) < m.mossy*0.55 {
			return m.moss.at(0) // moss in the crack, under a stone
		}
		if st.edge < 0 || st.nx+st.ny > 0 {
			return rp.at(0)
		}
		return rp.at(1)
	}
	if foot && f > 0.9 && st.edge < 0.35+(f-0.9)*40 && st.ny > 0.3 {
		return scaleCol(ground(), 0.4)
	}
	// The stone: rounded over towards its edge, a flat top facing the sky,
	// lit from the upper left.
	rim := 1 - smooth(clamp01(st.edge/m.pillow))
	ty := st.y + st.hh // pixels down from the stone's top, roughly
	topH := m.topFrac*2*st.hh - 2.4*(st.x/(m.width*0.6))*(st.x/(m.width*0.6))
	onTop := ty < topH
	// The top, facing the sky, eases into the front over a few pixels.
	up := smooth(clamp01((topH - ty + 1.5) / 3))
	l := lerpf(lambert(st.nx*rim, st.ny*rim, 1.05-0.75*rim), lambert(st.nx*rim*0.5, -0.8, 0.5), up)
	tone := 0.9 + 5.4*l + m.lift + (hashf(int(st.id), 0, 1223)-0.5)*1.0
	switch {
	case !onTop && st.edge < 3.2 && st.ny > 0.15:
		tone -= 1.1 * (3.2 - st.edge) / 3.2 // tucked into the crack below
	}
	if onTop {
		if m.snow && (st.course <= 0 || hashf(int(st.id), 2, 1233) < 0.4) && ty < topH*0.8 {
			return snowCap(hx, hy, ty)
		}
		if m.mossy > 0 && vnoise(float64(hx)/7, float64(hy)/5, 1231) < m.mossy*0.7 {
			return m.moss.at(int(clamp01(0.3+0.6*l+0.2*hashf(hx, hy, 1232)) * 2.99))
		}
	}
	// The whole face darker towards its foot and just under the lip, and
	// lit by which way it looks.
	tone -= 1.7 * f * f
	if dl < 7 && !p.stack {
		tone -= 1.7 * (1 - dl/7)
	}
	switch {
	case p.gy < -0.6:
		tone += 0.5 // the rim of a wall looking north: the stones' tops
	case p.gx > 0.55:
		tone -= 0.7 // looking east, out of the light
	case p.gx < -0.55:
		tone += 0.3
	}
	if hashf(hx, hy, 1225) < m.grain {
		tone += 0.9
	}
	// On the ramp three times as fine, so the stone rounds smoothly.
	fr := fineRock(rp)
	i := int(math.Round(tone*3 + 0.8*(dither(hx, hy)-0.5)))
	return fr.at(max(3, min(len(fr)-1, i)))
}

// fineRocks are the rocks' ramps three times as fine, made once.
var fineRocks sync.Map // the first colour of a ramp -> its fine ramp

// fineRock is ramp r three times as fine.
func fineRock(r ramp) ramp {
	key := [2]color.RGBA{r[0], r[len(r)-1]}
	if v, ok := fineRocks.Load(key); ok {
		return v.(ramp)
	}
	f := r.sub(3)
	fineRocks.Store(key, f)
	return f
}

// lipEdge is the edge of the high ground where it meets the top of a wall,
// lit: grass brighter, sand paler, snow white.
func lipEdge(m *rockMat, g color.RGBA, hx, hy int) color.RGBA {
	if m.snow {
		return mix(g, hex(0xfbfdff), 0.6)
	}
	return scaleCol(g, 1.22+0.08*hashf(hx, hy, 1216))
}

// crackPlant colours (hx, hy), at (a, d) on a wall of rock m, if it is on
// one of the small plants growing out of the cracks along the top of a
// course of stones: a fan of a few blades rising from the crack and arching
// over, or a cushion of moss on the stone below.
func crackPlant(m *rockMat, a, d float64, hx, hy int) (color.RGBA, bool) {
	const cw = 26.0
	k := courseOf(m, d, 1220)
	for c := k; c <= k+1; c++ { // the crack at the top of this course, or the next
		y0 := courseTop(m, c, 1220)
		if c <= 0 {
			continue // the lip is no crack
		}
		jc := int(math.Floor(a / cw))
		for j := jc - 1; j <= jc+1; j++ {
			h := hash2(int32(c), int32(j), 1236)
			if float64(h&0xff)/255 > 0.16 {
				continue
			}
			rx := (float64(j) + 0.2 + 0.6*float64(h>>8&0xff)/255) * cw
			dx, dy := a-rx, d-y0
			if h>>16&3 == 0 && m.mossy > 0 {
				// A cushion of moss over the crack.
				if r := dx*dx/36 + (dy+0.5)*(dy+0.5)/6; r < 1 {
					return m.moss.at(int(clamp01(0.75-0.5*r-0.12*dy+0.2*hashf(hx, hy, 1237)) * 2.99)), true
				}
				continue
			}
			// A fan of blades up out of the crack.
			n := 3 + int(h>>18&3)
			for i := 0; i < n; i++ {
				ang := (float64(i)/float64(n-1) - 0.5) * 2.0
				ln := 4 + 5*hashf(int(h), i, 1238)
				t := -dy / ln // how far up the blade
				if t < 0 || t > 1 {
					continue
				}
				x := ang*ln*0.7*t + ang*ln*0.35*t*t // arching out
				if math.Abs(dx-x) < 0.75-0.35*t {
					g := grassHD.at(int(clamp01(0.3+0.6*t-0.25*ang) * float64(len(grassHD)-1)))
					return g, true
				}
			}
		}
	}
	return color.RGBA{}, false
}

// rubbleTop is the rough top of a band of cliff wider than its face is
// tall: the ground above it, strewn with boulders of its rock.
func rubbleTop(m *rockMat, hx, hy int, g color.RGBA) color.RGBA {
	// The ground's grass, sand or snow lies on it in patches.
	if n := vnoise(float64(hx)/16, float64(hy)/12, 1252)*0.8 + 0.2*hashf(fdiv(hx, 2), fdiv(hy, 2), 1254); n > 0.56 {
		if n < 0.59 {
			return scaleCol(g, 0.78) // its edge, thin over the rock
		}
		return scaleCol(g, 0.96)
	}
	// The rock, broken into slabs, each lit along its upper left.
	edge, id, nx, ny := plates(float64(hx)/30, float64(hy)/22, 1253)
	if edge < 0.045 {
		return m.ramp.at(1)
	}
	r := clamp01((0.22 - edge) / 0.22)
	l := 4.5 + m.lift + (float64(id%97)/97-0.5)*1.2 - r*(nx+ny) // the rim to the lower right in shade
	fr := fineRock(m.ramp)
	return fr.at(max(3, min(len(fr)-1, int(math.Round(l*3+0.8*(dither(hx, hy)-0.5))))))
}

// snowCap is the snow lying on the top of a stone, ty pixels down it.
func snowCap(hx, hy int, ty float64) color.RGBA {
	if ty < 1.5 {
		return hex(0xfbfdff)
	}
	return snowHD.g(0.72+0.2*vnoise(float64(hx)/4, float64(hy)/3, 1227), hx, hy)
}

// stackTop is the top of a stack standing in the sea: rock with a tuft of
// green on it.
func stackTop(m *rockMat, hx, hy int, p wallPx) color.RGBA {
	if p.d < 1.2 {
		return m.ramp.at(1)
	}
	if vnoise(float64(hx)/7, float64(hy)/5, 1228) > 0.55 && m != &rockMats[matBasalt] {
		return grassHD.g(0.45+0.4*vnoise(float64(hx)/3, float64(hy)/3, 1229), hx, hy)
	}
	l := 5.2 - 1.5*p.f + (vnoise(float64(hx)/5, float64(hy)/4, 1230)-0.5)*1.4
	return m.ramp.at(int(math.Round(l)))
}

// wallShade darkens open ground pixel (x, y) of the chunk at the foot of a
// wall higher than it: a crisp dark band right under the stones, and the
// shadow the wall throws away from the sun (down and to the right).
func (f *wallField) shade(c color.RGBA, x, y, hx, hy int) color.RGBA {
	i := f.gi(x, y)
	l := f.lvl[i]
	if l >= sim.MaxLevel {
		return c
	}
	higher := func(j int) bool { return f.kind[j] == wallRock && f.lvl[j] > l }
	// Right under a face.
	k := 0.0
	if up := int(f.up[i]); up < 8 && y+wallMY-up >= 0 {
		if j := i - up*f.w; higher(j) {
			switch {
			case up <= 2:
				k = 0.58
			case up <= 4:
				k = 0.48
			default:
				k = 0.4
			}
		}
	}
	// Cast towards the lower right, as long as the wall is tall.
	if k < 0.36 {
		const reach = 24
		for s := 2; s <= reach; s += 2 {
			dx, dy := int(float64(s)*0.5), int(float64(s)*0.87)
			if x+wallMX-dx < 0 || y+wallMY-dy < 0 {
				break
			}
			if higher(i - dy*f.w - dx) {
				edge := float64(s) / reach
				if edge < 0.85 || dither(hx, hy) > (edge-0.85)*6.6 {
					k = max(k, 0.36)
				}
				break
			}
		}
	}
	if k == 0 {
		return c
	}
	m := rockMats[matGranite].shadow
	for _, d := range [4][2]int{{0, -2}, {-2, -2}, {-2, 0}, {0, -6}} {
		if j := i + d[1]*f.w + d[0]; j >= 0 && higher(j) {
			m = rockMats[f.mat[j]].shadow
			break
		}
	}
	return mix(c, m, k)
}

// seaFoot is the sea at the foot of a cliff standing in it: foam where the
// waves break on the stones, dark water in the cliff's shadow.
func (f *wallField) seaFoot(x, y, hx, hy int, s float64) (color.RGBA, bool) {
	i := f.gi(x, y)
	up := int(f.up[i])
	if f.kind[i] != wallOpen || up > 16 || y+wallMY-up < 0 || f.kind[i-up*f.w] == wallOpen {
		return color.RGBA{}, false
	}
	n := vnoise(float64(hx)/5, float64(hy)/3, 1240)
	switch {
	case float64(up) < 3+3*n:
		return withAlpha(colFoam, 235), true
	case float64(up) < 7+5*n && hashf(fdiv(hx, 2), fdiv(hy, 2), 1241) < 0.6:
		return withAlpha(colFoam, 170), true
	}
	return withAlpha(colDeep, uint8(150-float64(up)*4)), true
}

// underFall is how much open pixel (x, y) of the chunk lies in the pool at
// the foot of a waterfall: 1 right under it, fading over a tile, a little
// wider than the fall.
func (f *wallField) underFall(x, y int) float64 {
	best := 0.0
	for _, o := range [3]int{0, -9, 9} {
		xx := x + o
		if xx < -wallMX || xx >= chunkPW+wallMX {
			continue
		}
		i := f.gi(xx, y)
		up := int(f.up[i])
		if f.kind[i] != wallOpen || y+wallMY-up < 0 || f.kind[i-up*f.w] != wallFall {
			continue
		}
		a := 1 - float64(up)/(0.9*hdTH)
		if o != 0 {
			a *= 0.6
		}
		best = max(best, a)
	}
	return best
}

// raise makes the front faces of the walls stand taller than their tiles:
// each run of wall down a column that has lower ground under it reaches on
// down over that ground to wallFoot of a tile below its tile, and up over
// the last wallLift of the high ground above it. Nobody's feet come that
// far up into the high ground (a figure keeps a quarter of a tile from a
// wall); the low ground they do walk on, and the part of the face over it
// lower than wallBand is drawn again over whoever stands there (fronts), so
// they are behind the foot of the wall, never standing on its face. Ramps
// are never covered: they are the way through.
func (f *wallField) raise(b *bake, rows []int32) {
	t := b.t
	f.ext = make([]bool, len(f.kind))
	lifted := make([]bool, len(f.kind))
	for x := 0; x < f.w; x++ {
		hx := b.x0 + x - wallMX
		tx := fdiv(hx, hdTW)
		ramp := func(hy int) bool {
			ty := fdiv(hy, hdTH)
			return t.rel(tx, ty).Variant() == sim.VarRamp && t.tile(tx, ty).Walkable()
		}
		// Down, under the foot of each run of wall.
		for y := 1; y < f.h; y++ {
			i, j := y*f.w+x, (y-1)*f.w+x
			if f.kind[i] != wallOpen || f.kind[j] == wallOpen || f.ext[j] || f.lvl[i] >= f.lvl[j] {
				continue
			}
			// Half a tile below the wall's edge, whatever its shape, so the foot
			// follows it round its bends.
			foot := float64(b.y0+y-wallMY) + wallFoot*hdTH - 2*vnoise(float64(hx)/9, 0, 1260)
			for k := y; k < f.h; k++ {
				ii := k*f.w + x
				hy := b.y0 + k - wallMY
				if float64(hy)+0.5 >= foot || f.kind[ii] != wallOpen || f.lvl[ii] >= f.lvl[j] || ramp(hy) {
					break
				}
				f.kind[ii], f.lvl[ii], f.mat[ii], rows[ii], f.ext[ii] = f.kind[j], f.lvl[j], f.mat[j], rows[j], true
			}
		}
		// Up, over the edge of the high ground above each run.
		for y := f.h - 2; y >= 0; y-- {
			i, j := y*f.w+x, (y+1)*f.w+x
			if f.kind[i] != wallOpen || f.kind[j] == wallOpen || f.ext[j] || lifted[j] || f.lvl[i] < f.lvl[j] {
				continue
			}
			lip := math.Max(float64(b.y0+y+1-wallMY)-wallLift*hdTH, (float64(rows[j])-0.3)*hdTH)
			for k := y; k >= 0; k-- {
				ii := k*f.w + x
				hy := b.y0 + k - wallMY
				if float64(hy)+0.5 < lip || f.kind[ii] != wallOpen || f.lvl[ii] < f.lvl[j] || ramp(hy) {
					break
				}
				f.kind[ii], f.lvl[ii], f.mat[ii], rows[ii] = f.kind[j], f.lvl[j], f.mat[j], rows[j]
				lifted[ii] = true
			}
		}
	}
}

// front reports whether chunk pixel (x, y) is part of a face drawn again
// over the figures at its foot (see raise), and if so which row of the
// chunk's tiles it is in and how far down that row's band it lies.
func (f *wallField) front(b *bake, x, y int) (row, by int, ok bool) {
	i := f.gi(x, y)
	if !f.ext[i] || f.kind[i] == wallOpen {
		return 0, 0, false
	}
	hy := b.y0 + y
	in := hy - fdiv(hy, hdTH)*hdTH
	if in < wallBand || in >= wallBand+frontBandH {
		return 0, 0, false
	}
	return y / hdTH, in - wallBand, true
}

// frontBandH is how tall the band of a tile row redrawn over the figures
// is: from wallBand down to the deepest a face reaches.
const frontBandH = int(wallFoot*hdTH) - wallBand + 1

// belowWall reports whether point (x, y), in tiles, lies on the low ground
// a front face reaches down over (see raise): nothing small grows there.
func (g *Game) belowWall(x, y float64) bool {
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	if y-float64(ty) > wallFoot+0.06 {
		return false
	}
	k := g.tileAt(tx, ty-1)
	if !isWallTile(k) || g.relief == nil {
		return false
	}
	lvl := func(x, y int) int {
		if x < 0 || y < 0 || x >= g.width || y >= g.height || y*g.width+x >= len(g.relief) {
			return 0
		}
		return g.relief[y*g.width+x].Level()
	}
	return lvl(tx, ty) < lvl(tx, ty-1)
}

// wallStrip is a band of the foot of the cliffs' faces in one row of
// tiles of a chunk, to be drawn over the figures standing in it.
type wallStrip struct {
	img  *ebiten.Image // the band, at groundDens
	x, y float64       // its corner, in art pixels of the world
}

// wallDrawables adds to list the bands of the cliffs' feet in view (see
// raise), each sorted just before what stands in the middle of its row of
// tiles: a figure further up the row, its feet hidden by the stones of the
// foot, is drawn behind it; a bush or a tree in the row, and anyone lower
// down, in front of it.
func (g *Game) wallDrawables(list []drawable, cam camera) []drawable {
	gc := g.ground
	if gc == nil {
		return list
	}
	gc.strips = gc.strips[:0]
	for _, k := range gc.keys {
		img, rows := gc.fronts[k], gc.frows[k]
		if img == nil {
			continue
		}
		for r := 0; r < chunkTiles; r++ {
			if rows&(1<<r) == 0 {
				continue
			}
			sub := img.SubImage(image.Rect(0, r*frontBandH, chunkPW, (r+1)*frontBandH)).(*ebiten.Image)
			x := float64(k.cx * chunkW)
			y := float64(k.cy*chunkH+r*TileH) + float64(wallBand)/groundDens
			gc.strips = append(gc.strips, wallStrip{sub, x, y})
			list = append(list, drawable{y: float64(k.cy*chunkTiles+r) + 0.495, id: uint32(len(gc.strips)), wall: true,
				idx: len(gc.strips) - 1, sx: x - cam.ox, sy: y - cam.oy})
		}
	}
	return list
}

// drawWallFront draws the band of a cliff's foot d stands for.
func (g *Game) drawWallFront(dst drawTarget, d drawable) {
	s := g.ground.strips[d.idx]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(d.sx, d.sy)
	drawImageD(dst, s.img, op, groundDens)
}

// blockAt finds the stone at (a, d) of a wall of rock m, a pixels along it
// and d down from its lip: big rounded blocks laid in courses down the
// wall, each course's joints staggered from the last, each block a rounded
// box (a superellipse) a little lumpy, its own width, a little higher or
// lower than its neighbours, with a crack all round it. Outside its outline
// edge is negative: the crack.
func blockAt(m *rockMat, a, d float64, seed uint32) stone {
	k := courseOf(m, d, seed)
	y0, y1 := courseTop(m, k, seed), courseTop(m, k+1, seed)
	w := m.width
	off := hashf(k, 0, seed+11) * w
	// The joints of the course: every width, give or take.
	joint := func(j int) float64 {
		return float64(j)*w - off + (hashf(k, j, seed+13)-0.5)*w*0.55
	}
	j := int(math.Floor((a + off) / w))
	if a < joint(j) {
		j--
	} else if a >= joint(j+1) {
		j++
	}
	x0, x1 := joint(j), joint(j+1)
	h := hash2(int32(k), int32(j), seed+12)
	const gap = 1.6
	cx := (x0 + x1) / 2
	hw := (x1-x0)/2 - gap/2
	hh := (y1-y0)/2 - gap/2
	cy := (y0+y1)/2 + (float64(h>>8&0xff)/255-0.5)*hh*0.25
	hh *= 0.9 + 0.2*float64(h>>16&0xff)/255
	dx, dy := (a-cx)/hw, (d-cy)/hh
	ang := math.Atan2(dy, dx)
	lump := 1 + 0.05*math.Sin(3*ang+float64(h&0xff)) + 0.03*math.Sin(5*ang+float64(h>>4&0xff))
	const p = 3.2
	r := math.Pow(math.Pow(math.Abs(dx), p)+math.Pow(math.Abs(dy), p), 1/p) / lump
	st := stone{id: h, course: k, x: a - cx, y: d - cy, hh: hh}
	st.edge = (1 - r) * math.Min(hw, hh) * 1.1
	// Its outward normal, from the superellipse.
	nx := math.Copysign(math.Pow(math.Abs(dx), p-1), dx)
	ny := math.Copysign(math.Pow(math.Abs(dy), p-1), dy) * hw / hh
	if n := math.Hypot(nx, ny); n > 0 {
		st.nx, st.ny = nx/n, ny/n
	}
	return st
}
