package client

import (
	"image/color"
	"math"
)

// The hearths under the fires, at HD (the flames are fireart.go's): a
// campfire is a ring of field stones round a shallow bed of ash and char,
// with four logs laid crossed in it, burnt black where they meet; the
// village bonfire is the same, bigger, its logs heaped. The coals that glow
// and the flames are drawn over them while it burns, so the same hearth
// stands cold when it is out.

// hearthStoneRamp is a field stone's: warm greys, darker than the quarry's.
var hearthStoneRamp = mkRamp(0x24201f, 0x3a3432, 0x524a46, 0x6c635d, 0x887d75, 0xa3978c, 0xbcb0a3)

// hearthStones sets n stones in a ring rx by ry round (cx, cy), image
// pixels: those of the back half or the front half. Each is its own size
// and shape, sooted along the edge that faces the fire.
func hearthStones(c *canvas, cx, cy, rx, ry float64, n int, r float64, front bool, seed uint32) {
	type st struct{ x, y, rx, ry, a float64 }
	var sts []st
	for i := 0; i < n; i++ {
		a := 2*math.Pi*(float64(i)+0.3*(hashf(i, 0, seed)-0.5))/float64(n) + 0.15
		if (math.Sin(a) >= 0) != front {
			continue
		}
		v := 0.78 + hashf(i, 1, seed)*0.42
		sts = append(sts, st{cx + math.Cos(a)*rx, cy + math.Sin(a)*ry, r * v * (1 + 0.25*hashf(i, 2, seed)), r * 0.72 * v, a})
	}
	// Far ones first, so the nearer overlap them.
	for i := 1; i < len(sts); i++ {
		for j := i; j > 0 && sts[j].y < sts[j-1].y; j-- {
			sts[j], sts[j-1] = sts[j-1], sts[j]
		}
	}
	for i, s := range sts {
		in := [2]float64{-math.Cos(s.a), -math.Sin(s.a)} // towards the fire
		sd := seed + uint32(i)*31
		c.ellipse(s.x, s.y, s.rx, s.ry, func(px, py int, nx, ny float64) color.RGBA {
			// A few flat faces, lit from the upper left; a darker foot.
			fx, fy := math.Round(nx*1.5)/1.5, math.Round(ny*1.5)/1.5
			l := lum(0.5*nx+0.5*fx, 0.5*ny+0.5*fy)*0.95 + (vnoise(float64(px)*0.35, float64(py)*0.35, sd)-0.5)*0.18
			if ny > 0.5 {
				l -= 0.18 * (ny - 0.5) * 2
			}
			if hashf(px/2, py/2, sd+1) < 0.035 {
				l -= 0.2 // a pit
			}
			col := hearthStoneRamp.pick(l, px, py)
			// Soot on the side towards the fire.
			if f := nx*in[0] + ny*in[1]*1.4; f > 0.25 {
				col = mix(col, hex(0x1c1715), clamp01((f-0.25)*1.1))
			}
			return col
		})
	}
}

// hearthBed is the shallow pit of ash and char inside the stones, rx by ry
// round (cx, cy), image pixels: grey ash at its rim, black char and dead
// coals at its heart.
func hearthBed(c *canvas, cx, cy, rx, ry float64, seed uint32) {
	c.ellipse(cx, cy, rx, ry, func(x, y int, nx, ny float64) color.RGBA {
		r := math.Hypot(nx, ny)
		n := vnoise(float64(x)*0.22, float64(y)*0.3, seed)
		col := mix(hex(0x6b625c), hex(0x3d3633), clamp01(1.2-r+(n-0.5)*0.6)) // ash, darker inwards
		if r < 0.75 {
			col = mix(col, hex(0x1d1715), clamp01((0.75-r)*2.4+(n-0.5)*0.5)) // char
		}
		if hashf(x/2, y/2, seed+1) < 0.07 {
			col = mix(col, hex(0x8e857c), 0.6) // a fleck of white ash
		}
		if r < 0.6 && hashf(x/3, y/3, seed+2) < 0.1 {
			col = hex(0x2c1612) // a dead coal
		}
		return col
	})
}

// hearthLog lays a log from (x0, y0) to (x1, y1), image pixels, w half
// thick: bark lit from above along its length, burnt to black char with a
// grain of cracks towards (cx, cy) where the fire is, its cut end at
// (x0, y0) pale with rings.
func hearthLog(c *canvas, x0, y0, x1, y1, w, cx, cy float64, seed uint32) {
	dx, dy := x1-x0, y1-y0
	L := math.Hypot(dx, dy)
	ux, uy := dx/L, dy/L
	nxv, nyv := -uy, ux
	if nyv > 0 {
		nxv, nyv = -nxv, -nyv // the normal that points up the screen
	}
	reach := math.Hypot(x0-cx, y0-cy)
	bx0, bx1 := int(math.Min(x0, x1)-w-1), int(math.Max(x0, x1)+w+1)
	by0, by1 := int(math.Min(y0, y1)-w-1), int(math.Max(y0, y1)+w+1)
	for y := by0; y <= by1; y++ {
		for x := bx0; x <= bx1; x++ {
			px, py := float64(x)+0.5-x0, float64(y)+0.5-y0
			t := px*ux + py*uy
			if t < 0 || t > L {
				continue
			}
			s := (px*nxv + py*nyv) / w // 1 at its upper edge, -1 at its lower
			if s*s > 1 {
				continue
			}
			// The bark: lit along the top, furrows running its length.
			f := vnoise(s*2.2+float64(seed), t*0.09, seed)
			l := 0.42 + 0.42*s
			if f < 0.38 {
				l -= 0.22
			} else if f > 0.7 {
				l += 0.08
			}
			col := trunkRamp.pick(l+(hashf(x, y, seed+3)-0.5)*0.08, x, y)
			// Burnt towards the fire.
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			if burn := clamp01((reach*0.78 - d) / (reach * 0.35)); burn > 0 {
				char := mix(hex(0x2a211e), hex(0x15100f), 0.5-0.5*s)
				if g := vnoise(t*0.45, s*3, seed+9); g > 0.62 {
					char = mix(char, hex(0x4a3c36), 0.6) // the grey sheen on char
				}
				if hashf(int(t/2.5), int((s+1)*2), seed+7) < 0.3 {
					char = hex(0x0e0a0a) // a crack across the grain
				}
				col = mix(col, char, smooth(burn))
			}
			c.put(x, y, col)
		}
	}
	// The cut end: pale wood with its rings, darker at the bark.
	c.ellipse(x0, y0, w*0.62, w*0.95, func(x, y int, nx, ny float64) color.RGBA {
		r := math.Hypot(nx, ny)
		col := woodRamp.pick(0.85-r*0.55, x, y)
		if r > 0.82 {
			col = trunkRamp.at(1)
		} else if math.Mod(r*4.5, 1) < 0.22 {
			col = mix(col, woodRamp.at(0), 0.6) // a ring
		}
		return col
	})
}

// The campfire's logs, art pixels from its middle: from the cut end to the
// burnt end. The far two lie under the flames; the near two over the front
// stones, so the flames rise from among them.
var (
	campLogsFar  = [][4]float64{{-8.8, -2.4, 1.4, 0.5}, {8.6, -2.6, -1.3, 0.5}}
	campLogsNear = [][4]float64{{-9.6, 3.4, -2.6, 1.0}, {9.4, 3.2, 2.4, 0.9}}
)

func genCampfireHD() (back, front sprite) {
	const k = HDDensity
	const cx, cy = 15.0 * k, 10.0 * k
	b, f := hdCanvas(30, 18), hdCanvas(30, 18)
	hearthStones(b, cx, cy, 10.6*k, 4.8*k, 13, 2.3*k, false, 200)
	hearthBed(b, cx, cy+0.3*k, 8.4*k, 3.7*k, 201)
	for i, l := range campLogsFar {
		hearthLog(b, cx+l[0]*k, cy+l[1]*k, cx+l[2]*k, cy+l[3]*k, 1.2*k, cx, cy, 7+uint32(i))
	}
	hearthStones(f, cx, cy, 10.6*k, 4.8*k, 13, 2.3*k, true, 200)
	for i, l := range campLogsNear {
		hearthLog(f, cx+l[0]*k, cy+l[1]*k, cx+l[2]*k, cy+l[3]*k, 1.3*k, cx, cy, 17+uint32(i))
	}
	b.outline(0.7)
	f.outline(0.7)
	return mkUp(b, cx, cy), mkUp(f, cx, cy)
}

// genBonfireHD is the village's bonfire: its back (the far stones, the bed,
// the heap of logs under the flames) and its front (the near stones and the
// near logs, over the flames' root).
func genBonfireHD() (back, front sprite) {
	const k = HDDensity
	const cx, cy = 20.0 * k, 14.0 * k
	b, f := hdCanvas(40, 24), hdCanvas(40, 24)
	hearthStones(b, cx, cy, 15.4*k, 6.6*k, 16, 2.8*k, false, 210)
	hearthBed(b, cx, cy+0.4*k, 12.6*k, 5.2*k, 211)
	for i, l := range [][4]float64{{-12.5, -3.4, 2, 0.6}, {12.4, -3.6, -2, 0.5}, {-4, -5.6, 0.5, 0}, {4.6, -5.4, -0.6, 0}} {
		hearthLog(b, cx+l[0]*k, cy+l[1]*k, cx+l[2]*k, cy+l[3]*k, 1.55*k, cx, cy, 11+uint32(i))
	}
	hearthStones(f, cx, cy, 15.4*k, 6.6*k, 16, 2.8*k, true, 210)
	for i, l := range [][4]float64{{-13.8, 4.4, -3.6, 1.2}, {13.6, 4.6, 3.4, 1.2}, {1.5, 7.2, 0.8, 2.6}} {
		hearthLog(f, cx+l[0]*k, cy+l[1]*k, cx+l[2]*k, cy+l[3]*k, 1.6*k, cx, cy, 21+uint32(i))
	}
	b.outline(0.7)
	f.outline(0.7)
	return mkUp(b, cx, cy), mkUp(f, cx, cy)
}
