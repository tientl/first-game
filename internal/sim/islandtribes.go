package sim

import (
	"math"
	"math/rand/v2"
)

// The tribes of the archipelago. Every island but the smallest islets has a
// village, whatever grows on it: huts in a clearing of the woods, on the
// meadow, in the dunes, on the snow or in the ash. A big island has two or
// three; a small or narrow one makes do with a hamlet of three huts. On the
// home island the village keeps well away from where players start, so a
// new arrival has time to find their feet.

// Island village tuning.
const (
	villageIslandMin  = 90   // tiles of land an island needs for a hamlet at least
	villagePerTiles   = 4000 // one more village per this many tiles
	villagesPerIsland = 3
	villageApart      = 30 // tiles between two villages of one island at least
	homeVillageAway   = 22 // tiles from the spawn, on the home island
	hamletRadius      = 4  // tiles cleared for a hamlet
	hamletHuts        = 3
	hamletFolk        = 2
	smallIslandFolk   = 3 // warriors in a village on an island under bigIslandTiles
	bigIslandTiles    = 600
)

// pickIslandVillages puts villages on every island with room for one.
func (w *World) pickIslandVillages(rng *rand.Rand, sx, sy int) {
	W := w.width
	// The pyramid goes up later, but its place is known now: keep clear of it.
	px, py, pyramid := w.pyramidSite(float64(sx)+0.5, float64(sy)+0.5)
	if pyramid {
		w.pyramidAt, w.pyramidPlanned = [2]int{px, py}, true
	}
	clearOfPyramid := func(tx, ty, r int) bool {
		return !pyramid || math.Hypot(float64(tx-px), float64(ty-py)+float64(PyramidD)/2) >= float64(r+PyramidD+1) // from the middle of its foot
	}
	// fits reports whether a clearing of radius r fits round (tx, ty): land
	// all round, nothing burning underfoot.
	fits := func(tx, ty, r int) bool {
		for y := ty - r; y <= ty+r; y++ {
			for x := tx - r; x <= tx+r; x++ {
				if (x-tx)*(x-tx)+(y-ty)*(y-ty) > r*r {
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
		home := false
		for _, i := range isl {
			if i%W == sx && i/W == sy {
				home = true
				break
			}
		}
		folk := villageFolk
		if len(isl) < bigIslandTiles {
			folk = smallIslandFolk
		}
		want := min(villagesPerIsland, 1+len(isl)/villagePerTiles)
		spots := append([]int(nil), isl...)
		rng.Shuffle(len(spots), func(a, b int) { spots[a], spots[b] = spots[b], spots[a] })
		var here []Village
		apart := func(tx, ty int) bool {
			for _, v := range here {
				if math.Hypot(v.X-float64(tx), v.Y-float64(ty)) < villageApart {
					return false
				}
			}
			return true
		}
		allowed := func(tx, ty int) bool {
			return !home || math.Hypot(float64(tx-sx), float64(ty-sy)) >= homeVillageAway
		}
		for _, i := range spots {
			if len(here) == want {
				break
			}
			tx, ty := i%W, i/W
			if allowed(tx, ty) && apart(tx, ty) && clearOfPyramid(tx, ty, villageRadius) && fits(tx, ty, villageRadius-1) {
				here = append(here, Village{X: float64(tx) + 0.5, Y: float64(ty) + 0.5, Folk: folk})
			}
		}
		if len(here) == 0 { // too small or too narrow for a village: a hamlet
			for _, i := range spots {
				if tx, ty := i%W, i/W; allowed(tx, ty) && clearOfPyramid(tx, ty, hamletRadius) && fits(tx, ty, hamletRadius-1) {
					here = append(here, Village{X: float64(tx) + 0.5, Y: float64(ty) + 0.5, Folk: hamletFolk, Small: true})
					break
				}
			}
		}
		w.villages = append(w.villages, here...)
	}
}

// radius is how far round its post a village is cleared.
func (v Village) radius() int {
	if v.Small {
		return hamletRadius
	}
	return villageRadius
}

// inVillage reports whether (tx, ty) is within margin tiles of a village's
// clearing.
func (w *World) inVillage(tx, ty, margin int) bool {
	for _, v := range w.villages {
		if math.Hypot(v.X-float64(tx)-0.5, v.Y-float64(ty)-0.5) <= float64(v.radius()+margin) {
			return true
		}
	}
	return false
}
