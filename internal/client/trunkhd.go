package client

import (
	"cmp"
	"image/color"
	"math"
	"slices"
)

// Trunks at HD. A trunk tapers from a flared foot to where it goes into the
// crown, bows a little, and stands on a round base seen from above at a
// slant, so it meets the ground as a solid thing, not a cut. Roots run out
// from the foot over the ground; grass, snow, sand or mud lies over their
// ends. Each kind of tree has its own bark, lit from the top left like
// everything else, with a faint rim of light on the far edge and the foot
// darkened where it meets the ground.

// barkKind is the look of a trunk's bark.
type barkKind uint8

const (
	barkFurrowed barkKind = iota // oak-like: ridges split by long furrows, cracked across
	barkSmooth                   // ginkgo, fig: smooth and grey, mottled, with lenticels
	barkPlates                   // pines: tall scaly plates with dark seams
	barkScales                   // araucaria: flat scales in rings, where branches fell
	barkFibre                    // redwood, cypress: long fibrous strips, peeling
	barkRinged                   // palms: the rings of old leaf bases
	barkShaggy                   // tree ferns: a mat of fibres and the scars of fronds
	barkDead                     // a snag: bleached, split along its length
)

// The barks' colours, dark to light.
var (
	barkOak       = mkRamp(0x22160f, 0x321f15, 0x462d1d, 0x5a3c26, 0x704c30, 0x885f3c, 0xa1754c)
	barkGrey      = mkRamp(0x2b2926, 0x403c37, 0x58534b, 0x716b60, 0x8b8477, 0xa59e90, 0xbfb8a9)
	barkPine      = mkRamp(0x29150f, 0x3f2117, 0x58301f, 0x724128, 0x8d5532, 0xa86b40, 0xc08452)
	barkAraucaria = mkRamp(0x26201c, 0x39302a, 0x4e433a, 0x65574b, 0x7d6d5e, 0x968572, 0xae9c88)
	barkRedwood   = mkRamp(0x2a110b, 0x441a10, 0x622617, 0x80351e, 0x9d4828, 0xb75f36, 0xcd7a4a)
	barkPalm      = mkRamp(0x2b2015, 0x41311f, 0x594429, 0x725935, 0x8b6f44, 0xa48756, 0xbca06c)
	barkFern      = mkRamp(0x170f0a, 0x25180f, 0x352316, 0x47301e, 0x5a3f28, 0x6f5034)
	barkCypress   = mkRamp(0x2a211b, 0x3d3028, 0x534337, 0x6b5747, 0x846d59, 0x9c846d, 0xb29b83)
	barkSnag      = mkRamp(0x38342f, 0x524d45, 0x6c665b, 0x878074, 0xa29b8d, 0xbab4a6, 0xd0cabd)
	crumbRamp     = mkRamp(0x1d150e, 0x2c2016, 0x3d2d1f, 0x51402c)
	mudRamp       = mkRamp(0x1a1810, 0x272418, 0x353122, 0x45402e)
)

// barkOf is the bark each colour of a trunk belongs to: what tells wood
// from leaves (isTrunk), and what a trunk's pixel is darkened with.
var barkOf = func() map[color.RGBA]ramp {
	m := map[color.RGBA]ramp{}
	for _, r := range []ramp{trunkRamp, barkOak, barkGrey, barkPine, barkAraucaria, barkRedwood, barkPalm, barkFern, barkCypress, barkSnag} {
		for _, c := range r {
			if _, ok := m[c]; !ok {
				m[c] = r
			}
		}
	}
	return m
}()

// barkPick is rp at l, quantised with a grain of noise rather than a
// regular dither, which would draw a cross-hatch on a big round trunk.
func barkPick(rp ramp, l float64, x, y int, seed uint32) color.RGBA {
	return rp.at(int(math.Floor(clamp01(l)*float64(len(rp)-1) + 0.5 + (hashf(x, y, seed)-0.5)*0.55)))
}

// trunkHD is a trunk to draw, in the image pixels of an HD canvas.
type trunkHD struct {
	bx, by float64 // the ground point: the middle of the foot
	tx, ty float64 // where it ends, in the crown
	w0, w1 float64 // half its width above the flare, and at the top
	bend   float64 // how far it bows aside half way up
	lean   float64 // >1: upright at the foot, leaning more higher up (palms); 0 is 1
	flare  float64 // how much wider the foot spreads, as a share of w0
	flutes int     // deep clefts down the flare (bald cypress), or 0
	bark   barkKind
	rp     ramp
	seed   uint32
}

// trunkFrom makes the trunk of limb l, which runs up from the ground.
func trunkFrom(l limb, bark barkKind, rp ramp, flare float64, seed uint32) trunkHD {
	mid := 0.25*l.x0 + 0.5*l.cx + 0.25*l.x1
	return trunkHD{bx: l.x0, by: l.y0, tx: l.x1, ty: l.y1, w0: l.w0, w1: l.w1, bend: mid - (l.x0+l.x1)/2, flare: flare, bark: bark, rp: rp, seed: seed}
}

// at is the trunk's middle and half width at height y.
func (t trunkHD) at(y float64) (cx, w float64) {
	f := clamp01((t.by - y) / (t.by - t.ty))
	p := t.lean
	if p == 0 {
		p = 1
	}
	cx = t.bx + (t.tx-t.bx)*math.Pow(f, p) + t.bend*4*f*(1-f)
	w = lerpf(t.w0, t.w1, math.Pow(f, 0.8))
	w *= 1 + t.flare*math.Exp(-math.Max(0, t.by-y)/(t.w0*1.15))
	w *= 1 + 0.05*(vnoise(y*0.03, 3, t.seed)-0.5) // never quite even
	return cx, w
}

// footW is the half width of the foot, where it meets the ground.
func (t trunkHD) footW() float64 { _, w := t.at(t.by); return w }

// draw paints the trunk, its roots and the dark where it meets the ground.
// roots is how many run out over the ground, reach how far past the foot.
func (t trunkHD) draw(c *canvas, roots int, reach float64) {
	rs := t.rootLimbs(roots, reach)
	for _, r := range rs {
		if r.y1 < t.by { // behind the trunk: first
			t.drawRoot(c, r)
		}
	}
	t.drawBody(c)
	for _, r := range rs {
		if r.y1 >= t.by {
			t.drawRoot(c, r)
		}
	}
}

// drawBody paints the trunk itself.
func (t trunkHD) drawBody(c *canvas) {
	k := hdK()
	wb := t.footW()
	ry := wb * 0.3 // the foot is round: its front edge comes down towards us
	for y := int(math.Floor(t.ty)); y <= int(math.Ceil(t.by+ry)); y++ {
		yy := float64(y) + 0.5
		cx, w := t.at(math.Min(yy, t.by))
		for x := int(math.Floor(cx - w)); x <= int(math.Ceil(cx+w)); x++ {
			u := (float64(x) + 0.5 - cx) / w
			if u <= -1 || u >= 1 {
				continue
			}
			if yy > t.by && yy > t.by+ry*math.Sqrt(1-u*u) {
				continue
			}
			sx := math.Asin(u) * w // across the bark, unrolled
			// A round trunk lit from the top left; a little light comes
			// round the far edge.
			l := 0.42 + 0.55*(-0.52*u+0.52*math.Sqrt(1-u*u))
			if u > 0.84 {
				l += 0.6 * (u - 0.84)
			}
			l += t.barkL(sx, yy, u, w, x, y)
			if d := t.by - yy; d < 4*k { // dark where it meets the ground
				l -= 0.24 * (1 - math.Max(0, d)/(4*k))
			}
			c.put(x, y, barkPick(t.rp, l, x, y, t.seed+9))
		}
	}
}

// barkL is how much lighter or darker the bark is at (sx across it, unrolled,
// and y), u across the trunk from -1 to 1 where it is w half wide.
func (t trunkHD) barkL(sx, y, u, w float64, x, yi int) float64 {
	k := hdK()
	sd := t.seed
	flute := 0.0
	if t.flutes > 0 { // clefts down the flared foot
		if d := t.by - y; d < t.w0*2.6 {
			g := math.Sin(math.Asin(u)*float64(t.flutes) + float64(sd%7))
			depth := 1 - d/(t.w0*2.6)
			if math.Abs(g) < 0.22*depth {
				flute = -0.34 * depth
			}
		}
	}
	switch t.bark {
	case barkFurrowed:
		f := vnoise(sx*0.12+float64(sd%17), y*0.021, sd) - 0.5
		switch g := math.Abs(f); {
		case g < 0.04:
			return flute - 0.36 // a furrow
		case g < 0.075:
			return flute - 0.14
		case f > 0.075 && f < 0.13:
			return flute + 0.1 // the edge of a ridge in the light
		}
		cell := int(math.Floor(y / (4.5 * k)))
		if hashf(int(math.Floor(sx/(3*k))), cell, sd+1) < 0.16 && math.Mod(y, 4.5*k) < 1.2 {
			return flute - 0.22 // a crack across a ridge
		}
		return flute + (hashf(int(sx), int(y/3), sd+2)-0.5)*0.08
	case barkSmooth:
		l := (vnoise(sx*0.05, y*0.035, sd)-0.5)*0.16 + (vnoise(sx*0.02, y*0.2, sd+1)-0.5)*0.06
		if yi%int(2*k) == 0 && hashf(int(math.Floor(sx/(1.5*k))), yi/int(2*k), sd+2) < 0.1 {
			l -= 0.24 // lenticels: short dashes across
		}
		return flute + l
	case barkPlates, barkScales:
		pw, ph := 3.6*k, 6.5*k // tall plates
		if t.bark == barkScales {
			pw, ph = 6*k, 1.9*k // flat scales, overlapping like shingles
		}
		d1, d2, id := barkCells(sx, y, pw, ph, sd)
		seam, deep, vary := 0.13, -0.34, 0.24
		if t.bark == barkScales {
			seam, deep, vary = 0.08, -0.24, 0.14
		}
		if d2-d1 < seam {
			return flute + deep // the seam between two plates
		}
		l := (hashf(id, 0, sd+1)-0.5)*vary + (0.35-d1)*0.3 + (hashf(x, yi/2, sd+2)-0.5)*0.06
		if t.bark == barkScales && math.Mod(y+hashf(int(sx/(6*k)), 0, sd+3)*k, 11*k) < 1.2*k {
			l -= 0.25 // a ring where a whorl of branches fell
		}
		return flute + l
	case barkFibre:
		f := vnoise(sx*0.17, y*0.006, sd) - 0.5
		switch g := math.Abs(f); {
		case g < 0.03:
			return flute - 0.32
		case g < 0.06:
			return flute - 0.12
		}
		l := (hashf(int(sx/2), int(y/7), sd+1) - 0.5) * 0.14
		if vnoise(sx*0.07, y*0.014, sd+5) > 0.68 {
			l += 0.13 // a strip peeling, paler
		}
		return flute + l
	case barkRinged:
		p := 2.3 * k
		ph := (y + u*u*1.2*k) / p
		fr := ph - math.Floor(ph)
		switch {
		case fr < 0.17:
			return -0.3 // the groove below a ring
		case fr < 0.32:
			return 0.13 // its lip in the light
		}
		return (hashf(int(sx/2), int(ph), sd)-0.5)*0.14 - (fr-0.32)*0.18
	case barkShaggy:
		col := int(math.Floor(sx / 2))
		off := hashf(col, 0, sd) * 9
		l := (hashf(col, int((y+off)/5), sd+1) - 0.5) * 0.36
		// The scars of fallen fronds, in a spiral.
		rowH := 7 * k
		row := math.Floor(y / rowH)
		ox := math.Mod(sx+row*2.4*k+100*rowH, 5*k) - 2.5*k
		oy := math.Mod(y, rowH) - rowH*0.5
		if ox*ox/(1.6*k*1.6*k)+oy*oy/(1.1*k*1.1*k) < 1 {
			l -= 0.3
		}
		return l
	case barkDead:
		f := vnoise(sx*0.1, y*0.011, sd) - 0.5
		if math.Abs(f) < 0.025 {
			return -0.4 // a split
		}
		return (vnoise(sx*0.45, y*0.02, sd+1)-0.5)*0.12 + (hashf(x, yi/3, sd+2)-0.5)*0.05
	}
	return flute
}

// barkCells is cell noise for bark in plates pw x ph: how far (x, y) is from
// the middle of the plate it is on and from the next one's (in plate
// widths), and which plate it is.
func barkCells(x, y, pw, ph float64, seed uint32) (d1, d2 float64, id int) {
	ci, cj := int(math.Floor(x/pw)), int(math.Floor(y/ph))
	d1, d2 = 9, 9
	for j := cj - 1; j <= cj+1; j++ {
		for i := ci - 1; i <= ci+1; i++ {
			px := (float64(i) + 0.15 + 0.7*hashf(i, j, seed+11)) * pw
			py := (float64(j) + 0.15 + 0.7*hashf(i, j, seed+12)) * ph
			dx, dy := (x-px)/pw, (y-py)/ph
			d := math.Sqrt(dx*dx + dy*dy)
			switch {
			case d < d1:
				d1, d2, id = d, d1, i*7919+j
			case d < d2:
				d2 = d
			}
		}
	}
	return d1, d2, id
}

// rootLimbs are the roots of the trunk: n of them run out over the ground
// round the front and sides of the foot, reach past it (give or take),
// arching out of the flare and sinking at their tips.
func (t trunkHD) rootLimbs(n int, reach float64) []limb {
	if n <= 0 {
		return nil
	}
	k := hdK()
	wb := t.footW()
	var rs []limb
	for i := range n {
		a := -0.12*math.Pi + 1.24*math.Pi*(float64(i)+0.5)/float64(n) + (hashf(i, 0, t.seed+20)-0.5)*0.35
		ca, sa := math.Cos(a), math.Sin(a)
		L := reach * (0.65 + 0.55*hashf(i, 1, t.seed+20))
		x0, y0 := t.bx+ca*wb*0.45, t.by+sa*wb*0.12-wb*0.5
		x1, y1 := t.bx+ca*(wb+L), t.by+sa*(wb+L)*0.3+k
		cx, cy := t.bx+ca*(wb+L*0.25), t.by+sa*(wb+L*0.25)*0.3-wb*0.18
		rs = append(rs, limb{x0, y0, cx, cy, x1, y1, wb * (0.3 + 0.08*hashf(i, 2, t.seed+20)) * (0.55 + 0.45*math.Abs(ca)), 0.7 * k}) // narrower end-on
	}
	slices.SortStableFunc(rs, func(p, q limb) int { return cmp.Compare(p.y1, q.y1) })
	return rs
}

// drawRoot paints a root lying on the ground: lit from above, its grain
// along it, darker where it meets the ground.
func (t trunkHD) drawRoot(c *canvas, l limb) {
	length := math.Hypot(l.x1-l.x0, l.y1-l.y0)
	n := int(length*2) + 2
	for i := 0; i <= n; i++ {
		f := float64(i) / float64(n)
		x, y := l.at(f)
		w := lerpf(l.w0, l.w1, math.Pow(f, 0.7))
		for py := int(math.Floor(y - w)); py <= int(math.Ceil(y+w)); py++ {
			for px := int(math.Floor(x - w)); px <= int(math.Ceil(x+w)); px++ {
				dx, dy := float64(px)+0.5-x, float64(py)+0.5-y
				if dx*dx+dy*dy > w*w+0.25 {
					continue
				}
				nx, ny := dx/math.Max(w, 0.5), dy/math.Max(w, 0.5)
				lt := 0.5 - ny*0.34 - nx*0.14 - f*0.18 + (vnoise(f*length*0.12, nx*2, t.seed+31)-0.5)*0.2
				if ny > 0.45 {
					lt -= 0.18 // the underside, in its own shadow
				}
				c.put(px, py, barkPick(t.rp, lt, px, py, t.seed+33))
			}
		}
	}
}

// groundKind is what lies about the foot of a tree.
type groundKind uint8

const (
	groundGrass groundKind = iota
	groundSnow
	groundSand
	groundMud
)

// dressFoot lays what is on the ground over the foot of trunk t and the
// ends of its roots, reach past it: tufts of grass and crumbs of soil, or
// drifts of snow, sand, or mud and moss. It goes on after the outline, so
// the blades stay fine.
func dressFoot(c *canvas, t trunkHD, reach float64, g groundKind) {
	k := hdK()
	wb := t.footW()
	sd := t.seed + 40
	spot := func(i int) (float64, float64) {
		a := 0.04*math.Pi + 0.92*math.Pi*hashf(i, 0, sd) // round the front
		d := wb*0.7 + reach*(0.2+0.9*hashf(i, 1, sd))
		return t.bx + math.Cos(a)*d, t.by + math.Sin(a)*d*0.3 + k
	}
	switch g {
	case groundGrass, groundMud:
		blades := grassRamp
		if g == groundMud {
			blades = mossRamp2
			for i := range 4 { // puddles of mud against the foot
				x, y := spot(i + 20)
				c.ellipse(x, y, (2.5+hashf(i, 2, sd)*3)*k, (0.8+hashf(i, 3, sd)*0.6)*k, func(px, py int, nx, ny float64) color.RGBA {
					return mudRamp.pick(0.55-ny*0.3+(hashf(px, py, sd)-0.5)*0.3, px, py)
				})
			}
		} else {
			for i := range 7 { // crumbs of soil about the roots
				x, y := spot(i + 30)
				for j := range 3 {
					px, py := int(x)+int((hashf(i, j, sd+1)-0.5)*4*k), int(y)-int(hashf(i, j, sd+2)*k)
					c.put(px, py, crumbRamp.at(1+j%3))
					c.put(px+1, py, crumbRamp.at(j%3))
				}
			}
		}
		for i := range 5 + int(t.seed%3) { // tufts of grass
			x, y := spot(i)
			nb := 5 + int(hashf(i, 4, sd)*5)
			for b := range nb {
				bx := x + (hashf(i, b, sd+3)-0.5)*3*k
				h := (2 + hashf(i, b, sd+4)*3.5) * k
				lean := (hashf(i, b, sd+5) - 0.5) * 0.9
				for j := 0; j < int(h); j++ {
					f := float64(j) / h
					px := int(math.Round(bx + lean*f*f*h*0.6))
					py := int(y) - j
					lt := 0.15 + f*0.75 + (hashf(i, b, sd+6)-0.5)*0.3
					c.put(px, py, blades.pick(lt, px, py))
					if f < 0.35 {
						c.put(px+1, py, blades.pick(lt-0.1, px+1, py))
					}
				}
			}
		}
	case groundSnow, groundSand:
		rp := snowRamp
		if g == groundSand {
			rp = sandRamp
		}
		// A drift against the foot, and lumps over the roots' ends.
		c.ellipse(t.bx-wb*0.15, t.by+0.6*k, wb*1.05, 1.9*k, func(px, py int, nx, ny float64) color.RGBA {
			return rp.pick(lum(nx, ny)*0.75+0.25, px, py)
		})
		for i := range 4 {
			x, y := spot(i)
			c.ellipse(x, y, (2+hashf(i, 2, sd)*2.5)*k, (0.9+hashf(i, 3, sd)*0.7)*k, func(px, py int, nx, ny float64) color.RGBA {
				return rp.pick(lum(nx, ny)*0.75+0.25, px, py)
			})
		}
		if g == groundSand {
			for i := range 5 { // pebbles
				x, y := spot(i + 10)
				c.ellipse(x, y, 0.9*k, 0.6*k, func(px, py int, nx, ny float64) color.RGBA {
					return stoneRamp.pick(lum(nx, ny)*0.7+0.2, px, py)
				})
			}
		}
	}
}

// stubs are a few short dead branches broken off a trunk below its crown,
// n of them between heights f0 and f1 (0 at the foot, 1 at the top).
func (t trunkHD) stubs(c *canvas, n int, f0, f1 float64) {
	k := hdK()
	for i := range n {
		f := f0 + (f1-f0)*(float64(i)+0.5)/float64(n)
		y := t.by - f*(t.by-t.ty)
		cx, w := t.at(y)
		side := float64(1 - 2*(i%2))
		if hashf(i, 0, t.seed+50) < 0.3 {
			side = -side
		}
		// A short dead twig drooping from a swollen collar, broken off.
		ln := (1.2 + hashf(i, 1, t.seed+50)*1.8) * k
		x0 := cx + side*w*0.8
		x1, y1 := x0+side*ln, y+ln*(0.35+0.3*hashf(i, 2, t.seed+50))
		r := math.Max(0.7*k, w*0.14)
		knotHD(c, cx+side*w*0.55, y, math.Max(1.4, r*0.9), t.rp)
		drawLimbHD(c, limb{x0, y, (x0+x1)/2 + side*r*0.3, (y + y1) / 2, x1, y1, r, r * 0.4}, t.rp, 0.62, t.seed+51)
	}
}

// mossSide grows moss up the shaded side of a trunk, thickest at the foot.
// It runs last: the moss is not wood, to isTrunk.
func (t trunkHD) mossSide(c *canvas, amount float64) {
	for y := int(t.ty); y <= int(t.by)+2; y++ {
		cx, w := t.at(math.Min(float64(y), t.by))
		up := clamp01((t.by - float64(y)) / (t.by - t.ty))
		for x := int(cx - w); x <= int(cx+w); x++ {
			p := c.at(x, y)
			if !isTrunk(p) {
				continue
			}
			u := (float64(x) + 0.5 - cx) / w
			m := vnoise(float64(x)*0.07, float64(y)*0.05, t.seed+60) + u*0.25 + (1-up)*(1-up)*0.3*amount - (1-amount)*0.3
			if m > 0.86+(vnoise(float64(x)*0.4, float64(y)*0.4, t.seed+61)-0.5)*0.1 {
				lt := 0.55 - u*0.25 + (vnoise(float64(x)*0.3, float64(y)*0.25, t.seed+62)-0.5)*0.4
				c.put(x, y, mossRamp.pick(lt, x, y))
			}
		}
	}
}
