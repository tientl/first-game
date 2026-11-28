package sim

import "math"

// Troodon: the night hunters. When the dark comes down, packs of these
// small, big-eyed dinosaurs come out of the woods after anyone out in the
// open: they slink up, dart in to bite, and fall back. They will not come
// into the light of a fire, a lit furnace, lava or a burning torch: they
// circle at its edge, watching. At first light they slip away and are gone.
// They never come for someone asleep in bed, swimming or out in a boat.

// Night hunter tuning.
const (
	TroodonPack       = 3
	TroodonPacksNear  = 2                   // packs about one player at most
	TroodonSpawnEvery = 15 * TicksPerSecond // how often the dark is searched for a player to hunt
	TroodonNight      = 0.15                // ambient light under which they hunt
	TroodonSight      = 11.0                // tiles: their eyes are made for the dark
	TroodonGiveUp     = 22.0                // and they let prey go this far off
	TroodonGone       = 30.0                // a hunter this far from everyone melts away
	TroodonBite       = 7.0
	TroodonBiteRange  = 1.2
	TroodonBiteCD     = 3 * TicksPerSecond / 2
	TroodonBackOff    = TicksPerSecond     // after a bite it falls back this long
	troodonSpawnNear  = 16.0               // tiles from the player a pack comes out of the dark,
	troodonSpawnFar   = 24.0               // at least and at most
	troodonDawn       = TroodonNight + 0.1 // light at which they slip away home
)

var troodonDef = creatureDef{
	name: "Troodon", hp: 5,
	walk: 1.8 / TicksPerSecond, run: 3.8 / TicksPerSecond, // a little slower than a player: running to a fire works
	radius: 0.3, reach: 1.7,
	yield: ItemMeat, yieldN: 2, drops: []Ingredient{{ItemHide, 1}, {ItemBone, 1}},
	leash: 40, idle: [2]int{20, 60}, roam: [2]int{20, 60},
	fights: true, openOnly: true,
}

func init() {
	creatureDefs[CreatureTroodon] = troodonDef
	nameHead(CreatureTroodon)
}

// inLight reports whether (x, y) is lit against the dark: by a fire, a lit
// furnace, lava, or someone's burning torch.
func (w *World) inLight(x, y float64) bool { return w.warm(x, y) }

// huntable reports whether the night hunters go after p.
func (w *World) huntable(p *Player) bool {
	return !p.Dead && !p.Asleep && !w.Swimming(p) && p.BoatID() == 0 && !w.Underground(int(math.Floor(p.Y))) &&
		w.outdoors(p.X, p.Y)
}

// stepNightHunters lets packs of Troodon come out of the dark near players
// who are out at night.
func (w *World) stepNightHunters() {
	if w.style == StyleIsland || w.tick%TroodonSpawnEvery != 0 || Ambient(w.tick) >= TroodonNight {
		return
	}
	r := w.random()
	for _, id := range w.playerOrder {
		p := w.players[id]
		if !w.huntable(p) {
			continue
		}
		near := 0
		for _, c := range w.fauna.creatures {
			if c.Kind == CreatureTroodon && !c.gone && math.Hypot(c.X-p.X, c.Y-p.Y) < TroodonGone {
				near++
			}
		}
		if near >= TroodonPack*w.difficulty.tune().packs {
			continue
		}
		for range 12 {
			a := r.Float64() * 2 * math.Pi
			d := troodonSpawnNear + r.Float64()*(troodonSpawnFar-troodonSpawnNear)
			x, y := p.X+math.Cos(a)*d, p.Y+math.Sin(a)*d
			if w.Underground(int(math.Floor(y))) || !w.outdoors(x, y) || w.inLight(x, y) || !w.creatureFits(CreatureTroodon, x, y) {
				continue
			}
			if _, dist := w.nearestPlayer(x, y); dist < troodonSpawnNear-2 {
				continue // out of sight of everyone
			}
			w.addHerd(CreatureTroodon, x, y, TroodonPack)
			break
		}
	}
}

// troodonPrey is the nearest player the night hunter c can go after.
func (w *World) troodonPrey(c *Creature) (*Player, float64) {
	var best *Player
	bestD := math.Inf(1)
	for _, id := range w.playerOrder {
		p := w.players[id]
		if !w.huntable(p) {
			continue
		}
		if d := math.Hypot(p.X-c.X, p.Y-c.Y); d < bestD {
			best, bestD = p, d
		}
	}
	return best, bestD
}

func (w *World) stepTroodon(c *Creature) {
	d := c.Kind.def()
	if c.biteCD > 0 {
		c.biteCD--
	}
	p, dist := w.troodonPrey(c)
	if _, anyone := w.nearestPlayer(c.X, c.Y); anyone > TroodonGone {
		c.gone = true // melted back into the dark
		return
	}
	if Ambient(w.tick) >= troodonDawn {
		// First light: away from people, and gone once out of sight.
		if _, anyone := w.nearestPlayer(c.X, c.Y); anyone > troodonSpawnNear {
			c.gone = true
			return
		}
		c.State = CreatureFlee
		if q, _ := w.nearestPlayer(c.X, c.Y); q != nil {
			c.steerAway(q.X, q.Y, 0.4)
		}
		w.moveCreature(c, c.hx*d.run, c.hy*d.run)
		return
	}
	if w.inLight(c.X, c.Y) {
		// Caught in the light: out of it.
		c.State = CreatureFlee
		if p != nil {
			c.steerAway(p.X, p.Y, 0.8)
		}
		if !w.moveCreature(c, c.hx*d.run, c.hy*d.run) && p != nil {
			c.dodge(w, p.X, p.Y)
		}
		return
	}
	switch c.State {
	case CreatureFlee: // falling back after a bite
		if c.timer--; c.timer <= 0 {
			c.State, c.timer = CreatureChase, 0
		} else if p != nil {
			c.steerAway(p.X, p.Y, 0.5)
			w.troodonMove(c, c.hx*d.run, c.hy*d.run)
		}
		return
	}
	if p == nil || dist > TroodonGiveUp || (c.State != CreatureChase && dist > TroodonSight) {
		if c.State == CreatureChase {
			c.State, c.timer = CreatureIdle, 0
		}
		w.wander(c)
		return
	}
	c.State = CreatureChase
	c.faceToward(p.X)
	if w.packFight(c, p, dist, func(dx, dy float64) { w.troodonMove(c, dx, dy) }) {
		return // a cry, a lunge out of the dark, a venomous bite, or circling its turn: see skillai.go
	}
	// Close in, each from its own side so the pack surrounds its prey, but
	// never into the light: at its edge they wait, watching.
	a := float64(c.ID%7) * 0.9
	tx, ty := p.X+math.Cos(a)*0.9, p.Y+math.Sin(a)*0.9
	dx, dy := tx-c.X, ty-c.Y
	if l := math.Hypot(dx, dy); l > 1e-6 {
		step := min(d.run, l)
		w.troodonMove(c, dx/l*step, dy/l*step)
	}
}

// troodonMove moves c by (dx, dy) unless that would take it into the light.
func (w *World) troodonMove(c *Creature, dx, dy float64) {
	if w.inLight(c.X+dx, c.Y+dy) {
		return
	}
	w.moveCreature(c, dx, dy)
}

// winterNight reports whether it is a winter night, which freezes anyone
// out of reach of a fire. Nights in the other seasons are only dark.
func (w *World) winterNight() bool {
	return Ambient(w.tick) < ColdAmbient && w.Season() == Winter
}
