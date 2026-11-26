// Command worldstat generates a world and reports what it costs: the time
// to make it, the memory the server holds for it, the size of its save, and
// what is in it. It is a dev tool for tuning the world generator:
//
//	go run ./cmd/worldstat -size 2048
//	go run ./cmd/worldstat -size 1024 -style 7
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"runtime"
	"sort"
	"time"

	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

func main() {
	size := flag.Int("size", sim.SizeHuge, "world size in tiles")
	seed := flag.Uint64("seed", 42, "world seed")
	style := flag.Int("style", int(sim.StyleLatest), "world style (the generator)")
	flag.Parse()

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	t0 := time.Now()
	w, err := sim.Generate(sim.GenConfig{Seed: *seed, Width: *size, Height: *size, Style: sim.WorldStyle(*style)})
	if err != nil {
		log.Fatal(err)
	}
	took := time.Since(t0)
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	id := w.AddPlayer("stat")
	_ = id
	t1 := time.Now()
	const ticks = 60
	for range ticks {
		w.Step()
	}
	tick := time.Since(t1) / ticks

	t2 := time.Now()
	b, err := json.Marshal(server.SaveFile{SavedAt: time.Now(), World: w.Save()})
	if err != nil {
		log.Fatal(err)
	}
	saveT := time.Since(t2)
	t3 := time.Now()
	var f server.SaveFile
	if err := json.Unmarshal(b, &f); err != nil {
		log.Fatal(err)
	}
	if _, err := sim.Load(f.World); err != nil {
		log.Fatal(err)
	}
	loadT := time.Since(t3)

	W, H := w.Size()
	fmt.Printf("world %dx%d (map %dx%d), style %d, seed %d\n", *size, *size, W, H, *style, *seed)
	fmt.Printf("generate: %v\n", took.Round(time.Millisecond))
	fmt.Printf("heap held: %.1f MB\n", float64(after.HeapAlloc-min(after.HeapAlloc, before.HeapAlloc))/1e6)
	fmt.Printf("save: %.1f MB, write %v, read+load %v\n", float64(len(b))/1e6, saveT.Round(time.Millisecond), loadT.Round(time.Millisecond))
	fmt.Printf("tick with one player: %v\n", tick.Round(time.Microsecond))

	tiles := w.Tiles()
	surf := w.SurfaceHeight() * W
	var count [sim.NumTiles]int
	for _, t := range tiles[:surf] {
		count[t]++
	}
	land := 0
	for k, n := range count {
		if k != int(sim.TileWater) && sim.Tile(k).Walkable() {
			land += n
		}
	}
	fmt.Printf("surface tiles: %d walkable land; by kind:", land)
	for k, n := range count {
		if n > 0 {
			fmt.Printf(" %d:%d", k, n)
		}
	}
	fmt.Println()
	fmt.Printf("islands: %d\n", len(w.Islands()))
	ents := map[string]int{}
	w.EachEntityIn(0, 0, W, H, func(e sim.Entity) { ents[e.Kind.Name()]++ })
	crs := map[string]int{}
	for _, c := range w.Creatures() {
		crs[c.Kind.Name()]++
	}
	show := func(title string, m map[string]int) {
		keys := make([]string, 0, len(m))
		total := 0
		for k, n := range m {
			keys = append(keys, k)
			total += n
		}
		sort.Strings(keys)
		fmt.Printf("%s (%d):", title, total)
		for _, k := range keys {
			fmt.Printf(" %s=%d", k, m[k])
		}
		fmt.Println()
	}
	show("entities", ents)
	show("creatures", crs)
	fmt.Printf("villages: %d\n", len(w.Villages()))
}
