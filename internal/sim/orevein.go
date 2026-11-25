package sim

import (
	"math"
	"math/rand/v2"
)

// Ore veins. Ores lie in clusters of a few blocks with wide bare ground
// between them, not sprinkled one by one: a player who finds one finds a
// handful. How common each is goes with its tier:
//
//   - flint, the first tier, in veins on the rocky hills and in small
//     outcrops in the grass and woods, and always a few near the spawn;
//   - jade deep in the caves, far from any way in;
//   - agate in the deserts, obsidian on the volcano's ash, both rare;
//   - coal in seams through the caves and the ash.
//
// In a world of 1024 by 1024 that is about 2,000 flint, 1,000–2,000 jade,
// 400 agate, 400 obsidian and 3,000–4,000 coal.

// veinDef is where one kind of ore lies and how its veins grow.
type veinDef struct {
	kind EntityKind
	on   []Tile  // tiles it lies on
	per  float64 // veins started per tile of those
	size [2]int  // blocks to a vein, least and most
}

var veins = []veinDef{
	{EntityCopperOre, []Tile{TileRocky}, 1.0 / 35, [2]int{3, 7}},
	{EntityCopperOre, []Tile{TileGrass, TileForest}, 1.0 / 500, [2]int{2, 4}}, // outcrops
	{EntityPlatinumOre, []Tile{TileDesert}, 1.0 / 950, [2]int{3, 6}},
	{EntityDiamondOre, []Tile{TileAsh}, 1.0 / 220, [2]int{3, 6}},
	{EntityCoal, []Tile{TileCave}, 1.0 / 900, [2]int{4, 8}},
	{EntityCoal, []Tile{TileAsh}, 1.0 / 200, [2]int{3, 6}},
}

// Jade and the flint by the spawn.
var (
	jadeVein = veinDef{EntityIronOre, []Tile{TileCave}, 1.0 / 1050, [2]int{3, 6}}
)

const (
	jadeDeep       = 30 // tiles of cave from any way in, at least, for jade
	homeFlintVeins = 3  // flint veins always near the spawn
	homeFlintNear  = 10 // tiles from the spawn, least...
	homeFlintFar   = 34 // ...and most
	homeFlintTries = 400
	minVeins       = 3 // of each kind, in a world however small
)

func onTile(t Tile, on []Tile) bool {
	for _, o := range on {
		if t == o {
			return true
		}
	}
	return false
}

// scatterVeins starts veins on the tiles of each def.
func (w *World) scatterVeins(rng *rand.Rand, defs []veinDef, ok func(tx, ty int) bool) {
	// A small world still gets a few veins of each: at least minVeins.
	per := make([]float64, len(defs))
	for k, v := range defs {
		n := 0
		for i, t := range w.tiles {
			if onTile(t, v.on) && (ok == nil || ok(i%w.width, i/w.width)) {
				n++
			}
		}
		per[k] = math.Max(v.per, minVeins/float64(max(1, n)))
	}
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			t := w.tiles[ty*w.width+tx]
			for k, v := range defs {
				if !onTile(t, v.on) {
					continue
				}
				if rng.Float64() < per[k] && (ok == nil || ok(tx, ty)) {
					w.growVein(rng, v, tx, ty)
				}
			}
		}
	}
}

// growVein lays a vein of v from (tx, ty), wandering from block to block.
func (w *World) growVein(rng *rand.Rand, v veinDef, tx, ty int) int {
	n := v.size[0] + rng.IntN(v.size[1]-v.size[0]+1)
	placed, x, y := 0, tx, ty
	for tries := 0; placed < n && tries < n*8; tries++ {
		if onTile(w.TileAt(x, y), v.on) && w.canPlace(x, y) {
			w.addEntity(v.kind, x, y)
			placed++
		}
		nx, ny := x+rng.IntN(3)-1, y+rng.IntN(3)-1
		if onTile(w.TileAt(nx, ny), v.on) {
			x, y = nx, ny
		}
	}
	return placed
}

// scatterDeepOres lays jade deep in the caves, once the ways in are known,
// and makes sure some flint lies near the spawn (sx, sy).
func (w *World) scatterDeepOres(rng *rand.Rand, sx, sy int) {
	depth := w.caveReach()
	deep := int32(0) // in small caves, the deeper half
	for _, d := range depth {
		deep = max(deep, d)
	}
	deep = min(jadeDeep, deep/2)
	w.scatterVeins(rng, []veinDef{jadeVein}, func(tx, ty int) bool {
		d := depth[ty*w.width+tx]
		return d >= deep
	})
	for v, tries := 0, 0; v < homeFlintVeins && tries < homeFlintTries; tries++ {
		a := rng.Float64() * 2 * math.Pi
		r := homeFlintNear + rng.Float64()*(homeFlintFar-homeFlintNear)
		x, y := sx+int(math.Cos(a)*r), sy+int(math.Sin(a)*r*0.8)
		t := w.TileAt(x, y)
		if t != TileGrass && t != TileForest && t != TileRocky || !w.canPlace(x, y) {
			continue
		}
		if w.growVein(rng, veinDef{EntityCopperOre, []Tile{t}, 0, [2]int{3, 5}}, x, y) >= 2 {
			v++
		}
	}
}

// caveReach is, for each cave tile, how many tiles of cave lie between it
// and the nearest way in (a cave tile by open ground, or a way up out of the
// caves); -1 for tiles that are no cave.
func (w *World) caveReach() []int32 {
	d := make([]int32, len(w.tiles))
	var queue []int32
	for i := range d {
		d[i] = -1
		if w.tiles[i] != TileCave {
			continue
		}
		tx, ty := i%w.width, i/w.width
		if e := w.entityAt(tx, ty); e != nil && (e.Kind == EntityCaveExit || e.Kind == EntityCaveMouth) {
			d[i] = 0
			queue = append(queue, int32(i))
			continue
		}
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if t := w.TileAt(tx+n[0], ty+n[1]); t != TileCave && t.Walkable() {
				d[i] = 0
				queue = append(queue, int32(i))
				break
			}
		}
	}
	for len(queue) > 0 {
		i := int(queue[0])
		queue = queue[1:]
		tx, ty := i%w.width, i/w.width
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			x, y := tx+n[0], ty+n[1]
			if x < 0 || y < 0 || x >= w.width || y >= w.height {
				continue
			}
			j := y*w.width + x
			if w.tiles[j] == TileCave && d[j] < 0 {
				d[j] = d[i] + 1
				queue = append(queue, int32(j))
			}
		}
	}
	return d
}

// oreLiesOn reports whether ore of kind k may lie on tile t.
func oreLiesOn(k EntityKind, t Tile) bool {
	if k == jadeVein.kind {
		return onTile(t, jadeVein.on)
	}
	for _, v := range veins {
		if v.kind == k && onTile(t, v.on) {
			return true
		}
	}
	return false
}
