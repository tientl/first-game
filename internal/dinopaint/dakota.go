package dinopaint

// Dakotaraptor: dinos/dakota.png, 60 x 37, facing right, its long stiff tail
// held out straight behind to a point. Its rig is "dakota" in
// internal/client/dinorigs.go: hips (35, 19) and (25, 19), knees (37.5, 24)
// and (27.5, 24), ankles (34.5, 28.5) and (24.5, 28.5); the legs are cut from
// row 23, columns 32-41 (near) and 22-31 (far); the head turns about
// (41, 10), the jaw opens about (50.5, 8) along the line to (58, 8.4), and
// the tail turns about (22.5, 12.8).

// The colours shared by the feathered hunters.
var (
	sickleHorn  = []uint32{0x1a1411, 0x352b24, 0x5a4c40, 0x857664, 0xb2a48c, 0xdcd2bc, 0xf6f0e0}
	raptorIris  = []uint32{0x5a2a04, 0x8e4a08, 0xc87a12, 0xeeaa26, 0xffd04c, 0xffe890}
	raptorTeeth = []uint32{0x6e5e4a, 0x9c8c70, 0xc8ba98, 0xe8dec2, 0xfaf6e6}
)

func init() { Register(dakota()) }

func dakota() *Species {
	farLeg := Union(
		Cone(25.4, 21, 3.4, 27.2, 24, 2.6),     // the thigh
		Cone(27.2, 24, 2.4, 24.7, 28.5, 1.5),   // the shank
		Cone(24.7, 28.5, 1.3, 25.3, 33.6, 1.1), // the metatarsus
		Cone(25.3, 33.9, 1.1, 29.6, 34.4, 0.7), // the toes
		Cone(26.4, 33.3, 0.8, 28.4, 32.5, 0.6), // the second toe, raised
	)
	nearLeg := Union(
		Cone(35.6, 21.2, 3.5, 37.3, 24, 2.7),
		Cone(37.3, 24, 2.5, 34.7, 28.5, 1.5),
		Cone(34.7, 28.5, 1.3, 35.3, 33.6, 1.1),
		Cone(35.3, 33.9, 1.1, 39.6, 34.4, 0.7),
		Cone(36.4, 33.3, 0.8, 38.4, 32.5, 0.6),
	)
	body := Blend(2.5,
		Path([3]float64{25, 13, 3.8}, [3]float64{15, 12.1, 2.6}, [3]float64{6, 11.2, 1.2}, [3]float64{0.4, 10.7, 0.3}), // the long stiff tail
		Ellipse(31, 15.6, 11.5, 6.2, 0.06),                                           // the body
		Cone(38.5, 13.5, 4.6, 45, 7.5, 2.7),                                          // the neck
		Poly(43.5, 2.5, 52, 2.5, 56, 4, 58.5, 5.6, 59.6, 7.6, 50, 8, 45, 9.5, 43, 6), // the head
	)
	jaw := Poly(45.5, 8, 59.4, 8.1, 59.2, 9.6, 47, 9.9, 45, 9.6)
	wing := Union( // the arm folded, its long feathers along the flank
		Cone(41.6, 14.4, 2.3, 43.7, 18.3, 1.6),
		Cone(43.7, 18.5, 1.2, 45.4, 19.5, 0.8),
		Poly(44, 15.6, 43.3, 18.6, 38, 19.7, 30.2, 20.1, 32.5, 18.1, 37.5, 15.7, 41, 14.3),
	)
	return &Species{
		Name:        "dakota",
		GuideDetail: 0.3,
		Ink:         27,
		Materials: []Material{
			{Name: "rufous", Seeds: []uint32{0x4a1e12, 0x6e2e18, 0x924020, 0xb2562a, 0xcc7038, 0xb85a28},
				Ramp: []uint32{0x260e0a, 0x3a160e, 0x521e12, 0x6c2818, 0x86341e, 0x9e4226, 0xb4522e, 0xc86438}},
			{Name: "charcoal", Seeds: []uint32{0x22262e, 0x2e343e, 0x3c4450, 0x4c5664, 0x5e6878},
				Ramp: []uint32{0x121418, 0x1a1e24, 0x232830, 0x2e343e, 0x3a424e, 0x48525f, 0x586472, 0x6a7886}},
			{Name: "edge", Ramp: []uint32{0x3a424e, 0x4c5664, 0x5e6a78, 0x72808e, 0x8694a2, 0x9aa8b4, 0xaebac4, 0xc2ccd4}}, // the pale edges of the flank's feathers
			{Name: "white", Seeds: []uint32{0x9a9ca0, 0xb8babc, 0xd0d0cc, 0xe4e2da, 0xf4f2ea},
				Ramp: []uint32{0x8a8c90, 0xa2a4a6, 0xb8babb, 0xcacbc8, 0xd8d8d2, 0xe4e2da, 0xeeece4, 0xfaf8f0}},
			{Name: "belly", Seeds: []uint32{0x7c8088, 0x9498a0, 0xacb0b6, 0xc2c6ca},
				Ramp: []uint32{0x3e4248, 0x52565e, 0x686c74, 0x7e838a, 0x949aa0, 0xaab0b6, 0xbec4c8, 0xd2d6da}},
			{Name: "snout", Seeds: []uint32{0x3e4a58, 0x52606e, 0x687888, 0x7e90a0},
				Ramp: []uint32{0x1e2630, 0x28323e, 0x343f4c, 0x424e5c, 0x52606e, 0x647282, 0x768696, 0x8a9aaa}, Relief: 1.3},
			{Name: "rust", Seeds: []uint32{0x5a2410, 0x7e3618, 0xa04a1e, 0xbe6428, 0xd47e34},
				Ramp: []uint32{0x3a1a0a, 0x5a2a0e, 0x7a3c12, 0x9a5018, 0xb8661e, 0xd07e26, 0xe29634, 0xf0ae4a}},
			{Name: "tipDark", Seeds: []uint32{0x2a1a14, 0x2a1f1a},
				Ramp: []uint32{0x0e0a08, 0x16100c, 0x1e1610, 0x281e16, 0x32261c, 0x3e3022, 0x4a3a2a, 0x564432}},
			{Name: "thigh", Seeds: []uint32{0x40444a, 0x565a62, 0x6e737a, 0x868b92},
				Ramp: []uint32{0x24262a, 0x303338, 0x3e4248, 0x4e5258, 0x60646a, 0x72767e, 0x868a92, 0x9a9ea6}},
			{Name: "feet", Seeds: []uint32{0x34363a, 0x46494e, 0x5a5e64, 0x6e7278},
				Ramp: []uint32{0x1a1b1e, 0x242629, 0x2e3034, 0x3a3c40, 0x46494e, 0x54585e, 0x64686e, 0x76797e}},
			{Name: "ivory", Seeds: []uint32{0xf2ecdc, 0xf4e8c2, 0xd0c8b4, 0xd8ccb0, 0xdcc696, 0xf6cf36},
				Ramp: []uint32{0x4e4440, 0x6e6458, 0x948a7a, 0xb4aa96, 0xcec6b0, 0xe2dcc8, 0xf0ecdc, 0xfffcf0}},
			{Name: "farFeet", Ramp: []uint32{0x101112, 0x18191b, 0x202124, 0x292b2e, 0x333538, 0x3e4044, 0x4a4d52, 0x575a60}}, // the far foot, in shadow
			// The dark streak through the eye and back from it.
			{Name: "streak", Ramp: []uint32{0x0e0e12, 0x141418, 0x1a1a20, 0x222228, 0x2a2a32, 0x34343c, 0x3e3e48, 0x4a4a54},
				Region: Poly(43.6, 3.7, 52.8, 3.5, 54.2, 4.6, 52.8, 5.9, 47, 6.2, 43.6, 6.0), Over: true, Only: []string{"white", "rufous", "snout", "charcoal"}},
			// The tail ends in a narrow vane banded dark, to a dark point.
			{Name: "tailBand", Ramp: []uint32{0x0e0a08, 0x16100c, 0x1e1610, 0x281e16, 0x32261c, 0x3e3022, 0x4a3a2a, 0x564432},
				Region: Union(stripes(4, 12.5, 6, 15, 2.6, 0.95, 0), Poly(-1, 6, 4, 6, 4, 15, -1, 15)), Over: true, Only: []string{"rufous", "tipDark"}},
		},
		Recolor: []Recolor{
			{Region: Poly(21.5, 27, 32, 27, 32, 37, 21.5, 37), From: []string{"feet"}, To: "farFeet"},
			// The guide's claws are painted again.
			{Region: Poly(42, 19.5, 47, 19.5, 47, 22, 42, 22), From: []string{"ivory"}, To: "rust"},
			{Region: Poly(27.5, 30, 32, 30, 32, 33.5, 27.5, 33.5), From: []string{"ivory"}, To: "feet"},
			{Region: Poly(37.5, 30, 42, 30, 42, 33.5, 37.5, 33.5), From: []string{"ivory"}, To: "feet"},
			{Region: Poly(20, 14, 44, 14, 44, 21, 20, 21), From: []string{"ivory"}, To: "tipDark"},
		},
		Remove: []SDF{
			Poly(42.5, 19.6, 47, 19.6, 47, 22, 42.5, 22), // the guide's hand claws
			Poly(28.6, 30, 32, 30, 32, 33, 28.6, 33),     // and its sickle claws
			Poly(38.6, 30, 42, 30, 42, 33, 38.6, 33),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.6, Scale: 0.7, Muscles: []Muscle{
				Mu(26.2, 22, 2.8, 2.8, 0, 0.9),
				Mu(26, 25.5, 1.4, 1.6, 0.5, 0.4),
			}},
			{Name: "body", Shape: body, Round: 7, RoundAt: func(x, y float64) float64 { return lerp(7.5, 2.8, smoothstep(40, 47, x)) }, Muscles: []Muscle{
				Mu(31, 14.5, 8, 4.5, 0, 0.8),     // the ribs
				Mu(39.5, 13.5, 3, 3.5, 0.3, 0.7), // the shoulder
				Mu(21, 11.5, 4.5, 2.5, 0, 0.7),   // the root of the tail
				Mu(48.5, 4.6, 3, 1.6, 0, 0.7),    // the jaw muscle behind the eye
				Mu(55, 5.6, 3.5, 1.3, 0, 0.5),    // the snout
				Mu(52.6, 3.6, 1.6, 0.7, 0, 0.6),  // the brow over the eye
			}},
			{Name: "jaw", Shape: jaw, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "snout"},
			{Name: "nearLeg", Shape: nearLeg, Round: 3, Scale: 0.7, Onto: "body", Seam: Below(20, 21.5, 44, 20.5), Muscles: []Muscle{
				Mu(36, 22, 3, 3, 0, 1.0),
				Mu(36, 25.5, 1.5, 1.6, 0.5, 0.4),
			}},
			{Name: "wing", Shape: wing, Round: 2.2, Soft: 0.6, Muscles: []Muscle{
				Mu(42, 15.5, 1.5, 2.4, 0.35, 0.9), // the upper arm
				Mu(44, 18.6, 1.4, 1, 0.4, 0.5),    // the forearm
			}},
		},
		Skin: Skin{Flank: 1.4, Back: 1.8, Belly: 1.2, BackDepth: 3, BellyDepth: 3, Relief: 0.8, Seed: 31},
		Folds: []Fold{
			{Pts: []float64{24, 30.2, 26.2, 30.2}, N: 3, DY: 1.1, Depth: 0.6}, // across the metatarsus
			{Pts: []float64{34, 30.2, 36.2, 30.2}, N: 3, DY: 1.1, Depth: 0.6},
		},
		Details: dakotaDetails,
	}
}

func dakotaPlumage() *Plumage {
	return &Plumage{Seed: 5, Edge: 0.45, Tone: 0.6, Tracts: []Tract{
		// The far thigh: grey feathers.
		{Name: "farThigh", Region: Poly(20, 18, 32, 18, 32, 26.6, 20, 26.6), Parts: []string{"farLeg"},
			Flow: Along(25, 19, 27.5, 24, 25.5, 28.5), Len: 2.5, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The body: rufous along the back, charcoal blue-grey down the flank
		// in scalloped rows, each feather edged pale.
		{Name: "body", Region: Union(Poly(20, 6, 43, 6, 47.5, 9.5, 47.5, 14.5, 43, 22, 20, 23), Ellipse(45.5, 5.8, 2.4, 2.6, 0)),
			Parts: []string{"body", "wing"},
			Flow:  Along(48, 2.6, 45.5, 6.5, 41, 10, 34, 11.5, 25, 12, 20, 11.5), Len: 2.9, Wid: 1.9, Rows: 0.45, Jitter: 0.2, Fluff: 0.6, Snap: true,
			Hang: Poly(-1, 21.8, 61, 21.8, 61, 40, -1, 40), TipMat: "edge", TipLen: 0.2, TipOn: []string{"charcoal"}},
		// The crown and its small rusty crest, the nape behind it.
		{Name: "crown", Region: Poly(42, -1, 50.6, -1, 50.6, 3.4, 48.5, 5.4, 45.5, 9, 42, 9), Parts: []string{"body"},
			Flow: Along(51, 2.8, 47, 2.2, 44.5, 5.5, 42.5, 10), Len: 1.6, Wid: 1.0, Fluff: 0.6, Snap: true},
		// The tail: long and stiff, its feathers lying flat along both sides,
		// a little longer at the end, a narrow vane to the point.
		{Name: "tail", Region: Poly(-1, 7, 24.5, 7, 24.5, 17.5, -1, 13.5), Parts: []string{"body"},
			Flow: Along(26, 12.9, 15, 12.1, 5, 11.2, -1, 10.6), Len: 3.0, Wid: 1.3, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05, Grow: 0.015, Fluff: 0.3},
		// The near thigh.
		{Name: "nearThigh", Region: Poly(30, 18, 42, 18, 42, 26.6, 30, 26.6), Parts: []string{"nearLeg"},
			Flow: Along(35, 19, 37.5, 24, 35.5, 28.5), Len: 2.5, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The folded wing: its long rusty flight feathers laid back along the
		// flank, dark at their tips.
		{Name: "wing", Region: Poly(42, 14.4, 43, 17.8, 37, 19.6, 30.2, 20.2, 31.2, 18.2, 35, 16.4, 38.5, 14.8), Parts: []string{"wing"},
			Flow: Along(45, 16, 41.5, 17.3, 37, 18.5, 31, 19.5), Len: 4.6, Wid: 1.35, Rows: 0.34, Tip: 0.55, Shaft: 0.1, Fluff: 0.35,
			Mat: "rust", TipMat: "tipDark", TipLen: 0.28},
		// The coverts over the arm.
		{Name: "arm", Region: Poly(40.6, 12.6, 45.6, 14.4, 46, 18.8, 43.6, 19.4, 41.2, 17.4, 40, 14.2), Parts: []string{"wing"},
			Flow: Along(41, 12.5, 43.3, 16.2, 44.6, 19.5), Len: 2.2, Wid: 1.6, Fluff: 0.35, Mat: "rust"},
	}}
}

func dakotaDetails(p *Painter) {
	p.Plumage(dakotaPlumage())
	// The mouth: the upper teeth along the lip, above the rig's jaw line
	// (drawnTeeth), and a lip over them.
	p.Teeth(TeethSpec{X0: 49.6, Y0: 7.25, X1: 58.6, Y1: 7.35, Bottom: 8.05, N: 8, Len: 0.85,
		Ramp: raptorTeeth, Gap: 0x2a1210, Seed: 9})
	p.Lip([]float64{49.2, 7.3, 54, 7.3, 58.8, 7.4}, 4)
	p.Nostril(57.6, 5.4, 0.55, 0.32, -0.3)
	p.Eye(EyeSpec{X: 52.6, Y: 5.0, R: 0.95, Iris: raptorIris, Slit: true, Lid: 0.25, Squint: 1.25})
	// The hand's two claws, hooked.
	p.Claw(43.2, 19.3, 43.6, 21.3, 0.85, 0.35, sickleHorn[:6])
	p.Claw(45, 19.3, 45.7, 21.2, 0.85, 0.35, sickleHorn[:6])
	// The feet: a small claw on each toe on the ground, and the great
	// sickle claw of the second toe held up off it. Each stays in its leg's
	// columns (12-21 far, 22-31 near).
	p.Claw(29.3, 34.2, 31.4, 35.0, 0.75, 0.2, sickleHorn[:5])
	p.Claw(28.1, 32.3, 31.2, 32.9, 1.25, -0.95, sickleHorn)
	p.Claw(39.3, 34.2, 41.4, 35.0, 0.8, 0.2, sickleHorn[:6])
	p.Claw(38.1, 32.3, 41.4, 32.9, 1.3, -0.95, sickleHorn)
}
