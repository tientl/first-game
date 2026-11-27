package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The water of worlds made since sim.StyleIsles, as the ground bake draws
// it: rivers in their beds, fords, waterfalls (part of the cliffs' walls:
// see terraincliff.go), lakes and the shallows of the sea. What moves on
// the water (flow lines, foam, glints, the falling sheet of a waterfall) is
// drawn over the bake every frame: see terrainanim.go.
//
// A river sits in a bed below its banks: the far bank (above it on the
// screen) shows a short face of earth falling to the water, the near bank
// throws a dark edge on it. Its water is lighter along its edges, where the
// bed shows through, and darker in its middle; boulders stand in it, more
// where it runs fast down a step of the levels (rapids), and a ford is a
// shallow run over pebbles with a row of stepping stones across it.

var (
	riverHD   = mkRamp(0x1c4f6c, 0x245f7f, 0x2d7193, 0x3884a6, 0x4a98b6, 0x63adc4, 0x86c4d3, 0xa9d8df).sub(2)
	riverBed  = mkRamp(0x4d5a3e, 0x5d6a49, 0x6f7a55, 0x86906a)
	bankEarth = mkRamp(0x2b2118, 0x3b2e21, 0x4c3c2b, 0x5e4b36, 0x735d43, 0x8a7253)
	stepStone = mkRamp(0x3a3a3c, 0x55544f, 0x706d64, 0x8c887c, 0xa8a393, 0xc4bfae)
	foamHD    = mkRamp(0x9fcfdc, 0xc3e3ea, 0xe0f1f4, 0xf7fcfd)
	lakeHD    = mkRamp(0x173f5e, 0x1d4c6d, 0x245a7d, 0x2d6a8d, 0x3a7d9e, 0x4f93b0, 0x6aa9c1, 0x8dc0cf).sub(2)
	lakeReeds = mkRamp(0x3d4a22, 0x56652c, 0x718236, 0x8c9a45)
)

const (
	riverUpM   = 18   // rows above a chunk its river field covers
	riverDownM = 8    // and below
	riverEdge  = 0.44 // the share of water where the water begins
	riverBank  = 0.3  // and its bank
)

// riverField is where the running water lies round a chunk being baked.
type riverField struct {
	f        []float32 // the share of water, rows from riverUpM above the chunk to riverDownM below
	up, down []uint8   // per chunk pixel: pixels up and down to the edge of the water, capped
}

// wetTile reports whether tile (tx, ty) is water a river runs in or into.
func (t *terrain) wetTile(tx, ty int) bool {
	k := t.tile(tx, ty)
	return runningWater(k) || k.Sea()
}

// waterShare is how much of (u, v) is a river's water: the share of the
// four tiles round it, eased between their centres and warped, so the
// banks wander.
func (t *terrain) waterShare(u, v float64) float64 {
	wu := u + (vnoise(u*0.9, v*0.9, 994)-0.5)*0.5
	wv := v + (vnoise(u*0.9+3, v*0.9, 995)-0.5)*0.5
	f := cornersAt(wu, wv).share(t.wetTile)
	return f + (vnoise(u*2.3, v*2.3, 966)-0.5)*0.14
}

// rivers finds the running water round the chunk, or returns nil if there
// is none.
func (b *bake) rivers() *riverField {
	t := b.t
	tx0, ty0 := fdiv(b.x0, hdTW), fdiv(b.y0-riverUpM, hdTH)
	tx1, ty1 := fdiv(b.x0+chunkPW, hdTW), fdiv(b.y0+chunkPH+riverDownM, hdTH)
	any := false
	for ty := ty0; ty <= ty1 && !any; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			if t.has(tx, ty, featRiver) {
				any = true
				break
			}
		}
	}
	if !any {
		return nil
	}
	rows := chunkPH + riverUpM + riverDownM
	rf := &riverField{f: make([]float32, chunkPW*rows), up: make([]uint8, chunkPW*chunkPH), down: make([]uint8, chunkPW*chunkPH)}
	for y := 0; y < rows; y++ {
		v := (float64(b.y0+y-riverUpM) + 0.5) / hdTH
		ty := int(math.Floor(v))
		for x := 0; x < chunkPW; x++ {
			u := (float64(b.x0+x) + 0.5) / hdTW
			if t.has(int(math.Floor(u)), ty, featRiver) {
				rf.f[y*chunkPW+x] = float32(t.waterShare(u, v))
			}
		}
	}
	wet := func(y, x int) bool { return rf.f[y*chunkPW+x] >= riverEdge }
	for x := 0; x < chunkPW; x++ {
		run := 0
		for y := 0; y < rows; y++ {
			if y == 0 && wet(y, x) {
				run = 200 // on past the margin: far from the edge
			} else if wet(y, x) {
				run = min(run+1, 255)
			} else {
				run = 0
			}
			if y >= riverUpM && y < riverUpM+chunkPH {
				rf.up[(y-riverUpM)*chunkPW+x] = uint8(run)
			}
		}
		run = 0
		for y := rows - 1; y >= 0; y-- {
			if y == rows-1 && wet(y, x) {
				run = 200
			} else if wet(y, x) {
				run = min(run+1, 255)
			} else {
				run = 0
			}
			if y >= riverUpM && y < riverUpM+chunkPH {
				rf.down[(y-riverUpM)*chunkPW+x] = uint8(run)
			}
		}
	}
	return rf
}

// share is the share of water at chunk pixel (x, y).
func (rf *riverField) share(x, y int) float64 { return float64(rf.f[(y+riverUpM)*chunkPW+x]) }

// waterTile is the tile of water nearest (u, v): the one under it, or the
// nearest beside it.
func (t *terrain) waterTile(u, v float64) (int, int) {
	tx, ty := int(math.Floor(u)), int(math.Floor(v))
	if t.wetTile(tx, ty) {
		return tx, ty
	}
	best, wx, wy := math.Inf(1), tx, ty
	for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		nx, ny := tx+d[0], ty+d[1]
		if !t.wetTile(nx, ny) {
			continue
		}
		if dd := math.Hypot(float64(nx)+0.5-u, float64(ny)+0.5-v); dd < best {
			best, wx, wy = dd, nx, ny
		}
	}
	return wx, wy
}

// flowAt is the way the water flows at (u, v), eased between the tiles of
// running water round it (zero over still water).
func (t *terrain) flowAt(u, v float64) (fx, fy float64) {
	cs := cornersAt(u, v)
	for k := range 4 {
		if !runningWater(t.tile(cs.tx[k], cs.ty[k])) {
			continue
		}
		dx, dy := t.rel(cs.tx[k], cs.ty[k]).Flow().Delta()
		fx += cs.wt[k] * float64(dx)
		fy += cs.wt[k] * float64(dy)
	}
	if n := math.Hypot(fx, fy); n > 0 {
		return fx / n, fy / n
	}
	return 0, 0
}

// rapid reports whether river tile (tx, ty) runs fast: down a step of the
// levels, or to the lip of a waterfall.
func (t *terrain) rapid(tx, ty int) bool {
	k := t.tile(tx, ty)
	if k != sim.TileRiver {
		return false
	}
	dx, dy := t.rel(tx, ty).Flow().Delta()
	n := t.tile(tx+dx, ty+dy)
	return n == sim.TileWaterfall || runningWater(n) && t.rel(tx+dx, ty+dy).Level() < t.rel(tx, ty).Level()
}

// riverPixel colours chunk pixel (x, y) if it is a river or its bank.
func (b *bake) riverPixel(x, y, hx, hy int, u, v float64) (color.RGBA, bool) {
	rf := b.river
	if rf == nil {
		return color.RGBA{}, false
	}
	f := rf.share(x, y)
	if f < riverBank {
		return color.RGBA{}, false
	}
	t := b.t
	if f < riverEdge {
		// The bank: wet earth and gravel, the grass hanging over its top.
		w := (f - riverBank) / (riverEdge - riverBank)
		g := b.land(hx, hy, u, v)
		if w < 0.18 {
			return scaleCol(g, 1.12), true // the grass's edge in the light
		}
		c := mix(g, bankEarth.at(2), 0.45+0.5*w)
		switch p, l := pebble(hx, hy, 7, 6, 1302, 0.4, 0.8, 1.8); p {
		case pebStone:
			c = stepStone.at(1 + int(l*4))
		case pebShadow:
			c = scaleCol(c, 0.8)
		}
		return c, true
	}
	j := y*chunkPW + x
	up, down := int(rf.up[j]), int(rf.down[j])
	// The far bank: the earth falls a little way to the water.
	if bh := 5 + int(5*vnoise(float64(hx)/13, float64(hy)/40, 1301)); up <= bh {
		return bankFace(hx, hy, up, bh), true
	}
	wx, wy := t.waterTile(u, v)
	fx, fy := t.flowAt(u, v)
	depth := clamp01((f - riverEdge) / 0.42)
	depth = min(depth, float64(up-6)/20, float64(down)/16)
	cs := cornersAt(u, v)
	fast := cs.share(t.rapid)
	ford := cs.share(func(x, y int) bool { return t.tile(x, y) == sim.TileFord })
	c := riverWater(hx, hy, u, v, fx, fy, depth*(1-0.7*ford), fast)
	if ford > 0.05 {
		c = fordPixel(c, hx, hy, fx, fy, ford, t.fordLine(wx, wy, math.Abs(fy) > math.Abs(fx)))
	} else if r, ok := riverRock(hx, hy, fx, fy, fast, smooth(clamp01(depth*3))); ok {
		c = r
	}
	if fast > 0.3 {
		// Where it drops a level, a line of white water across it.
		if d := t.levelAt(u, v) - math.Floor(t.levelAt(u, v)); d > 0.4 && d < 0.6 {
			s := vnoise(float64(hx)/2.5, float64(hy)/6, 1303)
			if s < 0.25+0.75*(1-math.Abs(d-0.5)/0.1) {
				c = foamHD.g(0.45+0.5*s, hx, hy)
			}
		}
	}
	// The pool under a waterfall churns white.
	if b.wall != nil {
		if a := b.wall.underFall(x, y); a > 0 {
			n := vnoise(float64(hx)/5, float64(hy)/4, 1304)*0.6 + hashf(fdiv(hx, 2), fdiv(hy, 2), 1305)*0.4
			if n < a*1.25 {
				c = foamHD.g(0.5+0.5*a+(n-0.5)*0.3, hx, hy)
			} else if n < a*1.25+0.12 {
				c = mix(c, foamHD.at(1), 0.5)
			}
		}
	}
	// The near bank's lip shades the water under it.
	if down <= 3 {
		c = scaleCol(c, 0.72)
	} else if down <= 5 {
		c = scaleCol(c, 0.86)
	}
	return c, true
}

// bankFace is the earth of a river's far bank, up pixels below its top,
// bh pixels tall: lit along its top, roots and stones in it, wet and dark
// where it meets the water.
func bankFace(hx, hy, up, bh int) color.RGBA {
	if up <= 1 {
		return bankEarth.at(5)
	}
	f := float64(up) / float64(bh)
	l := 0.75 - 0.55*f + (vnoise(float64(hx)/3, float64(hy)/9, 1306)-0.5)*0.3
	if p, pl := pebble(hx, hy, 6, 5, 1307, 0.3, 0.8, 1.6); p == pebStone {
		return stepStone.at(1 + int(pl*3))
	}
	if up >= bh-1 {
		return mix(bankEarth.at(0), riverHD.at(4), 0.4) // wet, at the waterline
	}
	return bankEarth.pick(l, hx, hy)
}

// riverWater is running water at (u, v) flowing along (fx, fy), depth 0 at
// its edge to 1 in its middle: the bed showing through at the edges, dark
// in the middle, long streaks along the current.
func riverWater(hx, hy int, u, v, fx, fy, depth, fast float64) color.RGBA {
	along := u*fx + v*fy
	across := -u*fy + v*fx
	s := vnoise(across*5, along*0.6, 969) - 0.5
	l := 0.86 - 0.8*depth + s*0.08 + 0.12*fast
	if fast > 0 && vnoise(across*9+0.4*vnoise(u*5, v*5, 1309), along*1.4, 1308) > 0.76-0.1*fast {
		return foamHD.g(0.2+0.3*fast, hx, hy)
	}
	// In steps, not a smooth ramp: the bed, the shallows, the deep.
	q := math.Floor(l*5+dither(hx, hy)*0.6) / 5
	c := riverHD.at(int(clamp01(q) * float64(len(riverHD)-1)))
	if depth < 0.1 {
		c = mix(c, riverBed.at(2), 0.4) // the bed, seen through
	}
	return c
}

// riverRock colours (hx, hy) if it is on a boulder standing in the water,
// or the foam it raises: a cushion against its upstream side and a wake
// trailing downstream (fx, fy). rapid (0 to 1) is how fast the water runs
// there (more boulders), deep how far from the bank it is.
func riverRock(hx, hy int, fx, fy, rapid, deep float64) (color.RGBA, bool) {
	dens := (0.05 + 0.3*rapid) * deep
	if dens <= 0 {
		return color.RGBA{}, false
	}
	const cw, ch = 34, 26
	cx, cy := fdiv(hx, cw), fdiv(hy, ch)
	for oy := -1; oy <= 1; oy++ {
		for ox := -1; ox <= 1; ox++ {
			h := hash2(int32(cx+ox), int32(cy+oy), 1310)
			if float64(h&0xffff)/65536 >= dens {
				continue
			}
			rx := 4 + 5*float64(h>>16&0xff)/255
			ry := rx * 0.7
			px := float64((cx+ox)*cw) + 7 + float64(h>>24&0xf)/15*(cw-14)
			py := float64((cy+oy)*ch) + 7 + float64(h>>28)/15*(ch-14)
			qx, qy := float64(hx)+0.5-px, float64(hy)+0.5-py
			dx, dy := qx/rx, qy/ry
			// A lumpy outline, not an ellipse.
			ang := math.Atan2(dy, dx)
			lump := 1 + 0.16*math.Sin(3*ang+float64(h&7)) + 0.08*math.Sin(5*ang+float64(h>>3&7))
			r := math.Hypot(dx, dy) / lump
			along := qx*fx + qy*fy   // pixels downstream of its middle
			across := -qx*fy + qy*fx // pixels to one side
			switch {
			case r < 1:
				if r > 0.84 && dx+dy > -0.2 {
					return stepStone.at(0), true
				}
				l := 0.62 - 0.36*dx - 0.5*dy
				if dy > 0.3 {
					l -= 0.28 // wet and dark at the waterline
				}
				return stepStone.at(int(clamp01(l) * 5.4)), true
			case r < 1.45 && along < rx*0.3:
				if hashf(hx, hy, 1311) < 0.8 {
					return foamHD.at(2 + int(hashf(hx, hy, 1312)*1.99)), true // the cushion
				}
			case along > 0 && along < rx*2.6:
				w := rx * (0.75 - 0.2*along/(rx*2.6))
				if math.Abs(across) < w && hashf(fdiv(hx, 2), fdiv(hy, 2), 1316) < 0.55-along/(rx*6) {
					return foamHD.at(1 + int(hashf(hx, hy, 1317)*2.99)), true // the wake
				}
			}
		}
	}
	return color.RGBA{}, false
}

// fordLine is the middle of the ford across river tile (tx, ty), along
// the river: in pixels down the screen if the river runs north-south
// (acrossX), across it if it runs east-west.
func (t *terrain) fordLine(tx, ty int, acrossX bool) float64 {
	dx, dy := 0, 1
	if !acrossX {
		dx, dy = 1, 0
	}
	if t.tile(tx, ty) != sim.TileFord {
		// Beside the ford: the nearest ford tile along the river.
		for k := 1; k <= 2; k++ {
			if t.tile(tx+dx*k, ty+dy*k) == sim.TileFord {
				tx, ty = tx+dx*k, ty+dy*k
				break
			}
			if t.tile(tx-dx*k, ty-dy*k) == sim.TileFord {
				tx, ty = tx-dx*k, ty-dy*k
				break
			}
		}
	}
	lo, hi := 0, 0
	for lo > -4 && t.tile(tx+dx*(lo-1), ty+dy*(lo-1)) == sim.TileFord {
		lo--
	}
	for hi < 4 && t.tile(tx+dx*(hi+1), ty+dy*(hi+1)) == sim.TileFord {
		hi++
	}
	if acrossX {
		return (float64(ty) + float64(lo+hi+1)/2) * hdTH
	}
	return (float64(tx) + float64(lo+hi+1)/2) * hdTW
}

// fordPixel lays a ford over river water c: ford (0 to 1) is how much of
// the point is ford, line where the stepping stones cross. The water runs
// shallow over pebbles, rippling round a row of flat stones.
func fordPixel(c color.RGBA, hx, hy int, fx, fy, ford, line float64) color.RGBA {
	c = mix(c, riverBed.at(3), 0.3*ford)
	if p, l := pebble(hx, hy, 9, 7, 1313, 0.65*ford, 1.4, 3.2); p == pebStone {
		c = mix(stepStone.at(1+int(l*4)), c, 0.4) // under the water
	} else if p == pebShadow {
		c = scaleCol(c, 0.9)
	}
	if ford < 0.5 {
		return c
	}
	acrossX := math.Abs(fy) > math.Abs(fx) // the river runs north-south: cross it east-west
	var a, b float64                       // pixels across the river, and along it from the line
	if acrossX {
		a, b = float64(hx), float64(hy)-line
	} else {
		a, b = float64(hy)*1.3, float64(hx)-line
	}
	const gap = 25
	k := math.Floor(a / gap)
	for _, kk := range []float64{k - 1, k, k + 1} {
		h := hash2(int32(kk), int32(line), 1314)
		ca := kk*gap + gap/2 + (float64(h&0xff)/255-0.5)*6
		cb := (float64(h>>8&0xff)/255 - 0.5) * 7
		ra, rb := 8+3*float64(h>>16&0xff)/255, 6+2*float64(h>>24)/255
		if !acrossX {
			ra, rb = ra/1.3, rb*1.25
		}
		da, db := (a-ca)/ra, (b-cb)/rb
		dx, dy := da, db
		if !acrossX {
			dx, dy = db, da
		}
		switch r := math.Hypot(da, db); {
		case r < 0.86:
			l := 0.78 - 0.3*dx - 0.45*dy
			if dy > 0.4 {
				l -= 0.3 // its wet side
			}
			return stepStone.at(1 + int(clamp01(l)*4.2))
		case r < 1:
			return stepStone.at(0)
		case r < 1.4 && hashf(hx, hy, 1315) < 0.55:
			return foamHD.at(2)
		}
	}
	return c
}

// fallPixel is a pixel of a waterfall pouring over a wall (see
// terraincliff.go): the river bending glassy over the lip, then a sheet of
// streaks, white where it breaks on a ledge half way down and at its foot.
// Its moving streaks are drawn over it (terrainanim.go).
func fallPixel(hx, hy int, p wallPx) color.RGBA {
	a := float64(hx)
	if math.Abs(p.gx) > math.Abs(p.gy) {
		a = float64(hy)
	}
	f := p.f
	switch {
	case f < 0.07:
		return riverHD.at(len(riverHD) - 4 - int(f*40)) // the lip, glassy
	case f < 0.11:
		return foamHD.at(3)
	}
	s := vnoise(a/2.2, p.d/24, 973)*0.7 + vnoise(a/1.1, p.d/8, 974)*0.3
	c := foamHD.g(0.15+0.95*s, hx, hy)
	if s < 0.3 {
		c = mix(riverHD.at(9), c, s/0.3) // thin water over the rock
	}
	// A ledge half way down it breaks on.
	if d := f - (0.52 + 0.06*vnoise(a/14, 0, 975)); d > -0.04 && d < 0.05 {
		return foamHD.g(0.7+0.3*hashf(hx, hy, 976), hx, hy)
	} else if d >= 0.05 && d < 0.12 {
		c = mix(c, riverHD.at(6), 0.35) // in the shadow of the ledge
	}
	if f > 0.84 {
		c = mix(c, foamHD.at(3), clamp01((f-0.84)*7))
	}
	return c
}

// --- lakes -------------------------------------------------------------------------

// lakePixel colours a pixel of a lake, s (< 0) tiles in from its shore and
// up pixels under the land above it, if (u, v) is one: still water, dark in
// the middle, the sky's light on it, a muddy rim with reeds, and a short
// bank of earth on its far shore.
func (t *terrain) lakePixel(hx, hy int, u, v, s float64, up int) (color.RGBA, bool) {
	if t.relief == nil || !t.has(int(math.Floor(u)), int(math.Floor(v)), featShallow) {
		return color.RGBA{}, false
	}
	cs := cornersAt(u, v)
	if cs.share(func(x, y int) bool { return t.rel(x, y).Variant() == sim.VarLake && t.tile(x, y) == sim.TileWater }) <= 0 {
		return color.RGBA{}, false
	}
	if bh := 5 + int(4*vnoise(float64(hx)/13, float64(hy)/40, 1320)); up <= bh {
		return bankFace(hx, hy, up, bh), true
	}
	edge := -0.1 - (vnoise(u*3, v*3, 1321)-0.5)*0.1
	if s > edge {
		// The rim: mud, and reeds standing in clumps.
		if reedAt(hx, hy) {
			return lakeReeds.at(int(hashf(hx, fdiv(hy, 3), 1322) * 3.99)), true
		}
		return bankEarth.pick(0.35+(hashf(hx, hy, 963)-0.5)*0.25, hx, hy), true
	}
	if s > edge-0.25 && reedAt(hx, hy) && hashf(fdiv(hx, 9), fdiv(hy, 7), 1323) < 0.5 {
		return lakeReeds.at(1 + int(hashf(hx, fdiv(hy, 3), 1322)*2.99)), true // reeds in the shallows
	}
	d := clamp01(-s / 2.6)
	l := 0.88 - 0.8*math.Sqrt(d)
	// The sky in it: long faint streaks of light across the still water.
	if vnoise(float64(hx)/40, float64(hy)/2.5, 1324) > 0.72 {
		l += 0.1
	}
	c := lakeHD.g(l+(vnoise(float64(hx)/20, float64(hy)/12, 964)-0.5)*0.08, hx, hy)
	if up <= 12 {
		c = scaleCol(c, 0.8+0.02*float64(up-6)) // the far bank's shadow
	}
	if s > edge-0.08 {
		c = mix(c, riverBed.at(1), 0.4) // the shallow rim, the mud showing
	}
	return c, true
}

// reedAt reports whether (hx, hy) is on a reed: thin stems in clumps.
func reedAt(hx, hy int) bool {
	cx, cy := fdiv(hx, 14), fdiv(hy, 11)
	h := hash2(int32(cx), int32(cy), 1325)
	if h&0xff > 70 {
		return false
	}
	x0 := cx*14 + 3 + int(h>>8&7)
	dx := hx - x0
	top := cy*11 + int(h>>12&3)
	return dx >= 0 && dx < 7 && dx%2 == 0 && hy >= top+int(hashf(x0+dx, cy, 1326)*4) && hy < cy*11+11
}
