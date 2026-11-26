package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// A rider on the jointed figure. Each mount has its seat (seat in ride.go)
// and its way of being ridden: a raptor runs, so its rider rides it like a
// jockey, low over the neck, the knees up; a three-horn walks, and its rider
// sits up holding the reins to its frill; an Ankylosaurus is slow and broad,
// and its rider sits high with the thighs spread over its back, rocking.
// The rider rises and settles with each step of the mount's own stride (the
// clock creatureSprite strides it by), sways when it walks, breathes when
// it stands. The far leg is drawn before the mount, so it is behind it, the
// near one after. Getting on (moMount) the rider puts a hand on its back,
// springs and swings a leg over; getting off (moDismount) the reverse, down
// to where the server set them.
//
// The beasts are only ever drawn from the side, so a rider is too.

// riderHip is how far below the seat a rider's feet are put: the hips on
// the cloth.
const riderHip = hipHeight - 1.5

// strideOf is how many frames a stride of mount kind k takes as
// creatureSprite draws it walking (twice its step's period).
func strideOf(k sim.CreatureKind) int {
	switch k {
	case sim.CreatureRaptor:
		return 16
	case sim.CreatureAnkylo:
		return 32
	}
	return 24
}

// rideMotion is how a mount of kind k is ridden, moving or standing.
func rideMotion(k sim.CreatureKind, moving bool) motionID {
	switch {
	case !moving:
		return moRideIdle
	case k == sim.CreatureRaptor:
		return moRideRun
	case k == sim.CreatureAnkylo:
		return moRideWide
	}
	return moRide
}

// riderPose is where p sits on c drawn at d (the feet's point), facing dir,
// with the halves to draw, and how far through getting on they are (1:
// on).
func (g *Game) riderPose(p protocol.PlayerView, c protocol.CreatureView, d drawable) (x, fy float64, dir sim.Dir, lo loKey, up upKey, on float64) {
	back, height := seat(c.Kind)
	side := 1.0
	dir = sim.DirRight
	if c.FaceLeft {
		side, dir = -1, sim.DirLeft
	}
	_, bob := g.creatureSprite(c, d.moving)
	x, fy = d.sx-back*side, d.sy-bob-height+riderHip
	m := rideMotion(c.Kind, d.moving)
	fr := moRideIdle.frameAt(float64(g.frame + int(p.ID)*29))
	if m != moRideIdle { // with the mount's stride
		n := strideOf(c.Kind)
		ph := float64((g.frame+int(c.ID)*7)%n) / float64(n)
		fr = int(ph*8) % 8
	}
	lo, up = m.keys(dir, fr)
	on = 1
	st := g.moveState(p.ID)
	if age := float64(g.frame - st.mountAt); st.mountAt != never && age >= 0 && age < moMount.span() {
		fr := moMount.frameAt(age)
		lo, up = moMount.keys(dir, fr)
		on = [5]float64{0, 0.35, 0.8, 1, 1}[fr]
		// From the ground beside it, in front of it, up onto the seat.
		gx, gy := d.sx-back*side-3*side, d.sy+3
		x, fy = lerpf(gx, x, on), lerpf(gy, fy, on)
	}
	return x, fy, dir, lo, up, on
}

// drawRiderBehind draws what of p is behind its mount c: the far leg.
func (g *Game) drawRiderBehind(dst drawTarget, p protocol.PlayerView, c protocol.CreatureView, d drawable) {
	if !(playerHD && playerJointed) || p.Dead {
		return
	}
	x, fy, _, lo, up, on := g.riderPose(p, c, d)
	if on < 0.75 {
		return // still on the ground in front of it
	}
	if _, ok := g.jolting(p.ID); ok {
		return // thrown about: drawn whole by drawJolted
	}
	lo.frame |= legFar
	f := figDraw{b: g.art.bodyIn(p.Look, p.Wear), up: up, lo: lo, legsOnly: true}
	drawFig(dst, f, x, fy)
}

// drawRiderHD draws p astride c: the saddle cloth, the rider but for the
// far leg, and the reins.
func (g *Game) drawRiderHD(dst drawTarget, p protocol.PlayerView, c protocol.CreatureView, d drawable) {
	x, fy, dir, lo, up, on := g.riderPose(p, c, d)
	side := facingSide(dir)
	back, height := seat(c.Kind)
	_, bob := g.creatureSprite(c, d.moving)
	sx, sy := d.sx-back*side, d.sy-bob-height
	saddleCloth(dst, c.Kind, sx, sy)
	if on >= 0.75 {
		lo.frame |= legNear
	}
	rider := func(dst drawTarget, x, fy float64) {
		g.riderFig(dst, p, c, dir, lo, up, x, fy, on)
	}
	if age, ok := g.jolting(p.ID); ok {
		lo.frame &^= legNear
		g.drawJolted(dst, age, dir, x, fy, func(buf drawTarget, x, fy float64) {
			g.riderFig(buf, p, c, dir, lo, up, x, fy, on)
		})
		return
	}
	rider(dst, x, fy)
}

// riderFig draws the rider with the reins in the hands when seated.
func (g *Game) riderFig(dst drawTarget, p protocol.PlayerView, c protocol.CreatureView, dir sim.Dir, lo loKey, up upKey, x, fy, on float64) {
	g.drawFigureJ(dst, p, dir, lo, up, x, fy, p.Armor)
	if on < 1 || p.Acting {
		return
	}
	// The reins: from the hands to the mount's neck or frill, sagging.
	nx, ny, ok := reinsTo(c.Kind)
	if !ok {
		return
	}
	u := g.art.bodyIn(p.Look, p.Wear).upper(up)
	side := facingSide(dir)
	_, lob := 0.0, g.art.bodyIn(p.Look, p.Wear).lower(lo).bob
	ux, uy := x, fy-lob
	tx, ty := x+nx*side, fy+ny
	leather := color.RGBA{70, 44, 28, 255}
	for i := 0; i < 2; i++ {
		h := u.hand[i]
		hx, hy := ux+h.x, uy+h.y
		px, py := hx, hy
		for s := 0.1; s <= 1.001; s += 0.1 {
			qx := lerpf(hx, tx, s)
			qy := lerpf(hy, ty, s) + 2.2*math.Sin(math.Pi*s) // the sag
			strokeLine(dst, float32(px), float32(py), float32(qx), float32(qy), 0.6, leather, false)
			px, py = qx, qy
		}
	}
}

// reinsTo is where the reins of a mount of kind k are tied, from the
// rider's feet facing right, in art pixels; none for a beast ridden by its
// spikes.
func reinsTo(k sim.CreatureKind) (x, y float64, ok bool) {
	switch k {
	case sim.CreatureRaptor:
		return 13, -20, true
	case sim.CreatureTrike:
		return 14, -15, true
	}
	return 0, 0, false
}

// saddleCloth draws the cloth a rider sits on over the back of a mount of
// kind k, at (x, y): as wide as the back.
func saddleCloth(dst drawTarget, k sim.CreatureKind, x, y float64) {
	w := 7.0
	switch k {
	case sim.CreatureAnkylo:
		w = 9
	case sim.CreatureRaptor:
		w = 5.5
	}
	for row := 0.0; row < 4; row++ {
		r := w - row*0.5
		fillRect(dst, float32(math.Round(x-r)), float32(math.Round(y+row-1)), float32(math.Round(2*r)), 1, saddleRamp.at(3-int(row)), false)
	}
	fillRect(dst, float32(math.Round(x-w)), float32(math.Round(y+3)), float32(math.Round(2*w)), 1, saddleRamp[0], false)
	// Tassels at the corners.
	for _, s := range []float64{-1, 1} {
		fillRect(dst, float32(math.Round(x+s*w-0.5)), float32(math.Round(y+4)), 1, 2, saddleRamp[1], false)
	}
}

// dismounting is the frame of moDismount p is on if they got off a mount a
// moment ago, and how far from where they now stand their feet are drawn:
// from the seat, down to the ground.
func (g *Game) dismounting(p protocol.PlayerView) (fr int, dx, dy float64, ok bool) {
	st := g.moveState(p.ID)
	age := float64(g.frame - st.dismountAt)
	if st.dismountAt == never || age < 0 || age >= moDismount.span() {
		return 0, 0, 0, false
	}
	fr = moDismount.frameAt(age)
	k := [5]float64{1, 0.9, 0.55, 0.2, 0}[fr]
	return fr, st.mountX * k, st.mountY * k, true
}

// rideReels are the motion sheets' scenes of riding: getting onto a
// raptor, a three-horn and an Ankylosaurus, riding each standing and on the
// move, and getting off.
func (g *Game) rideReels() []reel {
	var reels []reel
	names := map[sim.CreatureKind]string{sim.CreatureRaptor: "raptor", sim.CreatureTrike: "three-horn", sim.CreatureAnkylo: "ankylosaurus"}
	for _, k := range []sim.CreatureKind{sim.CreatureRaptor, sim.CreatureTrike, sim.CreatureAnkylo} {
		k := k
		const n = 300
		mount := protocol.CreatureView{ID: 1<<30 + 55, Kind: k, Tamed: true, Ridden: true, Owner: sheetID, State: sim.CreatureIdle}
		reels = append(reels, reel{name: "ride " + names[k], n: n, dirs: []sim.Dir{sim.DirRight, sim.DirLeft},
			at: func(g *Game, t int, p *protocol.PlayerView, d *drawable) {
				st := g.moveState(p.ID)
				p.Mount = mount.ID
				switch {
				case t < 40: // getting on
					st.mountAt = g.frame - t
				case t < 250:
					st.mountAt = never
					d.moving = t >= 90
				default: // and off
					p.Mount = 0
					if t == 250 {
						back, height := seat(k)
						side := facingSide(p.Dir)
						st.dismountAt = g.frame
						st.mountX, st.mountY = -back*side-6*side, -height+riderHip
					}
				}
				if p.Mount == 0 {
					d.sx += 6 * facingSide(p.Dir) // stood down beside it
				}
			},
			draw: func(g *Game, dst drawTarget, p protocol.PlayerView, d drawable) {
				c := mount
				c.FaceLeft = p.Dir == sim.DirLeft
				c.Ridden = p.Mount != 0
				md := d
				if p.Mount == 0 {
					md.sx -= 6 * facingSide(p.Dir)
				}
				if c.Ridden {
					g.drawRiderBehind(dst, p, c, md)
				}
				g.drawCreature(dst, c, md)
				if c.Ridden {
					g.drawRiderHD(dst, p, c, md)
				} else {
					g.drawPlayer(dst, p, d)
				}
			}})
	}
	return reels
}
