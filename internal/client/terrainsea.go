package client

import (
	"image/color"
	"math"
	"sync"

	"survivalgame/internal/sim"
)

// The sea of the newest worlds as the ground bake draws it: one smooth
// depth over the whole of it, worked out per tile once (seaDepths) and eased
// between the tiles, so the water runs from pale green over the sand by a
// beach through clear turquoise over the shallows to the blue of the
// drop-off and the navy of the deep, every edge between them soft. Reefs
// show their coral and darker beds, lagoons their calm rippled sand and sea
// grass, shoals the sand of their banks, all in soft shapes. What moves on
// it (light rippling over the shallows) is drawn by the water shader: see
// terrainanim.go.
//
// On the land side of the waterline (beachShore): the wet sand the waves
// have just left, the tide line with its sea-wrack, and shells.

// seaStops is the colour of the water by its depth, 0 at the waterline to 1
// in the deep, with how much of the moving water under it shows through
// (less where the sand of the bottom shows).
var seaStops = []struct {
	d float64
	c color.RGBA
	a float64
}{
	{0.00, hex(0xc7ebd3), 238},
	{0.10, hex(0xa9e3cd), 234},
	{0.24, hex(0x80d7cb), 229},
	{0.40, hex(0x52c1c5), 222},
	{0.52, hex(0x3aa5bc), 210},
	{0.62, hex(0x2a7cab), 192},
	{0.72, hex(0x22659f), 182},
	{0.84, hex(0x1a4c86), 174},
	{1.00, hex(0x0f3060), 168}, // the open ocean, darkest
}

// seaColor is the water at depth d (0 to 1).
func seaColor(d float64) color.RGBA {
	d = clamp01(d)
	for k := 1; k < len(seaStops); k++ {
		a, b := seaStops[k-1], seaStops[k]
		if d <= b.d {
			f := (d - a.d) / (b.d - a.d)
			c := mix(a.c, b.c, f)
			return withAlpha(c, uint8(math.Round(lerpf(a.a, b.a, f))))
		}
	}
	last := seaStops[len(seaStops)-1]
	return withAlpha(last.c, uint8(last.a))
}

// Depths by the kind of water, in seaDepths.
const (
	depthShelf = 0.6  // the deep, where it meets the shallows
	depthDeep  = 14.0 // tiles beyond the shallows it is deepest
	depthCliff = 0.55 // at the foot of a sea cliff, which drops straight into the deep
)

// seaDepths works out every tile's depth, 0 to 255 for 0 to 1: 0 on land
// (but deep at a cliff's foot), the shallows deeper away from the shore and
// towards their edge, shoals and lagoons shallower, reefs deeper, the open
// sea deeper the further it lies beyond the shallows. Blurred, so it eases
// across the tiles. The last world's is kept: the ground is made again
// with each season, on the same relief.
func seaDepths(w, h int, tiles []sim.Tile, r []sim.Relief) []uint8 {
	depthMemo.Lock()
	defer depthMemo.Unlock()
	if len(r) > 0 && depthMemo.rel == &r[0] && depthMemo.w == w && depthMemo.h == h {
		return depthMemo.d
	}
	// What each tile is, and how many rows down the sea reaches (the caves
	// under the world have none).
	const (
		clsLand = iota
		clsShallow
		clsDeep
		clsCliff
	)
	cls := make([]uint8, w*h)
	rows := 0
	for i, k := range tiles {
		switch {
		case k == sim.TileShallow:
			cls[i], rows = clsShallow, i/w+1
		case k == sim.TileWater && r[i].Variant() != sim.VarLake:
			cls[i], rows = clsDeep, i/w+1
		case k == sim.TileCliff:
			cls[i] = clsCliff
		}
	}
	n := w * rows
	cls = cls[:n]
	var toLand, toDeep, toShelf []uint16
	var wg sync.WaitGroup
	for _, job := range []struct {
		out  *[]uint16
		mask uint8
	}{{&toLand, 1<<clsLand | 1<<clsCliff}, {&toDeep, 1 << clsDeep}, {&toShelf, 1<<clsLand | 1<<clsCliff | 1<<clsShallow}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			*job.out = chamferDist(w, rows, cls, job.mask)
		}()
	}
	wg.Wait()
	d := make([]float32, n)
	for i, c := range cls {
		switch c {
		case clsShallow:
			q := float64(toLand[i]) / float64(toLand[i]+toDeep[i])
			var v float64
			switch r[i].Variant() {
			case sim.VarShoal:
				v = 0.05 + 0.3*q
			case sim.VarLagoon:
				v = 0.08 + 0.22*q
			case sim.VarReef:
				v = 0.3 + 0.22*q
			default:
				v = 0.1 + 0.42*q
			}
			d[i] = float32(v)
		case clsDeep:
			x := clamp01((float64(toShelf[i])/chamferStep - 0.5) / depthDeep)
			d[i] = float32(depthShelf + (1-depthShelf)*(1-(1-x)*(1-x))) // steep by the shelf, then on darker into the open ocean
		case clsCliff:
			d[i] = depthCliff
		}
	}
	for range 2 {
		d = boxBlur(w, rows, d)
	}
	out := make([]uint8, w*h)
	for i, v := range d {
		out[i] = uint8(clamp01(float64(v))*255 + 0.5)
	}
	if len(r) > 0 {
		depthMemo.rel, depthMemo.w, depthMemo.h, depthMemo.d = &r[0], w, h, out
	}
	return out
}

// depthMemo is the last world's sea depths: see seaDepths.
var depthMemo struct {
	sync.Mutex
	rel  *sim.Relief
	w, h int
	d    []uint8
}

// chamferStep is a straight step in chamferDist's units; a diagonal one is 7.
const chamferStep = 5

// chamferDist is, for every tile, about how far it lies from the nearest
// tile whose class is in mask (0 on them), in fifths of a tile: straight
// steps 5, diagonal ones 7.
func chamferDist(w, h int, cls []uint8, mask uint8) []uint16 {
	const far, diag = 60000, 7
	d := make([]uint16, w*h)
	for i, c := range cls {
		if mask&(1<<c) == 0 {
			d[i] = far
		}
	}
	for y := 0; y < h; y++ {
		row := d[y*w : (y+1)*w]
		var up []uint16
		if y > 0 {
			up = d[(y-1)*w : y*w]
		}
		for x := range row {
			v := row[x]
			if x > 0 {
				v = min(v, row[x-1]+chamferStep)
			}
			if up != nil {
				v = min(v, up[x]+chamferStep)
				if x > 0 {
					v = min(v, up[x-1]+diag)
				}
				if x < w-1 {
					v = min(v, up[x+1]+diag)
				}
			}
			row[x] = v
		}
	}
	for y := h - 1; y >= 0; y-- {
		row := d[y*w : (y+1)*w]
		var down []uint16
		if y < h-1 {
			down = d[(y+1)*w : (y+2)*w]
		}
		for x := w - 1; x >= 0; x-- {
			v := row[x]
			if x < w-1 {
				v = min(v, row[x+1]+chamferStep)
			}
			if down != nil {
				v = min(v, down[x]+chamferStep)
				if x < w-1 {
					v = min(v, down[x+1]+diag)
				}
				if x > 0 {
					v = min(v, down[x-1]+diag)
				}
			}
			row[x] = v
		}
	}
	return d
}

// boxBlur is d averaged over each tile and the eight round it (the edge of
// the map repeats outward).
func boxBlur(w, h int, d []float32) []float32 {
	row := make([]float32, len(d))
	for y := 0; y < h; y++ {
		s, o := d[y*w:(y+1)*w], row[y*w:(y+1)*w]
		for x := range s {
			o[x] = (s[max(x-1, 0)] + s[x] + s[min(x+1, w-1)]) / 3
		}
	}
	out := make([]float32, len(d))
	for y := 0; y < h; y++ {
		a, b, c := row[max(y-1, 0)*w:], row[y*w:], row[min(y+1, h-1)*w:]
		o := out[y*w : (y+1)*w]
		for x := range o {
			o[x] = (a[x] + b[x] + c[x]) / 3
		}
	}
	return out
}

// depthAt is the sea's depth at (u, v), eased between the tile centres; 1
// off the map, and 0 everywhere in worlds without relief.
func (t *terrain) depthAt(u, v float64) float64 {
	if t.depth == nil {
		return 0
	}
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := x-x0, y-y0
	at := func(tx, ty int) float64 {
		if tx < 0 || ty < 0 || tx >= t.w || ty >= t.h {
			return 1
		}
		return float64(t.depth[ty*t.w+tx]) / 255
	}
	return lerpf(lerpf(at(ix, iy), at(ix+1, iy), fx), lerpf(at(ix, iy+1), at(ix+1, iy+1), fx), fy)
}

// seaWarp is (u, v) pushed about a little, so the shapes of the shallows
// wander off the lines of the tiles.
func seaWarp(u, v float64) (float64, float64) {
	wu := u + (vnoise(u*0.4, v*0.4, 958)-0.5)*0.9 + (vnoise(u*1.3, v*1.3, 1330)-0.5)*0.2
	wv := v + (vnoise(u*0.4+5, v*0.4, 959)-0.5)*0.9 + (vnoise(u*1.3+4, v*1.3, 1331)-0.5)*0.2
	return wu, wv
}

// seaDepthAt is the depth the sea is drawn at at (u, v): warped, with a
// little slow unevenness of the bottom where it is shallow.
func (t *terrain) seaDepthAt(u, v float64) float64 {
	wu, wv := seaWarp(u, v)
	d := t.depthAt(wu, wv)
	if d < 0.75 {
		d += (fbm2(u*0.45+3, v*0.5, 1351, 3) - 0.5) * 0.12 * (1 - d)
	}
	return clamp01(d)
}

var (
	lagoonCol = hex(0x74dccb)
	seaGrass  = hex(0x2f7f6a)
	reefBed   = hex(0x2f8c8f)
	shoalSand = hex(0xe3e6b8)
)

// reliefSea colours a pixel of the sea s (< 0) tiles out from the shore,
// in a world with relief.
func (t *terrain) reliefSea(hx, hy int, u, v, s float64) color.RGBA {
	if c, ok := shoreFoam(hx, hy, u, v, s); ok {
		return c
	}
	if t.depthAt(u, v) >= 0.999 {
		return seaColor(1) // the open sea
	}
	d := t.seaDepthAt(u, v)
	c := seaColor(d)
	if d >= 0.7 {
		return c
	}
	wu, wv := seaWarp(u, v)
	cs := cornersAt(wu, wv)
	is := func(vr sim.GroundVariant) float64 {
		return cs.share(func(x, y int) bool { return t.tile(x, y) == sim.TileShallow && t.rel(x, y).Variant() == vr })
	}
	clear := smooth(clamp01((0.62 - d) / 0.4)) // how plainly the bottom shows
	sb := is(sim.VarShoal)
	// Sand ripples on the bottom, faint, plainer where it is shallow and on
	// the banks of a shoal.
	rip := math.Sin((float64(hx)*0.22+float64(hy)*0.95)/3.4 + 7*fbm2(u*0.15, v*0.2, 1352, 2))
	if rip > 0.55 {
		c = mix(c, withAlpha(shoalSand, c.A), (0.12+0.1*sb)*clear*(rip-0.55)/0.45)
	}
	if lg := is(sim.VarLagoon); lg > 0 {
		c = mix(c, withAlpha(lagoonCol, c.A), 0.45*lg)
		if g := fbm2(u*0.32+9, v*0.4, 1334, 3); g > 0.56 {
			c = mix(c, withAlpha(seaGrass, c.A), 0.42*lg*smooth(clamp01((g-0.56)/0.12))) // sea grass
		}
	}
	if sb > 0 {
		// The crest of the bank, where it is shallowest: the sand itself.
		if k := clamp01((0.24 - d) / 0.16); k > 0 {
			c = mix(c, withAlpha(shoalSand, c.A), 0.45*sb*smooth(k))
		}
	}
	if rf := is(sim.VarReef); rf > 0.2 {
		if cc, ok := coralHead(hx, hy, rf*clear); ok {
			return cc
		}
		// The reef's bed under the heads, darker than the sand.
		if p := fbm2(u*0.3+2, v*0.38, 1339, 3); p > 0.48 {
			c = mix(c, withAlpha(reefBed, c.A), 0.5*rf*clear*smooth(clamp01((p-0.48)/0.12)))
		}
	}
	return c
}

// seaLight is how much of the light rippling over the shallows shows at
// sea point (u, v) (see terrainanim.go): all of it where they are
// shallowest, none out in the deep.
func (t *terrain) seaLight(u, v float64) float64 {
	return smooth(clamp01((0.6 - t.seaDepthAt(u, v)) / 0.45))
}

// --- the beach at the waterline ---------------------------------------------------

var (
	wetSand  = hex(0x9a8257)
	wetSheen = hex(0xd5e3da)
	wrack    = mkRamp(0x2a2a17, 0x3c4321, 0x52602a, 0x6c7a33, 0x8d9443, 0xaeac5c)
	driftRmp = mkRamp(0x4a4640, 0x6b665c, 0x8d877a, 0xaea798, 0xcbc4b3, 0xe2dccd)
	shellHD  = []ramp{mkRamp(0xb8aa96, 0xe2d6c4, 0xf7f0e4), mkRamp(0xb58a86, 0xdcaea6, 0xf3d6cc), mkRamp(0x9c8a6c, 0xcdb894, 0xeedfbe)}
)

// sandAbove reports whether the land up pixels above sea point (u, v) is a
// beach, in a world with relief: its sand runs down into the water, with no
// wall of earth under it.
func (t *terrain) sandAbove(u, v float64, up int) bool {
	if t.relief == nil {
		return false
	}
	cs := cornersAt(u, v-float64(up+1)/hdTH)
	sand, land := 0.0, 0.0
	for k := range 4 {
		switch kt := t.tile(cs.tx[k], cs.ty[k]); {
		case kt.Sea():
		case kt == sim.TileSand:
			sand += cs.wt[k]
			land += cs.wt[k]
		default:
			land += cs.wt[k]
		}
	}
	return land > 0 && sand > 0.5*land
}

// beachShore lays the edge of the beach over land pixel c at (u, v), s
// tiles in from the waterline: the sand darker and smoother where the
// waves have just run up it, the tide line higher up with its sea-wrack in
// broken strands, and shells. Other ground by the sea darkens a little,
// wet, at the very edge.
func (t *terrain) beachShore(c color.RGBA, hx, hy int, u, v, s float64) color.RGBA {
	sand := cornersAt(u, v).share(func(x, y int) bool { return t.tile(x, y) == sim.TileSand })
	if sand < 1 && s < 0.22 { // mud at the edge of anything else
		w := (1 - s/0.22) * (1 - sand)
		c = mix(c, hex(0x6b5a3e), 0.45*w*(0.85+0.3*hashf(hx, hy, 73)))
	}
	if sand <= 0 {
		return c
	}
	// The wet band, wandering in and out along the beach.
	wet := 0.7 + 0.24*(vnoise(u*0.7, v*0.7, 1360)-0.5)
	if s < wet {
		k := smooth(clamp01(1-s/wet)) * sand
		c = mix(c, wetSand, 0.58*k)
		if s < 0.16 {
			c = mix(c, wetSheen, 0.3*(1-s/0.16)*sand) // the sky in the wettest
		}
	}
	// The tide line: clumps of sea-wrack strung along it, with gaps.
	line := 1.05 + 0.3*(vnoise(u*0.5+7, v*0.5, 1361)-0.5) + 0.12*(vnoise(u*2.2, v*2.2, 1362)-0.5)
	if dl := s - line; math.Abs(dl) < 0.14 && sand > 0.5 {
		clump := vnoise(float64(hx)/17, float64(hy)/12, 1363)*0.75 + vnoise(float64(hx)/6, float64(hy)/5, 1364)*0.25
		wd := 0.08 * clamp01((clump-0.5)/0.16) // its half-width: none in the gaps
		switch e := math.Abs(dl) / max(wd, 1e-6); {
		case wd > 0.012 && e < 1:
			// Weed in tangled strands, darker at its edges, a glint of wet
			// here and there.
			l := 0.62 - 0.45*e + (hashf(fdiv(hx, 2), hy, 1365)-0.5)*0.55
			if dl > 0 && e > 0.55 {
				l += 0.18 // its top, inland, in the light
			}
			if hashf(hx, hy, 1368) < 0.03 {
				return wetSheen
			}
			return wrack.at(int(clamp01(l) * 5.99))
		case wd > 0.012 && e < 1.35:
			c = mix(c, wetSand, 0.35) // damp round it
		}
		if hashf(fdiv(hx, 2), fdiv(hy, 2), 1366) < 0.012*(1-math.Abs(dl)/0.14) {
			return wrack.at(1 + int(hashf(hx, hy, 1367)*2.99)) // a bit washed up
		}
	}
	// Shells, here and there below the tide line.
	if s > 0.2 && s < 2.2 {
		if sc, ok := shellAt(hx, hy, sand*(1.1-s/2.2)); ok {
			return sc
		}
	}
	return c
}

// shellAt colours (hx, hy) if a shell lies there: a small fan or a round
// one, pale, lit from the upper left, with a shadow under it. dens (0 to 1)
// is how many lie about.
func shellAt(hx, hy int, dens float64) (color.RGBA, bool) {
	const cw, ch = 30, 23
	cx, cy := fdiv(hx, cw), fdiv(hy, ch)
	h := hash2(int32(cx), int32(cy), 1368)
	if float64(h&0xffff)/65536 >= 0.07*dens {
		return color.RGBA{}, false
	}
	px := cx*cw + 4 + int(h>>16&0xf)%(cw-8)
	py := cy*ch + 4 + int(h>>20&0xf)%(ch-8)
	rp := shellHD[int(h>>24)%len(shellHD)]
	dx, dy := hx-px, hy-py
	if h>>28&1 == 0 {
		// A fan: ribbed, wider at the top.
		if dy == 2 && dx >= -1 && dx <= 2 {
			return hex(0x7d6a4a), true // its shadow
		}
		if dy < -2 || dy > 1 || dx < -2 || dx > 2 || dy == 1 && (dx < -1 || dx > 1) || dy == -2 && (dx == -2 || dx == 2) {
			return color.RGBA{}, false
		}
		l := 2 - (dy+2)/2
		if dx%2 != 0 && dy > -2 {
			l--
		}
		return rp.at(max(0, l)), true
	}
	// A round one, a small spiral.
	if dy == 2 && dx >= -1 && dx <= 1 {
		return hex(0x7d6a4a), true
	}
	if dx*dx+dy*dy > 3 || dy == 2 {
		return color.RGBA{}, false
	}
	switch {
	case dx == 0 && dy == 0:
		return rp.at(0), true
	case dx < 0 || dy < 0:
		return rp.at(2), true
	}
	return rp.at(1), true
}

// genDriftwood is a branch washed up on a beach: bleached grey, its bark
// long gone, cracked along its grain, a stub where a bough broke off.
func genDriftwood() sprite {
	if !envHD {
		return genTwig()
	}
	c := hdCanvas(24, 12)
	const k = hdK3
	branch := func(x0, y0, x1, y1, w0, w1 float64) {
		drawLimbHD(c, limb{x0 * k, y0 * k, (x0 + x1) / 2 * k, (y0+y1)/2*k - 1, x1 * k, y1 * k, w0, w1}, driftRmp, 0.95, 1372)
	}
	branch(2.5, 8, 21, 6.2, 4.2, 2.6)
	branch(14, 6.8, 18.5, 2.5, 2.2, 1.3)
	branch(5.5, 7.6, 3.6, 4.4, 1.9, 1.2)
	// Cracks along the grain, and the broken end showing its rings.
	for i := 0; i < 5; i++ {
		y0 := 7.4 + 0.5*float64(i%3) - 0.12*float64(i)
		x0 := 4.0 + 3.2*float64(i)
		for x := x0 * k; x < (x0+2.6)*k; x++ {
			c.put(int(x), int(y0*k-(x/k-2.5)*0.1*k), driftRmp.at(1))
		}
	}
	c.ellipse(2.5*k, 8*k, 2.2, 3.6, func(x, y int, nx, ny float64) color.RGBA {
		if nx*nx+ny*ny < 0.3 {
			return driftRmp.at(2)
		}
		return driftRmp.pick(0.75-nx*0.3, x, y)
	})
	c.outline(0.6)
	return mkUp(c, 12*k, 9*k)
}

// beachDunes lays the dunes at the back of a wide beach over sand pixel c,
// a (0 to 1) how much of the point is dunes: low ridges across the wind,
// the slope up to each crest in the light, its lee in shade.
func beachDunes(c color.RGBA, hx, hy int, a float64) color.RGBA {
	ph := (float64(hx)*0.3+float64(hy)*0.95)/46 + 1.6*fbm2(float64(hx)/150, float64(hy)/110, 1370, 2)
	f := ph - math.Floor(ph)
	amp := a * (0.6 + 0.4*vnoise(float64(hx)/90, float64(hy)/70, 1371))
	var l float64
	switch {
	case f < 0.7:
		l = 0.09 * (f / 0.7) * (f / 0.7)
	case f < 0.74:
		l = 0.09 - (f-0.7)/0.04*0.17 // over the crest
	default:
		k := 1 - (f-0.74)/0.26
		l = -0.08 * k
	}
	return scaleCol(c, 1+l*amp)
}
