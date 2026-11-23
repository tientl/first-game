package client

import (
	"math"
	"math/rand/v2"
	"slices"
)

// Background music is composed and performed in code: a chord progression,
// a bass line, an arpeggio, a pad, and a melody built from a short motif
// that repeats with variations. Each track is rendered once to a seamless
// stereo loop.

// --- instruments ------------------------------------------------------------

// instrument renders one note: frequency, length in seconds, velocity.
type instrument func(r *rand.Rand, f, dur, vel float64) []float64

func marimba(_ *rand.Rand, f, dur, vel float64) []float64 {
	return render(min(dur+0.6, 1.6), func(t float64, _ int) float64 {
		s := fsin(f*t)*math.Exp(-t/0.45) +
			0.3*fsin(f*3.93*t)*math.Exp(-t/0.07) +
			0.08*fsin(f*9.2*t)*math.Exp(-t/0.02)
		return s * vel * min(1, t/0.002)
	})
}

func guitar(r *rand.Rand, f, dur, vel float64) []float64 {
	x := pluck(r, f, min(dur+1.2, 2.5), 0.9965, 0.45)
	for i := range x {
		x[i] *= vel
	}
	return x
}

func softPluck(r *rand.Rand, f, dur, vel float64) []float64 {
	x := pluck(r, f, min(dur+0.8, 2), 0.994, 0.3)
	for i := range x {
		x[i] *= vel
	}
	return x
}

func chime(_ *rand.Rand, f, dur, vel float64) []float64 {
	x := bell(f, dur+2.2, 2.0, 1.1, 1.1)
	for i := range x {
		x[i] *= vel
	}
	return x
}

func pad(r *rand.Rand, f, dur, vel float64) []float64 {
	const attack, release = 0.9, 1.4
	det := []float64{0.997, 1, 1.004}
	ph := make([]float64, len(det))
	for i := range ph {
		ph[i] = r.Float64()
	}
	lp := newFilter(lowpass, 1400, 0.6)
	return render(dur+release, func(t float64, _ int) float64 {
		s := 0.0
		vib := 1 + 0.002*fsin(4.6*t)
		for i, d := range det {
			ph[i] += f * d * vib / sampleRate
			p := ph[i]
			s += fsin(p) + 0.35*fsin(2*p) + 0.15*fsin(3*p) + 0.07*fsin(4*p)
		}
		a := min(1, t/attack)
		if t > dur {
			a *= math.Max(0, 1-(t-dur)/release)
		}
		return lp.do(s) * a * vel * 0.33
	})
}

func bass(_ *rand.Rand, f, dur, vel float64) []float64 {
	const release = 0.25
	return render(dur+release, func(t float64, _ int) float64 {
		s := fsin(f*t) + 0.3*fsin(2*f*t) + 0.1*fsin(3*f*t)
		a := min(1, t/0.012) * math.Exp(-t/1.4)
		if t > dur {
			a *= math.Max(0, 1-(t-dur)/release)
		}
		return s * a * vel
	})
}

func kick(_ *rand.Rand, _, _, vel float64) []float64 {
	return scale(tone(0.3, 110, 48, 0.08, 0.002, 0.11), vel)
}

func shaker(r *rand.Rand, _, _, vel float64) []float64 {
	return scale(noiseBurst(r, 0.08, highpass, 6500, 0.8, 0.004, 0.018), vel)
}

func scale(x []float64, g float64) []float64 {
	for i := range x {
		x[i] *= g
	}
	return x
}

// --- score ------------------------------------------------------------------

type scoreNote struct {
	beat, dur float64 // in beats
	midi      int
	vel, pan  float64
	inst      instrument
}

type score struct {
	bpm   float64
	beats float64
	notes []scoreNote
}

func (s *score) add(beat, dur float64, midi int, vel, pan float64, inst instrument) {
	s.notes = append(s.notes, scoreNote{beat, dur, midi, vel, pan, inst})
}

// performance renders the score to a seamless stereo loop with reverb.
func (s *score) perform(r *rand.Rand, reverbWet float64) []float32 {
	spb := 60 / s.bpm
	n := samples(s.beats * spb)
	tail := samples(4)
	l, rt := make([]float64, n+tail), make([]float64, n+tail)
	for _, nt := range s.notes {
		// A touch of human timing and dynamics.
		start := samples(nt.beat*spb + (r.Float64()-0.5)*0.012)
		vel := nt.vel * (0.92 + r.Float64()*0.16)
		x := nt.inst(r, midiFreq(float64(nt.midi)), nt.dur*spb, vel)
		a := (nt.pan + 1) * math.Pi / 4
		gl, gr := math.Cos(a), math.Sin(a)
		for i, v := range x {
			j := start + i
			if j >= 0 && j < len(l) {
				l[j] += v * gl
				rt[j] += v * gr
			}
		}
	}
	reverb(l, rt, reverbWet)

	// Fold the tail (notes ringing past the end, reverb) onto the start.
	for i := 0; i < tail; i++ {
		l[i] += l[n+i]
		rt[i] += rt[n+i]
	}
	peak := 0.0
	for i := 0; i < n; i++ {
		peak = max(peak, math.Abs(l[i]), math.Abs(rt[i]))
	}
	g := 0.0
	if peak > 0 {
		g = 0.8 / peak
	}
	out := make([]float32, 2*n)
	for i := 0; i < n; i++ {
		out[2*i] = float32(l[i] * g)
		out[2*i+1] = float32(rt[i] * g)
	}
	return out
}

// reverb is a small Schroeder/Freeverb-style room: parallel damped combs,
// then allpasses, slightly different on each side for width.
func reverb(l, r []float64, wet float64) {
	do := func(x []float64, spread int) {
		combs := []int{1116, 1188, 1277, 1356, 1422, 1491}
		aps := []int{556, 441, 341}
		const fb, damp = 0.84, 0.25
		y := make([]float64, len(x))
		for _, c := range combs {
			c += spread
			buf := make([]float64, c)
			store, idx := 0.0, 0
			for i, v := range x {
				o := buf[idx]
				store = o*(1-damp) + store*damp
				buf[idx] = v*0.015 + store*fb
				y[i] += o
				idx = (idx + 1) % c
			}
		}
		for _, a := range aps {
			a += spread
			buf := make([]float64, a)
			idx := 0
			for i, v := range y {
				o := buf[idx]
				buf[idx] = v + o*0.5
				y[i] = o - v
				idx = (idx + 1) % a
			}
		}
		for i := range x {
			x[i] = x[i]*(1-wet*0.5) + y[i]*wet*4
		}
	}
	do(l, 0)
	do(r, 23)
}

// --- harmony and melody -----------------------------------------------------

// chord is a root (as a MIDI note in octave 3) and the intervals above it.
type chord struct {
	root  int
	ivals []int
}

func (c chord) pcs() []int {
	p := make([]int, len(c.ivals))
	for i, iv := range c.ivals {
		p[i] = (c.root + iv) % 12
	}
	return p
}

// tones lists the chord's notes inside [lo, hi], ascending.
func (c chord) tones(lo, hi int) []int {
	var out []int
	pcs := c.pcs()
	for m := lo; m <= hi; m++ {
		if slices.Contains(pcs, m%12) {
			out = append(out, m)
		}
	}
	return out
}

var (
	major  = []int{0, 4, 7}
	minorT = []int{0, 3, 7}
	maj7   = []int{0, 4, 7, 11}
	min7   = []int{0, 3, 7, 10}
	dom7   = []int{0, 4, 7, 10}
)

// rhythms are one-bar onset patterns for melodies, in beats.
var rhythms = [][]float64{
	{0, 1, 2, 3},
	{0, 1.5, 2, 3},
	{0, 0.5, 1, 2},
	{0, 2, 2.5, 3},
	{0, 1, 1.5, 2, 3},
	{0, 2},
	{0, 1.5, 3},
}

// melodyOver writes a melody over the bars' chords: chord tones on strong
// beats, stepwise motion through the scale in between. It returns notes with
// beats relative to the first bar.
func melodyOver(r *rand.Rand, chords []chord, scalePCs []int, lo, hi int, rhy [][]float64, prev int) []scoreNote {
	var pool []int
	for m := lo; m <= hi; m++ {
		if slices.Contains(scalePCs, m%12) {
			pool = append(pool, m)
		}
	}
	nearest := func(cands []int, to int) int {
		slices.SortFunc(cands, func(a, b int) int { return iabs(a-to) - iabs(b-to) })
		if len(cands) > 1 && r.IntN(3) == 0 {
			return cands[1]
		}
		return cands[0]
	}
	var out []scoreNote
	for b, ch := range chords {
		pat := rhy[r.IntN(len(rhy))]
		for k, on := range pat {
			end := 4.0
			if k+1 < len(pat) {
				end = pat[k+1]
			}
			if on != 0 && r.IntN(9) == 0 {
				continue // a breath
			}
			var m int
			if on == 0 || on == 2 {
				m = nearest(ch.tones(lo, hi), prev)
			} else {
				i := slices.Index(pool, prev)
				if i < 0 {
					i = len(pool) / 2
				}
				i += []int{-2, -1, -1, 1, 1, 2}[r.IntN(6)]
				m = pool[max(0, min(len(pool)-1, i))]
			}
			out = append(out, scoreNote{beat: float64(b)*4 + on, dur: (end - on) * 0.95, midi: m})
			prev = m
		}
	}
	return out
}

// cadence replaces the melody in the last bar with one long chord tone.
func cadence(mel []scoreNote, lastBar int, ch chord, lo, hi int) []scoreNote {
	out := mel[:0:0]
	prev := (lo + hi) / 2
	for _, n := range mel {
		if n.beat < float64(lastBar)*4 {
			out = append(out, n)
			prev = n.midi
		}
	}
	tones := ch.tones(lo, hi)
	best := tones[0]
	for _, t := range tones {
		if t%12 == ch.root%12 && abs(float64(t-prev)) < abs(float64(best-prev))+3 {
			best = t
		}
	}
	return append(out, scoreNote{beat: float64(lastBar) * 4, dur: 3.8, midi: best})
}

func (s *score) addMelody(mel []scoreNote, offsetBars int, vel, pan float64, inst instrument) {
	for _, n := range mel {
		s.add(float64(offsetBars)*4+n.beat, n.dur, n.midi, vel, pan, inst)
	}
}

// --- tracks -----------------------------------------------------------------

func genMusic(name string) []float32 {
	r := rand.New(rand.NewPCG(7, uint64(len(name))*131+uint64(name[0])))
	var s *score
	wet := 0.3
	switch name {
	case "day":
		s, wet = composeDay(r), 0.22
	case "night":
		s, wet = composeNight(r), 0.38
	case "cave":
		s, wet = composeCave(r), 0.55
	default:
		s = composeMenu(r)
	}
	return s.perform(r, wet)
}

// composeDay: a light, walking tune in C major. Marimba arpeggios, a plucked
// melody, soft bass, and a shaker once the tune gets going.
func composeDay(r *rand.Rand) *score {
	C, Am, F, G := chord{48, major}, chord{57, minorT}, chord{53, major}, chord{55, major}
	Em, Dm := chord{52, minorT}, chord{50, minorT}
	progA := []chord{C, Am, F, G, C, Am, F, G}
	progB := []chord{F, G, Em, Am, Dm, F, G, G}
	bars := append(append(append([]chord{}, progA...), progB...), progA...)

	s := &score{bpm: 88, beats: float64(len(bars)) * 4}
	penta := []int{0, 2, 4, 7, 9}
	motif := melodyOver(r, progA[:4], penta, 67, 84, rhythms[:5], 76)
	melA := append(append([]scoreNote{}, motif...), shift(motif[:countBefore(motif, 12)], 4)...)
	melA = cadence(melA, 7, C, 67, 84)
	melB := cadence(melodyOver(r, progB, penta, 71, 88, rhythms, 79), 7, G, 71, 88)

	s.addMelody(melA, 0, 0.55, 0.15, softPluck)
	s.addMelody(melB, 8, 0.55, 0.15, softPluck)
	s.addMelody(melA, 16, 0.55, 0.15, softPluck)
	s.addMelody(melA, 16, 0.2, -0.3, marimba) // doubled the last time round

	arp := [][]int{{0, 1, 2, 3, 2, 1, 2, 1}, {0, 2, 1, 3, 0, 2, 1, 2}}
	for b, ch := range bars {
		tones := ch.tones(60, 79)
		pat := arp[(b/8)%2]
		for k, idx := range pat {
			v := 0.32
			if k%2 == 0 {
				v = 0.42
			}
			s.add(float64(b*4)+float64(k)*0.5, 0.5, tones[idx%len(tones)], v, -0.35, marimba)
		}
		root := ch.root - 12
		s.add(float64(b*4), 1.8, root, 0.6, 0, bass)
		s.add(float64(b*4)+2, 1.8, root+7, 0.45, 0, bass)
		for _, t := range ch.tones(55, 71)[:3] {
			s.add(float64(b*4), 4, t, 0.18, 0.35, pad)
		}
		if b >= 8 {
			for k := 0; k < 8; k++ {
				v := 0.18
				if k%2 == 1 {
					v = 0.3
				}
				s.add(float64(b*4)+float64(k)*0.5, 0.25, 0, v, 0.45, shaker)
			}
			s.add(float64(b*4), 1, 0, 0.5, 0, kick)
			s.add(float64(b*4)+2, 1, 0, 0.4, 0, kick)
		}
	}
	return s
}

// composeNight: slow and quiet in A minor. Long pads, sparse bells, a gently
// plucked pulse.
func composeNight(r *rand.Rand) *score {
	Am, F, C, G := chord{57, minorT}, chord{53, major}, chord{48, major}, chord{55, major}
	Dm, Em, E := chord{50, minorT}, chord{52, minorT}, chord{52, dom7}
	bars := []chord{Am, F, C, G, Am, F, Em, Em, Dm, Am, F, E, Dm, Am, E, Am}

	s := &score{bpm: 62, beats: float64(len(bars)) * 4}
	pentaMinor := []int{9, 0, 2, 4, 7}
	sparse := [][]float64{{0, 2}, {0, 1.5, 3}, {0}, {0, 3}}
	m1 := melodyOver(r, bars[:4], pentaMinor, 69, 84, sparse, 76)
	m2 := cadence(melodyOver(r, bars[8:], pentaMinor, 69, 86, sparse, 81), 7, Am, 69, 86)
	s.addMelody(m1, 4, 0.45, 0.25, chime)
	s.addMelody(m2, 8, 0.45, 0.25, chime)

	for b, ch := range bars {
		for _, t := range ch.tones(52, 69)[:3] {
			s.add(float64(b*4), 4, t, 0.3, -0.2, pad)
		}
		s.add(float64(b*4), 3.8, ch.root-12, 0.45, 0, bass)
		if b >= 2 {
			tones := ch.tones(57, 76)
			for k := 0; k < 4; k++ {
				s.add(float64(b*4+k), 1, tones[(k*2+b)%len(tones)], 0.22, 0.4, softPluck)
			}
		}
	}
	return s
}

// composeMenu: warm fingerpicked guitar in D with a pad and a bell melody.
func composeMenu(r *rand.Rand) *score {
	D, Bm, G, A := chord{50, maj7}, chord{59 - 12, min7}, chord{55, maj7}, chord{57, major}
	Em, Fsm := chord{52, min7}, chord{54, min7}
	bars := []chord{D, Bm, G, A, D, Bm, G, A, Em, A, Fsm, Bm, G, A, D, D}

	s := &score{bpm: 74, beats: float64(len(bars)) * 4}
	scaleD := []int{2, 4, 6, 9, 11}
	m := melodyOver(r, bars[:4], scaleD, 69, 86, rhythms[:6], 78)
	mel := append(append([]scoreNote{}, m...), shift(m[:countBefore(m, 12)], 4)...)
	mel = cadence(mel, 7, A, 69, 86)
	mB := cadence(melodyOver(r, bars[8:], scaleD, 71, 88, rhythms, 81), 7, D, 71, 88)
	s.addMelody(mel, 0, 0.4, 0.2, chime)
	s.addMelody(mB, 8, 0.42, 0.2, chime)

	for b, ch := range bars {
		bassNote := 40 + ((ch.root%12-40%12)+12)%12
		hi := ch.tones(57, 74)
		pick := []int{-1, 1, 2, 3, 0, 2, 1, 3} // -1: bass string
		for k, idx := range pick {
			note := bassNote
			v := 0.5
			if idx >= 0 {
				note, v = hi[idx%len(hi)], 0.32
			}
			s.add(float64(b*4)+float64(k)*0.5, 0.5, note, v, -0.25+float64(k%3)*0.1, guitar)
		}
		for _, t := range ch.tones(55, 72)[:3] {
			s.add(float64(b*4), 4, t, 0.16, 0.3, pad)
		}
	}
	return s
}

func shift(notes []scoreNote, bars int) []scoreNote {
	out := make([]scoreNote, len(notes))
	for i, n := range notes {
		n.beat += float64(bars) * 4
		out[i] = n
	}
	return out
}

// countBefore counts the notes that start before the given beat.
func countBefore(notes []scoreNote, beat float64) int {
	n := 0
	for _, x := range notes {
		if x.beat < beat {
			n++
		}
	}
	return n
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// composeCave: deep and still in D minor, for underground. A low drone, a
// slow pad, and a few bell notes falling into a big echo.
func composeCave(r *rand.Rand) *score {
	Dm, Bb, Gm, A := chord{50, minorT}, chord{58 - 12, major}, chord{55, minorT}, chord{57, minorT}
	bars := []chord{Dm, Dm, Bb, A, Dm, Gm, Bb, A, Gm, Dm, Bb, Dm}

	s := &score{bpm: 54, beats: float64(len(bars)) * 4}
	scale := []int{2, 5, 7, 9, 0} // D minor pentatonic
	sparse := [][]float64{{0}, {2}, {0, 3}, {1.5}}
	mel := melodyOver(r, bars, scale, 62, 81, sparse, 74)
	s.addMelody(mel, 0, 0.35, -0.3, chime)
	for b, ch := range bars {
		s.add(float64(b*4), 4, 38, 0.32, 0, bass) // the drone on D, always
		for _, t := range ch.tones(50, 65)[:3] {
			s.add(float64(b*4), 4, t, 0.24, 0.25, pad)
		}
		if b%2 == 1 { // a far-off pluck now and then, like a drip
			s.add(float64(b*4)+2.5, 1, ch.tones(74, 86)[0], 0.18, 0.6, softPluck)
		}
	}
	return s
}
