package sim

import "math"

// The bosses' lairs since StyleWilds. Each boss of the land lives on the
// biggest island of its kind, deep in its heart where there is room: the
// way to Titanoboa is a long trek into ever thicker jungle (the biggest
// island of all; see ringIsles), past the signs of it the lair's dressing
// leaves (lair.go). The Megalodon lives out in the open sea kept for it
// (deepBasin): deep, dark water all round, and no coast within reach of
// its whirlpool. Where none of that can be had, the lair is sited as
// before (bossLairSite).

// bossIsleKind is the kind of island each boss of the land lives on.
var bossIsleKind = map[CreatureKind]IslandKind{
	CreatureTitanoboa: IslandForest, CreatureSpino: IslandMarsh, CreatureMammothKing: IslandSnow,
	CreatureCarcharo: IslandDesert, CreatureGiganoto: IslandAsh,
}

// Lair tuning since StyleWilds.
const (
	wildsLairCore = 26 // tiles of land at least between a land boss's lair and the sea, where there is room
	wildsMegaSea  = 30 // tiles of open sea at least round the Megalodon's lair: its whirlpool's ring and ten more
	wildsMegaWant = 50 // and how much it would like, from land and from the edge of the world
)

// bossIsland is the index in w.islands of the biggest island of the kind
// boss k lives on, -1 if there is none.
func (w *World) bossIsland(k CreatureKind) int {
	kind, ok := bossIsleKind[k]
	if !ok {
		return -1
	}
	best := -1
	for i, is := range w.islands {
		if is.Kind == kind && !is.Home && (best < 0 || is.Tiles > w.islands[best].Tiles) {
			best = i
		}
	}
	return best
}

// wildsLairSite is where boss k lives in a world made since StyleWilds;
// ok is false where the island or the sea it wants has no room for it.
func (w *World) wildsLairSite(k CreatureKind) (x, y float64, ok bool) {
	W, H := w.width, w.SurfaceHeight()
	size := float64(max(W, H))
	step := 2
	if W*H > 600*600 {
		step = 3
	}
	if k == CreatureMegalodon {
		off := w.coastDistance(60)
		best := math.Inf(-1)
		edge := 16 + int(MegaArena)
		for ty := edge; ty < H-edge; ty += step {
			for tx := edge; tx < W-edge; tx += step {
				i := ty*W + tx
				cx, cy := float64(tx)+0.5, float64(ty)+0.5
				if w.tiles[i] != TileWater || int(off[i]) < wildsMegaSea || math.Hypot(cx-w.spawnX, cy-w.spawnY) < size/5 || !w.creatureFits(k, cx, cy) {
					continue
				}
				room := min(int(off[i]), tx, ty, W-1-tx, H-1-ty) // the edge of the world is no open sea to sail
				score := float64(min(room, wildsMegaWant)) + unit(hash2(int32(tx), int32(ty), uint32(w.seed)^(0xb055+uint32(k))))
				if score > best {
					best, x, y, ok = score, cx, cy, true
				}
			}
		}
		return x, y, ok
	}
	isl := w.bossIsland(k)
	if isl < 0 {
		return 0, 0, false
	}
	im := w.islandIndex()
	inland := w.landDistance(60)
	on := func(tx, ty int) bool {
		if w.inBounds(tx, ty) && ty < H && im[ty*W+tx] == int16(isl) {
			return true
		}
		if k != CreatureSpino {
			return false
		}
		for dy := -3; dy <= 3; dy++ { // a pond of the island
			for dx := -3; dx <= 3; dx++ {
				if nx, ny := tx+dx, ty+dy; w.inBounds(nx, ny) && ny < H && im[ny*W+nx] == int16(isl) {
					return true
				}
			}
		}
		return false
	}
	for _, lv := range []struct {
		core    int
		village float64
	}{{wildsLairCore, 14}, {10, 7}} {
		best := math.Inf(-1)
		for ty := 6; ty < H-6; ty += step {
			for tx := 6; tx < W-6; tx += step {
				i := ty*W + tx
				cx, cy := float64(tx)+0.5, float64(ty)+0.5
				core := int(inland[i])
				if k != CreatureSpino && core < lv.core || !on(tx, ty) || math.Hypot(cx-w.spawnX, cy-w.spawnY) < size/5 {
					continue
				}
				if k == CreatureCarcharo && w.hasPyramid && math.Hypot(float64(tx-w.pyramidAt[0]), float64(ty-w.pyramidAt[1])) < 16 {
					continue // the scorpions keep the pyramid
				}
				score, good := w.lairScore(k, tx, ty, nil, 1, 0)
				if !good || w.nearVillage(cx, cy, lv.village) || !w.creatureFits(k, cx, cy) {
					continue
				}
				score += 0.4*float64(min(core, 50)) + unit(hash2(int32(tx), int32(ty), uint32(w.seed)^(0xb055+uint32(k))))
				if score > best {
					best, x, y, ok = score, cx, cy, true
				}
			}
		}
		if ok {
			return x, y, true
		}
	}
	return 0, 0, false
}

// landDistance is, for every tile of the surface, how many steps it lies
// from the sea or a lake, up to most; 0 on the water.
func (w *World) landDistance(most int) []uint8 {
	W, H := w.width, w.SurfaceHeight()
	d := make([]uint8, W*H)
	var q []int32
	for i := range d {
		if w.tiles[i].Sea() {
			q = append(q, int32(i))
		} else {
			d[i] = uint8(most)
		}
	}
	for k := 0; k < len(q); k++ {
		i := int(q[k])
		x, y := i%W, i/W
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+n[0], y+n[1]
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
