package client

import (
	"cmp"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The swamp: olive mud pocked with puddles, murky green ponds with muddy
// banks, bald cypresses hung with moss, dead snags, and mushrooms growing
// on rotting logs.

var (
	swampMud   = mkRamp(0x2c2214, 0x3b2e1c, 0x4d3d25, 0x5f4c2f)
	pondRamp   = mkRamp(0x1d3329, 0x24392e, 0x2d4434, 0x37503a, 0x435d40)
	algaeRamp  = mkRamp(0x3f6a2a, 0x5a8a34, 0x78a83e, 0x98c450)
	puddleRamp = mkRamp(0x263f33, 0x2f4a3b, 0x3c5a46, 0x5b7a62)

	cypressLeaf = mkRamp(0x14291f, 0x1b3a28, 0x244e31, 0x2f633a, 0x427a45, 0x5c9252)
	cypressBark = mkRamp(0x3a2c22, 0x54402f, 0x6e5640, 0x8a6f54, 0xa58a6c)
	mossRamp2   = mkRamp(0x4c5a3e, 0x66745a, 0x84906f, 0xa0aa88)
	snagRamp    = mkRamp(0x403b35, 0x5d574e, 0x7c766a, 0x9b9487, 0xbcb5a6)
	fungusRamp  = mkRamp(0x8a4a1c, 0xc06a26, 0xe08f34, 0xf2b85c)
	capRamp     = mkRamp(0x7a1a14, 0xa8281e, 0xcf3c2c, 0xe86a52)
)

// --- ponds -------------------------------------------------------------------

// swampPondMax is the most tiles a body of water can have and still be a
// pond; anything bigger is a lake or the sea.
const swampPondMax = 3000

// swampPonds marks, per tile, where the water is a swamp pond: a small body
// of water whose shore is mostly marsh, and the land right around one. The
// ground colours that water murky green, with a muddy bank instead of surf.
func swampPonds(w, h int, tiles []sim.Tile) []float32 {
	at := func(x, y int) sim.Tile {
		if x < 0 || y < 0 || x >= w || y >= h {
			return sim.TileWater
		}
		return tiles[y*w+x]
	}
	pond := make([]float32, w*h)
	seen := make([]bool, w*h)
	var body []int
	for start, t := range tiles {
		if t != sim.TileWater || seen[start] {
			continue
		}
		body = append(body[:0], start)
		seen[start] = true
		shore, marsh, edge := 0, 0, false
		for k := 0; k < len(body); k++ {
			x, y := body[k]%w, body[k]/w
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					edge = true // it runs off the map: the sea
					continue
				}
				j := ny*w + nx
				switch nt := tiles[j]; {
				case nt == sim.TileWater:
					if !seen[j] && len(body) <= swampPondMax {
						seen[j] = true
						body = append(body, j)
					}
				case nt == sim.TileMarsh:
					shore++
					marsh++
				default:
					shore++
				}
			}
		}
		if edge || len(body) > swampPondMax || marsh*2 < shore {
			// Too big to be a pond: mark the rest of it seen, so it is not
			// started again from another tile.
			for k := 0; k < len(body); k++ {
				x, y := body[k]%w, body[k]/w
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if j := ny*w + nx; nx >= 0 && ny >= 0 && nx < w && ny < h && !seen[j] && tiles[j] == sim.TileWater {
						seen[j] = true
						body = append(body, j)
					}
				}
			}
			continue
		}
		for _, i := range body {
			pond[i] = 1
			x, y := i%w, i/w
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if nx, ny := x+dx, y+dy; at(nx, ny) != sim.TileWater {
						pond[ny*w+nx] = 1 // the bank
					}
				}
			}
		}
	}
	return pond
}

// pondAt is how much (u, v), in tiles, is in or beside a swamp pond, from 0
// to 1, eased between tile centres like the coast.
func (t *terrain) pondAt(u, v float64) float64 {
	if t.pond == nil {
		return 0
	}
	at := func(x, y int) float64 {
		if x < 0 || y < 0 || x >= t.w || y >= t.h {
			return 0
		}
		return float64(t.pond[y*t.w+x])
	}
	x, y := u-0.5, v-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	fx, fy := x-x0, y-y0
	return lerpf(lerpf(at(ix, iy), at(ix+1, iy), fx), lerpf(at(ix, iy+1), at(ix+1, iy+1), fx), fy)
}

// pondTile reports whether tile (tx, ty) is the water of a swamp pond.
func (t *terrain) pondTile(tx, ty int) bool {
	return t.pond != nil && t.tile(tx, ty) == sim.TileWater && tx >= 0 && ty >= 0 && tx < t.w && ty < t.h && t.pond[ty*t.w+tx] > 0
}

// --- trees ---------------------------------------------------------------------

// drawBark paints l as bark of ramp rp, lit from the left, with streaks
// running along it.
func drawBark(c *canvas, l limb, rp ramp) {
	if c.hd() {
		drawLimbHD(c, l, rp, 0.85, 160)
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
				if hashf(px, py/4, 160) < 0.25 {
					k-- // a furrow in the bark
				}
				c.put(px, py, rp.at(k))
			}
		}
	}
}

// genCypress draws a bald cypress at scale s: a tall trunk flaring into
// buttresses at its foot, layers of cloudy needle tufts on short limbs, and
// beards of Spanish moss hanging from them. Cypresses tower over the other
// trees.
func genCypress(variant int, s float64) sprite {
	rng := rand.New(rand.NewPCG(uint64(variant)*41+3, 161))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	hk := hdK()
	s *= 0.85 * hk
	c := markHD(newCanvas(int(math.Ceil(66*s)), int(math.Ceil(86*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*hk
	girth := 2.9 * s
	top := by - r(54, 62)*s
	tx := bx + r(-2.5, 2.5)*s
	trunk := limb{bx, by - 8*s, bx + (tx-bx)*0.3, (by - 8*s + top) / 2, tx, top, girth, girth * 0.5}

	// Tufts: a wide one on top, the rest on short limbs up the trunk,
	// alternating sides.
	limbs := []limb{trunk}
	crowns := []crown{flatCrown(tx, top-r(1, 3)*s, r(10, 13)*s, 6.5*s, rng)}
	n := 4 + variant%2
	side := float64(1 - 2*rng.IntN(2))
	for i := range n {
		f := 0.42 + 0.48*float64(i)/float64(n-1) + r(-0.03, 0.03)
		x0, y0 := trunk.at(f)
		x1, y1 := x0+side*r(7, 12)*s, y0-r(2, 5)*s
		limbs = append(limbs, limb{x0, y0, (x0 + x1) / 2, (y0+y1)/2 + s, x1, y1, max(0.8*hk, girth*0.35), 0.6 * hk})
		crowns = append(crowns, flatCrown(x1, y1-s, r(6, 9)*s, 5*s, rng))
		side = -side
	}
	for _, l := range limbs {
		drawBark(c, l, cypressBark)
	}

	// The flared foot: buttresses spreading out to the ground, with dark
	// clefts between them.
	flareH := r(12, 16) * s
	for y := int(by - flareH); y <= int(by); y++ {
		f := (float64(y) + 0.5 - (by - flareH)) / flareH
		hw := girth * (1 + 1.6*f*f*f)
		for x := int(math.Floor(bx - hw - 1)); x <= int(math.Ceil(bx+hw+1)); x++ {
			dx := float64(x) + 0.5 - bx
			wave := dx/girth*2.2 + float64(variant)
			lobe := hw * (0.86 + 0.14*math.Cos(wave))
			if math.Abs(dx) > lobe {
				continue
			}
			k := cypressBark.idx(0.85-(dx/lobe+1)/2*0.7, x, y)
			if f > 0.3 && math.Abs(math.Sin(wave)) < 0.2 {
				k -= 2
			}
			if hashf(x, y/4, 160) < 0.2 {
				k--
			}
			if c.hd() {
				col := barkAt(cypressBark, dx/lobe, lobe, float64(y)*1.2, x, y, 0.85, 160)
				if f > 0.3 && math.Abs(math.Sin(wave)) < 0.2 {
					col = cypressBark.at(0)
				}
				c.put(x, y, col)
				continue
			}
			c.put(x, y, cypressBark.at(k))
		}
	}

	// The needles go on their own layer, so the moss can hang from them.
	leaves := newCanvas(c.w, c.h)
	leaves.dens = c.dens
	slices.SortStableFunc(crowns, func(a, b crown) int { return cmp.Compare(a.cy, b.cy) })
	for i, cr := range crowns {
		leafCluster(leaves, cr.blobs, cr.cx, cr.cy, cr.rx, cr.ry, cypressLeaf, 170+uint32(variant)*13+uint32(i)*7)
	}
	c.over(leaves, 0, 0)
	c.outline(0.75)

	// Beards of moss, a few columns wide, longest in the middle, wispy at
	// the ends. They go on after the outline, so they stay light.
	under := make([]int, c.w)
	var cols []int
	for x := range c.w {
		under[x] = -1
		for y := c.h - 1; y >= 0; y-- {
			if leaves.at(x, y).A > 0 {
				under[x] = y
				cols = append(cols, x)
				break
			}
		}
	}
	for range len(cols)/6 + 2 {
		x := cols[rng.IntN(len(cols))]
		half := (1 + rng.IntN(2)) * int(hk)
		L := r(5, 13) * s
		for dx := -half; dx <= half; dx++ {
			px := x + dx
			if px < 0 || px >= c.w || under[px] < 0 {
				continue
			}
			ln := int(L * (1 - math.Abs(float64(dx))/float64(half+1)) * r(0.6, 1))
			for i := range ln {
				if i > ln*2/3 && hashf(px, i, 172) < 0.45 || c.hd() && hashf(px, 0, 173) < 0.3 {
					continue // wispy at the end; and at HD, air between the strands
				}
				py := under[px] + 1 + i
				l := 0.85 - float64(i)/float64(max(ln, 1))*0.45 + (hashf(px, py, 171)-0.5)*0.3
				c.put(px, py, mossRamp2.pick(l, px, py))
			}
		}
	}
	return trimmed(c, bx, by)
}

// genSnag draws a dead tree at scale s: a bare, pale trunk snapped off at
// the top, a few broken limbs, shelves of orange fungus and moss at its foot.
func genSnag(variant int, s float64) sprite {
	rng := rand.New(rand.NewPCG(uint64(variant)*53+11, 181))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	hk := hdK()
	s *= 0.8 * hk
	c := markHD(newCanvas(int(math.Ceil(40*s)), int(math.Ceil(64*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*hk
	girth := 2.4 * s
	top := by - r(38, 46)*s
	tx := bx + r(-2, 2)*s
	trunk := limb{bx, by, bx + (tx-bx)*0.4, (by + top) / 2, tx, top, girth, girth * 0.75}
	limbs := []limb{trunk}
	for _, side := range []float64{-1, 1} {
		limbs = append(limbs, limb{bx, by - 4*s, bx + side*girth, by - s, bx + side*r(3, 5)*s, by, girth * 0.7, 0.6 * hk})
	}
	side := float64(1 - 2*rng.IntN(2))
	for range 2 + variant%2 {
		x0, y0 := trunk.at(r(0.45, 0.85))
		x1, y1 := x0+side*r(4, 8)*s, y0-r(3, 8)*s
		limbs = append(limbs, limb{x0, y0, (x0 + x1) / 2, (y0 + y1) / 2, x1, y1, max(0.6*hk, girth*0.45), 0.5 * hk})
		side = -side
	}
	for _, l := range limbs {
		drawBark(c, l, snagRamp)
	}
	// The snapped top: a few jagged teeth of wood.
	for i := -2 * int(hk); i <= 2*int(hk); i++ {
		x := int(math.Floor(tx)) + i
		for k := range 1 + int(hashf(i/int(hk), variant, 182)*3*s*(1-0.3*float64((i+60)%int(hk)))) {
			c.put(x, int(math.Floor(top))-1-k, snagRamp.at(2+i%2))
		}
	}
	// Shelves of bracket fungus up one side.
	for i := range 2 + variant%2 {
		x, y := trunk.at(0.3 + 0.12*float64(i))
		fx := x + girth*0.6
		if i%2 == 1 {
			fx = x - girth*0.6 - 3*s
		}
		for dx := 0.0; dx < 3.5*s; dx++ {
			px, py := int(math.Floor(fx+dx)), int(math.Floor(y))
			if c.hd() { // a shelf: pale on top, banded and gilled below
				edge := int(1.5 * (1 - math.Abs(dx/(3.5*s)*2-1)))
				for j := -edge; j < 2; j++ {
					c.put(px, py+j, fungusRamp.at(3-(j+edge)/2))
				}
				c.put(px, py+2, fungusRamp.at(1+int(dx)%2))
				c.put(px, py+3, fungusRamp.at(int(dx)%2))
				continue
			}
			c.put(px, py, fungusRamp.at(3))
			c.put(px, py+1, fungusRamp.at(1+int(dx)%2))
		}
	}
	// Moss creeping up from the ground.
	for x := 0; x < c.w; x++ {
		for y := int(by) - int(6*s); y <= int(by); y++ {
			if p := c.at(x, y); p.A > 0 && hashf(x, y, 183) < 0.45-float64(int(by)-y)*0.06 {
				c.put(x, y, cypressLeaf.pick(0.6, x, y))
			}
		}
	}
	c.outline(0.75)
	return trimmed(c, bx, by)
}

// swampSets holds the swamp's trees per size, drawn the first time they are
// wanted (see lazyart.go).
var swampSets = [2]lazy[swampSet]{{gen: func() swampSet { return genSwampSet(0) }}, {gen: func() swampSet { return genSwampSet(1) }}}

type swampSet struct{ cypress, snags []sprite }

func genSwampSet(size int) swampSet {
	t := swampSet{make([]sprite, 4), make([]sprite, 2)}
	inParallel(6, func(v int) {
		if v < 4 {
			t.cypress[v] = genCypress(v, treeScale[size])
		} else {
			t.snags[v-4] = genSnag(v-4, treeScale[size])
		}
	})
	return t
}

// swampTree picks the look of a tree standing in the swamp from h, its
// tile's hash: mostly cypresses, one in four long dead.
func swampTree(size int, h uint32) sprite {
	t := swampSets[size].get()
	if h%4 == 0 {
		return t.snags[int(h/8)%len(t.snags)]
	}
	return t.cypress[int(h/8)%len(t.cypress)]
}

// --- mushroom logs ---------------------------------------------------------------

var mushroomLogs [2]sprite // bare, then with mushrooms

// onLog reports whether mushroom e grows on a rotting log: most do in the swamp.
func (g *Game) onLog(e protocol.EntityView) bool {
	return e.Kind == sim.EntityMushroom && g.tileAt(int(e.TX), int(e.TY)) == sim.TileMarsh &&
		sim.HashTile(int(e.TX), int(e.TY))%3 != 0
}

// mushroomLog is a mossy fallen log, with red toadstools on it when full.
func mushroomLog(full bool) sprite {
	if mushroomLogs[0].img == nil {
		for i := range mushroomLogs {
			mushroomLogs[i] = genMushroomLog(i == 1)
		}
	}
	if full {
		return mushroomLogs[1]
	}
	return mushroomLogs[0]
}

func genMushroomLog(full bool) sprite {
	c := newCanvas(34, 20)
	x0, x1, cy, rad := 5.0, 29.0, 14.0, 3.6
	// The log: a lit cylinder lying across, its cut end showing rings.
	for y := int(cy - rad - 1); y <= int(cy+rad+1); y++ {
		for x := int(x0); x <= int(x1); x++ {
			ny := (float64(y) + 0.5 - cy) / rad
			if math.Abs(ny) > 1 {
				continue
			}
			l := 0.75 - (ny+1)/2*0.65
			k := cypressBark.idx(l, x, y)
			if hashf(x/3, y, 190) < 0.2 {
				k--
			}
			col := cypressBark.at(k)
			if ny < -0.35 && hashf(x, y, 191) < 0.7 { // moss along the top
				col = cypressLeaf.pick(0.5+(hashf(x, y, 192)-0.5)*0.5, x, y)
			}
			c.put(x, y, col)
		}
	}
	c.ellipse(x0, cy, 1.8, rad, func(x, y int, nx, ny float64) color.RGBA {
		if d := math.Hypot(nx, ny); d < 0.35 || (d > 0.55 && d < 0.7) {
			return woodRamp.at(1)
		}
		return woodRamp.at(2)
	})
	if full {
		for _, m := range [][3]float64{{12, 3.2, 4}, {17, 2.6, 6}, {22, 3.4, 3}} {
			mx, r, h := m[0], m[1], m[2]
			top := cy - rad - h
			for y := int(top); y < int(cy-rad); y++ { // the stalk
				c.put(int(mx), y, hex(0xe8e0cc))
			}
			c.ellipse(mx+0.5, top, r, r*0.7, func(x, y int, nx, ny float64) color.RGBA {
				if ny > 0.35 {
					return color.RGBA{} // a flat underside
				}
				if hashf(x, y, 193) < 0.2 {
					return hex(0xf4eee0) // white spots
				}
				return capRamp.pick(0.8-(nx+ny+1)*0.3, x, y)
			})
		}
	}
	c.outline(0.7)
	return mkUp(c, 17, cy+rad)
}

// --- on the ponds ------------------------------------------------------------------

// floaterKind is something floating on a swamp pond.
type floaterKind uint8

const (
	floatLily  floaterKind = iota // a raft of lily pads, some in flower
	floatAlgae                    // a slick of bright green algae
	floatStick                    // a fallen branch, half sunk
	numFloaterKinds
)

// floaterChance is how likely each spot on a pond holds each floater.
var floaterChance = [numFloaterKinds]float64{0.26, 0.1, 0.035}

const floaterVariants = 3

// floater is one thing floating on a pond.
type floater struct {
	kind    floaterKind
	variant int
	x, y    float64 // in tiles
}

// floatersOn appends to out what floats on pond tile (tx, ty).
func floatersOn(tx, ty int, out []floater) []floater {
	for i := 0; i < 2; i++ {
		h := func(s uint32) float64 { return hashf(tx*2+i, ty, 860+s) }
		r, sum := h(0), 0.0
		for k, c := range floaterChance {
			sum += c
			if r >= sum {
				continue
			}
			out = append(out, floater{
				kind:    floaterKind(k),
				variant: int(h(1) * floaterVariants),
				x:       float64(tx) + (float64(i)+0.2+0.6*h(2))/2,
				y:       float64(ty) + 0.25 + 0.6*h(3),
			})
			break
		}
	}
	return out
}

var floaterArt [numFloaterKinds][floaterVariants]sprite

func floaterSprite(k floaterKind, v int) sprite {
	if floaterArt[0][0].img == nil {
		for kind := range numFloaterKinds {
			for i := range floaterVariants {
				floaterArt[kind][i] = genFloater(kind, i)
			}
		}
	}
	return floaterArt[k][v]
}

var (
	lilyRamp   = mkRamp(0x24502a, 0x346c32, 0x4a873a, 0x66a046, 0x86b856)
	lilyFlower = mkRamp(0xc0607e, 0xe68aa6, 0xf6bfd0, 0xfff0f4)
)

func genFloater(k floaterKind, v int) sprite {
	rng := rand.New(rand.NewPCG(uint64(k)*19+5, uint64(v)*23+1))
	switch k {
	case floatLily:
		if envHD {
			return mkUp(genLilyRaftHD(rng, v)) // see florawet.go
		}
		c := newCanvas(22, 12)
		n := 2 + v
		var flowerAt [2]float64
		for i := range n {
			x, y := 5+rng.Float64()*12, 3.5+rng.Float64()*5
			rx := 2.4 + rng.Float64()*1.4
			notch := rng.Float64() * 2 * math.Pi
			c.ellipse(x, y, rx, rx*0.62, func(px, py int, nx, ny float64) color.RGBA {
				a := math.Atan2(ny, nx)
				if d := math.Abs(math.Remainder(a-notch, 2*math.Pi)); d < 0.3 && nx*nx+ny*ny > 0.05 {
					return color.RGBA{} // the cleft in a lily pad
				}
				return lilyRamp.pick(0.75-(nx+ny)*0.25-(nx*nx+ny*ny)*0.2, px, py)
			})
			if i == 0 {
				flowerAt = [2]float64{x, y - 1}
			}
		}
		if v != 1 { // a pink water lily
			fx, fy := int(flowerAt[0]), int(flowerAt[1])
			for _, p := range [][3]int{{0, -1, 2}, {-1, 0, 1}, {1, 0, 1}, {-1, -1, 2}, {1, -1, 2}, {0, 0, 3}} {
				c.put(fx+p[0], fy+p[1], lilyFlower.at(p[2]))
			}
			c.put(fx, fy, hex(0xf6d860))
		}
		c.outline(0.45)
		return mkUp(c, 11, 8)
	case floatAlgae:
		c := newCanvas(20, 10)
		for y := range c.h {
			for x := range c.w {
				nx, ny := (float64(x)+0.5-10)/9, (float64(y)+0.5-5)/4.5
				d := nx*nx + ny*ny
				n := vnoise(float64(x)*0.45, float64(y)*0.7, 870+uint32(v))
				if d < 1 && n > 0.25+d*0.45 && hashf(x, y, 871+uint32(v)) < 0.8 {
					c.put(x, y, algaeRamp.pick(0.3+n*0.6, x, y))
				}
			}
		}
		return mkUp(c, 10, 6)
	default:
		c := newCanvas(18, 8)
		y0, y1 := 3+rng.Float64()*2, 3+rng.Float64()*2
		c.line(2, y0, 15, y1, func(x, y int, t float64) color.RGBA { return trunkRamp.at(2 + int(t*2)%2) })
		c.line(8, (y0+y1)/2, 10+rng.Float64()*3, 0.5, func(x, y int, t float64) color.RGBA { return trunkRamp.at(2) })
		// A faint ring of ripples where it lies in the water.
		c.ellipse(9, (y0+y1)/2+1.5, 8.5, 2, func(x, y int, nx, ny float64) color.RGBA {
			if d := nx*nx + ny*ny; d > 0.7 && c.at(x, y).A == 0 {
				return withAlpha(pondRamp.at(4), 160)
			}
			return color.RGBA{}
		})
		c.outline(0.4)
		return mkUp(c, 9, 5)
	}
}

// drawFloaters draws what floats on tile (tx, ty) if it is a swamp pond.
func (g *Game) drawFloaters(dst drawTarget, cam camera, tx, ty int) {
	if g.ground == nil || g.ground.terr == nil || !g.ground.terr.pondTile(tx, ty) {
		return
	}
	var buf [2]floater
	for _, f := range floatersOn(tx, ty, buf[:0]) {
		sx, sy := cam.toScreen(f.x, f.y)
		drawAt(dst, floaterSprite(f.kind, f.variant), sx, sy, 1, 1, 0, nil)
	}
}
