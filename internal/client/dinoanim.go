package client

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The dinosaurs in motion. A species' standing poses (eating, calling, the
// blow itself) are drawn when the game starts (creatures.go, hellwire.go).
// The frames between are poses of the same rig (dinorig.go) worked out here
// and drawn the first time each is shown:
//
//   - a stride of gaitFrames frames, each leg stepping in its own time: the
//     foot planted and carried back under the body, the heel rising as it
//     pushes off, the knee and ankle folding to bring it forward; the body
//     sinking onto each footfall and rising over the leg, the head nodding
//     against it and the tail swaying behind;
//   - a wingbeat of as many frames;
//   - a blow that is wound up first: the beast draws back (rears its head,
//     drops its horns, coils its tail, crouches) and only then lunges into
//     the pose of the blow, through a frame or two between.

// gaitFrames is how many frames a stride or a wingbeat has.
const gaitFrames = 8

// rigKey names one drawn frame: a pose of a species, facing right or left.
type rigKey struct {
	species string
	side    int
	pose    rigPose
}

// rigFrame is a frame drawn, and where its eye is from its anchor (the night
// hunter's, whose eyes shine in the dark; see troodon.go).
type rigFrame struct {
	s      sprite
	eye    [2]float64
	hasEye bool
}

// rigFrames holds the frames drawn so far. They are drawn as they are first
// shown, one at a time, so a species nobody has met costs nothing.
var rigFrames = map[rigKey]rigFrame{}

// rigSprite is pose p of species (a name of dinoRigs) facing side. It must
// be called inside the game loop, where images can be created.
func rigSprite(species string, side int, p rigPose) sprite { return rigFrameOf(species, side, p).s }

func rigFrameOf(species string, side int, p rigPose) rigFrame {
	k := rigKey{species, side, p}
	if f, ok := rigFrames[k]; ok {
		return f
	}
	var right, left rigFrame
	if c, ax, ay, ok := rigCanvas(dinoRigs[species], p); ok {
		if species == "troodon" {
			if ex, ey, ok := eyeOf(c); ok {
				redrawEye(c, int(math.Floor(ex)), int(math.Floor(ey)))
				right.eye, left.eye = eyeOffsets(c, ex, ey, ax, ay)
				right.hasEye, left.hasEye = true, true
			}
		}
		right.s, left.s = sidedPair(mk(c, ax, ay))
	}
	rigFrames[rigKey{species, 0, p}], rigFrames[rigKey{species, 1, p}] = right, left
	if side == 1 {
		return left
	}
	return right
}

// stridePose is frame f of a stride amp wide (1: a walk), on top of pose
// base: what the legs do is the rig's (stepJoints); the head nods against the
// body's bob, twice a stride, and the tail sways once, a little behind.
func stridePose(base rigPose, f int, amp float64) rigPose {
	p := base
	p.cycle = float64(f%gaitFrames+1) / gaitFrames
	p.step = amp
	th := 2 * math.Pi * p.cycle
	p.head -= 0.03 * amp * math.Cos(2*th)
	p.tail += 0.045 * amp * math.Sin(th-0.8)
	return p
}

// stride is species mid-stride at frame count t. period is how many frames
// a step takes (half a stride), as it was when a walk had two poses.
func stride(species string, side int, base rigPose, amp float64, t, period int) sprite {
	return rigSprite(species, side, stridePose(base, t*gaitFrames/(2*period), amp))
}

// flapPose is frame f of a flier's wingbeat, seen from above: the wings
// shorten as they rise and fall, and sweep back on the way down.
func flapPose(f int) rigPose {
	s := math.Sin(2 * math.Pi * float64(f%gaitFrames) / gaitFrames)
	if s >= 0 {
		return rigPose{squash: 1 - 0.3*s, wings: 0.1 * s}
	}
	return rigPose{squash: 1 + 0.14*s, wings: 0.2 * s}
}

// windUp is how a species draws back before its blow, and for how long.
type windUp struct {
	pose   rigPose
	frames int // held drawn back this long, at 60 frames a second
}

// lungeFrames is how many frames lie between drawn back and the blow.
const lungeFrames = 3

// dinoWinds is the wind-up of each family of poses (see posesOf).
var dinoWinds = map[string]windUp{
	"rex":     {rigPose{head: -0.3, open: 0.14, tilt: -0.06, squash: 0.96, tail: -0.08}, 7}, // rears its head back
	"trike":   {rigPose{head: 0.5, squash: 0.94}, 6},                                        // drops its horns
	"stego":   {rigPose{tail: -0.32, squash: 0.96, head: 0.15}, 6},                          // coils its tail low
	"raptor":  {rigPose{squash: 0.78, head: 0.3, tail: 0.14, step: 0.5}, 4},                 // crouches to spring
	"anzu":    {rigPose{tilt: -0.14, head: -0.22, squash: 0.95}, 5},                         // leans back to kick
	"quetzal": {rigPose{neck: -0.2, head: -0.28}, 6},                                        // draws its beak back
	"plesio":  {rigPose{head: -0.32, open: 0.1}, 6},                                         // rears its neck
}

// lerpPose is the pose t of the way from a to b.
func lerpPose(a, b rigPose, t float64) rigPose {
	sq := func(v float64) float64 {
		if v == 0 {
			return 1 // not crouched
		}
		return v
	}
	l := func(x, y float64) float64 { return math.Round((x+(y-x)*t)*1000) / 1000 }
	return rigPose{
		step: l(a.step, b.step), head: l(a.head, b.head), neck: l(a.neck, b.neck),
		open: l(a.open, b.open), tail: l(a.tail, b.tail), wings: l(a.wings, b.wings),
		squash: l(sq(a.squash), sq(b.squash)), tilt: l(a.tilt, b.tilt), lift: l(a.lift, b.lift),
	}
}

// strikeClock is when a creature's blow began, and when it was last seen
// striking.
type strikeClock struct{ at, seen int }

var strikeClocks = map[sim.CreatureID]strikeClock{}

// strikeAge is how many frames ago c began the blow it is striking, or -1 if
// it is not striking.
func (g *Game) strikeAge(c protocol.CreatureView) int {
	if c.State != sim.CreatureStrike {
		delete(strikeClocks, c.ID)
		return -1
	}
	if len(strikeClocks) > 512 {
		clear(strikeClocks) // long gone
	}
	k, ok := strikeClocks[c.ID]
	if !ok || g.frame < k.at || g.frame-k.seen > 6 {
		k.at = g.frame
	}
	k.seen = g.frame
	strikeClocks[c.ID] = k
	return g.frame - k.at
}

// blow is species striking, age frames into the blow (see strikeAge): drawn
// back first, then lunging into pose hit, whose finished sprite is held. far
// is how much of the blow has been thrown, 0 to 1, for a leap's height.
func blow(species string, side, hit, age int, held sprite) (s sprite, far float64) {
	fam := poseFamily(species)
	w, ok := dinoWinds[fam]
	if !ok || age < 0 || age >= w.frames+lungeFrames {
		return held, 1
	}
	if age < w.frames { // drawing back, over the first three frames, and held there
		return rigSprite(species, side, lerpPose(rigPose{}, w.pose, min(1, float64(age+1)/3))), 0
	}
	far = float64(age-w.frames+1) / (lungeFrames + 1)
	return rigSprite(species, side, lerpPose(w.pose, dinoPoses[fam][hit], far)), far
}
