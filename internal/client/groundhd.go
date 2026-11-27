package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// Baking the ground at HD density: groundDens image pixels to an art pixel
// (see hd.go). Every image pixel is coloured on its own, from the same
// smooth fields as before (the coast, the biome scores, the rock of the
// mountains), so coastlines and borders are smooth at screen resolution, and
// the look of each kind of ground (groundart.go) is drawn pixel for pixel at
// that density: blades, grains, pebbles, cracks.
//
// A chunk is baked in two steps. First the fields other pixels look up or
// down at are worked out for the whole chunk and a margin around it: the
// coast distance (a sea pixel under a south-facing shore is an earth wall)
// and the rock mask (a rock pixel near the rock's south edge is its face,
// open ground just under one lies in its shadow). Then each pixel is
// coloured.

const (
	hdTW = TileW * groundDens // a tile, in image pixels
	hdTH = TileH * groundDens

	coastUp  = (cliffH + 2) * groundDens // rows above a pixel that can make it an earth wall or its foam
	rockUp   = 3*groundDens + 1          // rows below a rock its shadow reaches
	rockDown = mtnFaceH * groundDens     // rows of rock face along a mountain's south side
)

// GroundStyle is a look of the ground.
type GroundStyle int

const (
	GroundGreen GroundStyle = iota // lush green grass and dark woods
	GroundWarm                     // ochre earth with ragged patches of olive grass, mossy woods
)

var groundStyle GroundStyle

// SetGroundStyle picks the look of the ground and keeps it with the
// settings. The ground in view is baked again in the new look.
func SetGroundStyle(s GroundStyle) {
	name := ""
	if s == GroundWarm {
		name = "warm"
	}
	changeSettings(func(st *Settings) { st.Ground = name }) // sets groundStyle: see settings.go
}

// CurrentGroundStyle is the look of the ground now.
func CurrentGroundStyle() GroundStyle { return groundStyle }

// SetWarmGround turns the warm ground on or off.
func SetWarmGround(on bool) {
	if on {
		SetGroundStyle(GroundWarm)
	} else {
		SetGroundStyle(GroundGreen)
	}
}

// fdiv is a / b rounded down, for b > 0.
func fdiv(a, b int) int {
	q := a / b
	if a%b < 0 {
		q--
	}
	return q
}

// bake is one chunk being baked.
type bake struct {
	t      *terrain
	warm   bool
	x0, y0 int // the chunk's corner, in image pixels of the world

	coast  []float32   // the signed coast distance of each pixel, from coastUp rows above the chunk
	landUp []uint8     // per pixel: pixels up to the nearest land (0 on land), 255 for none near
	rock   *rockField  // nil when no rock is near
	roofed *rockField  // the same with caves roofed over; nil when no cave is near
	wall   *wallField  // the cliffs of the newest worlds; nil when none is near (terraincliff.go)
	river  *riverField // their rivers; nil when none is near (terrainwater.go)
	reef   *reefField  // their coral; nil when no reef is near (terrainreef.go)
}

// rockField says where each pixel of a chunk lies on or by the rock.
type rockField struct {
	foot []uint8 // on rock: pixels down to the open ground at its foot, 255 when further than a face is tall; 0 off the rock
	over []uint8 // off the rock: pixels up to the rock above, 0 when there is none within rockUp
}

func (t *terrain) newBake(k chunkKey) *bake {
	b := &bake{t: t, warm: groundStyle == GroundWarm, x0: k.cx * chunkPW, y0: k.cy * chunkPH}
	b.coast = make([]float32, chunkPW*(chunkPH+coastUp))
	for y := 0; y < chunkPH+coastUp; y++ {
		v := (float64(b.y0+y-coastUp) + 0.5) / hdTH
		row := b.coast[y*chunkPW : (y+1)*chunkPW]
		for x := range row {
			row[x] = float32(t.coast((float64(b.x0+x)+0.5)/hdTW, v))
		}
	}
	b.landUp = make([]uint8, chunkPW*chunkPH)
	for x := 0; x < chunkPW; x++ {
		up := 255
		for y := 0; y < chunkPH+coastUp; y++ {
			if b.coast[y*chunkPW+x] >= 0 {
				up = 0
			} else if up < 255 {
				up++
			}
			if y >= coastUp {
				b.landUp[(y-coastUp)*chunkPW+x] = uint8(up)
			}
		}
	}
	tx0, ty0 := k.cx*chunkTiles, k.cy*chunkTiles
	rock, cave := false, false
	for ty := ty0 - 2; ty <= ty0+chunkTiles+3; ty++ {
		for tx := tx0 - 1; tx <= tx0+chunkTiles; tx++ {
			switch t.tile(tx, ty) {
			case sim.TileMountain:
				rock = true
			case sim.TileCave:
				cave = true
			}
		}
	}
	if rock {
		b.rock = b.rocks(false)
	}
	if cave {
		b.roofed = b.rocks(true)
	}
	if t.relief != nil {
		b.wall = b.walls()
		b.river = b.rivers()
		b.reef = b.reefs()
	}
	return b
}

// rocks works out the chunk's rock field: with roof, caves count as rock.
func (b *bake) rocks(roof bool) *rockField {
	rows := rockUp + chunkPH + rockDown + 1
	mask := make([]bool, chunkPW*rows)
	some := false
	for y := 0; y < rows; y++ {
		v := (float64(b.y0+y-rockUp) + 0.5) / hdTH
		for x := 0; x < chunkPW; x++ {
			if b.t.isMountain((float64(b.x0+x)+0.5)/hdTW, v, roof) {
				mask[y*chunkPW+x] = true
				some = true
			}
		}
	}
	if !some {
		return nil
	}
	f := &rockField{foot: make([]uint8, chunkPW*chunkPH), over: make([]uint8, chunkPW*chunkPH)}
	for x := 0; x < chunkPW; x++ {
		foot := 255 // below the margin: unknown, so further than a face is tall
		for y := rows - 1; y >= rockUp; y-- {
			if !mask[y*chunkPW+x] {
				foot = 0
			} else if foot < 255 {
				foot++
			}
			if y < rockUp+chunkPH {
				f.foot[(y-rockUp)*chunkPW+x] = uint8(foot)
			}
		}
		over := 0
		for y := 0; y < rockUp+chunkPH; y++ {
			switch {
			case mask[y*chunkPW+x]:
				over = -1 // on the rock
			case over == -1:
				over = 1
			case over > 0 && over < rockUp:
				over++
			default:
				over = 0
			}
			if y >= rockUp && over > 0 {
				f.over[(y-rockUp)*chunkPW+x] = uint8(over)
			}
		}
	}
	return f
}

// bakeChunk returns the chunk's ground pixels (chunkPW by chunkPH) and, if
// it has a coastline, its shore image for the surf (chunkW by chunkH), and
// if it has caves, their roof (else nil).
func (t *terrain) bakeChunk(k chunkKey) (ground, shore, roof []byte) {
	ground, shore, roof, _ = t.bakeChunkAll(k)
	return ground, shore, roof
}

// bakeChunkAll is bakeChunk, and the chunk's water map for the moving
// water drawn over it (nil when it has none; see terrainanim.go).
func (t *terrain) bakeChunkAll(k chunkKey) (ground, shore, roof, water []byte) {
	bc := t.bakeChunkFull(k)
	return bc.ground, bc.shore, bc.roof, bc.water
}

// bakedChunk is all a chunk's bake makes.
type bakedChunk struct {
	ground, shore, roof, water []byte
	front                      []byte // the feet of the cliffs' faces, drawn again over the figures by them: see fronts in terraincliff.go
	frontRows                  uint8  // which of the chunk's rows of tiles have any
}

// bakeChunkFull bakes everything of chunk k.
func (t *terrain) bakeChunkFull(k chunkKey) bakedChunk {
	if t.caveChunk(k) {
		return bakedChunk{ground: t.bakeCaveChunk(k)} // the natural caves: see cavebake.go
	}
	b := t.newBake(k)
	var wm []byte
	wet := false
	c := newCanvas(chunkPW, chunkPH)
	var rf *canvas
	var front *canvas
	rows := uint8(0)
	sh := make([]byte, chunkW*chunkH*4)
	coast := false
	for y := 0; y < chunkPH; y++ {
		v := (float64(b.y0+y) + 0.5) / hdTH
		for x := 0; x < chunkPW; x++ {
			u := (float64(b.x0+x) + 0.5) / hdTW
			col, surf := b.pixel(x, y, u, v)
			c.px[y*chunkPW+x] = col
			if b.wall != nil {
				if row, by, ok := b.wall.front(b, x, y); ok {
					if front == nil {
						front = newCanvas(chunkPW, chunkTiles*frontBandH)
					}
					front.px[(row*frontBandH+by)*chunkPW+x] = col
					rows |= 1 << row
				}
			}
			if b.roofed != nil && t.caveNear(u, v) {
				if rc, ok := b.roofPixel(x, y, u, v, col); ok {
					if rf == nil {
						rf = newCanvas(chunkPW, chunkPH)
					}
					rf.px[y*chunkPW+x] = rc
				}
			}
			// The shore image has one pixel to an art pixel: its middle.
			if surf && x%groundDens == groundDens/2 && y%groundDens == groundDens/2 {
				if sb, ok := shoreByte(float64(b.coast[(y+coastUp)*chunkPW+x])); ok {
					i := (y/groundDens*chunkW + x/groundDens) * 4
					sh[i], sh[i+3] = sb, 255
					coast = true
				}
			}
			// So has the water map.
			if t.relief != nil && x%groundDens == groundDens/2 && y%groundDens == groundDens/2 {
				if tx, ok := b.waterTexel(x, y, u, v); ok {
					if wm == nil {
						wm = make([]byte, chunkW*chunkH*4)
					}
					i := (y/groundDens*chunkW + x/groundDens) * 4
					copy(wm[i:i+4], tx[:])
					wet = true
				}
			}
		}
	}
	if !wet {
		wm = nil
	}
	if !coast {
		sh = nil
	}
	bc := bakedChunk{ground: c.bytes(), shore: sh, water: wm, frontRows: rows}
	if rf != nil {
		bc.roof = rf.bytes()
	}
	if front != nil {
		bc.front = front.bytes()
	}
	return bc
}

// pixel colours image pixel (x, y) of the chunk, at (u, v) in tiles, and
// reports whether waves may be drawn over it (not on mountains, ponds or
// the earth walls of the coast).
func (b *bake) pixel(x, y int, u, v float64) (color.RGBA, bool) {
	t := b.t
	hx, hy := b.x0+x, b.y0+y
	s := float64(b.coast[(y+coastUp)*chunkPW+x])
	if b.wall != nil { // the cliffs: see terraincliff.go
		if p, ok := b.wall.at(x, y); ok {
			return b.wallPixel(p, hx, hy, u, v, func() color.RGBA { return t.reliefShade(b.land(hx, hy, u, v), hx, hy, u, v) }), false
		}
	}
	if s >= 0 {
		if c, ok := b.rockPixel(b.rock, x, y, u, v, false); ok {
			return c, false // mountains stand above everything else
		}
		if c, ok := b.riverPixel(x, y, hx, hy, u, v); ok {
			if b.wall != nil {
				c = b.wall.shade(c, x, y, hx, hy)
			}
			return c, false // see terrainfeatures.go
		}
		c := t.reliefShade(b.land(hx, hy, u, v), hx, hy, u, v)
		if b.wall != nil {
			c = b.wall.shade(c, x, y, hx, hy)
		}
		if b.rock != nil {
			if o := b.rock.over[y*chunkPW+x]; o > 0 {
				c = mix(c, colInk, rockShadow(int(o))) // at the foot of a mountain's face
			}
		}
		if t.relief != nil { // the Megalodon's reef: coral rock awash (see atollart.go)
			wu, wv := u+(vnoise(u*1.7, v*1.9, 1510)-0.5)*0.6, v+(vnoise(u*1.9, v*1.7, 1511)-0.5)*0.6 // ragged, not by the tile
			if tx, ty := int(math.Floor(wu)), int(math.Floor(wv)); t.tile(tx, ty) == sim.TileSand && t.rel(tx, ty).Variant() == sim.VarReef {
				return reefFlat(hx, hy, u, v, s), true
			}
		}
		if s < surfLand && t.pondAt(u, v) > 0.5 { // a swamp pond's bank: wet mud, and no surf
			if s < 0.22 {
				c = swampBank(c, hx, hy, s)
			}
			return c, false
		}
		if t.relief != nil && s < 2.3 { // the beach at the waterline: see terrainsea.go
			c = t.beachShore(c, hx, hy, u, v, s)
		} else if s < 0.22 { // wet sand at the water's edge
			w := 1 - s/0.22
			c = mix(c, hex(0x6b5a3e), 0.45*w*(0.85+0.3*hashf(hx, hy, 73)))
			if s < 0.05 {
				c = mix(c, colFoam, 0.3*(1-s/0.05))
			}
		}
		return c, true
	}
	up := int(b.landUp[y*chunkPW+x])
	if t.pondAt(u, v) > 0.5 {
		if c, ok := t.lakePixel(hx, hy, u, v, s, up); ok {
			return c, false // a lake: see terrainfeatures.go
		}
		return swampWater(hx, hy, u, v, s, up), false // still, murky water: no surf
	}
	// Sea. A south-facing shore gets an earth wall hanging below it, a cliff
	// a line of foam at its foot.
	if b.wall != nil {
		if c, ok := b.wall.seaFoot(x, y, hx, hy, s); ok {
			return c, false
		}
	}
	if up <= coastUp && !t.sandAbove(u, v, up) { // a beach runs into the water: no wall
		if up <= cliffH*groundDens {
			return seaCliff(hx, hy, up), false
		}
		if hashf(fdiv(hx, 2), hy, 74) < 1.3-float64(up-cliffH*groundDens)/float64(2*groundDens) {
			return withAlpha(colFoam, 200), false // foam at the wall's foot, breaking up
		}
	}
	if t.relief != nil {
		return b.reliefSea(x, y, hx, hy, u, v, s), true // see terrainsea.go
	}
	return seaWater(hx, hy, u, v, s), true
}

// rockShadow is how dark open ground is o pixels under a rock face.
func rockShadow(o int) float64 { return 0.34 * (1 - float64(o-1)/float64(rockUp)) }

// roofPixel is how chunk pixel (x, y) looks with the caves roofed over, if
// that differs from ground, the pixel without the roof.
func (b *bake) roofPixel(x, y int, u, v float64, ground color.RGBA) (color.RGBA, bool) {
	c, ok := b.rockPixel(b.roofed, x, y, u, v, true)
	if !ok {
		o := b.roofed.over[y*chunkPW+x]
		if o == 0 {
			return color.RGBA{}, false
		}
		if b.rock != nil && b.rock.over[y*chunkPW+x] == o {
			return color.RGBA{}, false // the same shadow as without the roof
		}
		c = mix(ground, colInk, rockShadow(int(o)))
	}
	return c, c != ground
}

// rockPixel colours a pixel on rock: its top or its south face. With roof,
// f has the caves as rock, and their mouths are cut into the faces.
func (b *bake) rockPixel(f *rockField, x, y int, u, v float64, roof bool) (color.RGBA, bool) {
	if f == nil {
		return color.RGBA{}, false
	}
	foot := int(f.foot[y*chunkPW+x])
	if foot == 0 {
		return color.RGBA{}, false
	}
	t := b.t
	hx, hy := b.x0+x, b.y0+y
	if foot > rockDown {
		if t.under > 0 && int(v) >= t.under {
			return caveRockTop(hx, hy, u, v), true // underground: solid rock, nearly black
		}
		return mountainTop(hx, hy, u, v), true
	}
	// The face. What lies at its foot decides what it looks like.
	if roof {
		// The face's foot wanders a little either side of the tile edge, so
		// the mouth tile is the one at the foot or above it.
		ftx, fty := int(math.Floor(u)), int(math.Floor(v+(float64(foot)-0.5*groundDens)/hdTH))
		for _, ty := range []int{fty, fty - 1} {
			if c, ok := t.mouthPixel(hx, hy, ftx, ty, foot); ok {
				return c, true
			}
		}
	}
	wall := t.tile(int(math.Floor(u)), int(math.Floor(v+float64(foot)/hdTH))) == sim.TileCave
	return faceColor(hx, hy, foot, wall), true
}

// edgeW is how far from a border between two kinds of ground, in landType's
// margin, the two still mix.
const edgeW = 0.26

// land colours a pixel of open ground: its biome, mixed along the border
// with the next one in ragged strokes and clumps.
func (b *bake) land(hx, hy int, u, v float64) color.RGBA {
	best, second, margin := b.t.landType(u, v)
	ew := b.t.edgeWidth(u, v) // see terrainfeatures.go
	if second == best || margin >= ew {
		return b.biome(best, hx, hy, u, v)
	}
	// Which of the two shows must come out the same from either side of the
	// border, so it is worked out for the pair in a fixed order.
	lo, hi, d := best, second, margin
	if lo > hi {
		lo, hi, d = hi, lo, -d
	}
	n := vnoise(float64(hx)/8, float64(hy)/7, 803)
	if grassy(lo) || grassy(hi) || lo == sim.TileMarsh || hi == sim.TileMarsh {
		// Blades reach over the border.
		n = 0.45*hashf(hx, fdiv(hy+int(hashf(hx, 0, 801)*6), 6), 802) + 0.55*n
	} else {
		n = 0.3*hashf(fdiv(hx, 2), fdiv(hy, 2), 802) + 0.7*n
	}
	e := d/ew*0.5 + n - 0.5
	k, other := lo, hi
	if e <= 0 {
		k, other, e = hi, lo, -e
	}
	c := b.biome(k, hx, hy, u, v)
	// Grass stands over what it borders: a little shade under its edge.
	if e < 0.06 {
		switch {
		case grassy(other) && !grassy(k):
			c = mix(c, colInk, 0.2)
		case grassy(k) && !grassy(other):
			c = mix(c, colInk, 0.1)
		}
	}
	return c
}

func grassy(k sim.Tile) bool { return k == sim.TileGrass || k == sim.TileForest }

// biome colours a pixel of ground of kind k.
func (b *bake) biome(k sim.Tile, hx, hy int, u, v float64) color.RGBA {
	switch k {
	case sim.TileGrass:
		if b.warm {
			return warmEarth(hx, hy, u, v, false)
		}
		return grassGround(hx, hy, u, v)
	case sim.TileForest:
		if b.warm {
			return warmEarth(hx, hy, u, v, true)
		}
		return forestGround(hx, hy, u, v)
	case sim.TileRocky:
		return rockyGround(hx, hy, u, v)
	case sim.TileDesert:
		return desertGround(hx, hy, u, v)
	case sim.TileMarsh:
		return marshGround(hx, hy, u, v)
	case sim.TileAsh:
		return ashGround(hx, hy, u, v)
	case sim.TileLava:
		return lavaGround(hx, hy, u, v)
	case sim.TileCave:
		return caveGround(hx, hy, u, v)
	case sim.TileMountain:
		return mountainTop(hx, hy, u, v)
	case sim.TileSnow:
		return snowGround(hx, hy, u, v)
	case sim.TilePath:
		return pathGround(hx, hy, u, v, b.warm)
	case sim.TileIce:
		return iceGround(hx, hy, u, v)
	}
	return sandGround(hx, hy, u, v)
}
