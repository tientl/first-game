package sim

import (
	"errors"
	"reflect"
	"testing"
)

func TestSaveAndLoadRestoresTheWorld(t *testing.T) {
	w, err := Generate(GenConfig{Seed: 7, Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	id := w.AddPlayer("Tiến")
	p := w.players[id]
	p.Inv.Add(ItemWood, 5)
	p.Inv.Move(0, HotbarSlots+3) // a layout that is not what picking up gives
	p.Inv.Add(ItemBerry, 3)
	p.Selected = 4
	p.Health, p.Hunger = 42, 33
	// Harvest one tree so its state is not the generated one.
	var tree *Entity
	for _, e := range w.entities {
		if e.Kind == EntityTree {
			tree = e
			break
		}
	}
	tree.Ready, tree.RegrowAt = false, w.tick+500
	stepN(w, 30)

	data := w.Save()
	w2, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if w2.Tick() != w.Tick() {
		t.Errorf("tick = %d, want %d", w2.Tick(), w.Tick())
	}
	if !reflect.DeepEqual(w2.Tiles(), w.Tiles()) {
		t.Error("terrain differs after load")
	}
	if len(w2.entities) != len(w.entities) {
		t.Fatalf("entities = %d, want %d", len(w2.entities), len(w.entities))
	}
	if got := *w2.entities[tree.ID]; got != *tree {
		t.Errorf("tree = %+v, want %+v", got, *tree)
	}

	// The player rejoins by name and gets their state back.
	id2 := w2.AddPlayer("Tiến")
	p2 := mustPlayer(t, w2, id2)
	if p2.Inv != p.Inv || p2.Selected != 4 || p2.Health != p.Health || p2.Hunger != p.Hunger || p2.X != p.X || p2.Y != p.Y {
		t.Errorf("restored player = %+v, want %+v", p2, *p)
	}
	// Someone new starts fresh.
	if fresh := mustPlayer(t, w2, w2.AddPlayer("Khác")); fresh.Inv != (Inventory{}) {
		t.Errorf("new player inventory = %v, want empty", fresh.Inv)
	}
}

func TestLeftPlayersAreKeptInTheSave(t *testing.T) {
	w := flatWorld()
	id := w.AddPlayer("An")
	w.players[id].Inv.Add(ItemStone, 9)
	w.RemovePlayer(id)

	if got := w.Save().Players; len(got) != 1 || got[0].Slots.Count(ItemStone) != 9 {
		t.Fatalf("saved players = %+v, want An with 9 stone", got)
	}
	back := mustPlayer(t, w, w.AddPlayer("An"))
	if back.Inv.Count(ItemStone) != 9 {
		t.Errorf("rejoined inventory = %v, want 9 stone", back.Inv)
	}
}

func TestLoadRejectsAndSanitises(t *testing.T) {
	if _, err := Load(SaveData{Version: SaveVersion + 1}); !errors.Is(err, ErrBadSave) {
		t.Errorf("wrong version: err = %v, want ErrBadSave", err)
	}
	if _, err := Load(SaveData{Version: SaveVersion, Width: 5, Height: 5}); !errors.Is(err, ErrBadSave) {
		t.Errorf("bad size: err = %v, want ErrBadSave", err)
	}

	w, _ := Generate(GenConfig{Seed: 3, Width: 48, Height: 48})
	d := w.Save()
	d.Entities = append(d.Entities,
		Entity{ID: 99999, Kind: EntityBoulder, TX: -4, TY: 2},             // off the map
		Entity{ID: 99998, Kind: NumEntityKinds, TX: 1, TY: 1},             // unknown kind
		Entity{ID: d.Entities[0].ID, Kind: EntityTree, TX: 30, TY: 30},    // duplicate id
		Entity{ID: 99997, Kind: EntityCampfire, TX: 0, TY: 0, Fuel: 1000}, // on water
	)
	d.Players = []PlayerSave{{Name: "X", X: -100, Y: -100, Health: 1e9, Selected: 99,
		Slots: Inventory{{ItemWood, 5000}, {NumItems + 1, 3}, {ItemStone, -2}}}}
	w2, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(w2.entities) != len(w.entities) {
		t.Errorf("entities = %d, want the %d valid ones", len(w2.entities), len(w.entities))
	}
	p := mustPlayer(t, w2, w2.AddPlayer("X"))
	sx, sy := w2.Spawn()
	if p.X != sx || p.Y != sy || p.Health != MaxHealth || p.Selected != 0 {
		t.Errorf("player = %+v, want clamped and at the spawn", p)
	}
	if p.Inv[0] != (Slot{ItemWood, MaxStack}) || p.Inv[1] != (Slot{}) || p.Inv[2] != (Slot{}) {
		t.Errorf("inventory = %v, want the wood capped and the bad slots emptied", p.Inv[:3])
	}
}
