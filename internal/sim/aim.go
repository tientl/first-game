package sim

import "math"

// Aiming with the mouse.
//
// The client sends where the mouse points (Input.Aim), in world tiles. The
// player faces the way they walk, and turns to what they work on; they face
// the pointer only while aiming: holding the button down with something
// that is aimed in hand (see AimedItem: a bow, a throw). Riding and sailing
// steer as before.
// Action works on what is under the pointer when the player reaches it, by
// the usual reach rules; when nothing actable is under it, or it is out of
// reach, Action falls back to the nearest target, preferring those towards
// the pointer. Without an aim (a keyboard player) everything is as it was.
//
// Building: a structure from a recipe, or a torch, a head on a post or a
// seed from the hand, may be put on a tile the player chooses (Command.At),
// which must be within PlaceReach and free.
//
// E (CmdInteract) picks up a pile, opens a chest, or gets on or off a beast
// or a boat; right-click with food on a beast (CmdFeed) feeds it.
//
// The rules here are shared with the client, which highlights the same
// target the server will work on.

// Aim tuning.
const (
	PlaceReach = InteractReach + 1 // tiles from a player's centre to the centre of a tile they build on
	aimPad     = 0.08              // a pointer this close outside a box still counts as on it
	dropAimR   = 0.45              // a pointer this close to a pile points at it
)

// Aim returns where p points, and whether they do.
func (p *Player) Aim() (x, y float64, ok bool) {
	return float64(p.input.AimX), float64(p.input.AimY), p.input.Aim
}

// aimBox is the box, in tiles from the tile's bottom-left corner going up,
// in which a pointer is on an entity of kind k: as wide as its tile (wider
// for the giant tree) and as tall as it stands.
func aimBox(k EntityKind) (halfW, height float64) {
	switch k {
	case EntityGiantTree:
		return 1.6, 5.5
	case EntityLeafTent:
		return 0.95, 2.4 // two tiles wide and a little taller than a player: see tent.go
	case EntityTree, EntityFigTree, EntityGinkgoTree, EntityPalm, EntityBananaTree:
		return 0.6, 2.6
	case EntityBoulder, EntityBerryBush, EntityCactus, EntityWall, EntityStoneWall, EntityDoor,
		EntityFurnace, EntityBed, EntityChest, EntitySpikes, EntityTrophy, EntityTorchStand,
		EntityManEater, EntityTreasure, EntityGlowPlant, EntityReed, EntityCampfire, EntityGrave,
		EntityOven, EntityDryRack, EntityFlora:
		return 0.5, 1.6
	}
	return 0.5, 1
}

// entityHit reports whether a pointer at (ax, ay) is on an entity of kind
// k on tile (tx, ty), and whether it is on the tile itself.
func entityHit(k EntityKind, tx, ty int, ax, ay float64) (hit, ground bool) {
	halfW, h := aimBox(k)
	cx, bottom := float64(tx)+0.5, float64(ty)+1
	if math.Abs(ax-cx) > halfW+aimPad || ay > bottom+aimPad || ay < bottom-h-aimPad {
		return false, false
	}
	return true, ay >= float64(ty)-aimPad && math.Abs(ax-cx) <= 0.5+aimPad
}

// creatureHit reports whether a pointer at (ax, ay) is on a creature of
// kind k standing at (x, y), and how near its middle (0 at the middle, 1 at
// the edge).
func creatureHit(k CreatureKind, x, y, ax, ay float64) (bool, float64) {
	r := k.def().radius
	rx, ry := r+0.45, r+0.6
	dx, dy := (ax-x)/rx, (ay-(y-r*0.8-0.25))/ry
	d := dx*dx + dy*dy
	return d <= 1, d
}

// AimedTarget is the target under a pointer at (ax, ay) for a player at
// (x, y). ok says something actable is under it; inReach says the player
// also reaches it, by the same rules as PickTarget. When several things are
// under the pointer, one in reach wins, then a creature over an entity, then
// the entity whose own tile the pointer is on, then the one in front.
func AimedTarget(x, y, ax, ay float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool), creatures []CreatureCandidate, reachBonus float64) (t Target, inReach, ok bool) {
	type pick struct {
		t        Target
		in       bool
		rank     float64 // lower is better among the same in-reach-ness
		creature bool
	}
	var best pick
	found := false
	better := func(p pick) bool {
		switch {
		case !found:
			return true
		case p.in != best.in:
			return p.in
		case p.creature != best.creature:
			return p.creature
		}
		return p.rank < best.rank
	}
	for _, c := range creatures {
		if c.Kind == CreatureNone || c.Kind >= NumCreatureKinds {
			continue
		}
		hit, d := creatureHit(c.Kind, c.X, c.Y, ax, ay)
		if !hit {
			continue
		}
		r := c.Kind.Reach() + reachBonus
		dx, dy := c.X-x, c.Y-y
		p := pick{t: Target{Creature: c.ID}, in: dx*dx+dy*dy <= r*r, rank: d + float64(c.ID)*1e-9, creature: true}
		if better(p) {
			best, found = p, true
		}
	}
	fx, fy := int(math.Floor(ax)), int(math.Floor(ay))
	for ty := fy - 1; ty <= fy+5; ty++ {
		for tx := fx - 2; tx <= fx+2; tx++ {
			c, ok := at(tx, ty)
			if !ok || !c.targetable(hasWood) {
				continue
			}
			hit, ground := entityHit(c.Kind, tx, ty, ax, ay)
			if !hit {
				continue
			}
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			rank := 1000 - float64(ty) + float64(c.ID)*1e-12 // the one in front
			if ground {
				rank = -1000
			}
			p := pick{t: Target{Entity: c.ID}, in: dx*dx+dy*dy <= InteractReach*InteractReach, rank: rank}
			if better(p) {
				best, found = p, true
			}
		}
	}
	return best.t, best.in, found
}

// ResolveTarget is the whole targeting rule, shared by the server and the
// client: with an aim, the target under the pointer if it is in reach, else
// the nearest target with those towards the pointer preferred; without one,
// PickTarget.
func ResolveTarget(x, y float64, aim bool, ax, ay float64, hasWood bool, at func(tx, ty int) (TargetCandidate, bool), creatures []CreatureCandidate, reachBonus float64) (Target, bool) {
	if !aim {
		return PickTarget(x, y, hasWood, at, creatures, reachBonus)
	}
	if t, in, ok := AimedTarget(x, y, ax, ay, hasWood, at, creatures, reachBonus); ok && in {
		return t, true
	}
	return pickTarget(x, y, hasWood, at, creatures, reachBonus, towards(ax-x, ay-y))
}

// towards weighs a candidate's squared distance by its direction from the
// player: those towards (dx, dy) count as they are, those to the side half
// as far again, those behind twice as far.
func towards(dx, dy float64) func(cx, cy, d2 float64) float64 {
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		return nil
	}
	ux, uy := dx/l, dy/l
	return func(cx, cy, d2 float64) float64 {
		cl := math.Hypot(cx, cy)
		if cl < 1e-6 {
			return d2
		}
		cos := (cx*ux + cy*uy) / cl
		return d2 * (1.5 - 0.5*cos)
	}
}

// AimedDrop is the pile under a pointer at (ax, ay) that a player at (x, y)
// picks up: the nearest to the pointer within dropAimR of it and DropReach
// of the player.
func AimedDrop(x, y, ax, ay float64, drops []DropCandidate) (DropID, bool) {
	var best DropID
	bestD := dropAimR * dropAimR
	found := false
	for _, d := range drops {
		if (d.X-x)*(d.X-x)+(d.Y-y)*(d.Y-y) > DropReach*DropReach {
			continue
		}
		dd := (d.X-ax)*(d.X-ax) + (d.Y-ay)*(d.Y-ay)
		if dd < bestD || (dd == bestD && found && d.ID < best) {
			best, bestD, found = d.ID, dd, true
		}
	}
	return best, found
}

// PlacedBy is what an item held in the hand becomes when it is set down on
// a tile: a torch stands in the ground, a head goes on a post, a seed is
// planted. soft says it needs soft ground (a seed). ok is false for
// anything else, a boat and dung included: they are not built on a tile.
func PlacedBy(held ItemKind) (k EntityKind, soft, ok bool) {
	switch {
	case held == ItemTorch:
		return EntityTorchStand, false, true
	case held == ItemLeafTent:
		return EntityLeafTent, false, true // see tent.go
	case IsSeed(held):
		return EntitySapling, true, true
	}
	if _, isHead := HeadOf(held); isHead {
		return EntityTrophy, false, true
	}
	return EntityNone, false, false
}

// BlocksPlayers reports whether a structure of kind k stops players: it
// may not be built where someone stands.
func BlocksPlayers(k EntityKind) bool { return blocksPlayers(k) }

// BuildSpot is the check on tile (tx, ty), chosen to build kind k on by a
// player at (px, py), shared by the server and the client's preview: within
// PlaceReach, not the builder's own tile, walkable (soft, for a seed) and
// free. taken reports a tile already built on; standers are where players
// stand, which a structure that blocks them may not be put on. why is the
// notice when it fails.
func BuildSpot(px, py float64, tx, ty int, k EntityKind, soft bool, tileAt func(tx, ty int) Tile, taken func(tx, ty int) bool, standers [][2]float64) (ok bool, why NoticeKind) {
	return BuildSpotWithin(PlaceReach, px, py, tx, ty, k, soft, tileAt, taken, standers)
}

// BuildSpotWithin is BuildSpot with reach in place of PlaceReach: DevReach
// in free build (see devbuild.go).
func BuildSpotWithin(reach, px, py float64, tx, ty int, k EntityKind, soft bool, tileAt func(tx, ty int) Tile, taken func(tx, ty int) bool, standers [][2]float64) (ok bool, why NoticeKind) {
	if dx, dy := float64(tx)+0.5-px, float64(ty)+0.5-py; dx*dx+dy*dy > reach*reach {
		return false, NoticeTooFar
	}
	t := tileAt(tx, ty)
	switch {
	case tx == int(math.Floor(px)) && ty == int(math.Floor(py)):
		return false, NoticeNoSpace
	case !t.Walkable() || (soft && !t.Soft()) || taken(tx, ty):
		return false, NoticeNoSpace
	case k == EntityLeafTent && !TentGround(t):
		return false, NoticeHardGround // see tent.go
	}
	if blocksPlayers(k) {
		for _, s := range standers {
			r := PlayerRadius
			if s[0]+r > float64(tx) && s[0]-r < float64(tx+1) && s[1]+r > float64(ty) && s[1]-r < float64(ty+1) {
				return false, NoticeNoSpace
			}
		}
	}
	return true, NoticeNone
}

// buildSpot is BuildSpot in w for p; on failure p is told why.
func (w *World) buildSpot(p *Player, tx, ty int, k EntityKind, soft bool) bool {
	ok, why := BuildSpotWithin(w.placeReach(p), p.X, p.Y, tx, ty, k, soft, w.TileAt, func(tx, ty int) bool { return w.entityAt(tx, ty) != nil }, w.standers())
	if ok && !w.canPlace(tx, ty) {
		ok, why = false, NoticeNoSpace
	}
	if !ok {
		p.notify(why, 0)
		return false
	}
	p.Dir = dirToward(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y)
	return true
}

// faceAim turns p to where they point, unless riding, sailing or fishing:
// those keep their own facing.
func (w *World) faceAim(p *Player) {
	ax, ay, ok := p.Aim()
	if !ok || !p.input.Action || !AimedItem(p.Held()) || p.mount != 0 || p.boat != 0 || p.line.state != CastNone {
		return
	}
	if dx, dy := ax-p.X, ay-p.Y; dx*dx+dy*dy > 0.04 {
		p.Dir = dirToward(dx, dy)
	}
}

// aimedItems are the things used by aiming them: held with the button down,
// the player turns to the pointer. Bows and thrown weapons go here when they
// come (see PLAN.md).
var aimedItems = map[ItemKind]bool{}

// AimedItem reports whether k is used by aiming it.
func AimedItem(k ItemKind) bool { return aimedItems[k] }

// targetFor returns what p's Action works on, by ResolveTarget.
func (w *World) targetFor(p *Player) (*Entity, *Creature) {
	ax, ay, aim := p.Aim()
	t, ok := ResolveTarget(p.X, p.Y, aim, ax, ay, p.Inv.HasFuel(), w.targetCandidate, w.creatureCandidates(p.X, p.Y), p.reachBonus())
	switch {
	case !ok:
		return nil, nil
	case t.Creature != 0:
		return nil, w.creatureByID(t.Creature)
	default:
		return w.entities[t.Entity], nil
	}
}

// aimedTarget is what is under p's pointer and in reach, if anything.
func (w *World) aimedTarget(p *Player) (*Entity, *Creature, bool) {
	ax, ay, aim := p.Aim()
	if !aim {
		return nil, nil, false
	}
	t, in, ok := AimedTarget(p.X, p.Y, ax, ay, p.Inv.HasFuel(), w.targetCandidate, w.creatureCandidates(p.X, p.Y), p.reachBonus())
	switch {
	case !ok || !in:
		return nil, nil, false
	case t.Creature != 0:
		c := w.creatureByID(t.Creature)
		return nil, c, c != nil
	default:
		e := w.entities[t.Entity]
		return e, nil, e != nil
	}
}

func (w *World) targetCandidate(tx, ty int) (TargetCandidate, bool) {
	e := w.entityAt(tx, ty)
	if e == nil {
		return TargetCandidate{}, false
	}
	return TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready, Var: e.Var}, true
}

// dropByID returns the pile with id, or nil.
func (w *World) dropByID(id DropID) *Drop {
	for _, d := range w.drops {
		if d.ID == id && !d.gone {
			return d
		}
	}
	return nil
}

// pickAimed picks up the pile under p's pointer, if there is one in reach.
func (w *World) pickAimed(p *Player) (acted, found bool) {
	ax, ay, aim := p.Aim()
	if !aim {
		return false, false
	}
	id, ok := AimedDrop(p.X, p.Y, ax, ay, w.dropCandidates(p.X, p.Y))
	d := w.dropByID(id)
	if !ok || d == nil {
		return false, false
	}
	p.Dir = dirToward(d.X-p.X, d.Y-p.Y)
	return w.pickUp(p, d, true), true
}

// act is one Action by p: on what is pointed at, else as before a pile on
// the ground, then the soil in front for a held hoe, then the nearest
// target. found is false when there was nothing to work on; cd is the wait
// before the next.
func (w *World) act(p *Player) (acted, found bool, cd int) {
	cd = ActionCooldownTicks
	if a, ok := w.pickAimed(p); ok {
		return a, true, PickCooldown
	}
	e, c, aimed := w.aimedTarget(p)
	if !aimed {
		if a, ok := w.pickNearest(p); ok { // a pile on the ground comes first; see drops.go
			return a, true, PickCooldown
		}
		if w.dig(p) { // a held hoe tills the free soil in front
			return true, true, cd
		}
		e, c = w.targetFor(p)
	}
	switch {
	case c != nil:
		p.Dir = dirToward(c.X-p.X, c.Y-p.Y)
		acted = w.catch(p, c)
		if !c.Kind.def().onLine {
			cd = SwingTicks(p.Held()) // see combat.go
		}
	case e != nil:
		p.Dir = dirToward(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y)
		acted = w.interact(p, e)
	default:
		return false, false, cd
	}
	return acted, true, cd
}

// interactKey is E: get off the beast or out of the boat p is on; else pick
// up the pile pointed at, use the chest, bed or body pointed at, or get on
// the beast pointed at; else the nearest pile, then the nearest chest, then
// the nearest boat or beast to get on.
func (w *World) interactKey(p *Player) {
	if p.Dead || p.Asleep {
		return
	}
	if p.boat != 0 || p.mount != 0 {
		if !w.board(p) {
			w.ride(p)
		}
		return
	}
	did := func(acted bool) {
		if acted {
			p.actedAt = w.tick
		}
	}
	if a, ok := w.pickAimed(p); ok {
		did(a)
		return
	}
	usable := func(e *Entity, aimed bool) bool {
		switch e.Kind {
		case EntityChest, EntityGrave, EntityTreasure:
			return e.Kind != EntityTreasure || w.treasureLeft(e)
		case EntityBed, EntityLeafTent:
			return aimed // a bed only when pointed at: E by the bed is mostly for something else
		}
		return isStation(e) // see cooking.go
	}
	ax, ay, aim := p.Aim()
	if aim {
		t, in, ok := AimedTarget(p.X, p.Y, ax, ay, true, w.targetCandidate, w.creatureCandidates(p.X, p.Y), p.reachBonus())
		if ok && in && t.Entity != 0 {
			if e := w.entities[t.Entity]; e != nil && usable(e, true) {
				p.Dir = dirToward(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y)
				did(w.interact(p, e))
				return
			}
		}
		if ok && t.Creature != 0 {
			if c := w.creatureByID(t.Creature); c != nil && c.ownerName == p.Name && c.Kind.Rideable() {
				w.ride(p) // the nearest of p's mounts: the one pointed at, near enough
				return
			}
		}
	}
	if a, ok := w.pickNearest(p); ok {
		did(a)
		return
	}
	var near *Entity
	bestD := InteractReach * InteractReach
	cx, cy := int(math.Floor(p.X)), int(math.Floor(p.Y))
	for ty := cy - 2; ty <= cy+2; ty++ {
		for tx := cx - 2; tx <= cx+2; tx++ {
			e := w.entityAt(tx, ty)
			if e == nil || !usable(e, false) {
				continue
			}
			dx, dy := float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y
			if d := dx*dx + dy*dy; d <= bestD {
				near, bestD = e, d
			}
		}
	}
	if near != nil {
		p.Dir = dirToward(float64(near.TX)+0.5-p.X, float64(near.TY)+0.5-p.Y)
		did(w.interact(p, near))
		return
	}
	if w.board(p) {
		return
	}
	if w.hasMountNear(p) {
		w.ride(p)
		return
	}
	p.notify(NoticeNothingNear, 0)
}

// hasMountNear reports whether any of p's own beasts is close enough for
// the ride command to find it (or say why it cannot be ridden).
func (w *World) hasMountNear(p *Player) bool {
	for _, c := range w.fauna.creatures {
		if !c.gone && c.ownerName == p.Name && c.rider == 0 && math.Hypot(c.X-p.X, c.Y-p.Y) <= MountReach {
			return true
		}
	}
	return false
}

// feedKey is a right click with food on a beast: p gives the held food to
// the beast pointed at if p reaches it, else to the nearest in reach that
// eats it. It never hits.
func (w *World) feedKey(p *Player) {
	if p.Dead || p.Asleep || w.Swimming(p) || p.actionCD > 0 {
		return
	}
	held := p.Held()
	if !held.Valid() {
		return
	}
	var c *Creature
	if _, ac, ok := w.aimedTarget(p); ok && ac != nil && ac.Kind.TameFood() == held {
		c = ac
	}
	if c == nil {
		bestD := math.Inf(1)
		for _, cand := range w.creatureCandidates(p.X, p.Y) {
			if cand.Kind.TameFood() != held {
				continue
			}
			r := cand.Kind.Reach() + p.reachBonus()
			if d := math.Hypot(cand.X-p.X, cand.Y-p.Y); d <= r && d < bestD {
				c, bestD = w.creatureByID(cand.ID), d
			}
		}
	}
	if c == nil || (c.Tamed() && c.HP >= c.Kind.MaxHP()) {
		return // a pet that is not hurt is told to stay or come by a left click, not fed
	}
	if handled, acted := w.petAction(p, c); handled && acted {
		p.Dir = dirToward(c.X-p.X, c.Y-p.Y)
		p.actedAt = w.tick
		p.actionCD = ActionCooldownTicks
	}
}
