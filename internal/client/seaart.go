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
}

func genSeaArt() seaArt { return seaBoats(envHD) }

// seaBoats is the boats' art: redrawn at HD (boathd.go), or the old.
func seaBoats(hd bool) seaArt {
	var a seaArt
	if hd {
		const K = HDDensity
		rb, rf := raftCanvasesHD()
		a.raftBack, a.raftFront = mkUp(rb, raftAX*K, raftWY*K), mkUp(rf, raftAX*K, raftWY*K)
		sb, sf := sailboatCanvasesHD()
		a.sailBack, a.sailFront = mkUp(sb, boatAX*K, boatWY*K), mkUp(sf, boatAX*K, boatWY*K)
		for i := range a.sail {
			a.sail[i] = mkUp(sailCanvasHD(i), sailMastX*K, sailFootY*K)
		}
		return a
	}
	rb, rf := raftCanvases()
	a.raftBack, a.raftFront = mk(rb, raftAX, raftWY), mk(rf, raftAX, raftWY)
	sb, sf := sailboatCanvases()
	a.sailBack, a.sailFront = mk(sb, boatAX, boatWY), mk(sf, boatAX, boatWY)
	for i := range a.sail {
		a.sail[i] = mk(sailCanvas(i), sailMastX, sailFootY)
	}
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
