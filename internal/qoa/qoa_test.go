package qoa

import (
	"math"
	"math/rand/v2"
	"testing"
)

// A sound comes back from Encode and Decode close to what went in, at its
// length and rate, across frame boundaries.
func TestRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	const rate = 44100
	x := make([]int16, frameLen*2+777)
	ph := 0.0
	for i := range x {
		f := 110 + 900*float64(i)/float64(len(x))
		ph += f / rate
		v := 0.5*math.Sin(2*math.Pi*ph) + 0.2*math.Sin(6*math.Pi*ph) + 0.05*(r.Float64()*2-1)
		x[i] = int16(v * 20000)
	}
	y, gotRate, err := Decode(Encode(x, rate))
	if err != nil {
		t.Fatal(err)
	}
	if gotRate != rate || len(y) != len(x) {
		t.Fatalf("rate %d, %d samples; want %d, %d", gotRate, len(y), rate, len(x))
	}
	var sig, noise float64
	for i := range x {
		sig += float64(x[i]) * float64(x[i])
		d := float64(x[i]) - float64(y[i])
		noise += d * d
	}
	if snr := 10 * math.Log10(sig/noise); snr < 25 {
		t.Errorf("SNR %.1f dB, want at least 25", snr)
	}
}

func TestDecodeRejectsJunk(t *testing.T) {
	for _, b := range [][]byte{nil, []byte("RIFF....WAVE"), append([]byte("qoaf"), 0, 0, 0, 9)} {
		if _, _, err := Decode(b); err == nil {
			t.Errorf("%q: no error", b)
		}
	}
}
