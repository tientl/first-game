package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Ores, bars, the furnace and the tools of each tier.

var (
	ironRamp  = mkRamp(0x2b3036, 0x495159, 0x6c7680, 0x97a1ab, 0xc8d0d7)
	brickRamp = mkRamp(0x3a2f2c, 0x584640, 0x76605a, 0x947c72, 0xb39c90)

	// The stone-age tiers that took the metals' places: dark glossy flint,
	// green jade, banded orange agate and black glassy obsidian.
	flintRamp    = mkRamp(0x221d1b, 0x3b3330, 0x5a4e47, 0x7f7166, 0xaa9b8c)
	jadeRamp     = mkRamp(0x10392a, 0x1b5a42, 0x2a8260, 0x49ab84, 0x8fd6b2)
	agateRamp    = mkRamp(0x561c16, 0x8a3222, 0xbb5733, 0xdf8a56, 0xf3c39a)
	obsidianRamp = mkRamp(0x08070c, 0x17121f, 0x2c2338, 0x4e4266, 0x9686bc)

	// tierRamps colour the heads of each tier's tools, indexed by sim.Tier.
	tierRamps = [sim.NumTiers]ramp{stoneRamp, flintRamp, jadeRamp, agateRamp, obsidianRamp}
)

// oreLook is how one ore shows in its rock: coloured flecks, or crystals.
type oreLook struct {
	fleck   ramp
	crystal bool
}

// oreLooks is indexed by a vein's tier less one: flint nodules, jade,
// agate, and obsidian glass.
var oreLooks = [sim.NumTiers - 1]oreLook{{flintRamp, false}, {jadeRamp, false}, {agateRamp, false}, {obsidianRamp, true}}

// interior reports whether (x, y) is opaque with opaque neighbours all
// round: on the face of a shape, not its outline.
func interior(c *canvas, x, y int) bool {
	return c.opaque(x, y) && c.opaque(x-1, y) && c.opaque(x+1, y) && c.opaque(x, y-1) && c.opaque(x, y+1) &&
		c.at(x, y-1) != colInk
}

// flecks scatters n small clusters of rp over the face of what c holds, lit
// from the top-left.
func flecks(c *canvas, rng *rand.Rand, n int, rp ramp) {
	for placed, tries := 0, 0; placed < n && tries < 400; tries++ {
		x, y := rng.IntN(c.w), rng.IntN(c.h)
		if !interior(c, x, y) {
			continue
		}
		placed++
		for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {-1, 0}}[:2+rng.IntN(3)] {
			px, py := x+d[0], y+d[1]
			if interior(c, px, py) {
				l := 0.9 - float64(py)/float64(c.h)*0.5 - float64(d[0]+d[1])*0.15
				c.set(px, py, rp.pick(l, px, py))
			}
		}
	}
}

// crystals stands n gem crystals of rp on the face of what c holds, each
// with its own outline.
func crystals(c *canvas, rng *rand.Rand, n int, rp ramp, size float64) {
	g := newCanvas(c.w, c.h)
	for placed, tries := 0, 0; placed < n && tries < 400; tries++ {
		x, y := rng.IntN(c.w), rng.IntN(c.h)
		if !interior(c, x, y) || y < c.h/3 {
			continue
		}
		placed++
		h, w := size*(0.8+rng.Float64()*0.5), size*0.45
		lean := (rng.Float64() - 0.5) * size * 0.6
		bx, by := float64(x)+0.5, float64(y)+0.5
		tx, ty := bx+lean, by-h
		g.polygon([]float64{bx - w, by, tx, ty, bx + w, by, bx, by + w*0.6}, func(px, py int) color.RGBA {
			side := float64(px) + 0.5 - (bx + (tx-bx)*(by-float64(py))/h) // left of the ridge is lit
			l := 0.55 - side*0.25 + (by-float64(py))/h*0.35
			return rp.pick(l, px, py)
		})
		g.set(int(tx), int(ty)+1, rp[len(rp)-1])
	}
	g.outline(0.8)
	c.over(g, 0, 0)
}

// The ore deposits stand on a canvas of oreW by oreH with their foot on row
// oreGround: a craggy host rock of angular blocks, unlike a boulder's round
// back, with the ore itself standing proud of it.
const (
	oreW, oreH = 30, 30
	oreGround  = 27
)

var (
	chalkRamp   = mkRamp(0x5e5648, 0x8a806c, 0xb3a98f, 0xd3cab0, 0xe9e2cc, 0xf8f4e6)
	flintOre    = mkRamp(0x121013, 0x221e20, 0x38322f, 0x574e47, 0x857a6c)
	jadeOre     = mkRamp(0x0c3329, 0x16604a, 0x27966f, 0x55c795, 0xa9f0c6)
	agateOre    = mkRamp(0x4a1610, 0x8f2f19, 0xd25f2a, 0xf29a50, 0xfbd39c, 0xfff3dc)
	obsidianOre = mkRamp(0x0b0812, 0x1a1429, 0x33284f, 0x5d4c8c, 0x9d8dd2, 0xe2dcff)
	coalOre     = mkRamp(0x09080c, 0x131118, 0x201d27, 0x35313f, 0x57536a)
	ashRock     = mkRamp(0x1c1a22, 0x2b2932, 0x3d3b46, 0x53515c, 0x6d6b75)
	jadeSheen   = hex(0xd6fbe9)
	colGloss    = hex(0x8a8496)
	colGlint    = hex(0xfdfcff)
)

// oreBlock is one angular block of a host rock: its outline, and the ridge
// from (rx, ry) down to bx on the ground that parts its lit face from its
// shaded one.
type oreBlock struct {
	pts        []float64
	rx, ry, bx float64
}

// oreBlocks is the host rock from the back to the front, as variant 0 has
// it; variant 1 is its mirror image, lit from the same side.
var oreBlocks = [...]oreBlock{
	{[]float64{7, 28, 7, 17, 9, 13, 13, 11, 19, 12, 23, 16, 24, 28}, 13, 11, 15},
	{[]float64{2, 27, 2, 22, 4, 19, 7, 17, 11, 19, 13, 28, 3, 28}, 7, 17, 8},
	{[]float64{17, 28, 18, 19, 23, 16, 27, 20, 28, 26, 27, 28}, 22, 16, 22},
	{[]float64{9, 28, 10, 23, 15, 21, 19, 23, 20, 28}, 14, 21, 15},
}

// oreX is x on variant v's canvas.
func oreX(v int, x float64) float64 {
	if v == 1 {
		return oreW - x
	}
	return x
}

// overInked draws g onto c, inking a rim round it wherever it lies on what
// c already holds, so that each piece stands apart from the one behind.
func overInked(c, g *canvas, strength float64) {
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			if g.opaque(x, y) || !c.opaque(x, y) {
				continue
			}
			if g.opaque(x-1, y) || g.opaque(x+1, y) || g.opaque(x, y-1) || g.opaque(x, y+1) {
				c.put(x, y, mix(c.at(x, y), colInk, strength))
			}
		}
	}
	c.over(g, 0, 0)
}

// oreBlockOn draws block i of variant v's host rock onto c in rp, its
// height scaled by hs and its light by dim.
func oreBlockOn(c *canvas, v, i int, rp ramp, hs, dim float64) {
	b := oreBlocks[i]
	const foot = oreGround + 1
	lift := math.Round(hashf(i, v, 77)*3) - 1
	pts := make([]float64, len(b.pts))
	for j := 0; j < len(pts); j += 2 {
		x, y := b.pts[j], b.pts[j+1]
		if y < oreGround-1 {
			y = foot - (foot-y+lift)*hs
		}
		pts[j], pts[j+1] = oreX(v, x), y
	}
	rx, bx := oreX(v, b.rx), oreX(v, b.bx)
	ry := foot - (foot-b.ry+lift)*hs
	ridge := func(py float64) float64 { return rx + (bx-rx)*clamp01((py-ry)/(foot-ry)) }
	g := newCanvas(c.w, c.h)
	g.polygon(pts, func(x, y int) color.RGBA {
		px, py := float64(x)+0.5, float64(y)+0.5
		l := 0.34
		if px < ridge(py) {
			l = 0.7
		}
		l += (vnoise(px*0.55, py*0.55, 150+uint32(i))-0.5)*0.16 - clamp01((py-oreGround+3)/4)*0.16
		return rp.pick(l*dim, x, y)
	})
	for y := 0; y < g.h; y++ { // the top edges catch the light
		for x := 0; x < g.w; x++ {
			if g.opaque(x, y) && !g.opaque(x, y-1) {
				l := 0.62
				if float64(x)+0.5 < ridge(float64(y)+0.5) {
					l = 1
				}
				g.put(x, y, rp.pick(l*dim, x, y))
			}
		}
	}
	overInked(c, g, 0.55)
}

// oreHost is variant v's host rock in rp.
func oreHost(v int, rp ramp, hs, dim float64) *canvas {
	c := newCanvas(oreW, oreH)
	for i := range oreBlocks {
		oreBlockOn(c, v, i, rp, hs, dim)
	}
	return c
}

// oreLump is a rounded lump of rp at (x, y) with a wet gloss on its
// shoulder: a nodule or a polished cobble.
func oreLump(c *canvas, x, y, rx, ry float64, rp ramp, gloss color.RGBA) {
	g := newCanvas(c.w, c.h)
	g.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		return rp.pick(lum(nx, ny)*0.95+0.05, px, py)
	})
	gx, gy := int(math.Floor(x-rx*0.4)), int(math.Floor(y-ry*0.45))
	g.put(gx, gy, gloss)
	if rx >= 2.5 {
		g.put(gx+1, gy, mix(gloss, rp[len(rp)-1], 0.4))
		g.put(gx, gy+1, rp[len(rp)-1])
	}
	overInked(c, g, 0.8)
}

// genFlintOre is a bank of pale chalk with dark, glossy flint nodules
// weathering out of it.
func genFlintOre(v int) *canvas {
	c := oreHost(v, chalkRamp, 1, 1)
	for _, n := range [][4]float64{{13, 11.5, 3.3, 2.7}, {5.5, 20.5, 2.7, 2.2}, {22.5, 17, 3, 2.5}, {9.5, 15.5, 1.9, 1.6}, {25.5, 23.5, 1.9, 1.6}, {7.5, 24.5, 1.6, 1.3}, {17, 22.5, 3.3, 2.8}} {
		oreLump(c, oreX(v, n[0]), n[1], n[2], n[3], flintOre, hex(0xf2ecdc))
	}
	c.outline(0.8)
	return c
}

// gemCrystals is how a cluster of gem crystals stands on variant 0's canvas,
// from the back to the front: each from its foot (bx, by) up to its point
// (tx, ty), w wide.
var gemCrystals = [...][5]float64{
	{14, 25, 13.5, 7.5, 4.3}, {7.5, 25, 5.5, 12.5, 3.3}, {21.5, 25, 23.5, 11, 3.5},
	{3.8, 26.5, 2.5, 19, 2}, {26, 26.5, 27.5, 18.5, 2}, {17, 27.5, 17.5, 17.5, 2.9}, {10.5, 27.5, 9.5, 20, 2.3},
}

// gemCrystal stands crystal i of a cluster on c: a stout prism that comes
// to a point, its lit and shaded faces coloured by face (up runs from 0 at
// the foot to 1 at the point), with a bright edge between them in edge and
// a glint at the point.
func gemCrystal(c *canvas, v, i int, edge color.RGBA, face func(x, y int, lit bool, up float64) color.RGBA) {
	s := gemCrystals[i]
	bx, by, tx, ty, w := oreX(v, s[0]), s[1], oreX(v, s[2]), s[3], s[4]
	h := by - ty
	lean := (tx - bx) * 0.62
	ridge := func(py float64) float64 { return bx + (tx-bx)*(by-py)/h + w*0.12 }
	g := newCanvas(c.w, c.h)
	g.polygon([]float64{bx - w, by, bx - w + lean, by - h*0.62, tx, ty, bx + w + lean, by - h*0.55, bx + w, by}, func(x, y int) color.RGBA {
		px, py := float64(x)+0.5, float64(y)+0.5
		return face(x, y, px < ridge(py), (by-py)/h)
	})
	for y := int(ty) + 1; float64(y) < by-h*0.2; y++ {
		x := int(math.Floor(ridge(float64(y) + 0.5)))
		if !g.opaque(x, y) {
			continue
		}
		col := edge
		if float64(y) < ty+h*0.35 {
			col = colGlint
		}
		g.put(x, y, col)
	}
	g.put(int(math.Floor(tx)), int(ty)+1, colGlint)
	if w > 2.5 { // a sparkle on the lit face
		sx, sy := int(math.Floor(bx-w*0.45+lean*0.5)), int(by-h*0.42)
		g.put(sx, sy, colGlint)
		g.put(sx, sy+1, edge)
	}
	overInked(c, g, 0.8)
}

// genJadeOre is a cluster of stout jade crystals, green through and
// through, with a milky sheen sliding down their lit faces.
func genJadeOre(v int) *canvas {
	c := oreHost(v, jadeOre, 0.5, 0.62)
	for i := range gemCrystals {
		gemCrystal(c, v, i, jadeOre[4], func(x, y int, lit bool, up float64) color.RGBA {
			if !lit {
				return jadeOre.pick(0.12+up*0.2, x, y)
			}
			if (x+y+i)%6 == 0 {
				return jadeSheen // the sheen
			}
			return jadeOre.pick(0.52+up*0.3, x, y)
		})
	}
	c.outline(0.8)
	return c
}

// genAgateOre is a cluster of pointed agate crystals, banded across from
// dark red through orange to cream.
func genAgateOre(v int) *canvas {
	c := oreHost(v, agateOre, 0.5, 0.42)
	bands := [...]int{1, 2, 3, 4, 2, 3, 5, 3, 2, 4}
	for i := range gemCrystals {
		gemCrystal(c, v, i, agateOre[5], func(x, y int, lit bool, up float64) color.RGBA {
			sag := float64(x) * 0.35
			if v == 1 {
				sag = -sag
			}
			b := bands[int(float64(y)+sag+float64(i*3)+40)/2%len(bands)]
			if !lit {
				return agateOre.at(b - 2)
			}
			return agateOre.at(b)
		})
	}
	c.outline(0.8)
	return c
}

// obsidianShard stands a blade of volcanic glass on c from (bx, by) up to
// its tip (tx, ty), w wide at the foot.
func obsidianShard(c *canvas, bx, by, tx, ty, w float64) {
	g := newCanvas(c.w, c.h)
	h := by - ty
	ridge := func(py float64) float64 { return bx + (tx-bx)*(by-py)/h }
	g.polygon([]float64{bx - w, by, tx - 0.6, ty + 1, tx, ty, tx + 0.7, ty + 1.5, bx + w, by, bx, by + w*0.5}, func(x, y int) color.RGBA {
		px, py := float64(x)+0.5, float64(y)+0.5
		up := (by - py) / h
		if px < ridge(py) {
			return obsidianOre.pick(0.3+up*0.24-math.Mod(py*0.37+px*0.2, 1)*0.14, x, y)
		}
		return obsidianOre.pick(0.06+up*0.14, x, y)
	})
	for y := int(ty) + 1; float64(y) < by-h*0.25; y++ { // the edge of the glass
		x := int(math.Floor(ridge(float64(y) + 0.5)))
		if !g.opaque(x, y) {
			continue
		}
		col := obsidianOre[4]
		if float64(y) < ty+h*0.3 {
			col = obsidianOre[5]
		}
		g.put(x, y, col)
	}
	g.put(int(math.Floor(tx)), int(ty)+1, colGlint)
	overInked(c, g, 0.8)
}

// genObsidianOre is a low heap of ash-dark rock with blades of black
// volcanic glass standing out of it.
func genObsidianOre(v int) *canvas {
	c := oreHost(v, ashRock, 0.62, 1)
	for _, s := range [][5]float64{{9, 23, 5.5, 9, 2.7}, {20.5, 23, 23.5, 7.5, 2.9}, {14.5, 24, 13.5, 3.5, 3.4}, {4.5, 25, 2.5, 17.5, 1.9}, {25, 25.5, 27.5, 16.5, 1.9}, {18, 26, 19, 15, 2.2}} {
		obsidianShard(c, oreX(v, s[0]), s[1], oreX(v, s[2]), s[3], s[4])
	}
	oreBlockOn(c, v, 3, ashRock, 0.5, 1) // rubble heaped at their feet
	c.outline(0.8)
	return c
}

// genCoalCanvas is a coal seam: grey rock banded with glossy black coal,
// blocks of it broken out on top and fallen at its foot.
func genCoalCanvas(v int) (*canvas, float64, float64) {
	c := oreHost(v, stoneRamp, 0.92, 0.95)
	tilt := 0.42
	if v == 1 {
		tilt = -tilt
	}
	for y := 0; y < oreH; y++ {
		for x := 0; x < oreW; x++ {
			b := math.Mod(float64(y)+float64(x)*tilt+vnoise(float64(x)*0.3, 0, 170)*2+40, 7)
			if !interior(c, x, y) || b >= 2.6 {
				continue
			}
			col := coalOre.at(int(b))
			if b < 1 && hashf(x/2, y, 171) < 0.45 {
				col = colGloss // along the top of a seam
			}
			c.set(x, y, col)
		}
	}
	for i, n := range [][3]float64{{12, 12, 4.2}, {20, 15.5, 3.2}, {5.5, 20.5, 2.6}, {16.5, 24.5, 2.4}, {25, 24.5, 1.8}} {
		g := newCanvas(oreW, oreH)
		x, y, r := oreX(v, n[0]), n[1], n[2]
		j := func(k int) float64 { return (hashf(i, k, 172) - 0.5) * r * 0.4 }
		g.polygon([]float64{x - r, y + j(0), x - r*0.5 + j(1), y - r*0.9, x + r*0.6, y - r + j(2), x + r, y - r*0.2 + j(3), x + r*0.7, y + r*0.8, x - r*0.4 + j(4), y + r*0.9}, func(px, py int) color.RGBA {
			u, w := (float64(px)+0.5-x)/r, (float64(py)+0.5-y)/r
			switch s := u + w*0.9; {
			case s < -0.95:
				return coalOre[4]
			case s < -0.55:
				return colGloss
			case s < -0.3:
				return coalOre[3]
			case u > 0.25:
				return coalOre.at(int(1.2 - w))
			default:
				return coalOre.pick(0.45-w*0.2, px, py)
			}
		})
		if r > 3 {
			g.put(int(x-r*0.35), int(y-r*0.45), colGlint)
		}
		overInked(c, g, 0.85)
	}
	c.outline(0.85)
	return c, oreW / 2, oreGround
}

// genOreCanvas is variant v of the deposit of ore o: flint in chalk, jade,
// agate, or obsidian.
func genOreCanvas(o, v int) (*canvas, float64, float64) {
	var c *canvas
	switch o {
	case 0:
		c = genFlintOre(v)
	case 1:
		c = genJadeOre(v)
	case 2:
		c = genAgateOre(v)
	default:
		c = genObsidianOre(v)
	}
	return c, oreW / 2, oreGround
}

// genFurnaceCanvas is a domed stone furnace with a chimney, its mouth dark
// (lit=false) or glowing.
func genFurnaceCanvas(lit bool) (*canvas, float64, float64) {
	c := newCanvas(26, 28)
	const cx, cy, rx, ry, base = 13.0, 15.0, 11.0, 10.5, 25
	// The chimney, behind the dome.
	c.rect(16, 1, 5, 10, func(x, y int) color.RGBA { return brickRamp.pick(0.8-float64(x-16)*0.15, x, y) })
	c.rect(16, 1, 5, 2, func(x, y int) color.RGBA { return ashRamp.at(1 + (x+y)%2) })
	// The dome of bricks, lit from the top-left, mortar between the rows.
	for y := 0; y < base; y++ {
		for x := 0; x < c.w; x++ {
			nx, ny := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if nx*nx+ny*ny > 1 && !(ny > 0 && math.Abs(nx) <= 1) {
				continue
			}
			l := lum(nx, math.Min(ny, 0.6))
			row := y / 3
			if y%3 == 0 || (x+row*3)%6 == 0 {
				l -= 0.35 // mortar
			}
			c.set(x, y, brickRamp.pick(l, x, y))
		}
	}
	// The mouth: an arch in the front.
	for y := 15; y < base; y++ {
		for x := 8; x < 18; x++ {
			nx, ny := (float64(x)+0.5-13)/5, (float64(y)+0.5-19)/4.5
			if ny < 0 && nx*nx+ny*ny > 1 {
				continue
			}
			col := ashRamp.at(int(float64(y-15) / 4))
			if lit {
				col = fireRamp.pick(0.25+float64(y-15)/10*0.8-math.Abs(nx)*0.4+(hashf(x, y, 61)-0.5)*0.3, x, y)
			}
			c.set(x, y, col)
		}
	}
	c.outline(0.8)
	return c, cx, base - 0.5
}

// --- icons ------------------------------------------------------------------

// iconOreCanvas is a lump of rock with ore o in it.
func iconOreCanvas(o int) *canvas {
	c := newCanvas(16, 16)
	c.ellipse(8, 9.5, 6.5, 5.2, func(x, y int, nx, ny float64) color.RGBA { return stoneRamp.pick(lum(nx, ny)*0.85, x, y) })
	c.ellipse(6, 7.5, 3.5, 3, func(x, y int, nx, ny float64) color.RGBA { return stoneRamp.pick(lum(nx, ny)*0.95, x, y) })
	c.outline(0.8)
	rng := rand.New(rand.NewPCG(uint64(o), 77))
	if look := oreLooks[o]; look.crystal {
		crystals(c, rng, 3, look.fleck, 5)
	} else {
		flecks(c, rng, 5, look.fleck)
	}
	return c
}

// iconBarCanvas is worked stone of rp, ready to be hafted: a leaf-shaped
// blade knapped all over into shallow scars, or for agate, banded.
func iconBarCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{2, 13.5, 5, 7, 10, 3, 14, 2, 12.5, 6.5, 8, 11.5}, func(x, y int) color.RGBA {
		u := float64(x-y) / 12 // across the blade
		l := 0.78 - u*0.35
		if hashf(x/2, y/2, 61) < 0.35 {
			l -= 0.18 // flake scars
		}
		if rp[0] == agateRamp[0] && (x+y)%3 == 0 {
			l += 0.2 // bands
		}
		return rp.pick(l, x, y)
	})
	c.line(4, 12, 12.5, 3.5, func(x, y int, _ float64) color.RGBA { return rp[len(rp)-1] }) // the ridge
	c.outline(0.8)
	return c
}

// iconDiamondCanvas is a shard of obsidian: black volcanic glass with a
// sheen down its sharp edges.
func iconDiamondCanvas() *canvas {
	c := newCanvas(16, 16)
	rp := obsidianRamp
	c.polygon([]float64{3, 13, 2, 8, 6, 3, 11, 2, 14, 5, 12, 10, 8, 14}, func(x, y int) color.RGBA {
		l := 0.45 - float64(x+y-10)/18*0.3
		if (x*2+y)%5 == 0 {
			l += 0.25 // conchoidal ripples
		}
		return rp.pick(l, x, y)
	})
	c.line(6, 3, 3, 12, func(x, y int, _ float64) color.RGBA { return rp[4] }) // the glassy edge
	c.set(10, 3, rp[4])
	c.outline(0.8)
	return c
}

// iconAxeCanvas, iconPickaxeCanvas and iconHoeCanvas draw a tool with its
// head in rp.
func iconAxeCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	c.line(3, 15, 10, 3, func(x, y int, t float64) color.RGBA { return woodRamp.pick(0.8, x, y) })
	c.line(4, 15, 11, 3, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.6, x, y) })
	c.polygon([]float64{7, 1, 13, 0, 15, 6, 10, 7}, func(x, y int) color.RGBA {
		return rp.pick(0.95-float64(y)*0.08+float64(x-10)*0.03, x, y)
	})
	c.line(14, 1, 15, 5, func(x, y int, t float64) color.RGBA { return rp[len(rp)-1] }) // the sharp edge
	for _, b := range [][2]int{{8, 6}, {9, 7}, {8, 7}} {
		c.set(b[0], b[1], strawRamp[3]) // lashing
	}
	c.outline(0.85)
	return c
}

func iconPickaxeCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	c.line(4, 15, 9, 4, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.8, x, y) })
	c.line(5, 15, 10, 4, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.55, x, y) })
	// A curved head across the top of the handle.
	for s := 0; s <= 30; s++ {
		t := float64(s) / 30
		a := math.Pi * (1.08 + 0.84*t)
		x := 9.5 + math.Cos(a)*7
		y := 7.5 + math.Sin(a)*5.5
		for d := 0; d < 2; d++ {
			c.set(int(x), int(y)+d, rp.pick(0.95-float64(d)*0.35-t*0.15, int(x), int(y)+d))
		}
	}
	c.set(9, 4, ropeRamp[2])
	c.set(10, 5, ropeRamp[1])
	c.outline(0.85)
	return c
}

// hoeBlade is the head of a hoe whose handle ends at (hx, hy): a thin arm
// reaching dx sideways and a flat blade hanging down from its end, a "7"
// that does not read as an axe.
func hoeBlade(c *canvas, hx, hy, dx float64, rp ramp) {
	c.polygon([]float64{hx, hy - 0.5, hx + dx, hy - 0.5, hx + dx + 0.5, hy + 6, hx + dx - 2.5, hy + 5.5, hx + dx - 2.5, hy + 1.2, hx, hy + 1.2},
		func(x, y int) color.RGBA { return rp.pick(0.95-(float64(y)-hy)*0.08, x, y) })
}

func iconHoeCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	c.line(2, 15, 7, 3, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.8, x, y) })
	c.line(3, 15, 8, 3, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.55, x, y) })
	hoeBlade(c, 7, 2.5, 7, rp)
	c.set(7, 3, ropeRamp[2])
	c.set(8, 4, ropeRamp[1])
	c.outline(0.85)
	return c
}

// genToolOf is an axe with its head in rp, as swung: pivot at the grip.
func genToolOf(rp ramp) sprite {
	c := newCanvas(10, 14)
	c.rect(4, 3, 2, 11, func(x, y int) color.RGBA { return woodRamp[1+(5-x)] })
	c.polygon([]float64{4, 1, 9, 0, 9, 6, 4, 5}, func(x, y int) color.RGBA {
		return rp.pick(0.95-float64(y)*0.1, x, y)
	})
	c.outline(0.8)
	return mk(c, 5, 12)
}

// genHeldPickOf is a pickaxe with its head in rp, as held.
func genHeldPickOf(rp ramp) sprite {
	c := newCanvas(12, 14)
	c.rect(5, 3, 2, 11, func(x, y int) color.RGBA { return woodRamp[1+(6-x)] })
	for s := 0; s <= 20; s++ {
		t := float64(s) / 20
		a := math.Pi * (1.1 + 0.8*t)
		x, y := 6+math.Cos(a)*5.5, 4.5+math.Sin(a)*3.5
		c.set(int(x), int(y), rp.pick(0.95-t*0.3, int(x), int(y)))
		c.set(int(x), int(y)+1, rp.pick(0.5, int(x), int(y)+1))
	}
	c.outline(0.8)
	return mk(c, 6, 12)
}

// genHeldHoeOf is a hoe with its blade in rp, as held.
func genHeldHoeOf(rp ramp) sprite {
	c := newCanvas(13, 16)
	c.rect(4, 3, 2, 13, func(x, y int) color.RGBA { return woodRamp[1+(5-x)] })
	hoeBlade(c, 5, 2, 6, rp)
	c.outline(0.8)
	return mk(c, 5, 14)
}

// oreArt is the art of the ores, the furnace and the tools of every tier.
type oreArt struct {
	rocks             [sim.NumTiers - 1][2]sprite // each ore, in two boulder shapes
	furnace           [2]sprite                   // cold, lit
	axes, picks, hoes [sim.NumTiers]sprite        // in the hand, by tier: the axe as swung, the others as held
}

func genOreArt() oreArt {
	var a oreArt
	for o := range a.rocks {
		for v := range a.rocks[o] {
			c, ax, ay := genOreCanvas(o, v)
			a.rocks[o][v] = mk(c, ax, ay)
		}
	}
	for i, lit := range []bool{false, true} {
		c, ax, ay := genFurnaceCanvas(lit)
		a.furnace[i] = mk(c, ax, ay)
	}
	for t, rp := range tierRamps {
		a.axes[t], a.picks[t], a.hoes[t] = genToolOf(rp), genHeldPickOf(rp), genHeldHoeOf(rp)
	}
	return a
}

// genOreIcons draws the bag icons of the ores, bars and diamonds, and of the
// axe, pickaxe and hoe of every tier.
func genOreIcons(icons *[sim.NumItems]*ebiten.Image) {
	for o, k := range []sim.ItemKind{sim.ItemCopperOre, sim.ItemIronOre, sim.ItemPlatinumOre} {
		icons[k] = iconOreCanvas(o).image()
	}
	icons[sim.ItemDiamond] = iconDiamondCanvas().image()
	icons[sim.ItemCopperBar] = iconBarCanvas(flintRamp).image()
	icons[sim.ItemIronBar] = iconBarCanvas(jadeRamp).image()
	icons[sim.ItemPlatinumBar] = iconBarCanvas(agateRamp).image()
	for t := sim.TierStone; t < sim.NumTiers; t++ {
		icons[sim.ToolItem(sim.ToolAxe, t)] = iconAxeCanvas(tierRamps[t]).image()
		icons[sim.ToolItem(sim.ToolPickaxe, t)] = iconPickaxeCanvas(tierRamps[t]).image()
		icons[sim.ToolItem(sim.ToolHoe, t)] = iconHoeCanvas(tierRamps[t]).image()
	}
}

// oreRock is the sprite of the ore entity e.
func (g *Game) oreRock(e protocol.EntityView, tier sim.Tier) sprite {
	return g.art.ore.rocks[tier-1][sim.HashTile(int(e.TX), int(e.TY))/8%2]
}

// furnaceSprite is the furnace e, glowing while it burns.
func (g *Game) furnaceSprite(e protocol.EntityView) sprite {
	if e.Fuel > 0 {
		return g.art.ore.furnace[1]
	}
	return g.art.ore.furnace[0]
}

// toolTier is what tool k is made of (stone for anything that is no tool).
func toolTier(k sim.ItemKind) sim.Tier {
	_, t := k.ToolKind()
	return t
}

// isHoe reports whether k is a hoe of any tier.
func isHoe(k sim.ItemKind) bool {
	c, _ := k.ToolKind()
	return c == sim.ToolHoe
}

// has reports whether I carry a tool of class c, of any tier.
func (g *Game) has(c sim.ToolClass) bool {
	_, ok := g.curr.Self.Inv.BestTool(c)
	return ok
}

// orePick is the pickaxe tier an ore of kind k needs; ok is false for
// anything that is not an ore.
func orePick(k sim.EntityKind) (need sim.Tier, ok bool) {
	t, ok := sim.OreTier(k)
	return t - 1, ok
}

// canMine reports whether I could mine k, if it is an ore (true otherwise).
func (g *Game) canMine(k sim.EntityKind) bool {
	need, ok := orePick(k)
	if !ok {
		return true
	}
	best, has := g.curr.Self.Inv.BestTool(sim.ToolPickaxe)
	return has && best >= need
}

// byFurnace reports whether I stand beside a lit furnace: the server's test
// for smelting.
func (g *Game) byFurnace() bool {
	for _, p := range g.curr.Players {
		if p.ID != g.myID {
			continue
		}
		for _, e := range g.curr.Entities {
			if e.Kind != sim.EntityFurnace || e.Fuel <= 0 {
				continue
			}
			dx, dy := float64(e.TX)+0.5-float64(p.X), float64(e.TY)+0.5-float64(p.Y)
			if dx*dx+dy*dy <= sim.FurnaceRadius*sim.FurnaceRadius {
				return true
			}
		}
	}
	return false
}

// The bag's notes on ores, bars and the tools of every tier.
func init() {
	notes := map[sim.ItemKind]string{
		sim.ItemCopperOre:   "Đào ở mạch đá lửa trên đồi đá quanh núi (cần Cúp đá). Nung ở Lò nung cho giòn rồi ghè thành Lưỡi đá lửa.",
		sim.ItemIronOre:     "Đào ở mạch ngọc bích trong hang tối trong núi — mang đuốc (cần Cúp đá lửa). Mài ở Lò nung thành Ngọc bích mài.",
		sim.ItemPlatinumOre: "Đào ở mạch mã não giữa sa mạc (cần Cúp ngọc bích). Nung ở Lò nung thành Mã não mài.",
		sim.ItemDiamond:     "Đá thủy tinh núi lửa, sắc nhất thời đồ đá. Đào ở vùng tro quanh núi lửa (cần Cúp mã não). Không cần nung.",
		sim.ItemCopperBar:   "Đá lửa nung ở Lò nung rồi ghè thành lưỡi. Làm rìu, cúp, cuốc, dao đá lửa.",
		sim.ItemIronBar:     "Ngọc bích mài nhẵn ở Lò nung. Làm rìu, cúp, cuốc, dao ngọc bích.",
		sim.ItemPlatinumBar: "Mã não nung ở Lò nung. Làm rìu, cúp, cuốc, dao mã não và Giáp mã não.",
	}
	mines := []string{"tảng đá, mỏ than và mạch đá lửa", "cả mạch ngọc bích", "cả mạch mã não", "cả mạch hắc diện thạch", "mọi loại mạch đá"}
	for t := sim.TierCopper; t < sim.NumTiers; t++ {
		notes[sim.ToolItem(sim.ToolAxe, t)] = "Dụng cụ. Chặt cây, xương rồng và dỡ tường nhanh hơn rìu bậc dưới."
		notes[sim.ToolItem(sim.ToolPickaxe, t)] = "Dụng cụ. Đập đá nhanh hơn cúp bậc dưới, đào được " + mines[t] + "."
		notes[sim.ToolItem(sim.ToolHoe, t)] = "Cầm trên tay (ô nhanh), nhấn Space để xới đất; xới bằng cuốc tốt hay đào được khoai hơn."
	}
	for k, s := range notes {
		itemInfo[k] = s
	}
}

// isOre reports whether k is an ore in the world.
func isOre(k sim.EntityKind) bool {
	_, ok := sim.OreTier(k)
	return ok
}
