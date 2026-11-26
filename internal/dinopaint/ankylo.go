package dinopaint

import "math"

// The armoured one: dinos/ankylo.png, 104 x 50, facing right. Its rig is
// "ankylo" in internal/client/dinorigs.go: legs near hind (hip 44, 30), far
// hind (53, 31), far fore (65.5, 33), near fore (74, 32); the head turns
// about (81, 30); the beak opens about (92.5, 39.6) along the line to
// (102.8, 39.4); the tail, with its club, swings about (29, 24).
//
// Its back is armour (armour.go): bands of keeled plates with pale rims of
// bone, short cones along the spine, blades along the flanks and over the
// shoulders, bony half-rings across the neck, tiles over the skull, horns
// at the corners of the head; the tail stiff and spiked, ending in a club
// of two knobs. The legs come out from under the armour's edge.

var (
	ankyloArmour = []uint32{0x121316, 0x1b1c1e, 0x262724, 0x32332d, 0x404137, 0x505142, 0x63634f, 0x787760}
	ankyloRim    = []uint32{0x2b2219, 0x45382a, 0x625039, 0x80694c, 0x9c8462, 0xb59e7a, 0xcab693, 0xddcdad}
	ankyloSkin   = []uint32{0x110e0d, 0x1a1513, 0x241e1a, 0x302822, 0x3d332b, 0x4b3f35, 0x5a4c40, 0x6b5b4c}
	ankyloBelly  = []uint32{0x241f1a, 0x332b24, 0x443a30, 0x564a3d, 0x6a5c4b, 0x7e6f5a, 0x91826b, 0xa3957d}
	ankyloBeak   = []uint32{0x100d0b, 0x1a1512, 0x251e19, 0x312821, 0x3e332a, 0x4c4034, 0x5b4d3f, 0x6b5c4c}
	ankyloSpike  = []uint32{0x17161a, 0x26241f, 0x3a3629, 0x544c38, 0x726549, 0x938260, 0xb19f7b, 0xcdbd99, 0xe2d5b6}
	ankyloNail   = []uint32{0x2c2419, 0x55483a, 0x837360, 0xb0a084, 0xd2c6aa}
	ankyloIris   = []uint32{0x0c0806, 0x1c1208, 0x2e1e0c, 0x46300f, 0x624415, 0x80591c}
)

func init() {
	Register(ankylo())
	// Its legs, in the rig's order, cut as painted and swung whole from the
	// hip and the shoulder under the armour's edge, which stays in front of
	// them, the legs' tops painted on under it (thigh.go).
	RegisterThighs("ankylo", &Thighs{
		Legs:  LegMask{{"nearHind"}, {"farHind"}, {"farFore"}, {"nearFore"}},
		Front: func(x, y float64) float64 { return y - ankyloSkirt(x) - 1.5 },
		Thigh: []SDF{Cone(44, 29, 7.4, 45, 39, 5.4), Cone(53, 30, 6.4, 53.5, 39.5, 4.8),
			Cone(65.5, 32, 5.6, 65, 40, 4.4), Cone(74, 31, 6.4, 73.5, 40, 5)},
	})
}

// ankyloSkirt is the lower edge of the armour down the flank: the legs come
// out from under it, the blades stand along it.
func ankyloSkirt(x float64) float64 { return 30.5 + 0.0016*(x-50)*(x-50) }

// The masses of the ankylosaur, as its guide was drawn.
var (
	ankyloTail     = Path([3]float64{31, 22, 8.5}, [3]float64{22, 25, 5}, [3]float64{15, 28, 2.8}, [3]float64{10, 30, 2.3})
	ankyloBody     = Blend(4, Ellipse(47, 23, 20, 14.2, 0), Ellipse(67, 26.5, 15.5, 10.8, 0), ankyloTail)
	ankyloNeck     = Cone(78, 27, 8.5, 87, 33, 6.2)
	ankyloHeadPoly = Poly(84.5, 29.5, 88, 27.6, 92, 28.2, 96, 29.8, 99.5, 32, 102, 34.6, 103.4, 37.4, 102.8, 40.2,
		100, 41.6, 95, 42.2, 90, 42, 86.5, 40.5, 85, 37)
	ankyloHead     = Blend(2, ankyloHeadPoly, ankyloNeck)
	ankyloClubNear = Ellipse(8.5, 31.5, 6.2, 4.6, 0)
	ankyloClubTip  = Ellipse(2.8, 32, 2.1, 2.3, 0)
	ankyloClubFar  = Ellipse(6.6, 28.7, 4.5, 3.4, 0)
)

// underArmour cuts a leg off where the armour's edge comes down over it.
func underArmour(leg SDF) SDF {
	return Inter(leg, func(x, y float64) float64 { return ankyloSkirt(x) + 0.2 - y })
}

func ankylo() *Species {
	farHind := Union(Cone(53, 30, 6, 53.5, 39.5, 4.4), Cone(53.5, 39.5, 4, 53.2, 45, 3.8), Ellipse(54.4, 45.9, 5, 2.1, 0))
	farFore := Union(Cone(65.5, 32, 5.2, 65, 40, 4.0), Cone(65, 40, 3.8, 65.4, 44.6, 3.6), Ellipse(66.4, 45.9, 4.8, 2.1, 0))
	nearHind := Union(Cone(44, 29, 7, 45, 39, 5), Cone(45, 39, 4.6, 44.6, 45.5, 4.3), Ellipse(45.6, 47.2, 5.8, 2.4, 0))
	nearFore := Union(Cone(74, 31, 6, 73.5, 40, 4.6), Cone(73.5, 40, 4.4, 74, 45.4, 4.2), Ellipse(75.2, 47.2, 5.4, 2.4, 0))
	masses := Union(farHind, farFore, nearHind, nearFore, ankyloBody, ankyloHead, ankyloClubNear, ankyloClubFar, ankyloClubTip)
	beak := Poly(99.3, 35.4, 102, 34.7, 103.5, 37.4, 102.9, 40.3, 100, 41.5, 98.8, 38.5)
	// Under the armour: the flanks below its edge, the throat and the jaw.
	below := func(x, y float64) float64 { return ankyloSkirt(x) - y }
	return &Species{
		Name:        "ankylo",
		GuideDetail: 0.12,
		Ink:         28,
		// Every spike and horn is painted again: the guide's, standing out of
		// the masses, are cut away first.
		Remove: []SDF{func(x, y float64) float64 { return 0.35 - masses(x, y) }},
		Recolor: []Recolor{
			// The guide's plates and rims on the back are armour, to be laid
			// again as plates; its bone in the masses was spikes.
			{Region: Cut(Union(ankyloBody, ankyloClubNear, ankyloClubFar, ankyloClubTip), below), From: []string{"skin", "bone", "belly"}, To: "armour"},
			{Region: Union(farHind, farFore, nearHind, nearFore), From: []string{"bone", "armour"}, To: "skin"},
			{Region: ankyloHead, From: []string{"bone"}, To: "armour"},
		},
		Materials: []Material{
			{Name: "armour", Seeds: []uint32{0x23241f, 0x30322a, 0x3f4136, 0x4f5143, 0x606251, 0x737461}, Ramp: ankyloArmour, Relief: 0.8},
			{Name: "bone", Seeds: []uint32{0x6d5c42, 0x8f7b5a, 0xb09a74, 0xcbb690, 0xe0cfaa}, Ramp: ankyloRim, Relief: 0.4},
			{Name: "skin", Seeds: []uint32{0x251f1b, 0x332a24, 0x433830, 0x54473c, 0x655649}, Ramp: ankyloSkin, Relief: 1.1},
			{Name: "belly", Seeds: []uint32{0x5d5444, 0x716753, 0x877c64, 0x9b9077}, Ramp: ankyloBelly, Relief: 0.9},
			{Name: "beak", Ramp: ankyloBeak, Relief: 0.3, Region: beak},
			// The club: fused bone, paler than the back.
			{Name: "knob", Ramp: []uint32{0x14110e, 0x1f1a15, 0x2b251d, 0x3a3227, 0x4c4233, 0x615542, 0x7a6c55, 0x95866b}, Relief: 0.5},
		},
		Parts: []Part{
			{Name: "farHind", Shape: underArmour(farHind), Far: true, Round: 3.4, Scale: 0.85, Mat: "skin", Muscles: []Muscle{
				Mu(53.3, 36, 3.8, 3.6, 0, 0.9),
				Mu(54.4, 45.6, 3.4, 1.4, 0, 0.5),
			}},
			{Name: "farFore", Shape: underArmour(farFore), Far: true, Round: 3.2, Scale: 0.85, Mat: "skin", Muscles: []Muscle{
				Mu(65.3, 37, 3.4, 3.4, 0, 0.8),
				Mu(66.4, 45.6, 3.2, 1.4, 0, 0.5),
			}},
			// The club: the far knob, behind the near one.
			{Name: "clubFar", Shape: ankyloClubFar, Round: 3, Soft: 0.6, Scale: 1.5, Mat: "knob", Muscles: []Muscle{
				Mu(6.2, 28, 3, 2.2, 0, 1.2),
			}},
			{Name: "body", Shape: ankyloBody, Round: 9, Muscles: []Muscle{
				Mu(46, 21, 15, 11, 0, 1.4),       // the dome over the hips
				Mu(66, 25, 10, 7.5, 0, 0.9),      // the shoulders
				Mu(50, 35, 15, 2.6, 0, 0.5),      // the belly
				Mu(25, 23.5, 7, 4, -0.25, 1.0),   // the great muscles at the root of the tail
				Mu(15, 28.2, 5, 1.8, -0.35, 0.5), // the stiff handle of the club
			}},
			// The near knob, a great dome of bone, and a small one at the tip.
			{Name: "club", Shape: ankyloClubNear, Round: 4.2, Soft: 0.6, Scale: 1.5, Mat: "knob", Muscles: []Muscle{
				Mu(8.2, 30.8, 4.4, 3.4, 0, 1.6),
			}},
			{Name: "clubTip", Shape: ankyloClubTip, Round: 1.8, Soft: 0.6, Scale: 1.2, Mat: "knob", Muscles: []Muscle{
				Mu(2.6, 31.6, 1.4, 1.6, 0, 0.7),
			}},
			{Name: "head", Shape: Blend(2, ankyloHeadPoly, Cone(81, 28.5, 6.8, 87, 33, 6.2)), Round: 3.6, Soft: 0.7, Scale: 0.7, Onto: "body", Seam: Below(70, 36, 86, 41), Muscles: []Muscle{
				Mu(81.5, 29, 4, 3.6, 0.5, 0.8),   // the neck
				Mu(90.5, 30.5, 4, 2.2, 0.2, 0.8), // the flat of the skull
				Mu(92.5, 37.6, 3.4, 2.6, 0, 1.1), // the cheek
				Mu(93.6, 32.7, 2.4, 1.0, 0, 1.3), // the brow over the eye
				Mu(101, 38, 2, 2.2, 0, 0.6),      // the beak
			}},
			{Name: "nearHind", Shape: underArmour(nearHind), Round: 4.4, Onto: "body", Muscles: []Muscle{
				Mu(44.6, 35.5, 5, 4, 0, 1.4), // the thigh
				Mu(45, 42, 3, 2.4, 0, 0.6),   // the knee
				Mu(45.6, 47.2, 4.2, 1.6, 0, 0.6),
			}},
			{Name: "nearFore", Shape: underArmour(nearFore), Round: 4.2, Onto: "body", Muscles: []Muscle{
				Mu(74, 35.5, 4.6, 4, 0, 1.3), // the upper arm
				Mu(73.6, 42, 2.8, 2.4, 0, 0.6),
				Mu(75.2, 47.2, 3.8, 1.6, 0, 0.6),
			}},
		},
		Skin: Skin{Flank: 1.25, Back: 1.5, Belly: 1.05, BackDepth: 3, BellyDepth: 3, Relief: 1.1, Seed: 61},
		Folds: []Fold{
			{Pts: []float64{92.5, 39.6, 97.5, 39.5, 102.6, 39.3}, N: 1, Depth: 1.8}, // the line of the mouth
			{Pts: []float64{41.2, 43.6, 48.4, 43.9}, N: 3, DY: 1.2, Depth: 1.0},     // wrinkled legs, as an elephant's
			{Pts: []float64{70, 43.6, 77.6, 43.9}, N: 3, DY: 1.2, Depth: 1.0},
			{Pts: []float64{50.6, 43, 56, 43.2}, N: 2, DY: 1.2, Depth: 0.7},
			{Pts: []float64{62.6, 43, 68, 43.2}, N: 2, DY: 1.2, Depth: 0.7},
			{Pts: []float64{84, 33, 83.6, 36.5, 84.2, 40}, N: 2, DX: 1.6, Depth: 0.9}, // the skin of the neck between its rings
		},
		Details: ankyloDetails,
	}
}

func ankyloDetails(p *Painter) {
	carapace := Cut(ankyloBody, func(x, y float64) float64 { return ankyloSkirt(x) + 0.4 - y })
	tailTop := func(x, y float64) float64 { return y - (22 + (31-x)*0.4) - 2.6 } // the tail armoured down its sides
	// The back: bands of keeled plates, big over the hips, smaller down the
	// flanks, a row of them cresting the outline.
	p.Scutes(ScuteSpec{
		Region: Inter(carapace, func(x, y float64) float64 { return 29.5 - x }), Parts: []string{"body"},
		X0: 28.5, X1: 82, Band: 6.2, Rows: []float64{5.2, 4.8, 4.3, 3.8, 3.2, 2.8}, Gap: 0.45, Grow: -0.07,
		Keel: 0.9, Dome: 0.42, Crest: 0.8, Jitter: 0.3, Seed: 3,
		PlateMat: "armour", RimMat: "bone", SkinMat: "skin",
	})
	// The tail: smaller plates down its sides.
	p.Scutes(ScuteSpec{
		Region: Inter(Inter(ankyloBody, func(x, y float64) float64 { return x - 29.5 }), tailTop), Parts: []string{"body"},
		X0: 7, X1: 30, Band: 3.9, Rows: []float64{3.0, 2.8, 2.4}, Gap: 0.7, Keel: 0.5, Crest: 0.5, Jitter: 0.25, Seed: 5,
		PlateMat: "armour", RimMat: "bone", SkinMat: "skin",
	})
	// The club: knobs of fused bone, pitted, pale where they are worn.
	club := Union(ankyloClubNear, ankyloClubFar, ankyloClubTip)
	p.Speck(club, 0.07, -0.16, 71)
	p.Speck(club, 0.04, 0.1, 73)
	// The bony half-rings across the neck.
	rings := Union(Inter(ankyloNeck, Inter(func(x, y float64) float64 { return math.Abs(x-81.6) - 1.9 }, func(x, y float64) float64 { return y - 31 })),
		Inter(ankyloNeck, Inter(func(x, y float64) float64 { return math.Abs(x-86) - 1.7 }, func(x, y float64) float64 { return y - 34 })))
	p.Scutes(ScuteSpec{
		Region: rings, Parts: []string{"head"},
		X0: 79.4, X1: 88, Band: 2.2, Rows: []float64{3.4, 3.0, 2.6}, Gap: 0.55, Keel: 0.8, Crest: 0.4, Seed: 13,
		PlateMat: "armour", RimMat: "bone", SkinMat: "skin",
	})
	// Small tiles over the skull.
	skull := Inter(ankyloHead, func(x, y float64) float64 { return math.Max(85.6-x, y-(36.6-0.1*(x-92))) })
	p.Scutes(ScuteSpec{
		Region: Cut(skull, Union(Circle(93.5, 34.3, 1.5), Poly(99, 33, 104, 33, 104, 42, 99, 42))), Parts: []string{"head"},
		X0: 84.5, X1: 100, Band: 3.0, Rows: []float64{2.7, 2.4, 2.1}, Gap: 0.4, Dome: 0.34, Rim: 0.22, Jitter: 0.3, Seed: 17,
		PlateMat: "armour", RimMat: "bone", SkinMat: "skin",
	})
	// The armour's edge shades the legs just under it.
	for _, n := range []string{"nearHind", "nearFore", "farHind", "farFore"} {
		j := p.PartID(n)
		for i, in := range p.In {
			if !in || p.Part[i] != j {
				continue
			}
			x, y := G(i%p.W, i/p.W)
			p.Val[i] -= 0.3 * math.Exp(-(y-ankyloSkirt(x))/1.6)
		}
	}
	p.Redraw()

	// Spikes on the body, which stay with it when a leg steps.
	mark := p.Mark()
	// Short cones along the spine.
	for i, x := range []float64{33.2, 38.6, 44, 49.4, 54.8, 60.2, 65.6, 71, 76} {
		t := float64(p.TopEdge(int(x*K))) / K
		h := 2.4 + 0.6*math.Sin(float64(i)*1.7)
		p.Horn(HornSpec{X0: x + 0.4, Y0: t + 1.4, X1: x - 0.6, Y1: t - h, W: 2.3, Bend: 0.2, Ramp: ankyloSpike})
	}
	// Blades along the flank, at the armour's edge, pointing out and back;
	// bigger towards the shoulders.
	for i, x := range []float64{36, 42, 48, 54, 60, 66} {
		y := ankyloSkirt(x) + 0.2
		l := 4.4 + 0.45*float64(i)
		p.Horn(HornSpec{X0: x + 0.6, Y0: y - 0.6, X1: x - l, Y1: y + l*0.36, W: 3.1 + 0.15*float64(i), Bend: 0.7, Ramp: ankyloSpike, Grooves: true})
	}
	// The great blades over the shoulder.
	p.Horn(HornSpec{X0: 77, Y0: 24.2, X1: 68.6, Y1: 17.2, W: 4.4, Bend: 1.1, Ramp: ankyloSpike, Grooves: true})
	p.Horn(HornSpec{X0: 80.6, Y0: 27.6, X1: 73.2, Y1: 22.8, W: 3.8, Bend: 0.9, Ramp: ankyloSpike, Grooves: true})
	// The keels of the neck's half-rings.
	p.Horn(HornSpec{X0: 81.8, Y0: 23.4, X1: 80.4, Y1: 19.8, W: 2.5, Bend: 0.3, Ramp: ankyloSpike})
	p.Horn(HornSpec{X0: 86.2, Y0: 27, X1: 84.9, Y1: 24.2, W: 2.2, Bend: 0.3, Ramp: ankyloSpike})
	// Small spikes down both sides of the tail, the lower ones darker.
	for i, x := range []float64{13.6, 18, 22.4, 26.8} {
		yt := 28 - (x-14)*0.33
		r := 2.6 + float64(i)*0.85
		p.Horn(HornSpec{X0: x + 0.4, Y0: yt - r*0.5, X1: x - 3.4, Y1: yt - r*0.5 - 2.4, W: 2.5, Bend: 0.4, Ramp: ankyloSpike})
		p.Horn(HornSpec{X0: x + 0.4, Y0: yt + r*0.5, X1: x - 3.5, Y1: yt + r*0.5 + 2.0, W: 2.4, Bend: 0.4, Ramp: ankyloSpike, Shade: -0.2})
	}
	p.Claim(mark, "body")

	// The head: horns at the back corners of the skull and on the cheek.
	mark = p.Mark()
	p.Horn(HornSpec{X0: 88.4, Y0: 29.8, X1: 84, Y1: 25, W: 3.8, Bend: 0.6, Ramp: ankyloSpike, Grooves: true})
	p.Horn(HornSpec{X0: 90.4, Y0: 39.2, X1: 87, Y1: 44.4, W: 3.6, Bend: 0.5, Ramp: ankyloSpike, Grooves: true})
	// The bony brow the eye sits deep under.
	p.Horn(HornSpec{X0: 91, Y0: 33.2, X1: 96.6, Y1: 32.4, W: 2.2, Bend: -0.3, Ramp: ankyloSpike, Shade: -0.1})
	p.Claim(mark, "head")
	// The eye, small and deep under its brow; the nostril.
	p.Eye(EyeSpec{X: 93.6, Y: 34.4, R: 0.75, Iris: ankyloIris, Lid: 0.45, Squint: 1.25, Brow: -1})
	p.Nostril(100.8, 35.8, 0.6, 0.4, -0.4)
	// Hoof-like nails round the front of each broad foot; the far ones darker.
	for _, n := range [][3]float64{{46.6, 49.4, 1}, {48.8, 49.5, 1.05}, {50.7, 49.2, 0.9}, {75.8, 49.4, 1}, {77.9, 49.5, 1.05}, {79.6, 49.2, 0.85}} {
		p.Nail(n[0], n[1], 2.2*n[2], 1.5*n[2], ankyloNail, -0.1)
	}
	for _, n := range [][2]float64{{56.6, 47.8}, {58.5, 47.6}, {68.2, 47.8}, {70, 47.6}} {
		p.Nail(n[0], n[1], 1.8, 1.25, ankyloNail, -0.45)
	}
}
