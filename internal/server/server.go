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
	"log"
	"sync"
	"sync/atomic"
	"time"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Defaults for Config fields left at zero.
const (
	DefaultWorldSize      = 128
	DefaultInterestRadius = 44 // tiles; covers a 1920px-wide screen zoomed all the way out
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
	Width, Height  int            // in tiles; 0 means DefaultWorldSize
	Style          sim.WorldStyle // which generator a new world uses
	Difficulty     sim.Difficulty // how hard a new world is; a loaded world keeps its own
	InterestRadius int            // clients receive what is within this many tiles; 0 means default

	// SavePath, if set, is where the world is saved: on request, every
	// AutosaveEvery, and when Run returns.
	SavePath      string
	AutosaveEvery time.Duration // 0 means no autosave

	// Dev runs the dev console's commands (ClientDev) and puts every player
	// who joins in god mode with the whole map seen and free build on; see
	// dev.go. Without it the server refuses dev commands, free build's among
	// them.
	Dev bool
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
	paused  atomic.Bool
	peers   []*peer // owned by the Run goroutine

	nextAutosave uint64 // tick of the next autosave; 0 means never
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
	cfg = withDefaults(cfg)
	w, err := sim.Generate(sim.GenConfig{Seed: cfg.Seed, Width: cfg.Width, Height: cfg.Height, Style: cfg.Style, Difficulty: cfg.Difficulty})
	if err != nil {
		return nil, err
	}
	return newServer(cfg, w), nil
}

// NewFromSave rebuilds the world from a save; cfg's seed and size are
// replaced by the saved ones.
func NewFromSave(cfg Config, d sim.SaveData) (*Server, error) {
	w, err := sim.Load(d)
	if err != nil {
		return nil, err
	}
	cfg.Seed, cfg.Style = d.Seed, d.Style
	cfg.Width, cfg.Height = w.Size()
	return newServer(withDefaults(cfg), w), nil
}

func newServer(cfg Config, w *sim.World) *Server {
	w.SetDevMode(cfg.Dev) // free build, and what a dev save keeps of it: see sim/devbuild.go
	s := &Server{cfg: cfg, world: w}
	s.scheduleAutosave()
	return s
}

func (s *Server) scheduleAutosave() {
	if s.cfg.SavePath == "" || s.cfg.AutosaveEvery <= 0 {
		return
	}
	s.nextAutosave = s.world.Tick() + uint64(max(1, s.cfg.AutosaveEvery/sim.TickDuration))
}

// Save writes the world to cfg.SavePath now. It must only be called from the
// Run goroutine or before Run starts.
func (s *Server) save() error {
	err := writeSave(s.cfg.SavePath, SaveFile{SavedAt: time.Now(), World: s.world.Save()})
	if err != nil {
		log.Printf("save failed: %v", err)
	}
	s.scheduleAutosave()
	return err
}

// SetPaused freezes or resumes the world clock. Clients stay connected and
// keep receiving snapshots. Safe to call from any goroutine.
func (s *Server) SetPaused(p bool) { s.paused.Store(p) }

func withDefaults(cfg Config) Config {
	if cfg.Width == 0 {
		cfg.Width = DefaultWorldSize
	}
	if cfg.Height == 0 {
		cfg.Height = DefaultWorldSize
	}
	if cfg.InterestRadius <= 0 {
		cfg.InterestRadius = DefaultInterestRadius
	}
	return cfg
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
	if s.paused.Load() {
		s.broadcast()
		return
	}
	// While everyone sleeps, the night goes by faster.
	steps := 1
	if s.world.AllAsleep() {
		steps = sim.SleepSpeedup
	}
	for range steps {
		s.world.Step()
	}
	s.broadcast()
	if s.nextAutosave != 0 && s.world.Tick() >= s.nextAutosave {
		s.notifySaved(s.save(), true, nil)
	}
}

// notifySaved tells clients the outcome of a save: everyone for an
// autosave, only the asker otherwise.
func (s *Server) notifySaved(err error, auto bool, to *peer) {
	res := protocol.SaveResult{OK: err == nil, Auto: auto}
	if err != nil {
		res.Error = err.Error()
	}
	for _, p := range s.peers {
		if p.joined && (to == nil || p == to) {
			s.send(p, protocol.ServerMsg{Kind: protocol.ServerSaved, Saved: res})
		}
	}
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
		p.player = s.world.AddPlayerWithLook(m.Hello.Name, m.Hello.Look)
		p.joined = true
		s.devJoin(p) // see dev.go
		w, h := s.world.Size()
		// Nothing was sent to p before, so the buffered channel has room.
		return s.send(p, protocol.ServerMsg{
			Kind:    protocol.ServerWelcome,
			Welcome: protocol.Welcome{PlayerID: p.player, Width: w, Height: h, Tiles: s.world.Tiles(), Relief: s.world.Relief(), Explored: s.world.Explored(p.player), SurfaceH: surfaceH(s.world), Difficulty: s.world.Difficulty()},
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
		if s.devCommandRefused(m.Command) {
			return true // free build outside dev mode: dropped, the connection kept; see dev.go
		}
		s.world.Apply(p.player, m.Command)
	case protocol.ClientDev:
		if !p.joined {
			return false
		}
		s.devCommand(p, m.Dev) // see dev.go
	case protocol.ClientSave:
		if !p.joined {
			return false
		}
		if s.cfg.SavePath != "" {
			s.notifySaved(s.save(), false, p)
		}
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
	if s.cfg.SavePath != "" {
		s.save() // players still connected are saved where they stand
	}
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

// surfaceH is the Welcome's SurfaceH for w: 0 when the whole map is above ground.
func surfaceH(w *sim.World) int {
	if _, h := w.Size(); w.SurfaceHeight() < h {
		return w.SurfaceHeight()
	}
	return 0
}
