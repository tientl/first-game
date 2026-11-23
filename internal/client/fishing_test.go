package client

import (
	"slices"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

func TestRodTipIsUpAndOut(t *testing.T) {
	for _, d := range []sim.Dir{sim.DirDown, sim.DirUp, sim.DirLeft, sim.DirRight} {
		x, y := rodTip(d, 100, 100)
		if y > 100-15 {
			t.Errorf("dir %d: tip at y %.1f, want well above the feet", d, y)
		}
		if side := facingSide(d); (x-100)*side < 5 {
			t.Errorf("dir %d: tip at x %.1f, want out on the hand's side", d, x)
		}
	}
}

func TestBitePlopsAndCatchSplashes(t *testing.T) {
	played := testSound(t)
	snd.setListener(10.5, 10.5)
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Held: sim.ItemRod, Cast: sim.CastWaiting, BobX: 10.5, BobY: 12.5}
	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me}})

	me.Cast = sim.CastBite
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me}})
	if !slices.Contains(*played, "plop") {
		t.Fatalf("played %v, want a plop on the bite", *played)
	}

	me.Cast, me.Acting = sim.CastNone, true // the strike
	s := protocol.Snapshot{Tick: 3, Players: []protocol.PlayerView{me}}
	s.Self.Notice, s.Self.NoticeArg, s.Self.NoticeSeq = sim.NoticeCaught, uint8(sim.ItemFish), 1
	s.Self.Inv.Add(sim.ItemFish, 1)
	*played = nil
	g.onSnapshot(s)
	if !slices.Contains(*played, "splash") {
		t.Fatalf("played %v, want a splash on the catch", *played)
	}
	if noticeText(sim.NoticeCaught, uint8(sim.ItemFish)) != "Câu được 1 Cá sống!" {
		t.Errorf("notice %q", noticeText(sim.NoticeCaught, uint8(sim.ItemFish)))
	}
}

func TestNoTargetWhileTheLineIsOut(t *testing.T) {
	g := readyGame()
	me := protocol.PlayerView{ID: 1, X: 10.5, Y: 10.5, Held: sim.ItemRod}
	bush := protocol.EntityView{ID: 5, Kind: sim.EntityBerryBush, TX: 11, TY: 10, Ready: true}
	hooked := protocol.CreatureView{ID: 9, Kind: sim.CreatureFish, X: 10.5, Y: 11.5, State: sim.CreatureHooked}
	free := protocol.CreatureView{ID: 8, Kind: sim.CreatureFish, X: 10.5, Y: 12.4}

	g.onSnapshot(protocol.Snapshot{Tick: 1, Players: []protocol.PlayerView{me},
		Creatures: []protocol.CreatureView{hooked, free}})
	if tg := g.target(); !tg.ok || tg.c.ID != free.ID {
		t.Fatalf("target %+v, want the free fish, never the hooked one", tg)
	}

	me.Cast, me.BobX, me.BobY = sim.CastWaiting, 10.5, 11.5
	g.onSnapshot(protocol.Snapshot{Tick: 2, Players: []protocol.PlayerView{me},
		Entities: []protocol.EntityView{bush}, Creatures: []protocol.CreatureView{hooked, free}})
	if tg := g.target(); tg.ok {
		t.Fatalf("target %+v while fishing, want none", tg)
	}
}
