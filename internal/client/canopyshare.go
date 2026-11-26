package client

// The canopy measure: the share of the view that the drawings of tall
// things (trees and the like, taller than a player) cover, read from their
// cover masks (see occlude.go) as they are drawn. A dev measure for
// cmd/shot, to check how thick the woods look; off unless asked for.

var canopy struct {
	on    bool
	w, h  int // the view, in cells
	cells []bool
	last  float64
}

// canopyCell is how many art pixels a cell of the measure is across.
const canopyCell = 4

// SetMeasureCanopy turns the canopy measure on or off.
func SetMeasureCanopy(on bool) { canopy.on = on }

// CanopyShare is the share of the last frame's view the canopy covered.
func CanopyShare() float64 { return canopy.last }

// beginCanopy starts the measure of a view drawn on dst.
func beginCanopy(dst drawTarget) {
	if !canopy.on {
		return
	}
	b := dst.Bounds()
	canopy.w, canopy.h = b.Dx()/canopyCell, b.Dy()/canopyCell
	canopy.cells = make([]bool, canopy.w*canopy.h)
}

// noteCanopy counts the opaque part of s, drawn with its anchor at (x, y),
// if it stands taller than a player.
func noteCanopy(s sprite, x, y float64) {
	if !canopy.on || s.img == nil || s.ay < playerArtH*1.2 {
		return
	}
	v, ok := coverMasks.Load(s.img)
	if !ok {
		return
	}
	m := v.(*coverMask)
	k := float64(s.density())
	left, top := x-s.ax, y-s.ay
	for cy := max(0, int(top)/canopyCell); cy < canopy.h && float64(cy*canopyCell) < y; cy++ {
		for cx := max(0, int(left)/canopyCell); cx < canopy.w && float64(cx*canopyCell) < left+s.w(); cx++ {
			px, py := float64(cx*canopyCell)+canopyCell/2, float64(cy*canopyCell)+canopyCell/2
			if m.at((px-left)*k, (py-top)*k) {
				canopy.cells[cy*canopy.w+cx] = true
			}
		}
	}
}

// endCanopy works out the share of the view measured.
func endCanopy() {
	if !canopy.on || len(canopy.cells) == 0 {
		return
	}
	n := 0
	for _, c := range canopy.cells {
		if c {
			n++
		}
	}
	canopy.last = float64(n) / float64(len(canopy.cells))
}
