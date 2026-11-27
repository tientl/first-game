package client

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Shared pieces of the menus: the art (built once), and a vertical list of
// buttons driven by keyboard or mouse. The type is in uifont.go.

var sharedArt *art

// getArt builds the art on first use. It must run inside the game loop,
// where images can be created.
func getArt() *art {
	if sharedArt == nil {
		start := time.Now()
		sharedArt = newArt()
		artTime = time.Since(start) // see ArtCost in scalesheet.go
		sharedArt.warmCommon()      // the trees, drawn ahead; see lazyart.go
	}
	return sharedArt
}

type button struct {
	label    string
	sub      string // optional second line
	disabled bool
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
func (b *buttonList) update() int {
	prev := b.sel
	i := b.handle()
	if b.sel != prev {
		snd.ui("ui_move")
	}
	if i >= 0 {
		snd.ui("ui_select")
	}
	return i
}

func (b *buttonList) handle() int {
	b.fix()
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		b.move(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		b.move(-1)
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)) && !altHeld() { // Alt+Enter is fullscreen
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

func (b *buttonList) draw(dst *uiDst, x, y, w float64) {
	b.x, b.y, b.w, b.h, b.gap = x, y, w, 44, 10
	for i, it := range b.items {
		ry, rh := b.rowY(i), b.rowH(i)
		left := x - w/2
		sel := i == b.sel && !it.disabled
		st, fg := tabIdle, colBoneText
		switch {
		case it.disabled:
			st, fg = tabOff, color.RGBA{128, 116, 100, 255}
		case sel:
			st, fg = tabOn, color.RGBA{255, 240, 206, 255}
		}
		tab(dst, left, ry, w, rh, 12, st)
		label := it.label
		const ls = 1.55 // the label's size
		ly := ry + (b.h-lineUI*ls)/2
		uiText(dst, label, x, ly, fg, text.AlignCenter, ls, fxShadow)
		if it.sub != "" {
			sc := colMuted
			if it.disabled {
				sc = fg
			}
			drawText(dst, it.sub, x, ry+38, sc, text.AlignCenter, 1)
		}
		if sel { // a small amber point at the left
			cy := float32(ry + b.h/2)
			var p vector.Path
			p.MoveTo(float32(left)+15, cy-6)
			p.LineTo(float32(left)+22, cy)
			p.LineTo(float32(left)+15, cy+6)
			p.Close()
			op := &vector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(colAmber)
			fillPath(dst, &p, op)
		}
	}
}
