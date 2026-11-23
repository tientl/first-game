package protocol

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"

	"survivalgame/internal/sim"
)

// The messages must survive a gob stream unchanged: that is how a TCP
// transport will carry them. The "stop moving" input is the case that breaks
// with pointer payloads, because all its fields are zero.
func TestMessagesRoundTripThroughGob(t *testing.T) {
	client := []ClientMsg{
		{Kind: ClientHello, Hello: Hello{Version: Version, Name: "Tiến"}},
		{Kind: ClientInput, Input: sim.Input{MoveX: -1, MoveY: 1, Action: true}},
		{Kind: ClientInput, Input: sim.Input{}}, // stop moving
		{Kind: ClientCommand, Command: sim.Command{Kind: sim.CmdCraft, Recipe: sim.RecipeCampfire}},
	}
	server := []ServerMsg{
		{Kind: ServerWelcome, Welcome: Welcome{
			PlayerID: 3, Width: 2, Height: 2,
			Tiles: []sim.Tile{sim.TileWater, sim.TileSand, sim.TileGrass, sim.TileRocky},
		}},
		{Kind: ServerSnapshot, Snapshot: Snapshot{
			Tick: 12345,
			Self: SelfState{
				Health: 87.5, Hunger: 40,
				Inv:      sim.InventoryWith(sim.Ingredient{Item: sim.ItemWood, Count: 3}, sim.Ingredient{Item: sim.ItemBerry, Count: 1}),
				Selected: 7,
				Cold:     true, Notice: sim.NoticeBagFull, NoticeArg: uint8(sim.ItemBerry), NoticeSeq: 9,
			},
			Players: []PlayerView{
				{ID: 3, Name: "Tiến", X: 10.25, Y: -4, Dir: sim.DirLeft, Acting: true, Held: sim.ItemAxe},
				{ID: 4, Name: "B", Held: sim.ItemRod, Cast: sim.CastBite, BobX: 12.5, BobY: 3.75},
			},
			Entities: []EntityView{{ID: 77, Kind: sim.EntityCampfire, TX: 5, TY: 6, Ready: true, Fuel: 1200}},
			Creatures: []CreatureView{
				{ID: 4, Kind: sim.CreatureSnake, X: 7.5, Y: 8.25, State: sim.CreatureChase, FaceLeft: true},
				{ID: 5, Kind: sim.CreatureFish, X: 1, Y: 2}, // all zero but the position
			},
		}},
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, m := range client {
		if err := enc.Encode(m); err != nil {
			t.Fatalf("encode %+v: %v", m, err)
		}
	}
	for _, m := range server {
		if err := enc.Encode(m); err != nil {
			t.Fatalf("encode %+v: %v", m, err)
		}
	}

	dec := gob.NewDecoder(&buf)
	for _, want := range client {
		// Decode into a fresh value every time: gob skips zero fields, so
		// reusing a variable would keep stale values from the last message.
		var got ClientMsg
		if err := dec.Decode(&got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("client message changed in transit:\n got %+v\nwant %+v", got, want)
		}
	}
	for _, want := range server {
		var got ServerMsg
		if err := dec.Decode(&got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("server message changed in transit:\n got %+v\nwant %+v", got, want)
		}
	}
}
