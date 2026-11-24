package r3d

import "math"

// Vertex is a corner of the mesh in its rest pose. It follows bone B0, or a
// blend of B0 and B1 (W is B1's share), which keeps a neck or a tail smooth
// where it bends.
type Vertex struct {
	P, N   Vec3
	C      RGB
	B0, B1 int
	W      float64
	Emit   float64 // how much of the colour ignores the light: 1 glows
}

// Mesh is a bag of triangles over shared vertices.
type Mesh struct {
	V []Vertex
	T [][3]int32
}

// Ring is one cross-section of a loft: an ellipse (or, with Exp above 2, a
// rounded box) about centre C, spanned by the half-axes U and V. Angle 0 is
// along U, a quarter turn along V.
type Ring struct {
	C, U, V Vec3
	Exp     float64 // superellipse exponent; 0 means 2, a plain ellipse
	B0, B1  int
	W       float64
}

// At is the point of the ring at angle th.
func (r Ring) At(th float64) Vec3 {
	s, c := math.Sincos(th)
	if r.Exp > 2 {
		e := 2 / r.Exp
		c = math.Copysign(math.Pow(math.Abs(c), e), c)
		s = math.Copysign(math.Pow(math.Abs(s), e), s)
	}
	return r.C.Add(r.U.Mul(c)).Add(r.V.Mul(s))
}

// Loft skins a row of rings with segs quads each and closes both ends with a
// fan, so the result is a closed, smooth surface. col gives the colour of
// the vertex at ring i, angle th, position p; emit is how much it glows.
// Normals are smoothed over the surface and point outwards.
func (m *Mesh) Loft(rings []Ring, segs int, col func(i int, th float64, p Vec3) RGB, emit float64) {
	if len(rings) < 2 {
		return
	}
	base := len(m.V)
	for i, r := range rings {
		for k := 0; k < segs; k++ {
			th := 2 * math.Pi * float64(k) / float64(segs)
			p := r.At(th)
			m.V = append(m.V, Vertex{P: p, C: col(i, th, p), B0: r.B0, B1: r.B1, W: r.W, Emit: emit})
		}
	}
	at := func(i, k int) int32 { return int32(base + i*segs + k%segs) }
	for i := 0; i+1 < len(rings); i++ {
		for k := 0; k < segs; k++ {
			a, b, c, d := at(i, k), at(i, k+1), at(i+1, k), at(i+1, k+1)
			m.T = append(m.T, [3]int32{a, b, d}, [3]int32{a, d, c})
		}
	}
	// The end caps: a fan about the middle of the first and the last ring.
	for _, i := range [2]int{0, len(rings) - 1} {
		r := rings[i]
		ci := int32(len(m.V))
		m.V = append(m.V, Vertex{P: r.C, C: col(i, math.Pi/2, r.C), B0: r.B0, B1: r.B1, W: r.W, Emit: emit})
		for k := 0; k < segs; k++ {
			m.T = append(m.T, [3]int32{ci, at(i, k), at(i, k+1)})
		}
	}
	m.smooth(base, rings, segs)
}

// smooth sets the normals of the loft whose vertices start at base.
func (m *Mesh) smooth(base int, rings []Ring, segs int) {
	n := len(rings)
	mid := rings[0].C.Add(rings[n-1].C).Mul(0.5)
	for ti := range m.T {
		t := m.T[ti]
		if int(t[0]) < base && int(t[1]) < base {
			continue
		}
		a, b, c := m.V[t[0]].P, m.V[t[1]].P, m.V[t[2]].P
		fn := b.Sub(a).Cross(c.Sub(a)) // its length weighs by area
		// Face it away from the loft's own axis.
		ctr := a.Add(b).Add(c).Mul(1.0 / 3)
		ref := mid
		if i := (int(t[1]) - base) / segs; i < n {
			ref = rings[i].C
			if j := (int(t[2]) - base) / segs; j < n {
				ref = ref.Add(rings[j].C).Mul(0.5)
			}
		}
		out := ctr.Sub(ref)
		if e := int(t[0]) - base - n*segs; e == 0 { // a cap looks along the axis
			out = rings[0].C.Sub(rings[1].C)
		} else if e == 1 {
			out = rings[n-1].C.Sub(rings[n-2].C)
		}
		if out.Len() < 1e-9 {
			out = ctr.Sub(mid)
		}
		if fn.Dot(out) < 0 {
			fn = fn.Mul(-1)
		}
		for _, vi := range t {
			if int(vi) >= base {
				m.V[vi].N = m.V[vi].N.Add(fn)
			}
		}
	}
	for i := base; i < len(m.V); i++ {
		m.V[i].N = m.V[i].N.Norm()
	}
	// The two cap centres look along the loft's axis.
	for e, i := range [2]int{0, n - 1} {
		j := 1
		if e == 1 {
			j = n - 2
		}
		d := rings[i].C.Sub(rings[j].C).Norm()
		if d.Len() > 0 {
			m.V[base+n*segs+e].N = d
		}
	}
}

// Ball adds an ellipsoid with half-axes rx, ry, rz about c, in one colour.
func (m *Mesh) Ball(c Vec3, rx, ry, rz float64, col RGB, bone int, emit float64) {
	const n = 6
	rings := make([]Ring, 0, n+1)
	for i := 0; i <= n; i++ {
		a := math.Pi * (float64(i) + 0.5) / float64(n+1)
		s, co := math.Sincos(a)
		rings = append(rings, Ring{C: c.Add(V(-co*rx, 0, 0)), U: V(0, ry*s, 0), V: V(0, 0, rz*s), B0: bone, B1: bone})
	}
	m.Loft(rings, 8, func(int, float64, Vec3) RGB { return col }, emit)
}

// Spike adds a cone from base (radius r) to tip.
func (m *Mesh) Spike(base, tip Vec3, r float64, col, tipCol RGB, bone int, emit float64) {
	ax := tip.Sub(base)
	u := ax.Cross(V(0, 1, 0))
	if u.Len() < 1e-6 {
		u = ax.Cross(V(1, 0, 0))
	}
	u = u.Norm()
	v := ax.Cross(u).Norm()
	rings := []Ring{
		{C: base, U: u.Mul(r), V: v.Mul(r), B0: bone, B1: bone},
		{C: base.Lerp(tip, 0.55), U: u.Mul(r * 0.62), V: v.Mul(r * 0.62), B0: bone, B1: bone},
		{C: tip, U: u.Mul(r * 0.12), V: v.Mul(r * 0.12), B0: bone, B1: bone},
	}
	m.Loft(rings, 6, func(i int, _ float64, _ Vec3) RGB {
		if i == 2 {
			return tipCol
		}
		return col
	}, emit)
}

// --- skeleton ---------------------------------------------------------------

// Bone is a joint: where it pivots in the rest pose, and what it hangs from
// (-1 for the root). A parent always comes before its children.
type Bone struct {
	Name   string
	Parent int
	Pivot  Vec3
}

// Skeleton is the bones of a model.
type Skeleton struct{ Bones []Bone }

// Add appends a bone and returns its index.
func (s *Skeleton) Add(name string, parent int, pivot Vec3) int {
	s.Bones = append(s.Bones, Bone{name, parent, pivot})
	return len(s.Bones) - 1
}

// Pose is a turn (see Euler) and a shift for every bone, on top of the rest
// pose.
type Pose struct {
	Rot []Vec3
	Off []Vec3
}

// NewPose is the rest pose.
func (s *Skeleton) NewPose() *Pose {
	return &Pose{Rot: make([]Vec3, len(s.Bones)), Off: make([]Vec3, len(s.Bones))}
}

// Matrices returns, for each bone, the transform that carries a rest-pose
// point attached to it to where pose p puts it.
func (s *Skeleton) Matrices(p *Pose) []Mat {
	world := make([]Mat, len(s.Bones))
	skin := make([]Mat, len(s.Bones))
	for i, b := range s.Bones {
		local := b.Pivot
		parent := Identity()
		if b.Parent >= 0 {
			local = local.Sub(s.Bones[b.Parent].Pivot)
			parent = world[b.Parent]
		}
		world[i] = parent.Mul(Translate(local.Add(p.Off[i]))).Mul(Euler(p.Rot[i]))
		skin[i] = world[i].Mul(Translate(b.Pivot.Mul(-1)))
	}
	return skin
}

// Skin writes mesh m, moved by the bone transforms, into out (which keeps
// m's triangles and colours).
func (m *Mesh) Skin(mats []Mat, out *Mesh) {
	if len(out.V) != len(m.V) {
		out.V = make([]Vertex, len(m.V))
		copy(out.V, m.V)
		out.T = m.T
	}
	for i := range m.V {
		v := &m.V[i]
		p, n := mats[v.B0].Point(v.P), mats[v.B0].Dir(v.N)
		if v.W > 0 && v.B1 != v.B0 {
			p = p.Lerp(mats[v.B1].Point(v.P), v.W)
			n = n.Lerp(mats[v.B1].Dir(v.N), v.W)
		}
		out.V[i].P, out.V[i].N = p, n
	}
}
