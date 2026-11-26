package client

import (
	"embed"
	"io/fs"
	"log"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"

	"survivalgame/internal/qoa"
)

// The creatures' voices are made from recordings of real animals, the way
// films make their dinosaurs: lions, bison, elk, bears, seals, cranes and
// more, slowed, pitched, filtered and laid over one another. They are built
// by cmd/voicebuild from the clips in assets/voices/src (whose sources and
// licences are in assets/voices/SOURCES.md) and kept here in QOA, a small
// lossy format quick to decode: voices/<species>_<event>_<n>.qoa, the n-th
// variant of a voice. The great beasts' voices, whose sound lies all below
// 8 kHz, are kept at half the game's sample rate and played at half speed.
//
// They are decoded at start-up on a goroutine of their own, alongside the
// synthesised sounds (sfx.go). The synthesised voices stay in the bank: if
// a recorded voice will not load, the game plays the synthesised voice it
// replaced instead (voiceFallback).

//go:embed voices/*.qoa
var voiceFiles embed.FS

// recorded is the recorded voices, decoded: the bank's variants of each,
// and the speed each is played at (1 but for those kept at another rate).
type recorded struct {
	sfx   map[string][][]float32
	speed map[string]float64
}

// goRecordedVoices decodes the recorded voices on a goroutine; they come
// down the channel when done.
func goRecordedVoices() <-chan recorded {
	ch := make(chan recorded, 1)
	go func() { ch <- loadRecordedVoices(voiceFiles) }()
	return ch
}

// loadRecordedVoices decodes every voice in fsys. A file that will not
// decode is left out, and logged.
func loadRecordedVoices(fsys fs.FS) recorded {
	rec := recorded{sfx: map[string][][]float32{}, speed: map[string]float64{}}
	files, _ := fs.Glob(fsys, "voices/*.qoa")
	type variant struct {
		n int
		x []float32
	}
	// Decoded a few at a time, each file on its own.
	type decoded struct {
		s   []int16
		sr  int
		err error
	}
	dec := make([]decoded, len(files))
	var wg sync.WaitGroup
	next := make(chan int)
	for range 4 {
		wg.Go(func() {
			for i := range next {
				b, err := fs.ReadFile(fsys, files[i])
				if err == nil {
					dec[i].s, dec[i].sr, err = qoa.Decode(b)
				}
				dec[i].err = err
			}
		})
	}
	for i := range files {
		next <- i
	}
	close(next)
	wg.Wait()

	vars := map[string][]variant{}
	for i, f := range files {
		name, n, ok := voiceFileName(path.Base(f))
		if !ok {
			continue
		}
		s, sr, err := dec[i].s, dec[i].sr, dec[i].err
		if err != nil || sr <= 0 || len(s) < 2 {
			log.Printf("audio: %s: %v", f, err)
			continue
		}
		if sp := float64(sr) / sampleRate; rec.speed[name] == 0 {
			rec.speed[name] = sp
		} else if rec.speed[name] != sp {
			log.Printf("audio: %s: %d Hz, unlike its other variants", f, sr)
			continue
		}
		x := make([]float32, len(s))
		for i, v := range s {
			x[i] = float32(v) / 32768
		}
		vars[name] = append(vars[name], variant{n, x})
	}
	for name, vs := range vars {
		slices.SortFunc(vs, func(a, b variant) int { return a.n - b.n })
		for _, v := range vs {
			rec.sfx[name] = append(rec.sfx[name], v.x)
		}
	}
	return rec
}

// voiceFileName splits "rex_roar_2.qoa" into the voice "rex_roar" and its
// variant 2.
func voiceFileName(f string) (string, int, bool) {
	base, ok := strings.CutSuffix(f, ".qoa")
	i := strings.LastIndexByte(base, '_')
	if !ok || i <= 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(base[i+1:])
	if err != nil {
		return "", 0, false
	}
	return base[:i], n, true
}

// --- playing speed ------------------------------------------------------------------

// speeds holds, for the sounds of the bank not played as they are, the
// speed to play them at: the recorded voices kept at half rate, and the
// synthesised voices standing in for recorded ones that would not load.
var speeds = struct {
	sync.RWMutex
	m map[string]float64
}{m: map[string]float64{}}

func setSpeed(name string, s float64) {
	speeds.Lock()
	defer speeds.Unlock()
	if s == 1 {
		delete(speeds.m, name)
	} else {
		speeds.m[name] = s
	}
}

// speedOf is the speed sound name is played at, times the speed asked for.
func speedOf(name string) float64 {
	speeds.RLock()
	defer speeds.RUnlock()
	if s, ok := speeds.m[name]; ok {
		return s
	}
	return 1
}

// addRecordedVoices puts the recorded voices into the bank s, and for each
// voice that did not load, the synthesised one it replaced.
func addRecordedVoices(s map[string][][]float32, rec recorded) {
	for name, vs := range rec.sfx {
		s[name] = vs
		setSpeed(name, rec.speed[name])
	}
	for name, fb := range voiceFallback {
		if len(rec.sfx[name]) > 0 {
			continue
		}
		if len(s[fb.name]) == 0 {
			continue
		}
		log.Printf("audio: no recorded voice %s; playing %s instead", name, fb.name)
		s[name] = s[fb.name]
		setSpeed(name, fb.rate*speedOf(fb.name))
	}
}

// voiceFallback is, for each recorded voice, the synthesised voice played
// in its place if it will not load, and the speed to play it at: what the
// game played before it had recorded voices.
var voiceFallback = map[string]struct {
	name string
	rate float64
}{
	"rex_roar":            {"roar", 1},
	"rex_roar_far":        {"roar_far", 1},
	"rex_growl":           {"rex_growl", 1},
	"rex_hurt":            {"roar", 1.08},
	"rex_death":           {"beast_death", 0.75},
	"spino_roar":          {"roar", 1.12},
	"spino_hurt":          {"bellow_hurt", 0.65},
	"spino_death":         {"beast_death", 0.66},
	"carcharo_roar":       {"roar", 1.06},
	"carcharo_hurt":       {"bellow_hurt", 0.7},
	"carcharo_death":      {"beast_death", 0.7},
	"giganoto_roar":       {"roar", 0.82},
	"giganoto_hurt":       {"bellow_hurt", 0.62},
	"giganoto_death":      {"beast_death", 0.6},
	"trike_call":          {"bellow", 1},
	"trike_alarm":         {"bellow", 1},
	"trike_hurt":          {"bellow_hurt", 1},
	"trike_death":         {"beast_death", 1},
	"toro_call":           {"bellow", 0.85},
	"toro_alarm":          {"bellow", 1},
	"toro_hurt":           {"bellow_hurt", 0.88},
	"toro_death":          {"beast_death", 0.92},
	"stego_call":          {"bellow", 0.92},
	"stego_alarm":         {"bellow", 1},
	"stego_hurt":          {"bellow_hurt", 0.94},
	"stego_death":         {"beast_death", 0.96},
	"brachio_call":        {"hoot", 1},
	"brachio_alarm":       {"bellow", 1},
	"brachio_hurt":        {"bellow_hurt", 0.7},
	"brachio_death":       {"beast_death", 0.72},
	"ankylo_call":         {"bellow", 0.88},
	"ankylo_alarm":        {"bellow", 0.95},
	"ankylo_hurt":         {"bellow_hurt", 0.85},
	"ankylo_death":        {"beast_death", 0.9},
	"thesc_call":          {"honk", 1},
	"thesc_alarm":         {"bellow", 1},
	"thesc_hurt":          {"honk", 1.05},
	"thesc_death":         {"beast_death", 1.45},
	"ornitho_call":        {"croak", 1.55},
	"ornitho_alarm":       {"squeak", 1},
	"ornitho_hurt":        {"honk", 1.4},
	"ornitho_death":       {"honk", 1.2},
	"raptor_call":         {"chirp", 1},
	"raptor_alarm":        {"raptor_screech", 1},
	"raptor_growl":        {"raptor_click", 1},
	"raptor_hurt":         {"raptor_yelp", 1},
	"raptor_death":        {"raptor_yelp", 0.85},
	"raptor_attack":       {"hiss", 1},
	"achero_call":         {"chirp", 1.22},
	"achero_alarm":        {"raptor_screech", 1.2},
	"achero_growl":        {"raptor_click", 1.2},
	"achero_hurt":         {"raptor_yelp", 1.2},
	"achero_death":        {"raptor_yelp", 1.05},
	"achero_attack":       {"hiss", 1.2},
	"troodon_call":        {"troodon_call", 1},
	"troodon_alarm":       {"troodon_shriek", 1},
	"troodon_growl":       {"troodon_click", 1},
	"troodon_hurt":        {"raptor_yelp", 1.4},
	"troodon_death":       {"troodon_shriek", 0.85},
	"troodon_attack":      {"troodon_shriek", 1.15},
	"ptero_call":          {"shriek", 1.1},
	"ptero_alarm":         {"shriek", 1},
	"ptero_hurt":          {"shriek", 1.2},
	"ptero_death":         {"shriek", 0.85},
	"quetzal_call":        {"croak", 1},
	"quetzal_alarm":       {"shriek", 1},
	"quetzal_hurt":        {"croak", 1.2},
	"quetzal_death":       {"croak", 0.8},
	"anzu_call":           {"boom", 1},
	"anzu_roar":           {"screech", 1},
	"anzu_hurt":           {"screech", 1.1},
	"anzu_death":          {"screech", 0.8},
	"plesio_alarm":        {"mosa_rumble", 1.6},
	"plesio_hurt":         {"mosa_rumble", 1.7},
	"plesio_death":        {"beast_death", 1.2},
	"mosa_call":           {"mosa_rumble", 1},
	"mosa_hurt":           {"mosa_rumble", 1.3},
	"mosa_death":          {"mosa_rumble", 0.8},
	"megalodon_hurt":      {"mosa_rumble", 0.8},
	"megalodon_death":     {"mosa_rumble", 0.65},
	"kraken_call":         {"kraken_groan", 1},
	"kraken_hurt":         {"kraken_groan", 1.25},
	"kraken_death":        {"kraken_groan", 0.8},
	"croc_alarm":          {"croc_hiss", 1},
	"croc_hurt":           {"croc_hiss", 1.3},
	"croc_death":          {"croc_hiss", 0.8},
	"mammoth_call":        {"trumpet", 1},
	"mammoth_alarm":       {"trumpet", 1},
	"mammoth_death":       {"beast_death", 0.82},
	"mammothking_roar":    {"trumpet", 0.72},
	"mammothking_special": {"trumpet", 0.8},
	"mammothking_hurt":    {"trumpet", 0.85},
	"mammothking_death":   {"beast_death", 0.62},
	"sabre_roar":          {"sabre_roar", 1},
	"sabre_snarl":         {"sabre_snarl", 1},
	"sabre_growl":         {"sabre_growl", 1},
	"sabre_cower":         {"hiss", 1},
	"sabre_hurt":          {"snarl", 1.15},
	"sabre_death":         {"sabre_roar", 0.8},
	"bear_growl":          {"bear_growl", 1},
	"bear_roar":           {"bear_roar", 1},
	"bear_hurt":           {"bear_roar", 1.2},
	"bear_death":          {"beast_death", 1.35},
}
