package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// srcDir holds the clips cut from the recordings, the raw material of the
// voices; outDir the voices made from them, which the game embeds.
const (
	srcDir = "assets/voices/src"
	outDir = "internal/client/voices"
)

// trim cuts every clip out of its recording in raw, cleans it and writes it
// to srcDir. With only set, just the clips named.
func trim(raw string, only map[string]bool) error {
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}
	type loaded struct{ x, nf []float64 }
	cache := map[string]loaded{}
	for _, c := range clips {
		if len(only) > 0 && !only[c.name] {
			continue
		}
		l, ok := cache[c.src]
		if !ok {
			x, err := load(filepath.Join(raw, c.src))
			if err != nil {
				return err
			}
			// The background is measured over the whole recording, so a
			// clip that is all call is not taken for noise.
			l = loaded{x, noiseProfile(highpass(x, 25))}
			cache[c.src] = l
		}
		y := cut(l.x, c.t0, c.t1)
		y = highpass(y, 25)
		if c.clean > 0 {
			y = denoiseWith(y, l.nf, c.clean, 0.03)
		}
		if c.hp > 0 {
			y = highpass(y, c.hp)
		}
		if c.lp > 0 {
			y = lowpass(y, c.lp)
		}
		y = norm(fade(y, 0.01, 0.03), 0.9)
		if err := writeQOA(filepath.Join(srcDir, c.name+".qoa"), y, rate); err != nil {
			return err
		}
		fmt.Printf("%-14s %5.2f s  from %s %.2f–%.2f\n", c.name, dur(y), c.src, c.t0, c.t1)
	}
	return nil
}

// srcCache keeps the clips loaded for the recipes.
var srcCache = map[string][]float64{}

// src is the clip name, from srcDir.
func src(name string) []float64 {
	if x, ok := srcCache[name]; ok {
		return x
	}
	x, err := load(filepath.Join(srcDir, name+".qoa"))
	if err != nil {
		panic(err)
	}
	srcCache[name] = x
	return x
}
