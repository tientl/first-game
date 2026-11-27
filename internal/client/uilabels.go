package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The labels floating over the world, drawn on the screen in screen pixels
// with the interface's look: one at a time, for what the pointer is on, on
// a soft dark plate that fades in as the pointer comes to a thing and out
// as it leaves; what is out of reach gets a small hint instead. What the
// pointer is on is marked by four thin corners that breathe and close in
// when it is first pointed at (drawCorners).

// labelKey says what a label is on.
type labelKey struct {
	kind uint8 // 1 an entity, 2 a creature, 3 a pile, 4 a far thing, 5 a ghost
	id   uint64
}

// floatLabel is one label to show this frame: its text and colour, and the
// point it stands on (screen pixels), its bottom middle.
type floatLabel struct {
	ok   bool
	key  labelKey
	text string
	col  color.RGBA
	x, y float64
	hint bool // a small hint (out of reach), not a name
}

// labelFX remembers the label shown, to fade it in and out, and the
// corners' target, to close them in.
type labelFX struct {
	key       labelKey
	since     int // the frame the label's thing was first pointed at
	last      floatLabel
	lastSeen  int
	cornerKey labelKey
	cornerAt  int
}

// labelFade is how many frames a label takes to come or go.
const labelFade = 8

// labelText is the size of a label's text, in screen pixels to a UI pixel.
func labelScale() float64 { return 0.9 * uiK() }

// showLabel draws this frame's label l, fading it in when it is new; with
// none, the last fades out.
func (g *Game) showLabel(screen *ebiten.Image, l floatLabel) {
	fx := &g.hudFX.labels
	now := g.frame
	if l.ok {
		if l.key != fx.key || now-fx.lastSeen > 2 {
			fx.key, fx.since = l.key, now
		}
		fx.last, fx.lastSeen = l, now
		a := min(1, float64(now-fx.since+1)/labelFade)
		drawFloatLabel(screen, l, smooth(a))
		return
	}
	if gone := now - fx.lastSeen; fx.last.ok && gone > 0 && gone < labelFade {
		drawFloatLabel(screen, fx.last, smooth(1-float64(gone)/labelFade))
	}
}

// drawFloatLabel draws l at alpha a: rising a little as it comes.
func drawFloatLabel(screen *ebiten.Image, l floatLabel, a float64) {
	if a <= 0.01 || l.text == "" {
		return
	}
	d := screenDst(screen)
	k := uiK()
	s := labelScale()
	if l.hint {
		s *= 0.88
	}
	w := textWidthW(l.text, s, wText+30)
	padX, h := 7*k, lineUI*s+5*k
	x := l.x - w/2 - padX
	y := l.y - h + (1-a)*4*k
	// Kept on the screen.
	sw := float64(screen.Bounds().Dx())
	x = max(4*k, min(x, sw-w-2*padX-4*k))
	y = max(4*k, y)
	if l.hint {
		plateA(d, x, y, w+2*padX, h, h/2, 0.55*a)
	} else {
		plateA(d, x, y, w+2*padX, h, h/2, 0.82*a)
	}
	c := fade(l.col, uint8(255*a))
	uiTextW(d, l.text, x+padX, y+2.5*k, c, text.AlignStart, s, wText+30, fxShadow)
}

// labelColor softens the interface's red for a label over the world.
func labelColor(c color.RGBA) color.RGBA {
	if c == colBad {
		return color.RGBA{255, 156, 136, 255}
	}
	return c
}

// nameTag draws a player's name over their head: small, on a soft plate.
func nameTag(screen *ebiten.Image, name string, x, y float64) {
	if name == "" {
		return
	}
	d := screenDst(screen)
	k, s := uiK(), labelScale()
	w := textWidthW(name, s, wText+30)
	h := lineUI*s + 4*k
	plateA(d, x-w/2-6*k, y-h, w+12*k, h, h/2, 0.6)
	uiTextW(d, name, x-w/2, y-h+2*k, colBoneText, text.AlignStart, s, wText+30, fxShadow)
}

// cornerAge is how many frames the corners have marked key, starting a
// new count when key changes.
func (g *Game) cornerAge(key labelKey) int {
	fx := &g.hudFX.labels
	if key != fx.cornerKey || g.frame-fx.cornerAt > 100000 {
		fx.cornerKey, fx.cornerAt = key, g.frame
	}
	return g.frame - fx.cornerAt
}

// drawCorners marks the box (x, y, w, h), in screen pixels, with four thin
// soft corners: they close in from a little way out over the first frames
// (age), then breathe.
func drawCorners(screen *ebiten.Image, x, y, w, h float64, col color.RGBA, age, frame int, strength float64) {
	k := uiK()
	in := smooth(min(1, float64(age)/10))
	out := (1-in)*10*k + (1+math.Sin(float64(frame)*0.09))*0.9*k
	x, y, w, h = x-out, y-out, w+2*out, h+2*out
	l := max(6*k, min(13*k, math.Min(w, h)/4))
	a := in * strength
	var p vector.Path
	corner := func(cx, cy, dx, dy float64) {
		p.MoveTo(float32(cx+dx*l), float32(cy))
		p.LineTo(float32(cx+dx*2*k), float32(cy))
		p.QuadTo(float32(cx), float32(cy), float32(cx), float32(cy+dy*2*k))
		p.LineTo(float32(cx), float32(cy+dy*l))
	}
	corner(x, y, 1, 1)
	corner(x+w, y, -1, 1)
	corner(x, y+h, 1, -1)
	corner(x+w, y+h, -1, -1)
	stroke := func(width float64, c color.RGBA) {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(c)
		vector.StrokePath(screen, &p, &vector.StrokeOptions{Width: float32(width), LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}, op)
	}
	stroke(4.2*k, fade(color.RGBA{0, 0, 0, 255}, uint8(70*a)))    // a soft dark edge
	stroke(2.6*k, fade(col, uint8(60*a)))                         // a glow
	stroke(max(1.2, 0.8*k), fade(lighten(col, 30), uint8(235*a))) // the line
}

// targetKey is what a label on tg is keyed by.
func targetKey(tg targetView) labelKey {
	switch {
	case tg.drop:
		return labelKey{3, uint64(tg.d.ID)}
	case tg.creature:
		return labelKey{2, uint64(tg.c.ID)}
	}
	return labelKey{1, uint64(tg.e.ID)}
}
