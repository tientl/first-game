package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The leaf tent in the bag (sim.ItemLeafTent): an HD icon of the tent as it
// stands in the world, from the front, its open door dark, the lit slope on
// the left; and an old 16x16 icon for the small sizes, a drop on the
// ground and the hand.

func init() {
	regIcon(sim.ItemLeafTent, icLeafTent)
	itemInfo[sim.ItemLeafTent] = "Lều lá cuộn lại. Cầm trên tay (ô nhanh), nhấp chuột vào chỗ đất trống (hoặc V) để dựng — " +
		"không dựng được trên đá hay chỗ lội. Ngủ trong lều như giường: lúc chạng vạng và ban đêm, chết thì hồi sinh cạnh lều. " +
		"Ban ngày cầm rìu đánh vào để dỡ, lấy lại một nửa vật liệu."
}

var (
	icTentLeafR = mkRamp(0x0d2a20, 0x1d5634, 0x2a6e3c, 0x3b8744, 0x56a04e, 0x78b95c, 0xa2d272)
	icTentDarkR = mkRamp(0x0a0908, 0x191510, 0x2d2518, 0x4a3e26)
)

// The tent's corners in the icon's design box: the front feet, the front
// apex and the back apex straight above it, as the world draws it.
const (
	itFLX, itFRX, itFY = 1.8, 14.2, 14.0
	itAX, itAY         = 8.0, 4.6
	itBY               = 1.2
)

// icLeafTent paints the leaf tent: the two slopes thatched in rows of big
// leaves, the open front dark with a bed of grass inside, the poles crossed
// and lashed at the top, a leaf fallen in front.
func icLeafTent(p *icv) {
	back := itBY - itAY // the back, up the box from the front
	// The inside, seen through the open front: the back wall, the floor.
	door := sPoly(itFLX+0.6, itFY, itFRX-0.6, itFY, itAX, itAY+0.7)
	p.fill(door, func(f *ifrag) color.RGBA {
		if f.y > itFY-3.4 {
			l := 0.45 - (itFY-f.y)*0.08 - math.Abs(f.x-8)*0.03
			if math.Mod(f.x*1.7+f.y*2.3, 1.3) < 0.35 {
				l += 0.15
			}
			return pick(icStrawR, l)
		}
		return pick(icTentDarkR, 0.15+(f.y-itAY)*0.05)
	})
	p.bareLast()
	// The slopes, the shaded one first, each a bed of thatch under its leaves.
	for _, side := range []float64{1, -1} {
		fx := 8 + side*(itFRX-8)
		base := 0.62
		if side > 0 {
			base = 0.3
		}
		slope := sPoly(itAX, itAY, fx, itFY, fx, itFY+back, itAX, itAY+back)
		p.fill(slope, func(f *ifrag) color.RGBA { return pick(icTentLeafR, base-0.18) })
		// Rows of leaves from the foot up, each pointing down the slope.
		ang := math.Atan2(itFY-itAY, fx-itAX)
		for _, u := range []float64{0.92, 0.7, 0.48, 0.26, 0.06} {
			for _, v := range []float64{0.95, 0.62, 0.3, 0.0} {
				cx := itAX + u*(fx-itAX)
				cy := itAY + u*(itFY-itAY) + v*back
				l := base + (0.5-u)*0.08 - v*0.1
				ux, uy := math.Cos(ang), math.Sin(ang)
				cx, cy = cx+ux*1.2, cy+uy*1.2
				p.fill(iEll(cx, cy, 2.3, 1.15, ang).and(slope.grow(0.5)), func(f *ifrag) color.RGBA {
					dx, dy := f.x-cx, f.y-cy
					along, across := dx*ux+dy*uy, -dx*uy+dy*ux
					ll := l
					if across*side < 0 {
						ll += 0.1 // the half facing up catches the light
					}
					if math.Abs(across) < f.px*0.9 && along < 1.8 {
						ll += 0.16 // the midrib
					}
					if f.d > -f.px*1.2 {
						ll -= 0.1
					}
					return pick(icTentLeafR, ll)
				})
			}
		}
	}
	// The front poles, crossed and lashed at the top.
	for _, side := range []float64{-1, 1} {
		x0, y0 := 8+side*(itFRX-8+0.3), itFY+0.3
		x1, y1 := itAX-side*1.8, itAY-2.5
		a := mkAxis(x0, y0, x1, y1)
		p.fill(sCap(x0, y0, x1, y1, 0.46, 0.36), shRod(icBarkR, a, 0.44, 0.05, 0.4, uint32(5+side)))
		p.fill(sEll(x1, y1, 0.34, 0.34), shBulge(icStrawR, 0.3, 0.15))
	}
	a := mkAxis(itAX-1, itAY-0.6, itAX+1, itAY-0.6)
	p.fill(sEll(itAX, itAY-0.6, 1.0, 0.8), shCord(icRopeR, a, 0.8, 2.6))
	// A leaf fallen in front, and the stones at the feet.
	ux, uy := math.Cos(0.25), math.Sin(0.25)
	p.fill(iEll(11.2, 14.6, 1.5, 0.5, 0.25), func(f *ifrag) color.RGBA {
		across := -(f.x-11.2)*uy + (f.y-14.6)*ux
		if math.Abs(across) < f.px*0.8 {
			return pick(icTentLeafR, 0.75)
		}
		return pick(icTentLeafR, 0.5)
	})
	for _, s := range [][3]float64{{1.8, 14.4, 0.8}, {14.2, 14.4, 0.8}} {
		p.fill(sEll(s[0], s[1], s[2], s[2]*0.7), shSphere(icStoneR, s[0], s[1], s[2], s[2]*0.7, 0, false))
	}
}

// tentIcons draws the old icon of the leaf tent, for where an old icon is
// shown.
func tentIcons(icons *[sim.NumItems]*ebiten.Image) {
	icons[sim.ItemLeafTent] = tentOldCanvas().image()
}

// tentOldCanvas is the old 16x16 icon of the leaf tent.
func tentOldCanvas() *canvas {
	c := newCanvas(16, 16)
	// The open front, dark, with grass at the bottom.
	c.polygon([]float64{2.5, 15, 13.5, 15, 8, 5.5}, func(x, y int) color.RGBA {
		if y >= 12 {
			return strawRamp.pick(0.35+float64(x%3)*0.1, x, y)
		}
		return color.RGBA{0x19, 0x15, 0x10, 0xff}
	})
	slope := func(side float64, base float64) {
		fx := 8 + side*6.5
		c.polygon([]float64{8, 5, fx, 15, fx, 11, 8, 1}, func(x, y int) color.RGBA {
			l := base
			if (x+2*y)%4 == 0 {
				l -= 0.25
			} else if (x+2*y)%4 == 1 {
				l += 0.12
			}
			return tentBroadR.pick(l, x, y)
		})
	}
	slope(1, 0.35)
	slope(-1, 0.68)
	c.line(1.5, 15, 9.5, 3, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.8, x, y) })
	c.line(14.5, 15, 6.5, 3, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	c.set(8, 5, ropeRamp[2])
	c.outline(0.8)
	return c
}
