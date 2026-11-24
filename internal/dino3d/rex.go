// Package dino3d holds creatures modelled in 3D, in code, and baked into
// pixel-art sprites with package r3d: a low-poly mesh on a skeleton, posed
// per frame, rendered from the game's camera, shrunk to the game's pixel
// size, snapped to a small palette and outlined. So far: the tyrant.
//
// Nothing here touches Ebitengine; the client turns the baked frames into
// its own sprites (see client/rex3d.go).
package dino3d

import (
	"math"

	"survivalgame/internal/r3d"
)

// The tyrant's hide, after the drawing it replaces (client/dinos/rex.png):
// a dark olive back with darker bands, warm tan flanks, a cream belly.
var (
	rexBack   = r3d.Hex(0x6c5830)
	rexStripe = r3d.Hex(0x2a2016)
	rexFlank  = r3d.Hex(0xb08c4c)
	rexBelly  = r3d.Hex(0xdcc696)
	rexShank  = r3d.Hex(0x86683c) // the lower legs, a shade duller
	rexToe    = r3d.Hex(0x5e4c2c)
	rexClaw   = r3d.Hex(0x241c16)
	rexMouth  = r3d.Hex(0x601818)
	rexTongue = r3d.Hex(0xa23c3a)
	rexTooth  = r3d.Hex(0xf4ecd4)
	rexSocket = r3d.Hex(0x1a1210)
	rexEye    = r3d.Hex(0xffb428)
	rexInk    = r3d.Hex(0x1c1418)
)

// Lengths of the leg's three long bones, and how high the ball of the foot
// sits when the foot is flat on the ground.
const (
	rexThigh = 2.0
	rexShin  = 2.0
	rexMeta  = 1.35
	rexBall  = 0.2
	rexHipZ  = 4.0
	// The whole animal stands this much higher than its rest pose is modelled.
	rexLift = 0.95
	// The ball of a resting foot is this far ahead of the hip: under the
	// animal's weight, which is mostly in front of its hips.
	rexFootX = 0.45
)

// Rex is the tyrant's mesh and bones.
type Rex struct {
	skel r3d.Skeleton
	mesh r3d.Mesh

	root, chest, neck, head, jaw int
	tail                         [4]int
	thigh, shin, meta, foot      [2]int // 0: its left (+Y), 1: its right
	arm                          [2]int
	eyes                         [2]r3d.Vec3 // rest positions, on the head bone
}

// hide is the colour of the trunk where the skin faces up by up (1: the
// spine, -1: the belly line), s along the body for the bands.
func hide(up, s float64) r3d.RGB {
	c := rexFlank.Mix(rexBack, r3d.Smooth(0.45, 0.95, up))
	c = c.Mix(rexBelly, r3d.Smooth(-0.15, -0.55, up))
	// Blotches, so the hide is not one flat tone.
	c = c.Scale(1 + 0.1*math.Sin(s*7.3+up*5)*math.Sin(s*3.1-up*8.3))
	v := math.Sin(s*4.4 + 0.9*math.Sin(s*1.9) + 0.5*up)
	th := 0.2 + (1-up)*0.36
	return c.Mix(rexStripe, 0.9*r3d.Smooth(th, th+0.18, v))
}

// crPoint is a control point of a lofted body: where its centre is, its
// half-width and half-height, and the bone it follows.
type crPoint struct {
	x, z, ry, rz float64
	bone         int
}

func catmull(p0, p1, p2, p3, t float64) float64 {
	return 0.5 * ((2 * p1) + (-p0+p2)*t + (2*p0-5*p1+4*p2-p3)*t*t + (-p0+3*p1-3*p2+p3)*t*t*t)
}

// spineRings turns control points into rings, sub to a span, each ring
// square to the path and blended between the bones of the span's two ends.
func spineRings(cp []crPoint, sub int) []r3d.Ring {
	get := func(i int) crPoint { return cp[max(0, min(len(cp)-1, i))] }
	var rings []r3d.Ring
	for i := 0; i+1 < len(cp); i++ {
		a, b, c, d := get(i-1), get(i), get(i+1), get(i+2)
		n := sub
		if i+2 == len(cp) {
			n = sub + 1 // the last span includes its end
		}
		for j := 0; j < n; j++ {
			t := float64(j) / float64(sub)
			x, z := catmull(a.x, b.x, c.x, d.x, t), catmull(a.z, b.z, c.z, d.z, t)
			ry, rz := catmull(a.ry, b.ry, c.ry, d.ry, t), catmull(a.rz, b.rz, c.rz, d.rz, t)
			const e = 0.01
			tx := catmull(a.x, b.x, c.x, d.x, t+e) - catmull(a.x, b.x, c.x, d.x, t-e)
			tz := catmull(a.z, b.z, c.z, d.z, t+e) - catmull(a.z, b.z, c.z, d.z, t-e)
			tl := math.Hypot(tx, tz)
			up := r3d.V(-tz/tl, 0, tx/tl) // square to the path, upwards
			rings = append(rings, r3d.Ring{
				C: r3d.V(x, 0, z), U: r3d.V(0, max(ry, 0.01), 0), V: up.Mul(max(rz, 0.01)),
				B0: b.bone, B1: c.bone, W: r3d.Smooth(0, 1, t),
			})
		}
	}
	return rings
}

// limbRing is a cross-section of a leg bone hanging straight down in the
// rest pose: dz below the joint, fore is its half-depth front to back, side
// its half-width.
type limbRing struct{ dz, fore, side float64 }

func (r *Rex) limb(top r3d.Vec3, bone int, rs []limbRing, col func(t float64, th float64) r3d.RGB) {
	rings := make([]r3d.Ring, len(rs))
	for i, q := range rs {
		rings[i] = r3d.Ring{C: top.Add(r3d.V(0, 0, q.dz)), U: r3d.V(q.fore, 0, 0), V: r3d.V(0, q.side, 0), B0: bone, B1: bone}
	}
	lo, hi := rs[len(rs)-1].dz, rs[0].dz
	r.mesh.Loft(rings, 12, func(i int, th float64, _ r3d.Vec3) r3d.RGB {
		return col((rs[i].dz-hi)/(lo-hi), th)
	}, 0)
}

// NewRex builds the tyrant.
func NewRex() *Rex {
	r := &Rex{}
	s := &r.skel
	r.root = s.Add("root", -1, r3d.V(0, 0, 4.1))
	r.chest = s.Add("chest", r.root, r3d.V(1.0, 0, 4.0))
	r.neck = s.Add("neck", r.chest, r3d.V(2.4, 0, 4.4))
	headAt := r3d.V(3.4, 0, 5.8)
	r.head = s.Add("head", r.neck, headAt)
	r.jaw = s.Add("jaw", r.head, headAt.Add(r3d.V(0.1, 0, -0.3).Mul(rexHead)))
	tailX := [4]float64{-0.8, -1.6, -2.5, -3.4}
	parent := r.root
	for i, x := range tailX {
		r.tail[i] = s.Add("tail", parent, r3d.V(x, 0, 4.2))
		parent = r.tail[i]
	}
	for side := 0; side < 2; side++ {
		y := 1.05
		if side == 1 {
			y = -y
		}
		hip := r3d.V(0.1, y, rexHipZ)
		r.thigh[side] = s.Add("thigh", r.root, hip)
		r.shin[side] = s.Add("shin", r.thigh[side], hip.Add(r3d.V(0, 0, -rexThigh)))
		r.meta[side] = s.Add("meta", r.shin[side], hip.Add(r3d.V(0, 0, -rexThigh-rexShin)))
		r.foot[side] = s.Add("foot", r.meta[side], hip.Add(r3d.V(0, 0, -rexThigh-rexShin-rexMeta)))
		r.arm[side] = s.Add("arm", r.chest, r3d.V(2.3, y*0.85, 3.3))
	}

	// The trunk, one skin from the tip of the tail to the back of the skull.
	t := r.tail
	trunk := spineRings([]crPoint{
		{-4.5, 4.4, 0.03, 0.04, t[3]},
		{-3.9, 4.36, 0.2, 0.27, t[3]},
		{-3.0, 4.3, 0.36, 0.5, t[2]},
		{-2.1, 4.24, 0.5, 0.68, t[1]},
		{-1.2, 4.16, 0.72, 0.98, t[0]},
		{-0.4, 4.06, 1.0, 1.34, r.root},
		{0.5, 3.9, 1.22, 1.62, r.root},
		{1.4, 3.92, 1.22, 1.6, r.chest},
		{2.1, 4.15, 1.08, 1.38, r.chest},
		{2.65, 4.6, 0.9, 1.14, r.neck},
		{3.0, 5.15, 0.78, 0.98, r.neck},
		{3.25, 5.6, 0.7, 0.86, r.head},
		{3.45, 5.9, 0.6, 0.72, r.head},
	}, 6)
	r.mesh.Loft(trunk, 20, func(i int, th float64, p r3d.Vec3) r3d.RGB {
		return hide(math.Sin(th), p.X)
	}, 0)

	r.buildHead(headAt)
	for side := 0; side < 2; side++ {
		r.buildLeg(side)
		r.buildArm(side)
	}
	return r
}

// skullRing is a cross-section of the skull or the jaw, in the head's own
// frame and before rexHead scales it: lx ahead of the neck joint, half-width
// w, from bot up to top.
type skullRing struct{ lx, w, top, bot float64 }

// rexHead enlarges the head: it is what the eye goes to at this size.
const rexHead = 1.4

func (r *Rex) skullLoft(at r3d.Vec3, bone int, rs []skullRing, exp float64, col func(lx, th float64) r3d.RGB) {
	rings := make([]r3d.Ring, len(rs))
	for i, q := range rs {
		rings[i] = r3d.Ring{
			C: at.Add(r3d.V(q.lx, 0, (q.top+q.bot)/2).Mul(rexHead)), U: r3d.V(0, q.w*rexHead, 0), V: r3d.V(0, 0, (q.top-q.bot)/2*rexHead),
			Exp: exp, B0: bone, B1: bone,
		}
	}
	r.mesh.Loft(rings, 20, func(i int, th float64, _ r3d.Vec3) r3d.RGB { return col(rs[i].lx, th) }, 0)
}

func (r *Rex) buildHead(at r3d.Vec3) {
	const hs = rexHead
	hp := func(lx, y, z float64) r3d.Vec3 { return at.Add(r3d.V(lx, y, z).Mul(hs)) }
	// The skull: deep and wide over the cheeks, a long boxy snout.
	skull := []skullRing{
		{-0.4, 0.32, 0.4, -0.05},
		{-0.25, 0.56, 0.68, -0.24},
		{0.2, 0.7, 0.8, -0.3},
		{0.65, 0.66, 0.78, -0.3},
		{1.0, 0.54, 0.62, -0.28},
		{1.45, 0.48, 0.58, -0.28},
		{1.9, 0.46, 0.56, -0.28},
		{2.2, 0.4, 0.48, -0.26},
		{2.36, 0.24, 0.26, -0.12},
	}
	r.skullLoft(at, r.head, skull, 3, func(lx, th float64) r3d.RGB {
		up := math.Sin(th)
		if up < -0.7 && lx > 0.4 && lx < 2.3 {
			return rexMouth // the roof of the mouth
		}
		c := rexFlank.Mix(rexBack, r3d.Smooth(0.3, 0.8, up))
		if lx > 1.0 { // a darker muzzle top
			c = c.Mix(rexStripe, 0.3*r3d.Smooth(0.6, 0.95, up))
		}
		return c
	})
	// The lower jaw, deep at the hinge, hung from its own bone.
	jaw := []skullRing{
		{-0.3, 0.3, -0.26, -0.55},
		{0, 0.6, -0.28, -0.95},
		{0.5, 0.58, -0.28, -0.92},
		{1.0, 0.44, -0.27, -0.76},
		{1.6, 0.38, -0.27, -0.66},
		{2.0, 0.33, -0.27, -0.6},
		{2.16, 0.18, -0.3, -0.5},
	}
	r.skullLoft(at, r.jaw, jaw, 3, func(lx, th float64) r3d.RGB {
		up := math.Sin(th)
		if up > 0.7 && lx > 0.4 && lx < 2.1 {
			return rexTongue
		}
		return rexFlank.Mix(rexBelly, r3d.Smooth(0.1, -0.5, up))
	})
	for side := 0; side < 2; side++ {
		y := 1.0
		if side == 1 {
			y = -1
		}
		// A heavy brow over a dark socket: the scowl.
		r.mesh.Ball(hp(0.66, y*0.52, 0.72), 0.36*hs, 0.2*hs, 0.13*hs, rexBack.Mix(rexStripe, 0.5), r.head, 0)
		r.mesh.Ball(hp(0.72, y*0.6, 0.42), 0.17*hs, 0.12*hs, 0.13*hs, rexSocket, r.head, 0.3)
		r.eyes[side] = hp(0.74, y*0.73, 0.43)
		// A low ridge down the muzzle ahead of each eye.
		r.mesh.Ball(hp(1.25, y*0.3, 0.6), 0.26*hs, 0.12*hs, 0.08*hs, rexBack.Mix(rexStripe, 0.3), r.head, 0)
	}
	// Teeth: a row down each side of the upper jaw, a shorter row below.
	width := func(rs []skullRing, lx float64) (w, top, bot float64) {
		for i := 0; i+1 < len(rs); i++ {
			if lx <= rs[i+1].lx {
				f := (lx - rs[i].lx) / (rs[i+1].lx - rs[i].lx)
				return rs[i].w + (rs[i+1].w-rs[i].w)*f, rs[i].top + (rs[i+1].top-rs[i].top)*f, rs[i].bot + (rs[i+1].bot-rs[i].bot)*f
			}
		}
		l := rs[len(rs)-1]
		return l.w, l.top, l.bot
	}
	for i := 0; i < 7; i++ {
		lx := 0.9 + float64(i)*0.2
		w, _, bot := width(skull, lx)
		long := 0.3 + 0.1*math.Sin(float64(i)*1.9) // uneven, like the real thing
		if i == 6 {
			long = 0.26
		}
		for _, y := range [2]float64{w - 0.07, -(w - 0.07)} {
			r.mesh.Spike(hp(lx, y, bot+0.04), hp(lx-0.04, y, bot+0.04-long), 0.085*hs, rexTooth, rexTooth, r.head, 0.75)
		}
	}
	for _, y := range [2]float64{0.14, -0.14} { // the front pair
		r.mesh.Spike(hp(2.26, y, -0.2), hp(2.26, y, -0.46), 0.08*hs, rexTooth, rexTooth, r.head, 0.75)
	}
	for i := 0; i < 5; i++ {
		lx := 0.95 + float64(i)*0.24
		w, top, _ := width(jaw, lx)
		for _, y := range [2]float64{w - 0.1, -(w - 0.1)} {
			r.mesh.Spike(hp(lx, y, top-0.04), hp(lx, y, top+0.2), 0.075*hs, rexTooth, rexTooth, r.jaw, 0.75)
		}
	}
}

func (r *Rex) buildLeg(side int) {
	hip := r.skel.Bones[r.thigh[side]].Pivot
	// The thigh: a great slab of muscle, banded like the back at the top.
	r.limb(hip, r.thigh[side], []limbRing{
		{0.8, 0.45, 0.26}, {0.4, 1.0, 0.48}, {-0.25, 1.2, 0.62}, {-0.85, 1.06, 0.58},
		{-1.4, 0.78, 0.5}, {-1.85, 0.6, 0.44}, {-2.12, 0.34, 0.26},
	}, func(t, th float64) r3d.RGB {
		c := rexBack.Mix(rexFlank, r3d.Smooth(0.0, 0.3, t))
		band := math.Sin(t*14 + 0.4 + 1.2*math.Cos(th))
		return c.Mix(rexStripe, 0.85*r3d.Smooth(0.4, 0.75, band)*r3d.Smooth(0.75, 0.3, t))
	})
	knee := hip.Add(r3d.V(0, 0, -rexThigh))
	r.limb(knee, r.shin[side], []limbRing{
		{0.24, 0.34, 0.28}, {0, 0.6, 0.44}, {-0.45, 0.7, 0.46}, {-1.0, 0.52, 0.38}, {-1.6, 0.38, 0.32}, {-1.88, 0.36, 0.31}, {-2.04, 0.18, 0.16},
	}, func(t, th float64) r3d.RGB { return rexFlank.Mix(rexShank, r3d.Smooth(0.1, 0.7, t)) })
	ankle := knee.Add(r3d.V(0, 0, -rexShin))
	r.limb(ankle, r.meta[side], []limbRing{
		{0.2, 0.2, 0.18}, {0, 0.4, 0.33}, {-0.5, 0.34, 0.3}, {-1.1, 0.36, 0.36}, {-1.3, 0.36, 0.38}, {-1.45, 0.16, 0.18},
	}, func(t, th float64) r3d.RGB { return rexShank.Mix(rexToe, r3d.Smooth(0.5, 1, t)) })
	// Three toes, splayed, each ending in a dark claw.
	ball := ankle.Add(r3d.V(0, 0, -rexMeta))
	for k := -1; k <= 1; k++ {
		a := float64(k) * 0.5
		d := r3d.V(math.Cos(a), math.Sin(a), 0)
		lat := r3d.V(-d.Y, d.X, 0)
		long := 0.85
		if k == 0 {
			long = 1.05
		}
		rings := []r3d.Ring{}
		for _, q := range [][3]float64{{-0.1, 0.16, 0.06}, {0.1, 0.3, 0.06}, {0.55, 0.27, 0.02}, {0.85, 0.22, -0.02}, {1, 0.14, -0.04}} {
			c := ball.Add(d.Mul(q[0] * long)).Add(r3d.V(0, 0, q[2]))
			rings = append(rings, r3d.Ring{C: c, U: lat.Mul(q[1]), V: r3d.V(0, 0, q[1]*0.85), B0: r.foot[side], B1: r.foot[side]})
		}
		r.mesh.Loft(rings, 8, func(int, float64, r3d.Vec3) r3d.RGB { return rexToe }, 0)
		tip := ball.Add(d.Mul(long)).Add(r3d.V(0, 0, -0.05))
		r.mesh.Spike(tip, tip.Add(d.Mul(0.42)).Add(r3d.V(0, 0, -0.12)), 0.14, rexClaw, rexClaw, r.foot[side], 0.2)
	}
}

func (r *Rex) buildArm(side int) {
	sh := r.skel.Bones[r.arm[side]].Pivot
	out := 1.0
	if side == 1 {
		out = -1
	}
	sh = sh.Add(r3d.V(0, out*0.14, 0))
	elbow := sh.Add(r3d.V(-0.24, out*0.1, -0.56))
	wrist := elbow.Add(r3d.V(0.5, 0, -0.28))
	seg := func(a, b r3d.Vec3, ra, rb float64) {
		ax := b.Sub(a).Norm()
		u := ax.Cross(r3d.V(0, 1, 0)).Norm()
		v := r3d.V(0, 1, 0)
		rings := []r3d.Ring{
			{C: a.Sub(ax.Mul(ra * 0.6)), U: u.Mul(ra * 0.5), V: v.Mul(ra * 0.5)},
			{C: a, U: u.Mul(ra), V: v.Mul(ra)},
			{C: b, U: u.Mul(rb), V: v.Mul(rb)},
			{C: b.Add(ax.Mul(rb * 0.6)), U: u.Mul(rb * 0.5), V: v.Mul(rb * 0.5)},
		}
		for i := range rings {
			rings[i].B0, rings[i].B1 = r.arm[side], r.arm[side]
		}
		r.mesh.Loft(rings, 8, func(int, float64, r3d.Vec3) r3d.RGB { return rexFlank.Mix(rexShank, 0.5) }, 0)
	}
	seg(sh, elbow, 0.22, 0.17)
	seg(elbow, wrist, 0.17, 0.12)
	for _, dy := range [2]float64{-0.08, 0.08} { // two fingers
		b := wrist.Add(r3d.V(0, dy, 0))
		r.mesh.Spike(b, b.Add(r3d.V(0.3, dy*0.6, -0.18)), 0.08, rexClaw, rexClaw, r.arm[side], 0.2)
	}
}
