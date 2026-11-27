package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The interface's pieces, each in the look of uipanel.go: panels, frames,
// plates, tabs, wells, glows, bars, and the inks for text on parchment.
// Every piece that holds text tells uiProbe its box (uiprobe.go).

// Warm colours of the interface.
var (
	colAmber     = color.RGBA{242, 182, 78, 255} // the selection's rim
	colAmberGlow = color.RGBA{255, 196, 92, 255} // its glow
	colHideDark  = color.RGBA{24, 19, 15, 255}   // under everything
	colInkText   = color.RGBA{58, 38, 22, 255}   // ink on parchment
	colInkMuted  = color.RGBA{112, 82, 54, 255}
)

// panel draws the dark hide panel the HUD's pieces sit on.
func panel(dst *uiDst, x, y, w, h float32) {
	X, Y, W, H := float64(x), float64(y), float64(w), float64(h)
	drawPanel(dst, psHide, X, Y, W, H, 9)
	noteBox(dst, X, Y, W, H)
}

// drawFrame is the frame of the bag, the hotbar and the windows: a wooden
// rim with bone studs at the corners round dark hide.
func drawFrame(dst *uiDst, x, y, w, h float32) {
	X, Y, W, H := float64(x), float64(y), float64(w), float64(h)
	drawPanel(dst, psWood, X-3, Y-3, W+6, H+6, 12)
	noteBox(dst, X, Y, W, H)
}

// plate is a soft dark plate: a hint, a name, a label.
func plate(dst *uiDst, x, y, w, h, r float64) {
	drawPanelLook(dst, psPlate, panelLooks[psPlate], x, y, w, h, r, 0.35)
	noteBox(dst, x, y, w, h)
}

// plateA is plate at alpha a, for one that fades in or out.
func plateA(dst *uiDst, x, y, w, h, r, a float64) {
	if a <= 0.01 {
		return
	}
	drawPanelAlpha(dst, psPlate, panelLooks[psPlate], x, y, w, h, r, 0.35, a)
	noteBox(dst, x, y, w, h)
}

// parchment is a note of warm parchment; text on it is inked (ink).
func parchment(dst *uiDst, x, y, w, h float64) {
	drawPanel(dst, psParchment, x, y, w, h, 6)
	noteBox(dst, x, y, w, h)
}

// tabState is how a tab or button shows.
type tabState uint8

const (
	tabIdle tabState = iota
	tabHover
	tabOn // selected or chosen
	tabOff
)

// tab draws a small leather tab: a button, a card, a choice.
func tab(dst *uiDst, x, y, w, h, r float64, st tabState) {
	l := panelLooks[psLeather]
	switch st {
	case tabHover:
		l.body = color.RGBA{78, 58, 40, 245}
		l.rim = color.RGBA{150, 112, 70, 255}
	case tabOn:
		glow(dst, x, y, w, h, r, colAmberGlow, 0.32)
		l.body = color.RGBA{92, 64, 34, 250}
		l.rim = colAmber
		l.stitch = color.RGBA{255, 222, 150, 110}
	case tabOff:
		l.body = color.RGBA{40, 33, 28, 200}
		l.rim = color.RGBA{70, 58, 48, 220}
		l.stitch = color.RGBA{}
	}
	drawPanelLook(dst, psLeather, l, x, y, w, h, r, 0.45)
	noteBox(dst, x, y, w, h)
}

// well draws a sunken well: a slot, a bar's track.
func well(dst *uiDst, x, y, w, h, r float64) {
	drawPanelLook(dst, psWell, panelLooks[psWell], x, y, w, h, r, 0)
}

// glow lays a soft glow of colour c round the rectangle, strength 0 to 1.
func glow(dst *uiDst, x, y, w, h, r float64, c color.RGBA, strength float64) {
	if strength <= 0.01 {
		return
	}
	l := panelLook{body: color.RGBA{c.R, c.G, c.B, 255}}
	drawPanelAlpha(dst, psGlow, l, x, y, w, h, r, 0, strength)
}

// inkOf is the colour to ink text of colour c on parchment: the interface's
// light colours turned dark.
func inkOf(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	k := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	switch k {
	case colWhite:
		return colInkText
	case colMuted:
		return colInkMuted
	case colGood:
		return color.RGBA{46, 102, 34, 255}
	case colBad, colGhostBad:
		return color.RGBA{150, 34, 26, 255}
	case colKey, colGold:
		return color.RGBA{140, 74, 10, 255}
	case colHunger:
		return color.RGBA{150, 70, 10, 255}
	}
	// Anything else: much darker, keeping its hue.
	return color.RGBA{k.R / 3, k.G / 3, k.B / 3, k.A}
}

// ink draws s in ink on parchment.
func ink(dst *uiDst, s string, x, y float64, c color.Color, align text.Align, scale float64) float64 {
	return uiTextW(dst, s, x, y, inkOf(c), align, scale, weightFor(scale)+40, fxPlain)
}

// --- bars ------------------------------------------------------------------------------

const barSrc = `//kage:unit pixels

package main

var Size vec2
var Radius float
var K float
var Fill float  // the share full now
var Trail float // the share it was: what was lost drains away
var Flash float // a blow just landed: the bar flashes
var Col vec4
var Time float

func sdBox(p, b vec2, r float) float {
	q := abs(p) - b + r
	return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - r
}

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := srcPos // over the rectangle drawn, from (0, 0)
	half := Size * 0.5
	d := sdBox(pos-half, half, min(Radius, half.y))
	if d > 0.5 {
		return vec4(0.0)
	}
	cover := clamp(0.5-d, 0.0, 1.0)
	v := pos.y / Size.y
	// The end of the fill wavers a little, like something alive.
	wave := sin(pos.y/(2.2*K)+Time*3.0)*0.8*K + sin(pos.y/(1.1*K)-Time*5.0)*0.35*K
	edge := Fill*Size.x + wave
	trail := Trail*Size.x + wave
	if pos.x > max(edge, trail) {
		return vec4(0.0)
	}
	c := Col.rgb
	// Rounded across: lit on top, deep below.
	c *= 1.18 - 0.5*v
	c += vec3(0.32) * (1.0 - smoothstep(0.08, 0.32, v)) * 0.6
	c *= 1.0 - 0.35*smoothstep(0.75, 1.0, v)
	// A slow sheen running along it, and a fine grain.
	sheen := pow(max(0.0, sin((pos.x-pos.y*0.6)/(18.0*K)-Time*1.3)), 12.0)
	c += vec3(0.16) * sheen * (1.0 - v)
	c *= 0.96 + 0.08*hash(floor(pos/max(1.0, K*0.75)))
	a := 1.0
	if pos.x > edge {
		// Lost a moment ago: pale, draining away.
		c = mix(Col.rgb, vec3(1.0, 0.94, 0.82), 0.62) * (1.05 - 0.3*v)
		a = 0.85
	} else {
		// A bright lip at the end of the fill.
		c += vec3(0.25) * (1.0 - smoothstep(0.0, 1.6*K, edge-pos.x))
	}
	c = mix(c, vec3(1.0, 0.98, 0.92), Flash*0.75)
	a *= cover
	return vec4(c*a, a)
}
`

var barShader *ebiten.Shader

// barFX is what a bar shows beyond its value: the share it had a moment
// ago (draining away), a flash for a blow just taken.
type barFX struct {
	trail float64
	flash float64
}

// drawBarFill fills the UI rectangle (x, y, w, h) as a bar frac full, in
// colour c, with fx.
func drawBarFill(d *uiDst, x, y, w, h, frac float64, c color.RGBA, fx barFX, frame int) {
	if barShader == nil {
		s, err := ebiten.NewShader([]byte(barSrc))
		if err != nil {
			panic("uitheme: " + err.Error())
		}
		barShader = s
	}
	X0, Y0 := math.Round(d.X(x)), math.Round(d.Y(y))
	pw, ph := int(math.Round(d.X(x+w))-X0), int(math.Round(d.Y(y+h))-Y0)
	if pw <= 0 || ph <= 0 || (frac <= 0 && fx.trail <= 0) {
		return
	}
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(X0, Y0)
	op.Uniforms = map[string]any{
		"Size":   []float32{float32(pw), float32(ph)},
		"Radius": float32(h * d.k / 2),
		"K":      float32(d.k),
		"Fill":   float32(frac),
		"Trail":  float32(max(frac, fx.trail)),
		"Flash":  float32(fx.flash),
		"Col":    []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, 1},
		"Time":   float32(float64(frame) / 60),
	}
	d.img.DrawRectShader(pw, ph, barShader, op)
}

// barTrail follows a bar's share: it holds a moment after a loss, then
// drains down to the share; a gain it takes at once. hit is when the last
// loss came, frame now.
type barTrail struct {
	shown float64
	last  float64
	hitAt int
	init  bool
}

// step moves the trail on to share f at frame now and gives what to draw.
func (t *barTrail) step(f float64, now int) barFX {
	if !t.init {
		t.shown, t.last, t.init = f, f, true
	}
	if f < t.last-1e-9 {
		t.hitAt = now
	}
	t.last = f
	if t.shown < f {
		t.shown = f
	}
	since := now - t.hitAt
	if since < 0 || since > 24 {
		t.shown += (f - t.shown) * 0.08
		if t.shown-f < 0.002 {
			t.shown = f
		}
	}
	flash := 0.0
	if since >= 0 && since < 10 {
		flash = 1 - float64(since)/10
	}
	return barFX{trail: t.shown, flash: flash}
}

// drawPointer draws a small rounded triangle pointing left (dir -1) or
// right (1), its middle at (x, y), size UI pixels across.
func drawPointer(d *uiDst, x, y, dir, size float64, c color.RGBA) {
	var p vector.Path
	p.MoveTo(float32(x+dir*size*0.55), float32(y))
	p.LineTo(float32(x-dir*size*0.45), float32(y-size*0.62))
	p.LineTo(float32(x-dir*size*0.45), float32(y+size*0.62))
	p.Close()
	var sh vector.Path
	sh.AddPath(&p, &vector.AddPathOptions{GeoM: func() ebiten.GeoM { var m ebiten.GeoM; m.Translate(0, 1); return m }()})
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 120})
	fillPath(d, &sh, op)
	op = &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	fillPath(d, &p, op)
}

// dimScreen darkens all of d with c: what was drawn is now under it, and no
// longer holds text for uiProbe.
func dimScreen(d *uiDst, c color.RGBA) {
	b := d.Bounds()
	fillRect(d, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), c, false)
	if uiProbe != nil {
		uiProbe.texts, uiProbe.boxes = uiProbe.texts[:0], uiProbe.boxes[:0]
	}
}
