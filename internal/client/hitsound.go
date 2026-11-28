package client

import "math/rand/v2"

// The sounds of a blow by what it strikes (see hitfx.go): flesh thuds and
// slaps (the "whack" of dinosound.go for the big beasts), a shell crunches,
// a bone knocks dry. The clang stays for stone and metal.
//
//   - flesh_slap: a hand or a stick on a small beast: a wet slap over a short thump.
//   - shell_crunch: chitin giving: a dry crackle over a low tick.
//   - bone_knock: bone struck: a short hollow tok with a dry click.
func genHitSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(2026, 61))
	for v := 0; v < 3; v++ {
		k := 1 + (float64(v)-1)*0.08
		x := make([]float64, samples(0.22))
		addAt(x, tone(0.16, 210*k, 110*k, 0.05, 0.002, 0.04), 0, 1)
		addAt(x, noiseBurst(r, 0.08, bandpass, 900*k, 0.9, 0.001, 0.02), 0, 0.6)
		addAt(x, noiseBurst(r, 0.05, lowpass, 1800, 0.7, 0.0005, 0.012), 0, 0.4)
		add("flesh_slap", 0.9, x)
	}
	for v := 0; v < 3; v++ {
		k := 1 + (float64(v)-1)*0.06
		x := make([]float64, samples(0.3))
		addAt(x, grains(r, 0.14, 120, 0.003, bandpass, 2200*k, 1.2, func(t float64) float64 { return 1 - t/0.14 }), 0, 0.7)
		addAt(x, noiseBurst(r, 0.03, highpass, 3000, 0.7, 0.0002, 0.006), 0, 0.6)
		addAt(x, tone(0.1, 150*k, 90*k, 0.04, 0.001, 0.03), 0, 0.7)
		add("shell_crunch", 0.9, x)
	}
	for v := 0; v < 3; v++ {
		k := 1 + (float64(v)-1)*0.07
		x := make([]float64, samples(0.25))
		addAt(x, partials(0.16, 520*k, []float64{1, 1.7, 2.6}, []float64{0.7, 0.4, 0.2}, []float64{0.03, 0.02, 0.015}), 0, 0.8)
		addAt(x, noiseBurst(r, 0.02, highpass, 3200, 0.8, 0.0002, 0.005), 0, 0.7)
		addAt(x, tone(0.08, 190*k, 120*k, 0.03, 0.001, 0.025), 0, 0.5)
		add("bone_knock", 0.9, x)
	}
}
