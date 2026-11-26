package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"survivalgame/internal/sim"
)

// The plants of the water and its banks at HD: water lilies and lotus on
// the ponds, seaweed and mangroves in the shallows of the sea, horsetail
// where the ground is wet. What floats is seen from above, a little
// foreshortened like the ground itself; its anchor is the middle of the
// leaf a toad would sit on. See floraart.go for the tools.

var (
	padRamp        = mkRamp(0x183e24, 0x24582a, 0x367634, 0x4e943e, 0x70b24e, 0x9cd06c)
	lilyPetalRamp  = mkRamp(0xa84474, 0xd0709c, 0xeaa0c0, 0xf8cce0, 0xfff0f6, 0xffffff)
	lotusPetalRamp = mkRamp(0xb03c78, 0xd85c98, 0xf088b4, 0xfab4d0, 0xffdcea, 0xfff4f8)
	podRamp        = mkRamp(0x3c4a1c, 0x5c6e28, 0x829438, 0xa8b850, 0xccd878)
	kelpRamp       = mkRamp(0x262e12, 0x3e4a1a, 0x5e6a26, 0x848c38, 0xaeb054, 0xd2d07c)
	mangroveBark   = mkRamp(0x2a221e, 0x443830, 0x625244, 0x82705a, 0xa29076, 0xbeae94)
	mangroveLeaf   = mkRamp(0x143626, 0x1e502e, 0x2c6c36, 0x448a42, 0x66a852, 0x94c66e)
	horsetailRamp  = mkRamp(0x1c4628, 0x2c6634, 0x448c42, 0x66b056, 0x92d074, 0xbce89c)
	waterShade     = color.RGBA{8, 30, 48, 110}     // the shadow a floating leaf casts into the water
	rippleCol      = color.RGBA{214, 240, 246, 120} // a ring of ripples; too thin to be outlined
)

// padYK is how much what lies flat on the water is foreshortened.
const padYK = 0.62

// pad is a floating leaf at art point (cx, cy), rx wide, with its cleft at
// angle notch: veins fan out from the middle, the rim curls up a little.
func (f *flo) pad(cx, cy, rx, notch float64, rp ramp, bias float64, seed uint32) {
	c := f.c
	ry := rx * padYK
	c.ellipse((cx+0.5)*fk, (cy+0.8)*fk, rx*fk, ry*fk, func(x, y int, nx, ny float64) color.RGBA {
		if c.at(x, y).A > 0 {
			return color.RGBA{}
		}
		return waterShade
	})
	n := len(rp)
	c.ellipse(cx*fk, cy*fk, rx*fk, ry*fk, func(x, y int, nx, ny float64) color.RGBA {
		d := math.Hypot(nx, ny)
		a := math.Atan2(ny, nx)
		if da := math.Abs(math.Remainder(a-notch, 2*math.Pi)); da < 0.2*(0.35+d) && d > 0.1 {
			return color.RGBA{} // the cleft
		}
		l := 0.6 + bias - (nx+ny)*0.17
		switch {
		case d > 0.9:
			l += 0.1 - (nx+ny)*0.2 // the rim, turned up: lit towards the sun, dark away from it
		case d > 0.82:
			l -= 0.12
		}
		if v := math.Abs(math.Remainder(a*11/(2*math.Pi)+float64(seed%7)*0.13, 1)); v < 0.11 && d > 0.16 && d < 0.84 {
			l += 0.16 // a vein
		}
		if d < 0.13 {
			l += 0.2 // where the stalk joins underneath
		}
		l += (hashf(x, y, seed) - 0.5) * 0.05
		return rp.at(int(clamp01(l) * float64(n)))
	})
	for i := range 2 { // drops of water sitting on it
		a, d := hashf(i, 0, seed+3)*2*math.Pi, 0.3+0.4*hashf(i, 1, seed+3)
		if math.Abs(math.Remainder(a-notch, 2*math.Pi)) < 0.6 {
			continue
		}
		x, y := cx+math.Cos(a)*d*rx, cy+math.Sin(a)*d*ry
		f.q.dot(x, y, 0, 0, hex(0xdaf2f6))
		f.q.dot(x, y, 1, 0, hex(0x9cd0de))
	}
}

// bloom is a water flower at art point (x, y), seen a little from above:
// rings of pointed petals round a golden heart. open is how far the outer
// petals lie back (1: flat on the water, like a water lily; less: a cup,
// like a lotus), s its size.
func (f *flo) bloom(x, y, s, open float64, rp ramp, rings int) {
	for ring := 0; ring < rings; ring++ {
		k := float64(ring) / float64(max(1, rings-1)) // 0 the outer ring, 1 the innermost
		n := 9 - 2*ring
		L := s * (1 - 0.3*k)
		// Back petals first, then round to the front ones, which lie over the heart.
		order := make([]float64, 0, n)
		for i := 0; i < n; i++ {
			order = append(order, 2*math.Pi*(float64(i)+0.5*float64(ring%2))/float64(n))
		}
		for pass := 0; pass < 2; pass++ {
			for _, a := range order {
				up := math.Sin(a)
				if (up > 0) != (pass == 0) {
					continue
				}
				// Petals pointing up or down the screen are seen end on: shorter.
				spread := lerpf(0.55, 1, open) * (1 - 0.5*k)
				ln := L * (0.62 + 0.38*math.Abs(math.Cos(a))*spread + 0.3*(1-open)*math.Max(0, up))
				ang := math.Atan2(math.Sin(a)*lerpf(1.25, padYK, open*(1-k)), math.Cos(a)*spread)
				f.leafAt(x+math.Cos(a)*0.4*s*(1-k), y-math.Sin(a)*0.22*s*(1-k), ang, ln, s*0.3, 0, leafLance, rp, 0, 0.1+0.2*k-0.1*float64(pass))
			}
		}
	}
	heart := aEll(x, y-0.1*s, 0.3*s, 0.24*s)
	f.q.flat(heart, plain(hex(0xf0b82c)))
	f.q.flat(aEll(x-0.05*s, y-0.14*s, 0.17*s, 0.12*s), plain(hex(0xfce46c)))
}

// --- the water lily --------------------------------------------------------------------

func genLilyHD(full bool) (*canvas, float64, float64) {
	f := newFlo(34, 22)
	const bx, by = 17.0, 12.0
	f.pad(bx+10.2, by-3.6, 4.6, 0.6, padRamp, -0.08, 831)
	f.pad(bx-9.6, by+2.6, 4.2, 2.5, padRamp, -0.04, 832)
	f.pad(bx+0.4, by, 7.2, -0.9, padRamp, 0.02, 833) // the big one: a toad's seat
	f.pad(bx+7.8, by+4.6, 3.3, 4.2, padRamp, 0.06, 834)
	f.pad(bx-10.5, by-4.2, 2.6, 1.2, padRamp, 0, 835)
	if full {
		f.bloom(bx-6.6, by-2.6, 4.6, 0.9, lilyPetalRamp, 3)
	} else { // a bud, still shut, waiting for its day
		bud := aCap(bx-6.6, by-1.6, bx-6.2, by-4.2, 1.1, 0.35)
		f.q.fill(bud, func(x, y int, _ float64) color.RGBA {
			if float64(y)/fk < by-3 {
				return toneOf(lilyPetalRamp, bud.lum(x, y, 1.2*fk)*0.8+0.1)
			}
			return toneOf(padRamp, bud.lum(x, y, 1.2*fk)*0.8+0.15)
		})
	}
	return f.done(0.55, bx, by)
}

// genLilyRaftHD is a small raft of lily pads adrift on a swamp pond: only
// to look at (see swampart.go). Variant 1 has no flower.
func genLilyRaftHD(rng *rand.Rand, v int) (*canvas, float64, float64) {
	f := newFlo(24, 14)
	n := 2 + v
	var fx, fy float64
	for i := 0; i < n; i++ {
		x, y, r := 6+rng.Float64()*12, 5+rng.Float64()*5, 2.4+rng.Float64()*1.3
		f.pad(x, y, r, rng.Float64()*2*math.Pi, padRamp, (rng.Float64()-0.5)*0.12, 870+uint32(v*7+i))
		if i == 0 {
			fx, fy = x-0.6, y-0.8
		}
	}
	if v != 1 {
		f.bloom(fx, fy, 2.6, 0.9, lilyPetalRamp, 2)
	}
	return f.done(0.5, 12, 8)
}

// --- the lotus ---------------------------------------------------------------------------

// lotusLeaf is a lotus leaf held above the water at art point (cx, cy): a
// shallow dish with a wavy rim, on its stalk from (sx, sy).
func (f *flo) lotusLeaf(sx, sy, cx, cy, rx, tilt float64, seed uint32) {
	f.stalk(padRamp, 0.5, 0.42, -0.1, [2]float64{sx, sy}, [2]float64{sx, lerpf(sy, cy, 0.5)}, [2]float64{cx, lerpf(sy, cy, 0.4)}, [2]float64{cx, cy + 0.4}, nil)
	c := f.c
	ry := rx * 0.5
	n := len(padRamp)
	cs, sn := math.Cos(tilt), math.Sin(tilt)
	c.ellipse(cx*fk, cy*fk, (rx+1)*fk, (rx+1)*fk, func(x, y int, _, _ float64) color.RGBA {
		px, py := float64(x)+0.5-cx*fk, float64(y)+0.5-cy*fk
		nx, ny := (px*cs+py*sn)/(rx*fk), (-px*sn+py*cs)/(ry*fk)
		d, a := math.Hypot(nx, ny), math.Atan2(ny, nx)
		if d > 1+0.07*math.Sin(a*7+float64(seed)) {
			return color.RGBA{} // the rim waves
		}
		// A dish: the far side of it faces the sun, the near side is in its
		// own shade, and the middle dips to where the stalk holds it.
		l := 0.56 - ny*0.24 - nx*0.1 + 0.1*d
		if v := math.Abs(math.Remainder(a*13/(2*math.Pi), 1)); v < 0.1 && d > 0.14 {
			l += 0.15
		}
		switch {
		case d < 0.14:
			l -= 0.2
		case d > 0.88:
			l += 0.12 - ny*0.14
		}
		return padRamp.at(int(clamp01(l+(hashf(x, y, seed)-0.5)*0.05) * float64(n)))
	})
}

func genLotusHD(full bool) (*canvas, float64, float64) {
	f := newFlo(38, 40)
	const bx, by = 19.0, 31.0
	f.pad(bx-10.5, by-1.5, 4.4, 2.2, padRamp, -0.06, 841)
	f.pad(bx+10, by+2, 4, 0.4, padRamp, -0.02, 842)
	f.pad(bx, by, 7.4, -1.2, padRamp, 0.03, 843) // a toad's seat
	f.lotusLeaf(bx+7.5, by-2.5, bx+10.5, by-12, 6.6, -0.14, 845)
	f.lotusLeaf(bx-6, by-3.5, bx-10, by-17.5, 6, 0.2, 846)
	// The seed head on its stalk: a flat-topped cone, its seeds showing in
	// their pits. Picked, only the cut stalk stands.
	px, py := bx+4.5, by-19.0
	f.stalk(podRamp, 0.45, 0.4, -0.05, [2]float64{bx + 3, by - 3}, [2]float64{bx + 2.4, by - 10}, [2]float64{px, py + 7}, [2]float64{px, py + 2}, nil)
	if full {
		cone := aPoly(px-2.9, py-0.6, px+2.9, py-0.6, px+1, py+3.2, px-1, py+3.2)
		f.lump(cone.grow(0.6), podRamp, 1.6, -0.06)
		top := aEll(px, py-0.6, 3.1, 1.25)
		f.q.fill(top, shd(podRamp, top, 1, 0.2))
		for i, p := range [][2]float64{{-1.8, 0}, {-0.6, -0.5}, {0.7, -0.45}, {1.9, 0.05}, {-0.9, 0.42}, {0.5, 0.5}} {
			f.q.flat(aEll(px+p[0], py-0.6+p[1], 0.42, 0.3), plain(podRamp[0]))
			if i%2 == 0 {
				f.q.dot(px+p[0], py-0.6+p[1], 0, 0, hex(0xe8e2b0)) // a seed
			}
		}
	} else {
		f.q.flat(aEll(px, py+2, 0.5, 0.3), plain(podRamp[4]))
	}
	// The flower, tall on its own stalk.
	fx, fy := bx-1.5, by-22.0
	f.stalk(padRamp, 0.5, 0.42, 0, [2]float64{bx - 1, by - 3}, [2]float64{bx - 0.4, by - 10}, [2]float64{fx, fy + 9}, [2]float64{fx, fy + 2}, nil)
	f.bloom(fx, fy, 6, 0.35, lotusPetalRamp, 3)
	return f.done(0.6, bx, by)
}

// --- seaweed -------------------------------------------------------------------------------

// kelp draws one ribbon of weed from art point (x0, y0), L long along ang,
// waving as the water moves it; under marks it as lying below the surface:
// dim and half seen.
func (f *flo) kelp(x0, y0, ang, L, r float64, under bool, seed uint32) *ctube {
	var pts []tpt
	ux, uy := math.Cos(ang), math.Sin(ang)*padYK
	ph := float64(seed%11) * 0.6
	for i := 0; i <= 24; i++ {
		t := float64(i) / 24
		w := math.Sin(t*7+ph) * 2.3 * (0.25 + 0.75*t)
		pts = append(pts, tpt{x0 + ux*L*t - uy*w, y0 + uy*L*t + ux*w*padYK, r * (0.45 + 0.55*math.Sin(math.Pi*math.Min(1, 0.12+t*0.92)))})
	}
	tb := newCTube(pts)
	f.q.draw(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
		tone := 0.34 + 0.3*l - 0.16*v
		switch {
		case math.Abs(v) < 0.26:
			tone += 0.2 // the midrib
		case math.Abs(v) > 0.62 && math.Sin(s*L*2.6+ph) > 0:
			tone -= 0.2 // the frilled edge, in and out
		}
		col := toneOf(kelpRamp, tone)
		if under {
			col = mix(col, hex(0x1c4a5c), 0.6)
			col.A = 110
		}
		return col
	}), fillFlat)
	return tb
}

func genSeaweedHD(full bool) (*canvas, float64, float64) {
	f := newFlo(40, 24)
	rng := rand.New(rand.NewPCG(85, 23))
	const bx, by = 20.0, 12.0
	// The weed streams one way with the current, from where it holds on to
	// the bottom. What lies under the surface is always there.
	x0, y0 := bx-11.0, by+1.0
	for i := 0; i < 7; i++ {
		a := -1.0 + float64(i)*0.33 + rng.Float64()*0.1
		f.kelp(x0+rng.Float64()*3, y0+(rng.Float64()-0.5)*4, a, 20+rng.Float64()*8, 1.5, true, 850+uint32(i))
	}
	if full { // and the blades that reach the top: long, frilled ribbons, with their floats
		kelpForm := leafForm(func(t float64) float64 {
			return math.Min(1, t*5) * math.Pow(1-t, 0.3) * (0.78 + 0.22*math.Sin(t*34))
		})
		for i, b := range [][4]float64{{0.62, 15, 0.3, 0}, {-0.5, 17, 0.02, -0.1}, {0.3, 21, 0.16, 0.04}, {-0.2, 14, -0.02, -0.04}, {0.05, 19, 0.1, 0.08}, {0.95, 10, 0.2, -0.08}} {
			bx0, by0 := x0+2+rng.Float64()*2.5, y0+(rng.Float64()-0.5)*3
			f.leafAt(bx0, by0, b[0]*padYK, b[1], 1.5, b[2], kelpForm, kelpRamp, 0, b[3])
			for k := 0; k < 2; k++ { // air bladders along the midrib, shining
				t := 0.3 + 0.3*float64(k) + 0.1*float64(i%2)
				px := bx0 + math.Cos(b[0]*padYK)*b[1]*t
				py := by0 - math.Sin(b[0]*padYK)*b[1]*t + b[2]*b[1]*t*t
				bl := aEll(px, py, 0.8, 0.66)
				f.q.flat(bl, shd(kelpRamp, bl, 0.9, 0.3))
				f.q.dot(px-0.2, py-0.3, 0, 0, colWhite)
			}
		}
	}
	return f.done(0.5, bx, by)
}

// --- the mangrove ----------------------------------------------------------------------------

// mangroveS is how much bigger the mangrove stands than it was first drawn:
// on its stilts, its crown three times the player's height. The leaves and
// the rings of the bark keep their size, so the bigger tree has more of them.
const mangroveS = 1.65

func genMangroveHD(full bool) (*canvas, float64, float64) {
	const S = mangroveS
	const T = 1.45 // how much thicker the wood is drawn
	f := newFlo(int(math.Ceil(52*S)), int(math.Ceil(55*S))+7)
	rng := rand.New(rand.NewPCG(86, 29))
	bx, base := math.Round(26*S), math.Ceil(55*S)+1
	// P is a point of the first drawing, dx from the trunk and h above the
	// ground, where it stands now.
	P := func(dx, h float64) [2]float64 { return [2]float64{bx + dx*S, base - h*S} }
	bark := func(s, v, l float64, col color.RGBA) color.RGBA {
		if u := s*9*S + v*0.4; u-math.Floor(u) < 0.14 {
			return toneOf(mangroveBark, l*0.86-0.1) // rings on the bark
		}
		return col
	}
	// The stilt roots: arches from the trunk down into the water, the back
	// ones first and darker.
	type root struct{ dx, h, r, bias float64 }
	roots := []root{{-13, 13, 0.8, -0.14}, {12, 15, 0.8, -0.14}, {-5, 11, 0.75, -0.14}, {6, 12, 0.75, -0.12},
		{-18, 17, 0.95, 0}, {17.5, 16, 0.95, 0}, {-10, 14, 1, 0.02}, {10.5, 13, 1, 0.02}, {-3, 10, 0.9, 0.04}, {3.5, 9.5, 0.9, 0.04}}
	for i, r := range roots {
		foot := P(r.dx, 0)
		foot[1] += 0.4 + float64(i%3)*0.5
		f.stalk(mangroveBark, r.r*1.25*T, r.r*0.75*T, r.bias, P(r.dx*0.08, r.h), P(r.dx*0.75, r.h+1.5), P(r.dx*1.02, r.h*0.55), foot, bark)
		if i >= 4 && i < 8 { // a root off the root, half way down
			mx, mh := r.dx*0.92, r.h*0.6
			side := math.Copysign(1, r.dx)
			foot := P(mx+side*4.6, 0)
			foot[1] += 0.8
			f.stalk(mangroveBark, r.r*0.7*T, r.r*0.5*T, r.bias-0.05, P(mx, mh), P(mx+side*3, mh-1), P(mx+side*4.5, 3), foot, bark)
		}
	}
	// The trunk rises from where the roots meet, short and crooked.
	th := 30.0
	if !full {
		th = 15
	}
	f.stalk(mangroveBark, 2.1*T, 1.5*T, 0.02, P(0, 8), P(-1.5, 16), P(2, 22), P(0.5, th), bark)
	if !full { // felled: a pale cut on the stump
		cut := P(0.9, th)
		f.c.ellipse(cut[0]*fk, cut[1]*fk, 1.9*T*fk, 0.9*T*fk, func(x, y int, nx, ny float64) color.RGBA { return woodEndHD(x, y, nx, ny, 4, 86) })
		return f.done(0.66, bx, base)
	}
	for _, b := range [][4]float64{{-0.5, 26, -11, 36}, {0.8, 27, 11, 38}, {0.5, 29, 1, 42}, {-0.3, 24, -16, 31}, {0.4, 25, 15, 33}} {
		f.stalk(mangroveBark, 1.3*T, 0.7*T, 0, P(b[0], b[1]), P(b[0], b[1]+3), P(b[2]*0.7, b[3]-2), P(b[2], b[3]), bark)
	}
	// The crown: thick, glossy leaves in a broad low dome. Its blobs are
	// those of the first drawing (x across it, y down from its top).
	var blobs []blob
	for _, b := range []blob{{13, 25, 8.5}, {39, 26, 8.5}, {26, 18, 10.5}, {18, 18, 8}, {34, 17.5, 8}, {26, 26, 9}, {8, 29, 5}, {44, 30, 5}} {
		p := P(b.x-26, 61-b.y)
		blobs = append(blobs, blob{p[0], p[1], b.r * S})
	}
	cc := P(0, 61-22)
	leafClusterHD(f.c, scaledBlobs(blobs), cc[0]*fk, cc[1]*fk, 21*S*fk, 13*S*fk, mangroveLeaf, 866, leafLook{4.8, 4.6, 2})
	for i := 0; i < 24; i++ { // old leaves turning yellow, as mangrove leaves do
		p := P(7+rng.Float64()*38-26, 61-(12+rng.Float64()*19))
		if f.c.opaque(int(p[0]*fk), int(p[1]*fk)) {
			leafHD(f.c, p[0]*fk, p[1]*fk, 3.6, 1.7, 0.5+rng.Float64()*2, mkRamp(0x7c7c24, 0xa4a030, 0xc8c044, 0xe4dc6c), 0.55)
		}
	}
	for i := 0; i < 16; i++ { // the seedlings: green pencils hanging under the leaves
		p := P(9+float64(i)*2.4+rng.Float64()*1.2-26, 61-(30+3.5*math.Sin(float64(i)*1.3)+rng.Float64()*2))
		if !f.c.opaque(int(p[0]*fk), int((p[1]-1.5)*fk)) {
			continue
		}
		pr := aCap(p[0], p[1], p[0]+0.2, p[1]+4+rng.Float64()*1.5, 0.3, 0.42)
		f.q.fill(pr, shd(mangroveLeaf, pr, 0.8, 0.22))
	}
	return f.done(0.66, bx, base)
}

// genMangroveRipples is the rings the stilt roots stand in, drawn under a
// mangrove that grows in the water.
func genMangroveRipples() (*canvas, float64, float64) {
	c := hdCanvas(int(math.Ceil(52*mangroveS)), 14)
	bx, by := math.Round(26*mangroveS)*fk, 7.0*fk
	for i, dx := range []float64{-18, -10, -3, 3.5, 10.5, 17.5, -14, 14} {
		rx := 4.2 + float64(i%3)
		cx, cy := bx+dx*mangroveS*fk, by+float64(i%3)*1.5
		for k := 0; k < 40; k++ {
			a := 2 * math.Pi * float64(k) / 40
			if (k+i)%5 == 0 {
				continue // broken rings
			}
			for _, r := range []float64{rx, rx * 1.9} {
				c.put(int(cx+math.Cos(a)*r*fk/2), int(cy+math.Sin(a)*r*fk/2*0.45), rippleCol)
			}
		}
	}
	return c, bx, by
}

// --- horsetail ---------------------------------------------------------------------------------

func genHorsetailHD(full bool) (*canvas, float64, float64) {
	f := newFlo(34, 36)
	rng := rand.New(rand.NewPCG(87, 31))
	const bx, base = 17.0, 33.0
	type stem struct {
		x, h, lean float64
		fertile    bool
	}
	var stems []stem
	for i := 0; i < 9; i++ {
		s := stem{x: bx - 6.5 + float64(i)*1.5 + rng.Float64()*1.6, h: 12 + rng.Float64()*15, fertile: i == 2 || i == 7}
		s.lean = (s.x-bx)*0.3 + (rng.Float64()-0.5)*1.6
		if s.fertile {
			s.h *= 0.72
		}
		if !full {
			s.h, s.lean = 2+rng.Float64()*2.5, 0
		}
		stems = append(stems, s)
	}
	pale := mkRamp(0x6a5a34, 0x94824c, 0xbcaa6c, 0xdcd094, 0xf0e8bc)
	for pass := 0; pass < 2; pass++ {
		for i, s := range stems {
			if (i%2 == 0) != (pass == 0) {
				continue
			}
			rp, bias := horsetailRamp, -0.1+0.14*float64(pass)
			if s.fertile {
				rp = pale
			}
			top := [2]float64{s.x + s.lean, base - s.h}
			const joint = 2.3
			// Whorls of thin branches at every joint, longest low down: a
			// bottle brush. They go on before the stem, which lies over them.
			if full && !s.fertile {
				n := int(s.h / joint)
				for j := 2; j < n; j++ {
					t := float64(j) * joint / s.h
					jx, jy := s.x+s.lean*t*t, base-s.h*t
					ln := (5.2*(1-t) + 1) * (0.8 + 0.4*rng.Float64())
					for k := -3; k <= 3; k++ {
						if k == 0 {
							continue
						}
						a := 0.5 + 0.2*math.Abs(float64(k)) // how far up it points
						dx := math.Copysign(math.Cos(a), float64(k)) * ln * (0.4 + 0.2*math.Abs(float64(k)))
						dy := -math.Sin(a) * ln * 0.75
						tone := 2 + (k+j+3)%3
						if pass == 0 {
							tone--
						}
						f.c.line(jx*fk, jy*fk, (jx+dx)*fk, (jy+dy)*fk, func(x, y int, t float64) color.RGBA { return rp.at(tone + int(t*1.5)) })
					}
				}
			}
			f.stalk(rp, 0.52, 0.3, bias, [2]float64{s.x, base}, [2]float64{s.x, base - s.h*0.4}, [2]float64{s.x + s.lean*0.5, base - s.h*0.75}, top,
				func(sv, v, l float64, col color.RGBA) color.RGBA {
					if u := sv * s.h / joint; u-math.Floor(u) < 0.16 {
						return toneOf(rp, l*0.86+bias-0.3) // the dark, toothed sheath at each joint
					}
					return col
				})
			if full && s.fertile { // a cone on top
				cone := aEll(top[0], top[1]-1.3, 0.95, 1.8)
				f.q.fill(cone, func(x, y int, _ float64) color.RGBA {
					l := cone.lum(x, y, 1.2*fk)
					if (x+2*y)%4 < 2 {
						l -= 0.2
					}
					return toneOf(mkRamp(0x4a3018, 0x74502a, 0x9c7440, 0xc09a5c, 0xdcc084), l)
				})
			}
		}
	}
	f.tuft(rng, bx, base+0.6, 18, 10, 3.2, bladeRamp)
	return f.done(0.55, bx, base)
}

func init() {
	floraGens[sim.FloraLily] = genLilyHD
	floraGens[sim.FloraLotus] = genLotusHD
	floraGens[sim.FloraSeaweed] = genSeaweedHD
	floraGens[sim.FloraMangrove] = genMangroveHD
	floraGens[sim.FloraHorsetail] = genHorsetailHD
}
