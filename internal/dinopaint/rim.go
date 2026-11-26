package dinopaint

import "math"

// RimSpec is a row of spikes along the edge of an ellipse (a frill's
// epoccipitals): the ellipse as Ellipse takes it, the spikes N of them from
// angle T0 to T1 (radians, 0 to the right, π/2 down), each rooted In inside
// the edge and standing Out beyond it, W wide at the root.
type RimSpec struct {
	X, Y, RX, RY, Rot float64
	T0, T1            float64
	N                 int
	In, Out, W        float64
	Ramp              []uint32
	Shade             float64
}

// RimSpikes paints the spikes of r, each a little horn pointing straight
// out of the edge.
func (p *Painter) RimSpikes(r RimSpec) {
	s, c := math.Sincos(r.Rot)
	for k := 0; k < r.N; k++ {
		t := r.T0
		if r.N > 1 {
			t += (r.T1 - r.T0) * float64(k) / float64(r.N-1)
		}
		st, ct := math.Sincos(t)
		lx, ly := r.RX*ct, r.RY*st
		nx, ny := ct/r.RX, st/r.RY // the normal, before turning
		l := math.Hypot(nx, ny)
		nx, ny = nx/l, ny/l
		ex, ey := r.X+lx*c-ly*s, r.Y+lx*s+ly*c
		ux, uy := nx*c-ny*s, nx*s+ny*c
		p.Horn(HornSpec{X0: ex - ux*r.In, Y0: ey - uy*r.In, X1: ex + ux*r.Out, Y1: ey + uy*r.Out,
			W: r.W, Ramp: r.Ramp, Shade: r.Shade})
	}
}
