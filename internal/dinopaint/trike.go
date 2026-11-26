package dinopaint

// The three-horn: dinos/trike.png, 78 x 56, facing right. Its rig is
// "trike" in internal/client/dinorigs.go: legs near hind (hip 20, 34), far
// hind (30, 35), far fore (39, 35), near fore (48.5, 34); the head, frill
// and horns turn about (44.6, 31.5); the beak opens about (62, 39.5) along
// the line to (71.5, 41.6); the tail turns about (15, 31).

var (
	hornIvory = []uint32{0x4a3e30, 0x7a6a54, 0xa8987a, 0xcdbf9e, 0xe8dfc4, 0xfaf5e6}
	beakHorn  = []uint32{0x2a2418, 0x433a26, 0x5e5334, 0x7a6e44, 0x968a58, 0xb0a670}
	hoofHorn  = []uint32{0x3a3024, 0x6a5c48, 0x9c8e74, 0xc8bc9e, 0xe6dec6}
	greenIris = []uint32{0x3a2a08, 0x6a4a0a, 0x9c7210, 0xcaa024, 0xeed054, 0xfff0a0}
)

func init() {
	Register(trike())
	// The shoulders and the neck under the frill, for when the head turns.
	RegisterUnder("trike", &Under{Parts: []string{"frill", "head"}, Mat: "hide"})
}

func trike() *Species {
	nearHind := Union(
		Cone(20, 33, 6.2, 21, 44, 3.8),
		Cone(21, 44, 3.6, 20.5, 51, 3.1),
		Ellipse(21.2, 53.6, 4.6, 2.5, 0), // the foot, broad as an elephant's
	)
	farHind := Union(
		Cone(30, 34, 5, 30.5, 44, 3.3),
		Cone(30.5, 44, 3.2, 30, 49.5, 2.8),
		Ellipse(31, 51.6, 3.7, 1.9, 0),
	)
	farFore := Union(
		Cone(39, 34, 5, 38.5, 44, 3.3),
		Cone(38.5, 44, 3.2, 39, 50, 2.8),
		Ellipse(40, 52, 3.6, 1.9, 0),
	)
	nearFore := Union(
		Cone(48.5, 34.5, 5.6, 48, 44, 3.8),
		Cone(48, 44, 3.6, 48.5, 51, 3.1),
		Ellipse(49.6, 53.7, 4.4, 2.4, 0),
	)
	body := Blend(3,
		Path([3]float64{-0.5, 38.8, 0.7}, [3]float64{8, 36.4, 2.8}, [3]float64{16, 32.5, 6}), // the tail
		Ellipse(31, 30.2, 18.5, 11.6, 0),
		Ellipse(46, 33, 8, 8.5, 0), // the shoulders
	)
	frill := Union(Ellipse(49.5, 20.6, 10.8, 11.8, 0.15), Ellipse(45, 25.8, 5, 5.4, 0))
	head := Cut(Blend(2,
		Poly(50, 27, 56, 24.4, 62, 25.6, 68, 28.5, 73, 33, 77.5, 36, 78.5, 40.5, 76, 43.5, 66, 44.2, 57, 43, 51, 39.5),
		Ellipse(58, 34, 7.5, 7.5, 0), // the cheek and the jaw's corner
	), Poly(43, 34.5, 52.6, 34.5, 53.2, 46, 43, 46)) // not the near shoulder, in front of it
	beak := Poly(70.2, 35.2, 74, 34.6, 78.6, 37.8, 78.8, 41.6, 76.4, 43.6, 72, 43.2, 70.4, 40.6)
	return &Species{
		Name:        "trike",
		GuideDetail: 0.6,
		Ink:         28,
		// The horns are painted again, smoothly: the guide's, above the
		// face, are cut out of the silhouette first.
		Remove: []SDF{
			Poly(60.8, 24.8, 62, 9, 80, 9, 80, 34.4, 73, 32.6, 68, 28.3, 62, 25.5),
		},
		// The dark bands are the markings across the back; elsewhere the
		// guide's dark browns are its shading, and on the frill its rim.
		Recolor: []Recolor{
			{Region: frill.Grow(0.5), From: []string{"band"}, To: "rim"},
			{Region: Below(0, 32.5, 80, 32.5), From: []string{"band"}, To: "hide"},
		},
		Materials: []Material{
			{Name: "hide", Seeds: []uint32{0x783f1c, 0x7c4020, 0x9c5826, 0x9a562c, 0xb26e3c, 0xbc7636, 0xd4964e},
				Ramp:   []uint32{0x2a140e, 0x3e1c12, 0x56281a, 0x723820, 0x8e4a26, 0xa85e30, 0xc2783e, 0xd89a58},
				Region: Poly(14, 49, 56, 49, 56, 57, 14, 57), Only: []string{"ivory", "belly"}}, // the guide's nails are painted again
			{Name: "band", Seeds: []uint32{0x542a14, 0x5c2c18},
				Ramp: []uint32{0x1e0e0a, 0x2c140e, 0x3a1a10, 0x4a2214, 0x5a2c18, 0x6c361e, 0x7e4224, 0x905030}, Smooth: 0.55, Bias: 0.14},
			{Name: "belly", Seeds: []uint32{0xab9162, 0xa89a7c, 0x846c44},
				Ramp: []uint32{0x3a2e22, 0x4e3f2c, 0x665438, 0x7e6a46, 0x978256, 0xae9868, 0xc2ae7e, 0xd4c294}, Relief: 0.7},
			{Name: "gold", Seeds: []uint32{0xd29c1c, 0xb07a12, 0xecbc30},
				Ramp: []uint32{0x4a2a08, 0x6a3c0a, 0x8c540e, 0xac6c12, 0xc8861a, 0xdca226, 0xecbc3c, 0xf6d462}, Relief: 0.5},
			{Name: "rim", Seeds: []uint32{0x641c12, 0x842818},
				Ramp: []uint32{0x240a08, 0x36100c, 0x4a1610, 0x5e1c12, 0x742416, 0x8a2e1a, 0xa03c22, 0xb44e2c}},
			{Name: "ivory", Seeds: []uint32{0xeee2bc, 0xfffaea},
				Ramp: []uint32{0x6a5a44, 0x8c7c62, 0xaa9a7c, 0xc6b896, 0xdcd0b0, 0xece4ca, 0xf8f2e0, 0xfffcf2}, Relief: 0.4},
			// The eyespots on the frill, drawn round over the guide's.
			{Name: "ring", Ramp: []uint32{0x6a5a44, 0x8c7c62, 0xaa9a7c, 0xc6b896, 0xdcd0b0, 0xece4ca, 0xf8f2e0, 0xfffcf2}, Relief: 0.3,
				Region: Union(Ellipse(47.2, 17.2, 2.9, 2.6, 0.2), Ellipse(51.4, 23.6, 2.6, 2.3, 0.2))},
			{Name: "spot", Ramp: []uint32{0x140806, 0x1e0c08, 0x28100a, 0x34160e, 0x401c12, 0x4c2216, 0x582a1a, 0x643220}, Relief: 0.3,
				Region: Union(Ellipse(47.2, 17.2, 1.9, 1.65, 0.2), Ellipse(51.4, 23.6, 1.65, 1.4, 0.2))},
			{Name: "beak", Ramp: []uint32{0x241e14, 0x3a3222, 0x524830, 0x6c613e, 0x877b50, 0xa29664, 0xbab07c, 0xcec694}, Relief: 0.3,
				Region: beak},
		},
		Parts: []Part{
			{Name: "farHind", Shape: farHind, Far: true, Round: 3.5, Scale: 0.85, Mat: "band", Muscles: []Muscle{
				Mu(30, 38, 4, 4.5, 0, 1.0),
			}},
			{Name: "farFore", Shape: farFore, Far: true, Round: 3.5, Scale: 0.85, Mat: "band", Muscles: []Muscle{
				Mu(39, 38, 4, 4.5, 0, 1.0),
			}},
			{Name: "body", Shape: body, Round: 8, Muscles: []Muscle{
				Mu(20, 30, 7, 6, 0, 1.5),       // the hip
				Mu(33, 29, 10, 7, 0, 1.0),      // the ribs
				Mu(45, 33, 5, 6, 0.2, 1.4),     // the shoulder
				Mu(31, 38.5, 12, 3, 0, 0.6),    // the belly
				Mu(10, 35.5, 7, 2.4, 0.3, 0.9), // the tail
			}},
			// The neck, short and thick, all under the frill and the head: it shows
			// only when the head turns (the under-layer).
			{Name: "neck", Shape: Cone(44, 29.5, 7.5, 56, 36, 6.5), Round: 6, Onto: "body", Seam: Below(0, 999, 100, 999), Muscles: []Muscle{
				Mu(50, 29, 6, 4, 0.4, 1.0),
			}},
			{Name: "nearHind", Shape: nearHind, Round: 4.5, Onto: "body", Seam: Below(5, 38.5, 35, 38.5), Muscles: []Muscle{
				Mu(20, 37, 4.6, 5.5, 0, 1.7), // the thigh
				Mu(21, 44.5, 2.4, 2.2, 0, 0.6),
				Mu(21.4, 53, 3.6, 1.8, 0, 0.6), // the foot
			}},
			{Name: "nearFore", Shape: nearFore, Round: 4.5, Onto: "body", Seam: Below(40, 38.5, 60, 38.5), Muscles: []Muscle{
				Mu(48.4, 37, 4.2, 5.2, 0, 1.6), // the upper arm
				Mu(48, 44.5, 2.4, 2.2, 0, 0.6), // the elbow
				Mu(49.6, 53, 3.4, 1.8, 0, 0.6),
			}},
			// The frill and the head in front of the near shoulder.
			{Name: "frill", Shape: frill, Round: 3.2, Soft: 0.6, Scale: 0.9, Muscles: []Muscle{
				Mu(48, 18, 6, 6.5, 0, 0.9), // a shallow dome
			}},
			{Name: "head", Shape: head, Round: 4.5, Soft: 0.7, Scale: 0.75, Muscles: []Muscle{
				Mu(58.5, 36, 4.5, 4, 0, 1.6),     // the cheek and jaw muscle
				Mu(63.5, 30.6, 2.6, 1.3, 0, 1.3), // the brow over the eye
				Mu(68.5, 33, 2.5, 2.2, 0, 0.9),   // the base of the nose horn
				Mu(74, 38.5, 3.5, 2.6, 0, 0.9),   // the beak
			}},
		},
		Skin: Skin{Flank: 2.1, Back: 3.4, Belly: 1.5, BackDepth: 5, BellyDepth: 4, Seed: 23},
		Osteoderms: []Row{
			{X0: 12, X1: 40, Off: 1.1, Gap: 3.2, R: 1.05, Crest: 0.35, Parts: []string{"body"}},
			{X0: 16, X1: 38, Off: 3.6, Gap: 4.2, R: 0.9, Parts: []string{"body"}},
		},
		Folds: []Fold{
			{Pts: []float64{52.5, 31, 51.6, 35.5, 52.2, 39.5}, N: 2, DX: -1.9, DY: 0.4, Depth: 1.2}, // the neck, under the frill
			{Pts: []float64{18, 47.5, 24, 47.3}, N: 3, DY: 1.4, Depth: 1.0},                         // wrinkled legs
			{Pts: []float64{45.5, 47.5, 51, 47.3}, N: 3, DY: 1.4, Depth: 1.0},
			{Pts: []float64{28, 46.5, 33, 46.3}, N: 2, DY: 1.4, Depth: 0.7},
			{Pts: []float64{36.5, 46.5, 41.5, 46.3}, N: 2, DY: 1.4, Depth: 0.7},
			{Pts: []float64{60.5, 39.2, 66, 40.7, 71.5, 41.5}, N: 1, Depth: 2.2}, // the line of the mouth
		},
		Details: trikeDetails,
	}
}

func trikeDetails(p *Painter) {
	// The far brow horn, behind; the nose horn; the near brow horn.
	p.Horn(HornSpec{X0: 66.8, Y0: 28.2, X1: 77.2, Y1: 14.6, W: 2.6, Bend: 0.9, Ramp: hornIvory, Grooves: true, Shade: -0.28})
	p.Horn(HornSpec{X0: 69.2, Y0: 35.0, X1: 70.8, Y1: 28.8, W: 2.2, Bend: 0.4, Ramp: hornIvory})
	p.Horn(HornSpec{X0: 62.4, Y0: 29.2, X1: 72.0, Y1: 12.6, W: 3.0, Bend: 1.0, Ramp: hornIvory, Grooves: true})
	p.Eye(EyeSpec{X: 63.4, Y: 32.6, R: 0.95, Iris: greenIris, Lid: 0.3, Squint: 1.3})
	p.Nostril(74.2, 35.6, 0.7, 0.45, -0.3)
	// The nails of the near feet, then the far ones, darker.
	for _, n := range [][3]float64{{18.5, 55.1, 1}, {21.3, 55.3, 1.05}, {24.1, 55.0, 0.9}, {47, 55.1, 1}, {49.8, 55.3, 1.05}, {52.5, 54.9, 0.9}} {
		p.Nail(n[0], n[1], 2.4*n[2], 1.7*n[2], hoofHorn, -0.15)
	}
	for _, n := range [][2]float64{{30.3, 52.9}, {32.8, 52.8}, {39.5, 53.6}, {42, 53.4}} {
		p.Nail(n[0], n[1], 2.0, 1.4, hoofHorn, -0.45)
	}
}
