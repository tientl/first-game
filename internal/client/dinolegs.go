package client

import (
	"image/color"
	"math"
	"sync"
)

// Jointed legs. A leg whose rig gives its joints (rigLimb.j: hip, knee,
// ankle, then the ball of the foot for a toe-walker, then the tip of the
// toes) is cut into segments, thigh, shin, (metatarsus,) foot, each the
// leg's pixels nearest its bone. A pose places the tip of the foot: on the
// ground through the stance, moving back at a steady speed under the body,
// then lifted and swung forward. The foot and metatarsus turn about the
// toes (the heel rising at push-off, the ankle folding through the swing),
// and the knee is where the thigh and shin must meet between hip and ankle.
// Each segment's pixels are then turned about its joint, the neighbours
// overlapping around the joint so it never opens.
//
// The body rides on the legs: it sinks as much as the reaching leg needs at
// each footfall and rises as the leg passes under it.

func turnPt(v rigPt, a float64) rigPt {
	s, c := math.Sincos(a)
	return rigPt{v.x*c - v.y*s, v.x*s + v.y*c}
}

func (a rigPt) sub(b rigPt) rigPt { return rigPt{a.x - b.x, a.y - b.y} }
func (a rigPt) add(b rigPt) rigPt { return rigPt{a.x + b.x, a.y + b.y} }
func (a rigPt) len() float64      { return math.Hypot(a.x, a.y) }

// segDist is how far p is from the bone a-b.
func segDist(p, a, b rigPt) float64 {
	d := b.sub(a)
	l2 := d.x*d.x + d.y*d.y
	t := 0.0
	if l2 > 0 {
		t = min(1, max(0, ((p.x-a.x)*d.x+(p.y-a.y)*d.y)/l2))
	}
	return math.Hypot(p.x-a.x-t*d.x, p.y-a.y-t*d.y)
}

func ease(t float64) float64 {
	t = min(1, max(0, t))
	return t * t * (3 - 2*t)
}

// jointed reports whether every leg of the rig has its joints measured.
func (r dinoRig) jointed() bool {
	for _, l := range r.legs {
		if len(l.j) < 4 {
			return false
		}
	}
	return len(r.legs) > 0
}

// legCut is a leg lifted off the working canvas: its pixels, which segment
// each belongs to, and its joints as drawn, in canvas pixels.
type legCut struct {
	P              []rigPt
	fore, far      bool
	inside         bool // the thigh above top is inside the body: it shows only past the body's edge
	top            int  // above this row the leg is the body's flank too
	bx, by, bw, bh int
	px             []color.RGBA
	seg            []int8    // -1: not the leg's
	rad            []float64 // how far each segment reaches from its joint
	jr             []float64 // how far around each joint the neighbours overlap
	l0, l1         float64
	own            *legOwner // its pixels as painted (dinolegmask.go), or nil
}

func cutLeg(w *canvas, l rigLimb, pad, k int, far bool, own *legOwner) legCut {
	kf := float64(k)
	c := legCut{fore: l.fore, far: far, inside: (far || l.r == 0) && !own.whole(), top: l.top*k + pad, own: own}
	for _, q := range l.j {
		c.P = append(c.P, rigPt{q.x*kf + float64(pad), q.y*kf + float64(pad)})
	}
	n := len(c.P) - 1
	x0, x1 := l.x0*k+pad, (l.x1+1)*k-1+pad
	if own != nil {
		x0, x1 = own.x0, own.x1 // the leg as painted, not its columns (dinolegmask.go)
	}
	zr := l.r * kf
	if zr == 0 {
		zr = c.P[1].sub(c.P[0]).len()/2 + kf
	}
	c.bx = max(0, min(x0, int(c.P[0].x-zr)))
	c.by = max(0, min(c.top, int(c.P[0].y-zr)))
	if own.whole() {
		c.by = min(c.by, own.y0) // the thigh up to the hip (dinolegthigh.go)
	}
	c.bw = min(w.w, max(x1+1, int(c.P[0].x+zr)+1)) - c.bx
	c.bh = w.h - c.by
	c.px = make([]color.RGBA, c.bw*c.bh)
	c.seg = make([]int8, c.bw*c.bh)
	c.rad = make([]float64, n)
	c.jr = make([]float64, n+1)
	for y := 0; y < c.bh; y++ {
		for x := 0; x < c.bw; x++ {
			i := y*c.bw + x
			c.seg[i] = -1
			q, mine := own.pick(w.px[(y+c.by)*w.w+x+c.bx], x+c.bx, y+c.by) // dinolegmask.go
			if q.A < 128 {
				continue
			}
			p := rigPt{float64(x+c.bx) + 0.5, float64(y+c.by) + 0.5}
			s := -1
			if own.whole() {
				if !mine {
					continue
				}
				best := math.Inf(1)
				for j := 0; j < n; j++ {
					if d := segDist(p, c.P[j], c.P[j+1]); d < best {
						best, s = d, j
					}
				}
				if own.thigh(x+c.bx, y+c.by) {
					s = 0
				}
			} else if y+c.by >= c.top {
				if x+c.bx < x0 || x+c.bx > x1 || !mine {
					continue
				}
				best := math.Inf(1)
				for j := 0; j < n; j++ {
					if d := segDist(p, c.P[j], c.P[j+1]); d < best {
						best, s = d, j
					}
				}
			} else if segDist(p, c.P[0], c.P[1]) <= zr && mine {
				s = 0
			} else {
				continue
			}
			c.px[i], c.seg[i] = q, int8(s)
			c.rad[s] = max(c.rad[s], p.sub(c.P[s]).len())
		}
	}
	// Around a joint the neighbours overlap as far as the leg is wide there.
	for y := 1; y < c.bh; y++ {
		for x := 1; x < c.bw; x++ {
			a := c.seg[y*c.bw+x]
			for _, b := range [2]int8{c.seg[y*c.bw+x-1], c.seg[(y-1)*c.bw+x]} {
				if a < 0 || b < 0 || a == b {
					continue
				}
				j := max(a, b)
				d := rigPt{float64(x+c.bx) + 0.5, float64(y+c.by) + 0.5}.sub(c.P[j]).len()
				c.jr[j] = max(c.jr[j], d)
			}
		}
	}
	for j := 1; j < n; j++ {
		c.jr[j] = c.jr[j]*1.2 + kf
		c.rad[j-1] = max(c.rad[j-1], c.P[j].sub(c.P[j-1]).len()+c.jr[j])
		c.rad[j] = max(c.rad[j], c.jr[j])
	}
	c.l0, c.l1 = c.P[1].sub(c.P[0]).len(), c.P[2].sub(c.P[1]).len()
	return c
}

// footPose is where a foot is in a pose: its tip dx forward of where it is
// drawn and lift off the ground, the metatarsus turned dm and the toes dt
// from the drawing (+ raises the heel).
type footPose struct {
	dx, lift, dm, dt float64
	stance, heel     bool
	u                float64 // how far through the stance
}

// foot places the joints from the ankle down, the body as drawn.
func (c *legCut) foot(f footPose) []rigPt {
	n := len(c.P) - 1
	J := make([]rigPt, n+1)
	J[n] = c.P[n].add(rigPt{f.dx, -f.lift})
	J[n-1] = J[n].add(turnPt(c.P[n-1].sub(c.P[n]), f.dt))
	if n == 4 {
		J[2] = J[3].add(turnPt(c.P[2].sub(c.P[3]), f.dm))
	}
	return J
}

// reach is the longest the thigh and shin stretch: never quite straight.
func (c *legCut) reach() float64 { return 0.985 * (c.l0 + c.l1) }

// need is how far the body must sink for the leg to reach foot pose f.
func (c *legCut) need(f footPose) float64 {
	a := c.foot(f)[2]
	dx := a.x - c.P[0].x
	if l := c.reach(); math.Abs(dx) < l {
		return (a.y - c.P[0].y) - math.Sqrt(l*l-dx*dx)
	}
	return math.Inf(1)
}

// solve places every joint for foot pose f with the body sunk by drop, and
// gives how far each segment has turned from the drawing.
func (c *legCut) solve(f footPose, drop float64) (J []rigPt, th []float64) {
	n := len(c.P) - 1
	hip := c.P[0].add(rigPt{0, drop})
	lmax := c.reach()
	J = c.foot(f)
	for i := 0; i < 16 && f.heel && J[2].sub(hip).len() > lmax; i++ {
		f.dt += 0.05 // a leg left behind rises onto its toes
		f.dm += 0.03
		J = c.foot(f)
	}
	v := J[2].sub(hip)
	d := v.len()
	if lo := math.Abs(c.l0-c.l1) + 0.5; d > lmax || d < lo {
		// Out of reach still: the foot comes to the leg.
		t := min(lmax, max(lo, d)) / max(d, 1e-6)
		mv := rigPt{v.x * (t - 1), v.y * (t - 1)}
		for i := 2; i <= n; i++ {
			J[i] = J[i].add(mv)
		}
		v, d = J[2].sub(hip), d*t
	}
	J[0] = hip
	al := math.Acos(min(1, max(-1, (c.l0*c.l0+d*d-c.l1*c.l1)/(2*c.l0*d))))
	base := math.Atan2(v.y, v.x)
	if !c.fore {
		al = -al // a knee bends forward, an elbow back
	}
	J[1] = hip.add(rigPt{c.l0 * math.Cos(base+al), c.l0 * math.Sin(base+al)})
	th = make([]float64, n)
	for i := 0; i < n; i++ {
		a, b := J[i+1].sub(J[i]), c.P[i+1].sub(c.P[i])
		th[i] = math.Atan2(a.y, a.x) - math.Atan2(b.y, b.x)
	}
	if n == 4 {
		th[2] = f.dm
	}
	th[n-1] = f.dt
	return J, th
}

// legWork is the room the legs are drawn in, kept from one frame to the
// next: the layer of the leg being drawn, the far legs' own layer, and where
// a thigh was lifted off the flank. All but hole are left clear.
type legScratch struct {
	px, back []color.RGBA
	pr       []uint8
	hole     []bool
}

var (
	legWork legScratch
	legMu   sync.Mutex
)

// under marks a pixel of a leg's layer that goes behind the body.
const under = 0x80

// legLayer is where a leg is drawn before it goes on the canvas.
type legLayer struct {
	w, h           int
	px             []color.RGBA
	pr             []uint8
	x0, y0, x1, y1 int
}

// draw turns the segments of c to joints J and paints them on the layer:
// the overlap around each joint underneath, then the segments from the
// foot up.
func (c *legCut) draw(ly *legLayer, J []rigPt, th []float64) {
	n := len(J) - 1
	for i := 0; i < n; i++ {
		sn, cs := math.Sincos(-th[i])
		r := c.rad[i] + 2
		x0, x1 := max(0, int(J[i].x-r)), min(ly.w, int(J[i].x+r)+1)
		y0, y1 := max(0, int(J[i].y-r)), min(ly.h, int(J[i].y+r)+1)
		ly.x0, ly.y0, ly.x1, ly.y1 = min(ly.x0, x0), min(ly.y0, y0), max(ly.x1, x1), max(ly.y1, y1)
		own := uint8(2 + n - i)
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				dx, dy := float64(x)+0.5-J[i].x, float64(y)+0.5-J[i].y
				if dx*dx+dy*dy > r*r {
					continue
				}
				fx, fy := c.P[i].x+dx*cs-dy*sn, c.P[i].y+dx*sn+dy*cs
				if i == 0 && c.own.whole() {
					fx, fy = c.bendBack(dx, dy, th[0], fx, fy) // dinolegthigh.go
				}
				sx, sy := int(math.Floor(fx))-c.bx, int(math.Floor(fy))-c.by
				if sx < 0 || sy < 0 || sx >= c.bw || sy >= c.bh {
					continue
				}
				sg := int(c.seg[sy*c.bw+sx])
				pr := own
				switch {
				case sg < 0:
					continue
				case sg == i:
				case sg == i-1 && math.Hypot(fx-c.P[i].x, fy-c.P[i].y) <= c.jr[i]:
					pr = 1
				case sg == i+1 && math.Hypot(fx-c.P[i+1].x, fy-c.P[i+1].y) <= c.jr[i+1]:
					pr = 1
				default:
					continue
				}
				if o := y*ly.w + x; pr > ly.pr[o]&^under {
					if c.inside && sy+c.by < c.top {
						pr |= under
					}
					ly.px[o], ly.pr[o] = c.px[sy*c.bw+sx], pr
				}
			}
		}
	}
}

type legKey struct {
	file string
	k    int
}

// legCuts keeps the legs cut from each drawing.
var legCuts = map[legKey][]legCut{}

// gaitTiming is how rig r strides at amp (1: a walk, 1.25 and over: a run):
// how much of a run it is, the share of a stride each foot is on the ground,
// and how high a foot lifts, as a share of the leg's height.
func gaitTiming(r dinoRig, amp float64) (run, duty, liftK float64) {
	run = min(1, max(0, (math.Abs(amp)-1)/0.25))
	duty, liftK = 0.62-0.1*run, 0.09+0.03*run
	if r.runner {
		duty, liftK = 0.62-0.24*run, 0.09+0.07*run // a flight phase between footfalls
	}
	if len(r.legs) > 2 {
		duty, liftK = 0.7-0.1*run, 0.05+0.03*run
	}
	return run, duty, liftK
}

// legPoses is where pose p puts each foot of rig r, and how far the body
// sinks on its legs, in canvas pixels (kf to a pixel of the drawing).
func legPoses(r dinoRig, p rigPose, cuts []legCut, kf float64) ([]footPose, float64) {
	n := len(cuts)
	feet := make([]footPose, n)
	drop := 0.0
	height := cuts[0].P[len(cuts[0].P)-1].y - cuts[0].P[0].y
	const follow = 0.85 // how much of the leg's lean the metatarsus takes up
	if p.cycle == 0 {
		for i := range feet {
			dx := p.step * r.stride * kf
			if i%2 == 1 {
				dx = -dx
			}
			h := cuts[i].P[len(cuts[i].P)-1].y - cuts[i].P[0].y
			feet[i] = footPose{dx: dx, dm: -follow * math.Atan2(dx, h), stance: true, heel: dx < 0}
			drop = max(drop, cuts[i].need(feet[i]))
		}
	} else {
		run, duty, liftK := gaitTiming(r, p.step)
		S := math.Abs(p.step) * r.stride * gaitReach * kf
		sw := 1 - duty
		squeeze := 0.0
		for i := range feet {
			c := &cuts[i]
			g := float64(i) * 0.5
			if i < len(r.gait) {
				g = r.gait[i]
			}
			ph := p.cycle + g
			ph -= math.Floor(ph)
			nn := len(c.P) - 1
			h := c.P[nn].y - c.P[0].y
			lean := follow * math.Atan2(S, h)
			push, curl := 0.3, 0.1 // the heel's rise at push-off, the toes' droop through the swing
			if nn == 3 {
				push, curl = 0.25, 0.15
			}
			contact := footPose{dx: S, dm: -lean, stance: true}
			f := footPose{}
			if v := ph + sw/2 - math.Floor(ph+sw/2); v < sw {
				// The swing: off the toes, folded up under the body, then reaching.
				v /= sw
				f.dx = -S * math.Cos(math.Pi*v)
				f.lift = (h*liftK + 0.1*S) * math.Pow(math.Sin(math.Pi*v), 0.8)
				fold := -(0.1 + 0.2*run)
				if v < 0.5 {
					f.dm = lean + push + (fold-lean-push)*ease(v/0.5)
				} else {
					f.dm = fold + (-lean-fold)*ease((v-0.5)/0.5)
				}
				f.dt = push*(1-ease(v/0.55)) + curl*math.Sin(math.Pi*v)*(1-v)
			} else {
				u := (ph - sw/2) / duty
				dx := S * (1 - 2*u)
				e := ease((u - 0.72) / 0.28)
				f = footPose{dx: dx, dm: -follow*math.Atan2(dx, h) + push*e, dt: push * e,
					stance: true, heel: dx < 0, u: u}
				if dx >= 0 {
					drop = max(drop, c.need(f))
				}
				squeeze = max(squeeze, math.Sin(math.Pi*u))
			}
			feet[i] = f
			// The body is lowest as a foot comes down, and rises between.
			d := math.Abs(ph - sw/2)
			d = min(d, 1-d)
			bump := 0.5 + 0.5*math.Cos(math.Pi*min(1, d/0.25))
			if r.runner {
				bump += (1 - bump) * run
			}
			if nd := c.need(contact); !math.IsInf(nd, 1) {
				drop = max(drop, nd*bump)
			}
		}
		drop += 0.05 * height * run * squeeze // a runner sinks onto the leg that carries it
	}
	drop = math.Round(min(max(drop, 0), 0.22*height))
	return feet, drop
}

// stepJoints is stepLegs for a rig with jointed legs; it also gives which
// pixels of w the legs drew, which nothing turning after takes with it.
func stepJoints(w *canvas, r dinoRig, p rigPose, pad, k int) (float64, []bool) {
	kf := float64(k)
	n := len(r.legs)
	// The legs are cut from the drawing as it stands, the same every time.
	key := legKey{r.file, k}
	dinoMu.Lock()
	cuts, ok := legCuts[key]
	dinoMu.Unlock()
	if !ok {
		cuts = make([]legCut, n)
		for i, l := range r.legs {
			cuts[i] = cutLeg(w, l, pad, k, r.far>>uint(i)&1 == 1, legOwnerOf(r.file, i, pad, k))
		}
		dinoMu.Lock()
		legCuts[key] = cuts
		dinoMu.Unlock()
	}
	feet, drop := legPoses(r, p, cuts, kf)

	// Lift the legs off the canvas, lower the body, and put them back.
	legMu.Lock()
	defer legMu.Unlock()
	if len(legWork.pr) < len(w.px) {
		legWork = legScratch{make([]color.RGBA, len(w.px)), make([]color.RGBA, len(w.px)), make([]uint8, len(w.px)), make([]bool, len(w.px))}
	}
	hole := legWork.hole[:len(w.px)]
	clear(hole)
	for i := range cuts {
		c := &cuts[i]
		for y := 0; y < c.bh; y++ {
			for x := 0; x < c.bw; x++ {
				if c.seg[y*c.bw+x] < 0 || (c.inside && y+c.by < c.top) {
					continue
				}
				if X, Y := x+c.bx, y+c.by; c.own.whole() {
					if c.own.owns(X, Y) {
						w.px[Y*w.w+X] = c.own.bodyAt(X, Y) // the body painted under it (dinolegthigh.go)
					}
					continue
				}
				w.px[(y+c.by)*w.w+x+c.bx] = color.RGBA{}
			}
		}
	}
	if drop > 0 {
		raise(w, -int(drop))
	}
	for i := range cuts {
		c := &cuts[i]
		if c.inside || c.own.whole() {
			continue
		}
		for y := 0; y < min(c.bh, c.top-c.by); y++ {
			for x := 0; x < c.bw; x++ {
				if yy := y + c.by + int(drop); c.seg[y*c.bw+x] >= 0 && yy < w.h {
					hole[yy*w.w+x+c.bx] = true
				}
			}
		}
	}
	leg := make([]bool, len(w.px)) // what the legs drew
	ly := &legLayer{w: w.w, h: w.h, px: legWork.px, pr: legWork.pr}
	back := legWork.back // the far legs, one over another, to go behind the body
	bx0, by0, bx1, by1 := w.w, w.h, 0, 0
	// Where each leg was drawn: turned, it reaches past where it was cut.
	drawn := make([][4]int, len(cuts))
	for pass := range 2 { // the far legs, then the near ones over them
		for i := range cuts {
			c := &cuts[i]
			if c.far != (pass == 0) {
				continue
			}
			ly.x0, ly.y0, ly.x1, ly.y1 = w.w, w.h, 0, 0
			J, th := c.solve(feet[i], drop)
			c.draw(ly, J, th)
			drawn[i] = [4]int{ly.x0, ly.y0, ly.x1, ly.y1}
			ly.trimJoints(k)
			to := w.px
			if c.far {
				to = back
				bx0, by0, bx1, by1 = min(bx0, ly.x0), min(by0, ly.y0), max(bx1, ly.x1), max(by1, ly.y1)
			}
			for y := ly.y0; y < ly.y1; y++ {
				for x := ly.x0; x < ly.x1; x++ {
					if o := y*w.w + x; ly.pr[o] != 0 {
						if (c.far || ly.pr[o]&under == 0 || to[o].A < 128) && (c.far || !c.own.whole() || !c.own.front(x, y, int(drop))) {
							to[o] = ly.px[o]
							leg[o] = true
						}
						ly.pr[o] = 0
					}
				}
			}
		}
		if pass == 0 {
			for y := by0; y < by1; y++ {
				for x := bx0; x < bx1; x++ {
					o := y*w.w + x
					if back[o].A >= 128 && w.px[o].A < 128 && !hole[o] {
						w.px[o] = back[o]
						leg[o] = true
					}
					back[o] = color.RGBA{}
				}
			}
		}
	}
	type fill struct {
		o int
		q color.RGBA
	}
	var fills []fill
	solid := func(x, y int) bool { return x >= 0 && y >= 0 && x < w.w && y < w.h && w.px[y*w.w+x].A >= 128 }
	near := func(x, y, dx, dy, n int) bool {
		for i := 1; i <= n; i++ {
			if solid(x+dx*i, y+dy*i) {
				return true
			}
		}
		return false
	}
	// Where a thigh has swung off the flank, the flank shows: the thigh's
	// own middle, drawn in. Not at the body's edge, which goes with the leg.
	for i := range cuts {
		c := &cuts[i]
		if c.inside || c.own.whole() {
			continue
		}
		for y := 0; y < min(c.bh, c.top-c.by); y++ {
			for x := 0; x < c.bw; x++ {
				X, Y := x+c.bx, y+c.by+int(drop)
				if c.seg[y*c.bw+x] < 0 || Y >= w.h || w.px[Y*w.w+X].A >= 128 {
					continue
				}
				if !near(X, Y, -1, 0, 5*k) || !near(X, Y, 1, 0, 5*k) {
					continue
				}
				sx := int(c.P[0].x + (float64(X)+0.5-c.P[0].x)*0.5 - float64(c.bx))
				sy := int(c.P[0].y + (float64(y+c.by)+0.5-c.P[0].y)*0.5 - float64(c.by))
				q := c.px[y*c.bw+x]
				if sx >= 0 && sy >= 0 && sx < c.bw && sy < c.bh && c.seg[sy*c.bw+sx] >= 0 {
					q = c.px[sy*c.bw+sx]
				}
				fills = append(fills, fill{Y*w.w + X, q})
			}
		}
	}
	for _, f := range fills {
		w.px[f.o] = f.q
	}
	fills = fills[:0]
	// A crack left between a thigh and the body or the leg beside it closes,
	// in shadow.
	for i := range cuts {
		c := &cuts[i]
		if c.own.whole() {
			continue // what is behind it is painted (dinolegthigh.go)
		}
		for y := 0; y < c.bh; y++ {
			for x := 0; x < c.bw; x++ {
				X, Y := x+c.bx, y+c.by+int(drop)
				if c.seg[y*c.bw+x] != 0 || Y >= w.h || w.px[Y*w.w+X].A >= 128 {
					continue
				}
				if near(X, Y, -1, 0, 2*k) && near(X, Y, 1, 0, 2*k) && near(X, Y, 0, -1, 3*k) {
					q := c.px[y*c.bw+x]
					fills = append(fills, fill{Y*w.w + X, color.RGBA{uint8(int(q.R) * 2 / 3), uint8(int(q.G) * 2 / 3), uint8(int(q.B) * 2 / 3), q.A}})
				}
			}
		}
	}
	for _, f := range fills {
		w.px[f.o] = f.q
	}
	// Where a leg inside the body was cut away under it, the flank ends
	// cleanly, and no specks are left about the legs.
	for i := range cuts {
		if c := &cuts[i]; c.inside {
			tidyCut(w, leg, c.bx, c.bx+c.bw, c.top+int(drop), k)
		}
	}
	for i := range cuts {
		c := &cuts[i]
		m := 0
		if c.own.whole() {
			// What was left of the body along its edge goes too (dinolegthigh.go).
			c.trimEdge(w, leg, int(drop), k)
			m = 2 * k
		}
		despeck(w, c.bx-m, c.by-m, c.bx+c.bw+m, min(w.h, c.by+c.bh+int(drop)), 3*k)
		if d := drawn[i]; c.own.whole() {
			despeck(w, d[0]-m, d[1], d[2]+m, min(w.h, d[3]+m), 3*k)
		}
	}
	return drop, leg
}

// tidyCut cleans the flank of canvas w where a leg inside the body was cut
// away from row top down (the body left behind, the leg posed elsewhere):
// in the band of rows about the cut, columns x0-x1, the body's slivers one
// pixel wide go, and its pixels left over nothing are shaded as its edge.
// leg marks the pixels the legs drew, which are left alone.
func tidyCut(w *canvas, leg []bool, x0, x1, top, k int) {
	y0, y1 := max(1, top-3*k), min(w.h-1, top+k)
	x0, x1 = max(1, x0), min(w.w-1, x1)
	solid := func(x, y int) bool { return w.px[y*w.w+x].A >= 128 }
	body := func(x, y int) bool { return solid(x, y) && !leg[y*w.w+x] }
	for range 2 {
		var gone []int
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				if body(x, y) && !solid(x-1, y) && !solid(x+1, y) && !solid(x, y+1) {
					gone = append(gone, y*w.w+x) // a sliver hanging down
				}
			}
		}
		for _, o := range gone {
			w.px[o] = color.RGBA{}
		}
	}
	var edge []int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if body(x, y) && !solid(x, y+1) {
				edge = append(edge, y*w.w+x)
			}
		}
	}
	for _, o := range edge {
		q := w.px[o]
		w.px[o] = color.RGBA{uint8(int(q.R) * 3 / 5), uint8(int(q.G) * 3 / 5), uint8(int(q.B) * 3 / 5), q.A}
	}
}

// despeck clears, in the box x0-x1, y0-y1 of canvas w, the bits of solid
// pixels smaller than n that touch nothing else (8-connected) and stay
// clear of the box's edge: loose specks of a turned leg.
func despeck(w *canvas, x0, y0, x1, y1, n int) {
	x0, y0, x1, y1 = max(0, x0), max(0, y0), min(w.w, x1), min(w.h, y1)
	if x1 <= x0 || y1 <= y0 {
		return
	}
	bw := x1 - x0
	seen := make([]bool, bw*(y1-y0))
	var stack, comp []int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			s := (y-y0)*bw + x - x0
			if seen[s] || w.px[y*w.w+x].A < 128 {
				continue
			}
			comp, stack = comp[:0], append(stack[:0], s)
			seen[s] = true
			edge := false
			for len(stack) > 0 { // all of it: a piece left half seen would come back as specks
				q := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				comp = append(comp, q)
				qx, qy := q%bw+x0, q/bw+y0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						X, Y := qx+dx, qy+dy
						if X < x0 || Y < y0 || X >= x1 || Y >= y1 {
							if X >= 0 && Y >= 0 && X < w.w && Y < w.h && w.px[Y*w.w+X].A >= 128 {
								edge = true // it goes on past the box
							}
							continue
						}
						if t := (Y-y0)*bw + X - x0; !seen[t] && w.px[Y*w.w+X].A >= 128 {
							seen[t] = true
							stack = append(stack, t)
						}
					}
				}
			}
			if len(comp) > n || edge {
				continue
			}
			for _, q := range comp {
				w.px[(q/bw+y0)*w.w+q%bw+x0] = color.RGBA{}
			}
		}
	}
}

// trimJoints keeps, of the pixels a leg's segments lent each other around a
// joint (priority 1), only those that close a crack: with the leg's own
// pixels on both sides of them, across or along, within a few pixels. Lent
// pixels anywhere else are the neighbour's flesh turned the wrong way, and
// would stick out of the leg as specks and slivers when a joint bends far.
func (ly *legLayer) trimJoints(k int) {
	reach := 3 * k
	own := func(x, y int) bool {
		if x < ly.x0 || y < ly.y0 || x >= ly.x1 || y >= ly.y1 {
			return false
		}
		return ly.pr[y*ly.w+x]&^under > 1
	}
	hit := func(x, y, dx, dy int) bool {
		for i := 1; i <= reach; i++ {
			if own(x+dx*i, y+dy*i) {
				return true
			}
		}
		return false
	}
	var drop []int
	for y := ly.y0; y < ly.y1; y++ {
		for x := ly.x0; x < ly.x1; x++ {
			o := y*ly.w + x
			if ly.pr[o]&^under != 1 {
				continue
			}
			if (hit(x, y, -1, 0) && hit(x, y, 1, 0)) || (hit(x, y, 0, -1) && hit(x, y, 0, 1)) ||
				(hit(x, y, -1, -1) && hit(x, y, 1, 1)) || (hit(x, y, -1, 1) && hit(x, y, 1, -1)) {
				continue
			}
			drop = append(drop, o)
		}
	}
	for _, o := range drop {
		ly.pr[o], ly.px[o] = 0, color.RGBA{}
	}
}
