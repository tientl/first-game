package client

import "math/rand/v2"

// The sounds of a shield (see sim/shield.go and feel.go).
//
//   - shield_block: a blow taken on the boards: the dull knock of a wooden
//     disc struck flat (a plate's inharmonic partials, damped fast by the
//     arm behind it), the slap of the rawhide rim, a short scrape.
//   - shield_parry: the blow turned aside just in time: a sharp crack, a
//     higher, brighter knock and a ring from the boss, and the swish of the
//     blow sliding off.
func genCombatSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(2026, 50))
	for v := 0; v < 3; v++ {
		k := 1 + (float64(v)-1)*0.07
		x := make([]float64, samples(0.42))
		addAt(x, noiseBurst(r, 0.03, highpass, 1800, 0.7, 0.0003, 0.006), 0, 0.7) // the rim's slap
		addAt(x, noiseBurst(r, 0.16, bandpass, 340*k, 2.5, 0.0005, 0.045), 0, 1)  // the boards
		addAt(x, partials(0.3, 150*k, []float64{1, 1.59, 2.14, 2.65}, []float64{0.7, 0.45, 0.25, 0.15}, []float64{0.06, 0.04, 0.03, 0.02}), 0, 0.9)
		addAt(x, tone(0.18, 95*k, 70*k, 0.05, 0.002, 0.05), 0, 0.6)                                                         // the arm taking it
		addAt(x, grains(r, 0.12, 60, 0.004, bandpass, 2600, 1.2, func(t float64) float64 { return 1 - t/0.12 }), 0.01, 0.2) // a scrape
		add("shield_block", 0.9, x)
	}
	for v := 0; v < 3; v++ {
		k := 1 + (float64(v)-1)*0.05
		x := make([]float64, samples(0.6))
		addAt(x, noiseBurst(r, 0.02, highpass, 3200, 0.8, 0.0002, 0.004), 0, 1)                                                                     // the crack
		addAt(x, noiseBurst(r, 0.1, bandpass, 620*k, 3, 0.0004, 0.03), 0, 0.8)                                                                      // the boards, struck edge on
		addAt(x, partials(0.5, 410*k, []float64{1, 2.31, 3.9, 5.2}, []float64{0.5, 0.35, 0.2, 0.1}, []float64{0.14, 0.09, 0.05, 0.03}), 0.002, 0.6) // the boss rings
		addAt(x, noiseBurst(r, 0.22, bandpass, 1500*k, 1.2, 0.02, 0.08), 0.015, 0.35)                                                               // the blow sliding off
		add("shield_parry", 0.9, x)
	}
}
