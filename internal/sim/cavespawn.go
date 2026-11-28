package sim

import (
	"math"
	"math/rand/v2"
)

// Where the cave beasts are (StyleCaveFight): see cavefight.go. A few
// encounters to a chamber or a stretch of tunnel, not a crowd; more of them,
// and worse, on the lower level. Each island kind has its own beasts. Every
// nest chamber has its queen, with her egg sacs and webs and her spiders
// about her; the destination at the end of the main route has its boss
// standing on the hoard, and the fossil chambers their skeletons.

type caveGroup uint8

const (
	groupMeadow caveGroup = iota
	groupJungle
	groupDesert
	groupVolcano
	groupSnow
	groupMarsh
	numCaveGroups
)

func groupOf(k IslandKind) caveGroup {
	switch k {
	case IslandForest, IslandMonkey:
		return groupJungle
	case IslandDesert:
		return groupDesert
	case IslandAsh:
		return groupVolcano
	case IslandSnow:
		return groupSnow
	case IslandMarsh:
		return groupMarsh
	}
	return groupMeadow
}

// CaveQueenOf is the spider queen of the caves under an island of kind k.
func CaveQueenOf(k IslandKind) CreatureKind {
	switch groupOf(k) {
	case groupDesert:
		return CreatureQueenDesert
	case groupMarsh:
		return CreatureQueenMarsh
	case groupSnow:
		return CreatureQueenSnow
	case groupVolcano:
		return CreatureQueenVolcano
	}
	return CreatureQueenMeadow
}

// CaveBossOf is the boss guarding the deep hoard under an island of kind k.
func CaveBossOf(k IslandKind) CreatureKind {
	switch groupOf(k) {
	case groupJungle:
		return CreatureArthroQueen
	case groupDesert:
		return CreatureScorpQueen
	case groupVolcano:
		return CreatureMegalania
	case groupSnow:
		return CreatureLionAlpha
	case groupMarsh:
		return CreatureBlindCroc
	}
	return CreatureAlphaBear
}

// caveEnc is one kind of encounter a chamber or a tunnel may hold.
type caveEnc struct {
	kind   CreatureKind
	lv     [2]int // the levels it may be on, from and to
	weight int
	pile   bool // skeletons: it lies as bones
}

var caveEncounters = [numCaveGroups][]caveEnc{
	groupMeadow: {
		{CreatureHyena, [2]int{0, 1}, 5, false},
		{CreatureWhipScorp, [2]int{0, 1}, 3, false},
		{CreatureSkelRaptor, [2]int{1, 1}, 3, true},
		{CreatureHyena, [2]int{1, 1}, 3, false},
	},
	groupJungle: {
		{CreatureArthro, [2]int{0, 1}, 5, false},
		{CreatureCentipede, [2]int{0, 1}, 4, false},
		{CreatureAntSoldier, [2]int{0, 1}, 4, false},
		{CreatureHyena, [2]int{1, 1}, 2, false},
	},
	groupDesert: {
		{CreatureWhipScorp, [2]int{0, 1}, 5, false},
		{CreatureSkelRaptor, [2]int{0, 1}, 4, true},
		{CreatureSkelTrike, [2]int{1, 1}, 3, true},
		{CreatureCentipede, [2]int{1, 1}, 2, false},
	},
	groupVolcano: {
		{CreatureMonitor, [2]int{0, 1}, 5, false},
		{CreatureWhipScorp, [2]int{0, 1}, 3, false},
		{CreatureSkelRaptor, [2]int{1, 1}, 3, true},
		{CreatureCentipede, [2]int{1, 1}, 2, false},
	},
	groupSnow: {
		{CreatureHyena, [2]int{0, 1}, 5, false},
		{CreatureCaveLion, [2]int{1, 1}, 2, false},
		{CreatureHyena, [2]int{1, 1}, 3, false},
		{CreatureSkelTrike, [2]int{1, 1}, 2, true},
	},
	groupMarsh: {
		{CreatureCentipede, [2]int{0, 1}, 5, false},
		{CreatureWhipScorp, [2]int{0, 1}, 3, false},
		{CreatureHyena, [2]int{1, 1}, 2, false},
		{CreatureSkelRaptor, [2]int{1, 1}, 2, true},
	},
}

// Cave spawn tuning.
const (
	caveEncChance    = 0.38 // of a chamber of no part of its own, on the upper level...
	caveEncChanceLow = 0.62 // ...and on the lower
	caveAmbushPer    = 240  // tunnel cells to each ambush
	caveAmbushApart  = 14.0
	caveSacsMin      = 3
	caveVampChance   = 0.4
	caveAntNest      = 0.5 // of the jungle's plain chambers on the upper level
)

// settleCaveFight puts the fighting beasts in the caves of s.
func (w *World) settleCaveFight(s *caveSystem, floors []*caveFloor) {
	rng := rand.New(rand.NewPCG(w.seed, 0xca5e0000+uint64(s.island)))
	f := &w.fauna
	grp := groupOf(w.islands[s.island].Kind)
	spotsOf := func(fl *caveFloor, ci int) []int32 {
		var out []int32
		for _, i := range fl.cellsOf(ci) {
			if fl.wall[i] >= 2 {
				tx, ty := fl.at(i)
				out = append(out, int32(ty*w.width+tx))
			}
		}
		return out
	}
	// place puts n creatures of kind k on random spots (a group keeps
	// together round the first); reg records the spots so the cave tops
	// itself up.
	place := func(k CreatureKind, spots []int32, n int, reg, pile bool) []*Creature {
		if len(spots) == 0 || n <= 0 {
			return nil
		}
		var made []*Creature
		var hx, hy float64
		for tries := 0; len(made) < n && tries < n*14; tries++ {
			i := spots[rng.IntN(len(spots))]
			x, y := float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5
			if len(made) > 0 {
				a, d := rng.Float64()*2*math.Pi, 0.9+rng.Float64()*1.6
				x, y = hx+math.Cos(a)*d, hy+math.Sin(a)*d
			}
			if !w.creatureFits(k, x, y) {
				continue
			}
			c := w.addCreature(k, x, y)
			if len(made) == 0 {
				hx, hy = x, y
			}
			c.homeX, c.homeY = hx, hy
			if pile {
				c.State = CreatureSleep
			}
			made = append(made, c)
		}
		if reg && len(made) > 0 {
			f.spawnAt[k] = append(f.spawnAt[k], spots...)
			f.target[k] += len(made)
		}
		return made
	}
	pick := func(lv int) *caveEnc {
		tot := 0
		for i := range caveEncounters[grp] {
			if e := &caveEncounters[grp][i]; lv >= e.lv[0] && lv <= e.lv[1] {
				tot += e.weight
			}
		}
		if tot == 0 {
			return nil
		}
		r := rng.IntN(tot)
		for i := range caveEncounters[grp] {
			e := &caveEncounters[grp][i]
			if lv < e.lv[0] || lv > e.lv[1] {
				continue
			}
			if r -= e.weight; r < 0 {
				return e
			}
		}
		return nil
	}
	encounter := func(spots []int32, lv int) {
		e := pick(lv)
		if e == nil {
			return
		}
		n := 1
		if d := e.kind.def(); d.herd > 1 {
			n = max(1, d.herdMin) + rng.IntN(d.herd-max(1, d.herdMin)+1)
		}
		place(e.kind, spots, n, !e.pile, e.pile)
	}
	bossDone, queenDone, vampDone, antDone := false, false, false, false
	for ci, c := range s.chambers {
		if c.junction || c.hall {
			continue
		}
		fl := floors[c.level]
		spots := spotsOf(fl, ci)
		if len(spots) == 0 {
			continue
		}
		switch {
		case ci == s.dest:
			if !bossDone {
				bossDone = w.placeCaveBoss(rng, fl, ci, CaveBossOf(w.islands[s.island].Kind))
			}
		case c.zone == ZoneNest:
			if !queenDone {
				queenDone = true
				w.placeQueen(rng, s, fl, ci, spots, place)
			}
		case c.zone == ZoneFossil:
			if rex := place(CreatureSkelRex, spots, 1, false, true); len(rex) > 0 {
				place(CreatureSkelRaptor, spots, 2, false, true)
			}
		case c.zone == ZoneRoost:
			place(CreatureBat, spots, 5, true, false)
		case c.zone == ZoneLake || c.zone == ZoneStream || c.zone == ZoneFlooded:
			var shore []int32
			for _, i := range fl.allOf(ci) {
				if fl.wall[i] >= 1 {
					tx, ty := fl.at(i)
					if w.TileAt(tx, ty).Walkable() {
						shore = append(shore, int32(ty*w.width+tx))
					}
				}
			}
			place(CreatureOlm, shore, 1+rng.IntN(2), true, false)
			place(CreatureCaveCrab, shore, 1+rng.IntN(2), true, false)
		case c.zone == ZoneHall:
			if c.level == 1 && s.levels == 2 && !vampDone && !c.route && rng.Float64() < caveVampChance {
				vampDone = true
				place(CreatureVampBat, spots, 1, false, false)
				break
			}
			if grp == groupJungle && c.level == 0 && !antDone && rng.Float64() < caveAntNest {
				antDone = true
				place(CreatureAnt, spots, 4, true, false)
				place(CreatureAntSoldier, spots, 2, true, false)
				break
			}
			chance := caveEncChance
			if c.level == 1 {
				chance = caveEncChanceLow
			}
			if rng.Float64() < chance {
				encounter(spots, c.level)
			}
		}
		if c.r >= collapseMinR && ci != s.dest && rng.Float64() < collapseChance {
			w.cf.spots = append(w.cf.spots, collapseSpot{x: c.x, y: float64(fl.top) + c.y, r: c.r})
		}
	}
	// Ambush points in the tunnels.
	for lv, fl := range floors {
		var cells []int32
		for _, i := range fl.cellsOf(-1) {
			if fl.wall[i] >= 2 {
				tx, ty := fl.at(i)
				cells = append(cells, int32(ty*w.width+tx))
			}
		}
		n := len(cells) / caveAmbushPer * (1 + lv)
		var at [][2]float64
		for tries := 0; len(at) < n && tries < n*12 && len(cells) > 0; tries++ {
			i := cells[rng.IntN(len(cells))]
			x, y := float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5
			near := false
			for _, q := range at {
				if math.Hypot(q[0]-x, q[1]-y) < caveAmbushApart {
					near = true
					break
				}
			}
			if near {
				continue
			}
			at = append(at, [2]float64{x, y})
			encounter([]int32{i}, lv)
		}
	}
}

// placeQueen puts a spider queen in her nest chamber ci, with her egg sacs
// along the walls and a web or two on the floor.
func (w *World) placeQueen(rng *rand.Rand, s *caveSystem, fl *caveFloor, ci int, spots []int32, place func(CreatureKind, []int32, int, bool, bool) []*Creature) {
	q := place(CaveQueenOf(w.islands[s.island].Kind), spots, 1, false, false)
	if len(q) == 0 {
		return
	}
	var wall []int32
	for _, i := range fl.cellsOf(ci) {
		if fl.wall[i] == 2 {
			tx, ty := fl.at(i)
			wall = append(wall, int32(ty*w.width+tx))
		}
	}
	if len(wall) == 0 {
		wall = spots
	}
	place(CreatureEggSac, wall, caveSacsMin+rng.IntN(3), false, false)
	for range 2 {
		i := spots[rng.IntN(len(spots))]
		w.addHazard(HazWeb, float64(int(i)%w.width)+0.5, float64(int(i)/w.width)+0.5, 1.6, -1)
	}
}

// placeCaveBoss stands boss k on the hoard of chamber ci: the chest stays
// shut while it lives.
func (w *World) placeCaveBoss(rng *rand.Rand, fl *caveFloor, ci int, k CreatureKind) bool {
	var chest *Entity
	for _, i := range fl.cellsOf(ci) {
		tx, ty := fl.at(i)
		if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityTreasure {
			chest = e
			break
		}
	}
	if chest == nil {
		return false
	}
	cx, cy := float64(chest.TX)+0.5, float64(chest.TY)+0.5
	for tries := 0; tries < 60; tries++ {
		a, d := rng.Float64()*2*math.Pi, 1.2+float64(tries/10)*0.6
		x, y := cx+math.Cos(a)*d, cy+math.Sin(a)*d
		if !w.creatureFits(k, x, y) {
			continue
		}
		c := w.addCreature(k, x, y)
		c.homeX, c.homeY = cx, cy
		c.cv.guard = chest.ID
		return true
	}
	return false
}

// caveQueenFrom is the size of a cave that always has a spider queen's nest.
const caveQueenFrom = 1800

// ensureNest makes a nest of one plain chamber of a big cave that has none,
// so every such cave has its queen. It draws from its own stream.
func (s *caveSystem) ensureNest(seed uint64) {
	if s.area < caveQueenFrom {
		return
	}
	var plain []int
	for i, c := range s.chambers {
		if c.zone == ZoneNest {
			return
		}
		if c.zone == ZoneHall && !c.hall && !c.route && !c.junction && !c.leaf && c.pocket == 0 && i != s.dest && i != s.descent && i != s.arrival && !s.shaft(i) {
			plain = append(plain, i)
		}
	}
	if len(plain) == 0 {
		return
	}
	r := rand.New(rand.NewPCG(seed, 0x9ee))
	s.chambers[plain[r.IntN(len(plain))]].zone = ZoneNest
}
