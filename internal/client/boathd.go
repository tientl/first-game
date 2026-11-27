package client

import (
	"image/color"
	"math"
)

// The boats at HD: the log raft and the sailing dugout with its hide sail,
// redrawn three times as fine on the same lines as the old art (seaart.go),
// so they sit, carry their rider and take the wind exactly as before. Each
// part is painted pixel by pixel from its shape in art pixels (hdPaint):
// logs round with furrowed bark and ringed ends, lashings of twisted rope,
// the dugout's planks stitched along a seam, ribs inside the far wall, the
// sail's hides stitched and daubed with a sun; foam along the waterline.
// The wake behind a boat under way is drawn in drawBoatWake.

// hdPaint paints every pixel of c (an hdCanvas) inside the art-pixel box
// x0, y0, x1, y1 that fn covers, with fn's colour; fn gets the pixel's
// middle in art pixels.
func hdPaint(c *canvas, x0, y0, x1, y1 float64, fn func(fx, fy float64) (color.RGBA, bool)) {
	const K = HDDensity
	for y := max(0, int(math.Floor(y0*K))); y < min(c.h, int(math.Ceil(y1*K))); y++ {
		for x := max(0, int(math.Floor(x0*K))); x < min(c.w, int(math.Ceil(x1*K))); x++ {
			if col, ok := fn((float64(x)+0.5)/K, (float64(y)+0.5)/K); ok {
				c.put(x, y, col)
			}
		}
	}
}

// boatSink clears everything of c under art row wy.
func boatSink(c *canvas, wy float64) {
	for y := int(wy * HDDensity); y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.put(x, y, color.RGBA{})
		}
	}
}

// boatFoam breaks the water along art row wy from x0 to x1: a ragged band of
// foam, a fleck above and a fainter line below, three times as fine as
// seaFoam's.
func boatFoam(c *canvas, x0, x1, wy float64, seed uint32) {
	const K = HDDensity
	for x := int(x0 * K); x <= int(x1*K); x++ {
		edge := math.Min(float64(x)/K-x0, x1-float64(x)/K)
		if edge < 0 {
			continue
		}
		y := int(wy*K) - 1
		h := hashf(x/2, 0, seed)
		w := 1 + int(math.Min(2, edge))
		for dy := 0; dy < w; dy++ {
			col := colFoam
			if h > 0.55 || dy == w-1 {
				col = colFoamDim
			}
			c.put(x, y+dy, col)
		}
		if edge > 1.5 && hashf(x/3, 1, seed+1) < 0.3 {
			c.put(x, y-1, withAlpha(colFoam, 220))
		}
		if edge > 1 && hashf(x/2, 2, seed+2) < 0.45 {
			c.put(x, y+w+1, withAlpha(colFoamDim, 170))
		}
	}
}

// inPoly reports whether (x, y) is inside polygon p (x, y pairs).
func inPoly(p []float64, x, y float64) bool {
	in := false
	n := len(p) / 2
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi, xj, yj := p[2*i], p[2*i+1], p[2*j], p[2*j+1]
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
	}
	return in
}

// boatSeg is how far (x, y) is from the segment (x0, y0)-(x1, y1), and how
// far along it, 0 to 1.
func boatSeg(x, y, x0, y0, x1, y1 float64) (d, t float64) {
	dx, dy := x1-x0, y1-y0
	t = clamp01(((x-x0)*dx + (y-y0)*dy) / (dx*dx + dy*dy))
	return math.Hypot(x-x0-dx*t, y-y0-dy*t), t
}

// ropeAt is twisted rope at (fx, fy) lit at l: strands slanting across.
func ropeAt(fx, fy, l float64) color.RGBA {
	if math.Mod(fx*1.6+fy*1.6+10, 1) < 0.45 {
		l -= 0.28
	}
	return pick(ropeRamp, l)
}

// --- the raft ------------------------------------------------------------------

// raftLogHD paints a log along the raft from x0 to x1, its top at y, h
// thick: round, its bark furrowed along it, its stern end rough, its bow
// end sawn and ringed.
func raftLogHD(c *canvas, x0, x1, y, h, tone float64, seed uint32) {
	mid := y + h/2
	hdPaint(c, x0-1, y-0.5, x1+1, y+h+0.5, func(fx, fy float64) (color.RGBA, bool) {
		v := (fy - y) / h
		if v < 0 || v > 1 {
			return color.RGBA{}, false
		}
		ny := 2*v - 1
		cap := math.Sqrt(math.Max(0, 1-ny*ny)) // the round of the log, end on
		if fx < x0+0.6-0.6*cap || fx > x1+0.4*cap {
			return color.RGBA{}, false
		}
		if fx > x1-0.8*cap { // the sawn end: rings
			r := math.Hypot((fx-(x1-0.8))*1.8, fy-mid)
			l := 0.62 - 0.12*ny + 0.12*math.Sin(r*5.5)
			if r > h/2-0.45 {
				l = 0.35 // the bark round it
				return pick(deckRamp, l), true
			}
			return pick(woodRamp, l), true
		}
		l := 0.92 - 0.75*v + tone
		furrow := math.Sin(fy*4.2+math.Sin(fx*0.35+float64(seed))*1.3) * 0.5
		if furrow > 0.3 {
			l -= 0.14
		}
		if hashf(int(fx*1.3), int(fy*2), seed) < 0.05 {
			l -= 0.2 // a knot
		}
		if v < 0.12 {
			l += 0.08 // the light along its top
		}
		return pick(deckRamp, l), true
	})
}

// raftCanvasesHD is raftCanvases redrawn at HD: back the deck, the
// steering oar and the bundle at the bow; front the nearest log and the
// lashings over it.
func raftCanvasesHD() (back, front *canvas) {
	back, front = hdCanvas(54, 26), hdCanvas(54, 26)
	const wy = raftWY
	// The oar, trailing in the water behind, under the deck.
	hdPaint(back, 0, 5, 18, 22, func(fx, fy float64) (color.RGBA, bool) {
		if d, t := boatSeg(fx, fy, 17, 6.5, 5, 15); d < 0.6 {
			return pick(woodRamp, 0.75-t*0.5-d*0.4), true
		}
		if inPoly([]float64{6.5, 14, 4, 13.5, 0.5, 18, 1, 21, 4, 21}, fx, fy) {
			return pick(woodRamp, 0.65-(fy-13)*0.06), true
		}
		return color.RGBA{}, false
	})
	raftLogHD(back, 10, 44, 6, 3, -0.05, 1)
	raftLogHD(back, 8, 47, 9, 3, 0.03, 2)
	raftLogHD(back, 11, 45, 12, 3, -0.02, 3)
	raftLogHD(front, 7, 48, 14, 6, 0, 4)
	for _, px := range []float64{15, 37} {
		// A pole across the logs, lashed down at every one.
		hdPaint(back, px-0.5, 4.5, px+2.5, 15.5, func(fx, fy float64) (color.RGBA, bool) {
			u := (fx - px) / 2
			if u < 0 || u > 1 || fy < 5 || fy > 15 {
				return color.RGBA{}, false
			}
			if math.Mod(fy, 3) > 2 {
				return ropeAt(fx, fy, 0.7-u*0.3), true
			}
			return pick(deckRamp, 0.95-u*0.55), true
		})
		hdPaint(front, px-1.5, 13.5, px+3.5, 20.5, func(fx, fy float64) (color.RGBA, bool) {
			u := (fx - (px - 1)) / 4
			if u < 0 || u > 1 || fy < 14 || fy > 20 {
				return color.RGBA{}, false
			}
			return ropeAt(fx, fy, 0.95-(fy-14)*0.13-math.Abs(u-0.4)*0.3), true
		})
	}
	// A bundle of supplies wrapped in hide at the bow, tied with rope.
	hdPaint(back, 37, 3, 46, 11, func(fx, fy float64) (color.RGBA, bool) {
		nx, ny := (fx-41.5)/3.6, (fy-7)/3.2
		if nx*nx+ny*ny > 1 {
			return color.RGBA{}, false
		}
		if math.Abs(fx-41.3) < 0.45 || (math.Abs(fy-7.2) < 0.4 && fx > 39) {
			return ropeAt(fx, fy, 0.35+lum(nx, ny)*0.6), true
		}
		l := lum(nx, ny) * 0.95
		if math.Sin(fx*2.1+fy*0.7) > 0.8 {
			l -= 0.1 // the hide's creases
		}
		return pick(hideRamp, l), true
	})
	back.outline(0.78)
	front.outline(0.78)
	boatSink(back, wy)
	boatSink(front, wy)
	boatFoam(front, 5, 49, wy, 301)
	boatFoam(back, 0, 4, wy, 305)
	return back, front
}

// --- the sailing boat ---------------------------------------------------------

// sailboatCanvasesHD is sailboatCanvases redrawn at HD.
func sailboatCanvasesHD() (back, front *canvas) {
	back, front = hdCanvas(78, 36), hdCanvas(78, 36)
	const wy = boatWY
	// The far wall seen from inside over the near gunwale, its ribs and
	// thwarts.
	hdPaint(back, 0, 0, 78, 30, func(fx, fy float64) (color.RGBA, bool) {
		top, bot := boatSheer(fx)-3, boatKeel(fx)
		if fy < top || fy > bot {
			return color.RGBA{}, false
		}
		l := 0.3 + 0.07*(fy-top)
		if fy-top < 0.7 {
			l = 0.8 // the far rail
		}
		if math.Mod(fx, 9) > 3.7 && math.Mod(fx, 9) < 4.7 && fy-top > 0.7 {
			l -= 0.12 // ribs
		}
		return pick(hullRamp, l-(fx-boatAX)/boatHalf*0.06), true
	})
	hdPaint(back, 66, 1, 78, 17, func(fx, fy float64) (color.RGBA, bool) {
		if inPoly([]float64{70.5, 11, 73, 5.5, 77.5, 2.5, 75.5, 7.5, 73.5, 12.5}, fx, fy) {
			l := 0.9 - fy*0.05
			if math.Sin((fx+fy)*2.2) > 0.6 {
				l -= 0.12 // the carving's grooves
			}
			return pick(hornRamp, l), true
		}
		if inPoly([]float64{68, 13, 71, 10, 74, 12, 72, 16}, fx, fy) {
			return pick(hullRamp, 0.6), true
		}
		return color.RGBA{}, false
	})
	for _, tx := range []float64{24, 52} {
		y := math.Round(boatSheer(tx)) - 1
		hdPaint(back, tx, y, tx+3, y+2, func(fx, fy float64) (color.RGBA, bool) {
			return pick(deckRamp, 0.9-(fy-y)*0.3), true
		})
	}

	// The near wall: the dugout below, a plank stitched on above it, the
	// rail on top.
	hdPaint(front, 0, 0, 78, 32, func(fx, fy float64) (color.RGBA, bool) {
		top, bot := boatSheer(fx), boatKeel(fx)
		if fy < top || fy > bot {
			return color.RGBA{}, false
		}
		d := fy - top
		v := d / math.Max(1, bot-top)
		l := 0.72 - 0.42*v - (fx-boatAX)/boatHalf*0.07
		l += (vnoise(fx/5, fy*1.3, 311) - 0.5) * 0.2 // the grain along the hull
		switch {
		case d < 0.8:
			l = 0.95 // the rail
		case d < 1.4:
			l = 0.42
		case d > 3.8 && d < 4.5:
			l -= 0.22 // the seam between plank and dugout
		}
		if d > 3.2 && d < 5.1 && math.Mod(fx, 7) > 2.9 && math.Mod(fx, 7) < 3.5 {
			return ropeAt(fx, fy, 0.6), true // stitches across the seam
		}
		return pick(hullRamp, l), true
	})
	// A painted eye on the bow, to watch for what swims below.
	ex, ey := 63.5, math.Round(boatSheer(62.5))+3.5
	hdPaint(front, ex-3, ey-2, ex+3, ey+2, func(fx, fy float64) (color.RGBA, bool) {
		nx, ny := (fx-ex)/2.3, (fy-ey)/1.3
		r := nx*nx + ny*ny
		switch {
		case r > 1:
			return color.RGBA{}, false
		case math.Hypot(fx-ex, fy-ey) < 0.65:
			return colSailPaint, true
		case r > 0.7:
			return hex(0x2a1a12), true
		}
		return hex(0xe9dfc6), true
	})
	front.outline(0.78)
	back.outline(0.78)
	boatSink(front, wy)
	boatSink(back, wy)
	boatFoam(front, 11, 67, wy, 302)

	// The outrigger: a float on two booms lashed to the gunwale, drawn on
	// its own so its outline lies over the hull.
	out := hdCanvas(78, 36)
	const fy0 = wy + 6
	for _, bx := range []float64{25, 53} {
		gy := boatSheer(bx)
		hdPaint(out, bx-4, gy-2, bx+1, fy0+1, func(fx, fy float64) (color.RGBA, bool) {
			d, t := boatSeg(fx, fy, bx, gy-1, bx-2.5, fy0+0.5)
			if d > 0.55 {
				return color.RGBA{}, false
			}
			if t < 0.14 {
				return ropeAt(fx, fy, 0.6), true
			}
			return pick(deckRamp, 0.8-t*0.35-d*0.3), true
		})
	}
	hdPaint(out, 11, fy0-1, 64, fy0+4, func(fx, fy float64) (color.RGBA, bool) {
		if !inPoly([]float64{12, fy0 + 1.5, 16, fy0, 58, fy0, 63, fy0 + 1, 58, fy0 + 3, 16, fy0 + 3}, fx, fy) {
			return color.RGBA{}, false
		}
		if (fx > 22 && fx < 24) || (fx > 50 && fx < 52) {
			return ropeAt(fx, fy, 0.75-(fy-fy0)*0.2), true // lashings
		}
		l := 0.85 - (fy-fy0)*0.25 + (hashf(int(fx/3), int(fy), 312)-0.5)*0.12
		return pick(deckRamp, l), true
	})
	out.outline(0.78)
	boatSink(out, fy0+3)
	front.over(out, 0, 0)
	boatFoam(front, 13, 61, fy0+3, 303)
	// The bow wave.
	hdPaint(front, 65, wy-3, 72, wy, func(fx, fy float64) (color.RGBA, bool) {
		u := (fx - 66) / 5
		if u < 0 || u > 1 {
			return color.RGBA{}, false
		}
		crest := wy - 0.5 - 1.8*math.Sin(u*math.Pi)
		if math.Abs(fy-crest) < 0.5 {
			return colFoam, true
		}
		if fy > crest && fy < wy-0.3 && hashf(int(fx*3), int(fy*3), 304) < 0.5 {
			return withAlpha(colFoamDim, 220), true
		}
		return color.RGBA{}, false
	})
	return back, front
}

// --- the sail ------------------------------------------------------------------

// sailCanvasHD is sailCanvas redrawn at HD: the mast and its hide sail hung
// from a slanting yard and laced to a boom; fill 0 hangs slack in folds, 1
// bellies, 2 is full of wind.
func sailCanvasHD(fill int) *canvas {
	c := hdCanvas(42, 52)
	belly := float64(fill) * 3.2
	sag := 0.0
	if fill == 0 {
		sag = 1
	}
	const yx0, yx1, bx0, bx1 = 3.0, 33.0, 4.0, 31.0
	yardY := func(x float64) float64 { return lerpf(13, 5, (x-yx0)/(yx1-yx0)) + sag*2*((x-yx0)/(yx1-yx0)) }
	boomY := func(x float64) float64 { return lerpf(40, 37, (x-bx0)/(bx1-bx0)) + sag*1.5 }
	const vy0, vy1 = 8.0, 40.0
	hdPaint(c, 0, 0, 42, 52, func(fx, fy float64) (color.RGBA, bool) {
		v := clamp01((fy - vy0) / (vy1 - vy0))
		bow := math.Sin(v * math.Pi)
		left := lerpf(yx0+1, bx0+1, v) + belly*0.55*bow + sag*2.6*bow
		right := lerpf(yx1-1, bx1-1, v) + belly*bow - sag*3.6*bow
		if fx < left || fx > right {
			return color.RGBA{}, false
		}
		u := (fx - left) / (right - left)
		hx := lerpf(yx0+1, yx1-1, u)
		top, bot := yardY(hx)+1, boomY(lerpf(bx0+1, bx1-1, u))-1
		if fy < top || fy > bot {
			return color.RGBA{}, false
		}
		vv := (fy - top) / (bot - top)
		l := 0.8 - 0.22*vv - 0.2*u
		l += belly / 6.4 * 0.22 * math.Cos((u-0.42)*math.Pi*1.3) * math.Sin(vv*math.Pi)
		if sag > 0 {
			l += 0.21*math.Sin(u*15+vv*3+math.Sin(vv*5)) - 0.08
		}
		l += (vnoise(fx/3, fy/3, 321) - 0.5) * 0.14 // the mottle of hide
		col := pick(sailRamp, l)
		// Hides stitched together: seams across, and one down, with stitches.
		pw := bot - top
		seam := 0.34 + 0.03*math.Sin(u*9)
		seam2 := 0.68 + 0.03*math.Cos(u*7)
		for _, s := range [2]float64{seam, seam2} {
			if d := math.Abs(vv-s) * pw; d < 0.35 {
				col = pick(sailRamp, l-0.3)
			} else if d < 0.8 && math.Mod(fx*1.4, 1) < 0.35 {
				col = pick(sailRamp, l-0.22)
			}
		}
		if vv > seam && vv < seam2 && math.Abs(u-0.3)*(right-left) < 0.3 {
			col = pick(sailRamp, l-0.3)
		}
		// A sun daubed in red ochre.
		dx, dy := (u-0.6)*(right-left), (vv-0.5)*pw
		if d := math.Hypot(dx, dy*0.95); (d > 3.2 && d < 4.3) || d < 1.4 {
			col = mix(colSailPaint, col, 0.2)
		} else if d >= 4.3 && d < 6 && math.Mod(math.Atan2(dy, dx)*8/math.Pi+16, 2) < 0.6 {
			col = mix(colSailPaint, col, 0.4) // its rays
		}
		return col, true
	})
	// Lacing along yard and boom.
	spar := func(x0, x1 float64, yf func(float64) float64) {
		hdPaint(c, x0, 0, x1+1, 52, func(fx, fy float64) (color.RGBA, bool) {
			y := yf(fx)
			if fy < y-0.2 || fy > y+1.6 {
				return color.RGBA{}, false
			}
			if math.Mod(fx, 4) < 0.8 {
				return ropeAt(fx, fy, 0.6), true
			}
			return pick(deckRamp, 0.9-(fy-y)*0.35), true
		})
	}
	spar(yx0, yx1, yardY)
	spar(bx0, bx1, boomY)
	// The mast, in front of the sail, lashed at yard and boom, and a
	// streamer at its head.
	hdPaint(c, sailMastX-1.2, 3.5, sailMastX+1.2, sailFootY, func(fx, fy float64) (color.RGBA, bool) {
		u := (fx - (sailMastX - 1)) / 2
		if u < 0 || u > 1 || fy < 4 {
			return color.RGBA{}, false
		}
		for _, y := range []float64{yardY(sailMastX) + 1, boomY(sailMastX) + 1} {
			if math.Abs(fy-y) < 1 {
				return ropeAt(fx, fy, 0.7), true
			}
		}
		return pick(deckRamp, 0.95-u*0.6-fy*0.003), true
	})
	wave := float64(fill)
	hdPaint(c, sailMastX-8, 1, sailMastX, 10, func(fx, fy float64) (color.RGBA, bool) {
		u := (sailMastX - 1.2 - fx) / (5 + wave*0.5) // 0 at the mast, 1 at its tail
		if u < 0 || u > 1 {
			return color.RGBA{}, false
		}
		cy := 4.5 + (3-wave)*u*1.2 + 0.8*math.Sin(u*5+wave)
		if math.Abs(fy-cy) > 0.8*(1-u*0.5) {
			return color.RGBA{}, false
		}
		if u > 0.7 {
			return hex(0xc8502e), true
		}
		return colSailPaint, true
	})
	c.outline(0.78)
	return c
}

// drawBoatWake draws the wake of a boat under way at (sx, sy), heading h
// tiles a tick at speed sp: two arms of foam spreading back from the stern
// in a V, broken and fading, churned water right behind, at the canvas's
// own resolution.
func drawBoatWake(dst drawTarget, sx, sy, h, sp, t float64, raft bool) {
	hx, hy := math.Cos(h), math.Sin(h)*TileH/TileW
	l := math.Hypot(hx, hy)
	hx, hy = hx/l, hy/l
	nx, ny := -hy, hx
	stern := 26.0
	if raft {
		stern = 22
	}
	k := clamp01(sp * 14)
	ox, oy := sx-hx*stern*0.8, sy+2-hy*stern*0.5
	var b fxBatch
	n := int(4 + 8*k)
	for arm := -1.0; arm <= 1; arm += 2 {
		for i := 0; i < n; i++ {
			ph := math.Mod(t*0.9+float64(i)*0.37, 1)
			d := (float64(i) + ph) * 3.2
			spread := d * 0.42
			x := ox - hx*d + nx*arm*spread
			y := oy - hy*d*0.6 + ny*arm*spread*0.45
			a := (1 - float64(i)/float64(n)) * k * 0.85
			w := 2.2 + d*0.08
			b.taper(x-hx*w*0.5, y-hy*w*0.3, x+hx*w*0.5, y+hy*w*0.3, 0.7, 0.4, colFoam, a, 0)
			if i%2 == 0 {
				b.dot(x-nx*arm*1.2, y+0.8, 0.6, colFoamDim, a*0.8, 0)
			}
		}
	}
	for i := 0; i < int(3+5*k); i++ { // the churn right behind
		ph := math.Mod(t*1.3+float64(i)*0.29, 1)
		x := ox - hx*ph*6 + math.Sin(float64(i)*2.7)*2
		y := oy - hy*ph*3 + math.Cos(float64(i)*1.9)*0.8
		b.disc(x, y, 1.2*(1-ph)+0.4, 0.6*(1-ph)+0.25, colFoam, (1-ph)*0.7*k, 0)
	}
	b.flush(dst)
}
