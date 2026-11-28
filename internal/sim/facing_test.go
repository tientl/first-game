package sim

import (
	"encoding/json"
	"testing"
)

// R turns a thing of four facings a quarter clockwise each press, the
// wheel back the other way; a run and a profile flip between their two;
// what does not turn stays facing south.
func TestNextFacingTurns(t *testing.T) {
	d := DirDown
	for i, want := range []Dir{DirLeft, DirUp, DirRight, DirDown} {
		if d = NextFacing(EntityBed, 0, d, 1); d != want {
			t.Fatalf("bed, press %d: %v, want %v", i+1, d, want)
		}
	}
	if d := NextFacing(EntityBed, 0, DirDown, -1); d != DirRight {
		t.Errorf("bed turned back from south: %v, want east", d)
	}
	if d := NextFacing(EntityWall, 0, DirDown, 1); d != DirRight {
		t.Errorf("wall turned: %v, want north–south (DirRight)", d)
	}
	if d := NextFacing(EntityWall, 0, DirRight, -1); d != DirDown {
		t.Errorf("wall turned back: %v, want east–west", d)
	}
	if d := NextFacing(EntityTrophy, 0, DirDown, 1); d != DirLeft {
		t.Errorf("head turned: %v, want looking left", d)
	}
	if d := NextFacing(EntityTorchStand, 0, DirDown, 1); d != DirDown || Turns(EntityTorchStand, 0) {
		t.Errorf("a standing torch turned to %v", d)
	}
	if !Turns(EntityCampfire, uint8(GearGrill)) || Turns(EntityCampfire, uint8(GearPot)) {
		t.Error("a grill should lie either way and a pot not turn")
	}
}

// Every piece of the build palette that turns is put down facing each way
// it can, and keeps its facing through a save.
func TestFreeBuildTurnsAndSavesFacings(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	type put struct {
		tx, ty int
		k      EntityKind
		face   Dir
	}
	var puts []put
	for i, pc := range DevPieces() {
		if !Turns(pc.Kind, pc.Var) {
			continue
		}
		for _, d := range clockwise {
			spots := freeTiles(w, p, 1)
			if len(spots) == 0 {
				t.Fatal("no free tile left")
			}
			tx, ty := spots[0][0], spots[0][1]
			w.Apply(p.ID, Command{Kind: CmdDevBuild, Slot: uint8(i), At: true, TX: int16(tx), TY: int16(ty), Face: d})
			e, ok := w.EntityAt(tx, ty)
			if !ok || e.Kind != pc.Kind {
				t.Fatalf("%s facing %v was not put down", pc.Name, d)
			}
			if want := FacingOf(pc.Kind, pc.Var, d); e.Face != want {
				t.Errorf("%s put down facing %v faces %v, want %v", pc.Name, d, e.Face, want)
			}
			puts = append(puts, put{tx, ty, pc.Kind, e.Face})
		}
	}
	if len(puts) < 4*8 {
		t.Fatalf("only %d turned pieces put down", len(puts))
	}
	raw, err := json.Marshal(w.Save())
	if err != nil {
		t.Fatal(err)
	}
	var d SaveData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, pt := range puts {
		if e, ok := w2.EntityAt(pt.tx, pt.ty); !ok || e.Kind != pt.k || e.Face != pt.face {
			t.Errorf("after loading, %v at %d,%d faces %v, want %v", pt.k, pt.tx, pt.ty, e.Face, pt.face)
		}
	}
}

// A structure crafted onto a tile and a tent set down from the hand face
// the way the command says.
func TestCraftedAndPlacedThingsFace(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	spots := freeTiles(w, p, 2)
	w.Apply(p.ID, Command{Kind: CmdCraft, Recipe: RecipeBed, At: true, TX: int16(spots[0][0]), TY: int16(spots[0][1]), Face: DirRight})
	if e, ok := w.EntityAt(spots[0][0], spots[0][1]); !ok || e.Kind != EntityBed || e.Face != DirRight {
		t.Errorf("crafted bed: %+v, want a bed facing east", e)
	}
	p.Inv = Inventory{}
	p.Inv.Add(ItemLeafTent, 1)
	p.Selected = 0
	w.Apply(p.ID, Command{Kind: CmdPlace, At: true, TX: int16(spots[1][0]), TY: int16(spots[1][1]), Face: DirUp})
	if e, ok := w.EntityAt(spots[1][0], spots[1][1]); !ok || e.Kind != EntityLeafTent || e.Face != DirUp {
		t.Errorf("tent set down: %+v, want a leaf tent facing north", e)
	}
}

// A save from before things turned has no facings: everything in it faces
// south, and a facing a kind cannot have is folded onto one it can.
func TestOlderSavesFaceSouth(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	spots := freeTiles(w, p, 3)
	w.addEntity(EntityDoor, spots[0][0], spots[0][1])
	w.addEntity(EntityFlower, spots[1][0], spots[1][1]).Face = DirLeft // no flower turns
	w.addEntity(EntityWall, spots[2][0], spots[2][1]).Face = DirLeft   // a run: north–south
	d := w.Save()
	d.Version = 30
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []Dir{DirDown, DirDown, DirRight} {
		if e, _ := w2.EntityAt(spots[i][0], spots[i][1]); e.Face != want {
			t.Errorf("%v faces %v after loading, want %v", e.Kind, e.Face, want)
		}
	}
}

// A door of an older save standing in a wall that runs north–south is
// turned on loading to be passed east–west, as it was; one in a wall
// running east–west, or standing alone, stays facing south.
func TestOlderSavesKeepDoorsPassable(t *testing.T) {
	w, p := newFreeBuildWorld(t)
	tx, ty := int(p.X)+3, int(p.Y)-4
	for dy := 0; dy < 5; dy++ {
		k := EntityStoneWall
		if dy == 2 {
			k = EntityDoor
		}
		if e := w.entityAt(tx, ty+dy); e != nil {
			w.removeEntity(e)
		}
		w.addEntity(k, tx, ty+dy)
	}
	for dx := -1; dx <= 1; dx++ {
		k := EntityWall
		if dx == 0 {
			k = EntityDoor
		}
		if e := w.entityAt(tx-4+dx, ty); e != nil {
			w.removeEntity(e)
		}
		w.addEntity(k, tx-4+dx, ty)
	}
	d := w.Save()
	d.Version = 30
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if e, _ := w2.EntityAt(tx, ty+2); e.Kind != EntityDoor || e.Face != DirRight {
		t.Errorf("door in a wall running north–south: %v facing %v, want a door facing east", e.Kind, e.Face)
	}
	if e, _ := w2.EntityAt(tx-4, ty); e.Kind != EntityDoor || e.Face != DirDown {
		t.Errorf("door in a wall running east–west: %v facing %v, want a door facing south", e.Kind, e.Face)
	}
}

// A door is walked through along its way through, and its posts stop
// whoever tries to cross it the other way.
func TestDoorsArePassedAlongTheirWay(t *testing.T) {
	walk := func(face Dir, mx, my int8, sx, sy float64) Player {
		w := flatWorld()
		w.addEntity(EntityDoor, 16, 14).face(face)
		id := w.AddPlayer("a")
		p := w.players[id]
		p.X, p.Y = sx, sy
		w.SetInput(id, Input{MoveX: mx, MoveY: my})
		stepN(w, 2*TicksPerSecond)
		return mustPlayer(t, w, id)
	}
	// Facing south, in a wall running east–west: through from the south.
	if p := walk(DirDown, 0, -1, 16.5, 16.5); p.Y > 13.9 {
		t.Errorf("walking north through a door facing south stopped at y = %.2f", p.Y)
	}
	// A little off its middle, it still lets one through.
	if p := walk(DirDown, 0, -1, 16.3, 16.5); p.Y > 13.9 {
		t.Errorf("walking north a little off the middle stopped at y = %.2f", p.Y)
	}
	// Facing east, in a wall running north–south: its posts on its north
	// and south edges stop one walking north into it, and let one walking
	// east through.
	if p := walk(DirRight, 0, -1, 16.5, 16.5); p.Y < 14.9 {
		t.Errorf("walking north into a door facing east got to y = %.2f", p.Y)
	}
	if p := walk(DirRight, 1, 0, 14.5, 14.5); p.X < 17.1 {
		t.Errorf("walking east through a door facing east stopped at x = %.2f", p.X)
	}
}

// A bed, a tent, a furnace and an oven are used from before or beside
// them, not from behind.
func TestUsedFromTheFront(t *testing.T) {
	for _, c := range []struct {
		face   Dir
		px, py float64
		ok     bool
	}{
		{DirDown, 10.5, 11.3, true}, {DirDown, 10.5, 9.6, false}, {DirDown, 11.4, 10.5, true},
		{DirRight, 11.3, 10.5, true}, {DirRight, 9.6, 10.5, false},
		{DirUp, 10.5, 9.6, true}, {DirLeft, 9.6, 10.5, true}, {DirLeft, 11.4, 10.5, false},
	} {
		if got := FrontSide(c.face, 10, 10, c.px, c.py); got != c.ok {
			t.Errorf("facing %v, standing at %.1f,%.1f: front side %v, want %v", c.face, c.px, c.py, got, c.ok)
		}
	}
}
