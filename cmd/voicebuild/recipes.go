package main

import (
	"math/rand/v2"
)

// The recipes: what each creature's voice is made of, the way films make
// their dinosaurs: real animals slowed, pitched, filtered and laid over one
// another. The bigger the beast, the slower and lower. Each voice has its
// variants, made from different calls of the same animals, so no cry comes
// twice the same.

func voices() []voice {
	var vs []voice
	add := func(name string, n int, f func(v int, r *rand.Rand) []float64) {
		vs = append(vs, voice{name: name, n: n, make: f})
	}
	rexVoices(add)
	bossVoices(add)
	grazerVoices(add)
	smallVoices(add)
	hunterVoices(add)
	flyerVoices(add)
	seaVoices(add)
	beastVoices(add)
	return vs
}

type adder func(name string, n int, f func(v int, r *rand.Rand) []float64)

// pick is the v-th of xs, round and round.
func pick[T any](v int, xs ...T) T { return xs[v%len(xs)] }

// falling is a speed sinking from s to s*end along a call: a voice
// failing.
func falling(s, end float64) [][2]float64 {
	return [][2]float64{{0, s}, {0.4, s * (1 - (1-end)*0.3)}, {1, s * end}}
}

// dying is x fading away over its last share, with a last breath after it.
func dying(x []float64, r *rand.Rand, breathAt float64) []float64 {
	x = shape(x, pt{0, 1}, pt{0.5, 0.9}, pt{1, 0})
	b := lowpass(breath(r, 0.9, 500, 0.7), 1400)
	return mixAt(x, b, dur(x)*breathAt, 0.18)
}

// --- the tyrant -------------------------------------------------------------------

// rexRoar is the tyrant's roar: a lion's roar an octave down for its throat,
// an elk's bugle slowed into a trumpeting scream riding on top, a bison's
// bellow for its chest, an alligator's bellow and a deep rumble under it.
func rexRoar(v int, r *rand.Rand) []float64 {
	lion := pick(v, "lion_1", "lion_2", "lion_1")
	elk := pick(v, "elk_1", "elk_5", "elk_7")
	bison := pick(v, "bison_b1", "bison_c5", "bison_c4")
	gator := pick(v, "gator_2", "gator_3", "gator_4")
	k := pick(v, 1.0, 0.96, 1.05)

	// The throat: the lion an octave down, sinking as it runs out of breath.
	throat := bend(lion, pt{0, 0.53 * k}, pt{0.5, 0.5 * k}, pt{1, 0.44 * k})
	throat = lowpass(drive(eq(throat, 160, 0.8, 4), 1.6), 6000)
	// The scream: the elk's whistle, from where it rises to where it falls.
	scream := band(bendX(part(elk, 0.55, 2.55, 1), pt{0, 0.5 * k}, pt{1, 0.44 * k}), 300, 5000)
	scream = shape(scream, pt{0, 0}, pt{0.12, 1}, pt{0.6, 0.85}, pt{1, 0})
	// The chest: the bison's bellow, and the alligator's under it.
	chest := lowpass(use(bison, 0.6*k), 1500)
	low := lowpass(use(gator, 0.7), 450)
	// And the film's own trick: an elephant's trumpet slowed, in the scream.
	trumpet := band(bend("elephant_1", pt{0, 0.5 * k}, pt{0.3, 0.53 * k}, pt{1, 0.42 * k}), 200, 6000)
	x := mix(
		layer{throat, 0, 0},
		layer{scream, 0.3, -8},
		layer{trumpet, 0.22, -9},
		layer{chest, 0, -3},
		layer{low, 0.1, -9},
	)
	x = mixAt(x, rumble(x, 36*k, 26*k), 0, 0.3)
	x = shape(x, pt{0, 0.6}, pt{0.06, 1}, pt{0.55, 0.9}, pt{1, 0.15})
	x = eq(x, 2400, 1, 2)
	return reverb(x, 1.6, 0.2, 0.45)
}

func rexVoices(add adder) {
	add("rex_roar", 3, rexRoar)
	add("rex_roar_far", 2, func(v int, r *rand.Rand) []float64 {
		// Far off, only its low end comes through, and the hills echo it.
		x := lowpass(rexRoar(v+1, r), 420)
		return reverb(x, 2.6, 0.55, 0.7)
	})
	add("rex_growl", 2, func(v int, r *rand.Rand) []float64 {
		// Closing in: a bear's moan far down, rattling in its throat over a
		// bison's grunt, and its breath.
		moan := part("grizzly_2", pick(v, 0.4, 2.6), pick(v, 2.6, 4.9), 0.55)
		moan = rattle(lowpass(moan, 1600), 16, 0.55, r)
		grunt := lowpass(part(pick(v, "bison_b1", "bison_c2"), 0, 0.7, 0.5), 900)
		x := mix(layer{moan, 0, 0}, layer{grunt, 0.2, -5}, layer{lowpass(breath(r, 1.2, 400, 0.6), 900), 0.4, -14})
		x = mixAt(x, rumble(x, 30, 26), 0, 0.35)
		return reverb(drive(x, 1.6), 1.0, 0.15, 0.5)
	})
	add("rex_hurt", 3, func(v int, r *rand.Rand) []float64 {
		// A short, broken roar: the lion's onset and the elk's scream
		// jumping up, cut off.
		lion := part(pick(v, "lion_2", "lion_1", "lion_2"), 0.05, 1.0, 1)
		throat := bendX(lion, pt{0, 0.55}, pt{0.3, 0.62}, pt{1, 0.45})
		scream := band(bendX(part(pick(v, "elk_2", "elk_8", "elk_4"), 0.7, 1.8, 1), pt{0, 0.55}, pt{0.25, 0.62}, pt{1, 0.48}), 300, 5000)
		chest := lowpass(part(pick(v, "bison_a3", "bison_b3", "bison_c2"), 0, 0.9, 0.62), 1500)
		x := mix(layer{drive(throat, 1.8), 0, 0}, layer{shape(scream, pt{0, 0}, pt{0.1, 1}, pt{1, 0}), 0.08, -6}, layer{chest, 0, -4})
		x = shape(x, pt{0, 1}, pt{0.6, 0.8}, pt{1, 0})
		return reverb(x, 1.2, 0.18, 0.45)
	})
	add("rex_death", 2, func(v int, r *rand.Rand) []float64 {
		// A long groan sinking away: the bison and the lion falling an
		// octave and more, the bear's moan under them, the last breath.
		groan := bend(pick(v, "bison_c4", "bison_c5"), falling(0.5, 0.62)...)
		lion := bend(pick(v, "lion_1", "lion_2"), falling(0.46, 0.6)...)
		moan := part("grizzly_2", 1, 4.5, 0.5)
		x := mix(layer{lowpass(groan, 1600), 0, 0}, layer{lowpass(drive(lion, 1.4), 5000), 0.1, -3}, layer{lowpass(moan, 900), 0.8, -8})
		x = mixAt(x, rumble(x, 30, 20), 0, 0.3)
		x = dying(x, r, 0.8)
		return reverb(x, 1.8, 0.22, 0.5)
	})
}

// --- the bosses -----------------------------------------------------------------

func bossVoices(add adder) {
	// The Spinosaurus: a crocodile's throat. Alligators bellowing and a
	// seal's underwater roar, far down, with a hiss over them; no scream.
	spinoRoar := func(v int, r *rand.Rand) []float64 {
		gator := lowpass(use(pick(v, "gator_2", "gator_3", "gator_4"), 0.62), 700)
		seal := bend(pick(v, "seal_1", "seal_2", "seal_1"), pt{0, 0.42}, pt{1, 0.36})
		seal = rattle(lowpass(seal, 3500), 11, 0.5, r)
		lion := lowpass(bend(pick(v, "lion_2", "lion_1", "lion_2"), pt{0, 0.6}, pt{1, 0.48}), 2800)
		x := mix(layer{gator, 0, 0}, layer{stretch(seal, 1.6), 0.05, -1}, layer{lion, 0.25, -5},
			layer{hiss(r, 2.4, 2600), 0.15, -15})
		x = mixAt(x, rumble(x, 40, 30), 0, 0.35)
		x = shape(x, pt{0, 0.7}, pt{0.1, 1}, pt{0.6, 0.85}, pt{1, 0.1})
		return reverb(drive(x, 1.5), 1.6, 0.22, 0.5)
	}
	add("spino_roar", 3, spinoRoar)
	add("spino_hurt", 3, func(v int, r *rand.Rand) []float64 {
		seal := bend(pick(v, "seal_2", "seal_1", "seal_2"), pt{0, 0.5}, pt{0.3, 0.56}, pt{1, 0.42})
		gator := lowpass(part(pick(v, "gator_1", "gator_3", "gator_4"), 0, 1.2, 0.7), 600)
		x := mix(layer{rattle(seal, 12, 0.4, r), 0, 0}, layer{gator, 0, -3}, layer{hiss(r, 0.8, 3000), 0.05, -16})
		return reverb(drive(shape(x, pt{0, 1}, pt{1, 0}), 1.6), 1.1, 0.18, 0.5)
	})
	add("spino_death", 2, func(v int, r *rand.Rand) []float64 {
		gator := lowpass(bend(pick(v, "gator_2", "gator_4"), falling(0.55, 0.7)...), 650)
		seal := bendX(stretch(use(pick(v, "seal_1", "seal_2"), 1), 2.2), falling(0.4, 0.65)...)
		x := mix(layer{gator, 0, 0}, layer{rattle(lowpass(seal, 3000), 9, 0.5, r), 0.1, -2})
		x = mixAt(x, rumble(x, 34, 22), 0, 0.3)
		return reverb(dying(x, r, 0.75), 1.8, 0.22, 0.5)
	})

	// The Carcharodontosaurus: a rasping roar, higher than the tyrant's:
	// the grizzly's roar and the lion over it, pushed hard, and an elk's
	// scream torn ragged.
	add("carcharo_roar", 3, func(v int, r *rand.Rand) []float64 {
		bear := part("grizzly_1", pick(v, 1.3, 2.2, 1.6), pick(v, 4.6, 5.2, 4.9), 0.7)
		rasp := lowpass(part(pick(v, "tiger_2", "tiger_3", "tiger_1"), 0, 1.5, 0.66), 4500) // a tiger's snarl far down
		lion := bend(pick(v, "lion_2", "lion_1", "lion_2"), pt{0, 0.62}, pt{0.5, 0.6}, pt{1, 0.5})
		scream := band(bendX(part(pick(v, "elk_3", "elk_6", "elk_9"), 0.5, 2.3, 1), pt{0, 0.6}, pt{1, 0.52}), 400, 6000)
		scream = rattle(scream, 34, 0.6, r)
		x := mix(layer{lowpass(bear, 5000), 0, -2}, layer{drive(lion, 2.4), 0.1, 0},
			layer{shape(scream, pt{0, 0}, pt{0.15, 1}, pt{1, 0}), 0.3, -9},
			layer{lowpass(use(pick(v, "bison_a1", "bison_b2", "bison_c1"), 0.7), 1400), 0, -6}, layer{drive(rasp, 1.8), 0.05, -4})
		x = rattle(x, 27, 0.25, r)
		x = shape(x, pt{0, 0.6}, pt{0.07, 1}, pt{0.6, 0.85}, pt{1, 0.1})
		x = shelf(x, false, 2500, 3)
		return reverb(drive(x, 1.8), 1.5, 0.2, 0.4)
	})
	add("carcharo_hurt", 3, func(v int, r *rand.Rand) []float64 {
		lion := bendX(part(pick(v, "lion_1", "lion_2", "lion_1"), 0.1, 0.9, 1), pt{0, 0.68}, pt{0.3, 0.75}, pt{1, 0.55})
		bear := part("grizzly_1", pick(v, 1.4, 2.8, 3.9), pick(v, 2.4, 3.8, 4.9), 0.8)
		x := mix(layer{drive(lion, 2.5), 0, 0}, layer{lowpass(bear, 5000), 0, -5})
		x = rattle(x, 30, 0.3, r)
		return reverb(shape(x, pt{0, 1}, pt{0.6, 0.8}, pt{1, 0}), 1.1, 0.18, 0.4)
	})
	add("carcharo_death", 2, func(v int, r *rand.Rand) []float64 {
		lion := bend(pick(v, "lion_2", "lion_1"), falling(0.56, 0.6)...)
		bear := bendX(part("grizzly_1", 1.3, 5.1, 1), falling(0.62, 0.6)...)
		x := mix(layer{drive(lion, 2), 0, 0}, layer{lowpass(bear, 3000), 0.2, -4})
		x = rattle(x, 18, 0.35, r)
		return reverb(dying(x, r, 0.8), 1.6, 0.22, 0.45)
	})

	// The Giganotosaurus: deeper and longer than any: the lion two octaves
	// down and drawn out, a bison under it, the elk's scream low and long.
	add("giganoto_roar", 3, func(v int, r *rand.Rand) []float64 {
		k := pick(v, 1.0, 0.95, 1.04)
		lion := stretch(bend(pick(v, "lion_1", "lion_2", "lion_1"), pt{0, 0.4 * k}, pt{0.5, 0.38 * k}, pt{1, 0.33 * k}), 1.25)
		scream := band(bendX(part(pick(v, "elk_10", "elk_4", "elk_6"), 0.5, 2.8, 1), pt{0, 0.36 * k}, pt{1, 0.3 * k}), 180, 4000)
		scream = shape(scream, pt{0, 0}, pt{0.15, 1}, pt{0.7, 0.8}, pt{1, 0})
		chest := lowpass(use(pick(v, "bison_c4", "bison_b1", "bison_c3"), 0.45*k), 1100)
		x := mix(layer{lowpass(drive(eq(lion, 120, 0.8, 4), 1.5), 4500), 0, 0}, layer{scream, 0.5, -8}, layer{chest, 0.1, -3},
			layer{lowpass(use(pick(v, "gator_3", "gator_2", "gator_4"), 0.55), 400), 0, -8})
		x = mixAt(x, rumble(x, 28*k, 20*k), 0, 0.4)
		x = shape(x, pt{0, 0.5}, pt{0.08, 1}, pt{0.6, 0.9}, pt{1, 0.12})
		return reverb(x, 2.2, 0.26, 0.55)
	})
	add("giganoto_hurt", 3, func(v int, r *rand.Rand) []float64 {
		lion := bendX(part(pick(v, "lion_1", "lion_2", "lion_1"), 0.05, 1.1, 1), pt{0, 0.42}, pt{0.3, 0.48}, pt{1, 0.36})
		chest := lowpass(part(pick(v, "bison_b3", "bison_a3", "bison_c2"), 0, 0.9, 0.48), 1000)
		x := mix(layer{lowpass(drive(lion, 1.7), 4000), 0, 0}, layer{chest, 0, -3})
		x = mixAt(x, rumble(x, 30, 24), 0, 0.3)
		return reverb(shape(x, pt{0, 1}, pt{0.6, 0.8}, pt{1, 0}), 1.5, 0.22, 0.5)
	})
	add("giganoto_death", 2, func(v int, r *rand.Rand) []float64 {
		lion := stretch(bend(pick(v, "lion_1", "lion_2"), falling(0.38, 0.6)...), 1.2)
		groan := bend(pick(v, "bison_c5", "bison_c4"), falling(0.42, 0.6)...)
		x := mix(layer{lowpass(drive(lion, 1.4), 4000), 0, 0}, layer{lowpass(groan, 1000), 0.2, -2})
		x = mixAt(x, rumble(x, 26, 17), 0, 0.35)
		return reverb(dying(x, r, 0.8), 2.2, 0.25, 0.55)
	})
}

// --- the grazers -----------------------------------------------------------------

// snort is a big grazer's hot snort through its nose, as a rhino's.
func snort(r *rand.Rand, k float64) []float64 {
	x := breath(r, 0.28/k, 700*k, 1.2)
	x = mixAt(x, lowpass(breath(r, 0.2/k, 260*k, 1.5), 700), 0.02, 1.2)
	return drive(x, 1.5)
}

// grazerCall is a horned beast's bellow: a bison at speed s, its grunt
// first if it has one, a snort before it.
func grazerCall(r *rand.Rand, clip string, s float64, snortGain float64) []float64 {
	x := lowpass(use(clip, s), 4500)
	if snortGain > -40 {
		x = mixAt(x, snort(r, s), 0, dbg(snortGain))
	}
	x = mixAt(x, rumble(x, 70*s, 58*s), 0, 0.2)
	return reverb(x, 1.0, 0.14, 0.45)
}

func dbg(d float64) float64 { return pow10(d / 20) }

func grazerVoices(add adder) {
	// The Triceratops: a bison's bellow a little down, a rhino's snort.
	add("trike_call", 3, func(v int, r *rand.Rand) []float64 {
		return grazerCall(r, pick(v, "bison_a1", "bison_b2", "bison_a2"), pick(v, 0.84, 0.8, 0.86), -40)
	})
	add("trike_alarm", 2, func(v int, r *rand.Rand) []float64 {
		x := grazerCall(r, pick(v, "bison_b1", "bison_a2"), 0.82, -40)
		return mix(layer{snort(r, 0.85), 0, -2}, layer{drive(x, 1.6), 0.18, 0})
	})
	add("trike_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "bison_a3", "bison_b3", "bison_c2"), 0, 0.9, 1), pt{0, 0.86}, pt{0.25, 0.98}, pt{1, 0.78})
		return reverb(drive(shape(lowpass(x, 4500), pt{0, 1}, pt{0.7, 0.8}, pt{1, 0}), 1.8), 0.8, 0.12, 0.45)
	})
	add("trike_death", 2, func(v int, r *rand.Rand) []float64 {
		x := lowpass(bend(pick(v, "bison_c4", "bison_b1"), falling(0.74, 0.62)...), 3500)
		return reverb(dying(x, r, 0.75), 1.2, 0.16, 0.5)
	})

	// The Torosaurus: deeper and drawn out, an alligator under the bison.
	add("toro_call", 3, func(v int, r *rand.Rand) []float64 {
		b := stretch(use(pick(v, "bison_c3", "bison_c1", "bison_c5"), pick(v, 0.68, 0.66, 0.7)), 1.2)
		x := mix(layer{lowpass(b, 3500), 0, 0}, layer{lowpass(use(pick(v, "gator_1", "gator_2", "gator_4"), 0.8), 400), 0.1, -10})
		x = mixAt(x, rumble(x, 50, 42), 0, 0.25)
		return reverb(x, 1.2, 0.16, 0.5)
	})
	add("toro_alarm", 2, func(v int, r *rand.Rand) []float64 {
		b := lowpass(use(pick(v, "bison_c4", "bison_b2"), 0.7), 3500)
		x := mix(layer{snort(r, 0.7), 0, -3}, layer{drive(b, 1.8), 0.2, 0})
		x = mixAt(x, rumble(x, 50, 40), 0, 0.25)
		return reverb(x, 1.2, 0.16, 0.5)
	})
	add("toro_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "bison_c2", "bison_c1", "bison_b3"), 0, 1.0, 1), pt{0, 0.72}, pt{0.25, 0.82}, pt{1, 0.64})
		return reverb(drive(shape(lowpass(x, 3500), pt{0, 1}, pt{0.7, 0.8}, pt{1, 0}), 1.8), 1.0, 0.14, 0.5)
	})
	add("toro_death", 2, func(v int, r *rand.Rand) []float64 {
		x := lowpass(stretch(bend(pick(v, "bison_c5", "bison_c3"), falling(0.62, 0.6)...), 1.15), 3000)
		x = mixAt(x, rumble(x, 44, 30), 0, 0.25)
		return reverb(dying(x, r, 0.75), 1.4, 0.18, 0.5)
	})

	// The Stegosaurus: a low groaning moan, bear cubs' bawling slowed far
	// down over a bison's low bellow.
	add("stego_call", 3, func(v int, r *rand.Rand) []float64 {
		moan := part(pick(v, "cub_2", "cub_3", "cub_1"), 0.05, pick(v, 1.0, 0.75, 0.62), pick(v, 0.36, 0.34, 0.38))
		low := lowpass(use(pick(v, "bison_c1", "bison_c3", "bison_a2"), 0.6), 900)
		x := mix(layer{lowpass(moan, 3000), 0, 0}, layer{low, 0.15, -7})
		x = mixAt(x, rumble(x, 55, 45), 0, 0.2)
		return reverb(x, 1.2, 0.18, 0.5)
	})
	add("stego_alarm", 2, func(v int, r *rand.Rand) []float64 {
		moan := part(pick(v, "cub_2", "cub_3"), 0, 1.1, 0.42)
		low := lowpass(use(pick(v, "bison_b2", "bison_c2"), 0.66), 1200)
		return reverb(drive(mix(layer{lowpass(moan, 3500), 0, 0}, layer{low, 0.05, -3}), 1.7), 1.0, 0.15, 0.5)
	})
	add("stego_hurt", 3, func(v int, r *rand.Rand) []float64 {
		moan := bendX(part(pick(v, "cub_3", "cub_1", "cub_2"), 0, pick(v, 0.7, 0.6, 0.7), 1), pt{0, 0.44}, pt{0.2, 0.5}, pt{1, 0.38})
		return reverb(drive(shape(lowpass(moan, 3500), pt{0, 1}, pt{1, 0}), 1.6), 0.9, 0.14, 0.5)
	})
	add("stego_death", 2, func(v int, r *rand.Rand) []float64 {
		moan := bendX(part(pick(v, "cub_2", "cub_3"), 0, pick(v, 1.0, 0.75), 1), falling(0.36, 0.6)...)
		low := lowpass(bend(pick(v, "bison_c4", "bison_c1"), falling(0.55, 0.65)...), 900)
		x := mix(layer{lowpass(moan, 3000), 0, 0}, layer{low, 0.2, -5})
		return reverb(dying(x, r, 0.75), 1.4, 0.18, 0.5)
	})

	// The Brachiosaurus: a long hooting call, an owl's hoot and an elk's
	// whistle slowed into a deep horn, echoing in the valleys.
	add("brachio_call", 3, func(v int, r *rand.Rand) []float64 {
		// The donkey's "hee" far down, the owl's hoot and the elk's whistle
		// in it, as a horn.
		bray := part(pick(v, "donkey_hee", "donkey_hee2", "donkey_hee"), pick(v, 0.1, 0.1, 0.9), pick(v, 1.6, 1.5, 2.3), pick(v, 0.44, 0.42, 0.46))
		bray = shape(lowpass(bray, 2500), pt{0, 0}, pt{0.12, 1}, pt{0.75, 0.85}, pt{1, 0})
		hoot := lowpass(use(pick(v, "owl_1", "owl_2", "owl_1"), pick(v, 0.42, 0.45, 0.4)), 2000)
		horn := band(part(pick(v, "elk_9", "elk_7", "elk_3"), 0.8, 2.3, pick(v, 0.33, 0.31, 0.34)), 150, 2500)
		horn = shape(horn, pt{0, 0}, pt{0.2, 1}, pt{0.75, 0.8}, pt{1, 0})
		x := mix(layer{bray, 0, 0}, layer{hoot, 0.1, -5}, layer{horn, 0.3, -9})
		x = mixAt(x, rumble(x, 46, 38), 0, 0.4)
		return reverb(x, 3.2, 0.4, 0.55)
	})
	add("brachio_alarm", 2, func(v int, r *rand.Rand) []float64 {
		horn := band(part("donkey_haw", pick(v, 0.05, 0.4), pick(v, 1.2, 1.5), pick(v, 0.5, 0.47)), 120, 3000)
		low := lowpass(use(pick(v, "bison_c3", "bison_b1"), 0.52), 900)
		x := mix(layer{drive(horn, 1.5), 0, 0}, layer{low, 0, -3})
		x = mixAt(x, rumble(x, 40, 32), 0, 0.35)
		return reverb(x, 2.2, 0.3, 0.55)
	})
	add("brachio_hurt", 3, func(v int, r *rand.Rand) []float64 {
		horn := bendX(part(pick(v, "donkey_haw", "donkey_hee", "donkey_haw"), pick(v, 0.05, 0.2, 0.6), pick(v, 0.8, 1.0, 1.3), 1), pt{0, 0.5}, pt{0.25, 0.56}, pt{1, 0.44})
		low := lowpass(part(pick(v, "bison_a3", "bison_b3", "bison_c2"), 0, 0.9, 0.5), 900)
		x := mix(layer{band(horn, 150, 3000), 0, 0}, layer{low, 0, -2})
		return reverb(drive(shape(x, pt{0, 1}, pt{0.7, 0.8}, pt{1, 0}), 1.5), 1.6, 0.25, 0.55)
	})
	add("brachio_death", 2, func(v int, r *rand.Rand) []float64 {
		horn := band(bendX(part(pick(v, "donkey_hee", "donkey_hee2"), 0.1, 2.0, 1), falling(0.42, 0.6)...), 120, 2500)
		low := lowpass(bend(pick(v, "bison_c5", "bison_c4"), falling(0.45, 0.6)...), 800)
		x := mix(layer{horn, 0, 0}, layer{low, 0.3, -3})
		x = mixAt(x, rumble(x, 38, 24), 0, 0.35)
		return reverb(dying(x, r, 0.8), 2.6, 0.32, 0.55)
	})

	// The armoured Ankylosaurus, for the branch that adds it: a deep,
	// grunting bellow from its armour-plated chest, a pig frog's grunts and
	// a bison far down. The game plays these once the beast is wired in.
	add("ankylo_call", 2, func(v int, r *rand.Rand) []float64 {
		grunts := lowpass(use("pigfrog_1", pick(v, 0.55, 0.5)), 1500)
		low := lowpass(use(pick(v, "bison_c2", "bison_a3"), 0.55), 900)
		x := mix(layer{grunts, 0, 0}, layer{low, 0.3, -4})
		x = mixAt(x, rumble(x, 48, 40), 0, 0.3)
		return reverb(x, 1.0, 0.15, 0.5)
	})
	add("ankylo_alarm", 2, func(v int, r *rand.Rand) []float64 {
		low := lowpass(use(pick(v, "bison_b2", "bison_c1"), 0.6), 1500)
		x := mix(layer{snort(r, 0.6), 0, -2}, layer{drive(low, 2), 0.15, 0}, layer{lowpass(use("pigfrog_1", 0.6), 1500), 0.3, -6})
		return reverb(x, 1.0, 0.15, 0.5)
	})
	add("ankylo_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "bison_b3", "bison_a3"), 0, 0.8, 1), pt{0, 0.6}, pt{0.25, 0.7}, pt{1, 0.52})
		return reverb(drive(shape(lowpass(x, 2500), pt{0, 1}, pt{1, 0}), 2), 0.9, 0.14, 0.5)
	})
	add("ankylo_death", 2, func(v int, r *rand.Rand) []float64 {
		x := lowpass(bend(pick(v, "bison_c1", "bison_c3"), falling(0.55, 0.6)...), 1500)
		x = mixAt(x, lowpass(bend("pigfrog_1", falling(0.45, 0.7)...), 900), 0.4, 0.5)
		return reverb(dying(x, r, 0.75), 1.2, 0.16, 0.5)
	})
}

// --- the small plant-eaters --------------------------------------------------------

func smallVoices(add adder) {
	// The Thescelosaurus: booms from the chest, like an emu's: a bittern's
	// boom and a pig frog's grunt; a heron's croak when it is hurt.
	add("thesc_call", 3, func(v int, r *rand.Rand) []float64 {
		boom := lowpass(use(pick(v, "bittern_1", "bittern_2", "bittern_4"), pick(v, 1.06, 1.0, 1.1)), 2500)
		if v == 1 {
			boom = seq(-0.08, boom, gain(lowpass(use("bittern_3", 1.04), 2500), 0.8)) // two booms
		}
		grunt := lowpass(part("pigfrog_1", 0, 0.8, 0.9), 2000)
		return reverb(mix(layer{boom, 0, 0}, layer{grunt, 0, -10}), 0.9, 0.15, 0.5)
	})
	add("thesc_alarm", 2, func(v int, r *rand.Rand) []float64 {
		croak := use(pick(v, "heron_1", "heron_3"), 0.72)
		boom := lowpass(part("pigfrog_1", 0, 0.8, 0.9), 2500)
		return reverb(seq(-0.04, drive(croak, 1.5), boom), 0.8, 0.14, 0.5)
	})
	add("thesc_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bend(pick(v, "heron_2", "heron_4", "heron_1"), pt{0, 0.7}, pt{0.3, 0.8}, pt{1, 0.62})
		return reverb(drive(x, 1.8), 0.6, 0.12, 0.5)
	})
	add("thesc_death", 2, func(v int, r *rand.Rand) []float64 {
		croak := bend(pick(v, "heron_1", "heron_3"), falling(0.6, 0.65)...)
		boom := lowpass(bend("pigfrog_1", falling(0.66, 0.65)...), 2000)
		return reverb(dying(seq(-0.05, drive(croak, 1.4), boom), r, 0.8), 0.9, 0.15, 0.5)
	})

	// The Ornithomimus: an ostrich-like honk, geese slowed a little.
	add("ornitho_call", 3, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "geese_1", "geese_2", "geese_3"), pick(v, 0, 1.25, 0), pick(v, 1.15, 2.6, 1.3), pick(v, 0.78, 0.74, 0.8))
		return reverb(lowpass(x, 5000), 0.8, 0.12, 0.45)
	})
	add("ornitho_alarm", 2, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "elkcalf_1", "elkcalf_2"), 0, 0.7, pick(v, 0.95, 1.0))
		honk := part(pick(v, "geese_2", "geese_3"), 0, 0.9, 0.85)
		return reverb(seq(0.18, x, honk), 0.7, 0.12, 0.45)
	})
	add("ornitho_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "geese_1", "geese_2", "geese_3"), pick(v, 0, 1.3, 0.1), pick(v, 0.6, 1.9, 0.7), 1), pt{0, 0.85}, pt{0.3, 1.0}, pt{1, 0.8})
		return reverb(drive(x, 1.6), 0.6, 0.1, 0.45)
	})
	add("ornitho_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "geese_1", "geese_3"), 0, 1.2, 1), falling(0.72, 0.6)...)
		return reverb(dying(x, r, 0.85), 0.8, 0.12, 0.45)
	})
}

// --- the hunters ------------------------------------------------------------------

func hunterVoices(add adder) {
	// The Dakotaraptor: an eagle's scream and an osprey's whistle with a
	// lion's snarl under them and a hiss; ravens' croaks and an anhinga's
	// rattle to its pack.
	add("raptor_alarm", 3, func(v int, r *rand.Rand) []float64 {
		// A dolphin's scream slowed into a shriek, a peacock's wail and an
		// osprey's whistle over it, the lion's snarl and a hiss under it.
		scream := part(pick(v, "dolphin_2", "dolphin_1", "dolphin_3"), pick(v, 0.3, 0.05, 0.0), pick(v, 1.1, 0.9, 1.0), pick(v, 0.62, 0.66, 0.6))
		wail := use(pick(v, "peacock_1", "peacock_3", "peacock_4"), pick(v, 0.74, 0.7, 0.78))
		whistle := part(pick(v, "osprey_1", "eagle_1", "osprey_1"), pick(v, 0.5, 0.4, 1.15), pick(v, 0.95, 1.05, 1.45), pick(v, 0.72, 0.75, 0.7))
		snarl := part(pick(v, "lion_2", "lion_1", "lion_2"), 0.05, 0.6, 1.25)
		x := mix(layer{drive(scream, 1.6), 0, 0}, layer{wail, 0.05, -5}, layer{drive(whistle, 1.5), 0.1, -9},
			layer{lowpass(drive(snarl, 2), 5000), 0, -6}, layer{hiss(r, 0.6, 3000), 0.15, -14})
		return reverb(x, 0.9, 0.14, 0.4)
	})
	add("raptor_call", 3, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "raven_2", "raven_1", "raven_3"), pick(v, 0.45, 0.4, 0), pick(v, 1.65, 1.5, 1.2), pick(v, 0.7, 0.66, 0.72))
		return reverb(drive(x, 1.4), 0.8, 0.12, 0.45)
	})
	add("raptor_growl", 2, func(v int, r *rand.Rand) []float64 {
		clicks := lowpass(use(pick(v, "anhinga_1", "anhinga_2"), 0.72), 3500)
		purr := rattle(lowpass(part("grizzly_2", pick(v, 1.0, 3.0), pick(v, 2.2, 4.2), 1.3), 1500), 22, 0.6, r)
		return reverb(mix(layer{clicks, 0, 0}, layer{purr, 0, -8}), 0.6, 0.1, 0.45)
	})
	add("raptor_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "dolphin_1", "dolphin_4", "dolphin_3"), pick(v, 0.05, 0.1, 0.0), pick(v, 0.55, 0.7, 0.6), 1), pt{0, 0.7}, pt{0.3, 0.82}, pt{1, 0.62})
		yelp := bendX(part(pick(v, "eagle_1", "osprey_1", "eagle_1"), pick(v, 0.4, 0.5, 0.8), pick(v, 0.62, 0.95, 1.05), 1), pt{0, 0.72}, pt{1, 0.65})
		return reverb(mix(layer{drive(x, 1.8), 0, 0}, layer{drive(yelp, 2), 0, -6}), 0.6, 0.1, 0.4)
	})
	add("raptor_death", 2, func(v int, r *rand.Rand) []float64 {
		scream := bend(pick(v, "dolphin_3", "dolphin_4"), falling(0.66, 0.55)...)
		wail := bend(pick(v, "peacock_2", "peacock_4"), falling(0.7, 0.6)...)
		croak := use(pick(v, "raven_4", "raven_3"), 0.6)
		x := seq(-0.02, mix(layer{drive(scream, 1.5), 0, 0}, layer{wail, 0.05, -5}), croak)
		return reverb(dying(x, r, 0.85), 0.9, 0.12, 0.45)
	})
	add("raptor_attack", 2, func(v int, r *rand.Rand) []float64 {
		snarl := part(pick(v, "tiger_4", "tiger_2"), 0, 0.5, 1.3) // a tiger's snarl, quicker
		return reverb(mix(layer{drive(snarl, 2.4), 0, 0}, layer{hiss(r, 0.4, 3500), 0, -10}), 0.5, 0.1, 0.4)
	})

	// The Acheroraptor: smaller and quicker: falcons' cacking and a fox's
	// bark where the Dakotaraptor screams.
	add("achero_alarm", 3, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "peregrine_1", "peregrine_2", "peregrine_3"), 0, pick(v, 0.85, 1.0, 1.1), pick(v, 0.82, 0.8, 0.85))
		scream := part(pick(v, "dolphin_4", "dolphin_2", "dolphin_1"), 0.0, 0.7, 0.85)
		return reverb(mix(layer{drive(x, 1.6), 0, 0}, layer{scream, 0.05, -7}, layer{hiss(r, 0.4, 3500), 0.1, -14}), 0.7, 0.12, 0.4)
	})
	add("achero_call", 3, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "fox_1", "fox_2", "fox_1"), 0, 0.6, pick(v, 0.8, 0.85, 0.76))
		return reverb(seq(pick(v, 0.28, 0.3, 0.25), x, gain(x, 0.7)), 0.7, 0.12, 0.45)
	})
	add("achero_growl", 2, func(v int, r *rand.Rand) []float64 {
		return reverb(lowpass(use(pick(v, "anhinga_3", "anhinga_2"), 0.95), 4500), 0.5, 0.1, 0.45)
	})
	add("achero_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "fox_2", "fox_1", "fox_2"), 0, 0.5, 1), pt{0, 0.9}, pt{0.3, 1.0}, pt{1, 0.8})
		return reverb(drive(x, 1.8), 0.5, 0.1, 0.4)
	})
	add("achero_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "peregrine_3", "peregrine_2"), 0, 1.0, 1), falling(0.82, 0.6)...)
		return reverb(dying(drive(x, 1.4), r, 0.85), 0.7, 0.12, 0.45)
	})
	add("achero_attack", 2, func(v int, r *rand.Rand) []float64 {
		snarl := part(pick(v, "tiger_1", "tiger_4"), 0, 0.4, 1.6)
		return reverb(mix(layer{drive(snarl, 2.4), 0, 0}, layer{hiss(r, 0.3, 4000), 0, -10}), 0.4, 0.1, 0.4)
	})

	// The Troodon, the night hunter: an owl's hoot high and quavering, an
	// elk calf's mew; a fox's scream when it sees you; the anhinga's clicks
	// as the pack closes in; a quick rising shriek as it bites.
	add("troodon_call", 3, func(v int, r *rand.Rand) []float64 {
		hoot := use(pick(v, "owl_1", "owl_2", "owl_1"), pick(v, 1.25, 1.3, 1.18))
		hoot = rattle(hoot, 9, 0.35, r)
		mew := part(pick(v, "elkcalf_1", "elkcalf_2", "elkcalf_3"), 0.05, 0.7, pick(v, 0.8, 0.78, 0.84))
		x := seq(0.3, hoot, gain(mew, 0.7))
		return reverb(x, 1.6, 0.3, 0.45)
	})
	add("troodon_alarm", 2, func(v int, r *rand.Rand) []float64 {
		x := use(pick(v, "fox_1", "fox_2"), 1.08)
		return reverb(drive(x, 1.6), 1.2, 0.25, 0.4)
	})
	add("troodon_growl", 2, func(v int, r *rand.Rand) []float64 {
		return reverb(band(use(pick(v, "anhinga_2", "anhinga_1"), 1.25), 400, 5000), 0.5, 0.12, 0.4)
	})
	add("troodon_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "elkcalf_1", "elkcalf_2", "elkcalf_3"), 0.05, 0.55, 1), pt{0, 0.95}, pt{0.3, 1.1}, pt{1, 0.85})
		return reverb(drive(x, 1.8), 0.6, 0.15, 0.4)
	})
	add("troodon_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bend(pick(v, "fox_1", "fox_2"), falling(0.85, 0.6)...)
		return reverb(dying(x, r, 0.9), 1.0, 0.22, 0.45)
	})
	add("troodon_attack", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "elkcalf_2", "elkcalf_3"), 0.05, 0.5, 1), pt{0, 0.9}, pt{1, 1.35})
		return reverb(drive(x, 2), 0.5, 0.12, 0.4)
	})

	// The sabre-tooth: the great cats near their own voices, a lion's roar
	// and a tiger's snarls and growls.
	add("sabre_roar", 2, func(v int, r *rand.Rand) []float64 {
		roar := bend(pick(v, "lion_1", "lion_2"), pt{0, 0.92}, pt{1, 0.82})
		snarl := part(pick(v, "tiger_4", "tiger_1"), 0, pick(v, 0.6, 0.7), 0.9)
		return reverb(lowpass(seq(-0.05, drive(snarl, 1.5), roar), 7000), 1.1, 0.16, 0.45)
	})
	add("sabre_snarl", 2, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "tiger_4", "tiger_2"), 0, pick(v, 0.6, 0.8), 1)
		lion := part(pick(v, "lion_2", "lion_1"), 0.05, 0.5, 1.05)
		return reverb(mix(layer{drive(x, 2), 0, 0}, layer{drive(lion, 2), 0.02, -6}), 0.6, 0.12, 0.45)
	})
	add("sabre_growl", 2, func(v int, r *rand.Rand) []float64 {
		x := use(pick(v, "tiger_5", "tiger_3"), pick(v, 0.92, 0.88))
		low := rattle(lowpass(part("grizzly_2", pick(v, 0.6, 3.0), pick(v, 2.0, 4.4), 0.95), 1500), 19, 0.5, r)
		return reverb(mix(layer{lowpass(x, 5000), 0, 0}, layer{low, 0, -6}), 0.8, 0.12, 0.5)
	})
	add("sabre_cower", 2, func(v int, r *rand.Rand) []float64 {
		x := hiss(r, pick(v, 0.7, 0.85), pick(v, 2200.0, 2600.0))
		snarl := lowpass(part(pick(v, "tiger_4", "tiger_1"), 0, 0.45, 1.15), 4000)
		return reverb(mix(layer{x, 0, 0}, layer{snarl, 0.05, -4}), 0.5, 0.1, 0.45)
	})
	add("sabre_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "tiger_1", "tiger_2"), 0, 0.7, 1), pt{0, 1.05}, pt{0.3, 1.18}, pt{1, 0.95})
		return reverb(drive(x, 2), 0.7, 0.12, 0.45)
	})
	add("sabre_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bend(pick(v, "lion_2", "lion_1"), falling(0.82, 0.6)...)
		growl := bend(pick(v, "tiger_3", "tiger_5"), falling(0.85, 0.6)...)
		return reverb(dying(mix(layer{x, 0, 0}, layer{lowpass(growl, 4000), 0.3, -5}), r, 0.85), 1.0, 0.15, 0.45)
	})
}

// --- the flyers -------------------------------------------------------------------

func flyerVoices(add adder) {
	// The pterosaur: a heron's squawk lowered, an osprey's whistle as it
	// dives.
	add("ptero_call", 2, func(v int, r *rand.Rand) []float64 {
		x := use(pick(v, "heron_1", "heron_3"), 0.72)
		return reverb(seq(0.36, x, gain(use(pick(v, "heron_2", "heron_4"), 0.74), 0.8)), 1.0, 0.2, 0.4)
	})
	add("ptero_alarm", 2, func(v int, r *rand.Rand) []float64 {
		x := use(pick(v, "peacock_2", "peacock_4"), 0.64) // a peacock's wail lowered, as it dives
		croak := use(pick(v, "heron_1", "heron_2"), 0.7)
		return reverb(mix(layer{drive(x, 1.6), 0, 0}, layer{drive(croak, 1.5), 0.02, -3}), 0.9, 0.18, 0.4)
	})
	add("ptero_hurt", 3, func(v int, r *rand.Rand) []float64 {
		x := bend(pick(v, "heron_2", "heron_4", "heron_3"), pt{0, 0.78}, pt{0.3, 0.9}, pt{1, 0.7})
		return reverb(drive(x, 2), 0.6, 0.12, 0.4)
	})
	add("ptero_death", 2, func(v int, r *rand.Rand) []float64 {
		x := seq(-0.02, bend(pick(v, "heron_1", "heron_3"), falling(0.65, 0.7)...), bend(pick(v, "heron_4", "heron_2"), falling(0.55, 0.6)...))
		return reverb(dying(drive(x, 1.5), r, 0.9), 1.0, 0.18, 0.4)
	})

	// The Quetzalcoatlus: deeper, a raven's croak far down and a stork's
	// bill clattering.
	add("quetzal_call", 3, func(v int, r *rand.Rand) []float64 {
		croak := part(pick(v, "raven_1", "raven_2", "raven_3"), pick(v, 0.6, 0.5, 0), pick(v, 1.75, 1.6, 1.2), pick(v, 0.5, 0.48, 0.52))
		x := mix(layer{drive(croak, 1.5), 0, 0}, layer{clatter(r, 9, 16), dur(croak) * 0.7, -8})
		return reverb(x, 1.2, 0.2, 0.45)
	})
	add("quetzal_alarm", 2, func(v int, r *rand.Rand) []float64 {
		scream := use(pick(v, "peacock_3", "peacock_1"), 0.46)
		croak := use(pick(v, "heron_1", "heron_3"), 0.48)
		return reverb(mix(layer{drive(scream, 1.6), 0, -2}, layer{drive(croak, 1.6), 0.05, 0}), 1.1, 0.2, 0.45)
	})
	add("quetzal_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "raven_2", "raven_3"), pick(v, 0.5, 0), pick(v, 0.9, 0.5), 1), pt{0, 0.56}, pt{0.3, 0.64}, pt{1, 0.5})
		return reverb(drive(x, 2), 0.8, 0.15, 0.45)
	})
	add("quetzal_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "raven_1", "raven_2"), pick(v, 0.5, 0.45), pick(v, 1.8, 1.6), 1), falling(0.46, 0.6)...)
		return reverb(dying(drive(x, 1.5), r, 0.85), 1.2, 0.2, 0.45)
	})

	// The Anzu: a cassowary's boom (a bittern's, far down), and a sandhill
	// crane's rolling rattle.
	add("anzu_call", 2, func(v int, r *rand.Rand) []float64 {
		boom := lowpass(use(pick(v, "bittern_2", "bittern_4"), pick(v, 0.7, 0.66)), 900) // a cassowary's boom
		boom = mixAt(boom, rumble(boom, 40, 34), 0, 0.5)
		rattleCall := part(pick(v, "crane_1", "crane_2"), 0, pick(v, 1.15, 1.0), 0.62)
		return reverb(seq(-0.1, boom, rattleCall), 1.0, 0.16, 0.5)
	})
	add("anzu_roar", 2, func(v int, r *rand.Rand) []float64 {
		x := part(pick(v, "crane_1", "crane_3"), 0, pick(v, 1.15, 1.2), 0.82)
		cack := part(pick(v, "peregrine_2", "peregrine_1"), 0, 0.8, 0.7)
		return reverb(mix(layer{drive(x, 1.8), 0, 0}, layer{cack, 0.05, -6}), 0.9, 0.15, 0.45)
	})
	add("anzu_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "crane_3", "crane_2"), 0, 0.6, 1), pt{0, 0.8}, pt{0.3, 0.92}, pt{1, 0.72})
		return reverb(drive(x, 1.8), 0.7, 0.12, 0.45)
	})
	add("anzu_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "crane_1", "crane_2"), 0, 1.2, 1), falling(0.62, 0.6)...)
		return reverb(dying(drive(x, 1.4), r, 0.85), 1.0, 0.15, 0.45)
	})
}

// --- the sea ------------------------------------------------------------------------

func seaVoices(add adder) {
	// The Plesiosaurus: a fur seal's growling call slowed, a dolphin's
	// scream far down behind it, heard through the water.
	add("plesio_alarm", 2, func(v int, r *rand.Rand) []float64 {
		seal := use(pick(v, "furseal_2", "furseal_3"), pick(v, 0.6, 0.56))
		scream := lowpass(part(pick(v, "dolphin_3", "dolphin_4"), 0, 1.2, 0.42), 2500)
		x := mix(layer{rattle(seal, 14, 0.3, r), 0, 0}, layer{scream, 0.2, -9})
		return underwater(x, r)
	})
	add("plesio_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part(pick(v, "furseal_3", "furseal_2"), 0, 0.8, 1), pt{0, 0.66}, pt{0.3, 0.76}, pt{1, 0.6})
		return underwater(drive(x, 1.6), r)
	})
	add("plesio_death", 2, func(v int, r *rand.Rand) []float64 {
		seal := bend(pick(v, "furseal_2", "furseal_3"), falling(0.56, 0.6)...)
		scream := lowpass(bend(pick(v, "dolphin_1", "dolphin_2"), falling(0.4, 0.6)...), 2000)
		return underwater(dying(mix(layer{seal, 0, 0}, layer{scream, 0.3, -8}), r, 0.9), r)
	})

	// The Mosasaurus: a deep rumble from the deep, the seal's roar far
	// down over an alligator's bellow.
	add("mosa_call", 2, func(v int, r *rand.Rand) []float64 {
		seal := stretch(bend(pick(v, "seal_2", "seal_1"), pt{0, 0.42}, pt{1, 0.36}), 1.5)
		gator := lowpass(use(pick(v, "gator_3", "gator_2"), 0.5), 400)
		x := mix(layer{rattle(seal, 8, 0.5, r), 0, 0}, layer{gator, 0, -2})
		x = mixAt(x, rumble(x, 30, 24), 0, 0.45)
		return underwater(x, r)
	})
	add("mosa_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := bend(pick(v, "seal_1", "seal_2"), pt{0, 0.5}, pt{0.3, 0.56}, pt{1, 0.44})
		x = mixAt(x, rumble(x, 34, 28), 0, 0.4)
		return underwater(drive(x, 1.6), r)
	})
	add("mosa_death", 2, func(v int, r *rand.Rand) []float64 {
		seal := bendX(stretch(use(pick(v, "seal_2", "seal_1"), 1), 2.4), falling(0.4, 0.6)...)
		gator := lowpass(bend(pick(v, "gator_4", "gator_1"), falling(0.45, 0.7)...), 400)
		x := mix(layer{seal, 0, 0}, layer{gator, 0.2, -3})
		x = mixAt(x, rumble(x, 28, 18), 0, 0.4)
		return underwater(dying(x, r, 0.9), r)
	})

	// The Megalodon gives no cry, but a great shark hit or dying groans and
	// thrashes: a bison's bellow sunk to a whale's depth, under the water.
	add("megalodon_hurt", 2, func(v int, r *rand.Rand) []float64 {
		x := lowpass(bend(pick(v, "bison_b2", "bison_c3"), pt{0, 0.36}, pt{0.3, 0.4}, pt{1, 0.32}), 900)
		x = mixAt(x, rumble(x, 24, 20), 0, 0.5)
		return underwater(drive(x, 1.5), r)
	})
	add("megalodon_death", 2, func(v int, r *rand.Rand) []float64 {
		x := lowpass(bend(pick(v, "bison_c4", "bison_b1"), falling(0.34, 0.6)...), 800)
		x = mixAt(x, rumble(x, 22, 15), 0, 0.5)
		return underwater(dying(x, r, 0.9), r)
	})

	// The Kraken: a wet, wavering groan from the deep: the bear's moan and
	// the elk calf's mews far down, bubbling.
	add("kraken_call", 2, func(v int, r *rand.Rand) []float64 {
		moan := part("grizzly_2", pick(v, 0.4, 2.4), pick(v, 3.2, 5.5), 0.62)
		mew := lowpass(part(pick(v, "elkcalf_2", "elkcalf_3"), 0.05, 1.5, 0.34), 1500)
		x := mix(layer{moan, 0, 0}, layer{mew, 0.4, -6})
		x = mixAt(x, bubbles(r, dur(x)), 0, 0.35)
		return underwater(x, r)
	})
	add("kraken_hurt", 2, func(v int, r *rand.Rand) []float64 {
		mew := bendX(part(pick(v, "elkcalf_1", "elkcalf_3"), 0.05, 0.7, 1), pt{0, 0.42}, pt{0.3, 0.5}, pt{1, 0.38})
		x := mix(layer{drive(mew, 1.5), 0, 0}, layer{bubbles(r, 1), 0, -6})
		return underwater(x, r)
	})
	add("kraken_death", 2, func(v int, r *rand.Rand) []float64 {
		moan := bendX(part("grizzly_2", pick(v, 0.5, 2.0), pick(v, 4.0, 5.6), 1), falling(0.55, 0.6)...)
		x := mixAt(moan, bubbles(r, dur(moan)), 0, 0.4)
		return underwater(dying(x, r, 0.9), r)
	})

	// The crocodiles: a hiss over a deep alligator growl, as one bursts
	// from the water. The smaller ones play it higher (hellwire.go).
	add("croc_alarm", 2, func(v int, r *rand.Rand) []float64 {
		growl := rattle(lowpass(part(pick(v, "gator_2", "gator_3"), 0, 1.6, 0.9), 600), 13, 0.5, r)
		x := mix(layer{hiss(r, 1.1, 2300), 0, -2}, layer{growl, 0.05, 0})
		return reverb(drive(x, 1.4), 0.6, 0.1, 0.5)
	})
	add("croc_hurt", 2, func(v int, r *rand.Rand) []float64 {
		growl := rattle(lowpass(part(pick(v, "gator_1", "gator_4"), 0, 0.8, 1.05), 700), 15, 0.5, r)
		return reverb(drive(mix(layer{growl, 0, 0}, layer{hiss(r, 0.5, 2600), 0, -5}), 1.6), 0.5, 0.1, 0.5)
	})
	add("croc_death", 2, func(v int, r *rand.Rand) []float64 {
		growl := rattle(lowpass(bend(pick(v, "gator_3", "gator_2"), falling(0.8, 0.65)...), 600), 10, 0.5, r)
		return reverb(dying(mix(layer{growl, 0, 0}, layer{hiss(r, 1.4, 2000), 0.1, -6}), r, 0.85), 0.7, 0.1, 0.5)
	})
}

// bubbles is d seconds of bubbles rising: short sines sweeping up.
func bubbles(r *rand.Rand, d float64) []float64 {
	out := make([]float64, secs(d))
	for range int(d * 14) {
		f := 160 + r.Float64()*380
		t0 := r.Float64() * d
		n := secs(0.05 + r.Float64()*0.05)
		ph := 0.0
		b := make([]float64, n)
		for i := range b {
			u := float64(i) / float64(n)
			ph += f * (1 + 0.9*u) / rate
			b[i] = sin2pi(ph) * (1 - u) * min(1, u*20)
		}
		out = mixAt(out, b, t0, 0.3+0.4*r.Float64())
	}
	return out
}

// --- the great beasts of the north, the cave and the woods -------------------------------

func beastVoices(add adder) {
	// The mammoth: an elephant's trumpet, a little lower, drawn out by an
	// elk's bugle, over a bison's rumble. The Mammoth King: deeper and longer.
	trumpet := func(_ float64, pts ...[2]float64) []float64 {
		return band(bend("elephant_1", pts...), 150, 7000)
	}
	add("mammoth_call", 3, func(v int, r *rand.Rand) []float64 {
		k := pick(v, 0.92, 0.86, 0.98)
		t := trumpet(k, pt{0, k}, pt{0.3, k * 1.04}, pt{1, k * 0.9})
		tail := band(part(pick(v, "elk_1", "elk_5", "elk_7"), 0.9, 2.5, 0.62*k), 200, 6000)
		tail = shape(tail, pt{0, 0}, pt{0.2, 1}, pt{0.6, 0.6}, pt{1, 0})
		low := lowpass(use(pick(v, "bison_a2", "bison_c1", "bison_b2"), 0.7), 900)
		x := mix(layer{t, 0, 0}, layer{tail, 0.35, -8}, layer{low, 0.1, -9})
		return reverb(x, 1.3, 0.2, 0.45)
	})
	add("mammoth_alarm", 2, func(v int, r *rand.Rand) []float64 {
		k := pick(v, 1.0, 0.95)
		t := trumpet(k, pt{0, k * 0.95}, pt{0.2, k * 1.06}, pt{1, k * 0.92})
		x := mix(layer{snort(r, 0.8), 0, -6}, layer{drive(t, 2), 0.12, 0})
		return reverb(x, 1.2, 0.18, 0.45)
	})
	add("mammoth_death", 2, func(v int, r *rand.Rand) []float64 {
		t := trumpet(0.8, falling(pick(v, 0.82, 0.78), 0.55)...)
		horn := band(bendX(part(pick(v, "elk_6", "elk_10"), 0.5, 2.6, 1), falling(0.55, 0.6)...), 150, 5000)
		low := lowpass(bend(pick(v, "bison_c3", "bison_c5"), falling(0.6, 0.6)...), 900)
		x := mix(layer{t, 0, 0}, layer{horn, 0.4, -6}, layer{low, 0.2, -6})
		return reverb(dying(x, r, 0.85), 1.4, 0.2, 0.5)
	})
	add("mammothking_roar", 2, func(v int, r *rand.Rand) []float64 {
		k := pick(v, 0.64, 0.6)
		t := stretch(trumpet(k, pt{0, k}, pt{0.25, k * 1.05}, pt{1, k * 0.86}), 1.4)
		horn := band(stretch(part(pick(v, "elk_10", "elk_4"), 0.5, 2.9, 0.47), 1.2), 150, 5000)
		horn = shape(horn, pt{0, 0}, pt{0.1, 1}, pt{0.7, 0.85}, pt{1, 0})
		low := lowpass(use(pick(v, "bison_c4", "bison_b1"), 0.5), 800)
		x := mix(layer{drive(t, 1.8), 0, 0}, layer{horn, 0.25, -6}, layer{low, 0.15, -4})
		x = mixAt(x, rumble(x, 32, 26), 0, 0.35)
		return reverb(x, 2.0, 0.25, 0.5)
	})
	add("mammothking_special", 2, func(v int, r *rand.Rand) []float64 {
		k := pick(v, 0.72, 0.68)
		t := trumpet(k, pt{0, k * 0.95}, pt{0.2, k * 1.08}, pt{1, k * 0.9})
		return reverb(mix(layer{snort(r, 0.6), 0, -4}, layer{drive(t, 2.2), 0.1, 0}), 1.6, 0.22, 0.5)
	})
	add("mammothking_hurt", 2, func(v int, r *rand.Rand) []float64 {
		t := trumpet(0.7, pt{0, pick(v, 0.7, 0.66)}, pt{0.25, 0.78}, pt{1, 0.6})
		low := lowpass(part(pick(v, "bison_b3", "bison_a3"), 0, 0.9, 0.55), 900)
		return reverb(drive(mix(layer{t, 0, 0}, layer{low, 0, -4}), 1.6), 1.4, 0.22, 0.5)
	})
	add("mammothking_death", 2, func(v int, r *rand.Rand) []float64 {
		t := stretch(trumpet(0.6, falling(pick(v, 0.62, 0.58), 0.55)...), 1.5)
		horn := band(bendX(stretch(part(pick(v, "elk_10", "elk_6"), 0.5, 2.8, 1), 1.2), falling(0.46, 0.55)...), 120, 4000)
		low := lowpass(bend(pick(v, "bison_c5", "bison_c4"), falling(0.46, 0.6)...), 800)
		x := mix(layer{t, 0, 0}, layer{horn, 0.5, -6}, layer{low, 0.3, -4})
		x = mixAt(x, rumble(x, 28, 18), 0, 0.35)
		return reverb(dying(x, r, 0.85), 2.0, 0.25, 0.5)
	})

	// The cave bear: the grizzly's own voice.
	add("bear_growl", 2, func(v int, r *rand.Rand) []float64 {
		x := part("grizzly_2", pick(v, 0.4, 2.8), pick(v, 2.4, 4.8), pick(v, 0.92, 0.88))
		return reverb(rattle(lowpass(x, 3500), 15, 0.4, r), 0.8, 0.12, 0.5)
	})
	add("bear_roar", 2, func(v int, r *rand.Rand) []float64 {
		x := part("grizzly_1", pick(v, 1.3, 2.6), pick(v, 3.4, 4.9), pick(v, 0.9, 0.86))
		low := lowpass(part("grizzly_2", 0.5, 2.5, 0.85), 1200)
		return reverb(drive(mix(layer{lowpass(x, 6000), 0, 0}, layer{low, 0, -6}), 1.5), 1.0, 0.15, 0.45)
	})
	add("bear_hurt", 2, func(v int, r *rand.Rand) []float64 {
		bawl := part(pick(v, "cub_3", "cub_2"), 0, 0.7, 0.62)
		x := mix(layer{drive(bawl, 1.8), 0, 0}, layer{lowpass(part("grizzly_1", pick(v, 1.4, 3.2), pick(v, 2.1, 3.9), 0.9), 5000), 0, -5})
		return reverb(x, 0.7, 0.12, 0.45)
	})
	add("bear_death", 2, func(v int, r *rand.Rand) []float64 {
		x := bendX(part("grizzly_2", pick(v, 0.4, 2.0), pick(v, 3.2, 4.8), 1), falling(0.82, 0.6)...)
		return reverb(dying(lowpass(x, 3500), r, 0.85), 1.0, 0.15, 0.5)
	})
}
