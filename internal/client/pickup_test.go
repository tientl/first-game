package client

import (
	"strings"
	"testing"

	"survivalgame/internal/protocol"
	"survivalgame/internal/server"
	"survivalgame/internal/sim"
)

// The left button's target is never a pile or a loose thing: E's corners,
// label and hint go to those, and the fight keeps the hint.
func TestTheLeftButtonNeverTargetsWhatEPicksUp(t *testing.T) {
	twig := protocol.EntityView{ID: 5, Kind: sim.EntityTwig, TX: 11, TY: 10, Ready: true}
	g, _ := fieldGame(twig) // I stand at (10.5, 10.5)
	meat := protocol.DropView{ID: 3, Item: sim.ItemMeat, Count: 2, X: 10.5, Y: 11.3, Age: 100}
	raptor := protocol.CreatureView{ID: 9, Kind: sim.CreatureRaptor, X: 9.4, Y: 10.5, State: sim.CreatureChase}
	g.curr.Drops = []protocol.DropView{meat}
	g.curr.Creatures = []protocol.CreatureView{raptor}
	view := func() (targetView, targetView) {
		tg, _ := g.target()
		g.hudTarget, g.pick.view = tg, g.pickTarget()
		return tg, g.pick.view
	}

	// No pointer: the beast for the left button and the hint, the pile for E.
	tg, pk := view()
	if !tg.ok || !tg.creature || tg.c.ID != raptor.ID {
		t.Fatalf("no pointer: target %+v, want the raptor", tg)
	}
	if !pk.ok || !pk.drop || pk.d.ID != meat.ID || pk.aimed {
		t.Fatalf("no pointer: E picks up %+v, want the nearest pile", pk)
	}
	if key, s, _ := g.contextHint(); key != "Chuột trái:" || !strings.HasPrefix(s, "đánh ") {
		t.Errorf("no pointer, a raptor at me: hint %q %q, want the fight", key, s)
	}

	// The pointer on the pile: E's, named; the left button still strikes.
	g.SetAim(0, 0.8)
	tg, pk = view()
	if !tg.ok || !tg.creature || tg.drop {
		t.Fatalf("on the pile: target %+v, want the raptor still", tg)
	}
	if !pk.ok || !pk.drop || !pk.aimed {
		t.Fatalf("on the pile: E picks up %+v", pk)
	}
	meatHint := "nhặt " + strings.ToLower(sim.ItemMeat.Def().Name) + " ×2"
	if key, s, _ := g.contextHint(); key != "E:" || s != meatHint {
		t.Errorf("on the pile: hint %q %q, want E: %q", key, s, meatHint)
	}
	if l := g.labelled(tg); !l.drop {
		t.Errorf("on the pile: the label names %+v", l)
	}

	// The pointer on the twig: never the left button's.
	g.SetAim(1, 0.1)
	tg, pk = view()
	if tg.ok && !tg.creature {
		t.Fatalf("on the twig: target %+v, want no thing on the ground", tg)
	}
	if !pk.ok || pk.drop || pk.e.ID != twig.ID || !pk.aimed {
		t.Fatalf("on the twig: E picks up %+v, want the twig", pk)
	}
	want := "nhặt " + strings.ToLower(sim.EntityTwig.Name())
	if key, s, _ := g.contextHint(); key != "E:" || s != want {
		t.Errorf("on the twig: hint %q %q, want E: %q", key, s, want)
	}

	// No beast: the nearest pile is E's, and the hint says so.
	g.curr.Creatures = nil
	g.ptr.fixed = false
	tg, pk = view()
	if tg.ok {
		t.Fatalf("no beast: target %+v, want nothing for the left button", tg)
	}
	if key, s, _ := g.contextHint(); key != "E:" || s != meatHint || !pk.drop {
		t.Errorf("no beast: hint %q %q, want E: %q", key, s, meatHint)
	}
}

// A chest pointed at is what E opens, so the nearest pile is not hinted or
// marked for E.
func TestEHintsTheChestPointedAtBeforeTheNearestPile(t *testing.T) {
	chest := protocol.EntityView{ID: 5, Kind: sim.EntityChest, TX: 9, TY: 10, Ready: true}
	g, _ := fieldGame(chest)
	g.curr.Drops = []protocol.DropView{{ID: 3, Item: sim.ItemWood, Count: 1, X: 11.2, Y: 10.5, Age: 100}}
	g.SetAim(-1, -0.1)
	tg, _ := g.target()
	g.hudTarget, g.pick.view = tg, g.pickTarget()
	if !tg.ok || !tg.aimed || tg.e.ID != chest.ID {
		t.Fatalf("target %+v, want the chest pointed at", tg)
	}
	if g.ePicks(tg, g.pick.view) {
		t.Error("E would open the chest pointed at, not pick up the pile")
	}
	if key, s, _ := g.contextHint(); key != "E:" || !strings.HasPrefix(s, "mở ") {
		t.Errorf("hint %q %q, want E opening the chest", key, s)
	}
}

// Against a real server: what the client marks for E is what E picks up,
// and the left button, held, picks nothing up.
func TestClientAndServerAgreeOnPickingUp(t *testing.T) {
	s, err := server.New(server.Config{Seed: 7, Width: 64, Height: 64, Dev: true, Style: sim.StyleIsland})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := s.ConnectLocal()
	if err != nil {
		t.Fatal(err)
	}
	g := readyGame()
	g.conn = conn
	send := func(m protocol.ClientMsg) {
		if err := conn.Send(m); err != nil {
			t.Fatal(err)
		}
	}
	tick := func(n int) {
		for range n {
			s.Tick()
			for drained := false; !drained; {
				select {
				case m := <-conn.Messages():
					switch m.Kind {
					case protocol.ServerWelcome:
						g.myID, g.width, g.height, g.tiles = m.Welcome.PlayerID, m.Welcome.Width, m.Welcome.Height, m.Welcome.Tiles
					case protocol.ServerSnapshot:
						g.onSnapshot(m.Snapshot)
					}
				default:
					drained = true
				}
			}
		}
	}
	input := func(action bool) {
		in := sim.Input{Action: action}
		if ax, ay, ok := g.aim(); ok {
			in.Aim, in.AimX, in.AimY = true, quantAim(ax), quantAim(ay)
		}
		send(protocol.ClientMsg{Kind: protocol.ClientInput, Input: in})
	}
	command := func(k sim.CommandKind) {
		send(protocol.ClientMsg{Kind: protocol.ClientCommand, Command: sim.Command{Kind: k}})
	}
	// taken reports whether what pk names is gone from the ground.
	taken := func(pk targetView) bool {
		if pk.drop {
			for _, d := range g.curr.Drops {
				if d.ID == pk.d.ID {
					return d.Count < pk.d.Count
				}
			}
			return true
		}
		for _, e := range g.curr.Entities {
			if e.ID == pk.e.ID {
				return !e.Ready
			}
		}
		return false
	}
	send(protocol.ClientMsg{Kind: protocol.ClientHello, Hello: protocol.Hello{Version: protocol.Version, Name: "a"}})
	tick(1)
	send(protocol.ClientMsg{Kind: protocol.ClientDev, Dev: "give stone 3"})
	tick(1)
	slot := -1
	for i, sl := range g.curr.Self.Inv {
		if sl.Item == sim.ItemStone {
			slot = i
		}
	}
	if slot < 0 {
		t.Fatal("no stone given")
	}
	send(protocol.ClientMsg{Kind: protocol.ClientCommand, Command: sim.Command{Kind: sim.CmdToss, Slot: uint8(slot), Count: 3}})
	tick(sim.TicksPerSecond)
	if len(g.curr.Drops) == 0 {
		t.Fatal("the tossed stone is not on the ground")
	}
	pile := g.curr.Drops[0]

	// The left button held a second, pointing at the pile: nothing is picked up.
	me, _ := g.me()
	g.SetAim(float64(pile.X-me.X), float64(pile.Y-me.Y))
	before := g.curr
	if tg, _ := g.target(); tg.drop || (tg.ok && !tg.creature && sim.LooseItem(tg.e.Kind)) {
		t.Fatalf("the left button's target is %+v", tg)
	}
	input(true)
	tick(sim.TicksPerSecond)
	input(false)
	tick(1)
	for _, d := range before.Drops {
		found := false
		for _, e := range g.curr.Drops {
			found = found || (e.ID == d.ID && e.Count == d.Count)
		}
		if !found {
			t.Errorf("pile %d of %v went while the left button was held", d.ID, d.Item)
		}
	}
	for _, e := range before.Entities {
		if !sim.LooseItem(e.Kind) || !e.Ready {
			continue
		}
		for _, c := range g.curr.Entities {
			if c.ID == e.ID && !c.Ready {
				t.Errorf("a %v was taken by the left button", e.Kind)
			}
		}
	}

	// E on the pile takes it; then E held takes what the client marks next,
	// one thing after another, till nothing is left in reach.
	pk := g.pickTarget()
	if !pk.ok || !pk.drop || pk.d.ID != pile.ID || !pk.aimed {
		t.Fatalf("E would pick up %+v, want the pile pointed at", pk)
	}
	input(false)
	command(sim.CmdInteract)
	tick(1)
	if !taken(pk) || g.curr.Self.Inv.Count(sim.ItemStone) != 3 {
		t.Fatalf("E on the pile: stone %d", g.curr.Self.Inv.Count(sim.ItemStone))
	}
	g.ptr.fixed = false // no pointer: the nearest
	input(false)
	tick(1)
	for i := 0; i < 20; i++ {
		pk := g.pickTarget()
		if !pk.ok {
			break
		}
		command(sim.CmdPickUp)
		tick(sim.PickCooldown)
		if !taken(pk) {
			t.Fatalf("E held: the client marked %+v, the server did not take it", pk)
		}
	}
}
