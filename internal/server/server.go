// Package server runs the authoritative simulation and exchanges messages
// with clients.
//
// Single player uses the same server as co-op: the client process starts a
// Server in a goroutine and talks to it through ConnectLocal, so the game
// never has a code path that skips the client/server boundary.
package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Defaults for Config fields left at zero.
const (
	DefaultWorldSize      = 128
	DefaultInterestRadius = 32 // tiles; covers a 2560px-wide screen
)

// maxMsgsPerTick bounds how many messages one client may have handled per
// tick, so a flooding client cannot stall the loop.
const maxMsgsPerTick = 64

var (
	// ErrServerClosed is returned by ConnectLocal after Run has returned.
	ErrServerClosed = errors.New("server: closed")
	// ErrAlreadyRunning is returned by a second call to Run.
	ErrAlreadyRunning = errors.New("server: Run called twice")
)

// Config configures a Server.
type Config struct {
	Seed           uint64
	Width, Height  int // in tiles; 0 means DefaultWorldSize
	InterestRadius int // clients receive what is within this many tiles; 0 means default
}

// Server owns the world and the connected clients. All game state is touched
// only by the goroutine inside Run.
type Server struct {
	cfg   Config
	world *sim.World

	mu      sync.Mutex
	pending []*peer // connections waiting to be picked up by the loop
	closed  bool

	running atomic.Bool
	peers   []*peer // owned by the Run goroutine
}

// peer is the server's end of one client connection.
type peer struct {
	in   <-chan protocol.ClientMsg
	out  chan<- protocol.ServerMsg // closed by the server when it drops the peer
	gone <-chan struct{}           // closed by the client side on disconnect

	joined bool // Hello accepted, player exists
	player sim.PlayerID
}

// New generates the world and returns a server ready to Run.
func New(cfg Config) (*Server, error) {
	if cfg.Width == 0 {
		cfg.Width = DefaultWorldSize
	}
	if cfg.Height == 0 {
		cfg.Height = DefaultWorldSize
	}
	if cfg.InterestRadius <= 0 {
		cfg.InterestRadius = DefaultInterestRadius
	}
	w, err := sim.Generate(sim.GenConfig{Seed: cfg.Seed, Width: cfg.Width, Height: cfg.Height})
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, world: w}, nil
}

// Run steps the simulation at sim.TicksPerSecond until ctx is cancelled, then
// disconnects every client and returns nil. It may be called only once.
func (s *Server) Run(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyRunning
	}
	defer s.shutdown()

	ticker := time.NewTicker(sim.TickDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick is one server frame: admit new connections, apply what clients sent,
// advance the world, and send everyone their view of it.
func (s *Server) tick() {
	s.acceptPending()
	s.readClients()
	s.world.Step()
	s.broadcast()
}

func (s *Server) acceptPending() {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()
	s.peers = append(s.peers, pending...)
}

func (s *Server) readClients() {
	live := s.peers[:0]
	for _, p := range s.peers {
		if s.readPeer(p) {
			live = append(live, p)
		} else {
			s.drop(p)
		}
	}
	clear(s.peers[len(live):]) // let dropped peers be collected
	s.peers = live
}

// readPeer handles what p sent since the last tick and reports whether p
// stays connected.
func (s *Server) readPeer(p *peer) bool {
	select {
	case <-p.gone:
		return false
	default:
	}
	for range maxMsgsPerTick {
		select {
		case m := <-p.in:
			if !s.handle(p, m) {
				return false
			}
		default:
			return true
		}
	}
	return true
}

// handle applies one client message and reports whether the client is still
// following the protocol.
func (s *Server) handle(p *peer, m protocol.ClientMsg) bool {
	switch m.Kind {
	case protocol.ClientHello:
		if p.joined {
			return true // a repeated Hello is harmless
		}
		if m.Hello.Version != protocol.Version {
			return false
		}
		p.player = s.world.AddPlayer(m.Hello.Name)
		p.joined = true
		w, h := s.world.Size()
		// Nothing was sent to p before, so the buffered channel has room.
		return s.send(p, protocol.ServerMsg{
			Kind:    protocol.ServerWelcome,
			Welcome: protocol.Welcome{PlayerID: p.player, Width: w, Height: h, Tiles: s.world.Tiles()},
		})
	case protocol.ClientInput:
		if !p.joined {
			return false
		}
		s.world.SetInput(p.player, m.Input)
	case protocol.ClientCommand:
		if !p.joined {
			return false
		}
		s.world.Apply(p.player, m.Command)
	default:
		return false
	}
	return true
}

func (s *Server) broadcast() {
	players := s.world.Players()
	for _, p := range s.peers {
		if !p.joined {
			continue
		}
		if snap, ok := s.snapshotFor(p.player, players); ok {
			// A client that is not reading just misses this snapshot; the
			// next one supersedes it, and the loop never blocks on a client.
			s.send(p, protocol.ServerMsg{Kind: protocol.ServerSnapshot, Snapshot: snap})
		}
	}
}

// send queues m for p without blocking and reports whether it fit.
func (s *Server) send(p *peer, m protocol.ServerMsg) bool {
	select {
	case p.out <- m:
		return true
	default:
		return false
	}
}

// drop removes p's player and closes its outgoing channel. Each peer is
// dropped exactly once: by readClients, or by shutdown for the rest.
func (s *Server) drop(p *peer) {
	if p.joined {
		s.world.RemovePlayer(p.player)
	}
	close(p.out)
}

func (s *Server) shutdown() {
	s.mu.Lock()
	s.closed = true
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()

	for _, p := range pending {
		close(p.out)
	}
	for _, p := range s.peers {
		s.drop(p)
	}
	s.peers = nil
}
