package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The natural caves of worlds made since sim.StyleCaves, as the ground bake
// draws them. A chunk of the caves is baked like the ground (at groundDens,
// on worker goroutines, cached), but by its own rules:
//
//   - The rock's edge is the tiles' smoothed and pushed about by noise, so
//     chambers and tunnels come out round and ragged rather than square.
//   - Rock with floor below it shows a face as tall as a man and a half,
//     inside its own tiles (so nothing walks behind it), built of flat facets
//     each turned its own way to the light, cracked between, darker under the
//     overhang at the top and in the crook at the foot. Each rock has its own
//     facets: pale rounded limestone with flowstone, mossy dark stone, red
//     sandstone in courses, black basalt in columns, blue ice, dripping mud
//     rock.
//   - Above the faces the rock's top is the dark of the cave's roof: a lit
//     rim where it breaks off, rough stone, then black further in.
//   - The floor is the rock's floor (flags of limestone, sand, cinders,
//     frost, mud), with grit and pebbles, damp patches and puddles, darker by
//     the walls and deep in the shadow at the foot of a face, and the main
//     route worn smoother and paler down the middle.
//
// What a tile is (its chamber, its rock, the route) comes from the cave cells
// the server sends (sim.CaveCell).

// caveTerrain is what the client knows of the natural caves.
type caveTerrain struct {
	cells  []sim.CaveCell // per tile from row from down
	from   int            // the surface's height: the first row the cells are for
	levels int
}

// setCaves gives t the natural caves' cells, those of the rows from
// surfaceH down; nil or the wrong size leaves t without them.
func (t *terrain) setCaves(cells []sim.CaveCell, surfaceH int) {
	if surfaceH <= 0 || len(cells) != t.w*(t.h-surfaceH) || len(cells) == 0 {
		t.cave = caveTerrain{}
		return
	}
	t.cave = caveTerrain{cells: cells, from: surfaceH, levels: sim.CaveLevelsOf(t.h, surfaceH)}
}

// caveCell is the cave cell of tile (tx, ty); zero off the caves.
func (t *terrain) caveCell(tx, ty int) sim.CaveCell {
	if t.cave.cells == nil || tx < 0 || tx >= t.w || ty < t.cave.from || ty >= t.h {
		return 0
	}
	return t.cave.cells[(ty-t.cave.from)*t.w+tx]
}

// caveChunk reports whether chunk k is baked as the natural caves.
func (t *terrain) caveChunk(k chunkKey) bool {
	return t.cave.cells != nil && k.cy*chunkTiles >= t.cave.from
}

// caveOpen reports whether tile k of the caves is not rock: floor, water,
// lava or ice.
func caveOpen(k sim.Tile) bool { return k != sim.TileMountain }

const (
	caveFaceH = 34 * groundDens // image pixels: how tall a cave wall's face is
	caveMX    = 48              // pixels of margin either side of a chunk its fields are worked out in
	caveMYT   = 48              // above it
	caveMYB   = caveFaceH + 48  // and below it, where the faces' feet are
)

// caveMat is how one rock of the caves looks.
type caveMat struct {
	face    ramp       // the wall's stone, dark to light
	floor   ramp       // the floor, dark to light (fine)
	grit    ramp       // the pebbles and grit on it
	cell    [2]float64 // pixels: how big the facets of the face are, across and down
	strata  []ramp     // the face in courses of these instead, by turns
	columns bool       // the facets stand in tall columns
	moss    float64    // how much moss on the faces and floor
	wet     float64    // how damp the floor is: patches and puddles
	flow    float64    // pale flowstone curtains down the faces
	gloss   float64    // glints on the faces: wet stone, glassy rock, ice
	lift    float64    // tones lighter than the rest
}

var (
	cmMoss   = mkRamp(0x3f4328, 0x5f7132, 0x94ad39, 0xc2d64f)
	cmPuddle = mkRamp(0x0d1018, 0x1a1f2e, 0x1e314b, 0x2f4c6c, 0x3d80a3)
	cmGlass  = mkRamp(0x1a1f2e, 0x2f193e, 0x39275e, 0x5c3a6f, 0x7e659b, 0xb8a1c2)
)

// caveMats are the rocks of the caves, by sim.CaveRock, in the tones of
// COMFY52 like the boulders (see rockart.go).
var caveMats = [sim.NumCaveRocks]caveMat{
	sim.RockLimestone: {
		face:  mkRamp(0x1a1f2e, 0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b, 0x867066, 0xb49d7e, 0xc4c6b8),
		floor: mkRamp(0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b, 0x867066, 0xb49d7e).sub(3),
		grit:  mkRamp(0x2b222a, 0x533b41, 0x867066, 0xb49d7e, 0xc4c6b8),
		cell:  [2]float64{34, 50}, wet: 0.45, flow: 0.55, gloss: 0.02,
	},
	sim.RockMossy: {
		face:  mkRamp(0x1a1f2e, 0x282e3b, 0x343f41, 0x414453, 0x4d6659, 0x696f80, 0x739a70, 0x8d99a4),
		floor: mkRamp(0x1a1f2e, 0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b, 0x867066).sub(3),
		grit:  mkRamp(0x1a1f2e, 0x343f41, 0x4d6659, 0x696f80, 0x8d99a4),
		cell:  [2]float64{30, 44}, moss: 0.7, wet: 0.6, gloss: 0.04,
	},
	sim.RockSandstone: {
		face:  mkRamp(0x3a1a12, 0x6a2f1d, 0x8c4227, 0xa95732, 0xc06f3f, 0xd38a50, 0xe3a766, 0xf2c98a),
		floor: mkRamp(0x3c252b, 0x5c2b34, 0x8d4830, 0xb77862, 0xe1a845, 0xeecb90).sub(3),
		grit:  mkRamp(0x3c252b, 0x8d4830, 0xb77862, 0xeecb90, 0xe3e6ac),
		cell:  [2]float64{64, 22}, strata: rockMats[matSandstone].strata, wet: 0.05,
	},
	sim.RockBasalt: {
		face:  mkRamp(0x0b0d14, 0x1a1f2e, 0x282e3b, 0x343f41, 0x414453, 0x696f80, 0x8d99a4, 0xbac9cd),
		floor: mkRamp(0x1a1f2e, 0x2b222a, 0x282e3b, 0x3f333b, 0x414453, 0x696f80).sub(3),
		grit:  mkRamp(0x1a1f2e, 0x2b222a, 0x414453, 0x696f80, 0x8d99a4),
		cell:  [2]float64{18, 96}, columns: true, wet: 0.1, gloss: 0.06,
	},
	sim.RockIce: {
		face:  mkRamp(0x1a1f2e, 0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5, 0xbac9cd, 0xe5efef),
		floor: mkRamp(0x2f4c6c, 0x414453, 0x696f80, 0x8d99a4, 0xbac9cd, 0xe5efef).sub(3),
		grit:  mkRamp(0x2f4c6c, 0x696f80, 0x8d99a4, 0xbac9cd, 0xe5efef),
		cell:  [2]float64{36, 58}, wet: 0.2, gloss: 0.12, lift: 0.4,
	},
	sim.RockMud: {
		face:  mkRamp(0x0d1018, 0x1a1f2e, 0x2b222a, 0x343f41, 0x3f4328, 0x4d6659, 0x5f7132, 0x739a70),
		floor: mkRamp(0x1a1f2e, 0x2b222a, 0x3f333b, 0x3f4328, 0x533b41, 0x5f7132).sub(3),
		grit:  mkRamp(0x1a1f2e, 0x2b222a, 0x3f333b, 0x533b41, 0x6d4e4b),
		cell:  [2]float64{32, 42}, moss: 0.35, wet: 0.9, gloss: 0.05,
	},
}

// caveBake is one chunk of the caves being baked: the fields it is drawn
// from, worked out for the chunk and a margin round it.
type caveBake struct {
	t      *terrain
	x0, y0 int // the chunk's corner, in image pixels of the world
	gw, gh int // the field grid: the chunk and its margins
	rock   []bool
	foot   []uint16 // on rock: pixels down to the floor at its foot (caveFar for none in the grid)
	up     []uint16 // off the rock: pixels up to the rock above (caveFar for none)
	toRock []uint16 // the chamfer distance (3 to a pixel) from each pixel to the rock
	toOpen []uint16 // and to the open cave

	tx0, ty0, tw, th int // the tiles round the chunk
	trock            []sim.CaveRock
	tzone            []sim.CaveZone
	troute           []bool
}

const caveFar = 0xffff

// bakeCaveChunk bakes chunk k of the caves: its pixels, chunkPW by chunkPH.
func (t *terrain) bakeCaveChunk(k chunkKey) []byte {
	if !t.caveOpenNear(k) {
		return deepRockChunk(k) // solid rock all round: the dark between the systems
	}
	b := t.newCaveBake(k)
	c := newCanvas(chunkPW, chunkPH)
	for y := 0; y < chunkPH; y++ {
		for x := 0; x < chunkPW; x++ {
			c.px[y*chunkPW+x] = b.pixel(x, y)
		}
	}
	return c.bytes()
}

func (t *terrain) newCaveBake(k chunkKey) *caveBake {
	b := &caveBake{t: t, x0: k.cx * chunkPW, y0: k.cy * chunkPH}
	b.gw, b.gh = chunkPW+2*caveMX, chunkPH+caveMYT+caveMYB
	// The tiles round the chunk, with the rock and zone of each: an open
	// tile's own, a rock tile's that of the nearest open one.
	b.tx0, b.ty0 = fdiv(b.x0-caveMX, hdTW)-3, fdiv(b.y0-caveMYT, hdTH)-3
	b.tw = fdiv(b.x0+chunkPW+caveMX, hdTW) + 4 - b.tx0
	b.th = fdiv(b.y0+chunkPH+caveMYB, hdTH) + 4 - b.ty0
	n := b.tw * b.th
	b.trock, b.tzone, b.troute = make([]sim.CaveRock, n), make([]sim.CaveZone, n), make([]bool, n)
	seen := make([]bool, n)
	var q []int
	solid := make([]bool, n)
	for i := range n {
		tx, ty := b.tx0+i%b.tw, b.ty0+i/b.tw
		if !caveOpen(t.tile(tx, ty)) || ty < t.cave.from {
			solid[i] = true
			continue
		}
		c := t.caveCell(tx, ty)
		b.trock[i], b.tzone[i], b.troute[i], seen[i] = c.Rock(), c.Zone(), c.Route(), true
		q = append(q, i)
	}
	for j := 0; j < len(q); j++ {
		x, y := q[j]%b.tw, q[j]/b.tw
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= b.tw || ny >= b.th {
					continue
				}
				if i := ny*b.tw + nx; !seen[i] {
					seen[i] = true
					b.trock[i], b.tzone[i] = b.trock[q[j]], b.tzone[q[j]]
					q = append(q, i)
				}
			}
		}
	}
	// The rock, its edge smoothed between the tiles and pushed about.
	b.rock = make([]bool, b.gw*b.gh)
	for gy := 0; gy < b.gh; gy++ {
		v := (float64(b.y0-caveMYT+gy) + 0.5) / hdTH
		for gx := 0; gx < b.gw; gx++ {
			u := (float64(b.x0-caveMX+gx) + 0.5) / hdTW
			b.rock[gy*b.gw+gx] = b.rockAt(u, v, solid)
		}
	}
	b.foot, b.up = make([]uint16, b.gw*b.gh), make([]uint16, b.gw*b.gh)
	for gx := 0; gx < b.gw; gx++ {
		f := uint16(caveFar)
		for gy := b.gh - 1; gy >= 0; gy-- {
			i := gy*b.gw + gx
			switch {
			case !b.rock[i]:
				f = 0
			case f < caveFar:
				f++
			}
			b.foot[i] = f
		}
		u := uint16(caveFar)
		for gy := 0; gy < b.gh; gy++ {
			i := gy*b.gw + gx
			switch {
			case b.rock[i]:
				u = 0
			case u < caveFar:
				u++
			}
			b.up[i] = u
		}
	}
	b.toRock = chamfer(b.gw, b.gh, func(i int) bool { return b.rock[i] })
	b.toOpen = chamfer(b.gw, b.gh, func(i int) bool { return !b.rock[i] })
	return b
}

// rockAt reports whether (u, v), in tiles, is rock: the tiles' rock eased
// between their middles, its edge wavering.
func (b *caveBake) rockAt(u, v float64, solid []bool) bool {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0)-b.tx0, int(y0)-b.ty0
	m := func(tx, ty int) float64 {
		if tx < 0 || ty < 0 || tx >= b.tw || ty >= b.th || solid[ty*b.tw+tx] {
			return 1
		}
		return 0
	}
	fx, fy := smooth(x-x0), smooth(y-y0)
	f := lerpf(lerpf(m(ix, iy), m(ix+1, iy), fx), lerpf(m(ix, iy+1), m(ix+1, iy+1), fx), fy)
	if f == 0 || f == 1 {
		return f == 1
	}
	n := (fbm2(u*0.9, v*1.1, 3101, 2)-0.5)*0.5 + (vnoise(u*3.6, v*4.1, 3102)-0.5)*0.14
	return f+n > 0.5
}

// tileAt is index of tile (tx, ty) in the bake's tile grid, clamped.
func (b *caveBake) tileAt(tx, ty int) int {
	x, y := max(0, min(b.tw-1, tx-b.tx0)), max(0, min(b.th-1, ty-b.ty0))
	return y*b.tw + x
}

// pixel colours image pixel (x, y) of the chunk.
func (b *caveBake) pixel(x, y int) color.RGBA {
	gi := (y+caveMYT)*b.gw + x + caveMX
	hx, hy := b.x0+x, b.y0+y
	u, v := (float64(hx)+0.5)/hdTW, (float64(hy)+0.5)/hdTH
	if b.rock[gi] {
		foot := int(b.foot[gi])
		if foot <= caveFaceH {
			ti := b.tileAt(int(math.Floor(u)), int(math.Floor(v+float64(foot)/hdTH)))
			return b.facePixel(&caveMats[b.trock[ti]], gi, hx, hy, foot)
		}
		ti := b.tileAt(int(math.Floor(u)), int(math.Floor(v)))
		return b.topPixel(&caveMats[b.trock[ti]], gi, hx, hy, u, v)
	}
	// The floor's rock, its borders ragged.
	ju, jv := (vnoise(u*1.7, v*1.9, 3110)-0.5)*0.8, (vnoise(u*1.7, v*1.9, 3111)-0.5)*0.8
	ti := b.tileAt(int(math.Floor(u+ju)), int(math.Floor(v+jv)))
	return b.floorPixel(&caveMats[b.trock[ti]], b.tzone[ti], gi, hx, hy, u, v)
}

// facePixel colours a pixel of a wall's face, foot pixels above its foot.
func (b *caveBake) facePixel(m *caveMat, gi, hx, hy, foot int) color.RGBA {
	d := float64(caveFaceH - foot) // pixels down from the lip
	// The lip: the edge of the rock's top, catching the light, ragged.
	lip := 3 + 3*vnoise(float64(hx)/9, float64(hy)/31, 3120)
	if d < lip {
		if d < 1.2 {
			return m.face.at(len(m.face) - 2)
		}
		return m.face.at(len(m.face) - 3)
	}
	// Which way the face turns: its foot running up to the right is a face
	// turned east, out of the light.
	turn := 0.0
	gx := gi % b.gw
	if gx >= 9 && gx < b.gw-9 {
		l, r := b.footRow(gi-9), b.footRow(gi+9)
		if l >= 0 && r >= 0 {
			turn = clamp((l-r)/18, -1.2, 1.2) // >0: turned east
		}
	}
	fx, fy := float64(hx), float64(hy)
	tone := 0.46 + m.lift*0.1 - 0.2*turn
	rp := m.face
	ledgeTop := false
	switch {
	case m.columns:
		// Basalt stands in columns, each a flat face of its own.
		cw, ch := m.cell[0], m.cell[1]
		e, id, nx, _ := plates(fx/cw, fy/ch, 3121)
		edge := e * cw
		tone += (idf(id)-0.5)*0.4 - 0.1
		switch {
		case edge < 1.2:
			return m.face.at(0)
		case edge < 3 && nx < 0:
			tone += 0.2 // the lit edge of a column
		case edge < 3:
			tone -= 0.12
		}
	default:
		// Facets: the rock broken into planes, taller than wide, each turned
		// its own way to the light from the upper left, a lit bevel along the
		// edges that face it and a shaded one along the rest.
		cw, ch := m.cell[0], m.cell[1]
		wx := fx/cw + 0.35*vnoise(fx/40, fy/40, 3132)
		e, id, nx, ny := plates(wx, fy/ch, 3121)
		edge := e * cw
		fnx := (idf(id) - 0.5) * 1.3
		fny := (idf(id*2654435761) - 0.5) * 0.7
		tone += -0.32*fnx - 0.2*fny + (vnoise(fx/6, fy/14, 3133)-0.5)*0.12
		switch {
		case edge < 1.3 && idf(id^0x5bd1e995) < 0.45:
			tone -= 0.34 // a crack between two
		case edge < 2.6 && nx+ny > 0:
			tone -= 0.16
		case edge < 3.6 && nx+ny < 0:
			tone += 0.14
		}
		// Ledges across it: a lit top, a dark underside.
		if l := vnoise(fx/70, fy/11, 3123) - 0.5; math.Abs(l) < 0.022 {
			if l < 0 {
				tone += 0.2
				ledgeTop = true
			} else {
				tone -= 0.3
			}
		}
		// A few long cracks running down.
		cx := fdiv(hx, 21)
		y := hy + int(hashf(cx, 5, 3124)*40)
		if h := hash2(int32(cx), int32(fdiv(y, 40)), 3125); h&0xff < 40 {
			x := cx*21 + 4 + int(h>>8&7) + int(vnoise(fx/9, fy/7, 3126)*4)
			if y-fdiv(y, 40)*40 < 18+int(h>>12&15) {
				switch hx - x {
				case 0:
					tone -= 0.38
				case 1:
					tone += 0.1
				}
			}
		}
	}
	if m.strata != nil {
		c := int(math.Floor((fy + 6*vnoise(fx/50, fy/200, 3127)) / m.cell[1]))
		rp = m.strata[(c%len(m.strata)+len(m.strata))%len(m.strata)]
		if b := math.Mod(fy+6*vnoise(fx/50, fy/200, 3127), m.cell[1]); b < 1.5 {
			tone -= 0.25 // the break between courses
		}
	}
	// Flowstone: pale curtains running down the face.
	if m.flow > 0 {
		s := vnoise(fx/7, fy/80, 3128)
		if s > 1-m.flow*0.45 {
			tone += 0.2 * (s - (1 - m.flow*0.45)) / (m.flow * 0.45)
		}
	}
	// Damp streaks.
	if s := vnoise(fx/4, fy/60, 3129); s < 0.2 {
		tone -= 0.1
	}
	// Moss on the ledges and up under the lip.
	if m.moss > 0 && (ledgeTop || d < 16) && vnoise(fx/6, fy/5, 3130) < m.moss*0.75 {
		return cmMoss.at(int(clamp01(0.15+0.5*tone+0.25*hashf(hx, hy, 3131)) * 2.4))
	}
	// Darker under the overhang and in the crook at the foot.
	if d < 22 {
		tone -= 0.32 * (1 - d/22)
	}
	if foot < 10 {
		tone -= 0.3 * (1 - float64(foot)/10)
	}
	if hashf(hx, hy, 3127) < m.gloss {
		tone += 0.35 // a glint
	}
	i := int(math.Floor(clamp01(tone)*float64(len(rp)-1) + (dither(hx, hy) - 0.5)))
	return rp.at(max(1, min(len(rp)-1, i)))
}

// footRow is the grid row of the foot of the face pixel gi lies in, -1 if
// it lies in none.
func (b *caveBake) footRow(gi int) float64 {
	if !b.rock[gi] || b.foot[gi] > caveFaceH {
		return -1
	}
	return float64(gi/b.gw) + float64(b.foot[gi])
}

// topPixel colours a pixel of the rock's top: the roof of the cave seen
// from the passages, black further in.
func (b *caveBake) topPixel(m *caveMat, gi, hx, hy int, u, v float64) color.RGBA {
	in := float64(b.toOpen[gi]) / 3 // pixels to the open cave
	over := int(b.foot[gi]) - caveFaceH
	switch {
	case over < 4:
		return m.face.at(1) // the thickness of the rock behind the lip
	case in < 1.7:
		return m.face.at(2) // the rim where it breaks off over the floor beyond
	}
	l := 0.06 + 0.08*vnoise(u*0.9, v*1.1, 3131) + 0.16*(1-smooth(clamp01(in/22)))
	return caveDeepHD.g(l, hx, hy)
}

// floorPixel colours a pixel of the cave's floor in zone z.
func (b *caveBake) floorPixel(m *caveMat, z sim.CaveZone, gi, hx, hy int, u, v float64) color.RGBA {
	near := float64(b.toRock[gi]) / 3 // pixels to the rock
	t := 0.5 + (fbm2(u*0.5, v*0.6, 3140, 3)-0.5)*0.8
	// Slabs of the rock, cracked apart here and there.
	e, id, nx, ny := plates(u*1.9, v*2.4, 3141)
	t += (idf(id) - 0.5) * 0.08
	if vnoise(u*0.7, v*0.8, 3151) > 0.62 {
		switch w := e * hdTW / 1.9; {
		case w < 0.7:
			t -= 0.16
		case w < 1.8 && nx+ny < 0:
			t += 0.05
		}
	}
	// Damp patches, and small puddles in the hollows of the damp.
	damp := fbm2(u*0.33, v*0.38, 3142, 2) + m.wet*0.12
	if damp > 0.62 {
		t -= 0.1 * min(1, (damp-0.62)/0.06)
		if c, ok := b.puddle(hx, hy, m.wet*(damp-0.62)/0.3, near); ok {
			return c
		}
	}
	// The main route, worn smooth.
	worn := b.routeAt(u, v)
	worn = clamp01((worn + 0.3*(vnoise(u*2.2, v*2.6, 3143)-0.5) - 0.45) * 3)
	t += 0.07 * worn
	// Grit and pebbles, fewer where it is worn.
	if worn < 0.6 {
		switch p, l := pebble(hx, hy, 15, 12, 3144, 0.09, 1, 2.6); p {
		case pebStone:
			return m.grit.at(1 + int(l*2.7))
		case pebShadow:
			t -= 0.14
		}
		switch p, l := pebble(hx+11, hy+7, 44, 34, 3157, 0.16, 3, 6.5); p {
		case pebStone:
			return m.grit.at(int(l * 3.6))
		case pebShadow:
			t -= 0.18
		}
		if h := hashf(hx, hy, 3145); h < 0.05 {
			t += 0.1
		} else if h < 0.09 {
			t -= 0.08
		}
	}
	// Moss where it is damp, and by the walls.
	if m.moss > 0 && (damp > 0.58 || near < 14) && vnoise(u*3.4, v*3.9, 3146) < m.moss*0.22 {
		k := clamp01(0.3 + (vnoise(u*9, v*10, 3147)-0.5)*1.1)
		c := cmMoss.at(int(k * 1.99))
		return mix(c, colInk, 0.15+b.shade(gi, near)*0.8)
	}
	// Daylight by the way up: a little green creeps in.
	if z == sim.ZoneEntrance && vnoise(u*2.4, v*2.7, 3148) > 0.74 {
		c := cmMoss.at(int(hashf(hx, hy, 3149) * 2.2))
		return mix(c, colInk, 0.1+b.shade(gi, near)*0.8)
	}
	t -= b.shade(gi, near)
	return m.floor.g(t, hx, hy)
}

// shade is how much darker the floor at gi is for the walls near: by all
// of them, and deep at the foot of a face.
func (b *caveBake) shade(gi int, near float64) float64 {
	k := 0.42 * math.Exp(-near/24)
	if up := float64(b.up[gi]); up < 30 {
		k += 0.3 * (1 - up/30)
	}
	return k
}

// routeAt is how much (u, v) lies on the main route, eased between tiles.
func (b *caveBake) routeAt(u, v float64) float64 {
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	r := func(tx, ty int) float64 {
		if b.troute[b.tileAt(tx, ty)] {
			return 1
		}
		return 0
	}
	ix, iy := int(x0), int(y0)
	fx, fy := smooth(x-x0), smooth(y-y0)
	return lerpf(lerpf(r(ix, iy), r(ix+1, iy), fx), lerpf(r(ix, iy+1), r(ix+1, iy+1), fx), fy)
}

// puddle is a pixel of a puddle in a hollow of the floor: dark still water
// with the rock's light in its rim.
func (b *caveBake) puddle(hx, hy int, chance, near float64) (color.RGBA, bool) {
	const cw, ch = 46, 34
	cx, cy := fdiv(hx, cw), fdiv(hy, ch)
	h := hash2(int32(cx), int32(cy), 3153)
	if float64(h&0xffff)/65536 >= chance*0.55 || near < 18 {
		return color.RGBA{}, false
	}
	rx := 7 + 9*float64(h>>16&0xff)/255
	ry := rx * (0.5 + 0.2*float64(h>>24)/255)
	px := float64(cx*cw) + cw/2 + (hashf(cx, cy, 3154)-0.5)*(cw-2*rx)
	py := float64(cy*ch) + ch/2 + (hashf(cx, cy, 3155)-0.5)*(ch-2*ry)
	dx, dy := (float64(hx)+0.5-px)/rx, (float64(hy)+0.5-py)/ry
	d := dx*dx + dy*dy + (vnoise(float64(hx)/5, float64(hy)/4, 3156)-0.5)*0.5
	switch {
	case d > 1:
		return color.RGBA{}, false
	case d > 0.78:
		return cmPuddle.at(2), true // its wet rim
	case dy < -0.2 && dx > -0.6 && dx < 0.1 && d > 0.35:
		return cmPuddle.at(4), true // the light in it
	}
	return cmPuddle.pick(0.15+0.25*(1-d), hx, hy), true
}

// caveOpenNear reports whether chunk k or the ground round it has any of the
// caves open in it.
func (t *terrain) caveOpenNear(k chunkKey) bool {
	for ty := k.cy*chunkTiles - 3; ty < (k.cy+1)*chunkTiles+caveFaceH/hdTH+3; ty++ {
		for tx := k.cx*chunkTiles - 3; tx < (k.cx+1)*chunkTiles+3; tx++ {
			if ty >= t.cave.from && caveOpen(t.tile(tx, ty)) {
				return true
			}
		}
	}
	return false
}

// deepRockChunk is a chunk of nothing but rock, far from any cave: all but
// black, a faint grain in it.
func deepRockChunk(k chunkKey) []byte {
	c := newCanvas(chunkPW, chunkPH)
	x0, y0 := k.cx*chunkPW, k.cy*chunkPH
	for y := 0; y < chunkPH; y++ {
		for x := 0; x < chunkPW; x++ {
			hx, hy := x0+x, y0+y
			c.px[y*chunkPW+x] = caveDeepHD.g(0.12+0.1*vnoise(float64(hx)/90, float64(hy)/70, 3160), hx, hy)
		}
	}
	return c.bytes()
}
