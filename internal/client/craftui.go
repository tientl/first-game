package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/sim"
)

// The crafting window (C) lists every recipe, by kind under tabs. What can
// be made now is lit; for the rest, the ingredients missing are red. A
// click makes the thing; a structure (a campfire, a wall…) is placed with
// the mouse afterwards: see build mode in mouse.go. The bag lists the
// recipes of the item clicked there, with the same cards.

type craftTab uint8

const (
	tabAll craftTab = iota
	tabTools
	tabBuild
	tabCook
	tabOther
	numCraftTabs
)

var craftTabNames = [numCraftTabs]string{"Tất cả", "Dụng cụ & vũ khí", "Xây dựng", "Nấu & nung", "Đồ dùng"}

const craftCols = 4 // recipe cards to a row

// recipeTab is the tab recipe r is listed under, besides "all".
func recipeTab(r sim.Recipe) craftTab {
	switch {
	case !r.Gives.Valid() || isBoat(r.Gives):
		return tabBuild
	case r.NearFire || r.NearFurnace:
		return tabCook
	}
	_, armor := sim.ArmorTier(r.Gives)
	if r.Gives.Def().Tool || armor || r.Gives == sim.ItemDinoArmor || sim.WeaponDamage(r.Gives) > sim.WeaponDamage(sim.ItemNone) {
		return tabTools
	}
	return tabOther
}

// craftView is the crafting window's state.
type craftView struct {
	tab      craftTab
	scroll   int     // rows of cards scrolled past
	wheel    float64 // wheel movement not yet a whole row
	hover    int     // the card under the pointer, -1 if none
	hoverTab int     // the tab under the pointer, -1 if none

	// Layout from the last draw, for the mouse.
	drawn               bool
	cardX, cardY, cardW float64
	rowsShown           int
	tabX                [numCraftTabs]float64
	tabW                [numCraftTabs]float64
	tabY, tabH          float64
}

// shown are the recipes the open tab lists, in the table's order.
func (c *craftView) shown() []sim.Recipe {
	var out []sim.Recipe
	for _, r := range recipes() {
		if c.tab == tabAll || recipeTab(r) == c.tab {
			out = append(out, r)
		}
	}
	return out
}

func (c *craftView) cardXY(i int) (x, y float64) {
	return c.cardX + float64(i%craftCols)*(c.cardW+12), c.cardY + float64(i/craftCols-c.scroll)*(cardH+cardGap)
}

func (c *craftView) cardShown(i int) bool {
	row := i/craftCols - c.scroll
	return row >= 0 && row < c.rowsShown
}

// scrollBy moves the cards by rows, keeping them in range.
func (c *craftView) scrollBy(rows int) {
	last := (len(c.shown())+craftCols-1)/craftCols - c.rowsShown
	c.scroll = max(0, min(c.scroll+rows, last))
}

// canCraft reports whether r can be made right here, right now.
func (g *Game) canCraft(r sim.Recipe) bool {
	return g.curr.Self.Inv.HasAll(r.Inputs) && !g.haveTool(r) &&
		(!r.NearFire || g.byFire()) && (!r.NearFurnace || g.byFurnace())
}

// updateCraft handles the mouse in the crafting window.
func (g *Game) updateCraft() {
	c := &g.craft
	_, wy := ebiten.Wheel()
	c.wheel += wy
	for ; c.wheel >= 1; c.wheel-- {
		c.scrollBy(-1)
	}
	for ; c.wheel <= -1; c.wheel++ {
		c.scrollBy(1)
	}
	if justPressed(ebiten.KeyArrowRight, ebiten.KeyD) {
		c.tab, c.scroll = (c.tab+1)%numCraftTabs, 0
	}
	if justPressed(ebiten.KeyArrowLeft, ebiten.KeyA) {
		c.tab, c.scroll = (c.tab+numCraftTabs-1)%numCraftTabs, 0
	}
	if !c.drawn {
		return
	}
	mx, my := ebiten.CursorPosition()
	g.craftMouse(float64(mx), float64(my), inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft))
}

// craftMouse is the pointer at (x, y) in the crafting window; press is a
// fresh left click.
func (g *Game) craftMouse(x, y float64, press bool) {
	c := &g.craft
	c.hover, c.hoverTab = -1, -1
	for t := range numCraftTabs {
		if x >= c.tabX[t] && x < c.tabX[t]+c.tabW[t] && y >= c.tabY && y < c.tabY+c.tabH {
			c.hoverTab = int(t)
			if press && c.tab != t {
				c.tab, c.scroll = t, 0
				snd.ui("ui_move")
			}
			return
		}
	}
	for i, r := range c.shown() {
		if !c.cardShown(i) {
			continue
		}
		cx, cy := c.cardXY(i)
		if x >= cx && x <= cx+c.cardW && y >= cy && y <= cy+cardH {
			c.hover = i
			if press {
				snd.ui("ui_select")
				g.craftRecipe(r) // see mouse.go
			}
			return
		}
	}
}

func (g *Game) drawCraft(screen *ebiten.Image, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	c := &g.craft
	rs := c.shown()

	pw := min(float64(sw)-32, craftCols*cardMinW+(craftCols-1)*12+40)
	allRows := (len(recipes()) + craftCols - 1) / craftCols
	fixedH := 56.0 + 40 + 16 + 44
	c.rowsShown = max(1, min(allRows, int((float64(sh)-24-fixedH+cardGap)/(cardH+cardGap))))
	c.scrollBy(0)
	ph := fixedH + float64(c.rowsShown)*(cardH+cardGap) - cardGap
	px, py := float64(int((float64(sw)-pw)/2)), float64(int(max(8, (float64(sh)-ph)/2)))
	panel(screen, float32(px), float32(py), float32(pw), float32(ph))

	n := 0
	for _, r := range recipes() {
		if g.canCraft(r) {
			n++
		}
	}
	g.text(screen, "Chế tạo", px+20, py+14, colWhite, text.AlignStart, 2)
	g.text(screen, fmt.Sprintf("%d món làm được ngay", n), px+pw-20, py+22, colGood, text.AlignEnd, 1)

	// The tabs.
	c.tabY, c.tabH = py+56, 28
	tx := px + 20
	for t := range numCraftTabs {
		label := craftTabNames[t]
		w := g.textWidth(label) + 24
		c.tabX[t], c.tabW[t] = tx, w
		bg := color.RGBA{0, 0, 0, 90}
		col := colMuted
		switch {
		case t == c.tab:
			bg, col = color.RGBA{58, 46, 22, 230}, colKey
		case int(t) == c.hoverTab:
			bg, col = color.RGBA{40, 40, 48, 200}, colWhite
		}
		fillRoundRect(screen, float32(tx), float32(c.tabY), float32(w), float32(c.tabH), 7, bg)
		g.text(screen, label, tx+w/2, c.tabY+6, col, text.AlignCenter, 1)
		tx += w + 8
	}

	// The cards.
	c.cardX, c.cardY = px+20, c.tabY+c.tabH+16
	c.cardW = (pw - 40 - 12*(craftCols-1)) / craftCols
	for i, r := range rs {
		if !c.cardShown(i) {
			continue
		}
		x, y := c.cardXY(i)
		lit := g.canCraft(r)
		if lit {
			strokeRoundRect(screen, float32(x), float32(y), float32(c.cardW), cardH, 8, color.RGBA{100, 156, 95, 170})
		}
		g.drawRecipeCard(screen, r, x, y, c.cardW, i == c.hover)
		if !lit {
			fillRoundRect(screen, float32(x), float32(y), float32(c.cardW), cardH, 8, color.RGBA{0, 0, 0, 95})
		}
	}

	hint := "Nhấp để làm · món xây dựng: nhấp rồi chọn chỗ đặt bằng chuột · C / Esc: đóng"
	if rows := (len(rs) + craftCols - 1) / craftCols; rows > c.rowsShown {
		hint = fmt.Sprintf("Lăn chuột xem thêm (hàng %d-%d / %d) · ", c.scroll+1, c.scroll+c.rowsShown, rows) + hint
	}
	g.text(screen, hint, px+pw/2, py+ph-30, colMuted, text.AlignCenter, 1)
	c.drawn = true
}

// strokeRoundRect outlines a rounded box.
func strokeRoundRect(dst *ebiten.Image, x, y, w, h, r float32, clr color.Color) {
	fillRoundRect(dst, x-1.5, y-1.5, w+3, h+3, r+1.5, clr)
	fillRoundRect(dst, x, y, w, h, r, color.RGBA{14, 16, 26, 255})
}
