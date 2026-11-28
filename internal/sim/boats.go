package sim

import (
	"math"
	"slices"
)

// Boats and the open sea.
//
// A raft or a sailing boat is made in the bag, then set on the water in
// front with V. R gets aboard the nearest boat, and off it again onto the
// nearest shore. The walk keys steer: a raft is paddled, slow but the same
// in any direction; a sail runs fast before the wind and crawls into it.
// The wind turns with the weather.
//
// Anyone can swim, but not far: after a while in the water a swimmer tires
// and starts to drown. A boat broken by a sea monster sinks, and whoever was
// aboard is in the water.

// BoatID identifies a boat. It is never reused within a world.
type BoatID uint32

// BoatKind is the kind of a boat.
type BoatKind uint8

const (
	BoatNone BoatKind = iota
	BoatRaft
	BoatSail
	NumBoatKinds
)

type boatDef struct {
	name   string
	item   ItemKind
	hp     int
	speed  float64 // tiles per tick at full speed
	sail   bool    // driven by the wind
	radius float64
}

var boatDefs = [NumBoatKinds]boatDef{
	BoatNone: {name: "?"},
	BoatRaft: {name: "Bè gỗ", item: ItemRaft, hp: 60, speed: 3.0 / TicksPerSecond, radius: 0.55},
	BoatSail: {name: "Thuyền buồm", item: ItemSailboat, hp: 140, speed: 7.0 / TicksPerSecond, sail: true, radius: 0.75},
}

// Name is the display name of boats of kind k.
func (k BoatKind) Name() string { return boatDefs[min(k, NumBoatKinds-1)].name }

// MaxHP is how much damage a new boat of kind k takes before it sinks.
func (k BoatKind) MaxHP() int { return boatDefs[min(k, NumBoatKinds-1)].hp }

// Sails reports whether boats of kind k are driven by the wind.
func (k BoatKind) Sails() bool { return boatDefs[min(k, NumBoatKinds-1)].sail }

func boatOf(item ItemKind) BoatKind {
	for k, d := range boatDefs {
		if k > 0 && d.item == item {
			return BoatKind(k)
		}
	}
	return BoatNone
}

// Boat tuning.
const (
	BoatReach      = 1.9                  // R boards a boat this close
	LandReach      = 1.7                  // and lands on shore this close
	BoatAccel      = 0.04                 // share of the way to its wanted speed a boat gets each tick
	BoatTurn       = 0.12                 // radians a boat turns at most each tick
	SailUpwind     = 0.3                  // a sail's speed straight into the wind, as a share of full
	SwimSpeed      = 0.45                 // a swimmer's share of walking speed
	SwimTireTicks  = 8 * TicksPerSecond   // a swimmer can keep going this long
	DrownPerTick   = 3.0 / TicksPerSecond // and then loses this much health a tick
	SwimRecover    = 3                    // ticks of rest given back per tick out of the water
	launchReach    = 2                    // V puts a boat on water at most this many tiles ahead
	windSlotOffset = 0x77a1
)

// Boat is a boat on the water.
type Boat struct {
	ID      BoatID
	Kind    BoatKind
	X, Y    float64
	Heading float64 // radians; 0 points east (right), pi/2 south
	Speed   float64 // tiles per tick, now
	HP      int
	Rider   PlayerID `json:"-"`
	Held    int      `json:"-"` // ticks a giant squid still holds it fast
	gone    bool
}

// Boats returns copies of every boat.
func (w *World) Boats() []Boat {
	out := make([]Boat, 0, len(w.boats))
	for _, b := range w.boats {
		out = append(out, *b)
	}
	return out
}

func (w *World) boatByID(id BoatID) *Boat {
	for _, b := range w.boats {
		if b.ID == id && !b.gone {
			return b
		}
	}
	return nil
}

// WindAt is the direction the wind blows towards at tick (radians, 0 east),
// and how hard, 0 to 1. Like the weather it is a pure function of the seed
// and the tick.
func WindAt(seed uint64, tick uint64) (dir, strength float64) {
	slot := tick / WeatherTicks
	h := hash2(int32(slot), int32(slot>>31), uint32(seed)^uint32(seed>>32)^windSlotOffset)
	return unit(h) * 2 * math.Pi, windStrength(WeatherAt(seed, tick))
}

// windStrength is how hard the wind blows in weather wt, 0 to 1.
func windStrength(wt Weather) float64 {
	switch wt {
	case WeatherStorm, WeatherTornado:
		return 1
	case WeatherWind:
		return 0.9
	case WeatherRain, WeatherSnow:
		return 0.6
	default:
		return 0.45
	}
}

// Wind is the wind now.
func (w *World) Wind() (dir, strength float64) {
	dir, strength = WindAt(w.seed, w.tick)
	if w.dev.forced {
		strength = windStrength(w.dev.weather) // dev mode; see dev.go
	}
	return dir, strength
}

// SailFactor is how much of a sail's full speed it makes heading at an
// angle to a wind of the given direction and strength.
func SailFactor(heading, windDir, strength float64) float64 {
	along := math.Cos(heading - windDir) // 1 before the wind, -1 into it
	f := SailUpwind + (1-SailUpwind)*(along+1)/2
	return f * (0.55 + 0.45*strength)
}

// boatFits reports whether a boat of kind k can float centred at (x, y):
// open water (not pond ice) under it.
func (w *World) boatFits(k BoatKind, x, y float64) bool {
	r := boatDefs[k].radius
	for _, c := range [][2]float64{{-r, -r * 0.6}, {r, -r * 0.6}, {-r, r * 0.6}, {r, r * 0.6}, {0, 0}} {
		tx, ty := int(math.Floor(x+c[0])), int(math.Floor(y+c[1]))
		if !w.inBounds(tx, ty) || !w.tiles[ty*w.width+tx].Swimmable() || w.frozenAt(tx, ty) {
			return false
		}
	}
	return true
}

// launch puts the boat held in p's hand on the water ahead.
func (w *World) launch(p *Player, k BoatKind) {
	dx, dy := p.Dir.Delta()
	for d := 1.0; d <= launchReach+0.01; d += 0.5 {
		x, y := p.X+float64(dx)*d, p.Y+float64(dy)*d
		if !w.boatFits(k, x, y) {
			continue
		}
		p.Inv.TakeSlot(p.Selected, 1)
		w.nextBoat++
		heading := math.Atan2(float64(dy), float64(dx))
		w.boats = append(w.boats, &Boat{ID: w.nextBoat, Kind: k, X: x, Y: y, Heading: heading, HP: k.MaxHP()})
		p.notify(NoticeLaunched, uint8(k))
		return
	}
	p.notify(NoticeNeedWater, uint8(k))
}

// board gets p aboard the nearest free boat, or off the one p is in. It
// reports whether there was a boat to do it with.
func (w *World) board(p *Player) bool {
	if b := w.boatByID(p.boat); b != nil {
		w.disembark(p, b)
		return true
	}
	var best *Boat
	bestD := BoatReach
	for _, b := range w.boats {
		if b.gone || b.Rider != 0 {
			continue
		}
		if d := math.Hypot(b.X-p.X, b.Y-p.Y); d <= bestD {
			best, bestD = b, d
		}
	}
	if best == nil {
		return false
	}
	best.Rider, p.boat = p.ID, best.ID
	p.X, p.Y = best.X, best.Y
	p.swim = 0
	return true
}

// disembark puts p ashore from b, on the nearest ground it can stand on.
func (w *World) disembark(p *Player, b *Boat) {
	bx, by, best := 0.0, 0.0, math.Inf(1)
	for dy := -LandReach; dy <= LandReach; dy += 0.25 {
		for dx := -LandReach; dx <= LandReach; dx += 0.25 {
			x, y := b.X+dx, b.Y+dy
			if d := math.Hypot(dx, dy); d <= LandReach && d < best && w.standable(x, y) {
				bx, by, best = x, y, d
			}
		}
	}
	if math.IsInf(best, 1) {
		p.notify(NoticeNoShore, 0)
		return
	}
	b.Rider, b.Speed = 0, 0
	p.boat = 0
	p.X, p.Y = bx, by
}

// leaveBoat takes p out of a boat without landing them: into the water.
func (w *World) leaveBoat(p *Player) {
	if b := w.boatByID(p.boat); b != nil {
		b.Rider = 0
	}
	p.boat = 0
}

// sailStep moves p's boat by the held input and carries p with it. It
// reports whether p is aboard.
func (w *World) sailStep(p *Player) bool {
	b := w.boatByID(p.boat)
	if b == nil {
		p.boat = 0
		return false
	}
	def := boatDefs[b.Kind]
	want := 0.0
	if mx, my := float64(p.input.MoveX), float64(p.input.MoveY); mx != 0 || my != 0 {
		goal := math.Atan2(my, mx)
		turn := math.Remainder(goal-b.Heading, 2*math.Pi)
		b.Heading += max(-BoatTurn, min(BoatTurn, turn))
		want = def.speed * sailBoost(&p.Inv) // see legendary.go
		if math.Abs(turn) > math.Pi/2 {
			want *= 0.3 // turning about
		}
		if def.sail {
			dir, strength := w.Wind()
			want *= SailFactor(b.Heading, dir, strength)
		}
	}
	if b.Held > 0 {
		b.Held--
		want = 0 // in the squid's arms
	}
	b.Speed += (want - b.Speed) * BoatAccel * 2
	if want < b.Speed {
		b.Speed += (want - b.Speed) * BoatAccel // slows a little faster than it gathers way
	}
	dx, dy := math.Cos(b.Heading)*b.Speed, math.Sin(b.Heading)*b.Speed
	if w.boatFits(b.Kind, b.X+dx, b.Y) {
		b.X += dx
	} else {
		b.Speed *= 0.5 // aground
	}
	if w.boatFits(b.Kind, b.X, b.Y+dy) {
		b.Y += dy
	} else {
		b.Speed *= 0.5
	}
	p.X, p.Y = b.X, b.Y
	p.Dir = dirToward(math.Cos(b.Heading), math.Sin(b.Heading))
	return true
}

// damageBoat takes dmg from b; at nothing left it sinks, dropping whoever
// was aboard into the sea.
func (w *World) damageBoat(b *Boat, dmg int) {
	b.HP -= dmg
	if b.HP > 0 {
		return
	}
	b.gone = true
	if p := w.players[b.Rider]; p != nil {
		p.boat = 0
		p.notify(NoticeSunk, uint8(b.Kind))
	}
	w.dropAt(b.X, b.Y, []Ingredient{{Item: ItemWood, Count: 2}})
}

func (w *World) stepBoats() {
	for _, b := range w.boats {
		if b.gone {
			continue
		}
		if b.Rider != 0 {
			if p := w.players[b.Rider]; p == nil || p.Dead || p.boat != b.ID {
				b.Rider = 0
			}
		}
		if b.Rider == 0 && b.Speed > 0 { // drifts to a stop
			b.Speed *= 0.9
			dx, dy := math.Cos(b.Heading)*b.Speed, math.Sin(b.Heading)*b.Speed
			if w.boatFits(b.Kind, b.X+dx, b.Y+dy) {
				b.X, b.Y = b.X+dx, b.Y+dy
			}
		}
	}
	w.boats = slices.DeleteFunc(w.boats, func(b *Boat) bool { return b.gone })
}

// --- swimming --------------------------------------------------------------

// swimmable reports whether a player can be at (x, y) at all: on their feet
// or swimming. Only the map's edge stops a swimmer.
func (w *World) swimmable(x, y float64) bool {
	r := PlayerRadius
	ok := func(x, y float64) bool {
		tx, ty := int(math.Floor(x)), int(math.Floor(y))
		if !w.inBounds(tx, ty) {
			return false
		}
		t := w.tiles[ty*w.width+tx]
		return (t.Swimmable() || t.Walkable()) && !w.blocked(tx, ty)
	}
	return ok(x-r, y-r) && ok(x+r, y-r) && ok(x-r, y+r) && ok(x+r, y+r) && w.jambsClear(x, y)
}

// inWater reports whether (x, y) is open water: someone there swims.
func (w *World) inWater(x, y float64) bool {
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	return w.TileAt(tx, ty).Swimmable() && !w.frozenAt(tx, ty)
}

// stepSwim keeps p's breath in the water: a swimmer tires, and then drowns.
func (w *World) stepSwim(p *Player) {
	if p.boat != 0 || !w.inWater(p.X, p.Y) {
		p.swim = max(0, p.swim-SwimRecover)
		return
	}
	if swimBoost(&p.Inv) == 1 || w.tick%3 != 0 { // the shark's teeth: a third of the time without tiring
		p.swim++
	}
	if p.swim > SwimTireTicks {
		hurt(p, DrownPerTick)
		if w.tick%TicksPerSecond == 0 {
			p.notify(NoticeDrowning, 0)
		}
	}
}

// Swimming reports whether p is in the water, not in a boat.
func (w *World) Swimming(p *Player) bool { return p.boat == 0 && !p.Dead && w.inWater(p.X, p.Y) }

// BoatID is the boat p is aboard, 0 if none.
func (p *Player) BoatID() BoatID { return p.boat }

// Breath is how much of a swimmer's strength is left, 1 fresh to 0 drowning.
func (p *Player) Breath() float64 { return 1 - min(1, float64(p.swim)/SwimTireTicks) }

// --- saving ----------------------------------------------------------------

// BoatSave is a boat as kept in a save.
type BoatSave struct {
	Kind          BoatKind
	X, Y, Heading float64
	HP            int
}

func (w *World) saveBoats() []BoatSave {
	var out []BoatSave
	for _, b := range w.boats {
		if !b.gone {
			out = append(out, BoatSave{Kind: b.Kind, X: b.X, Y: b.Y, Heading: b.Heading, HP: b.HP})
		}
	}
	return out
}

func (w *World) loadBoats(saved []BoatSave) {
	for _, s := range saved {
		if s.Kind == BoatNone || s.Kind >= NumBoatKinds || !finite(s.X, s.Y, s.Heading) || !w.boatFits(s.Kind, s.X, s.Y) {
			continue
		}
		w.nextBoat++
		w.boats = append(w.boats, &Boat{ID: w.nextBoat, Kind: s.Kind, X: s.X, Y: s.Y, Heading: s.Heading, HP: min(max(1, s.HP), s.Kind.MaxHP())})
	}
}
