package netplay

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
)

// maxConns bounds the connections held at once, handshaken or not, so a
// flood of idle sockets cannot use up the host's file handles.
const maxConns = 64

// helloLimit is how long a connection that has shaken hands may go without
// saying Hello before it is let go.
const helloLimit = 10 * time.Second

// Listener puts a running server on the network. Every accepted connection
// is a client of the same server the host plays on.
type Listener struct {
	srv *server.Server
	ln  net.Listener

	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	closed bool
	wg     sync.WaitGroup

	in, out counter // bytes received from and sent to clients, all connections
}

// Listen starts accepting clients for srv on addr ("host:port", or ":7777"
// for every interface). The server must be running, or about to be: clients
// that arrive before Run starts wait for its first tick.
func Listen(srv *server.Server, addr string) (*Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	l := &Listener{srv: srv, ln: ln, conns: map[net.Conn]struct{}{}}
	l.wg.Add(1)
	go l.acceptLoop()
	return l, nil
}

// Addr is where the listener is bound.
func (l *Listener) Addr() net.Addr { return l.ln.Addr() }

// Port is the TCP port the listener is bound to.
func (l *Listener) Port() int {
	if a, ok := l.ln.Addr().(*net.TCPAddr); ok {
		return a.Port
	}
	return 0
}

// Traffic is how many bytes have been received from and sent to clients
// since Listen, handshakes and keepalives included.
func (l *Listener) Traffic() (in, out int64) { return l.in.get(), l.out.get() }

// Close stops accepting and hangs up on every client; it returns once their
// goroutines have finished. The server itself is not touched.
func (l *Listener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	conns := make([]net.Conn, 0, len(l.conns))
	for c := range l.conns {
		conns = append(conns, c)
	}
	l.mu.Unlock()
	err := l.ln.Close()
	for _, c := range conns {
		_ = c.Close()
	}
	l.wg.Wait()
	return err
}

func (l *Listener) acceptLoop() {
	defer l.wg.Done()
	for {
		c, err := l.ln.Accept()
		if err != nil {
			l.mu.Lock()
			closed := l.closed
			l.mu.Unlock()
			if closed || !isTemporary(err) {
				if !closed {
					log.Printf("netplay: accept: %v", err)
				}
				return
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		l.mu.Lock()
		if l.closed || len(l.conns) >= maxConns {
			l.mu.Unlock()
			_ = c.Close()
			continue
		}
		l.conns[c] = struct{}{}
		l.wg.Add(1)
		l.mu.Unlock()
		go func() {
			defer l.wg.Done()
			defer func() {
				l.mu.Lock()
				delete(l.conns, c)
				l.mu.Unlock()
				_ = c.Close()
			}()
			l.serve(c)
		}()
	}
}

func isTemporary(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// serve runs one client: handshake, then relay in both directions until
// either side is gone.
func (l *Listener) serve(nc net.Conn) {
	tuneTCP(nc)
	_ = nc.SetDeadline(time.Now().Add(handshakeLimit))
	var hdr [6]byte
	if _, err := io.ReadFull(deadlineReader{c: nc, idle: handshakeLimit, n: &l.in}, hdr[:]); err != nil || string(hdr[:4]) != magic {
		return // not us: say nothing
	}
	w := deadlineWriter{c: nc, n: &l.out}
	if v := int(binary.BigEndian.Uint16(hdr[4:])); v != protocol.Version {
		_ = writeReply(w, statusVersion, versionMessage(protocol.Version, v))
		linger(nc)
		return
	}
	lc, err := l.srv.ConnectRemote()
	if err != nil {
		_ = writeReply(w, statusRefused, "Phòng đã đóng.")
		linger(nc)
		return
	}
	if err := writeReply(w, statusOK, ""); err != nil {
		_ = lc.Close()
		return
	}
	_ = nc.SetDeadline(time.Time{})

	var closing atomic.Bool
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		l.writeLoop(nc, lc.Messages())
		// The server dropped us (or we failed): let the client read what is
		// left, then see the end; if it does not hang up, the deadline does.
		closing.Store(true)
		if tc, ok := nc.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		_ = nc.SetReadDeadline(time.Now().Add(2 * time.Second))
	}()

	br := bufio.NewReader(deadlineReader{c: nc, idle: idleTimeout, n: &l.in})
	var buf []byte
	sawHello, helloBy := false, time.Now().Add(helloLimit)
	for !closing.Load() {
		var err error
		buf, err = readFrame(br, buf)
		if err != nil {
			break
		}
		if !sawHello && time.Now().After(helloBy) {
			break // keepalives alone do not hold a place in the room
		}
		if len(buf) == 0 {
			continue // keepalive
		}
		m, err := decodeClient(buf)
		if err != nil {
			break
		}
		sawHello = sawHello || m.Kind == protocol.ClientHello
		if err := lc.Send(m); errors.Is(err, server.ErrConnClosed) {
			break
		} // a full inbox just drops the message: the client is flooding
	}
	_ = lc.Close() // the server removes the player on its next tick...
	select {
	case <-writerDone:
	case <-time.After(writeTimeout + time.Second):
	}
	// ...and closing its channel ended the writer.
}

// writeLoop sends everything the server queues for the client. When the
// client reads slower than the server produces, the snapshots that pile up
// are skipped but the newest: the next one supersedes them anyway, and the
// client is not made to catch up on a past that is gone.
func (l *Listener) writeLoop(nc net.Conn, msgs <-chan protocol.ServerMsg) {
	enc := newDownEncoder(deadlineWriter{c: nc, n: &l.out})
	var batch []protocol.ServerMsg
	for {
		m, ok := <-msgs
		if !ok {
			return
		}
		batch = append(batch[:0], m)
		open := true
	drain:
		for {
			select {
			case m, ok := <-msgs:
				if !ok {
					open = false
					break drain
				}
				batch = append(batch, m)
			default:
				break drain
			}
		}
		batch = newestSnapshotOnly(batch)
		for _, m := range batch {
			if err := enc.send(m); err != nil {
				_ = nc.Close() // the reader sees it and cleans up
				for range msgs {
				} // let the server's channel drain so it never blocks on us
				return
			}
		}
		clear(batch)
		if !open {
			return
		}
	}
}

// newestSnapshotOnly drops every snapshot in b but the last, keeping the
// order of everything else.
func newestSnapshotOnly(b []protocol.ServerMsg) []protocol.ServerMsg {
	last := -1
	for i, m := range b {
		if m.Kind == protocol.ServerSnapshot {
			last = i
		}
	}
	out := b[:0]
	for i, m := range b {
		if m.Kind != protocol.ServerSnapshot || i == last {
			out = append(out, m)
		}
	}
	return out
}

// linger lets a refused client read the reason before the socket closes: a
// close with unread input would reset the connection and could discard it.
func linger(nc net.Conn) {
	if tc, ok := nc.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	_ = nc.SetReadDeadline(time.Now().Add(time.Second))
	_, _ = io.Copy(io.Discard, io.LimitReader(nc, 4096))
}

func tuneTCP(nc net.Conn) {
	if tc, ok := nc.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(15 * time.Second)
	}
}
