package client

import (
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func stats(x []float32) (peak, rms float64) {
	sum := 0.0
	for _, v := range x {
		f := float64(v)
		peak = max(peak, math.Abs(f))
		sum += f * f
	}
	return peak, math.Sqrt(sum / float64(max(1, len(x))))
}

func checkFinite(t *testing.T, name string, x []float32) {
	t.Helper()
	for i, v := range x {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Fatalf("%s: sample %d is %v", name, i, v)
		}
	}
}

func TestSoundEffectsAreUsable(t *testing.T) {
	t0 := time.Now()
	bank := genSFX()
	t.Logf("sound effects: %v", time.Since(t0))
	for _, name := range []string{
		"chop", "treefall", "stone", "rockbreak", "rustle", "pick_wood", "pick_stone",
		"pick_grass", "pick_berry", "eat", "ignite", "stoke", "fizzle", "craft",
		"step_grass", "step_sand", "step_rock", "step_forest", "hurt", "shiver",
		"death", "respawn", "error", "ui_move", "ui_select", "ui_pause", "save", "bird",
	} {
		vs := bank[name]
		if len(vs) == 0 {
			t.Errorf("%s: missing", name)
			continue
		}
		for _, x := range vs {
			checkFinite(t, name, x)
			peak, rms := stats(x)
			if peak > 1.0001 || peak < 0.5 || rms < 0.005 {
				t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
			}
			if end := math.Abs(float64(x[len(x)-1])); end > 0.01 {
				t.Errorf("%s: ends at %.3f, would click", name, end)
			}
		}
	}
}

// seam compares the jump across a loop's wrap point with the typical jump
// between neighbouring samples; a click shows up as an outlier.
func seam(x []float32, channels int) (jump, typical float64) {
	n := len(x) / channels
	sum := 0.0
	for c := 0; c < channels; c++ {
		for i := 1; i < n; i++ {
			sum += math.Abs(float64(x[i*channels+c] - x[(i-1)*channels+c]))
		}
		jump = max(jump, math.Abs(float64(x[c]-x[(n-1)*channels+c])))
	}
	return jump, sum / float64(channels*(n-1))
}

func TestLoopsAreSeamless(t *testing.T) {
	t0 := time.Now()
	loops := genLoops()
	t.Logf("ambience loops: %v", time.Since(t0))
	for _, name := range []string{"ocean", "fire", "wind", "crickets", "cave"} {
		x := loops[name]
		checkFinite(t, name, x)
		if peak, rms := stats(x); peak < 0.5 || rms < 0.01 {
			t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
		}
		if j, typ := seam(x, 1); j > 8*typ+0.02 {
			t.Errorf("%s: seam jump %.4f vs typical %.4f", name, j, typ)
		}
	}
}

func TestMusicTracks(t *testing.T) {
	for _, name := range []string{"menu", "day", "night", "cave"} {
		t0 := time.Now()
		x := genMusic(name)
		secs := float64(len(x)/2) / sampleRate
		peak, rms := stats(x)
		t.Logf("%s: %.0fs long, rendered in %v, peak %.2f rms %.3f", name, secs, time.Since(t0).Round(time.Millisecond), peak, rms)
		checkFinite(t, name, x)
		if secs < 40 || peak > 0.81 || rms < 0.03 {
			t.Errorf("%s: %.0fs, peak %.3f, rms %.4f", name, secs, peak, rms)
		}
		if j, typ := seam(x, 2); j > 8*typ+0.02 {
			t.Errorf("%s: seam jump %.4f vs typical %.4f", name, j, typ)
		}
	}
}

func TestMixer(t *testing.T) {
	m := &mixer{}
	m.busGain = [numBuses]float32{1, 1}
	m.busTarget = m.busGain
	short := make([]float32, 100)
	for i := range short {
		short[i] = 0.5
	}
	m.add(&voice{data: short, rate: 1, gain: 1, target: 1})
	m.add(&voice{data: short, rate: 1, gain: 1, target: 1, loop: true})

	buf := make([]byte, 8*256)
	n, err := m.Read(buf)
	if err != nil || n != len(buf) {
		t.Fatalf("Read = %d, %v", n, err)
	}
	if len(m.voices) != 1 || !m.voices[0].loop {
		t.Fatalf("voices after one buffer = %d, want only the loop", len(m.voices))
	}
	for i := 0; i < len(buf); i += 4 {
		v := math.Float32frombits(binary.LittleEndian.Uint32(buf[i:]))
		if math.IsNaN(float64(v)) || v > 1 || v < -1 {
			t.Fatalf("sample %d = %v", i/4, v)
		}
	}
	// A partial frame is never returned.
	if n, _ := m.Read(make([]byte, 13)); n != 8 {
		t.Errorf("Read of 13 bytes = %d, want 8", n)
	}
}
