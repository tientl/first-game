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
// oven's mouths. Each is a bed of flame tongues after pixelfx's fire/campfire
// and fire/torch (see fx.go): tongues side by side, the middle ones tallest,
// each leaning, leaping and wavering on its own clock, hottest low in the
// middle, in clean bands from a dark red rim through orange and yellow to a
// white heart. fireFrames frames make one loop that closes on itself; each
// is painted the first time it is shown.
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
	numFireKinds
)

// fireShape is how a kind of fire is built: its height and half-width in art
// pixels, and its tongues.
type fireShape struct {
	h, w    float64
	tongues [][4]float64 // across (of w), base (art pixels up), height (of h), half-width (art pixels)
}

// The campfire's tongues are pixelfx's eleven, a back row and a front row,
// the middle tallest; the others are fewer.
var campTongues = [][4]float64{
	{-0.6, 0.5, 0.42, 1.25}, {0.6, 0.5, 0.46, 1.25}, {-0.24, 0.8, 0.78, 1.4}, {0.26, 0.8, 0.82, 1.4},
	{-0.96, 0, 0.16, 0.9}, {1, 0, 0.18, 0.9}, {-0.76, 0.2, 0.32, 1.3}, {0.76, 0.2, 0.36, 1.3},
	{-0.4, 0.4, 0.6, 1.8}, {0.42, 0.4, 0.66, 1.8}, {0, 0.5, 1, 2.3},
}

var fireShapes = [numFireKinds]fireShape{
	fireCamp:    {h: 20, w: 7.5, tongues: campTongues},
	fireCampLow: {h: 11, w: 6, tongues: campTongues},
	fireBonfire: {h: 30, w: 11, tongues: append(append([][4]float64{}, campTongues...), [4]float64{-0.15, 0.6, 0.9, 2.6}, [4]float64{0.2, 0.7, 0.86, 2.4})},
	fireCook:    {h: 10, w: 6.5, tongues: [][4]float64{{-0.8, 0, 0.45, 1.2}, {0.8, 0, 0.5, 1.2}, {-0.4, 0.2, 0.8, 1.6}, {0.4, 0.2, 0.75, 1.6}, {0, 0.3, 1, 1.9}}},
	fireTorch:   {h: 10.5, w: 2.4, tongues: [][4]float64{{-0.45, 0, 0.6, 1.1}, {0.5, 0, 0.55, 1}, {0, 0.2, 1, 1.75}}},
	fireMouth:   {h: 6.5, w: 4.5, tongues: [][4]float64{{-0.7, 0, 0.6, 1.1}, {0.7, 0, 0.55, 1.1}, {-0.25, 0.1, 0.9, 1.3}, {0.3, 0.1, 1, 1.3}}},
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

// paintFire paints shape sh at loop phase ph, rooted at its bottom middle.
func paintFire(sh fireShape, ph float64, seed uint32) sprite {
	const K = HDDensity
	type tongue struct{ x, base, h, hw, sweep float64 }
	ts := make([]tongue, len(sh.tongues))
	lean := fireLean(ph)
	for i, t := range sh.tongues {
		o := float64(i)*1.7 + float64(seed%13)
		h := sh.h * t[2] * (0.94 + 0.12*hashf(i, int(seed), 601))
		leap := math.Max(0, loopWave(ph, 2+i%3, o+70)-0.55) * 1.4
		flick := fireFlicker(ph + float64(i)*0.071)
		h = (h + 0.22*h*loopWave(ph, 3+i%2, o) + 0.5*h*leap*0.5) * (0.9 + 0.1*flick)
		sweep := loopWave(ph, 2+(i+1)%3, o+40)*(0.6+0.05*h) + 0.9*h*lean - math.Max(-1.5, math.Min(1.5, 0.25*t[0]*sh.w))
		ts[i] = tongue{x: t[0] * sh.w, base: t[1], h: h, hw: t[3] * math.Max(1, sh.w/7.5*0.9), sweep: sweep * 0.55}
	}
	top := 0.0
	for _, t := range ts {
		top = math.Max(top, t.base+t.h*1.12)
	}
	wArt := 2*(sh.w+3.5) + 2
	hArt := top + 3
	W, H := int(math.Ceil(wArt*K)), int(math.Ceil(hArt*K))
	ox, oy := wArt/2, hArt-1 // the root on the canvas, art pixels
	heatAt := func(x, y float64) float64 {
		best := 0.0
		for i, t := range ts {
			v := (t.base - y) / t.h // 0 at its root, 1 at its tip
			if v < -0.15 || v > 1 {
				continue
			}
			vv := math.Max(v, 0)
			cx := t.x + t.sweep*math.Pow(vv, 1.6)
			w := t.hw * 1.3 * math.Pow(1-vv, 0.62) * math.Min(1, vv*3+0.75)
			if w <= 0.05 {
				continue
			}
			// Licks along the edge, on the loop's clock.
			wob := 0.12*math.Sin(y*1.1+2*math.Pi*ph*3+float64(i)) + 0.06*math.Sin(y*2.3-2*math.Pi*ph*5+float64(i)*2)
			d := math.Abs(x-cx)/w + wob*vv
			h := (1-math.Pow(math.Max(0, d), 1.4))*(1.02-vv*0.62) + 0.12*(1-vv)
			if v < 0 {
				h *= 1 + v/0.15 // rounded off below its root
			}
			best = math.Max(best, h+0.04*math.Max(0, best)) // where tongues meet, hotter
		}
		// The bed is rounded: thinner and cooler towards its ends.
		end := clamp01(1.35 - math.Abs(x)/(sh.w+1.2))
		return best * (0.55 + 0.45*end)
	}
	px := make([]color.RGBA, W*H)
	const ss = 3
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			n, hs := 0, 0.0
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					ax := (float64(x)+(float64(i)+0.5)/ss)/K - ox
					ay := (float64(y)+(float64(j)+0.5)/ss)/K - oy
					if h := heatAt(ax, ay); h > 0.06 {
						n++
						hs += h
					}
				}
			}
			if n == 0 {
				continue
			}
			heat := hs / float64(n)
			c := fireBand(heat)
			c.A = uint8(255 * float64(n) / (ss * ss))
			px[y*W+x] = c
		}
	}
	img := ebiten.NewImage(W, H)
	img.WritePixels(straightBytes(px))
	artBytes.Add(int64(4 * W * H))
	return spriteHD(img, ox, oy)
}

// fireBand is the colour of fire at heat h, in clean bands.
func fireBand(h float64) color.RGBA {
	steps := [...]float64{0.17, 0.33, 0.48, 0.61, 0.73, 0.85, 0.95}
	for i, s := range steps {
		if h < s {
			return palFire[i+1]
		}
	}
	return palFire[8]
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
			out = append(out, fireSource{sx, sy - 1, fireMouth, uint32(e.ID)})
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
		gw := sh.w * 2.2
		b.glowEll(f.x, f.y-0.5, gw*1.4*br, gw*0.5*br, palFire[4], (0.16+0.3*dark)*br)
		b.glowEll(f.x, f.y-0.5, gw*0.7, gw*0.25, palFire[6], (0.1+0.2*dark)*br)
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
