package sim

import "math"

// The great beasts' moves. Each kind fights with two to four moves of its
// own (skills): a bite, a tail sweep, a roar, a charge, a leap. Every move is
// wound up first, in a posture the player learns to read (the client draws
// it: dinoskills.go), and lands only when the wind-up is done, on whoever is
// still where it strikes: a cone before the beast, an arc behind it, a line
// it charges along, a circle round it or round where it leaps. A light move
// is wound up for 0.4–0.5 s, a heavy one 0.8–1 s with a sound of its own, a
// special one 1.2–1.5 s with a red zone on the ground showing where it will
// land (Zones). After a move the beast is open for a moment (its recovery):
// the time to strike back.
//
// A beast picks a move by its situation (how far its foe is, whether the foe
// is behind it, its health, whether the foe is in the water...), at random
// among those ready, each with a cooldown of its own. Packs take turns: only
// one or two of a pack go in at a time (packBusy), the others circle. A heavy
// blow while a small beast winds up knocks it out of its move (staggered).
// What a move does to a player besides damage (stun, bleeding, slowing,
// pinned down) is in status.go; who uses which move when, in skillai.go.

// SkillID names one move of one kind of beast.
type SkillID uint8

const (
	SkillNone SkillID = iota

	SkillRexBite   // Cắn nghiền: head back, jaws open, then a crushing bite in a cone before it
	SkillRexRoar   // Gầm choáng: rears up, chest out; stuns everyone close, small animals run
	SkillRexTail   // Quật đuôi: half turned, the tail drawn back, swept through an arc behind it
	SkillRexCharge // Lao húc: badly hurt, head down and scraping, then an 8-tile charge

	SkillDakotaPounce  // Vồ: crouched, tail up, hips wiggling; a leap that pins the player
	SkillDakotaSickle  // Vuốt liềm: a foot raised, two quick slashes
	SkillDakotaScreech // Gọi bầy vây: the pack leader screeches; the pack spreads round the foe

	SkillAcheroSnap // Đớp: the neck ducked, a quick bite, then a hop back

	SkillTroodonAmbush // Phục kích: eyes glinting in the dark, then a lunge
	SkillTroodonVenom  // Cắn độc: a hiss, then a bite that slows
	SkillTroodonCall   // Gọi bầy: a shrill cry that brings the others

	SkillPteroDive   // Bổ nhào: holds still high over the foe, then dives to peck
	SkillPteroSnatch // Cướp mồi: swoops on food in the hand or on the ground

	SkillQuetzalStab   // Mổ đâm: the neck drawn back, a stab straight ahead
	SkillQuetzalBuffet // Vỗ cánh: wings spread, a buffet that throws back

	SkillPlesioNeck   // Vươn cổ: the neck reared out of the water, a bite
	SkillPlesioAmbush // Phục kích dưới nước: a ripple coming closer, then up from under

	SkillSabrePounce // Vồ rình: low to the ground, tail twitching; a long pounce
	SkillSabreBite   // Cắn răng kiếm: jaws open wide; the bite bleeds

	SkillTrikeCharge // Húc lao: head down, scraping, then a charge that throws
	SkillTrikeToss   // Hất sừng: head down, a toss at close range
	SkillTrikeCall   // Gọi đàn: a bellow when hit: the herd turns on the attacker

	SkillToroDisplay // Phô diềm: the frill flushes, the head shakes; a warning only
	SkillToroCharge  // Húc mạnh: as the three-horn's, stronger

	SkillStegoFlush // Phiến đỏ: the plates flush red, then it turns its tail
	SkillStegoTail  // Đuôi gai: the tail curled up, then swung: it bleeds

	SkillBrachioStomp // Dậm đất: rears up on its hind legs, comes down: a shockwave that stuns
	SkillBrachioWhip  // Quất đuôi: the tail drawn back, a wide arc behind

	SkillAnkyloClub   // Quật chùy: crouched, the club drawn back, swung round
	SkillAnkyloSlam   // Nện chùy: the club raised high, slammed down: stuns
	SkillAnkyloTurtle // Rụt đầu: badly hurt, head in, it shrugs off most of a blow

	SkillAnzuKick    // Đá móng: leans back, a leg lifted, then a kick
	SkillAnzuDisplay // Xù mào: wings spread, crest flushed, hissing; then two quick pecks

	SkillThescKick   // Đá hậu: flinches back, kicks, then runs
	SkillOrnithoKick // Đá hậu

	SkillMammothCharge  // Húc ngà: head down, ears flared, then a charge
	SkillMammothSwipe   // Quất vòi: the trunk curled up, swung sideways: it throws
	SkillMammothTrumpet // Rống gọi đàn: the trunk raised: the herd comes

	NumSkills
)

// SkillPhase is where a beast is in a move.
type SkillPhase uint8

const (
	PhaseNone    SkillPhase = iota
	PhaseWindup             // drawing back: its posture warns of what is coming
	PhaseActive             // the blow lands; a leap, a charge or a pin is under way
	PhaseRecover            // after it: open to a counter
)

// SkillClass is how big a move is, which sets its wind-up and its warning.
type SkillClass uint8

const (
	SkillLight   SkillClass = iota // 0.4–0.5 s: a clear change of posture
	SkillHeavy                     // 0.8–1 s: posture, a sound of its own, dust or a glint
	SkillSpecial                   // 1.2–1.5 s: as heavy, and a red zone where it lands
	SkillCry                       // no blow: a call, a warning, a display
)

// ZoneShape is the shape of the ground a move strikes.
type ZoneShape uint8

const (
	ZoneNone   ZoneShape = iota
	ZoneCone             // from the beast's centre: Reach long, Width radians each side of its aim
	ZoneCircle           // round a point: Reach is its radius
	ZoneLine             // from the beast's centre along its aim: Reach long, Width tiles each side
)

// Effect is what a move does besides its damage.
type Effect uint16

const (
	FxPin      Effect = 1 << iota // holds the foe down and bites, till they break free (status.go)
	FxSteal                       // takes food from the hand, or off the ground
	FxCall                        // brings the herd or the pack against the foe
	FxFlank                       // the pack spreads round the foe and comes in one at a time
	FxScare                       // small animals near by run
	FxWarn                        // a warning only: it turns on a foe who did not back off
	FxTurnTail                    // it ends with its tail to the foe
	FxTurtle                      // head in: a blow does a quarter through the active ticks
)

// Motion is how a beast moves through a move.
type Motion uint8

const (
	MoveStill   Motion = iota
	MoveDash           // charges along its aim through the active ticks, striking whoever it runs into
	MoveLeap           // leaps on the point it aimed at, landing at the end of the active ticks
	MoveHover          // a flier holding still high over where it will dive
	MoveSwim           // closes in under water through the wind-up: a ripple coming closer
	MoveHopBack        // springs back from the foe through its recovery
)

// Anchor is where a round zone lies.
type Anchor uint8

const (
	AtSelf   Anchor = iota
	AtAhead         // Offset tiles before it
	AtBehind        // Offset tiles behind it
	AtTarget        // on the foe as it aimed (a leap, a dive, a swoop)
)

// Need is something a move needs besides its range.
type Need uint8

const (
	NeedNone    Need = iota
	NeedLeader       // the leader of its pack, with others near
	NeedFood         // food in the foe's hand or on the ground by them
	NeedWet          // the foe in the water or a boat
	NeedUp           // a sea beast up out of the water
	NeedHit          // just hit, while it was calm (a herd call, a kick)
	NeedCalm         // nobody has angered it yet (a warning)
	NeedAngry        // angered
	NeedFacing       // the foe before it, not at its tail
)

// SkillDef is one move. Times are in ticks, distances in tiles.
type SkillDef struct {
	Kind  CreatureKind
	Name  string // for the dev console: skill <beast> <name>
	Label string // what it is called in the game
	Class SkillClass

	Windup, Active, Recover int

	Shape  ZoneShape
	Reach  float64
	Width  float64
	Anchor Anchor
	Offset float64
	Zone   bool // a red zone shows on the ground through the wind-up
	Behind bool // strikes behind it (a tail): used on a foe at its tail, which it does not turn to

	Damage            float64
	Hits              int // blows in the active ticks, Every apart (0: one)
	Every             int
	Knock             float64 // tiles it throws the player
	Stun, Bleed, Slow int     // ticks of each on the player (status.go)
	Fx                Effect
	Motion            Motion
	Speed             float64 // tiles a tick, for a dash
	Back              float64 // tiles it hops back through its recovery

	Cooldown   int
	MinD, MaxD float64 // the foe's distance for it to be used; a leap's longest
	HPBelow    float64 // used only at this share of its health or less (0: any)
	Stages     uint8   // the stages of the fight it is used in, a bit each (see SkillStages); 0: every stage
	Need       Need
	Lock       int  // ticks before it lands that its aim stops following the foe; -1: fixed when it starts
	Stagger    bool // a heavy blow while it winds up knocks it out of the move
	Weight     int  // how often it is picked among the moves ready
	Torch      int  // extra wind-up at a foe holding a burning torch: it is seen coming

	Wind  CreatureState // its state through the wind-up; 0: CreatureStrike
	After CreatureState // its state when done; 0: back to what it was doing
	AfterT int
}

// Skill tuning.
const (
	maxKindSkills  = 8 // a boss may have more: see Stages
	StaggerDamage  = 24                     // a blow this heavy knocks a small beast out of its wind-up
	StaggerTicks   = 3 * TicksPerSecond / 5 // and it reels this long: a free blow
	FlankTicks     = 3 * TicksPerSecond     // a pack spreads round its foe this long after the screech
	ScareTicks     = 3 * TicksPerSecond     // small animals run from a roar this long
	ScareDist      = 10.0
	CallDist       = 16.0 // a herd call carries this far
	PackCallDist   = 30.0 // a Troodon's cry, further: the night is quiet
	WarnBackOff    = 4.5  // a warned foe further than this got the message
	TurtleShare    = 0.25 // a blow on an Ankylosaurus with its head in does this share
	PinHold        = 3 * TicksPerSecond
	PinBiteEvery   = 14
	PinBite        = 5.0
)

// skillDefs is every move, by id.
var skillDefs = [NumSkills]SkillDef{
	SkillRexBite: {Kind: CreatureRex, Name: "bite", Label: "Cắn nghiền", Class: SkillHeavy,
		Windup: 14, Active: 2, Recover: 14, Shape: ZoneCone, Reach: 3.4, Width: 0.7,
		Damage: 34, Cooldown: 70, MaxD: 3.3, Lock: 4, Weight: 6},
	SkillRexRoar: {Kind: CreatureRex, Name: "roar", Label: "Gầm choáng", Class: SkillHeavy,
		Windup: 20, Active: 6, Recover: 14, Shape: ZoneCircle, Reach: 6, Zone: true,
		Stun: TicksPerSecond, Fx: FxScare, Cooldown: 300, MaxD: 6, Lock: -1, Weight: 1, Wind: CreatureRoar},
	SkillRexTail: {Kind: CreatureRex, Name: "tail", Label: "Quật đuôi", Class: SkillHeavy,
		Windup: 16, Active: 3, Recover: 14, Shape: ZoneCone, Reach: 3.8, Width: 1.1, Behind: true,
		Damage: 28, Knock: 1.5, Cooldown: 100, MaxD: 3.8, Lock: -1, Weight: 8},
	SkillRexCharge: {Kind: CreatureRex, Name: "charge", Label: "Lao húc", Class: SkillSpecial,
		Windup: 24, Active: 20, Recover: 20, Shape: ZoneLine, Reach: 8, Width: 0.9, Zone: true,
		Damage: 45, Knock: 2.5, Motion: MoveDash, Speed: 0.45, Cooldown: 200, MinD: 3, MaxD: 8,
		HPBelow: 0.4, Lock: -1, Weight: 10},

	SkillDakotaPounce: {Kind: CreatureRaptor, Name: "pounce", Label: "Vồ", Class: SkillHeavy,
		Windup: 12, Active: 8, Recover: 14, Shape: ZoneCircle, Reach: 0.9, Anchor: AtTarget,
		Damage: 8, Fx: FxPin, Motion: MoveLeap, Cooldown: 100, MinD: 2.2, MaxD: 4.2, Lock: 6, Stagger: true, Weight: 6},
	SkillDakotaSickle: {Kind: CreatureRaptor, Name: "sickle", Label: "Vuốt liềm", Class: SkillLight,
		Windup: 8, Active: 6, Recover: 12, Shape: ZoneCone, Reach: 1.9, Width: 0.9,
		Damage: 7, Hits: 2, Every: 5, Cooldown: 50, MaxD: 1.9, Lock: 3, Stagger: true, Weight: 6},
	SkillDakotaScreech: {Kind: CreatureRaptor, Name: "screech", Label: "Gọi bầy vây", Class: SkillCry,
		Windup: 12, Active: 2, Recover: 6, Fx: FxFlank, Cooldown: 300, MinD: 2.5, MaxD: 9,
		Need: NeedLeader, Lock: -1, Stagger: true, Weight: 20},

	SkillAcheroSnap: {Kind: CreatureAchero, Name: "snap", Label: "Đớp", Class: SkillLight,
		Windup: 7, Active: 2, Recover: 14, Shape: ZoneCone, Reach: 1.6, Width: 0.8,
		Damage: 7, Motion: MoveHopBack, Back: 2, Cooldown: 40, MaxD: 1.6, Lock: 2, Stagger: true, Weight: 1},

	SkillTroodonAmbush: {Kind: CreatureTroodon, Name: "ambush", Label: "Phục kích", Class: SkillHeavy,
		Windup: 20, Active: 7, Recover: 14, Shape: ZoneLine, Reach: 3.8, Width: 0.6,
		Damage: 10, Motion: MoveDash, Speed: 0.6, Cooldown: 140, MinD: 2, MaxD: 3.8, Lock: 4,
		Stagger: true, Weight: 4, Torch: 8},
	SkillTroodonVenom: {Kind: CreatureTroodon, Name: "venom", Label: "Cắn độc", Class: SkillLight,
		Windup: 10, Active: 2, Recover: 12, Shape: ZoneCone, Reach: 1.4, Width: 0.8,
		Damage: 4, Slow: 5 * TicksPerSecond, Cooldown: 80, MaxD: 1.4, Lock: 3, Stagger: true, Weight: 4},
	SkillTroodonCall: {Kind: CreatureTroodon, Name: "call", Label: "Gọi bầy", Class: SkillCry,
		Windup: 16, Active: 2, Recover: 8, Fx: FxCall, Cooldown: 600, MinD: 3, MaxD: 14,
		Need: NeedLeader, Lock: -1, Stagger: true, Weight: 20},

	SkillPteroDive: {Kind: CreaturePtero, Name: "dive", Label: "Bổ nhào", Class: SkillHeavy,
		Windup: 16, Active: 3, Recover: 14, Shape: ZoneCircle, Reach: 0.9,
		Damage: 9, Motion: MoveHover, Cooldown: 120, MaxD: 0.8, Lock: -1, Stagger: true, Weight: 3,
		Wind: CreatureSoar, After: CreatureFlee, AfterT: PteroClimbTicks},
	SkillPteroSnatch: {Kind: CreaturePtero, Name: "snatch", Label: "Cướp mồi", Class: SkillLight,
		Windup: 10, Active: 2, Recover: 6, Shape: ZoneCircle, Reach: 1.2,
		Damage: 2, Fx: FxSteal, Motion: MoveHover, Cooldown: 200, MaxD: 0.8, Need: NeedFood, Lock: -1,
		Stagger: true, Weight: 10, Wind: CreatureSoar, After: CreatureFlee, AfterT: 2 * PteroClimbTicks},

	SkillQuetzalStab: {Kind: CreatureQuetzal, Name: "stab", Label: "Mổ đâm", Class: SkillHeavy,
		Windup: 16, Active: 2, Recover: 14, Shape: ZoneLine, Reach: 3.4, Width: 0.5,
		Damage: 22, Cooldown: 60, MaxD: 3.2, Lock: 3, Weight: 6},
	SkillQuetzalBuffet: {Kind: CreatureQuetzal, Name: "buffet", Label: "Vỗ cánh", Class: SkillHeavy,
		Windup: 20, Active: 3, Recover: 16, Shape: ZoneCone, Reach: 3, Width: 1,
		Damage: 5, Knock: 3, Cooldown: 160, MaxD: 2, Lock: -1, Weight: 5},

	SkillPlesioNeck: {Kind: CreaturePlesio, Name: "neck", Label: "Vươn cổ", Class: SkillHeavy,
		Windup: 14, Active: 2, Recover: 14, Shape: ZoneCone, Reach: 3, Width: 0.6,
		Damage: 18, Cooldown: 50, MaxD: 2.9, Need: NeedUp, Lock: 3, Weight: 5},
	SkillPlesioAmbush: {Kind: CreaturePlesio, Name: "ambush", Label: "Phục kích dưới nước", Class: SkillSpecial,
		Windup: 24, Active: 3, Recover: 16, Shape: ZoneCircle, Reach: 1.4, Zone: true,
		Damage: 24, Knock: 1.5, Motion: MoveSwim, Cooldown: 200, MaxD: 7, Need: NeedWet, Lock: 10, Weight: 5,
		Wind: CreatureLurk, After: CreatureIdle, AfterT: PlesioSurfaceTicks},

	SkillSabrePounce: {Kind: CreatureSabre, Name: "pounce", Label: "Vồ rình", Class: SkillHeavy,
		Windup: 20, Active: 10, Recover: 16, Shape: ZoneCircle, Reach: 1, Anchor: AtTarget,
		Damage: 22, Knock: 1, Motion: MoveLeap, Cooldown: 120, MinD: 2.5, MaxD: 5.2, Lock: 8, Stagger: true, Weight: 6},
	SkillSabreBite: {Kind: CreatureSabre, Name: "bite", Label: "Cắn răng kiếm", Class: SkillLight,
		Windup: 12, Active: 2, Recover: 14, Shape: ZoneCone, Reach: 1.8, Width: 0.8,
		Damage: 16, Bleed: 5 * TicksPerSecond, Cooldown: 60, MaxD: 1.8, Lock: 3, Stagger: true, Weight: 5},

	SkillTrikeCharge: {Kind: CreatureTrike, Name: "charge", Label: "Húc lao", Class: SkillHeavy,
		Windup: 18, Active: 14, Recover: 18, Shape: ZoneLine, Reach: 6, Width: 0.85,
		Damage: 26, Knock: 2.5, Motion: MoveDash, Speed: 0.45, Cooldown: 120, MinD: 2.5, MaxD: 6.5, Lock: 6, Weight: 6},
	SkillTrikeToss: {Kind: CreatureTrike, Name: "toss", Label: "Hất sừng", Class: SkillLight,
		Windup: 10, Active: 2, Recover: 12, Shape: ZoneCone, Reach: 2.6, Width: 0.8,
		Damage: 18, Knock: 1.2, Cooldown: 50, MaxD: 2.5, Lock: 3, Weight: 6},
	SkillTrikeCall: {Kind: CreatureTrike, Name: "call", Label: "Gọi đàn", Class: SkillCry,
		Windup: 10, Active: 2, Recover: 6, Fx: FxCall, Cooldown: 400, MaxD: 99, Need: NeedHit, Lock: -1, Weight: 50},

	SkillToroDisplay: {Kind: CreatureToro, Name: "display", Label: "Phô diềm", Class: SkillCry,
		Windup: 20, Active: 2, Recover: 6, Fx: FxWarn, Cooldown: 120, MaxD: 3.5, Need: NeedCalm, Lock: -1, Weight: 1,
		Wind: CreatureRoar},
	SkillToroCharge: {Kind: CreatureToro, Name: "charge", Label: "Húc mạnh", Class: SkillHeavy,
		Windup: 20, Active: 16, Recover: 20, Shape: ZoneLine, Reach: 7, Width: 0.95,
		Damage: 32, Knock: 3.2, Motion: MoveDash, Speed: 0.45, Cooldown: 100, MaxD: 7, Need: NeedAngry, Lock: 6, Weight: 1},

	SkillStegoFlush: {Kind: CreatureStego, Name: "flush", Label: "Phiến đỏ", Class: SkillCry,
		Windup: 20, Active: 2, Recover: 4, Fx: FxTurnTail, Cooldown: 60, MaxD: 5, Need: NeedFacing, Lock: -1, Weight: 1,
		Wind: CreatureRoar},
	SkillStegoTail: {Kind: CreatureStego, Name: "tail", Label: "Đuôi gai", Class: SkillHeavy,
		Windup: 18, Active: 3, Recover: 18, Shape: ZoneCone, Reach: 3.2, Width: 1.2, Behind: true,
		Damage: 22, Bleed: 6 * TicksPerSecond, Cooldown: 60, MaxD: 3.2, Lock: -1, Weight: 1},

	SkillBrachioStomp: {Kind: CreatureBrachio, Name: "stomp", Label: "Dậm đất", Class: SkillSpecial,
		Windup: 24, Active: 4, Recover: 24, Shape: ZoneCircle, Reach: 4, Anchor: AtAhead, Offset: 1.4, Zone: true,
		Damage: 20, Stun: 3 * TicksPerSecond / 2, Cooldown: 200, MaxD: 4.5, Need: NeedFacing, Lock: -1, Weight: 4},
	SkillBrachioWhip: {Kind: CreatureBrachio, Name: "whip", Label: "Quất đuôi", Class: SkillHeavy,
		Windup: 20, Active: 3, Recover: 16, Shape: ZoneCone, Reach: 5.5, Width: 1.3, Behind: true,
		Damage: 20, Knock: 2, Cooldown: 120, MaxD: 5.5, Lock: -1, Weight: 6},

	SkillAnkyloClub: {Kind: CreatureAnkylo, Name: "club", Label: "Quật chùy", Class: SkillHeavy,
		Windup: AnkyloWindup, Active: 3, Recover: 12, Shape: ZoneCone, Reach: 2.9, Width: 1.2, Behind: true,
		Damage: AnkyloClubDamage, Knock: AnkyloClubKnock, Cooldown: AnkyloClubCD, MaxD: AnkyloClubRange, Lock: -1, Weight: 6,
		After: CreatureRoar, AfterT: AnkyloGuardTicks},
	SkillAnkyloSlam: {Kind: CreatureAnkylo, Name: "slam", Label: "Nện chùy", Class: SkillSpecial,
		Windup: 24, Active: 4, Recover: 24, Shape: ZoneCircle, Reach: 3, Anchor: AtBehind, Offset: 2, Zone: true,
		Damage: 24, Stun: 6 * TicksPerSecond / 5, Behind: true, Cooldown: 200, MaxD: 4, Need: NeedAngry, Lock: -1, Weight: 3,
		After: CreatureRoar, AfterT: AnkyloGuardTicks},
	SkillAnkyloTurtle: {Kind: CreatureAnkylo, Name: "turtle", Label: "Rụt đầu", Class: SkillCry,
		Windup: 4, Active: 6 * TicksPerSecond, Recover: 10, Fx: FxTurtle, Cooldown: 20 * TicksPerSecond, MaxD: 99,
		HPBelow: 0.35, Lock: -1, Weight: 30, After: CreatureRoar, AfterT: AnkyloGuardTicks},

	SkillAnzuKick: {Kind: CreatureAnzu, Name: "kick", Label: "Đá móng", Class: SkillLight,
		Windup: 12, Active: 2, Recover: 12, Shape: ZoneCone, Reach: 2, Width: 0.8,
		Damage: 12, Knock: 0.8, Cooldown: 40, MaxD: 1.9, Lock: 3, Stagger: true, Weight: 6},
	SkillAnzuDisplay: {Kind: CreatureAnzu, Name: "display", Label: "Xù mào", Class: SkillHeavy,
		Windup: 16, Active: 10, Recover: 12, Shape: ZoneCone, Reach: 1.9, Width: 0.9,
		Damage: 6, Hits: 2, Every: 5, Cooldown: 160, MaxD: 3.5, Lock: 3, Stagger: true, Weight: 4, Wind: CreatureRoar},

	SkillThescKick: {Kind: CreatureThesc, Name: "kick", Label: "Đá hậu", Class: SkillLight,
		Windup: 8, Active: 2, Recover: 8, Shape: ZoneCone, Reach: 1.9, Width: 0.9,
		Damage: 6, Knock: 1.2, Cooldown: 100, MaxD: 2, Need: NeedHit, Lock: 3, Stagger: true, Weight: 1,
		After: CreatureFlee, AfterT: 3 * TicksPerSecond},
	SkillOrnithoKick: {Kind: CreatureOrnitho, Name: "kick", Label: "Đá hậu", Class: SkillLight,
		Windup: 8, Active: 2, Recover: 8, Shape: ZoneCone, Reach: 1.9, Width: 0.9,
		Damage: 6, Knock: 1.2, Cooldown: 100, MaxD: 2, Need: NeedHit, Lock: 3, Stagger: true, Weight: 1,
		After: CreatureFlee, AfterT: OrnithoSprintTicks},

	SkillMammothCharge: {Kind: CreatureMammoth, Name: "charge", Label: "Húc ngà", Class: SkillHeavy,
		Windup: 18, Active: 15, Recover: 18, Shape: ZoneLine, Reach: 6, Width: 1,
		Damage: 28, Knock: 2.5, Motion: MoveDash, Speed: 0.42, Cooldown: 120, MinD: 2.5, MaxD: 6.5, Lock: 6, Weight: 6},
	SkillMammothSwipe: {Kind: CreatureMammoth, Name: "swipe", Label: "Quất vòi", Class: SkillLight,
		Windup: 12, Active: 3, Recover: 12, Shape: ZoneCone, Reach: 2.9, Width: 1,
		Damage: 16, Knock: 2, Cooldown: 50, MaxD: 2.8, Lock: 3, Weight: 6},
	SkillMammothTrumpet: {Kind: CreatureMammoth, Name: "trumpet", Label: "Rống gọi đàn", Class: SkillCry,
		Windup: 16, Active: 4, Recover: 8, Fx: FxCall, Cooldown: 400, MaxD: 99, Need: NeedHit, Lock: -1, Weight: 50},
}

// kindSkills lists each kind's moves, in the order of skillDefs.
var kindSkills [NumCreatureKinds][]SkillID

// skillStages splits a kind's fight into stages by its health: the shares
// of its full health at which each next stage begins, highest first. A
// boss with stages {0.66, 0.33} fights in stage 0 above two thirds, 1 down
// to a third and 2 below; a move's Stages says which of them it is used in.
// Kinds with none fight in one stage, 0.
var skillStages [NumCreatureKinds][]float64

// SetSkillStages gives kind k the stages of its fight (see skillStages): a
// boss's moves change as it weakens.
func SetSkillStages(k CreatureKind, at ...float64) { skillStages[k] = at }

// addSkills adds moves to the table, for a boss (or any kind) that gets its
// moves from its own file; it returns their ids in order.
func addSkills(defs ...SkillDef) []SkillID {
	var ids []SkillID
	for _, d := range defs {
		id := SkillID(len(extraSkills)) + NumSkills
		extraSkills = append(extraSkills, d)
		kindSkills[d.Kind] = append(kindSkills[d.Kind], id)
		if len(kindSkills[d.Kind]) > maxKindSkills {
			panic("too many skills for " + d.Kind.Name())
		}
		ids = append(ids, id)
	}
	return ids
}

// extraSkills are moves added by addSkills, numbered on from NumSkills.
var extraSkills []SkillDef

// SkillStage is the stage of its fight c is in (see skillStages).
func (c Creature) SkillStage() int {
	n := 0
	for _, at := range skillStages[min(c.Kind, NumCreatureKinds-1)] {
		if float64(c.HP) <= at*float64(c.Kind.MaxHP()) {
			n++
		}
	}
	return n
}

func init() {
	for id := SkillID(1); id < NumSkills; id++ {
		k := skillDefs[id].Kind
		kindSkills[k] = append(kindSkills[k], id)
		if len(kindSkills[k]) > maxKindSkills {
			panic("too many skills for " + k.Name())
		}
	}
	// The windup of the club against a hunter, which still lands through
	// telegraph.go: as against a player.
	windupTicks[CreatureAnkylo] = skillDefs[SkillAnkyloClub].Windup
}

// Skill is move id's description.
func Skill(id SkillID) *SkillDef {
	if id >= NumSkills {
		if i := int(id - NumSkills); i < len(extraSkills) {
			return &extraSkills[i]
		}
		return &skillDefs[SkillNone]
	}
	return &skillDefs[id]
}

// SkillsOf lists the moves of a kind of beast.
func SkillsOf(k CreatureKind) []SkillID {
	if k >= NumCreatureKinds {
		return nil
	}
	return kindSkills[k]
}

// FindSkill finds a move of kind k by its dev name or its label.
func FindSkill(k CreatureKind, name string) (SkillID, bool) {
	f := fold(name)
	for _, id := range SkillsOf(k) {
		if d := Skill(id); fold(d.Name) == f || fold(d.Label) == f {
			return id, true
		}
	}
	return SkillNone, false
}

// PhaseLen is how long phase ph of the move lasts as a rule (a wind-up seen
// coming by torchlight, or a pin, lasts longer: see SkillNow).
func (d *SkillDef) PhaseLen(ph SkillPhase) int {
	switch ph {
	case PhaseWindup:
		return d.Windup
	case PhaseActive:
		return d.Active
	case PhaseRecover:
		return d.Recover
	}
	return 0
}

func (d *SkillDef) windState() CreatureState {
	if d.Wind != 0 {
		return d.Wind
	}
	return CreatureStrike
}

// skillState is a beast's moves: the one it is making, and when each of
// its kind's moves may be used again.
type skillState struct {
	id     SkillID
	phase  SkillPhase
	t      int // ticks into the phase
	wind   int // ticks this wind-up lasts
	act    int // ticks the active phase lasts
	state  CreatureState
	foe    PlayerID
	x, y   float64 // the zone: a cone's or a line's start, a circle's centre
	ang    float64 // its aim, radians
	sx, sy float64 // where a leap or a dash set off from
	n      int     // blows landed in the active ticks
	hit    bool    // a dash ran into someone; a warning was not heeded
	pin    PlayerID
	struck [4]PlayerID // players a dash has struck already
	prev   CreatureState
	prevT  int

	cd    [maxKindSkills]int16
	flank int  // ticks it circles its foe before it goes in (a pack's screech)
	call  bool // hit while it was calm: it calls the herd first
	hitT  int  // ticks since a player hit it, counting down
}

// SkillNow is the move c is making: which, how far into which phase, and how
// long that phase lasts this time; SkillNone if none.
func (c Creature) SkillNow() (id SkillID, ph SkillPhase, t, length int) {
	s := &c.sk
	if s.id == SkillNone {
		return SkillNone, PhaseNone, 0, 0
	}
	switch s.phase {
	case PhaseWindup:
		length = s.wind
	case PhaseActive:
		length = s.act
	default:
		length = Skill(s.id).Recover
	}
	return s.id, s.phase, s.t, length
}

// Zone is a red warning on the ground: where a special move will land, and
// how soon.
type Zone struct {
	Creature     CreatureID
	Kind         CreatureKind
	Skill        SkillID
	Shape        ZoneShape
	X, Y         float64 // a cone's or a line's start, a circle's centre
	Angle        float64 // the aim, radians (0 east, y down)
	Reach, Width float64 // as in SkillDef
	Left, Total  int     // ticks till it lands, and the whole wind-up
}

// Zones lists the warnings on the ground in the rectangle [x0, x1] x [y0, y1].
func (w *World) Zones(x0, y0, x1, y1 float64) []Zone {
	var out []Zone
	for _, c := range w.fauna.creatures {
		s := &c.sk
		if s.id == SkillNone || c.gone || c.X < x0-10 || c.X > x1+10 || c.Y < y0-10 || c.Y > y1+10 {
			continue
		}
		d := Skill(s.id)
		if !d.Zone || d.Shape == ZoneNone {
			continue
		}
		left := 0
		switch {
		case s.phase == PhaseWindup:
			left = s.wind - s.t
		case s.phase == PhaseActive && d.Motion == MoveDash && s.act > s.t:
			// the charge is under way: its line stays till it is run
		default:
			continue
		}
		out = append(out, Zone{Creature: c.ID, Kind: c.Kind, Skill: s.id, Shape: d.Shape, X: s.x, Y: s.y,
			Angle: s.ang, Reach: d.Reach, Width: d.Width, Left: left, Total: s.wind})
	}
	return out
}

// behind reports whether (x, y) is at c's tail: c faces left or right.
func behind(c *Creature, x float64) bool {
	if c.FaceLeft {
		return x > c.X+0.15
	}
	return x < c.X-0.15
}

// facing is the way c looks, radians: 0 east, pi west.
func facing(c *Creature) float64 {
	if c.FaceLeft {
		return math.Pi
	}
	return 0
}

// angDiff is a - b turned into (-pi, pi].
func angDiff(a, b float64) float64 {
	d := math.Mod(a-b, 2*math.Pi)
	switch {
	case d > math.Pi:
		d -= 2 * math.Pi
	case d <= -math.Pi:
		d += 2 * math.Pi
	}
	return d
}

// covers reports whether a body of radius r at (x, y) is in the zone of
// move d made as s.
func (s *skillState) covers(d *SkillDef, x, y, r float64) bool {
	dx, dy := x-s.x, y-s.y
	switch d.Shape {
	case ZoneCone:
		dist := math.Hypot(dx, dy)
		if dist > d.Reach+r {
			return false
		}
		if dist < 0.6 {
			return true // up against its chest
		}
		return math.Abs(angDiff(math.Atan2(dy, dx), s.ang)) <= d.Width+math.Asin(min(1, r/dist))
	case ZoneCircle:
		return math.Hypot(dx, dy) <= d.Reach+r
	case ZoneLine:
		ux, uy := math.Cos(s.ang), math.Sin(s.ang)
		along, across := dx*ux+dy*uy, math.Abs(-dx*uy+dy*ux)
		return along >= -r && along <= d.Reach+r && across <= d.Width+r
	}
	return false
}

// skillSlot is the index of move id among its kind's moves.
func skillSlot(k CreatureKind, id SkillID) int {
	for i, s := range SkillsOf(k) {
		if s == id {
			return i
		}
	}
	return -1
}

// packKind reports whether k hunts in packs that take turns: one or two go
// in at a time.
func packKind(k CreatureKind) bool {
	return k == CreatureRaptor || k == CreatureAchero || k == CreatureTroodon
}

// packLimit is how many of a pack of kind k strike at one foe at once.
func packLimit(k CreatureKind) int {
	if k == CreatureRaptor {
		return 1
	}
	return 2
}

// packBusy reports whether enough of c's pack are already striking at p:
// c must wait its turn.
func (w *World) packBusy(c *Creature, p *Player) bool {
	n := 0
	for _, o := range w.fauna.creatures {
		if o == c || o.Kind != c.Kind || o.gone || o.sk.id == SkillNone || o.sk.foe != p.ID {
			continue
		}
		if Skill(o.sk.id).Damage > 0 && o.sk.phase != PhaseRecover {
			if n++; n >= packLimit(c.Kind) {
				return true
			}
		}
	}
	return false
}

// packLeader reports whether c leads its pack here: the oldest of its kind
// close by, with at least one other.
func (w *World) packLeader(c *Creature) bool {
	others := 0
	for _, o := range w.fauna.creatures {
		if o == c || o.Kind != c.Kind || o.gone || o.Tamed() || math.Hypot(o.X-c.X, o.Y-c.Y) > 12 {
			continue
		}
		if o.ID < c.ID {
			return false
		}
		others++
	}
	return others > 0
}

// foodNear is whether p holds food, or food lies on the ground by them.
func (w *World) foodNear(p *Player) bool {
	if p.Held().Edible() {
		return true
	}
	found := false
	w.EachDropIn(p.X-2, p.Y-2, p.X+2, p.Y+2, func(d Drop) {
		found = found || (!d.gone && d.Item.Edible())
	})
	return found
}

// ready reports whether c can begin move d (its i-th) at p, dist away.
func (w *World) ready(c *Creature, i int, d *SkillDef, p *Player, dist float64) bool {
	s := &c.sk
	if s.cd[i] > 0 || dist < d.MinD || dist > d.MaxD {
		return false
	}
	if d.HPBelow > 0 && float64(c.HP) > d.HPBelow*float64(c.Kind.MaxHP()) {
		return false
	}
	if d.Stages != 0 && d.Stages&(1<<c.SkillStage()) == 0 {
		return false // not in this stage of the fight
	}
	if d.Behind && !behind(c, p.X) {
		return false
	}
	switch d.Need {
	case NeedLeader:
		if !w.packLeader(c) {
			return false
		}
	case NeedFood:
		if !w.foodNear(p) {
			return false
		}
	case NeedWet:
		if p.boat == 0 && !w.Swimming(p) {
			return false
		}
	case NeedUp:
		if c.State == CreatureLurk {
			return false
		}
	case NeedHit:
		if !s.call {
			return false
		}
	case NeedCalm:
		if c.rage > 0 {
			return false
		}
	case NeedAngry:
		if c.rage <= 0 {
			return false
		}
	case NeedFacing:
		if behind(c, p.X) {
			return false
		}
	}
	if d.Damage > 0 && packKind(c.Kind) && w.packBusy(c, p) {
		return false
	}
	return true
}

// trySkill has c begin one of its moves at p, dist away, if any is ready,
// and reports whether it did. A move at its tail goes before any other: it
// does not have to turn round for it.
func (w *World) trySkill(c *Creature, p *Player, dist float64) bool {
	if p == nil || p.Dead || c.Tamed() || c.rider != 0 || c.sk.id != SkillNone {
		return false
	}
	list := SkillsOf(c.Kind)
	var ok [maxKindSkills]bool
	tail := false
	for i, id := range list {
		d := Skill(id)
		if ok[i] = w.ready(c, i, d, p, dist); ok[i] && d.Behind {
			tail = true
		}
	}
	total := 0
	for i, id := range list {
		if ok[i] && (!tail || Skill(id).Behind) {
			total += max(1, Skill(id).Weight)
		}
	}
	if total == 0 {
		return false
	}
	roll := 0
	if total > 1 {
		roll = w.random().IntN(total)
	}
	for i, id := range list {
		if !ok[i] || tail && !Skill(id).Behind {
			continue
		}
		if roll -= max(1, Skill(id).Weight); roll < 0 {
			w.startSkill(c, id, p)
			return true
		}
	}
	return false
}

// startSkill has c begin move id at p (p may be nil for a move at no one).
func (w *World) startSkill(c *Creature, id SkillID, p *Player) {
	d := Skill(id)
	s := &c.sk
	if i := skillSlot(c.Kind, id); i >= 0 {
		s.cd[i] = int16(d.Cooldown)
	}
	s.prev, s.prevT = c.State, c.timer
	if s.prev == CreatureStrike || s.prev == CreatureHurt || s.prev == CreatureSoar && c.Kind != CreaturePtero {
		s.prev, s.prevT = CreatureChase, 0
	}
	s.id, s.phase, s.t, s.n, s.hit, s.pin = id, PhaseWindup, 0, 0, false, 0
	s.struck = [4]PlayerID{}
	s.foe = 0
	s.wind, s.act = d.Windup, d.Active
	if p != nil {
		s.foe = p.ID
		if d.Torch > 0 && p.torchFuel > 0 {
			s.wind += d.Torch // the torch shows it crouched in the dark a moment sooner
		}
	}
	if d.Need == NeedHit {
		s.call = false
	}
	s.sx, s.sy = c.X, c.Y
	c.State = d.windState()
	s.state = c.State
	w.aimSkill(c, d, p)
}

// aimSkill turns move d of c towards p and sets its zone.
func (w *World) aimSkill(c *Creature, d *SkillDef, p *Player) {
	s := &c.sk
	if p != nil {
		switch {
		case d.Behind:
			s.ang = facing(c) + math.Pi // the tail points away from the head
		case d.Motion == MoveHover || d.Motion == MoveSwim:
			s.ang = math.Atan2(p.Y-c.Y, p.X-c.X)
		default:
			c.faceToward(p.X)
			s.ang = math.Atan2(p.Y-c.Y, p.X-c.X)
			if d.Shape == ZoneCone { // a bite or a kick reaches out before it, not up or down its back
				f := facing(c)
				s.ang = f + max(-1.1, min(1.1, angDiff(s.ang, f)))
			}
		}
	}
	s.x, s.y = c.X, c.Y
	switch d.Anchor {
	case AtAhead:
		s.x += math.Cos(facing(c)) * d.Offset
	case AtBehind:
		s.x -= math.Cos(facing(c)) * d.Offset
	case AtTarget:
		if p != nil {
			dx, dy := p.X-c.X, p.Y-c.Y
			if l := math.Hypot(dx, dy); l > d.MaxD && l > 1e-6 {
				dx, dy = dx/l*d.MaxD, dy/l*d.MaxD
			}
			s.x, s.y = c.X+dx, c.Y+dy
		}
	}
}

// stepSkills runs c's moves for a tick: the cooldowns, and the move it is
// making, if any. It reports whether that was its whole step.
func (w *World) stepSkills(c *Creature) bool {
	s := &c.sk
	for i := range s.cd {
		if s.cd[i] > 0 {
			s.cd[i]--
		}
	}
	if s.flank > 0 {
		s.flank--
	}
	if s.hitT > 0 {
		s.hitT--
	}
	if s.id == SkillNone {
		return false
	}
	if c.Tamed() || c.rider != 0 || c.State != s.state {
		w.cancelSkill(c) // tamed, ridden, snared, knocked about: the move is lost
		return false
	}
	if s.phase != PhaseActive && breaksOff(c) {
		w.cancelSkill(c) // beaten: it gives up the fight, cowers or flies off
		return false
	}
	d := Skill(s.id)
	p := w.players[s.foe]
	if p != nil && (p.Dead || p.Asleep) {
		p = nil
	}
	s.t++
	switch s.phase {
	case PhaseWindup:
		w.windStep(c, d, p)
		if s.t >= s.wind {
			s.phase, s.t = PhaseActive, 0
			c.State, s.state = CreatureStrike, CreatureStrike
			if d.Motion == MoveDash || d.Motion == MoveLeap {
				s.sx, s.sy = c.X, c.Y
			} else {
				w.skillLand(c, d)
			}
		}
	case PhaseActive:
		w.activeStep(c, d, p)
		if s.id != SkillNone && s.phase == PhaseActive && s.t >= s.act && s.pin == 0 {
			s.phase, s.t = PhaseRecover, 0
		}
	case PhaseRecover:
		if d.Motion == MoveHopBack && p != nil {
			if l := math.Hypot(c.X-p.X, c.Y-p.Y); l > 1e-6 {
				step := d.Back / float64(d.Recover)
				face := c.FaceLeft
				w.moveCreature(c, (c.X-p.X)/l*step, (c.Y-p.Y)/l*step)
				c.FaceLeft = face // hopping back, still looking at its foe
			}
		}
		if s.t >= d.Recover {
			w.endSkill(c)
		}
	}
	return true
}

// breaksOff reports whether c is beaten enough to give up the fight for its
// own way out: a raptor cowers, a sabre-tooth too, a Quetzalcoatlus flies.
func breaksOff(c *Creature) bool {
	switch c.Kind {
	case CreatureRaptor, CreatureAchero:
		_, _, _, _, cowerAt := raptorTune(c.Kind)
		return c.HP <= cowerAt*HealthScale
	case CreatureSabre:
		return c.HP <= SabreCowerBelow*HealthScale
	case CreatureQuetzal:
		return c.cool == 0 && float64(c.HP) <= float64(c.Kind.MaxHP())*QuetzalFlyBelow
	}
	return false
}

// windStep is a tick of c winding up move d at p.
func (w *World) windStep(c *Creature, d *SkillDef, p *Player) {
	s := &c.sk
	if p == nil || d.Lock < 0 || s.wind-s.t <= d.Lock {
		return
	}
	if d.Motion == MoveSwim { // closing in under the water
		if dist := math.Hypot(p.X-c.X, p.Y-c.Y); dist > 0.3 {
			step := min(dist-0.3, c.Kind.def().run*1.6)
			w.moveCreature(c, (p.X-c.X)/dist*step, (p.Y-c.Y)/dist*step)
		}
	}
	w.aimSkill(c, d, p)
}

// activeStep is a tick of move d's active phase.
func (w *World) activeStep(c *Creature, d *SkillDef, p *Player) {
	s := &c.sk
	if s.pin != 0 {
		w.pinStep(c, d)
		return
	}
	switch d.Motion {
	case MoveDash:
		moved := w.moveCreature(c, math.Cos(s.ang)*d.Speed, math.Sin(s.ang)*d.Speed)
		s.x, s.y = s.sx, s.sy // the line stays where it was drawn
		w.dashHits(c, d)
		if !moved || s.hit || math.Hypot(c.X-s.sx, c.Y-s.sy) >= d.Reach {
			s.act = s.t // run into someone, into something, or to its end
		}
	case MoveLeap:
		n := float64(d.Active)
		w.moveCreature(c, (s.x-s.sx)/n, (s.y-s.sy)/n)
		if s.t >= d.Active {
			s.x, s.y = c.X, c.Y // where it came down
			w.skillLand(c, d)
		}
	default:
		if d.Hits > 1 && d.Every > 0 && s.t%d.Every == 0 && s.t/d.Every < d.Hits {
			w.aimSkill(c, d, p) // the next blow follows the foe
			w.skillLand(c, d)
		}
	}
}

// dashHits strikes whoever c runs into on its charge, each once.
func (w *World) dashHits(c *Creature, d *SkillDef) {
	s := &c.sk
	r := c.Kind.def().radius + 0.5
	for _, id := range w.playerOrder {
		p := w.players[id]
		if !w.skillTarget(c, p) || math.Hypot(p.X-c.X, p.Y-c.Y) > r || s.struckAlready(p.ID) {
			continue
		}
		w.skillHit(c, d, p, c.X, c.Y)
		s.hit = true
	}
}

func (s *skillState) struckAlready(id PlayerID) bool {
	for i, q := range s.struck {
		if q == id {
			return true
		}
		if q == 0 {
			s.struck[i] = id
			return false
		}
	}
	return true
}

// skillTarget reports whether c's move can strike p at all.
func (w *World) skillTarget(c *Creature, p *Player) bool {
	if p == nil || p.Dead || p.Asleep {
		return false
	}
	if p.boat != 0 && c.Kind != CreaturePlesio {
		return false // out on the water, out of reach
	}
	return true
}

// skillLand lands move d: on every player in its zone, and whatever it does
// besides.
func (w *World) skillLand(c *Creature, d *SkillDef) {
	s := &c.sk
	switch d.Anchor { // a cone, a line or a circle on the beast goes where the beast is now
	case AtSelf, AtAhead, AtBehind:
		if d.Motion != MoveDash {
			ang := s.ang
			w.aimSkill(c, d, nil)
			s.ang = ang
		}
	}
	s.n++
	w.skillCry(c, d)
	if d.Shape == ZoneNone {
		return
	}
	for _, id := range w.playerOrder {
		p := w.players[id]
		if w.skillTarget(c, p) && s.covers(d, p.X, p.Y, PlayerRadius) {
			w.skillHit(c, d, p, s.x, s.y)
		}
	}
}

// skillHit is move d of c landing on p, thrown from (fx, fy).
func (w *World) skillHit(c *Creature, d *SkillDef, p *Player, fx, fy float64) {
	s := &c.sk
	if d.Fx&FxSteal != 0 {
		w.snatch(c, p)
	}
	if d.Damage > 0 {
		if p.boat != 0 {
			if b := w.boatByID(p.boat); b != nil {
				w.damageBoat(b, int(d.Damage/4+0.5)) // a sea beast's bite on a boat
			}
		} else {
			struck(p, d.Damage)
			p.notify(NoticeBitten, uint8(c.Kind))
		}
	}
	if p.Dead {
		return
	}
	if d.Knock > 0 {
		if fx == c.X && fy == c.Y || d.Shape != ZoneCircle {
			fx, fy = c.X, c.Y
		}
		w.shove(p, fx, fy, d.Knock)
	}
	p.stunFor(d.Stun)
	p.bleedFor(d.Bleed)
	p.slowFor(d.Slow)
	if d.Fx&FxPin != 0 && p.ID == s.foe && p.status.pin == 0 && p.mount == 0 && p.boat == 0 {
		p.status.pin, p.status.mash = c.ID, 0
		s.pin = p.ID
		s.act = s.t + PinHold
	}
}

// skillCry is what move d does besides striking: a call, a warning, a turn.
func (w *World) skillCry(c *Creature, d *SkillDef) {
	s := &c.sk
	p := w.players[s.foe]
	if d.Fx&FxCall != 0 && p != nil {
		far := CallDist
		if c.Kind == CreatureTroodon {
			far = PackCallDist
		}
		for _, o := range w.fauna.creatures {
			if o == c || o.Kind != c.Kind || o.gone || o.Tamed() || math.Hypot(o.X-c.X, o.Y-c.Y) > far {
				continue
			}
			o.foe, o.rage = p.ID, HerdRageTicks
			o.faceToward(p.X)
			if o.Kind == CreatureTroodon && o.sk.id == SkillNone {
				o.State = CreatureChase
			}
		}
		if c.Kind == CreatureTroodon {
			w.troodonsCalled(p) // more come out of the dark: see nightstalker.go
		}
	}
	if d.Fx&FxFlank != 0 && p != nil {
		for _, o := range w.fauna.creatures {
			if o != c && o.Kind == c.Kind && !o.gone && !o.Tamed() && math.Hypot(o.X-c.X, o.Y-c.Y) <= 12 {
				o.sk.flank = FlankTicks
				o.foe, o.rage = p.ID, max(o.rage, HerdRageTicks)
			}
		}
	}
	if d.Fx&FxScare != 0 {
		for _, o := range w.fauna.creatures {
			if o != c && !o.gone && !o.Tamed() && smallAnimal(o.Kind) && math.Hypot(o.X-c.X, o.Y-c.Y) <= ScareDist {
				o.hunter, o.huntedT = c.ID, ScareTicks // it runs: see evade in needs.go
			}
		}
	}
	if d.Fx&FxWarn != 0 && p != nil {
		if math.Hypot(p.X-c.X, p.Y-c.Y) <= WarnBackOff {
			c.foe, c.rage = p.ID, HerdRageTicks // it was not heeded
			s.hit = true
		}
	}
	if d.Fx&FxTurnTail != 0 && p != nil {
		c.FaceLeft = p.X > c.X
	}
}

// smallAnimal reports whether k runs from a tyrant's roar.
func smallAnimal(k CreatureKind) bool {
	switch k {
	case CreatureRabbit, CreatureCrab, CreatureOrnitho, CreatureThesc, CreatureMonkey, CreatureToad,
		CreatureGlowToad, CreatureAchero, CreatureTroodon, CreatureSnake:
		return true
	}
	return false
}

// snatch has c take food from p's hand, or off the ground near by.
func (w *World) snatch(c *Creature, p *Player) {
	if k := p.Held(); k.Edible() {
		p.Inv.TakeSlot(p.Selected, 1)
		p.notify(NoticeStolen, uint8(k))
		return
	}
	var best *Drop
	bd := 2.5
	for _, d := range w.drops {
		if dd := math.Hypot(d.X-c.X, d.Y-c.Y); !d.gone && d.Item.Edible() && dd < bd {
			best, bd = d, dd
		}
	}
	if best != nil {
		if best.Count--; best.Count <= 0 {
			best.gone = true
		}
		p.notify(NoticeStolen, uint8(best.Item))
	}
}

// pinStep is a tick of c holding its foe down: it bites now and then, till
// they break free (status.go) or it lets go.
func (w *World) pinStep(c *Creature, d *SkillDef) {
	s := &c.sk
	p := w.players[s.pin]
	if p == nil || p.Dead || p.status.pin != c.ID || s.t >= s.act {
		w.releasePin(c)
		return
	}
	side := 0.45 // on top of them, from the side it came
	if c.X > p.X {
		side = -side
	}
	c.X, c.Y = p.X-side, p.Y
	c.FaceLeft = side < 0
	if s.t%PinBiteEvery == 0 {
		struck(p, PinBite)
		p.notify(NoticeBitten, uint8(c.Kind))
		if p.Dead {
			w.releasePin(c)
		}
	}
}

// releasePin lets go of whoever c holds, and it recovers.
func (w *World) releasePin(c *Creature) {
	s := &c.sk
	if p := w.players[s.pin]; p != nil && p.status.pin == c.ID {
		p.status.pin = 0
	}
	s.pin = 0
	if s.id != SkillNone && s.phase == PhaseActive {
		s.phase, s.t = PhaseRecover, 0
	}
}

// endSkill finishes c's move.
func (w *World) endSkill(c *Creature) {
	s := &c.sk
	d := Skill(s.id)
	warned := d.Fx&FxWarn != 0 && s.hit
	s.id, s.phase, s.t, s.pin = SkillNone, PhaseNone, 0, 0
	switch {
	case warned:
		c.State, c.timer = CreatureChase, 0
	case d.Fx&FxWarn != 0:
		c.State, c.timer = CreatureIdle, 0
	case d.After != 0:
		c.State, c.timer = d.After, d.AfterT
		if d.After == CreatureFlee {
			c.hx, c.hy = 0, 0
		}
	default:
		c.State, c.timer = s.prev, s.prevT
	}
	if c.Kind == CreatureAnkylo && d.Behind {
		c.cool = AnkyloTurnTicks // swung round, it is slow to turn again
	}
}

// cancelSkill drops c's move at once, letting go of anyone it holds.
func (w *World) cancelSkill(c *Creature) {
	s := &c.sk
	if s.id == SkillNone {
		return
	}
	if s.pin != 0 {
		if p := w.players[s.pin]; p != nil && p.status.pin == c.ID {
			p.status.pin = 0
		}
	}
	s.id, s.phase, s.t, s.pin = SkillNone, PhaseNone, 0, 0
	if c.State == s.state {
		c.State, c.timer = s.prev, s.prevT
	}
}

// staggered is a blow of dmg on c while it makes a move: a heavy one knocks
// a small beast out of its wind-up, or off the player it holds down; any
// blow at all cuts short a cry. It reports whether c reels from it.
func (w *World) staggered(c *Creature, dmg int) bool {
	s := &c.sk
	if s.id == SkillNone {
		return false
	}
	d := Skill(s.id)
	if !d.Stagger || s.phase == PhaseRecover || s.phase == PhaseActive && s.pin == 0 {
		return false
	}
	if dmg < StaggerDamage && d.Class != SkillCry {
		return false
	}
	w.cancelSkill(c)
	c.State, c.timer = CreatureHurt, StaggerTicks
	c.flinchCD = FlinchEvery
	return true
}

// skillProvoked notes that p hit c: a beast hit while calm calls its herd
// first (or kicks out and runs).
func (w *World) skillProvoked(c *Creature, p *Player) {
	if c.rage <= 0 && c.sk.id == SkillNone {
		c.sk.call = true
	}
	c.sk.hitT = TicksPerSecond
}

// turtled reports whether c has its head in, shrugging off most of a blow.
func (c *Creature) turtled() bool {
	return c.sk.id == SkillAnkyloTurtle && c.sk.phase == PhaseActive
}

// inSkill reports whether c is making a move: it does not flinch from a blow.
func (c *Creature) inSkill() bool { return c.sk.id != SkillNone }

// ForceSkill has creature id begin move sk at player pid now, whatever its
// range and cooldown (the dev console's skill command). It reports whether
// it could.
func (w *World) ForceSkill(id CreatureID, sk SkillID, pid PlayerID) bool {
	c := w.creatureByID(id)
	p := w.players[pid]
	if c == nil || p == nil || Skill(sk).Kind != c.Kind {
		return false
	}
	w.cancelSkill(c)
	if Skill(sk).Need == NeedAngry || Skill(sk).Need == NeedHit {
		c.foe, c.rage = p.ID, HerdRageTicks
	}
	if Skill(sk).Need == NeedCalm {
		c.rage = 0
	}
	if c.Kind == CreatureRex {
		c.cool = 0
	}
	if c.State == CreatureHurt || c.State == CreatureCower || c.State == CreatureEat || c.State == CreatureDrink {
		c.State, c.timer = CreatureChase, 0
	}
	if c.Kind == CreaturePtero || c.Kind == CreatureQuetzal && c.State == CreatureSoar {
		if c.Kind == CreaturePtero {
			c.X, c.Y = p.X, p.Y // a dive or a swoop comes from straight over its foe
			c.State = CreatureSoar
		} else {
			c.State = CreatureChase
		}
	}
	if c.Kind == CreaturePlesio && sk == SkillPlesioAmbush {
		c.State = CreatureLurk
	}
	if c.Kind == CreaturePlesio && sk == SkillPlesioNeck && c.State == CreatureLurk {
		c.State, c.timer = CreatureChase, PlesioSurfaceTicks
	}
	if sk == SkillRexTail || sk == SkillStegoTail || sk == SkillBrachioWhip || sk == SkillAnkyloClub || sk == SkillAnkyloSlam {
		c.FaceLeft = p.X > c.X // its tail to them
	}
	w.startSkill(c, sk, p)
	return true
}
