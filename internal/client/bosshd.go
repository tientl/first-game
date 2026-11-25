package client

import (
	"image/color"
	"math"
	"runtime"
	"sync"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The kit of the bosses and sea monsters drawn at HDDensity (Titanoboa,
// Megalodon, the Mosasaurus and the giant squid): animations whose frames
// are drawn the first time they are shown, a tube painter for long bodies,
// the water they break, and a clock that says how long a boss has been in
// its state. Everything is laid out in art pixels on canvases three times as
// fine (newScaledCanvas(w, h, HDDensity)).

// --- animations -----------------------------------------------------------

// bhAnim is a run of frames drawn on demand and kept. draw must not touch
// anything but its canvas: frames are also drawn ahead on other threads.
type bhAnim struct {
	draw func(i int) (c *canvas, ax, ay float64) // the anchor in art pixels
	fr   []bhFrame
	warm sync.Once
}

type bhFrame struct {
	once   sync.Once
	c      *canvas
	ax, ay float64
	s      sprite
	made   bool
}

func newBHAnim(n int, draw func(i int) (*canvas, float64, float64)) *bhAnim {
	return &bhAnim{draw: draw, fr: make([]bhFrame, n)}
}

func (a *bhAnim) n() int { return len(a.fr) }

func (a *bhAnim) canvas(i int) *bhFrame {
	f := &a.fr[i]
	f.once.Do(func() { f.c, f.ax, f.ay = a.draw(i) })
	return f
}

// sprite is frame i (wrapped round), mirrored if left. Call it from the
// game's thread only.
func (a *bhAnim) sprite(i int, left bool) sprite {
	n := len(a.fr)
	i = ((i % n) + n) % n
	f := a.canvas(i)
	if !f.made {
		f.s, f.made = spriteHD(f.c.image(), f.ax, f.ay), true
		f.c = nil
	}
	if left {
		return flipped(f.s)
	}
	return f.s
}

// at is the frame at fraction u (0..1) of a run played once.
func (a *bhAnim) at(u float64, left bool) sprite {
	return a.sprite(min(len(a.fr)-1, int(clamp01(u)*float64(len(a.fr)))), left)
}

// hdWarm draws the frames of the animations ahead, off the game's thread,
// so that a fight does not stutter the first time each frame comes up.
func hdWarm(first *bhAnim, as ...*bhAnim) {
	first.warm.Do(func() {
		all := append([]*bhAnim{first}, as...)
		type job struct {
			a *bhAnim
			i int
		}
		jobs := make(chan job, 256)
		for range max(1, min(4, runtime.NumCPU()-1)) {
			go func() {
				for j := range jobs {
					j.a.canvas(j.i)
				}
			}()
		}
		go func() {
			for _, a := range all {
				for i := range a.fr {
					jobs <- job{a, i}
				}
			}
			close(jobs)
		}()
	})
}

// hdCrop cuts a canvas drawn at HDDensity down to what is on it (on whole
// art pixels) and gives the anchor in art pixels.
func hdCrop(c *canvas, ax, ay float64) (*canvas, float64, float64) {
	const k = HDDensity
	x0, y0, x1, y1 := c.w, c.h, 0, 0
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.px[y*c.w+x].A > 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x+1), max(y1, y+1)
			}
		}
	}
	if x1 <= x0 {
		return newCanvas(k, k), 0, 0
	}
	x0, y0 = max(0, x0/k-1)*k, max(0, y0/k-1)*k
	x1, y1 = min(c.w, (x1/k+2)*k), min(c.h, (y1/k+2)*k)
	out := newCanvas(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		copy(out.px[(y-y0)*out.w:(y-y0+1)*out.w], c.px[y*c.w+x0:y*c.w+x1])
	}
	return out, ax - float64(x0)/k, ay - float64(y0)/k
}

func hdBossCanvas(w, h int) *canvas { return newScaledCanvas(w, h, HDDensity) }

// rq picks a tone of r for light l: clean bands, their borders a little
// broken up.
func rq(r ramp, l float64, x, y int) color.RGBA {
	return r.at(int(math.Round(clamp01(l)*float64(len(r)-1) + (hashf(x, y, 77)-0.5)*0.34)))
}

// blend lays col over what is at (x, y) by its alpha.
func (c *canvas) blend(x, y int, col color.RGBA) {
	if !c.in(x, y) || col.A == 0 {
		return
	}
	o := c.px[y*c.w+x]
	if o.A == 0 || col.A == 255 {
		c.put(x, y, col)
		return
	}
	a := float64(col.A) / 255
	m := mix(o, col, a)
	m.A = uint8(min(255, float64(o.A)+float64(col.A)*(1-float64(o.A)/255)))
	c.put(x, y, m)
}

// cutOut is a colour for polygon that clears what it covers.
func cutOut(c *canvas) func(x, y int) color.RGBA {
	return func(x, y int) color.RGBA {
		if c.in(x, y) {
			c.px[y*c.w+x] = color.RGBA{}
		}
		return color.RGBA{}
	}
}

// blendOver lays o over c, with its top-left at (dx, dy), by its alpha.
func (c *canvas) blendOver(o *canvas, dx, dy int) {
	for y := 0; y < o.h; y++ {
		for x := 0; x < o.w; x++ {
			if p := o.px[y*o.w+x]; p.A > 0 {
				c.blend(x+dx, y+dy, p)
			}
		}
	}
}

// dot fills a disc of radius r art pixels about (x, y), blending.
func (c *canvas) dot(x, y, r float64, col color.RGBA) {
	px, py := c.toPx(x, y)
	c.dotPx(px, py, r*c.k(), col)
}

// dotPx is dot in the canvas's own pixels.
func (c *canvas) dotPx(px, py, r float64, col color.RGBA) {
	for j := int(math.Floor(py - r)); j <= int(py+r); j++ {
		for i := int(math.Floor(px - r)); i <= int(px+r); i++ {
			dx, dy := float64(i)+0.5-px, float64(j)+0.5-py
			if dx*dx+dy*dy <= r*r {
				c.blend(i, j, col)
			}
		}
	}
}

// --- tubes ----------------------------------------------------------------

// tubePt is what the tube painter knows about one pixel of a body.
type tubePt struct {
	s      float64 // 0 at the tail's tip, 1 at the neck
	arc    float64 // the same in art pixels along the body
	t      float64 // across it: -1 its left edge (the far side of a body heading right), 1 its right
	nx, ny float64 // the body's unit normal there, to its right
	r      float64 // half its width there, in art pixels
}

// hdSpine fills in the normals of a spine and returns the length along it
// at each point, in the units it is laid out in.
func hdSpine(pts []spinePt) []float64 {
	arc := make([]float64, len(pts))
	for i := range pts {
		a, b := pts[max(0, i-1)], pts[min(len(pts)-1, i+1)]
		tx, ty := b.x-a.x, b.y-a.y
		if l := math.Hypot(tx, ty); l > 1e-9 {
			pts[i].nx, pts[i].ny = -ty/l, tx/l
		}
		if i > 0 {
			arc[i] = arc[i-1] + math.Hypot(pts[i].x-pts[i-1].x, pts[i].y-pts[i-1].y)
		}
	}
	return arc
}

// hdTube paints a body along pts (in art pixels, tail first, a point every
// third of a pixel or closer): a later stretch lies over an earlier one, and
// where it does by more than gap art pixels of body it shades what is under
// it. shade colours a pixel; a clear colour leaves it out.
func hdTube(c *canvas, pts []spinePt, gap float64, shade func(x, y int, q tubePt) color.RGBA) {
	arc := hdSpine(pts)
	k := c.k()
	sp := make([]spinePt, len(pts))
	for i, p := range pts {
		p.x, p.y = c.toPx(p.x, p.y)
		p.r *= k
		sp[i] = p
	}
	idx := make([]int32, c.w*c.h)
	for i := range idx {
		idx[i] = -1
	}
	for i := range sp {
		p := &sp[i]
		r2 := p.r * p.r
		for y := max(0, int(p.y-p.r)); y <= min(c.h-1, int(p.y+p.r)); y++ {
			dy := float64(y) + 0.5 - p.y
			for x := max(0, int(p.x-p.r)); x <= min(c.w-1, int(p.x+p.r)); x++ {
				if dx := float64(x) + 0.5 - p.x; dx*dx+dy*dy <= r2 {
					idx[y*c.w+x] = int32(i)
				}
			}
		}
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			i := idx[y*c.w+x]
			if i < 0 {
				continue
			}
			p := &sp[i]
			dx, dy := float64(x)+0.5-p.x, float64(y)+0.5-p.y
			along := (dx*p.ny - dy*p.nx) / k // ahead of the point, in art pixels
			q := tubePt{s: p.s, arc: arc[i] + along, t: (dx*p.nx + dy*p.ny) / p.r, nx: p.nx, ny: p.ny, r: p.r / k}
			if col := shade(x, y, q); col.A > 0 {
				c.put(x, y, col)
			} else {
				idx[y*c.w+x] = -1
			}
		}
	}
	// What lies under a later coil is in its shadow.
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			i := idx[y*c.w+x]
			if i < 0 {
				continue
			}
			for d := 1; d <= 5; d++ {
				dark := 0.0
				for _, o := range [4][2]int{{0, -d}, {-d, 0}, {d, 0}, {0, d}} {
					nx, ny := x+o[0], y+o[1]
					if !c.in(nx, ny) {
						continue
					}
					if j := idx[ny*c.w+nx]; j >= 0 && arc[j] > arc[i]+gap {
						dark = 0.6 - 0.1*float64(d)
						if o[0] > 0 || o[1] > 0 {
							dark *= 0.5 // the lit side of what is above
						}
						break
					}
				}
				if dark > 0 {
					c.put(x, y, mix(c.at(x, y), colInk, dark))
					break
				}
			}
		}
	}
}

// hdGroundShadow lays the shadow of the stretch of body from..to on the
// ground under it.
func hdGroundShadow(c *canvas, pts []spinePt, from, to float64) {
	n := len(pts)
	for i := int(from * float64(n-1)); i <= int(to*float64(n-1)); i += 3 {
		p := pts[i]
		cx, cy := c.toPx(p.x+1.5, p.y+p.r*0.62+1.2)
		rx, ry := p.r*1.05*c.k(), p.r*0.5*c.k()
		for y := int(cy - ry); y <= int(cy+ry); y++ {
			for x := int(cx - rx); x <= int(cx+rx); x++ {
				dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
				if d := dx*dx + dy*dy; d <= 1 && c.in(x, y) {
					a := uint8(86)
					if d > 0.62 {
						a = 46
					}
					if c.px[y*c.w+x].A < a {
						c.px[y*c.w+x] = color.RGBA{10, 14, 20, a}
					}
				}
			}
		}
	}
}

// --- frames with a scale ------------------------------------------------------

// hdFrameAt is a bossFrame at (x, y) art pixels on c, its units k art pixels.
func hdFrameAt(c *canvas, x, y, a, k float64) bossFrame {
	px, py := c.toPx(x, y)
	return bossFrame{ox: px, oy: py, a: a, sc: k * c.k()}
}

// tri fills a triangle given in the frame.
func (f bossFrame) tri(c *canvas, u0, v0, u1, v1, u2, v2 float64, col func(x, y int) color.RGBA) {
	f.poly(c, [][2]float64{{u0, v0}, {u1, v1}, {u2, v2}}, col)
}

// rawPoly fills a polygon of canvas pixels (c may be a scaled canvas).
func rawPoly(c *canvas, pts []float64, col func(x, y int) color.RGBA) {
	s, tx, ty := c.s, c.tx, c.ty
	c.s, c.tx, c.ty = 0, 0, 0
	c.polygon(pts, col)
	c.s, c.tx, c.ty = s, tx, ty
}

// hdEye sets an eye of radius r (frame units) at (u, v): a dark socket, an
// iris with a slit or round pupil, and a glint from the upper left.
func hdEye(c *canvas, f bossFrame, u, v, r float64, rim, iris, hot color.RGBA, slit bool) {
	cx, cy := f.at(u, v)
	pr := r * f.k1()
	for j := int(cy - pr - 2); j <= int(cy+pr+2); j++ {
		for i := int(cx - pr - 2); i <= int(cx+pr+2); i++ {
			dx, dy := float64(i)+0.5-cx, float64(j)+0.5-cy
			d := math.Hypot(dx, dy) / pr
			lu, _ := rotPt(-f.a, dx, dy)
			switch {
			case d > 1.28:
			case d > 1:
				c.put(i, j, rim)
			case slit && math.Abs(lu) < pr*0.2*(1.15-d*d) || !slit && d < 0.55:
				c.put(i, j, colInk)
			default:
				c.put(i, j, mix(hot, iris, clamp01(d*1.3-0.1+(dy/pr)*0.35)))
			}
		}
	}
	g := max(1, pr*0.3)
	for j := 0; j < int(math.Ceil(g)); j++ {
		for i := 0; i < int(math.Ceil(g)); i++ {
			c.put(int(cx-pr*0.5)+i, int(cy-pr*0.55)+j, colWhite)
		}
	}
}

// --- water ----------------------------------------------------------------

var (
	colSeaDeep = hex(0x0a1c30)
	colSpray   = hex(0xd4eef4)
)

// hdSink puts what is under the waterline wy (art pixels) under water: seen
// dimly through it for depth art pixels, then gone.
func hdSink(c *canvas, wy, depth float64) {
	_, py := c.toPx(0, wy)
	y0 := int(py)
	for y := y0; y < c.h; y++ {
		d := float64(y-y0) / (depth * c.k())
		for x := 0; x < c.w; x++ {
			p := &c.px[y*c.w+x]
			if p.A == 0 {
				continue
			}
			if d >= 1 || p.A < 255 {
				*p = color.RGBA{}
				continue
			}
			wob := math.Sin(float64(y)*0.9+float64(x)*0.13) * 0.08
			m := mix(*p, colSeaDeep, 0.7)
			m.A = uint8(150 * clamp01(1-d+wob) * clamp01(1-d+wob))
			*p = m
		}
	}
}

// hdFoam breaks the water along the waterline from x0 to x1: a ragged band
// of foam that boils with ph (0..1, looping).
func hdFoam(c *canvas, x0, x1, wy, thick, ph float64, seed uint32) {
	for x := x0; x <= x1; x += 0.34 {
		edge := clamp01(min(x-x0, x1-x) / 3)
		i := int(x * 3)
		w := 2 * math.Pi * (ph + hashf(i/5, 1, seed))
		h := thick * edge * (0.45 + 0.55*hashf(i/4, 2, seed)) * (0.75 + 0.25*math.Sin(w))
		if h < 0.2 {
			continue
		}
		col := colFoam
		if hashf(i/3, 3, seed) < 0.3 {
			col = colFoamDim
		}
		c.dot(x, wy-h*0.35, h*0.55, col)
		if hashf(i, 4, seed) < 0.3 {
			c.dot(x+0.3, wy+h*0.5+0.4, 0.4, withAlpha(colFoamDim, 200))
		}
	}
}

// hdRing is a ring of foam spreading on the water about (cx, wy): rx by ry
// art pixels, breaking up as fade goes to 1.
func hdRing(c *canvas, cx, wy, rx, ry, fade float64, seed uint32) {
	n := int(rx * 5)
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		if hashf(i/3, 0, seed) < fade*0.9 || math.Sin(a) < -0.35 {
			continue
		}
		col := withAlpha(colFoam, uint8(235*(1-fade*0.6)))
		if hashf(i/2, 1, seed) < 0.4 {
			col = withAlpha(colFoamDim, uint8(210*(1-fade*0.6)))
		}
		c.dot(cx+rx*math.Cos(a), wy+ry*math.Sin(a), 0.35+0.4*hashf(i/4, 2, seed)*(1-fade), col)
	}
}

// hdSplash throws water up from (cx, wy): n drops on their arcs at time t
// (0 the burst, 1 all fallen back), power art pixels high and spread wide,
// and a crown of spray about the burst while it is young.
func hdSplash(c *canvas, cx, wy, power, spread, t float64, n int, seed uint32) {
	if t <= 0 || t >= 1 {
		return
	}
	for i := 0; i < n; i++ {
		h0, h1, h2 := hashf(i, 0, seed), hashf(i, 1, seed), hashf(i, 2, seed)
		life := 0.55 + 0.45*h2
		u := t / life
		if u >= 1 {
			continue
		}
		vx := (h0 - 0.5) * 2 * spread
		top := power * (0.35 + 0.65*h1) * (1 - math.Abs(h0-0.5)*0.9)
		x := cx + vx*u
		y := wy - top*4*u*(1-u)
		r := (0.3 + 0.75*h2*h1) * (1 - u*0.5)
		col := colFoam
		switch {
		case i%4 == 0:
			col = colFoamDim
		case i%7 == 0:
			col = colSpray
		}
		c.dot(x, y, r, col)
		if r > 0.6 && u > 0.1 { // a tail behind a big drop
			c.dot(x-vx*0.05, y+top*0.2*(1-2*u)*0.3, r*0.6, withAlpha(colFoamDim, 200))
		}
	}
	if t < 0.6 { // the crown: a mound of spray billowing up about the burst
		k := math.Sin(t / 0.6 * math.Pi)
		for i := 0; i < int(spread*power*0.16); i++ {
			dx := (hashf(i, 5, seed) - 0.5) * 2
			env := power * 0.55 * k * (1 - dx*dx) * (0.75 + 0.25*math.Sin(dx*9+float64(seed)))
			up := hashf(i, 6, seed)
			x, y := cx+dx*spread*0.62, wy-env*up
			r := (0.7 + 1.5*hashf(i, 8, seed)) * (1 - up*0.55) * (0.5 + 0.5*k)
			c.dot(x+0.35, y+0.35, r, colFoamDim)
			col := colFoam
			if hashf(i, 9, seed) < 0.2 {
				col = colSpray
			}
			c.dot(x, y, r*0.86, col)
		}
	}
}

// hdSilhouette turns everything drawn on c into a dark shape under the
// water: col, thinner at its edges.
func hdSilhouette(c *canvas, col color.RGBA, alpha float64) {
	in := make([]bool, len(c.px))
	for i, p := range c.px {
		in[i] = p.A > 0
	}
	const reach = 4
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if !in[y*c.w+x] {
				continue
			}
			d := reach + 1
			for r := 1; r <= reach && d > reach; r++ {
				for _, o := range [8][2]int{{r, 0}, {-r, 0}, {0, r}, {0, -r}, {r, r}, {-r, r}, {r, -r}, {-r, -r}} {
					if nx, ny := x+o[0], y+o[1]; !c.in(nx, ny) || !in[ny*c.w+nx] {
						d = r
						break
					}
				}
			}
			a := alpha * (0.35 + 0.65*float64(d-1)/reach)
			// The light through the water plays on its back.
			a *= 0.95 + 0.05*math.Sin(float64(x)*0.045+math.Sin(float64(y)*0.07)*2)
			c.px[y*c.w+x] = withAlpha(col, uint8(255*clamp01(a)))
		}
	}
}

// --- the clock of a boss's state ------------------------------------------

type bossClock struct {
	st, prev sim.CreatureState
	at, seen int
}

var bossClocks = map[sim.CreatureID]bossClock{}

// bossAgeShot, if not negative, is the age every state shows (for stills).
var bossAgeShot = -1

// SetBossAge fixes how many frames into its state every boss and sea monster
// is drawn; a negative age lets them run. For screenshots.
func SetBossAge(age int) { bossAgeShot = age }

// stateAge is how many frames c has been in its state, and the state it was
// in before. A flinch does not count as a state.
func (g *Game) stateAge(c protocol.CreatureView) (int, sim.CreatureState) {
	if len(bossClocks) > 256 {
		clear(bossClocks)
	}
	k, ok := bossClocks[c.ID]
	st := c.State
	if st == sim.CreatureHurt && ok {
		st = k.st
	}
	if !ok || g.frame < k.seen || g.frame-k.seen > 30 {
		k = bossClock{st: st, prev: st, at: g.frame - 100000}
	} else if st != k.st {
		k.prev, k.st, k.at = k.st, st, g.frame
	}
	k.seen = g.frame
	bossClocks[c.ID] = k
	if bossAgeShot >= 0 {
		return bossAgeShot, k.prev
	}
	return g.frame - k.at, k.prev
}
