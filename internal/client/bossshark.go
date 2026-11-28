package client

import (
	"image/color"
	"math"
)

// The Megalodon, drawn at HDDensity. It lives in the water like the sea
// monsters of seahd.go and is drawn the same way: swimming, it is a dark
// shape under the surface with its fin cutting the water and a wake behind
// it; when it bites, bellows or leaps, it comes out side-on, cut off at the
// waterline in boiling foam. Every frame is anchored on the waterline. The
// lurking Spinosaurus (density 1) is here too.

var (
	megaSkin   = mkRamp(0x0e1620, 0x182634, 0x24384a, 0x344e62, 0x4a687c, 0x668698, 0x88a4b0)
	megaBelly  = mkRamp(0x7c8a96, 0xa8b6be, 0xd0dadc, 0xecf2f0, 0xffffff)
	megaShadow = hex(0x07131f)
	megaWet    = hex(0xd6ecf2)
	megaGum    = hex(0xd8808a)
)

// megaK is how many art pixels a unit of the shark's drawing is: it is 230
// art pixels from the tips of its tail to its snout, nearly ten tiles.
const megaK = 1.45

// megaMouth is the line of the upper jaw in the shark's frame: v at u.
func megaMouth(u float64) float64 { return 8.5 - (u-megaHingeU)*0.24 }

const megaHingeU, megaHingeV = 39.0, 8.5

// megaSide paints the shark side-on in frame fr (u from the root of the tail
// at -68 to the tip of the snout at 70, v down), its jaws gape wide and its
// tail swung by wag (-1..1).
func megaSide(c *canvas, fr bossFrame, gape, wag float64) {
	fin := func(q [][2]float64, bias float64, seed uint32) {
		u0, v0 := q[0][0], q[0][1]
		fr.poly(c, q, func(x, y int) color.RGBA {
			u, v := fr.local(x, y)
			d := math.Hypot(u-u0, v-v0)
			l := 0.4 + bias - v*0.004 + u*0.001 - d*0.004
			if f := math.Atan2(v-v0, u-u0) * 7; f-math.Floor(f) < 0.2 {
				l -= 0.07 // the rays of the fin
			}
			return rq(megaSkin, l+(hashf(x/2, y/2, seed)-0.5)*0.06, x, y)
		})
	}
	// The tail's crescent, swung with the stroke, and the fins of the back.
	tv := wag * 5
	fin([][2]float64{{-62, -3}, {-74, -14 + tv}, {-88, -30 + tv*1.6}, {-80, -9 + tv}, {-76, tv}, {-83, 16 + tv}, {-86, 22 + tv*1.4}, {-72, 12 + tv*0.6}, {-62, 3}}, 0, 961)
	fin([][2]float64{{14, -15}, {2, -30}, {-8, -39}, {-9, -28}, {-13, -14}}, 0.06, 962)
	fin([][2]float64{{-36, -9}, {-44, -15}, {-45, -8}}, 0, 963)
	fin([][2]float64{{-34, 9}, {-42, 16}, {-44, 9}}, -0.1, 964)
	fin([][2]float64{{30, 8}, {22, 30}, {14, 38}, {17, 12}}, -0.16, 965) // the far pectoral
	// The notch torn out of its back fin.
	fr.tri(c, -9.6, -30, -6.5, -31.5, -9.4, -33.5, cutOut(c))

	hx, hy := fr.at(megaHingeU, megaHingeV)
	fj := bossFrame{ox: hx, oy: hy, a: fr.a + gape*0.72, sc: fr.sc}
	fr.poly2(c, fj, [][3]float64{{megaHingeU - 6, megaHingeV - 2, 0}, {65, megaMouth(65), 0}, {22, 0, 1}, {-6, 6, 1}}, func(x, y int) color.RGBA {
		u, v := fj.local(x, y)
		d := math.Hypot(u, v)
		col := mix(hex(0x1c0610), hex(0xc0505a), clamp01((d-5)/20)-hashf(x/2, y/2, 966)*0.14)
		if f := d / 3.4; f-math.Floor(f) < 0.2 && d > 8 {
			col = mix(col, hex(0x1c0610), 0.22) // the arches of its gullet
		}
		return col
	})
	tooth := func(f bossFrame, u, v, w, h float64, dim bool) {
		f.tri(c, u-w, v, u+w, v, u+w*0.15, v+h, func(x, y int) color.RGBA {
			lu, _ := f.local(x, y)
			col := colTooth
			if lu > u+w*0.2 {
				col = hex(0xd2ccba) // its shaded edge
			}
			if dim {
				col = mix(col, hex(0x7a5560), 0.4)
			}
			return col
		})
	}
	// The rows of teeth behind the first.
	for u := megaHingeU + 5.3; u < 63; u += 2.7 {
		tooth(fr, u, megaMouth(u)-0.6, 1.1, 3.2, true)
	}
	for u := 4.8; u < 20; u += 2.7 {
		tooth(fj, u, 0.8, 1.1, -3, true)
	}
	// The lower jaw.
	bossPaint(c, 0, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if v < 0 && u > -2 {
			return color.RGBA{}
		}
		if v < 1.5 && u > 0 {
			return mix(megaGum, hex(0xa85864), clamp01(v/1.5)) // the gum
		}
		return rq(megaBelly, 0.2+q.l*0.75-max(0, v-6)*0.03, x, y)
	}, fj.knobs([]knob{{-5, 4.5, 7.5}, {8, 3.6, 5.6}, {20, 2.6, 3.4}}))
	// The body.
	bossPaint(c, 0.2, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > megaHingeU && v > megaMouth(u) {
			return color.RGBA{}
		}
		if u > megaHingeU+1 && v > megaMouth(u)-1.5 {
			return mix(hex(0xa85864), megaGum, clamp01((v-megaMouth(u)+1.5)/1.5))
		}
		l := 0.1 + q.l*0.84
		// Countershaded: slate above, white below, the line between them ragged.
		edge := 0.2 + 0.08*math.Sin(u*0.21) + 0.05*math.Sin(u*0.57+1) + (hashf(x/3, y/3, 967)-0.5)*0.1
		if q.v > edge {
			if u > 20 && u < 37 && v < 9 { // the gills run down into the white
				if g := (u - 21 - v*v*0.02) / 3.2; g-math.Floor(g) < 0.14 {
					return megaBelly[0]
				}
			}
			return rq(megaBelly, l+0.06-max(0, q.v-0.5)*0.34, x, y)
		}
		if q.v > edge-0.07 {
			return rq(megaSkin, l+0.2, x, y)
		}
		if u > 20 && u < 37 && v > -9 { // the gills: five dark slits, a pale lip to each
			g := (u - 21 - v*v*0.02) / 3.2
			switch gf := g - math.Floor(g); {
			case gf < 0.16:
				return megaSkin[0]
			case gf < 0.3:
				return rq(megaSkin, l+0.2, x, y)
			}
		}
		// Dapples and old nicks on the hide.
		l += -0.1*q.v + 0.05*math.Sin(u*0.37+v*0.5)*math.Sin(v*0.61-u*0.11)
		if h := hashf(x/2, y/2, 968); h < 0.05 {
			l += 0.08
		} else if h > 0.95 {
			l -= 0.07
		}
		if math.Abs(q.v+0.06) < 0.025 && u < 18 && u > -55 {
			l += 0.1 // the line along its flank
		}
		if u > 56 && hashf(int(math.Floor(u*1.3)), int(math.Floor(v*1.3)), 969) < 0.1 {
			return megaSkin[1] // the pores of its snout
		}
		// Wet: the light slides along its back in streaks.
		if q.v < -0.42 && q.v > -0.8 && q.l > 0.62 {
			if s := hashf(int(math.Floor(u/5+q.v*3)), int(math.Floor(q.v*26)), 970); s < 0.13 {
				return mix(megaSkin[6], megaWet, 0.4+s*3)
			} else if s < 0.3 {
				return megaSkin[6]
			}
		}
		return rq(megaSkin, l+0.02, x, y)
	}, fr.knobs([]knob{{-68, wag * 2, 3.2}, {-50, wag * 0.6, 7}, {-25, 0, 13}, {0, 0, 17}, {22, 0.5, 17.5}, {40, 1, 15.5}, {52, 0.5, 12.5}, {62, -1.5, 8}, {70, -3, 3.6}}))
	// Teeth: broad triangles, a row to each jaw.
	for u := megaHingeU + 4; u < 64; u += 2.7 {
		tooth(fr, u, megaMouth(u)-0.4, 1.35, 4.8, false)
	}
	for u := 3.5; u < 21; u += 2.7 {
		tooth(fj, u, 0.6, 1.35, -4.2, false)
	}
	// The near pectoral fin.
	fr.poly(c, [][2]float64{{27, 9}, {17, 33}, {6, 44}, {8, 30}, {12, 11}}, func(x, y int) color.RGBA {
		u, v := fr.local(x, y)
		l := 0.52 - (v-9)*0.008 + (u-6)*0.006
		if f := math.Atan2(v-9, 27-u) * 8; f-math.Floor(f) < 0.18 {
			l -= 0.07
		}
		if u > 9+(44-v)*0.42 {
			l += 0.1 // its leading edge
		}
		return rq(megaSkin, l, x, y)
	})
	// The eye: black and blank, a pale ring under it.
	frameQuad(c, fr, 49.4, -1.4, 53, -1.6, 0.7, megaSkin[5])
	hdEye(c, fr, 51, -3.5, 1.55, megaSkin[0], colBlack, hex(0x18222c), false)
	nx, ny := fr.at(66, -4.6)
	c.dotPx(nx, ny, 0.6*fr.k1(), megaSkin[0])
	// Old scars across the snout, the flank and the tail.
	for _, q := range [][4]float64{{54, -9, 60, -4}, {57, -10.5, 62.5, -6}, {-6, -8, 6, -2}, {-2, -11, 8, -6}, {-1, -5, 4, -1.5}, {-40, -3, -31, 1}, {30, -12, 35, -15}} {
		frameQuad(c, fr, q[0], q[1]+0.5, q[2], q[3]+0.5, 0.5, megaSkin[1])
		frameQuad(c, fr, q[0], q[1], q[2], q[3], 0.55, megaSkin[6])
	}
}

// megaFin stands the back fin on the waterline wy with its middle at x (art
// pixels), k times its size on the shark's back, leaning by lean; ph boils
// the foam at its foot.
func megaFin(c *canvas, x, wy, k, lean, ph float64, seed uint32) {
	f := hdBossCanvas(int(float64(c.w)/c.k()), int(float64(c.h)/c.k()))
	fr := hdFrameAt(f, x, wy+2, 0, megaK*k)
	sk := func(u, v float64) [2]float64 { return [2]float64{u - v*lean, v} }
	fr.poly(f, [][2]float64{sk(10, 0), sk(1, -13), sk(-7, -21), sk(-7.6, -12), sk(-11, 0)}, func(px, py int) color.RGBA {
		u, v := fr.local(px, py)
		u += v * lean
		front := u - (10 + v*0.78) // 0 on its leading edge
		l := 0.66 + front*0.035 + v*0.006
		if g := (u + v*0.5) / 2.2; g-math.Floor(g) < 0.2 {
			l -= 0.05
		}
		if front > -0.8 {
			return mix(megaSkin[6], megaWet, 0.3) // wet, and catching the light
		}
		if h := hashf(px/2, py/5, seed+3); h < 0.07 && v < -3 {
			return megaWet // water running off it
		}
		return rq(megaSkin, l, px, py)
	})
	fr.tri(f, -7.9, -11, -5.2, -11.8, -7.7, -13.8, cutOut(f)) // a notch torn out
	frameQuad(f, fr, -4, -5, 0.5, -8.5, 0.5, megaSkin[6])
	frameQuad(f, fr, -2.5, -3.5, 1.5, -6.5, 0.4, megaSkin[6])
	f.outline(0.8)
	_, py := f.toPx(0, wy)
	for y := int(py); y < f.h; y++ {
		for i := 0; i < f.w; i++ {
			f.px[y*f.w+i] = color.RGBA{}
		}
	}
	c.blendOver(f, 0, 0)
	w := 12 * megaK * k
	hdFoam(c, x-w*1.1, x+w*1.05, wy, 2.6*k, ph, seed)
}

// megaWake is the wake a fin at (x, wy) leaves behind it (to the left): two
// lines of foam spreading, breaking up with distance; ph moves them.
func megaWake(c *canvas, x, wy, length, ph float64, seed uint32) {
	for i := 0; i < int(length*1.4); i++ {
		d := float64(i)/1.4 + ph*6
		if d > length {
			d -= length
		}
		k := d / length
		if hashf(i, 0, seed) < k*0.75 {
			continue
		}
		for _, side := range []float64{-1, 1} {
			y := wy + side*(1.2+d*0.2) + side*math.Sin(d*0.5+ph*6.28)*0.5 + 1
			col := colFoam
			if hashf(i, int(side)+2, seed) < 0.4+k*0.4 {
				col = colFoamDim
			}
			c.dot(x-d, y, (0.5+0.5*hashf(i, 5, seed))*(1-k*0.5), withAlpha(col, uint8(235*(1-k*0.7))))
		}
		if i%3 == 0 { // the churned water between them
			c.dot(x-d, wy+1+(hashf(i, 7, seed)-0.5)*d*0.3, 0.35, withAlpha(colFoamDim, uint8(170*(1-k))))
		}
	}
}

// megaSwimOn draws the shark from above with the middle of its body at
// (ox, oy): its shape under the water (as dark as alpha), the fin above it,
// its wake and the tip of its tail. up brings its back out of the water too.
// ph (0..1, looping) is the stroke of its tail.
func megaSwimOn(c *canvas, ox, oy, ph, alpha float64, up bool) {
	w, h := int(float64(c.w)/c.k()), int(float64(c.h)/c.k())
	t := hdBossCanvas(w, h)
	width := func(s float64) float64 {
		switch {
		case s < 0.12:
			return megaK * lerpf(1.2, 3.4, s/0.12)
		case s < 0.5:
			return megaK * lerpf(3.4, 10.5, smooth((s-0.12)/0.38))
		case s < 0.72:
			return megaK * lerpf(10.5, 10, (s-0.5)/0.22)
		}
		return megaK * lerpf(10, 2.6, math.Pow((s-0.72)/0.28, 1.6))
	}
	const L = 146 * megaK
	at := func(s float64) (float64, float64) {
		back := 1 - s
		return ox + (s-0.5)*L, oy + 12*math.Sin(1.2+3.4*back-2*math.Pi*ph)*math.Pow(back, 1.3)
	}
	const n = 700
	pts := make([]spinePt, 0, n+1)
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		x, y := at(s)
		pts = append(pts, spinePt{x: x, y: y, r: width(s), s: s})
	}
	dark := func(int, int, tubePt) color.RGBA { return megaShadow }
	hdTube(t, pts, 1e9, dark)
	// The pectoral fins swept back, the second fins, and the blade of the tail.
	fx, fy := at(0.66)
	for _, side := range []float64{-1, 1} {
		t.polygon([]float64{fx + 9, fy + side*11, fx - 30, fy + side*36, fx - 37, fy + side*33, fx - 14, fy + side*11}, solid(megaShadow))
		px, py := at(0.3)
		t.polygon([]float64{px + 6, py + side*6, px - 10, py + side*14, px - 6, py + side*5}, solid(megaShadow))
	}
	tx, ty := at(0)
	t2x, t2y := at(0.07)
	sw := 9 * math.Cos(1.2+3.4-2*math.Pi*ph)
	t.polygon([]float64{t2x, t2y - 3, tx - 14, ty + sw - 5, tx - 16, ty + sw + 3, t2x, t2y + 3}, solid(megaShadow))
	hdSilhouette(t, megaShadow, alpha)
	c.blendOver(t, 0, 0)
	dx, dy := at(0.56)
	if up { // its back out of the water, wet
		b := hdBossCanvas(w, h)
		bx, by := at(0.6)
		b.ellipse(bx, by+1, 46, 6.5, func(x, y int, nx, ny float64) color.RGBA {
			l := 0.5 - ny*0.4 - nx*0.08
			if ny < -0.25 && ny > -0.75 {
				if s := hashf(x/5, y/2, 971); s < 0.22 {
					return mix(megaSkin[6], megaWet, 0.6)
				}
			}
			return rq(megaSkin, l, x, y)
		})
		b.outline(0.8)
		hdSink(b, by+2, 5)
		c.blendOver(b, 0, 0)
		hdFoam(c, bx-48, bx+47, by+2.2, 2.4, ph, 972)
		megaWake(c, bx-44, by+2, 60, ph, 973)
		dx, dy = bx-4, by-1
	}
	megaWake(c, dx-10, dy, 70, ph, 974)
	k := 1.0
	if up {
		k = 1.12
	}
	megaFin(c, dx, dy, k*1.5, 0.06*math.Sin(2*math.Pi*ph), ph, 964)
	// The tip of the tail breaks the surface too, and sweeps.
	f := hdBossCanvas(w, h)
	f.polygon([]float64{tx + 6, ty + 1, tx - 7, ty - 14 + sw*0.2, tx - 12, ty - 9 + sw*0.2, tx - 6, ty + 1}, func(px, py int) color.RGBA {
		return rq(megaSkin, 0.5-float64(px%7)*0.02, px, py)
	})
	f.outline(0.8)
	hdSink(f, ty, 0.01)
	c.blendOver(f, 0, 0)
	hdFoam(c, tx-10, tx+8, ty, 1.8, ph+0.4, 965)
	// The swell it pushes ahead of its snout.
	sx, sy := at(1)
	for i := -5; i <= 5; i++ {
		a := uint8(200 - 22*absInt(i))
		c.dot(sx+7-math.Abs(float64(i))*1.5+math.Sin(2*math.Pi*ph)*0.8, sy+float64(i)*2.2, 0.55, withAlpha(colFoamDim, a))
	}
}

const (
	megaSwimN   = 8
	megaRoarN   = 8
	megaSlamN   = 9
	megaW       = 440
	megaH       = 400
	megaWY      = 310.0
	megaCX      = 220.0
	megaSlamAge = 6 // frames of the game to each frame of the slam
)

// megaSwimAnim is the shark under the surface.
func megaSwimAnim(alpha float64, up bool) *bhAnim {
	return newBHAnim(megaSwimN, func(i int) (*canvas, float64, float64) {
		c := hdBossCanvas(300, 130)
		megaSwimOn(c, 150, 70, float64(i)/megaSwimN, alpha, up)
		return hdCrop(c, 150, 70)
	})
}

// hdOut finishes a frame of a beast out of the water at waterline wy, painted
// on c about cx: its outline, the water it stands in and the foam it boils up.
// ph boils the foam; burst (0..1, or outside that for none) is how far a
// splash thrown up as it broke the surface has gone, power art pixels high.
func hdOut(c *canvas, wy, cx, depth, ph, burst, power float64, seed uint32) {
	c.outline(0.8)
	// Where its body crosses the waterline.
	_, py := c.toPx(0, wy)
	row := int(py) - 1
	x0, x1 := -1.0, -1.0
	for x := 0; x < c.w; x++ {
		if c.px[row*c.w+x].A > 200 {
			if x0 < 0 {
				x0 = c.lx(x)
			}
			x1 = c.lx(x)
		}
	}
	hdSink(c, wy, depth)
	if x0 < 0 {
		if burst > 0 && burst < 1 {
			hdRing(c, cx, wy+2, 30+60*burst, 4+7*burst, burst, seed)
			hdSplash(c, cx, wy, power, power*1.3, burst, int(power*2.2), seed+1)
		}
		return
	}
	mid, half := (x0+x1)/2, (x1-x0)/2
	// Water streaming off it.
	for i := 0; i < int(half*1.4); i++ {
		x := x0 - 6 + hashf(i, 0, seed+2)*(x1-x0+50)
		top := wy - 6 - hashf(i, 1, seed+2)*90
		ix, iy := c.toPx(x, top)
		if !c.opaque(int(ix), int(iy)) {
			continue
		}
		for y := top; y < wy; y += 0.34 {
			jx, jy := c.toPx(x, y)
			if !c.opaque(int(jx), int(jy)) {
				n := 3 + hashf(i, 2, seed+2)*9
				off := math.Mod(ph*n*2+hashf(i, 3, seed+2)*n, n)
				for k := 0.0; k < n && y+k < wy; k += 0.34 {
					if a := 1 - math.Abs(k-off)/3; a > 0 {
						c.dot(x, y+k, 0.28, withAlpha(colSpray, uint8(230*a)))
					}
				}
				break
			}
		}
	}
	hdRing(c, mid, wy+2.5, half+14, 6, 0.15+0.2*math.Sin(2*math.Pi*ph), seed)
	hdRing(c, mid, wy+2.5, half+26+6*ph, 9, 0.6, seed+5)
	hdFoam(c, x0-5, x1+5, wy, 4.2, ph, seed+3)
	hdSplash(c, mid, wy, 16, half+18, math.Mod(ph+0.3, 1), int(half), seed+4)
	if burst > 0 && burst < 1 {
		hdSplash(c, mid, wy, power, half+power*0.8, burst, int(power*2.4), seed+1)
	}
}

func megaOut(c *canvas, ph, burst, power float64, seed uint32) {
	hdOut(c, megaWY, megaCX, 16, ph, burst, power, seed)
}

// megaStrikeAges and megaBreachAges are the ages, in frames at 60 a second,
// the frames of its bite and of its leap show. The bite lands 36 frames in;
// the leap lasts 72.
var (
	megaStrikeAges = [...]int{0, 4, 8, 12, 17, 22, 27, 31, 34, 37, 40, 44, 48, 52}
	megaBreachAges = [...]int{0, 4, 8, 13, 18, 24, 30, 36, 42, 48, 54, 59, 64, 68}
)

var (
	megaSwim    = megaSwimAnim(0.6, false)
	megaSurface = megaSwimAnim(0.62, true)
	megaLurk    = newBHAnim(megaSwimN, func(i int) (*canvas, float64, float64) {
		// Under, circling: only its fin, its wake and a darkness below.
		c := hdBossCanvas(300, 130)
		megaSwimOn(c, 150, 70, float64(i)/megaSwimN, 0.22, false)
		return hdCrop(c, 150, 70)
	})
	megaStrike = newBHAnim(len(megaStrikeAges), func(i int) (*canvas, float64, float64) {
		// Up out of the water side-on, jaws opening; the snap; and down.
		age := float64(megaStrikeAges[i])
		rise := smooth(clamp01(age / 26))
		lunge := smooth(clamp01((age - 30) / 6))
		sink := smooth(clamp01((age - 38) / 16))
		c := hdBossCanvas(megaW, megaH)
		fr := hdFrameAt(c, megaCX-44+18*lunge-8*sink, megaWY+46-54*rise+50*sink, -0.2-0.3*rise+0.12*lunge+0.2*sink, megaK)
		gape := 0.08 + 0.95*smooth(clamp01(age/28))
		if age >= 36 {
			gape = 0.12
		}
		megaSide(c, fr, gape, math.Sin(age*0.35))
		megaOut(c, math.Mod(age/20, 1), (age-34)/20, 34, 975)
		return hdCrop(c, megaCX, megaWY)
	})
	megaRoar = newBHAnim(megaRoarN, func(i int) (*canvas, float64, float64) {
		// Its head straight up out of the sea, jaws wide, shaking.
		w := 2 * math.Pi * float64(i) / megaRoarN
		c := hdBossCanvas(megaW, megaH)
		fr := hdFrameAt(c, megaCX-14+3*math.Sin(w), megaWY+6+3*math.Cos(w), -1.12+0.07*math.Sin(w), megaK)
		megaSide(c, fr, 1.12+0.1*math.Sin(2*w), math.Sin(w))
		megaOut(c, float64(i)/megaRoarN, -1, 0, 976)
		return hdCrop(c, megaCX+4, megaWY)
	})
	megaBreach = newBHAnim(len(megaBreachAges), func(i int) (*canvas, float64, float64) {
		// The leap: up out of the water nose first, over, and down on its prey.
		u := float64(megaBreachAges[i]) / 72
		c := hdBossCanvas(megaW, megaH)
		y := megaWY + 40 - 150*4*u*(1-u)
		fr := hdFrameAt(c, megaCX-10+20*u, y, -1.05+1.85*smooth(u), megaK)
		megaSide(c, fr, 1.05*(1-smooth(clamp01((u-0.75)/0.2)))+0.1, math.Sin(u*9))
		megaOut(c, math.Mod(u*3, 1), u*1.6, 46, 977)
		return hdCrop(c, megaCX, megaWY)
	})
	megaSlam = newBHAnim(megaSlamN, func(i int) (*canvas, float64, float64) {
		// It has come down: the sea thrown up round it, falling back in rings.
		t := (float64(i) + 0.5) / megaSlamN
		c := hdBossCanvas(megaW, megaH)
		megaSwimOn(c, megaCX, megaWY, t*1.5, 0.62, true)
		for k, r := range []float64{0, 0.25, 0.5} {
			if f := t*1.3 - r; f > 0 && f < 1 {
				hdRing(c, megaCX, megaWY+3, 40+110*f, 8+22*f, f, uint32(978+k))
			}
		}
		hdSplash(c, megaCX+10, megaWY, 86, 120, t, 260, 981)
		hdSplash(c, megaCX-40, megaWY, 50, 70, t*1.25, 110, 982)
		return hdCrop(c, megaCX, megaWY)
	})
)

// megaFrameAt is the last frame of ages that age has reached.
func megaFrameAt(ages []int, age int) int {
	f := 0
	for i, a := range ages {
		if age >= a {
			f = i
		}
	}
	return f
}

// spinoLurkCanvas is the Spinosaurus under the swamp's water: its sail, the
// top of its skull with the eye, and the end of its snout.
func spinoLurkCanvas() (*canvas, float64, float64) {
	const W, H, wy = 150, 70, 56
	c := newCanvas(W, H)
	spinoSailOn(c, 10, wy+7, 82, wy+7, 4, 38)
	pelt := spinoLook.pelt(theroHead, 0.04)
	bossPaint(c, 0, pelt, []knob{{84, wy + 2, 6.5}, {98, wy - 0.5, 7}, {107, wy + 1.5, 5.5}, {120, wy + 3, 4.6}})
	bossPaint(c, 0, pelt, []knob{{134, wy + 1.5, 4.6}, {140, wy + 2, 3.8}})
	fr := bossFrame{ox: 98, oy: wy - 1}
	fr.poly(c, [][2]float64{{6, -6}, {9, -9.5}, {13, -4.5}}, func(x, y int) color.RGBA { return spinoSail.pick(0.55, x, y) })
	for q := -1; q <= 1; q++ {
		for w := -1; w <= 2; w++ {
			c.put(101+w, wy-4+q, spinoSkin[0])
		}
	}
	c.put(101, wy-4, spinoLook.eye)
	c.put(102, wy-4, colInk)
	c.put(101, wy-5, mix(spinoLook.eye, colWhite, 0.5))
	bossLine(c, 97, float64(wy)-7.5, 105, float64(wy)-6.5, spinoSkin[1])
	c.put(138, wy-2, spinoSkin[0])
	c.outline(0.78)
	sinkBelow(c, wy)
	// Ripples where it breaks the still water.
	for _, r := range [][2]int{{8, 84}, {80, 126}, {128, 145}} {
		for x := r[0]; x <= r[1]; x++ {
			switch h := hashf(x, wy, 969); {
			case h < 0.3:
				c.put(x, wy, colFoam)
			case h < 0.75:
				c.put(x, wy, colFoamDim)
			}
			if hashf(x, wy+1, 970) < 0.2 {
				c.put(x, wy+1, withAlpha(colFoamDim, 190))
			}
		}
	}
	return bossCrop(c, 76, wy)
}

// megaBeachN is how many frames the shark's lunge onto the reef has.
const megaBeachN = 10

// megaBeach is the shark hurled out over the edge of its lagoon (the lunge
// of sim/sharkfight.go): frames 0 to 3 its body coming up out of the water
// along the reef, jaws wide; 4 to 9 lying beached on its belly, thrashing,
// jaws snapping, the sea streaming off it.
var megaBeach = newBHAnim(megaBeachN, func(i int) (*canvas, float64, float64) {
	u := float64(i) / (megaBeachN - 1)
	rise := smooth(clamp01(u / 0.35))
	c := hdBossCanvas(megaW, megaH)
	thrash := math.Sin(float64(i) * 1.9)
	// Lying low along the surface, the head over the reef, a little down.
	fr := hdFrameAt(c, megaCX+10+16*rise, megaWY+30-26*rise+2*thrash*(1-rise*0.3), 0.06-0.22*(1-rise)+0.04*thrash, megaK)
	gape := 0.9 - 0.5*clamp01((u-0.4)/0.6) + 0.25*math.Max(0, math.Sin(float64(i)*2.4))
	megaSide(c, fr, gape, thrash*1.2)
	hdOut(c, megaWY, megaCX, 7, math.Mod(u*2.5, 1), u*1.8, 30, 990)
	return hdCrop(c, megaCX, megaWY)
})
