package client

import (
	"image/color"
	"math"
	"math/rand/v2"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/sim"
)

// charCreator is the character screen of a new game: a name, man or woman,
// hair style, hair colour and skin colour, beside a large preview that
// turns around.
type charCreator struct {
	name       []rune // as typed; trimmed when the game starts
	look       sim.Look
	hairSet    bool // the player picked a style, so a change of body keeps it
	focus      int  // the highlighted row
	frame      int
	startLabel string // the start button's words when it is not "Bắt đầu"
	turn       int    // quarter turns added by clicking the preview
	chars      []rune // reused buffer for the characters typed this frame

	// Layout from the last draw, for the mouse.
	rows           [ccRows]box
	swatches       [ccRows][]box // the colour rows' swatches
	preview        box
	lastMX, lastMY int
}

// box is a rectangle on the screen.
type box struct{ x, y, w, h float64 }

func (b box) has(x, y float64) bool { return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h }

// The rows of the character screen, top to bottom: the name, four choices,
// then three buttons.
const (
	ccName = iota
	ccBody
	ccHair
	ccColor
	ccSkin
	ccRandom
	ccStart
	ccBack
	ccRows
)

// isColorRow reports whether row is picked from a row of swatches.
func isColorRow(row int) bool { return row == ccColor || row == ccSkin }

// createResult is what the player did on the character screen.
type createResult uint8

const (
	createNone   createResult = iota
	createDone                // start the game with this look
	createCancel              // back to the main menu
)

// newCharCreator opens on the name, ready to type.
func newCharCreator() *charCreator {
	return &charCreator{look: sim.Look{Hair: sim.DefaultHair(false)}}
}

// NewName is the name typed on the character screen, after MenuNew. It is
// empty if the player left it blank; the server then picks one.
func (m *Menu) NewName() string { return m.create.nameText() }

// SetDefaultName fills the character screen's name field, for example with
// the -name flag.
func (m *Menu) SetDefaultName(name string) { m.create.setName(name) }

// update handles input; it runs once per frame while the screen is open.
func (c *charCreator) update() createResult {
	c.frame++
	if justPressed(ebiten.KeyEscape) {
		snd.ui("ui_move")
		return createCancel
	}
	prev := c.focus
	res := createNone
	// While the name has focus the letter keys type, so only the arrows,
	// Tab and Enter move.
	typing := c.focus == ccName
	if typing {
		c.typeName()
	}
	switch {
	case justPressed(ebiten.KeyArrowDown, ebiten.KeyTab) || !typing && justPressed(ebiten.KeyS):
		c.focus = (c.focus + 1) % ccRows
	case justPressed(ebiten.KeyArrowUp) || !typing && justPressed(ebiten.KeyW):
		c.focus = (c.focus + ccRows - 1) % ccRows
	case typing && justPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter):
		c.focus++
	case typing:
		res = c.mouse()
	case justPressed(ebiten.KeyArrowLeft, ebiten.KeyA):
		c.change(c.focus, -1)
	case justPressed(ebiten.KeyArrowRight, ebiten.KeyD):
		c.change(c.focus, 1)
	case justPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace):
		res = c.activate(c.focus)
	default:
		res = c.mouse()
	}
	if c.focus != prev {
		snd.ui("ui_move")
	}
	return res
}

// typeName edits the name with what was typed this frame. Backspace goes
// first: input methods such as Unikey rewrite a letter by sending Backspace
// and then the accented letter.
func (c *charCreator) typeName() {
	if d := inpututil.KeyPressDuration(ebiten.KeyBackspace); d == 1 || d > 30 && d%3 == 0 {
		c.name = eraseLast(c.name)
	}
	c.chars = ebiten.AppendInputChars(c.chars[:0])
	for _, r := range c.chars {
		c.name = typeRune(c.name, r)
	}
}

// typeRune adds r to a name being typed, unless it is a control character,
// a space the name does not need (leading or doubled), or past the length
// the server keeps.
func typeRune(name []rune, r rune) []rune {
	space := unicode.IsSpace(r)
	switch {
	case !space && !unicode.IsPrint(r), len(name) >= sim.MaxNameRunes:
		return name
	case space && (len(name) == 0 || name[len(name)-1] == ' '):
		return name
	case space:
		r = ' '
	}
	return append(name, r)
}

func eraseLast(name []rune) []rune {
	if len(name) == 0 {
		return name
	}
	snd.ui("ui_move")
	return name[:len(name)-1]
}

// nameText is the name as the game will use it.
func (c *charCreator) nameText() string { return strings.TrimSpace(string(c.name)) }

// setName fills the name field, cut to the length the server keeps.
func (c *charCreator) setName(name string) {
	c.name = c.name[:0]
	for _, r := range name {
		c.name = typeRune(c.name, r)
	}
}

// palette describes a colour row: how many swatches it has, the shades of
// swatch k (a ramp from dark to light) and which swatch is chosen.
func (c *charCreator) palette(row int) (n int, shades func(k int) ramp, chosen int) {
	if row == ccSkin {
		return sim.NumSkinTones, func(k int) ramp { return skinTones[k].ramp }, int(c.look.Skin)
	}
	return sim.NumHairColors, func(k int) ramp { return hairColors[k].ramp }, int(c.look.HairColor)
}

// pick chooses swatch k on a colour row.
func (c *charCreator) pick(row, k int) {
	if row == ccSkin {
		c.look.Skin = uint8(k)
	} else {
		c.look.HairColor = uint8(k)
	}
}

// change steps the choice on row by delta and reports whether there was one.
func (c *charCreator) change(row, delta int) bool {
	step := func(v, n int) int { return ((v+delta)%n + n) % n }
	switch {
	case row == ccBody:
		c.look.Female = !c.look.Female
		if !c.hairSet {
			c.look.Hair = sim.DefaultHair(c.look.Female)
		}
	case row == ccHair:
		c.look.Hair = sim.HairStyle(step(int(c.look.Hair), int(sim.NumHairStyles)))
		c.hairSet = true
	case isColorRow(row):
		n, _, chosen := c.palette(row)
		c.pick(row, step(chosen, n))
	default:
		return false
	}
	snd.ui("ui_move")
	return true
}

// activate is Enter or a click on row: a choice steps forward, a button acts.
func (c *charCreator) activate(row int) createResult {
	switch row {
	case ccName:
	case ccRandom:
		c.look = sim.Look{
			Female:    rand.IntN(2) == 1,
			Hair:      sim.HairStyle(rand.IntN(int(sim.NumHairStyles))),
			HairColor: uint8(rand.IntN(sim.NumHairColors)),
			Skin:      uint8(rand.IntN(sim.NumSkinTones)),
		}
		c.hairSet = true
		snd.ui("ui_select")
	case ccStart:
		snd.ui("ui_select")
		return createDone
	case ccBack:
		snd.ui("ui_move")
		return createCancel
	default:
		c.change(row, 1)
	}
	return createNone
}

func (c *charCreator) mouse() createResult {
	if c.rows[0].w == 0 {
		return createNone // not drawn yet
	}
	mx, my := uiCursor()
	moved := mx != c.lastMX || my != c.lastMY
	c.lastMX, c.lastMY = mx, my
	x, y := float64(mx), float64(my)
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if click && c.preview.has(x, y) {
		c.turn++
		snd.ui("ui_move")
		return createNone
	}
	for i, r := range c.rows {
		if !r.has(x, y) {
			continue
		}
		if moved {
			c.focus = i // hovering selects, but a still mouse does not fight the keys
		}
		if !click {
			return createNone
		}
		c.focus = i
		switch {
		case i == ccName:
			return createNone // focused: type away
		case isColorRow(i):
			_, _, chosen := c.palette(i)
			for k, s := range c.swatches[i] {
				if s.has(x, y) && k != chosen {
					c.pick(i, k)
					snd.ui("ui_move")
				}
			}
			return createNone
		case i == ccBody || i == ccHair:
			// The left of the value steps back, the right forward.
			d := 1
			if x < r.x+r.w*valueAt {
				d = -1
			}
			c.change(i, d)
			return createNone
		}
		return c.activate(i)
	}
	return createNone
}

// Layout of the character screen, in UI pixels. Rows are a little tighter
// on a short canvas (Full HD at UI scale 2 is 540 tall); see draw.
var ccRowH, ccGap, ccSwatchH = 44.0, 10.0, 40.0 // a row, between rows, the extra height of a colour row

const (
	ccColW  = 420.0 // the column of choices and buttons
	valueAt = 0.66  // where a choice's value is centred, across its row
)

// previewDirs is the order the preview turns through.
var previewDirs = [4]sim.Dir{sim.DirDown, sim.DirLeft, sim.DirUp, sim.DirRight}

func (c *charCreator) draw(dst *uiDst) {
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	ccRowH, ccGap, ccSwatchH = 44, 10, 40
	titleY, below := max(12, sh*0.08-24), 56.0
	if sh < 640 {
		ccRowH, ccGap, ccSwatchH = 38, 6, 34
		titleY, below = 10, 48
	}
	fillRect(dst, 0, 0, float32(sw), float32(sh), color.RGBA{6, 8, 16, 150}, false)
	cx := sw / 2
	drawTitle(dst, "Tạo nhân vật", cx, titleY, 3, color.RGBA{250, 214, 110, 255}, color.RGBA{60, 30, 10, 255})

	// The rows, top to bottom, with a little space after the name and
	// between the choices and the buttons.
	spaceAfter := func(i int) float64 {
		if i == ccName || i == ccSkin {
			return 2 * ccGap
		}
		return 0
	}
	colH := 0.0
	heights := [ccRows]float64{}
	for i := range heights {
		heights[i] = ccRowH
		if isColorRow(i) {
			heights[i] += ccSwatchH
		}
		colH += heights[i] + ccGap + spaceAfter(i)
	}
	colH -= ccGap
	pw := min(300, sw-ccColW-72)
	gap := 40.0
	if pw < 120 {
		pw, gap = 0, 0 // too narrow for the preview
	}
	left := cx - (pw+gap+ccColW)/2
	top := titleY + below

	c.preview = box{}
	if pw > 0 {
		c.preview = box{left, top, pw, colH} // drawn by drawPreview, on the screen itself
	}
	x, y := left+pw+gap, top
	for i := range c.rows {
		c.rows[i] = box{x, y, ccColW, heights[i]}
		c.drawRow(dst, i, c.rows[i])
		y += heights[i] + ccGap + spaceAfter(i)
	}
	hint := "W/S chọn dòng · A/D đổi · Enter xác nhận · Esc quay lại · nhấp vào nhân vật để xoay"
	if c.focus == ccName {
		hint = "Gõ tên (tối đa 16 chữ) · Backspace xoá · Enter hoặc Tab sang dòng sau · Esc quay lại"
	}
	drawText(dst, hint, cx, sh-26, colMuted, text.AlignCenter, 1)
}

// drawPreview draws the character in its box from the last draw, on the
// screen itself (k screen pixels to a UI pixel) so the art keeps its detail.
func (c *charCreator) drawPreview(dst *ebiten.Image, k float64) {
	if c.preview.w <= 0 {
		return
	}
	f := k
	b := box{c.preview.x * f, c.preview.y * f, c.preview.w * f, c.preview.h * f}
	drawFrame(&uiDst{img: dst, k: f}, float32(c.preview.x), float32(c.preview.y), float32(c.preview.w), float32(c.preview.h))
	a := getArt()
	s := float64(int(min((b.w-40*f)/18, (b.h-80*f)/38)))
	if playerHD {
		s = max(HDDensity, math.Floor(s/HDDensity)*HDDensity) // a whole number of screen pixels to each of the HD art's
	}
	fx, fy := b.x+b.w/2, b.y+b.h-36*f

	op := &ebiten.DrawImageOptions{} // the shadow
	op.GeoM.Translate(-shadowSize/2, -shadowSize/2)
	op.GeoM.Scale(12*s/shadowSize, 5*s/shadowSize)
	op.GeoM.Translate(fx, fy)
	op.ColorScale.ScaleAlpha(0.4)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(a.shadow, op)

	// Walk on the spot, turning a quarter every second and a half.
	dir := previewDirs[(c.frame/90+c.turn)%len(previewDirs)]
	frame := (c.frame / 4) % walkFrames
	bob := walkBob(frame)
	drawBody(dst, a.body(c.look), dir, frame, fx, fy, s, bob, playerColors[1])

	// The name above the head, as other players will see it.
	if name := c.nameText(); name != "" {
		drawText(dst, name, fx, fy-34*s-24*f, colWhite, text.AlignCenter, 2*f)
	}
}

func (c *charCreator) drawRow(dst *uiDst, i int, r box) {
	focused := i == c.focus
	st := tabIdle
	if focused {
		st = tabOn
	}
	tab(dst, r.x, r.y, r.w, r.h, 12, st)

	var label, value string
	switch i {
	case ccName:
		c.drawName(dst, r, focused)
		return
	case ccBody:
		label, value = "Giới tính", bodyName(c.look.Female)
	case ccHair:
		label, value = "Kiểu tóc", hairNames[c.look.Hair]
	case ccColor:
		label, value = "Màu tóc", hairColors[c.look.HairColor].name
	case ccSkin:
		label, value = "Màu da", skinTones[c.look.Skin].name
	case ccRandom:
		label = "Ngẫu nhiên"
	case ccStart:
		label = "Bắt đầu"
		if c.startLabel != "" {
			label = c.startLabel // "Mở phòng", "Vào phòng": see menu.go
		}
	case ccBack:
		label = "Quay lại"
	}
	ty := r.y + (ccRowH-lineUI*ccText)/2
	if value == "" { // a button
		uiText(dst, label, r.x+r.w/2, ty, colBoneText, text.AlignCenter, ccText, fxShadow)
		if focused {
			drawPointer(dst, r.x+16, r.y+ccRowH/2, 1, 6, colAmber)
		}
		return
	}
	uiText(dst, label, r.x+18, ty, colMuted, text.AlignStart, ccText, fxShadow)
	vx := r.x + r.w*valueAt
	uiText(dst, value, vx, ty, colWhite, text.AlignCenter, ccText, fxShadow)
	arrow := colMuted
	if focused {
		arrow = colAmber
	}
	drawPointer(dst, vx-88, r.y+ccRowH/2, -1, 6, arrow)
	drawPointer(dst, vx+88, r.y+ccRowH/2, 1, 6, arrow)

	if !isColorRow(i) {
		return
	}
	n, shades, chosen := c.palette(i)
	const size, gap = 30.0, 10.0
	sx := r.x + (r.w-(size*float64(n)+gap*float64(n-1)))/2
	sy := r.y + ccRowH
	c.swatches[i] = c.swatches[i][:0]
	for k := range n {
		b := box{sx + float64(k)*(size+gap), sy, size, size}
		c.swatches[i] = append(c.swatches[i], b)
		sh := shades(k)
		if k == chosen {
			glow(dst, b.x-3, b.y-3, b.w+6, b.h+6, 8, colAmberGlow, 0.55)
			strokeRoundRectHair(dst, b.x-3.5, b.y-3.5, b.w+7, b.h+7, 8, 1.6, colAmber)
		}
		well(dst, b.x-2, b.y-2, b.w+4, b.h+4, 7)
		fillRoundRect(dst, float32(b.x), float32(b.y), float32(b.w), float32(b.h), 5, sh[1])
		fillRoundRect(dst, float32(b.x+3), float32(b.y+3), float32(b.w-9), float32(b.h-12), 4, sh[2])
		fillRect(dst, float32(b.x+6), float32(b.y+6), 6, 3, sh[3], false)
	}
}

// drawName draws the name row: a label and a text box with a blinking caret.
// Left blank, it shows what the server will call the player.
func (c *charCreator) drawName(dst *uiDst, r box, focused bool) {
	ty := r.y + (r.h-lineUI*ccText)/2
	uiText(dst, "Tên", r.x+18, ty, colMuted, text.AlignStart, ccText, fxShadow)
	bx, bw := r.x+82, r.w-82-10
	well(dst, bx, r.y+6, bw, r.h-12, 7)
	tx := bx + 10
	name := string(c.name)
	if name == "" && !focused {
		uiText(dst, "Người chơi 1", tx, ty, color.RGBA{130, 118, 102, 255}, text.AlignStart, ccText, fxPlain)
		return
	}
	uiText(dst, name, tx, ty, colWhite, text.AlignStart, ccText, fxShadow)
	if focused && (c.frame/30)%2 == 0 {
		x := float32(tx + textWidth(name, ccText) + 2)
		fillRect(dst, x, float32(r.y+11), 2, float32(r.h-22), colKey, false)
	}
}

// ccText is the size of the character screen's row text.
const ccText = 1.45
