package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Combat and world effects, drawn at the canvas's own resolution (one image
// pixel to a screen pixel at 1080p, as the HD art): sparks, slashes, claw
// marks, splashes, dust and leaves. Their recipes follow pixelfx
// (github.com/patch-gg/pixelfx, MIT; see CREDITS.md and LICENSES/): the
// timings, shapes and palettes of its impacts/hit-spark, impacts/crit-hit,
// impacts/hit-flash, impacts/parry, melee/claw-swipe, melee/thrust,
// melee/spear-sweep, melee/heavy-cleave and water/splash, scaled from its
// pixels to our art pixels (a pixelfx figure is about two thirds of ours)
// and redrawn as banded vector shapes, a few flat tones each, like the
// painted art.
//
// Every effect is a pure function of its age and its seed, like the embers
// of a campfire: nothing is simulated, so an effect costs nothing until it
// is drawn and nothing is kept but where and when it began (fxInst).

// fxKind names an effect.
type fxKind uint8

const (
	fxSpark  fxKind = iota // impacts/hit-spark: a blow landing
	fxCrit                 // impacts/crit-hit: a critical blow
	fxLines                // impacts/hit-flash: the lines round a struck player
	fxClaw                 // melee/claw-swipe: a raptor's, a bear's, a sabre-tooth's, a monkey's blow
	fxThrust               // melee/thrust: a spear driven home
	fxSweep                // melee/spear-sweep: the ivory spear's wide low blow
	fxCleave               // melee/heavy-cleave: the battle axe brought down
	fxSplash               // water/splash
	fxDust                 // a puff of dust: a charge, a throw
	fxStomp                // a ring of dust: a stomp
	fxLeaves               // leaves knocked loose
	fxBits                 // what flies off a struck beast: red sparks and a little dust
	fxBlock                // a blow on a shield: chips and sparks
	fxParry                // a parry: a crossed flash and gold sparks
	fxStep                 // a footstep's dust on sand
	numFXKinds
)

// fxDef is how long an effect lasts, in frames at 60 a second, and whether
// it gives light (drawn over the dark of night) or lies on the ground
// (under the figures).
type fxDef struct {
	frames int
	glow   bool
	ground bool
}

var fxDefs = [numFXKinds]fxDef{
	fxSpark:  {30, true, false},
	fxCrit:   {54, true, false},
	fxLines:  {16, true, false},
	fxClaw:   {52, true, false},
	fxThrust: {34, true, false},
	fxSweep:  {40, true, false},
	fxCleave: {60, true, false},
	fxSplash: {90, false, false},
	fxDust:   {48, false, false},
	fxStomp:  {56, false, true},
	fxLeaves: {70, false, false},
	fxBits:   {36, true, false},
	fxBlock:  {30, true, false},
	fxParry:  {40, true, false},
	fxStep:   {30, false, true},
}

// fxInst is one effect in the world.
type fxInst struct {
	kind fxKind
	x, y float64 // where, tiles
	z    float64 // art pixels above the ground
	ang  float64 // the way the blow goes, on the screen (0 right, y down)
	at   int     // the frame it began
	seed uint32
	k    float64 // its size, 1 as made
	pal  int     // a colour variant: the ground of a dust puff, a blade's tier
}

// fxState is the effects playing.
type fxState struct {
	live []fxInst
	seq  uint32
}

// spawnFX starts an effect.
func (g *Game) spawnFX(f fxInst) {
	if f.k == 0 {
		f.k = 1
	}
	g.fx.seq++
	f.at, f.seed = g.frame, f.seed^(g.fx.seq*2654435761)
	g.fx.live = append(g.fx.live, f)
	if len(g.fx.live) > 160 { // a brawl: the oldest go first
		g.fx.live = g.fx.live[len(g.fx.live)-160:]
	}
}

// drawFX draws the effects of one pass: on the ground (under the figures),
// over the figures before the dark of night, or glowing over it.
func (g *Game) drawFX(dst drawTarget, cam camera, ground, glow bool) {
	if len(g.fx.live) == 0 {
		return
	}
	var b fxBatch
	kept := g.fx.live[:0]
	for _, f := range g.fx.live {
		d := fxDefs[f.kind]
		age := g.frame - f.at
		if age >= d.frames || age < 0 {
			continue
		}
		kept = append(kept, f)
		if d.ground != ground || (!ground && d.glow != glow) {
			continue
		}
		if g.hiddenInCave(f.x, f.y) {
			continue
		}
		sx, sy := cam.toScreenFine(f.x, f.y)
		drawFXAt(&b, f, sx, sy-f.z, sy, float64(age)/60)
		if len(b.vs) > 40000 {
			b.flush(dst)
		}
	}
	g.fx.live = kept
	b.flush(dst)
}

// drawFXAt draws effect f at (x, y), art pixels, t seconds in; gy is the
// ground under it.
func drawFXAt(b *fxBatch, f fxInst, x, y, gy, t float64) {
	r := fxRng{f.seed}
	switch f.kind {
	case fxSpark:
		fxHitSpark(b, &r, x, y, gy, f.ang, f.k, t)
	case fxCrit:
		fxCritHit(b, &r, x, y, gy, f.ang, f.k, t)
	case fxLines:
		fxHitLines(b, &r, x, y, f.k, t)
	case fxClaw:
		fxClawSwipe(b, &r, x, y, f.ang, f.k, t)
	case fxThrust:
		fxThrustFlare(b, &r, x, y, f.ang, f.k, t)
	case fxSweep:
		fxSpearSweep(b, &r, x, y, gy, f.ang, f.k, t)
	case fxCleave:
		fxHeavyCleave(b, &r, x, y, gy, f.ang, f.k, t)
	case fxSplash:
		fxWaterSplash(b, &r, x, y, f.k, t)
	case fxDust:
		fxDustPuff(b, &r, x, y, f.ang, f.k, f.pal, t, false)
	case fxStomp:
		fxStompRing(b, &r, x, y, f.k, f.pal, t)
	case fxLeaves:
		fxLeafFall(b, &r, x, y, gy, f.k, t)
	case fxBits:
		fxImpactBits(b, &r, x, y, gy, f.ang, f.k, f.pal, t)
	case fxBlock:
		fxShieldBlock(b, &r, x, y, f.ang, f.k, t)
	case fxParry:
		fxParryFlash(b, &r, x, y, f.ang, f.k, t)
	case fxStep:
		fxDustPuff(b, &r, x, y, f.ang, f.k, f.pal, t, true)
	}
}

// --- palettes ---------------------------------------------------------------

func hexes(v ...uint32) []color.RGBA {
	out := make([]color.RGBA, len(v))
	for i, h := range v {
		out[i] = hex(h)
	}
	return out
}

var (
	// impacts/hit-spark, dark to white.
	palSpark = hexes(0x4a1712, 0xc9281f, 0xf2561d, 0xff8a2a, 0xffe26b, 0xfff3a6, 0xffffff)
	// impacts/crit-hit.
	palCrit = hexes(0x2a0510, 0x5c0a1c, 0xa3142a, 0xe0302c, 0xff7a2a, 0xffc23d, 0xfff09a, 0xffffff)
	// melee/claw-swipe.
	palClaw = hexes(0x2a0808, 0x5c0f12, 0xa51d1d, 0xe2401f, 0xff8a2a, 0xffd166, 0xfff3c4, 0xffffff)
	// melee/sword-slash and thrust: a blade's cold light.
	palBlade = hexes(0x212f6b, 0x2c58aa, 0x3a8bd8, 0x62c8f4, 0xb8c9e8, 0xe8f1ff, 0xffffff)
	// heavy-cleave's embers.
	palEmber = hexes(0x3a1208, 0x7a240e, 0xc2401a, 0xf07a26, 0xffb347, 0xffe08a, 0xfff6d8)
	// water.
	palWater = hexes(0x0b3f5c, 0x145f84, 0x2a87ad, 0x5cb3cf, 0x9fd8e6, 0xd6f1f6, 0xffffff)
	// parry: gold.
	palGold = hexes(0x4a2a08, 0x8a5a12, 0xc98a1e, 0xf2b63a, 0xffd96b, 0xfff1b0, 0xffffff)
)

// Dust by the ground it rises from: earth, sand, snow, ash.
const (
	dustEarth = iota
	dustSand
	dustSnow
	dustAsh
	dustGrass
)

var dustPals = [...][3]color.RGBA{
	dustEarth: {hex(0x6e5236), hex(0x9c7a56), hex(0xc4a47c)},
	dustSand:  {hex(0xa88a58), hex(0xd2b27c), hex(0xeedcaa)},
	dustSnow:  {hex(0x9aa8b8), hex(0xcfd9e4), hex(0xf4f8fc)},
	dustAsh:   {hex(0x3e3a3c), hex(0x67605e), hex(0x8f8884)},
	dustGrass: {hex(0x6c6a48), hex(0x9a946a), hex(0xc2bc92)},
}

// --- a little random, from a seed ---------------------------------------------

type fxRng struct{ s uint32 }

func (r *fxRng) next() float64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 17
	r.s ^= r.s << 5
	r.s = r.s*1664525 + 1013904223
	return float64(r.s>>8) / (1 << 24)
}

func (r *fxRng) in(a, b float64) float64 { return a + (b-a)*r.next() }

// --- easing -----------------------------------------------------------------

func outExpo(x float64) float64 {
	if x >= 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*x)
}

func outCubic(x float64) float64 { x = 1 - clamp01(x); return 1 - x*x*x }

func inQuad(x float64) float64 { x = clamp01(x); return x * x }

func outBack(x float64) float64 {
	x = clamp01(x)
	const c1, c3 = 1.70158, 2.70158
	return 1 + c3*math.Pow(x-1, 3) + c1*math.Pow(x-1, 2)
}

func smoothstep(a, b, x float64) float64 {
	u := clamp01((x - a) / (b - a))
	return u * u * (3 - 2*u)
}

// ballistic is where a particle thrown from (x0, y0) at (vx, vy) art pixels a
// second is t seconds on, slowed by drag (per second) and pulled down by
// grav (art pixels a second squared), and its speed there.
func ballistic(x0, y0, vx, vy, grav, drag, t float64) (x, y, svx, svy float64) {
	if drag < 1e-6 {
		return x0 + vx*t, y0 + vy*t + grav*t*t/2, vx, vy + grav*t
	}
	e := math.Exp(-drag * t)
	f := (1 - e) / drag
	x = x0 + vx*f
	y = y0 + vy*f + grav/drag*(t-f)
	return x, y, vx * e, vy*e + grav/drag*(1-e)
}

// --- the triangle batch ----------------------------------------------------------

// fxBatch gathers triangles in art pixels, each vertex its own colour, and
// draws them at the canvas's resolution (drawTrianglesFine).
type fxBatch struct {
	vs []ebiten.Vertex
	is []uint16
}

// vert adds a vertex of colour c at opacity a; add, 0 to 1, makes it light
// added to what is beneath instead of laid over it.
func (b *fxBatch) vert(x, y float64, c color.RGBA, a, add float64) uint16 {
	a = clamp01(a)
	b.vs = append(b.vs, ebiten.Vertex{
		DstX: float32(x), DstY: float32(y),
		ColorR: float32(c.R) / 255 * float32(a), ColorG: float32(c.G) / 255 * float32(a), ColorB: float32(c.B) / 255 * float32(a),
		ColorA: float32(a * (1 - add)),
	})
	return uint16(len(b.vs) - 1)
}

func (b *fxBatch) flush(dst drawTarget) {
	if len(b.is) > 0 {
		drawTrianglesFine(dst, b.vs, b.is)
	}
	b.vs, b.is = b.vs[:0], b.is[:0]
}

// quad fills the quadrilateral p0 p1 p2 p3 (in order round it).
func (b *fxBatch) quad(x0, y0, x1, y1, x2, y2, x3, y3 float64, c color.RGBA, a, add float64) {
	if a <= 0.003 {
		return
	}
	i := b.vert(x0, y0, c, a, add)
	b.vert(x1, y1, c, a, add)
	b.vert(x2, y2, c, a, add)
	b.vert(x3, y3, c, a, add)
	b.is = append(b.is, i, i+1, i+2, i, i+2, i+3)
}

// taper is a stroke from (x0, y0), w0 wide, to (x1, y1), w1 wide.
func (b *fxBatch) taper(x0, y0, x1, y1, w0, w1 float64, c color.RGBA, a, add float64) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		return
	}
	nx, ny := -dy/l, dx/l
	b.quad(x0+nx*w0/2, y0+ny*w0/2, x1+nx*w1/2, y1+ny*w1/2, x1-nx*w1/2, y1-ny*w1/2, x0-nx*w0/2, y0-ny*w0/2, c, a, add)
}

// disc fills an ellipse.
func (b *fxBatch) disc(x, y, rx, ry float64, c color.RGBA, a, add float64) {
	if a <= 0.003 || rx <= 0 || ry <= 0 {
		return
	}
	n := max(8, min(28, int(math.Max(rx, ry)*3)))
	ci := b.vert(x, y, c, a, add)
	for i := 0; i <= n; i++ {
		th := 2 * math.Pi * float64(i) / float64(n)
		b.vert(x+math.Cos(th)*rx, y+math.Sin(th)*ry, c, a, add)
		if i > 0 {
			b.is = append(b.is, ci, ci+uint16(i), ci+uint16(i)+1)
		}
	}
}

// glow is soft light about (x, y): a at the middle, nothing at r.
func (b *fxBatch) glow(x, y, r float64, c color.RGBA, a float64) {
	if a <= 0.003 || r <= 0 {
		return
	}
	const n = 20
	ci := b.vert(x, y, c, a, 1)
	for i := 0; i <= n; i++ {
		th := 2 * math.Pi * float64(i) / n
		b.vert(x+math.Cos(th)*r, y+math.Sin(th)*r*0.85, c, 0, 1)
		if i > 0 {
			b.is = append(b.is, ci, ci+uint16(i), ci+uint16(i)+1)
		}
	}
}

// diamond fills a square stood on its point, r from the middle to a point.
func (b *fxBatch) diamond(x, y, r float64, c color.RGBA, a, add float64) {
	b.quad(x, y-r, x+r, y, x, y+r, x-r, y, c, a, add)
}

// ring is a band between radii r0 and r1 from angle a0 to a1, squashed
// sy down the screen.
func (b *fxBatch) ring(x, y, r0, r1, sy, a0, a1 float64, c color.RGBA, a, add float64) {
	if a <= 0.003 || r1 <= r0 {
		return
	}
	n := max(4, int(math.Abs(a1-a0)*r1*0.8))
	var p uint16
	for i := 0; i <= n; i++ {
		th := a0 + (a1-a0)*float64(i)/float64(n)
		cs, sn := math.Cos(th), math.Sin(th)
		q := b.vert(x+cs*r0, y+sn*r0*sy, c, a, add)
		b.vert(x+cs*r1, y+sn*r1*sy, c, a, add)
		if i > 0 {
			b.is = append(b.is, p, p+1, q, p+1, q+1, q)
		}
		p = q
	}
}

// streak is a particle drawn as a line from where it is back along its
// speed, thinning to its tail.
func (b *fxBatch) streak(x, y, vx, vy, k, w float64, c color.RGBA, a, add float64) {
	l := math.Hypot(vx, vy) * k
	if l < 0.6 {
		b.quad(x-w/2, y-w/2, x+w/2, y-w/2, x+w/2, y+w/2, x-w/2, y+w/2, c, a, add)
		return
	}
	b.taper(x, y, x-vx*k, y-vy*k, w, w*0.3, c, a, add)
}

// dot is a square particle w across.
func (b *fxBatch) dot(x, y, w float64, c color.RGBA, a, add float64) {
	b.quad(x-w/2, y-w/2, x+w/2, y-w/2, x+w/2, y+w/2, x-w/2, y+w/2, c, a, add)
}

// band picks a palette tone from heat, 0 (dark) to 1 (white).
func band(pal []color.RGBA, heat float64) color.RGBA {
	i := int(math.Round(clamp01(heat) * float64(len(pal)-1)))
	return pal[i]
}

// --- impacts/hit-spark -----------------------------------------------------------

// fxHitSpark: an everyday blow landing. A flare of five rays (the blow's
// line, a stub behind it, two arms across and a glint) round a white
// diamond, longer and thinner on the second beat, only the main ray left
// and burning down to red on the third; then a cone of sparks and hot chips
// thrown along the blow, a glow popping and gone.
func fxHitSpark(b *fxBatch, r *fxRng, x, y, gy, ang, k, t float64) {
	const sc = 0.62 // pixelfx pixels to our art pixels
	tilt := r.in(-0.08, 0.08)
	type ray struct{ a, l, w float64 }
	rays := [5]ray{
		{ang + tilt, r.in(33, 37), 2.2},
		{ang + math.Pi + tilt, r.in(8, 10), 2.2},
		{ang - math.Pi/2 + tilt, r.in(15, 18), 2.6},
		{ang + math.Pi/2 + tilt, r.in(10, 12), 2.3},
		{ang - math.Pi/4 + tilt, r.in(5, 7), 1},
	}
	pop := math.Exp(-t / 0.045)
	if pop > 0.02 {
		b.glow(x, y, 22*sc*k, palSpark[4], 0.3*pop)
		b.glow(x, y, 7*sc*k, palSpark[6], 0.15*pop)
	}
	drawRay := func(rr ray, lk, wk, from float64, cols [4]color.RGBA) {
		cs, sn := math.Cos(rr.a), math.Sin(rr.a)
		bands := [5]float64{0, 0.3, 0.62, 0.88, 1}
		for i := 0; i < 4; i++ {
			u0, u1 := math.Max(bands[i], from), bands[i+1]
			if u1 <= u0 {
				continue
			}
			l := rr.l * lk * sc * k
			w := rr.w * wk * sc * k
			w0, w1 := w*math.Pow(1-u0, 1.6), w*math.Pow(1-u1, 1.6)
			b.taper(x+cs*l*u0, y+sn*l*u0, x+cs*l*u1, y+sn*l*u1, w0*1.8+0.25, w1*1.8+0.15, cols[i], 1, 0)
		}
	}
	hot := [4]color.RGBA{palSpark[6], palSpark[5], palSpark[4], palSpark[3]}
	switch {
	case t < 0.034:
		for _, rr := range rays {
			drawRay(rr, 1, 1, 0, hot)
		}
		b.diamond(x, y, 5*sc*k, palSpark[3], 1, 0)
		b.diamond(x, y, 4*sc*k, palSpark[6], 1, 0)
	case t < 0.067:
		for _, rr := range rays {
			drawRay(rr, 1.2, 0.55, 0.12, hot)
		}
		b.diamond(x, y, 4*sc*k, palSpark[3], 1, 0)
		b.diamond(x, y, 3*sc*k, palSpark[6], 1, 0)
	case t < 0.1:
		drawRay(rays[0], 1.3, 0.6, 0.4, [4]color.RGBA{palSpark[1], palSpark[2], palSpark[3], palSpark[4]})
	}
	// The spark cone and the hot chips.
	for i := 0; i < 9; i++ {
		a := ang + r.in(-0.6, 0.6) - 0.15
		v := r.in(170, 360) * sc * k
		life := r.in(0.25, 0.45)
		if t >= life {
			continue
		}
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-40*sc, 420*sc, 3.2, t)
		if py > gy {
			continue
		}
		u := t / life
		b.streak(px, py, vx, vy, 0.022, 0.75, band(palSpark[2:], 1-u), 1, 0.3)
	}
	for i := 0; i < 6; i++ {
		a := ang + r.in(-1.1, 0.5) - 0.3
		v := r.in(70, 170) * sc * k
		life := r.in(0.35, 0.45)
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-r.in(30, 80)*sc, 520*sc, 1.5, t)
		py = math.Min(py, gy)
		c := []color.RGBA{palSpark[5], palSpark[4], palSpark[4], palSpark[3]}[i%4]
		b.dot(px, py, 0.9, c, 1-t/life*0.6, 0)
	}
}

// --- impacts/crit-hit ---------------------------------------------------------------

// fxCritHit: a critical blow. A jagged star of fourteen spikes stepping
// through four key frames, a ring that bursts out and breaks into arcs
// thrown forward, a lens flare across, and a spray of sparks, shards and
// embers.
func fxCritHit(b *fxBatch, r *fxRng, x, y, gy, ang, k, t float64) {
	const sc = 0.6
	type spike struct{ a, l, w float64 }
	var spikes [14]spike
	longest := [2]int{}
	for i := range spikes {
		a := ang + 2*math.Pi*float64(i)/14 + r.in(-0.12, 0.12)
		fwd := math.Cos(a - ang)
		l := r.in(11, 15) * (1 + 0.4*math.Max(0, fwd) - 0.4*math.Max(0, -fwd)*0.6) * (1 + r.in(-0.25, 0.25))
		spikes[i] = spike{a, l, r.in(1.8, 2.5)}
		if l > spikes[longest[0]].l {
			longest[1], longest[0] = longest[0], i
		} else if l > spikes[longest[1]].l {
			longest[1] = i
		}
	}
	spikes[longest[0]].l *= 1.3
	spikes[longest[1]].l *= 1.3
	star := func(scale, thick float64, rim, body, inner, core color.RGBA) {
		for pass, c := range [4]color.RGBA{rim, body, inner, core} {
			shrink := []float64{1, 0.78, 0.52, 0.3}[pass]
			for _, s := range spikes {
				l := s.l * scale * sc * k * shrink
				w := s.w * thick * sc * k * (1.6 - 0.25*float64(pass))
				cs, sn := math.Cos(s.a), math.Sin(s.a)
				nx, ny := -sn, cs
				b.quad(x+nx*w, y+ny*w, x+cs*l, y+sn*l, x-nx*w, y-ny*w, x, y, c, 1, 0)
			}
			b.disc(x, y, 2.2*scale*sc*k*shrink*thick+0.5, 2.2*scale*sc*k*shrink*thick+0.5, c, 1, 0)
		}
	}
	// The ring and its arcs.
	if t >= 0.03 && t < 0.4 {
		u := (t - 0.03) / 0.37
		rad := (10 + 30*outCubic(u)) * sc * k
		brk := smoothstep(0.12, 0.6, u)
		th := math.Max(1, 3.6*math.Pow(1-u, 1.2)) * sc * k
		ci := 7
		switch {
		case u < 0.12:
		case u < 0.3:
			ci = 6
		case u < 0.5:
			ci = 5
		case u < 0.75:
			ci = 4
		default:
			ci = 3
		}
		al := 1.0
		if u > 0.6 {
			al = (1 - u) / 0.4
		}
		if u < 0.6 {
			b.ring(x, y, rad-3*sc*k, rad+3*sc*k, 0.8, 0, 2*math.Pi, palCrit[5], 0.35*math.Pow(1-u/0.6, 2), 1)
		}
		for s := 0; s < 7; s++ {
			a := ang + 2*math.Pi*float64(s)/7
			gap := r.in(0.55, 0.85) * brk
			fwd := math.Max(0, math.Cos(a-ang))
			drift := (r.in(0.08, 0.4) + 0.15*fwd) * brk * rad
			span := 2*math.Pi/7 - gap
			wd := th * (0.7 + 0.5*fwd)
			cx, cy := x+math.Cos(a+span/2)*drift, y+math.Sin(a+span/2)*drift*0.8
			b.ring(cx, cy, rad-wd/2, rad+wd/2, 0.8, a, a+span, palCrit[max(2, ci-2)], al, 0)
			b.ring(cx, cy, rad-wd/4, rad+wd/4, 0.8, a+0.05, a+span-0.05, palCrit[ci], al, 0)
		}
	}
	// The lens flare.
	if t > 0.04 && t < 0.32 {
		f := (t - 0.04) / 0.28
		l := 64 * math.Pow(1-f, 1.3) * sc * k
		kk := 1 - f
		b.taper(x-l, y, x, y, 0.3, 1.6*sc*k, palCrit[6], kk, 0.4)
		b.taper(x+l, y, x, y, 0.3, 1.6*sc*k, palCrit[6], kk, 0.4)
		b.taper(x-l*0.35, y, x+l*0.35, y, 0.8*sc*k, 0.8*sc*k, palCrit[7], kk, 0.2)
		b.taper(x, y-0.45*l, x, y+0.45*l, 0.3, 0.3, palCrit[5], kk, 0.4)
		for _, d := range [2]float64{1, -1} {
			b.taper(x-0.16*l*0.7, y-0.16*l*0.7*d, x+0.16*l*0.7, y+0.16*l*0.7*d, 0.6, 0.6, palCrit[5], kk, 0.4)
		}
		b.glow(x, y, 56*sc*k, palCrit[5], 0.4*(1-f))
	}
	// The star, key frame by key frame.
	switch {
	case t < 0.025:
		star(1, 1, palCrit[6], palCrit[7], palCrit[7], palCrit[7])
	case t < 0.058:
		star(1.3, 1, palCrit[2], palCrit[5], palCrit[6], palCrit[7])
	case t < 0.092:
		star(1.05, 0.72, palCrit[2], palCrit[4], palCrit[5], palCrit[6])
	case t < 0.125:
		star(1.22, 0.3, palCrit[1], palCrit[2], palCrit[3], palCrit[4])
	}
	pop := math.Exp(-t / 0.05)
	if pop > 0.02 {
		b.glow(x, y, 44*sc*k, palCrit[5], 0.4*pop)
		b.glow(x, y, 12*sc*k, palCrit[7], 0.6*pop)
	}
	// Sparks, shards, embers, glints.
	for i := 0; i < 36; i++ {
		back := r.next() < 0.22
		a, v := ang+r.in(-0.8, 0.8), r.in(170, 430)
		if back {
			a, v = ang+math.Pi+r.in(-0.6, 0.6), v*0.6
		}
		delay := 0.0
		if r.next() < 0.3 {
			delay = r.in(0, 0.04)
		}
		life := r.in(0.15, 0.5)
		tt := t - delay
		if tt < 0 || tt >= life {
			continue
		}
		v *= sc * k
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-50*sc, 420*sc, 3.4, tt)
		if py > gy {
			continue
		}
		b.streak(px, py, vx, vy, 0.02, 0.8, band([]color.RGBA{palCrit[3], palCrit[5], palCrit[6], palCrit[7]}, 1-tt/life), 1, 0.3)
	}
	for i := 0; i < 10; i++ {
		a := ang + r.in(-1, 1)*1.1 - 0.35
		v := r.in(110, 280) * sc * k
		life := r.in(0.55, 0.85)
		spin := r.in(-25, 25)
		a0 := r.in(0, 2*math.Pi)
		size := r.in(2, 4.5) * sc * k * 0.7
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-40*sc, 800*sc, 0.7, t)
		if py > gy {
			py = gy - (py-gy)*0.15 // bounced, settling
		}
		th := a0 + spin*t
		cs, sn := math.Cos(th)*size, math.Sin(th)*size
		c := []color.RGBA{palCrit[7], palCrit[6], palCrit[6], palCrit[5]}[i%4]
		b.quad(px+cs, py+sn, px-sn*0.4, py+cs*0.4, px-cs, py-sn, px+sn*0.4, py-cs*0.4, c, 1-t/life, 0)
	}
	for i := 0; i < 12; i++ {
		a := ang - math.Pi/2 + r.in(-1.4, 1.4)
		v := r.in(35, 110) * sc * k
		life := r.in(0.35, 0.7)
		delay := r.in(0, 0.05)
		tt := t - delay
		if tt < 0 || tt >= life {
			continue
		}
		px, py, _, _ := ballistic(x+math.Cos(a)*2, y+math.Sin(a)*2, math.Cos(a)*v, math.Sin(a)*v-15*sc, -20*sc, 1.2, tt)
		b.dot(px, py, 0.7, []color.RGBA{palCrit[6], palCrit[5], palCrit[5]}[i%3], 1-tt/life, 0.4)
	}
	for i := 0; i < 5; i++ {
		a := r.in(0, 2*math.Pi)
		d := r.in(14, 34) * sc * k
		life, delay := r.in(0.22, 0.38), r.in(0.08, 0.32)
		size := r.in(1.6, 3) * sc * k
		tt := t - delay
		if tt < 0 || tt >= life {
			continue
		}
		u := tt / life
		s := size * math.Sin(u*math.Pi)
		px, py := x+math.Cos(a)*d, y+math.Sin(a)*d*0.8-4*sc*tt
		c := palCrit[6]
		if i%2 == 0 {
			c = palCrit[7]
		}
		b.taper(px-s, py, px+s, py, 0.2, 0.2, c, 1, 0.3)
		b.taper(px, py-s, px, py+s, 0.2, 0.2, c, 1, 0.3)
		b.taper(px-s, py, px, py, 0.1, 0.7, c, 1, 0.3)
		b.taper(px+s, py, px, py, 0.1, 0.7, c, 1, 0.3)
		b.taper(px, py-s, px, py, 0.1, 0.7, c, 1, 0.3)
		b.taper(px, py+s, px, py, 0.1, 0.7, c, 1, 0.3)
	}
}

// --- impacts/hit-flash -----------------------------------------------------------

// fxHitLines: the lines that burst out round a struck player, white, then
// red, flying apart. (The flash on the figure itself is drawn with it: see
// jolt.go and feel.go.)
func fxHitLines(b *fxBatch, r *fxRng, x, y, k, t float64) {
	u := t / (16.0 / 60)
	c := palSpark[6]
	if t > 0.05 {
		c = palSpark[1]
	}
	for i := 0; i < 7; i++ {
		a := 2*math.Pi*float64(i)/7 + r.in(-0.3, 0.3)
		r0 := (5 + 9*outCubic(u)) * k
		l := (5 + r.in(0, 3)) * (1 - u) * k
		cs, sn := math.Cos(a), math.Sin(a)*0.85
		b.taper(x+cs*r0, y+sn*r0, x+cs*(r0+l), y+sn*(r0+l), 1.1*k*(1-u*0.5), 0.2, c, 1-u*u, 0)
	}
}

// --- melee/claw-swipe ----------------------------------------------------------------

// fxClawSwipe: three marks raked one after another, torn open white-hot in a
// twentieth of a second, cooling through orange to red, closing from both
// ends and leaving a fading dotted scar, with hot flecks and motes thrown
// off them. ang is the way the claws rake.
func fxClawSwipe(b *fxBatch, r *fxRng, x, y, ang, k, t float64) {
	const rake, open, cool = 0.05, 0.035, 0.42
	cs, sn := math.Cos(ang), math.Sin(ang)
	nx, ny := -sn, cs
	S := 17 * k
	for i := 0; i < 3; i++ {
		t0 := 0.04 * float64(i)
		u := t - t0
		if u < 0 {
			continue
		}
		off := (float64(i) - 1) * 3.2 * k
		ox, oy := x+nx*off-cs*S/2+cs*float64(i)*1.2*k, y+ny*off-sn*S/2+sn*float64(i)*1.2*k
		bow := 0.12 * S * (1 - 0.15*float64(i))
		pt := func(v float64) (float64, float64) { // along the mark, bowed
			s := v * S
			b := bow * 4 * v * (1 - v)
			return ox + cs*s + nx*b, oy + sn*s + ny*b
		}
		head := outExpo(math.Min(1, u/rake))
		heat := clamp01(1 - u/cool)
		c := smoothstep(0.16, 0.46, u)
		lo, hi := 0.5*c, math.Min(head, 1-0.5*c)
		W := []float64{2.1, 1.9, 1.7}[i] * k
		if hi > lo {
			const n = 18
			for pass := 0; pass < 3; pass++ {
				wk := []float64{1, 0.62, 0.3}[pass]
				ci := int(math.Round((1-heat)*5.2 + float64(2-pass)*1.6*0.6))
				col := palClaw[7-max(0, min(6, ci))]
				for j := 0; j < n; j++ {
					v0 := lo + (hi-lo)*float64(j)/n
					v1 := lo + (hi-lo)*float64(j+1)/n
					w0 := W * math.Pow(math.Sin(math.Pi*v0), 1.15) * (1 - 0.55*c) * wk
					w1 := W * math.Pow(math.Sin(math.Pi*v1), 1.15) * (1 - 0.55*c) * wk
					x0, y0 := pt(v0)
					x1, y1 := pt(v1)
					b.taper(x0, y0, x1, y1, w0*2, w1*2, col, 1, 0)
				}
			}
			mx, my := pt((lo + hi) / 2)
			b.glow(mx, my, 7*k, palClaw[4], 0.42*heat*heat)
		}
		// The scar.
		if c > 0.05 {
			al := 0.8 * (1 - clamp01((u-0.31)/0.4))
			for v := 0.04; v < 0.96; v += 0.07 {
				if v > lo && v < 1-lo {
					continue
				}
				px, py := pt(v)
				b.dot(px, py, 0.7*k, palClaw[2], al, 0)
			}
		}
		// What flies off it, once it is open.
		if u >= open {
			tu := u - open
			tx, ty := pt(1)
			for j := 0; j < 7; j++ {
				a := ang + r.in(-0.5, 0.5)
				v := r.in(110, 260) * 0.6 * k
				life := r.in(0.12, 0.32)
				if tu >= life {
					continue
				}
				px, py, vx, vy := ballistic(tx, ty, math.Cos(a)*v, math.Sin(a)*v, 300, 2.5, tu)
				b.streak(px, py, vx, vy, 0.02, 0.7, band([]color.RGBA{palClaw[3], palClaw[4], palClaw[5], palClaw[6]}, 1-tu/life), 1, 0.3)
			}
			for j := 0; j < 5; j++ {
				px0, py0 := pt(r.in(0.15, 0.9))
				life := r.in(0.2, 0.4)
				vx, vy := r.in(-60, -10)*0.6*cs, r.in(-30, 30)*0.6
				if tu >= life {
					continue
				}
				col := palClaw[5]
				if j%2 == 0 {
					col = palClaw[4]
				}
				b.dot(px0+vx*tu, py0+vy*tu, 0.6, col, 1-tu/life, 0.3)
			}
			for j := 0; j < 2; j++ { // small red sparks falling
				px0, py0 := pt(r.in(0.35, 0.75))
				life := 0.7
				if tu >= life {
					continue
				}
				px, py, _, _ := ballistic(px0, py0+1, r.in(-8, 8)*0.6, r.in(10, 40)*0.6, 260, 0.5, tu)
				b.dot(px, py, 0.8, palClaw[3], 1-tu/life, 0)
			}
		}
		if u >= 0.02 && u < 0.16 {
			gg := u - 0.02
			rad := 5 * outBack(math.Min(1, gg/0.04)) * (1 - gg/0.16) * 0.6 * k
			tx, ty := pt(math.Min(head, 1))
			b.taper(tx-rad, ty, tx+rad, ty, 0.3, 0.3, palClaw[6], 0.9, 0.4)
			b.taper(tx, ty-rad, tx, ty+rad, 0.3, 0.3, palClaw[6], 0.9, 0.4)
			b.glow(tx, ty, rad*2.2, palClaw[4], 0.4)
		}
	}
}

// --- melee/thrust ------------------------------------------------------------------

// fxThrustFlare: a spear driven home. A narrow streak shoots along the
// thrust to the point, and a four-pointed star flares there and is gone,
// with a few sparks thrown on.
func fxThrustFlare(b *fxBatch, r *fxRng, x, y, ang, k, t float64) {
	cs, sn := math.Cos(ang), math.Sin(ang)
	head := outExpo(t / 0.05)
	fade := clamp01(1 - (t-0.06)/0.22)
	l := 16 * k * head
	if fade > 0 {
		tail := l * (0.3 + 0.7*smoothstep(0.05, 0.25, t))
		b.taper(x-cs*l, y-sn*l, x-cs*(l-tail)*0, y-sn*(l-tail)*0, 0.2, 2.4*k*fade, palBlade[3], fade*0.8, 0.3)
		b.taper(x-cs*l*0.8, y-sn*l*0.8, x, y, 0.2, 1.3*k*fade, palBlade[5], fade, 0.1)
		b.taper(x-cs*l*0.5, y-sn*l*0.5, x, y, 0.1, 0.6*k*fade, palBlade[6], fade, 0)
	}
	if t > 0.03 && t < 0.25 {
		g := t - 0.03
		s := 8 * outBack(math.Min(1, g/0.05)) * (1 - g/0.22) * k
		nx, ny := -sn, cs
		b.glow(x, y, s*2.5, palBlade[3], 0.5*(1-g/0.22))
		b.taper(x-cs*s*1.5, y-sn*s*1.5, x, y, 0.1, 1.2*k, palBlade[6], 1, 0.2)
		b.taper(x+cs*s*1.5, y+sn*s*1.5, x, y, 0.1, 1.2*k, palBlade[6], 1, 0.2)
		b.taper(x-nx*s*0.7, y-ny*s*0.7, x, y, 0.1, 1*k, palBlade[5], 1, 0.2)
		b.taper(x+nx*s*0.7, y+ny*s*0.7, x, y, 0.1, 1*k, palBlade[5], 1, 0.2)
		b.diamond(x, y, 1.2*k*(1-g/0.22), palBlade[6], 1, 0)
	}
	for i := 0; i < 8; i++ {
		a := ang + r.in(-0.45, 0.45)
		v := r.in(120, 280) * 0.6 * k
		life := r.in(0.12, 0.3)
		if t < 0.04 || t-0.04 >= life {
			continue
		}
		tt := t - 0.04
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v, 250, 3, tt)
		b.streak(px, py, vx, vy, 0.02, 0.6, band(palBlade[3:], 1-tt/life), 1, 0.3)
	}
}

// --- melee/spear-sweep ---------------------------------------------------------------

// fxSpearSweep: a long spear swept low and wide in front, a flat crescent
// along the ground that kicks up dust where it passes.
func fxSpearSweep(b *fxBatch, r *fxRng, x, y, gy, ang, k, t float64) {
	side := 1.0
	if math.Cos(ang) < 0 {
		side = -1
	}
	R := 17 * k
	const sy = 0.42
	a0, a1 := -2.4, 0.6 // from behind, round the front
	if side < 0 {
		a0, a1 = math.Pi-a0, math.Pi-a1
	}
	head := outExpo(t / 0.08)
	tail := smoothstep(0.1, 0.4, t)
	cx, cy := x, gy-2*k
	if tail < 1 {
		const n = 26
		for i := 0; i < n; i++ {
			v0, v1 := float64(i)/n, float64(i+1)/n
			if v1 > head || v0 < tail {
				continue
			}
			th0, th1 := a0+(a1-a0)*v0, a0+(a1-a0)*v1
			w := 3.4 * k * math.Pow(math.Sin(math.Pi*math.Min(1, (v0-tail)/math.Max(0.01, head-tail))), 0.6)
			for pass, c := range [3]color.RGBA{palBlade[3], palBlade[5], palBlade[6]} {
				ww := w * []float64{1, 0.55, 0.22}[pass]
				b.ring(cx, cy, R-ww, R, sy, th0, th1, c, (1-tail)*math.Min(1, v0/0.3+0.2), 0)
			}
		}
	}
	// Dust kicked up along the ground where it passed.
	for i := 0; i < 7; i++ {
		v := 0.35 + 0.6*float64(i)/6
		start := v * 0.08
		tt := t - start
		if tt < 0 {
			continue
		}
		th := a0 + (a1-a0)*v
		px, py := cx+math.Cos(th)*R, cy+math.Sin(th)*R*sy
		puff(b, r, px, py+1, side*r.in(10, 30), tt, 0.6, k*0.8, dustPals[dustEarth])
	}
}

// --- melee/heavy-cleave ------------------------------------------------------------

// fxHeavyCleave: the battle axe slammed down. An arc of embers, burning out
// along it, ends where the blade bites the ground; dust bursts out low and
// wide from there, chips are flung up, and embers drift on.
func fxHeavyCleave(b *fxBatch, r *fxRng, x, y, gy, ang, k, t float64) {
	side := 1.0
	if math.Cos(ang) < 0 {
		side = -1
	}
	// The arc: a crescent over the head, down to where it struck (x, gy).
	R := 15 * k
	cx, cy := x-side*R*0.55, gy-R*0.75
	a0, a1 := -math.Pi/2-side*1.3, math.Pi/2-side*0.95
	if side < 0 {
		a0, a1 = -math.Pi/2+1.3, math.Pi/2+0.95
	}
	head := outExpo(t / 0.07)
	tail := smoothstep(0.08, 0.5, t)
	const n = 24
	for i := 0; i < n; i++ {
		v0, v1 := float64(i)/n, float64(i+1)/n
		if v1 > head || v0 < tail {
			continue
		}
		th0, th1 := a0+(a1-a0)*v0, a0+(a1-a0)*v1
		heat := clamp01(1 - t/0.5)
		w := 4 * k * math.Pow(math.Sin(math.Pi*math.Pow(v0, 2)), 0.6) * (1 - 0.5*tail)
		for pass := 0; pass < 3; pass++ {
			ww := w * []float64{1, 0.6, 0.25}[pass]
			c := band(palEmber, heat*0.7+float64(pass)*0.15)
			b.ring(cx, cy, R-ww, R, 1, th0, th1, c, 1, 0)
		}
	}
	if t < 0.3 {
		b.glow(x, gy-2, 22*k, palEmber[4], 0.5*(1-t/0.3))
	}
	// The flash where it bites.
	if t < 0.07 {
		b.taper(x-9*k, gy, x+9*k, gy, 0.4, 0.4, palEmber[6], 1, 0)
		b.diamond(x, gy-1, 3*k, palEmber[6], 1, 0)
	}
	// Dust, low and wide; chips flung up; embers.
	for i := 0; i < 2; i++ {
		s := float64(i*2 - 1)
		puff(b, r, x+s*2, gy, s*r.in(30, 55), t, 0.9, k*1.2, dustPals[dustEarth])
		puff(b, r, x+s*4, gy, s*r.in(15, 30), t-0.05, 0.8, k, dustPals[dustEarth])
	}
	for i := 0; i < 9; i++ {
		a := -math.Pi/2 + r.in(-1.1, 1.1)
		v := r.in(60, 150) * k
		life := r.in(0.4, 0.7)
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, gy-1, math.Cos(a)*v, math.Sin(a)*v, 520, 0.8, t)
		if py > gy {
			py = gy
		}
		b.dot(px, py, 1.1, []color.RGBA{hex(0x5a4632), hex(0x8a6e4e), hex(0x3a2e22)}[i%3], 1-t/life*0.5, 0)
	}
	for i := 0; i < 10; i++ {
		life := r.in(0.6, 1)
		delay := r.in(0, 0.15)
		tt := t - delay
		if tt < 0 || tt >= life {
			continue
		}
		px := x + r.in(-8, 8)*k + math.Sin(tt*5+float64(i))*3
		py := gy - 2 - r.in(14, 30)*tt*k
		b.dot(px, py, 0.7, band(palEmber[2:], 1-tt/life), 1-tt/life, 0.3)
	}
}

// --- water/splash ------------------------------------------------------------------

// fxWaterSplash: something heavy into the water. A crown rises round where
// it went in and falls back, drops are thrown up and out and fall, rings
// spread on the surface and mist drifts off. k scales it: a stone is 0.5,
// the Megalodon 2.5.
func fxWaterSplash(b *fxBatch, r *fxRng, x, y, k, t float64) {
	const sy = 0.4
	const grav = 380.0
	// Rings on the surface.
	for i := 0; i < 3; i++ {
		u := (t - float64(i)*0.16) / 1.1
		if u <= 0 || u >= 1 {
			continue
		}
		rad := (4 + 22*outCubic(u)) * k
		b.ring(x, y, rad-0.7, rad+0.7, sy, 0, 2*math.Pi, palWater[5], (1-u)*0.8, 0)
	}
	// The crown: a sheet from the foot to the lip, its edge scalloped.
	h := 19 * 0.6 * k * outCubic(math.Min(t/0.09, 1)) * (1 - inQuad((t-0.12)/0.3))
	foot := (3 + 7.5*outCubic(math.Min(t/0.45, 1))) * 0.6 * k
	if h > 0.3 {
		const n = 22
		for half := 0; half < 2; half++ { // the far half behind, the near in front
			for i := 0; i < n; i++ {
				a := math.Pi * (float64(i) + 0.5) / n
				if half == 0 {
					a = -a
				}
				lane := 0.86 + 0.14*math.Sin(a*7+float64(r.s%7))
				lip := foot + 0.42*h
				fx, fy := x+math.Cos(a)*foot, y+math.Sin(a)*foot*sy
				lx, ly := x+math.Cos(a)*lip, y+math.Sin(a)*lip*sy-h*lane
				c := palWater[4]
				if half == 1 {
					c = palWater[5]
				}
				if t < 0.045 {
					c = palWater[6]
				}
				b.taper(fx, fy, lx, ly, 1.6*k*0.6, 1.1*k*0.6, c, 0.85, 0)
				b.dot(lx, ly, 0.8*k*0.6+0.3, palWater[6], 0.9, 0)
			}
		}
	}
	// Drops.
	for i := 0; i < 26; i++ {
		a := r.in(0, 2*math.Pi)
		hv := r.in(50, 125) * 0.6 * k
		vv := r.in(160, 270) * 0.6 * math.Sqrt(k)
		delay := r.in(0, 0.04)
		size := 0.8 + 0.5*float64(i%4/3)
		tt := t - delay
		if tt < 0 {
			continue
		}
		px := x + math.Cos(a)*hv*tt
		py := y + math.Sin(a)*hv*tt*sy - (vv*tt - grav*tt*tt/2)
		ground := y + math.Sin(a)*hv*tt*sy
		if py > ground {
			continue // back in the water: a ring would show where; too small to draw
		}
		b.dot(px, py, size*math.Sqrt(k), palWater[5+i%2], 0.95, 0)
	}
	// Mist.
	for i := 0; i < 9; i++ {
		s := float64(i%2*2 - 1)
		life := r.in(0.9, 1.5)
		delay := r.in(0, 0.1)
		tt := t - delay
		if tt < 0 || tt >= life {
			continue
		}
		px := x + s*r.in(4, 12)*0.6*k + s*r.in(25, 70)*0.6*tt*k*0.5
		py := y - r.in(1, 7)*0.6*k - r.in(4, 16)*0.6*tt
		rad := (2.5 + 3*tt/life) * k * 0.8
		b.disc(px, py, rad, rad*0.8, palWater[6], 0.35*(1-tt/life), 0.2)
	}
}

// --- dust and leaves ---------------------------------------------------------------

// puff is one cloud of dust from (x, y) drifting sideways at vx, t seconds
// into its life of life seconds: a few round lumps that grow, rise a little
// and thin out, lit on top.
func puff(b *fxBatch, r *fxRng, x, y, vx, t, life, k float64, pal [3]color.RGBA) {
	seed := r.next()
	if t < 0 || t >= life {
		return
	}
	u := t / life
	al := (1 - u) * (1 - u) * math.Min(1, t/0.04)
	for j := 0; j < 4; j++ {
		jx := (seed*13 + float64(j)*2.7)
		ox := math.Sin(jx)*2.2*k + vx*t*(1-u*0.5)
		oy := -math.Abs(math.Cos(jx*1.3))*1.5*k - (3+4*float64(j%2))*k*outCubic(u)
		rad := (1.4 + 1.6*float64(j%3)/2 + 2.6*outCubic(u)) * k
		b.disc(x+ox, y+oy+rad*0.15, rad, rad*0.82, pal[0], al*0.32, 0)
		b.disc(x+ox, y+oy, rad*0.9, rad*0.74, pal[1], al*0.45, 0)
		b.disc(x+ox-rad*0.25, y+oy-rad*0.3, rad*0.5, rad*0.4, pal[2], al*0.5, 0)
	}
}

// fxDustPuff: dust from the ground, blown back from a charge (ang the way
// it is thrown) or a footstep's small puff.
func fxDustPuff(b *fxBatch, r *fxRng, x, y, ang, k float64, pal int, t float64, step bool) {
	p := dustPals[min(max(pal, 0), len(dustPals)-1)]
	if step {
		pp := [3]color.RGBA{mix(p[0], color.RGBA{60, 40, 20, 255}, 0.3), p[1], mix(p[2], colWhite, 0.4)}
		puff(b, r, x, y, math.Cos(ang)*8, t, 0.5, 0.6*k, pp)
		puff(b, r, x+math.Cos(ang)*1.5, y, math.Cos(ang)*14, t-0.04, 0.45, 0.45*k, pp)
		return
	}
	for i := 0; i < 3; i++ {
		puff(b, r, x+float64(i-1)*2.5*k, y+float64(i%2), math.Cos(ang)*r.in(12, 30)*k, t-float64(i)*0.05, 0.75, k, p)
	}
}

// fxStompRing: a stomp: dust rolls out in a ring on the ground and pebbles
// hop.
func fxStompRing(b *fxBatch, r *fxRng, x, y, k float64, pal int, t float64) {
	p := dustPals[min(max(pal, 0), len(dustPals)-1)]
	const sy = 0.45
	u := t / (56.0 / 60)
	rad := (5 + 20*outCubic(u)) * k
	b.ring(x, y, rad-1.5*k*(1-u), rad, sy, 0, 2*math.Pi, p[1], 0.5*(1-u), 0)
	for i := 0; i < 10; i++ {
		a := 2*math.Pi*float64(i)/10 + r.in(-0.2, 0.2)
		d := (4 + 18*outCubic(u)) * k
		puff(b, r, x+math.Cos(a)*d, y+math.Sin(a)*d*sy, math.Cos(a)*10, t, 0.9, k*0.9, p)
	}
	for i := 0; i < 6; i++ {
		a := r.in(0, 2*math.Pi)
		v := r.in(20, 50) * k
		life := 0.5
		if t >= life {
			continue
		}
		hop := r.in(80, 140) * k * t * (1 - t/life)
		b.dot(x+math.Cos(a)*v*t, y+math.Sin(a)*v*t*sy-hop, 1, p[0], 1, 0)
	}
}

var leafPal = hexes(0x2d5a22, 0x3f7a2c, 0x5c9a38, 0x86b84e)

// fxLeafFall: leaves knocked loose, spinning down past the struck thing to
// the ground.
func fxLeafFall(b *fxBatch, r *fxRng, x, y, gy, k, t float64) {
	for i := 0; i < 6; i++ {
		ox := r.in(-7, 7) * k
		oy := r.in(-6, 4) * k
		vx := r.in(-14, 14)
		life := r.in(0.8, 1.15)
		ph := r.in(0, 6)
		if t >= life {
			continue
		}
		px := x + ox + vx*t + math.Sin(t*7+ph)*3*k
		py := math.Min(y+oy+t*t*30+t*8, gy-0.5)
		a := math.Sin(t*9+ph) * 1.2
		cs, sn := math.Cos(a)*1.6*k, math.Sin(a)*1.6*k
		c := leafPal[i%len(leafPal)]
		al := math.Min(1, (life-t)/0.25)
		b.quad(px-cs, py-sn, px-sn*0.45, py+cs*0.45, px+cs, py+sn, px+sn*0.45, py-cs*0.45, c, al, 0)
	}
}

// fxImpactBits: what flies off a struck beast, stylised: a few small red
// sparks, a puff of dust from its feet, and leaves in the woods (pal is
// the ground's dust, or dustGrass+1 for the woods).
func fxImpactBits(b *fxBatch, r *fxRng, x, y, gy, ang, k float64, pal int, t float64) {
	for i := 0; i < 5; i++ {
		a := ang + r.in(-0.7, 0.7) - 0.4
		v := r.in(60, 140) * k
		life := r.in(0.25, 0.5)
		if t >= life {
			continue
		}
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-30, 360, 1.5, t)
		if py > gy {
			continue
		}
		b.streak(px, py, vx, vy, 0.015, 0.8, band([]color.RGBA{palClaw[2], palClaw[3], palClaw[4]}, 1-t/life), 1, 0.2)
	}
	if pal <= dustGrass {
		puff(b, r, x+math.Cos(ang)*3, gy, math.Cos(ang)*18, t, 0.55, 0.7*k, dustPals[pal])
	}
}

// --- shields ------------------------------------------------------------------------

var palWood = hexes(0x3a2414, 0x5e3c20, 0x8a5c32, 0xb88a52)

// fxShieldBlock: a blow on a raised shield: a clack of white and gold
// sparks thrown back off its face, splinters, and a short arc where it
// was struck.
func fxShieldBlock(b *fxBatch, r *fxRng, x, y, ang, k, t float64) {
	if t < 0.05 {
		b.glow(x, y, 9*k, palGold[5], 0.6)
		b.diamond(x, y, 2*k, palGold[6], 1, 0)
	}
	if t < 0.12 {
		u := t / 0.12
		rad := (3 + 6*outCubic(u)) * k
		b.ring(x, y, rad-0.8*k, rad, 1, ang-math.Pi-0.9, ang-math.Pi+0.9, palGold[5], 1-u, 0.3)
	}
	for i := 0; i < 9; i++ {
		a := ang + math.Pi + r.in(-1, 1)
		v := r.in(90, 200) * k
		life := r.in(0.15, 0.35)
		if t >= life {
			continue
		}
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-30, 300, 3, t)
		b.streak(px, py, vx, vy, 0.018, 0.65, band(palGold[2:], 1-t/life), 1, 0.3)
	}
	for i := 0; i < 5; i++ {
		a := ang + math.Pi + r.in(-1.2, 1.2) - 0.4
		v := r.in(40, 90) * k
		life := r.in(0.35, 0.5)
		if t >= life {
			continue
		}
		px, py, _, _ := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-40, 420, 0.8, t)
		th := t*20 + float64(i)
		cs, sn := math.Cos(th)*1.1*k, math.Sin(th)*1.1*k
		b.quad(px-cs, py-sn, px-sn*0.3, py+cs*0.3, px+cs, py+sn, px+sn*0.3, py-cs*0.3, palWood[1+i%3], 1, 0)
	}
}

// fxParryFlash: a parry: two blades of light crossed in an X where the
// blow was turned, a ring pulsing out, and gold sparks.
func fxParryFlash(b *fxBatch, r *fxRng, x, y, ang, k, t float64) {
	if t < 0.2 {
		u := t / 0.2
		l := (7 + 5*outBack(math.Min(1, t/0.05))) * k * (1 - u*0.4)
		w := 1.6 * k * (1 - u)
		for _, d := range [2]float64{0.78, -0.78} {
			cs, sn := math.Cos(d), math.Sin(d)
			b.taper(x-cs*l, y-sn*l, x, y, 0.2, w*1.6, palGold[4], 1-u, 0.2)
			b.taper(x+cs*l, y+sn*l, x, y, 0.2, w*1.6, palGold[4], 1-u, 0.2)
			b.taper(x-cs*l*0.7, y-sn*l*0.7, x, y, 0.1, w*0.7, palGold[6], 1-u, 0)
			b.taper(x+cs*l*0.7, y+sn*l*0.7, x, y, 0.1, w*0.7, palGold[6], 1-u, 0)
		}
		b.glow(x, y, 16*k, palGold[4], 0.55*(1-u))
	}
	if t < 0.35 {
		u := t / 0.35
		rad := (4 + 14*outCubic(u)) * k
		b.ring(x, y, rad-1.2*k*(1-u), rad, 0.85, 0, 2*math.Pi, palGold[5], 1-u, 0.3)
	}
	for i := 0; i < 16; i++ {
		a := r.in(0, 2*math.Pi)
		if i < 8 {
			a = ang + math.Pi + r.in(-0.8, 0.8)
		}
		v := r.in(100, 240) * k
		life := r.in(0.2, 0.5)
		if t >= life {
			continue
		}
		px, py, vx, vy := ballistic(x, y, math.Cos(a)*v, math.Sin(a)*v-40, 320, 2.8, t)
		b.streak(px, py, vx, vy, 0.02, 0.7, band(palGold[3:], 1-t/life), 1, 0.3)
	}
}
