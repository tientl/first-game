package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Fire, at HD: the campfire, a fire low under a pot, the bonfire of a
// village, a torch on its stand and in the hand, the furnace's and the
// oven's mouths. Each is a few teardrop tongues off a bed of glowing coals
// (paintFire), kept small so the hearth under it shows (hearthart.go: the
// ring of stones, the crossed logs): the middle tongue tallest, each leaning
// and wavering on its own clock, its edge licked by noise rising through it,
// a wisp now and then breaking off a tip, in clean bands from a dark red rim
// through orange and amber to a pale yellow heart. fireFrames frames make
// one loop that closes on itself; each is painted the first time it is
// shown.
//
// Round the fire (drawFireFX, drawFireSmoke): embers rising and wandering,
// sparks popping out now and then, smoke rising and spreading on the wind,
// and the ground warm with its light. How bright it burns this frame
// (fireFlicker) is what the light of lighting.go flickers by, so the glow
// on the ground leaps with the flames.

// fireHD draws fire at HD; off, the old flames (shot -firehd=false).
var fireHD = true

// SetFireHD turns the HD fire on or off.
func SetFireHD(on bool) { fireHD = on }

// fireFrames is the frames of one loop of every flame; a frame shows for
// fireStep game frames.
const (
	fireFrames = 12
	fireStep   = 5
)

// fireKind is a size and shape of fire.
type fireKind uint8

const (
	fireCamp    fireKind = iota // a campfire burning well
	fireCampLow                 // a campfire burning low
	fireBonfire                 // a village's bonfire
	fireCook                    // low and wide under a pot, a grill or a pan
	fireTorch                   // a torch's flame, on its stand or in the hand
	fireMouth                   // in the mouth of a furnace or an oven
	fireHand                    // a torch's flame in the hand: the torch there is smaller than on its stand
	numFireKinds
)

// fireShape is how a kind of fire is built, in art pixels: the height of
// its tallest tongue and the half-width of the flames at their root, how
// many tongues burn side by side, and the half-width of the bed of coals
// glowing under them (0: none).
type fireShape struct {
	h, w  float64
	lobes int
	coals float64
}

// A campfire is kept small: a few tongues off a bed of coals, well inside its
// ring of stones, so the logs show round them.
var fireShapes = [numFireKinds]fireShape{
	fireCamp:    {h: 14, w: 4.4, lobes: 3, coals: 5.6},
	fireCampLow: {h: 6, w: 3.2, lobes: 3, coals: 5.6},
	fireBonfire: {h: 21, w: 7, lobes: 5, coals: 9},
	fireCook:    {h: 6.5, w: 4.4, lobes: 3, coals: 5.4},
	fireTorch:   {h: 8, w: 1.9, lobes: 2, coals: 1.7},
	fireMouth:   {h: 5.4, w: 3, lobes: 2, coals: 3.1},
	fireHand:    {h: 5.8, w: 1.5, lobes: 2, coals: 1.2},
}

// The fire's bands, coolest first: pixelfx's campfire palette.
var palFire = hexes(0x4a1712, 0x8a1f17, 0xc9281f, 0xf2561d, 0xff8a2a, 0xffc23d, 0xffe26b, 0xfff3a6, 0xffffff)

var fireSprites [numFireKinds][fireFrames]*sprite

// loopWave is a wave that comes round exactly once per loop at whole
// rates: ph in [0, 1), rate whole, off any.
func loopWave(ph float64, rate int, off float64) float64 {
	return math.Sin(2*math.Pi*(ph*float64(rate)) + off)
}

// fireLean is the lean of the whole fire at ph: a slow breath of air.
func fireLean(ph float64) float64 { return 0.12 * loopWave(ph, 1, 0.7) }

// fireFlicker is how bright a fire burns at loop phase ph: about 1, up to a
// tenth either way; the light round it follows it.
func fireFlicker(ph float64) float64 {
	return 0.74 + 0.16*loopWave(ph, 3, 1.1) + 0.1*loopWave(ph, 7, 2.3) + 0.26
}

// fireSprite is frame f of fire k, painted the first time.
func fireSprite(k fireKind, f int) sprite {
	f = ((f % fireFrames) + fireFrames) % fireFrames
	if s := fireSprites[k][f]; s != nil {
		return *s
	}
	s := paintFire(fireShapes[k], float64(f)/fireFrames, uint32(k)*97+5)
	fireSprites[k][f] = &s
	return s
}

// The flame's bands, coolest first: a dark red rim, through orange and
// amber, to a pale yellow heart (never white).
var palFlame = hexes(0x8a2414, 0xbf3a18, 0xe5621f, 0xf68f2b, 0xfcb93f, 0xffd96e, 0xfff0b0)

// The coals' tones: char, the dull red of a coal going grey, to the
// yellow of one breathing hot.
var palCoal = hexes(0x1c1210, 0x3a1410, 0x6a1c12, 0xa22e16, 0xd8561c, 0xf8902a, 0xffc85a)

// paintFire paints shape sh at loop phase ph, rooted at its bottom middle:
// the coals glowing first, breathing each on its own clock; then the
// tongues, each a teardrop leaning and wavering, its edge licked by noise
// that rises through the flame and comes round once a loop (pnoise tiles,
// so the frames close on themselves); now and then a wisp breaks off a tip
// and rises, shrinking. The bands go by heat, from a dark red rim to a pale
// yellow heart, and the edge is soft (each pixel is nine samples).
func paintFirePx(sh fireShape, ph float64, seed uint32) (px []color.RGBA, W, H int, ox, oy float64) {
	const K = HDDensity
	type lobe struct{ x, h, w, lean, wisp float64 }
	n := max(1, sh.lobes)
	ls := make([]lobe, n)
	mid := float64(n-1) / 2
	lean := fireLean(ph)
	for i := range ls {
		u := 0.0
		if mid > 0 {
			u = (float64(i) - mid) / mid // -1 to 1 across the bed
		}
		o := float64(i)*2.1 + float64(seed%17)
		tall := 1 - 0.42*math.Pow(math.Abs(u), 1.2)
		if n == 2 {
			tall = []float64{1, 0.7}[i]
		}
		h := sh.h * tall * (0.86 + 0.14*loopWave(ph, 2+i%2, o)) * (0.94 + 0.06*fireFlicker(ph+float64(i)*0.13))
		w := sh.w * (0.5 + 0.28*tall)
		if n == 1 {
			w = sh.w
		}
		x := u * sh.w * 0.62
		if n == 2 {
			x = []float64{-0.18, 0.42}[i] * sh.w
		}
		ls[i] = lobe{x: x, h: h, w: w, lean: lean + 0.08*loopWave(ph, 3, o+1), wisp: math.Mod(ph*float64(1+i%2)+hashf(i, 3, seed), 1)}
	}
	top := sh.h * 1.45
	half := math.Max(sh.w*1.5, sh.coals) + 2.5
	wArt, hArt := 2*half, top+4
	W, H = int(math.Ceil(wArt*K)), int(math.Ceil(hArt*K))
	ox, oy = wArt/2, hArt-2.2 // the root on the canvas, art pixels
	const P = 5                // the noise's period up the flame, cells: it rises P cells a loop
	heatAt := func(x, y float64) float64 {
		best := 0.0
		for i, l := range ls {
			v := -y / l.h // 0 at its root, 1 at its tip
			if v < -0.3 || v > 1.05 {
				continue
			}
			vv := math.Max(0, v)
			// Licked by noise rising through it, more so further up.
			ny := -y*0.42 + ph*P
			nx := pnoise(x*0.55+float64(i)*7.3, ny, 64, P, seed+uint32(i)) - 0.5
			cx := l.x + l.lean*l.h*vv*vv + 0.5*vv*math.Sin(2*math.Pi*ph*2+vv*3.5+float64(i)*1.9) + nx*2.2*vv
			hw := l.w * math.Pow(1-math.Min(vv, 0.999), 0.8) * (0.9 + 0.7*vv*(1-vv))
			if v < 0 {
				hw = l.w * (1 + v/0.3) // rounded off below its root
			}
			if hw <= 0.05 {
				continue
			}
			d := math.Abs(x-cx) / hw
			erode := (pnoise(x*0.9, ny*1.7, 64, P, seed+31+uint32(i)) - 0.5) * 0.9 * math.Pow(vv, 1.3)
			h := (1-math.Pow(d, 1.6))*(1.0-0.74*vv) + 0.06 - erode
			best = math.Max(best, h+0.05*math.Max(0, best)) // where tongues meet, hotter
			// A wisp broken off the tip, rising and shrinking.
			if l.wisp < 0.65 {
				wy := -l.h * (0.92 + 0.55*l.wisp)
				wx := l.x + l.lean*l.h*1.3 + nx*1.5
				r := l.w * 0.42 * (1 - l.wisp/0.65)
				if r > 0.25 {
					if dd := math.Hypot((x-wx)/r, (y-wy)/(r*1.5)); dd < 1 {
						best = math.Max(best, (1-dd)*0.42)
					}
				}
			}
		}
		return best
	}
	px = make([]color.RGBA, W*H)
	// The coals, under the flames: lumps in a low mound, cracked dark
	// between, each breathing on its own clock, hottest in the middle.
	if sh.coals > 0 {
		cr, cry := sh.coals, math.Max(0.9, sh.coals*0.34)
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				ax := (float64(x)+0.5)/K - ox
				ay := (float64(y)+0.5)/K - oy + 0.35
				r := math.Hypot(ax/cr, ay/cry)
				if r >= 1 {
					continue
				}
				// Lumps: a jittered grid of cells about an art pixel and a half across.
				gx, gy := (ax+cr)/1.5, (ay+cry)/1.05
				ix, iy := int(math.Floor(gx+0.5*hashf(int(gy), 7, seed))), int(math.Floor(gy))
				fx := gx + 0.5*hashf(int(gy), 7, seed) - float64(ix)
				fy := gy - float64(iy)
				edge := math.Min(math.Min(fx, 1-fx)*1.5, math.Min(fy, 1-fy))
				hot := 0.32 + 0.45*(1-r*r) + 0.25*hashf(ix, iy, seed+5) + 0.2*loopWave(ph, 1+int(hashf(ix, iy, seed+6)*3), hashf(ix, iy, seed+7)*6.3)
				if ay < -cry*0.2 {
					hot += 0.1 // the top of the mound, under the flames
				}
				if edge < 0.16 {
					hot -= 0.45 // a crack between lumps
				}
				t := clamp01(hot) * float64(len(palCoal)-1)
				i := int(t)
				if t-float64(i) > dither(x, y) {
					i++
				}
				c := palCoal[min(i, len(palCoal)-1)]
				if r > 0.86 {
					c.A = 200 // ash at the rim of the mound
				}
				px[y*W+x] = c
			}
		}
	}
	const ss = 3
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			cnt, hs := 0, 0.0
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					ax := (float64(x)+(float64(i)+0.5)/ss)/K - ox
					ay := (float64(y)+(float64(j)+0.5)/ss)/K - oy
					if h := heatAt(ax, ay); h > 0.07 {
						cnt++
						hs += h
					}
				}
			}
			if cnt == 0 {
				continue
			}
			heat := hs / float64(cnt)
			c := flameBand(heat)
			a := float64(cnt) / (ss * ss)
			if heat < 0.2 {
				a *= 0.8 // the rim thin, the ground or the coals showing through
			}
			if under := px[y*W+x]; under.A > 0 {
				c = mix(under, c, a)
				c.A = 255
			} else {
				c.A = uint8(255 * a)
			}
			px[y*W+x] = c
		}
	}
	return px, W, H, ox, oy
}

func paintFire(sh fireShape, ph float64, seed uint32) sprite {
	px, W, H, ox, oy := paintFirePx(sh, ph, seed)
	img := ebiten.NewImage(W, H)
	img.WritePixels(straightBytes(px))
	artBytes.Add(int64(4 * W * H))
	return spriteHD(img, ox, oy)
}

// flameBand is the colour of flame at heat h, in clean bands.
func flameBand(h float64) color.RGBA {
	steps := [...]float64{0.2, 0.34, 0.48, 0.62, 0.77, 0.92}
	for i, s := range steps {
		if h < s {
			return palFlame[i]
		}
	}
	return palFlame[6]
}

// drawFire draws frame f of fire k with its root at (x, y), art pixels.
func drawFire(dst drawTarget, k fireKind, f int, x, y float64) {
	s := fireSprite(k, f)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round((x-s.ax)*HDDensity)/HDDensity, math.Round((y-s.ay)*HDDensity)/HDDensity)
	drawSprite(dst, s, op)
}

// fireFrame is the frame a fire of id shows now: each fire on its own clock.
func (g *Game) fireFrame(id uint32) int { return (g.frame/fireStep + int(id)*5) % fireFrames }

// fireBright is how bright fire id burns now, for its light.
func (g *Game) fireBright(id uint32) float64 {
	ph := (float64(g.frame)/fireStep + float64(id)*5) / fireFrames
	return fireFlicker(ph - math.Floor(ph))
}

// campFire is the fire a campfire with fuel left burns: low when little is
// left.
func campFire(fuel int32) fireKind {
	if float64(fuel) < 0.35*sim.FireStartFuel {
		return fireCampLow
	}
	return fireCamp
}

// --- round the fire ------------------------------------------------------------------

// fireSource is a fire in view: where its flames are, how big, which.
type fireSource struct {
	x, y float64 // the root of its flames, art pixels
	k    fireKind
	id   uint32
}

// fireSources lists the fires burning in view, at their flames' roots.
func (g *Game) fireSources(cam camera) []fireSource {
	var out []fireSource
	for _, e := range g.curr.Entities {
		if e.Fuel <= 0 && e.Kind != sim.EntityBonfire {
			continue
		}
		wx, wy := float64(e.TX)+0.5, float64(e.TY)+0.5
		if g.hiddenInCave(wx, wy) {
			continue
		}
		sx, sy := cam.toScreen(wx, wy)
		switch e.Kind {
		case sim.EntityCampfire:
			k := campFire(e.Fuel)
			if sim.StationOf(e.Kind, e.Var) != sim.StationNone {
				k = fireCook
			}
			out = append(out, fireSource{sx, sy + 1, k, uint32(e.ID)})
		case sim.EntityBonfire:
			out = append(out, fireSource{sx, sy + 1, fireBonfire, uint32(e.ID)})
		case sim.EntityTorchStand:
			out = append(out, fireSource{sx, sy - g.art.tribal.torchStick.ay + 4, fireTorch, uint32(e.ID)})
		case sim.EntityFurnace:
			out = append(out, fireSource{sx, sy - 2.5, fireMouth, uint32(e.ID)})
		}
	}
	return out
}

// drawFireFX draws, over the dark of night, the warmth on the ground round
// each fire, the embers rising from it and the sparks it throws.
func (g *Game) drawFireFX(dst drawTarget, cam camera) {
	if !fireHD {
		return
	}
	g.drawFireFXOf(dst, g.fireSources(cam), g.darknessNow())
}

// drawFireFXOf draws the warmth, embers and sparks of fires fs, dark the
// dark of night (0 by day).
func (g *Game) drawFireFXOf(dst drawTarget, fs []fireSource, dark float64) {
	var b fxBatch
	t := float64(g.frame) / 60
	for _, f := range fs {
		sh := fireShapes[f.k]
		br := g.fireBright(f.id)
		// The ground lit warm.
		gw := math.Max(sh.w*1.6, sh.coals) * 2.2
		b.glowEll(f.x, f.y-0.5, gw*1.4*br, gw*0.5*br, palFire[4], (0.14+0.3*dark)*br)
		b.glowEll(f.x, f.y-0.5, gw*0.7, gw*0.25, palFire[6], (0.08+0.2*dark)*br)
		// A soft glow round the flames themselves, in the air.
		b.glow(f.x, f.y-sh.h*0.4, (sh.h*0.75+sh.w)*br, palFire[5], (0.07+0.16*dark)*br)
		// Embers: rising, wandering, fading.
		n := map[fireKind]int{fireCamp: 9, fireCampLow: 4, fireBonfire: 16, fireCook: 5, fireTorch: 3, fireMouth: 3}[f.k]
		for i := 0; i < n; i++ {
			seed := float64(f.id)*0.37 + float64(i)*1.618
			life := 1.1 + math.Mod(seed*3.7, 1)*1.4
			ph := math.Mod(t/life+seed, 1)
			x0 := f.x + (math.Mod(seed*7.3, 1)-0.5)*sh.w*1.1
			y0 := f.y - sh.h*0.3
			rise := (25 + 35*math.Mod(seed*5.1, 1)) * ph * life * (0.6 + sh.h/40)
			x := x0 + 7*(math.Sin(2.3*ph*life+seed)-math.Sin(seed))*0.6 + math.Cos(float64(g.curr.WindDir))*float64(g.curr.Wind)*ph*18
			y := y0 - rise + 2*(math.Sin(3*ph*life+seed*2)-math.Sin(seed*2))
			c := band(palFire[2:8], 1-ph)
			size := 0.55
			if i%7 == 0 {
				size = 0.85
			}
			b.dot(x, y, size, c, (1-ph)*(1-ph)*1.1, 0.35)
		}
		// Now and then a pop: a spark or three thrown up.
		if f.k == fireCamp || f.k == fireBonfire || f.k == fireMouth {
			period := 1.6
			if f.k == fireBonfire {
				period = 1.0
			}
			pt := math.Mod(t+float64(f.id%97)*0.31, period)
			r := fxRng{f.id*2654435761 + uint32(math.Floor((t+float64(f.id%97)*0.31)/period))*7919 + 1}
			px, py := f.x+r.in(-0.6, 0.6)*sh.w, f.y-r.in(0.2, 0.5)*sh.h
			if pt < 0.08 {
				b.glow(px, py, 6, palFire[6], 0.5*(1-pt/0.08))
			}
			for j := 0; j < 3+int(r.next()*3); j++ {
				a := -math.Pi/2 + r.in(-0.8, 0.8)
				v := r.in(60, 120) * math.Min(1.2, sh.h/18)
				life := r.in(0.25, 0.6)
				if pt >= life {
					continue
				}
				x, y, vx, vy := ballistic(px, py, math.Cos(a)*v, math.Sin(a)*v, 160, 1.2, pt)
				b.streak(x, y, vx, vy, 0.015, 0.6, band(palFire[3:8], 1-pt/life), 1, 0.4)
			}
		}
	}
	b.flush(dst)
}

// glowEll is soft light in an ellipse rx by ry about (x, y).
func (b *fxBatch) glowEll(x, y, rx, ry float64, c color.RGBA, a float64) {
	if a <= 0.003 || rx <= 0 {
		return
	}
	const n = 24
	ci := b.vert(x, y, c, a, 1)
	for i := 0; i <= n; i++ {
		th := 2 * math.Pi * float64(i) / n
		b.vert(x+math.Cos(th)*rx, y+math.Sin(th)*ry, c, 0, 1)
		if i > 0 {
			b.is = append(b.is, ci, ci+uint16(i), ci+uint16(i)+1)
		}
	}
}

var palSmoke = hexes(0x2c2a2e, 0x4a4650, 0x6e6a72, 0x96929a)

// drawFireSmoke draws the smoke going up from every fire, under the dark of
// night: soft grey puffs that rise, swell, drift with the wind and thin.
func (g *Game) drawFireSmoke(dst drawTarget, cam camera) {
	if !fireHD {
		return
	}
	g.drawFireSmokeOf(dst, g.fireSources(cam))
}

// drawFireSmokeOf draws the smoke of fires fs.
func (g *Game) drawFireSmokeOf(dst drawTarget, fs []fireSource) {
	var b fxBatch
	t := float64(g.frame) / 60
	wind := math.Cos(float64(g.curr.WindDir)) * (0.25 + float64(g.curr.Wind))
	for _, f := range fs {
		sh := fireShapes[f.k]
		n := map[fireKind]int{fireCamp: 7, fireCampLow: 5, fireBonfire: 10, fireCook: 4, fireTorch: 3, fireMouth: 5}[f.k]
		y0 := f.y - sh.h*0.8
		if f.k == fireMouth {
			y0 = f.y - 22 // out of the furnace's top
		}
		for i := 0; i < n; i++ {
			seed := float64(f.id)*0.731 + float64(i)*1.618
			life := 2.2 + math.Mod(seed*2.3, 1)*0.9
			ph := math.Mod(t/life+seed, 1)
			x := f.x + math.Sin(seed*9.1)*1.5 + math.Sin(ph*4+seed*3)*(2+3*ph)*ph + wind*ph*ph*24
			y := y0 - ph*(26+sh.h*0.8)
			rad := (1.6 + 4.5*ph) * math.Min(1.4, 0.6+sh.w/10)
			a := math.Sin(math.Min(1, ph*5)*math.Pi/2) * (1 - ph) * 0.42
			c := palSmoke[1+i%3]
			b.disc(x, y, rad*1.15, rad, c, a*0.55, 0)
			b.disc(x-rad*0.2, y-rad*0.2, rad*0.7, rad*0.6, palSmoke[2+i%2], a*0.35, 0)
		}
	}
	b.flush(dst)
}

// drawFireOf draws the flames of entity e, if it burns, at its place d;
// it reports whether it drew them (false: the old flames are wanted).
func (g *Game) drawFireOf(dst drawTarget, e protocol.EntityView, k fireKind, x, y float64) bool {
	if !fireHD {
		return false
	}
	drawFire(dst, k, g.fireFrame(uint32(e.ID)), x, y)
	return true
}
