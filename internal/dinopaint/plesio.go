package dinopaint

import "image/color"

// The sea serpent: dinos/plesio.png, 64 x 50, facing right, a plesiosaur
// swimming at the surface, its long neck raised, the water's foam and
// ripples drawn under it. Its rig is "plesio" in
// internal/client/dinorigs.go: the neck and head turn about (34, 40); the
// jaw opens about (51.5, 10.15) along the line to (60.6, 10.25), the
// upper teeth painted above that line.

var (
	plesioTeal = []uint32{0x0a1a22, 0x0f2a34, 0x16404c, 0x1e5864, 0x2a727c, 0x3c8e96, 0x58aab0, 0x7cc6c8}
	plesioIris = []uint32{0x4a2a04, 0x7a4a06, 0xb07808, 0xdca414, 0xffc838, 0xffe68a}
	plesioFang = []uint32{0x6a6a5c, 0x9a9a88, 0xc4c2ae, 0xe2e0cc, 0xf6f4e6, 0xfffdf4}
)

func init() { Register(plesio()) }

// plesioWater is the drawing's water: the foam along the waterline and the
// ripples under it.
var plesioWater = Poly(-1, 43.7, 65, 43.7, 65, 51, -1, 51)

func plesio() *Species {
	tail := Path([3]float64{2.8, 33.4, 1.6}, [3]float64{4.4, 37.6, 2.5}, [3]float64{7.6, 42.4, 3}) // its tail, curled up out of the water
	body := Blend(3,
		tail,
		Ellipse(22.5, 42.4, 14.5, 6.8, 0),
	)
	neck := Blend(2,
		Path([3]float64{34.2, 41, 5}, [3]float64{36, 30, 4.6}, [3]float64{40.6, 20, 4}, [3]float64{44.2, 12.6, 3.4}, [3]float64{47.4, 8.6, 3.2}),
		Ellipse(33, 40, 5, 4, -1),
	)
	head := Blend(1.2,
		Poly(44, 8, 47, 4.6, 50, 3.2, 56, 3.1, 59.2, 4.4, 61.8, 6.4, 62.6, 8.6, 61.8, 10.4, 60.6, 11.6, 56, 12.6, 51, 12.9, 47.6, 12.3, 45, 11),
		Circle(47.4, 8.6, 3.2),
	)
	jaw := Poly(48.6, 10.2, 61.6, 10.2, 60.6, 11.6, 56, 12.6, 51, 12.9, 48.4, 11.8)
	// The near flippers, paddles held flat along its side at the waterline.
	fore := Blend(0.6, Cone(32.4, 40.6, 1.7, 27.2, 42.4, 1.9), Cone(27.2, 42.4, 1.85, 22.4, 43.3, 0.7))
	hind := Blend(0.6, Cone(15.4, 40.8, 1.3, 12.2, 42.4, 1.35), Cone(12.2, 42.4, 1.3, 9, 43.2, 0.5))
	return &Species{
		Name:        "plesio",
		GuideDetail: 0.25,
		Ink:         20,
		Recolor: []Recolor{
			// The guide's teeth are painted again; under them the lips.
			{Region: Poly(47, 8.4, 63, 8.4, 63, 11, 47, 11), From: []string{"foam", "pale"}, To: "dark"},
		},
		Materials: []Material{
			{Name: "teal", Seeds: []uint32{0x3a808a, 0x5ea4a6, 0x26606e, 0x1d4856, 0x1a4656, 0x94cac4, 0x659ca4, 0x81b8ba, 0xa8c9ca, 0xaacfd0, 0xafd7d6, 0xc1e5e2, 0x4f707b},
				Ramp: plesioTeal},
			// Dark across the eye and along the back of the head, and the
			// shadowed side under the waterline.
			{Name: "dark", Seeds: []uint32{0x0b1e2c, 0x12303f, 0x143342, 0x0d2230, 0x102025, 0x17292d, 0x0a171d},
				Ramp: []uint32{0x050c12, 0x08121a, 0x0c1a24, 0x10222e, 0x142b38, 0x1a3544, 0x204052, 0x284c60}, Bias: 0.04},
			// The pale throat and belly.
			{Name: "pale", Seeds: []uint32{0xbcbe98, 0x8e9a7e, 0x5e6e62, 0xe0dab4, 0x41505a, 0x333738, 0x202725, 0x171e1f},
				Ramp: []uint32{0x22261e, 0x343a2e, 0x4c5442, 0x666e58, 0x828a70, 0x9ea68a, 0xbac0a4, 0xd4d8c0}, Relief: 0.4, Smooth: 0.55},
			{Name: "foam", Seeds: []uint32{0xffffff, 0xd6f0f6, 0xf4f0e0, 0xf8f2d6}, Ramp: plesioTeal},
			{Name: "ripple", Seeds: []uint32{0x96c8e0, 0xa0d0e4}, Ramp: plesioTeal},
		},
		Parts: []Part{
			{Name: "body", Shape: body, Round: 5, Soft: 0.9, Scale: 1, Muscles: []Muscle{
				Mu(22, 38.5, 9, 3.5, 0, 1.2),     // the broad back
				Mu(30, 38.5, 4, 3, 0.4, 1.0),     // the shoulders
				Mu(4.6, 37, 1.8, 3.4, -0.4, 0.7), // the tail
			}},
			{Name: "hind", Shape: hind, Round: 1.2, Soft: 0.6, Scale: 0.6, Mat: "teal", Muscles: []Muscle{
				Mu(13.6, 41.4, 1.8, 0.8, -0.4, 0.4),
			}},
			{Name: "neck", Shape: neck, Round: 4, Soft: 0.9, Scale: 0.9, Onto: "body", Seam: Below(20, 50, 40, 50), Muscles: []Muscle{
				Mu(36.6, 31, 6, 2.2, -1.25, 0.9), // the long muscles of the neck
				Mu(41, 20.5, 5, 1.9, -1.1, 0.8),
				Mu(44.4, 13, 3.4, 1.6, -0.9, 0.6),
			}},
			{Name: "head", Shape: head, Round: 2.4, Soft: 0.7, Scale: 0.7, Onto: "neck", Seam: Below(40, 14, 64, 14), Muscles: []Muscle{
				Mu(51, 5.6, 3.4, 1.6, 0, 0.9),   // the skull
				Mu(55.6, 4.8, 1.6, 0.9, 0, 0.8), // the brow over the eye
				Mu(59.6, 7, 2.4, 1.6, 0.3, 0.6), // the snout
			}},
			{Name: "jaw", Shape: jaw, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "pale", Muscles: []Muscle{
				Mu(54, 11.4, 4, 0.9, 0, 0.4),
			}},
			{Name: "fore", Shape: fore, Round: 1.5, Soft: 0.6, Scale: 0.6, Mat: "teal", Muscles: []Muscle{
				Mu(28.4, 41.6, 3, 1.1, -0.3, 0.6), // the paddle's thick leading edge
			}},
		},
		Skin: Skin{Flank: 1.6, Back: 2.0, Belly: 1.3, BackDepth: 3, BellyDepth: 3, Relief: 0.45, Seed: 71},
		Folds: []Fold{
			// Creases across the pale throat, close at the root of the neck.
			{Pts: []float64{37.4, 38.6, 40.2, 38.2}, N: 5, DX: 0.6, DY: -2.6, Depth: 0.9},
			{Pts: []float64{40.4, 25.4, 43, 25.6}, N: 4, DX: 0.9, DY: -2.6, Depth: 0.7},
			// The lines between the flippers' bones.
			{Pts: []float64{31, 41.6, 24.4, 43.1}, N: 1, Depth: 0.6},
		},
		Details: plesioDetails,
	}
}

func plesioDetails(p *Painter) {
	// The skin is wet: a sharp sheen along the lit side of every form.
	p.Gloss(GlossSpec{Region: Cut(Poly(-1, -1, 65, -1, 65, 51, -1, 51), plesioWater), Strength: 0.75, Sharp: 22, Tint: 0xeaffff, Steps: 3})
	// Long, thin teeth hanging over the lower jaw, then the lips' edge.
	p.Teeth(TeethSpec{X0: 50, Y0: 8.75, X1: 61.2, Y1: 8.95, Bottom: 10.15, N: 8, Len: 1.5,
		Ramp: plesioFang, Gap: 0x0c1418, Seed: 9})
	p.Eye(EyeSpec{X: 54.4, Y: 6.4, R: 0.85, Iris: plesioIris, Slit: true, Lid: 0.25, Squint: 1.3})
	p.Nostril(60.8, 6.2, 0.45, 0.3, -0.3)
	// The water, as drawn but traced smoothly: the body dark at the
	// waterline, white foam along it flecked with blue, and the ripples
	// with the light catching on their tops.
	p.Clear(plesioWater)
	isFoam := func(g color.NRGBA) bool { return lum(g) > 225 }
	isRipple := func(g color.NRGBA) bool { return lum(g) > 170 && lum(g) <= 225 && g.B > g.R }
	p.Trace(plesioWater, func(g color.NRGBA) bool { return !isFoam(g) && !isRipple(g) }, func(px, py int) color.NRGBA {
		return rgb([]uint32{0x0a1820, 0x0d1e28, 0x10242e}[int(hash(px, py, 75)*3)])
	})
	p.Trace(plesioWater, isRipple, func(px, py int) color.NRGBA {
		if !p.Inside(px, py-1) || hash(px, py, 79) < 0.05 {
			return rgb(0xc4e6f2) // the lit top of a ripple
		}
		return rgb(0x9ccce2)
	})
	p.Trace(plesioWater, isFoam, func(px, py int) color.NRGBA {
		if hash(px, py, 77) < 0.16 {
			return rgb(0xd2ecf4)
		}
		return rgb(0xf4fbfc)
	})
}
