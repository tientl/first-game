package client

import (
	"cmp"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
)

// Broadleaf trees come in several shapes, each with its own branching, in
// three leaf colours, and some are hung with vines. Every tree is drawn at a
// scale s: small trees at treeScale[0], big ones at treeScale[1].

type treeShape uint8

const (
	shapeRound   treeShape = iota // one round crown on a straight trunk
	shapeTall                     // a narrow column of leaf clumps
	shapeFork                     // the trunk splits in two, a crown on each limb
	shapeSpread                   // a leaning trunk fanning out under a flat, wide crown
	shapeOak                      // three lobes on bare, crooked branches
	shapeWeeping                  // a round crown with curtains of leaves hanging down
	numTreeShapes
)

var (
	vineRamp    = mkRamp(0x1b331d, 0x2a4d27, 0x3b6a2f, 0x568a3a)
	vineFlowers = []color.RGBA{hex(0xf0c6dc), hex(0xfff0b0), hex(0xf6f6f0)}
)

// limb is one stretch of wood: a curve from (x0, y0) bending towards
// (cx, cy) and ending at (x1, y1), tapering from w0 to w1 half-thick.
type limb struct{ x0, y0, cx, cy, x1, y1, w0, w1 float64 }

// at is the point a fraction t along l.
func (l limb) at(t float64) (x, y float64) {
	u := 1 - t
	return u*u*l.x0 + 2*u*t*l.cx + t*t*l.x1, u*u*l.y0 + 2*u*t*l.cy + t*t*l.y1
}

// drawLimb paints l as bark lit from the left, streaked along its length.
func drawLimb(c *canvas, l limb) {
	n := int(math.Hypot(l.x1-l.x0, l.y1-l.y0)*2) + 2
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := l.at(t)
		w := lerpf(l.w0, l.w1, t)
		for py := int(math.Floor(y - w)); py <= int(math.Ceil(y+w)); py++ {
			for px := int(math.Floor(x - w)); px <= int(math.Ceil(x+w)); px++ {
				dx, dy := float64(px)+0.5-x, float64(py)+0.5-y
				if dx*dx+dy*dy > w*w+0.25 {
					continue
				}
				k := trunkRamp.idx(0.8-(dx/max(w, 0.5)+1)/2*0.7, px, py)
				if hashf(px, py/3, 7) < 0.22 {
					k--
				}
				c.set(px, py, trunkRamp.at(k))
			}
		}
	}
}

// crown is a mass of leaves: its blobs, and the centre and extent its
// shading is lit around.
type crown struct {
	blobs          []blob
	cx, cy, rx, ry float64
}

// roundCrown is a ball of leaves about R across centred on (cx, cy).
func roundCrown(cx, cy, R float64, rng *rand.Rand) crown {
	bs := []blob{{cx, cy + 1, R * 0.75}}
	n := 5 + int(R/4)
	for i := range n {
		a := 2*math.Pi*float64(i)/float64(n) + rng.Float64()*0.5
		bs = append(bs, blob{cx + math.Cos(a)*R*0.72, cy + math.Sin(a)*R*0.55, R * (0.34 + rng.Float64()*0.14)})
	}
	bs = append(bs, blob{cx - 2 + rng.Float64()*4, cy - R*0.5, R * 0.45})
	return crown{bs, cx, cy, R * 1.25, R * 1.1}
}

// flatCrown is a wide, shallow canopy half wide either side of cx.
func flatCrown(cx, cy, half, thick float64, rng *rand.Rand) crown {
	var bs []blob
	n := int(half*2/thick) + 3
	for i := 0; i <= n; i++ {
		t := float64(i)/float64(n)*2 - 1
		bs = append(bs, blob{
			cx + t*half,
			cy + (rng.Float64()-0.5)*thick*0.5 + t*t*thick*0.4,
			thick * (0.6 + rng.Float64()*0.3) * (1 - 0.3*t*t),
		})
	}
	bs = append(bs, blob{cx + (rng.Float64()-0.5)*half*0.6, cy - thick*0.45, thick * 0.7})
	return crown{bs, cx, cy, half * 1.15, thick * 1.2}
}

// genBroadleaf draws a leafy tree of the given shape at scale s. seed varies
// the details, so two trees of a shape are not twins.
func genBroadleaf(shape treeShape, palette int, vines bool, s float64, seed uint64) sprite {
	rng := rand.New(rand.NewPCG(seed, 99))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	c := newCanvas(int(math.Ceil(56*s)), int(math.Ceil(64*s)))
	bx, by := math.Round(float64(c.w)/2), float64(c.h-3) // ground point
	girth := 2.5 * s * s                                 // trunk half-thickness: big trees are stout
	lean := r(-2.5, 2.5) * s

	var limbs []limb
	var crowns []crown
	trunkTo := func(tx, ty float64) {
		limbs = append(limbs, limb{bx, by, bx + (tx-bx)*0.15 + r(-1.5, 1.5)*s, (by + ty) / 2, tx, ty, girth, girth * 0.7})
	}
	branch := func(x0, y0, x1, y1, w float64) {
		mx, my := (x0+x1)/2+r(-1.5, 1.5)*s, (y0+y1)/2+r(-0.5, 2)*s // sagging a little
		limbs = append(limbs, limb{x0, y0, mx, my, x1, y1, w, max(0.5, w*0.45)})
	}

	switch shape {
	case shapeRound:
		tx, ty := bx+lean, by-26*s
		trunkTo(tx, ty)
		crowns = append(crowns, roundCrown(tx, by-33*s, r(13, 15)*s, rng))
	case shapeTall:
		tx, ty := bx+lean*0.5, by-22*s
		trunkTo(tx, ty)
		for i, R := range []float64{9.5, 9, 7.5} {
			crowns = append(crowns, roundCrown(tx+r(-1.5, 1.5)*s, by-(28+float64(i)*8.5)*s, R*s*r(0.95, 1.1), rng))
		}
	case shapeFork:
		fx, fy := bx+lean*0.5, by-r(15, 19)*s
		trunkTo(fx, fy)
		lx, ly := fx-r(8, 11)*s, by-r(30, 33)*s
		rx, ry := fx+r(7, 10)*s, by-r(33, 37)*s
		branch(fx, fy, lx, ly, girth*0.8)
		branch(fx, fy, rx, ry, girth*0.8)
		crowns = append(crowns, roundCrown(lx, ly-2*s, r(9, 10.5)*s, rng), roundCrown(rx, ry-2*s, r(9, 10.5)*s, rng))
	case shapeSpread:
		lean = math.Copysign(r(3, 5), lean) * s
		tx, ty := bx+lean, by-r(21, 24)*s
		trunkTo(tx, ty)
		cy := by - r(31, 34)*s
		for _, dx := range []float64{-15, -6, 5, 14} {
			branch(tx, ty, tx+(dx+r(-2, 2))*s, cy+r(0, 3)*s, girth*0.6)
		}
		crowns = append(crowns, flatCrown(tx, cy-2*s, r(17, 20)*s, 6.5*s, rng))
	case shapeOak:
		tx, ty := bx+lean*0.5, by-r(17, 21)*s
		trunkTo(tx, ty)
		for _, p := range [][3]float64{{-r(9, 12), r(27, 31), 8}, {r(8, 11), r(25, 29), 8.5}, {r(-2, 2), r(37, 40), 10}} {
			ex, ey := tx+p[0]*s, by-p[1]*s
			branch(tx, ty, ex, ey, girth*0.7)
			crowns = append(crowns, roundCrown(ex, ey-s, p[2]*s*r(0.95, 1.1), rng))
		}
	case shapeWeeping:
		tx, ty := bx+lean, by-27*s
		trunkTo(tx, ty)
		crowns = append(crowns, roundCrown(tx, by-34*s, r(12, 13.5)*s, rng))
	}
	trunk := limbs[0]

	// Roots, and a few side branches off the trunk: some carry a tuft of
	// leaves, some are bare.
	for _, side := range []float64{-1, 1} {
		limbs = append(limbs, limb{bx, by - 3*s, bx + side*2*s, by - 0.5*s, bx + side*r(3, 4.5)*s, by, girth * 0.6, 0.6})
	}
	for range rng.IntN(3) {
		x0, y0 := trunk.at(r(0.45, 0.8))
		side := float64(1 - 2*rng.IntN(2))
		x1, y1 := x0+side*r(5, 9)*s, y0-r(2, 6)*s
		branch(x0, y0, x1, y1, max(0.7, girth*0.4))
		if rng.Float64() < 0.6 {
			crowns = append(crowns, roundCrown(x1, y1-s, r(3, 4.5)*s, rng))
		}
	}

	for _, l := range limbs {
		drawLimb(c, l)
	}
	// Leaves, back (higher up) to front.
	slices.SortStableFunc(crowns, func(a, b crown) int { return cmp.Compare(a.cy, b.cy) })
	for i, cr := range crowns {
		leafCluster(c, cr.blobs, cr.cx, cr.cy, cr.rx, cr.ry, leafRamps[palette], 100+uint32(palette)+uint32(i)*17)
	}
	if shape == shapeWeeping {
		hangCurtains(c, rng, leafRamps[palette], s)
	}
	shadeUnderCrown(c)
	if vines {
		growVines(c, rng, trunk, s)
	}
	c.outline(0.75)
	return trimmed(c, bx, by)
}

// crownUnderside is, per column, the lowest leaf pixel of the crown, or -1.
func crownUnderside(c *canvas) []int {
	bottom := make([]int, c.w)
	for x := range c.w {
		bottom[x] = -1
		for y := c.h - 1; y >= 0; y-- {
			if p := c.at(x, y); p.A > 0 && !isTrunk(p) {
				bottom[x] = y
				break
			}
		}
	}
	return bottom
}

// hangCurtains drapes strands of leaves from the crown's underside.
func hangCurtains(c *canvas, rng *rand.Rand, rp ramp, s float64) {
	for x, y0 := range crownUnderside(c) {
		if y0 < 0 || rng.Float64() < 0.3 {
			continue
		}
		n := int((5 + rng.Float64()*10) * s)
		for i := range n {
			y := y0 + i
			l := 0.5 - float64(i)/float64(n)*0.3 + (hashf(x, y, 77)-0.5)*0.35
			c.set(x, y, rp.pick(l, x, y))
		}
	}
}

// shadeUnderCrown darkens the wood just below where leaves hang over it.
func shadeUnderCrown(c *canvas) {
	for y := 1; y < c.h; y++ {
		for x := range c.w {
			above := c.at(x, y-1)
			if above.A > 0 && !isTrunk(above) && isTrunk(c.at(x, y)) {
				c.set(x, y, trunkRamp[0])
				if isTrunk(c.at(x, y+1)) {
					c.set(x, y+1, trunkRamp[1])
				}
			}
		}
	}
}

// growVines hangs vines from the crown, some flowering, and often winds one
// up the trunk.
func growVines(c *canvas, rng *rand.Rand, trunk limb, s float64) {
	bottom := crownUnderside(c)
	var cols []int
	for x, y := range bottom {
		if y >= 0 {
			cols = append(cols, x)
		}
	}
	for range 4 + rng.IntN(5) {
		if len(cols) == 0 {
			break
		}
		x := cols[rng.IntN(len(cols))]
		y0, n, ph := bottom[x]-1, int((5+rng.Float64()*11)*s), rng.Float64()*6
		for j := range n {
			px, py := x+int(math.Round(math.Sin(float64(j)*0.5+ph)*0.7)), y0+j
			c.set(px, py, vineRamp.at(1+(j/2)%2))
			if j%3 == 1 { // a leaf, on alternate sides
				side := 1 - 2*((j/3)%2)
				c.set(px+side, py, vineRamp.pick(0.85, px+side, py))
			}
			if rng.Float64() < 0.05 {
				c.set(px, py, vineFlowers[rng.IntN(len(vineFlowers))])
			}
		}
	}
	if rng.Float64() < 0.3 {
		return
	}
	ph := rng.Float64() * 6
	for t := 0.04; t < 0.9; t += 0.015 {
		x, y := trunk.at(t)
		w := lerpf(trunk.w0, trunk.w1, t)
		a := y*0.6 + ph // once around every ten pixels or so
		if math.Cos(a) < -0.1 {
			continue // behind the trunk
		}
		px, py := int(math.Floor(x+math.Sin(a)*w)), int(math.Floor(y))
		c.set(px, py, vineRamp.pick(0.45+0.4*math.Cos(a), px, py))
		if hashf(px, py, 91) < 0.25 {
			c.set(px+1, py, vineRamp.pick(0.9, px+1, py))
		}
	}
}

// genPine draws a conifer at scale s: variant sets the number of tiers, and
// seed the ragged edges.
func genPine(variant int, s float64, seed uint64) sprite {
	rng := rand.New(rand.NewPCG(seed, 121))
	c := newCanvas(int(math.Ceil(34*s)), int(math.Ceil(62*s)))
	bx, by := c.w/2, c.h-3               // ground point
	tw := max(2, int(math.Round(2*s*s))) // half the trunk's width
	for y := by - int(math.Round(11*s)); y <= by; y++ {
		for x := bx - tw; x < bx+tw; x++ {
			c.set(x, y, trunkRamp.pick(0.7-float64(x-(bx-tw))/float64(2*tw-1)*0.6, x, y))
		}
	}
	tiers := 4 + variant%3
	lean := (rng.Float64() - 0.5) * 2.5 * s
	for k := range tiers {
		fk, ft := float64(k), float64(tiers)
		yb := float64(by) - (8+fk*36/ft)*s
		yt := yb - (17-fk*0.5)*s
		hw := (14 - fk*10/ft) * s * (0.85 + rng.Float64()*0.3)
		ox := float64(bx) + lean*fk/ft + (rng.Float64()-0.5)*1.5
		for y := int(yt); y <= int(yb)+2; y++ {
			for x := int(ox - hw - 1); x <= int(ox+hw+1); x++ {
				px := float64(x) + 0.5 - ox
				f := (float64(y) + 0.5 - yt) / (yb - yt)
				saw := 0.0
				if (x/2)%2 == 0 {
					saw = 1.5
				}
				if f < 0 || math.Abs(px) > hw*math.Min(f, 1)+0.3 || float64(y) > yb+saw {
					continue
				}
				l := 0.62 - px/hw*0.38 + (1-f)*0.12 + (vnoise(float64(x)*0.6, float64(y)*0.8, 120+uint32(variant))-0.5)*0.3
				if f > 0.82 {
					l -= 0.2
				}
				if hashf(x, y, 121) < 0.08 {
					l += 0.15
				}
				c.set(x, y, pineRamp.pick(l, x, y))
			}
		}
	}
	c.outline(0.8)
	return trimmed(c, float64(bx), float64(by))
}

// pineVariants is how many conifers are drawn per size.
const pineVariants = 6

// genTrees draws every tree at scale s: each leafy shape in each leaf
// colour, the same trees hung with vines, and the conifers.
func genTrees(s float64) (trees, vined, pines []sprite) {
	for sh := treeShape(0); sh < numTreeShapes; sh++ {
		for pal := range leafRamps {
			seed := uint64(sh)*7 + uint64(pal)*31 + 5
			trees = append(trees, genBroadleaf(sh, pal, false, s, seed))
			vined = append(vined, genBroadleaf(sh, pal, true, s, seed))
		}
	}
	for v := range pineVariants {
		pines = append(pines, genPine(v, s, uint64(v)*13+1))
	}
	return trees, vined, pines
}

// trimmed crops c to its opaque pixels, keeping the anchor (ax, ay) on the
// same pixel, so a sprite's top is the top of what is drawn.
func trimmed(c *canvas, ax, ay float64) sprite {
	x0, y0, x1, y1 := c.w, c.h, -1, -1
	for y := range c.h {
		for x := range c.w {
			if c.px[y*c.w+x].A > 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return mk(c, ax, ay)
	}
	t := newCanvas(x1-x0+1, y1-y0+1)
	for y := range t.h {
		copy(t.px[y*t.w:(y+1)*t.w], c.px[(y+y0)*c.w+x0:(y+y0)*c.w+x1+1])
	}
	return mk(t, ax-float64(x0), ay-float64(y0))
}
