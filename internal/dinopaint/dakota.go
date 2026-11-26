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
		GuideDetail: 0.35,
		Ink:         37,
		Materials: []Material{
			{Name: "orange", Seeds: []uint32{0xe6aa5e, 0xcb843c, 0xad6229, 0x88461d, 0x602e15, 0x7a5a3c},
				Ramp: []uint32{0x2c1512, 0x48210f, 0x6c3214, 0x904a1c, 0xac5e24, 0xc4742e, 0xd88a3c, 0xe8a454}},
			{Name: "bar", Seeds: []uint32{0x5e483a},
				Ramp: []uint32{0x181010, 0x241814, 0x32211a, 0x412b20, 0x513628, 0x634332, 0x77523c, 0x8b6248}, Smooth: 0.6, Bias: 0.1},
			{Name: "dark", Seeds: []uint32{0x433228, 0x2a1f1a},
				Ramp: []uint32{0x140e0d, 0x221714, 0x31221b, 0x413023, 0x523e2d, 0x654d38, 0x7a5e45, 0x907254}, Bias: 0.05},
			{Name: "pale", Seeds: []uint32{0xb89a6c, 0xdcc696},
				Ramp: []uint32{0x3c2c24, 0x5a4434, 0x7a6048, 0x9a7e5e, 0xb59a74, 0xcab18a, 0xdcc8a2, 0xece0bf}},
			{Name: "khaki", Seeds: []uint32{0xa0884c, 0x756036, 0xc8b06a, 0x4a3c22, 0x8a6f4c},
				Ramp: []uint32{0x282018, 0x3a3022, 0x51452e, 0x6a5c38, 0x847442, 0x9e8c50, 0xb7a464, 0xcdbc80}},
			{Name: "white", Seeds: []uint32{0xf2ecdc, 0xf4e8c2, 0xd0c8b4, 0xd8ccb0, 0xf6cf36},
				Ramp: []uint32{0x4e4440, 0x6e6458, 0x948a7a, 0xb4aa96, 0xcec6b0, 0xe2dcc8, 0xf0ecdc, 0xfffcf0}, Bias: 0.1},
			{Name: "farKhaki", Ramp: []uint32{0x1c1712, 0x29221a, 0x383022, 0x483e2c, 0x5a4e36, 0x6c5e40, 0x80704c, 0x948258}}, // the far shank, in shadow
			// The end of the tail: a white band, then the dark point.
			{Name: "tailWhite", Ramp: []uint32{0x4e4440, 0x6e6458, 0x948a7a, 0xb4aa96, 0xcec6b0, 0xe2dcc8, 0xf0ecdc, 0xfffcf0},
				Region: Poly(7.7, 6, 10.9, 6, 10.9, 15, 7.7, 15), Over: true, Only: []string{"orange", "bar", "dark", "white"}},
			{Name: "tailDark", Ramp: []uint32{0x140e0d, 0x221714, 0x31221b, 0x413023, 0x523e2d, 0x654d38, 0x7a5e45, 0x907254},
				Region: Union(Poly(-1, 6, 7.7, 6, 7.7, 15, -1, 15), Poly(10.9, 6, 11.9, 6, 11.9, 15, 10.9, 15)), Over: true, Only: []string{"orange", "bar", "dark", "white"}},
		},
		// The hand's claws are painted again.
		Recolor: []Recolor{
			{Region: Poly(21.5, 26, 32, 26, 32, 37, 21.5, 37), From: []string{"khaki"}, To: "farKhaki"},
			{Region: Poly(42, 19.5, 47, 19.5, 47, 22, 42, 22), From: []string{"white"}, To: "khaki"},
			{Region: Poly(27.5, 30, 32, 30, 32, 33.5, 27.5, 33.5), From: []string{"white"}, To: "khaki"},
			{Region: Poly(37.5, 30, 42, 30, 42, 33.5, 37.5, 33.5), From: []string{"white"}, To: "khaki"},
			// Specks of the markings, a pixel each, that would be crosses.
			{Region: Circle(18.5, 10.5, 1), From: []string{"bar"}, To: "orange"},
			{Region: Circle(25.5, 14.5, 1), From: []string{"dark"}, To: "orange"},
			{Region: Circle(28.5, 19.5, 1.1), From: []string{"khaki", "dark"}, To: "orange"},
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
			{Name: "jaw", Shape: jaw, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "pale"},
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
	return &Plumage{Seed: 5, Edge: 0.45, Tone: 0.7, Tracts: []Tract{
		// The far thigh.
		{Name: "farThigh", Region: Poly(20, 18, 32, 18, 32, 26.6, 20, 26.6), Parts: []string{"farLeg"},
			Flow: Along(25, 19, 27.5, 24, 25.5, 28.5), Len: 2.5, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The body: small contour feathers from the nape back to the tail.
		{Name: "body", Region: Union(Poly(20, 6, 43, 6, 47.5, 9.5, 47.5, 14.5, 43, 22, 20, 23), Ellipse(45.5, 5.8, 2.4, 2.6, 0)),
			Parts: []string{"body", "wing"},
			Flow:  Along(48, 2.6, 45.5, 6.5, 41, 10, 34, 11.5, 25, 12, 20, 11.5), Len: 2.9, Wid: 1.9, Rows: 0.45, Jitter: 0.2, Fluff: 0.6, Snap: true,
			Hang: Poly(-1, 21.8, 61, 21.8, 61, 40, -1, 40)},
		// The crown and the nape: fine feathers from behind the eye.
		{Name: "crown", Region: Poly(42, 1, 50.6, 1, 50.6, 3.4, 48.5, 5.4, 45.5, 9, 42, 9), Parts: []string{"body"},
			Flow: Along(51, 2.8, 47, 3.4, 44.5, 6.5, 42.5, 10), Len: 1.4, Wid: 1.0, Fluff: 0.45, Snap: true},
		// The tail: long and stiff, its feathers lying flat along both sides
		// to a point.
		{Name: "tail", Region: Poly(-1, 7, 24.5, 7, 24.5, 17.5, -1, 13.5), Parts: []string{"body"},
			Flow: Along(26, 12.9, 15, 12.1, 5, 11.2, -1, 10.6), Len: 3.0, Wid: 1.3, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05},
		// The near thigh.
		{Name: "nearThigh", Region: Poly(30, 18, 42, 18, 42, 26.6, 30, 26.6), Parts: []string{"nearLeg"},
			Flow: Along(35, 19, 37.5, 24, 35.5, 28.5), Len: 2.5, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The folded wing: the arm's coverts, then its long feathers laid back
		// along the flank, dark with white tips.
		{Name: "wing", Region: Poly(42, 14.4, 43, 17.8, 37, 19.6, 30.2, 20.2, 31.2, 18.2, 35, 16.4, 38.5, 14.8), Parts: []string{"wing"},
			Flow: Along(45, 16, 41.5, 17.3, 37, 18.5, 31, 19.5), Len: 4.6, Wid: 1.35, Rows: 0.34, Tip: 0.55, Shaft: 0.1, Fluff: 0.35,
			Mat: "dark", TipMat: "white", TipLen: 0.2, TipGuide: true},
		// The coverts over the arm.
		{Name: "arm", Region: Poly(40.6, 12.6, 45.6, 14.4, 46, 18.8, 43.6, 19.4, 41.2, 17.4, 40, 14.2), Parts: []string{"wing"},
			Flow: Along(41, 12.5, 43.3, 16.2, 44.6, 19.5), Len: 2.2, Wid: 1.6, Fluff: 0.35, Snap: true},
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
