package client

import (
	"math"
	"math/rand/v2"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// The voices of the woodland tribes (sim/tribe.go). A warrior who sees a
// stranger gives a war cry, a man's yell pushed hard; the warriors near him
// take it up one after another with short shouts. He grunts as he thrusts
// his spear, cries out when he is hit and gives a long, falling cry as he
// dies. Every warrior has a voice of his own, a little higher or lower
// than the next man's, and no two war cries in a row are the same.

// --- a human voice ----------------------------------------------------------

// cry describes one call from a human throat, or a monkey's.
type cry struct {
	dur    float64
	pitch  func(u float64) float64    // Hz along the cry, u from 0 to 1
	vowel  func(u float64) [3]float64 // the first three formants along the cry
	amp    func(u float64) float64    // loudness of the voice along the cry
	breath func(u float64) float64    // loudness of breath along the cry: an "h", a gasp
	strain float64                    // 0 to 1: a throat pushed hard, rough with a sub-octave
	tract  float64                    // formants scaled: 1 a man, more for a smaller throat
}

// humanVoice renders a cry: a buzzing glottal pulse at a gliding, wavering
// pitch through three formants that move as the mouth shapes the vowels,
// with breath through the same mouth.
func humanVoice(r *rand.Rand, c cry) []float64 {
	if c.tract == 0 {
		c.tract = 1
	}
	top := 0.0
	for i := 0; i <= 64; i++ {
		top = max(top, c.pitch(float64(i)/64))
	}
	h := max(3, min(30, int(9000/top))) // no harmonic past the band, so nothing aliases
	const n = 2048
	cycle := make([]float64, n+1)
	for k := 1; k <= h; k++ {
		a := math.Pow(float64(k), -0.85) // falling gently: a bright, shouting voice
		for j := range cycle {
			cycle[j] += fsin(float64(j*k)/n) * a
		}
	}
	fa, fb, fc := newFilter(bandpass, 700, 5), newFilter(bandpass, 1200, 7), newFilter(bandpass, 2600, 9)
	asp := newFilter(highpass, 450, 0.7)
	broad := newFilter(bandpass, 1500, 0.8)
	ph, jit, jitTo := 0.0, 0.0, 0.0
	return render(c.dur, func(t float64, i int) float64 {
		u := t / c.dur
		if i%32 == 0 {
			f := c.vowel(u)
			fa.set(bandpass, f[0]*c.tract, 5)
			fb.set(bandpass, f[1]*c.tract, 7)
			fc.set(bandpass, f[2]*c.tract, 9)
			broad.set(bandpass, f[1]*c.tract, 0.8)
		}
		if i%220 == 0 {
			jitTo = (r.Float64()*2 - 1) * 0.01 // a voice is never quite steady
		}
		jit += (jitTo - jit) * 0.002
		ph += c.pitch(u) * (1 + 0.012*fsin(t*5.5) + jit) / sampleRate
		p := (ph - math.Floor(ph)) * n
		j := int(p)
		s := cycle[j] + (cycle[j+1]-cycle[j])*(p-float64(j))
		if c.strain > 0 {
			s *= 1 - c.strain*0.45*(1+fsin(ph*0.5)) // every other cycle weaker: a hoarse, strained yell
		}
		s *= c.amp(u)
		b := 0.0
		if c.breath != nil {
			b = asp.do(r.Float64()*2-1) * c.breath(u)
		}
		src := s + b*3
		return fa.do(src) + fb.do(src)*0.75 + fc.do(src)*0.4 + broad.do(b)*0.2
	})
}

// The vowels of a man shouting: his first three formants, in Hz.
var (
	vA  = [3]float64{850, 1250, 2600} // "a" in father
	vO  = [3]float64{560, 900, 2450}  // "o"
	vU  = [3]float64{380, 820, 2350}  // "oo"
	vE  = [3]float64{560, 1850, 2600} // "e" in hey
	vI  = [3]float64{330, 2250, 3000} // "ee"
	vUh = [3]float64{640, 1200, 2450} // "u" in hut
)

// vowels3 is the formants passing through the vowels at the given points
// along a cry, straight between them.
func vowels3(at []float64, vs ...[3]float64) func(u float64) [3]float64 {
	var f [3]func(float64) float64
	for k := range 3 {
		var ps [][2]float64
		for i, u := range at {
			ps = append(ps, [2]float64{u, vs[i][k]})
		}
		f[k] = glide(ps...)
	}
	return func(u float64) [3]float64 { return [3]float64{f[0](u), f[1](u), f[2](u)} }
}

// scaled is glide with every value times k.
func scaled(k float64, pts ...[2]float64) func(u float64) float64 {
	g := glide(pts...)
	return func(u float64) float64 { return g(u) * k }
}

// genTribeSFX adds the warriors' voices to the effect bank.
func genTribeSFX(r *rand.Rand, add func(name string, peak float64, x []float64)) {
	// War cries: six yells, each from a different man, pushed hard and
	// ringing back off the trees.
	warcries := []func(k float64) cry{
		func(k float64) cry { // "Ha-yaaah!"
			return cry{dur: 1.0,
				pitch:  scaled(k, pt{0, 190}, pt{0.12, 230}, pt{0.3, 200}, pt{0.4, 260}, pt{0.7, 330}, pt{1, 250}),
				vowel:  vowels3([]float64{0, 0.28, 0.36, 0.48, 1}, vA, vA, vI, vA, vUh),
				amp:    glide(pt{0, 0}, pt{0.06, 0.9}, pt{0.28, 1}, pt{0.34, 0.45}, pt{0.42, 1}, pt{0.85, 0.9}, pt{1, 0}),
				breath: glide(pt{0, 0.9}, pt{0.07, 0.12}, pt{0.8, 0.1}, pt{1, 0.3}), strain: 0.35}
		},
		func(k float64) cry { // "Ooo-aaah!"
			return cry{dur: 1.1,
				pitch:  scaled(k, pt{0, 170}, pt{0.3, 200}, pt{0.45, 290}, pt{0.8, 320}, pt{1, 220}),
				vowel:  vowels3([]float64{0, 0.35, 0.5, 1}, vU, vO, vA, vA),
				amp:    glide(pt{0, 0}, pt{0.08, 0.8}, pt{0.4, 0.9}, pt{0.5, 1}, pt{0.85, 0.95}, pt{1, 0}),
				breath: glide(pt{0, 0.3}, pt{0.1, 0.1}, pt{1, 0.25}), strain: 0.3}
		},
		func(k float64) cry { // "Hey-ya-haaa!"
			return cry{dur: 1.05,
				pitch:  scaled(k, pt{0, 240}, pt{0.2, 300}, pt{0.3, 260}, pt{0.4, 320}, pt{0.55, 280}, pt{0.62, 300}, pt{0.9, 350}, pt{1, 260}),
				vowel:  vowels3([]float64{0, 0.2, 0.27, 0.35, 0.5, 0.56, 0.62, 1}, vE, vE, vI, vA, vA, vUh, vA, vA),
				amp:    glide(pt{0, 0}, pt{0.05, 0.9}, pt{0.24, 0.9}, pt{0.28, 0.5}, pt{0.34, 1}, pt{0.52, 0.9}, pt{0.56, 0.1}, pt{0.63, 1}, pt{0.9, 0.9}, pt{1, 0}),
				breath: glide(pt{0, 0.8}, pt{0.05, 0.1}, pt{0.52, 0.1}, pt{0.56, 0.9}, pt{0.62, 0.12}, pt{1, 0.25}), strain: 0.4}
		},
		func(k float64) cry { // "Ay-ay-ay-ay-ay!": the tongue beating, a ululation
			const dur, beat = 1.2, 7.0
			yo := func(u float64) float64 { return 0.5 + 0.5*math.Cos(twoPi*beat*u*dur) }
			return cry{dur: dur,
				pitch: func(u float64) float64 { return k * (290 + 50*min(1, u*4)) * (1 + 0.05*yo(u)) },
				vowel: func(u float64) [3]float64 {
					y := yo(u)
					return [3]float64{vA[0] - (vA[0]-vE[0])*y, vA[1] + (vI[1]-vA[1])*y*0.8, vA[2] + 300*y}
				},
				amp:    func(u float64) float64 { return swell(0.05, 0.2)(u) * (0.5 + 0.5*(1-yo(u)*0.8)) },
				breath: glide(pt{0, 0.4}, pt{0.05, 0.1}, pt{1, 0.2}), strain: 0.2}
		},
		func(k float64) cry { // "Huh! Haaaah!"
			return cry{dur: 1.15,
				pitch:  scaled(k, pt{0, 200}, pt{0.2, 190}, pt{0.3, 250}, pt{0.6, 320}, pt{1, 260}),
				vowel:  vowels3([]float64{0, 0.22, 0.3, 1}, vUh, vUh, vA, vA),
				amp:    glide(pt{0, 0}, pt{0.03, 0.9}, pt{0.17, 0.8}, pt{0.21, 0}, pt{0.3, 0}, pt{0.36, 1}, pt{0.9, 0.85}, pt{1, 0}),
				breath: glide(pt{0, 0.9}, pt{0.04, 0.2}, pt{0.2, 0.2}, pt{0.23, 0}, pt{0.29, 0.9}, pt{0.36, 0.15}, pt{1, 0.3}), strain: 0.45}
		},
		func(k float64) cry { // "Yaaaah!"
			return cry{dur: 0.9,
				pitch:  scaled(k, pt{0, 260}, pt{0.25, 360}, pt{0.7, 330}, pt{1, 230}),
				vowel:  vowels3([]float64{0, 0.18, 0.3, 1}, vI, vE, vA, vA),
				amp:    glide(pt{0, 0}, pt{0.05, 0.8}, pt{0.3, 1}, pt{0.85, 0.9}, pt{1, 0}),
				breath: glide(pt{0, 0.2}, pt{0.1, 0.08}, pt{1, 0.3}), strain: 0.5}
		},
	}
	for i, w := range warcries {
		k := []float64{1, 0.9, 1.08, 0.95, 1.04, 0.92}[i]
		x := saturate(humanVoice(r, w(k)), 1.3)
		add("tribe_warcry", 0.9, echo(x, pt{0.13, 0.22}, pt{0.29, 0.1}))
	}

	// Answering shouts: short, as one warrior after another takes up the cry.
	shouts := []cry{
		{dur: 0.4, pitch: glide(pt{0, 220}, pt{0.3, 280}, pt{1, 210}), // "Ho!"
			vowel:  vowels3([]float64{0, 1}, vO, vO),
			amp:    glide(pt{0, 0}, pt{0.12, 1}, pt{0.7, 0.8}, pt{1, 0}),
			breath: glide(pt{0, 0.9}, pt{0.12, 0.15}, pt{1, 0.3}), strain: 0.35},
		{dur: 0.45, pitch: glide(pt{0, 240}, pt{0.35, 310}, pt{1, 250}), // "Hai!"
			vowel:  vowels3([]float64{0, 0.5, 1}, vA, vA, vI),
			amp:    glide(pt{0, 0}, pt{0.1, 1}, pt{0.75, 0.8}, pt{1, 0}),
			breath: glide(pt{0, 0.8}, pt{0.1, 0.1}, pt{1, 0.2}), strain: 0.3},
		{dur: 0.55, pitch: glide(pt{0, 260}, pt{0.35, 300}, pt{0.45, 250}, pt{0.6, 330}, pt{1, 260}), // "Ya-ha!"
			vowel:  vowels3([]float64{0, 0.12, 0.4, 0.48, 0.55, 1}, vI, vA, vA, vUh, vA, vA),
			amp:    glide(pt{0, 0}, pt{0.06, 1}, pt{0.38, 0.9}, pt{0.46, 0.1}, pt{0.55, 1}, pt{0.85, 0.8}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{0.42, 0.1}, pt{0.47, 0.9}, pt{0.55, 0.1}, pt{1, 0.3}), strain: 0.4},
		{dur: 0.5, pitch: glide(pt{0, 200}, pt{0.4, 180}, pt{0.55, 230}, pt{1, 200}), // "Hup! Hup!"
			vowel:  vowels3([]float64{0, 1}, vUh, vUh),
			amp:    glide(pt{0, 0}, pt{0.05, 1}, pt{0.3, 0.7}, pt{0.38, 0}, pt{0.52, 0}, pt{0.57, 1}, pt{0.85, 0.6}, pt{1, 0}),
			breath: glide(pt{0, 0.9}, pt{0.06, 0.2}, pt{0.38, 0.1}, pt{0.5, 0.9}, pt{0.58, 0.2}, pt{1, 0.4}), strain: 0.5},
		{dur: 0.55, pitch: glide(pt{0, 210}, pt{0.4, 240}, pt{0.55, 280}, pt{1, 330}), // "Oh-hoo!"
			vowel:  vowels3([]float64{0, 0.4, 0.55, 1}, vO, vO, vU, vU),
			amp:    glide(pt{0, 0}, pt{0.08, 0.9}, pt{0.4, 0.8}, pt{0.47, 0.2}, pt{0.55, 1}, pt{0.9, 0.8}, pt{1, 0}),
			breath: glide(pt{0, 0.3}, pt{0.45, 0.1}, pt{0.5, 0.8}, pt{0.56, 0.1}, pt{1, 0.2}), strain: 0.3},
	}
	for _, c := range shouts {
		add("tribe_shout", 0.85, echo(saturate(humanVoice(r, c), 1.3), pt{0.12, 0.18}))
	}

	// Grunts of effort as a spear goes in: a hard, short breath out.
	for _, k := range []float64{1, 0.88, 1.1, 0.95} {
		add("tribe_grunt", 0.8, humanVoice(r, cry{dur: 0.22 + 0.04*(k-0.88)/0.22,
			pitch:  scaled(k, pt{0, 150}, pt{0.25, 160}, pt{1, 110}),
			vowel:  vowels3([]float64{0, 0.6, 1}, vUh, vUh, vU),
			amp:    glide(pt{0, 0}, pt{0.1, 1}, pt{0.6, 0.55}, pt{1, 0}),
			breath: glide(pt{0, 0.6}, pt{0.3, 0.35}, pt{1, 0.6}), strain: 0.6}))
	}

	// Cries of pain: sudden, high, breaking.
	pains := []cry{
		{dur: 0.45, pitch: glide(pt{0, 300}, pt{0.15, 380}, pt{1, 210}), // "Agh!"
			vowel:  vowels3([]float64{0, 0.6, 1}, vA, vA, vUh),
			amp:    glide(pt{0, 0}, pt{0.04, 1}, pt{0.5, 0.7}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{0.7, 0.3}, pt{1, 0.7}), strain: 0.55},
		{dur: 0.4, pitch: glide(pt{0, 260}, pt{0.2, 320}, pt{1, 200}), // "Uhh!"
			vowel:  vowels3([]float64{0, 1}, vU, vUh),
			amp:    glide(pt{0, 0}, pt{0.05, 1}, pt{0.5, 0.6}, pt{1, 0}),
			breath: glide(pt{0, 0.3}, pt{1, 0.6}), strain: 0.6},
		{dur: 0.55, pitch: glide(pt{0, 330}, pt{0.2, 400}, pt{0.4, 280}, pt{0.5, 340}, pt{1, 220}), // "Aah-ah!"
			vowel:  vowels3([]float64{0, 1}, vA, vA),
			amp:    glide(pt{0, 0}, pt{0.04, 1}, pt{0.38, 0.7}, pt{0.44, 0.2}, pt{0.5, 0.8}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{0.44, 0.6}, pt{0.5, 0.2}, pt{1, 0.6}), strain: 0.5},
		{dur: 0.38, pitch: glide(pt{0, 360}, pt{0.2, 420}, pt{1, 270}), // "Eeh!"
			vowel:  vowels3([]float64{0, 1}, vE, vE),
			amp:    glide(pt{0, 0}, pt{0.05, 1}, pt{0.6, 0.6}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{1, 0.6}), strain: 0.45},
	}
	for _, c := range pains {
		add("tribe_pain", 0.85, saturate(humanVoice(r, c), 1.4))
	}

	// The death cry: a long, falling "aaah" that breaks into a last breath.
	deaths := []cry{
		{dur: 1.5, pitch: glide(pt{0, 320}, pt{0.15, 360}, pt{0.6, 220}, pt{1, 110}),
			vowel:  vowels3([]float64{0, 0.5, 0.8, 1}, vA, vA, vO, vU),
			amp:    glide(pt{0, 0}, pt{0.04, 1}, pt{0.6, 0.6}, pt{0.9, 0.2}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{0.5, 0.3}, pt{0.85, 0.6}, pt{1, 0}), strain: 0.4},
		{dur: 1.7, pitch: glide(pt{0, 260}, pt{0.1, 300}, pt{0.55, 190}, pt{1, 95}), // "Ohhh..."
			vowel:  vowels3([]float64{0, 0.6, 1}, vO, vO, vU),
			amp:    glide(pt{0, 0}, pt{0.05, 1}, pt{0.5, 0.65}, pt{0.88, 0.2}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{0.6, 0.3}, pt{0.9, 0.7}, pt{1, 0}), strain: 0.5},
		{dur: 1.45, pitch: glide(pt{0, 340}, pt{0.1, 390}, pt{0.3, 300}, pt{0.36, 330}, pt{0.7, 200}, pt{1, 120}), // "Aaa-ah... ahh"
			vowel:  vowels3([]float64{0, 0.7, 1}, vA, vUh, vO),
			amp:    glide(pt{0, 0}, pt{0.04, 1}, pt{0.3, 0.7}, pt{0.33, 0.15}, pt{0.37, 0.8}, pt{0.75, 0.4}, pt{1, 0}),
			breath: glide(pt{0, 0.2}, pt{0.33, 0.7}, pt{0.38, 0.2}, pt{0.8, 0.6}, pt{1, 0}), strain: 0.45},
	}
	for _, c := range deaths {
		add("tribe_death", 0.85, echo(saturate(humanVoice(r, c), 1.3), pt{0.14, 0.2}, pt{0.31, 0.1}))
	}
}

// --- when they are heard --------------------------------------------------------

const (
	tribeReCry   = 8 * time.Second         // a warrior back in a fight this soon does not cry again
	tribeCryGap  = 1200 * time.Millisecond // between any two war cries
	tribeAnswerR = 10.0                    // tiles: warriors this near the one who cries answer him
	tribeAnswers = 3                       // and at most this many
	tribeReach   = 24.0                    // tiles a war cry carries
)

// manRate is the pitch of warrior id's voice: each man has his own.
func manRate(id sim.CreatureID) float64 {
	h := uint32(id) * 2654435761
	return 0.92 + 0.16*float64(h>>16)/65535
}

// tribeSounds plays what warrior c did since o; hit is whether he was hurt.
func (vm *voiceMemory) tribeSounds(o, c protocol.CreatureView, hit bool, cur protocol.Snapshot) {
	now := vm.now()
	x, y := float64(c.X), float64(c.Y)
	rate := manRate(c.ID)
	switch {
	case c.State == sim.CreatureChase && o.State != sim.CreatureChase && o.State != sim.CreatureStrike:
		last := vm.engaged[c.ID]
		vm.engaged[c.ID] = now
		if now.Sub(last) < tribeReCry {
			break // back in the same fight, or he has just answered another's cry
		}
		if !nearAPlayer(cur.Players, x, y, sim.TribeSightDay+1.5) {
			// After game, not a stranger: one short shout to the others.
			if vm.ready("tribe_hunt", 3*time.Second) {
				vm.callPick("tribe_shout", x, y, 0.5, rate, 18)
			}
			break
		}
		if !vm.ready("tribe_warcry", tribeCryGap) {
			break
		}
		vm.callPick("tribe_warcry", x, y, 0.9, rate, tribeReach)
		vm.rallyTribe(c, cur, now)
	case c.State == sim.CreatureStrike && o.State != sim.CreatureStrike:
		if vm.cooled(c.ID, "grunt", 600*time.Millisecond) {
			vm.callPick("tribe_grunt", x, y, 0.6, rate, 14)
			snd.playAt("swish", x, y, 0.35, 0.1)
		}
	}
	if hit && vm.cooled(c.ID, "pain", 900*time.Millisecond) {
		vm.callPick("tribe_pain", x, y, 0.75, rate, 18)
	}
}

// rallyTribe has the warriors near c answer his war cry one after another.
func (vm *voiceMemory) rallyTribe(c protocol.CreatureView, cur protocol.Snapshot, now time.Time) {
	type near struct {
		c protocol.CreatureView
		d float64
	}
	var ns []near
	for _, t := range cur.Creatures {
		if t.Kind != sim.CreatureTribesman || t.ID == c.ID || t.Tamed {
			continue
		}
		if d := math.Hypot(float64(t.X-c.X), float64(t.Y-c.Y)); d <= tribeAnswerR && now.Sub(vm.engaged[t.ID]) >= tribeReCry {
			ns = append(ns, near{t, d})
		}
	}
	for i := 1; i < len(ns); i++ { // nearest first
		for j := i; j > 0 && ns[j].d < ns[j-1].d; j-- {
			ns[j], ns[j-1] = ns[j-1], ns[j]
		}
	}
	for i, n := range ns[:min(len(ns), tribeAnswers)] {
		vm.engaged[n.c.ID] = now // he has shouted: no war cry of his own when he joins in
		delay := time.Duration((0.3 + 0.32*float64(i) + 0.2*vm.roll()) * float64(time.Second))
		vm.later(now.Add(delay), n.c.ID, "tribe_shout", float64(n.c.X), float64(n.c.Y), 0.65, manRate(n.c.ID), 22)
	}
}

// keepEngaged notes every warrior still fighting, so a long fight is one
// war cry, not one for each time he loses sight of his foe for a moment.
func (vm *voiceMemory) keepEngaged(cur protocol.Snapshot) {
	now := vm.now()
	for _, c := range cur.Creatures {
		if c.Kind == sim.CreatureTribesman && (c.State == sim.CreatureChase || c.State == sim.CreatureStrike) {
			vm.engaged[c.ID] = now
		}
	}
}
