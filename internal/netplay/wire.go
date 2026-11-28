// Package netplay carries the client/server protocol over TCP, so friends can
// play one world together: Listen puts a running server on the network and
// Dial is the client end, satisfying client.Conn like server.LocalConn does.
// The rules are untouched: each remote client is relayed through
// server.ConnectRemote, the same peer path the host's own client uses.
//
// The wire:
//
//	handshake   client: "DINO" + protocol version (uint16, big endian)
//	            server: status byte, its protocol version, a short message
//	            (status statusOK: go on; anything else: the message says why,
//	            then the server hangs up). The handshake is plain bytes, so a
//	            version mismatch is reported clearly even if the message types
//	            themselves changed between the two builds.
//	upstream    frames: uint32 length (at most maxFrame) + one gob ClientMsg.
//	            A zero length is a keepalive. Messages are small and rare (the
//	            client sends input only when it changes), and every frame is
//	            self-contained, so a hostile client cannot make the server
//	            allocate more than maxFrame.
//	downstream  one gob stream of ServerMsg inside one DEFLATE stream, flushed
//	            after every message. Gob sends each type's description once
//	            and the compressor's window spans the whole connection, so a
//	            snapshot that looks like the last one costs very little; the
//	            Welcome (the whole tile map) compresses to a fraction too.
package netplay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"survivalgame/internal/protocol"
)

// DefaultPort is where a room listens unless told otherwise.
const DefaultPort = 7777

const (
	magic          = "DINO"
	statusOK       = 0
	statusVersion  = 1 // protocol versions differ
	statusRefused  = 2 // full, closed, anything else
	maxFrame       = 64 << 10
	maxHandshake   = 512 // the longest message a server may put in its reply
	handshakeLimit = 5 * time.Second

	// keepaliveEvery is how often an idle client says it is still there;
	// idleTimeout is how long either side waits in silence before giving the
	// other up for lost (the server sends a snapshot every tick, so silence
	// from it is real).
	keepaliveEvery = 3 * time.Second
	idleTimeout    = 12 * time.Second
	// writeTimeout bounds one write of up to writeChunk bytes: a client that
	// does not read for this long is dropped, but a slow link still carrying
	// the big Welcome is not.
	writeTimeout = 10 * time.Second
	writeChunk   = 32 << 10
)

// VersionError is returned by Dial when the server runs another protocol
// version: the two builds of the game must match.
type VersionError struct {
	Server, Client int
	Msg            string
}

func (e *VersionError) Error() string { return e.Msg }

// RefusedError is returned by Dial when the server hangs up with a reason
// (room full, closed).
type RefusedError struct{ Msg string }

func (e *RefusedError) Error() string { return e.Msg }

// errBadHandshake is a peer that does not speak this protocol at all.
var errBadHandshake = errors.New("netplay: not a Dino Isles server")

// JoinAddr turns what a player typed into host:port, adding DefaultPort when
// there is none. It accepts "192.168.1.5", "192.168.1.5:7777", a name, and a
// bare IPv6 address.
func JoinAddr(s string) string {
	if host, port, err := net.SplitHostPort(s); err == nil && port != "" && host != "" {
		return s
	}
	// A bare IPv6 address has colons but no brackets.
	if ip := net.ParseIP(s); ip != nil {
		return net.JoinHostPort(s, strconv.Itoa(DefaultPort))
	}
	if len(s) > 2 && s[0] == '[' && s[len(s)-1] == ']' {
		s = s[1 : len(s)-1]
	}
	return net.JoinHostPort(s, strconv.Itoa(DefaultPort))
}

// deadlineReader pushes the read deadline forward before every read, so a
// connection is lost only after a stretch of silence, however long the data.
type deadlineReader struct {
	c    net.Conn
	idle time.Duration
	n    *counter
}

func (r deadlineReader) Read(p []byte) (int, error) {
	_ = r.c.SetReadDeadline(time.Now().Add(r.idle))
	n, err := r.c.Read(p)
	r.n.add(n)
	return n, err
}

// deadlineWriter bounds each write, in chunks, so a stalled peer fails
// instead of holding the sender for ever.
type deadlineWriter struct {
	c net.Conn
	n *counter
}

func (w deadlineWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		chunk := p
		if len(chunk) > writeChunk {
			chunk = chunk[:writeChunk]
		}
		_ = w.c.SetWriteDeadline(time.Now().Add(writeTimeout))
		n, err := w.c.Write(chunk)
		total += n
		w.n.add(n)
		if err != nil {
			return total, err
		}
		p = p[n:]
	}
	return total, nil
}

// readFrame reads one upstream frame into buf and returns it; an empty
// frame is a keepalive.
func readFrame(r io.Reader, buf []byte) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > maxFrame {
		return nil, fmt.Errorf("netplay: frame of %d bytes", n)
	}
	if int(n) > cap(buf) {
		buf = make([]byte, n)
	}
	buf = buf[:n]
	_, err := io.ReadFull(r, buf)
	return buf, err
}

// frame prefixes payload with its length.
func frame(payload []byte) []byte {
	out := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(out, uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

// writeReply is the server's handshake answer.
func writeReply(w io.Writer, status byte, msg string) error {
	if len(msg) > maxHandshake {
		msg = msg[:maxHandshake]
	}
	b := make([]byte, 0, 5+len(msg))
	b = append(b, status)
	b = binary.BigEndian.AppendUint16(b, protocol.Version)
	b = binary.BigEndian.AppendUint16(b, uint16(len(msg)))
	b = append(b, msg...)
	_, err := w.Write(b)
	return err
}

// readReply reads the server's handshake answer.
func readReply(r io.Reader) (status byte, version int, msg string, err error) {
	var hdr [5]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return 0, 0, "", err
	}
	n := int(binary.BigEndian.Uint16(hdr[3:]))
	if n > maxHandshake {
		return 0, 0, "", errBadHandshake
	}
	b := make([]byte, n)
	if _, err = io.ReadFull(r, b); err != nil {
		return 0, 0, "", err
	}
	return hdr[0], int(binary.BigEndian.Uint16(hdr[1:])), string(b), nil
}
