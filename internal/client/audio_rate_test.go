package client

import (
	"math"
	"math/rand/v2"
	"testing"
)

// A sound played at a pitch at or below zero would run backwards off the
// start of its samples and crash the mixer (thunder once asked for a pitch
// spread of 40). Such a sound is skipped instead.
func TestSoundsAtABadPitchAreSkipped(t *testing.T) {
	s := &soundSystem{
		mix: &mixer{},
		rng: rand.New(rand.NewPCG(1, 2)),
		sfx: map[string][][]float32{"x": {make([]float32, 100)}},
	}
	for _, rate := range []float64{-39, -1, 0, math.NaN(), math.Inf(1)} {
		s.play("x", 1, 0, rate)
	}
	if n := len(s.mix.voices); n != 0 {
		t.Fatalf("%d voices from bad pitches, want none", n)
	}
	s.play("x", 1, 0, 1.2)
	if n := len(s.mix.voices); n != 1 {
		t.Fatalf("%d voices from a good pitch, want 1", n)
	}
}
