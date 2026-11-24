package client

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestAHeldHoeMarksTheTileInFront(t *testing.T) {
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight, Held: sim.ItemHoe}
	bush := protocol.EntityView{ID: 7, Kind: sim.EntityBerryBush, TX: 10, TY: 11, Ready: true}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{bush}})

	if tx, ty, ok := g.digSpot(); !ok || tx != 11 || ty != 10 {
		t.Fatalf("dig spot (%d, %d, %v), want (11, 10)", tx, ty, ok)
	}
	if tg, _ := g.target(); tg.ok {
		t.Errorf("with a tile to till, Space targets %+v as well", tg)
	}

	// Facing the bush: nothing to till, so Space works on the bush.
	me.Dir = sim.DirDown
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{bush}})
	if _, _, ok := g.digSpot(); ok {
		t.Error("marked the bush's tile for tilling")
	}
	if tg, _ := g.target(); !tg.ok || tg.e.ID != bush.ID {
		t.Errorf("facing the bush, the target is %+v, want the bush", tg)
	}

	// Without the hoe in hand there is no mark.
	me.Dir, me.Held = sim.DirRight, sim.ItemNone
	g.onSnapshot(protocol.Snapshot{Tick: 3, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{bush}})
	if _, _, ok := g.digSpot(); ok {
		t.Error("marked a tile with no hoe in hand")
	}
}
