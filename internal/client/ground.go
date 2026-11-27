package client

import (
	"image/color"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The ground is not drawn tile by tile. Each pixel is coloured from smooth
// fields interpolated between tile centres and warped with noise, so
// coastlines and biome borders come out organic instead of square. That is
// too slow per frame, so the map is baked into chunks on worker goroutines
// and cached. Water pixels are left translucent so an animated water layer
// shows through underneath. The baking itself is in groundhd.go and the
// look of each kind of ground in groundart.go.

const (
	chunkTiles = 8
	chunkW     = chunkTiles * TileW // in art pixels
	chunkH     = chunkTiles * TileH
	cliffH     = 6    // art pixels of earth wall below land that borders the sea
	minChunks  = 72   // chunks kept at least; more when the view shows more
	coastWarp  = 0.22 // how far (in tiles) noise may push the coastline

	// The ground and the cave roofs are baked at groundDens image pixels to
	// an art pixel (see hd.go and groundhd.go); the shore image the surf
	// shader reads stays at one pixel to an art pixel, as the shader runs
	// at art-pixel resolution.
	groundDens = HDDensity
	chunkPW    = chunkW * groundDens // a chunk's image, in pixels
	chunkPH    = chunkH * groundDens
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

	relief []sim.Relief // per tile, in worlds made since sim.StyleIsles; see terrainfeatures.go
	feat   []uint8      // per tile, what of the relief lies near: see terrainfeatures.go
	depth  []uint8      // per tile, the sea's depth, eased: see terrainsea.go
	cave   caveTerrain  // the natural caves, since sim.StyleCaves: see cavebake.go
}

func newTerrain(w, h int, tiles []sim.Tile) *terrain {
	t := &terrain{w: w, h: h, tiles: tiles, sdf: make([]float32, w*h)}
	land := func(x, y int) bool { return !t.tile(x, y).Sea() } // shallows are sea; rivers and cliffs, land
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
	if s > 1.2 {
		return s // well inland: nothing looks at the exact distance
	}
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
		switch {
		case runningWater(k):
			k = sim.TileSand // a river's bed and banks; the water is drawn over them
		case k == sim.TileCliff:
			k = t.cliffGround(tx, ty) // the high ground it is the edge of: see terrainfeatures.go
		}
		if k.Sea() {
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
	for k := sim.TileSand; k < sim.NumTiles; k++ {
		if score[k] > 0.999999 {
			return k, k, 1 // one kind all around: no border near
		}
	}

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

// --- chunk cache ------------------------------------------------------------

type chunkKey struct{ cx, cy int }

type chunkResult struct {
	key   chunkKey
	gen   int
	data  []byte
	shore []byte // nil when the chunk has no coast
	roof  []byte // nil when the chunk has no cave under a mountain
	water []byte // nil when the chunk has no water that moves: see terrainanim.go
	front []byte // nil when no cliff's face reaches down over figures: see terraincliff.go
	rows  uint8
}

type groundCache struct {
	terr   *terrain
	gen    int
	chunks map[chunkKey]*ebiten.Image
	shores map[chunkKey]*ebiten.Image // coast distances for the surf; absent inland
	roofs  map[chunkKey]*ebiten.Image // the rock over caves; absent where there are none
	waters map[chunkKey]*ebiten.Image // the water maps of the moving water; absent where there is none
	seas   map[chunkKey]bool          // the chunks with sea in them, which the sea's surface is drawn over: see seasurface.go
	fronts map[chunkKey]*ebiten.Image // the feet of the cliffs' faces, drawn over the figures by them (terraincliff.go)
	frows  map[chunkKey]uint8         // which rows of tiles of each have any
	strips []wallStrip                // the bands of them in view this frame

	// roofAlpha is how much of the cave roofs to draw: 1 from outside, 0
	// for a player inside a cave; caveIn eases between them (see caves.go).
	roofAlpha float32
	caveIn    float64
	used      map[chunkKey]int
	pending   map[chunkKey]bool
	results   chan chunkResult
	sem       chan struct{}

	sea   seaLook     // the weather on the sea, for its surface: see seasurface.go
	style GroundStyle // the look the chunks were baked in
	keys  []chunkKey  // the chunks in view, for draw
}

func newGroundCache() *groundCache {
	n := max(1, runtime.NumCPU()-1)
	gc := &groundCache{
		chunks:    make(map[chunkKey]*ebiten.Image),
		shores:    make(map[chunkKey]*ebiten.Image),
		roofs:     make(map[chunkKey]*ebiten.Image),
		waters:    make(map[chunkKey]*ebiten.Image),
		seas:      make(map[chunkKey]bool),
		fronts:    make(map[chunkKey]*ebiten.Image),
		frows:     make(map[chunkKey]uint8),
		roofAlpha: 1,
		used:      make(map[chunkKey]int),
		pending:   make(map[chunkKey]bool),
		results:   make(chan chunkResult, 64),
		sem:       make(chan struct{}, n),
		style:     groundStyle,
	}
	liveGround = gc // see terrainanim.go
	return gc
}

// reset switches to a new map and forgets every cached chunk.
func (gc *groundCache) reset(terr *terrain) {
	gc.terr = terr
	gc.style = groundStyle
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
	for k, img := range gc.waters {
		img.Deallocate()
		delete(gc.waters, k)
	}
	clear(gc.seas)
	for k, img := range gc.fronts {
		img.Deallocate()
		delete(gc.fronts, k)
	}
	clear(gc.frows)
	clear(gc.used)
	clear(gc.pending)
}

// store keeps a baked chunk.
func (gc *groundCache) store(k chunkKey, bc bakedChunk) *ebiten.Image {
	ground, shore, roof, water := bc.ground, bc.shore, bc.roof, bc.water
	if bc.front != nil {
		f := ebiten.NewImage(chunkPW, chunkTiles*frontBandH)
		f.WritePixels(bc.front)
		gc.fronts[k], gc.frows[k] = f, bc.frontRows
	}
	img := ebiten.NewImage(chunkPW, chunkPH)
	img.WritePixels(ground)
	gc.chunks[k] = img
	if hasSea(ground) {
		gc.seas[k] = true
	} else {
		delete(gc.seas, k)
	}
	if shore != nil {
		s := ebiten.NewImage(chunkW, chunkH)
		s.WritePixels(shore)
		gc.shores[k] = s
	}
	if roof != nil {
		r := ebiten.NewImage(chunkPW, chunkPH)
		r.WritePixels(roof)
		gc.roofs[k] = r
	}
	if water != nil {
		w := ebiten.NewImage(chunkW, chunkH)
		w.WritePixels(water)
		gc.waters[k] = w
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
		bc := terr.bakeChunkFull(k)
		<-gc.sem
		gc.results <- chunkResult{k, gen, bc.ground, bc.shore, bc.roof, bc.water, bc.front, bc.frontRows}
	}()
}

// collect takes in chunks baked in the background: a few a frame, as each
// is a megabyte to hand to the graphics card.
func (gc *groundCache) collect(most int) {
	for n := 0; n < most; n++ {
		select {
		case r := <-gc.results:
			if r.gen != gc.gen {
				continue
			}
			delete(gc.pending, r.key)
			if gc.chunks[r.key] != nil {
				continue // already baked synchronously
			}
			gc.store(r.key, bakedChunk{r.data, r.shore, r.roof, r.water, r.front, r.rows})
		default:
			return
		}
	}
}

// ensure bakes whichever of the chunks keys are not ready yet, all at once
// on every core, and waits for them: when the world first shows, or the
// view jumps further than the chunks baked ahead.
func (gc *groundCache) ensure(keys []chunkKey) {
	var missing []chunkKey
	for pass := 0; pass < 2; pass++ {
		missing = missing[:0]
		for _, k := range keys {
			if gc.chunks[k] == nil {
				missing = append(missing, k)
			}
		}
		if len(missing) == 0 {
			return
		}
		if pass == 0 {
			gc.collect(math.MaxInt) // some may be baked and waiting
		}
	}
	out := make([]bakedChunk, len(missing))
	var wg sync.WaitGroup
	next := int32(-1)
	for w := 0; w < min(len(missing), runtime.NumCPU()); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(atomic.AddInt32(&next, 1))
				if i >= len(missing) {
					return
				}
				out[i] = gc.terr.bakeChunkFull(missing[i])
			}
		}()
	}
	wg.Wait()
	for i, k := range missing {
		gc.store(k, out[i])
	}
}

// evict forgets the chunks used longest ago, down to keep of them.
func (gc *groundCache) evict(keep int) {
	for len(gc.chunks) > keep {
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
		if w := gc.waters[oldest]; w != nil {
			w.Deallocate()
			delete(gc.waters, oldest)
		}
		delete(gc.seas, oldest)
		if f := gc.fronts[oldest]; f != nil {
			f.Deallocate()
			delete(gc.fronts, oldest)
			delete(gc.frows, oldest)
		}
		delete(gc.used, oldest)
	}
}

// draw paints water and ground for the view (ox, oy, w, h) in world pixels.
func (gc *groundCache) draw(dst drawTarget, ox, oy float64, frame int) {
	if gc.style != groundStyle { // the look was switched: bake it all anew
		gc.style = groundStyle
		gc.reset(gc.terr)
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	gc.drawWater(dst)
	gc.collect(4)

	cx0, cy0 := int(math.Floor(ox/chunkW)), int(math.Floor(oy/chunkH))
	cx1, cy1 := int(math.Floor((ox+float64(w))/chunkW)), int(math.Floor((oy+float64(h))/chunkH))
	gc.keys = gc.keys[:0]
	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			gc.keys = append(gc.keys, chunkKey{cx, cy})
		}
	}
	gc.ensure(gc.keys)
	for cy := cy0 - 1; cy <= cy1+1; cy++ {
		for cx := cx0 - 1; cx <= cx1+1; cx++ {
			gc.request(chunkKey{cx, cy})
		}
	}
	for _, k := range gc.keys {
		gc.used[k] = frame
		drawChunkImage(dst, gc.chunks[k], float64(k.cx*chunkW)-ox, float64(k.cy*chunkH)-oy, ebiten.ColorScale{})
	}
	for _, k := range gc.keys {
		if gc.seas[k] {
			wx, wy := float64(k.cx*chunkW), float64(k.cy*chunkH)
			gc.drawSeaSurface(dst, gc.chunks[k], wx-ox, wy-oy, wx, wy, frame) // see seasurface.go
		}
	}
	if gc.roofAlpha > 0 {
		var cs ebiten.ColorScale
		cs.ScaleAlpha(gc.roofAlpha)
		for _, k := range gc.keys {
			if r := gc.roofs[k]; r != nil {
				drawChunkImage(dst, r, float64(k.cx*chunkW)-ox, float64(k.cy*chunkH)-oy, cs)
			}
		}
	}
	for _, k := range gc.keys {
		if s := gc.shores[k]; s != nil {
			wx, wy := float64(k.cx*chunkW), float64(k.cy*chunkH)
			drawSurf(dst, s, wx-ox, wy-oy, wx, wy, frame)
		}
	}
	for _, k := range gc.keys {
		if m := gc.waters[k]; m != nil {
			wx, wy := float64(k.cx*chunkW), float64(k.cy*chunkH)
			drawWaterAnim(dst, m, wx-ox, wy-oy, wx, wy, frame) // see terrainanim.go
		}
	}
	// Room for what is in view and the ring baked ahead, twice over.
	gc.evict(max(minChunks, 2*(cx1-cx0+3)*(cy1-cy0+3)))
}

// drawWater fills the view with the deep blue the sea of the chunks lies
// over, see-through as it is (more the deeper): the floor of the water. What
// moves on it is drawn over the chunks (seasurface.go).
func (gc *groundCache) drawWater(dst drawTarget) {
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	fillRect(dst, -1, -1, float32(w)+2, float32(h)+2, colWaterBG, false) // a pixel over, for the canvas's sub-pixel shift
}

// drawChunkImage draws img, baked at groundDens and lying edge to edge with
// its neighbours (a chunk of the ground, the rock roofing its caves, a band
// of a cliff's foot), its corner at (x, y) art pixels on dst, coloured by
// cs. No such image may blend with what lies outside it, or a thin line
// shows along every chunk's edge. At a whole scale its pixels land whole
// and are taken nearest, and drawn smaller (720p, zoomed out) a smooth
// sample never reaches past its edge: there it goes through drawImageD as
// any image does. Blown up by a fraction, the smooth samples along its edge
// would take in the transparent border round it. The world is not drawn at
// such scales (crispScale, display.go), but should it be, the image is
// drawn from just inside its edge, so its outermost pixels are held at the
// edge as by a texture clamped to its edge.
func drawChunkImage(dst drawTarget, img *ebiten.Image, x, y float64, cs ebiten.ColorScale) {
	wd, ok := dst.(*worldDst)
	if !ok || wd.r <= groundDens || whole(wd.r/groundDens) {
		op := &ebiten.DrawImageOptions{ColorScale: cs}
		op.GeoM.Translate(x, y)
		drawImageD(dst, img, op, groundDens)
		return
	}
	wd.flush()
	s := wd.r / groundDens // canvas pixels to an image pixel
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	// The source is taken in from each side by just enough that the
	// outermost canvas pixels sample a hair inside the middles of the
	// outermost image pixels, and nothing beyond them. (Right on a middle,
	// a smooth sample reads its pixel and the one past it on a border
	// between two, and which of those the card takes is a toss-up.)
	const hair = 1.0 / 32
	in := func(n float64) float64 { return (0.5 + hair - 0.5/s) / (1 - 1/(n*s)) }
	ix, iy := in(w), in(h)
	x0, y0 := math.Round(x*wd.r), math.Round(y*wd.r)
	x1, y1 := x0+w*s, y0+h*s
	sx0, sy0 := float64(b.Min.X)+ix, float64(b.Min.Y)+iy
	sx1, sy1 := float64(b.Max.X)-ix, float64(b.Max.Y)-iy
	vs := make([]ebiten.Vertex, 4)
	for i, c := range [4][4]float64{{x0, y0, sx0, sy0}, {x1, y0, sx1, sy0}, {x0, y1, sx0, sy1}, {x1, y1, sx1, sy1}} {
		vs[i] = ebiten.Vertex{DstX: float32(c[0]), DstY: float32(c[1]), SrcX: float32(c[2]), SrcY: float32(c[3]),
			ColorR: cs.R(), ColorG: cs.G(), ColorB: cs.B(), ColorA: cs.A()}
	}
	wd.canvas.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, img, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}
