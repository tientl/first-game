package client

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The ice age: snowy ground, snow falling over it, frosted pines, the
// mammoth and the sabre-tooth, and what they give.

var (
	snowRamp    = mkRamp(0x8fa3b8, 0xaebfd0, 0xc9d6e2, 0xdfe8f0, 0xf2f6fa, 0xffffff)
	mammothRamp = mkRamp(0x2a1a10, 0x43291a, 0x5f3c25, 0x7e5433, 0x9e6e45, 0xbb8a5c)
	ivoryRamp   = mkRamp(0x9c8b68, 0xc9b994, 0xe8dcc0, 0xfbf5e6)
)

// Pose indices.
const (
	mammothWalkA = iota
	mammothWalkB
	mammothCharge // head down, tusks levelled
	mammothGore   // head tossed up
	mammothEatA   // the trunk curled down, pulling grass...
	mammothEatB   // ...and up to the mouth
	mammothPoses
)

const (
	sabreStand = iota
	sabreWalkA
	sabreWalkB
	sabreStalk  // crouched, creeping
	sabreSpring // in the air, jaws open, claws out
	sabreCower  // flat on the ground, ears back
	sabreEatA   // crouched over a kill: jaws shut...
	sabreEatB   // ...and open, tearing
	sabrePoses
)

// --- painting fur -----------------------------------------------------------

// The mammoth and the sabre-tooth are painted pixel for pixel rather than
// drawn small and enlarged: each part of the body is a chain of knobs, its
// roundness lit from the upper left, and a pelt gives every pixel its colour.

// knob is a point on a limb's spine: where it is and how thick.
type knob = [3]float64

// pelt colours one pixel of a painted part: l is how brightly the sun
// catches it, ny which way its surface faces (-1 up, 1 down) and t how far
// along the part's spine it lies.
type pelt func(x, y int, l, ny, t float64) color.RGBA

// soft maps t in [0, 1] to a colour like pick, but dithers only gently, so
// fur keeps its strands instead of dissolving into a checkerboard.
func (r ramp) soft(t float64, x, y int) color.RGBA {
	p := clamp01(t) * float64(len(r)-1)
	i := int(p)
	if p-float64(i) > 0.5+(dither(x, y)-0.5)*0.55 {
		i++
	}
	return r.at(i)
}

// furShape is the pixels a painted part covers.
type furShape struct {
	x0, y0, w, h int
	in           []bool
}

func (s *furShape) has(x, y int) bool {
	x, y = x-s.x0, y-s.y0
	return x >= 0 && y >= 0 && x < s.w && y < s.h && s.in[y*s.w+x]
}

// bottom is the lowest row the shape covers in column x.
func (s *furShape) bottom(x int) (int, bool) {
	for y := s.y0 + s.h - 1; y >= s.y0; y-- {
		if s.has(x, y) {
			return y, true
		}
	}
	return 0, false
}

// furPaint draws one rounded part of a body: the chains of knobs swell into
// a single surface, shaded as a solid thing and coloured by f. Where the
// part lies over what is already drawn it casts a thin dark seam, as strong
// as seam says, so that a limb reads against the body behind it; seamTop is
// the row above which that seam fades away (a leg grows out of its shoulder).
func furPaint(c *canvas, seam, seamTop float64, f pelt, chains ...[]knob) *furShape {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, ch := range chains {
		for _, k := range ch {
			x0, y0 = min(x0, k[0]-k[2]), min(y0, k[1]-k[2])
			x1, y1 = max(x1, k[0]+k[2]), max(y1, k[1]+k[2])
		}
	}
	s := &furShape{x0: int(math.Floor(x0)) - 1, y0: int(math.Floor(y0)) - 1}
	s.w, s.h = int(math.Ceil(x1))+2-s.x0, int(math.Ceil(y1))+2-s.y0
	s.in = make([]bool, s.w*s.h)
	hgt := make([]float64, s.w*s.h)
	along := make([]float64, s.w*s.h)
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			px, py := float64(s.x0+i)+0.5, float64(s.y0+j)+0.5
			best, bt := 0.0, 0.0
			for _, ch := range chains {
				n := len(ch)
				for q := 0; q < n; q++ {
					a, b := ch[q], ch[min(q+1, n-1)]
					dx, dy := b[0]-a[0], b[1]-a[1]
					u := 0.0
					if d2 := dx*dx + dy*dy; d2 > 0 {
						u = clamp01(((px-a[0])*dx + (py-a[1])*dy) / d2)
					}
					r := lerpf(a[2], b[2], u)
					d := math.Hypot(px-(a[0]+dx*u), py-(a[1]+dy*u))
					if d < r {
						if hh := math.Sqrt(r*r - d*d); hh > best {
							best, bt = hh, (float64(q)+u)/float64(max(1, n-1))
						}
					}
				}
			}
			if best > 0 {
				s.in[j*s.w+i] = true
				hgt[j*s.w+i], along[j*s.w+i] = best, bt
			}
		}
	}
	// Soften the creases where knobs meet.
	at := func(g []float64, i, j int) float64 {
		if i < 0 || j < 0 || i >= s.w || j >= s.h {
			return 0
		}
		return g[j*s.w+i]
	}
	sm := make([]float64, len(hgt))
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			sm[j*s.w+i] = (4*at(hgt, i, j) + 2*(at(hgt, i-1, j)+at(hgt, i+1, j)+at(hgt, i, j-1)+at(hgt, i, j+1)) +
				at(hgt, i-1, j-1) + at(hgt, i+1, j-1) + at(hgt, i-1, j+1) + at(hgt, i+1, j+1)) / 16
		}
	}
	// The seam on what lies behind.
	if seam > 0 {
		done := map[[2]int]bool{}
		for j := 0; j < s.h; j++ {
			for i := 0; i < s.w; i++ {
				if !s.in[j*s.w+i] {
					continue
				}
				for _, d := range [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}} {
					x, y := s.x0+i+d[0], s.y0+j+d[1]
					if s.has(x, y) || done[[2]int{x, y}] || c.at(x, y).A < 128 {
						continue
					}
					k := seam * clamp01((float64(y)-seamTop)/4+1)
					if d[0] < 0 || d[1] < 0 {
						k *= 0.6 // the lit side
					}
					done[[2]int{x, y}] = true
					c.put(x, y, shadeRGB(c.at(x, y), 1-k))
				}
			}
		}
	}
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			if !s.in[j*s.w+i] {
				continue
			}
			gx := (at(sm, i+1, j) - at(sm, i-1, j)) / 2
			gy := (at(sm, i, j+1) - at(sm, i, j-1)) / 2
			n := math.Sqrt(gx*gx + gy*gy + 1)
			nx, ny, nz := -gx/n, -gy/n, 1/n
			l := clamp01(0.2 + 0.8*(nx*lightDir[0]+ny*lightDir[1]+nz*lightDir[2]))
			if col := f(s.x0+i, s.y0+j, l, ny, along[j*s.w+i]); col.A > 0 {
				c.put(s.x0+i, s.y0+j, col)
			}
		}
	}
	return s
}

// furHang lets long hair fall from the underside of s between columns xa and
// xb, as far as length says for each column, darkening towards the tips.
func furHang(c *canvas, s *furShape, xa, xb int, rp ramp, light float64, length func(x int) float64) {
	for x := xa; x <= xb; x++ {
		yb, ok := s.bottom(x)
		if !ok {
			continue
		}
		n := int(length(x)*(0.5+0.5*vnoise(float64(x)*0.45, 0.5, 312)) + hashf(x, 1, 313)*2.2)
		strand := (hashf(x, 0, 311) - 0.5) * 0.22
		for k := 1; k <= n; k++ {
			l := light - 0.26*float64(k)/float64(n) + strand
			c.put(x, yb+k, rp.soft(l, x, yb+k))
		}
	}
}

// furEdge rims the finished drawing in the darkest tone of whatever it
// borders, and clears anything that sank below the ground at row floor.
func furEdge(c *canvas, floor int, strength float64) {
	for y := floor; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.put(x, y, color.RGBA{})
		}
	}
	c.outline(strength)
}

// --- mammoth ----------------------------------------------------------------

var (
	mammothFur  = mkRamp(0x1d120e, 0x2e1b13, 0x43271a, 0x5b3520, 0x774727, 0x935c31, 0xb07841, 0xcb985d)
	mammothSkin = mkRamp(0x21160f, 0x35241a, 0x4c3526, 0x644835, 0x7e5d46, 0x98765b)
	tuskRamp    = mkRamp(0x7d6744, 0xab9165, 0xd3bd90, 0xeedfba, 0xfdf6e0)
)

func genMammoth(pose int) sprite { return critter("mammoth", 0, pose, 0) }

// mammothPelt is the shaggy coat: long strands hanging down it, lying in
// ragged tiers one over another.
func mammothPelt(bias float64) pelt {
	return func(x, y int, l, ny, t float64) color.RGBA {
		fx, fy := float64(x), float64(y)
		strand := vnoise(fx*0.75+fy*0.1, fy*0.15, 301)
		lock := vnoise(fx*0.4, fy*0.07, 302)
		l = 0.1 + l*0.76 + (strand-0.5)*0.2 + (lock-0.5)*0.16 - ny*0.08 + bias
		// Single locks, a few pixels long, lying down the coat.
		switch q := hashf(x, int(math.Floor((fy+hashf(x, 7, 307)*7)/6)), 308); {
		case q < 0.2:
			l += 0.09
		case q > 0.82:
			l -= 0.08
		}
		// The tiers: a dark ragged hem, the hair below it catching the light.
		v := fy + vnoise(fx*0.11, 0.5, 303)*9 + hashf(x, 0, 304)*2.6
		k := 0.4 + 0.6*vnoise(fx*0.2, fy*0.2, 305)
		switch ph := math.Mod(v, 10); {
		case ph < 1.7:
			l -= 0.25 * k
		case ph < 4.2:
			l += 0.1 * k
		}
		return mammothFur.soft(l, x, y)
	}
}

func mammothCanvas(pose int) (*canvas, float64, float64) {
	c := newCanvas(106, 84)
	const gy = 80.0
	sw := 3.0 // an ambling gait: the legs of one side swing together, far side forward
	switch pose {
	case mammothWalkB:
		sw = -3
	case mammothCharge, mammothGore:
		sw = 4.5
	case mammothEatA, mammothEatB:
		sw = 1.5
	}
	leg := func(x, hy, r, s float64, far bool) {
		bias, dark := 0.0, 0
		if far {
			bias, dark = -0.2, 2
		}
		lift := 0.0
		sh := furPaint(c, 0.3, 0, func(px, py int, l, ny, t float64) color.RGBA {
			return mammothPelt(bias-0.1*t)(px, py, l, ny, t)
		}, []knob{{x, hy, r}, {x + s*0.35, hy + 12, r * 0.84}, {x + s, gy - 6.5 - lift, r * 0.78}, {x + s + 0.6, gy - 4.6 - lift, r * 0.8}})
		// Toenails along the front of the foot.
		for i, nx := range []float64{-0.5, 2.2, 4.6} {
			px, py := int(x+s+nx), int(gy-1-lift)
			if far && i == 0 {
				continue
			}
			for _, q := range [][2]int{{px, py}, {px + 1, py}, {px, py - 1}} {
				if sh.has(q[0], q[1]) {
					c.put(q[0], q[1], tuskRamp.at(2+min(1, i)-dark))
				}
			}
		}
	}
	// The tail, then the legs of the far side.
	furPaint(c, 0, 0, mammothPelt(-0.12), []knob{{8, 40, 1.8}, {4.6, 47, 1.2}, {4, 54, 1.8}})
	far := sw
	if far < 0 {
		far /= 3 // so that they still show behind the near ones
	}
	leg(30, 52, 6.6, far*0.8, true)
	leg(59, 51, 6.6, far, true)
	leg(20.5, 54, 7.2, -sw*0.8, false)
	leg(49, 54, 7.2, -sw, false)

	// The body: a sloping back that rises to the hump over the shoulders.
	body := furPaint(c, 0.25, 0, mammothPelt(0),
		[]knob{{19.5, 48, 11.5}, {33, 45, 13.5}, {47, 41, 16.5}},
		[]knob{{49, 31, 12}, {43, 36, 12}})
	furHang(c, body, 8, 66, mammothFur, 0.36, func(x int) float64 {
		return 3.5 + 4.5*math.Sin(clamp01(float64(x-10)/52)*math.Pi) // longest under the belly
	})

	// The head swings about the neck, carrying trunk and tusks with it.
	a, hdy := 0.0, 0.0
	switch pose {
	case mammothCharge:
		a, hdy = 0.3, 3
	case mammothGore:
		a, hdy = -0.36, -1
	case mammothEatA, mammothEatB:
		a, hdy = 0.12, 1
	}
	h := func(x, y, r float64) knob {
		X, Y := rot(x, y, 58, 36, a)
		return knob{X, Y + hdy, r}
	}
	ivory := func(bias float64) pelt {
		return func(x, y int, l, ny, t float64) color.RGBA {
			return tuskRamp.soft(0.28+l*0.72-(1-t)*0.14+bias, x, y) // stained where it leaves the lip
		}
	}
	tusk := func(dx, dy, bias float64) {
		furPaint(c, 0.35, 0, ivory(bias), []knob{h(73+dx, 43+dy, 2.7), h(76+dx, 53+dy, 2.7), h(82.5+dx, 60+dy, 2.4),
			h(90.5+dx, 60.5+dy, 2), h(96+dx, 55+dy, 1.4), h(98+dx, 48+dy, 0.8)})
	}
	tusk(3, -2.5, -0.26) // the far tusk
	// The trunk.
	trunk := []knob{h(75.5, 39, 5.6), h(80, 50, 4.7), h(81, 62, 3.7), h(79.5, 71, 2.9), h(82.5, 75.5, 2.2)}
	switch pose {
	case mammothCharge: // tucked back out of the way of the tusks
		trunk = []knob{h(75.5, 39, 5.6), h(80, 48, 4.7), h(84, 55, 3.6), h(89, 53, 2.8), h(89, 48, 2.2)}
	case mammothGore: // flung up, trumpeting
		trunk = []knob{h(75.5, 39, 5.6), h(84, 40, 4.7), h(91, 33, 3.7), h(92.5, 23, 2.9), h(96.5, 19, 2.2)}
	case mammothEatA: // down to a tuft on the ground, ahead of the feet
		trunk = []knob{h(75.5, 39, 5.6), h(80, 50, 4.7), h(81.5, 62, 3.7), h(81, 71, 2.9), {86, gy - 3, 2.3}}
	case mammothEatB: // and curled up to the mouth with it
		trunk = []knob{h(75.5, 39, 5.6), h(80.5, 49, 4.7), h(80, 57, 3.7), h(74.5, 58, 3), h(71.5, 52, 2.4)}
	}
	furPaint(c, 0.3, 0, func(x, y int, l, ny, t float64) color.RGBA {
		l = 0.2 + l*0.72 + (vnoise(float64(x)*0.9, float64(y)*0.3, 306)-0.5)*0.14
		if math.Mod(t*15+0.3, 1) < 0.3 {
			l -= 0.1 // the rings of its wrinkles
		}
		return mammothSkin.soft(l, x, y)
	}, trunk)
	// The skull: a high dome, the face falling steeply to the trunk.
	chin := furPaint(c, 0.25, 0, mammothPelt(-0.1), []knob{h(68.5, 43, 5.2), h(72, 45, 3.6)})
	furHang(c, chin, 0, c.w, mammothFur, 0.3, func(int) float64 { return 4 })
	furPaint(c, 0.42, 0, mammothPelt(-0.02),
		[]knob{h(67, 31, 10.5), h(73, 38, 7.2)},
		[]knob{h(66.5, 22.5, 7.5), h(68.5, 28, 8.6)})
	tusk(0, 0, 0.04)
	// A small ear, an eye under its brow, and the shag of the crown.
	furPaint(c, 0.5, 0, mammothPelt(0.08), []knob{h(61, 29.5, 3.6), h(62, 36, 3)})
	e := h(73, 31.5, 0)
	ex, ey := int(e[0]), int(e[1])
	c.put(ex, ey, colInk)
	c.put(ex+1, ey, colInk)
	c.put(ex, ey+1, colInk)
	c.put(ex+1, ey+1, hex(0x3a2314))
	c.put(ex, ey, hex(0xf4e6c6)) // the glint in it
	c.put(ex-1, ey-1, mammothFur[1])
	c.put(ex, ey-1, mammothFur[1])
	c.put(ex+1, ey-1, mammothFur[1])
	for i := -6; i <= 5; i++ { // the shag on the crown
		if hashf(i, 3, 314) < 0.55 {
			t := h(66.5+float64(i), 24, 0)
			for y := int(t[1]); y > 0; y-- {
				if c.at(int(t[0]), y).A < 128 {
					c.put(int(t[0]), y, mammothFur.at(4+(i+6)%2))
					break
				}
			}
		}
	}
	for x := 12; x < 58; x++ { // and along the back
		if hashf(x, 5, 315) < 0.3 {
			for y := 60; y > 0; y-- {
				if c.at(x, y).A < 128 {
					c.put(x, y, mammothFur.at(5+x%2))
					break
				}
			}
		}
	}
	furEdge(c, int(gy), 0.72)
	// Room behind it, so that the front third of the sprite, which is what a
	// trophy shows, is the whole head with its tusks.
	const room = 18
	wide := newCanvas(c.w+room, c.h)
	wide.over(c, room, 0)
	return wide, 43 + room, gy - 2
}

// --- sabre-tooth ------------------------------------------------------------

var (
	sabreFur   = mkRamp(0x2b1a12, 0x4a2c1a, 0x6e4424, 0x93602f, 0xb7813f, 0xd3a257, 0xe7c27c)
	sabreCream = mkRamp(0x6b5440, 0x9a8164, 0xc4ad8a, 0xe3d2b0, 0xf6ecd4)
)

func genSabre(pose int) sprite { return critter("sabre", 0, pose, 0) }

// sabrePelt is a short tawny coat, dappled along the back and paling to
// cream wherever it faces the ground; pale is how readily it does.
func sabrePelt(bias, pale float64) pelt {
	return func(x, y int, l, ny, t float64) color.RGBA {
		l = 0.14 + l*0.78 - ny*0.13 + bias
		switch {
		case ny > 0.55-pale:
			return sabreCream.soft(l+0.45, x, y)
		case ny > 0.36-pale:
			l += 0.17
		}
		if row := y / 3; ny < 0.2 && y%3 == 0 && (x+row%2*3)%6 < 2 && hashf((x+row%2*3)/6, row, 321) < 0.75 {
			l -= 0.2 // a dapple
		}
		return sabreFur.soft(l, x, y)
	}
}

func sabreCanvas(pose int) (*canvas, float64, float64) {
	// Designed a quarter larger than it is drawn; G is the ground.
	const s, ox, oy, G = 0.88, 4.0, 5.0, 38.0
	c := newCanvas(64, 41)
	S := func(x, y, r float64) knob { return knob{x*s + ox, y*s + oy, max(r*s, 0.75)} }
	// The body leans by tilt about its middle and sinks by drop; the head
	// turns by nod on the neck and shifts by (hx, hy); the jaw drops by gape.
	tilt, drop, nod, hx, hy, gape := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	earBack := 0.0
	tail := []knob{{10.5, 17, 2.2}, {6.8, 16.2, 1.9}, {5.2, 14, 1.6}}
	// Where each paw stands (near fore, far fore, near hind, far hind); in
	// the air they are given in the body's own frame instead.
	paws := [4][2]float64{{35.5, 34}, {41.5, 34}, {15, 34}, {23, 34}}
	air := false
	switch pose {
	case sabreWalkA:
		paws = [4][2]float64{{41, 34}, {35.5, 33.6}, {12, 33.6}, {25, 34}}
	case sabreWalkB:
		paws = [4][2]float64{{32, 33.6}, {44, 34}, {21, 34}, {15.5, 33.6}}
	case sabreStalk:
		drop, hx, hy, nod = 5, 2.5, 4.5, 0.12
		paws = [4][2]float64{{42, 34}, {36, 34}, {13, 34}, {23, 34}}
		tail = []knob{{10.5, 17, 2}, {6.5, 17.5, 1.7}, {4, 19.5, 1.4}}
	case sabreSpring:
		tilt, drop, nod, gape, air = -0.3, -2, -0.12, 0.85, true
		paws = [4][2]float64{{48, 28}, {51, 24.5}, {4.5, 29.5}, {7, 32.5}}
		tail = []knob{{10.5, 17, 2}, {6.5, 15, 1.7}, {4.8, 11.8, 1.4}}
	case sabreCower:
		drop, hx, hy, nod, earBack = 8.6, 1, 5.5, 0.2, 1
		paws = [4][2]float64{{44, 34.4}, {47, 34.4}, {22, 34.4}, {25, 34.4}}
		tail = []knob{{10.5, 17, 2}, {6.5, 19.5, 1.7}, {7.5, 23.5, 1.4}}
	case sabreEatA, sabreEatB:
		tilt, drop, nod, hx, hy, gape = 0.14, 3.5, 0.5, 2.5, 3, 0.12
		if pose == sabreEatB {
			gape = 0.7
		}
		paws = [4][2]float64{{38, 34}, {42, 34}, {14, 34}, {20, 34}}
	}
	bd := func(x, y float64) (float64, float64) {
		X, Y := rot(x, y, 28, 22, tilt)
		return X, Y + drop
	}
	B := func(x, y, r float64) knob { X, Y := bd(x, y); return S(X, Y, r) }
	H := func(x, y, r float64) knob {
		X, Y := rot(x, y, 43, 15, nod)
		return B(X+hx, Y+hy, r)
	}
	leg := func(i int, hipX, hipY float64) {
		far := i%2 == 1
		X, Y := bd(hipX, hipY)
		px, py := paws[i][0], paws[i][1]+2
		if air {
			px, py = bd(px, py-2)
		}
		bias, pale := 0.0, -1.0
		if far {
			bias = -0.2
		}
		mid := func(t, dx float64) knob { return knob{lerpf(X, px, t) + dx, lerpf(Y, py, t), 0} }
		var ch []knob
		toe := S(px+2, py+0.3, 2)
		if air { // claws out
			tx, ty := px+(px-X)/math.Hypot(px-X, py-Y)*2.2, py+(py-Y)/math.Hypot(px-X, py-Y)*2.2
			toe = S(tx, ty, 1.7)
		}
		if i < 2 { // a fore leg: the elbow back
			fold := max(0, 14-math.Hypot(px-X, py-Y))
			e, w := mid(0.45, -0.8-fold*0.6), mid(0.84, 0.2-fold*0.15)
			ch = []knob{S(X, Y, 4.7), S(e[0], e[1], 3.3), S(w[0], w[1], 2.5), S(px, py, 2.3), toe}
		} else { // a hind leg: the knee forward, the hock back
			fold := max(0, 15-math.Hypot(px-X, py-Y))
			k, h := mid(0.4, 2+fold*0.55), mid(0.76, -2-fold*0.4)
			if air {
				k, h = mid(0.4, 0.5), mid(0.76, -0.6)
			}
			ch = []knob{S(X, Y, 5.6), S(k[0], k[1], 3.4), S(h[0], h[1], 1.9), S(px, py, 2.2), toe}
		}
		seam := 0.38
		furPaint(c, seam, ch[0][1]+1, func(x, y int, l, ny, t float64) color.RGBA {
			l = 0.22 + l*0.7 // slim things: keep them from going dark
			return sabrePelt(bias, pale)(x, y, l, ny, t)
		}, ch)
		if air { // the claws themselves
			tip := ch[len(ch)-1]
			c.put(int(tip[0]+1.4), int(tip[1]+0.6), tuskRamp[4])
			c.put(int(tip[0]+0.4), int(tip[1]+1.5), tuskRamp[3])
		}
	}
	leg(3, 19.5, 22)
	leg(1, 38.5, 23)
	tl := make([]knob, len(tail))
	for i, k := range tail {
		tl[i] = B(k[0], k[1], k[2])
	}
	furPaint(c, 0.2, 0, func(x, y int, l, ny, t float64) color.RGBA {
		return sabrePelt(-0.04-0.3*max(0, t-0.55), -1)(x, y, l, ny, t) // dark at its tip
	}, tl)
	// Deep-chested and high in the shoulder, the back sloping to lean
	// haunches; a thick neck.
	hump := 0.0
	if pose == sabreStalk || pose == sabreCower || pose == sabreEatA || pose == sabreEatB {
		hump = 1.6 // the shoulder blades stand up when it crouches
	}
	furPaint(c, 0.3, 0, sabrePelt(0, 0.12),
		[]knob{B(16, 20, 6), B(25, 19.6, 5.2), B(34.5, 18.8, 7.4)},
		[]knob{B(33.5, 14.8-hump, 5), B(38, 16, 6)})
	furPaint(c, 0, 0, sabrePelt(0, -1), []knob{B(38, 16, 6), H(45, 13.5, 5.6)})
	leg(2, 16, 22)
	leg(0, 35, 23)

	// The head: far ear, the open mouth, the jaw, then skull and muzzle.
	ear := func(x, y, r, bias float64) {
		x, y = x-earBack*2.2, y+earBack*1.6
		furPaint(c, 0.4, 0, func(px, py int, l, ny, t float64) color.RGBA {
			return sabreFur.soft(0.2+l*0.7+bias, px, py)
		}, []knob{H(x, y, r), H(x-0.2, y+1.5, r)})
	}
	ear(49.4, 5, 1.8, -0.22)
	jaw := func(x, y, r float64) knob {
		X, Y := rot(x, y, 47, 16.5, gape)
		return H(X, Y, r)
	}
	if gape > 0.3 {
		furPaint(c, 0, 0, func(x, y int, l, ny, t float64) color.RGBA { return hex(0x5c1616) },
			[]knob{H(49, 16.8, 2.4), jaw(53, 16.8, 1.6)})
	}
	fang := func(x, top, length, bias float64) {
		furPaint(c, 0, 0, func(px, py int, l, ny, t float64) color.RGBA {
			return tuskRamp.at(int(3.9 + bias - t*0.4 + l*0.6))
		}, []knob{H(x, top, 1.1), H(x-0.2, top+length*0.6, 0.9), H(x-0.7, top+length, 0.45)})
	}
	fang(56, 17, 4.2, -1.6)
	furPaint(c, 0.3, 0, func(x, y int, l, ny, t float64) color.RGBA {
		return sabreFur.soft(0.34+l*0.6, x, y) // the chin, in the shadow of the muzzle
	}, []knob{jaw(48, 17.6, 2.3), jaw(52.4, 18.3, 1.7)})
	furPaint(c, 0.32, 0, func(x, y int, l, ny, t float64) color.RGBA {
		return sabrePelt(0.03, -1)(x, y, l, ny, t)
	}, []knob{H(48, 11.6, 6.6)})
	furPaint(c, 0.12, 0, func(x, y int, l, ny, t float64) color.RGBA {
		l = 0.2 + l*0.78
		if ny > 0.1 {
			return sabreCream.soft(l+0.05, x, y) // a pale muzzle
		}
		return sabreFur.soft(l, x, y)
	}, []knob{H(52, 13.4, 4.4), H(54.6, 14.2, 3.8)})
	fang(54.4, 17.4, 5.8, 0)
	ear(44.8, 5.6, 2.1, 0.04)
	e := H(51.2, 10.6, 0)
	c.put(int(e[0]), int(e[1]), hex(0xf2c230))
	c.put(int(e[0])+1, int(e[1]), colInk)
	c.put(int(e[0]), int(e[1])-1, sabreFur[0])
	c.put(int(e[0])+1, int(e[1])-1, sabreFur[0])
	n := H(57.9, 12.2, 0)
	c.put(int(n[0]), int(n[1]), hex(0x2a1614))
	gy := G*s + oy
	furEdge(c, int(gy)+1, 0.72)
	if air {
		gy += 4 // off the ground
	}
	return c, 28*s + ox, gy
}

// --- what they give ---------------------------------------------------------

// iconFur is a thick, shaggy pelt.
func iconFur() *canvas {
	c := newCanvas(16, 16)
	c.ellipse(8, 8.5, 6.5, 5.5, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if (x+y/2)%3 == 0 {
			l -= 0.25
		}
		return mammothRamp.pick(l, x, y)
	})
	for x := 3; x <= 13; x += 2 { // a ragged edge of long hair
		c.set(x, 14, mammothRamp[2])
	}
	c.outline(0.8)
	return c
}

// iconIvory is a curved tusk.
func iconIvory() *canvas {
	c := newCanvas(16, 16)
	tube(c, spine([3]float64{2, 13, 1.8}, [3]float64{7, 12, 1.6}, [3]float64{12, 8, 1.2}, [3]float64{13.5, 3, 0.6}), ivoryRamp, 0.1)
	c.outline(0.8)
	return c
}

// iconFurCoat is a coat of fur with a hide belt.
func iconFurCoat() *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{2, 4, 6, 2, 10, 2, 14, 4, 13, 9, 12.5, 14.5, 3.5, 14.5, 3, 9}, func(x, y int) color.RGBA {
		l := 0.75 - (float64(x)-8)/8*0.3
		if (x+y/2)%3 == 0 {
			l -= 0.2
		}
		return mammothRamp.pick(l, x, y)
	})
	c.line(4, 9, 12, 9, func(x, y int, _ float64) color.RGBA { return hideRamp.pick(0.6, x, y) })
	c.line(6, 3, 10, 3, func(x, y int, _ float64) color.RGBA { return snowRamp[4] }) // pale fur at the collar
	c.outline(0.8)
	return c
}

// iconIvorySpear is a spear tipped with a long ivory point.
func iconIvorySpear() *canvas {
	c := newCanvas(16, 16)
	c.line(1, 15, 10, 6, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.8, x, y) })
	c.line(2, 15, 11, 6, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	c.polygon([]float64{9, 6, 15.5, 0.5, 11, 7.5}, func(x, y int) color.RGBA { return ivoryRamp.pick(0.95-float64(y)*0.05, x, y) })
	c.set(9, 7, hideRamp[2])
	c.set(10, 7, hideRamp[1])
	c.outline(0.85)
	return c
}

// genHeldIvorySpear is the ivory spear as held, like genHeldSpear.
func genHeldIvorySpear() sprite {
	c := newCanvas(6, 28)
	c.rect(2, 7, 2, 21, func(x, y int) color.RGBA { return woodRamp[1+(3-x)] })
	c.polygon([]float64{3, 0, 5, 6, 3, 8.5, 1, 6}, func(x, y int) color.RGBA { return ivoryRamp.pick(0.95-float64(x)*0.1, x, y) })
	c.set(2, 8, hideRamp[2])
	c.set(3, 9, hideRamp[1])
	c.outline(0.8)
	return mk(c, 3, 22)
}

// --- snow falling -----------------------------------------------------------

// snowCover is how much of the view around the camera is snow, 0 to 1.
func (g *Game) snowCover(cam camera) float64 {
	w, h := float64(g.world.Bounds().Dx()), float64(g.world.Bounds().Dy())
	n, s := 0, 0
	for y := 0.1; y < 1; y += 0.2 {
		for x := 0.1; x < 1; x += 0.2 {
			tx, ty := (w*x+cam.ox)/TileW, (h*y+cam.oy)/TileH
			n++
			if g.tileAt(int(math.Floor(tx)), int(math.Floor(ty))) == sim.TileSnow {
				s++
			}
		}
	}
	return float64(s) / float64(n)
}

// drawSnowfall drifts flakes down over the snowy parts of the world. Like
// the embers, each flake is a function of its index and the clock.
func (g *Game) drawSnowfall(dst drawTarget, cam camera) {
	cover := g.snowCover(cam)
	if cover <= 0 || g.caveIn() > 0.5 {
		return
	}
	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	n := int(140 * cover)
	t := float64(g.frame) / 60
	for i := range n {
		r := rand.New(rand.NewPCG(uint64(i), 177))
		speed := 12 + r.Float64()*14
		x := r.Float64()*w + math.Sin(t*0.8+float64(i))*6 + t*4
		y := r.Float64()*h + t*speed
		x, y = math.Mod(x+cam.ox*0.15, w), math.Mod(y+cam.oy*0.15, h)
		if x < 0 {
			x += w
		}
		if y < 0 {
			y += h
		}
		a := uint8(150 + r.IntN(100))
		size := float32(1)
		if r.Float64() < 0.2 {
			size = 2
		}
		fillRect(dst, float32(math.Round(x)), float32(math.Round(y)), size, size, color.RGBA{a, a, a, a}, false)
	}
}

// --- sounds -----------------------------------------------------------------

// genIceSFX adds the ice age's sounds to the effect bank.
func genIceSFX(r *rand.Rand, add func(name string, peak float64, x []float64)) {
	// A mammoth's trumpet: a bright, rising blare.
	add("trumpet", 0.9, growl(r, 1.1, 260, 420, 0.06, 0.5, 0.35, [2]float64{900, 1900}))
	// A sabre-tooth's snarl.
	add("snarl", 0.85, growl(r, 0.6, 140, 95, 0.03, 0.25, 0.9, [2]float64{500, 1100}))
	// A crunch of snow underfoot.
	x := make([]float64, samples(0.18))
	addAt(x, grains(r, 0.14, 500, 0.006, bandpass, 2600, 1.2, func(t float64) float64 { return env(t, 0.01, 0.05) }), 0, 1)
	add("step_snow", 1, x)
}

// iceSound plays what a mammoth or a sabre-tooth did when its state changed
// from o to c, and reports whether c is one of them.
func iceSound(o, c protocol.CreatureView) bool {
	x, y := float64(c.X), float64(c.Y)
	switch c.Kind {
	case sim.CreatureMammoth:
		switch {
		case c.State == sim.CreatureChase && o.State != sim.CreatureStrike:
			snd.playAt("mammoth_alarm", x, y, 0.8, 0.1)
		case c.State == sim.CreatureStrike:
			snd.playAt("whack", x, y, 0.9, 0.08)
		}
	case sim.CreatureSabre:
		switch {
		case c.State == sim.CreatureChase && calm(o.State): // it has picked its quarry
			if snd.ready("sabre_roar", 4*time.Second) {
				callAt("sabre_roar", x, y, 0.85, 1, 22)
			}
		case c.State == sim.CreatureStrike:
			callAt("sabre_snarl", x, y, 0.9, 1, 16) // see predsound.go
		case c.State == sim.CreatureCower:
			snd.playAt("sabre_cower", x, y, 0.4, 0.2)
		}
	default:
		return false
	}
	return true
}
