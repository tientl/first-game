package client

import (
	"math"
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The calls of the great beasts: the tyrant's roar, the long hoot of the
// long-necks, the bellows of the horned herds, the barking of raptor packs.
// Beasts call now and then on their own, so the world is heard before it is
// seen, and roar when they set off after prey or bring it down. A roar
// carries much further than other sounds: far off it comes muffled, with its
// echo.

// --- synthesis ------------------------------------------------------------------

// voiceSpec describes one call from a beast's throat.
type voiceSpec struct {
	dur           float64
	pitch         func(u float64) float64    // Hz along the call, u from 0 to 1
	amp           func(u float64) float64    // loudness along the call
	rough, rattle float64                    // depth (0 to 1) and rate (Hz) of the throat's rattle
	formants      [2]float64                 // the two resonances of the mouth and throat
	vowel         func(u float64) [2]float64 // if set, the formants along the call: a jaw opening and closing
	breath, sub   float64                    // shares of breath noise and of a sub-octave
	harmonics     int
}

// beastVoice renders a call: harmonics of a gliding, wavering pitch, rattled
// by the throat, shaped through two formants, with breath and a sub-octave.
func beastVoice(r *rand.Rand, v voiceSpec) []float64 {
	fa, fb := newFilter(bandpass, v.formants[0], 1.1), newFilter(bandpass, v.formants[1], 1.4)
	breath := newFilter(bandpass, v.formants[0]*1.6, 0.6)
	// One cycle of the harmonics, summed once: the sum for every sample
	// made a long roar slow to build.
	const n = 4096
	cycle := make([]float64, n+1)
	for j := range cycle {
		for k := 1; k <= v.harmonics; k++ {
			cycle[j] += fsin(float64(j*k)/n) / float64(k)
		}
	}
	ph, rp, jit := 0.0, 0.0, 0.0
	return render(v.dur, func(t float64, i int) float64 {
		u := t / v.dur
		if i%300 == 0 {
			jit = (r.Float64()*2 - 1) * 0.02 // the voice is never quite steady
		}
		if v.vowel != nil && i%64 == 0 {
			f := v.vowel(u)
			fa.set(bandpass, f[0], 1.1)
			fb.set(bandpass, f[1], 1.4)
			breath.set(bandpass, f[0]*1.6, 0.6)
		}
		f := v.pitch(u) * (1 + 0.025*fsin(t*4.7) + jit)
		ph += f / sampleRate
		p := (ph - math.Floor(ph)) * n
		j := int(p)
		s := cycle[j] + (cycle[j+1]-cycle[j])*(p-float64(j))
		s += v.sub * fsin(ph*0.5)
		rp += v.rattle * (1 + 0.3*fsin(t*2.3)) / sampleRate
		rattle := 1 - v.rough*0.5*(1+fsin(rp))
		s *= rattle
		n := breath.do(r.Float64()*2-1) * v.breath * (0.6 + 0.4*rattle)
		return (fa.do(s)*1.4 + fb.do(s) + s*0.2 + n) * v.amp(u)
	})
}

// glide is a pitch passing through the points {u, Hz} in order, straight
// between them.
func glide(pts ...[2]float64) func(u float64) float64 {
	return func(u float64) float64 {
		if u <= pts[0][0] {
			return pts[0][1]
		}
		for i := 1; i < len(pts); i++ {
			if a, b := pts[i-1], pts[i]; u <= b[0] {
				return a[1] + (b[1]-a[1])*(u-a[0])/(b[0]-a[0])
			}
		}
		return pts[len(pts)-1][1]
	}
}

// swell rises over the first share of a call and dies over the last, both
// along a quarter sine.
func swell(attack, release float64) func(u float64) float64 {
	return func(u float64) float64 {
		a := math.Sin(math.Pi / 2 * min(1, u/attack))
		if u > 1-release {
			a *= math.Sin(math.Pi / 2 * max(0, (1-u)/release))
		}
		return a
	}
}

// echo adds reflections {delay seconds, gain} of x, each duller than the
// sound itself, as off rock and trees.
func echo(x []float64, taps ...[2]float64) []float64 {
	last := 0.0
	for _, tp := range taps {
		last = max(last, tp[0])
	}
	out := make([]float64, len(x)+samples(last))
	copy(out, x)
	for _, tp := range taps {
		lp := newFilter(lowpass, 1600, 0.7)
		o := samples(tp[0])
		for i, v := range x {
			out[o+i] += lp.do(v) * tp[1]
		}
	}
	return out
}

// muffle is x heard from far away: only its low end, and more echo.
func muffle(x []float64, cutoff float64) []float64 {
	a, b := newFilter(lowpass, cutoff, 0.7), newFilter(lowpass, cutoff, 0.7)
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = b.do(a.do(v))
	}
	return echo(out, [2]float64{0.21, 0.45}, [2]float64{0.43, 0.3}, [2]float64{0.7, 0.16})
}

// genRoarSFX adds the beasts' calls to the effect bank. It has a random
// source of its own, so the sounds made before it stay as they were.
func genRoarSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0x60a2, 0x2a))
	genPredatorSFX(add) // the tyrant's roar and the hunters' voices: see predsound.go
	// The long-necks: a long, deep, hollow hoot rising and falling, like a
	// horn blown in a valley.
	for _, k := range []float64{1, 0.88} {
		x := beastVoice(r, voiceSpec{
			dur: 2.8, pitch: glide([2]float64{0, 50 * k}, [2]float64{0.45, 74 * k}, [2]float64{1, 44 * k}),
			amp: swell(0.3, 0.45), rough: 0.06, rattle: 18, formants: [2]float64{170 * k, 360 * k},
			breath: 0.12, sub: 0.7, harmonics: 8,
		})
		add("hoot", 1, echo(x, [2]float64{0.18, 0.3}, [2]float64{0.4, 0.15}))
	}
	// A raptor pack barking to itself: two or three short, harsh chirps.
	for _, v := range []struct {
		n int
		k float64
	}{{2, 1}, {3, 1}, {2, 1.15}} {
		x := make([]float64, samples(0.2*float64(v.n)+0.1))
		for i := range v.n {
			k := v.k * (1 + 0.06*float64(i%2))
			addAt(x, beastVoice(r, voiceSpec{
				dur: 0.14, pitch: glide([2]float64{0, 620 * k}, [2]float64{0.35, 980 * k}, [2]float64{1, 560 * k}),
				amp: swell(0.15, 0.5), rough: 0.55, rattle: 70, formants: [2]float64{1300 * k, 2500 * k},
				breath: 0.45, harmonics: 8,
			}), 0.19*float64(i), 1-0.12*float64(i))
		}
		add("chirp", 0.9, x)
	}
	// Anzu: deep booms from the chest, like a cassowary's.
	for _, k := range []float64{1, 0.9} {
		x := make([]float64, samples(1.3))
		for i := range 3 {
			addAt(x, beastVoice(r, voiceSpec{
				dur: 0.32, pitch: glide([2]float64{0, 78 * k}, [2]float64{1, 66 * k}),
				amp: swell(0.2, 0.6), rough: 0.15, rattle: 12, formants: [2]float64{220 * k, 480 * k},
				breath: 0.1, sub: 0.6, harmonics: 8,
			}), 0.42*float64(i), 1-0.15*float64(i))
		}
		add("boom", 1, x)
	}
	// Quetzalcoatlus: a hoarse croak, once or twice.
	for _, n := range []int{1, 2} {
		x := make([]float64, samples(1.05))
		for i := range n {
			d := 0.55 - 0.12*float64(i)
			addAt(x, beastVoice(r, voiceSpec{
				dur: d, pitch: glide([2]float64{0, 175}, [2]float64{0.3, 190}, [2]float64{1, 120}),
				amp: swell(0.08, 0.5), rough: 0.9, rattle: 48, formants: [2]float64{520, 1150},
				breath: 0.7, sub: 0.2, harmonics: 12,
			}), 0.5*float64(i), 1-0.2*float64(i))
		}
		add("croak", 0.9, x)
	}
	// More bellows for the horned herds, so a herd is not one voice.
	for _, f := range [][2]float64{{135, 100}, {104, 80}} {
		add("bellow", 0.8, growl(r, 1.1, f[0], f[1], 0.12, 0.5, 0.3, [2]float64{250, 500}))
	}
}

// --- playing --------------------------------------------------------------------

// callAt plays a call placed in the world at the given pitch, heard up to
// reach tiles away (further than most sounds).
func callAt(name string, x, y, vol, rate, reach float64) {
	if snd == nil {
		return
	}
	dx, dy := x-snd.lx, (y-snd.ly)*1.3
	d := math.Hypot(dx, dy)
	if d >= reach {
		return
	}
	a := 1 - d/reach
	snd.play(name, vol*a*a, dx/8, rate*(1+(snd.rng.Float64()*2-1)*0.04))
}

// roarAt is the tyrant's roar from (x, y): close by in full, from further
// off muffled, up to the edge of what a player can know of.
func roarAt(x, y, vol float64) { rexCryAt("rex_roar", x, y, vol) }

// rexCryAt is the tyrant crying out with voice name from (x, y): close by
// as it is, from further off its roar muffled by the distance.
func rexCryAt(name string, x, y, vol float64) {
	if snd == nil {
		return
	}
	lastRoar = time.Now()
	d := math.Hypot(x-snd.lx, (y-snd.ly)*1.3)
	roarShake(d)
	if d < hearingRange {
		callAt(name, x, y, vol, 1, roarReach)
		return
	}
	callAt("rex_roar_far", x, y, vol*1.4, 1, roarReach)
}

// roarReach is how far a roar carries, in tiles: about what a client is sent.
const roarReach = 32.0

// lastRoar is when a tyrant last roared, so its growl does not talk over it.
var lastRoar time.Time

// --- calling on their own ---------------------------------------------------------

// callVoice is how a kind of beast calls on its own.
type callVoice struct {
	name      string
	vol, rate float64
	reach     float64       // tiles
	every     time.Duration // on average, between one beast's calls
	gap       time.Duration // at least, between calls of this voice
	pose      time.Duration // how long it holds its calling pose, if it has one
}

func callOf(k sim.CreatureKind) (callVoice, bool) {
	s := time.Second
	switch k {
	case sim.CreatureRex:
		return callVoice{"rex_roar", 0.9, 1, roarReach, 28 * s, 9 * s, 3400 * time.Millisecond}, true
	case sim.CreatureBrachio:
		return callVoice{"brachio_call", 0.75, 1, 28, 26 * s, 7 * s, 0}, true
	case sim.CreatureTrike:
		return callVoice{"trike_call", 0.6, 1, 22, 22 * s, 4 * s, 1100 * time.Millisecond}, true
	case sim.CreatureToro:
		return callVoice{"toro_call", 0.65, 1, 22, 22 * s, 4 * s, 1100 * time.Millisecond}, true
	case sim.CreatureStego:
		return callVoice{"stego_call", 0.5, 1, 20, 30 * s, 4 * s, 0}, true
	case sim.CreatureAnkylo: // a low grunting call, recorded; see realvoice.go and ankylo.go
		return callVoice{"ankylo_call", 0.55, 1, 20, 28 * s, 5 * s, 900 * time.Millisecond}, true
	case sim.CreatureThesc:
		return callVoice{"thesc_call", 0.4, 1, 16, 30 * s, 4 * s, 0}, true // see creaturevoice.go
	case sim.CreatureMammoth:
		return callVoice{"mammoth_call", 0.65, 1, 22, 26 * s, 6 * s, 1200 * time.Millisecond}, true
	case sim.CreatureRaptor:
		return callVoice{"raptor_call", 0.45, 1, 16, 16 * s, 3 * s, 0}, true
	case sim.CreatureAchero:
		return callVoice{"achero_call", 0.4, 1, 16, 16 * s, 3 * s, 0}, true
	case sim.CreatureAnzu:
		return callVoice{"anzu_call", 0.55, 1, 18, 30 * s, 5 * s, 1300 * time.Millisecond}, true
	case sim.CreatureQuetzal:
		return callVoice{"quetzal_call", 0.5, 1, 18, 28 * s, 5 * s, 0}, true
	case sim.CreatureOrnitho:
		return callVoice{"ornitho_call", 0.35, 1, 16, 30 * s, 5 * s, 0}, true
	case sim.CreatureSabre:
		return callVoice{"sabre_roar", 0.4, 1, 22, 34 * s, 8 * s, 0}, true
	case sim.CreatureMonkey: // a troop hooting and chattering in the trees
		return callVoice{"monkey_chatter", 0.4, 1, 18, 14 * s, 3 * s, 0}, true
	case sim.CreatureTroodon: // the night hunters calling in the dark; see troodonsound.go
		return callVoice{"troodon_call", 0.5, 1, 24, 10 * s, 3 * s, 0}, true
	}
	return moreCalls(k) // the bear, the spider, the scorpion, the flying reptile: see creaturevoice.go
}

// callGap is the least time between any two beasts calling on their own, so
// the calls come one at a time.
const callGap = 2500 * time.Millisecond

// beastCall is one beast calling.
type beastCall struct {
	c     protocol.CreatureView
	voice callVoice
}

// chorus decides which beasts call, and remembers who is calling.
type chorus struct {
	last   time.Time                    // the last time it was asked
	any    time.Time                    // the last call of any beast
	byName map[string]time.Time         // the last call of each voice
	until  map[sim.CreatureID]time.Time // beasts holding their calling pose, until when
}

func newChorus() *chorus {
	return &chorus{byName: map[string]time.Time{}, until: map[sim.CreatureID]time.Time{}}
}

// beasts is the chorus of the game's beasts.
var beasts = newChorus()

// pick is the beast that calls now, if any, from those in cur heard at the
// listener (lx, ly). roll is a random number in [0, 1).
func (ch *chorus) pick(cur protocol.Snapshot, lx, ly float64, now time.Time, roll func() float64) (beastCall, bool) {
	dt := now.Sub(ch.last)
	first := ch.last.IsZero()
	ch.last = now
	if first || dt <= 0 || now.Sub(ch.any) < callGap {
		return beastCall{}, false
	}
	dt = min(dt, 500*time.Millisecond)
	for _, c := range cur.Creatures {
		v, ok := callOf(c.Kind)
		if !ok || c.Ridden || (c.State != sim.CreatureIdle && c.State != sim.CreatureWander) {
			continue
		}
		if math.Hypot(float64(c.X)-lx, (float64(c.Y)-ly)*1.3) >= v.reach {
			continue
		}
		every := v.every
		if c.Tamed {
			every *= 2 // a pet at the player's heels would call all day
		}
		if roll() >= float64(dt)/float64(every) || now.Sub(ch.byName[v.name]) < v.gap {
			continue
		}
		ch.any, ch.byName[v.name] = now, now
		if v.pose > 0 && c.State == sim.CreatureIdle {
			ch.until[c.ID] = now.Add(v.pose)
		}
		if len(ch.until) > 64 {
			for id, t := range ch.until {
				if now.After(t) {
					delete(ch.until, id)
				}
			}
		}
		return beastCall{c, v}, true
	}
	return beastCall{}, false
}

// calling reports whether c is standing calling, to be drawn so.
func (ch *chorus) calling(c protocol.CreatureView, now time.Time) bool {
	t, ok := ch.until[c.ID]
	return ok && c.State == sim.CreatureIdle && now.Before(t)
}

// beastCalls lets a beast near the player call now and then.
func (g *Game) beastCalls(cur protocol.Snapshot) {
	if snd == nil {
		return // and nobody is drawn calling unheard, so screenshots stay the same
	}
	call, ok := beasts.pick(cur, snd.lx, snd.ly, time.Now(), rand.Float64)
	if !ok {
		return
	}
	x, y := float64(call.c.X), float64(call.c.Y)
	if call.voice.name == "rex_roar" {
		roarAt(x, y, call.voice.vol)
		return
	}
	callAt(call.voice.name, x, y, call.voice.vol, call.voice.rate, call.voice.reach)
}

// calling reports whether c is standing calling on its own just now.
func calling(c protocol.CreatureView) bool { return beasts.calling(c, time.Now()) }
