package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The mammoth king, drawn at HDDensity with the tools of the furred beasts
// (critterhd.go, critbeast.go): a mammoth more than half as big again as
// those of its herd and far older. A great hump of shoulder with the head
// hung low before it; dark olive fur grown over with moss, which hangs from
// its belly, its chin and its tusks in long strands; a ridge of knobs of
// bone, like weathered rock, along the spine; tusks that sweep forward and
// up, stained and cracked; feet with nails like claws. Every pose is one
// kingPose: a walk, a charge, the swing of the tusks, the rearing stomp and
// the trumpeting are runs of them (bhAnim), timed from when the state began.

var (
	kingFur   = mkRamp(0x0e0c08, 0x1a170d, 0x282413, 0x3a341a, 0x4e4623, 0x645a2e, 0x7c713c, 0x978b50)
	kingMoss  = mkRamp(0x10190b, 0x1a2a10, 0x284016, 0x39581d, 0x4e7226, 0x678c33, 0x83a444)
	kingSkin  = mkRamp(0x15120f, 0x242019, 0x363026, 0x4a4335, 0x5f5746, 0x776e5a)
	kingIvory = mkRamp(0x40382a, 0x6a5d44, 0x938363, 0xb8a884, 0xd4c7a4, 0xe9e0c4)
	kingStone = mkRamp(0x1f1b15, 0x353026, 0x4f483a, 0x6b6250, 0x897f69, 0xa59b83)
)

// kingPose is how the mammoth king stands.
type kingPose struct {
	g     gait
	drop  float64    // the body sunk on its legs, art pixels
	lunge float64    // and thrown forward
	rear  float64    // reared up about its hind legs: 1 right up
	headA float64    // the head turned about the neck: down is positive
	trunk [6]float64 // the way the trunk points along it: see trunkHang
	swing float64    // the hair, the moss and the trunk swung to one side: -1 to 1
}

func kingStand() kingPose {
	return kingPose{g: gait{stride: 15, lift: 4.5, duty: 0.72, offs: walkOffs}, trunk: trunkHang}
}

// kingDraw draws the mammoth king facing right; the anchor is the middle of
// its feet on the ground, in art pixels.
func kingDraw(k kingPose) (*canvas, float64, float64) {
	q := newCritAt(284, 226, 24, 82)
	const gy = 140.0
	const pvx, pvy = 60.0, 102.0 // what it rears about
	const nx, ny = 132.0, 66.0   // the head turns about the neck
	g := k.g
	bob := g.bob(2) + k.drop
	ra := -0.52 * k.rear
	bp := int(math.Round(bob * cK)) // the coat's pattern rides with the body
	BP := func(x, y float64) (float64, float64) {
		return turnXY(x+k.lunge, y+bob, pvx+k.lunge, pvy+bob, ra)
	}
	BS := func(s shape) shape { return s.moved(k.lunge*cK, bob*cK).turned(pvx+k.lunge, pvy+bob, ra) }
	// rest is where image pixel (x, y) lies on the beast as it stands.
	rest := func(x, y int) (float64, float64) {
		u, v := turnXY((float64(x)+0.5)/cK, (float64(y)+0.5)/cK, pvx+k.lunge, pvy+bob, -ra)
		return u - k.lunge, v - bob
	}
	HS := func(s shape) shape { return BS(s.turned(nx, ny, k.headA)) }
	HP := func(x, y float64) (float64, float64) {
		x, y = turnXY(x, y, nx, ny, k.headA)
		return BP(x, y)
	}
	sway := func(x float64) float64 { return 2.4 * k.swing * math.Sin(x*0.07+2*math.Pi*g.ph) }

	// mossy is sh grown over with moss: thick on what faces up, down to
	// deep art pixels from a's upper edge, and in patches elsewhere; more
	// overall pushes it further.
	mossy := func(a shape, sh shader, light func(x, y int) float64, deep, more float64, seed uint32) shader {
		green := locksAt(kingMoss, func(x, y int) float64 { return light(x, y) - 0.04 }, 5, 8, 0.1, seed+3, 0, bp)
		return func(x, y int, d float64) color.RGBA {
			u, v := rest(x, y)
			fx, fy := float64(x)+0.5, float64(y)+0.5
			patch := 0.75*vnoise(u/13, v/9, seed+1) + 0.45*vnoise(u/4.3, v/3.1, seed+2) + 0.2*vnoise(u/1.7, v/1.3, seed+4) - 0.1 + more
			top := deep > 0 && a.d(fx-2, fy-deep*cK*(0.3+0.9*vnoise(u/9, 0.5, seed))) > 0
			if top && patch > 0.46 || patch > 0.8 {
				return green(x, y, d)
			}
			return sh(x, y, d)
		}
	}
	// strand is a strand of moss hanging long art pixels down from (x, y).
	strand := func(x, y, long, b float64, seed uint32) {
		sw := 0.7 * sway(x)
		s := aCap(x, y, x+0.35*sw, y+0.55*long, 0.8, 0.6).or(aCap(x+0.35*sw, y+0.55*long, x+sw, y+long, 0.6, 0.32))
		q.fill(s, func(px, py int, _ float64) color.RGBA {
			return kingMoss.at(2 + int(2.6*hashf(px/2, py/5, seed)) + int(math.Round(b*4)))
		})
	}

	leg := func(i int, hx, hy float64, near bool) {
		fore := i%2 == 1
		dx, up := g.foot(i)
		fx, fy := hx-1+dx, gy-up
		sx, sy := BP(hx, hy)
		if fore && k.rear > 0 { // off the ground, folded under the chest
			lx, ly := sx+18, sy+32
			if !near {
				lx, ly = sx+24, sy+27
			}
			fx, fy = lerpf(fx, lx, smooth(k.rear)), lerpf(fy, ly, smooth(k.rear))
		}
		bend := -1.0
		if fore {
			bend = 1
		}
		kx, ky := ik2(sx, sy, fx, fy-12, 17.5, 17.5, bend)
		b := -0.4
		if near {
			b = -0.12
		}
		seed := 911 + 7*uint32(i)
		foot := aEll(fx+2, fy-6.2, 14.5, 7)
		q.fill(foot, shd(kingSkin, foot, 4, b+0.25))
		for n := 0.0; n < 3; n++ { // nails like claws
			x0, y0 := fx+3+5.2*n, fy-8.4+0.7*n
			nail := aPoly(x0, y0, x0+4.4, y0+1.2, x0+6.6, fy-0.1+0.3*n, x0+1.2, fy-1.4+0.3*n)
			q.fill(nail, shd(kingIvory, nail, 1.6, b+0.14))
		}
		bone := aLeg(sx, sy, kx, ky, fx, fy-12, 16, 13.4, 12.2)
		lg := bone.jag(8, 10, (fy-17)*cK, seed)
		light := lit(bone, 12, b)
		fur := locksAt(kingFur, light, 9, 13, 0.08, seed+1, int(math.Round(0.6*dx*cK)), int(math.Round((bob-0.5*up)*cK)))
		q.fill(lg, mossy(bone, fur, light, 0, -0.14, seed+2))
	}
	tusk := func(far bool) {
		ox, oy, b := 0.0, 0.0, 0.08
		if far {
			ox, oy, b = 11, -5, -0.26
		}
		var c [5][2]float64
		for i, o := range [5][2]float64{{0, 0}, {-2, 27}, {28, 37}, {57, 19}, {60, -14}} {
			c[i][0], c[i][1] = HP(158+ox+o[0], 97+oy+o[1])
		}
		mid := lerp2(c[2], c[3], 0.62)
		pts := bez(16, 6.4, 4.8, c[0], c[1], c[2], mid)
		pts = append(pts, bez(12, 4.8, 1.2, mid, c[3], lerp2(c[3], c[4], 0.6), c[4])[1:]...)
		for j := 5; j < len(pts)-3; j++ { // the moss that hangs from it
			if h := hashf(j, 3, 931+uint32(ox)); h > 0.42 {
				strand(pts[j].x, pts[j].y, pts[j].r+3+13*(h-0.42)/0.58, b, 933+uint32(j))
			}
		}
		tb := newCTube(pts)
		sh := tb.shape()
		ivory := tb.skin(func(x, y int, s, v, l float64) color.RGBA {
			l = 0.16 + 0.8*l + b
			if u := s * 24; u-math.Floor(u) < 0.13 && s < 0.5 {
				l -= 0.1 // the rings of its years
			}
			if s < 0.1 {
				l -= 0.25 * (1 - s/0.1) // stained where it leaves the lip
			}
			if vnoise(s*70, v*2.4, 935) > 0.73 {
				l -= 0.17 // cracked along its length
			}
			return toneOf(kingIvory, l)
		})
		q.fill(sh, mossy(sh, ivory, lit(sh, 4, b), 2.8, 0.02, 937))
	}

	// The far side: legs, tail and tusk.
	leg(2, 66, 93, false)
	leg(3, 131, 93, false)
	var tl [4][2]float64
	for i, o := range [4][2]float64{{27, 64}, {17, 68}, {14.5, 86}, {16.5 - 1.5*k.swing, 101}} {
		tl[i][0], tl[i][1] = BP(o[0], o[1])
	}
	tb := newCTube(bez(8, 4.2, 2.1, tl[0], tl[1], tl[2], tl[3]))
	q.fill(tb.shape(), locks(kingFur, lit(tb.shape(), 3, -0.2), 6, 9, 0, 921))
	tuft := aEll(tl[3][0], tl[3][1]+4.5, 4.6, 7.5).jag(6, 6, (tl[3][1]+2)*cK, 923)
	q.fill(tuft, locks(kingFur, lit(tuft, 3, -0.3), 6, 10, 0, 925))
	tusk(true)

	// The ridge of knobs along the spine, tallest over the hump.
	back := sBlend(18, aEll(52, 78, 30, 31), aEll(86, 70, 40, 35), aEll(116, 55, 35, 46))
	topOf := func(x float64) float64 {
		for y := 0.0; y < 120; y += 0.5 {
			if back.d(x*cK, y*cK) < 0 {
				return y
			}
		}
		return 120
	}
	rock := func(s shape, seed uint32) {
		light := lit(s, 2.5, 0.04)
		q.fill(s, mossy(s, func(x, y int, _ float64) color.RGBA {
			l := light(x, y)
			if vnoise(float64(x)/5, float64(y)/7, seed) > 0.68 {
				l -= 0.16 // pitted
			}
			return toneOf(kingStone, l)
		}, light, 1.6, 0.12, seed+1))
	}
	for x, i := 31.0, 0; x < 139; i++ {
		h := hashf(i, 1, 941)
		r := 2.3 + 3.1*math.Exp(-(x-108)*(x-108)/900) + 1.3*h
		y := topOf(x) + 2.5
		t := r * (1 + 0.6*hashf(i, 2, 941))
		rock(BS(sBlend(1.5, aEll(x-0.2*r, y-0.6*t, 0.9*r, 1.1*t), aEll(x+0.35*r, y-0.2*t, 0.75*r, 0.7*t))), 943+3*uint32(i))
		x += 1.5*r + 1 + 3*h
	}

	leg(0, 52, 96, true)
	leg(1, 118, 96, true)

	// The body: a great hump over the shoulders, the back falling away to
	// low hindquarters, the long hair of the flanks and the mane of the
	// chest hanging over the tops of the legs, and moss hanging lower still.
	belly := func(x float64) float64 {
		sn := math.Sin((x - 52) * math.Pi / 66)
		return 106 - 22*smooth(clamp01((x-124)/16)) - 16*smooth(clamp01((38-x)/12)) - 2.5*sn*sn
	}
	for x, i := 36.0, 0; x < 132; i++ {
		h := hashf(i, 2, 947)
		sx, sy := BP(x, belly(x)-1)
		strand(sx, sy, 9+15*h*h, -0.1, 949+uint32(i))
		x += 2.5 + 7*hashf(i, 3, 947)
	}
	lump := BS(back)
	skirt := BS(hung(28, 140, 74, 11, belly, sway, 951))
	mane := BS(hung(122, 152, 70, 13, func(x float64) float64 { return 100 - 0.75*(x-122) }, sway, 953))
	body := lump.or(skirt).or(mane)
	bodyLight := func(x, y int) float64 {
		u, v := rest(x, y)
		l := lump.lum(x, y, 24*cK) - 0.08
		if lump.d(float64(x)-3, float64(y)-(9+10*vnoise(u/5, 1.5, 955))*cK) > 0 {
			l += 0.12 // the light along the back
		}
		l -= 0.16 * clamp01((v-88)/18)                                // the skirt hangs in the body's shade
		l -= 0.1 * math.Exp(-(u-97)*(u-97)/70) * clamp01((v-44)/16)   // the fold behind the shoulder
		l -= 0.15 * math.Exp(-(u-138)*(u-138)/70) * clamp01((v-36)/8) // and the neck, in the shade of the head
		return l
	}
	q.fill(body, mossy(lump, locksAt(kingFur, bodyLight, 11, 17, 0.12, 957, 0, bp), bodyLight, 10, 0.04, 959))

	// The head, hung low: a heavy dome with a knob on top, a brow like a
	// cliff over the small eye, the face running down into the trunk, a
	// beard under the jaw.
	lip := HS(aEll(159, 99, 7, 4.6))
	q.fill(lip, shd(kingSkin, lip, 3, -0.2))
	for i, p := range [3][3]float64{{141, 37.5, 3.2}, {149, 34.5, 4.2}, {157, 38, 3}} {
		x, y, r := p[0], p[1], p[2]
		rock(HS(aEll(x, y-0.7*r, 0.95*r, 1.3*r)), 981+3*uint32(i))
	}
	skull := sBlend(12, aEll(152, 62, 20, 23), aEll(149, 43, 12.5, 10.5), aEll(161, 86, 13, 16))
	beard := hung(138, 160, 78, 11, func(x float64) float64 { return 103 - 0.35*(x-138) }, sway, 961)
	for i := 0; i < 5; i++ {
		x := 140.5 + 4.2*float64(i)
		sx, sy := HP(x, 100-0.35*(x-138))
		strand(sx, sy, 7+11*hashf(i, 4, 962), 0, 962+uint32(i))
	}
	head := HS(skull.or(beard))
	hs := HS(skull)
	headLight := func(x, y int) float64 {
		l := hs.lum(x, y, 16*cK) - 0.03
		if hs.d(float64(x)-3, float64(y)-7*cK) > 0 {
			l += 0.12
		}
		return l
	}
	headFur := locksAt(kingFur, headLight, 8, 12, 0.3, 963, 0, bp)
	q.edge = 0.22
	q.fill(head, mossy(hs, headFur, headLight, 9, 0, 965))
	q.edge = 0.34
	// The trunk: thick and hairy where it leaves the face, bare ringed
	// skin below, a curl at the tip.
	sx, sy := HP(172, 89)
	const tn, tlen = 36, 56.0
	pts := make([]tpt, tn+1)
	for i := range pts {
		s := float64(i) / tn
		pts[i] = tpt{sx, sy, 9.4 - 6*math.Pow(s, 0.8)}
		u := [6]float64{0, 0.25, 0.5, 0.75, 0.9, 1}
		j := 0
		for j < 4 && s > u[j+1] {
			j++
		}
		a := lerpf(k.trunk[j], k.trunk[j+1], (s-u[j])/(u[j+1]-u[j])) - 0.7*k.headA - 0.7*ra + 0.07*k.swing*s
		sx, sy = sx+math.Sin(a)*tlen/tn, sy+math.Cos(a)*tlen/tn
	}
	tr := newCTube(pts)
	q.fill(tr.shape(), tr.skin(func(x, y int, s, v, l float64) color.RGBA {
		l = 0.12 + 0.8*l
		if u := s*30 + 0.25*v*v; u-math.Floor(u) < 0.24 {
			l -= 0.13 // the rings
		}
		if s < 0.34 && hashf(x/2, (y+x/3)/5, 967) > s/0.34 {
			return toneOf(kingFur, l-0.1)
		}
		return toneOf(kingSkin, l)
	}))
	q.edge = 0.12
	face := HS(aEll(163, 80, 10.5, 8.5)).jag(8, 8, 0, 969) // the fur of the face falls over the root of the trunk
	q.fill(face, headFur)
	ear := HS(aEll(137.5, 73, 7.5, 12.5))
	q.fill(ear, locks(kingFur, lit(ear, 5, -0.3), 6, 9, 0.3, 971))
	fringe := HS(aEll(139, 63.5, 8.5, 7.5)).jag(8, 7, 0, 973) // the ear is half lost in the fur
	q.fill(fringe, mossy(hs, headFur, headLight, 0, -0.05, 975))
	q.edge = 0.34
	tusk(false)
	// The eye: small, deep under the brow, with a dull glow in it.
	ex, ey := HP(165.5, 67.5)
	q.flat(HS(aEll(165.5, 67.5, 4, 3)), plain(kingFur[0]))
	q.flat(HS(aCap(161.5, 72, 168.5, 71.6, 0.5, 0.4)), plain(kingFur[4]))
	q.flat(aEll(ex, ey+0.2, 1.5, 1.2), plain(hex(0xc8791e)))
	q.flat(aEll(ex+0.2, ey+0.2, 0.85, 0.85), plain(hex(0xffd35a)))
	q.dot(ex, ey, 0, 0, colInk)
	q.dot(ex, ey, -1, -1, colWhite)
	q.edge = 0.3
	brow := HS(aEll(162, 61, 9, 5)).jag(5, 7, 0, 977)
	q.fill(brow, locksAt(kingFur, func(x, y int) float64 { return headLight(x, y) - 0.1 }, 8, 12, 0.3, 963, 0, bp))
	q.edge = 0.34
	return hdCrop(q.done(0.8, 90, 139.4))
}

// --- its runs of frames ---------------------------------------------------

const (
	kingIdleN = 6
	kingRoarN = 6
)

// When each frame of the swing of the tusks and of the rearing stomp is
// shown, in frames since it began. The swing lands bossWindup ticks in (36
// frames), the stomp KingRearTicks in (54): see internal/sim/bossfight.go.
var (
	kingSwingAges = [...]int{0, 6, 14, 27, 31, 35, 41, 47}
	kingRearAges  = [...]int{0, 7, 14, 22, 34, 46, 51, 54, 59, 65}
)

var (
	kingIdle = newBHAnim(kingIdleN, func(i int) (*canvas, float64, float64) {
		k := kingStand()
		ph := 2 * math.Pi * float64(i) / kingIdleN
		k.drop, k.swing, k.headA = 0.7*(1+math.Sin(ph)), 0.7*math.Sin(ph-0.9), 0.02*math.Sin(ph)
		return kingDraw(k)
	})
	kingWalk = newBHAnim(cycFrames, func(i int) (*canvas, float64, float64) {
		k := kingStand()
		k.g.on, k.g.ph = true, float64(i)/cycFrames
		k.swing, k.headA = math.Sin(2*math.Pi*k.g.ph), 0.035*math.Sin(4*math.Pi*k.g.ph)
		return kingDraw(k)
	})
	kingCharge = newBHAnim(cycFrames, func(i int) (*canvas, float64, float64) { // tusks low, at a run
		k := kingStand()
		k.g.on, k.g.ph, k.g.stride, k.g.lift, k.g.duty, k.g.offs = true, float64(i)/cycFrames, 23, 7, 0.55, runOffs
		k.swing, k.headA, k.trunk, k.lunge = 1.6*math.Sin(2*math.Pi*k.g.ph), 0.1, trunkCurl, 3
		return kingDraw(k)
	})
	kingSwing = newBHAnim(len(kingSwingAges), func(i int) (*canvas, float64, float64) {
		// Drawn back with the tusks at the ground, then thrown up and forward.
		amt := [...]float64{-0.4, -0.8, -1, -0.25, 0.5, 1, 0.6, 0.2}[i]
		k := kingStand()
		t := smooth((amt + 1) / 2)
		k.headA, k.lunge, k.drop, k.swing = lerpf(0.12, -0.56, t), 7*amt, 3*math.Max(0, -amt), amt
		for j := range k.trunk {
			k.trunk[j] = lerpf(trunkCurl[j], trunkHigh[j], t)
		}
		return kingDraw(k)
	})
	kingRear = newBHAnim(len(kingRearAges), func(i int) (*canvas, float64, float64) {
		// Up on its hind legs, held there, and down with all its weight.
		k := kingStand()
		k.rear = [...]float64{0.2, 0.5, 0.8, 1, 0.96, 1, 0.55, 0, 0, 0}[i]
		k.drop = [...]float64{0, 0, 0, 0, 0, 0, 0, 6, 3.5, 1}[i]
		k.headA, k.trunk, k.swing = -0.12*k.rear+0.02*k.drop, trunkHigh, 1-2*k.rear
		if k.rear == 0 {
			for j := range k.trunk {
				k.trunk[j] = lerpf(trunkHang[j], trunkCurl[j], k.drop/6)
			}
		}
		return kingDraw(k)
	})
	kingRoar = newBHAnim(kingRoarN, func(i int) (*canvas, float64, float64) { // trumpeting, trunk high
		k := kingStand()
		ph := 2 * math.Pi * float64(i) / kingRoarN
		k.headA, k.rear, k.swing, k.trunk = -0.3+0.04*math.Sin(ph), 0.07+0.03*math.Sin(ph), math.Sin(ph), trunkHigh
		for j := range k.trunk {
			k.trunk[j] += 0.16 * math.Sin(ph-0.9*float64(j))
		}
		return kingDraw(k)
	})
	kingLurk = newBHAnim(1, func(int) (*canvas, float64, float64) { // crouched low
		k := kingStand()
		k.drop, k.headA, k.trunk = 12, 0.05, trunkFeed
		return kingDraw(k)
	})
)

// kingSprite is the frame of the mammoth king to show for c: age frames
// into its state, t the frame count of the game.
func kingSprite(c protocol.CreatureView, moving bool, t, age int, left bool) sprite {
	hdWarm(kingIdle, kingWalk, kingCharge, kingSwing, kingRear, kingRoar, kingLurk)
	switch c.State {
	case sim.CreatureRoar:
		return kingRoar.sprite(t/7, left)
	case sim.CreatureStrike, sim.CreatureClaw:
		if moving { // the charge
			return kingCharge.sprite(t/4, left)
		}
		return kingSwing.sprite(megaFrameAt(kingSwingAges[:], age), left)
	case sim.CreatureSpecial, sim.CreatureGrab:
		return kingRear.sprite(megaFrameAt(kingRearAges[:], age), left)
	case sim.CreatureLurk:
		return kingLurk.sprite(0, left)
	case sim.CreatureChase:
		if moving {
			return kingWalk.sprite(t/5, left)
		}
	}
	if moving {
		return kingWalk.sprite(t/8, left)
	}
	return kingIdle.sprite(t/14, left)
}

// mammothKingCanvas is pose p of the mammoth king as a still, for bossCanvas.
func mammothKingCanvas(p bossPose) (*canvas, float64, float64) {
	a, i := kingIdle, 0
	switch p {
	case bossWalkA:
		a = kingWalk
	case bossWalkB:
		a, i = kingWalk, cycFrames/2
	case bossAttack:
		a, i = kingSwing, 5
	case bossSpecial:
		a, i = kingRear, 3
	case bossRoar:
		a = kingRoar
	case bossLurk:
		a = kingLurk
	}
	c, ax, ay := a.draw(i)
	c.dens = HDDensity
	return c, ax * HDDensity, ay * HDDensity // mk takes the anchor of a finer canvas in its own pixels
}
