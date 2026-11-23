package client

import (
	"image"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestFogThinsTowardsSeenGround(t *testing.T) {
	e := sim.NewExplored(40, 40)
	e.Reveal(20.5, 20.5)
	edge := 20 + sim.RevealRadius // the last seen tile east of the centre
	if !e.Has(edge, 20) || e.Has(edge+1, 20) {
		t.Fatalf("test assumes tile %d is the edge of what is seen", edge)
	}
	for i, want := range []uint8{0, 130, 205, 255, 255} {
		if got := fogAlpha(e, edge+i, 20); got != want {
			t.Errorf("%d tiles past the edge: fog %d, want %d", i, got, want)
		}
	}
}

func TestMapRevealsAroundTheSnapshotPlayer(t *testing.T) {
	g := readyGame() // 32x32
	g.wmap.explored = sim.NewExplored(32, 32)
	g.mapSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{
		{ID: 1, X: 5.5, Y: 5.5},   // me
		{ID: 2, X: 28.5, Y: 28.5}, // someone else: not my map
	}})
	if !g.wmap.explored.Has(5, 5) || g.wmap.explored.Has(28, 28) {
		t.Fatal("want my surroundings seen and the other player's not")
	}
	want := image.Rect(5-fogPad, 5-fogPad, 6+fogPad, 6+fogPad)
	if g.wmap.fogDirty != want {
		t.Errorf("fog to repaint %v, want %v", g.wmap.fogDirty, want)
	}
}

func TestUnknownTilesPaintAsGrass(t *testing.T) {
	if got := mapRamp(sim.NumTiles + 5); len(got) == 0 || got[0] != grassRamp[0] {
		t.Error("a tile kind the map does not know must still get colours")
	}
	for _, m := range mapTiles {
		if len(mapRamp(m.tile)) == 0 || m.label == "" {
			t.Errorf("tile %d has no colours or no legend name", m.tile)
		}
	}
}
