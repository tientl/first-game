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
const Version = 25

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

	Season    sim.Season
	Weather   sim.Weather
	Drops     []DropView    // piles on the ground nearby
	Webs      []WebView     // spider webs in the air
	Strikes   []StrikeView  // lightning in the last second
	Tornadoes []TornadoView // whirlwinds nearby
	Boats     []BoatView    // boats nearby
	WindDir   float32       // where the wind blows towards, radians (0 east)
	Wind      float32       // how hard, 0 to 1
}

// BoatView is a boat on the water.
type BoatView struct {
	ID      sim.BoatID
	Kind    sim.BoatKind
	X, Y    float32
	Heading float32 // radians, 0 east
	Speed   float32 // tiles per tick
	HP      int16
	Rider   sim.PlayerID // 0 if nobody is aboard
	Held    bool         // in a giant squid's arms
}

// DropView is a pile on the ground.
type DropView struct {
	ID           sim.DropID
	Item         sim.ItemKind
	Count        uint8
	X, Y         float32
	FromX, FromY float32 // where it fell from
	Age          uint16  // ticks since it fell, capped
}

// WebView is a web a spider spat, in flight.
type WebView struct {
	ID     uint32
	X, Y   float32
	VX, VY float32
}

// StrikeView is a bolt of lightning.
type StrikeView struct {
	X, Y float32
	Age  uint8 // ticks since it struck
}

// TornadoView is a whirlwind.
type TornadoView struct {
	ID   uint32
	X, Y float32
}

// SelfState is the client's own private state.
type SelfState struct {
	Health, Hunger float32
	Inv            sim.Inventory // hotbar slots first, then the bag
	Selected       uint8         // the hotbar slot held
	Dead, Cold     bool
	Poisoned       bool          // spider venom eats health
	Webbed         bool          // tangled in web: slowed
	HomeBed        bool          // a bed is where this player comes back
	Boat           sim.BoatID    // the boat this player is aboard, 0 if none
	Breath         float32       // a swimmer's strength left, 1 to 0
	TorchFuel      int32         // ticks left on the torch burning in the hand; 0 if none
	Chest          sim.EntityID  // the chest open, 0 when none
	ChestSlots     sim.Inventory // what that chest holds

	Notice    sim.NoticeKind
	NoticeArg uint8
	NoticeSeq uint32
}

// PlayerView is what everyone can see of a player.
type PlayerView struct {
	ID        sim.PlayerID
	Name      string
	X, Y      float32
	Dir       sim.Dir
	Dead      bool
	Acting    bool
	HasAxe    bool           // carries an axe
	AxeTier   sim.Tier       // what the best axe carried is made of
	Armor     sim.Tier       // what the best armour carried is made of; 0 (stone) means none
	Held      sim.ItemKind   // what is in the selected hotbar slot, drawn in the hand
	Asleep    bool           // in bed: drawn lying down
	Look      sim.Look       // body and hair
	Mount     sim.CreatureID // the beast being ridden, 0 when on foot: drawn on its back
	DinoArmor bool           // wears dinosaur-hide armour
	Boat      sim.BoatID     // the boat they are aboard, 0 if none: drawn in it
	Swimming  bool           // in the water: drawn swimming

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
	Fuel   int32 // campfires, furnaces and standing torches; a man-eater's ticks till it may snap again
	Var    uint8 // a body's or trophy's creature kind, a flower's colour, a seedling's seed, a glowing plant's look
}

// CreatureView is what clients see of an animal.
type CreatureView struct {
	ID       sim.CreatureID
	Kind     sim.CreatureKind
	X, Y     float32
	State    sim.CreatureState
	FaceLeft bool

	Owner  sim.PlayerID // the online player it belongs to, 0 if wild or its owner is away
	Tamed  bool         // it belongs to someone
	Tame   uint8        // meals towards taming it, while wild
	Stay   bool         // a pet told to wait
	Ridden bool         // someone is riding it: not a target
}
