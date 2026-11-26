package sim

import (
	"math"
	"slices"
)

// The woods of a StyleWoods world. Since the trees were drawn at the
// player's scale (a broadleaf three to five times a player's height, a
// redwood six, a giant seven), a tree on a third or half of the woodland
// tiles made a wall of leaves that hid the ground and the player alike.
// Here every tree keeps the distance from the next that its crown needs:
//
//   - Trees are put down one by one in a random order, each only where no
//     tree stands closer than the spacing of the two (a Poisson-disc): the
//     grown trees of the canopy first, then young ones in the gaps.
//   - How many take is the thickness of the ground: thin at the edge of a
//     wood and thicker inside, thickest in the dense groves, with glades
//     opening here and there. The densest jungle is still not a wall.
//   - Game trails wind through every larger wood (TilePath).
//   - The floor between the trunks has its undergrowth: bushes, tall grass,
//     fallen branches, flowers, and mushrooms on rotting logs.
//
// It is all decided by hashes of the tiles and noise of the seed, so a world
// comes out the same every time. Worlds of older styles keep their woods
// as they were made; a save from before version 20 has them thinned once
// on loading (see thinwoods.go).

// TreeLook is which drawing of a tree the client shows for an EntityTree. It
// follows from the tile and its ground alone, so the woods can be spaced by
// the crowns the client will draw.
type TreeLook uint8

const (
	LookBroadleaf TreeLook = iota // leafy trees, many hung with vines
	LookPine                      // the conifers of the woods
	LookRedwood                   // giant redwoods: narrow, very tall
	LookAraucaria                 // monkey-puzzles: wide and tall
	LookTreeFern
	LookCycad
	LookGinkgo
	LookSnowPine // frosted pines on the snow
	LookSwamp    // bald cypresses of the swamp; a dead snag now and then
	NumTreeLooks
)

// TreeLookAt is the look of a tree on tile (tx, ty) of ground t:
// prehistoric trees by terrain, giant redwoods and monkey-puzzles in the
// woods, cycads and ginkgos in the open, tree ferns in both.
func TreeLookAt(tx, ty int, t Tile) TreeLook {
	switch t {
	case TileSnow:
		return LookSnowPine
	case TileMarsh:
		return LookSwamp
	}
	h := HashTile(tx, ty)
	forest := t == TileForest
	switch pick := (h >> 20) % 20; {
	case forest && pick < 3:
		return LookRedwood
	case forest && pick < 5, t == TileGrass && pick < 1:
		return LookAraucaria
	case forest && pick < 7, t == TileGrass && pick < 4:
		return LookTreeFern
	case t == TileGrass && pick < 8:
		return LookCycad
	case t == TileGrass && pick < 11:
		return LookGinkgo
	}
	if forest && h%5 < 1 {
		return LookPine
	}
	return LookBroadleaf
}

// SwampSnag reports whether a swamp tree on (tx, ty) is drawn as a dead snag
// rather than a cypress.
func SwampSnag(tx, ty int) bool { return HashTile(tx, ty)%4 == 0 }

// crown is what a tree's drawing covers on a 1080p screen, measured from the
// art: its width and height in tiles (24 and 18 art pixels), and the area of
// its opaque pixels in tiles of the screen (the trunk with it). spacing is
// the least distance in tiles from its trunk to the next tree's that its
// crown needs; two trees keep the mean of their two.
type crown struct{ w, h, area, spacing float64 }

// crowns are those of each look, young (small) and grown (big). The wide
// and the tall crowns need 3 to 4 tiles, the slim conifers about 2, tree
// ferns and cycads less.
var crowns = [NumTreeLooks][2]crown{
	LookBroadleaf: {{2.6, 5.3, 7.0, 2.4}, {3.8, 7.6, 14.3, 3.5}},
	LookPine:      {{2.0, 5.5, 5.6, 1.7}, {2.9, 7.9, 11.2, 2.2}},
	LookRedwood:   {{1.4, 8.0, 5.9, 2.4}, {2.0, 11.3, 11.9, 3.2}},
	LookAraucaria: {{4.2, 6.7, 9.1, 3.0}, {6.1, 9.5, 18.6, 4.0}},
	LookTreeFern:  {{1.9, 3.6, 2.4, 1.2}, {3.1, 5.3, 5.1, 1.6}},
	LookCycad:     {{1.3, 2.2, 1.2, 1.0}, {1.9, 3.2, 2.5, 1.2}},
	LookGinkgo:    {{2.4, 5.5, 6.8, 2.2}, {3.5, 7.8, 13.2, 3.2}},
	LookSnowPine:  {{2.0, 5.5, 5.6, 1.7}, {2.9, 7.9, 11.2, 2.2}},
	LookSwamp:     {{2.5, 6.2, 7.4, 1.8}, {3.7, 9.0, 15.0, 2.4}},
}

// The crowns of the other things that stand as tall as trees.
var (
	snagCrown   = [2]crown{{1.0, 3.9, 1.5, 1.0}, {1.3, 5.4, 2.5, 1.2}}
	fruitCrown  = crown{2.6, 5.0, 6.0, 2.0} // figs and ginkgos bearing nuts
	palmCrown   = crown{4.0, 7.0, 8.2, 1.8}
	bananaCrown = crown{2.4, 4.1, 4.7, 1.6}
	giantCrown  = crown{7.1, 12.9, 46, 5.5}
)

// maxSpacing is the largest spacing of any crown, giants aside.
const maxSpacing = 4.0

// crownOf is the crown of a thing of kind k standing on (tx, ty), if it has one.
func (w *World) crownOf(k EntityKind, tx, ty int, big bool) (crown, bool) {
	b := 0
	if big {
		b = 1
	}
	switch k {
	case EntityTree:
		l := TreeLookAt(tx, ty, w.TileAt(tx, ty))
		if l == LookSwamp && SwampSnag(tx, ty) {
			return snagCrown[b], true
		}
		return crowns[l][b], true
	case EntityFigTree, EntityGinkgoTree:
		return fruitCrown, true
	case EntityPalm:
		return palmCrown, true
	case EntityBananaTree:
		return bananaCrown, true
	case EntityGiantTree:
		return giantCrown, true
	}
	return crown{}, false
}

// Woods tuning.
const (
	woodsEdgeReach = 5.0  // tiles in from a wood's edge before it is at its thickest
	woodsEdgeThin  = 0.35 // how thick a wood is at its very edge, of its thickness inside
	woodsYoung     = 0.75 // young trees take this often where grown ones would
	gladeScale     = 8.0  // tiles to a unit of the glades' noise
	gladeLevel     = 0.69 // the noise above this is a glade...
	groveLevel     = 0.29 // ...and below this a dense grove
	gladeDeep      = 3.0  // tiles in from the edge a glade or a grove needs
	groveCloser    = 0.85 // trees of a grove stand this much closer
	denseCloser    = 0.82 // and of the thick woods' cores
	trailScale     = 22.0 // tiles to a unit of the trails' noise
	trailWidth     = 0.017
	trailMinWood   = 160 // woods of this many tiles or more have a trail
	trailClear     = 1   // tiles either side of a trail where no tree stands
)

// woodsTake is how often a tree takes on free ground t of variant v, before
// the spacing has its say.
func woodsTake(t Tile, v GroundVariant) float64 {
	switch t {
	case TileForest:
		switch v {
		case VarDenseWood:
			return 0.8
		case VarOpenWood:
			return 0.12
		}
		return 0.16
	case TileGrass:
		switch v {
		case VarMeadow, VarBoulders:
			return 0.004
		case VarClearing:
			return 0.012
		}
		return 0.025
	case TileSnow:
		switch v {
		case VarDenseWood:
			return 0.3
		case VarBoulders:
			return 0.01
		}
		return 0.05
	case TileMarsh:
		if v == VarMudflat {
			return 0
		}
		return 0.06
	}
	return 0
}

// woodland reports whether tile i is woods: forest, or the snow's pinewoods.
func (w *World) woodland(i int) bool {
	t := w.tiles[i]
	return t == TileForest || t == TileSnow && w.relief != nil && w.relief[i].Variant() == VarDenseWood
}

// woodsDepth is, for every tile, how far in from the edge of its wood it
// lies, in tiles (0 outside the woods), by a chamfer distance.
func (w *World) woodsDepth() []float32 {
	W, H := w.width, w.SurfaceHeight()
	d := make([]float32, W*H)
	const far = 1e6
	for i := range d {
		if w.woodland(i) {
			d[i] = far
		}
	}
	const diag = 1.4142
	at := func(x, y int) float32 {
		if x < 0 || y < 0 || x >= W || y >= H {
			return 0
		}
		return d[y*W+x]
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if i := y*W + x; d[i] > 0 {
				d[i] = min(d[i], at(x-1, y)+1, at(x, y-1)+1, at(x-1, y-1)+diag, at(x+1, y-1)+diag)
			}
		}
	}
	for y := H - 1; y >= 0; y-- {
		for x := W - 1; x >= 0; x-- {
			if i := y*W + x; d[i] > 0 {
				d[i] = min(d[i], at(x+1, y)+1, at(x, y+1)+1, at(x+1, y+1)+diag, at(x-1, y+1)+diag)
			}
		}
	}
	return d
}

// woodsPatch is what a noise of the seed makes of woodland tile i, depth in:
// a glade (+1), a dense grove (-1) or neither (0).
func woodsPatch(seed uint32, i, W int, depth float32) int {
	if depth < gladeDeep {
		return 0
	}
	x, y := float64(i%W)+0.5, float64(i/W)+0.5
	switch n := fbm(x/gladeScale+41, y/gladeScale+13, seed+611, 2); {
	case n > gladeLevel:
		return 1
	case n < groveLevel:
		return -1
	}
	return 0
}

// carveTrails beats game trails through every larger wood: winding lines
// where a noise of the seed crosses its middle, joined up where they only
// touch at a corner, and a trail of its own across any big wood the noise
// missed.
func (w *World) carveTrails(seed uint32) {
	W, H := w.width, w.SurfaceHeight()
	comp := make([]int32, W*H)
	for i := range comp {
		comp[i] = -1
	}
	var woods [][]int
	for start := range comp {
		if comp[start] >= 0 || w.tiles[start] != TileForest {
			continue
		}
		n := int32(len(woods))
		comp[start] = n
		q := []int{start}
		for k := 0; k < len(q); k++ {
			x, y := q[k]%W, q[k]/W
			for _, dd := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+dd[0], y+dd[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; comp[j] < 0 && w.tiles[j] == TileForest {
					comp[j] = n
					q = append(q, j)
				}
			}
		}
		woods = append(woods, q)
	}
	trail := make([]bool, W*H)
	open := func(i int) bool { return w.tiles[i] == TileForest && !w.crossing(i) }
	for _, wood := range woods {
		if len(wood) < trailMinWood {
			continue
		}
		has := false
		for _, i := range wood {
			x, y := float64(i%W)+0.5, float64(i/W)+0.5
			if open(i) && math.Abs(fbm(x/trailScale+17, y/trailScale+71, seed+601, 3)-0.5) < trailWidth {
				trail[i], has = true, true
			}
		}
		if !has {
			w.trailAcross(wood, trail, seed)
		}
	}
	// A trail runs on from tile to tile by their sides, not their corners.
	for i, on := range trail {
		if !on {
			continue
		}
		x, y := i%W, i/W
		for _, dx := range [2]int{-1, 1} {
			nx, ny := x+dx, y+1
			if nx < 0 || nx >= W || ny >= H || !trail[ny*W+nx] || trail[y*W+nx] || trail[ny*W+x] {
				continue
			}
			switch {
			case open(ny*W + x):
				trail[ny*W+x] = true
			case open(y*W + nx):
				trail[y*W+nx] = true
			}
		}
	}
	for i, on := range trail {
		if on {
			w.tiles[i] = TilePath
			if w.relief != nil {
				w.relief[i] = w.relief[i].withVariant(VarPlain)
			}
		}
	}
}

// trailAcross marks a trail across wood, a patch of woodland the noise's
// trails missed: along its longer side through its middle, wandering.
func (w *World) trailAcross(wood []int, trail []bool, seed uint32) {
	W := w.width
	x0, y0, x1, y1 := W, w.height, 0, 0
	var sx, sy int
	for _, i := range wood {
		x, y := i%W, i/W
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
		sx, sy = sx+x, sy+y
	}
	cx, cy := float64(sx)/float64(len(wood)), float64(sy)/float64(len(wood))
	inWood := make(map[int]bool, len(wood))
	for _, i := range wood {
		inWood[i] = true
	}
	mark := func(x, y int) {
		if i := y*W + x; inWood[i] && !w.crossing(i) {
			trail[i] = true
		}
	}
	if x1-x0 >= y1-y0 {
		prev := -1
		for x := x0; x <= x1; x++ {
			y := int(math.Round(cy + 6*(fbm(float64(x)/9, cy/9, seed+602, 2)-0.5)))
			mark(x, y)
			for ; prev >= 0 && prev != y; prev += isign(y - prev) {
				mark(x, prev) // a step up or down stays joined
			}
			prev = y
		}
		return
	}
	prev := -1
	for y := y0; y <= y1; y++ {
		x := int(math.Round(cx + 6*(fbm(cx/9, float64(y)/9, seed+603, 2)-0.5)))
		mark(x, y)
		for ; prev >= 0 && prev != x; prev += isign(x - prev) {
			mark(prev, y)
		}
		prev = x
	}
}

func isign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}

// plantWoods grows the trees of a StyleWoods world on the free ground.
func (w *World) plantWoods(seed uint32) {
	W, H := w.width, w.SurfaceHeight()
	depth := w.woodsDepth()
	nearTrail := make([]bool, W*H)
	for i, t := range w.tiles[:W*H] {
		if t != TilePath {
			continue
		}
		x, y := i%W, i/W
		for ny := y - trailClear; ny <= y+trailClear; ny++ {
			for nx := x - trailClear; nx <= x+trailClear; nx++ {
				if nx >= 0 && ny >= 0 && nx < W && ny < H {
					nearTrail[ny*W+nx] = true
				}
			}
		}
	}

	type cand struct {
		i     int32
		pri   uint32
		take  float32 // how often a tree takes here
		close float32 // how much closer than their spacing trees stand here
	}
	var cands []cand
	for i := range W * H {
		x, y := i%W, i/W
		if !w.canPlace(x, y) || w.crossing(i) || nearTrail[i] {
			continue
		}
		var v GroundVariant
		if w.relief != nil {
			v = w.relief[i].Variant()
		}
		take := woodsTake(w.tiles[i], v)
		if take == 0 {
			continue
		}
		closer := 1.0
		if w.woodland(i) {
			// Thin at the edge, thicker inside.
			e := smoothstep(math.Min(float64(depth[i])/woodsEdgeReach, 1))
			take *= woodsEdgeThin + (1-woodsEdgeThin)*e
			closer = 1.3 - 0.15*e
			if v == VarDenseWood {
				closer *= denseCloser
			}
			switch woodsPatch(seed, i, W, depth[i]) {
			case 1:
				take *= 0.06 // a glade: a tree or two
			case -1:
				take, closer = math.Max(take, 0.6), groveCloser
			}
		}
		cands = append(cands, cand{int32(i), hash2(int32(x), int32(y), seed^0x30d5), float32(take), float32(closer)})
	}
	slices.SortFunc(cands, func(a, b cand) int {
		if a.pri != b.pri {
			if a.pri < b.pri {
				return -1
			}
			return 1
		}
		return int(a.i - b.i)
	})

	spacing := make([]float32, W*H) // of the tree on each tile; 0 for none
	fits := func(x, y int, s, closer float64) bool {
		r := int(math.Ceil((s + maxSpacing) / 2 * closer))
		for ny := max(y-r, 0); ny <= min(y+r, H-1); ny++ {
			for nx := max(x-r, 0); nx <= min(x+r, W-1); nx++ {
				o := spacing[ny*W+nx]
				if o == 0 {
					continue
				}
				d := (s + float64(o)) / 2 * closer
				if dx, dy := float64(nx-x), float64(ny-y); dx*dx+dy*dy < d*d {
					return false
				}
			}
		}
		return true
	}
	// The grown trees of the canopy first, then the young in the gaps.
	for pass := range 2 {
		for _, c := range cands {
			i := int(c.i)
			x, y := i%W, i/W
			if spacing[i] != 0 {
				continue
			}
			big := pass == 0
			take := float64(c.take)
			if big {
				if unit(hash2(int32(x), int32(y), seed^0xb16)) >= bigShare(EntityTree, w.tiles[i]) {
					continue // a young tree, if any
				}
			} else {
				take *= woodsYoung
			}
			if unit(hash2(int32(x), int32(y), seed^0x7a4e+uint32(pass))) >= take {
				continue
			}
			cr, _ := w.crownOf(EntityTree, x, y, big)
			if !fits(x, y, cr.spacing, float64(c.close)) {
				continue
			}
			spacing[i] = float32(cr.spacing)
			e := w.addEntity(EntityTree, x, y)
			if big {
				e.Big = true
				e.HP = maxHP(EntityTree, true)
			}
		}
	}
}

// crowded reports whether a tall thing of crown c on (tx, ty) would stand
// closer to a tree than their spacing allows.
func (w *World) crowded(tx, ty int, c crown) bool {
	r := int(math.Ceil((c.spacing + giantCrown.spacing) / 2))
	for ny := ty - r; ny <= ty+r; ny++ {
		for nx := tx - r; nx <= tx+r; nx++ {
			e := w.entityAt(nx, ny)
			if e == nil || nx == tx && ny == ty {
				continue
			}
			o, ok := w.crownOf(e.Kind, nx, ny, e.Big)
			if !ok {
				continue
			}
			d := (c.spacing + o.spacing) / 2
			if dx, dy := float64(nx-tx), float64(ny-ty); dx*dx+dy*dy < d*d {
				return true
			}
		}
	}
	return false
}

// crowdedFruit reports whether a fruit tree of kind k on (tx, ty) would stand
// too close to another tree, in a world whose trees are spaced.
func (w *World) crowdedFruit(k EntityKind, tx, ty int) bool {
	if w.style < StyleWoods {
		return false
	}
	c, _ := w.crownOf(k, tx, ty, false)
	return w.crowded(tx, ty, c)
}

// clearUnderGiants takes away the trees that would stand in a giant's
// crown: the old forest's giants shade out what grows close by.
func (w *World) clearUnderGiants() {
	for _, g := range w.sortedEntities() {
		if g.Kind != EntityGiantTree {
			continue
		}
		r := int(math.Ceil((giantCrown.spacing + maxSpacing) / 2))
		for ny := g.TY - r; ny <= g.TY+r; ny++ {
			for nx := g.TX - r; nx <= g.TX+r; nx++ {
				e := w.entityAt(nx, ny)
				if e == nil || e.Kind == EntityGiantTree {
					continue
				}
				o, ok := w.crownOf(e.Kind, nx, ny, e.Big)
				if !ok {
					continue
				}
				d := (giantCrown.spacing + o.spacing) / 2
				if dx, dy := float64(nx-g.TX), float64(ny-g.TY); dx*dx+dy*dy < d*d {
					w.removeEntity(e)
				}
			}
		}
	}
}

// Undergrowth tuning: the chance of each thing on a free tile of the woods.
type undergrowth struct{ tuft, bush, twig, flower, mushroom float64 }

var (
	underEdge  = undergrowth{tuft: 0.035, bush: 0.012, twig: 0.008, flower: 0.012}                 // the bright edge of a wood
	underWood  = undergrowth{tuft: 0.012, bush: 0.008, twig: 0.018, flower: 0.006, mushroom: 0.01} // under the trees
	underGlade = undergrowth{tuft: 0.05, bush: 0.025, twig: 0.006, flower: 0.07}                   // a sunny glade
	underGrove = undergrowth{tuft: 0.004, bush: 0.004, twig: 0.022, mushroom: 0.022}               // dark: branches, logs with mushrooms
)

// fillUndergrowth gives the floor of the woods its undergrowth, so the
// ground between the trunks is not bare.
func (w *World) fillUndergrowth(seed uint32) {
	W, H := w.width, w.SurfaceHeight()
	depth := w.woodsDepth()
	sx, sy := int(math.Floor(w.spawnX)), int(math.Floor(w.spawnY))
	for i := range W * H {
		x, y := i%W, i/W
		if w.tiles[i] != TileForest || !w.canPlace(x, y) || w.crossing(i) || max(abs(x-sx), abs(y-sy)) <= clearSpawnRadius {
			continue
		}
		u := underWood
		switch {
		case depth[i] < 2:
			u = underEdge
		case woodsPatch(seed, i, W, depth[i]) == 1:
			u = underGlade
		case woodsPatch(seed, i, W, depth[i]) == -1:
			u = underGrove
		}
		r := unit(hash2(int32(x), int32(y), seed^0x0dd5))
		kind := pick(r, chance{EntityGrassTuft, u.tuft}, chance{EntityBerryBush, u.bush}, chance{EntityTwig, u.twig},
			chance{EntityFlower, u.flower}, chance{EntityMushroom, u.mushroom})
		if kind == EntityNone {
			continue
		}
		e := w.addEntity(kind, x, y)
		if kind == EntityFlower {
			// Flowers grow in drifts of one colour.
			e.Var = uint8(int(fbm(float64(x)/14+3, float64(y)/14+7, seed+72, 2)*float64(NumFlowers)*1.6) % int(NumFlowers))
		}
	}
}
