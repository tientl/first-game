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

func init() {
	Register(rex())
	// Its legs, in the rig's order, cut as painted.
	RegisterLegs("rex", LegMask{{"nearLeg", "nearToe2"}, {"farLeg", "farToe2"}})
}

func rex() *Species {
	// The feet: the ball of the foot, then two toes seen from the side, the
	// outer one in front and low, the middle one behind it and longer, each
	// in three knuckles. The far foot is the same, smaller and darker.
	nearBall := Ellipse(51.6, 83.4, 3.0, 2.6, 0.1)
	nearToe := Path([3]float64{53.2, 84.2, 1.75}, [3]float64{56.6, 84.6, 1.45}, [3]float64{59.6, 85.0, 1.15}, [3]float64{61.9, 85.3, 0.85})
	nearToe2 := Path([3]float64{53.4, 82.2, 1.45}, [3]float64{56.8, 82.0, 1.2}, [3]float64{59.9, 82.5, 0.95}, [3]float64{62.3, 83.3, 0.7})
	farBall := Ellipse(35.2, 83.4, 2.8, 2.5, 0.1)
	farToe := Path([3]float64{36.8, 84.2, 1.6}, [3]float64{39.8, 84.6, 1.3}, [3]float64{42.3, 85.1, 0.85})
	farToe2 := Path([3]float64{37, 82.2, 1.35}, [3]float64{40, 82.1, 1.1}, [3]float64{42.5, 82.8, 0.7})
	finger1 := Cone(78.0, 52.2, 0.75, 80.1, 52.9, 0.55) // the near finger
	finger2 := Cone(77.5, 52.9, 0.72, 78.4, 54.8, 0.52) // the far one, behind
	feet := Union(nearBall, nearToe, nearToe2, farBall, farToe, farToe2)
	hand := Union(Cone(76.2, 52.1, 1.3, 78.2, 52.5, 1.05), finger1, finger2)
	// The masses, in depth order.
	farLeg := Union(
		Cone(41, 49, 8.5, 40.5, 60, 5.5), // thigh
		Cone(40.5, 60, 5.2, 34, 71, 3.4), // shin
		Cone(34, 71, 3.4, 35, 81.5, 2.8), // metatarsus
		farBall, farToe,
	)
	body := Blend(3,
		Path([3]float64{-1, 37.2, 0.8}, [3]float64{12, 36.6, 2.8}, [3]float64{24, 37, 5.8}, [3]float64{35, 39.5, 9.5}), // the tail
		Ellipse(51, 41.5, 20, 12.2, 0), // the barrel of the body
		Ellipse(67, 46, 9, 9.5, 0),     // the chest
		Cone(63, 38, 10, 80, 26, 8.5),  // the neck
		Poly(77, 21, 84, 14, 89, 12.5, 95, 14.5, 102, 15.5, 107, 19, 108, 27, 107, 33, 104, 40, 84, 40.5, 78, 36), // the head
	)
	jaw := Poly(82.5, 33.7, 105.5, 33.4, 105, 36.5, 102.5, 39.6, 86, 40.2, 81.6, 38.5) // the lower jaw
	// The little arm: upper arm, elbow, forearm, and a hand of two fingers,
	// spread, each with its claw.
	arm := Union(
		Cone(69.8, 44.2, 2.7, 71.3, 50.2, 2.0),   // upper arm
		Cone(71.3, 50.3, 1.85, 76.2, 52.1, 1.35), // forearm
		Cone(76.2, 52.1, 1.3, 78.2, 52.5, 1.05),  // the hand
		finger1,
	)
	nearLeg := Union(
		Cone(49, 47, 10.8, 57, 61, 5.2), // the great drumstick of the thigh
		Cone(58, 62, 5, 50, 72, 3.4),    // shin
		Cone(50, 72, 3.4, 51.5, 81.5, 2.9),
		nearBall, nearToe,
	)
	return &Species{
		Name:        "rex",
		GuideDetail: 0.3,
		Ink:         44,
		// The guide's toes and hand are drawn again, toe by toe and finger by
		// finger.
		Add: []SDF{feet, hand},
		Remove: []SDF{
			Cut(Poly(53.8, 78.5, 67, 78.5, 67, 89, 53.8, 89), feet),
			Cut(Poly(37.6, 78.8, 44.6, 78.8, 44.6, 89, 37.6, 89), feet),
			Cut(Poly(76.6, 50, 84, 50, 84, 58.5, 76.6, 58.5), hand),
			// The tip of the far toe reaches under the near leg in the guide;
			// the rig would take it for the near foot's.
			Poly(44.6, 81.6, 48.4, 81.6, 48.4, 88, 44.6, 88),
		},
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
			{Name: "farToe2", Shape: farToe2, Far: true, Round: 1.3, Soft: 0.45, Scale: 0.6, Mat: "far", Muscles: []Muscle{
				Mu(38.5, 81.9, 1.3, 0.9, 0, 0.5), Mu(41.2, 82.3, 1.0, 0.7, 0, 0.4),
			}},
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 4, Scale: 0.8, Mat: "far", Muscles: []Muscle{
				Mu(41, 52, 6, 7, 0, 1.6),                                           // the thigh
				Mu(38, 64.5, 3, 3.5, 0.5, 0.8),                                     // the calf
				Mu(35.2, 83.6, 2.2, 1.8, 0, 0.6),                                   // the ball of the foot
				Mu(38.3, 84.3, 1.4, 1.1, 0, 0.6), Mu(41.1, 84.8, 1.1, 0.8, 0, 0.5), // knuckles
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
			{Name: "finger2", Shape: finger2, Round: 0.7, Soft: 0.35, Scale: 0.5, Mat: "flank", Muscles: []Muscle{
				Mu(78, 53.8, 0.6, 0.8, 0.5, 0.3),
			}},
			{Name: "arm", Shape: arm, Round: 1.8, Soft: 0.4, Scale: 0.6, Mat: "flank", Onto: "body", Seam: Below(64, 46.5, 76, 44.5), Muscles: []Muscle{
				Mu(70.6, 46.4, 2.0, 2.8, 0.3, 0.9),   // the upper arm
				Mu(73.8, 51.3, 2.4, 1.2, 0.35, 0.55), // the forearm
				Mu(71.4, 50.2, 1.1, 1.1, 0, 0.35),    // the elbow
				Mu(79.0, 52.6, 0.9, 0.5, 0.35, 0.25), // the finger
			}},
			{Name: "nearToe2", Shape: nearToe2, Round: 1.4, Soft: 0.45, Scale: 0.6, Muscles: []Muscle{
				Mu(55.1, 81.9, 1.5, 1.0, 0, 0.55), Mu(58.4, 82.1, 1.3, 0.85, 0, 0.45), Mu(61.1, 82.8, 0.9, 0.6, 0, 0.3),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 7, Onto: "body", Seam: Below(30, 50, 66, 44), Muscles: []Muscle{
				Mu(48.5, 46.5, 8, 9.5, -0.3, 2.3),                                                                     // the drumstick
				Mu(56, 52.5, 4, 7.5, -0.35, 1.3),                                                                      // the front of the thigh
				Mu(42.5, 53.5, 3.6, 6, 0.2, 0.9),                                                                      // the back of the thigh
				Mu(59, 63.5, 2.2, 2.2, 0, 0.7),                                                                        // the knee
				Mu(53.5, 66, 3.4, 3.4, 0.6, 1.1),                                                                      // the calf
				Mu(50, 72, 2, 2, 0, 0.4),                                                                              // the ankle
				Mu(51.6, 83.6, 2.4, 2.0, 0, 0.7),                                                                      // the ball of the foot
				Mu(54.9, 84.3, 1.6, 1.3, 0, 0.6), Mu(58.1, 84.8, 1.4, 1.1, 0, 0.55), Mu(60.8, 85.1, 1.0, 0.8, 0, 0.4), // knuckles
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
			{Pts: []float64{56.6, 83.3, 56.6, 85.9}, N: 2, DX: 3.0, DY: 0.35, Depth: 1.3},           // between the knuckles of the near toe
			{Pts: []float64{56.8, 80.9, 56.8, 82.9}, N: 2, DX: 3.1, DY: 0.4, Depth: 1.0},            // and of the middle one
			{Pts: []float64{39.8, 83.5, 39.8, 85.7}, N: 1, Depth: 1.1},
			{Pts: []float64{40, 81.2, 40, 82.9}, N: 1, Depth: 0.8},
			{Pts: []float64{48.6, 82.6, 50.4, 85.6}, N: 1, Depth: 0.8}, // the heel's pad
			{Pts: []float64{32.4, 74, 35.6, 73.8}, N: 4, DY: 1.9, Depth: 0.6},
			{Pts: []float64{71.4, 49.6, 72.6, 51.4}, N: 1, Depth: 0.9}, // the elbow
			{Pts: []float64{76.3, 51.2, 76.5, 53.0}, N: 1, Depth: 0.9}, // the wrist
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
	// The far foot's, darker.
	p.Claw(42.0, 84.8, 44.8, 86.1, 1.3, 0.35, clawHorn[:5])
	p.Claw(42.2, 82.6, 44.9, 83.8, 1.1, 0.3, clawHorn[:4])
	p.Claw(32.7, 79.6, 31.3, 81.6, 0.9, 0.25, clawHorn[:4])
	// The near foot's: the middle toe's, the near toe's, the dewclaw.
	p.Claw(62.0, 83.1, 64.9, 84.3, 1.2, 0.35, clawHorn[:5])
	p.Claw(61.6, 85.0, 64.8, 86.3, 1.45, 0.4, clawHorn)
	p.Claw(49.2, 79.6, 47.6, 81.9, 1.0, 0.3, clawHorn)
	// The fingers'.
	p.Claw(78.3, 54.7, 78.7, 56.9, 0.8, 0.3, clawHorn)
	p.Claw(80.0, 52.7, 81.9, 54.6, 0.85, 0.35, clawHorn)
}
