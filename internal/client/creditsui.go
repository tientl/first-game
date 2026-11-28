package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The credits screen ("Ghi công"), opened from the main menu and from the
// pause menu. It is laid out like the settings screen: a framed panel (setW by
// setH UI pixels, shrunk to fit a small screen) with a title, a scrolling
// text of everything in credits.go, and "Quay lại" at the foot.
//
// Keys: W/S and the arrows scroll a line, PageUp/PageDown a page, Home and End
// go to the ends, Esc or Enter closes. The mouse wheel scrolls; so does
// dragging the text or the bar at its right.

const (
	crX, crY  = setPad, 64
	crW       = setW - 2*setPad - 18 // the text's width: a bar fits at its right
	crH       = setFootY - 10 - crY - 6
	crBarX    = setW - setPad - 8
	crLineGap = 3.0
	crNameSz  = 1.2 // an entry's name
	crInfoSz  = 1.0 // its author and licence
	crURLSz   = 0.92
)

// crKind is what a line of the text is.
type crKind uint8

const (
	crHead crKind = iota // a section's heading
	crName               // what is credited
	crInfo               // who and under what licence
	crURL                // where
	crNote               // a paragraph
)

// crLine is a line of the text and where it lies.
type crLine struct {
	s    string
	kind crKind
	y    float64 // from the text's top
	h    float64
}

// creditsLines lays the sections out in lines no wider than w.
func creditsLines(secs []creditSection, w float64) (lines []crLine, total float64) {
	y := 0.0
	add := func(s string, k crKind, h float64) {
		lines = append(lines, crLine{s, k, y, h})
		y += h
	}
	for si, sec := range secs {
		if si > 0 {
			y += 14
		}
		add(sec.Title, crHead, 34)
		if sec.Intro != "" {
			for _, l := range wrapWords(sec.Intro, w-6) {
				add(l, crNote, lineUI+2)
			}
			y += 6
		}
		for _, e := range sec.Entries {
			for _, l := range wrapWords(e.Name, (w-6)/crNameSz) {
				add(l, crName, lineUI*crNameSz+2)
			}
			info := e.By
			if e.Licence != "" {
				if info != "" {
					info += " · "
				}
				info += "Giấy phép: " + e.Licence
			}
			for _, l := range wrapWords(info, w-14) {
				add(l, crInfo, lineUI+crLineGap)
			}
			if e.URL != "" {
				add(e.URL, crURL, lineUI*crURLSz+crLineGap)
			}
			y += 9
		}
	}
	y += 6
	for _, l := range wrapWords(creditLicenceFile, w-6) {
		add(l, crNote, lineUI+2)
	}
	return lines, y + 8
}

type creditsScreen struct {
	lines  []crLine
	total  float64
	scroll float64
	frame  int

	drag            int // 0 none, 1 the text, 2 the bar
	dragY, dragBase float64

	// From the last draw, for the mouse.
	ox, oy, scale float64
	drawn         bool
}

func newCreditsScreen() *creditsScreen {
	s := &creditsScreen{}
	s.lines, s.total = creditsLines(creditSections(), crW-8)
	return s
}

// maxScroll is how far the text can go up.
func (s *creditsScreen) maxScroll() float64 { return max(0, s.total-crH) }

func (s *creditsScreen) scrollTo(v float64) { s.scroll = math.Max(0, math.Min(s.maxScroll(), v)) }

// creditsBackRect is where "Quay lại" lies, on the panel.
func creditsBackRect() (x, y, w, h float64) {
	return float64(setW-setPad) - setFootBtnW, setFootY, setFootBtnW, setFootH
}

// barThumb is the scroll bar's thumb: its top and height.
func (s *creditsScreen) barThumb() (y, h float64) {
	if s.total <= crH {
		return crY, crH
	}
	h = math.Max(34, crH*crH/s.total)
	return crY + (crH-h)*s.scroll/s.maxScroll(), h
}

// update handles a frame's input and reports whether the screen is done.
func (s *creditsScreen) update() bool {
	s.frame++
	if justPressed(ebiten.KeyEscape, ebiten.KeyEnter, ebiten.KeyNumpadEnter) && !altHeld() {
		snd.ui("ui_select")
		return true
	}
	const step = 36.0
	if justPressed(ebiten.KeyS, ebiten.KeyArrowDown) || repeating(ebiten.KeyS, ebiten.KeyArrowDown) {
		s.scrollTo(s.scroll + step)
	}
	if justPressed(ebiten.KeyW, ebiten.KeyArrowUp) || repeating(ebiten.KeyW, ebiten.KeyArrowUp) {
		s.scrollTo(s.scroll - step)
	}
	switch {
	case justPressed(ebiten.KeyPageDown, ebiten.KeySpace):
		s.scrollTo(s.scroll + crH - 40)
	case justPressed(ebiten.KeyPageUp):
		s.scrollTo(s.scroll - crH + 40)
	case justPressed(ebiten.KeyHome):
		s.scrollTo(0)
	case justPressed(ebiten.KeyEnd):
		s.scrollTo(s.maxScroll())
	}
	_, wy := ebiten.Wheel()
	s.scrollTo(s.scroll - wy*54)
	return s.mouse()
}

// repeating is whether one of keys is held long enough to repeat.
func repeating(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if d := inpututil.KeyPressDuration(k); d > 24 && d%4 == 0 {
			return true
		}
	}
	return false
}

func (s *creditsScreen) mouse() bool {
	if !s.drawn || s.scale <= 0 {
		return false
	}
	ux, uy := uiCursor()
	px, py := (float64(ux)-s.ox)/s.scale, (float64(uy)-s.oy)/s.scale
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	held := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in := func(x, y, w, h float64) bool { return px >= x && px < x+w && py >= y && py < y+h }

	if s.drag != 0 {
		if !held {
			s.drag = 0
			return false
		}
		switch s.drag {
		case 1: // the text follows the pointer
			s.scrollTo(s.dragBase - (py - s.dragY))
		case 2: // so does the thumb
			if _, th := s.barThumb(); crH > th {
				s.scrollTo(s.dragBase + (py-s.dragY)*s.maxScroll()/(crH-th))
			}
		}
		return false
	}
	switch {
	case click && in(creditsBackRect()):
		snd.ui("ui_select")
		return true
	case click && in(crBarX-6, crY, 20, crH):
		ty, th := s.barThumb()
		if py < ty || py > ty+th { // a click on the track puts the thumb there
			s.scrollTo((py - th/2 - crY) / math.Max(1, crH-th) * s.maxScroll())
		}
		s.drag, s.dragY, s.dragBase = 2, py, s.scroll
	case click && in(crX, crY, crW, crH):
		s.drag, s.dragY, s.dragBase = 1, py, s.scroll
	}
	return false
}

// draw draws the screen over whatever is behind it.
func (s *creditsScreen) draw(screen *uiDst) {
	uw, uh := screen.Bounds().Dx(), screen.Bounds().Dy()
	dimScreen(screen, color.RGBA{0, 0, 0, 120})
	s.scale, s.ox, s.oy = setFit(uw, uh)
	s.drawn = true
	b := screen.view(s.ox, s.oy, s.scale)
	drawFrame(b, 4, 4, setW-8, setH-8)
	uiTextW(b, "Ghi công", setPad+2, 18, colBoneText, text.AlignStart, 2.1, wTitle, fxShadow)
	fillRect(b, setPad, 58, setW-2*setPad, 1, color.RGBA{150, 112, 70, 150}, false)

	// The text, clipped to its box.
	c := b.clip(crX, crY, crW+10, crH)
	for _, l := range s.lines {
		y := crY + l.y - s.scroll
		if y+l.h < crY-2 || y > crY+crH+2 {
			continue
		}
		x := float64(crX)
		switch l.kind {
		case crHead:
			uiTextW(c, l.s, x, y+2, colAmber, text.AlignStart, 1.6, wTitle, fxShadow)
			fillRect(c, float32(x), float32(y+l.h-6), crW-8, 1, color.RGBA{150, 112, 70, 140}, false)
		case crName:
			uiText(c, l.s, x+6, y, colBoneText, text.AlignStart, crNameSz, fxShadow)
		case crInfo:
			uiText(c, l.s, x+14, y, colMuted, text.AlignStart, crInfoSz, fxPlain)
		case crURL:
			uiText(c, l.s, x+14, y, color.RGBA{150, 176, 196, 255}, text.AlignStart, crURLSz, fxPlain)
		case crNote:
			uiText(c, l.s, x+6, y, color.RGBA{214, 202, 176, 255}, text.AlignStart, 1, fxPlain)
		}
	}
	// A shade where the text runs off, so it reads as scrollable.
	if s.scroll > 1 {
		fillRect(b, crX, crY, crW, 3, color.RGBA{30, 20, 12, 90}, false)
	}
	if s.scroll < s.maxScroll()-1 {
		fillRect(b, crX, crY+crH-3, crW, 3, color.RGBA{30, 20, 12, 90}, false)
	}

	// The bar.
	if s.total > crH {
		well(b, crBarX, crY, 8, crH, 4)
		ty, th := s.barThumb()
		col := color.RGBA{190, 150, 92, 255}
		if s.drag == 2 {
			col = colAmber
		}
		plate(b, crBarX, ty, 8, th, 4)
		fillRect(b, crBarX+2, float32(ty+3), 4, float32(th-6), col, false)
	}

	// The foot.
	fillRect(b, setPad, setFootY-10, setW-2*setPad, 1, color.RGBA{150, 112, 70, 110}, false)
	drawText(b, "W/S, chuột cuộn hoặc kéo để xem tiếp", setPad+4, setFootY+(setFootH-lineUI)/2, colMuted, text.AlignStart, 1)
	x, y, w, h := creditsBackRect()
	tab(b, x, y, w, h, 10, tabOn)
	uiText(b, "Quay lại", x+w/2, y+(h-lineUI*1.3)/2, colBoneText, text.AlignCenter, 1.3, fxShadow)
	if hy := s.oy + setH*s.scale + 6; hy+lineUI < float64(uh) {
		drawText(screen, "W/S cuộn · PageUp/PageDown · Esc quay lại", float64(uw)/2, hy, colMuted, text.AlignCenter, 1)
	}
}
