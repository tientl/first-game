package sim

import "math"

// The Megalodon's fight at its atoll (atoll.go), on the creatures' moves
// (skills.go). It circles in the deep lagoon with its fin cutting the water
// and never leaves it, but it keeps after whoever comes to its atoll, on
// the reef or in the water:
//
//   - at whoever stands at the lagoon's edge it lunges: a wake runs in from
//     a run-up behind its fin, then it hurls itself out along a line over
//     the edge, and lies beached a moment: the time to strike it.
//   - at whoever stands on the reef it rams the reef: the footholds round
//     them break and drop them in the water, and settle back a while later.
//   - a boat in its lagoon it overturns; a swimmer it bites.
//   - and now and then it leaps clear of the water in the middle to come
//     down in a splash that drenches and throws back whoever is near.
//
// Below half its health its blood is in the water: it circles faster and
// rams the reef more often. Spears thrown from the reef (throw.go) reach it
// wherever it swims. In an older world (the open sea, lair.go) it fights as
// it always did: bossfight.go.

// The Megalodon's stage: where its frenzy begins, as a share of its health.
const SharkFrenzy = 0.5

// Shark tuning, in tiles and ticks.
const (
	sharkEdgeReach = 3.4                    // a foe this far from water it can be in stands at the edge
	sharkRunUp     = 5.0                    // it runs in at the edge from this far back
	sharkBeach     = 1.5                    // and comes this far up over it
	sharkPause     = 2 * TicksPerSecond     // it circles at least this long between two moves...
	sharkFrenzied  = 6 * TicksPerSecond / 5 // ...in its frenzy, this long
	sharkSlide     = 7                      // the last ticks it lies beached, sliding back
)

// The Megalodon's moves, in the order they are added.
var sharkLunge, sharkRam, sharkRamMore, sharkCapsize, sharkBite, sharkBreach SkillID

func init() {
	ids := addSkills(
		SkillDef{Kind: CreatureMegalodon, Name: "lunge", Label: "Lao lên bờ", Class: SkillSpecial,
			Windup: 26, Active: 12, Recover: 24, Shape: ZoneLine, Reach: 8, Width: 0.95, Zone: true,
			Damage: 40, Knock: 2.2, Motion: MoveBeach, Speed: 0.7, Cooldown: 100, MaxD: 40, Need: NeedEdge,
			Lock: 8, Weight: 9, Wind: CreatureLurk},
		SkillDef{Kind: CreatureMegalodon, Name: "ram", Label: "Húc vỡ rạn", Class: SkillSpecial,
			Windup: 28, Active: 3, Recover: 22, Shape: ZoneCircle, Reach: 1.9, Zone: true, Anchor: AtTarget,
			Damage: 16, Knock: 0.8, Fx: FxReef, Motion: MoveRam, Cooldown: 260, MaxD: 40, Need: NeedReef,
			Lock: 10, Weight: 6, Stages: 1 << 0, Wind: CreatureLurk},
		SkillDef{Kind: CreatureMegalodon, Name: "frenzy", Label: "Húc vỡ rạn", Class: SkillSpecial,
			Windup: 26, Active: 3, Recover: 20, Shape: ZoneCircle, Reach: 2.2, Zone: true, Anchor: AtTarget,
			Damage: 18, Knock: 0.8, Fx: FxReef, Motion: MoveRam, Cooldown: 140, MaxD: 40, Need: NeedReef,
			Lock: 10, Weight: 10, Stages: 1 << 1, Wind: CreatureLurk},
		SkillDef{Kind: CreatureMegalodon, Name: "capsize", Label: "Lật thuyền", Class: SkillHeavy,
			Windup: 18, Active: 3, Recover: 16, Shape: ZoneCircle, Reach: 1.9, Zone: true,
			Damage: 30, Fx: FxCapsize, Motion: MoveSwim, Cooldown: 120, MaxD: 40, Need: NeedBoat,
			Lock: 6, Weight: 12, Wind: CreatureLurk},
		SkillDef{Kind: CreatureMegalodon, Name: "bite", Label: "Đớp", Class: SkillHeavy,
			Windup: 16, Active: 2, Recover: 16, Shape: ZoneCircle, Reach: 1.7,
			Damage: 45, Motion: MoveSwim, Cooldown: 60, MaxD: 40, Need: NeedSwim,
			Lock: 5, Weight: 12, Wind: CreatureLurk},
		SkillDef{Kind: CreatureMegalodon, Name: "breach", Label: "Phóng mình", Class: SkillSpecial,
			Windup: 30, Active: 14, Recover: 26, Shape: ZoneCircle, Reach: 5.5, Zone: true,
			Damage: 22, Knock: 2.6, Slow: 3 * TicksPerSecond / 2, Fx: FxSoak, Motion: MoveLeap,
			Cooldown: 300, MinD: 3, MaxD: 40, Lock: 10, Weight: 5, Wind: CreatureLurk},
	)
	sharkLunge, sharkRam, sharkRamMore, sharkCapsize, sharkBite, sharkBreach = ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	SetSkillStages(CreatureMegalodon, SharkFrenzy)
}

// SharkSkills are the Megalodon's moves at its atoll, for the client and
// the tools: lunge, ram, frenzy (the ram in its frenzy), capsize, bite,
// breach.
func SharkSkills() []SkillID {
	return []SkillID{sharkLunge, sharkRam, sharkRamMore, sharkCapsize, sharkBite, sharkBreach}
}

// lagoonPoint is the point nearest (x, y) on a line from lair l's middle
// where the shark fits, in by in more tiles.
func (w *World) lagoonPoint(l *bossLair, x, y, in float64) (float64, float64) {
	dx, dy := x-l.x, y-l.y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		return l.x, l.y
	}
	ux, uy := dx/d, dy/d
	k := CreatureMegalodon
	for r := min(d, l.site.lagoon+1); r > 0; r -= 0.25 {
		if w.creatureFits(k, l.x+ux*r, l.y+uy*r) {
			r = max(0, r-in)
			for ; r > 0 && !w.creatureFits(k, l.x+ux*r, l.y+uy*r); r -= 0.25 {
			}
			return l.x + ux*r, l.y + uy*r
		}
	}
	return l.x, l.y
}

// onAtoll reports whether p is at lair l's atoll: within its reach.
func (l *bossLair) onAtoll(p *Player) bool {
	return math.Hypot(p.X-l.x, p.Y-l.y) <= l.site.r+3
}

// sharkNeed reports whether the shark of lair l has n against p.
func (w *World) sharkNeed(c *Creature, l *bossLair, n Need, p *Player) bool {
	if c.Kind != CreatureMegalodon || l.site.layout != LayoutAtoll || !l.onAtoll(p) {
		return false
	}
	ashore := p.boat == 0 && !w.inWater(p.X, p.Y)
	switch n {
	case NeedEdge:
		if !ashore {
			return false
		}
		wx, wy := w.lagoonPoint(l, p.X, p.Y, 0)
		return math.Hypot(p.X-wx, p.Y-wy) <= sharkEdgeReach
	case NeedReef:
		if !ashore {
			return false
		}
		_, _, ok := w.reefUnder(l, p.X, p.Y, 1.3)
		return ok
	case NeedBoat:
		return p.boat != 0 && math.Hypot(p.X-l.x, p.Y-l.y) <= l.site.lagoon+1
	case NeedSwim:
		return p.boat == 0 && w.inWater(p.X, p.Y) && math.Hypot(p.X-l.x, p.Y-l.y) <= l.site.lagoon+AtollReef
	}
	return false
}

// reefUnder is the whole foothold of lair l's reef nearest (x, y) within
// r, its tile's middle.
func (w *World) reefUnder(l *bossLair, x, y, r float64) (float64, float64, bool) {
	best, bx, by := math.Inf(1), 0.0, 0.0
	R := int(math.Ceil(r))
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - R; ty <= cy+R; ty++ {
		for tx := cx - R; tx <= cx+R; tx++ {
			if !w.inBounds(tx, ty) {
				continue
			}
			i := ty*w.width + tx
			fx, fy := float64(tx)+0.5, float64(ty)+0.5
			if d := math.Hypot(fx-x, fy-y); d <= r && d < best && l.reefTile(i) && w.tiles[i] == TileSand {
				best, bx, by = d, fx, fy
			}
		}
	}
	return bx, by, !math.IsInf(best, 1)
}

// sharkAim aims the shark's move d at p, and reports whether it did.
func (w *World) sharkAim(c *Creature, l *bossLair, d *SkillDef, p *Player) bool {
	if c.Kind != CreatureMegalodon || l.site.layout != LayoutAtoll {
		return false
	}
	s := &c.sk
	switch d.Motion {
	case MoveBeach:
		wx, wy := w.lagoonPoint(l, p.X, p.Y, 0)
		ux, uy := p.X-wx, p.Y-wy
		if n := math.Hypot(ux, uy); n > 1e-6 {
			ux, uy = ux/n, uy/n
		} else {
			ux, uy = (wx-l.x)/max(1e-6, math.Hypot(wx-l.x, wy-l.y)), (wy-l.y)/max(1e-6, math.Hypot(wx-l.x, wy-l.y))
		}
		sx, sy := wx-ux*sharkRunUp, wy-uy*sharkRunUp
		for k := sharkRunUp; k > 0 && !w.creatureFits(c.Kind, sx, sy); k -= 0.5 {
			sx, sy = wx-ux*k, wy-uy*k
		}
		s.x, s.y = sx, sy
		s.ang = math.Atan2(uy, ux)
		s.reach = math.Hypot(p.X-sx, p.Y-sy) + 1.2
		return true
	case MoveRam:
		tx, ty, ok := w.reefUnder(l, p.X, p.Y, 1.6)
		if !ok {
			tx, ty = p.X, p.Y
		}
		s.x, s.y = tx, ty
		s.bx, s.by = w.lagoonPoint(l, tx, ty, 0)
		s.ang = math.Atan2(ty-s.by, tx-s.bx)
		return true
	case MoveLeap: // the breach comes down in the water by them
		x, y := p.X, p.Y
		if !w.creatureFits(c.Kind, x, y) {
			x, y = w.lagoonPoint(l, p.X, p.Y, 2.4)
		}
		s.x, s.y = x, y
		s.ang = math.Atan2(y-c.Y, x-c.X)
		return true
	}
	return false
}

// sharkWindMove swims the shark of lair l into place through the wind-up of
// move d: to its run-up, or to the water by the reef it will ram.
func (w *World) sharkWindMove(c *Creature, l *bossLair, d *SkillDef) {
	s := &c.sk
	tx, ty := s.x, s.y
	if d.Motion == MoveRam {
		tx, ty = s.bx, s.by
	}
	dx, dy := tx-c.X, ty-c.Y
	if dd := math.Hypot(dx, dy); dd > 0.1 {
		step := min(dd, 0.5)
		if !w.moveCreature(c, dx/dd*step, dy/dd*step) {
			c.dodge(w, tx, ty)
			w.moveCreature(c, c.hx*step, c.hy*step)
		}
	}
	c.faceToward(c.X + math.Cos(s.ang))
}

// beachDash is a tick of the shark's lunge: along its line, out of the
// water and up over the edge, striking whoever it runs into.
func (w *World) beachDash(c *Creature, d *SkillDef) {
	s := &c.sk
	ux, uy := math.Cos(s.ang), math.Sin(s.ang)
	nx, ny := c.X+ux*d.Speed, c.Y+uy*d.Speed
	along := (nx-s.sx)*ux + (ny-s.sy)*uy
	beached := math.Hypot(nx-s.bx, ny-s.by)
	stop := along > s.reach-0.4
	switch {
	case stop:
	case w.creatureFits(c.Kind, nx, ny):
		c.X, c.Y = nx, ny
		s.bx, s.by = nx, ny // the last of the water
	case beached <= sharkBeach && !w.blocked(int(math.Floor(nx)), int(math.Floor(ny))):
		c.X, c.Y = nx, ny // up over the edge
	default:
		stop = true
	}
	c.faceToward(c.X + ux)
	s.x, s.y = s.sx, s.sy // the line stays where it was drawn
	w.dashHits(c, d)
	if stop || s.hit && beached > 0.5 {
		s.act = s.t
	}
}

// beachedStep is a tick of the shark lying beached after its lunge: it
// thrashes, then slides back into the water.
func (w *World) beachedStep(c *Creature, d *SkillDef) {
	s := &c.sk
	left := d.Recover - s.t
	if left > sharkSlide || left < 0 {
		return
	}
	k := 1 / float64(left+1)
	c.X += (s.bx - c.X) * k
	c.Y += (s.by - c.Y) * k
}

// sharkBack puts the shark back into the deep at the end of a move: never
// left on the reef, or where it does not fit.
func (w *World) sharkBack(c *Creature, d *SkillDef) {
	if c.Kind != CreatureMegalodon {
		return
	}
	l := &w.bosses.lairs[5]
	if l.c != c || l.site.layout != LayoutAtoll {
		return
	}
	if !w.creatureFits(c.Kind, c.X, c.Y) {
		s := &c.sk
		if w.creatureFits(c.Kind, s.bx, s.by) {
			c.X, c.Y = s.bx, s.by
		} else {
			c.X, c.Y = w.lagoonPoint(l, c.X, c.Y, 0.5)
		}
	}
	if c.State != CreatureRoar {
		c.State, c.timer = CreatureLurk, 0
	}
}

// ramReef is the shark c's ram landing on the reef at (x, y): the
// footholds within r break.
func (w *World) ramReef(c *Creature, x, y, r float64) {
	if c.Kind != CreatureMegalodon {
		return
	}
	if l := &w.bosses.lairs[5]; l.site.layout == LayoutAtoll {
		w.breakReef(l, x, y, r)
	}
}

// stepAtoll is a tick of the Megalodon at its atoll, between its moves:
// circling in the lagoon under the water, its fin showing.
func (w *World) stepAtoll(c *Creature, l *bossLair, wounded bool) {
	p, dist := w.bossFoe(c, l, wounded)
	if !w.bossAwake(c, l, p) {
		if !l.awake {
			w.clearLairHazards(l.x, l.y, l.site.r+2) // the fight is over: the reef settles, the blood clears
		}
		return
	}
	frenzy := c.SkillStage() >= 1
	if frenzy && !w.bloodied(l) {
		w.addLairHazard(LairHazard{Kind: HazardBlood, X: l.x, Y: l.y, R: l.site.lagoon, Ends: w.tick + ArenaMaxTicks})
	}
	c.State = CreatureLurk
	pause := sharkPause
	if frenzy {
		pause = sharkFrenzied
	}
	if l.act++; l.act >= pause && w.trySkill(c, p, dist) {
		l.act = 0
		return
	}
	w.sharkCircle(c, l, p, frenzy)
}

// bloodied reports whether the shark of lair l bleeds in its lagoon.
func (w *World) bloodied(l *bossLair) bool {
	for _, h := range w.hz.list {
		if h.Kind == HazardBlood && math.Hypot(h.X-l.x, h.Y-l.y) < 1 {
			return true
		}
	}
	return false
}

// sharkCircle swims the shark round its lagoon, keeping its foe's side.
func (w *World) sharkCircle(c *Creature, l *bossLair, p *Player, frenzy bool) {
	sp := c.Kind.def().run * 0.6
	spin := 0.07
	if frenzy {
		sp, spin = sp*1.5, 0.11
	}
	r := math.Hypot(p.X-l.x, p.Y-l.y) - 6
	r = max(5, min(r, l.site.lagoon-3.5))
	if c.ID%2 == 1 {
		spin = -spin
	}
	a := math.Atan2(c.Y-l.y, c.X-l.x) + spin
	tx, ty := l.x+math.Cos(a)*r, l.y+math.Sin(a)*r
	if !w.runAt(c, tx, ty, sp, 0) {
		c.dodge(w, l.x, l.y)
		w.moveCreature(c, c.hx*sp, c.hy*sp)
	}
	c.State = CreatureLurk
}
