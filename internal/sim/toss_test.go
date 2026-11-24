package sim

import (
	"math"
	"testing"
)

// tossWorld is a grass field with a player in the middle facing right,
// holding 5 stone in hotbar slot 0.
func tossWorld() (*World, *Player) {
	w := flatWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Dir = DirRight
	p.Inv[0] = Slot{Item: ItemStone, Count: 5}
	p.Selected = 0
	return w, p
}

func pileCount(w *World, k ItemKind) (n, piles int) {
	for _, d := range w.drops {
		if !d.gone && d.Item == k {
			n += d.Count
			piles++
		}
	}
	return n, piles
}

func TestTossOneLandsInFront(t *testing.T) {
	w, p := tossWorld()
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 1})
	if p.Inv[0].Count != 4 {
		t.Fatalf("slot holds %d after throwing one, want 4", p.Inv[0].Count)
	}
	if n, piles := pileCount(w, ItemStone); n != 1 || piles != 1 {
		t.Fatalf("%d stone in %d piles on the ground, want 1 in 1", n, piles)
	}
	d := w.drops[0]
	if d.X <= p.X || math.Abs(d.X-p.X-TossReach) > 1e-9 || d.Y != p.Y {
		t.Fatalf("pile at (%.2f,%.2f), player at (%.2f,%.2f) facing right", d.X, d.Y, p.X, p.Y)
	}
	if p.Notice != NoticeTossed || p.NoticeArg != uint8(ItemStone) {
		t.Fatalf("notice %d arg %d, want tossed stone", p.Notice, p.NoticeArg)
	}
	// Throwing again tops up the same pile.
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 1})
	if n, piles := pileCount(w, ItemStone); n != 2 || piles != 1 {
		t.Fatalf("%d stone in %d piles, want 2 in one pile", n, piles)
	}
}

func TestTossWholeStackIsClamped(t *testing.T) {
	w, p := tossWorld()
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 255})
	if !p.Inv[0].Empty() {
		t.Fatalf("slot holds %v after throwing the whole stack", p.Inv[0])
	}
	if n, _ := pileCount(w, ItemStone); n != 5 {
		t.Fatalf("%d stone on the ground, want 5", n)
	}
}

func TestTossInvalidDoesNothing(t *testing.T) {
	w, p := tossWorld()
	before := p.Inv
	for _, c := range []Command{
		{Kind: CmdToss, Slot: NumSlots, Count: 1},     // no such slot
		{Kind: CmdToss, Slot: 255, Count: 1},          // no such slot
		{Kind: CmdToss, Slot: 3, Count: 1},            // empty
		{Kind: CmdToss, Slot: 0, Count: 0},            // nothing
		{Kind: CmdToss, Slot: HotbarSlots, Count: 40}, // an empty bag slot
	} {
		w.Apply(p.ID, c)
	}
	if p.Inv != before || len(w.drops) != 0 {
		t.Fatalf("invalid throws changed the bag or left %d piles", len(w.drops))
	}
}

func TestTossRefusedWhileDeadAsleepOrAtAChest(t *testing.T) {
	for _, set := range []func(p *Player){
		func(p *Player) { p.Dead = true },
		func(p *Player) { p.Asleep = true },
		func(p *Player) { p.chest = 99 },
	} {
		w, p := tossWorld()
		set(p)
		w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 1})
		if p.Inv[0].Count != 5 || len(w.drops) != 0 {
			t.Fatalf("threw while dead, asleep or at a chest: slot %d, %d piles", p.Inv[0].Count, len(w.drops))
		}
	}
}

func TestTossAtFeetWhenTheFrontIsBlocked(t *testing.T) {
	w, p := tossWorld()
	w.addEntity(EntityWall, int(p.X)+1, int(p.Y))
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 2})
	if len(w.drops) != 1 || w.drops[0].X != p.X || w.drops[0].Y != p.Y {
		t.Fatalf("piles %+v: want one at the feet", w.drops)
	}
	// Standing on it, the thrower does not walk it straight back up...
	stepN(w, 2*TicksPerSecond)
	if n, _ := pileCount(w, ItemStone); n != 2 {
		t.Fatalf("%d stone left on the ground: the thrower walked it back up at once", n)
	}
	// ...but after stepping off and back on, they do.
	p.X += 1.5
	w.Step()
	p.X -= 1.5
	w.Step()
	if p.Inv.Count(ItemStone) != 5 || len(w.drops) != 0 {
		t.Fatalf("stone %d, %d piles: want it walked up after stepping off", p.Inv.Count(ItemStone), len(w.drops))
	}
}

func TestTossedPileIsPickedUpBySpaceAndByOthers(t *testing.T) {
	w, p := tossWorld()
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 3})
	w.SetInput(p.ID, Input{Action: true})
	w.Step()
	w.SetInput(p.ID, Input{})
	if p.Inv.Count(ItemStone) != 5 || len(w.drops) != 0 {
		t.Fatalf("stone %d, %d piles after Space: want it back", p.Inv.Count(ItemStone), len(w.drops))
	}

	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 3})
	d := w.drops[0]
	q := w.players[w.AddPlayer("b")]
	q.X, q.Y = d.X+3, d.Y
	w.SetInput(q.ID, Input{MoveX: -1})
	stepN(w, 2*TicksPerSecond)
	if q.Inv.Count(ItemStone) != 3 {
		t.Fatalf("the other player has %d stone, want the 3 thrown", q.Inv.Count(ItemStone))
	}
}

func TestTossedPileIsSaved(t *testing.T) {
	w, p := tossWorld()
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 2})
	d := w.Save()
	if len(d.Drops) != 1 || d.Drops[0].Item != ItemStone || d.Drops[0].Count != 2 {
		t.Fatalf("saved drops %+v", d.Drops)
	}
}

func TestTossRefusedOverOpenWater(t *testing.T) {
	w := seaWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	p.Inv[0] = Slot{Item: ItemStone, Count: 5}
	// Swimming out at sea, facing more sea: refused, nothing lost.
	p.X, p.Dir = 10.5, DirRight
	if !w.Swimming(p) {
		t.Fatal("not swimming")
	}
	w.Apply(id, Command{Kind: CmdToss, Slot: 0, Count: 5})
	if p.Inv[0].Count != 5 || len(w.drops) != 0 || p.Notice != NoticeCantToss {
		t.Fatalf("slot %d, %d piles, notice %d: want the throw refused", p.Inv[0].Count, len(w.drops), p.Notice)
	}
	// In the water next to the beach, facing it: it lands on the sand.
	p.X, p.Dir = 4.3, DirLeft
	w.Apply(id, Command{Kind: CmdToss, Slot: 0, Count: 1})
	if len(w.drops) != 1 || !w.walkableAt(w.drops[0].X, w.drops[0].Y) || p.Inv[0].Count != 4 {
		t.Fatalf("piles %+v: want one on the beach", w.drops)
	}
}

func TestTossFromABoatOnlyOntoLand(t *testing.T) {
	w := seaWorld()
	id := w.AddPlayer("a")
	p := w.players[id]
	w.boats = append(w.boats, &Boat{ID: 1, Kind: BoatRaft, X: 10.5, Y: 16.5, HP: BoatRaft.MaxHP()})
	w.nextBoat = 1
	p.X, p.Y = 9.5, 16.5
	if !w.board(p) || p.boat == 0 {
		t.Fatal("did not board")
	}
	p.Inv[0] = Slot{Item: ItemWood, Count: 3}
	w.Apply(id, Command{Kind: CmdToss, Slot: 0, Count: 3})
	if p.Inv[0].Count != 3 || len(w.drops) != 0 || p.Notice != NoticeCantToss {
		t.Fatalf("slot %d, %d piles: a throw from a boat at sea was not refused", p.Inv[0].Count, len(w.drops))
	}
}

func TestTossTheBurningTorchPutsItOut(t *testing.T) {
	w, p := tossWorld()
	p.Inv[0] = Slot{Item: ItemTorch, Count: 1}
	p.torchFuel = 100
	w.Apply(p.ID, Command{Kind: CmdToss, Slot: 0, Count: 1})
	if p.torchFuel != 0 {
		t.Fatalf("torch fuel %d after throwing the only torch", p.torchFuel)
	}
}
