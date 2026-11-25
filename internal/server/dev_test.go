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
