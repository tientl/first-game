package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The leaf tent (sim/tent.go) at HD: an A-frame of branches thatched with
// big leaves, its open front to the viewer.
//
// It is drawn in the tent's own space and seen from the game's angle. Its
// foot is two tiles wide (tentB to each side of its tile's middle) and its
// front stands a little in front of that middle (tentYF); the ridge runs
// back from the front apex, tentH above the ground, to the back apex, so on
// the screen it goes straight up. Each slope is a plane (slopeXY): a down it
// from the ridge, b back along it from the front, in art pixels of the
// slope itself. The leaves are laid on it as a thatcher would, row by row
// from the foot up to the ridge, each row over the one below and the tips
// pointing down the slope, each leaf a flat shape of that plane seen from
// the game's angle, folded along its midrib and casting a short shadow on
// what it lies over. Every pixel keeps a material and an amount of light
// until the end, so shadows and folds add up before the tones are picked
// from the material's ramp.
//
// Four thatches, by the ground it stands on (tentLeafOf): fern fronds on
// grass, broad leaves in the woods and the marsh, palm fronds on sand and in
// the desert, and fir boughs under snow on the snow.
//
// It is two drawings: what is inside (the dark back wall, a bed of dry grass,
// a pillow of leaves) and the shell over it (the slopes, the frame of poles
// lashed at the top, the leaves on the ground). Someone asleep in it is drawn
// between the two (drawSleeper), so the poles and the leaves of the front
// come over their head and feet: they are inside.

const (
	tentB  = 24.0 // half the width of its foot, art pixels
	tentYF = 4.0  // its front, below the middle of its tile
	tentH  = 32.0 // the ridge's height at the front
	tentL  = 20.0 // how deep it is, front to back
	tentDY = tentL * 0.75
	slopeA = 40.0 // the length of a slope, ridge to foot: math.Hypot(tentB, tentH)

	tentW, tentHt  = 64, 82 // the canvas, art pixels: room below for a tent turned east or west (tentturn.go)
	tentAX, tentAY = 32, 59 // its anchor on the canvas: the tile's middle

	// Where a sleeper lies in it, from a bed's place for them: a little to
	// the left and down, so they lie across the middle of the floor.
	tentSleepDX, tentSleepDY = -2.0, 1.0
)

// tentLeaf is what a tent is thatched with.
type tentLeaf uint8

const (
	tentFern  tentLeaf = iota // fern fronds, on grass
	tentBroad                 // broad leaves, in the woods and the marsh
	tentPalm                  // palm fronds, on sand and in the desert
	tentFir                   // fir boughs under snow, on the snow
	numTentLeaves
)

// tentLeafOf is the thatch of a tent standing on ground t.
func tentLeafOf(t sim.Tile) tentLeaf {
	switch t {
	case sim.TileSand, sim.TileDesert:
		return tentPalm
	case sim.TileSnow, sim.TileIce:
		return tentFir
	case sim.TileForest, sim.TileMarsh:
		return tentBroad
	}
	return tentFern
}

// The materials of a tent: each pixel is one of these, with an amount of
// light.
const (
	tmNone   uint8 = iota
	tmLeaf         // the thatch
	tmDry          // a dead leaf of it
	tmBark         // the poles
	tmCord         // the lashing
	tmStraw        // the dry grass of the bed
	tmInside       // the dark inside
	tmSnow
	tmStone
	numTentMats
)

var (
	tentFernR  = mkRamp(0x112a1e, 0x1a4026, 0x25592e, 0x337338, 0x478d40, 0x62a84a, 0x86c25a, 0xb0da78)
	tentBroadR = mkRamp(0x0d2a20, 0x14402b, 0x1d5634, 0x2a6e3c, 0x3b8744, 0x56a04e, 0x78b95c, 0xa2d272)
	tentPalmR  = mkRamp(0x1c2e18, 0x2c4620, 0x3f5f27, 0x56782e, 0x709136, 0x8eaa42, 0xb0c250, 0xd4da70)
	tentFirR   = mkRamp(0x0b1c20, 0x10282a, 0x173a34, 0x1f4d3c, 0x2b6245, 0x3c7850, 0x55905c, 0x76aa70)
	tentDryR   = mkRamp(0x2a1e12, 0x45301a, 0x634422, 0x84602c, 0xa67c38, 0xc49a4c, 0xdcb868, 0xeed490)
	tentBarkR  = mkRamp(0x1e140f, 0x2f1f19, 0x46301f, 0x5e4128, 0x7a5533, 0x946a40, 0xae8456)
	tentCordR  = mkRamp(0x3a3018, 0x5e5026, 0x857236, 0xa89448, 0xc8b462, 0xe2d088)
	tentStrawR = mkRamp(0x2a2410, 0x433a18, 0x5e5222, 0x7c6e2e, 0x9c8c3c, 0xbcaa50, 0xd8c66a, 0xeee08e)
	tentInR    = mkRamp(0x0a0908, 0x110f0c, 0x191510, 0x221c14, 0x2d2518, 0x3a301e, 0x4a3e26)
	tentStoneR = mkRamp(0x2e2d38, 0x45444f, 0x5f5e68, 0x7b7a83, 0x9b9aa1, 0xc2c1c4)
	tentSnowR  = mkRamp(0x6f86a8, 0x8fa3c0, 0xaebfd6, 0xc9d6e6, 0xdfe8f2, 0xf2f6fa, 0xffffff)
)

// tentRamps is the ramp of each material for thatch v.
func tentRamps(v tentLeaf) [numTentMats]ramp {
	leaf := [numTentLeaves]ramp{tentFernR, tentBroadR, tentPalmR, tentFirR}[v]
	return [numTentMats]ramp{nil, leaf, tentDryR, tentBarkR, tentCordR, tentStrawR, tentInR, tentSnowR, tentStoneR}
}

// tentPaint is a tent being painted: a material and a light for every
// image pixel.
type tentPaint struct {
	w, h   int
	mat    []uint8
	lit    []float32
	shadow []float32 // soft shadow on the ground under it, where nothing is painted
	rs     [numTentMats]ramp
	seed   uint32
	geo    *tentGeo // turned from facing south (tentturn.go); nil as it was drawn
	gable  bool     // painting the closed end: only within its triangle
	clip   float64   // painting this slope of a turned tent: only from the ridge to the ground
	shown  []float64 // the slopes of a turned tent the viewer sees
}

func newTentPaint(v tentLeaf) *tentPaint {
	w, h := tentW*HDDensity, tentHt*HDDensity
	return &tentPaint{w: w, h: h, mat: make([]uint8, w*h), lit: make([]float32, w*h), shadow: make([]float32, w*h),
		rs: tentRamps(v), seed: 977 + uint32(v)*131}
}

const tk = float64(HDDensity)

// px is the image pixel of art point (x, y) from the anchor.
func (t *tentPaint) px(x, y float64) (float64, float64) {
	return (x + tentAX) * tk, (y + tentAY) * tk
}

// art is the art point, from the anchor, of the middle of image pixel (ix, iy).
func (t *tentPaint) art(ix, iy int) (float64, float64) {
	return (float64(ix)+0.5)/tk - tentAX, (float64(iy)+0.5)/tk - tentAY
}

func (t *tentPaint) in(ix, iy int) bool { return ix >= 0 && iy >= 0 && ix < t.w && iy < t.h }

func (t *tentPaint) set(ix, iy int, m uint8, l float64) {
	if t.gable && !t.onGable(ix, iy) || t.clip != 0 && !t.onSlope(ix, iy) {
		return
	}
	if t.in(ix, iy) {
		i := iy*t.w + ix
		t.mat[i], t.lit[i] = m, float32(l)
	}
}

// darken takes dl of the light off what is painted at (ix, iy).
func (t *tentPaint) darken(ix, iy int, dl float64) {
	if t.gable && !t.onGable(ix, iy) || t.clip != 0 && !t.onSlope(ix, iy) {
		return
	}
	if t.in(ix, iy) {
		if i := iy*t.w + ix; t.mat[i] != tmNone {
			t.lit[i] -= float32(dl)
		}
	}
}

// box is the image pixels round art points (x0, y0)-(x1, y1), grown by pad.
func (t *tentPaint) box(x0, y0, x1, y1, pad float64) (int, int, int, int) {
	ax, ay := t.px(min(x0, x1)-pad, min(y0, y1)-pad)
	bx, by := t.px(max(x0, x1)+pad, max(y0, y1)+pad)
	return max(0, int(ax)), max(0, int(ay)), min(t.w-1, int(bx)+1), min(t.h-1, int(by)+1)
}

// canvas turns the paint into colours, outlines it and lays the soft
// shadow under it.
func (t *tentPaint) canvas() *canvas {
	c := hdCanvas(tentW, tentHt)
	for i, m := range t.mat {
		if m != tmNone {
			x, y := i%t.w, i/t.w
			if tentDithered[m] {
				c.px[i] = t.rs[m].pick(float64(t.lit[i]), x, y)
			} else {
				c.px[i] = pick(t.rs[m], float64(t.lit[i]))
			}
		}
	}
	c.outline(0.78)
	for i, a := range t.shadow {
		if a > 0 && c.px[i].A == 0 {
			c.px[i] = color.RGBA{12, 16, 10, uint8(255 * clamp01(float64(a)))}
		}
	}
	return c
}

// --- the slopes -------------------------------------------------------------

// slopeXY is the art point a down slope side (-1 left, 1 right) from the
// ridge and b back from the front, art pixels of the slope.
func slopeXY(side, a, b float64) (x, y float64) {
	u := a / slopeA
	return side * tentB * u, tentYF - tentH*(1-u) - tentDY*b/tentL
}

// slopeAB is the inverse of slopeXY.
func slopeAB(side, x, y float64) (a, b float64) {
	a = x * side / tentB * slopeA
	u := a / slopeA
	b = (tentYF - tentH*(1-u) - y) * tentL / tentDY
	return a, b
}

// tLeaf is one leaf, or a leaflet of a frond, lying on a slope (side 0: on
// the ground, a and b then being x and y).
type tLeaf struct {
	side         float64
	a0, b0       float64 // its stalk end
	a1, b1       float64 // its tip
	w            float64 // half its width at the widest
	bend         float64 // the midrib bowed this far sideways at its middle
	narrow       bool    // a leaflet: widest near its base, tapering to a point
	mat          uint8
	lit          float64
	fold         float64 // the half facing up the screen this much lighter
	veins        float64 // side veins along it (0 for none)
	tears        int     // splits in the blade, as a banana leaf's
	shade        float64 // the shadow it casts on what it lies over
	flat         float64 // on the ground: squashed this much up and down
	needles      bool    // a fir bough: a comb of needles, not a blade
	seed         uint32
	aoTip, aoEnd float64 // light taken off at the stalk end, under the row above
}

// prof is how wide a leaf is s of the way from stalk to tip, 0 to 1.
func (l *tLeaf) prof(s float64) float64 {
	if s < 0 || s > 1 {
		return 0
	}
	if l.narrow {
		return math.Min(1, s*5) * math.Pow(1-s, 0.85)
	}
	return math.Pow(math.Sin(math.Pi*math.Min(1, s*1.06)), 0.62) * (1 - 0.12*s)
}

// at maps image pixel (ix, iy) to the leaf's own frame: s along it (0 at
// the stalk, 1 at the tip) and q across, in units of its half width there
// (|q| <= 1 inside).
func (l *tLeaf) at(ix, iy int, t *tentPaint, dx, dy float64) (s, q float64, ok bool) {
	x, y := t.art(ix, iy)
	x -= dx / tk
	y -= dy / tk
	var a, b float64
	if l.side == 0 {
		a, b = x, y/l.flat
	} else {
		a, b = t.sAB(l.side, x, y)
	}
	ex, ey := l.a1-l.a0, l.b1-l.b0
	L := math.Hypot(ex, ey)
	ux, uy := ex/L, ey/L
	pa, pb := a-l.a0, b-l.b0
	s = (pa*ux + pb*uy) / L
	across := -pa*uy + pb*ux - l.bend*4*s*(1-s)
	w := l.w * l.prof(s)
	if w <= 0 {
		return s, 0, false
	}
	q = across / w
	return s, q, math.Abs(q) <= 1
}

// leaf paints l over what is there: first its shadow, then the leaf.
func (t *tentPaint) leaf(l tLeaf) {
	if l.flat == 0 {
		l.flat = 0.6
	}
	pts := [][2]float64{{l.a0, l.b0}, {l.a1, l.b1}}
	var x0, y0, x1, y1 = 1e9, 1e9, -1e9, -1e9
	for _, p := range pts {
		var x, y float64
		if l.side == 0 {
			x, y = p[0], p[1]*l.flat
		} else {
			x, y = t.sXY(l.side, p[0], p[1])
		}
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	pad := l.w + math.Abs(l.bend) + 2
	bx0, by0, bx1, by1 := t.box(x0, y0, x1, y1, pad)
	// The shadow: the leaf's shape a little down and to the right, on what
	// it lies over.
	if l.shade > 0 {
		for iy := by0; iy <= by1; iy++ {
			for ix := bx0; ix <= bx1; ix++ {
				if _, _, in := l.at(ix, iy, t, 0, 0); in {
					continue
				}
				if _, _, sh := l.at(ix, iy, t, 1.5, 1.5); sh {
					t.darken(ix, iy, l.shade)
				} else if _, _, sh := l.at(ix, iy, t, 4, 4); sh {
					t.darken(ix, iy, l.shade*0.45)
				}
			}
		}
	}
	// Where the tears are: s along the blade, and which side.
	type tear struct{ s, side float64 }
	var tears []tear
	for i := 0; i < l.tears; i++ {
		tears = append(tears, tear{0.25 + 0.6*hashf(i, 1, l.seed), float64(1 - 2*(i%2))})
	}
	ex, ey := l.a1-l.a0, l.b1-l.b0
	L := math.Hypot(ex, ey)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			s, q, in := l.at(ix, iy, t, 0, 0)
			if !in {
				continue
			}
			torn := false
			for _, tr := range tears {
				// A split runs from the edge in towards the midrib, along the veins.
				if q*tr.side > 0.3 && math.Abs((s-tr.s)*L-(math.Abs(q)-1)*l.w*0.55) < 0.22 {
					torn = true
				}
			}
			if torn {
				continue
			}
			if l.needles && !needled(s, q, L, l.w) {
				continue
			}
			v := l.lit
			if q*t.foldSign(&l) > 0 {
				v += l.fold
			} else {
				v -= l.fold * 0.7
			}
			v -= 0.16 * q * q
			v -= l.aoEnd * math.Pow(1-s, 3)
			v += l.aoTip * s
			wPx := l.w * l.prof(s) * tk // its half width here, image pixels
			if math.Abs(q)*wPx < 0.75 && s > 0.03 && s < 0.94 {
				v += 0.16 // the midrib
			}
			if l.needles {
				// Each needle lit along its top, the gaps between them dark.
				if f := needlePhase(s, q, L, l.w); f < 0.2 {
					v += 0.1
				} else if f > 0.55 {
					v -= 0.16
				}
			}
			if l.veins > 0 {
				ph := s*l.veins - math.Abs(q)*0.55*l.veins*l.w/L
				if f := ph - math.Floor(ph); f < 0.13 && math.Abs(q) > 0.12 {
					v -= 0.14
				} else if f < 0.26 && math.Abs(q) > 0.12 {
					v += 0.04
				}
			}
			if (1-math.Abs(q))*wPx < 1 {
				v -= 0.16 // the rim, turned away
			}
			if l.side == tentGable {
				// The closed end: darker near the ground (tentturn.go).
				x, y := t.art(ix, iy)
				a, _ := t.sAB(l.side, x, y)
				v -= 0.12 * clamp01((a-22)/10)
			} else if l.side != 0 {
				// Further back and nearer the ground the slope is in its own shade.
				x, y := t.art(ix, iy)
				a, b := t.sAB(l.side, x, y)
				v -= 0.12*clamp01(b/tentL) + 0.14*clamp01((a-32)/10)
			}
			v += (hashf(ix, iy, l.seed) - 0.5) * 0.05
			t.set(ix, iy, l.mat, v)
		}
	}
}

// foldSign is which side of the midrib (the sign of q) faces up the
// screen: that half catches the light. q grows along (-uy, ux) of the
// leaf's own direction; on a slope b grows up the screen, on the ground
// down it.
func (l *tLeaf) foldSign() float64 {
	up := l.a1 >= l.a0
	if l.side == 0 {
		up = !up
	}
	if up {
		return 1
	}
	return -1
}

// stick paints a pole of bark from (x0, y0) to (x1, y1), art pixels, r0 to
// r1 thick, lit from the left.
func (t *tentPaint) stick(x0, y0, x1, y1, r0, r1, lit float64, seed uint32) {
	bx0, by0, bx1, by1 := t.box(x0, y0, x1, y1, max(r0, r1)+1)
	ex, ey := x1-x0, y1-y0
	L := math.Hypot(ex, ey)
	ux, uy := ex/L, ey/L
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			x, y := t.art(ix, iy)
			s := ((x-x0)*ux + (y-y0)*uy) / L
			if s < 0 || s > 1 {
				continue
			}
			r := lerpf(r0, r1, s)
			d := (x-x0)*-uy + (y-y0)*ux // across, art pixels, towards (-uy, ux)
			if math.Abs(d) > r {
				continue
			}
			// Lit on its side towards the light.
			if toward(-uy, ux) < 0 {
				d = -d
			}
			v := lit + 0.1 - (1-d/r)/2*0.55
			f := vnoise(d*1.3+float64(seed), s*L*0.7, seed)
			switch {
			case f < 0.33:
				v -= 0.16 // a furrow in the bark
			case f > 0.72:
				v += 0.08
			}
			if math.Abs(d) > r-0.4 {
				v -= 0.1
			}
			t.set(ix, iy, tmBark, v+(hashf(ix, iy, seed)-0.5)*0.06)
		}
	}
	// The cut end, pale wood.
	cx, cy := t.px(x1, y1)
	rr := r1 * tk * 0.9
	for iy := int(cy - rr - 1); iy <= int(cy+rr+1); iy++ {
		for ix := int(cx - rr - 1); ix <= int(cx+rr+1); ix++ {
			dx, dy := (float64(ix)+0.5-cx)/rr, (float64(iy)+0.5-cy)/rr
			if d := dx*dx + dy*dy; d <= 1 {
				t.set(ix, iy, tmStraw, 0.82-0.25*d)
			}
		}
	}
}

// lash winds cord n turns round the point (x, y), r art pixels across.
func (t *tentPaint) lash(x, y, r float64, seed uint32) {
	bx0, by0, bx1, by1 := t.box(x, y, x, y, r+1)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			ax, ay := t.art(ix, iy)
			dx, dy := (ax-x)/r, (ay-y)/(r*0.8)
			d := dx*dx + dy*dy
			if d > 1 {
				continue
			}
			v := lum(dx*0.8, dy*0.8)
			// The turns, slanting across.
			if math.Mod((ay-y)*tk*0.55+(ax-x)*tk*0.22+50, 2) < 0.7 {
				v -= 0.3
			}
			t.set(ix, iy, tmCord, v+(hashf(ix, iy, seed)-0.5)*0.05)
		}
	}
}

// --- the drawing ------------------------------------------------------------

// shell paints the slopes, the frame and the leaves about its foot.
func (t *tentPaint) shell(v tentLeaf) {
	r := func(i, j int) float64 { return hashf(i, j, t.seed) }
	// The ridge pole along the top, behind the thatch: its end shows past
	// the back.
	_, by := slopeXY(1, 0, tentL)
	_, fy := slopeXY(1, 0, 0)
	t.stick(0.2, fy+1.5, -0.2, by-2.4, 1.0, 0.9, 0.62, t.seed+3)

	// The slopes: the shaded one first, so the lit one's top row lies over
	// the ridge. Under the leaves one sees, a thick bed of older ones, so
	// no daylight shows through.
	for _, side := range []float64{1, -1} {
		lit := 0.7
		if side > 0 {
			lit = 0.34
		}
		t.thatchBed(side, lit-0.3)
		switch v {
		case tentBroad:
			t.broadSlope(side, lit)
		case tentPalm:
			t.palmSlope(side, lit)
		case tentFir:
			t.firSlope(side, lit)
		default:
			t.fernSlope(side, lit)
		}
	}
	if v == tentFir {
		t.ridgeSnow()
	}
	t.cutEdge()

	// The front of the frame: two poles from the foot crossed at the top
	// and lashed, the leaves resting on them.
	ftop := tentYF - tentH
	ext := 0.2
	for _, side := range []float64{-1, 1} {
		x0, y0 := side*(tentB+0.6), tentYF+1.2
		x1, y1 := lerpf(x0, 0, 1+ext)+side*0.4, lerpf(y0, ftop, 1+ext)
		lit := 0.7
		if side > 0 {
			lit = 0.5
		}
		t.stick(x0, y0, x1, y1, 1.25, 0.9, lit, t.seed+uint32(10+side))
	}
	t.lash(0, ftop+0.4, 1.9, t.seed+20)

	// A few leaves of the thatch over the poles at the front.
	t.fringe(v)

	// Stones holding the feet of the poles, and the leaves fallen about.
	for _, side := range []float64{-1, 1} {
		t.stone(side*(tentB+1.6), tentYF+1.6, 2.1, 1.4, t.seed+uint32(30+side))
		t.stone(side*(tentB-1.2), tentYF+2.6, 1.4, 1.0, t.seed+uint32(33+side))
	}
	t.groundLeaves(v, r)
}

// cutEdge shows the thickness of the thatch along the open front: the
// undersides of its leaves, dark, ragged where they hang.
func (t *tentPaint) cutEdge() {
	apexY := tentYF - tentH
	for iy := 0; iy < t.h; iy++ {
		for ix := 0; ix < t.w; ix++ {
			x, y := t.art(ix, iy)
			if y < apexY+1 || y > tentYF {
				continue
			}
			hw := tentB * (y - apexY) / tentH
			in := hw - math.Abs(x) // how far inside the front it is
			th := 1.3 + 0.9*vnoise(y*0.8, x*0.1, t.seed+88)
			if in < 0 || in > th {
				continue
			}
			if t.mat[iy*t.w+ix] != tmNone && in < 0.4 {
				continue
			}
			t.set(ix, iy, tmLeaf, 0.1+0.1*(1-in/th))
		}
	}
}

// thatchBed fills slope side with the dark mass of the thatch under the
// leaves laid on it.
func (t *tentPaint) thatchBed(side, lit float64) {
	x0, y0 := t.sXY(side, 0, tentL)
	x1, y1 := t.sXY(side, slopeA, 0)
	bx0, by0, bx1, by1 := t.box(x0, y0, x1, y1, 1)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			x, y := t.art(ix, iy)
			a, b := t.sAB(side, x, y)
			if a < 0 || a > slopeA+0.5 || b < 0 || b > tentL+0.3 {
				continue
			}
			v := lit + (fbm2(a*0.3, b*0.5, t.seed+77, 2)-0.5)*0.2 - 0.1*b/tentL
			t.set(ix, iy, tmLeaf, v)
		}
	}
}

// row lays leaves along a slope at a (their tips), from the back to the
// front; each is made by mk from where its tip goes and its index.
func row(bFrom, bTo, step float64, mk func(b float64, i int)) {
	i := 0
	for b := bFrom; b >= bTo; b -= step {
		mk(b, i)
		i++
	}
}

func (t *tentPaint) broadSlope(side, lit float64) {
	tips := []float64{45, 38, 31, 24, 17, 10.5, 4.5}
	for ri, at := range tips {
		off := 0.0
		if ri%2 == 1 {
			off = 3
		}
		row(tentL+2.5-off, -2, 7.2, func(b float64, i int) {
			h := func(j int) float64 { return hashf(ri*17+i, j, t.seed+uint32(side+5)) }
			b += (h(0) - 0.5) * 2
			ln := 12 + h(1)*2.5
			if ri == len(tips)-1 {
				ln = 11 // the top row: over the ridge
				b = min(b, tentL-2.5)
			}
			mat := tmLeaf
			if h(2) < 0.14 && ri < 2 {
				mat = tmDry
			}
			tip := at + (h(3)-0.5)*2
			t.leaf(tLeaf{side: side, a0: tip - ln, b0: b + (h(4)-0.5)*2.5, a1: tip, b1: b + (h(5)-0.5)*4,
				w: 8 + h(6)*1.6, bend: (h(7) - 0.5) * 3, mat: mat, lit: lit + (h(8)-0.5)*0.12, fold: 0.14,
				veins: 7, tears: int(h(9) * 1.6), shade: 0.15, aoEnd: 0.2, aoTip: 0.05, seed: t.seed + uint32(ri*31+i)})
		})
	}
}

// frond lays a frond on a slope: its stalk from a0 to a1 at b (bowed by
// bend), leaflets every gap both sides, ln(s) long, swept down by sweep.
func (t *tentPaint) frond(side, a0, a1, b, bend float64, gap, sweep float64, ln func(s float64) float64, mat uint8, lit float64, w float64, seed uint32) {
	L := a1 - a0
	n := int(L / gap)
	for i := n; i >= 1; i-- { // from the tip back, so the leaflets nearer the stalk's end lie over
		s := float64(i) / float64(n)
		a := a0 + L*s
		bb := b + bend*4*s*(1-s)
		l := ln(s)
		for k, dir := range []float64{1, -1} {
			h := hashf(i, k, seed)
			sw := sweep + (h-0.5)*0.25
			t.leaf(tLeaf{side: side, a0: a, b0: bb, a1: a + l*sw, b1: bb + dir*l*(1-sw*0.35),
				w: w, narrow: true, bend: dir * 0.35 * l * 0.15, mat: mat, lit: lit + (h-0.5)*0.1 + dir*0.04,
				fold: 0.05, shade: 0.12, seed: seed + uint32(i*7+k)})
		}
	}
	// The stalk over its leaflets.
	for i := 0; i <= int(L*tk); i++ {
		s := float64(i) / (L * tk)
		a := a0 + L*s
		bb := b + bend*4*s*(1-s)
		x, y := t.sXY(side, a, bb)
		ix, iy := t.px(x, y)
		th := 1 + int(0.8*(1-s))
		for d := 0; d < th; d++ {
			t.set(int(ix), int(iy)+d, mat, lit+0.12-float64(d)*0.12)
		}
	}
}

func (t *tentPaint) palmSlope(side, lit float64) {
	bs := []float64{20.5, 16, 11.5, 7, 2.5, -1.2}
	for i, b := range bs {
		h := func(j int) float64 { return hashf(i, j, t.seed+uint32(side+7)) }
		mat := tmLeaf
		if h(0) < 0.14 {
			mat = tmDry
		}
		t.frond(side, -3.5+h(1)*2, 45+h(2)*3, b+(h(3)-0.5)*1.5, (h(4)-0.5)*3, 1.55, 0.7,
			func(s float64) float64 { return (3.2 + 4.2*math.Sin(math.Pi*math.Min(1, s*1.15+0.1))) * (1 - 0.25*s) },
			mat, lit+(h(5)-0.5)*0.12, 0.95, t.seed+uint32(i*53)+uint32(side+3))
	}
}

func (t *tentPaint) fernSlope(side, lit float64) {
	rows := [][2]float64{{20, 45}, {-3.5, 24}}
	for ri, span := range rows {
		off := 0.0
		if ri == 1 {
			off = 3
		}
		row(tentL+1-off, -1.5, 5.6, func(b float64, i int) {
			h := func(j int) float64 { return hashf(ri*13+i, j, t.seed+uint32(side+9)) }
			mat := tmLeaf
			if h(0) < 0.07 {
				mat = tmDry
			}
			t.frond(side, span[0]+(h(1)-0.5)*3, span[1]+(h(2)-0.5)*3, b+(h(3)-0.5)*1.5, (h(4)-0.5)*3.5, 1.25, 0.55,
				func(s float64) float64 { return 1.2 + 3.6*math.Pow(math.Sin(math.Pi*math.Min(1, s*0.95+0.05)), 0.8) },
				mat, lit+(h(5)-0.5)*0.12+0.04, 0.8, t.seed+uint32(ri*97+i*11)+uint32(side+5))
		})
	}
}

func (t *tentPaint) firSlope(side, lit float64) {
	bs := []float64{21, 16.5, 12, 7.5, 3, -1.2}
	for i, b := range bs {
		h := func(j int) float64 { return hashf(i, j, t.seed+uint32(side+11)) }
		t.bough(side, -3+h(1)*2, 45+h(2)*2, b+(h(3)-0.5)*1.5, (h(4)-0.5)*2.5, lit+(h(5)-0.5)*0.1, t.seed+uint32(i*41)+uint32(side+2))
	}
	// Snow on the boughs: more on the slope the sun is on.
	amount := 1.0
	if side > 0 {
		amount = 0.7
	}
	for i, b := range bs {
		t.snowCaps(side, b, amount, t.seed+uint32(i*17)+uint32(side+4))
	}
}

// bough lays a fir bough on a slope: side shoots and a main twig, each a
// comb of short needles swept towards its tip.
func (t *tentPaint) bough(side, a0, a1, b, bend, lit float64, seed uint32) {
	L := a1 - a0
	for i := 1; i <= 6; i++ {
		s := float64(i) / 7
		a := a0 + L*s
		bb := b + bend*4*s*(1-s)
		for k, dir := range []float64{1, -1} {
			if hashf(i, k+5, seed) < 0.2 {
				continue
			}
			l := 6.2 * (1 - 0.4*s)
			t.leaf(tLeaf{side: side, a0: a, b0: bb, a1: a + l*0.8, b1: bb + dir*l*0.6, w: 1.9, narrow: true, needles: true,
				mat: tmLeaf, lit: lit - 0.05 + dir*0.05, shade: 0.16, seed: seed + uint32(i*13+k)})
		}
	}
	t.leaf(tLeaf{side: side, a0: a0, b0: b, a1: a1, b1: b + bend*0.2, w: 2.5, needles: true, bend: bend * 0.4,
		mat: tmLeaf, lit: lit, shade: 0.2, seed: seed + 99})
}

// snowCaps lays snow along the top of the bough at b: soft lumps on its
// upper side, lit from the left, a blue shade under them.
func (t *tentPaint) snowCaps(side, b, amount float64, seed uint32) {
	type lump struct{ a, b, ra, rb float64 }
	var lumps []lump
	for i, a := 0, 1.0+3*hashf(0, 0, seed); a < 41; i++ {
		h := hashf(i, 1, seed)
		a += 2.5 + 4.5*hashf(i, 3, seed)
		if h > 0.35+0.5*amount {
			continue // bare here: it slid off, or never settled
		}
		lumps = append(lumps, lump{a, b + 0.6 + hashf(i, 2, seed)*1.4, (2.4 + 2*hashf(i, 4, seed)) * amount, (1.8 + 1.2*hashf(i, 5, seed)) * amount})
		if t.geo != nil { // a turned slope is seen more nearly face on: the snow lies along the boughs
			l := &lumps[len(lumps)-1]
			l.ra, l.rb = l.ra*1.6, l.rb*0.7
		}
	}
	x0, y0 := t.sXY(side, 0, b+4)
	x1, y1 := t.sXY(side, slopeA+2, b-3)
	bx0, by0, bx1, by1 := t.box(x0, y0, x1, y1, 2)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			if t.mat[iy*t.w+ix] == tmNone {
				continue
			}
			x, y := t.art(ix, iy)
			a, bb := t.sAB(side, x, y)
			best, nx, ny := 2.0, 0.0, 0.0
			for _, l := range lumps {
				dx, dy := (a-l.a)/l.ra, (bb-l.b)/l.rb
				if d := dx*dx + dy*dy; d < best {
					best, nx, ny = d, dx, dy
				}
			}
			// Soft lumps run into each other and break up at their edges, as
			// snow does on needles.
			best += (fbm2(a*0.9, bb*1.3, seed+7, 2) - 0.5) * 0.7
			switch {
			case best <= 1:
				// Seen on the screen, b grows upwards: a lump's top is its +b side.
				v := lum(nx*side*0.7, -ny*0.8) + 0.05
				if side > 0 {
					v -= 0.12
				}
				t.set(ix, iy, tmSnow, v)
			case best <= 1.7 && ny < 0:
				t.darken(ix, iy, 0.14) // its shade on the needles below
			}
		}
	}
}

// ridgeSnow lays a cap of snow along the ridge, where it settles deepest.
func (t *tentPaint) ridgeSnow() {
	_, top := slopeXY(1, 0, tentL)
	_, bot := slopeXY(1, 0, 0)
	top, bot = top+0.6, bot-1.5
	bx0, by0, bx1, by1 := t.box(-4, top-1, 4, bot+1, 1)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			x, y := t.art(ix, iy)
			if y < top || y > bot {
				continue
			}
			// A ridge of snow over the top, wider where it drifted, ragged
			// where it runs down between the boughs.
			hw := 1.5 + 0.9*vnoise(y*0.35, 3, t.seed+91) + 0.8*math.Max(0, vnoise(y*0.9, x*0.2, t.seed+92)-0.4)
			if f := (y - top) / 1.5; f < 1 {
				hw *= math.Sqrt(math.Max(0, f)) // rounded at the back
			}
			nx := x / hw
			if math.Abs(nx) > 1 {
				continue
			}
			v := lum(nx*0.75, -0.35) + 0.06
			if math.Abs(nx) > 0.82 {
				v -= 0.08
			}
			t.set(ix, iy, tmSnow, v)
		}
	}
}

// fringe lays a few leaves over the front poles, so the thatch rests on the
// frame rather than behind it.
func (t *tentPaint) fringe(v tentLeaf) {
	for _, side := range []float64{-1, 1} {
		lit := 0.68
		if side > 0 {
			lit = 0.42
		}
		if t.geo != nil { // turned: only over the slopes in sight, lit as they are (tentturn.go)
			if !t.geo.facesUs(side) {
				continue
			}
			lit = 0.17 + 0.72*t.geo.planeLit(side)
		}
		for i, at := range []float64{9, 21, 33} {
			h := func(j int) float64 { return hashf(i, j, t.seed+uint32(side+40)) }
			switch v {
			case tentBroad:
				t.leaf(tLeaf{side: side, a0: at - 9, b0: 2.5, a1: at + 2, b1: -2.2 - h(0)*1.2, w: 3.6, bend: 1,
					mat: tmLeaf, lit: lit + 0.05, fold: 0.1, veins: 6, shade: 0.18, seed: t.seed + uint32(70+i)})
			case tentFir:
				t.bough(side, at-8, at+4, -0.8, 0.5, lit, t.seed+uint32(80+i)+uint32(side+1))
			default:
				w := 0.8
				if v == tentPalm {
					w = 0.95
				}
				for k := 0; k < 3; k++ {
					a := at - 3 + float64(k)*2.2
					t.leaf(tLeaf{side: side, a0: a, b0: 1.5, a1: a + 3.2, b1: -3.4 - h(k)*1.3, w: w, narrow: true,
						mat: tmLeaf, lit: lit + (h(k+3)-0.5)*0.1, fold: 0.05, shade: 0.14, seed: t.seed + uint32(90+i*3+k)})
				}
			}
		}
	}
}

// stone is a small stone on the ground at (x, y), rx by ry art pixels.
func (t *tentPaint) stone(x, y, rx, ry float64, seed uint32) {
	bx0, by0, bx1, by1 := t.box(x, y, x, y, max(rx, ry)+1)
	for iy := by0; iy <= by1; iy++ {
		for ix := bx0; ix <= bx1; ix++ {
			ax, ay := t.art(ix, iy)
			nx, ny := (ax-x)/rx, (ay-y)/ry
			d := nx*nx + ny*ny
			if d > 1 {
				if d < 1.9 && ny > 0.2 {
					t.shadow[iy*t.w+ix] = max(t.shadow[iy*t.w+ix], 0.3)
				}
				continue
			}
			fx, fy := math.Round(nx*1.6)/1.6, math.Round(ny*1.6)/1.6
			v := lum(0.5*nx+0.5*fx, 0.5*ny+0.5*fy)*0.95 + (hashf(ix, iy, seed)-0.5)*0.08
			if ny > 0.5 {
				v -= 0.15
			}
			t.set(ix, iy, tmStone, v)
		}
	}
}

// groundLeaves strews a few leaves on the ground about the tent's foot.
func (t *tentPaint) groundLeaves(v tentLeaf, r func(i, j int) float64) {
	spots := [][3]float64{{-27, 2.5, 2.6}, {26.5, 1.2, 0.4}, {-12, 7.2, -0.3}, {13.5, 6.4, 2.9}, {3, 8.4, 1.6}, {-19, 6.8, 2.2}}
	for i, s := range spots {
		ang := s[2] + (r(i, 50)-0.5)*0.6
		mat := tmLeaf
		if r(i, 51) < 0.45 {
			mat = tmDry
		}
		ln := 6.0
		w := 2.4
		narrow := false
		switch v {
		case tentBroad:
			ln, w = 6, 2.3
		case tentPalm:
			ln, w, narrow = 7.5, 0.9, true
		case tentFir:
			ln, w, narrow = 4, 0.45, true
		case tentFern:
			ln, w, narrow = 5.5, 1.6, true
		}
		x0, y0 := s[0]-math.Cos(ang)*ln/2, s[1]/0.6-math.Sin(ang)*ln/2
		x1, y1 := s[0]+math.Cos(ang)*ln/2, s[1]/0.6+math.Sin(ang)*ln/2
		t.groundLeaf(tLeaf{side: 0, flat: 0.6, a0: x0, b0: y0, a1: x1, b1: y1, w: w, narrow: narrow, bend: 0.5,
			mat: mat, lit: 0.55 + (r(i, 52)-0.5)*0.15, fold: 0.08, veins: 5, seed: t.seed + uint32(200+i)})
		if v == tentFir || v == tentPalm || v == tentFern {
			// A sprig, not one leaflet: two more beside it.
			for k := 1; k <= 2; k++ {
				d := float64(k) * 1.3
				t.groundLeaf(tLeaf{side: 0, flat: 0.6, a0: x0 + d*math.Cos(ang), b0: y0 + d*math.Sin(ang), a1: x1 - math.Sin(ang)*float64(3-2*k)*1.6,
					b1: y1 + math.Cos(ang)*float64(3-2*k)*1.6, w: w * 0.9, narrow: true, mat: mat, lit: 0.5, fold: 0.05, seed: t.seed + uint32(220+i*3+k)})
			}
		}
	}
}

// inside paints what is seen through the open front: the back wall from
// within, the floor with its bed of dry grass, and a pillow of leaves.
func (t *tentPaint) inside() {
	backY := tentYF - tentDY // where the back wall meets the floor
	apexY := tentYF - tentH
	for iy := 0; iy < t.h; iy++ {
		for ix := 0; ix < t.w; ix++ {
			x, y := t.art(ix, iy)
			if y > tentYF+0.4 || y < apexY-0.5 {
				continue
			}
			hw := tentB*(y-apexY)/tentH + 1.4 // the front's half width here, and a little under the thatch
			if math.Abs(x) > hw {
				continue
			}
			edge := clamp01((hw - math.Abs(x)) / 7) // light falls off under the slopes
			if y < backY {
				// The back wall: the undersides of the leaves, darker up in
				// the apex, a chink of daylight here and there.
				up := (backY - y) / (backY - apexY)
				l := 0.3 - 0.22*up
				n := fbm2(x*0.45, y*0.7, t.seed+61, 3)
				l += (n - 0.5) * 0.14
				if math.Mod(x*0.35+y*0.6+n*3+20, 2.2) < 0.18 {
					l -= 0.08 // the edge of a leaf
				}
				l *= 0.55 + 0.45*edge
				// A crossbar lashed across the back poles.
				if cy := backY - 9.5 + x*0.04; math.Abs(y-cy) < 0.55 && math.Abs(x) < (tentB*(cy-apexY)/tentH) {
					t.set(ix, iy, tmBark, 0.26-(y-cy)*0.2)
					continue
				}
				t.set(ix, iy, tmInside, l)
				continue
			}
			// The floor: dry grass, lit from the front, dark at the back.
			depth := (tentYF - y) / tentDY // 0 at the front, 1 at the back
			l := 0.44 - 0.38*depth
			l *= 0.35 + 0.65*edge*edge
			// Stalks: short streaks lying every way, by patch.
			slant := 1.0
			if vnoise(x*0.3, y*0.4, t.seed+63) > 0.5 {
				slant = -1
			}
			sx, sy := float64(ix)+slant*float64(iy)*1.8, float64(iy)-slant*float64(ix)*0.25
			switch h := hashf(int(sx/1), int(sy/7), t.seed+64); {
			case h < 0.22:
				l += 0.14
			case h > 0.8:
				l -= 0.14
			}
			if y > backY && y < backY+1.2 {
				l -= 0.12 // where it meets the back wall
			}
			t.set(ix, iy, tmStraw, l)
		}
	}
	// The pillow: a bundle of leaves where the head goes, and a few leaves
	// lying on the bed.
	px, py := pillowX+tentSleepDX, pillowY+tentSleepDY+1.5
	for i := 0; i < 5; i++ {
		a := -0.4 + float64(i)*0.22
		t.leaf(tLeaf{side: 0, flat: 1, a0: px - 4 + float64(i)*0.6, b0: py + 1 - float64(i)*0.3, a1: px + 4*math.Cos(a) - 2 + float64(i)*0.8,
			b1: py - 2*math.Sin(a), w: 2.2, mat: tmLeaf, lit: 0.42 - float64(i)*0.03, fold: 0.06, veins: 4, shade: 0.12,
			seed: t.seed + uint32(300+i)})
	}
	for i, l := range [][4]float64{{9, -2.5, 14, -4}, {-2, -0.5, 3, 0.5}, {14, 1.5, 18, 0.2}} {
		mat := tmLeaf
		if i == 1 {
			mat = tmDry
		}
		t.leaf(tLeaf{side: 0, flat: 1, a0: l[0], b0: l[1], a1: l[2], b1: l[3], w: 1.6, mat: mat, lit: 0.45, fold: 0.05, veins: 4,
			shade: 0.1, seed: t.seed + uint32(320+i)})
	}
	// Dry grass spilling out over the front edge.
	for i := 0; i < 26; i++ {
		x := -18 + 36*hashf(i, 1, t.seed+70)
		hw := tentB * (tentYF - (tentYF - tentH)) / tentH
		if math.Abs(x) > hw-3 {
			continue
		}
		ln := 1.5 + 2*hashf(i, 2, t.seed+70)
		ang := (hashf(i, 3, t.seed+70) - 0.5) * 1.6
		for k := 0.0; k <= ln*tk; k++ {
			ax, ay := x+math.Sin(ang)*k/tk, tentYF-0.6+math.Cos(ang)*k/tk*0.6
			ix, iy := t.px(ax, ay)
			t.set(int(ix), int(iy), tmStraw, 0.72-k/(ln*tk)*0.2)
		}
	}
	// The soft shadow on the ground round the foot, falling right.
	for iy := 0; iy < t.h; iy++ {
		for ix := 0; ix < t.w; ix++ {
			x, y := t.art(ix, iy)
			x -= 1.2
			dx := math.Max(0, math.Abs(x)-tentB)
			dy := math.Max(0, math.Max(y-tentYF, tentYF-tentDY-y))
			d := math.Hypot(dx, dy)
			if d < 3.5 {
				t.shadow[iy*t.w+ix] = max(t.shadow[iy*t.w+ix], float32(0.3*(1-smooth(d/3.5))*(1-0.5*clamp01((tentYF-y)/tentDY))))
			}
		}
	}
}

// genTent paints a tent thatched with v: what is inside, and the shell.
func genTent(v tentLeaf) (back, front *canvas) { return genTentFaced(v, sim.DirDown) }

// genTentFaced paints a tent thatched with v, its door facing face: what is
// seen inside through the door, and the shell (tentturn.go for the turned).
func genTentFaced(v tentLeaf, face sim.Dir) (back, front *canvas) {
	in := newTentPaint(v)
	sh := newTentPaint(v)
	if geo := tentGeoOf(face); geo != nil {
		in.geo, sh.geo = geo, geo
		in.insideTurned()
		sh.shellTurned(v)
		return in.canvas(), sh.canvas()
	}
	in.inside()
	sh.shell(v)
	return in.canvas(), sh.canvas()
}

// tentArt is a thatch's drawings: what is inside, the shell, and both as one.
type tentArt struct{ back, front, whole sprite }

var tentArts [numTentLeaves][4]*tentArt

// tentArtOf is thatch v's drawings, its door to the south, made the first
// time they are wanted.
func tentArtOf(v tentLeaf) *tentArt { return tentArtFaced(v, sim.DirDown) }

// tentArtFaced is thatch v's drawings with its door facing face.
func tentArtFaced(v tentLeaf, face sim.Dir) *tentArt {
	face &= 3
	if a := tentArts[v][face]; a != nil {
		return a
	}
	back, front := genTentFaced(v, face)
	whole := hdCanvas(tentW, tentHt)
	whole.over(back, 0, 0)
	whole.over(front, 0, 0)
	for i, p := range back.px { // the soft shadow under it all
		if whole.px[i].A == 0 && p.A > 0 {
			whole.px[i] = p
		}
	}
	a := &tentArt{
		back:  mkUp(back, tentAX*tk, tentAY*tk),
		front: mkUp(front, tentAX*tk, tentAY*tk),
		whole: mkUp(whole, tentAX*tk, tentAY*tk),
	}
	tentArts[v][face] = a
	return a
}

// tentArtAt is the drawings of tent e, by its ground and its facing.
func (g *Game) tentArtAt(e protocol.EntityView) *tentArt {
	return tentArtFaced(tentLeafOf(g.tileAt(int(e.TX), int(e.TY))), sim.FacingOf(e.Kind, e.Var, e.Face))
}

// tentSprite is how a tent looks, whole: for the bag, a ghost, the label.
func (g *Game) tentSprite(e protocol.EntityView) sprite {
	return g.tentArtAt(e).whole
}

// tentSleeper reports whether someone is asleep in the tent on tile (tx, ty).
func (g *Game) tentSleeper(tx, ty int16) bool {
	for _, p := range g.curr.Players {
		if p.Asleep && !p.Dead && int16(math.Floor(float64(p.X))) == tx && int16(math.Floor(float64(p.Y))) == ty {
			return true
		}
	}
	return false
}

// drawTentEntity draws a tent standing at d: inside and shell, or only the
// inside while someone sleeps in it (drawSleeper puts the shell over them).
// It reports whether e was a tent.
func (g *Game) drawTentEntity(dst drawTarget, e protocol.EntityView, d drawable) bool {
	if e.Kind != sim.EntityLeafTent {
		return false
	}
	a := g.tentArtAt(e)
	solid := g.seeThrough(e, a.whole, d) // fading while it hides someone behind; see occlude.go
	noteCanopy(a.whole, d.sx, d.sy)
	put := func(s sprite) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(d.sx-s.ax, d.sy-s.ay)
		op.ColorScale.ScaleAlpha(solid)
		drawSprite(dst, s, op)
	}
	put(a.back)
	if !g.tentSleeper(e.TX, e.TY) || tentShut(e) {
		put(a.front)
	}
	return true
}

// tentShut reports whether tent e is turned so that whoever sleeps in it
// is out of sight: its door to the back or the sides (tentturn.go). Only
// the sleep floating up from them shows.
func tentShut(e protocol.EntityView) bool {
	return sim.FacingOf(e.Kind, e.Var, e.Face) != sim.DirDown
}

// bedLabel is what the target marker says over a bed or a tent named
// label: it is slept in by night; by day an axe in the hand takes it down.
func (g *Game) bedLabel(label string) string {
	if sim.PhaseOf(g.curr.Tick) != sim.PhaseDay {
		return label + " — ngủ"
	}
	if c, _ := g.myHeld().ToolKind(); c == sim.ToolAxe {
		return label + " — dỡ"
	}
	return label + " — ngủ khi trời tối · cầm rìu để dỡ"
}

// tentUnder is the tent p sleeps in, if they sleep in one.
func (g *Game) tentUnder(p protocol.PlayerView) (protocol.EntityView, bool) {
	if !p.Asleep || p.Dead {
		return protocol.EntityView{}, false
	}
	i, ok := g.entAt[[2]int{int(math.Floor(float64(p.X))), int(math.Floor(float64(p.Y)))}]
	if !ok || i >= len(g.curr.Entities) || g.curr.Entities[i].Kind != sim.EntityLeafTent {
		return protocol.EntityView{}, false
	}
	return g.curr.Entities[i], true
}

// drawTentShell puts the shell of tent e, standing at (x, y), over whoever
// sleeps in it.
func (g *Game) drawTentShell(dst drawTarget, e protocol.EntityView, x, y float64) {
	a := g.tentArtAt(e)
	putSprite(dst, a.front, x, y)
}

// tentDithered says which materials are shaded with dither between their
// tones: the grass of the bed and the dark inside, broken up; the leaves,
// the bark and the snow keep clean bands, so every leaf stands out.
var tentDithered = [numTentMats]bool{tmStraw: true, tmInside: true, tmStone: true}

// needlePhase is where a point s along a bough L long, q across it (of w),
// falls among its needles: they leave the twig every 0.75 art pixels and
// sweep forward to its tip, so the fraction is a needle below 0.55.
func needlePhase(s, q, L, w float64) float64 {
	p := (s*L - 0.9*math.Abs(q)*w) / 0.75
	return p - math.Floor(p)
}

// needled reports whether a point of a bough is on a needle: at its edge
// the gaps between them are open, a comb.
func needled(s, q, L, w float64) bool {
	return math.Abs(q) < 0.62 || needlePhase(s, q, L, w) < 0.55
}
