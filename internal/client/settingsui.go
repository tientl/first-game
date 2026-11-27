package client

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The settings screen ("Cài đặt"), opened from the main menu and from the
// pause menu: three tabs (Âm thanh, Hiển thị, Điều khiển), a row for each
// setting, and "Mặc định" and "Xong" at the foot. Every change takes effect
// at once and is saved (settings.go); "Mặc định" puts the tab's settings
// back as on a first run.
//
// The keys: W and S go from line to line (the tabs, the rows, the foot),
// A and D change what the line holds (a volume, a choice, a tab, a
// button), Enter flips a switch or steps a choice, Q and E (or Tab) turn
// the tabs from anywhere, and Esc closes it. The mouse points at a row,
// clicks the arrows or a switch, and drags a slider.
//
// The panel is laid out at one size (setW by setH UI pixels) and shrunk to
// fit a screen too small for it, like the difficulty picker.

// The panel's layout, in its own pixels.
const (
	setW, setH   = 880, 504
	setPad       = 22
	setTabW      = 156
	setTabH      = 38
	setTabY      = 16
	setRowsY     = 76
	setRowH      = 30
	setRowGap    = 2
	setSectionH  = 22
	setCtrlW     = 320 // the control at a row's right end
	setFootY     = setH - 52
	setFootH     = 36
	setFootBtnW  = 150
	setLabelSize = 1.15
	setValueSize = 1.1
)

// The tabs.
const (
	setTabAudio = iota
	setTabDisplay
	setTabControls
	numSetTabs
)

var setTabNames = [numSetTabs]string{"Âm thanh", "Hiển thị", "Điều khiển"}

// setRowID names a row.
type setRowID uint8

const (
	rowMaster setRowID = iota
	rowMusic
	rowEffects
	rowVoices
	rowMuteUnfocused
	rowWindow
	rowResolution
	rowQuality
	rowUIScale
	rowFPS
	rowVSync
	rowZoom
	rowShake
	rowPostFX
	rowGrade
	rowGround
	rowCaveLight
)

// setRowKind is the control a row has.
type setRowKind uint8

const (
	kindSlider setRowKind = iota // a volume, 0 to 100%
	kindSteps                    // a slider of a few steps (the zoom)
	kindChoice                   // ‹ a value › among a few
	kindToggle                   // on or off
)

type setRow struct {
	id      setRowID
	kind    setRowKind
	label   string
	section string // a heading drawn above the row, if any
}

// setRows is the rows of tab t.
func setRows(t int) []setRow {
	switch t {
	case setTabAudio:
		return []setRow{
			{rowMaster, kindSlider, "Âm lượng chung", "Âm lượng"},
			{rowMusic, kindSlider, "Nhạc", ""},
			{rowEffects, kindSlider, "Hiệu ứng", ""},
			{rowVoices, kindSlider, "Tiếng thú", ""},
			{rowMuteUnfocused, kindToggle, "Tắt tiếng khi chuyển sang cửa sổ khác", "Khác"},
		}
	case setTabDisplay:
		rows := []setRow{
			{rowWindow, kindChoice, "Chế độ màn hình", "Màn hình"},
			{rowResolution, kindChoice, "Độ phân giải", ""},
			{rowQuality, kindChoice, "Chất lượng vẽ", ""},
			{rowUIScale, kindChoice, "Cỡ giao diện", ""},
			{rowFPS, kindChoice, "Giới hạn khung hình", ""},
			{rowVSync, kindToggle, "VSync", ""},
			{rowZoom, kindSteps, "Thu phóng camera", "Hình ảnh"},
			{rowShake, kindToggle, "Rung màn hình", ""},
			{rowPostFX, kindToggle, "Ánh sáng và chỉnh màu", ""},
		}
		if len(colourGrades) > 0 {
			rows = append(rows, setRow{rowGrade, kindChoice, "Tông màu", ""})
		}
		return append(rows, setRow{rowGround, kindChoice, "Kiểu mặt đất", ""}, setRow{rowCaveLight, kindChoice, "Độ sáng hang", ""})
	}
	return nil
}

// settingsScreen is the screen's state.
type settingsScreen struct {
	tab    int
	line   int // -1 the tabs, 0 to len(rows)-1 a row, len(rows) the foot
	foot   int // on the foot: 0 "Mặc định", 1 "Xong"
	inGame bool

	frame int
	drag  int // the row whose slider the mouse holds, or -1

	// From the last draw, for the mouse: where the panel lies, and the
	// screen's size in its own pixels.
	ox, oy, scale  float64
	sw, sh         int
	lastMX, lastMY int
	drawn          bool
}

func newSettingsScreen(inGame bool) *settingsScreen {
	return &settingsScreen{inGame: inGame, drag: -1, sw: 1920, sh: 1080}
}

// rows is the rows of the tab shown.
func (s *settingsScreen) rows() []setRow { return setRows(s.tab) }

// setTabRect is where tab i lies on the panel: the tabs sit right of the
// title, the last against the right edge.
func setTabRect(i int) (x, y, w, h float64) {
	x0 := float64(setW-setPad) - numSetTabs*setTabW - (numSetTabs-1)*8
	return x0 + float64(i)*(setTabW+8), setTabY, setTabW, setTabH
}

// rowY is the top of row i of rows.
func rowY(rows []setRow, i int) float64 {
	y := float64(setRowsY)
	for j := 0; j <= i && j < len(rows); j++ {
		if rows[j].section != "" {
			y += setSectionH
		}
		if j < i {
			y += rowPitch(rows)
		}
	}
	return y
}

// rowPitch is how far apart the rows are: setRowH and a gap, or less when
// there are more than the panel holds.
func rowPitch(rows []setRow) float64 {
	sections := 0
	for _, r := range rows {
		if r.section != "" {
			sections++
		}
	}
	avail := float64(setFootY-14-setRowsY) - float64(sections*setSectionH)
	return min(setRowH+setRowGap, avail/float64(max(1, len(rows))))
}

// footRect is where foot button i lies: "Mặc định", then "Xong".
func footRect(i int) (x, y, w, h float64) {
	x = float64(setW-setPad) - 2*setFootBtnW - 10 + float64(i)*(setFootBtnW+10)
	return x, setFootY, setFootBtnW, setFootH
}

// ctrlX is the left end of a row's control.
const ctrlX = setW - setPad - 12 - setCtrlW

// --- values ---------------------------------------------------------------------

// setChoice is one value of a choice row: its name and how it is put.
type setChoice struct {
	label string
	put   func(s *Settings)
	on    bool // the value in force
	sharp bool // the art is at its sharpest (resolutions)
}

// choices is what choice row id offers now, on a screen sw by sh.
func (s *settingsScreen) choices(id setRowID) []setChoice {
	cur := settings
	var out []setChoice
	switch id {
	case rowWindow:
		for _, m := range []struct {
			mode  WindowMode
			label string
		}{{WindowFullscreen, "Toàn màn hình"}, {WindowBorderless, "Toàn màn hình không viền"}, {WindowWindowed, "Cửa sổ"}} {
			out = append(out, setChoice{label: m.label, on: cur.Window == m.mode, put: func(st *Settings) {
				if st.Window == WindowWindowed || m.mode == WindowWindowed {
					st.Width, st.Height = 0, 0 // a window's sizes are not fullscreen's
				}
				st.Window = m.mode
			}})
		}
	case rowResolution:
		mw, mh, dsf := monitorInfo()
		rs := resolutionChoices(cur.Window, mw, mh, dsf)
		in := currentResolution(rs, cur)
		for _, r := range rs {
			sharp := sharpest(r) || r.native() && sharpest(resolution{mw, mh})
			out = append(out, setChoice{label: resolutionLabel(r, mw, mh), on: r == in, sharp: sharp, put: func(st *Settings) {
				st.Width, st.Height = r.w, r.h
			}})
		}
	case rowQuality:
		for i, q := range qualityChoices {
			name := [...]string{"Cao (100%)", "Trung bình (75%)", "Thấp (50%)"}[i]
			out = append(out, setChoice{label: name, on: cur.Quality == q, put: func(st *Settings) { st.Quality = q }})
		}
	case rowUIScale:
		most := maxUIScale(s.sh)
		for _, p := range uiScaleChoices {
			if p > 0 && float64(p)/100 > most+1e-9 && p != cur.UIScale {
				continue // too big for this screen
			}
			label := fmt.Sprintf("%d%%", p)
			if p == 0 {
				label = fmt.Sprintf("Tự động (%s)", pct(autoUIScale(s.sh)))
			}
			out = append(out, setChoice{label: label, on: cur.UIScale == p, put: func(st *Settings) { st.UIScale = p }})
		}
	case rowFPS:
		for _, f := range fpsChoices {
			label := fmt.Sprintf("%d khung/giây", f)
			if f == 0 {
				label = "Không giới hạn"
			}
			out = append(out, setChoice{label: label, on: cur.FPS == f, put: func(st *Settings) { st.FPS = f }})
		}
	case rowGrade:
		out = append(out, setChoice{label: "Hiện tại", on: cur.Grade == "", put: func(st *Settings) { st.Grade = "" }})
		for _, g := range colourGrades {
			out = append(out, setChoice{label: g.label, on: cur.Grade == g.name, put: func(st *Settings) { st.Grade = g.name }})
		}
	case rowGround:
		out = append(out,
			setChoice{label: "Cỏ xanh", on: cur.Ground != "warm", put: func(st *Settings) { st.Ground = "" }},
			setChoice{label: "Đất ấm", on: cur.Ground == "warm", put: func(st *Settings) { st.Ground = "warm" }})
	case rowCaveLight:
		for i, name := range [...]string{"Tối (cần đuốc)", "Mờ", "Sáng"} {
			out = append(out, setChoice{label: name, on: cur.CaveLight == i, put: func(st *Settings) { st.CaveLight = i }})
		}
	}
	return out
}

// pct writes a scale as a percentage: 2.5 is "250%".
func pct(k float64) string { return fmt.Sprintf("%d%%", int(math.Round(k*100))) }

// chosen is the index of the choice in force, or 0.
func chosen(cs []setChoice) int {
	for i, c := range cs {
		if c.on {
			return i
		}
	}
	return 0
}

// level is a slider row's value, 0 to 1.
func level(id setRowID) float64 {
	st := settings
	switch id {
	case rowMaster:
		return st.Master
	case rowMusic:
		return st.Music
	case rowEffects:
		return st.Effects
	case rowVoices:
		return st.Voices
	case rowZoom:
		return float64(st.Zoom-minZoom) / float64(maxZoom-minZoom)
	}
	return 0
}

// setLevel sets a slider row to v, 0 to 1.
func setLevel(id setRowID, v float64) {
	v = clamp01(v)
	if id == rowZoom {
		z := minZoom + int(math.Round(v*float64(maxZoom-minZoom)))
		changeSettings(func(st *Settings) { st.Zoom = z })
		return
	}
	v = math.Round(v*20) / 20 // steps of 5%
	if math.Abs(level(id)-v) < 1e-9 {
		return
	}
	changeSettings(func(st *Settings) {
		switch id {
		case rowMaster:
			st.Master = v
		case rowMusic:
			st.Music = v
		case rowEffects:
			st.Effects = v
		case rowVoices:
			st.Voices = v
		}
	})
	// Let the player hear the new level.
	switch id {
	case rowMaster, rowEffects:
		if snd != nil && snd.ready("settings_tick", 90*time.Millisecond) {
			snd.play("ui_select", 0.5, 0, 1)
		}
	case rowVoices:
		if snd != nil && snd.ready("settings_voice", 700*time.Millisecond) {
			snd.play("trike_call", 0.5, 0, 1)
		}
	}
}

// isOn is a switch row's state.
func isOn(id setRowID) bool {
	st := settings
	switch id {
	case rowMuteUnfocused:
		return st.MuteUnfocused
	case rowVSync:
		return st.VSync
	case rowShake:
		return !st.NoShake
	case rowPostFX:
		return st.PostFX
	}
	return false
}

// flip turns a switch row over.
func flip(id setRowID) {
	changeSettings(func(st *Settings) {
		switch id {
		case rowMuteUnfocused:
			st.MuteUnfocused = !st.MuteUnfocused
		case rowVSync:
			st.VSync = !st.VSync
		case rowShake:
			st.NoShake = !st.NoShake
		case rowPostFX:
			st.PostFX = !st.PostFX
		}
	})
}

// step moves row r's value a step: dir -1 or 1.
func (s *settingsScreen) step(r setRow, dir int) {
	switch r.kind {
	case kindSlider:
		setLevel(r.id, level(r.id)+0.05*float64(dir))
	case kindSteps:
		setLevel(r.id, level(r.id)+float64(dir)/float64(maxZoom-minZoom))
	case kindToggle:
		if (dir > 0) != isOn(r.id) {
			flip(r.id)
		}
	case kindChoice:
		cs := s.choices(r.id)
		if len(cs) == 0 {
			return
		}
		i := (chosen(cs) + dir + len(cs)) % len(cs)
		changeSettings(cs[i].put)
	}
	snd.ui("ui_move")
}

// resetTab puts the tab's settings back as on a first run.
func (s *settingsScreen) resetTab() {
	d := DefaultSettings()
	changeSettings(func(st *Settings) {
		switch s.tab {
		case setTabAudio:
			st.Master, st.Music, st.Effects, st.Voices, st.MuteUnfocused = d.Master, d.Music, d.Effects, d.Voices, d.MuteUnfocused
		case setTabDisplay:
			keep := *st
			*st = d
			st.Master, st.Music, st.Effects, st.Voices, st.MuteUnfocused = keep.Master, keep.Music, keep.Effects, keep.Voices, keep.MuteUnfocused
		}
	})
	snd.ui("ui_select")
}

// --- input -----------------------------------------------------------------------

// update handles a frame's input and reports whether the screen is done.
func (s *settingsScreen) update() bool {
	s.frame++
	rows := s.rows()
	nLines := len(rows) + 2 // the tabs, the rows, the foot
	at := s.line + 1        // 0 to nLines-1
	moved := false
	switch {
	case justPressed(ebiten.KeyEscape):
		snd.ui("ui_select")
		return true
	case justPressed(ebiten.KeyQ) || justPressed(ebiten.KeyTab) && shiftHeld():
		s.setTab(s.tab - 1)
	case justPressed(ebiten.KeyE, ebiten.KeyTab):
		s.setTab(s.tab + 1)
	case justPressed(ebiten.KeyS, ebiten.KeyArrowDown):
		at, moved = (at+1)%nLines, true
	case justPressed(ebiten.KeyW, ebiten.KeyArrowUp):
		at, moved = (at+nLines-1)%nLines, true
	}
	if moved {
		s.line = at - 1
		snd.ui("ui_move")
	}
	rows = s.rows() // the tab may have turned
	left := justPressed(ebiten.KeyA, ebiten.KeyArrowLeft)
	right := justPressed(ebiten.KeyD, ebiten.KeyArrowRight)
	enter := justPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace) && !altHeld() // Alt+Enter is fullscreen
	switch {
	case s.line < 0: // the tabs
		switch {
		case left:
			s.setTab(s.tab - 1)
		case right:
			s.setTab(s.tab + 1)
		case enter && len(s.rows()) > 0:
			s.line = 0
			snd.ui("ui_move")
		}
	case s.line < len(rows):
		r := rows[s.line]
		switch {
		case left:
			s.step(r, -1)
		case right:
			s.step(r, 1)
		case enter && r.kind == kindToggle:
			flip(r.id)
			snd.ui("ui_select")
		case enter && r.kind == kindChoice:
			s.step(r, 1)
		}
	default: // the foot
		switch {
		case (left || right) && len(rows) > 0:
			s.foot = 1 - s.foot
			snd.ui("ui_move")
		case enter:
			if s.foot == 1 {
				snd.ui("ui_select")
				return true
			}
			s.resetTab()
		}
	}
	return s.mouse()
}

// setTab shows tab t (wrapping). The line in hand stays the same kind:
// the tabs, a row (the nearest there is), or the foot.
func (s *settingsScreen) setTab(t int) {
	old := len(s.rows())
	s.tab = (t + numSetTabs) % numSetTabs
	n := len(s.rows())
	switch {
	case s.line >= old: // the foot
		s.line = n
	case s.line >= 0 && n == 0:
		s.line = n
	case s.line >= 0:
		s.line = min(s.line, n-1)
	}
	if n == 0 {
		s.foot = 1 // "Xong": there is nothing to reset here
	}
	snd.ui("ui_move")
}

// mouse handles the pointer: hovering picks a line, a click acts. It
// reports whether "Xong" was clicked.
func (s *settingsScreen) mouse() bool {
	if !s.drawn || s.scale <= 0 {
		return false
	}
	ux, uy := uiCursor()
	moved := ux != s.lastMX || uy != s.lastMY
	s.lastMX, s.lastMY = ux, uy
	px, py := (float64(ux)-s.ox)/s.scale, (float64(uy)-s.oy)/s.scale
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	held := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	rows := s.rows()

	if s.drag >= 0 {
		if !held || s.drag >= len(rows) {
			s.drag = -1
		} else {
			setLevel(rows[s.drag].id, sliderAt(rows[s.drag], px))
		}
		return false
	}
	in := func(x, y, w, h float64) bool { return px >= x && px < x+w && py >= y && py < y+h }
	for i := range numSetTabs {
		if in(setTabRect(i)) {
			if click && i != s.tab {
				s.setTab(i)
			}
			if moved || click {
				s.line = -1
			}
			return false
		}
	}
	for i := range 2 {
		if i == 0 && len(rows) == 0 {
			continue // "Mặc định" is off where there is nothing to reset
		}
		if in(footRect(i)) {
			if moved || click {
				s.line, s.foot = len(rows), i
			}
			if click {
				if i == 1 {
					snd.ui("ui_select")
					return true
				}
				s.resetTab()
			}
			return false
		}
	}
	for i, r := range rows {
		y := rowY(rows, i)
		if !in(setPad, y, setW-2*setPad, rowPitch(rows)-setRowGap) {
			continue
		}
		if moved || click {
			s.line = i
		}
		if !click {
			return false
		}
		switch r.kind {
		case kindSlider, kindSteps:
			if px >= ctrlX-8 {
				s.drag = i
				setLevel(r.id, sliderAt(r, px))
			}
		case kindToggle:
			flip(r.id)
			snd.ui("ui_select")
		case kindChoice:
			if px >= ctrlX && px < ctrlX+setCtrlW/2 {
				s.step(r, -1)
			} else if px >= ctrlX {
				s.step(r, 1)
			}
		}
		return false
	}
	return false
}

// The slider's track, in panel pixels: from sliderX0 to sliderX1, its knob
// travelling from sliderA to sliderB, within the ends.
const (
	sliderX0 = ctrlX + 10
	sliderX1 = ctrlX + setCtrlW - 70
	sliderA  = sliderX0 + 7
	sliderB  = sliderX1 - 7
)

// sliderAt is the value under panel x on row r's slider.
func sliderAt(r setRow, x float64) float64 {
	return clamp01((x - sliderA) / (sliderB - sliderA))
}

// --- drawing -----------------------------------------------------------------------

// setFit is how much the panel is shrunk to fit a screen uw by uh UI
// pixels, and where its corner goes.
func setFit(uw, uh int) (scale, ox, oy float64) {
	scale = min(1, float64(uw-16)/setW, float64(uh-34)/setH)
	ox = math.Round((float64(uw) - setW*scale) / 2)
	oy = math.Round((float64(uh) - 24 - setH*scale) / 2)
	return scale, max(0, ox), max(4, oy)
}

// draw draws the screen over whatever is behind it.
func (s *settingsScreen) draw(screen *uiDst) {
	uw, uh := screen.Bounds().Dx(), screen.Bounds().Dy()
	s.sw, s.sh = screen.img.Bounds().Dx(), screen.img.Bounds().Dy()
	dimScreen(screen, color.RGBA{0, 0, 0, 120})
	s.scale, s.ox, s.oy = setFit(uw, uh)
	s.drawn = true
	b := screen.view(s.ox, s.oy, s.scale)
	drawFrame(b, 4, 4, setW-8, setH-8)

	uiTextW(b, "Cài đặt", setPad+2, 18, colBoneText, text.AlignStart, 2.1, wTitle, fxShadow)
	for i := range numSetTabs {
		x, y, w, h := setTabRect(i)
		st := tabIdle
		if i == s.tab {
			st = tabOn
		}
		tab(b, x, y, w, h, 10, st)
		fg := colMuted
		if i == s.tab {
			fg = color.RGBA{255, 240, 206, 255}
		}
		uiText(b, setTabNames[i], x+w/2, y+(h-lineUI*1.25)/2, fg, text.AlignCenter, 1.25, fxShadow)
		if i == s.tab && s.line < 0 {
			drawPointer(b, x+14, y+h/2, 1, 9, colAmber)
		}
	}
	fillRect(b, setPad, 64, setW-2*setPad, 1, color.RGBA{150, 112, 70, 150}, false)

	rows := s.rows()
	if s.tab == setTabControls {
		s.drawControls(b)
	}
	for i, r := range rows {
		s.drawRow(b, rows, i, r)
	}

	// The foot: what the line in hand does, and the two buttons.
	fillRect(b, setPad, setFootY-10, setW-2*setPad, 1, color.RGBA{150, 112, 70, 110}, false)
	fx, _, _, _ := footRect(0)
	help := fitText(s.help(), fx-setPad-16, 1)
	drawText(b, help, setPad+4, setFootY+(setFootH-lineUI)/2, colMuted, text.AlignStart, 1)
	for i, name := range []string{"Mặc định", "Xong"} {
		x, y, w, h := footRect(i)
		st := tabIdle
		if s.line == len(rows) && s.foot == i {
			st = tabOn
		}
		if i == 0 && s.tab == setTabControls {
			st = tabOff
		}
		tab(b, x, y, w, h, 10, st)
		fg := colBoneText
		if st == tabOff {
			fg = color.RGBA{128, 116, 100, 255}
		}
		uiText(b, name, x+w/2, y+(h-lineUI*1.3)/2, fg, text.AlignCenter, 1.3, fxShadow)
	}

	// The keys, under the panel.
	if hy := s.oy + setH*s.scale + 6; hy+lineUI < float64(uh) {
		drawText(screen, "W/S chọn dòng · A/D chỉnh · Q/E đổi thẻ · Enter bật/tắt · Esc xong", float64(uw)/2, hy, colMuted, text.AlignCenter, 1)
	}
}

// help is a line on the line in hand: what a row does, or the keys.
func (s *settingsScreen) help() string {
	rows := s.rows()
	if s.line < 0 || s.line >= len(rows) {
		switch {
		case s.line >= len(rows) && s.foot == 0 && s.tab != setTabControls:
			return "Đặt lại thẻ “" + setTabNames[s.tab] + "” như lúc mới cài game."
		case s.tab == setTabControls:
			return "Các phím của game. Trong game, F1 hiện bảng phím ở góc màn hình."
		}
		return "Mọi thay đổi có hiệu lực ngay và được lưu lại."
	}
	switch rows[s.line].id {
	case rowMaster:
		return "Âm lượng của mọi tiếng trong game."
	case rowMusic:
		return "Nhạc nền: ngày, đêm, trong hang và khi đánh trùm."
	case rowEffects:
		return "Tiếng chặt cây, đập đá, bước chân, đánh nhau, giao diện."
	case rowVoices:
		return "Tiếng gầm, tiếng kêu của khủng long, muông thú và thổ dân."
	case rowMuteUnfocused:
		return "Game im lặng khi bạn chuyển sang cửa sổ khác (Alt+Tab)."
	case rowWindow:
		switch settings.Window {
		case WindowBorderless:
			return "Cửa sổ không khung phủ kín màn hình: Alt+Tab nhanh, hợp với hai màn hình."
		case WindowWindowed:
			return "Cửa sổ có khung, kéo được cỡ. F11 hoặc Alt+Enter đổi nhanh."
		}
		return "Chiếm cả màn hình. F11 hoặc Alt+Enter đổi nhanh sang cửa sổ."
	case rowResolution:
		if settings.Window == WindowWindowed {
			return "Cỡ cửa sổ. Hình nét nhất ở 1920×1080 và gấp đôi (3840×2160)."
		}
		return "Cỡ hình vẽ, phóng vừa màn hình. Nét nhất ở 1920×1080 và 3840×2160."
	case rowQuality:
		r := worldScale(s.sh)
		rr := renderScale(r)
		if rr >= r {
			return "Thế giới vẽ đủ độ phân giải màn hình."
		}
		w, h := renderSize(s.sw, s.sh, r, rr)
		return fmt.Sprintf("Thế giới vẽ ở %d×%d rồi phóng lên; chữ và giao diện vẫn nét.", w, h)
	case rowUIScale:
		return fmt.Sprintf("Cỡ chữ, thanh máu, túi đồ. Tự động theo độ cao màn hình; đang dùng %s.", pct(uiScaleFor(s.sh)))
	case rowFPS:
		return "Số hình vẽ mỗi giây. Giới hạn thấp hơn thì máy đỡ nóng, đỡ tốn pin."
	case rowVSync:
		return "Chờ màn hình làm tươi mới vẽ: không bị xé hình khi lia camera."
	case rowZoom:
		return "Gần hay xa. Trong game: phím = và −, hoặc Ctrl + lăn chuột."
	case rowShake:
		return "Màn hình rung khi trúng đòn nặng, thú lớn gầm hay giậm."
	case rowPostFX:
		return "Lửa, đuốc toả sáng và màu theo giờ trong ngày. Tốn máy hơn một chút."
	case rowGrade:
		return "Tông màu của thế giới."
	case rowGround:
		return "Cỏ xanh mướt, hoặc đất vàng ấm lốm đốm cỏ."
	case rowCaveLight:
		return "Chỗ không có đèn trong hang tối đến đâu. Tối: phải cầm đuốc mới thấy đường."
	}
	return ""
}

// drawRow draws row i: its label, and its control on the right.
func (s *settingsScreen) drawRow(b *uiDst, rows []setRow, i int, r setRow) {
	y := rowY(rows, i)
	if r.section != "" {
		uiTextW(b, strings.ToUpper(r.section), setPad+6, y-setSectionH+4, color.RGBA{214, 170, 104, 255}, text.AlignStart, 0.9, wHeavy, fxShadow)
	}
	x, w := float64(setPad), float64(setW-2*setPad)
	rh := rowPitch(rows) - setRowGap
	sel := s.line == i
	if sel {
		tab(b, x, y, w, rh, 8, tabOn)
		drawPointer(b, x+12, y+rh/2, 1, 8, colAmber)
	} else if i%2 == 0 {
		fillRect(b, float32(x), float32(y), float32(w), float32(rh), color.NRGBA{255, 240, 210, 10}, false)
	}
	fg := colBoneText
	if sel {
		fg = color.RGBA{255, 240, 206, 255}
	}
	ty := y + (rh-lineUI*setLabelSize)/2
	uiText(b, r.label, x+26, ty, fg, text.AlignStart, setLabelSize, fxShadow)

	vy := y + (rh-lineUI*setValueSize)/2
	cy := y + rh/2
	switch r.kind {
	case kindSlider, kindSteps:
		v := level(r.id)
		well(b, sliderX0, cy-5, sliderX1-sliderX0, 10, 5)
		fill := color.RGBA{226, 160, 70, 255}
		if r.kind == kindSlider && v > 0 {
			drawBarFill(b, sliderX0+1, cy-4, sliderA-sliderX0-1+(sliderB-sliderA)*v, 8, 1, fill, barFX{}, s.frame)
		}
		if r.kind == kindSteps {
			for k := 0; k <= maxZoom-minZoom; k++ {
				tx := float32(sliderA + (sliderB-sliderA)*float64(k)/float64(maxZoom-minZoom))
				fillCircle(b, tx, float32(cy), 2, color.RGBA{150, 120, 86, 255}, true)
			}
		}
		kx := float32(sliderA + (sliderB-sliderA)*v)
		fillCircle(b, kx, float32(cy)+1, 8.5, color.RGBA{0, 0, 0, 110}, true)
		knob := color.RGBA{236, 224, 196, 255}
		if sel || s.drag == i {
			knob = color.RGBA{255, 226, 150, 255}
		}
		fillCircle(b, kx, float32(cy), 8, knob, true)
		strokeCircle(b, kx, float32(cy), 8, 1.5, color.RGBA{120, 84, 50, 255}, true)
		val := fmt.Sprintf("%d%%", int(math.Round(v*100)))
		if r.kind == kindSteps {
			z := settings.Zoom
			val = fmt.Sprintf("%+d", z)
			if z == 0 {
				val = "0"
			}
		}
		uiText(b, val, ctrlX+setCtrlW, vy, colKey, text.AlignEnd, setValueSize, fxShadow)
	case kindToggle:
		on := isOn(r.id)
		pw, ph := 52.0, 24.0
		px := float64(ctrlX+setCtrlW) - pw
		well(b, px, cy-ph/2, pw, ph, ph/2)
		if on {
			drawBarFill(b, px+2, cy-ph/2+2, pw-4, ph-4, 1, color.RGBA{120, 190, 96, 255}, barFX{}, s.frame)
		}
		kx := px + ph/2
		if on {
			kx = px + pw - ph/2
		}
		fillCircle(b, float32(kx), float32(cy)+1, 9.5, color.RGBA{0, 0, 0, 110}, true)
		fillCircle(b, float32(kx), float32(cy), 9, color.RGBA{236, 224, 196, 255}, true)
		label, c := "Tắt", colMuted
		if on {
			label, c = "Bật", colGood
		}
		uiText(b, label, px-12, vy, c, text.AlignEnd, setValueSize, fxShadow)
	case kindChoice:
		cs := s.choices(r.id)
		if len(cs) == 0 {
			return
		}
		c := cs[chosen(cs)]
		well(b, ctrlX, cy-12, setCtrlW, 24, 12)
		ac := color.RGBA{150, 120, 86, 255}
		if sel {
			ac = colAmber
		}
		if len(cs) > 1 {
			drawPointer(b, ctrlX+14, cy, -1, 9, ac)
			drawPointer(b, ctrlX+setCtrlW-14, cy, 1, 9, ac)
		}
		label := fitText(c.label, setCtrlW-56, setValueSize)
		uiText(b, label, ctrlX+setCtrlW/2, vy, colKey, text.AlignCenter, setValueSize, fxShadow)
		if c.sharp {
			const cw = 74.0
			cx := float64(ctrlX) - 12 - cw
			plate(b, cx, cy-10, cw, 20, 10)
			uiTextW(b, "nét nhất", cx+cw/2, cy-10+(20-lineUI*0.9)/2, colGood, text.AlignCenter, 0.9, wHeavy, fxShadow)
		}
	}
}

// setKey is a line of the controls: the keys, and what they do.
type setKey struct {
	keys []string
	does string
	all  bool // the keys go together (W A S D), not one or another
}

var setKeysLeft = []setKey{
	{keys: []string{"W", "A", "S", "D"}, does: "đi lại (hoặc phím mũi tên)", all: true},
	{keys: []string{"Chuột trái", "Space"}, does: "chặt, đập, hái, đánh (giữ: làm liền tay)"},
	{keys: []string{"Chuột phải"}, does: "dùng món đang cầm: ăn, cắm đuốc…"},
	{keys: []string{"Q"}, does: "giữ để giơ khiên đỡ đòn"},
	{keys: []string{"E"}, does: "nhặt đồ, mở rương, cưỡi thú, lên thuyền"},
	{keys: []string{"R"}, does: "cưỡi thú, lên thuyền"},
	{keys: []string{"V"}, does: "đặt món đang cầm xuống đất"},
	{keys: []string{"G"}, does: "vứt một món (Shift + G: cả chồng)"},
	{keys: []string{"1 – 0", "Lăn chuột"}, does: "chọn ô trên thanh đồ"},
}

var setKeysRight = []setKey{
	{keys: []string{"Tab", "I", "B"}, does: "túi đồ"},
	{keys: []string{"C"}, does: "chế tạo"},
	{keys: []string{"M"}, does: "bản đồ (lăn chuột để phóng to)"},
	{keys: []string{"=", "−"}, does: "thu phóng (hoặc Ctrl + lăn chuột)"},
	{keys: []string{"F1"}, does: "bảng phím ở góc màn hình"},
	{keys: []string{"F11", "Alt + Enter"}, does: "toàn màn hình hoặc cửa sổ"},
	{keys: []string{"Esc"}, does: "tạm dừng, đóng bảng đang mở"},
}

// drawControls draws the keys in two columns.
func (s *settingsScreen) drawControls(b *uiDst) {
	colW := float64(setW-2*setPad-24) / 2
	for c, list := range [][]setKey{setKeysLeft, setKeysRight} {
		x := float64(setPad) + float64(c)*(colW+24)
		title := []string{"Di chuyển và hành động", "Bảng và màn hình"}[c]
		uiTextW(b, strings.ToUpper(title), x+6, setRowsY+4, color.RGBA{214, 170, 104, 255}, text.AlignStart, 0.9, wHeavy, fxShadow)
		y := float64(setRowsY + setSectionH + 4)
		for i, k := range list {
			if i%2 == 0 {
				fillRect(b, float32(x), float32(y-3), float32(colW), 30, color.NRGBA{255, 240, 210, 10}, false)
			}
			kx := x + 6
			for j, key := range k.keys {
				if j > 0 && !k.all {
					drawText(b, "/", kx+5, y+3, colMuted, text.AlignCenter, 0.9)
					kx += 10
				}
				kw := max(26, textWidthW(key, 0.95, wHeavy)+14)
				drawPanelLook(b, psLeather, keyCapLook, kx, y, kw, 24, 6, 0.4)
				noteBox(b, kx, y, kw, 24)
				uiTextW(b, key, kx+kw/2, y+(24-lineUI*0.95)/2, colKey, text.AlignCenter, 0.95, wHeavy, fxShadow)
				kx += kw + 4
			}
			drawText(b, k.does, x+156, y+(24-lineUI)/2, colBoneText, text.AlignStart, 1)
			y += 32
		}
	}
}

// keyCapLook is a key on the controls tab: a small raised leather cap.
var keyCapLook = panelLook{
	body: color.RGBA{70, 52, 36, 245}, rim: color.RGBA{140, 104, 66, 255},
	rimW: 1.5, stitch: color.RGBA{},
}

// showTab shows the tab named "audio", "display" or "controls".
func (s *settingsScreen) showTab(name string) {
	for i, n := range []string{"audio", "display", "controls"} {
		if n == name {
			s.tab = i
		}
	}
	if len(s.rows()) == 0 {
		s.line, s.foot = 0, 1 // the foot, on "Xong": there is nothing to reset
	}
}
