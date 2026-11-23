package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Swords, stone walls, and the icons of armour (worn armour: armorart.go).

// genStoneWall is a stretch of wall of stacked, squared stones.
func genStoneWall() (*canvas, float64, float64) {
	c := newCanvas(24, 30)
	const by = 26
	// Battlements: a few stones higher than the rest.
	tops := [6]int{6, 9, 6, 9, 6, 9}
	for y := 4; y <= by; y++ {
		for x := 0; x < 24; x++ {
			col := x / 4
			if y < tops[col] {
				continue
			}
			row := (y - 4) / 4
			// Bricks of 4x4 stones, every other row offset by 2.
			bx := (x + 2*(row%2)) % 4
			by4 := (y - 4) % 4
			l := 0.78 - float64(x)/24*0.35 + (hashf((x+2*(row%2))/4, row, 920)-0.5)*0.25
			if bx == 3 || by4 == 3 {
				l -= 0.45 // mortar
			} else if bx == 0 || by4 == 0 {
				l += 0.12 // lit edge
			}
			if y > by-3 {
				l -= 0.12
			}
			c.set(x, y, stoneRamp.pick(l, x, y))
		}
	}
	// A little moss in the cracks.
	for x := 0; x < 24; x++ {
		y := by - int(hashf(x, 0, 921)*3)
		if hashf(x, 1, 922) < 0.3 && c.opaque(x, y) {
			c.set(x, y, mossRamp.pick(0.5, x, y))
		}
	}
	c.outline(0.8)
	return c, 12, by
}

// iconSwordCanvas is a stone knife: a leaf-shaped blade knapped from rp,
// lashed into a wooden handle with a hide binding.
func iconSwordCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{5.5, 10.5, 7.5, 6, 11, 3, 14.5, 1.5, 13, 5, 10, 8.5}, func(x, y int) color.RGBA {
		l := 0.8 - float64(x-y)/14*0.3
		if hashf(x, y, 63) < 0.3 {
			l -= 0.2 // flake scars
		}
		return rp.pick(l, x, y)
	})
	c.line(7, 9, 13.5, 2.5, func(x, y int, _ float64) color.RGBA { return rp[len(rp)-1] })
	c.line(1, 15, 6, 10, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.6, x, y) }) // the handle
	c.line(2, 15, 6.5, 10.5, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.35, x, y) })
	for _, p := range [][2]int{{4, 11}, {5, 11}, {5, 12}, {6, 12}} { // the binding
		c.set(p[0], p[1], hideRamp.pick(0.5, p[0], p[1]))
	}
	c.outline(0.85)
	return c
}

// genHeldSwordOf is a stone knife with its blade in rp, as held: pivot at
// the grip.
func genHeldSwordOf(rp ramp) (*canvas, float64, float64) {
	c := newCanvas(7, 16)
	c.polygon([]float64{3.5, 0, 5.5, 4, 5, 9, 2, 9, 1.5, 4}, func(x, y int) color.RGBA {
		l := 0.95 - float64(x-1)*0.13
		if hashf(x, y, 64) < 0.25 {
			l -= 0.2
		}
		return rp.pick(l, x, y)
	})
	c.rect(2, 9, 3, 2, func(x, y int) color.RGBA { return hideRamp.pick(0.5, x, y) })  // the binding
	c.rect(3, 11, 1, 4, func(x, y int) color.RGBA { return woodRamp.pick(0.6, x, y) }) // the handle
	c.outline(0.85)
	return c, 3, 13
}

// iconArmorCanvas is tier t's armour: a hide tunic, then bones lashed
// across it, agate scales, and a tyrant's teeth.
func iconArmorCanvas(t sim.Tier) *canvas {
	c := newCanvas(16, 16)
	hide := hideRamp
	if t == sim.TierDiamond {
		hide = darkHideRamp
	}
	c.polygon([]float64{2, 3, 5.5, 2, 8, 4, 10.5, 2, 14, 3, 13, 8, 12.5, 14, 3.5, 14, 3, 8}, func(x, y int) color.RGBA {
		nx := (float64(x) + 0.5 - 8) / 6
		l := 0.8 - nx*0.3 - float64(y)*0.02
		if hashf(x, y, 65) < 0.15 {
			l -= 0.2
		}
		return hide.pick(l, x, y)
	})
	switch t {
	case sim.TierCopper: // a fur trim at the neck and hem
		c.line(4, 13, 12, 13, func(x, y int, _ float64) color.RGBA { return furTrim.pick(0.6+float64(x%2)*0.3, x, y) })
		c.line(5, 4, 11, 4, func(x, y int, _ float64) color.RGBA { return furTrim.pick(0.7, x, y) })
	case sim.TierIron: // ribs of bone lashed across
		for _, by := range []float64{6, 8.5, 11} {
			c.line(4, by, 12, by, func(x, y int, _ float64) color.RGBA { return hornRamp.pick(0.85-float64(x)*0.03, x, y) })
		}
	case sim.TierPlatinum: // scales of agate
		for y := 5; y <= 12; y += 2 {
			for x := 4 + (y/2)%2; x <= 11; x += 2 {
				c.set(x, y, agateRamp.pick(0.85-float64(x)*0.03, x, y))
				c.set(x, y+1, agateRamp.pick(0.45, x, y+1))
			}
		}
	case sim.TierDiamond: // a row of teeth on a cord
		c.line(4, 6, 12, 6, func(x, y int, _ float64) color.RGBA { return hide[0] })
		for x := 4; x <= 12; x += 2 {
			c.set(x, 7, toothArt[3])
			c.set(x, 8, toothArt[2])
		}
	}
	// Shoulders: hide, bone, agate.
	shoulder := hide
	switch t {
	case sim.TierIron, sim.TierDiamond:
		shoulder = hornRamp
	case sim.TierPlatinum:
		shoulder = agateRamp
	}
	c.ellipse(3, 4, 2, 1.6, func(x, y int, nx, ny float64) color.RGBA { return shoulder.pick(lum(nx, ny), x, y) })
	c.ellipse(13, 4, 2, 1.6, func(x, y int, nx, ny float64) color.RGBA { return shoulder.pick(lum(nx, ny)*0.8, x, y) })
	c.outline(0.85)
	return c
}

// gearArt is the art of stone walls and of the swords of every tier.
type gearArt struct {
	stoneWall sprite
	swords    [sim.NumTiers]sprite // as held: pivot at the grip (stone has none)
}

func genGearArt() gearArt {
	var a gearArt
	c, ax, ay := genStoneWall()
	a.stoneWall = mk(c, ax, ay)
	for t := sim.TierCopper; t < sim.NumTiers; t++ {
		c, ax, ay := genHeldSwordOf(tierRamps[t])
		a.swords[t] = mk(c, ax, ay)
	}
	return a
}

// genGearIcons draws the bag icons of the swords and armour of every tier.
func genGearIcons(icons *[sim.NumItems]*ebiten.Image) {
	for t := sim.TierCopper; t < sim.NumTiers; t++ {
		icons[sim.SwordItem(t)] = iconSwordCanvas(tierRamps[t]).image()
		icons[sim.ArmorItem(t)] = iconArmorCanvas(t).image()
	}
}

// isSword reports whether k is a sword of any tier.
func isSword(k sim.ItemKind) bool {
	_, ok := sim.SwordTier(k)
	return ok
}

// The bag's notes on swords and armour.
func init() {
	for t := sim.TierCopper; t < sim.NumTiers; t++ {
		itemInfo[sim.SwordItem(t)] = "Cầm trên tay (ô nhanh) để đánh thú: mạnh hơn giáo và dao bậc dưới."
		itemInfo[sim.ArmorItem(t)] = fmt.Sprintf("Mặc khi có trong túi: đỡ %d%% sức tấn công của thú dữ. Chỉ bộ tốt nhất có tác dụng.",
			int(sim.ArmorBlock(t)*100+0.5))
	}
}
