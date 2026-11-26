package client

import (
	"math"

	"survivalgame/internal/sim"
)

// Motions of the jointed figure beyond its stride and its blows, posed after
// the Universal LPC sheets' animation set (see HD_ART.md, "Motion
// reference (LPC)"): their key poses (anticipation, contact,
// follow-through, recovery) and their rhythm, on our own figure. Nothing of
// LPC's art is used.
//
// A motion is a run of key poses, each held for its own time. A key pose
// sets both halves of the figure (limbs.go): the arms, the legs, how far the
// hips drop, and what the trunk and the head do, which are drawn bent (a
// lean turns the trunk about the hips, a nod the head about the neck,
// hdfig.go's figMap), not skewed. Motion m is drawn as the upper half
// upMotion+m and the legs loMotion+m, frame by frame, the first time each is
// shown, like every other frame of the figure.

type motionID uint8

const (
	moNone       motionID = iota
	moIdle                // breathing, standing still
	moReady               // a fighter's stance, weapon up, bouncing on the knees
	moChop                // an axe, a pick or a hoe: over the shoulder, down, through
	moSlash               // a sword's forehand: raised behind, down across the front
	moBackslash           // and its backhand: from low in front up and back
	moThrust              // a spear: drawn back, driven home in a lunge, held, back
	moShoot               // a bow: raise, nock, draw, hold, loose, recover
	moThrow               // something tossed out of the bag: the same, short and underhand
	moHurt                // struck: snapped back, hunched over the blow, up again
	moDie                 // struck down: the knees go, a kneel, a slump
	moGather              // crouched, reaching down to the ground, and up with it
	moReachUp             // reaching up into a tree for its fruit
	moEatBite             // meat on the bone: raised, torn at twice, chewed
	moEatHand             // berries: hand to mouth, twice
	moEatBowl             // soup from a bowl: both hands, tilted, sipped
	moDrink               // a flask: tipped up, the head back, swallowed
	moScoop               // down on one knee at the water, dipping what is held
	moDrinkWater          // down on one knee, cupped hands to the mouth, twice
	moSitDown             // down onto the ground by a fire
	moSit                 // sitting, breathing
	moStandUp             // and back up
	moMount               // up onto a beast's back
	moDismount            // and down off it
	moRide                // astride, the mount walking: the rider sways with it
	moRideRun             // the mount running: lean in, bounce
	moSwim                // a breaststroke: reach, sweep out, pull in, shoot forward, glide
	moTread               // treading water: hands sculling, legs turning, bobbing
	moRideIdle            // astride a beast standing still: breathing
	moSleep               // lying asleep: on the back, on one side, the other, curled on the ground; each breathing in and out
	moWake                // sitting up out of sleep, standing, a stretch and a yawn
	numMotions
)

// motionDef is how a motion runs: its key poses' times in milliseconds, and
// whether it starts again when done. lpc names the LPC animation it is timed
// after, if one.
type motionDef struct {
	name string
	ms   []int
	loop bool
	lpc  string
}

// LPC's previewer steps every animation at 8 frames a second (125 ms) and
// holds a frame by repeating it in the cycle (idle is 0, 0, 1). Where our
// figure has fewer or more frames the times are LPC's spread over ours; a
// few are slowed where a 64-pixel figure's twitch would read as a jitter on
// one three times as fine (see HD_ART.md).
var motions = [numMotions]motionDef{
	moIdle:       {"idle", []int{700, 450, 700, 450}, true, "idle"},
	moReady:      {"combat idle", []int{250, 125, 250, 125}, true, "combat_idle"},
	moChop:       {"chop", []int{125, 60, 60, 125, 125, 125}, false, "slash"},
	moSlash:      {"slash", []int{125, 60, 60, 125, 125, 125}, false, "backslash"},
	moBackslash:  {"backslash", []int{125, 60, 60, 125, 125, 125}, false, "backslash"},
	moThrust:     {"thrust", []int{125, 125, 125, 125, 125, 125, 125, 125}, false, "thrust"},
	moShoot:      {"shoot", []int{125, 125, 125, 125, 125, 125, 125, 125, 125, 125, 125, 125, 125}, false, "shoot"},
	moThrow:      {"throw", []int{70, 70, 90, 60, 60, 90, 80}, false, "shoot"},
	moHurt:       {"hurt", []int{33, 33, 50, 50, 33, 33}, false, "hurt"},
	moDie:        {"die", []int{125, 125, 125, 125, 125, 250}, false, "hurt"},
	moGather:     {"gather", []int{60, 80, 110, 80, 90, 80}, false, "sit"},
	moReachUp:    {"reach up", []int{80, 100, 120, 100, 100}, false, "spellcast"},
	moEatBite:    {"eat (bite)", []int{150, 125, 110, 110, 125, 110, 110, 160}, false, "emote"},
	moEatHand:    {"eat (by hand)", []int{125, 110, 125, 125, 125, 110, 125, 155}, false, "emote"},
	moEatBowl:    {"eat (bowl)", []int{150, 125, 175, 175, 175, 200}, false, "spellcast"},
	moDrink:      {"drink", []int{125, 125, 175, 150, 150, 125, 150}, false, "spellcast"},
	moScoop:      {"scoop", []int{125, 125, 175, 200, 150, 175, 125, 125}, false, "sit"},
	moDrinkWater: {"drink water", []int{125, 125, 150, 150, 175, 150, 150, 175, 125, 125}, false, "sit"},
	moSitDown:    {"sit down", []int{125, 125, 125, 125, 250}, false, "sit"},
	moSit:        {"sit", []int{625, 625, 625, 625}, true, "sit"},
	moStandUp:    {"stand up", []int{100, 100, 100, 100, 125}, false, "sit"},
	moMount:      {"mount", []int{100, 100, 120, 120, 160}, false, "jump"},
	moDismount:   {"dismount", []int{100, 120, 120, 100, 160}, false, "jump"},
	moRide:       {"ride", []int{125, 125, 125, 125, 125, 125, 125, 125}, true, "walk"},
	moRideRun:    {"ride (run)", []int{80, 80, 80, 80, 80, 80, 80, 80}, true, "run"},
	moSwim:       {"swim", []int{70, 70, 70, 70, 70, 70, 70, 70}, true, "walk"},
	moTread:      {"tread water", []int{220, 200, 220, 220, 200, 220}, true, "idle"},
	moRideIdle:   {"ride (still)", []int{700, 450, 700, 450}, true, "idle"},
	moSleep:      {"sleep", []int{1500, 1500, 1500, 1500, 1500, 1500, 1500, 1500}, false, "sit"},
	moWake:       {"wake", []int{170, 150, 150, 260, 300, 170}, false, "spellcast"},
}

// frames is how many key poses m has.
func (m motionID) frames() int { return len(motions[m].ms) }

// span is how long m takes once through, in frames at 60 a second.
func (m motionID) span() float64 {
	t := 0
	for _, ms := range motions[m].ms {
		t += ms
	}
	return float64(t) * 60 / 1000
}

// frameAt is the key pose m shows age frames (at 60 a second) after it
// began: held on its last when it does not loop.
func (m motionID) frameAt(age float64) int {
	d := motions[m]
	t := age * 1000 / 60
	if d.loop {
		total := 0
		for _, ms := range d.ms {
			total += ms
		}
		t = math.Mod(t, float64(total))
	}
	for i, ms := range d.ms {
		if t < float64(ms) {
			return i
		}
		t -= float64(ms)
	}
	return len(d.ms) - 1
}

// keys returns the upper and lower halves' keys of frame f of m facing d.
func (m motionID) keys(d sim.Dir, f int) (loKey, upKey) {
	f = max(0, min(m.frames()-1, f))
	return loKey{d, loMotion + loAnim(m), uint8(f)}, upKey{d, upMotion + upAnim(m), uint8(f)}
}

// bodyPose is both halves of a frame.
type bodyPose struct {
	up upperPose
	lo lowerPose
}

// kp is a key pose, as it is written down: arms, legs and the rest. The
// arms and the legs are as limbs.go's armJ and legJ; the arms in the
// trunk's own frame, so they go with it when it leans.
type kp struct {
	a0, a1       armJ    // the hand that holds things, and the other
	l0, l1       legJ    // the leg on the hand's side (sideways: the near one), and the other
	drop         float64 // the hips' drop, image pixels
	tilt, nod    float64 // radians: see upperPose
	roll         float64
	jaw          float64
	spread       float64 // the knees apart, seen from the front or the back
	tuck0, tuck1 bool    // that leg is knelt on: its foot is behind the knee
	open0, open1 bool    // the hand is open
	fore0, fore1 bool    // the hand comes up in front of the face: drawn over the head (out of sight from behind)
	shut         bool
	breath       float64 // image pixels the chest rises
	lagX, lagY   float64
}

func (k kp) pose() bodyPose {
	var b bodyPose
	b.up = upperPose{arm: [2]armJ{k.a0, k.a1}, lagX: k.lagX, lagY: k.lagY, shut: k.shut,
		tilt: k.tilt, nod: k.nod, roll: k.roll, jaw: k.jaw, open: [2]bool{k.open0, k.open1}, breath: k.breath, fore: [2]bool{k.fore0, k.fore1}}
	b.lo = lowerPose{leg: [2]legJ{k.l0, k.l1}, drop: k.drop, spread: k.spread, tuck: [2]bool{k.tuck0, k.tuck1}}
	return b
}

var (
	armRest = armJ{el: armBend, ab: armOut}
	legRest = legJ{}
)

// kneelLeg is a leg knelt on: the knee on the ground under the hip, the
// shin lying back along it, the top of the foot down. It needs the hips
// dropped by kneelDrop.
var kneelLeg = legJ{f: -15.8, u: -0.5, pitch: 2.75}

const kneelDrop = 21.7

// sitDrop is how far the hips drop to sit on the ground.
const sitDrop = 40.0

// motionPose is frame f of motion m facing d (DirLeft is drawn mirrored
// from DirRight).
func motionPose(m motionID, f int, d sim.Dir) bodyPose {
	side := d == sim.DirRight || d == sim.DirLeft
	k := motionKey(m, f, side, d == sim.DirUp)
	return k.pose()
}

// motionKey is the key pose of frame f of m, sideways or from the front
// (back is from behind).
func motionKey(m motionID, f int, side, back bool) kp {
	switch m {
	case moIdle:
		// LPC's idle: the chest rises a pixel and settles. Here the trunk
		// fills with the breath (a pixel and a half of 93) and the arms
		// float out a little with it.
		b := [4]float64{0, 1, 1.5, 0.6}[f]
		a := armJ{el: armBend + 0.04*b, ab: armOut + 0.02*b}
		return kp{a0: a, a1: a, breath: b, lagY: -0.3 * b}

	case moReady:
		// LPC's combat idle: knees bent, weight forward, the weapon up in
		// front, the guard hand close; a dip of the knees on the off beat.
		dip := [4]float64{0, 2.5, 0.8, 2.5}[f]
		k := kp{
			a0: armJ{sh: 0.95, el: 1.05, ab: 0.12, out: 1},
			a1: armJ{sh: 0.55, el: 1.5, ab: 0.2},
			l0: legJ{f: 8}, l1: legJ{f: -9},
			drop: 5 + dip, tilt: 0.1, nod: 0.06, spread: 3.5,
			lagY: 0.5 * dip,
		}
		if !side {
			k.a0, k.a1 = armJ{sh: 0.7, el: 1.3, ab: 0.3}, armJ{sh: 0.5, el: 1.55, ab: 0.35}
			k.l0, k.l1 = legJ{f: 3}, legJ{f: -3}
		}
		return k

	case moChop, moSlash:
		return blowKey(m, f, side, back)
	case moBackslash:
		return backslashKey(f, side)
	case moThrust:
		return thrustKey(f, side)
	case moShoot:
		return shootKey(f, side)
	case moThrow:
		return throwKey(f, side)
	case moHurt:
		return hurtKey(f, side)
	case moDie:
		return dieKey(f, side)
	case moGather:
		return gatherKey(f, side)
	case moReachUp:
		return reachUpKey(f, side)
	case moEatBite, moEatHand, moEatBowl, moDrink:
		return eatKey(m, f, side, back)
	case moScoop, moDrinkWater:
		return scoopKey(m, f, side)
	case moSitDown, moSit, moStandUp:
		return sitKey(m, f, side)
	case moMount, moDismount, moRide, moRideRun:
		return rideKey(m, f, side)
	case moRideIdle:
		k := rideKey(moRide, 0, side)
		k.breath = [4]float64{0, 1, 1.5, 0.6}[f]
		k.drop = 0
		return k
	case moSleep:
		return sleepPoseKey(f)
	case moWake:
		return wakeKey(f, side)
	case moSwim:
		return swimKey(f, side, back)
	case moTread:
		return treadKey(f, side)
	}
	return kp{a0: armRest, a1: armRest}
}

// blowKey is a chop or a forehand slash: LPC's slash, as key poses. The
// tool's own angle comes from the server's timing (swing.go, slash.go); the
// frame is picked from it (atkFrame), so these are what the body does as
// the tool goes from raised (0) to landed (5): wound up with the weight
// back and the knees giving (anticipation), the drive down, the contact
// with the trunk thrown over it, the follow-through, and the recovery is
// the same poses run back.
func blowKey(m motionID, f int, side, back bool) kp {
	u := float64(f) / 5
	var k kp
	if side {
		k.a0 = armJ{sh: lerpf(2.8, 0.75, smooth(u)), el: lerpf(1.05, 0.18, u), out: 4 * u}
		k.a1 = armJ{sh: lerpf(1.7, 0.55, u), el: lerpf(1.1, 0.7, u), ab: 0.1}
		// The near foot steps into the blow; the back one pushes.
		k.l0 = legJ{f: lerpf(2, 10, u)}
		k.l1 = legJ{f: lerpf(-4, -11, u), pitch: lerpf(0, 0.35, u)}
		k.l1.u = footLift(k.l1.pitch)
		k.drop = [6]float64{3, 2, 3, 6, 7, 6}[f]
		k.tilt = [6]float64{-0.12, -0.06, 0.06, 0.2, 0.24, 0.2}[f]
		k.nod = [6]float64{-0.08, 0, 0.08, 0.16, 0.18, 0.14}[f]
		k.lagX = math.Round(-2 * (u - 0.4))
	} else {
		k.a0 = armJ{sh: lerpf(2.6, 0.55, u), el: lerpf(0.9, 0.25, u), ab: lerpf(0.25, 0.7, u)}
		k.a1 = armJ{sh: 0.3 * (1 - u), el: 0.6, ab: 0.15}
		k.l0, k.l1 = legJ{f: lerpf(0, 4, u)}, legJ{f: lerpf(0, -4, u)}
		k.drop = [6]float64{2, 1, 2, 5, 6, 5}[f]
		k.tilt = [6]float64{-0.1, -0.05, 0.05, 0.18, 0.22, 0.18}[f]
		k.spread = 2.5
		k.roll = [6]float64{-0.04, -0.02, 0, 0.03, 0.05, 0.04}[f]
	}
	k.lagY = math.Round(2 * (u - 0.5))
	if m == moSlash { // a sword: a longer reach, the trunk turned further
		k.tilt *= 1.15
		k.a0.out += 1
	}
	return k
}

// backslashKey is a sword's backhand (LPC's 1h backslash, its second
// half): the blade low across the front, swept up and back past the
// shoulder, the trunk turning back with it.
func backslashKey(f int, side bool) kp {
	u := float64(f) / 5
	var k kp
	if side {
		k.a0 = armJ{sh: lerpf(0.55, 2.55, smooth(u)), el: lerpf(0.5, 0.15, u), out: lerpf(4, 1, u)}
		k.a1 = armJ{sh: lerpf(0.9, 0.2, u), el: 1.1}
		k.l0 = legJ{f: lerpf(10, 5, u)}
		k.l1 = legJ{f: lerpf(-11, -7, u)}
		k.drop = [6]float64{6, 6, 5, 3, 3, 3}[f]
		k.tilt = [6]float64{0.2, 0.14, 0.04, -0.06, -0.1, -0.08}[f]
		k.nod = [6]float64{0.12, 0.08, 0, -0.06, -0.08, -0.05}[f]
	} else {
		k.a0 = armJ{sh: lerpf(0.6, 2.4, u), el: lerpf(0.3, 0.6, u), ab: lerpf(0.8, 0.35, u)}
		k.a1 = armJ{sh: 0.2, el: 0.9, ab: 0.2}
		k.drop = [6]float64{5, 5, 4, 3, 2, 2}[f]
		k.tilt = [6]float64{0.15, 0.1, 0.03, -0.05, -0.08, -0.06}[f]
		k.roll = [6]float64{0.04, 0.02, 0, -0.03, -0.05, -0.04}[f]
		k.spread = 2.5
	}
	k.lagX = math.Round(2 * (u - 0.5))
	return k
}

// thrustKey is a spear's blow, LPC's thrust: stand ready (0), draw it back
// with the weight on the back foot (1, 2), drive it home in a lunge (3),
// hold there (4 to 6), and come back (7).
func thrustKey(f int, side bool) kp {
	// How far out the spear hand is: back, out, held, back.
	out := [8]float64{0, -0.6, -1, 1, 1, 0.95, 0.9, 0.3}[f]
	lunge := clamp01(out)
	var k kp
	if side {
		k.a0 = armJ{sh: lerpf(0.75, 1.45, lunge), el: lerpf(1.1, 0.1, lunge), out: 5 * out}
		if out < 0 {
			k.a0 = armJ{sh: lerpf(0.75, 0.1, -out), el: lerpf(1.1, 1.75, -out), out: 3 * out}
		}
		k.a1 = armJ{sh: lerpf(0.9, 0.6, lunge), el: lerpf(0.9, 1.4, lunge)}
		k.l0 = legJ{f: lerpf(4, 15, lunge)}
		k.l1 = legJ{f: lerpf(-5, -15, lunge), pitch: 0.3 * lunge}
		k.l1.u = footLift(k.l1.pitch)
		k.drop = lerpf(2, 8, lunge) + 2*clamp01(-out)
		k.tilt = 0.22*lunge - 0.1*clamp01(-out)
		k.nod = 0.1 * lunge
	} else {
		k.a0 = armJ{sh: lerpf(0.5, 1.45, lunge), el: lerpf(1.3, 0.1, lunge), ab: -0.25 * lunge}
		if out < 0 {
			k.a0 = armJ{sh: lerpf(0.5, 0.05, -out), el: lerpf(1.3, 1.7, -out), ab: 0.3}
		}
		k.a1 = armJ{sh: 0.6, el: 1.2, ab: 0.2}
		k.drop = lerpf(2, 6, lunge)
		k.tilt = 0.18*lunge - 0.08*clamp01(-out)
		k.spread = 3
	}
	k.lagX = math.Round(-2 * out)
	return k
}

// shootKey is LPC's shoot, for a bow: the bow raised (0 to 2), the arrow
// nocked (3), drawn to the cheek (4, 5), held (6 to 8), loosed (9), and the
// arms come down (10 to 12). The bow is in the off hand, held out to the
// target; the hand that holds things draws the string.
func shootKey(f int, side bool) kp {
	raise := [13]float64{0, 0.4, 0.8, 1, 1, 1, 1, 1, 1, 1, 0.8, 0.45, 0.1}[f]
	draw := [13]float64{0, 0, 0, 0.1, 0.6, 1, 1, 1, 1, 0, 0, 0, 0}[f]
	var k kp
	if side {
		// Sideways the bow arm is the far one, straight out ahead; the near
		// hand pulls back past the cheek.
		k.a1 = armJ{sh: lerpf(0.2, 1.62, raise), el: lerpf(0.3, 0.05, raise)}
		k.a0 = armJ{sh: lerpf(0.3, 1.55, raise) - 0.15*draw, el: lerpf(0.5, 1.2, raise) + 1.5*draw, out: -2 * draw}
		k.tilt = -0.04 * raise
		k.nod = 0.06 * raise
	} else {
		k.a1 = armJ{sh: lerpf(0.2, 1.5, raise), el: 0.05, ab: lerpf(armOut, 0.35, raise)}
		k.a0 = armJ{sh: lerpf(0.3, 1.4, raise), el: 0.4 + 1.4*draw, ab: lerpf(armOut, 0.9, draw)}
	}
	if f == 9 { // loosed: the drawing hand flies back open
		k.a0.el, k.a0.sh, k.open0 = 0.6, k.a0.sh-0.3, true
	}
	k.l0, k.l1 = legJ{f: 6}, legJ{f: -7}
	if !side {
		k.l0, k.l1 = legJ{}, legJ{}
		k.spread = 3
	}
	k.drop = 2 + raise
	return k
}

// throwKey is shoot run short and underhand, for something tossed from the
// bag: the hand swings back (0, 1), forward and up (2, 3), lets go (4) and
// comes back (5, 6).
func throwKey(f int, side bool) kp {
	sw := [7]float64{0.1, -0.7, -0.4, 0.9, 1.5, 1.1, 0.4}[f]
	var k kp
	k.a0 = armJ{sh: sw, el: 0.3 + 0.2*math.Abs(sw), ab: armOut}
	k.open0 = f >= 4
	k.a1 = armJ{sh: -0.3 * sw, el: 0.4, ab: armOut}
	k.drop = [7]float64{1, 4, 5, 3, 2, 2, 1}[f]
	k.tilt = [7]float64{0, 0.1, 0.12, 0.06, -0.03, 0, 0}[f]
	if side {
		k.l0, k.l1 = legJ{f: 5}, legJ{f: -6}
	} else {
		k.a0.ab = armOut + 0.15
		k.spread = 2
	}
	return k
}

// hurtKey is a flinch, timed by LPC's hurt: snapped back by the blow with
// the arms flung out and the eyes shut (0, 1), hunched over it (2, 3), and
// straightening (4, 5). The flash and the hop are jolt.go's.
func hurtKey(f int, side bool) kp {
	k := kp{a0: armRest, a1: armRest}
	fl := armJ{sh: -1.2, el: 0.5, ab: 0.4}
	if !side {
		fl = armJ{sh: 0.3, el: 0.5, ab: 1.6}
	}
	switch f {
	case 0, 1:
		k.a0, k.a1 = fl, fl
		k.a0.sh += 0.2 * float64(f)
		k.tilt, k.nod, k.shut = -0.16, -0.3, true
		k.drop = 1
		k.lagX, k.lagY = 2, -1
	case 2, 3:
		k.a0 = armJ{sh: 0.7, el: 1.4, ab: 0.2}
		k.a1 = armJ{sh: 0.8, el: 1.6, ab: 0.25}
		k.tilt, k.nod, k.drop = 0.2, 0.25, 5
		k.shut = f == 2
		k.spread = 2
	case 4:
		k.a0, k.a1 = armJ{sh: 0.3, el: 0.7, ab: 0.2}, armJ{sh: 0.35, el: 0.8, ab: 0.2}
		k.tilt, k.nod, k.drop = 0.08, 0.1, 2
	}
	return k
}

// dieKey is a fall, LPC's hurt row frame by frame: the blow (0), the knees
// going (1), down on them (2), the trunk slumping (3), and over (4) onto the
// ground (5: the body as it lies; see bodyart.go). The last two are drawn
// whole, turned: see drawDying.
func dieKey(f int, side bool) kp {
	k := kp{a0: armRest, a1: armRest}
	limp := armJ{sh: 0.15, el: 0.35, ab: 0.25}
	switch f {
	case 0:
		k = hurtKey(0, side)
	case 1:
		k.a0, k.a1 = armJ{sh: 0.5, el: 1.2, ab: 0.3}, armJ{sh: 0.6, el: 1.3, ab: 0.3}
		k.l0, k.l1 = legJ{f: 4}, legJ{f: -3}
		k.drop, k.tilt, k.nod, k.shut = 12, 0.22, 0.3, true
		k.spread = 3
	case 2, 3, 4, 5:
		k.a0, k.a1 = limp, limp
		k.l0, k.l1 = kneelLeg, kneelLeg
		k.l0.f, k.l1.f = -14.8, -16.8
		k.tuck0, k.tuck1 = true, true
		k.drop = kneelDrop
		k.tilt, k.nod, k.shut = 0.3, 0.35, true
		k.spread = 2
		if f >= 3 {
			k.tilt, k.nod = 0.55, 0.5
			k.a0.sh, k.a1.sh = -0.35, -0.3
		}
	}
	return k
}

// gatherKey is a crouch to pick something off the ground (no LPC row does
// it; it is timed as LPC's sit, quicker): bending (0), down in a squat with
// the hand going out (1), reaching (2, the hand open), closing on it (3),
// rising with it (4), and up (5).
func gatherKey(f int, side bool) kp {
	sq := [6]float64{0.3, 0.85, 1, 1, 0.55, 0.15}[f]
	k := kp{
		l0: legJ{f: 6}, l1: legJ{f: -7},
		drop: 30 * sq, tilt: 0.74 * sq, nod: 0.2 * sq,
		spread: 6 * sq,
		lagY:   [6]float64{1, 2, 1, 0, -1, -1}[f],
	}
	reach := [6]float64{0.2, 0.7, 1, 0.9, 0.4, 0.1}[f]
	k.a0 = armJ{sh: lerpf(0.1, 0.5, reach), el: lerpf(0.3, 0.08, reach), ab: armOut, out: 2 * reach}
	k.a1 = armJ{sh: 0.55 * sq, el: 1.1, ab: 0.25}
	k.open0 = f == 1 || f == 2
	if !side {
		k.a0 = armJ{sh: 0.6 * reach, el: 0.2, ab: 0.35 + 0.2*reach}
		k.a1 = armJ{sh: 0.6 * sq, el: 0.9, ab: 0.3}
		k.l0, k.l1 = legJ{f: 2}, legJ{f: -2}
	}
	return k
}

// reachUpKey is reaching up into a tree (LPC's spellcast's arms, up): on
// the toes, the hand up and open (1, 2), closing (3), pulling it down (4).
func reachUpKey(f int, side bool) kp {
	up := [5]float64{0.3, 0.9, 1, 1, 0.4}[f]
	k := kp{nod: -0.25 * up, tilt: -0.06 * up}
	k.a0 = armJ{sh: lerpf(0.3, 2.85, up), el: lerpf(0.6, 0.15, up), ab: 0.15}
	k.a1 = armJ{sh: lerpf(0.1, 0.5, up), el: 0.6, ab: 0.2}
	k.open0 = f == 1 || f == 2
	toe := 0.55 * up
	k.l0, k.l1 = legJ{pitch: toe, u: footLift(toe)}, legJ{pitch: toe, u: footLift(toe)}
	k.drop = -footLift(toe)
	if !side {
		k.a0 = armJ{sh: lerpf(0.2, 2.7, up), el: 0.2, ab: 0.35}
		k.l0, k.l1 = legJ{u: 2 * up}, legJ{u: 2 * up}
		k.drop = -2 * up
	}
	return k
}

// eatKey is eating or drinking, on LPC's emote and spellcast rhythm (the
// hands come up and are held there).
//
// A bite (meat, fish, bread): the food to the mouth (0), the head dips and
// tears at it as the hand pulls away (1), chew, chew (2, 3), again (4 to 6),
// down (7). By hand (berries): up and in (0, 1), down to take more (2),
// chewing, up and in again (3 to 5), chew and down (6, 7). A bowl: both
// hands bring it up (0, 1), tipped, sipped with the head back (2 to 4), down
// (5). A flask: up (0, 1), tipped up with the head thrown back (2), swallow,
// swallow (3, 4), down (5), a breath (6).
func eatKey(m motionID, f int, side, back bool) kp {
	k := kp{a0: armRest, a1: armRest, fore0: true}
	// The fist just in front of the lips: sideways the upper arm down and
	// forward, the forearm nearly upright; from the front the arm comes in
	// across the chest. (Worked out from the arm's lengths: see armGeo.)
	mouth := armJ{sh: 0.55, el: 2.45, ab: 0.05}
	tear := armJ{sh: 0.5, el: 2.05, ab: 0.05, out: 1} // pulled away and down, tearing
	low := armJ{sh: 0.5, el: 1.3, ab: 0.1}            // held before the chest, chewing
	if !side {
		mouth = armJ{sh: 1.3, el: 1.84, ab: -0.3}
		tear = armJ{sh: 1.15, el: 1.55, ab: -0.22}
		low = armJ{sh: 0.8, el: 1.25, ab: -0.05}
	}
	switch m {
	case moEatBite:
		switch f {
		case 0:
			k.a0 = mouth
			k.a0.el -= 0.25
		case 1, 4:
			k.a0, k.nod, k.jaw, k.tilt = mouth, 0.2, 1, 0.05
		case 2, 5:
			k.a0, k.nod, k.jaw = tear, 0.08, 1
		case 3, 6:
			k.a0, k.nod = low, 0.03
		case 7:
			k.a0 = armJ{sh: 0.3, el: 0.7, ab: armOut}
			k.fore0 = false
		}
		k.a1 = armJ{sh: 0.2, el: 0.6, ab: 0.18}
	case moEatHand:
		switch f {
		case 0, 3:
			k.a0, k.open0 = mouth, true
			k.a0.el -= 0.2
		case 1, 4:
			k.a0, k.nod, k.jaw = mouth, 0.1, 1
		case 2, 5:
			k.a0, k.nod = low, 0.03
			k.a0.el += 0.15
		case 6:
			k.a0, k.jaw = low, 1
		case 7:
			k.a0 = armJ{sh: 0.3, el: 0.6, ab: armOut}
			k.fore0 = false
		}
		// The other hand holds the rest of the berries before the chest.
		k.a1 = armJ{sh: 0.6, el: 1.35, ab: 0.05}
		if !side {
			k.a1 = armJ{sh: 0.8, el: 1.3, ab: -0.08}
		}
	case moEatBowl:
		bowl := [6]float64{0.4, 0.85, 1, 1, 1, 0.4}[f]
		tip := [6]float64{0, 0, 0.6, 1, 0.8, 0}[f]
		a := armJ{sh: lerpf(0.5, 0.65, bowl), el: lerpf(1.1, 2.15, bowl) + 0.15*tip, ab: 0.05}
		if !side {
			a = armJ{sh: lerpf(0.7, 1.25, bowl), el: lerpf(1.1, 1.7, bowl) + 0.1*tip, ab: lerpf(0, -0.26, bowl)}
		}
		k.a0, k.a1 = a, a
		k.fore1 = true
		k.nod = -0.2 * tip
		k.tilt = -0.04 * tip
	case moDrink:
		up := [7]float64{0.4, 0.85, 1, 1, 1, 0.45, 0}[f]
		tip := [7]float64{0, 0.2, 1, 1, 0.9, 0, 0}[f]
		k.a0 = armJ{sh: lerpf(0.4, 0.55+0.25*tip, up), el: lerpf(1.0, 2.35+0.2*tip, up), ab: 0.05}
		if !side {
			k.a0 = armJ{sh: lerpf(0.6, 1.3+0.15*tip, up), el: lerpf(1.0, 1.8+0.1*tip, up), ab: lerpf(0.05, -0.3, up)}
		}
		k.a1 = armJ{sh: 0.25, el: 0.5, ab: 0.2}
		k.nod = -0.42 * tip
		k.tilt = -0.08 * tip
		k.fore0 = up > 0.5
		if f == 6 {
			k.breath = 1.2
		}
	}
	if back { // seen from behind, the hands go to the mouth out of sight
		k.a0.ab = math.Max(k.a0.ab, 0.1)
		k.a1.ab = math.Max(k.a1.ab, 0.1)
	}
	return k
}

// scoopKey is going down on one knee at the water (LPC's sit, its kneel):
// for a bowl or a flask, dipped (2), filled (3), lifted out dripping (4),
// looked at (5), and up (6, 7). By hand: cupped, dipped and raised to the
// mouth twice (2 to 7), and up (8, 9).
func scoopKey(m motionID, f int, side bool) kp {
	k := kp{
		l0: kneelLeg, l1: legJ{f: 15},
		drop: kneelDrop, tuck0: true, spread: 3,
	}
	down := func(frac float64) {
		k.drop = kneelDrop * frac
		k.l0 = legJ{f: lerpf(0, kneelLeg.f, frac), u: lerpf(0, kneelLeg.u, frac), pitch: lerpf(0, kneelLeg.pitch, frac)}
		k.l1 = legJ{f: 15 * frac}
		k.tuck0 = frac > 0.6
	}
	dip := armJ{sh: 0.35, el: 0.15, ab: 0.1, out: 3}
	lift := armJ{sh: 1.0, el: 1.25, ab: 0.06}
	cup := armJ{sh: 0.95, el: 2.0, ab: 0.02, out: 1}
	if !side {
		dip, lift, cup = armJ{sh: 0.6, el: 0.15, ab: 0.15}, armJ{sh: 0.9, el: 1.3, ab: 0.0}, armJ{sh: 1.1, el: 1.95, ab: -0.14}
	}
	switch m {
	case moScoop:
		switch f {
		case 0:
			down(0.45)
			k.a0, k.a1, k.tilt = armJ{sh: 0.4, el: 0.8, ab: 0.15}, armRest, 0.15
		case 1:
			k.a0, k.a1, k.tilt, k.nod = armJ{sh: 0.6, el: 0.5, ab: 0.12}, armJ{sh: 0.3, el: 0.5, ab: 0.2}, 0.35, 0.2
		case 2, 3:
			k.a0, k.a1, k.tilt, k.nod = dip, armJ{sh: 0.45, el: 0.4, ab: 0.2}, 0.6, 0.3
			k.a0.sh -= 0.08 * float64(f-2)
		case 4:
			k.a0, k.a1, k.tilt, k.nod = lift, armJ{sh: 0.3, el: 0.5, ab: 0.2}, 0.3, 0.2
		case 5:
			k.a0, k.a1, k.tilt, k.nod = lift, armRest, 0.15, 0.15
			k.a0.sh += 0.1
		case 6:
			down(0.45)
			k.a0, k.a1, k.tilt = armJ{sh: 0.6, el: 1.0, ab: 0.12}, armRest, 0.1
		case 7:
			down(0.1)
			k.a0, k.a1 = armJ{sh: 0.35, el: 0.7, ab: armOut}, armRest
		}
	case moDrinkWater:
		switch f {
		case 0:
			down(0.45)
			k.a0, k.a1, k.tilt = armJ{sh: 0.3, el: 0.6, ab: 0.15}, armJ{sh: 0.3, el: 0.6, ab: 0.15}, 0.15
		case 1:
			k.a0, k.a1, k.tilt, k.nod = armJ{sh: 0.6, el: 0.5, ab: 0.1}, armJ{sh: 0.6, el: 0.5, ab: 0.1}, 0.4, 0.2
		case 2, 5:
			k.a0, k.a1, k.tilt, k.nod = dip, dip, 0.62, 0.3
			k.open0, k.open1 = true, true
		case 3, 6:
			k.a0, k.a1, k.tilt, k.nod = cup, cup, 0.22, 0.12
			k.open0, k.open1 = true, true
		case 4, 7:
			k.a0, k.a1, k.tilt, k.nod = cup, cup, 0.12, -0.12
			k.a0.sh += 0.12
			k.a1.sh += 0.12
			k.open0, k.open1 = true, true
		case 8:
			down(0.45)
			k.a0, k.a1, k.tilt = armJ{sh: 0.3, el: 0.6, ab: 0.15}, armRest, 0.1
		case 9:
			down(0.1)
			k.a0, k.a1 = armRest, armRest
		}
	}
	if !side {
		k.tilt *= 0.8
	}
	return k
}

// sitKey is sitting down on the ground (LPC's sit, its first pose): the
// knees go (0), a squat with a hand down behind (1, 2), the seat down (3),
// settled with the forearms on the knees (4). Sitting breathes; standing up
// is sitting down run back, quicker.
func sitKey(m motionID, f int, side bool) kp {
	sitting := kp{
		a0: armJ{sh: 1.0, el: 0.55, ab: 0.12}, a1: armJ{sh: 0.95, el: 0.6, ab: 0.14},
		l0: legJ{f: 25, pitch: -0.25}, l1: legJ{f: 29, pitch: -0.3},
		drop: sitDrop, spread: 7, tilt: 0.1, nod: 0.05,
	}
	sitting.l0.u, sitting.l1.u = footLift(-0.25), footLift(-0.3)
	if !side {
		sitting.a0, sitting.a1 = armJ{sh: 0.95, el: 0.6, ab: 0.35}, armJ{sh: 0.95, el: 0.6, ab: 0.35}
		sitting.l0, sitting.l1 = legJ{f: 18}, legJ{f: 18}
		sitting.spread = 8
	}
	switch m {
	case moSit:
		b := [4]float64{0, 1, 1.4, 0.5}[f]
		sitting.breath = b
		sitting.a0.el += 0.03 * b
		return sitting
	case moStandUp:
		f = 4 - f
	}
	squat := kp{
		a0: armJ{sh: 0.6, el: 0.8, ab: 0.2}, a1: armJ{sh: -0.5, el: 0.2, ab: 0.25},
		l0: legJ{f: 6}, l1: legJ{f: 9},
		drop: 26, tilt: 0.35, nod: 0.1, spread: 6,
	}
	if !side {
		squat.a1 = armJ{sh: -0.3, el: 0.2, ab: 0.6}
		squat.l0, squat.l1 = legJ{f: 3}, legJ{f: 3}
	}
	switch f {
	case 0:
		k := squat
		k.drop, k.tilt, k.spread = 9, 0.15, 2
		k.a1 = armJ{sh: -0.2, el: 0.3, ab: 0.2}
		k.l0, k.l1 = legJ{f: 2}, legJ{f: 4}
		return k
	case 1:
		return squat
	case 2:
		k := squat
		k.drop, k.tilt = 34, 0.2
		k.l0, k.l1 = legJ{f: 14}, legJ{f: 17}
		if !side {
			k.l0, k.l1 = legJ{f: 10}, legJ{f: 10}
		}
		k.a1 = armJ{sh: -0.7, el: 0.1, ab: 0.3}
		return k
	case 3:
		k := sitting
		k.a1 = armJ{sh: -0.6, el: 0.2, ab: 0.3}
		k.a0 = armJ{sh: 0.7, el: 0.5, ab: 0.15}
		k.tilt = 0.02
		return k
	}
	return sitting
}

// rideKey is a rider astride a beast. Mounting (LPC's jump for its rhythm):
// a hand on its back (0), a spring (1), the leg swung over (2), settled
// (3, 4). Dismounting runs the other way. Riding: the rider sways with the
// mount's stride, a frame for each eighth of it; running, leans in and
// bounces.
func rideKey(m motionID, f int, side bool) kp {
	seat := kp{
		a0: armJ{sh: 0.85, el: 1.05, ab: 0.1}, a1: armJ{sh: 0.8, el: 1.1, ab: 0.12},
		l0: legJ{f: 13, u: 9, pitch: 0.35}, l1: legJ{f: 15, u: 10, pitch: 0.35},
		spread: 7,
	}
	if !side {
		seat.a0, seat.a1 = armJ{sh: 0.85, el: 1.1, ab: 0.2}, armJ{sh: 0.85, el: 1.1, ab: 0.2}
	}
	switch m {
	case moRide, moRideRun:
		ph := (float64(f) + 0.5) / 8
		run := m == moRideRun
		amp := 1.0
		if run {
			amp = 2.4
		}
		k := seat
		k.drop = -amp * math.Abs(math.Sin(2*math.Pi*ph)) // up off the seat each stride
		k.tilt = 0.05 + 0.03*math.Sin(2*math.Pi*ph)
		if run {
			k.tilt = 0.26 + 0.05*math.Sin(2*math.Pi*ph)
			k.nod = -0.1
			k.a0.sh, k.a1.sh = 1.15, 1.1
			k.lagX, k.lagY = -2, math.Round(1.5*math.Sin(2*math.Pi*ph))
		}
		k.l0.f += 1.5 * math.Sin(2*math.Pi*ph)
		k.l1.f -= 1.5 * math.Sin(2*math.Pi*ph)
		return k
	case moDismount:
		f = 4 - f
	}
	switch f {
	case 0: // standing beside it, a hand on its back
		return kp{a0: armJ{sh: 1.3, el: 0.9, ab: 0.15}, a1: armJ{sh: 1.1, el: 1.0, ab: 0.15}, drop: 3, tilt: 0.08}
	case 1: // a spring off the near foot
		return kp{a0: armJ{sh: 1.0, el: 1.4, ab: 0.15}, a1: armJ{sh: 0.9, el: 1.5, ab: 0.15},
			l0: legJ{f: 2}, l1: legJ{f: 10, u: 12, pitch: 0.3}, drop: 6, tilt: 0.2}
	case 2: // the leg swung over
		return kp{a0: armJ{sh: 0.9, el: 1.3, ab: 0.15}, a1: armJ{sh: 0.85, el: 1.3, ab: 0.15},
			l0: legJ{f: 6, u: 6, pitch: 0.4}, l1: legJ{f: 20, u: 14, pitch: 0.4}, drop: 0, tilt: 0.25, spread: 9}
	case 3:
		k := seat
		k.tilt, k.drop = 0.12, 1
		return k
	}
	return seat
}

// --- bending the figure -------------------------------------------------------

// bendKey is what a frame does to the trunk and the head: part of the
// head's key, as the head is drawn by itself and laid over the body.
type bendKey struct {
	tilt, nod, roll, jaw, breath float64
}

// neckX, neckY is where the head turns on the neck, in image pixels.
const (
	neckX = 27.0
	neckY = 44.5
)

func bendOf(p upperPose) bendKey {
	return bendKey{p.tilt, p.nod, p.roll, p.jaw, p.breath}
}

// trunk is how the trunk is bent facing d (not DirLeft): leaning, it turns
// about the hips sideways, and seen from the front or the back it is seen
// shorter; it fills with a breath, and sways.
func (b bendKey) trunk(d sim.Dir) []figMap {
	var xf []figMap
	if b.breath != 0 {
		xf = append(xf, xfScale(1+0.01*b.breath, 1+b.breath/(hipY-27), 27, hipY))
	}
	switch {
	case b.tilt == 0 && b.roll == 0:
	case d == sim.DirRight:
		xf = append(xf, xfRot(b.tilt, 27, hipY))
	default:
		if b.tilt != 0 {
			xf = append(xf, xfScale(1+0.04*math.Abs(math.Sin(b.tilt)), math.Cos(b.tilt), 27, hipY))
		}
		if b.roll != 0 {
			xf = append(xf, xfRot(b.roll, 27, hipY))
		}
	}
	return xf
}

// head is how the head is bent: its nod about the neck (sideways a turn;
// from the front or the back the face is seen shorter and lower, or
// higher), its jaw, and then the trunk's bend.
func (b bendKey) head(d sim.Dir, female bool) []figMap {
	var xf []figMap
	if b.jaw != 0 {
		switch d {
		case sim.DirRight:
			xf = append(xf, jawMap{29.5, 40, 41.5, b.jaw})
		case sim.DirDown:
			xf = append(xf, jawMap{22.5, 31.5, 41.5, b.jaw})
		}
	}
	switch {
	case b.nod == 0:
	case d == sim.DirRight:
		xf = append(xf, xfRot(b.nod, neckX-0.5, neckY))
	default:
		s := math.Sin(b.nod)
		k := 1 - 0.22*math.Abs(s)
		dy := 3.5 * s
		if d == sim.DirUp {
			dy = 2.5 * s
		}
		xf = append(xf, xfScale(1, k, neckX, neckY), xfMove(0, dy))
	}
	return append(xf, b.trunk(d)...)
}

// mouth draws the open mouth in the gap a dropped jaw leaves, on the head's
// top layer T.
func (b bendKey) mouth(f *figHD, T *figLayer, d sim.Dir, l sim.Look) {
	if b.jaw == 0 || d == sim.DirUp {
		return
	}
	keep := f.xf
	f.xf = f.xf[1:] // the jaw's own map left out: the gap is where the mouth was
	defer func() { f.xf = keep }()
	dark := mix(mouthDark, colInk, 0.3)
	f.n++
	if d == sim.DirRight {
		f.dot(T, 33, 42, dark)
		f.dot(T, 34, 42, mouthDark)
		return
	}
	for x := 25; x <= 28; x++ {
		c := dark
		if x == 25 || x == 28 {
			c = mouthDark
		}
		f.dot(T, x, 42, c)
	}
}

// swimKey is a breaststroke (no LPC row swims; its rhythm is the walk's
// eight beats): arms out ahead in the glide (0, 1), swept out and down (2),
// pulled in under the chin as the head comes up for air (3), the knees
// drawn up (4), the arms shot forward as the legs kick (5), and into the
// glide (6, 7). Sideways the body lies forward into the water, the head up.
// Only the head and shoulders show: the rest is drawn under the surface
// (swimhd.go).
func swimKey(f int, side, back bool) kp {
	var k kp
	type st struct{ sh, el, ab, lift, kf, ku float64 }
	s := [8]st{
		{0.95, 0.05, 0.1, 0, -9, 4},
		{0.9, 0.1, 0.15, 0, -9, 4},
		{0.55, 0.25, 0.55, -1, -6, 6},
		{0.15, 1.9, 0.35, -2.5, 2, 10},
		{0.35, 2.2, 0.15, -2, 5, 12},
		{0.75, 1.0, 0.1, -0.5, -12, 3},
		{0.95, 0.15, 0.08, 0.5, -11, 3},
		{0.95, 0.05, 0.1, 0.5, -10, 4},
	}[f]
	if side {
		k.a0 = armJ{sh: s.sh, el: s.el, ab: 0.05, out: 1}
		k.a1 = armJ{sh: s.sh - 0.08, el: s.el, ab: 0.05}
		k.tilt, k.nod = 0.62, -0.55-0.12*clamp01(-s.lift/2)
		k.l0 = legJ{f: s.kf, u: s.ku, pitch: 0.6}
		k.l1 = legJ{f: s.kf - 2, u: s.ku + 1, pitch: 0.6}
		k.lagX = -2
	} else {
		// From the front or the back the arms sweep out to the sides
		// and in.
		ab := [8]float64{0.25, 0.4, 1.15, 0.7, 0.2, 0.15, 0.2, 0.25}[f]
		k.a0 = armJ{sh: math.Max(0.6, s.sh+0.4), el: s.el * 0.8, ab: ab}
		k.a1 = k.a0
		k.tilt = 0.35
		k.nod = -0.2
		if back {
			k.nod = 0.1
		}
		k.l0, k.l1 = legJ{f: s.kf * 0.3, u: s.ku}, legJ{f: s.kf * 0.3, u: s.ku}
		k.spread = [8]float64{2, 2, 3, 6, 8, 4, 2, 2}[f]
	}
	k.drop = s.lift
	return k
}

// treadKey is treading water, upright: the hands scull out and in just
// under the surface, the legs turn under, the head bobs.
func treadKey(f int, side bool) kp {
	u := float64(f) / 6 * 2 * math.Pi
	sc := math.Sin(u)
	k := kp{
		a0:   armJ{sh: 0.5 + 0.2*sc, el: 0.7, ab: 0.75 + 0.3*sc},
		a1:   armJ{sh: 0.5 - 0.2*sc, el: 0.7, ab: 0.75 - 0.3*sc},
		l0:   legJ{f: 5 * math.Cos(u), u: 6 + 4*math.Sin(u)},
		l1:   legJ{f: -5 * math.Cos(u), u: 6 - 4*math.Sin(u)},
		drop: -1.2 * math.Cos(u),
		nod:  -0.08,
		lagY: math.Round(math.Cos(u)),
	}
	if side {
		k.a0 = armJ{sh: 0.6 + 0.35*sc, el: 0.6, ab: 0.1}
		k.a1 = armJ{sh: 0.6 - 0.35*sc, el: 0.6, ab: 0.1}
		k.tilt = 0.08
	} else {
		k.spread = 4
	}
	return k
}

// sleepPoseKey is a sleeper's pose, drawn standing and laid down
// (sleephd.go): frame 2*pose + breath. On the back (0) the knees are up and
// a hand on the chest; on the side (1, seen from the side and laid down
// either way) curled, the knees drawn up, the upper arm before the chest;
// curled on the ground (2) the lower arm folded under the head for a
// pillow; and (3) on the back, stirring, an arm thrown out.
func sleepPoseKey(f int) kp {
	b := float64(f % 2)
	k := kp{shut: true, breath: 1.4 * b}
	switch f / 2 {
	case 0, 3:
		k.a0 = armJ{sh: 0.55, el: 1.7, ab: -0.05}
		k.a1 = armJ{sh: 0.1, el: 0.3, ab: 0.22}
		k.l0, k.l1 = legJ{f: 14}, legJ{f: 12}
		k.drop, k.spread = 21, 3
		if f/2 == 3 {
			k.a1 = armJ{sh: 0.2, el: 0.4, ab: 1.5}
			k.l1 = legJ{f: 4}
			k.drop = 14
		}
	case 1:
		k.a0 = armJ{sh: 1.25, el: 1.35}
		k.a1 = armJ{sh: 1.5, el: 1.2}
		k.l0, k.l1 = legJ{f: 24, u: 13, pitch: 0.4}, legJ{f: 21, u: 10, pitch: 0.4}
		k.drop, k.tilt, k.nod = 16, 0.35, 0.25
	case 2:
		k.a0 = armJ{sh: 1.15, el: 1.3}
		k.a1 = armJ{sh: 2.85, el: 2.1} // folded under the head
		k.l0, k.l1 = legJ{f: 25, u: 14, pitch: 0.4}, legJ{f: 20, u: 9, pitch: 0.4}
		k.drop, k.tilt, k.nod = 16, 0.4, 0.3
	}
	return k
}

// wakeKey is getting up out of sleep: sitting up (0), a squat (1), up with
// the arms rising (2), the stretch, arms high, the head back in a yawn (3,
// 4), and down (5).
func wakeKey(f int, side bool) kp {
	switch f {
	case 0:
		return sitKey(moSit, 0, side)
	case 1:
		return sitKey(moSitDown, 1, side)
	}
	up := [6]float64{0, 0, 0.5, 1, 1, 0.3}[f]
	k := kp{
		a0:   armJ{sh: lerpf(0.3, 2.9, up), el: lerpf(0.6, 0.35, up), ab: lerpf(armOut, 0.4, up)},
		a1:   armJ{sh: lerpf(0.3, 2.9, up), el: lerpf(0.6, 0.35, up), ab: lerpf(armOut, 0.4, up)},
		tilt: -0.1 * up, nod: -0.25 * up, breath: 1.6 * up,
		drop: -footLift(0.3 * up),
		l0:   legJ{pitch: 0.3 * up, u: footLift(0.3 * up)}, l1: legJ{pitch: 0.3 * up, u: footLift(0.3 * up)},
		shut: f == 3 || f == 4,
	}
	if f == 3 || f == 4 {
		k.jaw = 1 // a yawn
	}
	if !side {
		k.a0.ab, k.a1.ab = lerpf(armOut, 0.55, up), lerpf(armOut, 0.55, up)
	}
	return k
}
