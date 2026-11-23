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

// iconSwordCanvas is a sword with its blade in rp.
func iconSwordCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	// Blade from the guard up to the tip, lit on one edge.
	for i := 0; i <= 9; i++ {
		x, y := 5+i, 10-i
		c.set(x, y, rp.pick(0.95, x, y))
		c.set(x, y+1, rp.pick(0.55, x, y+1))
		if i < 9 {
			c.set(x+1, y+1, rp.pick(0.35, x+1, y+1))
		}
	}
	c.set(14, 1, rp[len(rp)-1])
	// Guard across the blade, grip and pommel.
	c.line(2, 9, 6, 13, func(x, y int, _ float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	c.line(3, 12, 1, 14, func(x, y int, _ float64) color.RGBA { return woodRamp.pick(0.6, x, y) })
	c.set(0, 15, rp[2])
	c.outline(0.85)
	return c
}

// genHeldSwordOf is a sword with its blade in rp, as held: pivot at the grip.
func genHeldSwordOf(rp ramp) (*canvas, float64, float64) {
	c := newCanvas(7, 20)
	for y := 1; y <= 13; y++ {
		c.set(2, y, rp.pick(0.95, 2, y))
		c.set(3, y, rp.pick(0.6, 3, y))
		c.set(4, y, rp.pick(0.35, 4, y))
	}
	c.set(3, 0, rp[len(rp)-1])
	c.rect(0, 14, 7, 1, func(x, y int) color.RGBA { return trunkRamp.pick(0.5, x, y) }) // guard
	c.rect(3, 15, 1, 4, func(x, y int) color.RGBA { return woodRamp.pick(0.6, x, y) })  // grip
	c.set(3, 19, rp[2])
	c.outline(0.8)
	return c, 3, 17
}

// iconArmorCanvas is a breastplate of rp.
func iconArmorCanvas(rp ramp) *canvas {
	c := newCanvas(16, 16)
	c.polygon([]float64{2, 3, 5.5, 2, 8, 4, 10.5, 2, 14, 3, 13, 8, 12, 14, 4, 14, 3, 8}, func(x, y int) color.RGBA {
		nx := (float64(x) + 0.5 - 8) / 6
		l := 0.85 - nx*0.35 - float64(y)*0.02
		if x == 7 || x == 8 {
			l += 0.1 // the ridge down the middle
		}
		return rp.pick(l, x, y)
	})
	// Shoulders and a belt line.
	c.ellipse(3, 4, 2, 1.6, func(x, y int, nx, ny float64) color.RGBA { return rp.pick(lum(nx, ny), x, y) })
	c.ellipse(13, 4, 2, 1.6, func(x, y int, nx, ny float64) color.RGBA { return rp.pick(lum(nx, ny)*0.8, x, y) })
	c.line(4, 11, 12, 11, func(x, y int, _ float64) color.RGBA { return rp[1] })
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
		icons[sim.ArmorItem(t)] = iconArmorCanvas(tierRamps[t]).image()
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
		itemInfo[sim.SwordItem(t)] = "Cầm trên tay (ô nhanh) để đánh thú: mạnh hơn giáo và kiếm bậc dưới."
		itemInfo[sim.ArmorItem(t)] = fmt.Sprintf("Mặc khi có trong túi: đỡ %d%% sức cắn của rắn, cua. Chỉ bộ tốt nhất có tác dụng.",
			int(sim.ArmorBlock(t)*100+0.5))
	}
}
