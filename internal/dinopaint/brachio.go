package dinopaint

// The long-neck: dinos/brachio.png, 114 x 112, facing right, a
// Brachiosaurus. Its rig is "brachio" in internal/client/dinorigs.go: legs
// near hind (hip 29, 72), far hind (41, 74), far fore (55, 74), near fore
// (66, 72), each to its knee at row 90 and its foot at 102 to 111.5; the
// neck bends at its root about (72, 44) and the head about (83.5, 25.5);
// the mouth opens about (99, 12.5) along the line to (107, 13.5); the tail
// swings about (22, 66).

var (
	brachioHoof = []uint32{0x2a2a20, 0x58584a, 0x8a8a76, 0xb8b8a2, 0xdcdcc8}
	brachioIris = []uint32{0x4a2a04, 0x7a4a06, 0xb07808, 0xdca414, 0xffc838, 0xffe68a}
)

func init() {
	Register(brachio())
	// The withers and the chest under the root of the neck, for when it bends
	// down to eat.
	RegisterUnder("brachio", &Under{Parts: []string{"neck", "head"}, Mat: "hide"})
}

// brachioNeck is the neck's middle line, from its root to the head, and
// how thick it is there.
var brachioNeck = [][3]float64{{70, 50, 8.4}, {79, 36, 6}, {87, 22, 5.1}, {93.6, 11.6, 4.3}}

func brachio() *Species {
	nearHind := Union(
		Cone(29, 68, 8.4, 30, 90, 6.1),
		Cone(30, 90, 5.9, 29.5, 101.5, 5.6),
		Ellipse(30.4, 107.4, 6.9, 4.4, 0), // the foot, round and broad as an elephant's
	)
	farHind := Union(
		Cone(41, 74, 6.4, 41.5, 90, 5),
		Cone(41.5, 90, 4.8, 41, 100, 4.6),
		Ellipse(41.6, 104.5, 5.4, 4, 0),
	)
	farFore := Union(
		Cone(55, 74, 6, 54.5, 90, 4.9),
		Cone(54.5, 90, 4.7, 55, 100, 4.5),
		Ellipse(55.4, 104.5, 5.2, 4, 0),
	)
	nearFore := Union(
		Cone(66, 67, 7.4, 65.5, 90, 6.1),
		Cone(65.5, 90, 5.9, 66, 101.5, 5.6),
		Ellipse(66.6, 107.4, 6.8, 4.4, 0),
	)
	body := Blend(4,
		Path([3]float64{1.5, 83.6, 0.9}, [3]float64{10, 78, 3}, [3]float64{18, 70, 6}, [3]float64{26, 62, 10}), // the tail
		Ellipse(46, 59, 25, 15.4, -0.1), // the barrel of the body
		Ellipse(70, 57, 8.5, 11.5, 0),   // the chest
		Ellipse(66, 49, 8.5, 9.5, 0),    // the withers, under the root of the neck
	)
	neck := Blend(2,
		Path(brachioNeck...),
		Ellipse(70, 50, 8, 7, -1), // its root, into the shoulders
	)
	head := Blend(1.5,
		Poly(92.4, 6.6, 94.4, 3.2, 97, 2.3, 99.6, 3.1, 101.6, 5.5, 104.6, 8.6, 107.5, 10.6, 107.7, 13.4, 106, 15.1, 100, 15.3, 96, 14.7, 93.4, 13.2),
		Circle(94.5, 11, 4),
	)
	return &Species{
		Name:        "brachio",
		GuideDetail: 0.5,
		Ink:         32,
		Materials: []Material{
			{Name: "hide", Seeds: []uint32{0x7b8c5e, 0x94a474, 0xa2ae8a, 0xbcc6a4, 0x869370, 0x63734a, 0x6a7757},
				Ramp:   []uint32{0x10160c, 0x1a2414, 0x26321c, 0x344226, 0x465630, 0x566a3c, 0x6c7e4a, 0x84945c},
				Region: Poly(15, 99, 80, 99, 80, 113, 15, 113), Only: []string{"tan"}}, // the guide's nails are painted again
			// The dark mottling, a net of blotches as a giraffe's.
			{Name: "patch", Seeds: []uint32{0x4c5a38, 0x4e5a40, 0x353e2c, 0x374229, 0x404734, 0x20261c, 0x242c1c},
				Ramp: []uint32{0x080c06, 0x0e140a, 0x141c0e, 0x1c2614, 0x243018, 0x2e3a1e, 0x384626, 0x44522e}, Smooth: 0.25, Bias: 0.3},
			// The near legs, a shade darker than the flank above them.
			{Name: "leg", Ramp: []uint32{0x0c120a, 0x141c10, 0x1e2a18, 0x2a3820, 0x38482a, 0x4a5c36, 0x5e7044, 0x748656}},
			{Name: "tan", Seeds: []uint32{0x9f9e78, 0x7a7958, 0xc2c09a, 0xa09276, 0x5f5f48, 0x575742, 0xdddbb8, 0xeeecd2},
				Ramp: []uint32{0x2e2c22, 0x423f30, 0x5a5640, 0x726e52, 0x8c8868, 0xa6a280, 0xbebb98, 0xd4d2b2}, Relief: 0.7, Smooth: 0.5},
		},
		Recolor: []Recolor{
			// Down the legs the drawing's dark is their shading.
			{Region: Cut(Union(nearHind, nearFore), body.Grow(-1.5)), From: []string{"patch"}, To: "hide"},
		},
		Parts: []Part{
			{Name: "farHind", Shape: farHind, Far: true, Round: 4, Scale: 0.85, Mat: "patch", Muscles: []Muscle{
				Mu(41, 79, 4.8, 6, 0, 1.2),
			}},
			{Name: "farFore", Shape: farFore, Far: true, Round: 4, Scale: 0.85, Mat: "patch", Muscles: []Muscle{
				Mu(55, 79, 4.6, 6, 0, 1.2),
			}},
			{Name: "body", Shape: body, Round: 10, Muscles: []Muscle{
				Mu(30, 58, 9, 9, 0, 1.8),      // the hips
				Mu(48, 56, 12, 9, 0, 1.2),     // the ribs
				Mu(66, 55, 6, 8, 0.2, 1.6),    // the shoulder
				Mu(48, 70, 14, 3.4, 0, 0.7),   // the belly
				Mu(14, 74, 9, 3, -0.6, 1.0),   // the tail
				Mu(44, 48, 16, 3, -0.08, 0.7), // the long muscles of the back
				Mu(73, 62, 4, 6.5, 0, 0.8),    // the chest
			}},
			{Name: "neck", Shape: neck, Round: 5, Soft: 0.7, Scale: 0.85, Onto: "body", Seam: Below(60, 64, 82, 54), Muscles: []Muscle{
				// The long muscles down its upper side, then down the throat.
				Mu(72.8, 41.1, 6.5, 1.9, -1.02, 1.8), Mu(78.7, 31.5, 6.5, 1.9, -1.02, 1.7),
				Mu(84.6, 21.9, 6, 1.8, -1.02, 1.5), Mu(89.3, 14.2, 4, 1.6, -1.02, 1.2),
				// the groove between them and the throat's
				Mu(75.5, 42.8, 7, 0.8, -1.02, -0.9), Mu(81.4, 33.2, 7, 0.8, -1.02, -0.9), Mu(87.3, 23.6, 6, 0.7, -1.02, -0.8),
				Mu(77.3, 43.9, 6, 1.5, -1.02, 0.9), Mu(83.2, 34.3, 6, 1.5, -1.02, 0.9), Mu(89.1, 24.7, 5, 1.3, -1.02, 0.8),
				Mu(69, 47, 6, 6, 0, 1.2), // its root, a hump at the withers
			}},
			{Name: "head", Shape: head, Round: 2.8, Soft: 0.6, Scale: 0.7, Onto: "neck", Seam: Below(90, 16, 108, 16), Muscles: []Muscle{
				Mu(97.5, 4.2, 2.4, 1.6, 0, 1.2),   // the crest over the nostrils
				Mu(98, 9.6, 2.4, 2, 0, 0.8),       // the cheek
				Mu(104.4, 11.4, 2.6, 1.6, 0, 0.7), // the snout
			}},
			{Name: "nearHind", Shape: nearHind, Round: 5.6, Mat: "leg", Onto: "body", Seam: Below(15, 74, 40, 74), Muscles: []Muscle{
				Mu(29, 75, 6.4, 8, 0, 2.0),        // the thigh
				Mu(29.8, 90, 3.4, 2.6, 0, 0.7),    // the knee
				Mu(29.8, 96, 3.6, 4, 0, 0.6),      // the shin
				Mu(30.4, 106.6, 5.6, 2.8, 0, 0.8), // the foot
			}},
			{Name: "nearFore", Shape: nearFore, Round: 5.6, Mat: "leg", Onto: "body", Seam: Below(56, 75, 76, 72), Muscles: []Muscle{
				Mu(66, 74, 6, 7.5, 0, 1.8),     // the upper arm
				Mu(65.6, 90, 3.4, 2.6, 0, 0.7), // the elbow
				Mu(65.8, 96, 3.6, 4, 0, 0.6),   // the forearm
				Mu(66.6, 106.6, 5.6, 2.8, 0, 0.8),
			}},
		},
		Skin:    Skin{Flank: 2.3, Back: 3.4, Belly: 1.7, BackDepth: 5, BellyDepth: 5, Seed: 61},
		Folds:   brachioFolds(),
		Details: brachioDetails,
	}
}

// brachioFolds are the creases in its skin: across the throat all down the
// neck, thickest at its root where it bends, round the legs as an
// elephant's, and the line of the mouth.
func brachioFolds() []Fold {
	f := []Fold{
		{Pts: []float64{99, 12.8, 103, 13.2, 107, 13.6}, N: 1, Depth: 1.6}, // the mouth
	}
	// Across the throat: short creases from the middle of the neck to its
	// lower edge, every few pixels up it.
	for i, c := range [][2]float64{{74, 46}, {76.2, 42.6}, {78.4, 39.1}, {80.6, 35.5}, {83, 31.8}, {85.4, 28}, {87.8, 24.2}, {90.2, 20.4}, {92.4, 16.8}} {
		w := 6.2 - 0.35*float64(i)
		f = append(f, Fold{Pts: []float64{c[0] + 0.85*0.6, c[1] + 0.53*0.6, c[0] + 0.85*w, c[1] + 0.53*w}, N: 1, Depth: 1.5 - 0.06*float64(i)})
	}
	// Where it bends at the root, folds over the shoulder.
	f = append(f, Fold{Pts: []float64{67.5, 43.5, 70.5, 46.2, 73, 50}, N: 3, DX: 1.6, DY: -1.2, Depth: 1.0})
	// The legs, wrinkled round.
	for _, l := range [][3]float64{{24.5, 35, 92}, {37, 46, 92}, {50.5, 59.5, 92}, {60.5, 71.5, 92}} {
		f = append(f, Fold{Pts: []float64{l[0], l[2], (l[0] + l[1]) / 2, l[2] + 0.4, l[1], l[2]}, N: 4, DY: 2.4, Depth: 0.8})
	}
	return f
}

func brachioDetails(p *Painter) {
	p.Eye(EyeSpec{X: 99.2, Y: 8.4, R: 0.85, Iris: brachioIris, Lid: 0.3, Squint: 1.25})
	p.Nostril(97.6, 3.8, 0.6, 0.4, -0.4)
	// The nails along the front of the feet: the near ones, then the far.
	for _, n := range [][3]float64{{27.4, 111.8, 1}, {30.7, 111.9, 1.08}, {33.9, 111.8, 1}, {36.3, 111.4, 0.8}, {63.4, 111.8, 1}, {66.7, 111.9, 1.08}, {69.9, 111.8, 1}, {72.2, 111.4, 0.8}} {
		p.Nail(n[0], n[1], 3.0*n[2], 2.3*n[2], brachioHoof, -0.1)
	}
	for _, n := range [][2]float64{{39.4, 108.5}, {42.2, 108.6}, {45, 108.4}, {53.2, 108.5}, {56, 108.6}, {58.6, 108.4}} {
		p.Nail(n[0], n[1], 2.3, 1.6, brachioHoof, -0.45)
	}
}
