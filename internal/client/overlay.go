package client

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Overlays over the whole view: the night's dark (lighting.go), the light
// of the caves (cavedark.go), a flash of lightning.
//
// The world pass puts the view on the canvas on the camera's own grid,
// which can sit up to an art pixel off the screen's (see hd.go). An overlay
// exactly the size of the view then misses a strip a screen pixel or two
// wide along two edges of the screen, and at night the world shows through
// there undarkened: a pale, bluish line round the screen. So an overlay is
// drawn overlayMargin art pixels bigger than the view all round.

// overlayMargin is the margin round the view an overlay covers, in art pixels.
const overlayMargin = 1

// overlayTarget draws on an overlay image that has the margin round the
// view: what is drawn at (x, y) in the view lands at (x, y) plus the margin
// in the image. Its bounds are the view's. Draw on it with DrawImage,
// DrawRectShader or DrawTriangles; the vector shapes (fillRect and the
// like) do not know it.
type overlayTarget struct{ img *ebiten.Image }

func (o overlayTarget) Bounds() image.Rectangle {
	b := o.img.Bounds()
	return image.Rect(0, 0, b.Dx()-2*overlayMargin, b.Dy()-2*overlayMargin)
}

func (o overlayTarget) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	var p ebiten.DrawImageOptions
	if op != nil {
		p = *op
	}
	p.GeoM.Translate(overlayMargin, overlayMargin)
	o.img.DrawImage(img, &p)
}

func (o overlayTarget) DrawRectShader(w, h int, s *ebiten.Shader, op *ebiten.DrawRectShaderOptions) {
	var p ebiten.DrawRectShaderOptions
	if op != nil {
		p = *op
	}
	p.GeoM.Translate(overlayMargin, overlayMargin)
	o.img.DrawRectShader(w, h, s, &p)
}

func (o overlayTarget) DrawTriangles(vs []ebiten.Vertex, is []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	moved := make([]ebiten.Vertex, len(vs))
	for i, v := range vs {
		v.DstX += overlayMargin
		v.DstY += overlayMargin
		moved[i] = v
	}
	o.img.DrawTriangles(moved, is, img, op)
}

// drawOverlay puts an overlay made on an overlayTarget over the view on dst.
func drawOverlay(dst drawTarget, img *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-overlayMargin, -overlayMargin)
	dst.DrawImage(img, op)
}

// stretchOver sets m to stretch an image of bounds b, smoothly scaled, over
// a w by h screen so that the screen's edges fall on the centres of the
// image's edge pixels. Stretched edge to edge instead, the outer half of
// each edge pixel would be blended with nothing, and the vignette would
// fade out in a strip along the screen's edges.
func stretchOver(m *ebiten.GeoM, b image.Rectangle, w, h int) {
	kx, ky := float64(w)/float64(max(1, b.Dx()-1)), float64(h)/float64(max(1, b.Dy()-1))
	m.Translate(-0.5, -0.5)
	m.Scale(kx, ky)
}
