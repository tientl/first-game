package client

import (
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The voices of the great beasts, and when they are heard.

// growl is a buzzing voice: harmonics of a gliding pitch, roughened by fast
// tremolo, shaped through two formant filters, with breath noise mixed in.
func growl(r *rand.Rand, dur, f0, f1, attack, release, rough float64, formants [2]float64) []float64 {
	fa, fb := newFilter(bandpass, formants[0], 1.2), newFilter(bandpass, formants[1], 1.5)
	breath := newFilter(lowpass, formants[1]*1.5, 0.7)
	ph := 0.0
	return render(dur, func(t float64, _ int) float64 {
		u := t / dur
		f := f0 + (f1-f0)*u
		f *= 1 + 0.03*fsin(t*5.3) // a slow waver
		ph += f / sampleRate
		v := 0.0
		for k := 1.0; k <= 10; k++ {
			v += fsin(ph*k) / k
		}
		v *= 1 - rough*0.5*(1+fsin(t*(28+8*fsin(t*3)))) // a rattling throat
		v += breath.do(r.Float64()*2-1) * 0.6
		out := fa.do(v)*1.4 + fb.do(v)
		a := min(1, t/attack)
		if rem := dur - t; rem < release {
			a *= rem / release
		}
		return out * a
	})
}

// genDinoSFX adds the beasts' sounds to the effect bank.
func genDinoSFX(r *rand.Rand, add func(name string, peak float64, x []float64)) {
	// The tyrant's roar and the other beasts' calls: see roar.go.
	genRoarSFX(add)
	// The grazers' low bellow.
	add("bellow", 0.8, growl(r, 1.0, 120, 92, 0.12, 0.45, 0.25, [2]float64{260, 520}))
	{ // a raptor's screech: a harsh, chirping shriek
		x := growl(r, 0.42, 900, 620, 0.02, 0.15, 0.6, [2]float64{1800, 3200})
		addAt(x, tone(0.3, 2400, 1500, 0.25, 0.005, 0.08), 0.02, 0.4)
		add("screech", 0.85, x)
	}
	{ // a flying reptile's cry as it dives
		x := growl(r, 0.7, 1300, 800, 0.03, 0.3, 0.4, [2]float64{2200, 3600})
		addAt(x, noiseBurst(r, 0.5, bandpass, 2600, 2, 0.01, 0.15), 0, 0.3)
		add("shriek", 0.8, x)
	}
	{ // a heavy footfall
		x := make([]float64, samples(0.4))
		addAt(x, tone(0.35, 70, 38, 0.12, 0.003, 0.1), 0, 1)
		addAt(x, noiseBurst(r, 0.2, lowpass, 320, 0.7, 0.002, 0.05), 0, 0.7)
		add("stomp", 1, x)
	}
	{ // a blow landing on a thick hide
		x := make([]float64, samples(0.3))
		addAt(x, tone(0.22, 160, 90, 0.06, 0.002, 0.05), 0, 1)
		addAt(x, noiseBurst(r, 0.1, lowpass, 1400, 0.8, 0.001, 0.025), 0, 0.7)
		add("whack", 1, x)
	}
}

// bigBeast reports whether k is one of the great beasts, which shake the
// ground and take a thump to the hide.
func bigBeast(k sim.CreatureKind) bool {
	switch k {
	case sim.CreatureTrike, sim.CreatureStego, sim.CreatureBrachio, sim.CreatureRex:
		return true
	}
	return false
}

// dinoSound plays what a beast did when its state changed from o to c, and
// reports whether c is a beast it knows about.
func dinoSound(o, c protocol.CreatureView) bool {
	x, y := float64(c.X), float64(c.Y)
	switch c.Kind {
	case sim.CreatureRex:
		switch {
		case c.State == sim.CreatureRoar: // it has seen a player
			roarAt(x, y, 1)
		case c.State == sim.CreatureChase && calm(o.State): // off after prey
			if snd.ready("rexhunt", 4*time.Second) {
				roarAt(x, y, 0.85)
			}
		case c.State == sim.CreatureEat && (o.State == sim.CreatureChase || o.State == sim.CreatureStrike):
			roarAt(x, y, 0.75) // over its kill
		case c.State == sim.CreatureHurt:
			if snd.ready("rexhurt", 3*time.Second) {
				rexCryAt("rex_hurt", x, y, 0.75) // a short, broken roar
			}
		case c.State == sim.CreatureStrike:
			snd.playAt("chomp", x, y, 1, 0.04)
		}
	case sim.CreatureTrike, sim.CreatureStego, sim.CreatureBrachio:
		switch {
		case c.State == sim.CreatureChase && o.State != sim.CreatureStrike,
			c.State == sim.CreatureFlee && o.State != sim.CreatureHurt:
			if snd.ready("bellow", 700*time.Millisecond) {
				snd.playAt(alarmOf(c.Kind), x, y, 0.8, 0.12)
			}
		case c.State == sim.CreatureStrike:
			snd.playAt("whack", x, y, 0.9, 0.08)
		}
	case sim.CreatureRaptor:
		switch {
		case c.State == sim.CreatureChase && (o.State == sim.CreatureIdle || o.State == sim.CreatureWander):
			if snd.ready("screech", 350*time.Millisecond) {
				callAt("raptor_alarm", x, y, 0.75, 1, 18)
			}
		case c.State == sim.CreatureStrike:
			snd.playAt("raptor_attack", x, y, 0.5, 0.15)
		case c.State == sim.CreatureEat && o.State == sim.CreatureChase:
			callAt("raptor_call", x, y, 0.5, 1, 16) // the pack calls over its kill
		}
	case sim.CreaturePtero:
		if c.State == sim.CreatureChase {
			snd.playAt("ptero_alarm", x, y, 0.7, 0.1)
		}
	case sim.CreaturePlesio:
		switch {
		case o.State == sim.CreatureLurk:
			snd.playAt("splash", x, y, 0.6, 0.08) // rearing up
		case c.State == sim.CreatureLurk:
			snd.playAt("splash", x, y, 0.35, 0.12) // back under
		case c.State == sim.CreatureStrike:
			snd.playAt("chomp", x, y, 0.8, 0.08)
		}
	default:
		return false
	}
	return true
}

// calm reports whether a beast in state s was going about its own business:
// not chasing, fighting, fleeing or hurt.
func calm(s sim.CreatureState) bool {
	switch s {
	case sim.CreatureIdle, sim.CreatureWander, sim.CreatureEat, sim.CreatureDrink:
		return true
	}
	return false
}

// beastSteps thumps the ground under a tyrant walking near the player.
func (g *Game) beastSteps(cur protocol.Snapshot) {
	for _, c := range cur.Creatures {
		if c.Kind != sim.CreatureRex || c.State == sim.CreatureRoar || c.State == sim.CreatureStrike {
			continue
		}
		q, ok := g.prevCreatures[c.ID]
		if !ok || (q.X == c.X && q.Y == c.Y) {
			continue
		}
		gap := 700 * time.Millisecond
		if c.State == sim.CreatureChase {
			gap = 380 * time.Millisecond
		}
		if snd.ready("stomp", gap) {
			snd.playAt("stomp", float64(c.X), float64(c.Y), 0.9, 0.1)
		}
	}
}

// alarmOf is the bellow of a great grazer of kind k charging or bolting.
func alarmOf(k sim.CreatureKind) string {
	switch k {
	case sim.CreatureStego:
		return "stego_alarm"
	case sim.CreatureBrachio:
		return "brachio_alarm"
	}
	return "trike_alarm"
}
