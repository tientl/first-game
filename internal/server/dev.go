package server

import (
	"log"

	"survivalgame/internal/protocol"
)

// devMaxLine bounds a dev console line, in bytes.
const devMaxLine = 200

// devJoin readies a newly joined player in dev mode: god mode on and the
// whole map seen, before the Welcome carries it.
func (s *Server) devJoin(p *peer) {
	if !s.cfg.Dev {
		return
	}
	s.world.DevSetGod(p.player, true)
	s.world.DevReveal(p.player)
}

// devCommand runs a dev console line from p, or refuses it when the server
// is not in dev mode, and answers p either way.
func (s *Server) devCommand(p *peer, line string) {
	if !s.cfg.Dev {
		s.send(p, protocol.ServerMsg{Kind: protocol.ServerDev, Dev: protocol.DevReply{
			Rejected: true,
			Lines:    []string{"Máy chủ không ở chế độ dev: lệnh bị từ chối"},
		}})
		return
	}
	if len(line) > devMaxLine {
		line = line[:devMaxLine]
	}
	log.Printf("dev: %s", line)
	r := s.world.DevCommand(p.player, line)
	reply := protocol.DevReply{Lines: r.Lines}
	if r.Revealed {
		reply.Explored = s.world.Explored(p.player)
	}
	s.send(p, protocol.ServerMsg{Kind: protocol.ServerDev, Dev: reply})
}

// DevMode reports whether the server runs dev commands.
func (s *Server) DevMode() bool { return s.cfg.Dev }
