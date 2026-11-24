package r3d

import "math"

// Camera looks at the model from the south and from above, without
// perspective, the way the game looks at its ground: Elev is how high it
// stands above the horizon (0 is side-on, a quarter turn is straight down).
// Yaw turns the model about its upright axis first: 0 faces right, a quarter
// turn faces away from the viewer.
type Camera struct {
	Yaw, Elev float64
	Scale     float64 // pixels per unit
	OX, OY    float64 // the pixel where the model's origin lands
}

// View is the transform from model space to view space: x to the right and
// y down, both in pixels, z towards the viewer in units.
func (c Camera) View() Mat {
	s, co := math.Sincos(c.Elev)
	// World: x east, y north, z up. On screen north goes up and away.
	tilt := Mat{
		c.Scale, 0, 0, c.OX,
		0, -s * c.Scale, -co * c.Scale, c.OY,
		0, -co, s, 0,
	}
	return tilt.Mul(RotZ(c.Yaw))
}

// normals is View without the scale and the shift, for directions.
func (c Camera) normals() Mat {
	s, co := math.Sincos(c.Elev)
	tilt := Mat{1, 0, 0, 0, 0, -s, -co, 0, 0, -co, s, 0}
	return tilt.Mul(RotZ(c.Yaw))
}

// Light is how a surface is shaded, all in view space, so the light stays
// put on the screen whichever way the model faces.
type Light struct {
	Dir      Vec3    // towards the light
	Wrap     float64 // how far the light wraps round: brightness is Wrap + (1-Wrap) * n.l
	Dark     float64 // the colour's scale in full shadow
	Bright   float64 // and in full light
	Shadow   RGB     // a tint multiplied into the shadows
	Rim      float64 // strength of a pale rim along edges that face up
	RimColor RGB
}

// Target is what Render draws into.
type Target struct {
	W, H int
	Col  []float32 // r, g, b per pixel
	Z    []float32 // depth towards the viewer; -Inf where empty
}

// NewTarget makes an empty target.
func NewTarget(w, h int) *Target {
	t := &Target{W: w, H: h, Col: make([]float32, w*h*3), Z: make([]float32, w*h)}
	t.Clear()
	return t
}

// Clear empties the target.
func (t *Target) Clear() {
	ninf := float32(math.Inf(-1))
	for i := range t.Z {
		t.Z[i] = ninf
	}
}

type scrVert struct {
	x, y, z    float64
	nx, ny, nz float64
}

// Render draws mesh m, with a depth buffer and per-pixel lighting.
func Render(t *Target, m *Mesh, cam Camera, lt Light) {
	view, nrm := cam.View(), cam.normals()
	sv := make([]scrVert, len(m.V))
	for i := range m.V {
		p, n := view.Point(m.V[i].P), nrm.Dir(m.V[i].N)
		sv[i] = scrVert{p.X, p.Y, p.Z, n.X, n.Y, n.Z}
	}
	ld := lt.Dir.Norm()
	for _, tri := range m.T {
		a, b, c := &sv[tri[0]], &sv[tri[1]], &sv[tri[2]]
		area := (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
		if math.Abs(area) < 1e-9 {
			continue
		}
		x0 := max(0, int(math.Floor(min(a.x, b.x, c.x))))
		x1 := min(t.W-1, int(math.Ceil(max(a.x, b.x, c.x))))
		y0 := max(0, int(math.Floor(min(a.y, b.y, c.y))))
		y1 := min(t.H-1, int(math.Ceil(max(a.y, b.y, c.y))))
		if x0 > x1 || y0 > y1 {
			continue
		}
		va, vb, vc := &m.V[tri[0]], &m.V[tri[1]], &m.V[tri[2]]
		inv := 1 / area
		for y := y0; y <= y1; y++ {
			py := float64(y) + 0.5
			for x := x0; x <= x1; x++ {
				px := float64(x) + 0.5
				w0 := ((b.x-px)*(c.y-py) - (b.y-py)*(c.x-px)) * inv
				w1 := ((c.x-px)*(a.y-py) - (c.y-py)*(a.x-px)) * inv
				w2 := 1 - w0 - w1
				if w0 < 0 || w1 < 0 || w2 < 0 {
					continue
				}
				z := float32(w0*a.z + w1*b.z + w2*c.z)
				i := y*t.W + x
				if z <= t.Z[i] {
					continue
				}
				t.Z[i] = z
				n := Vec3{w0*a.nx + w1*b.nx + w2*c.nx, w0*a.ny + w1*b.ny + w2*c.ny, w0*a.nz + w1*b.nz + w2*c.nz}.Norm()
				if n.Z < 0 { // seen from behind: the inside of a mouth
					n = n.Mul(-1)
				}
				base := RGB{
					w0*va.C.R + w1*vb.C.R + w2*vc.C.R,
					w0*va.C.G + w1*vb.C.G + w2*vc.C.G,
					w0*va.C.B + w1*vb.C.B + w2*vc.C.B,
				}
				col := lt.shade(base, n, ld)
				if e := w0*va.Emit + w1*vb.Emit + w2*vc.Emit; e > 0 {
					col = col.Mix(base, e)
				}
				t.Col[i*3], t.Col[i*3+1], t.Col[i*3+2] = float32(col.R), float32(col.G), float32(col.B)
			}
		}
	}
}

// shade lights colour base on a surface with normal n.
func (lt *Light) shade(base RGB, n, ld Vec3) RGB {
	l := clamp01(lt.Wrap + (1-lt.Wrap)*n.Dot(ld))
	col := base.Scale(lt.Dark + (lt.Bright-lt.Dark)*l)
	sh := 1 - l
	col = RGB{col.R * (1 + (lt.Shadow.R-1)*sh), col.G * (1 + (lt.Shadow.G-1)*sh), col.B * (1 + (lt.Shadow.B-1)*sh)}
	if lt.Rim > 0 && n.Y < 0 {
		e := 1 - n.Z
		col = col.Mix(lt.RimColor, clamp01(lt.Rim*e*e*e*-n.Y))
	}
	return col
}

// Shade is the lit colour of base at brightness l in [0, 1], without a rim:
// what a palette is built from.
func (lt *Light) Shade(base RGB, l float64) RGB {
	col := base.Scale(lt.Dark + (lt.Bright-lt.Dark)*l)
	sh := 1 - l
	return RGB{col.R * (1 + (lt.Shadow.R-1)*sh), col.G * (1 + (lt.Shadow.G-1)*sh), col.B * (1 + (lt.Shadow.B-1)*sh)}
}
