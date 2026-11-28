package server

import (
	"strings"
	"testing"

	"survivalgame/internal/protocol"
)

// refusal drains c and returns the reason of a ServerRefused in it, and
// whether the connection was then closed.
func refusal(c *LocalConn) (why string, closed bool) {
	for {
		select {
		case m, ok := <-c.Messages():
			if !ok {
				return why, true
			}
			if m.Kind == protocol.ServerRefused {
				why = m.Refused
			}
		default:
			return why, false
		}
	}
}

func TestRefusalsSayWhy(t *testing.T) {
	s := newTestServer(t)
	host, _ := join(t, s, "Chủ")

	wrong := connect(t, s)
	send(t, wrong, protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version + 1, Name: "x"}})
	dup := mustRemote(t, s)
	send(t, dup, hello("Chủ")) // the host's own name
	s.tick()

	if why, closed := refusal(wrong); !closed || !strings.Contains(why, "phiên bản") {
		t.Errorf("wrong version: reason %q, closed %v", why, closed)
	}
	if why, closed := refusal(dup); !closed || !strings.Contains(why, "tên") {
		t.Errorf("duplicate name: reason %q, closed %v", why, closed)
	}
	if isClosed(host) {
		t.Error("the host was dropped")
	}
	if n := len(s.world.Players()); n != 1 {
		t.Errorf("%d players, want only the host", n)
	}
}

func mustRemote(t *testing.T, s *Server) *LocalConn {
	t.Helper()
	c, err := s.ConnectRemote()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRemoteLimit(t *testing.T) {
	s, err := New(Config{Seed: 42, Width: 64, Height: 64, InterestRadius: 10, MaxRemote: 2})
	if err != nil {
		t.Fatal(err)
	}
	var friends []*LocalConn
	for _, n := range []string{"a", "b"} {
		c := mustRemote(t, s)
		send(t, c, hello(n))
		friends = append(friends, c)
	}
	s.tick()
	if got := s.RemotePlayers(); got != 2 {
		t.Fatalf("RemotePlayers = %d, want 2", got)
	}
	third := mustRemote(t, s)
	send(t, third, hello("c"))
	local := connect(t, s) // the host's own client does not count against the limit
	send(t, local, hello("Chủ"))
	s.tick()
	if why, closed := refusal(third); !closed || why == "" {
		t.Errorf("third friend: reason %q, closed %v; want refused", why, closed)
	}
	if isClosed(local) {
		t.Error("the host was refused by a full room")
	}

	// A friend leaving frees the place.
	_ = friends[0].Close()
	s.tick()
	if got := s.RemotePlayers(); got != 1 {
		t.Fatalf("RemotePlayers = %d after one left, want 1", got)
	}
	again := mustRemote(t, s)
	send(t, again, hello("c"))
	s.tick()
	if why, closed := refusal(again); closed || why != "" {
		t.Errorf("a place was free but the friend got %q (closed %v)", why, closed)
	}
}

// A friend cannot make the host's save write; the host can.
func TestOnlyTheHostSaves(t *testing.T) {
	s := newTestServer(t)
	s.cfg.SavePath = t.TempDir() + "/save.json"
	friend := mustRemote(t, s)
	send(t, friend, hello("Bạn"))
	host := connect(t, s)
	send(t, host, hello("Chủ"))
	s.tick()
	latest(t, friend)
	latest(t, host)

	send(t, friend, protocol.ClientMsg{Kind: protocol.ClientSave})
	s.tick()
	var got protocol.ServerMsg
	for m := range drainKind(friend, protocol.ServerSaved) {
		got = m
	}
	if got.Saved.OK || got.Saved.Error == "" {
		t.Fatalf("a friend's save request answered %+v, want a refusal", got.Saved)
	}

	send(t, host, protocol.ClientMsg{Kind: protocol.ClientSave})
	s.tick()
	got = protocol.ServerMsg{}
	for m := range drainKind(host, protocol.ServerSaved) {
		got = m
	}
	if !got.Saved.OK {
		t.Fatalf("the host's save answered %+v", got.Saved)
	}
}

func drainKind(c *LocalConn, kind protocol.ServerMsgKind) <-chan protocol.ServerMsg {
	out := make(chan protocol.ServerMsg, 64)
	for {
		select {
		case m, ok := <-c.Messages():
			if !ok {
				close(out)
				return out
			}
			if m.Kind == kind {
				out <- m
			}
		default:
			close(out)
			return out
		}
	}
}

// The host's pause stops the clock only while nobody is in the room.
func TestHostPauseDoesNotFreezeFriends(t *testing.T) {
	s := newTestServer(t)
	host, _ := join(t, s, "Chủ")
	_ = host
	s.SetPaused(true)
	t0 := s.world.Tick()
	s.tick()
	if s.world.Tick() != t0 {
		t.Fatalf("a paused solo game moved from tick %d to %d", t0, s.world.Tick())
	}

	friend := mustRemote(t, s)
	send(t, friend, hello("Bạn"))
	s.tick() // the friend joins; the world runs on
	t1 := s.world.Tick()
	s.tick()
	if s.world.Tick() <= t1 {
		t.Fatalf("the host's pause froze the world for a friend: tick %d then %d", t1, s.world.Tick())
	}

	_ = friend.Close()
	s.tick() // the friend is gone
	t2 := s.world.Tick()
	s.tick()
	if s.world.Tick() != t2 {
		t.Fatalf("the pause did not hold once the friend left: tick %d then %d", t2, s.world.Tick())
	}
}
