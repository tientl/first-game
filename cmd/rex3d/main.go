// Command rex3d bakes the 3D tyrant and writes a sheet of its every frame
// facing each of the eight directions, enlarged, on the game's grass: a way
// to look the model over without starting the game.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"time"

	"survivalgame/internal/dino3d"
)

func main() {
	out := flag.String("o", "rex3d.png", "output PNG")
	zoom := flag.Int("zoom", 3, "how many times to enlarge")
	flag.Parse()
	dirs := make([]int, dino3d.Dirs)
	for i := range dirs {
		dirs[i] = i
	}
	start := time.Now()
	s := dino3d.BakeRex(dirs...)
	took := time.Since(start)
	start = time.Now()
	dino3d.BakeRex(dino3d.DirRight, dino3d.DirLeft)
	fmt.Printf("baked 8 directions in %v; the game's two in %v\n", took.Round(time.Millisecond), time.Since(start).Round(time.Millisecond))
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, dino3d.RexSheet(s, *zoom)); err != nil {
		log.Fatal(err)
	}
}
