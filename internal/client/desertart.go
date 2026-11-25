package client

import (
	"image/color"
	"math"
)

// This file draws the desert island's own sights: the ancient pyramid that
// stands in the dunes and the giant scorpions that guard it.

// desertArt holds the desert island's sprites. The scorpion faces right;
// the caller mirrors it for left (mirrorSprite).
type desertArt struct {
	pyramid  sprite                // the whole pyramid, seen in 3/4 view from the south
	scorpion [scorpionPoses]sprite // a giant scorpion, side view facing right
}

const (
	scorpionWalkA = iota // walk cycle, frame A
	scorpionWalkB        // walk cycle, frame B
	scorpionPinch        // claws thrust forward, snapping
	scorpionSting        // tail arched forward over the body, striking
	scorpionPoses
)

func genDesertArt() desertArt {
	var a desertArt
	c, ax, ay := pyramidCanvas()
	a.pyramid = mkUp(c, ax, ay)
	for p := range scorpionPoses {
		c, ax, ay := scorpionCanvas(p)
		a.scorpion[p] = mk(c, ax, ay)
	}
	return a
}

// --- the pyramid --------------------------------------------------------------

var (
	// Sandstone: paler and greyer than the dunes it stands in (desertRamp),
	// with deep brown shadows.
	sandstoneRamp = mkRamp(0x3a2420, 0x573627, 0x7a5034, 0x9c6f45, 0xb98d5a, 0xd2ab73, 0xe6c88f, 0xf5e0ae, 0xfdf1cf)
	capGoldRamp   = mkRamp(0x5e3a0e, 0x8f5d12, 0xc48d1c, 0xeebf34, 0xffe470, 0xfffad0)
	glyphBlue     = hex(0x2f5f8c)
	glyphRed      = hex(0x9a3a22)
)

// The pyramid's measurements, in art pixels. Its foot is pyrW by pyrD on
// the ground (7 by 5 tiles); pyrSteps courses of stone climb to a gilded
// capstone. Seen from the south, a course shows its front (pyrRise tall) and
// the top of the course below it (pyrTread deep).
const (
	pyrW      = 168.0
	pyrD      = 90.0
	pyrSteps  = 15 // the last is the capstone
	pyrRise   = 8.0
	pyrTread  = pyrD / 2 / pyrSteps // 3
	pyrCapH   = 11.0
	pyrMargin = 4 // room at the sides and top for the outline and the glint
	pyrBelow  = 3 // room under the base for drifted sand
	pyrCanW   = int(pyrW) + 2*pyrMargin
	pyrCanH   = int(pyrD/2+(pyrSteps-1)*pyrRise+pyrCapH) + pyrMargin + pyrBelow
	pyrAx     = float64(pyrCanW) / 2
	pyrAy     = float64(pyrCanH - pyrBelow)

	// The porch built out from the south face around the entrance.
	porchH    = 34.0 // its height
	porchRoof = 12.0 // the depth of its flat roof
	porchHW   = 22.0 // half its width
	doorHW    = 9.0
	doorH     = 24.0
)

// pyramidCanvas draws the pyramid; the anchor is the middle of the south
// edge of its footprint, on the ground. Points are worked out as (X, U):
// X east of the anchor and U up the screen from it.
func pyramidCanvas() (*canvas, float64, float64) {
	c := newCanvas(pyrCanW, pyrCanH)
	hw := pyrW / 2
	put := func(X, U float64, col color.RGBA) {
		c.put(int(math.Floor(X+pyrAx)), int(math.Floor(pyrAy-U)), col)
	}
	stone := func(l float64, x, y int) color.RGBA { return sandstoneRamp.pick(l, x, y) }

	for py := 0; py < c.h; py++ {
		for px := 0; px < c.w; px++ {
			X, U := float64(px)+0.5-pyrAx, pyrAy-(float64(py)+0.5)
			if U < 0 {
				continue
			}
			aX := math.Abs(X)
			// Weathering shared by every face: broad stains and faint streaks.
			stain := (fbm2(X*0.045+7, U*0.06, 301, 3) - 0.5) * 0.3
			if vnoise(X*0.6, U*0.07, 302) > 0.78 {
				stain -= 0.05
			}

			// The capstone: a little smooth pyramid of gold.
			{
				i := float64(pyrSteps - 1)
				chw := hw * (1 - i/pyrSteps)
				u0 := i * (pyrTread + pyrRise) // its south foot
				if aX < chw && U >= u0 {
					k := 1 - aX/chw
					switch {
					case U-u0 < (pyrCapH+pyrTread)*k:
						c.put(px, py, capGoldRamp.pick(0.5-X/chw*0.25+(U-u0)*0.02, px, py))
						continue
					case U-u0 < 2*pyrTread+(pyrCapH-pyrTread)*k:
						l := 0.98
						if X > 0 {
							l = 0.12
						}
						c.put(px, py, capGoldRamp.pick(l, px, py))
						continue
					}
				}
			}

			for i := pyrSteps - 2; i >= 0; i-- {
				fi := float64(i)
				shw := hw * (1 - fi/pyrSteps)       // half the width of this course
				nhw := hw * (1 - (fi+1)/pyrSteps)   // and of the one above
				u0 := fi * (pyrTread + pyrRise)     // the foot of its south front
				u1 := u0 + pyrRise                  // the south edge of its top
				u2 := u1 + pyrD - 2*fi*pyrTread     // the north edge of its top
				chip := hashf(i, 0, 303)*5 - 2      // corners knocked off at the back
				chipF := hashf(i, 1, 303)*4 - 1.5   // and worn at the front
				if aX >= shw || U < u0 || U >= u2 { // not on this course
					continue
				}
				if aX > nhw && (U > u2-chip-(shw-aX)*0.5 || U < u0+chipF-(shw-aX)) {
					break // broken corners of the course
				}
				switch {
				case U < u1: // the south front: a row of big blocks
					r := U - u0
					bw := 11 + math.Floor(hashf(i, 2, 303)*5)
					bp := (X + hw + hashf(i, 3, 303)*bw) / bw
					bx := int(math.Floor(bp))
					l := 0.47 - X/hw*0.05 + stain + (hashf(bx, i, 304)-0.5)*0.16
					gone := hashf(bx, i, 305) < 0.05 && aX < shw-6 && !(aX < porchHW+12 && U < porchH+16)
					switch {
					case gone: // a block fallen out: a dark socket
						l = 0.1 + r*0.02
						if (bp-math.Floor(bp))*bw < 2 {
							l += 0.12
						}
					case (bp-math.Floor(bp))*bw < 1:
						l -= 0.2 // the joint between two blocks
					case r >= pyrRise-1:
						l += 0.13 // the lit lip
					case r < 1:
						l -= 0.22 // where it stands on the course below
					case r < 2:
						l -= 0.08
					}
					if !gone && hashf(px, py, 306) < 0.05 {
						l -= 0.09 // pitting
					}
					c.put(px, py, stone(l, px, py))
				case aX <= nhw: // the top, seen in front of the next course
					l := 0.84 + stain*0.6
					col := stone(l, px, py)
					if fbm2(X*0.11, fi*1.7, 307, 2) > 0.58 { // sand lying on the ledge
						col = desertRamp.pick(0.7+stain, px, py)
					}
					c.put(px, py, col)
				default: // the top, seen beside the next course: a long ledge
					d := U - u1
					bl := 9 + math.Floor(hashf(i, 4, 303)*5)
					joint := math.Mod(d+hashf(i, 5, 303)*bl, bl) < 1
					edge := (aX - nhw) / (shw - nhw) // 0 by the course above, 1 at the drop
					var l float64
					if X < 0 { // the west side takes the sun
						l = 0.68 + stain*0.7 + (hashf(i, int(d/bl), 308)-0.5)*0.1
						switch {
						case joint:
							l -= 0.16
						case edge < 0.2:
							l -= 0.12
						case edge > 0.8:
							l += 0.1
						}
					} else { // the east side lies in the shadow of the courses above
						l = 0.2 + stain*0.4 + edge*0.12
						if joint {
							l -= 0.08
						}
					}
					c.put(px, py, stone(l, px, py))
				}
				break
			}
		}
	}

	// Cracks running down the south face.
	for k, cr := range [][3]float64{{-47, 96, 30}, {38, 74, 26}, {-20, 128, 18}, {61, 40, 22}} {
		X := cr[0]
		for d := 0.0; d < cr[2]; d++ {
			X += (hashf(int(d), k, 309) - 0.5) * 2.4
			px, py := int(math.Floor(X+pyrAx)), int(math.Floor(pyrAy-(cr[1]-d)))
			if c.opaque(px, py) {
				c.put(px, py, mix(c.at(px, py), sandstoneRamp.at(0), 0.6))
			}
		}
	}

	// A band of carved signs on the course above the porch.
	glyphs := func(X0, X1, U0 float64, seed uint32) {
		for X := X0; X < X1; X += 5 {
			g := hash2(int32(X), int32(U0), seed)
			ink := sandstoneRamp.at(1)
			switch g >> 20 & 7 {
			case 0:
				ink = glyphBlue
			case 1:
				ink = glyphRed
			}
			for b := range 12 {
				if g>>b&1 == 1 {
					put(X+float64(b%3), U0+float64(b/3), ink)
				}
			}
		}
	}
	glyphs(-42, 42, 5*(pyrTread+pyrRise)+2, 311)

	// shade darkens what is already drawn: cast shadows.
	shade := func(X, U, k float64) {
		px, py := int(math.Floor(X+pyrAx)), int(math.Floor(pyrAy-U))
		if c.opaque(px, py) {
			c.put(px, py, mix(c.at(px, py), sandstoneRamp.at(0), k))
		}
	}
	for U := 0.5; U < porchH+porchRoof; U++ { // the porch's shadow on the wall
		for d := 0.0; d < 4; d++ {
			if U < porchH+porchRoof-d*2 {
				shade(porchHW+1.5+d, U, 0.5-d*0.1)
			}
		}
		shade(-porchHW-1.5, U, 0.25)
	}

	// The porch: a battered front of great stones, a cornice, a flat roof.
	for U := 0.0; U < porchH+porchRoof; U++ {
		for X := -porchHW - 1; X <= porchHW+1; X++ {
			xc, aX := X+0.5, math.Abs(X+0.5)
			px, py := int(math.Floor(xc+pyrAx)), int(math.Floor(pyrAy-U-0.5))
			noise := (hashf(px, py, 312) - 0.5) * 0.08
			switch {
			case U >= porchH: // the roof
				if aX < porchHW {
					l := 0.9 + noise
					if U >= porchH+porchRoof-2 {
						l = 0.7
					}
					if int(aX)%9 == 8 {
						l -= 0.14
					}
					put(xc, U, stone(l, px, py))
				}
			case U >= porchH-3: // the cornice, jutting out
				l := 0.72 - xc/porchHW*0.08 + noise
				if U < porchH-2 {
					l = 0.34 // its underside
				}
				put(xc, U, stone(l, px, py))
			case aX < porchHW-1+(1-U/porchH)*2:
				l := 0.6 - xc/porchHW*0.1 + noise
				switch {
				case U >= doorH: // the lintel: one huge stone
					l += 0.06
					if U < doorH+1 && aX < doorHW+4 {
						l -= 0.25
					}
				case aX < doorHW: // the way in: dark, sand blown over the sill
					col := mix(hex(0x06040a), hex(0x2c1c16), clamp01((7-U)/9)+hashf(px, py, 313)*0.08)
					if xc > doorHW-2 { // the far jamb catches a little light
						col = stone(0.2+(doorH-U)*0.006, px, py)
					}
					if U < 2.5-aX*0.3 {
						col = desertRamp.pick(0.15, px, py)
					}
					put(xc, U, col)
					continue
				default: // the jambs
					if int(U)%7 == 6 || aX >= porchHW-1 && int(U)%7 == 2 {
						l -= 0.2
					}
					if aX < doorHW+1 {
						l += 0.14 // the edge of the opening
					}
				}
				put(xc, U, stone(l, px, py))
			}
		}
	}
	// Carving: a winged sun on the lintel, columns of signs down the jambs.
	for d := 3.0; d <= 13; d++ {
		for _, s := range []float64{-1, 1} {
			put(s*d, doorH+3+math.Floor(d/4), glyphBlue)
			if int(d)%2 == 0 {
				put(s*d, doorH+2+math.Floor(d/4), sandstoneRamp.at(1))
			}
		}
	}
	c.ellipse(pyrAx, pyrAy-doorH-3.5, 2.5, 2.5, func(x, y int, nx, ny float64) color.RGBA {
		return capGoldRamp.pick(0.75-nx*0.3-ny*0.3, x, y)
	})
	for _, s := range []float64{-17, 14} {
		for U := 3.0; U < 21; U += 5 {
			g := hash2(int32(s), int32(U), 314)
			for b := range 9 {
				if g>>b&1 == 1 {
					put(s+float64(b%3), U+float64(b/3), sandstoneRamp.at(1))
				}
			}
		}
	}

	// Two obelisks stand before the porch; the east one has snapped.
	obelisk := func(X0, top float64, broken bool) {
		for U := 0.5; U < top; U++ {
			w := 3.5
			if !broken && U >= top-5 {
				w = (top - U) * 0.7
			}
			if broken && U >= top-3 {
				w -= hashf(int(X0), int(U), 315) * 3.5
			}
			for d := 0.0; d < 3; d++ { // its shadow on the wall behind
				shade(X0+w+0.5+d, U, 0.45-d*0.12)
			}
			shade(X0-w-0.5, U, 0.3)
			for X := -3.0; X <= 3; X++ {
				if math.Abs(X) > w {
					continue
				}
				px, py := int(math.Floor(X0+X+pyrAx)), int(math.Floor(pyrAy-U))
				l := 0.86 - (X+3)*0.1
				if X >= 2 {
					l -= 0.15
				}
				if int(U)%4 == 1 && math.Abs(X) < 1.5 && U < top-6 && hashf(int(X0+X), int(U), 319) < 0.7 {
					l = 0.2 // signs cut into it
				}
				if !broken && U >= top-5 {
					c.put(px, py, capGoldRamp.pick(l, px, py))
					continue
				}
				c.put(px, py, stone(l, px, py))
			}
		}
	}
	obelisk(-34, 33, false)
	obelisk(34, 13, true)

	// Fallen blocks at the foot.
	for _, b := range [][4]float64{{43, 0, 9, 4}, {58, 0, 7, 5}, {67, 0, 5, 3}, {-66, 0, 7, 4}, {-52, 0, 4, 3}, {-74, 2, 4, 3}} {
		for U := b[1]; U < b[1]+b[3]+2; U++ {
			for X := b[0]; X < b[0]+b[2]; X++ {
				px, py := int(math.Floor(X+pyrAx)), int(math.Floor(pyrAy-U-0.5))
				l := 0.56 + (hashf(int(b[0]), 0, 316)-0.5)*0.12
				switch {
				case U >= b[1]+b[3]:
					l = 0.88
				case X >= b[0]+b[2]-1:
					l = 0.28
				case X < b[0]+1:
					l += 0.1
				}
				c.put(px, py, stone(l, px, py))
			}
		}
	}

	// Sand drifted against the base, deepest in the corners and beside the
	// porch, trodden away before the door.
	for px := 0; px < c.w; px++ {
		X := float64(px) + 0.5 - pyrAx
		aX := math.Abs(X)
		h := 1 + 4*fbm2(X*0.07, 3, 317, 2) +
			4*math.Exp(-math.Pow((aX-hw+4)/9, 2)) +
			3.5*math.Exp(-math.Pow((aX-porchHW-3)/5, 2)) +
			2.5*math.Exp(-math.Pow((X-50)/8, 2))
		h *= clamp01((aX - doorHW + 2) / 5)
		if aX > hw {
			h *= clamp01(1 - (aX-hw)/3)
		}
		below := clamp01((h-3)/2) * 2
		for U := -math.Round(below); U < math.Round(h); U++ {
			py := int(math.Floor(pyrAy - U - 0.5))
			l := 0.35 + 0.5*(U+below)/(h+below) + (hashf(px, py, 318)-0.5)*0.2
			if U >= math.Round(h)-1 {
				l += 0.15 // the sunlit crest
			}
			c.put(px, py, desertRamp.pick(l, px, py))
		}
	}

	c.outline(0.75)

	// The capstone glints in the sun.
	gx, gy := int(pyrAx)-3, int(pyrAy-(pyrSteps-1)*(pyrTread+pyrRise)-9)
	for d := -3; d <= 3; d++ {
		col := hex(0xfffce8)
		if d == -3 || d == 3 {
			col = hex(0xffe98a)
		}
		c.put(gx+d, gy, col)
		c.put(gx, gy+d, col)
	}
	c.put(gx-1, gy-1, hex(0xfff3b0))
	c.put(gx+1, gy+1, hex(0xfff3b0))
	c.put(gx+1, gy-1, hex(0xfff3b0))
	c.put(gx-1, gy+1, hex(0xfff3b0))
	return c, pyrAx, pyrAy
}

// --- the giant scorpion ---------------------------------------------------------

var (
	// Black chitin with a deep amber gleam.
	scorpRamp  = mkRamp(0x0b0709, 0x190d0e, 0x2a1411, 0x441d13, 0x682c15, 0x96461a, 0xcb7b2c, 0xf4c062)
	venomGreen = hex(0xa6ff5a)
)

// scorpClaw is one pincer: the hand at (x, y) reached by way of the elbow
// (ex, ey), pointing along ang with its fingers open radians apart.
type scorpClaw struct{ ex, ey, x, y, ang, open float64 }

// scorpPose is how a scorpion holds itself in one frame.
type scorpPose struct {
	dx, dy    float64       // the body's lunge and crouch
	tail      [6][2]float64 // five segments, root to tip, then the bulb of the sting
	sting     [2][2]float64 // the barb: its bend and its point
	near, far scorpClaw
	step      float64 // which legs are forward: 1 or -1
}

var scorpPoses = [scorpionPoses]scorpPose{
	scorpionWalkA: {
		tail:  [6][2]float64{{13.5, 18.5}, {9.5, 15.5}, {8, 10.8}, {10, 6.3}, {14.3, 4}, {19, 4.6}},
		sting: [2][2]float64{{23.2, 5.6}, {24, 10.4}},
		near:  scorpClaw{43, 19, 47.5, 22.5, 0.1, 0.55},
		far:   scorpClaw{41.5, 14.5, 46, 12.5, -0.3, 0.45},
		step:  1,
	},
	scorpionWalkB: {
		dy:    -1,
		tail:  [6][2]float64{{13.5, 17.5}, {9.8, 14}, {8.8, 9.3}, {11.2, 5}, {15.8, 3.2}, {20.5, 4.2}},
		sting: [2][2]float64{{24.8, 5.4}, {25.4, 10.2}},
		near:  scorpClaw{43, 18, 47.5, 21, 0, 0.3},
		far:   scorpClaw{41.5, 14, 46, 13, -0.2, 0.7},
		step:  -1,
	},
	scorpionPinch: {
		dx:    1.5,
		tail:  [6][2]float64{{14.5, 18}, {10, 15.5}, {7.5, 11.2}, {8.5, 6.5}, {12.5, 3.8}, {17.2, 3.8}},
		sting: [2][2]float64{{21.4, 4.6}, {22.6, 9.2}},
		near:  scorpClaw{44, 21.5, 49, 22, 0.05, 0.95},
		far:   scorpClaw{43, 15, 48.5, 12.5, -0.1, 0.1},
		step:  1,
	},
	scorpionSting: {
		dx: 0.5, dy: 0.5,
		tail:  [6][2]float64{{14.5, 18.5}, {12, 14}, {13.5, 9.2}, {17.8, 6}, {23, 4.5}, {28.3, 5.2}},
		sting: [2][2]float64{{32.8, 6.4}, {36, 11.6}},
		near:  scorpClaw{42.5, 21.5, 46.5, 25, 0.3, 0.6},
		far:   scorpClaw{42.5, 17.5, 47.5, 18, 0.1, 0.5},
		step:  -1,
	},
}

const (
	scorpW, scorpH = 60, 35
	scorpGy        = 32.0 // the line its feet stand on
	scorpAx        = 27.0
)

func scorpionCanvas(pose int) (*canvas, float64, float64) {
	p := scorpPoses[pose]
	c := newCanvas(scorpW, scorpH)
	ink := func(l float64) func(x, y int, t float64) color.RGBA {
		return func(x, y int, _ float64) color.RGBA { return scorpRamp.pick(l, x, y) }
	}
	bx, by := 24.5+p.dx, 20.5+p.dy // the middle of the trunk
	hx, hy := bx+10.5, by+0.8      // the head

	// A leg: out from under the body to a knee, then down to the foot.
	leg := func(i int, far bool) {
		fi := float64(i)
		hip := bx - 3 + fi*3.2
		swing := p.step * 1.4
		if i%2 == 1 {
			swing = -swing
		}
		sh, kx, fy := 0.0, 0.0, scorpGy-0.5
		if far {
			swing, sh, kx, fy = -swing, -0.25, 1.6, scorpGy-1.5
		}
		knee := [2]float64{hip + (fi-1.5)*3.4 + swing*0.5 + kx, by + 2.2 - p.dy*0.5}
		foot := [2]float64{hip + (fi-1.5)*6.6 + swing*2 + kx, fy}
		if swing > 0 {
			foot[1] -= 1 // lifted as it swings forward
		}
		for _, o := range []float64{0, 1} {
			c.line(hip+o, by+3.5, knee[0]+o, knee[1], ink(0.5+sh))
			c.line(knee[0]+o*0.7, knee[1], foot[0], foot[1], ink(0.62+sh-o*0.12))
		}
		c.set(int(knee[0]), int(knee[1]), scorpRamp.at(6+int(sh*8))) // the knee joint, lit
		c.set(int(foot[0]), int(foot[1]), scorpRamp.at(1))
	}

	// A pincer: the arm, the swollen hand and its two curved fingers.
	claw := func(k scorpClaw, bias float64) {
		k.ex, k.x = k.ex+p.dx, k.x+p.dx
		k.ey, k.y = k.ey+p.dy, k.y+p.dy
		tube(c, spine([3]float64{hx + 2, hy, 1.9}, [3]float64{k.ex, k.ey, 1.7}, [3]float64{k.x - 1.5, k.y, 1.8}), scorpRamp, bias-0.2)
		c.set(int(k.ex), int(k.ey-0.8), scorpRamp.at(5+int(bias*8)))
		finger := func(a, bulge, b float64) {
			ca, sa := math.Cos(a), math.Sin(a)
			tube(c, spine(
				[3]float64{k.x + 2.5*ca - 0.8*bulge*sa, k.y + 2.5*sa + 0.8*bulge*ca, 1.6},
				[3]float64{k.x + 6*ca - 1.4*bulge*sa, k.y + 6*sa + 1.4*bulge*ca, 1.2},
				[3]float64{k.x + 9.5*ca, k.y + 9.5*sa, 0.5}), scorpRamp, bias+b)
			c.set(int(k.x+9.5*ca), int(k.y+9.5*sa), scorpRamp.at(1))
		}
		finger(k.ang+k.open*0.4, 1, -0.12) // the fixed finger, below
		finger(k.ang-k.open*0.6, -1, 0.05) // the one that snaps, above
		roundShape(c, ell{k.x, k.y, 4.2, 3.3}, scorpRamp, bias-0.04)
		c.set(int(k.x-1.5), int(k.y-1.6), scorpRamp.at(7+int(bias*8))) // gloss
		c.set(int(k.x-0.5), int(k.y-1.6), scorpRamp.at(6+int(bias*8)))
	}

	for i := range 4 {
		leg(i, true)
	}
	claw(p.far, -0.22)

	// The tail: five beads of armour, each a little slimmer, and the bulb
	// that carries the sting.
	root := [2]float64{bx - 8.5, by - 0.5}
	c.line(root[0], root[1], p.tail[0][0], p.tail[0][1], ink(0.2))
	for i := 0; i+1 < len(p.tail); i++ {
		c.line(p.tail[i][0], p.tail[i][1], p.tail[i+1][0], p.tail[i+1][1], ink(0.2))
	}
	for i := 4; i >= 0; i-- {
		t := p.tail[i]
		r := 2.9 - float64(i)*0.13
		roundShape(c, ell{t[0], t[1], r, r}, scorpRamp, -0.16+float64(i)*0.03)
		c.set(int(t[0]-0.9), int(t[1]-1.4), scorpRamp.at(7)) // gloss
	}
	// The barb curves out of the bulb to a black point.
	b := p.tail[5]
	tube(c, spine([3]float64{b[0] + 1.5, b[1], 1.4}, [3]float64{p.sting[0][0], p.sting[0][1], 1.1}, [3]float64{p.sting[1][0], p.sting[1][1], 0.4}), scorpRamp, 0.05)
	c.set(int(p.sting[1][0]), int(p.sting[1][1]), scorpRamp.at(0))
	roundShape(c, ell{b[0], b[1], 3.1, 2.7}, scorpRamp, 0.06)
	c.set(int(b[0]-0.9), int(b[1]-1.4), scorpRamp.at(7))
	c.set(int(b[0]+0.1), int(b[1]-1.4), scorpRamp.at(6))

	// The trunk: one long armoured back, cut into plates, paler below.
	c.ellipse(bx, by, 9.6, 4.5, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx*0.6, ny)*0.8 - 0.26
		if ny > 0.5 {
			l += 0.22 // the soft underside
		}
		return scorpRamp.pick(l, x, y)
	})
	roundShape(c, ell{hx, hy, 4.4, 3.8}, scorpRamp, -0.2)
	for k := 0; k < 7; k++ {
		sx := int(bx-7.5) + k*3
		for y := 0; y < scorpH; y++ {
			u, v := (float64(sx)+0.5-bx)/9.6, (float64(y)+0.5-by)/4.5
			if u*u+v*v >= 1 {
				continue
			}
			if v < 0.5 {
				c.set(sx, y, scorpRamp.at(0)) // the seam between two plates
			}
			if v < -0.4 && v > -0.75 && c.opaque(sx+1, y) { // each plate shines along its top
				c.set(sx+1, y, scorpRamp.at(6))
				c.set(sx+2, y, scorpRamp.at(4))
			}
		}
	}
	// The eyes glow; the jaws work under them.
	c.set(int(hx+1), int(hy-2.3), hex(0xff5a1e))
	c.set(int(hx+2.5), int(hy-1.2), hex(0xffb03a))
	c.set(int(hx-1.5), int(hy-2.6), scorpRamp.at(6))
	c.set(int(hx-0.5), int(hy-2.6), scorpRamp.at(4))
	c.line(hx+3.2, hy+1.6, hx+4.8, hy+3, ink(0.7))

	for i := range 4 {
		leg(i, false)
	}
	claw(p.near, 0)

	c.outline(0.8)

	// Venom, bright against the outline.
	c.set(int(p.sting[1][0]), int(p.sting[1][1])+1, venomGreen)
	if pose == scorpionSting {
		c.set(int(p.sting[1][0])+1, int(p.sting[1][1])+2, venomGreen)
		c.set(int(p.sting[1][0])+2, int(p.sting[1][1])+5, venomGreen)
		c.set(int(p.sting[1][0])-1, int(p.sting[1][1])+4, hex(0xe6ffb0))
	}
	return c, scorpAx, scorpGy
}
