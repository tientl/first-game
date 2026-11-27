package client

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The interface (HUD, bag, chest, crafting, map, menus) is laid out in UI
// pixels, for a screen about 540 to 720 UI pixels tall: at 1080p a UI pixel
// is 2 screen pixels, at 1440p 2.5, at 4K 4 (the scale need not be whole,
// and the player can choose another in the settings: autoUIScale,
// uiScaleFor). The layout, the slot sizes and the mouse
// (uiCursor) all stay in UI pixels.
//
// It is drawn straight on the screen at the screen's own resolution, not on
// a small canvas blown up: beginUI hands out a uiDst, the screen seen in UI
// pixels, and everything drawn on it (panels, bars, icons, text in a vector
// font: uifont.go) is drawn k times finer, so a line can be one screen
// pixel thin (hair) and an edge is smooth at any scale. A uiDst is a
// drawTarget (hd.go), so the shapes of the world code (fillRect,
// strokeLine, fillCircle...) and sprites (drawSprite, drawAt) draw on it
// too, in UI pixels.
//
// World labels are drawn on the screen in screen pixels (screenDst), their
// text grown by uiK, the same factor.

// uiScaleSetting is the scale asked for, in screen pixels to a UI pixel
// (1.5 is 150%): 0 is automatic.
var uiScaleSetting float64

// uiScaleNow is the scale of the frame last drawn, for the mouse.
var uiScaleNow = 1.0

// SetUIScale sets the interface's scale for this session, without saving
// it: 1, 1.5, 2… screen pixels to a UI pixel, or 0 for automatic (by the
// screen's height).
func SetUIScale(s float64) {
	OverrideSettings(func(st *Settings) { st.UIScale = int(math.Round(max(0, min(8, s)) * 100)) })
}

// UIScale is the scale asked for: 0 is automatic.
func UIScale() float64 { return uiScaleSetting }

// uiScaleStep is what the automatic scale is a whole number of: a quarter
// of a screen pixel. Panels, bars and icons are painted at the size they
// land at on the screen and text is set at its true size (uifont.go), so a
// fractional scale is as sharp as a whole one; panels' edges and text are
// put on whole screen pixels.
const uiScaleStep = 0.25

// autoUIScale is the automatic scale on a screen sh pixels tall: the
// screen's height over 540, rounded down to a quarter, at least 1, so the
// interface is 540 to about 600 UI pixels tall at any resolution, the
// height it is laid out for: 1.25 at 720p, 1.5 at 900p, 2 at 1080p, 2.5 at
// 1440p, 4 at 4K.
func autoUIScale(sh int) float64 {
	return max(1, math.Floor(float64(sh)/540/uiScaleStep+1e-9)*uiScaleStep)
}

// maxUIScale is the biggest scale a screen sh pixels tall takes: the
// interface is never shorter than 480 UI pixels.
func maxUIScale(sh int) float64 { return max(1, float64(sh)/480) }

// uiScaleFor is the interface's scale on a screen sh pixels tall: the
// scale chosen in the settings, as far as maxUIScale lets it, or else the
// automatic one.
func uiScaleFor(sh int) float64 {
	if uiScaleSetting > 0 {
		return max(1, min(uiScaleSetting, maxUIScale(sh)))
	}
	return autoUIScale(sh)
}

// uiDst is an image seen in UI pixels: UI point (x, y) is image pixel
// (ox + k*x, oy + k*y). Bounds is in UI pixels.
type uiDst struct {
	img    *ebiten.Image
	k      float64
	ox, oy float64
}

// uiFrame counts the interface's passes, for the caches of uifont.go and
// uipanel.go.
var uiFrame int

// uiSize is the size in UI pixels of a screen sw by sh at scale k: every
// screen pixel is under the interface.
func uiSize(sw, sh int, k float64) (uw, uh int) {
	return int(math.Ceil(float64(sw)/k - 1e-9)), int(math.Ceil(float64(sh)/k - 1e-9))
}

// beginUI starts drawing the interface over screen. It returns the screen
// seen in UI pixels and its size in UI pixels; endUI ends the pass.
func beginUI(screen *ebiten.Image) (ui *uiDst, uw, uh int) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	k := uiScaleFor(sh)
	uiScaleNow = k
	uiFrame++
	uw, uh = uiSize(sw, sh, k)
	return &uiDst{img: screen, k: k}, uw, uh
}

// endUI ends a pass of the interface: what it baked and did not show for a
// while is let go.
func endUI(ui *uiDst) {
	sweepTextBakes()
	sweepPanelBakes()
	sweepPieceBakes()
}

// screenDst is the screen seen in its own pixels: for what is drawn in
// screen pixels with the interface's look, such as world labels.
func screenDst(screen *ebiten.Image) *uiDst { return &uiDst{img: screen, k: 1} }

// uiOf is dst seen as an interface target: itself, or a plain image in its
// own pixels.
func uiOf(dst drawTarget) *uiDst {
	switch d := dst.(type) {
	case *uiDst:
		return d
	case *ebiten.Image:
		return &uiDst{img: d, k: 1}
	}
	panic("uiOf: not an image")
}

// X and Y put a UI coordinate on the image.
func (d *uiDst) X(x float64) float64 { return d.ox + x*d.k }
func (d *uiDst) Y(y float64) float64 { return d.oy + y*d.k }

// hair is one image pixel, in UI pixels: the thinnest line.
func (d *uiDst) hair() float64 { return 1 / d.k }

// snap puts v, in UI pixels, on a whole image pixel.
func (d *uiDst) snap(v float64) float64 { return math.Round(v*d.k) / d.k }

func (d *uiDst) Bounds() image.Rectangle {
	b := d.img.Bounds()
	return image.Rect(
		int(math.Floor((float64(b.Min.X)-d.ox)/d.k)), int(math.Floor((float64(b.Min.Y)-d.oy)/d.k)),
		int(math.Ceil((float64(b.Max.X)-d.ox)/d.k)), int(math.Ceil((float64(b.Max.Y)-d.oy)/d.k)))
}

// geoM is the UI-to-image transform.
func (d *uiDst) geoM() ebiten.GeoM {
	var m ebiten.GeoM
	m.Scale(d.k, d.k)
	m.Translate(d.ox, d.oy)
	return m
}

// DrawImage draws img placed by op in UI pixels. An image that lands a
// whole number of image pixels to each of its own, unturned (an old 16x16
// icon, a canvas made at the screen's size), keeps its pixels crisp; the
// rest is filtered smoothly.
func (d *uiDst) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	o := *op
	o.GeoM.Concat(d.geoM())
	pickFilter(&o, op.Filter)
	d.img.DrawImage(img, &o)
}

// drawFine draws img placed by op, which already includes an image's own
// density (drawImageD), keeping it crisp when it lands one to one.
func (d *uiDst) drawFine(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	o := *op
	o.GeoM.Concat(d.geoM())
	pickFilter(&o, ebiten.FilterNearest)
	d.img.DrawImage(img, &o)
}

func (d *uiDst) DrawRectShader(w, h int, s *ebiten.Shader, op *ebiten.DrawRectShaderOptions) {
	o := *op
	o.GeoM.Concat(d.geoM())
	d.img.DrawRectShader(w, h, s, &o)
}

func (d *uiDst) DrawTriangles(vs []ebiten.Vertex, is []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	moved := make([]ebiten.Vertex, len(vs))
	for i, v := range vs {
		v.DstX, v.DstY = float32(d.X(float64(v.DstX))), float32(d.Y(float64(v.DstY)))
		moved[i] = v
	}
	d.img.DrawTriangles(moved, is, img, op)
}

// clip is d cut to the UI rectangle (x, y, w, h): what is drawn on it
// outside that is lost.
func (d *uiDst) clip(x, y, w, h float64) *uiDst {
	r := image.Rect(int(math.Round(d.X(x))), int(math.Round(d.Y(y))), int(math.Round(d.X(x+w))), int(math.Round(d.Y(y+h))))
	sub, _ := d.img.SubImage(r.Intersect(d.img.Bounds())).(*ebiten.Image)
	return &uiDst{img: sub, k: d.k, ox: d.ox, oy: d.oy}
}

// view is d with its UI point (x, y) as the origin and its UI pixels s
// times smaller: for a panel laid out at its own size and shrunk to fit.
func (d *uiDst) view(x, y, s float64) *uiDst {
	return &uiDst{img: d.img, k: d.k * s, ox: d.X(x), oy: d.Y(y)}
}

// uiPoint maps a screen point to the UI canvas at scale s.
func uiPoint(mx, my int, s float64) (int, int) {
	s = max(1, s)
	// By the pixel's centre, so at a fractional scale a screen pixel goes to
	// the UI pixel most of it lies in.
	return int(math.Floor((float64(mx) + 0.5) / s)), int(math.Floor((float64(my) + 0.5) / s))
}

// uiCursor is the mouse pointer on the UI canvas.
func uiCursor() (int, int) {
	mx, my := ebiten.CursorPosition()
	return uiPoint(mx, my, uiScaleNow)
}

// uiK is the factor world labels' text grows by.
func uiK() float64 { return uiScaleNow }
