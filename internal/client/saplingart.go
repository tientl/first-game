package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The sapling of a felled tree. A tree that is cut down grows back in
// stages: stump, sapling, young tree, grown tree. The stump is genStumpHD,
// the young and grown trees are genTrees at treeScale[0] and [1]; this is
// the stage between: a whip of a tree a little taller than a person, with a
// slim bowed trunk on a flared foot, a few twigs and a loose handful of
// leaves the sky shows through. (The seedling of a planted seed, genSapling,
// is a far smaller thing.)
//
// It is drawn in HD image pixels like the stump, in the bark of trunkhd.go
// and the leaves of the broadleaf trees, lit from the top left, and anchored
// at the ground point under its trunk, so it stands where the tree stood.

// saplingLook is one variant of the tree sapling. Lengths are art pixels.
type saplingLook struct {
	h          float64 // from the ground to the tip of the top leaf, about
	lean, bend float64 // how far the top stands aside, and the bow half way up
	girth      float64 // half the trunk's width above its foot
	twigs      int
	leaves     int // on each twig
	leaf       int // which of leafRamps
	bark       *ramp
	seed       uint32
}

var saplingLooks = [...]saplingLook{
	{h: 35, lean: 2.2, bend: -1.5, girth: 1.25, twigs: 4, leaves: 5, leaf: 1, bark: &barkOak, seed: 11},
	{h: 39, lean: -2.6, bend: 1.3, girth: 1.4, twigs: 5, leaves: 5, leaf: 0, bark: &trunkRamp, seed: 23},
	{h: 32, lean: 0.8, bend: 1.7, girth: 1.15, twigs: 3, leaves: 6, leaf: 2, bark: &barkCypress, seed: 37},
}

// The canvas of a tree sapling and its ground point, in art pixels.
const (
	saplingW, saplingH   = 36, 46
	saplingAX, saplingAY = 18, 42
)

// treeSaplings is the saplings, drawn the first time one is seen.
var treeSaplings = lazy[[]sprite]{gen: genTreeSaplingsHD}

// treeSaplingAt is the sapling of a tree growing on (tx, ty).
func treeSaplingAt(tx, ty int) sprite {
	s := treeSaplings.get()
	return s[int(sim.HashTile(tx, ty)/8)%len(s)]
}

// genTreeSaplingsHD draws the saplings a felled tree regrows through: three
// variants, differing in lean, leaves and bark.
func genTreeSaplingsHD() []sprite {
	out := make([]sprite, len(saplingLooks))
	for v := range out {
		out[v] = mkUp(treeSaplingCanvas(v))
	}
	return out
}

// treeSaplingCanvas draws variant v and gives its ground point, in image
// pixels.
func treeSaplingCanvas(v int) (*canvas, float64, float64) {
	const k = hdK3
	lk := saplingLooks[v%len(saplingLooks)]
	sd := lk.seed
	rnd := func(i, j int, lo, hi float64) float64 { return lo + hashf(i, j, sd)*(hi-lo) }
	c := hdCanvas(saplingW, saplingH)
	bx, by := float64(saplingAX)*k, float64(saplingAY)*k
	bark, leaf := *lk.bark, leafRamps[lk.leaf]

	tk := trunkHD{
		bx: bx, by: by, tx: bx + lk.lean*k, ty: by - (lk.h-2.5)*k,
		w0: lk.girth * k, w1: 0.45 * k, bend: lk.bend * k,
		flare: 0.6, bark: barkFurrowed, rp: bark, seed: sd,
	}
	mid := func(f float64) (float64, float64) { // the trunk's middle, f of the way up
		y := tk.by - f*(tk.by-tk.ty)
		x, _ := tk.at(y)
		return x, y
	}

	// Twigs, the lower ones longer, by turns left and right, each curving up.
	var twigs []limb
	side := float64(1 - 2*int(sd%2))
	for i := range lk.twigs {
		f := 0.46 + 0.46*(float64(i)+rnd(i, 0, 0.3, 0.7))/float64(lk.twigs)
		x0, y0 := mid(f)
		ln := lerpf(8.5, 4, (f-0.46)/0.46) * rnd(i, 1, 0.85, 1.15) * k
		rise := ln * rnd(i, 2, 0.55, 0.9)
		twigs = append(twigs, limb{x0, y0, x0 + side*ln*0.6, y0 - rise*0.2, x0 + side*ln, y0 - rise, 0.62 * k, 0.3 * k})
		side = -side
	}

	// Leaves: along each twig and in a fan at the top of the trunk. Each has
	// a darker one behind it, drawn before the wood.
	type lf struct{ x, y, a, b, ang, lit float64 }
	var front, back []lf
	ccx, ccy := mid(0.78)
	n := 0
	add := func(x, y, ang float64) {
		n++
		a := rnd(n, 3, 1.25, 1.75) * k
		lx, ly := x+math.Cos(ang)*a*0.9, y+math.Sin(ang)*a*0.9
		lit := 0.6 - ((lx-ccx)*0.5+(ly-ccy)*0.6)/(12*k)*0.3 + rnd(n, 4, -0.06, 0.06)
		front = append(front, lf{lx, ly, a, a * 0.5, ang, lit})
		if hashf(n, 5, sd) < 0.7 {
			ba := ang + rnd(n, 6, 0.5, 1.1)*float64(1-2*(n%2))
			bl := a * 1.05
			back = append(back, lf{x + math.Cos(ba)*bl*0.9, y + math.Sin(ba)*bl*0.9, bl, bl * 0.5, ba, lit - 0.3})
		}
	}
	for i, t := range twigs {
		for j := range lk.leaves {
			u := 0.3 + 0.7*float64(j)/float64(lk.leaves-1)
			x, y := t.at(u)
			x2, y2 := t.at(min(1, u+0.05))
			x1, y1 := t.at(u - 0.05)
			dir := math.Atan2(y2-y1, x2-x1)
			if j == lk.leaves-1 {
				add(x, y, dir+rnd(i, 10+j, -0.25, 0.25)) // the one at the tip
				continue
			}
			add(x, y, dir+rnd(i, 10+j, 0.7, 1.2)*float64(1-2*((i+j)%2)))
		}
	}
	_, ty := mid(1)
	for j := range 5 {
		ang := -math.Pi/2 + (float64(j)-2)*0.62 + rnd(j, 20, -0.15, 0.15)
		fy := ty + math.Abs(float64(j)-2)*1.6*k // the outer ones start lower
		x, _ := tk.at(fy)
		add(x, fy, ang)
	}

	// Turned earth about the foot.
	c.ellipse(bx, by+0.5*k, 4.6*k, 1.25*k, func(x, y int, nx, ny float64) color.RGBA {
		return crumbRamp.pick(0.6-ny*0.35-nx*0.15+(hashf(x/2, y/2, sd+7)-0.5)*0.5, x, y)
	})
	for _, l := range back {
		leafHD(c, l.x, l.y, l.a, l.b, l.ang, leaf, l.lit)
	}
	tk.drawBody(c)
	for _, t := range twigs {
		drawLimbHD(c, t, bark, 0.85, sd+3)
	}
	for _, l := range front {
		leafHD(c, l.x, l.y, l.a, l.b, l.ang, leaf, l.lit)
	}
	c.outline(0.75)

	// A few blades of grass against the foot, over the outline so they stay
	// fine.
	for i := range 3 {
		gx := bx + (float64(i)-1)*3.1*k + rnd(i, 30, -0.8, 0.8)*k
		gy := by + (1.3-math.Abs(float64(i)-1)*0.7)*k
		for b := range 4 + int(hashf(i, 31, sd)*3) {
			x0 := gx + rnd(i*8+b, 32, -1.2, 1.2)*k
			h := rnd(i*8+b, 33, 1.6, 3.6) * k
			ln := rnd(i*8+b, 34, -0.5, 0.5)
			for j := 0; j < int(h); j++ {
				f := float64(j) / h
				px, py := int(math.Round(x0+ln*f*f*h*0.6)), int(gy)-j
				lt := 0.15 + f*0.7 + rnd(i*8+b, 35, -0.15, 0.15)
				c.put(px, py, grassRamp.pick(lt, px, py))
				if f < 0.35 {
					c.put(px+1, py, grassRamp.pick(lt-0.1, px+1, py))
				}
			}
		}
	}
	return c, bx, by
}
