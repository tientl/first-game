package client

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// A chest opens when the server says the player has one open (E or Space
// beside it, or a click on it), and shows its slots above the player's own
// bag and hotbar: a click on a stack moves it across. An old chest in a cave
// (sim.EntityTreasure) opens the same way with its hoard inside. Closing it
// here (E, B, Esc) tells the server; walking away closes it from the
// server's side.

// chestView is the chest window's state.
type chestView struct {
	closing  sim.EntityID // closed here; ignored until the server agrees
	hover    int          // slot under the pointer: chest 0..NumSlots-1, mine NumSlots.., or -1
	btnHover bool

	// Layout from the last draw, for the mouse.
	drawn                  bool
	chestX, chestY, invY   float64
	btnX, btnY, btnW, btnH float64
}

// The chest window's slots are smaller than the bag's, so a chest's rows and
// all of mine fit on a 960×540 canvas (Full HD at UI scale 2).
const (
	chestPx  = 44
	chestGap = 4
	chestRow = chestPx + chestGap

	chestRows = (sim.NumSlots + bagCols - 1) / bagCols
)

func (c *chestView) chestSlotXY(i int) (x, y float64) {
	return c.chestX + float64(i%bagCols*chestRow), c.chestY + float64(i/bagCols*chestRow)
}

// invSlotXY places my slots like the bag does: the bag rows, then the hotbar.
func (c *chestView) invSlotXY(i int) (x, y float64) {
	row, col := slotPos(i)
	y = c.invY + float64(row*chestRow)
	if row == bagRows {
		y += 8 // the hotbar sits a little apart
	}
	return c.chestX + float64(col*chestRow), y
}

// slotAt is the slot under (x, y), in UI pixels: chest slots first, then
// mine.
func (c *chestView) slotAt(x, y float64) int {
	in := func(sx, sy float64) bool { return x >= sx && x < sx+chestPx && y >= sy && y < sy+chestPx }
	for i := range sim.NumSlots {
		if in(c.chestSlotXY(i)) {
			return i
		}
		if in(c.invSlotXY(i)) {
			return sim.NumSlots + i
		}
	}
	return -1
}

// syncChest opens or closes the chest window to match the snapshot.
func (g *Game) syncChest() {
	switch id := g.curr.Self.Chest; {
	case id == 0:
		g.chest.closing = 0
		if g.overlay == overlayChest {
			g.setOverlay(overlayNone)
		}
	case id == g.chest.closing:
		// Closed here a moment ago; the server has not heard yet.
	case g.overlay != overlayChest && !g.curr.Self.Dead:
		g.setOverlay(overlayChest)
	}
}

// closeChest tells the server the chest window was closed here.
func (g *Game) closeChest() {
	g.chest.closing = g.curr.Self.Chest
	g.command(sim.Command{Kind: sim.CmdChestClose})
}

// openChestKind is the kind of the chest open: a built chest, or an old one
// from a cave (which cannot be taken down).
func (g *Game) openChestKind() sim.EntityKind {
	id := g.curr.Self.Chest
	for _, e := range g.curr.Entities {
		if e.ID == id {
			return e.Kind
		}
	}
	return sim.EntityChest
}

// updateChest moves stacks between the chest and my inventory on a click.
func (g *Game) updateChest() {
	mx, my := uiCursor()
	g.chestMouse(float64(mx), float64(my), inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft))
}

// chestMouse is the pointer at (x, y) on the UI canvas over the chest
// window, pressed or not.
func (g *Game) chestMouse(x, y float64, press bool) {
	c := &g.chest
	c.hover, c.btnHover = -1, false
	if !c.drawn {
		return
	}
	c.hover = c.slotAt(x, y)
	c.btnHover = c.btnW > 0 && x >= c.btnX && x < c.btnX+c.btnW && y >= c.btnY && y < c.btnY+c.btnH
	if !press {
		return
	}
	switch {
	case c.btnHover:
		g.command(sim.Command{Kind: sim.CmdChestTakeDown})
	case c.hover >= sim.NumSlots && !g.curr.Self.Inv[c.hover-sim.NumSlots].Empty():
		g.command(sim.Command{Kind: sim.CmdChestPut, Slot: uint8(c.hover - sim.NumSlots)})
		snd.ui("ui_select")
	case c.hover >= 0 && c.hover < sim.NumSlots && !g.curr.Self.ChestSlots[c.hover].Empty():
		g.command(sim.Command{Kind: sim.CmdChestTake, Slot: uint8(c.hover)})
		snd.ui("ui_select")
	}
}

// layout places the chest window on a canvas sw by sh: its frame, and the
// first slots of the chest and of my bag.
func (c *chestView) layout(sw, sh int) (px, py, pw, ph float64) {
	pw = float64(bagCols*chestRow-chestGap) + 48
	ph = 40 + chestRows*chestRow + 26 + float64(bagRows+1)*chestRow + 8 + 34 + 26
	px, py = float64(int((float64(sw)-pw)/2)), float64(int(max(8, (float64(sh)-ph)/2)))
	c.chestX, c.chestY = px+24, py+40
	c.invY = c.chestY + chestRows*chestRow + 26
	return px, py, pw, ph
}

func (g *Game) drawChest(screen *uiDst, sw, sh int) {
	g.dim(screen, sw, sh, 0.55)
	c := &g.chest
	contents, inv := &g.curr.Self.ChestSlots, &g.curr.Self.Inv
	kind := g.openChestKind()
	px, py, pw, ph := c.layout(sw, sh)
	gridW := float64(bagCols*chestRow - chestGap)
	drawFrame(screen, float32(px), float32(py), float32(pw), float32(ph)) // see hud.go

	title := "Rương"
	if kind == sim.EntityTreasure {
		title = "Rương cổ"
	}
	g.text(screen, title, px+20, py+10, colWhite, text.AlignStart, 2)
	g.text(screen, "Nhấp vào một món để chuyển giữa rương và túi", px+pw-20, py+18, colMuted, text.AlignEnd, 1)
	style := func(i int) slotStyle {
		if i == c.hover {
			return slotFocus
		}
		return slotPlain
	}
	for i := range sim.NumSlots {
		x, y := c.chestSlotXY(i)
		g.drawSlotSized(screen, contents[i], x, y, chestPx, style(i))
	}

	fillRect(screen, float32(c.chestX), float32(c.invY)-21, float32(gridW), 1, fade(colKey, 50), false)
	g.text(screen, "Túi đồ", c.chestX, c.invY-18, colKey, text.AlignStart, 1)
	for i := range sim.NumSlots {
		x, y := c.invSlotXY(i)
		g.drawSlotSized(screen, inv[i], x, y, chestPx, style(sim.NumSlots+i))
		if sim.IsHotbar(i) {
			g.drawKeyBadge(screen, hotbarLabel(i), x-3, y-3, i == g.hotSel)
		}
	}

	// What is under the pointer, and the way to take an empty chest down.
	fy := c.invY + float64(bagRows+1)*chestRow + 8 + 10
	var s sim.Slot
	switch {
	case c.hover >= sim.NumSlots:
		s = inv[c.hover-sim.NumSlots]
	case c.hover >= 0:
		s = contents[c.hover]
	}
	if !s.Empty() {
		where := "→ cất vào rương"
		if c.hover < sim.NumSlots {
			where = "→ lấy về túi"
		}
		g.text(screen, fmt.Sprintf("%s ×%d  %s", s.Item.Def().Name, s.Count, where), c.chestX, fy, colWhite, text.AlignStart, 1)
	}
	c.btnW = 0
	if *contents == (sim.Inventory{}) && kind == sim.EntityChest {
		label := fmt.Sprintf("Dỡ rương (lấy lại %d gỗ)", sim.ChestRefund)
		c.btnW, c.btnH = g.textWidth(label)+24, 26
		c.btnX, c.btnY = px+pw-24-c.btnW, fy-6
		bg := color.RGBA{0, 0, 0, 90}
		if c.btnHover {
			bg = color.RGBA{58, 46, 22, 220}
		}
		fillRoundRect(screen, float32(c.btnX), float32(c.btnY), float32(c.btnW), float32(c.btnH), 7, bg)
		g.text(screen, label, c.btnX+c.btnW/2, c.btnY+5, colKey, text.AlignCenter, 1)
	}
	g.text(screen, "E / B / Esc: đóng  ·  đi xa rương cũng tự đóng", px+pw/2, py+ph-24, colMuted, text.AlignCenter, 1)
	c.drawn = true
}

// --- art ---------------------------------------------------------------------

// genChest is a wooden chest bound with iron, its lid shut or thrown open.
func genChest(open bool) sprite {
	if envHD {
		return mkUp(chestHD(open, woodRamp[:3], trunkRamp, ironRamp, ironRamp, 27), 12*HDDensity, 20*HDDensity)
	}
	c := newCanvas(22, 22)
	const x0, x1, top, base = 2, 20, 9, 19
	// The box: planks lit from the left, two iron bands.
	for y := top; y <= base; y++ {
		for x := x0; x < x1; x++ {
			l := 0.8 - float64(x-x0)/float64(x1-x0)*0.5
			if (y-top)%4 == 3 {
				l -= 0.25 // a seam between planks
			}
			col := woodRamp.pick(l*0.8, x, y)
			if x == x0+3 || x == x1-4 {
				col = ironRamp.pick(l, x, y)
			}
			c.set(x, y, col)
		}
	}
	if open {
		// The inside, dark, and the lid tipped back above it.
		c.rect(x0+1, top-2, x1-x0-2, 3, solid(ashRamp[0]))
		c.polygon([]float64{x0, top - 2, x1, top - 2, x1 - 1, 1, x0 + 1, 1}, func(x, y int) color.RGBA {
			return trunkRamp.pick(0.75-float64(x-x0)/18*0.4, x, y)
		})
		c.line(x0+3, 1, x0+3, top-3, func(x, y int, _ float64) color.RGBA { return ironRamp[2] })
		c.line(x1-4, 1, x1-4, top-3, func(x, y int, _ float64) color.RGBA { return ironRamp[2] })
	} else {
		// The lid: a rounded top over the box, and the lock.
		for y := top - 4; y < top; y++ {
			inset := max(0, 2-(y-(top-4)))
			for x := x0 + inset; x < x1-inset; x++ {
				l := 0.95 - float64(x-x0)/float64(x1-x0)*0.5 - float64(top-y)*0.05
				col := trunkRamp.pick(l, x, y)
				if x == x0+3 || x == x1-4 {
					col = ironRamp.pick(l, x, y)
				}
				c.set(x, y, col)
			}
		}
		c.rect(10, top-1, 2, 3, solid(ironRamp[3]))
		c.set(10, top+1, ashRamp[0])
	}
	c.outline(0.8)
	return mk(c, 11, base)
}

// chestSprite is chest e, open while it is the one I have open.
func (g *Game) chestSprite(e protocol.EntityView) sprite {
	if g.overlay == overlayChest && g.curr.Self.Chest == e.ID {
		return g.art.chestOpen
	}
	return g.art.chest
}
