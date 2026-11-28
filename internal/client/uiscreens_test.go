package client

import (
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/sfnt"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The interface's screens, drawn at 720p, 1080p, 1440p and 4K through the game's
// own drawing (drawInterface, the menu's Draw): none may panic, and every
// string must lie inside the box drawn to hold it (or on the screen, for
// one drawn on its own), and not run over another in the same box.
//
// The game's art reads some of its images back as it is made, which
// Ebitengine allows only once its loop runs, so the screens are drawn
// inside a short game loop: a small window, off the screen and unfocused,
// for the few seconds the checks take. A loop runs once in a process and no
// image may be made after it, so it runs in a process of its own: the test
// starts this test binary again for it alone. Without a display the test is
// skipped.

// uiLoop runs f once inside the game loop.
type uiLoop struct {
	f    func()
	done bool
}

func (l *uiLoop) Update() error {
	if l.done {
		return ebiten.Termination
	}
	l.f()
	l.done = true
	return nil
}

func (l *uiLoop) Draw(*ebiten.Image)         {}
func (l *uiLoop) Layout(int, int) (int, int) { return 64, 64 }

// inGameLoop runs f inside Ebitengine's loop, or skips t without a display.
func inGameLoop(t *testing.T, f func()) {
	t.Helper()
	ebiten.SetWindowSize(64, 64)
	ebiten.SetWindowPosition(-20000, -20000)
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowTitle("interface test")
	l := &uiLoop{f: f}
	if err := ebiten.RunGameWithOptions(l, &ebiten.RunGameOptions{InitUnfocused: true}); err != nil && !l.done {
		t.Skipf("no game loop here: %v", err)
	}
	if !l.done {
		t.Skip("the game loop did not run")
	}
}

// TestInterfaceScreens draws every screen at every size; see above.
func TestInterfaceScreens(t *testing.T) {
	if os.Getenv(uiChildEnv) == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestInterfaceScreens$", "-test.v", "-test.count=1")
		cmd.Env = append(os.Environ(), uiChildEnv+"=1")
		out, err := cmd.CombinedOutput()
		if strings.Contains(string(out), "--- SKIP") {
			t.Skip(string(out))
		}
		if err != nil {
			t.Fatalf("the screens, in their own process: %v\n%s", err, out)
		}
		return
	}
	defer func() { uiProbe = nil }()
	inGameLoop(t, func() {
		interfaceScreensAtEverySize(t)
		menuScreensAtEverySize(t)
		worldLabelsStayInTheirPlates(t)
	})
}

var uiTestSizes = []struct {
	name string
	w, h int
}{{"720p", 1280, 720}, {"1080p", 1920, 1080}, {"1440p", 2560, 1440}, {"4K", 3840, 2160}} // 1440p: a fractional UI scale, 2.5

// uiTestGame is a game in the world, with things in its bag and on its
// hotbar, ready to draw its interface.
func uiTestGame() *Game {
	g := readyGame()
	g.art = getArt()
	g.conn = &sentConn{}
	me := protocol.PlayerView{ID: 1, Name: "Tiến", X: 10.5, Y: 10.5, Dir: sim.DirRight}
	other := protocol.PlayerView{ID: 2, Name: "Bạn", X: 13.5, Y: 11.5, Dir: sim.DirLeft}
	self := protocol.SelfState{Health: 18, Hunger: 12, Cold: true, Poisoned: true, Webbed: true, Chest: 0}
	inv := []sim.Slot{
		{Item: sim.ItemAxe, Count: 1}, {Item: sim.ItemWood, Count: 25}, {Item: sim.ItemStone, Count: 3},
		{Item: sim.ItemBerry, Count: 12}, {Item: sim.ItemStick, Count: 40}, {Item: sim.ItemTorch, Count: 2},
		{Item: sim.ItemSpear, Count: 1}, {}, {Item: sim.ItemMeat, Count: 3}, {Item: sim.ItemRope, Count: 4},
	}
	copy(self.Inv[:], inv)
	for i := 0; i < 12; i++ {
		self.Inv[sim.HotbarSlots+i] = sim.Slot{Item: sim.ItemKind(1 + i), Count: 1 + i*3}
	}
	self.TorchFuel = sim.TorchBurnTicks / 2
	g.onSnapshot(protocol.Snapshot{Tick: sim.DayTicks + 400, Self: self, Players: []protocol.PlayerView{me, other},
		Entities: []protocol.EntityView{{ID: 9, Kind: sim.EntityChest, TX: 11, TY: 10}, {ID: 10, Kind: sim.EntityTree, TX: 8, TY: 9, Ready: true}}})
	g.hotSel = 1
	g.statusAt[statusCold] = g.frame - 10 // just begun: its long line shows
	return g
}

// uiScreen sets a game up to show one screen.
type uiScreen struct {
	name string
	set  func(g *Game)
}

var uiScreens = []uiScreen{
	{"hud", func(g *Game) {}},
	{"hud-help-notice", func(g *Game) {
		g.showHelp = true
		g.notice, g.noticeUntil = "Thiếu nguyên liệu: cần 2 Đá lửa thô và 1 Dây thừng", time.Now().Add(time.Hour)
		g.autosaved = time.Now()
	}},
	{"hud-statuses", func(g *Game) { // every status at once, and every buff
		s := &g.curr.Self
		s.Stunned, s.Bleeding, s.Slowed, s.Pinned = true, true, true, true
		s.Cook.Buffs = [sim.NumBuffs]int32{52 * 20, 215 * 20, 96 * 20, 171 * 20, 438 * 20}
		g.curr.Players[0].Bees = true
		g.statusAt[statusBleed] = g.frame + 5 // the newest: its long line shows
	}},
	{"hud-notices-boss", func(g *Game) { // three notices under a boss that holds the player
		g.ShowNotices("Thiếu nguyên liệu: cần 2 Đá lửa thô và 1 Dây thừng",
			"Chưa buồn ngủ — giường chỉ dùng được lúc chạng vạng và ban đêm · cầm rìu trên tay để dỡ", "Đã nhặt 3 Gỗ")
		g.curr.Boss = protocol.BossView{Kind: sim.CreatureTitanoboa, HP: 6600, MaxHP: 16500, Enraged: true, Holds: true}
		g.autosaved = time.Now()
	}},
	{"hud-sea", func(g *Game) { // aboard a sailing boat in a squid's arms
		g.curr.Self.Boat = 5
		g.curr.Boats = []protocol.BoatView{{ID: 5, Kind: sim.BoatSail, X: 10.5, Y: 10.5, HP: 40, Rider: 1, Held: true}}
		g.curr.Wind, g.curr.WindDir = 0.7, 2
	}},
	{"hud-swim", func(g *Game) { g.curr.Players[0].Swimming = true; g.curr.Self.Breath = 0.4 }},
	{"hud-hint", func(g *Game) { // the held item's name gone: the hint
		g.hudFX.held = heldFX{slot: g.hotSel, item: g.curr.Self.Inv[g.hotSel].Item, at: time.Now().Add(-time.Minute)}
	}},
	{"bag", func(g *Game) { g.setOverlay(overlayBag); g.bag.setFocus(sim.ItemStick) }},
	{"bag-empty-focus", func(g *Game) { g.setOverlay(overlayBag); g.bag.cursor = sim.HotbarSlots + 20 }},
	{"chest", func(g *Game) {
		g.curr.Self.Chest = 9
		g.curr.Self.ChestSlots[1] = sim.Slot{Item: sim.ItemBerry, Count: 4}
		g.setOverlay(overlayChest)
		g.chest.hover = sim.NumSlots + 1
	}},
	{"cook", func(g *Game) {
		c := &g.curr.Self.Cook
		c.ID, c.Kind, c.State, c.Fuel = 9, sim.StationPot, sim.CookWorking, sim.FireStartFuel
		c.Slots[0] = sim.Slot{Item: sim.ItemMeat, Count: 2}
		c.Slots[4] = sim.Slot{Item: sim.ItemStew, Count: 2}
		r, _ := sim.CookRecipeByID(sim.CookStew)
		c.Dish, c.Total, c.Progress = sim.CookStew, int32(r.Ticks), int32(r.Ticks/2)
		g.overlay = overlayCook
	}},
	{"craft", func(g *Game) { g.setOverlay(overlayCraft) }},
	{"craft-free", func(g *Game) { g.curr.Self.DevBuild = true; g.setOverlay(overlayCraft) }},
	{"map", func(g *Game) { g.setOverlay(overlayMap) }},
	{"map-every-ground", func(g *Game) { // the longest legend: two rows of it
		for i := range g.wmap.present {
			g.wmap.present[i] = true
		}
		g.setOverlay(overlayMap)
	}},
	{"palette", func(g *Game) { g.curr.Self.DevBuild = true; g.overlay = overlayPalette }},
	{"dev", func(g *Game) {
		g.SetDev(true)
		g.curr.Self.DevBuild = true
		g.dev.open = true
		for i := range 14 {
			g.dev.say(fmt.Sprintf("> give gỗ %d — Đã cho %d Gỗ", i, i*5), colWhite)
		}
	}},
	{"boss", func(g *Game) {
		g.curr.Boss = protocol.BossView{Kind: sim.CreatureTitanoboa, HP: 6600, MaxHP: 16500, Enraged: true, Holds: true}
	}},
	{"pause", func(g *Game) { g.paused = true }},
	{"pause-settings", func(g *Game) { g.ShowSettings("display") }}, // see settingsui.go
	{"pause-settings-controls", func(g *Game) { g.ShowSettings("controls") }},
	{"dead", func(g *Game) { g.curr.Self.Dead = true }},
	{"victory", func(g *Game) {
		g.curr.Victory = protocol.VictoryView{Show: true, Day: 42, Beaten: []sim.CreatureKind{sim.CreatureTitanoboa, sim.CreatureMegalodon}}
	}},
}

// checkTexts fails t for every string outside its box, or over another.
func checkTexts(t *testing.T, where string, p *probe, w, h int) {
	t.Helper()
	const tol = 2.0 // image pixels of slack for anti-aliasing and rounding
	inside := func(a, b probeRect) bool {
		return a.x >= b.x-tol && a.y >= b.y-tol && a.x+a.w <= b.x+b.w+tol && a.y+a.h <= b.y+b.h+tol
	}
	boxOf := func(r probeRect) (probeRect, bool) {
		cx, cy := r.x+r.w/2, r.y+r.h/2
		best, ok := probeRect{}, false
		for _, b := range p.boxes {
			if cx >= b.x && cx <= b.x+b.w && cy >= b.y && cy <= b.y+b.h && (!ok || b.w*b.h < best.w*best.h) {
				best, ok = b, true
			}
		}
		return best, ok
	}
	screen := probeRect{x: 0, y: 0, w: float64(w), h: float64(h)}
	bad := 0
	for i, r := range p.texts {
		if strings.TrimSpace(r.s) == "" {
			continue
		}
		b, ok := boxOf(r)
		if !ok {
			b = screen
		}
		if !inside(r, b) && bad < 8 {
			bad++
			t.Errorf("%s: %q at (%.0f, %.0f) %.0fx%.0f runs out of its box (%.0f, %.0f) %.0fx%.0f",
				where, r.s, r.x, r.y, r.w, r.h, b.x, b.y, b.w, b.h)
		}
		// Not over another string in the same box: a name into its status.
		for _, o := range p.texts[i+1:] {
			if o.s == r.s || strings.TrimSpace(o.s) == "" {
				continue // a title drawn twice, for its shadow
			}
			ix := math.Min(r.x+r.w, o.x+o.w) - math.Max(r.x, o.x)
			// The band the letters fill, not the whole line with room for
			// accents and tails.
			band := func(p probeRect) (float64, float64) { return p.y + 0.2*p.h, p.y + 0.8*p.h }
			r0, r1 := band(r)
			o0, o1 := band(o)
			iy := math.Min(r1, o1) - math.Max(r0, o0)
			if ix > 2*tol && iy > 0.25*math.Min(r1-r0, o1-o0) && bad < 8 {
				if ob, ok := boxOf(o); (!ok && b == screen) || (ok && ob == b) {
					bad++
					t.Errorf("%s: %q runs over %q", where, r.s, o.s)
				}
			}
		}
	}
	if len(p.texts) == 0 {
		t.Errorf("%s: no text drawn", where)
	}
}

func interfaceScreensAtEverySize(t *testing.T) {
	for _, sz := range uiTestSizes {
		img := ebiten.NewImage(sz.w, sz.h)
		for _, sc := range uiScreens {
			g := uiTestGame()
			sc.set(g)
			for range 3 { // a few frames: what fades in, has come in
				g.frame++
				uiProbe = &probe{}
				img.Clear()
				g.drawInterface(img)
			}
			checkTexts(t, sz.name+" "+sc.name, uiProbe, sz.w, sz.h)
			if strings.HasPrefix(sc.name, "hud") || sc.name == "boss" {
				checkHUDClear(t, sz.name+" "+sc.name, uiProbe)
			}
		}
		img.Deallocate()
	}
}

// checkHUDClear fails t where two pieces of the HUD lie over each other:
// boxes that hold text, each not inside another, that cross.
func checkHUDClear(t *testing.T, where string, p *probe) {
	t.Helper()
	const tol = 1.0 // image pixels
	contains := func(a, b probeRect) bool {
		return b.x >= a.x-tol && b.y >= a.y-tol && b.x+b.w <= a.x+a.w+tol && b.y+b.h <= a.y+a.h+tol
	}
	var top []probeRect
	for i, b := range p.boxes {
		inside := false
		for j, a := range p.boxes {
			if i != j && contains(a, b) && !(contains(b, a) && j > i) {
				inside = true
				break
			}
		}
		if !inside {
			top = append(top, b)
		}
	}
	bad := 0
	for i, a := range top {
		for _, b := range top[i+1:] {
			ix := math.Min(a.x+a.w, b.x+b.w) - math.Max(a.x, b.x)
			iy := math.Min(a.y+a.h, b.y+b.h) - math.Max(a.y, b.y)
			if ix > tol && iy > tol && bad < 6 {
				bad++
				t.Errorf("%s: (%.0f, %.0f) %.0fx%.0f lies over (%.0f, %.0f) %.0fx%.0f", where, a.x, a.y, a.w, a.h, b.x, b.y, b.w, b.h)
			}
		}
	}
}

func menuScreensAtEverySize(t *testing.T) {
	screens := []struct {
		name string
		set  func(m *Menu)
	}{
		{"menu", func(m *Menu) {}},
		{"confirm", func(m *Menu) { m.confirm = true }},
		{"difficulty", func(m *Menu) { m.picking = true }},
		{"create", func(m *Menu) { m.creating = true; m.create.name = []rune("Tiến Ự") }},
		{"busy", func(m *Menu) { m.busy = "Đang tạo thế giới" }},
		{"settings-audio", func(m *Menu) { m.ShowSettings("audio") }},
		{"settings-display", func(m *Menu) { m.ShowSettings("display"); m.settings.line = 1 }}, // on the resolution: its "nét nhất"
		{"settings-controls", func(m *Menu) { m.ShowSettings("controls") }},
	}
	for _, sz := range uiTestSizes {
		img := ebiten.NewImage(sz.w, sz.h)
		for _, sc := range screens {
			m := NewMenu(SaveInfo{Exists: true, Day: 3, Name: "Tiến", SavedAt: time.Now(), Size: 1024})
			if m.create == nil {
				m.create = newCharCreator()
			}
			sc.set(m)
			uiProbe = &probe{}
			img.Clear()
			m.Draw(img)
			checkTexts(t, sz.name+" "+sc.name, uiProbe, sz.w, sz.h)
		}
		img.Deallocate()
	}
}

// The labels over the world: on the screen, in their plates.
func worldLabelsStayInTheirPlates(t *testing.T) {
	for _, sz := range uiTestSizes {
		img := ebiten.NewImage(sz.w, sz.h)
		uiScaleNow = uiScaleFor(sz.h)
		uiProbe = &probe{}
		drawFloatLabel(img, floatLabel{ok: true, text: "Bụi cỏ — cắt cỏ", col: colWhite, x: float64(sz.w) / 2, y: float64(sz.h) / 2}, 1)
		drawFloatLabel(img, floatLabel{ok: true, text: "xa quá – lại gần hơn", col: colFarText, x: 4, y: 4, hint: true}, 1)
		nameTag(img, "Người chơi Ự", float64(sz.w)-10, 20)
		drawCorners(img, 100, 100, 80, 60, colCorner, 3, 10, 1)
		checkTexts(t, sz.name+" labels", uiProbe, sz.w, sz.h)
		img.Deallocate()
	}
	uiScaleNow = 1
}

// Every Vietnamese letter is in the font, so nothing is drawn as a box or
// pieced together: "Ự" was drawn wrongly by the old bitmap font.
func TestTheFontHasEveryVietnameseLetter(t *testing.T) {
	f, err := sfnt.Parse(nunitoTTF)
	if err != nil {
		t.Fatal(err)
	}
	const letters = "aàáảãạăằắẳẵặâầấẩẫậbcdđeèéẻẽẹêềếểễệghiìíỉĩịklmnoòóỏõọôồốổỗộơờớởỡợpqrstuùúủũụưừứửữựvxyỳýỷỹỵ"
	var buf sfnt.Buffer
	for _, r := range letters + strings.ToUpper(letters) + "0123456789·—–…×%/:()!?“”" {
		if i, err := f.GlyphIndex(&buf, r); err != nil || i == 0 {
			t.Errorf("Nunito has no %q", r)
		}
	}
	// What it lacks (arrows) comes from the fallback, at a width.
	if w := textWidth("→↔", 1); w <= 4 {
		t.Errorf("arrows are %.1f wide: no fallback", w)
	}
}

// The text bakes: the same string at the same size is one bake, shown
// again without baking anew.
func TestTextIsBakedOnce(t *testing.T) {
	img := ebiten.NewImage(200, 60)
	d := &uiDst{img: img, k: 2}
	uiText(d, "Ngày 2", 4, 4, colWhite, 0, 1, fxShadow)
	n := len(textBakes)
	for range 5 {
		uiText(d, "Ngày 2", 4, 4, colKey, 0, 1, fxShadow)
	}
	if len(textBakes) != n {
		t.Errorf("drawing a string again baked it again: %d bakes, then %d", n, len(textBakes))
	}
	b := img.Bounds()
	if b != image.Rect(0, 0, 200, 60) {
		t.Fatal(b)
	}
}

// uiChildEnv is set in the process that draws the screens.
const uiChildEnv = "UI_SCREENS_CHILD"
