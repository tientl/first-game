package sim

import "math"

// The zones of an island, since StyleWilds. An island keeps the character
// of its kind over most of its land, half to two thirds of it, and has a few
// other kinds of ground besides, each where it belongs: wetter low down by
// the coast, rockier and colder up high, greener where a hot island meets
// the sea. The borders wander, and the client blends them wide (see
// wetAndSoft).
//
//   - The meadows (the home island among them): grass, with woods and
//     copses, rocky outcrops on the hills, and a fen by the coast here and
//     there, where a river comes out.
//   - The jungle: dense woods, with meadows and clearings, rocky mountain
//     slopes, and a swampy coast in places. Its heart is all woods.
//   - The delta: swamp and ponds in the low middle, wooded hummocks, and wet
//     meadows round its edges.
//   - The desert: dunes, rocky mesas, an oasis or two (a pond in a ring of
//     grass), and a fringe of dry grassland with scattered trees by the coast.
//   - The volcano: ash and rock round the crater, and jungle on its lower
//     slopes down to the black sand.
//   - The snowy range: snowfields up high, frosted pinewoods on the slopes,
//     and bare tundra by the coast, out of the worst of the cold.

// zoneNoise is the noise paint has worked out at a tile already.
type zoneNoise struct{ woods, wet, fine, rock, crag float64 }

// zone is the ground of one of island s's other zones at (x, y), if the
// tile lies in one: ok false leaves it to the island's own ground (paint).
func (s *isle) zone(h float64, level, sea int, x, y float64, seed uint32, n zoneNoise) (Tile, GroundVariant, bool) {
	vr := min(s.r, wildsHills)
	jit := 0.04 * (unit(hash2(int32(x), int32(y), seed+0x20e1)) - 0.5)
	z1 := fbm(x/(vr*0.42)+31, y/(vr*0.42)+17, s.seed+0x20e, 3) + jit // big patches
	z2 := fbm(x/(vr*0.3)+5, y/(vr*0.3)+41, s.seed+0x20f, 3) + jit
	plain := s.flat[2] > 0 && math.Hypot(x-s.flat[0], y-s.flat[1]) < s.flat[2]+10 // round the spawn, the pyramid's plain
	switch s.kind {
	case IslandMeadow:
		switch {
		case !plain && h < 0.28 && sea >= 3 && sea <= 18 && z2 > 0.58:
			if sea <= 5 {
				return TileMarsh, VarMudflat, true
			}
			return TileMarsh, VarPlain, true // a fen
		case !plain && h > 0.5 && z2 < 0.42 && n.crag > 0.52:
			return TileRocky, VarOutcrop, true // a rocky hilltop
		}
	case IslandForest, IslandMonkey:
		core := s.kind == IslandForest && s.role == roleLarge && math.Hypot(x-s.x, y-s.y) < wildsJungleHeart*s.r // the heart of the jungle stays woods
		clearings := 0.61
		if s.kind == IslandMonkey {
			clearings = 0.7 // the monkeys' jungle is thicker
		}
		switch {
		case level >= 2 && z2 > 0.56 && n.rock > 0.42:
			if n.fine > 0.6 {
				return TileRocky, VarScree, true
			}
			return TileRocky, VarPlain, true // a mountain's rocky slope
		case !core && z1 > clearings:
			if n.fine > 0.62 {
				return TileGrass, VarClearing, true
			}
			return TileGrass, VarMeadow, true
		case sea <= 7 && h < 0.26 && z2 < 0.38:
			if sea <= 3 {
				return TileMarsh, VarMudflat, true
			}
			return TileMarsh, VarPlain, true // a swampy coast
		case core && z1 < 0.74:
			if n.woods > 0.4 {
				return TileForest, VarDenseWood, true
			}
			return TileForest, VarPlain, true // the heart of the jungle: woods all over, thicker in
		}
	case IslandMarsh:
		switch {
		case sea >= 3 && sea <= 14 && z1 > 0.58:
			return TileGrass, VarMeadow, true // a wet meadow at the edge
		case level == 0 && h > 0.22 && z2 > 0.66:
			return TileForest, VarOpenWood, true // a wooded hummock
		}
	case IslandDesert:
		if !plain {
			if t, v, ok := s.oasis(x, y, n); ok {
				return t, v, true
			}
			if sea >= 3 && sea <= 26 && level == 0 && z1 > 0.35 {
				if n.rock > 0.72 {
					return TileGrass, VarBoulders, true
				}
				return TileGrass, VarPlain, true // dry grassland, with a tree here and there
			}
		}
	case IslandAsh:
		switch {
		case h < 0.3 && sea >= 2 && z1 > 0.47:
			if n.woods > 0.55 {
				return TileForest, VarDenseWood, true
			}
			return TileForest, VarPlain, true // jungle on the lower slopes
		case h < 0.27 && sea >= 2 && z1 > 0.41:
			return TileGrass, VarMeadow, true
		}
	case IslandSnow:
		switch {
		case level <= 1 && h > 0.12 && h < 0.55 && z1 > 0.52:
			return TileSnow, VarDenseWood, true // frosted pinewoods
		case sea <= 14 && h < 0.22 && z2 > 0.5:
			if n.rock > 0.6 {
				return TileRocky, VarPlain, true
			}
			return TileGrass, VarBoulders, true // tundra
		}
	}
	return 0, 0, false
}

// Oases of a desert island since StyleWilds.
const (
	oasisPond        = 3.2  // tiles of radius: the pond
	oasisGrass       = 8.5  // and the grass round it
	wildsJungleHeart = 0.34 // of a large jungle's radius: its heart, woods all over, where Titanoboa lies
)

// oasis is the ground at (x, y) if it lies in one of desert island s's
// oases: two on a large island, one on a medium one.
func (s *isle) oasis(x, y float64, n zoneNoise) (Tile, GroundVariant, bool) {
	count := 0
	switch {
	case s.role == roleLarge:
		count = 2
	case s.role == roleMedium && s.r >= 28:
		count = 1
	}
	for k := range count {
		a := unit(hash2(int32(k), 7, s.seed+0x0a5)) * 2 * math.Pi
		ox, oy := s.edge(a, 0.38+0.18*unit(hash2(int32(k), 9, s.seed+0x0a6)))
		if s.flat[2] > 0 && math.Hypot(ox-s.flat[0], oy-s.flat[1]) < s.flat[2]+oasisGrass+12 {
			ox, oy = s.edge(a+math.Pi, 0.45) // not on the pyramid's plain: across the island
		}
		d := math.Hypot(x-ox, y-oy)
		edge := oasisGrass + 2.5*(n.fine-0.5)*2
		switch {
		case d < oasisPond:
			return TileWater, VarLake, true
		case d < edge:
			return TileGrass, VarMeadow, true
		case d < edge+2 && n.woods > 0.5:
			return TileGrass, VarPlain, true
		}
	}
	return 0, 0, false
}
