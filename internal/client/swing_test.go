package client

import (
	"slices"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// hitOnce shows me idle next to e, then starting a blow at it, and returns
// what was played for the blow.
func hitOnce(t *testing.T, g *Game, e protocol.EntityView, inv ...sim.ItemKind) []string {
	t.Helper()
	played := testSound(t)
	snd.setListener(10.5, 10.5)
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight}
	s := protocol.Snapshot{Tick: g.curr.Tick + 1, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{e}}
	for _, k := range inv {
		s.Self.Inv.Add(k, 1)
		if k == sim.ItemAxe {
			me.HasAxe = true
		}
	}
	s.Players[0] = me
	g.onSnapshot(s)
	*played = nil
	me.Acting = true
	s.Tick++
	s.Players = []protocol.PlayerView{me}
	g.onSnapshot(s)
	return *played
}

func TestEachJobSwingsItsOwnTool(t *testing.T) {
	cases := []struct {
		name  string
		e     sim.EntityKind
		want  swingKind
		sound string
	}{
		{"tree", sim.EntityTree, swingAxe, "chop"},
		{"boulder", sim.EntityBoulder, swingPick, "pick_hit"},
		{"berries", sim.EntityBerryBush, swingHand, "rustle"},
		{"grass", sim.EntityGrassTuft, swingHand, "rustle"},
	}
	for _, c := range cases {
		g := readyGame()
		e := protocol.EntityView{ID: 7, Kind: c.e, TX: 11, TY: 10, Ready: true}
		played := hitOnce(t, g, e, sim.ItemAxe, sim.ItemPickaxe) // both tools in the bag
		if got := g.swingWhat[1].kind; got != c.want {
			t.Errorf("%s: swings %d, want %d", c.name, got, c.want)
		}
		if !slices.Contains(played, c.sound) {
			t.Errorf("%s: played %v, want %q", c.name, played, c.sound)
		}
		if g.swingAt[1] != g.frame {
			t.Errorf("%s: the blow starts at frame %d, the sound at %d", c.name, g.swingAt[1], g.frame)
		}
	}
}

func TestTheBlowLandsWithItsSound(t *testing.T) {
	// The sound plays when the blow starts; the tool comes down from overhead
	// and is level with the target two frames (33 ms) later, then settles.
	if chopAngle(0) != chopRaised || chopAngle(chopStrike) != chopImpact {
		t.Fatalf("angles %.2f → %.2f, want raised %.2f then impact %.2f at frame %g",
			chopAngle(0), chopAngle(chopStrike), chopRaised, chopImpact, chopStrike)
	}
	for f := 0.0; f < chopStrike; f++ {
		if chopAngle(f) >= chopAngle(f+1) {
			t.Fatalf("frame %g: the tool is not coming down", f)
		}
	}
	end := chopStrike + chopBounce + chopReturn
	if end >= sim.ActingTicks*3 { // three frames to a tick
		t.Fatalf("the blow takes %g frames, longer than the working pose", end)
	}
	if chopAngle(end+5) != chopRest {
		t.Fatal("the tool does not come to rest")
	}
}

func TestHoeAndSpear(t *testing.T) {
	g := readyGame()
	farmland := protocol.EntityView{ID: 7, Kind: sim.EntityFarmland, TX: 11, TY: 10, Ready: true}
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight, Held: sim.ItemHoe}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{farmland}})
	me.Acting = true
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Entities: []protocol.EntityView{farmland}})
	if g.swingWhat[1].kind != swingHoe {
		t.Fatalf("swings %d with a hoe in hand, want the hoe", g.swingWhat[1].kind)
	}

	g = readyGame()
	rabbit := protocol.CreatureView{ID: 3, Kind: sim.CreatureRabbit, X: 11.3, Y: 10.5}
	me = protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Dir: sim.DirRight, Held: sim.ItemSpear}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{rabbit}})
	me.Acting = true
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}, Creatures: []protocol.CreatureView{rabbit}})
	if g.swingWhat[1].kind != swingThrust {
		t.Fatalf("swings %d with a spear at a rabbit, want a thrust", g.swingWhat[1].kind)
	}
}
