// Package client is the Ebitengine front end: it sends the player's input to
// a server and draws the snapshots it receives. It holds no game rules; the
// server is authoritative.
package client

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Conn is the client's connection to a server. The in-process
// server.LocalConn satisfies it today; a TCP connection will later.
type Conn interface {
	// Send queues a message without blocking; an error means it was not sent.
	Send(protocol.ClientMsg) error
	// Messages is closed when the connection is lost.
	Messages() <-chan protocol.ServerMsg
	Close() error
}

const (
	maxMsgsPerFrame = 256
	noticeDuration  = 2500 * time.Millisecond
	teleportTiles   = 2.0 // jumps larger than this (a respawn) are not interpolated
)

// Exit is what the player asked for from the pause menu.
type Exit uint8

const (
	ExitNone   Exit = iota
	ExitToMenu      // back to the main menu
	ExitQuit        // close the game
)

type overlay uint8

const (
	overlayNone overlay = iota
	overlayMap
	overlayBag
	overlayChest   // a chest the server has opened for me; see chestui.go
	overlayCraft   // the crafting window; see craftui.go
	overlayCook    // a cooking station the server has opened for me; see cookui.go
	overlayPalette // the build palette, in free build (dev mode); see buildpalette.go
)

// Game implements ebiten.Game.
type Game struct {
	conn Conn
	name string
	look sim.Look // the character made for a new game, sent in Hello

	helloSent    bool
	disconnected bool

	// Static map, from Welcome.
	ready         bool
	myID          sim.PlayerID
	width, height int
	tiles         []sim.Tile
	pondMask      []bool     // lakes and ponds, which freeze in winter
	surfaceH      int        // rows above ground; the caves lie below; 0 if none
	groundSeason  sim.Season // the season the ground is drawn for

	relief     []sim.Relief  // each tile's level and variant; nil for worlds made before sim.StyleIsles; see terrainfeatures.go
	seasonNext *seasonGround // the next season's ground, being worked out: see mapperf.go

	// The two latest snapshots; players are drawn interpolated between them.
	hasCurr       bool
	curr          protocol.Snapshot
	currAt        time.Time
	prevPlayers   map[sim.PlayerID]protocol.PlayerView
	swingAt       map[sim.PlayerID]int        // frame each player's latest blow began; see slash.go
	swingWhat     map[sim.PlayerID]swingInfo  // what that blow was at, and with what; see swing.go
	hurtAt        map[sim.PlayerID]int        // frame each player was last struck; see jolt.go
	joltBuf       *fineBuf                    // scratch for drawing a struck player
	gaits         map[sim.PlayerID]*gaitState // each player's stride; see limbdraw.go
	moves         map[sim.PlayerID]*moState   // what each player is doing besides; see motionpick.go
	drawCam       camera                      // the camera of the world being drawn: where the fall of motiondraw.go meets a body
	heldAge       int                         // see HoldBlows in punch.go
	prevCreatures map[sim.CreatureID]protocol.CreatureView
	hearts        []heartFX      // floating up from beasts being fed; see ride.go
	riders        []int          // scratch: players drawn on their mounts this frame
	entAt         map[[2]int]int // tile -> index into curr.Entities

	sentInput sim.Input

	// The held hotbar slot. The client shows its own choice at once and
	// keeps telling the server until it has sent it (sentSel); the first
	// snapshot after joining sets it to what the server remembers.
	hotSel, sentSel int
	selKnown        bool
	selChangedAt    time.Time // for the item name shown above the hotbar

	showHelp  bool              // F1: the full list of keys; see hud.go
	hudTarget targetView        // what Action works on, for the hint over the hotbar
	statusOn  [numStatuses]bool // the status effects last drawn
	statusAt  [numStatuses]int  // when each began, for its long line
	hudFX     hudFX             // the bars' trails, the labels' fades; see uitheme.go, uilabels.go
	wheel     float64           // mouse wheel travel not yet turned into a step

	notice      string
	noticeUntil time.Time
	noticeSeq   uint32
	autosaved   time.Time // when the last autosave finished, for the corner badge

	difficulty sim.Difficulty // the world's, from the welcome

	sm soundMemory

	paused  bool
	pause   buttonList
	onPause func(bool)
	exit    Exit

	// A full-screen view over the game: the map (M), the bag (Tab, I, B),
	// the crafting window (C) or a chest. The world keeps running while one
	// is open, but the player stands still.
	overlay overlay
	bag     bagView
	chest   chestView
	craft   craftView
	cook    cookView    // the cooking window; see cookui.go
	palette paletteView // the build palette, in free build; see buildpalette.go
	ptr     pointer     // the mouse in play; see mouse.go
	build   buildMode   // a structure chosen to build, waiting for a tile
	wmap    worldMap
	fires   map[sim.EntityID]knownFire

	// demo is the main menu's backdrop: no player, no HUD, a drifting camera.
	demo         bool
	demoX, demoY float64 // what the demo camera circles around

	art      *art
	ground   *groundCache
	world    *worldDst     // the world pass: art pixels on a full-resolution canvas; see hd.go
	worldHi  *ebiten.Image // the canvas when post-processing needs it apart from the screen
	darkness *ebiten.Image
	hd       *heroHDState  // the trial hero's frames and lights; see herohddraw.go
	caveMask *ebiten.Image // one pixel per tile in view: where it is always dark
	lavaMask *ebiten.Image // one pixel per tile in view: where lava glows
	canopy   *ebiten.Image // one pixel per tile in view: where the woods shade the ground
	drawList []drawable
	figures  []figure                 // who may be hidden behind a tree or a wall this frame; see occlude.go
	fades    map[sim.EntityID]float32 // how solid each fading entity is
	frame    int

	lastScreenH int // the UI canvas's height last drawn, for mouse hit-testing; see uiscale.go
	lastScreenW int
	mapZoom     int // the world map's zoom, 1 = the whole map
	zoomNoticed int // the zoom last shown in a notice
	strikesSeen map[[2]float32]bool
	prevBoats   map[sim.BoatID]protocol.BoatView
	swimBuf     *fineBuf    // scratch for drawing a swimmer cut off at the water
	boss        bossUI      // the boss bar and the victory screen; see bossui.go
	lair        lairFX      // the lairs' scenes and the moment a fight starts; see lairrender.go, bossintro.go
	dev         *devConsole // the dev console, only with -dev; see devconsole.go
}

// New returns a game that plays over conn under the given player name.
func New(conn Conn, name string) *Game {
	return &Game{
		conn:          conn,
		name:          name,
		zoomNoticed:   viewZoom,
		prevBoats:     make(map[sim.BoatID]protocol.BoatView),
		prevPlayers:   make(map[sim.PlayerID]protocol.PlayerView),
		prevCreatures: make(map[sim.CreatureID]protocol.CreatureView),
		entAt:         make(map[[2]int]int),
		pause: buttonList{items: []button{
			{label: "Tiếp tục"},
			{label: "Lưu game"},
			{label: "Nhạc", volume: volMusic},
			{label: "Hiệu ứng", volume: volEffects},
			{label: "Lưu và về menu chính"},
			{label: "Lưu và thoát"},
		}},
	}
}

// SetPauseHook registers f to be told when the pause menu opens and closes,
// so a single-player host can stop the clock.
func (g *Game) SetPauseHook(f func(paused bool)) { g.onPause = f }

// ShowOverlay opens "map", "bag" or "craft" as if its key were pressed, or
// "recipes": the bag with sticks clicked, which have the most recipes. It is
// for dev tools that capture frames; players use M, Tab and C.
func (g *Game) ShowOverlay(name string) {
	switch name {
	case "map":
		g.overlay = overlayMap
	case "bag":
		g.overlay = overlayBag
	case "craft":
		g.overlay = overlayCraft
	case "palette":
		g.overlay = overlayPalette // the build palette of free build; see buildpalette.go
		g.palette.hover = -1
	case "recipes":
		g.overlay = overlayBag
		g.bag.setFocus(sim.ItemStick)
	}
}

// Exit reports whether the player chose to leave from the pause menu.
func (g *Game) Exit() Exit { return g.exit }

// Layout uses the window size as the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return Layout(outsideWidth, outsideHeight)
}

// Update runs once per tick of the client loop (60 Hz).
func (g *Game) Update() error {
	g.frame++
	if g.art == nil {
		g.art = getArt() // images are created once the game loop is running
		g.ground = newGroundCache()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.showHelp = !g.showHelp // the full list of keys; see hud.go
	}
	if g.demo {
		g.receive()
		return nil
	}
	defer func() { g.frameSounds(g.alpha()) }()
	if g.disconnected {
		g.setOverlay(overlayNone)
	}
	if g.updateDev() { // the dev console has the keyboard; see devconsole.go
		g.receive()
		g.wmap.collect()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.build.on && g.overlay == overlayNone && !g.paused {
			g.build.on = false // Esc stops building first; see mouse.go
			g.showNotice("Đã thôi xây")
		} else if g.overlay != overlayNone && !g.paused {
			g.setOverlay(overlayNone) // Esc closes the map or bag first
		} else {
			g.setPaused(!g.paused)
		}
	}
	if g.disconnected {
		if g.paused {
			g.updatePause()
		}
		return nil
	}
	if !g.helloSent {
		hello := protocol.Hello{Version: protocol.Version, Name: g.name, Look: g.look}
		g.helloSent = g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: hello}) == nil
	}
	g.receive()
	g.wmap.collect()
	if g.paused {
		g.updatePause()
		return nil
	}
	if g.ready && g.hasCurr && !g.disconnected {
		g.lairUpdate() // see lairrender.go
	}
	if g.ready && g.hasCurr && !g.disconnected && !g.updateVictory() { // see bossui.go
		g.handleInput()
	}
	return nil
}

func (g *Game) setPaused(p bool) {
	if p == g.paused {
		return
	}
	g.paused = p
	g.pause.sel = 0
	snd.ui("ui_pause")
	if p && g.sentInput != (sim.Input{}) {
		// Let go of every key, so the player does not keep walking.
		if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput}) == nil {
			g.sentInput = sim.Input{}
		}
	}
	if g.onPause != nil {
		g.onPause(p)
	}
}

func (g *Game) updatePause() {
	switch g.pause.update() {
	case 0:
		g.setPaused(false)
	case 1:
		if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientSave}) != nil {
			g.showNotice("Không gửi được yêu cầu lưu")
		}
	case 4:
		g.exit = ExitToMenu
	case 5:
		g.exit = ExitQuit
	}
}

func (g *Game) setOverlay(o overlay) {
	if o == g.overlay {
		return
	}
	if o == overlayNone {
		snd.ui("ui_move")
	} else {
		snd.ui("ui_select")
	}
	if o == overlayBag {
		g.bag.open(g.hotSel, g.curr.Self.Inv[g.hotSel].Item)
	}
	if g.overlay == overlayChest {
		g.closeChest()
	}
	if g.overlay == overlayCook {
		g.closeCook()
	}
	g.overlay = o
}

// toggleOverlay opens o, or closes it if it is already open.
func (g *Game) toggleOverlay(o overlay) {
	if g.overlay == o {
		o = overlayNone
	}
	g.setOverlay(o)
}

func (g *Game) showNotice(s string) {
	g.notice, g.noticeUntil = s, time.Now().Add(noticeDuration)
}

func (g *Game) receive() {
	for range maxMsgsPerFrame {
		select {
		case m, ok := <-g.conn.Messages():
			if !ok {
				g.disconnected = true
				return
			}
			g.onMessage(m)
		default:
			return
		}
	}
}

func (g *Game) onMessage(m protocol.ServerMsg) {
	switch m.Kind {
	case protocol.ServerWelcome:
		w := m.Welcome
		if w.Width <= 0 || w.Height <= 0 || len(w.Tiles) != w.Width*w.Height {
			g.disconnected = true // malformed map; do not index into it
			return
		}
		g.myID, g.width, g.height, g.tiles = w.PlayerID, w.Width, w.Height, w.Tiles
		g.pondMask = sim.PondMask(w.Width, w.Height, w.Tiles)
		g.surfaceH = w.SurfaceH
		g.difficulty = w.Difficulty
		g.relief = w.Relief
		terr := newTerrain(w.Width, w.Height, w.Tiles)
		terr.setRelief(g.relief) // see terrainfeatures.go
		terr.under = g.caveRow()
		g.ground.reset(terr)
		g.groundSeason = sim.Spring
		if !g.demo {
			g.wmap.reset(terr, sim.ExploredFrom(w.Width, w.Height, w.Explored))
		}
		clear(g.fires)
		g.ready = true
	case protocol.ServerSaved:
		switch r := m.Saved; {
		case !r.OK:
			g.showNotice("Lưu thất bại: " + r.Error)
		case r.Auto:
			g.autosaved = time.Now()
		default:
			g.showNotice("Đã lưu game")
			snd.play("save", 0.5, 0, 1)
		}
	case protocol.ServerSnapshot:
		if !g.ready || (g.hasCurr && m.Snapshot.Tick <= g.curr.Tick) {
			return
		}
		g.onSnapshot(m.Snapshot)
	case protocol.ServerDev:
		g.onDev(m.Dev) // see devconsole.go
	}
}

func (g *Game) onSnapshot(s protocol.Snapshot) {
	old, hadOld := g.curr, g.hasCurr
	clear(g.prevPlayers)
	for _, p := range g.curr.Players {
		g.prevPlayers[p.ID] = p
	}
	clear(g.prevCreatures)
	for _, c := range g.curr.Creatures {
		g.prevCreatures[c.ID] = c
	}
	clear(g.prevBoats)
	for _, b := range g.curr.Boats {
		g.prevBoats[b.ID] = b
	}
	g.noteHits(s) // see combatui.go: before prevCreatures is lost to the next snapshot
	g.curr, g.currAt, g.hasCurr = s, time.Now(), true
	if want := groundSeason(s.Season); want != g.groundSeason {
		// Snow on the ground and ice on the ponds come and go with winter.
		g.groundSeason = want
		if g.width*g.height > bigGround {
			g.startSeasonGround(want) // off the game loop: see mapperf.go
		} else {
			terr := newTerrain(g.width, g.height, seasonTiles(g.width, g.height, g.tiles, want, g.pondMask))
			terr.setRelief(g.relief)
			terr.under = g.caveRow()
			g.ground.reset(terr)
		}
	}
	g.takeSeasonGround()
	g.noteSwings(s)
	g.classifySwings(old, s)
	g.noteHurts(s)
	g.noteMotions(old, s)
	if !g.selKnown && sim.IsHotbar(int(s.Self.Selected)) {
		g.hotSel, g.sentSel, g.selKnown = int(s.Self.Selected), int(s.Self.Selected), true
	}

	clear(g.entAt)
	for i, e := range s.Entities {
		g.entAt[[2]int{int(e.TX), int(e.TY)}] = i
	}
	g.syncChest()
	g.syncCook()
	if hadOld {
		g.snapshotSounds(old, s)
		g.noteBosses(old, s) // see bossui.go
	}
	g.mapSnapshot(s)

	if s.Self.NoticeSeq != g.noticeSeq {
		g.noticeSeq = s.Self.NoticeSeq
		if txt := noticeText(s.Self.Notice, s.Self.NoticeArg); txt != "" {
			g.notice, g.noticeUntil = txt, time.Now().Add(noticeDuration)
		}
	}
}

func (g *Game) handleInput() {
	g.notePointer()
	g.keepBuild()
	dead := g.curr.Self.Dead
	switch {
	case dead:
		g.setOverlay(overlayNone) // the death screen needs to be seen
	case inpututil.IsKeyJustPressed(ebiten.KeyM):
		g.toggleOverlay(overlayMap)
	case g.freeBuild() && inpututil.IsKeyJustPressed(ebiten.KeyB): // B is the build palette in free build; see buildpalette.go
		if g.overlay == overlayPalette {
			g.setOverlay(overlayNone)
		} else {
			g.openPalette()
		}
	case justPressed(ebiten.KeyTab, ebiten.KeyI, ebiten.KeyB):
		g.toggleOverlay(overlayBag)
	case inpututil.IsKeyJustPressed(ebiten.KeyC):
		g.toggleOverlay(overlayCraft)
	}

	// With the map or bag open the player stands still; the keys belong to it.
	var in sim.Input
	if g.overlay == overlayNone {
		if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			in.MoveY--
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			in.MoveY++
		}
		if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			in.MoveX--
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			in.MoveX++
		}
		in.Action = g.leftButton() || ebiten.IsKeyPressed(ebiten.KeySpace) // see mouse.go
		if g.build.on {
			in.Action = false // building: the left button places, it does not strike
		}
		if ax, ay, ok := g.aim(); ok {
			in.Aim, in.AimX, in.AimY = true, quantAim(ax), quantAim(ay)
		}
	}

	if g.introLocked() {
		in.MoveX, in.MoveY, in.Action = 0, 0, false // a boss's entrance: see bossintro.go
	}

	// Held input is sent only when it changes; if the send fails it is
	// retried next frame because sentInput is left untouched.
	if in != g.sentInput {
		if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: in}) == nil {
			g.sentInput = in
		}
	}

	switch g.overlay {
	case overlayBag:
		g.updateBag() // in the bag the number keys move stacks instead
	case overlayChest:
		g.updateChest()
	case overlayCook:
		g.updateCook()
	case overlayCraft:
		g.updateCraft() // see craftui.go
	case overlayPalette:
		g.updatePalette() // see buildpalette.go
	default:
		g.updateHotbar()
	}
	if g.hotSel != g.sentSel {
		c := sim.Command{Kind: sim.CmdSelect, Slot: uint8(g.hotSel)}
		if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientCommand, Command: c}) == nil {
			g.sentSel = g.hotSel
		}
	}
	g.updateKeys(dead)
	g.updateZoom()
	g.updateToss() // G throws things on the ground; see toss.go
}

// updateKeys handles the keys that do one thing per press: E, R, V and the
// right button.
func (g *Game) updateKeys(dead bool) {
	switch {
	case dead:
		// The death screen: R, E or Enter comes back.
		if justPressed(ebiten.KeyR, ebiten.KeyE, ebiten.KeyEnter, ebiten.KeyNumpadEnter) {
			g.command(sim.Command{Kind: sim.CmdRespawn})
		}
		return
	case g.overlay == overlayBag:
		if inpututil.IsKeyJustPressed(ebiten.KeyE) { // in the bag, E eats what is pointed at
			if slot, ok := g.slotToEat(); ok {
				g.command(sim.Command{Kind: sim.CmdEat, Slot: uint8(slot)})
			} else {
				g.showNotice("Món đang trỏ không ăn được")
			}
		}
		return
	case g.overlay == overlayChest || g.overlay == overlayCook:
		if inpututil.IsKeyJustPressed(ebiten.KeyE) {
			g.setOverlay(overlayNone) // E closes the chest it opened
		}
		return
	case g.overlay != overlayNone:
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.command(sim.Command{Kind: sim.CmdInteract}) // pick up, open, get on or off; see sim/aim.go
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.command(sim.Command{Kind: sim.CmdRide}) // as E for beasts and boats; see ride.go
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.rightClick() // use what is held; see mouse.go
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) { // the keyboard's way to set down what is held
		ctx := g.useContextNow()
		c := sim.Command{Kind: sim.CmdPlace}
		if ctx.spot.ok && ctx.spot.valid {
			c.At, c.TX, c.TY = true, int16(ctx.spot.tx), int16(ctx.spot.ty)
		}
		g.command(c)
	}
}

// hotbarKeys are the keys for hotbar slots 1 to 10.
var hotbarKeys = [sim.HotbarSlots]ebiten.Key{
	ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
	ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9, ebiten.KeyDigit0,
}

// hotbarKey returns the hotbar slot whose key was just pressed.
func hotbarKey() (int, bool) {
	for i, k := range hotbarKeys {
		if inpututil.IsKeyJustPressed(k) {
			return i, true
		}
	}
	return 0, false
}

// updateHotbar picks the held slot with the number keys or the mouse wheel.
func (g *Game) updateHotbar() {
	sel := g.hotSel
	if i, ok := hotbarKey(); ok {
		sel = i
	}
	if g.overlay == overlayNone && !ctrlHeld() { // Ctrl and the wheel zoom instead
		// A wheel notch is 1; touchpads send many small amounts, so they add
		// up to a step.
		_, dy := ebiten.Wheel()
		g.wheel += dy
		for ; g.wheel >= 1; g.wheel-- {
			sel = (sel + sim.HotbarSlots - 1) % sim.HotbarSlots // up goes left, as in most games
		}
		for ; g.wheel <= -1; g.wheel++ {
			sel = (sel + 1) % sim.HotbarSlots
		}
	}
	g.selectSlot(sel)
}

func (g *Game) selectSlot(i int) {
	if i == g.hotSel || !sim.IsHotbar(i) {
		return
	}
	g.hotSel, g.selChangedAt = i, time.Now()
	snd.ui("ui_move")
}

// slotToEat is the slot E eats from: the one pointed at with the bag open,
// else the held one; ok is false when that is not food.
func (g *Game) slotToEat() (int, bool) {
	inv := &g.curr.Self.Inv
	edible := func(i int) bool { return sim.ValidSlot(i) && !inv[i].Empty() && inv[i].Item.Edible() }
	if g.overlay == overlayBag && edible(g.bag.cursor) {
		return g.bag.cursor, true
	}
	if g.overlay != overlayBag && edible(g.hotSel) {
		return g.hotSel, true
	}
	// Only what is chosen is eaten: nothing else from the bag behind your back.
	return 0, false
}

// haveTool reports whether r makes a tool the player already carries, so
// making another would be refused.
func (g *Game) haveTool(r sim.Recipe) bool {
	return r.Gives.Valid() && r.Gives.Def().Tool && g.curr.Self.Inv.Count(r.Gives) > 0
}

func (g *Game) command(c sim.Command) {
	// A dropped command only costs the player a second key press.
	_ = g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientCommand, Command: c})
}

// alpha is how far the view is between the previous and current snapshot.
func (g *Game) alpha() float64 {
	return max(0, min(1, float64(time.Since(g.currAt))/float64(sim.TickDuration)))
}

// lerpPlayer returns where to draw p and whether it is moving.
func (g *Game) lerpPlayer(p protocol.PlayerView, alpha float64) (x, y float64, moving bool) {
	x, y = float64(p.X), float64(p.Y)
	q, ok := g.prevPlayers[p.ID]
	if !ok {
		return x, y, false
	}
	px, py := float64(q.X), float64(q.Y)
	if px == x && py == y {
		return x, y, false
	}
	if math.Abs(px-x) > teleportTiles || math.Abs(py-y) > teleportTiles {
		return x, y, false
	}
	return px + (x-px)*alpha, py + (y-py)*alpha, true
}

func (g *Game) tileAt(tx, ty int) sim.Tile {
	if tx < 0 || ty < 0 || tx >= g.width || ty >= g.height {
		return sim.TileWater
	}
	if t := g.tiles[ty*g.width+tx]; t < sim.NumTiles {
		return t
	}
	return sim.TileWater // unknown tile from a newer server: never index art with it
}

func noticeText(kind sim.NoticeKind, arg uint8) string {
	switch kind {
	case sim.NoticeNotEnough:
		if r, ok := sim.RecipeByID(sim.RecipeID(arg)); ok {
			return "Thiếu nguyên liệu — cần " + ingredientsText(r)
		}
	case sim.NoticeNoSpace:
		return "Không có chỗ trống để đặt"
	case sim.NoticeNotEdible:
		return "Thứ này không ăn được"
	case sim.NoticeNoneLeft:
		return "Hết " + sim.ItemKind(arg).Def().Name
	case sim.NoticeBagFull:
		return "Túi đầy: " + sim.ItemKind(arg).Def().Name
	case sim.NoticeCrafted:
		if r, ok := sim.RecipeByID(sim.RecipeID(arg)); ok {
			switch {
			case r.NearFire:
				return "Đã nướng xong 1 " + r.Gives.Def().Name
			case r.Gives.Valid():
				return "Đã làm xong " + r.Name
			}
			return "Đã dựng " + r.Name
		}
	case sim.NoticeNeedTool:
		k := sim.ItemKind(arg)
		switch class, tier := k.ToolKind(); {
		case k == sim.ItemAxe:
			return "Cần Rìu đá — nhặt cành cây và đá cuội rồi làm ở bảng chế tạo (C)"
		case k == sim.ItemPickaxe:
			return "Cần Cúp đá — nhặt cành cây và đá cuội, làm ở bảng chế tạo (C)"
		case k == sim.ItemSaddle:
			return "Cần Yên da trong túi để cưỡi — làm từ 4 Da thú, 2 Cỏ khô, 1 Xương (C)"
		case class == sim.ToolPickaxe:
			// Ores: the pickaxe one tier below, or better.
			return "Cần " + k.Def().Name + " trở lên — làm từ " + strings.ToLower(sim.TierMaterial(tier).Def().Name) + " ở bảng chế tạo (C)"
		case k.Def().Tool && class != sim.ToolAxe:
			return "Cầm " + k.Def().Name + " trên tay (chọn ở ô nhanh 1-0)"
		}
		return "Cần " + k.Def().Name
	case sim.NoticeNotTired:
		return "Chưa buồn ngủ — giường chỉ dùng được lúc chạng vạng và ban đêm"
	case sim.NoticeTooEarly:
		return "Giật sớm quá — cá sợ bỏ đi (đợi phao chìm rồi mới nhấp chuột)"
	case sim.NoticeBaitStolen:
		return "Chậm tay rồi — cá ăn mất " + strings.ToLower(sim.ItemKind(arg).Def().Name)
	case sim.NoticeCaught:
		return "Câu được 1 " + sim.ItemKind(arg).Def().Name + "!"
	case sim.NoticeFoodPoison:
		return sim.ItemKind(arg).Def().Name + " có độc — bạn trúng độc! (nhai Thảo dược để giải)"
	case sim.NoticeCured:
		return sim.ItemKind(arg).Def().Name + " đã giải độc"
	case sim.NoticeArmoured:
		return "Giáp xương đỡ mất nửa đòn — vòng lên đánh vào đầu " + strings.ToLower(sim.CreatureKind(arg).Name())
	case sim.NoticeNeedSeed:
		return "Cần giống để trồng: khoai lang, thóc, ngô hoặc hạt bí"
	case sim.NoticeChestFull:
		return "Rương đầy rồi — không cất thêm " + strings.ToLower(sim.ItemKind(arg).Def().Name) + " được"
	case sim.NoticeChestNotEmpty:
		return "Rương còn đồ — lấy hết ra rồi mới dỡ được"
	case sim.NoticeFed:
		return "Cho " + strings.ToLower(sim.CreatureKind(arg).Name()) + " ăn — nó bớt sợ bạn rồi"
	case sim.NoticeTamed:
		return "Đã thuần hóa " + strings.ToLower(sim.CreatureKind(arg).Name()) + "! Nó sẽ đi theo bạn"
	case sim.NoticeTooWild:
		return sim.CreatureKind(arg).Name() + " chưa chịu ăn — đánh cho nó gục đã rồi cho ăn"
	case sim.NoticeAngry:
		return sim.CreatureKind(arg).Name() + " đang nổi giận, không chịu ăn"
	case sim.NoticeNotYours:
		return "Đây là thú của người khác"
	case sim.NoticePetStay:
		return sim.CreatureKind(arg).Name() + " sẽ đứng chờ ở đây"
	case sim.NoticePetCome:
		return sim.CreatureKind(arg).Name() + " sẽ đi theo bạn"
	case sim.NoticeNoMount:
		return "Không có thú nào của bạn ở gần để cưỡi"
	case sim.NoticeCantRide:
		return sim.CreatureKind(arg).Name() + " không cưỡi được"
	case sim.NoticeHealed:
		return sim.CreatureKind(arg).Name() + " ăn xong, khỏe hơn rồi"
	case sim.NoticeSkittish:
		return sim.CreatureKind(arg).Name() + " chạy mất rồi — chờ nó dừng lại thở dốc mới cho ăn được"
	case sim.NoticeNeedFurnace:
		if r, ok := sim.RecipeByID(sim.RecipeID(arg)); ok {
			return "Đứng cạnh Lò nung đang cháy để làm " + r.Name + " (thêm than hoặc củi vào lò)"
		}
		return "Cần đứng cạnh Lò nung đang cháy"
	case sim.NoticeHaveOne:
		return "Bạn đã có " + sim.ItemKind(arg).Def().Name
	case sim.NoticeNeedBait:
		return "Cần " + sim.ItemKind(arg).Def().Name + " làm mồi câu — bắt giun trên bãi cỏ"
	case sim.NoticeBitten:
		verb := " cắn!"
		switch sim.CreatureKind(arg) {
		case sim.CreatureCrab:
			verb = " kẹp!"
		case sim.CreatureTrike, sim.CreatureToro, sim.CreatureMammoth:
			verb = " húc!"
		case sim.CreatureAnzu:
			verb = " đá!"
		case sim.CreatureQuetzal:
			verb = " mổ!"
		case sim.CreatureStego:
			verb = " quất đuôi!"
		case sim.CreatureAnkylo:
			verb = " quật chùy đuôi, văng ra xa!"
		case sim.CreaturePtero:
			verb = " mổ!"
		case sim.CreatureSpider:
			verb = " cắn — trúng độc!"
		case sim.CreatureBear:
			verb = " vồ!"
		case sim.CreatureTribesman:
			verb = " đâm giáo!"
		case sim.CreatureMosa:
			verb = " đớp!"
		case sim.CreatureKraken:
			verb = " siết!"
		case sim.CreatureScorpion:
			verb = " kẹp và chích!"
		case sim.CreatureToad:
			return "Chạm phải cóc độc — trúng độc! (nhai Thảo dược để giải)"
		}
		return "Bị " + strings.ToLower(sim.CreatureKind(arg).Name()) + verb
	case sim.NoticeNeedFire:
		if r, ok := sim.RecipeByID(sim.RecipeID(arg)); ok && r.Gives.Healing() {
			return "Phải đứng gần lửa trại đang cháy để nấu " + r.Name
		}
		return "Phải đứng gần lửa trại đang cháy để nướng"
	case sim.NoticePickedUp, sim.NoticeTossed:
		return "" // the pick-up or thump sound says it
	case sim.NoticeCantToss:
		return "Đồ vứt xuống nước sẽ mất — quay mặt vào bờ rồi hẵng vứt"
	case sim.NoticeCantPlace:
		return "Cầm đầu thú, đuốc, hạt giống hoặc phân trên tay rồi nhấp chuột phải"
	case sim.NoticePlantBite:
		return "Bị cây ăn thịt người đớp!"
	case sim.NoticeLightning:
		return "Bị sét đánh!"
	case sim.NoticeTornado:
		return "Bị lốc xoáy cuốn! Chạy ra xa"
	case sim.NoticeWebbed:
		return "Dính tơ nhện — đi chậm lại!"
	case sim.NoticeTreasure:
		return "Mở được rương cổ!"
	case sim.NoticeFertilized:
		if sim.EntityKind(arg) == sim.EntityFarmland {
			return "Đã bón phân cho luống — khoai trồng ở đây lớn nhanh gấp đôi"
		}
		return "Đã bón phân — cây lớn nhanh gấp đôi"
	case sim.NoticeBees:
		return "Ong đốt! Chạy thật xa hoặc nhảy xuống nước"
	case sim.NoticeBeesCalm:
		return "Khói đuốc làm ong hiền lại"
	case sim.NoticeBodyLeft:
		return "Đồ của bạn nằm lại ở xác chỗ bạn chết — quay lại lấy"
	case sim.NoticeBodyLooted:
		return "Đã lấy lại hết đồ"
	case sim.NoticeNothingToFeed:
		return "Trỏ vào cây non, luống đất hoặc cây đang mọc lại ở gần rồi nhấp chuột phải để bón phân"
	case sim.NoticeHomeSet:
		return "Đây là nơi bạn sẽ hồi sinh"
	case sim.NoticeLaunched:
		return "Đã thả " + sim.BoatKind(arg).Name() + " xuống nước — E để lên thuyền"
	case sim.NoticeNeedWater:
		return "Đứng sát mép nước, trỏ chuột ra nước rồi nhấp chuột phải để thả thuyền"
	case sim.NoticeNoShore:
		return "Không có bờ gần đây để lên — chèo vào sát bờ"
	case sim.NoticeSunk:
		return sim.BoatKind(arg).Name() + " bị phá chìm! Bơi vào bờ ngay"
	case sim.NoticeDrowning:
		return "Kiệt sức, đang chết đuối! Bơi vào bờ"
	case sim.NoticeTooFar:
		return buildRefusal(sim.NoticeTooFar)
	case sim.NoticeNothingNear:
		return "Không có gì ở gần để nhặt, mở hay cưỡi"
	case sim.NoticeIntoCave:
		return "Vào hang — đi lên dốc chỗ có ánh nắng để ra lại"
	case sim.NoticeOutOfCave:
		return "Đã ra khỏi hang"
	}
	return cookNoticeText(kind, arg) // see cookrender.go
}

// myHeld is the item kind in the hotbar slot this player holds.
func (g *Game) myHeld() sim.ItemKind {
	if s := g.curr.Self.Inv[g.curr.Self.Selected]; !s.Empty() {
		return s.Item
	}
	return sim.ItemNone
}

// groundSeason is the season whose ground is drawn: only winter looks different.
func groundSeason(s sim.Season) sim.Season {
	if s == sim.Winter {
		return sim.Winter
	}
	return sim.Spring
}

func ctrlHeld() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
}

// updateZoom changes the view's zoom with = and - (or Ctrl and the wheel),
// and on the map, the map's zoom with the wheel.
func (g *Game) updateZoom() {
	in := inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd)
	out := inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract)
	if g.overlay == overlayMap {
		_, dy := ebiten.Wheel()
		switch {
		case in || dy > 0:
			g.mapZoom = min(maxMapZoom, g.mapZoom+1)
		case out || dy < 0:
			g.mapZoom = max(1, g.mapZoom-1)
		}
		return
	}
	if g.overlay != overlayNone {
		return
	}
	if ctrlHeld() {
		_, dy := ebiten.Wheel()
		in, out = in || dy > 0, out || dy < 0
	}
	switch {
	case in:
		setZoom(viewZoom + 1)
	case out:
		setZoom(viewZoom - 1)
	}
	if g.zoomNoticed != viewZoom {
		g.zoomNoticed = viewZoom
		g.showNotice(fmt.Sprintf("Thu phóng: %+d", viewZoom))
	}
}

// setZoom sets and saves the view's zoom.
func setZoom(z int) {
	viewZoom = max(minZoom, min(maxZoom, z))
	snd.saveZoom(viewZoom)
}

// SetZoom sets the view's zoom, minZoom to maxZoom, without saving it: for
// the screenshot tool.
func SetZoom(z int) { viewZoom = max(minZoom, min(maxZoom, z)) }

// caveRow is the first map row of the caves under the world, 0 if none.
func (g *Game) caveRow() int {
	if g.surfaceH <= 0 {
		return 0
	}
	return g.surfaceH
}

// underground reports whether this player is down in the caves.
func (g *Game) underground() bool {
	if g.surfaceH <= 0 {
		return false
	}
	for _, p := range g.curr.Players {
		if p.ID == g.myID {
			return int(p.Y) >= g.surfaceH
		}
	}
	return false
}
