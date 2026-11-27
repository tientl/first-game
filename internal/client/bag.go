package client

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The bag (Tab, I or B) shows every inventory slot: the bag's rows, with the hotbar
// under them. Stacks are moved by picking one up and putting it down. Below
// that is what the pointed-at item is, and the recipes related to the item
// last clicked, with what is missing.
// The server owns the inventory: the bag only sends move commands.

var (
	itemInfo = [sim.NumItems]string{
		sim.ItemWood:  "Chặt cây mà có. Thêm vào lửa trại để lửa cháy lâu hơn.",
		sim.ItemStone: "Đập tảng đá mà có — cần Cúp đá.",
		sim.ItemGrass: "Cắt bụi cỏ mà có. Làm mồi nhóm lửa.",
		sim.ItemBerry: "Hái từ bụi quả mọng.",
		sim.ItemStick: "Nhặt dưới đất, quanh các bụi cây và trong rừng.",
		sim.ItemAxe:   "Dụng cụ. Có rìu trong túi thì chặt được cây; rìu bậc cao hơn chặt nhanh hơn.",
		sim.ItemMeat:  "Bắt thỏ (rình lúc nó chạy mệt, hoặc lúc ngủ ban đêm) hay đánh rắn mà có. Đứng gần lửa trại, nhấp chuột phải để nướng.",
		sim.ItemWorm:  "Bắt trên bãi cỏ và trong rừng. Làm mồi câu cá.",
		sim.ItemFish:  "Cầm Cần câu, đứng ở bờ, nhấp chuột vào con cá dưới nước; mỗi lần câu tốn 1 Giun. Đứng gần lửa trại, nhấp chuột phải để nướng.",

		sim.ItemCookedMeat: "Cầm thịt sống, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",
		sim.ItemCookedFish: "Cầm cá sống, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",
		sim.ItemPickaxe:    "Dụng cụ. Có cúp trong túi thì đập được tảng đá, mỏ than và mỏ đồng.",
		sim.ItemTorch:      "Cầm trên tay (ô nhanh) để có ánh sáng và sưởi ấm cả người đứng gần. Mỗi cây cháy 90 giây khi cầm.",
		sim.ItemSpear:      "Cầm trên tay: đánh thú mạnh gấp đôi (rắn chỉ còn 2 nhát) và với xa hơn nửa ô.",
		sim.ItemRod:        "Cầm trên tay để câu cá ở bờ nước; mỗi lần câu vẫn tốn 1 Giun làm mồi.",
		sim.ItemRoastBerry: "Cầm quả mọng, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",

		sim.ItemPebble:      "Nhặt dưới đất, nhiều ở bãi đá và bãi cát. Làm rìu, cúp và cuốc.",
		sim.ItemHoe:         "Cầm trên tay (ô nhanh), nhấp chuột (hoặc Space) để xới ô đất trước mặt thành luống; đôi khi đào được giun hay khoai.",
		sim.ItemPotato:      "Ăn sống được, nướng thì no hơn (chuột phải cạnh lửa trại). Nhấp chuột vào luống đất để trồng: 1 ngày sau đào được 3 củ.",
		sim.ItemRoastPotato: "Cầm khoai lang, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",
		sim.ItemCactus:      "Chặt xương rồng ở hoang mạc (cần rìu). Ăn được, đỡ đói lúc đi xa.",
		sim.ItemMushroom:    "Hái trong đầm lầy và trong hang. Ăn được.",
		sim.ItemCoal:        "Đào ở mỏ than trong hang và quanh núi lửa (cần cúp đá). Thêm vào lửa trại: cháy lâu gấp đôi gỗ.",
		sim.ItemClam:        "Nhặt ở bãi sò trên cát sát mép nước; thủy triều mang sò mới tới mỗi ngày. Ăn sống được, nướng thì ngon hơn (chuột phải cạnh lửa trại).",
		sim.ItemRoastClam:   "Cầm sò, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng 2 con.",
		sim.ItemCrabMeat:    "Bắt cua trên bãi cát sát mép nước: chặn đường ra biển kẻo nó lặn mất, coi chừng bị kẹp. Đứng gần lửa trại, nhấp chuột phải để nướng.",
		sim.ItemCookedCrab:  "Cầm thịt cua, đứng gần lửa trại đang cháy, nhấp chuột phải để nướng.",
		sim.ItemHide:        "Lột từ khủng long và thằn lằn bay khi săn được. Dùng may đồ da.",
		sim.ItemBone:        "Lấy từ khủng long và thằn lằn cổ dài khi săn được. Dùng làm dụng cụ và giáp.",
		sim.ItemRexTooth:    "Răng khủng long bạo chúa: lưỡi sắc nhất thời tiền sử. Chỉ có khi hạ được bạo chúa.",
		sim.ItemSaddle:      "Có trong túi thì cưỡi được thú đã thuần (khủng long ba sừng, raptor, voi ma mút): đứng gần nó, nhấn E.",
		sim.ItemFur:         "Lông dày của voi ma mút và hổ răng kiếm. Dùng may Áo lông.",
		sim.ItemIvory:       "Ngà voi ma mút. Làm mũi Giáo ngà.",
		sim.ItemFurCoat:     "Có trong túi thì không bị lạnh khi đi trên tuyết (ban đêm vẫn cần lửa).",
		sim.ItemIvorySpear:  "Cầm trên tay: đánh mạnh gấp bốn tay không và với xa nhất.",
		sim.ItemBattleAxe:   "Cầm trên tay: nhát chém nặng nhất (70), gấp bảy tay không, nhưng vung chậm. Không dùng chặt cây.",
	}
	colBad = color.RGBA{236, 110, 110, 255}
)

const (
	slotPx  = 56 // one inventory slot on screen, in the bag and on the hotbar
	slotGap = 6

	bagCols = sim.HotbarSlots // the bag's rows are as wide as the hotbar
	bagRows = (sim.BagSlots + bagCols - 1) / bagCols

	cardH    = 50.0 // one recipe
	cardGap  = 8.0
	cardMinW = 250.0
	cardCols = 3 // recipes per row: enough rows stay short for a 720px window
)

type slotStyle uint8

const (
	slotPlain  slotStyle = iota
	slotHeld             // the hotbar slot in the player's hand
	slotFocus            // under the bag's cursor
	slotLifted           // picked up in the bag, waiting to be put down
)

// drawSlot draws one inventory slot and what is in it.
func (g *Game) drawSlot(dst *uiDst, s sim.Slot, x, y float64, st slotStyle) {
	g.drawSlotSized(dst, s, x, y, slotPx, st)
}

// drawSlotSized draws a slot px across: a well sunk in the hide, lit warm
// round its rim when it is held or pointed at, its item's icon as big as
// fits, and the count.
func (g *Game) drawSlotSized(dst *uiDst, s sim.Slot, x, y float64, px int, st slotStyle) {
	p := float64(px)
	r := min(8, p/6)
	switch st {
	case slotHeld, slotFocus:
		pulse := 0.5 + 0.5*math.Sin(float64(g.frame)*0.07)
		glow(dst, x-1, y-1, p+2, p+2, r+1, colAmberGlow, 0.5+0.2*pulse)
		well(dst, x, y, p, p, r)
		drawBaked(dst, fmt.Sprintf("slot lit %d", px), x, y, p, p, 2, func(b *uiDst) {
			fillRoundRect(b, 0, 0, float32(p), float32(p), float32(r), color.RGBA{120, 72, 18, 70}) // a warm light in it
			strokeRoundRectHair(b, -1, -1, p+2, p+2, r+1, 1.5, colAmber)
			strokeRoundRectHair(b, 0.5, 0.5, p-1, p-1, r-0.5, b.hair(), color.RGBA{255, 236, 180, 150})
		})
	case slotLifted:
		well(dst, x, y, p, p, r)
		strokeRoundRectHair(dst, x-1, y-1, p+2, p+2, r+1, 1, color.RGBA{236, 224, 196, 160})
	default:
		well(dst, x, y, p, p, r)
	}
	noteBox(dst, x, y, p, p)
	if s.Empty() {
		return
	}
	a := float32(1)
	if st == slotLifted {
		a = 0.3
	}
	// The icon is the old one's size, 16 UI pixels times as many as fit.
	is := float64(16 * max(1, (px-8)/16))
	g.drawItem(dst, s.Item, x+(p-is)/2, y+(p-is)/2, is, a) // see iconhd.go
	if s.Item.MaxCount() > 1 {
		drawCount(dst, s.Count, x+p-4, y+p-4)
	}
}

// drawCount is a stack's count, small and heavy with a dark rim, its
// bottom right corner at (x, y).
func drawCount(dst *uiDst, n int, x, y float64) {
	const scale = 0.92
	uiTextW(dst, fmt.Sprintf("%d", n), x, y-baseUI*scale-1, colWhite, text.AlignEnd, scale, wHeavy+60, fxOutline)
}

// strokeRoundRectHair draws the rim of a rounded rectangle w UI pixels
// thick, its middle on the rectangle's edge.
func strokeRoundRectHair(dst *uiDst, x, y, w, h, r, sw float64, c color.RGBA) {
	var p vector.Path
	R := float32(min(r, w/2, h/2))
	X, Y, W, H := float32(x), float32(y), float32(w), float32(h)
	p.MoveTo(X+R, Y)
	p.LineTo(X+W-R, Y)
	p.ArcTo(X+W, Y, X+W, Y+R, R)
	p.LineTo(X+W, Y+H-R)
	p.ArcTo(X+W, Y+H, X+W-R, Y+H, R)
	p.LineTo(X+R, Y+H)
	p.ArcTo(X, Y+H, X, Y+H-R, R)
	p.LineTo(X, Y+R)
	p.ArcTo(X, Y, X+R, Y, R)
	p.Close()
	var q vector.Path
	q.AddPath(&p, &vector.AddPathOptions{GeoM: dst.geoM()})
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst.img, &q, &vector.StrokeOptions{Width: float32(sw * dst.k), LineJoin: vector.LineJoinRound}, op)
}

type bagView struct {
	cursor  int  // the slot pointed at
	lifted  int  // the slot picked up to move, while lifting
	lifting bool // a stack is picked up
	byMouse bool // it was picked up with the mouse, so it follows the pointer

	pressX, pressY float64 // where the left button went down, to tell a drag from a click

	// focus is the item last clicked: the recipes listed are those that make
	// or use it. It is not the cursor, which follows the pointer across the
	// hotbar on the way down to the recipes.
	focus sim.ItemKind

	// The recipe cards scroll by rows when they do not all fit.
	cardScroll int     // rows scrolled past
	wheel      float64 // wheel movement not yet a whole row

	// Layout from the last draw, for the mouse.
	drawn          bool
	panelX, panelY float64 // the bag's window: a stack let go outside it is thrown away; see toss.go
	panelW, panelH float64
	gridX, gridY   float64 // the bag's first slot
	hotY           float64 // the hotbar row
	cardX, cardY   float64 // the first recipe
	cardW          float64
	cardRowsShown  int // rows of cards the panel had room for
	hoverCard      int
	lastMX, lastMY int
}

// open resets the bag for a fresh look, pointing at the held slot and
// listing the recipes for focus, the item held.
func (b *bagView) open(held int, focus sim.ItemKind) {
	b.cursor, b.lifting, b.hoverCard = held, false, -1
	b.setFocus(focus)
}

// setFocus lists the recipes for k, from the top.
func (b *bagView) setFocus(k sim.ItemKind) {
	if k != b.focus {
		b.focus, b.cardScroll = k, 0
	}
}

// cardShown reports whether card i is in the rows the panel shows.
func (b *bagView) cardShown(i int) bool {
	row := i/cardCols - b.cardScroll
	return row >= 0 && row < b.cardRowsShown
}

// scrollCards moves the cards by rows, keeping them in range.
func (b *bagView) scrollCards(rows int) {
	last := (len(b.shownRecipes())+cardCols-1)/cardCols - b.cardRowsShown
	b.cardScroll = max(0, min(b.cardScroll+rows, last))
}

// slotPos is where slot i is in the bag's grid: the bag rows, then the
// hotbar as the last row.
func slotPos(i int) (row, col int) {
	if sim.IsHotbar(i) {
		return bagRows, i
	}
	j := i - sim.HotbarSlots
	return j / bagCols, j % bagCols
}

func slotAtPos(row, col int) int {
	if row == bagRows {
		return col
	}
	return min(sim.NumSlots-1, sim.HotbarSlots+row*bagCols+col)
}

func (b *bagView) slotXY(i int) (x, y float64) {
	row, col := slotPos(i)
	x = b.gridX + float64(col*(slotPx+slotGap))
	if row == bagRows {
		return x, b.hotY
	}
	return x, b.gridY + float64(row*(slotPx+slotGap))
}

// slotAt returns the slot under screen point (x, y).
func (b *bagView) slotAt(x, y float64) (int, bool) {
	for i := range sim.NumSlots {
		sx, sy := b.slotXY(i)
		if x >= sx && x < sx+slotPx && y >= sy && y < sy+slotPx {
			return i, true
		}
	}
	return 0, false
}

func recipes() []sim.Recipe {
	var rs []sim.Recipe
	for id := sim.RecipeID(0); id < sim.NumRecipes; id++ {
		if r, ok := sim.RecipeByID(id); ok {
			rs = append(rs, r)
		}
	}
	return rs
}

// relatedRecipes are the recipes that make item, then those that use it.
func relatedRecipes(item sim.ItemKind) []sim.Recipe {
	if !item.Valid() {
		return nil
	}
	var makes, uses []sim.Recipe
	for _, r := range recipes() {
		switch {
		case r.Gives == item:
			makes = append(makes, r)
		case slices.ContainsFunc(r.Inputs, func(in sim.Ingredient) bool { return in.Item == item }):
			uses = append(uses, r)
		}
	}
	return append(makes, uses...)
}

// shownRecipes are the recipe cards the bag lists, in order.
func (b *bagView) shownRecipes() []sim.Recipe { return relatedRecipes(b.focus) }

// cardRows is how many rows of recipe cards the bag makes room for: enough
// for the item with the most related recipes, so the panel keeps its size
// whatever is clicked.
var cardRows = func() int {
	most := 1
	for k := sim.ItemKind(1); k < sim.NumItems; k++ {
		most = max(most, len(relatedRecipes(k)))
	}
	return (most + cardCols - 1) / cardCols
}()

func (b *bagView) cardXY(i int) (x, y float64) {
	return b.cardX + float64(i%cardCols)*(b.cardW+12), b.cardY + float64(i/cardCols-b.cardScroll)*(cardH+cardGap)
}

func justPressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

// updateBag moves the cursor, picks stacks up and puts them down, and
// crafts a recipe that is clicked.
func (g *Game) updateBag() {
	b := &g.bag
	inv := &g.curr.Self.Inv
	prev := b.cursor
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)

	row, col := slotPos(b.cursor)
	switch {
	case justPressed(ebiten.KeyArrowLeft, ebiten.KeyA):
		col--
	case justPressed(ebiten.KeyArrowRight, ebiten.KeyD):
		col++
	case justPressed(ebiten.KeyArrowUp, ebiten.KeyW):
		row--
	case justPressed(ebiten.KeyArrowDown, ebiten.KeyS):
		row++
	}
	b.cursor = slotAtPos((row+bagRows+1)%(bagRows+1), (col+bagCols)%bagCols)
	if justPressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace) {
		b.byMouse = false
		g.bagActivate(shift)
	}

	b.hoverCard = -1
	// The wheel scrolls the recipes; the hotbar ignores it while the bag is open.
	_, wy := ebiten.Wheel()
	b.wheel += wy
	for ; b.wheel >= 1; b.wheel-- {
		b.scrollCards(-1)
	}
	for ; b.wheel <= -1; b.wheel++ {
		b.scrollCards(1)
	}
	if b.drawn {
		mx, my := uiCursor()
		moved := mx != b.lastMX || my != b.lastMY
		b.lastMX, b.lastMY = mx, my
		g.bagMouse(float64(mx), float64(my), moved,
			inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
			inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft), shift)
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			g.bagRightClick(float64(mx), float64(my)) // put it back, or throw one away
		}
	}

	// A number key sends what is pointed at to that hotbar slot.
	if i, ok := hotbarKey(); ok && i != b.cursor && !inv[b.cursor].Empty() {
		g.command(sim.Command{Kind: sim.CmdMoveSlot, Slot: uint8(b.cursor), To: uint8(i)})
		b.lifting = false
		snd.ui("ui_select")
	}
	if b.lifting && inv[b.lifted].Empty() {
		b.lifting = false // eaten or used up meanwhile
	}
	if b.cursor != prev {
		snd.ui("ui_move")
	}
}

// dragPx is how far the pointer must travel with the button down for a
// press to count as a drag rather than a click.
const dragPx = 6

// bagActivate is a click or Enter on the slot at the cursor. With Shift it
// sends the stack across between bag and hotbar; while a stack is lifted it
// puts it down there (into, onto or swapped); otherwise it lifts the stack.
func (g *Game) bagActivate(shift bool) {
	b := &g.bag
	inv := &g.curr.Self.Inv
	switch {
	case shift && !inv[b.cursor].Empty():
		g.command(sim.Command{Kind: sim.CmdQuickMove, Slot: uint8(b.cursor)})
		b.lifting = false
		b.setFocus(inv[b.cursor].Item)
		snd.ui("ui_select")
	case b.lifting:
		if b.lifted != b.cursor {
			g.command(sim.Command{Kind: sim.CmdMoveSlot, Slot: uint8(b.lifted), To: uint8(b.cursor)})
		}
		b.lifting = false
		snd.ui("ui_select")
	case !inv[b.cursor].Empty():
		b.lifted, b.lifting = b.cursor, true
		b.setFocus(inv[b.cursor].Item)
		snd.ui("ui_select")
	}
}

// bagMouse handles the pointer at (x, y) in the bag. Pressing on a stack
// lifts it and it follows the pointer. Letting go after dragging drops it on
// the slot underneath, on the ground outside the bag's window, or back where
// it was elsewhere; letting go where it was pressed keeps it lifted, so a
// second click puts it down.
// press and release are the left button's edges this frame.
func (g *Game) bagMouse(x, y float64, moved, press, release, shift bool) {
	b := &g.bag
	i, onSlot := b.slotAt(x, y)
	if onSlot && (moved || press) {
		b.cursor = i // a still mouse does not fight the keys
	}
	if press && onSlot {
		b.byMouse, b.pressX, b.pressY = true, x, y
		g.bagActivate(shift)
	}
	if release && b.lifting && b.byMouse && math.Hypot(x-b.pressX, y-b.pressY) > dragPx {
		if b.outsideBag(x, y) {
			g.tossSlot(b.lifted, true) // see toss.go
		} else if onSlot && i != b.lifted {
			g.command(sim.Command{Kind: sim.CmdMoveSlot, Slot: uint8(b.lifted), To: uint8(i)})
		}
		b.lifting = false
		snd.ui("ui_select")
	}
	for i, r := range b.shownRecipes() {
		if !b.cardShown(i) {
			continue
		}
		cx, cy := b.cardXY(i)
		if x >= cx && x <= cx+b.cardW && y >= cy && y <= cy+cardH {
			b.hoverCard = i
			if press {
				g.craftRecipe(r)  // a structure is then placed with the mouse; see mouse.go
				b.lifting = false // clicking the item to see its recipes lifted it: put it back
			}
		}
	}
}

func (g *Game) drawBag(screen *uiDst, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	b := &g.bag
	inv := &g.curr.Self.Inv
	rs := b.shownRecipes()

	gridW := float64(bagCols*slotPx + (bagCols-1)*slotGap)
	pw := max(gridW+48, cardCols*cardMinW+(cardCols-1)*12+32)
	gridH := float64(bagRows*(slotPx+slotGap) - slotGap)
	// As many rows of cards as the window has room for; the rest scroll.
	fixedH := 44 + gridH + 24 + slotPx + 14 + 70 + 20 + 52 // fits a 960×540 canvas: Full HD at UI scale 2
	b.cardRowsShown = max(1, min(cardRows, int((float64(sh)-16-fixedH+cardGap)/(cardH+cardGap))))
	b.scrollCards(0)
	cardsH := float64(b.cardRowsShown)*(cardH+cardGap) - cardGap
	ph := fixedH + cardsH
	px, py := float64(int((float64(sw)-pw)/2)), float64(int(max(8, (float64(sh)-ph)/2)))
	drawFrame(screen, float32(px), float32(py), float32(pw), float32(ph)) // see hud.go
	b.panelX, b.panelY, b.panelW, b.panelH = px, py, pw, ph

	used := 0
	for _, s := range inv[sim.HotbarSlots:] {
		if !s.Empty() {
			used++
		}
	}
	g.text(screen, "Túi đồ", px+20, py+10, colWhite, text.AlignStart, 2)
	g.text(screen, fmt.Sprintf("%d/%d ô  ·  mỗi ô chứa tối đa %d", used, sim.BagSlots, sim.MaxStack), px+pw-20, py+18, colMuted, text.AlignEnd, 1)

	// Slots: the bag, then the hotbar under it.
	b.gridX, b.gridY = px+(pw-gridW)/2, py+44
	b.hotY = b.gridY + gridH + 24
	fillRect(screen, float32(b.gridX), float32(b.hotY)-21, float32(gridW), 1, fade(colKey, 50), false)
	g.text(screen, "Ô nhanh — chọn bằng phím 1-0, món ở ô viền vàng là món đang cầm", b.gridX, b.hotY-18, colMuted, text.AlignStart, 1)
	for i := range sim.NumSlots {
		x, y := b.slotXY(i)
		st := slotPlain
		switch {
		case b.lifting && i == b.lifted:
			st = slotLifted
		case i == b.cursor:
			st = slotFocus
		}
		g.drawSlot(screen, inv[i], x, y, st)
		if sim.IsHotbar(i) {
			g.drawKeyBadge(screen, hotbarLabel(i), x-3, y-3, i == g.hotSel)
			if i == g.hotSel {
				fillRoundRect(screen, float32(x)+14, float32(y)+slotPx+4, slotPx-28, 3, 1.5, colKey)
			}
		}
	}
	// What is being moved hovers: on the pointer when the mouse carries it,
	// else over the slot it would go to. It is drawn last, over everything.
	defer func() {
		if !b.lifting {
			return
		}
		s := inv[b.lifted]
		if s.Empty() {
			return
		}
		if b.byMouse {
			x, y := float64(b.lastMX)-24, float64(b.lastMY)-24
			g.drawItem(screen, s.Item, x, y, 48, 0.9)
			if s.Item.MaxCount() > 1 {
				g.text(screen, fmt.Sprintf("%d", s.Count), x+51, y+33, colWhite, text.AlignEnd, 1)
			}
			g.drawTossHint(screen, x+24, y+24)
			return
		}
		x, y := b.slotXY(b.cursor)
		g.drawItem(screen, s.Item, x+slotPx-26, y-14, 32, 1)
	}()

	// What the pointed-at slot holds.
	s := inv[b.cursor]
	dx, dy := px+24, b.hotY+slotPx+14
	if s.Empty() {
		g.text(screen, "Ô trống", dx, dy, colMuted, text.AlignStart, 2)
		g.text(screen, "Nhấp vào ô có đồ để nhấc lên, rồi nhấp vào ô khác để đặt xuống, gộp hoặc đổi chỗ.", dx, dy+32, colMuted, text.AlignStart, 1)
	} else {
		def := s.Item.Def()
		g.text(screen, def.Name, dx, dy, colWhite, text.AlignStart, 2)
		g.text(screen, fmt.Sprintf("ô này: %d / %d   ·   cả túi: %d", s.Count, s.Item.MaxCount(), inv.Count(s.Item)),
			px+pw-24, dy+6, colMuted, text.AlignEnd, 1)
		g.text(screen, itemInfo[s.Item], dx, dy+32, colMuted, text.AlignStart, 1)
		if s.Item.Edible() {
			g.text(screen, "E", dx, dy+52, colKey, text.AlignStart, 1)
			eat, col := fmt.Sprintf("ăn: %+g no bụng · %+g máu", def.Hunger, def.Health), colGood
			if def.Health < 0 {
				eat, col = eat+" — nên nướng trước", colHunger
			}
			g.text(screen, eat, dx+18, dy+52, col, text.AlignStart, 1)
		} else {
			var uses []string
			for _, r := range recipes() {
				for _, in := range r.Inputs {
					if in.Item == s.Item {
						uses = append(uses, fmt.Sprintf("%s (cần %d)", r.Name, in.Count))
					}
				}
			}
			if len(uses) > 0 {
				g.text(screen, fitText("Dùng để làm: "+strings.Join(uses, ", "), px+pw-24-dx, 1), dx, dy+52, colWhite, text.AlignStart, 1)
			}
		}
	}

	// The recipes related to the item last clicked, a few to a row.
	sy := dy + 72
	fillRect(screen, float32(px)+20, float32(sy)-10, float32(pw)-40, 1, fade(colWhite, 40), false)
	b.cardX, b.cardY, b.cardW = px+16, sy+20, (pw-32-12*(cardCols-1))/cardCols
	switch {
	case !b.focus.Valid():
		g.text(screen, "Chế tạo", px+24, sy, colKey, text.AlignStart, 1)
		g.text(screen, "Nhấp vào một món để xem làm ra nó thế nào và nó dùng để làm gì.", px+24, b.cardY+8, colMuted, text.AlignStart, 1)
	case len(rs) == 0:
		g.text(screen, "Chế tạo — "+b.focus.Def().Name, px+24, sy, colKey, text.AlignStart, 1)
		g.text(screen, "Chưa có công thức nào làm ra hay dùng món này.", px+24, b.cardY+8, colMuted, text.AlignStart, 1)
	default:
		g.text(screen, "Chế tạo — "+b.focus.Def().Name, px+24, sy, colKey, text.AlignStart, 1)
		hint := "nhấp để làm"
		if rows := (len(rs) + cardCols - 1) / cardCols; rows > b.cardRowsShown {
			hint = fmt.Sprintf("nhấp để làm  ·  lăn chuột xem thêm (hàng %d-%d / %d)", b.cardScroll+1, b.cardScroll+b.cardRowsShown, rows)
		}
		g.text(screen, hint, px+pw-24, sy, colMuted, text.AlignEnd, 1)
	}
	for i, r := range rs {
		if !b.cardShown(i) {
			continue
		}
		x, y := b.cardXY(i)
		g.drawRecipeCard(screen, r, x, y, b.cardW, i == b.hoverCard)
	}

	g.text(screen, "Kéo thả: di chuyển  ·  Shift+nhấp: túi ↔ ô nhanh  ·  1-0: vào ô nhanh  ·  E: ăn  ·  C: chế tạo  ·  Tab / Esc: đóng",
		px+pw/2, py+ph-26, colMuted, text.AlignCenter, 1)
	g.text(screen, "Vứt xuống đất: kéo ra ngoài túi (cả chồng)  ·  chuột phải hoặc G: vứt 1  ·  Shift+G: vứt cả chồng",
		px+pw/2, py+ph-44, colMuted, text.AlignCenter, 1)
	b.drawn = true
}

func (g *Game) drawRecipeCard(screen *uiDst, r sim.Recipe, x, y, w float64, hover bool) {
	inv := &g.curr.Self.Inv
	free := g.freeBuild() // nothing needed and nothing used: see buildpalette.go
	// A leather card: rimmed green when it can be made now, lit warm under
	// the pointer.
	l := panelLooks[psLeather]
	l.stitch = color.RGBA{}
	if g.canCraft(r) {
		l.rim = color.RGBA{112, 168, 88, 255}
	}
	if hover {
		glow(screen, x, y, w, cardH, 8, colAmberGlow, 0.28)
		l.body = color.RGBA{84, 62, 40, 248}
	}
	drawPanelLook(screen, psLeather, l, x, y, w, cardH, 8, 0.4)
	noteBox(screen, x, y, w, cardH)

	// What it makes: an item's icon, or a structure's world sprite, kept on
	// the card.
	if r.Gives.Valid() {
		g.drawItem(screen, r.Gives, x+8, y+(cardH-32)/2, 32, 1)
	} else {
		ix, iy := x+24, y+cardH-9
		card := screen.clip(x+2, y+2, 44, cardH-4)
		s, _ := g.entitySprite(protocol.EntityView{Kind: r.Places, Ready: true})
		drawAt(card, s, ix, iy, 1.5, 1.5, 0, nil)
		if r.Places == sim.EntityCampfire {
			drawAt(card, g.art.flames[(g.frame/5)%flameFrames], ix, iy+1, 1, 1, 0, nil)
			drawAt(card, g.art.campFront, ix, iy, 1.5, 1.5, 0, nil)
		}
	}

	tx := x + 52
	rx, ry := x+w-12, y+7
	status, scol := "", color.RGBA{}
	switch {
	case g.haveTool(r):
		status, scol = "đã có", colMuted
	case free:
	case r.NearFire && inv.HasAll(r.Inputs) && !g.byFire():
		status, scol = "ra gần lửa trại", colHunger
	case r.NearFurnace && inv.HasAll(r.Inputs) && !g.byFurnace():
		status, scol = "ra cạnh lò nung", colHunger
	case inv.HasAll(r.Inputs):
		status, scol = "làm được", colGood
		if !r.Gives.Valid() {
			status = "dựng được" // then placed with the mouse
		}
	default:
		status, scol = "thiếu", colMuted
	}
	sw := g.textWidth(status)
	if free {
		sw = g.textWidth("DEV") + 10
		g.drawDevBadge(screen, rx+2, ry-1)
	} else {
		g.text(screen, status, rx, ry, scol, text.AlignEnd, 1)
	}
	// The name, shortened if it would run into what is said of it.
	g.text(screen, fitText(r.Name, rx-sw-8-tx, 1), tx, y+7, colWhite, text.AlignStart, 1)

	// Ingredients: icon, count and name; just icon and count when the
	// names would not fit on the card.
	label := func(in sim.Ingredient, named bool) string {
		s := fmt.Sprintf("%d/%d", min(inv.Count(in.Item), in.Count), in.Count)
		if free {
			s = fmt.Sprintf("×%d", in.Count) // what it takes in a normal game
		}
		if named {
			s += " " + in.Item.Def().Name
		}
		return s
	}
	named, need := true, 0.0
	for _, in := range r.Inputs {
		need += 19 + g.textWidth(label(in, true)) + 12
	}
	if tx+need > x+w {
		named = false
	}
	cx := tx
	for _, in := range r.Inputs {
		g.drawItem(screen, in.Item, cx, y+27, 16, 1)
		c := colGood
		switch {
		case free:
			c = colMuted // not needed
		case inv.Count(in.Item) < in.Count:
			c = colBad
		}
		l := label(in, named)
		g.text(screen, l, cx+19, y+28, c, text.AlignStart, 1)
		cx += 19 + g.textWidth(l) + 12
	}
}
