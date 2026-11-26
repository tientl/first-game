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
