package client

import (
	"image/color"
	"math"

	"survivalgame/internal/protocol"
	"survivalgame/internal/sim"
)

// Working swings. When a player hits something, what shows in their hand is
// what the job takes, chosen as the server chooses: an axe for trees, walls
// and cacti, a pickaxe for rock, coal and ore, the hoe when one is held, and
// bare hands for picking, gathering and stoking. Each blow is timed from the
// snapshot that first shows it (see noteSwings), the same moment its sound
// plays: the tool comes down from overhead and lands two frames later, so
// the eye and the ear agree.

type swingKind uint8

const (
	swingNone   swingKind = iota // not working: the held item as usual
	swingHand                    // picking, gathering, stoking: a reach of the hand
	swingAxe                     // chopping
	swingPick                    // mining
	swingHoe                     // hoeing
	swingThrust                  // a spear driven at an animal
	swingCleave                  // the war axe brought down on an animal; see battleaxe.go
)

// swingInfo is what a player's latest blow was.
type swingInfo struct {
	kind  swingKind
	tier  sim.Tier // of the tool swung
	beast bool     // at a creature: by hand, a punch (see punch.go)
	cross bool     // a punch thrown with the other hand than the last
	high  bool     // by hand at a tree's ripe fruit: a reach up, not a crouch
}

// A chop, in frames at 60 a second: overhead, down onto the target, a small
// bounce off it, and back to rest well before the working pose ends.
const (
	chopStrike = 2.0
	chopBounce = 4.0
	chopReturn = 10.0
	chopRaised = -0.9 // radians from upright, back over the shoulder
	chopImpact = 1.75 // level with the target in front
	chopRest   = 0.6
)

// chopAngle is the tool's angle age frames into a blow.
func chopAngle(age float64) float64 {
	switch {
	case age < chopStrike: // speeding up on the way down
		u := age / chopStrike
		return chopRaised + (chopImpact-chopRaised)*u*u
	case age < chopStrike+chopBounce:
		u := (age - chopStrike) / chopBounce
		return chopImpact - 0.25*math.Sin(u*math.Pi)
	case age < chopStrike+chopBounce+chopReturn:
		u := (age - chopStrike - chopBounce) / chopReturn
		return chopImpact + (chopRest-chopImpact)*smooth(u)
	}
	return chopRest
}

// reach is how far the hand or a spear is pushed out age frames into a
// blow: out fast, back slowly.
func reach(age float64) float64 {
	switch {
	case age < 3:
		return age / 3
	case age < 12:
		return 1 - (age-3)/9
	}
	return 0
}

// classifySwings works out, for each player whose blow begins in cur, what
// it was at: the target the server would have picked in old, the world just
// before the hit, and so the tool it took.
func (g *Game) classifySwings(old, cur protocol.Snapshot) {
	if g.swingWhat == nil {
		g.swingWhat = make(map[sim.PlayerID]swingInfo)
	}
	first := len(old.Players) == 0
	if first {
		old = cur // the first snapshot: the world as it is is the best guess
	}
	was := make(map[sim.PlayerID]bool, len(old.Players))
	for _, p := range old.Players {
		was[p.ID] = p.Acting
	}
	oldAt := make(map[[2]int]protocol.EntityView, len(old.Entities))
	for _, e := range old.Entities {
		oldAt[[2]int{int(e.TX), int(e.TY)}] = e
	}
	for _, p := range cur.Players {
		if p.Acting && (first || !was[p.ID]) {
			w := g.whatSwing(old, oldAt, p)
			if last := g.swingWhat[p.ID]; w.beast && last.beast {
				w.cross = !last.cross // jab, cross, jab
			}
			g.swingWhat[p.ID] = w
		}
	}
}

func (g *Game) whatSwing(old protocol.Snapshot, oldAt map[[2]int]protocol.EntityView, p protocol.PlayerView) swingInfo {
	if isHoe(p.Held) {
		return swingInfo{kind: swingHoe, tier: toolTier(p.Held)}
	}
	me := p.ID == g.myID
	hasFuel := !me || old.Self.Inv.HasFuel()
	px, py := float64(p.X), float64(p.Y)
	tg, ok := sim.ResolveTarget(px, py, p.Aim, float64(p.AimX), float64(p.AimY), hasFuel, func(tx, ty int) (sim.TargetCandidate, bool) {
		e, ok := oldAt[[2]int{tx, ty}]
		return sim.TargetCandidate{ID: e.ID, Kind: e.Kind, Ready: e.Ready, Var: e.Var}, ok
	}, creatureCandidates(old.Creatures, px, py), sim.ReachBonusFor(p.Held)+sim.RideReach(p.Mount != 0))
	switch {
	case !ok:
		return swingInfo{kind: swingHand}
	case tg.Creature != 0:
		if p.Held == sim.ItemSpear || p.Held == sim.ItemIvorySpear {
			return swingInfo{kind: swingThrust}
		}
		if p.Held == sim.ItemBattleAxe {
			return swingInfo{kind: swingCleave}
		}
		return swingInfo{kind: swingHand, beast: true}
	}
	// The best tool of the class carried: known for me, guessed for others
	// (their axe's tier is sent; a pickaxe is shown as the stone one).
	tool := func(c sim.ToolClass, k swingKind, seen sim.Tier) swingInfo {
		if me {
			if t, ok := old.Self.Inv.BestTool(c); ok {
				return swingInfo{kind: k, tier: t}
			}
		}
		return swingInfo{kind: k, tier: seen}
	}
	for _, e := range old.Entities {
		if e.ID != tg.Entity {
			continue
		}
		switch {
		case sim.IsFruitTree(e.Kind) && e.Ready:
			return swingInfo{kind: swingHand, high: true}
		case e.Kind == sim.EntityTree || e.Kind == sim.EntityWall || e.Kind == sim.EntityCactus,
			sim.IsFruitTree(e.Kind) && !e.Ready: // a bare fruit tree is felled
			return tool(sim.ToolAxe, swingAxe, p.AxeTier)
		case e.Kind == sim.EntityBoulder || e.Kind == sim.EntityStoneWall || e.Kind == sim.EntityCoal || isOre(e.Kind):
			return tool(sim.ToolPickaxe, swingPick, sim.TierStone)
		case sim.IsBedLike(e.Kind) && sim.PhaseOf(old.Tick) == sim.PhaseDay:
			if c, _ := p.Held.ToolKind(); c == sim.ToolAxe { // taking it down; see sim/tent.go
				return tool(sim.ToolAxe, swingAxe, p.AxeTier)
			}
		}
	}
	return swingInfo{kind: swingHand}
}

// drawSwing draws what p swings in the middle of a blow.
func (g *Game) drawSwing(dst drawTarget, p protocol.PlayerView, d drawable, bob, step float64) {
	w := g.swingWhat[p.ID]
	age := float64(g.frame - g.swingAt[p.ID])
	side := 1.0
	if p.Dir == sim.DirLeft {
		side = -1
	}
	tier := func(t sim.Tier) sim.Tier { return min(t, sim.NumTiers-1) }
	tool := func(s sprite, c sim.ToolClass) { // gripped at the hand, swung about the grip
		gx, gy := d.sx+6*side, d.sy-7-bob
		if limbGrip.on { // the jointed figure's hand; see limbdraw.go
			gx, gy = limbGrip.x, limbGrip.y
		}
		a := chopAngle(age)
		if hd := heldDefOf(sim.ToolItem(c, tier(w.tier))); hd != nil { // see heldhd.go
			// Coming down fast, the head leaves a trail of where it has
			// just been, fading once it has landed.
			if age < chopStrike+chopBounce+4 {
				fade := 1.0
				if age > chopStrike {
					fade = 1 - (age-chopStrike)/(chopBounce+4)
				}
				lead := min(a, chopImpact) // the bounce does not draw it on
				swingArc(dst, gx, gy, side, lead, chopRaised, 1, 1.5, 7.5, 15, color.RGBA{0xe8, 0xe4, 0xd8, 0xff}, 0.55*fade)
			}
			drawHeldDef(dst, hd, gx, gy, a, side, 1)
			return
		}
		drawAt(dst, s, gx, gy, side, 1, a*side, nil)
	}
	switch w.kind {
	case swingAxe:
		tool(g.art.ore.axes[tier(w.tier)], sim.ToolAxe)
	case swingPick:
		tool(g.art.ore.picks[tier(w.tier)], sim.ToolPickaxe)
	case swingHoe:
		tool(g.art.ore.hoes[tier(w.tier)], sim.ToolHoe)
	case swingCleave:
		g.drawCleave(dst, p, d, bob)
	default: // a hand or a spear pushed out towards the target, and back
		dx, dy := p.Dir.Delta()
		out := reach(age)
		if w.kind == swingThrust {
			out *= 2
		}
		g.drawHeld(dst, p.Held, p.Dir&3, d.sx+math.Round(float64(dx)*3*out), d.sy+math.Round(float64(dy)*2*out), bob, step)
	}
}
