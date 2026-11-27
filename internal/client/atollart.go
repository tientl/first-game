package client

import (
	"image/color"
	"math"
	"sync"
)

// The Megalodon's atoll (sim/atoll.go), drawn at HD: the reef flat a player
// stands on, baked with the ground (reefFlat: coral rock awash, tide pools,
// weed and crusts of pink coralline); the wreck stranded across the reef,
// its deck a floor to walk on; its mast, the landmark, a gull's perch.

// The reef's colours, from COMFY52: a bleached coral rock, warm and grey,
// pools of the sea in it, weed and the pink crust of coralline.
var (
	reefFlatRock  = mkRamp(0x414453, 0x696f80, 0x867066, 0xb49d7e, 0xc4c6b8, 0xe3e6ac)
	reefPool  = mkRamp(0x1e314b, 0x2f4c6c, 0x3d80a3, 0x63c4cc, 0x9ae5d5)
	reefWeed  = mkRamp(0x3f4328, 0x5f7132, 0x739a70, 0x94ad39)
	reefPink  = mkRamp(0x7b234c, 0xb23657, 0xd16974, 0xedaaa3)
	wreckDeck  = mkRamp(0x2b222a, 0x3c252b, 0x533b41, 0x684039, 0x825646, 0xb77862)
	deckGrey  = mkRamp(0x343f41, 0x414453, 0x696f80, 0x8d99a4, 0xbac9cd)
	hullSlime = mkRamp(0x1e314b, 0x343f41, 0x4d6659, 0x739a70)
)

// reefFlat colours a pixel of the atoll's reef at (u, v), s tiles in from
// the water: rock in rounded heads and the cracks between them, lit from
// the top left, tide pools, weed low down, crusts of coralline, darker and
// glossy where the sea washes over it.
func reefFlat(hx, hy int, u, v, s float64) color.RGBA {
	// The heads of old coral: a field of bumps.
	h := func(x, y float64) float64 {
		return fbm2(x*1.6, y*2.0, 1501, 3)*0.7 + vnoise(x*4.5, y*5.5, 1502)*0.3
	}
	const e = 0.04
	hc := h(u, v)
	gx, gy := (h(u+e, v)-hc)/e, (h(u, v+e)-hc)/e
	l := 0.55 - 0.18*gx - 0.22*gy + (hc-0.5)*0.5
	// Tide pools in the hollows, away from the open water's edge.
	pool := fbm2(u*0.7+3.1, v*0.9, 1503, 2)
	if pool < 0.34 && s > 0.35 {
		d := (0.34 - pool) / 0.34
		if d < 0.12 { // the pool's rim: wet, dark
			return rq(reefFlatRock, 0.15+l*0.3, hx, hy)
		}
		pl := 0.35 + 0.4*d + (vnoise(u*6, v*8, 1504)-0.5)*0.2
		if hashf(hx/3, hy/2, 1505) < 0.02 {
			return reefPool.at(4) // a glint of the sky
		}
		return rq(reefPool, pl, hx, hy)
	}
	switch {
	case hc < 0.32: // a crack between heads
		l = 0.08 + hc*0.4
	case hc > 0.72:
		l += 0.12
	}
	c := rq(reefFlatRock, l, hx, hy)
	if w := vnoise(u*1.1, v*1.3, 1506); w > 0.62 && hc < 0.55 { // weed in the low places
		c = rq(reefWeed, l*0.9+(w-0.62), hx, hy)
	} else if p := vnoise(u*2.3+7, v*2.9, 1507); p > 0.7 && hc > 0.45 { // pink coralline on the heads
		c = rq(reefPink, l*0.8+0.15, hx, hy)
	}
	if hashf(hx, hy, 1508) < 0.008 { // a shell, a speck of white
		c = hex(0xe5efef)
	}
	// Awash at the edge: darker, glossy, a film of water and foam.
	if s < 0.45 {
		w := 1 - s/0.45
		c = mix(c, hex(0x1e314b), 0.35*w)
		if s < 0.1 && hashf(hx/2, hy, 1509) < 0.5 {
			c = mix(c, colFoam, 0.5*(1-s/0.1))
		}
	}
	return c
}

// atollSprites is the atoll's art, made the first time it is wanted.
type atollSprites struct {
	deck sprite // the wreck's hull, flat on the reef: walked over
	mast sprite // its mast, standing on the deck
	palm sprite
}

var (
	atollOnce sync.Once
	atollArt  *atollSprites
)

func atollSpritesOf() *atollSprites {
	atollOnce.Do(func() {
		a := &atollSprites{}
		inParallel(2, func(i int) {
			if i == 0 {
				a.deck = genWreckDeck()
			} else {
				a.mast = genTallMast()
			}
		})
		atollArt = a
	})
	return atollArt
}

// genWreckDeck is the wreck's hull lying on the reef, listing to the
// south: eight tiles of deck, planks run its length with gaps into the
// dark hold, the rail along its far side, its near side seen down to the
// waterline, slimed and crusted; its bow pointed to the west, its stern
// broken off, ribs standing out of the water there. Its ground point is the
// middle of the deck, on the middle tile.
func genWreckDeck() sprite { return mkUp(wreckDeckCanvas()) }

// wreckDeckCanvas draws genWreckDeck.
func wreckDeckCanvas() (*canvas, float64, float64) {
	const L, B = 8 * TileW, 3 * TileH // art pixels of deck
	k := hdK3
	side := 13.0 // art pixels of hull seen below the deck
	c := hdCanvas(L+16, B+int(side)+18)
	ox, oy := 8.0, 6.0 // the deck's top left, art pixels
	// The deck's outline: a pointed bow (left), a full beam, the broken
	// stern (right), ragged.
	inDeck := func(x, y float64) (bool, float64) { // y across the beam, 0 to 1
		f := x / L
		half := 0.5
		switch {
		case f < 0.22:
			half *= math.Sqrt(clamp01(f / 0.22))
		case f > 0.86:
			if r := vnoise(y*6, 3, 1601); f > 0.86+0.12*r {
				return false, 0
			}
		}
		return math.Abs(y-0.5) <= half, (y - 0.5) / max(half, 1e-6)
	}
	for py := 0; py < c.h; py++ {
		for px := 0; px < c.w; px++ {
			x, y := float64(px)/k-ox, float64(py)/k-oy
			// The near side of the hull: under the deck's south edge.
			if ok, _ := inDeck(x, 1); ok || x < L*0.3 {
				f := x / L
				half := 0.5
				if f < 0.22 {
					half *= math.Sqrt(clamp01(f / 0.22))
				}
				edge := B * (0.5 + half)
				if y > edge && y < edge+side*(0.4+0.6*half*2) && x > 1 && x < L*0.95 {
					d := (y - edge) / side
					l := 0.55 - d*0.6
					if math.Mod(y-edge+(x*0.03), 3.2) < 0.45 {
						l -= 0.2 // the seams of the planks
					}
					col := rq(wreckDeck, l+(vnoise(x*0.2, y*0.6, 1602)-0.5)*0.2, px, py)
					if d > 0.45 { // slime and crust towards the water
						col = rq(hullSlime, 0.6-d*0.4+(hashf(px/2, py/2, 1603)-0.5)*0.3, px, py)
						if hashf(px, py, 1604) < 0.05 {
							col = reefFlatRock.at(4) // barnacles
						}
					}
					c.px[py*c.w+px] = col
					continue
				}
			}
			ok, across := inDeck(x, y/B)
			if !ok || y < 0 {
				continue
			}
			// Planks along its length, butted end to end.
			row := math.Floor(y / 2.3)
			fy := y/2.3 - row
			off := hashf(int(row), 0, 1605) * 30
			seg := math.Floor((x + off) / 26)
			fx := (x+off)/26 - seg
			id := int(row)*131 + int(seg)
			l := 0.52 + (hashf(id, 1, 1605)-0.5)*0.22
			switch {
			case fy < 0.16:
				l -= 0.3 // the seam
			case fy < 0.3:
				l += 0.12
			}
			if fx < 0.02 {
				l -= 0.25
			}
			l += (vnoise(x*0.6, y*2.4, 1606) - 0.5) * 0.25 // grain
			rp := wreckDeck
			if hashf(id, 2, 1605) < 0.45 {
				rp = deckGrey // bleached by the sun
			}
			col := rq(rp, l, px, py)
			// Planks gone: the dark of the hold, ribs across it.
			if h := vnoise(x*0.07, y*0.18, 1607); h > 0.72 && math.Abs(across) < 0.8 {
				col = hex(0x1a1f2e)
				if math.Mod(x, 9) < 1.6 {
					col = rq(wreckDeck, 0.3, px, py)
				}
			}
			// The rail along the far side.
			if across < -0.86 {
				col = rq(wreckDeck, 0.75-math.Abs(across+0.93)*3, px, py)
			}
			// Weed and gull droppings.
			if hashf(px/3, py/3, 1608) < 0.03 {
				col = mix(col, hex(0xe5efef), 0.6)
			}
			c.px[py*c.w+px] = col
		}
	}
	// Ribs standing out of the water past the broken stern.
	for i := range 4 {
		x := (ox + L*0.9 + float64(i)*4.5) * k
		y0, y1 := (oy+B*0.1)*k, (oy+B+side*0.6)*k
		drawLimbHD(c, limb{x, y1, x + 2*k, (y0 + y1) / 2, x - 1*k, y0 + float64(i)*2*k, 1.4 * k, 0.8 * k}, wreckRamp, 0.7, 1609+uint32(i))
	}
	c.outline(0.7)
	return c, (ox + L/2) * k, (oy + B/2 + TileH/2) * k
}

// genTallMast is the wreck's mast: leaning, five times a man's height, two
// yards askew, the rags of its sails, the rigging slack and a crow's nest;
// the landmark seen from far over the sea.
func genTallMast() sprite { return mkUp(tallMastCanvas()) }

// tallMastCanvas draws genTallMast.
func tallMastCanvas() (*canvas, float64, float64) {
	c := hdCanvas(80, 170)
	k := hdK3
	fx, fy := 40*k, 166*k
	tx, ty := 52*k, 8*k
	mast := limb{fx, fy, 44 * k, 90 * k, tx, ty, 2.6 * k, 1.2 * k}
	at := func(t float64) (float64, float64) { return mast.at(t) }
	sail := func(pts []float64, seed uint32) {
		c.polygon(pts, func(px, py int) color.RGBA {
			if vnoise(float64(px)*0.04, float64(py)*0.05, seed) < 0.32 {
				return color.RGBA{} // torn through
			}
			l := 0.35 + 0.5*vnoise(float64(px)*0.02, float64(py)*0.1, seed+1)
			if math.Mod(float64(px)*0.07+float64(py)*0.01, 2) < 0.15 {
				l -= 0.15 // the seams of its cloths
			}
			return ragRamp.pick(l, px, py)
		})
	}
	y1x, y1y := at(0.42)
	y2x, y2y := at(0.72)
	sail([]float64{y1x - 26*k, y1y + 2*k, y1x + 24*k, y1y - 4*k, y1x + 20*k, y1y + 30*k, y1x + 6*k, y1y + 22*k, y1x - 14*k, y1y + 36*k}, 1700)
	sail([]float64{y2x - 18*k, y2y + 1*k, y2x + 18*k, y2y - 3*k, y2x + 12*k, y2y + 20*k, y2x - 10*k, y2y + 16*k}, 1702)
	drawLimbHD(c, mast, wreckRamp, 0.9, 1703)
	drawLimbHD(c, limb{y1x - 28*k, y1y + 4*k, y1x, y1y - 1*k, y1x + 27*k, y1y - 5*k, 1.4 * k, 1.1 * k}, wreckRamp, 0.85, 1704)
	drawLimbHD(c, limb{y2x - 20*k, y2y + 2*k, y2x, y2y - 1*k, y2x + 19*k, y2y - 4*k, 1.1 * k, 0.9 * k}, wreckRamp, 0.85, 1705)
	nx, ny := at(0.86)
	c.ellipse(nx, ny, 6*k, 2.6*k, func(px, py int, ex, ey float64) color.RGBA { // the crow's nest
		return wreckRamp.pick(0.3+0.6*lum(ex, ey), px, py)
	})
	c.rect(int(nx-6*k), int(ny), int(12*k), int(3*k), func(px, py int) color.RGBA {
		return wreckRamp.pick(0.35+0.1*float64(px%int(2*k)), px, py)
	})
	for i, r := range [][4]float64{{-27, 4, -30, 150}, {27, -5, 30, 156}, {-19, 2, -6, 150}, {19, -4, 14, 154}} { // rigging, slack
		x0, y0 := y1x+r[0]*k, y1y+r[1]*k
		if i >= 2 {
			x0, y0 = y2x+r[0]*k, y2y+r[1]*k
		}
		x1, y1 := fx+r[2]*k, r[3]*k
		n := 60
		for j := range n {
			t := float64(j) / float64(n)
			x, y := lerpf(x0, x1, t), lerpf(y0, y1, t)+math.Sin(t*math.Pi)*8*k
			if (j+i)%7 != 0 {
				c.dot(x, y, 0.4*k, ropeRamp.at(0))
			}
		}
	}
	for i := range 3 { // a pennant of rag at the top
		c.line(tx, ty+float64(i)*k, tx+14*k, ty+3*k+float64(i)*1.4*k, func(px, py int, t float64) color.RGBA { return ragRamp.at(2) })
	}
	c.outline(0.8)
	return c, fx, fy
}
