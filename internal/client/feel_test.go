package client

import (
	"fmt"
	"strings"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// feelSnapshot is the part of onSnapshot that the feel of a fight reads:
// what came before, the blows begun, and the stops, flashes and effects of
// noteFeel.
func (g *Game) feelSnapshot(s protocol.Snapshot) {
	old, had := g.curr, g.hasCurr
	if g.prevPlayers == nil {
		g.prevPlayers = map[sim.PlayerID]protocol.PlayerView{}
		g.prevCreatures = map[sim.CreatureID]protocol.CreatureView{}
	}
	clear(g.prevPlayers)
	for _, p := range old.Players {
		g.prevPlayers[p.ID] = p
	}
	clear(g.prevCreatures)
	for _, c := range old.Creatures {
		g.prevCreatures[c.ID] = c
	}
	g.curr, g.hasCurr = s, true
	g.noteSwings(s)
	if had {
		g.noteFeel(old, s)
	}
	g.noticeSeq = s.Self.NoticeSeq
}

// TestHitStopLeavesTheSimAlone plays the same fight twice on a real server,
// the same input sent on the same ticks: once with nobody watching, once
// with a client taking every snapshot, its blows stopping the picture all
// the while. The world must come out the same: the hit-stop is the
// client's alone.
func TestHitStopLeavesTheSimAlone(t *testing.T) {
	run := func(watch bool) (string, int) {
		s, err := server.New(server.Config{Seed: 7, Width: 64, Height: 64, Dev: true, Style: sim.StyleIsland})
		if err != nil {
			t.Fatal(err)
		}
		conn, err := s.ConnectLocal()
		if err != nil {
			t.Fatal(err)
		}
		send := func(m protocol.ClientMsg) {
			if err := conn.Send(m); err != nil {
				t.Fatal(err)
			}
		}
		send(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: "a"}})
		s.Tick()
		for _, line := range []string{"spawn raptor", "spawn raptor", "spawn rabbit"} {
			send(protocol.ClientMsg{Kind: protocol.ClientDev, Dev: line})
		}
		g := &Game{}
		var last protocol.Snapshot
		for tick := 0; tick < 240; tick++ {
			in := sim.Input{Action: tick%8 < 5}
			if tick%60 >= 40 {
				in.MoveX = 1
			}
			send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: in})
			s.Tick()
			for drained := false; !drained; {
				select {
				case m := <-conn.Messages():
					switch m.Kind {
					case protocol.ServerWelcome:
						g.myID, g.width, g.height, g.tiles = m.Welcome.PlayerID, m.Welcome.Width, m.Welcome.Height, m.Welcome.Tiles
					case protocol.ServerSnapshot:
						last = m.Snapshot
						if watch {
							g.feelSnapshot(m.Snapshot)
							g.stopFor(stopMax) // and as long a stop as there is, at every snapshot
						}
					}
				default:
					drained = true
				}
			}
			if watch {
				for range 60 / sim.TicksPerSecond {
					g.frame++
					g.tickStop()
				}
			}
		}
		var b strings.Builder
		fmt.Fprintf(&b, "tick %d hp %.2f\n", last.Tick, last.Self.Health)
		for _, p := range last.Players {
			fmt.Fprintf(&b, "player %d %.4f %.4f %d\n", p.ID, p.X, p.Y, p.HurtSeq)
		}
		for _, c := range last.Creatures {
			fmt.Fprintf(&b, "%d %d %.4f %.4f %d %d\n", c.ID, c.Kind, c.X, c.Y, c.HP, c.State)
		}
		return b.String(), g.stop.total
	}
	alone, _ := run(false)
	watched, stops := run(true)
	if stops == 0 {
		t.Fatal("the watched fight never stopped the picture: the test proves nothing")
	}
	if alone != watched {
		t.Errorf("the world went differently with the hit-stops:\nalone:\n%s\nwatched:\n%s", alone, watched)
	}
}

func TestHitStopIsTwoToFiveFrames(t *testing.T) {
	if n := blowStop(1, sim.ItemNone, false, false); n != stopMin {
		t.Errorf("a light blow stops %d frames, want %d", n, stopMin)
	}
	if n := blowStop(90, sim.ItemBattleAxe, true, true); n != stopMax {
		t.Errorf("the heaviest blow stops %d frames, want %d", n, stopMax)
	}
	if blowStop(30, sim.ItemNone, true, false) <= blowStop(30, sim.ItemNone, false, false) {
		t.Error("a critical blow should stop longer")
	}
	g := &Game{}
	g.stopFor(9)
	if g.stop.left != stopMax || !g.stop.capture {
		t.Fatalf("a stop of 9 asked for holds %d frames: at most %d", g.stop.left, stopMax)
	}
	g.stop.capture = false // the frame drawn
	for range stopMax {
		g.tickStop()
	}
	if g.stop.left != 0 {
		t.Errorf("%d frames left after %d", g.stop.left, stopMax)
	}
}

func TestScreenShakeFollowsTheSetting(t *testing.T) {
	keep := screenShake
	defer func() { screenShake = keep }()
	screenShake = true
	addTrauma(1)
	if dx, dy := shake(0); dx == 0 && dy == 0 {
		t.Error("full trauma should shake the view")
	}
	screenShake = false
	if dx, dy := shake(0); dx != 0 || dy != 0 {
		t.Error("with screen shake off nothing shakes")
	}
	shakeState.trauma = 0
}
