package client

import (
	"math"
	"math/rand/v2"
)

// --- synthesis toolkit -------------------------------------------------------

const twoPi = 2 * math.Pi

// sineTab makes oscillators cheap: phase in cycles, linear interpolation.
var sineTab = func() []float64 {
	t := make([]float64, 4097)
	for i := range t {
		t[i] = math.Sin(twoPi * float64(i) / 4096)
	}
	return t
}()

func fsin(phase float64) float64 {
	phase -= math.Floor(phase)
	p := phase * 4096
	i := int(p)
	return sineTab[i] + (sineTab[i+1]-sineTab[i])*(p-float64(i))
}

type filterKind uint8

const (
	lowpass filterKind = iota
	highpass
	bandpass
)

// biquad is an RBJ cookbook filter.
type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func newFilter(k filterKind, f, q float64) *biquad {
	b := &biquad{}
	b.set(k, f, q)
	return b
}

func (b *biquad) set(k filterKind, f, q float64) {
	f = max(10, min(f, sampleRate*0.45))
	w := twoPi * f / sampleRate
	cs, sn := math.Cos(w), math.Sin(w)
	alpha := sn / (2 * q)
	var b0, b1, b2 float64
	switch k {
	case lowpass:
		b0, b1, b2 = (1-cs)/2, 1-cs, (1-cs)/2
	case highpass:
		b0, b1, b2 = (1+cs)/2, -(1 + cs), (1+cs)/2
	default:
		b0, b1, b2 = alpha, 0, -alpha
	}
	a0 := 1 + alpha
	b.b0, b.b1, b.b2 = b0/a0, b1/a0, b2/a0
	b.a1, b.a2 = -2*cs/a0, (1-alpha)/a0
}

func (b *biquad) do(x float64) float64 {
	y := b.b0*x + b.b1*b.x1 + b.b2*b.x2 - b.a1*b.y1 - b.a2*b.y2
	b.x2, b.x1, b.y2, b.y1 = b.x1, x, b.y1, y
	return y
}

func samples(sec float64) int { return int(sec * sampleRate) }

// env is a linear attack followed by an exponential decay.
func env(t, attack, decay float64) float64 {
	if t < 0 {
		return 0
	}
	if t < attack {
		return t / attack
	}
	return math.Exp(-(t - attack) / decay)
}

// render fills a buffer of dur seconds from f(t, i).
func render(dur float64, f func(t float64, i int) float64) []float64 {
	out := make([]float64, samples(dur))
	for i := range out {
		out[i] = f(float64(i)/sampleRate, i)
	}
	return out
}

// addAt mixes src into dst starting at time t.
func addAt(dst, src []float64, t, gain float64) {
	o := samples(t)
	for i, v := range src {
		if o+i >= 0 && o+i < len(dst) {
			dst[o+i] += v * gain
		}
	}
}

// finish normalises to the given peak, fades the last few milliseconds so
// nothing clicks, and converts to float32.
func finish(x []float64, peak float64) []float32 { return normalize(x, peak, true) }

func normalize(x []float64, peak float64, fadeEnd bool) []float32 {
	m := 0.0
	for _, v := range x {
		m = max(m, math.Abs(v))
	}
	g := 0.0
	if m > 0 {
		g = peak / m
	}
	out := make([]float32, len(x))
	fade := min(len(x)/4, samples(0.01))
	if !fadeEnd {
		fade = 0
	}
	for i, v := range x {
		if j := len(x) - 1 - i; j < fade {
			v *= float64(j) / float64(fade)
		}
		out[i] = float32(v * g)
	}
	return out
}

// noiseBurst is filtered noise shaped by an envelope.
func noiseBurst(r *rand.Rand, dur float64, k filterKind, f, q, attack, decay float64) []float64 {
	flt := newFilter(k, f, q)
	return render(dur, func(t float64, _ int) float64 {
		return flt.do(r.Float64()*2-1) * env(t, attack, decay)
	})
}

// tone is a sine whose frequency glides from f0 to f1 over the glide time.
func tone(dur, f0, f1, glide, attack, decay float64) []float64 {
	ph := 0.0
	return render(dur, func(t float64, _ int) float64 {
		f := f1
		if t < glide {
			f = f0 + (f1-f0)*t/glide
		}
		ph += f / sampleRate
		return fsin(ph) * env(t, attack, decay)
	})
}

// partials is a struck object: inharmonic sine partials, each decaying.
func partials(dur, f0 float64, ratios, amps, decays []float64) []float64 {
	return render(dur, func(t float64, _ int) float64 {
		s := 0.0
		for i, rt := range ratios {
			s += fsin(f0*rt*t) * amps[i] * math.Exp(-t/decays[i])
		}
		return s * min(1, t/0.0015)
	})
}

// grains is noise switched on in short random bursts: rustling, crunching.
func grains(r *rand.Rand, dur float64, density, grainLen float64, k filterKind, f, q float64, shape func(t float64) float64) []float64 {
	flt := newFilter(k, f, q)
	out := make([]float64, samples(dur))
	n := int(density * dur)
	for g := 0; g < n; g++ {
		start := r.Float64() * dur
		amp := 0.3 + r.Float64()*0.7
		gl := grainLen * (0.5 + r.Float64())
		for i := samples(start); i < samples(start+gl) && i < len(out); i++ {
			t := float64(i)/sampleRate - start
			out[i] += amp * math.Sin(math.Pi*t/gl) * (r.Float64()*2 - 1)
		}
	}
	for i := range out {
		out[i] = flt.do(out[i]) * shape(float64(i)/sampleRate)
	}
	return out
}

// pluck is a Karplus-Strong string.
func pluck(r *rand.Rand, freq, dur, damping, brightness float64) []float64 {
	n := max(2, int(math.Round(sampleRate/freq)))
	line := make([]float64, n)
	lp := 0.0
	for i := range line {
		lp += brightness * ((r.Float64()*2 - 1) - lp)
		line[i] = lp
	}
	out := make([]float64, samples(dur))
	idx := 0
	for i := range out {
		next := (idx + 1) % n
		v := line[idx]
		line[idx] = damping * 0.5 * (line[idx] + line[next])
		out[i] = v
		idx = next
	}
	return out
}

// bell is a two-operator FM bell.
func bell(freq, dur, ratio, index, decay float64) []float64 {
	return render(dur, func(t float64, _ int) float64 {
		m := fsin(freq*ratio*t) * index * math.Exp(-t/(decay*0.4))
		return fsin(freq*t+m/twoPi) * env(t, 0.004, decay)
	})
}

func midiFreq(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

// --- sound effects ------------------------------------------------------------

func genSFX() map[string][][]float32 {
	r := rand.New(rand.NewPCG(2026, 11))
	s := make(map[string][][]float32)
	add := func(name string, peak float64, x []float64) { s[name] = append(s[name], finish(x, peak)) }

	for v := 0; v < 3; v++ {
		k := 1 + float64(v)*0.08
		x := tone(0.35, 170*k, 75*k, 0.08, 0.001, 0.06)
		addAt(x, noiseBurst(r, 0.06, bandpass, 1900*k, 1.4, 0.0005, 0.012), 0, 0.7)
		addAt(x, partials(0.3, 400*k, []float64{1, 1.52, 2.4}, []float64{0.4, 0.25, 0.1}, []float64{0.06, 0.045, 0.03}), 0, 1)
		add("chop", 1, x)
	}

	{ // A creak, a crash, and leaves settling.
		x := make([]float64, samples(2.4))
		creak := newFilter(bandpass, 700, 3)
		ph := 0.0
		for i := 0; i < samples(0.7); i++ {
			t := float64(i) / sampleRate
			f := 70 + 25*math.Sin(twoPi*2.5*t) + 15*t
			ph += f / sampleRate
			saw := 0.0
			for h := 1.0; h <= 8; h++ {
				saw += fsin(ph*h) / h
			}
			x[i] += creak.do(saw) * math.Sin(math.Pi*t/0.7) * 0.8
		}
		crashF := newFilter(lowpass, 3000, 0.7)
		for i := samples(0.55); i < len(x); i++ {
			t := float64(i)/sampleRate - 0.55
			crashF.set(lowpass, 3500*math.Exp(-t/0.5)+300, 0.7)
			x[i] += crashF.do(r.Float64()*2-1) * env(t, 0.01, 0.35)
			x[i] += fsin(55*t) * env(t, 0.005, 0.18) * 1.2
		}
		addAt(x, grains(r, 1.6, 140, 0.02, highpass, 2500, 0.7, func(t float64) float64 { return math.Exp(-t / 0.5) }), 0.7, 0.5)
		add("treefall", 1, x)
	}

	for v := 0; v < 3; v++ {
		f := []float64{1300, 1520, 1150}[v]
		x := partials(0.4, f, []float64{1, 2.32, 4.25, 6.63}, []float64{1, 0.6, 0.4, 0.25}, []float64{0.1, 0.07, 0.04, 0.025})
		addAt(x, noiseBurst(r, 0.03, bandpass, 3200, 1, 0.0003, 0.006), 0, 0.9)
		addAt(x, tone(0.1, 130, 100, 0.05, 0.001, 0.03), 0, 0.6)
		add("stone", 1, x)
	}

	{
		x := make([]float64, samples(1.1))
		for g := 0; g < 7; g++ {
			addAt(x, noiseBurst(r, 0.12, bandpass, 700+r.Float64()*2200, 1.2, 0.001, 0.03+r.Float64()*0.03), r.Float64()*0.22, 0.5+r.Float64()*0.5)
		}
		addAt(x, tone(0.3, 110, 60, 0.15, 0.002, 0.08), 0, 1.2)
		for g := 0; g < 14; g++ { // pebbles skittering away
			t := 0.15 + math.Pow(r.Float64(), 1.6)*0.85
			addAt(x, partials(0.05, 2000+r.Float64()*2500, []float64{1, 2.7}, []float64{1, 0.4}, []float64{0.012, 0.006}), t, 0.25*(1.1-t))
		}
		add("rockbreak", 1, x)
	}

	for v := 0; v < 3; v++ {
		x := grains(r, 0.4, 260, 0.018, highpass, 1800+float64(v)*400, 0.7, func(t float64) float64 { return env(t, 0.03, 0.11) })
		add("rustle", 1, x)
	}

	sparkle := func(x []float64, f float64) {
		addAt(x, tone(0.25, f, f, 0, 0.002, 0.07), 0.04, 0.18)
		addAt(x, tone(0.25, f*1.5, f*1.5, 0, 0.002, 0.06), 0.08, 0.12)
	}
	{
		x := make([]float64, samples(0.35))
		addAt(x, partials(0.12, 520, []float64{1, 1.5, 2.9}, []float64{1, 0.5, 0.2}, []float64{0.03, 0.02, 0.01}), 0, 1)
		addAt(x, partials(0.12, 610, []float64{1, 1.5, 2.9}, []float64{1, 0.5, 0.2}, []float64{0.03, 0.02, 0.01}), 0.07, 0.8)
		sparkle(x, 1568)
		add("pick_wood", 1, x)
	}
	{
		x := make([]float64, samples(0.35))
		addAt(x, partials(0.1, 2100, []float64{1, 2.3, 4.1}, []float64{1, 0.5, 0.3}, []float64{0.025, 0.015, 0.01}), 0, 1)
		addAt(x, partials(0.1, 2400, []float64{1, 2.3, 4.1}, []float64{1, 0.5, 0.3}, []float64{0.02, 0.015, 0.01}), 0.06, 0.6)
		sparkle(x, 1760)
		add("pick_stone", 1, x)
	}
	{
		flt := newFilter(bandpass, 2000, 1.2)
		x := render(0.35, func(t float64, _ int) float64 {
			flt.set(bandpass, 1800+t*14000, 1.2)
			return flt.do(r.Float64()*2-1) * env(t, 0.03, 0.06)
		})
		sparkle(x, 1976)
		add("pick_grass", 1, x)
	}
	{ // A dry twig: a quick snap, then the sparkle.
		x := noiseBurst(r, 0.05, bandpass, 2600, 2.5, 0.0003, 0.006)
		addAt(x, partials(0.08, 900, []float64{1, 2.2, 3.7}, []float64{1, 0.6, 0.3}, []float64{0.02, 0.012, 0.008}), 0.003, 0.7)
		addAt(x, noiseBurst(r, 0.04, bandpass, 3800, 2, 0.0003, 0.004), 0.035, 0.5)
		x = append(x, make([]float64, samples(0.25))...)
		sparkle(x, 1661)
		add("pick_stick", 1, x)
	}
	{
		x := tone(0.35, 380, 950, 0.05, 0.002, 0.05)
		addAt(x, tone(0.1, 1400, 2200, 0.03, 0.001, 0.015), 0.02, 0.3)
		sparkle(x, 2093)
		add("pick_berry", 1, x)
	}

	{
		x := make([]float64, samples(0.65))
		for _, t := range []float64{0, 0.17, 0.34} {
			addAt(x, grains(r, 0.1, 500, 0.004, bandpass, 2600, 0.9, func(t float64) float64 { return env(t, 0.003, 0.035) }), t, 1)
			addAt(x, tone(0.08, 190, 150, 0.04, 0.002, 0.025), t, 0.5)
		}
		add("eat", 1, x)
	}

	crackles := func(x []float64, from, to float64, n int, gain float64) {
		for g := 0; g < n; g++ {
			t := from + r.Float64()*(to-from)
			addAt(x, noiseBurst(r, 0.02, bandpass, 1500+r.Float64()*4000, 2, 0.0002, 0.002+r.Float64()*0.004), t, gain*(0.3+r.Float64()*0.7))
		}
	}
	{
		flt := newFilter(lowpass, 300, 0.8)
		x := render(1.5, func(t float64, _ int) float64 {
			f := 300 + 2400*math.Sin(math.Pi*min(1, t/0.9))
			flt.set(lowpass, f, 0.8)
			return flt.do(r.Float64()*2-1) * env(t, 0.25, 0.4)
		})
		crackles(x, 0.15, 1.4, 22, 0.8)
		add("ignite", 1, x)
	}
	{
		x := tone(0.7, 150, 80, 0.06, 0.001, 0.05)
		flt := newFilter(lowpass, 600, 0.8)
		addAt(x, render(0.6, func(t float64, _ int) float64 {
			flt.set(lowpass, 500+1500*math.Sin(math.Pi*min(1, t/0.5)), 0.8)
			return flt.do(r.Float64()*2-1) * env(t, 0.08, 0.15)
		}), 0.05, 0.8)
		crackles(x, 0.1, 0.65, 10, 0.7)
		add("stoke", 1, x)
	}
	{
		x := noiseBurst(r, 1.2, highpass, 3500, 0.7, 0.02, 0.35)
		crackles(x, 0, 0.6, 6, 0.5)
		add("fizzle", 0.8, x)
	}
	{ // Hammer taps for a finished craft.
		x := make([]float64, samples(0.5))
		for i, t := range []float64{0, 0.13} {
			addAt(x, tone(0.2, 220, 120, 0.04, 0.001, 0.04), t, 1)
			addAt(x, noiseBurst(r, 0.04, bandpass, 2500, 1.5, 0.0003, 0.008), t, 0.6-float64(i)*0.2)
		}
		add("craft", 1, x)
	}

	for v := 0; v < 4; v++ {
		x := noiseBurst(r, 0.14, lowpass, 700+float64(v)*90, 0.8, 0.008, 0.028)
		addAt(x, noiseBurst(r, 0.08, highpass, 3000, 0.7, 0.004, 0.015), 0.005, 0.15)
		add("step_grass", 1, x)

		y := grains(r, 0.14, 1400, 0.002, bandpass, 2800+float64(v)*200, 0.8, func(t float64) float64 { return env(t, 0.006, 0.035) })
		addAt(y, noiseBurst(r, 0.08, lowpass, 500, 0.7, 0.005, 0.02), 0, 0.5)
		add("step_sand", 1, y)

		z := noiseBurst(r, 0.1, bandpass, 3400+float64(v)*300, 2, 0.0003, 0.005)
		addAt(z, tone(0.08, 140, 110, 0.03, 0.001, 0.018), 0, 0.8)
		add("step_rock", 1, z)

		w := grains(r, 0.16, 700, 0.004, bandpass, 1800+float64(v)*150, 0.9, func(t float64) float64 { return env(t, 0.008, 0.04) })
		addAt(w, noiseBurst(r, 0.1, lowpass, 600, 0.7, 0.006, 0.025), 0, 0.6)
		add("step_forest", 1, w)
	}

	{
		flt := newFilter(lowpass, 1300, 0.9)
		ph := 0.0
		x := render(0.3, func(t float64, _ int) float64 {
			f := 230 - 280*t
			ph += f / sampleRate
			saw := 0.0
			for h := 1.0; h <= 6; h++ {
				saw += fsin(ph*h) / h
			}
			return flt.do(saw) * env(t, 0.01, 0.09)
		})
		add("hurt", 1, x)
	}
	{
		flt := newFilter(bandpass, 450, 2)
		x := render(0.9, func(t float64, _ int) float64 {
			trem := 0.55 + 0.45*fsin(24*t)
			return flt.do(r.Float64()*2-1) * trem * math.Sin(math.Pi*t/0.9)
		})
		for t := 0.05; t < 0.8; t += 0.045 {
			addAt(x, partials(0.03, 2600, []float64{1}, []float64{1}, []float64{0.004}), t, 0.15)
		}
		add("shiver", 1, x)
	}

	{
		x := make([]float64, samples(3))
		for i, m := range []float64{69, 65, 62, 57} {
			addAt(x, bell(midiFreq(m), 1.8, 1.4, 2.5, 0.9), float64(i)*0.38, 0.9)
		}
		addAt(x, tone(3, midiFreq(45), midiFreq(45), 0, 0.4, 1.1), 0, 0.35)
		add("death", 1, x)
	}
	{
		x := make([]float64, samples(1.8))
		for i, m := range []float64{72, 76, 79, 84} {
			addAt(x, bell(midiFreq(m), 1.2, 3.5, 1.2, 0.5), float64(i)*0.1, 0.8)
		}
		add("respawn", 1, x)
	}
	{
		flt := newFilter(lowpass, 900, 0.8)
		x := render(0.3, func(t float64, _ int) float64 {
			on := (t < 0.07) || (t > 0.11 && t < 0.18)
			if !on {
				return flt.do(0)
			}
			sq := 1.0
			if fsin(196*t) < 0 {
				sq = -1
			}
			return flt.do(sq) * 0.8
		})
		add("error", 0.7, x)
	}

	{
		x := tone(0.06, 1500, 1500, 0, 0.001, 0.012)
		addAt(x, tone(0.05, 3000, 3000, 0, 0.001, 0.008), 0, 0.3)
		add("ui_move", 1, x)
	}
	{
		x := make([]float64, samples(0.45))
		addAt(x, pluck(r, midiFreq(76), 0.4, 0.996, 0.5), 0, 1)
		addAt(x, pluck(r, midiFreq(83), 0.4, 0.996, 0.5), 0.07, 0.9)
		add("ui_select", 1, x)
	}
	{
		x := tone(0.3, 330, 220, 0.12, 0.01, 0.08)
		addAt(x, noiseBurst(r, 0.25, lowpass, 800, 0.7, 0.04, 0.06), 0, 0.4)
		add("ui_pause", 1, x)
	}
	{
		x := make([]float64, samples(1.4))
		addAt(x, bell(midiFreq(79), 1.2, 3.5, 1.5, 0.45), 0, 0.9)
		addAt(x, bell(midiFreq(86), 1.2, 3.5, 1.5, 0.45), 0.12, 0.8)
		add("save", 1, x)
	}

	for v := 0; v < 5; v++ {
		x := make([]float64, samples(1.0))
		t := 0.0
		n := 2 + r.IntN(4)
		base := 2800 + r.Float64()*1600
		up := v%2 == 0
		for c := 0; c < n; c++ {
			dur := 0.04 + r.Float64()*0.06
			f0, f1 := base, base*(1.3+r.Float64()*0.4)
			if !up {
				f0, f1 = f1, f0
			}
			ph := 0.0
			chirp := render(dur, func(tt float64, _ int) float64 {
				f := f0 + (f1-f0)*tt/dur + 250*fsin(38*tt)
				ph += f / sampleRate
				return fsin(ph) * math.Sin(math.Pi*tt/dur)
			})
			addAt(x, chirp, t, 0.6+r.Float64()*0.4)
			t += dur + 0.03 + r.Float64()*0.08
		}
		add("bird", 1, x)
	}
	genCreatureSFX(r, add) // last, so the rng gives the sounds above as before
	genCraftSFX(r, add)
	return s
}

// --- ambience loops ---------------------------------------------------------

// seamless renders dur seconds plus a crossfade tail and folds the tail onto
// the head, so the result loops without a seam.
func seamless(dur float64, x []float64) []float32 {
	n := samples(dur)
	fade := len(x) - n
	out := make([]float64, n)
	copy(out, x[:n])
	for i := 0; i < fade; i++ {
		a := float64(i) / float64(fade)
		out[i] = out[i]*a + x[n+i]*(1-a)
	}
	return normalize(out, 0.9, false) // a loop must not fade at its end
}

func genLoops() map[string][]float32 {
	r := rand.New(rand.NewPCG(99, 1))
	loops := make(map[string][]float32)
	const xf = 1.5 // crossfade seconds

	{ // Ocean: brown noise swelling with each wave, foam hissing at the peaks.
		const dur = 24.0
		total := dur + xf
		swell := make([]float64, samples(total))
		for t := 0.0; t < total; t += 4 + r.Float64()*3 {
			amp := 0.6 + r.Float64()*0.4
			for i := samples(t); i < len(swell) && i < samples(t+7); i++ {
				tt := float64(i)/sampleRate - t
				swell[i] += amp * env(tt, 1.6, 1.6)
			}
		}
		lp, hp := newFilter(lowpass, 600, 0.7), newFilter(highpass, 3500, 0.7)
		brown := 0.0
		x := render(total, func(t float64, i int) float64 {
			w := r.Float64()*2 - 1
			brown = brown*0.995 + w*0.06
			s := 0.25 + swell[i]
			lp.set(lowpass, 350+900*min(1, s), 0.7)
			return lp.do(brown)*s*1.4 + hp.do(w)*s*s*0.09
		})
		loops["ocean"] = seamless(dur, x)
	}
	{ // Fire: a low rumble with crackles and pops.
		const dur = 8.0
		lp := newFilter(lowpass, 160, 0.7)
		x := render(dur+xf, func(t float64, _ int) float64 {
			return lp.do(r.Float64()*2-1) * (0.7 + 0.3*fsin(0.7*t))
		})
		for g := 0; g < int((dur+xf)*16); g++ {
			t := r.Float64() * (dur + xf)
			amp := math.Pow(r.Float64(), 2.5)
			addAt(x, noiseBurst(r, 0.02, bandpass, 1200+r.Float64()*5000, 1.5, 0.0002, 0.0015+r.Float64()*0.005), t, 0.15+amp*1.6)
		}
		loops["fire"] = seamless(dur, x)
	}
	{ // Wind: band-passed noise whose pitch and strength drift.
		const dur = 18.0
		bp := newFilter(bandpass, 500, 1.2)
		x := render(dur+xf, func(t float64, _ int) float64 {
			m := vnoise(t*0.35, 0, 600)
			bp.set(bandpass, 280+700*m, 1.4)
			return bp.do(r.Float64()*2-1) * (0.35 + 0.65*vnoise(t*0.25, 3, 601))
		})
		loops["wind"] = seamless(dur, x)
	}
	{ // Crickets: two near ones trilling in groups, one far away.
		const dur = 9.0
		x := make([]float64, samples(dur+xf))
		for _, cfg := range []struct{ f, gap, gain float64 }{{4300, 0.62, 1}, {4750, 0.81, 0.8}, {3900, 0.53, 0.35}} {
			t := r.Float64() * 0.5
			for t < dur+xf {
				pulses := 3 + r.IntN(2)
				for p := 0; p < pulses; p++ {
					pt := t + float64(p)*0.033
					addAt(x, render(0.025, func(tt float64, _ int) float64 {
						return fsin(cfg.f*tt) * math.Sin(math.Pi*tt/0.025)
					}), pt, cfg.gain)
				}
				t += cfg.gap * (0.85 + r.Float64()*0.3)
			}
		}
		loops["crickets"] = seamless(dur, x)
	}
	{ // A cave: a low hollow rumble, water dripping, each drip echoing.
		const dur = 14.0
		lp := newFilter(lowpass, 120, 0.7)
		x := render(dur+xf, func(t float64, _ int) float64 {
			return lp.do(r.Float64()*2-1) * (0.45 + 0.15*fsin(0.13*t))
		})
		for t := 0.4; t < dur+xf; t += 0.5 + r.Float64()*1.6 {
			f := 900 + r.Float64()*1400
			drip := render(0.12, func(tt float64, _ int) float64 {
				return fsin((f-3000*tt)*tt) * env(tt, 0.001, 0.025)
			})
			g := 0.5 + r.Float64()*0.5
			for e, delay := range []float64{0, 0.19, 0.41, 0.66} { // the echoes
				addAt(x, drip, t+delay, g*math.Pow(0.45, float64(e)))
			}
		}
		loops["cave"] = seamless(dur, x)
	}
	return loops
}
