package dinopaint

import "math"

// BeakSpec is a horny beak: the line of its ridge (the culmen) and the
// line of its mouth, each from the base of the beak to its tip, in guide
// pixels. The beak is the pixels of material Mat; Gloss is how bright the
// light along the ridge is.
type BeakSpec struct {
	Mat         string
	Ridge       []float64
	Mouth       []float64
	Gloss       float64 // 0: 0.22
	Nostril     [2]float64
	NoseR       float64 // the nostril's length (0: none)
	TipDark     float64 // how much darker the tip is (0: 0.12)
	GrainSeed   uint32
	LowerShadow float64 // how much darker the lower beak is (0: 0.08)
}

// Beak paints keratin over the lit beak: a gloss along the ridge, fine
// grain running to the tip, the mouth's line dark with the lower beak's
// lit edge under it, a darker tip, and the nostril.
func (p *Painter) Beak(b BeakSpec) {
	m := p.MatID(b.Mat)
	gloss := b.Gloss
	if gloss == 0 {
		gloss = 0.22
	}
	tipDark := b.TipDark
	if tipDark == 0 {
		tipDark = 0.12
	}
	lower := b.LowerShadow
	if lower == 0 {
		lower = 0.08
	}
	ridge, mouth := newCurve(b.Ridge), newCurve(b.Mouth)
	total := ridge.run[len(ridge.run)-1]
	for py := 0; py < p.H; py++ {
		for px := 0; px < p.W; px++ {
			i := py*p.W + px
			if !p.In[i] || p.Mat[i] != m || p.Pro[i] {
				continue
			}
			edge := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if !p.Inside(px+d[0], py+d[1]) {
					edge = true
				}
			}
			x, y := G(px, py)
			s, dr := ridge.at(x, y) // dr: + below the ridge, for a beak pointing right
			u := clamp01(s / total)
			_, dm := mouth.at(x, y) // + below the mouth line
			if ridge.x[len(ridge.x)-1] < ridge.x[0] {
				dr, dm = -dr, -dm // a beak pointing left
			}
			v := p.Val[i]
			// The gloss: a streak of light just under the ridge, fading to the tip.
			if dr > 0.15 && dr < 0.75 && dm < -0.3 {
				v += gloss * (1 - 0.6*u) * (1 - math.Abs(dr-0.45)/0.3)
			}
			// The grain of the horn, running along it.
			v += 0.05 * (fbm(u*14, dr*2.2, b.GrainSeed+3, 2) - 0.5)
			v -= tipDark * smoothstep(0.78, 1, u)
			switch {
			case math.Abs(dm) < 0.2:
				v -= 0.5 // the line of the mouth
			case dm > 0 && dm < 0.55:
				v += 0.08 // the lower beak's lit edge
			case dm > 0:
				v -= lower
			}
			if edge {
				continue // the outline stays
			}
			p.Val[i] = v
			p.Out[i] = p.Tone(m, v)
			p.Pro[i] = true
		}
	}
	if b.NoseR > 0 {
		p.Nostril(b.Nostril[0], b.Nostril[1], b.NoseR, b.NoseR*0.45, -0.15)
	}
}
