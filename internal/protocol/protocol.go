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
const Version = 1

// ClientMsgKind selects which payload of a ClientMsg is meaningful.
type ClientMsgKind uint8

const (
	ClientHello ClientMsgKind = iota + 1
	ClientInput
	ClientCommand
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
}

// ServerMsgKind selects which payload of a ServerMsg is meaningful.
type ServerMsgKind uint8

const (
	ServerWelcome ServerMsgKind = iota + 1
	ServerSnapshot
)

// ServerMsg is a message from the server to one client.
type ServerMsg struct {
	Kind     ServerMsgKind
	Welcome  Welcome
	Snapshot Snapshot
}

// Welcome answers Hello with the client's player id and the static map.
type Welcome struct {
	PlayerID      sim.PlayerID
	Width, Height int
	Tiles         []sim.Tile // row-major, Width*Height
}

// Snapshot is the state one client needs to draw a frame, sent every tick.
// It only holds what is near the client's player.
type Snapshot struct {
	Tick     uint64
	Self     SelfState
	Players  []PlayerView // includes the client's own player
	Entities []EntityView
}

// SelfState is the client's own private state.
type SelfState struct {
	Health, Hunger float32
	Inv            sim.Inventory
	Dead, Cold     bool

	Notice    sim.NoticeKind
	NoticeArg uint8
	NoticeSeq uint32
}

// PlayerView is what everyone can see of a player.
type PlayerView struct {
	ID     sim.PlayerID
	Name   string
	X, Y   float32
	Dir    sim.Dir
	Dead   bool
	Acting bool
}

// EntityView is what clients see of a world entity.
type EntityView struct {
	ID     sim.EntityID
	Kind   sim.EntityKind
	TX, TY int16
	Ready  bool
	Fuel   int32 // campfires only
}
