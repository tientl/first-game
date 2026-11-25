package client

import (
	"math"
	"math/rand/v2"

	"survivalgame/internal/sim"
)

// Battle music for the bosses, composed in code like the rest of the music
// (music.go). It rises when a boss wakes and fights a player (the boss bar
// shows), and fades back to the day's or night's music when the fight ends.
// Each boss has a theme of its land:
//
//   - "boss_jungle" (Titanoboa, and the land bosses): war drums of the deep
//     jungle, a driving low-string ostinato in D Phrygian, a brass call;
//   - "boss_sea" (Megalodon, Spinosaurus): slow and huge, a pulse in the deep,
//     tremolo strings, a choir and brass stabs, timpani at each phrase's end.
//
// When the last blow falls, a short brass fanfare plays (the "boss_win"
// sound).

// bossTrack is the battle music of boss k.
func bossTrack(k sim.CreatureKind) string {
	switch k {
	case sim.CreatureMegalodon, sim.CreatureSpino:
		return "boss_sea"
	}
	return "boss_jungle"
}

// --- instruments ------------------------------------------------------------

// tom is a big drum: a skin's thump falling in pitch, and the stick's slap.
func tom(r *rand.Rand, f, _, vel float64) []float64 {
	x := tone(0.7, f*1.7, f, 0.06, 0.002, 0.28)
	slap := noiseBurst(r, 0.05, bandpass, 900, 0.9, 0.001, 0.012)
	for i, v := range slap {
		x[i] += v * 0.35
	}
	return scale(x, vel)
}

// brass is a horn: harmonics that brighten as the note swells.
func brass(r *rand.Rand, f, dur, vel float64) []float64 {
	const attack, release = 0.06, 0.18
	lp := newFilter(lowpass, 600, 0.8)
	ph := r.Float64()
	return render(dur+release, func(t float64, i int) float64 {
		if i%64 == 0 {
			lp.set(lowpass, 500+2200*min(1, t/0.25), 0.8)
		}
		ph += f * (1 + 0.003*fsin(5.2*t)) / sampleRate
		s := 0.0
		for k := 1.0; k <= 9; k++ {
			s += fsin(ph*k) / k
		}
		a := min(1, t/attack)
		if t > dur {
			a *= math.Max(0, 1-(t-dur)/release)
		}
		return lp.do(s) * a * vel * 0.5
	})
}

// strings is a bowed section: three detuned saws, slow to speak. trem
// shakes the bow (0 for none).
func bowed(trem float64) instrument {
	return func(r *rand.Rand, f, dur, vel float64) []float64 {
		const attack, release = 0.05, 0.22
		det := []float64{0.996, 1, 1.005}
		ph := []float64{r.Float64(), r.Float64(), r.Float64()}
		lp := newFilter(lowpass, 2400, 0.7)
		return render(dur+release, func(t float64, _ int) float64 {
			s := 0.0
			for j, d := range det {
				ph[j] += f * d / sampleRate
				p := ph[j] - math.Floor(ph[j])
				s += 2*p - 1
			}
			a := min(1, t/attack)
			if t > dur {
				a *= math.Max(0, 1-(t-dur)/release)
			}
			if trem > 0 {
				a *= 0.6 + 0.4*math.Abs(fsin(trem*t/2))
			}
			return lp.do(s) * a * vel * 0.22
		})
	}
}

// choir is a dark "ah": a pad through two vowel formants.
func choir(r *rand.Rand, f, dur, vel float64) []float64 {
	x := pad(r, f, dur, vel)
	f1, f2 := newFilter(bandpass, 700, 1.4), newFilter(bandpass, 1150, 1.6)
	for i, v := range x {
		x[i] = (f1.do(v)*1.6 + f2.do(v) + v*0.3) * 0.9
	}
	return x
}

// --- tracks -----------------------------------------------------------------

// composeJungleBoss: war drums in the deep jungle, 132 beats a minute.
func composeJungleBoss(r *rand.Rand) *score {
	Dm, Eb, C, Bb := chord{50, minorT}, chord{51, major}, chord{48, major}, chord{46, major}
	A, Gm := chord{57, major}, chord{55, minorT}
	bars := []chord{Dm, Dm, Eb, Dm, Dm, Bb, C, A, Gm, Gm, Eb, Eb, Dm, Bb, Eb, A, Bb, Bb, C, C, Dm, Eb, A, A}
	s := &score{bpm: 132, beats: float64(len(bars)) * 4}

	// The brass call: a phrygian figure, answered.
	call := []motifNote{{0, 0.75, 62, 0.8}, {0.75, 0.25, 63, 0.7}, {1, 1, 62, 0.75}, {2, 0.5, 60, 0.7}, {2.5, 0.5, 58, 0.7}, {3, 1, 57, 0.8}}
	answer := []motifNote{{0, 0.5, 57, 0.7}, {0.5, 0.5, 58, 0.7}, {1, 0.5, 60, 0.75}, {1.5, 0.5, 62, 0.8}, {2, 2, 63, 0.85}}
	for _, b := range []int{2, 6, 10, 14, 18, 22} {
		phrase := call
		if b%4 == 2 && b >= 6 {
			phrase = answer
		}
		for _, n := range phrase {
			s.add(float64(b*4)+n.beat, n.dur, n.midi, n.vel*0.55, 0.15, brass)
			s.add(float64(b*4)+n.beat, n.dur, n.midi-12, n.vel*0.35, -0.15, brass)
		}
	}
	for b, ch := range bars {
		root := ch.root - 12
		// Low strings, sixteenths on the root, the fifth on the offbeats.
		for k := range 16 {
			note := root
			if k%4 == 3 {
				note = root + 7
			}
			v := 0.32
			if k%4 == 0 {
				v = 0.42
			}
			s.add(float64(b*4)+float64(k)*0.25, 0.24, note, v, -0.25, bowed(0))
		}
		s.add(float64(b*4), 4, root-12, 0.4, 0, bass)
		// War drums: a low tom on the beats, a higher one answering, a flam
		// into each fourth bar.
		for k := range 8 {
			beat := float64(b*4) + float64(k)*0.5
			switch {
			case k%2 == 0:
				s.add(beat, 0.5, 38, 0.7, -0.1, tom)
			case k == 3 || k == 7:
				s.add(beat, 0.5, 45, 0.5, 0.25, tom)
			}
		}
		if b%4 == 3 {
			for k := range 4 {
				s.add(float64(b*4)+3+float64(k)*0.25, 0.25, 43+k, 0.35+0.1*float64(k), 0.1, tom)
			}
		}
		s.add(float64(b*4), 1, 0, 0.55, 0, kick)
		s.add(float64(b*4)+2.5, 1, 0, 0.45, 0, kick)
		for k := range 8 {
			s.add(float64(b*4)+float64(k)*0.5+0.25, 0.2, 0, 0.18, 0.45, shaker)
		}
		if b >= 8 { // the second half: the choir comes in
			for _, t := range ch.tones(57, 72)[:3] {
				s.add(float64(b*4), 4, t, 0.16, 0.3, choir)
			}
		}
	}
	return clearSeam(s)
}

// composeSeaBoss: something vast in the deep, 96 beats a minute.
func composeSeaBoss(r *rand.Rand) *score {
	Cm, Ab, Fm, G := chord{48, minorT}, chord{44, major}, chord{53, minorT}, chord{55, major}
	Eb, Db := chord{51, major}, chord{49, major}
	bars := []chord{Cm, Cm, Ab, G, Cm, Fm, Db, G, Fm, Fm, Ab, Eb, Cm, Db, G, G}
	s := &score{bpm: 96, beats: float64(len(bars)) * 4}

	theme := []motifNote{{0, 1.5, 60, 0.8}, {1.5, 0.5, 63, 0.7}, {2, 2, 62, 0.75}, {4, 1, 60, 0.7}, {5, 1, 58, 0.7}, {6, 2, 56, 0.85}}
	for _, b := range []int{4, 12} {
		for _, n := range theme {
			s.add(float64(b*4)+n.beat, n.dur, n.midi, n.vel*0.5, 0.1, brass)
			s.add(float64(b*4)+n.beat, n.dur, n.midi-12, n.vel*0.4, -0.1, bowed(0))
		}
	}
	for b, ch := range bars {
		root := ch.root - 12
		// The pulse in the deep: eighths, accented in threes.
		for k := range 8 {
			v := 0.38
			if k == 0 || k == 3 || k == 6 {
				v = 0.55
			}
			s.add(float64(b*4)+float64(k)*0.5, 0.45, root-12, v, 0, bass)
		}
		for _, t := range ch.tones(55, 70)[:3] {
			s.add(float64(b*4), 4, t, 0.22, -0.3, bowed(12))
			s.add(float64(b*4), 4, t, 0.14, 0.3, choir)
		}
		// Brass stabs on the downbeat of every other bar.
		if b%2 == 0 {
			for _, t := range ch.tones(48, 62)[:3] {
				s.add(float64(b*4), 0.6, t, 0.38, 0.2, brass)
			}
		}
		s.add(float64(b*4), 1, 0, 0.6, 0, kick)
		s.add(float64(b*4)+1.5, 1, 0, 0.35, 0, kick)
		if b%4 == 3 { // timpani rolling into the next phrase
			for k := range 8 {
				s.add(float64(b*4)+2+float64(k)*0.25, 0.25, 36, 0.25+0.06*float64(k), 0, tom)
			}
		}
	}
	return clearSeam(s)
}

// genBossSFX adds the fanfare played when a boss falls.
func genBossSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0xb055, 0x51))
	x := make([]float64, samples(3.2))
	for _, n := range []struct {
		at, dur float64
		midi    int
	}{{0, 0.22, 60}, {0.25, 0.22, 64}, {0.5, 0.22, 67}, {0.75, 1.9, 72}} {
		addAt(x, brass(r, midiFreq(float64(n.midi)), n.dur, 0.8), n.at, 1)
		addAt(x, brass(r, midiFreq(float64(n.midi-12)), n.dur, 0.5), n.at, 1)
	}
	for _, m := range []int{48, 55, 60} {
		addAt(x, bowed(0)(r, midiFreq(float64(m)), 2.0, 0.6), 0.75, 1)
	}
	addAt(x, tom(r, midiFreq(38), 0, 0.9), 0.75, 1)
	add("boss_win", 0.9, x)
}

// motifNote is a note of a theme, played by the track in its own voices.
type motifNote struct {
	beat, dur float64
	midi      int
	vel       float64
}

// fallen is which bosses this player's map shows as fallen, seen last frame.
var fallen = map[sim.CreatureKind]bool{}

// bossWinSound plays the fanfare when a boss this player has seen falls.
func (g *Game) bossWinSound() {
	for _, m := range g.curr.BossMarks {
		if m.Defeated && !fallen[m.Kind] {
			if _, known := fallen[m.Kind]; known { // fell just now, not at loading
				snd.play("boss_win", 0.8, 0, 1)
			}
		}
		fallen[m.Kind] = m.Defeated
	}
}

// clearSeam moves the notes on the very first beat a hair later, so no drum
// stroke straddles the loop's seam (a note's timing wobbles a little).
func clearSeam(s *score) *score {
	for i := range s.notes {
		s.notes[i].beat = max(s.notes[i].beat, 0.03)
	}
	return s
}
