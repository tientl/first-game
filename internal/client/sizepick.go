package client

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/sim"
)

// The world's size, chosen on the difficulty picker beside "Quay lại": a
// button that steps through the sizes on a click or the C key. The first
// is what a new world is made at unless the player picks another.

// worldSizes are the sizes a new world can be made at, with their names.
var worldSizes = []struct {
	size int
	name string
}{{sim.SizeHuge, "Lớn"}, {sim.SizeColossal, "Khổng lồ"}}

// diffSizeRect is where the size button lies in the picker's panel: left of
// "Quay lại", on the same row.
func diffSizeRect() image.Rectangle {
	back := diffBackRect()
	return image.Rect(diffPad, back.Min.Y, back.Min.X-14, back.Max.Y)
}

// sizeKeys steps the size when C is pressed or the button is clicked at
// panel point (x, y), and reports whether it did.
func (p *diffPicker) sizeKeys(x, y float64, click bool) bool {
	hit := image.Pt(int(x), int(y)).In(diffSizeRect())
	if inpututil.IsKeyJustPressed(ebiten.KeyC) || click && hit {
		p.size = (p.size + 1) % len(worldSizes)
		snd.ui("ui_move")
		return true
	}
	return false
}

// worldSize is the size picked, in tiles.
func (p *diffPicker) worldSize() int { return worldSizes[p.size%len(worldSizes)].size }

// drawSize draws the size button on the panel.
func (p *diffPicker) drawSize(b *uiDst) {
	r := diffSizeRect()
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	tab(b, float64(x), float64(y), float64(w), float64(h), 10, tabIdle)
	s := worldSizes[p.size%len(worldSizes)]
	cx := float64(r.Min.X) + float64(r.Dx())/2
	drawText(b, "Cỡ thế giới (C)", cx, float64(r.Min.Y)+4, colMuted, text.AlignCenter, 1)
	drawText(b, fmt.Sprintf("< %s · %d >", s.name, s.size), cx, float64(r.Min.Y)+22, colKey, text.AlignCenter, 1)
}
