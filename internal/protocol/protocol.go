// Package protocol defines the messages exchanged between client and server.
//
// Every message is a plain struct of exported, fixed-shape fields so that any
// transport can carry it: the in-process channels used for single player
// today, and an encoder such as encoding/gob over TCP for co-op later.
//
// Each envelope has an explicit Kind and value (not pointer) payloads.
// Encoders like gob omit zero values, so a pointer payload whose fields are
// all zero (for example "stop moving") could arrive as nil and be lost; a
// Kind byte plus a value payload always decodes to what was sent.
//
// Ownership: once a message is passed to Send, the sender must not touch it
// or any slice it references again.
package protocol

import "survivalgame/internal/sim"

// Version is bumped on every incompatible change to these types.
const Version = 21

// ClientMsgKind selects which payload of a ClientMsg is meaningful.
type ClientMsgKind uint8

const (
	ClientHello ClientMsgKind = iota + 1
	ClientInput
	ClientCommand
	ClientSave // ask the server to save the world now
)

// ClientMsg is a message from a client to the server.
type ClientMsg struct {
	Kind    ClientMsgKind
	Hello   Hello
	Input   sim.Input
	Command sim.Command
}

// Hello must be the first message on a connection.
type Hello struct {
	Version int
	Name    string
	Look    sim.Look // the character made for a new game; a returning player keeps the saved one
}

// ServerMsgKind selects which payload of a ServerMsg is meaningful.
type ServerMsgKind uint8

const (
	ServerWelcome ServerMsgKind = iota + 1
	ServerSnapshot
	ServerSaved // the outcome of a save
)

// ServerMsg is a message from the server to one client.
type ServerMsg struct {
	Kind     ServerMsgKind
	Welcome  Welcome
	Snapshot Snapshot
	Saved    SaveResult
}

// SaveResult reports a save, requested by this client or automatic.
type SaveResult struct {
	OK    bool
	Auto  bool
	Error string
}

// Welcome answers Hello with the client's player id and the static map.
type Welcome struct {
	PlayerID      sim.PlayerID
	Width, Height int
	Tiles         []sim.Tile // row-major, Width*Height
	Explored      []byte     // tiles this player has seen (sim.Explored bytes)
}

// Snapshot is the state one client needs to draw a frame, sent every tick.
// It only holds what is near the client's player.
type Snapshot struct {
	Tick      uint64
	Self      SelfState
	Players   []PlayerView // includes the client's own player
	Entities  []EntityView
	Creatures []CreatureView
}

// SelfState is the client's own private state.
type SelfState struct {
	Health, Hunger float32
	Inv            sim.Inventory // hotbar slots first, then the bag
	Selected       uint8         // the hotbar slot held
	Dead, Cold     bool
	TorchFuel      int32         // ticks left on the torch burning in the hand; 0 if none
	Chest          sim.EntityID  // the chest open, 0 when none
	ChestSlots     sim.Inventory // what that chest holds

	Notice    sim.NoticeKind
	NoticeArg uint8
	NoticeSeq uint32
}

// PlayerView is what everyone can see of a player.
type PlayerView struct {
	ID      sim.PlayerID
	Name    string
	X, Y    float32
	Dir     sim.Dir
	Dead    bool
	Acting  bool
	HasAxe  bool         // carries an axe
	AxeTier sim.Tier     // what the best axe carried is made of
	Armor   sim.Tier     // what the best armour carried is made of; 0 (stone) means none
	Held    sim.ItemKind // what is in the selected hotbar slot, drawn in the hand
	Asleep  bool         // in bed: drawn lying down
	Look    sim.Look     // body and hair

	Cast       sim.CastState // fishing: whether the line is out, and whether the fish bites
	BobX, BobY float32       // where the float sits while the line is out
}

// EntityView is what clients see of a world entity.
type EntityView struct {
	ID     sim.EntityID
	Kind   sim.EntityKind
	TX, TY int16
	Ready  bool
	Big    bool  // trees and boulders: the big size
	Stage  uint8 // growing crops: 0 just planted to sim.CropStages-1
	Fuel   int32 // campfires only
}

// CreatureView is what clients see of an animal.
type CreatureView struct {
	ID       sim.CreatureID
	Kind     sim.CreatureKind
	X, Y     float32
	State    sim.CreatureState
	FaceLeft bool
}
