package client

import (
	"math"
	"math/rand/v2"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The sounds of swimming: a splash going in, a stroke of the arms through
// the water for every pull while moving, water lapping round someone
// treading water, and water running off whoever climbs out. Swimmers make
// no footsteps.

// genSwimSFX adds the swimming sounds to the effect bank, with a random
// source of its own so the sounds made before it stay as they were.
func genSwimSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0x5917, 0x3))
	// droplets scatters n small drops of water through [t0, t1) of x: each a
	// short sine rising in pitch, as a drop falling into water sounds.
	droplets := func(x []float64, n int, t0, t1, gain float64) {
		for range n {
			f := 900 + r.Float64()*1600
			addAt(x, tone(0.05, f, f*1.7, 0.03, 0.001, 0.012), t0+r.Float64()*(t1-t0), gain*(0.5+r.Float64()*0.5))
		}
	}
	// wash is water rushing, its brightness falling away over dur.
	wash := func(dur, from, to, attack, decay float64) []float64 {
		flt := newFilter(lowpass, from, 0.7)
		return render(dur, func(t float64, _ int) float64 {
			flt.set(lowpass, max(to, from-(from-to)*t/dur), 0.7)
			return flt.do(r.Float64()*2-1) * env(t, attack, decay)
		})
	}

	// A stroke: a hand cutting in, a bubble, the pull and its wash, drips.
	for _, k := range []float64{1, 0.9, 1.12} {
		x := make([]float64, samples(0.7))
		addAt(x, noiseBurst(r, 0.08, bandpass, 2300*k, 1.3, 0.002, 0.03), 0, 0.55) // the hand cutting in
		addAt(x, tone(0.08, 560*k, 230*k, 0.06, 0.002, 0.03), 0.01, 0.28)          // a bubble
		addAt(x, wash(0.5, 1300*k, 380, 0.04, 0.16), 0.02, 0.9)                    // the pull
		droplets(x, 4, 0.12, 0.45, 0.12)
		add("swim_stroke", 0.75, x)
	}
	// A kick of the legs between strokes: a soft, low churn.
	for _, k := range []float64{1, 0.85} {
		x := make([]float64, samples(0.4))
		addAt(x, wash(0.35, 700*k, 250, 0.03, 0.09), 0, 1)
		addAt(x, tone(0.06, 300*k, 160*k, 0.05, 0.003, 0.025), 0.02, 0.25)
		add("swim_kick", 0.5, x)
	}
	// Going in: a slap on the water, a thump, a great wash, falling drops.
	{
		x := make([]float64, samples(1.2))
		addAt(x, noiseBurst(r, 0.15, bandpass, 3200, 0.8, 0.002, 0.05), 0, 0.8)
		addAt(x, tone(0.18, 140, 60, 0.12, 0.003, 0.06), 0, 0.6)
		addAt(x, wash(0.9, 2600, 320, 0.01, 0.28), 0.01, 1)
		for range 3 { // bubbles welling up
			f := 300 + r.Float64()*300
			addAt(x, tone(0.09, f, f*0.55, 0.07, 0.002, 0.03), 0.08+r.Float64()*0.25, 0.25)
		}
		droplets(x, 10, 0.15, 0.9, 0.2)
		add("swim_in", 1, x)
	}
	// Climbing out: water streaming off, then drips onto the ground.
	{
		x := make([]float64, samples(1.1))
		addAt(x, wash(0.5, 1800, 500, 0.03, 0.15), 0, 0.7)
		droplets(x, 14, 0.05, 1.0, 0.22)
		add("swim_out", 0.8, x)
	}
	// Treading water: a slow slosh round the shoulders.
	for range 2 {
		x := make([]float64, samples(1.0))
		flt := newFilter(lowpass, 600, 0.7)
		addAt(x, render(0.95, func(t float64, _ int) float64 {
			return flt.do(r.Float64()*2-1) * math.Sin(math.Pi*t/0.95) * (0.6 + 0.4*math.Sin(t*7))
		}), 0, 1)
		droplets(x, 2, 0.2, 0.8, 0.12)
		add("swim_tread", 0.35, x)
	}
}

// swimCycle is how many frames, at 60 a second, one pull of the arms takes.
const swimCycle = 34

// swimSounds plays p's swimming at (x, y) this frame, and reports whether
// p is in the water (and so makes no footsteps).
func (g *Game) swimSounds(p protocol.PlayerView, x, y float64, moving bool) bool {
	was, seen := g.sm.swimming[p.ID]
	g.sm.swimming[p.ID] = p.Swimming
	play := func(name string, v float64) {
		if p.ID == g.myID {
			snd.playVar(name, v, 0, 0.07)
		} else {
			snd.playAt(name, x, y, v*0.9, 0.07)
		}
	}
	switch {
	case p.Dead:
		return p.Swimming
	case p.Swimming && seen && !was:
		play("swim_in", 0.6)
		g.sm.swimAt[p.ID] = g.frame // the first stroke comes a beat later
		return true
	case !p.Swimming && seen && was && p.Boat == 0:
		play("swim_out", 0.45)
		return false
	case !p.Swimming:
		return false
	}
	age := g.frame - g.sm.swimAt[p.ID]
	switch {
	case moving && age >= swimCycle:
		g.sm.swimAt[p.ID] = g.frame
		play("swim_stroke", 0.42)
	case moving && age == swimCycle/2:
		play("swim_kick", 0.22)
	case !moving && age >= 70:
		g.sm.swimAt[p.ID] = g.frame
		play("swim_tread", 0.25)
	}
	return true
}

// swimState is what the sounds remember of swimmers.
type swimState struct {
	swimming map[sim.PlayerID]bool // whether each player was in the water last frame
	swimAt   map[sim.PlayerID]int  // the frame of each swimmer's last stroke or slosh
}

func (s *swimState) init() {
	if s.swimming == nil {
		s.swimming = make(map[sim.PlayerID]bool)
		s.swimAt = make(map[sim.PlayerID]int)
	}
}
