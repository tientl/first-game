package dinopaint

// Acheroraptor: dinos/achero.png, 44 x 28, facing right: a small raptor,
// slate blue, with a black mask through its red eye, a long pointed tail
// banded dark, white and dark at its end, and gold legs. Its rig is "achero"
// in internal/client/dinorigs.go: hips (26, 14) and (18.5, 14), knees
// (27.5, 18) and (19.5, 18), ankles (25, 21) and (17.5, 21); the legs are cut
// from row 17, columns 23-30 (near) and 15-22 (far); the head turns about
// (29.6, 6.8), the jaw opens about (37, 5) along the line to (42.5, 5.2), and
// the tail turns about (16, 9.4).

func init() {
	Register(achero())
	// The legs swing whole from the hip, the near thigh under the folded
	// wing and the flank (thigh.go).
	RegisterThighs("achero", &Thighs{
		Legs:       LegMask{{"nearLeg"}, {"farLeg"}},
		FrontParts: []string{"wing"},
		Front:      Poly(14, 3, 34, 3, 34, 13.6, 14, 13.6),
		Thigh:      []SDF{Cone(26.2, 15, 3, 27.4, 18, 2.3), Cone(18.6, 15, 2.9, 19.6, 18, 2.2)},
		Coat:       Plumed(acheroPlumage),
	})
}

var redIris = []uint32{0x4a0c04, 0x7a1806, 0xb22e0c, 0xe25014, 0xff7a2a, 0xffa860}

func achero() *Species {
	farLeg := Union(
		Cone(18.6, 15, 2.6, 19.6, 18, 1.9),
		Cone(19.6, 18, 1.8, 17.6, 21, 1.15),
		Cone(17.6, 21, 1.0, 18.1, 24.9, 0.85),
		Cone(18.1, 25.2, 0.85, 21.4, 25.6, 0.55),
		Cone(18.9, 24.7, 0.6, 20.3, 23.9, 0.45), // the second toe, raised
	)
	nearLeg := Union(
		Cone(26.2, 15, 2.7, 27.4, 18, 2.0),
		Cone(27.4, 18, 1.8, 25.2, 21, 1.15),
		Cone(25.2, 21, 1.0, 25.7, 24.9, 0.85),
		Cone(25.7, 25.2, 0.85, 29.4, 25.6, 0.55),
		Cone(26.5, 24.7, 0.6, 27.9, 23.9, 0.45),
	)
	body := Blend(2,
		Path([3]float64{17.5, 9.5, 3.2}, [3]float64{10, 8.6, 2.0}, [3]float64{4, 8, 0.9}, [3]float64{0.4, 7.6, 0.3}), // the long stiff tail
		Ellipse(21.5, 10.8, 8.5, 4.6, 0.1),
		Cone(28, 9, 3, 31.5, 5.5, 2.2), // the neck
		Poly(30.8, 1.2, 34.5, 0.6, 37, 1.6, 42, 2.2, 43.4, 3.6, 43.3, 5.0, 37, 5.0, 33, 7, 30.4, 5), // the head
	)
	jaw := Poly(34.8, 5, 43.2, 5.1, 42.8, 6.2, 36, 6.4, 34.4, 6)
	wing := Union( // the arm folded, its long feathers along the flank
		Cone(29.9, 10, 1.6, 31.2, 12.8, 1.2),
		Cone(31.2, 13, 0.9, 32.6, 13.7, 0.6),
		Poly(31, 10.6, 30.6, 13, 26, 13.7, 21.4, 13.9, 23, 12.3, 27, 11, 29.5, 10),
	)
	return &Species{
		Name:        "achero",
		GuideDetail: 0.3,
		Ink:         30,
		Materials: []Material{
			{Name: "slate", Seeds: []uint32{0x7b8c9e, 0x5a6a7e, 0x3f4c5e, 0xa2b1bf, 0x28313f},
				Ramp: []uint32{0x13151d, 0x1d212d, 0x282f3f, 0x363f52, 0x475268, 0x5b687f, 0x76869b, 0x98a6b4}},
			{Name: "dark", Seeds: []uint32{0x262b36},
				Ramp: []uint32{0x0b0d12, 0x11141b, 0x171b23, 0x1d222b, 0x232934, 0x2a303c, 0x323946, 0x3b4352}, Bias: 0.05},
			{Name: "pale", Seeds: []uint32{0xaeaa9c, 0xd6d3c6, 0x7f7a70},
				Ramp: []uint32{0x38383c, 0x52514e, 0x6c6a63, 0x87847a, 0xa09c90, 0xb6b2a6, 0xcac7bc, 0xdedcd2}},
			{Name: "gold", Seeds: []uint32{0xbc922c, 0x8c6a20, 0xe2b844, 0x5a4416},
				Ramp: []uint32{0x2a1c08, 0x3e2a0c, 0x5a3e10, 0x7a5616, 0x9a6e1c, 0xb88a26, 0xd2a634, 0xe8c454}},
			{Name: "white", Seeds: []uint32{0xf2f1ec, 0xf2f0e6, 0xe8e2d0, 0xc8c6c0},
				Ramp: []uint32{0x4c4e56, 0x6c6e76, 0x909298, 0xb2b2b2, 0xccccc6, 0xe0dfd8, 0xefeee6, 0xfbfaf6}, Bias: 0.1},
			{Name: "farGold", Ramp: []uint32{0x1e1508, 0x2c1f0a, 0x3e2c0e, 0x543c12, 0x6a4c16, 0x82601c, 0x9a7424, 0xb08a30}}, // the far shank, in shadow
			// The end of the tail: dark, white, and the dark point.
			{Name: "tailWhite", Ramp: []uint32{0x4c4e56, 0x6c6e76, 0x909298, 0xb2b2b2, 0xccccc6, 0xe0dfd8, 0xefeee6, 0xfbfaf6},
				Region: Poly(5.9, 4, 8.9, 4, 8.9, 12, 5.9, 12), Over: true, Only: []string{"slate", "dark", "white"}},
			{Name: "tailDark", Ramp: []uint32{0x0b0d12, 0x11141b, 0x171b23, 0x1d222b, 0x232934, 0x2a303c, 0x323946, 0x3b4352},
				Region: Union(Poly(-1, 4, 5.9, 4, 5.9, 12, -1, 12), Poly(8.9, 4, 11.9, 4, 11.9, 12, 8.9, 12)), Over: true, Only: []string{"slate", "dark", "white"}},
			{Name: "eye", Seeds: []uint32{0xff6a2a}, Ramp: redIris},
			// The black mask from the eye forward along the top of the snout.
			{Name: "mask", Ramp: []uint32{0x0a0b10, 0x0f1118, 0x14171f, 0x1a1e28, 0x212632, 0x29303d, 0x323a49, 0x3c4556},
				Region: Poly(35, 0.6, 41.6, 1.6, 41.6, 2.9, 39.8, 3.7, 37.2, 4.0, 35.4, 3.2, 34.6, 2.0)},
		},
		Recolor: []Recolor{
			{Region: Poly(15, 19.5, 22.9, 19.5, 22.9, 28, 15, 28), From: []string{"gold"}, To: "farGold"},
			{Region: Poly(30, 13.6, 34.5, 13.6, 34.5, 16, 30, 16), From: []string{"white"}, To: "gold"},
			{Region: Poly(19.4, 22, 23, 22, 23, 25, 19.4, 25), From: []string{"white"}, To: "gold"},
			{Region: Poly(27.4, 22, 31, 22, 31, 25, 27.4, 25), From: []string{"white"}, To: "gold"},
		},
		Remove: []SDF{
			Poly(30.4, 13.8, 34.4, 13.8, 34.4, 16, 30.4, 16), // the guide's hand claws
			Poly(19.6, 22.5, 23, 22.5, 23, 24.9, 19.6, 24.9), // and its sickle claws
			Poly(27.6, 22.5, 31, 22.5, 31, 24.9, 27.6, 24.9),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.2, Scale: 0.65, Muscles: []Muscle{
				Mu(19, 16, 2.2, 2.2, 0, 0.8),
			}},
			{Name: "body", Shape: body, Round: 5.5, RoundAt: func(x, y float64) float64 { return lerp(5.5, 2.2, smoothstep(29, 34, x)) }, Muscles: []Muscle{
				Mu(21.5, 10, 6, 3.5, 0, 0.7),     // the ribs
				Mu(28, 9, 2.4, 2.8, 0.3, 0.6),    // the shoulder
				Mu(34.6, 2.6, 2.2, 1.3, 0, 0.6),  // the jaw muscle behind the eye
				Mu(40.5, 3.4, 2.6, 1, 0, 0.4),    // the snout
				Mu(38.4, 2.3, 1.2, 0.55, 0, 0.5), // the brow
			}},
			{Name: "jaw", Shape: jaw, Round: 0.9, Soft: 0.5, Scale: 0.55, Mat: "pale"},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.4, Scale: 0.65, Onto: "body", Seam: Below(12, 15.8, 32, 15), Muscles: []Muscle{
				Mu(26.4, 16, 2.3, 2.3, 0, 0.9),
			}},
			{Name: "wing", Shape: wing, Round: 1.7, Soft: 0.6, Muscles: []Muscle{
				Mu(30.2, 10.8, 1.1, 1.8, 0.35, 0.7),
				Mu(31.4, 13, 1.0, 0.7, 0.4, 0.4),
			}},
		},
		Skin: Skin{Flank: 1.2, Back: 1.5, Belly: 1.0, BackDepth: 2.5, BellyDepth: 2.5, Relief: 0.8, Seed: 37},
		Folds: []Fold{
			{Pts: []float64{16.2, 21.8, 17.8, 21.8}, N: 3, DY: 0.9, Depth: 0.5},
			{Pts: []float64{23.8, 21.8, 25.4, 21.8}, N: 3, DY: 0.9, Depth: 0.5},
		},
		Details: acheroDetails,
	}
}

func acheroPlumage() *Plumage {
	return &Plumage{Seed: 7, Edge: 0.45, Tone: 0.7, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(14, 13, 22.5, 13, 22.5, 19.4, 14, 19.4), Parts: []string{"farLeg"},
			Flow: Along(18.5, 14, 19.5, 18, 17.5, 21), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "body", Region: Union(Poly(12, 3.5, 29, 3.5, 33, 6.5, 33, 10, 30, 16, 12, 17), Ellipse(31, 3.6, 1.8, 2.2, 0)),
			Parts: []string{"body", "wing"},
			Flow:  Along(32.5, 1.2, 31, 4, 28, 6.8, 22, 8, 16, 8.4, 12, 8), Len: 2.2, Wid: 1.5, Rows: 0.45, Jitter: 0.2, Fluff: 0.5, Snap: true,
			Hang: Poly(-1, 16, 45, 16, 45, 30, -1, 30)},
		{Name: "crown", Region: Poly(30, 0, 35.2, 0, 35.2, 2.4, 33, 4, 31, 6.5, 29.5, 6.5), Parts: []string{"body"},
			Flow: Along(35.5, 1.4, 33, 1.6, 31, 4, 30, 7), Len: 1.2, Wid: 0.9, Fluff: 0.35, Snap: true},
		// The tail: long and stiff, its feathers lying flat to a point.
		{Name: "tail", Region: Poly(-1, 4, 17.5, 4, 17.5, 13.5, -1, 10), Parts: []string{"body"},
			Flow: Along(19, 9.4, 10, 8.6, 3, 7.9, -1, 7.5), Len: 2.6, Wid: 1.15, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05},
		{Name: "nearThigh", Region: Poly(22, 13, 31, 13, 31, 19.4, 22, 19.4), Parts: []string{"nearLeg"},
			Flow: Along(26, 14, 27.5, 18, 25, 21), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "wing", Region: Poly(30.5, 10.2, 30.8, 13.2, 26, 13.8, 21.2, 14.2, 22, 12.4, 26, 11, 29, 10), Parts: []string{"wing"},
			Flow: Along(31, 11.2, 28.5, 12.3, 25, 13, 21.5, 13.5), Len: 3.6, Wid: 1.1, Rows: 0.34, Tip: 0.55, Shaft: 0.1, Fluff: 0.3,
			TipMat: "white", TipLen: 0.3, TipGuide: true},
		{Name: "arm", Region: Poly(29, 9, 32.5, 10, 32.8, 13.5, 30.5, 13.8, 29.4, 11.5), Parts: []string{"wing"},
			Flow: Along(29.6, 9.2, 30.8, 11.6, 31.8, 13.6), Len: 1.7, Wid: 1.2, Fluff: 0.3, Snap: true},
	}}
}

func acheroDetails(p *Painter) {
	p.Plumage(acheroPlumage())
	p.Teeth(TeethSpec{X0: 36.4, Y0: 4.4, X1: 42.8, Y1: 4.45, Bottom: 5.02, N: 6, Len: 0.62,
		Ramp: raptorTeeth, Gap: 0x1c0e10, Seed: 13})
	p.Lip([]float64{36.2, 4.42, 39.5, 4.42, 43, 4.47}, 6)
	p.Nostril(42.2, 3.1, 0.42, 0.26, -0.3)
	p.Eye(EyeSpec{X: 38.5, Y: 3.25, R: 0.72, Iris: redIris, Slit: true, Lid: 0.22, Squint: 1.25})
	// The hand's claws.
	p.Claw(31.1, 13.4, 31.3, 15.0, 0.7, 0.3, sickleHorn[:6])
	p.Claw(32.6, 13.5, 33.2, 14.9, 0.7, 0.3, sickleHorn[:6])
	// The feet: a claw on the front toe, and the sickle claw held up.
	p.Claw(21, 25.3, 22.5, 25.9, 0.6, 0.15, sickleHorn[:5])
	p.Claw(19, 23.8, 21.8, 24.3, 1.0, -0.75, sickleHorn)
	p.Claw(28.9, 25.3, 30.6, 25.9, 0.65, 0.15, sickleHorn[:6])
	p.Claw(26.7, 23.8, 29.7, 24.3, 1.05, -0.75, sickleHorn)
}
