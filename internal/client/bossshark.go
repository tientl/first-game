package client

import (
	"image/color"
	"math"
)

// The Megalodon lives in the water like the sea monsters of seaart.go and is
// drawn the same way: swimming, it is a dark shape under the surface with
// its fin cutting the water; when it bites, bellows or breaches, it comes
// out side-on, cut off at the waterline in a ring of foam. Every pose is
// anchored on the waterline. The lurking Spinosaurus is here too.

var (
	megaSkin   = mkRamp(0x0e1620, 0x182634, 0x24384a, 0x344e62, 0x4a687c, 0x668698, 0x88a4b0)
	megaBelly  = mkRamp(0x7c8a96, 0xa8b6be, 0xd0dadc, 0xecf2f0, 0xffffff)
	megaShadow = hex(0x07131f)
)

// megaMouth is the line of the upper jaw in the shark's frame: v at u.
func megaMouth(u float64) float64 { return 8.5 - (u-megaHingeU)*0.24 }

const megaHingeU, megaHingeV = 39.0, 8.5

// megaSide paints the shark side-on in frame fr (u from the root of the tail
// at -68 to the tip of the snout at 70, v down), its jaws gape wide.
func megaSide(c *canvas, fr bossFrame, gape float64) {
	fin := func(q [][2]float64, bias float64) {
		fr.poly(c, q, func(x, y int) color.RGBA {
			u, v := fr.local(x, y)
			return megaSkin.pick(0.36+bias-v*0.004+u*0.001+bossScales(x, y, 961)*0.6, x, y)
		})
	}
	// The tail's crescent, the fins of the back.
	fin([][2]float64{{-62, -3}, {-74, -14}, {-88, -30}, {-80, -9}, {-76, 0}, {-83, 16}, {-86, 22}, {-72, 12}, {-62, 3}}, 0)
	fin([][2]float64{{14, -15}, {2, -30}, {-8, -39}, {-9, -28}, {-13, -14}}, 0.04)
	fin([][2]float64{{-36, -9}, {-44, -15}, {-45, -8}}, 0)
	fin([][2]float64{{-34, 9}, {-42, 16}, {-44, 9}}, -0.1)
	fin([][2]float64{{30, 8}, {22, 30}, {14, 38}, {17, 12}}, -0.16) // the far pectoral

	hx, hy := fr.at(megaHingeU, megaHingeV)
	fj := bossFrame{hx, hy, fr.a + gape*0.72}
	tx, ty := fr.at(65, megaMouth(65))
	jx, jy := fj.at(22, 0)
	bx, by := fr.at(megaHingeU-6, megaHingeV-2)
	b2x, b2y := fj.at(-6, 6)
	c.polygon([]float64{bx, by, tx, ty, jx, jy, b2x, b2y}, func(x, y int) color.RGBA {
		d := math.Hypot(float64(x)+0.5-hx, float64(y)+0.5-hy)
		return mix(hex(0x1c0610), hex(0xc0505a), clamp01((d-5)/20)-dither(x, y)*0.2)
	})
	skin := func(x, y int, q bossPt, v float64) color.RGBA {
		l := 0.1 + q.l*0.84 + (hashf(x, y, 962)-0.5)*0.05
		if v > 0.2+0.07*math.Sin(q.s*0.2)+dither(x, y)*0.12 {
			return megaBelly.soft(l+0.05-max(0, v-0.5)*0.3, x, y)
		}
		return megaSkin.soft(l+0.02, x, y)
	}
	// The lower jaw.
	bossPaint(c, 0, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if v < 0 && u > -2 {
			return color.RGBA{}
		}
		if v < 1.6 && u > 0 {
			return hex(0xd8808a) // the gum
		}
		return megaBelly.soft(0.2+q.l*0.75, x, y)
	}, fj.knobs([]knob{{-5, 4.5, 7.5}, {8, 3.6, 5.6}, {20, 2.6, 3.4}}))
	// The body.
	bossPaint(c, 0.2, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > megaHingeU && v > megaMouth(u) {
			return color.RGBA{}
		}
		if u > megaHingeU+1 && v > megaMouth(u)-1.6 {
			return hex(0xd8808a)
		}
		if u > 22 && u < 36 && int(u-22)%4 == 0 && v > -7 && v < 7 {
			return megaSkin[0] // the gills
		}
		return skin(x, y, q, q.v)
	}, fr.knobs([]knob{{-68, 0, 3.2}, {-50, 0, 7}, {-25, 0, 13}, {0, 0, 17}, {22, 0.5, 17.5}, {40, 1, 15.5}, {52, 0.5, 12.5}, {62, -1.5, 8}, {70, -3, 3.6}}))
	// Teeth: broad triangles, a row to each jaw.
	white := func(x, y int) color.RGBA { return colTooth }
	for u := megaHingeU + 4; u < 64; u += 3.2 {
		m := megaMouth(u)
		fr.poly(c, [][2]float64{{u - 1.7, m - 0.4}, {u + 1.7, m - 0.4}, {u + 0.2, m + 4.6}}, white)
	}
	for u := 3.5; u < 21; u += 3.2 {
		fj.poly(c, [][2]float64{{u - 1.6, 0.6}, {u + 1.6, 0.6}, {u, -3.8}}, white)
	}
	// The near pectoral fin.
	fr.poly(c, [][2]float64{{27, 9}, {17, 33}, {6, 44}, {8, 30}, {12, 11}}, func(x, y int) color.RGBA {
		u, v := fr.local(x, y)
		return megaSkin.pick(0.5-(v-9)*0.008+(u-6)*0.006, x, y)
	})
	// The eye: black and blank.
	ex, ey := fr.at(51, -3.5)
	ix, iy := int(math.Floor(ex)), int(math.Floor(ey))
	for q := -1; q <= 2; q++ {
		for w := -1; w <= 2; w++ {
			if (q == -1 || q == 2) && (w == -1 || w == 2) {
				continue
			}
			c.put(ix+w, iy+q, megaSkin[0])
		}
	}
	c.put(ix, iy, colBlack)
	c.put(ix+1, iy, colBlack)
	c.put(ix, iy+1, colBlack)
	c.put(ix+1, iy+1, colBlack)
	c.put(ix, iy, hex(0xdfe8f0))
	nx, ny := fr.at(66, -4.5)
	c.put(int(nx), int(ny), megaSkin[0])
	// An old scar across the snout and one on the flank.
	for _, q := range [][4]float64{{54, -9, 60, -4}, {57, -10, 62, -6}, {-6, -8, 6, -2}, {-2, -11, 8, -6}} {
		x0, y0 := fr.at(q[0], q[1])
		x1, y1 := fr.at(q[2], q[3])
		bossLine(c, x0, y0, x1, y1, megaSkin[6])
	}
}

// megaFinOn stands the back fin on the waterline wy with its middle at x.
func megaFinOn(c *canvas, x float64, wy int, seed uint32) {
	f := newCanvas(c.w, c.h)
	w := float64(wy)
	f.polygon([]float64{x + 13, w + 2, x + 1, w - 15, x - 9, w - 25, x - 10, w - 13, x - 14, w + 2}, func(px, py int) color.RGBA {
		return megaSkin.pick(0.62-(float64(px)-x+14)*0.012-(float64(py)-w+25)*0.004+bossScales(px, py, 963)*0.6, px, py)
	})
	f.put(int(x-10), wy-14, colInk) // a notch torn out of it
	f.put(int(x-11), wy-14, color.RGBA{})
	f.outline(0.8)
	sinkBelow(f, wy)
	c.over(f, 0, 0)
	seaFoam(c, int(x-17), int(x+17), wy, seed)
	for i := 0; i < 9; i++ { // its wake
		qx := x - 16 - float64(i)*3.2 - hashf(i, 0, seed)*2
		c.put(int(qx), wy+int(hashf(i, 1, seed)*3)-1, withAlpha(colFoamDim, 210))
	}
	c.put(int(x+17), wy-1, colFoam)
}

// megaSwim is the shark just under the surface: its shape from above, dark
// in the water, the tail swung by wag, and the fin above it.
func megaSwim(wag float64) (*canvas, float64, float64) {
	const W, H, cy = 172, 74, 44.0
	t := newCanvas(W, H)
	flat := func(int, int, *spinePt, float64) color.RGBA { return megaShadow }
	width := func(s float64) float64 {
		switch {
		case s < 0.12:
			return lerpf(1.2, 3.4, s/0.12)
		case s < 0.5:
			return lerpf(3.4, 10.5, smooth((s-0.12)/0.38))
		case s < 0.72:
			return lerpf(10.5, 10, (s-0.5)/0.22)
		}
		return lerpf(10, 2.6, math.Pow((s-0.72)/0.28, 1.6))
	}
	const n = 150
	pts := make([]spinePt, 0, n+1)
	at := func(s float64) (float64, float64) {
		back := 1 - s
		return 12 + s*146, cy + wag*9*math.Sin(1.2+3.4*back)*math.Pow(back, 1.3)
	}
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		x, y := at(s)
		pts = append(pts, spinePt{x: x, y: y, r: width(s), s: s})
	}
	drawTubeWith(t, withNormals(pts), flat)
	// The pectoral fins swept back, and the blade of the tail.
	fx, fy := at(0.66)
	for _, side := range []float64{-1, 1} {
		t.polygon([]float64{fx + 6, fy + side*8, fx - 22, fy + side*25, fx - 26, fy + side*23, fx - 10, fy + side*8}, solid(megaShadow))
		px, py := at(0.3)
		t.polygon([]float64{px + 4, py + side*4, px - 7, py + side*10, px - 4, py + side*4}, solid(megaShadow))
	}
	tx, ty := at(0)
	t2x, t2y := at(0.06)
	t.polygon([]float64{t2x, t2y - 2, tx - 9, ty + wag*5 - 3, tx - 10, ty + wag*5 + 2, t2x, t2y + 2}, solid(megaShadow))
	c := newCanvas(W, H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if !t.opaque(x, y) {
				continue
			}
			n := 0
			for _, d := range [8][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}, {0, 2}, {0, -2}, {-2, 0}, {2, 0}} {
				if t.opaque(x+d[0], y+d[1]) {
					n++
				}
			}
			a := uint8(138)
			if n < 8 {
				a = 92
			}
			if n < 6 {
				a = 56
			}
			c.put(x, y, withAlpha(megaShadow, a))
		}
	}
	dx, dy := at(0.56)
	megaFinOn(c, dx, int(dy), 964)
	// The tip of the tail breaks the surface too.
	f := newCanvas(W, H)
	f.polygon([]float64{tx + 4, ty + 1, tx - 5, ty - 9 + wag, tx - 8, ty - 6 + wag, tx - 4, ty + 1}, func(px, py int) color.RGBA {
		return megaSkin.pick(0.45, px, py)
	})
	f.outline(0.8)
	sinkBelow(f, int(ty))
	c.over(f, 0, 0)
	seaFoam(c, int(tx-7), int(tx+6), int(ty), 965)
	// The swell it pushes ahead of its snout.
	sx, sy := at(1)
	for i := -3; i <= 3; i++ {
		c.put(int(sx)+5-absInt(i), int(sy)+i*2, withAlpha(colFoamDim, 200))
	}
	return bossCrop(c, 12+146*0.5, cy)
}

func megalodonCanvas(p bossPose) (*canvas, float64, float64) {
	switch p {
	case bossIdle:
		return megaSwim(0.25)
	case bossWalkA:
		return megaSwim(1)
	case bossWalkB:
		return megaSwim(-1)
	case bossLurk:
		c := newCanvas(64, 40)
		megaFinOn(c, 36, 30, 964)
		return bossCrop(c, 34, 30)
	}
	const W, H, cx, wy = 340, 280, 170.0, 220
	c := newCanvas(W, H)
	fr, gape := bossFrame{cx - 34, wy - 5, -0.46}, 0.95
	switch p {
	case bossRoar:
		fr, gape = bossFrame{cx - 14, wy + 16, -1.02}, 1.2
	case bossSpecial:
		fr, gape = bossFrame{cx, wy - 36, -0.62}, 1.05
	}
	megaSide(c, fr, gape)
	c.outline(0.8)
	sinkBelow(c, wy)
	// Where its body breaks the water, the sea boils.
	u0 := (float64(wy) - fr.oy) / math.Sin(fr.a)
	rx := fr.ox + u0*math.Cos(fr.a)
	ringW := 30.0
	if p == bossSpecial {
		ringW = 40
	}
	seaRing(c, rx, wy, ringW, 4.5, 966)
	drops := 30
	if p == bossSpecial {
		drops = 70
	}
	for i := 0; i < drops; i++ {
		qx := rx + (hashf(i, 0, 967)-0.5)*ringW*2.6
		h := 2 + hashf(i, 1, 967)*hashf(i, 2, 967)*float64(drops)*0.9
		if c.at(int(qx), wy-int(h)).A > 0 {
			continue
		}
		col := colFoam
		if i%3 == 0 {
			col = colFoamDim
		}
		c.put(int(qx), wy-int(h), col)
		if i%4 == 0 {
			c.put(int(qx), wy-int(h)+1, colFoamDim)
		}
	}
	// Water streaming off its belly.
	for i := 0; i < 9; i++ {
		u := -30 + float64(i)*9.5
		x, y := fr.at(u, 17)
		for k := 0; k < 3+int(hashf(i, 3, 968)*5); k++ {
			yy := int(y) + 2 + k
			if yy < wy-1 && c.at(int(x), yy).A == 0 {
				c.put(int(x), yy, withAlpha(mix(colFoamDim, colFoam, float64(k%2)), 220))
			}
		}
	}
	ax := rx + 10
	if p == bossSpecial {
		ax = cx
	}
	return bossCrop(c, ax, float64(wy))
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
	fr := bossFrame{98, wy - 1, 0}
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
