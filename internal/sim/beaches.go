package sim

import (
	"math"
)

// The beaches of a StyleBeaches world. In the worlds before it the sand
// between the grass and the sea was a strip one to three tiles deep, a thin
// line at the scale the game is drawn at. Here a sandy coast is a proper
// beach, about five to nine tiles from the water to the grass: wider where
// the coast faces open water and round the home island's bay, narrower in
// coves, and changing slowly along the shore, never one even band. The sand
// takes the land behind the waterline, so the islands keep their shapes,
// and the ground under it is levelled, so a beach does not end at a cliff.
// Cliff coasts keep no sand under them, the beach narrowing as it comes up
// to one; beaches are narrower where mangroves grow, marsh shores stay
// muddy and narrow, snow and ash shores keep their own narrow edge. The
// widest beaches have dunes at their back with dune grass and palms, and
// driftwood lies along the tide line (see scatterIsles).

// Beach tuning, in tiles.
const (
	beachMean    = 8.0  // a beach's depth on a straight, ordinary coast
	beachAlong   = 4.2  // how much it changes along the coast, either way at most
	beachExpose  = 10.0 // more per unit of open water round it beyond a straight coast's
	beachMost    = 12.5
	beachLeast   = 3.0 // a sandy coast's narrowest, in a cove
	beachHomeBay = 3.5 // more round the home island's bay
	beachBigBay  = 1.5 // and round the great bay of a crescent island
	beachReach   = 16  // tiles from the sea worked out to
	beachExposeR = 18  // tiles either way of the square the open water is measured in
	beachLevel   = 2   // tiles of level ground kept behind a beach before the land may rise
	beachDunes   = 7.5 // a beach at least this deep has dunes at its back
	beachTaper   = 12  // tiles of shore over which a beach narrows to nothing at a cliff coast
)

// beachField is, for every land tile of a StyleBeaches world, how far it
// lies from the sea and how deep the beach is there.
type beachField struct {
	ed []float32 // the straight-line distance to the nearest sea, in tiles (1 beside it)
	bw []float32 // the beach's depth here; 0 where the coast has no beach
}

// beachSpot is one tile's entry of a beachField; ok is false in worlds
// without one, which keep the narrow beaches of old.
type beachSpot struct {
	ed, bw float64
	ok     bool
}

// at is tile i's spot.
func (b *beachField) at(i int) beachSpot {
	if b == nil {
		return beachSpot{}
	}
	return beachSpot{float64(b.ed[i]), float64(b.bw[i]), true}
}

// level reports whether tile i lies on or just behind a beach, where the
// ground is kept at the shore's level.
func (b *beachField) level(i int) bool {
	return b != nil && b.bw[i] > 0 && b.ed[i] <= b.bw[i]+beachLevel
}

// beachFactor is how deep island s's beaches are against an ordinary one.
func beachFactor(s *isle) float64 {
	if s.role == roleAtoll || s.role == roleIslet {
		return 0.75
	}
	switch s.kind {
	case IslandSnow, IslandMarsh:
		return 0.35 // a snowy edge; mud flats and mangroves
	case IslandAsh:
		return 0.6 // black sand
	case IslandDesert:
		return 0.9
	}
	if s.role == roleHome {
		return 1.1
	}
	return 1
}

// seaOffsets is, for every tile, the step (dx, dy) to the nearest tile of
// the sea, found in two sweeps over the map (8SSEDT); the sea has (0, 0).
// The map's edge counts as sea.
func (w *World) seaOffsets() (dx, dy []int16) {
	W, H := w.width, w.height
	const far = 4000
	dx, dy = make([]int16, W*H), make([]int16, W*H)
	for i := range dx {
		if !w.seaAt(i) {
			dx[i], dy[i] = far, far
		}
	}
	d2 := func(x, y int) int { return x*x + y*y }
	try := func(i, x, y, ox, oy int) {
		var qx, qy int
		if x < 0 || y < 0 || x >= W || y >= H {
			qx, qy = ox, oy // off the map: sea
		} else {
			j := y*W + x
			qx, qy = int(dx[j])+ox, int(dy[j])+oy
		}
		if d2(qx, qy) < d2(int(dx[i]), int(dy[i])) {
			dx[i], dy[i] = int16(qx), int16(qy)
		}
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			try(i, x-1, y, -1, 0)
			try(i, x, y-1, 0, -1)
			try(i, x-1, y-1, -1, -1)
			try(i, x+1, y-1, 1, -1)
		}
		for x := W - 1; x >= 0; x-- {
			try(y*W+x, x+1, y, 1, 0)
		}
	}
	for y := H - 1; y >= 0; y-- {
		for x := W - 1; x >= 0; x-- {
			i := y*W + x
			try(i, x+1, y, 1, 0)
			try(i, x, y+1, 0, 1)
			try(i, x-1, y+1, -1, 1)
			try(i, x+1, y+1, 1, 1)
		}
		for x := 0; x < W; x++ {
			try(y*W+x, x-1, y, -1, 0)
		}
	}
	return dx, dy
}

// beachWidths works out the beachField of a StyleBeaches world, once its
// coasts are laid (spits and all) and before its levels and ground.
func (w *World) beachWidths(im *isleMap, seed uint32) *beachField {
	W, H := w.width, w.height
	b := &beachField{ed: make([]float32, W*H), bw: make([]float32, W*H)}
	ox, oy := w.seaOffsets()
	deepest := make([]float32, len(im.isles)+1) // the furthest any of each island's land lies from the sea
	for i := range b.ed {
		if w.seaAt(i) {
			continue
		}
		d := float32(math.Hypot(float64(ox[i]), float64(oy[i])))
		b.ed[i] = d
		if o := im.owner[i]; d > deepest[o] {
			deepest[o] = d
		}
	}
	// The open water round each tile, from a table of sums of the sea.
	sum := make([]int32, (W+1)*(H+1))
	for y := 0; y < H; y++ {
		row := int32(0)
		for x := 0; x < W; x++ {
			if w.seaAt(y*W + x) {
				row++
			}
			sum[(y+1)*(W+1)+x+1] = sum[y*(W+1)+x+1] + row
		}
	}
	const R = beachExposeR
	open := func(x, y int) float64 {
		x0, y0, x1, y1 := max(x-R, 0), max(y-R, 0), min(x+R+1, W), min(y+R+1, H)
		in := sum[y1*(W+1)+x1] - sum[y0*(W+1)+x1] - sum[y1*(W+1)+x0] + sum[y0*(W+1)+x0]
		all := (2*R + 1) * (2*R + 1)
		off := all - (x1-x0)*(y1-y0) // off the map: sea
		return float64(int(in)+off) / float64(all)
	}
	for i, d := range b.ed {
		if d == 0 || d > beachReach {
			continue
		}
		s := im.isleAt(i)
		if s == nil {
			continue
		}
		x, y := i%W, i/W
		fx, fy := float64(x)+0.5, float64(y)+0.5
		// Along the coast, slowly wider and narrower.
		bw := beachMean + beachAlong*(fbm(fx/30+3.3, fy/30+1.7, seed+0xbea1, 3)-0.5)*2
		// Open water: a straight coast has about half the square round it
		// sea, whatever the distance in (allowed for); a point or an outer
		// coast more, a cove less.
		expose := open(x, y) + float64(d)/float64(2*R+1) - 0.5
		bw += clamp(expose*beachExpose, -2.5, 3)
		bw += s.bayBeach(fx, fy)
		bw *= beachFactor(s)
		// Narrower where mangroves grow on the shore (see scatterFlora),
		// and to nothing at a cliff coast.
		sx, sy := float64(x+int(ox[i]))+0.5, float64(y+int(oy[i]))+0.5
		mangrove := fbm(sx/11+float64(501%17), sy/11+float64(501%29), seed+501, 2)
		bw *= 1 - 0.55*smoothstep(clamp((mangrove-0.6)/0.06, 0, 1))
		bw = max(bw, beachLeast*beachFactor(s))
		if _, coast := cliffShare(s); coast > 0 {
			bw *= smoothstep(clamp(s.cliffMargin(fx, fy, coast)/beachTaper, 0, 1))
		}
		// A small island is not all beach.
		bw = min(bw, beachMost, 0.5*float64(deepest[im.owner[i]]))
		b.bw[i] = float32(bw)
	}
	return b
}

// bayBeach is how much deeper the beach is at (x, y) for lying round one
// of island s's bays: the home island's, or a crescent's great bay.
func (s *isle) bayBeach(x, y float64) float64 {
	if len(s.bays) == 0 {
		return 0
	}
	// Warped as the island's shape is (see field), so the bay is where its
	// shore is.
	ws := s.r * 0.55
	x += (fbm(x/ws+3.1, y/ws, s.seed, 3) - 0.5) * s.r * 0.36
	y += (fbm(x/ws, y/ws+7.7, s.seed+1, 3) - 0.5) * s.r * 0.36
	more := 0.0
	for _, b := range s.bays {
		if b.x0 != b.x1 || b.y0 != b.y1 {
			continue // a fjord
		}
		add := 0.0
		switch {
		case s.role == roleHome:
			add = beachHomeBay
		case b.r >= s.r*0.42:
			add = beachBigBay
		default:
			continue
		}
		bd := math.Hypot(x-b.x0, y-b.y0)
		k := 1 - smoothstep(clamp((bd-b.r)/(0.6*b.r+8), 0, 1))
		more = math.Max(more, add*k)
	}
	return more
}

// cliffMargin is how far round island s's shore (x, y) lies from its cliff
// coasts (see cliffCoast), in tiles along the shore: 0 on one.
func (s *isle) cliffMargin(x, y, share float64) float64 {
	a := math.Atan2(y-s.y, x-s.x)
	c1 := float64(s.seed%6283) / 1000
	c2 := c1 + math.Pi*(0.7+0.6*float64(s.seed>>16%1000)/1000)
	half := share * math.Pi / 2 * (1 + 0.3*(fbm(math.Cos(a)*2+5, math.Sin(a)*2, s.seed+0xc1f, 2)-0.5))
	m := math.Inf(1)
	for _, c := range []float64{c1, c2} {
		m = math.Min(m, math.Abs(math.Remainder(a-c, 2*math.Pi))-half)
	}
	return math.Max(m, 0) * math.Hypot(x-s.x, y-s.y)
}

// beachThing is what lies on beach tile (tx, ty) of variant v, d tiles from
// the sea, in a world made since StyleBeaches: clam beds at the water's
// edge, driftwood (twigs, drawn bleached on the sand) and more clams along
// the tide line, dune grass at the back.
func (w *World) beachThing(r float64, tx, ty int, v GroundVariant, d int) EntityKind {
	switch {
	case v == VarDunes:
		return pick(r, chance{EntityGrassTuft, 0.08}, chance{EntityTwig, 0.003})
	case w.seaside(tx, ty):
		return pick(r, chance{EntityClam, clamShare}, chance{EntityTwig, 0.006})
	case d <= 3:
		return pick(r, chance{EntityTwig, 0.014}, chance{EntityClam, 0.012})
	}
	return pick(r, chance{EntityGrassTuft, 0.02}, chance{EntityTwig, 0.004})
}

// beachPalms is the share of beach tiles of variant v, d tiles from the
// sea, that grow a coconut palm: few by the water, more at the back of the
// beach and in the dunes.
func beachPalms(d int, v GroundVariant) float64 {
	switch {
	case v == VarDunes:
		return 0.014
	case d <= 2:
		return 0.002
	}
	return 0.01
}

// BeachDepth is how deep the beach is at one tile of sand on the shore:
// how many tiles of sand there are straight inland from it, away from the
// water.
type BeachDepth struct{ X, Y, N int }

// BeachDepths measures the beaches of the world, for tuning and tests:
// the depth at every tile of sand on the shore (sand beside the sea).
// shore is how many tiles of land of any kind lie on the shore.
func (w *World) BeachDepths() (depths []BeachDepth, shore int) {
	W, H := w.width, w.SurfaceHeight()
	sea := func(x, y int) bool {
		return x < 0 || y < 0 || x >= W || y >= H || w.tiles[y*W+x].Sea() && !(w.relief != nil && w.lakeAt(y*W+x))
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if sea(x, y) || !w.tiles[y*W+x].Walkable() && w.tiles[y*W+x] != TileCliff {
				continue
			}
			nx, ny, by := 0, 0, false
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if sea(x+dx, y+dy) {
						nx, ny = nx+dx, ny+dy
						if max(abs(dx), abs(dy)) == 1 && (dx == 0 || dy == 0) {
							by = true
						}
					}
				}
			}
			if !by {
				continue
			}
			shore++
			if w.tiles[y*W+x] != TileSand || nx == 0 && ny == 0 {
				continue
			}
			// Straight inland, away from the water.
			l := math.Hypot(float64(nx), float64(ny))
			ux, uy := -float64(nx)/l, -float64(ny)/l
			n := 0
			for k := 0; k < 40; k++ {
				tx, ty := int(math.Floor(float64(x)+0.5+ux*float64(k))), int(math.Floor(float64(y)+0.5+uy*float64(k)))
				if tx < 0 || ty < 0 || tx >= W || ty >= H || w.tiles[ty*W+tx] != TileSand {
					break
				}
				n++
			}
			depths = append(depths, BeachDepth{x, y, n})
		}
	}
	return depths, shore
}
