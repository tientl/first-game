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

var (
	colPanel     = color.RGBA{14, 16, 26, 205}
	colPanelEdge = color.RGBA{0, 0, 0, 90}
	colSlot      = color.RGBA{0, 0, 0, 110}
	colHealth    = color.RGBA{214, 62, 70, 255}
	colHunger    = color.RGBA{232, 152, 52, 255}
	colGood      = color.RGBA{150, 230, 140, 255}
	colMuted     = color.RGBA{176, 182, 196, 255}
	colKey       = color.RGBA{246, 213, 92, 255}
	colColdTxt   = color.RGBA{150, 210, 255, 255}
	colDay       = color.RGBA{250, 214, 96, 255}
	colDusk      = color.RGBA{236, 128, 74, 255}
	colNight     = color.RGBA{52, 62, 132, 255}
)

func (g *Game) text(dst *ebiten.Image, s string, x, y float64, clr color.Color, align text.Align, scale float64) {
	drawText(dst, s, x, y, clr, align, scale)
}

func (g *Game) textWidth(s string) float64 { return textWidth(s, 1) }

func (g *Game) dim(dst *ebiten.Image, sw, sh int, alpha float64) {
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

func drawIcon(dst *ebiten.Image, img *ebiten.Image, x, y, scale float64, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(alpha)
	dst.DrawImage(img, op)
}

func (g *Game) drawBar(dst *ebiten.Image, icon *ebiten.Image, x, y float32, frac float64, fill color.RGBA, value string) {
	const w, h = 168, 16
	frac = max(0, min(1, frac))
	drawIcon(dst, icon, float64(x), float64(y)-5, 2, 1)
	bx := x + 34
	fillRoundRect(dst, bx, y, w, h, 5, color.RGBA{0, 0, 0, 150})
	if frac < 0.25 && (g.frame/15)%2 == 0 {
		fill = lighten(fill, 50) // flash when low
	}
	if fw := float32(frac * (w - 4)); fw >= 2 {
		fillRoundRect(dst, bx+2, y+2, fw, h-4, 3, fill)
		fillRoundRect(dst, bx+3, y+3, fw-2, 3, 1.5, withAlpha(lighten(fill, 60), 110))
	}
	g.text(dst, value, float64(bx)+w/2, float64(y)+1, colWhite, text.AlignCenter, 1)
}

func (g *Game) drawHUD(screen *ebiten.Image, sw, sh int) {
	self := g.curr.Self

	panel(screen, 12, 12, 230, 68)
	g.drawBar(screen, g.art.heart, 22, 24, float64(self.Health)/sim.MaxHealth, colHealth,
		fmt.Sprintf("Máu %d", int(math.Ceil(float64(self.Health)))))
	g.drawBar(screen, g.art.food, 22, 52, float64(self.Hunger)/sim.MaxHunger, colHunger,
		fmt.Sprintf("No bụng %d", int(math.Ceil(float64(self.Hunger)))))

	warn, warnCol := "", colColdTxt
	if self.Cold {
		warn = "Lạnh cóng! Hãy đến gần lửa trại"
		if sim.Ambient(g.curr.Tick) >= sim.ColdAmbient { // by day: it is the snow
			warn = "Tuyết lạnh cóng! Mặc Áo lông, cưỡi voi ma mút hoặc đứng gần lửa"
		}
		if g.curr.Season == sim.Winter && g.curr.Weather != sim.WeatherClear {
			warn = "Rét mùa đông! Mặc Áo lông hoặc áo da, hoặc đứng gần lửa, đuốc"
		}
	} else if self.Poisoned && !self.Dead {
		warn, warnCol = "Trúng độc nhện! Dùng Băng tơ nhện hoặc Bình máu lớn", color.RGBA{140, 230, 100, 255}
	} else if self.Webbed && !self.Dead {
		warn, warnCol = "Dính tơ nhện — đi chậm lại", colMuted
	} else if self.Hunger <= 0 && !self.Dead {
		warn, warnCol = "Đang chết đói!", colHunger
	}
	if warn != "" {
		w := float32(g.textWidth(warn)) + 24
		panel(screen, 12, 88, w, 26)
		c := warnCol
		if (g.frame/20)%2 == 0 {
			c = lighten(c, 60)
		}
		g.text(screen, warn, 24, 93, c, text.AlignStart, 1)
	}

	g.drawSeaHUD(screen) // see searender.go
	g.drawClock(screen, sw)
	if g.overlay == overlayNone { // the map and bag cover the bottom of the screen
		g.drawMinimap(screen, sw)
		g.drawHotbar(screen, sw, sh)
		g.drawHelp(screen, sh)
	}

	if since := time.Since(g.autosaved); since < 2500*time.Millisecond {
		a := float32(1)
		if since > 1500*time.Millisecond {
			a = float32(2500*time.Millisecond-since) / float32(time.Second)
		}
		msg := "Đã tự động lưu"
		w := float32(g.textWidth(msg)) + 24
		x := float32(sw) - w - 12
		y := float32(98)
		if g.overlay == overlayNone {
			y = float32(miniBottom()) + 8 // under the minimap
		}
		fillRoundRect(screen, x, y, w, 24, 8, color.RGBA{14, 16, 26, uint8(205 * a)})
		g.text(screen, msg, float64(x)+12, float64(y)+4, color.RGBA{150, 230, 140, uint8(255 * a)}, text.AlignStart, 1)
	}

	if self.Dead {
		g.dim(screen, sw, sh, 0.6)
		g.text(screen, "Bạn đã gục ngã", float64(sw/2), float64(sh/2-40), color.RGBA{236, 110, 110, 255}, text.AlignCenter, 3)
		where := "ở chỗ xuất phát"
		if self.HomeBed {
			where = "cạnh giường ngủ lần cuối"
		}
		g.text(screen, "Nhấn R để hồi sinh "+where+" (mất hết đồ)", float64(sw/2), float64(sh/2+20), colMuted, text.AlignCenter, 1)
	}
}

func (g *Game) drawNotice(screen *ebiten.Image, sw int) {
	if g.notice == "" || !time.Now().Before(g.noticeUntil) {
		return
	}
	w := float32(g.textWidth(g.notice)) + 28
	panel(screen, float32(sw)/2-w/2, 96, w, 28)
	g.text(screen, g.notice, float64(sw/2), 102, colWhite, text.AlignCenter, 1)
}

func (g *Game) drawClock(screen *ebiten.Image, sw int) {
	const w, h = 176, 8
	tick := g.curr.Tick
	px := float32(sw - w - 24 - 12)
	panel(screen, px, 12, w+24, 78)
	x, y := px+12, float32(46)
	phase := map[sim.Phase]string{sim.PhaseDay: "Ban ngày", sim.PhaseDusk: "Chạng vạng", sim.PhaseNight: "Ban đêm"}[sim.PhaseOf(tick)]
	g.text(screen, fmt.Sprintf("Ngày %d", sim.DayOf(tick)), float64(x), 20, colWhite, text.AlignStart, 1)
	g.text(screen, phase, float64(x)+w, 20, colMuted, text.AlignEnd, 1)
	season := fmt.Sprintf("%s %d/%d", g.curr.Season.Name(), sim.SeasonDay(tick), sim.SeasonDays)
	g.text(screen, season, float64(x), 62, seasonColors[g.curr.Season%sim.NumSeasons], text.AlignStart, 1)
	g.text(screen, g.curr.Weather.Name(), float64(x)+w, 62, colMuted, text.AlignEnd, 1)

	fillRoundRect(screen, x, y, w, h, 4, colNight)
	fillRoundRect(screen, x, y, w*sim.NightStart, h, 4, colDusk)
	fillRoundRect(screen, x, y, w*sim.DuskStart, h, 4, colDay)

	// A sun by day, a moon by night, riding along the bar.
	mx := x + float32(sim.DayFraction(tick))*w
	my := y + h/2
	vector.FillCircle(screen, mx, my, 8, color.RGBA{14, 16, 26, 255}, true)
	if sim.PhaseOf(tick) == sim.PhaseNight {
		vector.FillCircle(screen, mx, my, 6, color.RGBA{226, 232, 248, 255}, true)
		vector.FillCircle(screen, mx+3, my-2, 5, color.RGBA{14, 16, 26, 255}, true)
	} else {
		vector.FillCircle(screen, mx, my, 6, color.RGBA{255, 226, 110, 255}, true)
		vector.FillCircle(screen, mx-1.5, my-1.5, 2.5, color.RGBA{255, 246, 196, 255}, true)
	}
}

// drawHotbar draws the ten hotbar slots along the bottom, the held one
// outlined, and the held item's name for a moment after it changes.
func (g *Game) drawHotbar(screen *ebiten.Image, sw, sh int) {
	const n = sim.HotbarSlots
	total := n*slotPx + (n-1)*slotGap
	x0 := float64(sw-total) / 2
	y := float64(sh - slotPx - 20)
	panel(screen, float32(x0)-8, float32(y)-8, float32(total)+16, slotPx+16)
	inv := &g.curr.Self.Inv
	for i := range n {
		x := x0 + float64(i*(slotPx+slotGap))
		st := slotPlain
		if i == g.hotSel {
			st = slotHeld
		}
		g.drawSlot(screen, inv[i], x, y, st)
		g.text(screen, hotbarLabel(i), x+5, y+2, colMuted, text.AlignStart, 1)
		// How much of the burning torch is left.
		if i == g.hotSel && inv[i].Item == sim.ItemTorch && g.curr.Self.TorchFuel > 0 {
			f := float32(g.curr.Self.TorchFuel) / float32(sim.TorchBurnTicks)
			fillRoundRect(screen, float32(x)+6, float32(y)+slotPx-7, slotPx-12, 4, 2, color.RGBA{0, 0, 0, 160})
			fillRoundRect(screen, float32(x)+6, float32(y)+slotPx-7, (slotPx-12)*f, 4, 2, color.RGBA{255, 160, 60, 255})
		}
	}

	if s := inv[g.hotSel]; !s.Empty() {
		if since := time.Since(g.selChangedAt); since < 2*time.Second {
			a := min(1, float64(2*time.Second-since)/float64(500*time.Millisecond))
			name := s.Item.Def().Name
			w := float32(g.textWidth(name)) + 20
			fillRoundRect(screen, float32(sw)/2-w/2, float32(y)-38, w, 22, 7, color.RGBA{14, 16, 26, uint8(205 * a)})
			g.text(screen, name, float64(sw)/2, y-35, color.RGBA{255, 255, 255, uint8(255 * a)}, text.AlignCenter, 1)
		}
	}
}

// hotbarLabel is the key that picks hotbar slot i.
func hotbarLabel(i int) string { return fmt.Sprintf("%d", (i+1)%10) }

func (g *Game) drawHelp(screen *ebiten.Image, sh int) {
	type line struct {
		key, s string
		c      color.Color
	}
	lines := []line{
		{"WASD", "di chuyển", colMuted},
		{"Space", "nhặt, hái, đập đá (cần cúp), chặt cây (cần rìu)", colMuted},
		{"Chuột", "nhấp trái như Space · đồ rơi dưới đất: Space để nhặt", colMuted},
		{"", "cầm cuốc: xới đất, trồng và đào khoai", colMuted},
		{"", "bắt thỏ, đánh rắn, bắt giun, câu cá", colMuted},
		{"1-0", "chọn ô (hoặc lăn chuột)", colMuted},
		{"E", "ăn món đang cầm (quả mọng, thịt, cá)", colMuted},
	}
	for _, rk := range []struct {
		id   sim.RecipeID
		key  string
		verb string
	}{{sim.RecipeAxe, "Q", "làm"}, {sim.RecipeCampfire, "C", "dựng"}} {
		r, ok := sim.RecipeByID(rk.id)
		if !ok {
			continue
		}
		c := color.Color(colMuted)
		switch {
		case g.haveTool(r):
			continue // already carried: no need to show it
		case g.curr.Self.Inv.HasAll(r.Inputs):
			c = colGood
		}
		lines = append(lines, line{rk.key, fmt.Sprintf("%s %s (%s)", rk.verb, r.Name, ingredientsText(r)), c})
	}
	if g.hasRawFood() {
		c := color.Color(colMuted)
		if g.byFire() {
			c = colGood
		}
		lines = append(lines, line{"F", "nướng thịt / cá (đứng gần lửa trại)", c})
	}
	if verb, ok := g.boatHint(); ok { // see searender.go
		lines = append(lines, line{"R", verb, colGood})
	} else if verb, can, ok := g.mountHint(); ok { // see ride.go
		c := color.Color(colMuted)
		if can {
			c = colGood
		}
		lines = append(lines, line{"R", verb, c})
	}
	if held := g.myHeld(); held == sim.ItemRaft || held == sim.ItemSailboat {
		lines = append(lines, line{"V", "thả thuyền xuống nước trước mặt", colGood})
	} else if held == sim.ItemTorch || sim.IsSeed(held) {
		lines = append(lines, line{"V", "cắm đuốc / gieo hạt trước mặt", colGood})
	} else if _, ok := sim.HeadOf(g.myHeld()); ok {
		lines = append(lines, line{"V", "dựng đầu thú làm vật trang trí", colGood})
	}
	lines = append(lines,
		line{"B", "túi đồ", colMuted},
		line{"M", "bản đồ", colMuted},
		line{"= / −", "thu phóng (hoặc Ctrl + lăn chuột)", colMuted},
		line{"F11", "toàn màn hình", colMuted},
		line{"Esc", "tạm dừng · lưu game", colMuted})

	const keyW, rowH = 46, 18
	w := 0.0
	for _, l := range lines {
		w = max(w, g.textWidth(l.s))
	}
	h := float64(rowH*len(lines)) + 14
	x, y := 12.0, float64(sh)-h-12
	panel(screen, float32(x), float32(y), float32(w)+keyW+26, float32(h))
	for i, l := range lines {
		ly := y + 8 + float64(i*rowH)
		g.text(screen, l.key, x+12, ly, colKey, text.AlignStart, 1)
		g.text(screen, l.s, x+12+keyW, ly, l.c, text.AlignStart, 1)
	}
}

func (g *Game) drawPause(screen *ebiten.Image, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	cx := float64(sw) / 2
	top := float64(sh)/2 - 215
	g.text(screen, "Tạm dừng", cx, top, colWhite, text.AlignCenter, 3)
	g.pause.draw(screen, cx, top+64, 340)
	g.text(screen, "Game tự lưu mỗi phút và khi thoát", cx, g.pause.rowY(len(g.pause.items))+8, colMuted, text.AlignCenter, 1)
}

// seasonColors colour the season's name on the clock.
var seasonColors = [sim.NumSeasons]color.RGBA{
	{150, 230, 140, 255}, {255, 214, 100, 255}, {240, 150, 70, 255}, {190, 220, 255, 255},
}
