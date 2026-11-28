package client

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Co-op, the client's side: a game is either a single-player one, the host of
// a room (the world runs here and friends join over the network) or a guest
// in somebody else's room. The rules are the same in all three; what differs
// is the pause menu (a room's world does not stop for one player), who may
// save, and what to say when the server goes quiet.

// stallAfter is how long without a snapshot (they come 20 times a second)
// before a guest is told the server is not answering.
const stallAfter = 1500 * time.Millisecond

type roomState struct {
	joined   bool          // a guest: the server is somebody else's
	addr     string        // the room joined, for the screen
	hosting  bool          // friends can join this game
	hostInfo []string      // where friends find the room, one line each
	remote   func() int    // how many friends are connected, for the host
	lastSnap time.Time     // when the last snapshot arrived
	refused  string        // why the server turned this client away
	lag      time.Duration // how long the server has been quiet, set by Update
}

// SetJoined marks this game as a guest in the room at addr: the world is
// somebody else's, so the pause menu does not pause it and has no Save.
func (g *Game) SetJoined(addr string) {
	g.room.joined, g.room.addr = true, addr
	g.pause.items[1] = button{label: "Lưu game", sub: "Chủ phòng giữ bản lưu", disabled: true}
	g.pause.items[4].label = "Rời phòng"
	g.pause.items[5].label = "Rời phòng và thoát"
	g.pause.sel = 0
	g.room.lastSnap = time.Now()
}

// SetHosting marks this game as a room friends may join. info lists where
// they find it (an address per line) and remote counts the friends there now.
func (g *Game) SetHosting(info []string, remote func() int) {
	g.room.hosting, g.room.hostInfo, g.room.remote = true, info, remote
	if len(info) > 0 {
		g.showNotice("Phòng đã mở: " + strings.TrimPrefix(info[0], "• "))
	}
}

// friends is how many other people are in the game over the network.
func (g *Game) friends() int {
	if g.room.remote == nil {
		return 0
	}
	return g.room.remote()
}

// shared reports whether the world runs on for others while this player is in a menu.
func (g *Game) shared() bool { return g.room.joined || g.friends() > 0 }

// noteRoom runs every Update: it measures how long the server has been quiet.
func (g *Game) noteRoom() {
	g.room.lag = 0
	if g.room.joined && g.ready && !g.disconnected && !g.room.lastSnap.IsZero() {
		g.room.lag = time.Since(g.room.lastSnap)
	}
}

// drawRoomStatus draws what the player must know about the connection: why
// they were turned away, that the connection is lost, or that it has gone quiet.
func (g *Game) drawRoomStatus(ui *uiDst, uw, uh int) {
	cx, cy := float64(uw/2), float64(uh/2)
	switch {
	case g.disconnected:
		g.dim(ui, uw, uh, 0.6)
		title, hint := "Mất kết nối tới máy chủ", "Nhấn Esc để về menu"
		if g.room.refused != "" {
			title = "Không vào được phòng"
		}
		g.text(ui, title, cx, cy-34, colWhite, text.AlignCenter, 2)
		if g.room.refused != "" {
			for i, line := range wrapWords(g.room.refused, 460) {
				g.text(ui, line, cx, cy+4+float64(i)*18, color.RGBA{255, 214, 150, 255}, text.AlignCenter, 1)
			}
			cy += float64(len(wrapWords(g.room.refused, 460)))*18 - 18
		}
		g.text(ui, hint, cx, cy+30, colMuted, text.AlignCenter, 1)
	case g.room.lag > stallAfter:
		g.dim(ui, uw, uh, 0.35)
		dots := strings.Repeat(".", 1+int(time.Now().UnixMilli()/400)%3)
		g.text(ui, "Đang chờ máy chủ"+dots, cx, cy-16, colWhite, text.AlignCenter, 2)
		g.text(ui, fmt.Sprintf("Chưa nhận được gì %d giây · Esc để thoát ra menu", int(g.room.lag.Seconds())), cx, cy+20, colMuted, text.AlignCenter, 1)
	}
}

// drawRoomPause adds what the room says to the pause menu: where friends
// join, who is there, and that the world keeps running.
func (g *Game) drawRoomPause(ui *uiDst, cx, y float64) {
	switch {
	case g.room.joined:
		g.text(ui, "Phòng "+g.room.addr+" · thế giới vẫn chạy trong lúc bạn xem menu", cx, y, colMuted, text.AlignCenter, 1)
	case g.room.hosting:
		n := g.friends()
		head := "Phòng đang mở, chưa có ai vào"
		if n > 0 {
			head = fmt.Sprintf("Phòng đang mở · %d bạn đang chơi · thế giới vẫn chạy", n)
		}
		g.text(ui, head, cx, y, colMuted, text.AlignCenter, 1)
		for i, l := range g.room.hostInfo {
			g.text(ui, l, cx, y+20+float64(i)*18, colBoneText, text.AlignCenter, 1)
		}
		if n > 0 {
			g.text(ui, "Về menu chính sẽ đóng phòng", cx, y+22+float64(len(g.room.hostInfo))*18, color.RGBA{255, 190, 140, 255}, text.AlignCenter, 1)
		}
	}
}

