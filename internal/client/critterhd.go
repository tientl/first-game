package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The small and middling animals drawn in code, at HD density: the tools
// they share. A critter is drawn in art pixels (the shapes below take art
// pixels and come out HDDensity times finer) from signed-distance shapes
// (hdfig.go), so a body is one smooth lump lit as a whole, a limb turns
// about its joints, and a walk is as many frames as wanted: every pose is
// a function of a phase, not a drawing of its own.

// critterHD picks the HD animals; false brings back the density-1 art.
var critterHD = true

// SetCritterHD turns the HD drawings of the code-drawn animals on or off.
// Call it before New.
func SetCritterHD(on bool) { critterHD = on }

const cK = HDDensity

// crit is the canvas a critter is drawn on. Its drawing calls are shifted
// by (ox, oy) image pixels, so a drawing can be given room around it.
type crit struct {
	c      *canvas
	g      *gbuf // what body painted, to be lit when done (critpaint.go)
	ox, oy int
	edge   float64 // how dark a part's rim is where it lies over another
	zoom   float64 // the drawing comes out this many times bigger (a smaller cousin: less than 1)
}

// newCritZoom is newCritAt for a drawing zoom times bigger.
func newCritZoom(w, h, ox, oy int, zoom float64) *crit {
	q := newCritAt(int(math.Ceil(float64(w)*zoom)), int(math.Ceil(float64(h)*zoom)), 0, 0)
	q.ox, q.oy, q.zoom = int(math.Round(float64(ox)*cK*zoom)), int(math.Round(float64(oy)*cK*zoom)), zoom
	return q
}

func newCrit(w, h int) *crit { return newCritAt(w, h, 0, 0) }

// newCritAt is a canvas w by h art pixels whose drawing calls are shifted by
// (ox, oy) art pixels.
func newCritAt(w, h, ox, oy int) *crit {
	c := newCanvas(w*cK, h*cK)
	c.dens = HDDensity
	return &crit{c: c, ox: ox * cK, oy: oy * cK, edge: 0.34, zoom: 1}
}

// Shapes in art pixels.
func aEll(cx, cy, rx, ry float64) shape { return sEll(cx*cK, cy*cK, rx*cK, ry*cK) }
func aCap(ax, ay, bx, by, ra, rb float64) shape {
	return sCap(ax*cK, ay*cK, bx*cK, by*cK, ra*cK, rb*cK)
}
func aPoly(pts ...float64) shape {
	q := make([]float64, len(pts))
	for i, v := range pts {
		q[i] = v * cK
	}
	return sPoly(q...)
}

// aPath is a brush through pts (x, y, radius triples).
func aPath(pts ...float64) shape {
	s := aCap(pts[0], pts[1], pts[3], pts[4], pts[2], pts[5])
	for i := 3; i+5 < len(pts); i += 3 {
		s = s.or(aCap(pts[i], pts[i+1], pts[i+3], pts[i+4], pts[i+2], pts[i+5]))
	}
	return s
}

// aLeg is a limb of two bones, r0 thick at the hip, r1 at the knee and r2
// at the foot.
func aLeg(hx, hy, kx, ky, fx, fy, r0, r1, r2 float64) shape {
	return aCap(hx, hy, kx, ky, r0, r1).or(aCap(kx, ky, fx, fy, r1, r2))
}

// sBlend joins shapes into one lump, their seams filled in over k art pixels.
func sBlend(k float64, ss ...shape) shape {
	k *= cK
	out := ss[0]
	for _, b := range ss[1:] {
		a, b := out, b
		out = shape{math.Min(a.x0, b.x0) - k/4, math.Min(a.y0, b.y0) - k/4, math.Max(a.x1, b.x1) + k/4, math.Max(a.y1, b.y1) + k/4,
			func(x, y float64) float64 {
				da, db := a.d(x, y), b.d(x, y)
				h := math.Max(k-math.Abs(da-db), 0) / k
				return math.Min(da, db) - h*h*k/4
			}}
	}
	return out
}

// inside reports whether art point (x, y) lies in a.
func inside(a shape) func(x, y float64) bool {
	return func(x, y float64) bool { return a.d(x*cK, y*cK) < 0 }
}

// turned is a turned by ang about art point (cx, cy).
func (a shape) turned(cx, cy, ang float64) shape {
	if ang == 0 {
		return a
	}
	cx, cy = cx*cK, cy*cK
	sn, cs := math.Sin(ang), math.Cos(ang)
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range [4][2]float64{{a.x0, a.y0}, {a.x1, a.y0}, {a.x0, a.y1}, {a.x1, a.y1}} {
		dx, dy := p[0]-cx, p[1]-cy
		x, y := cx+dx*cs-dy*sn, cy+dx*sn+dy*cs
		x0, y0, x1, y1 = math.Min(x0, x), math.Min(y0, y), math.Max(x1, x), math.Max(y1, y)
	}
	return shape{x0, y0, x1, y1, func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		return a.d(cx+dx*cs+dy*sn, cy-dx*sn+dy*cs)
	}}
}

// turnXY turns point (x, y) by ang about (cx, cy).
func turnXY(x, y, cx, cy, ang float64) (float64, float64) {
	sn, cs := math.Sin(ang), math.Cos(ang)
	dx, dy := x-cx, y-cy
	return cx + dx*cs - dy*sn, cy + dx*sn + dy*cs
}

const (
	fillOver   = iota // a part: its rim is darkened where it lies over another
	fillFlat          // a marking: no rim
	fillOn            // a marking, only on what is drawn already
	fillBehind        // only where nothing is drawn yet
)

func (q *crit) draw(s shape, sh shader, mode int) {
	c := q.c
	z := q.zoom
	x0, y0 := max(0, int(math.Floor(s.x0*z))-1+q.ox), max(0, int(math.Floor(s.y0*z))-1+q.oy)
	x1, y1 := min(c.w-1, int(math.Ceil(s.x1*z))+1+q.ox), min(c.h-1, int(math.Ceil(s.y1*z))+1+q.oy)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*c.w + x
			has := c.px[i].A >= 128
			if mode == fillOn && !has || mode == fillBehind && has {
				continue
			}
			lx, ly := (float64(x-q.ox)+0.5)/z, (float64(y-q.oy)+0.5)/z
			d := s.d(lx, ly) * z
			if d > 0 {
				continue
			}
			p := sh(int(math.Floor(lx)), int(math.Floor(ly)), d)
			if p.A == 0 {
				continue
			}
			if mode == fillOver && d > -1.05 && has {
				p = mix(p, colInk, q.edge)
			}
			c.px[i] = p
			if q.g != nil {
				q.g.sf[i] = nil // drawn flat: not lit again
			}
		}
	}
}

// fill draws s; where its rim lies over what is already drawn it is
// darkened, the fine line between two parts.
func (q *crit) fill(s shape, sh shader) { q.draw(s, sh, fillOver) }

// flat draws s with no rim line: a marking on the part under it.
func (q *crit) flat(s shape, sh shader) { q.draw(s, sh, fillFlat) }

// mark draws s only where something is drawn already.
func (q *crit) mark(s shape, sh shader) { q.draw(s, sh, fillOn) }

// behind draws s only where nothing is drawn yet.
func (q *crit) behind(s shape, sh shader) { q.draw(s, sh, fillBehind) }

// dot colours the image pixel at art point (x, y), dx and dy image pixels off.
func (q *crit) dot(x, y float64, dx, dy int, col color.RGBA) {
	px, py := int(math.Floor(x*cK*q.zoom))+dx+q.ox, int(math.Floor(y*cK*q.zoom))+dy+q.oy
	q.c.put(px, py, col)
	if q.g != nil && q.c.in(px, py) {
		q.g.sf[py*q.c.w+px] = nil
	}
}

// eye is an eye at art point (x, y), r image pixels across its iris: a dark
// rim, the iris, a pupil and a glint on the lit side.
func (q *crit) eye(x, y, r float64, iris color.RGBA) {
	q.flat(aEll(x, y, (r+1)/cK, (r+1)/cK), plain(mix(iris, colInk, 0.85)))
	q.flat(aEll(x, y, r/cK, r/cK), plain(iris))
	if r >= 1.5 {
		q.flat(aEll(x+0.1, y, r*0.5/cK, r*0.62/cK), plain(colInk))
	}
	q.dot(x, y, -int(math.Round(r*0.5)), -int(math.Round(r*0.5)), colWhite)
}

// bead is a small dark eye with a glint.
func (q *crit) bead(x, y, r float64) {
	q.flat(aEll(x, y, r/cK, r/cK), plain(colInk))
	q.dot(x, y, -int(math.Ceil(r*0.4)), -int(math.Ceil(r*0.4)), colWhite)
}

// done outlines the drawing and hands it over with its anchor, in art pixels.
func (q *crit) done(ol, ax, ay float64) (*canvas, float64, float64) {
	if q.g != nil {
		q.light() // the parts drawn as forms: see critpaint.go
		q.outlineLit(ol)
	} else {
		q.c.outline(ol)
	}
	return q.c, ax*q.zoom + float64(q.ox)/cK, ay*q.zoom + float64(q.oy)/cK
}

// critSprite is the sprite of a critter canvas, whatever its density.
func critSprite(c *canvas, ax, ay float64) sprite {
	if c.dens > 1 && c.s == 0 {
		return sprite{img: c.image(), ax: ax, ay: ay, dens: c.dens}
	}
	return mk(c, ax, ay)
}

// --- materials -----------------------------------------------------------------

// shd shades a as a bulging surface in clean bands of r.
func shd(r ramp, a shape, round, bias float64) shader { return bevel(r, a, round*cK, bias) }

// fur is tufted fur over a.
func fur(r ramp, a shape, round, bias float64, seed uint32) shader {
	return furShade(r, a, round*cK, bias, seed)
}

// where paints lo instead of hi where pick says so (a pale belly, a muzzle).
func where(hi, lo shader, pick func(x, y float64) bool) shader {
	return func(x, y int, d float64) color.RGBA {
		if pick((float64(x)+0.5)/cK, (float64(y)+0.5)/cK) {
			return lo(x, y, d)
		}
		return hi(x, y, d)
	}
}

// scaly is a hide of scales over a: staggered rows of cells cw by ch image
// pixels, each with a dark seam below and to the right and a lit crown.
func scaly(r ramp, a shape, round, bias, cw, ch float64, seed uint32) shader {
	n := len(r)
	return func(x, y int, _ float64) color.RGBA {
		i := int((a.lum(x, y, round*cK) + bias) * float64(n))
		i = max(1, min(n-2, i))
		row := math.Floor(float64(y) / ch)
		u := float64(x)/cw + 0.5*math.Mod(row, 2)
		k := math.Floor(u)
		fx, fy := u-k, float64(y)/ch-row
		h := hashf(int(k), int(row), seed)
		switch {
		case fx > 1-1.01/cw || fy > 1-1.01/ch:
			i--
		case fx < 0.5 && fy < 0.5 && h > 0.35:
			i++
		}
		return r.at(i)
	}
}

// --- tubes ---------------------------------------------------------------------

// tpt is a point of a ctube's spine and the ctube's radius there.
type tpt struct{ x, y, r float64 }

// ctube is a body drawn along a spine, in art pixels.
type ctube struct {
	p   []tpt
	len []float64 // length along the spine up to each point
}

func newCTube(p []tpt) *ctube {
	t := &ctube{p: p, len: make([]float64, len(p))}
	for i := 1; i < len(p); i++ {
		t.len[i] = t.len[i-1] + math.Hypot(p[i].x-p[i-1].x, p[i].y-p[i-1].y)
	}
	return t
}

// at is where art point (x, y) lies on the ctube: d its distance to the
// surface (negative inside), s how far along (0 to 1), v across (-1 on the
// left of the way it runs to 1 on the right).
func (t *ctube) at(x, y float64) (d, s, v float64) {
	d = math.Inf(1)
	total := t.len[len(t.len)-1]
	for i := 0; i+1 < len(t.p); i++ {
		a, b := t.p[i], t.p[i+1]
		ex, ey := b.x-a.x, b.y-a.y
		ee := ex*ex + ey*ey
		u := 0.0
		if ee > 0 {
			u = clamp01(((x-a.x)*ex + (y-a.y)*ey) / ee)
		}
		r := lerpf(a.r, b.r, u)
		px, py := x-a.x-ex*u, y-a.y-ey*u
		if di := math.Hypot(px, py) - r; di < d {
			d = di
			s = (t.len[i] + u*math.Sqrt(ee)) / total
			v = (ex*py - ey*px) / math.Max(1e-6, math.Sqrt(ee)*r)
		}
	}
	return d, s, v
}

func (t *ctube) shape() shape {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range t.p {
		x0, y0, x1, y1 = math.Min(x0, p.x-p.r), math.Min(y0, p.y-p.r), math.Max(x1, p.x+p.r), math.Max(y1, p.y+p.r)
	}
	return shape{x0 * cK, y0 * cK, x1 * cK, y1 * cK, func(x, y float64) float64 {
		d, _, _ := t.at(x/cK, y/cK)
		return d * cK
	}}
}

// skin shades the ctube: f gets how far along and across the pixel is and
// the light on it there.
func (t *ctube) skin(f func(x, y int, s, v, l float64) color.RGBA) shader {
	sh := t.shape()
	return func(x, y int, _ float64) color.RGBA {
		_, s, v := t.at((float64(x)+0.5)/cK, (float64(y)+0.5)/cK)
		return f(x, y, s, v, sh.lum(x, y, 2.2*cK))
	}
}

// bez is n+1 points along the curve through control points a, b, c, d, the
// radius going from r0 to r1.
func bez(n int, r0, r1 float64, a, b, c, d [2]float64) []tpt {
	out := make([]tpt, n+1)
	for i := range out {
		t := float64(i) / float64(n)
		u := 1 - t
		out[i] = tpt{
			u*u*u*a[0] + 3*u*u*t*b[0] + 3*u*t*t*c[0] + t*t*t*d[0],
			u*u*u*a[1] + 3*u*u*t*b[1] + 3*u*t*t*c[1] + t*t*t*d[1],
			lerpf(r0, r1, t),
		}
	}
	return out
}

// --- limbs and gaits ---------------------------------------------------------------

// ik2 is the knee of a leg of two bones l1 and l2 long from hip to foot,
// bent to the side bend (1 or -1); a leg too short to reach is stretched out
// straight.
func ik2(hx, hy, fx, fy, l1, l2, bend float64) (kx, ky float64) {
	dx, dy := fx-hx, fy-hy
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		return hx + l1*bend, hy
	}
	d2 := math.Min(d, (l1+l2)*0.999)
	a := (l1*l1 - l2*l2 + d2*d2) / (2 * d2)
	h := math.Sqrt(math.Max(0, l1*l1-a*a))
	ux, uy := dx/d, dy/d
	return hx + ux*a - uy*h*bend, hy + uy*a + ux*h*bend
}

// pace is where a foot is at phase ph of a stride: on the ground it slides
// back from stride/2 ahead to stride/2 behind while ph < duty, then swings
// forward again, lifted.
func pace(ph, duty, stride, lift float64) (dx, up float64) {
	ph -= math.Floor(ph)
	if ph < duty {
		return stride * (0.5 - ph/duty), 0
	}
	t := (ph - duty) / (1 - duty)
	return stride * (ease(t) - 0.5), lift * math.Sin(math.Pi*t)
}

// --- the frames --------------------------------------------------------------------

// critKind is one animal: its old density-1 drawing and its HD one. Frame 0
// of a pose is the pose itself, as the density-1 art had it; the frames
// after it are that pose moving (cycFrames of a walk or a run, blowFrames of
// a blow being wound up and thrown).
type critKind struct {
	poses int
	old   func(pose int) (*canvas, float64, float64)
	hd    func(pose, f int) (*canvas, float64, float64)
	anim  map[int]int // pose: how many frames beyond 0 it has
}

type critKey struct {
	kind          string
	side, pose, f int
}

var critCache = map[critKey]sprite{}

// critter is frame f of pose of a kind, facing side, drawn the first time
// it is wanted.
func critter(kind string, side, pose, f int) sprite {
	key := critKey{kind, side, pose, f}
	if s, ok := critCache[key]; ok {
		return s
	}
	var s sprite
	k := critKinds[kind]
	if !critterHD || f > k.anim[pose] {
		f = 0
	}
	switch {
	case side == 1:
		s = mirrorSprite(critter(kind, 0, pose, f))
	case !critterHD:
		s = critSprite(k.old(pose))
	default:
		s = critSprite(k.hd(pose, f))
	}
	critCache[key] = s
	return s
}

// cycFrames is how many frames a walk or a run has.
const cycFrames = gaitFrames

// cyc is the frame of a stride at frame count t; period is how many frames a
// step (half a stride) takes.
func cyc(t, period int) int { return 1 + (t*cycFrames/(2*period))%cycFrames }

// cycPh is the phase of stride frame f (1 to cycFrames), 0 to 1.
func cycPh(f int) float64 { return float64(f-1) / cycFrames }

// blowFrames is how many frames a blow has before it lands: drawing back,
// drawn back, and two of the lunge.
const blowFrames = 4

// blowOf is the frame of a blow age frames old, drawn back for wind frames:
// wound up first, then thrown; 0 once it has landed.
func blowOf(age, wind int) int {
	switch {
	case age < 0 || age >= wind+4:
		return 0
	case age < 2:
		return 1
	case age < wind:
		return 2
	case age < wind+2:
		return 3
	}
	return 4
}

// blowAmt is how far blow frame f is thrown: -1 drawn right back, 1 landed.
func blowAmt(f int) float64 { return [blowFrames + 1]float64{1, -0.55, -1, -0.1, 0.6}[f] }

// blowClock is when a creature began the blow it is making, in which state,
// and when it was last seen at it.
type blowClock struct {
	at, seen int
	state    sim.CreatureState
}

var blowClocks = map[sim.CreatureID]blowClock{}

// blowAge is how many frames ago c went into the state it is in (a bite, a
// swipe, a spit): strikeAge for any blow.
func (g *Game) blowAge(c protocol.CreatureView) int {
	if c.ID == 0 {
		return -1 // a body or a trophy: the blow as it lands
	}
	if len(blowClocks) > 512 {
		clear(blowClocks)
	}
	k, ok := blowClocks[c.ID]
	if !ok || k.state != c.State || g.frame < k.at || g.frame-k.seen > 6 {
		k.at, k.state = g.frame, c.State
	}
	k.seen = g.frame
	blowClocks[c.ID] = k
	return g.frame - k.at
}

// shag is a coat of hanging hair over a: strands two image pixels wide and
// about long image pixels long, lit where they begin and dark under their tips.
func shag(r ramp, a shape, round, bias, long float64, seed uint32) shader {
	n := len(r)
	return func(x, y int, _ float64) color.RGBA {
		i := int((a.lum(x, y, round*cK) + bias) * float64(n))
		i = max(1, min(n-2, i))
		k := (x + y/5) / 2
		v := (float64(y) + hashf(k, 0, seed)*long*3) / long
		seg := math.Floor(v)
		h := hashf(k, int(seg), seed+1)
		switch t := v - seg; {
		case h < 0.34 && t > 0.55:
			i--
		case h > 0.6 && t < 0.5:
			i++
		}
		return r.at(i)
	}
}

// blowF is the frame of the blow c is making. A beast the sim makes draw
// back before its blow lands (sim/telegraph.go) is held drawn back as long,
// so the lunge meets the hit; the others strike at once, and their wind-up
// is only shown.
func (g *Game) blowF(c protocol.CreatureView) int {
	wind := max(7, sim.WindupTicks(c.Kind)*60/sim.TicksPerSecond-2)
	return blowOf(g.blowAge(c), wind)
}

// critterSprite is creatureSprite for the code-drawn animals when they
// move: the frames of their walks, runs and blows. ok is false for a pose
// that stands still (the art holds those) and for every other kind.
func (g *Game) critterSprite(c protocol.CreatureView, moving bool, side, t int) (sprite, float64, bool) {
	if !critterHD {
		return sprite{}, 0, false
	}
	fr := func(kind string, pose, f int) (sprite, float64, bool) { return critter(kind, side, pose, f), 0, true }
	switch c.Kind {
	case sim.CreatureRabbit:
		if !moving || c.State == sim.CreatureSleep {
			break
		}
		period := 16
		if c.State == sim.CreatureFlee {
			period = 9
		}
		ph := float64(t%period) / float64(period)
		bob := 3 * math.Sin(math.Pi*clamp01((ph-0.1)/0.66)) // off the ground from the push to the landing
		f := 1 + int(ph*cycFrames)%cycFrames                 // see hopKeys in critsmall.go
		return critter("rabbit", side, rabbitHop, f), bob, true
	case sim.CreatureSnake:
		switch {
		case c.State == sim.CreatureStrike:
			return fr("snake", snakeStrike, g.blowF(c))
		case moving && c.State != sim.CreatureIdle:
			return fr("snake", 0, 1+(t/3)%cycFrames)
		case moving:
			return fr("snake", 0, 1+(t/3)%cycFrames)
		}
	case sim.CreatureWorm:
		return fr("worm", 0, 1+(t/5)%cycFrames)
	case sim.CreatureFish:
		if g.art.tribal.fish != nil {
			return fr(fishKind(int(c.ID)%len(fishLooks)), 0, 1+(t/4)%cycFrames)
		}
		return fr("fish", 0, 1+(t/4)%cycFrames)
	case sim.CreatureCrab:
		switch {
		case c.State == sim.CreatureStrike:
			return critter("crab", 0, crabPinch, 1+(t/4)%4), 0, true
		case c.State == sim.CreatureChase:
		case moving:
			return critter("crab", 0, crabWalkA, cyc(t, 4)), 0, true
		}
	case sim.CreatureCroc, sim.CreatureBrachy, sim.CreatureChampso:
		kind := map[sim.CreatureKind]string{sim.CreatureCroc: "croc", sim.CreatureBrachy: "brachy", sim.CreatureChampso: "champso"}[c.Kind]
		switch {
		case c.State == sim.CreatureLurk:
		case c.State == sim.CreatureStrike:
			return fr(kind, crocBite, g.blowF(c))
		case moving && c.State == sim.CreatureChase:
			return fr(kind, crocWalkB, cyc(t, 4))
		case moving:
			return fr(kind, crocWalkA, cyc(t, 9))
		}
	case sim.CreatureMammoth:
		switch {
		case feeding(c):
		case c.State == sim.CreatureStrike:
			return fr("mammoth", mammothGore, g.blowF(c))
		case calling(c):
		case c.State == sim.CreatureChase && moving:
			return fr("mammoth", mammothCharge, cyc(t, 6))
		case c.State == sim.CreatureChase:
		case moving:
			return fr("mammoth", mammothWalkA, cyc(t, 14))
		}
	case sim.CreatureSabre:
		switch {
		case feeding(c):
		case c.State == sim.CreatureStrike:
			f := g.blowF(c)
			return critter("sabre", side, sabreSpring, f), [blowFrames + 1]float64{3, 0, 0, 0, 2}[f], true
		case c.State == sim.CreatureCower:
		case c.State == sim.CreatureChase && moving:
			return fr("sabre", sabreStalk, cyc(t, 6))
		case c.State == sim.CreatureChase:
		case moving && c.State == sim.CreatureFlee:
			return fr("sabre", sabreWalkB, cyc(t, 4))
		case moving:
			return fr("sabre", sabreWalkA, cyc(t, 9))
		}
	case sim.CreatureSpider:
		switch {
		case c.State == sim.CreatureSpit:
			return fr("spider", spiderSpit, g.blowF(c))
		case c.State == sim.CreatureStrike:
			return fr("spider", spiderBite, g.blowF(c))
		case moving:
			return fr("spider", spiderWalkA, cyc(t, 3))
		}
	case sim.CreatureBear:
		switch {
		case c.State == sim.CreatureClaw:
			return fr("bear", bearClaw, g.blowF(c))
		case c.State == sim.CreatureStrike:
			return fr("bear", bearBite, g.blowF(c))
		case moving && c.State == sim.CreatureChase:
			return fr("bear", bearWalkB, cyc(t, 5))
		case moving:
			return fr("bear", bearWalkA, cyc(t, 10))
		}
	case sim.CreatureTribesman:
		switch {
		case c.State == sim.CreatureStrike:
			return fr("tribe", tribeThrust, g.blowF(c))
		case moving && (c.State == sim.CreatureChase || c.State == sim.CreatureFlee):
			return fr("tribe", tribeWalkB, cyc(t, 4))
		case moving:
			return fr("tribe", tribeWalkA, cyc(t, 8))
		}
	case sim.CreatureScorpion:
		switch {
		case c.State == sim.CreatureClaw:
			return fr("scorpion", scorpionPinch, g.blowF(c))
		case c.State == sim.CreatureStrike:
			return fr("scorpion", scorpionSting, g.blowF(c))
		case moving && c.State == sim.CreatureChase:
			return fr("scorpion", scorpionWalkA, cyc(t, 4))
		case moving:
			return fr("scorpion", scorpionWalkA, cyc(t, 8))
		}
	case sim.CreatureMonkey:
		switch {
		case feeding(c):
		case moving && c.State == sim.CreatureFlee:
			f := cyc(t, 5)
			return critter("monkey", side, monkeyLeap, f), 1.5 * (1 + math.Cos(2*math.Pi*cycPh(f))), true
		case moving:
			return fr("monkey", monkeyWalkA, cyc(t, 7))
		}
	}
	return sprite{}, 0, false
}
