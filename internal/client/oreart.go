package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Ores, bars, the furnace and the tools of each tier. The ore veins, the
// coal seams and the furnace in the world are drawn with the rocks: see
// rockart.go.

var (
	ironRamp = mkRamp(0x2b3036, 0x495159, 0x6c7680, 0x97a1ab, 0xc8d0d7)

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

// genFurnaceCanvas is the stone furnace, its mouth dark or (lit) full of
// fire; see furnaceScene in rockart.go.
func genFurnaceCanvas(lit bool) (*canvas, float64, float64) {
	return furnaceScene(lit).draw(hdK(), 5)
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
	rocks             [sim.NumTiers - 1][3]sprite // each ore vein, in three shapes; see rockart.go
	coal              [3]sprite                   // the coal seam, in three shapes
	furnace           [2]sprite                   // cold, lit
	axes, picks, hoes [sim.NumTiers]sprite        // in the hand, by tier: the axe as swung, the others as held
}

func genOreArt() oreArt {
	var a oreArt
	for o := range a.rocks {
		for v := range a.rocks[o] {
			c, ax, ay := oreCanvas(o, v)
			a.rocks[o][v] = mkUp(c, ax, ay)
		}
	}
	for v := range a.coal {
		c, ax, ay := oreCanvas(rkCoalOre, v)
		a.coal[v] = mkUp(c, ax, ay)
	}
	for i, lit := range []bool{false, true} {
		c, ax, ay := genFurnaceCanvas(lit)
		a.furnace[i] = mkUp(c, ax, ay)
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
	rocks := &g.art.ore.rocks[tier-1]
	return rocks[sim.HashTile(int(e.TX), int(e.TY))/8%uint32(len(rocks))]
}

// coalSeam is the sprite of the coal seam e.
func (g *Game) coalSeam(e protocol.EntityView) sprite {
	seams := &g.art.ore.coal
	return seams[sim.HashTile(int(e.TX), int(e.TY))/8%uint32(len(seams))]
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
		notes[sim.ToolItem(sim.ToolHoe, t)] = "Cầm trên tay (ô nhanh), nhấp chuột để xới đất; xới bằng cuốc tốt hay đào được khoai hơn."
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
