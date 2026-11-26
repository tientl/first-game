package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The bag icons of the cooking gear and the dishes: 16 by 16 like the rest
// (itemicons.go), in the same palettes, lit from the top left.

var (
	ciClay  = mkRamp(0x4a342c, 0x76584a, 0x9c7a66, 0xbf9e86, 0xdcc2aa)
	ciPot   = mkRamp(0x4a1e12, 0x7a341c, 0xa8502a, 0xcc7440, 0xeaa468)
	ciBroth = mkRamp(0x7a3410, 0xa8581c, 0xd08630, 0xecb450, 0xfbe090)
	ciCream = mkRamp(0x8a6a3c, 0xc0a064, 0xe4cc94, 0xf6e8c0, 0xfffbe8)
	ciDark  = mkRamp(0x2a100c, 0x4a1c12, 0x70301c, 0x96482a, 0xbc6c40)
	ciTea   = mkRamp(0x24481c, 0x3c7028, 0x60983a, 0x8cc058, 0xc0e088)
	ciJuice = mkRamp(0x6a1440, 0xa0245c, 0xd04478, 0xf07c9c, 0xfcb8c8)
	ciWater = mkRamp(0x1c4a78, 0x2c74a8, 0x4ca0d0, 0x84c8ec, 0xc8ecfc)
	ciPond  = mkRamp(0x24402c, 0x386040, 0x548458, 0x7ca878, 0xa8cca0)
	ciYolk  = mkRamp(0xb85c0c, 0xe88c14, 0xfcb824, 0xffdc58, 0xfff4b0)
)

// ciBall is a lit round thing.
func ciBall(c *canvas, x, y, rx, ry float64, rp ramp, lit float64) {
	c.ellipse(x, y, rx, ry, func(px, py int, nx, ny float64) color.RGBA {
		return rp.pick(lum(nx, ny)*lit, px, py)
	})
}

// ciBowl is a clay bowl seen from a little above, filled with fill (nil:
// empty). It leaves the top five rows free for what stands in the bowl or
// rises from it.
func ciBowl(c *canvas, fill ramp) {
	// The body: the lower half of a wide ellipse.
	c.ellipse(8, 8.5, 6.5, 6, func(px, py int, nx, ny float64) color.RGBA {
		if ny < 0 {
			return color.RGBA{}
		}
		return ciPot.pick(0.85-(nx+1)/2*0.6-ny*0.15, px, py)
	})
	c.rect(6, 14, 4, 1, func(px, py int) color.RGBA { return ciPot[1] }) // its foot
	// The rim, and what is in it.
	c.ellipse(8, 8.5, 6.5, 2.4, func(px, py int, nx, ny float64) color.RGBA {
		return ciPot.pick(0.95-(nx+1)/2*0.4, px, py)
	})
	c.ellipse(8, 8.7, 5.3, 1.6, func(px, py int, nx, ny float64) color.RGBA {
		if fill == nil {
			return ciPot.pick(0.1+max(0, ny)*0.3, px, py)
		}
		return fill.pick(0.72-(nx+1)/2*0.45-ny*0.1, px, py)
	})
}

// ciSteam is two wisps rising from a bowl.
func ciSteam(c *canvas) {
	for _, w := range [][2]int{{5, 5}, {10, 4}} {
		for i := 0; i < 4; i++ {
			x := w[0] + []int{0, 1, 0, -1}[i]
			c.put(x, w[1]-i, color.RGBA{236, 240, 248, uint8(230 - i*40)})
		}
	}
}

// ciFlask is a hide flask: a bag sewn down one side, a neck with a wooden
// stopper and a thong.
func ciFlask(c *canvas) {
	ciBall(c, 8, 10, 5, 5, iiHide, 1)
	c.rect(6, 3, 4, 4, func(px, py int) color.RGBA { return iiHide.pick(0.75-float64(px-6)*0.15, px, py) })
	c.rect(6, 1, 4, 2, func(px, py int) color.RGBA { return iiWood.pick(0.9-float64(px-6)*0.18, px, py) })
	c.rect(5, 5, 6, 1, func(px, py int) color.RGBA { return iiStraw[2] }) // the thong round the neck
	for y := 7; y < 14; y += 2 {
		c.put(11, y, iiHide[0]) // the seam
	}
	c.put(5, 8, iiHide[4])
	c.put(6, 7, iiHide[4])
}

// ciDrop is a drop of rp in the lower right corner, marking what a vessel holds.
func ciDrop(c *canvas, rp ramp) {
	for _, p := range [][3]int{{12, 9, 3}, {11, 10, 3}, {12, 10, 4}, {13, 10, 2}, {11, 11, 2}, {12, 11, 3}, {13, 11, 1}, {11, 12, 2}, {12, 12, 2}, {13, 12, 1}, {12, 13, 1}} {
		c.put(p[0], p[1], rp[p[2]])
	}
	for _, p := range [][2]int{{12, 8}, {10, 10}, {10, 11}, {14, 10}, {14, 11}, {10, 12}, {14, 12}, {11, 13}, {13, 13}, {12, 14}, {11, 9}, {13, 9}} {
		c.put(p[0], p[1], colInk)
	}
}

// ciChop is a piece of meat: a rounded slab of rp with a rim of fat and a
// bone end, crossed by marks of mark (nil: none).
func ciChop(c *canvas, rp ramp, mark *color.RGBA) {
	c.line(10, 10, 13.5, 13.5, func(px, py int, t float64) color.RGBA { return iiBone[3] })
	c.line(10, 11, 13, 14, func(px, py int, t float64) color.RGBA { return iiBone[2] })
	c.put(14, 13, iiBone[4])
	c.put(13, 14, iiBone[4])
	c.ellipse(7, 7.5, 5.6, 4.8, func(px, py int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		if d > 0.7 && nx+ny < 0.2 {
			return iiBone.pick(0.9-d*0.2, px, py) // fat
		}
		if mark != nil && (px+py)%3 == 0 && d < 0.6 {
			return *mark
		}
		return rp.pick(lum(nx, ny)*0.95, px, py)
	})
}

// ciFish is a fish lying head left, of rp, with a tail of tail.
func ciFish(c *canvas, rp ramp, tail color.RGBA, y float64) {
	c.polygon([]float64{10.5, y, 14.5, y - 3.2, 13.5, y, 14.5, y + 3.2}, func(px, py int) color.RGBA { return tail })
	c.ellipse(7, y, 5.5, 2.9, func(px, py int, nx, ny float64) color.RGBA {
		return rp.pick(0.9-ny*0.4-nx*0.1, px, py)
	})
	c.put(3, int(y)-1, colInk)
}

func genCookIcons(icons *[sim.NumItems]*ebiten.Image) {
	for k := range cookIconDraws() {
		icons[k] = cookIconCanvas(k).image()
	}
}

// cookIconCanvas is the icon of cooking item k, nil if it has none.
func cookIconCanvas(k sim.ItemKind) *canvas {
	draw := cookIconDraws()[k]
	if draw == nil {
		return nil
	}
	c := newCanvas(16, 16)
	draw(c)
	c.outline(0.8)
	return c
}

// cookIconDraws is how each cooking item's icon is drawn.
func cookIconDraws() map[sim.ItemKind]func(c *canvas) {
	m := make(map[sim.ItemKind]func(c *canvas))
	set := func(k sim.ItemKind, draw func(c *canvas)) { m[k] = draw }
	set(sim.ItemClay, func(c *canvas) {
		ciBall(c, 5.5, 10.5, 4.5, 3.6, ciClay, 0.9)
		ciBall(c, 11, 11, 4, 3.2, ciClay, 0.8)
		ciBall(c, 8.2, 6.8, 4.6, 3.8, ciClay, 1)
		c.put(6, 5, hex(0xf6ece0)) // wet shine
		c.put(7, 5, hex(0xf6ece0))
		c.put(4, 9, hex(0xf6ece0))
	})
	set(sim.ItemClayPot, func(c *canvas) {
		c.ellipse(8, 9.5, 6.5, 5.4, func(px, py int, nx, ny float64) color.RGBA {
			l := lum(nx, ny)
			if py == 8 || (py == 10 && px%2 == 0) {
				l -= 0.22 // the scratched band
			}
			return ciPot.pick(l, px, py)
		})
		c.rect(4, 3, 8, 3, func(px, py int) color.RGBA { return ciPot.pick(0.85-float64(px-4)/8*0.5, px, py) })
		c.ellipse(8, 3.5, 4.8, 1.7, func(px, py int, nx, ny float64) color.RGBA { return ciPot.pick(0.95-(nx+1)/2*0.4, px, py) })
		c.ellipse(8, 3.6, 3.4, 0.9, func(px, py int, nx, ny float64) color.RGBA { return ciPot[0] })
		c.put(1, 8, ciPot[3]) // lugs
		c.put(14, 8, ciPot[1])
	})
	set(sim.ItemClayBowl, func(c *canvas) { ciBowl(c, nil) })
	set(sim.ItemGrill, func(c *canvas) {
		green := mkRamp(0x2c3a1c, 0x485c2a, 0x6a843a, 0x90aa50, 0xb8cc74)
		for i := 0; i < 6; i++ {
			x := 2.5 + float64(i)*2.2
			c.line(x+2, 3, x, 13, func(px, py int, t float64) color.RGBA { return green.at(3 - i%2) })
		}
		for _, y := range []float64{4, 12} {
			c.line(1, y+0.5, 15, y-0.5, func(px, py int, t float64) color.RGBA { return iiWood[3] })
			c.line(1, y+1.5, 15, y+0.5, func(px, py int, t float64) color.RGBA { return iiWood[1] })
		}
		for _, p := range [][2]int{{2, 4}, {13, 3}, {2, 12}, {13, 11}} {
			c.put(p[0], p[1], iiStraw[3]) // the lashings
			c.put(p[0], p[1]+1, iiStraw[1])
		}
	})
	set(sim.ItemStonePan, func(c *canvas) {
		c.ellipse(8, 9.5, 7, 4.2, func(px, py int, nx, ny float64) color.RGBA { return iiStone.pick(0.45-(nx+1)/2*0.3, px, py) })
		c.ellipse(8, 8, 7, 4.2, func(px, py int, nx, ny float64) color.RGBA {
			d := nx*nx + ny*ny
			l := 0.9 - (nx+1)/2*0.35 - ny*0.1
			if d < 0.62 {
				l -= 0.22 // the hollow
			}
			if d < 0.62 && d > 0.45 && nx+ny < 0 {
				l -= 0.15
			}
			return iiStone.pick(l, px, py)
		})
		c.put(6, 7, iiStone[4])
		c.put(7, 7, iiGlint)
	})
	set(sim.ItemFlask, ciFlask)
	set(sim.ItemDirtyWater, func(c *canvas) { ciFlask(c); ciDrop(c, ciPond) })
	set(sim.ItemCleanWater, func(c *canvas) { ciFlask(c); ciDrop(c, ciWater) })
	set(sim.ItemSeaWater, func(c *canvas) {
		ciBowl(c, ciWater)
		c.put(6, 8, iiGlint)
		c.put(7, 8, ciWater[4])
		c.put(10, 9, ciWater[4])
	})
	set(sim.ItemSalt, func(c *canvas) {
		c.ellipse(8, 13, 6.5, 2, func(px, py int, nx, ny float64) color.RGBA { return iiLeaf.pick(0.7-(nx+1)/2*0.5, px, py) })
		c.polygon([]float64{2.5, 13, 8, 4, 13.5, 13}, func(px, py int) color.RGBA {
			l := 0.95 - float64(px-3)/11*0.5 + (hashf(px, py, 77)-0.5)*0.35
			return iiWhite.pick(l, px, py)
		})
		c.put(7, 6, iiGlint)
		c.put(5, 10, iiGlint)
		c.put(9, 9, iiGlint)
	})
	set(sim.ItemFlour, func(c *canvas) {
		// A little sack, its neck tied, flour spilling from it.
		ciBall(c, 7.5, 10, 5.5, 4.6, iiPale, 1)
		c.rect(5, 3, 5, 3, func(px, py int) color.RGBA { return iiPale.pick(0.8-float64(px-5)*0.12, px, py) })
		c.rect(5, 5, 5, 1, func(px, py int) color.RGBA { return iiHide[1] })
		c.ellipse(11.5, 13, 3.4, 1.6, func(px, py int, nx, ny float64) color.RGBA { return iiWhite.pick(0.9-(nx+1)/2*0.3, px, py) })
		c.put(6, 9, iiPale[0])
		c.put(7, 10, iiPale[0])
		c.put(8, 9, iiPale[0])
	})
	set(sim.ItemSaltedMeat, func(c *canvas) {
		ciChop(c, iiMeat, nil)
		for _, p := range [][2]int{{5, 6}, {8, 5}, {7, 8}, {10, 8}, {4, 9}, {8, 10}, {6, 4}} {
			c.put(p[0], p[1], iiGlint)
		}
	})
	set(sim.ItemSmokedMeat, func(c *canvas) {
		ciChop(c, ciDark, nil)
		c.line(7, 1, 7, 4, func(px, py int, t float64) color.RGBA { return iiStraw[3] }) // the cord it hung by
		c.put(5, 6, ciDark[4])
		c.put(6, 5, ciDark[4])
	})
	set(sim.ItemJerky, func(c *canvas) {
		for i, s := range [][3]float64{{4, 2, 13}, {8, 3, 14}, {12, 2, 12}} {
			for y := int(s[1]); y < int(s[2]); y++ {
				x := s[0] + math.Sin(float64(y)*0.9+float64(i))*0.8
				for dx := -1; dx <= 1; dx++ {
					c.put(int(x)+dx, y, ciDark.at(3-dx-(y+i)%2))
				}
			}
		}
		c.line(2, 2, 14, 2, func(px, py int, t float64) color.RGBA { return iiStraw[2] })
	})
	set(sim.ItemDriedFish, func(c *canvas) {
		ciFish(c, iiHide, iiHide[1], 8)
		for x := 5; x < 11; x += 2 {
			c.put(x, 9, iiHide[1]) // the ribs showing through
		}
		c.line(3, 8, 11, 8, func(px, py int, t float64) color.RGBA { return iiHide[4] }) // split along the back
		c.put(6, 6, iiGlint)                                                             // salt
		c.put(9, 7, iiGlint)
	})
	set(sim.ItemStew, func(c *canvas) {
		ciBowl(c, ciBroth)
		c.put(6, 8, iiLeaf[3])
		c.put(9, 9, iiLeaf[3])
		c.put(7, 9, ciDark[3])
		c.put(8, 8, ciDark[2])
		c.put(11, 8, iiTuber[3])
		c.put(5, 9, iiGold[3])
		ciSteam(c)
	})
	set(sim.ItemHotSoup, func(c *canvas) {
		ciBowl(c, ciCream)
		c.put(6, 8, iiFish[3])
		c.put(7, 8, iiFish[2])
		c.put(10, 9, iiLeaf[3])
		c.put(9, 8, iiRed[3])
		ciSteam(c)
		c.put(8, 3, color.RGBA{236, 240, 248, 200})
		c.put(8, 2, color.RGBA{236, 240, 248, 140})
	})
	set(sim.ItemDinoStew, func(c *canvas) {
		ciBowl(c, ciDark)
		// A big bone standing in it.
		c.line(8, 9, 12, 3, func(px, py int, t float64) color.RGBA { return iiBone[3] })
		c.line(9, 9, 13, 3, func(px, py int, t float64) color.RGBA { return iiBone[2] })
		c.put(12, 2, iiBone[4])
		c.put(13, 2, iiBone[4])
		c.put(13, 4, iiBone[4])
		c.put(11, 3, iiBone[4])
		c.put(5, 8, ciDark[4])
		c.put(6, 9, iiLeaf[3])
		c.put(4, 5, color.RGBA{236, 240, 248, 220})
		c.put(5, 4, color.RGBA{236, 240, 248, 180})
		c.put(4, 3, color.RGBA{236, 240, 248, 140})
	})
	set(sim.ItemHerbTea, func(c *canvas) {
		ciBowl(c, ciTea)
		c.put(7, 8, ciTea[4])
		c.put(8, 9, ciTea[1])
		// A sprig laid on the rim.
		c.line(9, 8, 13, 5, func(px, py int, t float64) color.RGBA { return iiLeaf[1] })
		c.put(12, 4, iiLeaf[3])
		c.put(13, 6, iiLeaf[3])
		c.put(11, 5, iiLeaf[4])
		ciSteam(c)
	})
	set(sim.ItemHerbFish, func(c *canvas) {
		ciFish(c, iiCooked, iiChar[2], 9.5)
		for x := 5; x < 11; x += 2 {
			c.put(x, 9, iiChar[1])
			c.put(x+1, 10, iiChar[1])
		}
		// Sprigs laid along its back.
		c.line(3, 6, 12, 4, func(px, py int, t float64) color.RGBA { return iiLeaf[1] })
		for _, p := range [][2]int{{4, 5}, {6, 4}, {7, 6}, {9, 3}, {10, 5}, {12, 3}} {
			c.put(p[0], p[1], iiLeaf.at(3+p[0]%2))
		}
	})
	set(sim.ItemSearedMeat, func(c *canvas) {
		m := iiChar[1]
		ciChop(c, iiCooked, &m)
		c.put(5, 5, iiCooked[4])
	})
	set(sim.ItemFriedEgg, func(c *canvas) {
		for _, b := range [][4]float64{{8, 8.5, 6.5, 4.6}, {4.5, 10, 3, 2.5}, {11.5, 6.5, 3.2, 2.6}} {
			c.ellipse(b[0], b[1], b[2], b[3], func(px, py int, nx, ny float64) color.RGBA {
				if nx*nx+ny*ny > 0.8 && nx+ny > 0.3 {
					return ciCream[2]
				}
				return iiWhite.pick(0.98-(nx+1)/2*0.2-max(0, ny)*0.15, px, py)
			})
		}
		ciBall(c, 8, 8, 2.6, 2.3, ciYolk, 1.25)
		c.put(7, 7, iiGlint)
	})
	set(sim.ItemHerbMedicine, func(c *canvas) {
		// A leaf folded round a green paste and tied.
		c.ellipse(8, 9, 6, 4.6, func(px, py int, nx, ny float64) color.RGBA {
			l := lum(nx, ny)
			if px == 8 || (px+py)%4 == 0 && nx > -0.6 {
				l -= 0.18 // veins
			}
			return iiLeaf.pick(l, px, py)
		})
		c.ellipse(8, 5.2, 3, 1.6, func(px, py int, nx, ny float64) color.RGBA { return ciTea.pick(0.9-(nx+1)/2*0.4, px, py) })
		c.rect(2, 9, 12, 1, func(px, py int) color.RGBA { return iiStraw[3] })
		c.put(8, 10, iiStraw[1])
		c.put(9, 11, iiStraw[3])
		c.put(7, 11, iiStraw[3])
	})
	set(sim.ItemJuice, func(c *canvas) {
		ciBowl(c, ciJuice)
		c.put(6, 8, ciJuice[4])
		ciBall(c, 11, 4.5, 2.2, 2.2, iiBerry, 1.1)
		c.put(11, 2, iiLeaf[3])
		c.put(12, 2, iiLeaf[2])
		ciBall(c, 5, 5, 1.8, 1.8, iiBerry, 1.1)
	})
	set(sim.ItemBread, func(c *canvas) {
		crust := mkRamp(0x6a3c14, 0x96601e, 0xc08a34, 0xdcb05a, 0xf0d48e)
		c.ellipse(8, 9, 6.8, 4.8, func(px, py int, nx, ny float64) color.RGBA {
			if ny < 0.3 && (px-py+16)%4 == 0 && math.Abs(nx) < 0.7 {
				return crust[4] // a slash
			}
			return crust.pick(lum(nx, ny)*0.9, px, py)
		})
		c.put(5, 6, iiGlint)
		c.put(9, 5, hex(0xf6ecd4))
	})
	set(sim.ItemFruitPie, func(c *canvas) {
		crust := mkRamp(0x6a3c14, 0x96601e, 0xc08a34, 0xdcb05a, 0xf0d48e)
		c.ellipse(8, 10, 7, 4.4, func(px, py int, nx, ny float64) color.RGBA { return crust.pick(0.5-(nx+1)/2*0.3, px, py) })
		c.ellipse(8, 8.5, 7, 4.4, func(px, py int, nx, ny float64) color.RGBA {
			d := nx*nx + ny*ny
			if d > 0.66 {
				return crust.pick(0.95-(nx+1)/2*0.4, px, py) // the crimped edge
			}
			if (px+py)%3 == 0 || (px-py+16)%3 == 0 {
				return crust.pick(0.85-(nx+1)/2*0.3, px, py) // the lattice
			}
			return iiBerry.pick(0.7-(nx+1)/2*0.4, px, py)
		})
	})
	return m
}

// The bag's notes on the cooking things.
func init() {
	for k, s := range map[sim.ItemKind]string{
		sim.ItemClay:         "Đào bằng tay ở bãi đất sét ven ao, hồ (đôi khi trên bãi biển). Nung cạnh Lò nung thành Nồi đất, Bát đất; đắp Lò nướng bánh.",
		sim.ItemClayPot:      "Cầm trên tay, nhấn V cạnh lửa trại để bắc nồi lên. Nhấp vào bếp để bỏ nguyên liệu: nấu canh, súp, hầm, trà, đun sôi nước.",
		sim.ItemClayBowl:     "Đựng canh, súp, nước ép — ăn xong bát trở về túi. Cầm trên tay, nhấn V ở mép biển để múc nước biển đem phơi muối.",
		sim.ItemGrill:        "Cầm trên tay, nhấn V cạnh lửa trại để gác lên. Nướng 4 món cùng lúc; cá + thảo mộc thành Cá nướng thảo mộc.",
		sim.ItemStonePan:     "Cầm trên tay, nhấn V cạnh lửa trại để kê lên. Áp chảo thịt với muối, rán trứng.",
		sim.ItemFlask:        "Cầm trên tay, nhấn V ở mép ao, hồ để múc nước ngọt. Nước ao phải đun sôi trong Nồi đất mới sạch.",
		sim.ItemDirtyWater:   "Nước ao chưa đun. Cho vào Nồi đất trên lửa để đun sôi, hoặc nấu Trà thảo dược cùng 2 thảo mộc.",
		sim.ItemCleanWater:   "Nước đã đun sôi. Uống hồi chút máu; bình trở về túi.",
		sim.ItemSeaWater:     "Đặt lên Giàn phơi ở gần biển: nước bốc hơi để lại Muối, bát trở về.",
		sim.ItemSalt:         "Ướp Thịt muối; nêm vào canh, súp, bánh để được thêm một phần; cần cho Thịt áp chảo và Thịt hầm khủng long.",
		sim.ItemFlour:        "Giã từ hạt ở Cối giã. 2 bột nướng thành Bánh mì ở Lò nướng bánh; thêm quả thành Bánh quả.",
		sim.ItemSaltedMeat:   "Thịt ướp muối, ăn sống không hại. Treo lên Giàn phơi thành Thịt hun khói; nấu được như thịt.",
		sim.ItemSmokedMeat:   "Thịt muối hun trên Giàn phơi. Rất no, để dành đi đường xa.",
		sim.ItemJerky:        "Thịt phơi trên Giàn phơi (đặt giàn gần lửa thì hun khói, nhanh gấp đôi).",
		sim.ItemDriedFish:    "Cá phơi trên Giàn phơi.",
		sim.ItemStew:         "Nồi đất: thịt + rau củ + bát. Ăn xong hồi máu nhanh trong 1 phút.",
		sim.ItemHotSoup:      "Nồi đất: cá hoặc sò + rau củ + bát. Ăn xong không sợ lạnh trong 4 phút.",
		sim.ItemDinoStew:     "Nồi đất: 2 thịt + xương + muối + bát. No lâu 8 phút, đánh mạnh hơn 4 phút.",
		sim.ItemHerbTea:      "Nồi đất: 2 thảo mộc + bình nước ao. Giải độc, hồi máu nhanh nửa phút.",
		sim.ItemHerbFish:     "Giàn nướng: cá + thảo mộc. Ăn xong chạy nhanh hơn trong 2 phút.",
		sim.ItemSearedMeat:   "Chảo đá: thịt + muối. Ăn xong đánh mạnh hơn trong 1 phút rưỡi.",
		sim.ItemFriedEgg:     "Chảo đá: rán một quả trứng.",
		sim.ItemHerbMedicine: "Cối giã: 2 thảo mộc. Hồi máu và giải độc.",
		sim.ItemJuice:        "Cối giã: 2 quả + bát. Hồi máu, đỡ đói một chút.",
		sim.ItemBread:        "Lò nướng bánh: 2 bột. Rất no.",
		sim.ItemFruitPie:     "Lò nướng bánh: 2 bột + quả. No lâu, hồi máu nhanh 45 giây.",
	} {
		itemInfo[k] = s
	}
}
