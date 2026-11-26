package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Fruit trees: figs, fruiting ginkgos and coconut palms, each drawn hung
// with fruit while it is ripe and bare once picked. Their fruit and the bag
// icons are here too.

var (
	figLeafRamp = mkRamp(0x15301c, 0x204624, 0x2f602c, 0x447c34, 0x62993f)
	figRamp     = mkRamp(0x3a1430, 0x5e2148, 0x86325f, 0xa95278, 0xc77a96)
	nutRamp     = mkRamp(0x7a4a12, 0xa8701c, 0xd29a2c, 0xeec04a, 0xfbe08a)
	palmRamp    = mkRamp(0x173a1c, 0x255523, 0x386f29, 0x52902f, 0x78b13d)
	palmTrunk   = mkRamp(0x3b2b1a, 0x5a4328, 0x7a5c36, 0x977646)
	coconutRamp = mkRamp(0x2a1a10, 0x45301c, 0x644828, 0x846336)
)

// Fruit trees come in one size each: a fig or a ginkgo is a small tree, a
// coconut palm a tall one, a banana plant twice a man's height.
const (
	fruitScale  = 1.7
	palmScale   = 2.3
	bananaScale = 1.35
)

// fruitArt is the fruit trees' art, ripe and picked, two of each.
type fruitArt struct {
	figs, figsBare       []sprite
	ginkgos, ginkgosBare []sprite
	palms, palmsBare     []sprite
}

func genFruitArt() fruitArt {
	const each = 2
	mk := func() []sprite { return make([]sprite, each) }
	a := fruitArt{mk(), mk(), mk(), mk(), mk(), mk()}
	inParallel(6*each, func(i int) {
		v := uint64(i%each + 1)
		fig, ginkgo := []treeShape{shapeRound, shapeOak}[v-1], []treeShape{shapeOak, shapeTall}[v-1]
		switch i / each {
		case 0:
			a.figs[v-1] = fruitTree(fig, figLeafRamp, 11, 80+v, figRamp, 30)
		case 1:
			a.figsBare[v-1] = fruitTree(fig, figLeafRamp, 11, 80+v, nil, 0)
		case 2:
			a.ginkgos[v-1] = fruitTree(ginkgo, ginkgoRamp, 7, 90+v, nutRamp, 34)
		case 3:
			a.ginkgosBare[v-1] = fruitTree(ginkgo, ginkgoRamp, 7, 90+v, nil, 0)
		case 4:
			a.palms[v-1] = genPalm(palmScale, v, true)
		case 5:
			a.palmsBare[v-1] = genPalm(palmScale, v, false)
		}
	})
	return a
}

// fruitTree is a leafy tree with n fruits of rp hanging in its crown (none
// when rp is nil).
func fruitTree(shape treeShape, leaves ramp, tex uint32, seed uint64, rp ramp, n int) sprite {
	c, bx, by := broadleafCanvas(shape, leaves, tex, false, fruitScale, seed)
	if rp != nil {
		rng := rand.New(rand.NewPCG(seed, 17))
		for placed, tries := 0, 0; placed < n && tries < 600; tries++ {
			x, y := rng.IntN(c.w-1), rng.IntN(c.h-1)
			// On the leaves, in the lower part of the crown, not too high up.
			if !leafAt(c, x, y) || !leafAt(c, x+1, y+1) || !leafAt(c, x, y-2) {
				continue
			}
			if c.hd() {
				if !leafAt(c, x+4, y+4) || !leafAt(c, x-4, y-4) || !leafAt(c, x, y-7) {
					continue
				}
				placed++
				berryHD(c, float64(x), float64(y), 3.8, rp, leaves[0])
				continue
			}
			placed++
			c.set(x, y, rp.at(3))
			c.set(x+1, y, rp.at(2))
			c.set(x, y+1, rp.at(2))
			c.set(x+1, y+1, rp.at(1))
			c.set(x, y+2, rp.at(0)) // the shadow under it
		}
	}
	return trimmed(c, bx, by)
}

// leafAt reports whether (x, y) is a leaf pixel: opaque, not wood, not the
// outline.
func leafAt(c *canvas, x, y int) bool {
	p := c.at(x, y)
	return p.A > 0 && !isTrunk(p) && p != colInk && c.opaque(x-1, y) && c.opaque(x+1, y)
}

// genPalm is a coconut palm: a leaning, ringed trunk under a crown of long
// fronds, with a cluster of coconuts when ripe.
func genPalm(s float64, seed uint64, coconuts bool) sprite {
	rng := rand.New(rand.NewPCG(seed, 47))
	r := func(lo, hi float64) float64 { return lo + rng.Float64()*(hi-lo) }
	k := hdK()
	s *= k
	c := markHD(newCanvas(int(math.Ceil(64*s)), int(math.Ceil(66*s))))
	bx, by := math.Round(float64(c.w)/2), float64(c.h)-3*k
	lean := math.Copysign(r(5, 9), r(-1, 1)) * s
	tx, ty := bx+lean, by-r(40, 46)*s
	trunk := limb{bx, by, bx + lean*0.1, (by + ty) / 2, tx, ty, 1.6 * s, 1.1 * s}
	drawLimbIn(c, trunk, palmTrunk, 83)
	for t := 0.04; t < 0.98; t += 0.07 * k / s { // the rings left by old fronds, a hand apart
		x, y := trunk.at(t)
		w := lerpf(trunk.w0, trunk.w1, t)
		c.line(x-w, y, x+w, y, func(px, py int, _ float64) color.RGBA { return palmTrunk[0] })
		if c.hd() { // each ring a ridge: dark above, lit below
			c.line(x-w, y+1, x+w*0.2, y+1, func(px, py int, _ float64) color.RGBA { return palmTrunk[3] })
			c.line(x-w*0.6, y-1, x+w, y-1, func(px, py int, _ float64) color.RGBA { return palmTrunk[1] })
		}
	}
	n := 8 + rng.IntN(2)
	for i := range n {
		a := -math.Pi - 0.35 + (math.Pi+0.7)*float64(i)/float64(n-1) + r(-0.12, 0.12)
		frond(c, tx, ty, a, r(19, 24)*s, palmRamp, false, 150+uint32(i))
	}
	if coconuts { // hanging in front of the fronds' bases
		for _, d := range [][2]float64{{-3.6, 1.2}, {3.4, 1.6}, {-1.8, 2.6}, {1.9, 3}, {0, 4.6}} {
			nut := 1.25 * k // a coconut is the size of a head, whatever the palm's
			c.ellipse(tx+d[0]*nut, ty+d[1]*nut, 2.1*nut, 2.1*nut, func(x, y int, nx, ny float64) color.RGBA {
				if c.hd() { // the husk's fibres
					return coconutRamp.pick(lum(nx, ny)+(hashf(x/2, y, 151)-0.5)*0.25, x, y)
				}
				return coconutRamp.pick(lum(nx, ny), x, y)
			})
		}
	}
	c.outline(0.6)
	return trimmed(c, bx, by)
}

// fruitTreeSprite is fruit tree e, with fruit while it is ripe.
func (g *Game) fruitTreeSprite(e protocol.EntityView) (sprite, float64) {
	f := g.art.fruitOf()
	a := &f
	ripe, bare, sway := a.figs, a.figsBare, 0.03
	switch e.Kind {
	case sim.EntityGinkgoTree:
		ripe, bare = a.ginkgos, a.ginkgosBare
	case sim.EntityPalm:
		ripe, bare, sway = a.palms, a.palmsBare, 0.045
	}
	v := int(sim.HashTile(int(e.TX), int(e.TY)) / 8)
	if e.Ready {
		return ripe[v%len(ripe)], sway
	}
	return bare[v%len(bare)], sway
}

// fruitStumpSize is the size of tree stump a felled fruit tree of kind k
// leaves: a small one for the slender palms and bananas, a big one for figs
// and ginkgos.
func fruitStumpSize(k sim.EntityKind) int {
	if k == sim.EntityPalm || k == sim.EntityBananaTree {
		return 0
	}
	return 1
}

// --- icons -------------------------------------------------------------------

// genFruitIcons draws the bag icons of the fruit.
func genFruitIcons(icons *[sim.NumItems]*ebiten.Image) {
	fig := newCanvas(16, 16)
	fig.polygon([]float64{8, 3, 12, 8, 12.5, 12, 8, 14.5, 3.5, 12, 4, 8}, func(x, y int) color.RGBA {
		return figRamp.pick(lum((float64(x)-8)/5, (float64(y)-10)/5)+0.1, x, y)
	})
	fig.line(8, 1, 8, 3, func(x, y int, _ float64) color.RGBA { return trunkRamp[2] }) // the stalk
	fig.set(6, 8, figRamp[4])
	fig.outline(0.8)
	icons[sim.ItemFig] = fig.image()

	nut := func(rp ramp, char bool) *ebiten.Image {
		c := newCanvas(16, 16)
		for _, n := range [][2]float64{{6, 9}, {10.5, 8}} {
			c.ellipse(n[0], n[1], 3, 3.8, func(x, y int, nx, ny float64) color.RGBA { return rp.pick(lum(nx, ny)+0.1, x, y) })
		}
		if char {
			grillMarks(c, 0)
		}
		c.outline(0.8)
		return c.image()
	}
	icons[sim.ItemGinkgoNut] = nut(nutRamp, false)
	icons[sim.ItemRoastGinkgo] = nut(roastRamp, true)

	co := newCanvas(16, 16)
	co.ellipse(8, 9, 6, 5.6, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if hashf(x, y/2, 61) < 0.25 { // the husk's fibres
			l -= 0.2
		}
		return coconutRamp.pick(l, x, y)
	})
	for _, e := range [][2]int{{6, 7}, {9, 7}, {7, 9}} { // its three eyes
		co.set(e[0], e[1], coconutRamp[0])
	}
	co.outline(0.8)
	icons[sim.ItemCoconut] = co.image()
}

// The bag's notes on the fruit.
func init() {
	itemInfo[sim.ItemFig] = "Hái ở cây sung trong rừng và trên đồng cỏ. Ăn sống ngọt, mỗi cây cho 3 quả, 1 ngày sau lại ra quả."
	itemInfo[sim.ItemGinkgoNut] = "Hái ở cây bạch quả trên đồng cỏ. Ăn sống thì hơi hại; rang ở lửa trại (chuột phải) thì no và bổ."
	itemInfo[sim.ItemRoastGinkgo] = "Cầm hạt bạch quả, đứng gần lửa trại đang cháy, nhấp chuột phải để rang."
	itemInfo[sim.ItemCoconut] = "Hái ở cây dừa trên bãi cát ven biển. Nước dừa hồi máu tốt."
}
