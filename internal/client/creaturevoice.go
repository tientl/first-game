package client

import (
	"math"
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Every creature has a voice. Most of the great beasts' calls are made in
// roar.go, predsound.go, troodonsound.go and iceart.go, and heard when a
// beast calls on its own or turns on someone (dinosound.go, hellwire.go).
// Here are the rest: the cries of beasts that are hit and of beasts that
// die, a herd calling in alarm when one of it is hurt, and the voices of the
// cave bear, the spider, the scorpion, the monkeys and the monsters of the
// deep sea. The tribes' voices are in tribesound.go. Bigger animals have
// lower, longer voices; every voice is placed in the world, and a beast's
// cries are rate-limited for each beast and for each voice.

// --- the sounds -------------------------------------------------------------------

// goVoiceSFX makes the voices here and in tribesound.go on a goroutine of
// their own, alongside the rest of the effects, so they cost start-up
// nothing; the bank comes down the channel when it is done.
func goVoiceSFX() <-chan map[string][][]float32 {
	ch := make(chan map[string][][]float32, 1)
	go func() {
		s := make(map[string][][]float32)
		add := func(name string, peak float64, x []float64) { s[name] = append(s[name], finish(x, peak)) }
		genVoiceSFX(add)
		ch <- s
	}()
	return ch
}

// genVoiceSFX adds the voices made here, with a random source of their own.
func genVoiceSFX(add func(name string, peak float64, x []float64)) {
	r := rand.New(rand.NewPCG(0x701ce, 0x5))
	genTribeSFX(r, add)

	// A big grazer hit: a short bellow that jumps up and breaks.
	for _, k := range []float64{1, 0.9, 1.1} {
		x := beastVoice(r, voiceSpec{
			dur: 0.7, pitch: glide(pt{0, 130 * k}, pt{0.15, 200 * k}, pt{1, 95 * k}), amp: swell(0.05, 0.5),
			rough: 0.7, rattle: 30, vowel: vowel([3]float64{0, 520 * k, 900 * k}, [3]float64{1, 300 * k, 620 * k}),
			breath: 0.4, sub: 0.5, harmonics: 14,
		})
		add("bellow_hurt", 0.9, saturate(x, 1.7))
	}
	// A middling grazer's honk, through the hollow of its snout: once or twice.
	for _, n := range []int{1, 2, 1} {
		x := make([]float64, samples(0.42*float64(n)+0.1))
		for i := range n {
			k := 1 + 0.07*float64(i) + 0.05*float64(len(x)%3)
			addAt(x, beastVoice(r, voiceSpec{
				dur: 0.36, pitch: glide(pt{0, 150 * k}, pt{0.3, 178 * k}, pt{1, 138 * k}), amp: swell(0.1, 0.4),
				rough: 0.15, rattle: 20, formants: [2]float64{420 * k, 900 * k}, breath: 0.15, sub: 0.25, harmonics: 10,
			}), 0.42*float64(i), 1-0.15*float64(i))
		}
		add("honk", 0.85, echo(x, pt{0.15, 0.2}))
	}
	// A great beast dying: a long groan that sinks and runs out of breath.
	for _, k := range []float64{1, 0.88} {
		x := make([]float64, samples(2.6))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 2.2, pitch: glide(pt{0, 120 * k}, pt{0.15, 135 * k}, pt{0.7, 75 * k}, pt{1, 50 * k}),
			amp:   func(u float64) float64 { return swell(0.08, 0.5)(u) * (1 - 0.45*u) },
			rough: 0.6, rattle: 21, vowel: vowel([3]float64{0, 600 * k, 1000 * k}, [3]float64{0.6, 420 * k, 800 * k}, [3]float64{1, 280 * k, 560 * k}),
			breath: 0.8, sub: 0.6, harmonics: 14,
		}), 0, 1)
		addAt(x, noiseBurst(r, 0.9, lowpass, 900, 0.7, 0.2, 0.25), 1.6, 0.35) // the last breath
		add("beast_death", 0.9, echo(saturate(x, 1.4), pt{0.18, 0.25}, pt{0.4, 0.12}))
	}
	// A raptor hit: a short, cracking squawk.
	for _, k := range []float64{1, 1.1, 0.92} {
		x := beastVoice(r, voiceSpec{
			dur: 0.3, pitch: glide(pt{0, 800 * k}, pt{0.25, 1350 * k}, pt{1, 650 * k}), amp: swell(0.06, 0.5),
			rough: 0.6, rattle: 80, formants: [2]float64{1500 * k, 2900 * k}, breath: 0.4, harmonics: 8,
		})
		add("raptor_yelp", 0.85, saturate(x, 1.8))
	}

	// The cave bear: a deep growl as it wakes to someone, a roar as it rises.
	for _, k := range []float64{1, 0.9} {
		x := beastVoice(r, voiceSpec{
			dur: 1.5, pitch: glide(pt{0, 52 * k}, pt{0.4, 62 * k}, pt{1, 46 * k}), amp: swell(0.3, 0.3),
			rough: 1, rattle: 15, formants: [2]float64{210 * k, 500 * k}, breath: 0.55, sub: 0.8, harmonics: 16,
		})
		add("bear_growl", 0.9, saturate(x, 1.6))
	}
	for _, k := range []float64{1, 0.92} {
		x := make([]float64, samples(1.6))
		addAt(x, noiseBurst(r, 0.3, bandpass, 900, 0.8, 0.03, 0.1), 0, 0.3) // the breath drawn in
		addAt(x, beastVoice(r, voiceSpec{
			dur: 1.4, pitch: glide(pt{0, 95 * k}, pt{0.25, 140 * k}, pt{1, 72 * k}), amp: swell(0.08, 0.5),
			rough: 0.9, rattle: 24, vowel: vowel([3]float64{0, 450 * k, 850 * k}, [3]float64{0.3, 650 * k, 1050 * k}, [3]float64{1, 380 * k, 700 * k}),
			breath: 0.7, sub: 0.5, harmonics: 14,
		}), 0.1, 1)
		add("bear_roar", 0.95, echo(saturate(x, 2.1), pt{0.12, 0.25}, pt{0.28, 0.12}))
	}

	// A cave spider: the chitter of its mouthparts over a dry hiss.
	for v := range 3 {
		dur := 0.55 + 0.1*float64(v)
		x := noiseBurst(r, dur, highpass, 4200, 0.7, 0.05, dur*0.4)
		for i := range x {
			x[i] *= 0.25 * (0.6 + 0.4*fsin(float64(i)/sampleRate*19))
		}
		per := 1 / (26 + 8*r.Float64())
		for t := 0.01; t < dur-0.03; t += per * (0.7 + 0.6*r.Float64()) {
			a := 0.5 + 0.5*math.Sin(math.Pi*t/dur)
			addAt(x, noiseBurst(r, 0.012, bandpass, 3300+r.Float64()*2200, 4, 0.0002, 0.0025), t, a)
		}
		add("spider_chitter", 0.8, x)
	}
	// A scorpion: the clack of its claws and the rattle of its armour.
	for v := range 3 {
		x := make([]float64, samples(0.55))
		addAt(x, grains(r, 0.4, 260, 0.004, bandpass, 2500, 1.5, func(t float64) float64 { return env(t, 0.03, 0.12) }), 0.05, 0.35)
		t := 0.0
		for c := range 4 + v {
			f := 1700 + r.Float64()*900
			addAt(x, noiseBurst(r, 0.03, bandpass, f, 5, 0.0002, 0.004), t, 1-0.08*float64(c))
			addAt(x, partials(0.04, f*0.55, []float64{1, 2.3}, []float64{1, 0.4}, []float64{0.008, 0.004}), t, 0.4)
			t += 0.045 + r.Float64()*0.06
			if c == 1 {
				t += 0.08 // a pause between two snaps
			}
		}
		add("scorpion_click", 0.8, x)
	}
	{ // and its sting going in: a quick whip through the air and a click
		flt := newFilter(bandpass, 1500, 1.5)
		x := render(0.28, func(t float64, _ int) float64 {
			flt.set(bandpass, 1500+t*14000, 1.5)
			return flt.do(r.Float64()*2-1) * env(t, 0.06, 0.05)
		})
		addAt(x, noiseBurst(r, 0.03, bandpass, 3000, 4, 0.0002, 0.004), 0.12, 0.9)
		add("sting", 0.8, x)
	}

	// Monkeys: "hoo-hoo-ha-ha-HA!", a small throat chattering, and a scream.
	for v := range 3 {
		n := 4 + v
		x := make([]float64, samples(0.15*float64(n)+0.3))
		t := 0.0
		for i := range n {
			open := float64(i) / float64(n-1)
			vw := vU
			if open > 0.4 {
				vw = vA
			}
			f := 420 + 300*open + 30*r.Float64()
			d := 0.09 + 0.03*r.Float64()
			addAt(x, humanVoice(r, cry{dur: d, tract: 1.55,
				pitch:  glide(pt{0, f * 0.9}, pt{0.4, f * 1.15}, pt{1, f}),
				vowel:  vowels3([]float64{0, 1}, vw, vw),
				amp:    swell(0.2, 0.4),
				breath: glide(pt{0, 0.5}, pt{0.3, 0.1}, pt{1, 0.2}), strain: 0.3,
			}), t, 0.6+0.4*open)
			t += d + 0.03 + 0.03*r.Float64()
		}
		add("monkey_chatter", 0.8, x)
	}
	for _, k := range []float64{1, 1.1} {
		x := humanVoice(r, cry{dur: 0.35, tract: 1.7,
			pitch: scaled(k, pt{0, 900}, pt{0.2, 1400}, pt{1, 1000}),
			vowel: vowels3([]float64{0, 1}, vI, vE), amp: swell(0.05, 0.4),
			breath: glide(pt{0, 0.3}, pt{1, 0.3}), strain: 0.6,
		})
		add("monkey_screech", 0.85, saturate(x, 1.6))
	}

	// The Mosasaurus: a deep rumble under the water, bubbles rising through it.
	for _, k := range []float64{1, 0.85} {
		x := make([]float64, samples(2.5))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 2.2, pitch: glide(pt{0, 30 * k}, pt{0.4, 42 * k}, pt{1, 26 * k}), amp: swell(0.25, 0.4),
			rough: 0.85, rattle: 9, formants: [2]float64{110 * k, 260 * k}, breath: 0.3, sub: 1, harmonics: 16,
		}), 0, 1)
		addAt(x, noiseBurst(r, 2.2, lowpass, 300, 0.7, 0.4, 0.8), 0, 0.5) // the water churning
		for range 12 {
			f := 160 + r.Float64()*260
			addAt(x, tone(0.07, f, f*1.9, 0.05, 0.002, 0.02), 0.2+r.Float64()*1.9, 0.15)
		}
		add("mosa_rumble", 0.9, echo(saturate(x, 1.4), pt{0.25, 0.3}, pt{0.55, 0.15}))
	}
	// The Kraken: a wet, wavering groan from the deep, gurgling as it goes.
	for _, k := range []float64{1, 0.9} {
		x := make([]float64, samples(2.4))
		addAt(x, beastVoice(r, voiceSpec{
			dur: 2.1, pitch: glide(pt{0, 70 * k}, pt{0.3, 58 * k}, pt{0.55, 80 * k}, pt{1, 50 * k}), amp: swell(0.2, 0.4),
			rough: 0.7, rattle: 11,
			vowel:  vowel([3]float64{0, 260 * k, 620 * k}, [3]float64{0.35, 420 * k, 900 * k}, [3]float64{0.7, 300 * k, 700 * k}, [3]float64{1, 240 * k, 560 * k}),
			breath: 0.5, sub: 0.5, harmonics: 14,
		}), 0, 1)
		addAt(x, grains(r, 2.1, 50, 0.03, lowpass, 700, 0.9, func(t float64) float64 { return 0.5 + 0.5*math.Sin(math.Pi*t/2.1) }), 0, 0.6)
		for range 16 {
			f := 220 + r.Float64()*500
			addAt(x, tone(0.06, f, f*1.8, 0.04, 0.002, 0.015), r.Float64()*2.1, 0.18)
		}
		add("kraken_groan", 0.9, echo(saturate(x, 1.5), pt{0.2, 0.3}, pt{0.45, 0.15}))
	}
	genAnkyloSFX(add) // with a random source of its own: see ankylo.go
}

// --- who says what ----------------------------------------------------------------

// voiceUse is one voice played for a creature, at a volume and pitch.
type voiceUse struct {
	name      string
	vol, rate float64
}

// kindVoices is what a kind of creature says, beyond what the other files
// already play for it.
type kindVoices struct {
	alarm voiceUse // it turns on someone, or bolts, from going about its business
	hurt  voiceUse // it is hit
	herd  voiceUse // one of its herd near it is hit: an answering call
	death voiceUse // it dies
}

// say is voice name at a volume and a pitch.
func say(name string, vol, rate float64) voiceUse { return voiceUse{name, vol, rate} }

// voicesOf is the voices of kind k. Sizes set pitch: the bigger the beast,
// the lower the rate.
func voicesOf(k sim.CreatureKind) kindVoices {
	switch k {
	case sim.CreatureRabbit:
		return kindVoices{alarm: say("squeak", 0.25, 1.3), hurt: say("squeak", 0.45, 1.25)}
	case sim.CreatureCroc:
		return kindVoices{hurt: say("croc_hurt", 0.5, 1), death: say("croc_death", 0.6, 1)}
	case sim.CreatureBrachy: // the smaller crocodiles, higher: see hellwire.go
		return kindVoices{hurt: say("croc_hurt", 0.5, 1.15), death: say("croc_death", 0.6, 1.15)}
	case sim.CreatureChampso:
		return kindVoices{hurt: say("croc_hurt", 0.5, 1.35), death: say("croc_death", 0.6, 1.35)}
	case sim.CreatureTrike:
		return kindVoices{hurt: say("trike_hurt", 0.85, 1), herd: say("trike_call", 0.7, 1.06), death: say("trike_death", 0.85, 1)}
	case sim.CreatureToro:
		return kindVoices{hurt: say("toro_hurt", 0.9, 1), herd: say("toro_call", 0.75, 1.06), death: say("toro_death", 0.9, 1)}
	case sim.CreatureStego:
		return kindVoices{hurt: say("stego_hurt", 0.8, 1), herd: say("stego_call", 0.6, 1.06), death: say("stego_death", 0.85, 1)}
	case sim.CreatureAnkylo: // recorded (realvoice.go); its guard grunt is synthesised in ankylo.go
		return kindVoices{hurt: say("ankylo_hurt", 0.85, 1), herd: say("ankylo_call", 0.6, 1.06), death: say("ankylo_death", 0.9, 1)}
	case sim.CreatureBrachio:
		return kindVoices{hurt: say("brachio_hurt", 0.9, 1), herd: say("brachio_call", 0.75, 1.08), death: say("brachio_death", 0.95, 1)}
	case sim.CreatureThesc:
		return kindVoices{hurt: say("thesc_hurt", 0.6, 1), herd: say("thesc_call", 0.5, 1.06), death: say("thesc_death", 0.6, 1)}
	case sim.CreatureOrnitho:
		return kindVoices{hurt: say("ornitho_hurt", 0.5, 1), herd: say("ornitho_call", 0.4, 1.06), death: say("ornitho_death", 0.5, 1)}
	case sim.CreatureMammoth:
		return kindVoices{herd: say("mammoth_call", 0.65, 1.06), death: say("mammoth_death", 0.9, 1)} // hit, it trumpets as it turns: iceart.go
	case sim.CreatureRex:
		return kindVoices{death: say("rex_death", 1, 1)} // hit, it roars: dinosound.go
	case sim.CreatureRaptor:
		return kindVoices{hurt: say("raptor_hurt", 0.6, 1), death: say("raptor_death", 0.65, 1)}
	case sim.CreatureAchero:
		return kindVoices{hurt: say("achero_hurt", 0.5, 1), death: say("achero_death", 0.55, 1)}
	case sim.CreatureTroodon:
		return kindVoices{hurt: say("troodon_hurt", 0.5, 1), death: say("troodon_death", 0.55, 1)}
	case sim.CreaturePtero:
		return kindVoices{hurt: say("ptero_hurt", 0.6, 1), death: say("ptero_death", 0.65, 1)}
	case sim.CreatureQuetzal:
		return kindVoices{hurt: say("quetzal_hurt", 0.6, 1), death: say("quetzal_death", 0.7, 1)}
	case sim.CreatureAnzu:
		return kindVoices{hurt: say("anzu_hurt", 0.55, 1), death: say("anzu_death", 0.6, 1)}
	case sim.CreatureSabre:
		return kindVoices{hurt: say("sabre_hurt", 0.7, 1), death: say("sabre_death", 0.7, 1)}
	case sim.CreaturePlesio:
		return kindVoices{alarm: say("plesio_alarm", 0.45, 1), hurt: say("plesio_hurt", 0.5, 1), death: say("plesio_death", 0.7, 1)}
	case sim.CreatureSpider:
		return kindVoices{alarm: say("spider_chitter", 0.55, 1), hurt: say("spider_chitter", 0.5, 1.3), death: say("spider_chitter", 0.5, 0.75)}
	case sim.CreatureBear:
		return kindVoices{alarm: say("bear_roar", 0.85, 1), hurt: say("bear_hurt", 0.7, 1), death: say("bear_death", 0.8, 1)}
	case sim.CreatureScorpion:
		return kindVoices{alarm: say("scorpion_click", 0.55, 1), hurt: say("scorpion_click", 0.5, 1.25), death: say("scorpion_click", 0.5, 0.8)}
	case sim.CreatureMonkey:
		return kindVoices{alarm: say("monkey_screech", 0.5, 1), hurt: say("monkey_screech", 0.55, 1.05), death: say("monkey_screech", 0.5, 0.85)}
	case sim.CreatureToad:
		return kindVoices{hurt: say("croak", 0.4, 1.9), death: say("croak", 0.45, 1.5)}
	case sim.CreatureGlowToad:
		return kindVoices{hurt: say("croak", 0.35, 2.3), death: say("croak", 0.4, 1.9)}
	case sim.CreatureMosa:
		return kindVoices{alarm: say("mosa_call", 0.85, 1), hurt: say("mosa_hurt", 0.6, 1), death: say("mosa_death", 0.85, 1)}
	case sim.CreatureKraken:
		return kindVoices{alarm: say("kraken_call", 0.85, 1), hurt: say("kraken_hurt", 0.6, 1), death: say("kraken_death", 0.85, 1)}
	case sim.CreatureTitanoboa:
		return kindVoices{hurt: say("croc_hiss", 0.6, 0.6), death: say("beast_death", 1, 0.6)}
	case sim.CreatureSpino:
		return kindVoices{hurt: say("spino_hurt", 0.8, 1), death: say("spino_death", 1, 1)}
	case sim.CreatureMammothKing:
		return kindVoices{hurt: say("mammothking_hurt", 0.7, 1), death: say("mammothking_death", 1, 1)}
	case sim.CreatureCarcharo:
		return kindVoices{hurt: say("carcharo_hurt", 0.8, 1), death: say("carcharo_death", 1, 1)}
	case sim.CreatureGiganoto:
		return kindVoices{hurt: say("giganoto_hurt", 0.8, 1), death: say("giganoto_death", 1, 1)}
	case sim.CreatureMegalodon:
		return kindVoices{hurt: say("megalodon_hurt", 0.7, 1), death: say("megalodon_death", 1, 1)}
	}
	return kindVoices{}
}

// How often a beast's voices may come.
const (
	hurtCool     = 1500 * time.Millisecond // between one beast's cries when hit
	bossHurtCool = 6 * time.Second         // a boss is hit all the time
	alarmCool    = 6 * time.Second         // between one beast's alarms
	herdCool     = 5 * time.Second         // between a herd's alarm calls
	voiceReach   = 20.0                    // tiles a cry carries
	deathReach   = 24.0
	herdNear     = 9.0 // tiles: herd mates this near answer
)

// --- memory ---------------------------------------------------------------------

// queuedCall is a voice waiting to be heard: an answer a moment after a cry.
type queuedCall struct {
	at                     time.Time
	id                     sim.CreatureID // who calls; heard from where he is then
	name                   string
	x, y, vol, rate, reach float64
}

type coolKey struct {
	id    sim.CreatureID
	event string
}

// voiceMemory is what the creatures' voices remember between snapshots.
type voiceMemory struct {
	now     func() time.Time
	roll    func() float64 // a random number in [0, 1)
	cool    map[coolKey]time.Time
	lastVar map[string]int                 // the variant of each voice heard last
	engaged map[sim.CreatureID]time.Time   // warriors: the last time each cried out or fought
	queue   []queuedCall                   // answers still to come
	lastSea time.Time                      // the last time the sea monsters were asked to call
	sea     map[sim.CreatureKind]time.Time // the last call of each sea monster
}

func newVoiceMemory() *voiceMemory {
	return &voiceMemory{
		now:     time.Now,
		roll:    rand.Float64,
		cool:    map[coolKey]time.Time{},
		lastVar: map[string]int{},
		engaged: map[sim.CreatureID]time.Time{},
		sea:     map[sim.CreatureKind]time.Time{},
	}
}

// voices is the memory of the game's creature voices.
var voices = newVoiceMemory()

// cooled reports whether creature id may make sound event again after gap,
// and if so notes that it does now.
func (vm *voiceMemory) cooled(id sim.CreatureID, event string, gap time.Duration) bool {
	now := vm.now()
	k := coolKey{id, event}
	if t, ok := vm.cool[k]; ok && now.Sub(t) < gap {
		return false
	}
	vm.cool[k] = now
	if len(vm.cool) > 512 {
		for k, t := range vm.cool {
			if now.Sub(t) > time.Minute {
				delete(vm.cool, k)
			}
		}
	}
	return true
}

// ready is cooled for a voice, whoever makes it: no creature has ID 0.
func (vm *voiceMemory) ready(name string, gap time.Duration) bool {
	return vm.cooled(0, "v:"+name, gap)
}

// later queues a voice to be heard at t.
func (vm *voiceMemory) later(t time.Time, id sim.CreatureID, name string, x, y, vol, rate, reach float64) {
	if len(vm.queue) < 16 {
		vm.queue = append(vm.queue, queuedCall{t, id, name, x, y, vol, rate, reach})
	}
}

// flush plays the queued voices that are due, from where their creatures
// are in cur.
func (vm *voiceMemory) flush(cur protocol.Snapshot) {
	now := vm.now()
	keep := vm.queue[:0]
	for _, q := range vm.queue {
		if now.Before(q.at) {
			keep = append(keep, q)
			continue
		}
		if now.Sub(q.at) > 2*time.Second {
			continue // the game stopped a while: too late to answer
		}
		for _, c := range cur.Creatures {
			if c.ID == q.id {
				q.x, q.y = float64(c.X), float64(c.Y)
			}
		}
		vm.callPick(q.name, q.x, q.y, q.vol, q.rate, q.reach)
	}
	vm.queue = keep
}

// callPick is callAt, choosing a variant of the voice other than the one
// heard last, so no voice comes twice the same in a row.
func (vm *voiceMemory) callPick(name string, x, y, vol, rate, reach float64) {
	if snd == nil {
		return
	}
	vs := snd.sfx[name]
	if len(vs) == 0 {
		return
	}
	if reach <= 0 {
		reach = voiceReach
	}
	dx, dy := x-snd.lx, (y-snd.ly)*1.3
	d := math.Hypot(dx, dy)
	if d >= reach {
		return
	}
	i := snd.rng.IntN(len(vs))
	if last, ok := vm.lastVar[name]; ok && len(vs) > 1 && i == last {
		i = (i + 1 + snd.rng.IntN(len(vs)-1)) % len(vs)
	}
	vm.lastVar[name] = i
	a := 1 - d/reach
	snd.playVariant(name, i, vol*a*a, dx/8, rate*(1+(snd.rng.Float64()*2-1)*0.03))
}

// playVariant is play with variant i of the sound chosen.
func (s *soundSystem) playVariant(name string, i int, vol, pan, rate float64) {
	rate *= speedOf(name)
	if s == nil || vol <= 0.001 || !(rate > 0.01 && rate < 100) || i < 0 || i >= len(s.sfx[name]) {
		return
	}
	if s.onPlay != nil {
		s.onPlay(name)
	}
	s.mix.add(&voice{
		data: s.sfx[name][i], rate: rate, gain: float32(vol), target: float32(vol),
		pan: float32(max(-1, min(1, pan))), bus: busSFX,
	})
}

// use plays u for c, if this beast and this voice are both ready. A herd
// or a troop all bolting at once is one alarm, not one each.
func (vm *voiceMemory) use(u voiceUse, c protocol.CreatureView, event string, gap time.Duration, reach float64) bool {
	vgap := 250 * time.Millisecond
	if event == "alarm" {
		vgap = 1500 * time.Millisecond
	}
	if u.name == "" || !vm.cooled(c.ID, event, gap) || !vm.ready(u.name, vgap) {
		return false
	}
	vm.callPick(u.name, float64(c.X), float64(c.Y), u.vol, u.rate, reach)
	return true
}

// --- when they are heard ------------------------------------------------------------

// voiceSounds plays the creatures' voices for what changed between old and
// cur: cries when hit, alarms, herds answering, death cries, the tribes,
// and the deep sea's monsters calling. It runs after the other creature
// sounds (creatures.go).
func (g *Game) voiceSounds(old, cur protocol.Snapshot) {
	if snd == nil {
		return
	}
	vm := voices
	vm.flush(cur)
	was := make(map[sim.CreatureID]protocol.CreatureView, len(old.Creatures))
	for _, c := range old.Creatures {
		was[c.ID] = c
	}
	for _, c := range cur.Creatures {
		o, seen := was[c.ID]
		if !seen {
			continue
		}
		delete(was, c.ID)
		hit := (c.HP < o.HP && c.HP > 0) || (c.State == sim.CreatureHurt && o.State != sim.CreatureHurt)
		if c.Kind == sim.CreatureTribesman {
			vm.tribeSounds(o, c, hit, cur)
			continue
		}
		vm.beastCry(o, c, hit, cur)
	}
	vm.keepEngaged(cur)
	vm.deathCries(old, cur, was)
	vm.seaCalls(cur)
}

// beastCry plays what beast c says for what it did since o.
func (vm *voiceMemory) beastCry(o, c protocol.CreatureView, hit bool, cur protocol.Snapshot) {
	kv := voicesOf(c.Kind)
	x, y := float64(c.X), float64(c.Y)
	if hit {
		gap := hurtCool
		if c.Kind.Boss() {
			gap = bossHurtCool
		}
		if vm.use(kv.hurt, c, "hurt", gap, voiceReach) {
			// Its cry says it: the bellow a grazer gives as it turns
			// would only repeat it (dinosound.go, hellwire.go).
			snd.ready("bellow", 0)
		}
		// The herd's alarm is rate-limited for the whole kind: the
		// "creature" of the cool-down key is the kind itself.
		if kv.herd.name != "" && vm.cooled(sim.CreatureID(c.Kind), "herd", herdCool) {
			vm.herdAnswer(c, kv.herd, cur)
		}
		return
	}
	if o.State == c.State {
		return
	}
	switch c.Kind {
	case sim.CreatureMosa, sim.CreaturePlesio:
		// Rearing up out of the water: a rumble from the deep.
		if o.State == sim.CreatureLurk && c.State != sim.CreatureLurk {
			if c.Kind == sim.CreatureMosa {
				snd.playAt("splash", x, y, 0.75, 0.06)
			}
			vm.use(kv.alarm, c, "alarm", alarmCool, 26)
		}
		if c.Kind == sim.CreatureMosa {
			switch {
			case c.State == sim.CreatureStrike:
				snd.playAt("chomp", x, y, 0.9, 0.05)
			case c.State == sim.CreatureLurk:
				snd.playAt("splash", x, y, 0.35, 0.12) // back under
			}
		}
	case sim.CreatureKraken:
		switch {
		case c.State == sim.CreatureGrab:
			snd.playAt("splash", x, y, 0.8, 0.05)
			vm.use(kv.alarm, c, "alarm", alarmCool, 26)
		case o.State == sim.CreatureGrab:
			snd.playAt("splash", x, y, 0.5, 0.1) // it lets go and sinks
		}
	case sim.CreatureSpider:
		switch {
		case c.State == sim.CreatureChase && calm(o.State):
			vm.use(kv.alarm, c, "alarm", alarmCool, voiceReach)
		case c.State == sim.CreatureSpit:
			snd.playAt("hiss", x, y, 0.45, 0.15)
		case c.State == sim.CreatureStrike:
			snd.playAt("chomp", x, y, 0.4, 0.1)
		}
	case sim.CreatureBear:
		switch {
		case c.State == sim.CreatureChase && calm(o.State):
			vm.use(kv.alarm, c, "alarm", alarmCool, 24)
		case c.State == sim.CreatureClaw:
			snd.playAt("whack", x, y, 0.75, 0.1)
			if vm.cooled(c.ID, "growl", 2500*time.Millisecond) {
				callAt("bear_growl", x, y, 0.55, 1.1, 16)
			}
		case c.State == sim.CreatureStrike:
			snd.playAt("chomp", x, y, 0.85, 0.08)
		}
	case sim.CreatureScorpion:
		switch {
		case c.State == sim.CreatureChase && calm(o.State):
			vm.use(kv.alarm, c, "alarm", alarmCool, 18)
		case c.State == sim.CreatureClaw:
			snd.playAt("pinch", x, y, 0.65, 0.1)
		case c.State == sim.CreatureStrike:
			snd.playAt("sting", x, y, 0.6, 0.1)
		}
	case sim.CreatureMonkey, sim.CreatureRabbit:
		if c.State == sim.CreatureFlee && calm(o.State) && !c.Tamed {
			vm.use(kv.alarm, c, "alarm", alarmCool, 16)
		}
	}
}

// herdAnswer has a beast of c's herd near it call out in alarm, a moment
// after c is hit.
func (vm *voiceMemory) herdAnswer(c protocol.CreatureView, u voiceUse, cur protocol.Snapshot) {
	best, bd := protocol.CreatureView{}, herdNear
	for _, o := range cur.Creatures {
		if o.Kind != c.Kind || o.ID == c.ID || o.Tamed || !calm(o.State) {
			// One already charging or bolting is heard doing so.
			continue
		}
		if d := math.Hypot(float64(o.X-c.X), float64(o.Y-c.Y)); d < bd {
			best, bd = o, d
		}
	}
	if best.ID == 0 {
		return
	}
	delay := time.Duration((0.35 + 0.35*vm.roll()) * float64(time.Second))
	vm.later(vm.now().Add(delay), best.ID, u.name, float64(best.X), float64(best.Y), u.vol, u.rate, 24)
}

// deathCries plays the last cry of each creature in gone, those of old not
// in cur, that died: a body of its kind fell where it was, or it was badly
// hurt and vanished right by a player.
func (vm *voiceMemory) deathCries(old, cur protocol.Snapshot, gone map[sim.CreatureID]protocol.CreatureView) {
	if len(gone) == 0 {
		return
	}
	had := make(map[sim.EntityID]bool, len(old.Entities))
	for _, e := range old.Entities {
		had[e.ID] = true
	}
	var bodies []protocol.EntityView
	for _, e := range cur.Entities {
		if e.Kind == sim.EntityCorpse && !had[e.ID] {
			bodies = append(bodies, e)
		}
	}
	for _, c := range gone {
		x, y := float64(c.X), float64(c.Y)
		died := false
		for i, b := range bodies {
			if sim.CreatureKind(b.Var) == c.Kind && math.Hypot(float64(b.TX)+0.5-x, float64(b.TY)+0.5-y) < 3.5 {
				died = true
				bodies = append(bodies[:i], bodies[i+1:]...)
				break
			}
		}
		if !died && c.MaxHP > 0 && float64(c.HP) <= 0.3*float64(c.MaxHP) && nearAPlayer(cur.Players, x, y, c.Kind.Reach()+1.5) {
			died = true
		}
		if !died {
			continue
		}
		if c.Kind == sim.CreatureTribesman {
			vm.callPick("tribe_death", x, y, 0.85, manRate(c.ID), deathReach)
			continue
		}
		if u := voicesOf(c.Kind).death; u.name != "" {
			vm.callPick(u.name, x, y, u.vol, u.rate, deathReach)
		}
	}
}

// seaCall is how a sea monster lurking under the water calls on its own.
var seaCall = map[sim.CreatureKind]callVoice{
	sim.CreatureMosa:   {name: "mosa_call", vol: 0.5, rate: 1, reach: 22, every: 30 * time.Second, gap: 12 * time.Second},
	sim.CreatureKraken: {name: "kraken_call", vol: 0.45, rate: 1, reach: 20, every: 36 * time.Second, gap: 12 * time.Second},
}

// seaCalls lets a monster lurking in the deep near the player rumble or
// groan now and then. Calls on land and at sea share the one gap between
// any two calls (roar.go), so they come one at a time.
func (vm *voiceMemory) seaCalls(cur protocol.Snapshot) {
	now := vm.now()
	dt := now.Sub(vm.lastSea)
	first := vm.lastSea.IsZero()
	vm.lastSea = now
	if first || dt <= 0 || now.Sub(beasts.any) < callGap {
		return
	}
	dt = min(dt, 500*time.Millisecond)
	for _, c := range cur.Creatures {
		v, ok := seaCall[c.Kind]
		if !ok || c.State != sim.CreatureLurk || now.Sub(vm.sea[c.Kind]) < v.gap {
			continue
		}
		x, y := float64(c.X), float64(c.Y)
		if math.Hypot(x-snd.lx, (y-snd.ly)*1.3) >= v.reach || vm.roll() >= float64(dt)/float64(v.every) {
			continue
		}
		beasts.any, vm.sea[c.Kind] = now, now
		vm.callPick(v.name, x, y, v.vol, v.rate, v.reach)
		return
	}
}

// moreCalls is how the creatures not in callOf call on their own (roar.go).
func moreCalls(k sim.CreatureKind) (callVoice, bool) {
	s := time.Second
	switch k {
	case sim.CreatureBear:
		return callVoice{"bear_growl", 0.45, 1, 16, 30 * s, 8 * s, 0}, true
	case sim.CreatureSpider:
		return callVoice{"spider_chitter", 0.3, 1, 12, 24 * s, 5 * s, 0}, true
	case sim.CreatureScorpion:
		return callVoice{"scorpion_click", 0.3, 1, 12, 24 * s, 5 * s, 0}, true
	case sim.CreaturePtero:
		return callVoice{"ptero_call", 0.35, 1, 22, 30 * s, 6 * s, 0}, true
	}
	return callVoice{}, false
}
