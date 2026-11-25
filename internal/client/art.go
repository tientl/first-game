package client

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// The ground is drawn foreshortened (tiles shorter than they are wide) while
// sprites stand upright on it: the classic 3/4 view. Sizes are in art pixels;
// the world is scaled up to the screen by an integer factor.
const (
	TileW = 24
	TileH = 18

	shadowSize  = 64
	lightSize   = 256
	flameFrames = 8
	walkFrames  = 8
)

// sprite is an image plus its anchor: the point that sits on the ground
// point, in art pixels. dens is the image's pixel density, image pixels per
// art pixel: 0 or 1 for the original art, HDDensity for redrawn HD art (see
// hd.go and HD_ART.md). Draw sprites with drawAt, drawSwaying or
// drawSprite, and size them with s.w() and s.h(), so either density works.
type sprite struct {
	img    *ebiten.Image
	ax, ay float64
	dens   int
}

// mk makes a sprite of c anchored at design point (ax, ay).
func mk(c *canvas, ax, ay float64) sprite {
	x, y := c.toPx(ax, ay)
	return sprite{img: c.image(), ax: x, ay: y}
}

// art holds every placeholder image, generated in code at start-up. Swapping
// in real art later only touches the art files.
type art struct {
	// Trees and boulders come in two sizes, indexed by sizeIdx. Each tree
	// in vineTrees is the one at the same index in trees, hung with vines.
	trees, vineTrees, pines [2][]sprite
	ancient                 [2]ancientTrees // prehistoric trees; see treeart.go
	snowPines               [2][]sprite     // conifers on the snow, frosted; see iceart.go
	heldIvorySpear          sprite
	stumps                  [2]sprite
	boulders                [2][]sprite
	bushFull                sprite
	bushPicked              sprite
	tuftFull                sprite
	tuftPicked              sprite
	twig                    sprite
	wall, bed               sprite
	blanket                 sprite

	heldSpear, heldTorch, heldRod sprite
	farm                          farmArt  // pebbles, the hoe, soil and sweet potatoes
	ore                           oreArt   // ores, the furnace and the tools of every tier
	gear                          gearArt  // stone walls and the swords of every tier
	fruit                         fruitArt // fruit trees, ripe and picked; see fruitart.go
	open                          openArt  // cacti, reeds, mushrooms, coal
	campBack                      sprite
	campFront                     sprite
	flames                        [flameFrames]sprite
	tombstone                     sprite
	chest, chestOpen              sprite

	// Player sprites, one set per look (body, hair), drawn on first use; see look.go.
	bodies map[sim.Look]*bodyArt
	armors map[armorKey]*armorArt // drawn on first use; see armorart.go

	beasts creatureArt                 // animals; see creatures.go
	bosses bossSprites                 // the six bosses; see bosses.go
	tribal tribalArt                   // the tribal age; see tribalart.go
	heads  map[sim.CreatureKind]sprite // trophy heads cut from the beasts; see tribalrender.go
	sea    seaArt                      // boats and sea monsters; see seaart.go
	desert desertArt                   // the pyramid and the giant scorpion; see desertart.go
	cave   caveArt                     // cave mouths and the way up; see caveart.go

	shadow   *ebiten.Image // soft black disc, drawn squashed
	light    *ebiten.Image // radial white gradient, alpha-premultiplied
	cloud    *ebiten.Image
	vignette *ebiten.Image
	marker   *ebiten.Image
	icons    [sim.NumItems]*ebiten.Image
	heart    *ebiten.Image
	food     *ebiten.Image
}

// Palette.
var (
	leafRamps = []ramp{
		mkRamp(0x173a2c, 0x225234, 0x2f6b3b, 0x478a43, 0x6aa84e, 0x96c65c),
		mkRamp(0x1d3d27, 0x2b5a2e, 0x3f7632, 0x5e943a, 0x86b048, 0xb5cb5e),
		mkRamp(0x15322e, 0x1d4a39, 0x276141, 0x3a7c4a, 0x56985a, 0x7cb56a),
	}
	pineRamp    = mkRamp(0x10282a, 0x173a34, 0x1f4d3c, 0x2b6245, 0x3c7850, 0x55905c)
	bushRamp    = mkRamp(0x173b2a, 0x22533a, 0x2f6c42, 0x46884c, 0x67a55a)
	bushDryRamp = mkRamp(0x30402a, 0x435633, 0x596e3d, 0x718548, 0x8b9b55)
	trunkRamp   = mkRamp(0x2f1f19, 0x46301f, 0x5e4128, 0x7a5533, 0x946a40)
	woodRamp    = mkRamp(0x8a6038, 0xb08452, 0xcfa56c, 0xe6c48b)
	stoneRamp   = mkRamp(0x2e2d38, 0x45444f, 0x5f5e68, 0x7b7a83, 0x9b9aa1, 0xc2c1c4)
	mossRamp    = mkRamp(0x2c4a2c, 0x3d6334, 0x557f3d, 0x6f9947)
	berryRamp   = mkRamp(0x5c0f26, 0xa31d3b, 0xdc3b52, 0xff8a92)
	strawRamp   = mkRamp(0x4f5a22, 0x73802f, 0x9aa23f, 0xc4c257, 0xe8e08a)
	ashRamp     = mkRamp(0x1e1a1c, 0x2f292a, 0x433a37)
	fireRamp    = mkRamp(0x8f1d12, 0xd8401a, 0xff8a1e, 0xffc63c, 0xfff1a8)
	skinRamp    = mkRamp(0x9b5a43, 0xc9845f, 0xeab08a, 0xf8d0aa)
	hairRamp    = mkRamp(0x24150f, 0x3b2418, 0x573620, 0x74492b)
	pantsRamp   = mkRamp(0x1e1c2b, 0x2c2a40, 0x3d3b58, 0x504e70)
	shirtRamp   = mkRamp(0x6a6a6a, 0x9a9a9a, 0xc8c8c8, 0xf0f0f0)
	shoeRamp    = mkRamp(0x231710, 0x3a271a, 0x553a26)
	beltRamp    = mkRamp(0x3a2616, 0x5a3c22)

	colWhite = color.RGBA{255, 255, 255, 255}
	colBlack = color.RGBA{0, 0, 0, 255}
)

func newArt() *art {
	a := &art{}
	for size := range 2 {
		a.trees[size], a.vineTrees[size], a.pines[size] = genTrees(treeScale[size])
		a.ancient[size] = genAncientTrees(treeScale[size])
		a.stumps[size] = genStump(stumpScale[size])
		for v := 0; v < 3; v++ {
			a.boulders[size] = append(a.boulders[size], genBoulder(v, boulderScale[size]))
		}
	}
	a.bushFull, a.bushPicked = genBush(true), genBush(false)
	a.tuftFull, a.tuftPicked = genTuft(true), genTuft(false)
	a.twig = genTwig()
	a.wall, a.bed, a.blanket = genWall(), genBed(), genBlanket()
	a.heldSpear, a.heldTorch, a.heldRod = genHeldSpear(), genHeldTorch(), genHeldRod()
	a.heldIvorySpear = genHeldIvorySpear()
	for size := range 2 {
		for v := range pineVariants {
			a.snowPines[size] = append(a.snowPines[size], genPineOf(v, treeScale[size], uint64(v)*13+1, true))
		}
	}
	a.ore = genOreArt()
	a.gear = genGearArt()
	a.fruit = genFruitArt()
	a.farm = genFarmArt()
	a.open = genOpenArt()
	a.campBack, a.campFront = genCampfire()
	for f := range a.flames {
		a.flames[f] = genFlame(f)
	}
	a.tombstone = genTombstone()
	a.chest, a.chestOpen = genChest(false), genChest(true)

	a.beasts = genCreatureArt()
	a.bosses = genBossSprites() // before addTribalArt cuts their heads

	a.shadow = genSoftDisc(shadowSize, 1.0)
	a.light = genLight()
	a.cloud = genCloud()
	a.vignette = genVignette()
	a.marker = genMarker()
	a.icons = genIcons()
	genCreatureIcons(&a.icons)
	genCraftIcons(&a.icons)
	genOpenIcons(&a.icons)
	genFarmIcons(&a.icons)
	genOreIcons(&a.icons)
	genGearIcons(&a.icons)
	genFruitIcons(&a.icons)
	a.heart, a.food = genHeart(), genFood()
	a.sea = genSeaArt()
	a.desert = genDesertArt()
	a.cave = genCaveArt()
	genSeaIcons(&a.icons)
	a.addTribalArt()
	redrawItemIcons(&a.icons) // see itemicons.go
	bossItemIcons(&a.icons)   // the legendary items; see bosses.go
	return a
}

// --- plants -----------------------------------------------------------------

type blob struct{ x, y, r float64 }

// leafCluster shades a union of round blobs as one volume lit from the
// top-left, with a clumpy leaf texture.
func leafCluster(c *canvas, blobs []blob, cx, cy, rx, ry float64, rp ramp, seed uint32) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, b := range blobs {
		minX, maxX = math.Min(minX, b.x-b.r), math.Max(maxX, b.x+b.r)
		minY, maxY = math.Min(minY, b.y-b.r), math.Max(maxY, b.y+b.r)
	}
	for y := int(minY); y <= int(maxY); y++ {
		for x := int(minX); x <= int(maxX); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			best, bi := -1.0, -1
			for i, b := range blobs {
				dx, dy := (px-b.x)/b.r, (py-b.y)/b.r
				if h := 1 - (dx*dx + dy*dy); h >= 0 && h+float64(i)*0.02 > best {
					best, bi = h+float64(i)*0.02, i
				}
			}
			if bi < 0 {
				continue
			}
			b := blobs[bi]
			gx, gy := (px-cx)/rx, (py-cy)/ry
			nx, ny := 0.45*(px-b.x)/b.r+0.65*gx, 0.45*(py-b.y)/b.r+0.65*gy
			l := lum(nx, ny) + (vnoise(px*0.45, py*0.45, seed)-0.5)*0.3
			if hashf(x, y, seed+1) < 0.1 {
				l += 0.12
			}
			if gy > 0.5 {
				l -= (gy - 0.5) * 0.4
			}
			c.set(x, y, rp.pick(l, x, y))
		}
	}
}

// Trees, stumps and boulders are generated at two scales, small then big
// (indexed by sizeIdx); at scale 1 each gen function draws its original shape.
var (
	treeScale    = [2]float64{0.78, 1.12}
	stumpScale   = [2]float64{0.8, 1.25}
	boulderScale = [2]float64{0.75, 1.2}
)

func sizeIdx(big bool) int {
	if big {
		return 1
	}
	return 0
}

func isTrunk(col color.RGBA) bool {
	for _, t := range trunkRamp {
		if t == col {
			return true
		}
	}
	return false
}

func genStump(s float64) sprite {
	c := newCanvas(int(math.Ceil(16*s)), int(math.Ceil(12*s)))
	cx, rx := math.Round(float64(c.w)/2), 6*s
	base := math.Round(9 * s) // ground point
	top := base - 4.5*s
	x0 := int(math.Round(cx - rx))
	c.rect(x0, int(math.Round(top-0.5)), int(math.Round(2*rx)), int(math.Round(base-top+0.5)), func(x, y int) color.RGBA {
		return trunkRamp.pick(0.85-float64(x-x0)/(2*rx-1)*0.7, x, y)
	})
	c.ellipse(cx, base, rx, 2*s, func(x, y int, nx, ny float64) color.RGBA {
		return trunkRamp.pick(0.8-(nx+1)/2*0.7, x, y)
	})
	c.ellipse(cx, top, rx, 2.5*s, func(x, y int, nx, ny float64) color.RGBA {
		d := nx*nx + ny*ny
		switch {
		case d < 0.08:
			return woodRamp[0]
		case d > 0.3 && d < 0.5:
			return woodRamp[1]
		default:
			return woodRamp[2+int(hashf(x, y, 9)*1.99)]
		}
	})
	c.outline(0.75)
	return mk(c, cx, base)
}

func genBoulder(variant int, s float64) sprite {
	c, ax, ay := boulderCanvas(variant, s)
	return mk(c, ax, ay)
}

// boulderCanvas draws boulder variant at scale s, and gives its ground point.
func boulderCanvas(variant int, s float64) (*canvas, float64, float64) {
	rng := rand.New(rand.NewPCG(uint64(variant)+7, 3))
	c := newCanvas(int(math.Ceil(30*s)), int(math.Ceil(24*s)))
	cx, cy := math.Round(15*s), 13*s
	ground := math.Round(cy + 8*s)
	ells := []struct{ x, y, rx, ry float64 }{
		{cx, cy, (10 + rng.Float64()*2) * s, (8 + rng.Float64()) * s},
		{cx - (6+rng.Float64()*2)*s, cy + 3*s, (5 + rng.Float64()*2) * s, 4.5 * s},
		{cx + (6+rng.Float64()*2)*s, cy + 2*s, (5 + rng.Float64()*2) * s, 5 * s},
	}
	// Facets: each pixel leans towards the normal of its nearest seed.
	type seedPt struct{ x, y float64 }
	var seeds []seedPt
	for i := 0; i < 7; i++ {
		seeds = append(seeds, seedPt{cx + (-11+rng.Float64()*22)*s, cy + (-8+rng.Float64()*14)*s})
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			inside := false
			for _, e := range ells {
				dx, dy := (px-e.x)/e.rx, (py-e.y)/e.ry
				if dx*dx+dy*dy <= 1 {
					inside = true
				}
			}
			if !inside || py > ground+0.5 {
				continue
			}
			best, bd := seeds[0], math.Inf(1)
			for _, s := range seeds {
				if d := math.Hypot(px-s.x, (py-s.y)*1.3); d < bd {
					best, bd = s, d
				}
			}
			gx, gy := (px-cx)/(13*s), (py-cy+2*s)/(11*s)
			fx, fy := (best.x-cx)/(13*s), (best.y-cy+2*s)/(11*s)
			l := lum(0.4*gx+0.6*fx, 0.4*gy+0.6*fy) + (vnoise(px*0.5, py*0.5, 140)-0.5)*0.2
			if py > cy+6*s {
				l -= 0.2
			}
			col := stoneRamp.pick(l, x, y)
			if gy < -0.35 && vnoise(px*0.35, py*0.5, 141+uint32(variant)) > 0.45 {
				col = mossRamp.pick(l, x, y)
			}
			c.set(x, y, col)
		}
	}
	// A crack running down from the top.
	x, y := cx+(-2+rng.Float64()*4)*s, cy-6*s
	for i := 0; i < int(math.Round(7*s)); i++ {
		if c.opaque(int(x), int(y)) {
			c.set(int(x), int(y), stoneRamp[1])
		}
		x += rng.Float64()*2 - 1
		y++
	}
	c.outline(0.8)
	return c, cx, ground
}

func genBush(berries bool) sprite {
	c := newCanvas(28, 22)
	rp := bushRamp
	if !berries {
		rp = bushDryRamp
	}
	blobs := []blob{{8, 14, 6}, {20, 14, 6}, {14, 11, 8}, {10, 8, 5}, {18, 8, 5}, {14, 15, 6}}
	leafCluster(c, blobs, 14, 11, 12, 9, rp, 160)
	if berries {
		for _, b := range [][2]int{{7, 11}, {12, 7}, {18, 10}, {22, 14}, {10, 15}, {15, 13}, {19, 5}, {5, 15}} {
			bx, by := b[0], b[1]
			c.set(bx, by, berryRamp[3])
			c.set(bx+1, by, berryRamp[2])
			c.set(bx, by+1, berryRamp[2])
			c.set(bx+1, by+1, berryRamp[1])
		}
	}
	c.outline(0.75)
	return mk(c, 14, 19)
}

func genTuft(full bool) sprite {
	rng := rand.New(rand.NewPCG(11, 12))
	c := newCanvas(22, 20)
	const bx, by = 11, 18
	// Three rows of thin blades, back to front, each row lighter.
	for row := 0; row < 3; row++ {
		n := 5 + row
		for i := 0; i < n; i++ {
			x0 := bx - 5 + (float64(i)+rng.Float64()*0.6)*10/float64(n)
			h := 13 - float64(row)*2.5 + rng.Float64()*3
			lean := (x0-bx)*0.55 + (rng.Float64()-0.5)*2.5
			if !full {
				h = 2 + rng.Float64()*2
				lean *= 0.3
			}
			for s := 0; s <= 30; s++ {
				t := float64(s) / 30
				x, y := int(math.Floor(x0+lean*t*t)), int(math.Floor(by-h*t))
				l := 0.05 + float64(row)*0.12 + t*0.65
				if !full {
					l = 0.1 + t*0.25
				}
				c.set(x, y, strawRamp.pick(l, x, y))
			}
		}
	}
	c.rect(bx-5, by, 10, 1, func(x, y int) color.RGBA {
		if c.opaque(x, y-1) {
			return strawRamp[0]
		}
		return color.RGBA{}
	})
	c.outline(0.35)
	return mk(c, bx, by)
}

// --- campfire ---------------------------------------------------------------

func genCampfire() (back, front sprite) {
	const cx, cy = 15.0, 10.0
	b, f := newCanvas(30, 18), newCanvas(30, 18)
	stones := func(c *canvas, wantFront bool) {
		for i := 0; i < 11; i++ {
			a := 2*math.Pi*float64(i)/11 + 0.2
			if (math.Sin(a) >= 0) != wantFront {
				continue
			}
			sx, sy := cx+math.Cos(a)*10, cy+math.Sin(a)*4.5
			c.ellipse(sx, sy, 2.6, 2, func(x, y int, nx, ny float64) color.RGBA {
				return stoneRamp.pick(lum(nx, ny)*0.9+0.1, x, y)
			})
		}
	}
	stones(b, false)
	b.ellipse(cx, cy, 7.5, 3.2, func(x, y int, nx, ny float64) color.RGBA {
		if hashf(x, y, 200) < 0.12 {
			return fireRamp[1+int(hashf(x, y, 201)*2)]
		}
		return ashRamp.pick(0.3+nx*0.2, x, y)
	})
	logs := [][4]float64{{cx - 7, cy + 2, cx + 7, cy - 2}, {cx - 7, cy - 2, cx + 7, cy + 2}}
	for li, l := range logs {
		for s := 0; s <= 40; s++ {
			t := float64(s) / 40
			x, y := lerpf(l[0], l[2], t), lerpf(l[1], l[3], t)
			for dy := -1; dy <= 1; dy++ {
				col := trunkRamp.pick(0.75-float64(dy+1)*0.3+float64(li)*0.1, int(x), int(y)+dy)
				if t < 0.04 || t > 0.96 {
					col = woodRamp[1+(dy+1)/2]
				}
				b.set(int(x), int(math.Floor(y))+dy, col)
			}
		}
	}
	stones(f, true)
	b.outline(0.7)
	f.outline(0.7)
	return mk(b, cx, cy), mk(f, cx, cy)
}

func genFlame(frame int) sprite {
	c := newCanvas(16, 26)
	const cx, by = 8.0, 23.0
	ph := float64(frame) * 1.7
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			h := (by - py) / 21 // 0 at the base, 1 at the tip
			if h < -0.05 || h > 1 {
				continue
			}
			w := math.Pow(1-math.Max(h, 0), 0.8) * math.Min(1, h*4+0.55) * 6.5
			dx := px - cx + (fbm2(px*0.22, py*0.18+ph, 210, 2)-0.5)*7*h
			v := 1 - math.Abs(dx)/math.Max(w, 0.01) - h*0.35 + (vnoise(px*0.4, py*0.3+ph*1.3, 211)-0.5)*0.5
			if v <= 0.05 {
				continue
			}
			c.set(x, y, fireRamp.pick(v*1.35, x, y))
		}
	}
	return mk(c, cx, by)
}

func genTombstone() sprite {
	c := newCanvas(16, 20)
	shade := func(x, y int, nx float64) color.RGBA {
		return stoneRamp.pick(0.75-(nx+1)/2*0.55, x, y)
	}
	c.ellipse(8, 6, 5, 4.5, func(x, y int, nx, ny float64) color.RGBA { return shade(x, y, nx) })
	c.rect(3, 6, 10, 11, func(x, y int) color.RGBA { return shade(x, y, (float64(x)-7.5)/5) })
	c.rect(7, 5, 2, 8, solid(stoneRamp[1]))
	c.rect(5, 7, 6, 2, solid(stoneRamp[1]))
	c.rect(2, 16, 12, 2, func(x, y int) color.RGBA { return stoneRamp.pick(0.3, x, y) })
	for _, g := range [][2]int{{2, 15}, {4, 16}, {12, 15}, {13, 16}, {10, 16}} {
		c.set(g[0], g[1], grassRamp[3])
		c.set(g[0], g[1]+1, grassRamp[2])
	}
	c.outline(0.75)
	return mk(c, 8, 17)
}

// --- player -----------------------------------------------------------------

// mirrorCache holds the mirror images made by flipped, by source image.
var mirrorCache = map[*ebiten.Image]sprite{}

// flipped is mirrorSprite for drawing every frame: each image is mirrored
// once and kept.
func flipped(s sprite) sprite {
	m, ok := mirrorCache[s.img]
	if !ok {
		m = mirrorSprite(sprite{img: s.img, ay: s.ay, dens: s.dens})
		mirrorCache[s.img] = m
	}
	return sprite{img: m.img, ax: s.w() - s.ax, ay: s.ay, dens: s.dens}
}

func mirrorSprite(s sprite) sprite {
	w, h := s.img.Bounds().Dx(), s.img.Bounds().Dy()
	img := ebiten.NewImage(w, h)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(-1, 1)
	op.GeoM.Translate(float64(w), 0)
	img.DrawImage(s.img, op) // in image pixels: the mirror keeps s's density
	return sprite{img: img, ax: s.w() - s.ax, ay: s.ay, dens: s.dens}
}

// --- effects ----------------------------------------------------------------

func genSoftDisc(size int, hardness float64) *ebiten.Image {
	c := newCanvas(size, size)
	r := float64(size) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			d := math.Hypot(float64(x)+0.5-r, float64(y)+0.5-r) / r
			a := clamp01((1 - d) * 3 * hardness)
			c.px[y*size+x] = color.RGBA{0, 0, 0, uint8(smooth(a) * 255)}
		}
	}
	return c.image()
}

// genLight is a soft white disc for punching holes in the darkness and for
// additive fire glow. Pixels are alpha-premultiplied, as Ebitengine expects.
func genLight() *ebiten.Image {
	c := newCanvas(lightSize, lightSize)
	r := float64(lightSize) / 2
	for y := 0; y < lightSize; y++ {
		for x := 0; x < lightSize; x++ {
			d := math.Hypot(float64(x)+0.5-r, float64(y)+0.5-r) / r
			a := 1.0
			if d > 0.35 {
				t := min(1, (d-0.35)/0.65)
				a = 1 - t*t*(3-2*t)
			}
			c.px[y*lightSize+x] = color.RGBA{255, 255, 255, uint8(math.Round(a * 255))}
		}
	}
	return c.image()
}

func genCloud() *ebiten.Image {
	const w, h = 160, 96
	c := newCanvas(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := (float64(x)-w/2)/(w/2), (float64(y)-h/2)/(h/2)
			fall := clamp01(1 - math.Sqrt(dx*dx+dy*dy))
			n := fbm2(float64(x)/28, float64(y)/18, 300, 4)
			a := clamp01((n*fall*2.2 - 0.35) * 3)
			c.px[y*w+x] = color.RGBA{0, 0, 0, uint8(smooth(a) * 255)}
		}
	}
	return c.image()
}

func genVignette() *ebiten.Image {
	const s = 128
	c := newCanvas(s, s)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			dx, dy := (float64(x)+0.5)/s*2-1, (float64(y)+0.5)/s*2-1
			d := math.Sqrt(dx*dx*0.8 + dy*dy*1.1)
			c.px[y*s+x] = color.RGBA{0, 0, 0, uint8(smooth(clamp01((d-0.6)/0.7)) * 255)}
		}
	}
	return c.image()
}

func genMarker() *ebiten.Image {
	c := newCanvas(11, 8)
	c.polygon([]float64{1, 1, 10, 1, 5.5, 7}, func(x, y int) color.RGBA {
		if y <= 2 {
			return colWhite
		}
		return hex(0xf6d55c)
	})
	c.outline(1)
	return c.image()
}

// genTwig is a fallen, forked branch lying in the grass with a few leaves.
func genTwig() sprite {
	c := newCanvas(24, 14)
	stick := func(x0, y0, x1, y1 float64, lit float64) {
		c.line(x0, y0, x1, y1, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(lit+0.15, x, y) })
		c.line(x0, y0+1, x1, y1+1, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(lit-0.35, x, y) })
	}
	stick(2, 9, 20, 6, 0.75)
	stick(11, 7, 16, 2, 0.7)
	stick(6, 8, 4, 4, 0.6)
	for _, l := range [][2]int{{16, 1}, {17, 2}, {20, 5}, {21, 6}, {3, 3}} {
		c.set(l[0], l[1], leafRamps[1][4])
		c.set(l[0]+1, l[1], leafRamps[1][3])
	}
	c.set(2, 9, woodRamp[2]) // the broken end
	c.outline(0.75)
	return mk(c, 12, 9)
}

// --- icons ------------------------------------------------------------------

func genIcons() [sim.NumItems]*ebiten.Image {
	var icons [sim.NumItems]*ebiten.Image

	wood := newCanvas(16, 16)
	for _, lg := range [][2]int{{1, 3}, {3, 8}} {
		x0, y0 := lg[0], lg[1]
		wood.rect(x0, y0, 10, 5, func(x, y int) color.RGBA {
			i := trunkRamp.idx(0.95-float64(y-y0)*0.2, x, y)
			if hashf(x/3, y, 5) < 0.3 {
				i--
			}
			return trunkRamp.at(i)
		})
		wood.ellipse(float64(x0)+10.5, float64(y0)+2.5, 2, 2.6, func(x, y int, nx, ny float64) color.RGBA {
			if nx*nx+ny*ny < 0.25 {
				return woodRamp[1]
			}
			return woodRamp[3]
		})
	}
	wood.outline(0.8)
	icons[sim.ItemWood] = wood.image()

	stone := newCanvas(16, 16)
	stone.ellipse(6, 10, 5, 4, func(x, y int, nx, ny float64) color.RGBA { return stoneRamp.pick(lum(nx, ny)*0.8, x, y) })
	stone.ellipse(10, 8, 5, 5, func(x, y int, nx, ny float64) color.RGBA { return stoneRamp.pick(lum(nx, ny), x, y) })
	stone.outline(0.8)
	icons[sim.ItemStone] = stone.image()

	grass := newCanvas(16, 16)
	for i := -3; i <= 3; i++ {
		grass.line(8, 15, 8+float64(i)*1.6, 1+math.Abs(float64(i))*0.8, func(x, y int, t float64) color.RGBA {
			return strawRamp.pick(0.2+t*0.8-float64((i+3)%2)*0.15, x, y)
		})
	}
	grass.rect(6, 10, 5, 2, func(x, y int) color.RGBA { return beltRamp[1-(y-10)] })
	grass.outline(0.7)
	icons[sim.ItemGrass] = grass.image()

	berry := newCanvas(16, 16)
	berry.ellipse(11, 4, 3, 2, func(x, y int, nx, ny float64) color.RGBA { return bushRamp.pick(lum(nx, ny), x, y) })
	for _, b := range [][2]float64{{5.5, 10}, {10.5, 10.5}, {8, 6.5}} {
		berry.ellipse(b[0], b[1], 3.3, 3.3, func(x, y int, nx, ny float64) color.RGBA {
			if nx < -0.2 && nx > -0.65 && ny < -0.2 && ny > -0.65 {
				return berryRamp[3]
			}
			return berryRamp.pick(lum(nx, ny), x, y)
		})
	}
	berry.outline(0.8)
	icons[sim.ItemBerry] = berry.image()

	stick := newCanvas(16, 16)
	stick.line(2, 14, 13, 2, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.85, x, y) })
	stick.line(3, 14, 14, 2, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.35, x, y) })
	stick.line(8, 8, 4, 4, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.7, x, y) })
	stick.set(3, 3, leafRamps[1][4])
	stick.set(4, 3, leafRamps[1][3])
	stick.set(13, 1, leafRamps[1][4])
	stick.outline(0.8)
	icons[sim.ItemStick] = stick.image()

	return icons
}

func genHeart() *ebiten.Image {
	c := newCanvas(13, 12)
	rp := mkRamp(0x7a1426, 0xc42a3c, 0xee4f5a, 0xff9a9a)
	f := func(x, y int, nx, ny float64) color.RGBA { return rp.pick(lum(nx, ny), x, y) }
	c.ellipse(4, 4, 3.5, 3.5, f)
	c.ellipse(9, 4, 3.5, 3.5, f)
	c.polygon([]float64{0.6, 4.5, 12.4, 4.5, 6.5, 11}, func(x, y int) color.RGBA {
		return rp.pick(0.75-float64(y)*0.07-float64(x)*0.02, x, y)
	})
	c.set(3, 2, rp[3])
	c.set(2, 3, rp[3])
	c.outline(0.9)
	return c.image()
}

func genFood() *ebiten.Image {
	c := newCanvas(13, 13)
	meat := mkRamp(0x6b2d17, 0x9c4a22, 0xc7703a, 0xe8a060)
	c.line(7, 7, 11, 11, func(x, y int, t float64) color.RGBA { return hex(0xece4d2) })
	c.line(7, 8, 11, 12, func(x, y int, t float64) color.RGBA { return hex(0xc9bea6) })
	c.set(11, 12, hex(0xece4d2))
	c.set(12, 11, hex(0xece4d2))
	c.ellipse(5, 5, 4.5, 4, func(x, y int, nx, ny float64) color.RGBA { return meat.pick(lum(nx, ny), x, y) })
	c.outline(0.9)
	return c.image()
}
