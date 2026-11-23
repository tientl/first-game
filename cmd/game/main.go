// Command game is the playable client. For single player it also runs the
// server in the same process and connects to it locally.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
	"survivalgame/internal/server"
)

func main() {
	seed := flag.Uint64("seed", 0, "world seed (0 = random)")
	name := flag.String("name", "", "player name")
	size := flag.Int("size", server.DefaultWorldSize, "world width and height in tiles")
	flag.Parse()

	if *seed == 0 {
		*seed = uint64(time.Now().UnixNano())
	}
	log.Printf("world seed %d", *seed)

	srv, err := server.New(server.Config{Seed: *seed, Width: *size, Height: *size})
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Run(ctx) }()

	conn, err := srv.ConnectLocal()
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowTitle("Sinh Tồn — prototype")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	runErr := ebiten.RunGame(client.New(conn, *name))

	_ = conn.Close()
	cancel()
	<-serverDone
	if runErr != nil {
		log.Fatal(runErr)
	}
}
