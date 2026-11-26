package dinopaint

import (
	"image/color"
	"math"
)

// Troodon: dinos/troodon.png, 64 x 35, facing right: the night hunter,
// lean and low, leaning forward to hunt, on long slender legs with long
// feet. Olive-black above, the dark of the back a saddle that breaks into
// ragged bars down acid green flanks; a green throat and belly; a sleek
// short coat finely barred darker; the forearms folded with a fringe of
// longer feathers; a long stiff tail to a dark point; a narrow head with a
// long snout and a grin of small serrated teeth; and a huge amber eye. Its
// rig is "troodon" in internal/client/dinorigs.go: hips (39, 19) and
// (31, 19), knees (40.5, 24) and (31, 23.5), ankles (36.5, 28) and
// (26.5, 27.5); the legs swing whole from the hips (thigh.go); the head
// turns about (45, 11.6), the jaw opens about (50, 11) along the line to
// (60.4, 11.1), and the tail turns about (30, 13).
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
		Front:      Poly(31, 8, 46, 8, 46, 16.8, 31, 16.8),
		Thigh:      []SDF{Ellipse(38.6, 19.8, 3.7, 4.3, 0.35), Ellipse(29.1, 19.8, 3.7, 4.3, 0.35)},
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
		Ellipse(29.1, 19.8, 3.4, 4.0, 0.35),   // the thigh
		Cone(31, 23.5, 2.0, 26.5, 27.5, 1.45), // the shank
		Cone(26.5, 27.5, 1.4, 28.1, 33.4, 1.25),
		Cone(28.1, 33.6, 1.2, 33.1, 34, 0.75), // the toes
		Cone(28.9, 32.8, 0.8, 30.5, 31.2, 0.55),
	)
	nearLeg := Union(
		Ellipse(38.6, 19.8, 3.4, 4.0, 0.35),
		Cone(40.5, 24, 2.0, 36.5, 28, 1.45),
		Cone(36.5, 28, 1.4, 37.6, 33.4, 1.25),
		Cone(37.6, 33.6, 1.2, 42.6, 34, 0.75),
		Cone(38.4, 32.8, 0.8, 40, 31.2, 0.55),
	)
	body := Blend(1.6,
		Path([3]float64{32, 13.4, 2.7}, [3]float64{19.64, 12.4, 2.1}, [3]float64{7.14, 11.8, 0.9}, [3]float64{0.71, 11.4, 0.3}), // the long stiff tail
		Ellipse(36.2, 16, 8.2, 4.8, 0.06),
		Cone(42.4, 13.6, 2.8, 48.6, 9.8, 1.9), // the slim neck
		Poly(47.2, 8.6, 49.2, 6.6, 52, 6, 54.8, 6.9, 58, 8.6, 60.4, 9.8, 60.8, 10.7, 60, 11.1, 51, 11.4, 48.6, 11.8, 47, 10.6), // the narrow head
	)
	jaw := Poly(49.8, 10.9, 59.8, 11.0, 59.2, 11.8, 56.4, 12.2, 51.4, 12.4, 49.8, 12.0)
	arm := Union( // the forearm folded, its fringe of feathers trailing back
		Cone(42.6, 15, 1.3, 43.6, 18.2, 0.9),
		Cone(43.6, 18.2, 0.85, 45.4, 19, 0.6),
		Poly(42, 16.6, 44, 18.4, 43.6, 19.4, 41.2, 20.4, 40.2, 19.8),
	)
	dark := []uint32{0x0b0c0a, 0x111310, 0x181b16, 0x20241d, 0x2a2f25, 0x353b2e, 0x424937, 0x515a42}
	return &Species{
		Name:        "troodon",
		GuideDetail: 0.12,
		Ink:         20,
		Materials: []Material{
			{Name: "dark", Seeds: []uint32{0x1a1d17, 0x23271f, 0x2d3228, 0x393f31}, Ramp: dark},
			{Name: "green", Seeds: []uint32{0x34500f, 0x4c7416, 0x6a9a1e, 0x8cbe2c, 0xb0dc4a}, Ramp: troodonGreen, Bias: 0.04},
			{Name: "shank", Seeds: []uint32{0x2a2a20, 0x3c3c2c, 0x4e4e3a, 0x626048, 0x767458},
				Ramp: []uint32{0x1a1a14, 0x26261c, 0x323226, 0x404030, 0x504e3a, 0x605e46, 0x726e52, 0x86825e}},
			{Name: "white", Seeds: []uint32{0xd2cab0, 0xe6e0c6},
				Ramp: []uint32{0x4e4440, 0x6e6458, 0x948a7a, 0xb4aa96, 0xcec6b0, 0xe2dcc8, 0xf0ecdc, 0xfffcf0}},
			{Name: "eye", Seeds: []uint32{0xffb71e, 0xffdd62},
				Ramp: []uint32{0x5a2a04, 0x8e4a08, 0xc87a12, 0xeea226, 0xf2c64c, 0xf6dc86, 0xf8e6a4, 0xfaf0c4}},
			{Name: "gum", Seeds: []uint32{0x4a1416},
				Ramp: []uint32{0x14070a, 0x1e0a0d, 0x2a1012, 0x381618, 0x461c1e, 0x542226, 0x62282c, 0x703034}},
			{Name: "farShank", Ramp: []uint32{0x121210, 0x1a1a14, 0x22221a, 0x2c2c22, 0x38362a, 0x444232, 0x52503c, 0x605e48}}, // in shadow
			// The flank's own pattern, painted over the drawing's blocks: the
			// green runs up the flank to the saddle's ragged edge...
			{Name: "flank", Ramp: troodonGreen, Region: troodonFlank, Only: []string{"dark"}},
			// ...the saddle comes down over it, its bars hanging back and
			// down into the green, tapering...
			{Name: "saddle", Ramp: dark, Region: troodonSaddle, Only: []string{"green", "flank"}},
			// ...and the green goes yellow down the belly and the throat.
			{Name: "lime", Ramp: []uint32{0x2e3a0a, 0x465812, 0x627a18, 0x82a020, 0xa2c22c, 0xbcd83c, 0xd2ea56, 0xe4f47a},
				Region: troodonBelly, Only: []string{"green", "flank"}},
			// The tail barred green on the dark, the bars closer and smaller to
			// the point, their edges feathery: the drawing's blocks painted over.
			{Name: "tailDark", Ramp: dark, Region: Poly(-3, 7, 24.5, 7, 24.5, 17, -3, 17), Only: []string{"green", "flank"}},
			{Name: "tailBar", Ramp: troodonGreen, Region: troodonTailBars, Only: []string{"dark", "tailDark"}},
			// The point of the tail, dark.
			{Name: "tip", Ramp: []uint32{0x0b0c0a, 0x111310, 0x181b16, 0x20241d, 0x2a2f25, 0x353b2e, 0x424937, 0x515a42},
				Region: Poly(-1.79, 8, 9.82, 8, 9.82, 16, -1.79, 16), Over: true, Only: []string{"green", "dark"}},
		},
		Recolor: []Recolor{
			{Region: Poly(23.21, 24, 33.9, 24, 33.9, 36, 23.21, 36), From: []string{"shank"}, To: "farShank"},
			{Region: Poly(45, 18.4, 47.5, 18.4, 47.5, 21.5, 45, 21.5), From: []string{"white"}, To: "dark"},
			{Region: Union(Poly(29.6, 29.4, 32.5, 29.4, 32.5, 32.2, 29.6, 32.2), Poly(39.6, 29.4, 42.5, 29.4, 42.5, 32.2, 39.6, 32.2),
				Poly(42.6, 33.4, 44.2, 33.4, 44.2, 35, 42.6, 35), Poly(32.6, 33.4, 34, 33.4, 34, 35, 32.6, 35)), From: []string{"white"}, To: "shank"},
			{Region: Cut(Poly(41, 0, 64, 0, 64, 14, 41, 14), Circle(52, 8.4, 1.7)), From: []string{"eye"}, To: "dark"}, // the eye is painted again, round
		},
		// The tail tapers smooth to its fine point, however the drawing's steps.
		Add: []SDF{Path([3]float64{19.64, 12.4, 1.9}, [3]float64{7.14, 11.8, 0.8}, [3]float64{0.5, 11.4, 0.2})},
		Remove: []SDF{
			Poly(45.6, 18.6, 47.5, 18.6, 47.5, 21.5, 45.6, 21.5), // the guide's claws, painted again
			Poly(29.6, 29.4, 32.5, 29.4, 32.5, 31.6, 29.6, 31.6),
			Poly(39.6, 29.4, 42.5, 29.4, 42.5, 31.6, 39.6, 31.6),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.2, Scale: 0.6, Muscles: []Muscle{
				Mu(29.2, 19.6, 2.4, 2.8, 0.35, 0.8),
			}},
			{Name: "body", Shape: body, Round: 5, RoundAt: func(x, y float64) float64 { return lerp(5, 2.2, smoothstep(43, 48, x)) }, Muscles: []Muscle{
				Mu(36, 15, 6, 3.4, 0.06, 0.7),     // the ribs
				Mu(42, 13.4, 2.2, 2.6, 0.5, 0.5),  // the base of the neck
				Mu(51, 7.4, 2.2, 1.1, 0, 0.6),     // the brow over the great eye
				Mu(55.6, 8.6, 3.4, 1.2, 0.3, 0.4), // the snout
			}},
			{Name: "jaw", Shape: jaw, Round: 0.8, Soft: 0.5, Scale: 0.5, Mat: "green"},
			{Name: "arm", Shape: arm, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "dark", Muscles: []Muscle{
				Mu(42.8, 16, 1, 1.4, 0.4, 0.5),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.4, Scale: 0.6, Onto: "body", Seam: Below(25, 20, 45, 19.5), Muscles: []Muscle{
				Mu(38.6, 19.6, 2.6, 3, 0.35, 0.9),
			}},
		},
		Skin: Skin{Flank: 1.1, Back: 1.3, Belly: 1.0, BackDepth: 2, BellyDepth: 2, Relief: 0.8, Seed: 41},
		Folds: []Fold{
			{Pts: []float64{25.6, 29, 27.4, 29}, N: 3, DY: 1.2, Depth: 0.5},
			{Pts: []float64{35.6, 29, 37.4, 29}, N: 3, DY: 1.2, Depth: 0.5},
		},
		Details: troodonDetails,
	}
}

func troodonPlumage() *Plumage {
	return &Plumage{Seed: 11, Edge: 0.4, Tone: 0.25, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(23.21, 15, 34, 15, 34, 24.2, 23.21, 24.2), Parts: []string{"farLeg"},
			Flow: Along(31, 17, 31, 23.5, 26.5, 27.5), Len: 1.8, Wid: 1.3, Fluff: 0.3, Snap: true, SnapMat: true},
		// The sleek short coat, lying close from the nape back.
		{Name: "body", Region: Union(Poly(21.43, 8.5, 45, 9, 48.5, 9.5, 48.5, 16, 44, 21.5, 26, 21.5), Ellipse(48, 8.6, 2.4, 2.6, 0.3)),
			Parts: []string{"body"},
			Flow:  Along(50, 7.2, 47, 9.5, 42, 11.5, 35, 11.8, 29, 12.4, 21.43, 12.6), Len: 1.9, Wid: 1.3, Rows: 0.45, Jitter: 0.2, Fluff: 0.3, Snap: true, SnapMat: true,
			Hang: Poly(-1.79, 20.8, 65, 20.8, 65, 40, -1.79, 40)},
		{Name: "crown", Region: Poly(47, 5, 52.5, 5, 52.5, 7.2, 49, 9, 47, 9.6), Parts: []string{"body"},
			Flow: Along(53, 6.4, 50, 6.6, 47.8, 8.6), Len: 1.1, Wid: 0.8, Fluff: 0.25, Snap: true},
		// The long stiff tail: its feathers lie flat along it, a little
		// longer at its edges, to the dark point.
		{Name: "tail", Region: Poly(-1.79, 8.5, 30, 9, 30.5, 13, 30, 16.6, -1.79, 13.6), Parts: []string{"body"},
			Flow: Along(32, 13.3, 19.64, 12.4, 5.36, 11.7, -1.79, 11.3), Len: 2.4, Wid: 1.0, Rows: 0.38, Tip: 0.8, Shaft: 0.1, Inset: 0.05, Fluff: 0.3, Snap: true, SnapMat: true},
		{Name: "nearThigh", Region: Poly(34, 15, 44, 15, 44, 24.2, 34, 24.2), Parts: []string{"nearLeg"},
			Flow: Along(39, 17, 40.5, 24, 36.5, 28), Len: 1.8, Wid: 1.3, Fluff: 0.3, Snap: true, SnapMat: true},
		// The forearm folded, a fringe of longer feathers trailing back from it.
		{Name: "fringe", Region: Poly(39.6, 15, 46, 17.4, 45.8, 19.8, 42, 21, 39.4, 20.2), Parts: []string{"arm"},
			Flow: Along(45.4, 18.2, 43.2, 19, 40.6, 19.9, 39, 20.4), Len: 2.4, Wid: 0.9, Rows: 0.36, Tip: 0.6, Shaft: 0.08, Fluff: 0.4, Mat: "dark"},
	}}
}

func troodonDetails(p *Painter) {
	p.Plumage(troodonPlumage())
	// The grin: small serrated teeth along both jaws.
	p.Teeth(TeethSpec{X0: 50.6, Y0: 10.4, X1: 59.8, Y1: 10.5, Bottom: 11.02, N: 10, Len: 0.6,
		Ramp: raptorTeeth, Gap: 0x2a1012, Seed: 17})
	p.Teeth(TeethSpec{X0: 51.2, Y0: 11.75, X1: 59.2, Y1: 11.6, Bottom: 11.05, N: 9, Len: 0.5, Up: true,
		Ramp: raptorTeeth, Gap: 0x2a1012, Seed: 19})
	p.Lip([]float64{50.4, 10.45, 55, 10.45, 60, 10.55}, 8)
	p.Nostril(59.8, 9.9, 0.4, 0.24, -0.3)
	troodonEye(p, 52, 8.4)
	// The small clawed hand, tucked.
	p.Claw(45.4, 18.8, 46.2, 20.4, 0.65, 0.3, sickleHorn[:6])
	// The feet: a claw on the front toe and the sickle claw held up.
	p.Claw(32.5, 33.7, 33.8, 34.4, 0.6, 0.15, sickleHorn[:5])
	p.Claw(28.8, 32.4, 31.6, 30.6, 1.05, -0.6, sickleHorn)
	p.Claw(42, 33.7, 43.6, 34.4, 0.65, 0.15, sickleHorn[:6])
	p.Claw(38.4, 32.4, 41.2, 30.6, 1.1, -0.6, sickleHorn)
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

// troodonGreen is the acid green of its flanks: deeper up the flank.
var troodonGreen = []uint32{0x1c2e0a, 0x2c460e, 0x3e6012, 0x547e18, 0x6c9c1e, 0x86b628, 0xa0cc36, 0xb8de4a}

// troodonEdge is the lower edge of the dark saddle down the flank, from the
// root of the tail to the neck, and how ragged it is at x, y.
func troodonEdge(x, y float64) float64 {
	pts := [][2]float64{{22, 13.2}, {27, 14.0}, {31.5, 14.7}, {36, 14.9}, {40.5, 14.3}, {44, 13.4}, {47, 11.8}, {50, 9.8}}
	e := pts[0][1]
	for i := 0; i+1 < len(pts); i++ {
		if a, b := pts[i], pts[i+1]; x >= a[0] {
			t := clamp01((x - a[0]) / (b[0] - a[0]))
			e = a[1] + (b[1]-a[1])*smoothstep(0, 1, t)
		}
	}
	return e + 0.9*(fbm(x*1.1, y*1.1, 71, 2)-0.5)
}

// troodonBars are the saddle's bars, hanging back and down from its edge
// into the green, each tapering to a point, no two alike.
var troodonBars = func() SDF {
	var bs []SDF
	for _, b := range [][3]float64{{26.4, 2.8, 0.75}, {29.3, 4.2, 0.95}, {32.3, 3.3, 0.8}, {35.3, 4.6, 1.0}, {38.4, 3.6, 0.85}, {41.3, 2.9, 0.75}, {43.8, 2.0, 0.6}} {
		x, l, r := b[0], b[1], b[2]
		y := troodonEdge(x, 0) - 0.4
		bs = append(bs, Cone(x, y, r, x-0.5*l, y+l, 0.06))
	}
	return Union(bs...)
}()

// troodonSaddle is the dark of the back: down to its ragged edge along the
// body and the neck, and its bars.
func troodonSaddle(x, y float64) float64 {
	if x < 21 || x > 50.5 {
		return 1
	}
	d := y - troodonEdge(x, y)
	r := 0.25 * (fbm(x*2.3, y*2.3, 73, 2) - 0.5) // the bars' edges, ragged too
	return min(d, troodonBars(x, y)+r)
}

// troodonFlank is the flank below the saddle, the thighs and the throat,
// where the drawing's dark between its green blocks goes green.
func troodonFlank(x, y float64) float64 {
	if x < 22 || x > 50.5 || y > 25 {
		return 1
	}
	return -troodonSaddle(x, y)
}

// troodonBelly is the yellower green of the belly, the throat and the
// thighs' undersides, its edge soft and ragged.
func troodonBelly(x, y float64) float64 {
	n := 0.8 * (fbm(x*0.9, y*0.9, 79, 2) - 0.5)
	belly := 18.6 - y + n
	if x > 43 { // up the throat under the neck
		belly = min(belly, lerp(17.6, 12.4, clamp01((x-43)/6.5))-y+n)
	}
	thigh := 21.4 - y + n // the thighs' lower halves
	return max(min(belly, thigh), 22-x, x-51)
}

// troodonTailBars are the green bars down the sides of the tail, wrapping
// it, closer and narrower to the dark point.
var troodonTailBars = func() SDF {
	var bs []SDF
	for i, b := range [][2]float64{{31.2, 1.7}, {27.2, 1.55}, {23.6, 1.4}, {20.4, 1.25}, {17.5, 1.1}, {14.9, 0.95}, {12.6, 0.8}} {
		cx, w := b[0], b[1]
		cy := 12.1 + 0.025*cx
		bs = append(bs, Ellipse(cx, cy, 0.65*w, 1.6+0.1*float64(6-i), -0.35))
	}
	u := Union(bs...)
	return func(x, y float64) float64 { return u(x, y) + 0.3*(fbm(x*2.1, y*2.1, 83, 2)-0.5) }
}()
