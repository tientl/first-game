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

// The HUD, drawn at the screen's resolution in UI pixels (see uiscale.go)
// in the look of uitheme.go: the vitals and status chips top left on hide,
// the clock with the minimap top right, the hotbar along the bottom in its
// wooden frame with the held item's name and one line of what to do here
// above it. F1 shows the full list of keys.

var (
	colPanel     = color.RGBA{36, 28, 22, 215}
	colPanelEdge = color.RGBA{0, 0, 0, 90}
	colSlot      = color.RGBA{0, 0, 0, 110}
	colHealth    = color.RGBA{214, 56, 60, 255}
	colHunger    = color.RGBA{232, 148, 46, 255}
	colThirst    = color.RGBA{70, 150, 230, 255}
	colGood      = color.RGBA{150, 230, 140, 255}
	colMuted     = color.RGBA{196, 184, 162, 255}
	colKey       = color.RGBA{246, 206, 98, 255}
	colColdTxt   = color.RGBA{150, 210, 255, 255}
	colPoison    = color.RGBA{140, 230, 100, 255}
	colDay       = color.RGBA{250, 214, 96, 255}
	colDusk      = color.RGBA{236, 128, 74, 255}
	colNight     = color.RGBA{52, 62, 132, 255}
)

func (g *Game) text(dst drawTarget, s string, x, y float64, clr color.Color, align text.Align, scale float64) {
	drawText(dst, s, x, y, clr, align, scale)
}

func (g *Game) textWidth(s string) float64 { return textWidth(s, 1) }

func (g *Game) dim(dst *uiDst, sw, sh int, alpha float64) {
	dimScreen(dst, color.RGBA{0, 0, 0, uint8(alpha * 255)})
}

func fillRoundRect(dst drawTarget, x, y, w, h, r float32, clr color.Color) {
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
	fillPath(dst, &p, op)
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

func drawIcon(dst drawTarget, img *ebiten.Image, x, y, scale float64, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(alpha)
	dst.DrawImage(img, op)
}

// --- the vitals --------------------------------------------------------------------

// vital is one bar top left: health, hunger, and later thirst.
type vital struct {
	icon     uiIcon       // its picture, when item is none
	item     sim.ItemKind // or an item's icon
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

// hudFX is what the HUD remembers from frame to frame to animate: each
// vital's trail, and the world labels' fades (uilabels.go).
type hudFX struct {
	vitals [3]barTrail
	labels labelFX
}

// vitals are the bars shown, top to bottom. Thirst goes here once the sim
// has it: {uiThirst, 0, self.Thirst, sim.MaxThirst, colThirst}.
func (g *Game) vitals() []vital {
	self := g.curr.Self
	return []vital{
		{uiHeart, sim.ItemNone, float64(self.Health), sim.MaxHealth, colHealth},
		{uiHungry, sim.ItemNone, float64(self.Hunger), sim.MaxHunger, colHunger},
	}
}

// vitalsBottom is the y just under the vitals panel.
func vitalsBottom(n int) float32 { return vitalY + 10 + float32(n)*vitalRow }

// drawVital draws vital i, v, with its icon at (x, y), its value as
// "72/100" inside it: a sunken track, the fill lit like something round,
// what was just lost draining away pale, a flash at a blow, and a glow
// round it when it runs low.
func (g *Game) drawVital(dst *uiDst, i int, v vital, x, y float32) {
	frac := 0.0
	if v.max > 0 {
		frac = max(0, min(1, v.cur/v.max))
	}
	fx := g.hudFX.vitals[i%len(g.hudFX.vitals)].step(frac, g.frame)
	ix, iy := float64(x)+13, float64(y)+vitalBarH/2
	if v.item.Valid() {
		g.drawItem(dst, v.item, ix-12, iy-12, 24, 1)
	} else {
		beat := 1.0
		if v.icon == uiHeart && frac < 0.35 {
			beat += 0.08 * math.Max(0, math.Sin(float64(g.frame)*0.25)) // a heart that pounds
		}
		drawUIIcon(dst, v.icon, ix, iy, 24*beat, 1)
	}
	bx, by := float64(x)+30, float64(y)
	if frac < 0.25 {
		pulse := 0.5 + 0.5*math.Sin(float64(g.frame)*0.12)
		glow(dst, bx, by, vitalBarW, vitalBarH, vitalBarH/2, v.fill, 0.35+0.4*pulse)
	}
	well(dst, bx, by, vitalBarW, vitalBarH, vitalBarH/2)
	noteBox(dst, bx, by, vitalBarW, vitalBarH)
	drawBarFill(dst, bx+2.5, by+2.5, vitalBarW-5, vitalBarH-5, frac, v.fill, fx, g.frame)
	value := fmt.Sprintf("%d/%d", int(math.Ceil(v.cur)), int(v.max))
	uiTextW(dst, value, bx+vitalBarW/2, by+vitalBarH/2-lineUI*0.95/2, colWhite, text.AlignCenter, 0.95, wHeavy, fxOutline)
}

// drawDrop is a water drop centred on (x, y), r across.
func drawDrop(dst drawTarget, x, y, r float32, c color.RGBA) {
	fillCircle(dst, x, y+r*0.25, r*0.6, c, true)
	var p vector.Path
	p.MoveTo(x, y-r*0.85)
	p.LineTo(x+r*0.55, y+r*0.1)
	p.LineTo(x-r*0.55, y+r*0.1)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	fillPath(dst, &p, op)
	fillCircle(dst, x-r*0.2, y+r*0.1, r*0.15, color.RGBA{255, 255, 255, 170}, true)
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

// drawStatuses draws the chips from (x, y) rightwards (a leather tab rimmed
// in the status's colour, its HD picture and a short word), then, for a
// moment after one starts, its long line.
func (g *Game) drawStatuses(dst *uiDst, list []status, x, y float32) {
	var shout *status
	X, Y := float64(x), float64(y)
	for i := range list {
		s := &list[i]
		since := g.frame - g.statusAt[s.kind]
		if s.long != "" && since < statusShout && (shout == nil || g.statusAt[s.kind] > g.statusAt[shout.kind]) {
			shout = s
		}
		w := g.textWidth(s.label) + 36
		if since < statusShout {
			pulse := 0.5 + 0.5*math.Cos(float64(since)*0.16)
			glow(dst, X, Y, w, chipH, chipH/2, s.col, 0.45*pulse*(1-float64(since)/statusShout))
		}
		l := panelLooks[psLeather]
		l.body = color.RGBA{38, 29, 23, 236}
		l.rim = mix(s.col, colHideDark, 0.35)
		l.stitch = color.RGBA{}
		l.rimW = 1.25
		drawPanelLook(dst, psLeather, l, X, Y, w, chipH, chipH/2, 0.45)
		noteBox(dst, X, Y, w, chipH)
		drawUIIcon(dst, statusIcon(s.kind), X+13, Y+chipH/2, 19, 1)
		g.text(dst, s.label, X+26, Y+3, s.col, text.AlignStart, 1)
		X += w + 6
	}
	if shout != nil {
		a := min(1, float64(statusShout-(g.frame-g.statusAt[shout.kind]))/40)
		w := g.textWidth(shout.long) + 24
		plateA(dst, X, Y, w, chipH, chipH/2, a)
		c := fade(shout.col, uint8(255*a))
		g.text(dst, shout.long, X+12, Y+3, c, text.AlignStart, 1)
	}
}

// --- the HUD ----------------------------------------------------------------------------

func (g *Game) drawHUD(screen *uiDst, sw, sh int) {
	self := g.curr.Self

	vs := g.vitals()
	panel(screen, vitalX, vitalY, vitalPanelW, vitalsBottom(len(vs))-vitalY)
	for i, v := range vs {
		g.drawVital(screen, i, v, vitalX+10, vitalY+8+float32(i*vitalRow))
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
		a := 1.0
		if since > 1500*time.Millisecond {
			a = float64(2500*time.Millisecond-since) / float64(time.Second)
		}
		msg := "Đã tự động lưu"
		w := g.textWidth(msg) + 24
		x := float64(sw) - w - 12
		y := cornerBottom(g.overlay == overlayNone) + 8
		plateA(screen, x, y, w, 24, 12, a)
		g.text(screen, msg, x+12, y+4, fade(colGood, uint8(255*a)), text.AlignStart, 1)
	}

	if self.Dead {
		g.dim(screen, sw, sh, 0.6)
		drawTitle(screen, "Bạn đã gục ngã", float64(sw/2), float64(sh/2-48), 3.2, color.RGBA{236, 104, 96, 255}, color.RGBA{70, 16, 12, 255})
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

// drawNotice says the notice on a slip of parchment.
func (g *Game) drawNotice(screen *uiDst, sw int) {
	if g.notice == "" || !time.Now().Before(g.noticeUntil) {
		return
	}
	left := time.Until(g.noticeUntil).Seconds()
	a := min(1, left/0.35)
	w := g.textWidth(g.notice) + 32
	x := float64(sw)/2 - w/2
	if a < 1 {
		plateA(screen, x, noticeY, w, 28, 8, a)
		g.text(screen, g.notice, float64(sw)/2, noticeY+6, fade(colWhite, uint8(255*a)), text.AlignCenter, 1)
		return
	}
	parchment(screen, x, noticeY, w, 28)
	ink(screen, g.notice, float64(sw)/2, noticeY+6, colWhite, text.AlignCenter, 1)
}

// noticeY is the top of the notice: under the vitals' status line, which can
// run on to the middle of the screen.
const noticeY = float64(vitalY+10+2*vitalRow) + 6 + chipH + 10

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
func (g *Game) drawCorner(screen *uiDst, sw int, withMap bool) {
	tick := g.curr.Tick
	x0 := float32(sw) - 12 - cornerW
	panel(screen, x0, 12, cornerW, float32(cornerBottom(withMap))-12)
	x, w := x0+miniPad+2, float32(cornerW-2*miniPad-4)

	night := sim.PhaseOf(tick) == sim.PhaseNight
	if g.curr.Weather == sim.WeatherTornado || g.curr.Weather == sim.WeatherClear && !night {
		g.drawWeatherIcon(screen, g.curr.Weather, night, x+8, 27) // it moves
	} else {
		drawBaked(screen, fmt.Sprintf("weather %d %v", g.curr.Weather, night), float64(x)-4, 15, 24, 24, 2, func(b *uiDst) {
			g.drawWeatherIcon(b, g.curr.Weather, night, 12, 12)
		})
	}
	phase := map[sim.Phase]string{sim.PhaseDay: "Ban ngày", sim.PhaseDusk: "Chạng vạng", sim.PhaseNight: "Ban đêm"}[sim.PhaseOf(tick)]
	uiTextW(screen, fmt.Sprintf("Ngày %d", sim.DayOf(tick)), float64(x)+22, 19, colWhite, text.AlignStart, 1.05, 860, fxShadow)
	g.text(screen, phase, float64(x+w), 19, colMuted, text.AlignEnd, 1)
	season := fmt.Sprintf("%s %d/%d", g.curr.Season.Name(), sim.SeasonDay(tick), sim.SeasonDays)
	g.text(screen, season, float64(x), 38, seasonColors[g.curr.Season%sim.NumSeasons], text.AlignStart, 1)
	g.text(screen, g.curr.Weather.Name(), float64(x+w), 38, colMuted, text.AlignEnd, 1)

	// The day's bar, sunk in the hide: day, dusk and night, with a sun by
	// day and a moon by night riding along it.
	by, bh := float32(59), float32(8)
	nightAt, duskAt := float32(sim.NightAt(tick)), float32(sim.DuskAt(tick))
	drawBaked(screen, fmt.Sprintf("daybar %.0f %.3f %.3f", w, nightAt, duskAt), float64(x)-1, float64(by)-1, float64(w)+2, float64(bh)+2, 1, func(b *uiDst) {
		well(b, 0, 0, float64(w)+2, float64(bh)+2, float64(bh)/2+1)
		fillRoundRect(b, 1, 1, w, bh, bh/2, mix(colNight, colHideDark, 0.2))
		fillRoundRect(b, 1, 1, w*nightAt, bh, bh/2, colDusk)
		fillRoundRect(b, 1, 1, w*duskAt, bh, bh/2, colDay)
		fillRect(b, 1+bh/2, 2, w-bh, float32(b.hair()), color.RGBA{255, 255, 255, 70}, true)
	})
	mx, my := x+float32(sim.DayFraction(tick))*w, by+bh/2
	if !night {
		glow(screen, float64(mx)-5, float64(my)-5, 10, 10, 5, color.RGBA{255, 214, 110, 255}, 0.7)
	}
	fillCircle(screen, mx, my, 7.5, colHideDark, true)
	drawSunMoon(screen, mx, my, 5.5, night)

	if withMap {
		g.drawMinimap(screen, float64(x0))
	}
}

// drawSunMoon draws a sun, or a moon at night, of radius r at (x, y).
func drawSunMoon(dst drawTarget, x, y, r float32, night bool) {
	if night {
		fillCircle(dst, x, y, r, color.RGBA{226, 232, 248, 255}, true)
		fillCircle(dst, x+r*0.5, y-r*0.35, r*0.85, colHideDark, true)
		return
	}
	fillCircle(dst, x, y, r, color.RGBA{255, 214, 96, 255}, true)
	fillCircle(dst, x-r*0.12, y-r*0.12, r*0.72, color.RGBA{255, 232, 140, 255}, true)
	fillCircle(dst, x-r*0.3, y-r*0.3, r*0.32, color.RGBA{255, 250, 220, 255}, true)
}

// drawWeatherIcon draws the weather's little picture centred on (x, y).
func (g *Game) drawWeatherIcon(dst drawTarget, w sim.Weather, night bool, x, y float32) {
	cloud := func(c color.RGBA) {
		shade := lighten(c, -46)
		fillCircle(dst, x-4, y+1.6, 4, shade, true)
		fillCircle(dst, x+1, y-1.4, 5, shade, true)
		fillCircle(dst, x+5, y+1.6, 4, shade, true)
		fillCircle(dst, x-4, y+1, 4, c, true)
		fillCircle(dst, x+1, y-2, 5, c, true)
		fillCircle(dst, x+5, y+1, 4, c, true)
		fillRect(dst, x-4, y+1, 9, 4, c, true)
		fillCircle(dst, x-0.5, y-3.4, 2, lighten(c, 30), true)
	}
	switch w {
	case sim.WeatherRain:
		for i := range 3 {
			fx := x - 4 + float32(i)*4
			strokeLine(dst, fx, y+5, fx-1.5, y+9, 1.3, color.RGBA{120, 180, 255, 255}, true)
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
		fillPath(dst, &p, op)
	case sim.WeatherWind:
		c := color.RGBA{220, 230, 240, 255}
		strokeLine(dst, x-7, y-3, x+3, y-3, 1.5, c, true)
		strokeCircle(dst, x+3, y-5, 2, 1.5, c, true)
		strokeLine(dst, x-7, y+1, x+6, y+1, 1.5, c, true)
		strokeLine(dst, x-5, y+5, x+2, y+5, 1.5, c, true)
		strokeCircle(dst, x+2, y+7, 2, 1.5, c, true)
	case sim.WeatherTornado:
		c := color.RGBA{190, 196, 210, 255}
		for i := range 5 {
			hw := 7 - float32(i)*1.4
			fy := y - 6 + float32(i)*3
			off := float32(math.Sin(float64(i)+float64(g.frame)/12)) * 1.2
			strokeLine(dst, x-hw+off, fy, x+hw+off, fy, 1.6, c, true)
		}
	case sim.WeatherSnow:
		cloud(color.RGBA{210, 218, 230, 255})
		for i := range 3 {
			fillCircle(dst, x-4+float32(i)*4, y+8+float32(i%2)*2, 1.3, colWhite, true)
		}
	default:
		if !night {
			for i := range 8 {
				a := float64(i)*math.Pi/4 + float64(g.frame)*0.004
				cx, cy := float32(math.Cos(a)), float32(math.Sin(a))
				strokeLine(dst, x+cx*7.6, y+cy*7.6, x+cx*10, y+cy*10, 1.5, color.RGBA{255, 214, 110, 255}, true)
			}
		}
		drawSunMoon(dst, x, y, 6, night)
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

// drawHotbar draws the ten hotbar slots along the bottom in a wooden frame,
// the held one lit warm with its name over it, and one line of what to do
// above that.
func (g *Game) drawHotbar(screen *uiDst, sw, sh int) {
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
		g.drawKeyBadge(screen, hotbarLabel(i), x-3, y-3, i == g.hotSel)
		// How much of the burning torch is left.
		if i == g.hotSel && inv[i].Item == sim.ItemTorch && g.curr.Self.TorchFuel > 0 {
			f := float64(g.curr.Self.TorchFuel) / float64(sim.TorchBurnTicks)
			well(screen, x+6, y+hotPx-8, hotPx-12, 5, 2.5)
			drawBarFill(screen, x+6.5, y+hotPx-7.5, hotPx-13, 4, f, color.RGBA{255, 150, 50, 255}, barFX{}, g.frame)
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
		fresh := time.Since(g.selChangedAt) < 1500*time.Millisecond
		if fresh {
			c = colKey
		}
		w := g.textWidth(name) + 28
		if fresh {
			glow(screen, float64(sw)/2-w/2, ny, w, 24, 12, colAmberGlow, 0.25)
		}
		plate(screen, float64(sw)/2-w/2, ny, w, 24, 12)
		g.text(screen, name, float64(sw)/2, ny+4, c, text.AlignCenter, 1)
	} else {
		ny += 26 // no name: the hint comes down
	}

	if key, s, c := g.contextHint(); s != "" {
		hy := ny - 24
		kw := g.textWidth(key)
		tw := kw + 8 + g.textWidth(s)
		hx := float64(sw)/2 - tw/2
		plateA(screen, hx-12, hy-2.5, tw+24, 21, 10.5, 0.62)
		g.text(screen, key, hx, hy, colKey, text.AlignStart, 1)
		g.text(screen, s, hx+kw+8, hy, c, text.AlignStart, 1)
	}
}

// drawKeyBadge is the number key of a hotbar slot, on a little tab of bone
// at its top left corner; amber on the held slot.
func (g *Game) drawKeyBadge(dst *uiDst, key string, x, y float64, held bool) {
	const w, h = 17.0, 17.0
	bg, rim, fg := color.RGBA{30, 23, 18, 238}, color.RGBA{120, 92, 62, 255}, colBoneText
	if held {
		bg, rim, fg = colAmber, color.RGBA{255, 232, 170, 255}, color.RGBA{48, 28, 8, 255}
	}
	noteBox(dst, x, y, w, h)
	drawBaked(dst, fmt.Sprintf("badge %s %v", key, held), x, y, w, h, 1, func(b *uiDst) {
		fillRoundRect(b, -0.5, -0.5, w+1, h+1, 5.5, color.RGBA{0, 0, 0, 120})
		fillRoundRect(b, 0, 0, w, h, 5, rim)
		hair := float32(b.hair())
		fillRoundRect(b, hair, hair, w-2*hair, h-2*hair, 5-hair, bg)
		uiTextW(b, key, w/2, h/2-lineUI*0.85/2, fg, text.AlignCenter, 0.85, 900, fxPlain)
	})
}

// colBoneText is light bone, for small text on dark tabs.
var colBoneText = color.RGBA{236, 224, 196, 255}

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
	if g.curr.Self.Inv.Count(sim.ItemShield) > 0 { // see shieldart.go
		return "Chuột phải / Q:", "giữ để giơ khiên đỡ đòn", colGood
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

// drawHelp is the full list of keys, bottom left, while F1 is on: a sheet
// of parchment.
func (g *Game) drawHelp(screen *uiDst, sw, sh int) {
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
		line{"Esc", "tạm dừng · lưu game · cài đặt", colMuted})

	const rowH = 18
	w, keyW := 0.0, 0.0
	for _, l := range lines {
		w = max(w, textWidthW(l.s, 1, wText+40))
		keyW = max(keyW, textWidthW(l.key, 1, wText+80)+12)
	}
	h := float64(rowH*len(lines)) + 16
	x, y := 12.0, float64(sh)-h-12
	pw := w + keyW + 30
	if hotX := float64(hotbarX0(sw)); x+pw > hotX-12 {
		y = float64(hotbarY(sh)) - 72 - h // too wide to sit beside the hotbar: above it and its lines
	}
	parchment(screen, x, y, pw, h)
	for i, l := range lines {
		ly := y + 9 + float64(i*rowH)
		uiTextW(screen, l.key, x+14, ly, inkOf(colKey), text.AlignStart, 1, wText+80, fxPlain)
		ink(screen, l.s, x+14+keyW, ly, l.c, text.AlignStart, 1)
	}
}

func (g *Game) drawPause(screen *uiDst, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	if g.opts != nil {
		g.opts.draw(screen) // see settingsui.go
		return
	}
	cx := float64(sw) / 2
	top := max(16, float64(sh)/2-215)
	drawTitle(screen, "Tạm dừng", cx, top, 3, colBoneText, color.RGBA{60, 36, 20, 255})
	g.pause.draw(screen, cx, top+64, 340)
	g.text(screen, "Game tự lưu mỗi phút và khi thoát", cx, g.pause.rowY(len(g.pause.items))+8, colMuted, text.AlignCenter, 1)
}

// seasonColors colour the season's name on the clock.
var seasonColors = [sim.NumSeasons]color.RGBA{
	{150, 230, 140, 255}, {255, 214, 100, 255}, {240, 150, 70, 255}, {190, 220, 255, 255},
}
