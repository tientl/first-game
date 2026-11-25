package sim

// The crafted tools and buildings: what a pickaxe, torch, spear and rod do,
// wooden walls that block the way, and beds to sleep the night through.

// Tool and building tuning.
const (
	SpearDamage     = 20  // damage per thrust with a held spear; see combat.go
	SpearReachBonus = 0.5 // extra reach on creatures with a held spear, in tiles

	TorchBurnTicks = 90 * TicksPerSecond // one torch burns this long while held
	TorchRadius    = 2.5                 // tiles of light and warmth around a lit torch

	WallHits   = 3 // axe hits to take a wall down
	WallRefund = 1 // wood back from a wall taken down

	SleepSpeedup      = 8 // world ticks per server tick while every player sleeps
	SleepRegenFactor  = 4 // health comes back this much faster in bed
	SleepHungerFactor = 0.35 // a night is twice as long as it was: a night in bed costs what it did
)

// TorchLit reports whether p holds a burning torch.
func (p *Player) TorchLit() bool {
	return !p.Dead && !p.Asleep && p.Held() == ItemTorch
}

// TorchFuel is how many ticks the held torch has left; 0 when none is lit.
func (p *Player) TorchFuel() int { return p.torchFuel }

// burnTorch burns the held torch for one tick. A torch only burns while
// held; one that is put away keeps what is left for next time.
func (w *World) burnTorch(p *Player) {
	if !p.TorchLit() {
		return
	}
	if p.torchFuel <= 0 {
		p.torchFuel = TorchBurnTicks // lighting a fresh torch
	}
	p.torchFuel--
	if p.torchFuel == 0 {
		p.Inv.TakeSlot(p.Selected, 1) // burnt out
	}
}

// warm reports whether (x, y) is safe from the cold: in a campfire's light,
// or near someone holding a lit torch.
func (w *World) warm(x, y float64) bool {
	if w.nearFire(x, y) || w.nearLava(x, y) {
		return true
	}
	for _, id := range w.playerOrder {
		q := w.players[id]
		if q.TorchLit() && (q.X-x)*(q.X-x)+(q.Y-y)*(q.Y-y) <= TorchRadius*TorchRadius {
			return true
		}
	}
	return false
}

// blocked reports whether a structure on tile (tx, ty) blocks the way.
func (w *World) blocked(tx, ty int) bool {
	if w.arenaBlocks(tx, ty) {
		return true // the roots round a boss's arena: see lair.go
	}
	e := w.entityAt(tx, ty)
	return e != nil && blocksPlayers(e.Kind)
}

// sleep puts p to bed in e. Beds are for the evening and the night.
func (w *World) sleep(p *Player, e *Entity) bool {
	if PhaseOf(w.tick) == PhaseDay {
		p.notify(NoticeNotTired, 0)
		return false
	}
	p.Asleep, p.bed = true, e.ID
	if p.home != e.ID {
		p.home = e.ID // this is where p comes back after dying now
		p.notify(NoticeHomeSet, 0)
	}
	if x, y := float64(e.TX)+0.5, float64(e.TY)+0.5; w.standable(x, y) {
		p.X, p.Y = x, y
	}
	p.Dir = DirDown
	return true
}

func (p *Player) wake() { p.Asleep, p.bed = false, 0 }

// stepAsleep is a tick of sleep: no cold, hunger slower (a night in bed
// costs about a quarter of the bar), health back
// much faster. Morning, a walk key, the bed going away or getting hurt
// (see hurt) wake the sleeper.
func (w *World) stepAsleep(p *Player) {
	if b := w.entities[p.bed]; b == nil || b.Kind != EntityBed ||
		(PhaseOf(w.tick) == PhaseDay && Ambient(w.tick) >= 1) ||
		p.input.MoveX != 0 || p.input.MoveY != 0 {
		p.wake()
		return
	}
	p.Cold = false
	p.Hunger = max(0, p.Hunger-HungerPerTick*SleepHungerFactor)
	if p.Hunger == 0 {
		hurt(p, StarveDamagePerTick)
		return
	}
	p.Health = min(MaxHealth, p.Health+RegenPerTick*SleepRegenFactor)
}

// AllAsleep reports whether there are players and every living one is in
// bed; the server then lets the night pass faster.
func (w *World) AllAsleep() bool {
	any := false
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead {
			continue
		}
		if !p.Asleep {
			return false
		}
		any = true
	}
	return any
}

// reachBonus is how much further p reaches creatures with what is held.
func (p *Player) reachBonus() float64 {
	return RideReach(p.mount != 0) + ReachBonusFor(p.Held()) // see tame.go
}

// ReachBonusFor is reachBonus for a held item kind, for the client's
// highlight, which must pick the same target as the server.
func ReachBonusFor(held ItemKind) float64 {
	switch held {
	case ItemSpear:
		return SpearReachBonus
	case ItemIvorySpear:
		return IvorySpearReach // see iceage.go
	}
	return legendaryReach(held) // see legendary.go
}
