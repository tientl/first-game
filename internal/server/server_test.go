package server

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Config{Seed: 42, Width: 64, Height: 64, InterestRadius: 10})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func connect(t *testing.T, s *Server) *LocalConn {
	t.Helper()
	c, err := s.ConnectLocal()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func send(t *testing.T, c *LocalConn, m protocol.ClientMsg) {
	t.Helper()
	if err := c.Send(m); err != nil {
		t.Fatal(err)
	}
}

func hello(name string) protocol.ClientMsg {
	return protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: name}}
}

// next returns the next message already queued for c, failing if none is.
// Tests drive s.tick() by hand, so messages are queued synchronously.
func next(t *testing.T, c *LocalConn) protocol.ServerMsg {
	t.Helper()
	select {
	case m, ok := <-c.Messages():
		if !ok {
			t.Fatal("connection closed")
		}
		return m
	default:
		t.Fatal("no message queued")
		return protocol.ServerMsg{}
	}
}

// latest drains c and returns the newest snapshot.
func latest(t *testing.T, c *LocalConn) protocol.Snapshot {
	t.Helper()
	var snap protocol.Snapshot
	found := false
	for {
		select {
		case m, ok := <-c.Messages():
			if !ok {
				t.Fatal("connection closed")
			}
			if m.Kind == protocol.ServerSnapshot {
				snap, found = m.Snapshot, true
			}
		default:
			if !found {
				t.Fatal("no snapshot queued")
			}
			return snap
		}
	}
}

func isClosed(c *LocalConn) bool {
	for {
		select {
		case _, ok := <-c.Messages():
			if !ok {
				return true
			}
		default:
			return false
		}
	}
}

func join(t *testing.T, s *Server, name string) (*LocalConn, sim.PlayerID) {
	t.Helper()
	c := connect(t, s)
	send(t, c, hello(name))
	s.tick()
	m := next(t, c)
	if m.Kind != protocol.ServerWelcome {
		t.Fatalf("first message kind = %d, want welcome", m.Kind)
	}
	return c, m.Welcome.PlayerID
}

func findPlayer(snap protocol.Snapshot, id sim.PlayerID) (protocol.PlayerView, bool) {
	for _, p := range snap.Players {
		if p.ID == id {
			return p, true
		}
	}
	return protocol.PlayerView{}, false
}

func TestHandshakeAndSnapshots(t *testing.T) {
	s := newTestServer(t)
	c := connect(t, s)
	send(t, c, hello("Tiến"))
	s.tick()

	m := next(t, c)
	if m.Kind != protocol.ServerWelcome {
		t.Fatalf("kind = %d, want welcome", m.Kind)
	}
	if w := m.Welcome; w.Width != 64 || w.Height != 64 || len(w.Tiles) != 64*64 || w.PlayerID == 0 {
		t.Fatalf("welcome = %dx%d, %d tiles, player %d", w.Width, w.Height, len(w.Tiles), w.PlayerID)
	}

	snap := latest(t, c)
	me, ok := findPlayer(snap, m.Welcome.PlayerID)
	if !ok || me.Name != "Tiến" {
		t.Fatalf("own player missing from snapshot: %+v", snap.Players)
	}
	if snap.Self.Health != sim.MaxHealth {
		t.Fatalf("health = %v, want %v", snap.Self.Health, sim.MaxHealth)
	}
	if len(snap.Entities) == 0 {
		t.Fatal("snapshot has no entities around the spawn")
	}
	for _, e := range snap.Entities {
		if math.Abs(float64(e.TX)-float64(me.X)) > 11 || math.Abs(float64(e.TY)-float64(me.Y)) > 11 {
			t.Fatalf("entity at (%d,%d) is outside the interest radius of (%v,%v)", e.TX, e.TY, me.X, me.Y)
		}
	}

	s.tick()
	if next := latest(t, c); next.Tick != snap.Tick+1 {
		t.Fatalf("tick = %d, want %d", next.Tick, snap.Tick+1)
	}
}

func TestInputMovesThePlayer(t *testing.T) {
	s := newTestServer(t)
	c, id := join(t, s, "a")
	start, _ := findPlayer(latest(t, c), id)

	// Walk towards whichever side of the spawn has two land tiles in a row,
	// so the move cannot be cut short by the coast.
	sx, sy := int(start.X), int(start.Y)
	var in sim.Input
	for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirLeft, sim.DirRight} {
		dx, dy := d.Delta()
		if s.world.TileAt(sx+dx, sy+dy).Walkable() && s.world.TileAt(sx+2*dx, sy+2*dy).Walkable() {
			in = sim.Input{MoveX: int8(dx), MoveY: int8(dy)}
			break
		}
	}
	if in == (sim.Input{}) {
		t.Fatal("spawn is boxed in by water")
	}

	send(t, c, protocol.ClientMsg{Kind: protocol.ClientInput, Input: in})
	for range 5 {
		s.tick()
	}
	moved, _ := findPlayer(latest(t, c), id)
	dist := math.Hypot(float64(moved.X-start.X), float64(moved.Y-start.Y))
	if want := 5 * sim.PlayerSpeed; math.Abs(dist-want) > 1e-4 {
		t.Fatalf("moved %.4f tiles in 5 ticks, want %.4f", dist, want)
	}

	send(t, c, protocol.ClientMsg{Kind: protocol.ClientInput}) // stop
	s.tick()
	stopped, _ := findPlayer(latest(t, c), id)
	s.tick()
	if again, _ := findPlayer(latest(t, c), id); again.X != stopped.X || again.Y != stopped.Y {
		t.Fatalf("player kept moving after the stop input: %+v → %+v", stopped, again)
	}
}

func TestProtocolViolationsDisconnect(t *testing.T) {
	s := newTestServer(t)

	early := connect(t, s)
	send(t, early, protocol.ClientMsg{Kind: protocol.ClientInput, Input: sim.Input{MoveX: 1}})

	wrongVersion := connect(t, s)
	send(t, wrongVersion, protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version + 1}})

	unknown := connect(t, s)
	send(t, unknown, hello("x"))
	send(t, unknown, protocol.ClientMsg{Kind: 99})

	s.tick()
	for name, c := range map[string]*LocalConn{"input before hello": early, "wrong version": wrongVersion, "unknown kind": unknown} {
		if !isClosed(c) {
			t.Errorf("%s: connection still open", name)
		}
	}
	if n := len(s.world.Players()); n != 0 {
		t.Fatalf("%d players left in the world, want 0", n)
	}
}

func TestCloseRemovesThePlayer(t *testing.T) {
	s := newTestServer(t)
	a, _ := join(t, s, "a")
	b, idB := join(t, s, "b")

	// Both players stand at the spawn, so each sees the other.
	if _, ok := findPlayer(latest(t, a), idB); !ok {
		t.Fatal("player a does not see player b")
	}

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil { // idempotent
		t.Fatal(err)
	}
	if err := b.Send(hello("b")); !errors.Is(err, ErrConnClosed) {
		t.Fatalf("Send after Close = %v, want ErrConnClosed", err)
	}

	s.tick()
	if _, ok := s.world.Player(idB); ok {
		t.Fatal("closed player still in the world")
	}
	if !isClosed(b) {
		t.Fatal("server did not close the dropped connection")
	}
	if _, ok := findPlayer(latest(t, a), idB); ok {
		t.Fatal("player a still sees the disconnected player b")
	}
}

func TestSlowClientDoesNotBlockTheServer(t *testing.T) {
	s := newTestServer(t)
	c := connect(t, s)
	send(t, c, hello("slow"))

	// The client never reads while the server runs many more ticks than the
	// buffer holds. tick must not block, and the Welcome must survive.
	for range 4 * localOutboxSize {
		s.tick()
	}
	if m := next(t, c); m.Kind != protocol.ServerWelcome {
		t.Fatalf("first message kind = %d, want the welcome to be kept", m.Kind)
	}
	if snap := latest(t, c); snap.Tick == 0 {
		t.Fatal("no snapshot after draining")
	}
}

func TestRunAndShutdown(t *testing.T) {
	s := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	c := connect(t, s)
	send(t, c, hello("a"))
	select {
	case m := <-c.Messages():
		if m.Kind != protocol.ServerWelcome {
			t.Fatalf("kind = %d, want welcome", m.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no welcome from the running server")
	}

	if err := s.Run(ctx); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Run = %v, want ErrAlreadyRunning", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	// The connection is closed (after any snapshots still buffered).
	deadline := time.After(2 * time.Second)
	for open := true; open; {
		select {
		case _, open = <-c.Messages():
		case <-deadline:
			t.Fatal("connection not closed on shutdown")
		}
	}
	if _, err := s.ConnectLocal(); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("ConnectLocal after shutdown = %v, want ErrServerClosed", err)
	}
}

func TestBackpressureIsReported(t *testing.T) {
	s := newTestServer(t)
	c := connect(t, s) // never picked up: no ticks run
	var err error
	for i := 0; i <= localInboxSize && err == nil; i++ {
		err = c.Send(hello("x"))
	}
	if !errors.Is(err, ErrBackpressure) {
		t.Fatalf("Send on a full buffer = %v, want ErrBackpressure", err)
	}
}
