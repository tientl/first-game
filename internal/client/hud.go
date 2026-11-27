package client

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/sim"
)

// The HUD, on the interface canvas (see uiscale.go): the vitals and status
// chips top left, the clock with the minimap top right, the hotbar along the
// bottom with the held item's name and one line of what to do here above
// it. F1 shows the full list of keys.

var (
	colPanel     = color.RGBA{14, 16, 26, 205}
	colPanelEdge = color.RGBA{0, 0, 0, 90}
	colSlot      = color.RGBA{0, 0, 0, 110}
	colHealth    = color.RGBA{214, 62, 70, 255}
	colHunger    = color.RGBA{232, 152, 52, 255}
	colThirst    = color.RGBA{70, 150, 230, 255}
	colGood      = color.RGBA{150, 230, 140, 255}
	colMuted     = color.RGBA{176, 182, 196, 255}
	colKey       = color.RGBA{246, 213, 92, 255}
	colColdTxt   = color.RGBA{150, 210, 255, 255}
	colPoison    = color.RGBA{140, 230, 100, 255}
	colDay       = color.RGBA{250, 214, 96, 255}
	colDusk      = color.RGBA{236, 128, 74, 255}
	colNight     = color.RGBA{52, 62, 132, 255}
)

func (g *Game) text(dst *ebiten.Image, s string, x, y float64, clr color.Color, align text.Align, scale float64) {
	drawText(dst, s, x, y, clr, align, scale)
}

func (g *Game) textWidth(s string) float64 { return textWidth(s, 1) }

func (g *Game) dim(dst *ebiten.Image, sw, sh int, alpha float64) {
	uiBreak() // what it covers goes on the screen first, HD icons and all: see iconhd.go
	vector.FillRect(dst, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, uint8(alpha * 255)}, false)
}

func fillRoundRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	r = min(r, w/2, h/2)
	var p vector.Path
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r)
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, nil, op)
}

// panel draws the dark rounded backdrop every HUD element sits on.
func panel(dst *ebiten.Image, x, y, w, h float32) {
	fillRoundRect(dst, x-1, y-1, w+2, h+2, 9, colPanelEdge)
	fillRoundRect(dst, x, y, w, h, 8, colPanel)
}

func lighten(c color.RGBA, d int) color.RGBA {
	ch := func(v uint8) uint8 { return uint8(max(0, min(255, int(v)+d))) }
	return color.RGBA{ch(c.R), ch(c.G), ch(c.B), c.A}
}

func ingredientsText(r sim.Recipe) string {
	parts := make([]string, 0, len(r.Inputs))
	for _, in := range r.Inputs {
		parts = append(parts, fmt.Sprintf("%d %s", in.Count, in.Item.Def().Name))
	}
	return strings.Join(parts, " · ")
}

// fade is c at alpha a, premultiplied as the drawing functions expect: a
// see-through colour, not a glow.
func fade(c color.RGBA, a uint8) color.RGBA {
	k := func(v uint8) uint8 { return uint8(int(v) * int(a) / 255) }
	return color.RGBA{k(c.R), k(c.G), k(c.B), a}
}

func drawIcon(dst *ebiten.Image, img *ebiten.Image, x, y, scale float64, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(alpha)
	dst.DrawImage(img, op)
}

// --- the vitals --------------------------------------------------------------------

// vital is one bar top left: health, hunger, and later thirst.
type vital struct {
	icon     *ebiten.Image // nil: drawIcon draws a water drop
	cur, max float64
	fill     color.RGBA
}

// HUD layout, in UI pixels.
const (
	vitalX, vitalY = 12, 12 // the vitals panel's corner
	vitalBarW      = 176
	vitalBarH      = 22
	vitalRow       = 30
	vitalPanelW    = 12 + 30 + vitalBarW + 12
	chipH          = 22
)

// vitals are the bars shown, top to bottom. Thirst goes here once the sim
// has it: {nil, self.Thirst, sim.MaxThirst, colThirst}.
func (g *Game) vitals() []vital {
	self := g.curr.Self
	return []vital{
		{g.art.heart, float64(self.Health), sim.MaxHealth, colHealth},
		{g.art.food, float64(self.Hunger), sim.MaxHunger, colHunger},
	}
}

// vitalsBottom is the y just under the vitals panel.
func vitalsBottom(n int) float32 { return vitalY + 10 + float32(n)*vitalRow }

// drawVital draws one bar with its icon at (x, y), its value as "72/100"
// inside it.
func (g *Game) drawVital(dst *ebiten.Image, v vital, x, y float32) {
	frac := 0.0
	if v.max > 0 {
		frac = max(0, min(1, v.cur/v.max))
	}
	if v.icon != nil {
		b := v.icon.Bounds()
		drawIcon(dst, v.icon, float64(x)+13-float64(b.Dx()), float64(y)+vitalBarH/2-float64(b.Dy()), 2, 1)
	} else {
		drawDrop(dst, x+13, y+vitalBarH/2+1, 8, v.fill)
	}
	bx := x + 30
	fillRoundRect(dst, bx, y, vitalBarW, vitalBarH, 6, color.RGBA{0, 0, 0, 170})
	fill := v.fill
	if frac < 0.25 && (g.frame/15)%2 == 0 {
		fill = lighten(fill, 50) // flash when low
	}
	if fw := float32(frac * (vitalBarW - 4)); fw >= 2 {
		fillRoundRect(dst, bx+2, y+2, fw, vitalBarH-4, 4, fill)
		fillRoundRect(dst, bx+3, y+3, fw-2, 5, 2.5, fade(lighten(fill, 70), 110))
		fillRoundRect(dst, bx+3, y+vitalBarH-7, fw-2, 4, 2, fade(lighten(fill, -60), 120))
	}
	value := fmt.Sprintf("%d/%d", int(math.Ceil(v.cur)), int(v.max))
	g.text(dst, value, float64(bx)+vitalBarW/2, float64(y)+float64(vitalBarH)/2-8, colWhite, text.AlignCenter, 1)
}

// drawDrop is a water drop centred on (x, y), r across.
func drawDrop(dst *ebiten.Image, x, y, r float32, c color.RGBA) {
	vector.FillCircle(dst, x, y+r*0.25, r*0.6, c, true)
	var p vector.Path
	p.MoveTo(x, y-r*0.85)
	p.LineTo(x+r*0.55, y+r*0.1)
	p.LineTo(x-r*0.55, y+r*0.1)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, &p, nil, op)
	vector.FillCircle(dst, x-r*0.2, y+r*0.1, r*0.15, color.RGBA{255, 255, 255, 170}, true)
}

// --- status effects ------------------------------------------------------------------

type statusKind uint8

const (
	statusCold statusKind = iota
	statusPoison
	statusWeb
	statusStarving
	statusHungry
	statusBees
	statusBuff  // the first of the buffs of dishes, one for each sim.BuffKind; see cookrender.go
	numStatuses = statusBuff + statusKind(sim.NumBuffs)
)

// status is one effect on the player: a chip under the bars, and the long
// line said for a moment when it starts.
type status struct {
	kind  statusKind
	label string
	long  string
	col   color.RGBA
}

// statusShout is how long, in frames, the long line shows after a status
// starts.
const statusShout = 4 * 60

// statuses are the effects on the player now.
func (g *Game) statuses() []status {
	self := g.curr.Self
	if self.Dead {
		return nil
	}
	var out []status
	if self.Cold {
		long := "Lạnh cóng! Hãy đến gần lửa trại"
		if sim.Ambient(g.curr.Tick) >= sim.ColdAmbient { // by day: it is the snow
			long = "Tuyết lạnh cóng! Mặc Áo lông, cưỡi voi ma mút hoặc đứng gần lửa"
		}
		if g.curr.Season == sim.Winter && g.curr.Weather != sim.WeatherClear {
			long = "Rét mùa đông! Mặc Áo lông hoặc áo da, hoặc đứng gần lửa, đuốc"
		}
		out = append(out, status{statusCold, "Lạnh", long, colColdTxt})
	}
	if self.Poisoned {
		out = append(out, status{statusPoison, "Trúng độc", "Trúng độc nhện! Dùng Băng tơ nhện hoặc Bình máu lớn", colPoison})
	}
	if self.Webbed {
		out = append(out, status{statusWeb, "Dính tơ", "Dính tơ nhện — đi chậm lại", color.RGBA{220, 222, 232, 255}})
	}
	switch {
	case self.Hunger <= 0:
		out = append(out, status{statusStarving, "Chết đói", "Đang chết đói! Ăn gì đó ngay", colBad})
	case float64(self.Hunger) < sim.MaxHunger*0.2:
		out = append(out, status{statusHungry, "Đói", "", colHunger})
	}
	if me, ok := g.me(); ok && me.Bees {
		out = append(out, status{statusBees, "Ong đốt", "Bầy ong đang đuổi! Chạy thật xa tổ", colKey})
	}
	return append(out, g.buffStatuses()...)
}

// noteStatuses remembers when each status started.
func (g *Game) noteStatuses(now []status) {
	var on [numStatuses]bool
	for _, s := range now {
		on[s.kind] = true
		if !g.statusOn[s.kind] {
			g.statusAt[s.kind] = g.frame
		}
	}
	g.statusOn = on
}

// drawStatusIcon draws status k's little picture centred on (x, y).
func (g *Game) drawStatusIcon(dst *ebiten.Image, k statusKind, x, y float32, c color.RGBA) {
	switch k {
	case statusCold: // a snowflake
		for i := range 3 {
			a := float64(i) * math.Pi / 3
			dx, dy := float32(math.Cos(a))*6, float32(math.Sin(a))*6
			vector.StrokeLine(dst, x-dx, y-dy, x+dx, y+dy, 1.6, c, true)
		}
		vector.FillCircle(dst, x, y, 1.6, colWhite, true)
	case statusPoison: // a green drop with a skull's eyes
		drawDrop(dst, x, y, 13, c)
		vector.FillCircle(dst, x-1.8, y+1.5, 1.2, color.RGBA{20, 40, 10, 255}, true)
		vector.FillCircle(dst, x+1.8, y+1.5, 1.2, color.RGBA{20, 40, 10, 255}, true)
	case statusWeb: // a web
		for i := range 4 {
			a := float64(i)*math.Pi/4 + 0.2
			dx, dy := float32(math.Cos(a))*7, float32(math.Sin(a))*7
			vector.StrokeLine(dst, x-dx, y-dy, x+dx, y+dy, 1, c, true)
		}
		vector.StrokeCircle(dst, x, y, 3, 1, c, true)
		vector.StrokeCircle(dst, x, y, 6, 1, fade(c, 180), true)
	case statusStarving, statusHungry:
		b := g.art.food.Bounds()
		drawIcon(dst, g.art.food, float64(x)-float64(b.Dx())/2, float64(y)-float64(b.Dy())/2, 1, 1)
		if k == statusStarving {
			vector.StrokeLine(dst, x-6, y+6, x+6, y-6, 2, colBad, true)
		}
	case statusBuff, statusBuff + 1, statusBuff + 2, statusBuff + 3, statusBuff + 4:
		drawBuffIcon(dst, sim.BuffKind(k-statusBuff), x, y, c)
	case statusBees: // a bee
		vector.FillCircle(dst, x-2, y-4, 3, color.RGBA{230, 240, 255, 200}, true)
		vector.FillCircle(dst, x+2, y-4, 3, color.RGBA{230, 240, 255, 200}, true)
		vector.FillCircle(dst, x, y+1, 5, color.RGBA{250, 200, 40, 255}, true)
		vector.StrokeLine(dst, x-4, y, x+4, y, 1.6, color.RGBA{30, 24, 10, 255}, false)
		vector.StrokeLine(dst, x-4, y+3, x+4, y+3, 1.6, color.RGBA{30, 24, 10, 255}, false)
	}
}

// drawStatuses draws the chips from (x, y) rightwards, then, for a moment
// after one starts, its long line.
func (g *Game) drawStatuses(dst *ebiten.Image, list []status, x, y float32) {
	var shout *status
	for i := range list {
		s := &list[i]
		since := g.frame - g.statusAt[s.kind]
		if s.long != "" && since < statusShout && (shout == nil || g.statusAt[s.kind] > g.statusAt[shout.kind]) {
			shout = s
		}
		w := float32(g.textWidth(s.label)) + 34
		bg := color.RGBA{14, 16, 26, 215}
		if since < statusShout && (g.frame/20)%2 == 0 {
			bg = color.RGBA{60, 40, 30, 230}
		}
		fillRoundRect(dst, x-1, y-1, w+2, chipH+2, 8, fade(s.col, 120))
		fillRoundRect(dst, x, y, w, chipH, 7, bg)
		g.drawStatusIcon(dst, s.kind, x+13, y+chipH/2, s.col)
		g.text(dst, s.label, float64(x)+25, float64(y)+3, s.col, text.AlignStart, 1)
		x += w + 6
	}
	if shout != nil {
		a := min(1, float64(statusShout-(g.frame-g.statusAt[shout.kind]))/40)
		w := float32(g.textWidth(shout.long)) + 20
		fillRoundRect(dst, x, y, w, chipH, 7, color.RGBA{14, 16, 26, uint8(205 * a)})
		c := fade(shout.col, uint8(255*a))
		g.text(dst, shout.long, float64(x)+10, float64(y)+3, c, text.AlignStart, 1)
	}
}

// --- the HUD ----------------------------------------------------------------------------

func (g *Game) drawHUD(screen *ebiten.Image, sw, sh int) {
	self := g.curr.Self

	vs := g.vitals()
	panel(screen, vitalX, vitalY, vitalPanelW, vitalsBottom(len(vs))-vitalY)
	for i, v := range vs {
		g.drawVital(screen, v, vitalX+10, vitalY+8+float32(i*vitalRow))
	}
	st := g.statuses()
	g.noteStatuses(st)
	g.drawStatuses(screen, st, vitalX, vitalsBottom(len(vs))+6)

	g.drawSeaHUD(screen) // see searender.go
	g.drawCorner(screen, sw, g.overlay == overlayNone)
	if g.overlay == overlayNone { // the map and bag cover the bottom of the screen
		g.drawHotbar(screen, sw, sh)
		if g.showHelp {
			g.drawHelp(screen, sw, sh)
		}
	}

	if since := time.Since(g.autosaved); since < 2500*time.Millisecond {
		a := float32(1)
		if since > 1500*time.Millisecond {
			a = float32(2500*time.Millisecond-since) / float32(time.Second)
		}
		msg := "Đã tự động lưu"
		w := float32(g.textWidth(msg)) + 24
		x := float32(sw) - w - 12
		y := float32(cornerBottom(g.overlay == overlayNone)) + 8
		fillRoundRect(screen, x, y, w, 24, 8, color.RGBA{14, 16, 26, uint8(205 * a)})
		g.text(screen, msg, float64(x)+12, float64(y)+4, color.RGBA{150, 230, 140, uint8(255 * a)}, text.AlignStart, 1)
	}

	if self.Dead {
		g.dim(screen, sw, sh, 0.6)
		g.text(screen, "Bạn đã gục ngã", float64(sw/2), float64(sh/2-40), color.RGBA{236, 110, 110, 255}, text.AlignCenter, 3)
		where := "ở chỗ xuất phát"
		if self.HomeBed {
			where = "cạnh chỗ ngủ lần cuối (giường hoặc lều)"
		}
		cost := " (giữ nguyên đồ)"
		if !g.difficulty.KeepsItems() {
			cost = " (đồ nằm lại ở xác chỗ bạn chết)"
		}
		g.text(screen, "Nhấn R, E hoặc Enter để hồi sinh "+where+cost, float64(sw/2), float64(sh/2+20), colMuted, text.AlignCenter, 1)
	}
}

func (g *Game) drawNotice(screen *ebiten.Image, sw int) {
	if g.notice == "" || !time.Now().Before(g.noticeUntil) {
		return
	}
	uiBreak() // over the bag's HD icons: see iconhd.go
	w := float32(g.textWidth(g.notice)) + 28
	panel(screen, float32(sw)/2-w/2, 96, w, 28)
	g.text(screen, g.notice, float64(sw/2), 102, colWhite, text.AlignCenter, 1)
}

// --- the clock and minimap, top right -------------------------------------------------

// cornerW is the clock block's width: the minimap's frame.
const cornerW = miniW + 2*miniPad

// cornerBottom is the y just below the clock block, with or without the
// minimap in it.
func cornerBottom(withMap bool) float64 {
	if withMap {
		return miniBottom()
	}
	return miniTop - 4
}

// drawCorner draws the clock (day, phase, season, weather with its icon,
// and the day's bar) with the minimap under it, in one panel.
func (g *Game) drawCorner(screen *ebiten.Image, sw int, withMap bool) {
	tick := g.curr.Tick
	x0 := float32(sw) - 12 - cornerW
	panel(screen, x0, 12, cornerW, float32(cornerBottom(withMap))-12)
	x, w := x0+miniPad+2, float32(cornerW-2*miniPad-4)

	night := sim.PhaseOf(tick) == sim.PhaseNight
	g.drawWeatherIcon(screen, g.curr.Weather, night, x+8, 27)
	phase := map[sim.Phase]string{sim.PhaseDay: "Ban ngày", sim.PhaseDusk: "Chạng vạng", sim.PhaseNight: "Ban đêm"}[sim.PhaseOf(tick)]
	g.text(screen, fmt.Sprintf("Ngày %d", sim.DayOf(tick)), float64(x)+22, 19, colWhite, text.AlignStart, 1)
	g.text(screen, phase, float64(x+w), 19, colMuted, text.AlignEnd, 1)
	season := fmt.Sprintf("%s %d/%d", g.curr.Season.Name(), sim.SeasonDay(tick), sim.SeasonDays)
	g.text(screen, season, float64(x), 38, seasonColors[g.curr.Season%sim.NumSeasons], text.AlignStart, 1)
	g.text(screen, g.curr.Weather.Name(), float64(x+w), 38, colMuted, text.AlignEnd, 1)

	// The day's bar, a sun by day and a moon by night riding along it.
	by, bh := float32(60), float32(6)
	fillRoundRect(screen, x, by, w, bh, 3, colNight)
	fillRoundRect(screen, x, by, w*float32(sim.NightAt(tick)), bh, 3, colDusk)
	fillRoundRect(screen, x, by, w*float32(sim.DuskAt(tick)), bh, 3, colDay)
	mx, my := x+float32(sim.DayFraction(tick))*w, by+bh/2
	vector.FillCircle(screen, mx, my, 7, color.RGBA{14, 16, 26, 255}, true)
	drawSunMoon(screen, mx, my, 5, night)

	if withMap {
		g.drawMinimap(screen, float64(x0))
	}
}

// drawSunMoon draws a sun, or a moon at night, of radius r at (x, y).
func drawSunMoon(dst *ebiten.Image, x, y, r float32, night bool) {
	if night {
		vector.FillCircle(dst, x, y, r, color.RGBA{226, 232, 248, 255}, true)
		vector.FillCircle(dst, x+r*0.5, y-r*0.35, r*0.85, color.RGBA{14, 16, 26, 255}, true)
		return
	}
	vector.FillCircle(dst, x, y, r, color.RGBA{255, 226, 110, 255}, true)
	vector.FillCircle(dst, x-r*0.25, y-r*0.25, r*0.4, color.RGBA{255, 246, 196, 255}, true)
}

// drawWeatherIcon draws the weather's little picture centred on (x, y).
func (g *Game) drawWeatherIcon(dst *ebiten.Image, w sim.Weather, night bool, x, y float32) {
	cloud := func(c color.RGBA) {
		vector.FillCircle(dst, x-4, y+1, 4, c, true)
		vector.FillCircle(dst, x+1, y-2, 5, c, true)
		vector.FillCircle(dst, x+5, y+1, 4, c, true)
		vector.FillRect(dst, x-4, y+1, 9, 4, c, true)
	}
	switch w {
	case sim.WeatherRain:
		for i := range 3 {
			fx := x - 4 + float32(i)*4
			vector.StrokeLine(dst, fx, y+5, fx-1.5, y+9, 1.3, color.RGBA{120, 180, 255, 255}, true)
		}
		cloud(color.RGBA{180, 190, 206, 255})
	case sim.WeatherStorm:
		cloud(color.RGBA{110, 116, 134, 255})
		var p vector.Path
		p.MoveTo(x+1, y+3)
		p.LineTo(x-3, y+8)
		p.LineTo(x, y+8)
		p.LineTo(x-2, y+12)
		p.LineTo(x+4, y+6)
		p.LineTo(x+1, y+6)
		p.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(colKey)
		vector.FillPath(dst, &p, nil, op)
	case sim.WeatherWind:
		c := color.RGBA{220, 230, 240, 255}
		vector.StrokeLine(dst, x-7, y-3, x+3, y-3, 1.5, c, true)
		vector.StrokeCircle(dst, x+3, y-5, 2, 1.5, c, true)
		vector.StrokeLine(dst, x-7, y+1, x+6, y+1, 1.5, c, true)
		vector.StrokeLine(dst, x-5, y+5, x+2, y+5, 1.5, c, true)
		vector.StrokeCircle(dst, x+2, y+7, 2, 1.5, c, true)
	case sim.WeatherTornado:
		c := color.RGBA{190, 196, 210, 255}
		for i := range 5 {
			hw := 7 - float32(i)*1.4
			fy := y - 6 + float32(i)*3
			off := float32(math.Sin(float64(i)+float64(g.frame)/12)) * 1.2
			vector.StrokeLine(dst, x-hw+off, fy, x+hw+off, fy, 1.6, c, true)
		}
	case sim.WeatherSnow:
		cloud(color.RGBA{210, 218, 230, 255})
		for i := range 3 {
			vector.FillCircle(dst, x-4+float32(i)*4, y+8+float32(i%2)*2, 1.3, colWhite, true)
		}
	default:
		drawSunMoon(dst, x, y, 6, night)
		if !night {
			for i := range 8 {
				a := float64(i) * math.Pi / 4
				cx, cy := float32(math.Cos(a)), float32(math.Sin(a))
				vector.StrokeLine(dst, x+cx*7.5, y+cy*7.5, x+cx*9.5, y+cy*9.5, 1.3, color.RGBA{255, 226, 110, 255}, true)
			}
		}
	}
}

// --- the hotbar and the hint over it --------------------------------------------------------

// The hotbar's slots are a little bigger than the bag's.
const (
	hotPx  = 60
	hotGap = 6
)

// hotbarY is the top of the hotbar's slots on a canvas sh tall.
func hotbarY(sh int) int { return sh - hotPx - 16 }

// hotbarX0 is the left of the hotbar's first slot on a canvas sw wide.
func hotbarX0(sw int) int {
	return (sw - (sim.HotbarSlots*hotPx + (sim.HotbarSlots-1)*hotGap)) / 2
}

// drawHotbar draws the ten hotbar slots along the bottom, the held one
// outlined with its name over it, and one line of what to do above that.
func (g *Game) drawHotbar(screen *ebiten.Image, sw, sh int) {
	const n = sim.HotbarSlots
	total := n*hotPx + (n-1)*hotGap
	x0, y := float64(hotbarX0(sw)), float64(hotbarY(sh))
	drawFrame(screen, float32(x0)-10, float32(y)-10, float32(total)+20, hotPx+20)
	inv := &g.curr.Self.Inv
	for i := range n {
		x := x0 + float64(i*(hotPx+hotGap))
		st := slotPlain
		if i == g.hotSel {
			st = slotHeld
		}
		g.drawSlotSized(screen, inv[i], x, y, hotPx, st)
		top := overIcons(screen) // over the HD icons: see iconhd.go
		g.drawKeyBadge(top, hotbarLabel(i), x-3, y-3, i == g.hotSel)
		// How much of the burning torch is left.
		if i == g.hotSel && inv[i].Item == sim.ItemTorch && g.curr.Self.TorchFuel > 0 {
			f := float32(g.curr.Self.TorchFuel) / float32(sim.TorchBurnTicks)
			fillRoundRect(top, float32(x)+6, float32(y)+hotPx-7, hotPx-12, 4, 2, color.RGBA{0, 0, 0, 160})
			fillRoundRect(top, float32(x)+6, float32(y)+hotPx-7, (hotPx-12)*f, 4, 2, color.RGBA{255, 160, 60, 255})
		}
	}

	// The held item's name, always, brighter for a moment after it changes.
	ny := y - 40
	if s := inv[g.hotSel]; !s.Empty() {
		name := s.Item.Def().Name
		if s.Count > 1 {
			name += fmt.Sprintf(" ×%d", s.Count)
		}
		c := colWhite
		if time.Since(g.selChangedAt) < 1500*time.Millisecond {
			c = colKey
		}
		w := float32(g.textWidth(name)) + 24
		fillRoundRect(screen, float32(sw)/2-w/2-1, float32(ny)-1, w+2, 26, 8, fade(colKey, 90))
		fillRoundRect(screen, float32(sw)/2-w/2, float32(ny), w, 24, 7, color.RGBA{14, 16, 26, 225})
		g.text(screen, name, float64(sw)/2, ny+4, c, text.AlignCenter, 1)
	} else {
		ny += 26 // no name: the hint comes down
	}

	if key, s, c := g.contextHint(); s != "" {
		hy := ny - 24
		kw := g.textWidth(key)
		tw := kw + 8 + g.textWidth(s)
		hx := float64(sw)/2 - tw/2
		fillRoundRect(screen, float32(hx)-10, float32(hy)-2, float32(tw)+20, 21, 7, color.RGBA{14, 16, 26, 150})
		g.text(screen, key, hx, hy, colKey, text.AlignStart, 1)
		g.text(screen, s, hx+kw+8, hy, c, text.AlignStart, 1)
	}
}

// drawKeyBadge is the number key of a hotbar slot, on a little tab at its
// top left corner.
func (g *Game) drawKeyBadge(dst *ebiten.Image, key string, x, y float64, held bool) {
	bg, fg := color.RGBA{10, 12, 20, 235}, colKey
	if held {
		bg, fg = colKey, color.RGBA{30, 22, 8, 255}
	}
	fillRoundRect(dst, float32(x), float32(y), 17, 18, 5, bg)
	drawTextPlain(dst, key, x+8.5, y+1, fg, text.AlignCenter)
}

// drawTextPlain draws s without a shadow.
func drawTextPlain(dst *ebiten.Image, s string, x, y float64, clr color.Color, align text.Align) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	op.PrimaryAlign = align
	text.Draw(dst, s, uiFace, op)
}

// hotbarLabel is the key that picks hotbar slot i.
func hotbarLabel(i int) string { return fmt.Sprintf("%d", (i+1)%10) }

// contextHint is the one line over the hotbar: the key or button, what it
// does here and now, and its colour. It is the most useful line of the
// help (F1): building, the boat or beast at hand, what a click would set
// down or work on, then what the right button does with what is held.
func (g *Game) contextHint() (key, s string, c color.Color) {
	if g.curr.Self.Dead {
		return "", "", nil
	}
	if name, ok := g.buildName(); ok { // see buildpalette.go
		return "Chuột trái:", "dựng " + name + " · chuột phải hoặc Esc: thôi", colGood
	}
	if verb, ok := g.boatHint(); ok { // see searender.go
		return "E:", verb, colGood
	}
	if verb, can, ok := g.mountHint(); ok { // see ride.go
		if can {
			return "E:", verb, colGood
		}
		return "E:", verb, colMuted
	}
	if gv := g.ghost(); gv.ok {
		return "Chuột trái:", "đặt " + strings.ToLower(gv.item.Def().Name), colGood
	}
	if k, v := g.targetHint(g.hudTarget); v != "" {
		return k, v, colWhite
	}
	held := g.myHeld()
	if cmd, _ := rightClickUse(held, g.hotSel, g.useContextNow()); cmd.Kind != sim.CmdNone {
		return "Chuột phải:", useLabel(cmd, held), colGood
	}
	return "F1:", "xem các phím", colMuted
}

// targetHint is what Action does to the target outlined, if it is plain.
func (g *Game) targetHint(tg targetView) (key, s string) {
	if !tg.ok {
		return "", ""
	}
	if tg.drop {
		return "E:", "nhặt " + strings.ToLower(tg.d.Item.Def().Name)
	}
	if tg.creature {
		if tg.c.Tamed {
			return "", ""
		}
		return "Chuột trái:", "đánh " + strings.ToLower(tg.c.Kind.Name())
	}
	e := tg.e
	name := strings.ToLower(e.Kind.Name())
	switch e.Kind {
	case sim.EntityTree, sim.EntityFigTree, sim.EntityGinkgoTree, sim.EntityPalm, sim.EntityBananaTree, sim.EntityGiantTree, sim.EntityCactus:
		if !g.has(sim.ToolAxe) {
			return "", ""
		}
		if sim.IsFruitTree(e.Kind) && e.Ready {
			return "Chuột trái:", "hái quả"
		}
		return "Chuột trái:", "chặt " + name
	case sim.EntityBoulder, sim.EntityCoal, sim.EntityCopperOre, sim.EntityIronOre, sim.EntityPlatinumOre, sim.EntityDiamondOre:
		if !g.has(sim.ToolPickaxe) || !g.canMine(e.Kind) {
			return "", ""
		}
		return "Chuột trái:", "đập " + name
	case sim.EntityGrassTuft:
		return "Chuột trái:", "cắt cỏ"
	case sim.EntityBerryBush:
		return "Chuột trái:", "hái quả mọng"
	case sim.EntityTwig, sim.EntityPebble, sim.EntityMushroom, sim.EntityClam, sim.EntityReed, sim.EntityFlower:
		return "Chuột trái:", "nhặt " + name
	case sim.EntityFlora:
		f := sim.FloraKind(e.Var)
		if f.NeedsAxe() {
			if !g.has(sim.ToolAxe) {
				return "", ""
			}
			return "Chuột trái:", "chặt " + strings.ToLower(f.Name())
		}
		return "Chuột trái:", "hái " + strings.ToLower(f.Name())
	case sim.EntityChest, sim.EntityTreasure:
		return "E:", "mở " + name
	case sim.EntityCampfire:
		return "Chuột trái:", "thêm củi"
	}
	return "", ""
}

// drawHelp is the full list of keys, bottom left, while F1 is on.
func (g *Game) drawHelp(screen *ebiten.Image, sw, sh int) {
	type line struct {
		key, s string
		c      color.Color
	}
	held := g.myHeld()
	lines := []line{{"WASD", "di chuyển — nhân vật quay theo hướng đi", colMuted}}
	if name, ok := g.buildName(); ok { // see mouse.go, buildpalette.go
		lines = append(lines,
			line{"Chuột trái", "dựng " + name + " ở ô dưới con trỏ (xanh: được · đỏ: không)", colGood},
			line{"Chuột phải", "thôi xây (hoặc Esc)", colMuted})
	} else {
		lines = append(lines,
			line{"Chuột trái", "chặt, đập, hái, cắt cỏ, đánh, câu cá… thứ dưới con trỏ", colMuted},
			line{"", "giữ để làm liên tục · xa quá thì làm thứ gần nhất", colMuted},
			line{"Space", "như chuột trái, vào thứ gần nhất", colMuted})
		if gv := g.ghost(); gv.ok {
			lines = append(lines, line{"Chuột trái", "đặt " + strings.ToLower(gv.item.Def().Name) + " vào ô dưới con trỏ", colGood})
		}
		if cmd, _ := rightClickUse(held, g.hotSel, g.useContextNow()); cmd.Kind != sim.CmdNone {
			lines = append(lines, line{"Chuột phải", useLabel(cmd, held), colGood})
		} else {
			lines = append(lines, line{"Chuột phải", "dùng món đang cầm: ăn, nướng, cắm đuốc, gieo hạt…", colMuted})
		}
	}
	interact, ic := "nhặt đồ · mở rương · cưỡi thú · lên thuyền", color.Color(colMuted)
	if verb, ok := g.boatHint(); ok { // see searender.go
		interact, ic = verb, colGood
	} else if verb, can, ok := g.mountHint(); ok { // see ride.go
		interact = verb
		if can {
			ic = colGood
		}
	}
	lines = append(lines, line{"E", interact, ic}, line{"1-0", "chọn ô (lăn chuột, hoặc nhấp vào ô)", colMuted})
	n := 0
	for _, r := range recipes() {
		if g.canCraft(r) {
			n++
		}
	}
	craft, cc := "chế tạo", color.Color(colMuted)
	if n > 0 {
		craft, cc = fmt.Sprintf("chế tạo (%d món làm được)", n), colGood
	}
	lines = append(lines, line{"C", craft, cc}, line{"Tab / I", "túi đồ", colMuted})
	if g.freeBuild() { // see buildpalette.go
		lines = append(lines,
			line{"B", "bảng xây (xây tự do: không tốn gì)", colGood},
			line{"Shift + chuột phải", "gỡ công trình dưới con trỏ", colGood})
	}
	if held.Valid() {
		lines = append(lines, line{"G", "vứt 1 món đang cầm xuống đất (Shift+G: cả chồng)", colMuted})
	}
	lines = append(lines,
		line{"M", "bản đồ", colMuted},
		line{"= / −", "thu phóng (hoặc Ctrl + lăn chuột)", colMuted},
		line{"F11", "toàn màn hình", colMuted},
		line{"F1", "ẩn bảng phím này", colMuted},
		line{"Esc", "tạm dừng · lưu game", colMuted})

	const rowH = 18
	w, keyW := 0.0, 0.0
	for _, l := range lines {
		w = max(w, g.textWidth(l.s))
		keyW = max(keyW, g.textWidth(l.key)+12)
	}
	h := float64(rowH*len(lines)) + 14
	x, y := 12.0, float64(sh)-h-12
	pw := w + keyW + 26
	if hotX := float64(hotbarX0(sw)); x+pw > hotX-12 {
		y = float64(hotbarY(sh)) - 72 - h // too wide to sit beside the hotbar: above it and its lines
	}
	panel(screen, float32(x), float32(y), float32(pw), float32(h))
	for i, l := range lines {
		ly := y + 8 + float64(i*rowH)
		g.text(screen, l.key, x+12, ly, colKey, text.AlignStart, 1)
		g.text(screen, l.s, x+12+keyW, ly, l.c, text.AlignStart, 1)
	}
}

func (g *Game) drawPause(screen *ebiten.Image, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	cx := float64(sw) / 2
	top := max(16, float64(sh)/2-215)
	g.text(screen, "Tạm dừng", cx, top, colWhite, text.AlignCenter, 3)
	g.pause.draw(screen, cx, top+64, 340)
	g.text(screen, "Game tự lưu mỗi phút và khi thoát", cx, g.pause.rowY(len(g.pause.items))+8, colMuted, text.AlignCenter, 1)
}

// seasonColors colour the season's name on the clock.
var seasonColors = [sim.NumSeasons]color.RGBA{
	{150, 230, 140, 255}, {255, 214, 100, 255}, {240, 150, 70, 255}, {190, 220, 255, 255},
}

// drawFrame is the bag's and the hotbar's frame: a dark panel with a warm
// wooden rim, a thin gold inner line and little studs at the corners.
func drawFrame(dst *ebiten.Image, x, y, w, h float32) {
	fillRoundRect(dst, x-3, y-3, w+6, h+6, 12, color.RGBA{0, 0, 0, 110})
	fillRoundRect(dst, x-2, y-2, w+4, h+4, 11, color.RGBA{92, 62, 34, 245})
	fillRoundRect(dst, x-1, y-1, w+2, h+2, 10, color.RGBA{132, 92, 52, 255})
	fillRoundRect(dst, x+1, y+1, w-2, h-2, 9, color.RGBA{58, 38, 22, 255})
	fillRoundRect(dst, x+3, y+3, w-6, h-6, 7, color.RGBA{18, 20, 30, 240})
	strokeRoundRect(dst, x+3.5, y+3.5, w-7, h-7, 7, fade(colKey, 70))
	for _, c := range [][2]float32{{x + 2, y + 2}, {x + w - 2, y + 2}, {x + 2, y + h - 2}, {x + w - 2, y + h - 2}} {
		vector.FillCircle(dst, c[0], c[1], 3, color.RGBA{70, 48, 26, 255}, true)
		vector.FillCircle(dst, c[0], c[1], 2, color.RGBA{214, 176, 96, 255}, true)
	}
}
