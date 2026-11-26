package client

import (
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Which motion (motion.go) a player is in. The server sends state, not
// events, so what a player is doing is read off the snapshots: a deed
// counter that went up (eating, scooping, tossing), a blow begun, a mount
// gained or lost, death; and off the client's own clock: how long they have
// stood still, near what.

// never is a frame long past: "has not happened".
const never = -1 << 30

// moState is what the client keeps of a player to pick their motion.
type moState struct {
	movedAt     int // the last frame they were seen moving
	blowAt      int // when their latest blow began
	blows       int // blows begun: a sword's forehand and backhand take turns
	deed        sim.Deed
	deedItem    sim.ItemKind
	deedAt      int // when the deed began
	sitAt       int // when they began to sit down; never while up
	standAt     int // when they began to get up
	mountAt     int // when they got onto a beast
	dismountAt  int // and off it
	mountKind   sim.CreatureKind
	mountLeft   bool    // the beast faced left
	mountX      float64 // where the rider sat as they got off, from where they stand, art pixels
	mountY      float64
	deadAt      int // when they fell
	deadDir     sim.Dir
	deadX       float32 // where
	deadY       float32
	wasSwimming bool
	wadeAt      int // when they came out of the water or went in: see swimhd.go
}

// Sitting down by a fire comes after standing this long near it, in
// frames; a fighter keeps their stance this long after a blow.
const (
	sitAfter   = 4 * 60
	readyAfter = 150
	fireReach  = 2.6 // tiles from a burning campfire that count as by it
	foeReach   = 6.0 // tiles a beast on the attack puts a fighter on guard from
)

func (g *Game) moveState(id sim.PlayerID) *moState {
	if g.moves == nil {
		g.moves = make(map[sim.PlayerID]*moState)
	}
	st := g.moves[id]
	if st == nil {
		st = &moState{movedAt: g.frame, blowAt: never, deedAt: never, sitAt: never, standAt: never,
			mountAt: never, dismountAt: never, deadAt: never, wadeAt: never}
		g.moves[id] = st
	}
	return st
}

// noteMotions reads off what each player in cur began doing since old. It
// is called before cur replaces the previous snapshot.
func (g *Game) noteMotions(old, cur protocol.Snapshot) {
	kind := func(s protocol.Snapshot, id sim.CreatureID) (protocol.CreatureView, bool) {
		for _, c := range s.Creatures {
			if c.ID == id {
				return c, true
			}
		}
		return protocol.CreatureView{}, false
	}
	for _, p := range cur.Players {
		st := g.moveState(p.ID)
		q, ok := g.prevPlayers[p.ID]
		if !ok {
			continue
		}
		if q.DeedSeq != p.DeedSeq && p.Deed != sim.DeedNone {
			st.deed, st.deedItem, st.deedAt = p.Deed, p.DeedItem, g.frame
		}
		if p.Acting && !q.Acting {
			st.blowAt = g.frame
			st.blows++
		}
		switch {
		case p.Mount != 0 && q.Mount == 0:
			st.mountAt, st.dismountAt = g.frame, never
			if c, ok := kind(cur, p.Mount); ok {
				st.mountKind = c.Kind
			}
		case p.Mount == 0 && q.Mount != 0:
			st.dismountAt, st.mountAt = g.frame, never
			if c, ok := kind(old, q.Mount); ok {
				st.mountKind, st.mountLeft = c.Kind, c.FaceLeft
				back, height := seat(c.Kind)
				side := 1.0
				if c.FaceLeft {
					side = -1
				}
				st.mountX = (float64(c.X)-float64(p.X))*TileW - back*side
				st.mountY = (float64(c.Y)-float64(p.Y))*TileH - height + riderLegsTall
			}
		}
		switch {
		case p.Dead && !q.Dead:
			st.deadAt, st.deadDir, st.deadX, st.deadY = g.frame, q.Dir&3, q.X, q.Y
		case !p.Dead:
			st.deadAt = never
		}
		if p.Swimming != q.Swimming {
			st.wadeAt = g.frame
		}
	}
}

// deedMotion is the motion of deed d done with item k.
func deedMotion(d sim.Deed, k sim.ItemKind) motionID {
	switch d {
	case sim.DeedScoop:
		return moScoop
	case sim.DeedToss:
		return moThrow
	}
	return eatMotion(k)
}

// eatMotion is how food k is eaten: soup from its bowl, water from the
// flask, berries and nuts by the handful, the rest bitten.
func eatMotion(k sim.ItemKind) motionID {
	switch k {
	case sim.ItemStew, sim.ItemHotSoup, sim.ItemDinoStew, sim.ItemHerbTea, sim.ItemJuice, sim.ItemHerbMedicine:
		return moEatBowl
	case sim.ItemCleanWater, sim.ItemDirtyWater, sim.ItemPitcherWater:
		return moDrink
	case sim.ItemBerry, sim.ItemGinkgoNut, sim.ItemLotusSeed, sim.ItemNightshade, sim.ItemMushroom,
		sim.ItemPoisonMushroom, sim.ItemFig, sim.ItemCorn, sim.ItemRice:
		return moEatHand
	}
	return moEatBite
}

// motionOf is the motion p is in while not striding or striking, and its
// frame, if any: getting up, a deed, sitting by a fire, a fighter's stance,
// or breathing. moving is whether they are on the move this frame.
func (g *Game) motionOf(p protocol.PlayerView, moving bool) (motionID, int, bool) {
	st := g.moveState(p.ID)
	if moving {
		st.movedAt = g.frame
		if st.sitAt != never {
			st.sitAt, st.standAt = never, g.frame // up at once: they are off
		}
	}
	if st.deedAt != never {
		m := deedMotion(st.deed, st.deedItem)
		age := float64(g.frame - st.deedAt)
		if moving || age >= m.span() || age < 0 {
			st.deedAt = never // walking off cuts it short
		} else {
			if st.sitAt != never {
				st.sitAt, st.standAt = never, never
			}
			return m, m.frameAt(age), true
		}
	}
	// Getting up from the ground: at twice the speed of sitting down, the
	// feet already going.
	if age := float64(g.frame - st.standAt); age >= 0 && age < moStandUp.span()/2 {
		return moStandUp, moStandUp.frameAt(age * 2), true
	}
	if moving {
		return moNone, 0, false
	}
	busy := p.Acting || g.frame-st.blowAt < sitAfter || p.Mount != 0
	if st.sitAt == never && !busy && g.frame-st.movedAt > sitAfter && g.besideFire(p) {
		st.sitAt = g.frame
	}
	if st.sitAt != never {
		if busy || !g.besideFire(p) {
			st.sitAt, st.standAt = never, g.frame
			return moStandUp, 0, true
		}
		age := float64(g.frame - st.sitAt)
		if age < moSitDown.span() {
			return moSitDown, moSitDown.frameAt(age), true
		}
		return moSit, moSit.frameAt(age - moSitDown.span()), true
	}
	if g.onGuard(p, st) {
		return moReady, moReady.frameAt(float64(g.frame + int(p.ID)*31)), true
	}
	return moIdle, moIdle.frameAt(float64(g.frame + int(p.ID)*53)), true
}

// besideFire reports whether p stands by a burning campfire.
func (g *Game) besideFire(p protocol.PlayerView) bool {
	for _, e := range g.curr.Entities {
		if e.Kind == sim.EntityCampfire && e.Fuel > 0 &&
			math.Hypot(float64(e.TX)+0.5-float64(p.X), float64(e.TY)+0.5-float64(p.Y)) <= fireReach {
			return true
		}
	}
	return false
}

// fights reports whether k is something to fight with: a weapon, or a
// tool that swings.
func fights(k sim.ItemKind) bool {
	if c, _ := k.ToolKind(); c == sim.ToolAxe || c == sim.ToolPickaxe {
		return true
	}
	if _, ok := sim.SwordTier(k); ok {
		return true
	}
	return k == sim.ItemSpear || k == sim.ItemIvorySpear || k == sim.ItemBattleAxe
}

// onGuard reports whether p, standing, holds a weapon up: a beast is on
// the attack near them, or they struck a moment ago.
func (g *Game) onGuard(p protocol.PlayerView, st *moState) bool {
	if !fights(p.Held) {
		return false
	}
	if g.frame-st.blowAt < readyAfter {
		return true
	}
	for _, c := range g.curr.Creatures {
		if c.Tamed || c.Ridden {
			continue
		}
		switch c.State {
		case sim.CreatureChase, sim.CreatureStrike, sim.CreatureRoar, sim.CreatureClaw, sim.CreatureSpit, sim.CreatureSpecial:
			if math.Hypot(float64(c.X-p.X), float64(c.Y-p.Y)) <= foeReach {
				return true
			}
		}
	}
	return false
}

// gathering is the motion of a reach of the hand at something that is not a
// beast (picking, gathering, stoking): a crouch to the ground, or up into a
// fruit tree; on its own clock from the blow, so a held button repeats it.
func (g *Game) gathering(p protocol.PlayerView) (motionID, int, bool) {
	w := g.swingWhat[p.ID]
	if w.kind != swingHand || w.beast || p.Mount != 0 {
		return moNone, 0, false
	}
	at, ok := g.swingAt[p.ID]
	age := float64(g.since(at))
	m := moGather
	if w.high {
		m = moReachUp
	}
	if !ok || age < 0 || age >= m.span() {
		return moNone, 0, false
	}
	return m, m.frameAt(age), true
}

// backhand reports whether p's sword blow now is a backhand: forehand and
// backhand take turns.
func (g *Game) backhand(p protocol.PlayerView) bool {
	return g.moveState(p.ID).blows%2 == 0
}
