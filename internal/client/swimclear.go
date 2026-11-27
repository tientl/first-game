package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// How clear the water is where a swimmer is, so how much of what is under
// its surface shows (swimhd.go): the open sea, a deep lake and a river's
// middle hide all but a dim hint of the chest; the shallows, lagoons and
// reefs (the sea's own depth says how shallow, terrainsea.go), a ford, a
// lake's or a river's edge show the body to the feet. It eases from one to
// the other with the depth and the distance from the bank, so a swimmer
// going out fades into the water and coming in comes out of it.

// sheetClarity, when not negative, is the clarity waterClarity gives
// everywhere: for the motion sheets.
var sheetClarity = -1.0

// waterClarity is how clear the water at (x, y) tiles is: 1 clear to the
// bottom, 0 too deep to see into.
func (g *Game) waterClarity(x, y float64) float64 {
	if sheetClarity >= 0 {
		return sheetClarity
	}
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	var terr *terrain
	if g.ground != nil {
		terr = g.ground.terr
	}
	seaDepth := func() (float64, bool) {
		if terr == nil || terr.depth == nil {
			return 0, false
		}
		return terr.depthAt(x, y), true
	}
	byBank := func(clearTo, murkAt float64) float64 { // clear by the bank, murky out from it
		return 1 - smooth(clamp01((g.landDist(x, y)-clearTo)/(murkAt-clearTo)))
	}
	switch g.tileAt(tx, ty) {
	case sim.TileFord:
		return 1
	case sim.TileShallow:
		if d, ok := seaDepth(); ok {
			return 1 - 0.5*smooth(clamp01((d-0.42)/0.3))
		}
		return 0.85
	case sim.TileRiver:
		return byBank(0.15, 0.85)
	case sim.TileWater:
		if g.lakeAt(tx, ty) {
			return byBank(0.2, 1.4)
		}
		if d, ok := seaDepth(); ok {
			return 1 - smooth(clamp01((d-0.32)/0.33))
		}
		return byBank(0.3, 1.8) // a world without the sea's depths
	}
	return 1
}

// lakeAt reports whether tile (tx, ty) is a lake's or a pond's water.
func (g *Game) lakeAt(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= g.width || ty >= g.height {
		return false
	}
	i := ty*g.width + tx
	if g.relief != nil && i < len(g.relief) && g.relief[i].Variant() == sim.VarLake {
		return true
	}
	return g.pondMask != nil && i < len(g.pondMask) && g.pondMask[i]
}

// landDist is how far (x, y) tiles is from dry land, up to 3 tiles.
func (g *Game) landDist(x, y float64) float64 {
	best := 3.0
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - 3; ty <= cy+3; ty++ {
		for tx := cx - 3; tx <= cx+3; tx++ {
			if t := g.tileAt(tx, ty); t.Walkable() && t != sim.TileFord {
				dx := math.Max(0, math.Max(float64(tx)-x, x-float64(tx+1)))
				dy := math.Max(0, math.Max(float64(ty)-y, y-float64(ty+1)))
				best = math.Min(best, math.Hypot(dx, dy))
			}
		}
	}
	return best
}

// underShadow draws a soft dark round shadow of radii rx, ry about (x, y), alpha
// at its middle, fading to nothing at its edge.
func underShadow(dst drawTarget, x, y, rx, ry, alpha float64) {
	if alpha <= 0 {
		return
	}
	for yy := -ry; yy <= ry; yy++ {
		for xx := -rx; xx <= rx; xx++ {
			r := (xx*xx)/(rx*rx) + (yy*yy)/(ry*ry)
			if r >= 1 {
				continue
			}
			a := alpha * (1 - r) * (1 - r)
			fillRect(dst, float32(math.Round(x+xx)), float32(math.Round(y+yy)), 1, 1, premul(color.RGBA{8, 22, 40, 255}, a), false)
		}
	}
}
