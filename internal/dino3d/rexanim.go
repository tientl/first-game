package dino3d

import (
	"math"

	"survivalgame/internal/r3d"
)

// Anim is one of the tyrant's animations.
type Anim int

const (
	RexWalk Anim = iota // a full stride, looping
	RexIdle             // standing, breathing
	RexRoar             // head thrown up, jaws wide
	RexBite             // a lunge, jaws open
	RexEat              // head down at a kill: jaws nearly shut, then wide
	RexAnims
)

// RexFrames is how many frames each animation has.
var RexFrames = [RexAnims]int{RexWalk: 8, RexIdle: 2, RexRoar: 1, RexBite: 1, RexEat: 2}

// legPose is where a foot is: its ball x ahead of its resting place and z
// above the ground, how far the ankle leans back over it (radians from
// upright), and how far the toes droop.
type legPose struct{ x, z, lean, toe float64 }

// rexPose is a whole pose, in the terms an animator would use; apply turns
// it into bone angles (the legs by inverse kinematics, so feet stay planted
// whatever the body does).
type rexPose struct {
	dx, dz           float64 // the hips, moved from rest
	pitch, yaw, roll float64 // the hips, turned: + pitch dips the head
	chest, chestYaw  float64
	neck, neckYaw    float64
	head, headYaw    float64
	jaw              float64    // + opens
	tailP, tailY     [4]float64 // + pitch lifts the tip
	leg              [2]legPose
	arm              float64
}

// stand is the tyrant at rest: body tilted up towards the head, tail held
// out behind, both feet under it.
func stand() rexPose {
	p := rexPose{dz: -0.12, pitch: -0.1, chest: 0.02, neck: 0.02, head: 0.08}
	p.tailP = [4]float64{0.1, 0.05, 0.03, 0.02}
	p.leg[0] = legPose{x: 0.25, lean: 0.42}
	p.leg[1] = legPose{x: -0.3, lean: 0.5}
	return p
}

// stride is a leg at phase ph of the walk: on the ground and sliding back
// for the first part, lifted and swung forward for the rest.
func stride(ph float64) legPose {
	const (
		length = 2.0
		stance = 0.56
	)
	ph -= math.Floor(ph)
	if ph < stance {
		u := ph / stance
		return legPose{x: length/2 - length*u, lean: 0.22 + 0.6*u*u}
	}
	u := (ph - stance) / (1 - stance)
	s := u * u * (3 - 2*u)
	return legPose{
		x:    -length/2 + length*s,
		z:    0.6 * math.Sin(math.Pi*u),
		lean: 0.82 + 0.1*math.Sin(math.Pi*u) - 0.7*u*u,
		toe:  0.45 * math.Sin(math.Pi*math.Sqrt(u)),
	}
}

// Pose is frame i of animation a.
func rexFrame(a Anim, i int) rexPose {
	p := stand()
	switch a {
	case RexWalk:
		ph := float64(i) / float64(RexFrames[RexWalk])
		w := 2 * math.Pi * ph
		p.leg[0], p.leg[1] = stride(ph), stride(ph+0.5)
		p.dz = -0.12 - 0.11*math.Cos(2*w) // lowest as a foot lands
		p.dx = 0.05 * math.Sin(2*w)
		p.pitch += 0.025 * math.Sin(2*w+0.6)
		p.roll = 0.05 * math.Sin(w)
		p.yaw = 0.06 * math.Sin(w)
		p.chestYaw = -0.05 * math.Sin(w)
		p.neckYaw = -0.03 * math.Sin(w)
		p.headYaw = 0.02 * math.Sin(w)
		p.neck += 0.04 * math.Cos(2*w+0.4)
		p.head -= 0.05 * math.Cos(2*w)
		for k := range p.tailY {
			f := float64(k + 1)
			p.tailY[k] = 0.075 * f * math.Sin(w-0.75*f)
			p.tailP[k] += 0.035 * math.Cos(2*w-0.6*f)
		}
		p.arm = 0.15 * math.Sin(2*w)
	case RexIdle:
		if i == 1 { // breathing in
			p.dz -= 0.06
			p.chest -= 0.02
			p.neck += 0.03
			p.head += 0.03
			p.tailP[1] += 0.03
			p.tailP[3] += 0.04
		}
	case RexRoar:
		p.dz, p.dx = -0.3, -0.25
		p.pitch = -0.24
		p.chest = -0.08
		p.neck = -0.2
		p.head = 0.0
		p.jaw = 0.85
		p.tailP = [4]float64{0.26, 0.12, 0.1, 0.1}
		p.tailY = [4]float64{0.1, 0.12, 0.14, 0.14}
		p.leg[0] = legPose{x: 0.75, lean: 0.3}
		p.leg[1] = legPose{x: -0.7, lean: 0.62}
		p.arm = -0.5
	case RexBite:
		p.dz, p.dx = -0.5, 0.75
		p.pitch = 0.05
		p.chest = 0.08
		p.neck = 0.12
		p.head = -0.2
		p.jaw = 0.62
		p.tailP = [4]float64{-0.02, 0.06, 0.08, 0.1}
		p.leg[0] = legPose{x: -1.1, lean: 0.85}
		p.leg[1] = legPose{x: 1.25, lean: 0.25}
		p.arm = -0.3
	case RexEat:
		p.dz, p.dx = -1.0, 0.2
		p.pitch = 0.4
		p.chest = 0.2
		p.neck = 0.3
		p.head = 0.05
		p.jaw = 0.1
		if i == 1 {
			p.jaw = 0.5
			p.head = -0.05
			p.neck = 0.26
		}
		p.tailP = [4]float64{-0.12, -0.04, 0.04, 0.08}
		p.leg[0] = legPose{x: 0.5, lean: 0.5}
		p.leg[1] = legPose{x: -0.35, lean: 0.62}
	}
	return p
}

// apply turns p into bone transforms.
func (r *Rex) apply(p rexPose) []r3d.Mat {
	ps := r.skel.NewPose()
	ps.Off[r.root] = r3d.V(p.dx, 0, p.dz+rexLift)
	ps.Rot[r.root] = r3d.V(p.roll, p.pitch, p.yaw)
	ps.Rot[r.chest] = r3d.V(0, p.chest, p.chestYaw)
	ps.Rot[r.neck] = r3d.V(0, p.neck, p.neckYaw)
	ps.Rot[r.head] = r3d.V(0, p.head, p.headYaw)
	ps.Rot[r.jaw] = r3d.V(0, p.jaw, 0)
	for k, b := range r.tail {
		ps.Rot[b] = r3d.V(0, p.tailP[k], p.tailY[k])
	}
	rootRest := r.skel.Bones[r.root].Pivot
	rootAt := rootRest.Add(ps.Off[r.root])
	for side := 0; side < 2; side++ {
		ps.Rot[r.arm[side]] = r3d.V(0, p.arm, 0)
		lp := p.leg[side]
		hipRest := r.skel.Bones[r.thigh[side]].Pivot
		hip := rootAt.Add(r3d.RotY(p.pitch).Dir(hipRest.Sub(rootRest)))
		bx, bz := hipRest.X+rexFootX+lp.x, rexBall+lp.z
		ax, az := bx-rexMeta*math.Sin(lp.lean), bz+rexMeta*math.Cos(lp.lean)
		dx, dz := ax-hip.X, az-hip.Z
		d := math.Hypot(dx, dz)
		d = math.Max(math.Abs(rexThigh-rexShin)+0.05, math.Min((rexThigh+rexShin)*0.995, d))
		toAnkle := math.Atan2(dx, -dz)
		bend := math.Acos((rexThigh*rexThigh + d*d - rexShin*rexShin) / (2 * rexThigh * d))
		th1 := toAnkle + bend // the knee points forward
		kx, kz := hip.X+rexThigh*math.Sin(th1), hip.Z-rexThigh*math.Cos(th1)
		th2 := math.Atan2(ax-kx, -(az - kz))
		th3 := lp.lean
		ps.Rot[r.thigh[side]].Y = -th1 - p.pitch
		ps.Rot[r.shin[side]].Y = th1 - th2
		ps.Rot[r.meta[side]].Y = th2 - th3
		ps.Rot[r.foot[side]].Y = th3 + lp.toe
	}
	return r.skel.Matrices(ps)
}
