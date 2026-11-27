package sim

import "math"

// The wilds: where the animals live, in worlds made since StyleWilds.
//
// Before, every kind appeared wherever its ground was, the same anywhere in
// the world. Now each island has its own animals, by its kind and by how far
// it lies from home (its ring), and every island of some size has a few herds:
//
//   - The home island (ring 0) is kind: rabbits, Thescelosaurus and
//     Ornithomimus, a couple of groups of them within sight of the spawn,
//     and a few small herds of three-horns and plate-backs towards its far
//     side. Nothing hunts there by day (the night's Troodon still come; see
//     nightstalker.go), no snake has its den there, no crocodile lurks in
//     its fens and no sea reptile along its shore.
//   - The near islands (ring 1: the meadows, the jungle, the delta, the
//     monkeys' island) have their grazers and the lesser hunters: packs of
//     Acheroraptor, Anzu, flying reptiles on the rocks and the cliffs, the
//     marsh crocodiles; Dakotaraptor only deep in the jungle.
//   - The far islands (ring 2: the desert, the volcano, the snowy range, and
//     whatever lies out by the edge of a big world) are where the long-necks
//     roam in herds and the tyrant hunts, with Dakotaraptor packs and
//     Quetzalcoatlus; the bosses' lairs are there (bosses.go, wildlairs.go).
//
// Each kind of island has its own animals first (wildRules), others at a
// lower weight; within an island an animal keeps to its own ground (the
// zones of wildzones.go): grazers on the desert's grassy fringe, raptors in
// the volcano's jungle, sabre-tooths in the snow's pinewoods. Grazers
// outnumber hunters about five to one.
//
// What is planned is also what comes back: the world remembers where each
// group was put, and tops a kind up there (repopulate), so a cleared home
// island fills again with harmless animals, never with hunters.
//
// The sea's monsters keep to deep water well away from the home island, and
// there are far fewer of them; the coasts have more plesiosaurs and fish.
// The caves are left as they were (cavelife.go).

// wildRule is how a kind of animal is spread over the islands since
// StyleWilds.
type wildRule struct {
	kind CreatureKind
	herd [2]int // animals to a group, fewest and most
	// per is how many groups an island has for every wildsPerTiles tiles
	// of its land, before the weights: isle by the island's kind (indexed
	// by IslandKind), ring by its ring (home, near, far).
	per  float64
	isle [numIsleKinds]float64
	ring [3]float64
	// inland is how many tiles from the sea a group keeps at least (deep in
	// the woods); homeAway how far from the spawn it keeps on the home
	// island (on its far side).
	inland, homeAway int
	// each: every island of its kind (isle weight 1 or more) in a ring it
	// may live in, of wildsEachTiles tiles or more, has at least one group.
	each bool
}

// isles is a weight for each kind of island, in the order Meadow, Forest,
// Marsh, Snow, Desert, Ash, Monkey, Atoll.
func isles(meadow, forest, marsh, snow, desert, ash, monkey, atoll float64) [numIsleKinds]float64 {
	return [numIsleKinds]float64{meadow, forest, marsh, snow, desert, ash, monkey, atoll}
}

// Tuning of the wilds.
const (
	wildsPerTiles   = 10000 // the land a rule's per counts groups for
	wildsEachTiles  = 8000  // an island this big gets a group of each kind it is known for (wildRule.each)
	wildsMinTiles   = 1500  // an island this big has a few herds at least, whatever else it has
	wildsMinPer     = 8000  // one more herd at least for every this many tiles of it
	wildsNearSpawn  = 2     // groups of each of the gentlest kinds put within sight of the spawn
	wildsNearLo     = 16.0  // tiles from the spawn they are put at, least
	wildsNearHi     = 42.0  // and most
	wildsFarRing    = 0.85  // of half the world: an island further out is far, whatever its kind
	wildsSites      = 3     // places remembered for each group, where it may come back
	wildsHomeSea    = 14    // tiles of sea off the home island no sea reptile comes within
	wildsMosaAway   = 90    // and no Mosasaurus
	wildsKrakenAway = 110   // and no giant squid
)

// wildRules are the animals spread by island since StyleWilds. Grazers
// first, then the hunters. Kinds not here keep their old rules: the rabbits
// and snakes round their burrows and dens (homes.go), worms, crabs, fish,
// toads, monkeys, the sea's monsters (with their wilds tuning below), and
// the caves' life.
//
// One more grazer slots in as one more line among the grazers: its kind,
// its group size, its density and where it lives. Its own spawn rules in
// its creatureDef (spawnOn, away, the water and shore filters) still pick
// its ground within each island.
var wildRules = []wildRule{
	// Grazers.            herd   per    Meadow Forest Marsh Snow Desert Ash Monkey Atoll    home near far
	{kind: CreatureThesc, herd: [2]int{3, 4}, per: 1.8, isle: isles(1, 0.8, 0.6, 0, 0.2, 0.2, 0.6, 0.3), ring: [3]float64{1.2, 1, 0.6}, each: true},
	{kind: CreatureOrnitho, herd: [2]int{4, 6}, per: 1.0, isle: isles(1, 0.2, 0.3, 0, 1.4, 0.3, 0.1, 0.6), ring: [3]float64{1, 0.8, 1}, each: true},
	{kind: CreatureTrike, herd: [2]int{2, 5}, per: 1.0, isle: isles(1, 0.3, 0.3, 0, 0.3, 0.15, 0.1, 0), ring: [3]float64{0.5, 1, 0.8}, homeAway: 55, each: true},
	{kind: CreatureToro, herd: [2]int{2, 3}, per: 1.0, isle: isles(1, 0.5, 0.5, 0, 0.4, 0.2, 0.3, 0), ring: [3]float64{0, 1, 0.8}, each: true},
	{kind: CreatureStego, herd: [2]int{2, 3}, per: 1.0, isle: isles(0.3, 1, 0.4, 0, 0, 0.4, 0.8, 0), ring: [3]float64{0.35, 1, 0.8}, homeAway: 55, each: true},
	{kind: CreatureBrachio, herd: [2]int{3, 5}, per: 1.4, isle: isles(1.2, 0.6, 0.4, 0, 0.45, 0.6, 0.3, 0), ring: [3]float64{0, 0.7, 1}, each: true},
	{kind: CreatureAnzu, herd: [2]int{1, 2}, per: 0.9, isle: isles(0, 1, 1.2, 0, 0, 0.2, 0.8, 0), ring: [3]float64{0, 1, 0.8}, each: true},
	{kind: CreatureAnkylo, herd: [2]int{1, 3}, per: 0.6, isle: isles(0.6, 1, 0.6, 0, 0.2, 0.3, 0.5, 0), ring: [3]float64{0, 1, 1}},
	{kind: CreatureMammoth, herd: [2]int{3, 4}, per: 1.6, isle: isles(0, 0, 0, 1, 0, 0, 0, 0), ring: [3]float64{0, 1, 1}, each: true},
	// Hunters.
	{kind: CreatureAchero, herd: [2]int{4, 6}, per: 0.25, isle: isles(0.2, 1, 0.4, 0, 0, 0.6, 0.6, 0), ring: [3]float64{0, 1, 0.8}, inland: 8, each: true},
	{kind: CreatureRaptor, herd: [2]int{3, 3}, per: 0.45, isle: isles(0.4, 0.6, 0, 0, 0.3, 1, 0.3, 0), ring: [3]float64{0, 0.5, 1}, inland: 18},
	{kind: CreatureRex, herd: [2]int{1, 1}, per: 0.25, isle: isles(0.3, 0.2, 0, 0, 1, 1, 0, 0), ring: [3]float64{0, 0, 1}, each: true},
	{kind: CreatureSabre, herd: [2]int{1, 1}, per: 1.6, isle: isles(0, 0, 0, 1, 0, 0, 0, 0), ring: [3]float64{0, 1, 1}, each: true},
	{kind: CreatureQuetzal, herd: [2]int{1, 1}, per: 0.6, isle: isles(0.5, 0.1, 0.2, 0, 1, 0.4, 0, 0), ring: [3]float64{0, 0.3, 1}},
	{kind: CreaturePtero, herd: [2]int{1, 1}, per: 0.7, isle: isles(0.5, 0.5, 0.3, 0.4, 0.8, 0.8, 1, 0.5), ring: [3]float64{0, 1, 1}},
	{kind: CreatureScorpion, herd: [2]int{1, 1}, per: 0.85, isle: isles(0, 0, 0, 0, 1, 0, 0, 0), ring: [3]float64{0, 0.6, 1}},
	{kind: CreatureCroc, herd: [2]int{1, 1}, per: 1.5, isle: isles(0.15, 0.25, 1, 0, 0, 0, 0.25, 0), ring: [3]float64{0, 1, 1}},
	{kind: CreatureBrachy, herd: [2]int{1, 1}, per: 0.9, isle: isles(0, 0.2, 1, 0, 0, 0, 0.2, 0), ring: [3]float64{0, 1, 1}},
	{kind: CreatureChampso, herd: [2]int{1, 1}, per: 0.9, isle: isles(0, 0.2, 1, 0, 0, 0, 0.2, 0), ring: [3]float64{0, 1, 1}},
}

// wildsHunter reports whether k is one of the hunters of wildRules: none of
// them lives on the home island.
func wildsHunter(k CreatureKind) bool {
	switch k {
	case CreatureAchero, CreatureRaptor, CreatureRex, CreatureSabre, CreatureQuetzal, CreaturePtero,
		CreatureScorpion, CreatureCroc, CreatureBrachy, CreatureChampso:
		return true
	}
	return false
}

// wildsTilesPer is tilesPer for the kinds that keep their old rules with
// new numbers since StyleWilds: far fewer sea monsters, more plesiosaurs
// and fish.
var wildsTilesPer = map[CreatureKind]int{
	CreatureMosa:   6000,  // 2200 before
	CreatureKraken: 14000, // 4500 before
	CreaturePlesio: 1300,  // 4000 before
	CreatureFish:   28,    // 40 before
	CreatureMonkey: 420,   // 300 before: the woods are wider
}

// wildsOffshore is the offshore range of the deep swimmers since
// StyleWilds: the sea monsters keep further out.
var wildsOffshore = map[CreatureKind][2]int{
	CreatureMosa:   {16, 250},
	CreatureKraken: {22, 250},
}

// wildsHomeAway is how far from the home island's land each sea creature
// keeps since StyleWilds, in tiles.
var wildsHomeAway = map[CreatureKind]int{
	CreaturePlesio: wildsHomeSea,
	CreatureMosa:   wildsMosaAway,
	CreatureKraken: wildsKrakenAway,
}

// wildIsle is what the wilds know of one island while they are laid out.
type wildIsle struct {
	Island
	ring   int     // 0 home, 1 near, 2 far
	groups int     // groups of grazers planned so far
	spots  [][]int // per kind: the island's tiles where it may be put
}

// wilds is the plan of the animals of a world being made.
type wilds struct {
	isles  []wildIsle
	index  []int16 // per surface tile: its island; see islandIndex
	inland []uint8 // per surface tile: tiles from the sea; see landDistance
	rules  [NumCreatureKinds]*wildRule
}

// newWilds sizes up the islands of w for the animals: their rings, and
// where on each every kind of wildRules may go, out of all (allSpawnSpots).
func (w *World) newWilds(all *[NumCreatureKinds][]int32) *wilds {
	W := w.width
	wl := &wilds{index: w.islandIndex(), inland: w.landDistance(60)}
	for i := range wildRules {
		wl.rules[wildRules[i].kind] = &wildRules[i]
	}
	rings := w.islandRings(wl.index)
	for k, is := range w.islands {
		wl.isles = append(wl.isles, wildIsle{Island: is, ring: rings[k], spots: make([][]int, NumCreatureKinds)})
	}
	for _, r := range wildRules {
		for _, i := range all[r.kind] {
			k := wl.index[i]
			if k < 0 || int(wl.inland[i]) < r.inland {
				continue
			}
			is := &wl.isles[k]
			if is.ring == 0 {
				x, y := float64(int(i)%W)+0.5, float64(int(i)/W)+0.5
				if r.homeAway > 0 && math.Hypot(x-w.spawnX, y-w.spawnY) < float64(r.homeAway) {
					continue
				}
			}
			is.spots[r.kind] = append(is.spots[r.kind], int(i))
		}
	}
	return wl
}

// islandRings is the ring of each island of w, by index (islandIndex): 0
// the home island, 2 the far ones (the desert, the volcano, the snow, and
// any further out from the spawn than wildsFarRing of half the world), 1
// the near ones, the rest.
func (w *World) islandRings(index []int16) []int {
	W := w.width
	half := float64(max(w.width, w.SurfaceHeight())) / 2
	sum := make([][3]float64, len(w.islands))
	for i, k := range index {
		if k >= 0 {
			sum[k][0] += float64(i%W) + 0.5
			sum[k][1] += float64(i/W) + 0.5
			sum[k][2]++
		}
	}
	rings := make([]int, len(w.islands))
	for k, is := range w.islands {
		ring := 1
		far := 0.0
		if sum[k][2] > 0 {
			far = math.Hypot(sum[k][0]/sum[k][2]-w.spawnX, sum[k][1]/sum[k][2]-w.spawnY) / half
		}
		switch {
		case is.Home:
			ring = 0
		case is.Kind == IslandSnow || is.Kind == IslandDesert || is.Kind == IslandAsh || far > wildsFarRing:
			ring = 2
		}
		rings[k] = ring
	}
	return rings
}

// groupsOn is how many groups of r's kind island is gets: its share by
// area and weights, a chance of one more for the part of a group left over,
// and at least one where the kind is the island's own (wildRule.each).
func (w *World) groupsOn(r *wildRule, is *wildIsle) int {
	wt := r.isle[is.Kind] * r.ring[is.ring]
	if wt <= 0 || len(is.spots[r.kind]) == 0 {
		return 0
	}
	want := float64(is.Tiles) / wildsPerTiles * r.per * wt
	n := int(want)
	if w.random().Float64() < want-float64(n) {
		n++
	}
	if r.each && r.isle[is.Kind] >= 1 && is.Tiles >= wildsEachTiles {
		n = max(n, 1)
	}
	return n
}

// spawnWild puts the groups of rule r's kind on every island, and keeps
// where they went for repopulate. It returns how many animals it put.
func (w *World) spawnWild(wl *wilds, r *wildRule) int {
	f := &w.fauna
	k := r.kind
	f.herdRange[k] = r.herd
	f.spawnAt[k] = nil
	placed := 0
	for ii := range wl.isles {
		is := &wl.isles[ii]
		n := w.groupsOn(r, is)
		spots := is.spots[k]
		for g := 0; g < n; g++ {
			i := spots[w.random().IntN(len(spots))]
			if is.ring == 0 && g < wildsNearSpawn && (k == CreatureThesc || k == CreatureOrnitho) {
				i = w.nearSpawnSpot(spots, i) // where a newcomer sees them in the first minutes
			}
			placed += w.addGroup(wl, is, k, i, spots)
		}
	}
	f.target[k] = placed
	return placed
}

// addGroup puts a group of kind k at tile i of island is, remembers it and
// a few more of the island's spots for repopulate, and returns how many
// animals fitted.
func (w *World) addGroup(wl *wilds, is *wildIsle, k CreatureKind, i int, spots []int) int {
	f := &w.fauna
	x, y := float64(i%w.width)+0.5, float64(i/w.width)+0.5
	got := w.addHerd(k, x, y, w.herdSize(k))
	f.spawnAt[k] = append(f.spawnAt[k], int32(i))
	for range wildsSites - 1 {
		f.spawnAt[k] = append(f.spawnAt[k], int32(spots[w.random().IntN(len(spots))]))
	}
	if !wildsHunter(k) {
		is.groups++ // the grazers: see fillIsles
	}
	return got
}

// nearSpawnSpot is a spot of spots between wildsNearLo and wildsNearHi tiles
// from the spawn, or i if none is.
func (w *World) nearSpawnSpot(spots []int, i int) int {
	for range 40 {
		j := spots[w.random().IntN(len(spots))]
		d := math.Hypot(float64(j%w.width)+0.5-w.spawnX, float64(j/w.width)+0.5-w.spawnY)
		if d >= wildsNearLo && d <= wildsNearHi {
			return j
		}
	}
	return i
}

// fillIsles gives every island of wildsMinTiles or more that has fewer than
// a few groups the grazers that suit it best, until it has them.
func (w *World) fillIsles(wl *wilds) {
	f := &w.fauna
	for ii := range wl.isles {
		is := &wl.isles[ii]
		if is.Tiles < wildsMinTiles {
			continue
		}
		want := 1 + is.Tiles/wildsMinPer
		for tries := 0; is.groups < want && tries < 4*want; tries++ {
			// The grazer that suits this island best and has ground on it,
			// then the next: round them in turn.
			var best []*wildRule
			for i := range wildRules {
				r := &wildRules[i]
				if !wildsHunter(r.kind) && r.isle[is.Kind]*r.ring[is.ring] > 0 && len(is.spots[r.kind]) > 0 {
					best = append(best, r)
				}
			}
			if len(best) == 0 {
				break
			}
			for a := 1; a < len(best); a++ { // by weight, the highest first
				for b := a; b > 0 && best[b].isle[is.Kind]*best[b].ring[is.ring] > best[b-1].isle[is.Kind]*best[b-1].ring[is.ring]; b-- {
					best[b], best[b-1] = best[b-1], best[b]
				}
			}
			r := best[tries%min(len(best), 3)]
			spots := is.spots[r.kind]
			f.target[r.kind] += w.addGroup(wl, is, r.kind, spots[w.random().IntN(len(spots))], spots)
		}
	}
}

// homeSeaDistance is, for every tile of the surface, how many tiles it
// lies from the home island's land, up to 255.
func (w *World) homeSeaDistance(index []int16) []uint8 {
	W, H := w.width, w.SurfaceHeight()
	d := make([]uint8, W*H)
	var q []int32
	for i := range d {
		if k := index[i]; k >= 0 && w.islands[k].Home {
			q = append(q, int32(i))
		} else {
			d[i] = 255
		}
	}
	for n := 0; n < len(q); n++ {
		i := int(q[n])
		x, y := i%W, i/W
		for _, s := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+s[0], y+s[1]
			if nx < 0 || ny < 0 || nx >= W || ny >= H {
				continue
			}
			if j := ny*W + nx; d[j] > d[i]+1 {
				d[j] = d[i] + 1
				q = append(q, int32(j))
			}
		}
	}
	return d
}

// onHome reports whether tile (tx, ty) is on the home island, by index (see
// islandIndex).
func (w *World) onHome(index []int16, tx, ty int) bool {
	if !w.inBounds(tx, ty) || ty >= w.SurfaceHeight() {
		return false
	}
	k := index[ty*w.width+tx]
	return k >= 0 && w.islands[k].Home
}

// islandIndex is, for every tile of the surface, the index in w.islands of
// the island it is part of, or -1: the sea, the islets too small to be
// listed. Each island is found again from the tile it was listed by, so
// land changed since (a hut, a mouth, a ramp) does not shift the others.
func (w *World) islandIndex() []int16 {
	W, H := w.width, w.SurfaceHeight()
	out := make([]int16, W*H)
	for i := range out {
		out[i] = -1
	}
	var q []int
	for k, isl := range w.islands {
		if isl.At < 0 || isl.At >= W*H || w.tiles[isl.At].Sea() || out[isl.At] >= 0 {
			continue
		}
		out[isl.At] = int16(k)
		q = append(q[:0], isl.At)
		for n := 0; n < len(q); n++ {
			x, y := q[n]%W, q[n]/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= W || ny >= H {
					continue
				}
				if j := ny*W + nx; out[j] < 0 && !w.tiles[j].Sea() {
					out[j] = int16(k)
					q = append(q, j)
				}
			}
		}
	}
	return out
}

// IslandMap is, for every tile of the surface, the index in Islands() of
// the island it is part of, or -1 (the sea, and islets). For tools.
func (w *World) IslandMap() []int16 { return w.islandIndex() }
