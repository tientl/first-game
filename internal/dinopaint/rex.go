package dinopaint

// The tyrant: dinos/rex.png, 108 x 87, facing right. Its rig is "rex" in
// internal/client/dinorigs.go: hips (53, 47) and (40, 48), knees (60, 65.5)
// and (41, 60), ankles (49.5, 72) and (33, 71), the neck turning about
// (70, 39), the jaw opening about (84, 34) along the line to (103, 34).

// The colours shared by the hunters.
var (
	ivory     = []uint32{0x6e5c46, 0x9c8a6a, 0xc4b48e, 0xe2d6b4, 0xf6efdb, 0xfffbee}
	darkHorn  = []uint32{0x17110e, 0x2a211b, 0x3f342a, 0x5d4f40, 0x81705a, 0xa69479}
	clawHorn  = []uint32{0x1c1612, 0x3a3029, 0x5e5246, 0x8a7d6c, 0xb5aa96, 0xddd4c0}
	amberIris = []uint32{0x5a2206, 0x8e420a, 0xc66e10, 0xeea024, 0xffc94a, 0xffe58e}
)

func init() { Register(rex()) }

func rex() *Species {
	// The masses, in depth order.
	farLeg := Union(
		Cone(41, 49, 8.5, 40.5, 60, 5.5),      // thigh
		Cone(40.5, 60, 5.2, 34, 71, 3.4),      // shin
		Cone(34, 71, 3.4, 35, 81.5, 2.8),      // metatarsus
		Cone(35, 83.2, 2.8, 41.2, 84.9, 1.35), // toes
	)
	body := Blend(3,
		Path([3]float64{-1, 37.2, 0.8}, [3]float64{12, 36.6, 2.8}, [3]float64{24, 37, 5.8}, [3]float64{35, 39.5, 9.5}), // the tail
		Ellipse(51, 41.5, 20, 12.2, 0), // the barrel of the body
		Ellipse(67, 46, 9, 9.5, 0),     // the chest
		Cone(63, 38, 10, 80, 26, 8.5),  // the neck
		Poly(77, 21, 84, 14, 89, 12.5, 95, 14.5, 102, 15.5, 107, 19, 108, 27, 107, 33, 104, 40, 84, 40.5, 78, 36), // the head
	)
	jaw := Poly(82.5, 33.7, 105.5, 33.4, 105, 36.5, 102.5, 39.6, 86, 40.2, 81.6, 38.5) // the lower jaw
	arm := Union(
		Cone(70, 44, 2.9, 72.6, 50.2, 2.2), // upper arm
		Cone(72.6, 50.2, 2.1, 77.6, 52.6, 1.6),
		Cone(77.6, 52.6, 1.5, 80.2, 53.6, 1.0), // the hand
	)
	nearLeg := Union(
		Cone(49, 47, 10.8, 57, 61, 5.2), // the great drumstick of the thigh
		Cone(58, 62, 5, 50, 72, 3.4),    // shin
		Cone(50, 72, 3.4, 51.5, 81.5, 2.9),
		Cone(51.5, 83.2, 3.0, 54.5, 84.6, 2.1), // the ball of the foot
		Cone(54, 84.3, 2.0, 61.6, 85.1, 1.15),  // the near toe
	)
	nearToe2 := Cone(53.5, 82.6, 1.75, 60.2, 83.3, 1.0) // the middle toe, a little behind
	return &Species{
		Name:        "rex",
		GuideDetail: 0.3,
		Ink:         44,
		Add:         []SDF{Cone(54, 84.3, 2.0, 61.6, 85.1, 1.15), nearToe2, Cone(35, 83.2, 2.8, 41.2, 84.9, 1.35)},
		// The tip of the far toe reaches under the near leg in the guide; the
		// rig would take it for the near foot's.
		Remove: []SDF{Poly(44.6, 81.6, 48.4, 81.6, 48.4, 88, 44.6, 88)},
		Materials: []Material{
			{Name: "back", Seeds: []uint32{0x6b5e38, 0x857748, 0xa0915e, 0x918355, 0x7b6e48},
				Ramp: []uint32{0x2a221c, 0x3b3224, 0x4e442c, 0x625735, 0x776a3f, 0x8c7e4a, 0xa39459, 0xbcad72}},
			{Name: "stripe", Seeds: []uint32{0x3d3522, 0x4d4328, 0x53482c, 0x443b24},
				Ramp: []uint32{0x1f1915, 0x2a221a, 0x352c1f, 0x403624, 0x4c412a, 0x594d31, 0x675a39, 0x776944}, Smooth: 0.6, Bias: 0.14},
			{Name: "flank", Seeds: []uint32{0x84663c, 0xa2834e, 0xbc9f68, 0x664f2e, 0x775c37},
				Ramp:   []uint32{0x2c1f19, 0x3f2c1f, 0x553c26, 0x6c4e2f, 0x85633a, 0x9f7a48, 0xb89259, 0xd0ad72},
				Region: Poly(46, 66, 66, 66, 66, 88, 46, 88), Only: []string{"stripe", "back"}}, // no stripes on the near shin and foot
			{Name: "belly", Seeds: []uint32{0x705a42, 0x98805c, 0xbda67c, 0xcfc4a4, 0xd2ba86, 0xd8c79c, 0xf2ecd6},
				Ramp: []uint32{0x2f231f, 0x43322a, 0x5a4636, 0x725b45, 0x8b7356, 0xa48b68, 0xbba27e, 0xcfb995}, Relief: 0.7},
			{Name: "far", Seeds: []uint32{0x3f2f21, 0x453423, 0x50402f, 0x4a3a2c, 0x433a2f, 0x5b442b, 0x644b2e},
				Ramp: []uint32{0x2a1f18, 0x36281d, 0x433223, 0x523d2a, 0x624a33, 0x74583c, 0x886848, 0x9a7a58}},
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 4, Scale: 0.8, Mat: "far", Muscles: []Muscle{
				Mu(41, 52, 6, 7, 0, 1.6),                                           // the thigh
				Mu(38, 64.5, 3, 3.5, 0.5, 0.8),                                     // the calf
				Mu(37.2, 83.8, 1.2, 1.0, 0, 0.5), Mu(39.6, 84.2, 1.0, 0.9, 0, 0.4), // knuckles
			}},
			{Name: "body", Shape: body, Round: 9, RoundAt: func(x, y float64) float64 { return lerp(7.5, 4.5, smoothstep(74, 84, x)) }, Muscles: []Muscle{
				Mu(81, 24.5, 4.5, 5.5, 0.3, 2.3),   // the jaw muscle behind the eye
				Mu(90.5, 27.8, 6.5, 2.4, 0, 1.1),   // the cheek
				Mu(100, 24, 6, 4.5, 0, 1.2),        // the snout
				Mu(95.5, 21.5, 3.2, 1.8, 0, -0.9),  // the hollow in front of the eye
				Mu(86, 17.2, 3.4, 1.6, 0, 2.0),     // the brow ridge over the eye
				Mu(91.8, 17.4, 2.6, 1.3, 0, 1.3),   // the ridge in front of it
				Mu(99, 17.6, 5, 1.2, 0, 0.6),       // the rough top of the snout
				Mu(74, 31, 6, 7.5, 0.6, 1.4),       // the neck
				Mu(78.5, 40, 3.2, 4.5, 0.2, 0.9),   // the throat
				Mu(66.5, 39.5, 5.5, 4.2, 0.2, 1.3), // the shoulder
				Mu(69.5, 48.5, 6, 5.5, 0, 1.5),     // the chest
				Mu(57, 41, 11, 7.5, 0, 0.9),        // the ribs
				Mu(46, 33.5, 15, 3.2, 0, 0.8),      // the long muscles of the back
				Mu(31, 40.5, 9, 4.6, -0.12, 1.5),   // the tail's great muscle, to the thigh
				Mu(17, 37.5, 9, 2.6, -0.05, 0.8),
			}},
			{Name: "jaw", Shape: jaw, Round: 2.6, Soft: 0.6, Scale: 0.8, Muscles: []Muscle{
				Mu(91, 36.5, 8, 2.2, 0.05, 0.7),
			}},
			{Name: "arm", Shape: arm, Round: 2.2, Soft: 0.5, Scale: 0.65, Mat: "flank", Onto: "body", Seam: Below(64, 46.5, 76, 44.5), Muscles: []Muscle{
				Mu(70.8, 46.2, 2.2, 2.8, 0.3, 0.8),
			}},
			{Name: "nearToe2", Shape: nearToe2, Round: 1.6, Soft: 0.5, Scale: 0.7, Muscles: []Muscle{
				Mu(55.6, 82.6, 1.1, 0.9, 0, 0.4), Mu(58, 82.9, 1.0, 0.8, 0, 0.35),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 7, Onto: "body", Seam: Below(30, 50, 66, 44), Muscles: []Muscle{
				Mu(48.5, 46.5, 8, 9.5, -0.3, 2.3),                                                                   // the drumstick
				Mu(56, 52.5, 4, 7.5, -0.35, 1.3),                                                                    // the front of the thigh
				Mu(42.5, 53.5, 3.6, 6, 0.2, 0.9),                                                                    // the back of the thigh
				Mu(59, 63.5, 2.2, 2.2, 0, 0.7),                                                                      // the knee
				Mu(53.5, 66, 3.4, 3.4, 0.6, 1.1),                                                                    // the calf
				Mu(50, 72, 2, 2, 0, 0.4),                                                                            // the ankle
				Mu(56, 84.4, 1.3, 1.1, 0, 0.5), Mu(58.6, 84.6, 1.2, 1.0, 0, 0.45), Mu(60.8, 84.8, 1.0, 0.9, 0, 0.4), // knuckles
			}},
		},
		Skin: Skin{Flank: 1.75, Back: 3.6, Belly: 1.35, BackDepth: 5, BellyDepth: 5, Seed: 11},
		Osteoderms: []Row{
			{X0: 24, X1: 84, Off: 1.0, Gap: 2.6, R: 1.0, Crest: 0.45, Parts: []string{"body"}},
			{X0: 32, X1: 74, Off: 2.9, Gap: 3.3, R: 0.75, Parts: []string{"body"}},
		},
		Folds: []Fold{
			{Pts: []float64{78.8, 35.5, 77.6, 39, 77.2, 43}, N: 3, DX: -2.3, DY: 0.7, Depth: 1.6},   // the throat
			{Pts: []float64{79.5, 21.5, 78.6, 25.5, 78.4, 29}, N: 2, DX: -2.6, DY: 1.2, Depth: 1.2}, // behind the skull
			{Pts: []float64{55, 63.6, 57.8, 66.6}, N: 2, DY: 1.4, Depth: 1.3},                       // behind the knee
			{Pts: []float64{48.6, 74.5, 51.6, 74.2}, N: 4, DY: 1.9, Depth: 0.7},                     // across the metatarsus
			{Pts: []float64{57.3, 83.5, 57.1, 86}, N: 2, DX: 2.4, Depth: 1.2},                       // between the knuckles
			{Pts: []float64{56.8, 81.7, 56.7, 83.6}, N: 2, DX: 2.3, Depth: 0.9},
			{Pts: []float64{32.4, 74, 35.6, 73.8}, N: 4, DY: 1.9, Depth: 0.6},
			{Pts: []float64{38.4, 82.6, 38.3, 85.8}, N: 2, DX: 2.2, Depth: 0.9},
		},
		Details: rexDetails,
	}
}

func rexDetails(p *Painter) {
	p.Teeth(TeethSpec{X0: 85.3, Y0: 32.1, X1: 104.4, Y1: 31.8, Bottom: 34, N: 10, Len: 1.95,
		Ramp: ivory, Gap: 0x2a1311, Seed: 5})
	p.Lip([]float64{84.6, 32.2, 95, 32.05, 104.6, 31.8}, 3)
	p.Nostril(103.6, 21.4, 0.95, 0.55, -0.35)
	p.Eye(EyeSpec{X: 88.1, Y: 20.7, R: 1.2, Iris: amberIris, Slit: true, Lid: 0.28, Squint: 1.35})
	// Claws: on the toes of both feet, the dewclaw and the two fingers. A
	// foot's claws stay inside the columns the rig cuts its leg from
	// (45-64 for the near leg, 27-44 for the far one).
	p.Claw(41.0, 84.5, 44.4, 86.0, 1.3, 0.35, clawHorn[:4])
	p.Claw(59.9, 82.7, 62.8, 84.1, 1.2, 0.3, clawHorn[:5])
	p.Claw(61.2, 84.5, 64.5, 86.0, 1.5, 0.45, clawHorn)
	p.Claw(49.6, 80.2, 48.1, 82.2, 0.8, 0.2, clawHorn[:5])
	p.Claw(79.6, 53.2, 81.6, 55.2, 0.7, 0.3, clawHorn)
	p.Claw(78.6, 54.2, 79.8, 56.6, 0.6, 0.2, clawHorn)
}
