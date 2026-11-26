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
	spiderRamp   = mkRamp(0x120c16, 0x21162a, 0x32223e, 0x483454, 0x624a6c)
	spiderMark   = mkRamp(0x7a1010, 0xc0281c, 0xf05a30)
	bearRamp     = mkRamp(0x1c110c, 0x2e1c13, 0x45291b, 0x5e3a25, 0x774d31, 0x936541, 0xb08658)
	bearMuzzle   = mkRamp(0x4a2e1e, 0x6e4a30, 0x957048, 0xb8946a, 0xd4b48c)
	bearClawRamp = mkRamp(0x6a5c50, 0xb0a492, 0xece4d2)
	tribeSkin    = mkRamp(0x100909, 0x1f120f, 0x321e16, 0x48291c, 0x603a27, 0x7c4f36)
	tribeHair    = mkRamp(0x08060a, 0x1c1620)
	spearShaft   = mkRamp(0x4a3018, 0x8a6234, 0xb88a4e)
	potRamp      = mkRamp(0x4a1e12, 0x74301a, 0xa04a26, 0xc46a38, 0xe09058)
	paintRamp    = mkRamp(0xc8c0b0, 0xf2ece0)
	loinRamp     = mkRamp(0x4a2e1a, 0x6c4428, 0x8f5d36, 0xb07a48)
	thatchRamp   = mkRamp(0x5a4420, 0x7d602c, 0xa1803a, 0xc4a04e, 0xe0c46e)
	plankRamp    = mkRamp(0x4a301c, 0x654126, 0x825633, 0x9e6c42, 0xb98552)
	glowCyan     = mkRamp(0x0c3a4a, 0x146a7a, 0x22a8b4, 0x56e2e2, 0xc0fff8)
	glowViolet   = mkRamp(0x2a1048, 0x4a1c7a, 0x7a34b4, 0xb066e8, 0xe8b8ff)
	glowStem     = mkRamp(0x10202a, 0x18323a, 0x234a4a, 0x2e5e58)
	maneatRamp   = mkRamp(0x1c3a14, 0x2a561c, 0x3c7424, 0x55922e, 0x74b03a)
	maneatMouth  = mkRamp(0x5a0a14, 0x8e1424, 0xc42a36, 0xe8586a)
	bloodRamp    = mkRamp(0x3a0408, 0x5e0a10, 0x82141a, 0xa8202a)
	goldRamp     = mkRamp(0x6a4a10, 0xa07a1c, 0xd4aa34, 0xf4d870, 0xfff4c0)
	bandRamp     = mkRamp(0x24262c, 0x3a3e46, 0x565c66, 0x7a828c)
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
	manEater     [manEaterPoses]sprite
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
	for p := range manEaterPoses {
		a.manEater[p] = genManEater(p)
	}
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

func genSpider(pose int) sprite { return critter("spider", 0, pose, 0) }

func spiderCanvas(pose int) (*canvas, float64, float64) {
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
	return c, 12, 14
}

// bearCoat is the colour of a cave bear's shaggy coat at brightness l:
// strands of darker and lighter hair over broad, soft mottling.
func bearCoat(x, y int, l float64) color.RGBA {
	n := hashf(x, (y+x%2)/2, 91)
	if n < 0.2 {
		l -= 0.17
	} else if n > 0.88 {
		l += 0.1
	}
	l += (vnoise(float64(x)/5, float64(y)/5, 92) - 0.5) * 0.16
	return bearRamp.pick(l, x, y)
}

func bearFur(c *canvas, e ell, bias float64) {
	c.ellipse(e.x, e.y, e.rx, e.ry, func(x, y int, nx, ny float64) color.RGBA {
		return bearCoat(x, y, lum(nx, ny)*0.9+bias)
	})
}

func bearTube(c *canvas, bias float64, ctrl ...[3]float64) {
	drawTubeWith(c, spine(ctrl...), func(x, y int, p *spinePt, t float64) color.RGBA {
		return bearCoat(x, y, lum(p.nx*t*0.85, p.ny*t*0.85)+bias)
	})
}

// bearPaw is a broad paw at (x, y), its claws pointing along (dx, dy).
func bearPaw(c *canvas, x, y, dx, dy, bias float64) {
	bearFur(c, ell{x, y, 2.6 + math.Abs(dx)*0.6, 1.7 + math.Abs(dy)*0.9}, bias-0.08)
	claw := bearClawRamp.at(2)
	if bias < 0 {
		claw = bearClawRamp.at(1)
	}
	px, py := -dy, dx
	for k := -1.0; k <= 1; k++ {
		bx, by := x+dx*2.6+px*k*1.5, y+dy*1.8+py*k*1.5
		c.set(int(bx), int(by), claw)
		c.set(int(bx+dx*1.1), int(by+dy*1.1+math.Abs(dx)*0.6), bearClawRamp.at(0))
	}
}

// bearLeg is a thick leg from the hip to a paw flat on the ground at gy.
func bearLeg(c *canvas, hx, hy, fx, gy, r, bias float64) {
	bearTube(c, bias, [3]float64{hx, hy, r}, [3]float64{(hx+fx)/2 - 0.4, (hy + gy) / 2, r * 0.86}, [3]float64{fx, gy - 2.6, r * 0.72})
	bearFur(c, ell{fx + 1, gy - 1.5, r + 0.3, 1.6}, bias-0.1)
	claw := bearClawRamp.at(2)
	if bias < 0 {
		claw = bearClawRamp.at(1)
	}
	for k := 0; k < 3; k++ {
		c.set(int(fx+r+0.8)-k*2, int(gy-0.5), claw)
	}
}

// bearHead draws the head about (hx, hy), turned by a, the jaws open by gape.
func bearHead(c *canvas, hx, hy, a, gape float64) {
	p := func(x, y float64) (float64, float64) { return rot(hx+x, hy+y, hx, hy, a) }
	set := func(x, y float64, col color.RGBA) {
		px, py := p(x, y)
		c.set(int(px), int(py), col)
	}
	ex, ey := p(1.6, -4.2)
	bearFur(c, ell{ex, ey, 1.5, 1.5}, -0.28) // the far ear
	// The lower jaw, hinged under the cheek.
	jx, jy := p(0.5, 2.3)
	qx, qy := rot(jx+6.2, jy+0.5, jx, jy, a+gape)
	if gape > 0 {
		ux, uy := p(7.5, 1.6)
		c.polygon([]float64{jx, jy - 1, ux, uy, qx, qy}, func(x, y int) color.RGBA {
			return maneatMouth.at(1 - (x+y)%2)
		})
	}
	bearTube(c, -0.12, [3]float64{jx, jy, 1.9}, [3]float64{qx, qy, 1.1})
	sx, sy := p(0, 0)
	bearFur(c, ell{sx, sy, 4.7, 4.4}, 0.04) // the skull
	// The muzzle: long, square and paler than the coat.
	m0x, m0y := p(2.5, 0.9)
	m1x, m1y := p(7, 0.7)
	drawTubeWith(c, spine([3]float64{m0x, m0y, 2.5}, [3]float64{m1x, m1y, 1.9}), func(x, y int, sp *spinePt, t float64) color.RGBA {
		return bearMuzzle.pick(lum(sp.nx*t*0.85, sp.ny*t*0.85)+0.05, x, y)
	})
	set(8.4, -0.6, colInk) // the nose
	set(8.4, 0.4, colInk)
	set(7.6, -0.9, colInk)
	if gape > 0 { // fangs
		set(6.6, 2.7, colTooth)
		set(6.6, 3.6, colTooth)
		set(4.6, 2.7, colTooth)
		tx, ty := rot(jx+5.4, jy-1.1, jx, jy, a+gape)
		c.set(int(tx), int(ty), colTooth)
		tx, ty = rot(jx+3.6, jy-1.2, jx, jy, a+gape)
		c.set(int(tx), int(ty), colBoneSh)
	} else {
		for k := 3.0; k <= 7; k++ {
			set(k, 2.6, bearRamp.at(0)) // the line of the mouth
		}
		set(6.2, 3.2, colTooth)
	}
	// A small, mean eye under a heavy brow.
	set(1.6, -2.1, bearRamp.at(0))
	set(2.6, -1.7, bearRamp.at(0))
	set(3.6, -1.2, bearRamp.at(0))
	set(2.6, -0.7, hex(0xf0b030))
	set(1.6, -0.9, colInk)
	ex, ey = p(-1.9, -3.7)
	bearFur(c, ell{ex, ey, 1.8, 1.7}, 0.06) // the near ear
	c.set(int(ex), int(ey+0.3), bearRamp.at(0))
}

// bearShag roughens the silhouette between columns x0 and x1: tufts standing
// off the top of the coat, and hair hanging below it.
func bearShag(c *canvas, x0, x1 int, top, under bool) {
	for x := x0; x <= x1; x++ {
		ty, by := -1, -1
		for y := 0; y < c.h; y++ {
			if c.opaque(x, y) {
				if ty < 0 {
					ty = y
				}
				by = y
			}
		}
		if ty < 0 {
			continue
		}
		if top && hashf(x, 1, 93) < 0.42 {
			c.put(x, ty-1, bearRamp.at(3+x%2))
		}
		if under {
			for k := 1; k <= 1+int(hashf(x, 2, 93)*2.4); k++ {
				c.put(x, by+k, bearRamp.at(1+(x+k)%2))
			}
		}
	}
}

func genBear(pose int) sprite { return critter("bear", 0, pose, 0) }

// bearCanvas draws a cave bear side-on, facing right: a huge, shaggy brown
// beast with a hump over the shoulders, a long pale muzzle and great claws.
func bearCanvas(pose int) (*canvas, float64, float64) {
	c := newCanvas(52, 48)
	c.ty = 4 // headroom for the raised paw
	const gy = 41.0
	if pose == bearClaw {
		// Reared up on its hind legs, roaring, a forepaw high to swipe.
		bearLeg(c, 15, 30, 13, gy, 3.6, -0.22)
		bearTube(c, -0.24, [3]float64{24, 16, 3.2}, [3]float64{29.5, 21.5, 2.8}, [3]float64{33.5, 20.5, 2.4}) // the far arm
		bearPaw(c, 35, 20.5, 0.95, 0.3, -0.24)
		bearTube(c, 0, [3]float64{17, 30, 8.2}, [3]float64{18.5, 22, 8.4}, [3]float64{21, 14.5, 7.4})
		bearFur(c, ell{19.5, 12.5, 6.8, 5.6}, 0.06) // the hump of the shoulders
		// A paler chest and belly.
		c.ellipse(24, 23, 3.4, 7.5, func(x, y int, nx, ny float64) color.RGBA {
			if !c.opaque(x, y) || hashf(x, y, 94) < 0.3+0.5*(nx*nx+ny*ny) {
				return color.RGBA{}
			}
			return bearRamp.at(4)
		})
		bearShag(c, 9, 20, true, false)
		bearLeg(c, 21, 31, 22, gy, 3.8, 0.02)
		bearHead(c, 27.5, 10.5, -0.28, 0.75)
		bearTube(c, 0.06, [3]float64{20, 13, 3.6}, [3]float64{22, 5, 3.1}, [3]float64{27.5, 0.5, 2.6}) // the raised arm
		bearPaw(c, 29.5, 0, 0.55, 0.85, 0.06)
		c.outline(0.85)
		return c, 18, gy
	}
	off := [4]float64{} // far hind, near hind, far fore, near fore
	switch pose {
	case bearWalkA:
		off = [4]float64{2.4, -2.2, -2.4, 2.6}
	case bearWalkB:
		off = [4]float64{-2.4, 2.2, 2.6, -2.2}
	case bearBite:
		off = [4]float64{-1.5, 0.5, 1, 3}
	}
	bearLeg(c, 12, 30, 11+off[0], gy, 3.3, -0.24)
	bearLeg(c, 30, 30, 30+off[2], gy, 3.2, -0.24)
	bearTube(c, -0.1, [3]float64{6.5, 24, 1.6}, [3]float64{4.5, 26, 1.1}) // a stub of a tail
	// The body: heavy haunches, a deep barrel and a hump over the shoulders.
	bearFur(c, ell{13, 26.5, 8, 7.4}, -0.02)
	bearFur(c, ell{21.5, 26.5, 11.5, 7.8}, -0.04)
	bearFur(c, ell{30, 23.5, 8.2, 8}, 0.05)
	hx, hy, ha, gape := 40.5, 24.5, 0.12, 0.0
	if pose == bearBite {
		hx, hy, ha, gape = 42, 27.5, 0.2, 0.8
	}
	bearTube(c, 0.02, [3]float64{33, 23.5, 6.4}, [3]float64{hx - 2, hy, 4.8}) // the thick neck
	bearShag(c, 8, 38, true, false)
	bearShag(c, 14, 28, false, true)
	bearLeg(c, 16, 31, 16+off[1], gy, 3.6, 0.02)
	bearLeg(c, 33, 31, 33+off[3], gy, 3.5, 0.04)
	bearHead(c, hx, hy, ha, gape)
	c.outline(0.85)
	return c, 23, gy
}

// --- tribesman ----------------------------------------------------------------

// tribePose is where the parts of a warrior are: pelvis, chest and head, the
// knee and ankle of each leg, the elbow and hand of each arm, and the two
// ends of the spear.
type tribePose struct {
	pelvis, chest, head [2]float64
	nearLeg, farLeg     [2][2]float64
	nearArm, farArm     [2][2]float64
	butt, tip           [2]float64
}

// tribeFace is the warrior's head in profile, facing right. H and h are hair;
// K, D, d, S, L and l skin from darkest to lightest; W and w paint; E the
// eye, R red paint, M the mouth, T teeth and B bone.
var tribeFace = []string{
	"..hH.hH..",
	".HhHHhHh.",
	"HHHHHHLL.",
	"HHHhSLLLl",
	"HHSSKKKKK",
	"HHSLLKERL",
	".HBWWWLLl",
	".dSLLWSD.",
	".DSLLMTT.",
	"..DSSLLS.",
	"...DdSS..",
}

func genTribesman(pose int) sprite { return critter("tribe", 0, pose, 0) }

// tribesmanCanvas draws a warrior side-on, facing right: black-skinned and
// bare-chested, scowling under white war paint, in a hide loincloth and a
// necklace of bones, a crest of hair with a red feather, a stone-tipped
// spear in his fist.
func tribesmanCanvas(pose int) (*canvas, float64, float64) {
	c := newCanvas(46, 44)
	const fx, gy = 15.0, 41.0
	q := tribePose{
		pelvis:  [2]float64{fx + 0.5, 29},
		chest:   [2]float64{fx + 1, 21.5},
		head:    [2]float64{fx + 2, 14.5},
		nearLeg: [2][2]float64{{fx + 1.8, 35}, {fx + 1.5, 40}},
		farLeg:  [2][2]float64{{fx - 0.6, 35}, {fx - 1.2, 40}},
		nearArm: [2][2]float64{{fx + 2.5, 26.5}, {fx + 8.5, 26.5}},
		farArm:  [2][2]float64{{fx + 2.5, 25.5}, {fx + 4, 28}},
		butt:    [2]float64{fx + 9.5, 40.5},
		tip:     [2]float64{fx + 9.5, 5},
	}
	switch pose {
	case tribeWalkA: // near leg forward, far leg pushing off
		q.nearLeg = [2][2]float64{{fx + 4, 34.5}, {fx + 5, 40}}
		q.farLeg = [2][2]float64{{fx - 1.6, 35}, {fx - 5, 39}}
		q.nearArm = [2][2]float64{{fx + 0.5, 25.5}, {fx + 4.5, 27}}
		q.farArm = [2][2]float64{{fx + 4, 25}, {fx + 6.5, 26.5}}
		q.butt, q.tip = [2]float64{fx - 0.5, 36}, [2]float64{fx + 13.5, 9}
	case tribeWalkB: // far leg forward
		q.pelvis[1], q.chest[1], q.head[1] = 28, 20.5, 13.5
		q.nearLeg = [2][2]float64{{fx + 0.2, 35}, {fx - 3.6, 39.2}}
		q.farLeg = [2][2]float64{{fx + 3.4, 34.5}, {fx + 4.6, 40}}
		q.nearArm = [2][2]float64{{fx + 1.5, 24.5}, {fx + 5.5, 25.5}}
		q.farArm = [2][2]float64{{fx - 0.5, 24.5}, {fx - 1.5, 27.5}}
		q.butt, q.tip = [2]float64{fx + 0.5, 35}, [2]float64{fx + 14.5, 8}
	case tribeThrust: // a deep lunge, the spear driven level with both hands
		q.pelvis = [2]float64{fx, 30}
		q.chest = [2]float64{fx + 4, 23.5}
		q.head = [2]float64{fx + 7.5, 17.5}
		q.nearLeg = [2][2]float64{{fx + 5.5, 34}, {fx + 6.5, 40}}
		q.farLeg = [2][2]float64{{fx - 4, 35.5}, {fx - 8.5, 39.6}}
		q.nearArm = [2][2]float64{{fx + 7.5, 26.6}, {fx + 12, 24.6}}
		q.farArm = [2][2]float64{{fx + 0.5, 26.5}, {fx + 1.5, 24.8}}
		q.butt, q.tip = [2]float64{fx - 7, 25}, [2]float64{fx + 24, 24}
	}
	skin := func(bias float64, ctrl ...[3]float64) { tube(c, spine(ctrl...), tribeSkin, bias) }
	leg := func(hipX float64, l [2][2]float64, bias float64) {
		skin(bias, [3]float64{hipX, q.pelvis[1] + 1, 2.3}, [3]float64{l[0][0], l[0][1], 1.9}, [3]float64{l[1][0], l[1][1] - 1.6, 1.4})
		ax, ay := int(l[1][0]), int(l[1][1])
		for x := ax - 1; x <= ax+3; x++ { // the foot
			c.set(x, ay, tribeSkin.pick(0.5+bias, x, ay))
		}
		c.set(ax+2, ay-1, tribeSkin.pick(0.6+bias, ax+2, ay-1))
		if bias >= 0 {
			for x := ax - 1; x <= ax+1; x++ { // an anklet of white paint
				if c.opaque(x, ay-3) {
					c.set(x, ay-3, paintRamp.at(0))
				}
			}
		}
	}
	shoulder := [2]float64{q.chest[0] - 0.6, q.chest[1] - 1.4}
	arm := func(a [2][2]float64, bias float64) {
		skin(bias, [3]float64{shoulder[0], shoulder[1], 2.2}, [3]float64{a[0][0], a[0][1], 1.6}, [3]float64{a[1][0], a[1][1], 1.3})
		roundShape(c, ell{a[1][0], a[1][1], 1.5, 1.5}, tribeSkin, bias+0.1) // the fist
	}
	leg(q.pelvis[0]-1, q.farLeg, -0.3)
	arm(q.farArm, -0.3)
	leg(q.pelvis[0]+1, q.nearLeg, 0)

	// The spear: an ash shaft, a leaf of knapped stone lashed on with sinew,
	// and a red tassel under the head.
	dx, dy := q.tip[0]-q.butt[0], q.tip[1]-q.butt[1]
	n := math.Hypot(dx, dy)
	dx, dy = dx/n, dy/n
	c.line(q.butt[0], q.butt[1], q.tip[0], q.tip[1], func(x, y int, t float64) color.RGBA {
		return spearShaft.at(1 + (x+y)%2)
	})
	bx, by := q.tip[0]-dx*0.5, q.tip[1]-dy*0.5
	c.polygon([]float64{bx - dy*2.1, by + dx*2.1, bx + dx*2.5 - dy*1.9, by + dy*2.5 + dx*1.9, bx + dx*6.5, by + dy*6.5, bx + dx*2.5 + dy*1.9, by + dy*2.5 - dx*1.9, bx + dy*2.1, by - dx*2.1},
		func(x, y int) color.RGBA { return stoneRamp.pick(0.72-(hashf(x, y, 95))*0.3, x, y) })
	c.line(bx+dx*0.5, by+dy*0.5, bx+dx*5.5, by+dy*5.5, func(x, y int, t float64) color.RGBA { return stoneRamp.at(len(stoneRamp) - 1) })
	for k := 0.5; k <= 2; k += 1.5 { // the lashing
		c.set(int(bx-dx*k), int(by-dy*k), colBone)
	}
	tx, ty := bx-dx*3, by-dy*3
	c.set(int(tx-dy*1.2+0.5), int(ty+math.Abs(dx)*1.2+0.5), hex(0xd8301c))
	c.set(int(tx-dy*1.2-dx+0.5), int(ty+math.Abs(dx)*2.2+1), hex(0x9a1c12))

	// The torso: a deep bare chest over a hard, narrow waist.
	lean := (q.chest[0] - q.pelvis[0]) / (q.pelvis[1] - q.chest[1])
	skin(0.02, [3]float64{q.pelvis[0], q.pelvis[1], 3.1}, [3]float64{lerpf(q.pelvis[0], q.chest[0], 0.5), lerpf(q.pelvis[1], q.chest[1], 0.5), 2.9},
		[3]float64{q.chest[0], q.chest[1], 4.1}, [3]float64{q.chest[0] + lean*2, q.chest[1] - 2.4, 2.8})
	body := func(u, v float64) (int, int) { // a point on the torso: u forward, v down from the chest
		return int(math.Floor(q.chest[0] + u - lean*v)), int(math.Floor(q.chest[1] + v))
	}
	mark := func(u, v float64, col color.RGBA) {
		if x, y := body(u, v); c.opaque(x, y) {
			c.set(x, y, col)
		}
	}
	mark(1, 1.6, tribeSkin.at(0)) // the shadow under the chest
	mark(2, 1.6, tribeSkin.at(0))
	mark(3, 1.2, tribeSkin.at(0))
	mark(2, 4, tribeSkin.at(1)) // the belly's muscles
	mark(1, 6, tribeSkin.at(1))
	// War paint: slashes down the chest and dots on the belly.
	for k := 0.0; k < 2; k++ {
		mark(2+k, -0.6+k, paintRamp.at(1))
		mark(1.6+k, 3+k, paintRamp.at(1))
	}
	// A necklace of bones and teeth.
	for k := -1.0; k <= 4; k++ {
		col := colBone
		if int(k)%2 != 0 {
			col = colBoneSh
		}
		mark(k, -3.2+math.Max(0, 2-math.Abs(k-2.5))*0.5, col)
		if int(k)%2 == 0 {
			mark(k+0.2, -2.2+math.Max(0, 2-math.Abs(k-2.5))*0.5, colBone)
		}
	}
	// A loincloth of hide on a rope belt: a long flap before, a short behind.
	px, py := q.pelvis[0], q.pelvis[1]
	hide := func(x, y int) color.RGBA {
		l := 0.75 - (float64(y)-py)*0.09 + (hashf(x, y, 88)-0.5)*0.25
		return loinRamp.pick(l, x, y)
	}
	c.polygon([]float64{px - 3.6, py - 0.6, px + 0.5, py - 0.6, px - 0.4, py + 4.2, px - 2.2, py + 3.4, px - 4.2, py + 5}, hide)
	c.polygon([]float64{px - 0.5, py - 0.6, px + 4.2, py - 0.6, px + 4.8, py + 5.6, px + 3.2, py + 4.8, px + 1.8, py + 6.4, px + 0.2, py + 5}, hide)
	for x := int(px - 4); x <= int(px+4); x++ {
		if hashf(x, 3, 88) < 0.6 {
			c.set(x, int(py+4.6+hashf(x, 4, 88)*1.5), loinRamp.at(3)) // a trim of pale fur
		}
		c.set(x, int(py-1), thatchRamp.at(1+x%2*2)) // the belt
	}
	c.set(int(px+2), int(py), colBone) // a bone toggle
	c.set(int(px+2), int(py+1), colBoneSh)

	// The head, pixel by pixel: a crest of black hair, a heavy brow drawn
	// down over a glaring eye, paint across the cheek, teeth bared.
	ix, iy := int(math.Floor(q.head[0])), int(math.Floor(q.head[1]))
	skin(0.02, [3]float64{q.chest[0] + lean*2, q.chest[1] - 2, 1.8}, [3]float64{q.head[0] - 0.4, q.head[1] + 3, 1.7}) // the neck
	for r, row := range tribeFace {
		for k, ch := range row {
			var col color.RGBA
			switch ch {
			case 'H':
				col = tribeHair.at(0)
			case 'h':
				col = tribeHair.at(1)
			case 'D':
				col = tribeSkin.at(1)
			case 'd':
				col = tribeSkin.at(2)
			case 'S':
				col = tribeSkin.at(3)
			case 'L':
				col = tribeSkin.at(4)
			case 'l':
				col = tribeSkin.at(5)
			case 'K':
				col = tribeSkin.at(0)
			case 'W':
				col = paintRamp.at(1)
			case 'w':
				col = paintRamp.at(0)
			case 'E':
				col = hex(0xfff6e4)
			case 'R':
				col = hex(0xc8281a)
			case 'T':
				col = colTooth
			case 'M':
				col = hex(0x1a080a)
			case 'B':
				col = colBone
			default:
				continue
			}
			c.set(ix-4+k, iy-5+r, col)
		}
	}
	// A red feather, tied in at the back of the crest.
	hx, hy := float64(ix), float64(iy)
	c.line(hx-3, hy-4, hx-7, hy-10, func(x, y int, t float64) color.RGBA { return mix(hex(0xb81c14), hex(0xf86a30), t) })
	c.line(hx-2, hy-5, hx-6, hy-10, func(x, y int, t float64) color.RGBA { return mix(hex(0xd8301c), hex(0xfce0a0), t*t) })
	c.set(ix-3, iy-4, colBone)

	// The near arm over it all, its fist round the shaft.
	arm(q.nearArm, 0.06)
	ex, ey := lerpf(shoulder[0], q.nearArm[0][0], 0.6), lerpf(shoulder[1], q.nearArm[0][1], 0.6)
	for o := -1; o <= 1; o++ { // a band of paint round the upper arm
		x, y := int(ex)+o, int(ey)
		if a := c.at(x, y); a.A > 0 && (a == tribeSkin.at(2) || a == tribeSkin.at(3) || a == tribeSkin.at(4) || a == tribeSkin.at(5)) {
			c.set(x, y, paintRamp.at(1))
		}
	}
	c.outline(0.85)
	return c, fx, gy
}

func genFishLook(v, frame int) sprite { return critter(fishKind(v%len(fishLooks)), 0, frame, 0) }

func fishLookCanvas(v, frame int) (*canvas, float64, float64) {
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
	return c, 8, 7
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
	c := newEnvCanvas(14, 14)
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
			if envHD {
				for k := range 7 {
					fk := float64(k)
					roundShape(c, ell{h[0] + 0.45*float64(k%2*2-1)*(1-fk/8), h[1] + 1.4 - fk*0.55, 0.62 - fk*0.04, 0.5}, rp, 0.15+float64(k%3)*0.12)
				}
				continue
			}
			for k := range 4 {
				c.set(int(h[0]), int(h[1])-k+1, rp.at(1+k%3))
			}
		default:
			r := 1.7
			if i%2 == 1 {
				r = 1.4
			}
			centre := hex(0x3a2408)
			if f == sim.FlowerWhite || f == sim.FlowerYellow {
				centre = hex(0xe8a818)
			}
			if envHD { // petals round an eye
				bloomHD(c, h[0], h[1], r*1.25, rp[1:], centre, float64(i)*0.5)
				continue
			}
			roundShape(c, ell{h[0], h[1], r, r * 0.9}, rp, 0.1)
			c.set(int(h[0]), int(h[1]), centre)
		}
	}
	c.outline(0.6)
	return mkUp(c, 7, 13)
}

func genFlowerPicked() sprite {
	c := newEnvCanvas(10, 8)
	for i := range 4 {
		x := 2 + float64(i)*2
		c.line(x, 7, x+0.5, 3+float64(i%2), func(x, y int, t float64) color.RGBA { return leafRamps[0].pick(0.45, x, y) })
	}
	return mkUp(c, 5, 7)
}

func genSapling(s sim.SeedKind) sprite {
	c := newEnvCanvas(12, 12)
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
	return mkUp(c, 6, 11)
}

// glowTreeS is how much bigger the glowing tree is drawn than first designed.
const glowTreeS = 1.6

// genGlowPlant draws a plant that glows at night, and a mask of just its
// glowing parts. lit is false for one picked bare, which does not glow.
func genGlowPlant(v int, lit bool) (sprite, sprite) {
	if v == 2 && envHD {
		return glowTreeHD(lit) // see hdglow.go
	}
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
		// Drawn at glowTreeS of that: a tree twice the player's height and more.
		w, h, ax, ay = int(math.Ceil(34*glowTreeS)), int(math.Ceil(52*glowTreeS)), 17*glowTreeS, 51*glowTreeS
		for i := range stems {
			for j := range stems[i] {
				stems[i][j] *= glowTreeS
			}
		}
		for i := range parts {
			p := &parts[i].e
			p.x, p.y, p.rx, p.ry = p.x*glowTreeS, p.y*glowTreeS, p.rx*glowTreeS, p.ry*glowTreeS
		}
	}
	c, m := newEnvCanvas(w, h), newEnvCanvas(w, h)
	if v == 2 && envHD {
		c.lw = 6 // the tree's trunk and boughs
	}
	for _, s := range stems {
		thick := 0.0
		if v == 2 && s[1] > 20*glowTreeS {
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
	return mkUp(c, ax, ay), mkUp(m, ax, ay)
}

// genManEater draws a man-eating plant: a thick stalk rising from broad
// leaves to a great pair of toothed jaws, open and waiting, or snapped
// shut and lunging forward.
// The man-eating plant's poses.
const (
	manEaterWait = iota // jaws open, waiting
	manEaterSnap        // lunging, jaws snapping shut
	manEaterChew        // jaws shut on something it caught, swollen, chewing
	manEaterPoses
)

// The man-eater's own colours: the red of its jaws outside, the wet flesh
// inside them, its fangs, and what is left of its meals.
var (
	maneatHead  = mkRamp(0x30050c, 0x520812, 0x780c1a, 0x9c1220, 0xbc1a26, 0xd62c30, 0xec4c48)
	maneatFlesh = mkRamp(0x4a0818, 0x7a122a, 0xb02c44, 0xdc566a, 0xf48a96, 0xffbcbc)
	maneatFang  = mkRamp(0x8a7a5a, 0xcfc09a, 0xf2e8c8, 0xfffbe8)
	maneatPrey  = mkRamp(0x33260f, 0x5e4a1e, 0x8c7430, 0xb89c4a, 0xdcc478)
	maneatBone  = mkRamp(0x7a705c, 0xb8ae94, 0xe6dec6, 0xfaf6e6)
	maneatRoot  = mkRamp(0x22180e, 0x3a2a16, 0x554020, 0x70582e)
)

// manEaterTube fills a tapering tube along a cubic Bézier (four points in
// bz), r0 thick at its start and r1 at its end. shade colours a pixel from
// how far along the tube it is, in pixels, and its normal across it.
func manEaterTube(c *canvas, bz [8]float64, r0, r1 float64, shade func(x, y int, s, nx, ny float64) color.RGBA) {
	const n = 72
	var px, py, ps [n + 1]float64
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		u := 1 - t
		px[i] = u*u*u*bz[0] + 3*u*u*t*bz[2] + 3*u*t*t*bz[4] + t*t*t*bz[6]
		py[i] = u*u*u*bz[1] + 3*u*u*t*bz[3] + 3*u*t*t*bz[5] + t*t*t*bz[7]
		if i > 0 {
			ps[i] = ps[i-1] + math.Hypot(px[i]-px[i-1], py[i]-py[i-1])
		}
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			best, bi := math.Inf(1), 0
			for i := 0; i <= n; i++ {
				if d := (fx-px[i])*(fx-px[i]) + (fy-py[i])*(fy-py[i]); d < best {
					best, bi = d, i
				}
			}
			r := lerpf(r0, r1, float64(bi)/n)
			if best > r*r {
				continue
			}
			if col := shade(x, y, ps[bi], (fx-px[bi])/r, (fy-py[bi])/r); col.A > 0 {
				c.put(x, y, col)
			}
		}
	}
}

// manEaterLeaf draws one big serrated leaf from its base to its tip, w wide
// at its broadest and bowed sideways by bend.
func manEaterLeaf(c *canvas, bx, by, tx, ty, w, bend, bias float64) {
	l := math.Hypot(tx-bx, ty-by)
	ux, uy := (tx-bx)/l, (ty-by)/l
	nx, ny := -uy, ux
	lit := -(nx*0.55 + ny*0.7) // how much the +d side faces the light
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x)+0.5-bx, float64(y)+0.5-by
			s := (fx*ux + fy*uy) / l
			if s < 0 || s > 1 {
				continue
			}
			d := fx*nx + fy*ny - bend*math.Sin(math.Pi*s)
			saw := s*7 - math.Floor(s*7)
			half := w * math.Pow(math.Sin(math.Pi*math.Pow(s, 0.7)), 0.8) * (1 - 0.28*saw)
			if math.Abs(d) > half || half < 0.3 {
				continue
			}
			t := 0.5 + bias + 0.3*lit*d/w + 0.12*s
			switch {
			case math.Abs(d) < 0.55:
				t -= 0.3 // the midrib
			case saw < 0.16 && math.Abs(d) < half-0.6:
				t -= 0.16 // side veins
			case math.Abs(d) > half-1:
				t += 0.1
			}
			c.put(x, y, maneatRamp.pick(t, x, y))
		}
	}
}

// manEaterFang draws one fang from its root, n pixels long along (dx, dy)
// and two thick at the root along (wx, wy).
func manEaterFang(c *canvas, x, y, dx, dy, wx, wy, n, shadeBias float64) {
	for k := 0.0; k < n; k += 0.5 {
		i := 1 + int(k/n*3+shadeBias)
		px, py := x+dx*k, y+dy*k
		c.put(int(math.Floor(px)), int(math.Floor(py)), maneatFang.at(i))
		if k < n*0.5 {
			c.put(int(math.Floor(px+wx)), int(math.Floor(py+wy)), maneatFang.at(i-1))
		}
	}
}

// manEaterHead draws the jaws: two red lobes hinged at (hx, hy), pointing
// along ang and gaping by gape radians, each l long and t deep. swell fattens
// them around a meal.
func manEaterHead(c *canvas, hx, hy, ang, gape, l, t, swell float64) {
	open := gape > 0.6
	type lobe struct{ ux, uy, vx, vy, w float64 }
	aU, aL := ang-gape/2, ang+gape/2
	up := lobe{math.Cos(aU), math.Sin(aU), math.Sin(aU), -math.Cos(aU), 0}
	lo := lobe{math.Cos(aL), math.Sin(aL), -math.Sin(aL), math.Cos(aL), 0}
	if open {
		up.w, lo.w = l*0.2, l*0.34
	}
	prof := func(s float64) float64 {
		return math.Pow(math.Sin(math.Pi*math.Pow(s, 0.8)), 0.6-0.22*swell) * (1 + swell*0.06*math.Sin(s*10+1))
	}
	lens := func(s float64) float64 { return math.Pow(math.Sin(math.Pi*s), 0.8) }
	each := func(f func(x, y int, dx, dy float64)) {
		r := int(l + t + 8)
		for y := int(hy) - r; y <= int(hy)+r; y++ {
			for x := int(hx) - r; x <= int(hx)+r; x++ {
				f(x, y, float64(x)+0.5-hx, float64(y)+0.5-hy)
			}
		}
	}
	if open {
		// The throat, dark between the lobes.
		each(func(x, y int, dx, dy float64) {
			r := math.Hypot(dx, dy)
			d := math.Atan2(dy, dx) - ang
			if r < l*0.62 && math.Abs(d) < gape/2 {
				c.put(x, y, maneatFlesh.pick(0.25*r/(l*0.62)-0.05, x, y))
			}
		})
		// The far row of fangs, behind the lower lobe's inner face.
		for s := 0.2; s < 0.95; s += 4.2 / l {
			o := lo.w * lens(s)
			manEaterFang(c, hx+lo.ux*s*l-lo.vx*o, hy+lo.uy*s*l-lo.vy*o,
				-lo.vx+lo.ux*0.3, -lo.vy+lo.uy*0.3, lo.ux, lo.uy, 3+2*math.Sin(math.Pi*s), -1)
		}
	}
	for li, b := range []lobe{lo, up} {
		each(func(x, y int, dx, dy float64) {
			u, v := dx*b.ux+dy*b.uy, dx*b.vx+dy*b.vy
			s := u / l
			if s <= 0 || s >= 1 {
				return
			}
			p := t * prof(s)
			switch {
			case v >= 0 && v <= p: // the red outside
				nv := v / p
				nx := b.vx*nv*0.92 + b.ux*(2*s-1)*0.4
				ny := b.vy*nv*0.92 + b.uy*(2*s-1)*0.4
				k := lum(nx, ny)*0.85 + 0.06
				for _, rib := range []float64{0.36, 0.7} {
					if math.Abs(nv-rib)*p < 0.55 && s > 0.08 {
						k -= 0.2 // veins, running hinge to tip
					}
				}
				if v < 1.2 {
					k += 0.14 // the lip
				}
				col := maneatHead.pick(k, x, y)
				// Yellow-green spots.
				cu, cv := math.Floor(u/6), math.Floor(v/5)
				if h := hashf(int(cu)+li*31, int(cv), 77); h > 0.72 && v > 2 && v < p-1.5 {
					ox := (cu+0.3+0.4*hashf(int(cu), int(cv)+li*17, 78))*6 - u
					oy := (cv+0.3+0.4*hashf(int(cu)+li*13, int(cv), 79))*5 - v
					if ox*ox+oy*oy < 1.9 {
						col = mix(hex(0xd8d456), maneatHead.at(3), 0.5-k*0.5)
					}
				}
				c.put(x, y, col)
			case v < 0 && -v <= b.w*lens(s): // the wet inside
				q := -v / (b.w * lens(s))
				k := 0.42 + 0.4*s - 0.3*q
				if li == 1 {
					k = 0.34 - 0.26*q // the upper lobe's is in its own shade
				} else if math.Abs(q-0.5) < 0.09 {
					k -= 0.22
				}
				c.put(x, y, maneatFlesh.pick(k, x, y))
			}
		})
	}
	// The fangs along both rims: long and bare when open, meshed when shut.
	step := 4.4 / l
	if open {
		step = 5.2 / l
	}
	for i, b := range []lobe{lo, up} {
		s0 := 0.2
		if open {
			s0 = 0.36
		}
		for s := s0 + float64(i)*step/2; s < 0.98; s += step {
			n := 5.5
			if open {
				n = 5 + 3*math.Sin(math.Pi*s)
			}
			manEaterFang(c, hx+b.ux*s*l, hy+b.uy*s*l, -b.vx*0.95+b.ux*0.3, -b.vy*0.95+b.uy*0.3, b.ux, b.uy, n, 0)
		}
	}
}

// manEaterCanvas draws the plant and returns it with its anchor, the middle
// of its foot.
func manEaterCanvas(pose int) (*canvas, float64, float64) {
	const ax, ay = 24.0, 77.0
	c := newCanvas(72, 84)
	// The stalk's curve from foot to neck, and the head on it.
	var (
		stalk               [8]float64
		ang, gape, l, t, sw float64
	)
	switch pose {
	case manEaterSnap:
		stalk = [8]float64{ax, 75, 2, 62, 16, 30, 37, 42}
		ang, gape, l, t = 0.36, 0.14, 32, 11.5
	case manEaterChew:
		stalk = [8]float64{ax, 75, 42, 62, 4, 52, 23, 36}
		ang, gape, l, t, sw = -0.1, 0, 30, 15, 1
	default:
		stalk = [8]float64{ax, 75, 44, 60, 6, 50, 26, 33}
		ang, gape, l, t = -0.46, 1.24, 31, 12
	}
	hx, hy := stalk[6], stalk[7]

	// Roots gripping the ground, and the bones of old meals.
	for i, r := range [][8]float64{
		{ax, 75, 16, 73, 10, 79, 3, 78},
		{ax, 75, 32, 73, 38, 80, 46, 78},
		{ax, 76, 22, 79, 27, 80, 30, 82.5},
	} {
		manEaterTube(c, r, 2.1, 0.8, func(x, y int, s, nx, ny float64) color.RGBA {
			return maneatRoot.pick(lum(nx, ny)*0.9+0.1-0.2*hashf(x/2, y+i, 5), x, y)
		})
	}
	roundShape(c, ell{50.5, 78.5, 3.4, 2.8}, maneatBone, 0.15) // a small skull
	c.rect(52, 80, 4, 2, func(x, y int) color.RGBA { return maneatBone.pick(0.5, x, y) })
	c.set(49, 78, maneatBone.at(0))
	c.set(52, 78, maneatBone.at(0))
	for x := 53; x < 56; x += 2 {
		c.set(x, 81, maneatBone.at(0))
	}
	c.line(5, 81.5, 14, 80, func(x, y int, _ float64) color.RGBA { return maneatBone.at(2) })
	for _, e := range [][2]int{{4, 80}, {4, 82}, {14, 79}, {15, 81}} {
		c.set(e[0], e[1], maneatBone.at(3))
	}
	for i := 0; i < 3; i++ { // ribs
		c.line(float64(58+i*3), 79.5, float64(60+i*3), 75.5, func(x, y int, _ float64) color.RGBA { return maneatBone.at(1 + i%2) })
	}

	// The rosette of leaves behind the stalk.
	for _, f := range [][5]float64{
		{-22, -17, 6, 3, -0.08},
		{20, -19, 6, -3, -0.1},
		{-23, -4, 6.5, 3, 0},
		{27, -5, 6.5, -3.5, 0.02},
		{-9, -24, 5, 2, -0.14},
		{9, -25, 5, -2, -0.16},
	} {
		manEaterLeaf(c, ax, 74, ax+f[0], 74+f[1], f[2], f[3], f[4])
	}

	// The stalk: thick, ribbed, thorny.
	manEaterTube(c, stalk, 5, 3.6, func(x, y int, s, nx, ny float64) color.RGBA {
		k := lum(nx, ny)*0.85 + 0.1
		if math.Mod(s, 4.5) < 1.1 {
			k -= 0.22
		}
		return maneatRamp.pick(k, x, y)
	})
	for i, tt := range []float64{0.14, 0.3, 0.46, 0.62, 0.78} {
		u := 1 - tt
		bx := u*u*u*stalk[0] + 3*u*u*tt*stalk[2] + 3*u*tt*tt*stalk[4] + tt*tt*tt*stalk[6]
		by := u*u*u*stalk[1] + 3*u*u*tt*stalk[3] + 3*u*tt*tt*stalk[5] + tt*tt*tt*stalk[7]
		dx := 3*u*u*(stalk[2]-stalk[0]) + 6*u*tt*(stalk[4]-stalk[2]) + 3*tt*tt*(stalk[6]-stalk[4])
		dy := 3*u*u*(stalk[3]-stalk[1]) + 6*u*tt*(stalk[5]-stalk[3]) + 3*tt*tt*(stalk[7]-stalk[5])
		d := math.Hypot(dx, dy)
		side := float64(1 - 2*(i%2))
		nx, ny := -dy/d*side, dx/d*side
		for k := 4.5; k < 7.5; k += 0.5 { // a thorn
			c.put(int(bx+nx*k-dx/d*(k-4.5)*0.5), int(by+ny*k-dy/d*(k-4.5)*0.5), maneatRamp.at(4-int(k-4.5)/2))
		}
	}
	// Low leaves in front of the foot.
	manEaterLeaf(c, ax, 75, ax-9, 81, 4.5, 1.5, 0.1)
	manEaterLeaf(c, ax, 75, ax+11, 80.5, 4.5, -1.5, 0.06)

	// The collar of sepals where the head sits on the stalk.
	roundShape(c, ell{hx, hy, 5.6, 5.6}, maneatRamp, -0.05)
	ux, uy := math.Cos(ang), math.Sin(ang)
	tipX, tipY := hx+ux*l, hy+uy*l
	if pose == manEaterChew {
		// A tail still hanging out of the front of the jaws.
		manEaterTube(c, [8]float64{tipX - ux*6, tipY - uy*6, tipX + 6, tipY + 1, tipX + 11, tipY + 6, tipX + 9, tipY + 17}, 3.3, 0.8,
			func(x, y int, s, nx, ny float64) color.RGBA {
				k := lum(nx, ny)*0.8 + 0.25
				if math.Mod(s, 5) < 1.6 && ny < 0.3 {
					k -= 0.4
				}
				return maneatPrey.pick(k, x, y)
			})
	}
	manEaterHead(c, hx, hy, ang, gape, l, t, sw)
	for _, s := range []float64{-2.2, 0, 2.2} { // sepals lapping the lobes
		a := ang + s*0.42
		manEaterLeaf(c, hx-ux*3, hy-uy*3, hx+math.Cos(a)*8.5, hy+math.Sin(a)*8.5, 2.6, 0, -0.02+0.05*math.Abs(s))
	}
	if pose == manEaterChew {
		// A clawed foot caught between the fangs.
		fx, fy := hx+ux*l*0.62-uy*1, hy+uy*l*0.62+ux*1
		c.line(fx, fy, fx+2, fy+6, func(x, y int, _ float64) color.RGBA { return maneatPrey.at(3) })
		c.line(fx+1, fy, fx+3, fy+6, func(x, y int, _ float64) color.RGBA { return maneatPrey.at(1) })
		for i := 0.0; i < 3; i++ {
			c.line(fx+2, fy+6, fx+0.5+i*2.2, fy+9.5-math.Abs(i-1), func(x, y int, tt float64) color.RGBA {
				if tt > 0.7 {
					return maneatFang.at(1)
				}
				return maneatPrey.at(2)
			})
		}
	}
	c.outline(0.8)

	// Sticky spit, drawn over the outline so it stays thin.
	spit := func(x, y float64, n int) {
		for i := 0; i <= n; i++ {
			col := color.RGBA{0xd8, 0xf0, 0xc0, 215}
			if i == n {
				col = color.RGBA{0xf4, 0xff, 0xe8, 255}
				c.put(int(x)+1, int(y)+i, color.RGBA{0xb8, 0xdc, 0xa8, 230})
			}
			c.put(int(x), int(y)+i, col)
		}
	}
	switch pose {
	case manEaterWait:
		aL := ang + gape/2
		spit(hx+math.Cos(aL)*l*0.93, hy+math.Sin(aL)*l*0.93+3, 8)
	case manEaterSnap:
		spit(tipX-1, tipY+3, 4)
		for _, d := range [][2]float64{{4, -5}, {6, 1}, {2, 8}} { // flecks flung by the snap
			c.put(int(tipX+d[0]), int(tipY+d[1]), color.RGBA{0xe8, 0xfc, 0xd8, 255})
		}
	default:
		spit(tipX-4, tipY+4, 5)
	}
	return c, ax, ay
}

func genManEater(pose int) sprite {
	c, ax, ay := manEaterCanvas(pose)
	return mkUp(c, ax, ay)
}

// --- building ----------------------------------------------------------------

func genDoor() sprite {
	if envHD {
		return genDoorHD()
	}
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
	return mkUp(c, 12, 26)
}

func genFloor() sprite {
	if envHD {
		return genFloorHD()
	}
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
	return mkUp(c, 0, 0)
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
	return mkUp(c, 13, 24)
}

func genTrap(set bool) sprite {
	c := newEnvCanvas(22, 16)
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
	return mkUp(c, 11, 14)
}

func genTorchStick() sprite {
	if envHD {
		return torchStickHD()
	}
	c := newEnvCanvas(6, 30) // a torch stands to the player's shoulder
	c.line(3, 29, 3, 5, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.6, x, y) })
	c.line(2, 29, 2, 5, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.35, x, y) })
	c.rect(1, 3, 4, 3, func(x, y int) color.RGBA { return strawRamp.pick(0.4, x, y) }) // the wrapped head
	c.outline(0.6)
	return mkUp(c, 3, 29)
}

func genTreasure(open bool) sprite {
	if envHD {
		return mkUp(chestHD(open, plankRamp, plankRamp, bandRamp, goldRamp, 17), 12*HDDensity, 20*HDDensity)
	}
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
	return mkUp(c, 12, 20)
}

func genHut() sprite {
	if envHD {
		return mkUp(genHutHD()) // see hdhut.go
	}
	c, ax, ay := hutCanvas()
	return mkUp(c, ax, ay)
}

// hutCanvas draws a hut of the tribe: a round wall of lashed poles under a
// shaggy cone of thatch, a skull over its hide-hung doorway.
func hutCanvas() (*canvas, float64, float64) {
	c := newCanvas(62, 62)
	const (
		cx, gy   = 31.0, 57.0
		wallR    = 21.0
		apexY    = 6.0
		roofH    = 23.0
		roofW    = 28.0
		roofSag  = 4.5
		wallTop  = 28
		doorL    = 26
		doorR    = 35
		doorTop  = 39
		lintelY  = 37
		tiers    = 4.0
		doorMidX = 30
	)
	eave := func(x int, s float64) float64 { // the front edge of the roof at scale s
		d := (float64(x) + 0.5 - cx) / (roofW * s)
		return apexY + s*roofH + s*roofSag*math.Sqrt(math.Max(0, 1-d*d))
	}
	foot := func(x int) float64 { // where the wall meets the ground
		u := (float64(x) + 0.5 - cx) / wallR
		return gy - 3.5 + 3.5*math.Sqrt(math.Max(0, 1-u*u))
	}
	// The wall: upright poles, each a little different, round the curve.
	x0 := int(cx - wallR)
	for x := x0; x < int(cx+wallR); x++ {
		u := (float64(x) + 0.5 - cx) / wallR
		pole, in := (x-x0)/3, (x-x0)%3
		bot := int(foot(x) + hashf(pole, 0, 61)*1.4)
		for y := wallTop; y < bot; y++ {
			l := 0.42 - 0.3*u + 0.3*math.Sqrt(1-u*u) + (hashf(pole, 1, 61)-0.5)*0.16
			switch in {
			case 0:
				l += 0.1
			case 2:
				l -= 0.24
			}
			if hashf(x, y/3, 62) < 0.1 {
				l -= 0.12 // grain and knots
			}
			if d := float64(y) - eave(x, 1); d < 5 {
				l -= 0.3 - d*0.05 // in the shade of the eaves
			}
			if float64(y) > float64(bot)-2.5 {
				l -= 0.12 // damp at the foot
			}
			c.set(x, y, plankRamp.pick(l, x, y))
		}
	}
	// Two bands of rope lash the poles together.
	for _, h := range []float64{9, 17} {
		for x := x0; x < int(cx+wallR); x++ {
			y := int(foot(x) - h)
			if c.opaque(x, y) {
				u := (float64(x) + 0.5 - cx) / wallR
				c.set(x, y, thatchRamp.pick(0.75-0.35*u-float64((x-x0)%3)*0.12, x, y))
				c.set(x, y+1, plankRamp.at(0))
			}
		}
	}
	// The doorway: dark within, framed with stout posts and a lintel log, a
	// hide curtain drawn half aside.
	for x := doorL; x <= doorR; x++ {
		for y := doorTop; y < int(foot(x)); y++ {
			c.set(x, y, mix(hex(0x0c0708), hex(0x2a1a12), float64(y-doorTop)/26+hashf(x, y, 63)*0.1))
		}
	}
	c.polygon([]float64{doorL, doorTop, doorMidX + 3, doorTop, doorMidX + 1, doorTop + 6, doorMidX - 1.5, doorTop + 11, doorL + 2, doorTop + 15, doorL, doorTop + 14},
		func(x, y int) color.RGBA {
			l := 0.62 - float64(x-doorL)*0.06 + (hashf(x/2, y/2, 64)-0.5)*0.22
			if (x-doorL)%3 == 2 {
				l -= 0.15 // folds
			}
			return loinRamp.pick(l, x, y)
		})
	for _, p := range [][2]int{{1, 3}, {2, 3}, {3, 3}, {2, 4}, {2, 5}, {1, 7}, {3, 7}, {2, 8}} { // a mark painted on the hide
		c.set(doorL+p[0], doorTop+p[1], paintRamp.at(p[1]%2))
	}
	post := func(x int) {
		for y := lintelY; y < int(foot(x))+1; y++ {
			c.set(x, y, trunkRamp.pick(0.72, x, y))
			c.set(x+1, y, trunkRamp.pick(0.38, x, y))
		}
	}
	post(doorL - 2)
	post(doorR + 1)
	for x := doorL - 4; x <= doorR+4; x++ {
		c.set(x, lintelY, trunkRamp.pick(0.8-float64(x-doorL)*0.02, x, lintelY))
		c.set(x, lintelY+1, trunkRamp.pick(0.35, x, lintelY+1))
	}
	c.set(doorL-4, lintelY, trunkRamp.at(len(trunkRamp)-1)) // the cut ends of the lintel
	c.set(doorR+4, lintelY+1, trunkRamp.at(len(trunkRamp)-2))
	// Stones packed round the foot of the wall.
	for i, sx := range []int{12, 16, 21, 39, 44, 49} {
		roundShape(c, ell{float64(sx), foot(sx) - 0.6, 1.9 + float64(i%2)*0.6, 1.5}, stoneRamp, 0.05-float64(i)*0.03)
	}
	// A hide shield hung on the wall, and a spear leant by the door.
	c.ellipse(17, 45, 4, 5.2, func(x, y int, nx, ny float64) color.RGBA {
		if nx*nx+ny*ny > 0.72 {
			return loinRamp.pick(0.3, x, y)
		}
		d := nx*nx + ny*ny
		if d < 0.07 {
			return colBone // the boss
		}
		if (x+y)%2 == 0 && d > 0.3 && d < 0.55 {
			return paintRamp.at(1) // a ring of dots
		}
		if d < 0.3 {
			return hex(0x9a2a18)
		}
		return loinRamp.pick(lum(nx, ny)+0.15, x, y)
	})
	c.line(45.5, gy-1.5, 43.5, 33, func(x, y int, t float64) color.RGBA { return spearShaft.at(1 + y%2) })
	// A clay pot by the door.
	roundShape(c, ell{40.5, gy - 3.4, 2.7, 2.9}, potRamp, 0.05)
	c.rect(39, int(gy-7), 4, 1, func(x, y int) color.RGBA { return potRamp.at(3) })
	c.set(40, int(gy-4), paintRamp.at(1))
	c.set(41, int(gy-3), paintRamp.at(1))

	// The roof: a cone of thatch laid in ragged tiers, each casting a shadow
	// on the one below.
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			d := math.Abs(float64(x)+0.5-cx) / roofW
			a := (float64(x) + 0.5 - cx) / roofW
			lo, hi := math.Max(d, 0.02), 1.0
			fy := float64(y) + 0.5
			if fy < eave(x, lo) {
				continue
			}
			var s float64
			if over := fy - eave(x, hi); over > 0 { // the ragged eaves
				strand := int(math.Floor(a * 30))
				if d >= 1 || over > 0.4+hashf(strand, 9, 65)*2.6 {
					continue
				}
				s = 1
			} else {
				for range 14 {
					mid := (lo + hi) / 2
					if eave(x, mid) < fy {
						lo = mid
					} else {
						hi = mid
					}
				}
				s = (lo + hi) / 2
			}
			ang := a / s // -1 at the left edge of the cone, 1 at the right
			tf := s*tiers + (hashf(int(math.Floor(ang*(5+s*14))), 7, 65)-0.5)*0.24
			if s >= 1 {
				tf = tiers - 0.02
			}
			tier := math.Floor(math.Max(0, tf))
			f := tf - tier
			strand := int(math.Floor(ang * (5 + (tier+1)*5)))
			l := 0.6 - ang*0.3 + (hashf(strand, int(tier), 66)-0.5)*0.2 + (f-0.5)*0.34
			if hashf(x, y, 67) < 0.08 {
				l += 0.14 // stray bright straws
			}
			if tier > 0 {
				if f < 0.14 {
					l -= 0.45
				} else if f < 0.3 {
					l -= 0.18
				}
			}
			c.set(x, y, thatchRamp.pick(l, x, y))
		}
	}
	// The roof poles cross and stand out at the peak, bound with rope, the
	// smoke hole dark between them.
	for i, p := range [][2]float64{{-4, 0.5}, {4.5, 1}, {0.5, 0}} {
		c.line(cx+p[0]*0.2, apexY+3, cx+p[0], p[1]+1, func(x, y int, t float64) color.RGBA { return trunkRamp.pick(0.45+float64(i)*0.15, x, y) })
	}
	c.set(int(cx)-1, int(apexY)+2, hex(0x140c0a))
	c.set(int(cx), int(apexY)+2, hex(0x140c0a))
	for x := int(cx) - 2; x <= int(cx)+2; x++ {
		c.set(x, int(apexY)+3, thatchRamp.at(4-(x&1)*2))
	}
	// Teeth and bones strung along the eaves.
	for i, x := range []int{9, 14, 19, 24, 38, 43, 48, 53} {
		y := int(eave(x, 1)) + 1
		c.set(x, y, colBone)
		c.set(x, y+1, colBone)
		if i%2 == 0 {
			c.set(x, y+2, colBoneSh)
		} else {
			c.set(x, y+2, hex(0xc03020)) // a red feather
			c.set(x, y+3, hex(0x8a1c14))
		}
	}
	// A horned skull over the door: these people do not want visitors.
	sy := float64(lintelY) - 3.2
	c.line(cx-2.5, sy-0.5, cx-6, sy-3.5, func(x, y int, t float64) color.RGBA { return mix(colBoneSh, colBone, t) })
	c.line(cx+1.5, sy-0.5, cx+5, sy-3.5, func(x, y int, t float64) color.RGBA { return mix(colBoneSh, colBone, t) })
	roundShape(c, ell{cx - 0.5, sy, 3, 2.6}, ramp{colBoneSh, colBone, hex(0xfffaf0)}, 0.25)
	c.rect(int(cx)-2, int(sy)+2, 3, 2, func(x, y int) color.RGBA { return colBone })
	c.set(int(cx)-2, int(sy), colInk)
	c.set(int(cx), int(sy), colInk)
	c.set(int(cx)-1, int(sy)+2, colBoneSh)
	c.set(int(cx)-2, int(sy)+3, colInk)
	c.set(int(cx), int(sy)+3, colInk)
	c.outline(0.85)
	return c, cx, gy - 1
}

func genTotem() sprite {
	if envHD {
		return genTotemHD()
	}
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
	return mkUp(c, 8, 44)
}

func genBonfire() sprite {
	if envHD {
		return genBonfireHD()
	}
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
	return mkUp(c, 20, 14)
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
	if envHD {
		return genPostHD()
	}
	c := newCanvas(6, 22)
	c.rect(2, 2, 3, 19, func(x, y int) color.RGBA { return trunkRamp.pick(0.7-float64(x-2)*0.2, x, y) })
	c.polygon([]float64{2, 2, 3.5, 0, 5, 2}, solid(woodRamp.at(1)))
	c.outline(0.6)
	return mkUp(c, 3, 21)
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
