package client

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Free build (dev mode; the rules are in sim/devbuild.go): the crafting
// window lights every recipe with a DEV badge in place of what it needs, and
// B opens the build palette: every piece that can be set down, by section,
// each with its icon (the item's, HD where there is one) or, for a
// structure, its sprite. A click on a piece takes it in hand: a ghost under
// the pointer, a left click puts it there, and it stays in hand to put down
// again (holding the button and moving lays one on every tile passed); a
// right click or Esc lets it go. Shift and the right button take away the
// structure under the pointer, at once and for nothing.

// Palette layout, in UI pixels.
const (
	palTileW   = 84.0
	palTileH   = 60.0
	palGap     = 5.0
	palHeadH   = 18.0 // a section's heading
	palMaxCols = 10
	palTop     = 40.0 // the panel's top at the least: below the DEV tag
)

// freeBuild reports whether the server says this player builds freely.
func (g *Game) freeBuild() bool { return g.curr.Self.DevBuild }

// placeReach is how far from the player a tile may be built on.
func (g *Game) placeReach() float64 {
	if g.freeBuild() {
		return sim.DevReach
	}
	return sim.PlaceReach
}

// paletteView is the build palette's state.
type paletteView struct {
	hover  int     // the piece under the pointer, -1 if none
	scroll int     // rows scrolled past
	wheel  float64 // wheel movement not yet a whole row

	// Layout from the last draw, for the mouse.
	drawn bool
	cells []paletteCell
	rows  int // rows in all, headings included
	shown int // rows the panel had room for
}

// paletteCell is where piece i was drawn.
type paletteCell struct {
	i          int
	x, y, w, h float64
}

// paletteRow is a row of the palette: a section's heading, or pieces.
type paletteRow struct {
	head   bool
	group  sim.DevGroup
	pieces []int
}

// paletteRows lays the pieces out in rows of cols, section by section.
func paletteRows(cols int) []paletteRow {
	var rows []paletteRow
	pieces := sim.DevPieces()
	for gr := sim.DevGroup(0); gr < sim.NumDevGroups; gr++ {
		var in []int
		for i, pc := range pieces {
			if pc.Group == gr {
				in = append(in, i)
			}
		}
		if len(in) == 0 {
			continue
		}
		rows = append(rows, paletteRow{head: true, group: gr})
		for len(in) > 0 {
			n := min(cols, len(in))
			rows = append(rows, paletteRow{group: gr, pieces: in[:n]})
			in = in[n:]
		}
	}
	return rows
}

func rowH(r paletteRow) float64 {
	if r.head {
		return palHeadH
	}
	return palTileH + palGap
}

// openPalette opens the build palette (B in free build).
func (g *Game) openPalette() {
	g.palette.hover = -1
	g.setOverlay(overlayPalette)
}

// updatePalette handles the mouse and the wheel in the palette.
func (g *Game) updatePalette() {
	p := &g.palette
	_, wy := ebiten.Wheel()
	p.wheel += wy
	for ; p.wheel >= 1; p.wheel-- {
		p.scroll = max(0, p.scroll-1)
	}
	for ; p.wheel <= -1; p.wheel++ {
		p.scroll = max(0, min(p.scroll+1, p.rows-p.shown))
	}
	if !p.drawn || g.ptr.fixed { // a pointer held for a screenshot is not on the palette
		return
	}
	mx, my := uiCursor()
	x, y := float64(mx), float64(my)
	p.hover = -1
	for _, c := range p.cells {
		if x >= c.x && x < c.x+c.w && y >= c.y && y < c.y+c.h {
			p.hover = c.i
		}
	}
	if p.hover >= 0 && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		snd.ui("ui_select")
		g.pickPiece(p.hover)
	}
}

// pickPiece takes palette piece i in hand, to put down with the mouse.
func (g *Game) pickPiece(i int) {
	pc, ok := sim.DevPieceAt(i)
	if !ok {
		return
	}
	g.build = buildMode{on: true, dev: true, piece: i}
	g.setOverlay(overlayNone)
	g.ptr.block = true // the click that picked it does not put it down too
	g.showNotice("Đặt " + pc.Name + ": chuột trái (giữ kéo để đặt liền) · chuột phải hoặc Esc: thôi · B: bảng xây")
}

// buildName is what build mode puts down, by name.
func (g *Game) buildName() (string, bool) {
	if !g.build.on {
		return "", false
	}
	if g.build.dev {
		pc, ok := sim.DevPieceAt(g.build.piece)
		return pc.Name, ok
	}
	r, ok := sim.RecipeByID(g.build.recipe)
	return r.Name, ok
}

// devRemoveTarget is the entity Shift and the right button would take
// away: the one on the tile under the pointer, if free build may.
func (g *Game) devRemoveTarget() (e protocol.EntityView, inReach, ok bool) {
	me, found := g.me()
	if !g.freeBuild() || !found || me.Dead || g.overlay != overlayNone {
		return protocol.EntityView{}, false, false
	}
	ax, ay, aim := g.aim()
	if !aim {
		return protocol.EntityView{}, false, false
	}
	tx, ty := int(math.Floor(ax)), int(math.Floor(ay))
	i, there := g.entAt[[2]int{tx, ty}]
	if !there || i >= len(g.curr.Entities) {
		return protocol.EntityView{}, false, false
	}
	e = g.curr.Entities[i]
	if !sim.DevRemovable(e.Kind) || g.hiddenInCave(float64(tx)+0.5, float64(ty)+0.5) {
		return protocol.EntityView{}, false, false
	}
	return e, math.Hypot(float64(tx)+0.5-float64(me.X), float64(ty)+0.5-float64(me.Y)) <= sim.DevReach, true
}

// devRemove is Shift and the right button in free build.
func (g *Game) devRemove() {
	e, in, ok := g.devRemoveTarget()
	switch {
	case !ok:
		g.showNotice("Không có gì để gỡ dưới con trỏ")
	case !in:
		g.showNotice("Xa quá — lại gần hơn rồi gỡ")
	default:
		g.command(sim.Command{Kind: sim.CmdDevRemove, At: true, TX: e.TX, TY: e.TY})
		snd.ui("ui_move")
	}
}

// shiftHeld reports whether a Shift key is down.
func shiftHeld() bool { return ebiten.IsKeyPressed(ebiten.KeyShift) }

// drawDevBadge draws the small DEV tag that marks what free build makes out
// of nothing, with its right edge at x.
func (g *Game) drawDevBadge(dst *uiDst, x, y float64) {
	w := textWidthW("DEV", 0.82, 900) + 10
	fillRoundRect(dst, float32(x-w)-0.5, float32(y)-0.5, float32(w)+1, 16, 4.5, color.RGBA{60, 22, 6, 200})
	fillRoundRect(dst, float32(x-w), float32(y), float32(w), 15, 4, colDevTag)
	noteBox(dst, x-w, y, w, 15)
	uiTextW(dst, "DEV", x-w/2, y+7.5-lineUI*0.82/2, color.RGBA{48, 20, 6, 255}, text.AlignCenter, 0.82, 900, fxPlain)
}

// pieceIcon draws piece pc's icon in a box size across with its top-left
// at (x, y): the item's icon when it is an item in the hand, else the
// structure's sprite, shrunk to fit.
func (g *Game) pieceIcon(dst *uiDst, pc sim.DevPiece, x, y, size float64) {
	if pc.Item.Valid() {
		g.drawItem(dst, pc.Item, x, y, size, 1)
		return
	}
	ev := protocol.EntityView{Kind: pc.Kind, Var: pc.Var, Ready: true, Fuel: sim.FireStartFuel}
	s := g.pieceSprite(ev)
	if s.img == nil {
		return
	}
	sc := min(1.5, size/max(s.w(), 1), size/max(s.h(), 1))
	// The sprite stands on its anchor: put its box in the middle.
	cx := x + size/2 + (s.ax-s.w()/2)*sc
	cy := y + size/2 + (s.ay-s.h()/2)*sc
	drawAt(dst, s, cx, cy, sc, sc, 0, nil)
	switch pc.Kind {
	case sim.EntityCampfire: // the flames and the front stones, as on its recipe card
		drawAt(dst, g.art.flames[(g.frame/5)%flameFrames], cx, cy+sc*0.7, sc*0.7, sc*0.7, 0, nil)
		drawAt(dst, g.art.campFront, cx, cy, sc, sc, 0, nil)
	case sim.EntityBonfire:
		drawAt(dst, g.art.flames[(g.frame/4)%flameFrames], cx, cy+sc*0.7, sc*0.9, sc, 0, nil)
	}
}

// pieceSprite is the sprite of a piece set down as e, as the world draws
// it: a lair's props have their own.
func (g *Game) pieceSprite(e protocol.EntityView) sprite {
	if e.Kind == sim.EntityLairProp {
		return g.lairSprite(e) // see lairrender.go
	}
	s, _ := g.entitySprite(e)
	return s
}

// drawPalette draws the build palette.
func (g *Game) drawPalette(screen *uiDst, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	p := &g.palette
	cols := max(4, min(palMaxCols, int((float64(sw)-16-40+palGap)/(palTileW+palGap))))
	rows := paletteRows(cols)
	p.rows = len(rows)

	pw := float64(cols)*palTileW + float64(cols-1)*palGap + 40
	const topH, footH = 46.0, 28.0
	// As many rows as fit, from the scroll on.
	room := float64(sh) - palTop - 8 - topH - footH
	p.scroll = max(0, min(p.scroll, len(rows)-1))
	h, shown := 0.0, 0
	for _, r := range rows[p.scroll:] {
		if h+rowH(r) > room && shown > 0 {
			break
		}
		h += rowH(r)
		shown++
	}
	p.shown = shown
	if p.scroll+shown >= len(rows) { // all the rest fit: no scrolling past them
		for p.scroll > 0 && h+rowH(rows[p.scroll-1]) <= room {
			p.scroll--
			h += rowH(rows[p.scroll])
			p.shown++
		}
	}
	ph := topH + h + footH
	px, py := math.Floor((float64(sw)-pw)/2), math.Floor(max(palTop, (float64(sh)-ph)/2))
	drawFrame(screen, float32(px), float32(py), float32(pw), float32(ph))

	g.text(screen, "Bảng xây", px+20, py+10, colWhite, text.AlignStart, 2)
	g.drawDevBadge(screen, px+20+g.textWidth("Bảng xây")*2+14+g.textWidth("DEV")+10, py+18)
	g.text(screen, "Xây tự do: không tốn gì · chọn một món rồi đặt bằng chuột", px+pw-20, py+18, colMuted, text.AlignEnd, 1)

	pieces := sim.DevPieces()
	p.cells = p.cells[:0]
	y := py + topH
	for _, r := range rows[p.scroll : p.scroll+p.shown] {
		if r.head {
			g.text(screen, sim.DevGroupNames[r.group], px+22, y+3, colKey, text.AlignStart, 1)
			y += palHeadH
			continue
		}
		for j, i := range r.pieces {
			x := px + 20 + float64(j)*(palTileW+palGap)
			p.cells = append(p.cells, paletteCell{i: i, x: x, y: y, w: palTileW, h: palTileH})
			g.drawPaletteTile(screen, pieces[i], i, x, y)
		}
		y += palTileH + palGap
	}

	hint := "Nhấp chọn · chuột trái đặt, giữ kéo đặt liền · Shift + chuột phải: gỡ · B / Esc: đóng"
	if p.shown < p.rows {
		hint = "Lăn chuột xem thêm · " + hint
	}
	g.text(screen, hint, px+pw/2, py+ph-footH+6, colMuted, text.AlignCenter, 1)
	p.drawn = true
}

// drawPaletteTile draws piece pc, palette index i, in its tile at (x, y).
func (g *Game) drawPaletteTile(screen *uiDst, pc sim.DevPiece, i int, x, y float64) {
	held := g.build.on && g.build.dev && g.build.piece == i
	st := tabIdle
	switch {
	case held:
		st = tabOn
	case g.palette.hover == i:
		st = tabHover
	}
	tab(screen, x, y, palTileW, palTileH, 8, st)
	const icon = 28.0
	g.pieceIcon(screen, pc, x+(palTileW-icon)/2, y+4, icon)

	// The name, on two lines when one will not do.
	lines := []string{pc.Name}
	const ts = 0.86 // the names' size
	w := func(s string) float64 { return textWidth(s, ts) }
	if w(pc.Name) > palTileW-6 {
		words := strings.Fields(pc.Name)
		best := len(words) / 2
		for k := 1; k < len(words); k++ {
			a, b := strings.Join(words[:k], " "), strings.Join(words[k:], " ")
			if max(w(a), w(b)) < max(w(strings.Join(words[:best], " ")), w(strings.Join(words[best:], " "))) {
				best = k
			}
		}
		if best > 0 && best < len(words) {
			lines = []string{strings.Join(words[:best], " "), strings.Join(words[best:], " ")}
		}
	}
	ty := y + 31
	if len(lines) == 1 {
		ty += 6
	}
	col := colWhite
	if held {
		col = colKey
	}
	for _, l := range lines {
		g.text(screen, fitText(l, palTileW-4, ts), x+palTileW/2, ty, col, text.AlignCenter, ts)
		ty += 12
	}
}
