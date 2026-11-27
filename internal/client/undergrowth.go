package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The undergrowth is ferns, broad leaves, leafy clumps, lush grass and wild
// flowers covering the ground of the woods and meadows, and the shade the
// forest canopy casts. It is only for looks: there is nothing to pick or
// bump into, and each tile's plants come from a hash of its position, so
// they stay put without being stored anywhere.

type plantKind uint8

const (
	plantFern plantKind = iota
	plantBroadleaf
	plantClump
	plantBlades
	plantFlowers
	numPlantKinds
)

// plantVariants is how many drawings there are of each kind of plant at
// density 1 (each also mirrored); at HD there are coverVariants of them, in
// every season (see coverart.go).
const plantVariants = 3

// plantCover is the HD drawing of each kind of plant.
var plantCover = [numPlantKinds]coverKind{cvFern, cvBroad, cvClump, cvGrass, cvFlowers}

// plant is one plant of the undergrowth.
type plant struct {
	kind    plantKind
	variant int     // 0 to 2*plantVariants-1 at density 1, the second half mirrored; any at HD
	x, y    float64 // where it grows, in tiles
	spot    int     // which of the spots of its tile it grows on
	hash    uint32  // for its look: see coverLookAt
}

// The undergrowth does not lie evenly. Each kind of plant follows fields
// of noise as plants follow light, water and each other: green grass in
// drifts, flowers in patches of one kind, ferns and broad leaves in the
// shade of the trees, open glades and bare spots where little grows; and
// all of it thinner in winter, the flowers thickest in spring.
const (
	ugSpots     = 3    // spots tried on a tile
	ugBareFrom  = 0.64 // of the glade noise: a glade begins...
	ugBareWidth = 0.08 // ...and is bare this much further on
)

// plantChances is the chance of each kind of plant at a spot of tile
// (tx, ty) of kind t, shade (0 to 1.5) under the trees, in season s.
func plantChances(tx, ty int, t sim.Tile, shade float64, s sim.Season) (ch [numPlantKinds]float64) {
	x, y := float64(tx)+0.5, float64(ty)+0.5
	bare := smooth(clamp01((vnoise(x/11+3, y/11+7, 840) - ugBareFrom) / ugBareWidth))
	grow := 1 - bare
	drift := smooth(clamp01((vnoise(x/5.5+11, y/4.5+5, 841) - 0.36) / 0.32))
	patch := smooth(clamp01((vnoise(x/6+21, y/6+2, 842) - 0.68) / 0.1))
	lush := vnoise(x/14+1, y/14+9, 843)
	clumps := smooth(clamp01((vnoise(x/4+41, y/4+13, 845) - 0.5) / 0.25))
	shade = min(1, shade)
	bloom := [sim.NumSeasons]float64{1.8, 0.8, 0.45, 0}[s%sim.NumSeasons]
	switch t {
	case sim.TileGrass:
		ch[plantBlades] = (0.02 + 0.5*drift*drift*(0.5+lush)) * grow * (1 - 0.5*shade)
		ch[plantFlowers] = (0.002 + 0.3*patch) * grow * bloom * (1 - 0.7*shade)
		ch[plantClump] = (0.008+0.09*clumps)*grow + 0.06*shade
		ch[plantFern] = 0.3 * shade * shade
		ch[plantBroadleaf] = 0.004 + 0.1*shade
	case sim.TileForest:
		ferny := smooth(clamp01((vnoise(x/4+31, y/4+17, 844) - 0.28) / 0.4))
		sun := 1 - shade
		ch[plantFern] = (0.1 + 0.36*ferny) * (0.55 + 0.45*shade) * (0.4 + 0.6*grow)
		ch[plantBroadleaf] = (0.05 + 0.12*ferny*shade) * (0.5 + 0.5*grow)
		ch[plantClump] = 0.05 + 0.1*clumps*(1-ferny)
		ch[plantBlades] = 0.02 + 0.3*sun*drift + 0.25*bare
		ch[plantFlowers] = (0.002 + 0.2*patch*(sun+bare)) * bloom
	case sim.TileMarsh: // open mud between the cypresses: a few clumps and tufts, in patches
		ch[plantClump] = 0.015 + 0.05*clumps
		ch[plantBlades] = 0.02 + 0.08*drift
	case sim.TileRocky: // in the cracks
		ch[plantBlades] = 0.008 + 0.04*drift
		ch[plantClump] = 0.004 + 0.03*clumps
	}
	if s == sim.Winter { // most of it under the snow
		for k := range ch {
			ch[k] *= 0.35
		}
	}
	return ch
}

// plantsOn appends to out the plants growing on tile (tx, ty) of kind t,
// out of the shade of the trees, in summer.
func plantsOn(tx, ty int, t sim.Tile, out []plant) []plant {
	return plantsAt(tx, ty, t, 0, sim.Summer, out)
}

// plantsAt appends to out the plants growing on tile (tx, ty) of kind t,
// shade under the trees, in season s.
func plantsAt(tx, ty int, t sim.Tile, shade float64, s sim.Season, out []plant) []plant {
	ch := plantChances(tx, ty, t, shade, s)
	sum := 0.0
	for _, c := range ch {
		sum += c
	}
	if sum == 0 {
		return out
	}
	norm := 1.0
	if sum > 0.95 {
		norm = 0.95 / sum
	}
	for i := range ugSpots {
		h := func(s uint32) float64 { return hashf(tx*4+i, ty, 810+s) }
		r, acc := h(0), 0.0
		for k, c := range ch {
			acc += c * norm
			if r >= acc {
				continue
			}
			out = append(out, plant{
				kind:    plantKind(k),
				variant: plantVariant(plantKind(k), tx, ty, h(1)),
				x:       float64(tx) + (float64(i)+0.15+0.7*h(2))/ugSpots,
				y:       float64(ty) + 0.2 + 0.75*h(3),
				spot:    i,
				hash:    uint32(h(4) * (1 << 32)),
			})
			break
		}
	}
	return out
}

// plantSway is how far each kind of plant leans in the wind.
var plantSway = [numPlantKinds]float64{0.05, 0.025, 0.012, 0.09, 0.07}

// plantBend is how far each kind of plant bends under feet.
var plantBend = [numPlantKinds]float64{0.35, 0.25, 0.12, 0.5, 0.45}

// plantArt holds every density-1 drawing of every plant, made the first
// time it is needed.
var plantArt [numPlantKinds][2 * plantVariants]sprite

// plantSprite is the drawing of a plant of kind k: at HD drawing v of the
// summer set, at density 1 the old drawing v.
func plantSprite(k plantKind, v int) sprite {
	if envHD {
		return coverSprite(plantCover[k], v, sim.Summer)
	}
	if plantArt[0][0].img == nil {
		for kind := range numPlantKinds {
			for i := 0; i < plantVariants; i++ {
				c, ax, ay := genPlant(kind, i)
				plantArt[kind][i] = mkUp(c, ax, ay)
				plantArt[kind][i+plantVariants] = mkUp(c.mirrored(), float64(c.w)-ax, ay)
			}
		}
	}
	return plantArt[k][v%(2*plantVariants)]
}

var (
	fernRamp   = mkRamp(0x123024, 0x1a432b, 0x255c34, 0x34763c, 0x4b9046, 0x6aa852)
	broadRamp  = mkRamp(0x0f3326, 0x174a31, 0x22633a, 0x337e43, 0x4d984c, 0x70b05a)
	bladeRamp  = mkRamp(0x24502c, 0x316a34, 0x42843c, 0x5a9c45, 0x78b452)
	petalRamps = []ramp{
		mkRamp(0xc9c3b2, 0xece8da, 0xffffff), // white
		mkRamp(0xc99a1c, 0xeec83c, 0xfbe77a), // yellow
		mkRamp(0x7a4aa6, 0xa673d0, 0xcfa6ee), // violet
	}
)

// genPlant draws variant v of plant kind k and returns it with its anchor,
// the point where it meets the ground.
func genPlant(k plantKind, v int) (*canvas, float64, float64) {
	rng := rand.New(rand.NewPCG(uint64(k)*31+7, uint64(v)*17+3))
	if envHD { // see hdplants.go
		switch k {
		case plantFern:
			return genFernHD(rng)
		case plantBroadleaf:
			return genBigLeavesHD(rng)
		case plantClump:
			return genClumpHD(rng, uint32(v))
		case plantBlades:
			return genBladesHD(rng)
		default:
			return genFlowersHD(rng, petalRamps[v%len(petalRamps)])
		}
	}
	switch k {
	case plantFern:
		return genFern(rng)
	case plantBroadleaf:
		return genBigLeaves(rng)
	case plantClump:
		return genClump(rng, uint32(v))
	case plantBlades:
		return genBlades(rng)
	default:
		return genFlowers(rng, petalRamps[v%len(petalRamps)])
	}
}

// genFern: fronds fanning out and drooping from the root, each lined with
// little leaflets. The upright ones are behind and darker.
func genFern(rng *rand.Rand) (*canvas, float64, float64) {
	c := newCanvas(30, 17)
	const bx, by = 15.0, 16.0
	n := 6 + rng.IntN(2)
	type frond struct{ a, l float64 }
	fronds := make([]frond, n)
	for i := range fronds {
		f := (float64(i) + 0.5 + (rng.Float64()-0.5)*0.5) / float64(n)
		fronds[i] = frond{math.Pi * (1.08 - 1.16*f), 9 + rng.Float64()*4}
	}
	// Back to front: the most upright first.
	for pass := 0; pass < 2; pass++ {
		for _, fr := range fronds {
			upright := math.Abs(math.Cos(fr.a)) < 0.45
			if upright != (pass == 0) {
				continue
			}
			dark := 0.0
			if upright {
				dark = 0.18
			}
			at := func(t float64) (float64, float64) {
				return bx + math.Cos(fr.a)*fr.l*t, by - math.Sin(fr.a)*fr.l*t + fr.l*0.55*t*t
			}
			for s := 0; s <= 24; s++ {
				t := float64(s) / 24
				x, y := at(t)
				col := fernRamp.pick(0.25+0.45*t-dark, int(x), int(y))
				c.set(int(math.Floor(x)), int(math.Floor(y)), col)
				if s%3 != 1 || t < 0.12 {
					continue
				}
				// Leaflets on both sides, shorter towards the tip.
				x2, y2 := at(t + 0.02)
				dx, dy := x2-x, y2-y
				dl := math.Hypot(dx, dy)
				nx, ny := -dy/dl, dx/dl
				ln := 0.8 + 2.2*(1-t)
				for _, side := range []float64{-1, 1} {
					for q := 1.0; q <= ln; q++ {
						px, py := x+side*nx*q+dx/dl*q*0.5, y+side*ny*q+dy/dl*q*0.5
						l := 0.35 + 0.4*t - dark
						if side*ny < 0 { // the leaflets facing up catch the light
							l += 0.15
						}
						c.set(int(math.Floor(px)), int(math.Floor(py)), fernRamp.pick(l, int(px), int(py)))
					}
				}
			}
		}
	}
	c.outline(0.5)
	return c, bx, by
}

// genBigLeaves: a few big pointed leaves on short stalks, each with a vein.
func genBigLeaves(rng *rand.Rand) (*canvas, float64, float64) {
	c := newCanvas(26, 20)
	const bx, by = 13.0, 19.0
	n := 3 + rng.IntN(2)
	for i := 0; i < n; i++ {
		f := (float64(i) + 0.5) / float64(n)
		a := math.Pi * (0.85 - 0.7*f + (rng.Float64()-0.5)*0.15)
		stalk := 2 + rng.Float64()*3
		L, W := 8+rng.Float64()*3, 3.2+rng.Float64()*1.2
		ux, uy := math.Cos(a), -math.Sin(a)*0.8 // foreshortened: leaves lie back a little
		vx, vy := -uy, ux
		x0, y0 := bx+ux*stalk, by+uy*stalk
		c.line(bx, by, x0, y0, func(x, y int, _ float64) color.RGBA { return broadRamp[1] })
		var pts []float64
		const steps = 10
		for s := 0; s <= steps; s++ { // one edge out to the tip...
			t := float64(s) / steps
			w := W * math.Pow(math.Sin(math.Pi*math.Min(1, t*1.15)), 0.7) * (1 - 0.15*t)
			pts = append(pts, x0+ux*L*t+vx*w, y0+uy*L*t+vy*w)
		}
		for s := steps; s >= 0; s-- { // ...and the other back
			t := float64(s) / steps
			w := W * math.Pow(math.Sin(math.Pi*math.Min(1, t*1.15)), 0.7) * (1 - 0.15*t)
			pts = append(pts, x0+ux*L*t-vx*w, y0+uy*L*t-vy*w)
		}
		c.polygon(pts, func(x, y int) color.RGBA {
			px, py := float64(x)+0.5-x0, float64(y)+0.5-y0
			along, across := (px*ux+py*uy)/L, (px*vx+py*vy)/W
			l := 0.55 + 0.25*across*vy*-1 + 0.2*along - 0.25*math.Abs(across)
			return broadRamp.pick(l, x, y)
		})
		c.line(x0, y0, x0+ux*L*0.85, y0+uy*L*0.85, func(x, y int, t float64) color.RGBA {
			return broadRamp.pick(0.2+0.2*t, x, y)
		})
	}
	c.outline(0.6)
	return c, bx, by
}

// genClump: a low, round mound of small leaves.
func genClump(rng *rand.Rand, seed uint32) (*canvas, float64, float64) {
	c := newCanvas(18, 11)
	var blobs []blob
	for i := 0; i < 4; i++ {
		blobs = append(blobs, blob{4 + rng.Float64()*10, 5.5 + rng.Float64()*2.5, 2.8 + rng.Float64()*1.4})
	}
	leafCluster(c, blobs, 9, 6, 7, 4.5, fernRamp, 900+seed)
	c.outline(0.6)
	return c, 9, 9.5
}

// genBlades: a tuft of lush green grass.
func genBlades(rng *rand.Rand) (*canvas, float64, float64) {
	c := newCanvas(18, 14)
	const bx, by = 9.0, 13.0
	for row := 0; row < 2; row++ {
		n := 5 + row
		for i := 0; i < n; i++ {
			x0 := bx - 5 + (float64(i)+rng.Float64()*0.7)*10/float64(n)
			h := 9 - float64(row)*2 + rng.Float64()*3
			lean := (x0-bx)*0.5 + (rng.Float64()-0.5)*3
			for s := 0; s <= 20; s++ {
				t := float64(s) / 20
				x, y := int(math.Floor(x0+lean*t*t)), int(math.Floor(by-h*t))
				c.set(x, y, bladeRamp.pick(0.1+float64(row)*0.15+t*0.6, x, y))
			}
		}
	}
	c.outline(0.35)
	return c, bx, by
}

// genFlowers: a few thin stems with small blooms of one colour.
func genFlowers(rng *rand.Rand, petals ramp) (*canvas, float64, float64) {
	c := newCanvas(14, 11)
	const bx, by = 7.0, 10.0
	n := 3 + rng.IntN(3)
	for i := 0; i < n; i++ {
		x := bx - 4 + rng.Float64()*8
		h := 3 + rng.Float64()*4
		lean := (rng.Float64() - 0.5) * 2
		c.line(x, by, x+lean, by-h, func(px, py int, t float64) color.RGBA { return bladeRamp.pick(0.2+0.4*t, px, py) })
		fx, fy := int(math.Floor(x+lean)), int(math.Floor(by-h))
		c.set(fx, fy-1, petals[2])
		c.set(fx-1, fy, petals[1])
		c.set(fx+1, fy, petals[1])
		c.set(fx, fy+1, petals[0])
		c.set(fx, fy, hex(0xf6d860))
	}
	c.outline(0.3)
	return c, bx, by
}

// drawUndergrowth draws the shade of the canopy, the litter under the trees
// and the plants on the ground in view, before anything that stands on it.
func (g *Game) drawUndergrowth(dst drawTarget, cam camera) {
	if g.tiles == nil {
		return
	}
	g.drawCanopy(dst, cam)

	w, h := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	tx0, ty0 := int(math.Floor(cam.ox/TileW))-1, int(math.Floor(cam.oy/TileH))-1
	tx1, ty1 := int(math.Ceil((cam.ox+w)/TileW))+1, int(math.Ceil((cam.oy+h)/TileH))+2
	g.noteShade(tx0, ty0, tx1, ty1)
	g.noteMovers(g.alpha())
	g.drawLitter(dst, cam)
	season := g.curr.Season % sim.NumSeasons
	var set coverSet
	if envHD {
		set = coverSprites(season)
	}
	secs := float64(g.frame) / 60
	var plants []plant
	for ty := ty0; ty <= ty1; ty++ {
		for tx := tx0; tx <= tx1; tx++ {
			tile := g.tileAt(tx, ty)
			if tile == sim.TileWater {
				g.drawFloaters(dst, cam, tx, ty)
				continue
			}
			plants = plantsAt(tx, ty, tile, g.shade.at(tx, ty), season, plants[:0])
			if len(plants) == 0 {
				continue
			}
			tree := false
			if i, ok := g.entAt[[2]int{tx, ty}]; ok {
				if g.curr.Entities[i].Kind != sim.EntityTree {
					continue // keep what can be picked or used in plain sight
				}
				tree = true
			}
			for _, p := range plants {
				if tree && math.Abs(p.x-float64(tx)-0.5) < 0.3 {
					continue // not out of the middle of a trunk
				}
				sx, sy := cam.toScreen(p.x, p.y)
				if !envHD {
					skew := plantSway[p.kind] * math.Sin(secs*1.3+p.x*0.9+p.y*0.5)
					drawSwaying(dst, plantSprite(p.kind, p.variant), sx, sy, skew)
					continue
				}
				ss := set[plantCover[p.kind]]
				s := ss[p.variant%len(ss)]
				bend := g.bend.at(coverKey(tx, ty, p.spot), p.x, p.y)
				skew := plantSway[p.kind]*g.windSkew(p.x, p.y, float64(p.hash%628)/100) - plantBend[p.kind]*bend
				drawCover(dst, s, sx, sy, skew, math.Abs(bend)*plantBend[p.kind]*1.6, coverLookAt(p.x, p.y, p.hash))
			}
		}
	}
}

const (
	canopyShade = 0.26  // how much the forest canopy darkens the ground under it
	sunFlecks   = 0.075 // how bright the spots of sun through the leaves are at noon
)

// drawCanopy darkens the forest floor, fading at the edges of the woods, and
// by day lets a few spots of sunlight through, drifting as the leaves move.
func (g *Game) drawCanopy(dst drawTarget, cam camera) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	tx0, ty0, ok := g.tileMask(&g.canopy, cam, w, h, func(t sim.Tile) bool { return t == sim.TileForest })
	if !ok {
		return
	}
	op := maskOp(cam, tx0, ty0)
	const a = canopyShade
	op.ColorScale.Scale(0.02*a, 0.07*a, 0.04*a, a)
	dst.DrawImage(g.canopy, op)

	amb := sim.Ambient(g.curr.Tick)
	if amb < 0.3 {
		return
	}
	secs := float64(g.frame) / 60
	for ty := ty0; ty < ty0+h/TileH+5; ty++ {
		for tx := tx0; tx < tx0+w/TileW+5; tx++ {
			if hashf(tx, ty, 830) > 0.18 || g.tileAt(tx, ty) != sim.TileForest {
				continue
			}
			ph := 2 * math.Pi * hashf(tx, ty, 831)
			x := float64(tx) + 0.5 + 0.3*math.Sin(secs*0.21+ph)
			y := float64(ty) + 0.5 + 0.25*math.Cos(secs*0.17+ph)
			sx, sy := cam.toScreen(x, y)
			k := float32(sunFlecks * amb * (0.6 + 0.4*math.Sin(secs*0.8+ph*3)))
			op := g.lightOpAt(sx, sy, 0.45+0.25*hashf(tx, ty, 832))
			op.ColorScale.Scale(k, k*0.95, k*0.7, k)
			op.Blend = ebiten.BlendLighter
			dst.DrawImage(g.art.light, op)
		}
	}
}

// plantVariant is which drawing of kind k grows on tile (tx, ty), r a hash
// of the spot: any for most kinds, but one kind of flower to a patch.
func plantVariant(k plantKind, tx, ty int, r float64) int {
	if k == plantFlowers && envHD {
		n := vnoise(float64(tx)/9+5, float64(ty)/9+3, 846)
		return int(n*float64(coverVariants[cvFlowers])*1.8) % coverVariants[cvFlowers]
	}
	return int(r * 2 * plantVariants)
}
