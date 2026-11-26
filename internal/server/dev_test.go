package server

import (
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// devReply drains c and returns the dev answer queued among the snapshots.
func devReply(t *testing.T, c *LocalConn) protocol.DevReply {
	t.Helper()
	for {
		select {
		case m, ok := <-c.Messages():
			if !ok {
				t.Fatal("connection closed")
			}
			if m.Kind == protocol.ServerDev {
				return m.Dev
			}
		default:
			t.Fatal("no dev reply queued")
		}
	}
}

func devLine(line string) protocol.ClientMsg {
	return protocol.ClientMsg{Kind: protocol.ClientDev, Dev: line}
}

func TestDevCommandsRejectedOutsideDevMode(t *testing.T) {
	s := newTestServer(t)
	c, id := join(t, s, "Tien")
	send(t, c, devLine("give gỗ 50"))
	send(t, c, devLine("god"))
	s.tick()
	r := devReply(t, c)
	if !r.Rejected || len(r.Lines) == 0 {
		t.Errorf("reply = %+v, want a refusal", r)
	}
	p, _ := s.world.Player(id)
	if p.Inv.Count(sim.ItemWood) != 0 || s.world.DevGod(id) {
		t.Error("a dev command ran on a server not in dev mode")
	}
	if isClosed(c) {
		t.Error("a refused dev command should not cost the connection")
	}
	w, h := s.world.Size()
	if n := sim.ExploredFrom(w, h, s.world.Explored(id)).Count(); n == w*h {
		t.Error("the map is all seen without dev mode")
	}
}

func TestDevCommandsRunInDevMode(t *testing.T) {
	s, err := New(Config{Seed: 42, Width: 64, Height: 64, InterestRadius: 10, Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	c := connect(t, s)
	send(t, c, hello("Dev"))
	s.tick()
	m := next(t, c)
	if m.Kind != protocol.ServerWelcome {
		t.Fatalf("first message kind = %d, want welcome", m.Kind)
	}
	id := m.Welcome.PlayerID
	if n := sim.ExploredFrom(64, 64, m.Welcome.Explored).Count(); n != 64*64 {
		t.Errorf("dev mode: the welcome's map has %d tiles seen, want all", n)
	}
	if !s.world.DevGod(id) {
		t.Error("dev mode: a joining player should be in god mode")
	}

	send(t, c, devLine("give gỗ 50"))
	s.tick()
	if r := devReply(t, c); r.Rejected || len(r.Lines) == 0 {
		t.Errorf("reply = %+v", r)
	}
	p, _ := s.world.Player(id)
	if n := p.Inv.Count(sim.ItemWood); n != 50 {
		t.Errorf("wood = %d, want 50", n)
	}

	send(t, c, devLine("reveal"))
	s.tick()
	if r := devReply(t, c); len(r.Explored) == 0 {
		t.Error("reveal should send the explored map")
	}
}

// freeTileBy is an empty tile of soft ground a couple of tiles from the
// player, to build on.
func freeTileBy(t *testing.T, s *Server, id sim.PlayerID) (int16, int16) {
	t.Helper()
	p, _ := s.world.Player(id)
	px, py := int(p.X), int(p.Y)
	for r := 2; r <= 4; r++ {
		for ty := py - r; ty <= py+r; ty++ {
			for tx := px - r; tx <= px+r; tx++ {
				if _, taken := s.world.EntityAt(tx, ty); !taken && s.world.TileAt(tx, ty).Soft() {
					return int16(tx), int16(ty)
				}
			}
		}
	}
	t.Fatal("no free tile near the player")
	return 0, 0
}

func devBuild(tx, ty int16) protocol.ClientMsg {
	return protocol.ClientMsg{Kind: protocol.ClientCommand, Command: sim.Command{Kind: sim.CmdDevBuild, At: true, TX: tx, TY: ty}}
}

// A server not in dev mode drops free build's commands, refuses to turn it
// on, and never says it is on.
func TestFreeBuildRefusedOutsideDevMode(t *testing.T) {
	s := newTestServer(t)
	c, id := join(t, s, "Tien")
	tx, ty := freeTileBy(t, s, id)
	send(t, c, devBuild(tx, ty))
	send(t, c, devLine("freebuild on"))
	s.tick()
	if r := devReply(t, c); !r.Rejected {
		t.Errorf("freebuild on: %+v, want a refusal", r)
	}
	if _, built := s.world.EntityAt(int(tx), int(ty)); built {
		t.Error("a dev build went through on a server not in dev mode")
	}
	s.world.DevSetFreeBuild(id, true) // even asked of the world directly
	send(t, c, devBuild(tx, ty))
	s.tick()
	if _, built := s.world.EntityAt(int(tx), int(ty)); built || s.world.DevFreeBuild(id) {
		t.Error("free build turned on outside dev mode")
	}
	if snap := latest(t, c); snap.Self.DevBuild {
		t.Error("the snapshot says free build is on")
	}
	if isClosed(c) {
		t.Error("a refused dev build should not cost the connection")
	}
}

// In dev mode free build is on from the start: the snapshot says so and a
// piece goes down from nothing.
func TestFreeBuildOnInDevMode(t *testing.T) {
	s, err := New(Config{Seed: 42, Width: 64, Height: 64, InterestRadius: 10, Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	c, id := join(t, s, "Dev")
	if !s.world.DevFreeBuild(id) {
		t.Fatal("dev mode: free build should start on")
	}
	tx, ty := freeTileBy(t, s, id)
	send(t, c, devBuild(tx, ty)) // piece 0: a wooden wall
	s.tick()
	if e, built := s.world.EntityAt(int(tx), int(ty)); !built || e.Kind != sim.EntityWall {
		t.Errorf("dev build in dev mode: %+v", e)
	}
	if snap := latest(t, c); !snap.Self.DevBuild {
		t.Error("the snapshot should say free build is on")
	}
	send(t, c, devLine("freebuild off"))
	s.tick()
	if r := devReply(t, c); r.Rejected || s.world.DevFreeBuild(id) {
		t.Errorf("freebuild off: %+v", r)
	}
	if snap := latest(t, c); snap.Self.DevBuild {
		t.Error("the snapshot still says free build is on")
	}
}
