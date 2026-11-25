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
	if c.hd() {
		drawLimbHD(c, l, trunkRamp, 0.8, 7)
		return
	}
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
	k := hdK() // R is in pixels: three to an art pixel at HD
	bs := []blob{{cx, cy + k, R * 0.75}}
	n := 5 + int(R/k/4)
	for i := range n {
		a := 2*math.Pi*float64(i)/float64(n) + rng.Float64()*0.5
		bs = append(bs, blob{cx + math.Cos(a)*R*0.72, cy + math.Sin(a)*R*0.55, R * (0.34 + rng.Float64()*0.14)})
	}
	bs = append(bs, blob{cx + (rng.Float64()*4-2)*k, cy - R*0.5, R * 0.45})
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
	return genBroadleafIn(shape, leafRamps[palette], uint32(palette), vines, s, seed)
}

// genBroadleafIn is genBroadleaf with leaves of rp, their texture varied by tex.
func genBroadleafIn(shape treeShape, rp ramp, tex uint32, vines bool, s float64, seed uint64) sprite {
	return trimmed(broadleafCanvas(shape, rp, tex, vines, s, seed))
}

// broadleafCanvas draws genBroadleafIn's tree and gives its ground point.
//
// At HD the tree is drawn three times the size in image pixels (the canvas
// is marked, see markHD), and the ground point is in image pixels too:
// trimmed and mkUp make the sprite's anchor of it.
func broadleafCanvas(shape treeShape, rp ramp, tex uint32, vines bool, s float64, seed uint64) (*canvas, float64, float64) {
	rng := rand.New(rand.NewPCG(seed, 99))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	girth := 2.5 * s * s * k // trunk half-thickness: big trees are stout
	s *= k
	c := markHD(newCanvas(int(math.Ceil(56*s)), int(math.Ceil(64*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k // ground point
	lean := r(-2.5, 2.5) * s

	var limbs []limb
	var crowns []crown
	trunkTo := func(tx, ty float64) {
		limbs = append(limbs, limb{bx, by, bx + (tx-bx)*0.15 + r(-1.5, 1.5)*s, (by + ty) / 2, tx, ty, girth, girth * 0.7})
	}
	branch := func(x0, y0, x1, y1, w float64) {
		mx, my := (x0+x1)/2+r(-1.5, 1.5)*s, (y0+y1)/2+r(-0.5, 2)*s // sagging a little
		limbs = append(limbs, limb{x0, y0, mx, my, x1, y1, w, max(0.5*k, w*0.45)})
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
		limbs = append(limbs, limb{bx, by - 3*s, bx + side*2*s, by - 0.5*s, bx + side*r(3, 4.5)*s, by, girth * 0.6, 0.6 * k})
	}
	for range rng.IntN(3) {
		x0, y0 := trunk.at(r(0.45, 0.8))
		side := float64(1 - 2*rng.IntN(2))
		x1, y1 := x0+side*r(5, 9)*s, y0-r(2, 6)*s
		branch(x0, y0, x1, y1, max(0.7*k, girth*0.4))
		if rng.Float64() < 0.6 {
			crowns = append(crowns, roundCrown(x1, y1-s, r(3, 4.5)*s, rng))
		}
	}

	for _, l := range limbs {
		drawLimb(c, l)
	}
	if c.hd() { // a knot or two in the trunk
		for range 1 + rng.IntN(2) {
			t := r(0.25, 0.7)
			x, y := trunk.at(t)
			w := lerpf(trunk.w0, trunk.w1, t)
			knotHD(c, x+r(-0.4, 0.4)*w, y, min(4, max(2, w*0.3)), trunkRamp)
		}
	}
	// Leaves, back (higher up) to front.
	slices.SortStableFunc(crowns, func(a, b crown) int { return cmp.Compare(a.cy, b.cy) })
	for i, cr := range crowns {
		leafCluster(c, cr.blobs, cr.cx, cr.cy, cr.rx, cr.ry, rp, 100+tex+uint32(i)*17)
	}
	if shape == shapeWeeping {
		hangCurtains(c, rng, rp, s)
	}
	shadeUnderCrown(c)
	if vines {
		growVines(c, rng, trunk, s)
	}
	c.outline(0.75)
	return c, bx, by
}

// leafStrandHD hangs a strand of small leaves n pixels down from (x, y0).
func leafStrandHD(c *canvas, x, y0, n int, rp ramp, lit float64, seed uint32) {
	for i := 0; i < n; i += 3 {
		y := y0 + i
		side := 1 - 2*((i/3)%2) // leaves on alternate sides
		l := lit - float64(i)/float64(n)*0.3 + (hashf(x, y, seed)-0.5)*0.3
		c.put(x, y, rp.pick(l-0.25, x, y))
		c.put(x, y+1, rp.pick(l-0.25, x, y+1))
		c.put(x, y+2, rp.pick(l-0.3, x, y+2))
		c.put(x+side, y+1, rp.pick(l+0.1, x+side, y+1))
		c.put(x+side, y+2, rp.pick(l, x+side, y+2))
		c.put(x+2*side, y+2, rp.pick(l-0.1, x+2*side, y+2))
	}
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
	if c.hd() {
		bottom := crownUnderside(c)
		for x := 1; x < c.w; x += 2 + rng.IntN(2) {
			if y0 := bottom[x]; y0 >= 0 && rng.Float64() >= 0.2 {
				leafStrandHD(c, x, y0-2, int((5+rng.Float64()*10)*s), rp, 0.55, 77)
			}
		}
		return
	}
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
	if c.hd() {
		// Three pixels of deep shade, then a few of half shade.
		for x := range c.w {
			depth := 99
			for y := range c.h {
				p := c.at(x, y)
				switch {
				case p.A == 0:
					depth = 99
				case !isTrunk(p):
					depth = 0
				default:
					depth++
					if depth <= 3 {
						c.put(x, y, trunkRamp[0])
					} else if depth <= 7 && (p != trunkRamp[0]) && (depth <= 5 || dither(x, y) < 0.5) {
						c.put(x, y, trunkRamp[1])
					}
				}
			}
		}
		return
	}
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
		if c.hd() {
			vineHD(c, rng, x, y0-3, n, ph)
			continue
		}
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
	if c.hd() {
		for t := 0.04; t < 0.9; t += 0.004 {
			x, y := trunk.at(t)
			w := lerpf(trunk.w0, trunk.w1, t)
			a := y*0.2 + ph
			if math.Cos(a) < -0.1 {
				continue // behind the trunk
			}
			px, py := int(math.Floor(x+math.Sin(a)*w)), int(math.Floor(y))
			l := 0.4 + 0.4*math.Cos(a)
			c.put(px, py, vineRamp.pick(l, px, py))
			c.put(px+1, py, vineRamp.pick(l-0.25, px+1, py))
			if hashf(px/2, py/3, 91) < 0.3 && py%3 == 0 { // a leaf
				for _, d := range [][3]int{{2, -1, 3}, {3, -1, 3}, {2, 0, 2}, {3, 0, 3}, {4, 0, 2}, {3, 1, 1}} {
					c.put(px+d[0], py+d[1], vineRamp.at(d[2]))
				}
			}
		}
		return
	}
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

// vineHD hangs one vine n pixels down from (x, y0): a thin winding stem with
// a leaf every few pixels, on alternate sides, and now and then a flower.
func vineHD(c *canvas, rng *rand.Rand, x, y0, n int, ph float64) {
	for j := range n {
		px, py := x+int(math.Round(math.Sin(float64(j)*0.17+ph)*2)), y0+j
		c.put(px, py, vineRamp.at(1))
		c.put(px+1, py, vineRamp.at(0))
		if j%5 == 2 { // a leaf: a small pointed shape
			side := 1 - 2*((j/5)%2)
			for _, d := range [][3]int{{1, 0, 3}, {2, 0, 3}, {3, 0, 2}, {1, 1, 2}, {2, 1, 2}, {2, -1, 3}} {
				c.put(px+side*d[0]+(1-side)/2, py+d[1], vineRamp.at(d[2]))
			}
		}
		if rng.Float64() < 0.03 && j > 3 { // a flower: four petals round an eye
			col := vineFlowers[rng.IntN(len(vineFlowers))]
			for _, d := range [][2]int{{0, -1}, {-1, 0}, {1, 0}, {0, 1}} {
				c.put(px+d[0], py+d[1], col)
			}
			c.put(px, py, hex(0xe8b030))
		}
	}
}

// genPine draws a conifer at scale s: variant sets the number of tiers, and
// seed the ragged edges.
func genPine(variant int, s float64, seed uint64) sprite { return genPineOf(variant, s, seed, false) }

// genPineOf draws a conifer; a snowy one has snow lying on its boughs.
func genPineOf(variant int, s float64, seed uint64, snowy bool) sprite {
	if envHD {
		return genPineHD(variant, s, seed, snowy)
	}
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
				col := pineRamp.pick(l, x, y)
				if snowy && (f < 0.38 || f < 0.7 && hashf(x, y, 124) < 0.3) {
					col = snowRamp.pick(l+0.35, x, y) // snow on the bough
				}
				c.set(x, y, col)
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
		return mkUp(c, ax, ay)
	}
	t := newCanvas(x1-x0+1, y1-y0+1)
	t.dens = c.dens
	if c.ol != nil {
		t.ol = make([]bool, len(t.px))
	}
	for y := range t.h {
		copy(t.px[y*t.w:(y+1)*t.w], c.px[(y+y0)*c.w+x0:(y+y0)*c.w+x1+1])
		if c.ol != nil {
			copy(t.ol[y*t.w:(y+1)*t.w], c.ol[(y+y0)*c.w+x0:(y+y0)*c.w+x1+1])
		}
	}
	return mkUp(t, ax-float64(x0), ay-float64(y0))
}

// Prehistoric trees: tree ferns, cycads, araucarias, ginkgos and giant
// redwoods. They are trees like any other in the game (an axe, wood, a
// small and a big size); only their look differs, picked by terrain.

var (
	treeFernRamp  = mkRamp(0x173a1d, 0x245524, 0x37722b, 0x529332, 0x7cb444)
	treeFernTrunk = mkRamp(0x24180f, 0x382618, 0x4e3622, 0x66482e)
	cycadRamp     = mkRamp(0x18301f, 0x25472a, 0x376131, 0x507d39, 0x739a45)
	cycadTrunk    = mkRamp(0x3a2a17, 0x56401f, 0x725a2c, 0x90763c)
	araucariaRamp = mkRamp(0x0d2622, 0x153a30, 0x1f4f3c, 0x2d6648, 0x438054)
	ginkgoRamp    = mkRamp(0x3a561c, 0x587a22, 0x7d9d2b, 0xa6c03b, 0xcfdb59)
	redwoodRamp   = mkRamp(0x35160e, 0x582616, 0x7d3a1f, 0xa0532c, 0xbd7343)
	redwoodLeaf   = mkRamp(0x10291c, 0x183d24, 0x22542c, 0x326c36, 0x4a8743)
)

// drawLimbIn is drawLimb in the wood colours of rp.
func drawLimbIn(c *canvas, l limb, rp ramp, streak uint32) {
	if c.hd() {
		drawLimbHD(c, l, rp, 0.85, streak)
		return
	}
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
				k := rp.idx(0.85-(dx/max(w, 0.5)+1)/2*0.7, px, py)
				if hashf(px, py/3, streak) < 0.22 {
					k--
				}
				c.set(px, py, rp.at(k))
			}
		}
	}
}

// frond draws one fern-like leaf from (x0, y0): a stem arching out along
// angle a (radians, screen space: -π/2 is straight up) for length L,
// drooping at the tip, with leaflets either side that shrink to the tip.
// stiff fronds (cycads) barely droop and have short, dense leaflets.
func frond(c *canvas, x0, y0, a, L float64, rp ramp, stiff bool, seed uint32) {
	if c.hd() {
		frondHD(c, x0, y0, a, L, rp, stiff, seed)
		return
	}
	droop, leaf, every := 0.95, 0.2, 2
	sag := 0.18 // how much leaflets droop
	if stiff {
		droop, leaf, every = -0.05, 0.16, 2
		sag = 0.02
	}
	ex, ey := x0+math.Cos(a)*L, y0+math.Sin(a)*L*0.75+L*droop
	cx, cy := x0+math.Cos(a)*L*0.6, y0+math.Sin(a)*L*0.95
	at := func(t float64) (float64, float64) {
		u := 1 - t
		return u*u*x0 + 2*u*t*cx + t*t*ex, u*u*y0 + 2*u*t*cy + t*t*ey
	}
	n := int(L*1.5) + 4
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := at(t)
		x2, y2 := at(min(1, t+0.02))
		tx, ty := x2-x, y2-y
		d := math.Hypot(tx, ty)
		if d == 0 {
			continue
		}
		nx, ny := -ty/d, tx/d // across the stem
		ln := leaf * L * (1 - t*0.85) * (0.6 + 0.4*math.Sin(t*math.Pi))
		if t > 0.08 && i%every == 0 {
			for _, side := range []float64{-1, 1} {
				for k := 1.0; k <= ln; k++ {
					// Leaflets sweep forward a little, and droop.
					px := x + nx*side*k + tx/d*k*0.6
					py := y + ny*side*k + ty/d*k*0.6 + k*sag
					l := 0.75 - k/ln*0.25 - side*0.12 - t*0.15 + (hashf(int(px), int(py), seed)-0.5)*0.2
					c.set(int(px), int(py), rp.pick(l, int(px), int(py)))
				}
			}
		}
		c.set(int(x), int(y), rp.pick(0.35, int(x), int(y)))
	}
}

// genTreeFern is a tree fern: a slim shaggy trunk under a crown of long,
// arching fronds.
func genTreeFern(s float64, seed uint64) sprite {
	rng := rand.New(rand.NewPCG(seed, 31))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	s *= k
	c := markHD(newCanvas(int(math.Ceil(58*s)), int(math.Ceil(66*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k
	lean := r(-3, 3) * s
	tx, ty := bx+lean, by-r(34, 40)*s
	trunk := limb{bx, by, bx + lean*0.2, (by + ty) / 2, tx, ty, 2.2 * s, 1.5 * s}
	drawLimbIn(c, trunk, treeFernTrunk, 81)
	for t := 0.1; t < 0.95; t += 0.09 / k * 1.5 { // the scars of old fronds ring the trunk
		x, y := trunk.at(t)
		w := lerpf(trunk.w0, trunk.w1, t)
		c.line(x-w, y, x+w, y+k, func(px, py int, _ float64) color.RGBA { return treeFernTrunk[0] })
		if c.hd() { // a pale lip under each scar
			c.line(x-w*0.8, y+1, x+w*0.3, y+k, func(px, py int, _ float64) color.RGBA { return treeFernTrunk[3] })
		}
	}
	n := 7 + rng.IntN(3)
	type fr struct{ a, L float64 }
	var fronds []fr
	for i := range n {
		a := -math.Pi - 0.2 + (math.Pi+0.4)*float64(i)/float64(n-1) + r(-0.15, 0.15)
		fronds = append(fronds, fr{a, r(16, 22) * s})
	}
	// The ones pointing up (behind) first, the drooping side ones last.
	slices.SortFunc(fronds, func(p, q fr) int {
		return cmp.Compare(math.Abs(p.a+math.Pi/2), math.Abs(q.a+math.Pi/2))
	})
	for i, f := range fronds {
		frond(c, tx, ty, f.a, f.L, treeFernRamp, false, 90+uint32(i))
	}
	c.outline(0.55)
	return trimmed(c, bx, by)
}

// genCycad is a cycad: a stout, scaly trunk and a stiff crown of
// palm-like fronds.
func genCycad(s float64, seed uint64) sprite {
	rng := rand.New(rand.NewPCG(seed, 37))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	s *= k
	c := markHD(newCanvas(int(math.Ceil(58*s)), int(math.Ceil(50*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k
	h, half := r(14, 19)*s, 3.8*s
	for y := int(by - h); y <= int(by); y++ {
		for x := int(bx - half); x < int(math.Ceil(bx+half)); x++ {
			u := (float64(x) + 0.5 - (bx - half)) / (2 * half)
			l := 0.8 - u*0.6
			if !c.hd() {
				if (x+y*2)%5 == 0 || (x-y*2)%5 == 0 { // diamond scales
					l -= 0.25
				}
				c.set(x, y, cycadTrunk.pick(l, x, y))
				continue
			}
			// Diamond scales, each a little shingle: dark seams, a lit upper
			// edge, and the trunk bulging a little at every row.
			if bulge := 1 + 0.07*math.Sin(float64(y)*0.9); math.Abs(u-0.5)*2 > bulge*0.94 {
				continue
			}
			a, b := ((x+y*2)%12+12)%12, ((x-y*2)%12+12)%12
			switch {
			case a == 0 || b == 0:
				l -= 0.3
			case a == 1 || b == 11:
				l += 0.14
			}
			l += (hashf(x, y, 97) - 0.5) * 0.1
			c.put(x, y, cycadTrunk.pick(l, x, y))
		}
	}
	tx, ty := bx, by-h
	n := 9 + rng.IntN(3)
	for i := range n {
		a := -math.Pi + 0.5 + (math.Pi-1.0)*float64(i)/float64(n-1) + r(-0.1, 0.1)
		frond(c, tx, ty+k, a, r(15, 19)*s, cycadRamp, true, 140+uint32(i))
	}
	c.outline(0.6)
	return trimmed(c, bx, by)
}

// genAraucaria is a monkey-puzzle tree: a tall bare trunk, branches in
// tiers near the top, the whole crown a flat-topped umbrella of ropy
// foliage.
func genAraucaria(s float64, seed uint64) sprite {
	rng := rand.New(rand.NewPCG(seed, 41))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	girth := 2.0 * s * s * k
	s *= k
	c := markHD(newCanvas(int(math.Ceil(60*s)), int(math.Ceil(76*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k
	tx, ty := bx+r(-1.5, 1.5)*s, by-r(50, 56)*s
	trunk := limb{bx, by, bx, (by + ty) / 2, tx, ty, girth, 1.0 * s}
	drawLimb(c, trunk)
	var crowns []crown
	tiers := 4
	for k := range tiers {
		y := ty + float64(k)*5*s
		span := (10 + float64(k)*4.5) * s
		for _, side := range []float64{-1, 1} {
			ex, ey := tx+side*span, y-r(1, 3)*s
			l := limb{tx, y + hdK(), tx + side*span*0.5, y + 1.5*s, ex, ey, 1.1 * s, 0.6 * hdK()}
			drawLimb(c, l)
			// Ropy foliage: a row of small clumps along the outer branch.
			for t := 0.35; t <= 1.0; t += 0.16 {
				x, yy := l.at(t)
				crowns = append(crowns, crown{[]blob{{x, yy - 1.5*s, (2.2 + rng.Float64()) * s}, {x, yy, 2.3 * s}}, x, yy, 4 * s, 3 * s})
			}
		}
	}
	crowns = append(crowns, roundCrown(tx, ty-2*s, 5.5*s, rng)) // the domed top
	slices.SortStableFunc(crowns, func(a, b crown) int { return cmp.Compare(a.cy, b.cy) })
	for i, cr := range crowns {
		leafCluster(c, cr.blobs, cr.cx, cr.cy, cr.rx, cr.ry, araucariaRamp, 200+uint32(i))
	}
	shadeUnderCrown(c)
	c.outline(0.7)
	return trimmed(c, bx, by)
}

// genRedwood is a giant: a great fluted red trunk, bare a long way up, under
// a tall narrow crown of dark needles. The biggest thing in the woods.
func genRedwood(s float64, seed uint64) sprite {
	rng := rand.New(rand.NewPCG(seed, 43))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	base := 4.2 * s * s * k
	s *= k
	c := markHD(newCanvas(int(math.Ceil(48*s)), int(math.Ceil(100*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k
	top := by - r(88, 94)*s
	tip := 0.8 * s
	for y := int(top); y <= int(by); y++ {
		f := (by - float64(y)) / (by - top) // 0 at the ground, 1 at the top
		half := lerpf(base, tip, math.Pow(f, 0.7))
		if f < 0.06 {
			half += (0.06 - f) * 40 * s // the buttressed foot
		}
		for x := int(bx - half); x < int(math.Ceil(bx+half)); x++ {
			u := (float64(x) + 0.5 - (bx - half)) / (2 * half)
			l := 0.85 - u*0.65
			if c.hd() { // fibrous bark in long ropy furrows
				c.put(x, y, barkAt(redwoodRamp, u*2-1, half, float64(y)*1.3, x, y, 0.85, 43))
				continue
			}
			if (x+int(f*7))%3 == 0 { // deep vertical furrows
				l -= 0.28
			}
			c.set(x, y, redwoodRamp.pick(l, x, y))
		}
	}
	// The crown: clumps up the top three fifths, narrowing to a spire.
	var crowns []crown
	for y := by - (by-top)*0.42; y > top+2*s; y -= 4.2 * s {
		f := (by - y) / (by - top)
		w := (1 - f) * 15 * s
		for _, side := range []float64{-1, 1} {
			x := bx + side*(w*r(0.4, 0.9))
			crowns = append(crowns, crown{[]blob{{x, y, (2.6 + (1-f)*3.2) * s}, {x - side*2*s, y + 1.5*s, (2.2 + (1-f)*2.5) * s}}, x, y, 8 * s, 5 * s})
		}
	}
	crowns = append(crowns, crown{[]blob{{bx, top + 2*s, 2.2 * s}}, bx, top, 3 * s, 3 * s})
	slices.SortStableFunc(crowns, func(a, b crown) int { return cmp.Compare(a.cy, b.cy) })
	for i, cr := range crowns {
		leafCluster(c, cr.blobs, cr.cx, cr.cy, cr.rx, cr.ry, redwoodLeaf, 260+uint32(i))
	}
	c.outline(0.75)
	return trimmed(c, bx, by)
}

// ancientTrees are the prehistoric trees of one size, three of each.
type ancientTrees struct {
	ferns, cycads, araucarias, ginkgos, redwoods []sprite
}

func genAncientTrees(s float64) ancientTrees {
	var t ancientTrees
	for v := uint64(1); v <= 3; v++ {
		t.ferns = append(t.ferns, genTreeFern(s, v))
		t.cycads = append(t.cycads, genCycad(s, v))
		t.araucarias = append(t.araucarias, genAraucaria(s, v))
		t.ginkgos = append(t.ginkgos, genBroadleafIn([]treeShape{shapeOak, shapeRound, shapeTall}[v-1], ginkgoRamp, 7, false, s, 70+v))
		t.redwoods = append(t.redwoods, genRedwood(s, v))
	}
	return t
}
