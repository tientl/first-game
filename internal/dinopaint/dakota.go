package dinopaint

// Dakotaraptor: dinos/dakota.png, 77 x 37, facing right, its long stiff tail,
// half as long again as its body, held out straight behind to a fine point.
// Its rig is "dakota" in internal/client/dinorigs.go: hips (52, 19) and
// (42, 19), knees (54.5, 24) and (44.5, 24), ankles (51.5, 28.5) and
// (41.5, 28.5); the legs swing whole from the hips (thigh.go); the head turns
// about (58, 10), the jaw opens about (67.5, 8) along the line to (75, 8.4),
// and the tail turns about (39.5, 12.8).

// The colours shared by the feathered hunters.
var (
	sickleHorn  = []uint32{0x1a1411, 0x352b24, 0x5a4c40, 0x857664, 0xb2a48c, 0xdcd2bc, 0xf6f0e0}
	raptorIris  = []uint32{0x5a2a04, 0x8e4a08, 0xc87a12, 0xeeaa26, 0xffd04c, 0xffe890}
	raptorTeeth = []uint32{0x6e5e4a, 0x9c8c70, 0xc8ba98, 0xe8dec2, 0xfaf6e6}
)

func init() {
	Register(dakota())
	// The legs swing whole from the hip, the near thigh under the folded
	// wing and the flank over the hip (thigh.go).
	RegisterThighs("dakota", &Thighs{
		Legs:       LegMask{{"nearLeg"}, {"farLeg"}},
		FrontParts: []string{"wing"},
		Front:      Poly(46, 12, 60, 12, 60, 20.2, 46, 20.2),
		Thigh:      []SDF{Cone(52.6, 21.2, 3.8, 54.3, 24, 3), Cone(42.4, 21, 3.7, 44.2, 24, 2.9)},
		Coat:       Plumed(dakotaPlumage),
	})
}

func dakota() *Species {
	farLeg := Union(
		Cone(42.4, 21, 3.4, 44.2, 24, 2.6),     // the thigh
		Cone(44.2, 24, 2.4, 41.7, 28.5, 1.5),   // the shank
		Cone(41.7, 28.5, 1.3, 42.3, 33.6, 1.1), // the metatarsus
		Cone(42.3, 33.9, 1.1, 46.6, 34.4, 0.7), // the toes
		Cone(43.4, 33.3, 0.8, 45.4, 32.5, 0.6), // the second toe, raised
	)
	nearLeg := Union(
		Cone(52.6, 21.2, 3.5, 54.3, 24, 2.7),
		Cone(54.3, 24, 2.5, 51.7, 28.5, 1.5),
		Cone(51.7, 28.5, 1.3, 52.3, 33.6, 1.1),
		Cone(52.3, 33.9, 1.1, 56.6, 34.4, 0.7),
		Cone(53.4, 33.3, 0.8, 55.4, 32.5, 0.6),
	)
	body := Blend(2.5,
		Path([3]float64{42, 13, 3.8}, [3]float64{30.94, 12.1, 2.6}, [3]float64{12.38, 11.2, 1.2}, [3]float64{0.83, 10.7, 0.3}), // the long stiff tail
		Ellipse(48, 15.6, 11.5, 6.2, 0.06),                                           // the body
		Cone(55.5, 13.5, 4.6, 62, 7.5, 2.7),                                          // the neck
		Poly(60.5, 2.5, 69, 2.5, 73, 4, 75.5, 5.6, 76.6, 7.6, 67, 8, 62, 9.5, 60, 6), // the head
	)
	jaw := Poly(62.5, 8, 76.4, 8.1, 76.2, 9.6, 64, 9.9, 62, 9.6)
	wing := Union( // the arm folded, its long feathers along the flank
		Cone(58.6, 14.4, 2.3, 60.7, 18.3, 1.6),
		Cone(60.7, 18.5, 1.2, 62.4, 19.5, 0.8),
		Poly(58, 13.4, 62, 15, 61.8, 19.9, 52, 21.2, 44, 22, 43.6, 20.3, 49.6, 14.3, 55, 13.8),
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
				Region: Poly(60.6, 3.7, 69.8, 3.5, 71.2, 4.6, 69.8, 5.9, 64, 6.2, 60.6, 6.0), Over: true, Only: []string{"white", "rufous", "snout", "charcoal"}},
			// The tail ends in a narrow vane banded dark, to a dark point.
			{Name: "tailBand", Ramp: []uint32{0x0e0a08, 0x16100c, 0x1e1610, 0x281e16, 0x32261c, 0x3e3022, 0x4a3a2a, 0x564432},
				Region: Union(Poly(22.6, 6, 24.4, 6, 24.4, 15, 22.6, 15), Poly(17.8, 6, 19.4, 6, 19.4, 15, 17.8, 15),
					Poly(13.7, 6, 15.1, 6, 15.1, 15, 13.7, 15), Poly(10.3, 6, 11.5, 6, 11.5, 15, 10.3, 15), Poly(-3, 6, 7.8, 6, 7.8, 15, -3, 15)),
				Over: true, Only: []string{"rufous", "tipDark"}},
		},
		Recolor: []Recolor{
			// The end of the tail is rufous between its bands, the drawing's
			// bands stretched with it painted again.
			{Region: Poly(-3, 5, 27, 5, 27, 16, -3, 16), From: []string{"tipDark"}, To: "rufous"},
			{Region: Poly(38.5, 27, 49, 27, 49, 37, 38.5, 37), From: []string{"feet"}, To: "farFeet"},
			// The guide's claws are painted again.
			{Region: Poly(59, 19.5, 64, 19.5, 64, 22, 59, 22), From: []string{"ivory"}, To: "rust"},
			{Region: Poly(44.5, 30, 49, 30, 49, 33.5, 44.5, 33.5), From: []string{"ivory"}, To: "feet"},
			{Region: Poly(54.5, 30, 59, 30, 59, 33.5, 54.5, 33.5), From: []string{"ivory"}, To: "feet"},
			{Region: Poly(37, 14, 61, 14, 61, 21, 37, 21), From: []string{"ivory"}, To: "tipDark"},
		},
		// The tail tapers smooth to its fine point, however the drawing's steps.
		Add: []SDF{Path([3]float64{30.94, 12.1, 2.4}, [3]float64{12.38, 11.2, 1.1}, [3]float64{0.6, 10.7, 0.2})},
		Remove: []SDF{
			Poly(59.5, 19.6, 64, 19.6, 64, 22, 59.5, 22), // the guide's hand claws
			Poly(45.6, 30, 49, 30, 49, 33, 45.6, 33),     // and its sickle claws
			Poly(55.6, 30, 59, 30, 59, 33, 55.6, 33),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.6, Scale: 0.7, Muscles: []Muscle{
				Mu(43.2, 22, 2.8, 2.8, 0, 0.9),
				Mu(43, 25.5, 1.4, 1.6, 0.5, 0.4),
			}},
			{Name: "body", Shape: body, Round: 7, RoundAt: func(x, y float64) float64 { return lerp(7.5, 2.8, smoothstep(57, 64, x)) }, Muscles: []Muscle{
				Mu(48, 14.5, 8, 4.5, 0, 0.8),     // the ribs
				Mu(56.5, 13.5, 3, 3.5, 0.3, 0.7), // the shoulder
				Mu(38, 11.5, 4.5, 2.5, 0, 0.7),   // the root of the tail
				Mu(65.5, 4.6, 3, 1.6, 0, 0.7),    // the jaw muscle behind the eye
				Mu(72, 5.6, 3.5, 1.3, 0, 0.5),    // the snout
				Mu(69.6, 3.6, 1.6, 0.7, 0, 0.6),  // the brow over the eye
			}},
			{Name: "jaw", Shape: jaw, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "snout"},
			{Name: "nearLeg", Shape: nearLeg, Round: 3, Scale: 0.7, Onto: "body", Seam: Below(37, 21.5, 61, 20.5), Muscles: []Muscle{
				Mu(53, 22, 3, 3, 0, 1.0),
				Mu(53, 25.5, 1.5, 1.6, 0.5, 0.4),
			}},
			{Name: "wing", Shape: wing, Round: 2.2, Soft: 0.6, Muscles: []Muscle{
				Mu(59, 15.5, 1.5, 2.4, 0.35, 0.9), // the upper arm
				Mu(61, 18.6, 1.4, 1, 0.4, 0.5),    // the forearm
			}},
		},
		Skin: Skin{Flank: 1.4, Back: 1.8, Belly: 1.2, BackDepth: 3, BellyDepth: 3, Relief: 0.8, Seed: 31},
		Folds: []Fold{
			{Pts: []float64{41, 30.2, 43.2, 30.2}, N: 3, DY: 1.1, Depth: 0.6}, // across the metatarsus
			{Pts: []float64{51, 30.2, 53.2, 30.2}, N: 3, DY: 1.1, Depth: 0.6},
		},
		Details: dakotaDetails,
	}
}

func dakotaPlumage() *Plumage {
	return &Plumage{Seed: 5, Edge: 0.45, Tone: 0.6, Tracts: []Tract{
		// The far thigh: grey feathers.
		{Name: "farThigh", Region: Poly(37, 18, 49, 18, 49, 26.6, 37, 26.6), Parts: []string{"farLeg"},
			Flow: Along(42, 19, 44.5, 24, 42.5, 28.5), Len: 2.5, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The body: rufous along the back, charcoal blue-grey down the flank
		// in scalloped rows, each feather edged pale.
		{Name: "body", Region: Union(Poly(37, 6, 60, 6, 64.5, 9.5, 64.5, 14.5, 60, 22, 37, 23), Ellipse(62.5, 5.8, 2.4, 2.6, 0)),
			Parts: []string{"body", "wing"},
			Flow:  Along(65, 2.6, 62.5, 6.5, 58, 10, 51, 11.5, 42, 12, 37, 11.5), Len: 2.9, Wid: 1.9, Rows: 0.45, Jitter: 0.2, Fluff: 0.6, Snap: true,
			Hang: Poly(-2.06, 21.8, 78, 21.8, 78, 40, -2.06, 40), TipMat: "edge", TipLen: 0.2, TipOn: []string{"charcoal"}},
		// The crown and its small rusty crest, the nape behind it.
		{Name: "crown", Region: Poly(59, -1, 67.6, -1, 67.6, 3.4, 65.5, 5.4, 62.5, 9, 59, 9), Parts: []string{"body"},
			Flow: Along(68, 2.8, 64, 2.2, 61.5, 5.5, 59.5, 10), Len: 1.6, Wid: 1.0, Fluff: 0.6, Snap: true},
		// The tail: long and stiff, its feathers lying flat along both sides,
		// a little longer at the end, a narrow vane to the point.
		{Name: "tail", Region: Poly(-2.06, 7, 41.5, 7, 41.5, 17.5, -2.06, 13.5), Parts: []string{"body"},
			Flow: Along(43, 12.9, 30.94, 12.1, 10.31, 11.2, -2.06, 10.6), Len: 3.0, Wid: 1.3, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05, Grow: 0.015, Fluff: 0.3},
		// The near thigh.
		{Name: "nearThigh", Region: Poly(47, 18, 59, 18, 59, 26.6, 47, 26.6), Parts: []string{"nearLeg"},
			Flow: Along(52, 19, 54.5, 24, 52.5, 28.5), Len: 2.5, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The small coverts over the shoulder and the arm; the folded wing
		// below them is painted feather by feather (dakotaWing).
		{Name: "arm", Region: Poly(57.4, 12.4, 62.4, 13.8, 62.8, 16.4, 60, 16.6, 57.4, 15.4), Parts: []string{"wing"},
			Flow: Along(58, 12.5, 60.3, 16.2, 61.6, 19.5), Len: 1.8, Wid: 1.4, Fluff: 0.3, Mat: "rust"},
	}}
}

func dakotaDetails(p *Painter) {
	p.Plumage(dakotaPlumage())
	p.FoldedWing(dakotaWing)
	// The mouth: the upper teeth along the lip, above the rig's jaw line
	// (drawnTeeth), and a lip over them.
	p.Teeth(TeethSpec{X0: 66.6, Y0: 7.25, X1: 75.6, Y1: 7.35, Bottom: 8.05, N: 8, Len: 0.85,
		Ramp: raptorTeeth, Gap: 0x2a1210, Seed: 9})
	p.Lip([]float64{66.2, 7.3, 71, 7.3, 75.8, 7.4}, 4)
	p.Nostril(74.6, 5.4, 0.55, 0.32, -0.3)
	p.Eye(EyeSpec{X: 69.6, Y: 5.0, R: 0.95, Iris: raptorIris, Slit: true, Lid: 0.25, Squint: 1.25})
	// The hand's two claws, hooked.
	p.Claw(60.2, 19.3, 60.6, 21.3, 0.85, 0.35, sickleHorn[:6])
	p.Claw(62, 19.3, 62.7, 21.2, 0.85, 0.35, sickleHorn[:6])
	// The feet: a small claw on each toe on the ground, and the great
	// sickle claw of the second toe held up off it. Each stays in its leg's
	// columns (12-21 far, 22-31 near).
	p.Claw(46.3, 34.2, 48.4, 35.0, 0.75, 0.2, sickleHorn[:5])
	p.Claw(45.1, 32.3, 48.2, 32.9, 1.25, -0.95, sickleHorn)
	p.Claw(56.3, 34.2, 58.4, 35.0, 0.8, 0.2, sickleHorn[:6])
	p.Claw(55.1, 32.3, 58.4, 32.9, 1.3, -0.95, sickleHorn)
}

// dakotaWing is the folded wing: nine long rusty flight feathers laid back
// along the flank from the forearm and the hand, browner down the wing and
// along each feather, their tips dark and stepping back one past another to
// a row of points; a row of coverts over their roots.
var dakotaWing = FoldedWing{
	N: 8, Root0: [2]float64{58.6, 16.0}, Root1: [2]float64{60.6, 18.7},
	Tip0: [2]float64{49.8, 14.3}, Tip1: [2]float64{43.8, 21.3}, Width: 2.0,
	CN: 6, Covert0: [2]float64{58.4, 14.4}, Covert1: [2]float64{61.0, 18.2}, CLen: 3.8, CWid: 1.8,
	Root:   []uint32{0x32160a, 0x4c220c, 0x663212, 0x824418, 0x9c561e, 0xb46a26, 0xc87e32, 0xd89444},
	Vane:   []uint32{0x1e100a, 0x2c180e, 0x3c2212, 0x4e2e18, 0x623c20, 0x764a28, 0x8a5a32, 0x9c6a3e},
	Tip:    []uint32{0x0e0a08, 0x16100c, 0x1e1610, 0x281e16, 0x32261c, 0x3e3022, 0x4a3a2a, 0x564432},
	TipLen: 0.14, Seed: 21,
}
