package dinopaint

import "math"

// The plated one: dinos/stego.png, 92 x 56, facing right. Its rig is
// "stego" in internal/client/dinorigs.go: legs near hind (hip 40.5, 36),
// far hind (50, 37), far fore (58, 38), near fore (66, 37); the head turns
// about (73, 37.5), the beak opens about (82, 42.5) along the line to (89.5,
// 43); the end of the tail, with its spikes, swings about (17.5, 30).

var (
	plateFace = []uint32{0x0c0806, 0x140c08, 0x1e140c, 0x2a1c10, 0x382614, 0x48321a, 0x5a4022, 0x6e502c}
	plateRim  = []uint32{0x22160c, 0x3a2614, 0x56381c, 0x744e26, 0x926632, 0xae8042, 0xc69c56, 0xdab870}
	plateVein = []uint32{0x3e3012, 0x5c4818, 0x7c6420, 0x9a7e2a, 0xb69836, 0xccb04a, 0xdec462, 0xeed880}
	spikeBone = []uint32{0x4e4838, 0x7a725e, 0xa69c84, 0xcac0a6, 0xe6dec6, 0xfaf6ea}
	stegoHoof = []uint32{0x2a2a1c, 0x56523e, 0x86806a, 0xb4ae96, 0xd8d4c0}
	stegoIris = []uint32{0x0a0604, 0x140c06, 0x22160a, 0x34220e, 0x4a3214, 0x60421c}
	stegoHide = []uint32{0x12160a, 0x1e240e, 0x2c3414, 0x3c461a, 0x4e5a20, 0x626e28, 0x7a8432, 0x949a40}
	stegoRust = []uint32{0x1c0c06, 0x2c140a, 0x40200e, 0x582c12, 0x703a18, 0x8a4a1e, 0xa45c26, 0xbc7030}
)

// stegoThighs are its legs' upper parts, thigh and upper arm, which turn whole
// with the hip and the shoulder: near hind, far hind, far fore, near fore.
var stegoThighs = []SDF{
	Cone(40.5, 33, 6.9, 41.5, 45, 5.1), Cone(50, 35, 5.7, 50.5, 45, 4.5),
	Cone(58, 37, 4.9, 57.5, 45, 4.1), Cone(66, 36.5, 5.9, 65.5, 45, 4.9),
}

func init() {
	Register(stego())
	// Its legs, in the rig's order, cut as painted and swung whole from the
	// hip and the shoulder (thigh.go).
	RegisterThighs("stego", &Thighs{
		Legs:  LegMask{{"nearHind"}, {"farHind"}, {"farFore"}, {"nearFore"}},
		Thigh: stegoThighs,
	})
}

// The plates, back to front: the far row first, darker, between the near
// ones.
var stegoPlates = []PlateSpec{
	// The far row.
	{BX0: 29.4, BY0: 21.5, BX1: 37.4, BY1: 20.5, TX: 33.6, TY: 6, Bulge: 1.6},
	{BX0: 43.2, BY0: 20, BX1: 50.6, BY1: 20, TX: 47, TY: 3.6, Bulge: 1.6},
	{BX0: 53.6, BY0: 21.5, BX1: 60.4, BY1: 22.6, TX: 57.2, TY: 6.6, Bulge: 1.5},
	{BX0: 60.8, BY0: 25.5, BX1: 67.4, BY1: 26.5, TX: 64.6, TY: 15.4, Bulge: 1.2},
	// The near row.
	{BX0: 18.6, BY0: 29.6, BX1: 25.8, BY1: 28.2, TX: 21.8, TY: 17.2, Bulge: 1.2}, // clear of the tail's joint, at 17.5
	{BX0: 22.5, BY0: 24.5, BX1: 31.2, BY1: 22.5, TX: 27.8, TY: 10.8, Bulge: 1.8},
	{BX0: 35.8, BY0: 20.5, BX1: 44.4, BY1: 19.8, TX: 40.4, TY: 3.4, Bulge: 1.9},
	{BX0: 48.6, BY0: 20.2, BX1: 56.2, BY1: 20.8, TX: 52.4, TY: 6.4, Bulge: 1.6},
	{BX0: 56.8, BY0: 23.2, BX1: 64.2, BY1: 24.8, TX: 61, TY: 12, Bulge: 1.5},
	{BX0: 63.6, BY0: 28.5, BX1: 70.4, BY1: 29.5, TX: 67, TY: 21, Bulge: 1.2},
}

const stegoFar = 4 // the first four plates are the far row

func stego() *Species {
	nearHind := Union(
		Cone(40.5, 33, 6.4, 41.5, 45, 4.6),
		Cone(41.5, 45, 4.3, 41, 51, 3.8),
		Ellipse(41.6, 53.3, 4.6, 2.5, 0),
	)
	farHind := Union(
		Cone(50, 35, 5.2, 50.5, 45, 4),
		Cone(50.5, 45, 3.8, 50, 50, 3.4),
		Ellipse(50.8, 51.7, 4, 2, 0),
	)
	farFore := Union(
		Cone(58, 37, 4.4, 57.5, 45, 3.6),
		Cone(57.5, 45, 3.4, 58, 50, 3.1),
		Ellipse(58.6, 51.7, 3.6, 2, 0),
	)
	nearFore := Union(
		Cone(66, 36.5, 5.4, 65.5, 45, 4.4),
		Cone(65.5, 45, 4.2, 66, 51, 3.7),
		Ellipse(66.6, 53.3, 4.4, 2.5, 0),
	)
	tail := Path([3]float64{0.8, 33, 1.1}, [3]float64{8, 32, 2.5}, [3]float64{15, 31, 4}, [3]float64{24, 29, 7})
	body := Blend(3,
		tail,
		Ellipse(45, 29.8, 24, 11, 0),
		Ellipse(45, 22.5, 19, 6, 0),  // the high back under the plates
		Ellipse(63, 34, 7.5, 7.5, 0), // the shoulders
	)
	head := Blend(2,
		Cone(63.5, 32, 6.5, 73.5, 37.6, 4.2), // the neck, low, the head carried near the ground
		Poly(71, 33.4, 75, 32.8, 80, 33.4, 84, 34.6, 87.6, 36.4, 90.4, 38.6, 91.2, 41.2, 89.8, 43.4, 84, 43.8, 78, 42.8, 72.5, 42.2, 69.5, 41),
	)
	beak := Poly(87.6, 38.8, 89.4, 38.4, 91.4, 40.6, 91, 43.2, 88.6, 44, 87.4, 42.6)
	// The underside, rusty: below a wavering line along the belly (not the
	// near thigh and upper arm), the throat, the legs below the knees.
	bellyLine := func(x, y float64) float64 {
		return 35.6 + 0.004*(x-45)*(x-45) + 2.4*(fbm(x*0.2, y*0.2, 13, 2)-0.5) - y
	}
	throat := func(x, y float64) float64 { return 39.6 + (x-64)*0.17 + 1.2*(fbm(x*0.3, y*0.3, 17, 2)-0.5) - y }
	legs := func(x, y float64) float64 { return 46.6 + 1.6*(fbm(x*0.3, y*0.2, 19, 2)-0.5) - y }
	under := Union(
		Cut(bellyLine, Union(Cone(40.5, 33, 6.4, 41.5, 45, 4.6), Cone(66, 36.5, 5.4, 65.5, 45, 4.4))),
		Inter(throat, head),
		legs,
	)
	return &Species{
		Name:        "stego",
		GuideDetail: 0.4,
		Ink:         28,
		// The plates and the tail's spikes are painted again: the guide's
		// are cut out of the silhouette first, all of it above the back and
		// round the end of the tail.
		Remove: []SDF{
			Cut(Poly(15, 0, 75, 0, 75, 32, 15, 32), Union(body, head)),
			Cut(Poly(-1, 14, 15.5, 14, 15.5, 46, -1, 46), tail),
		},
		Recolor: []Recolor{
			// What is left of the guide's plates on the back is back.
			{Region: Poly(0, 0, 92, 0, 92, 56, 0, 56), From: []string{"plate", "bone"}, To: "hide"},
		},
		Materials: []Material{
			// Olive green, lime and ochre where the light catches it.
			{Name: "hide", Seeds: []uint32{0x495d2a, 0x445c30, 0x5a7640, 0x64803a, 0x728f52, 0x81a04d, 0x8ca866, 0x9fbb64, 0xbcd284, 0x1c2612, 0x1f2c17, 0x304322, 0x31401d, 0x2b2f1b, 0x29371d, 0x2c3819},
				Ramp:   stegoHide,
				Region: Poly(30, 49, 75, 49, 75, 57, 30, 57), Only: []string{"bone", "belly"}}, // the guide's nails are painted again
			{Name: "belly", Seeds: []uint32{0x99874b, 0xa09276, 0x938b74, 0x76743c, 0xc4c074, 0xe0dc98, 0x9e9c54},
				Ramp: stegoHide, Relief: 0.7},
			{Name: "plate", Seeds: []uint32{0xe0882a, 0xc4621c, 0xf47a4a, 0xe45232, 0xc4301e, 0x9a4216, 0x6a2a12, 0x4e130c}, Ramp: stegoHide},
			{Name: "bone", Seeds: []uint32{0xeee6cc, 0xfffaea, 0xd2c6a6, 0xa89a7c}, Ramp: stegoHide},
			// The throat, the belly and the lower legs shade to rust.
			{Name: "rust", Ramp: stegoRust, Relief: 0.8, Region: under, Only: []string{"hide", "belly"}},
			// Bold dark bands across it, as a tiger's: thick over the back,
			// thinning down the flanks, broken and forking.
			{Name: "stripe", Ramp: []uint32{0x050604, 0x090b06, 0x0e1108, 0x13170b, 0x191e0e, 0x1f2612, 0x272f16, 0x30391b},
				Region: stegoStripes, Only: []string{"hide", "rust"}},
			// Streaks of yellow ochre between them, high on the flanks.
			{Name: "ochre", Ramp: []uint32{0x241e08, 0x382e0c, 0x504212, 0x6a5818, 0x86701e, 0xa28a28, 0xbaa236, 0xd2ba4a},
				Region: stegoOchre, Only: []string{"hide"}},
			{Name: "beak", Ramp: []uint32{0x241c08, 0x3a2c0c, 0x544212, 0x6e5a1a, 0x8a7424, 0xa68e30, 0xbea840, 0xd4c058}, Relief: 0.3,
				Region: beak},
		},
		Parts: []Part{
			{Name: "farHind", Shape: farHind, Far: true, Round: 3.4, Scale: 0.85, Muscles: []Muscle{
				Mu(50, 39, 3.8, 4.4, 0, 1.0),
			}},
			{Name: "farFore", Shape: farFore, Far: true, Round: 3.2, Scale: 0.85, Muscles: []Muscle{
				Mu(58, 40, 3.4, 4, 0, 0.9),
			}},
			{Name: "body", Shape: body, Round: 8, Muscles: []Muscle{
				Mu(39, 28, 9, 7, 0, 1.6),        // the great hip
				Mu(52, 30, 9, 6.5, 0, 1.0),      // the ribs
				Mu(63, 33, 5, 5.5, 0.2, 1.3),    // the shoulder
				Mu(46, 37.5, 12, 3, 0, 0.6),     // the belly
				Mu(22, 31, 7, 3.2, -0.2, 1.2),   // the tail's great muscle
				Mu(10, 32.4, 6, 1.8, -0.1, 0.7), // the tail
				Mu(44, 22.5, 15, 2.6, 0, 0.6),   // the long muscles of the back
			}},
			{Name: "head", Shape: head, Round: 3.6, Soft: 0.65, Scale: 0.7, Onto: "body", Seam: Below(60, 39, 72, 46), Muscles: []Muscle{
				Mu(67.5, 34.5, 4, 3.4, 0.5, 1.1), // the neck
				Mu(78.5, 37.2, 3.2, 2.4, 0, 1.2), // the cheek
				Mu(83.8, 37.4, 1.8, 1.0, 0, 0.9), // the brow
				Mu(88.6, 40.6, 2.2, 1.8, 0, 0.7), // the beak
			}},
			{Name: "nearHind", Shape: nearHind, Round: 4.4, Onto: "body", Seam: Below(25, 38.5, 48, 40), Muscles: []Muscle{
				Mu(40, 37.5, 4.8, 5.6, 0, 1.8), // the thigh
				Mu(41.4, 45.5, 2.4, 2.2, 0, 0.6),
				Mu(41.8, 53, 3.4, 1.7, 0, 0.6),
			}},
			{Name: "nearFore", Shape: nearFore, Round: 4.2, Onto: "body", Seam: Below(55, 40, 75, 40), Muscles: []Muscle{
				Mu(66, 40, 4, 4.8, 0, 1.5), // the upper arm
				Mu(65.6, 45.5, 2.4, 2.2, 0, 0.6),
				Mu(66.8, 53, 3.2, 1.7, 0, 0.6),
			}},
		},
		Skin: Skin{Flank: 1.6, Back: 2.4, Belly: 1.3, BackDepth: 4, BellyDepth: 4, Seed: 41},
		Osteoderms: []Row{
			// Pebbly skin, with bigger round scutes on the thighs, the shoulders
			// and the neck, and a row of studs low on the flank.
			{X0: 26, X1: 60, Off: 13.5, Gap: 3.4, R: 0.75, Parts: []string{"body"}},
			{X0: 36.5, X1: 45, Off: 15.5, Gap: 2.6, R: 0.95, Parts: []string{"nearHind"}},
			{X0: 37, X1: 44.5, Off: 18.5, Gap: 2.8, R: 0.85, Parts: []string{"nearHind"}},
			{X0: 61.5, X1: 69.5, Off: 11.5, Gap: 2.6, R: 0.9, Parts: []string{"nearFore", "body"}},
			{X0: 62, X1: 69, Off: 14.2, Gap: 2.7, R: 0.8, Parts: []string{"nearFore"}},
			{X0: 66, X1: 80, Off: 2.6, Gap: 2.4, R: 0.8, Parts: []string{"head"}},
		},
		Folds: []Fold{
			{Pts: []float64{39.2, 44.2, 43.8, 44.6}, N: 2, DY: 1.3, Depth: 1.1}, // the knees
			{Pts: []float64{63.4, 44.2, 67.8, 44.6}, N: 2, DY: 1.3, Depth: 1.1},
			{Pts: []float64{62.5, 30.5, 61.6, 34.5, 62.2, 38.5}, N: 2, DX: 1.6, Depth: 1.1},        // the root of the neck
			{Pts: []float64{69, 33.5, 68.2, 37.5, 68.8, 41.5}, N: 3, DX: 1.6, DY: 0.2, Depth: 1.0}, // the neck
			{Pts: []float64{38, 47, 44.5, 46.8}, N: 3, DY: 1.4, Depth: 0.9},                        // wrinkled legs
			{Pts: []float64{62.5, 47, 69, 46.8}, N: 3, DY: 1.4, Depth: 0.9},
			{Pts: []float64{47.5, 46, 53, 45.8}, N: 2, DY: 1.4, Depth: 0.7},
			{Pts: []float64{55.5, 46, 60.5, 45.8}, N: 2, DY: 1.4, Depth: 0.7},
			{Pts: []float64{78, 42.2, 84, 42.7, 89.5, 43}, N: 1, Depth: 1.8}, // the line of the mouth
		},
		Details: stegoDetails,
	}
}

func stegoDetails(p *Painter) {
	// The tail's spikes, the far pair first, darker.
	p.Horn(HornSpec{X0: 7.2, Y0: 30.6, X1: 3.2, Y1: 20.4, W: 2.4, Bend: 0.5, Ramp: spikeBone, Shade: -0.3})
	p.Horn(HornSpec{X0: 6.6, Y0: 33.6, X1: 1.4, Y1: 38.8, W: 2.2, Bend: 0.4, Ramp: spikeBone, Shade: -0.3})
	p.Horn(HornSpec{X0: 12.8, Y0: 30.4, X1: 8.6, Y1: 17.8, W: 2.9, Bend: 0.6, Ramp: spikeBone, Grooves: true})
	p.Horn(HornSpec{X0: 11, Y0: 33.6, X1: 7.6, Y1: 39.8, W: 2.5, Bend: 0.4, Ramp: spikeBone})
	// Their bases dark, where they grow out of the skin.
	for _, r := range [][2]float64{{7.2, 30.6}, {6.6, 33.6}, {12.8, 30.4}, {11, 33.6}} {
		p.Tint(Circle(r[0], r[1], 2.2), 0x1e140c, 0.7, 1.4)
	}
	// The plates: the far row darker, behind the near one.
	for i, s := range stegoPlates {
		s.Face, s.Rim, s.Grooves, s.Seed = plateFace, plateRim, 6, uint32(i+1)
		s.Vein, s.VeinW = plateVein, 0.38
		if i < stegoFar {
			s.Shade = -0.2
		}
		p.Plate(s)
	}
	p.Eye(EyeSpec{X: 83.6, Y: 39.5, R: 0.8, Iris: stegoIris, Lid: 0.3, Squint: 1.3})
	p.Nostril(89.4, 40.2, 0.55, 0.38, -0.3)
	// The nails of the near feet, then the far ones, darker.
	for _, n := range [][3]float64{{38.9, 55.7, 0.95}, {41.6, 55.9, 1}, {44.2, 55.6, 0.85}, {64, 55.7, 0.95}, {66.7, 55.9, 1}, {69.2, 55.6, 0.85}} {
		p.Nail(n[0], n[1], 2.3*n[2], 1.6*n[2], stegoHoof, -0.15)
	}
	for _, n := range [][2]float64{{49.6, 53.6}, {52.2, 53.5}, {57.6, 53.6}, {60, 53.5}} {
		p.Nail(n[0], n[1], 1.9, 1.3, stegoHoof, -0.45)
	}
}

// stegoPhase is where (x, y) falls among the bands across the stegosaur:
// their phase, a band every whole number, and how deep below the back it
// is (0 on the back, 1 down at the belly). The bands lean to stand square
// to the arch of the back, fanning from the hips, and wander.
func stegoPhase(x, y float64) (ph, depth float64) {
	back := 17 + 0.012*(x-45)*(x-45) // about where the back is, under the plates
	depth = clamp01((y - back) / 22)
	lean := (x - 42) * 0.2 * clamp01((y-back)/12)
	return (x + lean + 4.6*(fbm(x*0.08, y*0.12, 21, 3)-0.5)) / 5.4, depth
}

// stegoStripes is the dark bands: thick over the back, thinning down the
// flanks, broken here and there, forking.
func stegoStripes(x, y float64) float64 {
	ph, depth := stegoPhase(x, y)
	f := math.Abs(ph - math.Floor(ph) - 0.5)
	half := 0.3 * (1 - 0.65*depth) * (0.7 + 0.6*fbm(x*0.3, y*0.2, 31, 2))
	if fbm(x*0.22, y*0.3, 23, 2) < 0.36 {
		half *= 0.25 // a break
	}
	d := (f - half) * 5.4
	if fbm(x*0.15, y*0.15, 27, 2) > 0.57 { // a fork beside it
		g := ph + 0.36
		d = min(d, (math.Abs(g-math.Floor(g)-0.5)-0.08*(1-depth))*5.4)
	}
	return d
}

// stegoOchre is the streaks of ochre between the bands, high on the flanks.
func stegoOchre(x, y float64) float64 {
	ph, depth := stegoPhase(x, y)
	half := 0.1 * (1 - 1.5*depth)
	if half <= 0 || fbm(x*0.25, y*0.25, 29, 2) < 0.42 {
		return 1
	}
	g := ph + 0.5
	return (math.Abs(g-math.Floor(g)-0.5) - half) * 5.4
}
