package client

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/sim"
)

// The cooking window. It opens when the server says the player has a
// station open (a click, Space or E on a fire with a pot, grill or pan on
// it, or on a mortar, an oven or a drying rack), like a chest: the
// station's ingredients and what came out of it above the player's own bag
// and hotbar, a click moving a stack across (the right button, one piece).
// Between the two rows of the station an arrow fills as the dish cooks; the
// fire's gauge and a button to feed it sit under them, and the dishes the
// station knows are listed down the side. Closing it here (E, B, Esc) tells
// the server; walking away closes it from the server's side.

// cookView is the cooking window's state.
type cookView struct {
	closing sim.EntityID // closed here; ignored until the server agrees
	hover   int          // slot under the pointer: the station's 0..CookSlots-1, mine CookSlots.., or -1
	btn     int          // button under the pointer: cookBtnStoke, cookBtnDown, or -1

	// Layout from the last draw, for the mouse.
	drawn      bool
	x0, slotY  float64 // the station's first slot
	outX       float64 // its first slot of what came out
	invY       float64
	btnX, btnY [2]float64
	btnW, btnH [2]float64
}

const (
	cookBtnStoke = iota
	cookBtnDown
)

const (
	cookSide = 264 // the list of dishes down the right side
	cookGrid = bagCols*chestRow - chestGap
)

func (c *cookView) stationSlotXY(i int) (x, y float64) {
	if i < sim.CookIn {
		return c.x0 + float64(i*chestRow), c.slotY
	}
	return c.outX + float64((i-sim.CookIn)*chestRow), c.slotY
}

func (c *cookView) invSlotXY(i int) (x, y float64) {
	row, col := slotPos(i)
	y = c.invY + float64(row*chestRow)
	if row == bagRows {
		y += 8
	}
	return c.x0 + float64(col*chestRow), y
}

func (c *cookView) slotAt(x, y float64) int {
	in := func(sx, sy float64) bool { return x >= sx && x < sx+chestPx && y >= sy && y < sy+chestPx }
	for i := range sim.CookSlots {
		if in(c.stationSlotXY(i)) {
			return i
		}
	}
	for i := range sim.NumSlots {
		if in(c.invSlotXY(i)) {
			return sim.CookSlots + i
		}
	}
	return -1
}

// syncCook opens or closes the cooking window to match the snapshot.
func (g *Game) syncCook() {
	switch id := g.curr.Self.Cook.ID; {
	case id == 0:
		g.cook.closing = 0
		if g.overlay == overlayCook {
			g.setOverlay(overlayNone)
		}
	case id == g.cook.closing:
		// Closed here a moment ago; the server has not heard yet.
	case g.overlay != overlayCook && !g.curr.Self.Dead:
		g.setOverlay(overlayCook)
	}
}

// closeCook tells the server the cooking window was closed here.
func (g *Game) closeCook() {
	g.cook.closing = g.curr.Self.Cook.ID
	g.command(sim.Command{Kind: sim.CmdCookClose})
}

func (g *Game) updateCook() {
	mx, my := uiCursor()
	g.cookMouse(float64(mx), float64(my),
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight))
}

// cookMouse is the pointer at (x, y) on the UI canvas over the cooking
// window: left moves a whole stack, right one piece.
func (g *Game) cookMouse(x, y float64, left, right bool) {
	c := &g.cook
	c.hover, c.btn = -1, -1
	if !c.drawn {
		return
	}
	c.hover = c.slotAt(x, y)
	for b := range c.btnW {
		if c.btnW[b] > 0 && x >= c.btnX[b] && x < c.btnX[b]+c.btnW[b] && y >= c.btnY[b] && y < c.btnY[b]+c.btnH[b] {
			c.btn = b
		}
	}
	if !left && !right {
		return
	}
	cv := &g.curr.Self.Cook
	switch {
	case c.btn == cookBtnStoke && left:
		g.command(sim.Command{Kind: sim.CmdCookStoke})
		snd.ui("ui_select")
	case c.btn == cookBtnDown && left:
		g.command(sim.Command{Kind: sim.CmdCookTakeDown})
	case c.hover >= sim.CookSlots:
		i := c.hover - sim.CookSlots
		s := g.curr.Self.Inv[i]
		if s.Empty() {
			return
		}
		if !sim.StationAccepts(cv.Kind, s.Item) {
			g.showNotice(s.Item.Def().Name + " không dùng được ở " + strings.ToLower(cv.Kind.Name()))
			return
		}
		cmd := sim.Command{Kind: sim.CmdCookPut, Slot: uint8(i)}
		if right {
			cmd.Count = 1
		}
		g.command(cmd)
		snd.ui("ui_select")
	case c.hover >= 0 && !cv.Slots[c.hover].Empty():
		g.command(sim.Command{Kind: sim.CmdCookTake, Slot: uint8(c.hover)})
		snd.ui("ui_select")
	}
}

// cookStateText says what the station is doing.
func cookStateText(cv sim.StationKind, st sim.CookState, dish sim.CookID) (string, color.RGBA) {
	switch st {
	case sim.CookWorking:
		verb := map[sim.StationKind]string{
			sim.StationPot: "Đang nấu", sim.StationGrill: "Đang nướng", sim.StationPan: "Đang rán",
			sim.StationMortar: "Đang giã", sim.StationOven: "Đang nướng bánh", sim.StationRack: "Đang phơi",
		}[cv]
		if r, ok := sim.CookRecipeByID(dish); ok {
			return verb + ": " + r.Gives.Def().Name, colGood
		}
		return verb + "…", colGood
	case sim.CookCold:
		return "Lửa đã tắt — thêm củi để nấu tiếp", colBad
	case sim.CookFull:
		return "Đầy rồi — lấy bớt món đã xong ra", colHunger
	case sim.CookNeedSea:
		return "Nước biển chỉ kết muối khi giàn đặt gần biển", colHunger
	case sim.CookNoDish:
		return "Chưa đủ nguyên liệu cho món nào — xem công thức bên phải", colMuted
	}
	return "Bỏ nguyên liệu vào các ô bên trái", colMuted
}

// cookNeedsText lists a dish's ingredients.
func cookNeedsText(r sim.CookRecipe) string {
	parts := make([]string, 0, len(r.Needs))
	for _, n := range r.Needs {
		if n.Count > 1 {
			parts = append(parts, fmt.Sprintf("%d %s", n.Count, strings.ToLower(n.Name())))
		} else {
			parts = append(parts, strings.ToLower(n.Name()))
		}
	}
	return strings.Join(parts, " + ")
}

// buffText says what eating k does, for the list of dishes.
func buffText(k sim.ItemKind) string {
	var parts []string
	for _, b := range sim.DishBuffs(k) {
		parts = append(parts, fmt.Sprintf("%s %s", b.Kind.Name(), clockText(b.Ticks)))
	}
	if sim.DishCures(k) {
		parts = append(parts, "Giải độc")
	}
	return strings.Join(parts, " · ")
}

// clockText is ticks as m:ss.
func clockText(ticks int) string {
	s := (ticks + sim.TicksPerSecond - 1) / sim.TicksPerSecond
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// cookLine is one line of the list of dishes.
type cookLine struct {
	item        sim.ItemKind
	name, needs string
	fx          string
}

// cookLines lists what station k makes.
func cookLines(k sim.StationKind) []cookLine {
	var out []cookLine
	for _, r := range sim.CookRecipes(k) {
		l := cookLine{item: r.Gives, name: r.Gives.Def().Name, needs: cookNeedsText(r), fx: buffText(r.Gives)}
		if r.Salty {
			l.needs += " (+ muối: thêm 1 phần)"
		}
		for _, n := range r.Needs {
			if !n.Item.Valid() && len(sim.ClassItems(n.Class)) == 0 {
				l.fx = "chưa có " + strings.ToLower(n.Class.Name()) + " trong thế giới này"
			}
		}
		out = append(out, l)
	}
	switch k {
	case sim.StationGrill:
		out = append(out, cookLine{item: sim.ItemCookedMeat, name: "Nướng 4 món cùng lúc", needs: "thịt, cá, khoai, sò, cua, quả mọng…", fx: "mỗi ô một món, chín cùng nhau"})
	case sim.StationRack:
		out = append(out,
			cookLine{item: sim.ItemJerky, name: "Thịt khô", needs: "thịt sống", fx: "gần lửa thì hun khói, nhanh gấp đôi"},
			cookLine{item: sim.ItemDriedFish, name: "Cá khô", needs: "cá sống"},
			cookLine{item: sim.ItemSmokedMeat, name: "Thịt hun khói", needs: "thịt muối (thịt + muối)"},
			cookLine{item: sim.ItemSalt, name: "Muối ×2", needs: "bát nước biển", fx: "giàn phải đặt gần biển"})
	}
	return out
}

func (g *Game) drawCook(screen *ebiten.Image, sw, sh int) {
	g.dim(screen, sw, sh, 0.5)
	c := &g.cook
	cv := &g.curr.Self.Cook
	inv := &g.curr.Self.Inv

	pw := float64(cookGrid) + 48 + cookSide
	ph := 46 + 18 + chestPx + 16 + 34 + 30 + float64(bagRows+1)*chestRow + 8 + 34 + 26
	px, py := float64(int((float64(sw)-pw)/2)), float64(int(max(8, (float64(sh)-ph)/2)))
	drawFrame(screen, float32(px), float32(py), float32(pw), float32(ph))
	c.x0, c.slotY = px+24, py+46+18
	c.outX = c.x0 + float64(cookGrid) - sim.CookOut*chestRow + chestGap
	c.invY = c.slotY + chestPx + 16 + 34 + 30

	g.text(screen, cv.Kind.Name(), px+20, py+10, colWhite, text.AlignStart, 2)
	msg, mcol := cookStateText(cv.Kind, cv.State, cv.Dish)
	g.text(screen, msg, c.x0+float64(cookGrid), py+18, mcol, text.AlignEnd, 1)

	// The station: ingredients, the arrow, what came out.
	g.text(screen, "Nguyên liệu", c.x0, c.slotY-17, colKey, text.AlignStart, 1)
	g.text(screen, "Món đã xong", c.outX, c.slotY-17, colKey, text.AlignStart, 1)
	style := func(i int) slotStyle {
		if i == c.hover {
			return slotFocus
		}
		return slotPlain
	}
	for i := range sim.CookSlots {
		x, y := c.stationSlotXY(i)
		g.drawSlotSized(screen, cv.Slots[i], x, y, chestPx, style(i))
	}
	ax0 := c.x0 + sim.CookIn*chestRow + 4
	ax1 := c.outX - 8
	g.drawCookArrow(screen, ax0, c.slotY+chestPx/2, ax1-ax0, cv.State == sim.CookWorking, float64(cv.Progress)/float64(max(1, cv.Total)))
	if cv.State == sim.CookWorking && cv.Total > 0 {
		g.text(screen, clockText(int(cv.Total-cv.Progress)), (ax0+ax1)/2, c.slotY+chestPx/2+10, colMuted, text.AlignCenter, 1)
	}

	// The fire: its gauge and the button to feed it.
	fy := c.slotY + chestPx + 16
	c.btnW = [2]float64{}
	if cv.Kind.NeedsHeat() {
		most := sim.FireMaxFuel
		if cv.Kind == sim.StationOven {
			most = sim.OvenMaxFuel
		}
		frac := clamp01(float64(cv.Fuel) / float64(most))
		g.drawFlameIcon(screen, float32(c.x0)+9, float32(fy)+13, cv.Fuel > 0)
		bx, bw := float32(c.x0)+24, float32(170)
		fillRoundRect(screen, bx, float32(fy)+7, bw, 12, 6, color.RGBA{0, 0, 0, 150})
		if frac > 0 {
			fillRoundRect(screen, bx+1, float32(fy)+8, (bw-2)*float32(frac), 10, 5, color.RGBA{240, 140, 40, 255})
			fillRoundRect(screen, bx+2, float32(fy)+9, (bw-4)*float32(frac), 3, 2, color.RGBA{255, 214, 110, 200})
		}
		label := "Lửa: " + clockText(int(cv.Fuel))
		col := colMuted
		if cv.Fuel <= 0 {
			label, col = "Lửa đã tắt", colBad
		}
		g.text(screen, label, float64(bx+bw)+10, fy+4, col, text.AlignStart, 1)

		fuel := inv.Count(sim.ItemCoal) + inv.Count(sim.ItemWood)
		blabel := fmt.Sprintf("Thêm củi (còn %d)", fuel)
		c.btnW[cookBtnStoke], c.btnH[cookBtnStoke] = g.textWidth(blabel)+24, 26
		c.btnX[cookBtnStoke], c.btnY[cookBtnStoke] = c.x0+float64(cookGrid)-c.btnW[cookBtnStoke], fy
		g.drawCookButton(screen, cookBtnStoke, blabel, fuel > 0)
	} else {
		hint := "Không cần lửa"
		if cv.Kind == sim.StationRack {
			hint = "Không cần lửa — đặt gần lửa trại thì hun khói, khô nhanh gấp đôi"
		}
		g.text(screen, hint, c.x0, fy+4, colMuted, text.AlignStart, 1)
	}

	// My bag and hotbar.
	vector.FillRect(screen, float32(c.x0), float32(c.invY)-21, float32(cookGrid), 1, fade(colKey, 50), false)
	g.text(screen, "Túi đồ", c.x0, c.invY-18, colKey, text.AlignStart, 1)
	g.text(screen, "chuột trái: cả chồng · chuột phải: 1 cái", c.x0+float64(cookGrid), c.invY-18, colMuted, text.AlignEnd, 1)
	for i := range sim.NumSlots {
		x, y := c.invSlotXY(i)
		g.drawSlotSized(screen, inv[i], x, y, chestPx, style(sim.CookSlots+i))
		if !inv[i].Empty() && !sim.StationAccepts(cv.Kind, inv[i].Item) {
			fillRoundRect(screen, float32(x), float32(y), chestPx, chestPx, 7, color.RGBA{10, 11, 17, 170}) // of no use here
		}
		if sim.IsHotbar(i) {
			g.drawKeyBadge(screen, hotbarLabel(i), x-3, y-3, i == g.hotSel)
		}
	}

	// What is under the pointer, and the way to take the station down.
	by := c.invY + float64(bagRows+1)*chestRow + 8 + 10
	var s sim.Slot
	switch {
	case c.hover >= sim.CookSlots:
		s = inv[c.hover-sim.CookSlots]
	case c.hover >= 0:
		s = cv.Slots[c.hover]
	}
	if !s.Empty() {
		where := "→ lấy về túi"
		if c.hover >= sim.CookSlots {
			where = "→ bỏ vào " + strings.ToLower(cv.Kind.Name())
			if !sim.StationAccepts(cv.Kind, s.Item) {
				where = "— không dùng được ở đây"
			}
		}
		g.text(screen, fmt.Sprintf("%s ×%d  %s", s.Item.Def().Name, s.Count, where), c.x0, by, colWhite, text.AlignStart, 1)
	}
	if cv.Slots == ([sim.CookSlots]sim.Slot{}) {
		label := "Dỡ " + strings.ToLower(cv.Kind.Name())
		if cv.Kind == sim.StationPot || cv.Kind == sim.StationGrill || cv.Kind == sim.StationPan {
			label = "Nhấc " + strings.ToLower(cv.Kind.Name()) + " khỏi lửa"
		}
		c.btnW[cookBtnDown], c.btnH[cookBtnDown] = g.textWidth(label)+24, 26
		c.btnX[cookBtnDown], c.btnY[cookBtnDown] = c.x0+float64(cookGrid)-c.btnW[cookBtnDown], by-6
		g.drawCookButton(screen, cookBtnDown, label, true)
	}
	g.text(screen, "E / B / Esc: đóng  ·  đi xa cũng tự đóng", c.x0+float64(cookGrid)/2, py+ph-24, colMuted, text.AlignCenter, 1)

	// The dishes it knows, down the side.
	sx := c.x0 + float64(cookGrid) + 24
	vector.FillRect(screen, float32(sx)-12, float32(py)+14, 1, float32(ph)-28, fade(colKey, 50), false)
	g.text(screen, "Công thức", sx, py+16, colKey, text.AlignStart, 1)
	ly := py + 40
	sideW := pw - (sx - px) - 18
	for _, l := range cookLines(cv.Kind) {
		g.drawItem(screen, l.item, sx, ly+2, 32, 1)
		g.text(screen, l.name, sx+38, ly-2, colWhite, text.AlignStart, 1)
		y := ly + 15
		for _, line := range wrapText(l.needs, sideW-38) {
			g.text(screen, line, sx+38, y, colMuted, text.AlignStart, 1)
			y += 15
		}
		if l.fx != "" {
			for _, line := range wrapText(l.fx, sideW-38) {
				g.text(screen, line, sx+38, y, colGood, text.AlignStart, 1)
				y += 15
			}
		}
		ly = max(y+7, ly+42)
	}
	c.drawn = true
}

// wrapText breaks s into lines no wider than w.
func wrapText(s string, w float64) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		try := word
		if line != "" {
			try = line + " " + word
		}
		if line != "" && textWidth(try, 1) > w {
			lines = append(lines, line)
			line = word
			continue
		}
		line = try
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func (g *Game) drawCookButton(screen *ebiten.Image, b int, label string, on bool) {
	c := &g.cook
	bg := color.RGBA{0, 0, 0, 90}
	col := colKey
	if !on {
		col = colMuted
	} else if c.btn == b {
		bg = color.RGBA{58, 46, 22, 220}
	}
	fillRoundRect(screen, float32(c.btnX[b]), float32(c.btnY[b]), float32(c.btnW[b]), float32(c.btnH[b]), 7, bg)
	strokeRoundRect(screen, float32(c.btnX[b])+0.5, float32(c.btnY[b])+0.5, float32(c.btnW[b])-1, float32(c.btnH[b])-1, 7, fade(col, 90))
	g.text(screen, label, c.btnX[b]+c.btnW[b]/2, c.btnY[b]+5, col, text.AlignCenter, 1)
}

// drawCookArrow is the arrow from the ingredients to the dish, w long with
// its middle line at y: a groove that fills with fire as the dish cooks.
func (g *Game) drawCookArrow(screen *ebiten.Image, x, y, w float64, on bool, frac float64) {
	const h, head = 10.0, 16.0
	shaft := w - head
	path := func(x1 float64) *vector.Path {
		var p vector.Path
		p.MoveTo(float32(x), float32(y-h/2))
		if x1 <= x+shaft {
			p.LineTo(float32(x1), float32(y-h/2))
			p.LineTo(float32(x1), float32(y+h/2))
		} else {
			t := (x1 - (x + shaft)) / head // how far into the head
			p.LineTo(float32(x+shaft), float32(y-h/2))
			p.LineTo(float32(x+shaft), float32(y-h))
			p.LineTo(float32(x1), float32(y-h*(1-t)))
			p.LineTo(float32(x1), float32(y+h*(1-t)))
			p.LineTo(float32(x+shaft), float32(y+h))
			p.LineTo(float32(x+shaft), float32(y+h/2))
		}
		p.LineTo(float32(x), float32(y+h/2))
		p.Close()
		return &p
	}
	fill := func(x1 float64, col color.Color) {
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(col)
		vector.FillPath(screen, path(x1), nil, op)
	}
	fill(x+w, color.RGBA{0, 0, 0, 150})
	if on && frac > 0 {
		fill(x+w*clamp01(frac), color.RGBA{240, 140, 40, 255})
		// A bright bead running along it, so it is seen to be alive.
		bx := x + w*clamp01(frac)
		pulse := 0.6 + 0.4*math.Sin(float64(g.frame)*0.2)
		vector.FillCircle(screen, float32(bx), float32(y), float32(2.5+pulse), color.RGBA{255, 232, 150, 255}, true)
	}
}

// drawFlameIcon is a little flame, lit or a cold grey.
func (g *Game) drawFlameIcon(screen *ebiten.Image, x, y float32, lit bool) {
	outer, inner := color.RGBA{232, 96, 32, 255}, color.RGBA{255, 208, 90, 255}
	if !lit {
		outer, inner = color.RGBA{90, 92, 104, 255}, color.RGBA{130, 132, 144, 255}
	}
	drawDrop(screen, x, y-1, 18, outer)
	drawDrop(screen, x, y+1.5, 10, inner)
}
