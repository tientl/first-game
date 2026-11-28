package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Bodies. A kill (sim.EntityCorpse, Var its CreatureKind, Big the way it
// faced) lies where it fell, drawn from the beast's own HD drawing laid
// down, not stood on its head:
//
//   - a dinosaur of dinos/ lies on its side through its rig (dinorig.go):
//     its legs go loose and splay out towards the viewer, turned about
//     their joints (limpJoints), the head and the neck sink to the ground,
//     the tail lies flat, and the body is pressed down onto the belly;
//   - a furred or scaled beast drawn in code (critterhd.go) is laid down
//     the same way from its standing drawing: each leg is lifted off below
//     the belly and turned out about its hip, bent at the knee;
//   - a crab, a spider or a scorpion lies on its back, legs curled up;
//     a snake belly up; a warrior on his back;
//   - anything else (the bosses, the sea monsters) is its own picture laid
//     low.
//
// Every body is drawn the colour gone out of it, its eye shut, over a dark
// stain where it touches the ground and its blood (bloodart.go). A body seen
// falling (its beast was there a moment ago: corpseMarks) tips over through
// corpseSteps frames into its lying pose; the blood then seeps out from
// under it.

type corpseStyle uint8

const (
	corpseSide   corpseStyle = iota // on its side, legs loose and splayed towards the viewer
	corpseBack                      // on its back, legs curled up: the crab, the spider, the scorpion
	corpseFlat                      // a man on his back: the figure turned down
	corpseBelly                     // a snake: belly up, stretched out
	corpseSprite                    // its own picture laid low: none of the others fits
)

// corpseSpec is how a kind of beast lies dead.
type corpseSpec struct {
	style corpseStyle
	rig   string  // a dinosaur drawn in dinos/ (dinorigs.go); its pose dead is dead
	dead  rigPose // head and neck sunk, tail down; limp is set as it falls
	crit  string  // a beast drawn in code (critreg.go), standing in pose
	pose  int
	flat  float64 // how high the body lies, as a share of its height standing (1: as drawn)
	wound float64 // where the blood comes from, 0 at its tail end to 1 at its head
	biped bool    // two legs: they fall one forward, one back
	trunk bool    // a trunk reaches the ground before its fore legs (splayLegs)
	pool  float64 // its blood as much as a body this long would leave (0: 1); a snake is long and thin
}

var corpseSpecs = map[sim.CreatureKind]corpseSpec{
	sim.CreatureTrike:   {style: corpseSide, rig: "trike", dead: rigPose{head: 0.32, tail: -0.3, tilt: 0.05}, flat: 0.78, wound: 0.62},
	sim.CreatureToro:    {style: corpseSide, rig: "toro", dead: rigPose{head: 0.22, tail: -0.15}, flat: 0.78, wound: 0.62},
	sim.CreatureStego:   {style: corpseSide, rig: "stego", dead: rigPose{head: 0.45, tail: -0.25}, flat: 0.78, wound: 0.6},
	sim.CreatureAnkylo:  {style: corpseSide, rig: "ankylo", dead: rigPose{head: 0.4, tail: -0.2}, flat: 0.78, wound: 0.6},
	sim.CreatureBrachio: {style: corpseSide, rig: "brachio", dead: rigPose{neck: 0.85, head: -0.15, tail: -0.25}, flat: 0.78, wound: 0.55},
	sim.CreatureRaptor:  {style: corpseSide, rig: "dakota", dead: rigPose{head: 0.5, tail: -0.2, open: 0.06, tilt: 0.12}, flat: 0.78, wound: 0.62, biped: true},
	sim.CreatureAchero:  {style: corpseSide, rig: "achero", dead: rigPose{head: 0.55, tail: -0.14, open: 0.15}, flat: 0.78, wound: 0.62, biped: true},
	sim.CreatureTroodon: {style: corpseSide, rig: "troodon", dead: rigPose{head: 0.5, tail: -0.14, open: 0.12}, flat: 0.78, wound: 0.62, biped: true},
	sim.CreatureRex:     {style: corpseSide, rig: "rex", dead: rigPose{head: 0.45, tail: -0.14, open: 0.2}, flat: 0.78, wound: 0.62, biped: true},
	sim.CreatureOrnitho: {style: corpseSide, rig: "ornitho", dead: rigPose{neck: 1.35, head: -0.3, tail: -0.22}, flat: 0.78, wound: 0.6, biped: true},
	sim.CreatureThesc:   {style: corpseSide, rig: "thesc", dead: rigPose{head: 0.5, tail: -0.22}, flat: 0.78, wound: 0.6, biped: true},
	sim.CreatureAnzu:    {style: corpseSide, rig: "anzu", dead: rigPose{head: 1.0, tail: -0.2, open: 0.15}, flat: 0.78, wound: 0.6, biped: true},
	sim.CreatureQuetzal: {style: corpseSide, rig: "quetzal", dead: rigPose{neck: 1.3, head: 0.2}, flat: 0.78, wound: 0.55},
	sim.CreaturePlesio:  {style: corpseSide, rig: "plesio", dead: rigPose{head: 0.6}, flat: 0.78, wound: 0.55},

	sim.CreatureMammoth: {style: corpseSide, crit: "mammoth", pose: mammothWalkA, trunk: true, dead: rigPose{head: 0.1, tail: -0.3}, flat: 0.78, wound: 0.6},
	sim.CreatureSabre:   {style: corpseSide, crit: "sabre", pose: sabreStand, dead: rigPose{head: 0.45, tail: -0.5}, flat: 0.78, wound: 0.62},
	sim.CreatureBear:    {style: corpseSide, crit: "bear", pose: bearWalkA, dead: rigPose{head: 0.4, tail: -0.3}, flat: 0.78, wound: 0.62},
	sim.CreatureMonkey:  {style: corpseSide, crit: "monkey", pose: monkeyWalkA, dead: rigPose{head: 0.4, tail: -0.4}, flat: 0.78, wound: 0.6},
	sim.CreatureRabbit:  {style: corpseSide, crit: "rabbit", pose: rabbitHop, dead: rigPose{head: 0.35}, flat: 0.78, wound: 0.6},
	sim.CreatureCroc:    {style: corpseBelly, crit: "croc", pose: crocWalkA, dead: rigPose{head: 0.12, tail: -0.1}, flat: 0.78, wound: 0.58},
	sim.CreatureBrachy:  {style: corpseBelly, crit: "brachy", pose: crocWalkA, dead: rigPose{head: 0.12, tail: -0.1}, flat: 0.78, wound: 0.58},
	sim.CreatureChampso: {style: corpseBelly, crit: "champso", pose: crocWalkA, dead: rigPose{head: 0.12, tail: -0.1}, flat: 0.78, wound: 0.58},

	sim.CreatureCrab:     {style: corpseBack, crit: "crab", pose: crabWalkA, flat: 0.95, wound: 0.5},
	sim.CreatureSpider:   {style: corpseBack, crit: "spider", pose: spiderWalkA, flat: 0.95, wound: 0.5},
	sim.CreatureScorpion: {style: corpseBack, crit: "scorpion", pose: scorpionWalkA, flat: 0.95, wound: 0.55},
	sim.CreatureSnake:    {style: corpseBelly, crit: "snake", pose: 0, flat: 1, wound: 0.55, pool: 0.45},

	sim.CreatureTribesman: {style: corpseFlat, crit: "tribe", pose: tribeStand, flat: 1, wound: 0.62},
}

// corpseSpecOf is how a body of kind k lies: its own way, or as its own
// picture laid low.
func corpseSpecOf(k sim.CreatureKind) corpseSpec {
	if sp, ok := corpseSpecs[k]; ok {
		return sp
	}
	return corpseSpec{style: corpseSprite, flat: 0.95, wound: 0.55}
}

// corpseSteps is how many frames a body takes to fall: step corpseSteps is
// the body lying; 0 the beast standing as it was struck.
const (
	corpseSteps     = 5
	corpseStepTicks = 4 // game frames a step of the fall shows
)

// corpseArt is a body drawn in one step of its fall, facing right or left.
type corpseArt struct {
	s      sprite
	ok     bool
	length float64 // how long the body lies, art pixels
	wx     float64 // where its blood comes from, art pixels right of the anchor
	high   float64 // how far it rises over the ground, art pixels
}

type corpseKey struct {
	k    sim.CreatureKind
	left bool
	step int
}

var corpseArts = map[corpseKey]corpseArt{}

// corpseArtOf is the body of kind k at step of its fall, drawn the first
// time it is wanted.
func corpseArtOf(k sim.CreatureKind, left bool, step int) corpseArt {
	step = max(0, min(corpseSteps, step))
	key := corpseKey{k, left, step}
	if a, ok := corpseArts[key]; ok {
		return a
	}
	var a corpseArt
	if left {
		a = corpseArtOf(k, false, step)
		if a.ok {
			a.s = mirrorSprite(a.s)
			a.wx = -a.wx
		}
	} else if c, ax, ay, ok := corpseCanvas(k, float64(step)/corpseSteps); ok {
		a = corpseArt{s: mk(c, ax, ay), ok: true}
		a.length = float64(c.w) / float64(c.dens)
		a.high = ay / float64(c.dens)
		a.wx = (float64(c.w)*corpseSpecOf(k).wound - ax) / float64(c.dens)
	}
	corpseArts[key] = a
	return a
}

// corpseCanvas draws a body of kind k a share t of the way through its
// fall (1: lying), facing right, at HD density; its anchor in image pixels
// is where it touches the ground. ok is false when kind k has no drawing to
// lay down.
func corpseCanvas(k sim.CreatureKind, t float64) (c *canvas, ax, ay float64, ok bool) {
	sp := corpseSpecOf(k)
	switch {
	case sp.rig != "":
		c, ax, ay, ok = corpseRigCanvas(sp, t)
	case sp.crit != "":
		c, ax, ay, ok = corpseCritCanvas(sp, t)
	}
	if !ok {
		return nil, 0, 0, false
	}
	if t > 0 {
		shutEyes(c, t)
		deaden(c, t)
	}
	return c, ax, ay, true
}

// corpseRigCanvas lays a dinosaur down through its rig.
func corpseRigCanvas(sp corpseSpec, t float64) (*canvas, float64, float64, bool) {
	r, ok := dinoRigs[sp.rig]
	if !ok {
		return nil, 0, 0, false
	}
	p := sp.dead
	e := smooth(t)
	p.head, p.neck, p.tail, p.open, p.tilt = p.head*e, p.neck*e, p.tail*e, p.open*e, p.tilt*e
	if r.jointed() {
		p.limp = t
	}
	c, ax, ay, ok := rigCanvas(r, p)
	if !ok {
		return nil, 0, 0, false
	}
	if c.dens == 0 {
		c.dens = 1
	}
	if !r.jointed() && len(r.legs) > 0 { // legs with no joints measured: lifted off and turned whole
		q := sp
		q.dead = rigPose{}
		c2, ax2, ay2, belly := splayLegs(c, ax, ay, t, q)
		return lieDown(c2, ax2, ay2, belly, sp.flat, t)
	}
	kf := float64(c.dens)
	// Where its legs leave the body: it lies on that.
	belly := ay
	if len(r.legs) > 0 {
		ay0 := r.ay
		if r.ax == 0 && r.ay == 0 {
			_, ay0 = underFeet(dinoImage(r.file, false))
		}
		top := 0.0
		for _, l := range r.legs {
			top = max(top, float64(l.top))
		}
		belly = ay + (top-ay0)*kf + 1.5*kf
	}
	return lieDown(c, ax, ay, belly, sp.flat, t)
}

// lieDown sets canvas c on the ground, t of the way (all of it at 1): its
// anchor goes from the ground under its feet (ay) to the row its belly
// rests on (or the lowest stretch of it, if that is higher). Its legs,
// splayed (limpJoints, splayLegs), lie on the ground out towards the
// viewer, seen at a slant: what is below the belly is pressed to half its
// height; and the flank it shows to the sky a little lower than it stood
// (flat).
func lieDown(c *canvas, ax, ay, belly, flat, t float64) (*canvas, float64, float64, bool) {
	e := smooth(t)
	g := lerpf(ay, math.Min(ay, math.Min(belly, float64(lowestRow(c))+0.5)), e)
	pressFlat(c, g, lerpf(1, flat, e), lerpf(1, 0.5, e))
	return c, ax, g, true
}

// lowestRow is the lowest row of c with a real stretch of the drawing on
// it: not a claw or a stray pixel.
func lowestRow(c *canvas) int {
	widest := 0
	n := make([]int, c.h)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A >= 128 {
				n[y]++
			}
		}
		widest = max(widest, n[y])
	}
	need := max(3*c.dens, widest/14)
	for y := c.h - 1; y >= 0; y-- {
		if n[y] >= need {
			return y
		}
	}
	return c.h - 1
}

// pressFlat squeezes canvas c towards row g: what is above it to up of its
// height, what is below to down.
func pressFlat(c *canvas, g, up, down float64) {
	if up > 0.999 && down > 0.999 {
		return
	}
	src := append([]color.RGBA(nil), c.px...)
	clear(c.px)
	for y := 0; y < c.h; y++ {
		d := float64(y) + 0.5 - g
		k := up
		if d > 0 {
			k = down
		}
		sy := int(math.Floor(g + d/k))
		if sy < 0 || sy >= c.h {
			continue
		}
		copy(c.px[y*c.w:(y+1)*c.w], src[sy*c.w:(sy+1)*c.w])
	}
}

// corpseCritCanvas lays down a beast drawn in code.
func corpseCritCanvas(sp corpseSpec, t float64) (*canvas, float64, float64, bool) {
	kd := critKinds[sp.crit]
	if kd == nil || kd.hd == nil {
		return nil, 0, 0, false
	}
	src, ax, ay := kd.hd(sp.pose, 0)
	if src.dens == 0 {
		src.dens = 1
	}
	kf := float64(src.dens)
	ax, ay = ax*kf, ay*kf // image pixels
	switch sp.style {
	case corpseSide:
		c, ax, ay, belly := splayLegs(src, ax, ay, t, sp)
		return lieDown(c, ax, ay, belly, sp.flat, t)
	case corpseBack:
		c := padCanvas(src, int(8*kf))
		ax, ay = ax+8*kf, ay+8*kf
		top, bot := opaqueRows(c)
		if t >= 0.5 {
			flipRows(c, top, bot) // over onto its back
		}
		f := lerpf(1, sp.flat, smooth(t))
		crouch(c, float64(bot)+1, f)
		return c, ax, ay, true
	case corpseBelly:
		c := padCanvas(src, int(4*kf))
		ax, ay = ax+4*kf, ay+4*kf
		if t >= 0.5 {
			top, bot := opaqueRows(c)
			flipRows(c, top, bot) // rolled over, the pale belly up
		}
		return c, ax, ay, true
	case corpseFlat:
		// Over onto his back, his head where his heels were behind him.
		c, nax, nay := turnCanvas(src, ax, ay, -math.Pi/2*smooth(t))
		if t >= 1 {
			nay -= 2 * kf // the drawing's depth lies over the ground: its anchor nearer its middle
		}
		return c, nax, nay, true
	}
	return nil, 0, 0, false
}

// --- canvas work ---------------------------------------------------------------------

// padCanvas is c with n clear pixels round it.
func padCanvas(c *canvas, n int) *canvas {
	out := newCanvas(c.w+2*n, c.h+2*n)
	out.dens = c.dens
	for y := 0; y < c.h; y++ {
		copy(out.px[(y+n)*out.w+n:(y+n)*out.w+n+c.w], c.px[y*c.w:(y+1)*c.w])
	}
	return out
}

// opaqueRows is the first and last row of c with anything drawn on it.
func opaqueRows(c *canvas) (top, bot int) {
	top, bot = c.h, -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A >= 128 {
				top, bot = min(top, y), max(bot, y)
				break
			}
		}
	}
	return top, bot
}

// flipRows turns rows top to bot of c upside down, and darkens what was
// underneath and is now on top a little less than what was lit.
func flipRows(c *canvas, top, bot int) {
	for a, b := top, bot; a < b; a, b = a+1, b-1 {
		ra, rb := c.px[a*c.w:(a+1)*c.w], c.px[b*c.w:(b+1)*c.w]
		for x := range ra {
			ra[x], rb[x] = rb[x], ra[x]
		}
	}
}

// turnCanvas turns c by a radians about its anchor (ax, ay), image pixels,
// onto a canvas big enough to hold it; quarter turns are exact.
func turnCanvas(c *canvas, ax, ay, a float64) (*canvas, float64, float64) {
	sn, cs := math.Sincos(a)
	// The corners, turned, give the new bounds.
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range [4][2]float64{{0, 0}, {float64(c.w), 0}, {0, float64(c.h)}, {float64(c.w), float64(c.h)}} {
		dx, dy := p[0]-ax, p[1]-ay
		x, y := dx*cs-dy*sn, dx*sn+dy*cs
		x0, y0, x1, y1 = math.Min(x0, x), math.Min(y0, y), math.Max(x1, x), math.Max(y1, y)
	}
	x0, y0 = math.Floor(x0+1e-6), math.Floor(y0+1e-6)
	w, h := int(math.Ceil(x1-1e-6)-x0), int(math.Ceil(y1-1e-6)-y0)
	out := newCanvas(w, h)
	out.dens = c.dens
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)+0.5+x0, float64(y)+0.5+y0
			sx, sy := ax+dx*cs+dy*sn, ay-dx*sn+dy*cs
			if p := c.at(int(math.Floor(sx)), int(math.Floor(sy))); p.A >= 128 {
				out.px[y*w+x] = p
			}
		}
	}
	return out, -x0, -y0
}

// deaden takes the colour out of a body: a little greyer and darker, the
// more so the further through its fall (t).
func deaden(c *canvas, t float64) {
	k := smooth(t)
	for i, p := range c.px {
		if p.A == 0 {
			continue
		}
		l := 0.3*float64(p.R) + 0.55*float64(p.G) + 0.15*float64(p.B)
		f := func(v uint8) uint8 {
			x := lerpf(float64(v), l, 0.22*k) * (1 - 0.1*k)
			return uint8(max(0, min(255, x)))
		}
		c.px[i] = color.RGBA{f(p.R), f(p.G), f(p.B), p.A}
	}
}

// shutEyes closes the eyes of a body: each glint (a near-white speck with
// dark round it) and the dark of the eye about it become the hide round
// it, with a dark line of the lid across.
func shutEyes(c *canvas, t float64) {
	if t < 0.5 {
		return
	}
	r := max(2, c.dens+1) // how far an eye reaches from its glint
	bright := func(p color.RGBA) bool { return p.A >= 128 && int(p.R)+int(p.G)+int(p.B) >= 690 }
	dark := func(p color.RGBA) bool { return p.A >= 128 && int(p.R)+int(p.G)+int(p.B) <= 200 }
	type pt struct{ x, y int }
	var glints []pt
	for y := 1; y < c.h-1; y++ {
		for x := 1; x < c.w-1; x++ {
			if !bright(c.px[y*c.w+x]) {
				continue
			}
			// A glint is small: not a tooth, a tusk or a horn.
			n, dk := 0, 0
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					q := c.at(x+dx, y+dy)
					if bright(q) {
						n++
					}
					if dark(q) {
						dk++
					}
				}
			}
			if n <= 2 && dk >= 3 {
				glints = append(glints, pt{x, y})
			}
		}
	}
	for _, g := range glints {
		// The hide round the eye: the commonest colour on a ring outside it.
		count := map[color.RGBA]int{}
		var best color.RGBA
		for dy := -r - 2; dy <= r+2; dy++ {
			for dx := -r - 2; dx <= r+2; dx++ {
				d := math.Hypot(float64(dx), float64(dy))
				if d < float64(r)+1 || d > float64(r)+2.5 {
					continue
				}
				q := c.at(g.x+dx, g.y+dy)
				if q.A < 128 || dark(q) || bright(q) {
					continue
				}
				q.A = 255
				count[q]++
				if count[q] > count[best] {
					best = q
				}
			}
		}
		if best.A == 0 {
			continue
		}
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				q := c.at(g.x+dx, g.y+dy)
				if q.A < 128 || math.Hypot(float64(dx), float64(dy)) > float64(r)+0.5 {
					continue
				}
				if dark(q) || bright(q) || colorDist(q, best) > 60 {
					c.put(g.x+dx, g.y+dy, best)
				}
			}
		}
		lid := mix(best, colInk, 0.7)
		for dx := -r + 1; dx <= r-1; dx++ {
			dy := 0
			if dx == -r+1 || dx == r-1 {
				dy = -1 // the lid's line curves up at its corners
			}
			c.put(g.x+dx, g.y+dy+1, lid)
		}
	}
}

func colorDist(a, b color.RGBA) int {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) + d(a.G, b.G) + d(a.B, b.B)
}

// --- legs ------------------------------------------------------------------------------

// limpAim is the way a leg points on a body lying dead, radians from
// straight down (+ towards the tail; the drawings face right), and how far
// it is left bent at each joint. The legs go stiff and splay apart, fore
// legs forwards and hind legs back, near ones further than far ones, so the
// body lying on its side shows them all and none stands under it.
func limpAim(fore, far, biped bool) (aim, bend float64) {
	switch {
	case biped && far:
		return 0.7, 0.15
	case biped:
		return 1.15, 0.15
	case fore && far:
		return -1.0, 0.08
	case fore:
		return -1.4, 0.08
	case far:
		return 0.95, 0.08
	}
	return 1.35, 0.08
}

// limpJoints is where the joints of a leg fall as a body goes down dead, t
// of the way: each bone turned from the drawing towards the leg's aim
// (limpAim), straightening it; a biped's toes curl forward.
func (c *legCut) limpJoints(t float64, biped bool) ([]rigPt, []float64) {
	n := len(c.P) - 1
	aim, bend := limpAim(c.fore, c.far, biped)
	th := make([]float64, n)
	J := make([]rigPt, n+1)
	J[0] = c.P[0]
	e := smooth(t)
	for k := range n {
		v := c.P[k+1].sub(c.P[k])
		a0 := math.Atan2(-v.x, v.y) // as drawn, from straight down
		to := aim + bend*float64(1-2*(k%2))
		if !c.fore {
			to = aim - bend*float64(1-2*(k%2)) // a knee bends the other way to an elbow
		}
		if biped && k == n-1 && n >= 3 {
			to = aim - 1.3 // the toes, curled forward
		}
		d := math.Remainder(to-a0, 2*math.Pi)
		th[k] = d * e
		J[k+1] = J[k].add(turnPt(v, th[k]))
	}
	return J, th
}

// splayLegs lifts the legs of a standing beast drawn in code off below its
// belly and turns each out about its hip, t of the way: the near legs over
// the body, the far ones behind it. It gives the canvas, its anchor and the
// row of the belly, image pixels.
func splayLegs(src *canvas, ax, ay, t float64, sp corpseSpec) (*canvas, float64, float64, float64) {
	biped := sp.biped
	k := src.dens
	kf := float64(k)
	pad := 16 * k
	w, h := src.w+2*pad, src.h+2*pad
	in := padCanvas(src, pad)
	ax, ay = ax+float64(pad), ay+float64(pad)
	gy := int(ay) // the ground row
	// The belly: the lowest row with a stretch of body unbroken half as
	// long as the longest (below it, only legs).
	widest := 0
	rowW := make([]int, h)
	for y := 0; y < h; y++ {
		run := 0
		for x := 0; x < w; x++ {
			if in.px[y*w+x].A >= 128 {
				run++
				rowW[y] = max(rowW[y], run)
			} else {
				run = 0
			}
		}
		widest = max(widest, rowW[y])
	}
	belly := gy
	for y := gy; y >= 0; y-- {
		if rowW[y] >= widest*45/100 {
			belly = y
			break
		}
	}
	// The legs: what reaches the ground below the belly, piece by piece.
	lab := make([]int32, w*h)
	type leg struct {
		px          []int
		x0, x1      int
		bottom      int
		hipX, hipY  float64
		lum         float64
		hind, front bool
	}
	var legs []*leg
	var stack []int
	for y := belly + 1; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if lab[i] != 0 || in.px[i].A < 128 {
				continue
			}
			l := &leg{x0: w, x1: -1}
			id := int32(len(legs) + 1)
			lab[i] = id
			stack = append(stack[:0], i)
			for len(stack) > 0 {
				q := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				l.px = append(l.px, q)
				qx, qy := q%w, q/w
				l.x0, l.x1, l.bottom = min(l.x0, qx), max(l.x1, qx), max(l.bottom, qy)
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := qx+d[0], qy+d[1]
					if nx < 0 || ny <= belly || nx >= w || ny >= h {
						continue
					}
					if j := ny*w + nx; lab[j] == 0 && in.px[j].A >= 128 {
						lab[j] = id
						stack = append(stack, j)
					}
				}
			}
			legs = append(legs, l)
		}
	}
	var keep []*leg
	for _, l := range legs {
		if l.bottom >= gy-2*k && l.bottom-belly >= 2*k && len(l.px) >= 6*k*k {
			keep = append(keep, l)
		} else {
			for _, q := range l.px {
				lab[q] = 0 // a tail, a trunk: the body's
			}
		}
	}
	legs = keep
	if sp.trunk && len(legs) > 1 { // the frontmost reaching the ground is its trunk: the body's
		fi := 0
		for i, l := range legs {
			if l.x0 > legs[fi].x0 {
				fi = i
			}
		}
		for _, q := range legs[fi].px {
			lab[q] = 0
		}
		legs = append(legs[:fi:fi], legs[fi+1:]...)
	}
	if len(legs) == 0 {
		return in, ax, ay, float64(belly)
	}
	// Each leg's hip, its colour, and which it is.
	var lums []float64
	for _, l := range legs {
		sx, n := 0.0, 0
		s := 0.0
		for _, q := range l.px {
			p := in.px[q]
			s += 0.3*float64(p.R) + 0.55*float64(p.G) + 0.15*float64(p.B)
			if q/w <= belly+2*k {
				sx += float64(q%w) + 0.5
				n++
			}
		}
		if n == 0 {
			sx, n = float64(l.x0+l.x1)/2, 1
		}
		l.hipX, l.hipY = sx/float64(n), float64(belly)-0.5*kf
		l.lum = s / float64(len(l.px))
		lums = append(lums, l.lum)
	}
	mid := 0.0
	for _, l := range legs {
		mid += l.hipX
	}
	mid /= float64(len(legs))
	avgLum := 0.0
	for _, v := range lums {
		avgLum += v
	}
	avgLum /= float64(len(lums))
	for _, l := range legs {
		l.hind = l.hipX < mid
		l.front = l.lum >= avgLum-2
	}
	if len(legs) == 1 {
		legs[0].front = true
	}
	// Lift them off: the body keeps a band above the belly that each leg
	// also takes with it, so its turned top covers the cut.
	out := newCanvas(w, h)
	out.dens = src.dens
	for i, p := range in.px {
		if lab[i] == 0 {
			out.px[i] = p
		}
	}
	// The cut along the belly where a leg came away: the body's edge, darkened.
	for _, l := range legs {
		for x := l.x0; x <= l.x1; x++ {
			if p := out.px[belly*w+x]; p.A >= 128 && lab[(belly+1)*w+x] != 0 {
				out.px[belly*w+x] = mix(p, colInk, 0.35)
			}
		}
	}
	e := smooth(t)
	// The head sinks about the neck, before the fore legs; the tail about
	// its root, behind the hind legs.
	front, back := legs[0].hipX, legs[0].hipX
	for _, l := range legs {
		front, back = max(front, l.hipX), min(back, l.hipX)
	}
	topAt := func(x float64) float64 {
		for y := 0; y < belly; y++ {
			if out.px[y*w+int(x)].A >= 128 {
				return float64(y)
			}
		}
		return float64(belly)
	}
	if sp.dead.head != 0 {
		px := front + 2.5*kf
		py := lerpf(topAt(px), float64(belly), 0.45)
		turn(out, rigPart{pivot: rigPt{px, py}, dir: -0.35, maxY: float64(belly) + 1}, sp.dead.head*e, 3*kf)
	}
	if sp.dead.tail != 0 {
		px := back - 2*kf
		py := lerpf(topAt(px), float64(belly), 0.4)
		turn(out, rigPart{pivot: rigPt{px, py}, dir: math.Pi, maxY: float64(belly) + 1}, sp.dead.tail*e, 3*kf)
	}
	layer := newCanvas(w, h)
	paint := func(l *leg, front bool) {
		clear(layer.px)
		// The angles: + swings the foot back.
		aim, _ := limpAim(!l.hind, !l.front, biped) // stiff: the leg straight, turned whole
		a0, a1 := aim, aim*1.05
		a0, a1 = a0*e, a1*e
		long := float64(l.bottom) - l.hipY
		kneeY := l.hipY + long*0.5
		s0, c0 := math.Sincos(a0)
		s1, c1 := math.Sincos(a1)
		kx, ky := l.hipX, kneeY
		kdx, kdy := kx-l.hipX, ky-l.hipY
		kx2, ky2 := l.hipX+kdx*c0-kdy*s0, l.hipY+kdx*s0+kdy*c0
		place := func(fx, fy float64) (float64, float64) {
			if fy < kneeY {
				dx, dy := fx-l.hipX, fy-l.hipY
				return l.hipX + dx*c0 - dy*s0, l.hipY + dx*s0 + dy*c0
			}
			dx, dy := fx-kx, fy-ky
			return kx2 + dx*c1 - dy*s1, ky2 + dx*s1 + dy*c1
		}
		// The leg's own pixels, and the body's just above its hip.
		var pts []int
		pts = append(pts, l.px...)
		for y := max(0, belly-2*k); y <= belly; y++ {
			for x := l.x0; x <= l.x1; x++ {
				if in.px[y*w+x].A >= 128 {
					pts = append(pts, y*w+x)
				}
			}
		}
		const ss = 3
		for _, q := range pts {
			p := in.px[q]
			for j := range ss {
				for i := range ss {
					fx := float64(q%w) + (float64(i)+0.5)/ss
					fy := float64(q/w) + (float64(j)+0.5)/ss
					tx, ty := place(fx, fy)
					layer.put(int(math.Floor(tx)), int(math.Floor(ty)), p)
				}
			}
		}
		for i, p := range layer.px {
			if p.A < 128 {
				continue
			}
			if front || out.px[i].A < 128 {
				out.px[i] = p
			}
		}
	}
	for _, l := range legs {
		if !l.front {
			paint(l, false)
		}
	}
	for _, l := range legs {
		if l.front {
			paint(l, true)
		}
	}
	despeck(out, 0, belly, w, h, 2*k)
	return out, ax, ay, float64(belly) + 0.5*kf
}

// --- when a body fell ------------------------------------------------------------------

// corpseMark is what is known of a body: when it was first seen, and
// whether it was seen falling (its beast there a moment before), from
// where.
type corpseMark struct {
	at     int     // the frame it was first seen
	seen   int     // the frame it was last seen
	fresh  bool    // it fell in view
	dx, dy float64 // where its beast stood, art pixels from its tile's middle
}

var corpseMarks = map[sim.EntityID]corpseMark{}

// corpseMarkOf is what is known of body e, noting it the first time.
func (g *Game) corpseMarkOf(e protocol.EntityView) corpseMark {
	m, ok := corpseMarks[e.ID]
	if !ok {
		m.at = g.frame
		cx, cy := float64(e.TX)+0.5, float64(e.TY)+0.5
		// Its beast was in the last snapshot and is gone from this one.
		best := 3.5
		for id, c := range g.prevCreatures {
			if c.Kind != sim.CreatureKind(e.Var) || g.creatureNow(id) {
				continue
			}
			if d := math.Hypot(float64(c.X)-cx, float64(c.Y)-cy); d < best {
				best, m.fresh = d, true
				m.dx = max(-0.6, min(0.6, float64(c.X)-cx)) * TileW
				m.dy = max(-0.6, min(0.6, float64(c.Y)-cy)) * TileH
			}
		}
		if len(corpseMarks) > 256 {
			for id, o := range corpseMarks {
				if g.frame-o.seen > 60*60 {
					delete(corpseMarks, id)
				}
			}
		}
	}
	m.seen = g.frame
	corpseMarks[e.ID] = m
	return m
}

// creatureNow reports whether creature id is in the current snapshot.
func (g *Game) creatureNow(id sim.CreatureID) bool {
	for _, c := range g.curr.Creatures {
		if c.ID == id {
			return true
		}
	}
	return false
}

// fbAgeCorpse makes body e seem to have fallen age frames ago (for the
// sheets; huge: long since).
func (g *Game) fbAgeCorpse(e protocol.EntityView, age int) {
	corpseMarks[e.ID] = corpseMark{at: g.frame - age, seen: g.frame, fresh: age < 60*60}
}

// --- drawing ---------------------------------------------------------------------------

// corpseShade is the dark of the ground right under a body.
var corpseShade *ebiten.Image

// drawCorpse draws body e at d: its blood, the ground darkened under it,
// the body; falling, if it fell a moment ago.
func (g *Game) drawCorpse(dst drawTarget, e protocol.EntityView, d drawable) {
	k := sim.CreatureKind(e.Var)
	m := g.corpseMarkOf(e)
	age := g.frame - m.at
	step := corpseSteps
	ox, oy := 0.0, 0.0
	if m.fresh {
		step = min(corpseSteps, age/corpseStepTicks)
		// Where the beast stood, easing onto the place the body lies.
		ox, oy = m.dx, m.dy
	}
	if !m.fresh {
		age = bloodDry / 3 // seen already lying: its blood out a while
	}
	a := corpseArtOf(k, e.Big, step)
	x, y := math.Round(d.sx+ox), math.Round(d.sy+oy)
	if !a.ok {
		g.drawCorpseSprite(dst, e, x, y, age, step)
		return
	}
	lying := corpseArtOf(k, e.Big, corpseSteps)
	pool := lying.length
	if sp := corpseSpecOf(k); sp.pool > 0 {
		pool *= sp.pool
	}
	g.drawBlood(dst, e, x+lying.wx, y, pool, age)
	if step >= corpseSteps/2 {
		drawGroundShade(dst, x, y, lying.length*0.4, math.Max(2, lying.length*0.07), 0.3*clamp01(float64(step)/corpseSteps))
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-a.s.ax, -a.s.ay)
	op.GeoM.Translate(x, y)
	drawSprite(dst, a.s, op)
}

// drawCorpseSprite draws the body of a beast with no drawing to lay down:
// its own picture, pressed low and darkened.
func (g *Game) drawCorpseSprite(dst drawTarget, e protocol.EntityView, x, y float64, age, step int) {
	k := sim.CreatureKind(e.Var)
	s, _ := g.creatureSprite(protocol.CreatureView{Kind: k, State: sim.CreatureIdle, FaceLeft: e.Big}, false)
	if s.img == nil {
		return
	}
	sp := corpseSpecOf(k)
	t := float64(step) / corpseSteps
	f := lerpf(1, sp.flat, smooth(t))
	wx := (sp.wound - 0.5) * s.w()
	if e.Big {
		wx = -wx
	}
	g.drawBlood(dst, e, x+wx, y, s.w(), age)
	drawGroundShade(dst, x, y, s.w()*0.4, math.Max(2, s.w()*0.07), 0.3*t)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.ax, -s.ay)
	op.GeoM.Scale(1, f)
	op.GeoM.Translate(x, y+2*t)
	c := float32(1 - 0.18*t)
	op.ColorScale.Scale(c, c*0.94, c*0.94, 1)
	drawSprite(dst, s, op)
}

// drawGroundShade darkens the ground in a soft ellipse rx by ry about (x, y):
// where a body presses on it.
func drawGroundShade(dst drawTarget, x, y, rx, ry, a float64) {
	if a <= 0.01 {
		return
	}
	if corpseShade == nil {
		corpseShade = genSoftDisc(64, 0.6)
	}
	op := &ebiten.DrawImageOptions{}
	const half = 32.0 / HDDensity // the disc's half width, art pixels
	op.GeoM.Translate(-half, -half)
	op.GeoM.Scale(rx/half, ry/half)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(float32(a))
	op.Filter = ebiten.FilterLinear
	drawImageD(dst, corpseShade, op, HDDensity)
}
