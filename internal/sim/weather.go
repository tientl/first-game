package sim

import (
	"math"
	"slices"
)

// Seasons and weather.
//
// A year has four seasons of SeasonDays days each: spring, summer, autumn
// and winter. The weather changes every few minutes, and what comes depends
// on the season: spring rains, summer storms with thunder and lightning and
// now and then a tornado, autumn winds, winter snow.
//
//   - Rain and storms make campfires and standing torches burn twice as fast.
//   - In a storm lightning strikes near players: anyone close is badly
//     hurt, beasts too, and a tree struck is felled.
//   - A tornado wanders the land, whirls anyone it catches about and hurts
//     them, and tears trees down.
//   - In winter it is cold out in the snow and the wind even by day, unless
//     wearing furs or hide, or by a fire or a torch; the lakes and ponds (not
//     the sea) freeze, and can be walked on.
//
// The weather is a pure function of the world's seed and the tick, so it
// needs no saving, and every player sees the same.

// Season is a part of the year.
type Season uint8

const (
	Spring Season = iota
	Summer
	Autumn
	Winter
	NumSeasons
)

// Weather is the sky over the whole world.
type Weather uint8

const (
	WeatherClear Weather = iota
	WeatherRain
	WeatherWind
	WeatherStorm   // rain, thunder and lightning
	WeatherTornado // wind, and a tornado about
	WeatherSnow
	NumWeathers
)

// Weather tuning.
const (
	SeasonDays    = 3
	WeatherTicks  = 100 * TicksPerSecond // the weather holds this long
	WetFireFactor = 2                    // fires burn this much faster in rain

	LightningMin    = 3 * TicksPerSecond // between strikes in a storm, least...
	LightningMax    = 9 * TicksPerSecond // ...and most
	LightningRange  = 18.0               // strikes fall this close to some player
	LightningRadius = 1.6                // and hurt whoever stands this close
	LightningDamage = 35.0
	LightningBeast  = 6
	LightningShow   = TicksPerSecond // how long a strike is sent to clients

	TornadoSpeed   = 1.6 / TicksPerSecond
	TornadoPull    = 3.0 // it drags players this close...
	TornadoHit     = 1.3 // ...and hurts those this close
	TornadoDamage  = 8.0 / TicksPerSecond
	TornadoSpin    = 2.6 / TicksPerSecond // how fast it whirls a player round it
	TornadoAway    = 14.0                 // it appears this far from a player...
	TornadoFell    = 0.04                 // per tick: a tree under it is torn down
	pondMaxTiles   = 900                  // water bodies up to this size freeze; bigger is sea
)

// SeasonOf returns the season at tick.
func SeasonOf(tick uint64) Season {
	return Season((DayOf(tick) - 1) / SeasonDays % int(NumSeasons))
}

// SeasonDay is the 1-based day within the season at tick.
func SeasonDay(tick uint64) int { return (DayOf(tick)-1)%SeasonDays + 1 }

// Name of the season.
func (s Season) Name() string {
	return [...]string{"Mùa xuân", "Mùa hạ", "Mùa thu", "Mùa đông", "?"}[min(s, NumSeasons)]
}

// Name of the weather.
func (wt Weather) Name() string {
	return [...]string{"Trời quang", "Mưa", "Gió", "Giông bão", "Lốc xoáy", "Tuyết rơi", "?"}[min(wt, NumWeathers)]
}

// Wet reports whether rain falls.
func (wt Weather) Wet() bool { return wt == WeatherRain || wt == WeatherStorm }

// Windy reports whether the wind blows hard.
func (wt Weather) Windy() bool { return wt == WeatherWind || wt == WeatherTornado || wt == WeatherStorm }

// weatherOdds is, per season, the chance of each weather.
var weatherOdds = [NumSeasons][NumWeathers]float64{
	Spring: {WeatherClear: 0.45, WeatherRain: 0.32, WeatherWind: 0.1, WeatherStorm: 0.13},
	Summer: {WeatherClear: 0.55, WeatherRain: 0.1, WeatherWind: 0.1, WeatherStorm: 0.19, WeatherTornado: 0.06},
	Autumn: {WeatherClear: 0.35, WeatherRain: 0.25, WeatherWind: 0.26, WeatherStorm: 0.09, WeatherTornado: 0.05},
	Winter: {WeatherClear: 0.35, WeatherWind: 0.2, WeatherSnow: 0.45},
}

// WeatherAt is the weather at tick in a world made from seed. The first
// spell of a new world is always fair.
func WeatherAt(seed uint64, tick uint64) Weather {
	slot := tick / WeatherTicks
	if slot <= StartTick/WeatherTicks+1 {
		return WeatherClear
	}
	r := unit(hash2(int32(slot), int32(slot>>31), uint32(seed)^uint32(seed>>32)^0x3ea7))
	odds := weatherOdds[SeasonOf(slot*WeatherTicks)]
	for k, p := range odds {
		if r < p {
			return Weather(k)
		}
		r -= p
	}
	return WeatherClear
}

// Weather returns the weather now.
func (w *World) Weather() Weather { return WeatherAt(w.seed, w.tick) }

// Season returns the season now.
func (w *World) Season() Season { return SeasonOf(w.tick) }

// Strike is a bolt of lightning.
type Strike struct {
	X, Y float64
	Tick uint64
}

// Tornado is a whirlwind on the move.
type Tornado struct {
	ID   uint32
	X, Y float64
	hx   float64
	hy   float64
	ends uint64
}

// weather is the world's live weather: strikes and tornadoes.
type weather struct {
	strikes   []Strike
	nextBolt  uint64
	tornadoes []*Tornado
	nextTwist uint32
	twistSlot uint64 // the weather spell the last tornado was raised in

	pond     []bool // per tile: a lake or pond that freezes in winter
	pondDone bool
}

// Strikes returns the lightning strikes of the last second.
func (w *World) Strikes() []Strike { return slices.Clone(w.wx.strikes) }

// Tornadoes returns copies of the tornadoes about.
func (w *World) Tornadoes() []Tornado {
	out := make([]Tornado, 0, len(w.wx.tornadoes))
	for _, t := range w.wx.tornadoes {
		out = append(out, *t)
	}
	return out
}

// PondMask marks the tiles of every lake and pond: bodies of water no
// bigger than pondMaxTiles. The client calls it on the same tiles, so both
// agree on what freezes.
func PondMask(width, height int, tiles []Tile) []bool {
	mask := make([]bool, len(tiles))
	seen := make([]bool, len(tiles))
	var comp []int
	for start, t := range tiles {
		if t != TileWater || seen[start] {
			continue
		}
		comp = append(comp[:0], start)
		seen[start] = true
		edge := false
		for k := 0; k < len(comp); k++ {
			x, y := comp[k]%width, comp[k]/width
			if x == 0 || y == 0 || x == width-1 || y == height-1 {
				edge = true
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= width || ny >= height {
					continue
				}
				if j := ny*width + nx; !seen[j] && tiles[j] == TileWater {
					seen[j] = true
					comp = append(comp, j)
				}
			}
		}
		if !edge && len(comp) <= pondMaxTiles {
			for _, i := range comp {
				mask[i] = true
			}
		}
	}
	return mask
}

// Frozen reports whether ponds are frozen in season s.
func Frozen(s Season) bool { return s == Winter }

// frozenAt reports whether tile (tx, ty) is pond ice now.
func (w *World) frozenAt(tx, ty int) bool {
	if !Frozen(w.Season()) || !w.inBounds(tx, ty) {
		return false
	}
	if !w.wx.pondDone {
		w.wx.pond = PondMask(w.width, w.height, w.tiles)
		w.wx.pondDone = true
	}
	return w.wx.pond[ty*w.width+tx]
}

// landAt reports whether a walker can stand on tile (tx, ty): walkable
// ground, or pond ice in winter.
func (w *World) landAt(tx, ty int) bool {
	return w.TileAt(tx, ty).Walkable() || w.frozenAt(tx, ty)
}

// stepWeather runs the live weather: lightning and tornadoes.
func (w *World) stepWeather() {
	wt := w.Weather()
	w.wx.strikes = slices.DeleteFunc(w.wx.strikes, func(s Strike) bool { return w.tick-s.Tick > LightningShow })
	if wt == WeatherStorm {
		if w.wx.nextBolt == 0 || w.wx.nextBolt > w.tick+LightningMax {
			w.wx.nextBolt = w.tick + LightningMin
		}
		if w.tick >= w.wx.nextBolt {
			w.lightning()
			w.wx.nextBolt = w.tick + LightningMin + uint64(w.random().IntN(LightningMax-LightningMin))
		}
	}
	slot := w.tick / WeatherTicks
	if wt == WeatherTornado && w.wx.twistSlot != slot && len(w.playerOrder) > 0 {
		if w.raiseTornado() {
			w.wx.twistSlot = slot
		}
	}
	for _, t := range w.wx.tornadoes {
		w.stepTornado(t)
	}
	w.wx.tornadoes = slices.DeleteFunc(w.wx.tornadoes, func(t *Tornado) bool { return w.tick >= t.ends })
}

// outdoors reports whether (x, y) is under the sky: not in a cave.
func (w *World) outdoors(x, y float64) bool {
	return w.TileAt(int(math.Floor(x)), int(math.Floor(y))) != TileCave
}

// lightning strikes once, somewhere near a player who is out of doors.
func (w *World) lightning() {
	var out []*Player
	for _, id := range w.playerOrder {
		if p := w.players[id]; !p.Dead && w.outdoors(p.X, p.Y) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return
	}
	r := w.random()
	p := out[r.IntN(len(out))]
	var x, y float64
	for range 12 {
		// Mostly a little way off; now and then right by them.
		d := 3 + r.Float64()*LightningRange
		if r.Float64() < 0.12 {
			d = r.Float64() * 2.5
		}
		a := r.Float64() * 2 * math.Pi
		x, y = p.X+math.Cos(a)*d, p.Y+math.Sin(a)*d
		// Tall things draw it: a tree near the spot takes the bolt.
		if e := w.tallNear(x, y); e != nil {
			x, y = float64(e.TX)+0.5, float64(e.TY)+0.5
		}
		tx, ty := int(math.Floor(x)), int(math.Floor(y))
		if w.inBounds(tx, ty) && w.TileAt(tx, ty) != TileCave && w.TileAt(tx, ty) != TileMountain {
			break
		}
	}
	w.wx.strikes = append(w.wx.strikes, Strike{X: x, Y: y, Tick: w.tick})
	for _, id := range w.playerOrder {
		q := w.players[id]
		if !q.Dead && math.Hypot(q.X-x, q.Y-y) <= LightningRadius {
			hurt(q, LightningDamage)
			q.notify(NoticeLightning, 0)
		}
	}
	for _, c := range w.fauna.creatures {
		if !c.gone && !c.State.OutOfReach() && math.Hypot(c.X-x, c.Y-y) <= LightningRadius+c.Kind.def().radius {
			if c.HP -= LightningBeast; c.HP <= 0 {
				w.fell(c)
			}
		}
	}
	if e := w.entityAt(int(math.Floor(x)), int(math.Floor(y))); e != nil && e.Kind == EntityTree && e.Ready {
		w.fellTree(e)
	}
}

// tallNear returns a standing tree within a tile of (x, y), if any.
func (w *World) tallNear(x, y float64) *Entity {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for ty := cy - 1; ty <= cy+1; ty++ {
		for tx := cx - 1; tx <= cx+1; tx++ {
			if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityTree && e.Ready {
				return e
			}
		}
	}
	return nil
}

// fellTree brings a standing tree down by itself (lightning, a tornado):
// a stump is left, and its wood falls on the ground.
func (w *World) fellTree(e *Entity) {
	x, y := float64(e.TX)+0.5, float64(e.TY)+0.5
	w.dropAt(x, y, []Ingredient{{Item: ItemWood, Count: bySize(e.Big, TreeYield, BigTreeYield) / 2}, {Item: ItemStick, Count: 1}})
	e.Ready = false
	e.HP = maxHP(e.Kind, e.Big)
	e.RegrowAt = w.tick + TreeRegrowTicks
	w.watch(e)
}

// raiseTornado sets a tornado going somewhere out in the open near a player.
func (w *World) raiseTornado() bool {
	r := w.random()
	p := w.players[w.playerOrder[r.IntN(len(w.playerOrder))]]
	if p.Dead || !w.outdoors(p.X, p.Y) {
		return false
	}
	for range 16 {
		a := r.Float64() * 2 * math.Pi
		x, y := p.X+math.Cos(a)*TornadoAway, p.Y+math.Sin(a)*TornadoAway
		if !w.walkableAt(x, y) || !w.outdoors(x, y) {
			continue
		}
		w.wx.nextTwist++
		t := &Tornado{ID: w.wx.nextTwist, X: x, Y: y, ends: (w.tick/WeatherTicks + 1) * WeatherTicks}
		// It sets off roughly towards the player, wandering as it goes.
		d := math.Hypot(p.X-x, p.Y-y)
		t.hx, t.hy = (p.X-x)/d, (p.Y-y)/d
		w.wx.tornadoes = append(w.wx.tornadoes, t)
		return true
	}
	return false
}

func (w *World) stepTornado(t *Tornado) {
	r := w.random()
	// Wander: the heading turns a little each tick.
	a := math.Atan2(t.hy, t.hx) + (r.Float64()-0.5)*0.25
	t.hx, t.hy = math.Cos(a), math.Sin(a)
	nx, ny := t.X+t.hx*TornadoSpeed, t.Y+t.hy*TornadoSpeed
	tx, ty := int(math.Floor(nx)), int(math.Floor(ny))
	if !w.inBounds(tx, ty) || w.TileAt(tx, ty) == TileMountain || w.TileAt(tx, ty) == TileCave {
		t.hx, t.hy = -t.hx, -t.hy // it turns back off the rock
	} else {
		t.X, t.Y = nx, ny
	}
	for _, id := range w.playerOrder {
		p := w.players[id]
		if p.Dead || p.Asleep {
			continue
		}
		dx, dy := p.X-t.X, p.Y-t.Y
		d := math.Hypot(dx, dy)
		if d > TornadoPull || !w.outdoors(p.X, p.Y) {
			continue
		}
		// Whirled round it and drawn in.
		if d > 1e-6 {
			mx := -dy/d*TornadoSpin - dx/d*TornadoSpin*0.4
			my := dx/d*TornadoSpin - dy/d*TornadoSpin*0.4
			if w.standable(p.X+mx, p.Y) {
				p.X += mx
			}
			if w.standable(p.X, p.Y+my) {
				p.Y += my
			}
		}
		if d <= TornadoHit {
			hurt(p, TornadoDamage)
			if w.tick%TicksPerSecond == 0 {
				p.notify(NoticeTornado, 0)
			}
		}
	}
	if e := w.entityAt(tx, ty); e != nil && e.Kind == EntityTree && e.Ready && r.Float64() < TornadoFell {
		w.fellTree(e)
	}
}

// wetFire reports whether rain is falling on (x, y), so fires there burn faster.
func (w *World) wetFire(tx, ty int) bool {
	return w.Weather().Wet() && w.TileAt(tx, ty) != TileCave
}

// wintry reports whether p feels the winter: out in the snow or the wind in
// winter, in nothing warm.
func (w *World) wintry(p *Player) bool {
	if w.Season() != Winter || !w.outdoors(p.X, p.Y) {
		return false
	}
	if wt := w.Weather(); wt != WeatherSnow && wt != WeatherWind {
		return false
	}
	return !warmlyDressed(&p.Inv)
}

// warmlyDressed reports whether inv holds clothes that keep the winter out.
func warmlyDressed(inv *Inventory) bool {
	return inv.Count(ItemFurCoat) > 0 || inv.Count(ItemDinoArmor) > 0 || inv.Count(ItemCopperArmor) > 0
}

// underIce reports whether c swims under pond ice now: out of sight and reach.
func (w *World) underIce(c *Creature) bool {
	d := c.Kind.def()
	if !d.water && !d.deep && !d.amphibious {
		return false
	}
	return w.frozenAt(int(math.Floor(c.X)), int(math.Floor(c.Y)))
}
