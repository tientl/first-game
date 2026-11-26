package dinopaint

// Pteranodon. dinos/ptero_fly.png, 44 x 58: flying, seen from above, head
// to the right, the far wing up the picture and the near one down it, the
// pale bones of the arm and the long finger along each wing's leading edge,
// a brown membrane, a cream furred body, an orange crest swept back over the
// neck and a long beak. Its rig is "pteroFly" in internal/client/dinorigs.go:
// each wing turns about its shoulder, (18.5, 22.2) and (18.5, 33.8), and the
// whole drawing is squashed up and down through a wingbeat.
//
// dinos/ptero_perch.png, 56 x 39: on the ground, side on, walking on its
// knuckles with the wings folded (rig "pteroPerch": drawn as it stands).

func init() { Register(pteroFly()) }

var (
	pteroMembrane = []uint32{0x1e0e0a, 0x2e160e, 0x44200f, 0x5c2c14, 0x76391a, 0x904a22, 0xa85e2c, 0xc07438}
	pteroGlow     = []uint32{0x5a2410, 0x82361a, 0xa84c22, 0xc86a2e, 0xe08a40, 0xf0aa5c, 0xf8c87e, 0xfde0a8}
	pteroBone     = []uint32{0x4a3a26, 0x6e5838, 0x947a50, 0xb49a6a, 0xcab284, 0xdccaa0, 0xece0c0, 0xf8f0dc}
	pteroCream    = []uint32{0x4e4234, 0x706048, 0x92805e, 0xb4a07a, 0xd0be98, 0xe6d8b8, 0xf4ecd6, 0xfffaee}
	pteroTan      = []uint32{0x2e2216, 0x463420, 0x60482c, 0x7c5e3a, 0x987448, 0xb08a58, 0xc6a26e, 0xdabc8c}
	pteroCrest    = []uint32{0x4a1008, 0x6e180c, 0x962410, 0xbc3416, 0xdc4c1e, 0xf06e2a, 0xf89a3c, 0xffc458}
	pteroBeak     = []uint32{0x3a2c10, 0x564418, 0x745e22, 0x92782e, 0xb0963e, 0xcab252, 0xe2ce6e, 0xf6e696}
	pteroEye      = []uint32{0x0a0808, 0x1c1410, 0x3a2614, 0x5e3c18, 0x845820, 0xa8742c}
)

func pteroFly() *Species {
	// The bones of each wing, shoulder to wingtip, along its leading edge.
	farBone := []float64{20.1, 22.8, 21.6, 18.5, 23.4, 15, 24.1, 12.6, 23.6, 10.5, 23, 8.5, 21.5, 6.2, 18.6, 4, 15.6, 2.2, 14.6, 0.9}
	nearBone := []float64{20.1, 32.6, 21.6, 35.5, 22.8, 38.8, 23.8, 42, 23.6, 44.5, 23, 47, 21.1, 49.8, 18.4, 51.3, 15.9, 52.5, 14.7, 53.2}
	farWing := Poly(14, -0.5, 18.5, 3, 22, 6.5, 23.8, 10, 24.2, 13, 23.2, 17, 21.6, 21, 20.6, 23.6, 4.5, 23.6, 6, 21, 9.5, 17, 11, 12, 11.5, 3)
	nearWing := Poly(19, 31.6, 21.5, 34, 23.8, 41, 23.8, 45, 22.6, 48, 20, 50.2, 16, 52.6, 13.8, 53.8, 11.4, 52, 11, 45, 10.2, 40, 9, 36, 5, 33.5, 3.5, 32.5, 12, 31.6)
	body := Blend(1.2,
		Ellipse(17, 27.4, 8.2, 2.7, 0),
		Cone(23, 27.2, 1.9, 27.4, 27.2, 1.6),
		Path([3]float64{10, 27.5, 1.3}, [3]float64{3, 27.5, 0.8}), // the short tail
	)
	head := Ellipse(27.4, 27.1, 2.2, 1.9, 0)
	legs := Union(Cone(11.5, 24.6, 1.0, 3.8, 23.6, 0.7), Cone(11.5, 30.4, 1.0, 3.8, 31.3, 0.7))
	crest := Poly(26.8, 23.2, 26.8, 26.4, 21.5, 26.2, 17.2, 24.8, 17.2, 22.9)
	beak := Poly(28.6, 26.5, 34, 26.6, 39.6, 27.5, 40.9, 28.6, 39.6, 29.5, 33, 29.4, 28.7, 29.1)
	return &Species{
		Name:        "ptero_fly",
		GuideDetail: 0.4,
		Ink:         36,
		Materials: []Material{
			{Name: "membrane", Seeds: []uint32{0x93552f, 0x6e3a22, 0x4a2618, 0xb47642, 0x2e1c16, 0x44281c, 0x2c1610}, Ramp: pteroMembrane, Relief: 0.05},
			{Name: "cream", Seeds: []uint32{0xfdf6e2, 0xeee2c6, 0xd4c2a2, 0xd7d1c0, 0xb09a7c, 0xc0a074, 0x967652}, Ramp: pteroCream},
			{Name: "tan", Seeds: []uint32{0xb08a58, 0x8e6840, 0x96754b, 0x86725a, 0x5a4a3a, 0x4c3f31}, Ramp: pteroTan},
			{Name: "crest", Seeds: []uint32{0xe0541e, 0xf88a2e, 0xffc050, 0x6e2418}, Ramp: pteroCrest, Region: crest},
			{Name: "beak", Seeds: []uint32{0xf8e69a, 0x6e5420, 0x9c7a2c, 0xffe27a}, Ramp: pteroBeak, Relief: 0.3, Region: beak},
		},
		Recolor: []Recolor{
			{Region: Union(farWing, nearWing), From: []string{"cream", "tan"}, To: "membrane"}, // the leading edge is painted as bone
			{Region: Poly(0, 0, 44, 0, 44, 22.4, 0, 22.4), From: []string{"crest"}, To: "membrane"},
		},
		Parts: []Part{
			{Name: "farWing", Shape: farWing, Round: 1.6, Soft: 0.8, Scale: 0.8},
			{Name: "nearWing", Shape: nearWing, Round: 1.6, Soft: 0.8, Scale: 0.8},
			{Name: "legs", Shape: legs, Round: 0.8, Soft: 0.5, Scale: 0.5, Mat: "cream"},
			{Name: "body", Shape: body, Round: 2.6, Soft: 0.7, Muscles: []Muscle{
				Mu(19, 27.2, 5, 2, 0, 0.6), // the back over the shoulders
				Mu(27.5, 26.8, 1.6, 1.2, 0, 0.4),
			}},
			{Name: "head", Shape: head, Round: 1.5, Soft: 0.5, Scale: 0.5, Mat: "cream", Muscles: []Muscle{Mu(27.2, 26.6, 1.4, 1, 0, 0.5)}},
			{Name: "crest", Shape: crest, Round: 0.8, Soft: 0.5, Scale: 0.6, Mat: "crest"},
			{Name: "beak", Shape: beak, Round: 0.9, Soft: 0.4, Scale: 0.5, Mat: "beak"},
		},
		Skin:    Skin{Flank: 1.0, Back: 1.2, Belly: 0.9, BackDepth: 1, BellyDepth: 1, Relief: 0.5, Seed: 53},
		Details: pteroFlyDetails(farBone, nearBone, farWing, nearWing, crest),
	}
}

func pteroFlyDetails(farBone, nearBone []float64, farWing, nearWing, crest SDF) func(p *Painter) {
	return func(p *Painter) {
		p.Plumage(&Plumage{Seed: 23, Edge: 0.35, Tracts: []Tract{
			// The furred body: fine pycnofibres from the nape back.
			{Name: "fur", Region: Poly(3, 24.5, 24.6, 24.6, 24.8, 30.2, 3, 30.6), Parts: []string{"body"},
				Flow: Along(26, 27.2, 18, 27.3, 9, 27.5, 2, 27.5), Len: 1.2, Wid: 0.55, Hair: true, Relief: 0.8, Fluff: 0.3, Snap: true},
		}})
		wing := func(region SDF, bone []float64, fan Coords, free SDF) {
			p.Membrane(MembraneSpec{Region: region, Mat: []string{"membrane"},
				Fibres: fan, Gap: 1.1, Fibre: 0.13, Flat: 0.55, Distal: 0.16, DistalFrom: 11, DistalTo: 21,
				Camber: Path(pts3(bone, 0)...), Glow: 0.6, GlowW: 1.7, GlowRamp: pteroGlow, Free: free,
				Bones: []Bone{{Pts: bone, R: 0.95, R1: 0.5, Ramp: pteroBone}}, Seed: 7})
		}
		wing(farWing, farBone, Fan(21.5, 23, 10, -1.78), Poly(-1, -1, 13, -1, 21, 12, 21, 21, -1, 22))
		wing(nearWing, nearBone, Fan(21.5, 33, 10, 1.78), Poly(-1, 59, 13, 59, 21, 44, 21, 34, -1, 33))
		p.Plane(PlaneSpec{Region: crest, Mat: []string{"crest"}, Tilt: 1.1, Angle: 0.08, Gap: 0.9, Lines: 0.06,
			FromX: 26.5, FromY: 23.5, Fall: 0.03, Seed: 9})
		p.Beak(BeakSpec{Mat: "beak", Ridge: []float64{28.8, 26.7, 34, 26.8, 40.6, 28.0}, Mouth: []float64{29, 28.05, 34.5, 28.15, 40.4, 28.5},
			Gloss: 0.2, LowerShadow: 0.32, GrainSeed: 7})
		p.Eye(EyeSpec{X: 27.4, Y: 27.9, R: 0.55, Iris: pteroEye, Lid: 0.15, Squint: 1.1})
	}
}

// pts3 is a polyline as points of radius r, for Path.
func pts3(xy []float64, r float64) [][3]float64 {
	var out [][3]float64
	for i := 0; i+1 < len(xy); i += 2 {
		out = append(out, [3]float64{xy[i], xy[i+1], r})
	}
	return out
}

// Quetzalcoatlus. dinos/quetzal_fly.png, 92 x 96: flying, seen from above
// as the Pteranodon is, its slate wings edged in pale bone, a cream and
// orange neck, a red crest with a dark crown and a long yellow beak tipped
// red. Its rig is "quetzalFly": the wings turn about (30.2, 36.1) and
// (30.2, 55.9).

var (
	quetzalWing  = []uint32{0x0e1018, 0x161a24, 0x1e2432, 0x283040, 0x323c50, 0x3e4a60, 0x4c5a72, 0x5e6c84}
	quetzalGlow  = []uint32{0x2a2630, 0x3e3640, 0x58484e, 0x765a5a, 0x946e64, 0xb08672, 0xc8a084, 0xdcbc9c}
	quetzalTaupe = []uint32{0x221a14, 0x34281e, 0x483a2a, 0x5e4c38, 0x766048, 0x8e785a, 0xa6906e, 0xbca886}
	quetzalOrng  = []uint32{0x3a160a, 0x5a2210, 0x7c3216, 0x9e461e, 0xbc5c28, 0xd27634, 0xe49446, 0xf2b462}
	quetzalRed   = []uint32{0x300808, 0x4c0c0e, 0x6c1214, 0x8e1a1a, 0xb02620, 0xcc3a2a, 0xe2563a, 0xf27c54}
	quetzalSlate = []uint32{0x101014, 0x18181e, 0x22222a, 0x2e2e38, 0x3a3a46, 0x484856, 0x585868, 0x6a6a7c}
)

func init() { Register(quetzalFly()) }

func quetzalFly() *Species {
	farBone := []float64{31.5, 37.6, 33.5, 32, 35, 27, 36.6, 23.6, 38.5, 20.4, 39.6, 18.4, 38.6, 15, 37.6, 11.5, 35.8, 8.4, 32.6, 5.6, 28, 2.8, 24, 0.9}
	nearBone := []float64{31, 53, 33, 56.5, 34.6, 60.5, 35.6, 64, 37.6, 68, 38.8, 70.5, 40, 72.5, 38.8, 75, 37.8, 78, 35.8, 81.5, 31.8, 84.5, 27.6, 86.6, 24.4, 88.4}
	farWing := Poly(22, -0.5, 25.5, 0.8, 30, 3, 34, 5.6, 37.6, 8.6, 39, 12, 40, 16, 42.5, 18.8, 41.5, 22, 39.5, 27, 37.6, 32, 36, 36.6, 34, 38.6,
		29, 38.6, 5, 38.4, 7, 36, 12, 32, 16, 26.5, 18.4, 19, 19.5, 12, 20, 4)
	nearWing := Poly(30, 51, 35.5, 52.5, 37, 58, 39.5, 64.5, 41.4, 69.5, 42.4, 71.6, 41, 74, 39.6, 77, 37, 81.5, 33, 84, 28.5, 86.8, 25, 88.6,
		23.5, 89.6, 20.6, 88, 20, 80, 19.5, 74, 18, 68, 15.5, 63, 12, 58.5, 8.5, 55.5, 5.5, 53.5, 4.5, 52.6, 14, 51.4)
	body := Blend(1.5,
		Ellipse(21, 45.2, 11, 3.6, 0),
		Cone(30, 45, 3.0, 50, 44.6, 1.9),
	)
	head := Ellipse(54.6, 44.9, 3.2, 2.2, 0)
	legs := Union(Cone(14, 39.6, 1.1, 4.5, 39.2, 0.7), Cone(15, 45.2, 1.1, 6.5, 45.4, 0.8), Cone(14, 51.4, 1.1, 4.5, 52, 0.7))
	crest := Poly(49, 38.2, 57, 38, 58.6, 41, 57.5, 44.6, 52, 45, 49.3, 44, 48.8, 41)
	beak := Poly(57.4, 43, 64, 43, 74, 43.6, 79.5, 44.6, 82.8, 46.2, 81.4, 47.4, 77, 48.4, 67, 49, 59, 49.2, 57.5, 47.5)
	return &Species{
		Name:        "quetzal_fly",
		GuideDetail: 0.4,
		Ink:         34,
		Materials: []Material{
			{Name: "wing", Seeds: []uint32{0x2c3342, 0x3e4858, 0x546274, 0x1e222d, 0x12141c}, Ramp: quetzalWing, Relief: 0.05},
			{Name: "cream", Seeds: []uint32{0xcdb594, 0xe9dabb, 0xfbf3df, 0xd8caa8, 0xd5cfbe, 0xc6b99f, 0xb8a888}, Ramp: pteroCream},
			{Name: "taupe", Seeds: []uint32{0xa6896a, 0x7a6044, 0x4e3c2a, 0x94846a, 0x6e604c}, Ramp: quetzalTaupe},
			{Name: "orange", Seeds: []uint32{0xcf873c, 0xae6228, 0x86451e, 0xb07333}, Ramp: quetzalOrng},
			{Name: "red", Seeds: []uint32{0xe85236, 0xf88a5c, 0xc22a22, 0x90181a, 0x5e0e12}, Ramp: quetzalRed},
			{Name: "crown", Seeds: []uint32{0x34323c}, Ramp: quetzalSlate},
			{Name: "beak", Seeds: []uint32{0xfbe694, 0xeecb5a, 0xd2a436, 0xa87a22, 0x7a5216, 0xffe27a}, Ramp: pteroBeak, Relief: 0.3, Region: beak, Over: true,
				Only: []string{"cream", "taupe", "orange", "wing"}},
		},
		Recolor: []Recolor{
			{Region: Union(farWing, nearWing), From: []string{"cream", "taupe", "orange"}, To: "wing"}, // the leading edge is painted as bone
			{Region: Poly(0, 0, 92, 0, 92, 37.6, 0, 37.6), From: []string{"crown"}, To: "wing"},
		},
		Parts: []Part{
			{Name: "farWing", Shape: farWing, Round: 2, Soft: 0.8, Scale: 0.8},
			{Name: "nearWing", Shape: nearWing, Round: 2, Soft: 0.8, Scale: 0.8},
			{Name: "legs", Shape: legs, Round: 0.9, Soft: 0.5, Scale: 0.5, Mat: "cream"},
			{Name: "body", Shape: body, Round: 3, Soft: 0.7, Muscles: []Muscle{
				Mu(22, 44.6, 7, 2.4, 0, 0.7),
			}},
			{Name: "head", Shape: head, Round: 1.8, Soft: 0.5, Scale: 0.5, Muscles: []Muscle{Mu(54.4, 44.2, 2, 1.2, 0, 0.5)}},
			{Name: "crest", Shape: crest, Round: 1, Soft: 0.5, Scale: 0.6},
			{Name: "beak", Shape: beak, Round: 1.1, Soft: 0.4, Scale: 0.5},
		},
		Skin: Skin{Flank: 1.0, Back: 1.2, Belly: 0.9, BackDepth: 1, BellyDepth: 1, Relief: 0.5, Seed: 59},
		Details: func(p *Painter) {
			p.Plumage(&Plumage{Seed: 29, Edge: 0.35, Tracts: []Tract{
				{Name: "fur", Region: Poly(8, 40.4, 49.6, 41, 50, 48.6, 8, 49.6), Parts: []string{"body"},
					Flow: Along(50, 44.6, 40, 44.6, 30, 44.8, 20, 45, 9, 45.5), Len: 1.4, Wid: 0.6, Hair: true, Relief: 0.8, Fluff: 0.3, Snap: true},
			}})
			wing := func(region SDF, bone []float64, fan Coords, free SDF) {
				p.Membrane(MembraneSpec{Region: region, Mat: []string{"wing"},
					Fibres: fan, Gap: 1.25, Fibre: 0.12, Flat: 0.55, Distal: 0.12, DistalFrom: 18, DistalTo: 36,
					Camber: Path(pts3(bone, 0)...), Glow: 0.5, GlowW: 2.2, GlowRamp: quetzalGlow, Free: free,
					Bones: []Bone{{Pts: bone, R: 1.35, R1: 0.5, Ramp: pteroBone}}, Seed: 9})
			}
			wing(farWing, farBone, Fan(31, 38, 14, -1.78), Poly(-1, -1, 22, -1, 37, 20, 33, 37, -1, 37))
			wing(nearWing, nearBone, Fan(31, 53, 14, 1.77), Poly(-1, 97, 22, 97, 38, 72, 33, 54, -1, 54))
			p.Plane(PlaneSpec{Region: crest, Mat: []string{"red", "crown"}, Tilt: 0.8, Angle: 0.1, Gap: 1.0, Lines: 0.06,
				FromX: 57, FromY: 40, Fall: 0.03, Seed: 11})
			p.Beak(BeakSpec{Mat: "beak", Ridge: []float64{57.6, 43.1, 66, 43.2, 75, 43.9, 82.6, 46.1}, Mouth: []float64{58, 46.1, 68, 46.2, 77, 46.4, 82, 46.7},
				Gloss: 0.22, LowerShadow: 0.3, GrainSeed: 9})
			p.Eye(EyeSpec{X: 57, Y: 46.6, R: 0.7, Iris: pteroEye, Lid: 0.15, Squint: 1.1})
		},
	}
}

func init() { Register(pteroPerch()) }

var (
	pteroDarkMem = []uint32{0x140a08, 0x20100c, 0x2e1610, 0x3e1e14, 0x502818, 0x64321e, 0x783e24, 0x8c4c2c}
	pteroGrey    = []uint32{0x1c1612, 0x2a221c, 0x3a3026, 0x4c4032, 0x5e5040, 0x726250, 0x887662, 0x9e8c76}
)

func pteroPerch() *Species {
	finger := []float64{32.4, 31.5, 29.5, 27.4, 26.6, 23.2, 23.6, 18.6, 20.4, 14.4, 18.4, 11.6}
	arm := []float64{33.2, 23.5, 32.6, 28.5, 32.3, 33.5}
	wing := Poly(14, 12, 17.5, 10.4, 19.8, 11.2, 24.4, 17, 31.6, 26.6, 33.2, 31, 30.5, 33.6, 22.5, 33.2, 19, 30, 15, 22, 13.6, 16)
	body := Blend(1.6,
		Ellipse(30.5, 25, 5.2, 4.8, 0),
		Cone(31.6, 21.5, 3.6, 36.2, 13, 2.4),
		Ellipse(38, 11.8, 2.8, 2.4, 0),
	)
	hind := Union(Cone(22.4, 29.2, 2.1, 20.6, 33, 1.3), Cone(20.6, 33, 1.2, 20.8, 36.4, 1.0), Cone(20.4, 37, 0.9, 26.6, 37.3, 0.6))
	hind2 := Union(Cone(25.6, 31.5, 1.3, 25, 35.2, 1.0), Cone(24.6, 36.2, 0.8, 28.6, 36.8, 0.55))
	foreArm := Union(Cone(33.2, 24, 1.9, 32.5, 33, 1.4), Cone(32.5, 33, 1.3, 34.6, 36, 1.0), Cone(34.2, 37, 0.9, 39.6, 37.4, 0.6))
	crest := Poly(27, 1.6, 29.6, 1.4, 33, 4, 37, 7, 39.5, 8, 38.5, 10.4, 36.5, 10.6, 35, 9.8, 31.5, 7.6, 28, 4.6)
	beak := Poly(41.6, 8.6, 44.6, 9.8, 48, 10.8, 51.5, 12, 54, 13.2, 55.6, 14.5, 53.5, 15.6, 50, 16, 45, 16.2, 41, 16, 40.8, 12.4)
	return &Species{
		Name:        "ptero_perch",
		GuideDetail: 0.4,
		Ink:         34,
		Materials: []Material{
			{Name: "membrane", Seeds: []uint32{0x93552f, 0x764426, 0xb47642, 0x673b21, 0x905e35, 0xa67b4b, 0xcf9a5e, 0x5c3824, 0x402719}, Ramp: pteroMembrane, Relief: 0.05},
			{Name: "dark", Seeds: []uint32{0x6e3a22, 0x4a2618, 0x582e1b, 0x3b1e13, 0x4d2918, 0x3e2013, 0x2c1610, 0x23120d, 0x341b11, 0x29150d, 0x1f0f0b}, Ramp: pteroDarkMem, Relief: 0.05},
			{Name: "tan", Seeds: []uint32{0xb08a58, 0x8e6840, 0x86725a, 0x7b613e, 0x6a5038, 0x6c4c30, 0x948871, 0xa79e8b, 0x8f8877, 0x7f7461, 0x989488, 0xaca186, 0x9e8c6a, 0x867051, 0x695339, 0x967652, 0xc0a074}, Ramp: pteroTan},
			{Name: "cream", Seeds: []uint32{0xfdf6e2, 0xeee2c6, 0xd4c2a2, 0xe2c898, 0xf6e6c0, 0xb09a7c}, Ramp: pteroCream},
			{Name: "grey", Seeds: []uint32{0x5a4a3a, 0x504436, 0x4a4134, 0x362c23, 0x3a3026, 0x726451, 0x9b9381}, Ramp: pteroGrey},
			{Name: "crest", Seeds: []uint32{0xffc050, 0xf88a2e, 0xe0541e, 0xb03018, 0x6e2418}, Ramp: pteroCrest, Region: crest, Over: true,
				Only: []string{"cream", "tan", "grey"}},
			{Name: "beak", Seeds: []uint32{0xf8e69a, 0xe8c860, 0xc8a240, 0x9c7a2c, 0x6e5420, 0xffd84a}, Ramp: pteroBeak, Relief: 0.3, Region: beak, Over: true,
				Only: []string{"cream", "tan", "grey"}},
		},
		Recolor: []Recolor{
			{Region: wing, From: []string{"tan", "cream"}, To: "membrane"},                 // the bones are painted again
			{Region: Circle(40.5, 11.2, 1.3), From: []string{"beak", "grey"}, To: "cream"}, // so is the eye
		},
		Parts: []Part{
			{Name: "hind2", Shape: hind2, Far: true, Round: 1, Soft: 0.5, Scale: 0.6},
			{Name: "body", Shape: body, Round: 3.6, Soft: 0.8, Muscles: []Muscle{
				Mu(30.5, 24, 3.6, 3.4, 0, 0.8), // the chest
				Mu(38, 11.2, 1.8, 1.4, 0, 0.5), // the head
			}},
			{Name: "crest", Shape: crest, Round: 1, Soft: 0.5, Scale: 0.6},
			{Name: "beak", Shape: beak, Round: 1.1, Soft: 0.4, Scale: 0.5},
			{Name: "wing", Shape: wing, Round: 2.4, Soft: 0.8, Scale: 0.8, Muscles: []Muscle{Mu(19, 17, 3, 4.5, -0.6, 0.8)}},
			{Name: "hind", Shape: hind, Round: 1.2, Soft: 0.5, Scale: 0.6, Muscles: []Muscle{Mu(22, 30, 1.6, 1.6, 0, 0.5)}},
			{Name: "foreArm", Shape: foreArm, Round: 1.4, Soft: 0.5, Scale: 0.6},
		},
		Skin: Skin{Flank: 1.0, Back: 1.2, Belly: 0.9, BackDepth: 1, BellyDepth: 1, Relief: 0.6, Seed: 61},
		Folds: []Fold{
			{Pts: []float64{15.5, 15, 19.5, 21.5, 24.5, 28.5}, N: 4, DX: 1.3, DY: -0.5, Depth: 1.1}, // the folded membrane's pleats
			{Pts: []float64{15, 19, 18, 25, 21.5, 30.5}, N: 2, DX: 1.1, DY: -0.6, Depth: 0.8},
		},
		Details: func(p *Painter) {
			p.Plumage(&Plumage{Seed: 31, Edge: 0.35, Tracts: []Tract{
				{Name: "fur", Region: Poly(26, 13, 37.5, 9.5, 39, 14, 37.5, 26, 34, 31, 26, 31), Parts: []string{"body"},
					Flow: Along(37.5, 10.5, 34.5, 16, 31.5, 21, 29.5, 26, 28.5, 31), Len: 1.3, Wid: 0.6, Hair: true, Relief: 0.8, Fluff: 0.3, Snap: true},
			}})
			p.Membrane(MembraneSpec{Region: wing, Mat: []string{"membrane", "dark"},
				Fibres: Along(finger...), Gap: 0.9, Fibre: 0.05, Flat: 0.15,
				Camber: Path(pts3(finger, 0)...), Glow: 0.5, GlowW: 1.4, GlowRamp: pteroGlow,
				Free: Poly(10, 8, 18, 9, 22, 30, 12, 34),
				Bones: []Bone{
					{Pts: finger, R: 0.8, R1: 0.45, Ramp: pteroBone},
					{Pts: arm, R: 1.2, R1: 0.95, Ramp: pteroBone, Shade: -0.05},
				}, Seed: 13})
			p.Plane(PlaneSpec{Region: crest, Mat: []string{"crest"}, Tilt: 0.9, Angle: -0.72, Gap: 1.0, Lines: 0.06,
				FromX: 37.5, FromY: 8.5, Fall: 0.05, Seed: 15})
			p.Beak(BeakSpec{Mat: "beak", Ridge: []float64{41.8, 8.8, 46, 10.3, 51, 11.9, 55.4, 14.2}, Mouth: []float64{41.6, 13.4, 47, 14.2, 54.4, 14.6},
				Gloss: 0.22, LowerShadow: 0.4, GrainSeed: 11})
			p.Eye(EyeSpec{X: 40.5, Y: 11.3, R: 0.75, Iris: []uint32{0x5a3a08, 0x8e600c, 0xc89216, 0xf0c22e, 0xffda54, 0xffec96}, Lid: 0.18, Squint: 1.15})
			// The claws of the feet and the hand on the ground.
			for _, c := range [][4]float64{{26.2, 37.1, 27.7, 37.7}, {28.2, 36.6, 29.6, 37.3}, {39.2, 37.2, 40.6, 37.8}, {37.6, 37.4, 38.6, 38.2}} {
				p.Claw(c[0], c[1], c[2], c[3], 0.55, 0.15, anzuClaws)
			}
		},
	}
}
