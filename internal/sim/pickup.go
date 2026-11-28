package sim

import "math"

// Picking things up is E's, never Action's (the left button or Space).
//
// Beasts drop what they give right where the fight is, and a click that
// went to a pile instead of the beast was a blow lost. So Action strikes,
// chops, mines, cuts and gathers from plants, and only E picks up: the pile
// under the pointer, else the loose thing on the ground under it (a fallen
// branch, a pebble, a mushroom, a clam, a reed, a flower: see LooseItem),
// else the nearest pile, else the nearest loose thing. A chest, a station or
// a beast under the pointer still comes before the nearest pile: see
// interactKey in aim.go. Aboard a boat or astride a beast, E picks up only
// what is pointed at, and otherwise gets off. Held down, E goes on picking
// up one thing after another and nothing else: the client sends CmdPickUp
// every PickCooldown.
//
// PickUpFrom is the rule, shared with the client, which marks and names
// what E would pick up.

// LooseItem reports whether an entity of kind k is a loose thing lying on
// the ground, picked up by hand with E rather than worked on with Action.
func LooseItem(k EntityKind) bool {
	switch k {
	case EntityTwig, EntityPebble, EntityMushroom, EntityClam, EntityReed, EntityFlower:
		return true
	}
	return false
}

// loose reports whether c is a loose thing there to pick up.
func (c TargetCandidate) loose() bool { return LooseItem(c.Kind) && c.Ready }

// Pickup is what E picks up: a pile (Drop) or a loose thing on the ground
// (Entity). Aimed says it is under the pointer.
type Pickup struct {
	Drop   DropID
	Entity EntityID
	Aimed  bool
}

// PickUpTarget is what E picks up for a player at (x, y), pointing at
// (ax, ay) when aim: the pile under the pointer, else the loose thing under
// it, else the nearest pile, else the nearest loose thing. drops are the
// piles about; at returns the entity on a tile.
func PickUpTarget(x, y float64, aim bool, ax, ay float64, drops []DropCandidate, at func(tx, ty int) (TargetCandidate, bool)) (Pickup, bool) {
	if aim {
		if id, ok := AimedDrop(x, y, ax, ay, drops); ok {
			return Pickup{Drop: id, Aimed: true}, true
		}
		if id, ok := aimedLoose(x, y, ax, ay, at); ok {
			return Pickup{Entity: id, Aimed: true}, true
		}
	}
	if id, ok := NearestDrop(x, y, drops); ok {
		return Pickup{Drop: id}, true
	}
	if id, ok := nearestLoose(x, y, at); ok {
		return Pickup{Entity: id}, true
	}
	return Pickup{}, false
}

// aimedLoose is the loose thing under a pointer at (ax, ay) that a player at
// (x, y) reaches: the one whose own tile the pointer is on, else the one in
// front.
func aimedLoose(x, y, ax, ay float64, at func(tx, ty int) (TargetCandidate, bool)) (EntityID, bool) {
	var best EntityID
	bestRank, found := 0.0, false
	fx, fy := int(math.Floor(ax)), int(math.Floor(ay))
	for ty := fy - 1; ty <= fy+2; ty++ {
		for tx := fx - 1; tx <= fx+1; tx++ {
			c, ok := at(tx, ty)
			if !ok || !c.loose() {
				continue
			}
			hit, ground := entityHit(c.Kind, tx, ty, ax, ay)
			if dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y; !hit || dx*dx+dy*dy > InteractReach*InteractReach {
				continue
			}
			rank := 1000 - float64(ty) + float64(c.ID)*1e-12 // the one in front
			if ground {
				rank = -1000 + float64(c.ID)*1e-12
			}
			if !found || rank < bestRank {
				best, bestRank, found = c.ID, rank, true
			}
		}
	}
	return best, found
}

// nearestLoose is the nearest loose thing whose tile centre is within
// InteractReach of (x, y), the lower id on a tie.
func nearestLoose(x, y float64, at func(tx, ty int) (TargetCandidate, bool)) (EntityID, bool) {
	reach := int(math.Ceil(InteractReach))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	var best EntityID
	bestD, found := InteractReach*InteractReach, false
	for ty := cy - reach; ty <= cy+reach; ty++ {
		for tx := cx - reach; tx <= cx+reach; tx++ {
			c, ok := at(tx, ty)
			if !ok || !c.loose() {
				continue
			}
			dx, dy := float64(tx)+0.5-x, float64(ty)+0.5-y
			if d := dx*dx + dy*dy; d < bestD || (d == bestD && (!found || c.ID < best)) {
				best, bestD, found = c.ID, d, true
			}
		}
	}
	return best, found
}

// pickUpTarget is PickUpTarget for p. Aboard a boat or astride a beast, E
// gets off, so only what is pointed at is picked up from there.
func (w *World) pickUpTarget(p *Player) (Pickup, bool) {
	ax, ay, aim := p.Aim()
	return PickUpFrom(p.X, p.Y, aim, ax, ay, p.boat != 0 || p.mount != 0, w.dropCandidates(p.X, p.Y), w.targetCandidate)
}

// PickUpFrom is PickUpTarget for a player who may be aboard a boat or
// astride a beast (seated): then only what is pointed at.
func PickUpFrom(x, y float64, aim bool, ax, ay float64, seated bool, drops []DropCandidate, at func(tx, ty int) (TargetCandidate, bool)) (Pickup, bool) {
	k, ok := PickUpTarget(x, y, aim, ax, ay, drops, at)
	if !ok || (seated && !k.Aimed) {
		return Pickup{}, false
	}
	return k, true
}

// takePickup picks up k for p, turned to it, and reports whether anything
// went into the bag (not when it is full).
func (w *World) takePickup(p *Player, k Pickup) bool {
	if k.Drop != 0 {
		d := w.dropByID(k.Drop)
		if d == nil {
			return false
		}
		p.Dir = dirToward(d.X-p.X, d.Y-p.Y)
		return w.pickUp(p, d, true)
	}
	e := w.entities[k.Entity]
	if e == nil || !LooseItem(e.Kind) {
		return false
	}
	p.Dir = dirToward(float64(e.TX)+0.5-p.X, float64(e.TY)+0.5-p.Y)
	return w.interact(p, e) // see player.go: one hit takes it
}

// pickUpKey is CmdPickUp, E held down: the next pile or loose thing, if
// there is one in reach, and nothing else: no chest, no beast, no getting
// off, no notice when there is nothing.
func (w *World) pickUpKey(p *Player) {
	if p.Dead || p.Asleep {
		return
	}
	if k, ok := w.pickUpTarget(p); ok && w.takePickup(p, k) {
		p.actedAt = w.tick
	}
}
