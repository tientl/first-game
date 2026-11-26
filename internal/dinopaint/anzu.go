package dinopaint

// Anzu: dinos/anzu.png, 50 x 55, facing right: a cassowary of a dinosaur,
// teal feathers, a gold casque with a coral top, a cyan face, a bare blue
// and purple neck with a red wattle, an orange wing edged in cream, a fan
// of a tail, a pale beak. Its rig is "anzu" in internal/client/dinorigs.go:
// hips (27, 31) and (18, 31), knees (27.5, 40) and (17.5, 40), ankles
// (25.5, 46.5) and (16, 45); the legs are cut from row 36, columns 23-36
// (near) and 13-22 (far); the head and neck turn about (30.5, 24.5), the
// beak opens about (39.5, 11.5) along the line to (46, 11.8), and the tail
// turns about (11, 24).

func init() {
	Register(anzu())
	// The legs swing whole from the hip, the near thigh under the folded
	// wing and the flank (thigh.go).
	RegisterThighs("anzu", &Thighs{
		Legs:       LegMask{{"nearLeg"}, {"farLeg"}},
		FrontParts: []string{"wing"},
		Front:      Poly(11, 18, 38, 18, 38, 31.4, 11, 31.4),
		Thigh:      []SDF{Cone(27, 33, 4.1, 27.5, 40, 2.8), Cone(17.8, 33, 3.8, 17.5, 40, 2.6)},
		Coat:       Plumed(anzuPlumage),
	})
}

var (
	anzuIris  = []uint32{0x5a3204, 0x8e5608, 0xc88412, 0xf0ac26, 0xffc84c, 0xffe08c}
	anzuHorn  = []uint32{0x2a2418, 0x433a2a, 0x5e5440, 0x7c7058, 0x9a8e74, 0xb6ac90, 0xccc4aa, 0xe0dac4}
	anzuClaws = []uint32{0x14100e, 0x2a221c, 0x463c32, 0x6a5e50, 0x948874, 0xbcb29c}
)

func anzu() *Species {
	farLeg := Union(
		Cone(17.8, 33, 3.5, 17.5, 40, 2.3),
		Cone(17.5, 40, 1.6, 16.2, 45, 1.2),
		Cone(16.2, 45, 1.1, 17.4, 51.6, 1.0),
		Cone(17.4, 52.4, 1.0, 21.9, 53.6, 0.7),
	)
	nearLeg := Union(
		Cone(27, 33, 3.8, 27.5, 40, 2.5),
		Cone(27.5, 40, 1.7, 25.6, 46.5, 1.25),
		Cone(25.6, 46.5, 1.15, 27, 51.6, 1.0),
		Cone(27, 52.4, 1.05, 35, 53.6, 0.75),
	)
	body := Blend(2.5,
		Path([3]float64{14, 25.5, 4}, [3]float64{8, 23.5, 5}, [3]float64{3, 23, 5.5}), // the tail and its fan
		Ellipse(24.5, 27.5, 9.5, 5.6, 0),
		Cone(31, 24, 4, 33.6, 13, 2.6),                                                                          // the neck
		Poly(33.5, 7, 36, 6.4, 43, 6.6, 46, 7.4, 48.4, 9.6, 48.4, 11.6, 47, 13.6, 41, 14, 38.4, 16.4, 34, 14.5), // the head
	)
	casque := Poly(33.3, 7.6, 33.5, 3.4, 35.6, 0.3, 42.8, 0.1, 44.8, 2.6, 45.4, 6.8, 41, 7.4)
	jaw := Poly(39.6, 11.3, 46.8, 11.6, 46.4, 12.9, 42, 13.9, 39.6, 13.3)
	wattle := Ellipse(36.8, 13.9, 1.55, 2.1, 0.4)
	wing := Poly(33.4, 26.4, 32.8, 30.6, 26, 31.9, 14.4, 33.5, 13.2, 30.4, 18, 28.2, 26, 26.8, 30, 26)
	beak := Poly(39.6, 7.1, 44.6, 7.0, 47.6, 8.4, 48.3, 10.6, 47.2, 12.8, 43, 13.8, 40.4, 13.6, 39.4, 11)
	return &Species{
		Name:        "anzu",
		GuideDetail: 0.35,
		Ink:         24,
		Materials: []Material{
			{Name: "teal", Seeds: []uint32{0x4c8c96, 0x2f6074, 0x21425a, 0x1a3540, 0x2a4d52, 0x426e81, 0x5c969f, 0x355369, 0x243d47},
				Ramp: []uint32{0x0c121c, 0x14202e, 0x1c3040, 0x244256, 0x2e566c, 0x3a6c80, 0x4c8694, 0x66a2aa}},
			// The navy of the underside and the thighs, as dark as the drawing has it.
			{Name: "navy", Seeds: []uint32{0x182c44, 0x111c30, 0x1d2e3a, 0x0b0f1c},
				Ramp: []uint32{0x080c14, 0x0c121e, 0x101a2a, 0x162236, 0x1c2c44, 0x243852, 0x2e4662, 0x3a5672}},
			{Name: "cyan", Seeds: []uint32{0x8cdce0, 0x4cb2c8, 0x2c86a4, 0x1c5a7a, 0x7cb8b4},
				Ramp: []uint32{0x0e2a3a, 0x164058, 0x1e5a76, 0x2a7a96, 0x3c9ab2, 0x58b6c8, 0x7ccedc, 0xa6e4ec}},
			{Name: "purple", Seeds: []uint32{0x7a6ad8, 0x5644ba, 0x3c82e2, 0x2658c2, 0x1a3a92, 0x12245e, 0x231d31},
				Ramp: []uint32{0x120e2a, 0x1c1846, 0x262668, 0x323a8e, 0x3e50b2, 0x5462cc, 0x7474dc, 0x9a90ea}},
			{Name: "red", Seeds: []uint32{0xd23428, 0xa01c1c, 0x2a0e0d, 0x34130f},
				Ramp: []uint32{0x240707, 0x380a0a, 0x500f0e, 0x6c1612, 0x8a1e16, 0xa8281c, 0xc43a26, 0xdc5a3c}},
			{Name: "coral", Seeds: []uint32{0xf06a48, 0x3a1d15},
				Ramp: []uint32{0x3a1410, 0x5a1e16, 0x7e2c1c, 0xa23c24, 0xc4502e, 0xdc663a, 0xf0824e, 0xfaa46c}},
			{Name: "gold", Seeds: []uint32{0xecd080, 0xd0aa50, 0xcbb36e, 0xb39245, 0xa88034, 0x906e2d, 0x7a5622, 0xd8c997, 0xfbeab0, 0x4e3214, 0x432b11},
				Ramp: []uint32{0x2a1c0a, 0x42300e, 0x5e4614, 0x7e601c, 0x9e7c28, 0xbc9838, 0xd6b654, 0xecd486}},
			{Name: "orange", Seeds: []uint32{0xe29446, 0xc4682a, 0x9a421e, 0x7b3518, 0x6e2a16, 0x9d5322, 0xb57638, 0x582212, 0x45180e, 0x37130b},
				Ramp: []uint32{0x2a0e08, 0x44180c, 0x642612, 0x863a1a, 0xa85224, 0xc66c30, 0xde8a42, 0xf0aa62}},
			{Name: "cream", Seeds: []uint32{0xfdf6dc, 0xf0e4bc, 0xd2c08e, 0xbcb494, 0xa89468},
				Ramp: []uint32{0x4a4036, 0x6a5e4c, 0x8c7e66, 0xaea080, 0xcabe9c, 0xe0d6b8, 0xf0e8d0, 0xfcf8ea}, Bias: 0.06},
			{Name: "horn", Seeds: []uint32{0xd6ccac, 0xb0a482, 0x84785c, 0x5e5442, 0x3e372c, 0x787056, 0x58523f, 0x3c382e, 0x3c3832, 0x9c9474, 0x5e5442},
				Ramp: anzuHorn},
			{Name: "beak", Ramp: anzuHorn, Relief: 0.3, Region: beak},
			{Name: "farHorn", Ramp: []uint32{0x16120e, 0x221c16, 0x30281e, 0x403628, 0x524632, 0x64583e, 0x786a4c, 0x8a7c5c}},
			{Name: "eye", Seeds: []uint32{0xffc032}, Ramp: anzuIris},
		},
		Recolor: []Recolor{
			{Region: Circle(38.3, 9.2, 1.5), From: []string{"eye", "teal", "horn"}, To: "cyan"}, // the eye is painted again
			{Region: Poly(10, 36, 40, 36, 40, 56, 10, 56), From: []string{"cream"}, To: "horn"},
			{Region: Poly(0, 0, 50, 0, 50, 7.4, 0, 7.4), From: []string{"red"}, To: "coral"},
			{Region: Poly(0, 22, 50, 22, 50, 56, 0, 56), From: []string{"purple"}, To: "teal"},
			{Region: Poly(12, 40.6, 23, 40.6, 23, 56, 12, 56), From: []string{"horn", "teal"}, To: "farHorn"}, // the far shank, in shadow
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.6, Scale: 0.6, Muscles: []Muscle{
				Mu(17.8, 34, 3, 3.2, 0, 1.0),
			}},
			{Name: "body", Shape: body, Round: 7, RoundAt: func(x, y float64) float64 {
				return lerp(7, 3, smoothstep(29, 33, x))
			}, Muscles: []Muscle{
				Mu(23.5, 26, 7, 4, 0, 0.9),        // the ribs
				Mu(30, 23.5, 3, 3.5, 0.3, 0.7),    // the base of the neck
				Mu(35.6, 9.4, 2.2, 1.5, 0, 0.5),   // the cheek
				Mu(37.6, 7.6, 1.6, 0.7, 0, 0.6),   // the brow
				Mu(43.4, 9.2, 3.4, 1.4, 0, 0.5),   // the beak
				Mu(32.6, 16.5, 1.8, 4, 0.25, 0.6), // the neck
			}},
			{Name: "casque", Shape: casque, Round: 1.4, Soft: 0.5, Scale: 0.7},
			{Name: "wattle", Shape: wattle, Round: 1.6, Soft: 0.5, Scale: 0.5, Mat: "red", Muscles: []Muscle{Mu(36.5, 13.5, 1.1, 1.4, 0.4, 0.4)}},
			{Name: "jaw", Shape: jaw, Round: 0.9, Soft: 0.4, Scale: 0.5},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.8, Scale: 0.6, Onto: "body", Seam: Below(10, 32.5, 40, 32), Muscles: []Muscle{
				Mu(27, 34, 3.2, 3.4, 0, 1.1),
			}},
			{Name: "wing", Shape: wing, Round: 2, Soft: 0.6, Muscles: []Muscle{
				Mu(29, 28, 3, 1.6, -0.1, 0.6),
			}},
		},
		Skin: Skin{Flank: 1.2, Back: 1.4, Belly: 1.0, BackDepth: 2, BellyDepth: 2, Relief: 0.9, Seed: 47},
		Folds: []Fold{
			{Pts: []float64{33.6, 12.5, 32.4, 16, 31.6, 19.5}, N: 3, DX: 0.9, DY: 0.2, Depth: 0.6}, // the bare neck's creases
			{Pts: []float64{16, 46.5, 18.2, 46.5}, N: 4, DY: 1.4, Depth: 0.5},
			{Pts: []float64{25, 47.5, 27.6, 47.5}, N: 4, DY: 1.3, Depth: 0.5},
		},
		Details: anzuDetails,
	}
}

func anzuPlumage() *Plumage {
	return &Plumage{Seed: 17, Edge: 0.4, Tone: 0.85, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(11, 30, 23, 30, 23, 38.8, 11, 38.8), Parts: []string{"farLeg"},
			Flow: Along(18, 31, 17.5, 40, 16, 45), Len: 2.4, Wid: 1.7, Fluff: 0.5, Snap: true},
		{Name: "body", Region: Poly(11, 19.5, 34, 19.5, 36.5, 24, 36, 31, 30, 35, 13, 34), Parts: []string{"body", "wing"},
			Flow: Along(34, 20.5, 28, 21.6, 20, 23.2, 13, 24.2), Len: 2.7, Wid: 1.9, Rows: 0.45, Jitter: 0.2, Fluff: 0.55, Snap: true,
			Hang: Poly(-1, 34.5, 51, 34.5, 51, 56, -1, 56)},
		// The fan of the tail, its long feathers spreading back and up.
		{Name: "tail", Region: Poly(-1, 14, 12.5, 17, 14, 24, 12.5, 32, -1, 32), Parts: []string{"body"},
			Flow: Fan(15, 25, 9, -2.85), Len: 5, Wid: 1.6, Rows: 0.34, Tip: 0.4, Shaft: 0.14, Fluff: 0.8, Grow: 0.02},
		{Name: "nearThigh", Region: Poly(21, 30, 34, 30, 34, 38.8, 21, 38.8), Parts: []string{"nearLeg"},
			Flow: Along(27, 31, 27.5, 40, 25.5, 46.5), Len: 2.4, Wid: 1.7, Fluff: 0.5, Snap: true},
		// The folded wing: orange, its feathers edged in cream along the bottom.
		{Name: "wing", Region: Poly(33.4, 26.2, 32.6, 30.8, 26, 32.2, 14, 33.8, 13, 30.4, 18, 28, 26, 26.5, 30, 25.8), Parts: []string{"wing"},
			Flow: Along(33, 27.5, 27, 28.8, 20, 30.4, 13, 31.4), Len: 4.4, Wid: 1.5, Rows: 0.36, Tip: 0.5, Shaft: 0.1, Fluff: 0.4, Snap: true},
	}}
}

func anzuDetails(p *Painter) {
	p.Plumage(anzuPlumage())
	// The casque: a plate of horn, lit flat, with growth lines up it.
	p.Plane(PlaneSpec{Region: Poly(33.3, 7.6, 33.5, 3.4, 35.6, 0.3, 42.8, 0.1, 44.8, 2.6, 45.4, 6.8, 41, 7.4),
		Mat: []string{"gold", "coral", "red"}, Tilt: 0.15, Angle: 1.35, Gap: 1.1, Lines: 0.07, FromX: 36, FromY: 2, Fall: 0.03, Seed: 3})
	p.Beak(BeakSpec{Mat: "beak", Ridge: []float64{39.8, 7.1, 44, 7.3, 47, 8.6, 48.2, 10.6},
		Mouth: []float64{39.6, 11.2, 43.5, 11.4, 46.8, 11.7}, Gloss: 0.22, Nostril: [2]float64{44.6, 8.8}, NoseR: 0.5,
		LowerShadow: 0.14, GrainSeed: 5})
	p.Eye(EyeSpec{X: 38.3, Y: 9.2, R: 0.8, Iris: anzuIris, Lid: 0.2, Squint: 1.15})
	// The claws: the hand's under the wing, and three on each foot.
	p.Claw(21.4, 53.6, 23.0, 54.4, 0.7, 0.2, anzuClaws)
	p.Claw(34.4, 53.5, 36.6, 54.4, 0.8, 0.2, anzuClaws)
	p.Claw(32.2, 53.0, 33.8, 54.2, 0.6, 0.2, anzuClaws)
}
