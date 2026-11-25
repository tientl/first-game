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
	set("mammoth", mammothHD, map[int]int{mammothWalkA: cycFrames, mammothCharge: cycFrames, mammothGore: blowFrames})
	set("sabre", sabreHD, map[int]int{sabreWalkA: cycFrames, sabreWalkB: cycFrames, sabreStalk: cycFrames, sabreSpring: blowFrames})
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
	ear := H(aEll(85.5, 32+bob, 4.2, 6.8))
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

var sabreCoat, sabrePale = finer(sabreFur), finer(sabreCream)

func sabreHD(pose, f int) (*canvas, float64, float64) {
	q := newCritAt(72, 48, 4, 6)
	const gy = 38.9
	g := gait{stride: 7.5, lift: 2.6, duty: 0.65, offs: walkOffs}
	// flex is the spine in a run: 1 gathered, the hind feet under the
	// belly, -1 stretched out.
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
		}
	case sabreStalk:
		crouch, hdx, hdy = 4.5, 3, 3.2
		if f > 0 {
			g.on, g.ph, g.stride, g.lift = true, cycPh(f), 5.5, 1.6
		}
	case sabreSpring:
		if amt := blowAmt(f); amt < 0 { // gathered on its haunches
			crouch, pitch, hdx, hdy, jaw = -6*amt, 0.16*amt, 1, -1.5*amt, 0.3
			flex = -0.8 * amt
		} else {
			air, reach, pitch, jaw = true, amt, -0.34*amt, 0.4+0.6*amt
			hdx, hdy, ha = 2, -1, -0.1
			flex = -amt
		}
	case sabreCower:
		crouch, hdx, hdy, ha, earsBack = 8.2, 1, 6, 0.2, true
	case sabreEatA, sabreEatB:
		crouch, pitch, hdx, hdy, ha = 2.5, 0.2, -1, 7.5, 0.75
		if pose == sabreEatB {
			jaw, hdy = 0.7, 6.5
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
	// Short dense fur in soft clumps, dappled darker over the back.
	spot := func(x, y int) float64 {
		ax, ay := float64(x)/cK, float64(y-py)/cK
		if ay < cy0+2.5 && vnoise(ax*0.6+ay*0.25, ay*0.95, 407) > 0.59 {
			return 0.26
		}
		return 0
	}
	var whole shape // the body, for the light on the muscles that lie on it
	coatAt := func(s shape, b float64, dx int, spots bool) shader {
		return locksAt(sabreCoat, func(x, y int) float64 {
			l := s.lum(x, y, 5*cK) + b + 0.03
			if spots && whole.d != nil {
				l = 0.4*l + 0.6*(whole.lum(x, y, 5*cK)+b+0.03)
			}
			if spots {
				l -= spot(x, y)
			}
			return l
		}, 3.2, 4.2, 0.3, 401, dx, py)
	}
	coat := func(s shape, b float64) shader { return coatAt(s, b, 0, false) }
	cream := func(s shape, b float64) shader {
		return locksAt(sabrePale, lit(s, 4, b+0.24), 3.2, 4.2, 0.3, 402, 0, py)
	}
	// pale is hi, turning to cream inside in: a soft, broken edge.
	pale := func(hi, lo shader, in shape) shader {
		return func(x, y int, d float64) color.RGBA {
			if in.d(float64(x)+0.5, float64(y)+0.5) < (hashf(x/2, y/2, 409)-0.6)*1.4*cK {
				return lo(x, y, d)
			}
			return hi(x, y, d)
		}
	}
	paw := func(fx, fy float64, b float64, claws bool, dx int) {
		pw := aCap(fx-0.1, fy-1.1, fx+2.4, fy-0.85, 1.55, 1.25)
		q.fill(pw, coatAt(pw, b+0.1, dx, false))
		q.flat(aCap(fx+2.2, fy-0.9, fx+2.5, fy-0.2, 0.22, 0.2), plain(sabreFur[1])) // between the toes
		if claws {
			for k := 0.0; k < 3; k++ {
				q.fill(aPoly(fx+2.6+0.5*k, fy-1.7+0.5*k, fx+4.5+0.3*k, fy-0.6+0.6*k, fx+2.9+0.4*k, fy-0.7+0.5*k), plain(colBone))
			}
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
				fx, fy = hx+7+4*reach, hy+5-3*reach
			}
		} else if !hind && pitch > 0.1 {
			fx += 3 // eating: the fore paws out in front
		}
		bend := 1.0
		if hind {
			bend = -1
		}
		kx, ky := ik2(hx, hy, fx, fy-1.2, 6.6, 6.6, bend)
		// A heavy forearm; a thigh that narrows to a thin hock.
		r0, r1, r2 := 4.3, 2.9, 2.1
		if hind {
			r0, r1, r2 = 4.6, 2.3, 1.75
		}
		lg := aLeg(hx, hy, kx, ky, fx, fy-1.7, r0, r1, r2)
		b := -0.22
		if near {
			b = 0
		}
		sl := int(math.Round((fx - hx0 + 1) * 0.6 * cK))
		q.fill(lg, coatAt(lg, b, sl, false))
		paw(fx, fy, b, air && !hind, sl)
	}
	leg(2, 19, 25, false)
	leg(3, 40.5, 25, false)
	// A bobbed tail.
	tx, ty := P(10.5+hipX, 21)
	tb := newCTube(bez(6, 1.9, 1.35, [2]float64{tx + 1, ty}, [2]float64{tx - 2, ty - 0.6}, [2]float64{tx - 4, ty - 1.4 + 2*clamp01(crouch/8)}, [2]float64{tx - 4.6, ty - 4 + 5*clamp01(crouch/8)}))
	q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
		if s > 0.72 {
			l -= 0.22 // its dark tip
		}
		return toneOf(sabreFur, l-0.02)
	}))

	// The body: a deep chest under heavy shoulders, the back sloping away
	// to lower, lighter hindquarters.
	hx, hy := P(46+hdx, 17+hdy)
	nkx, nky := P(38.5, 20.3)
	rump := T(aEll(17+hipX, cy+0.3, 7.5, 6.5))
	waist := T(aEll(27+0.5*hipX, cy+0.1-1.5*flex, 10.5, 5.2+0.5*flex))
	chest := T(aEll(37, cy-1.6, 8.6, 7.8))
	neck := aCap(nkx, nky, hx-1.5, hy+0.8, 6, 4.7)
	body := sBlend(4, rump, waist, chest, neck)
	belly := T(aEll(28+0.5*hipX, cy+5.7-0.8*flex, 13.5, 3)).or(aCap(nkx+3.5, nky+5.2, hx+0.5, hy+4.6, 2.2, 2))
	q.fill(body, pale(coatAt(body, 0, 0, true), cream(body, 0), belly))
	leg(0, 15.5, 26, true)
	leg(1, 37.5, 26.5, true)
	// The muscle of the haunch and the shoulder blade, which rolls as the
	// fore leg under it swings.
	q.edge, whole = 0.1, body
	haunch := T(aEll(16.6+hipX, cy+1.3, 5.7, 6))
	q.fill(haunch, coatAt(haunch, 0.01, 0, true))
	fdx, _ := g.foot(1)
	roll := -fdx / math.Max(1, g.stride)
	if !g.on {
		roll = 0
	}
	blade := T(aEll(37.4+1.6*roll, cy-2.4-1.3*roll-0.25*crouch+0.6*reach, 4.9, 6.2))
	q.fill(blade, coatAt(blade, 0.03, 0, true))
	q.edge = 0.34

	// The head: a heavy skull, a broad pale muzzle, and the two long fangs.
	R := func(s shape) shape { return s.turned(hx, hy, ha) }
	RP := func(x, y float64) (float64, float64) { return turnXY(x, y, hx, hy, ha) }
	fang := func(dx, b float64) {
		var c [4][2]float64
		for i, o := range [4][2]float64{{6.2, 2.4}, {7.4, 5.5}, {7.2, 8.4}, {5.4, 10.8}} {
			c[i][0], c[i][1] = RP(hx+o[0]+dx, hy+o[1])
		}
		tb := newCTube(bez(10, 1.35, 0.3, c[0], c[1], c[2], c[3]))
		q.fill(tb.shape(), tb.skin(func(x, y int, s, v, l float64) color.RGBA {
			return toneOf(tuskRamp, 0.42+0.75*l+b)
		}))
	}
	fang(-1.7, -0.3)
	if jaw > 0 {
		lower := R(aCap(hx+0.6, hy+3.6, hx+5.8, hy+4.8+3.6*jaw, 1.7, 1.25))
		q.fill(lower, cream(lower, -0.05))
		q.flat(R(aPoly(hx+1.2, hy+2.8, hx+7, hy+2.8, hx+5.6, hy+3.9+3.1*jaw)), plain(hex(0x8e2a30)))
	} else {
		chin := R(aCap(hx+1.5, hy+3.9, hx+5.6, hy+4.6, 1.5, 1.15))
		q.fill(chin, cream(chin, -0.08))
	}
	earA := -0.3
	if earsBack {
		earA = -1.5
	}
	ear := R(aEll(hx-1.8, hy-5.2, 1.7, 2.3).turned(hx-2, hy-3.6, earA))
	q.fill(ear, coat(ear, -0.14))
	skull := R(sBlend(2, aEll(hx, hy-0.2, 6.3, 5.7), aEll(hx+5.3, hy+1.4, 4.1, 3.5), aEll(hx-1.2, hy+2.8, 4.4, 3.6)))
	muzzle := R(aEll(hx+6, hy+3.1, 4.4, 2.7).or(aEll(hx+7.4, hy+1.6, 2.2, 2)))
	q.fill(skull, pale(coat(skull, 0.05), cream(skull, 0.02), muzzle))
	q.flat(R(aEll(hx-1.6, hy-4.9, 0.75, 1.25).turned(hx-2, hy-3.6, earA)), plain(sabreFur[1]))
	q.flat(R(aCap(hx+4.2, hy+3.9, hx+8.2, hy+3.3, 0.24, 0.2)), plain(sabreFur[1])) // the lip
	fang(0, 0.06)
	for k := 0.0; k < 3; k++ { // where the whiskers grow
		wx, wy := RP(hx+5.4+1.1*k, hy+2.3-0.25*k)
		q.dot(wx, wy, 0, 0, sabreFur[2])
	}
	ex, ey := RP(hx+2.6, hy-1.2)
	q.flat(R(aCap(hx+1, hy-2.5, hx+4, hy-2.2, 0.4, 0.3)), plain(sabreFur[0]))     // the brow
	q.flat(R(aCap(hx+3.6, hy-0.6, hx+4.6, hy+1.2, 0.3, 0.2)), plain(sabreFur[2])) // and the dark line down from the eye
	q.eye(ex, ey, 1.6, hex(0xf0b428))
	nose := R(aEll(hx+8.9, hy+0.7, 1.05, 0.9))
	q.flat(nose, plain(hex(0x4a2622)))
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
