package sim

import (
	"math"
	"math/rand/v2"
	"slices"
)

// The open world: a main continent and scattered islands with deserts,
// swamps, mountains with caves running through them, and one volcano. It is
// a different generator from the single island of the first worlds, picked
// by GenConfig.Style, so saves of island worlds still regenerate unchanged.

// WorldStyle picks the world generator.
type WorldStyle uint8

const (
	StyleIsland  WorldStyle = iota // one island; the first worlds
	StyleOpenV1                    // the first open worlds: caves open to the sky at any side
	StyleOpenV2                    // caves under the mountains, entered through mouths in the south cliffs
	StyleOpen                      // as V2, with larger deserts, swamps and beaches
	StyleIce                       // as StyleOpen, with a frozen north; see iceage.go
	StyleJungle                    // as StyleIce, with the woods spread over far more of the lowland
	StyleTribal                    // as StyleJungle, with thinner woods, trails, villages, deep caves; see tribe.go
	StyleSeas                      // as StyleTribal, but an archipelago: one land to an island; see archipelago.go
	StyleIsles                     // as StyleSeas, with designed islands, coasts, heights and rivers; see isles.go
	StyleWoods                     // as StyleIsles, with trees spaced by their crowns, trails and undergrowth; see woods.go
	StyleBeaches                   // as StyleWoods, with wide beaches, dunes and driftwood; see beaches.go
	StyleWilds                     // as StyleBeaches, with fewer, bigger islands, and animals by island and by distance from home; see wilds.go
	StyleGround                    // as StyleWilds, with the small plants in drifts, patches and glades, and worms on damp ground; see groundcover.go
	StyleCaves                     // as StyleGround, with a natural cave system under each island, on two levels; see cavesys.go
	StyleFlint                     // as StyleCaves, with about half the flint on the rocky hills, in small veins with gaps; see orevein.go

	// StyleLatest is what a new game is made with.
	StyleLatest = StyleFlint
)

// biomeShares is how much of the lowland each biome takes, and how wide the
// beaches are (as a share of the land's height above the sea).
type biomeShares struct{ beach, desert, marsh float64 }

func (w *World) biomeShares() biomeShares {
	if w.style >= StyleOpen {
		return biomeShares{beach: 0.075, desert: 0.3, marsh: 0.33}
	}
	return biomeShares{beach: openBeach, desert: openDesertShare, marsh: openMarshShare}
}

// World sizes offered for a new open world.
const (
	SizeMedium   = 256
	SizeLarge    = 512
	SizeHuge     = 1024
	SizeColossal = 2048 // "Khổng lồ": four times the land of SizeHuge
)

// Open-world tuning. Shares are of the land, so every biome shows up at every
// world size.
const (
	openSeaShare      = 0.46 // of all tiles
	openBeach         = 0.035
	openMountainShare = 0.07
	openRockyShare    = 0.06
	openDesertShare   = 0.17
	openMarshShare    = 0.13
	openCaveWidth     = 0.05 // half-width of the noise band carved into tunnels
	openPondLevel     = 0.63
	openLavaRiver     = 0.028
	openSpawnCalm     = 0.7 // share of grass and forest wanted around the spawn
	openMarshSpeed    = 0.65
	lavaWarmth        = 2.5 // tiles from lava that keep a player warm
)

// Resources of the new biomes.
const (
	CactusHits         = 2
	CactusYield        = 2
	CactusRegrowTicks  = 2 * DayTicks
	ReedYield          = 2 // dry grass
	ReedRegrowTicks    = DayTicks
	MushroomYield      = 1
	MushroomRegrowTick = 3 * DayTicks / 2
	CoalHits           = 3
	CoalYield          = 2
	CoalFuel           = 2 * FireFuelPerWood // one lump of coal burns like two logs
	ClamYield          = 2
	ClamRegrowTicks    = DayTicks
	clamShare          = 0.09 // of the sand along the sea
)

// openWorld fills w with the open-world terrain and returns the spawn tile.
func (w *World) openTerrain(seed uint32) (sx, sy int) {
	W, H := w.width, w.height
	n := float64(max(W, H))
	contScale := n / 3.2
	bScale := max(70, n/5)
	elev := make([]float64, W*H) // decides land and sea
	hill := make([]float64, W*H) // the same without the islands: decides the mountains

	// Elevation: continents, finer hills, a rise in the middle and the sea
	// all round the edges.
	for ty := 0; ty < H; ty++ {
		for tx := 0; tx < W; tx++ {
			x, y := float64(tx)+0.5, float64(ty)+0.5
			dx, dy := (x-float64(W)/2)/(float64(W)/2), (y-float64(H)/2)/(float64(H)/2)
			d := 0.6*math.Max(math.Abs(dx), math.Abs(dy)) + 0.4*math.Hypot(dx, dy)/math.Sqrt2
			e := 0.62*fbm(x/contScale, y/contScale, seed+11, 4) + 0.38*fbm(x/48, y/48, seed+12, 4)
			e += 0.16*(1-d)*(1-d) - 0.7*smoothstep(clamp((d-0.74)/0.26, 0, 1))
			hill[ty*W+tx] = e
			// An archipelago in the sea around the main land: noise peaks
			// raised into islands, fading out towards the map edge.
			if d > 0.45 {
				isl := smoothstep(clamp((fbm(x/26+91, y/26+37, seed+13, 3)-0.56)/0.12, 0, 1))
				e += 0.32 * isl * smoothstep(clamp((d-0.45)/0.2, 0, 1)) * (1 - smoothstep(clamp((d-0.86)/0.1, 0, 1)))
			}
			elev[ty*W+tx] = e
		}
	}
	sea := quantile(elev, openSeaShare)

	var land []int
	for i, e := range elev {
		if e < sea {
			w.tiles[i] = TileWater
		} else {
			land = append(land, i)
		}
	}
	landElev := make([]float64, len(land))
	for j, i := range land {
		landElev[j] = hill[i]
	}
	mShare := openMountainShare
	if w.style >= StyleTribal {
		mShare = tribalMountainShare // more mountain, for the deep caves
	}
	mountain := quantile(landElev, 1-mShare)
	rocky := quantile(landElev, 1-mShare-openRockyShare)
	top := slices.Max(landElev)

	// Temperature (warmer to the south) and moisture pick the lowland biomes.
	temp := make([]float64, W*H)
	wet := make([]float64, W*H)
	var lowDesert, lowMarsh, lowTemp []float64
	for _, i := range land {
		x, y := float64(i%W)+0.5, float64(i/W)+0.5
		temp[i] = fbm(x/bScale, y/bScale, seed+41, 3) + 0.3*(y/float64(H)-0.5)
		wet[i] = fbm(x/(bScale*0.8)+71, y/(bScale*0.8)+29, seed+42, 3)
		if hill[i] < rocky {
			e := clamp((elev[i]-sea)/(top-sea), 0, 1)
			lowDesert = append(lowDesert, temp[i]-wet[i])
			lowMarsh = append(lowMarsh, wet[i]-0.6*e-0.4*temp[i])
			lowTemp = append(lowTemp, temp[i]+0.25*wet[i])
		}
	}
	shares := w.biomeShares()
	desert := quantile(lowDesert, 1-shares.desert)
	marsh := quantile(lowMarsh, 1-shares.marsh)
	snow := math.Inf(-1) // the coldest lowland freezes over, in worlds made since StyleIce
	if w.style >= StyleIce {
		snow = quantile(lowTemp, snowShare)
	}

	for _, i := range land {
		x, y := float64(i%W)+0.5, float64(i/W)+0.5
		e := clamp((elev[i]-sea)/(top-sea), 0, 1)
		var t Tile
		switch {
		case hill[i] >= mountain:
			t = TileMountain
			// Tunnels: where a noise band crosses the mountain.
			if math.Abs(fbm(x/16, y/16, seed+31, 2)-0.5) < openCaveWidth {
				t = TileCave
			}
			if w.style >= StyleTribal && tribalCave(x, y, seed) {
				t = TileCave
			}
		case hill[i] >= rocky:
			t = TileRocky
		case temp[i]+0.25*wet[i] < snow:
			t = TileSnow
		case e < shares.beach:
			t = TileSand
		case temp[i]-wet[i] >= desert:
			t = TileDesert
		case wet[i]-0.6*e-0.4*temp[i] >= marsh:
			t = TileMarsh
			if fbm(x/7+13, y/7+57, seed+51, 2) > openPondLevel {
				t = TileWater // a pond
			}
		case fbm(x/28+57, y/28+213, seed+202, 4) > w.forestLine():
			t = TileForest
		default:
			t = TileGrass
		}
		w.tiles[i] = t
	}

	sx, sy = w.openSpawn()
	if w.style >= StyleTribal {
		w.thawNearSpawn(seed, sx, sy)
	}
	w.volcano(seed, sx, sy)
	if w.style >= StyleOpenV2 {
		w.cutCaveMouths()
	}
	return sx, sy
}

// forestLine is the noise level above which open lowland is forest.
func (w *World) forestLine() float64 {
	if w.style >= StyleJungle {
		return jungleForestLine
	}
	return forestLine
}

// minCave is the smallest cave system kept; smaller pockets are filled in.
const minCave = 6

// caveOutside reports whether (tx, ty) is open ground outside any cave.
func (w *World) caveOutside(tx, ty int) bool {
	t := w.TileAt(tx, ty)
	return t.Walkable() && t != TileCave
}

// cutCaveMouths reshapes the caves for the 3/4 view, where only a mountain's
// south face can be seen: tunnel ends leaving a mountain any other way are
// walled up, tiny pockets are filled in, and every cave system keeps, or
// gets dug, at least one mouth in a south face.
func (w *World) cutCaveMouths() {
	W := w.width
	// A mouth is a gap in a south face: open below, rock on both sides.
	// Every other way out of a tunnel is walled up.
	for i, t := range w.tiles {
		if t != TileCave {
			continue
		}
		x, y := i%W, i/W
		oW, oE, oN, oS := w.caveOutside(x-1, y), w.caveOutside(x+1, y), w.caveOutside(x, y-1), w.caveOutside(x, y+1)
		mouth := oS && !oW && !oE
		if (oW || oE || oN || oS) && !mouth {
			w.tiles[i] = TileMountain
		}
	}

	// Find every cave system first, then fill or dig: digging one system's
	// passage may touch another, which must not then be filled with it.
	seen := make([]bool, len(w.tiles))
	var systems [][]int
	for start, t := range w.tiles {
		if t != TileCave || seen[start] {
			continue
		}
		comp := []int{start}
		seen[start] = true
		for k := 0; k < len(comp); k++ {
			x, y := comp[k]%W, comp[k]/W
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if !w.inBounds(nx, ny) {
					continue
				}
				if j := ny*W + nx; !seen[j] && w.tiles[j] == TileCave {
					seen[j] = true
					comp = append(comp, j)
				}
			}
		}
		systems = append(systems, comp)
	}

	for _, comp := range systems {
		fill := func() {
			for _, j := range comp {
				w.tiles[j] = TileMountain
			}
		}
		if len(comp) < minCave {
			fill()
			continue
		}
		hasMouth, low := false, comp[0]
		for _, j := range comp {
			x, y := j%W, j/W
			if w.caveOutside(x, y+1) {
				hasMouth = true
				break
			}
			if y > low/W || (y == low/W && abs(x-W/2) < abs(low%W-W/2)) {
				low = j
			}
		}
		if hasMouth {
			continue
		}
		// Dig a passage straight down from the lowest point to open ground.
		x, y := low%W, low/W
		var dug []int
		for y++; w.inBounds(x, y) && w.TileAt(x, y) == TileMountain; y++ {
			dug = append(dug, y*W+x)
		}
		if !w.caveOutside(x, y) || w.caveOutside(x-1, y-1) || w.caveOutside(x+1, y-1) {
			fill() // it would open onto water, lava or the edge, or not in a face: no way in
			continue
		}
		for _, j := range dug {
			w.tiles[j] = TileCave
		}
	}
}

// openSpawn finds a grass or forest tile near the middle with mostly the same
// around it, so the first night is spent somewhere gentle.
func (w *World) openSpawn() (int, int) {
	cx, cy := w.width/2, w.height/2
	calm := func(tx, ty int) float64 {
		good, all := 0, 0
		for y := ty - 8; y <= ty+8; y += 2 {
			for x := tx - 8; x <= tx+8; x += 2 {
				all++
				if t := w.TileAt(x, y); t == TileGrass || t == TileForest {
					good++
				}
			}
		}
		return float64(good) / float64(all)
	}
	fallback := -1
	for r := 0; r < max(w.width, w.height); r += 2 {
		for ty := cy - r; ty <= cy+r; ty++ {
			for tx := cx - r; tx <= cx+r; tx++ {
				if max(abs(tx-cx), abs(ty-cy)) != r || !w.inBounds(tx, ty) {
					continue
				}
				if t := w.tiles[ty*w.width+tx]; t != TileGrass && t != TileForest {
					continue
				}
				if fallback < 0 {
					fallback = ty*w.width + tx
				}
				if calm(tx, ty) >= openSpawnCalm {
					return tx, ty
				}
			}
		}
	}
	if fallback >= 0 {
		return fallback % w.width, fallback / w.width
	}
	return cx, cy
}

// volcano raises one volcano on land far from the spawn: a lava lake in a
// ring of rock, ash all around, and lava running down through the ash.
func (w *World) volcano(seed uint32, sx, sy int) {
	n := max(w.width, w.height)
	R := min(max(float64(n)/14, 14), 40)
	minAway := math.Max(float64(n)/4, 3*R)
	if w.style >= StyleTribal {
		minAway = math.Max(float64(n)/2.6, 3.5*R) // well away: it is a long journey
	}
	if w.style >= StyleSeas {
		minAway = float64(n) / 5 // the ash island is far already
	}
	best, bestScore := -1, math.Inf(-1)
	for ty := int(R); ty < w.height-int(R); ty += 4 {
		for tx := int(R); tx < w.width-int(R); tx += 4 {
			d := math.Hypot(float64(tx-sx), float64(ty-sy))
			if d < minAway || w.tiles[ty*w.width+tx] == TileWater {
				continue
			}
			if w.style >= StyleSeas && w.tiles[ty*w.width+tx] != TileAsh {
				continue // in an archipelago the volcano stands on the ash island
			}
			// Prefer solid land under the whole cone, then distance.
			landN, all := 0, 0
			for a := 0.0; a < 2*math.Pi; a += math.Pi / 8 {
				for _, f := range []float64{0.4, 0.8} {
					all++
					if w.TileAt(tx+int(math.Cos(a)*R*f), ty+int(math.Sin(a)*R*f)) != TileWater {
						landN++
					}
				}
			}
			if s := float64(landN)/float64(all)*3 + d/float64(n); s > bestScore {
				best, bestScore = ty*w.width+tx, s
			}
		}
	}
	if best < 0 {
		return // a world too small or too wet for a volcano
	}
	w.cone(seed, best%w.width, best/w.width, R)
}

// cone raises a volcano of radius R round (vx, vy): its lava lake, the
// ring of rock round it, and the ash with lava running down it.
func (w *World) cone(seed uint32, vx, vy int, R float64) {
	r0 := int(math.Ceil(R * 1.15))
	for ty := vy - r0; ty <= vy+r0; ty++ {
		for tx := vx - r0; tx <= vx+r0; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			x, y := float64(tx)+0.5, float64(ty)+0.5
			r := math.Hypot(float64(tx-vx), float64(ty-vy))/R + (fbm(x/6, y/6, seed+61, 2)-0.5)*0.3
			i := ty*w.width + tx
			switch {
			case r < 0.16:
				w.tiles[i] = TileLava
			case r < 0.3:
				w.tiles[i] = TileMountain
			case r < 1 && !w.tiles[i].Sea():
				w.tiles[i] = TileAsh
				if r < 0.85 && math.Abs(fbm(x/10, y/10, seed+62, 2)-0.5) < openLavaRiver {
					w.tiles[i] = TileLava
				}
			}
		}
	}
}

// scatterOpen places the resources of every biome.
func (w *World) scatterOpen(rng, sizeRng *rand.Rand) {
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			r := rng.Float64()
			var kind EntityKind
			tile := w.tiles[ty*w.width+tx]
			switch tile {
			case TileForest:
				kind = pick(r, chance{EntityTree, w.forestTrees()}, chance{EntityBerryBush, 0.03}, chance{EntityGrassTuft, 0.03}, chance{EntityTwig, 0.035})
			case TileGrass:
				kind = pick(r, chance{EntityTree, grassTrees}, chance{EntityGrassTuft, 0.08}, chance{EntityBerryBush, 0.03}, chance{EntityTwig, 0.015})
			case TileRocky:
				kind = pick(r, chance{EntityBoulder, 0.12}, chance{EntityGrassTuft, 0.02})
			case TileSand:
				if w.seaside(tx, ty) {
					kind = pick(r, chance{EntityClam, clamShare}, chance{EntityGrassTuft, 0.02})
				} else {
					kind = pick(r, chance{EntityGrassTuft, 0.02})
				}
			case TileDesert:
				kind = pick(r, chance{EntityCactus, 0.045}, chance{EntityBoulder, 0.012}, chance{EntityTwig, 0.012})
			case TileMarsh:
				kind = pick(r, chance{EntityReed, 0.14}, chance{EntityMushroom, 0.03}, chance{EntityTree, marshTrees}, chance{EntityGrassTuft, 0.02})
			case TileCave:
				kind = pick(r, chance{EntityBoulder, 0.07}, chance{EntityNone, 0.05}, chance{EntityMushroom, 0.03})
			case TileAsh:
				kind = pick(r, chance{EntityBoulder, 0.05}, chance{EntityNone, 0.025}) // coal lies in seams: see orevein.go
			case TileSnow: // snowy pines, a few rocks and fallen branches, frozen tufts
				kind = pick(r, chance{EntityTree, 0.1}, chance{EntityBoulder, 0.02}, chance{EntityTwig, 0.02}, chance{EntityGrassTuft, 0.015}, chance{EntityPebble, 0.008})
			}
			if kind == EntityNone {
				continue
			}
			e := w.addEntity(kind, tx, ty)
			if kind.Sized() && sizeRng.Float64() < bigShare(kind, tile) {
				e.Big = true
				e.HP = maxHP(kind, true)
			}
		}
	}
}

// seaside reports whether tile (tx, ty) is beach sand right by the sea (or
// a lake): sand with water on one of its eight sides. Clams lie there.
func (w *World) seaside(tx, ty int) bool {
	if w.TileAt(tx, ty) != TileSand {
		return false
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && w.inBounds(tx+dx, ty+dy) && w.TileAt(tx+dx, ty+dy).Sea() {
				return true
			}
		}
	}
	return false
}

// speedAt is how fast a player walks on the ground at (x, y): mud is slow.
func (w *World) speedAt(x, y float64) float64 {
	if t := w.TileAt(int(math.Floor(x)), int(math.Floor(y))); t == TileSnow {
		return snowSpeed // deep snow
	}
	switch w.TileAt(int(math.Floor(x)), int(math.Floor(y))) {
	case TileMarsh:
		return openMarshSpeed
	case TilePath:
		return PathSpeed
	case TileFord:
		return FordSpeed // wading: see rivers.go
	}
	return 1
}

// nearLava reports whether (x, y) is within lavaWarmth of lava.
func (w *World) nearLava(x, y float64) bool {
	r := int(math.Ceil(lavaWarmth))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - r; ty <= cy+r; ty++ {
		for tx := cx - r; tx <= cx+r; tx++ {
			if w.TileAt(tx, ty) != TileLava {
				continue
			}
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			if dx*dx+dy*dy <= lavaWarmth*lavaWarmth {
				return true
			}
		}
	}
	return false
}

// quantile returns the value below which share of vs lie (vs is not changed).
func quantile(vs []float64, share float64) float64 {
	if len(vs) == 0 {
		return math.Inf(1)
	}
	s := slices.Clone(vs)
	slices.Sort(s)
	i := int(share * float64(len(s)))
	return s[min(max(i, 0), len(s)-1)]
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// Tribal-world tuning.
const (
	tribalMountainShare = 0.1
	tribalForestTrees   = 0.36 // the woods are thinner, with room to walk
	tribalSnowAway      = 0.22 // of the world's size: no snow this near the spawn
)

// forestTrees is the share of forest tiles with a tree.
func (w *World) forestTrees() float64 {
	if w.style >= StyleTribal {
		return tribalForestTrees
	}
	return forestTrees
}

// tribalCave reports whether the deep caves of a tribal world run through
// (x, y), inside a mountain: two crossing networks of tunnels and rounder
// chambers where they meet.
func tribalCave(x, y float64, seed uint32) bool {
	a := math.Abs(fbm(x/19+11, y/19+5, seed+33, 2)-0.5) < 0.05
	b := math.Abs(fbm(x/11+40, y/11+17, seed+34, 2)-0.5) < 0.032
	chamber := fbm(x/8+9, y/8+3, seed+35, 2) > 0.66
	return a || b || chamber
}

// thawNearSpawn turns the snow near the spawn to wood and meadow: the frozen
// north is a long way off.
func (w *World) thawNearSpawn(seed uint32, sx, sy int) {
	r := tribalSnowAway * float64(max(w.width, w.height))
	r = max(r, 50)
	for ty := sy - int(r); ty <= sy+int(r); ty++ {
		for tx := sx - int(r); tx <= sx+int(r); tx++ {
			if !w.inBounds(tx, ty) || w.tiles[ty*w.width+tx] != TileSnow {
				continue
			}
			d := math.Hypot(float64(tx-sx), float64(ty-sy))
			// The edge wavers a little so it does not show as a circle.
			x, y := float64(tx)+0.5, float64(ty)+0.5
			if d > r*(0.85+0.3*fbm(x/12, y/12, seed+91, 2)) {
				continue
			}
			t := TileGrass
			if fbm(x/28+57, y/28+213, seed+202, 4) > w.forestLine() {
				t = TileForest
			}
			w.tiles[ty*w.width+tx] = t
		}
	}
}
