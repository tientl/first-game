package sim

import "math"

// Dev mode: tools for testing the game quickly, reached only through a
// server started in dev mode (see server.Config.Dev and devcmd.go). Nothing
// here is saved but the free build setting (devbuild.go), and that only in
// a dev world's save: god mode, the speed factor and a forced weather last
// as long as the process, and a world that never had them steps exactly as
// before. Moving the clock (DevSetTime, DevSetSeason) is a real change to
// the world and is saved like any other.

// DevKillRadius is how far round the player `kill` reaches, in tiles.
const DevKillRadius = 12.0

// Bounds of the dev speed factor.
const (
	DevSpeedMin = 0.1
	DevSpeedMax = 8.0
)

// devPlayer is a player's dev-mode state.
type devPlayer struct {
	god      bool    // takes no harm, never hungry or cold
	speed    float64 // walking and swimming speed factor; 0 means 1
	build    bool    // free build: see devbuild.go
	buildSet bool    // build was chosen, here or in the dev save; until then it starts on
}

func (d devPlayer) speedFactor() float64 {
	if d.speed <= 0 {
		return 1
	}
	return d.speed
}

// devWorld is the world's dev-mode state.
type devWorld struct {
	weather Weather // forced weather, when forced is set
	forced  bool
	on      bool // the server runs in dev mode: see SetDevMode in devbuild.go
	calm    bool // the bosses sleep through anyone coming near (boss calm): see bossdev.go
}

// DevPlace is somewhere `tp` can go.
type DevPlace uint8

const (
	DevSpawn   DevPlace = iota // where players appear
	DevVillage                 // the nearest village's post
	DevCave                    // inside the caves, by the nearest cave mouth
	DevPyramid                 // the pyramid's doorway
	DevBoss                    // a boss's lair (with its number)
	DevCamp                    // the camp by a boss's lair (with its number), since StyleLairs
)

func (w *World) devPlayer(id PlayerID) *Player { return w.players[id] }

// DevSetGod turns god mode on or off for player id. It reports false for an
// unknown player.
func (w *World) DevSetGod(id PlayerID, on bool) bool {
	p := w.devPlayer(id)
	if p == nil {
		return false
	}
	p.dev.god = on
	if on {
		p.Cold = false
	}
	return true
}

// DevGod reports whether player id is in god mode.
func (w *World) DevGod(id PlayerID) bool {
	p := w.devPlayer(id)
	return p != nil && p.dev.god
}

// devNeeds keeps a god-mode player fed and warm; stepPlayer calls it after
// the hunger and the cold of the tick are worked out.
func (p *Player) devNeeds() {
	if p.dev.god {
		p.Hunger, p.Cold = MaxHunger, false
	}
}

// DevSetSpeed sets player id's speed factor, clamped to DevSpeedMin to
// DevSpeedMax, and returns what it was set to (0 for an unknown player).
func (w *World) DevSetSpeed(id PlayerID, f float64) float64 {
	p := w.devPlayer(id)
	if p == nil {
		return 0
	}
	if math.IsNaN(f) {
		f = 1
	}
	p.dev.speed = min(DevSpeedMax, max(DevSpeedMin, f))
	return p.dev.speed
}

// DevGive puts n of item k in player id's inventory and returns how many
// fitted.
func (w *World) DevGive(id PlayerID, k ItemKind, n int) int {
	p := w.devPlayer(id)
	if p == nil {
		return 0
	}
	return p.Inv.Add(k, n)
}

// DevHeal brings player id back to full health and a full belly, cured of
// venom, web and cold; a dead player gets up where they fell.
func (w *World) DevHeal(id PlayerID) bool {
	p := w.devPlayer(id)
	if p == nil {
		return false
	}
	p.Dead = false
	p.Health, p.Hunger, p.Cold = MaxHealth, MaxHunger, false
	p.poison, p.webbed = 0, 0
	return true
}

// DevReveal marks the whole map as seen by player id.
func (w *World) DevReveal(id PlayerID) bool {
	p := w.devPlayer(id)
	if p == nil {
		return false
	}
	all := NewExplored(w.width, w.height)
	for i := range all.bits {
		all.bits[i] = 0xff
	}
	p.explored = ExploredFrom(w.width, w.height, all.bits) // trims the padding
	return true
}

// DevFind returns where `tp` to place would land player id: the nearest
// village or cave, the pyramid, the spawn, or boss n (1 to NumBosses, in
// BossKind order). ok is false when the world has no such place.
func (w *World) DevFind(id PlayerID, place DevPlace, n int) (x, y float64, ok bool) {
	p := w.devPlayer(id)
	if p == nil {
		return 0, 0, false
	}
	switch place {
	case DevSpawn:
		return w.devLanding(w.spawnX, w.spawnY, 0)
	case DevVillage:
		best := math.Inf(1)
		for _, v := range w.villages {
			if d := math.Hypot(v.X-p.X, v.Y-p.Y); d < best {
				x, y, best = v.X, v.Y, d
			}
		}
		if math.IsInf(best, 1) {
			return 0, 0, false
		}
		return w.devLanding(x, y, 1.5)
	case DevCave:
		return w.devCave(p)
	case DevPyramid:
		tx, ty, ok := w.Pyramid()
		if !ok {
			return 0, 0, false
		}
		return w.devLanding(float64(tx)+0.5, float64(ty)+1.5, 0)
	case DevBoss:
		if !w.bosses.placed {
			w.placeBosses()
		}
		for _, b := range w.Bosses() {
			if b.Kind == BossKind(n-1) {
				switch s := &w.bosses.lairs[n-1].site; s.layout {
				case LayoutTemple: // at the courtyard's gate, the temple before them
					x, y := b.LairX, b.LairY+s.r+1.5
					if s.camp {
						a := math.Atan2(s.campY-b.LairY, s.campX-b.LairX)
						x, y = b.LairX+math.Cos(a)*(s.r+1.5), b.LairY+math.Sin(a)*(s.r+1.5)
					}
					return w.devLanding(x, y, 0)
				case LayoutAtoll: // on the reef, by the lagoon
					return w.devLanding(b.LairX, b.LairY+AtollLagoon+1.2, 0)
				}
				return w.devLanding(b.LairX, b.LairY, 7)
			}
		}
	case DevCamp:
		if !w.bosses.placed {
			w.placeBosses()
		}
		if l := &w.bosses.lairs[n-1]; l.ok && l.site.camp {
			return w.devLanding(l.site.campX, l.site.campY, 0)
		}
	}
	return 0, 0, false
}

// devCave finds the way into the caves nearest p: past the nearest cave
// mouth, or, in a world without caves below, the nearest cave floor.
func (w *World) devCave(p *Player) (x, y float64, ok bool) {
	best := math.Inf(1)
	for at, dest := range w.links {
		if w.Underground(at[1]) || !w.Underground(int(dest.y)) {
			continue // a way up, not a mouth
		}
		d := math.Hypot(float64(at[0])+0.5-p.X, float64(at[1])+0.5-p.Y)
		if d < best || (d == best && (dest.y < y || dest.y == y && dest.x < x)) { // ties: the same pick every run
			x, y, best = dest.x, dest.y, d
		}
	}
	if !math.IsInf(best, 1) {
		return x, y, true
	}
	for ty := 0; ty < w.height; ty++ {
		for tx := 0; tx < w.width; tx++ {
			if w.tiles[ty*w.width+tx] != TileCave {
				continue
			}
			if d := math.Hypot(float64(tx)+0.5-p.X, float64(ty)+0.5-p.Y); d < best {
				x, y, best = float64(tx)+0.5, float64(ty)+0.5, d
			}
		}
	}
	if math.IsInf(best, 1) {
		return 0, 0, false
	}
	return w.devLanding(x, y, 0)
}

// devLandReach is how far from the place asked for a landing is looked for.
const devLandReach = 40

// devLanding is the tile centre nearest (x, y), at least minD from it, where
// a player can stand (or, in a world of seas, swim).
func (w *World) devLanding(x, y, minD float64) (float64, float64, bool) {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	tests := []func(x, y float64) bool{w.standable}
	if w.style >= StyleSeas {
		tests = append(tests, w.swimmable) // anyone can swim there
	}
	for _, can := range tests {
		bx, by, best := 0.0, 0.0, math.Inf(1)
		for ty := max(0, cy-devLandReach); ty <= min(w.height-1, cy+devLandReach); ty++ {
			for tx := max(0, cx-devLandReach); tx <= min(w.width-1, cx+devLandReach); tx++ {
				px, py := float64(tx)+0.5, float64(ty)+0.5
				d := math.Hypot(px-x, py-y)
				if d < minD || d >= best || !can(px, py) {
					continue
				}
				bx, by, best = px, py, d
			}
		}
		if !math.IsInf(best, 1) {
			return bx, by, true
		}
	}
	return 0, 0, false
}

// DevTeleport puts player id at (x, y), off any boat or mount, and reports
// whether it did. The spot must be on the map.
func (w *World) DevTeleport(id PlayerID, x, y float64) bool {
	p := w.devPlayer(id)
	if p == nil || !finite(x, y) || x < 0 || y < 0 || x >= float64(w.width) || y >= float64(w.height) {
		return false
	}
	if p.mount != 0 {
		w.dismount(p)
	}
	if b := w.boatByID(p.boat); b != nil && w.boatFits(b.Kind, x, y) {
		b.X, b.Y, b.Speed = x, y, 0 // by sea to sea: the boat comes too
	} else if p.boat != 0 {
		w.leaveBoat(p)
	}
	p.wake()
	p.line = line{}
	p.X, p.Y = x, y
	p.travelCD = TravelHold // landing on a cave mouth does not take p on at once
	p.explored.Reveal(p.X, p.Y)
	return true
}

// DevSpawn puts up to n creatures of kind k round player id, a few tiles
// away, where they fit, and returns how many it placed. Bosses only come
// to their lairs.
func (w *World) DevSpawn(id PlayerID, k CreatureKind, n int) int {
	p := w.devPlayer(id)
	if p == nil || k == CreatureNone || k >= NumCreatureKinds || k.Boss() || k.Name() == "" {
		return 0
	}
	var placed [][2]float64
	for d := 3.0; d <= 14 && len(placed) < n; d++ {
		steps := int(d * 4)
		for i := 0; i < steps && len(placed) < n; i++ {
			a := 2 * math.Pi * float64(i) / float64(steps)
			x, y := p.X+d*math.Cos(a), p.Y+d*math.Sin(a)
			if !w.creatureFits(k, x, y) || w.Underground(int(y)) != w.Underground(int(p.Y)) {
				continue
			}
			near := false
			for _, q := range placed {
				if math.Hypot(q[0]-x, q[1]-y) < 1.5 {
					near = true
					break
				}
			}
			if near {
				continue
			}
			c := w.addCreature(k, x, y)
			c.FaceLeft = x > p.X // looking at the player
			placed = append(placed, [2]float64{x, y})
		}
	}
	return len(placed)
}

// DevKill fells every wild creature within r of player id (pets and
// mounts are spared) and returns how many.
func (w *World) DevKill(id PlayerID, r float64) int {
	p := w.devPlayer(id)
	if p == nil {
		return 0
	}
	n := 0
	for _, c := range w.fauna.creatures {
		if c.gone || c.ownerName != "" || c.rider != 0 || math.Hypot(c.X-p.X, c.Y-p.Y) > r {
			continue
		}
		if w.wound(c, max(c.HP, 1), p) {
			n++
		}
	}
	return n
}

// DevSetTime moves the clock forward to the next moment the day is frac
// through (0 dawn, DuskStart dusk, NightStart night). The clock never goes
// back, so nothing waiting on a tick is lost.
func (w *World) DevSetTime(frac float64) {
	if math.IsNaN(frac) {
		return
	}
	frac = min(math.Nextafter(1, 0), max(0, frac))
	day := w.tick / DayTicks
	t := day*DayTicks + uint64(frac*DayTicks)
	if t <= w.tick {
		t += DayTicks
	}
	w.tick = t
}

// DevSetSeason moves the clock forward to the first morning of season s,
// unless it is already s. It reports whether the clock moved.
func (w *World) DevSetSeason(s Season) bool {
	if s >= NumSeasons || w.Season() == s {
		return false
	}
	day := w.tick / DayTicks
	for SeasonOf(day*DayTicks) != s {
		day++
	}
	w.tick = day*DayTicks + StartTick // just after dawn, like a new world
	return true
}

// DevForceWeather holds the sky at wt until DevClearWeather.
func (w *World) DevForceWeather(wt Weather) {
	if wt < NumWeathers {
		w.dev.weather, w.dev.forced = wt, true
	}
}

// DevClearWeather lets the weather follow the seed again.
func (w *World) DevClearWeather() { w.dev.weather, w.dev.forced = 0, false }

// DevWeatherForced reports whether the weather is held by DevForceWeather.
func (w *World) DevWeatherForced() bool { return w.dev.forced }
