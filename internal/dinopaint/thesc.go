package dinopaint

import (
	"image/color"
	"math"
)

// The little runner, drawn again as the user asked: dinos/thesc.png, 126 x
// 48, facing right, a Thescelosaurus as the paleo-illustrators show it:
// long and low, a deep belly, a slim neck and a small blue head, the tail
// longer than head to hip and held straight out, short arms under the
// chest, strong hind legs with long feet; olive with pale cream spots, the
// tail banded pale and dark, a short fuzzy coat. Its rig is "thesc" in
// internal/client/dinorigs.go: hips (73, 24) and (65, 24), knees (80, 33)
// and (70.6, 33.6), ankles (74.4, 42.6) and (64.2, 42.6), the balls of the
// feet (76.6, 46.2) and (66, 46.2), the toes' tips (84.6, 46.8) and (73.4,
// 46.8); the neck and head turn about (101, 18); the beak opens about
// (117.6, 16.4) along the line to (124.4, 16.8); the tail swings about (60,
// 17).

var (
	thescEye  = []uint32{0x120a06, 0x24140a, 0x3e2610, 0x5c3a16, 0x7e521e, 0xa06e2a}
	thescClaw = []uint32{0x1e1a10, 0x3c3626, 0x645c44, 0x928a6c, 0xbeb696, 0xe0dabe}
)

func init() {
	Register(thesc())
	// Its legs, in the rig's order, cut as painted and swung whole from the
	// hip, the fuzzy thigh and all (thigh.go).
	RegisterThighs("thesc", &Thighs{
		Legs: LegMask{{"nearLeg"}, {"farLeg"}},
		Thigh: []SDF{Blend(2, Ellipse(73, 26, 8.8, 9.8, 0), Cone(73, 25, 6.8, 80, 33, 4)),
			Blend(2, Ellipse(65, 26, 7, 8.4, 0), Cone(65, 26, 5.4, 70.6, 33.6, 3.4))},
		Coat: func(p *Painter) { p.Fuzz(thescFuzz) },
	})
}

func thesc() *Species {
	tail := Path([3]float64{0.5, 21.6, 0.5}, [3]float64{20, 19.6, 1.6}, [3]float64{40, 17.6, 3.2}, [3]float64{56, 16.4, 5.6}, [3]float64{66, 18, 7.6})
	body := Blend(4,
		tail,
		Ellipse(84, 21.4, 21, 9.6, 0),   // the deep trunk
		Ellipse(70, 21, 10, 8.6, 0),     // the hips
		Cone(98, 21, 6.4, 106, 17.2, 4), // the chest and the root of the neck
	)
	neck := Path([3]float64{102, 19.5, 5}, [3]float64{109, 15.6, 3.6}, [3]float64{114, 14.4, 3.2})
	head := Blend(1.4,
		Ellipse(115.6, 14.4, 5, 3.9, 0),
		Cone(116, 15, 3.4, 124.6, 16.6, 1.5), // the snout
	)
	jaw := Poly(113, 16.4, 124.6, 16.6, 123.8, 18, 119, 19, 114.4, 18.8)
	beak := Poly(122.4, 14.6, 125.6, 15.6, 125.6, 17.8, 122.6, 18)
	arm := Union(Cone(99, 25, 2.6, 97.2, 31.5, 1.9), Cone(97.2, 31.5, 1.8, 99.4, 37.4, 1.4), Cone(99.4, 37.4, 1.3, 101.6, 39, 0.7))
	farArm := Union(Cone(101.5, 25, 1.8, 101, 30.5, 1.3), Cone(101, 30.5, 1.2, 103, 35.6, 0.9))
	nearLeg := Union(
		Blend(2, Ellipse(73, 26, 8.4, 9.4, 0), Cone(73, 25, 6.4, 80, 33, 3.6)), // the thigh
		Cone(80, 33, 3.4, 74.4, 42.6, 2.2),                                     // the shin
		Cone(74.4, 42.6, 2.0, 76.6, 46, 1.7),                                   // the metatarsus
		Cone(76.4, 46.2, 1.6, 84.6, 46.8, 0.8),                                 // the long foot
	)
	farLeg := Union(
		Blend(2, Ellipse(65, 26, 6.6, 8, 0), Cone(65, 26, 5, 70.6, 33.6, 3)),
		Cone(70.6, 33.6, 2.8, 64.2, 42.6, 1.8),
		Cone(64.2, 42.6, 1.7, 66, 46, 1.5),
		Cone(65.8, 46.2, 1.4, 73.4, 46.8, 0.7),
	)
	return &Species{
		Name:        "thesc",
		GuideDetail: 0.35,
		Ink:         30,
		Materials: []Material{
			// Olive, khaki in the light, darker along the back.
			{Name: "olive", Seeds: []uint32{0x6e7834, 0x7a8240, 0x5a6428, 0x3e4620, 0x5e6a3c, 0xd6d0a0}, // the spots are painted as details
				Ramp: []uint32{0x141608, 0x1e220c, 0x2a3012, 0x383f18, 0x48501e, 0x5a6426, 0x6e7830, 0x868e3e}},
			// The head a striking cobalt, to slate down the neck.
			{Name: "blue", Seeds: []uint32{0x3c5aa6, 0x2e4688, 0x485a7a},
				Ramp: []uint32{0x0c1230, 0x16204a, 0x202e66, 0x2c4084, 0x3a54a2, 0x4c6cba, 0x6688ce, 0x86a4de}, Relief: 0.5},
			// Fine pale streaks on the neck.
			{Name: "streak", Seeds: []uint32{0xb8c4d8, 0xa8b4cc},
				Ramp: []uint32{0x3a4250, 0x525c6c, 0x6c7688, 0x8692a4, 0xa0acbc, 0xb8c4d2, 0xccd6e2, 0xdee6ee}, Smooth: 0.3, Bias: 0.12},
			// The tail banded, dark olive and pale grey.
			{Name: "tailDark", Seeds: []uint32{0x3c4420, 0x4a5226},
				Ramp: []uint32{0x0c0e08, 0x14180c, 0x1c2210, 0x262e16, 0x30381c, 0x3c4622, 0x4a562a, 0x5a6834}},
			{Name: "band", Seeds: []uint32{0xc6cabc},
				Ramp: []uint32{0x4a4c46, 0x66685e, 0x828478, 0x9ea092, 0xb6b8aa, 0xcacbbe, 0xdadbd0, 0xe8e8e0}, Smooth: 0.35, Bias: 0.12, Relief: 0.5},
			// The tail's bands, painted again over the drawing's barcode: each
			// wraps the round of the tail, a little bowed, soft and ragged at
			// its edges as fuzz is, no two quite as wide, closer to the tip,
			// pale cream-grey shading darker underneath; the tip dark.
			{Name: "tailBase", Ramp: []uint32{0x0c0e08, 0x14180c, 0x1c2210, 0x262e16, 0x30381c, 0x3c4622, 0x4a562a, 0x5a6834},
				Region: Poly(-2, 8, 63, 8, 63, 30, -2, 30), Only: []string{"band"}},
			{Name: "tailBand", Ramp: []uint32{0x4c4a40, 0x666356, 0x807d6c, 0x9a9682, 0xb2ad96, 0xc6c1a8, 0xd6d2bc, 0xe4e1ce},
				Region: thescTailBands(false), Only: []string{"tailDark", "tailBase", "olive"}, Relief: 0.5},
			{Name: "tailBandUnder", Ramp: []uint32{0x36352c, 0x4a483c, 0x5e5b4c, 0x726e5c, 0x86826c, 0x9a957e, 0xaca890, 0xbcb8a2},
				Region: thescTailBands(true), Only: []string{"tailBand"}, Relief: 0.5},
			{Name: "tailTip", Ramp: []uint32{0x0c0e08, 0x14180c, 0x1c2210, 0x262e16, 0x30381c, 0x3c4622, 0x4a562a, 0x5a6834},
				Region: Poly(-2, 8, 5.2, 8, 5.2, 30, -2, 30)},
			// The legs, dark olive-brown, the lower legs and feet scaly.
			{Name: "leg", Seeds: []uint32{0x3e3e1e, 0x2e2e18, 0x525a2c, 0x3a4020},
				Ramp: []uint32{0x0e0c06, 0x18160c, 0x242210, 0x302e16, 0x3e3c1e, 0x4e4c26, 0x605e30, 0x74723c}},
			// The beak's tip, reddish.
			{Name: "beak", Ramp: []uint32{0x1c0806, 0x2c0e0a, 0x42160e, 0x5a2014, 0x722a1a, 0x8a3822, 0xa2482c, 0xb85a38}, Relief: 0.3,
				Region: beak},
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 3, Soft: 0.6, Scale: 0.8, Mat: "leg", Muscles: []Muscle{
				Mu(65.5, 27, 5, 6, 0.2, 1.4), // the thigh
				Mu(68, 37, 1.6, 3, 0.6, 0.6), // the calf
			}},
			{Name: "farArm", Shape: farArm, Far: true, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "leg"},
			{Name: "body", Shape: body, Round: 6, Soft: 0.8, Muscles: []Muscle{
				Mu(70, 18, 9, 6, 0, 1.4),      // the hips
				Mu(86, 21, 13, 7.5, 0, 1.2),   // the ribs and the deep belly
				Mu(97, 21, 5, 5, 0.3, 1.1),    // the shoulder
				Mu(84, 29.5, 13, 2.4, 0, 0.6), // the belly
				Mu(50, 17.4, 12, 3, 0, 0.9),   // the tail's great muscle
				Mu(25, 19.4, 14, 1.6, 0, 0.5), // the tail
				Mu(80, 13.5, 17, 2.2, 0, 0.6), // the long muscles of the back
			}},
			{Name: "neck", Shape: neck, Round: 3, Soft: 0.7, Scale: 0.8, Onto: "body", Seam: Below(0, 999, 200, 999), Muscles: []Muscle{
				Mu(107, 16.6, 4, 1.8, -0.45, 0.8),
			}},
			{Name: "head", Shape: head, Round: 2.4, Soft: 0.6, Scale: 0.6, Onto: "neck", Seam: Below(100, 999, 200, 999), Muscles: []Muscle{
				Mu(115, 13, 3, 1.8, 0, 0.9),       // the cheek and the skull
				Mu(118.4, 12.2, 1.4, 0.7, 0, 0.8), // the brow
				Mu(121.6, 15, 2.6, 1.2, 0.2, 0.6), // the snout
			}},
			{Name: "jaw", Shape: jaw, Round: 1, Soft: 0.45, Scale: 0.55, Mat: "blue", Muscles: []Muscle{
				Mu(117, 17.8, 3, 0.8, 0, 0.4),
			}},
			{Name: "arm", Shape: arm, Round: 1.4, Soft: 0.5, Scale: 0.6, Mat: "olive", Onto: "body", Seam: Below(90, 28, 110, 28), Muscles: []Muscle{
				Mu(98.4, 27.6, 1.8, 2.6, 0.2, 0.6),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 4.4, Soft: 0.7, Onto: "body", Seam: Below(60, 27, 90, 27), Muscles: []Muscle{
				Mu(73, 25, 6.6, 7.6, 0.1, 2.0),                                    // the great thigh
				Mu(78.6, 30.4, 2.4, 3.6, -0.5, 0.8),                               // its front
				Mu(77.4, 37.6, 1.8, 3.2, 0.6, 0.7),                                // the calf
				Mu(79, 46.4, 1.2, 0.5, 0, 0.3), Mu(81.6, 46.6, 1.0, 0.4, 0, 0.25), // knuckles
			}},
		},
		Skin: Skin{Flank: 1.3, Back: 1.7, Belly: 1.1, BackDepth: 3, BellyDepth: 3, Relief: 0.8, Seed: 53},
		Folds: []Fold{
			{Pts: []float64{103, 14.8, 102.4, 18, 103, 21.4}, N: 3, DX: 2, DY: -0.6, Depth: 0.7}, // the neck
			{Pts: []float64{78.4, 32, 80.6, 33.6}, N: 1, Depth: 0.8},                             // behind the knee
			{Pts: []float64{73, 43.4, 76, 43.2}, N: 2, DY: 1.1, Depth: 0.6},                      // the metatarsus
			{Pts: []float64{79.6, 45.8, 79.6, 47.4}, N: 2, DX: 2.4, Depth: 0.8},                  // the toes' knuckles
			{Pts: []float64{113.6, 16.4, 118, 16.5, 124, 16.8}, N: 1, Depth: 1.4},                // the line of the mouth
		},
		Details: thescDetails,
	}
}

// thescFuzz is its coat: fine filaments over the body, the upper arms, the
// thighs and the root of the tail, lying back along it, tufted at the
// outline.
var thescFuzz = FuzzSpec{Region: Cut(Poly(-2, 0, 112, 0, 112, 35, -2, 35), Poly(104, 0, 126, 0, 126, 24, 104, 24)), Flow: 2.9, Len: 0.9, Dens: 0.5, Seed: 7}

func thescDetails(p *Painter) {
	// Small pale cream spots over the shoulders and flanks, round, where the
	// drawing has them.
	cream := []uint32{0x6a6444, 0x878060, 0xa29c78, 0xbcb690, 0xd2cca6, 0xe2ddbc, 0xeeead2}
	for gy := 0; gy < p.GH; gy++ {
		for gx := 0; gx < p.GW; gx++ {
			if g := p.Guide[gy*p.GW+gx]; g.A < 128 || g.R != 0xd6 || g.G != 0xd0 || g.B != 0xa0 {
				continue
			}
			cx, cy := float64(gx)+0.5, float64(gy)+0.5
			p.Fill(Circle(cx+0.3*(hash(gx, gy, 41)-0.5), cy, 0.68+0.22*hash(gx, gy, 43)), false, func(px, py int, d float64) color.NRGBA {
				v := p.Val[py*p.W+px] + 0.2
				return rgb(cream[max(0, min(len(cream)-1, int(v*float64(len(cream)))))])
			})
		}
	}
	// The coat: fine filaments over the body, the upper arms, the thighs
	// and the root of the tail, lying back along it, tufted at the outline.
	p.Fuzz(thescFuzz)
	p.Eye(EyeSpec{X: 118.2, Y: 13.3, R: 0.9, Iris: thescEye, Lid: 0.15, Squint: 1.15})
	// The pale ring round the eye.
	p.Fill(Cut(Ellipse(118.2, 13.3, 1.9, 1.6, 0), Ellipse(118.2, 13.3, 1.45, 1.2, 0)), false, func(px, py int, d float64) color.NRGBA {
		if i := py*p.W + px; p.Pro[i] {
			return color.NRGBA{}
		}
		return rgb(0xa8b6d0)
	})
	p.Nostril(123.4, 15.6, 0.45, 0.3, -0.3)
	// The toes' claws, the far foot's darker; the little hand's.
	p.Claw(84.2, 46.6, 85.9, 47.4, 0.85, 0.2, thescClaw)
	p.Claw(73, 46.6, 74.9, 47.4, 0.75, 0.2, thescClaw[:4])
	p.Claw(101.2, 38.8, 102.6, 40.2, 0.6, 0.2, thescClaw)
	p.Claw(100.4, 39.2, 100.9, 40.6, 0.5, 0.15, thescClaw[:4])
}

// thescTail is the tail's middle line and its half-thickness at x, as its
// shape draws it.
func thescTail(x float64) (y, r float64) {
	pts := [][3]float64{{0.5, 21.6, 0.5}, {20, 19.6, 1.6}, {40, 17.6, 3.2}, {56, 16.4, 5.6}, {66, 18, 7.6}}
	if x <= pts[0][0] {
		return pts[0][1], pts[0][2]
	}
	for i := 0; i+1 < len(pts); i++ {
		if a, b := pts[i], pts[i+1]; x <= b[0] {
			t := (x - a[0]) / (b[0] - a[0])
			return lerp(a[1], b[1], t), lerp(a[2], b[2], t)
		}
	}
	return pts[len(pts)-1][1], pts[len(pts)-1][2]
}

// thescTailBands are the pale bands round the tail (under: only their
// undersides, shaded): from its root to its tip, closer together and a
// little narrower each, each bowed with the round of the tail, ragged at its
// edges along the run of the fuzz.
func thescTailBands(under bool) SDF {
	type band struct{ x, w float64 }
	var bs []band
	x, gap := 57.0, 5.4
	for i := 0; x > 7.5; i++ {
		w := gap * (0.42 + 0.28*(hash(i, 5, 61)-0.5))
		bs = append(bs, band{x, w})
		x -= gap
		gap = max(2.3, gap*0.91)
	}
	return func(px, py float64) float64 {
		yc, r := thescTail(px)
		v := (py - yc) / max(r, 0.4)
		if math.Abs(v) > 1.25 || px > 61 {
			return 1
		}
		if under && v < 0.3 {
			return 1
		}
		d := math.Inf(1)
		for _, b := range bs {
			bx := b.x + 0.5*min(1, r/2)*(v*v-0.35) // bowed: its middle nearer the tip
			d = min(d, math.Abs(px-bx)-b.w/2)
		}
		// ragged along the hair: streaks running back down the tail
		return d + 0.7*(noise(px*0.8, py*3.4, 67)-0.5) + 0.3*(noise(px*2.8, py*2.8, 69)-0.5)
	}
}
