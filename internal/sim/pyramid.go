package sim

import (
	"math"
	"math/rand/v2"
)

// The pyramid. In an archipelago one desert island, never the home island,
// has an ancient pyramid standing in its sands: a great block of stone 7
// tiles wide and 5 deep, with a dark doorway in the middle of its south
// face. An old chest lies in the doorway, holding the best hoard in the
// world, and giant scorpions guard it.
//
// Giant scorpions live on the desert's sand. Slower than a player, they
// come at anyone close, pinch with their claws and then sting with the
// tail; the sting poisons, like a cave spider's bite.

// Pyramid tuning.
const (
	PyramidW, PyramidD = 7, 5 // footprint in tiles: wide, deep
	pyramidGuards      = 4    // scorpions round it at first
	pyramidDepth       = 60   // the hoard is as rich as a chest this deep in a cave

	ScorpionSight  = 6.5
	ScorpionGiveUp = 11.0
	ScorpionReach  = 1.5
	ScorpionPinch  = 10.0
	ScorpionSting  = 14.0
	ScorpionCD     = 7 * TicksPerSecond / 5
	ScorpionVenom  = 8 * TicksPerSecond
)

var scorpionDef = creatureDef{
	name: "Bọ cạp khổng lồ", hp: 35,
	walk: 1.0 / TicksPerSecond, run: 3.3 / TicksPerSecond, // a player outruns it
	radius: 0.55, reach: 2.0,
	yield: ItemMeat, yieldN: 3, drops: []Ingredient{{ItemBone, 2}},
	leash: 10, idle: [2]int{40, 160}, roam: [2]int{20, 60},
	spawnOn: []Tile{TileDesert}, tilesPer: 260,
	fights: true, tough: true, openOnly: true, away: 30,
}

func init() {
	creatureDefs[CreatureScorpion] = scorpionDef
	nameHead(CreatureScorpion)
}

// Pyramid is where the world's pyramid stands: the tile of the middle of
// its south face (its doorway). ok is false when there is none.
func (w *World) Pyramid() (tx, ty int, ok bool) {
	for _, e := range w.entities {
		if e.Kind == EntityPyramid {
			return e.TX, e.TY, true
		}
	}
	return 0, 0, false
}

// raisePyramid stands the pyramid on the biggest desert island away from
// home, on open sand, with its chest in the doorway and its guards round it:
// where pickIslandVillages planned it, kept clear of villages, or else where
// pyramidSite finds now.
func (w *World) raisePyramid(rng *rand.Rand) {
	cx, cy, ok := w.pyramidAt[0], w.pyramidAt[1], w.pyramidPlanned
	if !ok {
		cx, cy, ok = w.pyramidSite(w.spawnX, w.spawnY)
	}
	if !ok {
		return
	}
	w.buildPyramid(cx, cy)
	_ = rng
}

// pyramidSite is where the pyramid would stand, by the lie of the land
// alone: deep in the desert, far from (sx, sy).
func (w *World) pyramidSite(sx, sy float64) (cx, cy int, ok bool) {
	fits := func(cx, cy int) bool {
		// The footprint and a tile round it all desert, and a clear tile in
		// front of the doorway.
		for ty := cy - PyramidD; ty <= cy+2; ty++ {
			for tx := cx - PyramidW/2 - 1; tx <= cx+PyramidW/2+1; tx++ {
				if t := w.TileAt(tx, ty); t != TileDesert && t != TileSand && t != TileRocky {
					return false
				}
			}
		}
		return true
	}
	best, bestScore := -1, math.Inf(-1)
	for i, t := range w.tiles[:w.width*w.SurfaceHeight()] {
		if t != TileDesert {
			continue
		}
		cx, cy := i%w.width, i/w.width
		d := math.Hypot(float64(cx)+0.5-sx, float64(cy)+0.5-sy)
		if d < float64(max(w.width, w.SurfaceHeight()))/5 || !fits(cx, cy) {
			continue
		}
		// Deep in the desert: as much desert round it as can be.
		score := 0.0
		for ty := cy - 10; ty <= cy+8; ty += 2 {
			for tx := cx - 12; tx <= cx+12; tx += 2 {
				if w.TileAt(tx, ty) == TileDesert {
					score++
				}
			}
		}
		score += unit(hash2(int32(cx), int32(cy), 0x9f1d)) // ties: somewhere
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	return best % w.width, best / w.width, true
}

// buildPyramid stands the pyramid with the middle of its front on (cx, cy).
func (w *World) buildPyramid(cx, cy int) {
	for ty := cy - PyramidD - 1; ty <= cy+3; ty++ {
		for tx := cx - PyramidW/2 - 2; tx <= cx+PyramidW/2+2; tx++ {
			if e := w.entityAt(tx, ty); e != nil {
				w.removeEntity(e)
			}
		}
	}
	for ty := cy - PyramidD + 1; ty <= cy; ty++ {
		for tx := cx - PyramidW/2; tx <= cx+PyramidW/2; tx++ {
			if tx == cx && ty == cy {
				continue
			}
			w.addEntity(EntityPyramidStone, tx, ty)
		}
	}
	w.addEntity(EntityPyramid, cx, cy)
	chest := w.addEntity(EntityTreasure, cx, cy+1) // in the doorway
	chest.Var = pyramidDepth
	w.pyramidAt = [2]int{cx, cy}
	w.hasPyramid = true
}

// guardPyramid sets scorpions round the pyramid.
func (w *World) guardPyramid() {
	if !w.hasPyramid {
		return
	}
	cx, cy := float64(w.pyramidAt[0])+0.5, float64(w.pyramidAt[1])+0.5
	r := w.random()
	for n, tries := 0, 0; n < pyramidGuards && tries < 60; tries++ {
		a, d := r.Float64()*math.Pi+0.1, 3+r.Float64()*4 // round the front
		x, y := cx+math.Cos(a)*d*1.4, cy+math.Sin(a)*d
		if w.creatureFits(CreatureScorpion, x, y) {
			c := w.addCreature(CreatureScorpion, x, y)
			c.homeX, c.homeY = cx, cy+2
			n++
		}
	}
}

func (w *World) stepScorpion(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	if c.State == CreatureStrike || c.State == CreatureClaw {
		if c.timer--; c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	p, dist := w.foeOf(c)
	if p == nil {
		p, dist = w.nearestPlayer(c.X, c.Y)
	}
	if p == nil || dist > ScorpionGiveUp || (c.State != CreatureChase && dist > ScorpionSight) || w.Swimming(p) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		w.wander(c)
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= ScorpionReach {
		if c.biteCD == 0 {
			// The claws, then the sting, turn about.
			if c.ang == 0 {
				strike(c, p, ScorpionPinch, ScorpionCD)
				c.State, c.ang = CreatureClaw, 1
			} else {
				strike(c, p, ScorpionSting, ScorpionCD)
				p.poison = max(p.poison, ScorpionVenom)
				c.ang = 0
			}
		}
		return
	}
	w.runAt(c, p.X, p.Y, d.run, ScorpionReach*0.8)
}
