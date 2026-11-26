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
	woodsMap := flag.String("woodsmap", "", "write a map of the woods to this PNG instead (thick woods dark, ordinary mid green, trees as dots) and stop; see woodsmap.go")
	region := flag.String("region", "", "with -woodsmap: x0,y0,x1,y1, the tiles to show (default: the whole surface)")
	px := flag.Int("px", 2, "with -woodsmap: pixels to a tile")
	isles := flag.Bool("isles", false, "report every island: its kind, tiles, extent and how many 1080p screens across it is, and stop; see fauna.go")
	fauna := flag.Bool("fauna", false, "report the animals by kind: how far from the spawn, on which kind of island, herbivores to predators, and stop; see fauna.go")
	faunaMap := flag.String("faunamap", "", "with -fauna: also write the map with the big animals as dots to this PNG (herbivores green, predators red)")
	islesMap := flag.String("islesmap", "", "with -isles: also write the map with the bosses' lairs ringed to this PNG")
	zones := flag.Bool("zones", false, "with -isles: print where on the biggest island of each kind its main ground meets another, for screenshots")
	flag.Parse()
	if *isles || *fauna {
		w, err := sim.Generate(sim.GenConfig{Seed: *seed, Width: *size, Height: *size, Style: sim.WorldStyle(*style)})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("world %dx%d, style %d, seed %d\n", *size, *size, *style, *seed)
		if *isles {
			isleReport(w)
			if *islesMap != "" {
				if err := writeIslesMap(w, *islesMap); err != nil {
					log.Fatal(err)
				}
			}
			if *zones {
				zoneSpots(w)
			}
		}
		if *fauna {
			faunaReport(w)
			if *faunaMap != "" {
				if err := writeFaunaMap(w, *faunaMap); err != nil {
					log.Fatal(err)
				}
			}
		}
		return
	}
	if *woodsMap != "" {
		w, err := sim.Generate(sim.GenConfig{Seed: *seed, Width: *size, Height: *size, Style: sim.WorldStyle(*style)})
		if err != nil {
			log.Fatal(err)
		}
		x0, y0, x1, y1 := 0, 0, *size, *size
		fmt.Sscanf(*region, "%d,%d,%d,%d", &x0, &y0, &x1, &y1)
		if err := writeWoodsMap(w, *woodsMap, x0, y0, x1, y1, *px); err != nil {
			log.Fatal(err)
		}
		return
	}

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
	// The beaches: how deep the sand is straight inland from each tile of
	// sand on the shore (see sim.BeachDepths).
	if bd, shore := w.BeachDepths(); len(bd) > 0 {
		ds := make([]int, len(bd))
		wide := 0
		for i, b := range bd {
			ds[i] = b.N
			if b.N >= 5 {
				wide++
			}
		}
		sort.Ints(ds)
		q := func(p float64) int { return ds[min(len(ds)-1, int(p*float64(len(ds))))] }
		fmt.Printf("beaches: %d of %d shore tiles sandy; depth p10 %d, p25 %d, median %d, p75 %d, p90 %d; %.0f%% at least 5 deep\n",
			len(bd), shore, q(0.1), q(0.25), q(0.5), q(0.75), q(0.9), 100*float64(wide)/float64(len(bd)))
	}
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
