package client

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// MenuChoice is what the player picked on the main menu.
type MenuChoice uint8

const (
	MenuNone MenuChoice = iota
	MenuContinue
	MenuNew
	MenuQuit
)

// SaveInfo describes the existing save, if any, for the main menu.
type SaveInfo struct {
	Exists  bool
	Broken  bool // the file exists but cannot be read
	Day     int
	Name    string
	SavedAt time.Time
	Size    int // the world's width in tiles, if known
}

const (
	btnContinue = iota
	btnNew
	btnMusic
	btnEffects
	btnQuit
)

// Menu is the title screen. Behind it a demo island drifts by.
type Menu struct {
	info     SaveInfo
	list     buttonList
	confirm  bool // asking before a new game overwrites the save
	yesNo    buttonList
	picking  bool // choosing the size of the new world
	sizes    buttonList
	size     int          // the size chosen for the new world
	creating bool         // on the character screen, after the size
	create   *charCreator // kept, so going back and forth keeps the choices
	look     sim.Look     // the look made for the new game
	busy     string       // shown over everything while a world is being made
	status   string
	bg       *Game
}

// NewMenu builds the title screen. The backdrop world is generated here.
func NewMenu(info SaveInfo) *Menu {
	m := &Menu{
		info:   info,
		create: newCharCreator(),
		yesNo:  buttonList{items: []button{{label: "Bắt đầu mới"}, {label: "Quay lại"}}},
		sizes: buttonList{sel: 1, items: []button{
			{label: "Vừa — 256 × 256", sub: "đi ngang hết khoảng 1 phút"},
			{label: "Lớn — 512 × 512", sub: "khoảng 2 phút · nên chọn"},
			{label: "Rất lớn — 1024 × 1024", sub: "khoảng 4 phút · tạo và lưu lâu hơn"},
			{label: "Quay lại"},
		}},
	}
	cont := button{label: "Chơi tiếp", disabled: !info.Exists || info.Broken}
	switch {
	case info.Broken:
		cont.sub = "Bản lưu bị hỏng, không đọc được"
	case info.Exists:
		cont.sub = fmt.Sprintf("Ngày %d · %s · lưu lúc %s", info.Day, info.Name, info.SavedAt.Local().Format("15:04 02/01/2006"))
		if info.Size > 0 {
			cont.sub = fmt.Sprintf("Ngày %d · %s · thế giới %d × %d · lưu lúc %s", info.Day, info.Name, info.Size, info.Size, info.SavedAt.Local().Format("15:04 02/01"))
		}
	default:
		cont.sub = "Chưa có bản lưu"
	}
	m.list = buttonList{items: []button{
		cont,
		{label: "Trò chơi mới"},
		{label: "Nhạc", volume: volMusic},
		{label: "Hiệu ứng", volume: volEffects},
		{label: "Thoát"},
	}}
	if cont.disabled {
		m.list.sel = btnNew
	}
	m.bg = newDemo()
	return m
}

// worldSizes are the sizes the picker offers, in its order.
var worldSizes = []int{sim.SizeMedium, sim.SizeLarge, sim.SizeHuge}

// NewWorldSize is the size picked for the new world, after MenuNew.
func (m *Menu) NewWorldSize() int { return m.size }

// NewLook is the look made on the character screen, after MenuNew.
func (m *Menu) NewLook() sim.Look { return m.look }

// ShowCharacterScreen opens the character screen as picking a size would.
// It is for dev tools that capture frames.
func (m *Menu) ShowCharacterScreen() { m.creating = true }

// ShowSizePicker opens the world-size choice as "Trò chơi mới" would. It is
// for dev tools that capture frames.
func (m *Menu) ShowSizePicker() { m.picking = true }

// SetBusy shows msg over the menu and ignores input until it is cleared
// with an empty msg; used while a world is being made or loaded.
func (m *Menu) SetBusy(msg string) { m.busy = msg }

// SetStatus shows a message under the buttons, such as why loading failed.
func (m *Menu) SetStatus(s string) { m.status = s }

// Layout implements ebiten.Game.
func (m *Menu) Layout(w, h int) (int, int) { return Layout(w, h) }

// Update handles input and returns the player's choice, if any.
func (m *Menu) Update() MenuChoice {
	if m.bg != nil {
		_ = m.bg.Update()
	}
	snd.silenceLoops()
	snd.setMusic("menu")
	snd.update()
	if m.busy != "" {
		return MenuNone
	}
	if m.creating {
		switch m.create.update() {
		case createDone:
			m.creating, m.look = false, m.create.look
			return MenuNew
		case createCancel:
			m.creating, m.picking = false, true // back to the size
		}
		return MenuNone
	}
	if m.picking {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.picking = false
			return MenuNone
		}
		switch i := m.sizes.update(); {
		case i >= 0 && i < len(worldSizes):
			m.picking, m.size, m.creating = false, worldSizes[i], true
			return MenuNone
		case i == len(worldSizes):
			m.picking = false
		}
		return MenuNone
	}
	if m.confirm {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.confirm = false
			return MenuNone
		}
		switch m.yesNo.update() {
		case 0:
			m.confirm, m.picking = false, true
			return MenuNone
		case 1:
			m.confirm = false
		}
		return MenuNone
	}
	switch m.list.update() {
	case btnContinue:
		return MenuContinue
	case btnNew:
		if m.info.Exists {
			m.confirm, m.yesNo.sel = true, 1
			return MenuNone
		}
		m.picking = true
		return MenuNone
	case btnQuit:
		return MenuQuit
	}
	return MenuNone
}

// Draw renders the title screen.
func (m *Menu) Draw(screen *ebiten.Image) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if m.bg != nil {
		m.bg.Draw(screen)
	}
	if m.creating {
		m.create.draw(screen)
		return
	}
	// Darken towards the middle column so the text reads over the island.
	vector.FillRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{6, 8, 16, 90}, false)
	cx := float64(sw) / 2

	ty := float64(sh)*0.2 - 30
	drawTitle(screen, "Dino Isles", cx, ty, 6, color.RGBA{250, 214, 110, 255}, color.RGBA{60, 30, 10, 255})
	drawText(screen, "Survival", cx+3, ty+81, color.RGBA{60, 30, 10, 255}, text.AlignCenter, 3)
	drawText(screen, "Survival", cx, ty+78, color.RGBA{214, 120, 72, 255}, text.AlignCenter, 3)
	drawText(screen, "Một thế giới tiền sử đang chờ bạn", cx, ty+128, color.RGBA{236, 226, 200, 255}, text.AlignCenter, 1)

	m.list.draw(screen, cx, float64(sh)*0.2+124, 420)

	if m.status != "" {
		drawText(screen, m.status, cx, float64(sh)-80, color.RGBA{255, 140, 130, 255}, text.AlignCenter, 1)
	}
	drawText(screen, "W/S hoặc chuột để chọn · Enter để xác nhận · A/D chỉnh âm lượng · F11 toàn màn hình", cx, float64(sh)-30, colMuted, text.AlignCenter, 1)

	if m.confirm {
		vector.FillRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, 150}, false)
		pw, ph := float32(480), float32(230)
		px, py := float32(cx)-pw/2, float32(sh)/2-ph/2
		fillRoundRect(screen, px-2, py-2, pw+4, ph+4, 12, color.RGBA{0, 0, 0, 120})
		fillRoundRect(screen, px, py, pw, ph, 10, color.RGBA{18, 20, 32, 245})
		drawText(screen, "Bắt đầu trò chơi mới?", cx, float64(py)+20, colWhite, text.AlignCenter, 2)
		msg := "Bản lưu hiện tại sẽ bị ghi đè."
		if m.info.Day > 0 {
			msg = fmt.Sprintf("Bản lưu hiện tại (Ngày %d) sẽ bị ghi đè.", m.info.Day)
		}
		drawText(screen, msg, cx, float64(py)+58, colMuted, text.AlignCenter, 1)
		m.yesNo.draw(screen, cx, float64(py)+90, 300)
	}

	if m.picking {
		vector.FillRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, 150}, false)
		pw, ph := float32(480), float32(340)
		px, py := float32(cx)-pw/2, float32(sh)/2-ph/2
		fillRoundRect(screen, px-2, py-2, pw+4, ph+4, 12, color.RGBA{0, 0, 0, 120})
		fillRoundRect(screen, px, py, pw, ph, 10, color.RGBA{18, 20, 32, 245})
		drawText(screen, "Thế giới rộng cỡ nào?", cx, float64(py)+18, colWhite, text.AlignCenter, 2)
		drawText(screen, "Đảo lớn, đảo nhỏ, hoang mạc, đầm lầy, núi có hang, núi lửa", cx, float64(py)+50, colMuted, text.AlignCenter, 1)
		m.sizes.draw(screen, cx, float64(py)+76, 400)
	}

	if m.busy != "" {
		vector.FillRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{0, 0, 0, 170}, false)
		dots := strings.Repeat(".", 1+int(time.Now().UnixMilli()/400)%3)
		drawText(screen, m.busy+dots, cx, float64(sh)/2-16, colWhite, text.AlignCenter, 2)
	}
}

// --- demo backdrop ----------------------------------------------------------

// demoConn feeds a Game a frozen world with no player in it.
type demoConn struct{ ch chan protocol.ServerMsg }

func (c *demoConn) Send(protocol.ClientMsg) error       { return nil }
func (c *demoConn) Messages() <-chan protocol.ServerMsg { return c.ch }
func (c *demoConn) Close() error                        { return nil }

func newDemo() *Game {
	const size = 96
	w, err := sim.Generate(sim.GenConfig{Seed: 20261123, Width: size, Height: size})
	if err != nil {
		return nil
	}
	snap := protocol.Snapshot{Tick: sim.DayTicks + sim.DayTicks*58/100} // golden dusk
	w.EachEntityIn(0, 0, size, size, func(e sim.Entity) {
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID: e.ID, Kind: e.Kind, TX: int16(e.TX), TY: int16(e.TY), Ready: e.Ready,
		})
	})
	sx, sy := demoSpot(w, size)
	if fx, fy, ok := freeGrass(w, int(sx), int(sy)-2); ok {
		snap.Entities = append(snap.Entities, protocol.EntityView{
			ID: 1 << 30, Kind: sim.EntityCampfire, TX: int16(fx), TY: int16(fy), Ready: true, Fuel: sim.FireStartFuel,
		})
	}
	// A few of the great beasts about the place, on dry land.
	beast := func(k sim.CreatureKind, dx, dy float64, st sim.CreatureState, left bool) {
		x, y := sx+dx, sy+dy
		if t := w.TileAt(int(x), int(y)); k != sim.CreaturePtero && (!t.Walkable() || t == sim.TileRocky) {
			return
		}
		snap.Creatures = append(snap.Creatures, protocol.CreatureView{
			ID: sim.CreatureID(1<<29 + len(snap.Creatures)), Kind: k, X: float32(x), Y: float32(y), State: st, FaceLeft: left,
		})
	}
	beast(sim.CreatureBrachio, -7, 1.5, sim.CreatureIdle, false)
	beast(sim.CreatureTrike, 5, 2.5, sim.CreatureIdle, true)
	beast(sim.CreatureTrike, 7.5, 3.6, sim.CreatureIdle, true)
	beast(sim.CreatureTrike, 6.2, 5.2, sim.CreatureIdle, false)
	beast(sim.CreaturePtero, 1, -3, sim.CreatureSoar, false)
	c := &demoConn{ch: make(chan protocol.ServerMsg, 2)}
	c.ch <- protocol.ServerMsg{Kind: protocol.ServerWelcome, Welcome: protocol.Welcome{Width: size, Height: size, Tiles: w.Tiles()}}
	c.ch <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: snap}
	g := New(c, "")
	g.demo = true
	g.demoX, g.demoY = sx, sy
	return g
}

// demoSpot finds a scenic place for the backdrop: a shore with sand, grass
// and forest, and not much bare rock.
func demoSpot(w *sim.World, size int) (float64, float64) {
	bx, by := w.Spawn()
	best := -1e9
	for cy := 12; cy < size-12; cy += 3 {
		for cx := 18; cx < size-18; cx += 3 {
			var c [sim.NumTiles]float64
			n := 0.0
			for ty := cy - 10; ty <= cy+10; ty += 2 {
				for tx := cx - 16; tx <= cx+16; tx += 2 {
					c[w.TileAt(tx, ty)]++
					n++
				}
			}
			for i := range c {
				c[i] /= n
			}
			score := -abs(c[sim.TileWater]-0.3) - 2*c[sim.TileRocky] +
				min(c[sim.TileForest], 0.25) + min(c[sim.TileSand], 0.15) + min(c[sim.TileGrass], 0.25)
			if score > best {
				best, bx, by = score, float64(cx)+0.5, float64(cy)+0.5
			}
		}
	}
	return bx, by
}

// freeGrass returns the nearest empty grass tile to (x, y).
func freeGrass(w *sim.World, x, y int) (int, int, bool) {
	for r := 0; r < 8; r++ {
		for ty := y - r; ty <= y+r; ty++ {
			for tx := x - r; tx <= x+r; tx++ {
				if w.TileAt(tx, ty) != sim.TileGrass {
					continue
				}
				if _, taken := w.EntityAt(tx, ty); !taken {
					return tx, ty, true
				}
			}
		}
	}
	return 0, 0, false
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// drawTitle draws the game's name large. The bitmap font squeezes the two
// accents of "ồ" into an unreadable blob at this size, so the grave accent
// is drawn by hand beside the circumflex, as Vietnamese type sets it.
func drawTitle(dst *ebiten.Image, title string, cx, y, scale float64, fg, shadow color.RGBA) {
	x := cx - textWidth(title, scale)/2
	for _, pass := range []struct {
		d float64
		c color.RGBA
	}{{scale * 0.7, shadow}, {0, fg}} {
		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x+pass.d, y+pass.d)
		op.ColorScale.ScaleWithColor(pass.c)
		text.Draw(dst, title, uiFace, op)
	}
}
