package dinopaint

// Ornithomimus: dinos/ornitho.png, 52 x 48, facing right: the ostrich
// mimic, a cream body on long pink legs, a bare pink neck and head, a
// yellow beak, a dark tail band with a white tip and a dark folded wing
// whose feathers end in white. Its rig is "ornitho" in
// internal/client/dinorigs.go: hips (27, 26) and (18, 26), knees (28.5,
// 32.5) and (19.5, 32.5), ankles (25, 37.5) and (16, 37.5); the legs are cut
// from row 31, columns 23-32 (near) and 14-22 (far); the neck bends about
// (31, 18) and the head about (35.2, 3.6); the beak opens about (41, 7)
// along the line to (46.5, 7.2); the tail turns about (13, 19).

func init() { Register(ornitho()) }

var (
	ornithoBeak = []uint32{0x3a2808, 0x58400e, 0x785a16, 0x987420, 0xb88e2a, 0xd2a836, 0xe6c250, 0xf4dc7c}
	darkIris    = []uint32{0x120c0a, 0x22160e, 0x3a2414, 0x56361c, 0x744a26, 0x946232}
)

func ornitho() *Species {
	farLeg := Union(
		Cone(17.4, 26.6, 3.8, 19.3, 31.8, 2.4),
		Cone(19.3, 32.3, 1.9, 16.2, 37.5, 1.2),
		Cone(16.2, 37.5, 1.1, 16.6, 44.3, 0.95),
		Cone(16.6, 44.8, 0.95, 21.6, 45.4, 0.6),
	)
	nearLeg := Union(
		Cone(26.4, 26.6, 4, 28.3, 31.8, 2.5),
		Cone(28.3, 32.3, 2.0, 25.2, 37.5, 1.2),
		Cone(25.2, 37.5, 1.1, 25.6, 44.3, 0.95),
		Cone(25.6, 44.8, 0.95, 30.6, 45.4, 0.6),
	)
	body := Blend(2.5,
		Path([3]float64{12, 19, 4}, [3]float64{5, 17.2, 3.4}, [3]float64{0.5, 17, 3.5}), // the tail
		Ellipse(20.5, 20.8, 12, 5.6, 0.04),
		Cone(29.5, 19, 3.4, 33.2, 12, 2.1), // the neck
		Cone(33.2, 12, 2.0, 35.6, 6, 1.9),
		Poly(34.6, 3.4, 36, 2.4, 41.5, 2.4, 43, 4, 46.5, 4.5, 47.7, 6.5, 47, 7.5, 40, 7.6, 36.5, 8.6, 34.4, 7), // the head
	)
	jaw := Poly(39.5, 7, 47.4, 7.1, 46.6, 8.0, 40.5, 8.2, 39, 7.9)
	wing := Union( // the arm, and the long feathers of the folded wing behind it
		Cone(30.2, 18.6, 1.9, 31.8, 22.6, 1.4),
		Cone(31.8, 22.8, 1.2, 33.2, 24.8, 0.8),
		Poly(31.4, 19.5, 32.2, 22.6, 25.5, 25, 13.6, 26.8, 14.2, 24, 21, 21.2, 27, 19.6),
	)
	beak := Poly(39.8, 4.2, 43, 4.0, 46.5, 4.5, 47.9, 6.6, 47.1, 8.1, 40.5, 8.3, 39.5, 6.6)
	return &Species{
		Name:        "ornitho",
		GuideDetail: 0.3,
		Ink:         30,
		Materials: []Material{
			{Name: "cream", Seeds: []uint32{0xf4dcae, 0xe0c088, 0xc49c60, 0x9f7844, 0x74532f, 0x4a3320},
				Ramp: []uint32{0x3a2616, 0x58391e, 0x7a5428, 0x9c7436, 0xbc9450, 0xd4b06a, 0xe6c88a, 0xf4e0b0}},
			{Name: "pink", Seeds: []uint32{0xd2b4b0, 0xb08e8e, 0x86686a, 0x5a4444, 0x6a4a3a},
				Ramp: []uint32{0x2a1a1c, 0x40282a, 0x583a3a, 0x725050, 0x8e6866, 0xa8807c, 0xc29c96, 0xd8bab2}},
			{Name: "brown", Seeds: []uint32{0x55331f, 0x33211a, 0x7a4a26},
				Ramp: []uint32{0x1a100c, 0x261810, 0x342014, 0x44291a, 0x553420, 0x684028, 0x7c4e30, 0x905c38}, Bias: 0.04},
			{Name: "white", Seeds: []uint32{0xf6f2e6, 0xd8d0c0, 0xe0d4bc, 0xffffff},
				Ramp: []uint32{0x5a5048, 0x7a7064, 0x9c9282, 0xbcb4a0, 0xd6cebc, 0xe8e2d2, 0xf4f0e4, 0xfffcf4}, Bias: 0.1},
			{Name: "pale", Seeds: []uint32{0xc8b894, 0x9a8a6c},
				Ramp: []uint32{0x30261c, 0x483a2c, 0x62503e, 0x7e6a52, 0x9a8468, 0xb49e80, 0xcab698, 0xdccab0}},
			{Name: "farPink", Ramp: []uint32{0x1e1416, 0x2c1e20, 0x3e2a2c, 0x523a3a, 0x684c4a, 0x7e5e5a, 0x94726c, 0xa8867e}}, // the far shank, in shadow
			{Name: "beak", Seeds: []uint32{0xdcb648, 0xa8842e, 0xf2d878, 0x5a4418}, Ramp: ornithoBeak, Relief: 0.3, Region: beak},
		},
		// The shanks as slender as the drawing's, whose outline is half their
		// width: the painted outline is lighter.
		Remove: []SDF{
			Cut(Poly(13, 33.5, 23, 33.5, 23, 37.8, 13, 37.8), Cone(17.4, 33.5, 1.25, 16.2, 37.5, 1.05)),
			Cut(Poly(13, 37.2, 23, 37.2, 23, 43.6, 13, 43.6), Cone(16.2, 37.5, 1.05, 16.5, 44, 0.95)),
			Cut(Poly(22, 33.5, 32, 33.5, 32, 37.8, 22, 37.8), Cone(26.4, 33.5, 1.3, 25.2, 37.5, 1.05)),
			Cut(Poly(22, 37.2, 32, 37.2, 32, 43.6, 22, 43.6), Cone(25.2, 37.5, 1.05, 25.5, 44, 0.95)),
		},
		Recolor: []Recolor{
			{Region: Poly(13, 31, 22.9, 31, 22.9, 48, 13, 48), From: []string{"pink"}, To: "farPink"},
			{Region: Circle(39.8, 4.8, 1.4), From: []string{"white", "brown"}, To: "pink"}, // the eye is painted again
			{Region: Poly(33, 24, 36, 24, 36, 27, 33, 27), From: []string{"white"}, To: "pale"},
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.4, Scale: 0.6, Muscles: []Muscle{
				Mu(17.6, 27.5, 3, 3, 0, 0.9),
			}},
			{Name: "body", Shape: body, Round: 7, RoundAt: func(x, y float64) float64 {
				return lerp(lerp(7, 2.2, smoothstep(27, 31, x)), 3, smoothstep(9, 3, y))
			}, Muscles: []Muscle{
				Mu(20, 20, 8, 4, 0, 0.8),          // the ribs
				Mu(27.5, 19.5, 3, 3.5, 0.3, 0.6),  // the shoulder
				Mu(11.5, 18.4, 4, 2.4, 0, 0.6),    // the root of the tail
				Mu(37, 5, 2.2, 1.6, 0, 0.5),       // the back of the head
				Mu(39.6, 3.6, 1.4, 0.6, 0, 0.5),   // the brow
				Mu(43.5, 5.2, 3, 0.9, -0.05, 0.4), // the beak
			}},
			{Name: "jaw", Shape: jaw, Round: 0.8, Soft: 0.4, Scale: 0.5},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.6, Scale: 0.6, Onto: "body", Seam: Below(10, 26, 34, 25), Muscles: []Muscle{
				Mu(26.4, 27.5, 3.2, 3.2, 0, 1.0),
			}},
			{Name: "wing", Shape: wing, Round: 1.8, Soft: 0.6, Muscles: []Muscle{
				Mu(30.4, 19.6, 1.4, 2.2, 0.35, 0.7),
				Mu(32, 23, 1.2, 1, 0.4, 0.4),
			}},
		},
		Skin:    Skin{Flank: 1.1, Back: 1.3, Belly: 1.0, BackDepth: 2, BellyDepth: 2, Relief: 0.9, Seed: 43},
		Folds:   []Fold{{Pts: []float64{15.2, 40, 16.8, 40}, N: 4, DY: 1.1, Depth: 0.5}, {Pts: []float64{24.2, 40, 25.8, 40}, N: 4, DY: 1.1, Depth: 0.5}},
		Details: ornithoDetails,
	}
}

func ornithoPlumage() *Plumage {
	return &Plumage{Seed: 13, Edge: 0.4, Tone: 0.7, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(11, 23, 23, 23, 23, 30.4, 11, 30.4), Parts: []string{"farLeg"},
			Flow: Along(18, 26, 19.5, 32.5, 16, 37.5), Len: 2.2, Wid: 1.6, Fluff: 0.45, Snap: true},
		{Name: "body", Region: Poly(8, 12, 31.5, 12.5, 33.5, 17, 31, 25, 10, 27, 7, 20), Parts: []string{"body", "wing"},
			Flow: Along(31.5, 15.5, 26, 15.2, 18, 15.6, 10, 16.6), Len: 2.5, Wid: 1.75, Rows: 0.45, Jitter: 0.2, Fluff: 0.55, Snap: true,
			Hang: Poly(-1, 29.5, 53, 29.5, 53, 50, -1, 50)},
		// Fine down on the neck: the drawing's bare pink, lightly furred.
		{Name: "neck", Region: Poly(29.5, 7, 37, 2.5, 38, 8.6, 34.6, 15, 33.6, 20, 28.5, 19.5), Parts: []string{"body"},
			Flow: Along(36, 3.4, 34.2, 9.5, 32, 15, 29.6, 19.5), Len: 1.1, Wid: 0.62, Hair: true, Relief: 0.55, Fluff: 0.25, Snap: true},
		{Name: "tail", Region: Poly(-1, 12, 11, 13, 12.5, 17, 11, 23, -1, 23), Parts: []string{"body"},
			Flow: Fan(16, 17.5, 8, 3.1416), Len: 3.8, Wid: 1.5, Rows: 0.36, Tip: 0.5, Shaft: 0.12, Fluff: 0.7, Grow: 0.02},
		{Name: "nearThigh", Region: Poly(20, 23, 33, 23, 33, 30.4, 20, 30.4), Parts: []string{"nearLeg"},
			Flow: Along(27, 26, 28.5, 32.5, 25, 37.5), Len: 2.2, Wid: 1.6, Fluff: 0.45, Snap: true},
		{Name: "wing", Region: Poly(30.4, 19, 31.4, 22.6, 25, 25.2, 13.4, 27.2, 14, 24, 21, 21.2, 27, 19.4), Parts: []string{"wing"},
			Flow: Along(31, 20.6, 26, 22.2, 20, 24.2, 14, 25.6), Len: 4.6, Wid: 1.4, Rows: 0.34, Tip: 0.55, Shaft: 0.1, Fluff: 0.35,
			Mat: "brown", TipMat: "white", TipLen: 0.25, TipGuide: true},
		{Name: "arm", Region: Poly(28.8, 17.2, 33.4, 19.4, 34.2, 25, 31.6, 25.2, 29.4, 21), Parts: []string{"wing"},
			Flow: Along(29.6, 17.6, 31.4, 21.4, 33, 24.8), Len: 2.0, Wid: 1.4, Fluff: 0.3, Snap: true},
	}}
}

func ornithoDetails(p *Painter) {
	p.Plumage(ornithoPlumage())
	p.Beak(BeakSpec{Mat: "beak", Ridge: []float64{39.6, 4.3, 43, 4.1, 46.4, 4.6, 47.8, 6.4},
		Mouth: []float64{39.8, 7.0, 43.5, 7.05, 47.2, 7.1}, Gloss: 0.24, Nostril: [2]float64{42.2, 5.2}, NoseR: 0.5, GrainSeed: 3})
	p.Eye(EyeSpec{X: 39.8, Y: 4.85, R: 0.82, Iris: darkIris, Lid: 0.18, Squint: 1.1})
	// The hand's claws and the feet's.
	p.Claw(33.4, 24.6, 34.2, 26.4, 0.7, 0.3, sickleHorn[:6])
	p.Claw(32.4, 24.8, 32.9, 26.5, 0.65, 0.25, sickleHorn[:5])
	p.Claw(20.8, 45.0, 22.4, 45.7, 0.6, 0.15, sickleHorn[:5])
	p.Claw(29.8, 45.0, 31.6, 45.7, 0.65, 0.15, sickleHorn[:6])
}
