package client

import (
	"math"
	"testing"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestBeastCallsAreUsable(t *testing.T) {
	bank := genSFX()
	for _, name := range []string{
		// The synthesised calls; the recorded voices that replaced rex_growl and
		// the sabre-tooth's are tested in realvoice_test.go.
		"roar", "roar_far", "hoot", "chirp", "boom", "croak", "bellow", "trumpet", "snarl",
		"raptor_screech", "raptor_click", "croc_hiss",
	} {
		vs := bank[name]
		if len(vs) == 0 {
			t.Errorf("%s: missing", name)
			continue
		}
		for _, x := range vs {
			checkFinite(t, name, x)
			peak, rms := stats(x)
			if peak > 1.0001 || peak < 0.5 || rms < 0.01 {
				t.Errorf("%s: peak %.3f rms %.4f", name, peak, rms)
			}
			if end := math.Abs(float64(x[len(x)-1])); end > 0.01 {
				t.Errorf("%s: ends at %.3f, would click", name, end)
			}
		}
	}
	if n := len(bank["roar"]); n < 3 {
		t.Errorf("%d roars: a tyrant should not always roar the same", n)
	}
	for _, x := range bank["roar"] {
		if d := float64(len(x)) / sampleRate; d < 2.5 {
			t.Errorf("a roar of %.2fs: it should be long", d)
		}
		// It gathers as a growl before it breaks: the first 0.4s is much
		// quieter than the roar itself.
		_, lead := stats(x[:samples(0.4)])
		_, all := stats(x)
		if lead > all*0.6 {
			t.Errorf("rms %.3f in the lead-in, %.3f overall: the roar should build from a growl", lead, all)
		}
	}
	for k := sim.CreatureKind(1); k < sim.NumCreatureKinds; k++ {
		if v, ok := callOf(k); ok && len(bank[v.name]) == 0 {
			t.Errorf("%s calls with %q, which is not a sound", k.Name(), v.name)
		}
	}
}

// muffled is the share of a sound's energy left after a high-pass at 1 kHz.
func highShare(x []float32) float64 {
	hp := newFilter(highpass, 1000, 0.7)
	all, high := 0.0, 0.0
	for _, v := range x {
		h := hp.do(float64(v))
		all += float64(v) * float64(v)
		high += h * h
	}
	return high / all
}

func TestAFarRoarIsMuffled(t *testing.T) {
	bank := genSFX()
	near, far := highShare(bank["roar"][0]), highShare(bank["roar_far"][0])
	if far >= near/3 {
		t.Fatalf("high end near %.3f, far %.3f: from far off a roar should come muffled", near, far)
	}
}

func snapOf(cs ...protocol.CreatureView) protocol.Snapshot {
	return protocol.Snapshot{Creatures: cs}
}

func TestBeastsCallNowAndThen(t *testing.T) {
	ch := newChorus()
	t0 := time.Unix(1000, 0)
	always := func() float64 { return 0 }
	rex := protocol.CreatureView{ID: 1, Kind: sim.CreatureRex, X: 5, Y: 0, State: sim.CreatureIdle}
	trike := protocol.CreatureView{ID: 2, Kind: sim.CreatureTrike, X: 3, Y: 0, State: sim.CreatureWander}
	snap := snapOf(rex, trike)

	ch.pick(snap, 0, 0, t0, always) // the first call only starts the clock
	call, ok := ch.pick(snap, 0, 0, t0.Add(100*time.Millisecond), always)
	if !ok || call.c.ID != rex.ID || call.voice.name != "rex_roar" {
		t.Fatalf("call %+v %v: the tyrant should roar", call, ok)
	}
	if !ch.calling(rex, t0.Add(time.Second)) {
		t.Fatal("a roaring tyrant should be drawn roaring")
	}
	if ch.calling(rex, t0.Add(4*time.Second)) {
		t.Fatal("still roaring long after")
	}
	if _, ok := ch.pick(snap, 0, 0, t0.Add(time.Second), always); ok {
		t.Fatal("another beast called right over the roar")
	}
	call, ok = ch.pick(snap, 0, 0, t0.Add(3*time.Second), always)
	if !ok || call.c.ID != trike.ID {
		t.Fatalf("call %+v %v: the tyrant roared just now; the three-horn should bellow", call, ok)
	}
	if ch.calling(trike, t0.Add(3100*time.Millisecond)) {
		t.Fatal("a walking three-horn is not drawn calling")
	}
	if _, ok := ch.pick(snap, 0, 0, t0.Add(3200*time.Millisecond), func() float64 { return 0.999 }); ok {
		t.Fatal("calls should be rare")
	}
}

func TestBusyOrFarBeastsDoNotCall(t *testing.T) {
	ch := newChorus()
	t0 := time.Unix(1000, 0)
	always := func() float64 { return 0 }
	snap := snapOf(
		protocol.CreatureView{ID: 1, Kind: sim.CreatureRex, State: sim.CreatureEat},
		protocol.CreatureView{ID: 2, Kind: sim.CreatureRaptor, State: sim.CreatureChase},
		protocol.CreatureView{ID: 3, Kind: sim.CreatureRaptor, X: 30, State: sim.CreatureIdle}, // out of earshot
		protocol.CreatureView{ID: 4, Kind: sim.CreatureTrike, State: sim.CreatureIdle, Ridden: true},
		protocol.CreatureView{ID: 5, Kind: sim.CreatureRabbit, State: sim.CreatureIdle},
	)
	ch.pick(snap, 0, 0, t0, always)
	if call, ok := ch.pick(snap, 0, 0, t0.Add(100*time.Millisecond), always); ok {
		t.Fatalf("%s called while busy, ridden or out of earshot", call.c.Kind.Name())
	}
}
