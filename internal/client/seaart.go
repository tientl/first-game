package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The open sea: the boats that cross it and the monsters that hunt them.
// Boats are drawn side-on in two layers, so whoever rides one stands between
// the far side and the near side of the hull. The monsters are only ever
// seen in part: a head, a back or an arm above the water, or a dark shape
// below it.

var (
	deckRamp   = mkRamp(0x3d2a1b, 0x5a3e25, 0x7a5632, 0x9a7244, 0xb98f5a, 0xd3ad76)
	hullRamp   = mkRamp(0x2c1c14, 0x45291a, 0x623b22, 0x82522d, 0xa16c3b, 0xbd8a50)
	sailRamp   = mkRamp(0x67492f, 0x8a6843, 0xab885c, 0xc8a87b, 0xe0c79c, 0xf1e1bd)
	mosaRamp   = mkRamp(0x0c1b25, 0x15303b, 0x1f4651, 0x2d6067, 0x437f80, 0x64a096)
	mosaBelly  = mkRamp(0x5f7069, 0x8a9a8a, 0xb5bfa5, 0xdcdfc2, 0xf3f1dc)
	krakenRamp = mkRamp(0x2e0b20, 0x4f122c, 0x7a1c36, 0xa8303f, 0xcf5549, 0xe98a66)
	krakenPale = mkRamp(0x8d4f5f, 0xc0808a, 0xe4b3ac, 0xf8dccf)

	colFoamDim    = hex(0xa9d6e2)
	colSailPaint  = hex(0x8e2f1c)
	colMosaShadow = hex(0x08131b)
	colKrakShadow = hex(0x170914)
	colGullet     = hex(0x3a0a18)
)

// Where things sit on a boat, in art pixels relative to the boat sprite's
// anchor, facing right. Mirror the x for a boat facing left.
const (
	raftSeatX, raftSeatY = -1, -5  // the rider's feet on the raft's deck
	sailSeatX, sailSeatY = -17, -3 // the rider's feet in the sailing boat, near the stern
	mastX, mastY         = 7, -4   // the foot of the mast in the sailing boat
)

// seaArt holds the sprites of the open sea. Boats and monsters face right;
// the caller mirrors them for left (mirrorSprite).
type seaArt struct {
	raftBack, raftFront sprite    // a log raft: back = deck and far side, front = the near side of the hull
	sailBack, sailFront sprite    // a sailing boat, in the same two layers
	sail                [3]sprite // the sail on its mast: 0 slack, 1 half full, 2 full of wind
	mosaRise            sprite    // the Mosasaurus' head and neck out of the water, jaws shut
	mosaBite            sprite    // the same lunging, jaws wide
	mosaBack            sprite    // its back and tail breaking the surface
	mosaShadow          sprite    // its whole shape under the water, seen from above
	krakenArm           [3]sprite // one arm of the giant squid out of the water, curling
	krakenHead          sprite    // the squid's mantle and eye above the water
	krakenShadow        sprite    // its shape under the water, seen from above
}

func genSeaArt() seaArt {
	var a seaArt
	rb, rf := raftCanvases()
	a.raftBack, a.raftFront = mk(rb, raftAX, raftWY), mk(rf, raftAX, raftWY)
	sb, sf := sailboatCanvases()
	a.sailBack, a.sailFront = mk(sb, boatAX, boatWY), mk(sf, boatAX, boatWY)
	for i := range a.sail {
		a.sail[i] = mk(sailCanvas(i), sailMastX, sailFootY)
	}
	a.mosaRise = mk(mosaHeadCanvas(false), mosaBaseX, mosaWY)
	a.mosaBite = mk(mosaHeadCanvas(true), mosaBaseX, mosaWY)
	a.mosaBack = mk(mosaBackCanvas(), 43, mosaBackWY)
	a.mosaShadow = mk(mosaShadowCanvas(), 56, 22)
	for i := range a.krakenArm {
		a.krakenArm[i] = mk(krakenArmCanvas(i), krakArmX, krakArmWY)
	}
	a.krakenHead = mk(krakenHeadCanvas(), 27, krakHeadWY)
	a.krakenShadow = mk(krakenShadowCanvas(), 48, 32)
	return a
}

// --- water ------------------------------------------------------------------

// sinkBelow clears everything under row wy: the part of a thing that is
// under water is not drawn.
func sinkBelow(c *canvas, wy int) {
	for y := wy; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.put(x, y, color.RGBA{})
		}
	}
}

// seaFoam breaks the water along row wy from x0 to x1: a ragged line of foam
// with flecks above and below it.
func seaFoam(c *canvas, x0, x1, wy int, seed uint32) {
	for x := x0; x <= x1; x++ {
		edge := min(x-x0, x1-x)
		switch h := hashf(x, wy, seed); {
		case h < 0.5:
			c.put(x, wy, colFoam)
		case h < 0.85 || edge > 1:
			c.put(x, wy, colFoamDim)
		}
		if edge >= 2 && hashf(x, wy-1, seed+1) < 0.35 {
			c.put(x, wy-1, colFoam)
		}
		if edge >= 1 && hashf(x, wy+1, seed+2) < 0.3 {
			c.put(x, wy+1, withAlpha(colFoamDim, 210))
		}
	}
}

// seaRing is the churned water around something big coming up through the
// surface at (cx, wy): foam along the waterline and a ring of it on the
// near side, with drops thrown up.
func seaRing(c *canvas, cx float64, wy int, rx, ry float64, seed uint32) {
	seaFoam(c, int(cx-rx*0.8), int(cx+rx*0.8), wy, seed)
	for x := int(cx - rx); x <= int(cx+rx); x++ {
		nx := (float64(x) + 0.5 - cx) / rx
		if nx*nx >= 1 {
			continue
		}
		y := wy + int(math.Round(ry*math.Sqrt(1-nx*nx)))
		switch h := hashf(x, y, seed+7); {
		case h < 0.55:
			c.put(x, y, colFoam)
		case h < 0.85:
			c.put(x, y, colFoamDim)
		}
		// Thin foam on the water inside the ring.
		for yy := wy + 1; yy < y; yy++ {
			if hashf(x, yy, seed+8) < 0.22 {
				c.put(x, yy, withAlpha(colFoamDim, 200))
			}
		}
	}
	// Drops thrown up at either side.
	for i := 0; i < 7; i++ {
		side := float64(i%2*2 - 1)
		dx := rx * (0.75 + 0.5*hashf(i, 1, seed+9))
		dy := 2 + 6*hashf(i, 2, seed+9)
		col := colFoam
		if i%3 == 0 {
			col = colFoamDim
		}
		c.put(int(cx+side*dx), wy-int(dy), col)
	}
}

// --- the raft ---------------------------------------------------------------

const raftAX, raftWY = 27.0, 19.0

// raftLog draws one log lying along the raft from x0 to x1, its top at row y
// and h rows thick: round, barked, its sawn end pale at the bow.
func raftLog(c *canvas, x0, x1, y, h int, tone float64) {
	c.rect(x0, y, x1-x0, h, func(x, yy int) color.RGBA {
		v := (float64(yy-y) + 0.5) / float64(h)
		if x >= x1-2 {
			return woodRamp.pick(0.9-0.7*v, x, yy)
		}
		l := 0.95 - 0.78*v + tone
		if hashf(x/3, yy, 72) < 0.16 {
			l -= 0.13 // furrows in the bark
		}
		if x == x0 {
			l -= 0.2
		}
		return deckRamp.pick(l, x, yy)
	})
}

// raftCanvases draws the raft: logs laid side by side along it, their ends
// uneven, lashed to two poles across. The back layer is the deck, with a
// steering oar over the stern and a bundle at the bow; the front layer is
// the nearest log, which hides the rider's feet.
func raftCanvases() (back, front *canvas) {
	back, front = newCanvas(54, 26), newCanvas(54, 26)
	const wy = int(raftWY)

	raftLog(back, 10, 44, 6, 3, -0.05)
	raftLog(back, 8, 47, 9, 3, 0.03)
	raftLog(back, 11, 45, 12, 3, -0.02)
	raftLog(front, 7, 48, 14, 6, 0)
	for _, px := range []int{15, 37} {
		// A pole across the logs, lashed down at every one.
		back.rect(px, 5, 2, 10, func(x, y int) color.RGBA {
			if y%3 == 2 {
				return ropeRamp.at(2 - (x - px))
			}
			return deckRamp.at(5 - (x-px)*2)
		})
		front.rect(px-1, 14, 4, 6, func(x, y int) color.RGBA {
			if (x+y)%3 == 0 {
				return ropeRamp[0]
			}
			return ropeRamp.pick(0.9-float64(y-14)*0.16, x, y)
		})
	}
	// The steering oar, trailing in the water behind.
	back.line(17, 6, 5, 15, func(x, y int, t float64) color.RGBA { return woodRamp.at(2 - int(t*2.5)) })
	back.polygon([]float64{6.5, 14, 4, 13.5, 0.5, 18, 1, 21, 4, 21}, func(x, y int) color.RGBA {
		return woodRamp.pick(0.75-float64(y-13)*0.08, x, y)
	})
	// A bundle of supplies wrapped in hide at the bow, tied with rope.
	back.ellipse(41.5, 7, 3.6, 3.2, func(x, y int, nx, ny float64) color.RGBA {
		if x == 41 || (y == 7 && x > 39) {
			return ropeRamp.pick(0.3+lum(nx, ny)*0.7, x, y)
		}
		return hideRamp.pick(lum(nx, ny)*0.95, x, y)
	})

	back.outline(0.78)
	front.outline(0.78)
	sinkBelow(back, wy)
	sinkBelow(front, wy)
	seaFoam(front, 5, 49, wy, 301)
	seaFoam(back, 0, 4, wy, 305)
	return back, front
}

// --- the sailing boat -------------------------------------------------------

const (
	boatAX, boatWY = 39.0, 24.0
	boatHalf       = 33.0 // half the hull's length
)

// boatSheer is the height of the gunwale at column x: low amidships, swept
// up to the stern and higher still to the prow.
func boatSheer(x float64) float64 {
	u := (x - boatAX) / boatHalf
	rise := 5.0
	if u > 0 {
		rise = 7.5
	}
	return boatWY - 7 - rise*math.Pow(math.Abs(u), 2.4)
}

// boatKeel is the bottom of the hull at column x, curving up to both ends.
func boatKeel(x float64) float64 {
	u := (x - boatAX) / boatHalf
	return boatWY + 3 - 17.5*math.Pow(math.Abs(u), 3.2)
}

// sailboatCanvases draws the sailing boat: a long dugout built up with
// stitched planks, its ends swept up, a horned prow, and an outrigger float
// on the near side. The back layer is the inside of the far wall; the front
// layer is the near wall, the outrigger and its booms.
func sailboatCanvases() (back, front *canvas) {
	back, front = newCanvas(78, 36), newCanvas(78, 36)
	const wy = int(boatWY)

	// The far wall, seen from inside over the near gunwale, and the thwarts.
	for x := 0; x < back.w; x++ {
		fx := float64(x) + 0.5
		top, bot := boatSheer(fx)-3, boatKeel(fx)
		for y := int(math.Round(top)); float64(y) < bot; y++ {
			l := 0.3 + 0.07*float64(y-int(math.Round(top)))
			if y == int(math.Round(top)) {
				l = 0.78 // the far rail
			}
			if x%9 == 4 && y > int(math.Round(top)) {
				l -= 0.12 // ribs
			}
			back.put(x, y, hullRamp.pick(l-(fx-boatAX)/boatHalf*0.06, x, y))
		}
	}
	// The prow post: the stem carried up into a carved horn.
	back.polygon([]float64{70.5, 11, 73, 5.5, 77.5, 2.5, 75.5, 7.5, 73.5, 12.5}, func(x, y int) color.RGBA {
		return hornRamp.pick(0.85-float64(y)*0.06, x, y)
	})
	back.polygon([]float64{68, 13, 71, 10, 74, 12, 72, 16}, func(x, y int) color.RGBA { return hullRamp.pick(0.6, x, y) })
	for _, tx := range []int{24, 52} { // thwarts
		y := int(math.Round(boatSheer(float64(tx)))) - 1
		back.rect(tx, y, 3, 2, func(x, yy int) color.RGBA { return deckRamp.at(4 - (yy - y)) })
	}

	// The near wall.
	for x := 0; x < front.w; x++ {
		fx := float64(x) + 0.5
		top, bot := boatSheer(fx), boatKeel(fx)
		ty := int(math.Round(top))
		for y := ty; float64(y) < bot; y++ {
			d := y - ty
			v := (float64(y) - top) / math.Max(1, bot-top)
			l := 0.72 - 0.42*v - (fx-boatAX)/boatHalf*0.07
			l += (vnoise(fx/5, float64(y)*1.3, 311) - 0.5) * 0.2 // grain along the hull
			switch {
			case d == 0:
				l = 0.95 // the rail
			case d == 1:
				l = 0.42
			case d == 4:
				l -= 0.2 // the seam between plank and dugout
			}
			col := hullRamp.pick(l, x, y)
			if d >= 3 && d <= 5 && x%7 == 3 {
				col = ropeRamp.at(5 - d) // stitches across the seam
			}
			front.put(x, y, col)
		}
	}
	// A painted eye on the bow, to watch for what swims below.
	ex, ey := 62, int(math.Round(boatSheer(62.5)))+2
	for _, p := range [][2]int{{0, 0}, {1, 0}, {2, 0}, {-1, 1}, {3, 1}, {0, 2}, {1, 2}, {2, 2}} {
		front.put(ex+p[0], ey+p[1], hex(0xe9dfc6))
	}
	front.put(ex+1, ey+1, colSailPaint)
	front.put(ex, ey+1, hex(0xe9dfc6))
	front.put(ex+2, ey+1, hex(0xe9dfc6))

	front.outline(0.78)
	back.outline(0.78)
	sinkBelow(front, wy)
	sinkBelow(back, wy)
	seaFoam(front, 11, 67, wy, 302)
	// The bow wave.
	for _, p := range [][2]int{{68, -1}, {69, -2}, {70, -1}, {67, -2}} {
		front.put(p[0], wy+p[1], colFoam)
	}

	// The outrigger: a float riding on the near side, on two booms lashed
	// to the gunwale. Drawn on its own so its outline lies over the hull.
	out := newCanvas(front.w, front.h)
	const fy = wy + 6 // the float's top row
	for _, bx := range []float64{25, 53} {
		gy := boatSheer(bx)
		out.line(bx, gy-1, bx-2.5, float64(fy)+0.5, func(x, y int, t float64) color.RGBA {
			if t < 0.14 {
				return ropeRamp[2]
			}
			return deckRamp.pick(0.75-t*0.35, x, y)
		})
	}
	out.polygon([]float64{12, float64(fy) + 1.5, 16, float64(fy), 58, float64(fy), 63, float64(fy) + 1, 58, float64(fy) + 3, 16, float64(fy) + 3},
		func(x, y int) color.RGBA {
			l := 0.85 - float64(y-fy)*0.25 + (hashf(x/3, y, 312)-0.5)*0.12
			if x == 22 || x == 23 || x == 50 || x == 51 {
				return ropeRamp.at(2 - (y - fy)) // lashings
			}
			return deckRamp.pick(l, x, y)
		})
	out.outline(0.78)
	sinkBelow(out, fy+3)
	front.over(out, 0, 0)
	seaFoam(front, 13, 61, fy+3, 303)
	return back, front
}

// --- the sail ---------------------------------------------------------------

const sailMastX, sailFootY = 13.0, 50.0

// sailCanvas draws the mast and its hide sail, hung from a slanting yard and
// laced to a boom: fill 0 hangs slack in folds, 1 bellies, 2 is full of
// wind and straining forward.
func sailCanvas(fill int) *canvas {
	c := newCanvas(42, 52)
	belly := float64(fill) * 3.2
	sag := 0.0
	if fill == 0 {
		sag = 1
	}
	// The yard and the boom, as lines y(x).
	const yx0, yx1, bx0, bx1 = 3.0, 33.0, 4.0, 31.0
	yardY := func(x float64) float64 { return lerpf(13, 5, (x-yx0)/(yx1-yx0)) + sag*2*((x-yx0)/(yx1-yx0)) }
	boomY := func(x float64) float64 { return lerpf(40, 37, (x-bx0)/(bx1-bx0)) + sag*1.5 }

	// The hide between them, bowed forward by the wind.
	const vy0, vy1 = 8.0, 40.0
	for y := 0; y < c.h; y++ {
		fy := float64(y) + 0.5
		v := clamp01((fy - vy0) / (vy1 - vy0))
		bow := math.Sin(v * math.Pi)
		left := lerpf(yx0+1, bx0+1, v) + belly*0.55*bow + sag*2.6*bow
		right := lerpf(yx1-1, bx1-1, v) + belly*bow - sag*3.6*bow
		for x := int(math.Floor(left)); x <= int(math.Ceil(right)); x++ {
			fx := float64(x) + 0.5
			if fx < left || fx > right {
				continue
			}
			// Follow the bow back to where this pixel hangs on the yard.
			u := (fx - left) / (right - left)
			hx := lerpf(yx0+1, yx1-1, u)
			top, bot := yardY(hx)+1, boomY(lerpf(bx0+1, bx1-1, u))-1
			if fy < top || fy > bot {
				continue
			}
			vv := (fy - top) / (bot - top)
			l := 0.8 - 0.22*vv - 0.2*u
			// The swell of a full sail catches the light; a slack one hangs in folds.
			l += belly / 6.4 * 0.22 * math.Cos((u-0.42)*math.Pi*1.3) * math.Sin(vv*math.Pi)
			if sag > 0 {
				l += 0.21 * math.Sin(u*15+vv*3+math.Sin(vv*5))
				l -= 0.08
			}
			l += (vnoise(fx/3, fy/3, 321) - 0.5) * 0.14 // the mottle of hide
			col := sailRamp.pick(l, x, y)
			// Hides stitched together: seams across, and one down.
			seam := 0.34 + 0.03*math.Sin(u*9)
			seam2 := 0.68 + 0.03*math.Cos(u*7)
			pw := bot - top
			if math.Abs(vv-seam)*pw < 0.5 || math.Abs(vv-seam2)*pw < 0.5 {
				col = sailRamp.at(1 + x%2)
			}
			if vv > seam && vv < seam2 && math.Abs(u-0.3)*(right-left) < 0.5 {
				col = sailRamp.at(1 + y%2)
			}
			// A sun painted in red ochre.
			dx, dy := (u-0.6)*(right-left), (vv-0.5)*pw
			if d := math.Hypot(dx, dy*0.95); (d > 3.2 && d < 4.4) || d < 1.3 {
				col = mix(colSailPaint, col, 0.25)
			}
			c.put(x, y, col)
		}
	}
	// Lacing along yard and boom.
	spar := func(x0, x1 float64, yf func(float64) float64) {
		for x := int(x0); x <= int(x1); x++ {
			y := int(math.Floor(yf(float64(x) + 0.5)))
			col := deckRamp[4]
			if x%4 == 0 {
				col = ropeRamp[2]
			}
			c.put(x, y, col)
			c.put(x, y+1, deckRamp[2])
		}
	}
	spar(yx0, yx1, yardY)
	spar(bx0, bx1, boomY)
	// The mast, in front of the sail, and a streamer at its head.
	c.rect(int(sailMastX)-1, 4, 2, int(sailFootY)-4, func(x, y int) color.RGBA {
		if x == int(sailMastX)-1 {
			return deckRamp.pick(0.8-float64(y)*0.004, x, y)
		}
		return deckRamp.pick(0.4, x, y)
	})
	for _, y := range []int{int(yardY(sailMastX)) + 1, int(boomY(sailMastX)) + 1} {
		c.rect(int(sailMastX)-2, y-1, 4, 2, func(x, yy int) color.RGBA { return ropeRamp.at(2 - (yy - y + 1) - (x+yy)%2) })
	}
	stream := [3][][2]int{
		{{-2, 5}, {-3, 6}, {-3, 7}, {-4, 8}},
		{{-2, 4}, {-3, 5}, {-4, 5}, {-5, 6}},
		{{-2, 3}, {-3, 3}, {-4, 4}, {-5, 4}, {-6, 3}},
	}[fill]
	for i, p := range stream {
		col := colSailPaint
		if i == len(stream)-1 {
			col = hex(0xc8502e)
		}
		c.put(int(sailMastX)+p[0], p[1], col)
	}
	c.outline(0.78)
	return c
}

// --- the Mosasaurus ---------------------------------------------------------

const (
	mosaBaseX, mosaWY = 24.0, 48.0
	mosaBackWY        = 15.0
)

// mosaSkin colours one pixel of the beast: l is the light on it, belly how
// far (0..1) it lies towards the pale underside.
func mosaSkin(x, y int, l, belly float64) color.RGBA {
	l += (hashf(x, y, 331) - 0.5) * 0.1
	// Scales: staggered cells with dark seams.
	row := y / 2
	if (x+row%2)%3 == 0 && y%2 == 0 {
		l -= 0.07
	}
	if belly > 0.35+dither(x, y)*0.3 {
		return mosaBelly.pick(l, x, y)
	}
	// Dark blotches down the back.
	if fbm2(float64(x)/5, float64(y)/5, 332, 2) > 0.6 {
		l -= 0.16
	}
	return mosaRamp.pick(l, x, y)
}

// mosaHeadCanvas draws the head and neck of the Mosasaurus out of the water:
// reared with its jaws shut, or lunging with them wide.
func mosaHeadCanvas(bite bool) *canvas {
	c := newCanvas(76, 56)
	const wy = int(mosaWY)
	bx := mosaBaseX

	neck := spine([3]float64{bx, mosaWY + 4, 8.5}, [3]float64{bx + 1.5, 36, 8}, [3]float64{bx + 5, 25, 7.4}, [3]float64{bx + 10, 18, 6.8})
	px, py, ang, gape := bx+12.5, 17.5, 0.14, 0.0
	if bite {
		neck = spine([3]float64{bx, mosaWY + 4, 8.5}, [3]float64{bx + 5, 37, 8}, [3]float64{bx + 12, 28, 7.4}, [3]float64{bx + 19, 23.5, 6.8})
		px, py, ang, gape = bx+21.5, 23.5, 0.1, 1.0
	}

	// The back, awash behind the neck.
	c.ellipse(bx-10, mosaWY+1, 11, 5, func(x, y int, nx, ny float64) color.RGBA {
		return mosaSkin(x, y, lum(nx, ny)*0.85, 0)
	})
	// A ridge of scutes down the back of the neck.
	for i := 6; i < len(neck)-4; i += 9 {
		p := neck[i]
		ox, oy := p.x-p.nx*(p.r-0.5), p.y-p.ny*(p.r-0.5)
		c.polygon([]float64{ox + p.ny*1.8, oy - p.nx*1.8, ox - p.nx*2.6 - p.ny*0.6, oy - p.ny*2.6 + p.nx*0.6, ox - p.ny*1.8, oy + p.nx*1.8},
			func(x, y int) color.RGBA { return mosaRamp.pick(0.3, x, y) })
	}
	drawTubeWith(c, neck, func(x, y int, p *spinePt, t float64) color.RGBA {
		l := lum(p.nx*t*0.85, p.ny*t*0.85)
		if t < 0.15 && math.Sin(p.s*30+t*1.5) > 0.45 {
			l -= 0.17 // bands across the back
		}
		if t > 0.3 {
			l = 0.85 - (t-0.3)*0.75 // the throat: pale, falling into shade underneath
			if int(p.s*46)%3 == 0 {
				l -= 0.1 // the folds of the throat
			}
		}
		return mosaSkin(x, y, l, clamp01((t-0.05)*2.2))
	})

	// The head, in its own frame: u along the snout, v down, the origin at
	// the hinge of the jaw. Each jaw turns about the hinge.
	upA, loA := ang-gape*0.42, ang+gape*0.56
	place := func(a, u, v float64) (float64, float64) {
		dx, dy := rotPt(a, u, v)
		return px + dx, py + dy
	}
	local := func(a float64, x, y int) (float64, float64) {
		return rotPt(-a, float64(x)+0.5-px, float64(y)+0.5-py)
	}
	poly := func(a float64, q [][2]float64, f func(x, y int) color.RGBA) {
		var pts []float64
		for _, v := range q {
			x, y := place(a, v[0], v[1])
			pts = append(pts, x, y)
		}
		c.polygon(pts, f)
	}
	if gape > 0 {
		// The inside of the mouth: red, falling to black down the gullet.
		var pts []float64
		for _, v := range [][3]float64{{upA, -7, -1}, {upA, 25.5, 0.2}, {(upA + loA) / 2, 11, 0.5}, {loA, 24.5, 0.4}, {loA, -7, 3}} {
			x, y := place(v[0], v[1], v[2])
			pts = append(pts, x, y)
		}
		c.polygon(pts, func(x, y int) color.RGBA {
			d := math.Hypot(float64(x)+0.5-px, float64(y)+0.5-py)
			return mix(colGullet, colMouth, clamp01((d-6)/13)-dither(x, y)*0.2)
		})
		// The roof of the mouth, ridged.
		poly(upA, [][2]float64{{2, 0.5}, {25, 0.5}, {23, 2.6}, {6, 3.2}}, func(x, y int) color.RGBA {
			u, _ := local(upA, x, y)
			if int(u)%3 == 0 {
				return hex(0x8c3440)
			}
			return hex(0xc96a6a)
		})
	}
	// The lower jaw: pale, deep at the hinge, its chin in shade.
	poly(loA, [][2]float64{{-8.5, -1}, {-3, 0.4}, {25, 0.2}, {25.2, 1.6}, {17, 3.8}, {7, 5.4}, {-2, 6.6}, {-8.5, 5.5}}, func(x, y int) color.RGBA {
		u, v := local(loA, x, y)
		if gape > 0 && v < 1.3 {
			return mix(colMouth, hex(0xe08a88), 0.4) // the gum
		}
		if v < 2.1 && u > -2 {
			return mosaSkin(x, y, 0.42-v*0.06, 0) // the dark lip the teeth close over
		}
		return mosaSkin(x, y, 0.95-v*0.1, clamp01(0.55+v*0.2+u*0.012))
	})
	if gape > 0 {
		// The tongue, forked.
		poly(loA, [][2]float64{{1, 0.4}, {13, -1.9}, {19, -1.6}, {22.5, -2.6}, {20, -0.8}, {22, 0.1}, {16, 0.4}}, func(x, y int) color.RGBA {
			_, v := local(loA, x, y)
			return mix(hex(0xe58a94), hex(0xa83a52), clamp01((v+1.5)/2))
		})
	}
	// The skull: long and wedge-shaped, a heavy brow over the eye.
	poly(upA, [][2]float64{{-9, -2}, {-7.5, -6.8}, {-1, -8.6}, {5, -8.2}, {8, -6.4}, {14, -5.4}, {21, -3.6}, {25.5, -2.6}, {27, -1}, {26.4, 0.7}, {-3, 1}, {-9, 1.5}}, func(x, y int) color.RGBA {
		u, v := local(upA, x, y)
		l := 0.84 - (v+8.6)*0.062
		if u > 6 && u < 22 && v > -4.2 && v < -3.2 {
			l -= 0.14 // the furrow along the snout
		}
		if v > 0 && u > -2 {
			return mosaRamp[0] // the line of the mouth
		}
		return mosaSkin(x, y, l, 0)
	})
	// Teeth. Shut, they interlock outside the lips; open, they stand in rows.
	tooth := func(a, u, v, length float64, col color.RGBA) {
		x0, y0 := place(a, u, v)
		x1, y1 := place(a, u+0.3, v+length)
		c.line(x0, y0, x1, y1, func(x, y int, t float64) color.RGBA {
			if t > 0.75 {
				return mix(col, hornRamp[1], 0.5)
			}
			return col
		})
	}
	if gape > 0 {
		for u := 3.0; u < 26; u += 2.4 {
			n := 3.4 - math.Abs(u-15)*0.1
			tooth(upA, u, 0.9, n, colTooth)
			tooth(upA, u+1, 0.9, 1.2, hornRamp[2])
		}
		for u := 3.5; u < 24.5; u += 2.4 {
			n := 3.0 - math.Abs(u-14)*0.09
			tooth(loA, u, 0.2, -n, colTooth)
			tooth(loA, u+1, 0.2, -1.1, hornRamp[2])
		}
	} else {
		for u := 2.5; u < 26; u += 2.5 {
			tooth(upA, u, 0.4, 2.3, colTooth)
			tooth(loA, u+1.25, 0.9, -1.4, hornRamp[2])
		}
	}
	// The eye: small, yellow and slit, sunk under the brow.
	ex, ey := place(upA, 3.6, -4.9)
	ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
	for dy := -2; dy <= 1; dy++ {
		for dx := -3; dx <= 2; dx++ {
			c.put(ix+dx, iy+dy, mosaRamp[0])
		}
	}
	for dy := -1; dy <= 0; dy++ {
		for dx := -2; dx <= 1; dx++ {
			col := hex(0xffd23c)
			switch {
			case dx == 0:
				col = colInk // the slit
			case dy == -1:
				col = hex(0xf08a1e)
			}
			c.put(ix+dx, iy+dy, col)
		}
	}
	bx0, by0 := place(upA, -0.5, -7.6)
	bx1, by1 := place(upA, 7.5, -5.6)
	c.line(bx0, by0, bx1, by1, func(x, y int, _ float64) color.RGBA { return mosaRamp[1] }) // the brow
	nx, ny := place(upA, 23.5, -2.4)
	c.put(int(nx), int(ny), mosaRamp[0]) // the nostril

	c.outline(0.78)
	sinkBelow(c, wy)
	// Water streaming off the jaw and the throat.
	for i := 0; i < 5; i++ {
		u := 5 + float64(i)*4.3
		x, y := place(loA, u, 6.2-u*0.18)
		n := 2 + int(hashf(i, 0, 333)*3)
		for k := 0; k < n; k++ {
			yy := int(y) + 1 + k + int(hashf(i, 1, 333)*2)
			if yy < wy-1 && !c.opaque(int(x), yy) {
				c.put(int(x), yy, withAlpha(mix(colFoamDim, colFoam, float64(k%2)), 220))
			}
		}
	}
	seaRing(c, bx-3, wy, 19, 3.4, 334)
	return c
}

// mosaBackCanvas draws the Mosasaurus swimming just under: the long ridge of
// its back and the blade of its tail cutting the surface.
func mosaBackCanvas() *canvas {
	c := newCanvas(86, 20)
	const wy = int(mosaBackWY)
	// The tail fluke, well behind.
	c.polygon([]float64{2, mosaBackWY + 1, 5.5, 3.5, 8, 3, 10, 8, 16, mosaBackWY + 1}, func(x, y int) color.RGBA {
		return mosaSkin(x, y, 0.72-float64(y)*0.03-float64(x)*0.02, 0)
	})
	// The root of the tail and the back.
	c.ellipse(27, mosaBackWY+1, 11, 4.2, func(x, y int, nx, ny float64) color.RGBA {
		return mosaSkin(x, y, lum(nx, ny)*0.9, 0)
	})
	for x := 44; x < 80; x += 4 { // scutes along the spine
		u := (float64(x) - 61) / 21
		top := mosaBackWY + 1 - 9.5*math.Sqrt(math.Max(0, 1-u*u))
		c.polygon([]float64{float64(x) - 1.5, top + 1.5, float64(x) - 0.5, top - 2.2, float64(x) + 2.5, top + 1.5}, func(x, y int) color.RGBA {
			return mosaRamp.pick(0.35, x, y)
		})
	}
	c.ellipse(61, mosaBackWY+1, 21, 9.5, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny) * 0.92
		if math.Sin(float64(x)*0.75+ny*1.2) > 0.5 {
			l -= 0.16 // bands
		}
		return mosaSkin(x, y, l, 0)
	})
	// Wet light along the top of the back.
	for x := 48; x < 70; x++ {
		u := (float64(x) + 0.5 - 61) / 21
		y := int(mosaBackWY + 1 - 9.5*math.Sqrt(1-u*u)*0.72)
		if hashf(x, 0, 341) < 0.6 {
			c.put(x, y, mosaRamp[5])
		}
	}
	c.outline(0.78)
	sinkBelow(c, wy)
	seaFoam(c, 1, 17, wy, 342)
	seaFoam(c, 17, 37, wy, 343)
	seaFoam(c, 40, 83, wy, 344)
	// The wave pushed up ahead, and the wake behind.
	for _, p := range [][2]int{{83, -1}, {84, 0}, {82, -2}, {81, -1}, {39, -1}, {0, 0}, {18, -1}} {
		c.put(p[0], wy+p[1], colFoam)
	}
	for x := 2; x < 60; x += 3 {
		if hashf(x, 3, 345) < 0.6 {
			c.put(x, wy+2+int(hashf(x, 4, 345)*2), withAlpha(colFoamDim, 200))
		}
	}
	return c
}

// softenShadow thins the rim of a flat silhouette so it melts into the water.
func softenShadow(c *canvas) {
	src := make([]color.RGBA, len(c.px))
	copy(src, c.px)
	solidAt := func(x, y int) bool { return c.in(x, y) && src[y*c.w+x].A > 0 }
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if !solidAt(x, y) {
				continue
			}
			n := 0
			for _, d := range [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
				if solidAt(x+d[0], y+d[1]) {
					n++
				}
			}
			if n < 4 {
				c.px[y*c.w+x].A = 150
			}
		}
	}
}

// mosaShadowCanvas draws the Mosasaurus from above, as a shape in the deep:
// a narrow head, four paddles and a long tail, all one dark colour.
func mosaShadowCanvas() *canvas {
	c := newCanvas(112, 44)
	paddle := func(ctrl ...[3]float64) {
		drawTubeWith(c, spine(ctrl...), func(int, int, *spinePt, float64) color.RGBA { return colMosaShadow })
	}
	// Paddles: the fore pair long, the hind pair short, all swept back.
	for _, side := range []float64{-1, 1} {
		paddle([3]float64{76, 22 + side*4, 2.4}, [3]float64{72, 22 + side*10, 3}, [3]float64{66.5, 22 + side*14.5, 1.2})
		paddle([3]float64{48, 23 + side*3.5, 2}, [3]float64{45, 23 + side*8.5, 2.4}, [3]float64{40.5, 23 + side*11.5, 1})
	}
	// The body, bending as it swims, the tail broadening into its fin.
	body := spine(
		[3]float64{2, 27, 0.7}, [3]float64{8, 25.6, 2.5}, [3]float64{15, 23.6, 1.7}, [3]float64{30, 20.5, 2.7}, [3]float64{44, 22.5, 4.4},
		[3]float64{58, 23, 6.2}, [3]float64{70, 22, 6}, [3]float64{80, 21.5, 4.2}, [3]float64{87, 21.8, 3.8},
		[3]float64{93, 22, 4.4}, [3]float64{101, 22.6, 2.8}, [3]float64{109, 23, 1.2})
	drawTubeWith(c, body, func(int, int, *spinePt, float64) color.RGBA { return colMosaShadow })
	softenShadow(c)
	return c
}

// --- the giant squid --------------------------------------------------------

const (
	krakArmX, krakArmWY = 20.0, 50.0
	krakHeadWY          = 50.0
)

// krakenSkin colours the squid's hide: wet red, freckled.
func krakenSkin(x, y int, l float64) color.RGBA {
	switch h := hashf(x, y, 351); {
	case h < 0.07:
		l += 0.2
	case h < 0.16:
		l -= 0.14
	}
	l += (fbm2(float64(x)/6, float64(y)/6, 352, 2) - 0.5) * 0.2
	return krakenRamp.pick(l, x, y)
}

// krakenTube draws an arm along pts: red above, pale with two rows of
// suckers on the side it curls towards. every is the spacing of the suckers
// in samples.
func krakenTube(c *canvas, pts []spinePt, every int) {
	n := float64(len(pts) - 1)
	drawTubeWith(c, pts, func(x, y int, p *spinePt, t float64) color.RGBA {
		l := lum(p.nx*t*0.85, p.ny*t*0.85)
		if t <= 0.2 || p.r < 1.2 {
			return krakenSkin(x, y, l*0.95+0.05)
		}
		i := int(p.s*n + 0.5)
		if k := i % every; k < every/2 && t > 0.35 {
			if k > 0 && k < every/2-1 && t > 0.5 && t < 0.85 && p.r > 2.4 {
				return krakenRamp[2] // the cup of the sucker
			}
			return krakenPale[3]
		}
		return krakenPale.pick(0.75-(t-0.2)*0.7, x, y)
	})
}

// krakenArmCanvas draws one arm reared out of the water: frame 0 coiled
// tight, 1 unrolling, 2 thrown forward to strike.
func krakenArmCanvas(frame int) *canvas {
	c := newCanvas(62, 58)
	const wy = int(krakArmWY)
	lean := [3]float64{-0.3, -0.08, 0.2}[frame]
	sway := [3]float64{0.022, 0.012, 0.02}[frame]
	curl := [3]float64{0.42, 0.3, 0.17}[frame]
	from := [3]float64{0.52, 0.56, 0.5}[frame]
	const steps, step = 150, 0.45
	x, y, h := krakArmX, krakArmWY+4, -math.Pi/2+lean
	pts := make([]spinePt, 0, steps+1)
	for i := 0; i <= steps; i++ {
		s := float64(i) / steps
		pts = append(pts, spinePt{x: x, y: y, r: 1.0 + 4.6*math.Pow(1-s, 0.85), s: s})
		k := sway*math.Sin(s*math.Pi*1.1+0.3) + curl*smooth(clamp01((s-from)/(1-from)))
		h += k * step
		x, y = x+math.Cos(h)*step, y+math.Sin(h)*step
	}
	krakenTube(c, withNormals(pts), 10)
	c.outline(0.78)
	sinkBelow(c, wy)
	// Water running off it.
	for i := 0; i < 4; i++ {
		p := pts[18+i*17]
		xx, yy := int(p.x-p.nx*(p.r+1.5)), int(p.y)+i%2*2
		if yy < wy-2 && !c.opaque(xx, yy) && !c.opaque(xx, yy+1) {
			c.put(xx, yy, colFoam)
			c.put(xx, yy+1, withAlpha(colFoamDim, 220))
		}
	}
	seaRing(c, krakArmX, wy, 9.5, 2.6, 353+uint32(frame))
	return c
}

// krakenHeadCanvas draws the squid risen to look: the tall mantle with its
// fins, and one huge eye just clear of the water.
func krakenHeadCanvas() *canvas {
	c := newCanvas(56, 58)
	const wy = int(krakHeadWY)
	// The fins at the tip of the mantle, spread like a spearhead.
	fin := func(pts []float64, bias float64) {
		c.polygon(pts, func(x, y int) color.RGBA {
			return krakenSkin(x, y, 0.6+bias-float64(y)*0.012)
		})
	}
	fin([]float64{21, 25, 5, 19, 9, 11, 21, 2}, 0.08)
	fin([]float64{27, 25, 42, 17, 37, 9, 23, 2}, -0.12)
	// Arms coiling at the surface on either side.
	arm := func(ctrl ...[3]float64) {
		krakenTube(c, spine(ctrl...), 9)
	}
	arm([3]float64{43, krakHeadWY + 2, 3}, [3]float64{46.5, krakHeadWY - 6, 2.5}, [3]float64{50.5, krakHeadWY - 11, 1.8}, [3]float64{53, krakHeadWY - 8, 1.2}, [3]float64{51.5, krakHeadWY - 6, 0.9})
	arm([3]float64{9, krakHeadWY + 2, 2.8}, [3]float64{6, krakHeadWY - 5, 2.2}, [3]float64{3, krakHeadWY - 8, 1.5}, [3]float64{2, krakHeadWY - 5, 1})
	// The mantle: a great wet cone.
	mantle := spine([3]float64{27.5, krakHeadWY + 4, 12.5}, [3]float64{27, 37, 12}, [3]float64{26, 25, 9.6}, [3]float64{24.5, 14, 6.2}, [3]float64{23, 6, 2.6}, [3]float64{22.5, 2.5, 1.2})
	drawTubeWith(c, mantle, func(x, y int, p *spinePt, t float64) color.RGBA {
		l := lum(p.nx*t*0.9-0.1, p.ny*t*0.9-0.2)
		if t < -0.2 && t > -0.55 {
			l += 0.1 // the wet shine down the lit side
		}
		return krakenSkin(x, y, l*0.95)
	})
	// The eye: as big as a shield, pale gold, under a scowling lid.
	const ex, ey = 32.5, 39.0
	c.ellipse(ex, ey, 6.6, 6.2, func(x, y int, nx, ny float64) color.RGBA { return krakenRamp.pick(0.12+ny*0.08, x, y) })
	c.ellipse(ex, ey, 5.4, 5, func(x, y int, nx, ny float64) color.RGBA {
		d := math.Hypot(nx, ny)
		return mix(hex(0xfff0a8), hex(0xd98a1e), clamp01(d*1.1-0.15)-dither(x, y)*0.15)
	})
	c.ellipse(ex+0.6, ey+0.4, 2.2, 3.1, func(x, y int, nx, ny float64) color.RGBA { return colInk }) // the pupil
	c.put(30, 37, colWhite)
	c.put(31, 37, colWhite)
	c.put(30, 38, colWhite)
	c.polygon([]float64{25, 31.5, 40, 36.2, 40, 31, 32, 30}, func(x, y int) color.RGBA { // the lid
		return krakenRamp.pick(0.3+float64(x-25)*0.012, x, y)
	})
	c.line(25.5, 32.2, 39.5, 36.6, func(x, y int, _ float64) color.RGBA { return krakenRamp[0] })

	c.outline(0.78)
	sinkBelow(c, wy)
	seaRing(c, 27.5, wy, 23, 3.6, 361)
	return c
}

// krakenShadowCanvas draws the squid from above, as a shape in the deep: the
// finned mantle behind, eight arms and two long tentacles spread ahead.
func krakenShadowCanvas() *canvas {
	c := newCanvas(96, 64)
	dark := func(int, int, *spinePt, float64) color.RGBA { return colKrakShadow }
	c.polygon([]float64{2, 32, 12, 19, 23, 28, 23, 36, 12, 45}, solid(colKrakShadow)) // the fins
	drawTubeWith(c, spine([3]float64{3, 32, 1.5}, [3]float64{14, 32, 6}, [3]float64{28, 32, 8.6}, [3]float64{42, 32, 7.8}, [3]float64{50, 32, 6.2}), dark)
	// One limb from the head at angle a, n long, snaking; a tentacle ends in a club.
	limb := func(a, n, r, wave, phase float64, club bool) {
		const step = 0.5
		x, y := 52.0, 32.0+math.Sin(a)*3
		var pts []spinePt
		steps := int(n / step)
		for i := 0; i <= steps; i++ {
			s := float64(i) / float64(steps)
			rr := lerpf(r, 0.7, s)
			if club {
				rr = 1.1 + 1.5*math.Exp(-math.Pow((s-0.88)/0.07, 2))
			}
			pts = append(pts, spinePt{x: x, y: y, r: rr, s: s})
			h := a + wave*math.Sin(s*math.Pi*2.2+phase)*(0.3+s)
			x, y = x+math.Cos(h)*step, y+math.Sin(h)*step
		}
		drawTubeWith(c, withNormals(pts), dark)
	}
	for i, a := range []float64{-1.15, -0.8, -0.48, -0.2, 0.2, 0.48, 0.8, 1.15} {
		limb(a, 27+float64(i%3)*3, 2.5, 0.55, float64(i)*1.9, false)
	}
	limb(-0.07, 41, 1.1, 0.3, 0.5, true)
	limb(0.07, 39, 1.1, 0.3, 3.4, true)
	softenShadow(c)
	return c
}

// --- icons ------------------------------------------------------------------

// genSeaIcons draws the 16x16 bag icons of sim.ItemRaft and sim.ItemSailboat
// into icons.
func genSeaIcons(icons *[sim.NumItems]*ebiten.Image) {
	// The raft: three logs lashed side by side.
	raft := newCanvas(16, 16)
	for i, l := range [][2]int{{2, 13}, {1, 15}, {2, 14}} {
		y0 := 4 + i*3
		raft.rect(l[0], y0, l[1]-l[0], 3, func(x, y int) color.RGBA {
			if x >= l[1]-2 {
				return woodRamp.at(3 - (y - y0))
			}
			return deckRamp.at(5 - (y-y0)*2 - (x/2+i)%2)
		})
	}
	for _, x := range []int{4, 10} {
		raft.rect(x, 3, 1, 10, func(_, y int) color.RGBA { return ropeRamp.at(2 - y%3) })
	}
	raft.outline(0.8)
	for x := 1; x < 15; x += 2 {
		raft.put(x+(x/2)%2, 14, withAlpha(colFoamDim, 230))
	}
	icons[sim.ItemRaft] = raft.image()

	// The sailing boat: hull, mast and a full hide sail.
	boat := newCanvas(16, 16)
	for x := 1; x < 15; x++ {
		u := (float64(x) + 0.5 - 8) / 7
		top := 11 - int(math.Round(2.2*math.Pow(math.Abs(u), 2)))
		bot := 14 - int(math.Round(3*math.Pow(math.Abs(u), 3)))
		for y := top; y <= bot; y++ {
			boat.put(x, y, hullRamp.at(5-(y-top)*2+x%2))
		}
	}
	boat.polygon([]float64{4, 2.5, 12.5, 1, 14, 5.5, 12, 9.5, 4.5, 9.5}, func(x, y int) color.RGBA {
		if y == 5 {
			return sailRamp[2]
		}
		return sailRamp.pick(0.9-float64(x-4)*0.035-float64(y)*0.02, x, y)
	})
	boat.put(9, 4, colSailPaint)
	boat.put(10, 3, colSailPaint)
	boat.put(10, 5, colSailPaint)
	boat.put(11, 4, colSailPaint)
	boat.rect(6, 1, 1, 10, func(x, y int) color.RGBA { return deckRamp.at(3 - y%2) })
	boat.outline(0.8)
	for x := 2; x < 14; x += 2 {
		boat.put(x+(x/2)%2, 15, withAlpha(colFoamDim, 230))
	}
	icons[sim.ItemSailboat] = boat.image()
}
