package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// What lies and lives in the natural caves: see cavesys.go. Once the
// systems are dug, every chamber gets a part to play (zones), and at the
// end of generation (finishCaves) the caves are dressed: stalagmites and
// boulders, coal in seams along the walls of the upper level, jade and the
// deep ores on the lower one, a hoard at the end of the main route and
// something at the end of each side branch, and their life: spiders only in
// their nests, bears only in their dens, as many as the caves are big.

// Cave dressing tuning.
const (
	caveCoalPer     = 650  // floor tiles of the upper level to each coal seam
	caveJadePer     = 300  // floor tiles of the lower level to each jade vein
	caveDeepOrePer  = 2600 // and to each vein of agate or obsidian, under the desert and the volcano
	caveBoulderPer  = 85   // floor tiles to each boulder
	cavePebblePer   = 45   // and pebble
	caveMushPer     = 320  // and mushroom to eat
	caveGlowPer     = 260  // and glowing mushroom
	caveNestPer     = 22000
	caveDenPer      = 32000
	caveSpiderNest  = 3 // spiders in a nest, besides one more for every 4 tiles of its radius
	caveBearsBig    = 12.0
	caveStalagPer   = 4.5  // tiles of a chamber's radius to each stalagmite cluster
	caveTreasureTop = 30   // an old chest's depth (its Var, see treasureLoot) at the end of the main route
	caveHoardPer    = 4400 // floor tiles of the upper level to each hoard off the main route (half as many below)
)

// assignZones gives the chambers of s their parts: the spiders' nests and
// the bears' dens, as many as the island is big, the nests off the main
// route where they can be, the dens down on the lower level.
func (s *caveSystem) assignZones(rng *rand.Rand) {
	free := func(lower, offRoute bool) []int {
		var out []int
		for i, c := range s.chambers {
			if c.hall || c.zone != ZoneHall || i == s.descent || i == s.arrival {
				continue
			}
			if lower && c.level != s.levels-1 || offRoute && c.route {
				continue
			}
			out = append(out, i)
		}
		rng.Shuffle(len(out), func(a, b int) { out[a], out[b] = out[b], out[a] })
		return out
	}
	pick := func(z CaveZone, n int, prefs ...[]int) {
		for _, p := range prefs {
			for _, i := range p {
				if n == 0 {
					return
				}
				if s.chambers[i].zone == ZoneHall {
					s.chambers[i].zone = z
					n--
				}
			}
		}
	}
	nests := int(math.Round(float64(s.area) / caveNestPer))
	dens := int(math.Round(float64(s.area) / caveDenPer))
	if s.levels == 2 {
		dens = max(dens, 1)
	}
	pick(ZoneDen, dens, free(true, true), free(true, false), free(false, true))
	pick(ZoneNest, nests, free(false, true), free(false, false))
}

// caveFloor is one level of a system as it is being dressed.
type caveFloor struct {
	s     *caveSystem
	lv    int
	top   int     // the band's first map row
	wall  []uint8 // per cell: steps to the nearest rock (1 beside it), 0 on rock
	taken []bool  // per cell: kept clear, or something stands there already
}

// finishCaves dresses every system once the rest of the world is made, and
// settles its life.
func (w *World) finishCaves() {
	rng := rand.New(rand.NewPCG(w.seed, 0xd2e55))
	w.fauna.creatures = slices.DeleteFunc(w.fauna.creatures, func(c *Creature) bool {
		return (c.Kind == CreatureSpider || c.Kind == CreatureBear) && w.Underground(int(c.Y))
	})
	f := &w.fauna
	f.spawnAt[CreatureSpider], f.spawnAt[CreatureBear] = nil, nil
	f.target[CreatureSpider], f.target[CreatureBear] = 0, 0
	for _, s := range w.caves.systems {
		floors := make([]*caveFloor, s.levels)
		for lv := range floors {
			floors[lv] = w.caveFloorOf(s, lv)
		}
		for _, fl := range floors {
			w.dressFloor(rng, fl)
		}
		w.settleLife(rng, s, floors)
	}
	w.fauna.shallow = nil // the caves' waters are fished too
}

// caveFloorOf works out level lv of s for dressing: how far each cell is
// from the walls, and the cells kept clear (the ways up and down, where
// they set a player down, and whatever stands there already).
func (w *World) caveFloorOf(s *caveSystem, lv int) *caveFloor {
	fl := &caveFloor{s: s, lv: lv, top: CaveBandTop(w.surfaceH, lv)}
	n := s.gw * s.gh
	fl.wall, fl.taken = make([]uint8, n), make([]bool, n)
	var q []int
	for i := range n {
		tx, ty := s.x0+i%s.gw, fl.top+s.y0+i/s.gw
		if !w.TileAt(tx, ty).Walkable() {
			q = append(q, i)
			continue
		}
		fl.wall[i] = 255
		if w.entityAt(tx, ty) != nil {
			fl.taken[i] = true
		}
	}
	for k := 0; k < len(q); k++ {
		x, y := q[k]%s.gw, q[k]/s.gw
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= s.gw || ny >= s.gh {
					continue
				}
				if j := ny*s.gw + nx; fl.wall[j] > fl.wall[q[k]]+1 {
					fl.wall[j] = fl.wall[q[k]] + 1
					q = append(q, j)
				}
			}
		}
	}
	keep := func(tx, ty, r int) { // map tile (tx, ty), and r tiles round it
		for y := ty - fl.top - r; y <= ty-fl.top+r; y++ {
			for x := tx - r; x <= tx+r; x++ {
				if c, ok := s.gridAt(x, y); ok {
					fl.taken[c] = true
				}
			}
		}
	}
	for _, e := range s.exits {
		if lv == 0 {
			keep(e[0], e[1], 1)
			keep(e[0], e[1]+2, 1)
		}
	}
	if s.levels == 2 {
		h := s.down
		if lv == 1 {
			h = s.up
		}
		keep(h[0], h[1], 2)
	}
	return fl
}

// at is the map tile of cell i of the level.
func (fl *caveFloor) at(i int) (int, int) {
	return fl.s.x0 + i%fl.s.gw, fl.top + fl.s.y0 + i/fl.s.gw
}

// cellsOf lists the floor cells of chamber c (or with c < 0, of the
// tunnels), in order.
func (fl *caveFloor) cellsOf(c int) []int {
	var out []int
	for i, o := range fl.s.owner[fl.lv] {
		if fl.wall[i] > 0 && int(o)-1 == c && (c >= 0 || o == 0) {
			out = append(out, i)
		}
	}
	return out
}

// put sets an entity of kind k on cell i if it is free, and reports
// whether it did.
func (w *World) caveAdd(fl *caveFloor, i int, k EntityKind) *Entity {
	if fl.taken[i] {
		return nil
	}
	tx, ty := fl.at(i)
	if !w.canPlace(tx, ty) {
		fl.taken[i] = true
		return nil
	}
	fl.taken[i] = true
	return w.addEntity(k, tx, ty)
}

// dressFloor lays what lies on one level of a system.
func (w *World) dressFloor(rng *rand.Rand, fl *caveFloor) {
	s := fl.s
	floor := s.floor[fl.lv]
	var all []int
	for i, d := range fl.wall {
		if d > 0 {
			all = append(all, i)
		}
	}
	if len(all) == 0 {
		return
	}
	// The hoards: at the end of the main route, and of each side branch.
	for ci, c := range s.chambers {
		if c.level != fl.lv {
			continue
		}
		switch {
		case ci == s.dest && c.hall: // a cave of one hall, on an islet
			if rng.Float64() < 0.35 {
				w.caveHoard(rng, fl, ci, treasureDepth+2)
			}
		case ci == s.dest:
			w.caveHoard(rng, fl, ci, caveTreasureTop+20*fl.lv)
		case c.leaf && rng.Float64() < 0.8:
			w.caveHoard(rng, fl, ci, treasureDepth+4+18*fl.lv+rng.IntN(8))
		}
	}
	// And a few more off the main route, as many as the level is big: more,
	// and richer, down below.
	var off []int
	for ci, c := range s.chambers {
		if c.level == fl.lv && !c.hall && !c.route && !c.leaf && ci != s.dest {
			off = append(off, ci)
		}
	}
	rng.Shuffle(len(off), func(a, b int) { off[a], off[b] = off[b], off[a] })
	for _, ci := range off[:min(len(off), floor/(caveHoardPer-caveHoardPer/2*fl.lv))] {
		w.caveHoard(rng, fl, ci, treasureDepth+2+20*fl.lv+rng.IntN(10))
	}
	// Stalagmites stand free in the chambers, off the tunnels.
	for ci, c := range s.chambers {
		if c.level != fl.lv || c.hall {
			continue
		}
		cells := fl.cellsOf(ci)
		rng.Shuffle(len(cells), func(a, b int) { cells[a], cells[b] = cells[b], cells[a] })
		want := int(c.r / caveStalagPer)
		for _, i := range cells {
			if want == 0 {
				break
			}
			tx, ty := fl.at(i)
			if fl.wall[i] < 3 || s.nearTunnel(fl.lv, float64(tx)+0.5, float64(ty-fl.top)+0.5, 2.6) || w.blockerNear(tx, ty, 3) {
				continue
			}
			if e := w.caveAdd(fl, i, EntityStalagmite); e != nil {
				e.Var = uint8(rng.IntN(4))
				want--
			}
		}
	}
	// Ores in seams along the walls: coal above, jade and the deep ores below.
	if fl.lv == 0 {
		w.caveSeams(rng, fl, all, EntityCoal, floor/caveCoalPer, [2]int{4, 7})
	} else {
		w.caveSeams(rng, fl, all, EntityIronOre, max(1, floor/caveJadePer), [2]int{3, 6})
		switch w.islands[s.island].Kind {
		case IslandDesert:
			w.caveSeams(rng, fl, all, EntityPlatinumOre, floor/caveDeepOrePer, [2]int{2, 4})
		case IslandAsh:
			w.caveSeams(rng, fl, all, EntityDiamondOre, floor/caveDeepOrePer, [2]int{2, 4})
		}
	}
	// Loose rock, and mushrooms by the walls.
	scatter := func(k EntityKind, per int, wall [2]uint8, v func() uint8) {
		n := floor / per
		for tries := 0; n > 0 && tries < n*12; tries++ {
			i := all[rng.IntN(len(all))]
			if fl.wall[i] < wall[0] || fl.wall[i] > wall[1] {
				continue
			}
			if e := w.caveAdd(fl, i, k); e != nil {
				if v != nil {
					e.Var = v()
				}
				if k == EntityBoulder && rng.Float64() < bigBoulder {
					e.Big, e.HP = true, maxHP(k, true)
				}
				n--
			}
		}
	}
	scatter(EntityBoulder, caveBoulderPer, [2]uint8{1, 2}, nil)
	scatter(EntityPebble, cavePebblePer, [2]uint8{1, 4}, nil)
	scatter(EntityMushroom, caveMushPer, [2]uint8{1, 2}, nil)
	glow := caveGlowPer
	if fl.lv == 1 {
		glow *= 2 // the deep is darker
	}
	scatter(EntityGlowPlant, glow, [2]uint8{1, 2}, func() uint8 { return 0 })
}

// blockerNear reports whether something that blocks the way stands within
// r tiles of (tx, ty).
func (w *World) blockerNear(tx, ty, r int) bool {
	for y := ty - r; y <= ty+r; y++ {
		for x := tx - r; x <= tx+r; x++ {
			if e := w.entityAt(x, y); e != nil && blocksPlayers(e.Kind) {
				return true
			}
		}
	}
	return false
}

// caveHoard hides an old chest of the given depth in chamber ci, against
// its far wall from the tunnels.
func (w *World) caveHoard(rng *rand.Rand, fl *caveFloor, ci, depth int) {
	s := fl.s
	best, bestV := -1, -1.0
	for _, i := range fl.cellsOf(ci) {
		if fl.taken[i] || fl.wall[i] < 2 || fl.wall[i] > 3 {
			continue
		}
		tx, ty := fl.at(i)
		v := 0.0
		for _, t := range s.tunnels {
			if t.level != fl.lv || t.a != ci && t.b != ci {
				continue
			}
			v = math.Inf(1)
			for j := 0; j+1 < len(t.path); j++ {
				p, q := t.path[j], t.path[j+1]
				_, d := segDist(float64(tx)+0.5, float64(ty-fl.top)+0.5, p[0], p[1], q[0], q[1])
				v = min(v, d)
			}
		}
		v += rng.Float64()
		if v > bestV {
			best, bestV = i, v
		}
	}
	if best < 0 {
		return
	}
	if e := w.caveAdd(fl, best, EntityTreasure); e != nil {
		e.Var = uint8(min(255, depth))
	}
}

// caveSeams lays n veins of ore k along the walls of a level, each of a few
// blocks following the wall.
func (w *World) caveSeams(rng *rand.Rand, fl *caveFloor, all []int, k EntityKind, n int, size [2]int) {
	s := fl.s
	for tries := 0; n > 0 && tries < n*20; tries++ {
		i := all[rng.IntN(len(all))]
		if fl.wall[i] != 1 || fl.taken[i] {
			continue
		}
		want := size[0] + rng.IntN(size[1]-size[0]+1)
		got := 0
		for steps := 0; got < want && steps < want*8; steps++ {
			if fl.wall[i] >= 1 && fl.wall[i] <= 2 && w.caveAdd(fl, i, k) != nil {
				got++
			}
			x, y := i%s.gw+rng.IntN(3)-1, i/s.gw+rng.IntN(3)-1
			if x < 0 || y < 0 || x >= s.gw || y >= s.gh {
				continue
			}
			if j := y*s.gw + x; fl.wall[j] >= 1 && fl.wall[j] <= 2 {
				i = j
			}
		}
		if got > 0 {
			n--
		}
	}
}

// settleLife puts the spiders in their nests and the bears in their dens,
// and keeps where each may come back.
func (w *World) settleLife(rng *rand.Rand, s *caveSystem, floors []*caveFloor) {
	f := &w.fauna
	for ci, c := range s.chambers {
		var k CreatureKind
		n := 0
		switch c.zone {
		case ZoneNest:
			k, n = CreatureSpider, caveSpiderNest+int(c.r/4)
		case ZoneDen:
			k, n = CreatureBear, 1
			if c.r >= caveBearsBig {
				n = 2
			}
		default:
			continue
		}
		fl := floors[c.level]
		cells := fl.cellsOf(ci)
		var spots []int32
		for _, i := range cells {
			if fl.wall[i] >= 2 {
				tx, ty := fl.at(i)
				spots = append(spots, int32(ty*w.width+tx))
			}
		}
		if len(spots) == 0 {
			continue
		}
		f.spawnAt[k] = append(f.spawnAt[k], spots...)
		for placed, tries := 0, 0; placed < n && tries < n*10; tries++ {
			i := spots[rng.IntN(len(spots))]
			x, y := float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5
			if w.creatureFits(k, x, y) {
				w.addCreature(k, x, y)
				placed++
			}
		}
		f.target[k] += n
	}
}
