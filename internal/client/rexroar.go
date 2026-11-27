package client

import (
	"math"
	"math/rand/v2"
	"time"
)

// The tyrant's roar, made to chill the blood. Like the roars made for
// films, it is several animals at once, pushed past what a throat can hold:
//
//   - a growl gathering deep in its chest, a wet click as the jaws part and
//     a breath dragged in;
//   - then the roar: a great cat's open-mouthed bellow, three throats a
//     hair out of tune with one another, over a deeper throat still;
//   - riding on top of it, an elephant's trumpeting scream, with a second
//     scream out of tune with the first, so it grates;
//   - under it all a rumble felt more than heard;
//   - and as it dies, a slow wet rattle and a hot snort of breath.
//
// A tyrant roaring close by shakes the ground: see roarShake.

// tyrantRoar renders one roar, pitched by k (1 is the usual voice).
func tyrantRoar(r *rand.Rand, k float64) []float64 {
	const pre = 0.9 // the growl before the jaws open
	body := 2.7 / k
	x := make([]float64, samples(pre+body+1.3))

	// The growl gathering in its chest.
	addAt(x, beastVoice(r, voiceSpec{
		dur: pre + 0.2, pitch: glide(pt{0, 28 * k}, pt{0.7, 44 * k}, pt{1, 52 * k}), amp: swell(0.75, 0.15),
		rough: 1, rattle: 15, formants: [2]float64{165 * k, 390 * k}, breath: 0.7, sub: 1, harmonics: 18,
	}), 0, 0.9)
	addAt(x, noiseBurst(r, 0.35, bandpass, 1100, 0.8, 0.28, 0.06), pre-0.4, 0.45) // the breath dragged in
	addAt(x, noiseBurst(r, 0.05, bandpass, 900, 3, 0.001, 0.012), pre-0.06, 0.6)  // the jaws coming apart
	addAt(x, noiseBurst(r, 0.05, bandpass, 650, 3, 0.001, 0.015), pre-0.02, 0.45)

	// The roar: a great cat's bellow from three throats.
	for i, d := range []float64{0.985, 1, 1.017} {
		kk := k * d
		addAt(x, beastVoice(r, voiceSpec{
			dur: body, pitch: glide(pt{0, 62 * kk}, pt{0.1, 124 * kk}, pt{0.5, 106 * kk}, pt{0.85, 70 * kk}, pt{1, 48 * kk}),
			amp: swell(0.06, 0.45), rough: 0.9, rattle: 29 + 3*float64(i), formants: [2]float64{300 * kk, 720 * kk},
			vowel: vowel([3]float64{0, 300 * kk, 720 * kk}, [3]float64{0.15, 700 * kk, 1350 * kk},
				[3]float64{0.6, 600 * kk, 1150 * kk}, [3]float64{1, 320 * kk, 680 * kk}),
			breath: 0.9, sub: 0.6, harmonics: 16,
		}), pre+0.012*float64(i), 0.55)
	}
	addAt(x, beastVoice(r, voiceSpec{ // a deeper throat under it
		dur: body, pitch: glide(pt{0, 30 * k}, pt{0.1, 54 * k}, pt{0.5, 47 * k}, pt{1, 20 * k}), amp: swell(0.1, 0.5),
		rough: 1, rattle: 21, formants: [2]float64{150 * k, 360 * k}, breath: 0.5, sub: 0.9, harmonics: 18,
	}), pre, 0.85)

	// The trumpeting scream on top, and its grating twin.
	scream := body * 0.62
	for _, v := range []struct{ ratio, gain, rattle float64 }{{1, 0.6, 52}, {1.37, 0.28, 61}} {
		kk := k * v.ratio
		addAt(x, beastVoice(r, voiceSpec{
			dur: scream, pitch: glide(pt{0, 300 * kk}, pt{0.22, 560 * kk}, pt{0.45, 520 * kk}, pt{0.7, 455 * kk}, pt{1, 240 * kk}),
			amp: swell(0.2, 0.5), rough: 0.75, rattle: v.rattle, formants: [2]float64{1100 * kk, 2600 * kk},
			vowel:  vowel([3]float64{0, 900 * kk, 2100 * kk}, [3]float64{0.3, 1300 * kk, 2900 * kk}, [3]float64{1, 850 * kk, 2000 * kk}),
			breath: 0.6, harmonics: 10,
		}), pre+0.14, v.gain)
	}
	addAt(x, noiseBurst(r, body*0.7, bandpass, 2500*k, 0.9, 0.12, body*0.3), pre+0.05, 0.25) // air through the teeth
	addAt(x, tone(body+0.5, 36*k, 22*k, body, 0.2, body*0.55), pre, 0.65)                    // the rumble

	// As it dies: a slow, wet rattle in the throat and a snort.
	end := pre + body
	addAt(x, beastVoice(r, voiceSpec{
		dur: 1.05, pitch: glide(pt{0, 46 * k}, pt{1, 27 * k}), amp: swell(0.25, 0.5),
		rough: 1, rattle: 8.5, formants: [2]float64{210 * k, 470 * k}, breath: 1, sub: 0.8, harmonics: 18,
	}), end-0.45, 0.7)
	addAt(x, noiseBurst(r, 0.5, lowpass, 650, 0.7, 0.03, 0.16), end+0.5, 0.55)

	saturate(x, 3)
	// Fade the very end, so no roar stops with a click.
	for i, n := len(x)-samples(0.15), len(x); i < n; i++ {
		x[i] *= float64(n-i) / float64(n-(len(x)-samples(0.15)))
	}
	return x
}

// The ground shaking under a roar.
var (
	shakeUntil time.Time
	shakeAmp   float64 // tiles, at its height
)

// shakeReach is how close, in tiles, a roar must be to shake the ground.
const shakeReach = 14.0

// roarShake shakes the view for a roar d tiles off.
func roarShake(d float64) {
	if !shakeOn || d >= shakeReach { // the "Rung màn hình" setting: see settings.go
		return
	}
	a := 0.16 * (1 - d/shakeReach)
	if time.Now().Before(shakeUntil) && a <= shakeAmp {
		return
	}
	shakeAmp, shakeUntil = a, time.Now().Add(1800*time.Millisecond)
}

// shake is how far the view is thrown now, in tiles.
func shake(frame int) (dx, dy float64) {
	left := time.Until(shakeUntil).Seconds()
	if left <= 0 {
		return 0, 0
	}
	// It starts as the jaws open and dies away with the roar.
	a := shakeAmp * min(1, left/1.2) * min(1, (1.8-left)/0.25)
	t := float64(frame)
	return a * math.Sin(t*2.1) * math.Cos(t*0.73), a * 0.7 * math.Sin(t*1.7+1)
}
