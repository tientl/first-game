package client

import (
	"image/color"
	"math"

	"survivalgame/internal/sim"
)

// The player with joints: a figure five and a half heads tall, its arms and
// legs built of segments that turn about a shoulder, an elbow and a wrist, a hip, a
// knee and an ankle, as the dinosaurs' legs are (dinolegs.go). A pose is
// angles and foot targets in the figure's own space (sideways, forwards,
// up); each facing is a projection of it, so one walk serves the side, the
// front and the back.
//
// A frame is drawn in two halves that are put together when shown: the legs
// (a walk or a run, sixteen steps of the stride each) and everything above
// them (the arms through a stride, or a blow). The legs say how far the
// hips drop and the body leans, so any arms go with any legs. Both are drawn
// the first time they are shown and kept (limbdraw.go).
//
// Everything is in image pixels: the figure's middle on column 27 and its
// soles on row limbGround, on a canvas with room round it for a stride and
// a thrown fist. The body and the head are drawn in tallfig.go and
// tallhead.go.

// playerJointed picks the jointed figure; off, the player is the stiff
// HD figure of lookhd.go with its eight walk frames.
var playerJointed = true

const (
	limbPad   = 11 // art pixels of room on each side of the body's 18
	limbBelow = 7  // and under the soles: a foot stepped towards the viewer
	limbFeet  = 40 // the anchor's row, in art pixels: the soles are just above it
	limbW     = (bodyW + 2*limbPad) * HDDensity
	limbH     = (limbFeet + limbBelow) * HDDensity
	limbAX    = 9 + limbPad // the anchor's column, in art pixels

	// The figure is five and a half heads tall: 93 image pixels from the
	// crown (row 27) to the soles (row limbGround), the hips half way.
	limbGround = limbFeet * HDDensity
	upArmLen   = 14.0 // shoulder to elbow
	foreLen    = 12.5 // elbow to wrist
	handOff    = 4.5  // wrist to the middle of the fist
	thighLen   = 21.5 // hip to knee
	shinLen    = 20.5 // knee to ankle
	shoulderY  = 49.5
	hipY       = 73.3
	ankleY     = limbGround - 5.0 // of a foot on the ground
	legReach   = 0.995 * (thighLen + shinLen)

	// groundK is how tall a step towards the viewer is on the screen: a tile
	// is TileH tall and TileW wide.
	groundK = float64(TileH) / float64(TileW)

	// armK is the same for an arm swung back from the viewer, or held out
	// away from them: less than the ground's, or it would all but vanish.
	armK = 0.6

	// legK is the same for a step seen from the front or the back. A stride
	// drawn at the ground's own scale would take the foot behind the body a
	// third of a leg's length below the ground line, the leg stretched to
	// reach it; so a step is drawn shorter, and set further from the viewer
	// (legBack): the nearest foot comes no more than an art pixel below the
	// line the figure stands on. The price is that a foot seen from the
	// front or the back goes under the body slower than the ground does.
	// Sideways it is planted exactly.
	legK    = 0.1
	legBack = 0.9 // of a foot's reach

	upFrames  = 8  // arm frames in a stride
	loFrames  = 16 // leg frames in a stride
	atkFrames = 6  // frames of a blow, from wound up to landed
)

// armJ is an arm's pose: the upper arm swung sh radians forwards from
// hanging, the elbow bent el more, the whole arm lifted ab out to the side.
// A blow pushes the shoulder out pixels forwards: the body turns into it.
type armJ struct{ sh, el, ab, out float64 }

// legJ is a leg's pose: the ankle f pixels ahead of the hip and u above the
// ground, the foot pitched toe-down by pitch.
type legJ struct{ f, u, pitch float64 }

// upAnim names what the arms do.
type upAnim uint8

const (
	upStand    upAnim = iota
	upWalk            // upFrames of a walking stride
	upRun             // and of a running one
	upPunch           // atkFrames from cocked to straight, the hand that holds things
	upPunchOff        // the same with the other hand
	upChop            // an axe, a pick, a hoe or a sword: from over the shoulder to the target
	upThrust          // a spear: from drawn back to driven home
	upFling           // both arms thrown out in a start
	upFlingOff        // only the empty one: the other holds a tool
	upSwim            // paddling, four frames
	upAsleep          // standing still with the eyes shut: laid in a bed
	upFallen          // the eyes shut and one arm thrown up: laid on the ground, dead (bodyart.go)
)

// loAnim names what the legs do.
type loAnim uint8

const (
	loStand loAnim = iota
	loWalk         // loFrames of a walking stride
	loRun
	loSit // astride a mount
)

// upMotion and loMotion are where the motions of motion.go begin among
// the arms' and the legs' animations: motion m is upMotion+m and
// loMotion+m, the same frame in both halves.
const (
	upMotion upAnim = 64
	loMotion loAnim = 64
)

type upKey struct {
	dir   sim.Dir
	anim  upAnim
	frame uint8 // with upSit set astride a mount: the skirt lies over the thighs
}

const upSit = 0x80

type loKey struct {
	dir   sim.Dir
	anim  loAnim
	frame uint8
}

// gaitDef is a way of going: cycle art pixels of ground to a stride of both
// legs, each foot down for duty of it and lifted lift image pixels between.
type gaitDef struct {
	cycle, duty, lift  float64
	crouch, bounce     float64 // the hips' drop, and how far it rises in the air, image pixels
	arm, elbow, elbow1 float64 // the arms' swing, their bend, and how much more when forwards
	lean               float64 // radians the body leans forwards
}

var (
	stepGait = gaitDef{cycle: 19, duty: 0.6, lift: 5, arm: 0.4, elbow: 0.15, elbow1: 0.3}
	runGait  = gaitDef{cycle: 46, duty: 0.32, lift: 13, crouch: 3.5, bounce: 2.5, arm: 0.9, elbow: 1.3, elbow1: 0.25, lean: 0.12}
)

// half is how far ahead of the hip a foot comes down, in image pixels: it
// leaves the ground as far behind.
func (gd gaitDef) half() float64 { return gd.cycle * gd.duty * HDDensity / 2 }

// footLift is how far an ankle must rise for a foot pitched by p to stay on
// the ground: it turns on its toe, or on its heel.
func footLift(p float64) float64 {
	return math.Max(0, math.Max(7*math.Sin(p), -4.8*math.Sin(p))+5*(math.Cos(p)-1))
}

// armOut is how far a hanging arm is held out from the body, in radians;
// armIn how far the forearm comes back in from that, and armBend how far
// the elbow is bent at rest: an arm never hangs quite straight.
const (
	armOut  = 0.17
	armIn   = 0.23
	armBend = 0.2
)

const toeOff = 0.5 // the pitch of a foot as it leaves the ground

// leg is one leg p of the way through its stride, from its foot coming down.
// While it is down the ankle goes back at the ground's own speed: the foot is
// planted.
func (gd gaitDef) leg(p float64) legJ {
	p -= math.Floor(p)
	a := gd.half()
	if p < gd.duty {
		s := p / gd.duty
		l := legJ{f: a - 2*a*s}
		switch {
		case s < 0.15: // on the heel
			l.pitch = -0.3 * (1 - s/0.15)
		case s > 0.75: // rolling onto the toe
			l.pitch = toeOff * (s - 0.75) / 0.25
		}
		l.u = footLift(l.pitch)
		return l
	}
	s := (p - gd.duty) / (1 - gd.duty)
	pitch := toeOff - (toeOff+0.3)*s
	return legJ{f: -a + 2*a*smooth(s), u: footLift(pitch) + gd.lift*math.Sin(math.Pi*s), pitch: pitch}
}

// stanceAt reports whether a leg p through its stride is on the ground.
func (gd gaitDef) stanceAt(p float64) bool { return p-math.Floor(p) < gd.duty }

// legs is both legs ph through the stride and how far the hips have dropped
// for them: as far as the gait has it, and as far as the reach of a planted
// leg needs.
func (gd gaitDef) legs(ph float64) (l [2]legJ, drop float64) {
	drop = math.Round(gd.crouch + gd.bounce*math.Cos(4*math.Pi*(ph-gd.duty/2)))
	for i := range l {
		l[i] = gd.leg(ph + 0.5*float64(i))
		drop = math.Max(drop, math.Ceil(legNeed(l[i])-1e-6))
	}
	return l, drop
}

// legNeed is how far the hips must drop for a leg to reach its ankle.
func legNeed(l legJ) float64 {
	if math.Abs(l.f) >= legReach {
		return ankleY - hipY
	}
	return ankleY - hipY - l.u - math.Sqrt(legReach*legReach-l.f*l.f)
}

// arms is both arms ph through the stride: each swings against the leg on
// its own side.
func (gd gaitDef) arms(ph float64) (a [2]armJ) {
	for i := range a {
		th := 2 * math.Pi * (ph + 0.5*float64(i) - gd.duty/2)
		s := math.Sin(th)
		a[i] = armJ{sh: gd.arm * s, el: gd.elbow + gd.elbow1*math.Max(0, s), ab: armOut}
		if gd.elbow > 1 {
			a[i].sh += 0.15
			a[i].el = gd.elbow + gd.elbow1*s
		}
	}
	return a
}

// lowerPose is the legs of a frame.
type lowerPose struct {
	leg    [2]legJ // on the hand side (the near one sideways), and the other
	drop   float64 // the hips' drop, in image pixels
	lean   float64
	reach  float64 // how far ahead of the hip a foot comes down: see legK
	spread float64 // seen from the front or the back, how far each knee goes out to its side, image pixels
	tuck   [2]bool // seen from the front or the back, the foot is hidden behind its knee: kneeling
}

func lowerPoseOf(k loKey) lowerPose {
	if k.anim >= loMotion {
		return motionPose(motionID(k.anim-loMotion), int(k.frame), k.dir).lo
	}
	ph := (float64(k.frame) + 0.5) / loFrames
	switch k.anim {
	case loWalk:
		l, drop := stepGait.legs(ph)
		return lowerPose{leg: l, drop: drop, reach: stepGait.half()}
	case loRun:
		l, drop := runGait.legs(ph)
		return lowerPose{leg: l, drop: drop, lean: runGait.lean, reach: runGait.half()}
	case loSit:
		return lowerPose{leg: [2]legJ{{f: 15, u: 13, pitch: 0.3}, {f: 18, u: 14, pitch: 0.3}}}
	}
	return lowerPose{}
}

// upperPose is the arms of a frame, and how far what hangs from the figure
// trails behind it.
type upperPose struct {
	arm        [2]armJ // the hand that holds things, and the other
	lagX, lagY float64
	shut       bool

	// What the trunk and the head do (motion.go): drawn bent, not skewed.
	tilt   float64 // the trunk leans forwards from the hips, radians; back if negative
	nod    float64 // the head bows on the neck, radians; thrown back if negative
	roll   float64 // seen from the front or the back, the trunk sways towards the hand's side, radians
	jaw    float64 // the jaw dropped, image pixels: biting, chewing
	open   [2]bool // a hand open, not a fist
	breath float64 // image pixels the chest rises with a breath
	fore   [2]bool // a hand up before the face: over the head, or hidden by it from behind
}

func upperPoseOf(k upKey) upperPose {
	k.frame &^= upSit
	if k.anim >= upMotion {
		return motionPose(motionID(k.anim-upMotion), int(k.frame), k.dir).up
	}
	side := k.dir == sim.DirRight
	rest := armJ{el: armBend, ab: armOut}
	p := upperPose{arm: [2]armJ{rest, rest}}
	u := float64(k.frame) / (atkFrames - 1)
	stride := func(gd gaitDef, back float64) {
		ph := (float64(k.frame) + 0.5) / upFrames
		p.arm = gd.arms(ph)
		_, d0 := gd.legs(ph - 1.5/loFrames)
		_, d1 := gd.legs(ph)
		p.lagY = max(-2, min(2, d0-d1)) // where the body was a moment ago
		if side {
			p.lagX = -back
		} else {
			p.lagX = math.Round(back * 0.75 * math.Sin(2*math.Pi*ph))
		}
	}
	switch k.anim {
	case upWalk:
		stride(stepGait, 1)
	case upRun:
		stride(runGait, 2)
	case upPunch, upPunchOff, upThrust:
		blow := armJ{sh: lerpf(-0.9, 1.45, u), el: lerpf(1.9, 0.08, u), out: 5 * u}
		if !side {
			blow.ab = -0.35 * u
		}
		guard := armJ{sh: lerpf(0.35, -0.45, u), el: 1.1}
		p.arm = [2]armJ{blow, guard}
		if k.anim == upPunchOff {
			p.arm = [2]armJ{guard, blow}
		}
		p.lagX = math.Round(-2 * (u - 0.5))
	case upChop:
		if side {
			p.arm[0] = armJ{sh: lerpf(2.75, 0.7, u), el: lerpf(1.0, 0.2, u), out: 4 * u}
			p.arm[1] = armJ{sh: lerpf(1.9, 0.9, u), el: lerpf(0.9, 0.5, u)}
		} else {
			p.arm[0] = armJ{sh: lerpf(2.6, 0.55, u), el: lerpf(0.9, 0.25, u), ab: lerpf(0.25, 0.7, u)}
			p.arm[1] = armJ{sh: 0.3 * (1 - u), el: 0.6, ab: 0.15}
		}
		p.lagX, p.lagY = math.Round(-2*(u-0.4)), math.Round(2*(u-0.5))
	case upFling, upFlingOff:
		fl := armJ{sh: 0.2, el: 0.3, ab: 2.0}
		if side {
			fl = armJ{sh: -2.3, el: 0.3}
		}
		p.arm[1] = fl
		if k.anim == upFling {
			p.arm[0] = fl
		}
		p.lagX, p.lagY = -2, -2
	case upSwim:
		s := math.Sin(2 * math.Pi * float64(k.frame) / 4)
		for i := range p.arm {
			p.arm[i] = armJ{sh: 0.5 + 0.3*s, el: 0.5, ab: 1.15 - 0.25*s}
			s = -s
		}
	case upAsleep:
		p.shut = true
	case upFallen:
		p.shut = true
		p.arm[1] = armJ{sh: 0.2, el: 0.5, ab: 2.3}
	}
	if !side {
		p.lagY = max(-1, min(1, p.lagY))
	}
	return p
}

// sRot is a turned by ang about (cx, cy): clockwise on the screen.
func sRot(a shape, cx, cy, ang float64) shape {
	if ang == 0 {
		return a
	}
	sn, cs := math.Sin(ang), math.Cos(ang)
	r := 0.0
	for _, c := range [4][2]float64{{a.x0, a.y0}, {a.x1, a.y0}, {a.x0, a.y1}, {a.x1, a.y1}} {
		r = math.Max(r, math.Hypot(c[0]-cx, c[1]-cy))
	}
	return shape{cx - r, cy - r, cx + r, cy + r, func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		return a.d(cx+dx*cs+dy*sn, cy-dx*sn+dy*cs)
	}}
}

// newLimbGen is a figGen on the jointed figure's canvas: lookhd.go's frame
// with room round it.
func newLimbGen(l sim.Look, wear sim.Wear) *figGen {
	g := newFigGen(l, limbW, limbH)
	g.wear = wear
	g.FA, g.A, g.AO, g.TO = newFigLayer(limbW, limbH), newFigLayer(limbW, limbH), newFigLayer(limbW, limbH), newFigLayer(limbW, limbH)
	g.U = newFigLayer(limbW, limbH)
	g.f.ox = limbPad * HDDensity
	return g
}

// --- the legs ----------------------------------------------------------------

// legGeom is a leg's joints on the screen.
type legGeom struct {
	hx, hy, kx, ky, ax, ay float64
	pitch                  float64
	tuck                   bool // the foot is behind the knee: not drawn
}

// hipX is the column of a leg's hip facing d: the hand side's, or the other's.
func hipX(d sim.Dir, i int) float64 {
	if d == sim.DirRight {
		return [2]float64{25.5, 28.5}[i]
	}
	return [2]float64{31.5, 22.5}[i]
}

// legGeo solves leg i of pose p facing d: the knee is where the thigh and
// the shin meet, bent forwards.
func legGeo(d sim.Dir, i int, p lowerPose) legGeom {
	l := p.leg[i]
	hu := ankleY - hipY - p.drop // the hip's height over a planted ankle
	dx, dy := l.f, l.u-hu
	dist := math.Hypot(dx, dy)
	if dist > legReach { // out of reach: as far as it goes
		dx, dy = dx*legReach/dist, dy*legReach/dist
		dist = legReach
	}
	dist = math.Max(dist, 1)
	along := (thighLen*thighLen - shinLen*shinLen + dist*dist) / (2 * dist)
	out := math.Sqrt(math.Max(0, thighLen*thighLen-along*along))
	kf := dx/dist*along - dy/dist*out
	ku := hu + dy/dist*along + dx/dist*out
	x0 := hipX(d, i)
	proj := func(f, u float64) (float64, float64) {
		switch d {
		case sim.DirRight:
			return x0 + f, ankleY - u
		case sim.DirUp:
			return x0, ankleY - u - legK*(f+legBack*p.reach)
		}
		return x0, ankleY - u + legK*(f-legBack*p.reach)
	}
	var g legGeom
	g.hx, g.hy = proj(0, hu)
	g.kx, g.ky = proj(kf, ku)
	g.ax, g.ay = proj(dx, hu+dy)
	if d != sim.DirRight && p.spread != 0 { // the knees part, the feet less
		out := p.spread * (1 - 2*float64(i))
		g.kx += out
		g.ax += out * 0.45
	}
	g.tuck = d != sim.DirRight && p.tuck[i]
	if d == sim.DirRight {
		g.pitch = l.pitch
	}
	return g
}

// leg draws one leg on L, facing view (never DirLeft): bare and barefoot
// (cavefig.go), or with what is worn: the trouser leg down over the knee and
// the cross-bound wraps of the shin, the boot and its fur cuff. dim darkens
// the far leg; toe is how far the toe points sideways. From the side a
// foot is seen at its full length.
func (g *figGen) leg(lg legGeom, dim, toe float64, view sim.Dir) {
	if g.wear&sim.WearLegWraps != 0 {
		g.wrappedLeg(lg, dim)
	} else {
		g.bareLeg(lg, dim, view)
	}
	switch {
	case lg.tuck:
	case g.wear&sim.WearBoots != 0:
		g.boot(lg, dim, toe, view == sim.DirRight)
	default:
		if g.wear&sim.WearLegWraps != 0 {
			g.f.n++ // a part of its own under the wraps
		}
		g.bareFoot(lg, dim, toe, view)
	}
}

// wrappedLeg draws the trouser leg and the wraps of a leg.
func (g *figGen) wrappedLeg(lg legGeom, dim float64) {
	f := &g.f
	rt0, rt1, rs0, rs1 := 5.2, 4.3, 4.1, 3.3
	if g.look.Female {
		rt0, rt1, rs0, rs1 = 4.6, 3.8, 3.5, 2.9
	}
	sx, sy := lg.ax-lg.kx, lg.ay-lg.ky
	n := math.Max(math.Hypot(sx, sy), 0.01)
	sx, sy = sx/n, sy/n
	k := math.Min(4.5, n/2)
	tr := sCap(lg.hx, lg.hy, lg.kx, lg.ky, rt0, rt1).or(sCap(lg.kx, lg.ky, lg.kx+sx*k, lg.ky+sy*k, rt1, rs0+0.3))
	f.fill(g.L, tr, func(px, py int, _ float64) color.RGBA {
		v := tr.lum(px, py, 4) - dim
		if math.Hypot(float64(px)+0.5-lg.kx-1, float64(py)+0.5-lg.ky) < 2 {
			v += 0.12 // the knee
		}
		return toneOf(trouserHD, v)
	})

	wr := sCap(lg.kx+sx*k, lg.ky+sy*k, lg.ax, lg.ay, rs0, rs1)
	f.fill(g.L, wr, func(px, py int, _ float64) color.RGBA {
		i := int((wr.lum(px, py, 4) - dim + 0.04) * 6)
		qx, qy := float64(px)+0.5-lg.kx, float64(py)+0.5-lg.ky
		a, b := int(math.Floor(qx*sy-qy*sx))+60, int(math.Floor(qx*sx+qy*sy))+60
		if (a+b)%6 == 0 || (a-b+60)%6 == 0 {
			i -= 2 // the bindings, crossed
		}
		return wrapHD.at(max(i, 0))
	})
}

// boot draws a leg's boot and its fur cuff.
func (g *figGen) boot(lg legGeom, dim, toe float64, side bool) {
	f := &g.f
	sx, sy := lg.ax-lg.kx, lg.ay-lg.ky
	ax, ay := math.Round(lg.ax*2)/2, lg.ay
	sn, cs := math.Sin(lg.pitch), math.Cos(lg.pitch)
	hw, tw, cw := 3.9, 4.4, 4.6 // half the boot's width, its toe's and its cuff's
	if side {
		hw, tw, cw = 4.8, 5.6, 5
	}
	boot := sRot(sBox(ax-hw, ay-0.6, ax+hw, ay+5, 2.4).or(sEll(ax+toe, ay+2.7, tw, 2.5)).and(sAbove(ay+5)), ax, ay, lg.pitch)
	f.fill(g.L, boot, func(px, py int, _ float64) color.RGBA {
		dx, dy := float64(px)+0.5-ax, float64(py)+0.5-ay
		if -dx*sn+dy*cs >= 4 {
			return bootHD[0] // the sole
		}
		return toneOf(bootHD, boot.lum(px, py, 3.5)-dim-0.06)
	})
	cuff := sRot(sBox(ax-cw, ay-4.4, ax+cw, ay+0.6, 2.4).jag(1.3, 3.6, ay-2, 5), ax, ay, -math.Atan2(sx, sy)*0.7)
	f.fill(g.L, cuff, furShade(furHD, cuff, 3, -dim-0.02, 11))
}

// legs draws both legs of pose p facing d (never DirLeft) on L.
func (g *figGen) legs(d sim.Dir, p lowerPose) {
	a, b := legGeo(d, 0, p), legGeo(d, 1, p)
	switch {
	case d == sim.DirRight:
		g.leg(b, 0.16, 2.8, d) // the far one first
		g.leg(a, 0, 2.8, d)
	case a.ay < b.ay: // the one nearer the viewer over the other
		g.leg(a, 0.1, 1.4, d)
		g.leg(b, 0, -1.4, d)
	default:
		g.leg(b, 0, -1.4, d)
		g.leg(a, 0.1, 1.4, d)
	}
}

// skirt is how the legs carry the hem of a tunic or a dress: the points it
// hangs from in front and behind when seen from the side, and how far each
// side of it is lifted or let down when seen from the front. The zero skirt
// hangs straight.
type skirt struct {
	set            bool
	fx, fy, bx, by float64    // sideways: the thighs' foremost and hindmost points at the hem's height, and how far each has risen
	dy             [2]float64 // from the front: the hem's shift over the hand's leg and the other
}

// skirtAt is the point of leg lg the hem hangs from, t of the way down the
// thigh, in the upper half's own pixels.
func skirtAt(lg legGeom, t, drop float64) (x, y float64) {
	return lerpf(lg.hx, lg.kx, t), lerpf(lg.hy, lg.ky, t) - drop
}

// skirtOf is the skirt of the upper half k, its hem hanging from t of the
// way down the thigh: swung by a stride's legs, laid over a rider's thighs.
func skirtOf(k upKey, t float64) skirt {
	var p lowerPose
	switch {
	case k.frame&upSit != 0:
		p = lowerPoseOf(loKey{dir: k.dir, anim: loSit})
	case k.anim == upWalk || k.anim == upRun:
		gd := stepGait
		if k.anim == upRun {
			gd = runGait
		}
		p.leg, p.drop = gd.legs((float64(k.frame&^upSit) + 0.5) / upFrames)
		p.reach = gd.half()
	case k.anim >= upMotion:
		p = motionPose(motionID(k.anim-upMotion), int(k.frame&^upSit), k.dir).lo
		if p == (lowerPose{}) {
			return skirt{}
		}
	default:
		return skirt{}
	}
	_, y0 := skirtAt(legGeo(k.dir, 0, lowerPose{}), t, 0)
	s := skirt{set: true, fx: -sFar, bx: sFar}
	for i := 0; i < 2; i++ {
		x, y := skirtAt(legGeo(k.dir, i, p), t, p.drop)
		if x > s.fx {
			s.fx, s.fy = x, y-y0
		}
		if x < s.bx {
			s.bx, s.by = x, y-y0
		}
		s.dy[i] = math.Round(max(-5, min(4, 0.7*(y-y0))))
	}
	return s
}

// --- the arms ----------------------------------------------------------------

// Where an arm is drawn among the layers.
const (
	armBehind = iota // behind the body: the far arm, sideways
	armMid           // over the clothes, under the mantle and the head
	armOver          // over everything: raised, or thrown at the viewer
)

// armGeom is an arm's joints on the screen.
type armGeom struct {
	sx, sy, ex, ey, wx, wy, hx, hy float64
	layer                          int
	size                           float64 // of the fist
	swing                          float64 // the forearm's angle on the screen from hanging, towards the right
	open                           bool    // the hand fallen open, not a fist
}

// armGeo places arm i (the hand's, or the other) of a figure facing d.
func armGeo(female bool, d sim.Dir, i int, a armJ) armGeom {
	side := d == sim.DirRight
	lat := 1.0 - 2*float64(i) // the hand is on the right of the screen
	var g armGeom
	g.sy = shoulderY
	switch {
	case side:
		g.sx = 27 + 1.2*float64(i)
	case female:
		g.sx = 27 + 9.5*lat
	default:
		g.sx = 27 + 10.9*lat
	}
	depth := func(f float64) float64 {
		if d == sim.DirUp {
			return -armK * f
		}
		return armK * f
	}
	seg := func(th, ab float64) (dx, dy float64) {
		f, up := math.Sin(th)*math.Cos(ab), -math.Cos(th)*math.Cos(ab)
		if side {
			return f, -up
		}
		return lat * math.Sin(ab), -up + depth(f)
	}
	if side {
		g.sx += a.out
	} else {
		g.sy += depth(a.out)
	}
	ux, uy := seg(a.sh, a.ab)
	fx, fy := seg(a.sh+a.el, a.ab-armIn*clamp01(a.ab/armOut)) // the forearm comes back in under the elbow
	g.ex, g.ey = g.sx+ux*upArmLen, g.sy+uy*upArmLen
	g.wx, g.wy = g.ex+fx*foreLen, g.ey+fy*foreLen
	g.hx, g.hy = g.wx+fx*handOff, g.wy+fy*handOff
	g.swing = math.Atan2(fx, fy)
	g.size = 6
	fwd := (math.Sin(a.sh)*upArmLen + math.Sin(a.sh+a.el)*(foreLen+handOff)) * math.Cos(a.ab) / (upArmLen + foreLen + handOff)
	switch {
	case side && i == 1:
		g.layer = armBehind
	case side: // a hand raised sideways is behind the head
		g.layer = armMid
	case g.hy < g.sy-1 || g.ey < g.sy-2:
		g.layer = armOver
	case d == sim.DirDown && fwd > 0.85:
		g.layer, g.size = armOver, 9 // at the viewer: the fist comes nearer and bigger
	default:
		g.layer = armMid
	}
	return g
}

// sleeve is the shape of an arm's sleeve: a man's arm is the thicker.
func (a armGeom) sleeve(female bool) shape {
	r0, r1, r2 := 3.7, 3.2, 2.8
	if female {
		r0, r1, r2 = 3, 2.6, 2.3
	}
	return sCap(a.sx, a.sy, a.ex, a.ey, r0, r1).or(sCap(a.ex, a.ey, a.wx, a.wy, r1, r2))
}

// arm draws arm i: its sleeve on sl, in the shirt's greys, and its fist on
// hd; without a tunic the arm is bare, on hd too.
func (g *figGen) arm(sl, hd *figLayer, d sim.Dir, i int, a armGeom) {
	f := &g.f
	bias, lit := 0.12, 0.1
	switch {
	case d == sim.DirRight && i == 1:
		bias, lit = -0.2, -0.22
	case d == sim.DirRight:
		bias = 0.14
	case i == 0: // on the right of the screen: in the body's shade
		bias, lit = -0.16, -0.12
	}
	if a.size > 6 {
		lit = 0.1
	}
	female := g.look.Female
	if g.wear&sim.WearTunic == 0 {
		g.bareArm(hd, a, bias*0.5, d, i)
	} else {
		s := a.sleeve(female)
		fx, fy := a.wx-a.ex, a.wy-a.ey
		n := math.Hypot(fx, fy)
		f.fill(sl, s, func(x, y int, _ float64) color.RGBA {
			i := int((s.lum(x, y, 3) + bias) * 6)
			if !female && n > 2.5 { // a turned cuff
				switch t := ((float64(x)+0.5-a.wx)*fx + (float64(y)+0.5-a.wy)*fy) / n; {
				case t > -2:
					i--
				case t > -3:
					i++
				}
			}
			return shirtHD.at(max(i, 0))
		})
	}
	if a.open { // the palm and the fingers along the forearm, the thumb aside
		ux, uy, _ := unitOf(a.wx-a.ex, a.wy-a.ey)
		px, py := uy, -ux
		k := 1.0
		if female {
			k = 0.85
		}
		at := func(t, s float64) (float64, float64) { return a.wx + ux*t + px*s, a.wy + uy*t + py*s }
		x0, y0 := at(0.6, 0)
		x1, y1 := at(3.6*k, 0)
		hand := sCap(x0, y0, x1, y1, 1.8*k, 2.1*k)
		for j := -1.5; j <= 1.5; j++ {
			fx0, fy0 := at(3.8*k, j*0.95*k)
			fx1, fy1 := at((6.6-math.Abs(j)*0.5)*k, j*1.15*k)
			hand = hand.or(sCap(fx0, fy0, fx1, fy1, 0.6, 0.5))
		}
		tx0, ty0 := at(1.4*k, -1.6*k)
		tx1, ty1 := at(3.4*k, -3.2*k)
		hand = hand.or(sCap(tx0, ty0, tx1, ty1, 0.75, 0.55))
		f.fill(hd, hand, bevel(g.sk, hand, 1.6, lit))
		return
	}
	// The fist, a woman's the smaller. Hanging at the side and seen from
	// the front or the back it is seen edge on: narrower than it is tall.
	fw, fh := a.size, a.size
	if female {
		fw, fh = fw-1, fh-1
	}
	if d != sim.DirRight && a.size <= 6 && math.Abs(a.swing) < 0.6 {
		fw--
	}
	x0, y0 := math.Round(a.hx-fw/2), math.Round(a.hy-fh/2)
	h := sBox(x0, y0, x0+fw, y0+fh, 2.2*math.Min(fw, fh)/6)
	f.fill(hd, h, bevel(g.sk, h, math.Min(fw, fh)/2, lit))
	x, y := int(x0), int(y0)
	f.dot(hd, x+2, y+3, g.sk.at(2)) // between the thumb and the fingers
	f.dot(hd, x+3, y+3, g.sk.at(2))
	f.dot(hd, x+3, y+4, g.sk.at(2))
}

// clear empties layer l under shape s.
func (f *figHD) clear(l *figLayer, s shape) {
	if len(f.xf) > 0 { // bent: mark what the shape covers, then empty it
		mark := newFigLayer(l.c.w, l.c.h)
		f.add(mark, s, plain(color.RGBA{A: 255}))
		for i, p := range mark.c.px {
			if p.A > 0 {
				l.c.px[i], l.part[i] = color.RGBA{}, 0
			}
		}
		return
	}
	f.add(l, s, func(x, y int, _ float64) color.RGBA {
		x, y = x+int(math.Round(f.ox)), y+int(math.Round(f.oy))
		l.c.px[y*l.c.w+x] = color.RGBA{}
		l.part[y*l.c.w+x] = 0
		return color.RGBA{}
	})
}

// --- frames ------------------------------------------------------------------

// headKey names a drawing of the head: it is the same from frame to frame
// but for what trails from it.
type headKey struct {
	dir        sim.Dir
	shut       bool
	lagX, lagY float64
	bend       bendKey // the trunk's lean and the head's nod: see motion.go
}

// headArt is a head by itself, on the layers it goes on.
type headArt struct {
	B, T *figLayer
	n    uint16
}

type headCache map[headKey]*headArt

func (hc headCache) get(l sim.Look, k headKey) *headArt {
	if h, ok := hc[k]; ok {
		return h
	}
	g := newLimbGen(l, 0)
	g.lagX, g.lagY = k.lagX, k.lagY
	g.f.xf = k.bend.head(k.dir, l.Female)
	g.tallHead(k.dir, k.shut)
	k.bend.mouth(&g.f, g.T, k.dir, l)
	h := &headArt{g.B, g.T, g.f.n}
	if hc != nil {
		hc[k] = h
	}
	return h
}

// layOver lays src over dst, its parts numbered on from base.
func layOver(dst, src *figLayer, base uint16) {
	for i, p := range src.c.px {
		if p.A > 0 {
			dst.c.px[i] = p
			dst.part[i] = src.part[i] + base
		}
	}
}

// The layers of the upper half of a frame, in the order they are drawn.
// The legs go between upBack and upBody, the armour over upShirt and its
// headgear over upTop.
const (
	upFar      = iota // the far arm's sleeve, in the shirt's greys
	upBack            // its hand (the arm, when bare), and hair behind the body
	upBody            // the bare trunk and the fur about the hips
	upShirt           // the clothes, in greys for the player's colour
	upArms            // the sleeves
	upTop             // the head, the hands, the belt and the mantle
	upArmsOver        // a raised arm's sleeve
	upTopOver         // and its fist
	numUpLayers
)

// upperTinted reports whether layer i takes the player's colour.
func upperTinted(i int) bool { return i == upFar || i == upShirt || i == upArms || i == upArmsOver }

// upperCanvas is the upper half of one frame, facing anything but left.
type upperCanvas struct {
	layer [numUpLayers]*canvas
	arm   [2]armGeom
}

// bakeUpper draws the upper half of look l wearing w for key k (never
// DirLeft).
func bakeUpper(l sim.Look, w sim.Wear, k upKey, heads headCache) upperCanvas {
	l = l.Sanitized()
	p := upperPoseOf(k)
	g := newLimbGen(l, w)
	f := &g.f
	bend := bendOf(p)
	f.xf = bend.trunk(k.dir)
	var out upperCanvas
	for i := range out.arm {
		out.arm[i] = armGeo(l.Female, k.dir, i, p.arm[i])
		out.arm[i].open = p.open[i]
		if p.fore[i] {
			out.arm[i].layer = armOver
			if k.dir == sim.DirUp {
				out.arm[i].layer = armBehind
			}
		}
	}
	if k.anim == upFallen {
		out.arm[1].open = true
	}
	each := func(layer int, draw func(i int, a armGeom)) {
		for i := 1; i >= 0; i-- { // the hand's arm over the other
			if a := out.arm[i]; a.layer == layer {
				draw(i, a)
			}
		}
	}
	each(armBehind, func(i int, a armGeom) { g.arm(g.FA, g.B, k.dir, i, a) })
	g.hook = func() {
		each(armMid, func(i int, a armGeom) {
			if w&sim.WearTunic != 0 {
				f.clear(g.T, a.sleeve(l.Female)) // the belt goes under the arm
			}
			g.arm(g.A, g.T, k.dir, i, a)
		})
		f.oy = max(-1, min(1, p.lagY)) // the mantle, a moment behind
	}
	g.tallBody(k.dir, skirtOf(k, skirtT(l.Female, w)))
	f.oy = 0
	h := heads.get(l, headKey{k.dir, p.shut, p.lagX, p.lagY, bend})
	layOver(g.B, h.B, f.n)
	layOver(g.T, h.T, f.n)
	f.n += h.n
	each(armOver, func(i int, a armGeom) { g.arm(g.AO, g.TO, k.dir, i, a) })

	innerLines(0.4, g.FA, g.B, g.U, g.S, g.A, g.T)
	innerLines(0.4, g.AO, g.TO)
	rimLayers(g.FA.c, g.B.c, g.U.c, g.S.c, g.A.c, g.T.c)
	rimLayers(g.AO.c, g.TO.c)
	out.layer = [numUpLayers]*canvas{g.FA.c, g.B.c, g.U.c, g.S.c, g.A.c, g.T.c, g.AO.c, g.TO.c}
	for i := range out.arm { // where the hands went with the trunk
		a := &out.arm[i]
		a.hx, a.hy = f.fwd(a.hx, a.hy)
		if k.dir == sim.DirRight {
			a.swing -= p.tilt
		}
	}
	return out
}

// bakeLower draws the legs of look l wearing w for key k (never DirLeft).
func bakeLower(l sim.Look, w sim.Wear, k loKey) (*canvas, lowerPose) {
	p := lowerPoseOf(k)
	g := newLimbGen(l.Sanitized(), w)
	g.legs(k.dir, p)
	innerLines(0.4, g.L)
	g.L.c.outline(0.72)
	return g.L.c, p
}

// limbSprite makes a sprite of a layer of the jointed figure, cut down to
// what is drawn on it and anchored between the feet. An empty layer has no
// image.
func limbSprite(c *canvas) sprite {
	x0, y0, x1, y1 := c.w, c.h, -1, -1
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A > 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return sprite{}
	}
	x0, y0 = x0/HDDensity*HDDensity, y0/HDDensity*HDDensity // on whole art pixels
	n := newCanvas(x1-x0+1, y1-y0+1)
	for y := 0; y < n.h; y++ {
		copy(n.px[y*n.w:(y+1)*n.w], c.px[(y+y0)*c.w+x0:(y+y0)*c.w+x0+n.w])
	}
	return spriteHD(n.image(), limbAX-float64(x0)/HDDensity, limbFeet-float64(y0)/HDDensity)
}
