package dinopaint

// Acheroraptor: dinos/achero.png, 58 x 28, facing right: a small raptor,
// slate blue, with a black mask through its red eye, a long tail tapering
// to a fine point, banded dark, white and dark at its end, and gold legs.
// Its rig is "achero" in internal/client/dinorigs.go: hips (40, 14) and
// (32.5, 14), knees (41.5, 18) and (33.5, 18), ankles (39, 21) and
// (31.5, 21); the legs swing whole from the hips (thigh.go); the head turns
// about (43.6, 6.8), the jaw opens about (51, 5) along the line to
// (56.5, 5.2), and the tail turns about (30, 9.4).

func init() {
	Register(achero())
	// The legs swing whole from the hip, the near thigh under the folded
	// wing and the flank (thigh.go).
	RegisterThighs("achero", &Thighs{
		Legs:       LegMask{{"nearLeg"}, {"farLeg"}},
		FrontParts: []string{"wing"},
		Front:      Poly(28, 3, 48, 3, 48, 13.6, 28, 13.6),
		Thigh:      []SDF{Cone(40.2, 15, 3, 41.4, 18, 2.3), Cone(32.6, 15, 2.9, 33.6, 18, 2.2)},
		Coat:       Plumed(acheroPlumage),
	})
}

var redIris = []uint32{0x4a0c04, 0x7a1806, 0xb22e0c, 0xe25014, 0xff7a2a, 0xffa860}

func achero() *Species {
	farLeg := Union(
		Cone(32.6, 15, 2.6, 33.6, 18, 1.9),
		Cone(33.6, 18, 1.8, 31.6, 21, 1.15),
		Cone(31.6, 21, 1.0, 32.1, 24.9, 0.85),
		Cone(32.1, 25.2, 0.85, 35.4, 25.6, 0.55),
		Cone(32.9, 24.7, 0.6, 34.3, 23.9, 0.45), // the second toe, raised
	)
	nearLeg := Union(
		Cone(40.2, 15, 2.7, 41.4, 18, 2.0),
		Cone(41.4, 18, 1.8, 39.2, 21, 1.15),
		Cone(39.2, 21, 1.0, 39.7, 24.9, 0.85),
		Cone(39.7, 25.2, 0.85, 43.4, 25.6, 0.55),
		Cone(40.5, 24.7, 0.6, 41.9, 23.9, 0.45),
	)
	body := Blend(2,
		Path([3]float64{31.5, 9.5, 3.2}, [3]float64{22.73, 8.6, 2.0}, [3]float64{9.09, 8, 0.9}, [3]float64{0.91, 7.6, 0.3}), // the long stiff tail
		Ellipse(35.5, 10.8, 8.5, 4.6, 0.1),
		Cone(42, 9, 3, 45.5, 5.5, 2.2), // the neck
		Poly(44.8, 1.2, 48.5, 0.6, 51, 1.6, 56, 2.2, 57.4, 3.6, 57.3, 5.0, 51, 5.0, 47, 7, 44.4, 5), // the head
	)
	jaw := Poly(48.8, 5, 57.2, 5.1, 56.8, 6.2, 50, 6.4, 48.4, 6)
	wing := Union( // the arm folded, its long feathers along the flank
		Cone(43.9, 10, 1.6, 45.2, 12.8, 1.2),
		Cone(45.2, 13, 0.9, 46.6, 13.7, 0.6),
		Poly(45, 10.6, 44.6, 13, 40, 13.7, 35.4, 13.9, 37, 12.3, 41, 11, 43.5, 10),
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
				Region: Poly(12, 4, 17.5, 4, 17.5, 12, 12, 12), Over: true, Only: []string{"slate", "dark", "white"}},
			{Name: "tailDark", Ramp: []uint32{0x0b0d12, 0x11141b, 0x171b23, 0x1d222b, 0x232934, 0x2a303c, 0x323946, 0x3b4352},
				Region: Union(Poly(-3, 4, 12, 4, 12, 12, -3, 12), Poly(17.5, 4, 23, 4, 23, 12, 17.5, 12)), Over: true, Only: []string{"slate", "dark", "white"}},
			{Name: "eye", Seeds: []uint32{0xff6a2a}, Ramp: redIris},
			// The black mask from the eye forward along the top of the snout.
			{Name: "mask", Ramp: []uint32{0x0a0b10, 0x0f1118, 0x14171f, 0x1a1e28, 0x212632, 0x29303d, 0x323a49, 0x3c4556},
				Region: Poly(49, 0.6, 55.6, 1.6, 55.6, 2.9, 53.8, 3.7, 51.2, 4.0, 49.4, 3.2, 48.6, 2.0)},
		},
		Recolor: []Recolor{
			{Region: Poly(29, 19.5, 36.9, 19.5, 36.9, 28, 29, 28), From: []string{"gold"}, To: "farGold"},
			{Region: Poly(44, 13.6, 48.5, 13.6, 48.5, 16, 44, 16), From: []string{"white"}, To: "gold"},
			{Region: Poly(33.4, 22, 37, 22, 37, 25, 33.4, 25), From: []string{"white"}, To: "gold"},
			{Region: Poly(41.4, 22, 45, 22, 45, 25, 41.4, 25), From: []string{"white"}, To: "gold"},
		},
		// The tail tapers smooth to its fine point, however the drawing's steps.
		Add: []SDF{Path([3]float64{22.73, 8.6, 1.8}, [3]float64{9.09, 8, 0.8}, [3]float64{0.6, 7.6, 0.2})},
		Remove: []SDF{
			Poly(44.4, 13.8, 48.4, 13.8, 48.4, 16, 44.4, 16), // the guide's hand claws
			Poly(33.6, 22.5, 37, 22.5, 37, 24.9, 33.6, 24.9), // and its sickle claws
			Poly(41.6, 22.5, 45, 22.5, 45, 24.9, 41.6, 24.9),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.2, Scale: 0.65, Muscles: []Muscle{
				Mu(33, 16, 2.2, 2.2, 0, 0.8),
			}},
			{Name: "body", Shape: body, Round: 5.5, RoundAt: func(x, y float64) float64 { return lerp(5.5, 2.2, smoothstep(43, 48, x)) }, Muscles: []Muscle{
				Mu(35.5, 10, 6, 3.5, 0, 0.7),     // the ribs
				Mu(42, 9, 2.4, 2.8, 0.3, 0.6),    // the shoulder
				Mu(48.6, 2.6, 2.2, 1.3, 0, 0.6),  // the jaw muscle behind the eye
				Mu(54.5, 3.4, 2.6, 1, 0, 0.4),    // the snout
				Mu(52.4, 2.3, 1.2, 0.55, 0, 0.5), // the brow
			}},
			{Name: "jaw", Shape: jaw, Round: 0.9, Soft: 0.5, Scale: 0.55, Mat: "pale"},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.4, Scale: 0.65, Onto: "body", Seam: Below(26, 15.8, 46, 15), Muscles: []Muscle{
				Mu(40.4, 16, 2.3, 2.3, 0, 0.9),
			}},
			{Name: "wing", Shape: wing, Round: 1.7, Soft: 0.6, Muscles: []Muscle{
				Mu(44.2, 10.8, 1.1, 1.8, 0.35, 0.7),
				Mu(45.4, 13, 1.0, 0.7, 0.4, 0.4),
			}},
		},
		Skin: Skin{Flank: 1.2, Back: 1.5, Belly: 1.0, BackDepth: 2.5, BellyDepth: 2.5, Relief: 0.8, Seed: 37},
		Folds: []Fold{
			{Pts: []float64{30.2, 21.8, 31.8, 21.8}, N: 3, DY: 0.9, Depth: 0.5},
			{Pts: []float64{37.8, 21.8, 39.4, 21.8}, N: 3, DY: 0.9, Depth: 0.5},
		},
		Details: acheroDetails,
	}
}

func acheroPlumage() *Plumage {
	return &Plumage{Seed: 7, Edge: 0.45, Tone: 0.7, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(28, 13, 36.5, 13, 36.5, 19.4, 28, 19.4), Parts: []string{"farLeg"},
			Flow: Along(32.5, 14, 33.5, 18, 31.5, 21), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "body", Region: Union(Poly(26, 3.5, 43, 3.5, 47, 6.5, 47, 10, 44, 16, 26, 17), Ellipse(45, 3.6, 1.8, 2.2, 0)),
			Parts: []string{"body", "wing"},
			Flow:  Along(46.5, 1.2, 45, 4, 42, 6.8, 36, 8, 30, 8.4, 26, 8), Len: 2.2, Wid: 1.5, Rows: 0.45, Jitter: 0.2, Fluff: 0.5, Snap: true,
			Hang: Poly(-2.27, 16, 59, 16, 59, 30, -2.27, 30)},
		{Name: "crown", Region: Poly(44, 0, 49.2, 0, 49.2, 2.4, 47, 4, 45, 6.5, 43.5, 6.5), Parts: []string{"body"},
			Flow: Along(49.5, 1.4, 47, 1.6, 45, 4, 44, 7), Len: 1.2, Wid: 0.9, Fluff: 0.35, Snap: true},
		// The tail: long and stiff, its feathers lying flat to a point.
		{Name: "tail", Region: Poly(-2.27, 4, 31.5, 4, 31.5, 13.5, -2.27, 10), Parts: []string{"body"},
			Flow: Along(33, 9.4, 22.73, 8.6, 6.82, 7.9, -2.27, 7.5), Len: 2.6, Wid: 1.15, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05},
		{Name: "nearThigh", Region: Poly(36, 13, 45, 13, 45, 19.4, 36, 19.4), Parts: []string{"nearLeg"},
			Flow: Along(40, 14, 41.5, 18, 39, 21), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "wing", Region: Poly(44.5, 10.2, 44.8, 13.2, 40, 13.8, 35.2, 14.2, 36, 12.4, 40, 11, 43, 10), Parts: []string{"wing"},
			Flow: Along(45, 11.2, 42.5, 12.3, 39, 13, 35.5, 13.5), Len: 3.6, Wid: 1.1, Rows: 0.34, Tip: 0.55, Shaft: 0.1, Fluff: 0.3,
			TipMat: "white", TipLen: 0.3, TipGuide: true},
		{Name: "arm", Region: Poly(43, 9, 46.5, 10, 46.8, 13.5, 44.5, 13.8, 43.4, 11.5), Parts: []string{"wing"},
			Flow: Along(43.6, 9.2, 44.8, 11.6, 45.8, 13.6), Len: 1.7, Wid: 1.2, Fluff: 0.3, Snap: true},
	}}
}

func acheroDetails(p *Painter) {
	p.Plumage(acheroPlumage())
	p.Teeth(TeethSpec{X0: 50.4, Y0: 4.4, X1: 56.8, Y1: 4.45, Bottom: 5.02, N: 6, Len: 0.62,
		Ramp: raptorTeeth, Gap: 0x1c0e10, Seed: 13})
	p.Lip([]float64{50.2, 4.42, 53.5, 4.42, 57, 4.47}, 6)
	p.Nostril(56.2, 3.1, 0.42, 0.26, -0.3)
	p.Eye(EyeSpec{X: 52.5, Y: 3.25, R: 0.72, Iris: redIris, Slit: true, Lid: 0.22, Squint: 1.25})
	// The hand's claws.
	p.Claw(45.1, 13.4, 45.3, 15.0, 0.7, 0.3, sickleHorn[:6])
	p.Claw(46.6, 13.5, 47.2, 14.9, 0.7, 0.3, sickleHorn[:6])
	// The feet: a claw on the front toe, and the sickle claw held up.
	p.Claw(35, 25.3, 36.5, 25.9, 0.6, 0.15, sickleHorn[:5])
	p.Claw(33, 23.8, 35.8, 24.3, 1.0, -0.75, sickleHorn)
	p.Claw(42.9, 25.3, 44.6, 25.9, 0.65, 0.15, sickleHorn[:6])
	p.Claw(40.7, 23.8, 43.7, 24.3, 1.05, -0.75, sickleHorn)
}
