package client

import (
	"image/color"
	"math"
)

// The cooking gear, at HD: the pot on its tripod, the grill and the stone
// pan that go over a campfire, the mortar, the earth oven, the drying rack
// and the clay beds by the water. Drawn in image pixels (ck to an art
// pixel) on canvases marked HD, like the buildings of hdbuild.go. What goes
// over a fire is drawn on a canvas as wide as the campfire's, anchored on
// the middle of the fire, in two layers: what stands behind the flames and
// what hangs in front of them. See cookrender.go for where they are drawn.

const ck = float64(HDDensity)

var (
	clayRamp   = mkRamp(0x4a281e, 0x6e3f2c, 0x94583c, 0xb47752, 0xd09a72, 0xe6bf9a)
	ovenRamp   = mkRamp(0x40241a, 0x633a26, 0x875537, 0xa9744c, 0xc79668, 0xe0b98c)
	brothRamp  = mkRamp(0x6a2c10, 0x98501c, 0xc47a2a, 0xe2a646, 0xf4cf7c)
	saltRamp   = mkRamp(0x9aa6b8, 0xc4ccd8, 0xe4e8ee, 0xffffff)
	greenStick = mkRamp(0x2c3a1c, 0x44562a, 0x607538, 0x80964a, 0xa2b662)
	jerkyRamp  = mkRamp(0x2e120c, 0x4c1e12, 0x6e2e1a, 0x8e4426, 0xb06238)
	breadRamp  = mkRamp(0x6a3c14, 0x96601e, 0xc08a34, 0xdcb05a, 0xf0d48e)
	herbRamp   = mkRamp(0x1e3c18, 0x2f5c22, 0x48802e, 0x6ea63e, 0x9cc85a)
)

// cookArt is the art of the cooking gear.
type cookArt struct {
	potBack, grillBack, panBack sprite    // what stands behind the flames
	pot                         [4]sprite // empty, then three frames of a full pot simmering
	grill, pan                  [2]sprite // bare, laden
	mortar                      [2]sprite // empty, with herbs in it
	oven                        [4]sprite // cold, lit; then the same with a loaf out to cool
	rack                        [5]sprite // bare, meat, fish, bowls of brine, dried strips
	clay                        [2]sprite // a bed of clay, and dug out
	puff                        [3]sprite // steam and smoke
}

// The canvas of what goes over a fire: 30 by 48 art pixels, the fire's
// middle at (15, 40).
const (
	gearW, gearH = 30, 48
	gearFX       = 15 * ck
	gearFY       = 40 * ck
)

func genCookArt() cookArt {
	var a cookArt
	up := func(c *canvas) sprite { return mkUp(c, gearFX, gearFY) }
	back, _ := genPotCanvas(-1)
	a.potBack = up(back)
	for i := range a.pot {
		_, front := genPotCanvas(i - 1)
		a.pot[i] = up(front)
	}
	gb, _ := genGrillCanvas(false)
	a.grillBack = up(gb)
	pb, _ := genPanCanvas(false)
	a.panBack = up(pb)
	for i := range 2 {
		_, gf := genGrillCanvas(i == 1)
		a.grill[i] = up(gf)
		_, pf := genPanCanvas(i == 1)
		a.pan[i] = up(pf)
		c, ax, ay := genMortarCanvas(i == 1)
		a.mortar[i] = mkUp(c, ax, ay)
		c, ax, ay = genClayCanvas(i == 1)
		a.clay[i] = mkUp(c, ax, ay)
	}
	for i := range a.oven {
		c, ax, ay := genOvenCanvas(i%2 == 1, i >= 2)
		a.oven[i] = mkUp(c, ax, ay)
	}
	for i := range a.rack {
		c, ax, ay := genRackCanvas(i)
		a.rack[i] = mkUp(c, ax, ay)
	}
	for i := range a.puff {
		a.puff[i] = genPuff(i)
	}
	return a
}

// ckTex is the grain of a hand-made surface at (x, y): fine speckle and
// broader blotches, around zero.
func ckTex(x, y int, seed uint32, fine, coarse float64) float64 {
	return (hashf(x, y, seed)-0.5)*fine + (vnoise(float64(x)*0.21, float64(y)*0.21, seed+7)-0.5)*coarse
}

// ckPole is a straight pole of rp from (x0, y0) to (x1, y1), w pixels half
// thick at the foot and a little less at the top.
func ckPole(c *canvas, x0, y0, x1, y1, w float64, rp ramp, lit float64, seed uint32) {
	drawLimbHD(c, limb{x0, y0, (x0 + x1) / 2, (y0 + y1) / 2, x1, y1, w, w * 0.8}, rp, lit, seed)
}

// ckLash winds cord round whatever is opaque about (x, y): n turns, each
// rising to the right, r pixels half wide.
func ckLash(c *canvas, x, y, r float64, n int) {
	for i := 0; i < n; i++ {
		yy := y + float64(i)*2.4 - float64(n)*1.2
		for dx := -r; dx <= r; dx++ {
			px, py := int(x+dx), int(yy-dx*0.25)
			if !c.opaque(px, py) {
				continue
			}
			c.put(px, py, ropeRamp.at(2-int(math.Abs(dx)/r*1.6)))
			if c.opaque(px, py+1) {
				c.put(px, py+1, ropeRamp.at(0))
			}
		}
	}
}

// ckCord draws a thin cord from (x0, y0) to (x1, y1), two pixels wide.
func ckCord(c *canvas, x0, y0, x1, y1 float64) {
	n := int(math.Hypot(x1-x0, y1-y0)*2) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := int(lerpf(x0, x1, t)), int(lerpf(y0, y1, t))
		c.put(x, y, ropeRamp.at(1+(i/5)%2))
		c.put(x+1, y, ropeRamp.at(0))
	}
}

// ckStick is a thin peeled stick from (x0, y0) to (x1, y1), three pixels
// across: a lit side, the wood, a shaded side. tone is its middle shade.
func ckStick(c *canvas, x0, y0, x1, y1 float64, rp ramp, tone int) {
	n := int(math.Hypot(x1-x0, y1-y0)*2) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := int(math.Round(lerpf(x0, x1, t))), int(math.Round(lerpf(y0, y1, t)))
		knot := 0
		if hashf(y/3, int(x0), 3129) < 0.12 {
			knot = 1
		}
		c.put(x-1, y, rp.at(tone+1-knot))
		c.put(x, y, rp.at(tone-knot))
		c.put(x+1, y, rp.at(tone-2))
	}
}

// potHD draws a round-bellied clay pot with its middle at (px, py), 6.5 art
// pixels half wide: a flared lip, two lugs, a band scratched round the
// shoulder, and soot up its belly from the fire. broth is -1 for an empty
// pot, else which frame of the simmering soup shows in its mouth.
func potHD(c *canvas, px, py float64, broth int) {
	var rx, ry = 6.5 * ck, 5.2 * ck
	rimY := py - ry + 2.4
	// The lugs, behind the belly.
	for _, s := range []float64{-1, 1} {
		c.ellipse(px+s*(rx-0.5), py-4, 4.2, 5, func(x, y int, nx, ny float64) color.RGBA {
			if nx*nx+ny*ny < 0.2 {
				return color.RGBA{} // the hole the cord goes through
			}
			return potRamp.pick(0.55-s*nx*0.25-ny*0.2, x, y)
		})
	}
	// The belly.
	c.ellipse(px, py, rx, ry, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx*0.92, ny*0.85)*0.92 + 0.06 + ckTex(x, y, 3101, 0.07, 0.16)
		// The band round the shoulder: two scratched lines, a zigzag between.
		v := ny + 0.16*nx*nx
		switch {
		case math.Abs(v+0.36) < 0.035 || math.Abs(v+0.06) < 0.035:
			l -= 0.2
		case v > -0.36 && v < -0.06:
			u := math.Asin(max(-1, min(1, nx))) * 4.2
			if z := math.Abs(math.Mod(u+8, 1)-0.5)*2*0.3 - 0.36; math.Abs(v-z-0.02) < 0.04 {
				l -= 0.17
			}
		}
		col := potRamp.pick(l, x, y)
		// Soot, thick under the belly and licking up the sides.
		if s := (ny-0.1)*1.5 + (vnoise(float64(x)*0.16, float64(y)*0.09, 3102)-0.5)*0.7; s > 0 {
			col = mix(col, ashRamp.pick(0.2+l*0.6, x, y), min(1, s*1.25))
		}
		return col
	})
	// The neck and the lip.
	for y := int(rimY) - 1; y < int(rimY)+7; y++ {
		half := 13.2 + float64(y-int(rimY))*0.35
		for x := int(px - half); x < int(px+half); x++ {
			u := (float64(x) + 0.5 - px) / half
			c.put(x, y, potRamp.pick(0.8-(u+1)/2*0.55+ckTex(x, y, 3103, 0.07, 0.1), x, y))
		}
	}
	c.ellipse(px, rimY, 15, 4.6, func(x, y int, nx, ny float64) color.RGBA {
		return potRamp.pick(0.95-(nx+1)/2*0.45-ny*0.15+ckTex(x, y, 3104, 0.06, 0), x, y)
	})
	c.ellipse(px, rimY+0.3, 12.4, 3.3, func(x, y int, nx, ny float64) color.RGBA {
		if broth < 0 {
			return mix(hex(0x1a0d08), potRamp[0], max(0, ny)*0.7+max(0, nx)*0.2) // the dark inside
		}
		l := 0.55 - nx*0.25 - ny*0.2 + (vnoise(float64(x)*0.3+float64(broth)*5, float64(y)*0.5, 3105)-0.5)*0.5
		col := brothRamp.pick(l, x, y)
		switch h := hashf(x/2, y/2, 3106+uint32(broth)*13); {
		case h < 0.07:
			col = herbRamp.at(2 + x%2) // a leaf turning over
		case h < 0.12:
			col = roastRamp.at(2 + y%2) // a piece of meat
		case h > 0.94:
			col = brothRamp.at(4) // a bubble's glint
		}
		if ny < -0.55 {
			col = mix(col, potRamp[0], 0.55) // the lip's shadow on the soup
		}
		return col
	})
}

// genPotCanvas is the pot over the fire: three poles lashed at the top, the
// pot hung from them by two cords. The pole at the back is on back, the
// rest on front.
func genPotCanvas(broth int) (back, front *canvas) {
	back, front = hdCanvas(gearW, gearH), hdCanvas(gearW, gearH)
	var ax, ay = gearFX + 1, gearFY - 36*ck // where the poles cross
	over := func(fx, fy float64) (float64, float64) {
		return ax + (ax-fx)*0.13, ay + (ay-fy)*0.13 // each pole runs on a little past the lashing
	}
	bx, by := over(gearFX+7*ck, gearFY-6*ck)
	ckPole(back, gearFX+7*ck, gearFY-6*ck, bx, by, 3.4, trunkRamp, 0.72, 3110)
	back.outline(0.75)

	pcx, pcy := gearFX, gearFY-15*ck
	lx, ly := over(gearFX-12.5*ck, gearFY+2.5*ck)
	rx, ry := over(gearFX+12.5*ck, gearFY+2.5*ck)
	ckPole(front, gearFX+12.5*ck, gearFY+2.5*ck, rx, ry, 3.6, trunkRamp, 0.8, 3112)
	ckCord(front, ax-1, ay+5, pcx-19, pcy-7)
	ckCord(front, ax+1, ay+5, pcx+18, pcy-7)
	potHD(front, pcx, pcy, broth)
	ckPole(front, gearFX-12.5*ck, gearFY+2.5*ck, lx, ly, 3.6, trunkRamp, 0.92, 3111)
	ckLash(front, ax, ay+1, 7, 4)
	front.outline(0.75)
	return back, front
}

// roastHD is a piece of meat on the grill or the pan at (x, y): a browned
// chop with a rim of fat, the marks of the heat across it, and a glint.
func roastHD(c *canvas, x, y, rx, ry float64, seed uint32) {
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		l := lum(nx*0.7, ny*0.7)*0.8 + 0.1 + ckTex(px, py, seed, 0.1, 0.2)
		if (px+py*2+int(seed))%9 < 2 {
			l -= 0.3 // seared stripes
		}
		col := roastRamp.pick(l, px, py)
		if d > 0.72 && nx < 0.2 {
			col = mix(col, hex(0xf2d9a6), 0.75) // fat along the lit edge
		}
		return col
	})
	c.put(int(x-rx*0.4), int(y-ry*0.45), hex(0xfff0c8))
	c.put(int(x-rx*0.4)+1, int(y-ry*0.45), hex(0xfff0c8))
}

// grilledFishHD is a whole fish lying at (x, y), l pixels half long, head to the
// left, grilled gold with dark bars.
func grilledFishHD(c *canvas, x, y, l, w float64, seed uint32) {
	// The tail.
	c.polygon([]float64{x + l*0.7, y, x + l*1.25, y - w*1.1, x + l*1.1, y, x + l*1.25, y + w*1.1}, func(px, py int) color.RGBA {
		return roastRamp.pick(0.45+ckTex(px, py, seed, 0.2, 0), px, py)
	})
	c.ellipse(x, y, l, w, func(px, py int, nx, ny float64) color.RGBA {
		lit := 0.82 - ny*0.35 - nx*0.1 + ckTex(px, py, seed+1, 0.08, 0.12)
		col := mix(breadRamp.pick(lit, px, py), fishRamp.pick(lit, px, py), 0.3)
		if nx > -0.45 && (px+int(seed))%7 < 2 {
			col = roastRamp.pick(lit*0.5, px, py) // the bars of the grill
		}
		if nx < -0.5 && ny > -0.2 && ny < 0.5 && nx > -0.62 {
			col = roastRamp.at(1) // the gill
		}
		return col
	})
	c.put(int(x-l*0.74), int(y-w*0.25), hex(0x1a1014))
	c.put(int(x-l*0.74), int(y-w*0.25)-1, hex(0xf8f0d8))
}

// genGrillCanvas is a rack of green sticks on four forked stakes over the
// fire, bare or laden with a chop, a fish and a skewer.
func genGrillCanvas(laden bool) (back, front *canvas) {
	back, front = hdCanvas(gearW, gearH), hdCanvas(gearW, gearH)
	var (
		nearY = gearFY - 7*ck // the rails the sticks lie on
		farY  = gearFY - 12*ck
		nearX = 12.5 * ck // half the rack's width at the near rail
		farX  = 11.3 * ck
	)
	fork := func(c *canvas, x, foot, top float64, lit float64, seed uint32) {
		ckPole(c, x, foot, x, top+5, 3, trunkRamp, lit, seed)
		ckPole(c, x, top+6, x-4, top-5, 1.9, trunkRamp, lit, seed+1)
		ckPole(c, x, top+6, x+4, top-5, 1.9, trunkRamp, lit-0.1, seed+2)
	}
	// The far stakes, behind the flames.
	fork(back, gearFX-farX, gearFY-4*ck, farY, 0.7, 3120)
	fork(back, gearFX+farX, gearFY-4*ck, farY, 0.62, 3123)
	back.outline(0.75)

	// The far rail, then the sticks laid across from it to the near rail,
	// with the gaps the flames come through.
	ckPole(front, gearFX-farX-5, farY+1, gearFX+farX+5, farY, 2, greenStick, 0.62, 3126)
	var n = 13
	for i := 0; i < n; i++ {
		f := float64(i)/float64(n-1)*2 - 1
		x0, x1 := gearFX+f*(farX-2)+(hashf(i, 1, 3127)-0.5)*1.5, gearFX+f*(nearX-2)+(hashf(i, 2, 3127)-0.5)*1.5
		ckStick(front, x1, nearY+2+hashf(i, 3, 3127)*2, x0, farY-1-hashf(i, 4, 3127)*2.5, greenStick, 3-i%2)
	}
	if laden {
		roastHD(front, gearFX-7*ck, gearFY-9.6*ck, 10, 5.4, 3141)
		grilledFishHD(front, gearFX+2.4*ck, gearFY-10.6*ck, 11, 3.4, 3142)
		// A skewer of mushrooms and tubers.
		sx, sy := gearFX+2.5*ck, gearFY-7.9*ck
		ckPole(front, sx-4, sy+1, sx+26, sy-3, 1, woodRamp, 0.8, 3143)
		for i := 0; i < 4; i++ {
			rp := roastRamp
			if i%2 == 1 {
				rp = breadRamp
			}
			bx, by := sx+2+float64(i)*5.6, sy-float64(i)*0.75
			c := front
			c.ellipse(bx, by, 3.1, 3.4, func(px, py int, nx, ny float64) color.RGBA {
				return rp.pick(lum(nx, ny)*0.85+0.1+ckTex(px, py, 3144, 0.1, 0), px, py)
			})
		}
	}
	// The near rail on its stakes.
	ckPole(front, gearFX-nearX-6, nearY+1, gearFX+nearX+6, nearY, 2.4, greenStick, 0.9, 3150)
	fork(front, gearFX-nearX, gearFY+3*ck, nearY, 0.9, 3151)
	fork(front, gearFX+nearX, gearFY+3*ck, nearY, 0.78, 3154)
	ckLash(front, gearFX-nearX, nearY+2, 5, 3)
	ckLash(front, gearFX+nearX, nearY+2, 5, 3)
	front.outline(0.75)
	return back, front
}

// genGrillLong is the grill turned to run north–south: its rails on the
// forked stakes go back up the screen either side of the fire, and the
// green sticks lie across them from side to side, one behind another; the
// far stakes stand behind the flames, the near ones before them.
func genGrillLong(laden bool) (back, front *canvas) {
	back, front = hdCanvas(gearW, gearH), hdCanvas(gearW, gearH)
	var (
		nearY = gearFY - 4*ck // the near and far ends of the rails, up on the stakes
		farY  = gearFY - 11.5*ck
		halfX = 8.6 * ck // the rails' distance from the middle
		lift  = 7 * ck // how high they stand
	)
	fork := func(c *canvas, x, foot float64, lit float64, seed uint32) {
		top := foot - lift
		ckPole(c, x, foot, x, top+5, 3, trunkRamp, lit, seed)
		ckPole(c, x, top+6, x-4, top-5, 1.9, trunkRamp, lit, seed+1)
		ckPole(c, x, top+6, x+4, top-5, 1.9, trunkRamp, lit-0.1, seed+2)
	}
	fork(back, gearFX-halfX, farY+lift, 0.7, 3300)
	fork(back, gearFX+halfX, farY+lift, 0.62, 3303)
	back.outline(0.75)
	// The rails, going back, and the sticks across them from the back.
	for _, s := range []float64{-1, 1} {
		ckPole(front, gearFX+s*halfX, farY-2, gearFX+s*halfX, nearY+2, 2.2, greenStick, 0.8-0.1*s, 3306)
	}
	const n = 9
	for i := 0; i < n; i++ {
		y := farY + 1 + (nearY-farY-2)*float64(i)/float64(n-1) + (hashf(i, 1, 3307)-0.5)*1.5
		x0, x1 := gearFX-halfX-2-hashf(i, 2, 3307)*3, gearFX+halfX+2+hashf(i, 3, 3307)*3
		ckPole(front, x0, y+0.5, x1, y-0.5, 1.4, greenStick, 0.62+float64(i%2)*0.12, 3308+uint32(i))
	}
	if laden {
		roastHD(front, gearFX-3.4*ck, gearFY-11.4*ck, 10, 5.4, 3141)
		grilledFishHD(front, gearFX+1.6*ck, gearFY-7.8*ck, 11, 3.4, 3142)
	}
	fork(front, gearFX-halfX, nearY+lift, 0.9, 3320)
	fork(front, gearFX+halfX, nearY+lift, 0.78, 3323)
	ckLash(front, gearFX-halfX, nearY-lift+3, 5, 3)
	ckLash(front, gearFX+halfX, nearY-lift+3, 5, 3)
	front.outline(0.75)
	return back, front
}


// genPanCanvas is a flat slab of stone, hollowed a little, propped over the
// fire on three stones; laden, an egg and a chop fry on it.
func genPanCanvas(laden bool) (back, front *canvas) {
	back, front = hdCanvas(gearW, gearH), hdCanvas(gearW, gearH)
	// The props: one behind the flames, two before them at the sides.
	stoneHD(back, gearFX+1*ck, gearFY-6.5*ck, 3.4*ck, 3*ck, 0.8, stoneRamp, 3160)
	back.outline(0.75)
	stoneHD(front, gearFX-9.5*ck, gearFY-2.5*ck, 3.3*ck, 3.6*ck, 0.95, stoneRamp, 3161)
	stoneHD(front, gearFX+9.5*ck, gearFY-2.5*ck, 3.3*ck, 3.6*ck, 0.8, stoneRamp, 3162)
	var sx, sy, rx, ry, thick = gearFX, gearFY - 9.5*ck, 12.5 * ck, 4.4 * ck, 2.2 * ck
	// The slab's edge, sooted underneath, then its face.
	for i := int(thick); i >= 1; i-- {
		f := float64(i) / thick
		front.ellipse(sx, sy+float64(i), rx, ry, func(x, y int, nx, ny float64) color.RGBA {
			l := 0.5 - (nx+1)/2*0.35 - f*0.25 + ckTex(x, y, 3163, 0.08, 0.15)
			return mix(stoneRamp.pick(l, x, y), ashRamp[0], f*0.6)
		})
	}
	front.ellipse(sx, sy, rx, ry, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		l := 0.78 - (nx+1)/2*0.3 - ny*0.12 + ckTex(x, y, 3164, 0.07, 0.2)
		switch {
		case d > 0.8:
			l += 0.12 - ny*0.1 // the rim, pecked round
		case d > 0.68:
			l -= 0.14 + ny*0.1 // the step down into the hollow
		default:
			l -= 0.08
			if vnoise(float64(x)*0.12, float64(y)*0.3, 3165) > 0.62 {
				l += 0.14 // grease catching the light
			}
		}
		if hashf(x/2, y/2, 3166) < 0.025 {
			l -= 0.2
		}
		return stoneRamp.pick(l, x, y)
	})
	if laden {
		// A fried egg: the white spread out, crisp at the edge, the yolk proud.
		ex, ey := sx-4.6*ck, sy-0.2*ck
		for i, b := range [][4]float64{{0, 0, 9, 4.6}, {-5, 1, 5, 3}, {5, -1.5, 5.5, 3}, {2, 2, 6, 2.6}} {
			front.ellipse(ex+b[0], ey+b[1], b[2], b[3], func(x, y int, nx, ny float64) color.RGBA {
				d := nx*nx + ny*ny
				col := mix(hex(0xfffdf4), hex(0xe9e2cf), max(0, ny*0.5+nx*0.3)+hashf(x, y, 3167)*0.1)
				if d > 0.78 && i > 0 {
					col = mix(col, breadRamp[2], 0.6) // the frilled, browned edge
				}
				return col
			})
		}
		front.ellipse(ex-0.5, ey-0.6, 3.6, 2.5, func(x, y int, nx, ny float64) color.RGBA {
			return mix(hex(0xffd23c), hex(0xe88a14), clamp01(0.35+nx*0.4+ny*0.5))
		})
		front.put(int(ex-2), int(ey-1.6), hex(0xfffbe0))
		front.put(int(ex-1), int(ey-1.6), hex(0xfffbe0))
		roastHD(front, sx+5.2*ck, sy+0.1*ck, 10.5, 5.4, 3168)
	}
	front.outline(0.75)
	return back, front
}

// genMortarCanvas is a mortar pecked out of a boulder, its pestle a stout
// stick standing in it; full, crushed herbs show in it and lie about.
func genMortarCanvas(full bool) (*canvas, float64, float64) { return genMortarLean(full, false) }

// genMortarLean is the mortar with its pestle leaning back over the rim to
// the right (as it was drawn), or (front) leaning out over the near rim
// towards the viewer, to the right: see cookFacedSprite.
func genMortarLean(full, front bool) (*canvas, float64, float64) {
	c := hdCanvas(22, 24)
	var cx, top, foot = 11 * ck, 11.5 * ck, 21 * ck
	if !front {
		// The pestle's upper part, behind the rim.
		ckPole(c, cx+1.5*ck, top+1, cx+7.2*ck, 2*ck, 4, woodRamp, 0.95, 3170)
		c.ellipse(cx+7.3*ck, 2*ck, 4.3, 3.6, func(x, y int, nx, ny float64) color.RGBA {
			return woodRamp.pick(0.9-(nx*nx+ny*ny)*0.4, x, y)
		})
	}
	// The body: wide at the lip, drawn in to a foot, pecked all over.
	for y := int(top); y <= int(foot); y++ {
		f := float64(y-int(top)) / (foot - top)
		half := (7.4 - 2.2*f*f + 0.5*math.Sin(f*math.Pi)) * ck
		if f > 0.9 {
			half += (f - 0.9) * 10 * ck * 0.35 // the foot flares
		}
		for x := int(cx - half); x < int(cx+half); x++ {
			u := (float64(x) + 0.5 - cx) / half
			l := 0.9 - (u+1)/2*0.62 - f*0.12 + ckTex(x, y, 3171, 0.1, 0.22)
			if hashf(x/2, y/2, 3172) < 0.08 {
				l -= 0.16 // peck marks
			}
			if math.Abs(u) > 0.9 {
				l -= 0.1
			}
			c.put(x, y, stoneRamp.pick(l, x, y))
		}
	}
	c.ellipse(cx, top, 7.6*ck, 2.9*ck, func(x, y int, nx, ny float64) color.RGBA {
		return stoneRamp.pick(0.98-(nx+1)/2*0.42-ny*0.12+ckTex(x, y, 3173, 0.07, 0.12), x, y)
	})
	c.ellipse(cx, top+0.6, 5.9*ck, 2*ck, func(x, y int, nx, ny float64) color.RGBA {
		if full && ny > -0.35 {
			l := 0.6 - nx*0.3 + ckTex(x, y, 3174, 0.3, 0.4)
			return herbRamp.pick(l, x, y)
		}
		return stoneRamp.pick(0.3+nx*0.14+ny*0.2+ckTex(x, y, 3175, 0.06, 0.1), x, y) // the worn hollow
	})
	if front {
		// The pestle from the hollow out over the near rim, its knob end
		// resting below it to the right, in front.
		ckPole(c, cx-2.4*ck, top-0.6*ck, cx+7.4*ck, top+6.2*ck, 4.2, woodRamp, 0.95, 3170)
		c.ellipse(cx+7.6*ck, top+6.4*ck, 4.4, 3.8, func(x, y int, nx, ny float64) color.RGBA {
			return woodRamp.pick(0.92-(nx*nx+ny*ny)*0.4-ny*0.1, x, y)
		})
	} else {
		// The pestle's foot, in the hollow.
		ckPole(c, cx-0.3*ck, top+3, cx+2.6*ck, top-5*ck+2, 4.2, woodRamp, 0.95, 3170)
	}
	if full {
		for i := 0; i < 7; i++ {
			a := hashf(i, 0, 3176) * math.Pi
			x, y := cx+math.Cos(a)*(9+hashf(i, 1, 3176)*3)*ck*0.95, foot+1-hashf(i, 2, 3176)*5
			if i < 2 {
				x, y = cx-4*ck+float64(i)*7*ck, top-2
			}
			c.ellipse(x, y, 3.4, 1.8, func(px, py int, nx, ny float64) color.RGBA {
				return herbRamp.pick(0.7-nx*0.3-ny*0.3, px, py)
			})
		}
	}
	// Chips of stone at its foot.
	stoneHD(c, cx-8.2*ck, foot+0.5*ck, 2.2*ck, 1.5*ck, 0.9, stoneRamp, 3177)
	stoneHD(c, cx+8.4*ck, foot+0.8*ck, 1.6*ck, 1.2*ck, 0.75, stoneRamp, 3178)
	c.outline(0.75)
	return c, cx, foot
}

// genOvenCanvas is an earth oven: a dome of clay daubed over a ring of
// stones, an arched mouth in front, a smoke hole in the crown and a peel
// leaning against it. Lit, the mouth glows; loaf sets a loaf to cool on the
// stone by the mouth.
func genOvenCanvas(lit, loaf bool) (*canvas, float64, float64) { return genOvenFacing(lit, loaf, ovenMouthFront) }

// Where an oven's mouth is, as seen: see cookFacedSprite.
const (
	ovenMouthFront = iota // towards the viewer, as it was drawn
	ovenMouthSide         // on the right flank, foreshortened (facing east; west is it mirrored)
	ovenMouthBack         // round the back, out of sight
)

// genOvenFacing is genOvenCanvas with its mouth where mouth says: on the
// flank the arch is narrowed as the dome turns away and the peel leans on
// the other shoulder; round the back only the soot on the crown and the
// glow over it show where it is.
func genOvenFacing(lit, loaf bool, mouth int) (*canvas, float64, float64) {
	c := hdCanvas(42, 38)
	var cx, base, rx, h = 19 * ck, 33 * ck, 15.5 * ck, 23 * ck
	// The peel, leaning on the dome's right shoulder (on the left one with
	// the mouth on the right).
	ps := 1.0
	if mouth == ovenMouthSide {
		ps = -1
	}
	ckPole(c, cx+ps*19*ck, base+1*ck, cx+ps*14.6*ck, base-19*ck, 2.2, woodRamp, 0.8, 3180)
	for y := 0; y < 20; y++ {
		f := float64(y) / (6.5 * ck) // 0 at the blade's tip
		half := 2.6 * ck * min(1, 0.55+f*1.6) * min(1, 0.7+(1-f)*1.2)
		mx := cx + ps*(13.5*ck+f*1.3*ck) // it leans with the handle
		for x := int(mx - half); x <= int(mx+half); x++ {
			u := (float64(x) + 0.5 - mx) / half
			c.put(x, int(base-25.2*ck)+y, grainHD(woodRamp, x, y, 0.84-(u+1)/2*0.38-f*0.08, true, 3181))
		}
	}
	// The mouth's middle and size, as seen.
	var mx, my, mw, mh = cx - 1.5*ck, base - 0.6*ck, 6.6 * ck, 9.6 * ck
	if mouth == ovenMouthSide {
		mx, mw = cx+rx*0.72, mw*0.5
	}
	// The dome.
	for y := int(base - h); y <= int(base); y++ {
		for x := int(cx - rx); x <= int(cx+rx); x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-base)/h
			if nx*nx+ny*ny > 1 {
				continue
			}
			l := lum(nx*0.95, ny*0.8-0.1)*0.95 + 0.04
			// Smoothed by hand: long strokes round the dome, a few cracks.
			l += (vnoise(float64(x)*0.05+ny*3, float64(y)*0.3, 3182) - 0.5) * 0.22
			l += (hashf(x, y, 3183) - 0.5) * 0.07
			if cr := vnoise(float64(x)*0.11, float64(y)*0.11, 3184); math.Abs(cr-0.5) < 0.012 && ny > -0.85 {
				l -= 0.3
			}
			if hashf(x/2, y/3, 3185) < 0.02 {
				l += 0.2 // a straw in the daub
			}
			col := ovenRamp.pick(l, x, y)
			// Soot above the mouth (over the crown, with the mouth behind).
			sx, sy, sw := mx, base-11.5*ck, 8.5*ck
			if mouth == ovenMouthBack {
				sx, sy, sw = cx-1.5*ck, base-h+3*ck, 9*ck
			} else if mouth == ovenMouthSide {
				sw = 5.5 * ck
			}
			ux, uy := (float64(x)+0.5-sx)/sw, (float64(y)+0.5-sy)/(8*ck)
			if s := 1 - (ux*ux + uy*uy*0.8); s > 0 && (uy < 0.3 || mouth == ovenMouthBack) {
				col = mix(col, ashRamp.pick(l*0.6, x, y), min(0.8, s*1.1))
			}
			c.put(x, y, col)
		}
	}
	// The smoke hole: a collar of clay with the dark in it.
	hx, hy := cx+4.5*ck, base-h+2.6*ck
	c.ellipse(hx, hy, 3.6*ck, 1.8*ck, func(x, y int, nx, ny float64) color.RGBA {
		return ovenRamp.pick(0.95-(nx+1)/2*0.4-ny*0.2, x, y)
	})
	c.ellipse(hx, hy+0.5, 2.3*ck, 1*ck, func(x, y int, nx, ny float64) color.RGBA {
		return mix(hex(0x120a08), ashRamp[1], max(0, ny)*0.6)
	})
	if lit && mouth == ovenMouthBack { // the fire's glow on the smoke over the crown
		c.ellipse(hx, hy-1, 1.6*ck, 0.6*ck, func(x, y int, nx, ny float64) color.RGBA {
			return fireRamp.pick(0.7-ny*0.2, x, y)
		})
	}
	// The stones it stands on.
	for i := 0; i < 10; i++ {
		f := float64(i)/9*2 - 1
		lift := (1 - math.Sqrt(max(0, 1-f*f))) * 2.6 * ck
		stoneHD(c, cx+f*(rx-1*ck), base+0.6*ck-lift, (2.3+hashf(i, 0, 3186)*0.5)*ck, (1.7+hashf(i, 1, 3186)*0.3)*ck, 1-float64(i)*0.035, stoneRamp, 3187+uint32(i))
	}
	if mouth != ovenMouthBack {
		// The mouth: an arch lined with a lip of paler clay.
		if mouth == ovenMouthSide {
			my -= 2.4 * ck // round the flank it stands higher on the stones
		}
		lipW := 4.0
		if mouth == ovenMouthSide {
			lipW = 2.5
		}
		arch := func(x, y int, w, hh float64) (bool, float64, float64) {
			ax, ay := (float64(x)+0.5-mx)/w, (float64(y)+0.5-my)/hh
			return ay <= 0 && ax*ax+ay*ay <= 1, ax, ay
		}
		for y := int(my - mh - 5); y <= int(my); y++ {
			for x := int(mx - mw - 5); x <= int(mx+mw+5); x++ {
				in, ax, ay := arch(x, y, mw, mh)
				if lip, lx, _ := arch(x, y, mw+lipW, mh+4); lip && !in {
					c.put(x, y, ovenRamp.pick(0.86-(lx+1)/2*0.4+ckTex(x, y, 3195, 0.1, 0.1), x, y))
					continue
				}
				if !in {
					continue
				}
				d := math.Sqrt(ax*ax + ay*ay)
				col := mix(hex(0x0e0706), ashRamp[2], max(0, d-0.55)*1.1)
				if lit {
					g := 0.95 + ay*0.55 - math.Abs(ax)*0.3 + (vnoise(float64(x)*0.2, float64(y)*0.2, 3196)-0.5)*0.4
					col = fireRamp.pick(g, x, y)
					if d > 0.82 {
						col = mix(col, fireRamp[0], 0.6)
					}
				} else if ay > -0.16 && hashf(x/2, y, 3197) < 0.35 {
					col = ashRamp.at(1 + x%2) // cold ash on the floor
				}
				c.put(x, y, col)
			}
		}
		if lit { // the glow thrown on the lip and the stones in front
			for y := int(my - mh - 5); y <= int(my+2*ck); y++ {
				for x := int(mx - mw - 7); x <= int(mx+mw+7); x++ {
					if in, _, _ := arch(x, y, mw, mh); in || !c.opaque(x, y) {
						continue
					}
					ax, ay := (float64(x)+0.5-mx)/(mw+7), (float64(y)+0.5-(my-mh*0.4))/(mh+5)
					if g := 1 - (ax*ax + ay*ay); g > 0 {
						c.put(x, y, mix(c.at(x, y), hex(0xff9a2c), g*0.5))
					}
				}
			}
		}
	}
	if loaf {
		// A flat stone by the mouth (to its left, or before the flank), and
		// the loaf on it.
		lx, ly := cx-14*ck, base+1.4*ck
		if mouth == ovenMouthSide {
			lx, ly = cx+12*ck, base+3.2*ck
		}
		stoneHD(c, lx, ly+1.5, 5*ck, 1.9*ck, 1, stoneRamp, 3198)
		loafHD(c, lx, ly-1.3*ck, 4.3*ck, 2.6*ck, 3199)
	}
	c.outline(0.75)
	return c, cx, base
}

// loafHD is a round loaf at (x, y): a golden crust slashed three times, the
// cuts pale, flour on its top.
func loafHD(c *canvas, x, y, rx, ry float64, seed uint32) {
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		l := lum(nx*0.9, ny*0.9)*0.85 + 0.14 + ckTex(px, py, seed, 0.08, 0.15)
		s := math.Mod(nx*2.6-ny*0.9+9, 1)
		if ny < 0.35 && math.Abs(nx) < 0.75 && s < 0.2 {
			return breadRamp.pick(0.95-s*1.5, px, py) // a slash, opened in the heat
		}
		if ny < -0.2 && hashf(px, py, seed+1) < 0.12 {
			return hex(0xf6ecd4) // flour
		}
		return breadRamp.pick(l*0.82, px, py)
	})
}

// genRackCanvas is a drying rack: two pairs of crossed poles, a pole laid in
// their crooks and a second lower down. look says what is on it: 0 nothing,
// 1 strips of raw meat, 2 fish hung by the tail, 3 a shelf of bowls of
// brine drying to salt, 4 dried strips.
func genRackCanvas(look int) (*canvas, float64, float64) { return genRackSide(look, false) }

// genRackSide is genRackCanvas with the low pole (or the shelf) on the
// side towards the viewer (lowFront), carried by the near legs: the rack
// turned about, facing north.
func genRackSide(look int, lowFront bool) (*canvas, float64, float64) {
	c := hdCanvas(34, 36)
	var cx, base, topY = 17 * ck, 32 * ck, 6.5 * ck
	var half = 12.5 * ck
	// Each end: two poles crossed near the top, the far one first.
	for i, s := range []float64{-1, 1} {
		ex := cx + s*half
		ckPole(c, ex+3.2*ck, base-2.5*ck, ex-2*ck, topY-2.5*ck, 3, trunkRamp, 0.66-float64(i)*0.06, 3200+uint32(i)*4)
	}
	// The low pole at the back, which the far legs carry (in front, the near).
	lowY := 17.5 * ck
	if lowFront {
		lowY += 3.2 * ck
	}
	low := func() {
		if look == 3 {
			// A shelf of split poles instead, with the bowls on it.
			for j := 0; j < 3; j++ {
				y := lowY + float64(j)*2.6 - 1
				ckPole(c, cx-half-2*ck, y+1, cx+half+2*ck, y, 2.1, woodRamp, 0.9-float64(j)*0.12, 3210+uint32(j))
			}
			for i := 0; i < 3; i++ {
				brineHD(c, cx-8*ck+float64(i)*8*ck, lowY-1.2*ck, i, 3240+uint32(i))
			}
		} else {
			ckPole(c, cx-half-0.5*ck, lowY+1, cx+half+2.5*ck, lowY, 2.2, trunkRamp, 0.72, 3214)
		}
	}
	if !lowFront {
		low()
	}
	// What hangs from the top pole.
	switch look {
	case 1, 4:
		for i := 0; i < 6; i++ {
			x := cx - half + 3.2*ck + float64(i)*3.9*ck + (hashf(i, 0, 3220)-0.5)*3
			stripHD(c, x, topY+2, (7.5+hashf(i, 1, 3220)*3.5)*ck, look == 4, 3221+uint32(i))
		}
	case 2:
		for i := 0; i < 5; i++ {
			x := cx - half + 3.6*ck + float64(i)*4.7*ck
			hungFishHD(c, x, topY+2, (8.6+hashf(i, 1, 3230)*1.6)*ck, 3231+uint32(i))
		}
	}
	// The near legs, the top pole in the crooks, and the lashings.
	for i, s := range []float64{-1, 1} {
		ex := cx + s*half
		ckPole(c, ex-3.2*ck, base, ex+2*ck, topY-3*ck, 3.3, trunkRamp, 0.92-float64(i)*0.1, 3202+uint32(i)*4)
	}
	if lowFront {
		low()
	}
	ckPole(c, cx-half-3*ck, topY+1, cx+half+3*ck, topY, 2.6, trunkRamp, 0.9, 3216)
	for _, s := range []float64{-1, 1} {
		ckLash(c, cx+s*half, topY+2, 6, 4)
	}
	if look == 1 || look == 2 || look == 4 { // the ties over the pole
		n, step, off := 6, 3.9*ck, 3.2*ck
		if look == 2 {
			n, step, off = 5, 4.7*ck, 3.6*ck
		}
		for i := 0; i < n; i++ {
			x := cx - half + off + float64(i)*step
			for dy := -3; dy <= 3; dy++ {
				if c.opaque(int(x), int(topY)+dy) {
					c.put(int(x), int(topY)+dy, ropeRamp.at(2))
					c.put(int(x)+1, int(topY)+dy, ropeRamp.at(0))
				}
			}
		}
	}
	c.outline(0.75)
	return c, cx, base - 1.2*ck
}

// genRackLong is the drying rack turned to run north–south, facing east,
// seen end on: its two pairs of crossed legs one behind the other, the top
// pole going back up the screen between their crooks, the low pole (or the
// shelf of bowls) going back on the west side, and what hangs from the top
// pole in a row going back, seen edgeways.
func genRackLong(look int) (*canvas, float64, float64) {
	c := hdCanvas(30, 52)
	var cx, base, rise = 15 * ck, 47 * ck, 25.5 * ck // rise: the top pole's height
	const depth = 15.0 * ck                           // from the front pair to the back, on the screen
	pair := func(foot float64, lit float64, seed uint32) {
		top := foot - rise
		ckPole(c, cx+5.2*ck, foot-1.2*ck, cx-2*ck, top-1.5*ck, 3, trunkRamp, lit-0.18, seed)
		ckPole(c, cx-5.2*ck, foot, cx+2*ck, top-2*ck, 3.3, trunkRamp, lit, seed+1)
		ckLash(c, cx, top+2, 6, 4)
	}
	backFoot := base - depth
	pair(backFoot, 0.8, 3250)
	// The low pole, on the west side going back; or the shelf with the bowls.
	lowRise := 13.5 * ck
	if look == 3 {
		for j := 0; j < 3; j++ {
			x := cx - 3.2*ck + float64(j)*3.1*ck
			ckPole(c, x, backFoot-lowRise, x+0.2*ck, base-lowRise, 2.1, woodRamp, 0.9-float64(j)*0.12, 3255+uint32(j))
		}
		for i := 0; i < 3; i++ {
			brineHD(c, cx, backFoot-lowRise+float64(i)*depth/3+2*ck, i, 3260+uint32(i))
		}
	} else {
		ckPole(c, cx-3.6*ck, backFoot-lowRise, cx-3.6*ck, base-lowRise, 2.2, trunkRamp, 0.72, 3256)
	}
	// The top pole, going back between the crooks.
	ckPole(c, cx, backFoot-rise-2, cx, base-rise, 2.6, trunkRamp, 0.9, 3257)
	// What hangs from it, back to front, staggered a little either side.
	switch look {
	case 1, 4:
		for i := 0; i < 5; i++ {
			y := backFoot - rise + 2 + float64(i)*depth/5 + 2*ck
			x := cx + (float64(i%2)-0.5)*1.6*ck
			stripHD(c, x, y, (7+hashf(i, 1, 3262)*3)*ck, look == 4, 3263+uint32(i))
		}
	case 2:
		for i := 0; i < 4; i++ {
			y := backFoot - rise + 2 + float64(i)*depth/4 + 2*ck
			x := cx + (float64(i%2)-0.5)*2*ck
			hungFishHD(c, x, y, (8.2+hashf(i, 1, 3270)*1.4)*ck, 3271+uint32(i))
		}
	}
	pair(base, 0.92, 3252)
	c.outline(0.75)
	return c, cx, base - depth/2
}

// stripHD is a strip of meat hung from (x, y), l pixels long: raw and red,
// or dried dark and curled.
func stripHD(c *canvas, x, y, l float64, dried bool, seed uint32) {
	rp := meatRamp
	if dried {
		rp = jerkyRamp
	}
	for i := 0; i < int(l); i++ {
		f := float64(i) / l
		w := (1.25 + 0.5*math.Sin(f*math.Pi)) * ck * (1 - f*0.25)
		cxp := x + math.Sin(f*5+float64(seed))*1.6*(0.4+f)
		for dx := -w; dx <= w; dx++ {
			u := dx / w
			lit := 0.82 - (u+1)/2*0.5 - f*0.12 + ckTex(int(cxp+dx), int(y)+i, seed, 0.12, 0.2)
			col := rp.pick(lit, int(cxp+dx), int(y)+i)
			if !dried && math.Abs(u-0.35+0.5*math.Sin(f*7)) < 0.16 {
				col = hex(0xf6dcc8) // a seam of fat
			}
			if dried && hashf(int(cxp+dx)/2, (int(y)+i)/3, seed+1) < 0.12 {
				col = jerkyRamp.at(4) // salt and smoke on the grain
			}
			c.put(int(cxp+dx), int(y)+i, col)
		}
	}
}

// hungFishHD is a fish hung by the tail from (x, y), l pixels long, head
// down, split and drying.
func hungFishHD(c *canvas, x, y, l float64, seed uint32) {
	// The tail fin, tied to the pole.
	c.polygon([]float64{x, y + 5, x - 4, y - 1, x, y + 2, x + 4, y - 1}, func(px, py int) color.RGBA {
		return finRamp.pick(0.5+ckTex(px, py, seed, 0.3, 0), px, py)
	})
	cy, ry := y+3+l*0.5, l*0.5
	c.ellipse(x, cy, 1.9*ck, ry, func(px, py int, nx, ny float64) color.RGBA {
		lit := 0.85 - (nx+1)/2*0.5 + ckTex(px, py, seed+1, 0.08, 0.14)
		col := fishRamp.pick(lit, px, py)
		if nx > 0.15 {
			col = mix(col, hex(0xf2e6c8), 0.55) // the pale belly
		}
		if int(float64(py)*0.9+float64(px)*0.5)%4 == 0 && nx < 0.1 {
			col = fishRamp.pick(lit-0.22, px, py) // scales
		}
		if ny > 0.45 && ny < 0.56 {
			col = fishRamp.at(0) // the gill
		}
		return col
	})
	c.put(int(x-1.5), int(cy+ry*0.74), hex(0x140c10))
	c.put(int(x-1.5), int(cy+ry*0.74)-1, hex(0xf4f0e0))
}

// brineHD is a shallow clay bowl at (x, y) with sea water drying in it:
// each one further along, the last a crust of salt.
func brineHD(c *canvas, x, y float64, stage int, seed uint32) {
	var rx, ry = 3.5 * ck, 1.5 * ck
	for i := 5; i >= 1; i-- {
		f := float64(i) / (1.6 * ck)
		c.ellipse(x, y+float64(i), rx*(1-f*0.35), ry*(1-f*0.2), func(px, py int, nx, ny float64) color.RGBA {
			return potRamp.pick(0.7-(nx+1)/2*0.45-f*0.2+ckTex(px, py, seed, 0.08, 0.1), px, py)
		})
	}
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		return potRamp.pick(0.95-(nx+1)/2*0.4, px, py)
	})
	c.ellipse(x, y+0.4, rx-2.2, ry-1.2, func(px, py int, nx, ny float64) color.RGBA {
		water := mix(hex(0x3f86c4), hex(0x9ee2ff), clamp01(0.4-nx*0.4-ny*0.3))
		salt := saltRamp.pick(0.6-nx*0.3-ny*0.2+ckTex(px, py, seed+1, 0.3, 0.2), px, py)
		d := nx*nx + ny*ny
		switch stage {
		case 0:
			if d > 0.75 {
				return salt
			}
			return water
		case 1:
			if d > 0.3+hashf(px/2, py, seed+2)*0.3 {
				return salt
			}
			return water
		}
		return salt
	})
	if stage == 2 { // a heap of it, glinting
		c.ellipse(x, y-1.5, rx*0.55, ry*0.9, func(px, py int, nx, ny float64) color.RGBA {
			if hashf(px, py, seed+3) < 0.1 {
				return hex(0xffffff)
			}
			return saltRamp.pick(0.75-nx*0.3-ny*0.3, px, py)
		})
	}
}

// genClayCanvas is a bed of clay at the water's edge: wet, smooth lumps the
// colour of old brick, cracked where they have dried; dug, a hollow with a
// little water in it.
func genClayCanvas(dug bool) (*canvas, float64, float64) {
	c := hdCanvas(24, 14)
	var cx, cy = 12 * ck, 8 * ck
	if dug {
		c.ellipse(cx, cy, 8.5*ck, 3.6*ck, func(x, y int, nx, ny float64) color.RGBA {
			d := nx*nx + ny*ny
			l := 0.2 + d*0.3 - ny*0.15 + ckTex(x, y, 3251, 0.08, 0.16)
			col := clayRamp.pick(l, x, y)
			if d < 0.3+ckTex(x, y, 3252, 0, 0.3) {
				col = mix(hex(0x35586a), hex(0x7fb2c4), clamp01(0.3-ny*0.6)) // seep water
			}
			return col
		})
		for i, p := range [][2]float64{{-8, 1.2}, {7.5, 1.6}, {-3, 3.2}} {
			lumpHD(c, cx+p[0]*ck, cy+p[1]*ck, 2.4*ck, 1.5*ck, 3253+uint32(i))
		}
		c.outline(0.7)
		return c, cx, cy + 1*ck
	}
	c.ellipse(cx, cy+0.6*ck, 10.5*ck, 4*ck, func(x, y int, nx, ny float64) color.RGBA { // the wet ground they lie in
		if nx*nx+ny*ny > 0.75+ckTex(x, y, 3259, 0, 0.5) {
			return color.RGBA{}
		}
		return clayRamp.pick(0.22+ckTex(x, y, 3258, 0.08, 0.2), x, y)
	})
	for i, p := range [][4]float64{{-5.5, -1.4, 4.6, 2}, {4.5, -1.6, 5, 2.1}, {-7.2, 1.4, 3.2, 1.6}, {0, 0.8, 5.6, 2.5}, {7.2, 1.7, 3, 1.5}, {-1.5, 3.2, 3, 1.3}} {
		lumpHD(c, cx+p[0]*ck, cy+p[1]*ck, p[2]*ck, p[3]*ck, 3260+uint32(i))
	}
	c.outline(0.7)
	return c, cx, cy + 1*ck
}

// lumpHD is a wet lump of clay at (x, y).
func lumpHD(c *canvas, x, y, rx, ry float64, seed uint32) {
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		l := lum(nx*0.85, ny*0.8)*0.8 + 0.12 + ckTex(px, py, seed, 0.05, 0.14)
		if cr := vnoise(float64(px)*0.16, float64(py)*0.16, seed+1); math.Abs(cr-0.5) < 0.02 {
			l -= 0.25 // a drying crack
		}
		col := clayRamp.pick(l, px, py)
		// The wet shine on its upper left.
		if g := (nx+0.45)*(nx+0.45) + (ny+0.5)*(ny+0.5)*2.2; g < 0.07 {
			col = mix(col, hex(0xf4e8dc), 0.8-g*8)
		}
		if ny > 0.55 {
			col = mix(col, clayRamp[0], 0.4)
		}
		return col
	})
}

// genPuff is a soft puff of steam or smoke, white, to be tinted: i picks
// one of three shapes.
func genPuff(i int) sprite {
	var size = 14
	c := hdCanvas(size, size)
	var m = float64(size) * ck / 2
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := (float64(x)+0.5-m)/m, (float64(y)+0.5-m)/m
			d := math.Sqrt(dx*dx+dy*dy) + (vnoise(float64(x)*0.14+float64(i)*9, float64(y)*0.14, 3270+uint32(i))-0.5)*0.55
			if d >= 1 {
				continue
			}
			a := (1 - d) * 1.7
			l := 0.86 - dy*0.14 - dx*0.06
			v := uint8(255 * clamp01(l))
			c.put(x, y, color.RGBA{v, v, v, uint8(255 * clamp01(a))})
		}
	}
	return mkUp(c, m, m)
}
