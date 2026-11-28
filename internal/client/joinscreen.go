package client

import (
	"image/color"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// joinScreen is the first step of "Vào phòng": the address of the room, as
// the host read it off their screen ("192.168.1.5", or with a port,
// "100.101.102.103:7777"). The character screen follows.
type joinScreen struct {
	addr   []rune
	frame  int
	chars  []rune
	status string // why the last try failed, shown under the field

	next, back box // the buttons, from the last draw
}

// maxAddrRunes bounds what can be typed: a host name or an IPv6 address with a port.
const maxAddrRunes = 64

type joinResult uint8

const (
	joinNone joinResult = iota
	joinNext            // the address is typed: on to the character
	joinBack
)

func (j *joinScreen) text() string { return strings.TrimSpace(string(j.addr)) }

func (j *joinScreen) set(s string) {
	j.addr = j.addr[:0]
	for _, r := range s {
		j.addr = typeAddrRune(j.addr, r)
	}
}

// typeAddrRune adds r to an address being typed if an address can have it.
func typeAddrRune(a []rune, r rune) []rune {
	ok := r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(".:-_[]", r))
	if !ok || len(a) >= maxAddrRunes {
		return a
	}
	return append(a, r)
}

func (j *joinScreen) update() joinResult {
	j.frame++
	if justPressed(ebiten.KeyEscape) {
		snd.ui("ui_move")
		return joinBack
	}
	if d := inpututil.KeyPressDuration(ebiten.KeyBackspace); (d == 1 || d > 30 && d%3 == 0) && len(j.addr) > 0 {
		j.addr = j.addr[:len(j.addr)-1]
	}
	j.chars = ebiten.AppendInputChars(j.chars[:0])
	for _, r := range j.chars {
		j.addr = typeAddrRune(j.addr, r)
	}
	if justPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter) && j.text() != "" {
		snd.ui("ui_select")
		return joinNext
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := uiCursor()
		switch {
		case j.next.has(float64(mx), float64(my)) && j.text() != "":
			snd.ui("ui_select")
			return joinNext
		case j.back.has(float64(mx), float64(my)):
			snd.ui("ui_move")
			return joinBack
		}
	}
	return joinNone
}

func (j *joinScreen) draw(dst *uiDst) {
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	fillRect(dst, 0, 0, float32(sw), float32(sh), color.RGBA{6, 8, 16, 150}, false)
	cx := sw / 2
	top := max(16, sh*0.2-30)
	drawTitle(dst, "Vào phòng", cx, top, 3, color.RGBA{250, 214, 110, 255}, color.RGBA{60, 30, 10, 255})
	drawText(dst, "Gõ địa chỉ chủ phòng đọc cho bạn", cx, top+64, colMuted, text.AlignCenter, 1)

	pw, rowH := 520.0, 48.0
	x, y := cx-pw/2, top+96
	tab(dst, x, y, pw, rowH, 12, tabOn)
	ty := y + (rowH-lineUI*ccText)/2
	uiText(dst, "Địa chỉ", x+18, ty, colMuted, text.AlignStart, ccText, fxShadow)
	bx, bw := x+120, pw-120-10
	well(dst, bx, y+6, bw, rowH-12, 7)
	s := string(j.addr)
	if s == "" {
		uiText(dst, "192.168.1.5", bx+10, ty, color.RGBA{130, 118, 102, 255}, text.AlignStart, ccText, fxPlain)
	} else {
		uiText(dst, s, bx+10, ty, colWhite, text.AlignStart, ccText, fxShadow)
	}
	if (j.frame/30)%2 == 0 {
		fillRect(dst, float32(bx+10+textWidth(s, ccText)+2), float32(y+11), 2, float32(rowH-22), colKey, false)
	}

	drawText(dst, "Không gõ cổng thì dùng cổng 7777 · cùng nhà: IP trong mạng nhà · qua Internet: IP Tailscale hoặc ZeroTier", cx, y+rowH+14, colMuted, text.AlignCenter, 1)
	if j.status != "" {
		for i, line := range wrapWords(j.status, 520) {
			drawText(dst, line, cx, y+rowH+46+float64(i)*18, color.RGBA{255, 140, 130, 255}, text.AlignCenter, 1)
		}
	}

	by := y + rowH + 110
	j.next = box{cx - 250, by, 240, 44}
	j.back = box{cx + 10, by, 240, 44}
	for _, b := range []struct {
		r     box
		label string
		off   bool
	}{{j.next, "Tiếp tục", j.text() == ""}, {j.back, "Quay lại", false}} {
		st, fg := tabIdle, colBoneText
		if b.off {
			st, fg = tabOff, color.RGBA{128, 116, 100, 255}
		}
		tab(dst, b.r.x, b.r.y, b.r.w, b.r.h, 12, st)
		uiText(dst, b.label, b.r.x+b.r.w/2, b.r.y+(b.r.h-lineUI*ccText)/2, fg, text.AlignCenter, ccText, fxShadow)
	}
	drawText(dst, "Enter để tiếp tục · Esc quay lại", cx, sh-26, colMuted, text.AlignCenter, 1)
}
