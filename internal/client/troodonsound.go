package client

import (
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The voices of the night hunters (sim/nightstalker.go). Out in the dark
// they call to one another with a quavering, rising hoot, like an owl that
// is not quite an owl; the one that spots someone gives a thin shriek and
// the pack comes, clicking in their throats. A bite is a quick snap.

// genTroodonSFX adds the Troodon's voices to the effect bank, with a random
// source of its own.
func genTroodonSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0x7200d, 0x9))
	// The call: two quavering notes, the second higher, with an echo off the
	// trees, so it carries through the dark.
	for _, k := range []float64{1, 0.9, 1.12} {
		x := make([]float64, samples(1.0))
		for i, n := range []struct{ f0, f1, f2, at float64 }{{520, 640, 560, 0}, {640, 860, 700, 0.3}} {
			addAt(x, beastVoice(r, voiceSpec{
				dur:   0.26 + 0.06*float64(i),
				pitch: glide(pt{0, n.f0 * k}, pt{0.5, n.f1 * k}, pt{1, n.f2 * k}),
				amp:   swell(0.2, 0.45), rough: 0.55, rattle: 34, formants: [2]float64{900 * k, 2100 * k},
				breath: 0.3, harmonics: 6,
			}), n.at, 1-0.15*float64(i))
		}
		add("troodon_call", 0.8, echo(x, pt{0.14, 0.3}, pt{0.3, 0.15}))
	}
	// The shriek of the one that has seen you.
	for _, k := range []float64{1, 1.08} {
		x := beastVoice(r, voiceSpec{
			dur: 0.42, pitch: glide(pt{0, 900 * k}, pt{0.25, 1650 * k}, pt{1, 1050 * k}),
			amp: swell(0.08, 0.5), rough: 0.7, rattle: 70, formants: [2]float64{1700 * k, 3200 * k},
			breath: 0.5, harmonics: 8,
		})
		add("troodon_shriek", 0.85, echo(saturate(x, 1.8), pt{0.1, 0.25}, pt{0.22, 0.12}))
	}
	// Clicking in the throat as the pack closes in.
	for range 2 {
		x := make([]float64, samples(0.8))
		for i := range 7 {
			at := 0.04 + 0.1*float64(i) + r.Float64()*0.02
			addAt(x, noiseBurst(r, 0.025, bandpass, 2600+r.Float64()*900, 3.5, 0.0005, 0.005), at, 0.9-0.06*float64(i))
		}
		add("troodon_click", 0.7, x)
	}
}

// troodonSounds plays what the night hunters did since old.
func (g *Game) troodonSounds(old, cur protocol.Snapshot) {
	was := make(map[sim.CreatureID]sim.CreatureState, len(old.Creatures))
	for _, c := range old.Creatures {
		if c.Kind == sim.CreatureTroodon {
			was[c.ID] = c.State
		}
	}
	for _, c := range cur.Creatures {
		o, ok := was[c.ID]
		if c.Kind != sim.CreatureTroodon || !ok || o == c.State {
			continue
		}
		x, y := float64(c.X), float64(c.Y)
		switch c.State {
		case sim.CreatureChase:
			if o != sim.CreatureFlee && snd.ready("troodon_shriek", 2500*time.Millisecond) {
				callAt("troodon_alarm", x, y, 0.7, 1, 20) // it has seen someone
			}
		case sim.CreatureStrike:
			if c.Skill != sim.SkillNone {
				break // its moves are heard by skillsound.go
			}
			snd.playAt("chomp", x, y, 0.55, 0.12)
			snd.playAt("troodon_attack", x, y, 0.5, 0.08) // a quick rising shriek
		}
	}
}
