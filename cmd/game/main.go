// Command game is the playable client. It opens on the main menu; for single
// player it runs the server in the same process and connects to it locally.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

const autosaveEvery = time.Minute

func main() {
	seed := flag.Uint64("seed", 0, "world seed for a new game (0 = random)")
	name := flag.String("name", "", "player name: filled in on the character screen, and who \"Chơi tiếp\" continues as")
	size := flag.Int("size", 0, "world width and height in tiles for a new game (0: the largest)")
	savePath := flag.String("save", "", "save file (default: in the user config directory)")
	dev := flag.Bool("dev", false, "dev mode: straight into a small new world, god mode, the console on the ` key; see dev.go")
	flag.Parse()

	if *savePath == "" {
		p, err := server.DefaultSavePath()
		if err != nil {
			log.Fatalf("no place to keep the save file: %v (use -save)", err)
		}
		if *dev {
			p = devSavePath(p)
		}
		*savePath = p
	}
	log.Printf("save file: %s", *savePath)
	client.InitAudio(filepath.Join(filepath.Dir(*savePath), "settings.json"))

	a := &app{seed: *seed, name: *name, size: *size, savePath: *savePath, dev: *dev}
	a.openMenu()
	if a.dev {
		a.startDev() // see dev.go
	}

	ebiten.SetWindowTitle("Dino Isles: Survival")
	ebiten.SetWindowIcon(client.GameIcons())
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	err := ebiten.RunGame(a)
	a.endSession() // saves, if a game is running
	if err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}

// app switches between the main menu and a running game.
type app struct {
	seed     uint64
	name     string
	size     int
	savePath string
	dev      bool // -dev: see dev.go

	menu *client.Menu
	sess *session

	// Who the next game is joined as: the character made for a new game, or
	// the saved player for "Chơi tiếp" (who keeps the saved look).
	look      sim.Look
	join      string
	savedName string // the player in the save, from openMenu

	// loading delivers the server being made or loaded in the background, so
	// the window stays alive while a big world is generated.
	loading chan loaded
}

type loaded struct {
	srv *server.Server
	err error
	why string // what failed, for the menu
}

// session is one game in progress: a local server and the client playing it.
type session struct {
	srv    *server.Server
	cancel context.CancelFunc
	done   chan error
	conn   *server.LocalConn
	game   *client.Game
}

func (a *app) Update() error {
	if ebiten.IsWindowBeingClosed() {
		return ebiten.Termination // main saves on the way out
	}
	if a.sess != nil {
		if err := a.sess.game.Update(); err != nil {
			return err
		}
		switch a.sess.game.Exit() {
		case client.ExitToMenu:
			a.endSession()
			a.openMenu()
		case client.ExitQuit:
			return ebiten.Termination
		}
		return nil
	}

	if a.loading != nil {
		select {
		case r := <-a.loading:
			a.loading = nil
			a.menu.SetBusy("")
			if r.err != nil {
				a.menu.SetStatus(r.why + ": " + r.err.Error())
				return nil
			}
			a.start(r.srv)
		default:
			a.menu.Update() // busy: only the backdrop moves
		}
		return nil
	}

	switch a.menu.Update() {
	case client.MenuContinue:
		a.look, a.join = sim.Look{}, a.name
		if a.join == "" {
			a.join = a.savedName
		}
		cfg, path := a.config(), a.savePath
		a.background("Đang mở thế giới", func() loaded {
			f, err := server.ReadSave(path)
			if err != nil {
				return loaded{err: err, why: "Không đọc được bản lưu"}
			}
			srv, err := server.NewFromSave(cfg, f.World)
			return loaded{srv, err, "Không mở được bản lưu"}
		})
	case client.MenuNew:
		a.look, a.join = a.menu.NewLook(), a.menu.NewName()
		cfg := a.config()
		cfg.Seed = a.seed
		if cfg.Seed == 0 {
			cfg.Seed = uint64(time.Now().UnixNano())
		}
		size := sim.SizeHuge // every new world is the largest
		if a.size > 0 {
			size = a.size // -size on the command line wins
		}
		cfg.Width, cfg.Height, cfg.Style = size, size, sim.StyleLatest
		cfg.Difficulty = a.menu.NewDifficulty()
		log.Printf("new world, seed %d, %dx%d, %s", cfg.Seed, size, size, cfg.Difficulty.Name())
		a.background("Đang tạo thế giới", func() loaded {
			srv, err := server.New(cfg)
			return loaded{srv, err, "Không tạo được thế giới"}
		})
	case client.MenuQuit:
		return ebiten.Termination
	}
	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	if a.sess != nil {
		a.sess.game.Draw(screen)
		return
	}
	a.menu.Draw(screen)
}

func (a *app) Layout(w, h int) (int, int) { return client.Layout(w, h) }

func (a *app) config() server.Config {
	return server.Config{SavePath: a.savePath, AutosaveEvery: autosaveEvery, Dev: a.dev}
}

// background runs job off the game loop while the menu shows msg.
func (a *app) background(msg string, job func() loaded) {
	ch := make(chan loaded, 1)
	go func() { ch <- job() }()
	a.loading = ch
	a.menu.SetStatus("")
	a.menu.SetBusy(msg)
}

func (a *app) start(srv *server.Server) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	conn, err := srv.ConnectLocal()
	if err != nil {
		cancel()
		<-done
		a.menu.SetStatus("Không kết nối được: " + err.Error())
		return
	}
	g := client.New(conn, a.join)
	g.SetLook(a.look)
	g.SetPauseHook(srv.SetPaused)
	g.SetDev(a.dev)
	a.sess = &session{srv: srv, cancel: cancel, done: done, conn: conn, game: g}
	a.menu = nil
}

// endSession stops the server, which saves the world as it shuts down.
func (a *app) endSession() {
	if a.sess == nil {
		return
	}
	a.sess.srv.SetPaused(false)
	a.sess.cancel()
	if err := <-a.sess.done; err != nil {
		log.Printf("server: %v", err)
	}
	_ = a.sess.conn.Close()
	a.sess = nil
}

func (a *app) openMenu() {
	info := client.SaveInfo{}
	f, err := server.ReadSave(a.savePath)
	switch {
	case err == nil:
		info.Exists = true
		info.SavedAt = f.SavedAt
		info.Day = sim.DayOf(f.World.Tick)
		if f.World.Style != sim.StyleIsland {
			info.Size = f.World.Width
			info.Difficulty = f.World.Difficulty
		}
		if len(f.World.Players) > 0 {
			info.Name = f.World.Players[0].Name
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		info.Exists, info.Broken = true, true
		log.Printf("save unreadable: %v", err)
	}
	a.menu = client.NewMenu(info)
	a.menu.SetDefaultName(a.name)
	a.savedName = info.Name
}
