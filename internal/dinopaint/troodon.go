package dinopaint

import (
	"image/color"
	"math"
)

// Troodon: dinos/troodon.png, 53 x 35, facing right: the night hunter,
// lean and low, leaning forward to hunt, on long slender legs with long
// feet. Olive-black above, the dark of the back a saddle that breaks into
// ragged bars down acid green flanks; a green throat and belly; a sleek
// short coat finely barred darker; the forearms folded with a fringe of
// longer feathers; a long stiff tail to a dark point; a narrow head with a
// long snout and a grin of small serrated teeth; and a huge amber eye. Its
// rig is "troodon" in internal/client/dinorigs.go: hips (28, 19) and
// (20, 19), knees (29.5, 24) and (20, 23.5), ankles (25.5, 28) and
// (15.5, 27.5); the legs are cut from row 22, columns 23-32 (near) and
// 13-22 (far); the head turns about (34, 11.6), the jaw opens about
// (39, 11) along the line to (49.4, 11.1), and the tail turns about (19, 13).
//
// The game finds the eye by its two ambers, which nothing else may use, and
// draws it again in every pose (internal/client/troodon.go: eyeOf,
// redrawEye): the painting marks it with them.

func init() {
	Register(troodon())
	// The legs swing whole from the hip, the near thigh under the flank and
	// the folded arm (thigh.go).
	RegisterThighs("troodon", &Thighs{
		Legs:       LegMask{{"nearLeg"}, {"farLeg"}},
		FrontParts: []string{"arm"},
		Front:      Poly(20, 8, 35, 8, 35, 16.8, 20, 16.8),
		Thigh:      []SDF{Ellipse(27.6, 19.8, 3.7, 4.3, 0.35), Ellipse(18.1, 19.8, 3.7, 4.3, 0.35)},
		Coat:       Plumed(troodonPlumage),
	})
}

// The eye's two ambers, as in the drawing.
var troodonAmber = [2]color.NRGBA{rgb(0xffb71e), rgb(0xffdd62)}

// stripes are thin bars Gap apart across the box x0-x1, y0-y1, each W wide,
// leaning lean pixels across for each pixel down.
func stripes(x0, x1, y0, y1, gap, w, lean float64) SDF {
	return func(x, y float64) float64 {
		u := (x + lean*y) / gap
		d := math.Abs(u-math.Floor(u)-0.5)*gap - w/2
		return max(d, x0-x, x-x1, y0-y, y-y1)
	}
}

func troodon() *Species {
	farLeg := Union(
		Ellipse(18.1, 19.8, 3.4, 4.0, 0.35),   // the thigh
		Cone(20, 23.5, 2.0, 15.5, 27.5, 1.45), // the shank
		Cone(15.5, 27.5, 1.4, 17.1, 33.4, 1.25),
		Cone(17.1, 33.6, 1.2, 22.1, 34, 0.75), // the toes
		Cone(17.9, 32.8, 0.8, 19.5, 31.2, 0.55),
	)
	nearLeg := Union(
		Ellipse(27.6, 19.8, 3.4, 4.0, 0.35),
		Cone(29.5, 24, 2.0, 25.5, 28, 1.45),
		Cone(25.5, 28, 1.4, 26.6, 33.4, 1.25),
		Cone(26.6, 33.6, 1.2, 31.6, 34, 0.75),
		Cone(27.4, 32.8, 0.8, 29, 31.2, 0.55),
	)
	body := Blend(1.6,
		Path([3]float64{21, 13.4, 2.7}, [3]float64{11, 12.4, 2.1}, [3]float64{4, 11.8, 0.9}, [3]float64{0.4, 11.4, 0.3}), // the long stiff tail
		Ellipse(25.2, 16, 8.2, 4.8, 0.06),
		Cone(31.4, 13.6, 2.8, 37.6, 9.8, 1.9), // the slim neck
		Poly(36.2, 8.6, 38.2, 6.6, 41, 6, 43.8, 6.9, 47, 8.6, 49.4, 9.8, 49.8, 10.7, 49, 11.1, 40, 11.4, 37.6, 11.8, 36, 10.6), // the narrow head
	)
	jaw := Poly(38.8, 10.9, 48.8, 11.0, 48.2, 11.8, 45.4, 12.2, 40.4, 12.4, 38.8, 12.0)
	arm := Union( // the forearm folded, its fringe of feathers trailing back
		Cone(31.6, 15, 1.3, 32.6, 18.2, 0.9),
		Cone(32.6, 18.2, 0.85, 34.4, 19, 0.6),
		Poly(31, 16.6, 33, 18.4, 32.6, 19.4, 30.2, 20.4, 29.2, 19.8),
	)
	dark := []uint32{0x0b0c0a, 0x111310, 0x181b16, 0x20241d, 0x2a2f25, 0x353b2e, 0x424937, 0x515a42}
	return &Species{
		Name:        "troodon",
		GuideDetail: 0.3,
		Ink:         20,
		Materials: []Material{
			{Name: "dark", Seeds: []uint32{0x1a1d17, 0x23271f, 0x2d3228, 0x393f31}, Ramp: dark},
			{Name: "green", Seeds: []uint32{0x34500f, 0x4c7416, 0x6a9a1e, 0x8cbe2c, 0xb0dc4a},
				Ramp: []uint32{0x22340a, 0x34500e, 0x4a7014, 0x62921a, 0x7cb222, 0x98cc30, 0xb4e044, 0xccf064}, Bias: 0.04},
			{Name: "shank", Seeds: []uint32{0x2a2a20, 0x3c3c2c, 0x4e4e3a, 0x626048, 0x767458},
				Ramp: []uint32{0x1a1a14, 0x26261c, 0x323226, 0x404030, 0x504e3a, 0x605e46, 0x726e52, 0x86825e}},
			{Name: "white", Seeds: []uint32{0xd2cab0, 0xe6e0c6},
				Ramp: []uint32{0x4e4440, 0x6e6458, 0x948a7a, 0xb4aa96, 0xcec6b0, 0xe2dcc8, 0xf0ecdc, 0xfffcf0}},
			{Name: "eye", Seeds: []uint32{0xffb71e, 0xffdd62},
				Ramp: []uint32{0x5a2a04, 0x8e4a08, 0xc87a12, 0xeea226, 0xf2c64c, 0xf6dc86, 0xf8e6a4, 0xfaf0c4}},
			{Name: "gum", Seeds: []uint32{0x4a1416},
				Ramp: []uint32{0x14070a, 0x1e0a0d, 0x2a1012, 0x381618, 0x461c1e, 0x542226, 0x62282c, 0x703034}},
			{Name: "farShank", Ramp: []uint32{0x121210, 0x1a1a14, 0x22221a, 0x2c2c22, 0x38362a, 0x444232, 0x52503c, 0x605e48}}, // in shadow
			// Fine darker barring across the green of the flanks and thighs.
			{Name: "bar", Ramp: []uint32{0x1a2a0a, 0x263c0e, 0x325012, 0x406416, 0x4e781c, 0x5e8e22, 0x70a428, 0x82b830},
				Region: stripes(13, 37, 11, 24, 1.35, 0.32, 0.35), Over: true, Only: []string{"green"}},
			// The point of the tail, dark.
			{Name: "tip", Ramp: []uint32{0x0b0c0a, 0x111310, 0x181b16, 0x20241d, 0x2a2f25, 0x353b2e, 0x424937, 0x515a42},
				Region: Poly(-1, 8, 5.5, 8, 5.5, 16, -1, 16), Over: true, Only: []string{"green", "dark"}},
		},
		Recolor: []Recolor{
			{Region: Poly(13, 24, 22.9, 24, 22.9, 36, 13, 36), From: []string{"shank"}, To: "farShank"},
			{Region: Poly(34, 18.4, 36.5, 18.4, 36.5, 21.5, 34, 21.5), From: []string{"white"}, To: "dark"},
			{Region: Union(Poly(18.6, 29.4, 21.5, 29.4, 21.5, 32.2, 18.6, 32.2), Poly(28.6, 29.4, 31.5, 29.4, 31.5, 32.2, 28.6, 32.2),
				Poly(31.6, 33.4, 33.2, 33.4, 33.2, 35, 31.6, 35), Poly(21.6, 33.4, 23, 33.4, 23, 35, 21.6, 35)), From: []string{"white"}, To: "shank"},
			{Region: Cut(Poly(30, 0, 53, 0, 53, 14, 30, 14), Circle(41, 8.4, 1.7)), From: []string{"eye"}, To: "dark"}, // the eye is painted again, round
		},
		Remove: []SDF{
			Poly(34.6, 18.6, 36.5, 18.6, 36.5, 21.5, 34.6, 21.5), // the guide's claws, painted again
			Poly(18.6, 29.4, 21.5, 29.4, 21.5, 31.6, 18.6, 31.6),
			Poly(28.6, 29.4, 31.5, 29.4, 31.5, 31.6, 28.6, 31.6),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.2, Scale: 0.6, Muscles: []Muscle{
				Mu(18.2, 19.6, 2.4, 2.8, 0.35, 0.8),
			}},
			{Name: "body", Shape: body, Round: 5, RoundAt: func(x, y float64) float64 { return lerp(5, 2.2, smoothstep(32, 37, x)) }, Muscles: []Muscle{
				Mu(25, 15, 6, 3.4, 0.06, 0.7),     // the ribs
				Mu(31, 13.4, 2.2, 2.6, 0.5, 0.5),  // the base of the neck
				Mu(40, 7.4, 2.2, 1.1, 0, 0.6),     // the brow over the great eye
				Mu(44.6, 8.6, 3.4, 1.2, 0.3, 0.4), // the snout
			}},
			{Name: "jaw", Shape: jaw, Round: 0.8, Soft: 0.5, Scale: 0.5, Mat: "green"},
			{Name: "arm", Shape: arm, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "dark", Muscles: []Muscle{
				Mu(31.8, 16, 1, 1.4, 0.4, 0.5),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.4, Scale: 0.6, Onto: "body", Seam: Below(14, 20, 34, 19.5), Muscles: []Muscle{
				Mu(27.6, 19.6, 2.6, 3, 0.35, 0.9),
			}},
		},
		Skin: Skin{Flank: 1.1, Back: 1.3, Belly: 1.0, BackDepth: 2, BellyDepth: 2, Relief: 0.8, Seed: 41},
		Folds: []Fold{
			{Pts: []float64{14.6, 29, 16.4, 29}, N: 3, DY: 1.2, Depth: 0.5},
			{Pts: []float64{24.6, 29, 26.4, 29}, N: 3, DY: 1.2, Depth: 0.5},
		},
		Details: troodonDetails,
	}
}

func troodonPlumage() *Plumage {
	return &Plumage{Seed: 11, Edge: 0.4, Tone: 0.6, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(13, 15, 23, 15, 23, 24.2, 13, 24.2), Parts: []string{"farLeg"},
			Flow: Along(20, 17, 20, 23.5, 15.5, 27.5), Len: 1.8, Wid: 1.3, Fluff: 0.3, Snap: true},
		// The sleek short coat, lying close from the nape back.
		{Name: "body", Region: Union(Poly(12, 8.5, 34, 9, 37.5, 9.5, 37.5, 16, 33, 21.5, 15, 21.5), Ellipse(37, 8.6, 2.4, 2.6, 0.3)),
			Parts: []string{"body"},
			Flow:  Along(39, 7.2, 36, 9.5, 31, 11.5, 24, 11.8, 18, 12.4, 12, 12.6), Len: 1.9, Wid: 1.3, Rows: 0.45, Jitter: 0.2, Fluff: 0.3, Snap: true,
			Hang: Poly(-1, 20.8, 54, 20.8, 54, 40, -1, 40)},
		{Name: "crown", Region: Poly(36, 5, 41.5, 5, 41.5, 7.2, 38, 9, 36, 9.6), Parts: []string{"body"},
			Flow: Along(42, 6.4, 39, 6.6, 36.8, 8.6), Len: 1.1, Wid: 0.8, Fluff: 0.25, Snap: true},
		// The long stiff tail: its feathers lie flat along it, a little
		// longer at its edges, to the dark point.
		{Name: "tail", Region: Poly(-1, 8.5, 19, 9, 19.5, 13, 19, 16.6, -1, 13.6), Parts: []string{"body"},
			Flow: Along(21, 13.3, 11, 12.4, 3, 11.7, -1, 11.3), Len: 2.4, Wid: 1.0, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05, Fluff: 0.3},
		{Name: "nearThigh", Region: Poly(23, 15, 33, 15, 33, 24.2, 23, 24.2), Parts: []string{"nearLeg"},
			Flow: Along(28, 17, 29.5, 24, 25.5, 28), Len: 1.8, Wid: 1.3, Fluff: 0.3, Snap: true},
		// The forearm folded, a fringe of longer feathers trailing back from it.
		{Name: "fringe", Region: Poly(28.6, 15, 35, 17.4, 34.8, 19.8, 31, 21, 28.4, 20.2), Parts: []string{"arm"},
			Flow: Along(34.4, 18.2, 32.2, 19, 29.6, 19.9, 28, 20.4), Len: 2.4, Wid: 0.9, Rows: 0.36, Tip: 0.6, Shaft: 0.08, Fluff: 0.4, Mat: "dark"},
	}}
}

func troodonDetails(p *Painter) {
	p.Plumage(troodonPlumage())
	// The grin: small serrated teeth along both jaws.
	p.Teeth(TeethSpec{X0: 39.6, Y0: 10.4, X1: 48.8, Y1: 10.5, Bottom: 11.02, N: 10, Len: 0.6,
		Ramp: raptorTeeth, Gap: 0x2a1012, Seed: 17})
	p.Teeth(TeethSpec{X0: 40.2, Y0: 11.75, X1: 48.2, Y1: 11.6, Bottom: 11.05, N: 9, Len: 0.5, Up: true,
		Ramp: raptorTeeth, Gap: 0x2a1012, Seed: 19})
	p.Lip([]float64{39.4, 10.45, 44, 10.45, 49, 10.55}, 8)
	p.Nostril(48.8, 9.9, 0.4, 0.24, -0.3)
	troodonEye(p, 41, 8.4)
	// The small clawed hand, tucked.
	p.Claw(34.4, 18.8, 35.2, 20.4, 0.65, 0.3, sickleHorn[:6])
	// The feet: a claw on the front toe and the sickle claw held up.
	p.Claw(21.5, 33.7, 22.8, 34.4, 0.6, 0.15, sickleHorn[:5])
	p.Claw(17.8, 32.4, 20.6, 30.6, 1.05, -0.6, sickleHorn)
	p.Claw(31.0, 33.7, 32.6, 34.4, 0.65, 0.15, sickleHorn[:6])
	p.Claw(27.4, 32.4, 30.2, 30.6, 1.1, -0.6, sickleHorn)
}

// troodonEye paints the great eye about (cx, cy) in the drawing's two
// ambers only, lit at the top left, in a dark socket: the game finds it by
// them and draws it again over them, a little smaller.
func troodonEye(p *Painter, cx, cy float64) {
	const r = 1.3
	ink := rgb(0x17120f)
	for py := int((cy - r - 1) * K); py <= int((cy+r+1)*K); py++ {
		for px := int((cx - r - 1) * K); px <= int((cx+r+1)*K); px++ {
			i := p.Idx(px, py)
			if i < 0 || !p.In[i] {
				continue
			}
			x, y := G(px, py)
			d := math.Hypot(x-cx, y-cy)
			switch {
			case d <= r && math.Abs(x-cx-0.1) < 0.3 && math.Abs(y-cy) < 0.95:
				p.Out[i] = rgb(0x120a04) // the slit of the pupil
			case d <= r && x-cx+y-cy < -0.5:
				p.Out[i] = troodonAmber[1]
			case d <= r:
				p.Out[i] = troodonAmber[0]
			case d <= r+0.45:
				p.Out[i] = ink
			default:
				continue
			}
			p.Pro[i] = true
		}
	}
}
