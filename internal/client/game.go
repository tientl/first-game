// Package client is the Ebitengine front end: it sends the player's input to
// a server and draws the snapshots it receives. It holds no game rules; the
// server is authoritative.
package client

import (
	"math"
	"time"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

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

// Game implements ebiten.Game.
type Game struct {
	conn Conn
	name string

	helloSent    bool
	disconnected bool

	// Static map, from Welcome.
	ready         bool
	myID          sim.PlayerID
	width, height int
	tiles         []sim.Tile

	// The two latest snapshots; players are drawn interpolated between them.
	hasCurr     bool
	curr        protocol.Snapshot
	currAt      time.Time
	prevPlayers map[sim.PlayerID]protocol.PlayerView
	entAt       map[[2]int]int // tile -> index into curr.Entities

	sentInput sim.Input

	notice      string
	noticeUntil time.Time
	noticeSeq   uint32

	art      *art
	face     text.Face
	darkness *ebiten.Image
	drawList []drawable
	frame    int
}

// New returns a game that plays over conn under the given player name.
func New(conn Conn, name string) *Game {
	return &Game{
		conn:        conn,
		name:        name,
		prevPlayers: make(map[sim.PlayerID]protocol.PlayerView),
		entAt:       make(map[[2]int]int),
		face:        text.NewGoXFace(bitmapfont.Face),
	}
}

// Layout uses the window size as the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return max(outsideWidth, 320), max(outsideHeight, 240)
}

// Update runs once per tick of the client loop (60 Hz).
func (g *Game) Update() error {
	g.frame++
	if g.art == nil {
		g.art = newArt() // images are created once the game loop is running
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if g.disconnected {
		return nil
	}
	if !g.helloSent {
		hello := protocol.Hello{Version: protocol.Version, Name: g.name}
		g.helloSent = g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: hello}) == nil
	}
	g.receive()
	if g.ready && g.hasCurr && !g.disconnected {
		g.handleInput()
	}
	return nil
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
		g.ready = true
	case protocol.ServerSnapshot:
		if !g.ready || (g.hasCurr && m.Snapshot.Tick <= g.curr.Tick) {
			return
		}
		g.onSnapshot(m.Snapshot)
	}
}

func (g *Game) onSnapshot(s protocol.Snapshot) {
	clear(g.prevPlayers)
	for _, p := range g.curr.Players {
		g.prevPlayers[p.ID] = p
	}
	g.curr, g.currAt, g.hasCurr = s, time.Now(), true

	clear(g.entAt)
	for i, e := range s.Entities {
		g.entAt[[2]int{int(e.TX), int(e.TY)}] = i
	}

	if s.Self.NoticeSeq != g.noticeSeq {
		g.noticeSeq = s.Self.NoticeSeq
		if txt := noticeText(s.Self.Notice, s.Self.NoticeArg); txt != "" {
			g.notice, g.noticeUntil = txt, time.Now().Add(noticeDuration)
		}
	}
}

func (g *Game) handleInput() {
	var in sim.Input
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
	in.Action = ebiten.IsKeyPressed(ebiten.KeySpace)

	// Held input is sent only when it changes; if the send fails it is
	// retried next frame because sentInput is left untouched.
	if in != g.sentInput {
		if g.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: in}) == nil {
			g.sentInput = in
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.command(sim.Command{Kind: sim.CmdEat, Item: sim.ItemBerry})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.command(sim.Command{Kind: sim.CmdCraft, Recipe: sim.RecipeCampfire})
	}
	if g.curr.Self.Dead && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.command(sim.Command{Kind: sim.CmdRespawn})
	}
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
			return "Đã dựng " + r.Name
		}
	}
	return ""
}
