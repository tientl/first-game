package client

import (
	"image/color"
	"math"
)

// The snake is drawn as a tube along a serpenoid curve, the shape real snakes
// take as they slither: the bend angle along the body follows a cosine, so
// the curves travel down the body from frame to frame. The body is seen from
// above at the game's 3/4 angle (lateral curves squashed into the ground),
// shaded round, with a ridge of light along the spine, dark saddle blotches
// down the back, a pale belly on the near side, and a wedge-shaped head with
// a dark stripe behind the eye. Resting snakes lie coiled with the head up.

var (
	snakeRamp  = mkRamp(0x2b2416, 0x4a3d22, 0x6b5a30, 0x8c7a42, 0xb3a160) // olive-brown
	blotchRamp = mkRamp(0x1a130b, 0x2e2216, 0x463422)                     // saddles on the back
	bellyRamp  = mkRamp(0x8f8456, 0xb8ab78, 0xd9cd9a)
	colTongue  = hex(0xd8344a)
	colFang    = hex(0xf2eee2)
)

// Sprite indices into creatureArt.snake.
const (
	snakeSlither = 4 // slither frames, 0 to snakeSlither-1
	snakeStrike  = snakeSlither
	snakeCoil    = snakeSlither + 1 // resting
	snakeCoilLik = snakeSlither + 2 // resting, flicking its tongue
	snakeSprites = snakeSlither + 3
)

const snakeSquash = 0.6 // how much the ground plane is foreshortened

// spinePt is one sample of a body's centre line, in canvas pixels.
type spinePt struct {
	x, y   float64
	nx, ny float64 // unit normal, pointing to the body's right (the near side, for a snake heading right)
	r      float64 // half the body's width here
	s      float64 // 0 at the tail tip, 1 at the neck
}

// withNormals fills in each point's normal from its neighbours.
func withNormals(pts []spinePt) []spinePt {
	for i := range pts {
		a, b := pts[max(0, i-1)], pts[min(len(pts)-1, i+1)]
		tx, ty := b.x-a.x, b.y-a.y
		if l := math.Hypot(tx, ty); l > 1e-9 {
			pts[i].nx, pts[i].ny = -ty/l, tx/l
		}
	}
	return pts
}

// snakeWidth is the body's half-width at s: a thin tail tip, a thick middle,
// a slimmer neck.
func snakeWidth(s float64) float64 {
	return 0.5 + 1.5*smooth(clamp01(s/0.35)) - 0.55*smooth(clamp01((s-0.8)/0.2))
}

// drawTube fills the body: each pixel takes the sample nearest the head that
// covers it, so a later stretch of body lies over an earlier one, as in a
// coil. Where a higher coil overlaps a lower one, the lower is shadowed.
func drawTube(c *canvas, pts []spinePt) { drawTubeWith(c, pts, snakeScale) }

// drawTubeWith is drawTube with its own skin: shade colours one pixel given
// the sample it belongs to and how far across the body it is (-1..1).
func drawTubeWith(c *canvas, pts []spinePt, shade func(x, y int, p *spinePt, t float64) color.RGBA) {
	if k := c.k(); k != 1 || c.tx != 0 || c.ty != 0 { // a scaled canvas: the spine is in design units
		scaled := make([]spinePt, len(pts))
		for i, p := range pts {
			p.x, p.y = c.toPx(p.x, p.y)
			p.r *= k
			scaled[i] = p
		}
		pts = scaled
	}
	best := make([]float64, c.w*c.h)
	for i := range best {
		best[i] = -1
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			var hit *spinePt
			for i := range pts {
				p := &pts[i]
				dx, dy := px-p.x, py-p.y
				if dx*dx+dy*dy <= p.r*p.r && (hit == nil || p.s > hit.s) {
					hit = p
				}
			}
			if hit == nil {
				continue
			}
			best[y*c.w+x] = hit.s
			t := ((px-hit.x)*hit.nx + (py-hit.y)*hit.ny) / hit.r // -1 far side .. 1 near side
			c.put(x, y, shade(x, y, hit, t))
		}
	}
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			s := best[y*c.w+x]
			if s < 0 {
				continue
			}
			for _, d := range [2][2]int{{0, -1}, {-1, 0}} {
				nx, ny := x+d[0], y+d[1]
				if c.in(nx, ny) && best[ny*c.w+nx] > s+0.2 {
					c.put(x, y, mix(c.at(x, y), colInk, 0.45)) // tucked under the coil above
					break
				}
			}
		}
	}
}

// snakeScale colours one pixel of the body.
func snakeScale(x, y int, p *spinePt, t float64) color.RGBA {
	l := lum(p.nx*t*0.9, p.ny*t*0.9)
	if t > -0.5 && t < -0.15 {
		l += 0.18 // the light along the spine
	}
	// Saddles: diamond-ish dark patches down the back, narrower at the
	// sides, every 1/9 of the body; none on the thin tail tip or the neck.
	at := math.Abs(math.Mod(p.s*9, 1)-0.5) * 2 // 0 mid-saddle .. 1 between
	saddle := p.s > 0.08 && p.s < 0.92 && at < 0.42-0.25*math.Abs(t)
	switch {
	case t > 0.7:
		return bellyRamp.pick(l+0.15, x, y)
	case saddle:
		return blotchRamp.pick(l+0.1, x, y)
	case p.s > 0.08 && p.s < 0.92 && at < 0.58-0.25*math.Abs(t):
		return snakeRamp.pick(l+0.35, x, y) // pale rim around each saddle
	}
	i := snakeRamp.idx(l+0.05, x, y)
	if hashf(x, y, 77) < 0.06 {
		i-- // scales
	}
	return snakeRamp.at(i)
}

// drawSnakeHead draws the wedge-shaped head at (hx, hy) pointing along
// (dx, dy). open shows the mouth and fangs; tongue flicks the forked tongue.
func drawSnakeHead(c *canvas, hx, hy, dx, dy float64, open, tongue bool) {
	px, py := -dy, dx // the head's right side
	at := func(along, side float64) (float64, float64) {
		return hx + dx*along + px*side, hy + dy*along + py*side*snakeSquash*1.4
	}
	var poly []float64
	// Wide at the jaws behind a narrow neck, tapering to a blunt snout.
	for _, v := range [][2]float64{{-2, -2.3}, {0.8, -2.9}, {3.4, -1.5}, {4.6, 0}, {3.4, 1.5}, {0.8, 2.9}, {-2, 2.3}} {
		x, y := at(v[0], v[1])
		poly = append(poly, x, y)
	}
	c.polygon(poly, func(x, y int) color.RGBA {
		fx, fy := float64(x)+0.5-hx, float64(y)+0.5-hy
		along, side := fx*dx+fy*dy, (fx*px+fy*py)/2.6
		l := lum(px*side*0.8, py*side*0.8-0.3) + 0.12
		if along > -1.2 && along < 2.4 && math.Abs(side) < 0.18+0.12*(2.4-along)/3.6 {
			return blotchRamp.pick(l, x, y) // a dark V on the crown
		}
		return snakeRamp.pick(l+0.12, x, y)
	})
	ex, ey := at(1.8, -1.3)
	c.set(int(ex), int(ey), colInk) // the eye
	for k := 1.0; k <= 2; k++ {
		sx, sy := at(1.8-k, -1.3+0.3*k)
		c.set(int(sx), int(sy), blotchRamp[0]) // the stripe behind it
	}
	tx, ty := at(5, 0)
	if open {
		for k := 0.0; k < 2.5; k++ {
			mx, my := at(3.6-k, 0.2)
			c.set(int(mx), int(my), hex(0x3a0d12))
		}
		fx, fy := at(3.4, 0.9)
		c.set(int(fx), int(fy), colFang)
	}
	if tongue {
		for k := 0.0; k < 2; k++ {
			c.set(int(tx+dx*k), int(ty+dy*k), colTongue)
		}
		fx, fy := tx+dx*2.2, ty+dy*2.2
		c.set(int(fx+px*0.8), int(fy-0.8), colTongue)
		c.set(int(fx-px*0.8), int(fy+0.8), colTongue)
	}
}

// serpenoid lays a body of length L along a serpenoid curve ending with its
// neck at (x1, y1), heading right. The head end barely sways, as a real
// snake keeps its head steady while the body waves behind it.
func serpenoid(L, bend, waves, phase, x1, y1 float64) []spinePt {
	const n = 140
	us, vs := make([]float64, n+1), make([]float64, n+1)
	for i := 1; i <= n; i++ {
		s := float64(i) / n
		steady := 1 - 0.75*smooth(clamp01((s-0.72)/0.28))
		th := bend * steady * math.Cos(2*math.Pi*waves*s-phase)
		us[i] = us[i-1] + math.Cos(th)*L/n
		vs[i] = vs[i-1] + math.Sin(th)*L/n
	}
	pts := make([]spinePt, n+1)
	for i := range pts {
		s := float64(i) / n
		pts[i] = spinePt{
			x: x1 + us[i] - us[n],
			y: y1 + (vs[i]-vs[n])*snakeSquash,
			r: snakeWidth(s), s: s,
		}
	}
	return withNormals(pts)
}

// genSnake draws one snake sprite, facing right. Frames below snakeSlither
// slither; then the strike and the resting coil, with and without the
// tongue out.
func genSnake(frame int) sprite {
	c, ax, ay := snakeCanvas(frame)
	return mk(c, ax, ay)
}

// snakeCanvas draws sprite frame on a canvas and returns its anchor.
func snakeCanvas(frame int) (c *canvas, ax, ay float64) {
	switch frame {
	case snakeCoil, snakeCoilLik:
		return snakeCoilCanvas(frame == snakeCoilLik)
	}
	c = newCanvas(44, 24)
	const neckX, neckY = 34.0, 13.0
	strike := frame == snakeStrike
	ph := float64(frame) * 2 * math.Pi / snakeSlither
	if strike {
		ph = 0
	}
	pts := serpenoid(32, 1.3, 1.15, ph, neckX, neckY)
	hx, hy, dx, dy := neckX+1.8, neckY, 1.0, 0.0
	if strike { // the fore-body rears up and the head lunges
		for i := range pts {
			pts[i].y -= 6 * smooth(clamp01((pts[i].s-0.55)/0.45))
			pts[i].x += 2 * smooth(clamp01((pts[i].s-0.7)/0.3))
		}
		pts = withNormals(pts)
		hx, hy, dx, dy = neckX+4, neckY-6.4, 0.97, 0.24
	}
	drawTube(c, pts)
	drawSnakeHead(c, hx, hy, dx, dy, strike, strike || frame == 1)
	c.outline(0.8)
	return c, 20, 14
}

// snakeCoilCanvas draws a snake at rest: coiled in a low pile, the head raised
// in the middle and looking right.
func snakeCoilCanvas(tongue bool) (*canvas, float64, float64) {
	c := newCanvas(30, 24)
	const cx, cy = 14.0, 16.0
	const n = 200
	const turns, inner, outer = 1.3, 2.6, 9.2
	const coilEnd = 0.86 // the rest is the neck
	pts := make([]spinePt, 0, n+1)
	spiral := func(s float64) (float64, float64) {
		ang := 0.3 + 2*math.Pi*turns*s
		rad := outer - (outer-inner)*s/coilEnd
		return cx + rad*math.Cos(ang), cy + rad*math.Sin(ang)*0.72 - 1.6*s // a flat pile, so the rings show
	}
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		var x, y float64
		if s < coilEnd { // spiralling in, each turn a little higher
			x, y = spiral(s)
		} else { // the neck rises out of the middle
			u := (s - coilEnd) / (1 - coilEnd)
			x0, y0 := spiral(coilEnd)
			x, y = lerpf(x0, cx+2, u), lerpf(y0, cy-7.5, smooth(u))
		}
		pts = append(pts, spinePt{x: x, y: y, r: snakeWidth(s) * 0.88, s: s})
	}
	drawTube(c, withNormals(pts))
	drawSnakeHead(c, cx+3.8, cy-8.2, 1, 0, false, tongue)
	c.outline(0.8)
	return c, cx, cy + 3
}
