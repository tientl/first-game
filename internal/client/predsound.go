package client

import (
	"math"
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The voices of the hunters, made to frighten. The tyrant's roar gathers as
// a growl in its chest, then breaks from three throats at once: a rattling
// bellow, a deeper one under it and a scream over it, pushed past what a
// throat can hold. Raptors shriek like geese and snakes together and click
// as they close in; the sabre-tooth roars like the great cats; crocodiles
// hiss as they burst from the water. A hunter closing on the player growls
// as it comes.

// saturate drives x into a soft clip in place, as a throat pushed past what
// it can hold: louder, rougher, richer in harmonics.
func saturate(x []float64, drive float64) []float64 {
	m := 0.0
	for _, v := range x {
		m = max(m, math.Abs(v))
	}
	if m == 0 {
		return x
	}
	k := math.Tanh(drive)
	for i, v := range x {
		x[i] = math.Tanh(drive*v/m) / k
	}
	return x
}

// vowel is formants passing through the points {u, f1, f2} in order: the
// jaws opening and closing along a call.
func vowel(pts ...[3]float64) func(u float64) [2]float64 {
	var a, b [][2]float64
	for _, p := range pts {
		a = append(a, [2]float64{p[0], p[1]})
		b = append(b, [2]float64{p[0], p[2]})
	}
	f1, f2 := glide(a...), glide(b...)
	return func(u float64) [2]float64 { return [2]float64{f1(u), f2(u)} }
}

type pt = [2]float64

// genPredatorSFX adds the hunters' voices to the effect bank, with a random
// source of its own.
func genPredatorSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0xd1e5, 0x7))

	// The tyrant's roar: see rexroar.go.
	for _, k := range []float64{1, 0.92, 1.07} {
		x := tyrantRoar(r, k)
		add("roar", 1, echo(x, pt{0.09, 0.4}, pt{0.2, 0.3}, pt{0.34, 0.22}, pt{0.52, 0.15}, pt{0.75, 0.1}, pt{1.05, 0.06}, pt{1.4, 0.03}))
		add("roar_far", 0.85, muffle(x, 380))
	}
	// The tyrant's growl as it closes in: slow, deep, wet breathing.
	for _, k := range []float64{1, 0.9} {
		x := make([]float64, samples(1.8))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 1.7, pitch: glide(pt{0, 38 * k}, pt{0.4, 46 * k}, pt{1, 34 * k}), amp: swell(0.35, 0.35),
			rough: 1, rattle: 17, formants: [2]float64{170 * k, 400 * k}, breath: 0.9, sub: 0.9, harmonics: 18,
		}), 0, 1)
		addAt(x, noiseBurst(r, 1.2, lowpass, 500, 0.7, 0.4, 0.4), 0.1, 0.5)
		add("rex_growl", 1, saturate(x, 1.8))
	}

	// A raptor's shriek as it goes for something: a hoarse, honking bark with
	// a thin scream over it, dying in a hiss.
	for _, k := range []float64{1, 1.08, 0.93} {
		x := make([]float64, samples(0.85))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 0.55, pitch: glide(pt{0, 430 * k}, pt{0.2, 780 * k}, pt{0.6, 650 * k}, pt{1, 380 * k}), amp: swell(0.08, 0.45),
			rough: 0.75, rattle: 58, formants: [2]float64{1000 * k, 2200 * k},
			vowel:  vowel([3]float64{0, 700 * k, 1700 * k}, [3]float64{0.3, 1100 * k, 2400 * k}, [3]float64{1, 800 * k, 1900 * k}),
			breath: 0.6, harmonics: 12,
		}), 0, 1)
		addAt(x, beastVoice(r, voiceSpec{
			dur: 0.42, pitch: glide(pt{0, 1500 * k}, pt{0.3, 2500 * k}, pt{1, 1400 * k}), amp: swell(0.15, 0.5),
			rough: 0.4, rattle: 95, formants: [2]float64{2600 * k, 4200 * k}, breath: 0.25, harmonics: 4,
		}), 0.06, 0.4)
		addAt(x, noiseBurst(r, 0.45, highpass, 3500, 0.7, 0.05, 0.14), 0.35, 0.35)
		add("raptor_screech", 0.9, echo(saturate(x, 2), pt{0.09, 0.25}, pt{0.2, 0.12}))
	}
	// A raptor closing in: a low, purring rattle and the clicking of its throat.
	for range 2 {
		x := make([]float64, samples(1.05))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 0.95, pitch: glide(pt{0, 95}, pt{1, 80}), amp: swell(0.2, 0.3),
			rough: 1, rattle: 13, formants: [2]float64{420, 1300}, breath: 0.35, harmonics: 10,
		}), 0, 0.7)
		for i := range 9 {
			at := 0.05 + 0.095*float64(i) + r.Float64()*0.015
			addAt(x, noiseBurst(r, 0.03, bandpass, 1900+r.Float64()*600, 3, 0.0005, 0.006), at, 0.9-0.05*float64(i))
		}
		add("raptor_click", 0.8, x)
	}

	// The sabre-tooth's roar: a hiss, then a great cat's "aaouugh".
	for _, k := range []float64{1, 0.9} {
		x := make([]float64, samples(1.8))
		addAt(x, noiseBurst(r, 0.4, bandpass, 3000, 0.8, 0.02, 0.14), 0, 0.55)
		addAt(x, beastVoice(r, voiceSpec{
			dur: 1.5, pitch: glide(pt{0, 120 * k}, pt{0.25, 170 * k}, pt{1, 82 * k}), amp: swell(0.1, 0.55),
			rough: 0.85, rattle: 26, formants: [2]float64{700 * k, 1200 * k},
			vowel:  vowel([3]float64{0, 500 * k, 1000 * k}, [3]float64{0.25, 750 * k, 1250 * k}, [3]float64{1, 360 * k, 640 * k}),
			breath: 0.8, sub: 0.4, harmonics: 14,
		}), 0.18, 1)
		add("sabre_roar", 0.95, echo(saturate(x, 2.2), pt{0.12, 0.28}, pt{0.27, 0.14}))
	}
	{ // its snarl as it springs: short and vicious
		x := make([]float64, samples(0.7))
		addAt(x, noiseBurst(r, 0.25, bandpass, 3200, 0.8, 0.01, 0.08), 0, 0.6)
		addAt(x, beastVoice(r, voiceSpec{
			dur: 0.6, pitch: glide(pt{0, 150}, pt{0.2, 190}, pt{1, 115}), amp: swell(0.06, 0.5),
			rough: 0.95, rattle: 34, vowel: vowel([3]float64{0, 600, 1100}, [3]float64{0.3, 800, 1350}, [3]float64{1, 500, 950}),
			breath: 0.8, sub: 0.3, harmonics: 14,
		}), 0.05, 1)
		add("sabre_snarl", 0.95, saturate(x, 2.5))
	}
	{ // and its growl as it stalks
		x := beastVoice(r, voiceSpec{
			dur: 1.3, pitch: glide(pt{0, 68}, pt{0.5, 74}, pt{1, 60}), amp: swell(0.3, 0.3),
			rough: 0.95, rattle: 19, formants: [2]float64{260, 620}, breath: 0.6, sub: 0.6, harmonics: 14,
		})
		add("sabre_growl", 0.9, saturate(x, 1.6))
	}

	// A crocodile bursting from the water: a long hiss over a deep growl.
	{
		x := make([]float64, samples(1.05))
		addAt(x, noiseBurst(r, 1, bandpass, 2200, 0.5, 0.05, 0.35), 0, 1)
		addAt(x, beastVoice(r, voiceSpec{
			dur: 0.9, pitch: glide(pt{0, 48}, pt{1, 40}), amp: swell(0.15, 0.5),
			rough: 1, rattle: 15, formants: [2]float64{150, 380}, breath: 0.2, sub: 0.9, harmonics: 16,
		}), 0.05, 0.8)
		add("croc_hiss", 0.9, saturate(x, 1.5))
	}
}

// growlReach is how far, in tiles, a hunter's growl is heard.
const growlReach = 12.0

// predatorGrowls lets a wild hunter after something close by growl, rattle
// or click as it comes, so the player hears it closing in.
func (g *Game) predatorGrowls(cur protocol.Snapshot) {
	for _, c := range cur.Creatures {
		if c.State != sim.CreatureChase || c.Tamed {
			continue
		}
		// key is what rate-limits the growl: the two raptors share one.
		key, name, gap, vol := "", "", time.Duration(0), 0.0
		switch c.Kind {
		case sim.CreatureRex:
			if time.Since(lastRoar) < 4500*time.Millisecond {
				continue // still roaring
			}
			key, name, gap, vol = "rex_growl", "rex_growl", 3200*time.Millisecond, 0.9
		case sim.CreatureRaptor:
			key, name, gap, vol = "raptor_click", "raptor_growl", 2200*time.Millisecond, 0.55
		case sim.CreatureAchero:
			key, name, gap, vol = "raptor_click", "achero_growl", 2200*time.Millisecond, 0.45
		case sim.CreatureSabre:
			key, name, gap, vol = "sabre_growl", "sabre_growl", 2800*time.Millisecond, 0.7
		case sim.CreatureTroodon: // see troodonsound.go
			key, name, gap, vol = "troodon_click", "troodon_growl", 1800*time.Millisecond, 0.5
		default:
			continue
		}
		x, y := float64(c.X), float64(c.Y)
		if math.Hypot(x-snd.lx, (y-snd.ly)*1.3) >= growlReach || !snd.ready(key, gap) {
			continue
		}
		callAt(name, x, y, vol, 1, growlReach)
	}
}
