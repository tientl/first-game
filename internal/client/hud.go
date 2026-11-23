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
	colPanel   = color.RGBA{0, 0, 0, 150}
	colHealth  = color.RGBA{206, 62, 62, 255}
	colHunger  = color.RGBA{226, 150, 50, 255}
	colGood    = color.RGBA{150, 230, 140, 255}
	colMuted   = color.RGBA{190, 190, 190, 255}
	colColdTxt = color.RGBA{140, 200, 255, 255}
	colDay     = color.RGBA{240, 210, 90, 255}
	colDusk    = color.RGBA{226, 130, 70, 255}
	colNight   = color.RGBA{50, 60, 120, 255}
)

// text draws s with a one-pixel drop shadow. For AlignCenter and AlignEnd, x
// is the centre or the right edge.
func (g *Game) text(dst *ebiten.Image, s string, x, y float64, clr color.Color, align text.Align, scale float64) {
	for _, pass := range []struct {
		dx, dy float64
		c      color.Color
	}{{scale, scale, color.RGBA{0, 0, 0, 200}}, {0, 0, clr}} {
		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x+pass.dx, y+pass.dy)
		op.ColorScale.ScaleWithColor(pass.c)
		op.PrimaryAlign = align
		text.Draw(dst, s, g.face, op)
	}
}

func (g *Game) dim(dst *ebiten.Image, sw, sh int, alpha float64) {
	vector.FillRect(dst, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, uint8(alpha * 255)}, false)
}

func ingredientsText(r sim.Recipe) string {
	parts := make([]string, 0, len(r.Inputs))
	for _, in := range r.Inputs {
		parts = append(parts, fmt.Sprintf("%d %s", in.Count, in.Item.Def().Name))
	}
	return strings.Join(parts, " · ")
}

func (g *Game) drawBar(dst *ebiten.Image, x, y float32, frac float64, fill color.Color, label string) {
	const w, h = 200, 18
	frac = max(0, min(1, frac))
	vector.FillRect(dst, x-2, y-2, w+4, h+4, colPanel, false)
	vector.FillRect(dst, x, y, float32(frac*w), h, fill, false)
	g.text(dst, label, float64(x)+6, float64(y)+1, colWhite, text.AlignStart, 1)
}

func (g *Game) drawHUD(screen *ebiten.Image, sw, sh int) {
	self := g.curr.Self

	g.drawBar(screen, 16, 16, float64(self.Health)/sim.MaxHealth, colHealth,
		fmt.Sprintf("Máu %d", int(math.Ceil(float64(self.Health)))))
	g.drawBar(screen, 16, 42, float64(self.Hunger)/sim.MaxHunger, colHunger,
		fmt.Sprintf("No bụng %d", int(math.Ceil(float64(self.Hunger)))))
	if self.Cold && (g.frame/20)%2 == 0 {
		g.text(screen, "Lạnh cóng! Hãy đến gần lửa trại", 16, 68, colColdTxt, text.AlignStart, 1)
	} else if self.Hunger <= 0 && !self.Dead {
		g.text(screen, "Đang chết đói!", 16, 68, colHunger, text.AlignStart, 1)
	}

	g.drawClock(screen, sw)
	g.drawInventory(screen, sw, sh)
	g.drawHelp(screen, sh)

	if g.notice != "" && time.Now().Before(g.noticeUntil) {
		w, _ := text.Measure(g.notice, g.face, 0)
		vector.FillRect(screen, float32(sw)/2-float32(w)/2-10, 84, float32(w)+20, 24, colPanel, false)
		g.text(screen, g.notice, float64(sw/2), 88, colWhite, text.AlignCenter, 1)
	}

	if self.Dead {
		g.dim(screen, sw, sh, 0.55)
		g.text(screen, "Bạn đã gục ngã", float64(sw/2), float64(sh/2-40), colWhite, text.AlignCenter, 3)
		g.text(screen, "Nhấn R để hồi sinh (mất hết đồ)", float64(sw/2), float64(sh/2+20), colMuted, text.AlignCenter, 1)
	}
}

func (g *Game) drawClock(screen *ebiten.Image, sw int) {
	const w, h = 180, 10
	tick := g.curr.Tick
	x, y := float32(sw-w-16), float32(40)
	phase := map[sim.Phase]string{sim.PhaseDay: "Ban ngày", sim.PhaseDusk: "Chạng vạng", sim.PhaseNight: "Ban đêm"}[sim.PhaseOf(tick)]
	g.text(screen, fmt.Sprintf("Ngày %d · %s", sim.DayOf(tick), phase), float64(sw-16), 16, colWhite, text.AlignEnd, 1)

	vector.FillRect(screen, x-2, y-2, w+4, h+4, colPanel, false)
	vector.FillRect(screen, x, y, w*sim.DuskStart, h, colDay, false)
	vector.FillRect(screen, x+w*sim.DuskStart, y, w*(sim.NightStart-sim.DuskStart), h, colDusk, false)
	vector.FillRect(screen, x+w*sim.NightStart, y, w*(1-sim.NightStart), h, colNight, false)
	mx := x + float32(sim.DayFraction(tick))*w
	vector.FillRect(screen, mx-1, y-4, 3, h+8, colWhite, false)
}

func (g *Game) drawInventory(screen *ebiten.Image, sw, sh int) {
	const slot, gap = 52, 8
	n := int(sim.NumItems) - 1
	x0 := float64(sw-(n*slot+(n-1)*gap)) / 2
	y := float64(sh - slot - 16)
	for i := 0; i < n; i++ {
		kind := sim.ItemKind(i + 1)
		count := g.curr.Self.Inv[kind]
		x := x0 + float64(i*(slot+gap))
		vector.FillRect(screen, float32(x), float32(y), slot, slot, colPanel, false)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(1.5, 1.5)
		op.GeoM.Translate(x+(slot-33)/2, y+4)
		if count == 0 {
			op.ColorScale.ScaleAlpha(0.3)
		}
		screen.DrawImage(g.art.icons[kind], op)

		clr := color.Color(colWhite)
		if count == 0 {
			clr = colMuted
		}
		g.text(screen, fmt.Sprintf("%d", count), x+slot-5, y+slot-18, clr, text.AlignEnd, 1)
		if kind.Edible() {
			g.text(screen, "E", x+4, y+2, colGood, text.AlignStart, 1)
		}
	}
}

func (g *Game) drawHelp(screen *ebiten.Image, sh int) {
	lines := []struct {
		s string
		c color.Color
	}{
		{"WASD / mũi tên: di chuyển", colMuted},
		{"Giữ Space: chặt, đập, hái, thêm củi", colMuted},
		{"E: ăn quả mọng · F11: toàn màn hình", colMuted},
	}
	if r, ok := sim.RecipeByID(sim.RecipeCampfire); ok {
		c := color.Color(colMuted)
		inv := g.curr.Self.Inv
		if inv.HasAll(r.Inputs) {
			c = colGood
		}
		lines = append(lines, struct {
			s string
			c color.Color
		}{fmt.Sprintf("C: dựng %s (%s)", r.Name, ingredientsText(r)), c})
	}
	y := float64(sh - 16 - 18*len(lines))
	for _, l := range lines {
		g.text(screen, l.s, 16, y, l.c, text.AlignStart, 1)
		y += 18
	}
}
