package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"survivalgame/internal/sim"
)

// The small plants of the ground at HD, each in several drawings and for
// each season: green grass, tall grass (the tufts that give fibre) and its
// stubble, round leafy clumps, ferns, wild flowers, broad leaves and berry
// bushes. Where the old drawings stood on the ground like stickers, these
// grow out of it: no outline along the foot, the lowest blades and leaves
// dark and fading into the ground, and a soft shadow where they meet it.
// Every plant is drawn flipped or not, a little bigger or smaller and a
// shade lighter, darker, warmer or cooler by where it grows (coverDraw), so
// no two neighbours look stamped from one mould.

// coverKind is a kind of small plant.
type coverKind uint8

const (
	cvGrass    coverKind = iota // a tuft of green grass (undergrowth)
	cvTall                      // tall grass: a sim.EntityGrassTuft that can be cut
	cvStubble                   // tall grass cut
	cvClump                     // a round, low clump of small leaves
	cvFern                      // a fern
	cvFlowers                   // wild flowers in a little patch
	cvBroad                     // a few big leaves
	cvBush                      // a berry bush with berries
	cvBushBare                  // a berry bush picked
	numCoverKinds
)

// coverVariants is how many drawings there are of each kind.
var coverVariants = [numCoverKinds]int{6, 6, 4, 5, 5, 6, 4, 4, 4}

// coverPal is the colours of the small plants in one season.
type coverPal struct {
	grass, tall, head, leaf, fern, broad, bush ramp
	bloom                                      float64 // 0: no flowers open, 1: all; below 0: gone to seed
	snow                                       bool    // snow lies on them
	blossom                                    bool    // white blossom on the bushes
}

// coverPals are the plants' colours by season. Summer's tall grass is a
// muted straw with olive in it, so it sits among the greens; spring's is
// still green, autumn's golden, winter's bleached under snow.
var coverPals = [sim.NumSeasons]coverPal{
	sim.Spring: {
		grass:   mkRamp(0x1f4422, 0x2c5d2a, 0x3d7833, 0x52923c, 0x6dab47, 0x8ec456, 0xb4dc78),
		tall:    mkRamp(0x2c4a22, 0x3e6229, 0x557a31, 0x70903c, 0x8ea24c, 0xadb466, 0xcfcd92),
		head:    mkRamp(0x5b6d2e, 0x7c8c3d, 0x9fab55, 0xc2c97a),
		leaf:    mkRamp(0x16361f, 0x214d29, 0x2f6732, 0x43833b, 0x5f9f46, 0x86bc57, 0xb0d677),
		fern:    mkRamp(0x123024, 0x1a432b, 0x255c34, 0x34763c, 0x4b9046, 0x6aa852, 0x93c66a),
		broad:   mkRamp(0x0f3326, 0x174a31, 0x22633a, 0x337e43, 0x4d984c, 0x70b05a, 0x98c977),
		bush:    mkRamp(0x173b2a, 0x22533a, 0x2f6c42, 0x46884c, 0x67a55a, 0x8dc06c),
		bloom:   1,
		blossom: true,
	},
	sim.Summer: {
		grass: mkRamp(0x1f4020, 0x2b5628, 0x3a6f31, 0x4e8839, 0x67a043, 0x85b752, 0xa8cc6e),
		tall:  mkRamp(0x3d4a24, 0x52602d, 0x6a7637, 0x838b45, 0x9c9e56, 0xb4b06c, 0xcac289),
		head:  mkRamp(0x6e6a3a, 0x8c8549, 0xaaa060, 0xc6bb80),
		leaf:  mkRamp(0x15331e, 0x1f4927, 0x2c6230, 0x3f7d38, 0x589843, 0x7cb452, 0xa3cc6f),
		fern:  mkRamp(0x122d22, 0x1a4029, 0x245832, 0x33713a, 0x488a43, 0x64a24e, 0x8abd63),
		broad: mkRamp(0x0f3125, 0x16472f, 0x215f38, 0x317a41, 0x4a9449, 0x6bab56, 0x91c372),
		bush:  mkRamp(0x173b2a, 0x22533a, 0x2f6c42, 0x46884c, 0x67a55a, 0x8dc06c),
		bloom: 0.6,
	},
	sim.Autumn: {
		grass: mkRamp(0x2e4020, 0x3e5527, 0x546b2f, 0x6e823a, 0x8b9747, 0xa9a85a, 0xc8bb76),
		tall:  mkRamp(0x5a3e1a, 0x7a5521, 0x9a6f2b, 0xb98a37, 0xd3a647, 0xe6c162, 0xf3db92),
		head:  mkRamp(0x7a4a1e, 0x9c6528, 0xbd8436, 0xdba653),
		leaf:  mkRamp(0x2a321a, 0x3c4720, 0x545e28, 0x6e7531, 0x8c8a3a, 0xae9a45, 0xcfb05a),
		fern:  mkRamp(0x2e1d10, 0x472a14, 0x633a19, 0x814c1f, 0x9e6127, 0xba7a35, 0xd29a50),
		broad: mkRamp(0x24301a, 0x34441f, 0x4a5a26, 0x63712e, 0x7f8836, 0x9f9c40, 0xc0b052),
		bush:  mkRamp(0x262a18, 0x3a3e1f, 0x515426, 0x6a6a2f, 0x858038, 0xa09646),
		bloom: -1,
	},
	sim.Winter: {
		grass: mkRamp(0x35402e, 0x46523a, 0x5a6548, 0x6f7958, 0x868c6c, 0x9ea183, 0xb7b79e),
		tall:  mkRamp(0x5a5444, 0x6f6855, 0x857d67, 0x9b927b, 0xb0a790, 0xc4bca6, 0xd8d1bd),
		head:  mkRamp(0x6e6450, 0x8a7f68, 0xa69b84, 0xc1b8a2),
		leaf:  mkRamp(0x1d2c22, 0x283b2b, 0x354c34, 0x445e3d, 0x567048, 0x6b8457, 0x849a6c),
		fern:  mkRamp(0x2a2018, 0x3b2c1f, 0x4e3a27, 0x624a31, 0x775c3d, 0x8c6f4c, 0xa0835e),
		broad: mkRamp(0x1a2a22, 0x24392b, 0x304934, 0x3e5a3d, 0x4e6b48, 0x627e57, 0x7a926a),
		bush:  mkRamp(0x1c2c26, 0x263a2f, 0x334a38, 0x425c42, 0x546f4e, 0x6b845e),
		bloom: -2,
		snow:  true,
	},
}

// coverSet is every drawing of every kind for one season.
type coverSet [numCoverKinds][]sprite

var coverSets [sim.NumSeasons]lazy[coverSet]

func init() {
	for s := range sim.NumSeasons {
		coverSets[s].gen = func() coverSet {
			var set coverSet
			for k := range numCoverKinds {
				set[k] = make([]sprite, coverVariants[k])
				for v := range coverVariants[k] {
					set[k][v] = genCover(k, v, s)
				}
			}
			return set
		}
	}
}

// coverSprites is the season's set, drawn the first time it is wanted.
func coverSprites(s sim.Season) coverSet { return coverSets[s%sim.NumSeasons].get() }

// coverSprite is drawing v (any number: it wraps) of kind k in season s.
func coverSprite(k coverKind, v int, s sim.Season) sprite {
	set := coverSprites(s)
	n := len(set[k])
	return set[k][((v%n)+n)%n]
}

// genCover draws variant v of kind k for season s.
func genCover(k coverKind, v int, s sim.Season) sprite {
	c, ax, ay := genCoverCanvas(k, v, s)
	return mkUp(c, ax, ay)
}

// genCoverCanvas is genCover on a canvas, with its anchor.
func genCoverCanvas(k coverKind, v int, s sim.Season) (*canvas, float64, float64) {
	p := coverPals[s%sim.NumSeasons]
	rng := rand.New(rand.NewPCG(uint64(k)*977+uint64(v)*131+5, uint64(v)*7+uint64(k)+11))
	switch k {
	case cvGrass:
		return genGrassCover(v, rng, p)
	case cvTall:
		return genTallCover(v, rng, p)
	case cvStubble:
		return genStubbleCover(v, rng, p)
	case cvClump:
		return genClumpCover(v, rng, p)
	case cvFern:
		return genFernCover(v, rng, p)
	case cvFlowers:
		return genFlowerCover(v, rng, p)
	case cvBroad:
		return genBroadCover(v, rng, p)
	case cvBush:
		return genBushCover(v, rng, p, true)
	default:
		return genBushCover(v, rng, p, false)
	}
}

// --- tools ------------------------------------------------------------------------

// rampAt is the colour at t along rp, blended between its steps, with a
// faint grain so a broad shape is not a flat gradient.
func rampAt(rp ramp, t float64, x, y int, seed uint32) color.RGBA {
	t = clamp01(t+(hashf(x, y, seed)-0.5)*0.05) * float64(len(rp)-1)
	i := min(int(t), len(rp)-2)
	return mix(rp[i], rp[i+1], t-float64(i))
}

// coverBlade draws one blade of grass from its foot (x0, y0), in image
// pixels: it rises h, its tip ends lean to the side and droop of h back
// down (an arching blade), w wide at the foot and tapering to a point. Its
// tone runs from lo at the foot to hi at the tip, lit along its left edge.
// snow lays white on the upper side of its last stretch.
func coverBlade(c *canvas, x0, y0, h, lean, droop, w float64, rp ramp, lo, hi float64, snow bool, seed uint32) {
	// A quadratic curve: up first, then over.
	x1, y1 := x0+lean*0.25, y0-h*(1.05+droop*0.6)
	x2, y2 := x0+lean, y0-h*(1-droop)
	n := int((h+math.Abs(lean))*2.2) + 4
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		cx, cy := u*u*x0+2*u*t*x1+t*t*x2, u*u*y0+2*u*t*y1+t*t*y2
		tx, ty := 2*u*(x1-x0)+2*t*(x2-x1), 2*u*(y1-y0)+2*t*(y2-y1)
		d := math.Hypot(tx, ty)
		if d == 0 {
			continue
		}
		nx, ny := -ty/d, tx/d // across the blade, to its left when it rises
		hw := max(0.5, w/2*math.Pow(1-t, 0.75))
		for s := -hw; s <= hw+0.01; s += 0.45 {
			px, py := cx+nx*s, cy+ny*s
			ix, iy := int(math.Floor(px)), int(math.Floor(py))
			l := lo + (hi-lo)*math.Pow(t, 0.8) + 0.13*s/hw*-math.Copysign(1, nx+0.001)
			col := rampAt(rp, l, ix, iy, seed)
			if snow && t > 0.72 && s*ny < 0.2 {
				col = mix(col, hex(0xf2f6fc), clamp01((t-0.72)*6))
			}
			c.put(ix, iy, col)
		}
	}
}

// coverFinish gives a plant drawn on c, standing at (bx, by) image pixels,
// its soft foot and turns it into a sprite: a faint outline round its top,
// its lowest pixels darker and fading into the ground, and under it a soft
// shadow rx pixels across.
func coverFinish(c *canvas, bx, by, rx, fade float64) (*canvas, float64, float64) {
	c.outline(0.28)
	root := hex(0x16220f)
	rootW := rx * 0.55 // the foot fades most round the root; leaf tips lying further out, less
	for y := 0; y < c.h; y++ {
		dy := by - (float64(y) + 0.5) // pixels above the ground
		if dy > fade {
			continue
		}
		f := clamp01(dy / fade) // 0 at the ground, 1 at the top of the band
		for x := 0; x < c.w; x++ {
			i := y*c.w + x
			p := c.px[i]
			if p.A == 0 {
				continue
			}
			wr := 1 - smooth(clamp01((math.Abs(float64(x)+0.5-bx)-rootW)/rootW))
			wr = 0.3 + 0.7*wr
			if c.ol != nil && c.ol[i] {
				p.A = uint8(float64(p.A) * lerpf(1, f*f, wr)) // no line along the foot
			} else {
				p = mix(p, root, 0.45*(1-f)*wr)
				p.A = uint8(float64(p.A) * lerpf(1, 0.25+0.75*smooth(f), wr))
			}
			c.px[i] = p
		}
	}
	// The contact shadow, under everything.
	ry := rx * 0.34
	sh := hex(0x0f180a)
	for y := int(by - ry - 1); y <= int(by+ry+1); y++ {
		for x := int(bx - rx - 1); x <= int(bx+rx+1); x++ {
			if !c.in(x, y) {
				continue
			}
			dx, dy := (float64(x)+0.5-bx)/rx, (float64(y)+0.5-by+ry*0.25)/ry
			d := dx*dx + dy*dy
			if d >= 1 {
				continue
			}
			a := 0.4 * math.Pow(1-d, 1.6)
			coverUnder(c, x, y, withAlpha(sh, uint8(a*255)))
		}
	}
	return c, bx, by
}

// coverUnder lays col beneath what is at (x, y), by its alpha.
func coverUnder(c *canvas, x, y int, col color.RGBA) {
	o := c.px[y*c.w+x]
	if o.A == 255 || col.A == 0 {
		return
	}
	if o.A == 0 {
		c.px[y*c.w+x] = col
		return
	}
	ao, ac := float64(o.A)/255, float64(col.A)/255
	a := ao + ac*(1-ao)
	m := mix(col, o, ao/a)
	m.A = uint8(math.Round(a * 255))
	c.px[y*c.w+x] = m
}

// coverSnow lays snow on every top edge of what is drawn on c, deep pixels
// thick where it lies flat.
func coverSnow(c *canvas, deep int, seed uint32) {
	src := make([]color.RGBA, len(c.px))
	copy(src, c.px)
	at := func(x, y int) uint8 {
		if !c.in(x, y) {
			return 0
		}
		return src[y*c.w+x].A
	}
	snow := mkRamp(0xb9c6d6, 0xdde6ef, 0xf4f8fc, 0xffffff)
	for x := 0; x < c.w; x++ {
		for y := 0; y < c.h; y++ {
			if at(x, y) < 128 || at(x, y-1) >= 128 {
				continue
			}
			// How flat the top is here: the neighbours' tops at about this height.
			flat := 0
			for _, dx := range []int{-2, -1, 1, 2} {
				if at(x+dx, y) >= 128 && at(x+dx, y-1) < 128 {
					flat++
				}
			}
			n := int(math.Round(float64(deep) * float64(flat) / 4 * (0.6 + 0.6*hashf(x/2, 0, seed))))
			for k := 0; k < n && at(x, y+k) >= 128; k++ {
				i := (y+k)*c.w + x
				c.px[i] = withAlpha(snow.at(3-k*3/max(1, n)-int(hashf(x, y+k, seed+1)*1.4)), src[i].A)
			}
		}
	}
}

// --- grass ------------------------------------------------------------------------

// fanHabit is how the blades of a tuft fan out from its root: n blades
// hlo to hhi art pixels long, their feet spread art pixels apart, splayed
// up to theta radians from upright (plus comb, all to one side), arching
// over by droop, wlo to whi image pixels wide.
type fanHabit struct {
	n                  int
	hlo, hhi, spread   float64
	theta, comb, droop float64
	wlo, whi           float64
}

// coverFan draws a tuft's blades from its root at (bx, by), behind ones
// first and darker, in the colours of rp.
func coverFan(c *canvas, rng *rand.Rand, bx, by float64, hb fanHabit, rp ramp, lo, hi float64, snow bool, seed uint32) {
	const k = hdK3
	type blade struct{ x, h, lean, droop, w, tone float64 }
	blades := make([]blade, hb.n)
	for i := range blades {
		f := (float64(i) + 0.2 + rng.Float64()*0.6) / float64(hb.n)
		th := (f-0.5)*2*hb.theta + hb.comb + (rng.Float64()-0.5)*0.25
		h := (hb.hlo + rng.Float64()*(hb.hhi-hb.hlo)) * k
		h *= 1 - 0.3*math.Abs(th)/(hb.theta+math.Abs(hb.comb)+0.01)
		blades[i] = blade{
			x:     bx + ((f-0.5)*hb.spread+(rng.Float64()-0.5)*0.8)*k,
			h:     h * math.Sqrt(math.Cos(th)),
			lean:  h * math.Sin(th) * 1.2,
			droop: hb.droop * (0.35 + 1.3*math.Abs(math.Sin(th))) * (0.6 + 0.8*rng.Float64()),
			w:     hb.wlo + rng.Float64()*(hb.whi-hb.wlo),
			tone:  rng.Float64(),
		}
	}
	rng.Shuffle(len(blades), func(i, j int) { blades[i], blades[j] = blades[j], blades[i] })
	for i, b := range blades {
		depth := float64(i) / float64(len(blades)) // 0 at the back
		l0 := lo + 0.1*b.tone - 0.08*(1-depth)
		l1 := hi - 0.22*(1-depth) + 0.12*b.tone
		coverBlade(c, b.x, by, b.h, b.lean, b.droop, b.w, rp, l0, l1, snow, seed)
	}
}

// genGrassCover is a tuft of green grass, knee high at most: a fan of blades
// from one root, in six habits.
func genGrassCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(26, 20)
	bx, by := 13*k, 17*k
	hb := []fanHabit{
		{18, 7, 11, 3, 0.75, 0, 0.2, 2.2, 3.2},
		{24, 5, 8, 5, 1.0, 0, 0.3, 2.0, 3.0},
		{13, 9, 13, 2.5, 0.45, 0.3, 0.2, 2.2, 3.0},
		{14, 5, 8, 3, 0.8, 0, 0.2, 2.0, 3.0},
		{9, 6, 10, 2.5, 0.95, 0, 0.5, 3.2, 4.2},
		{20, 6, 10, 7, 0.7, -0.15, 0.25, 2.0, 3.0},
	}[v%6]
	coverFan(c, rng, bx, by, hb, p.grass, 0.08, 0.92, p.snow, 31)
	if v%6 == 3 { // thin stalks with seed heads, above the leaves
		for i := range 3 {
			x := bx + (float64(i)-1)*1.5*k + (rng.Float64()-0.5)*k
			h := (11 + rng.Float64()*3) * k
			lean := (float64(i) - 1 + (rng.Float64() - 0.5)) * 2.5 * k
			coverBlade(c, x, by, h, lean, 0.12, 1.2, p.grass, 0.25, 0.7, false, 32)
			x1, y1 := x+lean*0.25, by-h*(1.05+0.12*0.6)
			x2, y2 := x+lean, by-h*(1-0.12)
			for j := range 7 { // the head: grains along the last of the stalk
				t := 0.78 + float64(j)/6*0.22
				u := 1 - t
				gx, gy := u*u*x+2*u*t*x1+t*t*x2, u*u*by+2*u*t*y1+t*t*y2
				side := float64(j%2)*2 - 1
				c.ellipse(gx+side*0.3*k, gy, 0.32*k, 0.5*k, func(px, py int, nx, ny float64) color.RGBA {
					return rampAt(p.head, 0.7-ny*0.2-nx*0.15, px, py, 33)
				})
			}
		}
	}
	if p.snow {
		coverSnow(c, 2, 34)
	}
	return coverFinish(c, bx, by, 6.5*k, 2.2*k)
}

// genTallCover is tall grass, up to the thigh: long thin blades fanning out
// and arching over, with seed heads, in six habits.
func genTallCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(32, 26)
	bx, by := 16*k, 23*k
	type habit struct {
		fan      fanHabit
		heads    int
		headLook int // 0 a drooping ear of grains, 1 a feathery plume, 2 a foxtail
		green    float64
	}
	hb := []habit{
		{fanHabit{26, 13, 18, 4, 0.55, 0, 0.25, 1.5, 2.5}, 5, 0, 0.15},
		{fanHabit{20, 15, 19, 3, 0.38, 0, 0.12, 1.5, 2.4}, 4, 1, 0.1},
		{fanHabit{30, 10, 14, 6, 0.85, 0, 0.4, 1.5, 2.5}, 3, 0, 0.2},
		{fanHabit{24, 13, 17, 4, 0.42, 0.35, 0.3, 1.5, 2.4}, 4, 0, 0.1},
		{fanHabit{26, 11, 16, 5, 0.6, 0, 0.25, 1.5, 2.5}, 3, 2, 0.45},
		{fanHabit{18, 15, 20, 3, 0.32, -0.2, 0.12, 1.5, 2.4}, 5, 2, 0.05},
	}[v%6]
	// Green blades low at the foot, in every season but winter.
	if !p.snow {
		coverFan(c, rng, bx, by, fanHabit{int(12 * (0.4 + hb.green)), 4, 8, 5, 0.9, 0, 0.35, 2.2, 3.0}, p.grass, 0.05, 0.6, false, 41)
	}
	coverFan(c, rng, bx, by, hb.fan, p.tall, 0.06, 0.95, p.snow, 42)
	// Seed heads on stalks above the blades.
	for i := range hb.heads {
		f := (float64(i) + 0.5 + (rng.Float64()-0.5)*0.6) / float64(hb.heads)
		th := (f-0.5)*2*hb.fan.theta*0.8 + hb.fan.comb
		h := (hb.fan.hhi + rng.Float64()*3) * k
		lean := h * math.Sin(th) * 1.1
		droop := 0.06
		if hb.headLook == 0 {
			droop = 0.22 // the ear's weight bends the stalk
		}
		x := bx + (f-0.5)*hb.fan.spread*0.6*k
		coverBlade(c, x, by, h*math.Cos(th), lean, droop, 1.2, p.tall, 0.3, 0.85, false, 43)
		// The stalk's last stretch, for the head to hang along.
		hh := h * math.Cos(th)
		x1, y1 := x+lean*0.25, by-hh*(1.05+droop*0.6)
		x2, y2 := x+lean, by-hh*(1-droop)
		at := func(t float64) (float64, float64) {
			u := 1 - t
			return u*u*x + 2*u*t*x1 + t*t*x2, u*u*by + 2*u*t*y1 + t*t*y2
		}
		for j := range 12 {
			t := 0.76 + float64(j)/11*0.24
			gx, gy := at(t)
			switch hb.headLook {
			case 0: // an ear: grains either side
				side := float64(j%2)*2 - 1
				c.ellipse(gx+side*0.28*k, gy+0.15*k, 0.3*k, 0.5*k, func(px, py int, nx, ny float64) color.RGBA {
					return rampAt(p.head, 0.72-ny*0.25-nx*0.2, px, py, 44)
				})
			case 1: // a plume: fine hairs
				if j%2 == 0 {
					for q := range 3 {
						a := -math.Pi/2 + (float64(q)-1)*0.6 + lean/h
						c.line(gx, gy, gx+math.Cos(a)*1.2*k, gy+(math.Sin(a)*1.2+0.7)*k, func(px, py int, t float64) color.RGBA {
							return rampAt(p.head, 0.55+0.4*t, px, py, 45)
						})
					}
				}
			default: // a foxtail: a soft, bristly cylinder
				if j%2 == 0 {
					c.ellipse(gx, gy, 0.5*k, 0.55*k, func(px, py int, nx, ny float64) color.RGBA {
						return rampAt(p.head, 0.72-ny*0.3-nx*0.25+(hashf(px, py, 46)-0.5)*0.3, px, py, 46)
					})
				}
			}
		}
	}
	if p.snow { // snow on the tips and in a little drift at the foot
		coverSnow(c, 2, 47)
		for i := range 3 {
			x := bx + (float64(i)-1)*3*k
			c.ellipse(x, by-0.4*k, (2.2+rng.Float64())*k, 0.9*k, func(px, py int, nx, ny float64) color.RGBA {
				return rampAt(mkRamp(0xc4cfdc, 0xe4ebf3, 0xfbfdff), 0.8-ny*0.5, px, py, 48)
			})
		}
	}
	return coverFinish(c, bx, by, 8*k, 2.6*k)
}

// genStubbleCover is tall grass cut down: stubble, and a few cut blades
// lying on the ground.
func genStubbleCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(26, 14)
	bx, by := 13*k, 11*k
	for range 2 + v%2 { // cut blades lying about
		x := bx + (rng.Float64()-0.5)*12*k
		y := by - rng.Float64()*1.5*k
		l := (5 + rng.Float64()*5) * k
		a := (rng.Float64() - 0.5) * 0.7
		c.line(x, y, x+math.Cos(a)*l, y+math.Sin(a)*l*0.4, func(px, py int, t float64) color.RGBA {
			return rampAt(p.tall, 0.45+0.3*t, px, py, 51)
		})
	}
	n := 14 + v*3
	for i := range n {
		f := (float64(i) + rng.Float64()) / float64(n)
		x := bx + (f-0.5)*(8+float64(v))*k
		h := (1.5 + rng.Float64()*2.2) * k
		coverBlade(c, x, by, h, (f-0.5)*2*k, 0, 2.2, p.tall, 0.15, 0.55, false, 52)
		c.put(int(x+(f-0.5)*2*k), int(by-h), rampAt(p.tall, 0.95, 0, 0, 0)) // the cut end, pale
	}
	for range 5 { // green shoots coming back
		x := bx + (rng.Float64()-0.5)*9*k
		coverBlade(c, x, by, (2+rng.Float64()*2.5)*k, (rng.Float64()-0.5)*2*k, 0.1, 2, p.grass, 0.2, 0.7, false, 53)
	}
	if p.snow {
		coverSnow(c, 2, 54)
	}
	return coverFinish(c, bx, by, 7*k, 1.6*k)
}

// --- leaves -----------------------------------------------------------------------

// genClumpCover is a low, round clump of small leaves, in five shapes.
func genClumpCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(24, 15)
	bx, by := 12*k, 12.5*k
	shapes := [][]blob{
		{{8, 9, 3.6}, {13, 8, 4.2}, {16.5, 9.5, 3.2}, {11, 10.5, 3.4}},
		{{6.5, 10, 3}, {10.5, 8.5, 3.8}, {14.5, 8.5, 3.8}, {18, 10, 3}, {12.5, 10.5, 3.6}},
		{{10, 8, 4.4}, {14, 8.5, 4}, {12, 10.5, 3.8}},
		{{8, 9.5, 3.4}, {11.5, 7.5, 3.4}, {12, 10.5, 3}, {16, 9.5, 3.6}, {14, 6.5, 2.6}},
		{{9, 9, 3.4}, {14.5, 9.5, 3.4}, {12, 7.5, 3.2}, {12, 10.5, 3.2}},
	}
	bs := shapes[v%len(shapes)]
	jit := make([]blob, len(bs))
	minX, maxX := math.Inf(1), math.Inf(-1)
	for i, b := range bs {
		jit[i] = blob{(b.x + (rng.Float64()-0.5)*0.8) * k, (b.y + (rng.Float64()-0.5)*0.6) * k, (b.r + (rng.Float64()-0.5)*0.5) * k}
		minX, maxX = math.Min(minX, jit[i].x-jit[i].r), math.Max(maxX, jit[i].x+jit[i].r)
	}
	look := leafSmall
	if v%2 == 1 {
		look = leafLook{2.9, 2.6, 1.3} // finer leaves
	}
	leafClusterHD(c, jit, bx, 8.5*k, (maxX-minX)/2, 3.6*k, p.leaf, 900+uint32(v)*7, look)
	switch {
	case v == 4 && p.bloom > 0: // tiny white flowers on it
		for i := range 6 {
			x, y := minX+(maxX-minX)*(0.15+0.7*rng.Float64()), (6.5+rng.Float64()*3)*k
			bloomHD(c, x, y, 1.9, mkRamp(0xd8d6c8, 0xf2f0e6, 0xffffff), hex(0xf2cf4a), float64(i))
		}
	case v == 2 && !p.snow: // new leaves, pale, on top
		for i := range 5 {
			x, y := minX+(maxX-minX)*(0.25+0.5*rng.Float64()), (5.5+rng.Float64()*2)*k
			leafHD(c, x, y, 3.2, 1.7, -1.2-rng.Float64()*0.8+float64(i%2)*1.8, p.leaf, 0.95)
		}
	}
	if p.snow {
		coverSnow(c, 4, 61)
	}
	return coverFinish(c, bx, by, (maxX-minX)/2+1.5*k, 2.4*k)
}

// genFernCover is a fern: fronds fanning out from the root and arching
// over, the upright ones behind and darker, in five habits.
func genFernCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(34, 24)
	bx, by := 17*k, 20*k
	type habit struct {
		n            int
		llo, lhi     float64 // frond length
		spread, lift float64 // angle spread; how upright
		droop        float64
		curls        int // fiddleheads
	}
	hb := []habit{
		{7, 10, 13, 1.1, 1, 0.3, 0},
		{9, 12, 15, 1.25, 0.9, 0.32, 0},
		{4, 7, 10, 1.0, 1.1, 0.2, 2},
		{7, 10, 14, 0.8, 1.35, 0.18, 0},
		{8, 9, 12, 1.35, 0.75, 0.38, 0},
	}[v%5]
	type frond struct{ a, l float64 }
	fronds := make([]frond, hb.n)
	for i := range fronds {
		f := (float64(i) + 0.5 + (rng.Float64()-0.5)*0.5) / float64(hb.n)
		a := math.Pi/2 + (f-0.5)*math.Pi*hb.spread
		a = math.Pi/2 + (a-math.Pi/2)/hb.lift
		fronds[i] = frond{a, (hb.llo + rng.Float64()*(hb.lhi-hb.llo)) * k}
	}
	for pass := range 2 {
		for i, fr := range fronds {
			upright := math.Abs(math.Cos(fr.a)) < 0.45
			if upright != (pass == 0) {
				continue
			}
			dark := 0.0
			if upright {
				dark = 0.16
			}
			at := func(t float64) (float64, float64) {
				return bx + math.Cos(fr.a)*fr.l*t, by - math.Sin(fr.a)*fr.l*t + fr.l*hb.droop*t*t
			}
			frondAlongHD(c, at, int(fr.l*2.5), func(t float64) float64 { return (0.8 + 2.5*(1-t)) * k * math.Min(1, t*5) }, 2.4, 0.12, p.fern, dark, 700+uint32(i)+uint32(v)*13)
		}
	}
	for i := range hb.curls { // fiddleheads: young fronds still curled
		x := bx + (float64(i)-0.5)*3*k
		h := (6 + float64(i)*2) * k
		coverBlade(c, x, by, h, (float64(i)-0.5)*1.5*k, 0, 1.6, p.fern, 0.3, 0.6, false, 71)
		cx, cy := x+(float64(i)-0.5)*1.5*k, by-h
		for q := 0.0; q < 2*math.Pi*1.4; q += 0.15 {
			r := (1.6 - q/(2*math.Pi*1.4)*1.1) * k
			px, py := cx+math.Cos(q)*r, cy+math.Sin(q)*r*0.9
			c.put(int(px), int(py), rampAt(p.fern, 0.55+0.25*math.Cos(q+2.3), int(px), int(py), 72))
			c.put(int(px)+1, int(py), rampAt(p.fern, 0.4, int(px), int(py), 72))
		}
	}
	if p.snow {
		coverSnow(c, 2, 73)
	}
	return coverFinish(c, bx, by, 9*k, 2.4*k)
}

// genBroadCover is a few big pointed leaves on short stalks.
func genBroadCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(28, 21)
	bx, by := 14*k, 18*k
	n := 3 + v%3
	for i := range n {
		f := (float64(i) + 0.5) / float64(n)
		a := math.Pi * (0.88 - 0.76*f + (rng.Float64()-0.5)*0.18)
		stalk := (1.5 + rng.Float64()*3) * k
		L, W := (7+rng.Float64()*3.5+float64(v%2)*1.5)*k, (3+rng.Float64()*1.4)*k
		if v == 3 {
			W *= 1.35 // round, hosta-like
		}
		ux, uy := math.Cos(a), -math.Sin(a)*0.78
		vx, vy := -uy, ux
		x0, y0 := bx+ux*stalk, by+uy*stalk
		for _, o := range []float64{0, 1} {
			c.line(bx+o, by, x0+o, y0, func(x, y int, _ float64) color.RGBA { return p.broad[1+int(o)] })
		}
		var pts []float64
		const steps = 18
		wAt := func(t float64) float64 {
			return W * math.Pow(math.Sin(math.Pi*math.Min(1, t*1.15)), 0.7) * (1 - 0.15*t)
		}
		for s := 0; s <= steps; s++ {
			t := float64(s) / steps
			pts = append(pts, x0+ux*L*t+vx*wAt(t), y0+uy*L*t+vy*wAt(t))
		}
		for s := steps; s >= 0; s-- {
			t := float64(s) / steps
			pts = append(pts, x0+ux*L*t-vx*wAt(t), y0+uy*L*t-vy*wAt(t))
		}
		c.polygon(pts, func(x, y int) color.RGBA {
			px, py := float64(x)+0.5-x0, float64(y)+0.5-y0
			along, across := (px*ux+py*uy)/L, (px*vx+py*vy)/W
			l := 0.55 + 0.25*across*vy*-1 + 0.2*along - 0.25*math.Abs(across)
			if vv := math.Mod(along*7-math.Abs(across)*1.6+8, 1); vv < 0.14 {
				l -= 0.12
			} else if vv < 0.3 {
				l += 0.05
			}
			if math.Abs(px*vx+py*vy) > wAt(clamp01(along))-1.2 {
				l -= 0.12
			}
			return rampAt(p.broad, l, x, y, 81)
		})
		c.line(x0, y0, x0+ux*L*0.9, y0+uy*L*0.9, func(x, y int, t float64) color.RGBA {
			return rampAt(p.broad, 0.85-0.25*t, x, y, 82)
		})
	}
	if p.snow {
		coverSnow(c, 3, 83)
	}
	return coverFinish(c, bx, by, 8*k, 2.2*k)
}

// --- flowers ----------------------------------------------------------------------

// flowerLooks are the wild flowers' colours, and how each is drawn.
var flowerLooks = []struct {
	petals ramp
	eye    color.RGBA
	look   int // 0 a daisy, 1 a cup, 2 hanging bells, 3 a round head, 4 ragged, 5 a flat umbel
}{
	{mkRamp(0xc9c3b2, 0xe9e5d8, 0xfbfaf4), hex(0xf0c43a), 0},
	{mkRamp(0xc9941c, 0xeac23a, 0xfbe27a), hex(0xd9892b), 1},
	{mkRamp(0x6a4a9c, 0x9370c6, 0xc1a2e6), hex(0xe8d8f4), 2},
	{mkRamp(0xa8487a, 0xd27aa4, 0xf0b0cc), hex(0xf6d6e4), 3},
	{mkRamp(0x3a54b0, 0x5f7fd8, 0x9ab4f4), hex(0x2a2a5a), 4},
	{mkRamp(0xc2c0b0, 0xe6e3d6, 0xfffef6), hex(0xe8e0b8), 5},
}

// genFlowerCover is a little patch of wild flowers of one kind over a
// rosette of leaves. In spring they are all open, in summer some, in
// autumn they have gone to seed, and in winter only dry stalks stand.
func genFlowerCover(v int, rng *rand.Rand, p coverPal) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(20, 17)
	bx, by := 10*k, 14*k
	fl := flowerLooks[v%len(flowerLooks)]
	// The leaves at the foot.
	if !p.snow {
		for i := range 5 {
			a := math.Pi*(0.1+0.8*float64(i)/4) + (rng.Float64()-0.5)*0.3
			lx, ly := bx+math.Cos(a)*3*k, by-math.Sin(a)*1.2*k-0.6*k
			leafHD(c, lx, ly, 3.4, 1.6, -a+math.Pi/2*0+math.Atan2(-math.Sin(a)*0.5, math.Cos(a)), p.leaf, 0.6)
		}
	}
	n := 4 + rng.IntN(3)
	if fl.look == 3 {
		n = 3 // clover: few heads, many leaves
		for range 7 {
			x, y := bx+(rng.Float64()-0.5)*9*k, by-(rng.Float64()*2.5+0.5)*k
			for q := range 3 { // a trefoil
				a := float64(q)*2*math.Pi/3 - math.Pi/2
				c.ellipse(x+math.Cos(a)*0.55*k, y+math.Sin(a)*0.4*k, 0.62*k, 0.5*k, func(px, py int, nx, ny float64) color.RGBA {
					return rampAt(p.leaf, 0.55+lum(nx, ny)*0.35, px, py, 91)
				})
			}
		}
	}
	for i := range n {
		x := bx + (-4+rng.Float64()*8)*k
		h := (4 + rng.Float64()*5) * k
		lean := (rng.Float64() - 0.5) * 2.5 * k
		stem := p.grass
		if p.bloom < -1.5 {
			stem = p.tall // dry stalks
		}
		coverBlade(c, x, by, h, lean, 0.04, 1.4, stem, 0.15, 0.55, false, 92)
		if fl.look != 3 {
			leafHD(c, x+lean*0.2+2.5, by-h*0.35, 2.6, 1.2, -0.6, p.leaf, 0.6)
		}
		hx, hy := x+lean, by-h
		open := p.bloom >= 0 && hashf(i, v, 93) < max(0.25, p.bloom)
		switch {
		case p.bloom < -1.5: // winter: a dry seed case
			c.ellipse(hx, hy, 0.6*k, 0.8*k, func(px, py int, nx, ny float64) color.RGBA {
				return rampAt(p.head, 0.5-ny*0.3, px, py, 94)
			})
		case p.bloom < 0: // autumn: gone to seed
			if v%2 == 0 { // a clock of fluff
				c.ellipse(hx, hy, 1.5*k, 1.5*k, func(px, py int, nx, ny float64) color.RGBA {
					if hashf(px, py, 95) < 0.35 {
						return color.RGBA{}
					}
					return withAlpha(mix(hex(0xe8e4d8), hex(0xb8b2a0), ny*0.5+0.5), 220)
				})
			} else {
				c.ellipse(hx, hy, 0.8*k, 1*k, func(px, py int, nx, ny float64) color.RGBA {
					return rampAt(p.head, 0.55-ny*0.3-nx*0.2, px, py, 96)
				})
			}
		case !open: // a bud
			c.ellipse(hx, hy, 0.55*k, 0.85*k, func(px, py int, nx, ny float64) color.RGBA {
				if ny < -0.2 {
					return fl.petals.at(1)
				}
				return rampAt(p.leaf, 0.6, px, py, 97)
			})
		default:
			s := 0.85 + 0.3*p.bloom // spring's blooms are the biggest
			switch fl.look {
			case 0, 4: // a daisy; a ragged cornflower
				bloomHD(c, hx, hy, 4.4*s, fl.petals, fl.eye, rng.Float64())
				if fl.look == 4 {
					for q := range 8 {
						a := float64(q) * math.Pi / 4
						c.put(int(hx+math.Cos(a)*5*s), int(hy+math.Sin(a)*4*s), fl.petals.at(2))
					}
				}
			case 1: // a glossy cup
				c.ellipse(hx, hy+0.2*k, 1.4*s*k, 1.1*s*k, func(px, py int, nx, ny float64) color.RGBA {
					l := 0.55 + lum(nx, ny)*0.4
					if ny < -0.35 {
						l -= 0.35 // the inside of the cup
					}
					return fl.petals.pick(l, px, py)
				})
				c.put(int(hx-1.2*s), int(hy), fl.petals.at(2))
			case 2: // bells hanging from an arching stalk
				for q := range 3 {
					bxx, byy := hx+float64(q)*1.6*k*math.Copysign(1, lean+0.01), hy+float64(q)*1.1*k+1.2*k
					c.ellipse(bxx, byy, 0.8*s*k, 1*s*k, func(px, py int, nx, ny float64) color.RGBA {
						return fl.petals.pick(0.5+lum(nx, ny)*0.45, px, py)
					})
				}
			case 3: // a round clover head
				c.ellipse(hx, hy, 1.4*s*k, 1.3*s*k, func(px, py int, nx, ny float64) color.RGBA {
					l := 0.45 + lum(nx, ny)*0.45
					if hashf(px/2, py/2, 98) < 0.3 {
						l -= 0.15
					}
					return fl.petals.pick(l, px, py)
				})
			default: // a flat umbel of tiny blooms
				for q := range 9 {
					a := float64(q) * 2.4
					r := math.Sqrt(float64(q)/9) * 1.8 * s
					c.ellipse(hx+math.Cos(a)*r*k, hy+math.Sin(a)*r*0.45*k, 0.5*k, 0.42*k, func(px, py int, nx, ny float64) color.RGBA {
						return fl.petals.pick(0.55+lum(nx, ny)*0.4, px, py)
					})
				}
			}
		}
	}
	if p.snow {
		coverSnow(c, 2, 99)
	}
	return coverFinish(c, bx, by, 6*k, 2*k)
}

// --- bushes -----------------------------------------------------------------------

// genBushCover is a berry bush, with berries or picked, in four shapes; in
// spring white blossom, in autumn its leaves turn, in winter snow lies on it.
func genBushCover(v int, rng *rand.Rand, p coverPal, berries bool) (*canvas, float64, float64) {
	const k = hdK3
	c := hdCanvas(30, 24)
	bx, by := 15*k, 20*k
	shapes := [][]blob{
		{{8, 14, 6}, {20, 14, 6}, {14, 11, 8}, {10, 8, 5}, {18, 8, 5}, {14, 15, 6}},
		{{9, 13, 6.5}, {16, 10, 7.5}, {21, 14, 5.5}, {13, 15, 6}},
		{{7, 15, 5}, {12, 11, 6}, {17, 9, 6.5}, {22, 13, 5.5}, {15, 15, 6}},
		{{10, 12, 6}, {18, 12, 6}, {14, 8, 5.5}, {14, 15, 6.5}},
	}
	bs := shapes[v%len(shapes)]
	blobs := make([]blob, len(bs))
	minX, maxX := math.Inf(1), math.Inf(-1)
	for i, b := range bs {
		blobs[i] = blob{(b.x + 1 + (rng.Float64()-0.5)*0.8) * k, (b.y + 1) * k, (b.r + (rng.Float64()-0.5)*0.6) * k}
		minX, maxX = math.Min(minX, blobs[i].x-blobs[i].r), math.Max(maxX, blobs[i].x+blobs[i].r)
	}
	rp := p.bush
	if !berries && !p.snow {
		rp = mkRamp(0x2c3e28, 0x3e5431, 0x546c3b, 0x6c8546, 0x869c55, 0xa2b268) // picked: duller
	}
	leafClusterHD(c, blobs, bx, 12*k, (maxX-minX)/2, 9*k, rp, 160+uint32(v)*11, leafSmall)
	if p.blossom {
		for i := range 7 {
			x, y := minX+(maxX-minX)*(0.12+0.76*rng.Float64()), (6+rng.Float64()*8)*k
			bloomHD(c, x, y, 2.2, mkRamp(0xe0d8d8, 0xf6eef0, 0xffffff), hex(0xf0c848), float64(i))
		}
	}
	if berries {
		for i := range 9 + v {
			x, y := minX+(maxX-minX)*(0.1+0.8*rng.Float64()), (6+rng.Float64()*10)*k
			berryHD(c, x, y, 3.0, berryRamp, rp[0])
			if i%3 == 0 {
				berryHD(c, x+4, y+2.5, 2.3, berryRamp, rp[0])
			}
		}
	} else {
		for range 5 { // the bare stalks the berries hung on
			x, y := int(minX+(maxX-minX)*(0.15+0.7*rng.Float64())), int((7+rng.Float64()*8)*k)
			c.put(x, y, trunkRamp[2])
			c.put(x, y+1, trunkRamp[1])
			c.put(x+1, y+1, trunkRamp[2])
		}
	}
	if p.snow {
		coverSnow(c, 5, 101)
	}
	return coverFinish(c, bx, by, (maxX-minX)/2+1*k, 3*k)
}
