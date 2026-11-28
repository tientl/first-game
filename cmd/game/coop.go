package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"survivalgame/internal/client"
	"survivalgame/internal/netplay"
)

// Playing together. "Tạo phòng" starts the world as usual and also lets
// friends in over TCP (internal/netplay); "Vào phòng" dials a host and plays
// in its world, with the same client and the same protocol as ever.

// dialing is a "Vào phòng" in progress: the connection is made off the game
// loop so the window stays alive, and Esc gives up.
type dialing struct {
	addr   string
	cancel context.CancelFunc
	result chan dialResult
}

type dialResult struct {
	conn *netplay.Conn
	err  error
}

// lastRoomPath is where the address of the last room joined is kept.
func (a *app) lastRoomPath() string {
	return filepath.Join(filepath.Dir(a.savePath), "lastroom.txt")
}

func (a *app) lastRoom() string {
	b, err := os.ReadFile(a.lastRoomPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// startDial begins joining the room the menu asked for.
func (a *app) startDial() {
	addr := a.menu.JoinAddress()
	a.look, a.join = a.menu.NewLook(), a.menu.NewName()
	ctx, cancel := context.WithCancel(context.Background())
	d := &dialing{addr: addr, cancel: cancel, result: make(chan dialResult, 1)}
	go func() {
		c, err := netplay.Dial(ctx, addr)
		d.result <- dialResult{c, err}
	}()
	a.dialing = d
	a.menu.SetStatus("")
	a.menu.SetBusy("Đang vào phòng " + netplay.JoinAddr(addr))
	a.menu.SetBusyHint("Esc để huỷ")
}

// updateDial runs each frame while a connection is being made.
func (a *app) updateDial() {
	d := a.dialing
	a.menu.Update() // busy: only the backdrop moves
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.cancel()
	}
	select {
	case r := <-d.result:
		a.dialing = nil
		d.cancel()
		a.menu.SetBusy("")
		a.menu.SetBusyHint("")
		if r.err != nil {
			log.Printf("join %s: %v", d.addr, r.err)
			a.menu.JoinFailed(netplay.Describe(r.err))
			return
		}
		if err := os.WriteFile(a.lastRoomPath(), []byte(d.addr), 0o644); err != nil {
			log.Printf("last room: %v", err)
		}
		a.startGuest(r.conn, netplay.JoinAddr(d.addr))
	default:
	}
}

// startGuest plays in the room conn is connected to.
func (a *app) startGuest(conn *netplay.Conn, addr string) {
	g := client.New(conn, a.join)
	g.SetLook(a.look) // a returning player keeps the character the host saved
	g.SetJoined(addr)
	g.SetDev(false)
	a.sess = &session{conn: conn, game: g}
	a.menu = nil
}

// roomLines are what the host's pause menu tells friends to type: every
// address of this machine that someone could reach, with the port.
func roomLines(port int) []string {
	var lines []string
	for _, ad := range netplay.HostAddresses() {
		l := fmt.Sprintf("• %s:%d", ad.IP, port)
		switch ad.Kind {
		case "LAN":
			l += "  (cùng mạng nhà)"
		case "Tailscale", "ZeroTier":
			l += "  (" + ad.Kind + ")"
		}
		lines = append(lines, l)
		if len(lines) == 4 {
			break
		}
	}
	if len(lines) == 0 {
		lines = append(lines, fmt.Sprintf("Chưa thấy mạng nào: cắm dây hoặc bật Wi-Fi (cổng %d)", port))
	}
	log.Printf("room open, friends join at: %s", strings.Join(lines, " "))
	return lines
}
