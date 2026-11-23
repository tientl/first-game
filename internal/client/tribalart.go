package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/sim"
)

// Art of the tribal age: cave spiders and bears, the tribes and their
// villages, colourful fish, flowers, glowing plants, man-eating plants,
// things to build, old chests, bodies and trophy heads. Creatures are drawn
// side-on facing right and mirrored for left, like the other animals.

var (
	spiderRamp  = mkRamp(0x120c16, 0x21162a, 0x32223e, 0x483454, 0x624a6c)
	spiderMark  = mkRamp(0x7a1010, 0xc0281c, 0xf05a30)
	bearRamp    = mkRamp(0x24160f, 0x3a2418, 0x553522, 0x6f4a2f, 0x8a603e, 0xa77b52)
	tribeSkin   = mkRamp(0x1e120d, 0x2f1c14, 0x45291c, 0x5c3926, 0x744a31)
	paintRamp   = mkRamp(0xc8c0b0, 0xf2ece0)
	loinRamp    = mkRamp(0x4a2e1a, 0x6c4428, 0x8f5d36, 0xb07a48)
	thatchRamp  = mkRamp(0x5a4420, 0x7d602c, 0xa1803a, 0xc4a04e, 0xe0c46e)
	plankRamp   = mkRamp(0x4a301c, 0x654126, 0x825633, 0x9e6c42, 0xb98552)
	glowCyan    = mkRamp(0x0c3a4a, 0x146a7a, 0x22a8b4, 0x56e2e2, 0xc0fff8)
	glowViolet  = mkRamp(0x2a1048, 0x4a1c7a, 0x7a34b4, 0xb066e8, 0xe8b8ff)
	glowStem    = mkRamp(0x10202a, 0x18323a, 0x234a4a, 0x2e5e58)
	maneatRamp  = mkRamp(0x1c3a14, 0x2a561c, 0x3c7424, 0x55922e, 0x74b03a)
	maneatMouth = mkRamp(0x5a0a14, 0x8e1424, 0xc42a36, 0xe8586a)
	bloodRamp   = mkRamp(0x3a0408, 0x5e0a10, 0x82141a, 0xa8202a)
	goldRamp    = mkRamp(0x6a4a10, 0xa07a1c, 0xd4aa34, 0xf4d870, 0xfff4c0)
	bandRamp    = mkRamp(0x24262c, 0x3a3e46, 0x565c66, 0x7a828c)
)

// fishLooks are the colours of fish, by id: clownfish, blue tang, golden
// carp, koi, rainbow trout and a green-gold perch.
var fishLooks = []struct {
	body, fin ramp
	stripes   color.RGBA // bands across the body; zero for none
	spots     color.RGBA // blotches; zero for none
}{
	{mkRamp(0x8a2a06, 0xd04a10, 0xf87a24, 0xffa850, 0xffd8a0), mkRamp(0x8a2a06, 0xd04a10, 0xf87a24), hex(0xfaf8f0), color.RGBA{}},
	{mkRamp(0x0e2a6e, 0x1a48b0, 0x2e6ee0, 0x5a98f4, 0xa8d0ff), mkRamp(0xb08a10, 0xe8c420, 0xffe460), color.RGBA{}, hex(0x0a1440)},
	{mkRamp(0x7a5a08, 0xc09010, 0xf0c020, 0xffe060, 0xfff6c0), mkRamp(0xa06808, 0xe09a18, 0xffc040), color.RGBA{}, color.RGBA{}},
	{mkRamp(0x9a9890, 0xd0ccc0, 0xf4f0e6, 0xffffff, 0xffffff), mkRamp(0xb0a8a0, 0xe8e0d8, 0xffffff), color.RGBA{}, hex(0xe8401c)},
	{mkRamp(0x2a4a3a, 0x4a7a5e, 0x80b090, 0xc0dcc4, 0xeef8f0), mkRamp(0x3a5a4a, 0x5e8a6e, 0x8ab89a), hex(0xf07a9a), hex(0x283a30)},
	{mkRamp(0x2a4a10, 0x4a7a18, 0x7aa82a, 0xb0d040, 0xe0f080), mkRamp(0xa04410, 0xe06a20, 0xff9a40), hex(0x1c3010), color.RGBA{}},
}

const (
	spiderWalkA = iota
	spiderWalkB
	spiderSpit
	spiderBite
	spiderPoses
)

const (
	bearWalkA = iota
	bearWalkB
	bearClaw
	bearBite
	bearPoses
)

const (
	tribeStand = iota
	tribeWalkA
	tribeWalkB
	tribeThrust
	tribePoses
)

// tribalArt holds the sprites of the tribal age.
type tribalArt struct {
	spider [2][spiderPoses]sprite
	bear   [2][bearPoses]sprite
	tribe  [2][tribePoses]sprite
	fish   [][2][fishFrames]sprite // by look, side, frame

	flowers      [sim.NumFlowers]sprite
	flowerPicked sprite
	saplings     [sim.NumSeeds]sprite
	glow         [3]sprite // mushrooms, fern, tree
	glowMask     [3]sprite // just their glowing parts, drawn bright at night
	glowPicked   [3]sprite
	manEater     [2]sprite // waiting, snapping
	door, floor  sprite
	spikes       sprite
	trapSet      sprite
	trapSprung   sprite
	torchStick   sprite
	treasure     [2]sprite // shut, open
	hut, totem   sprite
	bonfire      sprite
	blood        [3]sprite // pools under small, middling and big bodies
	post         sprite    // the stake a trophy head sits on
	web          sprite
	webbed       sprite // strands over a webbed player
	tornado      [4]sprite
}

func genTribalArt() tribalArt {
	var a tribalArt
	for p := range spiderPoses {
		a.spider[0][p], a.spider[1][p] = sidedPair(genSpider(p))
	}
	for p := range bearPoses {
		a.bear[0][p], a.bear[1][p] = sidedPair(genBear(p))
	}
	for p := range tribePoses {
		a.tribe[0][p], a.tribe[1][p] = sidedPair(genTribesman(p))
	}
	a.fish = make([][2][fishFrames]sprite, len(fishLooks))
	for v := range fishLooks {
		for f := range fishFrames {
			a.fish[v][0][f], a.fish[v][1][f] = sidedPair(genFishLook(v, f))
		}
	}
	for f := range sim.NumFlowers {
		a.flowers[f] = genFlower(f)
	}
	a.flowerPicked = genFlowerPicked()
	for s := range sim.NumSeeds {
		a.saplings[s] = genSapling(sim.SeedKind(s))
	}
	for v := range 3 {
		a.glow[v], a.glowMask[v] = genGlowPlant(v, true)
		a.glowPicked[v], _ = genGlowPlant(v, false)
	}
	a.manEater[0], a.manEater[1] = genManEater(false), genManEater(true)
	a.door, a.floor, a.spikes = genDoor(), genFloor(), genSpikes()
	a.trapSet, a.trapSprung = genTrap(true), genTrap(false)
	a.torchStick = genTorchStick()
	a.treasure[0], a.treasure[1] = genTreasure(false), genTreasure(true)
	a.hut, a.totem, a.bonfire = genHut(), genTotem(), genBonfire()
	for i, w := range []float64{9, 15, 24} {
		a.blood[i] = genBlood(w, uint32(i))
	}
	a.post = genPost()
	a.web, a.webbed = genWebBlob(), genWebbed()
	for f := range 4 {
		a.tornado[f] = genTornado(f)
	}
	return a
}

// --- creatures ----------------------------------------------------------------

func genSpider(pose int) sprite {
	c := newCanvas(26, 16)
	lift := 0.0
	if pose == spiderSpit {
		lift = -1.5 // rears its abdomen up to spit
	}
	// Eight legs, four seen on each side, bent at the knee.
	for i := range 4 {
		ph := float64(i)
		if pose == spiderWalkB {
			ph += 0.5
		}
		swing := math.Sin(ph*math.Pi) * 1.4
		hip := 13.5 + float64(i)*1.3
		for _, far := range []bool{true, false} {
			k := 1.0
			sh := 0.25
			if far {
				k, sh = 0.8, -0.1
			}
			kx := hip + (float64(i)-1.5)*3.2*k + swing
			ky := 4.5 + lift*0.2
			fx := hip + (float64(i)-1.5)*5.6*k + swing*1.6
			fy := 13.5
			if far {
				fy = 12.5
			}
			col := func(_, _ int, t float64) color.RGBA { return spiderRamp.pick(0.75+sh-t*0.25, 0, 0) }
			for _, o := range []float64{0, 0.6} {
				c.line(hip, 8+o, kx, ky+o, col)
				c.line(kx+o, ky, fx+o, fy, col)
			}
			c.set(int(kx), int(ky)-1, spiderRamp.at(4)) // the knee, lit
		}
	}
	// The abdomen, round and glossy, with a red mark; the head in front.
	ax := 8.0
	roundShape(c, ell{ax, 8 + lift, 6.2, 4.6}, spiderRamp, 0.08)
	c.ellipse(ax-0.5, 7.6+lift, 2.2, 1.3, func(x, y int, nx, ny float64) color.RGBA {
		return spiderMark.pick(0.6-ny*0.3, x, y)
	})
	hx := 16.0
	if pose == spiderBite {
		hx = 17.5
	}
	roundShape(c, ell{hx, 9, 3.4, 2.8}, spiderRamp, 0.15)
	// Eyes: a cluster of red beads.
	for _, e := range [][2]float64{{1.8, -1.2}, {2.4, -0.4}, {1.2, -0.6}} {
		c.set(int(hx+e[0]), int(9+e[1]), hex(0xff3a2a))
	}
	// Fangs.
	fy := 11.0
	if pose == spiderBite {
		fy = 12.5
	}
	c.line(hx+2.2, 10.5, hx+2.8, fy, func(int, int, float64) color.RGBA { return hex(0xd8d0c0) })
	if pose == spiderSpit {
		roundShape(c, ell{2.5, 4.5, 1.6, 1.4}, ramp{hex(0xc8c8d0), hex(0xf4f4f8)}, 0.2)
	}
	c.outline(0.8)
	return mk(c, 12, 14)
}

func genBear(pose int) sprite {
	c := newCanvas(40, 30)
	rear := pose == bearClaw
	legX := []float64{10, 14, 25, 29}
	off := []float64{0, 0, 0, 0}
	switch pose {
	case bearWalkA:
		off = []float64{1.5, -1.5, -1.5, 1.5}
	case bearWalkB:
		off = []float64{-1.5, 1.5, 1.5, -1.5}
	}
	leg := func(x, dx float64, far bool) {
		b := 0.0
		if far {
			b = -0.2
		}
		for y := 19; y <= 26; y++ {
			for xx := int(x + dx); xx < int(x+dx)+4; xx++ {
				c.set(xx, y, bearRamp.pick(0.45+b-float64(xx-int(x+dx))*0.08, xx, y))
			}
		}
		c.rect(int(x+dx)-1, 26, 5, 1, func(x, y int) color.RGBA { return bearRamp.at(0) })
	}
	if rear {
		// Up on its hind legs, one great paw raised.
		leg(12, 0, true)
		leg(16, 0, false)
		roundShape(c, ell{16, 13, 8, 9.5}, bearRamp, 0.05)
		roundShape(c, ell{20, 6, 4.6, 4}, bearRamp, 0.15) // head
		roundShape(c, ell{24, 7, 2.4, 1.8}, bearRamp, 0.3)
		c.set(25, 6, colInk)
		c.set(21, 4, colInk)
		roundShape(c, ell{18, 2.5, 1.4, 1.3}, bearRamp, 0)
		c.line(20, 10, 28, 4, func(x, y int, t float64) color.RGBA { return bearRamp.pick(0.5, x, y) })
		c.line(20, 11, 28, 5, func(x, y int, t float64) color.RGBA { return bearRamp.pick(0.4, x, y) })
		for k := range 3 {
			c.set(29, 2+k, hex(0xe8e0d0)) // claws
		}
		c.outline(0.8)
		return mk(c, 15, 27)
	}
	leg(legX[0], off[0], true)
	leg(legX[2], off[2], true)
	// The body: a heavy barrel with a hump over the shoulders.
	roundShape(c, ell{19, 15, 12.5, 6.8}, bearRamp, 0)
	roundShape(c, ell{24, 11, 6, 4.4}, bearRamp, 0.08)
	leg(legX[1], off[1], false)
	leg(legX[3], off[3], false)
	// The head, low and forward.
	hx, hy := 32.0, 14.0
	if pose == bearBite {
		hx, hy = 33.5, 16.0
	}
	roundShape(c, ell{hx, hy, 4.8, 4.2}, bearRamp, 0.12)
	roundShape(c, ell{hx + 3.6, hy + 1.2, 2.6, 2}, bearRamp, 0.3)
	c.set(int(hx+5.6), int(hy+0.6), colInk) // nose
	c.set(int(hx+1), int(hy-1.4), colInk)   // eye
	roundShape(c, ell{hx - 2, hy - 4, 1.5, 1.4}, bearRamp, 0)
	if pose == bearBite {
		c.rect(int(hx+2), int(hy+3), 4, 2, func(x, y int) color.RGBA { return maneatMouth.at(1) })
		c.set(int(hx+2), int(hy+3), hex(0xf0ece0))
		c.set(int(hx+5), int(hy+3), hex(0xf0ece0))
	}
	c.outline(0.8)
	return mk(c, 19, 27)
}

// genTribesman draws a warrior side-on, facing right: dark skin, bare chest
// painted with white stripes, a hide loincloth, a bone necklace, a crest of
// black hair with a feather, and a stone-tipped spear.
func genTribesman(pose int) sprite {
	c := newCanvas(30, 34)
	const fx = 12.0 // the line of the body
	step := 0.0
	switch pose {
	case tribeWalkA:
		step = 1.6
	case tribeWalkB:
		step = -1.6
	}
	limb := func(x0, y0, x1, y1, lit float64) {
		for o := -0.6; o <= 0.6; o += 0.6 {
			c.line(x0+o, y0, x1+o, y1, func(x, y int, t float64) color.RGBA { return tribeSkin.pick(lit+o*0.2, x, y) })
		}
	}
	// Legs.
	limb(fx-0.5, 21, fx-0.5-step, 30, 0.3)
	limb(fx+1, 21, fx+1+step, 30, 0.6)
	c.rect(int(fx-1.5-step), 30, 3, 1, func(x, y int) color.RGBA { return tribeSkin.at(0) })
	c.rect(int(fx+0+step), 30, 3, 1, func(x, y int) color.RGBA { return tribeSkin.at(1) })
	// The torso: broad and bare.
	c.ellipse(fx+0.5, 16, 3.6, 5.6, func(x, y int, nx, ny float64) color.RGBA {
		return tribeSkin.pick(lum(nx, ny)+0.15, x, y)
	})
	// War paint: white stripes across the chest.
	for _, y := range []int{14, 16} {
		for x := int(fx) - 1; x <= int(fx)+3; x++ {
			if c.opaque(x, y) && (x+y)%3 != 0 {
				c.set(x, y, paintRamp.at(1))
			}
		}
	}
	// A loincloth of hide.
	c.polygon([]float64{fx - 3, 20, fx + 4, 20, fx + 3.4, 24, fx + 1, 23, fx - 2.6, 24.4}, func(x, y int) color.RGBA {
		return loinRamp.pick(0.7-float64(y-20)*0.1+(hashf(x, y, 88)-0.5)*0.2, x, y)
	})
	c.rect(int(fx-3), 20, 8, 1, func(x, y int) color.RGBA { return loinRamp.at(0) })
	// A necklace of bones.
	for _, x := range []int{int(fx) - 1, int(fx) + 1, int(fx) + 3} {
		c.set(x, 12, colBone)
		c.set(x, 13, colBoneSh)
	}
	// The head: a fierce face, heavy brow, white paint across the eyes.
	hx, hy := fx+1.5, 7.0
	c.ellipse(hx, hy, 3.6, 3.8, func(x, y int, nx, ny float64) color.RGBA {
		return tribeSkin.pick(lum(nx, ny)+0.2, x, y)
	})
	for x := int(hx); x <= int(hx)+3; x++ {
		c.set(x, int(hy)-1, paintRamp.at(1)) // a band of paint
	}
	c.set(int(hx)+2, int(hy)-1, hex(0xd02818)) // a red eye-glint in the paint: angry
	c.set(int(hx)+1, int(hy)-2, tribeSkin.at(0))
	c.set(int(hx)+2, int(hy)-2, tribeSkin.at(0)) // the scowling brow
	c.set(int(hx)+3, int(hy)-2, tribeSkin.at(0))
	c.set(int(hx)+3, int(hy)+2, hex(0x200c0a)) // a grim mouth
	c.set(int(hx)+2, int(hy)+2, hex(0x200c0a))
	// A crest of black hair, and a feather.
	for x := int(hx) - 3; x <= int(hx)+1; x++ {
		top := int(hy) - 4 - (x-int(hx)+3)%2
		for y := top; y <= int(hy)-2; y++ {
			c.set(x, y, hex(0x0e0a0c))
		}
	}
	c.line(hx-3, hy-4, hx-6, hy-10, func(x, y int, t float64) color.RGBA { return mix(hex(0xd8381c), hex(0xf8e8c0), t) })
	// The spear, and the arm that holds it.
	if pose == tribeThrust {
		limb(fx+1, 12, fx+7, 14, 0.7)
		c.line(fx-6, 14.5, fx+16, 13.5, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.7, x, y) })
		c.polygon([]float64{fx + 16, 12, fx + 20, 13.5, fx + 16, 15}, solid(stoneRamp.at(4)))
	} else {
		limb(fx+1, 12, fx+4, 18, 0.7)
		sx := fx + 4.5
		c.line(sx, 30, sx+1, 2, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.7, x, y) })
		c.polygon([]float64{sx - 0.5, 3, sx + 1, -1.5, sx + 2.6, 3}, solid(stoneRamp.at(4)))
	}
	c.outline(0.8)
	return mk(c, fx, 31)
}

func genFishLook(v, frame int) sprite {
	look := fishLooks[v%len(fishLooks)]
	c := newCanvas(17, 10)
	tail := []float64{4.5, 5, 0.5, 1.6, 0.5, 8.4}
	if frame == 1 {
		tail = []float64{4.5, 5, 1.2, 2.4, 0.2, 7.8}
	}
	c.polygon(tail, func(x, y int) color.RGBA { return look.fin.pick(0.8-float64(y)*0.06, x, y) })
	c.polygon([]float64{6.5, 3, 10.5, 0.6, 11.5, 3.2}, solid(look.fin.at(1))) // dorsal fin
	c.polygon([]float64{9, 7, 10.5, 9.2, 11.5, 7}, solid(look.fin.at(2)))     // belly fin
	c.ellipse(9.2, 5, 5.4, 2.8, func(x, y int, nx, ny float64) color.RGBA {
		l := lum(nx, ny)
		if ny > 0.4 {
			l += 0.25 // a pale belly
		}
		col := look.body.pick(l, x, y)
		if look.stripes.A > 0 && (math.Abs(nx-0.1) < 0.12 || math.Abs(nx+0.45) < 0.11) {
			col = look.stripes
		}
		if look.spots.A > 0 && hashf(x/2, y/2, uint32(v)*31+5) < 0.28 && ny < 0.5 {
			col = look.spots
		}
		return col
	})
	c.set(13, 4, colWhite)
	c.set(13, 5, colInk)
	c.outline(0.6)
	return mk(c, 8, 7)
}

// --- plants ------------------------------------------------------------------

var flowerColors = [sim.NumFlowers]ramp{
	mkRamp(0x7a0a10, 0xc41a20, 0xf04040, 0xff8a7a), // poppies
	mkRamp(0x8a5a08, 0xd09a10, 0xf8cc28, 0xfff08a), // daisies, yellow
	mkRamp(0x1a2a8a, 0x2c48c8, 0x5078f0, 0xa0c0ff), // cornflowers
	mkRamp(0x3a1a6a, 0x6a34a8, 0x9a5ad8, 0xd0a8f8), // lavender
	mkRamp(0x9a948a, 0xd8d4cc, 0xf8f6f0, 0xffffff), // white daisies
	mkRamp(0x8a1a4a, 0xc83a7a, 0xf070a8, 0xffb8d8), // wild roses
}

func genFlower(f sim.FlowerKind) sprite {
	c := newCanvas(14, 14)
	rp := flowerColors[f]
	heads := [][2]float64{{4, 6}, {9, 4}, {10, 8.5}, {6, 9.5}}
	for _, h := range heads {
		c.line(h[0], h[1], 7, 13, func(x, y int, t float64) color.RGBA { return leafRamps[0].pick(0.5+t*0.1, x, y) })
	}
	c.ellipse(4, 12.5, 2.6, 1, func(x, y int, nx, ny float64) color.RGBA { return leafRamps[0].pick(0.6, x, y) })
	c.ellipse(10, 12.5, 2.4, 1, func(x, y int, nx, ny float64) color.RGBA { return leafRamps[0].pick(0.5, x, y) })
	for i, h := range heads {
		switch f {
		case sim.FlowerPurple: // lavender: a spike of tiny blooms
			for k := range 4 {
				c.set(int(h[0]), int(h[1])-k+1, rp.at(1+k%3))
			}
		default:
			r := 1.7
			if i%2 == 1 {
				r = 1.4
			}
			roundShape(c, ell{h[0], h[1], r, r * 0.9}, rp, 0.1)
			centre := hex(0x3a2408)
			if f == sim.FlowerWhite || f == sim.FlowerYellow {
				centre = hex(0xe8a818)
			}
			c.set(int(h[0]), int(h[1]), centre)
		}
	}
	c.outline(0.6)
	return mk(c, 7, 13)
}

func genFlowerPicked() sprite {
	c := newCanvas(10, 8)
	for i := range 4 {
		x := 2 + float64(i)*2
		c.line(x, 7, x+0.5, 3+float64(i%2), func(x, y int, t float64) color.RGBA { return leafRamps[0].pick(0.45, x, y) })
	}
	return mk(c, 5, 7)
}

func genSapling(s sim.SeedKind) sprite {
	c := newCanvas(12, 12)
	leaf := leafRamps[1]
	tip := leaf.at(4)
	switch s {
	case sim.SeedGlow:
		leaf, tip = glowStem, glowCyan.at(3)
	case sim.SeedManEater:
		leaf, tip = maneatRamp, maneatMouth.at(2)
	case sim.SeedBerry:
		tip = berryRamp.at(2)
	case sim.SeedFlower:
		tip = flowerColors[1].at(2)
	}
	c.ellipse(6, 11, 3.4, 1, func(x, y int, nx, ny float64) color.RGBA { return hex(0x4a3420) }) // turned earth
	c.line(6, 11, 6, 4, func(x, y int, t float64) color.RGBA { return leaf.pick(0.4, x, y) })
	roundShape(c, ell{4, 6, 2, 1.1}, leaf, 0.1)
	roundShape(c, ell{8, 5, 2, 1.1}, leaf, 0.2)
	c.set(6, 3, tip)
	if s == sim.SeedTree {
		roundShape(c, ell{6, 3.5, 2.4, 2}, leaf, 0.25)
	}
	c.outline(0.6)
	return mk(c, 6, 11)
}

// genGlowPlant draws a plant that glows at night, and a mask of just its
// glowing parts. lit is false for one picked bare, which does not glow.
func genGlowPlant(v int, lit bool) (sprite, sprite) {
	type part struct {
		e    ell
		glow bool
		rp   ramp
	}
	var w, h int
	var ax, ay float64
	var parts []part
	var stems [][4]float64
	switch v {
	case 0: // a cluster of mushrooms with glowing caps
		w, h, ax, ay = 18, 16, 9, 15
		for _, m := range [][3]float64{{5, 9, 3}, {10, 6, 4}, {14, 10, 2.6}, {8, 12, 2}} {
			stems = append(stems, [4]float64{m[0], m[1], m[0], 15})
			parts = append(parts, part{ell{m[0], m[1], m[2], m[2] * 0.6}, true, glowCyan})
		}
	case 1: // a fern whose fronds end in violet lights
		w, h, ax, ay = 22, 20, 11, 19
		for i := range 6 {
			a := -math.Pi/2 + (float64(i)-2.5)*0.42
			x1, y1 := 11+math.Cos(a)*9, 19+math.Sin(a)*15
			stems = append(stems, [4]float64{11, 19, x1, y1})
			parts = append(parts, part{ell{x1, y1, 1.6, 1.6}, true, glowViolet})
			parts = append(parts, part{ell{(11 + x1) / 2, (19 + y1) / 2, 1.2, 1}, true, glowViolet})
		}
	default: // a tall tree, its twisted trunk hung with luminous strands
		w, h, ax, ay = 34, 52, 17, 51
		stems = append(stems, [4]float64{17, 51, 15, 30}, [4]float64{15, 30, 18, 14})
		for i := range 7 {
			a := -math.Pi/2 + (float64(i)-3)*0.5
			x1, y1 := 18+math.Cos(a)*12, 16+math.Sin(a)*9
			stems = append(stems, [4]float64{18, 15, x1, y1})
			parts = append(parts, part{ell{x1, y1, 3.2, 2.4}, false, glowStem})
			for k := 1; k <= 3+i%3; k++ {
				parts = append(parts, part{ell{x1, y1 + float64(k)*3.2, 0.9, 1.1}, true, []ramp{glowCyan, glowViolet}[i%2]})
			}
		}
	}
	c, m := newCanvas(w, h), newCanvas(w, h)
	for _, s := range stems {
		thick := 0.0
		if v == 2 && s[1] > 20 {
			thick = 1
		}
		for o := -thick; o <= thick; o++ {
			c.line(s[0]+o, s[1], s[2]+o, s[3], func(x, y int, t float64) color.RGBA { return glowStem.pick(0.5-o*0.2, x, y) })
		}
	}
	for _, p := range parts {
		rp := p.rp
		if !lit && p.glow {
			rp = glowStem
		}
		roundShape(c, p.e, rp, 0.15)
		if p.glow && lit {
			m.ellipse(p.e.x, p.e.y, p.e.rx, p.e.ry, func(x, y int, nx, ny float64) color.RGBA {
				return p.rp.pick(0.9-ny*0.2, x, y)
			})
		}
	}
	c.outline(0.6)
	return mk(c, ax, ay), mk(m, ax, ay)
}

// genManEater draws a man-eating plant: a thick stalk rising from broad
// leaves to a great pair of toothed jaws, open and waiting, or snapped
// shut and lunging forward.
func genManEater(snap bool) sprite {
	c := newCanvas(32, 36)
	// Leaves at the foot.
	for _, l := range []ell{{8, 32, 7, 2.6}, {24, 32, 7, 2.6}, {16, 33, 6, 2.2}} {
		roundShape(c, l, maneatRamp, 0)
	}
	hx, hy := 16.0, 13.0
	if snap {
		hx, hy = 20.0, 15.0
	}
	for o := -1.0; o <= 1.0; o++ {
		c.line(16+o, 32, hx+o-1, hy+4, func(x, y int, t float64) color.RGBA { return maneatRamp.pick(0.45-o*0.15, x, y) })
	}
	if snap {
		roundShape(c, ell{hx, hy, 8, 5.4}, maneatRamp, 0.05)
		c.line(hx-7, hy, hx+7, hy, func(x, y int, t float64) color.RGBA { return maneatMouth.at(0) })
		for x := int(hx) - 6; x <= int(hx)+6; x += 2 {
			c.set(x, int(hy)-1, hex(0xf4f0e0))
			c.set(x+1, int(hy)+1, hex(0xf4f0e0))
		}
	} else {
		// Two lobes hinged at the back, gaping, red inside.
		c.ellipse(hx, hy-5, 8, 4.4, func(x, y int, nx, ny float64) color.RGBA {
			if ny > 0.2 {
				return maneatMouth.pick(0.8-ny*0.3, x, y)
			}
			return maneatRamp.pick(lum(nx, ny)+0.1, x, y)
		})
		c.ellipse(hx, hy+4, 8, 4.4, func(x, y int, nx, ny float64) color.RGBA {
			if ny < -0.2 {
				return maneatMouth.pick(0.5+ny*0.3, x, y)
			}
			return maneatRamp.pick(lum(nx, ny), x, y)
		})
		for x := int(hx) - 7; x <= int(hx)+7; x += 2 {
			c.set(x, int(hy)-2, hex(0xf4f0e0)) // teeth
			c.set(x, int(hy)+1, hex(0xf4f0e0))
		}
		for _, sp := range [][2]int{{-3, -7}, {2, -8}, {5, 6}} {
			c.set(int(hx)+sp[0], int(hy)+sp[1], hex(0xe84a5a)) // red spots
		}
	}
	c.outline(0.8)
	return mk(c, 16, 33)
}

// --- building ----------------------------------------------------------------

func genDoor() sprite {
	c := newCanvas(24, 30)
	// The frame: two posts and a lintel, like the wall's.
	for _, x := range []int{1, 20} {
		c.rect(x, 3, 3, 24, func(x, y int) color.RGBA { return trunkRamp.pick(0.6-float64(x%3)*0.2, x, y) })
	}
	c.rect(1, 2, 22, 3, func(x, y int) color.RGBA { return trunkRamp.pick(0.7, x, y) })
	// The door: upright planks, a cross-brace, an iron ring.
	for x := 4; x < 20; x++ {
		for y := 5; y < 27; y++ {
			t := 0.55 + float64((x-4)%4)*0.08 - float64(y-5)*0.01
			if (x-4)%4 == 0 {
				t -= 0.25
			}
			c.set(x, y, plankRamp.pick(t, x, y))
		}
	}
	c.line(5, 23, 18, 9, func(x, y int, t float64) color.RGBA { return plankRamp.at(1) })
	roundShape(c, ell{16, 16, 1.4, 1.4}, bandRamp, 0.2)
	c.outline(0.7)
	return mk(c, 12, 26)
}

func genFloor() sprite {
	c := newCanvas(TileW, TileH)
	for y := 0; y < TileH; y++ {
		row := y / 3
		for x := 0; x < TileW; x++ {
			t := 0.5 + (hashf(row, (x+row*7)/9, 91)-0.5)*0.35
			if y%3 == 2 || (x+row*7)%9 == 0 {
				t -= 0.3 // the gaps between planks
			}
			c.set(x, y, plankRamp.pick(t, x, y))
		}
	}
	return mk(c, 0, 0)
}

func genSpikes() sprite {
	c := newCanvas(26, 26)
	c.rect(1, 18, 24, 2, func(x, y int) color.RGBA { return trunkRamp.pick(0.5, x, y) }) // the rail
	for i := range 6 {
		x := 2.5 + float64(i)*4.2
		lean := float64(i%2)*2 - 1
		c.line(x, 24, x+lean*2.5, 6, func(x, y int, t float64) color.RGBA { return plankRamp.pick(0.35+t*0.4, x, y) })
		c.line(x+1, 24, x+1+lean*2.5, 6, func(x, y int, t float64) color.RGBA { return plankRamp.pick(0.2+t*0.3, x, y) })
		c.polygon([]float64{x + lean*2.5 - 0.5, 7, x + lean*2.9 + 0.5, 2, x + lean*2.5 + 1.6, 7}, solid(woodRamp.at(2)))
	}
	c.outline(0.7)
	return mk(c, 13, 24)
}

func genTrap(set bool) sprite {
	c := newCanvas(22, 16)
	rope := func(x, y int, _ float64) color.RGBA { return strawRamp.pick(0.6, x, y) }
	// Two stakes and a bent branch holding the snare.
	c.line(4, 14, 4, 7, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.6, x, y) })
	c.line(18, 14, 18, 7, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.5, x, y) })
	if set {
		c.line(4, 7, 11, 2, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.7, x, y) })
		c.line(11, 2, 18, 7, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.7, x, y) })
		for k := range 24 {
			a := 2 * math.Pi * float64(k) / 24
			c.set(int(math.Round(11+math.Cos(a)*5)), int(math.Round(12+math.Sin(a)*2)), strawRamp.at(3))
		}
		c.line(11, 2, 11, 10, rope)
	} else {
		c.line(4, 7, 18, 7, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.7, x, y) })
		c.line(11, 7, 11, 13, rope)
		roundShape(c, ell{11, 13, 1.5, 0.8}, strawRamp, 0)
	}
	c.outline(0.6)
	return mk(c, 11, 14)
}

func genTorchStick() sprite {
	c := newCanvas(6, 20)
	c.line(3, 19, 3, 5, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.6, x, y) })
	c.line(2, 19, 2, 5, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.35, x, y) })
	c.rect(1, 3, 4, 3, func(x, y int) color.RGBA { return strawRamp.pick(0.4, x, y) }) // the wrapped head
	c.outline(0.6)
	return mk(c, 3, 19)
}

func genTreasure(open bool) sprite {
	c := newCanvas(24, 22)
	const x0, x1, top, base = 2, 22, 10, 20
	for y := top; y <= base; y++ {
		for x := x0; x < x1; x++ {
			t := 0.45 - float64(x-x0)*0.015 + (hashf(x/3, y, 17)-0.5)*0.15
			c.set(x, y, plankRamp.pick(t, x, y))
		}
	}
	for _, bx := range []int{x0 + 3, x1 - 4} {
		c.rect(bx, top, 2, base-top+1, func(x, y int) color.RGBA { return bandRamp.pick(0.5, x, y) })
	}
	if open {
		c.rect(x0, top-6, x1-x0, 5, func(x, y int) color.RGBA { return plankRamp.pick(0.3, x, y) }) // the lid, raised
		c.rect(x0+1, top, x1-x0-2, 2, func(x, y int) color.RGBA { return hex(0x1a0e08) })           // dark inside
	} else {
		c.ellipse(12, top, 10, 3, func(x, y int, nx, ny float64) color.RGBA { return plankRamp.pick(lum(nx, ny), x, y) })
		roundShape(c, ell{12, top + 3, 2, 2}, goldRamp, 0.2) // the lock
		c.set(11, top+3, hex(0x3a2408))
	}
	c.outline(0.8)
	return mk(c, 12, 20)
}

func genHut() sprite {
	c := newCanvas(52, 50)
	// A round wall of upright poles.
	for x := 8; x < 44; x++ {
		for y := 26; y < 46; y++ {
			t := 0.5 + float64((x-8)%3)*0.1 - math.Abs(float64(x)-26)/40
			if (x-8)%3 == 0 {
				t -= 0.2
			}
			c.set(x, y, plankRamp.pick(t, x, y))
		}
	}
	// A doorway hung with hide.
	c.rect(22, 33, 8, 13, func(x, y int) color.RGBA { return hex(0x140c08) })
	c.rect(22, 33, 8, 5, func(x, y int) color.RGBA { return loinRamp.pick(0.5, x, y) })
	// A cone of thatch, ragged at the eaves, with a smoke hole.
	c.polygon([]float64{26, 2, 50, 30, 46, 31, 42, 29, 38, 31, 34, 29, 30, 31, 26, 29, 22, 31, 18, 29, 14, 31, 10, 29, 6, 31, 2, 30}, func(x, y int) color.RGBA {
		t := 0.7 - float64(x)/80 + (hashf(x/2, y, 61)-0.5)*0.3
		if (x+y/3)%4 == 0 {
			t -= 0.2
		}
		return thatchRamp.pick(t, x, y)
	})
	c.rect(24, 2, 4, 3, func(x, y int) color.RGBA { return trunkRamp.at(1) })
	// A skull over the door: these people do not want visitors.
	roundShape(c, ell{26, 30, 2, 1.8}, ramp{colBoneSh, colBone}, 0.2)
	c.set(25, 30, colInk)
	c.set(27, 30, colInk)
	c.outline(0.8)
	return mk(c, 26, 45)
}

func genTotem() sprite {
	c := newCanvas(16, 46)
	for y := 6; y < 45; y++ {
		for x := 4; x < 12; x++ {
			c.set(x, y, trunkRamp.pick(0.7-float64(x-4)*0.08, x, y))
		}
	}
	// Three carved, painted faces stacked up the post.
	for i, fy := range []int{10, 22, 34} {
		paint := []color.RGBA{hex(0xc83020), hex(0xf0e8d0), hex(0x2a6ab0)}[i]
		c.rect(5, fy-1, 2, 2, func(x, y int) color.RGBA { return paint })
		c.rect(9, fy-1, 2, 2, func(x, y int) color.RGBA { return paint })
		c.rect(5, fy+3, 6, 1, func(x, y int) color.RGBA { return hex(0x1a0c08) })
		c.set(7, fy+1, trunkRamp.at(0))
		c.set(8, fy+1, trunkRamp.at(0))
	}
	// Wings at the top.
	c.polygon([]float64{4, 6, 0, 2, 0, 8}, solid(hex(0xc83020)))
	c.polygon([]float64{12, 6, 16, 2, 16, 8}, solid(hex(0xc83020)))
	c.outline(0.8)
	return mk(c, 8, 44)
}

func genBonfire() sprite {
	c := newCanvas(40, 24)
	for i := range 12 {
		a := 2 * math.Pi * float64(i) / 12
		roundShape(c, ell{20 + math.Cos(a)*15, 14 + math.Sin(a)*6.5, 3, 2.4}, stoneRamp, 0)
	}
	for _, l := range [][4]float64{{8, 17, 32, 9}, {8, 9, 32, 17}, {12, 19, 28, 7}} {
		for o := -1.0; o <= 1.0; o++ {
			c.line(l[0], l[1]+o, l[2], l[3]+o, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.6-o*0.2, x, y) })
		}
	}
	c.outline(0.7)
	return mk(c, 20, 14)
}

// --- bodies, heads, weather ----------------------------------------------------

func genBlood(w float64, seed uint32) sprite {
	ww, hh := int(w*2+4), int(w+4)
	c := newCanvas(ww, hh)
	cx, cy := float64(ww)/2, float64(hh)/2
	for y := 0; y < hh; y++ {
		for x := 0; x < ww; x++ {
			dx, dy := (float64(x)+0.5-cx)/w, (float64(y)+0.5-cy)/(w*0.45)
			n := vnoise(float64(x)*0.35, float64(y)*0.5, 300+seed)
			r := math.Hypot(dx, dy) - (n-0.5)*0.7
			if r < 1 {
				col := bloodRamp.pick(0.55-r*0.4+(hashf(x, y, 9)-0.5)*0.2, x, y)
				if r > 0.82 {
					col = bloodRamp.at(0)
				}
				c.set(x, y, col)
			}
		}
	}
	// Splashes about it.
	for k := range 5 {
		a := float64(k)*1.3 + float64(seed)
		c.set(int(cx+math.Cos(a)*w*1.05), int(cy+math.Sin(a)*w*0.5), bloodRamp.at(1))
	}
	return mk(c, cx, cy)
}

func genPost() sprite {
	c := newCanvas(6, 22)
	c.rect(2, 2, 3, 19, func(x, y int) color.RGBA { return trunkRamp.pick(0.7-float64(x-2)*0.2, x, y) })
	c.polygon([]float64{2, 2, 3.5, 0, 5, 2}, solid(woodRamp.at(1)))
	c.outline(0.6)
	return mk(c, 3, 21)
}

func genWebBlob() sprite {
	c := newCanvas(9, 9)
	for k := range 8 {
		a := 2 * math.Pi * float64(k) / 8
		c.line(4.5, 4.5, 4.5+math.Cos(a)*4, 4.5+math.Sin(a)*4, func(x, y int, t float64) color.RGBA {
			return withAlpha(hex(0xf4f4fa), uint8(230-160*t))
		})
	}
	roundShape(c, ell{4.5, 4.5, 1.6, 1.6}, ramp{hex(0xc8c8d4), hex(0xffffff)}, 0.2)
	return mk(c, 4.5, 4.5)
}

func genWebbed() sprite {
	c := newCanvas(20, 30)
	for _, l := range [][4]float64{{1, 4, 19, 22}, {19, 6, 2, 26}, {3, 14, 17, 13}, {6, 2, 13, 28}, {0, 20, 20, 18}} {
		c.line(l[0], l[1], l[2], l[3], func(x, y int, t float64) color.RGBA { return withAlpha(hex(0xf0f0f8), 190) })
	}
	return mk(c, 10, 28)
}

func genTornado(frame int) sprite {
	c := newCanvas(44, 80)
	ph := float64(frame) * math.Pi / 2
	for y := 0; y < 76; y++ {
		t := float64(y) / 76
		w := 3 + 18*t*t + 2*math.Sin(t*9+ph)
		cx := 22 + 4*math.Sin(t*5+ph*0.7)*(1-t)
		for x := int(cx - w); x <= int(cx+w); x++ {
			u := (float64(x) - cx) / w
			band := math.Sin(u*3+t*18-ph*2) * 0.5
			a := 0.85 - 0.45*u*u
			col := mix(hex(0x4a4640), hex(0xb0a898), 0.45+band*0.4-t*0.2)
			c.set(x, 79-y, withAlpha(col, uint8(255*clamp01(a))))
		}
	}
	return mk(c, 22, 78)
}

// --- icons -----------------------------------------------------------------------

func genTribalIcons(icons *[sim.NumItems]*ebiten.Image) {
	icon := func(k sim.ItemKind, draw func(c *canvas)) {
		c := newCanvas(16, 16)
		draw(c)
		c.outline(0.8)
		icons[k] = c.image()
	}
	icon(sim.ItemDinoHide, func(c *canvas) {
		c.polygon([]float64{2, 4, 7, 2, 13, 3, 15, 8, 12, 14, 6, 13, 1, 10}, func(x, y int) color.RGBA {
			col := mkRamp(0x2a4a2a, 0x3e6a38, 0x5a8a48, 0x7aa85a).pick(0.6-float64(y)*0.03, x, y)
			if (x/2+y/2)%2 == 0 {
				col = mix(col, hex(0x1e3420), 0.4) // scales
			}
			return col
		})
	})
	icon(sim.ItemDinoArmor, func(c *canvas) {
		c.polygon([]float64{3, 3, 6, 2, 8, 4, 10, 2, 13, 3, 13, 13, 3, 13}, func(x, y int) color.RGBA {
			col := mkRamp(0x2a4a2a, 0x3e6a38, 0x5a8a48, 0x7aa85a).pick(0.7-float64(x)*0.03, x, y)
			if (x+y)%3 == 0 {
				col = mix(col, hex(0x1e3420), 0.5)
			}
			return col
		})
		c.rect(4, 8, 8, 1, func(x, y int) color.RGBA { return colBone })
	})
	potion := func(liquid ramp, big bool) func(c *canvas) {
		return func(c *canvas) {
			r := 4.6
			if big {
				r = 5.6
			}
			c.rect(7, 1, 3, 4, func(x, y int) color.RGBA { return trunkRamp.at(3) }) // the cork
			roundShape(c, ell{8.5, 15 - r, r, r}, ramp{hex(0x8ab0c0), hex(0xc8e4ee)}, 0.3)
			c.ellipse(8.5, 15.5-r*0.8, r-1.2, r*0.6, func(x, y int, nx, ny float64) color.RGBA { return liquid.pick(lum(nx, ny)+0.2, x, y) })
			c.set(6, int(15-r*1.4), colWhite)
		}
	}
	icon(sim.ItemPotion, potion(maneatMouth, false))
	icon(sim.ItemBigPotion, potion(mkRamp(0x5a0a30, 0x9a1a5a, 0xd8408a, 0xffa0d0), true))
	icon(sim.ItemBandage, func(c *canvas) {
		roundShape(c, ell{8, 9, 6, 4.5}, ramp{hex(0xb8b4b0), hex(0xe8e4e0), hex(0xffffff)}, 0.2)
		roundShape(c, ell{8, 9, 2, 1.5}, ramp{hex(0x908c88), hex(0xb8b4b0)}, 0)
		c.line(2, 12, 14, 6, func(x, y int, t float64) color.RGBA { return hex(0xd8d4d0) })
	})
	icon(sim.ItemSilk, func(c *canvas) {
		for k := range 6 {
			a := float64(k) * math.Pi / 3
			c.line(8, 8, 8+math.Cos(a)*6.5, 8+math.Sin(a)*6.5, func(x, y int, t float64) color.RGBA { return hex(0xf0f0f8) })
		}
		for _, r := range []float64{2.5, 4.5} {
			for k := range 18 {
				a := 2 * math.Pi * float64(k) / 18
				c.set(int(8+math.Cos(a)*r), int(8+math.Sin(a)*r), hex(0xd8d8e4))
			}
		}
	})
	icon(sim.ItemGlowSpore, func(c *canvas) {
		for i, p := range [][3]float64{{5, 9, 2.6}, {10, 6, 3}, {10, 12, 2.2}} {
			roundShape(c, ell{p[0], p[1], p[2], p[2]}, []ramp{glowCyan, glowViolet}[i%2], 0.3)
		}
	})
	for f := range sim.NumFlowers {
		icon(f.Item(), func(c *canvas) {
			c.line(8, 15, 8, 7, func(x, y int, t float64) color.RGBA { return leafRamps[0].pick(0.5, x, y) })
			roundShape(c, ell{5, 12, 2.4, 1.2}, leafRamps[0], 0.2)
			if f == sim.FlowerPurple {
				for k := range 6 {
					roundShape(c, ell{8, 2.5 + float64(k), 1.4, 1}, flowerColors[f], 0.1+float64(k%2)*0.2)
				}
				return
			}
			for k := range 6 {
				a := 2 * math.Pi * float64(k) / 6
				roundShape(c, ell{8 + math.Cos(a)*3, 6 + math.Sin(a)*3, 2, 2}, flowerColors[f], 0.1)
			}
			roundShape(c, ell{8, 6, 1.6, 1.6}, goldRamp, 0)
		})
	}
	seed := func(k sim.ItemKind, rp ramp) {
		icon(k, func(c *canvas) {
			for _, p := range [][2]float64{{5, 9}, {9, 6}, {10, 11}, {7, 13}} {
				roundShape(c, ell{p[0], p[1], 1.9, 1.4}, rp, 0.15)
			}
		})
	}
	seed(sim.ItemFlowerSeed, mkRamp(0x3a2a14, 0x5a4220, 0x7c5c30, 0xa07c44))
	seed(sim.ItemBerrySeed, berryRamp)
	seed(sim.ItemTreeSeed, mkRamp(0x4a2c10, 0x7a4a1c, 0xa86a2c, 0xd09048))
	seed(sim.ItemGlowSeed, glowCyan)
	seed(sim.ItemManEaterSeed, maneatMouth)
}
