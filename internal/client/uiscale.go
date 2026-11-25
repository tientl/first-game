package client

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// The interface (HUD, bag, chest, crafting, map, menus) is laid out for a
// screen about 540 to 720 pixels tall. On a bigger screen it is drawn on a
// UI canvas a whole number of times smaller than the screen and blown up
// with crisp pixels: at 1080p the canvas is 960×540 and every UI pixel is
// 2×2 on screen. The mouse is mapped back the same way (uiCursor), so every
// click lands where the drawing is. World labels stay in world space but
// their text grows by the same factor.

// uiScaleSetting is the scale asked for: 0 is automatic.
var uiScaleSetting int

// uiScaleNow is the scale of the frame last drawn, for the mouse.
var uiScaleNow = 1

// uiLayer is the canvas the interface is drawn on when the scale is not 1.
var uiLayer *ebiten.Image

// SetUIScale sets the interface's scale: 1, 2, 3… pixels per UI pixel, or
// 0 for automatic (by the screen's height).
func SetUIScale(s int) { uiScaleSetting = max(0, min(8, s)) }

// UIScale is the scale asked for: 0 is automatic.
func UIScale() int { return uiScaleSetting }

// uiScaleFor is the interface's scale on a screen sh pixels tall: automatic
// is one UI pixel per screen pixel up to 1079 pixels, 2 at 1080p and 1440p,
// 4 at 4K: the screen's height over 540, at least 1. A chosen scale never
// leaves the canvas shorter than 480 UI pixels.
func uiScaleFor(sh int) int {
	s := max(1, sh/540)
	if uiScaleSetting > 0 {
		s = max(1, min(uiScaleSetting, sh/480))
	}
	return s
}

// beginUI starts drawing the interface over screen. It returns the canvas
// to draw on and its size in UI pixels; endUI puts it on the screen.
func beginUI(screen *ebiten.Image) (ui *ebiten.Image, uw, uh int) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	s := uiScaleFor(sh)
	uiScaleNow = s
	if s == 1 {
		return screen, sw, sh
	}
	uw, uh = (sw+s-1)/s, (sh+s-1)/s
	if uiLayer == nil || uiLayer.Bounds().Dx() != uw || uiLayer.Bounds().Dy() != uh {
		if uiLayer != nil {
			uiLayer.Deallocate()
		}
		uiLayer = ebiten.NewImage(uw, uh)
	}
	uiLayer.Clear()
	return uiLayer, uw, uh
}

// endUI draws the interface canvas from beginUI on the screen.
func endUI(screen, ui *ebiten.Image) {
	if ui == screen {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(uiScaleNow), float64(uiScaleNow))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(ui, op)
}

// uiPoint maps a screen point to the UI canvas at scale s.
func uiPoint(mx, my, s int) (int, int) {
	s = max(1, s)
	return floorDiv(mx, s), floorDiv(my, s)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// uiCursor is the mouse pointer on the UI canvas.
func uiCursor() (int, int) {
	mx, my := ebiten.CursorPosition()
	return uiPoint(mx, my, uiScaleNow)
}

// uiK is the factor world labels' text grows by.
func uiK() float64 { return float64(uiScaleNow) }
