package server

import (
	"testing"

	"survivalgame/internal/sim"
)

func TestTheChosenLookIsShownToEveryone(t *testing.T) {
	s := newTestServer(t)
	a := connect(t, s)
	l := sim.Look{Female: true, Hair: sim.HairBun, HairColor: 6, Skin: 4}
	m := hello("An")
	m.Hello.Look = l
	send(t, a, m)
	s.tick()
	id := next(t, a).Welcome.PlayerID
	b, _ := join(t, s, "Bình")
	s.tick()
	for _, c := range []*LocalConn{a, b} {
		if p, ok := findPlayer(latest(t, c), id); !ok || p.Look != l {
			t.Errorf("seen as %+v (found %v), want %+v", p.Look, ok, l)
		}
	}
}
