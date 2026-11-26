package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// voice is one sound the game plays, in variants: <name>_<n>.qoa in outDir.
type voice struct {
	name string // <species>_<event>
	n    int    // variants
	make func(v int, r *rand.Rand) []float64
	peak float64 // the variants' peak once mastered (0: 0.95)
}

// uses records the clips each voice is made of, for SOURCES.md.
var uses = map[string]map[string]bool{}

// current is the voice being made, so src can note what it uses.
var current string

// render makes every voice (or those whose names start with one of only)
// and writes them to outDir, with a line for each.
func render(only []string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	total, bytes := 0.0, 0
	for _, vc := range voices() {
		if len(only) > 0 && !slices.ContainsFunc(only, func(p string) bool { return strings.HasPrefix(vc.name, p) }) {
			continue
		}
		current = vc.name
		for v := range vc.n {
			r := rand.New(rand.NewPCG(uint64(len(vc.name))*7919+uint64(v), 0x5eed))
			x := vc.make(v, r)
			p := vc.peak
			if p == 0 {
				p = 0.95
			}
			x = level(master(capLen(x, maxLen(vc.name)), p), vc.name, p)
			sr := rate
			if lowVoice(vc.name) {
				// Its sound is all below 8 kHz: half the samples hold it.
				m := peakOf(x)
				x = norm(resample(lowpass(x, 8000), 2), m)
				sr = rate / 2
			}
			path := filepath.Join(outDir, fmt.Sprintf("%s_%d.qoa", vc.name, v+1))
			if err := writeQOA(path, x, sr); err != nil {
				return err
			}
			st, _ := os.Stat(path)
			bytes += int(st.Size())
			total += float64(len(x)) / float64(sr)
			fmt.Printf("%-22s %5.2f s  peak %5.1f  loud %5.1f dB\n", filepath.Base(path), float64(len(x))/float64(sr), db(peakOf(x)), db(loudest(x, 0.05)))
		}
	}
	current = ""
	fmt.Printf("%.1f s of voices, %.2f MB\n", total, float64(bytes)/(1<<20))
	if len(only) > 0 {
		return nil // the manifest needs every voice
	}
	if err := writeSources(); err != nil {
		return err
	}
	return writeCredits()
}

// --- building blocks ---------------------------------------------------------------

// use is clip name played at speed s (0.5: an octave down, twice as long).
func use(name string, s float64) []float64 {
	if current != "" {
		if uses[current] == nil {
			uses[current] = map[string]bool{}
		}
		uses[current][name] = true
	}
	if s == 1 {
		return slices.Clone(src(name))
	}
	return resample(src(name), s)
}

// part is the clip from t0 to t1 seconds, at speed s.
func part(name string, t0, t1, s float64) []float64 {
	use(name, 1)
	x := cut(src(name), t0, t1)
	x = fade(x, 0.01, 0.04)
	if s != 1 {
		x = resample(x, s)
	}
	return x
}

// bend is the clip at a speed gliding through the points {u, speed}: a
// pitch that rises or sinks along the call.
func bend(name string, pts ...[2]float64) []float64 {
	use(name, 1)
	return varispeed(src(name), line(pts...))
}

// bendX is bend on any sound.
func bendX(x []float64, pts ...[2]float64) []float64 { return varispeed(x, line(pts...)) }

// mix lays sounds on one another: each {x, at seconds, gain dB}.
type layer struct {
	x      []float64
	at, db float64
}

func mix(ls ...layer) []float64 {
	var out []float64
	for _, l := range ls {
		out = mixAt(out, l.x, l.at, math.Pow(10, l.db/20))
	}
	return out
}

// seq plays sounds one after another, each starting gap seconds after the
// last began... or, if gap is negative, -gap seconds after it ended.
func seq(gap float64, xs ...[]float64) []float64 {
	var out []float64
	t := 0.0
	for _, x := range xs {
		out = mixAt(out, x, t, 1)
		if gap >= 0 {
			t += gap
		} else {
			t += dur(x) - gap
		}
	}
	return out
}

// band keeps x between lo and hi Hz (0: no limit).
func band(x []float64, lo, hi float64) []float64 {
	if lo > 0 {
		x = highpass(x, lo)
	}
	if hi > 0 {
		x = lowpass(x, hi)
	}
	return x
}

// underwater is x heard through the sea: dull, wavering, with a long tail.
func underwater(x []float64, r *rand.Rand) []float64 {
	x = lowpass(x, 1100)
	// A slow, deep wobble of pitch, as through moving water.
	ph := r.Float64() * 6
	x = varispeed(x, func(u float64) float64 { return 1 + 0.012*math.Sin(ph+u*dur(x)*2*math.Pi*0.7) })
	x = eq(x, 300, 0.8, 3)
	return reverb(x, 2.2, 0.45, 0.6)
}

// breath is a hot snort or a breath drawn: d seconds of noise through the
// band around f.
func breath(r *rand.Rand, d, f, q float64) []float64 {
	x := filter(noise(r, d), bp, f, q, 0)
	return shape(x, pt{0, 0}, pt{0.15, 1}, pt{0.5, 0.6}, pt{1, 0})
}

// hiss is a reptile's hiss: noise high and wide, rising and dying.
func hiss(r *rand.Rand, d, f float64) []float64 {
	x := band(noise(r, d), f, f*3.5)
	return shape(x, pt{0, 0}, pt{0.1, 1}, pt{0.6, 0.7}, pt{1, 0})
}

// rumble is a deep tone under x following its loudness, gliding from f0 to
// f1 Hz.
func rumble(x []float64, f0, f1 float64) []float64 {
	e := follow(lowpass(x, 300), 60)
	m := peakOf(e)
	if m > 0 {
		for i := range e {
			e[i] /= m
		}
	}
	return subTone(e, line(pt{0, f0}, pt{1, f1}))
}

// clatter is a stork's bill: n dry knocks, slowing.
func clatter(r *rand.Rand, n int, rate float64) []float64 {
	var out []float64
	t := 0.0
	for i := range n {
		k := band(noise(r, 0.02), 900+r.Float64()*600, 3500)
		k = shape(k, pt{0, 1}, pt{0.2, 0.6}, pt{1, 0})
		out = mixAt(out, k, t, 1-0.5*float64(i)/float64(n))
		t += 1 / rate * (1 + 0.6*float64(i)/float64(n)) * (0.85 + 0.3*r.Float64())
	}
	return out
}

type pt = [2]float64

// --- mastering ---------------------------------------------------------------------

// master readies a voice for the game: no rumble below hearing, no silence
// before or after, its loud and quiet parts brought nearer (as a voice heard
// in a game must carry), and its peak at p.
func master(x []float64, p float64) []float64 {
	x = highpass(x, 28)
	x = trimSilence(x, 0.01, 0.02)
	x = compress(x, -14, 2, 0.005, 0.15)
	x = norm(x, 1)
	x = limit(x, 0.89)
	x = fade(x, 0.004, min(0.25, dur(x)/6))
	return norm(x, p)
}

// compress turns down what is above thr dB (from the peak) by ratio, with
// an envelope rising in attack and falling in release seconds.
func compress(x []float64, thr, ratio, attack, release float64) []float64 {
	x = norm(x, 1)
	a, r := math.Exp(-1/(attack*rate)), math.Exp(-1/(release*rate))
	e := 0.0
	out := make([]float64, len(x))
	for i, v := range x {
		s := v * v
		if s > e {
			e = s + (e-s)*a
		} else {
			e = s + (e-s)*r
		}
		lv := 10 * math.Log10(e+1e-12)
		g := 0.0
		if lv > thr {
			g = (thr - lv) * (1 - 1/ratio)
		}
		out[i] = v * math.Pow(10, g/20)
	}
	return out
}

// limit keeps x's peaks under c, looking 2 ms ahead so nothing slips by.
func limit(x []float64, c float64) []float64 {
	look := secs(0.002)
	n := len(x)
	need := make([]float64, n)
	for i, v := range x {
		need[i] = 1
		if a := math.Abs(v); a > c {
			need[i] = c / a
		}
	}
	// The least gain needed over the window ahead.
	g := make([]float64, n)
	for i := range n {
		m := 1.0
		for j := i; j < min(n, i+look); j++ {
			m = min(m, need[j])
		}
		g[i] = m
	}
	rel := math.Exp(-1 / (0.06 * rate))
	cur := 1.0
	out := make([]float64, n)
	for i, v := range x {
		if g[i] < cur {
			cur = g[i]
		} else {
			cur = g[i] + (cur-g[i])*rel
		}
		out[i] = v * cur
	}
	for i, v := range out {
		out[i] = max(-c, min(c, v))
	}
	return out
}

// --- the manifest ------------------------------------------------------------------

// usedBy is, for each clip, the voices made from it.
func usedBy() map[string][]string {
	m := map[string][]string{}
	for v, cs := range uses {
		for c := range cs {
			m[c] = append(m[c], v)
		}
	}
	for _, vs := range m {
		sort.Strings(vs)
	}
	return m
}

// maxLen is the longest a voice may run, in seconds, by what it is: the
// bank is held in memory, and a cry that runs on drowns the next.
func maxLen(name string) float64 {
	big := strings.HasPrefix(name, "giganoto") || strings.HasPrefix(name, "brachio") || strings.HasPrefix(name, "mammothking") ||
		strings.HasPrefix(name, "mosa") || strings.HasPrefix(name, "megalodon") || strings.HasPrefix(name, "kraken")
	var d float64
	switch event := name[strings.LastIndex(name, "_")+1:]; event {
	case "roar", "far":
		d = 4.6
	case "call":
		d = 2.6
	case "death":
		d = 3.6
	case "hurt":
		d = 1.5
	case "growl", "alarm", "special":
		d = 2.4
	default:
		d = 1.2
	}
	if big {
		d *= 1.25
	}
	return d
}

// capLen cuts x to at most d seconds, fading out its last part.
func capLen(x []float64, d float64) []float64 {
	if dur(x) <= d {
		return x
	}
	x = trimSilence(x, 0.01, 0.02) // what is cut is sound, not silence
	if dur(x) <= d {
		return x
	}
	return fade(x[:secs(d)], 0, d*0.3)
}

// lowVoice reports whether voice name belongs to a great beast whose voice
// lies all low: it is kept at half the sample rate, which the game plays
// at half speed (client/realvoice.go).
func lowVoice(name string) bool {
	for _, p := range []string{"rex", "spino", "carcharo", "giganoto", "trike", "toro", "stego", "brachio", "ankylo",
		"mosa", "plesio", "megalodon", "kraken", "croc", "mammoth", "mammothking", "bear"} {
		if strings.HasPrefix(name, p+"_") {
			return true
		}
	}
	return false
}
