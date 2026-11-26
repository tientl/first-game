package client

import (
	"fmt"
	"image"
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
	shownHP float64 // the bar's trailing share, catching up with the real one
	lastHP  float64 // the share at the last frame, and the frame of the last blow
	hitAt   int
	closed  bool       // the victory screen was closed here
	sent    bool       // and the server was told
	btn     buttonList // "Tiếp tục chơi"
}

var (
	colBossBar  = color.RGBA{196, 40, 44, 255}
	colBossRage = color.RGBA{255, 92, 40, 255}
	colGold     = color.RGBA{255, 212, 90, 255}
)

// drawBossBar draws the health of the boss fighting this player: its name,
// a bar with a mark at half (where it flies into its rage), and behind the
// health left a paler stretch of what the last blows took, which holds a
// moment and then drains. It is half as big again on a Full HD screen.
func (g *Game) drawBossBar(screen *ebiten.Image, sw int) {
	b := g.curr.Boss
	u := &g.boss
	if !b.Kind.Boss() || b.MaxHP <= 0 || g.curr.Self.Dead {
		u.shownHP, u.lastHP = 1, 1
		return
	}
	f := max(0, float64(b.HP)/float64(b.MaxHP))
	if u.shownHP < f {
		u.shownHP = f
	}
	if f < u.lastHP-1e-9 {
		u.hitAt = g.frame // struck: the loss shows a while before it drains
	}
	u.lastHP = f
	since := g.frame - u.hitAt
	if since < 0 || since > 26 {
		u.shownHP += (f - u.shownHP) * 0.07
		if u.shownHP-f < 0.002 {
			u.shownHP = f
		}
	}
	k := float32(max(1, min(1.5, float64(sw)/1280)))
	w := min(640*k, float32(sw)-2*270)
	x, y := (float32(sw)-w)/2, 10*k
	panel(screen, x-14*k, y-4*k, w+28*k, 62*k)
	title, col := bossName(b.Kind), colGold
	if b.Enraged {
		title += " (nổi giận!)"
		if (g.frame/15)%2 == 0 {
			col = colBossRage
		}
	}
	g.text(screen, title, float64(sw)/2, float64(y), col, text.AlignCenter, float64(k)*1.15)
	by, bh := y+27*k, 18*k
	fillRoundRect(screen, x-2*k, by-2*k, w+4*k, bh+4*k, 6*k, color.RGBA{96, 74, 34, 255}) // a bronze rim
	fillRoundRect(screen, x, by, w, bh, 4*k, color.RGBA{30, 10, 12, 255})
	fill := colBossBar
	if b.Enraged {
		fill = colBossRage
	}
	if lost := float32(u.shownHP); u.shownHP > f {
		c := color.RGBA{240, 200, 170, 230}
		if since >= 0 && since < 8 {
			c = color.RGBA{255, 250, 240, 255} // the flash of the blow
		}
		fillRoundRect(screen, x, by, w*lost, bh, 4*k, c)
	}
	if fw := w * float32(f); fw > 1 {
		fillRoundRect(screen, x, by, fw, bh, 4*k, fill)
		vector.DrawFilledRect(screen, x+3*k, by+2*k, max(0, fw-6*k), 3*k, lighten(fill, 46), false)
		vector.DrawFilledRect(screen, x+3*k, by+bh-4*k, max(0, fw-6*k), 2*k, color.RGBA{fill.R / 2, fill.G / 2, fill.B / 2, 255}, false)
	}
	// The mark at half.
	mx := x + w/2
	mark := colGold
	if f < 0.5 {
		mark = color.RGBA{150, 124, 60, 255} // passed
	}
	vector.DrawFilledRect(screen, mx-k, by-3*k, 2*k, bh+6*k, mark, false)
	var d vector.Path
	d.MoveTo(mx, by-8*k)
	d.LineTo(mx+4*k, by-3*k)
	d.LineTo(mx, by+2*k)
	d.LineTo(mx-4*k, by-3*k)
	d.Close()
	vs, is := d.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(mark.R)/255, float32(mark.G)/255, float32(mark.B)/255, 1
	}
	screen.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{AntiAlias: true})
	g.text(screen, fmt.Sprintf("%d / %d", b.HP, b.MaxHP), float64(sw)/2, float64(by+bh/2)-7*float64(k), colWhite, text.AlignCenter, float64(k))
	if b.Holds {
		msg := "Bị siết! Giữ phím di chuyển để vùng ra, hoặc đánh nó"
		c := colBad
		if (g.frame/12)%2 == 0 {
			c = lighten(c, 50)
		}
		g.text(screen, msg, float64(sw)/2, float64(y+64*k), c, text.AlignCenter, float64(k))
	}
}

var whitePx *ebiten.Image

// whitePixel is a white image to fill triangles with.
func whitePixel() *ebiten.Image {
	if whitePx == nil {
		whitePx = ebiten.NewImage(3, 3)
		whitePx.Fill(color.White)
	}
	return whitePx.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
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
		s := g.bossPortrait(bi, pose)
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
