package sim

import (
	"math"
	"math/rand/v2"
)

// The coasts of a StyleIsles world. Round every island the sea is shallow
// for a few tiles, wide off beaches and deltas, none under sea cliffs, where
// the deep water comes right in; warm seas have reefs further out; narrow
// straits between islands are shallow sandbanks, easy to swim and sail.
// Spits of sand close off lagoons, an atoll rings one, sea stacks stand off
// the cliffs, and one island is cut in two by a channel.

// Coast tuning.
const (
	coastSpeck   = 6  // smaller bits of land go under
	coastPocket  = 10 // smaller pockets of sea inside the land are filled in; bigger ones are lakes
	shelfMost    = 24 // tiles out from land the shallows are worked out to
	straitMax    = 18 // a gap between two islands this narrow is a strait, shallow
	straitShoal  = 5  // tiles either side of a strait's middle that are shallow
	reefNear     = 2  // tiles from land, the nearest a reef lies
	reefFar      = 13 // and the furthest
	stackChance  = 0.03
	spitChance   = 0.7 // a large island with beaches has a spit and its lagoon this often
	spitMinShare = 0.6 // of a spit, the share that must lie over open water
)

// shelfWidth is how far the shallows reach off island kind k, before noise.
func shelfWidth(s *isle) float64 {
	switch {
	case s == nil:
		return 1
	case s.role == roleAtoll:
		return 5
	case s.role == roleIslet:
		return 2
	}
	switch s.kind {
	case IslandMarsh:
		return 4
	case IslandSnow, IslandAsh:
		return 0.6
	case IslandDesert:
		return 1.8
	}
	return 2.2
}

// warmSea reports whether the sea off island s grows reefs.
func warmSea(s *isle) bool {
	if s == nil {
		return false
	}
	switch s.kind {
	case IslandSnow, IslandAsh, IslandDesert:
		return false
	}
	return true
}

// cleanCoast sinks specks of land and fills or floods the pockets of sea
// left inside the land: small ones become land, bigger ones lakes (in an
// atoll, its lagoon).
func (w *World) cleanCoast(im *isleMap) {
	W, H := w.width, w.height
	seen := make([]bool, len(w.tiles))
	var comp []int
	for start := range w.tiles {
		if seen[start] {
			continue
		}
		land := w.tiles[start] != TileWater
		comp = append(comp[:0], start)
		seen[start] = true
		edge := false
		for k := 0; k < len(comp); k++ {
			x, y := comp[k]%W, comp[k]/W
			if x == 0 || y == 0 || x == W-1 || y == H-1 {
				edge = true
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; !seen[j] && (w.tiles[j] != TileWater) == land {
					seen[j] = true
					comp = append(comp, j)
				}
			}
		}
		switch {
		case land && len(comp) < coastSpeck:
			for _, i := range comp {
				w.tiles[i] = TileWater
			}
		case !land && !edge && len(comp) < coastPocket:
			for _, i := range comp {
				w.tiles[i] = TileGrass
			}
		case !land && !edge:
			for _, i := range comp {
				if im.lagoon[i] {
					w.tiles[i], w.relief[i] = TileShallow, w.relief[i].withVariant(VarLagoon)
				} else {
					w.relief[i] = w.relief[i].withVariant(VarLake)
				}
			}
		}
	}
	for i, t := range w.tiles {
		if t == TileWater && im.lagoon[i] {
			w.tiles[i], w.relief[i] = TileShallow, w.relief[i].withVariant(VarLagoon)
		}
	}
}

// splitTwins cuts the biggest medium island in two with a channel of
// shallow water: twin islands across a strait.
func (w *World) splitTwins(im *isleMap, seed uint32) {
	best := -1
	for k, s := range im.isles {
		if s.role == roleMedium && (best < 0 || s.r > im.isles[best].r) {
			best = k
		}
	}
	if best < 0 || im.isles[best].r < 14 {
		return
	}
	s := &im.isles[best]
	rng := rand.New(rand.NewPCG(uint64(seed), 0x7a1))
	a := rng.Float64() * math.Pi
	l := s.reach()
	ux, uy := math.Cos(a), math.Sin(a)
	for t := -l; t <= l; t += 0.5 {
		off := math.Sin(t/l*math.Pi*1.5+rng.Float64()*0.2) * s.r * 0.12
		x, y := s.x+ux*t-uy*off, s.y+uy*t+ux*off
		wd := 1.6 + 0.8*fbm(x/5, y/5, seed+0x7a2, 2)
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				tx, ty := int(x)+dx, int(y)+dy
				if !w.inBounds(tx, ty) || math.Hypot(float64(tx)+0.5-x, float64(ty)+0.5-y) > wd {
					continue
				}
				i := ty*w.width + tx
				if w.tiles[i] != TileWater && im.owner[i] == int16(best+1) {
					w.tiles[i], w.relief[i] = TileShallow, w.relief[i].withVariant(VarShoal)
				}
			}
		}
	}
}

// laySpits lays spits of sand off some beaches, each closing off a lagoon
// of shallow water behind it, open at its far end.
func (w *World) laySpits(im *isleMap, seed uint32) {
	W := w.width
	rng := rand.New(rand.NewPCG(uint64(seed), 0x5917))
	coast := make([][]int, len(im.isles))
	for i, t := range w.tiles {
		if t == TileWater || im.owner[i] == 0 {
			continue
		}
		x, y := i%W, i/W
		if w.TileAt(x+1, y) == TileWater || w.TileAt(x-1, y) == TileWater || w.TileAt(x, y+1) == TileWater || w.TileAt(x, y-1) == TileWater {
			coast[im.owner[i]-1] = append(coast[im.owner[i]-1], i)
		}
	}
	for k := range im.isles {
		s := &im.isles[k]
		chance := 0.0
		switch {
		case s.role == roleHome:
			chance = 0.8
		case s.role == roleLarge && s.kind != IslandSnow && s.kind != IslandAsh:
			chance = spitChance
		case s.role == roleMedium && (s.kind == IslandMeadow || s.kind == IslandForest || s.kind == IslandMarsh):
			chance = 0.25
		}
		if rng.Float64() >= chance || len(coast[k]) == 0 {
			continue
		}
		for try := 0; try < 12; try++ {
			if w.laySpit(im, k, coast[k][rng.IntN(len(coast[k]))], rng) {
				break
			}
		}
	}
}

// laySpit tries to lay a spit of island k from coast tile p.
func (w *World) laySpit(im *isleMap, k, p int, rng *rand.Rand) bool {
	W := w.width
	s := &im.isles[k]
	px, py := p%W, p/W
	nx, ny := 0.0, 0.0 // out to sea
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if w.TileAt(px+dx, py+dy) == TileWater {
				nx, ny = nx+float64(dx), ny+float64(dy)
			}
		}
	}
	nl := math.Hypot(nx, ny)
	if nl < 8 {
		return false // a cove or a point: no straight beach to run along
	}
	nx, ny = nx/nl, ny/nl
	tx, ty := -ny, nx
	if rng.IntN(2) == 0 {
		tx, ty = -tx, -ty
	}
	L := clamp(s.r*0.4, 14, 50)
	D := clamp(s.r*0.05+3, 4, 8)
	off := func(t float64) float64 { return D*smoothstep(math.Min(1, t/(0.3*L))) + 0.8*math.Sin(t/6) }
	at := func(t, o float64) (int, int) {
		return int(math.Floor(float64(px) + 0.5 + tx*t + nx*o)), int(math.Floor(float64(py) + 0.5 + ty*t + ny*o))
	}
	// The spit must run over open water, clear of other islands.
	sea, all := 0, 0
	for t := 0.0; t <= L; t += 1 {
		x, y := at(t, off(t))
		if !w.inBounds(x, y) || x < isleEdge || y < isleEdge || x >= W-isleEdge || y >= w.height-isleEdge {
			return false
		}
		i := y*W + x
		if o := im.owner[i]; o != 0 && o != int16(k+1) && w.tiles[i] != TileWater {
			return false
		}
		all++
		if w.tiles[i] == TileWater {
			sea++
		}
	}
	if float64(sea) < spitMinShare*float64(all) {
		return false
	}
	for t := 0.0; t <= L; t += 0.5 {
		o := off(t)
		for q := 0.5; q < o-0.6; q += 0.5 { // the lagoon behind it
			x, y := at(t, q)
			if i := y*W + x; w.inBounds(x, y) && w.tiles[i] == TileWater {
				w.tiles[i], w.relief[i] = TileShallow, w.relief[i].withVariant(VarLagoon)
				im.owner[i], im.lagoon[i] = int16(k+1), true
			}
		}
		width := 1.4
		if t > 0.85*L {
			width = 0.6
		}
		for q := o - 0.5; q <= o+width; q += 0.5 { // the sand
			x, y := at(t, q)
			if i := y*W + x; w.inBounds(x, y) && w.tiles[i].Sea() {
				w.tiles[i], w.relief[i] = TileGrass, w.relief[i].withVariant(VarPlain)
				im.owner[i], im.h[i], im.val[i] = int16(k+1), 0, 0.01
			}
		}
	}
	return true
}

// fillShallows lays the shallows round the land, the reefs off warm shores
// and the sandbanks in the straits.
func (w *World) fillShallows(im *isleMap, seed uint32) {
	W, H := w.width, w.height
	// Each piece of land is labelled; the sea takes the label and the
	// distance of the nearest land, and whether that land is a cliff.
	land := make([]int32, len(w.tiles))
	for i := range land {
		land[i] = -1
	}
	nLand := int32(0)
	var q []int32
	for start := range w.tiles {
		if land[start] >= 0 || w.seaAt(start) {
			continue
		}
		land[start] = nLand
		q = append(q[:0], int32(start))
		for k := 0; k < len(q); k++ {
			x, y := int(q[k])%W, int(q[k])/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; land[j] < 0 && !w.seaAt(j) {
					land[j] = nLand
					q = append(q, int32(j))
				}
			}
		}
		nLand++
	}
	dist := make([]uint8, len(w.tiles))
	lab := make([]int32, len(w.tiles))
	cliff := make([]bool, len(w.tiles))
	near := make([]int32, len(w.tiles)) // the land tile it is nearest to
	q = q[:0]
	for i := range w.tiles {
		if w.seaAt(i) {
			dist[i], lab[i] = shelfMost, -1
		} else {
			lab[i], near[i] = land[i], int32(i)
			cliff[i] = w.tiles[i] == TileCliff
			q = append(q, int32(i))
		}
	}
	for k := 0; k < len(q); k++ {
		i := int(q[k])
		x, y := i%W, i/W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			if j := ny*W + nx; dist[j] > dist[i]+1 {
				dist[j], lab[j], cliff[j], near[j] = dist[i]+1, lab[i], cliff[i], near[i]
				q = append(q, int32(j))
			}
		}
	}
	// Straits: where the seas of two pieces of land meet, close enough.
	var mid []int32
	strait := make([]uint8, len(w.tiles))
	for i := range w.tiles {
		strait[i] = 255
		if !w.seaAt(i) || lab[i] < 0 {
			continue
		}
		x, y := i%W, i/W
		for _, d := range [2][2]int{{1, 0}, {0, 1}} {
			nx, ny := x+d[0], y+d[1]
			if nx >= W || ny >= H {
				continue
			}
			j := ny*W + nx
			if w.seaAt(j) && lab[j] >= 0 && lab[j] != lab[i] && int(dist[i])+int(dist[j]) < straitMax {
				strait[i], strait[j] = 0, 0
				mid = append(mid, int32(i), int32(j))
			}
		}
	}
	for k := 0; k < len(mid); k++ {
		i := int(mid[k])
		if strait[i] >= straitShoal {
			continue
		}
		x, y := i%W, i/W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			if j := ny*W + nx; w.seaAt(j) && strait[j] > strait[i]+1 {
				strait[j] = strait[i] + 1
				mid = append(mid, int32(j))
			}
		}
	}
	for i := range w.tiles {
		if w.tiles[i] != TileWater || w.lakeAt(i) || lab[i] < 0 {
			continue
		}
		x, y := float64(i%W)+0.5, float64(i/W)+0.5
		s := im.isleAt(int(near[i]))
		d := float64(dist[i])
		switch {
		case strait[i] <= straitShoal:
			w.tiles[i], w.relief[i] = TileShallow, w.relief[i].withVariant(VarShoal)
		case cliff[i] && d <= 1 && fbm(x/9, y/9, seed+0x5a1, 2) > 0.6:
			w.tiles[i] = TileShallow // rocks at a cliff's foot, now and then
		case !cliff[i] && d <= shelfWidth(s)+2.6*(fbm(x/30+3, y/30+1, seed+0x5a2, 3)-0.5)*2:
			w.tiles[i] = TileShallow
		case warmSea(s) && d >= reefNear && d <= reefFar && fbm(x/8+9, y/8+2, seed+0x5a3, 3) > 0.64-0.01*float64(reefFar-dist[i]):
			w.tiles[i], w.relief[i] = TileShallow, w.relief[i].withVariant(VarReef)
		}
	}
}

// standStacks raises sea stacks off the sea cliffs: pillars of rock alone
// in the water, a little shallow round their feet.
func (w *World) standStacks(im *isleMap, seed uint32) {
	W := w.width
	for i, t := range w.tiles {
		if t != TileCliff {
			continue
		}
		x, y := i%W, i/W
		if unit(hash2(int32(x), int32(y), seed+0x57a)) >= stackChance {
			continue
		}
		nx, ny := 0, 0
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if j := (y+d[1])*W + x + d[0]; w.inBounds(x+d[0], y+d[1]) && w.seaAt(j) {
				nx, ny = nx+d[0], ny+d[1]
			}
		}
		if nx == 0 && ny == 0 {
			continue
		}
		out := 2 + int(hash2(int32(x), int32(y), seed+0x57b)%3)
		sx, sy := x+nx*out, y+ny*out
		free := true
		for dy := -2; dy <= 2 && free; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if j := (sy+dy)*W + sx + dx; !w.inBounds(sx+dx, sy+dy) || !w.seaAt(j) {
					free = false
					break
				}
			}
		}
		if !free {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				j := (sy+dy)*W + sx + dx
				w.tiles[j] = TileShallow
			}
		}
		j := sy*W + sx
		w.tiles[j] = TileCliff
		w.relief[j] = w.relief[j].withLevel(1).withVariant(VarStack)
	}
}

// roomForHamlets widens every island big enough for a village (see
// islandtribes.go) that is too thin for even a hamlet, a sandbar of a
// thing: a round of beach is laid out from its broadest point.
func (w *World) roomForHamlets() {
	W := w.width
	const R = hamletRadius + 1 // the clearing and the huts round it
	r := float64(R) + 0.6
	fits := func(cx, cy int) bool {
		for y := cy - R; y <= cy+R; y++ {
			for x := cx - R; x <= cx+R; x++ {
				if (x-cx)*(x-cx)+(y-cy)*(y-cy) > R*R {
					continue
				}
				if t := w.TileAt(x, y); !t.Walkable() || t == TileLava {
					return false
				}
			}
		}
		return true
	}
	for _, isl := range w.landMasses() {
		if len(isl) < villageIslandMin || isl[0]/W >= w.SurfaceHeight() {
			continue
		}
		room, best, most := false, -1, -1
		for _, i := range isl {
			x, y := i%W, i/W
			if !w.tiles[i].Walkable() {
				continue
			}
			if fits(x, y) {
				room = true
				break
			}
			// The broadest point: the most open ground round it.
			n := 0
			for dy := -3; dy <= 3; dy++ {
				for dx := -3; dx <= 3; dx++ {
					if w.TileAt(x+dx, y+dy).Walkable() {
						n++
					}
				}
			}
			if n > most {
				best, most = i, n
			}
		}
		if room || best < 0 {
			continue
		}
		bx, by := best%W, best/W
		for y := by - R - 1; y <= by+R+1; y++ {
			for x := bx - R - 1; x <= bx+R+1; x++ {
				if math.Hypot(float64(x-bx), float64(y-by)) > r || !w.inBounds(x, y) {
					continue
				}
				if i := y*W + x; w.seaAt(i) {
					w.tiles[i], w.relief[i] = TileSand, 0
				}
			}
		}
	}
}
