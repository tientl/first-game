package dinopaint

import "math"

// Coords lays a coat of feathers or a membrane's fibres over the body: it
// takes a guide point to s, how far along the flow it is (feathers point
// from root to tip towards +s), and t, how far across it, both in guide
// pixels.
type Coords func(x, y float64) (s, t float64)

// Dir is a flow straight along angle a (radians; 0: right, pi/2: down).
func Dir(a float64) Coords {
	c, s := math.Cos(a), math.Sin(a)
	return func(x, y float64) (float64, float64) { return x*c + y*s, -x*s + y*c }
}

// Fan is a flow out from (cx, cy), centred on angle a: a tail's fan, the
// wing's long feathers spreading from the hand. Across it, t is the angle
// times r0, so feathers are as wide as they are meant to be r0 out.
func Fan(cx, cy, r0, a float64) Coords {
	return func(x, y float64) (float64, float64) {
		dx, dy := x-cx, y-cy
		th := math.Remainder(math.Atan2(dy, dx)-a, 2*math.Pi)
		return math.Hypot(dx, dy), th * r0
	}
}

// Along is a flow along the line through the points (x, y pairs), from the
// first to the last, smoothed into a curve: the coat of a back from the
// nape to the tail.
func Along(pts ...float64) Coords {
	l := newCurve(pts)
	return func(x, y float64) (float64, float64) { return l.at(x, y) }
}

// Away is a flow away from the line through the points, on its side side
// (+1: to the right of the line taken from its first point to its last,
// on the screen; -1: to the left): long feathers along an arm, each
// growing out of it.
func Away(side float64, pts ...float64) Coords {
	l := newCurve(pts)
	return func(x, y float64) (float64, float64) {
		s, t := l.at(x, y)
		return t * side, s
	}
}

// curve is a smooth line through points, sampled finely.
type curve struct {
	x, y, run []float64
}

// newCurve runs a Catmull-Rom spline through the points.
func newCurve(pts []float64) *curve {
	n := len(pts) / 2
	pt := func(i int) (float64, float64) {
		i = max(0, min(n-1, i))
		return pts[2*i], pts[2*i+1]
	}
	c := &curve{}
	const sub = 8
	for i := 0; i+1 < n; i++ {
		x0, y0 := pt(i - 1)
		x1, y1 := pt(i)
		x2, y2 := pt(i + 1)
		x3, y3 := pt(i + 2)
		for k := 0; k < sub; k++ {
			t := float64(k) / sub
			c.x = append(c.x, catmull(x0, x1, x2, x3, t))
			c.y = append(c.y, catmull(y0, y1, y2, y3, t))
		}
	}
	x, y := pt(n - 1)
	c.x, c.y = append(c.x, x), append(c.y, y)
	c.run = make([]float64, len(c.x))
	for i := 1; i < len(c.x); i++ {
		c.run[i] = c.run[i-1] + math.Hypot(c.x[i]-c.x[i-1], c.y[i]-c.y[i-1])
	}
	return c
}

func catmull(p0, p1, p2, p3, t float64) float64 {
	return 0.5 * (2*p1 + (-p0+p2)*t + (2*p0-5*p1+4*p2-p3)*t*t + (-p0+3*p1-3*p2+p3)*t*t*t)
}

// at is how far along the curve the point nearest (x, y) is, and how far
// (x, y) is from it: + to the right of the curve on the screen. Past its
// ends the curve runs on straight.
func (c *curve) at(x, y float64) (float64, float64) {
	n := len(c.x)
	best, bs, bt := math.Inf(1), 0.0, 0.0
	for k := 0; k+1 < n; k++ {
		ax, ay, bx, by := c.x[k], c.y[k], c.x[k+1], c.y[k+1]
		dx, dy := bx-ax, by-ay
		l2 := dx*dx + dy*dy
		if l2 == 0 {
			continue
		}
		u := ((x-ax)*dx + (y-ay)*dy) / l2
		if k > 0 {
			u = max(u, 0)
		}
		if k+2 < n {
			u = min(u, 1)
		}
		qx, qy := ax+u*dx, ay+u*dy
		d := math.Hypot(x-qx, y-qy)
		if d < best {
			l := math.Sqrt(l2)
			best = d
			bs = c.run[k] + u*l
			bt = ((x-ax)*-dy + (y-ay)*dx) / l // + right of the line, on a screen whose y runs down
		}
	}
	return bs, bt
}
