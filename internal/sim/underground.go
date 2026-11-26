package sim

import (
	"math"
	"math/rand/v2"
)

// The caves under the archipelago. An archipelago has no mountains: its
// caves lie underground, a dark maze of tunnels and halls in the rock. On
// the islands stand cave mouths, low arches of boulders over a hole going
// down; walking into one goes down to the caves, onto a patch of rubble
// at the foot of a slope up to the daylight of the mouth. Walking back onto it
// climbs out of the same mouth.
//
// The caves are kept in the same tile map as the islands, in rows below
// them, cut off by a thick band of rock: nobody can walk or see from one to
// the other, only go through a mouth.

// UndergroundGap is how many rows of solid rock lie between the islands and
// the caves under them: the caves start at SurfaceHeight()+UndergroundGap.
const UndergroundGap = 48

// Underground tuning.
const (
	undergroundGap  = UndergroundGap
	undergroundPad  = 8    // rows of rock under the caves
	caveMinHall     = 40   // smaller pockets are filled in
	mouthSpacing    = 14   // tiles at least between two mouths
	mouthIslandMin  = 60   // islets smaller than this have no cave
	homeMouthAway   = 16   // tiles from the spawn, on the home island
	mouthPerTiles   = 1400 // an island gets one mouth, and one more per this many tiles
	mouthsPerIsland = 3
	caveLinkReach   = 30                 // a mouth's chamber is joined to the nearest hall this close
	TravelCooldown  = TicksPerSecond     // the least time between two trips
	TravelHold      = 4 * TicksPerSecond // the most, while a walk key is held
	caveExitLight   = 3.0                // tiles of daylight round the way up
)

// SurfaceHeight is how many rows of the map are the world above ground;
// the rest, if any, are the caves.
func (w *World) SurfaceHeight() int {
	if w.surfaceH > 0 {
		return w.surfaceH
	}
	return w.height
}

// Underground reports whether tile row ty is in the caves under the world.
func (w *World) Underground(ty int) bool { return w.surfaceH > 0 && ty >= w.surfaceH }

// addUnderground grows the map downwards by the caves: rock, with tunnels
// and halls carved through it.
func (w *World) addUnderground(seed uint32) {
	W, H := w.width, w.height
	newH := H + undergroundGap + H + undergroundPad
	tiles := make([]Tile, W*newH)
	copy(tiles, w.tiles)
	for i := W * H; i < len(tiles); i++ {
		tiles[i] = TileMountain
	}
	occ := make([]EntityID, W*newH)
	copy(occ, w.occ)
	w.tiles, w.occ, w.height, w.surfaceH = tiles, occ, newH, H
	if w.relief != nil { // the caves are plain, at level 0
		w.relief = append(w.relief, make([]Relief, W*newH-len(w.relief))...)
	}
	w.fauna.shallow = nil // sized for the old map; rebuilt on demand
	top := H + undergroundGap
	w.carveMaze(seed, top, H)
	// Fill in the pockets too small to matter.
	seen := make([]bool, len(w.tiles))
	for start := W * top; start < W*(top+H); start++ {
		if w.tiles[start] != TileCave || seen[start] {
			continue
		}
		comp := []int{start}
		seen[start] = true
		for k := 0; k < len(comp); k++ {
			x, y := comp[k]%W, comp[k]/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if j := (y+d[1])*W + x + d[0]; !seen[j] && w.tiles[j] == TileCave {
					seen[j] = true
					comp = append(comp, j)
				}
			}
		}
		if len(comp) < caveMinHall {
			for _, i := range comp {
				w.tiles[i] = TileMountain
			}
		}
	}
}

// caveLink is where walking onto a mouth or the way up takes a player.
type caveLink struct{ x, y float64 }

// digCaveMouths sets cave mouths on every island but the smallest islets,
// each with its way up in the caves below, joined to the halls. On the home
// island they keep away from where players start.
func (w *World) digCaveMouths(rng *rand.Rand) {
	if w.surfaceH == 0 {
		return
	}
	W, H := w.width, w.surfaceH
	top := H + undergroundGap
	w.links = make(map[[2]int]caveLink)
	sx, sy := int(w.spawnX), int(w.spawnY)
	var placed [][2]int
	good := func(tx, ty, margin int) bool {
		if w.inVillage(tx, ty, margin) {
			return false // not in a village's clearing; see islandtribes.go
		}
		for y := ty - 1; y <= ty+1; y++ {
			for x := tx - 2; x <= tx+2; x++ {
				if y >= H || !w.TileAt(x, y).Walkable() || w.TileAt(x, y) == TilePath || w.TileAt(x, y) == TileLava || w.crossing(y*W+x) {
					return false
				}
				if e := w.entityAt(x, y); e != nil && (blocksPlayers(e.Kind) || e.Kind == EntityTreasure) {
					return false // a hut, the pyramid and its chest
				}
			}
		}
		for _, q := range placed {
			if abs(q[0]-tx)+abs(q[1]-ty) < mouthSpacing {
				return false
			}
		}
		return true
	}
	for _, isl := range w.landMasses() {
		home := false
		for _, i := range isl {
			if i%W == sx && i/W == sy {
				home = true
				break
			}
		}
		if len(isl) < mouthIslandMin || isl[0]/W >= H {
			continue
		}
		want := min(mouthsPerIsland, 1+len(isl)/mouthPerTiles)
		spots := append([]int(nil), isl...)
		rng.Shuffle(len(spots), func(a, b int) { spots[a], spots[b] = spots[b], spots[a] })
		// Well clear of the village first; on a small island crowded by it,
		// at its edge, among the huts if need be.
		got := 0
		for k, margin := range []int{3, 0, -hamletRadius} {
			if want == 0 || k > 0 && got > 0 {
				break
			}
			for _, i := range spots {
				if want == 0 {
					break
				}
				tx, ty := i%W, i/W
				if home && math.Hypot(float64(tx-sx), float64(ty-sy)) < homeMouthAway {
					continue
				}
				if !good(tx, ty, margin) {
					continue
				}
				w.placeMouth(tx, ty, tx, top+ty)
				placed = append(placed, [2]int{tx, ty})
				want--
				got++
			}
		}
	}
}

// placeMouth puts a cave mouth on (tx, ty) and its way up at (ux, uy) in
// the caves, digging out a chamber there joined to the nearest hall.
// The mouth is a doorway in the south face of an outcrop: rock stands on the
// tiles beside and behind it, and it is walked into from the tile below,
// which is kept clear.
func (w *World) placeMouth(tx, ty, ux, uy int) {
	for y := ty - 1; y <= ty+1; y++ {
		for x := tx - 2; x <= tx+2; x++ {
			if e := w.entityAt(x, y); e != nil {
				w.removeEntity(e)
			}
		}
	}
	w.addEntity(EntityCaveMouth, tx, ty)
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {-1, -1}, {0, -1}, {1, -1}} {
		w.addEntity(EntityCaveRock, tx+d[0], ty+d[1])
	}
	// The chamber under it.
	for y := uy - 2; y <= uy+3; y++ {
		for x := ux - 3; x <= ux+3; x++ {
			if dx, dy := float64(x-ux)/3.2, float64(y-uy)/2.6; dx*dx+dy*dy <= 1 && w.inBounds(x, y) {
				w.tiles[y*w.width+x] = TileCave
				if e := w.entityAt(x, y); e != nil {
					w.removeEntity(e)
				}
			}
		}
	}
	w.joinToHalls(ux, uy)
	w.addEntity(EntityCaveExit, ux, uy)
	w.links[[2]int{tx, ty}] = caveLink{float64(ux) + 0.5, float64(uy) + 1.5}
	w.links[[2]int{ux, uy}] = caveLink{float64(tx) + 0.5, float64(ty) + 1.5}
}

// joinToHalls digs a tunnel from (ux, uy) to the nearest cave tile outside
// its own chamber, if one is near.
func (w *World) joinToHalls(ux, uy int) {
	bx, by, best := 0, 0, math.MaxInt
	for y := uy - caveLinkReach; y <= uy+caveLinkReach; y++ {
		for x := ux - caveLinkReach; x <= ux+caveLinkReach; x++ {
			if !w.inBounds(x, y) || !w.Underground(y) || w.tiles[y*w.width+x] != TileCave {
				continue
			}
			if abs(x-ux) <= 4 && abs(y-uy) <= 4 {
				continue // the chamber itself
			}
			if d := abs(x-ux) + abs(y-uy); d < best {
				bx, by, best = x, y, d
			}
		}
	}
	if best == math.MaxInt {
		return
	}
	// Across, then down or up: a tunnel two tiles wide.
	dig := func(x, y int) {
		for _, d := range [][2]int{{0, 0}, {0, 1}, {1, 0}} {
			if nx, ny := x+d[0], y+d[1]; w.inBounds(nx, ny) && w.Underground(ny) && w.tiles[ny*w.width+nx] == TileMountain {
				w.tiles[ny*w.width+nx] = TileCave
			}
		}
	}
	x, y := ux, uy
	for x != bx {
		dig(x, y)
		x += stepToward(bx - x)
	}
	for y != by {
		dig(x, y)
		y += stepToward(by - y)
	}
}

func stepToward(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// travel takes p (and the beast p rides) through a cave mouth or up the way
// out, when p steps onto one.
func (w *World) travel(p *Player) {
	// After a trip the next waits until the walk keys are let go (or a
	// while has passed), so holding a key does not take p straight back.
	if p.travelCD > 0 {
		p.travelCD--
		if p.input.MoveX == 0 && p.input.MoveY == 0 && p.travelCD < TravelHold-TravelCooldown {
			p.travelCD = 0
		}
		return
	}
	if w.links == nil || p.boat != 0 {
		return
	}
	dest, ok := w.links[[2]int{int(math.Floor(p.X)), int(math.Floor(p.Y))}]
	if !ok {
		return
	}
	if c := w.creatureByID(p.mount); c != nil {
		c.X, c.Y = dest.x, dest.y
	}
	p.X, p.Y = dest.x, dest.y
	p.travelCD = TravelHold
	p.explored.Reveal(p.X, p.Y)
	if w.Underground(int(dest.y)) {
		p.notify(NoticeIntoCave, 0)
	} else {
		p.notify(NoticeOutOfCave, 0)
	}
}

// CaveLink is where walking onto tile (tx, ty) takes a player, if it is a
// cave mouth or a way up.
func (w *World) CaveLink(tx, ty int) (x, y float64, ok bool) {
	l, ok := w.links[[2]int{tx, ty}]
	return l.x, l.y, ok
}

// Maze tuning: the caves are a maze of winding passages on a grid of
// cells, with a hall here and there and a few loops.
const (
	mazeCell  = 6    // tiles per maze cell
	mazeHall  = 0.05 // share of cells opened out into a hall
	mazeLoops = 0.08 // share of extra openings between cells, making loops
	mazeRough = 0.12 // share of rock beside a passage worn away
)

// carveMaze digs the cave maze into rows top to top+rows.
func (w *World) carveMaze(seed uint32, top, rows int) {
	W := w.width
	r := rand.New(rand.NewPCG(uint64(seed), 0x3a2e))
	cw, ch := (W-6)/mazeCell, rows/mazeCell
	if cw < 2 || ch < 2 {
		return
	}
	// A cell's passage is the 2x2 tiles at its corner; links run east and
	// south from there.
	cellXY := func(i, j int) (int, int) { return 3 + i*mazeCell + 1, top + j*mazeCell + 1 }
	dig := func(x, y int) {
		if w.inBounds(x, y) && y >= top && y < top+rows && x > 1 && x < W-2 {
			w.tiles[y*W+x] = TileCave
		}
	}
	digBox := func(x0, y0, x1, y1 int) {
		for y := min(y0, y1); y <= max(y0, y1); y++ {
			for x := min(x0, x1); x <= max(x0, x1); x++ {
				dig(x, y)
			}
		}
	}
	link := func(i, j, ni, nj int) {
		x0, y0 := cellXY(i, j)
		x1, y1 := cellXY(ni, nj)
		digBox(x0, y0, x1+1, y1+1)
	}
	// The recursive backtracker: one long winding maze through every cell.
	seen := make([]bool, cw*ch)
	stack := [][2]int{{r.IntN(cw), r.IntN(ch)}}
	seen[stack[0][1]*cw+stack[0][0]] = true
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		var next [][2]int
		for _, d := range dirs {
			ni, nj := c[0]+d[0], c[1]+d[1]
			if ni >= 0 && nj >= 0 && ni < cw && nj < ch && !seen[nj*cw+ni] {
				next = append(next, [2]int{ni, nj})
			}
		}
		if len(next) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		n := next[r.IntN(len(next))]
		seen[n[1]*cw+n[0]] = true
		link(c[0], c[1], n[0], n[1])
		stack = append(stack, n)
	}
	// Loops, and halls.
	for j := 0; j < ch; j++ {
		for i := 0; i < cw; i++ {
			x, y := cellXY(i, j)
			digBox(x, y, x+1, y+1)
			if i+1 < cw && r.Float64() < mazeLoops {
				link(i, j, i+1, j)
			}
			if j+1 < ch && r.Float64() < mazeLoops {
				link(i, j, i, j+1)
			}
			if r.Float64() < mazeHall {
				hw, hh := 2+r.IntN(3), 2+r.IntN(2)
				for yy := y - hh; yy <= y+1+hh; yy++ {
					for xx := x - hw; xx <= x+1+hw; xx++ {
						dx, dy := float64(xx)-float64(x)-0.5, float64(yy)-float64(y)-0.5
						if (dx*dx)/float64((hw+1)*(hw+1))+(dy*dy)/float64((hh+1)*(hh+1)) <= 1 {
							dig(xx, yy)
						}
					}
				}
			}
		}
	}
	// Worn walls: rock beside a passage is eaten away here and there, so the
	// passages wind and bulge instead of running ruler-straight.
	var worn []int
	for y := top + 1; y < top+rows-1; y++ {
		for x := 2; x < W-2; x++ {
			i := y*W + x
			if w.tiles[i] != TileMountain {
				continue
			}
			open := 0
			for _, d := range dirs {
				if w.tiles[(y+d[1])*W+x+d[0]] == TileCave {
					open++
				}
			}
			if open > 0 && unit(hash2(int32(x), int32(y), seed+0x3a2f)) < mazeRough*float64(open) {
				worn = append(worn, i)
			}
		}
	}
	for _, i := range worn {
		w.tiles[i] = TileCave
	}
}
