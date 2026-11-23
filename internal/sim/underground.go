package sim

import (
	"math"
	"math/rand/v2"
)

// The caves under the archipelago. An archipelago has no mountains: its
// caves lie underground, a dark maze of tunnels and halls in the rock. On
// the islands stand cave mouths, low arches of boulders over a hole going
// down; walking into one goes down to the caves, onto a patch of rubble
// under a shaft of daylight where a rope ladder hangs. Walking back onto it
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
	undergroundPad  = 8  // rows of rock under the caves
	caveMinHall     = 40 // smaller pockets are filled in
	mouthSpacing    = 14 // tiles at least between two mouths
	mouthIslandMin  = 160
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
	w.fauna.shallow = nil // sized for the old map; rebuilt on demand
	top := H + undergroundGap
	for ty := top; ty < top+H; ty++ {
		for tx := 3; tx < W-3; tx++ {
			x, y := float64(tx)+0.5, float64(ty)+0.5
			a := math.Abs(fbm(x/15+3, y/15+11, seed+501, 2)-0.5) < 0.06
			b := math.Abs(fbm(x/10+40, y/10+17, seed+502, 2)-0.5) < 0.036
			hall := fbm(x/9+9, y/9+3, seed+503, 2) > 0.66
			if a || b || hall {
				w.tiles[ty*W+tx] = TileCave
			}
		}
	}
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

// digCaveMouths sets cave mouths on the islands (not the home island), each
// with its way up in the caves below, joined to the halls.
func (w *World) digCaveMouths(rng *rand.Rand) {
	if w.surfaceH == 0 {
		return
	}
	W, H := w.width, w.surfaceH
	top := H + undergroundGap
	w.links = make(map[[2]int]caveLink)
	sx, sy := int(w.spawnX), int(w.spawnY)
	var placed [][2]int
	good := func(tx, ty int) bool {
		for y := ty - 1; y <= ty+1; y++ {
			for x := tx - 2; x <= tx+2; x++ {
				if y >= H || !w.TileAt(x, y).Walkable() || w.TileAt(x, y) == TilePath || w.TileAt(x, y) == TileLava {
					return false
				}
				if e := w.entityAt(x, y); e != nil && blocksPlayers(e.Kind) {
					return false // a hut, the pyramid
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
		if home || len(isl) < mouthIslandMin || isl[0]/W >= H {
			continue
		}
		want := min(mouthsPerIsland, 1+len(isl)/mouthPerTiles)
		spots := append([]int(nil), isl...)
		rng.Shuffle(len(spots), func(a, b int) { spots[a], spots[b] = spots[b], spots[a] })
		for _, i := range spots {
			if want == 0 {
				break
			}
			tx, ty := i%W, i/W
			if !good(tx, ty) {
				continue
			}
			w.placeMouth(tx, ty, tx, top+ty)
			placed = append(placed, [2]int{tx, ty})
			want--
		}
	}
}

// placeMouth puts a cave mouth on (tx, ty) and its way up at (ux, uy) in
// the caves, digging out a chamber there joined to the nearest hall.
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
