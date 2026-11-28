package sim

import "math"

// Titanoboa's fight in its temple (temple.go), on the creatures' moves
// (skills.go): every move wound up in a posture the client draws, most with
// a red zone on the ground, and a moment open to a counter after it. The
// fight has three stages by its health:
//
//   - 100 to 60%: the strike (a lunge along a line), the sweep of its tail
//     (a wide arc round it), the coils (it seizes whoever is close and
//     squeezes; Action pressed again and again breaks free) and the venom
//     (it spits an arc of puddles that slow whoever steps in them). A
//     fallen column between it and a player spares them the sweep and the
//     coils.
//   - 60 to 25%: it also dives into its black pool and bursts out at the
//     edge nearest its foe, and rears to slam its coils down round it.
//   - under 25%, enraged: its wind-ups are a quarter quicker (never under
//     0.4 s), and the pool floods the courtyard: everyone wades.
//
// In an older world (its hollow, lair.go) it fights as it always did:
// bossfight.go.

// Titanoboa's stages: where each begins, as a share of its health.
const (
	TitanStage2 = 0.6
	TitanStage3 = 0.25
)

// Titanoboa's moves, in the order they are added.
var (
	titanStrike, titanTail, titanCoil, titanVenom, titanDive, titanSlam SkillID
)

func init() {
	ids := addSkills(
		SkillDef{Kind: CreatureTitanoboa, Name: "strike", Label: "Lao cắn", Class: SkillHeavy,
			Windup: 18, Active: 2, Recover: 16, Shape: ZoneLine, Reach: 6, Width: 0.8, Zone: true,
			Damage: 36, Knock: 1.4, Cooldown: 50, MaxD: 5.6, Lock: 5, Weight: 7},
		SkillDef{Kind: CreatureTitanoboa, Name: "tail", Label: "Quét đuôi", Class: SkillHeavy,
			Windup: 18, Active: 3, Recover: 18, Shape: ZoneCone, Reach: 5.4, Width: 1.3, Zone: true, Cover: true,
			Damage: 30, Knock: 2.4, Cooldown: 110, MaxD: 5, Lock: 6, Weight: 5},
		SkillDef{Kind: CreatureTitanoboa, Name: "coil", Label: "Siết", Class: SkillSpecial,
			Windup: 26, Active: 4, Recover: 24, Shape: ZoneCircle, Reach: 2.7, Zone: true, Cover: true,
			Damage: 14, Fx: FxPin, PinHold: 4 * TicksPerSecond, PinEvery: TicksPerSecond, PinBite: 12, Mash: 8,
			Cooldown: 220, MaxD: 2.6, Lock: 8, Weight: 6},
		SkillDef{Kind: CreatureTitanoboa, Name: "venom", Label: "Phun độc", Class: SkillHeavy,
			Windup: 20, Active: 2, Recover: 16, Shape: ZoneCircle, Reach: 1.15, Anchor: AtTarget, Zone: true,
			Puddles: 5, Spread: 0.8, Damage: 8, Slow: 3 * TicksPerSecond,
			Cooldown: 150, MinD: 3.5, MaxD: 9, Lock: 6, Weight: 5},
		SkillDef{Kind: CreatureTitanoboa, Name: "dive", Label: "Lặn trồi", Class: SkillSpecial,
			Windup: 28, Active: 3, Recover: 24, Shape: ZoneCircle, Reach: 3.2, Zone: true, Anchor: AtTarget,
			Damage: 38, Knock: 2.2, Motion: MoveDive, Cooldown: 260, MaxD: 30, Need: NeedPool, Lock: 9, Weight: 9,
			Stages: 1<<1 | 1<<2, Wind: CreatureLurk},
		SkillDef{Kind: CreatureTitanoboa, Name: "slam", Label: "Nện thân", Class: SkillSpecial,
			Windup: 26, Active: 4, Recover: 26, Shape: ZoneCircle, Reach: 4.3, Zone: true,
			Damage: 40, Knock: 2.4, Stun: TicksPerSecond, Cooldown: 200, MaxD: 4.1, Lock: -1, Weight: 6,
			Stages: 1<<1 | 1<<2, Wind: CreatureRoar},
	)
	titanStrike, titanTail, titanCoil, titanVenom, titanDive, titanSlam = ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	SetSkillStages(CreatureTitanoboa, TitanStage2, TitanStage3)
	skillRage[CreatureTitanoboa] = 2
}

// TitanSkills are Titanoboa's moves in its temple, for the client and the
// tools: strike, tail, coil, venom, dive, slam.
func TitanSkills() []SkillID {
	return []SkillID{titanStrike, titanTail, titanCoil, titanVenom, titanDive, titanSlam}
}

// diveNear is how close to its pool Titanoboa must be to dive into it.
const diveNear = 5.0

// poolEdge is the point of site s's pool's edge nearest (x, y), pushed in
// by in (out, if negative).
func (s *lairSite) poolEdge(x, y, in float64) (float64, float64) {
	a := math.Atan2((y-s.poolY)/s.poolRY, (x-s.poolX)/s.poolRX)
	ex, ey := s.poolX+math.Cos(a)*s.poolRX, s.poolY+math.Sin(a)*s.poolRY
	nx, ny := math.Cos(a)/s.poolRX, math.Sin(a)/s.poolRY // the edge's normal
	l := math.Hypot(nx, ny)
	return ex - nx/l*in, ey - ny/l*in
}

// stepTemple is a tick of Titanoboa in its temple, between its moves.
func (w *World) stepTemple(c *Creature, l *bossLair, wounded bool) {
	p, dist := w.bossFoe(c, l, wounded)
	if l.awake && c.State == CreatureIdle && c.timer > 0 {
		c.timer-- // thrown off, reeling: a moment to strike back
		return
	}
	if !w.bossAwake(c, l, p) {
		if !l.awake {
			w.clearLairHazards(l.x, l.y, l.site.r+2) // the fight is over: the flood drains, the venom dries
		}
		return
	}
	if c.SkillStage() >= 2 && !w.flooded(l) {
		w.addLairHazard(LairHazard{Kind: HazardFlood, X: l.x, Y: l.y, R: l.site.r - 0.3, Ends: w.tick + ArenaMaxTicks})
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if w.trySkill(c, p, dist) {
		return
	}
	// With the dive ready it keeps near its pool; else after its foe.
	if i := skillSlot(c.Kind, titanDive); c.SkillStage() >= 1 && i >= 0 && c.sk.cd[i] == 0 && !w.frozenAt(int(l.x), int(l.y)) {
		if ex, ey := l.site.poolEdge(c.X, c.Y, 0); math.Hypot(ex-c.X, ey-c.Y) > diveNear-1 {
			w.bossRunAt(c, l, ex, ey, 0)
			return
		}
	}
	w.bossRunAt(c, l, p.X, p.Y, 2.2)
}

// bossNeed reports whether boss c's need n is met against p (skills.go).
func (w *World) bossNeed(c *Creature, n Need, p *Player) bool {
	i, ok := BossIndex(c.Kind)
	if !ok {
		return false
	}
	l := &w.bosses.lairs[i]
	switch n {
	case NeedPool:
		if l.site.layout != LayoutTemple || w.frozenAt(int(l.site.poolX), int(l.site.poolY)) {
			return false
		}
		ex, ey := l.site.poolEdge(c.X, c.Y, 0)
		return l.site.inPool(c.X, c.Y) || math.Hypot(ex-c.X, ey-c.Y) <= diveNear
	}
	return w.sharkNeed(c, l, n, p) // see sharkfight.go
}

// bossAim aims boss c's move d at p on its own ground, and reports whether
// it did (else the move is aimed as any other: skills.go).
func (w *World) bossAim(c *Creature, d *SkillDef, p *Player) bool {
	i, ok := BossIndex(c.Kind)
	if !ok {
		return false
	}
	l := &w.bosses.lairs[i]
	s := &c.sk
	switch d.Motion {
	case MoveDive:
		// The zone is on the paving just out from the pool's edge nearest
		// the foe; it comes up there.
		s.x, s.y = l.site.poolEdge(p.X, p.Y, -1.3)
		s.ang = math.Atan2(p.Y-c.Y, p.X-c.X)
		return true
	}
	return w.sharkAim(c, l, d, p) // see sharkfight.go
}

// bossWindMove moves boss c through the wind-up of move d: into its pool
// and along under it, or to the shark's run-up.
func (w *World) bossWindMove(c *Creature, d *SkillDef) {
	i, ok := BossIndex(c.Kind)
	if !ok {
		return
	}
	l := &w.bosses.lairs[i]
	s := &c.sk
	if d.Motion != MoveDive {
		w.sharkWindMove(c, l, d) // see sharkfight.go
		return
	}
	// Into the pool first, then under it towards where it comes up.
	tx, ty := l.site.poolEdge(s.x, s.y, 1.2)
	if !l.site.inPool(c.X, c.Y) {
		tx, ty = l.site.poolEdge(c.X, c.Y, 1.5)
	}
	if dd := math.Hypot(tx-c.X, ty-c.Y); dd > 0.05 {
		step := min(dd, 0.42)
		if !w.moveCreature(c, (tx-c.X)/dd*step, (ty-c.Y)/dd*step) {
			c.X, c.Y = c.X+(tx-c.X)/dd*step, c.Y+(ty-c.Y)/dd*step // over the rim it slides, whatever the stones
		}
	}
	c.faceToward(s.x)
}

// surface brings Titanoboa up out of its pool where its dive was aimed.
func (w *World) surface(c *Creature) {
	l := &w.bosses.lairs[0]
	s := &c.sk
	x, y := l.site.poolEdge(s.x, s.y, 0.9)
	if w.creatureFits(c.Kind, x, y) {
		c.X, c.Y = x, y
	}
	c.faceToward(s.x)
}
