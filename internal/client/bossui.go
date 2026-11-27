package client

import (
	"fmt"
	"image"
	"image/color"
	"math"

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

// bossUp reports whether the boss's bar is up, and whether the boss holds
// this player (a line under the bar says how to get free).
func (g *Game) bossUp() (up, holds bool) {
	b := g.curr.Boss
	if !b.Kind.Boss() || b.MaxHP <= 0 || g.curr.Self.Dead {
		return false, false
	}
	return true, b.Holds
}

// drawBossBar draws the health of the boss fighting this player, top
// middle on a slim slab of carved stone between the vitals and the minimap
// (hudlayout.go): its name in the title type, a thin bar with a mark at
// half (where it flies into its rage), and behind the health left a paler
// stretch of what the last blows took, which holds a moment and then
// drains.
func (g *Game) drawBossBar(screen *uiDst, sw int) {
	b := g.curr.Boss
	u := &g.boss
	if up, _ := g.bossUp(); !up {
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
	W := bossBarW(sw)
	X := math.Round((float64(sw) - W) / 2)
	// A slim slab of carved stone, a small tusk of bone at each end.
	const pad = 12.0
	drawPanel(screen, psStone, X-pad, bossTop, W+2*pad, bossSlabH, 8)
	noteBox(screen, X-pad, bossTop, W+2*pad, bossSlabH)
	drawTusk(screen, X-pad+2, bossTop+bossSlabH-7, -1, 0.7)
	drawTusk(screen, X+W+pad-2, bossTop+bossSlabH-7, 1, 0.7)
	title, col := bossName(b.Kind), colGold
	if b.Enraged {
		title += " (nổi giận!)"
		if (g.frame/15)%2 == 0 {
			col = colBossRage
		}
	}
	drawTitle(screen, title, float64(sw)/2, bossTop+3, 1.08, col, color.RGBA{50, 30, 12, 255}) // see menu.go
	BY, BH := float64(bossTop+25), 10.0
	well(screen, X-1.5, BY-1.5, W+3, BH+3, BH/2+1.5)
	noteBox(screen, X, BY, W, BH)
	fill := colBossBar
	if b.Enraged {
		fill = colBossRage
	}
	flash := 0.0
	if since >= 0 && since < 10 {
		flash = 1 - float64(since)/10 // the flash of the blow
	}
	drawBarFill(screen, X+1, BY+1, W-2, BH-2, f, fill, barFX{trail: u.shownHP, flash: flash}, g.frame)
	// The mark at half: a notch of bone.
	mx, by, bh := float32(X+W/2), float32(BY), float32(BH)
	mark := colBone
	if f < 0.5 {
		mark = color.RGBA{120, 104, 82, 255} // passed
	}
	fillRect(screen, mx-0.6, by, 1.2, bh, fade(colHideDark, 160), true)
	var d vector.Path
	d.MoveTo(mx, by-4.5)
	d.LineTo(mx+2.8, by-1.6)
	d.LineTo(mx, by+1.3)
	d.LineTo(mx-2.8, by-1.6)
	d.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(mark)
	fillPath(screen, &d, op)
	const vs = 0.58
	uiTextW(screen, fmt.Sprintf("%d / %d", b.HP, b.MaxHP), X+W-7, BY+BH/2-lineUI*vs/2, colWhite, text.AlignEnd, vs, 880, fxOutline) // clear of the mark at half
	if b.Holds {
		const s = 0.85
		msg := "Bị siết! Giữ phím di chuyển để vùng ra, hoặc đánh nó"
		c := colBad
		if (g.frame/12)%2 == 0 {
			c = lighten(c, 50)
		}
		w := textWidth(msg, s) + 22
		y := bossBottom(false) + 4
		plateA(screen, float64(sw)/2-w/2, y, w, noticeH, noticeH/2, 0.9)
		uiText(screen, msg, float64(sw)/2, y+(noticeH-lineUI*s)/2, c, text.AlignCenter, s, fxShadow)
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
func (g *Game) drawVictory(screen *uiDst, sw, sh int) {
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
		strokeLine(dst, ax-26, ay-26, ax+26, ay+26, 5, color.RGBA{220, 30, 30, 220}, true)
		strokeLine(dst, ax+26, ay-26, ax-26, ay+26, 5, color.RGBA{220, 30, 30, 220}, true)
		g.text(screen, k.Name(), x+cw/2, y+104, colGold, text.AlignCenter, 1)
		g.text(screen, sim.BossEpithet(k), x+cw/2, y+120, colMuted, text.AlignCenter, 1)
	}
	rows := (len(v.Beaten) + cols - 1) / cols
	g.boss.btn.draw(screen, cx, y0+float64(rows)*ch+10, 300)
}

// drawBossMarks draws the lairs this player has seen on the world map.
func (g *Game) drawBossMarks(dst *uiDst, at func(x, y float64) (float32, float32), r0 float64) {
	if r0 != 0 {
		return // the map shows the caves
	}
	for _, m := range g.curr.BossMarks {
		x, y := at(float64(m.X), float64(m.Y))
		bg := color.RGBA{150, 24, 28, 255}
		if m.Defeated {
			bg = color.RGBA{90, 90, 96, 255}
		}
		fillCircle(dst, x, y, 13, colInk, true)
		fillCircle(dst, x, y, 11.5, bg, true)
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
			strokeLine(dst, x-10, y-10, x+10, y+10, 3, color.RGBA{230, 40, 40, 255}, true)
			strokeLine(dst, x+10, y-10, x-10, y+10, 3, color.RGBA{230, 40, 40, 255}, true)
		}
		g.text(dst, m.Kind.Name(), float64(x), float64(y)+14, colWhite, text.AlignCenter, 1)
	}
}

// drawTusk draws a curved tusk of bone at (x, y) curling outwards to side
// (-1 left, 1 right), k times its size.
func drawTusk(d *uiDst, x, y float64, side, k float64) {
	pt := func(dx, dy float64) (float32, float32) { return float32(x + side*dx*k), float32(y + dy*k) }
	tusk := func(grow float64) *vector.Path {
		var p vector.Path
		p.MoveTo(pt(-2-grow, 6+grow))
		cx, cy := pt(12+grow, 4+grow)
		ex, ey := pt(14+grow*0.6, -22-grow*1.5)
		p.QuadTo(cx, cy, ex, ey)
		cx, cy = pt(8-grow, -6)
		ex, ey = pt(-2-grow, -4-grow)
		p.QuadTo(cx, cy, ex, ey)
		p.Close()
		return &p
	}
	fill := func(p *vector.Path, c color.RGBA) {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(c)
		fillPath(d, p, op)
	}
	fill(tusk(1.4), color.RGBA{30, 22, 16, 230})
	fill(tusk(0), colBone)
	fill(tusk(-1.6), color.RGBA{248, 240, 220, 255}) // the lit ridge
	for i := range 3 {                               // the rings where it grew
		ax, ay := pt(4+float64(i)*3, 2-float64(i)*5)
		bx, by := pt(9+float64(i)*2.2, 1-float64(i)*6)
		strokeLine(d, ax, ay, bx, by, float32(0.7*k), color.RGBA{170, 150, 118, 255}, true)
	}
}
