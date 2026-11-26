package dinopaint

// The little runner: dinos/thesc.png, 50 x 34, facing right, a
// Thescelosaurus. Its rig is "thesc" in internal/client/dinorigs.go: hips
// (27, 20) and (17.5, 20), knees (28.5, 25) and (18.5, 25), ankles (25, 28)
// and (15, 28), the balls of the feet (25.5, 31.5) and (15.5, 31.5), the
// toes' tips (31, 32.5) and (21, 32.5); the head turns about (33, 10.5),
// the beak opens about (41, 11) along the line to (47, 11.4), the tail
// swings about (12, 13).

var (
	thescEye  = []uint32{0x120a06, 0x24140a, 0x3e2610, 0x5c3a16, 0x7e521e, 0xa06e2a}
	thescClaw = []uint32{0x1e1a10, 0x3c3626, 0x645c44, 0x928a6c, 0xbeb696, 0xe0dabe}
)

func init() { Register(thesc()) }

func thesc() *Species {
	farLeg := Union(
		Cone(17.5, 18.8, 4.2, 18.5, 25, 2.4), // thigh
		Cone(18.5, 25, 2.2, 15, 28, 1.6),     // shin
		Cone(15, 28, 1.5, 15.5, 31.2, 1.2),   // metatarsus
		Cone(15.4, 31.6, 1.25, 20.6, 32.5, 0.7),
	)
	tail := Path([3]float64{-0.6, 11, 0.7}, [3]float64{6, 11.2, 1.7}, [3]float64{12.5, 12.2, 3.6})
	body := Blend(2.5,
		tail,
		Ellipse(21, 15.6, 11.6, 6.6, 0.04), // the trunk, deep at the hips
		Cone(31, 13.6, 4.6, 36, 11, 3.0),   // the chest and the neck
	)
	head := Blend(1.5,
		Cone(34, 11.6, 3, 38, 9.6, 2.6),
		Poly(36.4, 8.2, 37.6, 5.6, 40.4, 4.9, 43.2, 6.2, 45.6, 8.3, 47.4, 10.2, 47.5, 12.4, 45.6, 13.3, 41, 13.1, 37.5, 12.8, 35.4, 12.2),
	)
	jaw := Poly(37.8, 11.6, 46.6, 11.6, 47.3, 12.5, 45.6, 13.3, 41, 13.1, 37.8, 12.8)
	beak := Poly(45.2, 10.4, 46.8, 10, 47.7, 11, 47.5, 12.6, 46, 13.1, 45, 12.4)
	arm := Union(
		Cone(34.2, 16.2, 1.5, 35.6, 19.4, 1.0), // upper arm
		Cone(35.6, 19.4, 0.95, 36.8, 21.2, 0.62),
	)
	nearLeg := Union(
		Cone(27, 18.4, 4.7, 28.5, 25, 2.6), // the thigh, muscled for running
		Cone(28.5, 25, 2.4, 25, 28, 1.7),   // shin
		Cone(25, 28, 1.6, 25.5, 31.2, 1.3), // metatarsus
		Cone(25.4, 31.6, 1.3, 30.7, 32.5, 0.75),
	)
	return &Species{
		Name:        "thesc",
		GuideDetail: 0.45,
		Ink:         30,
		Materials: []Material{
			{Name: "hide", Seeds: []uint32{0x6a883b, 0x8faa55, 0x4a652a, 0x30441d, 0xb6ca7a, 0x1c2814, 0x223116},
				Ramp: []uint32{0x1c2a10, 0x2a3e16, 0x3a541e, 0x4c6a28, 0x628434, 0x7a9e42, 0x96b85a, 0xb6d07a}},
			{Name: "cream", Seeds: []uint32{0xb8b06e, 0xdcd594, 0xd4c488, 0x8a8450, 0xac9860},
				Ramp: []uint32{0x524a26, 0x6e6634, 0x8a8244, 0xa49c56, 0xbab268, 0xccc47a, 0xdcd690, 0xeae6aa}, Relief: 0.7, Smooth: 0.55, Bias: 0.08},
			{Name: "beak", Ramp: []uint32{0x221a0e, 0x342816, 0x4a3a20, 0x60502c, 0x786638, 0x907e48, 0xa8965a, 0xc0ae70}, Relief: 0.3,
				Region: beak},
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.4, Soft: 0.6, Scale: 0.8, Muscles: []Muscle{
				Mu(17.5, 20, 3.2, 3.6, 0.2, 1.1),   // the thigh
				Mu(16.8, 26.2, 1.4, 1.8, 0.8, 0.5), // the calf
			}},
			{Name: "body", Shape: body, Round: 5, Soft: 0.65, Muscles: []Muscle{
				Mu(24, 13.5, 7, 3.6, 0, 0.9),     // the hips
				Mu(19, 16.5, 6, 4.5, 0, 0.8),     // the belly and ribs
				Mu(31.5, 14.5, 3, 3.4, 0.3, 0.9), // the chest
				Mu(10, 12, 5.5, 1.8, 0, 0.7),     // the tail's muscle
				Mu(20, 10.6, 10, 1.4, 0, 0.5),    // the long muscles of the back
			}},
			{Name: "head", Shape: head, Round: 2.4, Soft: 0.5, Scale: 0.65, Onto: "body", Seam: Below(30, 15.5, 38, 13.5), Muscles: []Muscle{
				Mu(38.5, 9.2, 2.2, 2, 0, 0.9),      // the cheek
				Mu(41, 6.4, 1.6, 0.8, 0, 0.8),      // the brow over the eye
				Mu(44.6, 9.4, 1.8, 1.2, 0.3, 0.6),  // the snout
				Mu(35.4, 10.8, 1.8, 2.2, 0.4, 0.7), // the neck
			}},
			{Name: "jaw", Shape: jaw, Round: 1.2, Soft: 0.45, Scale: 0.6, Muscles: []Muscle{
				Mu(41, 12.4, 3, 0.8, 0, 0.4),
			}},
			{Name: "arm", Shape: arm, Round: 1.0, Soft: 0.4, Scale: 0.55, Onto: "body", Seam: Below(32, 17, 38, 17), Muscles: []Muscle{
				Mu(34.8, 17.6, 1.1, 1.6, 0.4, 0.4),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 3, Soft: 0.6, Onto: "body", Seam: Below(15, 21, 34, 19), Muscles: []Muscle{
				Mu(27, 19.5, 3.6, 4, 0.2, 1.3),                                    // the thigh
				Mu(29.6, 21.2, 1.4, 2.6, -0.4, 0.6),                               // its front
				Mu(26.8, 26.2, 1.4, 1.8, 0.8, 0.6),                                // the calf
				Mu(27.2, 32, 0.9, 0.6, 0, 0.3), Mu(29.2, 32.3, 0.8, 0.5, 0, 0.25), // knuckles
			}},
		},
		Skin: Skin{Flank: 1.35, Back: 2.0, Belly: 1.05, BackDepth: 3, BellyDepth: 3, Relief: 1.1, Seed: 53},
		Osteoderms: []Row{
			// A row of little scutes down the back, as the drawing's flecks.
			{X0: 6, X1: 33, Off: 0.6, Gap: 1.9, R: 0.55, Crest: 0.3, Parts: []string{"body"}},
		},
		Folds: []Fold{
			{Pts: []float64{35.6, 9, 35, 11, 35.2, 13}, N: 2, DX: 1.1, DY: -0.2, Depth: 0.8}, // the neck
			{Pts: []float64{27.6, 24.2, 29, 25.6}, N: 1, Depth: 0.8},                         // behind the knee
			{Pts: []float64{24.4, 29.2, 26.2, 29.1}, N: 2, DY: 1.1, Depth: 0.6},              // the metatarsus
			{Pts: []float64{27.6, 31.4, 27.6, 33}, N: 2, DX: 1.7, Depth: 0.8},                // the toes' knuckles
			{Pts: []float64{37.6, 11.6, 41, 11.7, 45.6, 11.5}, N: 1, Depth: 1.4},             // the line of the mouth
		},
		Details: thescDetails,
	}
}

func thescDetails(p *Painter) {
	p.Eye(EyeSpec{X: 41.2, Y: 7.7, R: 0.78, Iris: thescEye, Lid: 0.22, Squint: 1.2})
	p.Nostril(46.2, 9.9, 0.4, 0.28, -0.3)
	// The toes' claws, the far foot's darker; the little hand's.
	p.Claw(30.3, 32.3, 31.9, 33.1, 0.8, 0.2, thescClaw)
	p.Claw(20.3, 32.3, 21.9, 33.1, 0.7, 0.2, thescClaw[:4])
	p.Claw(36.5, 21, 37.4, 22.7, 0.55, 0.2, thescClaw)
	p.Claw(36, 21.2, 36.3, 22.8, 0.5, 0.15, thescClaw[:4])
}
