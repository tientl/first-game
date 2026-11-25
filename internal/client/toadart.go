package client

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The toads at HD (sim/toad.go): the green poison toad, bright and blotched
// with a yellow belly, and the glow toad, dark with spots that shine at
// night. Both are one drawing with two skins. A toad sits (its throat
// swelling now and then), swells up when touched, and hops: eight frames
// from the crouch through the leap, legs trailing straight behind, to the
// landing on its fore feet. See critterhd.go for the tools.

const (
	toadSit  = iota // frame 1: the throat puffed out
	toadHop         // frames 1 to toadHopFrames
	toadPuff        // swollen up: the green toad, touched
	toadPoses
)

// toadHopFrames is how many frames a hop has.
const toadHopFrames = 8

// toadPose is how a toad holds itself: its body turned by tilt (negative:
// nose up), its hind legs stretched by ext (0 folded, 1 straight behind),
// its fore leg's two bones at armU and armF (radians from pointing ahead,
// turning down), its body squashed to sq of its height and swollen by puff.
type toadPose struct{ tilt, ext, armU, armF, sq, puff float64 }

// toadHopPose is the pose of each frame of a hop; frame 0 is sitting.
var toadHopPose = [toadHopFrames + 1]toadPose{
	{0, 0, 1.8, 1.25, 1, 0},
	{0.1, 0, 2.1, 1.05, 0.86, 0},     // 1: crouched to spring
	{-0.36, 0.55, 2.3, 0.7, 1.04, 0}, // 2: pushing off, fore feet leaving the ground
	{-0.5, 1, 2.6, 0.5, 1.08, 0},     // 3: away, legs straight behind
	{-0.3, 1, 2.4, 0.35, 1.06, 0},    // 4: rising
	{-0.04, 0.72, 1.2, 0.35, 1, 0},   // 5: the top of the leap, reaching forward
	{0.3, 0.42, 0.8, 0.6, 1, 0},      // 6: coming down
	{0.42, 0.16, 1.4, 1.05, 0.95, 0}, // 7: the fore feet land
	{0.1, 0, 1.9, 1.15, 0.86, 0},     // 8: settling, squashed
}

// toadHopBob is how high off the ground each frame of a hop is, in art pixels.
var toadHopBob = [toadHopFrames + 1]float64{0, 0, 1.5, 4.5, 7, 7.5, 5.5, 2, 0}

var (
	toadGreen   = mkRamp(0x123c1a, 0x1c6024, 0x2c882e, 0x4aae3c, 0x7ad054, 0xb2ec82)
	toadBlotch  = mkRamp(0x06140c, 0x0c2212, 0x14321a, 0x1c4222, 0x245028, 0x2c5e2e)
	toadBelly   = mkRamp(0x94701a, 0xc49c28, 0xe8c844, 0xf8e478, 0xfff4b8)
	toadGland   = mkRamp(0x9c4410, 0xd06c18, 0xf09828, 0xfcc458)
	glowSkin    = mkRamp(0x0a1826, 0x122a42, 0x1a405e, 0x285a7a, 0x3c7896, 0x5c9cb4)
	glowBelly   = mkRamp(0x2c7c7c, 0x4ca8a0, 0x7cd4c0, 0xb4f0dc, 0xe4fff4)
	glowSpot    = hex(0x74f8c8)
	glowSpotHot = hex(0xd8fff0)
	maskOff     = color.RGBA{1, 0, 1, 255} // in a glow mask: something that does not glow, hiding what is behind it
)

// toadHD draws a toad: glow picks the glow toad, mask only what of it
// shines in the dark (for the light drawn over the night).
func toadHD(glow, mask bool, pose, f int) (*canvas, float64, float64) {
	q := newCritAt(27, 23, 6, 3)
	p := toadHopPose[0]
	throat := 0.0
	switch pose {
	case toadHop:
		p = toadHopPose[min(max(f, 0), toadHopFrames)]
	case toadPuff:
		p = toadPose{-0.08, 0.05, 1.62, 1.5, 1.12, 0.24}
	default:
		if f == 1 {
			throat = 1
		}
	}
	const px, py = 7.5, 9.6 // what the body turns about
	lift := 0.0
	if pose == toadPuff {
		lift = 1 // up on stiff legs
	}
	T := func(s shape) shape { return s.turned(px, py, p.tilt).moved(0, -lift*cK) }
	P := func(x, y float64) (float64, float64) {
		x, y = turnXY(x, y, px, py, p.tilt)
		return x, y - lift
	}
	// loc is where a pixel lies on the body itself, however it is turned:
	// the markings stay put on the skin.
	loc := func(x, y int) (float64, float64) {
		return turnXY((float64(x)+0.5)/cK, (float64(y)+0.5)/cK+lift, px, py, -p.tilt)
	}
	skinR, bellyR := toadGreen, toadBelly
	if glow {
		skinR, bellyR = glowSkin, glowBelly
	}
	// spot is how much of a shining spot (glow toad) or a dark blotch
	// (green toad) lies at body point (u, v).
	spot := func(u, v float64) float64 {
		if glow {
			cu, cv := math.Floor(u/2.1), math.Floor(v/1.7)
			su, sv := (cu+0.3+0.4*hashf(int(cu), int(cv), 951))*2.1, (cv+0.3+0.4*hashf(int(cu), int(cv), 952))*1.7
			if hashf(int(cu), int(cv), 953) < 0.2 {
				return 0
			}
			return clamp01(1.6 - math.Hypot(u-su, v-sv)/0.42)
		}
		return clamp01((vnoise(u*0.62+3, v*0.8+7, 961) - 0.56) * 9)
	}
	skin := func(s shape, bias float64) shader {
		return func(x, y int, _ float64) color.RGBA {
			u, v := loc(x, y)
			l := s.lum(x, y, 2.2*cK)*0.9 + 0.06 + bias
			k := spot(u, v)
			switch {
			case glow && k > 0.75:
				return glowSpotHot
			case glow && k > 0:
				return glowSpot
			case mask:
				return maskOff
			case k > 0.5:
				return toneOf(toadBlotch, l)
			}
			col := toneOf(skinR, l)
			if hashf(int(u*2.4), int(v*2.4), 962) < 0.1 {
				col = toneOf(skinR, l+0.2) // a wart catching the light
			}
			return col
		}
	}
	pale := func(s shape, bias float64) shader {
		return func(x, y int, _ float64) color.RGBA {
			if mask && !glow {
				return maskOff
			}
			return toneOf(bellyR, s.lum(x, y, 1.8*cK)*0.85+0.12+bias)
		}
	}
	// A leg: three bones from the hip, each at an angle from pointing ahead,
	// turning down and back.
	hind := func(hx, hy, bias float64, far bool) {
		a1 := lerpf(-0.18, 2.88, p.ext)
		a2 := lerpf(2.62, 3.0, p.ext)
		a3 := lerpf(0.09, 2.93, p.ext)
		kx, ky := hx+2.9*math.Cos(a1), hy+2.9*math.Sin(a1)
		ax, ay := kx+3.3*math.Cos(a2), ky+3.3*math.Sin(a2)
		tx, ty := ax+2.8*math.Cos(a3), ay+2.8*math.Sin(a3)
		hx, hy = P(hx, hy)
		kx, ky = P(kx, ky)
		ax, ay = P(ax, ay)
		tx, ty = P(tx, ty)
		if p.ext < 0.1 && pose != toadPuff { // sitting: the foot lies flat on the ground whatever the body does
			ty, ay = 11.7, math.Min(ay, 11.2)
		}
		shin := aCap(kx, ky, ax, ay, 1.05, 0.72)
		q.fill(shin, skin(shin, bias-0.06))
		foot := aCap(ax, ay, tx, ty, 0.68, 0.5)
		q.fill(foot, pale(foot, bias-0.1))
		if !far { // the long toes, spread
			for _, d := range [][2]float64{{0.9, -0.5}, {1.1, 0.25}} {
				toe := aCap(tx, ty, tx+d[0]*math.Cos(a3)-d[1]*math.Sin(a3), ty+d[0]*math.Sin(a3)*0.4+d[1]*0.5, 0.36, 0.3)
				q.fill(toe, pale(toe, bias-0.1))
			}
		}
		thigh := sBlend(0.8, aEll(hx, hy, 1.9, 1.7), aCap(hx, hy, kx, ky, 1.6, 1.2))
		q.fill(thigh, skin(thigh, bias))
	}
	fore := func(sx, sy, bias float64) {
		ex, ey := sx+1.5*math.Cos(p.armU), sy+1.5*math.Sin(p.armU)
		hx, hy := ex+1.7*math.Cos(p.armF), ey+1.7*math.Sin(p.armF)
		sx, sy = P(sx, sy)
		ex, ey = P(ex, ey)
		hx, hy = P(hx, hy)
		if pose != toadHop || f == 0 || f == 1 || f >= 7 {
			hy = math.Min(hy, 11.7) // planted
		}
		arm := aLeg(sx, sy, ex, ey, hx, hy, 0.95, 0.72, 0.52)
		q.fill(arm, skin(arm, bias))
		hand := aEll(hx+0.5, hy+0.1, 0.95, 0.45)
		q.fill(hand, pale(hand, bias-0.06))
	}

	hind(5.4, 9.6, -0.2, true)
	fore(10.9, 8.9, -0.2)
	// The body: a squat rump and chest, the broad flat head, all one lump.
	g := 1 + p.puff
	body := T(sBlend(1.6,
		aEll(6.0, 8.5+1.2*(1-p.sq), 4.2*g/math.Sqrt(p.sq), 3.1*p.sq*g),
		aEll(9.2, 7.7+1.0*(1-p.sq), 3.3*g, 2.8*p.sq*g),
		aEll(11.8, 6.8, 2.5, 2.05),
		aEll(13.4, 7.2, 1.2, 1.2)))
	belly := T(aEll(8.6, 10.4, 5.2*g, 1.9))
	chin := T(aEll(12.2, 8.5, 2.4, 1.1))
	under := func(x, y float64) bool { return belly.d(x*cK, y*cK) < 0 || chin.d(x*cK, y*cK) < 0 }
	q.fill(body, where(skin(body, 0), pale(body, 0), under))
	if throat > 0 || pose == toadPuff { // the sac under the chin, blown out
		tx, ty := P(11.9, 9.3)
		sac := aEll(tx, ty, 2.1, 1.6)
		q.fill(sac, pale(sac, 0.14))
	}
	if !glow { // the poison gland behind the eye: a swollen orange ridge
		gx, gy := P(9.3, 5.6)
		gl := aEll(gx, gy, 1.5, 0.8).turned(gx, gy, p.tilt+0.25)
		q.fill(gl, func(x, y int, _ float64) color.RGBA {
			if mask {
				return maskOff
			}
			return toneOf(toadGland, gl.lum(x, y, 1*cK)*0.9+0.1)
		})
	}
	// The mouth: a long line back from the snout.
	if !mask {
		mx0, my0 := P(10.2, 7.9)
		mx1, my1 := P(14.3, 7.6)
		q.mark(aCap(mx0, my0, mx1, my1, 0.17, 0.17), plain(mix(skinR[0], colInk, 0.5)))
		nx, ny := P(14.1, 6.6)
		q.dot(nx, ny, 0, 0, colInk) // a nostril
	}
	// The eye, bulging on top of the head: a gold iris, the pupil a slit lying flat.
	ex, ey := P(11.5, 5.0)
	lid := aEll(ex, ey, 1.4, 1.35)
	q.fill(lid, skin(lid, 0.1))
	iris := hex(0xf0a41c)
	if glow {
		iris = hex(0x9cf8d4)
	}
	q.flat(aEll(ex+0.15, ey+0.1, 1.05, 1.0), plain(mix(iris, colInk, 0.8)))
	q.flat(aEll(ex+0.15, ey+0.1, 0.82, 0.8), plain(iris))
	if !mask {
		q.flat(aEll(ex+0.2, ey+0.12, 0.62, 0.3), plain(colInk))
		q.dot(ex, ey, -1, -1, colWhite)
	}
	hind(5.0, 10.2, 0.02, false)
	fore(10.5, 9.4, 0.04)

	if mask { // only the light is left
		for i, c := range q.c.px {
			if c == maskOff {
				q.c.px[i] = color.RGBA{}
			}
		}
		return q.c, 8 + float64(q.ox)/cK, 12 + float64(q.oy)/cK
	}
	return q.done(0.72, 8, 12)
}

type toadKey struct {
	glow, mask    bool
	side, pose, f int
}

var toadCache = map[toadKey]sprite{}

// toadArt is frame f of pose of a toad facing side (1: left), or with mask
// the light of it, drawn the first time it is wanted.
func toadArt(glow, mask bool, side, pose, f int) sprite {
	key := toadKey{glow, mask, side, pose, f}
	if s, ok := toadCache[key]; ok {
		return s
	}
	var s sprite
	if side == 1 {
		s = mirrorSprite(toadArt(glow, mask, 0, pose, f))
	} else {
		s = critSprite(toadHD(glow, mask, pose, f))
	}
	toadCache[key] = s
	return s
}

// toadFrame is the pose and frame a toad is shown in, and how high off
// the ground it is.
func (g *Game) toadFrame(c protocol.CreatureView, t int) (pose, f int, bob float64) {
	age := g.blowAge(c) // frames since it began what it is doing
	switch c.State {
	case sim.CreatureStrike:
		return toadPuff, 0, 0
	case sim.CreatureWander: // a hop: the crouch for a blink, then through the air
		hop := sim.ToadHopTicks * 60 / sim.TicksPerSecond
		f = min(toadHopFrames-1, 1+age*(toadHopFrames-1)/hop)
		return toadHop, f, toadHopBob[f]
	case sim.CreatureIdle:
		if age >= 0 && age < 5 && c.ID != 0 {
			return toadHop, toadHopFrames, 0 // just landed
		}
	}
	if (t/22)%5 == 0 { // its throat works
		return toadSit, 1, 0
	}
	return toadSit, 0, 0
}

// toadSprite is creatureSprite for the toads.
func (g *Game) toadSprite(c protocol.CreatureView, side, t int) (sprite, float64, bool) {
	if !sim.IsToad(c.Kind) {
		return sprite{}, 0, false
	}
	pose, f, bob := g.toadFrame(c, t)
	return toadArt(c.Kind == sim.CreatureGlowToad, false, side, pose, f), bob, true
}

// toadGlows draws the light of the glow toads over the night: with punch,
// holes in the darkness round each; without, their soft green glow.
func (g *Game) toadGlows(dst drawTarget, cam camera, punch bool, dark float64) {
	alpha := g.alpha()
	for _, c := range g.curr.Creatures {
		if c.Kind != sim.CreatureGlowToad || c.State == sim.CreatureHidden {
			continue
		}
		x, y, _ := g.lerpCreature(c, alpha)
		if g.hiddenInCave(x, y) {
			continue
		}
		sx, sy := cam.toScreenFine(x, y)
		pulse := 1 + 0.12*math.Sin(float64(g.frame)*0.05+float64(c.ID)*1.7)
		op := g.lightOpAt(sx, sy-3, sim.ToadGlowRadius*pulse)
		if punch {
			op.ColorScale.ScaleAlpha(0.4)
			op.Blend = ebiten.BlendDestinationOut
		} else {
			k := float32(0.22 * dark * pulse)
			op.ColorScale.Scale(k*0.3, k, k*0.72, k)
			op.Blend = ebiten.BlendLighter
		}
		dst.DrawImage(g.art.light, op)
	}
}

// drawToadGlow draws the shining spots of the glow toads bright over the dark.
func (g *Game) drawToadGlow(dst drawTarget, cam camera, dark float64) {
	if dark < 0.05 {
		return
	}
	alpha := g.alpha()
	for _, c := range g.curr.Creatures {
		if c.Kind != sim.CreatureGlowToad || c.State == sim.CreatureHidden {
			continue
		}
		x, y, _ := g.lerpCreature(c, alpha)
		if g.hiddenInCave(x, y) {
			continue
		}
		sx, sy := cam.toScreenFine(x, y)
		align(dst, sx, sy)
		side := 0
		if c.FaceLeft {
			side = 1
		}
		pose, f, bob := g.toadFrame(c, g.frame+int(c.ID)*7)
		m := toadArt(true, true, side, pose, f)
		k := float32(math.Min(1, dark*1.3) * (0.8 + 0.2*math.Sin(float64(g.frame)*0.05+float64(c.ID)*1.7)))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx-m.ax, sy-bob-m.ay)
		op.ColorScale.Scale(k, k, k, k)
		op.Blend = ebiten.BlendLighter
		drawSprite(dst, m, op)
	}
	alignCamera(dst)
}
