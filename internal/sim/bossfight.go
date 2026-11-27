package sim

import "math"

// How the six bosses fight; who they are is in bosses.go.
//
// Every boss telegraphs its blows: it bares its teeth (CreatureStrike) and
// the bite lands bossWindup ticks later on whoever is still in reach, so a
// quick player can step back from it. Its special move (CreatureSpecial,
// or CreatureGrab for the snake's coils) has a longer wind-up still.

// Boss move tuning. Damage is before armour; speeds in tiles per tick.
const (
	TitanoboaGrabReach = 2.4                // it seizes a player this close instead of biting
	TitanoboaGrabTicks = 5 * TicksPerSecond // and holds them at most this long...
	TitanoboaSqueeze   = 14.0               // ...squeezing this hard every second,
	TitanoboaGrabBreak = 4                  // unless it loses this many percent of its health first

	SpinoAmbush     = 5.0 // it bursts out of the water at a player this close
	SpinoLungeSpeed = 7.5 / TicksPerSecond
	SpinoLungeTicks = 14
	SpinoLungeBite  = 46.0
	SpinoUpTicks    = 14 * TicksPerSecond // out of the water at most this long, then back in
	SpinoUnderTicks = 3 * TicksPerSecond  // under at least this long before it bursts out again
	SpinoWaitTicks  = 6 * TicksPerSecond  // a foe who keeps from the water is hunted on land after this

	KingStompReach  = 3.6
	KingStomp       = 44.0
	KingStompKnock  = 1.2
	KingRearTicks   = 18 // up on its hind legs before it comes down
	KingChargeCD    = 7 * TicksPerSecond
	KingChargeSpeed = 7.0 / TicksPerSecond
	KingChargeTicks = 30
	KingCharge      = 40.0
	KingChargeKnock = 3.0
	KingWinded      = 16
	KingCallRange   = 16.0 // the mammoths this close come to its call
	KingCallEvery   = 5 * TicksPerSecond

	CarchaCrouchTicks = 10 // head low, eyes on its prey...
	CarchaRunTicks    = 24 // ...then off at a run
	CarchaRunSpeed    = 9.0 / TicksPerSecond
	CarchaCharge      = 44.0
	CarchaChargeKnock = 2.5
	CarchaWinded      = 24

	GigaTailReach  = 4.2
	GigaTail       = 36.0
	GigaTailKnock  = 2.0
	GigaTailWindup = 14

	MegaCircle      = 5.0  // it circles its prey this far off
	megaAwakeSight  = 25.6 // once in a fight it follows its prey across all its water
	MegaCircleTicks = 4 * TicksPerSecond
	MegaRamBoat     = 5 // boat hit points
	MegaRamRider    = 12.0
	MegaSwimBite    = 45.0
	MegaBiteBoat    = 0
	MegaUpTicks     = 5 * TicksPerSecond // up at the surface after a ram: it can be fought
	MegaBreachTicks = 24
	MegaBreachReach = 2.8
	MegaBreachBoat  = 12
	MegaBreachRider = 26.0
	MegaBreachSwim  = 55.0
)

// stepBoss is one tick of boss c.
func (w *World) stepBoss(c *Creature) {
	i, _ := BossIndex(c.Kind)
	l := &w.bosses.lairs[i]
	if l.c != c {
		w.wander(c) // not this world's own: it only roams
		return
	}
	if c.State == CreatureHurt {
		c.State, c.timer = c.after, c.afterT // far too big to flinch
	}
	if c.biteCD > 0 {
		c.biteCD--
	}
	if l.special > 0 {
		l.special--
	}
	if l.charge > 0 {
		l.charge--
	}
	wounded := c.HP < l.hp
	l.hp = c.HP
	switch l.site.layout {
	case LayoutTemple:
		w.stepTemple(c, l, wounded) // see titanfight.go
		return
	case LayoutAtoll:
		w.stepAtoll(c, l, wounded) // see sharkfight.go
		return
	}
	switch c.Kind {
	case CreatureTitanoboa:
		w.stepTitanoboa(c, l, wounded)
	case CreatureSpino:
		w.stepSpino(c, l, wounded)
	case CreatureMammothKing:
		w.stepMammothKing(c, l, wounded)
	case CreatureCarcharo:
		w.stepCarcharo(c, l, wounded)
	case CreatureGiganoto:
		w.stepGiganoto(c, l, wounded)
	case CreatureMegalodon:
		w.stepMegalodon(c, l, wounded)
	}
}

// --- what every boss does -----------------------------------------------------

// bossCanFight reports whether boss c goes for p at all: someone alive and
// awake near its lair, on its own ground. The shark hunts those on the
// water; the beasts of the land leave swimmers and sailors alone, but for
// Spinosaurus, which takes swimmers in its pond.
func (w *World) bossCanFight(c *Creature, l *bossLair, p *Player) bool {
	if p == nil || p.Dead || p.Asleep {
		return false
	}
	if math.Hypot(p.X-l.x, p.Y-l.y) > bossTuneOf(c.Kind).leash+4 {
		return false
	}
	onWater := p.boat != 0 || w.inWater(p.X, p.Y)
	if l.site.layout == LayoutAtoll {
		return onWater || l.onAtoll(p) // on its reef too: it cannot reach dry land, but it keeps after them
	}
	switch c.Kind {
	case CreatureMegalodon:
		return onWater
	case CreatureSpino:
		return p.boat == 0
	}
	return !onWater
}

// bossFoe is whom boss c fights now, and how far off they are: the foe it
// has while they stay in sight, else the nearest it can see. A blow wakes
// it to whoever is close.
func (w *World) bossFoe(c *Creature, l *bossLair, wounded bool) (*Player, float64) {
	if l.site.layout != LayoutHollow {
		return w.arenaFoe(c, l, wounded)
	}
	sight := bossTuneOf(c.Kind).sight
	if l.awake {
		sight = max(sight*1.6, 12)
		if c.Kind == CreatureMegalodon {
			sight = megaAwakeSight
		}
	}
	if p := w.players[l.foe]; w.bossCanFight(c, l, p) {
		if d := math.Hypot(p.X-c.X, p.Y-c.Y); d <= sight {
			return p, d
		}
	}
	if wounded {
		sight = max(sight, bossWoundWake)
	}
	var best *Player
	bestD := sight
	for _, id := range w.playerOrder {
		p := w.players[id]
		if !w.bossCanFight(c, l, p) {
			continue
		}
		if d := math.Hypot(p.X-c.X, p.Y-c.Y); d <= bestD {
			best, bestD = p, d
		}
	}
	if best == nil {
		return nil, math.Inf(1)
	}
	l.foe = best.ID
	return best, bestD
}

// arenaFoe is bossFoe at a lair of StyleLairs: asleep, the boss wakes for
// whoever comes into its arena (the courtyard, the atoll) or wounds it;
// awake, it keeps after its foe anywhere in it, and turns on the nearest
// other when it loses them.
func (w *World) arenaFoe(c *Creature, l *bossLair, wounded bool) (*Player, float64) {
	r := l.arenaR(c.Kind) - 0.5
	if l.awake {
		r += 4
	}
	in := func(p *Player) bool {
		return w.bossCanFight(c, l, p) && math.Hypot(p.X-l.x, p.Y-l.y) <= r
	}
	if p := w.players[l.foe]; l.awake && p != nil && in(p) {
		return p, math.Hypot(p.X-c.X, p.Y-c.Y)
	}
	var best *Player
	bestD := math.Inf(1)
	for _, id := range w.playerOrder {
		p := w.players[id]
		if !in(p) && !(wounded && w.bossCanFight(c, l, p) && math.Hypot(p.X-c.X, p.Y-c.Y) <= bossWoundWake) {
			continue
		}
		if d := math.Hypot(p.X-c.X, p.Y-c.Y); d < bestD {
			best, bestD = p, d
		}
	}
	if best == nil {
		return nil, math.Inf(1)
	}
	l.foe = best.ID
	return best, bestD
}

// bossAwake runs what every boss does between its own moves: resting or
// going home with no foe, waking with a roar when it finds one, and its roar
// of rage below half its health. It reports whether c is free to go for p.
func (w *World) bossAwake(c *Creature, l *bossLair, p *Player) bool {
	if p == nil {
		l.lost++
		if l.awake && l.lost <= bossLoseTicks {
			c.State, c.timer = CreatureIdle, 0 // it looks about for whoever it lost
			return false
		}
		l.awake, l.foe, l.roared = false, 0, false
		w.bossHome(c, l)
		return false
	}
	l.lost, l.stuck, l.prey = 0, 0, 0
	if !l.awake || !l.roared {
		l.awake, l.roared = true, true
		l.special = w.bossCD(l, bossTuneOf(c.Kind).specialCD/2)
		w.bossRoar(c, l, p)
		return false
	}
	if c.State == CreatureRoar {
		c.faceToward(p.X)
		if c.timer--; c.timer > 0 {
			return false
		}
		c.State = CreatureChase
	}
	if !l.enraged && c.HP*2 < c.Kind.MaxHP() {
		l.enraged = true
		w.bossRoar(c, l, p)
		return false
	}
	return true
}

// bossRoar starts c roaring at p: when a fight begins, and in its rage.
func (w *World) bossRoar(c *Creature, l *bossLair, p *Player) {
	c.State, c.timer = CreatureRoar, BossRoarTicks
	c.faceToward(p.X)
	if c.Kind == CreatureMammothKing {
		w.callTheHerd(c, p)
	}
}

// bossHome takes c back to its lair; there it rests and heals.
func (w *World) bossHome(c *Creature, l *bossLair) {
	w.bossLetGo(c, l)
	if c.Kind == CreatureTitanoboa && w.titanProwl(c, l) {
		return // out after a small animal: see lair.go
	}
	d := c.Kind.def()
	sea := c.Kind == CreatureMegalodon || (c.Kind == CreatureSpino && w.inWater(c.X, c.Y))
	if math.Hypot(l.x-c.X, l.y-c.Y) > d.leash+1 {
		c.State = CreatureWander
		if sea {
			c.State = CreatureLurk
		}
		if !w.runAt(c, l.x, l.y, d.walk*2, 0) {
			if l.stuck++; l.stuck > bossStuckTicks {
				if _, pd := w.nearestPlayer(c.X, c.Y); pd > 20 && w.creatureFits(c.Kind, l.x, l.y) {
					c.X, c.Y = l.x, l.y // home by a way nobody saw
				}
			}
		}
		return
	}
	l.stuck = 0
	switch c.Kind {
	case CreatureTitanoboa:
		c.State = CreatureIdle // coiled in the undergrowth
	case CreatureSpino:
		if w.inWater(c.X, c.Y) {
			c.State = CreatureLurk // only its sail and eyes above the water
		} else if !w.runAt(c, l.x, l.y, d.walk*2, 0) {
			c.State = CreatureIdle
		} else {
			c.State = CreatureWander
		}
	case CreatureMegalodon:
		w.megaIdle(c, l) // see lair.go
	default:
		w.wander(c)
	}
	if max := c.Kind.MaxHP(); c.HP < max && w.tick%bossHealEvery == 0 {
		c.HP = min(max, c.HP+max/bossHealShare)
		if c.HP*2 >= max {
			l.enraged = false
		}
	}
	l.hp = c.HP
}

// bossCD is cooldown cd, shortened in a rage.
func (w *World) bossCD(l *bossLair, cd int) int {
	if l.enraged {
		return cd * bossEnragePct / 100
	}
	return cd
}

// bossRunAt moves c towards (x, y), stopping keep short of it, but never
// past its leash.
func (w *World) bossRunAt(c *Creature, l *bossLair, x, y, keep float64) {
	c.faceToward(x)
	sp := c.Kind.def().run
	if l.enraged {
		sp *= bossEnrageRun
	}
	dx, dy := x-c.X, y-c.Y
	d := math.Hypot(dx, dy)
	if d <= keep || d < 1e-9 {
		return
	}
	step := min(sp, d-keep)
	nx, ny := c.X+dx/d*step, c.Y+dy/d*step
	leash := bossTuneOf(c.Kind).leash
	if from := math.Hypot(nx-l.x, ny-l.y); from > leash && from > math.Hypot(c.X-l.x, c.Y-l.y) {
		return // at the end of its leash: it will not come further
	}
	if !w.moveCreature(c, dx/d*step, dy/d*step) {
		c.hx, c.hy = dx/d, dy/d
		c.dodge(w, c.X-dx, c.Y-dy) // round whatever is in the way
		w.moveCreature(c, c.hx*step, c.hy*step)
	}
	c.faceToward(x)
}

// bossBite starts c's main blow at p: it bares its teeth, and the blow
// lands bossWindup ticks later. grab makes it a seizing (Titanoboa).
func (w *World) bossBite(c *Creature, l *bossLair, p *Player, grab bool) {
	c.State, c.timer = CreatureStrike, bossWindup+bossRecover
	l.landed, l.grab = false, grab
	c.faceToward(p.X)
}

// bossBiteStep is a tick of c's main blow: it lands on p if they are still
// in reach when the wind-up ends.
func (w *World) bossBiteStep(c *Creature, l *bossLair, p *Player) {
	t := bossTuneOf(c.Kind)
	c.timer--
	if !l.landed && c.timer <= bossRecover {
		l.landed = true
		c.biteCD = w.bossCD(l, t.biteCD)
		if p != nil && math.Hypot(p.X-c.X, p.Y-c.Y) <= t.reach+0.15 {
			if l.grab && p.boat == 0 {
				w.titanoboaSeize(c, l, p)
				return
			}
			w.bossHit(c, p, t.bite, t.knock)
		}
	}
	if c.timer <= 0 {
		c.State = CreatureChase
	}
}

// bossHit lands a blow of c's on p, throwing them back knock tiles.
func (w *World) bossHit(c *Creature, p *Player, dmg, knock float64) {
	struck(p, dmg)
	p.notify(NoticeBitten, uint8(c.Kind))
	if knock > 0 {
		w.shove(p, c.X, c.Y, knock)
	}
}

// bossBlast lands a blow of c's on everyone within r of (x, y) on land: a
// stomp, the sweep of a tail.
func (w *World) bossBlast(c *Creature, x, y, r, dmg, knock float64) {
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || p.boat != 0 || math.Hypot(p.X-x, p.Y-y) > r {
			continue
		}
		struck(p, dmg)
		p.notify(NoticeBitten, uint8(c.Kind))
		w.shove(p, x, y, knock)
	}
}

// bossCharge is a tick of c running along its heading at speed: it throws
// the first player it runs into, and is winded at the end of the run.
func (w *World) bossCharge(c *Creature, l *bossLair, speed, dmg, knock float64, winded int) {
	moved := w.moveCreature(c, l.hx*speed, l.hy*speed)
	hit := false
	r := c.Kind.def().radius + 0.7
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || p.boat != 0 || w.inWater(p.X, p.Y) || math.Hypot(p.X-c.X, p.Y-c.Y) > r {
			continue
		}
		w.bossHit(c, p, dmg, knock)
		hit = true
	}
	far := math.Hypot(c.X-l.x, c.Y-l.y) > bossTuneOf(c.Kind).leash
	if hit || !moved || c.timer <= 0 || far {
		c.State, c.timer = CreatureIdle, winded // out of breath: a moment to strike back
	}
}

// shove throws p up to d tiles away from (x, y), as far as the ground lets
// it. Riders and sailors keep their seats.
func (w *World) shove(p *Player, x, y, d float64) {
	if p.Dead || p.mount != 0 || p.boat != 0 {
		return
	}
	dx, dy := p.X-x, p.Y-y
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		dx, dy, l = 1, 0, 1
	}
	if p.guard.up && p.guard.at == w.tick {
		d *= GuardKnock // taken on a raised shield; see shield.go
	}
	// Eased: most of the way now, the rest over the next ticks; see knock.go.
	p.knockX += dx / l * d
	p.knockY += dy / l * d
	w.knockMove(p, KnockFirst)
}

// bossLetGo releases whoever c holds in its coils.
func (w *World) bossLetGo(c *Creature, l *bossLair) {
	if l.held == 0 {
		return
	}
	if p := w.players[l.held]; p != nil && !p.Dead {
		w.shove(p, c.X, c.Y-0.3, 0.8)
	}
	l.held = 0
	if c.State == CreatureGrab {
		c.State, c.timer = CreatureChase, 0
		c.biteCD = max(c.biteCD, bossTuneOf(c.Kind).biteCD/2)
	}
}

// --- Titanoboa ----------------------------------------------------------------

func (w *World) stepTitanoboa(c *Creature, l *bossLair, wounded bool) {
	t := bossTuneOf(c.Kind)
	if c.State == CreatureGrab {
		w.titanoboaSqueeze(c, l)
		return
	}
	p, dist := w.bossFoe(c, l, wounded)
	if c.State == CreatureStrike {
		w.bossBiteStep(c, l, p)
		return
	}
	if !w.bossAwake(c, l, p) {
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= t.reach && c.biteCD == 0 {
		w.bossBite(c, l, p, l.special == 0 && dist <= TitanoboaGrabReach)
		return
	}
	w.bossRunAt(c, l, p.X, p.Y, 1.8)
}

// titanoboaSeize coils c round p.
func (w *World) titanoboaSeize(c *Creature, l *bossLair, p *Player) {
	if p.mount != 0 {
		w.dismount(p)
	}
	c.State, c.timer = CreatureGrab, TitanoboaGrabTicks
	l.held, l.grabHP, l.act = p.ID, c.HP, 0
	l.hx, l.hy = p.X, p.Y
	l.special = w.bossCD(l, bossTuneOf(c.Kind).specialCD)
	if w.creatureFits(c.Kind, p.X, p.Y-0.05) {
		c.X, c.Y = p.X, p.Y-0.05 // its coils round them, they in front
	}
	struck(p, bossTuneOf(c.Kind).bite/2)
	p.notify(NoticeBitten, uint8(c.Kind))
}

// titanoboaSqueeze is a tick of c holding its prey: they cannot move and
// are squeezed every second, until they wriggle free (holding a walk key
// frees them twice as fast), hurt it enough, or it lets go.
func (w *World) titanoboaSqueeze(c *Creature, l *bossLair) {
	p := w.players[l.held]
	if p == nil || p.Dead || c.HP <= l.grabHP-c.Kind.MaxHP()*TitanoboaGrabBreak/100 {
		w.bossLetGo(c, l)
		return
	}
	c.timer--
	if p.input.MoveX != 0 || p.input.MoveY != 0 {
		c.timer-- // struggling
	}
	if c.timer <= 0 {
		w.bossLetGo(c, l)
		return
	}
	p.X, p.Y = l.hx, l.hy // held fast
	if l.act++; l.act%TicksPerSecond == 0 {
		struck(p, TitanoboaSqueeze)
		p.notify(NoticeBitten, uint8(c.Kind))
	}
}

// --- Spinosaurus --------------------------------------------------------------

func (w *World) stepSpino(c *Creature, l *bossLair, wounded bool) {
	t := bossTuneOf(c.Kind)
	d := c.Kind.def()
	p, dist := w.bossFoe(c, l, wounded)
	switch c.State {
	case CreatureStrike:
		w.bossBiteStep(c, l, p)
		return
	case CreatureSpecial:
		w.spinoLunge(c, l, p, dist)
		return
	case CreatureLurk:
		w.spinoLurk(c, l, p, dist)
		return
	case CreatureFlee: // back to the water
		if w.inWater(c.X, c.Y) {
			c.State, c.timer, l.act = CreatureLurk, SpinoUnderTicks, 0
			return
		}
		if !w.runAt(c, l.x, l.y, d.run, 0) {
			c.State, l.act = CreatureChase, 0 // cut off from the water: it fights on
		}
		return
	}
	if !w.bossAwake(c, l, p) {
		return
	}
	c.State = CreatureChase
	if l.act++; l.act > SpinoUpTicks && math.Hypot(l.x-c.X, l.y-c.Y) < bossLeash {
		c.State, l.act = CreatureFlee, 0 // it slides back into its pond a while
		return
	}
	c.faceToward(p.X)
	if dist <= t.reach && c.biteCD == 0 {
		w.bossBite(c, l, p, false) // a swipe of its claws
		return
	}
	w.bossRunAt(c, l, p.X, p.Y, 1.9)
}

// spinoLurk is a tick of c under the water: asleep at home, or stalking its
// foe along the pond until it can burst out at them.
func (w *World) spinoLurk(c *Creature, l *bossLair, p *Player, dist float64) {
	if c.timer > 0 {
		c.timer--
	}
	if p == nil {
		if l.awake {
			if l.lost++; l.lost > bossLoseTicks {
				l.awake, l.foe, l.roared = false, 0, false
			}
		}
		if !l.awake {
			w.bossHome(c, l)
		}
		return
	}
	l.lost = 0
	ambush := SpinoAmbush
	if l.enraged {
		ambush++
	}
	if c.timer == 0 && dist <= ambush {
		if !l.awake {
			l.awake = true
			l.special = 0
		}
		c.State, c.timer, l.landed = CreatureSpecial, SpinoLungeTicks, false
		l.hx, l.hy = (p.X-c.X)/max(dist, 1e-6), (p.Y-c.Y)/max(dist, 1e-6)
		c.faceToward(p.X)
		return
	}
	if !l.awake {
		return // waiting, still as a log
	}
	if l.act++; l.act > SpinoWaitTicks && c.timer == 0 {
		c.State, l.act = CreatureChase, 0 // they keep from the water: it climbs out after them
		return
	}
	// Along the pond towards them, staying under.
	x, y := c.X, c.Y
	sp := c.Kind.def().run * 0.7
	dx, dy := p.X-c.X, p.Y-c.Y
	if w.moveCreature(c, dx/dist*sp, dy/dist*sp) && (!w.inWater(c.X, c.Y) || math.Hypot(c.X-l.x, c.Y-l.y) > bossLeash) {
		c.X, c.Y = x, y // that would be out on the bank
	}
	c.faceToward(p.X)
}

// spinoLunge is a tick of c bursting out of the water, jaws wide.
func (w *World) spinoLunge(c *Creature, l *bossLair, p *Player, dist float64) {
	c.timer--
	if !l.landed {
		w.moveCreature(c, l.hx*SpinoLungeSpeed, l.hy*SpinoLungeSpeed)
	}
	if p != nil {
		dist = math.Hypot(p.X-c.X, p.Y-c.Y)
	}
	if !l.landed && p != nil && dist <= bossTuneOf(c.Kind).reach {
		l.landed = true
		w.bossHit(c, p, SpinoLungeBite, 0)
		c.timer = min(c.timer, 4)
	}
	if c.timer > 0 {
		return
	}
	c.State, l.act = CreatureChase, 0
	c.biteCD = max(c.biteCD, bossTuneOf(c.Kind).biteCD/2)
	if !l.roared && p != nil {
		l.roared = true
		w.bossRoar(c, l, p) // out of the water, it roars its challenge
	}
}

// --- the mammoth king ---------------------------------------------------------

func (w *World) stepMammothKing(c *Creature, l *bossLair, wounded bool) {
	t := bossTuneOf(c.Kind)
	p, dist := w.bossFoe(c, l, wounded)
	switch c.State {
	case CreatureStrike:
		if l.grab { // the charge, tusks low
			c.timer--
			w.bossCharge(c, l, KingChargeSpeed, KingCharge, KingChargeKnock, KingWinded)
			return
		}
		w.bossBiteStep(c, l, p)
		return
	case CreatureSpecial: // reared up; then down it comes
		c.timer--
		if !l.landed && c.timer <= bossRecover {
			l.landed = true
			w.bossBlast(c, c.X, c.Y, KingStompReach, KingStomp, KingStompKnock)
		}
		if c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	case CreatureIdle:
		if l.awake && c.timer > 0 { // winded after a charge
			c.timer--
			return
		}
	}
	if !w.bossAwake(c, l, p) {
		return
	}
	if w.tick%KingCallEvery == 0 {
		w.callTheHerd(c, p)
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	switch {
	case l.special == 0 && dist <= KingStompReach-0.6:
		c.State, c.timer, l.landed = CreatureSpecial, KingRearTicks+bossRecover, false
		l.special = w.bossCD(l, t.specialCD)
	case l.charge == 0 && dist >= 5 && dist <= 12:
		c.State, c.timer, l.grab = CreatureStrike, KingChargeTicks, true
		l.hx, l.hy = (p.X-c.X)/dist, (p.Y-c.Y)/dist
		l.charge = w.bossCD(l, KingChargeCD)
	case dist <= t.reach && c.biteCD == 0:
		w.bossBite(c, l, p, false) // a swing of its tusks
	default:
		w.bossRunAt(c, l, p.X, p.Y, 2.2)
	}
}

// callTheHerd sets the mammoths round the king on p: those close enough
// charge, the others come closer.
func (w *World) callTheHerd(c *Creature, p *Player) {
	for _, o := range w.fauna.creatures {
		if o.Kind != CreatureMammoth || o.gone || o.Tamed() || math.Hypot(o.X-c.X, o.Y-c.Y) > KingCallRange {
			continue
		}
		o.homeX, o.homeY = c.X, c.Y // the herd keeps to its king
		if math.Hypot(o.X-p.X, o.Y-p.Y) <= GrazerGiveUp {
			o.foe, o.rage = p.ID, 3*HerdRageTicks
		}
	}
}

// --- Carcharodontosaurus ------------------------------------------------------

func (w *World) stepCarcharo(c *Creature, l *bossLair, wounded bool) {
	t := bossTuneOf(c.Kind)
	p, dist := w.bossFoe(c, l, wounded)
	switch c.State {
	case CreatureStrike:
		w.bossBiteStep(c, l, p)
		return
	case CreatureSpecial: // the charge
		c.timer--
		if c.timer >= CarchaRunTicks { // crouched, head low, eyes on its prey
			if p != nil {
				c.faceToward(p.X)
				l.hx, l.hy = (p.X-c.X)/max(dist, 1e-6), (p.Y-c.Y)/max(dist, 1e-6)
			}
			return
		}
		w.bossCharge(c, l, CarchaRunSpeed, CarchaCharge, CarchaChargeKnock, CarchaWinded)
		return
	case CreatureIdle:
		if l.awake && c.timer > 0 { // winded after a charge
			c.timer--
			return
		}
	}
	if !w.bossAwake(c, l, p) {
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	switch {
	case l.special == 0 && dist >= 2 && dist <= 12:
		c.State, c.timer = CreatureSpecial, CarchaCrouchTicks+CarchaRunTicks
		l.special = w.bossCD(l, t.specialCD)
	case dist <= t.reach && c.biteCD == 0:
		w.bossBite(c, l, p, false)
	default:
		w.bossRunAt(c, l, p.X, p.Y, 1.8)
	}
}

// --- Giganotosaurus -----------------------------------------------------------

func (w *World) stepGiganoto(c *Creature, l *bossLair, wounded bool) {
	t := bossTuneOf(c.Kind)
	p, dist := w.bossFoe(c, l, wounded)
	switch c.State {
	case CreatureStrike:
		w.bossBiteStep(c, l, p)
		return
	case CreatureSpecial: // the tail comes round
		c.timer--
		if !l.landed && c.timer <= bossRecover+2 {
			l.landed = true
			w.bossBlast(c, c.X, c.Y, GigaTailReach, GigaTail, GigaTailKnock)
		}
		if c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	}
	if !w.bossAwake(c, l, p) {
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	switch {
	case l.special == 0 && dist <= GigaTailReach-0.5:
		c.State, c.timer, l.landed = CreatureSpecial, GigaTailWindup+bossRecover+2, false
		l.special = w.bossCD(l, t.specialCD)
		c.FaceLeft = !c.FaceLeft // it turns, to bring the tail round
	case dist <= t.reach && c.biteCD == 0:
		w.bossBite(c, l, p, false)
	default:
		w.bossRunAt(c, l, p.X, p.Y, 2.0)
	}
}

// --- Megalodon ----------------------------------------------------------------

func (w *World) stepMegalodon(c *Creature, l *bossLair, wounded bool) {
	t := bossTuneOf(c.Kind)
	d := c.Kind.def()
	p, dist := w.bossFoe(c, l, wounded)
	switch c.State {
	case CreatureStrike: // a bite at the surface
		c.timer--
		if !l.landed && c.timer <= bossRecover {
			l.landed = true
			c.biteCD = w.bossCD(l, t.biteCD)
			if p != nil && dist <= t.reach+0.4 {
				w.seaBite(c, p, MegaBiteBoat, t.bite, MegaSwimBite)
			}
		}
		if c.timer <= 0 {
			c.State = CreatureChase
		}
		return
	case CreatureSpecial:
		w.megaBreach(c, l)
		return
	}
	if !w.bossAwake(c, l, p) {
		return
	}
	if c.State == CreatureLurk { // under, circling its prey: only its fin shows
		l.act++
		circle := MegaCircleTicks
		if l.enraged {
			circle = circle * 2 / 3
		}
		sp := d.run
		if l.enraged {
			sp *= bossEnrageRun
		}
		if l.act >= circle {
			if l.special == 0 && dist <= 9 { // up and out of the water, onto them
				c.State, c.timer = CreatureSpecial, MegaBreachTicks
				l.hx, l.hy, l.landed = p.X-(p.X-c.X)/max(dist, 1e-6)*1.2, p.Y-(p.Y-c.Y)/max(dist, 1e-6)*1.2, false // beside the boat, not on it
				l.special, l.act = w.bossCD(l, t.specialCD), 0
				c.faceToward(p.X)
				return
			}
			if dist <= t.reach { // the ram
				w.seaBite(c, p, MegaRamBoat, MegaRamRider, MegaSwimBite)
				c.State, c.timer, l.landed, l.act = CreatureStrike, AttackTicks, true, 0
				c.faceToward(p.X)
				return
			}
			w.runAt(c, p.X, p.Y, sp, 0)
			return
		}
		a := math.Atan2(c.Y-p.Y, c.X-p.X) + 0.07
		if !w.runAt(c, p.X+math.Cos(a)*MegaCircle, p.Y+math.Sin(a)*MegaCircle, sp, 0) {
			c.dodge(w, p.X, p.Y)
			w.moveCreature(c, c.hx*sp, c.hy*sp)
		}
		return
	}
	// Up at the surface, after a ram or a leap: it can be fought.
	if l.act++; l.act > MegaUpTicks {
		c.State, l.act = CreatureLurk, 0 // it dives
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if dist <= t.reach && c.biteCD == 0 {
		c.State, c.timer, l.landed = CreatureStrike, bossWindup+bossRecover, false
		return
	}
	w.runAt(c, p.X, p.Y, d.walk*1.5, 2.0)
}

// megaBreach is a tick of c's leap: clear of the water it flies at where
// its prey was, and comes down on whoever is there.
func (w *World) megaBreach(c *Creature, l *bossLair) {
	c.timer--
	k := 1 / float64(c.timer+1)
	if x, y := c.X+(l.hx-c.X)*k, c.Y+(l.hy-c.Y)*k; w.creatureFits(c.Kind, x, y) {
		c.faceToward(x)
		c.X, c.Y = x, y
	}
	if c.timer > 0 {
		return
	}
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || (p.boat == 0 && !w.inWater(p.X, p.Y)) || math.Hypot(p.X-c.X, p.Y-c.Y) > MegaBreachReach {
			continue
		}
		w.seaBite(c, p, MegaBreachBoat, MegaBreachRider, MegaBreachSwim)
	}
	c.State, l.act = CreatureChase, MegaUpTicks-3*TicksPerSecond // up a while after its fall
}
