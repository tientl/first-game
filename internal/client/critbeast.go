package client

import (
	"image/color"
	"math"
)

// The furred beasts at HD density: the woolly mammoth, the sabre-tooth, the
// cave bear and the monkey. Each is a body of blended lumps under tufted
// fur, on legs of two bones that reach for where the gait puts each foot.

func init() {
	set := func(name string, hd func(pose, f int) (*canvas, float64, float64), anim map[int]int) {
		critKinds[name].hd, critKinds[name].anim = hd, anim
	}
	set("mammoth", mammothHD, map[int]int{mammothWalkA: cycFrames, mammothCharge: cycFrames, mammothGore: blowFrames, mammothSkHeadDown: 2})
	set("sabre", sabreHD, map[int]int{sabreWalkA: cycFrames, sabreWalkB: cycFrames, sabreStalk: cycFrames, sabreSpring: blowFrames, sabreSkLow: 1})
	set("bear", bearHD, map[int]int{bearWalkA: cycFrames, bearWalkB: cycFrames, bearClaw: blowFrames, bearBite: blowFrames})
	set("monkey", monkeyHD, map[int]int{monkeyWalkA: cycFrames, monkeyLeap: cycFrames})
}

// gait is a four-footed stride: where each foot is at phase ph. The feet
// are numbered near hind, near fore, far hind, far fore.
type gait struct {
	on           bool
	ph           float64
	stride, lift float64
	duty         float64
	offs         [4]float64
}

var (
	walkOffs = [4]float64{0, 0.25, 0.5, 0.75} // one foot after another
	trotOffs = [4]float64{0, 0.5, 0.5, 0}     // in diagonal pairs
	runOffs  = [4]float64{0, 0.42, 0.1, 0.52} // a gallop: the hind pair, then the fore pair
)

func (g gait) foot(i int) (dx, up float64) {
	if !g.on {
		return 0, 0
	}
	return pace(g.ph+g.offs[i], g.duty, g.stride, g.lift)
}

// bob is how far the body sinks at this point of the stride.
func (g gait) bob(amp float64) float64 {
	if !g.on {
		return 0
	}
	return amp * 0.5 * (1 + math.Sin(4*math.Pi*g.ph))
}

// lerp2 is the point t of the way from a to b.
func lerp2(a, b [2]float64, t float64) [2]float64 {
	return [2]float64{lerpf(a[0], b[0], t), lerpf(a[1], b[1], t)}
}

// locks is a coat of long hair over a surface lit by light: overlapping
// rows of pointed locks w wide and about two rows of h long (image pixels),
// each lit down its left side and dark down its right and at its tip, the
// upper rows lying over the lower. shear leans the locks back as they fall.
func locks(r ramp, light func(x, y int) float64, w, h, shear float64, seed uint32) shader {
	return locksAt(r, light, w, h, shear, seed, 0, 0)
}

// locksAt is locks with its pattern moved (dx, dy) image pixels, so the
// hair of a part that moves goes with it.
func locksAt(r ramp, light func(x, y int) float64, w, h, shear float64, seed uint32, dx, dy int) shader {
	n := float64(len(r))
	return func(x, y int, _ float64) color.RGBA {
		i := int(light(x, y) * n)
		x, y = x-dx, y-dy
		fx, fy := float64(x)+0.5+shear*float64(y), float64(y)+0.5
		r0 := int(math.Floor(fy / h))
		di := -1 // between the locks: the dark undercoat
	find:
		for row := r0 - 2; row <= r0; row++ {
			st := 0.5*float64(row&1) + hashf(row, 1, seed)
			j0 := int(math.Floor(fx/w - st))
			for j := j0 - 1; j <= j0+1; j++ {
				h1, h2, h3 := hashf(j, row, seed), hashf(j, row, seed+1), hashf(j, row, seed+2)
				t := (fy - (float64(row)+0.6*h2)*h) / (h * (1.5 + 0.9*h3))
				if t < 0 || t >= 1 {
					continue
				}
				cx := (float64(j) + st + 0.5 + 0.5*(h1-0.5) + 0.7*(h3-0.5)*t*t) * w
				u := (fx - cx) / (w * 0.7 * (1 - t*t*t))
				if u < -1 || u > 1 {
					continue
				}
				switch {
				case u > 0.5 || t > 0.9:
					di = -1
				case u < -0.15 && t > 0.12 && t < 0.8:
					di = 1
					if h1 > 0.78 && u < -0.45 {
						di = 2 // a strand that catches the light
					}
				case row == r0 && t < 0.2:
					di = -1 // its root, under the row above
				default:
					di = 0
				}
				if h2 > 0.86 {
					di--
				}
				break find
			}
		}
		return r.at(max(0, min(len(r)-1, max(1, i)+di)))
	}
}

// lit is the light on a as a bulging surface, shifted by bias.
func lit(a shape, round, bias float64) func(x, y int) float64 {
	return func(x, y int) float64 { return a.lum(x, y, round*cK) + bias }
}

// frosted lays snow on what faces up: sh with a dusting over the top of a,
// up to deep art pixels down from its upper edge.
func frosted(a shape, sh shader, deep float64, seed uint32) shader {
	return func(x, y int, d float64) color.RGBA {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		dep := deep * cK * (0.25 + 0.75*vnoise(fx/13, 0.5, seed))
		if a.d(fx, fy) <= 0 && a.d(fx-1.5, fy-dep) > 0 && vnoise(fx/4.5, fy/3.5, seed+1) > 0.52 {
			return snowRamp.at(2 + int(3.4*hashf(x/2, y/2, seed+2)))
		}
		return sh(x, y, d)
	}
}

// hung is the hair that hangs under a coat between art columns x0 and x1,
// from row top down to base(x) and on in pointed locks up to long art
// pixels longer; sway swings their tips sideways.
func hung(x0, x1, top, long float64, base func(x float64) float64, sway func(x float64) float64, seed uint32) shape {
	return shape{x0 * cK, top * cK, x1 * cK, 200 * cK, func(px, py float64) float64 {
		x, y := px/cK, py/cK
		b := base(x)
		x -= sway(x) * clamp01((y-b+2)/(long+2))
		tooth := func(period float64, sd uint32) float64 {
			u := x/period + 40
			k := math.Floor(u)
			w := math.Abs(u-k-0.5) * 2
			return long * (0.35 + 0.65*hashf(int(k), 5, sd)) * (1 - w*math.Sqrt(w))
		}
		yb := b + math.Max(tooth(3.1, seed), tooth(4.7, seed+1))
		return math.Max(math.Max(y-yb, top-y), math.Max(x0-x, x-x1)) * cK
	}}
}

// --- the woolly mammoth ----------------------------------------------------------

// The trunk as the way it points at six places along it (0 straight down,
// a quarter turn forwards, a half turn straight up).
var (
	trunkHang = [6]float64{0.75, 0.2, -0.02, 0.1, 0.9, 2.3}
	trunkCurl = [6]float64{0.7, 0.05, -0.9, -2.0, -3.0, -3.9}
	trunkHigh = [6]float64{1.5, 2.25, 2.9, 3.3, 3.0, 2.1}
	trunkCrop = [6]float64{0.7, 0.15, 0, -0.3, -1.1, -1.9}
	trunkFeed = [6]float64{0.7, 0, -1.1, -2.3, -3.1, -3.7}
)

func mammothHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(134, 94, 2, 10)
	const gy = 79.6
	g := gait{stride: 9, lift: 3, duty: 0.7, offs: walkOffs}
	headA, swing := 0.0, 0.0
	earK := 1.0 // how far its ears stand out: flared to charge
	trunk := trunkHang
	switch pose {
	case mammothWalkA:
		if f > 0 {
			g.on, g.ph = true, cycPh(f)
			swing = math.Sin(2 * math.Pi * g.ph)
		}
	case mammothWalkB:
		g.on, g.ph = true, 0.5
	case mammothCharge:
		headA, trunk = 0.34, trunkCurl
		g.on, g.ph, g.stride, g.lift, g.duty, g.offs = true, 0.1, 13, 4.5, 0.55, runOffs
		if f > 0 {
			g.ph = cycPh(f)
			swing = 1.6 * math.Sin(2*math.Pi*g.ph)
		}
	case mammothGore:
		t := smooth((blowAmt(f) + 1) / 2) // head down and trunk curled, then tossed up
		headA = lerpf(0.42, -0.38, t)
		for i := range trunk {
			trunk[i] = lerpf(trunkCurl[i], trunkHigh[i], t)
		}
	case mammothSkHeadDown: // head down, ears flared, a forefoot scraping: see critskills.go
		headA, trunk, earK = 0.5, trunkCurl, 1.5
		g.on, g.ph = true, [3]float64{0.02, 0.12, 0.2}[min(f, 2)]
	case mammothSkCurl: // the trunk curled up high, to swipe
		headA, trunk, earK = -0.14, trunkCoil, 1.2
	case mammothSkSwipe: // the trunk lashed out forwards
		headA, trunk = 0.16, trunkLash
	case mammothSkTrumpet: // the trunk raised, trumpeting
		headA, trunk, earK = -0.42, trunkHigh, 1.35
	case mammothEatA:
		headA, trunk = 0.13, trunkCrop
	case mammothEatB:
		headA, trunk = 0.04, trunkFeed
	}
	bob := g.bob(1.2)
	bp := int(math.Round(bob * cK)) // the coat's pattern rides with the body
	const nx, ny = 80.0, 38.0       // the head turns about the neck
	H := func(s shape) shape { return s.turned(nx, ny+bob, headA) }
	HP := func(x, y float64) (float64, float64) { return turnXY(x, y+bob, nx, ny+bob, headA) }
	sway := func(x float64) float64 { return 1.5 * swing * math.Sin(x*0.11+2*math.Pi*g.ph) }

	leg := func(i int, hx, hy float64, near bool) {
		dx, up := g.foot(i)
		fx, fy := hx-1+dx, gy-up
		bend := -1.0
		if i%2 == 1 {
			bend = 1
		}
		kx, ky := ik2(hx, hy+bob, fx, fy-3, 9.9, 9.8, bend)
		b := -0.42
		if near {
			b = -0.16
		}
		foot := aEll(fx+0.8, fy-2.8, 7, 3.4)
		q.fill(foot, shd(mammothSkin, foot, 2.4, b+0.2))
		for k := 0.0; k < 3; k++ { // toenails
			nail := aEll(fx+2.4+2.3*k, fy-1.2+0.25*k, 1.05, 1.25)
			q.flat(nail, shd(tuskRamp, nail, 1, b-0.1))
		}
		bone := aLeg(hx, hy+bob, kx, ky, fx, fy-6.6, 8.2, 6.6, 5.9)
		lg := bone.jag(5, 6, (fy-10)*cK, 311+uint32(i))
		q.fill(lg, locksAt(mammothFur, lit(bone, 7, b), 5, 7, 0.08, 313+uint32(i), int(math.Round(0.6*dx*cK)), int(math.Round((bob-0.5*up)*cK))))
	}
	tusk := func(far bool) {
		ox, oy, b := 0.0, 0.0, 0.1
		if far {
			ox, oy, b = 6, -2.5, -0.24
		}
		var c [5][2]float64
		for i, o := range [5][2]float64{{0, 0}, {-2, 15}, {16, 27}, {26, 10}, {19, 1}} {
			c[i][0], c[i][1] = HP(97+ox+o[0], 46+oy+o[1])
		}
		pts := bez(14, 2.7, 1.8, c[0], c[1], c[2], lerp2(c[2], c[3], 0.62))
		pts = append(pts, bez(10, 1.8, 0.5, lerp2(c[2], c[3], 0.62), c[3], lerp2(c[3], c[4], 0.6), c[4])[1:]...)
		tb := newCTube(pts)
		q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
			l = 0.2 + 0.8*l + b
			if u := s * 16; u-math.Floor(u) < 0.14 && s < 0.4 {
				l -= 0.12 // growth rings near the root
			}
			if s < 0.12 {
				l -= 0.25 * (1 - s/0.12) // stained where it leaves the lip
			}
			return toneOf(tuskRamp, l)
		}))
	}

	// The far side: legs, tail and tusk.
	leg(2, 45, 55, false)
	leg(3, 83, 54, false)
	tb := newCTube(bez(8, 2.3, 1.2, [2]float64{21, 36 + bob}, [2]float64{15.5, 37 + bob}, [2]float64{15, 44 + bob}, [2]float64{16 - 0.8*swing, 50 + bob}))
	q.fill(tb.shape(), locks(mammothFur, lit(tb.shape(), 2, -0.2), 4, 6, 0, 317))
	tuft := aEll(16-1.2*swing, 53+bob, 2.5, 4.2).jag(4, 4, (52+bob)*cK, 319)
	q.fill(tuft, locks(mammothFur, lit(tuft, 2, -0.3), 4, 7, 0, 321))
	tusk(true)
	leg(0, 36, 57, true)
	leg(1, 75, 57, true)

	// The body: a high shoulder hump, the back sloping away to low
	// hindquarters, and the long hair of the flanks and belly hanging over
	// the tops of the legs.
	lump := sBlend(9, aEll(36, 46+bob, 17, 17.5), aEll(55, 43+bob, 24, 19.5), aEll(69, 34+bob, 18.5, 21.5))
	skirt := hung(24, 87, 46+bob, 5.5, func(x float64) float64 {
		sn := math.Sin((x - 36) * math.Pi / 39)
		return 62.5 + bob - 13*smooth(clamp01((x-76)/11)) - 10*smooth(clamp01((31-x)/8)) - 1.6*sn*sn
	}, sway, 323)
	body := lump.or(skirt)
	bodyLight := func(x, y int) float64 {
		ax, ay := float64(x)/cK, float64(y)/cK-bob
		l := lump.lum(x, y, 15*cK) - 0.07
		if lump.d(float64(x)-3, float64(y)-(6+7*vnoise(ax/3, 1.5, 325))*cK) > 0 {
			l += 0.13 // sun-bleached along the back
		}
		l -= 0.15 * clamp01((ay-52)/12)                                // the skirt hangs in the body's shade
		l -= 0.1 * math.Exp(-(ax-62)*(ax-62)/30) * clamp01((ay-28)/10) // the fold behind the shoulder
		l -= 0.14 * math.Exp(-(ax-81)*(ax-81)/24)                      // and the neck, in the shade of the head
		return l
	}
	q.fill(body, frosted(lump, locksAt(mammothFur, bodyLight, 8, 12, 0.12, 327, 0, bp), 4.5, 329))

	// The head: a high dome with a knob on top, the face running down into
	// the trunk, a beard under the jaw.
	lip := H(aEll(97, 48.5+bob, 3.6, 2.4))
	q.fill(lip, shd(mammothSkin, lip, 2, -0.2))
	skull := sBlend(9, aEll(94.5, 23.5+bob, 12.5, 14.5), aEll(94, 10.5+bob, 7.4, 6.4), aEll(99, 40+bob, 8, 9.5))
	beard := hung(84, 98, 36+bob, 4.5, func(x float64) float64 { return 48 + bob - 0.3*(x-84) }, sway, 331)
	head := H(skull.or(beard))
	hs := H(skull)
	headLight := func(x, y int) float64 {
		l := hs.lum(x, y, 10*cK) - 0.02
		if hs.d(float64(x)-3, float64(y)-5*cK) > 0 {
			l += 0.12
		}
		return l
	}
	ear := H(aEll(85.5-1.5*(earK-1), 32+bob, 4.2*earK, 6.8*(1+0.4*(earK-1))))
	q.edge = 0.22
	q.fill(head, frosted(hs, locksAt(mammothFur, headLight, 6, 8, 0.3, 333, 0, bp), 3, 335))
	q.edge = 0.34
	// The trunk: thick and hairy where it leaves the face, bare ringed skin
	// below, a curl at the tip.
	sx, sy := HP(104.5, 42)
	const tn, tlen = 30, 39.0
	pts := make([]tpt, tn+1)
	for i := range pts {
		s := float64(i) / tn
		pts[i] = tpt{sx, sy, 5 - 3.3*math.Pow(s, 0.8)}
		u := [6]float64{0, 0.25, 0.5, 0.75, 0.9, 1}
		k := 0
		for k < 4 && s > u[k+1] {
			k++
		}
		a := lerpf(trunk[k], trunk[k+1], (s-u[k])/(u[k+1]-u[k])) - 0.7*headA + 0.07*swing*s
		sx, sy = sx+math.Sin(a)*tlen/tn, sy+math.Cos(a)*tlen/tn
	}
	tr := newCTube(pts)
	q.fill(tr.shape(), tr.skin(func(x, y int, s, v, l float64) color.RGBA {
		l = 0.12 + 0.8*l
		if u := s*26 + 0.25*v*v; u-math.Floor(u) < 0.24 {
			l -= 0.13 // the rings
		}
		if s < 0.3 && hashf(x/2, (y+x/3)/5, 337) > s/0.3 {
			return toneOf(mammothFur, l-0.14)
		}
		return toneOf(mammothSkin, l)
	}))
	brow := H(aEll(100, 37.5+bob, 6.4, 5)).jag(5, 5, 0, 339) // the fur of the face falls over the root of the trunk
	q.edge = 0.12
	q.fill(brow, locksAt(mammothFur, headLight, 6, 8, 0.3, 333, 0, bp))
	q.fill(ear, locks(mammothFur, lit(ear, 3, -0.3), 4, 6, 0.3, 341))
	fringe := H(aEll(86.5, 27.5+bob, 4.6, 4)).jag(5, 4, 0, 343) // the ear is half lost in the fur
	q.fill(fringe, locksAt(mammothFur, headLight, 6, 8, 0.3, 333, 0, bp))
	q.edge = 0.34
	tusk(false)
	ex, ey := HP(100, 30.5)
	q.flat(H(aEll(100, 30.5+bob, 2.1, 1.7)), plain(mammothFur[1]))
	q.flat(H(aCap(98.6, 32.6+bob, 101.6, 32.4+bob, 0.45, 0.4)), plain(mammothFur[5]))
	q.flat(H(aCap(97.6, 28.3+bob, 102.3, 28.7+bob, 0.55, 0.4)), plain(mammothFur[0]))
	q.bead(ex, ey+0.1, 2.2)
	return q.done(0.8, 61, 78)
}

// --- the sabre-tooth ---------------------------------------------------------------

// finer is r with a tone between each two of its own: short fur shades
// more softly than long.
func finer(r ramp) ramp {
	out := ramp{r[0]}
	for i := 1; i < len(r); i++ {
		out = append(out, mix(r[i-1], r[i], 0.5), r[i])
	}
	return out
}

// The sabre-tooth's coat: tawny orange over the back and flanks, marked
// with dark rosettes and broken stripes; pale grey-white legs, chest,
// throat, belly and jaws.
var (
	sabreTawny = finer(mkRamp(0x2e160b, 0x54290f, 0x853f14, 0xb35c1c, 0xd27d2c, 0xe69c45, 0xf2bd6e))
	sabreGrey  = finer(mkRamp(0x4b4744, 0x77736e, 0x9f9b94, 0xc2bfb6, 0xdddbd2, 0xf2f1eb))
	sabreDark  = mkRamp(0x180c07, 0x26130a, 0x3a1e0e, 0x4d2a14)
	sabreIvory = mkRamp(0x8a8478, 0xc4c0b4, 0xe8e6dc, 0xfbfbf5, 0xffffff)
)

// sabreMark is the marking of the coat at (u, v), art pixels of the cat's
// side: 1 the dark ring of a rosette, open on one side, 2 the deeper tawny
// it rings, 0 the plain coat. solid makes plain spots of them, k times as
// big.
func sabreMark(u, v, k float64, solid bool) int {
	u, v = u/k, v/k
	u, v = u+1.5*(vnoise(u*0.3, v*0.3, 411)-0.5), v+1.5*(vnoise(u*0.3+7, v*0.3, 412)-0.5)
	const cw, ch = 5.3, 4.3
	row := int(math.Floor(v / ch))
	for r := row - 1; r <= row+1; r++ {
		st := 0.5 * float64(r&1)
		col := int(math.Floor(u/cw - st))
		for c := col - 1; c <= col+1; c++ {
			h1, h2, h3 := hashf(c, r, 413), hashf(c, r, 414), hashf(c, r, 415)
			mx := (float64(c) + st + 0.5 + 0.5*(h1-0.5)) * cw
			my := (float64(r) + 0.5 + 0.4*(h2-0.5)) * ch
			rx := 1.7 + 0.75*h3
			ry := rx * (0.7 + 0.2*h1)
			du, dv := u-mx, v-my
			e := math.Hypot(du/rx, dv/ry)
			if e >= 1 {
				continue
			}
			if solid {
				if e < 0.72 {
					return 1
				}
				continue
			}
			if e < 0.46 {
				return 2
			}
			if a := math.Atan2(dv, du); math.Cos(a-2*math.Pi*h2) > 0.66 {
				return 0 // where the ring is broken
			}
			return 1
		}
	}
	return 0
}

func sabreHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(82, 54, 6, 12)
	const gy = 38.9
	g := gait{stride: 7.5, lift: 2.6, duty: 0.65, offs: walkOffs}
	// flex is the spine in a run: 1 gathered, the hind feet under the
	// belly, -1 stretched out. jaw is how wide the jaws gape: at 1 nearly a
	// right angle, as only a sabre-tooth's can.
	var crouch, pitch, hdx, hdy, ha, jaw, reach, flex float64
	earsBack, air := false, false
	switch pose {
	case sabreWalkA:
		g.on, g.ph = true, 0.04
		if f > 0 {
			g.ph = cycPh(f)
			hdy = 0.5 * math.Sin(4*math.Pi*g.ph)
		}
	case sabreWalkB:
		g.on, g.ph = true, 0.54
		if f > 0 { // running
			g.ph, g.stride, g.lift, g.duty, g.offs = cycPh(f), 12, 4, 0.48, runOffs
			flex = math.Sin(2*math.Pi*g.ph + 0.5)
			pitch = 0.07 * math.Sin(2*math.Pi*g.ph-0.6)
			hdx, hdy = 1-flex, 0.8*flex
			jaw = 0.12
		}
	case sabreStalk:
		crouch, hdx, hdy, ha = 4.5, 3, 1.2, -0.12
		if f > 0 {
			g.on, g.ph, g.stride, g.lift = true, cycPh(f), 5.5, 1.6
		}
	case sabreSpring:
		if amt := blowAmt(f); amt < 0 { // gathered on its haunches, snarling
			crouch, pitch, hdx, hdy = -6*amt, 0.16*amt, 1, -1.5*amt
			jaw = 0.35 + 0.4*(1+amt)
			flex = -0.8 * amt
		} else {
			air, reach, pitch, jaw = true, amt, -0.34*amt, 0.85+0.15*amt
			hdx, hdy, ha = 2, -1, -0.08
			flex = -amt
		}
	case sabreSkLow: // flat to the ground, about to pounce; f twitches the haunches: see critskills.go
		crouch, hdx, hdy, ha, pitch = 7.5, 3.5, 2.6, -0.12, 0.05
		if f%2 == 1 {
			pitch, hdy = 0.09, 2.2
		}
	case sabreSkGape: // the jaws open very wide
		crouch, hdx, hdy, ha, jaw, earsBack = 1.5, 1.2, -1.8, -0.38, 1, true
	case sabreSkBite: // the head thrown forward, the jaws shutting
		crouch, hdx, hdy, ha, jaw, pitch = 1, 3.8, 0.6, 0.22, 0.3, 0.05
	case sabreCower:
		crouch, hdx, hdy, ha, earsBack = 8.2, 1, -2.2, 0.1, true
	case sabreEatA, sabreEatB:
		crouch, pitch, hdx, hdy, ha = 2.5, 0.2, -1, 3.4, 1.1
		if pose == sabreEatB {
			jaw, hdy = 0.42, 2.6
		}
	}
	crouch += g.bob(0.8)
	const cx, cy0 = 27.0, 24.0
	cy := cy0 + crouch
	if air {
		cy -= 3
	}
	py := int(math.Round((cy - cy0) * cK)) // the coat's pattern rides with the body
	T := func(s shape) shape { return s.turned(cx, cy, pitch) }
	P := func(x, y float64) (float64, float64) { return turnXY(x, y+cy-cy0, cx, cy, pitch) }
	// side is where image pixel (x, y) lies on the cat's side as it stands.
	side := func(x, y int) (float64, float64) {
		u, v := turnXY((float64(x)+0.5)/cK, (float64(y)+0.5)/cK, cx, cy, -pitch)
		return u, v - (cy - cy0)
	}
	var whole shape // the body, for the light on the muscles that lie on it
	// tawny is the marked coat of the back over s; marks 0 leaves it
	// plain, 1 gives the rosettes of the flanks and the stripes of the
	// spine and neck.
	tawny := func(s shape, b float64, marks int) shader {
		light := func(x, y int) float64 {
			l := s.lum(x, y, 5*cK) + b + 0.05
			if marks == 1 && whole.d != nil {
				l = 0.4*l + 0.6*(whole.lum(x, y, 5*cK)+b+0.05)
			}
			return l
		}
		base := locksAt(sabreTawny, light, 3.2, 4.2, 0.3, 401, 0, py)
		deep := locksAt(sabreTawny, func(x, y int) float64 { return light(x, y) - 0.13 }, 3.2, 4.2, 0.3, 401, 0, py)
		if marks == 0 {
			return base
		}
		return func(x, y int, d float64) color.RGBA {
			u, v := side(x, y)
			m := sabreMark(u, v, 1, false)
			switch {
			case u > 39.5: // the neck: broken stripes running down it
				m = 0
				if vnoise((u+0.45*v)*0.62, (v-0.45*u)*0.2, 416) > 0.58 {
					m = 1
				}
			case v < cy0-4.6 && vnoise(u*0.3, v*1.1, 417) > 0.63:
				m = 1 // and along the spine
			case v < cy0-6.4:
				m = 0
			}
			switch m {
			case 1:
				return toneOf(sabreDark, light(x, y)+0.05)
			case 2:
				return deep(x, y, d)
			}
			return base(x, y, d)
		}
	}
	grey := func(s shape, b float64, dx int) shader {
		return locksAt(sabreGrey, lit(s, 4, b+0.22), 3.2, 4.2, 0.3, 402, dx, py)
	}
	// pale is hi, turning to lo inside in: a soft, broken edge.
	pale := func(hi, lo shader, in shape) shader {
		return func(x, y int, d float64) color.RGBA {
			if in.d(float64(x)+0.5, float64(y)+0.5) < (hashf(x/2, y/2, 409)-0.6)*1.5*cK {
				return lo(x, y, d)
			}
			return hi(x, y, d)
		}
	}
	// limb is the pale coat of a leg, barred with dark down to the knee;
	// (dx, hy) is where its pattern hangs from.
	limb := func(s shape, b float64, dx int, hy float64, seed uint32) shader {
		lo := grey(s, b, dx)
		return func(x, y int, d float64) color.RGBA {
			u, v := (float64(x-dx)+0.5)/cK, (float64(y)+0.5)/cK-hy
			if v < 6.6+1.5*hashf(x/3, 0, seed) {
				w := (v + 0.45*math.Sin(u*0.8+float64(seed))) / 2.9
				if w-math.Floor(w) < 0.36 && vnoise(u*0.8, v*0.5, seed) > 0.33 {
					return toneOf(sabreDark, s.lum(x, y, 4*cK)+b+0.15)
				}
			}
			return lo(x, y, d)
		}
	}
	claw := hex(0x2b211c)
	// A broad paw, its claws showing; out makes them long, unsheathed.
	paw := func(fx, fy, b float64, fore, out bool, dx int) {
		r := 1.75
		if fore {
			r = 2.15
		}
		pw := aCap(fx-0.2, fy-r+0.5, fx+1.5*r, fy-r+0.85, r, r*0.8)
		for k := 0.0; k < 3; k++ {
			x, l := fx+0.95*r+0.62*r*k, 1.0
			if out {
				l = 1.9
			}
			q.fill(aPoly(x, fy-1.1, x+0.9+0.5*l, fy+0.3*l+0.1, x+0.1, fy+0.25), plain(claw))
		}
		q.fill(pw, grey(pw, b+0.05, dx))
		for k := 1.0; k < 3; k++ { // between the toes
			x := fx + 0.85*r + 0.6*r*k
			q.flat(aCap(x, fy-0.95, x+0.25, fy-0.1, 0.2, 0.17), plain(sabreGrey[2]))
		}
	}
	hipX := 2.6 * flex
	leg := func(i int, hx0, hy0 float64, near bool) {
		hind := i%2 == 0
		if hind {
			hx0 += hipX
		}
		hx, hy := P(hx0, hy0)
		dx, up := g.foot(i)
		fx, fy := hx0-1+dx, gy-up
		if air {
			if hind {
				fx, fy = hx-5-5*reach, hy+9-2*reach
			} else {
				fx, fy = hx+5+4*reach, hy+9-2*reach
			}
		} else if !hind && pitch > 0.1 {
			fx += 3 // eating: the fore paws out in front
		}
		bend := 1.0
		if hind {
			bend = -1
		}
		kx, ky := ik2(hx, hy, fx, fy-1.6, 6.6, 6.6, bend)
		// A fore leg like a pillar; a thigh that narrows to a thin hock.
		r0, r1, r2 := 5.2, 3.9, 2.9
		if hind {
			r0, r1, r2 = 4.8, 2.6, 2.0
		}
		lg := aLeg(hx, hy, kx, ky, fx, fy-2.2, r0, r1, r2)
		b := -0.24
		if near {
			b = 0
		}
		sl := int(math.Round((fx - hx0 + 1) * 0.6 * cK))
		q.fill(lg, limb(lg, b, sl, hy, 431+uint32(i)))
		paw(fx, fy, b, !hind, air && !hind, sl)
	}
	leg(2, 19, 25, false)
	leg(3, 41.5, 25.5, false)
	// A bobbed tail.
	tx, ty := P(10.5+hipX, 21)
	tb := newCTube(bez(6, 2, 1.5, [2]float64{tx + 1, ty}, [2]float64{tx - 1.6, ty - 0.6}, [2]float64{tx - 3.2, ty - 1.2 + 2*clamp01(crouch/8)}, [2]float64{tx - 3.8, ty - 3 + 4*clamp01(crouch/8)}))
	q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
		if s > 0.7 || s > 0.3 && s < 0.48 {
			return toneOf(sabreDark, l+0.1) // ringed, and dark at the tip
		}
		return toneOf(sabreTawny, l)
	}))

	// The body: a deep chest under a hump of shoulder, the back sloping
	// away to lower, lighter hindquarters; the head carried low before it.
	hx, hy := P(47.5+hdx, 18.5+hdy)
	nkx, nky := P(39.5, 19.6)
	rump := T(aEll(17+hipX, cy+0.5, 7.6, 6.6))
	waist := T(aEll(27+0.5*hipX, cy+0.3-1.5*flex, 10.5, 5.5+0.5*flex))
	chest := T(aEll(37.5, cy-1.9, 9.6, 8.7))
	neck := aCap(nkx, nky, hx-1.5, hy+0.8, 6.7, 5.2)
	body := sBlend(4, rump, waist, chest, neck)
	under := T(sBelow((cy + 2.9) * cK)).or(T(aEll(44, cy+2.2, 4.6, 6.4))).or(aCap(nkx+4, nky+5.4, hx+0.5, hy+5, 2.9, 2.6))
	q.fill(body, pale(tawny(body, 0, 1), grey(body, 0, 0), under))
	leg(0, 15.5, 26, true)
	leg(1, 38, 27, true)
	// The muscle of the haunch and the shoulder blade, which rolls as the
	// fore leg under it swings.
	q.edge, whole = 0.1, body
	low := T(sBelow((cy + 4.6) * cK))
	haunch := T(aEll(16.6+hipX, cy+1.5, 6, 6.3))
	q.fill(haunch, pale(tawny(haunch, 0.01, 1), grey(haunch, 0, 0), low))
	fdx, _ := g.foot(1)
	roll := -fdx / math.Max(1, g.stride)
	if !g.on {
		roll = 0
	}
	blade := T(aEll(37.8+1.6*roll, cy-1.6-1.3*roll-0.25*crouch+0.6*reach, 5.8, 7.3))
	q.fill(blade, pale(tawny(blade, 0.03, 1), grey(blade, 0, 0), low))
	q.edge = 0.34

	// The head: a heavy skull, small ears, a ruff under the cheek, pale
	// jaws and the two long sabres. The upper jaw lifts and the lower one
	// drops away from it.
	ha -= 0.5 * jaw
	ja := 1.38 * jaw
	R := func(s shape) shape { return s.turned(hx, hy, ha) }
	RP := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ha) }
	J := func(s shape) shape { return R(s.turned(hx+0.3, hy+3, ja)) } // on the lower jaw
	fang := func(dx, b float64) {
		var c [4][2]float64
		for i, o := range [4][2]float64{{6.3, 2.2}, {8.4, 6.4}, {7.9, 10.6}, {5.2, 13.6}} {
			c[i][0], c[i][1] = RP(hx+o[0]+dx, hy+o[1])
		}
		tb := newCTube(bez(12, 1.3, 0.28, c[0], c[1], c[2], c[3]))
		q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
			return toneOf(sabreIvory, 0.5+0.7*l+b)
		}))
	}
	fang(-1.8, -0.34)
	ruff := R(aEll(hx-2.4, hy+4, 4, 3.6).jag(6, 5, (hy+3.4)*cK, 421))
	q.fill(ruff, grey(ruff, -0.06, 0))
	if jaw > 0 {
		tx, ty := turnXY(hx+7.6, hy+5, hx+0.3, hy+3, ja)
		q.fill(R(aPoly(hx+0.3, hy+2.2, hx+8.2, hy+3.4, tx, ty)), plain(hex(0x5c1219))) // the mouth
		q.flat(J(aCap(hx+1.6, hy+3, hx+5.9, hy+3.2, 1.15, 0.8)), plain(hex(0xdc7480))) // the tongue
		q.flat(J(aCap(hx+2.4, hy+2.8, hx+4.8, hy+3, 0.35, 0.3)), plain(hex(0xf0a0a6)))
		lower := J(aCap(hx+0.8, hy+4.5, hx+7.3, hy+5.1, 1.75, 1.25))
		q.fill(J(aPoly(hx+6.2, hy+4.3, hx+7.4, hy+4.3, hx+7, hy+2.3)), plain(sabreIvory[3])) // a lower fang
		q.fill(lower, grey(lower, -0.02, 0))
	} else {
		chin := R(aCap(hx+1.5, hy+4.1, hx+5.8, hy+4.8, 1.6, 1.25))
		q.fill(chin, grey(chin, -0.06, 0))
	}
	earA := -0.3
	if earsBack {
		earA = -1.5
	}
	ear := R(aEll(hx-2.6, hy-6, 1.7, 2.1).turned(hx-2.4, hy-4, earA))
	q.fill(ear, plain(sabreDark[2]))
	skull := R(sBlend(2, aEll(hx, hy-0.2, 6.4, 5.8), aEll(hx+5.4, hy+1.3, 4.3, 3.6), aEll(hx-1, hy+2.8, 4.6, 3.8)))
	muzzle := R(aEll(hx+6.2, hy+3, 4.7, 2.9).or(aEll(hx+7.4, hy+1.4, 2.5, 2.1)).or(aEll(hx+1.2, hy+4.9, 5.2, 2.5)).or(aEll(hx+3.4, hy-0.2, 1.9, 1.3)))
	q.fill(skull, pale(tawny(skull, 0.06, 0), grey(skull, 0.02, 0), muzzle))
	q.flat(R(aEll(hx-2.5, hy-6, 0.7, 1.1).turned(hx-2.4, hy-4, earA)), plain(sabreGrey[8])) // the pale spot of the ear
	// The marks of the face: stripes back over the crown, two across the
	// cheek, the brow and the dark line down from the eye.
	dark := plain(sabreDark[1])
	for k := 0.0; k < 3; k++ {
		q.mark(R(aCap(hx+1.6-1.9*k, hy-5+0.25*k, hx-0.6-1.9*k, hy-2.9+0.5*k, 0.36, 0.26)), dark)
	}
	q.mark(R(aCap(hx-0.4, hy+0.4, hx-4.4, hy+1.5, 0.36, 0.26)), dark)
	q.mark(R(aCap(hx-0.6, hy+2.1, hx-3.8, hy+3.4, 0.33, 0.25)), dark)
	if jaw == 0 {
		q.flat(R(aCap(hx+4.2, hy+4, hx+8.4, hy+3.4, 0.24, 0.2)), plain(sabreGrey[1])) // the lip
	}
	fang(0, 0.08)
	for k := 0.0; k < 3; k++ { // where the whiskers grow
		wx, wy := RP(hx+5.6+1.1*k, hy+2.2-0.25*k)
		q.dot(wx, wy, 0, 0, sabreGrey[3])
	}
	ex, ey := RP(hx+2.8, hy-1.4)
	q.flat(R(aCap(hx+1, hy-2.7, hx+4.2, hy-2.3, 0.42, 0.3)), dark)
	q.flat(R(aCap(hx+3.8, hy-0.7, hx+4.9, hy+1.2, 0.3, 0.2)), dark)
	q.eye(ex, ey, 1.6, hex(0xf2b21e))
	nose := R(aEll(hx+9.1, hy+0.6, 1.1, 0.95))
	q.flat(nose, plain(hex(0x5a2a2a)))
	ay := 38.4
	if air {
		ay = 42.4
	}
	return q.done(0.8, 28.6, ay)
}

// --- the cave bear ---------------------------------------------------------------------

// bearHeadHD draws the bear's head at (hx, hy), turned by ha, the jaws gape open.
func bearHeadHD(q *crit, coat func(shape, float64) shader, hx, hy, ha, gape float64) {
	R := func(s shape) shape { return s.turned(hx, hy, ha) }
	RP := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ha) }
	muz := func(s shape, b float64) shader { return shd(bearMuzzle, s, 1.6, b) }
	if gape > 0 {
		lower := R(aCap(hx+1, hy+2.4, hx+6.4, hy+3+3.2*gape, 1.5, 1.1))
		q.fill(lower, muz(lower, -0.1))
		q.flat(R(aPoly(hx+1.5, hy+1.8, hx+7.6, hy+1.6, hx+6.2, hy+2.6+3*gape)), plain(hex(0x7a1c22)))
		for k := 0.0; k < 2; k++ {
			q.fill(R(aPoly(hx+4.4+1.6*k, hy+2.9+2.6*gape, hx+5+1.6*k, hy+2.9+2.6*gape, hx+4.8+1.6*k, hy+1.6+2.4*gape)), plain(colTooth))
		}
	}
	ear := R(aEll(hx-2.4, hy-4.3, 1.8, 1.8))
	q.fill(ear, coat(ear, -0.1))
	q.flat(R(aEll(hx-2.3, hy-4.2, 0.8, 0.9)), plain(bearRamp[1]))
	skull := R(aEll(hx, hy, 5.1, 4.7))
	q.fill(skull, coat(skull, 0.05))
	snout := R(aCap(hx+2.6, hy+0.9, hx+7.3, hy+1.2, 2.7, 2))
	q.fill(snout, muz(snout, 0.02))
	if gape > 0 {
		for k := 0.0; k < 2; k++ {
			q.fill(R(aPoly(hx+5+1.5*k, hy+2.8, hx+5.7+1.5*k, hy+2.8, hx+5.4+1.5*k, hy+4.3)), plain(colTooth))
		}
	} else {
		q.flat(R(aCap(hx+3.4, hy+2.4, hx+7.6, hy+2.3, 0.25, 0.2)), plain(bearRamp[0]))
	}
	q.flat(R(aEll(hx+9, hy+0.5, 1.15, 0.95)), plain(colInk))
	nx, ny := RP(hx+8.8, hy+0.2)
	q.dot(nx, ny, 0, 0, bearMuzzle[4])
	ex, ey := RP(hx+2.2, hy-1.4)
	q.bead(ex, ey, 1.6)
	q.flat(R(aCap(hx+1, hy-2.6, hx+3.4, hy-2.3, 0.3, 0.25)), plain(bearRamp[0]))
}

func bearHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(60, 53, 4, 7)
	const gy = 41.0
	coat := func(s shape, b float64) shader { return shag(bearRamp, s, 8, b-0.04, 9, 501) }
	paw := func(fx, fy, b float64) {
		pw := aCap(fx-0.4, fy-1.2, fx+2.6, fy-0.9, 2.1, 1.7)
		q.fill(pw, coat(pw, b))
		for k := 0.0; k < 3; k++ {
			q.fill(aPoly(fx+2.6+0.9*k, fy-1.7, fx+4.6+0.7*k, fy-0.1, fx+3+0.9*k, fy-0.2), shd(bearClawRamp, aEll(fx+3.5, fy-1, 2, 1.2), 1, 0.1))
		}
	}
	// An arm or a leg from (hx, hy) to a paw at (fx, fy).
	limb := func(hx, hy, fx, fy, l, r, b, bend float64) {
		kx, ky := ik2(hx, hy, fx, fy-1.6, l, l, bend)
		lg := aLeg(hx, hy, kx, ky, fx, fy-1.8, r, r*0.8, r*0.62)
		q.fill(lg, coat(lg, b))
	}
	if pose == bearClaw {
		// Reared up on its hind legs, roaring: a forepaw held high, then
		// brought down in a swipe.
		amt := blowAmt(f)
		limb(15, 30, 13, gy, 5.9, 3.7, -0.22, -1)
		paw(13, gy, -0.22)
		limb(24, 16, 33.5, 21.5, 5.6, 3.1, -0.24, -1)
		paw(33, 22.5, -0.24)
		body := sBlend(4, aPath(17, 30, 8.2, 18.5, 22, 8.4, 21, 14.5, 7.4), aEll(19.5, 12.5, 6.8, 5.6)).jag(4, 7, 30*cK, 505)
		chest := aEll(24.5, 23, 3.4, 7.5)
		q.fill(body, where(coat(body, 0), coat(body, 0.14), inside(chest)))
		limb(21, 31, 22, gy, 5.9, 3.9, 0.02, -1)
		paw(22, gy, 0.02)
		bearHeadHD(q, coat, 27.5, 10.5, -0.28-0.08*amt, 0.75)
		a := lerpf(-1.25, 0.35, smooth((amt+1)/2)) // the raised arm comes down
		hx, hy := 20.5+15*math.Cos(a), 13+15*math.Sin(a)
		kx, ky := ik2(20.5, 13, hx, hy, 7.8, 7.8, -1)
		arm := aLeg(20.5, 13, kx, ky, hx, hy, 3.7, 3.2, 2.7)
		q.fill(arm, coat(arm, 0.06))
		hand := aEll(hx+1, hy+0.3, 2.8, 2.5)
		q.fill(hand, coat(hand, 0.1))
		for k := -1.0; k <= 1; k++ { // the claws, spread
			ca := a + 0.5 + 0.35*k
			cx, cy := hx+1+2.6*math.Cos(ca), hy+0.3+2.6*math.Sin(ca)
			q.fill(aCap(cx, cy, cx+2.6*math.Cos(ca+0.3), cy+2.6*math.Sin(ca+0.3), 0.6, 0.2), plain(bearClawRamp.at(2-int(math.Abs(k)))))
		}
		return q.done(0.85, 18, gy)
	}
	g := gait{stride: 6, lift: 2.2, duty: 0.68, offs: walkOffs}
	hx, hy, ha, gape := 40.5, 24.5, 0.12, 0.0
	switch pose {
	case bearWalkA:
		g.on, g.ph = true, 0.05
		if f > 0 {
			g.ph = cycPh(f)
		}
	case bearWalkB:
		g.on, g.ph = true, 0.55
		if f > 0 { // a lumbering run
			g.ph, g.stride, g.lift, g.duty, g.offs = cycPh(f), 9, 3, 0.55, runOffs
		}
	case bearBite: // the head drawn back, then lunging, jaws wide
		t := (blowAmt(f) + 1) / 2
		hx, hy, ha, gape = lerpf(37.5, 42.5, t), lerpf(22.5, 27.5, t), lerpf(-0.1, 0.2, t), lerpf(0.15, 0.85, t)
	}
	bob := g.bob(0.9)
	hy += bob
	leg := func(i int, x, y, r, b float64) {
		dx, up := g.foot(i)
		bend := 1.0
		if i%2 == 0 {
			bend = -1
		}
		limb(x, y+bob, x+dx, gy-up, 4.9, r, b, bend)
		paw(x+dx, gy-up, b)
	}
	leg(2, 12, 30, 3.5, -0.24)
	leg(3, 30, 30, 3.4, -0.24)
	tl := aCap(6.5, 24+bob, 4.6, 26+bob, 1.7, 1.2)
	q.fill(tl, coat(tl, -0.1))
	body := sBlend(4, aEll(13, 26.5+bob, 8, 7.4), aEll(21.5, 26.5+bob, 11.5, 7.8), aEll(30, 23.5+bob, 8.2, 8), aCap(33, 23.5+bob, hx-2, hy, 6.4, 4.9)).jag(7, 8, (29+bob)*cK, 503)
	q.fill(body, coat(body, 0))
	leg(0, 16, 31, 3.8, 0.02)
	leg(1, 33, 31, 3.7, 0.04)
	bearHeadHD(q, coat, hx, hy, ha, gape)
	return q.done(0.85, 23, gy)
}

// --- the monkey ----------------------------------------------------------------------------

func monkeyHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(30, 27, 2, 2)
	coat := func(s shape, b float64) shader { return shag(monkeyRamp, s, 2.4, b, 4, 601) }
	face := func(s shape, b float64) shader { return shd(monkeyFace, s, 1.4, b) }
	limb := func(x0, y0, x1, y1, b, bend float64) {
		kx, ky := ik2(x0, y0, x1, y1, 3.3, 3.3, bend)
		lg := aLeg(x0, y0, kx, ky, x1, y1, 1.3, 1, 0.85)
		q.fill(lg, coat(lg, b))
		hand := aEll(x1+0.5, y1+0.1, 1.1, 0.75)
		q.fill(hand, face(hand, -0.1))
	}
	tail := func(pts [4][2]float64) {
		tb := newCTube(bez(12, 0.95, 0.6, pts[0], pts[1], pts[2], pts[3]))
		q.fill(tb.shape(), coat(tb.shape(), -0.12))
	}
	head := func(hx, hy float64) {
		ear := aEll(hx-3.7, hy-0.4, 1.4, 1.6)
		q.fill(ear, face(ear, 0))
		q.flat(aEll(hx-3.6, hy-0.3, 0.6, 0.8), plain(monkeyFace[0]))
		sk := aEll(hx, hy, 4.3, 4)
		q.fill(sk, coat(sk, 0.08))
		fc := sBlend(1, aEll(hx+1.2, hy-0.6, 2.4, 1.9), aEll(hx+1.9, hy+1.5, 2.3, 1.7))
		q.fill(fc, face(fc, 0.1))
		q.bead(hx+1.5, hy-0.7, 1.4)
		q.flat(aCap(hx+2.2, hy+2.1, hx+3.6, hy+1.9, 0.18, 0.18), plain(monkeyFace[0]))
		q.dot(hx+3.4, hy+0.7, 0, 0, monkeyFace[0])
	}
	switch pose {
	case monkeySit, monkeyEat:
		tail([4][2]float64{{10, 20}, {6, 22}, {3, 20}, {3.5, 16.5}})
		body := sBlend(2, aEll(12, 15, 4.6, 5.6), aEll(13, 20.3, 4.1, 2.1))
		chest := aEll(13.9, 16.4, 2.3, 3.3)
		q.fill(body, where(coat(body, 0.05), face(body, 0.05), inside(chest)))
		thigh := aEll(13.6, 19.8, 3, 2.2)
		q.fill(thigh, coat(thigh, 0.1))
		foot := aCap(15.6, 21.2, 18.2, 21.6, 0.95, 0.75)
		q.fill(foot, face(foot, -0.1))
		head(14, 8)
		if pose == monkeyEat {
			limb(14.6, 13.2, 17.4, 10.6, 0.02, 1)
			bn := aCap(18.2, 11, 19.2, 6.8, 0.9, 0.6)
			q.fill(bn, shd(bananaRamp, bn, 0.8, 0.1))
		} else {
			limb(14.4, 13.4, 16.2, 19.2, 0.02, 1)
		}
	case monkeyLeap:
		// A bound: the limbs flung out, gathered under it, flung out again.
		s := 1.0
		if f > 0 {
			s = math.Cos(2 * math.Pi * cycPh(f))
		}
		lag := s
		if f > 0 {
			lag = math.Cos(2*math.Pi*cycPh(f) - 0.9)
		}
		tail([4][2]float64{{6.5, 12}, {3, 11}, {1.5, 7 + lag}, {4, 4.5 + 1.5*lag}})
		limb(8.4, 13.4, 5.5-2.5*s, 17.5+1.5*(1-s), -0.2, -1)
		limb(15.6, 13.4, 17.5+2*s, 17.6+1.2*(1-s), -0.2, 1)
		body := aEll(12, 13, 6.4, 3.2+0.4*(1-s)).turned(12, 13, -0.08*s)
		q.fill(body, coat(body, 0.05))
		limb(7.4, 14, 4.5-2.5*s, 18+1.5*(1-s), 0.02, -1)
		limb(16.4, 14, 18.5+2*s, 18+1.2*(1-s), 0.02, 1)
		head(19, 9+0.5*(1-s))
	default: // on all fours
		g := gait{on: true, ph: 0.05, stride: 3.6, lift: 1.5, duty: 0.6, offs: trotOffs}
		if pose == monkeyWalkB {
			g.ph = 0.55
		}
		if f > 0 {
			g.ph = cycPh(f)
		}
		bob := g.bob(0.5)
		tail([4][2]float64{{7, 13 + bob}, {3.5, 12}, {1.5, 7}, {5, 4.5}})
		leg := func(i int, x, y, b float64) {
			dx, up := g.foot(i)
			bend := 1.0
			if i%2 == 0 {
				bend = -1
			}
			limb(x, y+bob, x+dx, 21.7-up, b, bend)
		}
		leg(2, 9.4, 16, -0.2)
		leg(3, 16.4, 16, -0.2)
		body := sBlend(2, aEll(10.5, 15.2+bob, 4.4, 3.7), aEll(14.5, 14.8+bob, 4.2, 3.4))
		q.fill(body, coat(body, 0.05))
		leg(0, 8.2, 16.6, 0.02)
		leg(1, 15.4, 16.6, 0.02)
		head(18, 11+bob)
	}
	return q.done(0.8, 13, 22)
}
