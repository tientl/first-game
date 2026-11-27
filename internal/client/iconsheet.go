package client

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/sim"
)

// IconSheet draws the HD item icons next to the old ones (shot -iconsheet):
// first in hotbar slots as a 1080p screen shows them (the interface at
// twice its size, the icons at the screen's own resolution), the old row
// over the new; then each icon three times bigger, old to the left. Like
// ScaleSheet it reads the image back, so it runs inside the game loop.
func (g *Game) IconSheet() image.Image {
	if g.art == nil {
		g.art = getArt()
	}
	items := iconSheetItems()
	const (
		perRow = 10
		zoom   = 3
		big    = 96 * zoom // a 1080p bag icon, zoomed
		pairW  = 2*big + 16
		cols   = 3
		pad    = 24
	)
	slotRows := (len(items) + perRow - 1) / perRow
	realH := 2 * (16 + 2*(20+slotRows*(hotPx+hotGap+8)+8))
	zoomRows := (len(items) + cols - 1) / cols
	w := max(cols*pairW+(cols+1)*pad, 2*(perRow*(hotPx+hotGap))+2*pad+120)
	sizesH := 0
	for _, n := range iconSheetSizes {
		sizesH += n + 16
	}
	w = max(w, 48+2*pad+len(items)*(iconSheetSizes[len(iconSheetSizes)-1]+12))
	h := realH + sizesH + 60 + zoomRows*(big+44) + pad
	h += h % 2
	sheet := ebiten.NewImage(w, h)
	sheet.Fill(color.RGBA{24, 26, 36, 255})

	// The slots, through the interface's own drawing at scale 2.
	keep, keepHD := uiScaleSetting, iconHD
	SetUIScale(2)
	ui, _, _ := beginUI(sheet)
	y := 16.0
	for _, hd := range []bool{false, true} {
		iconHD = hd
		label := "Cũ (16×16, phóng ×6)"
		if hd {
			label = "HD (1 điểm ảnh icon = 1 điểm ảnh màn hình)"
		}
		drawText(ui, label, pad/2, y, colKey, text.AlignStart, 1)
		y += 20
		for i, k := range items {
			x := float64(pad/2 + 60 + (i%perRow)*(hotPx+hotGap))
			yy := y + float64((i/perRow)*(hotPx+hotGap+8))
			st := slotPlain
			if i == 1 {
				st = slotHeld
			}
			g.drawSlotSized(ui, sim.Slot{Item: k, Count: 1 + (i*7)%13}, x, yy, hotPx, st)
		}
		y += float64(slotRows*(hotPx+hotGap+8)) + 8
	}
	endUI(ui)
	SetUIScale(keep)
	iconHD = keepHD

	// Each icon at the other sizes it is baked at: a recipe's ingredients
	// (32 at 1080p; 24 at a scale of 3), a 720p bag (48), a 1080p chest (64).
	slot := color.RGBA{34, 37, 50, 255}
	ys := float64(realH + 8)
	for _, n := range iconSheetSizes {
		drawText(sheet, fmt.Sprintf("%d", n), pad, ys+float64(n)/2-8, colMuted, text.AlignStart, 1)
		for i, k := range items {
			x := float64(pad + 48 + i*(n+12))
			fillRoundRect(sheet, float32(x)-4, float32(ys)-4, float32(n)+8, float32(n)+8, 6, slot)
			iconHD = true
			img := hdIcon(k, n)
			iconHD = keepHD
			if img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(x, ys)
				sheet.DrawImage(img, op)
			}
		}
		ys += float64(n + 16)
	}

	// Each icon, three times as big: old and new at a 1080p bag's size.
	y0 := float64(realH + sizesH + 40)
	drawText(sheet, "×3: cũ | HD, cỡ ô túi đồ ở 1080p (96 điểm ảnh)", pad, y0, colKey, text.AlignStart, 2)
	y0 += 44
	for i, k := range items {
		x := float64(pad + (i%cols)*(pairW+pad))
		yy := y0 + float64((i/cols)*(big+44))
		drawText(sheet, k.Def().Name, x, yy-6, colWhite, text.AlignStart, 2)
		yy += 22
		for j := range 2 {
			bx := x + float64(j*(big+16))
			fillRoundRect(sheet, float32(bx), float32(yy), big, big, 18, slot)
			op := &ebiten.DrawImageOptions{}
			var img *ebiten.Image
			if j == 0 {
				img = g.art.icons[k]
				op.GeoM.Scale(6*zoom, 6*zoom)
			} else {
				iconHD = true
				img = hdIcon(k, 96)
				iconHD = keepHD
				op.GeoM.Scale(zoom, zoom)
			}
			if img == nil {
				continue
			}
			op.GeoM.Translate(bx, yy)
			sheet.DrawImage(img, op)
		}
	}

	b := sheet.Bounds()
	img := image.NewRGBA(b)
	sheet.ReadPixels(img.Pix)
	sheet.Deallocate()
	return img
}

// iconSheetItems are the items the sheet shows: those with an HD icon, in
// the order of the pilot first.
func iconSheetItems() []sim.ItemKind {
	var out []sim.ItemKind
	seen := map[sim.ItemKind]bool{}
	for _, k := range iconPilot {
		if hdIcons[k] != nil && !seen[k] {
			out, seen[k] = append(out, k), true
		}
	}
	for k := sim.ItemKind(1); k < sim.NumItems; k++ {
		if hdIcons[k] != nil && !seen[k] {
			out, seen[k] = append(out, k), true
		}
	}
	return out
}

// iconSheetSizes are the smaller sizes the sheet shows each icon at (at
// 16 the old icon is drawn: see drawItem).
var iconSheetSizes = []int{24, 32, 48, 64}
