package client

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

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
	// Difficulty is the saved world's difficulty.
	Difficulty sim.Difficulty
}

const (
	btnContinue = iota
	btnNew
	btnSettings
	btnCredits
	btnQuit
)

// Menu is the title screen. Behind it a demo island drifts by.
type Menu struct {
	info     SaveInfo
	list     buttonList
	confirm  bool // asking before a new game overwrites the save
	yesNo    buttonList
	picking  bool // choosing the difficulty of the new world
	diffs    diffPicker
	diff     sim.Difficulty  // the difficulty chosen for the new world
	creating bool            // on the character screen, after the difficulty
	create   *charCreator    // kept, so going back and forth keeps the choices
	look     sim.Look        // the look made for the new game
	busy     string          // shown over everything while a world is being made
	settings *settingsScreen // the settings screen, while open; see settingsui.go
	credits  *creditsScreen  // the credits ("Ghi công"), while open; see creditsui.go
	status   string
	bg       *Game
}

// NewMenu builds the title screen. The backdrop world is generated here.
func NewMenu(info SaveInfo) *Menu {
	m := &Menu{
		info:   info,
		create: newCharCreator(),
		yesNo:  buttonList{items: []button{{label: "Bắt đầu mới"}, {label: "Quay lại"}}},
	}
	cont := button{label: "Chơi tiếp", disabled: !info.Exists || info.Broken}
	switch {
	case info.Broken:
		cont.sub = "Bản lưu bị hỏng, không đọc được"
	case info.Exists:
		cont.sub = fmt.Sprintf("Ngày %d · %s · lưu lúc %s", info.Day, info.Name, info.SavedAt.Local().Format("15:04 02/01/2006"))
		if info.Size > 0 {
			cont.sub = fmt.Sprintf("Ngày %d · %s · %s · lưu lúc %s", info.Day, info.Name, info.Difficulty.Name(), info.SavedAt.Local().Format("15:04 02/01"))
		}
	default:
		cont.sub = "Chưa có bản lưu"
	}
	m.list = buttonList{items: []button{
		cont,
		{label: "Trò chơi mới"},
		{label: "Cài đặt"},
		{label: "Ghi công"},
		{label: "Thoát"},
	}}
	if cont.disabled {
		m.list.sel = btnNew
	}
	m.bg = newDemo()
	return m
}

// difficulties are the levels the picker offers, in its order.
var difficulties = []sim.Difficulty{sim.DiffBasic, sim.DiffSkilled, sim.DiffLord}

// NewDifficulty is the difficulty picked for the new world, after MenuNew.
func (m *Menu) NewDifficulty() sim.Difficulty { return m.diff }

// NewSize is the size picked for the new world, after MenuNew, in tiles.
func (m *Menu) NewSize() int { return m.diffs.worldSize() }

// NewLook is the look made on the character screen, after MenuNew.
func (m *Menu) NewLook() sim.Look { return m.look }

// ShowCharacterScreen opens the character screen as picking a difficulty
// would. It is for dev tools that capture frames.
func (m *Menu) ShowCharacterScreen() { m.creating = true }

// ShowDifficultyPicker opens the difficulty choice as "Trò chơi mới" would.
// It is for dev tools that capture frames.
func (m *Menu) ShowDifficultyPicker() { m.picking = true }

// ShowSettings opens the settings screen at tab "audio", "display" or
// "controls", as "Cài đặt" would. It is for dev tools that capture frames.
func (m *Menu) ShowSettings(tab string) {
	m.settings = newSettingsScreen(false)
	m.settings.showTab(tab)
}

// ShowCredits opens the credits screen, scrolled down by scroll UI pixels, as
// "Ghi công" would. It is for dev tools that capture frames.
func (m *Menu) ShowCredits(scroll float64) {
	m.credits = newCreditsScreen()
	m.credits.scrollTo(scroll)
}

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
	fullscreenKeys() // F11, Alt+Enter: see display.go
	if m.settings != nil {
		if m.settings.update() {
			m.settings = nil
		}
		return MenuNone
	}
	if m.credits != nil {
		if m.credits.update() {
			m.credits = nil
		}
		return MenuNone
	}
	if m.creating {
		switch m.create.update() {
		case createDone:
			m.creating, m.look = false, m.create.look
			return MenuNew
		case createCancel:
			m.creating, m.picking = false, true // back to the difficulty
		}
		return MenuNone
	}
	if m.picking {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.picking = false
			return MenuNone
		}
		switch i := m.diffs.update(); {
		case i >= 0 && i < len(difficulties):
			m.picking, m.diff, m.creating = false, difficulties[i], true
			return MenuNone
		case i == len(difficulties):
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
	case btnSettings:
		m.settings = newSettingsScreen(false)
		return MenuNone
	case btnCredits:
		m.credits = newCreditsScreen()
		return MenuNone
	case btnQuit:
		return MenuQuit
	}
	return MenuNone
}

// Draw renders the title screen.
func (m *Menu) Draw(real *ebiten.Image) {
	if m.bg != nil {
		m.bg.Draw(real)
	}
	screen, sw, sh := beginUI(real) // see uiscale.go
	defer endUI(screen)
	if m.creating {
		m.create.draw(screen)
		defer m.create.drawPreview(real, uiScaleNow) // after the interface: at the screen's own resolution
		return
	}
	if m.settings != nil {
		m.settings.draw(screen)
		return
	}
	if m.credits != nil {
		m.credits.draw(screen)
		return
	}
	// Darken towards the middle column so the text reads over the island.
	fillRect(screen, 0, 0, float32(sw), float32(sh), color.RGBA{10, 8, 6, 90}, false)
	cx := float64(sw) / 2

	ty, listY := float64(sh)*0.2-30, float64(sh)*0.2+124
	if sh < 640 { // Full HD at UI scale 2: everything a little higher, so the list clears the hint
		ty, listY = ty-12, listY-20
	}
	drawTitle(screen, "Dino Isles", cx, ty, 6, color.RGBA{250, 214, 110, 255}, color.RGBA{60, 30, 10, 255})
	drawTitle(screen, "Survival", cx, ty+80, 2.6, color.RGBA{226, 128, 74, 255}, color.RGBA{60, 30, 10, 255})
	drawText(screen, "Một thế giới tiền sử đang chờ bạn", cx, ty+128, color.RGBA{236, 226, 200, 255}, text.AlignCenter, 1)

	m.list.draw(screen, cx, listY, 420)

	if m.status != "" {
		drawText(screen, m.status, cx, float64(sh)-80, color.RGBA{255, 140, 130, 255}, text.AlignCenter, 1)
	}
	drawText(screen, "W/S hoặc chuột để chọn · Enter để xác nhận · F11 toàn màn hình", cx, float64(sh)-24, colMuted, text.AlignCenter, 1)

	if m.confirm {
		dimScreen(screen, color.RGBA{0, 0, 0, 150})
		pw, ph := float32(480), float32(230)
		px, py := float32(cx)-pw/2, float32(sh)/2-ph/2
		drawFrame(screen, px, py, pw, ph)
		drawText(screen, "Bắt đầu trò chơi mới?", cx, float64(py)+20, colWhite, text.AlignCenter, 2)
		msg := "Bản lưu hiện tại sẽ bị ghi đè."
		if m.info.Day > 0 {
			msg = fmt.Sprintf("Bản lưu hiện tại (Ngày %d) sẽ bị ghi đè.", m.info.Day)
		}
		drawText(screen, msg, cx, float64(py)+58, colMuted, text.AlignCenter, 1)
		m.yesNo.draw(screen, cx, float64(py)+90, 300)
	}

	if m.picking {
		m.diffs.draw(screen)
	}

	if m.busy != "" {
		dimScreen(screen, color.RGBA{0, 0, 0, 170})
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

// drawTitle draws a title large, centred on cx, in the heaviest weight: a
// deep shadow of colour shadow under letters of colour fg, both rimmed dark.
func drawTitle(dst *uiDst, title string, cx, y, scale float64, fg, shadow color.RGBA) {
	uiTextW(dst, title, cx, y+scale*0.9, shadow, text.AlignCenter, scale, wTitle+50, fxOutline)
	uiTextW(dst, title, cx, y, fg, text.AlignCenter, scale, wTitle+50, fxOutline)
}
