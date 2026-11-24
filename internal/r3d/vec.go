// Package r3d is a very small software 3D toolkit: enough to build a
// low-poly creature in code, pose it with a skeleton, and render it into
// pixel-art sprites from the game's camera angle. Nothing here knows about
// the game or about Ebitengine; it works on plain slices and image.NRGBA.
//
// Model space: +X is forward (where the creature looks), +Y its left side,
// +Z up. One unit is whatever the model says it is; the camera's Scale turns
// units into pixels.
package r3d

import "math"

// Vec3 is a point or a direction.
type Vec3 struct{ X, Y, Z float64 }

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3             { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3             { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3          { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64          { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64                { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Mul(t)) }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

// Norm returns a at unit length, or a itself when it has none.
func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return a
	}
	return a.Mul(1 / l)
}

// Mat is an affine transform: a 3x3 matrix and a translation, row by row
// (m[0..2] and m[3] make the first row).
type Mat [12]float64

// Identity leaves everything where it is.
func Identity() Mat { return Mat{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0} }

// Translate moves by t.
func Translate(t Vec3) Mat { return Mat{1, 0, 0, t.X, 0, 1, 0, t.Y, 0, 0, 1, t.Z} }

// RotX turns about the X axis (a roll).
func RotX(a float64) Mat {
	s, c := math.Sincos(a)
	return Mat{1, 0, 0, 0, 0, c, -s, 0, 0, s, c, 0}
}

// RotY turns about the Y axis (a pitch): a positive angle dips whatever
// points forward along +X.
func RotY(a float64) Mat {
	s, c := math.Sincos(a)
	return Mat{c, 0, s, 0, 0, 1, 0, 0, -s, 0, c, 0}
}

// RotZ turns about the Z axis (a yaw): a positive angle swings +X towards +Y.
func RotZ(a float64) Mat {
	s, c := math.Sincos(a)
	return Mat{c, -s, 0, 0, s, c, 0, 0, 0, 0, 1, 0}
}

// Euler is a yaw, then a pitch, then a roll, as one transform: r.Z about Z,
// r.Y about Y, r.X about X.
func Euler(r Vec3) Mat {
	m := Identity()
	if r.Z != 0 {
		m = RotZ(r.Z)
	}
	if r.Y != 0 {
		m = m.Mul(RotY(r.Y))
	}
	if r.X != 0 {
		m = m.Mul(RotX(r.X))
	}
	return m
}

// Mul returns the transform that applies b, then a.
func (a Mat) Mul(b Mat) Mat {
	var m Mat
	for r := 0; r < 3; r++ {
		a0, a1, a2, a3 := a[r*4], a[r*4+1], a[r*4+2], a[r*4+3]
		m[r*4] = a0*b[0] + a1*b[4] + a2*b[8]
		m[r*4+1] = a0*b[1] + a1*b[5] + a2*b[9]
		m[r*4+2] = a0*b[2] + a1*b[6] + a2*b[10]
		m[r*4+3] = a0*b[3] + a1*b[7] + a2*b[11] + a3
	}
	return m
}

// Point transforms a position.
func (m Mat) Point(p Vec3) Vec3 {
	return Vec3{
		m[0]*p.X + m[1]*p.Y + m[2]*p.Z + m[3],
		m[4]*p.X + m[5]*p.Y + m[6]*p.Z + m[7],
		m[8]*p.X + m[9]*p.Y + m[10]*p.Z + m[11],
	}
}

// Dir transforms a direction: no translation.
func (m Mat) Dir(p Vec3) Vec3 {
	return Vec3{
		m[0]*p.X + m[1]*p.Y + m[2]*p.Z,
		m[4]*p.X + m[5]*p.Y + m[6]*p.Z,
		m[8]*p.X + m[9]*p.Y + m[10]*p.Z,
	}
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// Smooth is the usual smoothstep of v between lo and hi (which may run
// downwards).
func Smooth(lo, hi, v float64) float64 {
	t := clamp01((v - lo) / (hi - lo))
	return t * t * (3 - 2*t)
}

// RGB is a colour with channels in [0, 1].
type RGB struct{ R, G, B float64 }

// Hex makes a colour of 0xRRGGBB.
func Hex(h uint32) RGB {
	return RGB{float64(h>>16&0xff) / 255, float64(h>>8&0xff) / 255, float64(h&0xff) / 255}
}

func (a RGB) Mix(b RGB, t float64) RGB {
	return RGB{a.R + (b.R-a.R)*t, a.G + (b.G-a.G)*t, a.B + (b.B-a.B)*t}
}

func (a RGB) Scale(s float64) RGB { return RGB{a.R * s, a.G * s, a.B * s} }
