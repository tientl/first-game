package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"

	"survivalgame/internal/sim"
)

// The islands and their animals: what -isles, -fauna and -faunamap report.

var isleKindNames = map[sim.IslandKind]string{
	sim.IslandMeadow: "meadow", sim.IslandForest: "forest", sim.IslandMarsh: "marsh",
	sim.IslandSnow: "snow", sim.IslandDesert: "desert", sim.IslandAsh: "ash",
	sim.IslandMonkey: "monkey", sim.IslandAtoll: "atoll",
}

func isleName(is sim.Island) string {
	if is.Home {
		return "home"
	}
	if n, ok := isleKindNames[is.Kind]; ok {
		return n
	}
	return fmt.Sprintf("kind%d", is.Kind)
}

// isleBox is an island's extent and middle.
type isleBox struct {
	x0, y0, x1, y1 int
	mx, my         float64
	n              int
}

func isleBoxes(w *sim.World) ([]isleBox, []int16) {
	W, _ := w.Size()
	im := w.IslandMap()
	boxes := make([]isleBox, len(w.Islands()))
	for i := range boxes {
		boxes[i] = isleBox{x0: math.MaxInt, y0: math.MaxInt, x1: -1, y1: -1}
	}
	for i, k := range im {
		if k < 0 {
			continue
		}
		b := &boxes[k]
		x, y := i%W, i/W
		b.x0, b.y0, b.x1, b.y1 = min(b.x0, x), min(b.y0, y), max(b.x1, x), max(b.y1, y)
		b.mx += float64(x) + 0.5
		b.my += float64(y) + 0.5
		b.n++
	}
	for i := range boxes {
		if boxes[i].n > 0 {
			boxes[i].mx /= float64(boxes[i].n)
			boxes[i].my /= float64(boxes[i].n)
		}
	}
	return boxes, im
}

// A 1080p screen shows about this many tiles at the default zoom.
const screenW, screenH = 40.0, 22.0

func isleReport(w *sim.World) {
	isl := w.Islands()
	boxes, _ := isleBoxes(w)
	sx, sy := w.Spawn()
	type row struct {
		i int
		d float64
	}
	var rows []row
	for i := range isl {
		rows = append(rows, row{i, math.Hypot(boxes[i].mx-sx, boxes[i].my-sy)})
	}
	sort.Slice(rows, func(a, b int) bool { return isl[rows[a].i].Tiles > isl[rows[b].i].Tiles })
	land := 0
	for _, is := range isl {
		land += is.Tiles
	}
	fmt.Printf("islands: %d, %d tiles of land in them\n", len(isl), land)
	fmt.Printf("  %-7s %7s %9s %13s %6s\n", "kind", "tiles", "w x h", "screens", "away")
	for _, r := range rows {
		is, b := isl[r.i], boxes[r.i]
		bw, bh := b.x1-b.x0+1, b.y1-b.y0+1
		if is.Tiles < 400 {
			continue
		}
		fmt.Printf("  %-7s %7d %4dx%-4d %5.1fx%-5.1f %6.0f\n", isleName(is), is.Tiles, bw, bh, float64(bw)/screenW, float64(bh)/screenH, r.d)
	}
	// By kind: how many, and the biggest.
	type agg struct {
		n, small, tiles, big, bw, bh int
	}
	by := map[string]*agg{}
	for i, is := range isl {
		a := by[isleName(is)]
		if a == nil {
			a = &agg{}
			by[isleName(is)] = a
		}
		a.n++
		if is.Tiles < 400 {
			a.small++
		}
		a.tiles += is.Tiles
		if is.Tiles > a.big {
			a.big, a.bw, a.bh = is.Tiles, boxes[i].x1-boxes[i].x0+1, boxes[i].y1-boxes[i].y0+1
		}
	}
	names := make([]string, 0, len(by))
	for k := range by {
		names = append(names, k)
	}
	sort.Strings(names)
	fmt.Printf("  by kind: %-7s %4s %6s %8s %8s %9s %11s\n", "kind", "n", "<400", "tiles", "biggest", "w x h", "screens")
	for _, k := range names {
		a := by[k]
		fmt.Printf("           %-7s %4d %6d %8d %8d %4dx%-4d %5.1fx%-5.1f\n", k, a.n, a.small, a.tiles, a.big, a.bw, a.bh, float64(a.bw)/screenW, float64(a.bh)/screenH)
	}
	// The ground of each big island, by biome.
	fmt.Println("  biomes of the islands of 8000+ tiles (% of the land; sand is the beaches):")
	_, im := isleBoxes(w)
	tiles, rel := w.Tiles(), w.Relief()
	shares := make([]map[string]int, len(isl))
	for i, k := range im {
		if k < 0 {
			continue
		}
		if shares[k] == nil {
			shares[k] = map[string]int{}
		}
		shares[k][biomeName(tiles[i], rel[i])]++
	}
	for _, r := range rows {
		is := isl[r.i]
		if is.Tiles < 8000 {
			continue
		}
		type kv struct {
			k string
			n int
		}
		var kvs []kv
		for k, n := range shares[r.i] {
			kvs = append(kvs, kv{k, n})
		}
		sort.Slice(kvs, func(a, b int) bool { return kvs[a].n > kvs[b].n })
		fmt.Printf("    %-7s %6d:", isleName(is), is.Tiles)
		for _, e := range kvs {
			if p := 100 * float64(e.n) / float64(is.Tiles); p >= 1 {
				fmt.Printf(" %s %.0f%%", e.k, p)
			}
		}
		fmt.Println()
	}
	// The bosses: on which island, and how it ranks among its kind by size;
	// for one at sea, how far the nearest land is.
	W, _ := w.Size()
	tiles2 := tiles[:W*w.SurfaceHeight()]
	for _, c := range w.Creatures() {
		if !c.Kind.Boss() {
			continue
		}
		tx, ty := int(c.X), int(c.Y)
		near, nearD := -1, math.Inf(1)
		for dy := -70; dy <= 70; dy++ {
			for dx := -70; dx <= 70; dx++ {
				x, y := tx+dx, ty+dy
				if x < 0 || y < 0 || x >= W || y >= w.SurfaceHeight() || tiles2[y*W+x].Sea() || im[y*W+x] < 0 {
					continue
				}
				if d := math.Hypot(float64(dx), float64(dy)); d < nearD {
					near, nearD = int(im[y*W+x]), d
				}
			}
		}
		if near < 0 {
			fmt.Printf("  boss %-20s at %4d,%-4d: nearest land %.0f tiles\n", c.Kind.Name(), tx, ty, nearD)
			continue
		}
		rank := 1
		for _, o := range isl {
			if o.Kind == isl[near].Kind && !o.Home && o.Tiles > isl[near].Tiles {
				rank++
			}
		}
		fmt.Printf("  boss %-20s at %4d,%-4d: on %s of %d tiles (#%d of its kind), nearest land %.0f tiles\n", c.Kind.Name(), tx, ty, isleName(isl[near]), isl[near].Tiles, rank, nearD)
	}
	// How many of the islands of each size band.
	bands := []int{40, 400, 2000, 8000, 20000, 50000, math.MaxInt}
	cnt := make([]int, len(bands)-1)
	for _, is := range isl {
		for b := 0; b+1 < len(bands); b++ {
			if is.Tiles >= bands[b] && is.Tiles < bands[b+1] {
				cnt[b]++
			}
		}
	}
	fmt.Printf("  by size: 40-399 %d, 400-1999 %d, 2000-7999 %d, 8000-19999 %d, 20000-49999 %d, 50000+ %d\n", cnt[0], cnt[1], cnt[2], cnt[3], cnt[4], cnt[5])
}

// Kinds the reports group as big herbivores and as predators of the land.
var herbivores = map[sim.CreatureKind]bool{
	sim.CreatureTrike: true, sim.CreatureStego: true, sim.CreatureBrachio: true, sim.CreatureMammoth: true,
	sim.CreatureOrnitho: true, sim.CreatureThesc: true, sim.CreatureToro: true, sim.CreatureAnzu: true, sim.CreatureAnkylo: true,
}
var predators = map[sim.CreatureKind]bool{
	sim.CreatureRaptor: true, sim.CreatureAchero: true, sim.CreatureRex: true, sim.CreatureSabre: true,
	sim.CreatureQuetzal: true, sim.CreaturePtero: true, sim.CreatureScorpion: true,
	sim.CreatureCroc: true, sim.CreatureBrachy: true, sim.CreatureChampso: true,
}

var faunaOrder = []sim.CreatureKind{
	sim.CreatureRabbit, sim.CreatureThesc, sim.CreatureOrnitho, sim.CreatureTrike, sim.CreatureToro, sim.CreatureStego,
	sim.CreatureBrachio, sim.CreatureAnzu, sim.CreatureAnkylo, sim.CreatureMammoth, sim.CreatureMonkey,
	sim.CreatureAchero, sim.CreatureRaptor, sim.CreatureRex, sim.CreatureSabre, sim.CreatureQuetzal, sim.CreaturePtero,
	sim.CreatureScorpion, sim.CreatureCroc, sim.CreatureBrachy, sim.CreatureChampso, sim.CreatureSnake,
	sim.CreaturePlesio, sim.CreatureMosa, sim.CreatureKraken, sim.CreatureFish,
	sim.CreatureSpider, sim.CreatureBear,
}

var shortNames = map[sim.CreatureKind]string{
	sim.CreatureRabbit: "rabbit", sim.CreatureThesc: "thesc", sim.CreatureOrnitho: "ornitho", sim.CreatureTrike: "trike",
	sim.CreatureToro: "toro", sim.CreatureStego: "stego", sim.CreatureBrachio: "brachio", sim.CreatureAnzu: "anzu",
	sim.CreatureMammoth: "mammoth", sim.CreatureAnkylo: "ankylo", sim.CreatureMonkey: "monkey", sim.CreatureAchero: "achero", sim.CreatureRaptor: "dakota",
	sim.CreatureRex: "rex", sim.CreatureSabre: "sabre", sim.CreatureQuetzal: "quetzal", sim.CreaturePtero: "ptero",
	sim.CreatureScorpion: "scorpion", sim.CreatureCroc: "croc", sim.CreatureBrachy: "brachy", sim.CreatureChampso: "champso",
	sim.CreatureSnake: "snake", sim.CreaturePlesio: "plesio", sim.CreatureMosa: "mosasaur", sim.CreatureKraken: "kraken",
	sim.CreatureFish: "fish", sim.CreatureSpider: "spider", sim.CreatureBear: "bear",
}

func faunaReport(w *sim.World) {
	W, _ := w.Size()
	SH := w.SurfaceHeight()
	isl := w.Islands()
	_, im := isleBoxes(w)
	sx, sy := w.Spawn()
	bands := []float64{40, 80, 150, 250, 400, math.Inf(1)}
	cols := []string{"meadow", "forest", "marsh", "snow", "desert", "ash", "monkey", "atoll"}
	colOf := func(c sim.Creature) string {
		tx, ty := int(c.X), int(c.Y)
		if ty >= SH {
			return "cave"
		}
		if k := im[ty*W+tx]; k >= 0 {
			if isl[k].Home {
				return "home"
			}
			return isleName(isl[k])
		}
		return "sea"
	}
	type stat struct {
		total int
		band  [6]int
		col   map[string]int
	}
	st := map[sim.CreatureKind]*stat{}
	for _, c := range w.Creatures() {
		s := st[c.Kind]
		if s == nil {
			s = &stat{col: map[string]int{}}
			st[c.Kind] = s
		}
		s.total++
		d := math.Hypot(c.X-sx, c.Y-sy)
		for b := range bands {
			if d < bands[b] {
				s.band[b]++
				break
			}
		}
		s.col[colOf(c)]++
	}
	all := append([]string{"home"}, cols...)
	all = append(all, "sea", "cave")
	fmt.Printf("%-9s %6s | %5s %5s %6s %7s %7s %5s |", "kind", "total", "<40", "40-80", "80-150", "150-250", "250-400", "far")
	for _, c := range all {
		fmt.Printf(" %6s", c)
	}
	fmt.Println()
	herb, pred := 0, 0
	for _, k := range faunaOrder {
		s := st[k]
		if s == nil {
			s = &stat{col: map[string]int{}}
		}
		if herbivores[k] {
			herb += s.total
		}
		if predators[k] {
			pred += s.total
		}
		fmt.Printf("%-9s %6d | %5d %5d %6d %7d %7d %5d |", shortNames[k], s.total, s.band[0], s.band[1], s.band[2], s.band[3], s.band[4], s.band[5])
		for _, c := range all {
			fmt.Printf(" %6d", s.col[c])
		}
		fmt.Println()
	}
	total := 0
	for _, s := range st {
		total += s.total
	}
	fmt.Printf("all creatures %d; big herbivores %d, land predators %d: %.1f to 1\n", total, herb, pred, float64(herb)/math.Max(1, float64(pred)))

	// Islands of a meaningful size with no big animal at all.
	big := make([]int, len(isl))
	for _, c := range w.Creatures() {
		tx, ty := int(c.X), int(c.Y)
		if ty >= SH || !(herbivores[c.Kind] || predators[c.Kind] && c.Kind != sim.CreaturePtero) {
			continue
		}
		if k := im[ty*W+tx]; k >= 0 {
			big[k]++
		}
	}
	for _, min := range []int{400, 1500, 4000} {
		n, empty := 0, 0
		for k, is := range isl {
			if is.Tiles >= min {
				n++
				if big[k] == 0 {
					empty++
				}
			}
		}
		fmt.Printf("islands of %d+ tiles: %d, with no big animal: %d\n", min, n, empty)
	}
}

// writeFaunaMap draws the surface, a pixel to a tile, with every big animal
// as a dot: herbivores green, predators red, the sea's beasts orange.
func writeFaunaMap(w *sim.World, path string) error {
	W, _ := w.Size()
	H := w.SurfaceHeight()
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	tiles := w.Tiles()
	for i := 0; i < W*H; i++ {
		img.Set(i%W, i/W, tileColour(tiles[i]))
	}
	dot := func(x, y float64, c color.RGBA, r int) {
		cx, cy := int(x), int(y)
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r+1 && cx+dx >= 0 && cy+dy >= 0 && cx+dx < W && cy+dy < H {
					img.Set(cx+dx, cy+dy, c)
				}
			}
		}
	}
	for _, c := range w.Creatures() {
		if int(c.Y) >= H {
			continue
		}
		switch {
		case herbivores[c.Kind]:
			dot(c.X, c.Y, color.RGBA{30, 230, 60, 255}, 2)
		case predators[c.Kind]:
			dot(c.X, c.Y, color.RGBA{240, 30, 30, 255}, 2)
		case c.Kind == sim.CreatureMosa || c.Kind == sim.CreatureKraken:
			dot(c.X, c.Y, color.RGBA{255, 150, 0, 255}, 2)
		case c.Kind == sim.CreaturePlesio:
			dot(c.X, c.Y, color.RGBA{255, 240, 60, 255}, 1)
		case c.Kind.Boss():
			dot(c.X, c.Y, color.RGBA{200, 0, 200, 255}, 4)
		}
	}
	sx, sy := w.Spawn()
	dot(sx, sy, color.RGBA{255, 255, 255, 255}, 3)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// biomeName is the biome a tile of land counts as in the island report.
func biomeName(t sim.Tile, r sim.Relief) string {
	switch t {
	case sim.TileGrass:
		return "grass"
	case sim.TileForest:
		return "woods"
	case sim.TileMarsh:
		return "marsh"
	case sim.TileDesert:
		return "desert"
	case sim.TileSnow:
		if r.Variant() == sim.VarDenseWood {
			return "taiga"
		}
		return "snow"
	case sim.TileAsh, sim.TileLava:
		return "ash"
	case sim.TileRocky, sim.TileCliff:
		return "rock"
	case sim.TileSand:
		return "sand"
	case sim.TileRiver, sim.TileFord, sim.TileWaterfall, sim.TileWater, sim.TileShallow:
		return "water"
	case sim.TilePath:
		return "path"
	}
	return "other"
}

func tileColour(t sim.Tile) color.RGBA {
	switch t {
	case sim.TileWater:
		return color.RGBA{28, 60, 110, 255}
	case sim.TileShallow:
		return color.RGBA{50, 110, 160, 255}
	case sim.TileSand:
		return color.RGBA{214, 198, 140, 255}
	case sim.TileGrass:
		return color.RGBA{96, 150, 70, 255}
	case sim.TileForest:
		return color.RGBA{44, 92, 46, 255}
	case sim.TileMarsh:
		return color.RGBA{80, 104, 70, 255}
	case sim.TileDesert:
		return color.RGBA{200, 160, 100, 255}
	case sim.TileSnow:
		return color.RGBA{230, 236, 240, 255}
	case sim.TileAsh:
		return color.RGBA{80, 76, 74, 255}
	case sim.TileLava:
		return color.RGBA{220, 80, 20, 255}
	case sim.TileRocky:
		return color.RGBA{140, 136, 128, 255}
	case sim.TileCliff, sim.TileMountain:
		return color.RGBA{100, 92, 84, 255}
	case sim.TileRiver, sim.TileWaterfall:
		return color.RGBA{70, 130, 200, 255}
	}
	if t.Sea() {
		return color.RGBA{40, 80, 130, 255}
	}
	return color.RGBA{120, 130, 100, 255}
}

// writeIslesMap draws the surface, a pixel to a tile, with each boss's lair
// ringed in magenta (its arena's size, at least) and the spawn in white.
func writeIslesMap(w *sim.World, path string) error {
	W, _ := w.Size()
	H := w.SurfaceHeight()
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	tiles := w.Tiles()
	for i := 0; i < W*H; i++ {
		img.Set(i%W, i/W, tileColour(tiles[i]))
	}
	ring := func(x, y, r float64, c color.RGBA) {
		for a := 0.0; a < 2*math.Pi; a += 0.5 / r {
			for t := -1.0; t <= 1; t++ {
				px, py := int(x+math.Cos(a)*(r+t)), int(y+math.Sin(a)*(r+t))
				if px >= 0 && py >= 0 && px < W && py < H {
					img.Set(px, py, c)
				}
			}
		}
	}
	for _, c := range w.Creatures() {
		if c.Kind.Boss() {
			ring(c.X, c.Y, 22, color.RGBA{255, 0, 255, 255})
			ring(c.X, c.Y, 4, color.RGBA{255, 0, 255, 255})
		}
	}
	sx, sy := w.Spawn()
	ring(sx, sy, 6, color.RGBA{255, 255, 255, 255})
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// zoneSpots prints, for the biggest island of each kind, the tile where its
// main ground and its most common other ground meet most evenly: a place to
// see a zone's border from.
func zoneSpots(w *sim.World) {
	W, _ := w.Size()
	H := w.SurfaceHeight()
	isl := w.Islands()
	_, im := isleBoxes(w)
	tiles, rel := w.Tiles(), w.Relief()
	best := map[string]int{}
	for i, is := range isl {
		n := isleName(is)
		if b, ok := best[n]; !ok || is.Tiles > isl[b].Tiles {
			best[n] = i
		}
	}
	names := make([]string, 0, len(best))
	for n := range best {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		k := best[n]
		if isl[k].Tiles < 8000 {
			continue
		}
		count := map[string]int{}
		for i, x := range im {
			if int(x) == k {
				count[biomeName(tiles[i], rel[i])]++
			}
		}
		main, second := "", ""
		for b, c := range count {
			if b == "sand" || b == "path" || b == "water" || b == "other" {
				continue
			}
			if main == "" || c > count[main] {
				main, second = b, main
			} else if second == "" || c > count[second] {
				second = b
			}
		}
		bx, by, bs := 0, 0, -1
		for y := 12; y < H-12; y += 5 {
			for x := 12; x < W-12; x += 5 {
				if int(im[y*W+x]) != k {
					continue
				}
				a, b := 0, 0
				for dy := -12; dy <= 12; dy += 2 {
					for dx := -20; dx <= 20; dx += 2 {
						switch biomeName(tiles[(y+dy)*W+x+dx], rel[(y+dy)*W+x+dx]) {
						case main:
							a++
						case second:
							b++
						}
					}
				}
				if s := min(a, b); s > bs {
					bx, by, bs = x, y, s
				}
			}
		}
		fmt.Printf("zone %-7s %s/%s at %d,%d\n", n, main, second, bx, by)
	}
}
