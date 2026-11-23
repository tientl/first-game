package server

import (
	"errors"
	"sync"

	"survivalgame/internal/protocol"
)

// Buffer sizes of an in-process connection. The client sends input only when
// it changes, and drains snapshots every frame, so these are generous.
const (
	localInboxSize  = 256
	localOutboxSize = 32
)

var (
	// ErrConnClosed is returned by Send after Close.
	ErrConnClosed = errors.New("server: connection closed")
	// ErrBackpressure is returned by Send when the server is not keeping up;
	// the message was not queued.
	ErrBackpressure = errors.New("server: send buffer full")
)

// LocalConn is the client end of an in-process connection. It behaves like a
// network connection: send Hello, wait for Welcome, then exchange messages.
type LocalConn struct {
	toServer   chan protocol.ClientMsg // never closed, so Send cannot panic
	fromServer chan protocol.ServerMsg // closed by the server
	gone       chan struct{}           // closed by Close
	closeOnce  sync.Once
}

// ConnectLocal opens an in-process connection to s. The server picks it up on
// its next tick. It fails with ErrServerClosed once Run has returned.
func (s *Server) ConnectLocal() (*LocalConn, error) {
	c := &LocalConn{
		toServer:   make(chan protocol.ClientMsg, localInboxSize),
		fromServer: make(chan protocol.ServerMsg, localOutboxSize),
		gone:       make(chan struct{}),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrServerClosed
	}
	s.pending = append(s.pending, &peer{in: c.toServer, out: c.fromServer, gone: c.gone})
	return c, nil
}

// Send queues m for the server without blocking. It never panics, even after
// the server has gone away.
func (c *LocalConn) Send(m protocol.ClientMsg) error {
	select {
	case <-c.gone:
		return ErrConnClosed
	default:
	}
	select {
	case c.toServer <- m:
		return nil
	default:
		return ErrBackpressure
	}
}

// Messages delivers messages from the server. The channel is closed when the
// server drops this connection or shuts down.
func (c *LocalConn) Messages() <-chan protocol.ServerMsg { return c.fromServer }

// Close disconnects. The server removes the player on its next tick. It is
// safe to call more than once.
func (c *LocalConn) Close() error {
	c.closeOnce.Do(func() { close(c.gone) })
	return nil
}
