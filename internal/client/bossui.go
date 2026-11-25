package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The bosses as the HUD shows them: a big health bar at the top while one
// fights you, its lair on the map once seen, and the "Phá đảo!" screen once
// every one has fallen.

// bossUI is the client's own state of the boss screens.
type bossUI struct {
	shownHP float64    // the bar's trailing share, catching up with the real one
	closed  bool       // the victory screen was closed here
	sent    bool       // and the server was told
	btn     buttonList // "Tiếp tục chơi"
}

var (
	colBossBar  = color.RGBA{196, 40, 44, 255}
	colBossRage = color.RGBA{255, 92, 40, 255}
	colGold     = color.RGBA{255, 212, 90, 255}
)

// drawBossBar draws the health of the boss fighting this player.
func (g *Game) drawBossBar(screen *ebiten.Image, sw int) {
	b := g.curr.Boss
	if !b.Kind.Boss() || b.MaxHP <= 0 || g.curr.Self.Dead {
		g.boss.shownHP = 1
		return
	}
	f := float64(b.HP) / float64(b.MaxHP)
	if g.boss.shownHP < f {
		g.boss.shownHP = f
	}
	g.boss.shownHP += (f - g.boss.shownHP) * 0.04
	w := float32(min(560, max(240, sw-2*262)))
	x, y := (float32(sw)-w)/2, float32(12)
	panel(screen, x-12, y-4, w+24, 62)
	title, col := bossName(b.Kind), colWhite
	if b.Enraged {
		title += " (nổi giận!)"
		if (g.frame/15)%2 == 0 {
			col = colBossRage
		}
	}
	g.text(screen, title, float64(sw)/2, float64(y), col, text.AlignCenter, 1)
	by, bh := y+28, float32(16)
	fillRoundRect(screen, x, by, w, bh, 5, color.RGBA{30, 10, 12, 230})
	fillRoundRect(screen, x, by, w*float32(g.boss.shownHP), bh, 5, color.RGBA{240, 200, 170, 200})
	fill := colBossBar
	if b.Enraged {
		fill = colBossRage
	}
	fillRoundRect(screen, x, by, w*float32(f), bh, 5, fill)
	g.text(screen, fmt.Sprintf("%d / %d", b.HP, b.MaxHP), float64(sw)/2, float64(by)+1, colWhite, text.AlignCenter, 1)
	if b.Holds {
		msg := "Bị siết! Giữ phím di chuyển để vùng ra, hoặc đánh nó"
		c := colBad
		if (g.frame/12)%2 == 0 {
			c = lighten(c, 50)
		}
		g.text(screen, msg, float64(sw)/2, float64(y)+64, c, text.AlignCenter, 1)
	}
}

// noteBosses tells the player when a boss they knew of has fallen.
func (g *Game) noteBosses(old, cur protocol.Snapshot) {
	was := map[sim.CreatureKind]bool{}
	for _, m := range old.BossMarks {
		was[m.Kind] = m.Defeated
	}
	for _, m := range cur.BossMarks {
		if down, ok := was[m.Kind]; ok && !down && m.Defeated {
			g.showNotice("Đã hạ " + m.Kind.Name() + "! Nhặt " + sim.LegendaryItem(m.Kind).Def().Name)
		}
	}
}

func (g *Game) victoryShowing() bool { return g.curr.Victory.Show && !g.boss.closed }

// updateVictory runs the victory screen; it reports whether it is up, so
// the game behind it takes no input.
func (g *Game) updateVictory() bool {
	if g.boss.closed && !g.boss.sent {
		g.boss.sent = g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientCommand, Command: sim.Command{Kind: sim.CmdVictorySeen}}) == nil
	}
	if !g.victoryShowing() {
		return false
	}
	if g.sentInput != (sim.Input{}) && g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput}) == nil {
		g.sentInput = sim.Input{}
	}
	g.setOverlay(overlayNone)
	if g.boss.btn.items == nil {
		g.boss.btn.items = []button{{label: "Tiếp tục chơi"}}
	}
	if g.boss.btn.update() == 0 {
		g.boss.closed = true
	}
	return true
}

// drawVictory draws the game-beaten screen.
func (g *Game) drawVictory(screen *ebiten.Image, sw, sh int) {
	if !g.victoryShowing() {
		return
	}
	v := g.curr.Victory
	g.dim(screen, sw, sh, 0.75)
	cx := float64(sw) / 2
	drawTitle(screen, "Phá đảo!", cx, float64(sh)*0.08, 5, colGold, color.RGBA{60, 30, 0, 255})
	g.text(screen, fmt.Sprintf("Bạn đã hạ cả %d trùm của thế giới này — ngày thứ %d", len(v.Beaten), v.Day), cx, float64(sh)*0.08+84, colWhite, text.AlignCenter, 1.5)
	cols := 3
	cw, ch := 220.0, 150.0
	x0 := cx - cw*float64(cols)/2
	y0 := float64(sh)*0.08 + 130
	for i, k := range v.Beaten {
		bi, ok := sim.BossIndex(k)
		if !ok {
			continue
		}
		x, y := x0+float64(i%cols)*cw, y0+float64(i/cols)*ch
		panel(screen, float32(x)+8, float32(y), float32(cw)-16, float32(ch)-12)
		pose := bossIdle
		if bossID(bi) == bossMegalodon {
			pose = bossSpecial // its idle is only a shadow under the water
		}
		s := g.art.bosses[bi][pose][0]
		sw, sh := s.w(), s.h()
		k2 := min(1.4, 180/sw, 90/sh)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(k2, k2)
		op.GeoM.Translate(x+cw/2-sw*k2/2, y+8+90-sh*k2)
		dst := screen
		drawSprite(dst, s, op)
		// Struck off: a red cross.
		ax, ay := float32(x+cw/2), float32(y+8+50)
		vector.StrokeLine(dst, ax-26, ay-26, ax+26, ay+26, 5, color.RGBA{220, 30, 30, 220}, true)
		vector.StrokeLine(dst, ax+26, ay-26, ax-26, ay+26, 5, color.RGBA{220, 30, 30, 220}, true)
		g.text(screen, k.Name(), x+cw/2, y+104, colGold, text.AlignCenter, 1)
		g.text(screen, sim.BossEpithet(k), x+cw/2, y+120, colMuted, text.AlignCenter, 1)
	}
	rows := (len(v.Beaten) + cols - 1) / cols
	g.boss.btn.draw(screen, cx, y0+float64(rows)*ch+10, 300)
}

// drawBossMarks draws the lairs this player has seen on the world map.
func (g *Game) drawBossMarks(dst *ebiten.Image, at func(x, y float64) (float32, float32), r0 float64) {
	if r0 != 0 {
		return // the map shows the caves
	}
	for _, m := range g.curr.BossMarks {
		x, y := at(float64(m.X), float64(m.Y))
		bg := color.RGBA{150, 24, 28, 255}
		if m.Defeated {
			bg = color.RGBA{90, 90, 96, 255}
		}
		vector.FillCircle(dst, x, y, 13, colInk, true)
		vector.FillCircle(dst, x, y, 11.5, bg, true)
		if h, ok := g.art.heads[m.Kind]; ok {
			hw, hh := h.w(), h.h()
			k := 18 / max(hw, hh)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-hw/2, -hh/2)
			op.GeoM.Scale(k, k)
			op.GeoM.Translate(float64(x), float64(y))
			drawSprite(dst, h, op)
		}
		if m.Defeated {
			vector.StrokeLine(dst, x-10, y-10, x+10, y+10, 3, color.RGBA{230, 40, 40, 255}, true)
			vector.StrokeLine(dst, x+10, y-10, x-10, y+10, 3, color.RGBA{230, 40, 40, 255}, true)
		}
		g.text(dst, m.Kind.Name(), float64(x), float64(y)+14, colWhite, text.AlignCenter, 1)
	}
}
