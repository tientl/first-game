// Command atlas writes the game's item icons, object and animal sprites and
// a catalog.json of items and recipes to a directory, for documentation:
//
//	go run ./cmd/atlas -o atlas
package main

import (
	"errors"
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/client"
)

type exporter struct {
	dir string
	err error
}

func (e *exporter) Update() error {
	e.err = client.ExportAtlas(e.dir)
	return ebiten.Termination
}

func (e *exporter) Draw(*ebiten.Image)         {}
func (e *exporter) Layout(int, int) (int, int) { return 64, 64 }

func main() {
	dir := flag.String("o", "atlas", "output directory")
	flag.Parse()
	ebiten.SetWindowSize(64, 64)
	ebiten.SetWindowTitle("atlas")
	e := &exporter{dir: *dir}
	if err := ebiten.RunGame(e); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if e.err != nil {
		log.Fatal(e.err)
	}
	log.Printf("wrote %s", *dir)
}
