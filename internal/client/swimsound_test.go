package client

import (
	"slices"
	"strings"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestSwimmingSoundsAreUsable(t *testing.T) {
	bank := genSFX()
	for _, name := range []string{"swim_stroke", "swim_kick", "swim_in", "swim_out", "swim_tread"} {
		vs := bank[name]
		if len(vs) == 0 {
			t.Errorf("%s: missing", name)
			continue
		}
		for _, x := range vs {
			checkFinite(t, name, x)
			if peak, rms := stats(x); peak > 1.0001 || peak < 0.3 || rms < 0.005 {
				t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
			}
		}
	}
}

func TestSwimmersStrokeInsteadOfStepping(t *testing.T) {
	played := testSound(t)
	snd.loops, snd.loopWant, snd.loopCur = map[string]*voice{}, map[string]float64{}, map[string]float64{} // frameSounds sets the ambience too
	snd.music, snd.musicCur, snd.musicReady = map[string]*voice{}, map[string]float64{}, make(chan musicTrack)
	g := readyGame()
	for i := range g.tiles {
		g.tiles[i] = sim.TileWater
	}
	tick := uint64(0)
	me := protocol.PlayerView{ID: 1, X: 5.5, Y: 10.5, Dir: sim.DirRight}
	snap := func() {
		tick++
		g.onSnapshot(protocol.Snapshot{Tick: tick, Players: []protocol.PlayerView{me}})
	}
	snap()
	g.frameSounds(1)

	me.Swimming = true // in the water
	snap()
	g.frameSounds(1)
	if !slices.Contains(*played, "swim_in") {
		t.Fatalf("going in played %v, want a splash", *played)
	}

	for f := range 120 { // swimming along
		g.frame++
		if f%3 == 0 {
			me.X += 0.1
			snap()
		}
		g.frameSounds(0.5)
	}
	strokes := 0
	for _, n := range *played {
		if n == "swim_stroke" {
			strokes++
		}
		if strings.HasPrefix(n, "step_") {
			t.Fatalf("a swimmer made footsteps: %v", *played)
		}
	}
	if strokes < 2 {
		t.Fatalf("two seconds of swimming played %v, want strokes", *played)
	}

	me.Swimming = false // climbing out
	snap()
	g.frameSounds(1)
	if !slices.Contains(*played, "swim_out") {
		t.Fatalf("climbing out played %v, want water running off", *played)
	}
}
