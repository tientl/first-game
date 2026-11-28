package client

import (
	"image/color"
	"math"
	"math/rand/v2"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Titanoboa's temple (sim/temple.go), drawn at HD in the manner of the
// rocks (rockart.go): the paving of its courtyard and the coping round its
// black pool laid flat on the ground in baked blocks; the colonnade's broken
// pillars, the columns fallen across the courtyard and the stone serpents'
// heads built of lit stone blocks; the giant dead tree over it all, the
// landmark; and at the camp, an old ring of fire stones and a log.

// Stone of the temple, from COMFY52: a cold grey granite and a warm
// sandstone, laid together, both greened by moss and lichen.
var (
	tplGrey   = mkRamp(0x282e3b, 0x414453, 0x696f80, 0x8d99a4, 0xbac9cd)
	tplWarm   = mkRamp(0x3f333b, 0x533b41, 0x6d4e4b, 0x867066, 0xb49d7e)
	tplMoss   = mkRamp(0x282e3b, 0x343f41, 0x3f4328, 0x5f7132, 0x739a70)
	tplJoint  = mkRamp(0x1a1f2e, 0x282e3b, 0x343f41, 0x3f4328)
	tplJointDark = mkRamp(0x1a1917, 0x24231f, 0x2e2d2b)
	tplFlag      = mkRamp(0x2e2d2b, 0x3e3c38, 0x504d47, 0x645f57, 0x79736a, 0x8f887d, 0xa59d90, 0xb9b1a2) // weathered limestone
	tplBuff      = mkRamp(0x3c3530, 0x4f463e, 0x645849, 0x7a6b58, 0x917f68, 0xa8947a)
	tplEarth  = mkRamp(0x2b222a, 0x3c252b, 0x533b41, 0x684039)
	tplLeaf   = mkRamp(0x5c2b34, 0x8d4830, 0xc57835, 0xe1a845)
	tplBlack  = mkRamp(0x05080c, 0x0a0f16, 0x111a22, 0x1a2a30, 0x24403f)
	tplStoneM = &rkMat{tone: tplGrey, ink: hex(0x1a1f2e), rim: hex(0x414453), glint: hex(0xe5efef)}
	tplWarmM  = &rkMat{tone: tplWarm, ink: hex(0x2b222a), rim: hex(0x533b41), glint: hex(0xeecb90), bias: 0.06}
	tplPillarM = &rkMat{tone: mkRamp(0x343f41, 0x4d5459, 0x696f80, 0x8d99a4, 0xb4b8b0, 0xd5d6cc), ink: hex(0x1a1f2e), rim: hex(0x414453), glint: hex(0xe5efef), bias: 0.24}
)

// templeLook is what the bake of the courtyard needs to know of a temple.
type templeLook struct {
	x, y, cx, cy   float64 // the courtyard's middle and half widths, tiles (sim.ArenaQ's oblong)
	px, py, rx, ry float64 // the black pool
}

// templeOf is the temple of lair l.
func templeOf(x, y, cx, cy, px, py, rx, ry float32) templeLook {
	return templeLook{float64(x), float64(y), float64(cx), float64(cy), float64(px), float64(py), float64(rx), float64(ry)}
}

// poolQ is how far (u, v) lies out from the pool's middle, 1 at its edge.
func (t templeLook) poolQ(u, v float64) float64 {
	return math.Hypot((u-t.px)/t.rx, (v-t.py)/t.ry)
}

// The coping round the pool is this many tiles wide; the ring of radial
// flags round it this much more.
const (
	tplCope = 0.85
	tplRing = 1.6
)

// pavePixel colours image pixel (hx, hy) of the courtyard at (u, v) tiles:
// the paving, the coping round the pool and the black water in it. ok is
// false where none of that lies (outside the courtyard, a flag gone).
func (t templeLook) pavePixel(u, v float64, hx, hy int) (color.RGBA, bool) {
	m := math.Min(t.cx, t.cy)
	d := sim.ArenaQ(sim.LayoutTemple, t.cx, t.cy, u-t.x, v-t.y) * m // how far out, in tiles at its narrower
	rim := m + (vnoise(u*0.9, v*0.9, 911)-0.5)*1.4 // ragged where the jungle takes it back
	if d > rim {
		return color.RGBA{}, false
	}
	q := t.poolQ(u, v)
	rr := math.Min(t.rx, t.ry)
	switch {
	case q < 1:
		return t.water(u, v, q, hx, hy), true
	case q < 1+tplCope/rr:
		return t.coping(u, v, q, hx, hy), true
	case q < 1+(tplCope+tplRing)/rr:
		return t.radial(u, v, q, rr, hx, hy, d, rim)
	}
	return t.flags(u, v, hx, hy, d, rim)
}

// water is the pool: black, deepest in the middle, the sky's grey in it
// towards the far side and under the coping's lip a band of shade.
func (t templeLook) water(u, v, q float64, hx, hy int) color.RGBA {
	l := 0.12 + 0.3*smooth(clamp01((q-0.55)/0.45))
	a := math.Atan2((v-t.py)/t.ry, (u-t.px)/t.rx)
	far := -math.Sin(a) // the far (north) half catches the sky
	if far > 0 {
		l += 0.22 * far * smooth(clamp01((q-0.6)/0.3)) * (0.7 + 0.3*vnoise(u*3, v*7, 913))
	}
	if q > 0.9 && math.Sin(a) > -0.2 { // in the shade of the coping's near lip
		l -= 0.12
	}
	l += (fbm2(u*2.1, v*3.3, 914, 2) - 0.5) * 0.08
	c := rq(tplBlack, l, hx, hy)
	// A dead leaf or two floating.
	if lf := hashf(int(math.Floor(u*5)), int(math.Floor(v*7)), 915); lf < 0.004 && q < 0.9 {
		c = mix(tplLeaf.at(int(lf*700)), c, 0.4)
	}
	return c
}

// coping is the ring of long curb stones round the pool: their tops lit,
// their inner faces dark down to the water on the far side.
func (t templeLook) coping(u, v, q float64, hx, hy int) color.RGBA {
	rr := math.Min(t.rx, t.ry)
	w := (q - 1) * rr / tplCope // 0 at the water, 1 at the paving
	a := math.Atan2((v-t.py)/t.ry, (u-t.px)/t.rx)
	seg := a / (2 * math.Pi) * 22
	j := math.Floor(seg)
	f := seg - j
	id := int(j) + 100
	rp := tplFlag
	if hashf(id, 1, 916) < 0.1 {
		rp = tplBuff
	}
	l := 0.62 + (hashf(id, 2, 916)-0.5)*0.14
	switch {
	case f < 0.035 || f > 0.965 || w > 0.9:
		return rq(tplJoint, 0.4+hashf(hx, hy, 917)*0.3, hx, hy)
	case w < 0.16 && math.Sin(a) < 0: // the inner face of the far coping, seen: dark
		return rq(rp, 0.12+w*1.4, hx, hy)
	case w < 0.2:
		l += 0.22 // the lit lip over the water
	case w > 0.78:
		l -= 0.16
	}
	if f < 0.09 {
		l += 0.1
	} else if f > 0.9 {
		l -= 0.1
	}
	l += (fbm2(u*3, v*4, 918, 2) - 0.5) * 0.16
	if m := vnoise(u*1.3, v*1.6, 919); m > 0.66 && w > 0.3 {
		return rq(tplMoss, 0.3+l*0.6, hx, hy)
	}
	return rq(rp, l, hx, hy)
}

// radial is the ring of flags laid round the coping, set out from the pool
// like the spokes of a wheel, two courses deep.
func (t templeLook) radial(u, v, q, rr float64, hx, hy int, d, rim float64) (color.RGBA, bool) {
	a := math.Atan2((v-t.py)/t.ry, (u-t.px)/t.rx)
	w := ((q-1)*rr - tplCope) / tplRing // 0 to 1 across the ring
	row := 0.0
	if w > 0.5 {
		row = 1
	}
	n := 26.0 + 8*row
	seg := a/(2*math.Pi)*n + row*0.5
	j := math.Floor(seg)
	fu := seg - j
	fv := w*2 - row
	id := int(j)*3 + int(row)*1000 + 7
	wpx := 2 * math.Pi * rr * (1 + w*0.3) / n * TileW
	hpx := tplRing / 2 * TileH
	return t.stone(u, v, hx, hy, id, fu, fv, wpx, hpx, d, rim)
}

// flags is the paving: courses of big flags of many widths, each course of
// its own depth and staggered on the last.
func (t templeLook) flags(u, v float64, hx, hy int, d, rim float64) (color.RGBA, bool) {
	// Courses of 0.95 to 1.35 tiles' depth, laid north to south.
	vv := v + (vnoise(u*0.3, v*0.3, 920)-0.5)*0.25
	base := math.Floor(vv / 1.4)
	row, top, depth := base, 0.0, 1.4
	for k := base - 1; k <= base+1; k++ { // the course this point falls in, each a little deeper or shallower
		t0 := k*1.4 + (hashf(int(k), 9, 921)-0.5)*0.45
		t1 := (k+1)*1.4 + (hashf(int(k+1), 9, 921)-0.5)*0.45
		if vv >= t0 && vv < t1 {
			row, top, depth = k, t0, t1-t0
		}
	}
	fv := (vv - top) / depth
	off := hashf(int(row), 0, 921) * 2.9
	a := u + off
	bound := func(j float64) float64 { return j*2.1 + (hashf(int(j), int(row), 922)-0.5)*1.1 }
	j := math.Floor(a / 2.1)
	if a < bound(j) {
		j--
	} else if a >= bound(j+1) {
		j++
	}
	b0, b1 := bound(j), bound(j+1)
	fu := (a - b0) / (b1 - b0)
	id := int(j)*7919 + int(row)*104729
	return t.stone(u, v, hx, hy, id, fu, fv, (b1-b0)*TileW, depth*TileH, d, rim)
}

// stone colours a pixel of one flag (id) at (fu, fv) across it (0 to 1
// each way, the flag wpx by hpx art pixels): the joint, the flag's worn,
// rounded edges lit from the top left, its weathered face with lichen and
// cracks, moss creeping in from the joints; gone near the ragged rim.
func (t templeLook) stone(u, v float64, hx, hy, id int, fu, fv, wpx, hpx float64, d, rim float64) (color.RGBA, bool) {
	edge := clamp01((d - (rim - 3.5)) / 3.5) // how near the ragged rim
	if hashf(id, 3, 923) < 0.012+0.5*edge*edge*edge {
		// The flag is gone: the earth it lay on, a little dead leaf.
		if hashf(id, 4, 923) < 0.6 && edge < 0.6 {
			e := rq(tplEarth, 0.35+(fbm2(u*4, v*4, 924, 2)-0.5)*0.6, hx, hy)
			if hashf(hx/3, hy/3, 925) < 0.03 {
				e = tplLeaf.at(int(hashf(hx, hy, 926) * 3))
			}
			return e, true
		}
		return color.RGBA{}, false
	}
	// Distances to its four edges, art pixels; the joints a little
	// irregular, the flags' corners chipped.
	el, er := fu*wpx, (1-fu)*wpx
	et, eb := fv*hpx, (1-fv)*hpx
	jw := 0.55 + (vnoise(u*5, v*5, 927)-0.5)*0.5
	near := min(el, er, et, eb)
	if corner := math.Hypot(min(el, er), min(et, eb)); near < jw || corner < jw+1.1 {
		if m := vnoise(u*1.7, v*2.1, 928); m > 0.55-0.25*edge {
			return rq(tplMoss, 0.12+m*0.4, hx, hy), true
		}
		return tplJointDark.at(int(hashf(hx, hy, 929) * 1.99)), true
	}
	rp := tplFlag
	l := 0.5 + (hashf(id, 6, 923)-0.5)*0.1
	if hashf(id, 5, 923) < 0.15 {
		rp = tplBuff // a flag of buff sandstone among the limestone
	}
	// Each a little domed, worn smooth: lighter in the middle, to the light.
	cu, cv := fu-0.5, fv-0.5
	l += 0.07*(1-4*cu*cu)*(1-4*cv*cv) - 0.07*cv - 0.04*cu
	l += (hashf(id, 7, 923) - 0.5) * 0.06 // laid a little askew
	// Its edges worn round: the top and left catch the light, the bottom
	// and right fall into shade, and the stone by a joint is a little dark.
	switch {
	case et-jw < 1.4:
		l += 0.13 * (1 - (et-jw)/1.4)
	case eb-jw < 2:
		l -= 0.16 * (1 - (eb-jw)/2)
	}
	switch {
	case el-jw < 1.1:
		l += 0.06
	case er-jw < 1.6:
		l -= 0.08 * (1 - (er-jw)/1.6)
	}
	// Its face: weathered, pitted.
	l += (fbm2(u*3.1, v*3.7, 930, 2) - 0.5) * 0.12
	l += (vnoise(u*11, v*13, 931) - 0.5) * 0.07
	if hashf(hx/2, hy/2, 932) < 0.02 {
		l -= 0.14
	}
	// A crack across some.
	if hashf(id, 8, 923) < 0.22 {
		if cr := vnoise(u*2.6+float64(id%97), v*3.1, 933); math.Abs(cr-0.5) < 0.011 {
			return tplJointDark.at(1), true
		}
	}
	// Moss creeping out of the joints, more by the rim; pale lichen.
	m := fbm2(u*0.6, v*0.7, 934, 2) + 0.35*edge - near*0.02
	switch {
	case m > 0.76, m > 0.72 && (hx+hy)%2 == 0:
		return rq(tplMoss, 0.18+(m-0.72)*1.6+l*0.3, hx, hy), true
	}
	if lc := vnoise(u*4.3, v*5.1, 935); lc > 0.83 {
		return mix(rq(rp, l+0.08, hx, hy), hex(0xa5c67c), 0.22), true
	}
	return rq(rp, l, hx, hy), true
}

// --- the paving, baked in blocks ----------------------------------------------

// paveBlockTiles is how many tiles a side a baked block of paving is.
const paveBlockTiles = 4

type paveKey struct{ bx, by int }

// templePaving keeps the baked blocks of a temple's courtyard.
type templePaving struct {
	look   templeLook
	blocks map[paveKey]*ebiten.Image // nil: nothing to draw there
	used   map[paveKey]int
}

// paveBake bakes block k of t's paving: an HD image paveBlockTiles tiles a
// side, or nil if none of it is paved.
func (t templeLook) paveBake(k paveKey) *ebiten.Image {
	const W, H = paveBlockTiles * TileW * HDDensity, paveBlockTiles * TileH * HDDensity
	c := newCanvas(W, H)
	any := false
	x0, y0 := k.bx*paveBlockTiles*TileW*HDDensity, k.by*paveBlockTiles*TileH*HDDensity
	for y := 0; y < H; y++ {
		v := (float64(y0+y) + 0.5) / (TileH * HDDensity)
		for x := 0; x < W; x++ {
			u := (float64(x0+x) + 0.5) / (TileW * HDDensity)
			if col, ok := t.pavePixel(u, v, x0+x, y0+y); ok {
				c.px[y*W+x] = col
				any = true
			}
		}
	}
	if !any {
		return nil
	}
	// A soft edge where the paving gives way to the jungle floor.
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if c.px[y*W+x].A == 0 {
				continue
			}
			n := 0
			for _, o := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if nx, ny := x+o[0], y+o[1]; nx >= 0 && ny >= 0 && nx < W && ny < H && c.px[ny*W+nx].A == 0 {
					n++
				}
			}
			if n > 0 {
				c.px[y*W+x] = mix(c.px[y*W+x], tplJoint.at(0), 0.35)
			}
		}
	}
	return c.image()
}

// --- the temple's stones ----------------------------------------------------------

// rkDrum is a drum of a column, sides faces round, standing on z0, h high
// and r round, its axis at (cx, cy).
func rkDrum(cx, cy, z0, r, h, rot float64, sides int) rkPart {
	pl := []rkPlane{{rkDown, -z0}, rkPlaneAt(rkTop, rkVec{cx, cy, z0 + h})}
	for i := range sides {
		a := rot + float64(i)*2*math.Pi/float64(sides)
		pl = append(pl, rkPlaneAt(rkVec{math.Cos(a), math.Sin(a), 0}, rkVec{cx + math.Cos(a)*r, cy + math.Sin(a)*r, 0}))
	}
	return rkPart{pl: pl, c: rkVec{cx, cy, z0 + h/2}, r: r}
}

// rkLog is a drum lying along x, from x0 to x1, r round, its axis at y = cy,
// a little turned about z by yaw.
func rkLog(x0, x1, cy, r, yaw float64, sides int) rkPart {
	c := rkVec{(x0 + x1) / 2, cy, r}
	ax := rkVec{math.Cos(yaw), math.Sin(yaw), 0}
	side := rkVec{-math.Sin(yaw), math.Cos(yaw), 0}
	var pl []rkPlane
	for i := range sides {
		a := float64(i) * 2 * math.Pi / float64(sides)
		n := side.mul(math.Cos(a)).add(rkTop.mul(math.Sin(a)))
		pl = append(pl, rkPlaneAt(n, c.add(n.mul(r))))
	}
	half := (x1 - x0) / 2
	pl = append(pl, rkPlaneAt(ax, c.add(ax.mul(half))), rkPlaneAt(ax.mul(-1), c.add(ax.mul(-half))))
	return rkPart{pl: pl, c: c, r: r}
}

// tplFlutes are the grooves down a column: a darker band between the lit
// ones, by the angle round its axis.
func tplFlutes(cx, cy float64, n int, along bool) rkDecor {
	return func(p, nr rkVec, l float64, pt *rkPart) (ramp, float64, bool) {
		var a float64
		if along { // lying: round its axis along x
			a = math.Atan2(p[2]-pt.c[2], p[1]-pt.c[1])
		} else {
			if nr[2] > 0.8 {
				return nil, 0, false // the top, broken off
			}
			a = math.Atan2(p[1]-cy, p[0]-cx)
		}
		f := a / (2 * math.Pi) * float64(n)
		if f-math.Floor(f) < 0.3 {
			l -= 0.07
		}
		if m := vnoise(p[0]*0.3, p[1]*0.4+p[2]*0.3, 941); m > 0.62 {
			return tplMoss, l*0.8 + 0.05, true
		}
		return pt.mat.tone, l, true
	}
}

// templeSprites is the temple's art, made the first time it is wanted.
type templeSprites struct {
	pillar [3]sprite
	column [2]sprite // lying across three tiles, capital to the left and to the right
	tree   sprite
	snake  [2]sprite // the stone serpent's head, looking right and left
	ring   sprite
	log    sprite
}

var (
	templeOnce sync.Once
	templeArt  *templeSprites
)

func templeSpritesOf() *templeSprites {
	templeOnce.Do(func() {
		a := &templeSprites{}
		inParallel(8, func(i int) {
			switch {
			case i < 3:
				a.pillar[i] = genPillar(i)
			case i == 3:
				a.column[0] = genFallenColumn(0)
				a.column[1] = mirrorSprite(a.column[0])
			case i == 4:
				a.tree = genDeadGiant()
			case i == 5:
				a.snake[0] = genSerpentHead()
				a.snake[1] = mirrorSprite(a.snake[0])
			case i == 6:
				a.ring = genCampRing()
			case i == 7:
				a.log = genCampLog()
			}
		})
		templeArt = a
	})
	return templeArt
}

// genPillar is a pillar of the colonnade still standing: a plinth, drums
// stacked on it, the top broken off at a slant, moss on it and a vine.
func genPillar(v int) sprite { return mkUp(templePillarCanvas(v)) }

// templePillarCanvas draws genPillar.
func templePillarCanvas(v int) (*canvas, float64, float64) {
	s := &rkScene{seed: uint32(3100 + v*17)}
	rng := rand.New(rand.NewPCG(uint64(s.seed), 0x9111))
	mat := tplPillarM
	if v == 2 {
		mat = &rkMat{tone: tplWarm, ink: hex(0x2b222a), rim: hex(0x533b41), glint: hex(0xeecb90), bias: 0.26}
	}
	if p := s.add(rkBlock(rng, rkVec{0, 0, 3}, rkVec{1, 0, 0}, rkVec{0, 1, 0}, rkVec{0, 0, 1}, 11, 9.5, 3, 0.25)); p != nil {
		p.mat, p.decor = mat, rkMoss(s.seed+1)
	}
	if p := s.add(rkBlock(rng, rkVec{0, 0, 7.3}, rkVec{1, 0, 0}, rkVec{0, 1, 0}, rkVec{0, 0, 1}, 9.4, 8.4, 1.3, 0.3)); p != nil {
		p.mat = mat
	}
	drums := []int{4, 3, 5}[v]
	z := 8.6
	for i := range drums {
		h := 11 + rng.Float64()*3
		dx, dy := (rng.Float64()-0.5)*0.8, (rng.Float64()-0.5)*0.6
		p := rkDrum(dx, dy, z, 7.4-0.2*float64(i), h, rng.Float64(), 18)
		if i == drums-1 { // broken off: a slant across its top
			a := rng.Float64() * 2 * math.Pi
			p.pl = append(p.pl, rkPlaneAt(rkVec{math.Cos(a) * 0.7, math.Sin(a) * 0.7, 1}, rkVec{dx, dy, z + h*0.62}))
		}
		if q := s.add(p); q != nil {
			q.mat, q.decor = mat, tplFlutes(dx, dy, 10, false)
			q.nudge = (rng.Float64() - 0.5) * 0.06
			q.crack = 1.2
		}
		z += h + 0.3
	}
	c, ox, oy := s.draw(hdK(), 5)
	tplVine(c, ox, oy, rng, z*rkCos*hdK())
	return c, ox, oy
}

// tplVine hangs a vine of leaves down a pillar from near its top.
func tplVine(c *canvas, ox, oy float64, rng *rand.Rand, top float64) {
	if !c.hd() {
		return
	}
	k := hdK()
	x := ox + (rng.Float64()-0.5)*8*k
	y0 := oy - top*0.9
	n := int(top * 0.7 / k)
	for i := range n {
		y := y0 + float64(i)*k*0.9
		xx := x + math.Sin(float64(i)*0.35)*2*k
		c.dot(xx, y, 0.45*k, jungleRamp.at(1))
		if i%3 == 0 {
			side := float64(1 - 2*(i/3%2))
			leafHD(c, xx+side*1.6*k, y+0.6*k, 2.2*k, 1.1*k, side*0.6+0.4, jungleRamp, 0.5+rng.Float64()*0.4)
		}
	}
}

// genFallenColumn is a column fallen across three tiles: three drums
// lying askew in a row, the capital at the left end, chips about.
func genFallenColumn(v int) sprite { return mkUp(fallenColumnCanvas(v)) }

// fallenColumnCanvas draws genFallenColumn.
func fallenColumnCanvas(v int) (*canvas, float64, float64) {
	s := &rkScene{seed: uint32(3200 + v*29)}
	rng := rand.New(rand.NewPCG(uint64(s.seed), 0x9222))
	const r = 8.2
	x := -28.0
	for i := range 3 {
		ln := 15 + rng.Float64()*3
		yaw := (rng.Float64() - 0.5) * 0.16
		dy := (rng.Float64() - 0.5) * 2.4
		p := rkLog(x, x+ln, dy, r-0.2*float64(i), yaw, 16)
		if q := s.add(p); q != nil {
			q.mat, q.decor = tplPillarM, tplFlutes(0, 0, 12, true)
			q.nudge = (rng.Float64() - 0.5) * 0.08
			q.crack = 1.4
		}
		x += ln + 0.8 + rng.Float64()*1.4
	}
	// The capital, tumbled at the foot end: a block over a short neck.
	if p := s.add(rkBlock(rng, rkVec{-34, 0.5, 6.6}, rkVec{1, 0.1, 0}, rkVec{-0.1, 1, 0}, rkVec{0, 0, 1}, 6, 10, 6.6, 0.35)); p != nil {
		p.mat, p.decor = tplWarmM, rkMoss(s.seed+3)
	}
	for i := range 4 { // chips knocked off it as it fell
		cx, cy := -20+rng.Float64()*50, 7+rng.Float64()*4
		if p := s.add(rkNodule(rng, rkVec{cx, cy, 1}, 1.4+rng.Float64(), 1.1, 1, 7, 0.3)); p != nil {
			p.mat = tplStoneM
			_ = i
		}
	}
	c, ox, oy := s.draw(hdK(), 5)
	return c, ox, oy
}

// genSerpentHead is a stone serpent's head at the pool's edge, carved in
// blocks, looking out over the water, moss in its folds.
func genSerpentHead() sprite { return mkUp(serpentCanvas()) }

// serpentCanvas draws genSerpentHead.
func serpentCanvas() (*canvas, float64, float64) {
	s := &rkScene{seed: 3300}
	rng := rand.New(rand.NewPCG(3300, 0x9333))
	add := func(p rkPart, m *rkMat) {
		if q := s.add(p); q != nil {
			q.mat, q.decor = m, rkMoss(s.seed+uint32(len(s.parts)))
		}
	}
	add(rkBlock(rng, rkVec{0, 0, 3}, rkVec{1, 0, 0}, rkVec{0, 1, 0}, rkVec{0, 0, 1}, 7, 6, 3, 0.2), tplStoneM)
	add(rkBlock(rng, rkVec{1, 0, 9}, rkVec{1, 0, 0.25}, rkVec{0, 1, 0}, rkVec{-0.25, 0, 1}, 6.5, 5, 3.5, 0.45), tplWarmM) // the skull
	add(rkBlock(rng, rkVec{8, 0, 8}, rkVec{1, 0, -0.15}, rkVec{0, 1, 0}, rkVec{0.15, 0, 1}, 4.5, 3.6, 2.4, 0.5), tplWarmM) // the snout
	add(rkBlock(rng, rkVec{6.5, 0, 4.6}, rkVec{1, 0, 0.1}, rkVec{0, 1, 0}, rkVec{-0.1, 0, 1}, 4.4, 3.4, 1.3, 0.4), tplStoneM) // the jaw
	for _, side := range []float64{-1, 1} { // brow ridges
		add(rkNodule(rng, rkVec{2.5, side * 3.4, 12}, 2.4, 1.2, 1.1, 8, 0.15), tplWarmM)
	}
	c, ox, oy := s.draw(hdK(), 5)
	if c.hd() { // its eyes, worn hollows with a dark gleam
		k := hdK()
		c.dot(ox+3.5*k, oy-12.5*k, 0.9*k, hex(0x1a1f2e))
		c.dot(ox+3.8*k, oy-12.8*k, 0.3*k, hex(0x63c4cc))
	}
	return c, ox, oy
}

// genCampRing is the ring of stones of an old fire, the ash in it, the
// ends of two sticks charred black.
func genCampRing() sprite { return mkUp(campRingCanvas()) }

// campRingCanvas draws genCampRing.
func campRingCanvas() (*canvas, float64, float64) {
	s := &rkScene{seed: 3400}
	rng := rand.New(rand.NewPCG(3400, 0x9444))
	for i := range 9 {
		a := float64(i)*2*math.Pi/9 + rng.Float64()*0.3
		if p := s.add(rkNodule(rng, rkVec{math.Cos(a) * 9, math.Sin(a) * 7, 1.8}, 2.8+rng.Float64(), 2.4, 2.1, 9, 0.25)); p != nil {
			p.mat = tplStoneM
		}
	}
	c, ox, oy := s.draw(hdK(), 5)
	if c.hd() {
		k := hdK()
		bg := newCanvas(c.w, c.h)
		bg.dens = c.dens
		bg.ellipse(ox, oy-1*k, 8*k, 5.5*k, func(x, y int, nx, ny float64) color.RGBA {
			return rq(mkRamp(0x1a1f2e, 0x282e3b, 0x414453, 0x696f80), 0.35+(hashf(x, y, 951)-0.5)*0.5-0.3*(nx*nx+ny*ny), x, y)
		})
		for _, st := range [][4]float64{{-5, 0, 4, -2}, {-3, 2, 5, -0.5}, {-1, -2.5, 1.5, 1.5}} {
			bg.line(ox+st[0]*k, oy+st[1]*k-k, ox+st[2]*k, oy+st[3]*k-k, func(x, y int, t float64) color.RGBA {
				if t > 0.7 {
					return hex(0x684039)
				}
				return hex(0x1a1f2e)
			})
		}
		bg.over(c, 0, 0)
		c = bg
	}
	return c, ox, oy
}

// genCampLog is a fallen log to sit on, mossed along its top.
func genCampLog() sprite { return mkUp(campLogCanvas()) }

// campLogCanvas draws genCampLog.
func campLogCanvas() (*canvas, float64, float64) {
	c := hdCanvas(40, 18)
	k := hdK3
	l := limb{4 * k, 13 * k, 20 * k, 12.5 * k, 36 * k, 13.5 * k, 3.6 * k, 3.2 * k}
	drawLimbHD(c, l, barkOak, 0.75, 961)
	for _, e := range [][2]float64{{4, 13}, {36, 13.5}} { // its cut ends, the rings
		c.ellipse(e[0]*k, e[1]*k, 2.6*k, 3.2*k, func(x, y int, nx, ny float64) color.RGBA {
			r := math.Hypot(nx, ny)
			if math.Mod(r*4, 1) < 0.25 {
				return hex(0x684039)
			}
			return hex(0xb49d7e)
		})
	}
	for x := 6 * k; x < 34*k; x++ {
		if vnoise(x*0.06, 1, 962) > 0.45 {
			c.dot(x, 10*k+math.Sin(x*0.05)*k*0.5, 0.9*k, tplMoss.at(2+int(hashf(int(x), 0, 963)*2)))
		}
	}
	c.outline(0.6)
	return c, 20 * k, 14 * k
}

// genDeadGiant is the giant dead tree over the temple: bleached and split,
// its limbs bare and twisted, broken off here and there, beards of grey moss
// hanging from them, its roots gripping the stones of the fallen wall at its
// foot. Seven times as tall as a man: seen over the canopy from far off.
func genDeadGiant() sprite { return trimmed(deadGiantCanvas()) }

// deadGiantCanvas draws genDeadGiant.
func deadGiantCanvas() (*canvas, float64, float64) {
	rng := rand.New(rand.NewPCG(3500, 0x9555))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	W, H := 150, 250
	c := markHD(newCanvas(int(float64(W)*k), int(float64(H)*k)))
	bx, by := float64(c.w)/2, float64(c.h)-12*k
	// The stones of the old wall it split, under its roots.
	stones := &rkScene{seed: 3501}
	srng := rand.New(rand.NewPCG(3501, 0x9556))
	for i := range 7 {
		x := -26 + float64(i)*8.5 + srng.Float64()*2
		if p := stones.add(rkBlock(srng, rkVec{x, srng.Float64()*3 - 1, 3.2}, rkVec{1, (srng.Float64() - 0.5) * 0.3, 0}, rkVec{0, 1, 0}, rkVec{0, 0, 1},
			3.6+srng.Float64(), 3.2, 3.2+srng.Float64()*1.4, 0.3)); p != nil {
			p.mat, p.decor = tplStoneM, rkMoss(3502+uint32(i))
		}
	}
	stones.fit(9, 1)
	sc, sox, soy := stones.draw(k, 5)
	girth := 9.5 * k
	top := by - 150*k
	trunk := limb{bx, by, bx - 6*k, by - 80*k, bx + 4*k, top, girth, girth * 0.5}
	tk := trunkFrom(trunk, barkDead, barkGrey, 0.9, 3503)
	// The limbs: three great ones out of the crown, each forking twice,
	// twisting up and out; some snapped short.
	var limbs []limb
	var grow func(x, y, ang, ln, w float64, depth int)
	grow = func(x, y, ang, ln, w float64, depth int) {
		bend := r(-0.5, 0.5)
		x1, y1 := x+math.Cos(ang)*ln, y+math.Sin(ang)*ln
		cx, cy := x+math.Cos(ang+bend)*ln*0.55, y+math.Sin(ang+bend)*ln*0.55
		w1 := w * 0.62
		limbs = append(limbs, limb{x, y, cx, cy, x1, y1, w, w1})
		if depth == 0 || (depth < 2 && rng.Float64() < 0.22) {
			return // snapped off, or the end of it
		}
		for i := range 2 + depth%2 {
			a := ang + r(-0.75, 0.75) + float64(i-1)*0.35
			grow(x1, y1, a, ln*r(0.55, 0.75), w1, depth-1)
		}
	}
	for i, a := range []float64{-2.35, -1.6, -0.85} {
		tx, ty := trunk.at(0.82 + 0.06*float64(i%2))
		grow(tx, ty, a+r(-0.15, 0.15), r(42, 56)*k, girth*0.55, 3)
	}
	lx, ly := trunk.at(0.5) // a low limb, long dead, snapped
	limbs = append(limbs, limb{lx, ly, lx - 10*k, ly - 8*k, lx - 16*k, ly - 22*k, girth * 0.42, girth * 0.3})
	for _, l := range limbs {
		drawLimbHD(c, l, barkGrey, 0.8, 3504)
	}
	tk.draw(c, 6, girth*2.6)
	// Roots gripping the stones: the stones laid over the roots' foot, then
	// roots over them again.
	c.over(sc, int(bx-sox), int(by-soy+2*k))
	for i := range 5 {
		x0 := bx + float64(i-2)*6*k
		x1 := bx + float64(i-2)*17*k + r(-4, 4)*k
		drawLimbHD(c, limb{x0, by - 10*k, (x0 + x1) / 2, by - 12*k, x1, by + r(-1, 4)*k, girth * 0.32, girth * 0.12}, barkGrey, 0.75, 3505+uint32(i))
	}
	// Beards of grey moss hanging from the limbs.
	for i, l := range limbs {
		if i%2 == 1 {
			continue
		}
		for j := range 3 {
			x, y := l.at(0.3 + 0.3*float64(j))
			ln := r(6, 16) * k
			for d := 0.0; d < ln; d += k * 0.5 {
				c.dot(x+math.Sin(d*0.08+float64(i))*1.2*k, y+d, (0.55-0.4*d/ln)*k, mkRamp(0x3f4328, 0x4d6659, 0x5f7132, 0x739a70).at(int(hashf(i, int(d), 3506)*4)))
			}
		}
	}
	// A hollow in the trunk, black.
	hx, hy := trunk.at(0.3)
	c.ellipse(hx+2*k, hy, 3*k, 6*k, func(x, y int, nx, ny float64) color.RGBA {
		if c.at(x, y).A == 0 {
			return color.RGBA{}
		}
		return mix(hex(0x1a1f2e), hex(0x2b222a), clamp01(ny+0.5))
	})
	c.outline(0.75)
	return c, bx, by
}
