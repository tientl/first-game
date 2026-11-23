package client

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// testSound installs a silent sound system that records what it plays.
func testSound(t *testing.T) *[]string {
	t.Helper()
	var played []string
	s := &soundSystem{
		mix:      &mixer{},
		rng:      rand.New(rand.NewPCG(1, 2)),
		sfx:      genSFX(),
		lastPlay: make(map[string]time.Time),
		onPlay:   func(n string) { played = append(played, n) },
	}
	old := snd
	snd = s
	t.Cleanup(func() { snd = old })
	return &played
}

func readyGame() *Game {
	g := New(nil, "")
	g.ready, g.myID, g.width, g.height = true, 1, 32, 32
	g.tiles = make([]sim.Tile, 32*32)
	for i := range g.tiles {
		g.tiles[i] = sim.TileGrass
	}
	return g
}

func TestChoppingDownATreeSounds(t *testing.T) {
	played := testSound(t)
	snd.setListener(10.5, 10.5)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight}
	tree := protocol.EntityView{ID: 7, Kind: sim.EntityTree, TX: 11, TY: 10, Ready: true}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{tree}})

	me.Acting = true
	tree.Ready = false
	s := protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{tree}}
	s.Self.Inv.Add(sim.ItemWood, 2)
	g.onSnapshot(s)

	for _, want := range []string{"chop", "treefall", "pick_wood"} {
		if !slices.Contains(*played, want) {
			t.Errorf("played %v, missing %q", *played, want)
		}
	}
}

func TestEatingHurtAndDeathSounds(t *testing.T) {
	played := testSound(t)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 5.5, Y: 5.5}
	s := protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}}
	s.Self.Health, s.Self.Hunger = 50, 40
	s.Self.Inv[0] = sim.Slot{Item: sim.ItemBerry, Count: 3}
	g.onSnapshot(s)

	s.Tick = 2
	s.Self.Inv[0].Count = 2
	s.Self.Hunger = 52
	g.onSnapshot(s)
	if !slices.Contains(*played, "eat") || slices.Contains(*played, "pick_berry") {
		t.Errorf("after eating played %v, want eat and no pickup", *played)
	}

	s.Tick = 3
	s.Self.Health = 40
	g.onSnapshot(s)
	s.Tick = 4
	s.Self.Health, s.Self.Dead = 0, true
	g.onSnapshot(s)
	if !slices.Contains(*played, "hurt") || !slices.Contains(*played, "death") {
		t.Errorf("played %v, want hurt and death", *played)
	}
}

func TestDemoIsSilentAndFarIsQuiet(t *testing.T) {
	played := testSound(t)
	snd.setListener(0, 0)
	snd.playAt("chop", 40, 40, 1, 0)
	if len(*played) != 0 {
		t.Errorf("a far sound played: %v", *played)
	}
	g := readyGame()
	g.demo = true
	g.onSnapshot(protocol.Snapshot{Tick: 1})
	g.onSnapshot(protocol.Snapshot{Tick: 2, Self: protocol.SelfState{Dead: true}})
	if len(*played) != 0 {
		t.Errorf("the menu backdrop made sounds: %v", *played)
	}
}
