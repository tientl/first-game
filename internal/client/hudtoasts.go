package client

import (
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// What the HUD says for a moment and then lets go: the notices, stacked at
// the top in the middle (three at most, the newest on top); the long line
// of a status just begun, under its chip; and the name of what is under
// the pointer on the HUD (a chip, the day's line). All small, on soft dark
// plates that read over bright sand and snow.

// The sizes of their text.
const (
	noticeScale      = 0.88
	statusToastScale = 0.8
	tipScale         = 0.8
)

// note is one notice on the screen.
type note struct {
	text  string
	born  time.Time
	until time.Time
}

// noteFX is the notices showing, newest first, and the game's notice last
// taken onto them (by its time).
type noteFX struct {
	list  []note
	taken time.Time
}

// noteFade is how long a notice takes to go.
const noteFade = 350 * time.Millisecond

// takeNotices puts a new notice of the game's on the stack (the same one
// again only keeps it longer), and lets go of those gone or pushed out.
func (g *Game) takeNotices(now time.Time) {
	fx := &g.hudFX.notes
	if g.notice != "" && !g.noticeUntil.Equal(fx.taken) {
		fx.taken = g.noticeUntil
		if len(fx.list) > 0 && fx.list[0].text == g.notice && now.Before(fx.list[0].until) {
			fx.list[0].until = g.noticeUntil
		} else {
			fx.list = append([]note{{text: g.notice, born: now, until: g.noticeUntil}}, fx.list...)
		}
	}
	kept := fx.list[:0]
	for _, n := range fx.list {
		if len(kept) < maxNotes && now.Before(n.until) {
			kept = append(kept, n)
		}
	}
	fx.list = kept
}

// drawNotice draws the notices, top middle under the boss's bar, if one is
// up: each slides in and fades away by itself.
func (g *Game) drawNotice(screen *uiDst, sw int) {
	now := time.Now()
	g.takeNotices(now)
	boss, holds := g.bossUp() // see bossui.go
	y := noticesTop(boss, holds)
	maxW := noticeMaxW(sw)
	cx := float64(sw) / 2
	lh := lineUI * noticeScale
	for i, n := range g.hudFX.notes.list {
		in := smooth(min(1, now.Sub(n.born).Seconds()/0.18))
		a := min(in, n.until.Sub(now).Seconds()/noteFade.Seconds())
		if i > 0 {
			a *= 0.88 // the older, a little fainter
		}
		lines := wrapAt(n.text, maxW-24, noticeScale)
		w := 0.0
		for _, l := range lines {
			w = max(w, textWidth(l, noticeScale))
		}
		w += 24
		h := math.Round(float64(len(lines))*lh + (noticeH - lh))
		yy := y - (1-in)*6
		if a > 0.01 {
			plateA(screen, cx-w/2, yy, w, h, min(h/2, noticeH/2), 0.9*a)
			c := fade(colBoneText, uint8(255*a))
			for j, l := range lines {
				uiText(screen, l, cx, yy+(noticeH-lh)/2+float64(j)*lh, c, text.AlignCenter, noticeScale, fxShadow)
			}
		}
		y += h + noticeGap
	}
}

// drawStatusToast draws the long line of status s, since frames after it
// began, under the chips with its left at x and its top at y: its picture
// and the line, wrapped to statusToastW, sliding in and fading out.
func (g *Game) drawStatusToast(dst *uiDst, s status, x, y float64, since int) {
	a := min(1, float64(since+3)/10, float64(statusShout-since)/40)
	if a <= 0.01 {
		return
	}
	const sc = statusToastScale
	lh := lineUI * sc
	lines := wrapAt(s.long, statusToastW-34, sc)
	w := 0.0
	for _, l := range lines {
		w = max(w, textWidth(l, sc))
	}
	w += 34
	h := math.Round(float64(len(lines))*lh + 6)
	x -= (1 - smooth(min(1, float64(since)/12))) * 8
	plateA(dst, x, y, w, h, min(h/2, 9), 0.92*a)
	drawUIIcon(dst, statusIcon(s.kind), x+12, y+min(h/2, 9), 13, float32(a))
	c := fade(mix(s.col, colWhite, 0.35), uint8(255*a))
	for j, l := range lines {
		uiText(dst, l, x+23, y+3+float64(j)*lh, c, text.AlignStart, sc, fxShadow)
	}
}

// hudTip is the name of the piece of the HUD under the pointer, said just
// under it: centred on x, its top at y.
type hudTip struct {
	s    string
	x, y float64
	ok   bool
}

// hovered reports whether the pointer is on the UI rectangle (x, y, w, h).
func (g *Game) hovered(x, y, w, h float64) bool {
	if g.overlay != overlayNone || g.paused {
		return false
	}
	mx, my := uiCursor()
	return float64(mx) >= x && float64(mx) < x+w && float64(my) >= y && float64(my) < y+h
}

// drawTip draws the tip, kept on the screen.
func (g *Game) drawTip(screen *uiDst, sw, sh int) {
	t := g.hudFX.tip
	if !t.ok || t.s == "" {
		return
	}
	const s = tipScale
	w := textWidth(t.s, s) + 14
	h := math.Round(lineUI*s + 5)
	x := max(4, min(float64(sw)-w-4, t.x-w/2))
	y := min(float64(sh)-h-4, t.y)
	plate(screen, x, y, w, h, h/2)
	uiText(screen, t.s, x+7, y+(h-lineUI*s)/2, colBoneText, text.AlignStart, s, fxShadow)
}

// wrapAt breaks s into lines no wider than w at scale.
func wrapAt(s string, w, scale float64) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		try := word
		if line != "" {
			try = line + " " + word
		}
		if line != "" && textWidth(try, scale) > w {
			lines = append(lines, line)
			line = word
			continue
		}
		line = try
	}
	if line != "" || len(lines) == 0 {
		lines = append(lines, line)
	}
	return lines
}

// ShowNotices puts notices on the screen at once, the last newest, as if
// they had come one after another (for the screenshot tool).
func (g *Game) ShowNotices(list ...string) {
	now := time.Now()
	for i, s := range list {
		born := now.Add(time.Duration(i-len(list)) * 400 * time.Millisecond)
		g.hudFX.notes.list = append([]note{{text: s, born: born, until: now.Add(time.Minute)}}, g.hudFX.notes.list...)
	}
}
