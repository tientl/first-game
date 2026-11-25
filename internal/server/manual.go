package server

// Tick runs one server frame by hand, for a tool that drives the server in
// step with what it draws (cmd/shot's -encounter) instead of letting Run
// keep time. Do not call it on a server that is running.
func (s *Server) Tick() {
	if !s.running.Load() {
		s.tick()
	}
}
