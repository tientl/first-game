package dinopaint

import (
	"image/color"
	"math"
)

// Troodon: dinos/troodon.png, 53 x 35, facing right: the night hunter,
// charcoal flecked with pale, a pale throat, a barred fan on its stiff
// tail and a huge amber eye. Its rig is "troodon" in
// internal/client/dinorigs.go: hips (28, 19) and (20, 19), knees (29.5, 24)
// and (20, 23.5), ankles (25.5, 28) and (15.5, 27.5); the legs are cut from
// row 22, columns 23-32 (near) and 13-22 (far); the head turns about
// (34.5, 10.5), the jaw opens about (40.5, 9) along the line to
// (48.5, 9.4), and the tail turns about (17, 14.5).
//
// The game finds the eye by its two ambers, which nothing else may use, and
// draws it again in every pose (internal/client/troodon.go: eyeOf,
// redrawEye): the painting marks it with them.

func init() { Register(troodon()) }

// The eye's two ambers, as in the drawing.
var troodonAmber = [2]color.NRGBA{rgb(0xffb71e), rgb(0xffdd62)}

func troodon() *Species {
	farLeg := Union(
		Cone(20, 19.6, 2.7, 20, 23.5, 2.0),
		Cone(20, 23.5, 1.8, 15.8, 27.5, 1.1),
		Cone(15.8, 27.5, 1.0, 16.3, 32.9, 0.85),
		Cone(16.3, 33.3, 0.85, 19.9, 33.6, 0.55),
		Cone(17, 32.6, 0.6, 18.4, 31.6, 0.45),
	)
	nearLeg := Union(
		Cone(28, 19.6, 2.9, 29.3, 23.8, 2.1),
		Cone(29.3, 23.8, 1.9, 25.8, 28, 1.1),
		Cone(25.8, 28, 1.0, 26.4, 32.9, 0.85),
		Cone(26.4, 33.3, 0.85, 29.9, 33.6, 0.55),
		Cone(27, 32.6, 0.6, 28.4, 31.6, 0.45),
	)
	body := Blend(2,
		Path([3]float64{19, 13.6, 3.2}, [3]float64{10, 12.6, 2.0}, [3]float64{0.5, 11.8, 2.0}), // the stiff tail
		Ellipse(25, 15.4, 7.5, 4.6, 0.05),
		Cone(30, 13, 3.2, 35, 6.5, 2.4), // the neck
		Poly(32, 2, 34, 1.6, 39, 2.4, 45, 4.6, 49, 5.5, 49.5, 7.5, 47, 8.6, 39, 9, 36, 10, 33, 10, 31, 6), // the head
	)
	jaw := Poly(38, 8.9, 48.5, 9.0, 47.5, 9.9, 39, 10.1)
	arm := Union(
		Cone(31, 16, 1.4, 32.2, 18.4, 1.0),
		Cone(32.2, 18.6, 0.8, 32.8, 19.6, 0.55),
	)
	return &Species{
		Name:        "troodon",
		GuideDetail: 0.3,
		Ink:         22,
		Materials: []Material{
			{Name: "char", Seeds: []uint32{0x383335, 0x4a4343, 0x5f5653, 0x282426, 0x1b1819, 0x7a6f68},
				Ramp: []uint32{0x0e0c0e, 0x171416, 0x211d1e, 0x2d2827, 0x3b3532, 0x4b443f, 0x5d554e, 0x71685e}},
			{Name: "fleck", Seeds: []uint32{0x9a8c76, 0x8c7f6c},
				Ramp: []uint32{0x2e2824, 0x423a32, 0x574d42, 0x6e6252, 0x857762, 0x9a8b72, 0xae9f84, 0xc0b296}, Bias: 0.12, Smooth: 0.5},
			{Name: "pale", Seeds: []uint32{0xb0a48c, 0xd0c6ac},
				Ramp: []uint32{0x3a342c, 0x564e42, 0x726858, 0x8e836e, 0xa89c84, 0xbeb398, 0xd2c8ae, 0xe4dcc4}, Bias: 0.13},
			{Name: "olive", Seeds: []uint32{0x716650, 0x524a3c, 0x35302a, 0x8f8366},
				Ramp: []uint32{0x1e1a14, 0x2c261c, 0x3c3426, 0x4e4430, 0x62563a, 0x766a46, 0x8a7d54, 0x9e9064}},
			{Name: "white", Seeds: []uint32{0xe2dcc6, 0xf0ead8},
				Ramp: []uint32{0x4e4440, 0x6e6458, 0x948a7a, 0xb4aa96, 0xcec6b0, 0xe2dcc8, 0xf0ecdc, 0xfffcf0}},
			{Name: "eye", Seeds: []uint32{0xffb71e, 0xffdd62},
				Ramp: []uint32{0x5a2a04, 0x8e4a08, 0xc87a12, 0xeea226, 0xf2c64c, 0xf6dc86, 0xf8e6a4, 0xfaf0c4}},
			// The black cap of the head, as the drawing has it.
			{Name: "cap", Ramp: []uint32{0x09080a, 0x0f0d0f, 0x161314, 0x1e1a1a, 0x272221, 0x312b29, 0x3c3532, 0x48403c},
				Region: Poly(30.4, 1.6, 36.6, 1.6, 38.6, 2.6, 37.4, 3.9, 36.6, 7.4, 34.6, 9.4, 31.4, 9.4, 30.6, 6)},
			{Name: "gum", Seeds: []uint32{0x2a1012},
				Ramp: []uint32{0x14070a, 0x1e0a0d, 0x2a1012, 0x381618, 0x461c1e, 0x542226, 0x62282c, 0x703034}},
		},
		Recolor: []Recolor{
			{Region: Poly(32.2, 19.4, 35, 19.4, 35, 22, 32.2, 22), From: []string{"white"}, To: "olive"},
			{Region: Poly(18.4, 29.4, 22, 29.4, 22, 34, 18.4, 34), From: []string{"white"}, To: "olive"},
			{Region: Poly(28.4, 29.4, 32, 29.4, 32, 34, 28.4, 34), From: []string{"white"}, To: "olive"},
			// The eye is painted again, round.
			{Region: Cut(Poly(30, 0, 50, 0, 50, 12, 30, 12), Circle(38.55, 5.45, 1.7)), From: []string{"eye"}, To: "char"},
			// The snout's lighter grey is no fleck.
			{Region: Poly(36, 3, 50, 3, 50, 9, 36, 9), From: []string{"fleck"}, To: "char"},
		},
		Remove: []SDF{
			Poly(32.4, 19.5, 34.6, 19.5, 34.6, 21.6, 32.4, 21.6), // the guide's claws, painted again
			Poly(18.6, 29.5, 22, 29.5, 22, 32.4, 18.6, 32.4),
			Poly(28.6, 29.5, 32, 29.5, 32, 32.4, 28.6, 32.4),
		},
		Parts: []Part{
			{Name: "farLeg", Shape: farLeg, Far: true, Round: 2.2, Scale: 0.65, Muscles: []Muscle{
				Mu(20, 20.5, 2.2, 2.4, 0, 0.8),
			}},
			{Name: "body", Shape: body, Round: 5.5, RoundAt: func(x, y float64) float64 { return lerp(5.5, 2.4, smoothstep(32, 37, x)) }, Muscles: []Muscle{
				Mu(24.5, 14.5, 6, 3.5, 0, 0.7),     // the ribs
				Mu(30.5, 13.5, 2.4, 2.8, 0.3, 0.6), // the shoulder
				Mu(35.6, 4.6, 2.2, 1.4, 0, 0.6),    // behind the eye
				Mu(44, 6.2, 3.4, 1.2, 0, 0.5),      // the snout
				Mu(38.4, 3.6, 1.6, 0.7, 0, 0.6),    // the brow over the eye
			}},
			{Name: "jaw", Shape: jaw, Round: 0.8, Soft: 0.5, Scale: 0.55, Mat: "char"},
			{Name: "arm", Shape: arm, Round: 1.2, Soft: 0.5, Scale: 0.6, Mat: "olive", Muscles: []Muscle{
				Mu(31.3, 16.4, 1, 1.4, 0.4, 0.5),
			}},
			{Name: "nearLeg", Shape: nearLeg, Round: 2.4, Scale: 0.65, Onto: "body", Seam: Below(14, 20, 34, 19.5), Muscles: []Muscle{
				Mu(28.2, 20.5, 2.4, 2.6, 0, 0.9),
			}},
		},
		Skin: Skin{Flank: 1.2, Back: 1.5, Belly: 1.0, BackDepth: 2.5, BellyDepth: 2.5, Relief: 0.8, Seed: 41},
		Folds: []Fold{
			{Pts: []float64{14.9, 29, 16.6, 29}, N: 3, DY: 1.0, Depth: 0.5},
			{Pts: []float64{24.9, 29, 26.6, 29}, N: 3, DY: 1.0, Depth: 0.5},
		},
		Details: troodonDetails,
	}
}

func troodonPlumage() *Plumage {
	return &Plumage{Seed: 11, Edge: 0.4, Tone: 0.7, Tracts: []Tract{
		{Name: "farThigh", Region: Poly(15, 17, 23, 17, 23, 26.4, 15, 26.4), Parts: []string{"farLeg"},
			Flow: Along(20, 19, 20, 23.5, 15.5, 27.5), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "body", Region: Union(Poly(12, 9, 31, 9, 35.5, 9.5, 35.5, 16, 31, 21, 15, 19), Ellipse(32.8, 6, 2.2, 3.6, 0.3)),
			Parts: []string{"body"},
			Flow:  Along(34, 2.6, 33, 7, 30, 11, 24, 12, 18, 12.6, 12, 12.4), Len: 2.2, Wid: 1.5, Rows: 0.45, Jitter: 0.2, Fluff: 0.5, Snap: true,
			TipMat: "fleck", TipLen: 0.3, TipGuide: true, TipElse: "char"},
		{Name: "crown", Region: Poly(31, 1, 37.4, 1, 37.4, 3.2, 35, 4.6, 33.4, 9.6, 30.5, 9.6), Parts: []string{"body"},
			Flow: Along(38, 2.3, 35, 2.6, 32.6, 6, 31.6, 9.5), Len: 1.2, Wid: 0.9, Fluff: 0.35, Snap: true},
		// The stiff tail: its feathers lie back along it, longest at the fan.
		{Name: "tail", Region: Poly(-1, 8.5, 13, 9.4, 13.5, 12.8, 13, 15.5, -1, 15.5), Parts: []string{"body"},
			Flow: Along(16, 12.9, 8, 12.3, 0, 11.8), Len: 2.6, Wid: 1.1, Rows: 0.36, Tip: 0.5, Shaft: 0.1, Fluff: 0.6, Grow: 0.04},
		{Name: "nearThigh", Region: Poly(23, 17, 33, 17, 33, 26.4, 23, 26.4), Parts: []string{"nearLeg"},
			Flow: Along(28, 19, 29.5, 24, 25.5, 28), Len: 1.9, Wid: 1.35, Fluff: 0.4, Snap: true},
		{Name: "arm", Region: Poly(30, 14.4, 33.4, 15.6, 33.6, 19.8, 31.6, 19.6, 30.2, 16.8), Parts: []string{"arm"},
			Flow: Along(30.6, 15.2, 31.8, 17.6, 32.7, 19.6), Len: 1.4, Wid: 1.0, Fluff: 0.3, Snap: true},
	}}
}

func troodonDetails(p *Painter) {
	p.Plumage(troodonPlumage())
	p.Teeth(TeethSpec{X0: 41.4, Y0: 8.2, X1: 48.2, Y1: 8.3, Bottom: 9.02, N: 6, Len: 0.72,
		Ramp: raptorTeeth, Gap: 0x2a1012, Seed: 17})
	p.Lip([]float64{41, 8.2, 44.5, 8.2, 48.4, 8.3}, 8)
	p.Nostril(47.7, 5.8, 0.45, 0.28, -0.3)
	troodonEye(p, 38.55, 5.45)
	p.Claw(32.7, 19.4, 33.4, 21.0, 0.7, 0.3, sickleHorn[:6])
	// The feet: a claw on the front toe and the sickle claw held up.
	p.Claw(19.6, 33.3, 21.4, 33.9, 0.65, 0.15, sickleHorn[:5])
	p.Claw(18.0, 32.1, 20.5, 30.4, 1.05, -0.55, sickleHorn)
	p.Claw(29.6, 33.3, 31.4, 33.9, 0.7, 0.15, sickleHorn[:6])
	p.Claw(28.0, 32.1, 30.5, 30.4, 1.1, -0.55, sickleHorn)
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
