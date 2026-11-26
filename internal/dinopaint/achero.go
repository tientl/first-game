package dinopaint

// Acheroraptor: dinos/achero.png, 38 x 28, facing right: a small raptor,
// slate blue, with a black mask through its red eye, a white-tipped tail
// and gold legs. Its rig is "achero" in internal/client/dinorigs.go: hips
// (20, 14) and (12.5, 14), knees (21.5, 18) and (13.5, 18), ankles (19, 21)
// and (11.5, 21); the legs are cut from row 17, columns 17-24 (near) and
// 9-16 (far); the head turns about (23.6, 6.8), the jaw opens about (31, 5)
// along the line to (36.5, 5.2), and the tail turns about (11, 8).

func init() { Register(achero()) }

var redIris = []uint32{0x4a0c04, 0x7a1806, 0xb22e0c, 0xe25014, 0xff7a2a, 0xffa860}

func achero() *Species {
	farLeg := Union(
		Cone(12.6, 15, 2.6, 13.6, 18, 1.9),
		Cone(13.6, 18, 1.8, 11.6, 21, 1.15),
		Cone(11.6, 21, 1.0, 12.1, 24.9, 0.85),
		Cone(12.1, 25.2, 0.85, 15.4, 25.6, 0.55),
		Cone(12.9, 24.7, 0.6, 14.3, 23.9, 0.45), // the second toe, raised
	)
	nearLeg := Union(
		Cone(20.2, 15, 2.7, 21.4, 18, 2.0),
		Cone(21.4, 18, 1.8, 19.2, 21, 1.15),
		Cone(19.2, 21, 1.0, 19.7, 24.9, 0.85),
		Cone(19.7, 25.2, 0.85, 23.4, 25.6, 0.55),
		Cone(20.5, 24.7, 0.6, 21.9, 23.9, 0.45),
	)
	body := Blend(2,
		Path([3]float64{11, 9.4, 3.6}, [3]float64{5, 7.5, 3}, [3]float64{0.5, 6.5, 3.3}), // the tail
		Ellipse(15.5, 10.8, 8.5, 4.6, 0.1),
		Cone(22, 9, 3, 25.5, 5.5, 2.2),                                                              // the neck
		Poly(24.8, 1.2, 28.5, 0.6, 31, 1.6, 36, 2.2, 37.4, 3.6, 37.3, 5.0, 31, 5.0, 27, 7, 24.4, 5), // the head
	)
	jaw := Poly(28.8, 5, 37.2, 5.1, 36.8, 6.2, 30, 6.4, 28.4, 6)
	wing := Union( // the arm folded, its long feathers along the flank
		Cone(23.9, 10, 1.6, 25.2, 12.8, 1.2),
		Cone(25.2, 13, 0.9, 26.6, 13.7, 0.6),
		Poly(25, 10.6, 24.6, 13, 20, 13.7, 15.4, 13.9, 17, 12.3, 21, 11, 23.5, 10),
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
			{Name: "eye", Seeds: []uint32{0xff6a2a}, Ramp: redIris},
			// The black mask from the eye forward along the top of the snout.
			{Name: "mask", Ramp: []uint32{0x0a0b10, 0x0f1118, 0x14171f, 0x1a1e28, 0x212632, 0x29303d, 0x323a49, 0x3c4556},
				Region: Poly(29, 0.6, 35.6, 1.6, 35.6, 2.9, 33.8, 3.7, 31.2, 4.0, 29.4, 3.2, 28.6, 2.0)},
		},
		Recolor: []Recolor{
			{Region: Poly(24, 13.6, 28.5, 13.6, 28.5, 16, 24, 16), From: []string{"white"}, To: "gold"},
			{Region: Poly(13.4, 22, 17, 22, 17, 25, 13.4, 25), From: []string{"white"}, To: "gold"},
			{Region: Poly(21.4, 22, 25, 22, 25, 25, 21.4, 25), From: []string{"white"}, To: "gold"},
			// The tail's dark band, the drawing's darkest blue there.
			{Region: Poly(1.8, 3, 4.9, 3, 4.9, 5.9, 1.8, 5.9), From: []string{"slate"}, To: "dark"},
			{Region: Poly(1.8, 6.9, 4.9, 6.9, 4.9, 9.5, 1.8, 9.5), From: []string{"slate"}, To: "dark"},
		},
		Remove: []SDF{
			Poly(24.4, 13.8, 28.4, 13.8, 28.4, 16, 24.4, 16), // the guide's hand claws
			Poly(13.6, 22.5, 17, 22.5, 17, 24.9, 13.6, 24.9), // and its sickle claws
			Poly(21.6, 22.5, 25, 22.5, 25, 24.9, 21.6, 24.9),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.2, Scale: 0.65, Muscles: []Muscle{
				Mu(13, 16, 2.2, 2.2, 0, 0.8),
			}},
			{Name: "body", Shape: body, Round: 5.5, RoundAt: func(x, y float64) float64 { return lerp(5.5, 2.2, smoothstep(23, 28, x)) }, Muscles: []Muscle{
				Mu(15.5, 10, 6, 3.5, 0, 0.7),     // the ribs
				Mu(22, 9, 2.4, 2.8, 0.3, 0.6),    // the shoulder
				Mu(28.6, 2.6, 2.2, 1.3, 0, 0.6),  // the jaw muscle behind the eye
				Mu(34.5, 3.4, 2.6, 1, 0, 0.4),    // the snout
				Mu(32.4, 2.3, 1.2, 0.55, 0, 0.5), // the brow
			}},
			{Name: "jaw", Shape: jaw, Round: 0.9, Soft: 0.5, Scale: 0.55, Mat: "pale"},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.4, Scale: 0.65, Onto: "body", Seam: Below(6, 15.8, 26, 15), Muscles: []Muscle{
				Mu(20.4, 16, 2.3, 2.3, 0, 0.9),
			}},
			{Name: "wing", Shape: wing, Round: 1.7, Soft: 0.6, Muscles: []Muscle{
				Mu(24.2, 10.8, 1.1, 1.8, 0.35, 0.7),
				Mu(25.4, 13, 1.0, 0.7, 0.4, 0.4),
			}},
		},
		Skin: Skin{Flank: 1.2, Back: 1.5, Belly: 1.0, BackDepth: 2.5, BellyDepth: 2.5, Relief: 0.8, Seed: 37},
		Folds: []Fold{
			{Pts: []float64{10.2, 21.8, 11.8, 21.8}, N: 3, DY: 0.9, Depth: 0.5},
			{Pts: []float64{17.8, 21.8, 19.4, 21.8}, N: 3, DY: 0.9, Depth: 0.5},
		},
		Details: acheroDetails,
	}
}

func acheroPlumage() *Plumage {
	return &Plumage{Seed: 7, Edge: 0.45, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(8, 13, 16.5, 13, 16.5, 20.4, 8, 20.4), Parts: []string{"farLeg"},
			Flow: Along(12.5, 14, 13.5, 18, 11.5, 21), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "body", Region: Union(Poly(6, 3.5, 23, 3.5, 27, 6.5, 27, 10, 24, 16, 6, 17), Ellipse(25, 3.6, 1.8, 2.2, 0)),
			Parts: []string{"body", "wing"},
			Flow:  Along(26.5, 1.2, 25, 4, 22, 6.8, 16, 8, 10, 8.4, 6, 8), Len: 2.2, Wid: 1.5, Rows: 0.45, Jitter: 0.2, Fluff: 0.5, Snap: true},
		{Name: "crown", Region: Poly(24, 0, 29.2, 0, 29.2, 2.4, 27, 4, 25, 6.5, 23.5, 6.5), Parts: []string{"body"},
			Flow: Along(29.5, 1.4, 27, 1.6, 25, 4, 24, 7), Len: 1.2, Wid: 0.9, Fluff: 0.35, Snap: true},
		{Name: "tail", Region: Poly(-1, 2, 8, 3.2, 9.5, 6.5, 8, 11, -1, 11), Parts: []string{"body"},
			Flow: Fan(13, 7, 7, 3.1416), Len: 3.6, Wid: 1.25, Rows: 0.34, Tip: 0.5, Shaft: 0.12, Fluff: 0.6, Grow: 0.02},
		{Name: "nearThigh", Region: Poly(16, 13, 25, 13, 25, 20.4, 16, 20.4), Parts: []string{"nearLeg"},
			Flow: Along(20, 14, 21.5, 18, 19, 21), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "wing", Region: Poly(24.5, 10.2, 24.8, 13.2, 20, 13.8, 15.2, 14.2, 16, 12.4, 20, 11, 23, 10), Parts: []string{"wing"},
			Flow: Along(25, 11.2, 22.5, 12.3, 19, 13, 15.5, 13.5), Len: 3.6, Wid: 1.1, Rows: 0.34, Tip: 0.55, Shaft: 0.1, Fluff: 0.3,
			TipMat: "white", TipLen: 0.3, TipGuide: true},
		{Name: "arm", Region: Poly(23, 9, 26.5, 10, 26.8, 13.5, 24.5, 13.8, 23.4, 11.5), Parts: []string{"wing"},
			Flow: Along(23.6, 9.2, 24.8, 11.6, 25.8, 13.6), Len: 1.7, Wid: 1.2, Fluff: 0.3, Snap: true},
	}}
}

func acheroDetails(p *Painter) {
	p.Plumage(acheroPlumage())
	p.Teeth(TeethSpec{X0: 30.4, Y0: 4.4, X1: 36.8, Y1: 4.45, Bottom: 5.02, N: 6, Len: 0.62,
		Ramp: raptorTeeth, Gap: 0x1c0e10, Seed: 13})
	p.Lip([]float64{30.2, 4.42, 33.5, 4.42, 37, 4.47}, 6)
	p.Nostril(36.2, 3.1, 0.42, 0.26, -0.3)
	p.Eye(EyeSpec{X: 32.5, Y: 3.25, R: 0.72, Iris: redIris, Slit: true, Lid: 0.22, Squint: 1.25})
	// The hand's claws.
	p.Claw(25.1, 13.4, 25.3, 15.0, 0.7, 0.3, sickleHorn[:6])
	p.Claw(26.6, 13.5, 27.2, 14.9, 0.7, 0.3, sickleHorn[:6])
	// The feet: a claw on the front toe, and the sickle claw held up.
	p.Claw(15.0, 25.3, 16.5, 25.9, 0.6, 0.15, sickleHorn[:5])
	p.Claw(13.0, 23.8, 15.8, 24.3, 1.0, -0.75, sickleHorn)
	p.Claw(22.9, 25.3, 24.6, 25.9, 0.65, 0.15, sickleHorn[:6])
	p.Claw(20.7, 23.8, 23.7, 24.3, 1.05, -0.75, sickleHorn)
}
