package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"survivalgame/internal/sim"
)

// The wild plants of sim/flora.go and the crops, drawn at HD: image pixels,
// HDDensity to an art pixel, like the rest of the small plants
// (hdplants.go). Stems, fruit and anything round are signed-distance shapes
// lit as one bulge (the critters' tools, critterhd.go); leaves are drawn one
// by one, with a midrib and veins. Every plant has two looks: full, and
// picked or cut.
//
// floraart.go holds the tools and the plants of the land; florawet.go the
// plants of the water and its banks; floracrop.go what grows in a bed.

// flo is a canvas a plant is drawn on, with the brushes of both toolkits.
type flo struct {
	c *canvas
	q *crit
}

func newFlo(w, h int) *flo {
	c := hdCanvas(w, h)
	return &flo{c: c, q: &crit{c: c, edge: 0.3, zoom: 1}}
}

const fk = hdK3 // image pixels to an art pixel

// leafForm is the outline of a leaf: its half width, as a share of the
// widest, at t from the stalk (0) to the tip (1).
type leafForm func(t float64) float64

var (
	leafLance = leafForm(func(t float64) float64 {
		return math.Pow(math.Sin(math.Pi*math.Min(1, t*1.12)), 0.7) * (1 - 0.15*t)
	})
	leafHeart = leafForm(func(t float64) float64 {
		return math.Pow(math.Sin(math.Pi*math.Pow(t, 0.62)), 0.55)
	})
	leafStrap = leafForm(func(t float64) float64 {
		return math.Min(1, t*7) * math.Pow(1-t, 0.42)
	})
	leafSaw = leafForm(func(t float64) float64 {
		f := t*6.5 - math.Floor(t*6.5)
		return math.Pow(math.Sin(math.Pi*math.Min(1, t*1.1)), 0.75) * (1 - 0.24*f)
	})
	leafRound = leafForm(func(t float64) float64 {
		return math.Sqrt(math.Max(0, 1-(2*t-1)*(2*t-1)))
	})
)

// leafAt draws one leaf from (x0, y0), in art pixels, L long and W half
// wide, pointing along ang (0 is right, a quarter turn up), sagging by sag
// of its length at the tip. veins is how many side veins it shows (0: only
// the midrib); lit moves it lighter or darker.
func (f *flo) leafAt(x0, y0, ang, L, W, sag float64, form leafForm, rp ramp, veins, lit float64) {
	x0, y0, L, W = x0*fk, y0*fk, L*fk, W*fk
	ux, uy := math.Cos(ang), -math.Sin(ang)
	vx, vy := -uy, ux
	if vy < 0 { // v points down the screen, so sag bends the leaf down
		vx, vy = -vx, -vy
	}
	const steps = 22
	var pts []float64
	for s := 0; s <= steps; s++ {
		t := float64(s) / steps
		w, b := W*form(t), sag*L*t*t
		pts = append(pts, x0+ux*L*t+vx*(b+w), y0+uy*L*t+vy*(b+w))
	}
	for s := steps; s >= 0; s-- {
		t := float64(s) / steps
		w, b := W*form(t), sag*L*t*t
		pts = append(pts, x0+ux*L*t+vx*(b-w), y0+uy*L*t+vy*(b-w))
	}
	n := len(rp)
	seed := uint32(int(x0)*31 + int(y0)*17)
	f.c.polygon(pts, func(x, y int) color.RGBA {
		px, py := float64(x)+0.5-x0, float64(y)+0.5-y0
		along := clamp01((px*ux + py*uy) / L)
		off := px*vx + py*vy - sag*L*along*along
		across := off / math.Max(W*form(along), 0.7)
		// The half that faces up the screen catches the light; the leaf
		// darkens to its stalk and its rim.
		l := 0.52 + lit - 0.2*across + 0.16*along - 0.1*across*across
		switch {
		case math.Abs(off) < 0.6 && along < 0.9:
			l += 0.2 // the midrib
		case veins > 0:
			if v := math.Mod(along*veins-math.Abs(across)*1.3+8, 1); v < 0.16 {
				l -= 0.14
			}
		}
		if math.Abs(across) > 0.8 {
			l -= 0.12
		}
		l += (hashf(x, y, seed) - 0.5) * 0.06
		return rp.at(int(clamp01(l) * float64(n)))
	})
}

// stalk draws a round stem along the curve through a, b, c and d (art
// pixels), r0 thick at a and r1 at d. mark, if not nil, may recolour each
// pixel from how far along (s, 0 to 1) and across (v, -1 to 1) it lies and
// the light on it.
func (f *flo) stalk(rp ramp, r0, r1, bias float64, a, b, c, d [2]float64, mark func(s, v, l float64, col color.RGBA) color.RGBA) *ctube {
	length := math.Hypot(b[0]-a[0], b[1]-a[1]) + math.Hypot(c[0]-b[0], c[1]-b[1]) + math.Hypot(d[0]-c[0], d[1]-c[1])
	tb := newCTube(bez(max(6, int(length*1.2)), r0, r1, a, b, c, d))
	f.q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
		col := toneOf(rp, l*0.86+0.08+bias)
		if mark != nil {
			col = mark(s, v, l, col)
		}
		return col
	}))
	return tb
}

// lump draws a round fruit or cap of shape s, shaded as one bulge of rp,
// with a glint on its lit shoulder at art point (gx, gy) when glint is set.
func (f *flo) lump(s shape, rp ramp, round, bias float64) {
	f.q.fill(s, shd(rp, s, round, bias))
}

// glint puts a small highlight at art point (x, y).
func (f *flo) glint(x, y float64, col color.RGBA) {
	f.q.dot(x, y, 0, 0, col)
	f.q.dot(x, y, 1, 0, col)
	f.q.dot(x, y, 0, 1, mix(col, colInk, 0.12))
}

// tuft plants n blades of grass round art point (x, y), spread over w.
func (f *flo) tuft(rng *rand.Rand, x, y, w float64, n int, h float64, rp ramp) {
	for i := 0; i < n; i++ {
		bx := (x + (rng.Float64()-0.5)*w) * fk
		bladeHD(f.c, bx, y*fk, (bx-x*fk)*0.5+(rng.Float64()-0.5)*4, (h*(0.6+0.6*rng.Float64()))*fk, 2.2, rp, 0.15, 0.7)
	}
}

func (f *flo) done(ol, ax, ay float64) (*canvas, float64, float64) {
	f.c.outline(ol)
	return f.c, ax * fk, ay * fk
}

// The palettes of the new plants, dark to light.
var (
	bambooRamp     = mkRamp(0x28401e, 0x416428, 0x648a32, 0x8fb146, 0xbcd266, 0xe4ee9e)
	bambooDryRamp  = mkRamp(0x4a3c1e, 0x70602c, 0x988a40, 0xbcb060, 0xdcd48c)
	bambooLeafRamp = mkRamp(0x183a22, 0x27562a, 0x3a7632, 0x55963e, 0x7cb650, 0xa8d470)
	gourdRamp      = mkRamp(0x425c24, 0x678434, 0x90aa4a, 0xb8cc6c, 0xdae498, 0xf2f4c8)
	vineLeafRamp   = mkRamp(0x16361e, 0x245228, 0x377232, 0x50923e, 0x74b250, 0xa2d070)
	lianaRamp      = mkRamp(0x38341a, 0x5a5626, 0x7e7a34, 0xa29c48, 0xc4bc66)
	pitcherRamp    = mkRamp(0x4e1420, 0x7c2428, 0xa84a30, 0xc48a40, 0xd2c060, 0xe8e494)
	pitcherLeaf    = mkRamp(0x1e3e1c, 0x305c24, 0x487e2e, 0x66a03c, 0x8cc054, 0xb8dc7c)
	flaxStem       = mkRamp(0x2c4a26, 0x44682e, 0x628a3a, 0x86aa4c, 0xaac866)
	flaxBlue       = mkRamp(0x2c3e9c, 0x4664cc, 0x6c90ec, 0xa0bcfa, 0xd4e4ff)
	poisonCap      = mkRamp(0x28103e, 0x441e66, 0x683090, 0x8e4eb6, 0xb67cd8, 0xdcb8f2)
	poisonStem     = mkRamp(0x6a7460, 0x94a084, 0xbcc6a4, 0xdce4c4, 0xf2f6e2)
	shadeLeafRamp  = mkRamp(0x12282a, 0x1c4034, 0x285c42, 0x3c7a50, 0x569a5e, 0x7cb874)
	shadeBloom     = mkRamp(0x3c2068, 0x5e3c98, 0x8864c4, 0xb494e4, 0xdcc8f8)
	blackBerry     = mkRamp(0x0c0814, 0x1a1430, 0x30264e, 0x8278b4)
	aloeRamp       = mkRamp(0x1a4236, 0x28604a, 0x3e805c, 0x5ea074, 0x8cc096, 0xbcdeba)
	aloeBloom      = mkRamp(0x9c2c14, 0xd04c1c, 0xf07c2c, 0xfcb04c, 0xffe08c)
	herbRamp       = mkRamp(0x1a4424, 0x2a682e, 0x42903a, 0x64b44a, 0x92d466, 0xc2ec96)
	creamRamp      = mkRamp(0x8c8c74, 0xbcbc9c, 0xe0e0c4, 0xf6f6e4, 0xffffff)
	mossPadRamp    = mkRamp(0x18341a, 0x285220, 0x3e722a, 0x5e9436, 0x88b44a, 0xb4d46c)
)

// --- bamboo ----------------------------------------------------------------------

// bambooCulm draws one cane from (x0, base) up h art pixels, leaning over
// by lean at its top, r thick: green, ringed at every joint.
func (f *flo) bambooCulm(x0, base, h, lean, r, bias float64, rp ramp) *ctube {
	joint := 4.4 + r
	return f.stalk(rp, r, r*0.5, bias,
		[2]float64{x0, base}, [2]float64{x0, base - h*0.4}, [2]float64{x0 + lean*0.45, base - h*0.75}, [2]float64{x0 + lean, base - h},
		func(s, v, l float64, col color.RGBA) color.RGBA {
			u := s * h / joint
			switch fr := u - math.Floor(u); {
			case fr < 0.07:
				return toneOf(rp, l*0.86+bias-0.24) // the joint: a dark ring...
			case fr < 0.15:
				return toneOf(rp, l*0.86+bias+0.3) // ...and the pale band above it
			}
			return col
		})
}

func genBambooHD(full bool) (*canvas, float64, float64) {
	f := newFlo(46, 50)
	rng := rand.New(rand.NewPCG(71, 3))
	const bx, base = 23.0, 46.0
	type culm struct {
		x, h, lean, r float64
		back          bool
		tb            *ctube
	}
	var culms []culm
	for i := 0; i < 9; i++ {
		c := culm{x: bx + (float64(i)-4)*2.2 + (rng.Float64()-0.5)*1.2, h: 27 + rng.Float64()*14, r: 0.85 + rng.Float64()*0.4, back: i%2 == 1}
		c.lean = (c.x-bx)*0.3 + (rng.Float64()-0.5)*3
		if c.back {
			c.h *= 0.85
		}
		if !full {
			c.h, c.lean = 2.2+rng.Float64()*3.2, 0
		}
		culms = append(culms, c)
	}
	// The leaves grow in fans from the joints of the upper half: slender
	// blades hanging outwards.
	leaves := func(c culm, lit float64, front bool) {
		if !full {
			return
		}
		n := int(c.h / 5.4)
		for j := n / 3; j <= n; j++ {
			s := clamp01((float64(j)*5.4 + 1) / c.h)
			p := c.tb.p[min(len(c.tb.p)-1, int(s*float64(len(c.tb.p)-1)))]
			for _, side := range []float64{-1, 1} {
				if hashf(j, int(side)+int(c.x*7), 91) < 0.3 || (hashf(j, int(c.x*3), 92) < 0.5) != front {
					continue
				}
				tw := 1.6 + rng.Float64()*1.6 // a thin twig out from the joint
				tx, ty := p.x+side*tw, p.y-0.6+rng.Float64()*0.6
				f.c.line(p.x*fk, p.y*fk, tx*fk, ty*fk, func(x, y int, _ float64) color.RGBA { return bambooRamp[2] })
				for m := range 3 {
					a := 0.25 + 0.38*float64(m) + rng.Float64()*0.2
					if side < 0 {
						a = math.Pi - a
					}
					f.leafAt(tx, ty, -a, 4.2+rng.Float64()*2.6, 0.62, 0.05, leafLance, bambooLeafRamp, 0, lit+(rng.Float64()-0.5)*0.1)
				}
			}
		}
	}
	for pass := 0; pass < 2; pass++ {
		for i := range culms {
			c := &culms[i]
			if c.back != (pass == 0) {
				continue
			}
			bias := 0.02
			if c.back {
				bias = -0.14
			}
			if c.tb == nil {
				c.tb = newCTube(bez(12, c.r, c.r*0.5, [2]float64{c.x, base}, [2]float64{c.x, base - c.h*0.4}, [2]float64{c.x + c.lean*0.45, base - c.h*0.75}, [2]float64{c.x + c.lean, base - c.h}))
			}
			leaves(*c, bias-0.16, false)
			f.bambooCulm(c.x, base, c.h, c.lean, c.r, bias, bambooRamp)
			if !full { // the cut: a hollow ring
				top := aEll(c.x, base-c.h, c.r, c.r*0.5)
				f.q.flat(top, plain(bambooRamp[4]))
				f.q.flat(aEll(c.x, base-c.h, c.r*0.55, c.r*0.26), plain(bambooRamp[0]))
			}
		}
	}
	for _, c := range culms {
		leaves(c, 0.04, true)
	}
	if !full { // a young shoot coming up: a brown cone in its sheaths
		for _, s := range [][3]float64{{bx + 2.6, 5.5, 1.5}, {bx - 6.5, 3.4, 1.1}} {
			sh := aCap(s[0], base, s[0]+0.3, base-s[1], s[2], 0.2)
			f.q.fill(sh, func(x, y int, _ float64) color.RGBA {
				l := sh.lum(x, y, 2*fk)
				if (y+x/3)%6 < 2 {
					l -= 0.22 // the overlapping sheaths
				}
				return toneOf(bambooDryRamp, l)
			})
		}
	}
	f.tuft(rng, bx, base+0.4, 22, 14, 3.4, bladeRamp)
	for i := 0; i < 5; i++ { // fallen leaves
		f.leafAt(bx-11+rng.Float64()*22, base+0.6+rng.Float64(), rng.Float64()*math.Pi, 2.6, 0.5, 0, leafLance, bambooDryRamp, 0, 0.1)
	}
	return f.done(0.62, bx, base)
}

// --- the gourd vine ----------------------------------------------------------------

// gourd is a bottle gourd hanging at art point (x, y), its neck up, s big.
func (f *flo) gourd(x, y, s float64, rp ramp, seed uint32) {
	stem := aCap(x, y-3.4*s, x+0.2, y-5*s, 0.3, 0.3)
	f.q.fill(stem, plain(vineLeafRamp[1]))
	g := sBlend(1.3*s, aEll(x, y, 2.5*s, 2.7*s), aEll(x, y-3*s, 1.25*s, 1.4*s))
	f.q.fill(g, func(px, py int, _ float64) color.RGBA {
		l := g.lum(px, py, 2.6*fk)*0.92 + 0.06
		if hashf(px/2, py/3, seed) < 0.07 {
			l += 0.14 // pale freckles
		}
		return toneOf(rp, l)
	})
	f.glint(x-1*s, y-1.1*s, rp.at(len(rp)-1))
}

func genGourdHD(full bool) (*canvas, float64, float64) {
	f := newFlo(42, 30)
	rng := rand.New(rand.NewPCG(72, 5))
	const o = 4.0 // room at the sides
	const bx, base = 17.0 + o, 26.0
	// It climbs a fallen bough that arches over the ground.
	drawLimbHD(f.c, limb{(4 + o) * fk, base * fk, (13 + o) * fk, 4 * fk, (30 + o) * fk, (base - 1) * fk, 3.4, 2.2}, trunkRamp, 0.85, 31)
	drawLimbHD(f.c, limb{(18 + o) * fk, 12.5 * fk, (22 + o) * fk, 8 * fk, (26 + o) * fk, 7 * fk, 2, 1.2}, trunkRamp, 0.8, 32)
	arch := limb{4 + o, base, 13 + o, 4, 30 + o, base - 1, 0, 0}
	if full {
		for i, g := range [][3]float64{{12.5 + o, 20.5, 1}, {20.5 + o, 21.5, 0.86}} {
			f.gourd(g[0], g[1], g[2], gourdRamp, 720+uint32(i))
		}
	}
	// The vine winds along the bough, with a broad leaf at every turn.
	prevX, prevY := arch.at(0.04)
	for i := 1; i <= 26; i++ {
		t := 0.04 + 0.92*float64(i)/26
		x, y := arch.at(t)
		y += math.Sin(float64(i)*1.9) * 1.1
		f.c.line(prevX*fk, prevY*fk, x*fk, y*fk, func(px, py int, _ float64) color.RGBA { return vineLeafRamp[2] })
		prevX, prevY = x, y
	}
	for i := 0; i < 15; i++ {
		t := 0.03 + 0.94*(float64(i)+rng.Float64()*0.6)/15
		x, y := arch.at(t)
		a := math.Pi*(0.95-0.9*t) + (rng.Float64()-0.5)*1.3
		if i%3 == 2 {
			a = -a * 0.5 // some hang below the bough
		}
		f.leafAt(x, y-0.4, a, 4.6+rng.Float64()*2, 2.5+rng.Float64()*0.8, 0.08, leafHeart, vineLeafRamp, 5, (rng.Float64()-0.5)*0.14)
	}
	for _, b := range [][2]float64{{9.5 + o, 10.5}, {22.5 + o, 10}, {27 + o, 16.5}} { // white flowers
		bloomHD(f.c, b[0]*fk, b[1]*fk, 4.6, creamRamp[1:], hex(0xf2d24a), rng.Float64())
	}
	if full { // one more, hanging in front of the leaves
		f.gourd(26.2+o, 22.4, 0.72, gourdRamp, 733)
	}
	f.tuft(rng, bx, base+0.4, 26, 10, 3, bladeRamp)
	return f.done(0.62, bx, base)
}

// --- the pitcher plant --------------------------------------------------------------

// pitcher is one cup at art point (x, y), its mouth: s big, full of rain
// water or empty.
func (f *flo) pitcher(x, y, s float64, water bool, seed uint32) {
	body := sBlend(1.2*s, aEll(x, y+3.9*s, 1.75*s, 2.3*s), aCap(x, y+2.6*s, x, y+0.4*s, 1.3*s, 1.55*s))
	f.q.fill(body, func(px, py int, _ float64) color.RGBA {
		ay := (float64(py)/fk - y) / (6.2 * s) // 0 at the mouth, 1 at the foot
		l := body.lum(px, py, 2.2*fk)*0.8 + 0.34 - 0.22*(1-ay)
		// Maroon blotches, thick at the mouth and thinning down the cup.
		if vnoise(float64(px)*0.42, float64(py)*0.3, seed) > 0.42+0.38*ay {
			l -= 0.36
		}
		return toneOf(pitcherRamp, l)
	})
	lid := aEll(x+0.9*s, y-1.5*s, 1.7*s, 0.75*s).turned(x+0.9*s, y-1.5*s, -0.45)
	f.q.fill(lid, shd(pitcherRamp, lid, 1.2, -0.12))
	rim := aEll(x, y+0.2*s, 1.75*s, 0.8*s)
	f.q.fill(rim, plain(pitcherRamp[1]))
	f.q.flat(aEll(x, y+0.05*s, 1.75*s, 0.8*s).sub(aEll(x, y+0.2*s, 1.3*s, 0.5*s)), plain(pitcherRamp[2]))
	in := aEll(x, y+0.25*s, 1.25*s, 0.48*s)
	if water {
		f.q.flat(in, plain(hex(0x3c8cae)))
		f.q.flat(aEll(x-0.25*s, y+0.15*s, 0.75*s, 0.22*s), plain(hex(0x9ce0ee)))
		f.q.dot(x-0.5*s, y+0.1*s, 0, 0, colWhite)
	} else {
		f.q.flat(in, plain(pitcherRamp[0]))
	}
}

func genPitcherHD(full bool) (*canvas, float64, float64) {
	f := newFlo(32, 28)
	rng := rand.New(rand.NewPCG(73, 7))
	const bx, base = 16.0, 25.0
	// A rosette of long strap leaves; from the tips of three hang the cups,
	// each on a tendril.
	var cups [][3]float64
	for i, a := range []float64{2.62, 0.5, 1.5, 2.95, 0.2, 2.2, 0.95} {
		l, sag := 9+rng.Float64()*2, 0.3-0.25*math.Sin(a)
		if i < 3 { // these three arch high and carry a cup
			l, sag = 12.5+rng.Float64()*1.5, 0.16
		}
		f.leafAt(bx, base-0.5, a, l, 1.6, sag, leafStrap, pitcherLeaf, 0, -0.1+0.035*float64(i))
		if i < 3 { // where the leaf ends, a tendril drops to the cup's back
			tx, ty := bx+math.Cos(a)*l, base-0.5-math.Sin(a)*l+sag*l
			cx, cy := tx+math.Copysign(1.2, math.Cos(a)), math.Min(ty+2.6, base-5.6)
			if i == 2 {
				cx = tx + 2.4
			}
			cups = append(cups, [3]float64{cx, cy, 0.84 + 0.1*float64(i%2)})
			f.stalk(pitcherLeaf, 0.3, 0.3, 0, [2]float64{tx, ty}, [2]float64{tx, ty + 1.4}, [2]float64{cx + 1.2, cy - 1.6}, [2]float64{cx + 1.1, cy + 0.6}, nil)
		}
	}
	for i, cp := range cups {
		f.pitcher(cp[0], cp[1], cp[2], full, 740+uint32(i))
	}
	f.tuft(rng, bx, base+0.4, 18, 8, 2.6, bladeRamp)
	return f.done(0.62, bx, base)
}

// --- flax ---------------------------------------------------------------------------

func genFlaxHD(full bool) (*canvas, float64, float64) {
	f := newFlo(36, 30)
	rng := rand.New(rand.NewPCG(74, 9))
	const bx, base = 18.0, 28.0
	type stem struct{ x, h, lean float64 }
	var stems []stem
	for i := 0; i < 9; i++ {
		// From a narrow foot the stems fan out, thin and airy.
		s := stem{x: bx - 2.6 + float64(i)*0.65 + rng.Float64()*0.4, h: 13 + rng.Float64()*9}
		s.lean = (float64(i)-4)*1.9 + (rng.Float64()-0.5)*2
		if !full {
			s.h, s.lean = 2+rng.Float64()*2.2, s.lean*0.15
		}
		stems = append(stems, s)
	}
	for i, s := range stems {
		lo := 0.1 + 0.25*float64(i%3)/2
		bladeHD(f.c, s.x*fk, base*fk, s.lean*fk, s.h*fk, 2, flaxStem, lo, lo+0.5)
		if !full {
			continue
		}
		for j := 1; j < int(s.h/3.2); j++ { // narrow leaves up the stem, left and right in turn
			t := (float64(j)*3.2 + float64(i%2)*1.4) / s.h
			x, y := s.x+s.lean*t*t, base-s.h*t
			a := 0.9 + rng.Float64()*0.4
			if (j+i)%2 == 0 {
				a = math.Pi - a
			}
			f.leafAt(x, y, a, 2.8+rng.Float64(), 0.45, 0.1, leafLance, flaxStem, 0, 0.12)
		}
	}
	if full {
		for i, s := range stems {
			x, y := (s.x+s.lean)*fk, (base-s.h)*fk
			switch i % 4 {
			case 2: // a round seed boll, ripening
				f.lump(aEll(x/fk, y/fk, 0.75, 0.8), bambooDryRamp, 1.2, 0.2)
			default:
				bloomHD(f.c, x, y, 4.4, flaxBlue, hex(0xf6e27a), rng.Float64())
			}
		}
	}
	f.tuft(rng, bx, base+0.5, 9, 7, 3, bladeRamp)
	return f.done(0.5, bx, base)
}

// --- the jungle vine -----------------------------------------------------------------

// liana draws a rope of a vine along the curve through a, b, c, d: two
// strands twisted together.
func (f *flo) liana(r float64, a, b, c, d [2]float64) *ctube {
	length := newCTube(bez(16, r, r, a, b, c, d)).len[16]
	return f.stalk(lianaRamp, r, r*0.8, 0, a, b, c, d, func(s, v, l float64, col color.RGBA) color.RGBA {
		u := s*length/1.5 + v*0.5
		if u-math.Floor(u) < 0.34 {
			return toneOf(lianaRamp, l*0.86-0.14) // the groove between the strands
		}
		return col
	})
}

func genVineHD(full bool) (*canvas, float64, float64) {
	f := newFlo(32, 36)
	rng := rand.New(rand.NewPCG(75, 11))
	const bx, base = 16.0, 32.0
	// A dead snag for the lianas to hang on.
	drawLimbHD(f.c, limb{bx * fk, base * fk, (bx - 1.5) * fk, 20 * fk, (bx - 0.5) * fk, 7 * fk, 7.5, 4.2}, trunkRamp, 0.85, 41)
	drawLimbHD(f.c, limb{(bx - 0.8) * fk, 14 * fk, 21 * fk, 11 * fk, 26.5 * fk, 12.5 * fk, 3.6, 2}, trunkRamp, 0.8, 42)
	drawLimbHD(f.c, limb{(bx - 1) * fk, 18 * fk, 10 * fk, 15 * fk, 6.5 * fk, 16.5 * fk, 3, 1.6}, trunkRamp, 0.75, 43)
	knotHD(f.c, (bx-0.6)*fk, 24*fk, 2.4, trunkRamp)
	var vines []*ctube
	// Always: one old liana wound up the trunk.
	vines = append(vines, f.liana(0.8, [2]float64{bx - 5, base + 0.5}, [2]float64{bx + 4.5, 27}, [2]float64{bx - 5.5, 19}, [2]float64{bx + 1.5, 12.5}))
	if full {
		vines = append(vines,
			f.liana(0.85, [2]float64{26, 12.6}, [2]float64{28.5, 30}, [2]float64{18.5, 30}, [2]float64{bx + 1.2, 13.5}), // a loop hanging from the right limb
			f.liana(0.7, [2]float64{6.8, 16.6}, [2]float64{2.5, 24}, [2]float64{7, 29}, [2]float64{4.5, base + 0.6}),    // down from the left limb to the ground
			f.liana(0.6, [2]float64{23, 12}, [2]float64{24.5, 18}, [2]float64{21, 21}, [2]float64{23.5, 24.5}),          // a loose end, curling
			f.liana(0.7, [2]float64{bx - 0.5, 7.4}, [2]float64{bx + 6, 2}, [2]float64{24, 6}, [2]float64{26.8, 12}))
	}
	for i, tb := range vines {
		step := 5
		if !full {
			step = 8
		}
		for j := 2 + i%3; j < len(tb.p)-1; j += step {
			p := tb.p[j]
			a := rng.Float64() * 2 * math.Pi
			f.leafAt(p.x, p.y, a, 2.6+rng.Float64()*1.2, 1.4+rng.Float64()*0.4, 0.1, leafHeart, vineLeafRamp, 4, (rng.Float64()-0.5)*0.16-0.03*float64(i%2))
		}
	}
	f.tuft(rng, bx, base+0.6, 18, 9, 3, bladeRamp)
	return f.done(0.62, bx, base)
}

// --- the poison mushroom ----------------------------------------------------------------

// toadstool is one mushroom standing on art point (x, base), h tall, its
// cap r wide; picked, only the stump of its stem is left.
func (f *flo) toadstool(x, base, h, r, tilt float64, picked bool, seed uint32) {
	tx := x + tilt*h
	if picked {
		st := aCap(x, base, x+tilt, base-1.2, 0.62, 0.55)
		f.lump(st, poisonStem, 1, 0)
		f.q.flat(aEll(x+tilt, base-1.3, 0.55, 0.28), plain(poisonStem[4]))
		return
	}
	st := aCap(x, base, tx, base-h, 0.7, 0.5)
	f.lump(st, poisonStem, 1.1, 0.02)
	skirt := aEll(lerpf(x, tx, 0.62), base-h*0.62, 1.05, 0.45) // the torn ring on the stem
	f.q.fill(skirt, shd(poisonStem, skirt, 0.8, -0.1))
	gills := aEll(tx, base-h+0.25, r*0.92, r*0.3)
	f.q.fill(gills, func(px, py int, _ float64) color.RGBA {
		if px%2 == 0 {
			return poisonCap[4]
		}
		return poisonCap[2]
	})
	cap := aEll(tx, base-h, r, r*0.78).and(sAbove((base - h + 0.25) * fk))
	f.q.fill(cap, func(px, py int, _ float64) color.RGBA {
		l := cap.lum(px, py, 2.4*fk)*0.9 + 0.08
		// Sickly yellow-green warts.
		cx, cy := math.Floor(float64(px)/5), math.Floor(float64(py)/4)
		wx, wy := (cx+0.3+0.4*hashf(int(cx), int(cy), seed))*5, (cy+0.3+0.4*hashf(int(cx), int(cy), seed+1))*4
		if hashf(int(cx), int(cy), seed+2) < 0.62 && math.Hypot(float64(px)+0.5-wx, (float64(py)+0.5-wy)*1.25) < 1.35 {
			return mix(hex(0xe2f48a), hex(0x8aa83c), clamp01(1.1-l*1.3))
		}
		return toneOf(poisonCap, l)
	})
	f.glint(tx-r*0.42, base-h-r*0.42, poisonCap[5])
}

func genPoisonShroomHD(full bool) (*canvas, float64, float64) {
	f := newFlo(30, 22)
	rng := rand.New(rand.NewPCG(76, 13))
	const bx, base = 15.0, 19.0
	f.tuft(rng, bx, base+0.4, 18, 7, 2.6, bladeRamp)
	f.toadstool(bx+5.4, base-0.3, 5, 2.4, 0.12, !full, 761)
	f.toadstool(bx-5.2, base, 6.2, 2.9, -0.1, !full, 762)
	f.toadstool(bx+0.4, base+0.6, 9.4, 4.3, 0.03, !full, 763)
	f.toadstool(bx+3.2, base+1.2, 3, 1.5, 0.2, !full, 764)
	f.tuft(rng, bx, base+1.2, 16, 5, 2, bladeRamp)
	return f.done(0.66, bx, base)
}

// --- nightshade -----------------------------------------------------------------------

func genNightshadeHD(full bool) (*canvas, float64, float64) {
	f := newFlo(30, 24)
	rng := rand.New(rand.NewPCG(77, 15))
	const bx, base = 15.0, 21.0
	for _, s := range [][4]float64{{bx - 1, base, bx - 6, base - 9}, {bx, base, bx + 1, base - 12}, {bx + 1, base, bx + 7, base - 8}} {
		f.stalk(mkRamp(0x2a1c34, 0x44305a, 0x5e4a70, 0x7a6688), 0.6, 0.35, 0, [2]float64{s[0], s[1]}, [2]float64{s[0], s[1] - 4}, [2]float64{s[2], s[3] + 4}, [2]float64{s[2], s[3]}, nil)
	}
	blobs := scaledBlobs([]blob{{8, 13, 5}, {22, 13, 5}, {15, 10, 7}, {11, 7, 4.2}, {19, 7, 4.2}, {15, 15, 5}})
	leafClusterHD(f.c, blobs, bx*fk, 11*fk, 12*fk, 9*fk, shadeLeafRamp, 770, leafLook{5.2, 5, 2.5})
	for _, b := range [][2]float64{{9, 8}, {20.5, 6.5}, {15.5, 4.2}, {24.5, 12}} { // violet stars with a yellow beak
		bloomHD(f.c, b[0]*fk, b[1]*fk, 4.2, shadeBloom, hex(0xf2c83c), rng.Float64())
	}
	if full {
		for i, b := range [][2]float64{{6.5, 14}, {12, 11.5}, {18, 13.5}, {23, 16}, {10.5, 17}, {15.5, 8.5}, {20.5, 10.5}} {
			x, y := b[0]*fk, b[1]*fk
			// A green star of a calyx behind each shining black berry.
			for k := range 5 {
				a := float64(k)*2*math.Pi/5 + 0.3
				f.c.line(x, y, x+math.Cos(a)*4.6, y+math.Sin(a)*4.2, func(px, py int, _ float64) color.RGBA { return shadeLeafRamp[4] })
			}
			berryHD(f.c, x, y, 3.2, blackBerry, shadeLeafRamp[0])
			if i%2 == 0 {
				berryHD(f.c, x+4.6, y+2.2, 2.6, blackBerry, shadeLeafRamp[0])
			}
		}
	}
	return f.done(0.72, bx, base)
}

// --- aloe -------------------------------------------------------------------------------

// aloeLeaf is one thick leaf from (x0, y0) to (x1, y1), arching, r thick at
// its foot; cut, it ends short in a pale wet face.
func (f *flo) aloeLeaf(x0, y0, x1, y1, r, bias float64, cut bool, seed uint32) {
	mx, my := (x0+x1)/2+(y1-y0)*0.12, (y0+y1)/2-math.Abs(x1-x0)*0.22
	if cut {
		x1, y1 = lerpf(x0, mx, 0.8), lerpf(y0, my, 0.8)
		s := aCap(x0, y0, x1, y1, r, r*0.8)
		f.q.fill(s, shd(aloeRamp, s, 1.6, bias))
		f.q.flat(aEll(x1, y1, r*0.75, r*0.6), plain(hex(0xcfe6b4)))
		f.q.flat(aEll(x1, y1, r*0.4, r*0.3), plain(hex(0xeaf6d4)))
		return
	}
	s := aPath(x0, y0, r, mx, my, r*0.72, x1, y1, 0.16)
	f.q.fill(s, func(px, py int, d float64) color.RGBA {
		l := s.lum(px, py, 1.7*fk)*0.9 + 0.06 + bias
		switch {
		case d > -1.3 && (px+py*2)%5 == 0:
			return aloeRamp[5] // the pale teeth along its edges
		case hashf(px/3, py/2, seed) < 0.1:
			l += 0.16 // and the pale flecks on its skin
		}
		return toneOf(aloeRamp, l)
	})
}

func genAloeHD(full bool) (*canvas, float64, float64) {
	f := newFlo(30, 30)
	rng := rand.New(rand.NewPCG(78, 17))
	const bx, base = 15.0, 25.0
	if full { // the flower spike, behind the leaves
		f.stalk(mkRamp(0x5a3a24, 0x86583a, 0xb07c50, 0xd0a070), 0.5, 0.3, 0, [2]float64{bx + 0.5, base - 4}, [2]float64{bx + 1.5, base - 12}, [2]float64{bx + 1.2, base - 17}, [2]float64{bx + 2, base - 21.5}, nil)
		for i := 0; i < 11; i++ { // drooping tubes, red at the top and yellowing down
			t := float64(i) / 10
			y := base - 22 + t*7.5
			x := bx + 1.9 - t*0.5
			side := float64(i%2)*2 - 1
			fl := aCap(x, y, x+side*(1.3+t*0.5), y+1.5, 0.52, 0.42)
			tone := 0.5 + 0.4*t
			f.q.fill(fl, func(px, py int, _ float64) color.RGBA { return toneOf(aloeBloom, fl.lum(px, py, 1*fk)*0.5+tone-0.25) })
		}
	}
	// The upright leaves first, then the ones that spread and lie in front.
	type lf struct{ a, l, r float64 }
	leaves := []lf{{1.45, 12, 1.5}, {1.9, 11.5, 1.5}, {1.1, 11, 1.5}, {2.35, 11, 1.6}, {0.72, 11, 1.6}, {2.75, 10, 1.7}, {0.36, 10.5, 1.7}, {3.05, 8, 1.6}, {0.08, 8.5, 1.6}, {1.62, 7, 1.7}}
	for i, l := range leaves {
		ln := l.l * (0.9 + 0.2*rng.Float64())
		cut := !full && i%2 == 1
		f.aloeLeaf(bx+math.Cos(l.a)*1.2, base-0.6, bx+math.Cos(l.a)*ln, base-0.6-math.Sin(l.a)*ln*0.92, l.r, -0.1+0.025*float64(i), cut, 780+uint32(i))
	}
	return f.done(0.66, bx, base)
}

// --- the healing herb ---------------------------------------------------------------------

func genHerbHD(full bool) (*canvas, float64, float64) {
	f := newFlo(26, 26)
	rng := rand.New(rand.NewPCG(79, 19))
	const bx, base = 13.0, 23.0
	if full { // the flowering stems: flat heads of tiny cream flowers
		for i, s := range [][3]float64{{-4.4, 14.5, -1}, {0.6, 18, 0.2}, {4.8, 13, 1}} {
			tx, ty := bx+s[0], base-s[1]
			f.stalk(herbRamp, 0.45, 0.3, -0.12, [2]float64{bx + s[2]*0.6, base - 2}, [2]float64{bx + s[2], base - s[1]*0.5}, [2]float64{tx, ty + 4}, [2]float64{tx, ty}, nil)
			f.leafAt(lerpf(bx, tx, 0.6), base-s[1]*0.55, 0.7+float64(i), 3.4, 0.9, 0.1, leafSaw, herbRamp, 0, -0.05)
			for k := 0; k < 13; k++ { // the head: a low dome of florets on thin rays
				a := math.Pi * (0.08 + 0.84*float64(k)/12)
				rr := 2.6 + 0.9*hashf(k, i, 791)
				x, y := tx-math.Cos(a)*rr*1.25, ty-math.Sin(a)*rr*0.62-0.3
				f.c.line(tx*fk, (ty+0.8)*fk, x*fk, y*fk, func(px, py int, _ float64) color.RGBA { return herbRamp[2] })
				f.c.ellipse(x*fk, y*fk, 2.3, 2, func(px, py int, nx, ny float64) color.RGBA {
					return creamRamp.pick(0.72+lum(nx, ny)*0.4-0.12*float64(k%2), px, py)
				})
				f.c.put(int(x*fk), int(y*fk), hex(0xf2dc7a))
			}
		}
	}
	// A clump of toothed leaves, bright and fresh.
	n := 9
	for i := 0; i < n; i++ {
		t := (float64(i) + 0.5) / float64(n)
		a := math.Pi * (0.04 + 0.92*t)
		if i%2 == 1 {
			a = math.Pi - a // back and forth, so they overlap from both sides
		}
		l := 6.5 + rng.Float64()*3
		if !full {
			l *= 0.5
		}
		f.leafAt(bx+math.Cos(a)*0.8, base-0.4, a, l, 1.9, 0.22-0.2*math.Sin(a), leafSaw, herbRamp, 6, (rng.Float64()-0.5)*0.14)
	}
	return f.done(0.6, bx, base)
}

// --- moss ----------------------------------------------------------------------------------

// mossPad is a cushion of moss at image point (x, y), rx by ry: a stipple of
// tiny shoots, lit from the upper left, drawn only on what is there already
// (or anywhere, with free).
func mossPad(c *canvas, x, y, rx, ry float64, free bool, seed uint32) {
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		if !free && !c.opaque(px, py) || d > 0.72+0.28*hashf(px/2, py/2, seed) {
			return color.RGBA{}
		}
		l := lum(nx, ny)*0.72 + 0.12 + (hashf(px, py, seed+1)-0.5)*0.42
		if (px+py)%3 == 0 {
			l += 0.1 // the tips of the shoots
		}
		return mossPadRamp.pick(l, px, py)
	})
}

func genMossHD(full bool) (*canvas, float64, float64) {
	c := hdCanvas(34, 18)
	const bx, base = 17.0 * fk, 14.0 * fk
	stoneHD(c, bx-12, base-10, 30, 17, 1, stoneRamp, 801)
	stoneHD(c, bx+24, base-5, 13, 9, 0.9, stoneRamp, 802)
	stoneHD(c, bx-31, base-3, 8, 6, 0.85, stoneRamp, 803)
	pads := [][4]float64{{-20, -19, 15, 8}, {-2, -22, 14, 8}, {8, -14, 10, 7}, {24, -10, 10, 5.5}, {-12, -11, 9, 5}, {-31, -6, 6, 3.5}}
	if !full {
		pads = [][4]float64{{-22, -20, 6, 3.5}, {6, -20, 5, 3}, {24, -11, 4, 2.5}}
	}
	for i, p := range pads {
		mossPad(c, bx+p[0], base+p[1], p[2], p[3], full && i < 3, 810+uint32(i))
	}
	if full { // the thin stalks of its capsules
		for i := 0; i < 9; i++ {
			x, y := int(bx-26+float64(i)*5.6+hashf(i, 0, 820)*3), int(base-25+hashf(i, 1, 820)*5)
			if !c.opaque(x, y+3) {
				continue
			}
			for d := 0; d < 4; d++ {
				c.put(x, y-d+2, mossPadRamp[4])
			}
			c.put(x, y-2, hex(0xb8683a))
			c.put(x, y-3, hex(0xd88a4c))
		}
	}
	c.outline(0.7)
	return c, bx, base
}

// --- the sprites ---------------------------------------------------------------------------

// floraGens draws each wild plant, full or picked.
var floraGens = [sim.NumFlora]func(full bool) (*canvas, float64, float64){
	sim.FloraBamboo:         genBambooHD,
	sim.FloraGourd:          genGourdHD,
	sim.FloraPitcher:        genPitcherHD,
	sim.FloraFlax:           genFlaxHD,
	sim.FloraVine:           genVineHD,
	sim.FloraPoisonMushroom: genPoisonShroomHD,
	sim.FloraNightshade:     genNightshadeHD,
	sim.FloraAloe:           genAloeHD,
	sim.FloraHerb:           genHerbHD,
	sim.FloraMoss:           genMossHD,
}

var floraArt [sim.NumFlora][2]sprite

// floraSprite is the sprite of wild plant f, full or picked, drawn the
// first time it is wanted.
func floraSprite(f sim.FloraKind, full bool) sprite {
	f %= sim.NumFlora
	i := 0
	if full {
		i = 1
	}
	if floraArt[f][i].img == nil {
		floraArt[f][i] = mkUp(floraGens[f](full))
	}
	return floraArt[f][i]
}
