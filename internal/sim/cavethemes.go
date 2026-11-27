package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// The themes of the natural caves (see cavesys.go). Each system takes the
// character of the island above it: its rock everywhere (pale limestone
// under the meadows, mossy stone with roots under the woods, red sandstone
// under the desert, black basalt under the volcano, ice under the snow,
// dripping mud rock under the marsh), and its own chambers, each where it
// belongs: an underground stream under the jungle, the desert's ancient
// tomb, lava tubes under the volcano, ice caves round frozen pools under
// the snow, flooded caves with crocodiles under the marsh, a quiet lake
// under the meadows. The destination at the end of the main route is the
// island's own: the tomb's hoard, the lava lake, the frozen pool...
//
// Every system has chambers of the common kinds too, by its size: a cavern
// of glowing crystals (on the lower level where it can be, with jade in its
// walls), a lake with blind fish, a grove of glowing mushrooms with some to
// eat, spiders' nests hung with webs, bears' dens strewn with bones, a roost
// of bats, a great skeleton in the rock to dig bones out of, and pictures
// painted on a wall long ago that tell of a hoard, of the beasts, or of a
// star that fell.

// CaveProp is which thing of the caves an EntityCaveProp is.
type CaveProp uint8

const (
	PropCrystal     CaveProp = iota // a cluster of glowing jade crystals by a wall: it lights the cave round it
	PropGlowShroom                  // a giant glowing mushroom
	PropWeb                         // a spider's web across the floor: it catches whoever walks into it
	PropEggSac                      // spiders' egg sacs in a corner
	PropBones                       // bones of a beast's meals
	PropFossil                      // a great skeleton in the rock: bones to dig out once
	PropPainting                    // pictures painted on the wall above: Fuel says what they show
	PropSarcophagus                 // the stone coffin of the desert's tomb
	PropUrn                         // the tomb's clay jars
	PropRoots                       // roots hanging through the roof, under the woods
	PropIcicles                     // spikes of ice by a wall, under the snow
	PropObsidian                    // shards of black glass by the lava
	PropRoost                       // where the bats hang, over their droppings
	NumCaveProps
)

// The looks of EntityStalagmite past its ordinary ones (0 to 3).
const (
	StalagPillar  = 8  // a carved pillar of the tomb
	StalagIce     = 9  // a spire of ice
	StalagBasalt  = 10 // a column of basalt
	StalagCrystal = 11 // a crystal taller than a man, glowing
)

// Theme tuning.
const (
	caveThemedPer   = 40000 // tiles of island to each more chamber of its own kind, past the first
	caveCrystalPer  = 45000
	caveMushPer2    = 50000
	caveLakePer     = 60000
	caveFishPer     = 45 // tiles of water to each fish
	caveLavaBurn    = 3.0
	caveLavaBurnCD  = TicksPerSecond / 2
	caveFossilBones = 5
	cavePathClear   = 2.7 // tiles each side of a tunnel's path kept dry and clear
)

// themeOf is the chamber of its own an island of kind k has under it.
func themeOf(k IslandKind) CaveZone {
	switch k {
	case IslandDesert:
		return ZoneTomb
	case IslandAsh:
		return ZoneLava
	case IslandSnow:
		return ZoneFrozen
	case IslandMarsh:
		return ZoneFlooded
	case IslandForest, IslandMonkey:
		return ZoneStream
	}
	return ZoneLake
}

// assignZones gives the chambers of s their parts: the destination the
// island's own, more of the island's chambers under rock of its kind, the
// spiders' nests and the bears' dens as many as the island is big, and one
// or a few of each common kind, where each belongs.
func (s *caveSystem) assignZones(rng *rand.Rand, kind IslandKind) {
	own := themeOf(kind)
	free := func(ok func(i int, c caveChamber) bool) []int {
		var out []int
		for i, c := range s.chambers {
			if c.hall || c.zone != ZoneHall || i == s.descent || i == s.arrival || s.shaft(i) || !ok(i, c) {
				continue
			}
			out = append(out, i)
		}
		rng.Shuffle(len(out), func(a, b int) { out[a], out[b] = out[b], out[a] })
		return out
	}
	anywhere := func(int, caveChamber) bool { return true }
	pick := func(z CaveZone, n int, prefs ...func(i int, c caveChamber) bool) {
		for _, p := range prefs {
			for _, i := range free(p) {
				if n <= 0 {
					return
				}
				s.chambers[i].zone = z
				n--
			}
		}
	}
	if d := &s.chambers[s.dest]; !d.hall {
		d.zone = own
	}
	// More of the island's own, under rock of its kind.
	if own != ZoneTomb { // the tomb is one
		rock := rockOfIsland(kind)
		pick(own, s.area/caveThemedPer, func(_ int, c caveChamber) bool { return c.rock == rock })
	}
	lower := func(_ int, c caveChamber) bool { return c.level == s.levels-1 }
	upper := func(_ int, c caveChamber) bool { return c.level == 0 }
	offRoute := func(_ int, c caveChamber) bool { return !c.route }
	leaf := func(_ int, c caveChamber) bool { return c.leaf }
	both := func(a, b func(int, caveChamber) bool) func(int, caveChamber) bool {
		return func(i int, c caveChamber) bool { return a(i, c) && b(i, c) }
	}
	dens := int(math.Round(float64(s.area) / caveDenPer))
	if s.levels == 2 {
		dens = max(dens, 1)
	}
	pick(ZoneDen, dens, both(lower, offRoute), lower, offRoute)
	pick(ZoneNest, int(math.Round(float64(s.area)/caveNestPer)), offRoute, anywhere)
	if s.area < caveLowerFrom/2 {
		return // a small cave: its hall and a chamber or two
	}
	pick(ZoneCrystal, 1+s.area/caveCrystalPer, both(lower, leaf), lower, leaf, anywhere)
	pick(ZonePaintings, 1, both(upper, leaf), upper)
	pick(ZoneFossil, 1, leaf, anywhere)
	pick(ZoneMushroom, 1+s.area/caveMushPer2, both(upper, leaf), upper, anywhere)
	pick(ZoneRoost, 1, both(upper, offRoute), upper)
	if own != ZoneLake {
		pick(ZoneLake, 1+s.area/caveLakePer, offRoute, anywhere)
	}
}

// pathMask marks the cells of level fl within cavePathClear of a tunnel's
// path: the ways through the chambers, kept dry and clear.
func (w *World) pathMask(fl *caveFloor) []bool {
	s := fl.s
	out := make([]bool, s.gw*s.gh)
	stamp := func(path [][2]float64) {
		r := cavePathClear
		for j := 0; j+1 < len(path); j++ {
			p, q := path[j], path[j+1]
			l := math.Hypot(q[0]-p[0], q[1]-p[1])
			for k := 0.0; k <= l; k += 0.5 {
				x, y := p[0]+(q[0]-p[0])*k/max(l, 1e-9), p[1]+(q[1]-p[1])*k/max(l, 1e-9)
				for ty := int(math.Floor(y - r)); ty <= int(math.Ceil(y+r)); ty++ {
					for tx := int(math.Floor(x - r)); tx <= int(math.Ceil(x+r)); tx++ {
						if math.Hypot(float64(tx)+0.5-x, float64(ty)+0.5-y) > r {
							continue
						}
						if c, ok := s.gridAt(tx, ty); ok {
							out[c] = true
						}
					}
				}
			}
		}
	}
	for _, t := range s.tunnels {
		if t.level == fl.lv {
			stamp(t.path)
		}
	}
	if fl.lv == 1 {
		for _, sec := range w.caves.secrets {
			if w.caves.systems[sec.a] == s || w.caves.systems[sec.b] == s {
				stamp(sec.path)
			}
		}
	}
	return out
}

// pourCaves lays the waters, the lava and the ice of the themed chambers
// of s, off the ways through them, and takes away whatever stood there.
func (w *World) pourCaves(rng *rand.Rand, s *caveSystem, floors []*caveFloor) {
	for _, fl := range floors {
		path := w.pathMask(fl)
		for ci, c := range s.chambers {
			if c.level != fl.lv {
				continue
			}
			seed := uint32(w.seed) + uint32(ci)*977 + uint32(s.island)*131
			switch c.zone {
			case ZoneLake:
				lx, ly := s.poolSpot(rng, ci, 0.48)
				w.pour(fl, ci, TileRiver, func(i int, x, y float64) bool {
					d := math.Hypot(x-lx, y-ly)/(0.66*c.r) + 0.45*(fbm(x/5, y/5, seed, 2)-0.5)
					return fl.wall[i] >= 3 && !path[i] && d < 1
				})
			case ZoneFlooded:
				w.pour(fl, ci, TileRiver, func(i int, x, y float64) bool {
					return fl.wall[i] >= 3 && !path[i] && fbm(x/4.5, y/4.5, seed, 2) > 0.54
				})
			case ZoneLava:
				lx, ly := s.poolSpot(rng, ci, 0.45)
				w.pour(fl, ci, TileLava, func(i int, x, y float64) bool {
					d := math.Hypot(x-lx, y-ly)/(0.55*c.r) + 0.45*(fbm(x/5, y/5, seed, 2)-0.5)
					return fl.wall[i] >= 3 && !path[i] && d < 1
				})
				w.pourChannel(rng, fl, ci, path, TileLava, TileCave)
			case ZoneFrozen:
				w.pour(fl, ci, TileIce, func(i int, x, y float64) bool {
					d := math.Hypot(x-c.x, y-c.y) / c.r
					return fl.wall[i] >= 3 && d+0.4*(fbm(x/5, y/5, seed, 2)-0.5) < 0.62
				})
			case ZoneStream:
				w.pourChannel(rng, fl, ci, path, TileRiver, TileFord)
			}
		}
	}
}

// poolSpot is where a pool lies in chamber ci: f of its radius out from its
// middle, the way furthest from every tunnel leaving it.
func (s *caveSystem) poolSpot(rng *rand.Rand, ci int, f float64) (float64, float64) {
	c := s.chambers[ci]
	var angles []float64
	for _, t := range s.chambers[ci].adj {
		p := s.tunnels[t].path
		if s.tunnels[t].b == ci {
			p = slices.Clone(p)
			slices.Reverse(p)
		}
		for _, q := range p {
			if math.Hypot(q[0]-c.x, q[1]-c.y) >= c.r*0.6 {
				angles = append(angles, math.Atan2(q[1]-c.y, q[0]-c.x))
				break
			}
		}
	}
	a := rng.Float64() * 2 * math.Pi
	if len(angles) > 0 {
		slices.Sort(angles)
		best := -1.0
		for i, x := range angles {
			next := angles[(i+1)%len(angles)]
			if i == len(angles)-1 {
				next += 2 * math.Pi
			}
			if gap := next - x; gap > best {
				best, a = gap, x+gap/2
			}
		}
	}
	return c.x + math.Cos(a)*c.r*f*c.aspect, c.y + math.Sin(a)*c.r*f/c.aspect
}

// pour turns the floor cells of chamber ci that ok holds for to tile k.
func (w *World) pour(fl *caveFloor, ci int, k Tile, ok func(i int, x, y float64) bool) {
	for _, i := range fl.cellsOf(ci) {
		tx, ty := fl.at(i)
		if fl.taken[i] && w.entityAt(tx, ty) == nil {
			continue // kept clear: the ways up and down
		}
		if !ok(i, float64(tx)+0.5, float64(ty-fl.top)+0.5) {
			continue
		}
		w.setCaveTile(fl, i, k)
	}
}

// setCaveTile makes cell i of the level tile k, clearing what stood on it.
func (w *World) setCaveTile(fl *caveFloor, i int, k Tile) {
	tx, ty := fl.at(i)
	if e := w.entityAt(tx, ty); e != nil {
		if e.Kind == EntityCaveExit || e.Kind == EntityCaveDown || e.Kind == EntityCaveUp {
			return
		}
		w.removeEntity(e)
	}
	w.tiles[ty*w.width+tx] = k
}

// pourChannel runs a channel of tile k across chamber ci from wall to wall,
// two or three tiles wide, crossing the ways through it as tile cross: a
// stream with a ford, lava under a bridge of rock.
func (w *World) pourChannel(rng *rand.Rand, fl *caveFloor, ci int, path []bool, k, cross Tile) {
	c := fl.s.chambers[ci]
	a := rng.Float64() * math.Pi
	nx, ny := -math.Sin(a), math.Cos(a)
	half := 1.0 + 0.6*rng.Float64()
	for _, i := range fl.cellsOf(ci) {
		tx, ty := fl.at(i)
		x, y := float64(tx)+0.5, float64(ty-fl.top)+0.5
		// Its distance across the channel, which wanders.
		along := (x-c.x)*math.Cos(a) + (y-c.y)*math.Sin(a)
		d := math.Abs((x-c.x)*nx + (y-c.y)*ny - 1.6*math.Sin(along/4.5))
		if d > half || fl.taken[i] && w.entityAt(tx, ty) == nil {
			continue
		}
		if path[i] {
			if cross != TileCave {
				w.setCaveTile(fl, i, cross)
			}
			continue
		}
		w.setCaveTile(fl, i, k)
	}
}

// dressThemes sets the props of each themed chamber of level fl.
func (w *World) dressThemes(rng *rand.Rand, fl *caveFloor) {
	s := fl.s
	path := w.pathMask(fl)
	for ci, c := range s.chambers {
		if c.level != fl.lv {
			continue
		}
		cells := fl.cellsOf(ci)
		if len(cells) == 0 {
			continue
		}
		rng.Shuffle(len(cells), func(a, b int) { cells[a], cells[b] = cells[b], cells[a] })
		// place sets up to n props of kind p on cells that ok holds for.
		place := func(p CaveProp, n int, ok func(i int) bool) []*Entity {
			var out []*Entity
			for _, i := range cells {
				if len(out) >= n {
					break
				}
				if ok != nil && !ok(i) {
					continue
				}
				if e := w.caveAdd(fl, i, EntityCaveProp); e != nil {
					e.Var, e.Ready = uint8(p), false
					out = append(out, e)
				}
			}
			return out
		}
		spire := func(look uint8, n int) {
			for _, i := range cells {
				if n <= 0 {
					break
				}
				tx, ty := fl.at(i)
				if fl.wall[i] < 3 || path[i] || w.blockerNear(tx, ty, 3) {
					continue
				}
				if e := w.caveAdd(fl, i, EntityStalagmite); e != nil {
					e.Var = look
					n--
				}
			}
		}
		byWall := func(i int) bool { return fl.wall[i] <= 2 }
		inside := func(i int) bool { return fl.wall[i] >= 3 && !path[i] }
		underRock := func(i int) bool { // roots hang over a wall to the north
			tx, ty := fl.at(i)
			return fl.wall[i] == 1 && w.TileAt(tx, ty-1) == TileMountain
		}
		n := int(c.r)
		switch c.zone {
		case ZoneCrystal:
			place(PropCrystal, n/2+2, byWall)
			spire(StalagCrystal, 1+n/8)
			if fl.lv == 1 {
				w.caveSeams(rng, fl, cells, EntityIronOre, 2+n/7, [2]int{3, 6})
			}
		case ZoneMushroom:
			place(PropGlowShroom, 2+n/5, inside)
			for k := range 4 + n/2 {
				i := cells[(k*7)%len(cells)]
				if e := w.caveAdd(fl, i, EntityGlowPlant); e != nil {
					e.Var = 0
				}
			}
			for k := range 2 + n/4 {
				w.caveAdd(fl, cells[(k*13+5)%len(cells)], EntityMushroom)
			}
		case ZoneNest:
			place(PropWeb, 3+n/2, nil)
			place(PropEggSac, 2+n/6, byWall)
		case ZoneDen:
			place(PropBones, 3+n/3, byWall)
		case ZoneRoost:
			place(PropRoost, 2+n/8, byWall)
		case ZoneFossil:
			es := place(PropFossil, 1, func(i int) bool { return fl.wall[i] >= 3 && !path[i] })
			if len(es) == 0 {
				es = place(PropFossil, 1, func(i int) bool { return fl.wall[i] >= 2 })
			}
			if len(es) > 0 {
				es[0].Ready = true // its bones are there to dig out
			}
			place(PropBones, 2, byWall)
		case ZonePaintings:
			for _, e := range place(PropPainting, 1+n/12, func(i int) bool { return w.underFace(fl, i) }) {
				e.Ready, e.Fuel = true, w.paintingOf(rng, s, e)
			}
		case ZoneTomb:
			place(PropSarcophagus, 1, inside)
			spire(StalagPillar, 2+n/6)
			place(PropUrn, 3+n/4, byWall)
		case ZoneLava:
			place(PropObsidian, 2+n/4, func(i int) bool { return w.nextTo(fl, i, TileLava) })
			spire(StalagBasalt, 1+n/7)
		case ZoneFrozen:
			place(PropIcicles, 3+n/3, byWall)
			spire(StalagIce, 1+n/6)
		case ZoneStream, ZoneFlooded:
			place(PropRoots, 3+n/3, underRock)
		}
		// Glowing mushrooms by the waters, their light on the water.
		switch c.zone {
		case ZoneLake, ZoneStream, ZoneFlooded:
			k := 0
			for _, i := range cells {
				if k >= 3+n/3 {
					break
				}
				if w.nextTo(fl, i, TileRiver) && !path[i] {
					if e := w.caveAdd(fl, i, EntityGlowPlant); e != nil {
						e.Var = 0
						k++
					}
				}
			}
		}
		// The rock's own, here and there in every chamber and tunnel.
		switch c.rock {
		case RockMossy:
			place(PropRoots, n/6, underRock)
		case RockIce:
			place(PropIcicles, n/7, byWall)
		case RockBasalt:
			place(PropObsidian, n/9, byWall)
		}
	}
}

// underFace reports whether cell i lies at the foot of a wall to the north,
// where a picture on the wall can be seen.
func (w *World) underFace(fl *caveFloor, i int) bool {
	tx, ty := fl.at(i)
	return fl.wall[i] == 1 && w.TileAt(tx, ty-1) == TileMountain && w.TileAt(tx, ty-2) == TileMountain &&
		w.TileAt(tx-1, ty-1) == TileMountain && w.TileAt(tx+1, ty-1) == TileMountain
}

// nextTo reports whether cell i of the level lies beside a tile of kind k.
func (w *World) nextTo(fl *caveFloor, i int, k Tile) bool {
	tx, ty := fl.at(i)
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if w.TileAt(tx+d[0], ty+d[1]) == k {
			return true
		}
	}
	return false
}

// The pictures on the cave walls.
const (
	PaintingHunt  = iota // hunters round a horned beast
	PaintingHoard        // a chest, and which way to go for it
	PaintingStar         // a star of fire falling on the island in the middle of the sea
	PaintingHerds        // the great beasts going from island to island
	numPaintings
)

// PaintingOf splits what a painting shows (its Fuel, the Arg of
// NoticePainting) into its picture and, for the hoard, which way the hoard
// lies: 0 north, then round by the east in eighths.
func PaintingOf(arg int) (picture, dir int) { return arg >> 3, arg & 7 }

// paintingOf chooses what painting e shows: now and then the way to its
// system's hoard at the end of the main route.
func (w *World) paintingOf(rng *rand.Rand, s *caveSystem, e *Entity) int {
	pic := rng.IntN(numPaintings)
	d := s.chambers[s.dest]
	_, row, _ := CaveLevelAt(w.height, w.surfaceH, e.TY)
	a := math.Atan2(d.x-(float64(e.TX)+0.5), -(d.y - (float64(row) + 0.5))) // 0 north, clockwise
	dir := int(math.Round(a/(math.Pi/4))+8) % 8
	return pic<<3 | dir
}

// settleThemes puts the life of the themed chambers in them: blind fish in
// the waters, crocodiles in the flooded caves, scorpions in the tomb and
// under the desert.
func (w *World) settleThemes(rng *rand.Rand, s *caveSystem, floors []*caveFloor) {
	f := &w.fauna
	add := func(k CreatureKind, spots []int32, n int) {
		if len(spots) == 0 || n <= 0 {
			return
		}
		f.spawnAt[k] = append(f.spawnAt[k], spots...)
		placed := 0
		for tries := 0; placed < n && tries < n*12; tries++ {
			i := spots[rng.IntN(len(spots))]
			x, y := float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5
			if w.creatureFits(k, x, y) {
				w.addCreature(k, x, y)
				placed++
			}
		}
		f.target[k] += placed
	}
	desert := w.islands[s.island].Kind == IslandDesert
	for ci, c := range s.chambers {
		fl := floors[c.level]
		var water, floor []int32
		for _, i := range fl.allOf(ci) {
			tx, ty := fl.at(i)
			switch t := w.TileAt(tx, ty); {
			case t == TileRiver:
				water = append(water, int32(ty*w.width+tx))
			case t.Walkable():
				floor = append(floor, int32(ty*w.width+tx))
			}
		}
		// The waters of the chamber, with the shore round them, for the fish.
		if len(water) > 0 {
			add(CreatureFish, water, max(1, len(water)/caveFishPer))
		}
		switch {
		case c.zone == ZoneFlooded:
			add(CreatureCroc, water, 1+int(c.r/12))
		case c.zone == ZoneTomb:
			add(CreatureScorpion, floor, 2+int(c.r/10))
		case desert && !c.hall && rng.Float64() < 0.3:
			add(CreatureScorpion, floor, 1)
		}
	}
}

// allOf lists the cells of chamber ci dug out, whatever they are now:
// floor, water, lava or ice.
func (fl *caveFloor) allOf(ci int) []int {
	var out []int
	for i, o := range fl.s.owner[fl.lv] {
		if int(o)-1 == ci && fl.s.open[fl.lv][i] {
			out = append(out, i)
		}
	}
	return out
}

// touchProp is a player's look at a cave prop: bones dug out of a fossil,
// the pictures on a wall read.
func (w *World) touchProp(p *Player, e *Entity) bool {
	switch CaveProp(e.Var) {
	case PropFossil:
		if !e.Ready {
			return false
		}
		e.Ready = false
		w.dropAt(float64(e.TX)+0.5, float64(e.TY)+0.5, []Ingredient{{Item: ItemBone, Count: caveFossilBones}})
		p.notify(NoticeFossil, uint8(ItemBone))
		return true
	case PropPainting:
		p.notify(NoticePainting, uint8(e.Fuel))
		return true
	}
	return false
}

// caveHazards is what the caves' webs and lava do to p: a web p walks into
// holds p a moment and is torn, lava beside p burns.
func (w *World) caveHazards(p *Player) {
	if p.boat != 0 {
		return
	}
	tx, ty := int(math.Floor(p.X)), int(math.Floor(p.Y))
	if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityCaveProp && CaveProp(e.Var) == PropWeb {
		p.webbed = WebTicks
		p.notify(NoticeWebbed, 0)
		w.removeEntity(e)
	}
	if w.tick%caveLavaBurnCD != 0 {
		return
	}
	for y := ty - 1; y <= ty+1; y++ {
		for x := tx - 1; x <= tx+1; x++ {
			if w.TileAt(x, y) != TileLava {
				continue
			}
			// How far p stands from the lava's edge.
			dx := math.Max(0, math.Max(float64(x)-p.X, p.X-float64(x+1)))
			dy := math.Max(0, math.Max(float64(y)-p.Y, p.Y-float64(y+1)))
			if math.Hypot(dx, dy) < PlayerRadius+0.3 {
				hurt(p, caveLavaBurn)
				if w.tick%(2*TicksPerSecond) == 0 || p.Notice != NoticeLavaBurn {
					p.notify(NoticeLavaBurn, 0)
				}
				return
			}
		}
	}
}
