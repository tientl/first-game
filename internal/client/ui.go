package client

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Shared pieces of the menus: the font, the art (built once), and a vertical
// list of buttons driven by keyboard or mouse.

var uiFace = text.NewGoXFace(bitmapfont.Face)

var sharedArt *art

// getArt builds the art on first use. It must run inside the game loop,
// where images can be created.
func getArt() *art {
	if sharedArt == nil {
		start := time.Now()
		sharedArt = newArt()
		artTime = time.Since(start) // see ArtCost in scalesheet.go
	}
	return sharedArt
}

// Layout is the screen layout every scene uses: the window size in pixels.
func Layout(outsideWidth, outsideHeight int) (int, int) {
	return max(outsideWidth, 320), max(outsideHeight, 240)
}

// drawText draws s with a drop shadow. For AlignCenter and AlignEnd, x is
// the centre or the right edge.
func drawText(dst *ebiten.Image, s string, x, y float64, clr color.Color, align text.Align, scale float64) {
	for _, pass := range []struct {
		dx, dy float64
		c      color.Color
	}{{scale, scale, color.RGBA{0, 0, 0, 200}}, {0, 0, clr}} {
		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x+pass.dx, y+pass.dy)
		op.ColorScale.ScaleWithColor(pass.c)
		op.PrimaryAlign = align
		text.Draw(dst, s, uiFace, op)
	}
}

func textWidth(s string, scale float64) float64 {
	w, _ := text.Measure(s, uiFace, 0)
	return w * scale
}

type button struct {
	label    string
	sub      string // optional second line
	disabled bool
	volume   volumeKind // if set, the button is a volume control
}

type volumeKind uint8

const (
	volNone volumeKind = iota
	volMusic
	volEffects
)

func (k volumeKind) get() float64 { return snd.volume(k == volMusic) }

// set changes a volume and lets the player hear the new level.
func (k volumeKind) set(v float64) {
	snd.setVolume(k == volMusic, v)
	if k == volEffects {
		snd.play("ui_select", 0.5, 0, 1)
	}
}

// buttonList is a column of buttons centred on x.
type buttonList struct {
	items []button
	sel   int

	// Layout, set by draw and used for mouse hit-testing on the next update.
	x, y, w, h, gap float64
	lastMX, lastMY  int
}

func (b *buttonList) rowH(i int) float64 {
	if b.items[i].sub != "" {
		return b.h + 16
	}
	return b.h
}

func (b *buttonList) rowY(i int) float64 {
	y := b.y
	for j := 0; j < i; j++ {
		y += b.rowH(j) + b.gap
	}
	return y
}

func (b *buttonList) move(d int) {
	for range b.items {
		b.sel = (b.sel + d + len(b.items)) % len(b.items)
		if !b.items[b.sel].disabled {
			return
		}
	}
}

// fix moves the selection off a disabled button.
func (b *buttonList) fix() {
	if b.items[b.sel].disabled {
		b.move(1)
	}
}

// update handles input and returns the index of the chosen button, or -1.
// Volume buttons are handled here and never returned.
func (b *buttonList) update() int {
	prev := b.sel
	i := b.handle()
	if b.sel != prev {
		snd.ui("ui_move")
	}
	if i >= 0 && b.items[i].volume != volNone {
		k := b.items[i].volume
		v := k.get() + 0.2
		if v > 1.05 {
			v = 0
		}
		k.set(v)
		return -1
	}
	if i >= 0 {
		snd.ui("ui_select")
	}
	return i
}

func (b *buttonList) handle() int {
	b.fix()
	if k := b.items[b.sel].volume; k != volNone && snd != nil {
		left := inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA)
		right := inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD)
		switch {
		case left:
			k.set(k.get() - 0.1)
		case right:
			k.set(k.get() + 0.1)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		b.move(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		b.move(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		return b.sel
	}
	if b.w == 0 {
		return -1 // not drawn yet
	}
	mx, my := uiCursor()
	moved := mx != b.lastMX || my != b.lastMY
	b.lastMX, b.lastMY = mx, my
	fx, fy := float64(mx), float64(my)
	for i, it := range b.items {
		y := b.rowY(i)
		if it.disabled || fx < b.x-b.w/2 || fx > b.x+b.w/2 || fy < y || fy > y+b.rowH(i) {
			continue
		}
		if moved {
			b.sel = i // hovering selects, but a still mouse does not fight the keys
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			b.sel = i
			return i
		}
	}
	return -1
}

func (b *buttonList) draw(dst *ebiten.Image, x, y, w float64) {
	b.x, b.y, b.w, b.h, b.gap = x, y, w, 44, 10
	for i, it := range b.items {
		ry, rh := b.rowY(i), b.rowH(i)
		left := float32(x - w/2)
		bg, edge, fg := color.RGBA{14, 16, 26, 200}, color.RGBA{0, 0, 0, 90}, colWhite
		if i == b.sel && !it.disabled {
			bg, edge = color.RGBA{58, 46, 22, 235}, colKey
		}
		if it.disabled {
			bg, fg = color.RGBA{14, 16, 26, 140}, color.RGBA{110, 114, 126, 255}
		}
		fillRoundRect(dst, left-2, float32(ry)-2, float32(w)+4, float32(rh)+4, 10, edge)
		fillRoundRect(dst, left, float32(ry), float32(w), float32(rh), 8, bg)
		label := it.label
		if it.volume != volNone {
			pct := int(math.Round(it.volume.get() * 100))
			switch {
			case snd == nil:
				label += ": không có"
			case i == b.sel:
				label = fmt.Sprintf("< %s: %d%% >", label, pct)
			default:
				label = fmt.Sprintf("%s: %d%%", label, pct)
			}
		}
		drawText(dst, label, x, ry+10, fg, text.AlignCenter, 2)
		if it.sub != "" {
			sc := colMuted
			if it.disabled {
				sc = fg
			}
			drawText(dst, it.sub, x, ry+38, sc, text.AlignCenter, 1)
		}
		if i == b.sel && !it.disabled {
			drawText(dst, ">", float64(left)+16, ry+10, colKey, text.AlignStart, 2)
		}
	}
}
