package dinopaint

// The bull-frill: dinos/toro.png, 86 x 62, facing right, a Torosaurus. Its
// rig is "toro" in internal/client/dinorigs.go: legs near hind (hip 21.5,
// 39), far hind (32.5, 40), far fore (42, 40), near fore (53, 39); the head,
// frill and horns turn about (46.5, 35.5); the beak opens about (67.6, 44.2)
// along the line to (77.9, 46.5); the tail turns about (16, 36).

var (
	toroIvory = []uint32{0x4c4436, 0x7c705c, 0xaa9c80, 0xd0c4a6, 0xeae2c8, 0xfaf6ea}
	toroHoof  = []uint32{0x2e2c2a, 0x5c5850, 0x8e887a, 0xbab4a2, 0xdcd8c8}
)

// toroThighs are its legs' upper parts, thigh and upper arm, which turn whole
// with the hip and the shoulder: near hind, far hind, far fore, near fore.
var toroThighs = []SDF{
	Cone(21.5, 37.5, 7.9, 22.5, 49.5, 5.5), Cone(32.5, 39, 6.5, 33, 49.5, 4.9),
	Cone(42, 39, 6.5, 41.5, 49.5, 4.9), Cone(53, 40.5, 6.7, 52.5, 49.5, 5.5),
}

func init() {
	Register(toro())
	// The shoulders and the neck under the frill, for when the head turns.
	RegisterUnder("toro", &Under{Parts: []string{"frill", "head"}, Mat: "hide"})
	// Its legs, in the rig's order, cut as painted and swung whole from the
	// hip and the shoulder (thigh.go).
	RegisterThighs("toro", &Thighs{
		Legs:  LegMask{{"nearHind"}, {"farHind"}, {"farFore"}, {"nearFore"}},
		Thigh: toroThighs,
	})
}

func toro() *Species {
	nearHind := Union(
		Cone(21.5, 37.5, 7.4, 22.5, 49.5, 5.0),
		Cone(22.5, 49.5, 4.8, 22, 56, 4.3),
		Ellipse(22.2, 58.5, 5.4, 2.7, 0), // the foot, round as an elephant's
	)
	farHind := Union(
		Cone(32.5, 39, 6, 33, 49.5, 4.4),
		Cone(33, 49.5, 4.2, 32.5, 54.5, 3.8),
		Ellipse(33, 56.4, 4.6, 2.2, 0),
	)
	farFore := Union(
		Cone(42, 39, 6, 41.5, 49.5, 4.4),
		Cone(41.5, 49.5, 4.2, 42, 54.5, 3.8),
		Ellipse(42.6, 56.4, 4.6, 2.2, 0),
	)
	nearFore := Union(
		Cone(53, 40.5, 6.2, 52.5, 49.5, 5.0),
		Cone(52.5, 49.5, 4.8, 53, 56, 4.3),
		Ellipse(53.4, 58.5, 5.4, 2.7, 0),
	)
	body := Blend(3,
		Path([3]float64{0.8, 42.6, 0.8}, [3]float64{9, 38.6, 3}, [3]float64{18, 33, 6.5}), // the tail
		Ellipse(30.5, 33.2, 21.5, 12.4, 0),
		Ellipse(48.5, 37, 9, 9, 0), // the shoulders
	)
	frill := Ellipse(49, 19.5, 12, 14.8, -0.3)
	head := Cut(Blend(2,
		Poly(55.6, 27.6, 58, 26.8, 61, 26.5, 66, 28.6, 71, 31, 76, 34.5, 80, 38, 83.5, 40.2, 85.2, 43.5, 84.6, 47.4, 80.5, 49.2, 72, 49.4, 62, 49.2, 56.5, 46, 52.5, 41, 51.8, 36, 53, 31),
		Ellipse(60.5, 39, 7.8, 8, 0), // the cheek and the jaw's corner
	), Poly(44, 41.5, 57.4, 41.5, 58.6, 52, 44, 52)) // not the near shoulder, in front of it
	beak := Poly(75.6, 40.4, 79.4, 39.4, 83.4, 41, 85.4, 44, 84.8, 47.6, 81.4, 48.8, 77.4, 47.8, 75.6, 44.6)
	return &Species{
		Name:        "toro",
		GuideDetail: 0.6,
		Ink:         31,
		// The horns are painted again, smoothly: the guide's, above the
		// face, are cut out of the silhouette first.
		Remove: []SDF{
			Poly(62.5, 27.2, 64, 8, 88, 8, 88, 40, 82.6, 40.2, 78, 37.6, 74, 34.2, 70, 31.2, 66, 29),
		},
		Recolor: []Recolor{
			// The frill's white spikes are painted again, and its grey
			// specks are its rim.
			{Region: Cut(frill, Union(head, Ellipse(45.6, 15.6, 6.2, 6, 0), Ellipse(52.4, 24.8, 5.8, 5.6, 0))),
				From: []string{"ivory", "hide", "grey", "patch", "belly"}, To: "rim"},
			// A white spike of the guide's left on the back.
			{Region: Poly(30, 18, 40, 18, 40, 30, 30, 30), From: []string{"ivory"}, To: "hide"},
			// Below the flank the dark of the drawing is its shading.
			{Region: Below(0, 37, 90, 37), From: []string{"patch"}, To: "grey"},
		},
		Materials: []Material{
			{Name: "hide", Seeds: []uint32{0x768a9c, 0x5e7184, 0x485868, 0x90a3b2},
				Ramp: []uint32{0x161a26, 0x222838, 0x30384a, 0x3e4a5e, 0x4e5c72, 0x627288, 0x7a8ca0, 0x96a8ba}},
			{Name: "grey", Seeds: []uint32{0x7d8b92, 0x616e76, 0x47525a, 0x30383f, 0x9aa8ac, 0xb6c2c2},
				Ramp:   []uint32{0x17191e, 0x22262c, 0x2e343a, 0x3c434a, 0x4c555c, 0x5e6970, 0x748087, 0x8e9aa0},
				Region: Poly(14, 54, 60, 54, 60, 63, 14, 63), Only: []string{"ivory", "belly"}}, // the guide's nails are painted again
			{Name: "patch", Seeds: []uint32{0x34404e, 0x222a34},
				Ramp: []uint32{0x0c0f16, 0x131720, 0x1a1f2c, 0x222938, 0x2c3446, 0x364054, 0x424e64, 0x505e76}, Smooth: 0.55, Bias: 0.15},
			{Name: "belly", Seeds: []uint32{0x969282, 0x716e62, 0xbab6a2, 0xa09276, 0x625f54, 0x494840},
				Ramp: []uint32{0x2a2820, 0x3a372c, 0x4c4839, 0x605b48, 0x767058, 0x8c866c, 0xa49e84, 0xbcb69e}, Relief: 0.7},
			{Name: "red", Seeds: []uint32{0xa82c24, 0xc84430, 0x821e1e, 0x7a2626},
				Ramp: []uint32{0x2c0a0c, 0x420e10, 0x5c1414, 0x781c1a, 0x962620, 0xb03226, 0xc84630, 0xdc6040}, Relief: 0.6},
			{Name: "rim", Seeds: []uint32{0x601c20, 0x48141a, 0x300e14},
				Ramp: []uint32{0x16060a, 0x200a0e, 0x2c0e12, 0x3a1216, 0x4a161a, 0x5a1c20, 0x6c2428, 0x7e2e30}},
			{Name: "ivory", Seeds: []uint32{0xeee6cc, 0xfffaea, 0xd2c6a6},
				Ramp: []uint32{0x5a5444, 0x7a725e, 0x9a927a, 0xb8b096, 0xd2caae, 0xe6dec4, 0xf4eedc, 0xfcfaf0}, Relief: 0.4},
			// The two windows in the frill, ringed white, drawn round over the guide's.
			{Name: "ring", Ramp: []uint32{0x5a6266, 0x7a8486, 0x9aa4a4, 0xb8c2c0, 0xd0d8d6, 0xe2eae6, 0xf0f6f2, 0xfafefa}, Relief: 0.3,
				Region: Cut(Union(Ellipse(45.6, 15.6, 5.9, 5.7, -0.1), Ellipse(52.4, 24.8, 5.5, 5.3, -0.1)), head)},
			{Name: "spot", Ramp: []uint32{0x0a0e16, 0x10141e, 0x141a28, 0x1a2232, 0x202a3c, 0x263246, 0x2c3a50, 0x34445a}, Relief: 0.3,
				Region: Cut(Union(Ellipse(45.6, 15.6, 4.7, 4.5, -0.1), Ellipse(52.4, 24.8, 4.3, 4.1, -0.1)), head)},
			{Name: "beak", Ramp: []uint32{0x2a2010, 0x3e3016, 0x56441e, 0x6e5828, 0x887034, 0xa08a44, 0xb8a25a, 0xcebc78}, Relief: 0.3,
				Region: beak},
		},
		Parts: []Part{
			{Name: "farHind", Shape: farHind, Far: true, Round: 3.6, Scale: 0.85, Mat: "grey", Muscles: []Muscle{
				Mu(32.5, 43, 4.2, 4.8, 0, 1.0),
			}},
			{Name: "farFore", Shape: farFore, Far: true, Round: 3.6, Scale: 0.85, Mat: "grey", Muscles: []Muscle{
				Mu(42, 43, 4.2, 4.8, 0, 1.0),
			}},
			{Name: "body", Shape: body, Round: 8, Muscles: []Muscle{
				Mu(21, 31, 7.5, 6.5, 0, 1.6),     // the hip
				Mu(34, 30, 10, 7, 0, 1.0),        // the ribs
				Mu(47, 36, 5.5, 6.5, 0.2, 1.4),   // the shoulder
				Mu(32, 42, 12, 3, 0, 0.6),        // the belly
				Mu(10, 37.5, 7, 2.6, 0.45, 0.9),  // the tail
				Mu(26, 25, 9, 2.6, -0.05, 0.6),   // the long muscles of the back
				Mu(15.5, 35.5, 3.5, 4, 0.5, 0.8), // the tail's root
			}},
			// The neck, short and thick, all under the frill and the head: it
			// shows only when the head turns (the under-layer).
			{Name: "neck", Shape: Cone(45, 33, 8, 59.5, 40, 7), Round: 6, Onto: "body", Seam: Below(0, 999, 100, 999), Muscles: []Muscle{
				Mu(52, 32, 6.5, 4.5, 0.4, 1.0),
			}},
			{Name: "nearHind", Shape: nearHind, Round: 4.6, Onto: "body", Seam: Below(5, 42, 36, 42), Muscles: []Muscle{
				Mu(21, 41.5, 5, 6, 0, 1.8), // the thigh
				Mu(22.5, 50, 2.6, 2.3, 0, 0.6),
				Mu(22.3, 58, 3.8, 1.9, 0, 0.6), // the foot
			}},
			{Name: "nearFore", Shape: nearFore, Round: 4.6, Onto: "body", Seam: Below(43, 42.5, 62, 42.5), Muscles: []Muscle{
				Mu(53, 43, 4.6, 5.4, 0, 1.6), // the upper arm
				Mu(52.6, 50, 2.6, 2.3, 0, 0.6),
				Mu(53.6, 58, 3.6, 1.9, 0, 0.6),
			}},
			// The frill and the head in front of the near shoulder.
			{Name: "frill", Shape: frill, Round: 3.4, Soft: 0.6, Scale: 0.9, Muscles: []Muscle{
				Mu(47, 17, 7, 8, -0.3, 1.0), // a shallow dome
				Mu(41, 26, 3, 4, 0, 0.5),    // the thick bar at its root
			}},
			{Name: "head", Shape: head, Round: 4.6, Soft: 0.7, Scale: 0.75, Muscles: []Muscle{
				Mu(60.5, 40.5, 5, 4.5, 0, 1.7),      // the cheek and jaw muscle
				Mu(68.5, 34.6, 2.8, 1.3, -0.2, 1.4), // the brow over the eye
				Mu(74.5, 37.5, 2.6, 2.2, 0, 0.9),    // the base of the nose horn
				Mu(80.5, 43.5, 3.8, 3, 0, 0.9),      // the beak
				Mu(57, 30.5, 4, 2.5, 0, 0.8),        // the back of the skull
			}},
		},
		Skin: Skin{Flank: 2.1, Back: 3.4, Belly: 1.5, BackDepth: 5, BellyDepth: 4, Seed: 29},
		Osteoderms: []Row{
			{X0: 13, X1: 38, Off: 1.1, Gap: 3.2, R: 1.0, Crest: 0.3, Parts: []string{"body"}},
			{X0: 17, X1: 36, Off: 3.6, Gap: 4.2, R: 0.85, Parts: []string{"body"}},
		},
		Folds: []Fold{
			{Pts: []float64{57, 35, 56, 39.5, 56.6, 44}, N: 2, DX: -2, DY: 0.4, Depth: 1.2}, // the neck, under the frill
			{Pts: []float64{18.5, 52.5, 26, 52.3}, N: 3, DY: 1.4, Depth: 1.0},               // wrinkled legs
			{Pts: []float64{49, 52.5, 56.5, 52.3}, N: 3, DY: 1.4, Depth: 1.0},
			{Pts: []float64{29.5, 51, 36, 50.8}, N: 2, DY: 1.4, Depth: 0.7},
			{Pts: []float64{39, 51, 45.5, 50.8}, N: 2, DY: 1.4, Depth: 0.7},
			{Pts: []float64{65, 43.4, 71.5, 45, 77.6, 46.4}, N: 1, Depth: 2.2}, // the line of the mouth
		},
		Details: toroDetails,
	}
}

func toroDetails(p *Painter) {
	// The spikes round the frill's edge, from its root over the top to the
	// side of the head.
	p.RimSpikes(RimSpec{X: 49, Y: 19.5, RX: 12, RY: 14.8, Rot: -0.3, T0: 2.9, T1: 6.45, N: 14, In: 0.5, Out: 2.1, W: 2.3, Ramp: toroIvory})
	// The far brow horn, behind; the nose horn; the near brow horn.
	p.Horn(HornSpec{X0: 71.8, Y0: 32.4, X1: 85.4, Y1: 14.6, W: 2.7, Bend: 0.7, Ramp: toroIvory, Grooves: true, Shade: -0.3})
	p.Horn(HornSpec{X0: 76.4, Y0: 39.2, X1: 78.6, Y1: 31.4, W: 2.4, Bend: 0.4, Ramp: toroIvory})
	p.Horn(HornSpec{X0: 67.2, Y0: 31.8, X1: 80.0, Y1: 12.4, W: 3.3, Bend: 0.9, Ramp: toroIvory, Grooves: true})
	p.Eye(EyeSpec{X: 68.7, Y: 36.5, R: 0.95, Iris: greenIris, Lid: 0.3, Squint: 1.3})
	p.Nostril(80, 40.8, 0.7, 0.45, -0.3)
	// The nails of the near feet, then the far ones, darker.
	for _, n := range [][3]float64{{19.4, 61, 1}, {22.3, 61.2, 1.05}, {25.2, 60.9, 0.9}, {50.6, 61, 1}, {53.5, 61.2, 1.05}, {56.3, 60.8, 0.9}} {
		p.Nail(n[0], n[1], 2.5*n[2], 1.8*n[2], toroHoof, -0.15)
	}
	for _, n := range [][2]float64{{31.6, 58.4}, {34.3, 58.3}, {41.2, 58.4}, {43.9, 58.3}} {
		p.Nail(n[0], n[1], 2.1, 1.5, toroHoof, -0.45)
	}
}
