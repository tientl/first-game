package dinopaint

import "math"

// SDF is a shape as its signed distance: negative inside, in pixels of the
// density-1 drawing (the guide), the units every description is written in.
type SDF func(x, y float64) float64

// Circle is a disc.
func Circle(cx, cy, r float64) SDF {
	return func(x, y float64) float64 { return math.Hypot(x-cx, y-cy) - r }
}

// Ellipse is an ellipse rx by ry about (cx, cy), turned rot radians
// (clockwise on screen). The distance is a good estimate near the edge.
func Ellipse(cx, cy, rx, ry, rot float64) SDF {
	s, c := math.Sincos(-rot)
	return func(x, y float64) float64 {
		dx, dy := x-cx, y-cy
		u, v := dx*c-dy*s, dx*s+dy*c
		k0 := math.Hypot(u/rx, v/ry)
		if k0 == 0 {
			return -min(rx, ry)
		}
		k1 := math.Hypot(u/(rx*rx), v/(ry*ry))
		return k0 * (k0 - 1) / k1
	}
}

// Cone is a limb: round at both ends, ra at (ax, ay) and rb at (bx, by).
func Cone(ax, ay, ra, bx, by, rb float64) SDF {
	pbx, pby := bx-ax, by-ay
	h := pbx*pbx + pby*pby
	b := ra - rb
	if b*b >= h { // one end swallows the other
		if ra > rb {
			return Circle(ax, ay, ra)
		}
		return Circle(bx, by, rb)
	}
	cx, cy := math.Sqrt(h-b*b), b
	return func(x, y float64) float64 {
		px, py := x-ax, y-ay
		qx := math.Abs(px*pby-py*pbx) / h
		qy := (px*pbx + py*pby) / h
		k := cx*qy - cy*qx
		m := cx*qx + cy*qy
		n := qx*qx + qy*qy
		switch {
		case k < 0:
			return math.Sqrt(h*n) - ra
		case k > cx:
			return math.Sqrt(h*(n+1-2*qy)) - rb
		}
		return m - ra
	}
}

// Path is a chain of cones through points {x, y, r}.
func Path(pts ...[3]float64) SDF {
	var s []SDF
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		s = append(s, Cone(a[0], a[1], a[2], b[0], b[1], b[2]))
	}
	return Union(s...)
}

// Poly is a polygon through x, y pairs.
func Poly(xy ...float64) SDF {
	n := len(xy) / 2
	return func(x, y float64) float64 {
		d := (x-xy[0])*(x-xy[0]) + (y-xy[1])*(y-xy[1])
		s := 1.0
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			vix, viy, vjx, vjy := xy[2*i], xy[2*i+1], xy[2*j], xy[2*j+1]
			ex, ey := vjx-vix, vjy-viy
			wx, wy := x-vix, y-viy
			t := clamp01((wx*ex + wy*ey) / (ex*ex + ey*ey))
			bx, by := wx-ex*t, wy-ey*t
			d = min(d, bx*bx+by*by)
			c1, c2, c3 := y >= viy, y < vjy, ex*wy > ey*wx
			if (c1 && c2 && c3) || (!c1 && !c2 && !c3) {
				s = -s
			}
		}
		return s * math.Sqrt(d)
	}
}

// Union is everything in any of s.
func Union(s ...SDF) SDF {
	return func(x, y float64) float64 {
		d := math.Inf(1)
		for _, f := range s {
			d = min(d, f(x, y))
		}
		return d
	}
}

// Blend is the union of s with the joins filled k pixels round, like
// flesh over bones.
func Blend(k float64, s ...SDF) SDF {
	return func(x, y float64) float64 {
		d := s[0](x, y)
		for _, f := range s[1:] {
			b := f(x, y)
			h := clamp01(0.5 + 0.5*(b-d)/k)
			d = lerp(b, d, h) - k*h*(1-h)
		}
		return d
	}
}

// Cut is a without b.
func Cut(a, b SDF) SDF { return func(x, y float64) float64 { return max(a(x, y), -b(x, y)) } }

// Inter is what is in both a and b.
func Inter(a, b SDF) SDF { return func(x, y float64) float64 { return max(a(x, y), b(x, y)) } }

// Grow is s r pixels bigger all round (smaller for r < 0).
func (s SDF) Grow(r float64) SDF { return func(x, y float64) float64 { return s(x, y) - r } }

// Below is the half-plane under the line through (ax, ay) and (bx, by),
// taken left to right.
func Below(ax, ay, bx, by float64) SDF {
	nx, ny := -(by - ay), bx-ax
	l := math.Hypot(nx, ny)
	nx, ny = nx/l, ny/l
	return func(x, y float64) float64 { return -((x-ax)*nx + (y-ay)*ny) }
}

// segDist is how far (x, y) is from the segment a-b, and how far along it
// the nearest point is (0 to 1).
func segDist(x, y, ax, ay, bx, by float64) (float64, float64) {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = clamp01(((x-ax)*dx + (y-ay)*dy) / l2)
	}
	return math.Hypot(x-ax-t*dx, y-ay-t*dy), t
}
