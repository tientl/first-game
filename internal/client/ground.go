package client

import (
	"image/color"
	"math"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The ground is not drawn tile by tile. Each pixel is coloured from smooth
// fields interpolated between tile centres and warped with noise, so
// coastlines and biome borders come out organic instead of square. That is
// too slow per frame, so the map is baked into chunks on worker goroutines
// and cached. Water pixels are left translucent so an animated water layer
// shows through underneath.

const (
	chunkTiles = 16
	chunkW     = chunkTiles * TileW
	chunkH     = chunkTiles * TileH
	cliffH     = 6 // art pixels of earth wall below land that borders the sea
	maxChunks  = 64
	coastWarp  = 0.22 // how far (in tiles) noise may push the coastline
)

var (
	grassRamp  = mkRamp(0x3b7236, 0x4b8a3d, 0x5d9e45, 0x74b34f, 0x8fc65c)
	forestRamp = mkRamp(0x26472d, 0x2f5733, 0x3a663a, 0x467341, 0x548148)
	sandRamp   = mkRamp(0xb8975f, 0xcfb179, 0xe0c58c, 0xeed6a0, 0xf7e6bb)
	rockRamp   = mkRamp(0x544f49, 0x6a645b, 0x80786c, 0x958c7d, 0xaaa190)
	cliffRamp  = mkRamp(0x3d2a22, 0x4f372a, 0x664632, 0x7d573b, 0x9a7048)
	litter     = mkRamp(0x5a4a2f, 0x75603b)
	flowers    = []color.RGBA{hex(0xf4f0e6), hex(0xf6d55c), hex(0xe98bb4), hex(0xa7b8f0)}

	colFoam    = hex(0xeaf6f8)
	colShallow = hex(0x7fd3cf)
	colDeep    = hex(0x163565)
	colWaterBG = hex(0x3a7cbc)
)

// terrain is the static map plus a signed distance field to the coast:
// positive on land, negative at sea, in tiles (capped).
type terrain struct {
	w, h  int
	tiles []sim.Tile
	sdf   []float32
	pond  []float32 // 1 in and beside swamp ponds; see swampart.go
	under int       // the first row of the caves under the world, 0 if none: their rock is drawn black
}

func newTerrain(w, h int, tiles []sim.Tile) *terrain {
	t := &terrain{w: w, h: h, tiles: tiles, sdf: make([]float32, w*h)}
	land := func(x, y int) bool { return t.tile(x, y) != sim.TileWater }
	const capDist = 8
	for _, wantLand := range []bool{true, false} {
		dist := make([]int8, w*h)
		var queue []int32
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if land(x, y) != wantLand {
					continue
				}
				// Seed with tiles that touch the other kind (or the map edge,
				// which counts as sea).
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if land(x+dx, y+dy) != wantLand {
							dist[y*w+x] = 1
						}
					}
				}
				if dist[y*w+x] == 1 {
					queue = append(queue, int32(y*w+x))
				}
			}
		}
		for len(queue) > 0 {
			i := int(queue[0])
			queue = queue[1:]
			x, y, d := i%w, i/w, dist[i]
			if d >= capDist {
				continue
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h || land(nx, ny) != wantLand {
						continue
					}
					if j := ny*w + nx; dist[j] == 0 {
						dist[j] = d + 1
						queue = append(queue, int32(j))
					}
				}
			}
		}
		for i, d := range dist {
			if land(i%w, i/w) != wantLand {
				continue
			}
			if d == 0 {
				d = capDist
			}
			v := float32(d) - 0.5
			if !wantLand {
				v = -v
			}
			t.sdf[i] = v
		}
	}
	t.pond = swampPonds(w, h, tiles)
	return t
}

func (t *terrain) tile(tx, ty int) sim.Tile {
	if tx < 0 || ty < 0 || tx >= t.w || ty >= t.h {
		return sim.TileWater
	}
	if v := t.tiles[ty*t.w+tx]; v < sim.NumTiles {
		return v
	}
	return sim.TileWater
}

func (t *terrain) sdfAt(tx, ty int) float64 {
	if tx < 0 || ty < 0 || tx >= t.w || ty >= t.h {
		return -8
	}
	return float64(t.sdf[ty*t.w+tx])
}

// coast is the warped signed distance to the shore at (u, v) in tiles.
func (t *terrain) coast(u, v float64) float64 {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := x-x0, y-y0
	s := lerpf(lerpf(t.sdfAt(ix, iy), t.sdfAt(ix+1, iy), fx),
		lerpf(t.sdfAt(ix, iy+1), t.sdfAt(ix+1, iy+1), fx), fy)
	warp := (fbm2(u*0.7, v*0.9, 11, 2)-0.5)*2*coastWarp + (vnoise(u*3.1, v*3.7, 12)-0.5)*0.12
	return s + warp
}

// landType picks the biome at (u, v) and reports how close the pixel is to
// the border with the runner-up (0 = right on it).
func (t *terrain) landType(u, v float64) (best, second sim.Tile, margin float64) {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := smooth(x-x0), smooth(y-y0)
	var score [sim.NumTiles]float64
	var present [sim.NumTiles]bool
	add := func(tx, ty int, w float64) {
		k := t.tile(tx, ty)
		if k == sim.TileWater {
			k = sim.TileSand // the sea floor and beaches are sand
			if t.pondTile(tx, ty) {
				k = sim.TileMarsh // but a swamp pond's bed is mud
			}
		}
		score[k] += w
		present[k] = true
	}
	add(ix, iy, (1-fx)*(1-fy))
	add(ix+1, iy, fx*(1-fy))
	add(ix, iy+1, (1-fx)*fy)
	add(ix+1, iy+1, fx*fy)

	best, second = sim.TileSand, sim.TileSand
	bs, ss := math.Inf(-1), math.Inf(-1)
	for k := sim.TileSand; k < sim.NumTiles; k++ {
		if !present[k] {
			continue
		}
		s := score[k] + (fbm2(u*1.2+float64(k)*17.3, v*1.5+float64(k)*9.1, 20+uint32(k), 2)-0.5)*0.55
		if s > bs {
			second, ss = best, bs
			best, bs = k, s
		} else if s > ss {
			second, ss = k, s
		}
	}
	if math.IsInf(ss, -1) {
		return best, best, 1
	}
	return best, second, bs - ss
}

// biomeIdx returns the palette and index for a land pixel of kind k.
func biomeIdx(k sim.Tile, px, py int, u, v float64) (ramp, int, color.RGBA) {
	var none color.RGBA
	switch k {
	case sim.TileGrass:
		n := fbm2(u*0.22, v*0.3, 31, 3)
		i := grassRamp.idx(0.42+(n-0.5)*1.3+(vnoise(u*1.7, v*2.2, 32)-0.5)*0.35, px, py)
		// Short dark blades with a lit tip above.
		const p = 0.045
		switch {
		case hashf(px, py, 33) < p || hashf(px, py-1, 33) < p:
			i--
		case hashf(px, py+1, 33) < p:
			i++
		}
		if fbm2(u*0.16+40, v*0.2, 34, 2) > 0.6 && hashf(px, py, 35) < 0.012 {
			return grassRamp, i, flowers[int(hashf(px, py, 36)*float64(len(flowers)))]
		}
		return grassRamp, i, none
	case sim.TileForest:
		n := fbm2(u*0.3, v*0.4, 41, 3)
		i := forestRamp.idx(0.45+(n-0.5)*1.2, px, py)
		if h := hashf(px, py, 42); h < 0.07 {
			return forestRamp, i, litter.at(int(h * 30))
		}
		if hashf(px, py-1, 43) < 0.05 {
			i--
		}
		return forestRamp, i, none
	case sim.TileRocky:
		n := fbm2(u*0.35, v*0.45, 51, 3)
		i := rockRamp.idx(0.45+(n-0.5)*1.1, px, py)
		// Pebbles: one per small cell, lit on top with a shadow below.
		cx, cy := px/5, py/4
		ox, oy := cx*5+1+int(hashf(cx, cy, 52)*3), cy*4+1+int(hashf(cx, cy, 53)*2)
		if hashf(cx, cy, 54) < 0.35 {
			switch {
			case (px == ox || px == ox+1) && py == oy:
				i += 2
			case (px == ox || px == ox+1) && py == oy+1:
				i -= 2
			}
		}
		if math.Abs(vnoise(u*2.3, v*3.1, 55)-0.5) < 0.018 {
			i -= 2 // cracks
		}
		return rockRamp, i, none
	default: // sand
		n := fbm2(u*0.3, v*0.4, 61, 2)
		i := sandRamp.idx(0.55+(n-0.5)*0.8, px, py)
		r := math.Sin((v*2.4 + u*0.5 + fbm2(u*0.5, v*0.6, 62, 2)*2.5) * 2 * math.Pi)
		if r > 0.93 {
			i--
		} else if r < -0.96 {
			i++
		}
		if h := hashf(px, py, 63); h < 0.03 {
			i += []int{-1, 1}[int(h*66)%2]
		}
		return sandRamp, i, none
	}
}

func biomeColor(k sim.Tile, px, py int, u, v float64) color.RGBA {
	if c, ok := warmBiome(k, px, py, u, v); ok {
		return c // see warmground.go
	}
	if c, ok := openBiome(k, px, py, u, v); ok {
		return c
	}
	r, i, special := biomeIdx(k, px, py, u, v)
	if special.A > 0 {
		return special
	}
	return r.at(i)
}

// groundPixel colours one art pixel of the ground at world pixel (px, py).
// It also returns the signed coast distance there and whether waves may be
// drawn over the pixel (not on mountains or the earth walls of the coast).
func (t *terrain) groundPixel(px, py int) (color.RGBA, float64, bool) {
	u, v := (float64(px)+0.5)/TileW, (float64(py)+0.5)/TileH
	s := t.coast(u, v)

	if s >= 0 {
		mc, shadow, ok := t.mountainPixel(px, py, u, v, false)
		if ok {
			return mc, s, false // mountains stand above everything else
		}
		best, second, margin := t.landType(u, v)
		c := biomeColor(best, px, py, u, v)
		if margin < 0.09 && second != best {
			// Dithered blend into the neighbouring biome, then a darker rim.
			if dither(px, py) < (0.09-margin)/0.2 {
				c = biomeColor(second, px, py, u, v)
			}
			if margin < 0.025 && (best == sim.TileGrass || best == sim.TileForest) {
				c = mix(c, colInk, 0.22)
			}
		}
		if shadow > 0 {
			c = mix(c, colInk, shadow) // at the foot of a mountain's face
		}
		if s < surfLand && t.pondAt(u, v) > 0.5 { // a swamp pond's bank: wet mud, and no surf
			if s < 0.22 {
				c = swampBank(c, px, py, s)
			}
			return c, s, false
		}
		if s < 0.22 { // wet sand at the water's edge
			c = mix(c, hex(0x6b5a3e), 0.45*(1-s/0.22))
			if s < 0.05 {
				c = mix(c, colFoam, 0.25)
			}
		}
		return c, s, true
	}

	if t.pondAt(u, v) > 0.5 {
		return t.swampWater(px, py, u, v, s), s, false // still, murky water: no surf
	}

	// Sea. A south-facing shore gets an earth wall hanging below it.
	if s > -1.6 {
		for k := 1; k <= cliffH+2; k++ {
			if t.coast(u, v-float64(k)/TileH) < 0 {
				continue
			}
			if k > cliffH {
				return withAlpha(colFoam, 200), s, false
			}
			f := float64(k-1) / (cliffH - 1)
			i := cliffRamp.idx(0.95-f*0.85+(vnoise(float64(px)*0.6, float64(py)*0.25, 70)-0.5)*0.4, px, py)
			if k == 1 {
				i = len(cliffRamp) - 1
			}
			if hashf(px/2, py, 71) < 0.12 {
				i--
			}
			return cliffRamp.at(i), s, false
		}
	}

	foamEdge := -0.1 - (vnoise(u*4, v*4, 72)-0.5)*0.12
	if s > foamEdge {
		return withAlpha(colFoam, 220), s, true
	}
	// Depth: turquoise shallows fading to deep navy, in dithered bands.
	d := clamp01(-s / 4.5)
	steps := []color.RGBA{
		withAlpha(colShallow, 150), withAlpha(colShallow, 70), color.RGBA{},
		withAlpha(colDeep, 70), withAlpha(colDeep, 130), withAlpha(colDeep, 170),
	}
	p := math.Sqrt(d) * float64(len(steps)-1)
	i := int(p)
	if p-float64(i) > dither(px, py) {
		i++
	}
	return steps[min(i, len(steps)-1)], s, true
}

// --- chunk cache ------------------------------------------------------------

type chunkKey struct{ cx, cy int }

type chunkResult struct {
	key   chunkKey
	gen   int
	data  []byte
	shore []byte // nil when the chunk has no coast
	roof  []byte // nil when the chunk has no cave under a mountain
}

type groundCache struct {
	terr   *terrain
	gen    int
	chunks map[chunkKey]*ebiten.Image
	shores map[chunkKey]*ebiten.Image // coast distances for the surf; absent inland
	roofs  map[chunkKey]*ebiten.Image // the rock over caves; absent where there are none

	// roofAlpha is how much of the cave roofs to draw: 1 from outside, 0
	// for a player inside a cave; caveIn eases between them (see caves.go).
	roofAlpha float32
	caveIn    float64
	used      map[chunkKey]int
	pending   map[chunkKey]bool
	results   chan chunkResult
	sem       chan struct{}

	wave *ebiten.Image
}

func newGroundCache() *groundCache {
	n := max(1, runtime.NumCPU()-1)
	return &groundCache{
		chunks:    make(map[chunkKey]*ebiten.Image),
		shores:    make(map[chunkKey]*ebiten.Image),
		roofs:     make(map[chunkKey]*ebiten.Image),
		roofAlpha: 1,
		used:      make(map[chunkKey]int),
		pending:   make(map[chunkKey]bool),
		results:   make(chan chunkResult, 64),
		sem:       make(chan struct{}, n),
		wave:      genWaveTexture(),
	}
}

// reset switches to a new map and forgets every cached chunk.
func (gc *groundCache) reset(terr *terrain) {
	gc.terr = terr
	gc.gen++
	for k, img := range gc.chunks {
		img.Deallocate()
		delete(gc.chunks, k)
	}
	for k, img := range gc.shores {
		img.Deallocate()
		delete(gc.shores, k)
	}
	for k, img := range gc.roofs {
		img.Deallocate()
		delete(gc.roofs, k)
	}
	clear(gc.used)
	clear(gc.pending)
}

// bakeChunk returns the chunk's ground pixels and, if it has a coastline,
// its shore image for the surf, and if it has caves, their roof (else nil).
func (t *terrain) bakeChunk(k chunkKey) (ground, shore, roof []byte) {
	c := newCanvas(chunkW, chunkH)
	rf := newCanvas(chunkW, chunkH)
	sh := make([]byte, chunkW*chunkH*4)
	coast, caves := false, false
	x0, y0 := k.cx*chunkW, k.cy*chunkH
	for y := 0; y < chunkH; y++ {
		for x := 0; x < chunkW; x++ {
			col, s, surf := t.groundPixel(x0+x, y0+y)
			c.px[y*chunkW+x] = col
			if rc, ok := t.roofPixel(x0+x, y0+y, col); ok {
				rf.px[y*chunkW+x] = rc
				caves = true
			}
			if b, ok := shoreByte(s); ok && surf {
				i := (y*chunkW + x) * 4
				sh[i], sh[i+3] = b, 255
				coast = true
			}
		}
	}
	if !coast {
		sh = nil
	}
	if caves {
		roof = rf.bytes()
	}
	return c.bytes(), sh, roof
}

// store keeps a baked chunk.
func (gc *groundCache) store(k chunkKey, ground, shore, roof []byte) *ebiten.Image {
	img := ebiten.NewImage(chunkW, chunkH)
	img.WritePixels(ground)
	gc.chunks[k] = img
	if shore != nil {
		s := ebiten.NewImage(chunkW, chunkH)
		s.WritePixels(shore)
		gc.shores[k] = s
	}
	if roof != nil {
		r := ebiten.NewImage(chunkW, chunkH)
		r.WritePixels(roof)
		gc.roofs[k] = r
	}
	return img
}

func (gc *groundCache) request(k chunkKey) {
	if gc.chunks[k] != nil || gc.pending[k] {
		return
	}
	gc.pending[k] = true
	terr, gen := gc.terr, gc.gen
	go func() {
		gc.sem <- struct{}{}
		data, shore, roof := terr.bakeChunk(k)
		<-gc.sem
		gc.results <- chunkResult{k, gen, data, shore, roof}
	}()
}

func (gc *groundCache) collect() {
	for {
		select {
		case r := <-gc.results:
			if r.gen != gc.gen {
				continue
			}
			delete(gc.pending, r.key)
			if gc.chunks[r.key] != nil {
				continue // already baked synchronously
			}
			gc.store(r.key, r.data, r.shore, r.roof)
		default:
			return
		}
	}
}

// get returns the chunk image, baking it on the spot if it is not ready.
func (gc *groundCache) get(k chunkKey, frame int) *ebiten.Image {
	img := gc.chunks[k]
	if img == nil {
		ground, shore, roof := gc.terr.bakeChunk(k)
		img = gc.store(k, ground, shore, roof)
	}
	gc.used[k] = frame
	return img
}

func (gc *groundCache) evict(frame int) {
	for len(gc.chunks) > maxChunks {
		var oldest chunkKey
		of := math.MaxInt
		for k := range gc.chunks {
			if f := gc.used[k]; f < of {
				oldest, of = k, f
			}
		}
		gc.chunks[oldest].Deallocate()
		delete(gc.chunks, oldest)
		if s := gc.shores[oldest]; s != nil {
			s.Deallocate()
			delete(gc.shores, oldest)
		}
		if r := gc.roofs[oldest]; r != nil {
			r.Deallocate()
			delete(gc.roofs, oldest)
		}
		delete(gc.used, oldest)
	}
}

// draw paints water and ground for the view (ox, oy, w, h) in world pixels.
func (gc *groundCache) draw(dst drawTarget, ox, oy float64, frame int) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	gc.drawWater(dst, ox, oy, frame)
	gc.collect()

	cx0, cy0 := int(math.Floor(ox/chunkW)), int(math.Floor(oy/chunkH))
	cx1, cy1 := int(math.Floor((ox+float64(w))/chunkW)), int(math.Floor((oy+float64(h))/chunkH))
	for cy := cy0 - 1; cy <= cy1+1; cy++ {
		for cx := cx0 - 1; cx <= cx1+1; cx++ {
			gc.request(chunkKey{cx, cy})
		}
	}
	op := &ebiten.DrawImageOptions{}
	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(cx*chunkW)-ox, float64(cy*chunkH)-oy)
			dst.DrawImage(gc.get(chunkKey{cx, cy}, frame), op)
		}
	}
	if gc.roofAlpha > 0 {
		rop := &ebiten.DrawImageOptions{}
		rop.ColorScale.ScaleAlpha(gc.roofAlpha)
		for cy := cy0; cy <= cy1; cy++ {
			for cx := cx0; cx <= cx1; cx++ {
				if r := gc.roofs[chunkKey{cx, cy}]; r != nil {
					rop.GeoM.Reset()
					rop.GeoM.Translate(float64(cx*chunkW)-ox, float64(cy*chunkH)-oy)
					dst.DrawImage(r, rop)
				}
			}
		}
	}
	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			if s := gc.shores[chunkKey{cx, cy}]; s != nil {
				wx, wy := float64(cx*chunkW), float64(cy*chunkH)
				drawSurf(dst, s, wx-ox, wy-oy, wx, wy, frame)
			}
		}
	}
	gc.evict(frame)
}

const waveW, waveH = 96, 72

// drawWater fills the view with water and two drifting layers of ripples.
func (gc *groundCache) drawWater(dst drawTarget, ox, oy float64, frame int) {
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	fillRect(dst, -1, -1, float32(w)+2, float32(h)+2, colWaterBG, false) // a pixel over, for the canvas's sub-pixel shift
	t := float64(frame)
	for _, l := range []struct{ vx, vy, a float64 }{{0.12, 0.05, 0.55}, {-0.08, 0.09, 0.35}} {
		sx := math.Mod(-ox+t*l.vx, waveW)
		sy := math.Mod(-oy+t*l.vy, waveH)
		if sx > 0 {
			sx -= waveW
		}
		if sy > 0 {
			sy -= waveH
		}
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(float32(l.a))
		for y := math.Floor(sy); y < h; y += waveH {
			for x := math.Floor(sx); x < w; x += waveW {
				op.GeoM.Reset()
				op.GeoM.Translate(x, y)
				dst.DrawImage(gc.wave, op)
			}
		}
	}
}

// genWaveTexture draws tileable contour-line ripples.
func genWaveTexture() *ebiten.Image {
	c := newCanvas(waveW, waveH)
	light := hex(0x9fd4f2)
	for y := 0; y < waveH; y++ {
		for x := 0; x < waveW; x++ {
			n := pnoise(float64(x)/16, float64(y)/9, waveW/16, waveH/9, 80)*0.7 +
				pnoise(float64(x)/8, float64(y)/4.5, waveW/8, waveH/9*2, 81)*0.3
			switch {
			case math.Abs(n-0.5) < 0.012:
				c.set(x, y, withAlpha(light, 230))
			case math.Abs(n-0.36) < 0.01 || math.Abs(n-0.64) < 0.01:
				c.set(x, y, withAlpha(light, 110))
			}
		}
	}
	return c.image()
}
