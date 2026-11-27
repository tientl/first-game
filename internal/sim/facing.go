package sim

import "math"

// Facing: which way a built thing faces (Entity.Face). It is chosen while
// placing (R or the mouse wheel turns the ghost; Command.Face carries it)
// and kept in the save; DirDown, south, towards the viewer, is how
// everything stood before things turned, and what an older save's
// entities face.
//
//   - Four facings, each its own drawing: a door (its front; the way
//     through it runs along the facing), a bed (the long side climbed into,
//     the head to its left as one stands before it), a leaf tent (its
//     door), a chest (its lock), a furnace and an oven (their mouths), a
//     mortar (where its pestle leans) and a drying rack (the side its low
//     pole is on: south and north run east–west, east and west north–south).
//   - A run: walls, stone walls, fences of stakes and the grill over a fire
//     lie east–west (DirDown) or north–south (DirRight). Walls and fences
//     join their neighbours into lines whatever their facing; the facing
//     is the run of one standing alone (the client draws the joins).
//   - A profile: a head on a post looks right as drawn (DirDown) or left
//     (DirLeft), since the beasts are drawn from the side.
//
// It matters to play where it should: a door is passed only along its
// way through (DoorBlocks), and a bed, a tent, a furnace and an oven are
// used from before or beside them, not from behind (FrontSide).

// turn is how an entity turns.
type turn uint8

const (
	turnNone   turn = iota
	turnFour        // four facings
	turnRun         // east–west or north–south
	turnMirror      // a profile to the right or the left
)

func turnOf(k EntityKind, v uint8) turn {
	switch k {
	case EntityDoor, EntityBed, EntityLeafTent, EntityChest, EntityFurnace, EntityOven, EntityMortar, EntityDryRack:
		return turnFour
	case EntityWall, EntityStoneWall, EntitySpikes:
		return turnRun
	case EntityCampfire:
		if CookGear(v) == GearGrill {
			return turnRun
		}
	case EntityTrophy:
		return turnMirror
	}
	return turnNone
}

// Turns reports whether an entity of kind k, variant v, takes a facing.
func Turns(k EntityKind, v uint8) bool { return turnOf(k, v) != turnNone }

// FacingOf is facing d as an entity of kind k, variant v, can face: DirDown
// for one that does not turn, a run's two facings folded onto DirDown
// (east–west) and DirRight (north–south), a profile's onto DirDown and
// DirLeft.
func FacingOf(k EntityKind, v uint8, d Dir) Dir {
	d &= 3
	side := d == DirLeft || d == DirRight
	switch turnOf(k, v) {
	case turnFour:
		return d
	case turnRun:
		if side {
			return DirRight
		}
	case turnMirror:
		if d == DirLeft || d == DirUp {
			return DirLeft
		}
	}
	return DirDown
}

// clockwise is the facings in turn, clockwise seen from above: south, west,
// north, east.
var clockwise = [4]Dir{DirDown, DirLeft, DirUp, DirRight}

// NextFacing turns facing d of kind k, variant v, by steps: clockwise seen
// from above for steps > 0 (R, the wheel towards you), back for < 0. A run
// or a profile has two facings, so a step is a flip.
func NextFacing(k EntityKind, v uint8, d Dir, steps int) Dir {
	d = FacingOf(k, v, d)
	switch turnOf(k, v) {
	case turnNone:
		return DirDown
	case turnRun, turnMirror:
		if steps%2 == 0 || d != DirDown {
			if steps%2 == 0 {
				return d
			}
			return DirDown
		}
		return FacingOf(k, v, DirLeft) // the other of the two
	}
	return clockwise[((clockIndex(d)+steps)%4+4)%4]
}

func clockIndex(d Dir) int {
	for i, c := range clockwise {
		if c == d&3 {
			return i
		}
	}
	return 0
}

// RunsNS reports whether a run (a wall, a fence, a grill) facing d lies
// north–south, and a door facing d stands in such a wall: it is passed
// east–west.
func RunsNS(d Dir) bool { return d&3 == DirLeft || d&3 == DirRight }

// DoorJamb is how far a door's posts reach into its tile from the two sides
// its wall runs to, in tiles: wide enough that nobody slips in past them
// sideways, narrow enough that a player walking up to it goes through.
const DoorJamb = 0.14

// DoorBlocks reports whether a box r tiles to each side of (x, y) runs
// into the posts of a door facing d on tile (tx, ty). A door facing south
// or north stands in a wall running east–west, its posts on its east and
// west edges, and is passed north–south; one facing east or west the other
// way about. A box touching a post's edge does not run into it.
func DoorBlocks(d Dir, tx, ty int, x, y, r float64) bool {
	x0, y0, x1, y1 := x-r, y-r, x+r, y+r
	hit := func(px0, py0, px1, py1 float64) bool {
		return x1 > px0 && x0 < px1 && y1 > py0 && y0 < py1
	}
	fx, fy := float64(tx), float64(ty)
	if RunsNS(d) {
		return hit(fx, fy, fx+1, fy+DoorJamb) || hit(fx, fy+1-DoorJamb, fx+1, fy+1)
	}
	return hit(fx, fy, fx+DoorJamb, fy+1) || hit(fx+1-DoorJamb, fy, fx+1, fy+1)
}

// jambsClear reports whether a player's box at (x, y) keeps off the posts
// of every door it reaches into.
func (w *World) jambsClear(x, y float64) bool {
	r := PlayerRadius
	for ty := int(math.Floor(y - r)); ty <= int(math.Floor(y+r)); ty++ {
		for tx := int(math.Floor(x - r)); tx <= int(math.Floor(x+r)); tx++ {
			if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityDoor && DoorBlocks(e.Face, tx, ty, x, y, r) {
				return false
			}
		}
	}
	return true
}

// doorNudge helps p, walking at a door along its way through but held on a
// post, slide towards its middle, as far as sp: one need not line up to
// the pixel to go through. dx, dy is the step that was refused, can the
// test it was refused by.
func (w *World) doorNudge(p *Player, dx, dy, sp float64, can func(x, y float64) bool) {
	r := PlayerRadius
	sign := func(v float64) float64 { return math.Copysign(1, v) }
	if dy != 0 {
		tx, ty := int(math.Floor(p.X)), int(math.Floor(p.Y+dy+sign(dy)*r))
		if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityDoor && !RunsNS(e.Face) {
			mid := float64(tx) + 0.5
			if nx := p.X + math.Max(-sp, math.Min(sp, mid-p.X)); nx != p.X && can(nx, p.Y) {
				p.X = nx
			}
		}
	}
	if dx != 0 {
		tx, ty := int(math.Floor(p.X+dx+sign(dx)*r)), int(math.Floor(p.Y))
		if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityDoor && RunsNS(e.Face) {
			mid := float64(ty) + 0.5
			if ny := p.Y + math.Max(-sp, math.Min(sp, mid-p.Y)); ny != p.Y && can(p.X, ny) {
				p.Y = ny
			}
		}
	}
}

// bySide is v when refused, else 0: the part of a step that was refused.
func bySide(refused bool, v float64) float64 {
	if refused {
		return v
	}
	return 0
}

// FrontSlack is how far behind the line through its middle one may stand
// and still use a thing from its front, in tiles.
const FrontSlack = 0.25

// UsedFromFront reports whether an entity of kind k is used only from its
// front or its sides: a bed or a tent climbed into from its open side, a
// furnace stoked and an oven opened at the mouth. Taking them down is
// done from anywhere.
func UsedFromFront(k EntityKind) bool {
	return k == EntityBed || k == EntityLeafTent || k == EntityFurnace || k == EntityOven
}

// FrontSide reports whether someone at (px, py) stands before or beside a
// thing on tile (tx, ty) facing d, not behind it. Shared with the client,
// which says so over it.
func FrontSide(d Dir, tx, ty int, px, py float64) bool {
	fx, fy := d.Delta()
	return (px-float64(tx)-0.5)*float64(fx)+(py-float64(ty)-0.5)*float64(fy) > -FrontSlack
}

// fromFront reports whether p may use e from where they stand, and tells
// them to go round when not.
func (w *World) fromFront(p *Player, e *Entity) bool {
	if !UsedFromFront(e.Kind) || FrontSide(e.Face, e.TX, e.TY, p.X, p.Y) {
		return true
	}
	p.notify(NoticeWrongSide, uint8(e.Kind))
	return false
}

// face sets e, just put down, to face d, as its kind can.
func (e *Entity) face(d Dir) { e.Face = FacingOf(e.Kind, e.Var, d) }
