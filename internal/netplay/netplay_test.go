package netplay

import (
	"context"
	"errors"
	"math"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// room is a server running on a loopback port, with its listener.
type room struct {
	srv  *server.Server
	ln   *Listener
	addr string
	stop func()
}

func startRoom(t *testing.T, size int, cfg server.Config) *room {
	t.Helper()
	cfg.Seed, cfg.Width, cfg.Height = 42, size, size
	cfg.Style = sim.StyleLatest
	srv, err := server.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()
	ln, err := Listen(srv, "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	r := &room{srv: srv, ln: ln, addr: ln.Addr().String()}
	var once sync.Once
	r.stop = func() {
		once.Do(func() {
			_ = ln.Close()
			cancel()
			<-done
		})
	}
	t.Cleanup(r.stop)
	return r
}

// player is a client of the room with the latest of everything it was told.
type player struct {
	t    *testing.T
	conn interface {
		Send(protocol.ClientMsg) error
		Messages() <-chan protocol.ServerMsg
		Close() error
	}
	id      sim.PlayerID
	welcome protocol.Welcome
	snap    protocol.Snapshot
	refused string
	closed  bool
}

func (r *room) join(t *testing.T, name string, look sim.Look) *player {
	t.Helper()
	c, err := Dial(context.Background(), r.addr)
	if err != nil {
		t.Fatalf("%s: dial: %v", name, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	p := &player{t: t, conn: c}
	if err := c.Send(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: name, Look: look}}); err != nil {
		t.Fatal(err)
	}
	return p
}

// pump reads messages until stop reports true (checked after each) or the
// timeout passes, and reports whether stop was satisfied.
func (p *player) pump(timeout time.Duration, stop func() bool) bool {
	deadline := time.After(timeout)
	for {
		if stop != nil && stop() {
			return true
		}
		select {
		case m, ok := <-p.conn.Messages():
			if !ok {
				p.closed = true
				return stop != nil && stop()
			}
			switch m.Kind {
			case protocol.ServerWelcome:
				p.welcome, p.id = m.Welcome, m.Welcome.PlayerID
			case protocol.ServerSnapshot:
				p.snap = m.Snapshot
			case protocol.ServerRefused:
				p.refused = m.Refused
			}
		case <-deadline:
			return false
		}
	}
}

func (p *player) welcomed(t *testing.T) {
	t.Helper()
	if !p.pump(10*time.Second, func() bool { return p.id != 0 && p.snap.Tick != 0 }) {
		t.Fatalf("no welcome and snapshot (refused: %q, closed: %v)", p.refused, p.closed)
	}
}

func (p *player) self() (protocol.PlayerView, bool) {
	for _, v := range p.snap.Players {
		if v.ID == p.id {
			return v, true
		}
	}
	return protocol.PlayerView{}, false
}

func (p *player) view(id sim.PlayerID) (protocol.PlayerView, bool) {
	for _, v := range p.snap.Players {
		if v.ID == id {
			return v, true
		}
	}
	return protocol.PlayerView{}, false
}

// The whole co-op story over real TCP: a host and two friends, each seeing
// the others, one friend walking, leaving and coming back as the same
// character.
func TestHostAndTwoClients(t *testing.T) {
	r := startRoom(t, 128, server.Config{})

	// The host plays through the in-process connection.
	hc, err := r.srv.ConnectLocal()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = hc.Close() })
	host := &player{t: t, conn: hc}
	if err := hc.Send(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: "Chủ"}}); err != nil {
		t.Fatal(err)
	}

	look := sim.Look{Female: true, Skin: 3, Hair: sim.HairBun, HairColor: 5}
	a := r.join(t, "Anh", look)
	b := r.join(t, "Bình", sim.Look{})
	host.welcomed(t)
	a.welcomed(t)
	b.welcomed(t)

	w, h := r.srv.RemotePlayers(), 0
	_ = h
	if w != 2 {
		t.Errorf("RemotePlayers = %d, want 2", w)
	}
	if a.welcome.Width != 128 || len(a.welcome.Tiles) != a.welcome.Width*a.welcome.Height {
		t.Fatalf("welcome carries %dx%d with %d tiles", a.welcome.Width, a.welcome.Height, len(a.welcome.Tiles))
	}
	if a.id == b.id || a.id == host.id {
		t.Fatalf("player ids collide: host %d, a %d, b %d", host.id, a.id, b.id)
	}

	// Everyone stands at the spawn: all three see each other, with their names and looks.
	for name, p := range map[string]*player{"host": host, "a": a, "b": b} {
		if !p.pump(5*time.Second, func() bool { return len(p.snap.Players) == 3 }) {
			t.Fatalf("%s sees %d players, want 3", name, len(p.snap.Players))
		}
	}
	if v, ok := b.view(a.id); !ok || v.Name != "Anh" || v.Look != look.Sanitized() {
		t.Fatalf("b sees a as %+v (found %v), want the name and look a joined with", v, ok)
	}

	// A walks east; B and the host watch it go.
	start, _ := a.self()
	if err := a.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: sim.Input{MoveX: 1}}); err != nil {
		t.Fatal(err)
	}
	moved := func(p *player) bool {
		v, ok := p.view(a.id)
		return ok && math.Abs(float64(v.X-start.X)) > 1
	}
	for name, p := range map[string]*player{"b": b, "host": host} {
		if !p.pump(5*time.Second, func() bool { return moved(p) }) {
			t.Fatalf("%s never saw a move away from x=%.1f", name, start.X)
		}
	}
	_ = a.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput})
	a.pump(500*time.Millisecond, nil)
	last, _ := a.self()

	// A leaves; the others stop seeing it, and the server forgets the connection.
	_ = a.conn.Close()
	for name, p := range map[string]*player{"b": b, "host": host} {
		if !p.pump(5*time.Second, func() bool { _, ok := p.view(a.id); return !ok && len(p.snap.Players) == 2 }) {
			t.Fatalf("%s still sees a after it left", name)
		}
	}
	if !waitFor(5*time.Second, func() bool { return r.srv.RemotePlayers() == 1 }) {
		t.Fatalf("RemotePlayers = %d after a left, want 1", r.srv.RemotePlayers())
	}

	// A comes back under the same name with a different look: same character, same place.
	a2 := r.join(t, "Anh", sim.Look{})
	a2.welcomed(t)
	v, ok := a2.self()
	if !ok || v.Look != look.Sanitized() {
		t.Errorf("rejoined look = %+v, want the saved %+v", v.Look, look.Sanitized())
	}
	if math.Hypot(float64(v.X-last.X), float64(v.Y-last.Y)) > 0.6 {
		t.Errorf("rejoined at (%.1f, %.1f), left at (%.1f, %.1f)", v.X, v.Y, last.X, last.Y)
	}
	if !b.pump(5*time.Second, func() bool { _, ok := b.view(a2.id); return ok }) {
		t.Fatal("b does not see a again")
	}
}

func waitFor(d time.Duration, f func() bool) bool {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if f() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return f()
}

// A client of another protocol version is turned away with a clear reason,
// before any game message is exchanged.
func TestVersionMismatchRefused(t *testing.T) {
	r := startRoom(t, 64, server.Config{})

	nc, err := net.Dial("tcp", r.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	hello := append([]byte(magic), 0, byte(protocol.Version+1)) // a build one protocol ahead
	if _, err := nc.Write(hello); err != nil {
		t.Fatal(err)
	}
	status, version, msg, err := readReply(nc)
	if err != nil {
		t.Fatal(err)
	}
	if status != statusVersion || version != protocol.Version || msg == "" {
		t.Fatalf("reply = status %d, version %d, %q; want a version refusal naming %d", status, version, msg, protocol.Version)
	}
	t.Logf("the refusal says: %s", msg)
	if n := r.srv.RemotePlayers(); n != 0 {
		t.Errorf("%d remote players after a refused handshake", n)
	}

	// Garbage that is not our protocol at all is hung up on silently.
	g, err := net.Dial("tcp", r.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	_, _ = g.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	_ = g.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 16)
	if n, err := g.Read(buf); n != 0 || err == nil {
		t.Errorf("garbage got %d bytes back (%v), want a hang-up", n, err)
	}
}

// Dial reports the mismatch as a *VersionError the menu can show: here
// against a stand-in server that speaks protocol.Version+1.
func TestDialReportsVersionError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 6)
		_, _ = c.Read(buf)
		_ = writeReply(c, statusVersion, versionMessage(protocol.Version+7, protocol.Version))
		time.Sleep(200 * time.Millisecond)
	}()
	_, err = Dial(context.Background(), ln.Addr().String())
	var ve *VersionError
	if !errors.As(err, &ve) {
		t.Fatalf("Dial error = %v, want a *VersionError", err)
	}
	if got := Describe(err); got == "" || got != ve.Msg {
		t.Errorf("Describe = %q", got)
	}
}

// A second player of a name already in the room is told so, and the one
// there is untouched.
func TestNameInUseRefused(t *testing.T) {
	r := startRoom(t, 64, server.Config{})
	a := r.join(t, "Anh", sim.Look{})
	a.welcomed(t)
	dup := r.join(t, "Anh", sim.Look{})
	if dup.pump(5*time.Second, func() bool { return dup.refused != "" }); dup.refused == "" {
		t.Fatal("a second Anh was not refused")
	}
	t.Logf("the refusal says: %s", dup.refused)
	if !dup.pump(3*time.Second, func() bool { return dup.closed }) {
		t.Error("the refused connection was not closed")
	}
	a.pump(500*time.Millisecond, nil)
	if _, ok := a.self(); !ok || a.closed {
		t.Error("the first Anh was disturbed")
	}
}

func TestRoomFull(t *testing.T) {
	r := startRoom(t, 64, server.Config{MaxRemote: 1})
	a := r.join(t, "A", sim.Look{})
	a.welcomed(t)
	b := r.join(t, "B", sim.Look{})
	b.pump(5*time.Second, func() bool { return b.refused != "" })
	if b.refused == "" {
		t.Fatal("a second remote player got into a room of one")
	}
}

// A client that stops reading must not hold up the others or the server
// loop: its snapshots are skipped, and its connection is dropped in the end.
func TestStalledClientDoesNotBlockTheRoom(t *testing.T) {
	r := startRoom(t, 64, server.Config{})
	good := r.join(t, "Tốt", sim.Look{})
	good.welcomed(t)

	// A raw client that shakes hands, says Hello and then never reads again.
	nc, err := net.Dial("tcp", r.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	if _, err := nc.Write(append([]byte(magic), 0, byte(protocol.Version))); err != nil {
		t.Fatal(err)
	}
	if st, _, _, err := readReply(nc); err != nil || st != statusOK {
		t.Fatalf("handshake: %d %v", st, err)
	}
	body, _ := encodeClient(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: "Câm"}})
	if _, err := nc.Write(frame(body)); err != nil {
		t.Fatal(err)
	}

	// The room carries on: the good client keeps getting fresh ticks.
	t0 := good.snap.Tick
	if !good.pump(5*time.Second, func() bool { return good.snap.Tick > t0+60 }) {
		t.Fatalf("the room stalled: tick %d -> %d in 5s", t0, good.snap.Tick)
	}
}

// Silence from a client is a lost client: the server hangs up and frees the name.
func TestSilentClientTimesOut(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the idle timeout")
	}
	r := startRoom(t, 64, server.Config{})
	nc, err := net.Dial("tcp", r.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	_, _ = nc.Write(append([]byte(magic), 0, byte(protocol.Version)))
	if st, _, _, err := readReply(nc); err != nil || st != statusOK {
		t.Fatalf("handshake: %d %v", st, err)
	}
	body, _ := encodeClient(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: "Im"}})
	_, _ = nc.Write(frame(body))
	if !waitFor(2*time.Second, func() bool { return r.srv.RemotePlayers() == 1 }) {
		t.Fatal("never joined")
	}
	if !waitFor(idleTimeout+5*time.Second, func() bool { return r.srv.RemotePlayers() == 0 }) {
		t.Fatal("a silent client was never dropped")
	}
}

// Stopping the host closes every client's connection.
func TestHostShutdownClosesClients(t *testing.T) {
	r := startRoom(t, 64, server.Config{})
	a := r.join(t, "A", sim.Look{})
	a.welcomed(t)
	r.stop()
	if !a.pump(5*time.Second, func() bool { return a.closed }) {
		t.Fatal("the client was not told the room closed")
	}
}

func TestJoinAddr(t *testing.T) {
	for in, want := range map[string]string{
		"192.168.1.5":       "192.168.1.5:7777",
		"192.168.1.5:9000":  "192.168.1.5:9000",
		"host.example.com":  "host.example.com:7777",
		"host.example:1234": "host.example:1234",
		"::1":               "[::1]:7777",
		"[::1]:5000":        "[::1]:5000",
		"[::1]":             "[::1]:7777",
	} {
		if got := JoinAddr(in); got != want {
			t.Errorf("JoinAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

// Bandwidth, measured: the host and three friends in the lived-in
// world, one of them running, for ten seconds. Logged, and held to a budget
// a home connection can carry at 20 snapshots a second.
func TestBandwidth(t *testing.T) {
	size := 256
	if testing.Short() {
		size = 128
	}
	r := startRoom(t, size, server.Config{})
	names := []string{"An", "Bình", "Chi"}
	var ps []*player
	var joinedAt time.Time
	t0 := time.Now()
	for _, n := range names {
		p := r.join(t, n, sim.Look{})
		ps = append(ps, p)
	}
	for _, p := range ps {
		p.welcomed(t)
	}
	joinedAt = time.Now()
	inJoin, outJoin := r.ln.Traffic()
	t.Logf("%dx%d world: joining 3 clients took %v and %d KB down (%d KB each, the whole map included), %d B up",
		size, size, joinedAt.Sub(t0).Round(time.Millisecond), outJoin/1024, outJoin/1024/3, inJoin)

	// Everyone walks about, so snapshots change; spread them out toward beasts.
	dirs := [][2]int8{{1, 0}, {0, 1}, {-1, 1}}
	for i, p := range ps {
		_ = p.conn.Send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: sim.Input{MoveX: dirs[i][0], MoveY: dirs[i][1]}})
	}
	const span = 10 * time.Second
	in0, out0 := r.ln.Traffic()
	end := time.Now().Add(span)
	creatures, snaps := 0, 0
	for time.Now().Before(end) {
		for _, p := range ps {
			p.pump(50*time.Millisecond, nil)
			creatures += len(p.snap.Creatures)
			snaps++
		}
	}
	in1, out1 := r.ln.Traffic()
	down := float64(out1-out0) / span.Seconds()
	up := float64(in1-in0) / span.Seconds()
	t.Logf("steady state, 3 clients: down %.1f KB/s in all (%.1f KB/s = %.0f kbit/s per client), up %.2f KB/s in all; about %.0f creatures in view",
		down/1024, down/1024/3, down/3*8/1000, up/1024, float64(creatures)/float64(snaps))
	if perClient := down / 3; perClient > 20*1024 {
		t.Errorf("%.1f KB/s per client down is too much for 20 snapshots a second", perClient/1024)
	}
}

// The Welcome of the biggest world, measured: set NETPLAY_BIG=1 (it
// generates a 2048x2048 world, which takes a while and some memory).
func TestWelcomeOfAColossalWorld(t *testing.T) {
	if os.Getenv("NETPLAY_BIG") == "" {
		t.Skip("set NETPLAY_BIG=1 to generate a 2048x2048 world")
	}
	t0 := time.Now()
	r := startRoom(t, 2048, server.Config{})
	t.Logf("world made in %v", time.Since(t0).Round(time.Millisecond))
	t1 := time.Now()
	p := r.join(t, "An", sim.Look{})
	p.welcomed(t)
	_, out := r.ln.Traffic()
	t.Logf("2048x2048: Welcome and first snapshot took %v and %.2f MB on the wire (%d tiles raw)",
		time.Since(t1).Round(time.Millisecond), float64(out)/(1<<20), len(p.welcome.Tiles))
}

// The server end of a client that stops reading: the write times out, the
// connection is closed, and the channel the server fills keeps being emptied,
// so the server loop never waits on it.
func TestWriteLoopGivesUpOnAClientThatDoesNotRead(t *testing.T) {
	old := writeTimeout
	writeTimeout = 200 * time.Millisecond
	defer func() { writeTimeout = old }()

	srvSide, cliSide := net.Pipe() // a pipe has no buffer: a write blocks until the other side reads
	defer cliSide.Close()
	l := &Listener{}
	msgs := make(chan protocol.ServerMsg, 32)
	done := make(chan struct{})
	go func() { l.writeLoop(srvSide, msgs); close(done) }()

	for i := range 5 {
		msgs <- protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: protocol.Snapshot{Tick: uint64(i + 1)}}
	}
	// Keep the way the server does: queue without ever blocking.
	sent := 0
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case msgs <- protocol.ServerMsg{Kind: protocol.ServerSnapshot}:
			sent++
		default:
		}
		time.Sleep(time.Millisecond)
		if sent > 500 {
			break
		}
	}
	if sent <= 32 {
		t.Fatalf("the server could queue only %d messages: the writer stopped emptying the channel", sent)
	}
	_ = srvSide.Close()
	close(msgs)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("writeLoop never returned")
	}
}

// Only the newest snapshot of a backlog is sent, and nothing else is dropped.
func TestNewestSnapshotOnly(t *testing.T) {
	snap := func(n uint64) protocol.ServerMsg {
		return protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: protocol.Snapshot{Tick: n}}
	}
	saved := protocol.ServerMsg{Kind: protocol.ServerSaved}
	got := newestSnapshotOnly([]protocol.ServerMsg{snap(1), snap(2), saved, snap(3), snap(4), saved})
	var kinds []protocol.ServerMsgKind
	var tick uint64
	for _, m := range got {
		kinds = append(kinds, m.Kind)
		if m.Kind == protocol.ServerSnapshot {
			tick = m.Snapshot.Tick
		}
	}
	if len(got) != 3 || tick != 4 || kinds[0] != protocol.ServerSaved || kinds[1] != protocol.ServerSnapshot || kinds[2] != protocol.ServerSaved {
		t.Fatalf("got kinds %v, last tick %d", kinds, tick)
	}
}
