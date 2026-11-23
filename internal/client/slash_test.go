package client

import (
	"math"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestABlowStartsWhenAPlayerStartsActing(t *testing.T) {
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Held: sim.ItemIronSword}
	g.frame = 5
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}})
	if _, ok := g.swingAt[me.ID]; ok {
		t.Fatal("a blow started while standing still")
	}

	me.Acting = true
	g.frame = 9
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}})
	g.frame = 12
	g.onSnapshot(protocol.Snapshot{Tick: 3, Players: []protocol.PlayerView{me}}) // still the same blow
	if got := g.swingAt[me.ID]; got != 9 {
		t.Fatalf("blow began at frame %d, want 9 (when acting started)", got)
	}

	me.Acting = false
	g.onSnapshot(protocol.Snapshot{Tick: 4, Players: []protocol.PlayerView{me}})
	me.Acting = true
	g.frame = 30
	g.onSnapshot(protocol.Snapshot{Tick: 5, Players: []protocol.PlayerView{me}})
	if got := g.swingAt[me.ID]; got != 30 {
		t.Errorf("the next blow began at frame %d, want 30", got)
	}
}

func TestTheBladeComesDownThroughTheBlow(t *testing.T) {
	if math.Abs(slashAngle(0)-slashFrom) > 1e-9 || math.Abs(slashAngle(1)-slashTo) > 1e-9 {
		t.Fatalf("blow runs %v..%v, want %v..%v", slashAngle(0), slashAngle(1), slashFrom, slashTo)
	}
	for k := 0.1; k <= 1; k += 0.1 {
		if slashAngle(k) <= slashAngle(k-0.1) {
			t.Fatalf("the blade goes back up at %.1f of the blow", k)
		}
	}
}
