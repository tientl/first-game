package client

import (
	"bytes"
	_ "embed"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
)

// The interface's type: Nunito (fonts/, SIL Open Font License 1.1, see
// fonts/OFL.txt and CREDITS.md), a rounded sans with every Vietnamese
// letter, set from its variable font at the weight each use wants: bold
// for text, heavier for small counts, black for titles. Text is shaped
// (go-text), so stacked accents such as "Ự", "ẫ" or "ổ" sit where the type
// designer put them, also when a string spells them as combining marks.
//
// Text is set at its true size on the image it goes to: a line of UI text
// at 1080p is drawn 2 times bigger, not drawn small and blown up, so it is
// crisp at every interface scale. A string drawn with an effect (a soft
// shadow, a halo) is baked once into an image with the effect around it
// and kept while it is shown (textBakes), so a frame's text is one image
// draw per string; a bake not shown for a few seconds is let go.
//
// Sizes keep the bitmap font's layout: at scale 1 a line is lineUI (16) UI
// pixels tall, the text's top at the y it is drawn at, and textWidth gives
// widths in UI pixels, so boxes laid out for it still hold their text.

//go:embed fonts/Nunito-Variable.ttf
var nunitoTTF []byte

var (
	fontOnce sync.Once
	fontSrc  *text.GoTextFaceSource
	symSrc   *text.GoTextFaceSource // Go Bold (BSD licence), for the arrows Nunito lacks
)

// fontSource is the parsed font, read the first time it is wanted.
func fontSource() *text.GoTextFaceSource {
	fontOnce.Do(func() {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(nunitoTTF))
		if err != nil {
			panic("uifont: " + err.Error())
		}
		fontSrc = s
		if symSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
			panic("uifont: " + err.Error())
		}
	})
	return fontSrc
}

// The line of text at scale 1, in UI pixels.
const (
	lineUI = 16.0 // a line's height
	sizeUI = 12.5 // the type's size (em)
	baseUI = 12.3 // the baseline, down from the line's top
)

// fontWeight is a value of the font's weight axis (wght, 200 to 1000).
type fontWeight uint16

const (
	wText  fontWeight = 720 // text
	wHeavy fontWeight = 820 // small text that must still read: counts, badges
	wTitle fontWeight = 930 // titles
)

// weightFor is the weight of text drawn at scale.
func weightFor(scale float64) fontWeight {
	switch {
	case scale >= 2.4:
		return wTitle
	case scale >= 1.4:
		return 860
	}
	return wText
}

type faceKey struct {
	px float32
	w  fontWeight
}

var faces = map[faceKey]text.Face{}

// faceAt is the font at px pixels (an em) and weight w, with Go Bold behind
// it for the few signs Nunito has not got (arrows).
func faceAt(px float64, w fontWeight) text.Face {
	k := faceKey{float32(math.Round(px*8) / 8), w}
	if f := faces[k]; f != nil {
		return f
	}
	f := &text.GoTextFace{Source: fontSource(), Size: float64(k.px)}
	f.SetVariation(text.MustParseTag("wght"), float32(w))
	m, err := text.NewMultiFace(f, &text.GoTextFace{Source: symSrc, Size: float64(k.px) * 0.94})
	if err != nil {
		panic("uifont: " + err.Error())
	}
	faces[k] = m
	return m
}

// textFX is what is drawn around a string.
type textFX uint8

const (
	fxShadow  textFX = iota // a soft dark edge and a drop shadow: text on panels
	fxOutline               // a fuller dark halo: text over the world
	fxPlain                 // the letters alone: dark ink on parchment
)

type textKey struct {
	s  string
	px float32
	w  fontWeight
	fx textFX
}

// textBake is a string set in white with its effect in black, ready to be
// tinted: the pen starts at (ox, oy) on img (the line's left end, on its
// baseline) and moves adv.
type textBake struct {
	img    *ebiten.Image
	ox, oy float64
	adv    float64
	used   int
}

var textBakes = map[textKey]*textBake{}

// bakeText sets s at px pixels and weight w with effect fx.
func bakeText(s string, px float64, w fontWeight, fx textFX) *textBake {
	key := textKey{s, float32(px), w, fx}
	if b := textBakes[key]; b != nil {
		b.used = uiFrame
		return b
	}
	f := faceAt(px, w)
	m := f.Metrics()
	adv := text.Advance(s, f)
	r := max(0.8, px*0.055) // the halo's reach
	if fx == fxOutline {
		r = max(1.1, px*0.085)
	}
	pad := math.Ceil(2.4*r) + 1
	if fx == fxPlain {
		pad = 2
	}
	iw, ih := int(math.Ceil(adv+2*pad)), int(math.Ceil(m.HAscent+m.HDescent+2*pad))
	img := ebiten.NewImage(max(1, iw), max(1, ih))
	put := func(dx, dy float64, a float32) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(pad+dx, pad+dy)
		op.ColorScale.Scale(0, 0, 0, a) // black, premultiplied
		text.Draw(img, s, f, op)
	}
	ring := func(rad float64, n int, a float32) {
		for i := range n {
			t := float64(i) * 2 * math.Pi / float64(n)
			put(rad*math.Cos(t), rad*math.Sin(t), a)
		}
	}
	switch fx {
	case fxShadow:
		put(0, 1.3*r, 0.55) // the drop shadow
		ring(r, 8, 0.32)    // a soft dark edge
	case fxOutline:
		put(0.4*r, 1.5*r, 0.6)
		ring(2*r, 12, 0.12) // a wide faint halo
		ring(r, 12, 0.5)    // the dark edge
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(pad, pad)
	text.Draw(img, s, f, op) // the letters, white
	b := &textBake{img: img, ox: pad, oy: pad + m.HAscent, adv: adv, used: uiFrame}
	textBakes[key] = b
	artBytes.Add(int64(4 * iw * ih))
	return b
}

// sweepTextBakes lets go of the bakes not shown for a few seconds.
func sweepTextBakes() {
	if uiFrame%60 != 0 {
		return
	}
	for k, b := range textBakes {
		if uiFrame-b.used > 240 {
			b.img.Deallocate()
			artBytes.Add(-int64(4 * b.img.Bounds().Dx() * b.img.Bounds().Dy()))
			delete(textBakes, k)
		}
	}
	if len(textWidths) > 4096 {
		clear(textWidths)
	}
}

// uiText draws s on d with its line's top-left at (x, y) (the centre or the
// right end for AlignCenter and AlignEnd), at scale times the size of UI
// text, in colour clr with effect fx. It returns the text's width in UI
// pixels.
func uiText(d *uiDst, s string, x, y float64, clr color.Color, align text.Align, scale float64, fx textFX) float64 {
	return uiTextW(d, s, x, y, clr, align, scale, weightFor(scale), fx)
}

// uiTextW is uiText at weight w.
func uiTextW(d *uiDst, s string, x, y float64, clr color.Color, align text.Align, scale float64, w fontWeight, fx textFX) float64 {
	if s == "" || scale <= 0 {
		return 0
	}
	px := sizeUI * scale * d.k
	b := bakeText(s, px, w, fx)
	X, Y := d.X(x), d.Y(y)+baseUI*scale*d.k
	switch align {
	case text.AlignCenter:
		X -= b.adv / 2
	case text.AlignEnd:
		X -= b.adv
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(X)-b.ox, math.Round(Y)-b.oy)
	op.ColorScale.ScaleWithColor(clr)
	d.img.DrawImage(b.img, op)
	wUI := b.adv / d.k
	if uiProbe != nil {
		uiProbe.text(d, s, (math.Round(X)-d.ox)/d.k, y, wUI, lineUI*scale)
	}
	return wUI
}

// drawText draws s with a soft shadow. For AlignCenter and AlignEnd, x is
// the centre or the right edge.
func drawText(dst drawTarget, s string, x, y float64, clr color.Color, align text.Align, scale float64) {
	uiText(uiOf(dst), s, x, y, clr, align, scale, fxShadow)
}

// drawTextPlain draws s without a shadow.
func drawTextPlain(dst drawTarget, s string, x, y float64, clr color.Color, align text.Align) {
	uiText(uiOf(dst), s, x, y, clr, align, 1, fxPlain)
}

type widthKey struct {
	s     string
	scale float32
	w     fontWeight
	k     int
}

var textWidths = map[widthKey]float64{}

// textWidth is how wide s is at scale, in UI pixels.
func textWidth(s string, scale float64) float64 {
	return textWidthW(s, scale, weightFor(scale))
}

// textWidthW is how wide s is at scale and weight w, in UI pixels.
func textWidthW(s string, scale float64, w fontWeight) float64 {
	if s == "" {
		return 0
	}
	k := widthKey{s, float32(scale), w, uiScaleNow}
	if v, ok := textWidths[k]; ok {
		return v
	}
	// Measured at the size it is drawn at on the screen now: glyphs are
	// placed on whole pixels, so small text is a little wider than its
	// size alone says.
	u := float64(max(1, uiScaleNow))
	v := text.Advance(s, faceAt(sizeUI*scale*u, w)) / u
	textWidths[k] = v
	return v
}

// fitText is s cut short with "…" to fit w UI pixels at scale, if it must.
func fitText(s string, w, scale float64) string {
	if textWidth(s, scale) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 {
		r = r[:len(r)-1]
		t := string(r) + "…"
		if textWidth(t, scale) <= w {
			return t
		}
	}
	return "…"
}
