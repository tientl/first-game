package client

import (
	"image/color"
	"math"
)

// Titanoboa, drawn at HDDensity: a body as thick as a tree trunk and a dozen
// tiles long, a tube along a curve (hdTube), deep jungle green with black
// saddles rimmed in gold, a golden belly of broad plates, every scale drawn
// and the light running over them in oil colours; a head painted like the
// other bosses' heads, with jaws that gape and long fangs. Its frames:
// slithering in a wave that travels down its body, asleep in a pile of
// coils, rearing to hiss, drawing back and striking, and squeezing its prey.

var (
	titanSkin  = mkRamp(0x0c2012, 0x15341a, 0x214c24, 0x326930, 0x48873e, 0x68a650, 0x8fc068)
	titanDark  = mkRamp(0x070b08, 0x0e150e, 0x172214, 0x22321c)
	titanBelly = mkRamp(0x7a5a16, 0xb08a26, 0xdcb83e, 0xf2da64, 0xfcf0a0)
	titanEye   = hex(0xff4a28)
	titanSheen = [...]color.RGBA{hex(0x3fe6c8), hex(0x4f9dff), hex(0xa66bff), hex(0xff8fd0), hex(0xffd36b)}
)

const (
	titanLen    = 340.0 // art pixels of body in a slither
	titanSaddle = 11.0  // art pixels from one saddle to the next
	titanHeadK  = 1.55  // art pixels to a unit of the head's drawing
)

// titanWidth is the body's half-width at s: 0 the tip of the tail, 1 the neck.
func titanWidth(s float64) float64 {
	return 2 + 10.8*smooth(clamp01(s/0.26)) - 3.4*smooth(clamp01((s-0.8)/0.2))
}

// titanSkinAt colours the body; slide moves its markings along it.
func titanSkinAt(slide float64) func(x, y int, q tubePt) color.RGBA {
	lx, ly := lightDir[0], lightDir[1]
	ll := math.Hypot(lx, ly)
	return func(x, y int, q tubePt) color.RGBA {
		t := max(-1, min(1, q.t))
		a := q.arc + slide
		l := lum(q.nx*t*0.92, q.ny*t*0.92)
		b := math.Asin(t) * q.r
		// The scales: a lattice of diamonds, each catching the light.
		pu, pv := a/2.75+b/2.3, a/2.75-b/2.3
		fu, fv := pu-math.Floor(pu), pv-math.Floor(pv)
		seam, shine := fu < 0.22 || fv < 0.22, fu > 0.55 && fv > 0.55
		down := t * q.ny      // how far towards the ground side of the body
		back := t + 0.22*q.ny // from the line of the spine
		toLight := t * (q.nx*lx + q.ny*ly) / ll
		sheen := math.Exp(-math.Pow((toLight-0.5)/0.3, 2))
		tone := func(r ramp, l float64, irid float64) color.RGBA {
			if seam {
				l -= 0.13
			} else if shine {
				l += 0.09
				irid *= 2.6
			}
			col := rq(r, l, x, y)
			if k := irid * sheen; k > 0.02 {
				h := a*0.045 + toLight*2.2 + hashf(int(math.Floor(pu)), int(math.Floor(pv)), 931)*0.9
				h -= math.Floor(h/5) * 5
				i := int(h)
				col = mix(col, mix(titanSheen[i%5], titanSheen[(i+1)%5], h-float64(i)), min(0.6, k))
			}
			return col
		}
		if down > 0.6+(hashf(x/2, y/2, 932)-0.5)*0.08 { // the belly's plates
			f := a / (titanSaddle / 3)
			if f-math.Floor(f) < 0.14 {
				l -= 0.2
			}
			return rq(titanBelly, l+0.14-(down-0.6)*0.5, x, y)
		}
		if q.s > 0.03 {
			f := a / titanSaddle
			at := math.Abs(f-math.Floor(f)-0.5) * 2 // 0 between two saddles, 1 in the middle of one
			d := (1 - at) + math.Abs(back)*1.05
			switch {
			case d < 0.15:
				return tone(titanSkin, l+0.16, 0.1)
			case d < 0.62:
				return tone(titanDark, l+0.1, 0.3)
			case d < 0.8:
				return rq(titanBelly, l*0.6+0.22, x, y) // the gold rim of each saddle
			}
			// A dark spot on the flank between two saddles.
			if d2 := math.Hypot(at*1.15, (down-0.36)*2.7); d2 < 0.3 {
				return tone(titanDark, l+0.12, 0.3)
			} else if d2 < 0.42 {
				return rq(titanBelly, l*0.6+0.16, x, y)
			}
		}
		return tone(titanSkin, l+0.03, 0.14)
	}
}

var (
	titanSkull = []knob{{-4, -1.5, 7.4}, {6, -2.6, 7.3}, {15, -1.5, 5.6}, {22.5, -0.3, 3.6}}
	titanJaw   = []knob{{-4, 3, 4.8}, {8, 3.3, 4}, {20, 2.6, 2.7}}
)

// frameQuad fills a line from (u0, v0) to (u1, v1) of the frame, w thick.
func frameQuad(c *canvas, f bossFrame, u0, v0, u1, v1, w float64, col color.RGBA) {
	dx, dy := u1-u0, v1-v0
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	f.poly(c, [][2]float64{{u0 + nx, v0 + ny}, {u1 + nx, v1 + ny}, {u1 - nx, v1 - ny}, {u0 - nx, v0 - ny}}, solid(col))
}

// titanHead paints the head with the back of its skull at (hx, hy) art
// pixels, looking along a; gape opens the jaws (0..1) and tongue (0..1) puts
// the tongue out.
func titanHead(c *canvas, hx, hy, a, gape, tongue float64) {
	fr := hdFrameAt(c, hx, hy, a-gape*0.42, titanHeadK)
	fj := hdFrameAt(c, hx, hy, a+gape*0.58, titanHeadK)
	if gape > 0 { // the inside of the mouth
		fr.poly2(c, fj, [][3]float64{{-3, 1, 0}, {23, 1.6, 0}, {20.5, 1.2, 1}, {-3, 2, 1}}, func(x, y int) color.RGBA {
			u, v := fr.local(x, y)
			d := math.Hypot(u, v)
			col := mix(colGullet, hex(0xe0707a), clamp01((d-3)/14)-hashf(x/2, y/2, 933)*0.12)
			if f := u / 2.6; f-math.Floor(f) < 0.16 && d > 6 {
				col = mix(col, colGullet, 0.25) // the ridges of its palate
			}
			return col
		})
	}
	if tongue > 0 {
		f, tu, tv := fj, 19.0, 0.4
		if gape < 0.1 {
			f, tu, tv = fr, 21.5, 1.5
		}
		e := 8 * tongue
		frameQuad(c, f, tu-6, tv, tu+e, tv-0.4, 0.95, colTongue)
		frameQuad(c, f, tu-6, tv+0.3, tu+e-2, tv, 0.5, hex(0x8c1c30))
		for _, dv := range []float64{-2.6, 1.9} {
			frameQuad(c, f, tu+e-0.3, tv-0.4, tu+e+4.2*tongue, tv-0.4+dv*tongue, 0.6, colTongue)
		}
	}
	bossPaint(c, 0.25, func(x, y int, q bossPt) color.RGBA {
		u, v := fj.local(x, y)
		if u > -1 && v < 1.3 {
			return color.RGBA{}
		}
		if u > 0 && v < 2.2 {
			if gape > 0.05 {
				return mix(hex(0xe89aa0), hex(0xb05868), clamp01((v-1.3)/0.9))
			}
			return titanDark[0]
		}
		l := 0.15 + q.l*0.8
		if f := u / 2.4; f-math.Floor(f) < 0.13 && v < 4.4 {
			l -= 0.16 // the scales of the lip
		}
		return rq(titanBelly, l, x, y)
	}, fj.knobs(titanJaw))
	bossPaint(c, 0.3, func(x, y int, q bossPt) color.RGBA {
		u, v := fr.local(x, y)
		if u > -1 && v > 1.7 {
			return color.RGBA{}
		}
		l := 0.12 + q.l*0.84
		switch {
		case v > 0.25: // the pale lip, with its heat pits
			f := u / 2.4
			ff := f - math.Floor(f)
			if ff < 0.13 {
				l -= 0.16
			}
			if u > 5 && u < 21 && math.Hypot((ff-0.56)*2.4, v-1) < 0.42 {
				return titanDark[1]
			}
			return rq(titanBelly, l+0.1, x, y)
		case v > -0.4 && u > 9.5:
			return rq(titanDark, l+0.2, x, y) // the line of the mouth
		case v > -2.9 && u < 9.6:
			return rq(titanDark, l+0.12, x, y) // the stripe behind the eye
		case v > -3.6 && u < 10:
			return rq(titanBelly, l*0.7+0.2, x, y)
		}
		// The scales of the crown: bigger than the body's, lighter on top.
		pu, pv := u/3.1+v/2.6, u/3.1-v/2.6
		fu, fv := pu-math.Floor(pu), pv-math.Floor(pv)
		l += -0.02*v - 0.04
		shine := fu > 0.5 && fv > 0.5
		if fu < 0.16 || fv < 0.16 {
			l -= 0.13
		} else if shine {
			l += 0.08
		}
		col := rq(titanSkin, l+0.05, x, y)
		// A dark arrow down the middle of the skull, rimmed in gold.
		if d := math.Abs(v+6.6) + max(0, u-3)*0.2; u > -3 && u < 15 && d < 1.3 {
			col = rq(titanDark, l+0.2, x, y)
		} else if u > -3 && u < 16 && d < 1.8 {
			col = rq(titanBelly, l*0.6+0.2, x, y)
		}
		// The same oil colours as the body, where the light falls.
		if k := (q.l - 0.6) * 2; k > 0 && shine {
			h := u*0.12 + v*0.3 + 5
			i := int(h)
			col = mix(col, mix(titanSheen[i%5], titanSheen[(i+1)%5], h-float64(i)), min(0.5, k))
		}
		return col
	}, fr.knobs(titanSkull))
	fang := func(x, y int) color.RGBA {
		if hashf(x/2, y, 934) < 0.25 {
			return hex(0xcfc8b0)
		}
		return colFang
	}
	if gape > 0.25 {
		g := min(1, gape)
		for _, u := range []float64{16.2, 20.4} { // the fangs, curved back
			fr.poly(c, [][2]float64{{u - 0.2, 1.5}, {u + 1.7, 1.5}, {u + 0.7, 1.6 + 4.6*g}, {u - 1.7, 1.6 + 8.6*g}, {u - 0.6, 1.6 + 4*g}}, fang)
		}
		for u := 3.6; u < 15; u += 2.3 {
			fr.tri(c, u-0.6, 1.6, u+0.6, 1.6, u-0.4, 3.9, fang)
			fj.tri(c, u+0.6, 1.4, u+1.8, 1.4, u+1.5, -0.9, fang)
		}
		fj.tri(c, 17.2, 1.3, 18.8, 1.3, 18.4, -2.6, fang)
	}
	hdEye(c, fr, 10.2, -3.7, 1.5, titanDark[0], titanEye, hex(0xffd070), true)
	frameQuad(c, fr, 7.4, -5.9, 12.6, -5.2, 0.9, titanDark[1]) // the brow
	frameQuad(c, fr, 7.6, -6.5, 12.2, -5.9, 0.5, titanSkin[5])
	nx, ny := fr.at(21.6, -1.7)
	c.dotPx(nx, ny, 0.55*fr.k1(), titanDark[0])
}

// titanSerp lays a body of length L along a serpenoid curve ending with its
// neck at (x1, y1), heading right; the head end barely sways.
func titanSerp(L, bend, waves, phase, squash, x1, y1 float64) []spinePt {
	const n = 1000
	us, vs := make([]float64, n+1), make([]float64, n+1)
	for i := 1; i <= n; i++ {
		s := float64(i) / n
		steady := 1 - 0.8*smooth(clamp01((s-0.7)/0.3))
		th := bend * steady * math.Cos(2*math.Pi*waves*s-phase)
		us[i] = us[i-1] + math.Cos(th)*L/n
		vs[i] = vs[i-1] + math.Sin(th)*L/n
	}
	// Keep it on the line of its travel: take out the drift of the whole wave.
	drift := (vs[n*7/10] - vs[0]) / (us[n*7/10] - us[0])
	pts := make([]spinePt, n+1)
	for i := range pts {
		s := float64(i) / n
		v := vs[i] - vs[n] - drift*(us[i]-us[n])*(1-smooth(clamp01((s-0.7)/0.3)))
		pts[i] = spinePt{x: x1 + us[i] - us[n], y: y1 + v*squash, r: titanWidth(s), s: s}
	}
	return pts
}

// titanCoil lays the body in a spiral pile about (cx, cy), winding in from
// radius outer to inner over turns and climbing rise as it goes; the last of
// it leaves the pile as a neck, by way of (mx, my) to (nx, ny).
func titanCoil(cx, cy, outer, inner, turns, rise, squash, mx, my, nx, ny float64) []spinePt {
	const n = 1500
	const coilEnd = 0.8
	spiral := func(s float64) (float64, float64) {
		ang := 0.5 + 2*math.Pi*turns*s
		rad := outer - (outer-inner)*s/coilEnd
		return cx + rad*math.Cos(ang), cy + rad*math.Sin(ang)*squash - rise*s/coilEnd
	}
	pts := make([]spinePt, 0, n+1)
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		var x, y float64
		if s < coilEnd {
			x, y = spiral(s)
		} else {
			u := (s - coilEnd) / (1 - coilEnd)
			x0, y0 := spiral(coilEnd)
			ax, ay := lerpf(x0, mx, u), lerpf(y0, my, u)
			bx, by := lerpf(mx, nx, u), lerpf(my, ny, u)
			x, y = lerpf(ax, bx, u), lerpf(ay, by, u)
		}
		pts = append(pts, spinePt{x: x, y: y, r: titanWidth(s), s: s})
	}
	return pts
}

// titanPose is one frame of the snake.
type titanPose struct {
	pts          []spinePt
	hx, hy, ha   float64 // the back of the skull, and where it looks
	gape, tongue float64
	slide        float64 // its markings moved along it
	ax, ay       float64 // the anchor
	shade        float64 // the body up to here (0..1 along it) lies on the ground
}

const titanW, titanH, titanCX, titanCY = 400, 260, 250.0, 190.0

func (p titanPose) canvas() (*canvas, float64, float64) {
	c := hdBossCanvas(titanW, titanH)
	hdGroundShadow(c, p.pts, 0, p.shade)
	body := hdBossCanvas(titanW, titanH)
	hdTube(body, p.pts, 34, titanSkinAt(p.slide))
	titanHead(body, p.hx, p.hy, p.ha, p.gape, p.tongue)
	body.outline(0.8)
	for i, q := range body.px {
		if q.A > 0 {
			c.px[i] = q
		}
	}
	return hdCrop(c, p.ax, p.ay)
}

// titanSlitherPose is the snake under way: ph is the phase of the wave.
func titanSlitherPose(ph float64) titanPose {
	const nx, ny = titanCX + 32, titanCY - 10
	pts := titanSerp(titanLen, 1.32, 1.9, ph, 0.56, nx, ny)
	return titanPose{pts: pts, hx: nx - 2, hy: ny - 1, ax: titanCX, ay: titanCY, shade: 1}
}

const (
	titanSlitherN = 12
	titanIdleN    = 8
	titanRoarN    = 8
	titanGrabN    = 8
)

// titanStrikeAges is the age, in frames at 60 a second, each frame of the
// strike shows: drawing back, held there, the lunge (the blow lands 36
// frames in), the bite, and back.
var titanStrikeAges = [...]int{0, 3, 6, 9, 14, 20, 26, 31, 33, 35, 37, 41, 45, 49, 53}

var (
	titanSlither = newBHAnim(titanSlitherN, func(i int) (*canvas, float64, float64) {
		p := titanSlitherPose(-2 * math.Pi * float64(i) / titanSlitherN)
		if i == 5 || i == 6 {
			p.tongue = 0.6 + 0.4*float64(i-5)
		}
		return p.canvas()
	})
	titanStrike = newBHAnim(len(titanStrikeAges), func(i int) (*canvas, float64, float64) {
		age := float64(titanStrikeAges[i])
		p := titanSlitherPose(0)
		rear := smooth(clamp01(age / 9))
		lunge := smooth(clamp01((age - 31) / 5))
		if age > 42 {
			lunge = 1 - smooth(clamp01((age-42)/12))
			rear = 0
		} else {
			rear *= 1 - 0.85*lunge
		}
		shake := 0.0
		if age > 9 && age < 31 {
			shake = math.Sin(age * 1.9)
		}
		for j := range p.pts {
			u := clamp01((p.pts[j].s - 0.56) / 0.44)
			q := &p.pts[j]
			q.y -= rear*(54*smooth(u)-6*u*u*u) + lunge*(10*smooth(u)-3*u*u) + shake*0.7*u
			q.x += -rear*50*u*u + rear*20*math.Sin(u*math.Pi) + lunge*42*u*u
		}
		last := p.pts[len(p.pts)-1]
		p.hx, p.hy = last.x-2, last.y-1
		p.ha = -0.5*rear + 0.3*lunge
		p.gape = 0.12 + 0.88*smooth(clamp01(age/14))
		switch {
		case age >= 42:
			p.gape = 0.3 * (1 - clamp01((age-42)/8))
		case age >= 36:
			p.gape = 0.3 // shut on its prey
		}
		if age >= 9 && age < 31 {
			p.tongue = 1
		}
		p.shade = 0.6 + 0.4*(1-max(rear, lunge))
		return p.canvas()
	})
	titanIdle = newBHAnim(titanIdleN, func(i int) (*canvas, float64, float64) {
		// Asleep: a pile of coils that swells as it breathes, the head laid on it.
		b := math.Sin(2 * math.Pi * float64(i) / titanIdleN)
		const cx, cy = titanCX - 20, titanCY - 30
		pts := titanCoil(cx, cy, 60+b*0.7, 22, 2.3, 17+b, 0.5, cx+34, cy-34, cx+52, cy-9+b*0.5)
		for j := range pts {
			pts[j].r *= 1 + 0.03*b*math.Sin(pts[j].s*math.Pi)
		}
		p := titanPose{pts: pts, hx: cx + 50, hy: cy - 10 + b*0.5, ha: 0.14, ax: cx, ay: cy + 39, shade: 0.42}
		if i == 2 || i == 3 {
			p.tongue = 0.5 + 0.5*float64(i-2)
		}
		return p.canvas()
	})
	titanRoar = newBHAnim(titanRoarN, func(i int) (*canvas, float64, float64) {
		// Reared out of its coils, its neck spread, hissing from side to side.
		w := 2 * math.Pi * float64(i) / titanRoarN
		sx := math.Sin(w) * 7
		const cx, cy = titanCX - 20, titanCY - 30
		pts := titanCoil(cx, cy, 54, 22, 1.8, 13, 0.5, cx-22+sx*0.4, cy-104, cx+8+sx, cy-112+math.Cos(w)*2)
		for j := range pts {
			pts[j].r *= 1 + 0.5*math.Sin(clamp01((pts[j].s-0.86)/0.13)*math.Pi)
		}
		return titanPose{pts: pts, hx: cx + 6 + sx, hy: cy - 112 + math.Cos(w)*2, ha: -0.12 + math.Sin(w)*0.1,
			gape: 0.92 + 0.08*math.Sin(2*w), tongue: 0.6 + 0.4*math.Sin(2*w), ax: cx, ay: cy + 36, shade: 0.5}.canvas()
	})
	titanGrab = newBHAnim(titanGrabN, func(i int) (*canvas, float64, float64) {
		// Wound tight, ring on ring, round whatever it has caught: the rings
		// draw in and ease, and slide round as it tightens.
		w := 2 * math.Pi * float64(i) / titanGrabN
		sq := 0.5 - 0.5*math.Cos(w)
		const cx, cy = titanCX - 20, titanCY - 26
		pts := titanCoil(cx, cy, 38-3.5*sq, 29-3*sq, 3.3, 62+3*sq, 0.36, cx+8, cy-102-2*sq, cx+33, cy-86+2*sq)
		for j := range pts {
			pts[j].r *= 1 + 0.06*sq*math.Sin(pts[j].s*math.Pi)
		}
		return titanPose{pts: pts, hx: cx + 31, hy: cy - 86 + 2*sq, ha: 0.55 + 0.08*math.Sin(w), gape: 0.4 + 0.25*sq, tongue: sq,
			slide: -titanSaddle * float64(i) / titanGrabN, ax: cx, ay: cy + 20, shade: 0.3}.canvas()
	})
)

// titanStrikeFrame is the frame of the strike age frames into it.
func titanStrikeFrame(age int) int {
	f := 0
	for i, a := range titanStrikeAges {
		if age >= a {
			f = i
		}
	}
	return f
}
