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
// in the look of uitheme.go, small so the world is what is seen (the
// layout is in hudlayout.go): thin vital bars and round status chips top
// left on a little see-through hide; the round minimap in the ring of the
// day top right, with the day, season and weather under it; the hotbar
// along the bottom in its wooden frame, with one line over it, the held
// item's name for a moment or else the one hint that matters. F1 shows the
// full list of keys.

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

// vital is one bar top left: health, hunger, later thirst, and at sea a
// swimmer's strength and the boat's.
type vital struct {
	id       int          // which: vitalHealth…, for its trail
	icon     uiIcon       // its picture, when item is none
	item     sim.ItemKind // or an item's icon
	cur, max float64
	fill     color.RGBA
}

// The vitals, for their trails.
const (
	vitalHealth = iota
	vitalHunger
	vitalBreath // a swimmer's strength: see searender.go
	vitalBoat
)

// The layout is in hudlayout.go. vitalValueScale is the size of the
// numbers inside a bar.
const vitalValueScale = 0.6

// hudFX is what the HUD remembers from frame to frame to animate: each
// vital's trail, the world labels' fades (uilabels.go), the statuses seen,
// the notices (hudtoasts.go), the line over the hotbar and the tip under
// the pointer.
type hudFX struct {
	vitals     [maxVitals]barTrail
	labels     labelFX
	statusSeen [numStatuses]bool // ever on
	statusLast [numStatuses]int  // the frame each was last on
	buffFull   [sim.NumBuffs]int32
	notes      noteFX
	held       heldFX
	hint       hintFX
	tip        hudTip
}

// vitals are the bars shown, top to bottom. Thirst goes here once the sim
// has it: {vitalThirst, uiThirst, 0, self.Thirst, sim.MaxThirst, colThirst}.
func (g *Game) vitals() []vital {
	self := g.curr.Self
	vs := []vital{
		{vitalHealth, uiHeart, sim.ItemNone, float64(self.Health), sim.MaxHealth, colHealth},
		{vitalHunger, uiHungry, sim.ItemNone, float64(self.Hunger), sim.MaxHunger, colHunger},
	}
	return append(vs, g.seaVitals()...) // see searender.go
}

// drawVital draws v with its picture at (x, y) and its thin bar beside it,
// its value small inside ("72/100"): a sunken track, the fill lit like
// something round, what was just lost draining away pale, a flash at a
// blow, and a glow round it when it runs low.
func (g *Game) drawVital(dst *uiDst, v vital, x, y float64) {
	frac := 0.0
	if v.max > 0 {
		frac = max(0, min(1, v.cur/v.max))
	}
	fx := g.hudFX.vitals[v.id%maxVitals].step(frac, g.frame)
	ix, iy := x+vitalIcon/2, y+vitalBarH/2
	switch {
	case v.item.Valid():
		g.drawItem(dst, v.item, ix-vitalIcon/2-1, iy-vitalIcon/2-1, vitalIcon+2, 1)
	case v.id == vitalBreath:
		drawDrop(dst, float32(ix), float32(iy), 12, v.fill)
	default:
		beat := 1.0
		if v.icon == uiHeart && frac < 0.35 {
			beat += 0.1 * math.Max(0, math.Sin(float64(g.frame)*0.25)) // a heart that pounds
		}
		drawUIIcon(dst, v.icon, ix, iy, (vitalIcon+1)*beat, 1)
	}
	bx, by := x+vitalIcon+4, y
	if frac < 0.25 {
		pulse := 0.5 + 0.5*math.Sin(float64(g.frame)*0.12)
		glow(dst, bx, by, vitalBarW, vitalBarH, vitalBarH/2, v.fill, 0.3+0.35*pulse)
	}
	well(dst, bx, by, vitalBarW, vitalBarH, vitalBarH/2)
	noteBox(dst, bx, by, vitalBarW, vitalBarH)
	drawBarFill(dst, bx+1.5, by+1.5, vitalBarW-3, vitalBarH-3, frac, v.fill, fx, g.frame)
	value := fmt.Sprintf("%d/%d", int(math.Ceil(v.cur)), int(v.max))
	if v.id == vitalBreath {
		value = fmt.Sprintf("%d%%", int(math.Ceil(frac*100)))
	}
	uiTextW(dst, value, bx+vitalBarW/2, by+vitalBarH/2-lineUI*vitalValueScale/2, colWhite, text.AlignCenter, vitalValueScale, 880, fxOutline)
}

// hudHide is the look of the HUD's panels: dark hide, see-through enough
// to keep the world in view, a thin rim, faint stitches.
var hudHide = panelLook{
	body:   color.RGBA{32, 25, 20, 200},
	rim:    color.RGBA{88, 64, 44, 235},
	rimW:   1.5,
	stitch: color.RGBA{196, 156, 102, 55},
}

// hudPanel draws a panel of the HUD in hudHide.
func hudPanel(dst *uiDst, x, y, w, h, r float64) {
	drawPanelLook(dst, psHide, hudHide, x, y, w, h, r, 0.35)
	noteBox(dst, x, y, w, h)
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
	statusStun // what the beasts' moves leave: see skillhud.go
	statusBleed
	statusSlow
	statusPinned
	statusHeld  // the boat in a giant squid's arms: see searender.go
	statusWind  // the wind, while sailing: an arrow, not a picture
	statusBuff  // the first of the buffs of dishes, one for each sim.BuffKind; see cookrender.go
	numStatuses = statusBuff + statusKind(sim.NumBuffs)
)

// status is one effect on the player: a chip under the bars (its label
// shows under the pointer), and the long line said for a moment when it
// starts.
type status struct {
	kind  statusKind
	label string
	long  string
	col   color.RGBA
}

// statusShout is how long, in frames, the long line shows after a status
// starts; statusQuiet, how long a status must have been gone for it to be
// said again (a cold that comes and goes at the edge of a fire's warmth is
// said once).
const (
	statusShout = 4 * 60
	statusQuiet = 45 * 60
)

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
	out = append(out, skillStatuses(self)...) // see skillhud.go
	out = append(out, g.seaStatuses()...)     // see searender.go
	return append(out, g.buffStatuses()...)
}

// noteStatuses remembers when each status started: one back after only a
// moment away starts quietly, its long line not said again.
func (g *Game) noteStatuses(now []status) {
	var on [numStatuses]bool
	fx := &g.hudFX
	for _, s := range now {
		on[s.kind] = true
		if !g.statusOn[s.kind] {
			if !fx.statusSeen[s.kind] || g.frame-fx.statusLast[s.kind] > statusQuiet {
				g.statusAt[s.kind] = g.frame
			} else {
				g.statusAt[s.kind] = g.frame - statusShout
			}
		}
		fx.statusSeen[s.kind], fx.statusLast[s.kind] = true, g.frame
	}
	g.statusOn = on
	// The buffs' rings run from the time each had when it began (or was
	// topped up by another dish).
	for b, t := range g.curr.Self.Cook.Buffs {
		switch {
		case t <= 0:
			fx.buffFull[b] = 0
		case t > fx.buffFull[b]:
			fx.buffFull[b] = t
		}
	}
}

// statusTimer is how much of status k's time is left, 1 to 0, for a status
// that runs out (a dish's buff); ok is false for the rest.
func (g *Game) statusTimer(k statusKind) (left float64, ok bool) {
	if k < statusBuff || k >= numStatuses {
		return 0, false
	}
	b := int(k - statusBuff)
	full := g.hudFX.buffFull[b]
	if full <= 0 {
		return 0, false
	}
	return max(0, min(1, float64(g.curr.Self.Cook.Buffs[b])/float64(full))), true
}

// drawStatuses draws the chips from (x, y) in rows as wide as the vitals:
// each a small round leather tab, its picture, its rim in the status's
// colour (for a buff, running down like a clock as it wears off). For a
// moment after one starts it glows and its long line shows under the
// chips. Under the pointer a chip says its name.
func (g *Game) drawStatuses(dst *uiDst, list []status, x, y float64) {
	var shout *status
	per := chipsPerRow()
	for i := range list {
		s := &list[i]
		since := g.frame - g.statusAt[s.kind]
		if s.long != "" && since < statusShout && (shout == nil || g.statusAt[s.kind] > g.statusAt[shout.kind]) {
			shout = s
		}
		cx := x + float64(i%per*(chipD+chipGap)) + chipD/2
		cy := y + float64(i/per*(chipD+chipGap)) + chipD/2
		g.drawChip(dst, *s, cx, cy, since)
		if g.hovered(cx-chipD/2, cy-chipD/2, chipD, chipD) {
			g.hudFX.tip = hudTip{s: s.label, x: cx, y: cy + chipD/2 + 3, ok: true}
		}
	}
	if shout != nil {
		g.drawStatusToast(dst, *shout, x, y+float64((len(list)+per-1)/per*(chipD+chipGap)), g.frame-g.statusAt[shout.kind]) // see hudtoasts.go
	}
}

// drawChip draws status s's chip centred on (cx, cy), since frames after
// it began.
func (g *Game) drawChip(dst *uiDst, s status, cx, cy float64, since int) {
	const r = chipD / 2
	fresh := since < statusShout && s.kind != statusWind
	if fresh {
		pulse := 0.5 + 0.5*math.Cos(float64(since)*0.16)
		glow(dst, cx-r, cy-r, chipD, chipD, r, s.col, 0.5*pulse*(1-float64(since)/statusShout))
	}
	l := panelLooks[psLeather]
	l.body = color.RGBA{30, 24, 19, 228}
	l.rim = mix(s.col, colHideDark, 0.3)
	l.stitch = color.RGBA{}
	l.rimW = 1.4
	left, timed := g.statusTimer(s.kind)
	if timed {
		l.rim = color.RGBA{56, 44, 34, 240} // the clock's face: the rim is drawn round it
	}
	drawPanelLook(dst, psLeather, l, cx-r, cy-r, chipD, chipD, r, 0.4)
	if s.kind == statusWind {
		g.drawWindArrow(dst, cx, cy, 7) // see searender.go
	} else {
		drawUIIcon(dst, statusIcon(s.kind), cx, cy, 14, 1)
	}
	if timed && left > 0 {
		strokeArc(dst, cx, cy, r-0.8, -math.Pi/2, -math.Pi/2+2*math.Pi*left, 1.7, s.col)
	}
}

// strokeArc strokes the arc of radius r round (cx, cy) from angle a0 to a1
// (clockwise on the screen), sw wide, in c.
func strokeArc(dst *uiDst, cx, cy, r, a0, a1, sw float64, c color.RGBA) {
	if a1 <= a0 {
		return
	}
	var p vector.Path
	p.Arc(float32(dst.X(cx)), float32(dst.Y(cy)), float32(r*dst.k), float32(a0), float32(a1), vector.Clockwise)
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst.img, &p, &vector.StrokeOptions{Width: float32(sw * dst.k)}, op)
}

// --- the HUD ----------------------------------------------------------------------------

func (g *Game) drawHUD(screen *uiDst, sw, sh int) {
	self := g.curr.Self
	g.hudFX.tip = hudTip{}

	vs := g.vitals()
	hudPanel(screen, vitalX, vitalY, vitalPanelW, vitalsBottom(len(vs))-vitalY, 8)
	for i, v := range vs {
		g.drawVital(screen, v, vitalX+vitalPad, vitalY+vitalPad+float64(i*vitalRow))
	}
	st := g.statuses()
	g.noteStatuses(st)
	g.drawStatuses(screen, st, vitalX, chipsTop(len(vs)))
	g.drawTutorial(screen, len(vs), st) // see tutorialdraw.go

	withMap := g.overlay == overlayNone
	g.drawCorner(screen, sw, withMap)
	if withMap { // the map and bag cover the bottom of the screen
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
		const s = 0.8
		msg := "Đã tự động lưu"
		w := textWidth(msg, s) + 16
		_, cave := g.caveLevel()
		x := float64(sw) - hudM - w
		y := cornerBottom(withMap, cave) + 5
		plateA(screen, x, y, w, infoH, infoH/2, 0.9*a)
		uiText(screen, msg, x+8, y+(infoH-lineUI*s)/2, fade(colGood, uint8(255*a)), text.AlignStart, s, fxShadow)
	}
	g.drawTip(screen, sw, sh) // see hudtoasts.go

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

// --- the clock and minimap, top right -------------------------------------------------

// drawCorner draws the top right corner: the round minimap in the ring of
// the day (day, dusk and night round it, a sun or a moon riding on it at
// the hour), and under it one line of the day, the season and the weather;
// underground, the level. With a window open there is no minimap, and the
// line goes up into the corner.
func (g *Game) drawCorner(screen *uiDst, sw int, withMap bool) {
	tick := g.curr.Tick
	night := sim.PhaseOf(tick) == sim.PhaseNight
	cx, _ := miniCentre(sw)
	if withMap {
		g.drawDayRing(screen, sw, tick, night)
		g.drawMinimap(screen, sw) // see uimap.go
	}
	y := infoTop(withMap)
	g.drawDayLine(screen, cx, y, tick, night)
	g.drawMiniLevel(screen, cx, y+infoH+3) // see uimap.go
}

// The ring of the day: the angle of the hour f (a share of the day) on the
// ring, with the middle of the light (between dawn and nightfall) at the
// top, so the sun climbs up the left and goes down the right, and the
// night runs along the bottom.
func dayAngle(f, nightAt float64) float64 {
	return -math.Pi/2 + 2*math.Pi*(f-nightAt/2)
}

// colNightRing is the night on the ring: lighter than the night's colour,
// to show on the dark hide.
var colNightRing = color.RGBA{92, 108, 190, 255}

// drawDayRing draws the dark disc the minimap sits in, the ring of the day
// round it, and the sun or moon on it at the hour.
func (g *Game) drawDayRing(screen *uiDst, sw int, tick uint64, night bool) {
	cx, cy := miniCentre(sw)
	const R = cornerW / 2           // the disc
	const rr = miniD/2 + miniRing/2 // the ring's middle
	nightAt, duskAt := sim.NightAt(tick), sim.DuskAt(tick)
	drawBaked(screen, fmt.Sprintf("dayring %.3f %.3f", nightAt, duskAt), cx-R, cy-R, 2*R, 2*R, 6, func(b *uiDst) {
		c := float32(R)
		fillCircle(b, c+0.6, c+1.6, R, color.RGBA{0, 0, 0, 70}, true) // its shadow
		fillCircle(b, c, c, R, color.RGBA{30, 24, 19, 222}, true)
		strokeCircle(b, c, c, R-float32(b.hair())/2, float32(b.hair()), color.RGBA{12, 9, 7, 200}, true)
		strokeCircle(b, c, c, R-1.4, float32(b.hair()), color.RGBA{120, 92, 62, 110}, true)
		const gap = 0.035 // between the parts
		a := func(f float64) float64 { return dayAngle(f, nightAt) }
		track := color.RGBA{14, 11, 9, 200}
		strokeArc(b, R, R, rr, 0, 2*math.Pi, 4.6, track)
		strokeArc(b, R, R, rr, a(0)+gap, a(duskAt)-gap, 3, colDay)
		strokeArc(b, R, R, rr, a(duskAt)+gap, a(nightAt)-gap, 3, colDusk)
		strokeArc(b, R, R, rr, a(nightAt)+gap, a(1)-gap, 3, colNightRing)
	})
	ang := dayAngle(sim.DayFraction(tick), nightAt)
	mx, my := cx+rr*math.Cos(ang), cy+rr*math.Sin(ang)
	if !night {
		glow(screen, mx-5, my-5, 10, 10, 5, color.RGBA{255, 214, 110, 255}, 0.75)
	}
	fillCircle(screen, float32(mx), float32(my), 6.2, colHideDark, true)
	drawSunMoon(screen, float32(mx), float32(my), 4.6, night)
}

// seasonShort are the seasons' names on the corner's line.
var seasonShort = [sim.NumSeasons]string{"Xuân", "Hạ", "Thu", "Đông"}

// dayLineScale is the size of the corner's line.
const dayLineScale = 0.8

// drawDayLine draws the line under the minimap, centred on cx with its top
// at y: the weather's picture, the day, and the season with its day. Under
// the pointer it says the weather and the hour in words.
func (g *Game) drawDayLine(screen *uiDst, cx, y float64, tick uint64, night bool) {
	const s = dayLineScale
	day := fmt.Sprintf("Ngày %d", sim.DayOf(tick))
	season := fmt.Sprintf("%s %d/%d", seasonShort[g.curr.Season%sim.NumSeasons], sim.SeasonDay(tick), sim.SeasonDays)
	const dot = " · "
	wd, wdot, ws := textWidthW(day, s, wHeavy), textWidth(dot, s), textWidth(season, s)
	const icon = 15
	w := 6 + icon + 3 + wd + wdot + ws + 8
	x := math.Round(min(cx-w/2, cx+cornerW/2-w)) // a long one keeps to the margin
	plateA(screen, x, y, w, infoH, infoH/2, 0.92)
	ix, iy := x+6+icon/2, y+infoH/2
	weather := g.curr.Weather
	if weather == sim.WeatherTornado || weather == sim.WeatherClear && !night {
		g.drawWeatherIcon(screen.view(ix, iy, 0.62), weather, night, 0, 0) // it moves
	} else {
		drawBaked(screen, fmt.Sprintf("weather small %d %v", weather, night), ix-8, iy-8, 16, 16, 2, func(b *uiDst) {
			g.drawWeatherIcon(b.view(8, 8, 0.62), weather, night, 0, 0)
		})
	}
	ty := y + (infoH-lineUI*s)/2
	tx := x + 6 + icon + 3
	tx += uiTextW(screen, day, tx, ty, colWhite, text.AlignStart, s, wHeavy, fxShadow)
	tx += uiText(screen, dot, tx, ty, colMuted, text.AlignStart, s, fxShadow)
	uiText(screen, season, tx, ty, seasonColors[g.curr.Season%sim.NumSeasons], text.AlignStart, s, fxShadow)
	if g.hovered(x, y, w, infoH) {
		phase := map[sim.Phase]string{sim.PhaseDay: "ban ngày", sim.PhaseDusk: "chạng vạng", sim.PhaseNight: "ban đêm"}[sim.PhaseOf(tick)]
		g.hudFX.tip = hudTip{s: g.curr.Weather.Name() + " · " + phase + " · " + g.curr.Season.Name(), x: cx, y: y + infoH + 3, ok: true}
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

// The hotbar's slots: small enough that the bar takes about half the
// screen's width (60 made it two thirds, too heavy over the world), big
// enough for the HD icons and their counts to read.
const (
	hotPx  = 44
	hotGap = 4
)

// hotbarY is the top of the hotbar's slots on a canvas sh tall.
func hotbarY(sh int) int { return sh - hotPx - 16 }

// hotbarX0 is the left of the hotbar's first slot on a canvas sw wide.
func hotbarX0(sw int) int {
	return (sw - (sim.HotbarSlots*hotPx + (sim.HotbarSlots-1)*hotGap)) / 2
}

// drawHotbar draws the ten hotbar slots along the bottom in a wooden frame,
// the held one lit warm, and the one line over them (drawHotbarLine).
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

	g.drawHotbarLine(screen, sw, sh, inv[g.hotSel])
}

// heldFX remembers what was held, to show its name for a moment after it
// changes.
type heldFX struct {
	slot int
	item sim.ItemKind
	at   time.Time
}

// heldShow is how long the held item's name shows after it changes, and
// heldFade how long it then takes to go.
const (
	heldShow = 2200 * time.Millisecond
	heldFade = 600 * time.Millisecond
)

// hintFX is the hint over the hotbar as it fades in and out: the line
// shown, how much it shows, and when it came.
type hintFX struct {
	key, s string
	c      color.Color
	a      float64
	at     int // the frame it came
	frame  int // the frame last stepped
}

// hintIdle is how long, in frames, the hint with nothing better to say
// (F1: the keys) stays before it fades.
const hintIdle = 8 * 60

// step moves the hint on to (key, s, c) at frame now and gives how much to
// show it: it fades in when it comes and out when it goes, quickly across
// to another.
func (h *hintFX) step(key, s string, c color.Color, now int) float64 {
	dt := max(0, min(10, now-h.frame))
	h.frame = now
	if s != "" && (key != h.key || s != h.s) {
		if h.s != "" {
			h.a = min(h.a, 0.3)
		}
		h.key, h.s, h.c, h.at = key, s, c, now
	}
	want := 0.0
	if s != "" && !(key == "F1:" && now-h.at > hintIdle) {
		want = 1
	}
	if uiProbe != nil {
		h.a = want // the interface's tests see it as it settles
	}
	step := float64(dt) / 9
	if h.a < want {
		h.a = min(want, h.a+step)
	} else {
		h.a = max(want, h.a-step)
	}
	return smooth(h.a)
}

// drawHotbarLine draws the one line over the hotbar: the held item's name
// for a moment after it changes, then the hint, never both at once.
func (g *Game) drawHotbarLine(screen *uiDst, sw, sh int, held sim.Slot) {
	y := lineTop(sh)
	cx := float64(sw) / 2

	f := &g.hudFX.held
	if held.Item != f.item || g.hotSel != f.slot {
		f.slot, f.item, f.at = g.hotSel, held.Item, time.Now()
	}
	if g.selChangedAt.After(f.at) {
		f.at = g.selChangedAt
	}
	nameA := 0.0
	if age := time.Since(f.at); !held.Empty() && age < heldShow+heldFade {
		nameA = 1 - smooth(max(0, float64(age-heldShow))/float64(heldFade))
		const s = 0.88
		name := held.Item.Def().Name
		if held.Count > 1 {
			name += fmt.Sprintf(" ×%d", held.Count)
		}
		c := colWhite
		if age < 1200*time.Millisecond {
			c = colKey
		}
		w := textWidth(name, s) + 22
		if age < 900*time.Millisecond {
			glow(screen, cx-w/2, y, w, lineH, lineH/2, colAmberGlow, 0.22*nameA)
		}
		plateA(screen, cx-w/2, y, w, lineH, lineH/2, 0.85*nameA)
		uiText(screen, name, cx, y+(lineH-lineUI*s)/2, fade(c, uint8(255*nameA)), text.AlignCenter, s, fxShadow)
	}

	key, s, c := g.contextHint()
	a := g.hudFX.hint.step(key, s, c, g.frame) * (1 - nameA)
	h := &g.hudFX.hint
	if a <= 0.02 || h.s == "" {
		return
	}
	const ts = 0.85
	kw := textWidthW(h.key, ts, wHeavy)
	tw := kw + 6 + textWidth(h.s, ts)
	hx := math.Round(cx - tw/2)
	ty := y + (lineH-lineUI*ts)/2
	plateA(screen, hx-11, y, tw+22, lineH, lineH/2, 0.82*a)
	uiTextW(screen, h.key, hx, ty, fade(colKey, uint8(255*a)), text.AlignStart, ts, wHeavy, fxShadow)
	hc := colWhite
	if rc, ok := h.c.(color.RGBA); ok {
		hc = rc
	}
	uiText(screen, h.s, hx+kw+6, ty, fade(hc, uint8(255*a)), text.AlignStart, ts, fxShadow)
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
	if verb, ok := g.pickHint(g.hudTarget, g.pick.view); ok { // E picks up first; see pickup.go
		return "E:", verb, colWhite
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
// Piles and loose things are E's: see aimHint, pickup.go.
func (g *Game) targetHint(tg targetView) (key, s string) {
	if !tg.ok || tg.drop {
		return "", ""
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
			line{"", "giữ để làm liên tục · xa quá thì làm thứ gần nhất · không nhặt đồ", colMuted},
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
	interact, ic := "nhặt đồ, cành, đá cuội… (giữ: nhặt liền)", color.Color(colMuted)
	if pk := g.pick.view; g.ePicks(g.hudTarget, pk) { // see pickup.go
		interact, ic = "nhặt "+pickName(pk)+" (giữ: nhặt liền)", colGood
	} else if verb, ok := g.boatHint(); ok { // see searender.go
		interact, ic = verb, colGood
	} else if verb, can, ok := g.mountHint(); ok { // see ride.go
		interact = verb
		if can {
			ic = colGood
		}
	}
	lines = append(lines, line{"E", interact, ic}, line{"", "mở rương · cưỡi thú · lên thuyền", colMuted},
		line{"1-0", "chọn ô (lăn chuột, hoặc nhấp vào ô)", colMuted})
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
		y = lineTop(sh) - 6 - h // too wide to sit beside the hotbar: above it and its line
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
	if g.credits != nil {
		g.credits.draw(screen) // see creditsui.go
		return
	}
	cx := float64(sw) / 2
	top := max(16, float64(sh)/2-215)
	title := "Tạm dừng"
	if g.shared() {
		title = "Menu" // a room's world does not stop for one player: see room.go
	}
	drawTitle(screen, title, cx, top, 3, colBoneText, color.RGBA{60, 36, 20, 255})
	g.pause.draw(screen, cx, top+64, 340)
	below := g.pause.rowY(len(g.pause.items)) + 8
	if !g.room.joined { // a guest's line is the room's: see room.go
		g.text(screen, "Game tự lưu mỗi phút và khi thoát", cx, below, colMuted, text.AlignCenter, 1)
	}
	g.drawRoomPause(screen, sw, cx, below, top)
}

// seasonColors colour the season's name on the clock.
var seasonColors = [sim.NumSeasons]color.RGBA{
	{150, 230, 140, 255}, {255, 214, 100, 255}, {240, 150, 70, 255}, {190, 220, 255, 255},
}
